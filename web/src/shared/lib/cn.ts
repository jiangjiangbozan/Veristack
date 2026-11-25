import { clsx, type ClassValue } from 'clsx'
import { twMerge } from 'tailwind-merge'

/**
 * 合并条件类名并消除 Tailwind 冲突类。
 *
 * - 统一所有组件的类名组合入口，避免各页面重复拼接。
 */
export function cn(...inputs: ClassValue[]): string {
  return twMerge(clsx(inputs))
}
