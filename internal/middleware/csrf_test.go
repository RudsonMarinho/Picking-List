package middleware

import (
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func newCSRFTestApp() *fiber.App {
	a := fiber.New()
	a.Use(CSRF("csrf_token", false))
	a.Get("/get", func(c *fiber.Ctx) error { return c.SendString("ok") })
	a.Post("/mutate", func(c *fiber.Ctx) error { return c.SendString("ok") })
	return a
}

// mutação sem token CSRF é rejeitada com 403.
func TestCSRF_RejectsMutationWithoutToken(t *testing.T) {
	a := newCSRFTestApp()

	req := httptest.NewRequest(fiber.MethodPost, "/mutate", nil)
	resp, err := a.Test(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	if resp.StatusCode != fiber.StatusForbidden {
		t.Fatalf("esperava 403, obteve %d", resp.StatusCode)
	}
}

func TestCSRF_RejectsMutationWithMismatchedToken(t *testing.T) {
	a := newCSRFTestApp()

	req := httptest.NewRequest(fiber.MethodPost, "/mutate", nil)
	req.Header.Set("Cookie", "csrf_token=aaa")
	req.Header.Set("X-CSRF-Token", "bbb")
	resp, err := a.Test(req)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	if resp.StatusCode != fiber.StatusForbidden {
		t.Fatalf("esperava 403 com tokens divergentes, obteve %d", resp.StatusCode)
	}
}

func TestCSRF_AllowsMutationWithMatchingToken(t *testing.T) {
	a := newCSRFTestApp()

	// GET emite o cookie csrf_token.
	getReq := httptest.NewRequest(fiber.MethodGet, "/get", nil)
	getResp, err := a.Test(getReq)
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	var csrfCookie string
	for _, c := range getResp.Cookies() {
		if c.Name == "csrf_token" {
			csrfCookie = c.Value
		}
	}
	if csrfCookie == "" {
		t.Fatal("GET deveria ter emitido cookie csrf_token")
	}

	postReq := httptest.NewRequest(fiber.MethodPost, "/mutate", nil)
	postReq.Header.Set("Cookie", "csrf_token="+csrfCookie)
	postReq.Header.Set("X-CSRF-Token", csrfCookie)
	postResp, err := a.Test(postReq)
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	if postResp.StatusCode != fiber.StatusOK {
		t.Fatalf("esperava 200 com tokens batendo, obteve %d", postResp.StatusCode)
	}
}
