import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { CheckCircle2, CircleSlash, FileText, ShieldCheck } from 'lucide-react'
import { useState } from 'react'

import { Alert } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input, Textarea } from '@/components/ui/field'
import { requestVerificationCode, verifyCustomerEmail } from '@/services/acceptance-service'
import { fetchPortalDelivery, savePortalVerdict, submitPortalDecision } from '@/services/delivery-service'
import { formatDateTime } from '@/shared/lib/datetime'
import type { AcceptanceVerdict, ManifestCriterion } from '@/shared/types/delivery'
import { ConfirmationEffects } from '@/features/acceptance/confirmation-effects'

/** 客户验收门户页面，供客户通过分享链接逐条验收并给出结论。 */
export function AcceptancePortalPage({ token }: { token: string }) {
  const [email, setEmail] = useState('')
  const [code, setCode] = useState('')
  const [isVerified, setIsVerified] = useState(false)
  const [comment, setComment] = useState('')
  const [objectionInputs, setObjectionInputs] = useState<Record<string, string>>({})
  const [result, setResult] = useState('')
  const queryClient = useQueryClient()
  const { data, isLoading, error } = useQuery({ queryKey: ['portal', token], queryFn: () => fetchPortalDelivery(token) })

  const codeMutation = useMutation({ mutationFn: () => requestVerificationCode(token, email) })
  const verifyMutation = useMutation({
    mutationFn: () => verifyCustomerEmail(token, email, code),
    onSuccess: () => setIsVerified(true),
  })
  const verdictMutation = useMutation({
    mutationFn: (input: { criterionKey: string; verdict: string }) =>
      savePortalVerdict(token, input.criterionKey, { comment: '', email, verdict: input.verdict }),
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: ['portal', token] }),
  })
  const decisionMutation = useMutation({
    mutationFn: (value: string) =>
      submitPortalDecision(token, {
        comment,
        decision: value,
        email,
        objections: Object.entries(objectionInputs)
          .filter(([, text]) => text.trim() !== '')
          .map(([criterionKey, text]) => ({ comment: text, criterionKey })),
      }),
    onSuccess: () => {
      setResult('验收结论已提交，开发团队将收到你的确认。')
      void queryClient.invalidateQueries({ queryKey: ['portal', token] })
    },
  })

  if (isLoading) {
    return <div className="flex min-h-svh items-center justify-center text-sm text-muted-foreground">正在打开验收记录…</div>
  }
  if (error || !data) {
    return (
      <div className="flex min-h-svh flex-col items-center justify-center gap-3 px-6 text-center">
        <CircleSlash className="size-8 text-muted-foreground" />
        <h1 className="text-lg font-medium">链接无效或已过期</h1>
        <p className="text-sm text-muted-foreground">请向开发团队索取新的验收链接。</p>
      </div>
    )
  }

  return (
    <div className="mx-auto min-h-svh max-w-3xl px-6 py-12">
      <header className="space-y-4 border-b pb-8">
        <div className="flex items-center gap-2 text-sm text-muted-foreground">
          <ShieldCheck className="size-4" />
          Veristack 验迹 · 交付验收
        </div>
        <div className="space-y-2">
          <h1 className="text-3xl font-semibold tracking-tight">{data.manifest.projectName}</h1>
          <p className="text-sm text-muted-foreground">
            交付版本 v{data.delivery.version} · 需求基线 v{data.manifest.baselineVersion}
          </p>
        </div>
        <Badge tone={data.delivery.status === 'approved' ? 'success' : data.delivery.status === 'rejected' ? 'danger' : 'warning'}>
          {data.delivery.status === 'approved' ? '已验收通过' : data.delivery.status === 'rejected' ? '已提出异议' : '待验收'}
        </Badge>
      </header>

      <section className="space-y-4 py-8">
        <h2 className="text-lg font-medium">验证你的身份</h2>
        <p className="text-sm text-muted-foreground">
          为让验收结论可追溯到具体的人，请先验证邮箱。验证通过后才能提交验收结论。
        </p>
        {isVerified ? (
          <div className="flex items-center gap-2 rounded-lg bg-success/10 px-3 py-2.5 text-sm text-success">
            <CheckCircle2 className="size-4" />
            邮箱 {email} 已验证，可以提交验收结论。
          </div>
        ) : (
          <VerificationPanel
            code={code}
            email={email}
            isSending={codeMutation.isPending}
            isVerifying={verifyMutation.isPending}
            hasError={verifyMutation.isError}
            onCodeChange={setCode}
            onEmailChange={setEmail}
            onRequestCode={() => codeMutation.mutate()}
            onVerify={() => verifyMutation.mutate()}
            showCode={codeMutation.isSuccess}
          />
        )}
      </section>

      <section className="space-y-4 border-t py-8">
        <h2 className="text-lg font-medium">验收条件</h2>
        <p className="text-sm text-muted-foreground">请逐条判断交付内容是否满足下列条件，判定结果会记录验收人与时间。</p>
        <ul className="space-y-3">
          {data.manifest.criteria.map((criterion) => (
            <CriterionCard
              criterion={criterion}
              disabled={!isVerified}
              key={criterion.key}
              onObjection={(value) => setObjectionInputs((current) => ({ ...current, [criterion.key]: value }))}
              onVerdict={(value) => verdictMutation.mutate({ criterionKey: criterion.key, verdict: value })}
              verdict={data.verdicts.find((item) => item.criterionKey === criterion.key)}
            />
          ))}
        </ul>
      </section>

      <section className="space-y-4 border-t py-8">
        <h2 className="text-lg font-medium">给出最终结论</h2>
        <ConfirmationEffects />
        <Textarea label="总体说明" value={comment} onChange={(e) => setComment(e.target.value)} placeholder="简要说明你的验收意见" />
        <div className="flex flex-col gap-3 sm:flex-row">
          <Button disabled={!isVerified || decisionMutation.isPending} onClick={() => decisionMutation.mutate('approve')}>
            确认验收通过
          </Button>
          <Button disabled={!isVerified || decisionMutation.isPending} variant="outline" onClick={() => decisionMutation.mutate('reject')}>
            提出异议
          </Button>
        </div>
        {decisionMutation.isError && (
          <Alert title="提交失败" tone="danger">请稍后重试，或联系项目方重新获取验收链接。</Alert>
        )}
        {result && (
          <div className="flex items-center gap-2 rounded-md bg-success/10 px-3 py-2 text-sm text-success">
            <CheckCircle2 className="size-4" />
            {result}
          </div>
        )}
      </section>

      <section className="space-y-3 border-t py-8">
        <h2 className="flex items-center gap-2 text-lg font-medium">
          <FileText className="size-4" />
          交付证据
        </h2>
        {data.evidence.length === 0 && <p className="text-sm text-muted-foreground">本次交付尚未附带证据材料。</p>}
        <ul className="space-y-2">
          {data.evidence.map((item) => (
            <li className="rounded-lg bg-surface-muted p-3" key={item.id}>
              <p className="text-sm font-medium">{item.title}</p>
              {item.note && <p className="mt-1 text-sm text-muted-foreground">{item.note}</p>}
              <p className="mt-1 text-xs text-muted-foreground">{formatDateTime(item.occurredAt)}</p>
            </li>
          ))}
        </ul>
      </section>
    </div>
  )
}

interface VerificationPanelProps {
  code: string
  email: string
  hasError: boolean
  isSending: boolean
  isVerifying: boolean
  onCodeChange: (value: string) => void
  onEmailChange: (value: string) => void
  onRequestCode: () => void
  onVerify: () => void
  showCode: boolean
}

/** 邮箱验证面板，先获取验证码再完成验证。 */
function VerificationPanel({
  code,
  email,
  hasError,
  isSending,
  isVerifying,
  onCodeChange,
  onEmailChange,
  onRequestCode,
  onVerify,
  showCode,
}: VerificationPanelProps) {
  return (
    <div className="space-y-3 rounded-lg border p-4">
      <div className="flex flex-col gap-3 sm:flex-row">
        <Input
          className="sm:flex-1"
          label="邮箱"
          required
          type="email"
          value={email}
          onChange={(e) => onEmailChange(e.target.value)}
          placeholder="client@example.com"
        />
        <Button className="sm:self-end" disabled={!email || isSending} onClick={onRequestCode} variant="outline">
          {isSending ? '发送中…' : '获取验证码'}
        </Button>
      </div>
      {showCode && (
        <div className="flex flex-col gap-3 sm:flex-row">
          <Input
            className="sm:flex-1"
            label="验证码"
            required
            value={code}
            onChange={(e) => onCodeChange(e.target.value)}
            placeholder="6 位数字"
          />
          <Button className="sm:self-end" disabled={code.length !== 6 || isVerifying} onClick={onVerify}>
            {isVerifying ? '验证中…' : '完成验证'}
          </Button>
        </div>
      )}
      {hasError && <p className="text-sm text-danger">验证码无效或已过期，请重新获取。</p>}
    </div>
  )
}

/** 单条验收条件卡片，提供通过、不通过与异议填写。 */
function CriterionCard({
  criterion,
  disabled,
  onObjection,
  onVerdict,
  verdict,
}: {
  criterion: ManifestCriterion
  disabled: boolean
  onObjection: (value: string) => void
  onVerdict: (value: string) => void
  verdict?: AcceptanceVerdict
}) {
  const [objection, setObjection] = useState('')

  return (
    <li className="space-y-3 rounded-lg border p-4">
      <div className="flex items-start justify-between gap-3">
        <div className="space-y-1">
          <span className="text-xs text-muted-foreground">{criterion.requirementCode}</span>
          <p className="text-sm font-medium">{criterion.description}</p>
        </div>
        {verdict && (
          <Badge tone={verdict.verdict === 'pass' ? 'success' : 'danger'}>
            {verdict.verdict === 'pass' ? '通过' : '不通过'}
          </Badge>
        )}
      </div>
      <div className="flex flex-wrap gap-2">
        <Button disabled={disabled} onClick={() => onVerdict('pass')} size="sm" variant={verdict?.verdict === 'pass' ? 'default' : 'outline'}>
          符合
        </Button>
        <Button disabled={disabled} onClick={() => onVerdict('fail')} size="sm" variant={verdict?.verdict === 'fail' ? 'default' : 'outline'}>
          不符合
        </Button>
      </div>
      <Textarea
        disabled={disabled}
        label="提出异议（可选）"
        value={objection}
        onChange={(e) => setObjection(e.target.value)}
        onBlur={() => objection.trim() && onObjection(objection)}
        placeholder="若不符合，请说明具体问题"
      />
    </li>
  )
}
