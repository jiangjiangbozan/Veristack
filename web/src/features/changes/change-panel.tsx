import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useState, type FormEvent } from 'react'

import { Badge } from '@/components/ui/badge'
import { toneForStatus } from '@/components/ui/badge-tone'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Input, Select, Textarea } from '@/components/ui/field'
import { Modal } from '@/components/ui/modal'
import { createChangeRequest, decideChangeRequest, listChangeRequests } from '@/services/project-service'
import { formatDateTime } from '@/shared/lib/datetime'
import type { ChangeRequestView } from '@/shared/types/domain'

/** 变更申请面板，支持发起变更与客户确认。 */
export function ChangePanel({ projectId }: { projectId: string }) {
  const [isCreateOpen, setIsCreateOpen] = useState(false)
  const queryClient = useQueryClient()
  const { data, isLoading } = useQuery({ queryKey: ['changes', projectId], queryFn: () => listChangeRequests(projectId) })

  async function refresh() {
    await queryClient.invalidateQueries({ queryKey: ['changes', projectId] })
    await queryClient.invalidateQueries({ queryKey: ['baselines', projectId] })
    await queryClient.invalidateQueries({ queryKey: ['overview', projectId] })
  }

  return (
    <div className="space-y-4">
      <header className="flex items-center justify-between">
        <h2 className="text-lg font-medium">变更申请</h2>
        <Button size="sm" onClick={() => setIsCreateOpen(true)}>发起变更</Button>
      </header>
      {isLoading && <p className="text-sm text-muted-foreground">加载中…</p>}
      {!isLoading && data?.items.length === 0 && (
        <Card><CardContent className="py-10 text-center text-sm text-muted-foreground">尚无变更申请。</CardContent></Card>
      )}
      {data?.items.map((change) => (
        <ChangeCard change={change} key={change.id} onDecided={refresh} />
      ))}
      {isCreateOpen && (
        <Modal onClose={() => setIsCreateOpen(false)} title="发起变更申请">
          <ChangeForm onCreated={async () => { setIsCreateOpen(false); await refresh() }} projectId={projectId} />
        </Modal>
      )}
    </div>
  )
}

/** 单条变更申请卡片，待确认时可做决定。 */
function ChangeCard({ change, onDecided }: { change: ChangeRequestView; onDecided: () => Promise<void> }) {
  const [comment, setComment] = useState('')
  const [isDeciding, setIsDeciding] = useState(false)
  const [error, setError] = useState('')

  async function onDecision(decision: string) {
    setIsDeciding(true)
    setError('')
    try {
      await decideChangeRequest(change.id, decision, comment)
      await onDecided()
    } catch (err) {
      setError(err instanceof Error ? err.message : '操作失败')
    } finally {
      setIsDeciding(false)
    }
  }

  return (
    <Card>
      <CardContent className="space-y-3">
        <div className="flex items-center justify-between gap-2">
          <span className="font-medium">{change.type} · {change.reason}</span>
          <Badge tone={toneForStatus(change.status)}>{change.status}</Badge>
        </div>
        {change.impact && <p className="text-sm text-muted-foreground">影响：{change.impact}</p>}
        <p className="text-xs text-muted-foreground">发起于 {formatDateTime(change.createdAt)}</p>
        {change.status === 'pending' && (
          <div className="space-y-2 border-t pt-3">
            <Input label="决定意见（选填）" value={comment} onChange={(e) => setComment(e.target.value)} placeholder="填写确认或拒绝的说明" />
            {error && <p className="text-sm text-danger">{error}</p>}
            <div className="flex gap-2">
              <Button disabled={isDeciding} size="sm" onClick={() => onDecision('approve')}>批准并生成新基线</Button>
              <Button disabled={isDeciding} size="sm" variant="outline" onClick={() => onDecision('reject')}>拒绝</Button>
            </div>
          </div>
        )}
        {change.status !== 'pending' && change.decidedAt && (
          <p className="text-xs text-muted-foreground">决定于 {formatDateTime(change.decidedAt)}</p>
        )}
      </CardContent>
    </Card>
  )
}

/** 发起变更表单。 */
function ChangeForm({ onCreated, projectId }: { onCreated: () => Promise<void>; projectId: string }) {
  const [reason, setReason] = useState('')
  const [impact, setImpact] = useState('')
  const [type, setType] = useState('add')
  const [code, setCode] = useState('')
  const [title, setTitle] = useState('')
  const [description, setDescription] = useState('')
  const [priority, setPriority] = useState('medium')
  const [criteria, setCriteria] = useState('')
  const [error, setError] = useState('')
  const [isSubmitting, setIsSubmitting] = useState(false)

  async function onSubmit(event: FormEvent) {
    event.preventDefault()
    setError('')
    setIsSubmitting(true)
    try {
      await createChangeRequest(projectId, {
        impact,
        reason,
        requirements: [{
          acceptanceCriteria: criteria.split('\n').map((value) => value.trim()).filter(Boolean),
          code,
          description,
          operation: type,
          priority,
          title,
        }],
        type,
      })
      await onCreated()
    } catch (err) {
      setError(err instanceof Error ? err.message : '发起失败')
    } finally {
      setIsSubmitting(false)
    }
  }

  return (
    <form className="space-y-4" onSubmit={onSubmit}>
      <Select label="变更类型" value={type} onChange={(e) => setType(e.target.value)}>
        <option value="add">新增需求</option>
        <option value="modify">修改需求</option>
        <option value="remove">删除需求</option>
      </Select>
      <Textarea label="变更原因" required value={reason} onChange={(e) => setReason(e.target.value)} />
      <Textarea label="影响说明" value={impact} onChange={(e) => setImpact(e.target.value)} />
      <div className="space-y-3 rounded-lg border p-4">
        <p className="text-xs font-medium text-muted-foreground">拟议需求</p>
        <div className="grid grid-cols-2 gap-3">
          <Input label="编号" required value={code} onChange={(e) => setCode(e.target.value)} placeholder="REQ-002" />
          <Select label="优先级" value={priority} onChange={(e) => setPriority(e.target.value)}>
            <option value="high">高</option>
            <option value="medium">中</option>
            <option value="low">低</option>
          </Select>
        </div>
        <Input label="标题" required value={title} onChange={(e) => setTitle(e.target.value)} />
        <Textarea label="描述" value={description} onChange={(e) => setDescription(e.target.value)} />
        <Textarea label="验收条件（每行一条）" value={criteria} onChange={(e) => setCriteria(e.target.value)} />
      </div>
      {error && <p className="rounded-md bg-danger/10 px-3 py-2 text-sm text-danger">{error}</p>}
      <Button className="w-full" disabled={isSubmitting} type="submit">{isSubmitting ? '提交中…' : '提交变更'}</Button>
    </form>
  )
}
