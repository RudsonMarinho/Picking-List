//go:build integration

package app

import (
	"context"
	"io"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/Fortcargo/invtech/internal/repository"
	"github.com/Fortcargo/invtech/internal/services"
	"github.com/Fortcargo/invtech/pkg/config"
	"github.com/Fortcargo/invtech/pkg/logger"
	"github.com/Fortcargo/invtech/pkg/security"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func testDBParams(t *testing.T, envUser, envPassword string) repository.DBParams {
	t.Helper()
	host := os.Getenv("TEST_DB_HOST")
	if host == "" {
		t.Skip("TEST_DB_HOST não definido — pulando teste de integração")
	}
	return repository.DBParams{
		Host:     host,
		Port:     os.Getenv("TEST_DB_PORT"),
		Database: os.Getenv("TEST_DB_NAME"),
		User:     os.Getenv(envUser),
		Password: os.Getenv(envPassword),
	}
}

// MAIL_DRIVER=noop: criar tenant + usuário + login funciona.
func TestLoginFlow_MailDriverNoop(t *testing.T) {
	ctx := context.Background()

	ownerPool, err := repository.NewPool(ctx, testDBParams(t, "TEST_DB_OWNER_USER", "TEST_DB_OWNER_PASSWORD"))
	if err != nil {
		t.Fatalf("pool owner: %v", err)
	}
	defer ownerPool.Close()
	appPool, err := repository.NewPool(ctx, testDBParams(t, "TEST_DB_APP_USER", "TEST_DB_APP_PASSWORD"))
	if err != nil {
		t.Fatalf("pool app: %v", err)
	}
	defer appPool.Close()

	email := "user-" + uuid.NewString() + "@example.com"
	const plainPassword = "senha-inicial-123"

	var tenantID uuid.UUID
	if err := ownerPool.QueryRow(ctx,
		`INSERT INTO tenants (nome, slug) VALUES ('tenant login', $1) RETURNING id`,
		"tenant-login-"+uuid.NewString(),
	).Scan(&tenantID); err != nil {
		t.Fatalf("criar tenant: %v", err)
	}

	hash, err := security.HashPassword(plainPassword)
	if err != nil {
		t.Fatalf("hash senha: %v", err)
	}

	err = repository.WithTenant(ctx, appPool, tenantID, func(tx pgx.Tx) error {
		var roleID uuid.UUID
		if err := tx.QueryRow(ctx,
			`INSERT INTO roles (tenant_id, name) VALUES ($1, 'admin') RETURNING id`, tenantID,
		).Scan(&roleID); err != nil {
			return err
		}
		_, err := repository.CreateUserWithLookup(ctx, tx, tenantID, roleID, email, &hash, "local", false)
		return err
	})
	if err != nil {
		t.Fatalf("criar usuário: %v", err)
	}

	cfg := &config.Config{
		AppEnv:            "development",
		MailDriver:        "noop",
		SessionCookieName: "invtech_session",
		CSRFCookieName:    "csrf_token",
	}
	authSvc := &services.AuthService{Pool: appPool, SessionTTL: 24 * 3600 * 1e9}
	fapp := New(cfg, appPool, authSvc, logger.New(cfg.AppEnv))

	// 1) qualquer GET sob /api/v1 passa pelo middleware de CSRF do grupo,
	// que emite o cookie csrf_token em métodos seguros.
	getResp, err := fapp.Test(httptest.NewRequest(fiber.MethodGet, "/api/v1/auth/login", nil))
	if err != nil {
		t.Fatalf("GET /api/v1/auth/login: %v", err)
	}
	var csrfToken string
	for _, c := range getResp.Cookies() {
		if c.Name == "csrf_token" {
			csrfToken = c.Value
		}
	}
	if csrfToken == "" {
		t.Fatal("esperava cookie csrf_token emitido em GET")
	}

	// 2) POST /api/v1/auth/login com credenciais corretas + CSRF.
	body := `{"email":"` + email + `","password":"` + plainPassword + `"}`
	loginReq := httptest.NewRequest(fiber.MethodPost, "/api/v1/auth/login", strings.NewReader(body))
	loginReq.Header.Set("Content-Type", "application/json")
	loginReq.Header.Set("Cookie", "csrf_token="+csrfToken)
	loginReq.Header.Set("X-CSRF-Token", csrfToken)

	loginResp, err := fapp.Test(loginReq)
	if err != nil {
		t.Fatalf("POST /auth/login: %v", err)
	}
	defer loginResp.Body.Close()
	respBody, _ := io.ReadAll(loginResp.Body)

	if loginResp.StatusCode != fiber.StatusOK {
		t.Fatalf("esperava 200 no login, obteve %d: %s", loginResp.StatusCode, string(respBody))
	}

	var sessionCookie string
	for _, c := range loginResp.Cookies() {
		if c.Name == cfg.SessionCookieName {
			sessionCookie = c.Value
		}
	}
	if sessionCookie == "" {
		t.Fatal("login deveria ter emitido cookie de sessão")
	}
}
