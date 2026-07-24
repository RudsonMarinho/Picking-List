// Package app monta o *fiber.App com todas as rotas e middlewares — usado
// tanto por cmd/server/main.go quanto pelos testes de integração, para que
// o comportamento testado seja exatamente o que roda em produção.
package app

import (
	"errors"
	"log/slog"

	"github.com/Fortcargo/invtech/internal/handlers"
	"github.com/Fortcargo/invtech/internal/middleware"
	"github.com/Fortcargo/invtech/internal/services"
	"github.com/Fortcargo/invtech/pkg/config"
	"github.com/Fortcargo/invtech/web"
	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
)

func New(cfg *config.Config, pool *pgxpool.Pool, authSvc *services.AuthService, logg *slog.Logger) *fiber.App {
	fapp := fiber.New(fiber.Config{
		DisableStartupMessage: true,
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			var fe *fiber.Error
			if errors.As(err, &fe) {
				code = fe.Code
			} else if logg != nil {
				logg.Error("erro não tratado", slog.Any("err", err), slog.String("path", c.Path()))
			}
			return c.Status(code).JSON(fiber.Map{"error": err.Error()})
		},
	})

	fapp.Get("/", func(c *fiber.Ctx) error {
		c.Set(fiber.HeaderContentType, "text/html; charset=utf-8")
		return c.Send(web.IndexHTML)
	})
	fapp.Get("/health", handlers.Health)
	fapp.Get("/health/ready", handlers.HealthReady(pool))

	secureCookies := cfg.AppEnv == "production"
	api := fapp.Group("/api/v1", middleware.CSRF(cfg.CSRFCookieName, secureCookies))

	api.Post("/auth/login", handlers.Login(authSvc, cfg))
	handlers.RegisterDevRoutes(api)

	authenticated := api.Group("", middleware.RequireSession(pool, cfg.SessionCookieName))
	authenticated.Post("/auth/logout", handlers.Logout(cfg))
	authenticated.Post("/auth/change-password", handlers.ChangePassword(authSvc))

	business := api.Group("", middleware.RequireSession(pool, cfg.SessionCookieName), middleware.RequirePasswordChanged())
	handlers.RegisterBusinessRoutes(business)

	return fapp
}
