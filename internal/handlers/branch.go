package handlers

import (
	"github.com/Fortcargo/invtech/internal/middleware"
	"github.com/Fortcargo/invtech/internal/repository"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type branchRequest struct {
	Nome  string  `json:"nome"`
	CNPJ  string  `json:"cnpj"`
	City  *string `json:"city,omitempty"`
	State *string `json:"state,omitempty"`
}

func ListBranches(c *fiber.Ctx) error {
	companyID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "id de empresa inválido")
	}
	list, err := repository.ListBranchesByCompany(c.Context(), middleware.Tx(c), middleware.TenantID(c), companyID)
	if err != nil {
		return mapDBError(err)
	}
	return c.JSON(list)
}

func CreateBranch(c *fiber.Ctx) error {
	companyID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "id de empresa inválido")
	}
	if err := requireCompanyScope(c, companyID); err != nil {
		return err
	}
	var body branchRequest
	if err := c.BodyParser(&body); err != nil || body.Nome == "" || body.CNPJ == "" {
		return fiber.NewError(fiber.StatusBadRequest, "nome e cnpj são obrigatórios")
	}
	out, err := repository.CreateBranch(c.Context(), middleware.Tx(c), middleware.TenantID(c), companyID, body.Nome, body.CNPJ, body.City, body.State)
	if err != nil {
		return mapDBError(err)
	}
	return c.Status(fiber.StatusCreated).JSON(out)
}

func UpdateBranch(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "id inválido")
	}
	if err := requireBranchScope(c, id); err != nil {
		return err
	}
	var body branchRequest
	if err := c.BodyParser(&body); err != nil || body.Nome == "" || body.CNPJ == "" {
		return fiber.NewError(fiber.StatusBadRequest, "nome e cnpj são obrigatórios")
	}
	out, err := repository.UpdateBranch(c.Context(), middleware.Tx(c), middleware.TenantID(c), id, body.Nome, body.CNPJ, body.City, body.State)
	if err != nil {
		return mapDBError(err)
	}
	return c.JSON(out)
}

func DeleteBranch(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "id inválido")
	}
	if err := requireBranchScope(c, id); err != nil {
		return err
	}
	if err := repository.DeleteBranch(c.Context(), middleware.Tx(c), middleware.TenantID(c), id); err != nil {
		return mapDBError(err)
	}
	return c.JSON(fiber.Map{"status": "ok"})
}
