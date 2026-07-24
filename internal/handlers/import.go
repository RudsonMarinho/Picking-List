package handlers

import (
	"errors"

	"github.com/Fortcargo/invtech/internal/middleware"
	"github.com/Fortcargo/invtech/internal/repository"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type importAssetRow struct {
	AssetTypeID  string  `json:"asset_type_id"`
	BranchID     string  `json:"branch_id"`
	SectorID     string  `json:"sector_id"`
	Nome         string  `json:"nome"`
	SerialNumber *string `json:"serial_number,omitempty"`
}

type importRequest struct {
	IdempotencyKey string           `json:"idempotency_key"`
	Data           []importAssetRow `json:"data"`
}

// Import é idempotente por idempotency_key: reenviar o mesmo POST com a
// mesma chave devolve o resultado já processado, sem duplicar ativos.
func Import(c *fiber.Ctx) error {
	var body importRequest
	if err := c.BodyParser(&body); err != nil || body.IdempotencyKey == "" {
		return fiber.NewError(fiber.StatusBadRequest, "idempotency_key é obrigatório")
	}

	tx := middleware.Tx(c)
	tenantID := middleware.TenantID(c)
	ctx := c.Context()

	existing, err := repository.FindImportBatch(ctx, tx, tenantID, body.IdempotencyKey)
	if err == nil {
		return c.JSON(fiber.Map{"status": existing.Status, "count": existing.RecordsProcessed})
	}
	if !errors.Is(err, repository.ErrNotFound) {
		return mapDBError(err)
	}

	batch, err := repository.CreateImportBatch(ctx, tx, tenantID, body.IdempotencyKey)
	if err != nil {
		return mapDBError(err)
	}

	count := 0
	for _, row := range body.Data {
		assetTypeID, e1 := uuid.Parse(row.AssetTypeID)
		branchID, e2 := uuid.Parse(row.BranchID)
		sectorID, e3 := uuid.Parse(row.SectorID)
		if e1 != nil || e2 != nil || e3 != nil || row.Nome == "" {
			return fiber.NewError(fiber.StatusBadRequest, "registro de import inválido")
		}
		if _, err := repository.CreateAsset(ctx, tx, tenantID, assetTypeID, branchID, sectorID, row.Nome, row.SerialNumber, repository.AssetDetails{}); err != nil {
			return mapDBError(err)
		}
		count++
	}

	if err := repository.CompleteImportBatch(ctx, tx, tenantID, batch.ID, count); err != nil {
		return mapDBError(err)
	}

	return c.JSON(fiber.Map{"status": "completed", "count": count})
}
