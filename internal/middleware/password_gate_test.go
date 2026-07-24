package middleware

import (
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
)

// must_change_password=true bloqueia rotas de negócio até a troca.
func TestRequirePasswordChanged_BlocksWhenPending(t *testing.T) {
	a := fiber.New()
	a.Use(func(c *fiber.Ctx) error {
		c.Locals(LocalMustChangePassword, true)
		return c.Next()
	})
	a.Use(RequirePasswordChanged())
	a.Get("/business", func(c *fiber.Ctx) error { return c.SendString("ok") })

	resp, err := a.Test(httptest.NewRequest(fiber.MethodGet, "/business", nil))
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	if resp.StatusCode != fiber.StatusForbidden {
		t.Fatalf("esperava 403 com troca de senha pendente, obteve %d", resp.StatusCode)
	}
}

func TestRequirePasswordChanged_AllowsWhenDone(t *testing.T) {
	a := fiber.New()
	a.Use(func(c *fiber.Ctx) error {
		c.Locals(LocalMustChangePassword, false)
		return c.Next()
	})
	a.Use(RequirePasswordChanged())
	a.Get("/business", func(c *fiber.Ctx) error { return c.SendString("ok") })

	resp, err := a.Test(httptest.NewRequest(fiber.MethodGet, "/business", nil))
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("esperava 200 sem troca de senha pendente, obteve %d", resp.StatusCode)
	}
}
