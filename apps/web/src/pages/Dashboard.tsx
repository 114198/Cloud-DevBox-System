import { useState, useEffect } from 'react'
import { Card, Row, Col, Statistic, List, Tag, Button, Progress, Spin, Empty, Timeline, Tooltip } from 'antd'
import {
  CloudServerOutlined,
  PlayCircleOutlined,
  PauseCircleOutlined,
  PlusOutlined,
  RocketOutlined,
  CodeOutlined,
  TeamOutlined,
  ClockCircleOutlined,
  ThunderboltOutlined,
  SettingOutlined,
  AppstoreOutlined,
  ReloadOutlined,
  WarningOutlined,
} from '@ant-design/icons'
import { useNavigate } from 'react-router-dom'
import { environmentService, EnvironmentStats, RecentActivity } from '@/services/environment'
import { useEnvironmentStore, Environment } from '@/stores/environment'

// Mock data for development
const mockStats: EnvironmentStats = {
  totalEnvironments: 8,
  runningEnvironments: 3,
  stoppedEnvironments: 4,
  suspendedEnvironments: 1,
  totalProjects: 5,
  totalCpuUsage: 45,
  totalMemoryUsage: 62,
  totalStorageUsage: 38,
}

const mockRecentEnvironments: Environment[] = [
  { id: '1', name: 'react-frontend', description: 'React 前端项目', templateId: 't1', templateName: 'React 18', status: 'running', resources: { cpu: '2 核', memory: '4 GB', storage: '20 GB' }, createdAt: '2026-01-12T10:00:00Z' },
  { id: '2', name: 'node-api-server', description: 'Node.js API 服务', templateId: 't2', templateName: 'Node.js 18', status: 'stopped', resources: { cpu: '1 核', memory: '2 GB', storage: '10 GB' }, createdAt: '2026-01-11T08:00:00Z' },
  { id: '3', name: 'python-ml-project', description: 'Python 机器学习项目', templateId: 't3', templateName: 'Python 3.11', status: 'running', resources: { cpu: '4 核', memory: '8 GB', storage: '50 GB' }, createdAt: '2026-01-10T14:00:00Z' },
  { id: '4', name: 'go-microservice', description: 'Go 微服务', templateId: 't4', templateName: 'Go 1.21', status: 'running', resources: { cpu: '2 核', memory: '4 GB', storage: '15 GB' }, createdAt: '2026-01-09T09:00:00Z' },
]

const mockActivities: RecentActivity[] = [
  { id: 'a1', type: 'environment_started', environmentId: '1', environmentName: 'react-frontend', description: '启动了环境', timestamp: '2026-01-12T10:30:00Z', userId: 'u1', userName: '张三' },
  { id: 'a2', type: 'environment_created', environmentId: '4', environmentName: 'go-microservice', description: '创建了新环境', timestamp: '2026-01-12T09:00:00Z', userId: 'u1', userName: '张三' },
  { id: 'a3', type: 'deployment', environmentId: '1', environmentName: 'react-frontend', description: '部署到生产环境', timestamp: '2026-01-11T16:00:00Z', userId: 'u2', userName: '李四' },
  { id: 'a4', type: 'collaboration', environmentId: '3', environmentName: 'python-ml-project', description: '开始协作编辑', timestamp: '2026-01-11T14:30:00Z', userId: 'u3', userName: '王五' },
  { id: 'a5', type: 'environment_stopped', environmentId: '2', environmentName: 'node-api-server', description: '停止了环境', timestamp: '2026-01-11T12:00:00Z', userId: 'u1', userName: '张三' },
]

export default function Dashboard() {
  const navigate = useNavigate()
  const { environments, setEnvironments } = useEnvironmentStore()
  const [stats, setStats] = useState<EnvironmentStats>(mockStats)
  const [activities, setActivities] = useState<RecentActivity[]>(mockActivities)
  const [isLoading, setIsLoading] = useState(false)
  const [isRefreshing, setIsRefreshing] = useState(false)

  // Use mock data if no real environments
  const recentEnvironments = environments.length > 0 ? environments.slice(0, 4) : mockRecentEnvironments

  useEffect(() => {
    loadDashboardData()
  }, [])

  const loadDashboardData = async () => {
    setIsLoading(true)
    try {
      // Try to load real data, fall back to mock data
      const [envList] = await Promise.allSettled([
        environmentService.list(),
        environmentService.getStats(),
        environmentService.getRecentActivities(5),
      ])

      if (envList.status === 'fulfilled') {
        setEnvironments(envList.value)
      }
      // Stats and activities would be set similarly if API is available
    } catch (error) {
      console.log('Using mock data for dashboard')
    } finally {
      setIsLoading(false)
    }
  }

  const handleRefresh = async () => {
    setIsRefreshing(true)
    await loadDashboardData()
    setIsRefreshing(false)
  }

  const getStatusTag = (status: string) => {
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

  const getActivityIcon = (type: string) => {
    const icons: Record<string, React.ReactNode> = {
      environment_created: <PlusOutlined style={{ color: '#52c41a' }} />,
      environment_started: <PlayCircleOutlined style={{ color: '#1890ff' }} />,
      environment_stopped: <PauseCircleOutlined style={{ color: '#8c8c8c' }} />,
      environment_deleted: <WarningOutlined style={{ color: '#ff4d4f' }} />,
      deployment: <RocketOutlined style={{ color: '#722ed1' }} />,
      collaboration: <TeamOutlined style={{ color: '#13c2c2' }} />,
    }
    return icons[type] || <ClockCircleOutlined />
  }

  const formatTime = (timestamp: string) => {
    const date = new Date(timestamp)
    const now = new Date()
    const diff = now.getTime() - date.getTime()
    const minutes = Math.floor(diff / 60000)
    const hours = Math.floor(diff / 3600000)
    const days = Math.floor(diff / 86400000)

    if (minutes < 60) return `${minutes} 分钟前`
    if (hours < 24) return `${hours} 小时前`
    if (days < 7) return `${days} 天前`
    return date.toLocaleDateString('zh-CN')
  }

  // Quick action buttons
  const quickActions = [
    { icon: <PlusOutlined />, title: '创建环境', description: '从模板快速创建', onClick: () => navigate('/templates'), color: '#1890ff' },
    { icon: <CodeOutlined />, title: '我的环境', description: '管理开发环境', onClick: () => navigate('/environments'), color: '#52c41a' },
    { icon: <AppstoreOutlined />, title: '浏览模板', description: '查看所有模板', onClick: () => navigate('/templates'), color: '#722ed1' },
    { icon: <SettingOutlined />, title: '系统设置', description: '个人偏好设置', onClick: () => navigate('/settings'), color: '#fa8c16' },
  ]

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-64">
        <Spin size="large" tip="加载中..." />
      </div>
    )
  }

  return (
    <div>
      {/* Header */}
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-2xl font-semibold">仪表盘</h1>
        <div className="flex gap-2">
          <Button
            icon={<ReloadOutlined spin={isRefreshing} />}
            onClick={handleRefresh}
          >
            刷新
          </Button>
          <Button
            type="primary"
            icon={<PlusOutlined />}
            onClick={() => navigate('/templates')}
          >
            创建环境
          </Button>
        </div>
      </div>

      {/* Statistics Cards */}
      <Row gutter={16} className="mb-6">
        <Col xs={24} sm={12} lg={6}>
          <Card hoverable onClick={() => navigate('/environments')}>
            <Statistic
              title="总环境数"
              value={stats.totalEnvironments}
              prefix={<CloudServerOutlined />}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card hoverable onClick={() => navigate('/environments?status=running')}>
            <Statistic
              title="运行中"
              value={stats.runningEnvironments}
              prefix={<PlayCircleOutlined />}
              valueStyle={{ color: '#52c41a' }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card hoverable onClick={() => navigate('/environments?status=stopped')}>
            <Statistic
              title="已停止"
              value={stats.stoppedEnvironments}
              prefix={<PauseCircleOutlined />}
              valueStyle={{ color: '#8c8c8c' }}
            />
          </Card>
        </Col>
        <Col xs={24} sm={12} lg={6}>
          <Card hoverable onClick={() => navigate('/projects')}>
            <Statistic
              title="项目数"
              value={stats.totalProjects}
              prefix={<ThunderboltOutlined />}
              valueStyle={{ color: '#1890ff' }}
            />
          </Card>
        </Col>
      </Row>

      {/* Quick Actions */}
      <Card title="快速操作" className="mb-6">
        <Row gutter={16}>
          {quickActions.map((action, index) => (
            <Col xs={12} sm={6} key={index}>
              <Card
                hoverable
                className="text-center"
                onClick={action.onClick}
                styles={{ body: { padding: '16px' } }}
              >
                <div
                  className="text-3xl mb-2"
                  style={{ color: action.color }}
                >
                  {action.icon}
                </div>
                <div className="font-medium">{action.title}</div>
                <div className="text-gray-500 text-sm">{action.description}</div>
              </Card>
            </Col>
          ))}
        </Row>
      </Card>

      <Row gutter={16}>
        {/* Recent Environments */}
        <Col xs={24} lg={14}>
          <Card
            title="最近环境"
            extra={
              <Button type="link" onClick={() => navigate('/environments')}>
                查看全部
              </Button>
            }
            className="mb-6"
          >
            {recentEnvironments.length > 0 ? (
              <List
                dataSource={recentEnvironments}
                renderItem={(item) => (
                  <List.Item
                    actions={[
                      item.status === 'running' ? (
                        <Button
                          type="primary"
                          size="small"
                          icon={<CodeOutlined />}
                          onClick={() => navigate(`/environments?id=${item.id}`)}
                        >
                          连接
                        </Button>
                      ) : (
                        <Button
                          size="small"
                          icon={<PlayCircleOutlined />}
                          onClick={() => navigate(`/environments?id=${item.id}`)}
                        >
                          启动
                        </Button>
                      ),
                    ]}
                  >
                    <List.Item.Meta
                      avatar={
                        <div className="w-10 h-10 rounded-lg bg-blue-50 flex items-center justify-center">
                          <CloudServerOutlined className="text-blue-500 text-lg" />
                        </div>
                      }
                      title={
                        <div className="flex items-center gap-2">
                          <span>{item.name}</span>
                          {getStatusTag(item.status)}
                        </div>
                      }
                      description={
                        <div className="text-gray-500">
                          <span>{item.templateName}</span>
                          <span className="mx-2">•</span>
                          <span>{item.resources.cpu} / {item.resources.memory}</span>
                        </div>
                      }
                    />
                  </List.Item>
                )}
              />
            ) : (
              <Empty description="暂无环境" />
            )}
          </Card>
        </Col>

        {/* Recent Activities & Resource Usage */}
        <Col xs={24} lg={10}>
          {/* Resource Usage */}
          <Card title="资源使用概览" className="mb-6">
            <div className="space-y-4">
              <div>
                <div className="flex justify-between mb-1">
                  <span>CPU 使用率</span>
                  <span>{stats.totalCpuUsage}%</span>
                </div>
                <Progress
                  percent={stats.totalCpuUsage}
                  strokeColor="#1890ff"
                  showInfo={false}
                />
              </div>
              <div>
                <div className="flex justify-between mb-1">
                  <span>内存使用率</span>
                  <span>{stats.totalMemoryUsage}%</span>
                </div>
                <Progress
                  percent={stats.totalMemoryUsage}
                  strokeColor="#52c41a"
                  showInfo={false}
                />
              </div>
              <div>
                <div className="flex justify-between mb-1">
                  <span>存储使用率</span>
                  <span>{stats.totalStorageUsage}%</span>
                </div>
                <Progress
                  percent={stats.totalStorageUsage}
                  strokeColor="#722ed1"
                  showInfo={false}
                />
              </div>
            </div>
          </Card>

          {/* Recent Activities */}
          <Card
            title="最近活动"
            extra={
              <Tooltip title="显示最近 5 条活动">
                <ClockCircleOutlined />
              </Tooltip>
            }
          >
            <Timeline
              items={activities.map((activity) => ({
                dot: getActivityIcon(activity.type),
                children: (
                  <div>
                    <div className="font-medium">
                      {activity.userName} {activity.description}
                    </div>
                    <div className="text-gray-500 text-sm">
                      {activity.environmentName && (
                        <span className="text-blue-500">{activity.environmentName}</span>
                      )}
                      <span className="ml-2">{formatTime(activity.timestamp)}</span>
                    </div>
                  </div>
                ),
              }))}
            />
          </Card>
        </Col>
      </Row>
    </div>
  )
}
