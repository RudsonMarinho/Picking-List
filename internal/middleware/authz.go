package middleware

import (
	"github.com/Fortcargo/invtech/internal/repository"
	"github.com/gofiber/fiber/v2"
)

// LocalScope guarda o ResolvedScope do usuário atual — carregado uma única
// vez por requisição (primeira chamada a Scope()), já que vários handlers
// de uma mesma rota podem precisar dele.
const LocalScope = "invtech_scope"

// RequirePermission bloqueia a rota se a role do usuário não tiver o par
// (module, action) em role_permissions — DEVE vir depois de RequireSession
// (precisa de Tx/TenantID/RoleID em Locals) e depois de RequirePasswordChanged.
func RequirePermission(module, action string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		ok, err := repository.HasPermission(c.Context(), Tx(c), TenantID(c), RoleID(c), module, action)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "falha ao verificar permissão")
		}
		if !ok {
			return fiber.NewError(fiber.StatusForbidden, "sem permissão para "+action+" em "+module)
		}
		return c.Next()
	}
}

// Scope resolve (e cacheia em Locals, por requisição) o escopo de
// empresa/filial do usuário atual — usado pelos handlers dos módulos
// vinculados a empresa/filial pra autorizar create/update/delete.
func Scope(c *fiber.Ctx) (repository.ResolvedScope, error) {
	if cached, ok := c.Locals(LocalScope).(repository.ResolvedScope); ok {
		return cached, nil
	}
	scope, err := repository.ResolveUserScope(c.Context(), Tx(c), TenantID(c), UserID(c))
	if err != nil {
		return repository.ResolvedScope{}, err
	}
	c.Locals(LocalScope, scope)
	return scope, nil
}
