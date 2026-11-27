import { clearAccessToken, request, setAccessToken } from '@/services/http-client'
import type { UserView } from '@/shared/types/domain'

interface AuthResult {
  token: string
  user: UserView
}

/** 使用邮箱和密码登录，成功后保存令牌。 */
export async function login(email: string, password: string): Promise<AuthResult> {
  const result = await request<AuthResult>('/auth/login', {
    body: JSON.stringify({ email, password }),
    method: 'POST',
  })
  setAccessToken(result.token)
  return result
}

/** 注册组织与 owner 用户，成功后保存令牌。 */
export async function register(payload: {
  displayName: string
  email: string
  organizationName: string
  password: string
}): Promise<AuthResult> {
  const result = await request<AuthResult>('/auth/register', {
    body: JSON.stringify(payload),
    method: 'POST',
  })
  setAccessToken(result.token)
  return result
}

/** 读取当前用户信息。 */
export function fetchCurrentUser(): Promise<{ user: UserView }> {
  return request<{ user: UserView }>('/auth/me')
}

/** 退出登录：先通知服务端撤销会话，再清除本地令牌。 */
export async function logout(): Promise<void> {
  try {
    await request('/auth/logout', { body: '{}', method: 'POST' })
  } finally {
    clearAccessToken()
  }
}

/** 修改当前用户密码。 */
export function changePassword(currentPassword: string, newPassword: string): Promise<{ changed: boolean }> {
  return request('/auth/password', {
    body: JSON.stringify({ currentPassword, newPassword }),
    method: 'POST',
  })
}

/** 申请密码重置邮件。 */
export function forgotPassword(email: string): Promise<{ sent: boolean }> {
  return request('/auth/password/forgot', { body: JSON.stringify({ email }), method: 'POST' })
}

/** 使用一次性令牌重置密码。 */
export function resetPassword(token: string, newPassword: string): Promise<{ reset: boolean }> {
  return request('/auth/password/reset', {
    body: JSON.stringify({ newPassword, token }),
    method: 'POST',
  })
}

/** 为当前用户邮箱申请验证码。 */
export function requestEmailVerificationCode(): Promise<{ sent: boolean }> {
  return request('/auth/email/verification-code', { body: '{}', method: 'POST' })
}

/** 提交邮箱验证码。 */
export function verifyEmail(code: string): Promise<{ user: UserView }> {
  return request('/auth/email/verify', { body: JSON.stringify({ code }), method: 'POST' })
}
