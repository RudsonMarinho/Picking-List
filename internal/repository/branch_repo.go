package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type Branch struct {
	TenantID  uuid.UUID `json:"tenant_id"`
	ID        uuid.UUID `json:"id"`
	CompanyID uuid.UUID `json:"company_id"`
	Nome      string    `json:"nome"`
	CNPJ      string    `json:"cnpj"`
	City      *string   `json:"city,omitempty"`
	State     *string   `json:"state,omitempty"`
}

func ListBranchesByCompany(ctx context.Context, q Queryer, tenantID, companyID uuid.UUID) ([]Branch, error) {
	rows, err := q.Query(ctx,
		`SELECT tenant_id, id, company_id, nome, cnpj, city, state FROM branch WHERE tenant_id = $1 AND company_id = $2 ORDER BY nome`,
		tenantID, companyID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []Branch{}
	for rows.Next() {
		var b Branch
		if err := rows.Scan(&b.TenantID, &b.ID, &b.CompanyID, &b.Nome, &b.CNPJ, &b.City, &b.State); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

func CreateBranch(ctx context.Context, q Queryer, tenantID, companyID uuid.UUID, nome, cnpj string, city, state *string) (Branch, error) {
	b := Branch{TenantID: tenantID, CompanyID: companyID, Nome: nome, CNPJ: cnpj, City: city, State: state}
	err := q.QueryRow(ctx,
		`INSERT INTO branch (tenant_id, company_id, nome, cnpj, city, state) VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`,
		tenantID, companyID, nome, cnpj, city, state,
	).Scan(&b.ID)
	return b, err
}

func UpdateBranch(ctx context.Context, q Queryer, tenantID, id uuid.UUID, nome, cnpj string, city, state *string) (Branch, error) {
	var companyID uuid.UUID
	err := q.QueryRow(ctx,
		`UPDATE branch SET nome = $1, cnpj = $2, city = $3, state = $4 WHERE tenant_id = $5 AND id = $6
		 RETURNING company_id`,
		nome, cnpj, city, state, tenantID, id).Scan(&companyID)
	if errors.Is(err, pgx.ErrNoRows) {
		return Branch{}, ErrNotFound
	}
	if err != nil {
		return Branch{}, err
	}
	return Branch{TenantID: tenantID, ID: id, CompanyID: companyID, Nome: nome, CNPJ: cnpj, City: city, State: state}, nil
}

func DeleteBranch(ctx context.Context, q Queryer, tenantID, id uuid.UUID) error {
	tag, err := q.Exec(ctx, `DELETE FROM branch WHERE tenant_id = $1 AND id = $2`, tenantID, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
