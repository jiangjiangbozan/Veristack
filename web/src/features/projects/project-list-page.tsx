import { useNavigate } from '@tanstack/react-router'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { Plus } from 'lucide-react'
import { useState } from 'react'

import { Badge } from '@/components/ui/badge'
import { toneForStatus } from '@/components/ui/badge-tone'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Modal } from '@/components/ui/modal'
import { ProjectForm } from '@/features/projects/project-form'
import { listProjects } from '@/services/project-service'
import { formatDate } from '@/shared/lib/datetime'
import type { ProjectView } from '@/shared/types/domain'

/** 项目列表页，按用户操作路径进入项目详情。 */
export function ProjectListPage() {
  const [isCreateOpen, setIsCreateOpen] = useState(false)
  const queryClient = useQueryClient()
  const navigate = useNavigate()
  const { data, isLoading, error } = useQuery({ queryKey: ['projects'], queryFn: () => listProjects() })

  async function onCreated(project: ProjectView) {
    setIsCreateOpen(false)
    await queryClient.invalidateQueries({ queryKey: ['projects'] })
    await navigate({ to: '/projects/$projectId', params: { projectId: project.id } })
  }

  return (
    <section className="space-y-6">
      <header className="flex items-center justify-between gap-4">
        <div className="space-y-1">
          <h1 className="text-2xl font-semibold tracking-tight">项目</h1>
          <p className="text-sm text-muted-foreground">管理客户项目、需求范围与交付验收。</p>
        </div>
        <Button onClick={() => setIsCreateOpen(true)}>
          <Plus className="size-4" />
          新建项目
        </Button>
      </header>

      {isLoading && <ListSkeleton />}
      {error && <p className="text-sm text-danger">加载项目失败，请稍后重试。</p>}
      {data?.items.length === 0 && <EmptyState onCreate={() => setIsCreateOpen(true)} />}
      {data && data.items.length > 0 && (
        <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
          {data.items.map((project) => (
            <ProjectCard key={project.id} project={project} onOpen={() => navigate({ to: '/projects/$projectId', params: { projectId: project.id } })} />
          ))}
        </div>
      )}

      {isCreateOpen && (
        <Modal onClose={() => setIsCreateOpen(false)} title="新建项目">
          <ProjectForm onCreated={onCreated} />
        </Modal>
      )}
    </section>
  )
}

/** 单个项目卡片。 */
function ProjectCard({ onOpen, project }: { onOpen: () => void; project: ProjectView }) {
  return (
    <Card className="cursor-pointer transition-shadow hover:shadow-lg" onClick={onOpen}>
      <CardContent className="space-y-3">
        <div className="flex items-start justify-between gap-2">
          <h2 className="font-medium">{project.name}</h2>
          <Badge tone={toneForStatus(project.status)}>{project.status}</Badge>
        </div>
        <p className="text-sm text-muted-foreground">{project.clientName}</p>
        <div className="flex items-center justify-between text-xs text-muted-foreground">
          <span>更新于 {formatDate(project.updatedAt)}</span>
          {project.contractNo && <span>{project.contractNo}</span>}
        </div>
      </CardContent>
    </Card>
  )
}

/** 空状态引导用户创建第一个项目。 */
function EmptyState({ onCreate }: { onCreate: () => void }) {
  return (
    <Card>
      <CardContent className="flex flex-col items-center gap-4 py-16 text-center">
        <div className="size-12 rounded-full bg-surface-muted" />
        <div className="space-y-1">
          <h2 className="font-medium">尚无项目</h2>
          <p className="text-sm text-muted-foreground">创建第一个项目后，即可录入需求范围并发起变更。</p>
        </div>
        <Button variant="outline" onClick={onCreate}>新建项目</Button>
      </CardContent>
    </Card>
  )
}

/** 加载骨架屏。 */
function ListSkeleton() {
  return (
    <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
      {[1, 2, 3].map((key) => (
        <Card key={key}>
          <CardContent className="h-28 animate-pulse rounded bg-surface-muted" />
        </Card>
      ))}
    </div>
  )
}
