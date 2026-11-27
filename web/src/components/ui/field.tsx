import { CircleAlert } from 'lucide-react'
import { forwardRef, useId, type InputHTMLAttributes, type ReactNode, type SelectHTMLAttributes, type TextareaHTMLAttributes } from 'react'

import { cn } from '@/shared/lib/cn'

const controlClass =
  'w-full rounded-[var(--radius-card)] border bg-surface px-3 py-2 text-sm text-foreground outline-none transition-colors duration-200 placeholder:text-muted-foreground focus:ring-2 focus:ring-[var(--ring)] disabled:opacity-50'

interface FieldChromeProps {
  error?: string // 字段级错误，展示在控件下方
  hint?: string // 辅助说明，错误出现时让位给错误
  id?: string // 外部指定控件 ID，便于测试与 label 关联
  label?: string // 字段标签
  required?: boolean // 用于必填标记与无障碍语义；是否弹出浏览器原生提示由表单 noValidate 决定
}

/** 渲染标签、控件下方错误与辅助说明，统一所有输入控件的表单语义。 */
function FieldChrome({
  children,
  error,
  hint,
  id,
  label,
  required,
}: FieldChromeProps & { children: ReactNode }) {
  const generatedId = useId()
  const fieldId = id ?? generatedId
  const messageId = `${fieldId}-message`
  return (
    <div className="space-y-1.5">
      {label && (
        <label className="flex items-center gap-1 text-xs font-medium text-muted-foreground" htmlFor={fieldId}>
          {label}
          {required && <span aria-hidden="true" className="text-danger">*</span>}
        </label>
      )}
      {children}
      {error ? (
        <p className="flex items-start gap-1 text-xs leading-5 text-danger" id={messageId} role="alert">
          <CircleAlert className="mt-0.5 size-3.5 shrink-0" />
          <span>{error}</span>
        </p>
      ) : (
        hint && <p className="text-xs leading-5 text-muted-foreground" id={messageId}>{hint}</p>
      )}
    </div>
  )
}

interface InputProps extends InputHTMLAttributes<HTMLInputElement>, FieldChromeProps {}

/** 输入框，带标签、必填标记、字段级错误与辅助说明。 */
export const Input = forwardRef<HTMLInputElement, InputProps>(function Input(
  { className, error, hint, id, label, required, ...props },
  ref,
) {
  const generatedId = useId()
  const fieldId = id ?? generatedId
  const describedBy = error || hint ? `${fieldId}-message` : undefined
  return (
    <FieldChrome error={error} hint={hint} id={fieldId} label={label} required={required}>
      <input
        aria-describedby={describedBy}
        aria-invalid={error ? true : undefined}
        className={cn(controlClass, error && 'border-danger focus:border-danger focus:ring-danger/30', className)}
        id={fieldId}
        ref={ref}
        required={required}
        {...props}
      />
    </FieldChrome>
  )
})

interface TextareaProps extends TextareaHTMLAttributes<HTMLTextAreaElement>, FieldChromeProps {}

/** 多行输入框，与单行输入共享标签和错误语义。 */
export const Textarea = forwardRef<HTMLTextAreaElement, TextareaProps>(function Textarea(
  { className, error, hint, id, label, required, ...props },
  ref,
) {
  const generatedId = useId()
  const fieldId = id ?? generatedId
  const describedBy = error || hint ? `${fieldId}-message` : undefined
  return (
    <FieldChrome error={error} hint={hint} id={fieldId} label={label} required={required}>
      <textarea
        aria-describedby={describedBy}
        aria-invalid={error ? true : undefined}
        className={cn(controlClass, 'min-h-20 resize-y', error && 'border-danger focus:border-danger focus:ring-danger/30', className)}
        id={fieldId}
        ref={ref}
        required={required}
        {...props}
      />
    </FieldChrome>
  )
})

interface SelectProps extends SelectHTMLAttributes<HTMLSelectElement>, FieldChromeProps {
  children: ReactNode
}

/** 下拉选择，统一触发器样式与错误提示。 */
export const Select = forwardRef<HTMLSelectElement, SelectProps>(function Select(
  { children, className, error, hint, id, label, required, ...props },
  ref,
) {
  const generatedId = useId()
  const fieldId = id ?? generatedId
  const describedBy = error || hint ? `${fieldId}-message` : undefined
  return (
    <FieldChrome error={error} hint={hint} id={fieldId} label={label} required={required}>
      <select
        aria-describedby={describedBy}
        aria-invalid={error ? true : undefined}
        className={cn(controlClass, error && 'border-danger focus:border-danger focus:ring-danger/30', className)}
        id={fieldId}
        ref={ref}
        required={required}
        {...props}
      >
        {children}
      </select>
    </FieldChrome>
  )
})
