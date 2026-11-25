import { createContext, useContext } from 'react'

import type { UserView } from '@/shared/types/domain'

/** 认证上下文暴露的当前用户与登录状态。 */
export interface AuthContextValue {
  isAuthenticated: boolean
  setUser: (user: UserView | null) => void
  user: UserView | null
}

/** 认证上下文。 */
export const AuthContext = createContext<AuthContextValue | null>(null)

/** 读取认证状态。 */
export function useAuth(): AuthContextValue {
  const context = useContext(AuthContext)
  if (!context) {
    throw new Error('useAuth 必须在 AuthProvider 内使用')
  }
  return context
}
