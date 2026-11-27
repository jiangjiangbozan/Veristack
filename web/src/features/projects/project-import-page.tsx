import { useMutation, useQueryClient } from '@tanstack/react-query'
import { Link, useNavigate } from '@tanstack/react-router'
import { ArrowLeft, CheckCircle2, ClipboardCopy, Download, FileJson, Upload } from 'lucide-react'
import { useState, type ChangeEvent } from 'react'

import { Alert } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Textarea } from '@/components/ui/field'
import { importProject, type ImportProjectPayload } from '@/services/project-service'

const templatePayload: ImportProjectPayload = {
  milestones: [
    {
      name: '首个可用版本',
      requirementCodes: ['REQ-001', 'REQ-002'],
      targetDate: '2027-03-15',
    },
  ],
  project: {
    clientName: '天津星河科技有限公司',
    contractNo: 'HT-2027-001',
    description: '官网信息架构与视觉升级，覆盖 PC 与移动端。',
    endDate: '2027-05-31',
    name: '客户官网改版',
    startDate: '2026-12-01',
  },
  requirements: [
    {
      acceptanceCriteria: ['首页首屏 2 秒内加载完成', '核心入口在移动端可单手操作'],
      code: 'REQ-001',
      description: '重新组织首页导航、核心卖点与转化入口。',
      priority: 'high',
      title: '首页信息架构',
    },
    {
      acceptanceCriteria: ['所有页面在 1440px 与 390px 宽度下无横向滚动'],
      code: 'REQ-002',
      description: '统一字体、间距、色彩与组件样式。',
      priority: 'medium',
      title: '视觉规范落地',
    },
  ],
}

const templateText = JSON.stringify(templatePayload, null, 2)
const validPriorities = new Set(['high', 'medium', 'low'])

/** 已有项目导入页，同时承担导入指南与 JSON 上传入口。 */
export function ProjectImportPage() {
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const [jsonText, setJsonText] = useState('')
  const [fileName, setFileName] = useState('')
  const [error, setError] = useState('')
  const [notice, setNotice] = useState('')
  const mutation = useMutation({
    mutationFn: importProject,
    onSuccess: async (result) => {
      await queryClient.invalidateQueries({ queryKey: ['projects'] })
      await navigate({ to: '/projects/$projectId', params: { projectId: result.project.id } })
    },
  })

  async function onFileChange(event: ChangeEvent<HTMLInputElement>) {
    const file = event.target.files?.[0]
    if (!file) {
      return
    }
    setFileName(file.name)
    setJsonText(await file.text())
    setError('')
    setNotice('')
  }

  async function onSubmit() {
    setError('')
    setNotice('')
    let payload: ImportProjectPayload
    try {
      payload = parseImportPayload(jsonText)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'JSON 格式不正确')
      return
    }
    try {
      await mutation.mutateAsync(payload)
    } catch (err) {
      setError(err instanceof Error ? err.message : '导入失败，请稍后重试')
    }
  }

  async function onCopyTemplate() {
    setError('')
    try {
      await navigator.clipboard.writeText(templateText)
      setNotice('模板已复制到剪贴板。')
    } catch {
      setError('浏览器未允许复制，请手动选择模板内容。')
    }
  }

  function onDownloadTemplate() {
    const blob = new Blob([templateText], { type: 'application/json;charset=utf-8' })
    const url = URL.createObjectURL(blob)
    const anchor = document.createElement('a')
    anchor.href = url
    anchor.download = 'veristack-project-import-template.json'
    anchor.click()
    URL.revokeObjectURL(url)
  }

  return (
    <div className="mx-auto w-full max-w-4xl space-y-8 px-6 py-10">
      <Link className="inline-flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground" to="/">
        <ArrowLeft className="size-4" />
        返回项目列表
      </Link>

      <header className="space-y-3">
        <div className="flex items-center gap-2 text-sm text-muted-foreground">
          <FileJson className="size-4 text-primary" />
          项目导入指南
        </div>
        <h1 className="text-3xl font-semibold tracking-tight">导入已有项目</h1>
        <p className="max-w-2xl text-sm leading-6 text-muted-foreground">
          如果你已经有一份项目清单，不需要在页面上一条一条录入。把项目基本信息、需求与验收条件整理成一份 JSON，上传后系统会一次创建项目、v1 需求基线和里程碑。当前导入不包含项目成员与历史交付证据，导入后可在项目页继续补充。
        </p>
      </header>

      <section className="grid gap-3 sm:grid-cols-3">
        {[
          { index: '01', text: '下载或复制右侧模板' },
          { index: '02', text: '替换项目、需求与里程碑内容' },
          { index: '03', text: '上传 JSON，确认导入结果' },
        ].map((step) => (
          <Card key={step.index}>
            <CardContent className="flex items-center gap-3">
              <span className="text-lg font-semibold text-primary">{step.index}</span>
              <span className="text-sm text-muted-foreground">{step.text}</span>
            </CardContent>
          </Card>
        ))}
      </section>

      <section className="space-y-3">
        <h2 className="text-xl font-semibold tracking-tight">导入内容</h2>
        <Card>
          <CardContent className="space-y-4">
            <FieldRow label="项目基本信息" value="项目名称、客户名称、合同编号、项目描述、开始与结束日期；项目名称和客户名称为必填。" />
            <FieldRow label="需求与验收条件" value="每条需求包含编号、标题、描述、优先级与验收条件数组；需求编号在同一次导入内不能重复。" />
            <FieldRow label="里程碑（可选）" value="用需求编号关联已导入的需求；目标日期使用 YYYY-MM-DD 格式，引用不存在的编号会取消整次导入。" />
          </CardContent>
        </Card>
      </section>

      <section className="space-y-3">
        <div className="flex flex-wrap items-end justify-between gap-3">
          <div>
            <h2 className="text-xl font-semibold tracking-tight">JSON 模板</h2>
            <p className="mt-1 text-sm text-muted-foreground">只保留实际需要的字段，未填写的描述、合同编号和日期可以留空。</p>
          </div>
          <div className="flex gap-2">
            <Button size="sm" variant="outline" onClick={() => void onCopyTemplate()}>
              <ClipboardCopy className="size-4" />
              复制模板
            </Button>
            <Button size="sm" variant="outline" onClick={onDownloadTemplate}>
              <Download className="size-4" />
              下载模板
            </Button>
          </div>
        </div>
        <pre className="max-h-96 overflow-auto rounded-[var(--radius-card)] border bg-surface-muted p-4 text-xs leading-6 text-foreground">
          {templateText}
        </pre>
      </section>

      <section className="space-y-4">
        <div>
          <h2 className="text-xl font-semibold tracking-tight">上传 JSON</h2>
          <p className="mt-1 text-sm text-muted-foreground">支持 .json 文件，也可以直接粘贴内容。导入成功后会自动打开项目详情。</p>
        </div>
        <Card>
          <CardContent className="space-y-4">
            <label className="flex cursor-pointer flex-col items-center gap-2 rounded-[var(--radius-card)] border border-dashed px-5 py-8 text-center transition-colors hover:bg-surface-muted">
              <Upload className="size-6 text-primary" />
              <span className="text-sm font-medium">{fileName || '选择 JSON 文件'}</span>
              <span className="text-xs text-muted-foreground">文件只在浏览器中读取，不会上传到第三方服务</span>
              <input accept=".json,application/json" className="sr-only" type="file" onChange={(event) => void onFileChange(event)} />
            </label>
            <Textarea
              className="min-h-48 font-mono text-xs"
              label="或粘贴 JSON 内容"
              placeholder="粘贴导出的项目 JSON，或使用上方模板。"
              value={jsonText}
              onChange={(event) => setJsonText(event.target.value)}
            />
            {error && <Alert title="无法导入项目" tone="danger">{error}</Alert>}
            {notice && <p className="rounded-md bg-success/10 px-3 py-2 text-sm text-success">{notice}</p>}
            <div className="flex flex-wrap items-center justify-between gap-3">
              <p className="text-xs text-muted-foreground">一次最多导入 500 条需求、100 个里程碑。</p>
              <Button disabled={mutation.isPending || jsonText.trim() === ''} onClick={() => void onSubmit()}>
                <CheckCircle2 className="size-4" />
                {mutation.isPending ? '导入中…' : '开始导入'}
              </Button>
            </div>
          </CardContent>
        </Card>
      </section>

      <section className="space-y-3">
        <h2 className="text-xl font-semibold tracking-tight">通过 API 导入</h2>
        <p className="text-sm leading-6 text-muted-foreground">
          自动化脚本可以调用同一个接口。请求体与页面模板一致，接口会在事务内创建项目、基线、需求与里程碑，任一校验失败都会整体回滚。
        </p>
        <pre className="overflow-auto rounded-[var(--radius-card)] border bg-surface-muted p-4 text-xs leading-6 text-foreground">
{`curl -X POST http://localhost:8080/api/v1/projects/import \\
  -H "Authorization: Bearer <access-token>" \\
  -H "Content-Type: application/json" \\
  --data-binary @project-import.json`}
        </pre>
      </section>
    </div>
  )
}

/** 文档式字段说明行。 */
function FieldRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="grid gap-1 border-b pb-3 last:border-b-0 last:pb-0 sm:grid-cols-[140px_1fr] sm:gap-4">
      <span className="text-sm font-medium">{label}</span>
      <span className="text-sm leading-6 text-muted-foreground">{value}</span>
    </div>
  )
}

/** 解析并校验导入 JSON，返回后端可直接接收的规范化结构。 */
function parseImportPayload(raw: string): ImportProjectPayload {
  let parsed: unknown
  try {
    parsed = JSON.parse(raw)
  } catch {
    throw new Error('JSON 解析失败，请检查逗号、引号与括号是否完整。')
  }
  if (!isRecord(parsed)) {
    throw new Error('导入内容必须是一个 JSON 对象。')
  }
  const projectValue = parsed.project
  if (!isRecord(projectValue)) {
    throw new Error('缺少 project 对象。')
  }
  const projectName = readString(projectValue.name)
  const clientName = readString(projectValue.clientName)
  if (!projectName || !clientName) {
    throw new Error('project.name 与 project.clientName 为必填字段。')
  }
  const requirements = readRequirements(parsed.requirements)
  const milestones = readMilestones(parsed.milestones)
  if (requirements.length === 0 && milestones.some((item) => item.requirementCodes.length > 0)) {
    throw new Error('里程碑引用了需求编号，但 requirements 为空。')
  }
  const codes = new Set(requirements.map((item) => item.code))
  for (const milestone of milestones) {
    for (const code of milestone.requirementCodes) {
      if (!codes.has(code)) {
        throw new Error(`里程碑「${milestone.name}」引用了不存在的需求编号 ${code}。`)
      }
    }
  }
  return {
    milestones,
    project: {
      clientName,
      contractNo: readString(projectValue.contractNo),
      description: readString(projectValue.description),
      endDate: readString(projectValue.endDate),
      name: projectName,
      startDate: readString(projectValue.startDate),
    },
    requirements,
  }
}

/** 读取并规范化需求数组。 */
function readRequirements(value: unknown): ImportProjectPayload['requirements'] {
  if (value === undefined || value === null) {
    return []
  }
  if (!Array.isArray(value)) {
    throw new Error('requirements 必须是数组。')
  }
  const seenCodes = new Set<string>()
  return value.map((item, index) => {
    if (!isRecord(item)) {
      throw new Error(`requirements[${index}] 必须是对象。`)
    }
    const code = readString(item.code)
    const title = readString(item.title)
    if (!code || !title) {
      throw new Error(`requirements[${index}] 缺少 code 或 title。`)
    }
    if (seenCodes.has(code)) {
      throw new Error(`需求编号 ${code} 重复。`)
    }
    seenCodes.add(code)
    const priority = readString(item.priority) || 'medium'
    if (!validPriorities.has(priority)) {
      throw new Error(`需求 ${code} 的 priority 只能是 high、medium 或 low。`)
    }
    return {
      acceptanceCriteria: readStringArray(item.acceptanceCriteria),
      code,
      description: readString(item.description),
      priority,
      title,
    }
  })
}

/** 读取并规范化里程碑数组。 */
function readMilestones(value: unknown): ImportProjectPayload['milestones'] {
  if (value === undefined || value === null) {
    return []
  }
  if (!Array.isArray(value)) {
    throw new Error('milestones 必须是数组。')
  }
  return value.map((item, index) => {
    if (!isRecord(item)) {
      throw new Error(`milestones[${index}] 必须是对象。`)
    }
    const name = readString(item.name)
    if (!name) {
      throw new Error(`milestones[${index}] 缺少 name。`)
    }
    return {
      name,
      requirementCodes: readStringArray(item.requirementCodes),
      targetDate: readString(item.targetDate),
    }
  })
}

/** 判断未知值是否为普通对象。 */
function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value)
}

/** 读取字符串字段，非字符串按空串处理。 */
function readString(value: unknown): string {
  return typeof value === 'string' ? value.trim() : ''
}

/** 读取字符串数组，过滤空值与非字符串。 */
function readStringArray(value: unknown): string[] {
  return Array.isArray(value) ? value.filter((item): item is string => typeof item === 'string' && item.trim() !== '').map((item) => item.trim()) : []
}
