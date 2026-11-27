/** 审计事件视图。 */
export interface AuditEventView {
  action: string
  actorId: string
  actorName: string
  id: string
  ip: string
  objectId: string
  objectType: string
  occurredAt: string
}

/** 交付版本视图。 */
export interface DeliveryView {
  baselineVersion: number
  buildNo: string
  createdAt: string
  gitRef: string
  id: string
  manifestHash: string
  milestoneIds: string[]
  projectId: string
  status: string
  updatedAt: string
  version: number
}

/** 交付清单中的验收条件。 */
export interface ManifestCriterion {
  description: string
  key: string
  requirementCode: string
}

/** 交付清单中的需求。 */
export interface ManifestRequirement {
  code: string
  criteria: ManifestCriterion[]
  description: string
  priority: string
  title: string
}

/** 交付清单快照。 */
export interface Manifest {
  baselineVersion: number
  criteria: ManifestCriterion[]
  gitRef: string
  projectName: string
  requirements: ManifestRequirement[]
  version: number
}

/** 证据条目视图。 */
export interface EvidenceView {
  createdAt: string
  fileKey: string
  id: string
  note: string
  occurredAt: string
  projectId: string
  requirementId: string
  sha256: string
  source: string
  title: string
  type: string
}

/** 单条验收条件的裁定结果。 */
export interface AcceptanceVerdict {
  comment: string
  criterionKey: string
  decidedAt: string
  decidedById: string
  deliveryId: string
  requirementCode: string
  requirementTitle: string
  verdict: string
}

/** 开发者对异议的回复。 */
export interface ObjectionReply {
  authorId: string
  content: string
  createdAt: string
  id: string
  objectionId: string
}

/** 客户提出的异议。 */
export interface Objection {
  content: string
  createdAt: string
  criterionKey: string
  deliveryId: string
  id: string
  raisedById: string
  replies: ObjectionReply[]
  resolvedAt: string
  status: string
}
