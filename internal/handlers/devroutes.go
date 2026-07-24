//go:build devroutes

package handlers

import "github.com/gofiber/fiber/v2"

const DevRoutesEnabled = true

// RegisterDevRoutes só compila com `-tags devroutes` — nunca no build de
// produção (Dockerfile não usa essa tag).
func RegisterDevRoutes(r fiber.Router) {
	dev := r.Group("/dev")
	dev.Get("/ping", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})
}
