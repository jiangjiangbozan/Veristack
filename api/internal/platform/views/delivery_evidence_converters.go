// Package views 把数据库实体转换为对外安全的响应结构，避免暴露密码哈希与 pgtype 细节。
package views

import (
	"github.com/google/uuid"

	"api/internal/platform/db"
	"api/internal/platform/dbutil"
)

// FromDelivery 把交付实体转换为视图，并解析关联里程碑。
func FromDelivery(model db.Delivery) DeliveryView {
	milestoneIDs := make([]uuid.UUID, 0)
	_ = dbutil.FromJSONBytes(model.MilestoneIds, &milestoneIDs)
	return DeliveryView{
		BaselineVersion: model.BaselineVersion,
		BuildNo:         dbutil.StringFromText(model.BuildNo),
		CreatedAt:       dbutil.TimeString(model.CreatedAt),
		GitRef:          dbutil.StringFromText(model.GitRef),
		ID:              model.ID,
		ManifestHash:    dbutil.StringFromText(model.ManifestHash),
		MilestoneIDs:    milestoneIDs,
		ProjectID:       model.ProjectID,
		Status:          model.Status,
		UpdatedAt:       dbutil.TimeString(model.UpdatedAt),
		Version:         model.Version,
	}
}

// FromEvidence 把证据实体转换为视图。
func FromEvidence(model db.EvidenceItem) EvidenceView {
	return EvidenceView{
		CreatedAt:     dbutil.TimeString(model.CreatedAt),
		FileKey:       dbutil.StringFromText(model.FileKey),
		ID:            model.ID,
		Note:          dbutil.StringFromText(model.Note),
		OccurredAt:    dbutil.TimeString(model.OccurredAt),
		ProjectID:     model.ProjectID,
		RequirementID: pgUUIDString(model.RequirementID),
		SHA256:        dbutil.StringFromText(model.Sha256),
		Source:        dbutil.StringFromText(model.Source),
		Title:         model.Title,
		Type:          model.Type,
	}
}
