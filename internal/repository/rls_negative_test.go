//go:build integration

// Suíte negativa de nível banco (ARQUITETURA.md § Suíte negativa).
// Roda contra um Postgres real com as migrations aplicadas — não é
// substituível por unit test com mocks, porque o que está sendo provado é
// o comportamento das políticas RLS e dos GRANTs, não código Go.
//
// Requer as variáveis TEST_DB_HOST, TEST_DB_PORT, TEST_DB_NAME,
// TEST_DB_OWNER_USER, TEST_DB_OWNER_PASSWORD, TEST_DB_APP_USER,
// TEST_DB_APP_PASSWORD apontando para um banco com as migrations
// 000000..000003 já aplicadas.
package repository

import (
	"context"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func testParams(t *testing.T) (owner, app DBParams) {
	t.Helper()
	host := os.Getenv("TEST_DB_HOST")
	if host == "" {
		t.Skip("TEST_DB_HOST não definido — pulando suíte negativa de integração")
	}
	owner = DBParams{
		Host:     host,
		Port:     os.Getenv("TEST_DB_PORT"),
		Database: os.Getenv("TEST_DB_NAME"),
		User:     os.Getenv("TEST_DB_OWNER_USER"),
		Password: os.Getenv("TEST_DB_OWNER_PASSWORD"),
	}
	app = DBParams{
		Host:     host,
		Port:     os.Getenv("TEST_DB_PORT"),
		Database: os.Getenv("TEST_DB_NAME"),
		User:     os.Getenv("TEST_DB_APP_USER"),
		Password: os.Getenv("TEST_DB_APP_PASSWORD"),
	}
	return owner, app
}

// newTenant cria um tenant via pool owner (tenants não tem RLS — é a raiz).
func newTenant(t *testing.T, ctx context.Context, ownerPool *pgxpool.Pool, slug string) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	err := ownerPool.QueryRow(ctx,
		`INSERT INTO tenants (nome, slug) VALUES ($1, $2) RETURNING id`,
		"tenant "+slug, slug,
	).Scan(&id)
	if err != nil {
		t.Fatalf("criar tenant: %v", err)
	}
	return id
}

// newRole cria uma role dentro do tenant (via SET LOCAL, como faria o app).
func newRole(t *testing.T, ctx context.Context, appPool *pgxpool.Pool, tenantID uuid.UUID, name string) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	err := WithTenant(ctx, appPool, tenantID, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx,
			`INSERT INTO roles (tenant_id, name) VALUES ($1, $2) RETURNING id`,
			tenantID, name,
		).Scan(&id)
	})
	if err != nil {
		t.Fatalf("criar role: %v", err)
	}
	return id
}

func TestRLS_TenantIsolation(t *testing.T) {
	ownerParams, appParams := testParams(t)
	ctx := context.Background()

	ownerPool, err := NewPool(ctx, ownerParams)
	if err != nil {
		t.Fatalf("pool owner: %v", err)
	}
	defer ownerPool.Close()
	appPool, err := NewPool(ctx, appParams)
	if err != nil {
		t.Fatalf("pool app: %v", err)
	}
	defer appPool.Close()

	tenantA := newTenant(t, ctx, ownerPool, "tenant-a-"+uuid.NewString())
	tenantB := newTenant(t, ctx, ownerPool, "tenant-b-"+uuid.NewString())
	newRole(t, ctx, appPool, tenantA, "role-a")

	var count int
	err = WithTenant(ctx, appPool, tenantB, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx, `SELECT count(*) FROM roles`).Scan(&count)
	})
	if err != nil {
		t.Fatalf("query como tenant B: %v", err)
	}
	if count != 0 {
		t.Fatalf("tenant B enxergou %d linha(s) do tenant A — RLS vazando", count)
	}
}

func TestRLS_InsertForeignTenantRejected(t *testing.T) {
	ownerParams, appParams := testParams(t)
	ctx := context.Background()

	ownerPool, err := NewPool(ctx, ownerParams)
	if err != nil {
		t.Fatalf("pool owner: %v", err)
	}
	defer ownerPool.Close()
	appPool, err := NewPool(ctx, appParams)
	if err != nil {
		t.Fatalf("pool app: %v", err)
	}
	defer appPool.Close()

	tenantA := newTenant(t, ctx, ownerPool, "tenant-a-"+uuid.NewString())
	tenantB := newTenant(t, ctx, ownerPool, "tenant-b-"+uuid.NewString())

	err = WithTenant(ctx, appPool, tenantA, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx,
			`INSERT INTO roles (tenant_id, name) VALUES ($1, 'cross-tenant')`, tenantB)
		return err
	})
	if err == nil {
		t.Fatal("INSERT com tenant_id alheio deveria ser rejeitado pelo WITH CHECK, mas passou")
	}
}

func TestRLS_AppendOnlyAuditAndErrorsLog(t *testing.T) {
	_, appParams := testParams(t)
	ctx := context.Background()

	appPool, err := NewPool(ctx, appParams)
	if err != nil {
		t.Fatalf("pool app: %v", err)
	}
	defer appPool.Close()

	for _, table := range []string{"audit_log", "errors_log"} {
		_, err := appPool.Exec(ctx, "UPDATE "+table+" SET created_at = now() WHERE false")
		if err == nil {
			t.Fatalf("invtech_app conseguiu UPDATE em %s — deveria ser append-only", table)
		}
	}
}

func TestRLS_UserScopeAllWithNullScopeID(t *testing.T) {
	ownerParams, appParams := testParams(t)
	ctx := context.Background()

	ownerPool, err := NewPool(ctx, ownerParams)
	if err != nil {
		t.Fatalf("pool owner: %v", err)
	}
	defer ownerPool.Close()
	appPool, err := NewPool(ctx, appParams)
	if err != nil {
		t.Fatalf("pool app: %v", err)
	}
	defer appPool.Close()

	tenant := newTenant(t, ctx, ownerPool, "tenant-scope-"+uuid.NewString())
	roleID := newRole(t, ctx, appPool, tenant, "role-scope")

	err = WithTenant(ctx, appPool, tenant, func(tx pgx.Tx) error {
		var userID uuid.UUID
		if err := tx.QueryRow(ctx,
			`INSERT INTO users (tenant_id, role_id, email, password_hash, auth_provider)
			 VALUES ($1, $2, $3, 'hash', 'local') RETURNING id`,
			tenant, roleID, "user-"+uuid.NewString()+"@example.com",
		).Scan(&userID); err != nil {
			return err
		}
		_, err := tx.Exec(ctx,
			`INSERT INTO user_scopes (tenant_id, user_id, scope_type, scope_id) VALUES ($1, $2, 'all', NULL)`,
			tenant, userID)
		return err
	})
	if err != nil {
		t.Fatalf("scope_type='all' com scope_id NULL deveria inserir com sucesso: %v", err)
	}
}

// fixture completa para os testes de asset/collaborator × sector/branch.
type orgFixture struct {
	tenant    uuid.UUID
	branch1   uuid.UUID
	branch2   uuid.UUID
	sectorIn2 uuid.UUID // setor que pertence ao branch2
}

func newOrgFixture(t *testing.T, ctx context.Context, appPool *pgxpool.Pool, ownerPool *pgxpool.Pool) orgFixture {
	t.Helper()
	tenant := newTenant(t, ctx, ownerPool, "tenant-org-"+uuid.NewString())
	f := orgFixture{tenant: tenant}

	err := WithTenant(ctx, appPool, tenant, func(tx pgx.Tx) error {
		var companyID uuid.UUID
		if err := tx.QueryRow(ctx,
			`INSERT INTO company (tenant_id, nome, cnpj) VALUES ($1, 'empresa', '11111111000191') RETURNING id`,
			tenant).Scan(&companyID); err != nil {
			return err
		}
		if err := tx.QueryRow(ctx,
			`INSERT INTO branch (tenant_id, company_id, nome, cnpj) VALUES ($1, $2, 'filial 1', '22222222000172') RETURNING id`,
			tenant, companyID).Scan(&f.branch1); err != nil {
			return err
		}
		if err := tx.QueryRow(ctx,
			`INSERT INTO branch (tenant_id, company_id, nome, cnpj) VALUES ($1, $2, 'filial 2', '33333333000153') RETURNING id`,
			tenant, companyID).Scan(&f.branch2); err != nil {
			return err
		}
		return tx.QueryRow(ctx,
			`INSERT INTO sector (tenant_id, branch_id, nome) VALUES ($1, $2, 'setor filial 2') RETURNING id`,
			tenant, f.branch2).Scan(&f.sectorIn2)
	})
	if err != nil {
		t.Fatalf("fixture org: %v", err)
	}
	return f
}

func TestRLS_AssetRejectsSectorFromDifferentBranch(t *testing.T) {
	ownerParams, appParams := testParams(t)
	ctx := context.Background()

	ownerPool, err := NewPool(ctx, ownerParams)
	if err != nil {
		t.Fatalf("pool owner: %v", err)
	}
	defer ownerPool.Close()
	appPool, err := NewPool(ctx, appParams)
	if err != nil {
		t.Fatalf("pool app: %v", err)
	}
	defer appPool.Close()

	f := newOrgFixture(t, ctx, appPool, ownerPool)

	err = WithTenant(ctx, appPool, f.tenant, func(tx pgx.Tx) error {
		var assetTypeID uuid.UUID
		if err := tx.QueryRow(ctx,
			`INSERT INTO asset_type (tenant_id, nome) VALUES ($1, 'notebook') RETURNING id`,
			f.tenant).Scan(&assetTypeID); err != nil {
			return err
		}
		// branch1 + setor que pertence ao branch2 → deve violar a FK composta (R2)
		_, err := tx.Exec(ctx,
			`INSERT INTO asset (tenant_id, asset_type_id, branch_id, sector_id, nome)
			 VALUES ($1, $2, $3, $4, 'notebook 1')`,
			f.tenant, assetTypeID, f.branch1, f.sectorIn2)
		return err
	})
	if err == nil {
		t.Fatal("asset com sector_id de outra filial deveria ser rejeitado pela FK composta")
	}
}

func TestRLS_CollaboratorRejectsSectorFromDifferentBranch(t *testing.T) {
	ownerParams, appParams := testParams(t)
	ctx := context.Background()

	ownerPool, err := NewPool(ctx, ownerParams)
	if err != nil {
		t.Fatalf("pool owner: %v", err)
	}
	defer ownerPool.Close()
	appPool, err := NewPool(ctx, appParams)
	if err != nil {
		t.Fatalf("pool app: %v", err)
	}
	defer appPool.Close()

	f := newOrgFixture(t, ctx, appPool, ownerPool)

	err = WithTenant(ctx, appPool, f.tenant, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx,
			`INSERT INTO collaborator (tenant_id, branch_id, sector_id, nome, document)
			 VALUES ($1, $2, $3, 'colaborador', '12345678900')`,
			f.tenant, f.branch1, f.sectorIn2)
		return err
	})
	if err == nil {
		t.Fatal("collaborator com sector_id de outra filial deveria ser rejeitado pela FK composta")
	}
}

func TestRLS_AllocationSecondActiveRejected(t *testing.T) {
	ownerParams, appParams := testParams(t)
	ctx := context.Background()

	ownerPool, err := NewPool(ctx, ownerParams)
	if err != nil {
		t.Fatalf("pool owner: %v", err)
	}
	defer ownerPool.Close()
	appPool, err := NewPool(ctx, appParams)
	if err != nil {
		t.Fatalf("pool app: %v", err)
	}
	defer appPool.Close()

	f := newOrgFixture(t, ctx, appPool, ownerPool)

	err = WithTenant(ctx, appPool, f.tenant, func(tx pgx.Tx) error {
		var assetTypeID, assetID, collabID uuid.UUID
		if err := tx.QueryRow(ctx,
			`INSERT INTO asset_type (tenant_id, nome) VALUES ($1, 'notebook') RETURNING id`,
			f.tenant).Scan(&assetTypeID); err != nil {
			return err
		}
		if err := tx.QueryRow(ctx,
			`INSERT INTO asset (tenant_id, asset_type_id, branch_id, sector_id, nome)
			 VALUES ($1, $2, $3, $4, 'notebook 1') RETURNING id`,
			f.tenant, assetTypeID, f.branch2, f.sectorIn2).Scan(&assetID); err != nil {
			return err
		}
		if err := tx.QueryRow(ctx,
			`INSERT INTO collaborator (tenant_id, branch_id, sector_id, nome, document)
			 VALUES ($1, $2, $3, 'colaborador', '12345678900') RETURNING id`,
			f.tenant, f.branch2, f.sectorIn2).Scan(&collabID); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx,
			`INSERT INTO allocation (tenant_id, asset_id, collaborator_id) VALUES ($1, $2, $3)`,
			f.tenant, assetID, collabID); err != nil {
			return err
		}
		// 2ª alocação ativa do mesmo asset — deve violar ux_allocation_ativa
		_, err := tx.Exec(ctx,
			`INSERT INTO allocation (tenant_id, asset_id, collaborator_id) VALUES ($1, $2, $3)`,
			f.tenant, assetID, collabID)
		return err
	})
	if err == nil {
		t.Fatal("2ª alocação ativa do mesmo asset deveria ser rejeitada")
	}
}

func TestRLS_FailClosedWithoutSetLocal(t *testing.T) {
	_, appParams := testParams(t)
	ctx := context.Background()

	appPool, err := NewPool(ctx, appParams)
	if err != nil {
		t.Fatalf("pool app: %v", err)
	}
	defer appPool.Close()

	// Sem SET LOCAL app.tenant_id — a política usa current_setting() sem o
	// segundo argumento (missing_ok), então deve falhar explicitamente em
	// vez de silenciosamente devolver tudo ou nada sem erro.
	_, err = appPool.Exec(ctx, `SELECT * FROM roles`)
	if err == nil {
		t.Fatal("query sem SET LOCAL app.tenant_id deveria falhar (fail-closed), mas passou")
	}
}
