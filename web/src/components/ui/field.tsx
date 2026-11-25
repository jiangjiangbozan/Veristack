import type { InputHTMLAttributes, ReactNode, SelectHTMLAttributes, TextareaHTMLAttributes } from 'react'

import { cn } from '@/shared/lib/cn'

const controlClass =
  'w-full rounded-[var(--radius-card)] border bg-surface px-3 py-2 text-sm text-foreground outline-none transition-colors duration-200 placeholder:text-muted-foreground focus:ring-2 focus:ring-[var(--ring)] disabled:opacity-50'

interface InputProps extends InputHTMLAttributes<HTMLInputElement> {
  label?: string
}

/** 输入框，可带标签。 */
export function Input({ className, label, ...props }: InputProps) {
  return (
    <label className="block space-y-1.5">
      {label && <span className="block text-xs font-medium text-muted-foreground">{label}</span>}
      <input className={cn(controlClass, className)} {...props} />
    </label>
  )
}

interface TextareaProps extends TextareaHTMLAttributes<HTMLTextAreaElement> {
  label?: string
}

/** 多行输入框，可带标签。 */
export function Textarea({ className, label, ...props }: TextareaProps) {
  return (
    <label className="block space-y-1.5">
      {label && <span className="block text-xs font-medium text-muted-foreground">{label}</span>}
      <textarea className={cn(controlClass, 'min-h-20 resize-y', className)} {...props} />
    </label>
  )
}

interface SelectProps extends SelectHTMLAttributes<HTMLSelectElement> {
  children: ReactNode
  label?: string
}

/** 下拉选择，统一触发器样式。 */
export function Select({ children, className, label, ...props }: SelectProps) {
  return (
    <label className="block space-y-1.5">
      {label && <span className="block text-xs font-medium text-muted-foreground">{label}</span>}
      <select className={cn(controlClass, className)} {...props}>
        {children}
      </select>
    </label>
  )
}
