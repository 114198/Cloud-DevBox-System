import { Modal, Form, Input, Select, InputNumber, Button, Tabs, Space, Tag, Divider, message } from 'antd'
import type { FormListFieldData, FormListOperation } from 'antd/es/form/FormList'
import { PlusOutlined, MinusCircleOutlined, InfoCircleOutlined } from '@ant-design/icons'
import { useState, useEffect } from 'react'
import { Template } from '@/services/template'
import { environmentService } from '@/services/environment'
import { useNavigate } from 'react-router-dom'

interface TemplateConfigModalProps {
  visible: boolean
  template: Template | null
  onClose: () => void
}

interface EnvironmentConfig {
  name: string
  description?: string
  resources: {
    cpu: string
    memory: string
    storage: string
  }
  environment: Array<{ key: string; value: string }>
  ports: Array<{ containerPort: number; protocol: string; name?: string }>
}

// CPU options
const cpuOptions = [
  { value: '0.5', label: '0.5 核' },
  { value: '1', label: '1 核' },
  { value: '2', label: '2 核' },
  { value: '4', label: '4 核' },
  { value: '8', label: '8 核' },
]

// Memory options
const memoryOptions = [
  { value: '512Mi', label: '512 MB' },
  { value: '1Gi', label: '1 GB' },
  { value: '2Gi', label: '2 GB' },
  { value: '4Gi', label: '4 GB' },
  { value: '8Gi', label: '8 GB' },
  { value: '16Gi', label: '16 GB' },
]

// Storage options
const storageOptions = [
  { value: '5Gi', label: '5 GB' },
  { value: '10Gi', label: '10 GB' },
  { value: '20Gi', label: '20 GB' },
  { value: '50Gi', label: '50 GB' },
  { value: '100Gi', label: '100 GB' },
]

// Protocol options
const protocolOptions = [
  { value: 'TCP', label: 'TCP' },
  { value: 'UDP', label: 'UDP' },
]

export default function TemplateConfigModal({ visible, template, onClose }: TemplateConfigModalProps) {
  const [form] = Form.useForm<EnvironmentConfig>()
  const [isSubmitting, setIsSubmitting] = useState(false)
  const [activeTab, setActiveTab] = useState('basic')
  const navigate = useNavigate()

  // Reset form when template changes
  useEffect(() => {
    if (template && visible) {
      const defaultConfig = template.default_config || {}
      const defaultPorts = defaultConfig.ports || []
      const defaultEnv = defaultConfig.environment || {}

      form.setFieldsValue({
        name: '',
        description: '',
        resources: {
          cpu: defaultConfig.cpu || '1',
          memory: defaultConfig.memory || '2Gi',
          storage: defaultConfig.storage || '10Gi',
        },
        environment: Object.entries(defaultEnv).map(([key, value]) => ({
          key,
          value: String(value),
        })),
        ports: defaultPorts.map((p: { containerPort: number; protocol?: string }) => ({
          containerPort: p.containerPort,
          protocol: p.protocol || 'TCP',
          name: '',
        })),
      })
      setActiveTab('basic')
    }
  }, [template, visible, form])

  // Handle form submission
  const handleSubmit = async () => {
    try {
      const values = await form.validateFields()
      setIsSubmitting(true)

      if (!template?.id) {
        throw new Error('缺少模板信息，无法创建环境')
      }

      const environmentVars =
        values.environment
          ?.filter((env: { key: string; value: string }) => env.key && env.value)
          .map((env: { key: string; value: string }) => ({
            name: env.key,
            value: env.value,
          })) || []

      const ports = values.ports?.filter((p) => p.containerPort) || []

      await environmentService.create({
        name: values.name,
        description: values.description,
        templateId: template.id,
        resources: {
          cpu: values.resources.cpu,
          memory: values.resources.memory,
          storage: values.resources.storage,
        },
        environment: environmentVars,
        ports,
      })

      message.success('环境创建成功')
      onClose()
      navigate('/environments')
    } catch (error) {
      if (error instanceof Error) {
        message.error(error.message)
      }
    } finally {
      setIsSubmitting(false)
    }
  }

  // Handle modal close
  const handleClose = () => {
    form.resetFields()
    onClose()
  }

  if (!template) return null

  const tabItems = [
    {
      key: 'basic',
      label: '基本信息',
      children: (
        <div className="space-y-4">
          <Form.Item
            name="name"
            label="环境名称"
            rules={[
              { required: true, message: '请输入环境名称' },
              { min: 2, max: 50, message: '名称长度应在 2-50 个字符之间' },
              { pattern: /^[a-zA-Z0-9-_]+$/, message: '名称只能包含字母、数字、横线和下划线' },
            ]}
          >
            <Input placeholder="例如: my-react-app" />
          </Form.Item>

          <Form.Item
            name="description"
            label="环境描述"
            rules={[{ max: 200, message: '描述不能超过 200 个字符' }]}
          >
            <Input.TextArea
              placeholder="可选，描述这个环境的用途"
              rows={3}
              showCount
              maxLength={200}
            />
          </Form.Item>

          {/* Template info display */}
          <div className="bg-gray-50 p-4 rounded-lg">
            <h4 className="font-medium mb-2">模板信息</h4>
            <div className="grid grid-cols-2 gap-2 text-sm">
              <div>
                <span className="text-gray-500">模板名称:</span>
                <span className="ml-2">{template.display_name}</span>
              </div>
              <div>
                <span className="text-gray-500">版本:</span>
                <span className="ml-2">v{template.version}</span>
              </div>
              <div>
                <span className="text-gray-500">运行时:</span>
                <span className="ml-2">
                  {template.runtime?.language} {template.runtime?.version}
                </span>
              </div>
              <div>
                <span className="text-gray-500">分类:</span>
                <Tag color="blue" className="ml-2">{template.category}</Tag>
              </div>
            </div>
          </div>
        </div>
      ),
    },
    {
      key: 'resources',
      label: '资源配置',
      children: (
        <div className="space-y-4">
          <div className="bg-blue-50 p-3 rounded-lg mb-4 flex items-start gap-2">
            <InfoCircleOutlined className="text-blue-500 mt-0.5" />
            <span className="text-sm text-blue-700">
              根据您的项目需求选择合适的资源配置。资源配置会影响环境的性能和费用。
            </span>
          </div>

          <Form.Item
            name={['resources', 'cpu']}
            label="CPU"
            rules={[{ required: true, message: '请选择 CPU 配置' }]}
          >
            <Select options={cpuOptions} placeholder="选择 CPU 核数" />
          </Form.Item>

          <Form.Item
            name={['resources', 'memory']}
            label="内存"
            rules={[{ required: true, message: '请选择内存配置' }]}
          >
            <Select options={memoryOptions} placeholder="选择内存大小" />
          </Form.Item>

          <Form.Item
            name={['resources', 'storage']}
            label="存储"
            rules={[{ required: true, message: '请选择存储配置' }]}
          >
            <Select options={storageOptions} placeholder="选择存储大小" />
          </Form.Item>

          {/* Cost estimation */}
          <div className="bg-gray-50 p-4 rounded-lg">
            <h4 className="font-medium mb-2">费用估算</h4>
            <p className="text-sm text-gray-500">
              基于当前配置，预计费用约为 <span className="text-green-600 font-medium">¥0.15/小时</span>
            </p>
            <p className="text-xs text-gray-400 mt-1">
              * 实际费用以账单为准，包含 CPU、内存、存储等资源使用
            </p>
          </div>
        </div>
      ),
    },
    {
      key: 'environment',
      label: '环境变量',
      children: (
        <div>
          <div className="bg-yellow-50 p-3 rounded-lg mb-4 flex items-start gap-2">
            <InfoCircleOutlined className="text-yellow-600 mt-0.5" />
            <span className="text-sm text-yellow-700">
              环境变量用于配置应用程序的运行参数。敏感信息（如密码、密钥）请谨慎处理。
            </span>
          </div>

          <Form.List name="environment">
            {(fields: FormListFieldData[], { add, remove }: FormListOperation) => (
              <>
                {fields.map(({ key, name, ...restField }) => (
                  <div key={key} className="flex gap-2 mb-2">
                    <Form.Item
                      {...restField}
                      name={[name, 'key']}
                      rules={[
                        { required: true, message: '请输入变量名' },
                        { pattern: /^[A-Z_][A-Z0-9_]*$/, message: '变量名应为大写字母和下划线' },
                      ]}
                      className="flex-1 mb-0"
                    >
                      <Input placeholder="变量名 (如 NODE_ENV)" />
                    </Form.Item>
                    <Form.Item
                      {...restField}
                      name={[name, 'value']}
                      rules={[{ required: true, message: '请输入变量值' }]}
                      className="flex-1 mb-0"
                    >
                      <Input placeholder="变量值" />
                    </Form.Item>
                    <Button
                      type="text"
                      danger
                      icon={<MinusCircleOutlined />}
                      onClick={() => remove(name)}
                    />
                  </div>
                ))}
                <Button
                  type="dashed"
                  onClick={() => add({ key: '', value: '' })}
                  block
                  icon={<PlusOutlined />}
                >
                  添加环境变量
                </Button>
              </>
            )}
          </Form.List>
        </div>
      ),
    },
    {
      key: 'ports',
      label: '端口映射',
      children: (
        <div>
          <div className="bg-green-50 p-3 rounded-lg mb-4 flex items-start gap-2">
            <InfoCircleOutlined className="text-green-600 mt-0.5" />
            <span className="text-sm text-green-700">
              配置需要对外暴露的端口。系统会自动为每个端口分配外部访问地址。
            </span>
          </div>

          <Form.List name="ports">
            {(fields: FormListFieldData[], { add, remove }: FormListOperation) => (
              <>
                {fields.map(({ key, name, ...restField }) => (
                  <div key={key} className="flex gap-2 mb-2 items-start">
                    <Form.Item
                      {...restField}
                      name={[name, 'containerPort']}
                      rules={[
                        { required: true, message: '请输入端口号' },
                        { type: 'number', min: 1, max: 65535, message: '端口范围 1-65535' },
                      ]}
                      className="w-32 mb-0"
                    >
                      <InputNumber placeholder="端口号" min={1} max={65535} className="w-full" />
                    </Form.Item>
                    <Form.Item
                      {...restField}
                      name={[name, 'protocol']}
                      className="w-24 mb-0"
                    >
                      <Select options={protocolOptions} placeholder="协议" />
                    </Form.Item>
                    <Form.Item
                      {...restField}
                      name={[name, 'name']}
                      className="flex-1 mb-0"
                    >
                      <Input placeholder="端口名称 (可选，如 http, api)" />
                    </Form.Item>
                    <Button
                      type="text"
                      danger
                      icon={<MinusCircleOutlined />}
                      onClick={() => remove(name)}
                    />
                  </div>
                ))}
                <Button
                  type="dashed"
                  onClick={() => add({ containerPort: undefined, protocol: 'TCP', name: '' })}
                  block
                  icon={<PlusOutlined />}
                >
                  添加端口映射
                </Button>
              </>
            )}
          </Form.List>

          {/* Common ports suggestion */}
          <Divider>常用端口</Divider>
          <div className="flex flex-wrap gap-2">
            {[
              { port: 3000, name: 'React/Next.js' },
              { port: 8080, name: 'Spring/Tomcat' },
              { port: 5000, name: 'Flask' },
              { port: 8000, name: 'Django/FastAPI' },
              { port: 4200, name: 'Angular' },
              { port: 5173, name: 'Vite' },
            ].map((item) => (
              <Tag
                key={item.port}
                className="cursor-pointer hover:bg-blue-50"
                onClick={() => {
                  const currentPorts = form.getFieldValue('ports') || []
                  if (!currentPorts.some((p: { containerPort: number }) => p.containerPort === item.port)) {
                    form.setFieldsValue({
                      ports: [...currentPorts, { containerPort: item.port, protocol: 'TCP', name: item.name }],
                    })
                  }
                }}
              >
                {item.port} ({item.name})
              </Tag>
            ))}
          </div>
        </div>
      ),
    },
  ]

  return (
    <Modal
      title={
        <div className="flex items-center gap-2">
          <span>配置开发环境</span>
          <Tag color="blue">{template.display_name}</Tag>
        </div>
      }
      open={visible}
      onCancel={handleClose}
      width={700}
      footer={
        <Space>
          <Button onClick={handleClose}>取消</Button>
          <Button type="primary" onClick={handleSubmit} loading={isSubmitting}>
            创建环境
          </Button>
        </Space>
      }
      destroyOnClose
    >
      <Form
        form={form}
        layout="vertical"
        className="mt-4"
        initialValues={{
          resources: {
            cpu: '1',
            memory: '2Gi',
            storage: '10Gi',
          },
          environment: [],
          ports: [],
        }}
      >
        <Tabs
          activeKey={activeTab}
          onChange={setActiveTab}
          items={tabItems}
        />
      </Form>
    </Modal>
  )
}
