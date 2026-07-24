// Command invtech-api é o binário único do InvTech: serve a API (modo
// padrão), aplica migrations (-migrate) ou faz o healthcheck do container
// (-healthcheck). FAIL-FAST roda antes de qualquer outra coisa.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/Fortcargo/invtech/internal/app"
	"github.com/Fortcargo/invtech/internal/handlers"
	"github.com/Fortcargo/invtech/internal/repository"
	"github.com/Fortcargo/invtech/internal/seed"
	"github.com/Fortcargo/invtech/internal/services"
	"github.com/Fortcargo/invtech/migrations"
	"github.com/Fortcargo/invtech/pkg/config"
	applogger "github.com/Fortcargo/invtech/pkg/logger"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
)

func main() {
	migrateFlag := flag.Bool("migrate", false, "aplica as migrations pendentes (usa DB_MIGRATE_USER) e sai")
	healthcheckFlag := flag.Bool("healthcheck", false, "checa GET /health do processo já em execução e sai (usado pelo HEALTHCHECK do Docker)")
	seedFlag := flag.Bool("seed", false, "cria o tenant demo completo (R10) e sai — NUNCA em produção")
	flag.Parse()

	if *healthcheckFlag {
		runHealthcheck()
		return
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	logg := applogger.New(cfg.AppEnv)

	// FAIL-FAST: bloqueador absoluto, antes de qualquer migration ou boot.
	if err := cfg.CheckFailFast(config.FailFastOptions{
		SeedMode:            *seedFlag,
		DevRoutesRegistered: handlers.DevRoutesEnabled,
	}); err != nil {
		logg.Error("FAIL-FAST: abortando boot", slog.Any("err", err))
		os.Exit(1)
	}

	if *migrateFlag {
		runMigrations(cfg, logg)
		return
	}

	if *seedFlag {
		runSeed(cfg, logg)
		return
	}

	runServer(cfg, logg)
}

func runHealthcheck() {
	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "8080"
	}
	client := http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(fmt.Sprintf("http://127.0.0.1:%s/health", port))
	if err != nil {
		os.Exit(1)
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	if resp.StatusCode != http.StatusOK {
		os.Exit(1)
	}
	os.Exit(0)
}

func runMigrations(cfg *config.Config, logg *slog.Logger) {
	connCfg, err := pgx.ParseConfig("")
	if err != nil {
		logg.Error("pgx.ParseConfig", slog.Any("err", err))
		os.Exit(1)
	}
	port, err := strconv.ParseUint(cfg.DBPort, 10, 16)
	if err != nil {
		logg.Error("porta de banco inválida", slog.String("port", cfg.DBPort))
		os.Exit(1)
	}
	connCfg.Host = cfg.DBHost
	connCfg.Port = uint16(port)
	connCfg.Database = cfg.DBName
	connCfg.User = cfg.DBMigrateUser
	connCfg.Password = cfg.DBMigratePassword

	sqlDB := stdlib.OpenDB(*connCfg)
	defer sqlDB.Close()

	driver, err := postgres.WithInstance(sqlDB, &postgres.Config{})
	if err != nil {
		logg.Error("postgres.WithInstance", slog.Any("err", err))
		os.Exit(1)
	}
	sourceDriver, err := iofs.New(migrations.FS, ".")
	if err != nil {
		logg.Error("iofs.New", slog.Any("err", err))
		os.Exit(1)
	}
	m, err := migrate.NewWithInstance("iofs", sourceDriver, cfg.DBName, driver)
	if err != nil {
		logg.Error("migrate.NewWithInstance", slog.Any("err", err))
		os.Exit(1)
	}

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		logg.Error("migrate.Up", slog.Any("err", err))
		os.Exit(1)
	}
	logg.Info("migrations aplicadas com sucesso")
}

func runSeed(cfg *config.Config, logg *slog.Logger) {
	ctx := context.Background()

	pool, err := repository.NewPool(ctx, repository.DBParams{
		Host:     cfg.DBHost,
		Port:     cfg.DBPort,
		Database: cfg.DBName,
		User:     cfg.DBUser,
		Password: cfg.DBPassword,
	})
	if err != nil {
		logg.Error("repository.NewPool", slog.Any("err", err))
		os.Exit(1)
	}
	defer pool.Close()

	if err := seed.Run(ctx, pool, logg); err != nil {
		logg.Error("seed.Run", slog.Any("err", err))
		os.Exit(1)
	}
}

func runServer(cfg *config.Config, logg *slog.Logger) {
	ctx := context.Background()

	pool, err := repository.NewPool(ctx, repository.DBParams{
		Host:     cfg.DBHost,
		Port:     cfg.DBPort,
		Database: cfg.DBName,
		User:     cfg.DBUser,
		Password: cfg.DBPassword,
	})
	if err != nil {
		logg.Error("repository.NewPool", slog.Any("err", err))
		os.Exit(1)
	}
	defer pool.Close()

	authSvc := &services.AuthService{Pool: pool, SessionTTL: 24 * time.Hour}
	fapp := app.New(cfg, pool, authSvc, logg)

	logg.Info("invtech-api ouvindo", slog.String("port", cfg.AppPort))
	if err := fapp.Listen(":" + cfg.AppPort); err != nil {
		logg.Error("app.Listen", slog.Any("err", err))
		os.Exit(1)
	}
}
