import type { ApiEnvelope } from '@/shared/types/api'

const apiBaseUrl = '/api/v1'
const tokenStorageKey = 'veristack.accessToken'

/** HTTP 请求失败时抛出的错误，携带后端稳定错误码与 HTTP 状态。 */
export class ApiError extends Error {
  readonly code: string
  readonly status: number

  constructor(code: string, message: string, status: number) {
    super(message)
    this.code = code
    this.status = status
  }
}

/** 网络层不可达，通常意味着本地后端未启动或网络中断。 */
const networkErrorMessage = '暂时无法连接交付服务，请稍后重试；如果刚启动本地环境，请确认服务已经运行。'

/** 按 HTTP 状态给出稳定、面向用户的兜底文案，避免把技术错误暴露到界面。 */
function fallbackMessage(status: number): string {
  if (status === 401) {
    return '登录状态已失效，请重新登录。'
  }
  if (status === 403) {
    return '你没有权限执行这个操作。'
  }
  if (status === 404) {
    return '请求的内容不存在，可能已被移除。'
  }
  if (status === 409) {
    return '当前数据状态已变化，请刷新后重试。'
  }
  if (status === 422) {
    return '提交的信息不完整或格式不正确，请检查后重试。'
  }
  if (status === 429) {
    return '操作太频繁了，请稍后再试。'
  }
  if (status >= 500) {
    return '服务暂时不可用，请稍后重试；如果持续出现，请联系管理员。'
  }
  return '请求失败，请稍后重试。'
}

/** 判断未知值是否具备统一响应外壳的必要字段。 */
function isApiEnvelope(value: unknown): value is ApiEnvelope<unknown> {
  if (typeof value !== 'object' || value === null) {
    return false
  }
  const candidate = value as Record<string, unknown>
  return typeof candidate.success === 'boolean' && typeof candidate.message === 'string'
}

/** 只在响应声明为 JSON 时解析，防止 HTML/空响应把原始解析异常抛给用户。 */
async function readEnvelope<T>(response: Response): Promise<ApiEnvelope<T> | null> {
  const contentType = response.headers.get('content-type') ?? ''
  if (!contentType.toLowerCase().includes('application/json')) {
    return null
  }
  try {
    const value: unknown = await response.json()
    return isApiEnvelope(value) ? (value as ApiEnvelope<T>) : null
  } catch {
    return null
  }
}

/** 组装可展示给用户的 ApiError，优先使用后端业务文案。 */
function apiErrorFromResponse<T>(response: Response, envelope: ApiEnvelope<T> | null): ApiError {
  const message = envelope?.message?.trim() || fallbackMessage(response.status)
  const code = envelope?.code || `http_${response.status}`
  return new ApiError(code, message, response.status)
}

/**
 * 统一 API 请求客户端。
 *
 * - 自动附加 Bearer 令牌与 JSON 头。
 * - 网络异常、空响应、HTML 错误页都转换为可读的 ApiError。
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
  let response: Response
  try {
    response = await fetch(`${apiBaseUrl}${path}`, { ...options, headers })
  } catch {
    throw new ApiError('network_error', networkErrorMessage, 0)
  }
  const envelope = await readEnvelope<T>(response)
  if (!response.ok) {
    throw apiErrorFromResponse(response, envelope)
  }
  if (!envelope) {
    throw new ApiError('invalid_response', '服务返回了无法识别的响应，请稍后重试。', response.status)
  }
  if (!envelope.success) {
    throw apiErrorFromResponse(response, envelope)
  }
  return envelope.data as T
}

/** 读取本地保存的访问令牌。 */
export function getAccessToken(): string | null {
  return localStorage.getItem(tokenStorageKey)
}

/**
 * 下载二进制响应（如证据包 ZIP）。
 *
 * - 与 JSON 请求共用访问令牌，避免把令牌拼进 URL。
 * - 网络失败与非 JSON 错误页都转换成统一错误，不向界面抛原始解析异常。
 */
export async function requestBlob(path: string): Promise<Blob> {
  const token = getAccessToken()
  const headers = new Headers()
  if (token) {
    headers.set('Authorization', `Bearer ${token}`)
  }
  let response: Response
  try {
    response = await fetch(`${apiBaseUrl}${path}`, { headers })
  } catch {
    throw new ApiError('network_error', networkErrorMessage, 0)
  }
  if (!response.ok) {
    const envelope = await readEnvelope<unknown>(response)
    throw apiErrorFromResponse(response, envelope)
  }
  return response.blob()
}

/** 保存访问令牌，供后续请求自动附加。 */
export function setAccessToken(token: string): void {
  localStorage.setItem(tokenStorageKey, token)
}

/** 清除访问令牌，用于退出登录与令牌失效处理。 */
export function clearAccessToken(): void {
  localStorage.removeItem(tokenStorageKey)
}
