package repository

import (
	"context"

	"github.com/google/uuid"
)

type Role struct {
	TenantID    uuid.UUID `json:"tenant_id"`
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description *string   `json:"description,omitempty"`
}

type Permission struct {
	Module string `json:"module"`
	Action string `json:"action"`
}

func ListRoles(ctx context.Context, q Queryer, tenantID uuid.UUID) ([]Role, error) {
	rows, err := q.Query(ctx,
		`SELECT tenant_id, id, name, description FROM roles WHERE tenant_id = $1 ORDER BY name`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []Role{}
	for rows.Next() {
		var r Role
		if err := rows.Scan(&r.TenantID, &r.ID, &r.Name, &r.Description); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func CreateRole(ctx context.Context, q Queryer, tenantID uuid.UUID, name string, description *string) (Role, error) {
	r := Role{TenantID: tenantID, Name: name, Description: description}
	err := q.QueryRow(ctx,
		`INSERT INTO roles (tenant_id, name, description) VALUES ($1, $2, $3) RETURNING id`,
		tenantID, name, description,
	).Scan(&r.ID)
	return r, err
}

// AddRolePermissions insere os pares (module, action) da role — chamado na
// mesma transação de CreateRole.
func AddRolePermissions(ctx context.Context, q Queryer, tenantID, roleID uuid.UUID, perms []Permission) error {
	for _, p := range perms {
		if _, err := q.Exec(ctx,
			`INSERT INTO role_permissions (tenant_id, role_id, module, action) VALUES ($1, $2, $3, $4)`,
			tenantID, roleID, p.Module, p.Action); err != nil {
			return err
		}
	}
	return nil
}

// AllModules e AllActions — vocabulário fechado de módulo/ação usado tanto
// pelo seed (internal/seed) quanto por fixtures de teste que precisam de uma
// role com acesso total, sem duplicar a lista em cada lugar.
var AllModules = []string{
	"dashboard", "companies", "branches", "sectors", "collaborators",
	"asset_types", "assets", "allocations", "users", "roles", "import",
}
var AllActions = []string{"create", "read", "update", "delete"}

// FullPermissionSet devolve o produto cartesiano AllModules × AllActions —
// acesso total, usado pra popular a role admin do seed e de fixtures de
// teste que simulam um usuário sem restrição de RBAC.
func FullPermissionSet() []Permission {
	perms := make([]Permission, 0, len(AllModules)*len(AllActions))
	for _, m := range AllModules {
		for _, a := range AllActions {
			perms = append(perms, Permission{Module: m, Action: a})
		}
	}
	return perms
}

// HasPermission confere se a role tem o par (module, action) em
// role_permissions — base do enforcement de RBAC (middleware.RequirePermission).
func HasPermission(ctx context.Context, q Queryer, tenantID, roleID uuid.UUID, module, action string) (bool, error) {
	var exists bool
	err := q.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM role_permissions WHERE tenant_id = $1 AND role_id = $2 AND module = $3 AND action = $4)`,
		tenantID, roleID, module, action,
	).Scan(&exists)
	return exists, err
}

func ListRolePermissions(ctx context.Context, q Queryer, tenantID, roleID uuid.UUID) ([]Permission, error) {
	rows, err := q.Query(ctx,
		`SELECT module, action FROM role_permissions WHERE tenant_id = $1 AND role_id = $2`, tenantID, roleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []Permission{}
	for rows.Next() {
		var p Permission
		if err := rows.Scan(&p.Module, &p.Action); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}
