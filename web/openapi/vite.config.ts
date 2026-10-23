import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// 生产构建/预览挂载在 /openapi/ 路径下；开发态保持根路径，便于本机直接访问。
export default defineConfig(({ mode }) => ({
  plugins: [react()],
  base: mode === 'production' ? '/openapi/' : '/',
  server: {
    host: '127.0.0.1',
    port: 5181,
    strictPort: true,
    proxy: {
      // 开发态同源转发到本地 opengtw，避免浏览器跨域。
      '/v1': {
        target: 'http://127.0.0.1:11005',
        changeOrigin: true,
      },
      '/open-admin': {
        target: 'http://127.0.0.1:11005',
        changeOrigin: true,
      },
    },
  },
  build: {
    rollupOptions: {
      output: {
        manualChunks(id) {
          if (id.includes('node_modules')) return 'vendor'
        },
      },
    },
  },
}))
