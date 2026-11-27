import { zodResolver } from '@hookform/resolvers/zod'
import { Link, useNavigate } from '@tanstack/react-router'
import { ArrowLeft, ShieldCheck } from 'lucide-react'
import { useState } from 'react'
import { useForm, useWatch } from 'react-hook-form'
import { z } from 'zod'

import { Alert } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/field'
import { useAuth } from '@/features/auth/auth-context'
import { forgotPassword, login, register } from '@/services/auth-service'
import type { UserView } from '@/shared/types/domain'

export type AuthMode = 'forgot' | 'login' | 'register'

/** 认证表单值，mode 也作为表单状态参与校验，避免切换模式时重建表单。 */
type AuthFormValues = z.infer<typeof authSchema>

const emailField = z.string().trim().min(1, '请输入邮箱').email('请输入有效的邮箱地址')
const authSchema = z.object({
  displayName: z.string().trim().max(50, '姓名不能超过 50 个字符'),
  email: emailField,
  mode: z.enum(['forgot', 'login', 'register']),
  organizationName: z.string().trim().max(80, '组织名称不能超过 80 个字符'),
  password: z.string(),
}).superRefine((values, ctx) => {
  if (values.mode === 'register') {
    if (!values.displayName) {
      ctx.addIssue({ code: 'custom', message: '请输入你的姓名', path: ['displayName'] })
    }
    if (!values.organizationName) {
      ctx.addIssue({ code: 'custom', message: '请输入组织名称', path: ['organizationName'] })
    }
  }
  if (values.mode !== 'forgot') {
    if (!values.password) {
      ctx.addIssue({ code: 'custom', message: '请输入密码', path: ['password'] })
    } else if (values.password.length < 8) {
      ctx.addIssue({ code: 'custom', message: '密码至少 8 位', path: ['password'] })
    }
  }
})

/** 归一化 URL 传入的初始模式，让官网「免费创建组织」能直接打开注册表单。 */
function initialAuthMode(mode?: AuthMode): AuthMode {
  return mode === 'register' || mode === 'forgot' ? mode : 'login'
}

const modeCopy: Record<AuthMode, { description: string; submit: string; title: string }> = {
  forgot: {
    description: '输入注册邮箱，我们会发送一封重置密码邮件。',
    submit: '发送重置邮件',
    title: '找回登录密码',
  },
  login: {
    description: '登录后继续管理项目、基线、交付证据与客户验收。',
    submit: '登录',
    title: '登录你的交付工作台',
  },
  register: {
    description: '创建组织后即可新建项目，并把交付过程留成可核验的记录。',
    submit: '创建组织',
    title: '创建你的交付工作台',
  },
}

/** 认证页面，使用 React Hook Form + Zod 提供字段级校验，不触发浏览器原生提示。 */
export function AuthPage({ initialMode }: { initialMode?: AuthMode }) {
  const navigate = useNavigate()
  const { setUser } = useAuth()
  const [serverError, setServerError] = useState('')
  const [notice, setNotice] = useState('')
  const form = useForm<AuthFormValues>({
    defaultValues: {
      displayName: '',
      email: '',
      mode: initialAuthMode(initialMode),
      organizationName: '',
      password: '',
    },
    mode: 'onTouched',
    resolver: zodResolver(authSchema),
  })
  const mode = useWatch({ control: form.control, name: 'mode' })
  const { errors, isSubmitting } = form.formState
  const copy = modeCopy[mode]

  function switchMode(nextMode: AuthMode) {
    form.setValue('mode', nextMode, { shouldDirty: false })
    form.clearErrors()
    setServerError('')
    setNotice('')
  }

  const onSubmit = form.handleSubmit(async (values) => {
    setServerError('')
    setNotice('')
    try {
      if (values.mode === 'forgot') {
        await forgotPassword(values.email)
        setNotice('如果该邮箱已注册，你会收到一封重置密码邮件。')
        return
      }
      const result = values.mode === 'login'
        ? await login(values.email, values.password)
        : await register({
          displayName: values.displayName,
          email: values.email,
          organizationName: values.organizationName,
          password: values.password,
        })
      setUser(result.user as UserView)
      await navigate({ to: '/' })
    } catch (error) {
      setServerError(error instanceof Error ? error.message : '操作失败，请重试')
    }
  })

  return (
    <div className="relative grid min-h-svh lg:grid-cols-2">
      <div className="absolute left-5 top-5 z-10 sm:left-8 sm:top-7">
        <Link to="/">
          <Button size="sm" variant="ghost">
            <ArrowLeft className="size-4" />
            返回首页
          </Button>
        </Link>
      </div>

      <section className="flex items-center justify-center px-6 pb-12 pt-20 lg:px-12 lg:py-16">
        <div className="w-full max-w-md space-y-8">
          <header className="space-y-3">
            <div className="flex items-center gap-2.5">
              <span className="flex size-9 items-center justify-center rounded-xl bg-primary text-primary-foreground shadow-[0_8px_24px_-12px_var(--primary)]">
                <ShieldCheck className="size-5" />
              </span>
              <div>
                <p className="text-xs font-medium uppercase tracking-[0.18em] text-primary">Independent Delivery Evidence</p>
                <h1 className="text-2xl font-semibold tracking-tight">Veristack</h1>
              </div>
            </div>
            <div className="space-y-1">
              <h2 className="text-2xl font-semibold tracking-tight">{copy.title}</h2>
              <p className="text-sm leading-6 text-muted-foreground">{copy.description}</p>
            </div>
          </header>

          <form className="space-y-4" noValidate onSubmit={onSubmit}>
            {mode === 'register' && (
              <>
                <Input
                  autoComplete="organization"
                  error={errors.organizationName?.message}
                  label="组织名称"
                  placeholder="例如：验迹工作室"
                  required
                  {...form.register('organizationName')}
                />
                <Input
                  autoComplete="name"
                  error={errors.displayName?.message}
                  label="你的姓名"
                  placeholder="例如：张开发"
                  required
                  {...form.register('displayName')}
                />
              </>
            )}

            <Input
              autoComplete="email"
              error={errors.email?.message}
              label="邮箱"
              placeholder="you@example.com"
              required
              type="email"
              {...form.register('email', { onChange: () => setServerError('') })}
            />

            {mode !== 'forgot' && (
              <Input
                autoComplete={mode === 'register' ? 'new-password' : 'current-password'}
                error={errors.password?.message}
                hint="至少 8 位字符"
                label="密码"
                placeholder="输入至少 8 位密码"
                required
                type="password"
                {...form.register('password', { onChange: () => setServerError('') })}
              />
            )}

            {serverError && (
              <Alert title="操作未完成" tone="danger">
                {serverError}
              </Alert>
            )}
            {notice && (
              <Alert title="请查收邮件" tone="success">
                {notice}
              </Alert>
            )}

            <Button className="w-full" disabled={isSubmitting} size="lg" type="submit">
              {isSubmitting ? '请稍候…' : copy.submit}
            </Button>
          </form>

          <div className="flex items-center justify-between gap-4 text-sm text-muted-foreground">
            {mode === 'login' && (
              <Button className="h-auto p-0 text-sm" onClick={() => switchMode('forgot')} variant="link">
                忘记密码？
              </Button>
            )}
            <p className="ml-auto text-center">
              {mode === 'register' ? '已有账号？' : mode === 'forgot' ? '想起密码了？' : '还没有账号？'}
              <Button
                className="ml-1 h-auto p-0 text-sm"
                onClick={() => switchMode(mode === 'register' ? 'login' : 'register')}
                variant="link"
              >
                {mode === 'register' ? '返回登录' : mode === 'forgot' ? '返回登录' : '注册组织'}
              </Button>
            </p>
          </div>
        </div>
      </section>

      <aside className="hidden border-l bg-surface-muted/45 lg:flex lg:flex-col lg:justify-center lg:px-16">
        <div className="max-w-md space-y-7">
          <p className="text-xs font-medium uppercase tracking-[0.2em] text-primary">让交付可核验，让验收可追溯</p>
          <h2 className="text-4xl font-semibold leading-[1.15] tracking-tight">
            每一份交付，
            <br />
            都值得被认真记录。
          </h2>
          <p className="text-sm leading-7 text-muted-foreground">
            Veristack 把需求、变更、证据与客户确认串成一条只追加的记录链。登录后，你看到的不是一堆文件，而是一份双方都能独立复核的交付说明。
          </p>
          <div className="space-y-4 border-t pt-5 text-sm">
            <AuthPoint index="01" text="冻结范围基线，每一次调整都由客户确认。" />
            <AuthPoint index="02" text="交付清单与 SHA-256 摘要绑定，不可被事后替换。" />
            <AuthPoint index="03" text="客户逐条验收，异议与回复形成闭环。" />
          </div>
        </div>
      </aside>
    </div>
  )
}

/** 登录页右侧的能力点，保持与官网一致的信息层级。 */
function AuthPoint({ index, text }: { index: string; text: string }) {
  return (
    <div className="flex gap-3">
      <span className="font-mono text-xs text-primary">{index}</span>
      <p className="leading-6 text-muted-foreground">{text}</p>
    </div>
  )
}
