// Package project 的业务服务，负责项目增删改查、成员维护与概览统计。
package project

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
	"api/internal/scope"
)

const (
	roleDeveloper          = "developer"
	importBaselineVersion  = 1
	importRequirementLimit = 500
	importMilestoneLimit   = 100
)

// Service 提供项目相关的业务方法。
type Service struct {
	Database *pgxpool.Pool // 数据库连接池
}

// AddMember 把组织内用户添加为项目成员。
func (s *Service) AddMember(ctx context.Context, projectID uuid.UUID, req AddMemberRequest, organizationID uuid.UUID) (db.User, error) {
	queries := database.Queries(ctx, s.Database)
	user, err := queries.GetUserByEmail(ctx, req.Email)
	if err != nil || user.OrganizationID != organizationID {
		return db.User{}, errs.NotFound("用户不存在", err)
	}
	_, err = queries.CreateProjectMember(ctx, db.CreateProjectMemberParams{
		ProjectID: projectID,
		Role:      req.Role,
		UserID:    user.ID,
	})
	if err != nil {
		return db.User{}, fmt.Errorf("添加项目成员失败: %w", err)
	}
	return user, nil
}

// CountByOrganization 统计组织下项目总数。
func (s *Service) CountByOrganization(ctx context.Context, organizationID uuid.UUID) (int64, error) {
	return database.Queries(ctx, s.Database).CountProjects(ctx, organizationID)
}

// Create 在项目所属组织下创建项目，并把创建人登记为 developer 成员。
func (s *Service) Create(ctx context.Context, organizationID uuid.UUID, actorID uuid.UUID, req CreateProjectRequest) (db.Project, error) {
	return createProjectRecord(ctx, database.Queries(ctx, s.Database), organizationID, actorID, req)
}

// createProjectRecord 在给定查询器下创建项目与创建人成员关系，供创建和导入复用。
func createProjectRecord(ctx context.Context, queries *db.Queries, organizationID, actorID uuid.UUID, req CreateProjectRequest) (db.Project, error) {
	startDate, err := dbutil.DateFromString(req.StartDate)
	if err != nil {
		return db.Project{}, errs.BadRequest(err.Error(), err)
	}
	endDate, err := dbutil.DateFromString(req.EndDate)
	if err != nil {
		return db.Project{}, errs.BadRequest(err.Error(), err)
	}
	project, err := queries.CreateProject(ctx, db.CreateProjectParams{
		ClientName:     req.ClientName,
		ContractNo:     dbutil.TextFromString(req.ContractNo),
		Description:    dbutil.TextFromString(req.Description),
		EndDate:        endDate,
		Name:           req.Name,
		OrganizationID: organizationID,
		StartDate:      startDate,
	})
	if err != nil {
		return db.Project{}, fmt.Errorf("创建项目失败: %w", err)
	}
	if _, err := queries.CreateProjectMember(ctx, db.CreateProjectMemberParams{
		ProjectID: project.ID,
		Role:      roleDeveloper,
		UserID:    actorID,
	}); err != nil {
		return db.Project{}, fmt.Errorf("登记项目成员失败: %w", err)
	}
	return project, nil
}

// Get 按 ID 读取项目，不存在时返回 404。
func (s *Service) Get(ctx context.Context, id uuid.UUID) (db.Project, error) {
	project, err := database.Queries(ctx, s.Database).GetProject(ctx, id)
	if err != nil {
		return db.Project{}, errs.NotFound("项目不存在", err)
	}
	return project, nil
}

// Import 在请求事务内导入已有项目、初始需求基线与里程碑。
//
// - 需求编号在本次导入内必须唯一；
// - 里程碑通过需求编号关联基线需求，未知编号会让整次导入回滚；
// - 项目、基线、需求、里程碑与审计事件共用请求级事务，避免半成品项目。
func (s *Service) Import(ctx context.Context, organizationID, actorID uuid.UUID, req ImportProjectRequest) (ImportProjectResponse, error) {
	if len(req.Requirements) > importRequirementLimit {
		return ImportProjectResponse{}, errs.BadRequest(fmt.Sprintf("单个项目最多导入 %d 条需求", importRequirementLimit), nil)
	}
	if len(req.Milestones) > importMilestoneLimit {
		return ImportProjectResponse{}, errs.BadRequest(fmt.Sprintf("单个项目最多导入 %d 个里程碑", importMilestoneLimit), nil)
	}
	requirementViews, requirementIDs, err := buildImportRequirements(req.Requirements)
	if err != nil {
		return ImportProjectResponse{}, err
	}
	milestoneDates, err := validateImportMilestones(req.Milestones, requirementIDs)
	if err != nil {
		return ImportProjectResponse{}, err
	}
	queries := database.Queries(ctx, s.Database)
	project, err := createProjectRecord(ctx, queries, organizationID, actorID, req.Project)
	if err != nil {
		return ImportProjectResponse{}, err
	}
	actualRequirementIDs, err := createImportBaseline(ctx, queries, project.ID, req.Requirements, requirementViews)
	if err != nil {
		return ImportProjectResponse{}, err
	}
	milestoneCount := 0
	for index, milestone := range req.Milestones {
		requirementIDsForMilestone := make([]uuid.UUID, 0, len(milestone.RequirementCodes))
		for _, code := range milestone.RequirementCodes {
			requirementIDsForMilestone = append(requirementIDsForMilestone, actualRequirementIDs[code])
		}
		requirementIDsJSON, err := dbutil.JSONBytes(requirementIDsForMilestone)
		if err != nil {
			return ImportProjectResponse{}, err
		}
		if _, err := queries.CreateMilestone(ctx, db.CreateMilestoneParams{
			Name:           milestone.Name,
			ProjectID:      project.ID,
			RequirementIds: requirementIDsJSON,
			TargetDate:     milestoneDates[index],
		}); err != nil {
			return ImportProjectResponse{}, fmt.Errorf("创建里程碑 %s 失败: %w", milestone.Name, err)
		}
		milestoneCount++
	}
	return ImportProjectResponse{
		MilestoneCount:   milestoneCount,
		Project:          views.FromProject(project),
		RequirementCount: len(req.Requirements),
	}, nil
}

// buildImportRequirements 生成基线快照，并建立需求编号到快照标识的映射。
func buildImportRequirements(inputs []scope.RequirementInput) ([]views.RequirementView, map[string]uuid.UUID, error) {
	viewsList := make([]views.RequirementView, 0, len(inputs))
	requirementIDs := make(map[string]uuid.UUID, len(inputs))
	for _, input := range inputs {
		if _, exists := requirementIDs[input.Code]; exists {
			return nil, nil, errs.BadRequest(fmt.Sprintf("需求编号 %s 重复", input.Code), nil)
		}
		id := uuid.New()
		requirementIDs[input.Code] = id
		viewsList = append(viewsList, views.RequirementView{
			AcceptanceCriteria: input.AcceptanceCriteria,
			Code:               input.Code,
			Description:        input.Description,
			ID:                 id,
			Priority:           input.Priority,
			Status:             "pending",
			Title:              input.Title,
		})
	}
	return viewsList, requirementIDs, nil
}

// createImportBaseline 创建首版需求基线并冻结，返回需求编号到数据库实际 ID 的映射。
func createImportBaseline(ctx context.Context, queries *db.Queries, projectID uuid.UUID, inputs []scope.RequirementInput, snapshotViews []views.RequirementView) (map[string]uuid.UUID, error) {
	if len(inputs) == 0 {
		return map[string]uuid.UUID{}, nil
	}
	snapshot, err := dbutil.JSONBytes(snapshotViews)
	if err != nil {
		return nil, err
	}
	baseline, err := queries.CreateScopeBaseline(ctx, db.CreateScopeBaselineParams{
		ProjectID:        projectID,
		RequirementsJson: snapshot,
		Version:          importBaselineVersion,
	})
	if err != nil {
		return nil, fmt.Errorf("创建初始基线失败: %w", err)
	}
	requirementIDs := make(map[string]uuid.UUID, len(inputs))
	for _, requirement := range inputs {
		criteria, err := dbutil.JSONBytes(requirement.AcceptanceCriteria)
		if err != nil {
			return nil, err
		}
		created, err := queries.CreateRequirement(ctx, db.CreateRequirementParams{
			AcceptanceCriteria: criteria,
			BaselineID:         baseline.ID,
			Code:               requirement.Code,
			Description:        dbutil.TextFromString(requirement.Description),
			Priority:           requirement.Priority,
		})
		if err != nil {
			return nil, fmt.Errorf("创建需求 %s 失败: %w", requirement.Code, err)
		}
		requirementIDs[requirement.Code] = created.ID
	}
	if _, err := queries.FreezeScopeBaseline(ctx, db.FreezeScopeBaselineParams{
		ID:              baseline.ID,
		ChangeRequestID: pgtype.UUID{},
	}); err != nil {
		return nil, fmt.Errorf("冻结初始基线失败: %w", err)
	}
	return requirementIDs, nil
}

// validateImportMilestones 校验里程碑日期与需求编号，返回可写入数据库的日期。
func validateImportMilestones(milestones []ImportMilestone, requirementIDs map[string]uuid.UUID) ([]pgtype.Date, error) {
	dates := make([]pgtype.Date, 0, len(milestones))
	for _, milestone := range milestones {
		date, err := dbutil.DateFromString(milestone.TargetDate)
		if err != nil {
			return nil, errs.BadRequest(err.Error(), err)
		}
		for _, code := range milestone.RequirementCodes {
			if _, exists := requirementIDs[code]; !exists {
				return nil, errs.BadRequest(fmt.Sprintf("里程碑 %s 引用了不存在的需求编号 %s", milestone.Name, code), nil)
			}
		}
		dates = append(dates, date)
	}
	return dates, nil
}

// List 分页列出组织下项目。
func (s *Service) List(ctx context.Context, organizationID uuid.UUID, limit, offset int32) ([]db.Project, error) {
	return database.Queries(ctx, s.Database).ListProjects(ctx, db.ListProjectsParams{
		Limit:          limit,
		Offset:         offset,
		OrganizationID: organizationID,
	})
}

// ListMembers 列出项目成员。
func (s *Service) ListMembers(ctx context.Context, projectID uuid.UUID) ([]db.ListProjectMembersRow, error) {
	return database.Queries(ctx, s.Database).ListProjectMembers(ctx, projectID)
}

// Overview 统计项目的基线、变更、里程碑与需求数量。
func (s *Service) Overview(ctx context.Context, projectID uuid.UUID) (ProjectOverviewResponse, error) {
	queries := database.Queries(ctx, s.Database)
	baselineCount, err := queries.CountScopeBaselines(ctx, projectID)
	if err != nil {
		return ProjectOverviewResponse{}, err
	}
	changeCount, err := queries.CountChangeRequestsByProject(ctx, projectID)
	if err != nil {
		return ProjectOverviewResponse{}, err
	}
	milestoneCount, err := queries.CountMilestonesByProject(ctx, projectID)
	if err != nil {
		return ProjectOverviewResponse{}, err
	}
	result := ProjectOverviewResponse{
		BaselineCount:      baselineCount,
		ChangeRequestCount: changeCount,
		MilestoneCount:     milestoneCount,
	}
	baseline, err := queries.GetLatestBaseline(ctx, projectID)
	if err == nil {
		result.RequirementCount, _ = queries.CountRequirementsByBaseline(ctx, baseline.ID)
	}
	return result, nil
}

// RemoveMember 从项目移除成员。
func (s *Service) RemoveMember(ctx context.Context, projectID, userID uuid.UUID) error {
	return database.Queries(ctx, s.Database).DeleteProjectMember(ctx, db.DeleteProjectMemberParams{
		ProjectID: projectID,
		UserID:    userID,
	})
}

// SoftDelete 软删除项目。
func (s *Service) SoftDelete(ctx context.Context, id uuid.UUID) error {
	return database.Queries(ctx, s.Database).SoftDeleteProject(ctx, id)
}

// Update 更新项目可编辑字段。
func (s *Service) Update(ctx context.Context, id uuid.UUID, req UpdateProjectRequest) (db.Project, error) {
	startDate, err := dbutil.DateFromString(req.StartDate)
	if err != nil {
		return db.Project{}, errs.BadRequest(err.Error(), err)
	}
	endDate, err := dbutil.DateFromString(req.EndDate)
	if err != nil {
		return db.Project{}, errs.BadRequest(err.Error(), err)
	}
	return database.Queries(ctx, s.Database).UpdateProject(ctx, db.UpdateProjectParams{
		Archived:    req.Archived,
		ClientName:  req.ClientName,
		ContractNo:  dbutil.TextFromString(req.ContractNo),
		Description: dbutil.TextFromString(req.Description),
		EndDate:     endDate,
		ID:          id,
		Name:        req.Name,
		StartDate:   startDate,
		Status:      req.Status,
	})
}
