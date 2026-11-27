import { CircleAlert, CircleCheck, Info } from 'lucide-react'
import type { ReactNode } from 'react'
import { cva, type VariantProps } from 'class-variance-authority'

import { cn } from '@/shared/lib/cn'

const alertVariants = cva('flex items-start gap-3 rounded-[var(--radius-card)] border px-3.5 py-3 text-sm leading-6', {
  variants: {
    tone: {
      danger: 'border-danger/30 bg-danger/10 text-danger',
      info: 'border-info/30 bg-info/10 text-info',
      success: 'border-success/30 bg-success/10 text-success',
      warning: 'border-warning/35 bg-warning/10 text-warning',
    },
  },
  defaultVariants: {
    tone: 'info',
  },
})

const iconByTone = {
  danger: CircleAlert,
  info: Info,
  success: CircleCheck,
  warning: CircleAlert,
} as const

interface AlertProps extends VariantProps<typeof alertVariants> {
  children: ReactNode // 提示正文
  className?: string // 外部布局样式
  title?: string // 可选标题，用于区分错误来源
}

/** 统一提示条，用于接口错误、表单成功反馈与服务状态说明。 */
export function Alert({ children, className, title, tone = 'info' }: AlertProps) {
  const Icon = iconByTone[tone ?? 'info']
  return (
    <div aria-live="polite" className={cn(alertVariants({ tone }), className)} role={tone === 'danger' ? 'alert' : 'status'}>
      <Icon className="mt-0.5 size-4 shrink-0" />
      <div className="space-y-1">
        {title && <p className="font-medium">{title}</p>}
        <div className="opacity-90">{children}</div>
      </div>
    </div>
  )
}
