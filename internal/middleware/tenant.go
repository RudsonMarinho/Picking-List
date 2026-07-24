// Package middleware implementa a resolução de tenant (sessão →
// SET LOCAL app.tenant_id), o gate de troca de senha obrigatória e o CSRF
// double-submit — nesta ordem quando compostos numa rota.
package middleware

import (
	"fmt"
	"time"

	"github.com/Fortcargo/invtech/internal/repository"
	"github.com/Fortcargo/invtech/pkg/security"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Chaves de fiber.Locals preenchidas por RequireSession.
const (
	LocalTx                 = "invtech_tx"
	LocalTenantID           = "invtech_tenant_id"
	LocalUserID             = "invtech_user_id"
	LocalRoleID             = "invtech_role_id"
	LocalMustChangePassword = "invtech_must_change_password"
	LocalSessionTokenHash   = "invtech_session_token_hash"
)

// Tx recupera a transação da requisição atual (aberta por RequireSession,
// já com SET LOCAL app.tenant_id aplicado). Handlers de rotas protegidas
// SEMPRE devem usar esta Tx — nunca abrir outra ou usar o pool cru — senão
// operam fora do escopo do tenant e o FORCE RLS rejeita tudo.
func Tx(c *fiber.Ctx) pgx.Tx {
	return c.Locals(LocalTx).(pgx.Tx)
}

func TenantID(c *fiber.Ctx) uuid.UUID {
	return c.Locals(LocalTenantID).(uuid.UUID)
}

func UserID(c *fiber.Ctx) uuid.UUID {
	return c.Locals(LocalUserID).(uuid.UUID)
}

func RoleID(c *fiber.Ctx) uuid.UUID {
	return c.Locals(LocalRoleID).(uuid.UUID)
}

func MustChangePassword(c *fiber.Ctx) bool {
	v, _ := c.Locals(LocalMustChangePassword).(bool)
	return v
}

// RequireSession resolve o cookie de sessão, valida contra session_lookup
// (pré-tenant) e sessions (pós SET LOCAL), e abre uma transação que
// permanece aberta durante todo o handler — fechada (commit/rollback) só
// no fim desta própria middleware, depois que c.Next() retorna.
//
// Fluxo obrigatório de ARQUITETURA.md § lookups pré-tenant:
// 1. cookie → session_lookup (sem RLS) → tenant_id
// 2. abre Tx, SET LOCAL app.tenant_id
// 3. reconfirma em sessions (com RLS) que a sessão existe e não foi revogada
func RequireSession(pool *pgxpool.Pool, sessionCookieName string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ctx := c.Context()

		rawToken := c.Cookies(sessionCookieName)
		if rawToken == "" {
			return fiber.NewError(fiber.StatusUnauthorized, "sessão ausente")
		}
		tokenHash := security.HashToken(rawToken)

		lookup, err := repository.FindSessionLookupByHash(ctx, pool, tokenHash)
		if err != nil {
			return fiber.NewError(fiber.StatusUnauthorized, "sessão inválida")
		}
		if lookup.ExpiresAt.Before(time.Now().UTC()) {
			return fiber.NewError(fiber.StatusUnauthorized, "sessão expirada")
		}

		tx, err := pool.Begin(ctx)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "falha ao abrir transação")
		}
		committed := false
		defer func() {
			if !committed {
				_ = tx.Rollback(ctx)
			}
		}()

		if _, err := tx.Exec(ctx, fmt.Sprintf("SET LOCAL app.tenant_id = '%s'", lookup.TenantID.String())); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "falha ao aplicar tenant")
		}

		valid, err := repository.IsSessionValid(ctx, tx, lookup.TenantID, lookup.UserID, tokenHash)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "falha ao validar sessão")
		}
		if !valid {
			return fiber.NewError(fiber.StatusUnauthorized, "sessão revogada")
		}

		user, err := repository.FindUserByID(ctx, tx, lookup.TenantID, lookup.UserID)
		if err != nil || user.Status != "active" {
			return fiber.NewError(fiber.StatusUnauthorized, "usuário inválido")
		}

		c.Locals(LocalTx, tx)
		c.Locals(LocalTenantID, lookup.TenantID)
		c.Locals(LocalUserID, lookup.UserID)
		c.Locals(LocalRoleID, user.RoleID)
		c.Locals(LocalMustChangePassword, user.MustChangePassword)
		c.Locals(LocalSessionTokenHash, tokenHash)

		handlerErr := c.Next()

		if handlerErr != nil {
			return handlerErr
		}
		if c.Response().StatusCode() >= 400 {
			return nil
		}
		if err := tx.Commit(ctx); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "falha ao commitar transação")
		}
		committed = true
		return nil
	}
}

// RequirePasswordChanged bloqueia rotas de negócio até a troca de senha
// obrigatória (must_change_password=true). Deve vir DEPOIS de
// RequireSession na cadeia, e NUNCA na rota de troca de senha em si.
func RequirePasswordChanged() fiber.Handler {
	return func(c *fiber.Ctx) error {
		if MustChangePassword(c) {
			return fiber.NewError(fiber.StatusForbidden, "troca de senha obrigatória pendente")
		}
		return c.Next()
	}
}
