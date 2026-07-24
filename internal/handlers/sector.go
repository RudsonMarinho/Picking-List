package handlers

import (
	"github.com/Fortcargo/invtech/internal/middleware"
	"github.com/Fortcargo/invtech/internal/repository"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type sectorRequest struct {
	Nome        string  `json:"nome"`
	Description *string `json:"description,omitempty"`
}

func ListSectors(c *fiber.Ctx) error {
	branchID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "id de filial inválido")
	}
	list, err := repository.ListSectorsByBranch(c.Context(), middleware.Tx(c), middleware.TenantID(c), branchID)
	if err != nil {
		return mapDBError(err)
	}
	return c.JSON(list)
}

func CreateSector(c *fiber.Ctx) error {
	branchID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "id de filial inválido")
	}
	if err := requireBranchScope(c, branchID); err != nil {
		return err
	}
	var body sectorRequest
	if err := c.BodyParser(&body); err != nil || body.Nome == "" {
		return fiber.NewError(fiber.StatusBadRequest, "nome é obrigatório")
	}
	out, err := repository.CreateSector(c.Context(), middleware.Tx(c), middleware.TenantID(c), branchID, body.Nome, body.Description)
	if err != nil {
		return mapDBError(err)
	}
	return c.Status(fiber.StatusCreated).JSON(out)
}

func UpdateSector(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "id inválido")
	}
	branchID, err := repository.GetSectorBranchID(c.Context(), middleware.Tx(c), middleware.TenantID(c), id)
	if err != nil {
		return mapDBError(err)
	}
	if err := requireBranchScope(c, branchID); err != nil {
		return err
	}
	var body sectorRequest
	if err := c.BodyParser(&body); err != nil || body.Nome == "" {
		return fiber.NewError(fiber.StatusBadRequest, "nome é obrigatório")
	}
	out, err := repository.UpdateSector(c.Context(), middleware.Tx(c), middleware.TenantID(c), id, body.Nome, body.Description)
	if err != nil {
		return mapDBError(err)
	}
	return c.JSON(out)
}

func DeleteSector(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "id inválido")
	}
	branchID, err := repository.GetSectorBranchID(c.Context(), middleware.Tx(c), middleware.TenantID(c), id)
	if err != nil {
		return mapDBError(err)
	}
	if err := requireBranchScope(c, branchID); err != nil {
		return err
	}
	if err := repository.DeleteSector(c.Context(), middleware.Tx(c), middleware.TenantID(c), id); err != nil {
		return mapDBError(err)
	}
	return c.JSON(fiber.Map{"status": "ok"})
}
