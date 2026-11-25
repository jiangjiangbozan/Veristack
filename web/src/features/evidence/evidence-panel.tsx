import { useQuery, useQueryClient } from '@tanstack/react-query'
import { FileUp, Link2 } from 'lucide-react'
import { useRef, useState } from 'react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Input, Select, Textarea } from '@/components/ui/field'
import { Modal } from '@/components/ui/modal'
import { createManualEvidence, listEvidence, uploadEvidence } from '@/services/delivery-service'
import { formatDateTime } from '@/shared/lib/datetime'
import type { EvidenceView } from '@/shared/types/delivery'

const evidenceTypeLabels: Record<string, string> = {
  ci_report: 'CI 报告',
  document: '文档',
  git_commit: 'Git 提交',
  manual: '人工说明',
  recording: '录屏',
  screenshot: '截图',
  test_report: '测试报告',
}

/** 证据时间线面板，按时间倒序展示交付的证据并支持上传。 */
export function EvidencePanel({ deliveryId }: { deliveryId: string }) {
  const [isUploadOpen, setIsUploadOpen] = useState(false)
  const queryClient = useQueryClient()
  const { data, isLoading } = useQuery({ queryKey: ['evidence', deliveryId], queryFn: () => listEvidence(deliveryId) })
  const items = data?.items ?? []

  async function refresh() {
    await queryClient.invalidateQueries({ queryKey: ['evidence', deliveryId] })
  }

  return (
    <div className="space-y-4">
      <header className="flex items-center justify-between">
        <h2 className="text-lg font-medium">证据时间线</h2>
        <Button size="sm" onClick={() => setIsUploadOpen(true)}>
          <FileUp className="size-4" />
          登记证据
        </Button>
      </header>
      {isLoading && <p className="text-sm text-muted-foreground">加载中…</p>}
      {!isLoading && items.length === 0 && (
        <Card><CardContent className="py-10 text-center text-sm text-muted-foreground">尚无证据。上传截图、测试报告或人工说明，让验收条件有据可依。</CardContent></Card>
      )}
      <ol className="relative space-y-3 border-l pl-5">
        {items.map((item) => (
          <EvidenceTimelineItem item={item} key={item.id} />
        ))}
      </ol>
      {isUploadOpen && (
        <Modal onClose={() => setIsUploadOpen(false)} title="登记证据">
          <EvidenceForm deliveryId={deliveryId} onCreated={async () => { setIsUploadOpen(false); await refresh() }} />
        </Modal>
      )}
    </div>
  )
}

/** 时间线上的单条证据，展示类型、摘要与来源。 */
function EvidenceTimelineItem({ item }: { item: EvidenceView }) {
  return (
    <li className="relative">
      <span className="absolute -left-[1.6rem] top-2 size-2.5 rounded-full bg-primary" />
      <Card>
        <CardContent className="space-y-2">
          <div className="flex items-center justify-between gap-3">
            <span className="font-medium">{item.title}</span>
            <Badge>{evidenceTypeLabels[item.type] ?? item.type}</Badge>
          </div>
          {item.note && <p className="text-sm text-muted-foreground">{item.note}</p>}
          <div className="flex flex-wrap items-center gap-4 text-xs text-muted-foreground">
            <span>{formatDateTime(item.occurredAt)}</span>
            {item.source && <span>来源：{item.source}</span>}
            {item.sha256 && <span className="truncate">摘要：{item.sha256.slice(0, 16)}…</span>}
            {item.fileKey && (
              <span className="inline-flex items-center gap-1 text-primary">
                <Link2 className="size-3" />
                含附件
              </span>
            )}
          </div>
        </CardContent>
      </Card>
    </li>
  )
}

/** 证据登记表单，支持附件上传与纯文字说明两种方式。 */
function EvidenceForm({ deliveryId, onCreated }: { deliveryId: string; onCreated: () => Promise<void> }) {
  const fileInputRef = useRef<HTMLInputElement>(null)
  const [title, setTitle] = useState('')
  const [type, setType] = useState('screenshot')
  const [source, setSource] = useState('')
  const [note, setNote] = useState('')
  const [file, setFile] = useState<File | null>(null)
  const [error, setError] = useState('')
  const [isSubmitting, setIsSubmitting] = useState(false)

  async function onSubmit(event: React.FormEvent) {
    event.preventDefault()
    setError('')
    setIsSubmitting(true)
    try {
      if (file) {
        const formData = new FormData()
        formData.append('file', file)
        formData.append('title', title)
        formData.append('type', type)
        formData.append('source', source)
        formData.append('note', note)
        await uploadEvidence(deliveryId, formData)
      } else {
        await createManualEvidence(deliveryId, { note, source, title, type })
      }
      await onCreated()
    } catch (err) {
      setError(err instanceof Error ? err.message : '登记失败')
    } finally {
      setIsSubmitting(false)
    }
  }

  return (
    <form className="space-y-4" onSubmit={onSubmit}>
      <Input label="证据标题" required value={title} onChange={(e) => setTitle(e.target.value)} placeholder="例如：首页深色模式截图" />
      <Select label="证据类型" value={type} onChange={(e) => setType(e.target.value)}>
        {Object.entries(evidenceTypeLabels).map(([value, label]) => (
          <option key={value} value={value}>{label}</option>
        ))}
      </Select>
      <Input label="来源" value={source} onChange={(e) => setSource(e.target.value)} placeholder="选填，例如：Jenkins #42" />
      <Textarea label="说明" required value={note} onChange={(e) => setNote(e.target.value)} placeholder="描述这份证据证明了什么" />
      <div className="space-y-2">
        <span className="block text-xs font-medium text-muted-foreground">附件（可选）</span>
        <input
          className="block w-full text-sm text-muted-foreground file:mr-3 file:rounded-md file:border-0 file:bg-surface-muted file:px-3 file:py-2 file:text-sm file:text-foreground"
          ref={fileInputRef}
          type="file"
          onChange={(e) => setFile(e.target.files?.[0] ?? null)}
        />
      </div>
      {error && <p className="rounded-md bg-danger/10 px-3 py-2 text-sm text-danger">{error}</p>}
      <Button className="w-full" disabled={isSubmitting} type="submit">{isSubmitting ? '登记中…' : '登记证据'}</Button>
    </form>
  )
}
