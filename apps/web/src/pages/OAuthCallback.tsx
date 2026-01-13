import { useEffect, useState } from 'react'
import { useNavigate, useParams, useSearchParams } from 'react-router-dom'
import { Spin, Result, Button } from 'antd'
import { useAuthStore } from '@/stores/auth'
import { authService } from '@/services/auth'

export default function OAuthCallback() {
  const navigate = useNavigate()
  const { provider } = useParams<{ provider: string }>()
  const [searchParams] = useSearchParams()
  const { login } = useAuthStore()
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    const handleCallback = async () => {
      const code = searchParams.get('code')
      const errorParam = searchParams.get('error')
      const errorDescription = searchParams.get('error_description')

      // Handle OAuth error
      if (errorParam) {
        setError(errorDescription || `OAuth 认证失败: ${errorParam}`)
        setLoading(false)
        return
      }

      // Validate required parameters
      if (!code || !provider) {
        setError('无效的回调参数')
        setLoading(false)
        return
      }

      try {
        // Exchange code for tokens
        const response = await authService.handleOAuthCallback(provider, code)
        
        // Login user
        login(
          response.user,
          response.accessToken,
          response.refreshToken,
          response.expiresIn
        )

        // Redirect to dashboard
        navigate('/', { replace: true })
      } catch (err: any) {
        const errorMessage = err.response?.data?.message || err.message || 'OAuth 认证失败'
        setError(errorMessage)
        
        // Handle specific errors
        if (err.response?.status === 409) {
          setError('该账号已绑定其他用户，请使用其他方式登录')
        } else if (err.response?.status === 403) {
          setError('该 OAuth 提供商已被禁用')
        }
      } finally {
        setLoading(false)
      }
    }

    handleCallback()
  }, [provider, searchParams, login, navigate])

  if (loading) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gray-50">
        <div className="text-center">
          <Spin size="large" />
          <p className="mt-4 text-gray-600">正在完成认证...</p>
        </div>
      </div>
    )
  }

  if (error) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gray-50">
        <Result
          status="error"
          title="认证失败"
          subTitle={error}
          extra={[
            <Button 
              type="primary" 
              key="login"
              onClick={() => navigate('/login')}
            >
              返回登录
            </Button>,
            <Button 
              key="retry"
              onClick={() => window.location.reload()}
            >
              重试
            </Button>,
          ]}
        />
      </div>
    )
  }

  return null
}
