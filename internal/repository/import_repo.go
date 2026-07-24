package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type ImportBatch struct {
	TenantID         uuid.UUID `json:"tenant_id"`
	ID               uuid.UUID `json:"id"`
	IdempotencyKey   string    `json:"idempotency_key"`
	Status           string    `json:"status"`
	RecordsProcessed int       `json:"records_processed"`
}

// FindImportBatch é a base da idempotência: reenviar o mesmo POST /import
// com a mesma idempotency_key não duplica dados — o handler devolve o
// resultado já processado em vez de reimportar.
func FindImportBatch(ctx context.Context, q Queryer, tenantID uuid.UUID, idempotencyKey string) (ImportBatch, error) {
	var b ImportBatch
	err := q.QueryRow(ctx,
		`SELECT tenant_id, id, idempotency_key, status, records_processed
		 FROM import_batches WHERE tenant_id = $1 AND idempotency_key = $2`,
		tenantID, idempotencyKey,
	).Scan(&b.TenantID, &b.ID, &b.IdempotencyKey, &b.Status, &b.RecordsProcessed)
	if errors.Is(err, pgx.ErrNoRows) {
		return ImportBatch{}, ErrNotFound
	}
	return b, err
}

func CreateImportBatch(ctx context.Context, q Queryer, tenantID uuid.UUID, idempotencyKey string) (ImportBatch, error) {
	b := ImportBatch{TenantID: tenantID, IdempotencyKey: idempotencyKey, Status: "processing"}
	err := q.QueryRow(ctx,
		`INSERT INTO import_batches (tenant_id, idempotency_key) VALUES ($1, $2) RETURNING id, status, records_processed`,
		tenantID, idempotencyKey,
	).Scan(&b.ID, &b.Status, &b.RecordsProcessed)
	return b, err
}

func CompleteImportBatch(ctx context.Context, q Queryer, tenantID, id uuid.UUID, recordsProcessed int) error {
	_, err := q.Exec(ctx,
		`UPDATE import_batches SET status = 'completed', records_processed = $1 WHERE tenant_id = $2 AND id = $3`,
		recordsProcessed, tenantID, id)
	return err
}
