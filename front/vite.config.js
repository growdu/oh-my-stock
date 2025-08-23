import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import path from "path"; // ✅ 这里必须导入 path

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
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'), // ✅ 让 @ 指向 src
    },
  },
})
