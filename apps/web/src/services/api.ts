import axios, { AxiosError, InternalAxiosRequestConfig } from 'axios'
import { useAuthStore } from '@/stores/auth'

// Create axios instance
const api = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL || '/api/v1',
  timeout: 10000,
  headers: {
    'Content-Type': 'application/json',
  },
})

// Flag to prevent multiple refresh requests
let isRefreshing = false
// Queue of failed requests to retry after token refresh
let failedQueue: Array<{
  resolve: (value: unknown) => void
  reject: (reason?: unknown) => void
}> = []

// Process the queue of failed requests
const processQueue = (error: Error | null, token: string | null = null) => {
  failedQueue.forEach((promise) => {
    if (error) {
      promise.reject(error)
    } else {
      promise.resolve(token)
    }
  })
  failedQueue = []
}

// Request interceptor to add auth token
api.interceptors.request.use(
  (config: InternalAxiosRequestConfig) => {
    const { accessToken, isTokenExpired } = useAuthStore.getState()
    
    // Don't add token to refresh endpoint
    if (config.url?.includes('/auth/refresh')) {
      return config
    }
    
    if (accessToken && !isTokenExpired()) {
      config.headers.Authorization = `Bearer ${accessToken}`
    }
    
    return config
  },
  (error) => Promise.reject(error)
)

// Response interceptor for error handling and token refresh
api.interceptors.response.use(
  (response) => response,
  async (error: AxiosError) => {
    const originalRequest = error.config as InternalAxiosRequestConfig & { _retry?: boolean }
    
    // Handle 401 Unauthorized errors
    if (error.response?.status === 401 && !originalRequest._retry) {
      // Don't retry refresh token requests
      if (originalRequest.url?.includes('/auth/refresh')) {
        useAuthStore.getState().logout()
        window.location.href = '/login'
        return Promise.reject(error)
      }
      
      // If already refreshing, queue this request
      if (isRefreshing) {
        return new Promise((resolve, reject) => {
          failedQueue.push({ resolve, reject })
        })
          .then((token) => {
            originalRequest.headers.Authorization = `Bearer ${token}`
            return api(originalRequest)
          })
          .catch((err) => Promise.reject(err))
      }
      
      originalRequest._retry = true
      isRefreshing = true
      
      const { refreshToken, setTokens, logout } = useAuthStore.getState()
      
      if (!refreshToken) {
        logout()
        window.location.href = '/login'
        return Promise.reject(error)
      }
      
      try {
        // Attempt to refresh the token
        const response = await axios.post(
          `${api.defaults.baseURL}/auth/refresh`,
          { refreshToken },
          { headers: { 'Content-Type': 'application/json' } }
        )
        
        const { accessToken: newAccessToken, refreshToken: newRefreshToken, expiresIn } = response.data
        
        // Update tokens in store
        setTokens(newAccessToken, newRefreshToken, expiresIn)
        
        // Process queued requests
        processQueue(null, newAccessToken)
        
        // Retry original request
        originalRequest.headers.Authorization = `Bearer ${newAccessToken}`
        return api(originalRequest)
      } catch (refreshError) {
        // Refresh failed, logout user
        processQueue(refreshError as Error, null)
        logout()
        window.location.href = '/login'
        return Promise.reject(refreshError)
      } finally {
        isRefreshing = false
      }
    }
    
    // Handle other errors
    if (error.response?.status === 403) {
      // Forbidden - user doesn't have permission
      console.error('Access forbidden:', error.response.data)
    } else if (error.response?.status === 429) {
      // Rate limited
      console.error('Rate limited:', error.response.data)
    } else if (error.response?.status === 500) {
      // Server error
      console.error('Server error:', error.response.data)
    }
    
    return Promise.reject(error)
  }
)

export default api

// Helper function to check if user is authenticated
export const isAuthenticated = (): boolean => {
  const { isAuthenticated, isTokenExpired } = useAuthStore.getState()
  return isAuthenticated && !isTokenExpired()
}

// Helper function to get current access token
export const getAccessToken = (): string | null => {
  const { accessToken, isTokenExpired } = useAuthStore.getState()
  if (accessToken && !isTokenExpired()) {
    return accessToken
  }
  return null
}
