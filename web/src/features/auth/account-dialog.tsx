import { CheckCircle2, MailCheck } from 'lucide-react'
import { useState, type FormEvent } from 'react'
import { useNavigate } from '@tanstack/react-router'

import { Alert } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/field'
import { Modal } from '@/components/ui/modal'
import { useAuth } from '@/features/auth/auth-context'
import { changePassword, logout, requestEmailVerificationCode, verifyEmail } from '@/services/auth-service'
import { formatDateTime } from '@/shared/lib/datetime'

/** 账户设置弹窗，提供邮箱验证与密码修改。 */
export function AccountDialog({ onClose }: { onClose: () => void }) {
  const { setUser, user } = useAuth()
  const navigate = useNavigate()
  const [code, setCode] = useState('')
  const [isSending, setIsSending] = useState(false)
  const [isVerifying, setIsVerifying] = useState(false)
  const [emailMessage, setEmailMessage] = useState('')
  const [emailError, setEmailError] = useState('')
  const [currentPassword, setCurrentPassword] = useState('')
  const [newPassword, setNewPassword] = useState('')
  const [passwordError, setPasswordError] = useState('')
  const [isChanging, setIsChanging] = useState(false)

  if (!user) {
    return null
  }

  async function onSendCode() {
    setIsSending(true)
    setEmailError('')
    setEmailMessage('')
    try {
      await requestEmailVerificationCode()
      setEmailMessage('验证码已发送，请查收邮箱。')
    } catch (err) {
      setEmailError(err instanceof Error ? err.message : '发送失败')
    } finally {
      setIsSending(false)
    }
  }

  async function onVerify(event: FormEvent) {
    event.preventDefault()
    if (code.trim().length !== 6) {
      setEmailError('请输入 6 位邮箱验证码。')
      return
    }
    setIsVerifying(true)
    setEmailError('')
    try {
      const result = await verifyEmail(code)
      setUser(result.user)
      setEmailMessage('邮箱已验证。')
      setCode('')
    } catch (err) {
      setEmailError(err instanceof Error ? err.message : '验证失败')
    } finally {
      setIsVerifying(false)
    }
  }

  async function onChangePassword(event: FormEvent) {
    event.preventDefault()
    if (!currentPassword) {
      setPasswordError('请输入当前密码。')
      return
    }
    if (newPassword.length < 8) {
      setPasswordError('新密码至少需要 8 位字符。')
      return
    }
    setIsChanging(true)
    setPasswordError('')
    try {
      await changePassword(currentPassword, newPassword)
      // 改密后服务端已使旧令牌失效，主动清理本地状态并回到登录页。
      try {
        await logout()
      } catch {
        // 令牌已失效，退出接口返回 401 属预期行为。
      }
      setUser(null)
      onClose()
      await navigate({ to: '/auth' })
    } catch (err) {
      setPasswordError(err instanceof Error ? err.message : '修改失败')
    } finally {
      setIsChanging(false)
    }
  }

  return (
    <Modal onClose={onClose} title="账户设置">
      <div className="space-y-6">
        <section className="space-y-3">
          <div className="flex items-center justify-between gap-3">
            <div>
              <h3 className="flex items-center gap-2 text-sm font-medium">
                <MailCheck className="size-4 text-primary" />
                邮箱验证
              </h3>
              <p className="mt-1 text-xs text-muted-foreground">{user.email}</p>
            </div>
            {user.emailVerifiedAt ? (
              <Badge tone="success">已验证</Badge>
            ) : (
              <Badge tone="warning">未验证</Badge>
            )}
          </div>
          {user.emailVerifiedAt ? (
            <p className="flex items-center gap-2 rounded-lg bg-success/10 px-3 py-2 text-sm text-success">
              <CheckCircle2 className="size-4" />
              验证于 {formatDateTime(user.emailVerifiedAt)}
            </p>
          ) : (
            <form className="space-y-3" noValidate onSubmit={onVerify}>
              <div className="flex gap-2">
                <Button disabled={isSending} onClick={onSendCode} size="sm" type="button" variant="outline">
                  {isSending ? '发送中…' : '发送验证码'}
                </Button>
                <Input
                  aria-label="邮箱验证码"
                  maxLength={6}
                  placeholder="6 位验证码"
                  value={code}
                  onChange={(event) => {
                    setCode(event.target.value)
                    setEmailError('')
                  }}
                />
                <Button disabled={isVerifying} size="sm" type="submit">{isVerifying ? '验证中…' : '验证'}</Button>
              </div>
              {emailMessage && <p className="text-xs text-success">{emailMessage}</p>}
              {emailError && <p className="text-xs text-danger">{emailError}</p>}
            </form>
          )}
        </section>

        <section className="space-y-3 border-t pt-5">
          <h3 className="text-sm font-medium">修改密码</h3>
          <form className="space-y-3" noValidate onSubmit={onChangePassword}>
            <Input
              error={passwordError && !currentPassword ? passwordError : undefined}
              label="当前密码"
              required
              type="password"
              value={currentPassword}
              onChange={(event) => {
                setCurrentPassword(event.target.value)
                setPasswordError('')
              }}
            />
            <Input
              error={passwordError && currentPassword && newPassword.length < 8 ? passwordError : undefined}
              hint="至少 8 位字符"
              label="新密码"
              required
              type="password"
              value={newPassword}
              onChange={(event) => {
                setNewPassword(event.target.value)
                setPasswordError('')
              }}
            />
            {passwordError && currentPassword && newPassword.length >= 8 ? (
              <Alert title="密码没有更新" tone="danger">{passwordError}</Alert>
            ) : null}
            <Button className="w-full" disabled={isChanging} size="sm" type="submit">{isChanging ? '提交中…' : '更新密码'}</Button>
          </form>
        </section>
      </div>
    </Modal>
  )
}
