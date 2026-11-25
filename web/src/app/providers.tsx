import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { useState, type ReactNode } from 'react'

const defaultQueryOptions = {
  queries: {
    retry: 1,
    staleTime: 30_000,
  },
}

/**
 * 应用级 Providers，集中挂载服务端状态容器。
 *
 * - QueryClient 在组件生命周期内只创建一次，避免路由切换丢失缓存。
 */
export function AppProviders({ children }: { children: ReactNode }) {
  const [queryClient] = useState(() => new QueryClient({ defaultOptions: defaultQueryOptions }))
  return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
}
