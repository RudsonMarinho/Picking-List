package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type Collaborator struct {
	TenantID uuid.UUID `json:"tenant_id"`
	ID       uuid.UUID `json:"id"`
	BranchID uuid.UUID `json:"branch_id"`
	SectorID uuid.UUID `json:"sector_id"`
	Nome     string    `json:"nome"`
	Document *string   `json:"document,omitempty"`
	Cargo    *string   `json:"cargo,omitempty"`
	Email    *string   `json:"email,omitempty"`
	Status   string    `json:"status"`
}

func ListCollaborators(ctx context.Context, q Queryer, tenantID uuid.UUID) ([]Collaborator, error) {
	rows, err := q.Query(ctx,
		`SELECT tenant_id, id, branch_id, sector_id, nome, document, cargo, email, status
		 FROM collaborator WHERE tenant_id = $1 ORDER BY nome`,
		tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []Collaborator{}
	for rows.Next() {
		var c Collaborator
		if err := rows.Scan(&c.TenantID, &c.ID, &c.BranchID, &c.SectorID, &c.Nome, &c.Document, &c.Cargo, &c.Email, &c.Status); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// CollaboratorDetails agrupa os campos que a UI original coleta e o schema
// mínimo da Fase 1 não tinha (email, cargo) — document (CPF) é opcional
// aqui pois a UI original não o coleta.
type CollaboratorDetails struct {
	Document *string
	Cargo    *string
	Email    *string
	Status   string
}

func CreateCollaborator(ctx context.Context, q Queryer, tenantID, branchID, sectorID uuid.UUID, nome string, details CollaboratorDetails) (Collaborator, error) {
	status := details.Status
	if status == "" {
		status = "active"
	}
	c := Collaborator{TenantID: tenantID, BranchID: branchID, SectorID: sectorID, Nome: nome,
		Document: details.Document, Cargo: details.Cargo, Email: details.Email, Status: status}
	err := q.QueryRow(ctx,
		`INSERT INTO collaborator (tenant_id, branch_id, sector_id, nome, document, cargo, email, status)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8) RETURNING id`,
		tenantID, branchID, sectorID, nome, details.Document, details.Cargo, details.Email, status,
	).Scan(&c.ID)
	return c, err
}

func UpdateCollaborator(ctx context.Context, q Queryer, tenantID, id, branchID, sectorID uuid.UUID, nome string, details CollaboratorDetails) (Collaborator, error) {
	status := details.Status
	if status == "" {
		status = "active"
	}
	tag, err := q.Exec(ctx,
		`UPDATE collaborator SET branch_id = $1, sector_id = $2, nome = $3, document = $4, cargo = $5, email = $6, status = $7
		 WHERE tenant_id = $8 AND id = $9`,
		branchID, sectorID, nome, details.Document, details.Cargo, details.Email, status, tenantID, id)
	if err != nil {
		return Collaborator{}, err
	}
	if tag.RowsAffected() == 0 {
		return Collaborator{}, ErrNotFound
	}
	return Collaborator{TenantID: tenantID, ID: id, BranchID: branchID, SectorID: sectorID, Nome: nome,
		Document: details.Document, Cargo: details.Cargo, Email: details.Email, Status: status}, nil
}

// GetCollaboratorNome busca só o nome — usado ao montar o Termo de
// Responsabilidade na criação da alocação.
func GetCollaboratorNome(ctx context.Context, q Queryer, tenantID, id uuid.UUID) (string, error) {
	var nome string
	err := q.QueryRow(ctx, `SELECT nome FROM collaborator WHERE tenant_id = $1 AND id = $2`, tenantID, id).Scan(&nome)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	return nome, err
}

// GetCollaboratorBranchID resolve o branch_id atual — usado pra autorização
// de escopo (middleware.Scope) antes do delete.
func GetCollaboratorBranchID(ctx context.Context, q Queryer, tenantID, id uuid.UUID) (uuid.UUID, error) {
	var branchID uuid.UUID
	err := q.QueryRow(ctx, `SELECT branch_id FROM collaborator WHERE tenant_id = $1 AND id = $2`, tenantID, id).Scan(&branchID)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, ErrNotFound
	}
	return branchID, err
}

func DeleteCollaborator(ctx context.Context, q Queryer, tenantID, id uuid.UUID) error {
	tag, err := q.Exec(ctx, `DELETE FROM collaborator WHERE tenant_id = $1 AND id = $2`, tenantID, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
