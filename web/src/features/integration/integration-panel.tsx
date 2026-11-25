import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Check, CircleSlash, Sparkles, Webhook } from 'lucide-react'
import { useState } from 'react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Input, Select, Textarea } from '@/components/ui/field'
import { Modal } from '@/components/ui/modal'
import {
  applySuggestion,
  dismissSuggestion,
  extractRequirements,
  importCIReport,
  listSuggestions,
  listWebhooks,
} from '@/services/integration-service'
import { formatDateTime } from '@/shared/lib/datetime'
import type { AISuggestion } from '@/shared/types/integration'

const suggestionKindLabels: Record<string, string> = {
  change_impact: '变更影响分析',
  delivery_summary: '交付摘要',
  requirement_extraction: '需求提取',
}

/** 集成面板，汇总 CI 报告、AI 建议与 Webhook 事件。 */
export function IntegrationPanel({ projectId }: { projectId: string }) {
  const [isImportOpen, setIsImportOpen] = useState(false)
  const [isExtractOpen, setIsExtractOpen] = useState(false)
  const queryClient = useQueryClient()
  const suggestionsQuery = useQuery({ queryKey: ['suggestions', projectId], queryFn: () => listSuggestions(projectId) })
  const webhooksQuery = useQuery({ queryKey: ['webhooks', projectId], queryFn: () => listWebhooks(projectId) })

  async function refresh() {
    await queryClient.invalidateQueries({ queryKey: ['suggestions', projectId] })
    await queryClient.invalidateQueries({ queryKey: ['webhooks', projectId] })
  }

  return (
    <div className="space-y-4">
      <header className="flex flex-wrap items-center justify-between gap-2">
        <h2 className="text-lg font-medium">自动化集成</h2>
        <div className="flex gap-2">
          <Button onClick={() => setIsExtractOpen(true)} size="sm" variant="outline">
            <Sparkles className="size-4" />
            提取需求
          </Button>
          <Button onClick={() => setIsImportOpen(true)} size="sm">
            导入 CI 报告
          </Button>
        </div>
      </header>

      <Card>
        <CardContent className="space-y-3">
          <div className="flex items-center gap-2">
            <Webhook className="size-4 text-muted-foreground" />
            <h3 className="font-medium">Webhook 事件</h3>
          </div>
          <p className="text-xs text-muted-foreground">
            回调地址：<code className="text-foreground">/api/v1/webhooks/github/projects/{projectId}</code>
          </p>
          {webhooksQuery.data?.items.length === 0 && (
            <p className="text-sm text-muted-foreground">尚未收到外部事件。在 GitHub 或 GitLab 中配置回调地址后，提交与流水线事件会自动归档为证据。</p>
          )}
          <ul className="space-y-2">
            {webhooksQuery.data?.items.map((event) => (
              <li className="flex items-center justify-between gap-3 rounded-lg bg-surface-muted p-3" key={event.id}>
                <div className="space-y-0.5">
                  <p className="text-sm font-medium">{event.eventType}</p>
                  <p className="text-xs text-muted-foreground">{event.provider} · {formatDateTime(event.receivedAt)}</p>
                </div>
                <Badge tone={event.status === 'received' ? 'warning' : 'success'}>{event.status}</Badge>
              </li>
            ))}
          </ul>
        </CardContent>
      </Card>

      <Card>
        <CardContent className="space-y-3">
          <div className="flex items-center gap-2">
            <Sparkles className="size-4 text-primary" />
            <h3 className="font-medium">AI 建议</h3>
          </div>
          <p className="text-xs text-muted-foreground">AI 只提供建议，正式基线与交付结论必须由人工确认后才生效。</p>
          {suggestionsQuery.data?.items.length === 0 && (
            <p className="text-sm text-muted-foreground">暂无 AI 建议。可从需求文档提取候选需求，或对变更与交付生成分析建议。</p>
          )}
          {suggestionsQuery.data?.items.map((suggestion) => (
            <SuggestionCard key={suggestion.id} onChanged={refresh} suggestion={suggestion} />
          ))}
        </CardContent>
      </Card>

      {isImportOpen && (
        <Modal onClose={() => setIsImportOpen(false)} title="导入 CI 测试报告">
          <CIReportForm onCreated={async () => { setIsImportOpen(false); await refresh() }} projectId={projectId} />
        </Modal>
      )}
      {isExtractOpen && (
        <Modal onClose={() => setIsExtractOpen(false)} title="从文档提取候选需求">
          <RequirementExtractForm onCreated={async () => { setIsExtractOpen(false); await refresh() }} projectId={projectId} />
        </Modal>
      )}
    </div>
  )
}

/** AI 建议卡片，展示内容并提供采纳与忽略操作。 */
function SuggestionCard({ onChanged, suggestion }: { onChanged: () => Promise<void>; suggestion: AISuggestion }) {
  const applyMutation = useMutation({
    mutationFn: () => applySuggestion(suggestion.id),
    onSuccess: () => void onChanged(),
  })
  const dismissMutation = useMutation({
    mutationFn: () => dismissSuggestion(suggestion.id),
    onSuccess: () => void onChanged(),
  })
  const isBusy = applyMutation.isPending || dismissMutation.isPending

  return (
    <div className="space-y-3 rounded-lg border p-3">
      <div className="flex items-center justify-between gap-3">
        <span className="text-sm font-medium">{suggestionKindLabels[suggestion.kind] ?? suggestion.kind}</span>
        <Badge tone={suggestion.status === 'applied' ? 'success' : suggestion.status === 'dismissed' ? 'neutral' : 'warning'}>
          {suggestion.status === 'applied' ? '已采纳' : suggestion.status === 'dismissed' ? '已忽略' : '待确认'}
        </Badge>
      </div>
      {suggestion.content.summary && (
        <p className="whitespace-pre-wrap text-sm text-muted-foreground">{suggestion.content.summary}</p>
      )}
      {suggestion.content.requirements.length > 0 && (
        <ul className="space-y-2">
          {suggestion.content.requirements.map((requirement) => (
            <li className="rounded-md bg-surface-muted p-2.5" key={requirement.code}>
              <div className="flex items-center justify-between gap-2">
                <span className="text-sm font-medium">{requirement.code} · {requirement.title}</span>
                <Badge>{requirement.priority}</Badge>
              </div>
              {requirement.acceptanceCriteria.length > 0 && (
                <ul className="mt-1.5 space-y-1 text-xs text-muted-foreground">
                  {requirement.acceptanceCriteria.map((criterion, index) => (
                    <li key={index}>✓ {criterion}</li>
                  ))}
                </ul>
              )}
            </li>
          ))}
        </ul>
      )}
      {suggestion.content.impactNotes.length > 0 && (
        <ul className="space-y-1 text-sm text-muted-foreground">
          {suggestion.content.impactNotes.map((note, index) => (
            <li key={index}>· {note}</li>
          ))}
        </ul>
      )}
      <p className="text-xs text-muted-foreground">{suggestion.model} · {formatDateTime(suggestion.createdAt)}</p>
      {suggestion.status === 'draft' && (
        <div className="flex gap-2 border-t pt-3">
          <Button disabled={isBusy} size="sm" onClick={() => applyMutation.mutate()}>
            <Check className="size-4" />
            采纳
          </Button>
          <Button disabled={isBusy} size="sm" variant="outline" onClick={() => dismissMutation.mutate()}>
            <CircleSlash className="size-4" />
            忽略
          </Button>
        </div>
      )}
    </div>
  )
}

/** CI 报告导入表单，粘贴标准测试报告 XML 即可。 */
function CIReportForm({ onCreated, projectId }: { onCreated: () => Promise<void>; projectId: string }) {
  const [provider, setProvider] = useState('github')
  const [buildNo, setBuildNo] = useState('')
  const [gitRef, setGitRef] = useState('')
  const [reportXml, setReportXml] = useState('')
  const [error, setError] = useState('')
  const [isSubmitting, setIsSubmitting] = useState(false)

  async function onSubmit(event: React.FormEvent) {
    event.preventDefault()
    setError('')
    setIsSubmitting(true)
    try {
      await importCIReport(projectId, { buildNo, deliveryId: '', gitRef, provider, reportXml })
      await onCreated()
    } catch (err) {
      setError(err instanceof Error ? err.message : '导入失败')
    } finally {
      setIsSubmitting(false)
    }
  }

  return (
    <form className="space-y-4" onSubmit={onSubmit}>
      <Select label="来源平台" value={provider} onChange={(e) => setProvider(e.target.value)}>
        <option value="github">GitHub</option>
        <option value="gitlab">GitLab</option>
        <option value="generic">通用</option>
      </Select>
      <div className="grid grid-cols-2 gap-4">
        <Input label="构建号" value={buildNo} onChange={(e) => setBuildNo(e.target.value)} placeholder="选填" />
        <Input label="Git 提交" value={gitRef} onChange={(e) => setGitRef(e.target.value)} placeholder="选填" />
      </div>
      <Textarea label="测试报告（JUnit XML）" required value={reportXml} onChange={(e) => setReportXml(e.target.value)} placeholder="粘贴 CI 产出的 test-results.xml" />
      {error && <p className="rounded-md bg-danger/10 px-3 py-2 text-sm text-danger">{error}</p>}
      <Button className="w-full" disabled={isSubmitting} type="submit">{isSubmitting ? '导入中…' : '导入并汇总'}</Button>
    </form>
  )
}

/** 需求提取表单，粘贴原始需求文档生成候选需求。 */
function RequirementExtractForm({ onCreated, projectId }: { onCreated: () => Promise<void>; projectId: string }) {
  const [content, setContent] = useState('')
  const [error, setError] = useState('')
  const [isSubmitting, setIsSubmitting] = useState(false)
  const extractMutation = useMutation({
    mutationFn: () => extractRequirements(projectId, content),
    onSuccess: () => void onCreated(),
  })

  async function onSubmit(event: React.FormEvent) {
    event.preventDefault()
    setError('')
    setIsSubmitting(true)
    try {
      await extractMutation.mutateAsync()
    } catch (err) {
      setError(err instanceof Error ? err.message : '提取失败')
    } finally {
      setIsSubmitting(false)
    }
  }

  return (
    <form className="space-y-4" onSubmit={onSubmit}>
      <Textarea label="原始需求文档" required value={content} onChange={(e) => setContent(e.target.value)} placeholder="粘贴客户原始需求文本，每行一条便于识别" />
      <p className="text-xs text-muted-foreground">提取结果为候选建议，需人工确认后才会进入需求基线。</p>
      {error && <p className="rounded-md bg-danger/10 px-3 py-2 text-sm text-danger">{error}</p>}
      <Button className="w-full" disabled={isSubmitting} type="submit">{isSubmitting ? '分析中…' : '生成候选需求'}</Button>
    </form>
  )
}
