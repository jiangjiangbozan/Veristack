import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import { useState } from 'react'

import { Alert } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import { toneForStatus } from '@/components/ui/badge-tone'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Input, Select } from '@/components/ui/field'
import { Modal } from '@/components/ui/modal'
import { createDelivery, listDeliveries } from '@/services/delivery-service'
import { listBaselines } from '@/services/project-service'
import { formatDateTime } from '@/shared/lib/datetime'
import type { DeliveryView } from '@/shared/types/delivery'

/** 交付版本面板，展示交付列表并支持从基线创建新交付。 */
export function DeliveryPanel({ projectId }: { projectId: string }) {
  const [isCreateOpen, setIsCreateOpen] = useState(false)
  const queryClient = useQueryClient()
  const navigate = useNavigate()
  const { data, isLoading } = useQuery({ queryKey: ['deliveries', projectId], queryFn: () => listDeliveries(projectId) })
  const deliveries = data?.items ?? []

  async function onCreated() {
    setIsCreateOpen(false)
    await queryClient.invalidateQueries({ queryKey: ['deliveries', projectId] })
    await queryClient.invalidateQueries({ queryKey: ['overview', projectId] })
  }

  return (
    <div className="space-y-4">
      <header className="flex items-center justify-between">
        <h2 className="text-lg font-medium">交付版本</h2>
        <Button size="sm" onClick={() => setIsCreateOpen(true)}>新建交付</Button>
      </header>
      {isLoading && <p className="text-sm text-muted-foreground">加载中…</p>}
      {!isLoading && deliveries.length === 0 && (
        <Card><CardContent className="py-10 text-center text-sm text-muted-foreground">尚无交付版本。从已冻结的需求基线创建交付后即可归档证据并提交验收。</CardContent></Card>
      )}
      {deliveries.map((delivery) => (
        <DeliveryCard
          delivery={delivery}
          key={delivery.id}
          onOpen={() => navigate({ to: '/deliveries/$deliveryId', params: { deliveryId: delivery.id } })}
        />
      ))}
      {isCreateOpen && (
        <Modal onClose={() => setIsCreateOpen(false)} title="新建交付版本">
          <DeliveryForm onCreated={onCreated} projectId={projectId} />
        </Modal>
      )}
    </div>
  )
}

/** 单个交付版本卡片，展示清单摘要与状态。 */
function DeliveryCard({ delivery, onOpen }: { delivery: DeliveryView; onOpen: () => void }) {
  return (
    <Card className="cursor-pointer transition-shadow hover:shadow-lg" onClick={onOpen}>
      <CardContent className="space-y-3">
        <div className="flex items-center justify-between gap-3">
          <div className="flex items-center gap-3">
            <span className="text-base font-medium">v{delivery.version}</span>
            <Badge tone={toneForStatus(delivery.status)}>{delivery.status}</Badge>
          </div>
          <span className="text-xs text-muted-foreground">基线 v{delivery.baselineVersion} · {formatDateTime(delivery.createdAt)}</span>
        </div>
        <div className="grid gap-2 text-sm text-muted-foreground sm:grid-cols-3">
          <span>Git 提交：{delivery.gitRef || '未填写'}</span>
          <span>构建号：{delivery.buildNo || '未填写'}</span>
          <span className="truncate">清单摘要：{delivery.manifestHash.slice(0, 16)}…</span>
        </div>
      </CardContent>
    </Card>
  )
}

/** 创建交付表单，从已冻结的基线版本生成不可覆盖的交付清单。 */
function DeliveryForm({ onCreated, projectId }: { onCreated: () => Promise<void>; projectId: string }) {
  const [version, setVersion] = useState('1')
  const [baselineVersion, setBaselineVersion] = useState('')
  const [gitRef, setGitRef] = useState('')
  const [buildNo, setBuildNo] = useState('')
  const [error, setError] = useState('')
  const [isSubmitting, setIsSubmitting] = useState(false)
  const { data: baselines } = useQuery({ queryKey: ['baselines', projectId], queryFn: () => listBaselines(projectId) })
  const baselineList = baselines?.items ?? []

  async function onSubmit(event: React.FormEvent) {
    event.preventDefault()
    if (!baselineVersion) {
      setError('请选择需求基线版本。')
      return
    }
    const deliveryVersion = Number(version)
    if (!Number.isInteger(deliveryVersion) || deliveryVersion < 1) {
      setError('交付版本号必须是大于等于 1 的整数。')
      return
    }
    setError('')
    setIsSubmitting(true)
    try {
      await createDelivery(projectId, {
        baselineVersion: Number(baselineVersion),
        buildNo,
        gitRef,
        milestoneIds: [],
        version: deliveryVersion,
      })
      await onCreated()
    } catch (err) {
      setError(err instanceof Error ? err.message : '创建失败')
    } finally {
      setIsSubmitting(false)
    }
  }

  return (
    <form className="space-y-4" noValidate onSubmit={onSubmit}>
      <Select label="需求基线版本" required value={baselineVersion} onChange={(e) => setBaselineVersion(e.target.value)}>
        <option value="">请选择基线</option>
        {baselineList.map((baseline) => (
          <option key={baseline.id} value={baseline.version}>基线 v{baseline.version}</option>
        ))}
      </Select>
      <div className="grid grid-cols-2 gap-4">
        <Input label="交付版本号" required type="number" min={1} value={version} onChange={(e) => setVersion(e.target.value)} />
        <Input label="构建号" value={buildNo} onChange={(e) => setBuildNo(e.target.value)} placeholder="选填" />
      </div>
      <Input label="Git 提交或标签" value={gitRef} onChange={(e) => setGitRef(e.target.value)} placeholder="选填" />
      {error && <Alert title="无法创建交付" tone="danger">{error}</Alert>}
      <Button className="w-full" disabled={isSubmitting} type="submit">{isSubmitting ? '创建中…' : '创建并冻结清单'}</Button>
    </form>
  )
}
