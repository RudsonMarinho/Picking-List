//go:build integration

// Cobertura complementar ao business_flow_test.go: exercita os caminhos de
// UPDATE/DELETE/LIST de cada módulo e a criação de usuários/roles via API —
// condição do Mentor para manter o desvio SQLC-01 (queries pgx à mão em vez
// de sqlc): "todo caminho de query coberto por teste de integração contra
// Postgres real".
package app

import (
	"context"
	"strings"
	"testing"

	"github.com/Fortcargo/invtech/internal/repository"
	"github.com/Fortcargo/invtech/internal/services"
	"github.com/Fortcargo/invtech/pkg/config"
	"github.com/Fortcargo/invtech/pkg/logger"
	"github.com/Fortcargo/invtech/pkg/security"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func TestModuleCRUD_UpdateListDelete(t *testing.T) {
	ctx := context.Background()

	ownerPool, err := repository.NewPool(ctx, testDBParams(t, "TEST_DB_OWNER_USER", "TEST_DB_OWNER_PASSWORD"))
	if err != nil {
		t.Fatalf("pool owner: %v", err)
	}
	defer ownerPool.Close()
	appPool, err := repository.NewPool(ctx, testDBParams(t, "TEST_DB_APP_USER", "TEST_DB_APP_PASSWORD"))
	if err != nil {
		t.Fatalf("pool app: %v", err)
	}
	defer appPool.Close()

	email := "crud-" + uuid.NewString() + "@example.com"
	const password = "senha-inicial-123"

	var tenantID uuid.UUID
	if err := ownerPool.QueryRow(ctx,
		`INSERT INTO tenants (nome, slug) VALUES ('tenant crud', $1) RETURNING id`,
		"tenant-crud-"+uuid.NewString(),
	).Scan(&tenantID); err != nil {
		t.Fatalf("criar tenant: %v", err)
	}

	hash, err := security.HashPassword(password)
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	// must_change_password=false — este teste foca em CRUD, não no gate de senha.
	err = repository.WithTenant(ctx, appPool, tenantID, func(tx pgx.Tx) error {
		var roleID uuid.UUID
		if err := tx.QueryRow(ctx,
			`INSERT INTO roles (tenant_id, name) VALUES ($1, 'admin') RETURNING id`, tenantID,
		).Scan(&roleID); err != nil {
			return err
		}
		// role admin de teste precisa de role_permissions — sem isso, todo
		// endpoint de negócio devolve 403 desde o enforcement de RBAC.
		if err := repository.AddRolePermissions(ctx, tx, tenantID, roleID, repository.FullPermissionSet()); err != nil {
			return err
		}
		userID, err := repository.CreateUserWithLookup(ctx, tx, tenantID, roleID, email, &hash, "local", false)
		if err != nil {
			return err
		}
		// scope 'all' — sem nenhuma linha em user_scopes o usuário não passa
		// em NENHUMA checagem de escopo (middleware.Scope trata "sem linhas"
		// como "sem acesso a nada", não como "all"); a API real sempre grava
		// pelo menos essa linha default (ver CreateUser em user.go).
		return repository.CreateUserScopes(ctx, tx, tenantID, userID, []repository.UserScope{{ScopeType: "all"}})
	})
	if err != nil {
		t.Fatalf("criar usuário: %v", err)
	}

	cfg := &config.Config{
		AppEnv:            "development",
		MailDriver:        "noop",
		SessionCookieName: "invtech_session",
		CSRFCookieName:    "csrf_token",
	}
	authSvc := &services.AuthService{Pool: appPool, SessionTTL: 24 * 3600 * 1e9}
	fapp := New(cfg, appPool, authSvc, logger.New(cfg.AppEnv))
	tc := &testClient{t: t, fapp: fapp, sessionNm: cfg.SessionCookieName}

	tc.do(fiber.MethodGet, "/api/v1/auth/login", "")
	loginBody := `{"email":"` + email + `","password":"` + password + `"}`
	if status, resp := tc.do(fiber.MethodPost, "/api/v1/auth/login", loginBody); status != fiber.StatusOK {
		t.Fatalf("login: status %d, resp %v", status, resp)
	}

	// ---- company: create → update → list → (descartável) delete ----
	status, resp := tc.do(fiber.MethodPost, "/api/v1/companies", `{"nome":"Empresa A","cnpj":"11222333000181"}`)
	requireStatus(t, "create company", fiber.StatusCreated, status, resp)
	companyID := resp["id"].(string)

	status, resp = tc.do(fiber.MethodPut, "/api/v1/companies/"+companyID, `{"nome":"Empresa A Editada","cnpj":"11222333000181"}`)
	requireStatus(t, "update company", fiber.StatusOK, status, resp)

	status, resp = tc.do(fiber.MethodGet, "/api/v1/companies", "")
	requireStatus(t, "list companies", fiber.StatusOK, status, resp)
	requireSnakeCaseArray(t, tc, "/api/v1/companies", "nome", "cnpj")

	status, resp = tc.do(fiber.MethodPost, "/api/v1/companies", `{"nome":"Empresa Descartável","cnpj":"99888777000111"}`)
	requireStatus(t, "create company2", fiber.StatusCreated, status, resp)
	company2ID := resp["id"].(string)
	status, _ = tc.do(fiber.MethodDelete, "/api/v1/companies/"+company2ID, "")
	requireStatus(t, "delete company2", fiber.StatusOK, status, nil)

	// ---- branch: create → update → list → descartável delete ----
	status, resp = tc.do(fiber.MethodPost, "/api/v1/companies/"+companyID+"/branches", `{"nome":"Filial A","cnpj":"22333444000162"}`)
	requireStatus(t, "create branch", fiber.StatusCreated, status, resp)
	branchID := resp["id"].(string)

	status, resp = tc.do(fiber.MethodPut, "/api/v1/branches/"+branchID, `{"nome":"Filial A Editada","cnpj":"22333444000162"}`)
	requireStatus(t, "update branch", fiber.StatusOK, status, resp)
	if resp["company_id"] != companyID {
		t.Fatalf("update branch: company_id deveria ser %q, veio %v (resp completo: %v)", companyID, resp["company_id"], resp)
	}

	status, resp = tc.do(fiber.MethodGet, "/api/v1/companies/"+companyID+"/branches", "")
	requireStatus(t, "list branches", fiber.StatusOK, status, resp)
	requireSnakeCaseArray(t, tc, "/api/v1/companies/"+companyID+"/branches", "nome", "company_id")

	status, resp = tc.do(fiber.MethodPost, "/api/v1/companies/"+companyID+"/branches", `{"nome":"Filial Descartável","cnpj":"33444555000143"}`)
	requireStatus(t, "create branch2", fiber.StatusCreated, status, resp)
	branch2ID := resp["id"].(string)
	status, _ = tc.do(fiber.MethodDelete, "/api/v1/branches/"+branch2ID, "")
	requireStatus(t, "delete branch2", fiber.StatusOK, status, nil)

	// ---- sector: create → update → list → descartável delete ----
	status, resp = tc.do(fiber.MethodPost, "/api/v1/branches/"+branchID+"/sectors", `{"nome":"TI"}`)
	requireStatus(t, "create sector", fiber.StatusCreated, status, resp)
	sectorID := resp["id"].(string)

	status, resp = tc.do(fiber.MethodPut, "/api/v1/sectors/"+sectorID, `{"nome":"TI Editado"}`)
	requireStatus(t, "update sector", fiber.StatusOK, status, resp)
	if resp["branch_id"] != branchID {
		t.Fatalf("update sector: branch_id deveria ser %q, veio %v (resp completo: %v)", branchID, resp["branch_id"], resp)
	}

	status, resp = tc.do(fiber.MethodGet, "/api/v1/branches/"+branchID+"/sectors", "")
	requireStatus(t, "list sectors", fiber.StatusOK, status, resp)
	requireSnakeCaseArray(t, tc, "/api/v1/branches/"+branchID+"/sectors", "nome", "branch_id")

	status, resp = tc.do(fiber.MethodPost, "/api/v1/branches/"+branchID+"/sectors", `{"nome":"Setor Descartável"}`)
	requireStatus(t, "create sector2", fiber.StatusCreated, status, resp)
	sector2ID := resp["id"].(string)
	status, _ = tc.do(fiber.MethodDelete, "/api/v1/sectors/"+sector2ID, "")
	requireStatus(t, "delete sector2", fiber.StatusOK, status, nil)

	// ---- asset_type: create → update → list → descartável delete ----
	status, resp = tc.do(fiber.MethodPost, "/api/v1/asset-types", `{"nome":"Notebook"}`)
	requireStatus(t, "create asset_type", fiber.StatusCreated, status, resp)
	assetTypeID := resp["id"].(string)

	status, resp = tc.do(fiber.MethodPut, "/api/v1/asset-types/"+assetTypeID, `{"nome":"Notebook Editado"}`)
	requireStatus(t, "update asset_type", fiber.StatusOK, status, resp)

	status, resp = tc.do(fiber.MethodGet, "/api/v1/asset-types", "")
	requireStatus(t, "list asset_types", fiber.StatusOK, status, resp)
	requireSnakeCaseArray(t, tc, "/api/v1/asset-types", "nome")

	status, resp = tc.do(fiber.MethodPost, "/api/v1/asset-types", `{"nome":"Tipo Descartável"}`)
	requireStatus(t, "create asset_type2", fiber.StatusCreated, status, resp)
	assetType2ID := resp["id"].(string)
	status, _ = tc.do(fiber.MethodDelete, "/api/v1/asset-types/"+assetType2ID, "")
	requireStatus(t, "delete asset_type2", fiber.StatusOK, status, nil)

	// ---- asset: create → update → list → descartável delete ----
	assetBody := `{"asset_type_id":"` + assetTypeID + `","branch_id":"` + branchID + `","sector_id":"` + sectorID + `","nome":"Notebook 001","serial_number":"SN-CRUD-001"}`
	status, resp = tc.do(fiber.MethodPost, "/api/v1/assets", assetBody)
	requireStatus(t, "create asset", fiber.StatusCreated, status, resp)
	assetID := resp["id"].(string)

	assetUpdateBody := `{"asset_type_id":"` + assetTypeID + `","branch_id":"` + branchID + `","sector_id":"` + sectorID + `","nome":"Notebook 001 Editado","serial_number":"SN-CRUD-001"}`
	status, resp = tc.do(fiber.MethodPut, "/api/v1/assets/"+assetID, assetUpdateBody)
	requireStatus(t, "update asset", fiber.StatusOK, status, resp)
	if resp["status"] != "available" {
		t.Fatalf("update asset: status deveria ser %q (preservado via COALESCE), veio %v (resp completo: %v)", "available", resp["status"], resp)
	}

	status, resp = tc.do(fiber.MethodGet, "/api/v1/assets", "")
	requireStatus(t, "list assets", fiber.StatusOK, status, resp)
	requireSnakeCaseArray(t, tc, "/api/v1/assets", "nome", "asset_type_id", "branch_id", "sector_id", "status")

	asset2Body := `{"asset_type_id":"` + assetTypeID + `","branch_id":"` + branchID + `","sector_id":"` + sectorID + `","nome":"Notebook Descartável","serial_number":"SN-CRUD-002"}`
	status, resp = tc.do(fiber.MethodPost, "/api/v1/assets", asset2Body)
	requireStatus(t, "create asset2", fiber.StatusCreated, status, resp)
	asset2ID := resp["id"].(string)
	status, _ = tc.do(fiber.MethodDelete, "/api/v1/assets/"+asset2ID, "")
	requireStatus(t, "delete asset2", fiber.StatusOK, status, nil)

	// ---- collaborator: create → update → list → descartável delete ----
	collabBody := `{"branch_id":"` + branchID + `","sector_id":"` + sectorID + `","nome":"Colaborador A","document":"12345678900"}`
	status, resp = tc.do(fiber.MethodPost, "/api/v1/collaborators", collabBody)
	requireStatus(t, "create collaborator", fiber.StatusCreated, status, resp)
	collabID := resp["id"].(string)

	collabUpdateBody := `{"branch_id":"` + branchID + `","sector_id":"` + sectorID + `","nome":"Colaborador A Editado","document":"12345678900"}`
	status, resp = tc.do(fiber.MethodPut, "/api/v1/collaborators/"+collabID, collabUpdateBody)
	requireStatus(t, "update collaborator", fiber.StatusOK, status, resp)

	status, resp = tc.do(fiber.MethodGet, "/api/v1/collaborators", "")
	requireStatus(t, "list collaborators", fiber.StatusOK, status, resp)
	requireSnakeCaseArray(t, tc, "/api/v1/collaborators", "nome", "branch_id", "sector_id", "status")

	collab2Body := `{"branch_id":"` + branchID + `","sector_id":"` + sectorID + `","nome":"Colaborador Descartável","document":"98765432100"}`
	status, resp = tc.do(fiber.MethodPost, "/api/v1/collaborators", collab2Body)
	requireStatus(t, "create collaborator2", fiber.StatusCreated, status, resp)
	collab2ID := resp["id"].(string)
	status, _ = tc.do(fiber.MethodDelete, "/api/v1/collaborators/"+collab2ID, "")
	requireStatus(t, "delete collaborator2", fiber.StatusOK, status, nil)

	// ---- allocation: create → list (cobre ListAllocations) → return ----
	allocBody := `{"asset_id":"` + assetID + `","collaborator_id":"` + collabID + `"}`
	status, resp = tc.do(fiber.MethodPost, "/api/v1/allocations", allocBody)
	requireStatus(t, "create allocation", fiber.StatusCreated, status, resp)
	allocation := resp["allocation"].(map[string]any)
	allocationID := allocation["id"].(string)

	status, resp = tc.do(fiber.MethodGet, "/api/v1/allocations", "")
	requireStatus(t, "list allocations", fiber.StatusOK, status, resp)
	requireSnakeCaseArray(t, tc, "/api/v1/allocations", "asset_id", "collaborator_id", "allocated_at")

	status, _ = tc.do(fiber.MethodPost, "/api/v1/allocations/"+allocationID+"/return", "")
	requireStatus(t, "return allocation", fiber.StatusOK, status, nil)

	// ---- roles: create via API (com permissions) → list ----
	status, resp = tc.do(fiber.MethodPost, "/api/v1/roles", `{"name":"operador","permissions":[{"module":"assets","action":"read"}]}`)
	requireStatus(t, "create role", fiber.StatusCreated, status, resp)
	newRoleID := resp["id"].(string)

	status, resp = tc.do(fiber.MethodGet, "/api/v1/roles", "")
	requireStatus(t, "list roles", fiber.StatusOK, status, resp)
	requireSnakeCaseArray(t, tc, "/api/v1/roles", "name")

	// ---- users: create via API (com scopes) → update → list → delete ----
	newUserEmail := "user-crud-" + uuid.NewString() + "@example.com"
	userBody := `{"email":"` + newUserEmail + `","role_id":"` + newRoleID + `","senha_temporaria":"senha-temp-123","scopes":[{"scope_type":"branch","scope_id":"` + branchID + `"}]}`
	status, resp = tc.do(fiber.MethodPost, "/api/v1/users", userBody)
	requireStatus(t, "create user", fiber.StatusCreated, status, resp)
	newUserID := resp["id"].(string)

	status, resp = tc.do(fiber.MethodGet, "/api/v1/users", "")
	requireStatus(t, "list users", fiber.StatusOK, status, resp)
	// Regressão real de produção: repository.User sem json tags vazava
	// PasswordHash e serializava em PascalCase (Email/RoleID/Status em vez de
	// email/role_id/status) — quebrava a tela de usuários (undefined em tudo)
	// e expunha o hash Argon2id via API. "do" não pegava porque list users
	// devolve array, não objeto, e json.Unmarshal num map falha em silêncio.
	requireSnakeCaseArray(t, tc, "/api/v1/users", "email", "role_id", "status")

	status, resp = tc.do(fiber.MethodPut, "/api/v1/users/"+newUserID, `{"role_id":"`+newRoleID+`","status":"inactive"}`)
	requireStatus(t, "update user", fiber.StatusOK, status, resp)
	if resp["email"] != newUserEmail {
		t.Fatalf("update user: email deveria ser %q, veio %v (resp completo: %v)", newUserEmail, resp["email"], resp)
	}
	if resp["must_change_password"] != true {
		t.Fatalf("update user: must_change_password deveria continuar true (não alterado por este update), veio %v (resp completo: %v)", resp["must_change_password"], resp)
	}

	status, _ = tc.do(fiber.MethodDelete, "/api/v1/users/"+newUserID, "")
	requireStatus(t, "delete user", fiber.StatusOK, status, nil)

	// ---- delete de usuário revoga sessão + libera o e-mail (regressão) ----
	// auth_lookup/session_lookup não têm FK pra users (só sessions tem ON
	// DELETE CASCADE), então excluir sem limpá-los deixava o e-mail preso pra
	// sempre em auth_lookup (recriar batia em "registro duplicado"). A sessão
	// já emitida some por cascata via sessions, mas sem o fix aqui verificado
	// via reversão manual do DeleteUser. Usuário dedicado — não reaproveita
	// newUserID/newUserEmail, que o passo acima já desativou (status
	// inactive), o que por si só bloquearia o login.
	doomedEmail := "doomed-" + uuid.NewString() + "@example.com"
	doomedBody := `{"email":"` + doomedEmail + `","role_id":"` + newRoleID + `","senha_temporaria":"senha-temp-123","scopes":[{"scope_type":"branch","scope_id":"` + branchID + `"}]}`
	status, resp = tc.do(fiber.MethodPost, "/api/v1/users", doomedBody)
	requireStatus(t, "create doomed user", fiber.StatusCreated, status, resp)
	doomedUserID := resp["id"].(string)

	deletedUserClient := &testClient{t: t, fapp: fapp, sessionNm: cfg.SessionCookieName}
	deletedUserClient.do(fiber.MethodGet, "/api/v1/auth/login", "")
	status, resp = deletedUserClient.do(fiber.MethodPost, "/api/v1/auth/login", `{"email":"`+doomedEmail+`","password":"senha-temp-123"}`)
	requireStatus(t, "login usuário a ser excluído", fiber.StatusOK, status, resp)

	status, _ = tc.do(fiber.MethodDelete, "/api/v1/users/"+doomedUserID, "")
	requireStatus(t, "delete doomed user", fiber.StatusOK, status, nil)

	status, _ = deletedUserClient.do(fiber.MethodGet, "/api/v1/dashboard", "")
	if status != fiber.StatusUnauthorized {
		t.Fatalf("sessão do usuário excluído deveria estar revogada (401), obteve %d", status)
	}

	recreateBody := `{"email":"` + doomedEmail + `","role_id":"` + newRoleID + `","senha_temporaria":"senha-temp-456","scopes":[{"scope_type":"branch","scope_id":"` + branchID + `"}]}`
	status, resp = tc.do(fiber.MethodPost, "/api/v1/users", recreateBody)
	requireStatus(t, "recriar usuário com mesmo e-mail após exclusão", fiber.StatusCreated, status, resp)

	// ---- dashboard ----
	status, resp = tc.do(fiber.MethodGet, "/api/v1/dashboard", "")
	requireStatus(t, "dashboard", fiber.StatusOK, status, resp)
}

func requireStatus(t *testing.T, label string, want, got int, resp map[string]any) {
	t.Helper()
	if got != want {
		t.Fatalf("%s: esperava %d, obteve %d (resp=%v)", label, want, got, resp)
	}
}

// requireSnakeCaseArray confere o corpo cru de um endpoint de listagem
// (array JSON — "do" não decodifica isso em map[string]any, então esse
// shape nunca era validado antes). Regressão real: repository.User sem
// json tags vazava PasswordHash e serializava em PascalCase.
func requireSnakeCaseArray(t *testing.T, tc *testClient, path string, mustContain ...string) {
	t.Helper()
	_, raw := tc.doText(fiber.MethodGet, path, "")
	for _, field := range mustContain {
		if !strings.Contains(raw, `"`+field+`"`) {
			t.Fatalf("%s: campo %q ausente ou não está em snake_case: %s", path, field, raw)
		}
	}
	if strings.Contains(raw, "PasswordHash") || strings.Contains(raw, "password_hash") {
		t.Fatalf("%s: vazou hash de senha na resposta: %s", path, raw)
	}
}
