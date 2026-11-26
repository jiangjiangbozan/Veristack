// Package change 的业务服务，负责变更申请发起与客户确认生成新基线。
package change

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

const (
	decisionApprove = "approve"
	decisionReject  = "reject"
	statusApproved  = "approved"
	statusRejected  = "rejected"
)

// Service 提供变更申请相关的业务方法。
type Service struct {
	Database *pgxpool.Pool // 数据库连接池
}

// Create 基于项目最新基线创建变更申请，并保存拟议的需求快照。
func (s *Service) Create(ctx context.Context, projectID, requesterID uuid.UUID, req CreateChangeRequestRequest) (db.ChangeRequest, error) {
	queries := database.Queries(ctx, s.Database)
	baseline, err := queries.GetLatestBaseline(ctx, projectID)
	if err != nil {
		return db.ChangeRequest{}, errs.NotFound("项目尚无需求基线", err)
	}
	proposed := applyChanges(parseRequirements(baseline.RequirementsJson), req.Requirements)
	snapshot, err := dbutil.JSONBytes(proposed)
	if err != nil {
		return db.ChangeRequest{}, err
	}
	created, err := queries.CreateChangeRequest(ctx, db.CreateChangeRequestParams{
		BaselineID:  baseline.ID,
		Impact:      dbutil.TextFromString(req.Impact),
		ProjectID:   projectID,
		Reason:      req.Reason,
		RequesterID: requesterID,
		Type:        req.Type,
	})
	if err != nil {
		return db.ChangeRequest{}, fmt.Errorf("创建变更申请失败: %w", err)
	}
	if _, err := queries.SetChangeRequestSnapshot(ctx, db.SetChangeRequestSnapshotParams{
		ID:               created.ID,
		RequirementsJson: snapshot,
	}); err != nil {
		return db.ChangeRequest{}, fmt.Errorf("保存变更快照失败: %w", err)
	}
	result, err := queries.GetChangeRequest(ctx, created.ID)
	if err != nil {
		return db.ChangeRequest{}, err
	}
	return result, nil
}

// CountByProject 统计项目变更申请总数。
func (s *Service) CountByProject(ctx context.Context, projectID uuid.UUID) (int64, error) {
	return database.Queries(ctx, s.Database).CountChangeRequestsByProject(ctx, projectID)
}

// Decide 记录客户决定，批准时在同一事务内生成新的基线版本。
func (s *Service) Decide(ctx context.Context, changeID, decidedBy uuid.UUID, req DecisionRequest) (db.ChangeRequest, *db.ScopeBaseline, error) {
	queries := database.Queries(ctx, s.Database)
	changeRequest, err := queries.GetChangeRequest(ctx, changeID)
	if err != nil {
		return db.ChangeRequest{}, nil, errs.NotFound("变更申请不存在", err)
	}
	if changeRequest.Status != "pending" {
		return db.ChangeRequest{}, nil, errs.BadRequest("该变更申请已处理", nil)
	}
	status := statusRejected
	if req.Decision == decisionApprove {
		status = statusApproved
	}
	decided, err := queries.DecideChangeRequest(ctx, db.DecideChangeRequestParams{
		DecidedByID: pgtype.UUID{Bytes: decidedBy, Valid: true},
		ID:          changeID,
		Status:      status,
	})
	if err != nil {
		return db.ChangeRequest{}, nil, fmt.Errorf("记录变更决定失败: %w", err)
	}
	if req.Decision != decisionApprove {
		return decided, nil, nil
	}
	proposed := parseRequirements(changeRequest.RequirementsJson)
	baseline, err := createBaselineFromSnapshot(ctx, s.Database, changeRequest.ProjectID, changeID, proposed)
	if err != nil {
		return db.ChangeRequest{}, nil, err
	}
	return decided, baseline, nil
}

// Get 按 ID 读取变更申请。
func (s *Service) Get(ctx context.Context, id uuid.UUID) (db.ChangeRequest, error) {
	changeRequest, err := database.Queries(ctx, s.Database).GetChangeRequest(ctx, id)
	if err != nil {
		return db.ChangeRequest{}, errs.NotFound("变更申请不存在", err)
	}
	return changeRequest, nil
}

// List 分页列出项目变更申请。
func (s *Service) List(ctx context.Context, projectID uuid.UUID, limit, offset int32) ([]db.ChangeRequest, error) {
	return database.Queries(ctx, s.Database).ListChangeRequestsByProject(ctx, db.ListChangeRequestsByProjectParams{
		Limit:     limit,
		Offset:    offset,
		ProjectID: projectID,
	})
}

// createBaselineFromSnapshot 在事务内创建并冻结变更后的新基线版本。
func createBaselineFromSnapshot(ctx context.Context, pool *pgxpool.Pool, projectID, changeID uuid.UUID, requirements []views.RequirementView) (*db.ScopeBaseline, error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("开启变更基线事务失败: %w", err)
	}
	defer tx.Rollback(ctx)
	queries := db.New(tx)
	latest, err := queries.GetLatestBaseline(ctx, projectID)
	if err != nil {
		return nil, errs.NotFound("项目尚无需求基线", err)
	}
	snapshot, err := dbutil.JSONBytes(requirements)
	if err != nil {
		return nil, err
	}
	baseline, err := queries.CreateScopeBaseline(ctx, db.CreateScopeBaselineParams{
		ProjectID:        projectID,
		RequirementsJson: snapshot,
		Version:          latest.Version + 1,
	})
	if err != nil {
		return nil, fmt.Errorf("创建变更基线失败: %w", err)
	}
	frozen, err := queries.FreezeScopeBaseline(ctx, db.FreezeScopeBaselineParams{
		ID:              baseline.ID,
		ChangeRequestID: pgtype.UUID{Bytes: changeID, Valid: true},
	})
	if err != nil {
		return nil, fmt.Errorf("冻结变更基线失败: %w", err)
	}
	return &frozen, nil
}

// applyChanges 把需求变更应用到现有需求集合，生成拟议快照。
func applyChanges(current []views.RequirementView, changes []RequirementChange) []views.RequirementView {
	result := make([]views.RequirementView, 0, len(current)+len(changes))
	byCode := make(map[string]views.RequirementView, len(current))
	for _, item := range current {
		byCode[item.Code] = item
	}
	for _, item := range current {
		removed := false
		for _, change := range changes {
			if change.Operation == "remove" && change.Code == item.Code {
				removed = true
				break
			}
		}
		if !removed {
			result = append(result, item)
		}
	}
	for _, change := range changes {
		if change.Operation == "remove" {
			continue
		}
		item := views.RequirementView{
			AcceptanceCriteria: change.AcceptanceCriteria,
			Code:               change.Code,
			Description:        change.Description,
			ID:                 uuid.New(),
			Priority:           change.Priority,
			Status:             "pending",
			Title:              change.Title,
		}
		if existing, ok := byCode[change.Code]; ok {
			item.ID = existing.ID
		}
		result = append(result, item)
	}
	return result
}

// parseRequirements 从 JSONB 解析需求视图列表。
func parseRequirements(raw []byte) []views.RequirementView {
	items := make([]views.RequirementView, 0)
	_ = dbutil.FromJSONBytes(raw, &items)
	return items
}
