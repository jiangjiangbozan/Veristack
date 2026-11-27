import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useState, type FormEvent } from 'react'

import { Alert } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Input, Select, Textarea } from '@/components/ui/field'
import { Modal } from '@/components/ui/modal'
import { createBaseline, fetchBaselineDiff, listBaselines } from '@/services/project-service'
import { formatDateTime } from '@/shared/lib/datetime'
import type { BaselineView, RequirementView } from '@/shared/types/domain'

interface RequirementDraft {
  acceptanceCriteria: string
  code: string
  description: string
  priority: string
  title: string
}

function emptyRequirement(index: number): RequirementDraft {
  return { acceptanceCriteria: '', code: `REQ-${String(index + 1).padStart(3, '0')}`, description: '', priority: 'medium', title: '' }
}

/** 需求范围面板，展示基线演进并支持冻结新基线。 */
export function BaselinePanel({ projectId }: { projectId: string }) {
  const [isCreateOpen, setIsCreateOpen] = useState(false)
  const [diffPair, setDiffPair] = useState<{ base: BaselineView; against: BaselineView } | null>(null)
  const queryClient = useQueryClient()
  const { data, isLoading } = useQuery({ queryKey: ['baselines', projectId], queryFn: () => listBaselines(projectId) })
  const baselines = data?.items ?? []

  async function onCreated() {
    setIsCreateOpen(false)
    await queryClient.invalidateQueries({ queryKey: ['baselines', projectId] })
    await queryClient.invalidateQueries({ queryKey: ['overview', projectId] })
  }

  return (
    <div className="space-y-4">
      <header className="flex items-center justify-between">
        <h2 className="text-lg font-medium">需求范围基线</h2>
        <Button size="sm" onClick={() => setIsCreateOpen(true)}>新建基线</Button>
      </header>
      {isLoading && <p className="text-sm text-muted-foreground">加载中…</p>}
      {!isLoading && baselines.length === 0 && (
        <Card><CardContent className="py-10 text-center text-sm text-muted-foreground">尚无需求基线，创建后即可冻结为正式版本。</CardContent></Card>
      )}
      {baselines.slice().reverse().map((baseline) => {
        const previous = baselines.find((item) => item.version === baseline.version - 1)
        return (
          <Card key={baseline.id}>
            <CardContent className="space-y-4">
              <div className="flex flex-wrap items-center justify-between gap-3">
                <div className="flex items-center gap-3">
                  <span className="text-base font-medium">版本 v{baseline.version}</span>
                  <Badge tone="success">已冻结</Badge>
                </div>
                <div className="flex items-center gap-3">
                  {previous && (
                    <Button onClick={() => setDiffPair({ against: previous, base: baseline })} size="sm" variant="outline">
                      对比 v{previous.version}
                    </Button>
                  )}
                  <span className="text-xs text-muted-foreground">{formatDateTime(baseline.frozenAt)}</span>
                </div>
              </div>
              <RequirementList requirements={baseline.requirements} />
            </CardContent>
          </Card>
        )
      })}
      {diffPair && (
        <Modal onClose={() => setDiffPair(null)} title={`基线 v${diffPair.base.version} 对比 v${diffPair.against.version}`}>
          <BaselineDiffContent against={diffPair.base.id} baselineId={diffPair.against.id} />
        </Modal>
      )}
      {isCreateOpen && (
        <Modal onClose={() => setIsCreateOpen(false)} title="新建需求基线">
          <BaselineForm projectId={projectId} onCreated={onCreated} />
        </Modal>
      )}
    </div>
  )
}

/** 基线差异内容，按新增、修改、删除三组展示需求变化。 */
function BaselineDiffContent({ baselineId, against }: { baselineId: string; against: string }) {
  const { data, isLoading } = useQuery({
    queryKey: ['baseline-diff', baselineId, against],
    queryFn: () => fetchBaselineDiff(baselineId, against),
  })

  if (isLoading) {
    return <p className="text-sm text-muted-foreground">正在对比基线…</p>
  }
  if (!data) {
    return <p className="text-sm text-danger">基线对比失败，请稍后重试。</p>
  }
  const total = data.added.length + data.modified.length + data.removed.length
  if (total === 0) {
    return <p className="text-sm text-muted-foreground">两个基线的需求完全一致。</p>
  }

  return (
    <div className="space-y-4">
      <p className="text-sm text-muted-foreground">新增 {data.added.length} 条、修改 {data.modified.length} 条、删除 {data.removed.length} 条。</p>
      <DiffSection label="新增需求" requirements={data.added} tone="success" />
      <DiffSection label="修改需求" requirements={data.modified} tone="warning" />
      <DiffSection label="删除需求" requirements={data.removed} tone="danger" />
    </div>
  )
}

/** 单组差异需求列表。 */
function DiffSection({ label, requirements, tone }: { label: string; requirements: RequirementView[]; tone: 'danger' | 'success' | 'warning' }) {
  if (requirements.length === 0) {
    return null
  }
  return (
    <div className="space-y-2">
      <div className="flex items-center gap-2">
        <span className="text-sm font-medium">{label}</span>
        <Badge tone={tone}>{requirements.length}</Badge>
      </div>
      <ul className="space-y-1.5">
        {requirements.map((requirement) => (
          <li className="rounded-md bg-surface-muted px-3 py-2 text-sm" key={requirement.id}>
            <span className="font-medium">{requirement.code}</span> · {requirement.title}
          </li>
        ))}
      </ul>
    </div>
  )
}

/** 创建基线表单，支持动态增删需求条目。 */
function BaselineForm({ onCreated, projectId }: { onCreated: () => void | Promise<void>; projectId: string }) {
  const [drafts, setDrafts] = useState<RequirementDraft[]>([emptyRequirement(0)])
  const [error, setError] = useState('')
  const [isSubmitting, setIsSubmitting] = useState(false)

  function onDraftChange(index: number, patch: Partial<RequirementDraft>) {
    setDrafts((current) => current.map((item, i) => (i === index ? { ...item, ...patch } : item)))
  }

  async function onSubmit(event: FormEvent) {
    event.preventDefault()
    const invalidIndex = drafts.findIndex((item) => !item.code.trim() || !item.title.trim())
    if (invalidIndex >= 0) {
      setError(`请填写需求 ${invalidIndex + 1} 的编号和标题。`)
      return
    }
    setError('')
    setIsSubmitting(true)
    try {
      await createBaseline(projectId, drafts.map((item) => ({
        acceptanceCriteria: item.acceptanceCriteria.split('\n').map((value) => value.trim()).filter(Boolean),
        code: item.code,
        description: item.description,
        priority: item.priority,
        title: item.title,
      })))
      await onCreated()
    } catch (err) {
      setError(err instanceof Error ? err.message : '创建失败')
    } finally {
      setIsSubmitting(false)
    }
  }

  return (
    <form className="space-y-4" noValidate onSubmit={onSubmit}>
      {drafts.map((draft, index) => (
        <div className="space-y-3 rounded-lg border p-4" key={index}>
          <div className="flex items-center justify-between">
            <span className="text-xs font-medium text-muted-foreground">需求 {index + 1}</span>
            {drafts.length > 1 && (
              <Button size="sm" type="button" variant="ghost" onClick={() => setDrafts((current) => current.filter((_, i) => i !== index))}>移除</Button>
            )}
          </div>
          <div className="grid grid-cols-2 gap-3">
            <Input label="编号" required value={draft.code} onChange={(e) => onDraftChange(index, { code: e.target.value })} />
            <Select label="优先级" value={draft.priority} onChange={(e) => onDraftChange(index, { priority: e.target.value })}>
              <option value="high">高</option>
              <option value="medium">中</option>
              <option value="low">低</option>
            </Select>
          </div>
          <Input label="标题" required value={draft.title} onChange={(e) => onDraftChange(index, { title: e.target.value })} />
          <Textarea label="描述" value={draft.description} onChange={(e) => onDraftChange(index, { description: e.target.value })} />
          <Textarea label="验收条件（每行一条）" value={draft.acceptanceCriteria} onChange={(e) => onDraftChange(index, { acceptanceCriteria: e.target.value })} placeholder={'例如：首页加载小于 2 秒\n支持深色模式'} />
        </div>
      ))}
      <Button size="sm" type="button" variant="outline" onClick={() => setDrafts((current) => [...current, emptyRequirement(current.length)])}>添加需求</Button>
      {error && <Alert title="无法创建基线" tone="danger">{error}</Alert>}
      <Button className="w-full" disabled={isSubmitting} type="submit">{isSubmitting ? '创建中…' : '创建并冻结'}</Button>
    </form>
  )
}

/** 需求列表，展示验收条件。 */
function RequirementList({ requirements }: { requirements: RequirementView[] }) {
  if (requirements.length === 0) {
    return <p className="text-sm text-muted-foreground">该基线没有需求。</p>
  }
  return (
    <ul className="space-y-3">
      {requirements.map((requirement) => (
        <li className="rounded-lg bg-surface-muted p-3" key={requirement.id}>
          <div className="flex items-center justify-between gap-2">
            <span className="text-sm font-medium">{requirement.code} · {requirement.title}</span>
            <Badge>{requirement.priority}</Badge>
          </div>
          {requirement.description && <p className="mt-1 text-sm text-muted-foreground">{requirement.description}</p>}
          {requirement.acceptanceCriteria.length > 0 && (
            <ul className="mt-2 space-y-1 text-xs text-muted-foreground">
              {requirement.acceptanceCriteria.map((criterion, index) => (
                <li key={index}>✓ {criterion}</li>
              ))}
            </ul>
          )}
        </li>
      ))}
    </ul>
  )
}
