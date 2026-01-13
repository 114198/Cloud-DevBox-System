import api from './api'

export interface LoginRequest {
  email: string
  password: string
}

export interface RegisterRequest {
  email: string
  username: string
  password: string
  displayName?: string
}

export interface AuthResponse {
  user: {
    id: string
    email: string
    username: string
    displayName: string
    role: 'developer' | 'admin' | 'org_admin'
    avatar?: string
  }
  accessToken: string
  refreshToken: string
  expiresIn: number
}

export interface OAuthProvider {
  name: string
  authUrl: string
}

// Auth API service
export const authService = {
  // Login with email and password
  async login(data: LoginRequest): Promise<AuthResponse> {
    const response = await api.post<AuthResponse>('/auth/login', data)
    return response.data
  },

  // Register new user
  async register(data: RegisterRequest): Promise<AuthResponse> {
    const response = await api.post<AuthResponse>('/auth/register', data)
    return response.data
  },

  // Refresh access token
  async refreshToken(refreshToken: string): Promise<AuthResponse> {
    const response = await api.post<AuthResponse>('/auth/refresh', { refreshToken })
    return response.data
  },

  // Logout
  async logout(): Promise<void> {
    await api.post('/auth/logout')
  },

  // Get OAuth providers
  async getOAuthProviders(): Promise<OAuthProvider[]> {
    const response = await api.get<OAuthProvider[]>('/auth/oauth/providers')
    return response.data
  },

  // Initiate OAuth login
  getOAuthUrl(provider: string): string {
    return `/api/v1/auth/oauth/${provider}`
  },

  // Handle OAuth callback
  async handleOAuthCallback(provider: string, code: string): Promise<AuthResponse> {
    const response = await api.post<AuthResponse>(`/auth/oauth/${provider}/callback`, { code })
    return response.data
  },

  // Verify email
  async verifyEmail(token: string): Promise<void> {
    await api.post('/auth/verify-email', { token })
  },

  // Resend verification email
  async resendVerificationEmail(email: string): Promise<void> {
    await api.post('/auth/resend-verification', { email })
  },

  // Request password reset
  async requestPasswordReset(email: string): Promise<void> {
    await api.post('/auth/forgot-password', { email })
  },

  // Reset password
  async resetPassword(token: string, newPassword: string): Promise<void> {
    await api.post('/auth/reset-password', { token, newPassword })
  },

  // Get current user
  async getCurrentUser(): Promise<AuthResponse['user']> {
    const response = await api.get<AuthResponse['user']>('/auth/me')
    return response.data
  },
}

export default authService
