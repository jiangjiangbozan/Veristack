import { Link, type ErrorComponentProps } from '@tanstack/react-router'
import { ArrowLeft, ArrowRight, FileQuestion, Moon, RotateCcw, ServerCrash, ShieldCheck, Sun } from 'lucide-react'
import type { ReactNode } from 'react'

import { useTheme } from '@/app/theme-context'
import { Alert } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { ApiError } from '@/services/http-client'

interface StatusPageProps {
  actions: ReactNode // 页面底部的恢复动作
  code: string // 状态码，如 404、500、503
  description: string // 面向用户的原因说明
  eyebrow: string // 页面类型短标签
  title: string // 主标题
}

/** 统一的全屏状态页，沿用产品官网的网格与证据卡片语言。 */
function StatusPage({ actions, code, description, eyebrow, title }: StatusPageProps) {
  const { theme, toggleTheme } = useTheme()
  return (
    <div className="relative flex min-h-svh flex-col overflow-hidden bg-background">
      <div aria-hidden="true" className="landing-grid pointer-events-none absolute inset-0 opacity-40" />
      <div aria-hidden="true" className="pointer-events-none absolute -right-32 -top-40 size-[30rem] rounded-full bg-primary/10 blur-3xl" />
      <div aria-hidden="true" className="pointer-events-none absolute -left-40 top-1/2 size-[26rem] rounded-full bg-info/10 blur-3xl" />

      <header className="relative z-10 mx-auto flex h-16 w-full max-w-7xl items-center justify-between px-5 sm:px-8">
        <Link className="flex items-center gap-2 text-sm font-semibold tracking-tight" to="/">
          <span className="flex size-8 items-center justify-center rounded-lg bg-primary text-primary-foreground">
            <ShieldCheck className="size-4" />
          </span>
          Veristack
        </Link>
        <Button aria-label="切换主题" onClick={toggleTheme} size="icon" variant="ghost">
          {theme === 'dark' ? <Sun className="size-4" /> : <Moon className="size-4" />}
        </Button>
      </header>

      <main className="relative z-10 mx-auto flex w-full max-w-7xl flex-1 items-center px-5 py-16 sm:px-8">
        <div className="grid w-full gap-12 lg:grid-cols-[minmax(0,1fr)_22rem] lg:items-center">
          <div className="space-y-7">
            <p className="inline-flex items-center gap-2 rounded-full border bg-surface/80 px-3 py-1 text-xs font-medium text-muted-foreground backdrop-blur">
              <span className="size-1.5 rounded-full bg-primary" />
              {eyebrow}
            </p>
            <div className="space-y-4">
              <p className="font-mono text-6xl font-semibold tracking-tighter text-primary sm:text-8xl">{code}</p>
              <h1 className="max-w-2xl text-3xl font-semibold tracking-tight sm:text-4xl">{title}</h1>
              <p className="max-w-xl text-sm leading-7 text-muted-foreground sm:text-base">{description}</p>
            </div>
            <div className="flex flex-wrap gap-3">{actions}</div>
          </div>

          <div className="hidden lg:block">
            <div className="rounded-[calc(var(--radius-card)+0.25rem)] border bg-surface/85 p-5 shadow-[0_24px_60px_-32px_var(--primary)] backdrop-blur">
              <div className="flex items-center justify-between border-b pb-3 text-xs text-muted-foreground">
                <span className="flex items-center gap-2">
                  {code === '404' ? <FileQuestion className="size-4 text-primary" /> : <ServerCrash className="size-4 text-primary" />}
                  交付核验状态
                </span>
                <span className="rounded-full bg-surface-muted px-2 py-0.5">未完成</span>
              </div>
              <div className="space-y-4 pt-4">
                {['定位请求的记录', '读取不可变快照', '返回可核验结果'].map((item, index) => (
                  <div className="flex items-center gap-3 text-sm" key={item}>
                    <span className="flex size-6 items-center justify-center rounded-full bg-surface-muted font-mono text-[10px] text-muted-foreground">
                      {String(index + 1).padStart(2, '0')}
                    </span>
                    <span className="flex-1 text-muted-foreground">{item}</span>
                    <span className={index < 1 ? 'text-success' : 'text-danger'}>{index < 1 ? '完成' : '中断'}</span>
                  </div>
                ))}
              </div>
              <Alert className="mt-5" tone="info">
                你的项目数据没有被修改，可以安全地返回并重试。
              </Alert>
            </div>
          </div>
        </div>
      </main>
    </div>
  )
}

/** 未匹配路由的 404 页面。 */
export function NotFoundPage() {
  return (
    <StatusPage
      actions={(
        <>
          <Link to="/">
            <Button>
              返回首页
              <ArrowRight className="size-4" />
            </Button>
          </Link>
          <Button onClick={() => window.history.back()} variant="outline">
            <ArrowLeft className="size-4" />
            返回上一页
          </Button>
        </>
      )}
      code="404"
      description="这个地址可能已经变更，或者对应的项目、交付记录已被移除。请检查链接，或从项目列表重新进入。"
      eyebrow="页面未找到"
      title="这个页面没有留下证据。"
    />
  )
}

/** 根路由异常边界，用于承接未预期的渲染或数据错误。 */
export function ServerErrorPage({ error, reset }: ErrorComponentProps) {
  const isServiceUnavailable = error instanceof ApiError && (error.status === 0 || error.status >= 500)
  return (
    <StatusPage
      actions={(
        <>
          <Button onClick={reset}>
            <RotateCcw className="size-4" />
            重新尝试
          </Button>
          <Link to="/">
            <Button variant="outline">返回首页</Button>
          </Link>
        </>
      )}
      code={isServiceUnavailable ? '503' : '500'}
      description={isServiceUnavailable
        ? '交付服务可能尚未启动，或网络暂时中断。请确认服务已启动后重试。'
        : '系统在处理这次请求时遇到了异常。你的数据没有被修改，可以重试或返回首页。'}
      eyebrow={isServiceUnavailable ? '服务暂时不可用' : '系统异常'}
      title={isServiceUnavailable ? '暂时连接不上交付服务。' : '系统没有完成这次核验。'}
    />
  )
}
