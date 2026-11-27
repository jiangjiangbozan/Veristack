import { Link } from '@tanstack/react-router'
import {
  ArrowRight,
  Check,
  ClipboardCheck,
  Database,
  FileCheck2,
  Fingerprint,
  GitBranch,
  Layers3,
  LockKeyhole,
  MessageCircle,
  Moon,
  PackageCheck,
  ScanLine,
  ShieldCheck,
  Sun,
  Upload,
  type LucideIcon,
} from 'lucide-react'
import { useEffect, useRef, useState, type PointerEvent as ReactPointerEvent, type ReactNode } from 'react'

import { useTheme } from '@/app/theme-context'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { cn } from '@/shared/lib/cn'

const capabilities = [
  {
    description: '基线冻结后不可编辑，任何范围调整都必须经过变更申请与客户确认，旧版本原样保留。',
    icon: Layers3,
    title: '需求基线',
  },
  {
    description: '提交交付时自动生成清单快照和 SHA-256 摘要，后续补充证据只能追加，不能改写历史。',
    icon: FileCheck2,
    title: '交付清单',
  },
  {
    description: 'Git 提交、CI 报告、截图、测试报告与人工说明统一归档，按时间顺序串成完整证据链。',
    icon: GitBranch,
    title: '证据时间线',
  },
  {
    description: '客户通过签名链接逐条确认验收条件，异议、回复与复核形成闭环，非技术用户也能看懂。',
    icon: ClipboardCheck,
    title: '客户验收',
  },
] as const

const workflow = [
  { description: '新建项目，或用 JSON 一次导入已有项目、需求与里程碑。', title: '录入项目' },
  { description: '确认范围并冻结 v1 基线，后续调整走变更申请。', title: '锁定范围' },
  { description: '提交交付版本，归档代码、CI、截图与测试证据。', title: '留下凭据' },
  { description: '客户逐条验收，确认或提出异议，结论自动留痕。', title: '完成验收' },
] as const

const useCases = ['外包交付', '定制开发', '私有化部署', '多团队协作', '客户验收', '审计留痕', '变更闭环', '证据归档'] as const

interface SnapshotEvent {
  icon: LucideIcon
  meta: string
  title: string
}

interface SnapshotMetric {
  label: string
  mono?: boolean
  value: string
}

interface SnapshotSlide {
  badge: string
  badgeTone: 'success' | 'warning'
  detail: string
  events: SnapshotEvent[]
  footerLabel: string
  footerMeta: string
  id: string
  metrics: SnapshotMetric[]
  title: string
}

const snapshotSlides: SnapshotSlide[] = [
  {
    badge: '已冻结',
    badgeTone: 'success',
    detail: '示例项目 · 客户官网改版 · 范围基线',
    events: [
      { icon: Layers3, meta: '客户确认 · 09:18', title: '需求 REQ-008「导出字段」纳入 v2' },
      { icon: MessageCircle, meta: '变更申请 · 10:06', title: '新增“批量下载”能力，影响已评估' },
      { icon: LockKeyhole, meta: '冻结快照 · 14:32', title: '基线 v2 进入只读，旧版本保留' },
    ],
    footerLabel: '基线版本已冻结，旧版本保持只读',
    footerMeta: 'v2.0',
    id: 'baseline',
    metrics: [
      { label: '需求基线', value: 'v2 · 12 条' },
      { label: '变更申请', value: '3 次 · 全部确认' },
      { label: '验收条件', value: '12 / 12' },
      { label: '基线摘要', mono: true, value: '8f2a…c91e' },
    ],
    title: '双方确认的范围，已经锁定',
  },
  {
    badge: '已生成',
    badgeTone: 'success',
    detail: '示例项目 · 客户官网改版 · 交付 v2.0',
    events: [
      { icon: GitBranch, meta: 'GitHub · feat/checkout', title: '提交 8f2a1c：修复结算页空状态' },
      { icon: PackageCheck, meta: '附件 · 测试报告', title: 'JUnit 126 项单元与契约测试通过' },
      { icon: ShieldCheck, meta: '完整性校验 · 15:02', title: '清单摘要与签名均通过' },
    ],
    footerLabel: '清单与 SHA-256 摘要已绑定',
    footerMeta: 'Ed25519',
    id: 'delivery',
    metrics: [
      { label: '交付版本', value: 'v2.0' },
      { label: '证据附件', value: '8 份' },
      { label: 'CI 报告', value: '126 项通过' },
      { label: '清单摘要', mono: true, value: 'a08b…4f1c' },
    ],
    title: '这一次交付，证据是齐全的',
  },
  {
    badge: '验收中',
    badgeTone: 'warning',
    detail: '示例项目 · 客户官网改版 · 验收门户',
    events: [
      { icon: ClipboardCheck, meta: '客户验收 · 14:32', title: '「导出字段」验收条件已确认' },
      { icon: MessageCircle, meta: '客户异议 · 15:10', title: '“移动端入口不明显”，已回复' },
      { icon: Check, meta: '开发者复核 · 15:26', title: '补充截图后，客户复核通过' },
    ],
    footerLabel: '客户正在验收门户逐条确认',
    footerMeta: '2 待确认',
    id: 'acceptance',
    metrics: [
      { label: '验收条件', value: '12 / 12' },
      { label: '客户已确认', value: '10 条' },
      { label: '待确认', value: '2 条' },
      { label: '异议闭环', value: '1 条' },
    ],
    title: '每一条结论，都能回到原始条件',
  },
]

/** 未登录首页，按「交付证据层」的产品定位组织信息而非只放一个登录按钮。 */
export function LandingPage() {
  const { theme, toggleTheme } = useTheme()
  return (
    <div className="relative min-h-svh overflow-x-clip bg-background">
      <div aria-hidden="true" className="landing-grid pointer-events-none absolute inset-x-0 top-0 h-[46rem] opacity-35" />
      <div aria-hidden="true" className="landing-aurora pointer-events-none absolute inset-x-0 top-0 h-[42rem] opacity-80" />
      <div aria-hidden="true" className="landing-float pointer-events-none absolute -right-40 -top-48 size-[34rem] rounded-full bg-primary/10 blur-3xl" />
      <div aria-hidden="true" className="landing-float-delayed pointer-events-none absolute -left-56 top-[24rem] size-[30rem] rounded-full bg-info/10 blur-3xl" />

      <header className="sticky top-0 z-20 border-b border-border/70 bg-background/80 backdrop-blur-xl">
        <div className="mx-auto flex h-16 w-full max-w-7xl items-center justify-between gap-4 px-5 sm:px-8">
          <Link className="flex items-center gap-2.5" to="/">
            <span className="flex size-9 items-center justify-center rounded-xl bg-primary text-primary-foreground shadow-[0_8px_24px_-12px_var(--primary)]">
              <ShieldCheck className="size-5" />
            </span>
            <span className="text-base font-semibold tracking-tight">Veristack</span>
            <span className="hidden text-xs text-muted-foreground sm:inline">验迹</span>
          </Link>

          <nav className="hidden items-center gap-7 text-sm text-muted-foreground md:flex">
            <a className="transition-colors hover:text-foreground" href="#capabilities">产品能力</a>
            <a className="transition-colors hover:text-foreground" href="#workflow">工作流程</a>
            <a className="transition-colors hover:text-foreground" href="#why">为什么是验迹</a>
          </nav>

          <div className="flex items-center gap-1.5">
            <Button aria-label="切换主题" onClick={toggleTheme} size="icon" variant="ghost">
              {theme === 'dark' ? <Sun className="size-4" /> : <Moon className="size-4" />}
            </Button>
            <Link className="hidden sm:inline-flex" to="/auth">
              <Button size="sm" variant="ghost">登录</Button>
            </Link>
            <Link to="/auth" search={{ mode: 'register' }}>
              <Button size="sm">免费创建组织</Button>
            </Link>
          </div>
        </div>
      </header>

      <main>
        <section className="relative z-10 mx-auto grid w-full max-w-7xl gap-14 px-5 pb-20 pt-16 sm:px-8 lg:grid-cols-[minmax(0,1.02fr)_minmax(0,0.98fr)] lg:items-center lg:gap-20 lg:pb-28 lg:pt-24">
          <Reveal className="space-y-8">
            <p className="inline-flex items-center gap-2 rounded-full border bg-surface/75 px-3 py-1.5 text-xs font-medium tracking-wide text-muted-foreground backdrop-blur">
              <span className="status-pulse size-1.5 rounded-full bg-success" />
              独立于 Git、CI 与 Agent 平台的交付证据层
            </p>
            <div className="space-y-6">
              <h1 className="max-w-2xl text-balance text-4xl font-semibold leading-[1.08] tracking-[-0.035em] text-foreground sm:text-5xl lg:text-6xl">
                让每一次交付，
                <span className="text-primary">都有可独立核验的依据。</span>
              </h1>
              <p className="max-w-xl text-base leading-8 text-muted-foreground sm:text-lg">
                Veristack 把需求基线、变更确认、交付清单、证据与客户验收连接成一条只追加的记录链，
                让外包、定制开发与私有化交付不再靠截图和记忆争论。
              </p>
            </div>
            <div className="flex flex-wrap items-center gap-3">
              <Link to="/auth" search={{ mode: 'register' }}>
                <Button size="lg">
                  开始记录第一个项目
                  <ArrowRight className="size-4" />
                </Button>
              </Link>
              <Link to="/auth">
                <Button size="lg" variant="outline">已有账号，登录</Button>
              </Link>
            </div>
            <p className="flex flex-wrap items-center gap-x-5 gap-y-2 text-xs text-muted-foreground">
              <span className="inline-flex items-center gap-1.5"><Check className="size-3.5 text-success" />支持私有化部署</span>
              <span className="inline-flex items-center gap-1.5"><Check className="size-3.5 text-success" />数据保留在你的环境</span>
              <span className="inline-flex items-center gap-1.5"><Check className="size-3.5 text-success" />无需信用卡</span>
            </p>
          </Reveal>

          <Reveal delay={100}>
            <DeliverySnapshot />
          </Reveal>
        </section>

        <section className="border-y bg-surface/40">
          <div className="mx-auto grid w-full max-w-7xl gap-4 px-5 py-8 sm:grid-cols-3 sm:px-8">
            <Metric label="需求与变更" value="每一次调整都由客户确认" />
            <Metric label="交付清单" value="清单与 SHA-256 摘要冻结" />
            <Metric label="客户验收" value="逐条确认，异议自动闭环" />
          </div>
        </section>

        <UseCaseTicker />

        <section className="mx-auto w-full max-w-7xl scroll-mt-20 px-5 py-20 sm:px-8 lg:py-28" id="capabilities">
          <Reveal>
            <SectionHeading
              eyebrow="核心能力"
              lead="交付不是一份压缩包，而是一条从约定到验收的完整记录。"
              title="把每个争议点，变成可回看的证据。"
            />
          </Reveal>
          <div className="mt-10 grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
            {capabilities.map((item, index) => (
              <Reveal className="h-full" delay={index * 70} key={item.title}>
                <article className="feature-card group flex h-full flex-col rounded-[calc(var(--radius-card)+0.25rem)] border bg-surface/75 p-6">
                  <div className="flex items-center justify-between">
                    <span className="flex size-10 items-center justify-center rounded-xl bg-primary/10 text-primary">
                      <item.icon className="size-5 transition-transform duration-300 group-hover:-rotate-3 group-hover:scale-110" />
                    </span>
                    <span className="font-mono text-[10px] text-muted-foreground">0{index + 1}</span>
                  </div>
                  <h3 className="mt-6 text-base font-semibold">{item.title}</h3>
                  <p className="mt-2 text-sm leading-6 text-muted-foreground">{item.description}</p>
                </article>
              </Reveal>
            ))}
          </div>
        </section>

        <section className="border-y bg-surface/50 scroll-mt-20" id="workflow">
          <div className="mx-auto w-full max-w-7xl px-5 py-20 sm:px-8 lg:py-28">
            <Reveal>
              <SectionHeading
                eyebrow="工作流程"
                lead="从项目进入到客户确认，每一步都留下可以独立复核的结果。"
                title="四步，把交付过程留成记录。"
              />
            </Reveal>
            <div className="mt-12 grid gap-5 md:grid-cols-4">
              {workflow.map((item, index) => (
                <Reveal delay={index * 70} key={item.title}>
                  <div className="group relative h-full rounded-[calc(var(--radius-card)+0.25rem)] border bg-background/75 p-6 transition-all duration-300 hover:-translate-y-1 hover:border-primary/40 hover:bg-background">
                    <div className="flex items-center gap-3">
                      <span className="flex size-9 items-center justify-center rounded-full bg-primary text-sm font-semibold text-primary-foreground transition-transform duration-300 group-hover:scale-110">{index + 1}</span>
                      {index < workflow.length - 1 && <ArrowRight className="hidden size-4 text-muted-foreground transition-transform duration-300 group-hover:translate-x-1 md:block" />}
                    </div>
                    <h3 className="mt-5 text-base font-semibold">{item.title}</h3>
                    <p className="mt-2 text-sm leading-6 text-muted-foreground">{item.description}</p>
                  </div>
                </Reveal>
              ))}
            </div>
            <div className="mt-8 flex flex-wrap items-center gap-3 text-sm text-muted-foreground">
              <span className="inline-flex items-center gap-2"><Upload className="size-4 text-primary" />已有项目可上传 JSON 导入</span>
              <span className="text-border">/</span>
              <span className="inline-flex items-center gap-2"><Database className="size-4 text-primary" />支持本机或私有化数据库</span>
            </div>
          </div>
        </section>

        <section className="mx-auto w-full max-w-7xl scroll-mt-20 px-5 py-20 sm:px-8 lg:py-28" id="why">
          <div className="grid gap-12 lg:grid-cols-[1fr_0.9fr] lg:items-center">
            <Reveal className="space-y-6">
              <p className="text-xs font-medium uppercase tracking-[0.2em] text-primary">Why Veristack</p>
              <h2 className="max-w-xl text-balance text-3xl font-semibold leading-tight tracking-tight sm:text-4xl">
                交付纠纷，往往不是代码写没写，
                <span className="text-muted-foreground">而是当初约定的是什么。</span>
              </h2>
              <p className="max-w-xl text-sm leading-7 text-muted-foreground">
                客户说「我以为包含导出」，开发者说「需求单里没写」；需求改过几轮，没有人说得清最终范围。
                Veristack 不替你做交付，只负责让交付过程可被独立核验。
              </p>
              <div className="flex flex-wrap gap-2">
                {['外包交付', '定制开发', '私有化部署', '多团队协作'].map((tag) => (
                  <Badge key={tag} tone="neutral">{tag}</Badge>
                ))}
              </div>
            </Reveal>
            <Reveal delay={100} className="rounded-[calc(var(--radius-card)+0.35rem)] border bg-surface p-2 shadow-[0_24px_70px_-48px_var(--foreground)]">
              <div className="rounded-[var(--radius-card)] bg-surface-muted/70 p-5 sm:p-6">
                {[
                  { icon: Fingerprint, text: '谁在什么时候改了什么，全部可回溯' },
                  { icon: LockKeyhole, text: '冻结后的基线与交付清单不可被悄悄改写' },
                  { icon: ScanLine, text: '每个结论都有来源，AI 只提供待确认建议' },
                  { icon: ShieldCheck, text: '客户可以独立核验，不需要信任服务端' },
                ].map((item) => (
                  <div className="flex gap-3 border-b py-4 first:pt-0 last:border-b-0 last:pb-0" key={item.text}>
                    <item.icon className="mt-0.5 size-4 shrink-0 text-primary" />
                    <p className="text-sm leading-6">{item.text}</p>
                  </div>
                ))}
              </div>
            </Reveal>
          </div>
        </section>

        <section className="mx-auto w-full max-w-7xl px-5 pb-20 sm:px-8 lg:pb-28">
          <Reveal>
            <div className="relative overflow-hidden rounded-[calc(var(--radius-card)+0.5rem)] border bg-primary px-6 py-12 text-primary-foreground sm:px-10 sm:py-16">
              <div aria-hidden="true" className="landing-float pointer-events-none absolute -right-24 -top-24 size-80 rounded-full bg-primary-foreground/10 blur-3xl" />
              <div className="relative z-10 flex flex-col justify-between gap-8 md:flex-row md:items-end">
                <div className="max-w-xl space-y-3">
                  <p className="text-xs font-medium uppercase tracking-[0.2em] text-primary-foreground/70">Start with evidence</p>
                  <h2 className="text-3xl font-semibold leading-tight tracking-tight sm:text-4xl">从下一个项目开始，把交付记录留下来。</h2>
                  <p className="text-sm leading-7 text-primary-foreground/75">创建组织后即可新建项目、冻结基线，并邀请客户进入验收门户。</p>
                </div>
                <Link to="/auth" search={{ mode: 'register' }}>
                  <Button className="bg-background text-foreground hover:bg-background/90" size="lg">
                    免费创建组织
                    <ArrowRight className="size-4" />
                  </Button>
                </Link>
              </div>
            </div>
          </Reveal>
        </section>
      </main>

      <footer className="border-t bg-surface/40">
        <div className="mx-auto flex w-full max-w-7xl flex-col gap-4 px-5 py-8 text-xs text-muted-foreground sm:flex-row sm:items-center sm:justify-between sm:px-8">
          <p className="flex items-center gap-2"><ShieldCheck className="size-4 text-primary" />Veristack · 验迹，软件交付证据层</p>
          <div className="flex flex-wrap gap-5">
            <a className="hover:text-foreground" href="#capabilities">产品能力</a>
            <a className="hover:text-foreground" href="#workflow">工作流程</a>
            <Link className="hover:text-foreground" to="/auth">登录</Link>
            <Link className="hover:text-foreground" to="/auth" search={{ mode: 'register' }}>创建组织</Link>
          </div>
        </div>
      </footer>
    </div>
  )
}

/** 首屏右侧的交付快照轮播，用真实的产品字段解释「可核验」是什么。 */
function DeliverySnapshot() {
  const stageRef = useRef<HTMLDivElement>(null)
  const [activeSlide, setActiveSlide] = useState(0)
  const [isPaused, setIsPaused] = useState(false)
  const slide = snapshotSlides[activeSlide] ?? snapshotSlides[0]!

  useEffect(() => {
    if (isPaused) {
      return
    }
    if (window.matchMedia('(prefers-reduced-motion: reduce)').matches) {
      return
    }
    const timer = window.setInterval(() => {
      setActiveSlide((current) => (current + 1) % snapshotSlides.length)
    }, 5600)
    return () => window.clearInterval(timer)
  }, [isPaused])

  /** 鼠标在快照上移动时更新柔光位置与极轻微的 3D 倾斜，离开时复位。 */
  function handlePointerMove(event: ReactPointerEvent<HTMLDivElement>) {
    if (event.pointerType !== 'mouse') {
      return
    }
    const stage = stageRef.current
    if (!stage) {
      return
    }
    const rect = stage.getBoundingClientRect()
    const x = (event.clientX - rect.left) / rect.width
    const y = (event.clientY - rect.top) / rect.height
    stage.style.setProperty('--pointer-x', `${x * 100}%`)
    stage.style.setProperty('--pointer-y', `${y * 100}%`)
    stage.style.setProperty('--tilt-x', `${(0.5 - y) * 3}deg`)
    stage.style.setProperty('--tilt-y', `${(x - 0.5) * 3}deg`)
  }

  function resetPointer() {
    const stage = stageRef.current
    if (!stage) {
      return
    }
    stage.style.setProperty('--pointer-x', '50%')
    stage.style.setProperty('--pointer-y', '50%')
    stage.style.setProperty('--tilt-x', '0deg')
    stage.style.setProperty('--tilt-y', '0deg')
  }

  return (
    <div className="relative mx-auto w-full max-w-lg lg:max-w-none">
      <div aria-hidden="true" className="landing-float absolute -inset-5 rounded-[2rem] bg-primary/5 blur-2xl" />
      <div
        aria-label="交付快照示例轮播"
        aria-roledescription="carousel"
        className="delivery-stage group relative"
        role="region"
        onBlurCapture={() => setIsPaused(false)}
        onFocusCapture={() => setIsPaused(true)}
        onMouseEnter={() => setIsPaused(true)}
        onMouseLeave={() => setIsPaused(false)}
        onPointerLeave={resetPointer}
        onPointerMove={handlePointerMove}
        ref={stageRef}
      >
        <div className="relative overflow-hidden rounded-[calc(var(--radius-card)+0.45rem)] border bg-surface/90 shadow-[0_32px_90px_-48px_var(--foreground)] backdrop-blur">
          <div aria-hidden="true" className="delivery-spotlight pointer-events-none absolute inset-0 z-0" />
          <div className="relative z-10">
            <div className="flex items-center justify-between border-b bg-surface-muted/45 px-5 py-3.5">
              <div className="flex items-center gap-2 text-xs text-muted-foreground">
                <span className="flex gap-1.5" aria-hidden="true">
                  <span className="size-2 rounded-full bg-muted-foreground/30" />
                  <span className="size-2 rounded-full bg-muted-foreground/30" />
                  <span className="size-2 rounded-full bg-muted-foreground/30" />
                </span>
                <span className="ml-2 font-mono">delivery-snapshot.v2</span>
              </div>
              <span className="inline-flex items-center gap-1.5 text-xs font-medium text-success">
                <span className="size-1.5 rounded-full bg-success" />
                可核验
              </span>
            </div>

            <div className="p-5 sm:p-6">
              <div className="snapshot-slide-enter space-y-5" key={slide.id}>
                <div className="flex items-start justify-between gap-4">
                  <div className="space-y-1">
                    <p className="text-xs uppercase tracking-[0.16em] text-muted-foreground">{slide.detail}</p>
                    <h2 className="text-lg font-semibold tracking-tight">{slide.title}</h2>
                  </div>
                  <Badge tone={slide.badgeTone}>{slide.badge}</Badge>
                </div>

                <div className="grid grid-cols-2 gap-3">
                  {slide.metrics.map((metric) => (
                    <SnapshotStat
                      key={metric.label}
                      label={metric.label}
                      tone={metric.mono ? 'mono' : 'text'}
                      value={metric.value}
                    />
                  ))}
                </div>

                <div className="rounded-[var(--radius-card)] border bg-background/65 p-4">
                  <div className="mb-3 flex items-center justify-between text-xs">
                    <span className="font-medium">最近记录</span>
                    <span className="text-muted-foreground">按时间倒序</span>
                  </div>
                  <div className="space-y-3">
                    {slide.events.map((event) => (
                      <div className="flex items-center gap-3" key={event.title}>
                        <span className="flex size-8 shrink-0 items-center justify-center rounded-lg bg-surface-muted text-primary">
                          <event.icon className="size-4" />
                        </span>
                        <div className="min-w-0">
                          <p className="truncate text-xs font-medium">{event.title}</p>
                          <p className="mt-0.5 truncate text-[10px] text-muted-foreground">{event.meta}</p>
                        </div>
                      </div>
                    ))}
                  </div>
                </div>

                <div className={cn(
                  'flex items-center justify-between rounded-[var(--radius-card)] px-3.5 py-2.5 text-xs',
                  slide.badgeTone === 'warning' ? 'bg-warning/10 text-warning' : 'bg-success/10 text-success',
                )}>
                  <span className="inline-flex items-center gap-2"><Check className="size-3.5" />{slide.footerLabel}</span>
                  <span className="font-mono text-[10px] opacity-80">{slide.footerMeta}</span>
                </div>
              </div>
            </div>

            <div className="relative z-10 flex items-center justify-between gap-4 border-t bg-surface-muted/25 px-5 py-3">
              <div className="flex items-center gap-1.5" role="group" aria-label="切换交付快照">
                {snapshotSlides.map((item, index) => (
                  <button
                    aria-current={index === activeSlide ? 'true' : undefined}
                    aria-label={`查看${item.badge}的交付快照`}
                    className={cn(
                      'h-1.5 rounded-full transition-all duration-300',
                      index === activeSlide ? 'w-7 bg-primary' : 'w-3 bg-border hover:bg-muted-foreground',
                    )}
                    key={item.id}
                    onClick={() => setActiveSlide(index)}
                    type="button"
                  />
                ))}
              </div>
              <span className="text-[10px] text-muted-foreground">自动轮播 · 悬停暂停</span>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}

/** 使用场景滚动带，让首屏下方的信息密度更接近成熟产品官网。 */
function UseCaseTicker() {
  return (
    <section aria-label="适用场景" className="overflow-hidden border-b bg-surface/25">
      <div className="ticker-track flex w-max">
        {[0, 1, 2, 3].map((group) => (
          <div aria-hidden={group !== 0} className="flex shrink-0 items-center gap-2 px-3 py-3" key={group}>
            {useCases.map((item) => (
              <span className="inline-flex items-center gap-2 rounded-full border border-border/70 bg-background/65 px-3 py-1.5 text-xs text-muted-foreground" key={`${group}-${item}`}>
                <span className="size-1 rounded-full bg-primary" />
                {item}
              </span>
            ))}
          </div>
        ))}
      </div>
    </section>
  )
}

/** 首屏下方的能力指标。 */
function Metric({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex items-start gap-3 px-1 py-2 sm:px-4">
      <span className="mt-1 size-1.5 shrink-0 rounded-full bg-primary" />
      <div className="space-y-1">
        <p className="text-xs font-medium text-muted-foreground">{label}</p>
        <p className="text-sm font-medium text-foreground">{value}</p>
      </div>
    </div>
  )
}

/** 区块标题，统一 eyebrow、主标题和说明的层级。 */
function SectionHeading({ eyebrow, lead, title }: { eyebrow: string; lead: string; title: string }) {
  return (
    <div className="grid gap-5 md:grid-cols-[1.35fr_0.65fr] md:items-end">
      <div className="space-y-3">
        <p className="text-xs font-medium uppercase tracking-[0.2em] text-primary">{eyebrow}</p>
        <h2 className="max-w-2xl text-balance text-3xl font-semibold leading-tight tracking-tight sm:text-4xl">{title}</h2>
      </div>
      <p className="max-w-md text-sm leading-7 text-muted-foreground md:justify-self-end">{lead}</p>
    </div>
  )
}

/** 轻量滚动入场容器，减少首屏以下内容一次性出现的僵硬感。 */
function Reveal({ children, className, delay = 0 }: { children: ReactNode; className?: string; delay?: number }) {
  const ref = useRef<HTMLDivElement>(null)
  const [isVisible, setIsVisible] = useState(() => {
    if (typeof window === 'undefined') {
      return false
    }
    return typeof IntersectionObserver === 'undefined' || window.matchMedia('(prefers-reduced-motion: reduce)').matches
  })

  useEffect(() => {
    if (isVisible) {
      return
    }
    const element = ref.current
    if (!element) {
      return
    }
    const observer = new IntersectionObserver(([entry]) => {
      if (entry?.isIntersecting) {
        setIsVisible(true)
        observer.disconnect()
      }
    }, { rootMargin: '0px 0px -8% 0px', threshold: 0.08 })
    observer.observe(element)
    return () => observer.disconnect()
  }, [isVisible])

  return (
    <div className={cn('reveal', isVisible && 'reveal-visible', className)} ref={ref} style={{ transitionDelay: `${delay}ms` }}>
      {children}
    </div>
  )
}

/** 快照指标卡，支持普通文本和等宽摘要。 */
function SnapshotStat({ label, tone = 'text', value }: { label: string; tone?: 'mono' | 'text'; value: string }) {
  return (
    <div className="rounded-[var(--radius-card)] border bg-surface-muted/40 px-3.5 py-3">
      <p className="text-[10px] text-muted-foreground">{label}</p>
      <p className={tone === 'mono' ? 'mt-1 truncate font-mono text-xs font-medium' : 'mt-1 truncate text-sm font-medium'}>{value}</p>
    </div>
  )
}
