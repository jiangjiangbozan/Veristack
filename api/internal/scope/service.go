// Package scope 的业务服务，负责需求基线的版本创建、冻结与差异对比。
package scope

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"api/internal/platform/database"
	"api/internal/platform/db"
	"api/internal/platform/dbutil"
	"api/internal/platform/errs"
	"api/internal/platform/views"
)

// Service 提供范围基线相关的业务方法。
type Service struct {
	Database *pgxpool.Pool // 数据库连接池
}

// Create 在事务内创建下一版本基线并写入需求，创建后立即冻结为正式版本。
func (s *Service) Create(ctx context.Context, projectID uuid.UUID, req CreateBaselineRequest) (db.ScopeBaseline, error) {
	queries := database.Queries(ctx, s.Database)
	nextVersion, err := s.nextVersion(ctx, queries, projectID)
	if err != nil {
		return db.ScopeBaseline{}, err
	}
	requirementViews := toRequirementViews(req.Requirements)
	snapshot, err := dbutil.JSONBytes(requirementViews)
	if err != nil {
		return db.ScopeBaseline{}, err
	}
	baseline, err := queries.CreateScopeBaseline(ctx, db.CreateScopeBaselineParams{
		ProjectID:        projectID,
		RequirementsJson: snapshot,
		Version:          nextVersion,
	})
	if err != nil {
		return db.ScopeBaseline{}, fmt.Errorf("创建基线失败: %w", err)
	}
	for _, requirement := range req.Requirements {
		criteria, err := dbutil.JSONBytes(requirement.AcceptanceCriteria)
		if err != nil {
			return db.ScopeBaseline{}, err
		}
		if _, err := queries.CreateRequirement(ctx, db.CreateRequirementParams{
			AcceptanceCriteria: criteria,
			BaselineID:         baseline.ID,
			Code:               requirement.Code,
			Description:        dbutil.TextFromString(requirement.Description),
			Priority:           requirement.Priority,
		}); err != nil {
			return db.ScopeBaseline{}, fmt.Errorf("创建需求 %s 失败: %w", requirement.Code, err)
		}
	}
	frozen, err := queries.FreezeScopeBaseline(ctx, db.FreezeScopeBaselineParams{
		ID:              baseline.ID,
		ChangeRequestID: pgtype.UUID{},
	})
	if err != nil {
		return db.ScopeBaseline{}, fmt.Errorf("冻结基线失败: %w", err)
	}
	return frozen, nil
}

// CreateFromSnapshot 依据变更申请的需求快照创建新基线，供客户确认变更时调用。
func (s *Service) CreateFromSnapshot(ctx context.Context, projectID, changeRequestID uuid.UUID, requirements []views.RequirementView) (db.ScopeBaseline, error) {
	queries := database.Queries(ctx, s.Database)
	nextVersion, err := s.nextVersion(ctx, queries, projectID)
	if err != nil {
		return db.ScopeBaseline{}, err
	}
	snapshot, err := dbutil.JSONBytes(requirements)
	if err != nil {
		return db.ScopeBaseline{}, err
	}
	baseline, err := queries.CreateScopeBaseline(ctx, db.CreateScopeBaselineParams{
		ProjectID:        projectID,
		RequirementsJson: snapshot,
		Version:          nextVersion,
	})
	if err != nil {
		return db.ScopeBaseline{}, fmt.Errorf("创建变更基线失败: %w", err)
	}
	frozen, err := queries.FreezeScopeBaseline(ctx, db.FreezeScopeBaselineParams{
		ID:              baseline.ID,
		ChangeRequestID: pgtype.UUID{Bytes: changeRequestID, Valid: true},
	})
	if err != nil {
		return db.ScopeBaseline{}, fmt.Errorf("冻结变更基线失败: %w", err)
	}
	return frozen, nil
}

// Diff 对比两个基线的需求差异，以基准基线为参照。
func (s *Service) Diff(ctx context.Context, baseID, targetID uuid.UUID) (DiffBaselineResponse, error) {
	base, err := database.Queries(ctx, s.Database).GetScopeBaseline(ctx, baseID)
	if err != nil {
		return DiffBaselineResponse{}, errs.NotFound("基准基线不存在", err)
	}
	target, err := database.Queries(ctx, s.Database).GetScopeBaseline(ctx, targetID)
	if err != nil {
		return DiffBaselineResponse{}, errs.NotFound("对比基线不存在", err)
	}
	baseItems := parseRequirements(base.RequirementsJson)
	targetItems := parseRequirements(target.RequirementsJson)
	return diffRequirements(baseItems, targetItems), nil
}

// Get 按 ID 读取基线。
func (s *Service) Get(ctx context.Context, id uuid.UUID) (db.ScopeBaseline, error) {
	baseline, err := database.Queries(ctx, s.Database).GetScopeBaseline(ctx, id)
	if err != nil {
		return db.ScopeBaseline{}, errs.NotFound("基线不存在", err)
	}
	return baseline, nil
}

// List 列出项目的全部基线版本。
func (s *Service) List(ctx context.Context, projectID uuid.UUID) ([]db.ScopeBaseline, error) {
	return database.Queries(ctx, s.Database).ListScopeBaselines(ctx, projectID)
}

// nextVersion 计算下一版本号，项目尚无基线时从 1 开始。
func (s *Service) nextVersion(ctx context.Context, queries *db.Queries, projectID uuid.UUID) (int32, error) {
	latest, err := queries.GetLatestBaseline(ctx, projectID)
	if err != nil {
		return 1, nil
	}
	return latest.Version + 1, nil
}

// toRequirementViews 把输入需求转换为带稳定标识的视图，作为基线快照。
func toRequirementViews(inputs []RequirementInput) []views.RequirementView {
	result := make([]views.RequirementView, 0, len(inputs))
	for _, input := range inputs {
		result = append(result, views.RequirementView{
			AcceptanceCriteria: input.AcceptanceCriteria,
			Code:               input.Code,
			Description:        input.Description,
			ID:                 uuid.New(),
			Priority:           input.Priority,
			Status:             "pending",
			Title:              input.Title,
		})
	}
	return result
}

// parseRequirements 从 JSONB 字节解析需求视图列表。
func parseRequirements(raw []byte) []views.RequirementView {
	items := make([]views.RequirementView, 0)
	_ = dbutil.FromJSONBytes(raw, &items)
	return items
}

// diffRequirements 计算从基准到目标的需求变化。
func diffRequirements(base, target []views.RequirementView) DiffBaselineResponse {
	baseByCode := make(map[string]views.RequirementView, len(base))
	for _, item := range base {
		baseByCode[item.Code] = item
	}
	targetByCode := make(map[string]views.RequirementView, len(target))
	for _, item := range target {
		targetByCode[item.Code] = item
	}
	result := DiffBaselineResponse{
		Added:    make([]views.RequirementView, 0),
		Modified: make([]views.RequirementView, 0),
		Removed:  make([]views.RequirementView, 0),
	}
	for code, item := range targetByCode {
		old, exists := baseByCode[code]
		if !exists {
			result.Added = append(result.Added, item)
			continue
		}
		if old.Title != item.Title || old.Description != item.Description || old.Priority != item.Priority {
			result.Modified = append(result.Modified, item)
		}
	}
	for code, item := range baseByCode {
		if _, exists := targetByCode[code]; !exists {
			result.Removed = append(result.Removed, item)
		}
	}
	return result
}
