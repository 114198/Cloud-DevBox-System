import { create } from 'zustand'
import { persist, createJSONStorage } from 'zustand/middleware'

export interface User {
  id: string
  email: string
  username: string
  displayName: string
  role: 'developer' | 'admin' | 'org_admin'
  avatar?: string
}

interface AuthState {
  user: User | null
  accessToken: string | null
  refreshToken: string | null
  expiresAt: number | null
  isAuthenticated: boolean
  isLoading: boolean
  
  // Actions
  login: (user: User, accessToken: string, refreshToken: string, expiresIn: number) => void
  logout: () => void
  updateUser: (user: Partial<User>) => void
  setTokens: (accessToken: string, refreshToken: string, expiresIn: number) => void
  setLoading: (loading: boolean) => void
  isTokenExpired: () => boolean
  isTokenExpiringSoon: () => boolean
}

// Token expiration buffer (5 minutes before actual expiration)
const TOKEN_EXPIRY_BUFFER = 5 * 60 * 1000

export const useAuthStore = create<AuthState>()(
  persist(
    (set, get) => ({
      user: null,
      accessToken: null,
      refreshToken: null,
      expiresAt: null,
      isAuthenticated: false,
      isLoading: false,

      login: (user, accessToken, refreshToken, expiresIn) => {
        const expiresAt = Date.now() + expiresIn * 1000
        set({
          user,
          accessToken,
          refreshToken,
          expiresAt,
          isAuthenticated: true,
          isLoading: false,
        })
      },

      logout: () => {
        set({
          user: null,
          accessToken: null,
          refreshToken: null,
          expiresAt: null,
          isAuthenticated: false,
          isLoading: false,
        })
      },

      updateUser: (userData) =>
        set((state) => ({
          user: state.user ? { ...state.user, ...userData } : null,
        })),

      setTokens: (accessToken, refreshToken, expiresIn) => {
        const expiresAt = Date.now() + expiresIn * 1000
        set({
          accessToken,
          refreshToken,
          expiresAt,
        })
      },

      setLoading: (loading) => set({ isLoading: loading }),

      isTokenExpired: () => {
        const { expiresAt } = get()
        if (!expiresAt) return true
        return Date.now() >= expiresAt
      },

      isTokenExpiringSoon: () => {
        const { expiresAt } = get()
        if (!expiresAt) return true
        return Date.now() >= expiresAt - TOKEN_EXPIRY_BUFFER
      },
    }),
    {
      name: 'devbox-auth',
      storage: createJSONStorage(() => localStorage),
      partialize: (state) => ({
        user: state.user,
        accessToken: state.accessToken,
        refreshToken: state.refreshToken,
        expiresAt: state.expiresAt,
        isAuthenticated: state.isAuthenticated,
      }),
    }
  )
)

// Selector hooks for common use cases
export const useUser = () => useAuthStore((state) => state.user)
export const useIsAuthenticated = () => useAuthStore((state) => state.isAuthenticated)
export const useAccessToken = () => useAuthStore((state) => state.accessToken)
