import type { HTMLAttributes, ReactNode } from 'react'

import { cn } from '@/shared/lib/cn'

interface CardProps extends HTMLAttributes<HTMLDivElement> {
  children?: ReactNode
}

/** 卡片容器，提供统一圆角、边框和表面色。 */
export function Card({ className, ...props }: CardProps) {
  return <div className={cn('rounded-[var(--radius-card)] border bg-surface', className)} {...props} />
}

interface CardContentProps extends HTMLAttributes<HTMLDivElement> {
  children?: ReactNode
}

/** 卡片内容区，统一内边距。 */
export function CardContent({ className, ...props }: CardContentProps) {
  return <div className={cn('p-5', className)} {...props} />
}

interface CardTitleProps extends HTMLAttributes<HTMLHeadingElement> {
  children?: ReactNode
}

/** 卡片标题，使用前景色与中等字重建立层级。 */
export function CardTitle({ className, ...props }: CardTitleProps) {
  return <h3 className={cn('text-base font-medium text-foreground', className)} {...props} />
}
