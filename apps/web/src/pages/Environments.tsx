import { useState, useEffect } from 'react'
import {
  Table,
  Button,
  Tag,
  Space,
  Modal,
  Form,
  Input,
  Select,
  Card,
  Drawer,
  Tabs,
  message,
  Dropdown,
  Tooltip,
  Row,
  Col,
  Statistic,
  Badge,
} from 'antd'
import {
  PlusOutlined,
  PlayCircleOutlined,
  PauseCircleOutlined,
  DeleteOutlined,
  SettingOutlined,
  LinkOutlined,
  CodeOutlined,
  DesktopOutlined,
  SearchOutlined,
  FilterOutlined,
  ReloadOutlined,
  MoreOutlined,
  CloudServerOutlined,
  ExclamationCircleOutlined,
} from '@ant-design/icons'
import type { ColumnsType, TableRowSelection } from 'antd/es/table'
import { useSearchParams } from 'react-router-dom'
import { useEnvironmentStore, Environment, EnvironmentStatus } from '@/stores/environment'
import { environmentService } from '@/services/environment'
import ConnectionInfoPanel from '@/components/ConnectionInfoPanel'
import IDEConnectionGuide from '@/components/IDEConnectionGuide'
import WebTerminal from '@/components/WebTerminal'

// Mock data for development
const mockEnvironments: Environment[] = [
  {
    id: '1a2b3c4d-5e6f-7890-abcd-ef1234567890',
    name: 'react-app',
    description: 'React 前端项目',
    templateId: 't1',
    templateName: 'React 18',
    status: 'running',
    resources: { cpu: '2 核', memory: '4 GB', storage: '20 GB' },
    createdAt: '2026-01-15T10:00:00Z',
    lastAccessedAt: '2026-01-12T08:30:00Z',
  },
  {
    id: '2b3c4d5e-6f78-90ab-cdef-123456789012',
    name: 'node-api',
    description: 'Node.js API 服务',
    templateId: 't2',
    templateName: 'Node.js 18',
    status: 'stopped',
    resources: { cpu: '1 核', memory: '2 GB', storage: '10 GB' },
    createdAt: '2026-01-14T08:00:00Z',
  },
  {
    id: '3c4d5e6f-7890-abcd-ef12-345678901234',
    name: 'python-ml',
    description: 'Python 机器学习项目',
    templateId: 't3',
    templateName: 'Python 3.11',
    status: 'running',
    resources: { cpu: '4 核', memory: '8 GB', storage: '50 GB' },
    createdAt: '2026-01-13T14:00:00Z',
    lastAccessedAt: '2026-01-12T10:00:00Z',
  },
  {
    id: '4d5e6f78-90ab-cdef-1234-567890123456',
    name: 'go-service',
    description: 'Go 微服务',
    templateId: 't4',
    templateName: 'Go 1.21',
    status: 'creating',
    resources: { cpu: '2 核', memory: '4 GB', storage: '15 GB' },
    createdAt: '2026-01-12T16:00:00Z',
  },
  {
    id: '5e6f7890-abcd-ef12-3456-789012345678',
    name: 'java-spring',
    description: 'Java Spring Boot 项目',
    templateId: 't5',
    templateName: 'Java 17',
    status: 'failed',
    resources: { cpu: '2 核', memory: '4 GB', storage: '25 GB' },
    createdAt: '2026-01-11T09:00:00Z',
  },
  {
    id: '6f789012-cdef-1234-5678-901234567890',
    name: 'rust-backend',
    description: 'Rust 后端服务',
    templateId: 't6',
    templateName: 'Rust 1.75',
    status: 'suspended',
    resources: { cpu: '2 核', memory: '4 GB', storage: '20 GB' },
    createdAt: '2026-01-10T11:00:00Z',
  },
]

const statusOptions = [
  { label: '全部', value: 'all' },
  { label: '运行中', value: 'running' },
  { label: '已停止', value: 'stopped' },
  { label: '创建中', value: 'creating' },
  { label: '已挂起', value: 'suspended' },
  { label: '失败', value: 'failed' },
]

export default function Environments() {
  const [searchParams, setSearchParams] = useSearchParams()
  const [isModalOpen, setIsModalOpen] = useState(false)
  const [isConnectionDrawerOpen, setIsConnectionDrawerOpen] = useState(false)
  const [isTerminalDrawerOpen, setIsTerminalDrawerOpen] = useState(false)
  const [selectedEnv, setSelectedEnv] = useState<Environment | null>(null)
  const [selectedRowKeys, setSelectedRowKeys] = useState<React.Key[]>([])
  const [searchText, setSearchText] = useState('')
  const [statusFilter, setStatusFilter] = useState<string>('all')
  const [isLoading, setIsLoading] = useState(false)
  const [isRefreshing, setIsRefreshing] = useState(false)
  const [form] = Form.useForm()
  const { environments, setEnvironments, updateEnvironment, removeEnvironment } = useEnvironmentStore()

  const data = environments.length > 0 ? environments : mockEnvironments

  useEffect(() => {
    loadEnvironments()
    const statusParam = searchParams.get('status')
    if (statusParam) setStatusFilter(statusParam)
    const interval = setInterval(() => loadEnvironments(true), 30000)
    return () => clearInterval(interval)
  }, [])

  const loadEnvironments = async (silent = false) => {
    if (!silent) setIsLoading(true)
    try {
      const envList = await environmentService.list()
      setEnvironments(envList)
    } catch {
      console.log('Using mock data')
    } finally {
      setIsLoading(false)
    }
  }

  const handleRefresh = async () => {
    setIsRefreshing(true)
    await loadEnvironments()
    setIsRefreshing(false)
    message.success('已刷新')
  }

  const filteredData = data.filter((env: Environment) => {
    const matchesSearch = 
      env.name.toLowerCase().includes(searchText.toLowerCase()) ||
      env.description?.toLowerCase().includes(searchText.toLowerCase()) ||
      env.templateName.toLowerCase().includes(searchText.toLowerCase())
    const matchesStatus = statusFilter === 'all' || env.status === statusFilter
    return matchesSearch && matchesStatus
  })

  const stats = {
    total: data.length,
    running: data.filter((e: Environment) => e.status === 'running').length,
    stopped: data.filter((e: Environment) => e.status === 'stopped').length,
    other: data.filter((e: Environment) => !['running', 'stopped'].includes(e.status)).length,
  }

  const getStatusTag = (status: EnvironmentStatus) => {
    const config: Record<string, { color: string; text: string }> = {
      running: { color: 'green', text: '运行中' },
      stopped: { color: 'default', text: '已停止' },
      creating: { color: 'blue', text: '创建中' },
      failed: { color: 'red', text: '失败' },
      suspended: { color: 'orange', text: '已挂起' },
    }
    const { color, text } = config[status] || { color: 'default', text: status }
    return <Tag color={color}>{text}</Tag>
  }

  const handleOpenConnection = (env: Environment) => {
    setSelectedEnv(env)
    setIsConnectionDrawerOpen(true)
  }

  const handleOpenTerminal = (env: Environment) => {
    if (env.status !== 'running') {
      message.warning('请先启动环境')
      return
    }
    setSelectedEnv(env)
    setIsTerminalDrawerOpen(true)
  }

  const handleStart = async (env: Environment) => {
    try {
      await environmentService.start(env.id)
      updateEnvironment(env.id, { status: 'running' })
      message.success(`环境 ${env.name} 已启动`)
    } catch {
      message.error('启动失败')
    }
  }

  const handleStop = async (env: Environment) => {
    try {
      await environmentService.stop(env.id)
      updateEnvironment(env.id, { status: 'stopped' })
      message.success(`环境 ${env.name} 已停止`)
    } catch {
      message.error('停止失败')
    }
  }

  const handleDelete = async (env: Environment) => {
    try {
      await environmentService.delete(env.id)
      removeEnvironment(env.id)
      message.success(`环境 ${env.name} 已删除`)
    } catch {
      message.error('删除失败')
    }
  }

  const handleBatchStart = async () => {
    const selectedEnvs = data.filter((e: Environment) => selectedRowKeys.includes(e.id) && e.status === 'stopped')
    if (selectedEnvs.length === 0) {
      message.warning('没有可启动的环境')
      return
    }
    try {
      await environmentService.batchStart(selectedEnvs.map((e: Environment) => e.id))
      selectedEnvs.forEach((env: Environment) => updateEnvironment(env.id, { status: 'running' }))
      message.success(`已启动 ${selectedEnvs.length} 个环境`)
      setSelectedRowKeys([])
    } catch {
      message.error('批量启动失败')
    }
  }

  const handleBatchStop = async () => {
    const selectedEnvs = data.filter((e: Environment) => selectedRowKeys.includes(e.id) && e.status === 'running')
    if (selectedEnvs.length === 0) {
      message.warning('没有可停止的环境')
      return
    }
    try {
      await environmentService.batchStop(selectedEnvs.map((e: Environment) => e.id))
      selectedEnvs.forEach((env: Environment) => updateEnvironment(env.id, { status: 'stopped' }))
      message.success(`已停止 ${selectedEnvs.length} 个环境`)
      setSelectedRowKeys([])
    } catch {
      message.error('批量停止失败')
    }
  }

  const handleBatchDelete = () => {
    Modal.confirm({
      title: '确认删除',
      icon: <ExclamationCircleOutlined />,
      content: `确定要删除选中的 ${selectedRowKeys.length} 个环境吗？此操作不可恢复。`,
      okText: '删除',
      okType: 'danger',
      cancelText: '取消',
      onOk: async () => {
        try {
          await environmentService.batchDelete(selectedRowKeys as string[])
          selectedRowKeys.forEach((id: React.Key) => removeEnvironment(id as string))
          message.success(`已删除 ${selectedRowKeys.length} 个环境`)
          setSelectedRowKeys([])
        } catch {
          message.error('批量删除失败')
        }
      },
    })
  }

  const formatDate = (dateStr: string) => {
    return new Date(dateStr).toLocaleString('zh-CN', {
      year: 'numeric', month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit',
    })
  }

  const columns: ColumnsType<Environment> = [
    {
      title: '名称',
      dataIndex: 'name',
      key: 'name',
      sorter: (a: Environment, b: Environment) => a.name.localeCompare(b.name),
      render: (text: string, record: Environment) => (
        <div>
          <div className="font-medium flex items-center gap-2">
            <CloudServerOutlined className="text-blue-500" />
            {text}
          </div>
          {record.description && <div className="text-gray-500 text-sm">{record.description}</div>}
        </div>
      ),
    },
    {
      title: '模板',
      dataIndex: 'templateName',
      key: 'templateName',
      filters: [...new Set(data.map((e: Environment) => e.templateName))].map(t => ({ text: t, value: t })),
      onFilter: (value: React.Key | boolean, record: Environment) => record.templateName === value,
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 100,
      render: (status: EnvironmentStatus) => getStatusTag(status),
    },
    {
      title: '资源',
      key: 'resources',
      width: 150,
      render: (_: unknown, record: Environment) => (
        <div className="text-sm">
          <div>CPU: {record.resources.cpu}</div>
          <div>内存: {record.resources.memory}</div>
          <div>存储: {record.resources.storage}</div>
        </div>
      ),
    },
    {
      title: '创建时间',
      dataIndex: 'createdAt',
      key: 'createdAt',
      width: 160,
      sorter: (a: Environment, b: Environment) => new Date(a.createdAt).getTime() - new Date(b.createdAt).getTime(),
      render: (date: string) => formatDate(date),
    },
    {
      title: '操作',
      key: 'action',
      width: 280,
      render: (_: unknown, record: Environment) => (
        <Space size="small">
          {record.status === 'running' ? (
            <>
              <Tooltip title="停止"><Button icon={<PauseCircleOutlined />} size="small" onClick={() => handleStop(record)} /></Tooltip>
              <Tooltip title="连接"><Button icon={<LinkOutlined />} size="small" type="primary" onClick={() => handleOpenConnection(record)} /></Tooltip>
              <Tooltip title="终端"><Button icon={<CodeOutlined />} size="small" onClick={() => handleOpenTerminal(record)} /></Tooltip>
            </>
          ) : record.status === 'stopped' ? (
            <Tooltip title="启动"><Button icon={<PlayCircleOutlined />} size="small" type="primary" onClick={() => handleStart(record)} /></Tooltip>
          ) : record.status === 'creating' ? (
            <Badge status="processing" text="创建中..." />
          ) : record.status === 'failed' ? (
            <Tooltip title="重试"><Button icon={<ReloadOutlined />} size="small" danger onClick={() => handleStart(record)} /></Tooltip>
          ) : null}
          <Dropdown
            menu={{
              items: [
                { key: 'config', icon: <SettingOutlined />, label: '配置', onClick: () => message.info('配置功能开发中') },
                { type: 'divider' },
                { key: 'delete', icon: <DeleteOutlined />, label: '删除', danger: true, onClick: () => {
                  Modal.confirm({
                    title: '确认删除',
                    icon: <ExclamationCircleOutlined />,
                    content: `确定要删除环境 "${record.name}" 吗？`,
                    okText: '删除',
                    okType: 'danger',
                    cancelText: '取消',
                    onOk: () => handleDelete(record),
                  })
                }},
              ],
            }}
            trigger={['click']}
          >
            <Button icon={<MoreOutlined />} size="small" />
          </Dropdown>
        </Space>
      ),
    },
  ]

  const rowSelection: TableRowSelection<Environment> = {
    selectedRowKeys,
    onChange: (keys: React.Key[]) => setSelectedRowKeys(keys),
    getCheckboxProps: (record: Environment) => ({ disabled: record.status === 'creating' }),
  }

  const handleCreate = () => {
    form.validateFields().then((values: Record<string, unknown>) => {
      console.log('Creating environment:', values)
      setIsModalOpen(false)
      form.resetFields()
      message.success('环境创建中...')
    })
  }

  return (
    <div>
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-2xl font-semibold">开发环境</h1>
        <Button type="primary" icon={<PlusOutlined />} onClick={() => setIsModalOpen(true)}>创建环境</Button>
      </div>

      <Row gutter={16} className="mb-4">
        <Col span={6}><Card size="small"><Statistic title="总计" value={stats.total} prefix={<CloudServerOutlined />} /></Card></Col>
        <Col span={6}><Card size="small"><Statistic title="运行中" value={stats.running} valueStyle={{ color: '#52c41a' }} /></Card></Col>
        <Col span={6}><Card size="small"><Statistic title="已停止" value={stats.stopped} valueStyle={{ color: '#8c8c8c' }} /></Card></Col>
        <Col span={6}><Card size="small"><Statistic title="其他" value={stats.other} valueStyle={{ color: '#faad14' }} /></Card></Col>
      </Row>

      <Card>
        <div className="flex justify-between items-center mb-4">
          <Space>
            <Input placeholder="搜索环境名称..." prefix={<SearchOutlined />} value={searchText} onChange={(e: React.ChangeEvent<HTMLInputElement>) => setSearchText(e.target.value)} style={{ width: 250 }} allowClear />
            <Select value={statusFilter} onChange={(value: string) => { setStatusFilter(value); if (value === 'all') { searchParams.delete('status') } else { searchParams.set('status', value) }; setSearchParams(searchParams) }} options={statusOptions} style={{ width: 120 }} suffixIcon={<FilterOutlined />} />
          </Space>
          <Space>
            {selectedRowKeys.length > 0 && (
              <>
                <span className="text-gray-500">已选择 {selectedRowKeys.length} 项</span>
                <Button onClick={handleBatchStart}>批量启动</Button>
                <Button onClick={handleBatchStop}>批量停止</Button>
                <Button danger onClick={handleBatchDelete}>批量删除</Button>
              </>
            )}
            <Tooltip title="刷新"><Button icon={<ReloadOutlined spin={isRefreshing} />} onClick={handleRefresh} /></Tooltip>
          </Space>
        </div>
        <Table columns={columns} dataSource={filteredData} rowKey="id" rowSelection={rowSelection} loading={isLoading} pagination={{ showSizeChanger: true, showQuickJumper: true, showTotal: (total: number) => `共 ${total} 个环境` }} />
      </Card>

      <Modal title="创建开发环境" open={isModalOpen} onOk={handleCreate} onCancel={() => setIsModalOpen(false)} okText="创建" cancelText="取消">
        <Form form={form} layout="vertical">
          <Form.Item name="name" label="环境名称" rules={[{ required: true, message: '请输入环境名称' }, { pattern: /^[a-z0-9-]+$/, message: '只能包含小写字母、数字和连字符' }]}>
            <Input placeholder="my-project" />
          </Form.Item>
          <Form.Item name="description" label="描述"><Input.TextArea placeholder="环境描述（可选）" /></Form.Item>
          <Form.Item name="templateId" label="选择模板" rules={[{ required: true, message: '请选择模板' }]}>
            <Select placeholder="选择环境模板">
              <Select.Option value="react">React 18</Select.Option>
              <Select.Option value="vue">Vue 3</Select.Option>
              <Select.Option value="node">Node.js 18</Select.Option>
              <Select.Option value="python">Python 3.11</Select.Option>
              <Select.Option value="go">Go 1.21</Select.Option>
              <Select.Option value="java">Java 17</Select.Option>
              <Select.Option value="rust">Rust 1.75</Select.Option>
            </Select>
          </Form.Item>
        </Form>
      </Modal>

      <Drawer title={`连接到 ${selectedEnv?.name || ''}`} placement="right" width={720} open={isConnectionDrawerOpen} onClose={() => setIsConnectionDrawerOpen(false)}>
        {selectedEnv && (
          <Tabs defaultActiveKey="connection" items={[
            { key: 'connection', label: <span><LinkOutlined /> 连接信息</span>, children: <ConnectionInfoPanel environmentId={selectedEnv.id} environmentName={selectedEnv.name} isRunning={selectedEnv.status === 'running'} /> },
            { key: 'guide', label: <span><DesktopOutlined /> IDE 指引</span>, children: <IDEConnectionGuide environmentId={selectedEnv.id} environmentName={selectedEnv.name} /> },
          ]} />
        )}
      </Drawer>

      <Drawer title={`终端 - ${selectedEnv?.name || ''}`} placement="bottom" height="60vh" open={isTerminalDrawerOpen} onClose={() => setIsTerminalDrawerOpen(false)} styles={{ body: { padding: 0 } }}>
        {selectedEnv && <WebTerminal environmentId={selectedEnv.id} environmentName={selectedEnv.name} isRunning={selectedEnv.status === 'running'} onClose={() => setIsTerminalDrawerOpen(false)} />}
      </Drawer>
    </div>
  )
}
