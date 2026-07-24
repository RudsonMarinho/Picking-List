package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type Sector struct {
	TenantID    uuid.UUID `json:"tenant_id"`
	ID          uuid.UUID `json:"id"`
	BranchID    uuid.UUID `json:"branch_id"`
	Nome        string    `json:"nome"`
	Description *string   `json:"description,omitempty"`
}

func ListSectorsByBranch(ctx context.Context, q Queryer, tenantID, branchID uuid.UUID) ([]Sector, error) {
	rows, err := q.Query(ctx,
		`SELECT tenant_id, id, branch_id, nome, description FROM sector WHERE tenant_id = $1 AND branch_id = $2 ORDER BY nome`,
		tenantID, branchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []Sector{}
	for rows.Next() {
		var s Sector
		if err := rows.Scan(&s.TenantID, &s.ID, &s.BranchID, &s.Nome, &s.Description); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// GetSectorBranchID resolve o branch_id ANTES de mutar — usado pra
// autorização de escopo (middleware.Scope) em update/delete, já que
// branch_id não é editável (não muda entre "antes" e "depois").
func GetSectorBranchID(ctx context.Context, q Queryer, tenantID, id uuid.UUID) (uuid.UUID, error) {
	var branchID uuid.UUID
	err := q.QueryRow(ctx, `SELECT branch_id FROM sector WHERE tenant_id = $1 AND id = $2`, tenantID, id).Scan(&branchID)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, ErrNotFound
	}
	return branchID, err
}

func CreateSector(ctx context.Context, q Queryer, tenantID, branchID uuid.UUID, nome string, description *string) (Sector, error) {
	s := Sector{TenantID: tenantID, BranchID: branchID, Nome: nome, Description: description}
	err := q.QueryRow(ctx,
		`INSERT INTO sector (tenant_id, branch_id, nome, description) VALUES ($1, $2, $3, $4) RETURNING id`,
		tenantID, branchID, nome, description,
	).Scan(&s.ID)
	return s, err
}

func UpdateSector(ctx context.Context, q Queryer, tenantID, id uuid.UUID, nome string, description *string) (Sector, error) {
	var branchID uuid.UUID
	err := q.QueryRow(ctx,
		`UPDATE sector SET nome = $1, description = $2 WHERE tenant_id = $3 AND id = $4
		 RETURNING branch_id`,
		nome, description, tenantID, id).Scan(&branchID)
	if errors.Is(err, pgx.ErrNoRows) {
		return Sector{}, ErrNotFound
	}
	if err != nil {
		return Sector{}, err
	}
	return Sector{TenantID: tenantID, ID: id, BranchID: branchID, Nome: nome, Description: description}, nil
}

func DeleteSector(ctx context.Context, q Queryer, tenantID, id uuid.UUID) error {
	tag, err := q.Exec(ctx, `DELETE FROM sector WHERE tenant_id = $1 AND id = $2`, tenantID, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
