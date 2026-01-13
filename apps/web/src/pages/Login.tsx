import { useState } from 'react'
import { Form, Input, Button, Card, Divider, message, Alert, Checkbox } from 'antd'
import { 
  GithubOutlined, 
  GitlabOutlined, 
  MailOutlined, 
  LockOutlined,
  LoadingOutlined 
} from '@ant-design/icons'
import { Link, useNavigate, useLocation } from 'react-router-dom'
import { useAuthStore } from '@/stores/auth'
import { authService, LoginRequest } from '@/services/auth'

// Gitee icon component (not available in antd icons)
const GiteeIcon = () => (
  <svg viewBox="0 0 1024 1024" width="1em" height="1em" fill="currentColor">
    <path d="M512 1024C229.222 1024 0 794.778 0 512S229.222 0 512 0s512 229.222 512 512-229.222 512-512 512z m259.149-568.883h-290.74a25.293 25.293 0 0 0-25.292 25.293l-0.026 63.206c0 13.952 11.315 25.293 25.267 25.293h177.024c13.978 0 25.293 11.315 25.293 25.267v12.646a75.853 75.853 0 0 1-75.853 75.853h-240.23a25.293 25.293 0 0 1-25.267-25.293V417.203a75.853 75.853 0 0 1 75.827-75.853h353.946a25.293 25.293 0 0 0 25.267-25.292l0.077-63.207a25.293 25.293 0 0 0-25.268-25.293H417.152a189.62 189.62 0 0 0-189.62 189.645V771.15c0 13.977 11.316 25.293 25.294 25.293h372.94a170.65 170.65 0 0 0 170.65-170.65V480.384a25.293 25.293 0 0 0-25.293-25.267z" />
  </svg>
)

interface LoginFormValues {
  email: string
  password: string
  remember: boolean
}

export default function Login() {
  const navigate = useNavigate()
  const location = useLocation()
  const { login } = useAuthStore()
  const [form] = Form.useForm<LoginFormValues>()
  const [loading, setLoading] = useState(false)
  const [oauthLoading, setOauthLoading] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)

  // Get redirect path from location state
  const from = (location.state as { from?: string })?.from || '/'

  const handleLogin = async (values: LoginFormValues) => {
    setLoading(true)
    setError(null)

    try {
      const loginData: LoginRequest = {
        email: values.email,
        password: values.password,
      }

      const response = await authService.login(loginData)
      
      login(
        response.user,
        response.accessToken,
        response.refreshToken,
        response.expiresIn
      )
      
      message.success('登录成功')
      navigate(from, { replace: true })
    } catch (err: any) {
      const errorMessage = err.response?.data?.message || err.message || '登录失败，请检查邮箱和密码'
      setError(errorMessage)
      
      // Handle specific error codes
      if (err.response?.status === 401) {
        setError('邮箱或密码错误')
      } else if (err.response?.status === 403) {
        setError('账号已被禁用，请联系管理员')
      } else if (err.response?.status === 429) {
        setError('登录尝试次数过多，请稍后再试')
      }
    } finally {
      setLoading(false)
    }
  }

  const handleOAuthLogin = (provider: string) => {
    setOauthLoading(provider)
    setError(null)
    
    // Redirect to OAuth provider
    const oauthUrl = authService.getOAuthUrl(provider)
    window.location.href = oauthUrl
  }

  const oauthProviders = [
    { 
      key: 'github', 
      name: 'GitHub', 
      icon: <GithubOutlined />,
      color: '#24292e'
    },
    { 
      key: 'gitlab', 
      name: 'GitLab', 
      icon: <GitlabOutlined />,
      color: '#fc6d26'
    },
    { 
      key: 'gitee', 
      name: 'Gitee', 
      icon: <GiteeIcon />,
      color: '#c71d23'
    },
  ]

  return (
    <div className="min-h-screen flex items-center justify-center bg-gradient-to-br from-blue-50 to-indigo-100 p-4">
      <Card 
        className="w-full max-w-md shadow-xl border-0"
        styles={{ body: { padding: '40px' } }}
      >
        {/* Logo and Title */}
        <div className="text-center mb-8">
          <div className="inline-flex items-center justify-center w-16 h-16 bg-blue-600 rounded-xl mb-4">
            <span className="text-white text-2xl font-bold">CD</span>
          </div>
          <h1 className="text-2xl font-bold text-gray-800">Cloud DevBox</h1>
          <p className="text-gray-500 mt-2">云端开发环境管理平台</p>
        </div>

        {/* Error Alert */}
        {error && (
          <Alert
            message={error}
            type="error"
            showIcon
            closable
            onClose={() => setError(null)}
            className="mb-6"
          />
        )}

        {/* Login Form */}
        <Form
          form={form}
          layout="vertical"
          onFinish={handleLogin}
          initialValues={{ remember: true }}
          requiredMark={false}
        >
          <Form.Item
            name="email"
            label="邮箱地址"
            rules={[
              { required: true, message: '请输入邮箱地址' },
              { type: 'email', message: '请输入有效的邮箱地址' },
            ]}
          >
            <Input
              prefix={<MailOutlined className="text-gray-400" />}
              placeholder="your@email.com"
              size="large"
              autoComplete="email"
              disabled={loading}
            />
          </Form.Item>

          <Form.Item
            name="password"
            label="密码"
            rules={[
              { required: true, message: '请输入密码' },
              { min: 6, message: '密码至少6个字符' },
            ]}
          >
            <Input.Password
              prefix={<LockOutlined className="text-gray-400" />}
              placeholder="请输入密码"
              size="large"
              autoComplete="current-password"
              disabled={loading}
            />
          </Form.Item>

          <Form.Item>
            <div className="flex justify-between items-center">
              <Form.Item name="remember" valuePropName="checked" noStyle>
                <Checkbox disabled={loading}>记住我</Checkbox>
              </Form.Item>
              <Link 
                to="/forgot-password" 
                className="text-blue-600 hover:text-blue-700"
              >
                忘记密码？
              </Link>
            </div>
          </Form.Item>

          <Form.Item className="mb-4">
            <Button
              type="primary"
              htmlType="submit"
              size="large"
              block
              loading={loading}
              className="h-12 text-base font-medium"
            >
              {loading ? '登录中...' : '登录'}
            </Button>
          </Form.Item>
        </Form>

        {/* OAuth Divider */}
        <Divider className="!my-6">
          <span className="text-gray-400 text-sm">或使用以下方式登录</span>
        </Divider>

        {/* OAuth Buttons */}
        <div className="flex gap-3">
          {oauthProviders.map((provider) => (
            <Button
              key={provider.key}
              icon={oauthLoading === provider.key ? <LoadingOutlined /> : provider.icon}
              size="large"
              block
              onClick={() => handleOAuthLogin(provider.key)}
              disabled={loading || oauthLoading !== null}
              className="flex items-center justify-center"
              style={{ 
                borderColor: provider.color,
                color: oauthLoading === provider.key ? undefined : provider.color
              }}
            >
              {provider.name}
            </Button>
          ))}
        </div>

        {/* Register Link */}
        <div className="text-center mt-8 pt-6 border-t border-gray-100">
          <span className="text-gray-500">还没有账号？</span>
          <Link 
            to="/register" 
            className="text-blue-600 hover:text-blue-700 font-medium ml-1"
          >
            立即注册
          </Link>
        </div>
      </Card>

      {/* Footer */}
      <div className="fixed bottom-4 text-center text-gray-400 text-sm">
        © 2026 Cloud DevBox. All rights reserved.
      </div>
    </div>
  )
}
