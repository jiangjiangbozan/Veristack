import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { UserPlus, Users } from 'lucide-react'
import { useState, type FormEvent } from 'react'

import { Alert } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Input, Select } from '@/components/ui/field'
import { Modal } from '@/components/ui/modal'
import { addMember, listMembers, removeMember } from '@/services/project-service'
import type { MemberView } from '@/shared/types/domain'

const memberRoleLabels: Record<string, string> = {
  developer: '开发',
  reviewer: '评审',
}

/** 项目成员面板，支持成员列表、添加与移除。 */
export function MemberPanel({ projectId }: { projectId: string }) {
  const [isAddOpen, setIsAddOpen] = useState(false)
  const [removing, setRemoving] = useState<MemberView | null>(null)
  const queryClient = useQueryClient()
  const { data, isLoading } = useQuery({ queryKey: ['members', projectId], queryFn: () => listMembers(projectId) })
  const items = data?.items ?? []
  const removeMutation = useMutation({
    mutationFn: (memberId: string) => removeMember(projectId, memberId),
    onSuccess: async () => {
      setRemoving(null)
      await queryClient.invalidateQueries({ queryKey: ['members', projectId] })
    },
  })

  return (
    <div className="space-y-4">
      <header className="flex items-center justify-between">
        <div>
          <h2 className="flex items-center gap-2 text-lg font-medium">
            <Users className="size-5 text-primary" />
            项目成员
          </h2>
          <p className="mt-1 text-sm text-muted-foreground">成员来自当前组织，添加后按所选角色参与评审与交付。</p>
        </div>
        <Button onClick={() => setIsAddOpen(true)} size="sm">
          <UserPlus className="size-4" />
          添加成员
        </Button>
      </header>
      {isLoading && <p className="text-sm text-muted-foreground">加载中…</p>}
      {!isLoading && items.length === 0 && (
        <Card><CardContent className="py-10 text-center text-sm text-muted-foreground">尚无项目成员，添加组织内用户参与协作。</CardContent></Card>
      )}
      <div className="space-y-2">
        {items.map((member) => (
          <Card key={member.id}>
            <CardContent className="flex flex-wrap items-center justify-between gap-3">
              <div className="space-y-0.5">
                <p className="text-sm font-medium">{member.displayName || member.email}</p>
                <p className="text-xs text-muted-foreground">{member.email}</p>
              </div>
              <div className="flex items-center gap-3">
                <Badge>{memberRoleLabels[member.role] ?? member.role}</Badge>
                <Button
                  className="text-danger hover:bg-danger/10"
                  onClick={() => setRemoving(member)}
                  size="sm"
                  variant="ghost"
                >
                  移除
                </Button>
              </div>
            </CardContent>
          </Card>
        ))}
      </div>
      {isAddOpen && (
        <Modal onClose={() => setIsAddOpen(false)} title="添加项目成员">
          <AddMemberForm
            onCreated={async () => {
              setIsAddOpen(false)
              await queryClient.invalidateQueries({ queryKey: ['members', projectId] })
            }}
            projectId={projectId}
          />
        </Modal>
      )}
      {removing && (
        <Modal onClose={() => setRemoving(null)} title="移除项目成员">
          <div className="space-y-4">
            <p className="text-sm text-muted-foreground">
              确认把「{removing.displayName || removing.email}」移出项目？已产生的交付与审计记录不会删除。
            </p>
            {removeMutation.isError && (
              <Alert title="无法移除成员" tone="danger">
                {removeMutation.error instanceof Error ? removeMutation.error.message : '移除失败'}
              </Alert>
            )}
            <div className="flex justify-end gap-2">
              <Button onClick={() => setRemoving(null)} size="sm" variant="ghost">取消</Button>
              <Button
                className="text-danger hover:bg-danger/10"
                disabled={removeMutation.isPending}
                onClick={() => removeMutation.mutate(removing.id)}
                size="sm"
                variant="outline"
              >
                {removeMutation.isPending ? '移除中…' : '确认移除'}
              </Button>
            </div>
          </div>
        </Modal>
      )}
    </div>
  )
}

/** 添加成员表单，仅允许当前组织内的用户。 */
function AddMemberForm({ onCreated, projectId }: { onCreated: () => Promise<void>; projectId: string }) {
  const [email, setEmail] = useState('')
  const [role, setRole] = useState('developer')
  const [error, setError] = useState('')
  const [isSubmitting, setIsSubmitting] = useState(false)

  async function onSubmit(event: FormEvent) {
    event.preventDefault()
    const normalizedEmail = email.trim()
    if (!normalizedEmail) {
      setError('请输入成员邮箱。')
      return
    }
    if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(normalizedEmail)) {
      setError('请输入有效的邮箱地址。')
      return
    }
    setError('')
    setIsSubmitting(true)
    try {
      await addMember(projectId, normalizedEmail, role)
      await onCreated()
    } catch (err) {
      setError(err instanceof Error ? err.message : '添加失败')
    } finally {
      setIsSubmitting(false)
    }
  }

  return (
    <form className="space-y-4" noValidate onSubmit={onSubmit}>
      <Input label="成员邮箱" required type="email" value={email} onChange={(event) => setEmail(event.target.value)} placeholder="成员必须已在当前组织注册" />
      <Select label="项目角色" value={role} onChange={(event) => setRole(event.target.value)}>
        <option value="developer">开发</option>
        <option value="reviewer">评审</option>
      </Select>
      {error && <Alert title="无法添加成员" tone="danger">{error}</Alert>}
      <Button className="w-full" disabled={isSubmitting} type="submit">{isSubmitting ? '添加中…' : '添加成员'}</Button>
    </form>
  )
}
