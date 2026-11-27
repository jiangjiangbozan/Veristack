import { useInfiniteQuery, useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { ArrowLeft, Bell, CheckCheck, CircleAlert, GitPullRequest, Mail } from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import {
  fetchNotifications,
  fetchUnreadNotificationCount,
  markAllNotificationsRead,
  markNotificationRead,
  NOTIFICATION_PAGE_SIZE,
  type NotificationView,
} from '@/services/notification-service'
import { formatDateTime } from '@/shared/lib/datetime'

/** 站内通知页，支持逐条点击跳转、一键已读与分页加载。 */
export function NotificationsPage() {
  const queryClient = useQueryClient()
  const { data, fetchNextPage, hasNextPage, isFetchingNextPage, isLoading } = useInfiniteQuery({
    queryKey: ['notifications', 'list'],
    queryFn: ({ pageParam }) => fetchNotifications(pageParam),
    initialPageParam: 1,
    getNextPageParam: (lastPage, allPages) =>
      lastPage.length === NOTIFICATION_PAGE_SIZE ? allPages.length + 1 : undefined,
  })
  const items = data?.pages.flat() ?? []
  const { data: unreadCount } = useQuery({
    queryKey: ['notifications', 'unread-count'],
    queryFn: fetchUnreadNotificationCount,
  })
  const readMutation = useMutation({
    mutationFn: markNotificationRead,
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: ['notifications'] }),
  })
  const readAllMutation = useMutation({
    mutationFn: markAllNotificationsRead,
    onSuccess: () => void queryClient.invalidateQueries({ queryKey: ['notifications'] }),
  })
  const unread = unreadCount?.count ?? 0

  async function onOpen(item: NotificationView) {
    if (!item.readAt) {
      try {
        await readMutation.mutateAsync(item.id)
      } catch {
        // 标记已读失败不阻断跳转，用户仍可打开关联记录。
      }
    }
    if (item.link) {
      window.location.assign(item.link)
    }
  }

  return (
    <div className="mx-auto w-full max-w-3xl space-y-6 px-6 py-10">
      <Link className="inline-flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground" to="/">
        <ArrowLeft className="size-4" />
        返回项目列表
      </Link>
      <header className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h1 className="flex items-center gap-2 text-2xl font-semibold tracking-tight">
            <Bell className="size-5 text-primary" />
            通知
          </h1>
          <p className="mt-1 text-sm text-muted-foreground">验收结论、异议回复与变更审批结果会出现在这里。</p>
        </div>
        <Button disabled={unread === 0 || readAllMutation.isPending} onClick={() => readAllMutation.mutate()} size="sm" variant="outline">
          <CheckCheck className="size-4" />
          全部已读
        </Button>
      </header>

      {isLoading && <p className="text-sm text-muted-foreground">加载中…</p>}
      {!isLoading && items.length === 0 && (
        <Card>
          <CardContent className="flex flex-col items-center gap-2 py-12 text-center">
            <Mail className="size-8 text-muted-foreground" />
            <p className="text-sm text-muted-foreground">暂时没有通知。</p>
          </CardContent>
        </Card>
      )}
      <ul className="space-y-3">
        {items.map((item) => (
          <li key={item.id}>
            <Card className={item.readAt ? '' : 'border-primary/40'}>
              <CardContent className="space-y-2">
                <div className="flex items-start justify-between gap-3">
                  <button className="flex items-start gap-3 text-left" type="button" onClick={() => void onOpen(item)}>
                    <TypeIcon type={item.type} />
                    <span>
                      <span className="flex items-center gap-2 text-sm font-medium">
                        {item.title}
                        {!item.readAt && <Badge tone="info">未读</Badge>}
                      </span>
                      <span className="mt-1 block text-sm text-muted-foreground">{item.body}</span>
                    </span>
                  </button>
                  <span className="shrink-0 text-xs text-muted-foreground">{formatDateTime(item.createdAt)}</span>
                </div>
              </CardContent>
            </Card>
          </li>
        ))}
      </ul>
      {hasNextPage && (
        <div className="flex justify-center">
          <Button disabled={isFetchingNextPage} onClick={() => void fetchNextPage()} size="sm" variant="outline">
            {isFetchingNextPage ? '加载中…' : '加载更多'}
          </Button>
        </div>
      )}
    </div>
  )
}

/** 按通知类型展示图标，帮助快速识别事件来源。 */
function TypeIcon({ type }: { type: string }) {
  if (type.startsWith('objection')) {
    return <CircleAlert className="mt-0.5 size-4 shrink-0 text-warning" />
  }
  if (type.startsWith('change')) {
    return <GitPullRequest className="mt-0.5 size-4 shrink-0 text-info" />
  }
  return <Bell className="mt-0.5 size-4 shrink-0 text-primary" />
}
