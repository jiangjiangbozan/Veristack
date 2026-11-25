import { useEffect, useState, type ReactNode } from 'react'

import { AuthContext } from '@/features/auth/auth-context'
import { fetchCurrentUser } from '@/services/auth-service'
import { getAccessToken } from '@/services/http-client'
import type { UserView } from '@/shared/types/domain'

interface AuthState {
  isRestoring: boolean
  user: UserView | null
}

/** 根据是否存在本地令牌决定初始是否需要恢复会话。 */
function initialAuthState(): AuthState {
  return { isRestoring: getAccessToken() !== null, user: null }
}

/** 认证 Provider，根据本地令牌恢复当前用户。 */
export function AuthProvider({ children }: { children: ReactNode }) {
  const [state, setState] = useState<AuthState>(initialAuthState)

  useEffect(() => {
    if (!getAccessToken()) {
      return
    }
    let active = true
    fetchCurrentUser()
      .then((result) => {
        if (active) {
          setState({ isRestoring: false, user: result.user })
        }
      })
      .catch(() => {
        if (active) {
          setState({ isRestoring: false, user: null })
        }
      })
    return () => {
      active = false
    }
  }, [])

  if (state.isRestoring) {
    return <div className="flex min-h-svh items-center justify-center text-sm text-muted-foreground">正在恢复会话…</div>
  }

  return (
    <AuthContext.Provider value={{ isAuthenticated: state.user !== null, setUser: (user) => setState({ isRestoring: false, user }), user: state.user }}>
      {children}
    </AuthContext.Provider>
  )
}
