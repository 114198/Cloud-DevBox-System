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
  Progress,
  Divider,
  InputNumber,
  Switch,
  Collapse,
} from 'antd'
import {
  PlusOutlined,
  RocketOutlined,
  RollbackOutlined,
  DeleteOutlined,
  HistoryOutlined,
  FileTextOutlined,
  SearchOutlined,
  FilterOutlined,
  ReloadOutlined,
  MoreOutlined,
  CloudUploadOutlined,
  CheckCircleOutlined,
  CloseCircleOutlined,
  LoadingOutlined,
  ClockCircleOutlined,
  SettingOutlined,
  CodeOutlined,
} from '@ant-design/icons'
import type { ColumnsType } from 'antd/es/table'
import { useSearchParams, useNavigate } from 'react-router-dom'
import { useDeploymentStore, getPhaseInfo, formatDuration } from '@/stores/deployment'
import { 
  deploymentService, 
  Deployment, 
  DeploymentPhase,
  CreateDeploymentRequest,
  BuildConfig,
  DeployConfig,
  EnvVar,
  PortMapping,
} from '@/services/deployment'

// Mock data for development
const mockDeployments: Deployment[] = [
  {
    id: 'dep-1',
    environmentId: 'env-1',
    userId: 'user-1',
    name: 'react-app',
    version: 'v1704067200',
    phase: 'Running',
    message: 'Deployment successful',
    buildConfig: {
      dockerfilePath: 'Dockerfile',
      contextPath: '.',
      platform: 'linux/amd64',
      sourceType: 'git',
      gitUrl: 'https://github.com/user/react-app',
      gitBranch: 'main',
    },
    deployConfig: {
      replicas: 2,
      resources: { cpu: '500m', memory: '512Mi' },
      environment: [
        { name: 'NODE_ENV', value: 'production' },
        { name: 'API_URL', value: 'https://api.example.com' },
      ],
      ports: [{ name: 'http', containerPort: 3000 }],
      serviceType: 'ClusterIP',
    },
    imageInfo: {
      name: 'registry.devbox.io/user-1/react-app',
      tag: 'v1704067200',
      digest: 'sha256:abc123',
      buildDuration: 45.5,
      pushDuration: 12.3,
      createdAt: '2026-01-12T10:00:00Z',
    },
    k8sResources: {
      deploymentName: 'react-app-dep-1',
      serviceName: 'react-app-dep-1',
      namespace: 'devbox-deployments',
      externalUrl: 'https://react-app.devbox.io',
    },
    metrics: {
      buildTime: 45.5,
      pushTime: 12.3,
      deployTime: 8.2,
      totalTime: 66.0,
    },
    createdAt: '2026-01-12T10:00:00Z',
    updatedAt: '2026-01-12T10:01:06Z',
    startedAt: '2026-01-12T10:00:00Z',
    completedAt: '2026-01-12T10:01:06Z',
  },
  {
    id: 'dep-2',
    environmentId: 'env-2',
    userId: 'user-1',
    name: 'node-api',
    version: 'v1704153600',
    phase: 'Building',
    message: 'Building Docker image',
    buildConfig: {
      dockerfilePath: 'Dockerfile',
      contextPath: '.',
      platform: 'linux/amd64',
    },
    deployConfig: {
      replicas: 1,
      resources: { cpu: '250m', memory: '256Mi' },
    },
    createdAt: '2026-01-13T10:00:00Z',
    updatedAt: '2026-01-13T10:00:30Z',
    startedAt: '2026-01-13T10:00:00Z',
  },
  {
    id: 'dep-3',
    environmentId: 'env-3',
    userId: 'user-1',
    name: 'python-ml',
    version: 'v1704240000',
    phase: 'Failed',
    message: 'Build failed: Dockerfile not found',
    createdAt: '2026-01-14T10:00:00Z',
    updatedAt: '2026-01-14T10:02:00Z',
    startedAt: '2026-01-14T10:00:00Z',
  },
]

const phaseOptions = [
  { label: '全部', value: 'all' },
  { label: '运行中', value: 'Running' },
  { label: '构建中', value: 'Building' },
  { label: '部署中', value: 'Deploying' },
  { label: '失败', value: 'Failed' },
  { label: '已回滚', value: 'RolledBack' },
]

const languageOptions = [
  { label: 'Node.js', value: 'nodejs' },
  { label: 'Go', value: 'go' },
  { label: 'Python', value: 'python' },
  { label: 'Java', value: 'java' },
  { label: 'Rust', value: 'rust' },
]

const frameworkOptions: Record<string, { label: string; value: string }[]> = {
  nodejs: [
    { label: '无框架', value: '' },
    { label: 'Next.js', value: 'nextjs' },
    { label: 'Express', value: 'express' },
  ],
  python: [
    { label: '无框架', value: '' },
    { label: 'Django', value: 'django' },
    { label: 'FastAPI', value: 'fastapi' },
  ],
  go: [{ label: '无框架', value: '' }],
  java: [
    { label: '无框架', value: '' },
    { label: 'Spring Boot', value: 'springboot' },
  ],
  rust: [{ label: '无框架', value: '' }],
}

export default function Deployments() {
  const [searchParams, setSearchParams] = useSearchParams()
  const navigate = useNavigate()
  const [isCreateModalOpen, setIsCreateModalOpen] = useState(false)
  const [isConfigDrawerOpen, setIsConfigDrawerOpen] = useState(false)
  const [selectedDeployment, setSelectedDeployment] = useState<Deployment | null>(null)
  const [searchText, setSearchText] = useState('')
  const [phaseFilter, setPhaseFilter] = useState<string>('all')
  const [isLoading, setIsLoading] = useState(false)
  const [isRefreshing, setIsRefreshing] = useState(false)
  const [selectedLanguage, setSelectedLanguage] = useState<string>('nodejs')
  const [envVars, setEnvVars] = useState<EnvVar[]>([{ name: '', value: '' }])
  const [ports, setPorts] = useState<PortMapping[]>([{ name: 'http', containerPort: 8080 }])
  const [form] = Form.useForm()
  const { deployments, setDeployments, addDeployment, updateDeployment, removeDeployment } = useDeploymentStore()

  const data = deployments.length > 0 ? deployments : mockDeployments

  useEffect(() => {
    loadDeployments()
    const phaseParam = searchParams.get('phase')
    if (phaseParam) setPhaseFilter(phaseParam)
    const interval = setInterval(() => loadDeployments(true), 10000)
    return () => clearInterval(interval)
  }, [])

  const loadDeployments = async (silent = false) => {
    if (!silent) setIsLoading(true)
    try {
      const response = await deploymentService.list()
      setDeployments(response.deployments)
    } catch {
      console.log('Using mock data')
    } finally {
      setIsLoading(false)
    }
  }

  const handleRefresh = async () => {
    setIsRefreshing(true)
    await loadDeployments()
    setIsRefreshing(false)
    message.success('已刷新')
  }

  const filteredData = data.filter((dep: Deployment) => {
    const matchesSearch = 
      dep.name.toLowerCase().includes(searchText.toLowerCase()) ||
      dep.version.toLowerCase().includes(searchText.toLowerCase())
    const matchesPhase = phaseFilter === 'all' || dep.phase === phaseFilter
    return matchesSearch && matchesPhase
  })

  const stats = {
    total: data.length,
    running: data.filter((d: Deployment) => d.phase === 'Running').length,
    building: data.filter((d: Deployment) => ['Building', 'Pushing', 'Deploying', 'Pending'].includes(d.phase)).length,
    failed: data.filter((d: Deployment) => d.phase === 'Failed').length,
  }

  const getPhaseTag = (phase: DeploymentPhase) => {
    const info = getPhaseInfo(phase)
    const iconMap: Record<string, React.ReactNode> = {
      'clock-circle': <ClockCircleOutlined />,
      'loading': <LoadingOutlined />,
      'cloud-upload': <CloudUploadOutlined />,
      'deployment-unit': <RocketOutlined />,
      'check-circle': <CheckCircleOutlined />,
      'close-circle': <CloseCircleOutlined />,
      'rollback': <RollbackOutlined />,
    }
    return (
      <Tag color={info.color} icon={iconMap[info.icon]}>
        {info.text}
      </Tag>
    )
  }

  const handleOpenConfig = (dep: Deployment) => {
    setSelectedDeployment(dep)
    setIsConfigDrawerOpen(true)
    // Populate form with existing config
    if (dep.deployConfig) {
      setEnvVars(dep.deployConfig.environment || [{ name: '', value: '' }])
      setPorts(dep.deployConfig.ports || [{ name: 'http', containerPort: 8080 }])
    }
  }

  const handleViewHistory = (dep: Deployment) => {
    navigate(`/deployments/${dep.id}/history`)
  }

  const handleViewLogs = (dep: Deployment) => {
    navigate(`/deployments/${dep.id}/logs`)
  }

  const handleRollback = async (dep: Deployment) => {
    Modal.confirm({
      title: '确认回滚',
      content: `确定要回滚部署 "${dep.name}" 到上一个版本吗？`,
      okText: '回滚',
      cancelText: '取消',
      onOk: async () => {
        try {
          await deploymentService.rollback(dep.id, { targetVersion: 'previous', reason: '手动回滚' })
          message.success('回滚成功')
          loadDeployments()
        } catch {
          message.error('回滚失败')
        }
      },
    })
  }

  const handleDelete = async (dep: Deployment) => {
    Modal.confirm({
      title: '确认删除',
      content: `确定要删除部署 "${dep.name}" 吗？此操作不可恢复。`,
      okText: '删除',
      okType: 'danger',
      cancelText: '取消',
      onOk: async () => {
        try {
          await deploymentService.delete(dep.id)
          removeDeployment(dep.id)
          message.success('删除成功')
        } catch {
          message.error('删除失败')
        }
      },
    })
  }

  const formatDate = (dateStr?: string) => {
    if (!dateStr) return '-'
    return new Date(dateStr).toLocaleString('zh-CN', {
      year: 'numeric', month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit',
    })
  }

  const columns: ColumnsType<Deployment> = [
    {
      title: '名称',
      dataIndex: 'name',
      key: 'name',
      sorter: (a: Deployment, b: Deployment) => a.name.localeCompare(b.name),
      render: (text: string, record: Deployment) => (
        <div>
          <div className="font-medium flex items-center gap-2">
            <RocketOutlined className="text-blue-500" />
            {text}
          </div>
          <div className="text-gray-500 text-sm">{record.version}</div>
        </div>
      ),
    },
    {
      title: '状态',
      dataIndex: 'phase',
      key: 'phase',
      width: 120,
      render: (phase: DeploymentPhase) => getPhaseTag(phase),
    },
    {
      title: '镜像',
      key: 'image',
      width: 200,
      render: (_: unknown, record: Deployment) => (
        record.imageInfo ? (
          <div className="text-sm">
            <div className="truncate max-w-[180px]" title={record.imageInfo.name}>
              {record.imageInfo.name.split('/').pop()}
            </div>
            <div className="text-gray-500">{record.imageInfo.tag}</div>
          </div>
        ) : <span className="text-gray-400">-</span>
      ),
    },
    {
      title: '资源',
      key: 'resources',
      width: 120,
      render: (_: unknown, record: Deployment) => (
        record.deployConfig?.resources ? (
          <div className="text-sm">
            <div>CPU: {record.deployConfig.resources.cpu}</div>
            <div>内存: {record.deployConfig.resources.memory}</div>
          </div>
        ) : <span className="text-gray-400">-</span>
      ),
    },
    {
      title: '耗时',
      key: 'duration',
      width: 100,
      render: (_: unknown, record: Deployment) => (
        <span>{formatDuration(record.metrics?.totalTime)}</span>
      ),
    },
    {
      title: '创建时间',
      dataIndex: 'createdAt',
      key: 'createdAt',
      width: 160,
      sorter: (a: Deployment, b: Deployment) => new Date(a.createdAt).getTime() - new Date(b.createdAt).getTime(),
      render: (date: string) => formatDate(date),
    },
    {
      title: '操作',
      key: 'action',
      width: 200,
      render: (_: unknown, record: Deployment) => (
        <Space size="small">
          <Tooltip title="配置">
            <Button icon={<SettingOutlined />} size="small" onClick={() => handleOpenConfig(record)} />
          </Tooltip>
          <Tooltip title="日志">
            <Button icon={<FileTextOutlined />} size="small" onClick={() => handleViewLogs(record)} />
          </Tooltip>
          {record.phase === 'Running' && (
            <Tooltip title="回滚">
              <Button icon={<RollbackOutlined />} size="small" onClick={() => handleRollback(record)} />
            </Tooltip>
          )}
          <Dropdown
            menu={{
              items: [
                { key: 'history', icon: <HistoryOutlined />, label: '历史版本', onClick: () => handleViewHistory(record) },
                { type: 'divider' },
                { key: 'delete', icon: <DeleteOutlined />, label: '删除', danger: true, onClick: () => handleDelete(record) },
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

  const handleAddEnvVar = () => {
    setEnvVars([...envVars, { name: '', value: '' }])
  }

  const handleRemoveEnvVar = (index: number) => {
    setEnvVars(envVars.filter((_, i) => i !== index))
  }

  const handleEnvVarChange = (index: number, field: 'name' | 'value', value: string) => {
    const newEnvVars = [...envVars]
    newEnvVars[index][field] = value
    setEnvVars(newEnvVars)
  }

  const handleAddPort = () => {
    setPorts([...ports, { name: '', containerPort: 8080 }])
  }

  const handleRemovePort = (index: number) => {
    setPorts(ports.filter((_, i) => i !== index))
  }

  const handlePortChange = (index: number, field: 'name' | 'containerPort', value: string | number) => {
    const newPorts = [...ports]
    if (field === 'containerPort') {
      newPorts[index][field] = value as number
    } else {
      newPorts[index][field] = value as string
    }
    setPorts(newPorts)
  }

  const handleCreate = async () => {
    try {
      const values = await form.validateFields()
      const request: CreateDeploymentRequest = {
        environmentId: values.environmentId,
        name: values.name,
        buildConfig: {
          dockerfilePath: values.dockerfilePath || 'Dockerfile',
          contextPath: values.contextPath || '.',
          platform: values.platform || 'linux/amd64',
          sourceType: values.sourceType || 'environment',
          gitUrl: values.gitUrl,
          gitBranch: values.gitBranch || 'main',
        },
        deployConfig: {
          replicas: values.replicas || 1,
          resources: {
            cpu: values.cpu || '500m',
            memory: values.memory || '512Mi',
          },
          environment: envVars.filter(e => e.name && e.value),
          ports: ports.filter(p => p.name && p.containerPort),
          serviceType: values.serviceType || 'ClusterIP',
        },
      }
      
      const deployment = await deploymentService.create(request)
      addDeployment(deployment)
      setIsCreateModalOpen(false)
      form.resetFields()
      setEnvVars([{ name: '', value: '' }])
      setPorts([{ name: 'http', containerPort: 8080 }])
      message.success('部署创建成功')
    } catch (error) {
      console.error('Create deployment error:', error)
      message.error('创建部署失败')
    }
  }

  return (
    <div>
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-2xl font-semibold">部署管理</h1>
        <Button type="primary" icon={<PlusOutlined />} onClick={() => setIsCreateModalOpen(true)}>
          新建部署
        </Button>
      </div>

      <Row gutter={16} className="mb-4">
        <Col span={6}>
          <Card size="small">
            <Statistic title="总计" value={stats.total} prefix={<RocketOutlined />} />
          </Card>
        </Col>
        <Col span={6}>
          <Card size="small">
            <Statistic title="运行中" value={stats.running} valueStyle={{ color: '#52c41a' }} />
          </Card>
        </Col>
        <Col span={6}>
          <Card size="small">
            <Statistic title="进行中" value={stats.building} valueStyle={{ color: '#1890ff' }} />
          </Card>
        </Col>
        <Col span={6}>
          <Card size="small">
            <Statistic title="失败" value={stats.failed} valueStyle={{ color: '#ff4d4f' }} />
          </Card>
        </Col>
      </Row>

      <Card>
        <div className="flex justify-between items-center mb-4">
          <Space>
            <Input
              placeholder="搜索部署名称..."
              prefix={<SearchOutlined />}
              value={searchText}
              onChange={(e) => setSearchText(e.target.value)}
              style={{ width: 250 }}
              allowClear
            />
            <Select
              value={phaseFilter}
              onChange={(value) => {
                setPhaseFilter(value)
                if (value === 'all') {
                  searchParams.delete('phase')
                } else {
                  searchParams.set('phase', value)
                }
                setSearchParams(searchParams)
              }}
              options={phaseOptions}
              style={{ width: 120 }}
              suffixIcon={<FilterOutlined />}
            />
          </Space>
          <Tooltip title="刷新">
            <Button icon={<ReloadOutlined spin={isRefreshing} />} onClick={handleRefresh} />
          </Tooltip>
        </div>
        <Table
          columns={columns}
          dataSource={filteredData}
          rowKey="id"
          loading={isLoading}
          pagination={{
            showSizeChanger: true,
            showQuickJumper: true,
            showTotal: (total) => `共 ${total} 个部署`,
          }}
        />
      </Card>

      {/* Create Deployment Modal */}
      <Modal
        title="新建部署"
        open={isCreateModalOpen}
        onOk={handleCreate}
        onCancel={() => setIsCreateModalOpen(false)}
        okText="创建"
        cancelText="取消"
        width={700}
      >
        <Form form={form} layout="vertical">
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item
                name="name"
                label="部署名称"
                rules={[
                  { required: true, message: '请输入部署名称' },
                  { pattern: /^[a-z0-9-]+$/, message: '只能包含小写字母、数字和连字符' },
                ]}
              >
                <Input placeholder="my-app" />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item
                name="environmentId"
                label="关联环境"
                rules={[{ required: true, message: '请选择环境' }]}
              >
                <Select placeholder="选择环境">
                  <Select.Option value="env-1">react-app</Select.Option>
                  <Select.Option value="env-2">node-api</Select.Option>
                  <Select.Option value="env-3">python-ml</Select.Option>
                </Select>
              </Form.Item>
            </Col>
          </Row>

          <Divider>构建配置</Divider>

          <Row gutter={16}>
            <Col span={8}>
              <Form.Item name="language" label="语言">
                <Select
                  placeholder="选择语言"
                  options={languageOptions}
                  onChange={(value) => setSelectedLanguage(value)}
                />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item name="framework" label="框架">
                <Select
                  placeholder="选择框架"
                  options={frameworkOptions[selectedLanguage] || []}
                />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item name="platform" label="平台" initialValue="linux/amd64">
                <Select>
                  <Select.Option value="linux/amd64">linux/amd64</Select.Option>
                  <Select.Option value="linux/arm64">linux/arm64</Select.Option>
                </Select>
              </Form.Item>
            </Col>
          </Row>

          <Row gutter={16}>
            <Col span={12}>
              <Form.Item name="dockerfilePath" label="Dockerfile 路径" initialValue="Dockerfile">
                <Input placeholder="Dockerfile" />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="contextPath" label="构建上下文" initialValue=".">
                <Input placeholder="." />
              </Form.Item>
            </Col>
          </Row>

          <Divider>部署配置</Divider>

          <Row gutter={16}>
            <Col span={8}>
              <Form.Item name="replicas" label="副本数" initialValue={1}>
                <InputNumber min={1} max={10} style={{ width: '100%' }} />
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item name="cpu" label="CPU" initialValue="500m">
                <Select>
                  <Select.Option value="100m">100m</Select.Option>
                  <Select.Option value="250m">250m</Select.Option>
                  <Select.Option value="500m">500m</Select.Option>
                  <Select.Option value="1000m">1000m</Select.Option>
                  <Select.Option value="2000m">2000m</Select.Option>
                </Select>
              </Form.Item>
            </Col>
            <Col span={8}>
              <Form.Item name="memory" label="内存" initialValue="512Mi">
                <Select>
                  <Select.Option value="128Mi">128Mi</Select.Option>
                  <Select.Option value="256Mi">256Mi</Select.Option>
                  <Select.Option value="512Mi">512Mi</Select.Option>
                  <Select.Option value="1Gi">1Gi</Select.Option>
                  <Select.Option value="2Gi">2Gi</Select.Option>
                </Select>
              </Form.Item>
            </Col>
          </Row>

          <Form.Item name="serviceType" label="服务类型" initialValue="ClusterIP">
            <Select>
              <Select.Option value="ClusterIP">ClusterIP (内部访问)</Select.Option>
              <Select.Option value="NodePort">NodePort (节点端口)</Select.Option>
              <Select.Option value="LoadBalancer">LoadBalancer (负载均衡)</Select.Option>
            </Select>
          </Form.Item>

          <Collapse ghost>
            <Collapse.Panel header="环境变量" key="env">
              {envVars.map((env, index) => (
                <Row gutter={8} key={index} className="mb-2">
                  <Col span={10}>
                    <Input
                      placeholder="变量名"
                      value={env.name}
                      onChange={(e) => handleEnvVarChange(index, 'name', e.target.value)}
                    />
                  </Col>
                  <Col span={10}>
                    <Input
                      placeholder="变量值"
                      value={env.value}
                      onChange={(e) => handleEnvVarChange(index, 'value', e.target.value)}
                    />
                  </Col>
                  <Col span={4}>
                    <Button danger onClick={() => handleRemoveEnvVar(index)} disabled={envVars.length === 1}>
                      删除
                    </Button>
                  </Col>
                </Row>
              ))}
              <Button type="dashed" onClick={handleAddEnvVar} block icon={<PlusOutlined />}>
                添加环境变量
              </Button>
            </Collapse.Panel>

            <Collapse.Panel header="端口映射" key="ports">
              {ports.map((port, index) => (
                <Row gutter={8} key={index} className="mb-2">
                  <Col span={10}>
                    <Input
                      placeholder="端口名称"
                      value={port.name}
                      onChange={(e) => handlePortChange(index, 'name', e.target.value)}
                    />
                  </Col>
                  <Col span={10}>
                    <InputNumber
                      placeholder="容器端口"
                      value={port.containerPort}
                      onChange={(value) => handlePortChange(index, 'containerPort', value || 8080)}
                      min={1}
                      max={65535}
                      style={{ width: '100%' }}
                    />
                  </Col>
                  <Col span={4}>
                    <Button danger onClick={() => handleRemovePort(index)} disabled={ports.length === 1}>
                      删除
                    </Button>
                  </Col>
                </Row>
              ))}
              <Button type="dashed" onClick={handleAddPort} block icon={<PlusOutlined />}>
                添加端口
              </Button>
            </Collapse.Panel>
          </Collapse>
        </Form>
      </Modal>

      {/* Configuration Drawer */}
      <Drawer
        title={`部署配置 - ${selectedDeployment?.name || ''}`}
        placement="right"
        width={600}
        open={isConfigDrawerOpen}
        onClose={() => setIsConfigDrawerOpen(false)}
      >
        {selectedDeployment && (
          <Tabs
            defaultActiveKey="build"
            items={[
              {
                key: 'build',
                label: <span><CodeOutlined /> 构建配置</span>,
                children: (
                  <div>
                    <Card size="small" title="Dockerfile 配置" className="mb-4">
                      <Row gutter={[16, 16]}>
                        <Col span={12}>
                          <div className="text-gray-500 text-sm">Dockerfile 路径</div>
                          <div>{selectedDeployment.buildConfig?.dockerfilePath || 'Dockerfile'}</div>
                        </Col>
                        <Col span={12}>
                          <div className="text-gray-500 text-sm">构建上下文</div>
                          <div>{selectedDeployment.buildConfig?.contextPath || '.'}</div>
                        </Col>
                        <Col span={12}>
                          <div className="text-gray-500 text-sm">目标平台</div>
                          <div>{selectedDeployment.buildConfig?.platform || 'linux/amd64'}</div>
                        </Col>
                        <Col span={12}>
                          <div className="text-gray-500 text-sm">源类型</div>
                          <div>{selectedDeployment.buildConfig?.sourceType || 'environment'}</div>
                        </Col>
                      </Row>
                    </Card>
                    {selectedDeployment.buildConfig?.gitUrl && (
                      <Card size="small" title="Git 配置">
                        <Row gutter={[16, 16]}>
                          <Col span={24}>
                            <div className="text-gray-500 text-sm">仓库地址</div>
                            <div>{selectedDeployment.buildConfig.gitUrl}</div>
                          </Col>
                          <Col span={12}>
                            <div className="text-gray-500 text-sm">分支</div>
                            <div>{selectedDeployment.buildConfig.gitBranch || 'main'}</div>
                          </Col>
                          <Col span={12}>
                            <div className="text-gray-500 text-sm">Commit</div>
                            <div>{selectedDeployment.buildConfig.gitCommit || '-'}</div>
                          </Col>
                        </Row>
                      </Card>
                    )}
                  </div>
                ),
              },
              {
                key: 'deploy',
                label: <span><RocketOutlined /> 部署配置</span>,
                children: (
                  <div>
                    <Card size="small" title="资源配置" className="mb-4">
                      <Row gutter={[16, 16]}>
                        <Col span={8}>
                          <div className="text-gray-500 text-sm">副本数</div>
                          <div>{selectedDeployment.deployConfig?.replicas || 1}</div>
                        </Col>
                        <Col span={8}>
                          <div className="text-gray-500 text-sm">CPU</div>
                          <div>{selectedDeployment.deployConfig?.resources?.cpu || '-'}</div>
                        </Col>
                        <Col span={8}>
                          <div className="text-gray-500 text-sm">内存</div>
                          <div>{selectedDeployment.deployConfig?.resources?.memory || '-'}</div>
                        </Col>
                      </Row>
                    </Card>
                    <Card size="small" title="环境变量" className="mb-4">
                      {selectedDeployment.deployConfig?.environment?.length ? (
                        selectedDeployment.deployConfig.environment.map((env, index) => (
                          <div key={index} className="flex justify-between py-1 border-b last:border-b-0">
                            <span className="font-mono text-sm">{env.name}</span>
                            <span className="text-gray-500 text-sm">{env.value}</span>
                          </div>
                        ))
                      ) : (
                        <div className="text-gray-400">无环境变量</div>
                      )}
                    </Card>
                    <Card size="small" title="端口映射">
                      {selectedDeployment.deployConfig?.ports?.length ? (
                        selectedDeployment.deployConfig.ports.map((port, index) => (
                          <div key={index} className="flex justify-between py-1 border-b last:border-b-0">
                            <span>{port.name}</span>
                            <span className="text-gray-500">{port.containerPort}</span>
                          </div>
                        ))
                      ) : (
                        <div className="text-gray-400">无端口映射</div>
                      )}
                    </Card>
                  </div>
                ),
              },
              {
                key: 'k8s',
                label: <span><CloudUploadOutlined /> K8s 资源</span>,
                children: (
                  <div>
                    {selectedDeployment.k8sResources ? (
                      <Card size="small">
                        <Row gutter={[16, 16]}>
                          <Col span={12}>
                            <div className="text-gray-500 text-sm">Deployment</div>
                            <div>{selectedDeployment.k8sResources.deploymentName || '-'}</div>
                          </Col>
                          <Col span={12}>
                            <div className="text-gray-500 text-sm">Service</div>
                            <div>{selectedDeployment.k8sResources.serviceName || '-'}</div>
                          </Col>
                          <Col span={12}>
                            <div className="text-gray-500 text-sm">Namespace</div>
                            <div>{selectedDeployment.k8sResources.namespace || '-'}</div>
                          </Col>
                          <Col span={12}>
                            <div className="text-gray-500 text-sm">Ingress</div>
                            <div>{selectedDeployment.k8sResources.ingressName || '-'}</div>
                          </Col>
                          {selectedDeployment.k8sResources.externalUrl && (
                            <Col span={24}>
                              <div className="text-gray-500 text-sm">外部访问地址</div>
                              <a href={selectedDeployment.k8sResources.externalUrl} target="_blank" rel="noopener noreferrer">
                                {selectedDeployment.k8sResources.externalUrl}
                              </a>
                            </Col>
                          )}
                        </Row>
                      </Card>
                    ) : (
                      <div className="text-gray-400 text-center py-8">暂无 K8s 资源信息</div>
                    )}
                  </div>
                ),
              },
            ]}
          />
        )}
      </Drawer>
    </div>
  )
}
