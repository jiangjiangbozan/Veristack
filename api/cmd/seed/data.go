package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"api/internal/platform/datetime"
)

const requirementsPerProject = 3

var seedNamespace = uuid.MustParse("f47ac10b-58cc-4372-a567-0e02b2c3d479")

var (
	orgCities           = []string{"天津", "北京", "上海", "深圳", "杭州", "成都", "武汉", "西安", "南京", "苏州"}
	orgPrefixes         = []string{"星河", "云帆", "远山", "蓝海", "启明", "恒信", "鼎新", "知行", "万象", "青禾", "朗月", "嘉木", "长风", "新程", "博远", "锐思", "同舟", "华章", "承启", "光谷"}
	orgSuffixes         = []string{"数字科技", "信息技术", "软件服务", "智能系统", "数据科技"}
	projectDomains      = []string{"政务", "医疗", "教育", "制造", "零售", "物流", "金融", "能源", "交通", "园区"}
	projectModules      = []string{"服务门户", "数据中台", "审批系统", "结算平台", "巡检平台", "客服中心", "供应链系统", "风控平台", "调度中心", "资产管理系统"}
	projectPhases       = []string{"一期", "二期", "三期", "改版升级", "信创适配", "移动端建设", "体验优化", "标准化改造", "扩容工程", "运维提升"}
	projectDescriptions = []string{
		"覆盖需求调研、原型确认、开发联调、上线验收与运维交接的完整交付。",
		"围绕核心业务链路重构系统，优先保证数据一致性与操作可追溯。",
		"在现有系统上做增量改造，保持历史数据兼容并降低迁移风险。",
		"以移动端与多角色权限为重点，提升一线人员的使用效率。",
		"完成信创环境适配与性能压测，满足客户对稳定性的验收要求。",
	}
	featureNames   = []string{"登录与权限", "首页导航", "数据字典", "流程审批", "消息通知", "报表导出", "移动端适配", "检索筛选", "审计日志", "接口联调", "性能优化", "安全加固", "统一支付", "消息中心", "组织架构", "文件管理", "任务调度", "数据备份", "灰度发布", "监控告警"}
	featureAspects = []string{"需求澄清", "方案设计", "开发实现", "联调验证", "上线检查"}
	priorityNames  = []string{"high", "medium", "low"}
	milestoneNames = []string{"需求冻结", "首个可用版本", "联调完成", "UAT 验收", "正式上线", "项目结项", "数据迁移", "性能压测", "安全整改", "培训交付", "试运行", "终验"}
	changeReasons  = []string{
		"客户新增监管报送要求，需要补充审批留痕与导出字段。",
		"上线前压测发现峰值吞吐不足，需要调整缓存与索引策略。",
		"业务部门调整组织架构，权限模型需要同步扩展。",
		"第三方支付接口升级，需要适配新的签名与回调格式。",
		"移动端用户占比提升，需要补齐关键页面的小屏适配。",
		"验收标准补充数据一致性校验，需要增加对账任务。",
		"历史数据迁移发现字段缺失，需要补充清洗规则。",
		"安全扫描发现依赖漏洞，需要升级组件并回归。",
		"客户要求增加多语言入口，需要调整文案与路由。",
		"运维交接要求补充监控指标与告警阈值。",
		"业务规则变更导致结算口径调整，需要重算历史数据。",
		"页面交互评审提出简化流程，需要合并两个审批节点。",
	}
	changeTypes        = []string{"add", "modify", "remove"}
	ciProviders        = []string{"GitHub Actions", "GitLab CI", "Jenkins"}
	notificationTypes  = []string{"acceptance.result", "change.decision", "objection.reply"}
	notificationTitles = []string{"交付验收结果已更新", "变更申请已有审批结论", "客户异议已收到回复", "需求基线已冻结", "交付版本已提交验收"}
	surnames           = []string{"张", "王", "李", "赵", "陈", "刘", "杨", "黄", "周", "吴", "徐", "孙", "胡", "朱", "高", "林", "何", "郭", "马", "罗"}
	givenNames         = []string{"子轩", "雨桐", "浩然", "思远", "嘉宁", "明哲", "欣怡", "宇航", "若彤", "俊杰", "晨曦", "书瑶", "亦凡", "安然", "知远", "可欣", "泽宇", "语嫣", "承泽", "沐阳"}
)

// seedUser 保存生成用户所需的关联字段。
type seedUser struct {
	id             uuid.UUID
	organizationID uuid.UUID
	email          string
	role           string
	displayName    string
}

// seedRequirement 保存单个项目下的一条需求。
type seedRequirement struct {
	id          uuid.UUID
	code        string
	title       string
	description string
	priority    string
	criteria    []string
}

// seedState 汇总生成过程中需要复用的父实体标识。
type seedState struct {
	count            int
	mainOrganization uuid.UUID
	ownerID          uuid.UUID
	organizationIDs  []uuid.UUID
	users            []seedUser
	projectIDs       []uuid.UUID
	projectNames     []string
	clientNames      []string
	baselineIDs      []uuid.UUID
	requirementSets  [][]seedRequirement
	milestoneIDs     []uuid.UUID
	deliveryIDs      []uuid.UUID
	objectionIDs     []uuid.UUID
}

// newSeedState 初始化生成状态，并预分配各表的主键切片。
func newSeedState(count int, mainOrganization, ownerID uuid.UUID) *seedState {
	return &seedState{
		count:            count,
		mainOrganization: mainOrganization,
		ownerID:          ownerID,
		projectIDs:       make([]uuid.UUID, 0, count),
		projectNames:     make([]string, 0, count),
		clientNames:      make([]string, 0, count),
		baselineIDs:      make([]uuid.UUID, 0, count),
		requirementSets:  make([][]seedRequirement, 0, count),
		milestoneIDs:     make([]uuid.UUID, 0, count),
		deliveryIDs:      make([]uuid.UUID, 0, count),
		objectionIDs:     make([]uuid.UUID, 0, count),
	}
}

// seedAll 按外键依赖顺序执行全部写表步骤。
func seedAll(ctx context.Context, tx pgx.Tx, state *seedState) error {
	steps := []struct {
		name string
		run  func() error
	}{
		{"组织", func() error { return seedOrganizations(ctx, tx, state) }},
		{"用户", func() error { return seedUsers(ctx, tx, state) }},
		{"项目", func() error { return seedProjects(ctx, tx, state) }},
		{"项目成员", func() error { return seedProjectMembers(ctx, tx, state) }},
		{"基线", func() error { return seedBaselines(ctx, tx, state) }},
		{"需求", func() error { return seedRequirements(ctx, tx, state) }},
		{"里程碑", func() error { return seedMilestones(ctx, tx, state) }},
		{"变更申请", func() error { return seedChangeRequests(ctx, tx, state) }},
		{"交付版本", func() error { return seedDeliveries(ctx, tx, state) }},
		{"证据", func() error { return seedEvidenceItems(ctx, tx, state) }},
		{"审批", func() error { return seedApprovals(ctx, tx, state) }},
		{"异议", func() error { return seedObjections(ctx, tx, state) }},
		{"异议回复", func() error { return seedObjectionReplies(ctx, tx, state) }},
		{"审计", func() error { return seedAuditEvents(ctx, tx, state) }},
		{"分享令牌", func() error { return seedShareTokens(ctx, tx, state) }},
		{"Outbox", func() error { return seedOutboxTasks(ctx, tx, state) }},
		{"Webhook", func() error { return seedWebhooks(ctx, tx, state) }},
		{"CI 报告", func() error { return seedCIReports(ctx, tx, state) }},
		{"AI 建议", func() error { return seedAISuggestions(ctx, tx, state) }},
		{"验收裁定", func() error { return seedAcceptanceVerdicts(ctx, tx, state) }},
		{"验收身份", func() error { return seedAcceptanceIdentities(ctx, tx, state) }},
		{"邮箱验证码", func() error { return seedEmailCodes(ctx, tx, state) }},
		{"重置令牌", func() error { return seedPasswordResetTokens(ctx, tx, state) }},
		{"通知", func() error { return seedNotifications(ctx, tx, state) }},
	}
	for _, step := range steps {
		if err := step.run(); err != nil {
			return fmt.Errorf("%s: %w", step.name, err)
		}
	}
	return nil
}

// seedOrganizations 生成示范组织，名称由城市、字号、行业后缀组合而成。
func seedOrganizations(ctx context.Context, tx pgx.Tx, state *seedState) error {
	rows := make([][]any, 0, state.count)
	state.organizationIDs = make([]uuid.UUID, 0, state.count)
	for index := 0; index < state.count; index++ {
		id := seedID("organization", index)
		state.organizationIDs = append(state.organizationIDs, id)
		created := seedCreatedAt(index)
		rows = append(rows, []any{
			dbUUID(id), organizationName(index), created, created, nil,
		})
	}
	return copyRows(ctx, tx, "organizations", []string{"id", "name", "created_at", "updated_at", "deleted_at"}, rows)
}

// seedUsers 生成 1000 个用户，前 50 个归属当前演示组织，便于项目成员和通知直接使用。
func seedUsers(ctx context.Context, tx pgx.Tx, state *seedState) error {
	rows := make([][]any, 0, state.count)
	state.users = make([]seedUser, 0, state.count)
	for index := 0; index < state.count; index++ {
		id := seedID("user", index)
		organizationID := state.organizationIDs[index%len(state.organizationIDs)]
		role := "developer"
		if index < 2 {
			role = "customer"
		} else if index < 10 {
			role = "reviewer"
		}
		if index < 50 {
			organizationID = state.mainOrganization
		}
		email := fmt.Sprintf("demo.user%04d@veristack.local", index)
		displayName := surnames[index%len(surnames)] + givenNames[(index/len(surnames))%len(givenNames)]
		state.users = append(state.users, seedUser{id: id, organizationID: organizationID, email: email, role: role, displayName: displayName})
		created := seedCreatedAt(index)
		rows = append(rows, []any{
			dbUUID(id), dbUUID(organizationID), email, demoPasswordHash, displayName, role, "active", created, created, nil, created.Add(24 * time.Hour),
		})
	}
	return copyRows(ctx, tx, "users",
		[]string{"id", "organization_id", "email", "password_hash", "display_name", "role", "status", "created_at", "updated_at", "deleted_at", "email_verified_at"}, rows)
}

// seedProjects 生成 1000 个项目，全部放在当前演示组织下，确保登录后项目列表可见。
func seedProjects(ctx context.Context, tx pgx.Tx, state *seedState) error {
	rows := make([][]any, 0, state.count)
	state.projectIDs = make([]uuid.UUID, 0, state.count)
	state.projectNames = make([]string, 0, state.count)
	state.clientNames = make([]string, 0, state.count)
	for index := 0; index < state.count; index++ {
		id := seedID("project", index)
		name := projectName(index)
		client := organizationName(index)
		status := "active"
		archived := false
		if index%17 == 0 {
			status = "completed"
		} else if index%29 == 0 {
			status = "archived"
			archived = true
		}
		start := projectStart(index)
		end := start.AddDate(0, 3+(index%4)*2, (index%12)+5)
		created := seedCreatedAt(index)
		state.projectIDs = append(state.projectIDs, id)
		state.projectNames = append(state.projectNames, name)
		state.clientNames = append(state.clientNames, client)
		rows = append(rows, []any{
			dbUUID(id), dbUUID(state.mainOrganization), name, client,
			fmt.Sprintf("HT-2026-%04d", index+1), projectDescriptions[index%len(projectDescriptions)],
			pgDate(start), pgDate(end), status, archived, created, created, nil,
		})
	}
	return copyRows(ctx, tx, "projects",
		[]string{"id", "organization_id", "name", "client_name", "contract_no", "description", "start_date", "end_date", "status", "archived", "created_at", "updated_at", "deleted_at"}, rows)
}

// seedProjectMembers 为每个项目分配一名当前组织的开发者，保证成员列表可打开。
func seedProjectMembers(ctx context.Context, tx pgx.Tx, state *seedState) error {
	rows := make([][]any, 0, state.count)
	developers := state.developerUsers()
	for index := 0; index < state.count; index++ {
		created := seedCreatedAt(index).AddDate(0, 0, 1)
		rows = append(rows, []any{
			dbUUID(seedID("project-member", index)), dbUUID(state.projectIDs[index]), dbUUID(developers[index%len(developers)].id), "developer", created,
		})
	}
	return copyRows(ctx, tx, "project_members", []string{"id", "project_id", "user_id", "role", "created_at"}, rows)
}

// seedBaselines 为每个项目生成 v1 基线快照，项目内包含 3 条需求。
func seedBaselines(ctx context.Context, tx pgx.Tx, state *seedState) error {
	rows := make([][]any, 0, state.count)
	state.baselineIDs = make([]uuid.UUID, 0, state.count)
	state.requirementSets = make([][]seedRequirement, 0, state.count)
	for index := 0; index < state.count; index++ {
		baselineID := seedID("baseline", index)
		requirements := buildRequirements(index)
		state.baselineIDs = append(state.baselineIDs, baselineID)
		state.requirementSets = append(state.requirementSets, requirements)
		created := seedCreatedAt(index).AddDate(0, 0, 2)
		snapshot := requirementSnapshot(requirements)
		rows = append(rows, []any{
			dbUUID(baselineID), dbUUID(state.projectIDs[index]), 1, jsonBytes(snapshot), created, nil, created,
		})
	}
	return copyRows(ctx, tx, "scope_baselines",
		[]string{"id", "project_id", "version", "requirements_json", "frozen_at", "change_request_id", "created_at"}, rows)
}

// seedRequirements 为每个项目的 v1 基线写入 3 条需求。
func seedRequirements(ctx context.Context, tx pgx.Tx, state *seedState) error {
	rows := make([][]any, 0, state.count*requirementsPerProject)
	for index := 0; index < state.count; index++ {
		requirements := state.requirementSets[index]
		for _, requirement := range requirements {
			created := seedCreatedAt(index).AddDate(0, 0, 3)
			rows = append(rows, []any{
				dbUUID(requirement.id), dbUUID(state.baselineIDs[index]), requirement.code, requirement.title,
				requirement.description, requirement.priority, "pending", jsonBytes(requirement.criteria), created, created, nil,
			})
		}
	}
	return copyRows(ctx, tx, "requirements",
		[]string{"id", "baseline_id", "code", "title", "description", "priority", "status", "acceptance_criteria", "created_at", "updated_at", "deleted_at"}, rows)
}

// seedMilestones 为每个项目生成一个里程碑，关联该项目的全部需求。
func seedMilestones(ctx context.Context, tx pgx.Tx, state *seedState) error {
	rows := make([][]any, 0, state.count)
	state.milestoneIDs = make([]uuid.UUID, 0, state.count)
	for index := 0; index < state.count; index++ {
		milestoneID := seedID("milestone", index)
		state.milestoneIDs = append(state.milestoneIDs, milestoneID)
		status := "pending"
		if index%3 == 0 {
			status = "completed"
		}
		requirementIDs := make([]uuid.UUID, 0, requirementsPerProject)
		for _, requirement := range state.requirementSets[index] {
			requirementIDs = append(requirementIDs, requirement.id)
		}
		created := seedCreatedAt(index).AddDate(0, 0, 5)
		rows = append(rows, []any{
			dbUUID(milestoneID), dbUUID(state.projectIDs[index]), milestoneNames[index%len(milestoneNames)],
			pgDate(projectStart(index).AddDate(0, 1, 0)), status, jsonBytes(requirementIDs), created, created, nil,
		})
	}
	return copyRows(ctx, tx, "milestones",
		[]string{"id", "project_id", "name", "target_date", "status", "requirement_ids", "created_at", "updated_at", "deleted_at"}, rows)
}

// seedChangeRequests 为每个项目生成一条变更申请，状态在待审批、已批准与已驳回之间分布。
func seedChangeRequests(ctx context.Context, tx pgx.Tx, state *seedState) error {
	rows := make([][]any, 0, state.count)
	developers := state.developerUsers()
	for index := 0; index < state.count; index++ {
		status := "pending"
		var decidedBy any
		var decidedAt any
		if index%3 != 0 {
			status = "approved"
			if index%7 == 0 {
				status = "rejected"
			}
			decidedBy = dbUUID(state.ownerID)
			decidedAt = seedCreatedAt(index).AddDate(0, 0, 7)
		}
		created := seedCreatedAt(index).AddDate(0, 0, 4)
		rows = append(rows, []any{
			dbUUID(seedID("change", index)), dbUUID(state.projectIDs[index]), dbUUID(state.baselineIDs[index]),
			changeTypes[index%len(changeTypes)], changeReasons[index%len(changeReasons)],
			"影响范围：需求条目、验收条件、里程碑计划与交付清单，需要同步回归。",
			status, dbUUID(developers[index%len(developers)].id), decidedBy, decidedAt, created, created, nil,
			jsonBytes(requirementSnapshot(state.requirementSets[index])),
		})
	}
	return copyRows(ctx, tx, "change_requests",
		[]string{"id", "project_id", "baseline_id", "type", "reason", "impact", "status", "requester_id", "decided_by_id", "decided_at", "created_at", "updated_at", "deleted_at", "requirements_json"}, rows)
}

// seedDeliveries 为每个项目生成一个交付版本，并冻结包含 3 条需求的交付清单。
func seedDeliveries(ctx context.Context, tx pgx.Tx, state *seedState) error {
	rows := make([][]any, 0, state.count)
	state.deliveryIDs = make([]uuid.UUID, 0, state.count)
	for index := 0; index < state.count; index++ {
		deliveryID := seedID("delivery", index)
		state.deliveryIDs = append(state.deliveryIDs, deliveryID)
		status := deliveryStatus(index)
		gitRef := fmt.Sprintf("%040x", index+1)
		buildNo := fmt.Sprintf("build-%04d", index+1)
		manifest := deliveryManifest(state, index, gitRef)
		created := seedCreatedAt(index).AddDate(0, 0, 10)
		rows = append(rows, []any{
			dbUUID(deliveryID), dbUUID(state.projectIDs[index]), 1, 1,
			jsonBytes([]uuid.UUID{state.milestoneIDs[index]}), gitRef, buildNo, status,
			fingerprint(index), created, created, nil, jsonBytes(manifest),
		})
	}
	return copyRows(ctx, tx, "deliveries",
		[]string{"id", "project_id", "version", "baseline_version", "milestone_ids", "git_ref", "build_no", "status", "manifest_hash", "created_at", "updated_at", "deleted_at", "manifest_json"}, rows)
}

// seedEvidenceItems 为每个项目生成一条与首条需求关联的实现证据。
func seedEvidenceItems(ctx context.Context, tx pgx.Tx, state *seedState) error {
	rows := make([][]any, 0, state.count)
	for index := 0; index < state.count; index++ {
		requirement := state.requirementSets[index][index%requirementsPerProject]
		occurred := seedCreatedAt(index).AddDate(0, 0, 9)
		note := "CI 构建与回归通过，附件为测试报告摘要，提交记录可在 Git 仓库复核。"
		rows = append(rows, []any{
			dbUUID(seedID("evidence", index)), dbUUID(state.projectIDs[index]), dbUUID(state.deliveryIDs[index]),
			dbUUID(requirement.id), "commit", fmt.Sprintf("%s 实现证据", requirement.title),
			fmt.Sprintf("evidence/project-%04d/report.pdf", index+1), fingerprint(index + 1),
			"github", occurred, occurred, nil, note, fmt.Sprintf("commit-%040x", index+1),
		})
	}
	return copyRows(ctx, tx, "evidence_items",
		[]string{"id", "project_id", "delivery_id", "requirement_id", "type", "title", "file_key", "sha256", "source", "occurred_at", "created_at", "deleted_at", "note", "source_fingerprint"}, rows)
}

// seedApprovals 为每个交付生成一条客户总体审批结论。
func seedApprovals(ctx context.Context, tx pgx.Tx, state *seedState) error {
	rows := make([][]any, 0, state.count)
	customers := state.customerUsers()
	for index := 0; index < state.count; index++ {
		decision := "approved"
		if index%5 == 0 {
			decision = "rejected"
		}
		decided := seedCreatedAt(index).AddDate(0, 0, 12)
		rows = append(rows, []any{
			dbUUID(seedID("approval", index)), dbUUID(state.deliveryIDs[index]), dbUUID(customers[index%len(customers)].id),
			decision, "已按交付清单逐条核对，结论与验收会议记录一致。", decided, decided,
		})
	}
	return copyRows(ctx, tx, "approvals", []string{"id", "delivery_id", "customer_id", "decision", "comment", "decided_at", "created_at"}, rows)
}

// seedObjections 为每个交付生成一条客户异议，并保留部分已关闭状态。
func seedObjections(ctx context.Context, tx pgx.Tx, state *seedState) error {
	rows := make([][]any, 0, state.count)
	customers := state.customerUsers()
	state.objectionIDs = make([]uuid.UUID, 0, state.count)
	for index := 0; index < state.count; index++ {
		objectionID := seedID("objection", index)
		state.objectionIDs = append(state.objectionIDs, objectionID)
		status := "open"
		var resolvedAt any
		if index%4 == 0 {
			status = "resolved"
			resolvedAt = seedCreatedAt(index).AddDate(0, 0, 14)
		}
		created := seedCreatedAt(index).AddDate(0, 0, 11)
		rows = append(rows, []any{
			dbUUID(objectionID), dbUUID(state.deliveryIDs[index]), "REQ-001::0", dbUUID(customers[index%len(customers)].id),
			"验收时发现异常分支提示不清晰，希望补充错误码说明与操作指引。", status, resolvedAt, created,
		})
	}
	return copyRows(ctx, tx, "objections", []string{"id", "delivery_id", "criterion_id", "raised_by_id", "content", "status", "resolved_at", "created_at"}, rows)
}

// seedObjectionReplies 为每条异议写入开发者的首次回复。
func seedObjectionReplies(ctx context.Context, tx pgx.Tx, state *seedState) error {
	rows := make([][]any, 0, state.count)
	developers := state.developerUsers()
	for index := 0; index < state.count; index++ {
		created := seedCreatedAt(index).AddDate(0, 0, 13)
		rows = append(rows, []any{
			dbUUID(seedID("objection-reply", index)), dbUUID(state.objectionIDs[index]), dbUUID(developers[index%len(developers)].id),
			"已补充错误码说明与操作指引，并在下一版回归用例中加入该异常分支，请复核。", created,
		})
	}
	return copyRows(ctx, tx, "objection_replies", []string{"id", "objection_id", "author_id", "content", "created_at"}, rows)
}

// seedAuditEvents 写入项目、变更与交付相关的只追加审计事件。
func seedAuditEvents(ctx context.Context, tx pgx.Tx, state *seedState) error {
	rows := make([][]any, 0, state.count)
	users := state.userIDs()
	for index := 0; index < state.count; index++ {
		objectType := "project"
		objectID := state.projectIDs[index]
		action := "project.update"
		if index%3 == 1 {
			objectType = "delivery"
			objectID = state.deliveryIDs[index]
			action = "delivery.submit"
		} else if index%3 == 2 {
			objectType = "change_request"
			objectID = seedID("change", index)
			action = "change.decide"
		}
		rows = append(rows, []any{
			dbUUID(seedID("audit", index)), dbUUID(users[index%len(users)]), action, objectType, dbUUID(objectID),
			jsonBytes(map[string]any{"status": "before"}), jsonBytes(map[string]any{"status": "after"}), fmt.Sprintf("10.12.%d.%d", index%255, (index+7)%255),
			seedCreatedAt(index).AddDate(0, 0, 15),
		})
	}
	return copyRows(ctx, tx, "audit_events",
		[]string{"id", "actor_id", "action", "object_type", "object_id", "before", "after", "ip", "occurred_at"}, rows)
}

// seedShareTokens 为每个交付生成分享令牌摘要。
func seedShareTokens(ctx context.Context, tx pgx.Tx, state *seedState) error {
	rows := make([][]any, 0, state.count)
	for index := 0; index < state.count; index++ {
		created := seedCreatedAt(index).AddDate(0, 0, 11)
		var revokedAt any
		if index%6 == 0 {
			revokedAt = created.AddDate(0, 0, 4)
		}
		rows = append(rows, []any{
			dbUUID(seedID("share-token", index)), dbUUID(state.deliveryIDs[index]), fingerprint(index + 2),
			created.AddDate(0, 1, 0), revokedAt, created,
		})
	}
	return copyRows(ctx, tx, "share_tokens", []string{"id", "delivery_id", "token_hash", "expires_at", "revoked_at", "created_at"}, rows)
}

// seedOutboxTasks 生成已完成的历史任务，避免 Worker 启动后继续处理演示任务。
func seedOutboxTasks(ctx context.Context, tx pgx.Tx, state *seedState) error {
	rows := make([][]any, 0, state.count)
	created := seedCreatedAt(0)
	for index := 0; index < state.count; index++ {
		rows = append(rows, []any{
			dbUUID(seedID("outbox", index)), "seed.notification",
			jsonBytes(map[string]any{"kind": "demo", "projectId": state.projectIDs[index].String()}),
			"done", 1, created.Add(time.Minute), created, dbUUID(state.projectIDs[index]),
			dbUUID(state.deliveryIDs[index]), "", nil,
		})
	}
	return copyRows(ctx, tx, "outbox_tasks",
		[]string{"id", "event_type", "payload", "status", "attempts", "next_run_at", "created_at", "project_id", "delivery_id", "dedupe_key", "last_error"}, rows)
}

// seedWebhooks 生成已处理完成的第三方推送事件，用于集成页面的历史列表。
func seedWebhooks(ctx context.Context, tx pgx.Tx, state *seedState) error {
	rows := make([][]any, 0, state.count)
	for index := 0; index < state.count; index++ {
		received := seedCreatedAt(index).AddDate(0, 0, 8)
		processed := received.Add(2 * time.Minute)
		rows = append(rows, []any{
			dbUUID(seedID("webhook", index)), ciProviders[index%len(ciProviders)], "push",
			dbUUID(state.projectIDs[index]), dbUUID(state.deliveryIDs[index]), "sha256=" + fingerprint(index+3),
			jsonBytes(map[string]any{
				"ref":        "refs/heads/main",
				"after":      fmt.Sprintf("%040x", index+1),
				"repository": map[string]any{"name": "veristack-demo", "private": true},
			}), "processed", nil, received, processed, nil,
		})
	}
	return copyRows(ctx, tx, "webhooks",
		[]string{"id", "provider", "event_type", "project_id", "delivery_id", "signature", "payload", "status", "error_message", "received_at", "processed_at", "deleted_at"}, rows)
}

// seedCIReports 为每个项目生成一份测试报告汇总。
func seedCIReports(ctx context.Context, tx pgx.Tx, state *seedState) error {
	rows := make([][]any, 0, state.count)
	for index := 0; index < state.count; index++ {
		total := 120 + index%180
		failed := index % 4
		skipped := index % 3
		rows = append(rows, []any{
			dbUUID(seedID("ci-report", index)), dbUUID(state.projectIDs[index]), dbUUID(state.deliveryIDs[index]),
			ciProviders[index%len(ciProviders)], fmt.Sprintf("build-%04d", index+1), fmt.Sprintf("%040x", index+1),
			total, total - failed - skipped, failed, skipped, int64((index + 1) * 1200),
			jsonBytes(map[string]any{"framework": "JUnit", "suites": []string{"unit", "integration", "contract"}}),
			seedCreatedAt(index).AddDate(0, 0, 9), nil,
		})
	}
	return copyRows(ctx, tx, "ci_reports",
		[]string{"id", "project_id", "delivery_id", "provider", "build_no", "git_ref", "total_count", "passed_count", "failed_count", "skipped_count", "duration_ms", "summary", "created_at", "deleted_at"}, rows)
}

// seedAISuggestions 生成需求拆解与验收建议草稿，状态包含待采纳与已采纳。
func seedAISuggestions(ctx context.Context, tx pgx.Tx, state *seedState) error {
	rows := make([][]any, 0, state.count)
	for index := 0; index < state.count; index++ {
		status := "draft"
		var appliedAt any
		if index%5 == 0 {
			status = "adopted"
			appliedAt = seedCreatedAt(index).AddDate(0, 0, 6)
		} else if index%11 == 0 {
			status = "ignored"
		}
		created := seedCreatedAt(index).AddDate(0, 0, 3)
		rows = append(rows, []any{
			dbUUID(seedID("ai-suggestion", index)), dbUUID(state.projectIDs[index]), []string{"requirement", "acceptance"}[index%2], status,
			jsonBytes(map[string]any{
				"summary": fmt.Sprintf("建议将 %s 拆分为 3 个可验收的需求点", state.projectNames[index]),
				"points":  []string{"权限边界", "异常回滚", "审计留痕"},
			}), "doubao-seed-1.6", nil, created, appliedAt, nil,
		})
	}
	return copyRows(ctx, tx, "ai_suggestions",
		[]string{"id", "project_id", "kind", "status", "content", "model", "error_message", "created_at", "applied_at", "deleted_at"}, rows)
}

// seedAcceptanceVerdicts 为每个交付写入首条验收条件的裁定。
func seedAcceptanceVerdicts(ctx context.Context, tx pgx.Tx, state *seedState) error {
	rows := make([][]any, 0, state.count)
	customers := state.customerUsers()
	for index := 0; index < state.count; index++ {
		requirement := state.requirementSets[index][0]
		verdict := "passed"
		if index%9 == 0 {
			verdict = "failed"
		}
		decided := seedCreatedAt(index).AddDate(0, 0, 13)
		rows = append(rows, []any{
			dbUUID(seedID("acceptance-verdict", index)), dbUUID(state.deliveryIDs[index]), requirement.code + "::0",
			requirement.code, requirement.title, verdict, "按验收条件逐条核对，结论与验收记录一致。",
			dbUUID(customers[index%len(customers)].id), decided, decided,
		})
	}
	return copyRows(ctx, tx, "acceptance_verdicts",
		[]string{"id", "delivery_id", "criterion_key", "requirement_code", "requirement_title", "verdict", "comment", "decided_by_id", "decided_at", "created_at"}, rows)
}

// seedAcceptanceIdentities 为每个交付写入已验真的客户邮箱身份。
func seedAcceptanceIdentities(ctx context.Context, tx pgx.Tx, state *seedState) error {
	rows := make([][]any, 0, state.count)
	customers := state.customerUsers()
	for index := 0; index < state.count; index++ {
		customer := customers[index%len(customers)]
		created := seedCreatedAt(index).AddDate(0, 0, 10)
		rows = append(rows, []any{
			dbUUID(seedID("acceptance-identity", index)), dbUUID(state.deliveryIDs[index]), customer.email,
			true, dbUUID(customer.id), created.Add(5 * time.Minute), created,
		})
	}
	return copyRows(ctx, tx, "acceptance_identities", []string{"id", "delivery_id", "email", "verified", "user_id", "verified_at", "created_at"}, rows)
}

// seedEmailCodes 生成已使用或已过期的邮箱验证码摘要。
func seedEmailCodes(ctx context.Context, tx pgx.Tx, state *seedState) error {
	rows := make([][]any, 0, state.count)
	customers := state.customerUsers()
	for index := 0; index < state.count; index++ {
		customer := customers[index%len(customers)]
		created := seedCreatedAt(index).AddDate(0, 0, 9)
		var consumedAt any
		if index%2 == 0 {
			consumedAt = created.Add(8 * time.Minute)
		}
		rows = append(rows, []any{
			dbUUID(seedID("email-code", index)), customer.email, fingerprint(index + 4), index % 3,
			created.Add(30 * time.Minute), consumedAt, created,
		})
	}
	return copyRows(ctx, tx, "email_verification_codes", []string{"id", "email", "code_hash", "attempts", "expires_at", "consumed_at", "created_at"}, rows)
}

// seedPasswordResetTokens 生成一次性密码重置令牌摘要。
func seedPasswordResetTokens(ctx context.Context, tx pgx.Tx, state *seedState) error {
	rows := make([][]any, 0, state.count)
	users := state.userIDs()
	for index := 0; index < state.count; index++ {
		created := seedCreatedAt(index).AddDate(0, 0, 7)
		var consumedAt any
		if index%2 == 0 {
			consumedAt = created.Add(20 * time.Minute)
		}
		rows = append(rows, []any{
			dbUUID(seedID("password-reset", index)), dbUUID(users[index%len(users)]), fingerprint(index + 5),
			created.Add(time.Hour), consumedAt, created,
		})
	}
	return copyRows(ctx, tx, "password_reset_tokens", []string{"id", "user_id", "token_hash", "expires_at", "consumed_at", "created_at"}, rows)
}

// seedNotifications 为当前组织的用户生成站内通知，覆盖验收、变更与异议回复。
func seedNotifications(ctx context.Context, tx pgx.Tx, state *seedState) error {
	rows := make([][]any, 0, state.count)
	mainUsers := state.mainUsers()
	for index := 0; index < state.count; index++ {
		recipient := mainUsers[index%len(mainUsers)]
		created := seedCreatedAt(index).AddDate(0, 0, 12)
		var readAt any
		var emailedAt any
		if index%3 != 0 {
			readAt = created.Add(10 * time.Minute)
		}
		if index%2 == 0 {
			emailedAt = created.Add(time.Minute)
		}
		rows = append(rows, []any{
			dbUUID(seedID("notification", index)), dbUUID(state.mainOrganization), dbUUID(recipient.id),
			notificationTypes[index%len(notificationTypes)], notificationTitles[index%len(notificationTitles)],
			fmt.Sprintf("项目「%s」有新的交付动态，请进入详情查看并继续处理。", state.projectNames[index]),
			fmt.Sprintf("/projects/%s", state.projectIDs[index]), fmt.Sprintf("seed-notification-%d", index),
			readAt, emailedAt, created,
		})
	}
	return copyRows(ctx, tx, "notifications",
		[]string{"id", "organization_id", "recipient_id", "type", "title", "body", "link", "dedupe_key", "read_at", "emailed_at", "created_at"}, rows)
}

// userIDs 返回全部生成用户的标识。
func (s *seedState) userIDs() []uuid.UUID {
	result := make([]uuid.UUID, 0, len(s.users))
	for _, user := range s.users {
		result = append(result, user.id)
	}
	return result
}

// mainUsers 返回当前演示组织内的用户。
func (s *seedState) mainUsers() []seedUser {
	result := make([]seedUser, 0, 50)
	for _, user := range s.users {
		if user.organizationID == s.mainOrganization {
			result = append(result, user)
		}
	}
	return result
}

// developerUsers 返回当前演示组织内的开发者。
func (s *seedState) developerUsers() []seedUser {
	result := make([]seedUser, 0, 40)
	for _, user := range s.users {
		if user.organizationID == s.mainOrganization && user.role == "developer" {
			result = append(result, user)
		}
	}
	return result
}

// customerUsers 返回全部客户用户。
func (s *seedState) customerUsers() []seedUser {
	result := make([]seedUser, 0, 2)
	for _, user := range s.users {
		if user.role == "customer" {
			result = append(result, user)
		}
	}
	return result
}

// buildRequirements 按项目序号生成 3 条可验收的需求。
func buildRequirements(projectIndex int) []seedRequirement {
	result := make([]seedRequirement, 0, requirementsPerProject)
	for offset := 0; offset < requirementsPerProject; offset++ {
		feature := featureNames[(projectIndex+offset)%len(featureNames)]
		aspect := featureAspects[(projectIndex+offset)%len(featureAspects)]
		code := fmt.Sprintf("REQ-%03d", offset+1)
		result = append(result, seedRequirement{
			id:          seedID(fmt.Sprintf("requirement-%d", offset), projectIndex),
			code:        code,
			title:       feature,
			description: fmt.Sprintf("完成%s的%s，覆盖正常流程、异常分支与回滚路径。", feature, aspect),
			priority:    priorityNames[(projectIndex+offset)%len(priorityNames)],
			criteria: []string{
				fmt.Sprintf("%s在测试环境完成%s，并通过回归用例。", feature, aspect),
				"关键路径无 P0/P1 缺陷，接口 P95 响应时间小于 500ms。",
			},
		})
	}
	return result
}

// requirementSnapshot 把需求转换为基线快照 JSON 结构，字段名与 views.RequirementView 对齐。
func requirementSnapshot(requirements []seedRequirement) []map[string]any {
	result := make([]map[string]any, 0, len(requirements))
	for _, requirement := range requirements {
		result = append(result, map[string]any{
			"acceptanceCriteria": requirement.criteria,
			"code":               requirement.code,
			"description":        requirement.description,
			"id":                 requirement.id.String(),
			"priority":           requirement.priority,
			"status":             "pending",
			"title":              requirement.title,
		})
	}
	return result
}

// deliveryManifest 构造交付清单快照，criteria 键与验收裁定使用的稳定键一致。
func deliveryManifest(state *seedState, index int, gitRef string) map[string]any {
	requirements := state.requirementSets[index]
	manifestRequirements := make([]map[string]any, 0, len(requirements))
	criteria := make([]map[string]any, 0, len(requirements)*2)
	for _, requirement := range requirements {
		manifestCriteria := make([]map[string]any, 0, len(requirement.criteria))
		for criterionIndex, criterion := range requirement.criteria {
			key := fmt.Sprintf("%s::%d", requirement.code, criterionIndex)
			manifestCriteria = append(manifestCriteria, map[string]any{
				"description":     criterion,
				"key":             key,
				"requirementCode": requirement.code,
			})
			criteria = append(criteria, map[string]any{
				"description":     criterion,
				"key":             key,
				"requirementCode": requirement.code,
			})
		}
		manifestRequirements = append(manifestRequirements, map[string]any{
			"code":        requirement.code,
			"criteria":    manifestCriteria,
			"description": requirement.description,
			"priority":    requirement.priority,
			"title":       requirement.title,
		})
	}
	return map[string]any{
		"baselineVersion": 1,
		"criteria":        criteria,
		"gitRef":          gitRef,
		"projectName":     state.projectNames[index],
		"requirements":    manifestRequirements,
		"version":         1,
	}
}

// organizationName 根据序号组合一个可读的公司名称。
func organizationName(index int) string {
	return orgCities[index%len(orgCities)] + orgPrefixes[(index/len(orgCities))%len(orgPrefixes)] +
		orgSuffixes[(index/(len(orgCities)*len(orgPrefixes)))%len(orgSuffixes)] + "有限公司"
}

// projectName 根据序号组合一个可读的项目名称。
func projectName(index int) string {
	return orgCities[index%len(orgCities)] + projectDomains[(index/len(orgCities))%len(projectDomains)] +
		projectModules[(index/(len(orgCities)*len(projectDomains)))%len(projectModules)] + "·" +
		projectPhases[(index/(len(orgCities)*len(projectDomains)*len(projectModules)))%len(projectPhases)]
}

// projectStart 生成分布在不同年份的项目开始日期。
func projectStart(index int) time.Time {
	location := datetime.Shanghai()
	return time.Date(2024+index%4, time.Month(1+index%12), 1+index%20, 9, 0, 0, 0, location)
}

// deliveryStatus 让交付状态在草稿、已提交、已通过与已驳回之间分布。
func deliveryStatus(index int) string {
	switch index % 5 {
	case 0:
		return "draft"
	case 1, 2:
		return "submitted"
	case 3:
		return "accepted"
	default:
		return "rejected"
	}
}

// seedCreatedAt 生成分散在近两年的创建时间。
func seedCreatedAt(index int) time.Time {
	return datetime.Now().AddDate(0, -(index % 24), -(index % 27)).Add(-time.Duration(index%12) * time.Hour)
}

// seedID 用固定命名空间生成可重复的 UUID，便于失败后整体回滚重跑。
func seedID(kind string, index int) uuid.UUID {
	return uuid.NewSHA1(seedNamespace, []byte(fmt.Sprintf("%s:%d", kind, index)))
}

// dbUUID 把 UUID 转为 pgx 原生类型，避免 CopyFrom 推断类型失败。
func dbUUID(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: id, Valid: true}
}

// pgDate 把时间转为 PostgreSQL date。
func pgDate(value time.Time) pgtype.Date {
	return pgtype.Date{Time: value, Valid: true}
}

// jsonBytes 把任意值编码为 JSONB 字节。
func jsonBytes(value any) []byte {
	encoded, err := json.Marshal(value)
	if err != nil {
		panic(fmt.Sprintf("序列化演示数据失败: %v", err))
	}
	return encoded
}

// fingerprint 生成 64 位十六进制摘要样式字符串。
func fingerprint(seed int) string {
	return fmt.Sprintf("%064x", seed)
}

// copyRows 批量写入一张表，失败时带上表名便于定位。
func copyRows(ctx context.Context, tx pgx.Tx, table string, columns []string, rows [][]any) error {
	_, err := tx.CopyFrom(ctx, pgx.Identifier{table}, columns, pgx.CopyFromRows(rows))
	if err != nil {
		return fmt.Errorf("写入 %s 失败: %w", table, err)
	}
	return nil
}

// demoPasswordHash 是演示用户的占位密码摘要，不用于真实登录。
var demoPasswordHash = strings.Repeat("demo-password-hash-", 4) + "demo"
