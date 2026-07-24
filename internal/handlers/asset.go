package handlers

import (
	"github.com/Fortcargo/invtech/internal/middleware"
	"github.com/Fortcargo/invtech/internal/repository"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type assetRequest struct {
	AssetTypeID  string  `json:"asset_type_id"`
	BranchID     string  `json:"branch_id"`
	SectorID     string  `json:"sector_id"`
	Nome         string  `json:"nome"`
	SerialNumber *string `json:"serial_number,omitempty"`
	AssetTag     *string `json:"asset_tag,omitempty"`
	Hostname     *string `json:"hostname,omitempty"`
	Brand        *string `json:"brand,omitempty"`
	Model        *string `json:"model,omitempty"`
	CPU          *string `json:"cpu,omitempty"`
	RAM          *string `json:"ram,omitempty"`
	Storage      *string `json:"storage,omitempty"`
	Notes        *string `json:"notes,omitempty"`
	Status       *string `json:"status,omitempty"`
}

func (r assetRequest) details() repository.AssetDetails {
	return repository.AssetDetails{
		AssetTag: r.AssetTag, Hostname: r.Hostname, Brand: r.Brand, Model: r.Model,
		CPU: r.CPU, RAM: r.RAM, Storage: r.Storage, Notes: r.Notes, Status: r.Status,
	}
}

func ListAssets(c *fiber.Ctx) error {
	list, err := repository.ListAssets(c.Context(), middleware.Tx(c), middleware.TenantID(c))
	if err != nil {
		return mapDBError(err)
	}
	return c.JSON(list)
}

func CreateAsset(c *fiber.Ctx) error {
	var body assetRequest
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "corpo inválido")
	}
	assetTypeID, err1 := uuid.Parse(body.AssetTypeID)
	branchID, err2 := uuid.Parse(body.BranchID)
	sectorID, err3 := uuid.Parse(body.SectorID)
	if err1 != nil || err2 != nil || err3 != nil || body.Nome == "" {
		return fiber.NewError(fiber.StatusBadRequest, "asset_type_id, branch_id, sector_id e nome são obrigatórios")
	}
	if err := requireBranchScope(c, branchID); err != nil {
		return err
	}
	out, err := repository.CreateAsset(c.Context(), middleware.Tx(c), middleware.TenantID(c), assetTypeID, branchID, sectorID, body.Nome, body.SerialNumber, body.details())
	if err != nil {
		return mapDBError(err)
	}
	return c.Status(fiber.StatusCreated).JSON(out)
}

func UpdateAsset(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "id inválido")
	}
	var body assetRequest
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "corpo inválido")
	}
	assetTypeID, err1 := uuid.Parse(body.AssetTypeID)
	branchID, err2 := uuid.Parse(body.BranchID)
	sectorID, err3 := uuid.Parse(body.SectorID)
	if err1 != nil || err2 != nil || err3 != nil || body.Nome == "" {
		return fiber.NewError(fiber.StatusBadRequest, "asset_type_id, branch_id, sector_id e nome são obrigatórios")
	}
	if err := requireBranchScope(c, branchID); err != nil {
		return err
	}
	out, err := repository.UpdateAsset(c.Context(), middleware.Tx(c), middleware.TenantID(c), id, assetTypeID, branchID, sectorID, body.Nome, body.SerialNumber, body.details())
	if err != nil {
		return mapDBError(err)
	}
	return c.JSON(out)
}

func DeleteAsset(c *fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "id inválido")
	}
	branchID, err := repository.GetAssetBranchID(c.Context(), middleware.Tx(c), middleware.TenantID(c), id)
	if err != nil {
		return mapDBError(err)
	}
	if err := requireBranchScope(c, branchID); err != nil {
		return err
	}
	if err := repository.DeleteAsset(c.Context(), middleware.Tx(c), middleware.TenantID(c), id); err != nil {
		return mapDBError(err)
	}
	return c.JSON(fiber.Map{"status": "ok"})
}
