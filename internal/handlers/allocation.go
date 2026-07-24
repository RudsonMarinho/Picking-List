package handlers

import (
	"html"

	"github.com/Fortcargo/invtech/internal/middleware"
	"github.com/Fortcargo/invtech/internal/repository"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type allocationRequest struct {
	AssetID        string `json:"asset_id"`
	CollaboratorID string `json:"collaborator_id"`
}

func ListAllocations(c *fiber.Ctx) error {
	list, err := repository.ListAllocations(c.Context(), middleware.Tx(c), middleware.TenantID(c))
	if err != nil {
		return mapDBError(err)
	}
	return c.JSON(list)
}

func CreateAllocation(c *fiber.Ctx) error {
	var body allocationRequest
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "corpo inválido")
	}
	assetID, err1 := uuid.Parse(body.AssetID)
	collaboratorID, err2 := uuid.Parse(body.CollaboratorID)
	if err1 != nil || err2 != nil {
		return fiber.NewError(fiber.StatusBadRequest, "asset_id e collaborator_id são obrigatórios")
	}

	tx := middleware.Tx(c)
	tenantID := middleware.TenantID(c)
	ctx := c.Context()

	assetNome, err := repository.GetAssetNome(ctx, tx, tenantID, assetID)
	if err != nil {
		return mapDBError(err)
	}
	collaboratorNome, err := repository.GetCollaboratorNome(ctx, tx, tenantID, collaboratorID)
	if err != nil {
		return mapDBError(err)
	}
	assetBranchID, err := repository.GetAssetBranchID(ctx, tx, tenantID, assetID)
	if err != nil {
		return mapDBError(err)
	}
	if err := requireBranchScope(c, assetBranchID); err != nil {
		return err
	}

	out, err := repository.CreateAllocation(ctx, tx, tenantID, assetID, collaboratorID, assetNome, collaboratorNome)
	if err != nil {
		return mapDBError(err)
	}

	// html.EscapeString — nome de ativo/colaborador é texto livre do usuário;
	// sem escape aqui, um nome como "</pre><script>..." vira XSS armazenado
	// pra qualquer consumidor que renderize termo_html via innerHTML.
	termoHTML := ""
	if out.TermoResponsabilidade != nil {
		termoHTML = "<pre>" + html.EscapeString(*out.TermoResponsabilidade) + "</pre>"
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"allocation": out,
		"termo_html": termoHTML,
	})
}

func ReturnAllocation(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "id inválido")
	}
	tx := middleware.Tx(c)
	tenantID := middleware.TenantID(c)
	ctx := c.Context()

	assetID, err := repository.GetAllocationAssetID(ctx, tx, tenantID, id)
	if err != nil {
		return mapDBError(err)
	}
	assetBranchID, err := repository.GetAssetBranchID(ctx, tx, tenantID, assetID)
	if err != nil {
		return mapDBError(err)
	}
	if err := requireBranchScope(c, assetBranchID); err != nil {
		return err
	}
	if err := repository.ReturnAllocation(ctx, tx, tenantID, id); err != nil {
		return mapDBError(err)
	}
	return c.JSON(fiber.Map{"status": "ok"})
}
