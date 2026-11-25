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

/** 退出登录，清除本地令牌。 */
export function logout(): void {
  clearAccessToken()
}
