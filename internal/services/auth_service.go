// Package services contém a lógica de negócio que orquestra repositórios —
// handlers ficam finos, chamando aqui.
package services

import (
	"context"
	"errors"
	"time"

	"github.com/Fortcargo/invtech/internal/repository"
	"github.com/Fortcargo/invtech/pkg/security"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrInvalidCredentials cobre e-mail inexistente, senha errada, usuário
// inativo ou provedor incompatível — de propósito indistinguível para
// quem chama, para não vazar se o e-mail existe.
var ErrInvalidCredentials = errors.New("credenciais inválidas")

type AuthService struct {
	Pool       *pgxpool.Pool
	SessionTTL time.Duration
}

// Login resolve o e-mail via auth_lookup, abre a transação do tenant,
// confere a senha (Argon2id) e emite uma sessão nova.
func (s *AuthService) Login(ctx context.Context, email, password, ip, userAgent string) (rawToken string, expiresAt time.Time, user repository.User, err error) {
	lookup, err := repository.FindAuthLookupByEmail(ctx, s.Pool, email)
	if err != nil {
		return "", time.Time{}, repository.User{}, ErrInvalidCredentials
	}

	txErr := repository.WithTenant(ctx, s.Pool, lookup.TenantID, func(tx pgx.Tx) error {
		u, err := repository.FindUserByID(ctx, tx, lookup.TenantID, lookup.UserID)
		if err != nil {
			return ErrInvalidCredentials
		}
		if u.Status != "active" || u.AuthProvider != "local" || u.PasswordHash == nil {
			return ErrInvalidCredentials
		}
		ok, err := security.VerifyPassword(*u.PasswordHash, password)
		if err != nil || !ok {
			return ErrInvalidCredentials
		}

		token, err := security.NewSessionToken()
		if err != nil {
			return err
		}
		tokenHash := security.HashToken(token)
		exp := time.Now().UTC().Add(s.SessionTTL)

		if _, err := repository.CreateSessionWithLookup(ctx, tx, lookup.TenantID, lookup.UserID, tokenHash, exp, ip, userAgent); err != nil {
			return err
		}
		if err := repository.TouchUserLastLogin(ctx, tx, lookup.TenantID, lookup.UserID); err != nil {
			return err
		}

		user = u
		rawToken = token
		expiresAt = exp
		return nil
	})
	if txErr != nil {
		if errors.Is(txErr, ErrInvalidCredentials) {
			return "", time.Time{}, repository.User{}, ErrInvalidCredentials
		}
		return "", time.Time{}, repository.User{}, txErr
	}
	return rawToken, expiresAt, user, nil
}

// ChangePassword roda DENTRO da transação já aberta pelo middleware da
// requisição (RequireSession) — nunca abre outra.
func (s *AuthService) ChangePassword(ctx context.Context, tx pgx.Tx, tenantID, userID uuid.UUID, oldPassword, newPassword string) error {
	u, err := repository.FindUserByID(ctx, tx, tenantID, userID)
	if err != nil {
		return err
	}
	if u.AuthProvider != "local" || u.PasswordHash == nil {
		return ErrInvalidCredentials
	}
	ok, err := security.VerifyPassword(*u.PasswordHash, oldPassword)
	if err != nil || !ok {
		return ErrInvalidCredentials
	}

	newHash, err := security.HashPassword(newPassword)
	if err != nil {
		return err
	}
	return repository.SetUserPassword(ctx, tx, tenantID, userID, newHash)
}
