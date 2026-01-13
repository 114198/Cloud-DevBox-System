import { useState, useEffect } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import {
  Card,
  Form,
  Input,
  Select,
  Button,
  Space,
  Tabs,
  Table,
  Modal,
  message,
  Spin,
  Alert,
  InputNumber,
  Popconfirm,
  Tag,
  Tooltip,
} from 'antd'
import {
  ArrowLeftOutlined,
  SaveOutlined,
  PlusOutlined,
  DeleteOutlined,
  EditOutlined,
  CloudServerOutlined,
  SettingOutlined,
  ApiOutlined,
  KeyOutlined,
  ExclamationCircleOutlined,
} from '@ant-design/icons'
import type { ColumnsType } from 'antd/es/table'
import { environmentService } from '@/services/environment'
import { useEnvironmentStore, Environment } from '@/stores/environment'

interface EnvironmentVariable {
  key: string
  value: string
  isSecret: boolean
}

interface PortMapping {
  name: string
  containerPort: number
  protocol: 'TCP' | 'UDP'
  isPublic: boolean
  externalPort?: number
}

// Mock data
const mockEnvironment: Environment = {
  id: '1a2b3c4d-5e6f-7890-abcd-ef1234567890',
  name: 'react-app',
  description: 'React 前端项目',
  templateId: 't1',
  templateName: 'React 18',
  status: 'running',
  resources: { cpu: '2 核', memory: '4 GB', storage: '20 GB' },
  createdAt: '2026-01-10T10:00:00Z',
}

const mockEnvVars: EnvironmentVariable[] = [
  { key: 'NODE_ENV', value: 'development', isSecret: false },
  { key: 'API_URL', value: 'https://api.example.com', isSecret: false },
  { key: 'DATABASE_URL', value: '********', isSecret: true },
  { key: 'JWT_SECRET', value: '********', isSecret: true },
]

const mockPortMappings: PortMapping[] = [
  { name: 'HTTP', containerPort: 3000, protocol: 'TCP', isPublic: true, externalPort: 443 },
  { name: 'Dev Server', containerPort: 5173, protocol: 'TCP', isPublic: true, externalPort: 5173 },
  { name: 'Debug', containerPort: 9229, protocol: 'TCP', isPublic: false },
]

const cpuOptions = [
  { label: '1 核', value: 1 },
  { label: '2 核', value: 2 },
  { label: '4 核', value: 4 },
  { label: '8 核', value: 8 },
  { label: '16 核', value: 16 },
]

const memoryOptions = [
  { label: '1 GB', value: 1 },
  { label: '2 GB', value: 2 },
  { label: '4 GB', value: 4 },
  { label: '8 GB', value: 8 },
  { label: '16 GB', value: 16 },
  { label: '32 GB', value: 32 },
]

const storageOptions = [
  { label: '10 GB', value: 10 },
  { label: '20 GB', value: 20 },
  { label: '50 GB', value: 50 },
  { label: '100 GB', value: 100 },
  { label: '200 GB', value: 200 },
]

export default function EnvironmentConfig() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const [environment, setEnvironment] = useState<Environment | null>(null)
  const [envVars, setEnvVars] = useState<EnvironmentVariable[]>(mockEnvVars)
  const [portMappings, setPortMappings] = useState<PortMapping[]>(mockPortMappings)
  const [isLoading, setIsLoading] = useState(true)
  const [isSaving, setIsSaving] = useState(false)
  const [isEnvVarModalOpen, setIsEnvVarModalOpen] = useState(false)
  const [isPortModalOpen, setIsPortModalOpen] = useState(false)
  const [editingEnvVar, setEditingEnvVar] = useState<EnvironmentVariable | null>(null)
  const [editingPort, setEditingPort] = useState<PortMapping | null>(null)
  const [resourceForm] = Form.useForm()
  const [envVarForm] = Form.useForm()
  const [portForm] = Form.useForm()
  const { updateEnvironment } = useEnvironmentStore()

  useEffect(() => {
    loadEnvironment()
  }, [id])

  const loadEnvironment = async () => {
    setIsLoading(true)
    try {
      if (id) {
        const env = await environmentService.get(id)
        setEnvironment(env)
        // Parse resource values
        const cpuValue = parseInt(env.resources.cpu)
        const memoryValue = parseInt(env.resources.memory)
        const storageValue = parseInt(env.resources.storage)
        resourceForm.setFieldsValue({ cpu: cpuValue, memory: memoryValue, storage: storageValue })
      }
    } catch {
      setEnvironment(mockEnvironment)
      resourceForm.setFieldsValue({ cpu: 2, memory: 4, storage: 20 })
    } finally {
      setIsLoading(false)
    }
  }

  const handleSaveResources = async () => {
    if (!environment) return
    setIsSaving(true)
    try {
      const values = await resourceForm.validateFields()
      await environmentService.update(environment.id, {
        resources: {
          cpu: `${values.cpu} 核`,
          memory: `${values.memory} GB`,
          storage: `${values.storage} GB`,
        },
      })
      setEnvironment({
        ...environment,
        resources: {
          cpu: `${values.cpu} 核`,
          memory: `${values.memory} GB`,
          storage: `${values.storage} GB`,
        },
      })
      updateEnvironment(environment.id, {
        resources: {
          cpu: `${values.cpu} 核`,
          memory: `${values.memory} GB`,
          storage: `${values.storage} GB`,
        },
      })
      message.success('资源配置已更新')
    } catch {
      message.error('保存失败')
    } finally {
      setIsSaving(false)
    }
  }

  // Environment Variables handlers
  const handleAddEnvVar = () => {
    setEditingEnvVar(null)
    envVarForm.resetFields()
    setIsEnvVarModalOpen(true)
  }

  const handleEditEnvVar = (record: EnvironmentVariable) => {
    setEditingEnvVar(record)
    envVarForm.setFieldsValue(record)
    setIsEnvVarModalOpen(true)
  }

  const handleDeleteEnvVar = (key: string) => {
    setEnvVars(envVars.filter((v: EnvironmentVariable) => v.key !== key))
    message.success('环境变量已删除')
  }

  const handleSaveEnvVar = async () => {
    try {
      const values = await envVarForm.validateFields() as EnvironmentVariable
      if (editingEnvVar) {
        setEnvVars(envVars.map((v: EnvironmentVariable) => v.key === editingEnvVar.key ? values : v))
        message.success('环境变量已更新')
      } else {
        if (envVars.some((v: EnvironmentVariable) => v.key === values.key)) {
          message.error('环境变量名已存在')
          return
        }
        setEnvVars([...envVars, values])
        message.success('环境变量已添加')
      }
      setIsEnvVarModalOpen(false)
    } catch {
      // Validation failed
    }
  }

  // Port Mapping handlers
  const handleAddPort = () => {
    setEditingPort(null)
    portForm.resetFields()
    portForm.setFieldsValue({ protocol: 'TCP', isPublic: false })
    setIsPortModalOpen(true)
  }

  const handleEditPort = (record: PortMapping) => {
    setEditingPort(record)
    portForm.setFieldsValue(record)
    setIsPortModalOpen(true)
  }

  const handleDeletePort = (containerPort: number) => {
    setPortMappings(portMappings.filter((p: PortMapping) => p.containerPort !== containerPort))
    message.success('端口映射已删除')
  }

  const handleSavePort = async () => {
    try {
      const values = await portForm.validateFields() as PortMapping
      if (editingPort) {
        setPortMappings(portMappings.map((p: PortMapping) => p.containerPort === editingPort.containerPort ? values : p))
        message.success('端口映射已更新')
      } else {
        if (portMappings.some((p: PortMapping) => p.containerPort === values.containerPort)) {
          message.error('容器端口已被使用')
          return
        }
        setPortMappings([...portMappings, values])
        message.success('端口映射已添加')
      }
      setIsPortModalOpen(false)
    } catch {
      // Validation failed
    }
  }

  const envVarColumns: ColumnsType<EnvironmentVariable> = [
    {
      title: '变量名',
      dataIndex: 'key',
      key: 'key',
      render: (text: string) => <code className="bg-gray-100 px-2 py-1 rounded">{text}</code>,
    },
    {
      title: '值',
      dataIndex: 'value',
      key: 'value',
      render: (text: string, record: EnvironmentVariable) => (
        record.isSecret ? (
          <Tooltip title="敏感信息已隐藏">
            <span className="text-gray-400">••••••••</span>
          </Tooltip>
        ) : (
          <span className="font-mono">{text}</span>
        )
      ),
    },
    {
      title: '类型',
      dataIndex: 'isSecret',
      key: 'isSecret',
      width: 100,
      render: (isSecret: boolean) => (
        isSecret ? <Tag color="red" icon={<KeyOutlined />}>敏感</Tag> : <Tag>普通</Tag>
      ),
    },
    {
      title: '操作',
      key: 'action',
      width: 120,
      render: (_: unknown, record: EnvironmentVariable) => (
        <Space>
          <Button size="small" icon={<EditOutlined />} onClick={() => handleEditEnvVar(record)} />
          <Popconfirm
            title="确定删除此环境变量？"
            onConfirm={() => handleDeleteEnvVar(record.key)}
            okText="删除"
            cancelText="取消"
          >
            <Button size="small" danger icon={<DeleteOutlined />} />
          </Popconfirm>
        </Space>
      ),
    },
  ]

  const portColumns: ColumnsType<PortMapping> = [
    {
      title: '名称',
      dataIndex: 'name',
      key: 'name',
    },
    {
      title: '容器端口',
      dataIndex: 'containerPort',
      key: 'containerPort',
      render: (port: number) => <code>{port}</code>,
    },
    {
      title: '协议',
      dataIndex: 'protocol',
      key: 'protocol',
      width: 80,
      render: (protocol: string) => <Tag>{protocol}</Tag>,
    },
    {
      title: '公开访问',
      dataIndex: 'isPublic',
      key: 'isPublic',
      width: 100,
      render: (isPublic: boolean, record: PortMapping) => (
        isPublic ? (
          <Tag color="green">公开 (:{record.externalPort})</Tag>
        ) : (
          <Tag>内部</Tag>
        )
      ),
    },
    {
      title: '操作',
      key: 'action',
      width: 120,
      render: (_: unknown, record: PortMapping) => (
        <Space>
          <Button size="small" icon={<EditOutlined />} onClick={() => handleEditPort(record)} />
          <Popconfirm
            title="确定删除此端口映射？"
            onConfirm={() => handleDeletePort(record.containerPort)}
            okText="删除"
            cancelText="取消"
          >
            <Button size="small" danger icon={<DeleteOutlined />} />
          </Popconfirm>
        </Space>
      ),
    },
  ]

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-64">
        <Spin size="large" tip="加载中..." />
      </div>
    )
  }

  if (!environment) {
    return (
      <Alert
        message="环境不存在"
        type="error"
        showIcon
        action={
          <Button onClick={() => navigate('/environments')}>返回列表</Button>
        }
      />
    )
  }

  return (
    <div>
      {/* Header */}
      <div className="flex justify-between items-center mb-6">
        <div className="flex items-center gap-4">
          <Button icon={<ArrowLeftOutlined />} onClick={() => navigate(`/environments/${id}`)}>
            返回
          </Button>
          <div>
            <h1 className="text-2xl font-semibold flex items-center gap-2">
              <SettingOutlined className="text-blue-500" />
              配置 - {environment.name}
            </h1>
            <p className="text-gray-500 mt-1">调整环境资源、环境变量和端口映射</p>
          </div>
        </div>
      </div>

      {environment.status === 'running' && (
        <Alert
          message="环境正在运行"
          description="部分配置更改需要重启环境才能生效"
          type="warning"
          showIcon
          icon={<ExclamationCircleOutlined />}
          className="mb-4"
        />
      )}

      <Tabs
        defaultActiveKey="resources"
        items={[
          {
            key: 'resources',
            label: (
              <span>
                <CloudServerOutlined /> 资源配置
              </span>
            ),
            children: (
              <Card>
                <Form form={resourceForm} layout="vertical" style={{ maxWidth: 600 }}>
                  <Form.Item name="cpu" label="CPU 核心数" rules={[{ required: true }]}>
                    <Select options={cpuOptions} />
                  </Form.Item>

                  <Form.Item name="memory" label="内存大小" rules={[{ required: true }]}>
                    <Select options={memoryOptions} />
                  </Form.Item>

                  <Form.Item name="storage" label="存储空间" rules={[{ required: true }]}>
                    <Select options={storageOptions} />
                  </Form.Item>

                  <Form.Item>
                    <Space>
                      <Button
                        type="primary"
                        icon={<SaveOutlined />}
                        onClick={handleSaveResources}
                        loading={isSaving}
                      >
                        保存配置
                      </Button>
                      <Button onClick={() => resourceForm.resetFields()}>
                        重置
                      </Button>
                    </Space>
                  </Form.Item>
                </Form>

                <Alert
                  message="资源调整说明"
                  description={
                    <ul className="list-disc pl-4 mt-2">
                      <li>资源调整将在 30 秒内生效</li>
                      <li>增加资源不会影响正在运行的服务</li>
                      <li>减少资源可能导致服务重启</li>
                      <li>存储空间只能增加，不能减少</li>
                    </ul>
                  }
                  type="info"
                  showIcon
                  className="mt-4"
                />
              </Card>
            ),
          },
          {
            key: 'envvars',
            label: (
              <span>
                <KeyOutlined /> 环境变量
              </span>
            ),
            children: (
              <Card
                title="环境变量"
                extra={
                  <Button type="primary" icon={<PlusOutlined />} onClick={handleAddEnvVar}>
                    添加变量
                  </Button>
                }
              >
                <Table
                  columns={envVarColumns}
                  dataSource={envVars}
                  rowKey="key"
                  pagination={false}
                />

                <Alert
                  message="环境变量说明"
                  description={
                    <ul className="list-disc pl-4 mt-2">
                      <li>环境变量更改需要重启环境才能生效</li>
                      <li>敏感信息（如密码、密钥）建议标记为"敏感"类型</li>
                      <li>变量名只能包含大写字母、数字和下划线</li>
                    </ul>
                  }
                  type="info"
                  showIcon
                  className="mt-4"
                />
              </Card>
            ),
          },
          {
            key: 'ports',
            label: (
              <span>
                <ApiOutlined /> 端口映射
              </span>
            ),
            children: (
              <Card
                title="端口映射"
                extra={
                  <Button type="primary" icon={<PlusOutlined />} onClick={handleAddPort}>
                    添加端口
                  </Button>
                }
              >
                <Table
                  columns={portColumns}
                  dataSource={portMappings}
                  rowKey="containerPort"
                  pagination={false}
                />

                <Alert
                  message="端口映射说明"
                  description={
                    <ul className="list-disc pl-4 mt-2">
                      <li>公开端口可通过外部域名访问</li>
                      <li>内部端口仅在环境内部可访问</li>
                      <li>常用端口：HTTP(80/443)、SSH(22)、数据库(3306/5432)</li>
                    </ul>
                  }
                  type="info"
                  showIcon
                  className="mt-4"
                />
              </Card>
            ),
          },
        ]}
      />

      {/* Environment Variable Modal */}
      <Modal
        title={editingEnvVar ? '编辑环境变量' : '添加环境变量'}
        open={isEnvVarModalOpen}
        onOk={handleSaveEnvVar}
        onCancel={() => setIsEnvVarModalOpen(false)}
        okText="保存"
        cancelText="取消"
      >
        <Form form={envVarForm} layout="vertical">
          <Form.Item
            name="key"
            label="变量名"
            rules={[
              { required: true, message: '请输入变量名' },
              { pattern: /^[A-Z_][A-Z0-9_]*$/, message: '只能包含大写字母、数字和下划线' },
            ]}
          >
            <Input placeholder="例如: DATABASE_URL" disabled={!!editingEnvVar} />
          </Form.Item>
          <Form.Item
            name="value"
            label="值"
            rules={[{ required: true, message: '请输入值' }]}
          >
            <Input.TextArea placeholder="变量值" rows={3} />
          </Form.Item>
          <Form.Item name="isSecret" label="敏感信息" valuePropName="checked" initialValue={false}>
            <Select
              options={[
                { label: '否 - 普通变量', value: false },
                { label: '是 - 敏感信息（如密码、密钥）', value: true },
              ]}
            />
          </Form.Item>
        </Form>
      </Modal>

      {/* Port Mapping Modal */}
      <Modal
        title={editingPort ? '编辑端口映射' : '添加端口映射'}
        open={isPortModalOpen}
        onOk={handleSavePort}
        onCancel={() => setIsPortModalOpen(false)}
        okText="保存"
        cancelText="取消"
      >
        <Form form={portForm} layout="vertical">
          <Form.Item
            name="name"
            label="名称"
            rules={[{ required: true, message: '请输入名称' }]}
          >
            <Input placeholder="例如: HTTP Server" />
          </Form.Item>
          <Form.Item
            name="containerPort"
            label="容器端口"
            rules={[
              { required: true, message: '请输入端口号' },
              { type: 'number', min: 1, max: 65535, message: '端口范围 1-65535' },
            ]}
          >
            <InputNumber
              placeholder="例如: 3000"
              style={{ width: '100%' }}
              disabled={!!editingPort}
            />
          </Form.Item>
          <Form.Item name="protocol" label="协议" rules={[{ required: true }]}>
            <Select
              options={[
                { label: 'TCP', value: 'TCP' },
                { label: 'UDP', value: 'UDP' },
              ]}
            />
          </Form.Item>
          <Form.Item name="isPublic" label="公开访问" rules={[{ required: true }]}>
            <Select
              options={[
                { label: '否 - 仅内部访问', value: false },
                { label: '是 - 允许外部访问', value: true },
              ]}
            />
          </Form.Item>
          <Form.Item
            noStyle
            shouldUpdate={(prevValues: Record<string, unknown>, currentValues: Record<string, unknown>) => prevValues.isPublic !== currentValues.isPublic}
          >
            {({ getFieldValue }: { getFieldValue: (name: string) => unknown }) =>
              getFieldValue('isPublic') ? (
                <Form.Item
                  name="externalPort"
                  label="外部端口"
                  rules={[
                    { required: true, message: '请输入外部端口' },
                    { type: 'number', min: 1, max: 65535, message: '端口范围 1-65535' },
                  ]}
                >
                  <InputNumber placeholder="例如: 443" style={{ width: '100%' }} />
                </Form.Item>
              ) : null
            }
          </Form.Item>
        </Form>
      </Modal>
    </div>
  )
}
