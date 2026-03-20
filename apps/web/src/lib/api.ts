import axios from 'axios'
import { clearAuthSession, requiresMfa } from './auth'

const API_BASE = import.meta.env.VITE_API_URL || '/api'

const api = axios.create({
  baseURL: API_BASE,
  headers: { 'Content-Type': 'application/json' },
})

api.interceptors.request.use((config) => {
  const token = localStorage.getItem('user_token') || localStorage.getItem('admin_token')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

api.interceptors.response.use(
  (res) => res,
  (err) => {
    const status = err.response?.status
    if (status === 401) {
      const pathname = window.location.pathname
      const isAdminRoute = pathname.startsWith('/admin')
      
      if (isAdminRoute) {
        clearAuthSession()
        localStorage.removeItem('admin_token')
        localStorage.removeItem('admin_profile')
        window.location.href = '/admin/login'
      }
    }

    if (status === 403) {
      const pathname = window.location.pathname
      if (pathname.startsWith('/admin') && requiresMfa()) {
        window.location.href = '/admin/profile'
      }
    }

    return Promise.reject(err)
  }
)

export default api
