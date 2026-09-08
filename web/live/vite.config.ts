import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  server: {
    port: 5178,
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
})
