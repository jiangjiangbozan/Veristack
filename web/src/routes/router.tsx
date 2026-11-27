import { createRootRoute, createRoute, createRouter, redirect } from '@tanstack/react-router'

import { AcceptancePortalPage } from '@/features/acceptance/acceptance-portal-page'
import { AuthPage } from '@/features/auth/auth-page'
import { ResetPasswordPage } from '@/features/auth/reset-password-page'
import { DeliveryDetailPage } from '@/features/deliveries/delivery-detail-page'
import { NotificationsPage } from '@/features/notifications/notifications-page'
import { ProjectDetailPage } from '@/features/projects/project-detail-page'
import { ProjectImportPage } from '@/features/projects/project-import-page'
import { ProjectListPage } from '@/features/projects/project-list-page'
import { NotFoundPage, ServerErrorPage } from '@/features/system/status-page'
import { AppLayout } from '@/layouts/app-layout'
import { getAccessToken } from '@/services/http-client'

/** 管理端路由统一要求登录；未登录时回到登录页，而不是把首页伪装成目标页面。 */
function requireAuth() {
  if (!getAccessToken()) {
    throw redirect({ to: '/auth' })
  }
}

const rootRoute = createRootRoute({
  component: AppLayout,
  errorComponent: ServerErrorPage,
  notFoundComponent: NotFoundPage,
})

const authRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/auth',
  validateSearch: (search: Record<string, unknown>): { mode?: 'forgot' | 'login' | 'register' } => {
    const mode = search.mode
    return mode === 'register' || mode === 'forgot' ? { mode } : {}
  },
  component: () => {
    const { mode } = authRoute.useSearch()
    return <AuthPage initialMode={mode} />
  },
})

const resetPasswordRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/auth/reset/$token',
  component: () => {
    const { token } = resetPasswordRoute.useParams()
    return <ResetPasswordPage token={token} />
  },
})

const indexRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/',
  component: ProjectListPage,
})

const notificationsRoute = createRoute({
  beforeLoad: requireAuth,
  getParentRoute: () => rootRoute,
  path: '/notifications',
  component: NotificationsPage,
})

const projectDetailRoute = createRoute({
  beforeLoad: requireAuth,
  getParentRoute: () => rootRoute,
  path: '/projects/$projectId',
  component: () => {
    const { projectId } = projectDetailRoute.useParams()
    return <ProjectDetailPage projectId={projectId} />
  },
})

const projectImportRoute = createRoute({
  beforeLoad: requireAuth,
  getParentRoute: () => rootRoute,
  path: '/projects/import',
  component: ProjectImportPage,
})

const deliveryDetailRoute = createRoute({
  beforeLoad: requireAuth,
  getParentRoute: () => rootRoute,
  path: '/deliveries/$deliveryId',
  component: () => {
    const { deliveryId } = deliveryDetailRoute.useParams()
    return <DeliveryDetailPage deliveryId={deliveryId} />
  },
})

const acceptancePortalRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/accept/$token',
  component: () => {
    const { token } = acceptancePortalRoute.useParams()
    return <AcceptancePortalPage token={token} />
  },
})

const routeTree = rootRoute.addChildren([
  authRoute,
  resetPasswordRoute,
  indexRoute,
  notificationsRoute,
  projectDetailRoute,
  projectImportRoute,
  deliveryDetailRoute,
  acceptancePortalRoute,
])

/** 应用路由器，挂载认证、项目、交付与客户验收门户路由。 */
export const router = createRouter({ defaultNotFoundComponent: NotFoundPage, routeTree })

declare module '@tanstack/react-router' {
  interface Register {
    router: typeof router
  }
}
