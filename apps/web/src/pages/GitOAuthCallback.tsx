import { useEffect, useState } from 'react'
import { useNavigate, useParams, useSearchParams } from 'react-router-dom'
import { Spin, Result, Button } from 'antd'
import api from '@/services/api'
import { useGitStore } from '@/stores/git'

export default function GitOAuthCallback() {
  const navigate = useNavigate()
  const { provider } = useParams<{ provider: string }>()
  const [searchParams] = useSearchParams()
  const { addConnection } = useGitStore()
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    const handleCallback = async () => {
      const code = searchParams.get('code')
      const state = searchParams.get('state')
      const errorParam = searchParams.get('error')
      const errorDescription = searchParams.get('error_description')

      // Handle OAuth error
      if (errorParam) {
        setError(errorDescription || `Git OAuth 认证失败: ${errorParam}`)
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
        // Exchange code for connection
        const response = await api.get(`/git/callback/${provider}`, {
          params: { code, state },
        })

        // Add connection to store
        if (response.data.connection) {
          addConnection(response.data.connection)
        }

        // Redirect to Git repositories page
        navigate('/git', { replace: true })
      } catch (err: any) {
        const errorMessage =
          err.response?.data?.message || err.message || 'Git OAuth 认证失败'
        setError(errorMessage)

        // Handle specific errors
        if (err.response?.status === 409) {
          setError('该 Git 账号已绑定其他用户')
        } else if (err.response?.status === 403) {
          setError('该 Git 提供商已被禁用')
        } else if (err.response?.status === 400) {
          setError('授权码无效或已过期，请重新授权')
        }
      } finally {
        setLoading(false)
      }
    }

    handleCallback()
  }, [provider, searchParams, addConnection, navigate])

  if (loading) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gray-50">
        <div className="text-center">
          <Spin size="large" />
          <p className="mt-4 text-gray-600">正在连接 Git 平台...</p>
        </div>
      </div>
    )
  }

  if (error) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gray-50">
        <Result
          status="error"
          title="连接失败"
          subTitle={error}
          extra={[
            <Button
              type="primary"
              key="git"
              onClick={() => navigate('/git')}
            >
              返回 Git 仓库
            </Button>,
            <Button key="retry" onClick={() => window.location.reload()}>
              重试
            </Button>,
          ]}
        />
      </div>
    )
  }

  return null
}
