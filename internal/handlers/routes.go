package handlers

import (
	"github.com/Fortcargo/invtech/internal/middleware"
	"github.com/gofiber/fiber/v2"
)

// perm é atalho pra middleware.RequirePermission(module, action).
func perm(module, action string) fiber.Handler { return middleware.RequirePermission(module, action) }

// RegisterBusinessRoutes registra os 9 módulos de negócio + import,
// conforme o Contrato da API em ARQUITETURA.md. Todas essas rotas já
// passam por RequireSession + RequirePasswordChanged (montado em
// internal/app.New) antes de chegar aqui; cada uma agora também exige o par
// (module, action) correspondente em role_permissions — enforcement de RBAC
// que faltava (role_permissions/user_scopes eram gravados mas nunca lidos).
//
// /dashboard fica de fora do gate de permissão de propósito: é a rota que o
// frontend usa pra descobrir se a sessão está viva (checkAuth), tem que
// responder pra qualquer usuário autenticado independente de role.
//
// Escopo de empresa/filial (user_scopes) é autorizado dentro de cada handler
// de create/update/delete via middleware.Scope() — não dá pra fazer isso
// genericamente aqui porque cada módulo resolve o dono (empresa/filial) de um
// jeito diferente. Leitura (list/get) não é filtrada por escopo nesta rodada
// — ver nota em SESSAO.md.
func RegisterBusinessRoutes(r fiber.Router) {
	r.Get("/dashboard", Dashboard)

	r.Get("/companies", perm("companies", "read"), ListCompanies)
	r.Post("/companies", perm("companies", "create"), CreateCompany)
	r.Put("/companies/:id", perm("companies", "update"), UpdateCompany)
	r.Delete("/companies/:id", perm("companies", "delete"), DeleteCompany)
	r.Get("/companies/:id/branches", perm("branches", "read"), ListBranches)
	r.Post("/companies/:id/branches", perm("branches", "create"), CreateBranch)

	r.Put("/branches/:id", perm("branches", "update"), UpdateBranch)
	r.Delete("/branches/:id", perm("branches", "delete"), DeleteBranch)
	r.Get("/branches/:id/sectors", perm("sectors", "read"), ListSectors)
	r.Post("/branches/:id/sectors", perm("sectors", "create"), CreateSector)

	r.Put("/sectors/:id", perm("sectors", "update"), UpdateSector)
	r.Delete("/sectors/:id", perm("sectors", "delete"), DeleteSector)

	r.Get("/collaborators", perm("collaborators", "read"), ListCollaborators)
	r.Post("/collaborators", perm("collaborators", "create"), CreateCollaborator)
	r.Put("/collaborators/:id", perm("collaborators", "update"), UpdateCollaborator)
	r.Delete("/collaborators/:id", perm("collaborators", "delete"), DeleteCollaborator)

	r.Get("/asset-types", perm("asset_types", "read"), ListAssetTypes)
	r.Post("/asset-types", perm("asset_types", "create"), CreateAssetType)
	r.Put("/asset-types/:id", perm("asset_types", "update"), UpdateAssetType)
	r.Delete("/asset-types/:id", perm("asset_types", "delete"), DeleteAssetType)

	r.Get("/assets", perm("assets", "read"), ListAssets)
	r.Post("/assets", perm("assets", "create"), CreateAsset)
	r.Put("/assets/:id", perm("assets", "update"), UpdateAsset)
	r.Delete("/assets/:id", perm("assets", "delete"), DeleteAsset)

	r.Get("/allocations", perm("allocations", "read"), ListAllocations)
	r.Post("/allocations", perm("allocations", "create"), CreateAllocation)
	r.Post("/allocations/:id/return", perm("allocations", "update"), ReturnAllocation)

	r.Get("/users", perm("users", "read"), ListUsers)
	r.Post("/users", perm("users", "create"), CreateUser)
	r.Put("/users/:id", perm("users", "update"), UpdateUser)
	r.Delete("/users/:id", perm("users", "delete"), DeleteUser)

	r.Get("/roles", perm("roles", "read"), ListRoles)
	r.Post("/roles", perm("roles", "create"), CreateRole)

	r.Post("/import", perm("import", "create"), Import)
}
