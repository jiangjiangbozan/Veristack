// Package change 的业务服务，负责变更申请发起与客户确认生成新基线。
package change

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"api/internal/notification"
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
	Database *pgxpool.Pool          // 数据库连接池
	Notifier notification.Publisher // 站内与邮件通知，可为空
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
		s.notifyDecision(ctx, decided, status)
		return decided, nil, nil
	}
	proposed := parseRequirements(changeRequest.RequirementsJson)
	baseline, err := createBaselineFromSnapshot(ctx, s.Database, changeRequest.ProjectID, changeID, proposed)
	if err != nil {
		return db.ChangeRequest{}, nil, err
	}
	s.notifyDecision(ctx, decided, status)
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

// notifyDecision 把变更审批结果通知给申请人，通知失败不影响审批结果。
func (s *Service) notifyDecision(ctx context.Context, changeRequest db.ChangeRequest, status string) {
	if s.Notifier == nil {
		return
	}
	title := "变更申请已批准"
	body := "你提交的变更申请已被批准，项目需求基线已生成新版本。"
	if status == statusRejected {
		title = "变更申请未通过"
		body = "你提交的变更申请未通过审批，请查看项目变更记录了解详情。"
	}
	s.Notifier.PublishUserEvent(ctx, changeRequest.RequesterID, notification.ProjectEvent{
		Body:      body,
		Link:      fmt.Sprintf("/projects/%s", changeRequest.ProjectID),
		ProjectID: changeRequest.ProjectID,
		SendEmail: true,
		Title:     title,
		Type:      "change.decided",
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
//
// modify 必须原位替换而不是追加，否则同一需求会在快照里出现两次；
// remove 后要重建索引，避免后续变更命中已失效的下标。
func applyChanges(current []views.RequirementView, changes []RequirementChange) []views.RequirementView {
	result := append([]views.RequirementView(nil), current...)
	indexByCode := make(map[string]int, len(result))
	for index, item := range result {
		indexByCode[item.Code] = index
	}
	for _, change := range changes {
		switch change.Operation {
		case "remove":
			index, found := indexByCode[change.Code]
			if !found {
				continue
			}
			result = append(result[:index], result[index+1:]...)
			indexByCode = rebuildCodeIndex(result)
		case "modify":
			item := requirementFromChange(change)
			if index, found := indexByCode[change.Code]; found {
				item.ID = result[index].ID
				result[index] = item
				continue
			}
			result = append(result, item)
			indexByCode[change.Code] = len(result) - 1
		default:
			item := requirementFromChange(change)
			result = append(result, item)
			indexByCode[change.Code] = len(result) - 1
		}
	}
	return result
}

// requirementFromChange 把变更输入转换为拟议需求条目。
func requirementFromChange(change RequirementChange) views.RequirementView {
	return views.RequirementView{
		AcceptanceCriteria: change.AcceptanceCriteria,
		Code:               change.Code,
		Description:        change.Description,
		ID:                 uuid.New(),
		Priority:           change.Priority,
		Status:             "pending",
		Title:              change.Title,
	}
}

// rebuildCodeIndex 在删除后重建需求编号到下标的关系。
func rebuildCodeIndex(requirements []views.RequirementView) map[string]int {
	indexByCode := make(map[string]int, len(requirements))
	for index, item := range requirements {
		indexByCode[item.Code] = index
	}
	return indexByCode
}

// parseRequirements 从 JSONB 解析需求视图列表。
func parseRequirements(raw []byte) []views.RequirementView {
	items := make([]views.RequirementView, 0)
	_ = dbutil.FromJSONBytes(raw, &items)
	return items
}
