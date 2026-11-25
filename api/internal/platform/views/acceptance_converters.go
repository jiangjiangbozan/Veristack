// Package views 把数据库实体转换为对外安全的响应结构，避免暴露密码哈希与 pgtype 细节。
package views

import (
	"github.com/google/uuid"

	"api/internal/platform/db"
	"api/internal/platform/dbutil"
)

// FromAcceptanceVerdict 把验收裁定实体转换为视图。
func FromAcceptanceVerdict(model db.AcceptanceVerdict) AcceptanceVerdictView {
	return AcceptanceVerdictView{
		Comment:          dbutil.StringFromText(model.Comment),
		CriterionKey:     model.CriterionKey,
		DecidedAt:        dbutil.TimeString(model.DecidedAt),
		DecidedByID:      model.DecidedByID,
		DeliveryID:       model.DeliveryID,
		RequirementCode:  model.RequirementCode,
		RequirementTitle: model.RequirementTitle,
		Verdict:          model.Verdict,
	}
}

// FromObjection 把异议实体转换为视图，并附带回复列表。
func FromObjection(model db.Objection, replies []db.ObjectionReply) ObjectionView {
	replyViews := make([]ObjectionReplyView, 0, len(replies))
	for _, reply := range replies {
		replyViews = append(replyViews, FromObjectionReply(reply))
	}
	return ObjectionView{
		Content:      model.Content,
		CreatedAt:    dbutil.TimeString(model.CreatedAt),
		CriterionKey: model.CriterionID,
		DeliveryID:   model.DeliveryID,
		ID:           model.ID,
		RaisedByID:   model.RaisedByID,
		Replies:      replyViews,
		ResolvedAt:   dbutil.NullableTimeString(model.ResolvedAt),
		Status:       model.Status,
	}
}

// FromObjectionReply 把异议回复实体转换为视图。
func FromObjectionReply(model db.ObjectionReply) ObjectionReplyView {
	return ObjectionReplyView{
		AuthorID:    model.AuthorID,
		Content:     model.Content,
		CreatedAt:   dbutil.TimeString(model.CreatedAt),
		ID:          model.ID,
		ObjectionID: model.ObjectionID,
	}
}

// FromManifest 把交付清单快照转换为视图。
func FromManifest(model []byte) ManifestView {
	manifest := ManifestView{
		Criteria:     make([]ManifestCriterionView, 0),
		Requirements: make([]ManifestRequirementView, 0),
	}
	_ = dbutil.FromJSONBytes(model, &manifest)
	return manifest
}

// ObjectionDeliveryID 返回异议所属交付的 UUID。
func ObjectionDeliveryID(model db.Objection) uuid.UUID {
	return model.DeliveryID
}
