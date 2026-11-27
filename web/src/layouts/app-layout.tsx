import { useQuery } from '@tanstack/react-query'
import { Link, Outlet, useNavigate, useRouterState } from '@tanstack/react-router'
import { Bell, LogOut, Moon, Sun, UserCog } from 'lucide-react'
import { useState } from 'react'

import { useTheme } from '@/app/theme-context'
import { Button } from '@/components/ui/button'
import { AccountDialog } from '@/features/auth/account-dialog'
import { useAuth } from '@/features/auth/auth-context'
import { LandingPage } from '@/features/landing/landing-page'
import { logout } from '@/services/auth-service'
import { fetchUnreadNotificationCount } from '@/services/notification-service'

const authPath = '/auth'
const portalPathPrefix = '/accept/'

/** 管理端主布局，提供顶部导航、主题切换、退出与内容出口。 */
export function AppLayout() {
  const pathname = useRouterState({ select: (state) => state.location.pathname })
  // 根匹配上的全局 404 不会替换根组件，而是通过 Outlet 渲染；这里必须优先放行，避免被未登录首页吞掉。
  const hasGlobalNotFound = useRouterState({
    select: (state) => state.matches.some((match) => Boolean((match as { _notFound?: boolean })._notFound)),
  })
  if (hasGlobalNotFound) {
    return <Outlet />
  }
  // 登录页与客户验收门户面向外部用户，直接渲染内容而不套用管理端外壳。
  if (pathname === authPath || pathname.startsWith(`${authPath}/`) || pathname.startsWith(portalPathPrefix)) {
    return <Outlet />
  }
  return <AppShell />
}

/** 应用外壳，承载登录后的导航与内容。 */
function AppShell() {
  const { theme, toggleTheme } = useTheme()
  const { isAuthenticated, setUser, user } = useAuth()
  const [isAccountOpen, setIsAccountOpen] = useState(false)
  const navigate = useNavigate()
  const { data: unread } = useQuery({
    enabled: isAuthenticated,
    queryFn: fetchUnreadNotificationCount,
    queryKey: ['notifications', 'unread-count'],
    refetchInterval: 60_000,
  })

  async function onLogout() {
    try {
      await logout()
    } finally {
      setUser(null)
      await navigate({ to: authPath })
    }
  }

  if (!isAuthenticated) {
    return <LandingPage />
  }

  return (
    <div className="flex min-h-svh flex-col bg-background">
      <header className="sticky top-0 z-10 border-b bg-surface/85 backdrop-blur">
        <div className="mx-auto flex h-14 max-w-7xl items-center justify-between px-5">
          <Link className="text-base font-semibold tracking-tight" to="/">
            Veristack
          </Link>
          <nav className="flex items-center gap-2">
            {isAuthenticated && (
              <Link className="hidden text-sm text-muted-foreground transition-colors hover:text-foreground sm:inline-flex" to="/">
                项目列表
              </Link>
            )}
            {isAuthenticated && (
              <Link className="hidden text-sm text-muted-foreground transition-colors hover:text-foreground sm:inline-flex" to="/projects/import">
                导入指南
              </Link>
            )}
            {user && <span className="hidden text-sm text-muted-foreground sm:inline">{user.displayName}</span>}
            {isAuthenticated && (
              <Button aria-label="通知" className="relative" size="icon" variant="ghost" onClick={() => void navigate({ to: '/notifications' })}>
                <Bell className="size-4" />
                {unread && unread.count > 0 && (
                  <span className="absolute -right-1 -top-1 flex min-w-4 items-center justify-center rounded-full bg-primary px-1 text-[10px] font-medium text-primary-foreground">
                    {unread.count > 99 ? '99+' : unread.count}
                  </span>
                )}
              </Button>
            )}
            {isAuthenticated && (
              <Button aria-label="账户设置" size="icon" variant="ghost" onClick={() => setIsAccountOpen(true)}>
                <UserCog className="size-4" />
              </Button>
            )}
            <Button aria-label="切换主题" size="icon" variant="ghost" onClick={toggleTheme}>
              {theme === 'dark' ? <Sun className="size-4" /> : <Moon className="size-4" />}
            </Button>
            {isAuthenticated && (
              <Button aria-label="退出登录" size="icon" variant="ghost" onClick={onLogout}>
                <LogOut className="size-4" />
              </Button>
            )}
          </nav>
        </div>
      </header>
      {isAccountOpen && <AccountDialog onClose={() => setIsAccountOpen(false)} />}
      <main className="mx-auto w-full max-w-7xl flex-1 px-5 py-8">
        <Outlet />
      </main>
    </div>
  )
}
