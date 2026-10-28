import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import pkg from './package.json'

export default defineConfig({
  plugins: [react()],
  define: {
    __APP_VERSION__: JSON.stringify(pkg.version),
    __BUILD_TIME__: JSON.stringify(new Date().toISOString()),
  },
  server: {
    host: '127.0.0.1',
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
      '/health/openapi': {
        target: 'http://127.0.0.1:5181',
        changeOrigin: true,
        rewrite: () => '/',
      },
      // GitLab 走 nginx 子域名反代；本地没有网关时该项显示未启动，属于预期。
      '/health/gitlab': {
        target: 'http://127.0.0.1:80',
        changeOrigin: true,
        headers: { Host: 'gitlab.localhost' },
        rewrite: () => '/users/sign_in',
      },
    },
  },
})
