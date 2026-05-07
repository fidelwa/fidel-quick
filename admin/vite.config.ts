import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'
import path from 'path'

export default defineConfig(({ command }) => ({
  // Prod (build): SPA se sirve detrás de /admin/ en Cloud Run; los assets
  // y los imports relativos del bundle deben referenciar /admin/assets/...
  // Dev (serve): vite sirve en localhost:5173/ — base = '/'.
  base: command === 'build' ? '/admin/' : '/',
  plugins: [react(), tailwindcss()],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  server: {
    proxy: {
      '/api/v1/onboarding': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
      '/api/v1/auth': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },
}))
