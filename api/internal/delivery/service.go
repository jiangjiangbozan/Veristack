// Package delivery 的业务服务，负责交付版本的创建、清单冻结与验收状态流转。
package delivery

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"api/internal/platform/database"
	"api/internal/platform/datetime"
	"api/internal/platform/db"
	"api/internal/platform/dbutil"
	"api/internal/platform/errs"
	"api/internal/platform/pack"
	"api/internal/platform/views"
)

const (
	statusDraft   = "draft"
	statusPending = "pending"
)

// Service 提供交付版本相关的业务方法。
type Service struct {
	Database    *pgxpool.Pool // 数据库连接池
	PackRoot    string        // Evidence Pack 导出根目录
	StorageRoot string        // 证据附件本地存储根目录
}

// CountByProject 统计项目交付版本总数。
func (s *Service) CountByProject(ctx context.Context, projectID uuid.UUID) (int64, error) {
	return database.Queries(ctx, s.Database).CountDeliveriesByProject(ctx, projectID)
}

// Create 创建交付版本，并把基线需求冻结为不可覆盖的交付清单。
func (s *Service) Create(ctx context.Context, projectID uuid.UUID, req CreateDeliveryRequest) (db.Delivery, error) {
	queries := database.Queries(ctx, s.Database)
	project, err := queries.GetProject(ctx, projectID)
	if err != nil {
		return db.Delivery{}, errs.NotFound("项目不存在", err)
	}
	baseline, err := queries.GetScopeBaselineByVersion(ctx, db.GetScopeBaselineByVersionParams{
		ProjectID: projectID,
		Version:   req.BaselineVersion,
	})
	if err != nil {
		return db.Delivery{}, errs.NotFound("指定的需求基线版本不存在", err)
	}
	manifest := buildManifest(project.Name, baseline, req)
	snapshot, err := dbutil.JSONBytes(manifest)
	if err != nil {
		return db.Delivery{}, err
	}
	milestoneIDs, err := dbutil.JSONBytes(req.MilestoneIDs)
	if err != nil {
		return db.Delivery{}, err
	}
	return queries.CreateDelivery(ctx, db.CreateDeliveryParams{
		BaselineVersion: req.BaselineVersion,
		BuildNo:         dbutil.TextFromString(req.BuildNo),
		GitRef:          dbutil.TextFromString(req.GitRef),
		ManifestHash:    dbutil.TextFromString(hashManifest(snapshot)),
		ManifestJson:    snapshot,
		MilestoneIds:    milestoneIDs,
		ProjectID:       projectID,
		Status:          statusDraft,
		Version:         req.Version,
	})
}

// Get 按 ID 读取交付版本。
func (s *Service) Get(ctx context.Context, id uuid.UUID) (db.Delivery, error) {
	delivery, err := database.Queries(ctx, s.Database).GetDelivery(ctx, id)
	if err != nil {
		return db.Delivery{}, errs.NotFound("交付版本不存在", err)
	}
	return delivery, nil
}

// GetManifest 读取交付清单快照。
func (s *Service) GetManifest(ctx context.Context, id uuid.UUID) (Manifest, error) {
	delivery, err := s.Get(ctx, id)
	if err != nil {
		return Manifest{}, err
	}
	manifest := Manifest{}
	if err := dbutil.FromJSONBytes(delivery.ManifestJson, &manifest); err != nil {
		return Manifest{}, err
	}
	return manifest, nil
}

// List 分页列出项目交付版本。
func (s *Service) List(ctx context.Context, projectID uuid.UUID, limit, offset int32) ([]db.Delivery, error) {
	return database.Queries(ctx, s.Database).ListDeliveriesByProject(ctx, db.ListDeliveriesByProjectParams{
		Limit:     limit,
		Offset:    offset,
		ProjectID: projectID,
	})
}

// Submit 把草稿交付提交为待验收状态。
func (s *Service) Submit(ctx context.Context, id uuid.UUID) (db.Delivery, error) {
	delivery, err := database.Queries(ctx, s.Database).SubmitDelivery(ctx, id)
	if err != nil {
		return db.Delivery{}, errs.BadRequest("仅草稿状态的交付可以提交验收", err)
	}
	return delivery, nil
}

// UpdateStatus 更新交付验收状态。
func (s *Service) UpdateStatus(ctx context.Context, id uuid.UUID, status string) (db.Delivery, error) {
	delivery, err := database.Queries(ctx, s.Database).UpdateDeliveryStatus(ctx, db.UpdateDeliveryStatusParams{
		ID:     id,
		Status: status,
	})
	if err != nil {
		return db.Delivery{}, fmt.Errorf("更新交付状态失败: %w", err)
	}
	return delivery, nil
}

// buildManifest 依据基线快照生成交付清单，验收条件附带稳定键以支持逐条裁定。
func buildManifest(projectName string, baseline db.ScopeBaseline, req CreateDeliveryRequest) Manifest {
	requirements := make([]views.RequirementView, 0)
	_ = dbutil.FromJSONBytes(baseline.RequirementsJson, &requirements)
	manifest := Manifest{
		BaselineVersion: req.BaselineVersion,
		Criteria:        make([]ManifestCriterion, 0),
		GitRef:          req.GitRef,
		ProjectName:     projectName,
		Requirements:    make([]ManifestRequirement, 0),
		Version:         req.Version,
	}
	for _, requirement := range requirements {
		entry := ManifestRequirement{
			Code:        requirement.Code,
			Criteria:    make([]ManifestCriterion, 0),
			Description: requirement.Description,
			Priority:    requirement.Priority,
			Title:       requirement.Title,
		}
		for index, description := range requirement.AcceptanceCriteria {
			criterion := ManifestCriterion{
				Description:     description,
				Key:             fmt.Sprintf("%s::%d", requirement.Code, index),
				RequirementCode: requirement.Code,
			}
			entry.Criteria = append(entry.Criteria, criterion)
			manifest.Criteria = append(manifest.Criteria, criterion)
		}
		manifest.Requirements = append(manifest.Requirements, entry)
	}
	return manifest
}

// hashManifest 计算清单摘要，用于校验交付内容未被篡改。
func hashManifest(snapshot []byte) string {
	sum := sha256.Sum256(snapshot)
	return hex.EncodeToString(sum[:])
}

// ExportPack 把交付版本与其证据打包为 Evidence Pack，便于外部独立核验。
//
// 每次导出分配独立目录，写入临时目录校验通过后再原子发布，
// 保证历史导出不会被覆盖，与「证据快照不可覆盖」的产品设计一致。
func (s *Service) ExportPack(ctx context.Context, id uuid.UUID, privateKey ed25519.PrivateKey, organizationID uuid.UUID) (string, string, bool, error) {
	delivery, err := s.Get(ctx, id)
	if err != nil {
		return "", "", false, err
	}
	items, err := database.Queries(ctx, s.Database).ListEvidenceByDelivery(ctx, dbutil.NullableUUID(id))
	if err != nil {
		return "", "", false, fmt.Errorf("查询交付证据失败: %w", err)
	}
	exportID := uuid.New().String()
	finalRoot := filepath.Join(s.PackRoot, delivery.ID.String(), exportID)
	stagingRoot := finalRoot + ".staging"
	if err := os.MkdirAll(stagingRoot, 0o750); err != nil {
		return "", "", false, fmt.Errorf("创建 Pack 临时目录失败: %w", err)
	}
	defer os.RemoveAll(stagingRoot)
	meta, err := s.buildPack(ctx, stagingRoot, delivery, items, privateKey, organizationID)
	if err != nil {
		return "", "", false, err
	}
	if err := os.MkdirAll(filepath.Dir(finalRoot), 0o750); err != nil {
		return "", "", false, fmt.Errorf("创建 Pack 目录失败: %w", err)
	}
	if err := os.Rename(stagingRoot, finalRoot); err != nil {
		return "", "", false, fmt.Errorf("发布 Pack 目录失败: %w", err)
	}
	return finalRoot, meta.Manifest.SHA256, meta.Signature != nil, nil
}

// buildPack 组装 Pack 内容并写入元数据，附件从对象存储复制进 Pack。
func (s *Service) buildPack(ctx context.Context, root string, delivery db.Delivery, items []db.EvidenceItem, privateKey ed25519.PrivateKey, organizationID uuid.UUID) (pack.Pack, error) {
	builder := pack.NewBuilder(root)
	for _, item := range items {
		fileKey := item.FileKey.String
		if fileKey == "" {
			continue
		}
		sourcePath := filepath.Join(s.StorageRoot, filepath.FromSlash(fileKey))
		if _, err := os.Stat(sourcePath); err != nil {
			continue
		}
		if _, err := builder.AddFile(item.ID.String(), item.Title, sourcePath); err != nil {
			return pack.Pack{}, err
		}
	}
	manifestSHA, err := builder.WriteManifest(delivery.ManifestJson)
	if err != nil {
		return pack.Pack{}, err
	}
	meta := pack.Pack{
		SchemaVersion: pack.SchemaVersion,
		GeneratedAt:   datetime.Now().Format(time.RFC3339),
		Delivery: pack.DeliveryInfo{
			BaselineVersion: delivery.BaselineVersion,
			BuildNo:         delivery.BuildNo.String,
			GitRef:          delivery.GitRef.String,
			ID:              delivery.ID.String(),
			ProjectName:     s.projectName(ctx, delivery.ProjectID),
			Version:         delivery.Version,
		},
		Manifest: pack.ManifestRef{Path: pack.ManifestFileName, SHA256: manifestSHA},
		Files:    builder.Files(),
		Issuer: pack.Issuer{
			KeyID:          pack.KeyID(privateKey.Public().(ed25519.PublicKey)),
			OrganizationID: organizationID.String(),
		},
	}
	if len(privateKey) == ed25519.PrivateKeySize {
		signature, err := pack.Sign(meta, privateKey)
		if err != nil {
			return pack.Pack{}, err
		}
		meta.Signature = signature
	}
	if err := builder.Write(meta); err != nil {
		return pack.Pack{}, err
	}
	return meta, nil
}

// projectName 读取项目名称，读取失败时返回空串。
func (s *Service) projectName(ctx context.Context, projectID uuid.UUID) string {
	project, err := database.Queries(ctx, s.Database).GetProject(ctx, projectID)
	if err != nil {
		return ""
	}
	return project.Name
}
