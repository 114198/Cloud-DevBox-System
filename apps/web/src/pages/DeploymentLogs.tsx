import React, { useState, useEffect, useRef } from 'react'
import {
  Card,
  Button,
  Tag,
  Space,
  Input,
  Select,
  Switch,
  Empty,
  Spin,
  Tooltip,
  Badge,
  Row,
  Col,
  Statistic,
  message,
} from 'antd'
import {
  ArrowLeftOutlined,
  FileTextOutlined,
  SearchOutlined,
  FilterOutlined,
  ReloadOutlined,
  DownloadOutlined,
  ClearOutlined,
  PauseCircleOutlined,
  PlayCircleOutlined,
  VerticalAlignBottomOutlined,
  InfoCircleOutlined,
  WarningOutlined,
  CloseCircleOutlined,
} from '@ant-design/icons'
import { useParams, useNavigate } from 'react-router-dom'
import { 
  deploymentService, 
  Deployment, 
  DeploymentLog,
} from '@/services/deployment'

// Mock data for development
const mockDeployment: Deployment = {
  id: 'dep-1',
  environmentId: 'env-1',
  userId: 'user-1',
  name: 'react-app',
  version: 'v1704067200',
  phase: 'Running',
  message: 'Deployment successful',
  createdAt: '2026-01-12T10:00:00Z',
  updatedAt: '2026-01-12T10:01:06Z',
}

const mockLogs: DeploymentLog[] = [
  {
    id: 'log-1',
    deploymentId: 'dep-1',
    phase: 'pending',
    level: 'info',
    message: 'Deployment created',
    timestamp: '2026-01-12T10:00:00Z',
    details: 'Deployment ID: dep-1, Version: v1704067200',
  },
  {
    id: 'log-2',
    deploymentId: 'dep-1',
    phase: 'building',
    level: 'info',
    message: 'Starting Docker build',
    timestamp: '2026-01-12T10:00:01Z',
    details: 'Using Dockerfile at ./Dockerfile',
  },
  {
    id: 'log-3',
    deploymentId: 'dep-1',
    phase: 'building',
    level: 'info',
    message: 'Step 1/10: FROM node:18-alpine AS deps',
    timestamp: '2026-01-12T10:00:02Z',
  },
  {
    id: 'log-4',
    deploymentId: 'dep-1',
    phase: 'building',
    level: 'info',
    message: 'Step 2/10: WORKDIR /app',
    timestamp: '2026-01-12T10:00:03Z',
  },
  {
    id: 'log-5',
    deploymentId: 'dep-1',
    phase: 'building',
    level: 'info',
    message: 'Step 3/10: COPY package*.json ./',
    timestamp: '2026-01-12T10:00:04Z',
  },
  {
    id: 'log-6',
    deploymentId: 'dep-1',
    phase: 'building',
    level: 'info',
    message: 'Step 4/10: RUN npm ci --only=production',
    timestamp: '2026-01-12T10:00:05Z',
    details: 'Installing 156 packages...',
  },
  {
    id: 'log-7',
    deploymentId: 'dep-1',
    phase: 'building',
    level: 'warn',
    message: 'npm WARN deprecated package@1.0.0: This package is deprecated',
    timestamp: '2026-01-12T10:00:20Z',
  },
  {
    id: 'log-8',
    deploymentId: 'dep-1',
    phase: 'building',
    level: 'info',
    message: 'Step 5/10: COPY . .',
    timestamp: '2026-01-12T10:00:25Z',
  },
  {
    id: 'log-9',
    deploymentId: 'dep-1',
    phase: 'building',
    level: 'info',
    message: 'Step 6/10: RUN npm run build',
    timestamp: '2026-01-12T10:00:26Z',
  },
  {
    id: 'log-10',
    deploymentId: 'dep-1',
    phase: 'building',
    level: 'info',
    message: 'Build completed successfully',
    timestamp: '2026-01-12T10:00:45Z',
    details: 'Build duration: 45.5s',
  },
  {
    id: 'log-11',
    deploymentId: 'dep-1',
    phase: 'pushing',
    level: 'info',
    message: 'Starting image push',
    timestamp: '2026-01-12T10:00:46Z',
  },
  {
    id: 'log-12',
    deploymentId: 'dep-1',
    phase: 'pushing',
    level: 'info',
    message: 'Image pushed to registry',
    timestamp: '2026-01-12T10:00:58Z',
    details: 'Push duration: 12.3s',
  },
  {
    id: 'log-13',
    deploymentId: 'dep-1',
    phase: 'deploying',
    level: 'info',
    message: 'Starting Kubernetes deployment',
    timestamp: '2026-01-12T10:00:59Z',
  },
  {
    id: 'log-14',
    deploymentId: 'dep-1',
    phase: 'deploying',
    level: 'info',
    message: 'Created Deployment: react-app-dep-1',
    timestamp: '2026-01-12T10:01:00Z',
  },
  {
    id: 'log-15',
    deploymentId: 'dep-1',
    phase: 'deploying',
    level: 'info',
    message: 'Created Service: react-app-dep-1',
    timestamp: '2026-01-12T10:01:02Z',
  },
  {
    id: 'log-16',
    deploymentId: 'dep-1',
    phase: 'deploying',
    level: 'info',
    message: 'Kubernetes deployment successful',
    timestamp: '2026-01-12T10:01:06Z',
    details: 'Deploy duration: 8.2s, Total time: 66.0s',
  },
]

const levelOptions = [
  { label: '全部', value: 'all' },
  { label: '信息', value: 'info' },
  { label: '警告', value: 'warn' },
  { label: '错误', value: 'error' },
]

const phaseOptions = [
  { label: '全部阶段', value: 'all' },
  { label: '等待中', value: 'pending' },
  { label: '构建中', value: 'building' },
  { label: '推送中', value: 'pushing' },
  { label: '部署中', value: 'deploying' },
]

export default function DeploymentLogsPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const [deployment, setDeployment] = useState<Deployment | null>(null)
  const [logs, setLogs] = useState<DeploymentLog[]>([])
  const [isLoading, setIsLoading] = useState(true)
  const [searchText, setSearchText] = useState('')
  const [levelFilter, setLevelFilter] = useState<string>('all')
  const [phaseFilter, setPhaseFilter] = useState<string>('all')
  const [autoScroll, setAutoScroll] = useState(true)
  const [isStreaming, setIsStreaming] = useState(false)
  const logContainerRef = useRef<HTMLDivElement>(null)
  const eventSourceRef = useRef<EventSource | null>(null)

  useEffect(() => {
    loadData()
    return () => {
      // Cleanup EventSource on unmount
      if (eventSourceRef.current) {
        eventSourceRef.current.close()
      }
    }
  }, [id])

  useEffect(() => {
    if (autoScroll && logContainerRef.current) {
      logContainerRef.current.scrollTop = logContainerRef.current.scrollHeight
    }
  }, [logs, autoScroll])

  const loadData = async () => {
    setIsLoading(true)
    try {
      if (id) {
        const [dep, logData] = await Promise.all([
          deploymentService.get(id),
          deploymentService.getLogs(id),
        ])
        setDeployment(dep)
        setLogs(logData)
      }
    } catch {
      console.log('Using mock data')
      setDeployment(mockDeployment)
      setLogs(mockLogs)
    } finally {
      setIsLoading(false)
    }
  }

  const handleStartStreaming = () => {
    if (!id) return
    
    setIsStreaming(true)
    // Note: In production, this would connect to the SSE endpoint
    // const url = deploymentService.getLogsStreamUrl(id)
    
    // For now, we simulate streaming with mock data
    const simulateStreaming = () => {
      const newLog: DeploymentLog = {
        id: `log-stream-${Date.now()}`,
        deploymentId: id,
        phase: 'deploying',
        level: 'info',
        message: `Streaming log entry at ${new Date().toISOString()}`,
        timestamp: new Date().toISOString(),
      }
      setLogs((prev: DeploymentLog[]) => [...prev, newLog])
    }

    // Simulate streaming every 2 seconds
    const interval = setInterval(simulateStreaming, 2000)
    
    // Store cleanup function
    eventSourceRef.current = {
      close: () => {
        clearInterval(interval)
        setIsStreaming(false)
      },
    } as unknown as EventSource
  }

  const handleStopStreaming = () => {
    if (eventSourceRef.current) {
      eventSourceRef.current.close()
      eventSourceRef.current = null
    }
    setIsStreaming(false)
  }

  const handleRefresh = async () => {
    await loadData()
    message.success('日志已刷新')
  }

  const handleClearLogs = () => {
    setLogs([])
    message.success('日志已清空')
  }

  const handleDownloadLogs = () => {
    const logText = filteredLogs
      .map((log: DeploymentLog) => `[${log.timestamp}] [${log.level.toUpperCase()}] [${log.phase}] ${log.message}${log.details ? `\n  ${log.details}` : ''}`)
      .join('\n')
    
    const blob = new Blob([logText], { type: 'text/plain' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `deployment-${id}-logs.txt`
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
    URL.revokeObjectURL(url)
    message.success('日志已下载')
  }

  const scrollToBottom = () => {
    if (logContainerRef.current) {
      logContainerRef.current.scrollTop = logContainerRef.current.scrollHeight
    }
  }

  const filteredLogs = logs.filter((log: DeploymentLog) => {
    const matchesSearch = 
      log.message.toLowerCase().includes(searchText.toLowerCase()) ||
      log.details?.toLowerCase().includes(searchText.toLowerCase())
    const matchesLevel = levelFilter === 'all' || log.level === levelFilter
    const matchesPhase = phaseFilter === 'all' || log.phase === phaseFilter
    return matchesSearch && matchesLevel && matchesPhase
  })

  const stats = {
    total: logs.length,
    info: logs.filter((l: DeploymentLog) => l.level === 'info').length,
    warn: logs.filter((l: DeploymentLog) => l.level === 'warn').length,
    error: logs.filter((l: DeploymentLog) => l.level === 'error').length,
  }

  const getLevelTag = (level: string) => {
    const config: Record<string, { color: string; text: string }> = {
      info: { color: 'blue', text: 'INFO' },
      warn: { color: 'orange', text: 'WARN' },
      error: { color: 'red', text: 'ERROR' },
    }
    const { color, text } = config[level] || { color: 'default', text: level.toUpperCase() }
    return <Tag color={color}>{text}</Tag>
  }

  const getPhaseTag = (phase: string) => {
    const config: Record<string, { color: string; text: string }> = {
      pending: { color: 'default', text: '等待' },
      building: { color: 'processing', text: '构建' },
      pushing: { color: 'cyan', text: '推送' },
      deploying: { color: 'purple', text: '部署' },
    }
    const { color, text } = config[phase] || { color: 'default', text: phase }
    return <Tag color={color}>{text}</Tag>
  }

  const formatTimestamp = (timestamp: string) => {
    const date = new Date(timestamp)
    const hours = date.getHours().toString().padStart(2, '0')
    const minutes = date.getMinutes().toString().padStart(2, '0')
    const seconds = date.getSeconds().toString().padStart(2, '0')
    const ms = date.getMilliseconds().toString().padStart(3, '0')
    return `${hours}:${minutes}:${seconds}.${ms}`
  }

  if (isLoading) {
    return (
      <div className="flex justify-center items-center h-64">
        <Spin size="large" />
      </div>
    )
  }

  return (
    <div>
      <div className="flex items-center gap-4 mb-6">
        <Button icon={<ArrowLeftOutlined />} onClick={() => navigate('/deployments')}>
          返回
        </Button>
        <div>
          <h1 className="text-2xl font-semibold flex items-center gap-2">
            <FileTextOutlined />
            部署日志 - {deployment?.name}
          </h1>
          <div className="text-gray-500 text-sm">
            版本: {deployment?.version}
          </div>
        </div>
      </div>

      <Row gutter={16} className="mb-4">
        <Col span={6}>
          <Card size="small">
            <Statistic title="总日志数" value={stats.total} prefix={<FileTextOutlined />} />
          </Card>
        </Col>
        <Col span={6}>
          <Card size="small">
            <Statistic
              title="信息"
              value={stats.info}
              valueStyle={{ color: '#1890ff' }}
              prefix={<InfoCircleOutlined />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card size="small">
            <Statistic
              title="警告"
              value={stats.warn}
              valueStyle={{ color: '#faad14' }}
              prefix={<WarningOutlined />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card size="small">
            <Statistic
              title="错误"
              value={stats.error}
              valueStyle={{ color: '#ff4d4f' }}
              prefix={<CloseCircleOutlined />}
            />
          </Card>
        </Col>
      </Row>

      <Card>
        <div className="flex justify-between items-center mb-4">
          <Space>
            <Input
              placeholder="搜索日志内容..."
              prefix={<SearchOutlined />}
              value={searchText}
              onChange={(e: React.ChangeEvent<HTMLInputElement>) => setSearchText(e.target.value)}
              style={{ width: 250 }}
              allowClear
            />
            <Select
              value={levelFilter}
              onChange={setLevelFilter}
              options={levelOptions}
              style={{ width: 100 }}
              suffixIcon={<FilterOutlined />}
            />
            <Select
              value={phaseFilter}
              onChange={setPhaseFilter}
              options={phaseOptions}
              style={{ width: 120 }}
            />
          </Space>
          <Space>
            <Tooltip title="自动滚动">
              <Switch
                checked={autoScroll}
                onChange={setAutoScroll}
                checkedChildren="自动滚动"
                unCheckedChildren="手动滚动"
              />
            </Tooltip>
            {isStreaming ? (
              <Button icon={<PauseCircleOutlined />} onClick={handleStopStreaming}>
                停止
              </Button>
            ) : (
              <Button icon={<PlayCircleOutlined />} onClick={handleStartStreaming}>
                实时
              </Button>
            )}
            <Tooltip title="滚动到底部">
              <Button icon={<VerticalAlignBottomOutlined />} onClick={scrollToBottom} />
            </Tooltip>
            <Tooltip title="刷新">
              <Button icon={<ReloadOutlined />} onClick={handleRefresh} />
            </Tooltip>
            <Tooltip title="下载日志">
              <Button icon={<DownloadOutlined />} onClick={handleDownloadLogs} />
            </Tooltip>
            <Tooltip title="清空日志">
              <Button icon={<ClearOutlined />} onClick={handleClearLogs} danger />
            </Tooltip>
          </Space>
        </div>

        {/* Log Container */}
        <div
          ref={logContainerRef}
          className="bg-gray-900 rounded-lg p-4 font-mono text-sm overflow-auto"
          style={{ height: 'calc(100vh - 400px)', minHeight: '400px' }}
        >
          {filteredLogs.length > 0 ? (
            filteredLogs.map((log: DeploymentLog) => (
              <div
                key={log.id}
                className={`py-1 px-2 hover:bg-gray-800 rounded flex items-start gap-2 ${
                  log.level === 'error' ? 'bg-red-900/20' : 
                  log.level === 'warn' ? 'bg-yellow-900/20' : ''
                }`}
              >
                <span className="text-gray-500 whitespace-nowrap">
                  {formatTimestamp(log.timestamp)}
                </span>
                <span className="w-12">{getLevelTag(log.level)}</span>
                <span className="w-16">{getPhaseTag(log.phase)}</span>
                <div className="flex-1">
                  <span className={`${
                    log.level === 'error' ? 'text-red-400' : 
                    log.level === 'warn' ? 'text-yellow-400' : 'text-gray-200'
                  }`}>
                    {log.message}
                  </span>
                  {log.details && (
                    <div className="text-gray-500 text-xs mt-1 pl-2 border-l-2 border-gray-700">
                      {log.details}
                    </div>
                  )}
                </div>
              </div>
            ))
          ) : (
            <div className="flex justify-center items-center h-full">
              <Empty
                description={<span className="text-gray-500">暂无日志</span>}
                image={Empty.PRESENTED_IMAGE_SIMPLE}
              />
            </div>
          )}
          
          {isStreaming && (
            <div className="py-2 px-2 flex items-center gap-2 text-green-400">
              <Badge status="processing" />
              <span>正在接收实时日志...</span>
            </div>
          )}
        </div>

        <div className="mt-4 flex justify-between items-center text-gray-500 text-sm">
          <span>
            显示 {filteredLogs.length} / {logs.length} 条日志
          </span>
          <span>
            {isStreaming && (
              <Badge status="processing" text="实时模式" />
            )}
          </span>
        </div>
      </Card>
    </div>
  )
}
