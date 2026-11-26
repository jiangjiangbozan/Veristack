// Package evidence 的业务服务，负责证据的上传落盘、内容摘要计算与时间线查询。
package evidence

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"mime/multipart"
	"path"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"api/internal/platform/database"
	"api/internal/platform/datetime"
	"api/internal/platform/db"
	"api/internal/platform/dbutil"
	"api/internal/platform/errs"
	"api/internal/platform/storage"
)

const (
	// maxUploadBytes 限制单个证据附件大小，避免大文件拖垮存储与传输。
	maxUploadBytes = 32 << 20
	// uploadBufferBytes 是上传内容的内存缓冲大小。
	uploadBufferBytes = 1 << 20
)

// Service 提供证据相关的业务方法。
type Service struct {
	Database *pgxpool.Pool       // 数据库连接池
	Storage  storage.ObjectStore // 对象存储
}

// CreateManual 登记一条人工说明型证据，不涉及文件上传。
func (s *Service) CreateManual(ctx context.Context, projectID, deliveryID uuid.UUID, req CreateEvidenceRequest) (db.EvidenceItem, error) {
	requirementID, err := parseOptionalUUID(req.RequirementID)
	if err != nil {
		return db.EvidenceItem{}, errs.BadRequest("关联需求 ID 无效", err)
	}
	return database.Queries(ctx, s.Database).CreateEvidenceItem(ctx, db.CreateEvidenceItemParams{
		DeliveryID:    pgtype.UUID{Bytes: deliveryID, Valid: true},
		Note:          dbutil.TextFromString(req.Note),
		OccurredAt:    datetime.Now(),
		ProjectID:     projectID,
		RequirementID: requirementID,
		Source:        dbutil.TextFromString(req.Source),
		Title:         req.Title,
		Type:          req.Type,
	})
}

// CreateUpload 上传证据附件，落盘后记录内容摘要，保证交付内容可校验。
func (s *Service) CreateUpload(ctx context.Context, projectID, deliveryID uuid.UUID, req UploadRequest, file multipart.File, header *multipart.FileHeader) (db.EvidenceItem, error) {
	if header.Size > maxUploadBytes {
		return db.EvidenceItem{}, errs.BadRequest("附件超出允许的大小上限", nil)
	}
	hasher := newHashWriter()
	key := buildObjectKey(projectID, deliveryID, header.Filename)
	storedKey, err := s.Storage.Put(ctx, key, io.TeeReader(file, hasher))
	if err != nil {
		return db.EvidenceItem{}, fmt.Errorf("保存证据附件失败: %w", err)
	}
	requirementID, err := parseOptionalUUID(req.RequirementID)
	if err != nil {
		return db.EvidenceItem{}, errs.BadRequest("关联需求 ID 无效", err)
	}
	return database.Queries(ctx, s.Database).CreateEvidenceItem(ctx, db.CreateEvidenceItemParams{
		DeliveryID:    pgtype.UUID{Bytes: deliveryID, Valid: true},
		FileKey:       dbutil.TextFromString(storedKey),
		Note:          dbutil.TextFromString(req.Note),
		OccurredAt:    datetime.Now(),
		ProjectID:     projectID,
		RequirementID: requirementID,
		Sha256:        dbutil.TextFromString(hasher.digest()),
		Source:        dbutil.TextFromString(req.Source),
		Title:         req.Title,
		Type:          req.Type,
	})
}

// Get 按 ID 读取证据。
func (s *Service) Get(ctx context.Context, id uuid.UUID) (db.EvidenceItem, error) {
	item, err := database.Queries(ctx, s.Database).GetEvidenceItem(ctx, id)
	if err != nil {
		return db.EvidenceItem{}, errs.NotFound("证据不存在", err)
	}
	return item, nil
}

// ListByDelivery 列出交付版本的证据时间线。
func (s *Service) ListByDelivery(ctx context.Context, deliveryID uuid.UUID) ([]db.EvidenceItem, error) {
	return database.Queries(ctx, s.Database).ListEvidenceByDelivery(ctx, pgtype.UUID{Bytes: deliveryID, Valid: true})
}

// Open 打开证据附件内容供下载。
func (s *Service) Open(ctx context.Context, key string) (io.ReadCloser, error) {
	content, err := s.Storage.Open(ctx, key)
	if err != nil {
		return nil, errs.NotFound("证据附件不存在", err)
	}
	return content, nil
}

// buildObjectKey 生成对象存储键，按项目与交付分层便于清理与追溯。
func buildObjectKey(projectID, deliveryID uuid.UUID, filename string) string {
	return path.Join(projectID.String(), deliveryID.String(), sanitizeFilename(filename))
}

// sanitizeFilename 去掉路径分隔符，避免文件名作路径穿越。
func sanitizeFilename(filename string) string {
	cleaned := path.Base(filename)
	if cleaned == "." || cleaned == "/" {
		return "evidence"
	}
	return cleaned
}

// parseOptionalUUID 解析可选 UUID，空串返回 null。
func parseOptionalUUID(raw string) (pgtype.UUID, error) {
	if raw == "" {
		return pgtype.UUID{}, nil
	}
	parsed, err := uuid.Parse(raw)
	if err != nil {
		return pgtype.UUID{}, err
	}
	return pgtype.UUID{Bytes: parsed, Valid: true}, nil
}

// hashWriter 在写入内容的同时计算 SHA-256 摘要。
type hashWriter struct {
	hasher hash.Hash
}

// newHashWriter 创建摘要写入器。
func newHashWriter() *hashWriter {
	return &hashWriter{hasher: sha256.New()}
}

// Write 把内容写入摘要计算器。
func (h *hashWriter) Write(p []byte) (int, error) {
	return h.hasher.Write(p)
}

// digest 返回十六进制摘要。
func (h *hashWriter) digest() string {
	return hex.EncodeToString(h.hasher.Sum(nil))
}

// ListByProject 列出项目下全部证据，便于从项目视角查看自动化采集结果。
func (s *Service) ListByProject(ctx context.Context, projectID uuid.UUID) ([]db.EvidenceItem, error) {
	return database.Queries(ctx, s.Database).ListEvidenceByProject(ctx, projectID)
}
