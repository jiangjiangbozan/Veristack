-- +goose Up
-- 阶段三：外部集成、AI 建议与 CI 报告导入所需的表结构。

-- 接收到的第三方 Webhook 原始事件，先落库再异步处理，保证不丢事件。
CREATE TABLE webhooks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider TEXT NOT NULL,
    event_type TEXT NOT NULL,
    project_id UUID REFERENCES projects (id),
    delivery_id UUID REFERENCES deliveries (id),
    signature TEXT,
    payload JSONB NOT NULL,
    status TEXT NOT NULL DEFAULT 'received',
    error_message TEXT,
    received_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    processed_at TIMESTAMPTZ,
    deleted_at TIMESTAMPTZ
);
CREATE INDEX webhooks_pending_idx ON webhooks (received_at DESC) WHERE status = 'received' AND deleted_at IS NULL;
CREATE INDEX webhooks_project_idx ON webhooks (project_id, received_at DESC) WHERE deleted_at IS NULL;

-- 从 CI 平台导入的测试报告汇总，作为交付证据的来源之一。
CREATE TABLE ci_reports (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects (id),
    delivery_id UUID REFERENCES deliveries (id),
    provider TEXT NOT NULL,
    build_no TEXT,
    git_ref TEXT,
    total_count INTEGER NOT NULL DEFAULT 0,
    passed_count INTEGER NOT NULL DEFAULT 0,
    failed_count INTEGER NOT NULL DEFAULT 0,
    skipped_count INTEGER NOT NULL DEFAULT 0,
    duration_ms BIGINT,
    summary JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);
CREATE INDEX ci_reports_delivery_idx ON ci_reports (delivery_id, created_at DESC) WHERE deleted_at IS NULL;

-- AI 生成的建议一律先落为草稿，经人工确认后才会影响正式基线或交付。
CREATE TABLE ai_suggestions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects (id),
    kind TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'draft',
    content JSONB NOT NULL,
    model TEXT,
    error_message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    applied_at TIMESTAMPTZ,
    deleted_at TIMESTAMPTZ
);
CREATE INDEX ai_suggestions_project_idx ON ai_suggestions (project_id, kind, created_at DESC) WHERE deleted_at IS NULL;

-- 任务队列补充上下文，便于处理任务时定位项目与交付。
ALTER TABLE outbox_tasks ADD COLUMN project_id UUID REFERENCES projects (id);
ALTER TABLE outbox_tasks ADD COLUMN delivery_id UUID REFERENCES deliveries (id);

-- +goose Down
ALTER TABLE outbox_tasks DROP COLUMN IF EXISTS delivery_id;
ALTER TABLE outbox_tasks DROP COLUMN IF EXISTS project_id;
DROP TABLE IF EXISTS ai_suggestions;
DROP TABLE IF EXISTS ci_reports;
DROP TABLE IF EXISTS webhooks;
