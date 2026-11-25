import { useMutation, useQuery } from '@tanstack/react-query'
import { ArrowLeft, GitCommitHorizontal, Hash, PackageCheck } from 'lucide-react'
import { Link, useNavigate } from '@tanstack/react-router'

import { Badge } from '@/components/ui/badge'
import { toneForStatus } from '@/components/ui/badge-tone'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { AcceptancePanel } from '@/features/acceptance/acceptance-panel'
import { EvidencePanel } from '@/features/evidence/evidence-panel'
import { exportPack, fetchDelivery, fetchManifest, type PackExportResult } from '@/services/delivery-service'
import { formatDateTime } from '@/shared/lib/datetime'
import { useState } from 'react'

/** 交付详情页，集中展示交付清单、证据时间线与客户验收。 */
export function DeliveryDetailPage({ deliveryId }: { deliveryId: string }) {
  const navigate = useNavigate()
  const [activeTab, setActiveTab] = useState<'acceptance' | 'evidence' | 'manifest'>('manifest')
  const { data: manifest } = useQuery({ queryKey: ['manifest', deliveryId], queryFn: () => fetchManifest(deliveryId) })
  const { data: delivery } = useQuery({ queryKey: ['delivery', deliveryId], queryFn: () => fetchDelivery(deliveryId) })

  return (
    <section className="space-y-6">
      <div className="space-y-4">
        <Link className="inline-flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground" to="/">
          <ArrowLeft className="size-4" />
          返回项目列表
        </Link>
        <header className="flex items-start justify-between gap-4">
          <div className="space-y-1">
            <h1 className="text-2xl font-semibold tracking-tight">交付详情</h1>
            {manifest && (
              <p className="text-sm text-muted-foreground">
                {manifest.projectName} · 交付 v{manifest.version} · 基线 v{manifest.baselineVersion}
              </p>
            )}
          </div>
          <div className="flex items-center gap-2">
            {delivery && <Badge tone={toneForStatus(delivery.delivery.status)}>{delivery.delivery.status}</Badge>}
            <Button variant="outline" onClick={() => navigate({ to: '/' })}>返回项目</Button>
          </div>
        </header>
      </div>

      <nav className="flex gap-1 border-b">
        {([
          { key: 'manifest' as const, label: '交付清单' },
          { key: 'evidence' as const, label: '证据时间线' },
          { key: 'acceptance' as const, label: '客户验收' },
        ]).map((tab) => (
          <button
            className={`-mb-px border-b-2 px-4 py-2 text-sm transition-colors ${
              activeTab === tab.key
                ? 'border-primary font-medium text-primary'
                : 'border-transparent text-muted-foreground hover:text-foreground'
            }`}
            key={tab.key}
            type="button"
            onClick={() => setActiveTab(tab.key)}
          >
            {tab.label}
          </button>
        ))}
      </nav>

      <div>
        {activeTab === 'manifest' && <ManifestTab deliveryId={deliveryId} />}
        {activeTab === 'evidence' && <EvidencePanel deliveryId={deliveryId} />}
        {activeTab === 'acceptance' && <AcceptancePanel deliveryId={deliveryId} />}
      </div>
    </section>
  )
}

/** 交付清单标签页，展示冻结的需求与验收条件。 */
function ManifestTab({ deliveryId }: { deliveryId: string }) {
  const [packResult, setPackResult] = useState<PackExportResult | null>(null)
  const { data: manifest, isLoading } = useQuery({ queryKey: ['manifest', deliveryId], queryFn: () => fetchManifest(deliveryId) })
  const packMutation = useMutation({
    mutationFn: () => exportPack(deliveryId),
    onSuccess: (result) => setPackResult(result),
  })

  if (isLoading) {
    return <p className="text-sm text-muted-foreground">加载交付清单…</p>
  }
  if (!manifest) {
    return <p className="text-sm text-danger">加载交付清单失败。</p>
  }

  return (
    <div className="space-y-4">
      <Card>
        <CardContent className="flex flex-wrap items-center gap-6 text-sm text-muted-foreground">
          <span className="inline-flex items-center gap-1.5">
            <Hash className="size-4" />
            清单摘要 {manifest.version > 0 ? '已冻结' : '未生成'}
          </span>
          {manifest.gitRef && (
            <span className="inline-flex items-center gap-1.5">
              <GitCommitHorizontal className="size-4" />
              {manifest.gitRef}
            </span>
          )}
          <Badge tone="success">不可覆盖</Badge>
          <Button className="ml-auto" disabled={packMutation.isPending} onClick={() => packMutation.mutate()} size="sm" variant="outline">
            <PackageCheck className="size-4" />
            导出核验包
          </Button>
        </CardContent>
      </Card>
      {packResult && (
        <Card>
          <CardContent className="space-y-2">
            <p className="text-sm font-medium">Evidence Pack 已生成</p>
            <p className="text-xs text-muted-foreground">清单摘要 {packResult.manifestSha256}</p>
            <p className="text-xs text-muted-foreground">
              {packResult.signed ? '已附带 Ed25519 签名，客户可使用 veristack verify 离线核验。' : '当前服务未配置签名私钥，客户只能校验内容摘要。'}
            </p>
          </CardContent>
        </Card>
      )}
      <ul className="space-y-3">
        {manifest.requirements.map((requirement) => (
          <li className="rounded-lg border p-4" key={requirement.code}>
            <div className="flex items-center justify-between gap-3">
              <span className="font-medium">{requirement.code} · {requirement.title}</span>
              <Badge>{requirement.priority}</Badge>
            </div>
            {requirement.description && <p className="mt-1 text-sm text-muted-foreground">{requirement.description}</p>}
            <ul className="mt-3 space-y-1 text-sm">
              {requirement.criteria.map((criterion) => (
                <li className="flex items-start gap-2" key={criterion.key}>
                  <span className="mt-1.5 size-1.5 shrink-0 rounded-full bg-primary" />
                  <span>{criterion.description}</span>
                </li>
              ))}
            </ul>
          </li>
        ))}
      </ul>
      <p className="text-xs text-muted-foreground">生成时间 {formatDateTime(new Date().toISOString())}</p>
    </div>
  )
}
