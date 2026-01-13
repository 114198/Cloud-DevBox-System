import { useEffect, useCallback, useRef } from 'react'
import { useNavigate } from 'react-router-dom'
import { message } from 'antd'
import { useAuthStore } from '@/stores/auth'
import { authService } from '@/services/auth'

// Token refresh interval (check every minute)
const TOKEN_CHECK_INTERVAL = 60 * 1000

/**
 * Custom hook for authentication management
 * Handles automatic token refresh and session management
 */
export function useAuth() {
  const navigate = useNavigate()
  const {
    user,
    accessToken,
    refreshToken,
    isAuthenticated,
    isLoading,
    isTokenExpiringSoon,
    setTokens,
    logout: storeLogout,
    setLoading,
  } = useAuthStore()
  
  const refreshTimeoutRef = useRef<ReturnType<typeof setInterval> | null>(null)
  const isRefreshingRef = useRef(false)

  // Refresh token function
  const refreshAccessToken = useCallback(async () => {
    if (!refreshToken || isRefreshingRef.current) {
      return false
    }

    isRefreshingRef.current = true
    
    try {
      const response = await authService.refreshToken(refreshToken)
      setTokens(response.accessToken, response.refreshToken, response.expiresIn)
      return true
    } catch (error) {
      console.error('Token refresh failed:', error)
      return false
    } finally {
      isRefreshingRef.current = false
    }
  }, [refreshToken, setTokens])

  // Logout function
  const logout = useCallback(async () => {
    setLoading(true)
    
    try {
      // Call logout API to invalidate tokens on server
      await authService.logout()
    } catch (error) {
      // Ignore errors - we're logging out anyway
      console.error('Logout API error:', error)
    } finally {
      storeLogout()
      setLoading(false)
      navigate('/login', { replace: true })
      message.success('已退出登录')
    }
  }, [storeLogout, setLoading, navigate])

  // Check and refresh token periodically
  useEffect(() => {
    if (!isAuthenticated || !refreshToken) {
      return
    }

    const checkAndRefreshToken = async () => {
      if (isTokenExpiringSoon()) {
        const success = await refreshAccessToken()
        if (!success) {
          // Token refresh failed, logout user
          storeLogout()
          navigate('/login', { replace: true })
          message.warning('会话已过期，请重新登录')
        }
      }
    }

    // Check immediately
    checkAndRefreshToken()

    // Set up periodic check
    refreshTimeoutRef.current = setInterval(checkAndRefreshToken, TOKEN_CHECK_INTERVAL)

    return () => {
      if (refreshTimeoutRef.current) {
        clearInterval(refreshTimeoutRef.current)
      }
    }
  }, [isAuthenticated, refreshToken, isTokenExpiringSoon, refreshAccessToken, storeLogout, navigate])

  // Handle visibility change (refresh token when tab becomes visible)
  useEffect(() => {
    const handleVisibilityChange = async () => {
      if (document.visibilityState === 'visible' && isAuthenticated && isTokenExpiringSoon()) {
        await refreshAccessToken()
      }
    }

    document.addEventListener('visibilitychange', handleVisibilityChange)
    
    return () => {
      document.removeEventListener('visibilitychange', handleVisibilityChange)
    }
  }, [isAuthenticated, isTokenExpiringSoon, refreshAccessToken])

  return {
    user,
    accessToken,
    isAuthenticated,
    isLoading,
    logout,
    refreshAccessToken,
  }
}

/**
 * Hook to require authentication
 * Redirects to login if not authenticated
 */
export function useRequireAuth(redirectTo: string = '/login') {
  const navigate = useNavigate()
  const { isAuthenticated, isLoading } = useAuthStore()

  useEffect(() => {
    if (!isLoading && !isAuthenticated) {
      navigate(redirectTo, { replace: true })
    }
  }, [isAuthenticated, isLoading, navigate, redirectTo])

  return { isAuthenticated, isLoading }
}

/**
 * Hook to redirect authenticated users
 * Useful for login/register pages
 */
export function useRedirectIfAuthenticated(redirectTo: string = '/') {
  const navigate = useNavigate()
  const { isAuthenticated, isLoading } = useAuthStore()

  useEffect(() => {
    if (!isLoading && isAuthenticated) {
      navigate(redirectTo, { replace: true })
    }
  }, [isAuthenticated, isLoading, navigate, redirectTo])

  return { isAuthenticated, isLoading }
}

export default useAuth
