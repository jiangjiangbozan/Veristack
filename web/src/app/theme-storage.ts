import type { Theme } from '@/app/theme-context'

export const themeStorageKey = 'veristack.theme'
export const darkClass = 'dark'

/** 在首屏渲染前同步应用主题，避免闪白或闪黑。 */
export function applyInitialTheme(): void {
  const stored = localStorage.getItem(themeStorageKey)
  const prefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches
  const theme: Theme = stored === 'dark' || stored === 'light' ? stored : prefersDark ? 'dark' : 'light'
  document.documentElement.classList.toggle(darkClass, theme === 'dark')
}
