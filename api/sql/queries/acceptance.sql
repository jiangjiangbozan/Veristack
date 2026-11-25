-- name: UpsertAcceptanceVerdict :one
-- 保存客户对单条验收条件的裁定结论，同一条件重复提交时覆盖结论。
INSERT INTO acceptance_verdicts (delivery_id, criterion_key, requirement_code, requirement_title, verdict, comment, decided_by_id)
VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (delivery_id, criterion_key) DO UPDATE
SET verdict = EXCLUDED.verdict,
    comment = EXCLUDED.comment,
    decided_by_id = EXCLUDED.decided_by_id,
    decided_at = now()
RETURNING *;

-- name: ListAcceptanceVerdicts :many
-- 列出交付版本的全部验收裁定，按需求编号与条件键排序。
SELECT * FROM acceptance_verdicts
WHERE delivery_id = $1
ORDER BY requirement_code, criterion_key;

-- name: CountFailingVerdicts :one
-- 统计交付版本中未通过的验收条件数量。
SELECT count(*) FROM acceptance_verdicts
WHERE delivery_id = $1 AND verdict = 'fail';

-- name: CreateApproval :one
-- 记录客户对交付的最终确认结论。
INSERT INTO approvals (delivery_id, customer_id, decision, comment)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetApproval :one
-- 读取客户对交付的确认记录。
SELECT * FROM approvals
WHERE delivery_id = $1 AND customer_id = $2;

-- name: UpdateApproval :one
-- 更新客户对交付的确认结论。
UPDATE approvals
SET decision = $2, comment = $3, decided_at = now()
WHERE delivery_id = $1 AND customer_id = $4
RETURNING *;
