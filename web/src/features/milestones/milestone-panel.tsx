import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useState, type FormEvent } from 'react'

import { Badge } from '@/components/ui/badge'
import { toneForStatus } from '@/components/ui/badge-tone'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Input } from '@/components/ui/field'
import { Modal } from '@/components/ui/modal'
import { createMilestone, deleteMilestone, listMilestones, updateMilestone } from '@/services/project-service'
import { formatDate } from '@/shared/lib/datetime'

/** 里程碑面板，支持创建、完成与删除。 */
export function MilestonePanel({ projectId }: { projectId: string }) {
  const [isCreateOpen, setIsCreateOpen] = useState(false)
  const queryClient = useQueryClient()
  const { data, isLoading } = useQuery({ queryKey: ['milestones', projectId], queryFn: () => listMilestones(projectId) })

  async function refresh() {
    await queryClient.invalidateQueries({ queryKey: ['milestones', projectId] })
    await queryClient.invalidateQueries({ queryKey: ['overview', projectId] })
  }

  return (
    <div className="space-y-4">
      <header className="flex items-center justify-between">
        <h2 className="text-lg font-medium">里程碑</h2>
        <Button size="sm" onClick={() => setIsCreateOpen(true)}>新建里程碑</Button>
      </header>
      {isLoading && <p className="text-sm text-muted-foreground">加载中…</p>}
      {!isLoading && data?.items.length === 0 && (
        <Card><CardContent className="py-10 text-center text-sm text-muted-foreground">尚无里程碑。</CardContent></Card>
      )}
      {data?.items.map((milestone) => (
        <MilestoneCard key={milestone.id} milestone={milestone} onChanged={refresh} />
      ))}
      {isCreateOpen && (
        <Modal onClose={() => setIsCreateOpen(false)} title="新建里程碑">
          <MilestoneForm onCreated={async () => { setIsCreateOpen(false); await refresh() }} projectId={projectId} />
        </Modal>
      )}
    </div>
  )
}

/** 单个里程碑卡片。 */
function MilestoneCard({ milestone, onChanged }: { milestone: { id: string; name: string; status: string; targetDate: string }; onChanged: () => Promise<void> }) {
  const [isBusy, setIsBusy] = useState(false)

  async function onToggle() {
    setIsBusy(true)
    try {
      await updateMilestone(milestone.id, {
        name: milestone.name,
        requirementIds: [],
        status: milestone.status === 'completed' ? 'pending' : 'completed',
        targetDate: milestone.targetDate,
      })
      await onChanged()
    } finally {
      setIsBusy(false)
    }
  }

  async function onDelete() {
    setIsBusy(true)
    try {
      await deleteMilestone(milestone.id)
      await onChanged()
    } finally {
      setIsBusy(false)
    }
  }

  return (
    <Card>
      <CardContent className="flex items-center justify-between gap-4">
        <div className="space-y-1">
          <div className="flex items-center gap-2">
            <span className="font-medium">{milestone.name}</span>
            <Badge tone={toneForStatus(milestone.status)}>{milestone.status}</Badge>
          </div>
          <p className="text-xs text-muted-foreground">目标日期 {formatDate(milestone.targetDate)}</p>
        </div>
        <div className="flex gap-2">
          <Button disabled={isBusy} size="sm" variant="outline" onClick={onToggle}>
            {milestone.status === 'completed' ? '重新打开' : '标记完成'}
          </Button>
          <Button disabled={isBusy} size="sm" variant="ghost" onClick={onDelete}>删除</Button>
        </div>
      </CardContent>
    </Card>
  )
}

/** 创建里程碑表单。 */
function MilestoneForm({ onCreated, projectId }: { onCreated: () => Promise<void>; projectId: string }) {
  const [name, setName] = useState('')
  const [targetDate, setTargetDate] = useState('')
  const [error, setError] = useState('')
  const [isSubmitting, setIsSubmitting] = useState(false)

  async function onSubmit(event: FormEvent) {
    event.preventDefault()
    setError('')
    setIsSubmitting(true)
    try {
      await createMilestone(projectId, { name, requirementIds: [], targetDate })
      await onCreated()
    } catch (err) {
      setError(err instanceof Error ? err.message : '创建失败')
    } finally {
      setIsSubmitting(false)
    }
  }

  return (
    <form className="space-y-4" onSubmit={onSubmit}>
      <Input label="里程碑名称" required value={name} onChange={(e) => setName(e.target.value)} placeholder="例如：完成首页" />
      <Input label="目标日期" type="date" value={targetDate} onChange={(e) => setTargetDate(e.target.value)} />
      {error && <p className="rounded-md bg-danger/10 px-3 py-2 text-sm text-danger">{error}</p>}
      <Button className="w-full" disabled={isSubmitting} type="submit">{isSubmitting ? '创建中…' : '创建里程碑'}</Button>
    </form>
  )
}
