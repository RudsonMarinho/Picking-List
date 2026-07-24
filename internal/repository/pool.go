// Package repository concentra o acesso a dados: pool pgx, resolução de
// tenant via SET LOCAL e os repositórios por módulo.
package repository

import (
	"context"
	"fmt"
	"strconv"

	"github.com/jackc/pgx/v5/pgxpool"
)

// DBParams são os parâmetros de conexão individuais — nunca uma URL/DSN
// com credenciais embutidas ("@"). Ver regra do projeto: pgx.ParseConfig().
type DBParams struct {
	Host     string
	Port     string
	Database string
	User     string
	Password string
}

// NewPool cria o pool de conexões a partir de campos individuais, via
// pgxpool.ParseConfig("") + atribuição direta — nunca via string de conexão
// contendo usuário/senha.
func NewPool(ctx context.Context, p DBParams) (*pgxpool.Pool, error) {
	poolCfg, err := pgxpool.ParseConfig("")
	if err != nil {
		return nil, fmt.Errorf("pgxpool.ParseConfig: %w", err)
	}

	port, err := strconv.ParseUint(p.Port, 10, 16)
	if err != nil {
		return nil, fmt.Errorf("porta de banco inválida %q: %w", p.Port, err)
	}

	poolCfg.ConnConfig.Host = p.Host
	poolCfg.ConnConfig.Port = uint16(port)
	poolCfg.ConnConfig.Database = p.Database
	poolCfg.ConnConfig.User = p.User
	poolCfg.ConnConfig.Password = p.Password

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("pgxpool.NewWithConfig: %w", err)
	}
	return pool, nil
}
