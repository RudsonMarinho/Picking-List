// Package seed cria o tenant demo completo (R10) — plano trial → tenant →
// role admin com permissões → usuário admin → empresa → filial → setor →
// colaborador → tipo de ativo → ativo → alocação com Termo de
// Responsabilidade gerado. Idempotente: se o tenant "demo" já existir, não
// faz nada (seguro rodar mais de uma vez em dev).
//
// NUNCA roda em produção — FAIL-FAST barra `-seed` com APP_ENV=production
// antes mesmo de chegar aqui (ver pkg/config.CheckFailFast).
package seed

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/Fortcargo/invtech/internal/repository"
	"github.com/Fortcargo/invtech/pkg/security"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	demoSlug            = "demo"
	demoAdminEmail      = "admin@demo.invtech.local"
	demoAdminTempPasswd = "TrocarSenha123!"
)

var demoModules = []string{
	"dashboard", "companies", "branches", "sectors", "collaborators",
	"asset_types", "assets", "allocations", "users", "roles", "import",
}
var demoActions = []string{"create", "read", "update", "delete"}

func Run(ctx context.Context, pool *pgxpool.Pool, logg *slog.Logger) error {
	var existingTenantID uuid.UUID
	err := pool.QueryRow(ctx, `SELECT id FROM tenants WHERE slug = $1`, demoSlug).Scan(&existingTenantID)
	if err == nil {
		logg.Info("seed: tenant demo já existe, nada a fazer", slog.String("tenant_id", existingTenantID.String()))
		return nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("checar tenant demo: %w", err)
	}

	var planID uuid.UUID
	if err := pool.QueryRow(ctx,
		`INSERT INTO plans (codigo, limites) VALUES ('trial', $1::jsonb) RETURNING id`,
		`{"max_assets":50,"max_users":5,"max_branches":3}`,
	).Scan(&planID); err != nil {
		return fmt.Errorf("plano trial: %w", err)
	}

	var tenantID uuid.UUID
	if err := pool.QueryRow(ctx,
		`INSERT INTO tenants (nome, slug) VALUES ('Tenant Demo', $1) RETURNING id`, demoSlug,
	).Scan(&tenantID); err != nil {
		return fmt.Errorf("tenant demo: %w", err)
	}

	if _, err := pool.Exec(ctx,
		`INSERT INTO subscriptions (tenant_id, plan_id, status, current_period_end)
		 VALUES ($1, $2, 'trialing', now() + interval '30 days')`,
		tenantID, planID); err != nil {
		return fmt.Errorf("subscription: %w", err)
	}

	err = repository.WithTenant(ctx, pool, tenantID, func(tx pgx.Tx) error {
		role, err := repository.CreateRole(ctx, tx, tenantID, "admin", strPtr("Administrador do tenant"))
		if err != nil {
			return fmt.Errorf("role admin: %w", err)
		}
		perms := make([]repository.Permission, 0, len(demoModules)*len(demoActions))
		for _, m := range demoModules {
			for _, a := range demoActions {
				perms = append(perms, repository.Permission{Module: m, Action: a})
			}
		}
		if err := repository.AddRolePermissions(ctx, tx, tenantID, role.ID, perms); err != nil {
			return fmt.Errorf("permissões da role admin: %w", err)
		}

		hash, err := security.HashPassword(demoAdminTempPasswd)
		if err != nil {
			return fmt.Errorf("hash senha admin: %w", err)
		}
		userID, err := repository.CreateUserWithLookup(ctx, tx, tenantID, role.ID, demoAdminEmail, &hash, "local", true)
		if err != nil {
			return fmt.Errorf("usuário admin: %w", err)
		}
		if err := repository.CreateUserScopes(ctx, tx, tenantID, userID, []repository.UserScope{{ScopeType: "all"}}); err != nil {
			return fmt.Errorf("scope do admin: %w", err)
		}

		company, err := repository.CreateCompany(ctx, tx, tenantID, "Empresa Demo", "11222333000181", nil, nil)
		if err != nil {
			return fmt.Errorf("empresa demo: %w", err)
		}
		branch, err := repository.CreateBranch(ctx, tx, tenantID, company.ID, "Filial Matriz", "22333444000162", nil, nil)
		if err != nil {
			return fmt.Errorf("filial demo: %w", err)
		}
		sector, err := repository.CreateSector(ctx, tx, tenantID, branch.ID, "TI", nil)
		if err != nil {
			return fmt.Errorf("setor demo: %w", err)
		}
		document := "12345678900"
		collaborator, err := repository.CreateCollaborator(ctx, tx, tenantID, branch.ID, sector.ID, "Colaborador Demo",
			repository.CollaboratorDetails{Document: &document})
		if err != nil {
			return fmt.Errorf("colaborador demo: %w", err)
		}
		assetType, err := repository.CreateAssetType(ctx, tx, tenantID, "Notebook", nil, nil)
		if err != nil {
			return fmt.Errorf("tipo de ativo demo: %w", err)
		}
		serial := "SN-DEMO-0001"
		asset, err := repository.CreateAsset(ctx, tx, tenantID, assetType.ID, branch.ID, sector.ID, "Notebook Demo 001", &serial, repository.AssetDetails{})
		if err != nil {
			return fmt.Errorf("ativo demo: %w", err)
		}

		if _, err := repository.CreateAllocation(ctx, tx, tenantID, asset.ID, collaborator.ID, asset.Nome, collaborator.Nome); err != nil {
			return fmt.Errorf("alocação demo: %w", err)
		}

		return nil
	})
	if err != nil {
		return err
	}

	logg.Info("seed concluído",
		slog.String("tenant_slug", demoSlug),
		slog.String("admin_email", demoAdminEmail),
		slog.String("admin_temp_password", demoAdminTempPasswd),
	)
	return nil
}

func strPtr(s string) *string { return &s }
