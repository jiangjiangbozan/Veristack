import { createContext, useContext } from 'react'

/** 主题模式枚举。 */
export type Theme = 'dark' | 'light'

/** 主题上下文暴露的状态与操作。 */
export interface ThemeContextValue {
  theme: Theme
  toggleTheme: () => void
}

/** 主题上下文，供 Provider 与 Hook 共享。 */
export const ThemeContext = createContext<ThemeContextValue | null>(null)

/** 读取当前主题与切换方法。 */
export function useTheme(): ThemeContextValue {
  const context = useContext(ThemeContext)
  if (!context) {
    throw new Error('useTheme 必须在 ThemeProvider 内使用')
  }
  return context
}
