import axios from 'axios'
import { waitForAuthToken } from '../auth/demoAuth'

export const api = axios.create({
  baseURL: import.meta.env.VITE_PUBLIC_BASE_URL ?? '',
  headers: { 'Content-Type': 'application/json' },
})

declare module 'axios' {
  export interface AxiosRequestConfig {
    skipAuth?: boolean
  }
}

api.interceptors.request.use(async (config) => {
  if (config.skipAuth) {
    return config
  }

  const token = await waitForAuthToken()
  config.headers.Authorization = `Bearer ${token}`
  return config
})
