import { request } from '@/services/http-client'

/** 站内通知。 */
export interface NotificationView {
  body: string // 通知正文
  createdAt: string // 创建时间
  id: string // 通知 ID
  link: string // 站内跳转链接
  readAt: string // 阅读时间，未读为空
  title: string // 通知标题
  type: string // 事件类型
}

/** 通知分页大小，与后端默认值保持一致。 */
export const NOTIFICATION_PAGE_SIZE = 20

/** 查询当前用户的站内通知。 */
export function fetchNotifications(page = 1, pageSize = NOTIFICATION_PAGE_SIZE): Promise<NotificationView[]> {
  const params = new URLSearchParams({ page: String(page), pageSize: String(pageSize) })
  return request(`/notifications?${params.toString()}`)
}

/** 查询当前用户的未读通知数量。 */
export function fetchUnreadNotificationCount(): Promise<{ count: number }> {
  return request('/notifications/unread-count')
}

/** 把一条通知标记为已读。 */
export function markNotificationRead(id: string): Promise<{ read: boolean }> {
  return request(`/notifications/${id}/read`, { body: '{}', method: 'POST' })
}

/** 把当前用户的全部未读通知标记为已读。 */
export function markAllNotificationsRead(): Promise<{ updated: number }> {
  return request('/notifications/read-all', { body: '{}', method: 'POST' })
}
