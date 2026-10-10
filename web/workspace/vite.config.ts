import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  server: {
    port: 5180,
    strictPort: true,
    proxy: {
      // 开发环境同源健康检查：转发到本地子系统 dev server（生产由 nginx 代理）
      '/health/live': {
        target: 'http://127.0.0.1:5178',
        changeOrigin: true,
        rewrite: () => '/',
      },
      '/health/socketio': {
        target: 'http://127.0.0.1:5179',
        changeOrigin: true,
        rewrite: () => '/',
      },
    },
  },
})
