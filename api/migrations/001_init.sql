-- +goose Up
-- 建立 Veristack 全部核心业务表，字段类型、约束与索引按交付证据链的业务语义设计。

CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE organizations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);
CREATE INDEX organizations_active_idx ON organizations (name) WHERE deleted_at IS NULL;

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations (id),
    email TEXT NOT NULL,
    password_hash TEXT NOT NULL,
    display_name TEXT NOT NULL,
    role TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);
CREATE UNIQUE INDEX users_email_active_key ON users (email) WHERE deleted_at IS NULL;
CREATE INDEX users_organization_active_idx ON users (organization_id) WHERE deleted_at IS NULL;

CREATE TABLE projects (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations (id),
    name TEXT NOT NULL,
    client_name TEXT NOT NULL,
    contract_no TEXT,
    description TEXT,
    start_date DATE,
    end_date DATE,
    status TEXT NOT NULL DEFAULT 'active',
    archived BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);
CREATE INDEX projects_organization_active_idx ON projects (organization_id, updated_at DESC) WHERE deleted_at IS NULL;

CREATE TABLE project_members (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects (id),
    user_id UUID NOT NULL REFERENCES users (id),
    role TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX project_members_key ON project_members (project_id, user_id);

CREATE TABLE scope_baselines (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects (id),
    version INTEGER NOT NULL,
    requirements_json JSONB NOT NULL,
    frozen_at TIMESTAMPTZ,
    change_request_id UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX scope_baselines_version_key ON scope_baselines (project_id, version);

CREATE TABLE requirements (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    baseline_id UUID NOT NULL REFERENCES scope_baselines (id),
    code TEXT NOT NULL,
    title TEXT NOT NULL,
    description TEXT,
    priority TEXT NOT NULL DEFAULT 'medium',
    status TEXT NOT NULL DEFAULT 'pending',
    acceptance_criteria JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);
CREATE UNIQUE INDEX requirements_code_active_key ON requirements (baseline_id, code) WHERE deleted_at IS NULL;
CREATE INDEX requirements_baseline_active_idx ON requirements (baseline_id) WHERE deleted_at IS NULL;

CREATE TABLE change_requests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects (id),
    baseline_id UUID NOT NULL REFERENCES scope_baselines (id),
    type TEXT NOT NULL,
    reason TEXT NOT NULL,
    impact TEXT,
    status TEXT NOT NULL DEFAULT 'pending',
    requester_id UUID NOT NULL REFERENCES users (id),
    decided_by_id UUID REFERENCES users (id),
    decided_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);
CREATE INDEX change_requests_project_active_idx ON change_requests (project_id, created_at DESC) WHERE deleted_at IS NULL;

CREATE TABLE milestones (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects (id),
    name TEXT NOT NULL,
    target_date DATE,
    status TEXT NOT NULL DEFAULT 'pending',
    requirement_ids JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);
CREATE INDEX milestones_project_active_idx ON milestones (project_id) WHERE deleted_at IS NULL;

CREATE TABLE deliveries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects (id),
    version INTEGER NOT NULL,
    baseline_version INTEGER NOT NULL,
    milestone_ids JSONB NOT NULL DEFAULT '[]'::jsonb,
    git_ref TEXT,
    build_no TEXT,
    status TEXT NOT NULL DEFAULT 'draft',
    manifest_hash TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);
CREATE UNIQUE INDEX deliveries_version_key ON deliveries (project_id, version);
CREATE INDEX deliveries_project_active_idx ON deliveries (project_id, status) WHERE deleted_at IS NULL;

CREATE TABLE evidence_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID NOT NULL REFERENCES projects (id),
    delivery_id UUID REFERENCES deliveries (id),
    requirement_id UUID REFERENCES requirements (id),
    type TEXT NOT NULL,
    title TEXT NOT NULL,
    file_key TEXT,
    sha256 TEXT,
    source TEXT,
    occurred_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);
CREATE INDEX evidence_items_delivery_active_idx ON evidence_items (delivery_id, occurred_at) WHERE deleted_at IS NULL;
CREATE INDEX evidence_items_requirement_active_idx ON evidence_items (requirement_id) WHERE deleted_at IS NULL;

CREATE TABLE approvals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    delivery_id UUID NOT NULL REFERENCES deliveries (id),
    customer_id UUID NOT NULL REFERENCES users (id),
    decision TEXT NOT NULL,
    comment TEXT,
    decided_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX approvals_delivery_customer_key ON approvals (delivery_id, customer_id);

CREATE TABLE objections (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    delivery_id UUID NOT NULL REFERENCES deliveries (id),
    criterion_id UUID NOT NULL,
    raised_by_id UUID NOT NULL REFERENCES users (id),
    content TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'open',
    resolved_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX objections_delivery_idx ON objections (delivery_id, created_at);

CREATE TABLE audit_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    actor_id UUID REFERENCES users (id),
    action TEXT NOT NULL,
    object_type TEXT NOT NULL,
    object_id UUID NOT NULL,
    before JSONB,
    after JSONB,
    ip TEXT,
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX audit_events_object_idx ON audit_events (object_type, object_id, occurred_at);

CREATE TABLE share_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    delivery_id UUID NOT NULL REFERENCES deliveries (id),
    token_hash TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX share_tokens_hash_key ON share_tokens (token_hash);

CREATE TABLE outbox_tasks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_type TEXT NOT NULL,
    payload JSONB NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    attempts INTEGER NOT NULL DEFAULT 0,
    next_run_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX outbox_tasks_pending_idx ON outbox_tasks (status, next_run_at) WHERE status = 'pending';

-- +goose Down
-- 按外键依赖的相反顺序删除表。
DROP TABLE IF EXISTS outbox_tasks;
DROP TABLE IF EXISTS share_tokens;
DROP TABLE IF EXISTS audit_events;
DROP TABLE IF EXISTS objections;
DROP TABLE IF EXISTS approvals;
DROP TABLE IF EXISTS evidence_items;
DROP TABLE IF EXISTS deliveries;
DROP TABLE IF EXISTS milestones;
DROP TABLE IF EXISTS change_requests;
DROP TABLE IF EXISTS requirements;
DROP TABLE IF EXISTS scope_baselines;
DROP TABLE IF EXISTS project_members;
DROP TABLE IF EXISTS projects;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS organizations;
