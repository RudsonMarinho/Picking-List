package repository

import (
	"context"

	"github.com/google/uuid"
)

// ResolvedScope é o resultado, já expandido, dos user_scopes de um usuário —
// escopo 'company' é expandido pra incluir todas as filiais daquela empresa;
// escopo 'branch' fica só na filial explícita (não sobe pra empresa: um
// usuário restrito a uma filial não gerencia a empresa dona dela).
type ResolvedScope struct {
	All        bool
	CompanyIDs map[uuid.UUID]bool
	BranchIDs  map[uuid.UUID]bool
}

func (s ResolvedScope) AllowsCompany(id uuid.UUID) bool { return s.All || s.CompanyIDs[id] }
func (s ResolvedScope) AllowsBranch(id uuid.UUID) bool  { return s.All || s.BranchIDs[id] }

// ResolveUserScope carrega e expande os user_scopes do usuário — chamada uma
// vez por requisição (ver middleware.LoadScope) e usada pelos handlers pra
// autorizar create/update/delete nos módulos vinculados a empresa/filial.
func ResolveUserScope(ctx context.Context, q Queryer, tenantID, userID uuid.UUID) (ResolvedScope, error) {
	scopes, err := ListUserScopes(ctx, q, tenantID, userID)
	if err != nil {
		return ResolvedScope{}, err
	}

	out := ResolvedScope{CompanyIDs: map[uuid.UUID]bool{}, BranchIDs: map[uuid.UUID]bool{}}
	var companyIDs []uuid.UUID
	for _, s := range scopes {
		switch s.ScopeType {
		case "all":
			out.All = true
		case "company":
			if s.ScopeID != nil {
				companyIDs = append(companyIDs, *s.ScopeID)
				out.CompanyIDs[*s.ScopeID] = true
			}
		case "branch":
			if s.ScopeID != nil {
				out.BranchIDs[*s.ScopeID] = true
			}
		}
	}
	if out.All || len(companyIDs) == 0 {
		return out, nil
	}

	rows, err := q.Query(ctx,
		`SELECT id FROM branch WHERE tenant_id = $1 AND company_id = ANY($2)`, tenantID, companyIDs)
	if err != nil {
		return ResolvedScope{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var branchID uuid.UUID
		if err := rows.Scan(&branchID); err != nil {
			return ResolvedScope{}, err
		}
		out.BranchIDs[branchID] = true
	}
	return out, rows.Err()
}
