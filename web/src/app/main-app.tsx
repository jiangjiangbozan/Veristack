import { RouterProvider } from '@tanstack/react-router'

import { AppProviders } from '@/app/providers'
import { AuthProvider } from '@/features/auth/auth-provider'
import { router } from '@/routes/router'

/** 应用根组件，集中挂载全局 Providers 与路由。 */
export function MainApp() {
  return (
    <AppProviders>
      <AuthProvider>
        <RouterProvider router={router} />
      </AuthProvider>
    </AppProviders>
  )
}
