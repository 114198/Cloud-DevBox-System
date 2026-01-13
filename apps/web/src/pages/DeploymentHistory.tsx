import { useState, useEffect } from 'react'
import {
  Card,
  Table,
  Button,
  Tag,
  Space,
  Modal,
  Timeline,
  Descriptions,
  message,
  Tooltip,
  Empty,
  Spin,
  Input,
  Row,
  Col,
  Statistic,
} from 'antd'
import {
  RollbackOutlined,
  CheckCircleOutlined,
  CloseCircleOutlined,
  ClockCircleOutlined,
  ArrowLeftOutlined,
  HistoryOutlined,
  RocketOutlined,
  ExclamationCircleOutlined,
  SearchOutlined,
} from '@ant-design/icons'
import type { ColumnsType } from 'antd/es/table'
import { useParams, useNavigate } from 'react-router-dom'
import { 
  deploymentService, 
  Deployment, 
  DeploymentVersion, 
  DeploymentHistory,
  DeploymentPhase,
} from '@/services/deployment'
import { getPhaseInfo, formatDuration } from '@/stores/deployment'

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

const mockHistory: DeploymentHistory = {
  deploymentId: 'dep-1',
  versions: [
    {
      id: 'ver-1',
      deploymentId: 'dep-1',
      version: 'v1704067200',
      phase: 'Running',
      message: 'Deployment successful',
      imageInfo: {
        name: 'registry.devbox.io/user-1/react-app',
        tag: 'v1704067200',
        digest: 'sha256:abc123',
        buildDuration: 45.5,
        createdAt: '2026-01-12T10:00:00Z',
      },
      deployConfig: {
        replicas: 2,
        resources: { cpu: '500m', memory: '512Mi' },
      },
      createdAt: '2026-01-12T10:00:00Z',
      deployedAt: '2026-01-12T10:01:06Z',
      isActive: true,
    },
    {
      id: 'ver-2',
      deploymentId: 'dep-1',
      version: 'v1703980800',
      phase: 'RolledBack',
      message: 'Rolled back due to high error rate',
      imageInfo: {
        name: 'registry.devbox.io/user-1/react-app',
        tag: 'v1703980800',
        digest: 'sha256:def456',
        buildDuration: 42.3,
        createdAt: '2026-01-11T10:00:00Z',
      },
      deployConfig: {
        replicas: 2,
        resources: { cpu: '500m', memory: '512Mi' },
      },
      createdAt: '2026-01-11T10:00:00Z',
      deployedAt: '2026-01-11T10:00:50Z',
      rolledBackAt: '2026-01-12T09:30:00Z',
      isActive: false,
    },
    {
      id: 'ver-3',
      deploymentId: 'dep-1',
      version: 'v1703894400',
      phase: 'Running',
      message: 'Deployment successful',
      imageInfo: {
        name: 'registry.devbox.io/user-1/react-app',
        tag: 'v1703894400',
        digest: 'sha256:ghi789',
        buildDuration: 38.7,
        createdAt: '2026-01-10T10:00:00Z',
      },
      deployConfig: {
        replicas: 1,
        resources: { cpu: '250m', memory: '256Mi' },
      },
      createdAt: '2026-01-10T10:00:00Z',
      deployedAt: '2026-01-10T10:00:45Z',
      isActive: false,
    },
    {
      id: 'ver-4',
      deploymentId: 'dep-1',
      version: 'v1703808000',
      phase: 'Failed',
      message: 'Build failed: npm install error',
      createdAt: '2026-01-09T10:00:00Z',
      isActive: false,
    },
  ],
  total: 4,
}

export default function DeploymentHistoryPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const [deployment, setDeployment] = useState<Deployment | null>(null)
  const [history, setHistory] = useState<DeploymentHistory | null>(null)
  const [isLoading, setIsLoading] = useState(true)
  const [searchText, setSearchText] = useState('')
  const [selectedVersion, setSelectedVersion] = useState<DeploymentVersion | null>(null)
  const [isDetailModalOpen, setIsDetailModalOpen] = useState(false)

  useEffect(() => {
    loadData()
  }, [id])

  const loadData = async () => {
    setIsLoading(true)
    try {
      if (id) {
        const [dep, hist] = await Promise.all([
          deploymentService.get(id),
          deploymentService.getHistory(id),
        ])
        setDeployment(dep)
        setHistory(hist)
      }
    } catch {
      console.log('Using mock data')
      setDeployment(mockDeployment)
      setHistory(mockHistory)
    } finally {
      setIsLoading(false)
    }
  }

  const handleRollback = async (version: DeploymentVersion) => {
    Modal.confirm({
      title: '确认回滚',
      icon: <ExclamationCircleOutlined />,
      content: (
        <div>
          <p>确定要回滚到版本 <strong>{version.version}</strong> 吗？</p>
          <p className="text-gray-500 text-sm mt-2">
            回滚操作将在 30 秒内完成，期间服务可能短暂不可用。
          </p>
        </div>
      ),
      okText: '确认回滚',
      cancelText: '取消',
      onOk: async () => {
        try {
          if (id) {
            await deploymentService.rollback(id, {
              targetVersion: version.version,
              reason: '手动回滚到历史版本',
            })
            message.success('回滚成功')
            loadData()
          }
        } catch {
          message.error('回滚失败')
        }
      },
    })
  }

  const handleViewDetail = (version: DeploymentVersion) => {
    setSelectedVersion(version)
    setIsDetailModalOpen(true)
  }

  const formatDate = (dateStr?: string) => {
    if (!dateStr) return '-'
    return new Date(dateStr).toLocaleString('zh-CN', {
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit',
    })
  }

  const getPhaseTag = (phase: DeploymentPhase, isActive: boolean) => {
    const info = getPhaseInfo(phase)
    if (isActive) {
      return <Tag color="green" icon={<CheckCircleOutlined />}>当前版本</Tag>
    }
    return <Tag color={info.color}>{info.text}</Tag>
  }

  const filteredVersions = history?.versions.filter((v) =>
    v.version.toLowerCase().includes(searchText.toLowerCase()) ||
    v.message?.toLowerCase().includes(searchText.toLowerCase())
  ) || []

  const stats = {
    total: history?.total || 0,
    successful: history?.versions.filter((v) => v.phase === 'Running').length || 0,
    failed: history?.versions.filter((v) => v.phase === 'Failed').length || 0,
    rolledBack: history?.versions.filter((v) => v.phase === 'RolledBack').length || 0,
  }

  const columns: ColumnsType<DeploymentVersion> = [
    {
      title: '版本',
      dataIndex: 'version',
      key: 'version',
      render: (text: string, record: DeploymentVersion) => (
        <div>
          <div className="font-mono font-medium">{text}</div>
          {record.imageInfo?.digest && (
            <div className="text-gray-500 text-xs truncate max-w-[150px]" title={record.imageInfo.digest}>
              {record.imageInfo.digest}
            </div>
          )}
        </div>
      ),
    },
    {
      title: '状态',
      key: 'status',
      width: 120,
      render: (_: unknown, record: DeploymentVersion) => getPhaseTag(record.phase, record.isActive),
    },
    {
      title: '消息',
      dataIndex: 'message',
      key: 'message',
      ellipsis: true,
      render: (text: string) => text || '-',
    },
    {
      title: '资源配置',
      key: 'resources',
      width: 150,
      render: (_: unknown, record: DeploymentVersion) => (
        record.deployConfig?.resources ? (
          <div className="text-sm">
            <div>CPU: {record.deployConfig.resources.cpu}</div>
            <div>内存: {record.deployConfig.resources.memory}</div>
          </div>
        ) : <span className="text-gray-400">-</span>
      ),
    },
    {
      title: '构建耗时',
      key: 'buildTime',
      width: 100,
      render: (_: unknown, record: DeploymentVersion) => (
        <span>{formatDuration(record.imageInfo?.buildDuration)}</span>
      ),
    },
    {
      title: '部署时间',
      dataIndex: 'deployedAt',
      key: 'deployedAt',
      width: 170,
      sorter: (a: DeploymentVersion, b: DeploymentVersion) => {
        const aTime = a.deployedAt ? new Date(a.deployedAt).getTime() : 0
        const bTime = b.deployedAt ? new Date(b.deployedAt).getTime() : 0
        return aTime - bTime
      },
      render: (date: string) => formatDate(date),
    },
    {
      title: '操作',
      key: 'action',
      width: 150,
      render: (_: unknown, record: DeploymentVersion) => (
        <Space size="small">
          <Tooltip title="查看详情">
            <Button size="small" onClick={() => handleViewDetail(record)}>
              详情
            </Button>
          </Tooltip>
          {!record.isActive && record.phase !== 'Failed' && (
            <Tooltip title="回滚到此版本">
              <Button
                size="small"
                icon={<RollbackOutlined />}
                onClick={() => handleRollback(record)}
              >
                回滚
              </Button>
            </Tooltip>
          )}
        </Space>
      ),
    },
  ]

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
            <HistoryOutlined />
            部署历史 - {deployment?.name}
          </h1>
          <div className="text-gray-500 text-sm">
            当前版本: {deployment?.version}
          </div>
        </div>
      </div>

      <Row gutter={16} className="mb-4">
        <Col span={6}>
          <Card size="small">
            <Statistic title="总版本数" value={stats.total} prefix={<HistoryOutlined />} />
          </Card>
        </Col>
        <Col span={6}>
          <Card size="small">
            <Statistic
              title="成功部署"
              value={stats.successful}
              valueStyle={{ color: '#52c41a' }}
              prefix={<CheckCircleOutlined />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card size="small">
            <Statistic
              title="失败"
              value={stats.failed}
              valueStyle={{ color: '#ff4d4f' }}
              prefix={<CloseCircleOutlined />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card size="small">
            <Statistic
              title="已回滚"
              value={stats.rolledBack}
              valueStyle={{ color: '#faad14' }}
              prefix={<RollbackOutlined />}
            />
          </Card>
        </Col>
      </Row>

      <Card>
        <div className="flex justify-between items-center mb-4">
          <Input
            placeholder="搜索版本号或消息..."
            prefix={<SearchOutlined />}
            value={searchText}
            onChange={(e) => setSearchText(e.target.value)}
            style={{ width: 300 }}
            allowClear
          />
        </div>

        {filteredVersions.length > 0 ? (
          <Table
            columns={columns}
            dataSource={filteredVersions}
            rowKey="id"
            pagination={{
              showSizeChanger: true,
              showQuickJumper: true,
              showTotal: (total) => `共 ${total} 个版本`,
            }}
          />
        ) : (
          <Empty description="暂无部署历史" />
        )}
      </Card>

      {/* Version Detail Modal */}
      <Modal
        title={`版本详情 - ${selectedVersion?.version}`}
        open={isDetailModalOpen}
        onCancel={() => setIsDetailModalOpen(false)}
        footer={[
          <Button key="close" onClick={() => setIsDetailModalOpen(false)}>
            关闭
          </Button>,
          selectedVersion && !selectedVersion.isActive && selectedVersion.phase !== 'Failed' && (
            <Button
              key="rollback"
              type="primary"
              icon={<RollbackOutlined />}
              onClick={() => {
                setIsDetailModalOpen(false)
                handleRollback(selectedVersion)
              }}
            >
              回滚到此版本
            </Button>
          ),
        ]}
        width={700}
      >
        {selectedVersion && (
          <div>
            <Descriptions bordered column={2} size="small">
              <Descriptions.Item label="版本号" span={1}>
                <span className="font-mono">{selectedVersion.version}</span>
              </Descriptions.Item>
              <Descriptions.Item label="状态" span={1}>
                {getPhaseTag(selectedVersion.phase, selectedVersion.isActive)}
              </Descriptions.Item>
              <Descriptions.Item label="消息" span={2}>
                {selectedVersion.message || '-'}
              </Descriptions.Item>
              <Descriptions.Item label="创建时间" span={1}>
                {formatDate(selectedVersion.createdAt)}
              </Descriptions.Item>
              <Descriptions.Item label="部署时间" span={1}>
                {formatDate(selectedVersion.deployedAt)}
              </Descriptions.Item>
              {selectedVersion.rolledBackAt && (
                <Descriptions.Item label="回滚时间" span={2}>
                  {formatDate(selectedVersion.rolledBackAt)}
                </Descriptions.Item>
              )}
            </Descriptions>

            {selectedVersion.imageInfo && (
              <>
                <h4 className="mt-4 mb-2 font-medium">镜像信息</h4>
                <Descriptions bordered column={2} size="small">
                  <Descriptions.Item label="镜像名称" span={2}>
                    <span className="font-mono text-sm">{selectedVersion.imageInfo.name}</span>
                  </Descriptions.Item>
                  <Descriptions.Item label="标签" span={1}>
                    {selectedVersion.imageInfo.tag}
                  </Descriptions.Item>
                  <Descriptions.Item label="Digest" span={1}>
                    <span className="font-mono text-xs">
                      {selectedVersion.imageInfo.digest?.substring(0, 20)}...
                    </span>
                  </Descriptions.Item>
                  <Descriptions.Item label="构建耗时" span={1}>
                    {formatDuration(selectedVersion.imageInfo.buildDuration)}
                  </Descriptions.Item>
                  <Descriptions.Item label="创建时间" span={1}>
                    {formatDate(selectedVersion.imageInfo.createdAt)}
                  </Descriptions.Item>
                </Descriptions>
              </>
            )}

            {selectedVersion.deployConfig && (
              <>
                <h4 className="mt-4 mb-2 font-medium">部署配置</h4>
                <Descriptions bordered column={2} size="small">
                  <Descriptions.Item label="副本数" span={1}>
                    {selectedVersion.deployConfig.replicas || 1}
                  </Descriptions.Item>
                  <Descriptions.Item label="服务类型" span={1}>
                    {selectedVersion.deployConfig.serviceType || 'ClusterIP'}
                  </Descriptions.Item>
                  {selectedVersion.deployConfig.resources && (
                    <>
                      <Descriptions.Item label="CPU" span={1}>
                        {selectedVersion.deployConfig.resources.cpu}
                      </Descriptions.Item>
                      <Descriptions.Item label="内存" span={1}>
                        {selectedVersion.deployConfig.resources.memory}
                      </Descriptions.Item>
                    </>
                  )}
                </Descriptions>
              </>
            )}

            <h4 className="mt-4 mb-2 font-medium">版本时间线</h4>
            <Timeline
              items={[
                {
                  color: 'blue',
                  children: (
                    <div>
                      <div className="font-medium">创建</div>
                      <div className="text-gray-500 text-sm">{formatDate(selectedVersion.createdAt)}</div>
                    </div>
                  ),
                },
                ...(selectedVersion.deployedAt ? [{
                  color: 'green',
                  children: (
                    <div>
                      <div className="font-medium">部署完成</div>
                      <div className="text-gray-500 text-sm">{formatDate(selectedVersion.deployedAt)}</div>
                    </div>
                  ),
                }] : []),
                ...(selectedVersion.rolledBackAt ? [{
                  color: 'orange',
                  children: (
                    <div>
                      <div className="font-medium">已回滚</div>
                      <div className="text-gray-500 text-sm">{formatDate(selectedVersion.rolledBackAt)}</div>
                    </div>
                  ),
                }] : []),
              ]}
            />
          </div>
        )}
      </Modal>
    </div>
  )
}
