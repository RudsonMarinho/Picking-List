package handlers

import (
	"github.com/Fortcargo/invtech/internal/middleware"
	"github.com/Fortcargo/invtech/internal/repository"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type collaboratorRequest struct {
	BranchID string  `json:"branch_id"`
	SectorID string  `json:"sector_id"`
	Nome     string  `json:"nome"`
	Document *string `json:"document,omitempty"`
	Cargo    *string `json:"cargo,omitempty"`
	Email    *string `json:"email,omitempty"`
	Status   string  `json:"status,omitempty"`
}

func (r collaboratorRequest) details() repository.CollaboratorDetails {
	return repository.CollaboratorDetails{Document: r.Document, Cargo: r.Cargo, Email: r.Email, Status: r.Status}
}

func ListCollaborators(c *fiber.Ctx) error {
	list, err := repository.ListCollaborators(c.Context(), middleware.Tx(c), middleware.TenantID(c))
	if err != nil {
		return mapDBError(err)
	}
	return c.JSON(list)
}

func CreateCollaborator(c *fiber.Ctx) error {
	var body collaboratorRequest
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "corpo inválido")
	}
	branchID, err1 := uuid.Parse(body.BranchID)
	sectorID, err2 := uuid.Parse(body.SectorID)
	if err1 != nil || err2 != nil || body.Nome == "" {
		return fiber.NewError(fiber.StatusBadRequest, "branch_id, sector_id e nome são obrigatórios")
	}
	if err := requireBranchScope(c, branchID); err != nil {
		return err
	}
	out, err := repository.CreateCollaborator(c.Context(), middleware.Tx(c), middleware.TenantID(c), branchID, sectorID, body.Nome, body.details())
	if err != nil {
		return mapDBError(err)
	}
	return c.Status(fiber.StatusCreated).JSON(out)
}

func UpdateCollaborator(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "id inválido")
	}
	var body collaboratorRequest
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "corpo inválido")
	}
	branchID, err1 := uuid.Parse(body.BranchID)
	sectorID, err2 := uuid.Parse(body.SectorID)
	if err1 != nil || err2 != nil || body.Nome == "" {
		return fiber.NewError(fiber.StatusBadRequest, "branch_id, sector_id e nome são obrigatórios")
	}
	if err := requireBranchScope(c, branchID); err != nil {
		return err
	}
	out, err := repository.UpdateCollaborator(c.Context(), middleware.Tx(c), middleware.TenantID(c), id, branchID, sectorID, body.Nome, body.details())
	if err != nil {
		return mapDBError(err)
	}
	return c.JSON(out)
}

func DeleteCollaborator(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "id inválido")
	}
	branchID, err := repository.GetCollaboratorBranchID(c.Context(), middleware.Tx(c), middleware.TenantID(c), id)
	if err != nil {
		return mapDBError(err)
	}
	if err := requireBranchScope(c, branchID); err != nil {
		return err
	}
	if err := repository.DeleteCollaborator(c.Context(), middleware.Tx(c), middleware.TenantID(c), id); err != nil {
		return mapDBError(err)
	}
	return c.JSON(fiber.Map{"status": "ok"})
}
