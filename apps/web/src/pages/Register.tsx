import { useState, useMemo } from 'react'
import { Form, Input, Button, Card, Divider, message, Alert, Progress, Tooltip } from 'antd'
import { 
  GithubOutlined, 
  GitlabOutlined, 
  MailOutlined, 
  LockOutlined,
  UserOutlined,
  LoadingOutlined,
  CheckCircleFilled,
  CloseCircleFilled,
  InfoCircleOutlined
} from '@ant-design/icons'
import { Link, useNavigate } from 'react-router-dom'
import { useAuthStore } from '@/stores/auth'
import { authService, RegisterRequest } from '@/services/auth'

// Gitee icon component
const GiteeIcon = () => (
  <svg viewBox="0 0 1024 1024" width="1em" height="1em" fill="currentColor">
    <path d="M512 1024C229.222 1024 0 794.778 0 512S229.222 0 512 0s512 229.222 512 512-229.222 512-512 512z m259.149-568.883h-290.74a25.293 25.293 0 0 0-25.292 25.293l-0.026 63.206c0 13.952 11.315 25.293 25.267 25.293h177.024c13.978 0 25.293 11.315 25.293 25.267v12.646a75.853 75.853 0 0 1-75.853 75.853h-240.23a25.293 25.293 0 0 1-25.267-25.293V417.203a75.853 75.853 0 0 1 75.827-75.853h353.946a25.293 25.293 0 0 0 25.267-25.292l0.077-63.207a25.293 25.293 0 0 0-25.268-25.293H417.152a189.62 189.62 0 0 0-189.62 189.645V771.15c0 13.977 11.316 25.293 25.294 25.293h372.94a170.65 170.65 0 0 0 170.65-170.65V480.384a25.293 25.293 0 0 0-25.293-25.267z" />
  </svg>
)

interface RegisterFormValues {
  email: string
  username: string
  displayName: string
  password: string
  confirmPassword: string
}

interface PasswordStrength {
  score: number
  label: string
  color: string
  requirements: {
    length: boolean
    lowercase: boolean
    uppercase: boolean
    number: boolean
    special: boolean
  }
}

// Password strength calculator
function calculatePasswordStrength(password: string): PasswordStrength {
  const requirements = {
    length: password.length >= 8,
    lowercase: /[a-z]/.test(password),
    uppercase: /[A-Z]/.test(password),
    number: /[0-9]/.test(password),
    special: /[!@#$%^&*(),.?":{}|<>]/.test(password),
  }

  const score = Object.values(requirements).filter(Boolean).length

  let label: string
  let color: string

  if (score <= 1) {
    label = '弱'
    color = '#ff4d4f'
  } else if (score <= 2) {
    label = '较弱'
    color = '#faad14'
  } else if (score <= 3) {
    label = '中等'
    color = '#1890ff'
  } else if (score <= 4) {
    label = '强'
    color = '#52c41a'
  } else {
    label = '非常强'
    color = '#389e0d'
  }

  return { score, label, color, requirements }
}

export default function Register() {
  const navigate = useNavigate()
  const { login } = useAuthStore()
  const [form] = Form.useForm<RegisterFormValues>()
  const [loading, setLoading] = useState(false)
  const [oauthLoading, setOauthLoading] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [password, setPassword] = useState('')
  const [registrationSuccess, setRegistrationSuccess] = useState(false)
  const [registeredEmail, setRegisteredEmail] = useState('')

  // Calculate password strength
  const passwordStrength = useMemo(() => calculatePasswordStrength(password), [password])

  const handleRegister = async (values: RegisterFormValues) => {
    setLoading(true)
    setError(null)

    try {
      const registerData: RegisterRequest = {
        email: values.email,
        username: values.username,
        password: values.password,
        displayName: values.displayName || values.username,
      }

      const response = await authService.register(registerData)
      
      // Show success message and prompt for email verification
      setRegisteredEmail(values.email)
      setRegistrationSuccess(true)
      message.success('注册成功！请查收验证邮件')
      
      // Auto login after registration (if email verification is not required)
      // Uncomment below if auto-login is desired
      // login(response.user, response.accessToken, response.refreshToken, response.expiresIn)
      // navigate('/')
    } catch (err: any) {
      const errorMessage = err.response?.data?.message || err.message || '注册失败，请稍后重试'
      setError(errorMessage)
      
      // Handle specific error codes
      if (err.response?.status === 409) {
        const field = err.response?.data?.field
        if (field === 'email') {
          setError('该邮箱已被注册')
          form.setFields([{ name: 'email', errors: ['该邮箱已被注册'] }])
        } else if (field === 'username') {
          setError('该用户名已被使用')
          form.setFields([{ name: 'username', errors: ['该用户名已被使用'] }])
        }
      } else if (err.response?.status === 400) {
        setError('请检查输入信息是否正确')
      } else if (err.response?.status === 429) {
        setError('注册请求过于频繁，请稍后再试')
      }
    } finally {
      setLoading(false)
    }
  }

  const handleResendVerification = async () => {
    try {
      await authService.resendVerificationEmail(registeredEmail)
      message.success('验证邮件已重新发送')
    } catch (err: any) {
      message.error('发送失败，请稍后重试')
    }
  }

  const handleOAuthLogin = (provider: string) => {
    setOauthLoading(provider)
    setError(null)
    
    const oauthUrl = authService.getOAuthUrl(provider)
    window.location.href = oauthUrl
  }

  const oauthProviders = [
    { key: 'github', name: 'GitHub', icon: <GithubOutlined />, color: '#24292e' },
    { key: 'gitlab', name: 'GitLab', icon: <GitlabOutlined />, color: '#fc6d26' },
    { key: 'gitee', name: 'Gitee', icon: <GiteeIcon />, color: '#c71d23' },
  ]

  // Registration success view
  if (registrationSuccess) {
    return (
      <div className="min-h-screen flex items-center justify-center bg-gradient-to-br from-blue-50 to-indigo-100 p-4">
        <Card 
          className="w-full max-w-md shadow-xl border-0"
          styles={{ body: { padding: '40px' } }}
        >
          <div className="text-center">
            <div className="inline-flex items-center justify-center w-20 h-20 bg-green-100 rounded-full mb-6">
              <MailOutlined className="text-4xl text-green-600" />
            </div>
            <h2 className="text-2xl font-bold text-gray-800 mb-2">验证您的邮箱</h2>
            <p className="text-gray-500 mb-6">
              我们已向 <span className="font-medium text-gray-700">{registeredEmail}</span> 发送了一封验证邮件。
              请点击邮件中的链接完成注册。
            </p>
            
            <div className="bg-gray-50 rounded-lg p-4 mb-6">
              <p className="text-sm text-gray-600">
                没有收到邮件？请检查垃圾邮件文件夹，或者
              </p>
              <Button 
                type="link" 
                onClick={handleResendVerification}
                className="p-0 h-auto"
              >
                重新发送验证邮件
              </Button>
            </div>

            <Button 
              type="primary" 
              size="large" 
              block
              onClick={() => navigate('/login')}
            >
              返回登录
            </Button>
          </div>
        </Card>
      </div>
    )
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-gradient-to-br from-blue-50 to-indigo-100 p-4">
      <Card 
        className="w-full max-w-md shadow-xl border-0"
        styles={{ body: { padding: '40px' } }}
      >
        {/* Logo and Title */}
        <div className="text-center mb-6">
          <div className="inline-flex items-center justify-center w-16 h-16 bg-blue-600 rounded-xl mb-4">
            <span className="text-white text-2xl font-bold">CD</span>
          </div>
          <h1 className="text-2xl font-bold text-gray-800">创建账号</h1>
          <p className="text-gray-500 mt-2">开始您的云端开发之旅</p>
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

        {/* Register Form */}
        <Form
          form={form}
          layout="vertical"
          onFinish={handleRegister}
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
            name="username"
            label="用户名"
            rules={[
              { required: true, message: '请输入用户名' },
              { min: 3, message: '用户名至少3个字符' },
              { max: 20, message: '用户名最多20个字符' },
              { pattern: /^[a-zA-Z0-9_-]+$/, message: '用户名只能包含字母、数字、下划线和连字符' },
            ]}
          >
            <Input
              prefix={<UserOutlined className="text-gray-400" />}
              placeholder="your_username"
              size="large"
              autoComplete="username"
              disabled={loading}
            />
          </Form.Item>

          <Form.Item
            name="displayName"
            label={
              <span>
                显示名称
                <Tooltip title="可选，用于在界面上显示的名称">
                  <InfoCircleOutlined className="ml-1 text-gray-400" />
                </Tooltip>
              </span>
            }
          >
            <Input
              prefix={<UserOutlined className="text-gray-400" />}
              placeholder="您的昵称（可选）"
              size="large"
              disabled={loading}
            />
          </Form.Item>

          <Form.Item
            name="password"
            label="密码"
            rules={[
              { required: true, message: '请输入密码' },
              { min: 8, message: '密码至少8个字符' },
              {
                validator: (_, value) => {
                  if (!value) return Promise.resolve()
                  const strength = calculatePasswordStrength(value)
                  if (strength.score < 3) {
                    return Promise.reject(new Error('密码强度不足，请包含大小写字母、数字或特殊字符'))
                  }
                  return Promise.resolve()
                },
              },
            ]}
          >
            <Input.Password
              prefix={<LockOutlined className="text-gray-400" />}
              placeholder="请输入密码"
              size="large"
              autoComplete="new-password"
              disabled={loading}
              onChange={(e) => setPassword(e.target.value)}
            />
          </Form.Item>

          {/* Password Strength Indicator */}
          {password && (
            <div className="mb-4 -mt-2">
              <div className="flex items-center gap-2 mb-2">
                <Progress
                  percent={(passwordStrength.score / 5) * 100}
                  showInfo={false}
                  strokeColor={passwordStrength.color}
                  size="small"
                  className="flex-1"
                />
                <span 
                  className="text-sm font-medium"
                  style={{ color: passwordStrength.color }}
                >
                  {passwordStrength.label}
                </span>
              </div>
              <div className="grid grid-cols-2 gap-1 text-xs">
                <RequirementItem 
                  met={passwordStrength.requirements.length} 
                  text="至少8个字符" 
                />
                <RequirementItem 
                  met={passwordStrength.requirements.lowercase} 
                  text="包含小写字母" 
                />
                <RequirementItem 
                  met={passwordStrength.requirements.uppercase} 
                  text="包含大写字母" 
                />
                <RequirementItem 
                  met={passwordStrength.requirements.number} 
                  text="包含数字" 
                />
                <RequirementItem 
                  met={passwordStrength.requirements.special} 
                  text="包含特殊字符" 
                />
              </div>
            </div>
          )}

          <Form.Item
            name="confirmPassword"
            label="确认密码"
            dependencies={['password']}
            rules={[
              { required: true, message: '请确认密码' },
              ({ getFieldValue }) => ({
                validator(_, value) {
                  if (!value || getFieldValue('password') === value) {
                    return Promise.resolve()
                  }
                  return Promise.reject(new Error('两次输入的密码不一致'))
                },
              }),
            ]}
          >
            <Input.Password
              prefix={<LockOutlined className="text-gray-400" />}
              placeholder="请再次输入密码"
              size="large"
              autoComplete="new-password"
              disabled={loading}
            />
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
              {loading ? '注册中...' : '创建账号'}
            </Button>
          </Form.Item>

          <p className="text-xs text-gray-500 text-center mb-4">
            点击"创建账号"即表示您同意我们的
            <Link to="/terms" className="text-blue-600 hover:text-blue-700 mx-1">服务条款</Link>
            和
            <Link to="/privacy" className="text-blue-600 hover:text-blue-700 mx-1">隐私政策</Link>
          </p>
        </Form>

        {/* OAuth Divider */}
        <Divider className="!my-6">
          <span className="text-gray-400 text-sm">或使用以下方式注册</span>
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

        {/* Login Link */}
        <div className="text-center mt-8 pt-6 border-t border-gray-100">
          <span className="text-gray-500">已有账号？</span>
          <Link 
            to="/login" 
            className="text-blue-600 hover:text-blue-700 font-medium ml-1"
          >
            立即登录
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

// Password requirement indicator component
function RequirementItem({ met, text }: { met: boolean; text: string }) {
  return (
    <div className={`flex items-center gap-1 ${met ? 'text-green-600' : 'text-gray-400'}`}>
      {met ? (
        <CheckCircleFilled className="text-xs" />
      ) : (
        <CloseCircleFilled className="text-xs" />
      )}
      <span>{text}</span>
    </div>
  )
}
