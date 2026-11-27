import { useQuery } from '@tanstack/react-query'
import { ShieldCheck } from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Card, CardContent } from '@/components/ui/card'
import { listAuditEvents } from '@/services/delivery-service'
import { formatDateTime } from '@/shared/lib/datetime'
import type { AuditEventView } from '@/shared/types/delivery'

const auditActionLabels: Record<string, string> = {
  'acceptance.share': '生成客户验收链接',
  'delivery.create': '创建交付版本',
  'delivery.export': '导出证据包',
  'delivery.submit': '提交客户验收',
}

const auditObjectLabels: Record<string, string> = {
  acceptance: '验收',
  delivery: '交付',
}

/** 交付操作审计面板，按时间倒序还原交付过程中的关键写操作。 */
export function AuditPanel({ deliveryId }: { deliveryId: string }) {
  const { data, error, isError, isLoading } = useQuery({ queryKey: ['audit', deliveryId], queryFn: () => listAuditEvents(deliveryId) })
  const items = data?.items ?? []

  return (
    <div className="space-y-4">
      <header className="space-y-1">
        <h2 className="flex items-center gap-2 text-lg font-medium">
          <ShieldCheck className="size-5 text-primary" />
          操作审计
        </h2>
        <p className="text-sm text-muted-foreground">记录创建、提交、导出与客户链接生成的每一步，只追加不修改。</p>
      </header>
      {isLoading && <p className="text-sm text-muted-foreground">加载中…</p>}
      {isError && (
        <p className="text-sm text-danger">加载审计记录失败：{error instanceof Error ? error.message : '请稍后重试'}</p>
      )}
      {!isLoading && !isError && items.length === 0 && (
        <Card>
          <CardContent className="py-10 text-center text-sm text-muted-foreground">
            尚无审计记录。创建交付、提交验收或导出证据包后，操作会出现在这里。
          </CardContent>
        </Card>
      )}
      <ol className="relative space-y-3 border-l pl-5">
        {items.map((item) => (
          <AuditTimelineItem item={item} key={item.id} />
        ))}
      </ol>
    </div>
  )
}

/** 审计时间线上的单条事件，展示动作、操作人、时间与来源 IP。 */
function AuditTimelineItem({ item }: { item: AuditEventView }) {
  const actionLabel = auditActionLabels[item.action] ?? item.action
  return (
    <li className="relative">
      <span className="absolute -left-[1.6rem] top-2 size-2.5 rounded-full bg-primary" />
      <Card>
        <CardContent className="space-y-2">
          <div className="flex items-center justify-between gap-3">
            <span className="font-medium">{actionLabel}</span>
            <Badge>{auditObjectLabels[item.objectType] ?? item.objectType}</Badge>
          </div>
          <div className="flex flex-wrap items-center gap-4 text-xs text-muted-foreground">
            <span>{formatDateTime(item.occurredAt)}</span>
            <span>操作人：{item.actorName || '已移除成员'}</span>
            {item.ip && <span>来源 IP：{item.ip}</span>}
          </div>
        </CardContent>
      </Card>
    </li>
  )
}
