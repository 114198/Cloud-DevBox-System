import { Card, Form, Input, Button, Tabs, Switch, Select, message } from 'antd'
import { useAuthStore } from '@/stores/auth'

export default function Settings() {
  const { user, updateUser } = useAuthStore()
  const [profileForm] = Form.useForm()
  const [securityForm] = Form.useForm()

  const handleProfileSave = (values: Record<string, string>) => {
    updateUser(values)
    message.success('个人资料已更新')
  }

  const handlePasswordChange = () => {
    message.success('密码已更新')
    securityForm.resetFields()
  }

  const items = [
    {
      key: 'profile',
      label: '个人资料',
      children: (
        <Form
          form={profileForm}
          layout="vertical"
          initialValues={{
            displayName: user?.displayName,
            email: user?.email,
            username: user?.username,
          }}
          onFinish={handleProfileSave}
        >
          <Form.Item name="displayName" label="显示名称">
            <Input />
          </Form.Item>
          <Form.Item name="email" label="邮箱">
            <Input disabled />
          </Form.Item>
          <Form.Item name="username" label="用户名">
            <Input disabled />
          </Form.Item>
          <Form.Item>
            <Button type="primary" htmlType="submit">
              保存
            </Button>
          </Form.Item>
        </Form>
      ),
    },
    {
      key: 'security',
      label: '安全设置',
      children: (
        <Form form={securityForm} layout="vertical" onFinish={handlePasswordChange}>
          <Form.Item
            name="currentPassword"
            label="当前密码"
            rules={[{ required: true, message: '请输入当前密码' }]}
          >
            <Input.Password />
          </Form.Item>
          <Form.Item
            name="newPassword"
            label="新密码"
            rules={[{ required: true, message: '请输入新密码' }]}
          >
            <Input.Password />
          </Form.Item>
          <Form.Item
            name="confirmPassword"
            label="确认新密码"
            rules={[{ required: true, message: '请确认新密码' }]}
          >
            <Input.Password />
          </Form.Item>
          <Form.Item>
            <Button type="primary" htmlType="submit">
              更新密码
            </Button>
          </Form.Item>
        </Form>
      ),
    },
    {
      key: 'preferences',
      label: '偏好设置',
      children: (
        <Form layout="vertical">
          <Form.Item name="language" label="界面语言" initialValue="zh-CN">
            <Select>
              <Select.Option value="zh-CN">简体中文</Select.Option>
              <Select.Option value="zh-TW">繁体中文</Select.Option>
              <Select.Option value="en">English</Select.Option>
              <Select.Option value="ja">日本語</Select.Option>
            </Select>
          </Form.Item>
          <Form.Item name="theme" label="主题" initialValue="light">
            <Select>
              <Select.Option value="light">浅色</Select.Option>
              <Select.Option value="dark">深色</Select.Option>
              <Select.Option value="auto">跟随系统</Select.Option>
            </Select>
          </Form.Item>
          <Form.Item name="notifications" label="邮件通知" valuePropName="checked">
            <Switch defaultChecked />
          </Form.Item>
          <Form.Item>
            <Button type="primary">保存偏好</Button>
          </Form.Item>
        </Form>
      ),
    },
  ]

  return (
    <div>
      <h1 className="text-2xl font-semibold mb-6">设置</h1>
      <Card>
        <Tabs items={items} />
      </Card>
    </div>
  )
}
