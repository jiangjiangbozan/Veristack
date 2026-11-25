import { useEffect, useState, type ReactNode } from 'react'

import { ThemeContext, type Theme } from '@/app/theme-context'
import { darkClass, themeStorageKey } from '@/app/theme-storage'

/**
 * 主题 Provider，管理明暗双主题并在本地持久化选择。
 *
 * - 初始化时读取首屏已应用的主题，保持状态与 DOM 一致。
 */
export function ThemeProvider({ children }: { children: ReactNode }) {
  const [theme, setTheme] = useState<Theme>(() =>
    document.documentElement.classList.contains(darkClass) ? 'dark' : 'light',
  )

  useEffect(() => {
    document.documentElement.classList.toggle(darkClass, theme === 'dark')
    localStorage.setItem(themeStorageKey, theme)
  }, [theme])

  return (
    <ThemeContext.Provider value={{ theme, toggleTheme: () => setTheme((current) => (current === 'dark' ? 'light' : 'dark')) }}>
      {children}
    </ThemeContext.Provider>
  )
}
