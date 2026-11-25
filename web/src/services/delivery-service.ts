import { request } from '@/services/http-client'
import type {
  AcceptanceVerdict,
  DeliveryView,
  EvidenceView,
  Manifest,
  Objection,
  ObjectionReply,
} from '@/shared/types/delivery'

/** 客户验收门户读取到的交付内容。 */
export interface PortalDelivery {
  delivery: DeliveryView
  evidence: EvidenceView[]
  manifest: Manifest
  verdicts: AcceptanceVerdict[]
}

/** 开发者视角的验收概览。 */
export interface AcceptanceSummary {
  delivery: DeliveryView
  objections: Objection[]
  verdicts: AcceptanceVerdict[]
}

/** 登记人工说明型证据。 */
export function createManualEvidence(
  deliveryId: string,
  payload: { note: string; requirementId?: string; source: string; title: string; type: string },
): Promise<{ evidence: EvidenceView }> {
  return request(`/deliveries/${deliveryId}/evidence`, {
    body: JSON.stringify(payload),
    method: 'POST',
  })
}

/** 创建交付版本。 */
export function createDelivery(
  projectId: string,
  payload: { baselineVersion: number; buildNo: string; gitRef: string; milestoneIds: string[]; version: number },
): Promise<{ delivery: DeliveryView }> {
  return request(`/projects/${projectId}/deliveries`, { body: JSON.stringify(payload), method: 'POST' })
}

/** 读取开发者视角的验收概览。 */
export function fetchAcceptanceSummary(deliveryId: string): Promise<AcceptanceSummary> {
  return request(`/deliveries/${deliveryId}/acceptance`)
}

/** 读取交付版本详情。 */
export function fetchDelivery(deliveryId: string): Promise<{ delivery: DeliveryView }> {
  return request(`/deliveries/${deliveryId}`)
}

/** 读取交付清单快照。 */
export function fetchManifest(deliveryId: string): Promise<Manifest> {
  return request(`/deliveries/${deliveryId}/manifest`)
}

/** 列出交付版本的证据时间线。 */
export function listEvidence(deliveryId: string): Promise<{ items: EvidenceView[] }> {
  return request(`/deliveries/${deliveryId}/evidence`)
}

/** 列出项目交付版本。 */
export function listDeliveries(projectId: string): Promise<{ items: DeliveryView[] }> {
  return request(`/projects/${projectId}/deliveries`)
}

/** 读取客户验收门户内容。 */
export function fetchPortalDelivery(token: string): Promise<PortalDelivery> {
  return request(`/share/${token}`)
}

/** 客户保存单条验收条件的裁定。 */
export function savePortalVerdict(
  token: string,
  criterionKey: string,
  payload: { comment: string; email: string; verdict: string },
): Promise<{ verdict: AcceptanceVerdict }> {
  return request(`/share/${token}/verdicts/${encodeURIComponent(criterionKey)}`, {
    body: JSON.stringify(payload),
    method: 'POST',
  })
}

/** 客户提交最终验收结论。 */
export function submitPortalDecision(
  token: string,
  payload: {
    comment: string
    decision: string
    email: string
    objections: Array<{ comment: string; criterionKey: string }>
  },
): Promise<{ delivery: DeliveryView }> {
  return request(`/share/${token}/decision`, { body: JSON.stringify(payload), method: 'POST' })
}

/** 开发者回复客户异议。 */
export function replyObjection(objectionId: string, content: string): Promise<{ reply: ObjectionReply }> {
  return request(`/objections/${objectionId}/replies`, {
    body: JSON.stringify({ content }),
    method: 'POST',
  })
}

/** 关闭客户异议。 */
export function resolveObjection(objectionId: string): Promise<{ objection: Objection }> {
  return request(`/objections/${objectionId}/resolve`, { body: '{}', method: 'POST' })
}

/** 生成客户验收分享链接。 */
export function createShareToken(deliveryId: string, expiresInDays = 7): Promise<{ expiresAt: string; token: string; url: string }> {
  return request(`/deliveries/${deliveryId}/share-token`, {
    body: JSON.stringify({ expiresInDays }),
    method: 'POST',
  })
}

/** 提交交付进入待验收状态。 */
export function submitDelivery(deliveryId: string): Promise<{ delivery: DeliveryView }> {
  return request(`/deliveries/${deliveryId}/submit`, { body: '{}', method: 'POST' })
}

/** 上传证据附件。 */
export function uploadEvidence(
  deliveryId: string,
  formData: FormData,
): Promise<{ evidence: EvidenceView }> {
  return request(`/deliveries/${deliveryId}/evidence`, { body: formData, method: 'POST' })
}

/** 导出 Evidence Pack 的结果。 */
export interface PackExportResult {
  downloadUrl: string
  manifestSha256: string
  packPath: string
  signed: boolean
}

/** 导出 Evidence Pack，供客户或第三方离线核验交付内容。 */
export function exportPack(deliveryId: string): Promise<PackExportResult> {
  return request(`/deliveries/${deliveryId}/pack`, { body: '{}', method: 'POST' })
}
