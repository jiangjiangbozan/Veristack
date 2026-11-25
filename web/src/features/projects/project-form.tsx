import { useState, type FormEvent } from 'react'

import { Button } from '@/components/ui/button'
import { Input, Textarea } from '@/components/ui/field'
import { createProject } from '@/services/project-service'
import type { ProjectView } from '@/shared/types/domain'

interface ProjectFormProps {
  onCreated: (project: ProjectView) => void
}

/** 项目创建表单，字段与后端 CreateProjectRequest 对应。 */
export function ProjectForm({ onCreated }: ProjectFormProps) {
  const [name, setName] = useState('')
  const [clientName, setClientName] = useState('')
  const [contractNo, setContractNo] = useState('')
  const [description, setDescription] = useState('')
  const [startDate, setStartDate] = useState('')
  const [endDate, setEndDate] = useState('')
  const [error, setError] = useState('')
  const [isSubmitting, setIsSubmitting] = useState(false)

  async function onSubmit(event: FormEvent) {
    event.preventDefault()
    setError('')
    setIsSubmitting(true)
    try {
      const result = await createProject({ clientName, contractNo, description, endDate, name, startDate })
      onCreated(result.project)
    } catch (err) {
      setError(err instanceof Error ? err.message : '创建失败')
    } finally {
      setIsSubmitting(false)
    }
  }

  return (
    <form className="space-y-4" onSubmit={onSubmit}>
      <Input label="项目名称" required value={name} onChange={(e) => setName(e.target.value)} placeholder="例如：官网改版" />
      <Input label="客户名称" required value={clientName} onChange={(e) => setClientName(e.target.value)} placeholder="例如：客户 A" />
      <Input label="合同编号" value={contractNo} onChange={(e) => setContractNo(e.target.value)} placeholder="选填" />
      <div className="grid grid-cols-2 gap-4">
        <Input label="开始日期" type="date" value={startDate} onChange={(e) => setStartDate(e.target.value)} />
        <Input label="结束日期" type="date" value={endDate} onChange={(e) => setEndDate(e.target.value)} />
      </div>
      <Textarea label="项目描述" value={description} onChange={(e) => setDescription(e.target.value)} placeholder="简要说明项目目标与范围" />
      {error && <p className="rounded-md bg-danger/10 px-3 py-2 text-sm text-danger">{error}</p>}
      <Button className="w-full" disabled={isSubmitting} type="submit">
        {isSubmitting ? '创建中…' : '创建项目'}
      </Button>
    </form>
  )
}
