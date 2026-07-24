package repository

import (
	"context"

	"github.com/google/uuid"
)

type DashboardStats struct {
	TotalCompanies     int `json:"total_companies"`
	TotalBranches      int `json:"total_branches"`
	TotalCollaborators int `json:"total_collaborators"`
	TotalAssets        int `json:"total_assets"`
	AllocatedAssets    int `json:"allocated_assets"`
	AvailableAssets    int `json:"available_assets"`
}

func GetDashboardStats(ctx context.Context, q Queryer, tenantID uuid.UUID) (DashboardStats, error) {
	var s DashboardStats
	err := q.QueryRow(ctx, `
		SELECT
			(SELECT count(*) FROM company      WHERE tenant_id = $1),
			(SELECT count(*) FROM branch       WHERE tenant_id = $1),
			(SELECT count(*) FROM collaborator WHERE tenant_id = $1),
			(SELECT count(*) FROM asset        WHERE tenant_id = $1),
			(SELECT count(*) FROM asset        WHERE tenant_id = $1 AND status = 'allocated'),
			(SELECT count(*) FROM asset        WHERE tenant_id = $1 AND status = 'available')
	`, tenantID).Scan(
		&s.TotalCompanies, &s.TotalBranches, &s.TotalCollaborators,
		&s.TotalAssets, &s.AllocatedAssets, &s.AvailableAssets,
	)
	return s, err
}
