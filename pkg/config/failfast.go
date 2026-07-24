package config

import (
	"fmt"
	"strings"
)

// FailFastOptions carrega sinais de runtime que a config sozinha não conhece
// (modo de execução do binário, rotas registradas).
type FailFastOptions struct {
	SeedMode            bool
	DevRoutesRegistered bool
}

// CheckFailFast aplica o bloqueador de ARQUITETURA.md § FAIL-FAST: em
// APP_ENV=production, nenhuma das condições de bypass pode estar ativa.
// Retorna erro em vez de abortar o processo diretamente — quem decide
// encerrar (os.Exit) é sempre cmd/server/main.go, para manter esta função
// testável sem derrubar o processo de teste.
func (c *Config) CheckFailFast(opts FailFastOptions) error {
	if c.AppEnv != "production" {
		return nil
	}

	var violations []string
	if c.MailDriver != "provider" {
		violations = append(violations, fmt.Sprintf("MAIL_DRIVER=%q (exige \"provider\" em produção)", c.MailDriver))
	}
	if c.AutoConfirmSignup {
		violations = append(violations, "AUTO_CONFIRM_SIGNUP=true")
	}
	if opts.SeedMode {
		violations = append(violations, "seed habilitado")
	}
	if opts.DevRoutesRegistered {
		violations = append(violations, "rotas /api/v1/dev/* registradas")
	}

	if len(violations) > 0 {
		return fmt.Errorf("FAIL-FAST: APP_ENV=production com bypass(es) ativo(s): %s", strings.Join(violations, "; "))
	}
	return nil
}
