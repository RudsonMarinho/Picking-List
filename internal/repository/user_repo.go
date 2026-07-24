package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type UserScope struct {
	ScopeType string     `json:"scope_type"`
	ScopeID   *uuid.UUID `json:"scope_id,omitempty"`
}

func ListUsers(ctx context.Context, q Queryer, tenantID uuid.UUID) ([]User, error) {
	rows, err := q.Query(ctx,
		`SELECT tenant_id, id, role_id, email, password_hash, auth_provider, must_change_password, status
		 FROM users WHERE tenant_id = $1 ORDER BY email`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []User{}
	for rows.Next() {
		var u User
		if err := rows.Scan(&u.TenantID, &u.ID, &u.RoleID, &u.Email, &u.PasswordHash, &u.AuthProvider, &u.MustChangePassword, &u.Status); err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

// CreateUserScopes insere os escopos de acesso do usuário na mesma
// transação de criação (chamada logo após CreateUserWithLookup).
func CreateUserScopes(ctx context.Context, q Queryer, tenantID, userID uuid.UUID, scopes []UserScope) error {
	for _, s := range scopes {
		if _, err := q.Exec(ctx,
			`INSERT INTO user_scopes (tenant_id, user_id, scope_type, scope_id) VALUES ($1, $2, $3, $4)`,
			tenantID, userID, s.ScopeType, s.ScopeID); err != nil {
			return err
		}
	}
	return nil
}

func ListUserScopes(ctx context.Context, q Queryer, tenantID, userID uuid.UUID) ([]UserScope, error) {
	rows, err := q.Query(ctx,
		`SELECT scope_type, scope_id FROM user_scopes WHERE tenant_id = $1 AND user_id = $2`, tenantID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []UserScope{}
	for rows.Next() {
		var s UserScope
		if err := rows.Scan(&s.ScopeType, &s.ScopeID); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func UpdateUserRoleStatus(ctx context.Context, q Queryer, tenantID, id, roleID uuid.UUID, status string) (User, error) {
	var email, authProvider string
	var mustChangePassword bool
	err := q.QueryRow(ctx,
		`UPDATE users SET role_id = $1, status = $2 WHERE tenant_id = $3 AND id = $4
		 RETURNING email, auth_provider, must_change_password`,
		roleID, status, tenantID, id).Scan(&email, &authProvider, &mustChangePassword)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrNotFound
	}
	if err != nil {
		return User{}, err
	}
	return User{
		TenantID: tenantID, ID: id, RoleID: roleID, Email: email, AuthProvider: authProvider,
		MustChangePassword: mustChangePassword, Status: status,
	}, nil
}

// DeleteUser remove o usuário e os lookups pré-tenant associados na mesma
// transação — auth_lookup/session_lookup não têm FK pra users (só sessions
// tem ON DELETE CASCADE), então sem isso o e-mail ficava preso pra sempre em
// auth_lookup (recriar a conta batia em "registro duplicado") e
// session_lookup acumulava linhas órfãs (sem risco de acesso — IsSessionValid
// já falha fechado contra a linha real de sessions, que é limpa em cascata).
func DeleteUser(ctx context.Context, q Queryer, tenantID, id uuid.UUID) error {
	tag, err := q.Exec(ctx, `DELETE FROM users WHERE tenant_id = $1 AND id = $2`, tenantID, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	if _, err := q.Exec(ctx, `DELETE FROM auth_lookup WHERE tenant_id = $1 AND user_id = $2`, tenantID, id); err != nil {
		return err
	}
	if _, err := q.Exec(ctx, `DELETE FROM session_lookup WHERE tenant_id = $1 AND user_id = $2`, tenantID, id); err != nil {
		return err
	}
	if _, err := q.Exec(ctx, `DELETE FROM sessions WHERE tenant_id = $1 AND user_id = $2`, tenantID, id); err != nil {
		return err
	}
	return nil
}
