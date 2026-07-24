package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrNotFound é retornado pelos repositórios quando pgx.ErrNoRows ocorre —
// os chamadores não devem depender de pgx diretamente.
var ErrNotFound = errors.New("registro não encontrado")

// ---------------- auth_lookup / users ----------------

type AuthLookup struct {
	TenantID uuid.UUID
	UserID   uuid.UUID
}

// FindAuthLookupByEmail resolve o tenant de um e-mail ANTES de qualquer
// SET LOCAL — auth_lookup não tem RLS de propósito (R8: fonte de verdade
// da unicidade global de e-mail).
func FindAuthLookupByEmail(ctx context.Context, pool *pgxpool.Pool, email string) (AuthLookup, error) {
	var out AuthLookup
	err := pool.QueryRow(ctx,
		`SELECT tenant_id, user_id FROM auth_lookup WHERE email = $1`, email,
	).Scan(&out.TenantID, &out.UserID)
	if errors.Is(err, pgx.ErrNoRows) {
		return AuthLookup{}, ErrNotFound
	}
	return out, err
}

type User struct {
	TenantID     uuid.UUID `json:"tenant_id"`
	ID           uuid.UUID `json:"id"`
	RoleID       uuid.UUID `json:"role_id"`
	Email        string    `json:"email"`
	PasswordHash *string   `json:"-"` // nunca serializar hash de senha em resposta de API
	AuthProvider string    `json:"auth_provider"`
	// nome do campo Go mantido (usado internamente), json em snake_case
	// pra casar com o resto da API — sem isso, o frontend recebia
	// "MustChangePassword"/"Status" em vez de "must_change_password"/
	// "status" e a tela de usuários renderizava tudo como undefined.
	MustChangePassword bool   `json:"must_change_password"`
	Status             string `json:"status"`
}

func FindUserByID(ctx context.Context, q Queryer, tenantID, userID uuid.UUID) (User, error) {
	var u User
	err := q.QueryRow(ctx,
		`SELECT tenant_id, id, role_id, email, password_hash, auth_provider, must_change_password, status
		 FROM users WHERE tenant_id = $1 AND id = $2`,
		tenantID, userID,
	).Scan(&u.TenantID, &u.ID, &u.RoleID, &u.Email, &u.PasswordHash, &u.AuthProvider, &u.MustChangePassword, &u.Status)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, ErrNotFound
	}
	return u, err
}

// CreateUserWithLookup insere em users e auth_lookup NA MESMA transação
// (R6) — tx já deve estar com SET LOCAL app.tenant_id aplicado.
func CreateUserWithLookup(ctx context.Context, tx pgx.Tx, tenantID, roleID uuid.UUID, email string, passwordHash *string, authProvider string, mustChangePassword bool) (uuid.UUID, error) {
	var userID uuid.UUID
	err := tx.QueryRow(ctx,
		`INSERT INTO users (tenant_id, role_id, email, password_hash, auth_provider, must_change_password)
		 VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`,
		tenantID, roleID, email, passwordHash, authProvider, mustChangePassword,
	).Scan(&userID)
	if err != nil {
		return uuid.Nil, err
	}

	if _, err := tx.Exec(ctx,
		`INSERT INTO auth_lookup (email, tenant_id, user_id) VALUES ($1, $2, $3)`,
		email, tenantID, userID); err != nil {
		return uuid.Nil, err
	}
	return userID, nil
}

func SetUserPassword(ctx context.Context, tx pgx.Tx, tenantID, userID uuid.UUID, passwordHash string) error {
	_, err := tx.Exec(ctx,
		`UPDATE users SET password_hash = $1, must_change_password = false WHERE tenant_id = $2 AND id = $3`,
		passwordHash, tenantID, userID)
	return err
}

func TouchUserLastLogin(ctx context.Context, tx pgx.Tx, tenantID, userID uuid.UUID) error {
	_, err := tx.Exec(ctx,
		`UPDATE users SET last_login_at = now() WHERE tenant_id = $1 AND id = $2`,
		tenantID, userID)
	return err
}

// ---------------- session_lookup / sessions ----------------

type SessionLookup struct {
	TenantID  uuid.UUID
	UserID    uuid.UUID
	ExpiresAt time.Time
}

// FindSessionLookupByHash resolve o tenant de um token de sessão ANTES de
// qualquer SET LOCAL — session_lookup não tem RLS de propósito.
func FindSessionLookupByHash(ctx context.Context, pool *pgxpool.Pool, tokenHash string) (SessionLookup, error) {
	var out SessionLookup
	err := pool.QueryRow(ctx,
		`SELECT tenant_id, user_id, expires_at FROM session_lookup WHERE token_hash = $1`, tokenHash,
	).Scan(&out.TenantID, &out.UserID, &out.ExpiresAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return SessionLookup{}, ErrNotFound
	}
	return out, err
}

// CreateSessionWithLookup insere em sessions e session_lookup na mesma
// transação (R6) — tx já deve estar com SET LOCAL app.tenant_id aplicado.
func CreateSessionWithLookup(ctx context.Context, tx pgx.Tx, tenantID, userID uuid.UUID, tokenHash string, expiresAt time.Time, ip, userAgent string) (uuid.UUID, error) {
	var sessionID uuid.UUID
	err := tx.QueryRow(ctx,
		`INSERT INTO sessions (tenant_id, user_id, token_hash, expires_at, ip_address, user_agent)
		 VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`,
		tenantID, userID, tokenHash, expiresAt, ip, userAgent,
	).Scan(&sessionID)
	if err != nil {
		return uuid.Nil, err
	}

	if _, err := tx.Exec(ctx,
		`INSERT INTO session_lookup (token_hash, tenant_id, user_id, expires_at) VALUES ($1, $2, $3, $4)`,
		tokenHash, tenantID, userID, expiresAt); err != nil {
		return uuid.Nil, err
	}
	return sessionID, nil
}

// IsSessionValid confere, já sob RLS (tx com SET LOCAL aplicado), que a
// sessão existe e não foi revogada — segunda checagem além do
// session_lookup, que é só roteamento pré-tenant.
func IsSessionValid(ctx context.Context, tx pgx.Tx, tenantID, userID uuid.UUID, tokenHash string) (bool, error) {
	var revoked bool
	err := tx.QueryRow(ctx,
		`SELECT revoked FROM sessions WHERE tenant_id = $1 AND user_id = $2 AND token_hash = $3`,
		tenantID, userID, tokenHash,
	).Scan(&revoked)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return !revoked, nil
}

// RevokeSession marca a sessão como revogada e remove de session_lookup na
// mesma transação — logout efetivo mesmo se outra conexão ainda tiver o
// token_hash em memória.
func RevokeSession(ctx context.Context, tx pgx.Tx, tenantID, userID uuid.UUID, tokenHash string) error {
	if _, err := tx.Exec(ctx,
		`UPDATE sessions SET revoked = true WHERE tenant_id = $1 AND user_id = $2 AND token_hash = $3`,
		tenantID, userID, tokenHash); err != nil {
		return err
	}
	_, err := tx.Exec(ctx, `DELETE FROM session_lookup WHERE token_hash = $1`, tokenHash)
	return err
}
