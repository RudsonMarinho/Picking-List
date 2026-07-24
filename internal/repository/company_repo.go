package repository

import (
	"context"

	"github.com/google/uuid"
)

type Company struct {
	TenantID uuid.UUID `json:"tenant_id"`
	ID       uuid.UUID `json:"id"`
	Nome     string    `json:"nome"`
	CNPJ     string    `json:"cnpj"`
	City     *string   `json:"city,omitempty"`
	State    *string   `json:"state,omitempty"`
}

func ListCompanies(ctx context.Context, q Queryer, tenantID uuid.UUID) ([]Company, error) {
	rows, err := q.Query(ctx,
		`SELECT tenant_id, id, nome, cnpj, city, state FROM company WHERE tenant_id = $1 ORDER BY nome`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []Company{}
	for rows.Next() {
		var c Company
		if err := rows.Scan(&c.TenantID, &c.ID, &c.Nome, &c.CNPJ, &c.City, &c.State); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func CreateCompany(ctx context.Context, q Queryer, tenantID uuid.UUID, nome, cnpj string, city, state *string) (Company, error) {
	c := Company{TenantID: tenantID, Nome: nome, CNPJ: cnpj, City: city, State: state}
	err := q.QueryRow(ctx,
		`INSERT INTO company (tenant_id, nome, cnpj, city, state) VALUES ($1, $2, $3, $4, $5) RETURNING id`,
		tenantID, nome, cnpj, city, state,
	).Scan(&c.ID)
	return c, err
}

func UpdateCompany(ctx context.Context, q Queryer, tenantID, id uuid.UUID, nome, cnpj string, city, state *string) (Company, error) {
	tag, err := q.Exec(ctx,
		`UPDATE company SET nome = $1, cnpj = $2, city = $3, state = $4 WHERE tenant_id = $5 AND id = $6`,
		nome, cnpj, city, state, tenantID, id)
	if err != nil {
		return Company{}, err
	}
	if tag.RowsAffected() == 0 {
		return Company{}, ErrNotFound
	}
	return Company{TenantID: tenantID, ID: id, Nome: nome, CNPJ: cnpj, City: city, State: state}, nil
}

func DeleteCompany(ctx context.Context, q Queryer, tenantID, id uuid.UUID) error {
	tag, err := q.Exec(ctx, `DELETE FROM company WHERE tenant_id = $1 AND id = $2`, tenantID, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
