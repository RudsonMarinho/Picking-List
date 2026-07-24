package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// WithTenant abre uma transação, emite SET LOCAL app.tenant_id (fail-closed
// por padrão — sem isso, toda política RLS das tabelas de negócio rejeita
// tudo) e executa fn. Commita em sucesso, faz rollback em erro.
//
// SET LOCAL, nunca SET — o pool pgx reusa conexões entre requisições, e SET
// vazaria o tenant_id para a próxima requisição que pegar a mesma conexão.
//
// tenantID é uuid.UUID (não string) de propósito: UUID.String() só produz
// hex+hífens, então a interpolação abaixo é segura mesmo sem bind
// parameter — SET LOCAL não aceita parâmetros ($1) no protocolo estendido.
func WithTenant(ctx context.Context, pool *pgxpool.Pool, tenantID uuid.UUID, fn func(tx pgx.Tx) error) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck // no-op se já commitado

	if _, err := tx.Exec(ctx, fmt.Sprintf("SET LOCAL app.tenant_id = '%s'", tenantID.String())); err != nil {
		return fmt.Errorf("set tenant: %w", err)
	}

	if err := fn(tx); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	return nil
}
