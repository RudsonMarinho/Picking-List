package handlers

import (
	"github.com/Fortcargo/invtech/internal/middleware"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

// Checagem de escopo (empresa/filial) por cima do gate de permissão
// (module/action) já aplicado em routes.go — asset_type/user/role/import não
// entram aqui de propósito: são catálogos/administração de tenant, não
// dados vinculados a uma empresa/filial específica.

func requireCompanyScope(c *fiber.Ctx, companyID uuid.UUID) error {
	scope, err := middleware.Scope(c)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "falha ao verificar escopo")
	}
	if !scope.AllowsCompany(companyID) {
		return fiber.NewError(fiber.StatusForbidden, "fora do escopo de acesso do usuário")
	}
	return nil
}

func requireBranchScope(c *fiber.Ctx, branchID uuid.UUID) error {
	scope, err := middleware.Scope(c)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "falha ao verificar escopo")
	}
	if !scope.AllowsBranch(branchID) {
		return fiber.NewError(fiber.StatusForbidden, "fora do escopo de acesso do usuário")
	}
	return nil
}

// requireUnrestrictedScope — criar uma empresa é uma ação fora de qualquer
// hierarquia existente (não há empresa/filial "dona" pra checar contra o
// escopo do usuário), então só usuário com escopo 'all' pode.
func requireUnrestrictedScope(c *fiber.Ctx) error {
	scope, err := middleware.Scope(c)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "falha ao verificar escopo")
	}
	if !scope.All {
		return fiber.NewError(fiber.StatusForbidden, "fora do escopo de acesso do usuário")
	}
	return nil
}
