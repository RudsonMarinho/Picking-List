package handlers

import (
	"github.com/Fortcargo/invtech/internal/middleware"
	"github.com/Fortcargo/invtech/internal/repository"
	"github.com/gofiber/fiber/v2"
)

type permissionRequest struct {
	Module string `json:"module"`
	Action string `json:"action"`
}

type roleRequest struct {
	Name        string              `json:"name"`
	Permissions []permissionRequest `json:"permissions"`
}

func ListRoles(c *fiber.Ctx) error {
	list, err := repository.ListRoles(c.Context(), middleware.Tx(c), middleware.TenantID(c))
	if err != nil {
		return mapDBError(err)
	}
	return c.JSON(list)
}

func CreateRole(c *fiber.Ctx) error {
	var body roleRequest
	if err := c.BodyParser(&body); err != nil || body.Name == "" {
		return fiber.NewError(fiber.StatusBadRequest, "name é obrigatório")
	}

	tx := middleware.Tx(c)
	tenantID := middleware.TenantID(c)
	ctx := c.Context()

	out, err := repository.CreateRole(ctx, tx, tenantID, body.Name, nil)
	if err != nil {
		return mapDBError(err)
	}

	perms := make([]repository.Permission, 0, len(body.Permissions))
	for _, p := range body.Permissions {
		perms = append(perms, repository.Permission{Module: p.Module, Action: p.Action})
	}
	if err := repository.AddRolePermissions(ctx, tx, tenantID, out.ID, perms); err != nil {
		return mapDBError(err)
	}

	return c.Status(fiber.StatusCreated).JSON(out)
}
