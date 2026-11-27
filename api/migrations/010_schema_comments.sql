-- +goose Up
-- 为全部业务表与字段补充中文注释，便于通过 psql、数据库客户端和代码生成工具直接理解表结构。
-- 已应用的 001-009 迁移保持只读，本迁移只追加注释，不修改任何字段类型或约束。

COMMENT ON TABLE public.organizations IS '组织：租户级工作空间，隔离不同客户的数据';
COMMENT ON COLUMN public.organizations.id IS '主键';
COMMENT ON COLUMN public.organizations.name IS '组织名称';
COMMENT ON COLUMN public.organizations.created_at IS '创建时间（上海时区）';
COMMENT ON COLUMN public.organizations.updated_at IS '更新时间（上海时区）';
COMMENT ON COLUMN public.organizations.deleted_at IS '软删除时间；为空表示有效';

COMMENT ON TABLE public.users IS '用户：组织内的管理员、开发者、评审者与客户账号';
COMMENT ON COLUMN public.users.id IS '主键';
COMMENT ON COLUMN public.users.organization_id IS '所属组织 ID';
COMMENT ON COLUMN public.users.email IS '登录邮箱';
COMMENT ON COLUMN public.users.password_hash IS '密码摘要，不保存明文密码';
COMMENT ON COLUMN public.users.display_name IS '显示名称';
COMMENT ON COLUMN public.users.role IS '角色：owner/admin/developer/reviewer/customer';
COMMENT ON COLUMN public.users.status IS '账号状态：active/disabled';
COMMENT ON COLUMN public.users.created_at IS '创建时间（上海时区）';
COMMENT ON COLUMN public.users.updated_at IS '更新时间（上海时区）';
COMMENT ON COLUMN public.users.deleted_at IS '软删除时间；为空表示有效';
COMMENT ON COLUMN public.users.email_verified_at IS '邮箱验证通过时间；为空表示未验证';

COMMENT ON TABLE public.projects IS '项目：客户交付项目的基本信息与生命周期状态';
COMMENT ON COLUMN public.projects.id IS '主键';
COMMENT ON COLUMN public.projects.organization_id IS '所属组织 ID';
COMMENT ON COLUMN public.projects.name IS '项目名称';
COMMENT ON COLUMN public.projects.client_name IS '客户名称';
COMMENT ON COLUMN public.projects.contract_no IS '合同编号';
COMMENT ON COLUMN public.projects.description IS '项目描述与交付范围摘要';
COMMENT ON COLUMN public.projects.start_date IS '项目开始日期';
COMMENT ON COLUMN public.projects.end_date IS '项目结束日期';
COMMENT ON COLUMN public.projects.status IS '项目状态：active/completed/archived';
COMMENT ON COLUMN public.projects.archived IS '是否已归档';
COMMENT ON COLUMN public.projects.created_at IS '创建时间（上海时区）';
COMMENT ON COLUMN public.projects.updated_at IS '更新时间（上海时区）';
COMMENT ON COLUMN public.projects.deleted_at IS '软删除时间；为空表示有效';

COMMENT ON TABLE public.project_members IS '项目成员：项目内开发者与评审者的角色授权';
COMMENT ON COLUMN public.project_members.id IS '主键';
COMMENT ON COLUMN public.project_members.project_id IS '所属项目 ID';
COMMENT ON COLUMN public.project_members.user_id IS '成员用户 ID';
COMMENT ON COLUMN public.project_members.role IS '项目角色：developer/reviewer';
COMMENT ON COLUMN public.project_members.created_at IS '加入时间（上海时区）';

COMMENT ON TABLE public.scope_baselines IS '范围基线：项目某一版本冻结的需求快照，不可就地修改';
COMMENT ON COLUMN public.scope_baselines.id IS '主键';
COMMENT ON COLUMN public.scope_baselines.project_id IS '所属项目 ID';
COMMENT ON COLUMN public.scope_baselines.version IS '基线版本号，项目内从 1 开始递增';
COMMENT ON COLUMN public.scope_baselines.requirements_json IS '冻结的需求快照 JSON，保留当时的验收条件与稳定标识';
COMMENT ON COLUMN public.scope_baselines.frozen_at IS '冻结时间；为空表示仍在草稿中';
COMMENT ON COLUMN public.scope_baselines.change_request_id IS '来源变更申请 ID；首次基线为空';
COMMENT ON COLUMN public.scope_baselines.created_at IS '创建时间（上海时区）';

COMMENT ON TABLE public.requirements IS '需求条目：基线中冻结的单条需求及其验收条件';
COMMENT ON COLUMN public.requirements.id IS '主键';
COMMENT ON COLUMN public.requirements.baseline_id IS '所属基线 ID';
COMMENT ON COLUMN public.requirements.code IS '需求编号，同一基线内唯一';
COMMENT ON COLUMN public.requirements.title IS '需求标题';
COMMENT ON COLUMN public.requirements.description IS '需求详细描述';
COMMENT ON COLUMN public.requirements.priority IS '优先级：high/medium/low';
COMMENT ON COLUMN public.requirements.status IS '实现状态：pending/verified';
COMMENT ON COLUMN public.requirements.acceptance_criteria IS '验收条件 JSON 数组，逐条对应客户裁定项';
COMMENT ON COLUMN public.requirements.created_at IS '创建时间（上海时区）';
COMMENT ON COLUMN public.requirements.updated_at IS '更新时间（上海时区）';
COMMENT ON COLUMN public.requirements.deleted_at IS '软删除时间；为空表示有效';

COMMENT ON TABLE public.change_requests IS '变更申请：项目范围变更的原因、影响与审批结果';
COMMENT ON COLUMN public.change_requests.id IS '主键';
COMMENT ON COLUMN public.change_requests.project_id IS '所属项目 ID';
COMMENT ON COLUMN public.change_requests.baseline_id IS '变更所基于的基线 ID';
COMMENT ON COLUMN public.change_requests.type IS '变更类型：add/modify/remove';
COMMENT ON COLUMN public.change_requests.reason IS '变更原因';
COMMENT ON COLUMN public.change_requests.impact IS '变更影响说明';
COMMENT ON COLUMN public.change_requests.status IS '审批状态：pending/approved/rejected';
COMMENT ON COLUMN public.change_requests.requester_id IS '发起人用户 ID';
COMMENT ON COLUMN public.change_requests.decided_by_id IS '审批人用户 ID';
COMMENT ON COLUMN public.change_requests.decided_at IS '审批时间';
COMMENT ON COLUMN public.change_requests.created_at IS '创建时间（上海时区）';
COMMENT ON COLUMN public.change_requests.updated_at IS '更新时间（上海时区）';
COMMENT ON COLUMN public.change_requests.deleted_at IS '软删除时间；为空表示有效';
COMMENT ON COLUMN public.change_requests.requirements_json IS '拟议需求快照 JSON，审批通过后据此生成新基线';

COMMENT ON TABLE public.milestones IS '里程碑：项目阶段目标及其关联需求';
COMMENT ON COLUMN public.milestones.id IS '主键';
COMMENT ON COLUMN public.milestones.project_id IS '所属项目 ID';
COMMENT ON COLUMN public.milestones.name IS '里程碑名称';
COMMENT ON COLUMN public.milestones.target_date IS '目标完成日期';
COMMENT ON COLUMN public.milestones.status IS '状态：pending/completed';
COMMENT ON COLUMN public.milestones.requirement_ids IS '关联需求 ID 的 JSON 数组';
COMMENT ON COLUMN public.milestones.created_at IS '创建时间（上海时区）';
COMMENT ON COLUMN public.milestones.updated_at IS '更新时间（上海时区）';
COMMENT ON COLUMN public.milestones.deleted_at IS '软删除时间；为空表示有效';

COMMENT ON TABLE public.deliveries IS '交付版本：一次冻结并提交客户验收的交付清单';
COMMENT ON COLUMN public.deliveries.id IS '主键';
COMMENT ON COLUMN public.deliveries.project_id IS '所属项目 ID';
COMMENT ON COLUMN public.deliveries.version IS '交付版本号，项目内唯一且递增';
COMMENT ON COLUMN public.deliveries.baseline_version IS '对应需求基线版本号';
COMMENT ON COLUMN public.deliveries.milestone_ids IS '关联里程碑 ID 的 JSON 数组';
COMMENT ON COLUMN public.deliveries.git_ref IS '交付对应的提交哈希或分支';
COMMENT ON COLUMN public.deliveries.build_no IS '构建编号';
COMMENT ON COLUMN public.deliveries.status IS '状态：draft/submitted/accepted/rejected';
COMMENT ON COLUMN public.deliveries.manifest_hash IS '交付清单内容哈希，用于校验清单未被篡改';
COMMENT ON COLUMN public.deliveries.created_at IS '创建时间（上海时区）';
COMMENT ON COLUMN public.deliveries.updated_at IS '更新时间（上海时区）';
COMMENT ON COLUMN public.deliveries.deleted_at IS '软删除时间；为空表示有效';
COMMENT ON COLUMN public.deliveries.manifest_json IS '冻结的交付清单快照 JSON，后续补充证据只能关联追加';

COMMENT ON TABLE public.evidence_items IS '证据条目：需求实现与测试过程的文件、提交或人工说明证据';
COMMENT ON COLUMN public.evidence_items.id IS '主键';
COMMENT ON COLUMN public.evidence_items.project_id IS '所属项目 ID';
COMMENT ON COLUMN public.evidence_items.delivery_id IS '关联交付版本 ID';
COMMENT ON COLUMN public.evidence_items.requirement_id IS '关联需求条目 ID';
COMMENT ON COLUMN public.evidence_items.type IS '证据类型：file/commit/manual/ci';
COMMENT ON COLUMN public.evidence_items.title IS '证据标题';
COMMENT ON COLUMN public.evidence_items.file_key IS '附件在对象存储中的键';
COMMENT ON COLUMN public.evidence_items.sha256 IS '附件或内容摘要';
COMMENT ON COLUMN public.evidence_items.source IS '证据来源，例如 CI 平台或提交地址';
COMMENT ON COLUMN public.evidence_items.occurred_at IS '证据实际发生时间';
COMMENT ON COLUMN public.evidence_items.created_at IS '记录时间（上海时区）';
COMMENT ON COLUMN public.evidence_items.deleted_at IS '软删除时间；为空表示有效';
COMMENT ON COLUMN public.evidence_items.note IS '人工说明内容；无附件时由该字段承载证据';
COMMENT ON COLUMN public.evidence_items.source_fingerprint IS '来源指纹，例如提交哈希或事件唯一标识';

COMMENT ON TABLE public.approvals IS '交付审批：客户对交付版本的总体通过或驳回结论';
COMMENT ON COLUMN public.approvals.id IS '主键';
COMMENT ON COLUMN public.approvals.delivery_id IS '关联交付版本 ID';
COMMENT ON COLUMN public.approvals.customer_id IS '作出结论的客户用户 ID';
COMMENT ON COLUMN public.approvals.decision IS '审批结论：approved/rejected';
COMMENT ON COLUMN public.approvals.comment IS '审批意见';
COMMENT ON COLUMN public.approvals.decided_at IS '审批时间';
COMMENT ON COLUMN public.approvals.created_at IS '创建时间（上海时区）';

COMMENT ON TABLE public.objections IS '客户异议：客户对验收条件提出的不通过意见及闭环状态';
COMMENT ON COLUMN public.objections.id IS '主键';
COMMENT ON COLUMN public.objections.delivery_id IS '关联交付版本 ID';
COMMENT ON COLUMN public.objections.criterion_id IS '验收条件稳定键';
COMMENT ON COLUMN public.objections.raised_by_id IS '提出人用户 ID';
COMMENT ON COLUMN public.objections.content IS '异议内容';
COMMENT ON COLUMN public.objections.status IS '状态：open/resolved';
COMMENT ON COLUMN public.objections.resolved_at IS '异议关闭时间';
COMMENT ON COLUMN public.objections.created_at IS '提出时间（上海时区）';

COMMENT ON TABLE public.audit_events IS '审计事件：只追加记录所有写操作的操作人、对象与前后快照';
COMMENT ON COLUMN public.audit_events.id IS '主键';
COMMENT ON COLUMN public.audit_events.actor_id IS '操作人用户 ID；系统任务可为空';
COMMENT ON COLUMN public.audit_events.action IS '动作编码，例如 project.create';
COMMENT ON COLUMN public.audit_events.object_type IS '操作对象类型';
COMMENT ON COLUMN public.audit_events.object_id IS '操作对象 ID';
COMMENT ON COLUMN public.audit_events.before IS '变更前 JSON 快照';
COMMENT ON COLUMN public.audit_events.after IS '变更后 JSON 快照';
COMMENT ON COLUMN public.audit_events.ip IS '来源 IP';
COMMENT ON COLUMN public.audit_events.occurred_at IS '事件发生时间';

COMMENT ON TABLE public.share_tokens IS '分享令牌：交付验收链接的一次性访问凭证摘要';
COMMENT ON COLUMN public.share_tokens.id IS '主键';
COMMENT ON COLUMN public.share_tokens.delivery_id IS '关联交付版本 ID';
COMMENT ON COLUMN public.share_tokens.token_hash IS '访问令牌摘要，不保存明文';
COMMENT ON COLUMN public.share_tokens.expires_at IS '过期时间';
COMMENT ON COLUMN public.share_tokens.revoked_at IS '撤销时间；为空表示仍可使用';
COMMENT ON COLUMN public.share_tokens.created_at IS '创建时间（上海时区）';

COMMENT ON TABLE public.outbox_tasks IS 'Outbox 任务：事务内落库、异步执行的邮件与 Webhook 任务';
COMMENT ON COLUMN public.outbox_tasks.id IS '主键';
COMMENT ON COLUMN public.outbox_tasks.event_type IS '事件类型，例如 email.send';
COMMENT ON COLUMN public.outbox_tasks.payload IS '任务载荷 JSON';
COMMENT ON COLUMN public.outbox_tasks.status IS '状态：pending/processing/done/failed';
COMMENT ON COLUMN public.outbox_tasks.attempts IS '已尝试次数';
COMMENT ON COLUMN public.outbox_tasks.next_run_at IS '下次允许执行时间';
COMMENT ON COLUMN public.outbox_tasks.created_at IS '创建时间（上海时区）';
COMMENT ON COLUMN public.outbox_tasks.project_id IS '关联项目 ID';
COMMENT ON COLUMN public.outbox_tasks.delivery_id IS '关联交付版本 ID';
COMMENT ON COLUMN public.outbox_tasks.dedupe_key IS '幂等键，非空时同一事件类型内唯一';
COMMENT ON COLUMN public.outbox_tasks.last_error IS '最近一次失败原因';

COMMENT ON TABLE public.webhooks IS 'Webhook 事件：第三方平台推送的原始事件，先落库再异步处理';
COMMENT ON COLUMN public.webhooks.id IS '主键';
COMMENT ON COLUMN public.webhooks.provider IS '来源平台：github/gitlab/gitee';
COMMENT ON COLUMN public.webhooks.event_type IS '事件类型';
COMMENT ON COLUMN public.webhooks.project_id IS '关联项目 ID';
COMMENT ON COLUMN public.webhooks.delivery_id IS '关联交付版本 ID';
COMMENT ON COLUMN public.webhooks.signature IS '请求签名或令牌';
COMMENT ON COLUMN public.webhooks.payload IS '原始请求载荷 JSON，完整保留以便重放与排查';
COMMENT ON COLUMN public.webhooks.status IS '处理状态：received/processed/failed';
COMMENT ON COLUMN public.webhooks.error_message IS '处理失败原因';
COMMENT ON COLUMN public.webhooks.received_at IS '接收时间';
COMMENT ON COLUMN public.webhooks.processed_at IS '处理完成时间';
COMMENT ON COLUMN public.webhooks.deleted_at IS '软删除时间；为空表示有效';

COMMENT ON TABLE public.ci_reports IS 'CI 报告：从持续集成平台导入的构建与测试结果汇总';
COMMENT ON COLUMN public.ci_reports.id IS '主键';
COMMENT ON COLUMN public.ci_reports.project_id IS '所属项目 ID';
COMMENT ON COLUMN public.ci_reports.delivery_id IS '关联交付版本 ID';
COMMENT ON COLUMN public.ci_reports.provider IS 'CI 平台：GitHub Actions/GitLab CI/Jenkins';
COMMENT ON COLUMN public.ci_reports.build_no IS '构建编号';
COMMENT ON COLUMN public.ci_reports.git_ref IS '构建对应的提交哈希或分支';
COMMENT ON COLUMN public.ci_reports.total_count IS '测试用例总数';
COMMENT ON COLUMN public.ci_reports.passed_count IS '通过数';
COMMENT ON COLUMN public.ci_reports.failed_count IS '失败数';
COMMENT ON COLUMN public.ci_reports.skipped_count IS '跳过数';
COMMENT ON COLUMN public.ci_reports.duration_ms IS '构建耗时，单位毫秒';
COMMENT ON COLUMN public.ci_reports.summary IS '测试套件汇总 JSON';
COMMENT ON COLUMN public.ci_reports.created_at IS '导入时间（上海时区）';
COMMENT ON COLUMN public.ci_reports.deleted_at IS '软删除时间；为空表示有效';

COMMENT ON TABLE public.ai_suggestions IS 'AI 建议草稿：AI 生成的需求或验收建议，须人工采纳后才生效';
COMMENT ON COLUMN public.ai_suggestions.id IS '主键';
COMMENT ON COLUMN public.ai_suggestions.project_id IS '所属项目 ID';
COMMENT ON COLUMN public.ai_suggestions.kind IS '建议类型：requirement/acceptance/change';
COMMENT ON COLUMN public.ai_suggestions.status IS '状态：draft/adopted/ignored/failed';
COMMENT ON COLUMN public.ai_suggestions.content IS 'AI 建议内容 JSON';
COMMENT ON COLUMN public.ai_suggestions.model IS '生成建议的模型标识';
COMMENT ON COLUMN public.ai_suggestions.error_message IS '生成失败原因';
COMMENT ON COLUMN public.ai_suggestions.created_at IS '生成时间（上海时区）';
COMMENT ON COLUMN public.ai_suggestions.applied_at IS '人工采纳时间';
COMMENT ON COLUMN public.ai_suggestions.deleted_at IS '软删除时间；为空表示有效';

COMMENT ON TABLE public.acceptance_verdicts IS '验收裁定：客户对交付清单中每条验收条件的逐条结论';
COMMENT ON COLUMN public.acceptance_verdicts.id IS '主键';
COMMENT ON COLUMN public.acceptance_verdicts.delivery_id IS '关联交付版本 ID';
COMMENT ON COLUMN public.acceptance_verdicts.criterion_key IS '验收条件稳定键，同一交付内唯一';
COMMENT ON COLUMN public.acceptance_verdicts.requirement_code IS '需求编号';
COMMENT ON COLUMN public.acceptance_verdicts.requirement_title IS '需求标题';
COMMENT ON COLUMN public.acceptance_verdicts.verdict IS '裁定结果：passed/failed';
COMMENT ON COLUMN public.acceptance_verdicts.comment IS '裁定说明';
COMMENT ON COLUMN public.acceptance_verdicts.decided_by_id IS '裁定人用户 ID';
COMMENT ON COLUMN public.acceptance_verdicts.decided_at IS '裁定时间';
COMMENT ON COLUMN public.acceptance_verdicts.created_at IS '创建时间（上海时区）';

COMMENT ON TABLE public.objection_replies IS '异议回复：开发者对客户异议的答复记录';
COMMENT ON COLUMN public.objection_replies.id IS '主键';
COMMENT ON COLUMN public.objection_replies.objection_id IS '关联异议 ID';
COMMENT ON COLUMN public.objection_replies.author_id IS '回复人用户 ID';
COMMENT ON COLUMN public.objection_replies.content IS '回复内容';
COMMENT ON COLUMN public.objection_replies.created_at IS '回复时间（上海时区）';

COMMENT ON TABLE public.acceptance_identities IS '客户验收身份：记录交付分享页提交验收的邮箱及验证状态';
COMMENT ON COLUMN public.acceptance_identities.id IS '主键';
COMMENT ON COLUMN public.acceptance_identities.delivery_id IS '关联交付版本 ID';
COMMENT ON COLUMN public.acceptance_identities.email IS '验收人邮箱';
COMMENT ON COLUMN public.acceptance_identities.verified IS '邮箱是否已验证';
COMMENT ON COLUMN public.acceptance_identities.user_id IS '关联客户用户 ID；仅持分享链接时为空';
COMMENT ON COLUMN public.acceptance_identities.verified_at IS '邮箱验证通过时间';
COMMENT ON COLUMN public.acceptance_identities.created_at IS '创建时间（上海时区）';

COMMENT ON TABLE public.email_verification_codes IS '邮箱验证码：客户验收邮箱的一次性验证码摘要';
COMMENT ON COLUMN public.email_verification_codes.id IS '主键';
COMMENT ON COLUMN public.email_verification_codes.email IS '接收验证码的邮箱';
COMMENT ON COLUMN public.email_verification_codes.code_hash IS '验证码摘要，不保存明文';
COMMENT ON COLUMN public.email_verification_codes.attempts IS '已校验次数';
COMMENT ON COLUMN public.email_verification_codes.expires_at IS '过期时间';
COMMENT ON COLUMN public.email_verification_codes.consumed_at IS '使用时间；为空表示未使用';
COMMENT ON COLUMN public.email_verification_codes.created_at IS '创建时间（上海时区）';

COMMENT ON TABLE public.password_reset_tokens IS '密码重置令牌：只保存摘要的一次性密码重置凭证';
COMMENT ON COLUMN public.password_reset_tokens.id IS '主键';
COMMENT ON COLUMN public.password_reset_tokens.user_id IS '关联用户 ID';
COMMENT ON COLUMN public.password_reset_tokens.token_hash IS '令牌摘要，不保存明文';
COMMENT ON COLUMN public.password_reset_tokens.expires_at IS '过期时间';
COMMENT ON COLUMN public.password_reset_tokens.consumed_at IS '使用时间；为空表示未使用';
COMMENT ON COLUMN public.password_reset_tokens.created_at IS '创建时间（上海时区）';

COMMENT ON TABLE public.notifications IS '站内通知：按接收人保存的业务事件提醒与阅读状态';
COMMENT ON COLUMN public.notifications.id IS '主键';
COMMENT ON COLUMN public.notifications.organization_id IS '所属组织 ID';
COMMENT ON COLUMN public.notifications.recipient_id IS '接收人用户 ID';
COMMENT ON COLUMN public.notifications.type IS '事件类型，例如 acceptance.objection';
COMMENT ON COLUMN public.notifications.title IS '通知标题';
COMMENT ON COLUMN public.notifications.body IS '通知正文';
COMMENT ON COLUMN public.notifications.link IS '站内跳转链接，例如 /projects/<id>';
COMMENT ON COLUMN public.notifications.dedupe_key IS '幂等键，非空时同一接收人内唯一';
COMMENT ON COLUMN public.notifications.read_at IS '阅读时间；为空表示未读';
COMMENT ON COLUMN public.notifications.emailed_at IS '邮件发送时间';
COMMENT ON COLUMN public.notifications.created_at IS '创建时间（上海时区）';

COMMENT ON TABLE public.goose_db_version IS 'Goose 迁移版本记录：记录已应用的数据库迁移版本';
COMMENT ON COLUMN public.goose_db_version.id IS '自增主键';
COMMENT ON COLUMN public.goose_db_version.version_id IS '迁移版本号';
COMMENT ON COLUMN public.goose_db_version.is_applied IS '是否已应用';
COMMENT ON COLUMN public.goose_db_version.tstamp IS '应用时间';

-- +goose Down
-- 注释只影响数据库元数据，回退时清空本迁移写入的表与字段注释。
DO $$
DECLARE
    target_table TEXT;
    target_column TEXT;
BEGIN
    FOR target_table IN SELECT unnest(ARRAY[
        'acceptance_identities', 'acceptance_verdicts', 'ai_suggestions', 'approvals',
        'audit_events', 'change_requests', 'ci_reports', 'deliveries',
        'email_verification_codes', 'evidence_items', 'goose_db_version', 'milestones',
        'notifications', 'objection_replies', 'objections', 'organizations',
        'outbox_tasks', 'password_reset_tokens', 'project_members', 'projects',
        'requirements', 'scope_baselines', 'share_tokens', 'users', 'webhooks'
    ]) LOOP
        EXECUTE format('COMMENT ON TABLE public.%I IS NULL', target_table);
        FOR target_column IN
            SELECT column_name FROM information_schema.columns
            WHERE table_schema = 'public' AND table_name = target_table
        LOOP
            EXECUTE format('COMMENT ON COLUMN public.%I.%I IS NULL', target_table, target_column);
        END LOOP;
    END LOOP;
END $$;
