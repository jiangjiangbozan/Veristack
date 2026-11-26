// Package milestone 的业务服务，负责里程碑的增删改查与关联需求维护。
package milestone

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"api/internal/platform/database"
	"api/internal/platform/db"
	"api/internal/platform/dbutil"
	"api/internal/platform/errs"
)

// Service 提供里程碑相关的业务方法。
type Service struct {
	Database *pgxpool.Pool // 数据库连接池
}

// Create 在项目下创建里程碑。
func (s *Service) Create(ctx context.Context, projectID uuid.UUID, req CreateMilestoneRequest) (db.Milestone, error) {
	targetDate, err := dbutil.DateFromString(req.TargetDate)
	if err != nil {
		return db.Milestone{}, errs.BadRequest(err.Error(), err)
	}
	requirementIDs, err := dbutil.JSONBytes(req.RequirementIDs)
	if err != nil {
		return db.Milestone{}, err
	}
	milestone, err := database.Queries(ctx, s.Database).CreateMilestone(ctx, db.CreateMilestoneParams{
		Name:           req.Name,
		ProjectID:      projectID,
		RequirementIds: requirementIDs,
		TargetDate:     targetDate,
	})
	if err != nil {
		return db.Milestone{}, fmt.Errorf("创建里程碑失败: %w", err)
	}
	return milestone, nil
}

// Get 按 ID 读取里程碑。
func (s *Service) Get(ctx context.Context, id uuid.UUID) (db.Milestone, error) {
	milestone, err := database.Queries(ctx, s.Database).GetMilestone(ctx, id)
	if err != nil {
		return db.Milestone{}, errs.NotFound("里程碑不存在", err)
	}
	return milestone, nil
}

// List 列出项目全部里程碑。
func (s *Service) List(ctx context.Context, projectID uuid.UUID) ([]db.Milestone, error) {
	return database.Queries(ctx, s.Database).ListMilestonesByProject(ctx, projectID)
}

// SoftDelete 软删除里程碑。
func (s *Service) SoftDelete(ctx context.Context, id uuid.UUID) error {
	return database.Queries(ctx, s.Database).SoftDeleteMilestone(ctx, id)
}

// Update 更新里程碑字段。
func (s *Service) Update(ctx context.Context, id uuid.UUID, req UpdateMilestoneRequest) (db.Milestone, error) {
	targetDate, err := dbutil.DateFromString(req.TargetDate)
	if err != nil {
		return db.Milestone{}, errs.BadRequest(err.Error(), err)
	}
	requirementIDs, err := dbutil.JSONBytes(req.RequirementIDs)
	if err != nil {
		return db.Milestone{}, err
	}
	milestone, err := database.Queries(ctx, s.Database).UpdateMilestone(ctx, db.UpdateMilestoneParams{
		ID:             id,
		Name:           req.Name,
		RequirementIds: requirementIDs,
		Status:         req.Status,
		TargetDate:     targetDate,
	})
	if err != nil {
		return db.Milestone{}, fmt.Errorf("更新里程碑失败: %w", err)
	}
	return milestone, nil
}
