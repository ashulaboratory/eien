import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'

// https://vite.dev/config/
export default defineConfig({
  plugins: [react(), tailwindcss()],
  server: {
    proxy: {
      // /api/* リクエストを Goバックエンド (port 8080) に転送
      '/api': 'http://localhost:8080',
    },
  },
})