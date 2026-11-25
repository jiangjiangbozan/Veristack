import { Link, Outlet, useNavigate, useRouterState } from '@tanstack/react-router'
import { LogOut, Moon, Sun } from 'lucide-react'

import { useTheme } from '@/app/theme-context'
import { Button } from '@/components/ui/button'
import { useAuth } from '@/features/auth/auth-context'
import { logout } from '@/services/auth-service'

const authPath = '/auth'
const portalPathPrefix = '/accept/'

/** 管理端主布局，提供顶部导航、主题切换、退出与内容出口。 */
export function AppLayout() {
  const pathname = useRouterState({ select: (state) => state.location.pathname })
  // 登录页与客户验收门户面向外部用户，直接渲染内容而不套用管理端外壳。
  if (pathname === authPath || pathname.startsWith(portalPathPrefix)) {
    return <Outlet />
  }
  return <AppShell />
}

/** 应用外壳，承载登录后的导航与内容。 */
function AppShell() {
  const { theme, toggleTheme } = useTheme()
  const { isAuthenticated, setUser, user } = useAuth()
  const navigate = useNavigate()

  async function onLogout() {
    logout()
    setUser(null)
    await navigate({ to: authPath })
  }

  return (
    <div className="flex min-h-svh flex-col bg-background">
      <header className="sticky top-0 z-10 border-b bg-surface/85 backdrop-blur">
        <div className="mx-auto flex h-14 max-w-7xl items-center justify-between px-5">
          <Link className="text-base font-semibold tracking-tight" to="/">
            Veristack
          </Link>
          <nav className="flex items-center gap-2">
            {user && <span className="hidden text-sm text-muted-foreground sm:inline">{user.displayName}</span>}
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
      <main className="mx-auto w-full max-w-7xl flex-1 px-5 py-8">
        {isAuthenticated ? <Outlet /> : <AuthGate />}
      </main>
    </div>
  )
}

/** 未认证时展示登录引导，避免在路由层重复鉴权判断。 */
function AuthGate() {
  return (
    <div className="flex flex-col items-center gap-4 py-20 text-center">
      <h1 className="text-2xl font-semibold tracking-tight">Veristack · 验迹</h1>
      <p className="max-w-md text-sm text-muted-foreground">让软件需求、变更、交付与验收形成一条可核验的记录链。登录后开始管理项目。</p>
      <Link to={authPath}>
        <Button>前往登录</Button>
      </Link>
    </div>
  )
}
