import axios from 'axios'
import { ElMessage } from 'element-plus'

const DEFAULT_BASE = '/api/v1'
export const API_BASE = import.meta.env.VITE_API_BASE || DEFAULT_BASE

const request = axios.create({
  baseURL: API_BASE,
  timeout: 10000,
})

// 请求拦截：自动带 Bearer token
request.interceptors.request.use(config => {
  const token = localStorage.getItem('token')
  if (token) {
    config.headers['Authorization'] = `Bearer ${token}`
  }
  return config
})

// 响应拦截：401 / 网络错误统一提示
request.interceptors.response.use(
  res => res.data,
  err => {
    const status = err.response?.status
    const msg = err.response?.data?.error || err.message || '网络错误'
    if (status === 401) {
      ElMessage.error('登录状态已过期，请重新登录')
      localStorage.removeItem('token')
      localStorage.removeItem('user_id')
      if (!window.location.pathname.startsWith('/login')) {
        window.location.href = '/login'
      }
    } else {
      ElMessage.error(msg)
    }
    return Promise.reject(err)
  }
)

export default request
