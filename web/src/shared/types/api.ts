/** API 统一响应外壳，与后端 response.Envelope 保持一致。 */
export interface ApiEnvelope<T> {
  code: string
  data?: T
  message: string
  page?: ApiPage
  success: boolean
}

/** 分页元信息，与后端 response.Page 保持一致。 */
export interface ApiPage {
  page: number
  pageSize: number
  total: number
}

/** 统一分页请求参数。 */
export interface PaginationParams {
  page?: number
  pageSize?: number
}

/** 用户角色枚举，与后端权限模型对齐。 */
export type UserRole = 'admin' | 'customer' | 'developer' | 'owner' | 'reviewer'
