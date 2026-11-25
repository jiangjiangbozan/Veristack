import { request } from '@/services/http-client'
import type { AISuggestion, CIReport, TestSummary, WebhookView } from '@/shared/types/integration'

/** 采纳一条 AI 建议。 */
export function applySuggestion(suggestionId: string): Promise<AISuggestion> {
  return request(`/ai/suggestions/${suggestionId}/apply`, { body: '{}', method: 'POST' })
}

/** 生成交付版本的自然语言摘要建议。 */
export function generateDeliverySummary(projectId: string, deliveryId: string): Promise<{ suggestion: AISuggestion }> {
  return request(`/projects/${projectId}/ai/delivery-summary`, {
    body: JSON.stringify({ deliveryId }),
    method: 'POST',
  })
}

/** 忽略一条 AI 建议。 */
export function dismissSuggestion(suggestionId: string): Promise<AISuggestion> {
  return request(`/ai/suggestions/${suggestionId}/dismiss`, { body: '{}', method: 'POST' })
}

/** 导入 CI 测试报告。 */
export function importCIReport(
  projectId: string,
  payload: { buildNo: string; deliveryId: string; gitRef: string; provider: string; reportXml: string },
): Promise<{ report: CIReport; summary: TestSummary }> {
  return request(`/projects/${projectId}/ci-reports`, { body: JSON.stringify(payload), method: 'POST' })
}

/** 列出项目的 AI 建议。 */
export function listSuggestions(projectId: string, kind = ''): Promise<{ items: AISuggestion[] }> {
  return request(`/projects/${projectId}/ai/suggestions${kind ? `?kind=${kind}` : ''}`)
}

/** 列出项目收到的 Webhook 事件。 */
export function listWebhooks(projectId: string): Promise<{ items: WebhookView[] }> {
  return request(`/projects/${projectId}/webhooks`)
}

/** 从需求文档提取候选需求建议。 */
export function extractRequirements(projectId: string, content: string): Promise<{ suggestion: AISuggestion }> {
  return request(`/projects/${projectId}/ai/requirements`, {
    body: JSON.stringify({ content }),
    method: 'POST',
  })
}

/** 分析变更影响建议。 */
export function analyzeChangeImpact(projectId: string, changeRequestId: string): Promise<{ suggestion: AISuggestion }> {
  return request(`/projects/${projectId}/ai/change-impact`, {
    body: JSON.stringify({ changeRequestId }),
    method: 'POST',
  })
}
