//go:build integration

// Regressão do enforcement de RBAC (role_permissions/user_scopes): antes
// desta rodada, os dois eram gravados mas nenhum middleware os lia — qualquer
// usuário autenticado acessava qualquer módulo, sem respeitar role/escopo.
package app

import (
	"context"
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

func TestAuthz_PermissionAndScopeEnforcement(t *testing.T) {
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

	var tenantID uuid.UUID
	if err := ownerPool.QueryRow(ctx,
		`INSERT INTO tenants (nome, slug) VALUES ('tenant authz', $1) RETURNING id`,
		"tenant-authz-"+uuid.NewString(),
	).Scan(&tenantID); err != nil {
		t.Fatalf("criar tenant: %v", err)
	}

	// Fixtures: empresa → filial A e filial B → setor em cada uma + asset_type
	// compartilhado — tudo criado direto via repository (sem passar pelo
	// gate de RBAC, que é o que este teste está verificando).
	var companyID, branchAID, branchBID, sectorAID, assetTypeID uuid.UUID
	restrictedPassword := "senha-restrita-123"
	restrictedHash, err := security.HashPassword(restrictedPassword)
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	restrictedEmail := "restrito-" + uuid.NewString() + "@example.com"

	err = repository.WithTenant(ctx, appPool, tenantID, func(tx pgx.Tx) error {
		company, err := repository.CreateCompany(ctx, tx, tenantID, "Empresa Authz", "11222333000181", nil, nil)
		if err != nil {
			return err
		}
		companyID = company.ID

		branchA, err := repository.CreateBranch(ctx, tx, tenantID, companyID, "Filial A", "22333444000162", nil, nil)
		if err != nil {
			return err
		}
		branchAID = branchA.ID
		branchB, err := repository.CreateBranch(ctx, tx, tenantID, companyID, "Filial B", "33444555000143", nil, nil)
		if err != nil {
			return err
		}
		branchBID = branchB.ID

		sectorA, err := repository.CreateSector(ctx, tx, tenantID, branchAID, "Setor A", nil)
		if err != nil {
			return err
		}
		sectorAID = sectorA.ID

		assetType, err := repository.CreateAssetType(ctx, tx, tenantID, "Notebook Authz", nil, nil)
		if err != nil {
			return err
		}
		assetTypeID = assetType.ID

		// role restrita: só "assets" (todas as ações) — sem "companies".
		role, err := repository.CreateRole(ctx, tx, tenantID, "restrito", nil)
		if err != nil {
			return err
		}
		perms := []repository.Permission{
			{Module: "assets", Action: "create"}, {Module: "assets", Action: "read"},
			{Module: "assets", Action: "update"}, {Module: "assets", Action: "delete"},
		}
		if err := repository.AddRolePermissions(ctx, tx, tenantID, role.ID, perms); err != nil {
			return err
		}

		userID, err := repository.CreateUserWithLookup(ctx, tx, tenantID, role.ID, restrictedEmail, &restrictedHash, "local", false)
		if err != nil {
			return err
		}
		// escopo: só a filial A — filial B fica fora do alcance do usuário.
		return repository.CreateUserScopes(ctx, tx, tenantID, userID, []repository.UserScope{
			{ScopeType: "branch", ScopeID: &branchAID},
		})
	})
	if err != nil {
		t.Fatalf("montar fixtures: %v", err)
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
	loginBody := `{"email":"` + restrictedEmail + `","password":"` + restrictedPassword + `"}`
	if status, resp := tc.do(fiber.MethodPost, "/api/v1/auth/login", loginBody); status != fiber.StatusOK {
		t.Fatalf("login usuário restrito: status %d, resp %v", status, resp)
	}

	// ---- permissão: role não tem "companies" — 403, não importa o escopo ----
	if status, resp := tc.do(fiber.MethodGet, "/api/v1/companies", ""); status != fiber.StatusForbidden {
		t.Fatalf("GET /companies sem permissão: esperava 403, obteve %d (resp=%v)", status, resp)
	}

	// ---- escopo: role TEM "assets", mas filial B está fora do escopo ----
	assetBodyBranchB := `{"asset_type_id":"` + assetTypeID.String() + `","branch_id":"` + branchBID.String() + `","sector_id":"` + sectorAID.String() + `","nome":"Notebook Fora de Escopo"}`
	if status, resp := tc.do(fiber.MethodPost, "/api/v1/assets", assetBodyBranchB); status != fiber.StatusForbidden {
		t.Fatalf("create asset em filial fora do escopo: esperava 403, obteve %d (resp=%v)", status, resp)
	}

	// ---- escopo: filial A está dentro do escopo — deve funcionar ----
	assetBodyBranchA := `{"asset_type_id":"` + assetTypeID.String() + `","branch_id":"` + branchAID.String() + `","sector_id":"` + sectorAID.String() + `","nome":"Notebook Dentro do Escopo","serial_number":"SN-AUTHZ-001"}`
	status, resp := tc.do(fiber.MethodPost, "/api/v1/assets", assetBodyBranchA)
	requireStatus(t, "create asset em filial dentro do escopo", fiber.StatusCreated, status, resp)
	assetID := resp["id"].(string)

	// ---- criar empresa exige escopo 'all', mesmo que a role tivesse permissão ----
	// (aqui a role nem tem permissão de "companies" — 403 chega antes pelo
	// gate de permissão; o teste de escopo.All puro fica coberto pelos testes
	// que já usam o usuário admin com scope 'all' em business_flow_test.go.)

	if status, _ = tc.do(fiber.MethodDelete, "/api/v1/assets/"+assetID, ""); status != fiber.StatusOK {
		t.Fatalf("delete asset dentro do escopo: esperava 200, obteve %d", status)
	}
}
