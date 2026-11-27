import { useQuery } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { ArrowLeft } from 'lucide-react'
import { useState } from 'react'

import { Badge } from '@/components/ui/badge'
import { toneForStatus } from '@/components/ui/badge-tone'
import { Card, CardContent } from '@/components/ui/card'
import { ChangePanel } from '@/features/changes/change-panel'
import { DeliveryPanel } from '@/features/deliveries/delivery-panel'
import { IntegrationPanel } from '@/features/integration/integration-panel'
import { MilestonePanel } from '@/features/milestones/milestone-panel'
import { BaselinePanel } from '@/features/scope/baseline-panel'
import { MemberPanel } from '@/features/projects/member-panel'
import { ProjectActions } from '@/features/projects/project-actions'
import { fetchOverview, fetchProject } from '@/services/project-service'
import { formatDate } from '@/shared/lib/datetime'

type TabKey = 'baselines' | 'changes' | 'deliveries' | 'integration' | 'members' | 'milestones' | 'overview'

const tabs: Array<{ key: TabKey; label: string }> = [
  { key: 'overview', label: '概览' },
  { key: 'baselines', label: '需求基线' },
  { key: 'changes', label: '变更申请' },
  { key: 'milestones', label: '里程碑' },
  { key: 'deliveries', label: '交付版本' },
  { key: 'integration', label: '自动化集成' },
  { key: 'members', label: '项目成员' },
]

/** 项目详情页，按任务路径组织需求、变更与里程碑。 */
export function ProjectDetailPage({ projectId }: { projectId: string }) {
  const [activeTab, setActiveTab] = useState<TabKey>('overview')
  const { data, isLoading, error } = useQuery({ queryKey: ['project', projectId], queryFn: () => fetchProject(projectId) })

  if (isLoading) {
    return <p className="text-sm text-muted-foreground">加载项目…</p>
  }
  if (error || !data) {
    return <p className="text-sm text-danger">加载项目失败，请返回列表重试。</p>
  }

  const project = data.project

  return (
    <section className="space-y-6">
      <div className="space-y-4">
        <Link className="inline-flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground" to="/">
          <ArrowLeft className="size-4" />
          返回项目列表
        </Link>
        <header className="flex items-start justify-between gap-4">
          <div className="space-y-1">
            <div className="flex items-center gap-3">
              <h1 className="text-2xl font-semibold tracking-tight">{project.name}</h1>
              <Badge tone={toneForStatus(project.status)}>{project.status}</Badge>
            </div>
            <p className="text-sm text-muted-foreground">
              客户：{project.clientName} · 周期：{formatDate(project.startDate)} 至 {formatDate(project.endDate)}
            </p>
          </div>
          <ProjectActions project={project} />
        </header>
      </div>

      <nav className="flex gap-1 border-b">
        {tabs.map((tab) => (
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
        {activeTab === 'overview' && <OverviewTab projectId={projectId} project={project} />}
        {activeTab === 'baselines' && <BaselinePanel projectId={projectId} />}
        {activeTab === 'changes' && <ChangePanel projectId={projectId} />}
        {activeTab === 'milestones' && <MilestonePanel projectId={projectId} />}
        {activeTab === 'deliveries' && <DeliveryPanel projectId={projectId} />}
        {activeTab === 'integration' && <IntegrationPanel projectId={projectId} />}
        {activeTab === 'members' && <MemberPanel projectId={projectId} />}
      </div>
    </section>
  )
}

/** 概览标签页，展示项目统计与描述。 */
function OverviewTab({ project, projectId }: { project: { description: string; id: string }; projectId: string }) {
  const { data } = useQuery({ queryKey: ['overview', projectId], queryFn: () => fetchOverview(projectId) })
  const stats = [
    { label: '需求条目', value: data?.requirementCount ?? 0 },
    { label: '基线版本', value: data?.baselineCount ?? 0 },
    { label: '变更申请', value: data?.changeRequestCount ?? 0 },
    { label: '里程碑', value: data?.milestoneCount ?? 0 },
  ]

  return (
    <div className="space-y-4">
      <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
        {stats.map((stat) => (
          <Card key={stat.label}>
            <CardContent className="space-y-1">
              <p className="text-sm text-muted-foreground">{stat.label}</p>
              <p className="text-3xl font-semibold tabular-nums">{stat.value}</p>
            </CardContent>
          </Card>
        ))}
      </div>
      <Card>
        <CardContent className="space-y-2">
          <h2 className="font-medium">项目描述</h2>
          <p className="text-sm text-muted-foreground">{project.description || '暂无描述。'}</p>
        </CardContent>
      </Card>
    </div>
  )
}
