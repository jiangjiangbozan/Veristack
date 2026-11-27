import { Link } from '@tanstack/react-router'
import { KeyRound } from 'lucide-react'
import { useState, type FormEvent } from 'react'

import { Alert } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/field'
import { resetPassword } from '@/services/auth-service'

/** 密码重置页，使用邮件中的一次性令牌设置新密码。 */
export function ResetPasswordPage({ token }: { token: string }) {
  const [password, setPassword] = useState('')
  const [error, setError] = useState('')
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [isDone, setIsDone] = useState(false)

  async function onSubmit(event: FormEvent) {
    event.preventDefault()
    if (password.length < 8) {
      setError('新密码至少需要 8 位字符。')
      return
    }
    setError('')
    setIsSubmitting(true)
    try {
      await resetPassword(token, password)
      setIsDone(true)
    } catch (err) {
      setError(err instanceof Error ? err.message : '重置失败，链接可能已过期')
    } finally {
      setIsSubmitting(false)
    }
  }

  return (
    <div className="flex min-h-svh items-center justify-center px-6 py-12">
      <div className="w-full max-w-sm space-y-6">
        <div className="space-y-2 text-center">
          <KeyRound className="mx-auto size-8 text-primary" />
          <h1 className="text-2xl font-semibold tracking-tight">重置密码</h1>
          <p className="text-sm text-muted-foreground">重置链接 30 分钟内有效，提交后旧会话会全部失效。</p>
        </div>
        {isDone ? (
          <div className="space-y-4 text-center">
            <Alert title="密码已重置" tone="success">请使用新密码登录。</Alert>
            <Link className="text-sm font-medium text-primary hover:underline" to="/auth">返回登录</Link>
          </div>
        ) : (
          <form className="space-y-4" noValidate onSubmit={onSubmit}>
            <Input
              error={error && password.length < 8 ? error : undefined}
              hint="至少 8 位字符"
              label="新密码"
              required
              type="password"
              value={password}
              onChange={(event) => {
                setPassword(event.target.value)
                setError('')
              }}
              placeholder="至少 8 位"
            />
            {error && password.length >= 8 ? <Alert title="重置没有完成" tone="danger">{error}</Alert> : null}
            <Button className="w-full" disabled={isSubmitting} type="submit">{isSubmitting ? '提交中…' : '设置新密码'}</Button>
            <p className="text-center text-xs text-muted-foreground">
              <Link className="text-primary hover:underline" to="/auth">返回登录</Link>
            </p>
          </form>
        )}
      </div>
    </div>
  )
}
