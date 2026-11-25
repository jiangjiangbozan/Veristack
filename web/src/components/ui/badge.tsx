import type { ReactNode } from 'react'

import type { Tone } from '@/components/ui/badge-tone'
import { cn } from '@/shared/lib/cn'

const toneClasses: Record<Tone, string> = {
  danger: 'bg-danger/15 text-danger',
  info: 'bg-info/15 text-info',
  neutral: 'bg-surface-muted text-muted-foreground',
  success: 'bg-success/15 text-success',
  warning: 'bg-warning/15 text-warning',
}

interface BadgeProps {
  children: ReactNode
  tone?: Tone
}

/** 状态徽标，使用语义色并适配双主题。 */
export function Badge({ children, tone = 'neutral' }: BadgeProps) {
  return (
    <span className={cn('inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium', toneClasses[tone])}>
      {children}
    </span>
  )
}
