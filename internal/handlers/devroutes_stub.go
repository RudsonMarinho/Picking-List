//go:build !devroutes

package handlers

import "github.com/gofiber/fiber/v2"

// DevRoutesEnabled é false no build padrão. Rotas /api/v1/dev/* só existem
// compiladas com a tag de build "devroutes" — defesa em profundidade do
// FAIL-FAST: mesmo que o runtime esqueça de checar, elas nem existem no
// binário de produção.
const DevRoutesEnabled = false

func RegisterDevRoutes(_ fiber.Router) {}
