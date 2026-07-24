//go:build integration

package app

import (
	"context"
	"encoding/json"
	"io"
	"net/http/httptest"
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

// client mínimo para exercitar a API real (fiber app.Test) mantendo cookies
// e o header X-CSRF-Token entre chamadas — como um browser faria.
type testClient struct {
	t         *testing.T
	fapp      *fiber.App
	csrf      string
	sessionCk string
	sessionNm string
}

func (tc *testClient) do(method, path, body string) (int, map[string]any) {
	tc.t.Helper()
	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, reader)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if tc.csrf != "" {
		req.Header.Set("Cookie", "csrf_token="+tc.csrf+"; "+tc.sessionNm+"="+tc.sessionCk)
		req.Header.Set("X-CSRF-Token", tc.csrf)
	}
	resp, err := tc.fapp.Test(req)
	if err != nil {
		tc.t.Fatalf("%s %s: %v", method, path, err)
	}
	defer resp.Body.Close()

	for _, ck := range resp.Cookies() {
		switch ck.Name {
		case "csrf_token":
			tc.csrf = ck.Value
		case tc.sessionNm:
			tc.sessionCk = ck.Value
		}
	}

	raw, _ := io.ReadAll(resp.Body)
	var parsed map[string]any
	_ = json.Unmarshal(raw, &parsed)
	return resp.StatusCode, parsed
}

// doText é como do, mas devolve o corpo cru — necessário pra endpoints que
// respondem um array JSON (list *), que "do" não consegue decodificar em
// map[string]any (json.Unmarshal falha silenciosamente nesse caso).
func (tc *testClient) doText(method, path, body string) (int, string) {
	tc.t.Helper()
	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, reader)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if tc.csrf != "" {
		req.Header.Set("Cookie", "csrf_token="+tc.csrf+"; "+tc.sessionNm+"="+tc.sessionCk)
		req.Header.Set("X-CSRF-Token", tc.csrf)
	}
	resp, err := tc.fapp.Test(req)
	if err != nil {
		tc.t.Fatalf("%s %s: %v", method, path, err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(raw)
}

func TestBusinessFlow_EndToEnd(t *testing.T) {
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

	email := "flow-" + uuid.NewString() + "@example.com"
	const initialPassword = "senha-inicial-123"
	const newPassword = "senha-nova-456"

	var tenantID uuid.UUID
	if err := ownerPool.QueryRow(ctx,
		`INSERT INTO tenants (nome, slug) VALUES ('tenant flow', $1) RETURNING id`,
		"tenant-flow-"+uuid.NewString(),
	).Scan(&tenantID); err != nil {
		t.Fatalf("criar tenant: %v", err)
	}

	hash, err := security.HashPassword(initialPassword)
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
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
		// must_change_password = true (default) — testa o gate abaixo.
		userID, err := repository.CreateUserWithLookup(ctx, tx, tenantID, roleID, email, &hash, "local", true)
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

	// 1) prime CSRF — qualquer GET sob /api/v1 aciona o middleware que emite
	// o cookie csrf_token, independentemente de a rota existir.
	tc.do(fiber.MethodGet, "/api/v1/auth/login", "")
	if tc.csrf == "" {
		t.Fatal("csrf token não emitido")
	}

	// 2) login
	loginBody := `{"email":"` + email + `","password":"` + initialPassword + `"}`
	if status, resp := tc.do(fiber.MethodPost, "/api/v1/auth/login", loginBody); status != fiber.StatusOK {
		t.Fatalf("login: status %d, resp %v", status, resp)
	}
	if tc.sessionCk == "" {
		t.Fatal("cookie de sessão não emitido no login")
	}

	// 3) rota de negócio deve estar bloqueada até a troca de senha (403)
	if status, _ := tc.do(fiber.MethodGet, "/api/v1/dashboard", ""); status != fiber.StatusForbidden {
		t.Fatalf("esperava 403 (must_change_password pendente) em /dashboard, obteve %d", status)
	}

	// 4) troca de senha
	changeBody := `{"old":"` + initialPassword + `","new":"` + newPassword + `"}`
	if status, resp := tc.do(fiber.MethodPost, "/api/v1/auth/change-password", changeBody); status != fiber.StatusOK {
		t.Fatalf("change-password: status %d, resp %v", status, resp)
	}

	// 5) agora dashboard deve responder 200
	if status, _ := tc.do(fiber.MethodGet, "/api/v1/dashboard", ""); status != fiber.StatusOK {
		t.Fatalf("esperava 200 em /dashboard após troca de senha, obteve %d", status)
	}

	// 6) fluxo completo dos módulos: company → branch → sector → asset_type → asset → collaborator → allocation → return
	status, resp := tc.do(fiber.MethodPost, "/api/v1/companies", `{"nome":"Empresa Teste","cnpj":"11222333000181"}`)
	if status != fiber.StatusCreated {
		t.Fatalf("create company: status %d, resp %v", status, resp)
	}
	companyID, _ := resp["id"].(string)

	status, resp = tc.do(fiber.MethodPost, "/api/v1/companies/"+companyID+"/branches", `{"nome":"Filial 1","cnpj":"22333444000162"}`)
	if status != fiber.StatusCreated {
		t.Fatalf("create branch: status %d, resp %v", status, resp)
	}
	branchID, _ := resp["id"].(string)

	status, resp = tc.do(fiber.MethodPost, "/api/v1/branches/"+branchID+"/sectors", `{"nome":"TI"}`)
	if status != fiber.StatusCreated {
		t.Fatalf("create sector: status %d, resp %v", status, resp)
	}
	sectorID, _ := resp["id"].(string)

	status, resp = tc.do(fiber.MethodPost, "/api/v1/asset-types", `{"nome":"Notebook"}`)
	if status != fiber.StatusCreated {
		t.Fatalf("create asset_type: status %d, resp %v", status, resp)
	}
	assetTypeID, _ := resp["id"].(string)

	assetBody := `{"asset_type_id":"` + assetTypeID + `","branch_id":"` + branchID + `","sector_id":"` + sectorID + `","nome":"Notebook 001","serial_number":"SN-001"}`
	status, resp = tc.do(fiber.MethodPost, "/api/v1/assets", assetBody)
	if status != fiber.StatusCreated {
		t.Fatalf("create asset: status %d, resp %v", status, resp)
	}
	assetID, _ := resp["id"].(string)

	collabBody := `{"branch_id":"` + branchID + `","sector_id":"` + sectorID + `","nome":"Colaborador 1","document":"12345678900"}`
	status, resp = tc.do(fiber.MethodPost, "/api/v1/collaborators", collabBody)
	if status != fiber.StatusCreated {
		t.Fatalf("create collaborator: status %d, resp %v", status, resp)
	}
	collabID, _ := resp["id"].(string)

	allocBody := `{"asset_id":"` + assetID + `","collaborator_id":"` + collabID + `"}`
	status, resp = tc.do(fiber.MethodPost, "/api/v1/allocations", allocBody)
	if status != fiber.StatusCreated {
		t.Fatalf("create allocation: status %d, resp %v", status, resp)
	}
	allocation, _ := resp["allocation"].(map[string]any)
	allocationID, _ := allocation["id"].(string)
	if allocationID == "" {
		t.Fatalf("resposta de allocation sem id: %v", resp)
	}

	// 2ª alocação ativa do mesmo asset deve ser rejeitada (409) mesmo pela API.
	status, _ = tc.do(fiber.MethodPost, "/api/v1/allocations", allocBody)
	if status != fiber.StatusConflict {
		t.Fatalf("esperava 409 na 2ª alocação ativa via API, obteve %d", status)
	}

	status, _ = tc.do(fiber.MethodPost, "/api/v1/allocations/"+allocationID+"/return", "")
	if status != fiber.StatusOK {
		t.Fatalf("return allocation: status %d", status)
	}

	// DELETE de asset_type em uso deve ser rejeitado com 409 (ON DELETE RESTRICT).
	status, _ = tc.do(fiber.MethodDelete, "/api/v1/asset-types/"+assetTypeID, "")
	if status != fiber.StatusConflict {
		t.Fatalf("esperava 409 ao remover asset_type em uso, obteve %d", status)
	}

	// import idempotente: mesma idempotency_key não duplica.
	importBody := `{"idempotency_key":"import-1-` + uuid.NewString() + `","data":[{"asset_type_id":"` + assetTypeID + `","branch_id":"` + branchID + `","sector_id":"` + sectorID + `","nome":"Notebook Import 1"}]}`
	status, resp = tc.do(fiber.MethodPost, "/api/v1/import", importBody)
	if status != fiber.StatusOK {
		t.Fatalf("import: status %d, resp %v", status, resp)
	}
	if count, _ := resp["count"].(float64); count != 1 {
		t.Fatalf("esperava count=1 no import, obteve %v", resp["count"])
	}

	status, resp2 := tc.do(fiber.MethodPost, "/api/v1/import", importBody)
	if status != fiber.StatusOK {
		t.Fatalf("reimport: status %d", status)
	}
	if count, _ := resp2["count"].(float64); count != 1 {
		t.Fatalf("reimport com mesma idempotency_key deveria devolver count=1 sem duplicar, obteve %v", resp2["count"])
	}

	// logout
	status, _ = tc.do(fiber.MethodPost, "/api/v1/auth/logout", "")
	if status != fiber.StatusOK {
		t.Fatalf("logout: status %d", status)
	}

	// sessão revogada não deve mais acessar rotas protegidas.
	status, _ = tc.do(fiber.MethodGet, "/api/v1/dashboard", "")
	if status != fiber.StatusUnauthorized {
		t.Fatalf("esperava 401 após logout, obteve %d", status)
	}
}
