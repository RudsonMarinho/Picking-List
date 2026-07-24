package handlers

import (
	"errors"

	"github.com/Fortcargo/invtech/internal/middleware"
	"github.com/Fortcargo/invtech/internal/repository"
	"github.com/Fortcargo/invtech/internal/services"
	"github.com/Fortcargo/invtech/pkg/config"
	"github.com/gofiber/fiber/v2"
)

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Login é rota pública (fora de RequireSession) — mas ainda passa pelo
// middleware de CSRF, como qualquer outra mutação.
func Login(svc *services.AuthService, cfg *config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var body loginRequest
		if err := c.BodyParser(&body); err != nil || body.Email == "" || body.Password == "" {
			return fiber.NewError(fiber.StatusBadRequest, "email e password são obrigatórios")
		}

		token, expiresAt, user, err := svc.Login(c.Context(), body.Email, body.Password, c.IP(), string(c.Request().Header.UserAgent()))
		if err != nil {
			if errors.Is(err, services.ErrInvalidCredentials) {
				return fiber.NewError(fiber.StatusUnauthorized, "credenciais inválidas")
			}
			return fiber.NewError(fiber.StatusInternalServerError, "falha no login")
		}

		c.Cookie(&fiber.Cookie{
			Name:     cfg.SessionCookieName,
			Value:    token,
			HTTPOnly: true,
			Secure:   cfg.AppEnv == "production",
			SameSite: fiber.CookieSameSiteLaxMode,
			Path:     "/",
			Expires:  expiresAt,
		})

		return c.JSON(fiber.Map{
			"user_info": fiber.Map{
				"id":                   user.ID,
				"email":                user.Email,
				"must_change_password": user.MustChangePassword,
			},
		})
	}
}

// Logout usa a Tx e os dados de sessão já resolvidos por RequireSession.
func Logout(cfg *config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tx := middleware.Tx(c)
		tokenHash, _ := c.Locals(middleware.LocalSessionTokenHash).(string)

		if err := repository.RevokeSession(c.Context(), tx, middleware.TenantID(c), middleware.UserID(c), tokenHash); err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, "falha ao encerrar sessão")
		}

		c.ClearCookie(cfg.SessionCookieName)
		return c.JSON(fiber.Map{"status": "ok"})
	}
}

type changePasswordRequest struct {
	Old string `json:"old"`
	New string `json:"new"`
}

// ChangePassword é a ÚNICA rota de negócio isenta do gate
// RequirePasswordChanged — é como o usuário satisfaz o gate.
func ChangePassword(svc *services.AuthService) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var body changePasswordRequest
		if err := c.BodyParser(&body); err != nil || body.Old == "" || body.New == "" {
			return fiber.NewError(fiber.StatusBadRequest, "old e new são obrigatórios")
		}
		if len(body.New) < 8 {
			return fiber.NewError(fiber.StatusBadRequest, "nova senha muito curta")
		}

		err := svc.ChangePassword(c.Context(), middleware.Tx(c), middleware.TenantID(c), middleware.UserID(c), body.Old, body.New)
		if err != nil {
			if errors.Is(err, services.ErrInvalidCredentials) {
				return fiber.NewError(fiber.StatusUnauthorized, "senha atual incorreta")
			}
			return fiber.NewError(fiber.StatusInternalServerError, "falha ao trocar senha")
		}
		return c.JSON(fiber.Map{"status": "ok"})
	}
}
