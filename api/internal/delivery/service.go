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
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
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
	auditObjectAcceptance = "acceptance"
	auditObjectDelivery   = "delivery"
	statusDraft           = "draft"
	statusPending         = "pending"
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

// ListAuditEvents 返回与交付相关的审计事件，按发生时间倒序。
//
// 交付生命周期与客户分享动作分别以 delivery、acceptance 作为对象类型，
// 两者都用交付 ID 作为对象 ID，因此按对象过滤后在内存中合并。
// 单个交付的审计事件数量有界，不属于全表扫描；返回全部事件便于时间线完整还原。
func (s *Service) ListAuditEvents(ctx context.Context, deliveryID uuid.UUID) ([]views.AuditEventView, error) {
	queries := database.Queries(ctx, s.Database)
	deliveryEvents, err := queries.ListAuditEventsByObject(ctx, db.ListAuditEventsByObjectParams{
		ObjectType: auditObjectDelivery,
		ObjectID:   deliveryID,
	})
	if err != nil {
		return nil, fmt.Errorf("查询交付审计事件失败: %w", err)
	}
	acceptanceEvents, err := queries.ListAuditEventsByObject(ctx, db.ListAuditEventsByObjectParams{
		ObjectType: auditObjectAcceptance,
		ObjectID:   deliveryID,
	})
	if err != nil {
		return nil, fmt.Errorf("查询验收审计事件失败: %w", err)
	}
	events := mergeAuditEvents(deliveryEvents, acceptanceEvents)
	displayNames := s.actorNames(ctx, events)
	result := make([]views.AuditEventView, 0, len(events))
	for _, event := range events {
		result = append(result, views.FromAuditEvent(event, displayNames[auditActorKey(event.ActorID)]))
	}
	return result, nil
}

// PackArchivePath 返回指定导出目录的本地路径，供流式下载 ZIP 使用。
//
// 只接受交付 ID 与导出 ID 两个 UUID 组成的目录，路径不可能逃出 Pack 根目录；
// 目录不存在时按未导出处理，避免把服务端文件系统细节暴露给调用方。
func (s *Service) PackArchivePath(ctx context.Context, deliveryID, exportID uuid.UUID) (string, error) {
	if _, err := s.Get(ctx, deliveryID); err != nil {
		return "", err
	}
	root := filepath.Join(s.PackRoot, deliveryID.String(), exportID.String())
	info, err := os.Lstat(root)
	if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return "", errs.NotFound("证据包不存在或尚未导出", err)
	}
	return root, nil
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

// PackExport 描述一次 Evidence Pack 导出的结果。
type PackExport struct {
	ExportID    string // 导出编号，用于定位独立快照
	ManifestSHA string // 交付清单摘要
	Root        string // 导出目录的本地路径
	Signed      bool   // 是否附带签名
}

// ExportPack 把交付版本与其证据打包为 Evidence Pack，便于外部独立核验。
//
// 每次导出分配独立目录，写入临时目录校验通过后再原子发布，
// 保证历史导出不会被覆盖，与「证据快照不可覆盖」的产品设计一致。
func (s *Service) ExportPack(ctx context.Context, id uuid.UUID, privateKey ed25519.PrivateKey, organizationID uuid.UUID) (PackExport, error) {
	delivery, err := s.Get(ctx, id)
	if err != nil {
		return PackExport{}, err
	}
	items, err := database.Queries(ctx, s.Database).ListEvidenceByDelivery(ctx, dbutil.NullableUUID(id))
	if err != nil {
		return PackExport{}, fmt.Errorf("查询交付证据失败: %w", err)
	}
	exportID := uuid.New()
	finalRoot := filepath.Join(s.PackRoot, delivery.ID.String(), exportID.String())
	stagingRoot := finalRoot + ".staging"
	if err := os.MkdirAll(stagingRoot, 0o750); err != nil {
		return PackExport{}, fmt.Errorf("创建 Pack 临时目录失败: %w", err)
	}
	defer os.RemoveAll(stagingRoot)
	meta, err := s.buildPack(ctx, stagingRoot, delivery, items, privateKey, organizationID)
	if err != nil {
		return PackExport{}, err
	}
	if err := os.MkdirAll(filepath.Dir(finalRoot), 0o750); err != nil {
		return PackExport{}, fmt.Errorf("创建 Pack 目录失败: %w", err)
	}
	if err := os.Rename(stagingRoot, finalRoot); err != nil {
		return PackExport{}, fmt.Errorf("发布 Pack 目录失败: %w", err)
	}
	return PackExport{
		ExportID:    exportID.String(),
		ManifestSHA: meta.Manifest.SHA256,
		Root:        finalRoot,
		Signed:      meta.Signature != nil,
	}, nil
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

// mergeAuditEvents 合并多组审计事件并按发生时间倒序排列。
//
// 同一时刻的事件保持原有顺序，避免刷新时间线时条目位置跳动；
// 交付与验收事件分别来自两个对象查询，合并逻辑独立成纯函数便于测试。
func mergeAuditEvents(groups ...[]db.AuditEvent) []db.AuditEvent {
	total := 0
	for _, group := range groups {
		total += len(group)
	}
	events := make([]db.AuditEvent, 0, total)
	for _, group := range groups {
		events = append(events, group...)
	}
	sort.SliceStable(events, func(left, right int) bool {
		return events[left].OccurredAt.After(events[right].OccurredAt)
	})
	return events
}

// actorNames 批量解析审计事件的操作人显示名称。
//
// 同一交付的操作人通常只有少数几位，先去重再逐个读取用户，
// 避免为每条审计事件重复查询；用户已删除时保留空名称，前端回退为「已移除成员」。
func (s *Service) actorNames(ctx context.Context, events []db.AuditEvent) map[string]string {
	queries := database.Queries(ctx, s.Database)
	names := make(map[string]string, len(events))
	for _, event := range events {
		key := auditActorKey(event.ActorID)
		if key == "" {
			continue
		}
		if _, resolved := names[key]; resolved {
			continue
		}
		user, err := queries.GetUserByID(ctx, uuid.UUID(event.ActorID.Bytes))
		if err != nil {
			names[key] = ""
			continue
		}
		names[key] = user.DisplayName
	}
	return names
}

// auditActorKey 把可空操作人 ID 转为映射键，系统操作返回空串。
func auditActorKey(actorID pgtype.UUID) string {
	if !actorID.Valid {
		return ""
	}
	return uuid.UUID(actorID.Bytes).String()
}
