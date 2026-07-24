package handlers

import (
	"github.com/Fortcargo/invtech/internal/middleware"
	"github.com/Fortcargo/invtech/internal/repository"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type assetTypeRequest struct {
	Nome        string  `json:"nome"`
	Icon        *string `json:"icon,omitempty"`
	Description *string `json:"description,omitempty"`
}

func ListAssetTypes(c *fiber.Ctx) error {
	list, err := repository.ListAssetTypes(c.Context(), middleware.Tx(c), middleware.TenantID(c))
	if err != nil {
		return mapDBError(err)
	}
	return c.JSON(list)
}

func CreateAssetType(c *fiber.Ctx) error {
	var body assetTypeRequest
	if err := c.BodyParser(&body); err != nil || body.Nome == "" {
		return fiber.NewError(fiber.StatusBadRequest, "nome é obrigatório")
	}
	out, err := repository.CreateAssetType(c.Context(), middleware.Tx(c), middleware.TenantID(c), body.Nome, body.Icon, body.Description)
	if err != nil {
		return mapDBError(err)
	}
	return c.Status(fiber.StatusCreated).JSON(out)
}

func UpdateAssetType(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "id inválido")
	}
	var body assetTypeRequest
	if err := c.BodyParser(&body); err != nil || body.Nome == "" {
		return fiber.NewError(fiber.StatusBadRequest, "nome é obrigatório")
	}
	out, err := repository.UpdateAssetType(c.Context(), middleware.Tx(c), middleware.TenantID(c), id, body.Nome, body.Icon, body.Description)
	if err != nil {
		return mapDBError(err)
	}
	return c.JSON(out)
}

func DeleteAssetType(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "id inválido")
	}
	if err := repository.DeleteAssetType(c.Context(), middleware.Tx(c), middleware.TenantID(c), id); err != nil {
		return mapDBError(err)
	}
	return c.JSON(fiber.Map{"status": "ok"})
}
