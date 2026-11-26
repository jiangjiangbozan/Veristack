// Package project 的业务服务，负责项目增删改查、成员维护与概览统计。
package project

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

const roleDeveloper = "developer"

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
	startDate, err := dbutil.DateFromString(req.StartDate)
	if err != nil {
		return db.Project{}, errs.BadRequest(err.Error(), err)
	}
	endDate, err := dbutil.DateFromString(req.EndDate)
	if err != nil {
		return db.Project{}, errs.BadRequest(err.Error(), err)
	}
	queries := database.Queries(ctx, s.Database)
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
