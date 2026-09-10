import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  server: {
    port: 5178,
    strictPort: true,
    proxy: {
      '/livekit': {
        target: 'http://127.0.0.1:7880',
        changeOrigin: true,
        ws: true,
        rewrite: (path) => path.replace(/^\/livekit/, ''),
      },
      '/live': 'http://127.0.0.1:11002',
      '/socket.io': {
        target: 'http://127.0.0.1:11003',
        ws: true,
      },
    },
  },
  build: {
    rollupOptions: {
      output: {
        manualChunks(id) {
          if (id.includes('node_modules/livekit-client')) return 'livekit-client'
          if (id.includes('node_modules/@livekit')) return 'livekit-components'
          if (id.includes('node_modules')) return 'vendor'
        },
      },
    },
  },
})
