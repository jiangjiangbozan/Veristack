/** 建议类型，决定建议被采纳后作用于哪个环节。 */
export type SuggestionKind = 'change_impact' | 'delivery_summary' | 'requirement_extraction'

/** 建议的处理状态。 */
export type SuggestionStatus = 'applied' | 'dismissed' | 'draft'

/** AI 提取的候选需求。 */
export interface SuggestedRequirement {
  acceptanceCriteria: string[]
  code: string
  description: string
  priority: string
  title: string
}

/** AI 建议的内容载荷。 */
export interface SuggestionContent {
  impactNotes: string[]
  requirements: SuggestedRequirement[]
  summary: string
}

/** AI 建议视图。 */
export interface AISuggestion {
  content: SuggestionContent
  createdAt: string
  id: string
  kind: SuggestionKind
  model: string
  status: SuggestionStatus
}

/** 测试结果汇总。 */
export interface TestSummary {
  durationMs: number
  failed: number
  passed: number
  skipped: number
  total: number
}

/** CI 报告视图。 */
export interface CIReport {
  buildNo: string
  createdAt: string
  id: string
  provider: string
}

/** Webhook 事件视图。 */
export interface WebhookView {
  eventType: string
  id: string
  provider: string
  receivedAt: string
  status: string
}
