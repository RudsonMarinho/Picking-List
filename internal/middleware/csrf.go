package middleware

import (
	"crypto/rand"
	"encoding/base64"

	"github.com/Fortcargo/invtech/pkg/security"
	"github.com/gofiber/fiber/v2"
)

// CSRF implementa double-submit: em métodos seguros, garante que o cookie
// csrf_token exista (emite se faltar). Em métodos de mutação, exige que o
// cookie e o header X-CSRF-Token existam e sejam iguais (tempo constante) —
// SEM token, ou token divergente, 403 ANTES de qualquer acesso ao banco
// (roda antes de RequireSession na cadeia de middlewares).
func CSRF(cookieName string, secure bool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if isSafeMethod(c.Method()) {
			if c.Cookies(cookieName) == "" {
				token, err := newCSRFToken()
				if err != nil {
					return fiber.NewError(fiber.StatusInternalServerError, "falha ao gerar token CSRF")
				}
				c.Cookie(&fiber.Cookie{
					Name:     cookieName,
					Value:    token,
					HTTPOnly: false,
					Secure:   secure,
					SameSite: fiber.CookieSameSiteLaxMode,
					Path:     "/",
				})
			}
			return c.Next()
		}

		cookieToken := c.Cookies(cookieName)
		headerToken := c.Get("X-CSRF-Token")
		if cookieToken == "" || headerToken == "" || !security.ConstantTimeEquals(cookieToken, headerToken) {
			return fiber.NewError(fiber.StatusForbidden, "token CSRF ausente ou inválido")
		}
		return c.Next()
	}
}

func isSafeMethod(method string) bool {
	switch method {
	case fiber.MethodGet, fiber.MethodHead, fiber.MethodOptions:
		return true
	default:
		return false
	}
}

func newCSRFToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}
