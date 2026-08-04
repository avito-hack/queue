import axios from 'axios'


export const api = axios.create({
  baseURL: import.meta.env.VITE_API_URL ?? 'http://localhost:8080',
  headers: { 'Content-Type': 'application/json' },
})


api.interceptors.request.use((config) => {
    const token = localStorage.getItem('authToken') ?? 'user-1'
    config.headers.Authorization = `Bearer ${token}`
    return config
  })