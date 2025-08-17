import axios from 'axios'

const API_BASE = import.meta.env.VITE_API_BASE || 'http://localhost:3003/api/v1'

const request = axios.create({
  baseURL: API_BASE,
  timeout: 5000
})

// 请求拦截器：自动带 token 或 user_id
request.interceptors.request.use(config => {
  const token = localStorage.getItem('token')
  if (token) {
    config.headers['Authorization'] = `Bearer ${token}`
  }
  const userId = localStorage.getItem('user_id')
  if (userId) {
    config.headers['X-User-ID'] = userId
  }
  return config
})

export default request
