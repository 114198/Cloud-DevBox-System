import { useEffect, useState, ReactNode } from 'react'
import { Spin } from 'antd'
import { useAuthStore } from '@/stores/auth'
import { authService } from '@/services/auth'

interface AuthProviderProps {
  children: ReactNode
}

/**
 * AuthProvider component
 * Handles initial authentication state restoration and token validation
 */
export function AuthProvider({ children }: AuthProviderProps) {
  const [isInitializing, setIsInitializing] = useState(true)
  const { 
    isAuthenticated, 
    accessToken, 
    refreshToken,
    isTokenExpired,
    setTokens,
    updateUser,
    logout 
  } = useAuthStore()

  useEffect(() => {
    const initializeAuth = async () => {
      // If not authenticated, skip initialization
      if (!isAuthenticated || !accessToken) {
        setIsInitializing(false)
        return
      }

      try {
        // Check if token is expired
        if (isTokenExpired()) {
          // Try to refresh the token
          if (refreshToken) {
            try {
              const response = await authService.refreshToken(refreshToken)
              setTokens(response.accessToken, response.refreshToken, response.expiresIn)
              updateUser(response.user)
            } catch (refreshError) {
              // Refresh failed, logout
              console.error('Token refresh failed during initialization:', refreshError)
              logout()
            }
          } else {
            // No refresh token, logout
            logout()
          }
        } else {
          // Token is valid, verify with server and get latest user data
          try {
            const user = await authService.getCurrentUser()
            updateUser(user)
          } catch (error) {
            // If getting user fails with 401, try to refresh
            if ((error as any)?.response?.status === 401 && refreshToken) {
              try {
                const response = await authService.refreshToken(refreshToken)
                setTokens(response.accessToken, response.refreshToken, response.expiresIn)
                updateUser(response.user)
              } catch (refreshError) {
                logout()
              }
            } else {
              // Other error, keep current state but log it
              console.error('Failed to get current user:', error)
            }
          }
        }
      } catch (error) {
        console.error('Auth initialization error:', error)
        // On any unexpected error, logout for safety
        logout()
      } finally {
        setIsInitializing(false)
      }
    }

    initializeAuth()
  }, []) // Only run once on mount

  // Show loading spinner while initializing
  if (isInitializing) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gray-50">
        <div className="text-center">
          <Spin size="large" />
          <p className="mt-4 text-gray-500">正在加载...</p>
        </div>
      </div>
    )
  }

  return <>{children}</>
}

export default AuthProvider
