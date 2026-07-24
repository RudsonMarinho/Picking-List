package config

import "testing"

func TestCheckFailFast_AbortsOnBypassInProduction(t *testing.T) {
	cases := []struct {
		name string
		cfg  Config
		opts FailFastOptions
	}{
		{
			name: "mail driver diferente de provider",
			cfg:  Config{AppEnv: "production", MailDriver: "log"},
		},
		{
			name: "auto confirm signup ligado",
			cfg:  Config{AppEnv: "production", MailDriver: "provider", AutoConfirmSignup: true},
		},
		{
			name: "seed habilitado",
			cfg:  Config{AppEnv: "production", MailDriver: "provider"},
			opts: FailFastOptions{SeedMode: true},
		},
		{
			name: "rotas dev registradas",
			cfg:  Config{AppEnv: "production", MailDriver: "provider"},
			opts: FailFastOptions{DevRoutesRegistered: true},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.cfg.CheckFailFast(tc.opts); err == nil {
				t.Fatalf("esperava erro FAIL-FAST para o caso %q, obteve nil", tc.name)
			}
		})
	}
}

func TestCheckFailFast_PassesInProductionWithoutBypass(t *testing.T) {
	cfg := Config{AppEnv: "production", MailDriver: "provider"}
	if err := cfg.CheckFailFast(FailFastOptions{}); err != nil {
		t.Fatalf("não esperava erro, obteve: %v", err)
	}
}

func TestCheckFailFast_SkippedOutsideProduction(t *testing.T) {
	cfg := Config{AppEnv: "development", MailDriver: "log", AutoConfirmSignup: true}
	if err := cfg.CheckFailFast(FailFastOptions{SeedMode: true, DevRoutesRegistered: true}); err != nil {
		t.Fatalf("fora de produção não deve haver FAIL-FAST, obteve: %v", err)
	}
}
