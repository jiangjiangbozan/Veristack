import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'

import { ThemeProvider } from '@/app/theme-provider'
import { applyInitialTheme } from '@/app/theme-storage'
import { MainApp } from '@/app/main-app'
import './index.css'

applyInitialTheme()

const rootElement = document.getElementById('root')
if (!rootElement) {
  throw new Error('找不到应用挂载节点 #root')
}

createRoot(rootElement).render(
  <StrictMode>
    <ThemeProvider>
      <MainApp />
    </ThemeProvider>
  </StrictMode>,
)
