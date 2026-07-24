package handlers

import (
	"github.com/Fortcargo/invtech/internal/middleware"
	"github.com/Fortcargo/invtech/internal/repository"
	"github.com/Fortcargo/invtech/pkg/security"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type userScopeRequest struct {
	ScopeType string  `json:"scope_type"`
	ScopeID   *string `json:"scope_id,omitempty"`
}

type userRequest struct {
	Email           string             `json:"email"`
	RoleID          string             `json:"role_id"`
	SenhaTemporaria string             `json:"senha_temporaria"`
	Scopes          []userScopeRequest `json:"scopes"`
}

func ListUsers(c *fiber.Ctx) error {
	list, err := repository.ListUsers(c.Context(), middleware.Tx(c), middleware.TenantID(c))
	if err != nil {
		return mapDBError(err)
	}
	return c.JSON(list)
}

// CreateUser cria o usuário com senha temporária (must_change_password=true
// por padrão na migration) e grava os escopos de acesso — sem scopes no
// corpo, assume scope_type='all' (usuário sem restrição de empresa/filial).
func CreateUser(c *fiber.Ctx) error {
	var body userRequest
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "corpo inválido")
	}
	roleID, err := uuid.Parse(body.RoleID)
	if err != nil || body.Email == "" || body.SenhaTemporaria == "" {
		return fiber.NewError(fiber.StatusBadRequest, "email, role_id e senha_temporaria são obrigatórios")
	}
	if len(body.SenhaTemporaria) < 8 {
		return fiber.NewError(fiber.StatusBadRequest, "senha_temporaria muito curta")
	}

	hash, err := security.HashPassword(body.SenhaTemporaria)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "falha ao gerar senha")
	}

	tx := middleware.Tx(c)
	tenantID := middleware.TenantID(c)
	ctx := c.Context()

	userID, err := repository.CreateUserWithLookup(ctx, tx, tenantID, roleID, body.Email, &hash, "local", true)
	if err != nil {
		return mapDBError(err)
	}

	scopes := make([]repository.UserScope, 0, len(body.Scopes))
	for _, s := range body.Scopes {
		var scopeID *uuid.UUID
		if s.ScopeID != nil {
			id, err := uuid.Parse(*s.ScopeID)
			if err != nil {
				return fiber.NewError(fiber.StatusBadRequest, "scope_id inválido")
			}
			scopeID = &id
		}
		scopes = append(scopes, repository.UserScope{ScopeType: s.ScopeType, ScopeID: scopeID})
	}
	if len(scopes) == 0 {
		scopes = append(scopes, repository.UserScope{ScopeType: "all"})
	}
	if err := repository.CreateUserScopes(ctx, tx, tenantID, userID, scopes); err != nil {
		return mapDBError(err)
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"id": userID, "email": body.Email})
}

type userUpdateRequest struct {
	RoleID string `json:"role_id"`
	Status string `json:"status"`
}

func UpdateUser(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "id inválido")
	}
	var body userUpdateRequest
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "corpo inválido")
	}
	roleID, err := uuid.Parse(body.RoleID)
	if err != nil || body.Status == "" {
		return fiber.NewError(fiber.StatusBadRequest, "role_id e status são obrigatórios")
	}
	out, err := repository.UpdateUserRoleStatus(c.Context(), middleware.Tx(c), middleware.TenantID(c), id, roleID, body.Status)
	if err != nil {
		return mapDBError(err)
	}
	return c.JSON(out)
}

func DeleteUser(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "id inválido")
	}
	if err := repository.DeleteUser(c.Context(), middleware.Tx(c), middleware.TenantID(c), id); err != nil {
		return mapDBError(err)
	}
	return c.JSON(fiber.Map{"status": "ok"})
}
