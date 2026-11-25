import { createRootRoute, createRoute, createRouter } from '@tanstack/react-router'

import { AcceptancePortalPage } from '@/features/acceptance/acceptance-portal-page'
import { AuthPage } from '@/features/auth/auth-page'
import { DeliveryDetailPage } from '@/features/deliveries/delivery-detail-page'
import { ProjectDetailPage } from '@/features/projects/project-detail-page'
import { ProjectListPage } from '@/features/projects/project-list-page'
import { AppLayout } from '@/layouts/app-layout'

const rootRoute = createRootRoute({ component: AppLayout })

const authRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/auth',
  component: AuthPage,
})

const indexRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/',
  component: ProjectListPage,
})

const projectDetailRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: '/projects/$projectId',
  component: () => {
    const { projectId } = projectDetailRoute.useParams()
    return <ProjectDetailPage projectId={projectId} />
  },
})

const deliveryDetailRoute = createRoute({
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
  indexRoute,
  projectDetailRoute,
  deliveryDetailRoute,
  acceptancePortalRoute,
])

/** 应用路由器，挂载认证、项目、交付与客户验收门户路由。 */
export const router = createRouter({ routeTree })

declare module '@tanstack/react-router' {
  interface Register {
    router: typeof router
  }
}
