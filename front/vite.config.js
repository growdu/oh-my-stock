import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

// https://vite.dev/config/
export default defineConfig({
  plugins: [vue()],
  server: {
    host: '0.0.0.0',
    port: 5713,
    proxy: {
      '/api': {
        target: 'http://192.168.3.99:3003', // 你的 Gin 后端地址
        changeOrigin: true
      }
    }
  },
})
