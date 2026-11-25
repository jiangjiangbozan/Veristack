import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Check, Copy, Link2, MessageSquareWarning } from 'lucide-react'
import { useState } from 'react'

import { Badge } from '@/components/ui/badge'
import { toneForStatus } from '@/components/ui/badge-tone'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Textarea } from '@/components/ui/field'
import { createShareToken, fetchAcceptanceSummary, replyObjection, resolveObjection, submitDelivery } from '@/services/delivery-service'
import { formatDateTime } from '@/shared/lib/datetime'
import type { Objection } from '@/shared/types/delivery'

/** 验收面板，展示客户裁定与异议，并管理客户分享链接。 */
export function AcceptancePanel({ deliveryId }: { deliveryId: string }) {
  const [shareUrl, setShareUrl] = useState('')
  const [copied, setCopied] = useState(false)
  const queryClient = useQueryClient()
  const { data, isLoading } = useQuery({ queryKey: ['acceptance', deliveryId], queryFn: () => fetchAcceptanceSummary(deliveryId) })

  const shareMutation = useMutation({
    mutationFn: () => createShareToken(deliveryId, 7),
    onSuccess: (result) => {
      setShareUrl(`${window.location.origin}/accept/${result.token}`)
      void queryClient.invalidateQueries({ queryKey: ['acceptance', deliveryId] })
    },
  })

  const submitMutation = useMutation({
    mutationFn: () => submitDelivery(deliveryId),
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: ['acceptance', deliveryId] }),
  })

  async function onCopy() {
    await navigator.clipboard.writeText(shareUrl)
    setCopied(true)
    window.setTimeout(() => setCopied(false), 2000)
  }

  return (
    <div className="space-y-4">
      <header className="flex items-center justify-between">
        <h2 className="text-lg font-medium">客户验收</h2>
        <div className="flex items-center gap-2">
          {data?.delivery.status === 'draft' && (
            <Button disabled={submitMutation.isPending} size="sm" variant="outline" onClick={() => submitMutation.mutate()}>
              提交客户验收
            </Button>
          )}
          <Button disabled={shareMutation.isPending} size="sm" onClick={() => shareMutation.mutate()}>
            <Link2 className="size-4" />
            生成客户链接
          </Button>
        </div>
      </header>

      {shareUrl && (
        <Card>
          <CardContent className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
            <code className="truncate text-sm text-muted-foreground">{shareUrl}</code>
            <Button onClick={onCopy} size="sm" variant="outline">
              {copied ? <Check className="size-4" /> : <Copy className="size-4" />}
              {copied ? '已复制' : '复制链接'}
            </Button>
          </CardContent>
        </Card>
      )}

      {isLoading && <p className="text-sm text-muted-foreground">加载中…</p>}

      {data && (
        <>
          <Card>
            <CardContent className="space-y-3">
              <div className="flex items-center justify-between">
                <h3 className="font-medium">逐条验收裁定</h3>
                <Badge tone={toneForStatus(data.delivery.status)}>{data.delivery.status}</Badge>
              </div>
              {data.verdicts.length === 0 && <p className="text-sm text-muted-foreground">客户尚未对验收条件作出裁定。</p>}
              <ul className="space-y-2">
                {data.verdicts.map((verdict) => (
                  <li className="rounded-lg bg-surface-muted p-3" key={verdict.criterionKey}>
                    <div className="flex items-center justify-between gap-3">
                      <span className="text-sm font-medium">{verdict.requirementCode} · {verdict.requirementTitle}</span>
                      <Badge tone={verdict.verdict === 'pass' ? 'success' : 'danger'}>{verdict.verdict === 'pass' ? '通过' : '不通过'}</Badge>
                    </div>
                    <p className="mt-1 text-xs text-muted-foreground">{verdict.criterionKey}</p>
                    {verdict.comment && <p className="mt-1 text-sm text-muted-foreground">{verdict.comment}</p>}
                    <p className="mt-1 text-xs text-muted-foreground">{formatDateTime(verdict.decidedAt)}</p>
                  </li>
                ))}
              </ul>
            </CardContent>
          </Card>

          <Card>
            <CardContent className="space-y-3">
              <div className="flex items-center gap-2">
                <MessageSquareWarning className="size-4 text-warning" />
                <h3 className="font-medium">客户异议</h3>
              </div>
              {data.objections.length === 0 && <p className="text-sm text-muted-foreground">暂无异议。</p>}
              {data.objections.map((objection) => (
                <ObjectionCard key={objection.id} objection={objection} />
              ))}
            </CardContent>
          </Card>
        </>
      )}
    </div>
  )
}

/** 异议卡片，支持开发者回复与关闭，形成客户复核闭环。 */
function ObjectionCard({ objection }: { objection: Objection }) {
  const [content, setContent] = useState('')
  const queryClient = useQueryClient()
  const replyMutation = useMutation({
    mutationFn: () => replyObjection(objection.id, content),
    onSuccess: () => {
      setContent('')
      void queryClient.invalidateQueries({ queryKey: ['acceptance'] })
    },
  })
  const resolveMutation = useMutation({
    mutationFn: () => resolveObjection(objection.id),
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: ['acceptance'] }),
  })

  return (
    <div className="space-y-3 rounded-lg border p-3">
      <div className="flex items-center justify-between gap-3">
        <span className="text-sm font-medium">{objection.criterionKey}</span>
        <Badge tone={objection.status === 'resolved' ? 'success' : 'warning'}>{objection.status === 'resolved' ? '已关闭' : '待处理'}</Badge>
      </div>
      <p className="text-sm">{objection.content}</p>
      <p className="text-xs text-muted-foreground">提出于 {formatDateTime(objection.createdAt)}</p>
      {objection.replies.length > 0 && (
        <ul className="space-y-2 border-l-2 pl-3">
          {objection.replies.map((reply) => (
            <li className="text-sm" key={reply.id}>
              <p>{reply.content}</p>
              <p className="text-xs text-muted-foreground">{formatDateTime(reply.createdAt)}</p>
            </li>
          ))}
        </ul>
      )}
      {objection.status !== 'resolved' && (
        <div className="space-y-2 border-t pt-3">
          <Textarea label="回复客户" value={content} onChange={(e) => setContent(e.target.value)} placeholder="说明处理结果，请客户复核" />
          <div className="flex gap-2">
            <Button
              disabled={!content || replyMutation.isPending}
              size="sm"
              onClick={() => replyMutation.mutate()}
            >
              回复
            </Button>
            <Button disabled={resolveMutation.isPending} size="sm" variant="outline" onClick={() => resolveMutation.mutate()}>
              标记为已复核关闭
            </Button>
          </div>
        </div>
      )}
    </div>
  )
}
