import axios, { AxiosError, type AxiosInstance, type AxiosRequestConfig } from 'axios'
import { ElMessage } from 'element-plus'

export interface ApiEnvelope<T> {
  code: number
  message: string
  data: T
}

export const CODE_UNAUTHORIZED = 40100

const TOKEN_KEY = 'gonghe-token'

export function getToken(): string {
  return localStorage.getItem(TOKEN_KEY) || ''
}

export function setToken(token: string): void {
  localStorage.setItem(TOKEN_KEY, token)
}

export function clearToken(): void {
  localStorage.removeItem(TOKEN_KEY)
}

let onUnauthorized: (() => void) | null = null
export function setUnauthorizedHandler(handler: () => void): void {
  onUnauthorized = handler
}

export class ApiError extends Error {
  code: number
  status: number
  constructor(message: string, code: number, status: number) {
    super(message)
    this.name = 'ApiError'
    this.code = code
    this.status = status
  }
}

const http: AxiosInstance = axios.create({
  baseURL: '/api/v1',
  timeout: 15000,
  headers: { 'Content-Type': 'application/json' },
})

http.interceptors.request.use((config) => {
  const token = getToken()
  if (token) {
    config.headers = config.headers || {}
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

http.interceptors.response.use(
  (response) => response,
  (error: AxiosError<ApiEnvelope<unknown>>) => {
    const status = error.response?.status ?? 0
    const payload = error.response?.data
    const code = payload?.code ?? status * 100
    let message = payload?.message || ''

    if (!message) {
      if (error.code === 'ECONNABORTED') {
        message = '请求超时，请检查网络或服务状态'
      } else if (status === 0) {
        message = '无法连接服务器，请确认后端已启动'
      } else {
        message = `请求失败（HTTP ${status}）`
      }
    }

    if (status === 401) {
      clearToken()
      onUnauthorized?.()
      if (!error.config?.url?.includes('/auth/login')) {
        ElMessage.error(message)
      }
    } else if (status === 429) {
      ElMessage.warning(message)
    }

    return Promise.reject(new ApiError(message, code, status))
  },
)

async function unwrap<T>(config: AxiosRequestConfig): Promise<T> {
  const resp = await http.request<ApiEnvelope<T>>(config)
  const body = resp.data
  if (body && typeof body === 'object' && 'code' in body) {
    if (body.code !== 0) {
      throw new ApiError(body.message || '请求失败', body.code, resp.status)
    }
    return body.data
  }
  return body as unknown as T
}

export const request = {
  get: <T>(url: string, params?: Record<string, unknown>) => unwrap<T>({ method: 'GET', url, params }),
  post: <T>(url: string, data?: unknown) => unwrap<T>({ method: 'POST', url, data }),
  patch: <T>(url: string, data?: unknown) => unwrap<T>({ method: 'PATCH', url, data }),
  delete: <T>(url: string, params?: Record<string, unknown>) => unwrap<T>({ method: 'DELETE', url, params }),
}
