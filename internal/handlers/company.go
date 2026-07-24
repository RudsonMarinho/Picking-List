package handlers

import (
	"github.com/Fortcargo/invtech/internal/middleware"
	"github.com/Fortcargo/invtech/internal/repository"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type companyRequest struct {
	Nome  string  `json:"nome"`
	CNPJ  string  `json:"cnpj"`
	City  *string `json:"city,omitempty"`
	State *string `json:"state,omitempty"`
}

func ListCompanies(c *fiber.Ctx) error {
	list, err := repository.ListCompanies(c.Context(), middleware.Tx(c), middleware.TenantID(c))
	if err != nil {
		return mapDBError(err)
	}
	return c.JSON(list)
}

func CreateCompany(c *fiber.Ctx) error {
	var body companyRequest
	if err := c.BodyParser(&body); err != nil || body.Nome == "" || body.CNPJ == "" {
		return fiber.NewError(fiber.StatusBadRequest, "nome e cnpj são obrigatórios")
	}
	if err := requireUnrestrictedScope(c); err != nil {
		return err
	}
	out, err := repository.CreateCompany(c.Context(), middleware.Tx(c), middleware.TenantID(c), body.Nome, body.CNPJ, body.City, body.State)
	if err != nil {
		return mapDBError(err)
	}
	return c.Status(fiber.StatusCreated).JSON(out)
}

func UpdateCompany(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "id inválido")
	}
	if err := requireCompanyScope(c, id); err != nil {
		return err
	}
	var body companyRequest
	if err := c.BodyParser(&body); err != nil || body.Nome == "" || body.CNPJ == "" {
		return fiber.NewError(fiber.StatusBadRequest, "nome e cnpj são obrigatórios")
	}
	out, err := repository.UpdateCompany(c.Context(), middleware.Tx(c), middleware.TenantID(c), id, body.Nome, body.CNPJ, body.City, body.State)
	if err != nil {
		return mapDBError(err)
	}
	return c.JSON(out)
}

func DeleteCompany(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "id inválido")
	}
	if err := requireCompanyScope(c, id); err != nil {
		return err
	}
	if err := repository.DeleteCompany(c.Context(), middleware.Tx(c), middleware.TenantID(c), id); err != nil {
		return mapDBError(err)
	}
	return c.JSON(fiber.Map{"status": "ok"})
}
