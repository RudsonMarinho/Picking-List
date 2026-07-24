package repository

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Queryer é satisfeito tanto por *pgxpool.Pool quanto por pgx.Tx — permite
// que as funções de repositório recebam ou um pool (consultas às tabelas
// pré-tenant, sem RLS) ou uma transação já com SET LOCAL app.tenant_id
// aplicado (tabelas de negócio, sob RLS).
type Queryer interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}
