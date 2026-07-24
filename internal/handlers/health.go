package handlers

import (
	"runtime"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Health é o healthcheck mínimo — nunca depende de volume/disco, só
// responde que o processo está de pé.
func Health(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"status": "ok"})
}

// HealthReady confere dependências reais (banco) e reporta memória.
func HealthReady(pool *pgxpool.Pool) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if err := pool.Ping(c.Context()); err != nil {
			return fiber.NewError(fiber.StatusServiceUnavailable, "banco indisponível")
		}
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		return c.JSON(fiber.Map{
			"db": "ok",
			"memory": fiber.Map{
				"alloc_bytes":       m.Alloc,
				"heap_in_use_bytes": m.HeapInuse,
			},
		})
	}
}
