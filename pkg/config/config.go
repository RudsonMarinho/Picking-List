// Package config carrega a configuração da aplicação a partir de variáveis
// de ambiente e aplica o bloqueador FAIL-FAST descrito em ARQUITETURA.md.
package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	AppPort       string
	AppEnv        string
	ErrorIDPrefix string

	DBHost     string
	DBPort     string
	DBName     string
	DBUser     string
	DBPassword string

	// DBMigrateUser/DBMigratePassword: role dona do schema, usada SOMENTE
	// para rodar migrations. Nunca a mesma credencial de DBUser/DBPassword
	// (runtime) — ver P1 da MentorDecision.
	DBMigrateUser     string
	DBMigratePassword string

	SessionCookieName string
	CSRFCookieName    string

	MailDriver           string
	AutoConfirmSignup    bool
	FeaturePublicSignup  bool
	FeatureBilling       bool
	FeatureEmailDelivery bool
}

// Load lê a configuração do ambiente. Não valida FAIL-FAST — isso é
// responsabilidade explícita de CheckFailFast, chamada separadamente pelo
// cmd/server/main.go antes de qualquer outra inicialização.
func Load() (*Config, error) {
	cfg := &Config{
		AppPort:       getenv("APP_PORT", "8080"),
		AppEnv:        os.Getenv("APP_ENV"),
		ErrorIDPrefix: getenv("ERROR_ID_PREFIX", "INV"),

		DBHost:     os.Getenv("DB_HOST"),
		DBPort:     getenv("DB_PORT", "5432"),
		DBName:     os.Getenv("DB_NAME"),
		DBUser:     os.Getenv("DB_USER"),
		DBPassword: os.Getenv("DB_PASSWORD"),

		DBMigrateUser:     os.Getenv("DB_MIGRATE_USER"),
		DBMigratePassword: os.Getenv("DB_MIGRATE_PASSWORD"),

		SessionCookieName: getenv("SESSION_COOKIE_NAME", "invtech_session"),
		CSRFCookieName:    getenv("CSRF_COOKIE_NAME", "csrf_token"),

		MailDriver: getenv("MAIL_DRIVER", "log"),
	}

	var err error
	if cfg.AutoConfirmSignup, err = getenvBool("AUTO_CONFIRM_SIGNUP"); err != nil {
		return nil, err
	}
	if cfg.FeaturePublicSignup, err = getenvBool("FEATURE_PUBLIC_SIGNUP"); err != nil {
		return nil, err
	}
	if cfg.FeatureBilling, err = getenvBool("FEATURE_BILLING"); err != nil {
		return nil, err
	}
	if cfg.FeatureEmailDelivery, err = getenvBool("FEATURE_EMAIL_DELIVERY"); err != nil {
		return nil, err
	}

	return cfg, nil
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getenvBool(key string) (bool, error) {
	v := os.Getenv(key)
	if v == "" {
		return false, nil
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return false, fmt.Errorf("env %s: valor booleano inválido %q", key, v)
	}
	return b, nil
}
