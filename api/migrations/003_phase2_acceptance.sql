-- +goose Up
-- 阶段二：支撑交付清单、证据上传、逐条验收裁定与异议闭环的数据结构。

-- 交付清单快照：交付创建时冻结需求与验收条件，保证清单不可被后续改动覆盖。
ALTER TABLE deliveries ADD COLUMN manifest_json JSONB NOT NULL DEFAULT '{}'::jsonb;

-- 人工说明型证据没有附件，用备注字段承载内容。
ALTER TABLE evidence_items ADD COLUMN note TEXT;

-- 验收裁定：客户对每条验收条件逐条给出通过或不通过的结论。
CREATE TABLE acceptance_verdicts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    delivery_id UUID NOT NULL REFERENCES deliveries (id),
    criterion_key TEXT NOT NULL,
    requirement_code TEXT NOT NULL,
    requirement_title TEXT NOT NULL,
    verdict TEXT NOT NULL,
    comment TEXT,
    decided_by_id UUID NOT NULL REFERENCES users (id),
    decided_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX acceptance_verdicts_delivery_criterion_key ON acceptance_verdicts (delivery_id, criterion_key);
CREATE INDEX acceptance_verdicts_delivery_idx ON acceptance_verdicts (delivery_id);

-- 异议回复：开发者答复客户异议，形成客户提出、开发者回复、客户复核的闭环。
CREATE TABLE objection_replies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    objection_id UUID NOT NULL REFERENCES objections (id),
    author_id UUID NOT NULL REFERENCES users (id),
    content TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX objection_replies_objection_idx ON objection_replies (objection_id, created_at);

-- 验收条件来自基线快照中的文本，改用稳定键标识以便逐条裁定与异议定位。
ALTER TABLE objections ALTER COLUMN criterion_id TYPE TEXT;

-- 交付版本查询与分享链接查询需要按交付定位。
CREATE INDEX deliveries_project_version_idx ON deliveries (project_id, version DESC) WHERE deleted_at IS NULL;
CREATE INDEX share_tokens_delivery_idx ON share_tokens (delivery_id);

-- +goose Down
-- 回退阶段二的结构变更。
DROP INDEX IF EXISTS share_tokens_delivery_idx;
DROP INDEX IF EXISTS deliveries_project_version_idx;
ALTER TABLE objections ALTER COLUMN criterion_id TYPE UUID USING criterion_id::uuid;
DROP TABLE IF EXISTS objection_replies;
DROP TABLE IF EXISTS acceptance_verdicts;
ALTER TABLE evidence_items DROP COLUMN IF EXISTS note;
ALTER TABLE deliveries DROP COLUMN IF EXISTS manifest_json;
