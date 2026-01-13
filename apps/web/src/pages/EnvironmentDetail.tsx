import { useState, useEffect, useRef } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import {
  Card,
  Row,
  Col,
  Descriptions,
  Tag,
  Button,
  Space,
  Tabs,
  Progress,
  Spin,
  message,
  Tooltip,
  Drawer,
  Empty,
  Alert,
} from 'antd'
import {
  PlayCircleOutlined,
  PauseCircleOutlined,
  ReloadOutlined,
  DeleteOutlined,
  SettingOutlined,
  LinkOutlined,
  CodeOutlined,
  ArrowLeftOutlined,
  CloudServerOutlined,
  ClockCircleOutlined,
  ExclamationCircleOutlined,
  SyncOutlined,
  LineChartOutlined,
  BellOutlined,
  TeamOutlined,
} from '@ant-design/icons'
import { environmentService, ResourceUsage } from '@/services/environment'
import { useEnvironmentStore, Environment, EnvironmentStatus } from '@/stores/environment'
import ConnectionInfoPanel from '@/components/ConnectionInfoPanel'
import IDEConnectionGuide from '@/components/IDEConnectionGuide'
import WebTerminal from '@/components/WebTerminal'
import MonitoringCharts from '@/components/MonitoringCharts'
import AlertPanel from '@/components/AlertPanel'

// Mock environment data
const mockEnvironment: Environment = {
  id: '1a2b3c4d-5e6f-7890-abcd-ef1234567890',
  name: 'react-app',
  description: 'React 前端项目 - 使用 React 18 和 TypeScript 构建的现代化 Web 应用',
  templateId: 't1',
  templateName: 'React 18',
  status: 'running',
  resources: { cpu: '2 核', memory: '4 GB', storage: '20 GB' },
  createdAt: '2026-01-10T10:00:00Z',
  lastAccessedAt: '2026-01-12T08:30:00Z',
}

// Mock resource usage data
const mockResourceUsage: ResourceUsage = {
  cpu: 45,
  memory: 62,
  storage: 38,
  network: 25,
  timestamp: new Date().toISOString(),
}

// Mock logs
const mockLogs = [
  '[2026-01-12 10:30:15] INFO: Server started on port 3000',
  '[2026-01-12 10:30:16] INFO: Connected to database',
  '[2026-01-12 10:30:17] INFO: Loading configuration...',
  '[2026-01-12 10:30:18] INFO: Configuration loaded successfully',
  '[2026-01-12 10:30:20] INFO: Application ready',
  '[2026-01-12 10:31:05] INFO: GET /api/users - 200 OK (15ms)',
  '[2026-01-12 10:31:10] INFO: POST /api/auth/login - 200 OK (45ms)',
  '[2026-01-12 10:32:00] WARN: High memory usage detected: 75%',
  '[2026-01-12 10:32:30] INFO: GET /api/products - 200 OK (22ms)',
  '[2026-01-12 10:33:00] INFO: Garbage collection completed',
  '[2026-01-12 10:33:15] INFO: GET /api/orders - 200 OK (18ms)',
  '[2026-01-12 10:34:00] INFO: Health check passed',
]

export default function EnvironmentDetail() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const [environment, setEnvironment] = useState<Environment | null>(null)
  const [resourceUsage, setResourceUsage] = useState<ResourceUsage>(mockResourceUsage)
  const [logs, setLogs] = useState<string[]>(mockLogs)
  const [isLoading, setIsLoading] = useState(true)
  const [isTerminalOpen, setIsTerminalOpen] = useState(false)
  const [isConnectionOpen, setIsConnectionOpen] = useState(false)
  const [isRefreshingLogs, setIsRefreshingLogs] = useState(false)
  const [autoRefreshLogs, setAutoRefreshLogs] = useState(true)
  const logsEndRef = useRef<HTMLDivElement>(null)
  const { updateEnvironment, removeEnvironment } = useEnvironmentStore()

  useEffect(() => {
    loadEnvironment()
    
    // Auto-refresh resource usage every 10 seconds
    const resourceInterval = setInterval(() => {
      loadResourceUsage()
    }, 10000)

    return () => clearInterval(resourceInterval)
  }, [id])

  useEffect(() => {
    // Auto-refresh logs every 5 seconds if enabled
    if (!autoRefreshLogs) return
    
    const logsInterval = setInterval(() => {
      loadLogs(true)
    }, 5000)

    return () => clearInterval(logsInterval)
  }, [autoRefreshLogs, id])

  useEffect(() => {
    // Scroll to bottom when new logs arrive
    if (logsEndRef.current) {
      logsEndRef.current.scrollIntoView({ behavior: 'smooth' })
    }
  }, [logs])

  const loadEnvironment = async () => {
    setIsLoading(true)
    try {
      if (id) {
        const env = await environmentService.get(id)
        setEnvironment(env)
      }
    } catch {
      // Use mock data
      setEnvironment(mockEnvironment)
    } finally {
      setIsLoading(false)
    }
  }

  const loadResourceUsage = async () => {
    try {
      if (id && environment?.status === 'running') {
        const usage = await environmentService.getResourceUsage(id)
        setResourceUsage(usage)
      }
    } catch {
      // Simulate resource changes for demo
      setResourceUsage(prev => ({
        ...prev,
        cpu: Math.min(100, Math.max(10, prev.cpu + (Math.random() - 0.5) * 10)),
        memory: Math.min(100, Math.max(20, prev.memory + (Math.random() - 0.5) * 5)),
        network: Math.min(100, Math.max(5, prev.network + (Math.random() - 0.5) * 15)),
        timestamp: new Date().toISOString(),
      }))
    }
  }

  const loadLogs = async (silent = false) => {
    if (!silent) setIsRefreshingLogs(true)
    try {
      if (id) {
        const newLogs = await environmentService.getLogs(id, 100)
        setLogs(newLogs)
      }
    } catch {
      // Add mock log entry for demo
      if (environment?.status === 'running') {
        const timestamp = new Date().toLocaleString('zh-CN').replace(/\//g, '-')
        const newLog = `[${timestamp}] INFO: Health check passed`
        setLogs(prev => [...prev.slice(-99), newLog])
      }
    } finally {
      setIsRefreshingLogs(false)
    }
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

  const handleStart = async () => {
    if (!environment) return
    try {
      await environmentService.start(environment.id)
      setEnvironment({ ...environment, status: 'running' })
      updateEnvironment(environment.id, { status: 'running' })
      message.success('环境已启动')
    } catch {
      message.error('启动失败')
    }
  }

  const handleStop = async () => {
    if (!environment) return
    try {
      await environmentService.stop(environment.id)
      setEnvironment({ ...environment, status: 'stopped' })
      updateEnvironment(environment.id, { status: 'stopped' })
      message.success('环境已停止')
    } catch {
      message.error('停止失败')
    }
  }

  const handleRestart = async () => {
    if (!environment) return
    try {
      await environmentService.restart(environment.id)
      message.success('环境正在重启')
    } catch {
      message.error('重启失败')
    }
  }

  const handleDelete = async () => {
    if (!environment) return
    try {
      await environmentService.delete(environment.id)
      removeEnvironment(environment.id)
      message.success('环境已删除')
      navigate('/environments')
    } catch {
      message.error('删除失败')
    }
  }

  const formatDate = (dateStr?: string) => {
    if (!dateStr) return '-'
    return new Date(dateStr).toLocaleString('zh-CN')
  }

  const getLogLevel = (log: string) => {
    if (log.includes('ERROR')) return 'text-red-500'
    if (log.includes('WARN')) return 'text-yellow-500'
    if (log.includes('DEBUG')) return 'text-gray-400'
    return 'text-green-400'
  }

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-64">
        <Spin size="large" tip="加载中..." />
      </div>
    )
  }

  if (!environment) {
    return (
      <div className="text-center py-12">
        <Empty description="环境不存在" />
        <Button type="primary" onClick={() => navigate('/environments')} className="mt-4">
          返回环境列表
        </Button>
      </div>
    )
  }

  return (
    <div>
      {/* Header */}
      <div className="flex justify-between items-center mb-6">
        <div className="flex items-center gap-4">
          <Button icon={<ArrowLeftOutlined />} onClick={() => navigate('/environments')}>
            返回
          </Button>
          <div>
            <h1 className="text-2xl font-semibold flex items-center gap-2">
              <CloudServerOutlined className="text-blue-500" />
              {environment.name}
              {getStatusTag(environment.status)}
            </h1>
            {environment.description && (
              <p className="text-gray-500 mt-1">{environment.description}</p>
            )}
          </div>
        </div>
        <Space>
          {environment.status === 'running' ? (
            <>
              <Button icon={<PauseCircleOutlined />} onClick={handleStop}>
                停止
              </Button>
              <Button icon={<ReloadOutlined />} onClick={handleRestart}>
                重启
              </Button>
              <Button type="primary" icon={<LinkOutlined />} onClick={() => setIsConnectionOpen(true)}>
                连接
              </Button>
              <Button icon={<TeamOutlined />} onClick={() => navigate(`/environments/${id}/collaborate`)}>
                协作
              </Button>
              <Button icon={<CodeOutlined />} onClick={() => setIsTerminalOpen(true)}>
                终端
              </Button>
            </>
          ) : environment.status === 'stopped' ? (
            <Button type="primary" icon={<PlayCircleOutlined />} onClick={handleStart}>
              启动
            </Button>
          ) : null}
          <Button icon={<SettingOutlined />} onClick={() => navigate(`/environments/${id}/config`)}>
            配置
          </Button>
          <Button danger icon={<DeleteOutlined />} onClick={handleDelete}>
            删除
          </Button>
        </Space>
      </div>

      <Tabs
        defaultActiveKey="overview"
        items={[
          {
            key: 'overview',
            label: '概览',
            children: (
              <Row gutter={16}>
                {/* Basic Info */}
                <Col xs={24} lg={12}>
                  <Card title="基本信息" className="mb-4">
                    <Descriptions column={1} size="small">
                      <Descriptions.Item label="环境 ID">{environment.id}</Descriptions.Item>
                      <Descriptions.Item label="模板">{environment.templateName}</Descriptions.Item>
                      <Descriptions.Item label="状态">{getStatusTag(environment.status)}</Descriptions.Item>
                      <Descriptions.Item label="创建时间">{formatDate(environment.createdAt)}</Descriptions.Item>
                      <Descriptions.Item label="最后访问">{formatDate(environment.lastAccessedAt)}</Descriptions.Item>
                    </Descriptions>
                  </Card>

                  <Card title="资源配置" className="mb-4">
                    <Descriptions column={1} size="small">
                      <Descriptions.Item label="CPU">{environment.resources.cpu}</Descriptions.Item>
                      <Descriptions.Item label="内存">{environment.resources.memory}</Descriptions.Item>
                      <Descriptions.Item label="存储">{environment.resources.storage}</Descriptions.Item>
                    </Descriptions>
                  </Card>
                </Col>

                {/* Resource Usage */}
                <Col xs={24} lg={12}>
                  <Card
                    title="资源使用情况"
                    extra={
                      <Tooltip title="实时更新中">
                        <SyncOutlined spin={environment.status === 'running'} />
                      </Tooltip>
                    }
                    className="mb-4"
                  >
                    {environment.status === 'running' ? (
                      <div className="space-y-6">
                        <div>
                          <div className="flex justify-between mb-2">
                            <span>CPU 使用率</span>
                            <span className="font-medium">{resourceUsage.cpu.toFixed(1)}%</span>
                          </div>
                          <Progress
                            percent={resourceUsage.cpu}
                            strokeColor={resourceUsage.cpu > 80 ? '#ff4d4f' : '#1890ff'}
                            showInfo={false}
                          />
                        </div>
                        <div>
                          <div className="flex justify-between mb-2">
                            <span>内存使用率</span>
                            <span className="font-medium">{resourceUsage.memory.toFixed(1)}%</span>
                          </div>
                          <Progress
                            percent={resourceUsage.memory}
                            strokeColor={resourceUsage.memory > 80 ? '#ff4d4f' : '#52c41a'}
                            showInfo={false}
                          />
                        </div>
                        <div>
                          <div className="flex justify-between mb-2">
                            <span>存储使用率</span>
                            <span className="font-medium">{resourceUsage.storage.toFixed(1)}%</span>
                          </div>
                          <Progress
                            percent={resourceUsage.storage}
                            strokeColor={resourceUsage.storage > 90 ? '#ff4d4f' : '#722ed1'}
                            showInfo={false}
                          />
                        </div>
                        <div>
                          <div className="flex justify-between mb-2">
                            <span>网络 I/O</span>
                            <span className="font-medium">{resourceUsage.network.toFixed(1)} MB/s</span>
                          </div>
                          <Progress
                            percent={resourceUsage.network}
                            strokeColor="#13c2c2"
                            showInfo={false}
                          />
                        </div>
                      </div>
                    ) : (
                      <Alert
                        message="环境未运行"
                        description="启动环境后可查看资源使用情况"
                        type="info"
                        showIcon
                      />
                    )}
                  </Card>

                  {/* Connection Info */}
                  <Card title="连接信息">
                    {environment.status === 'running' ? (
                      <ConnectionInfoPanel
                        environmentId={environment.id}
                        environmentName={environment.name}
                        isRunning={true}
                      />
                    ) : (
                      <Alert
                        message="环境未运行"
                        description="启动环境后可获取连接信息"
                        type="info"
                        showIcon
                      />
                    )}
                  </Card>
                </Col>
              </Row>
            ),
          },
          {
            key: 'logs',
            label: '实时日志',
            children: (
              <Card
                title={
                  <div className="flex items-center gap-2">
                    <span>实时日志</span>
                    {autoRefreshLogs && environment.status === 'running' && (
                      <Tag color="green" icon={<SyncOutlined spin />}>
                        自动刷新
                      </Tag>
                    )}
                  </div>
                }
                extra={
                  <Space>
                    <Button
                      size="small"
                      onClick={() => setAutoRefreshLogs(!autoRefreshLogs)}
                    >
                      {autoRefreshLogs ? '暂停刷新' : '自动刷新'}
                    </Button>
                    <Button
                      size="small"
                      icon={<ReloadOutlined spin={isRefreshingLogs} />}
                      onClick={() => loadLogs()}
                    >
                      刷新
                    </Button>
                  </Space>
                }
              >
                {environment.status === 'running' ? (
                  <div
                    className="bg-gray-900 text-gray-100 p-4 rounded font-mono text-sm overflow-auto"
                    style={{ height: '400px' }}
                  >
                    {logs.map((log, index) => (
                      <div key={index} className={`${getLogLevel(log)} whitespace-pre-wrap`}>
                        {log}
                      </div>
                    ))}
                    <div ref={logsEndRef} />
                  </div>
                ) : (
                  <Alert
                    message="环境未运行"
                    description="启动环境后可查看实时日志"
                    type="info"
                    showIcon
                  />
                )}
              </Card>
            ),
          },
          {
            key: 'connection',
            label: 'IDE 连接',
            children: (
              <Card>
                {environment.status === 'running' ? (
                  <IDEConnectionGuide
                    environmentId={environment.id}
                    environmentName={environment.name}
                  />
                ) : (
                  <Alert
                    message="环境未运行"
                    description="启动环境后可获取 IDE 连接指引"
                    type="info"
                    showIcon
                  />
                )}
              </Card>
            ),
          },
          {
            key: 'monitoring',
            label: (
              <span>
                <LineChartOutlined />
                监控图表
              </span>
            ),
            children: (
              <MonitoringCharts
                environmentId={environment.id}
                isRunning={environment.status === 'running'}
              />
            ),
          },
          {
            key: 'alerts',
            label: (
              <span>
                <BellOutlined />
                告警管理
              </span>
            ),
            children: (
              <AlertPanel
                environmentId={environment.id}
                showRules={true}
              />
            ),
          },
        ]}
      />

      {/* Terminal Drawer */}
      <Drawer
        title={`终端 - ${environment.name}`}
        placement="bottom"
        height="60vh"
        open={isTerminalOpen}
        onClose={() => setIsTerminalOpen(false)}
        styles={{ body: { padding: 0 } }}
      >
        <WebTerminal
          environmentId={environment.id}
          environmentName={environment.name}
          isRunning={environment.status === 'running'}
          onClose={() => setIsTerminalOpen(false)}
        />
      </Drawer>

      {/* Connection Drawer */}
      <Drawer
        title={`连接到 ${environment.name}`}
        placement="right"
        width={600}
        open={isConnectionOpen}
        onClose={() => setIsConnectionOpen(false)}
      >
        <ConnectionInfoPanel
          environmentId={environment.id}
          environmentName={environment.name}
          isRunning={environment.status === 'running'}
        />
      </Drawer>
    </div>
  )
}
