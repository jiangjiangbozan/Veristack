-- +goose Up
-- 为变更申请增加拟议需求快照，客户确认后据此生成新基线版本。
ALTER TABLE change_requests ADD COLUMN IF NOT EXISTS requirements_json JSONB;

-- +goose Down
ALTER TABLE change_requests DROP COLUMN IF EXISTS requirements_json;
