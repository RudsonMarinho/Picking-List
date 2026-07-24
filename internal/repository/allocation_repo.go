package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type Allocation struct {
	TenantID              uuid.UUID  `json:"tenant_id"`
	ID                    uuid.UUID  `json:"id"`
	AssetID               uuid.UUID  `json:"asset_id"`
	CollaboratorID        uuid.UUID  `json:"collaborator_id"`
	AllocatedAt           time.Time  `json:"allocated_at"`
	ReturnedAt            *time.Time `json:"returned_at,omitempty"`
	TermoResponsabilidade *string    `json:"termo_responsabilidade,omitempty"`
}

// ErrAssetAlreadyAllocated cobre tanto a violação de ux_allocation_ativa
// quanto a checagem prévia feita aqui (mensagem melhor que 23505 cru).
var ErrAssetAlreadyAllocated = errors.New("ativo já possui alocação ativa")

func ListAllocations(ctx context.Context, q Queryer, tenantID uuid.UUID) ([]Allocation, error) {
	rows, err := q.Query(ctx,
		`SELECT tenant_id, id, asset_id, collaborator_id, allocated_at, returned_at, termo_responsabilidade
		 FROM allocation WHERE tenant_id = $1 ORDER BY allocated_at DESC`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []Allocation{}
	for rows.Next() {
		var a Allocation
		if err := rows.Scan(&a.TenantID, &a.ID, &a.AssetID, &a.CollaboratorID, &a.AllocatedAt, &a.ReturnedAt, &a.TermoResponsabilidade); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// CreateAllocation gera o Termo de Responsabilidade (texto simples — Fase 1
// não tem gerador de PDF) e marca o asset como 'allocated' na mesma
// transação. Depende de ux_allocation_ativa (índice único parcial) para a
// garantia final de "1 alocação ativa por ativo" — condição de corrida
// fechada no banco, não só na aplicação.
func CreateAllocation(ctx context.Context, q Queryer, tenantID, assetID, collaboratorID uuid.UUID, assetNome, collaboratorNome string) (Allocation, error) {
	termo := fmt.Sprintf(
		"Termo de Responsabilidade\n\nAtivo: %s\nColaborador: %s\nData: %s\n\nDeclaro ter recebido o ativo acima descrito, responsabilizando-me por sua guarda e conservação até a devolução.",
		assetNome, collaboratorNome, time.Now().UTC().Format("2006-01-02 15:04:05"),
	)

	a := Allocation{TenantID: tenantID, AssetID: assetID, CollaboratorID: collaboratorID, TermoResponsabilidade: &termo}
	err := q.QueryRow(ctx,
		`INSERT INTO allocation (tenant_id, asset_id, collaborator_id, termo_responsabilidade)
		 VALUES ($1, $2, $3, $4) RETURNING id, allocated_at`,
		tenantID, assetID, collaboratorID, termo,
	).Scan(&a.ID, &a.AllocatedAt)
	if err != nil {
		return Allocation{}, err
	}

	if err := SetAssetStatus(ctx, q, tenantID, assetID, "allocated"); err != nil {
		return Allocation{}, err
	}
	return a, nil
}

// GetAllocationAssetID resolve o asset_id de uma alocação — usado pra
// autorização de escopo (middleware.Scope) antes de ReturnAllocation, já que
// a filial relevante é a do asset, não a da alocação em si.
func GetAllocationAssetID(ctx context.Context, q Queryer, tenantID, id uuid.UUID) (uuid.UUID, error) {
	var assetID uuid.UUID
	err := q.QueryRow(ctx, `SELECT asset_id FROM allocation WHERE tenant_id = $1 AND id = $2`, tenantID, id).Scan(&assetID)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, ErrNotFound
	}
	return assetID, err
}

// ReturnAllocation marca returned_at = now() e devolve o asset para
// 'available' — mesma transação.
func ReturnAllocation(ctx context.Context, q Queryer, tenantID, allocationID uuid.UUID) error {
	var assetID uuid.UUID
	err := q.QueryRow(ctx,
		`UPDATE allocation SET returned_at = now()
		 WHERE tenant_id = $1 AND id = $2 AND returned_at IS NULL
		 RETURNING asset_id`,
		tenantID, allocationID,
	).Scan(&assetID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	return SetAssetStatus(ctx, q, tenantID, assetID, "available")
}
