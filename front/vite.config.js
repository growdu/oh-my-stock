import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import path from 'path'

// 把 /api 转到同机后端；部署机器不同则用 VITE_API_BACKEND=http://host:port 覆盖
const backend = process.env.VITE_API_BACKEND || 'http://127.0.0.1:3003'

export default defineConfig({
  plugins: [vue()],
  server: {
    host: '0.0.0.0',
    port: 5713,
    proxy: {
      '/api': {
        target: backend,
        changeOrigin: true,
        // 不再 strip /api：直接由后端处理 /api/* 路径
      },
    },
  },
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
})
