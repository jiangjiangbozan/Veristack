import { useNavigate } from '@tanstack/react-router'
import { useState, type FormEvent } from 'react'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/field'
import { useAuth } from '@/features/auth/auth-context'
import { login, register } from '@/services/auth-service'
import type { UserView } from '@/shared/types/domain'

type AuthMode = 'login' | 'register'

/** 认证页面，支持登录与注册组织两种模式。 */
export function AuthPage() {
  const [mode, setMode] = useState<AuthMode>('login')
  const [error, setError] = useState('')
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [displayName, setDisplayName] = useState('')
  const [organizationName, setOrganizationName] = useState('')
  const { setUser } = useAuth()
  const navigate = useNavigate()

  async function onSubmit(event: FormEvent) {
    event.preventDefault()
    setError('')
    setIsSubmitting(true)
    try {
      const result = mode === 'login'
        ? await login(email, password)
        : await register({ displayName, email, organizationName, password })
      setUser(result.user as UserView)
      await navigate({ to: '/' })
    } catch (err) {
      setError(err instanceof Error ? err.message : '操作失败，请重试')
    } finally {
      setIsSubmitting(false)
    }
  }

  return (
    <div className="grid min-h-svh lg:grid-cols-2">
      <section className="flex items-center justify-center px-6 py-12">
        <div className="w-full max-w-sm space-y-8">
          <div className="space-y-2">
            <h1 className="text-3xl font-semibold tracking-tight">Veristack</h1>
            <p className="text-sm text-muted-foreground">让每一次软件交付，都有可独立核验的依据。</p>
          </div>

          <form className="space-y-4" onSubmit={onSubmit}>
            {mode === 'register' && (
              <>
                <Input label="组织名称" value={organizationName} onChange={(e) => setOrganizationName(e.target.value)} placeholder="例如：验迹工作室" required />
                <Input label="你的姓名" value={displayName} onChange={(e) => setDisplayName(e.target.value)} placeholder="例如：张开发" required />
              </>
            )}
            <Input label="邮箱" type="email" value={email} onChange={(e) => setEmail(e.target.value)} placeholder="you@example.com" required />
            <Input label="密码" type="password" value={password} onChange={(e) => setPassword(e.target.value)} placeholder="至少 8 位" required minLength={8} />
            {error && <p className="rounded-md bg-danger/10 px-3 py-2 text-sm text-danger">{error}</p>}
            <Button className="w-full" disabled={isSubmitting} type="submit">
              {isSubmitting ? '请稍候…' : mode === 'login' ? '登录' : '创建组织'}
            </Button>
          </form>

          <p className="text-center text-sm text-muted-foreground">
            {mode === 'login' ? '还没有账号？' : '已有账号？'}
            <button
              className="ml-1 font-medium text-primary hover:underline"
              type="button"
              onClick={() => { setError(''); setMode(mode === 'login' ? 'register' : 'login') }}
            >
              {mode === 'login' ? '注册组织' : '返回登录'}
            </button>
          </p>
        </div>
      </section>

      <section className="hidden bg-surface-muted lg:flex lg:flex-col lg:justify-center lg:px-16">
        <div className="max-w-md space-y-6">
          <p className="text-sm font-medium uppercase tracking-widest text-primary">Independent Delivery Evidence</p>
          <h2 className="text-4xl font-semibold leading-tight tracking-tight">交付有据，验收可溯。</h2>
          <ul className="space-y-4 text-sm text-muted-foreground">
            <li className="flex gap-3"><span className="text-primary">01</span><span>需求有边界：冻结的范围基线不会被悄悄改写。</span></li>
            <li className="flex gap-3"><span className="text-primary">02</span><span>变更有确认：每一次调整都留下客户决定。</span></li>
            <li className="flex gap-3"><span className="text-primary">03</span><span>交付有凭据：证据与验收条件逐条关联。</span></li>
          </ul>
        </div>
      </section>
    </div>
  )
}
