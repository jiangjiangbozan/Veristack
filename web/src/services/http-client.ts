import type { ApiEnvelope } from '@/shared/types/api'

const apiBaseUrl = '/api/v1'
const tokenStorageKey = 'veristack.accessToken'

/** HTTP 请求失败时抛出的错误，携带后端稳定错误码。 */
export class ApiError extends Error {
  readonly code: string
  readonly status: number

  constructor(code: string, message: string, status: number) {
    super(message)
    this.code = code
    this.status = status
  }
}

/**
 * 统一 API 请求客户端。
 *
 * - 自动附加 Bearer 令牌与 JSON 头。
 * - 解包响应外壳，失败时抛出 ApiError。
 */
export async function request<T>(path: string, options: RequestInit = {}): Promise<T> {
  const token = getAccessToken()
  const headers = new Headers(options.headers)
  // FormData 需要由浏览器自行设置 multipart 边界，手动指定会破坏解析。
  if (!(options.body instanceof FormData)) {
    headers.set('Content-Type', 'application/json')
  }
  if (token) {
    headers.set('Authorization', `Bearer ${token}`)
  }
  const response = await fetch(`${apiBaseUrl}${path}`, { ...options, headers })
  const envelope = (await response.json()) as ApiEnvelope<T>
  if (!response.ok || !envelope.success) {
    throw new ApiError(envelope.code, envelope.message, response.status)
  }
  return envelope.data as T
}

/** 读取本地保存的访问令牌。 */
export function getAccessToken(): string | null {
  return localStorage.getItem(tokenStorageKey)
}

/** 保存访问令牌，供后续请求自动附加。 */
export function setAccessToken(token: string): void {
  localStorage.setItem(tokenStorageKey, token)
}

/** 清除访问令牌，用于退出登录与令牌失效处理。 */
export function clearAccessToken(): void {
  localStorage.removeItem(tokenStorageKey)
}
