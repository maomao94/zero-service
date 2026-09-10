import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  server: {
    port: 5179,
    proxy: {
      '/socket.io': {
        target: 'http://127.0.0.1:11003',
        ws: true,
      },
    },
  },
})
