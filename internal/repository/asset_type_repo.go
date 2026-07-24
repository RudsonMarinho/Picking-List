package repository

import (
	"context"

	"github.com/google/uuid"
)

type AssetType struct {
	TenantID    uuid.UUID `json:"tenant_id"`
	ID          uuid.UUID `json:"id"`
	Nome        string    `json:"nome"`
	Icon        *string   `json:"icon,omitempty"`
	Description *string   `json:"description,omitempty"`
}

func ListAssetTypes(ctx context.Context, q Queryer, tenantID uuid.UUID) ([]AssetType, error) {
	rows, err := q.Query(ctx,
		`SELECT tenant_id, id, nome, icon, description FROM asset_type WHERE tenant_id = $1 ORDER BY nome`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []AssetType{}
	for rows.Next() {
		var a AssetType
		if err := rows.Scan(&a.TenantID, &a.ID, &a.Nome, &a.Icon, &a.Description); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func CreateAssetType(ctx context.Context, q Queryer, tenantID uuid.UUID, nome string, icon, description *string) (AssetType, error) {
	a := AssetType{TenantID: tenantID, Nome: nome, Icon: icon, Description: description}
	err := q.QueryRow(ctx,
		`INSERT INTO asset_type (tenant_id, nome, icon, description) VALUES ($1, $2, $3, $4) RETURNING id`,
		tenantID, nome, icon, description,
	).Scan(&a.ID)
	return a, err
}

func UpdateAssetType(ctx context.Context, q Queryer, tenantID, id uuid.UUID, nome string, icon, description *string) (AssetType, error) {
	tag, err := q.Exec(ctx,
		`UPDATE asset_type SET nome = $1, icon = $2, description = $3 WHERE tenant_id = $4 AND id = $5`,
		nome, icon, description, tenantID, id)
	if err != nil {
		return AssetType{}, err
	}
	if tag.RowsAffected() == 0 {
		return AssetType{}, ErrNotFound
	}
	return AssetType{TenantID: tenantID, ID: id, Nome: nome, Icon: icon, Description: description}, nil
}

func DeleteAssetType(ctx context.Context, q Queryer, tenantID, id uuid.UUID) error {
	tag, err := q.Exec(ctx, `DELETE FROM asset_type WHERE tenant_id = $1 AND id = $2`, tenantID, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
