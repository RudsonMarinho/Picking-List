package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type Asset struct {
	TenantID     uuid.UUID `json:"tenant_id"`
	ID           uuid.UUID `json:"id"`
	AssetTypeID  uuid.UUID `json:"asset_type_id"`
	BranchID     uuid.UUID `json:"branch_id"`
	SectorID     uuid.UUID `json:"sector_id"`
	Nome         string    `json:"nome"`
	AssetTag     *string   `json:"asset_tag,omitempty"`
	Hostname     *string   `json:"hostname,omitempty"`
	Brand        *string   `json:"brand,omitempty"`
	Model        *string   `json:"model,omitempty"`
	CPU          *string   `json:"cpu,omitempty"`
	RAM          *string   `json:"ram,omitempty"`
	Storage      *string   `json:"storage,omitempty"`
	Notes        *string   `json:"notes,omitempty"`
	SerialNumber *string   `json:"serial_number,omitempty"`
	Status       string    `json:"status"`
}

// AssetDetails agrupa os campos opcionais herdados da UI original
// (marca/modelo/specs) — mantidos fora da lista de parâmetros obrigatórios
// de Create/Update para não deixar a assinatura ilegível.
type AssetDetails struct {
	AssetTag *string
	Hostname *string
	Brand    *string
	Model    *string
	CPU      *string
	RAM      *string
	Storage  *string
	Notes    *string
	// Status, quando preenchido, permite alternar manualmente para
	// 'maintenance' (ou de volta) via edição — fora desse caso, o status
	// é controlado pelo fluxo de alocação/devolução (SetAssetStatus).
	Status *string
}

func ListAssets(ctx context.Context, q Queryer, tenantID uuid.UUID) ([]Asset, error) {
	rows, err := q.Query(ctx,
		`SELECT tenant_id, id, asset_type_id, branch_id, sector_id, nome, asset_tag, hostname, brand,
		        model, cpu, ram, storage, notes, serial_number, status
		 FROM asset WHERE tenant_id = $1 ORDER BY nome`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []Asset{}
	for rows.Next() {
		var a Asset
		if err := rows.Scan(&a.TenantID, &a.ID, &a.AssetTypeID, &a.BranchID, &a.SectorID, &a.Nome, &a.AssetTag,
			&a.Hostname, &a.Brand, &a.Model, &a.CPU, &a.RAM, &a.Storage, &a.Notes, &a.SerialNumber, &a.Status); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func CreateAsset(ctx context.Context, q Queryer, tenantID, assetTypeID, branchID, sectorID uuid.UUID, nome string, serialNumber *string, details AssetDetails) (Asset, error) {
	a := Asset{
		TenantID: tenantID, AssetTypeID: assetTypeID, BranchID: branchID, SectorID: sectorID,
		Nome: nome, SerialNumber: serialNumber, Status: "available",
		AssetTag: details.AssetTag, Hostname: details.Hostname, Brand: details.Brand, Model: details.Model,
		CPU: details.CPU, RAM: details.RAM, Storage: details.Storage, Notes: details.Notes,
	}
	err := q.QueryRow(ctx,
		`INSERT INTO asset (tenant_id, asset_type_id, branch_id, sector_id, nome, serial_number,
		                    asset_tag, hostname, brand, model, cpu, ram, storage, notes)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14) RETURNING id, status`,
		tenantID, assetTypeID, branchID, sectorID, nome, serialNumber,
		details.AssetTag, details.Hostname, details.Brand, details.Model, details.CPU, details.RAM, details.Storage, details.Notes,
	).Scan(&a.ID, &a.Status)
	return a, err
}

func UpdateAsset(ctx context.Context, q Queryer, tenantID, id, assetTypeID, branchID, sectorID uuid.UUID, nome string, serialNumber *string, details AssetDetails) (Asset, error) {
	var status string
	err := q.QueryRow(ctx,
		`UPDATE asset SET asset_type_id = $1, branch_id = $2, sector_id = $3, nome = $4, serial_number = $5,
		                  asset_tag = $6, hostname = $7, brand = $8, model = $9, cpu = $10, ram = $11, storage = $12, notes = $13,
		                  status = COALESCE($14, status)
		 WHERE tenant_id = $15 AND id = $16
		 RETURNING status`,
		assetTypeID, branchID, sectorID, nome, serialNumber,
		details.AssetTag, details.Hostname, details.Brand, details.Model, details.CPU, details.RAM, details.Storage, details.Notes,
		details.Status,
		tenantID, id).Scan(&status)
	if errors.Is(err, pgx.ErrNoRows) {
		return Asset{}, ErrNotFound
	}
	if err != nil {
		return Asset{}, err
	}
	return Asset{
		TenantID: tenantID, ID: id, AssetTypeID: assetTypeID, BranchID: branchID, SectorID: sectorID,
		Nome: nome, SerialNumber: serialNumber,
		AssetTag: details.AssetTag, Hostname: details.Hostname, Brand: details.Brand, Model: details.Model,
		CPU: details.CPU, RAM: details.RAM, Storage: details.Storage, Notes: details.Notes,
		Status: status,
	}, nil
}

func DeleteAsset(ctx context.Context, q Queryer, tenantID, id uuid.UUID) error {
	tag, err := q.Exec(ctx, `DELETE FROM asset WHERE tenant_id = $1 AND id = $2`, tenantID, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// GetAssetNome busca só o nome — usado ao montar o Termo de
// Responsabilidade na criação da alocação.
func GetAssetNome(ctx context.Context, q Queryer, tenantID, id uuid.UUID) (string, error) {
	var nome string
	err := q.QueryRow(ctx, `SELECT nome FROM asset WHERE tenant_id = $1 AND id = $2`, tenantID, id).Scan(&nome)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	return nome, err
}

// GetAssetBranchID resolve o branch_id atual — usado pra autorização de
// escopo (middleware.Scope) antes do delete e na criação de alocação.
func GetAssetBranchID(ctx context.Context, q Queryer, tenantID, id uuid.UUID) (uuid.UUID, error) {
	var branchID uuid.UUID
	err := q.QueryRow(ctx, `SELECT branch_id FROM asset WHERE tenant_id = $1 AND id = $2`, tenantID, id).Scan(&branchID)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, ErrNotFound
	}
	return branchID, err
}

// SetAssetStatus é usado pelo fluxo de alocação/devolução — mesma
// transação da requisição, nunca uma tx separada.
func SetAssetStatus(ctx context.Context, q Queryer, tenantID, id uuid.UUID, status string) error {
	tag, err := q.Exec(ctx, `UPDATE asset SET status = $1 WHERE tenant_id = $2 AND id = $3`, status, tenantID, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
