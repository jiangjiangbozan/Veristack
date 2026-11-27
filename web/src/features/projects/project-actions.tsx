import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useNavigate } from '@tanstack/react-router'
import { Archive, Pencil, Trash2 } from 'lucide-react'
import { useState, type FormEvent } from 'react'

import { Alert } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Input, Select, Textarea } from '@/components/ui/field'
import { Modal } from '@/components/ui/modal'
import { deleteProject, updateProject } from '@/services/project-service'
import type { ProjectView } from '@/shared/types/domain'

interface ProjectDraft {
  clientName: string
  contractNo: string
  description: string
  endDate: string
  name: string
  startDate: string
  status: string
}

/** 项目页头操作，提供编辑、归档与删除入口。 */
export function ProjectActions({ project }: { project: ProjectView }) {
  const [isEditOpen, setIsEditOpen] = useState(false)
  const [isArchiveOpen, setIsArchiveOpen] = useState(false)
  const [isDeleteOpen, setIsDeleteOpen] = useState(false)
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const archiveMutation = useMutation({
    mutationFn: () => updateProject(project.id, {
      archived: true,
      clientName: project.clientName,
      contractNo: project.contractNo,
      description: project.description,
      endDate: project.endDate,
      name: project.name,
      startDate: project.startDate,
      status: 'archived',
    }),
    onSuccess: async () => {
      setIsArchiveOpen(false)
      await queryClient.invalidateQueries({ queryKey: ['project', project.id] })
      await queryClient.invalidateQueries({ queryKey: ['overview', project.id] })
      await queryClient.invalidateQueries({ queryKey: ['projects'] })
    },
  })
  const deleteMutation = useMutation({
    mutationFn: () => deleteProject(project.id),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ['projects'] })
      await navigate({ to: '/' })
    },
  })

  return (
    <div className="flex flex-wrap items-center gap-2">
      <Button onClick={() => setIsEditOpen(true)} size="sm" variant="outline">
        <Pencil className="size-4" />
        编辑项目
      </Button>
      {project.status !== 'archived' && (
        <Button disabled={archiveMutation.isPending} onClick={() => setIsArchiveOpen(true)} size="sm" variant="outline">
          <Archive className="size-4" />
          归档项目
        </Button>
      )}
      <Button
        className="text-danger hover:bg-danger/10"
        disabled={deleteMutation.isPending}
        onClick={() => setIsDeleteOpen(true)}
        size="sm"
        variant="outline"
      >
        <Trash2 className="size-4" />
        删除项目
      </Button>
      {isEditOpen && (
        <Modal onClose={() => setIsEditOpen(false)} title="编辑项目">
          <ProjectEditForm
            onSaved={async () => {
              setIsEditOpen(false)
              await queryClient.invalidateQueries({ queryKey: ['project', project.id] })
              await queryClient.invalidateQueries({ queryKey: ['overview', project.id] })
            }}
            project={project}
          />
        </Modal>
      )}
      {isArchiveOpen && (
        <Modal onClose={() => setIsArchiveOpen(false)} title="归档项目">
          <div className="space-y-4">
            <p className="text-sm text-muted-foreground">
              归档后项目会从进行中列表淡出，历史基线、交付与审计记录仍保留。确认归档「{project.name}」？
            </p>
            {archiveMutation.isError && (
              <Alert title="无法归档项目" tone="danger">
                {archiveMutation.error instanceof Error ? archiveMutation.error.message : '归档失败'}
              </Alert>
            )}
            <div className="flex justify-end gap-2">
              <Button onClick={() => setIsArchiveOpen(false)} size="sm" variant="ghost">取消</Button>
              <Button disabled={archiveMutation.isPending} onClick={() => archiveMutation.mutate()} size="sm" variant="outline">
                {archiveMutation.isPending ? '归档中…' : '确认归档'}
              </Button>
            </div>
          </div>
        </Modal>
      )}
      {isDeleteOpen && (
        <Modal onClose={() => setIsDeleteOpen(false)} title="删除项目">
          <div className="space-y-4">
            <p className="text-sm text-muted-foreground">
              删除后项目将从列表与所有业务入口隐藏，历史审计与证据仍然保留。确认删除「{project.name}」？
            </p>
            {deleteMutation.isError && (
              <Alert title="无法删除项目" tone="danger">
                {deleteMutation.error instanceof Error ? deleteMutation.error.message : '删除失败'}
              </Alert>
            )}
            <div className="flex justify-end gap-2">
              <Button onClick={() => setIsDeleteOpen(false)} size="sm" variant="ghost">取消</Button>
              <Button
                className="text-danger hover:bg-danger/10"
                disabled={deleteMutation.isPending}
                onClick={() => deleteMutation.mutate()}
                size="sm"
                variant="outline"
              >
                {deleteMutation.isPending ? '删除中…' : '确认删除'}
              </Button>
            </div>
          </div>
        </Modal>
      )}
    </div>
  )
}

/** 编辑项目表单，保存后刷新项目详情。 */
function ProjectEditForm({ onSaved, project }: { onSaved: () => Promise<void>; project: ProjectView }) {
  const [draft, setDraft] = useState<ProjectDraft>({
    clientName: project.clientName,
    contractNo: project.contractNo,
    description: project.description,
    endDate: project.endDate,
    name: project.name,
    startDate: project.startDate,
    status: project.status,
  })
  const [error, setError] = useState('')
  const [isSubmitting, setIsSubmitting] = useState(false)

  function updateField<K extends keyof ProjectDraft>(key: K, value: ProjectDraft[K]) {
    setDraft((current) => ({ ...current, [key]: value }))
  }

  async function onSubmit(event: FormEvent) {
    event.preventDefault()
    if (!draft.name.trim() || !draft.clientName.trim()) {
      setError('请填写项目名称和客户名称。')
      return
    }
    setError('')
    setIsSubmitting(true)
    try {
      await updateProject(project.id, {
        ...draft,
        archived: draft.status === 'archived',
      })
      await onSaved()
    } catch (err) {
      setError(err instanceof Error ? err.message : '保存失败')
    } finally {
      setIsSubmitting(false)
    }
  }

  return (
    <form className="space-y-4" noValidate onSubmit={onSubmit}>
      <Input label="项目名称" required value={draft.name} onChange={(event) => updateField('name', event.target.value)} />
      <div className="grid grid-cols-2 gap-3">
        <Input label="客户名称" required value={draft.clientName} onChange={(event) => updateField('clientName', event.target.value)} />
        <Input label="合同编号" value={draft.contractNo} onChange={(event) => updateField('contractNo', event.target.value)} />
      </div>
      <div className="grid grid-cols-2 gap-3">
        <Input label="开始日期" type="date" value={draft.startDate} onChange={(event) => updateField('startDate', event.target.value)} />
        <Input label="结束日期" type="date" value={draft.endDate} onChange={(event) => updateField('endDate', event.target.value)} />
      </div>
      <Select label="项目状态" value={draft.status} onChange={(event) => updateField('status', event.target.value)}>
        <option value="active">进行中</option>
        <option value="completed">已完成</option>
        <option value="archived">已归档</option>
      </Select>
      <Textarea label="项目描述" value={draft.description} onChange={(event) => updateField('description', event.target.value)} />
      {error && <Alert title="无法保存项目" tone="danger">{error}</Alert>}
      <Button className="w-full" disabled={isSubmitting} type="submit">
        {isSubmitting ? '保存中…' : '保存修改'}
      </Button>
    </form>
  )
}
