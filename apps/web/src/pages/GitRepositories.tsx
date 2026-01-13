import { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import {
  Card,
  Button,
  Table,
  Tag,
  Space,
  Modal,
  Select,
  Input,
  message,
  Tooltip,
  Empty,
  Spin,
  Avatar,
  Popconfirm,
  Row,
  Col,
  Statistic,
  Badge,
  Tabs,
  Typography,
} from 'antd'
import {
  GithubOutlined,
  GitlabOutlined,
  LinkOutlined,
  DisconnectOutlined,
  SearchOutlined,
  ReloadOutlined,
  LockOutlined,
  UnlockOutlined,
  BranchesOutlined,
  SyncOutlined,
  CheckCircleOutlined,
  CloseCircleOutlined,
  ClockCircleOutlined,
  CloudServerOutlined,
  PlusOutlined,
  EyeOutlined,
} from '@ant-design/icons'
import type { ColumnsType } from 'antd/es/table'
import { useGitStore } from '@/stores/git'
import {
  gitService,
  GitProvider,
  GitConnection,
  GitRepoInfo,
  GitRepository,
} from '@/services/git'

const { Text } = Typography

// Provider configurations
const providerConfig: Record<
  GitProvider,
  { name: string; icon: React.ReactNode; color: string }
> = {
  github: { name: 'GitHub', icon: <GithubOutlined />, color: '#24292e' },
  gitlab: { name: 'GitLab', icon: <GitlabOutlined />, color: '#fc6d26' },
  gitee: { name: 'Gitee', icon: <span style={{ fontWeight: 'bold' }}>G</span>, color: '#c71d23' },
  bitbucket: { name: 'Bitbucket', icon: <span style={{ fontWeight: 'bold' }}>B</span>, color: '#0052cc' },
}

// Mock data for development
const mockConnections: GitConnection[] = [
  {
    id: '1',
    provider: 'github',
    providerUsername: 'developer',
    scopes: ['repo', 'read:user'],
    createdAt: '2026-01-10T10:00:00Z',
    updatedAt: '2026-01-10T10:00:00Z',
  },
]

const mockRemoteRepos: GitRepoInfo[] = [
  {
    id: '1',
    name: 'react-app',
    fullName: 'developer/react-app',
    description: 'A React application',
    cloneUrl: 'https://github.com/developer/react-app.git',
    sshUrl: 'git@github.com:developer/react-app.git',
    defaultBranch: 'main',
    isPrivate: false,
    language: 'TypeScript',
  },
  {
    id: '2',
    name: 'node-api',
    fullName: 'developer/node-api',
    description: 'Node.js REST API',
    cloneUrl: 'https://github.com/developer/node-api.git',
    sshUrl: 'git@github.com:developer/node-api.git',
    defaultBranch: 'main',
    isPrivate: true,
    language: 'JavaScript',
  },
  {
    id: '3',
    name: 'python-ml',
    fullName: 'developer/python-ml',
    description: 'Machine learning project',
    cloneUrl: 'https://github.com/developer/python-ml.git',
    defaultBranch: 'master',
    isPrivate: false,
    language: 'Python',
  },
]

const mockLinkedRepos: GitRepository[] = [
  {
    id: 'lr1',
    userId: 'user1',
    connectionId: '1',
    environmentId: 'env1',
    provider: 'github',
    repoId: '1',
    repoName: 'react-app',
    repoFullName: 'developer/react-app',
    cloneUrl: 'https://github.com/developer/react-app.git',
    defaultBranch: 'main',
    isPrivate: false,
    syncStatus: 'synced',
    lastSyncAt: '2026-01-12T08:30:00Z',
    createdAt: '2026-01-10T10:00:00Z',
    updatedAt: '2026-01-12T08:30:00Z',
  },
]

export default function GitRepositories() {
  const navigate = useNavigate()
  const [activeTab, setActiveTab] = useState('connections')
  const [isLinkModalOpen, setIsLinkModalOpen] = useState(false)
  const [selectedProvider, setSelectedProvider] = useState<GitProvider | null>(null)
  const [searchText, setSearchText] = useState('')
  const [selectedRepoToLink, setSelectedRepoToLink] = useState<GitRepoInfo | null>(null)
  const [selectedEnvironmentId, setSelectedEnvironmentId] = useState<string>('')
  const [isLinking, setIsLinking] = useState(false)

  const {
    connections,
    isLoadingConnections,
    remoteRepositories,
    isLoadingRemoteRepos,
    linkedRepositories,
    isLoadingLinkedRepos,
    setConnections,
    removeConnection,
    setLoadingConnections,
    setRemoteRepositories,
    setLoadingRemoteRepos,
    setLinkedRepositories,
    addLinkedRepository,
    removeLinkedRepository,
    setLoadingLinkedRepos,
  } = useGitStore()

  // Use mock data if no real data
  const displayConnections = connections.length > 0 ? connections : mockConnections
  const displayRemoteRepos = remoteRepositories.length > 0 ? remoteRepositories : mockRemoteRepos
  const displayLinkedRepos = linkedRepositories.length > 0 ? linkedRepositories : mockLinkedRepos

  useEffect(() => {
    loadConnections()
    loadLinkedRepositories()
  }, [])

  useEffect(() => {
    if (selectedProvider) {
      loadRemoteRepositories(selectedProvider)
    }
  }, [selectedProvider])

  const loadConnections = async () => {
    setLoadingConnections(true)
    try {
      const conns = await gitService.getConnections()
      setConnections(conns)
    } catch {
      console.log('Using mock connections')
    } finally {
      setLoadingConnections(false)
    }
  }

  const loadRemoteRepositories = async (provider: GitProvider) => {
    setLoadingRemoteRepos(true)
    try {
      const { repositories, total } = await gitService.listRemoteRepositories({
        provider,
        search: searchText,
      })
      setRemoteRepositories(repositories, total)
    } catch {
      console.log('Using mock repositories')
    } finally {
      setLoadingRemoteRepos(false)
    }
  }

  const loadLinkedRepositories = async () => {
    setLoadingLinkedRepos(true)
    try {
      // Load all linked repositories (no specific environment filter)
      const repos = await gitService.getLinkedRepositories('')
      setLinkedRepositories(repos)
    } catch {
      console.log('Using mock linked repositories')
    } finally {
      setLoadingLinkedRepos(false)
    }
  }

  const handleConnect = async (provider: GitProvider) => {
    try {
      const { url } = await gitService.getAuthUrl(provider)
      // Open OAuth window
      window.location.href = url
    } catch {
      message.error('获取授权链接失败')
    }
  }

  const handleDisconnect = async (provider: GitProvider) => {
    try {
      await gitService.deleteConnection(provider)
      removeConnection(provider)
      message.success(`已断开 ${providerConfig[provider].name} 连接`)
    } catch {
      message.error('断开连接失败')
    }
  }

  const handleLinkRepository = async () => {
    if (!selectedRepoToLink || !selectedEnvironmentId || !selectedProvider) {
      message.warning('请选择仓库和环境')
      return
    }

    setIsLinking(true)
    try {
      const linkedRepo = await gitService.linkRepository({
        provider: selectedProvider,
        repoId: selectedRepoToLink.id,
        environmentId: selectedEnvironmentId,
        enableWebhook: true,
      })
      addLinkedRepository(linkedRepo)
      message.success('仓库关联成功')
      setIsLinkModalOpen(false)
      setSelectedRepoToLink(null)
      setSelectedEnvironmentId('')
    } catch {
      message.error('关联仓库失败')
    } finally {
      setIsLinking(false)
    }
  }

  const handleUnlinkRepository = async (repoId: string) => {
    try {
      await gitService.unlinkRepository(repoId)
      removeLinkedRepository(repoId)
      message.success('已取消关联')
    } catch {
      message.error('取消关联失败')
    }
  }

  const handleSyncRepository = async (repo: GitRepository) => {
    try {
      message.loading({ content: '同步中...', key: 'sync' })
      await gitService.syncRepository({
        repositoryId: repo.id,
        environmentId: repo.environmentId || '',
      })
      message.success({ content: '同步成功', key: 'sync' })
      loadLinkedRepositories()
    } catch {
      message.error({ content: '同步失败', key: 'sync' })
    }
  }

  const getSyncStatusTag = (status: string) => {
    const config: Record<string, { color: string; icon: React.ReactNode; text: string }> = {
      synced: { color: 'success', icon: <CheckCircleOutlined />, text: '已同步' },
      syncing: { color: 'processing', icon: <SyncOutlined spin />, text: '同步中' },
      pending: { color: 'warning', icon: <ClockCircleOutlined />, text: '待同步' },
      failed: { color: 'error', icon: <CloseCircleOutlined />, text: '同步失败' },
    }
    const { color, icon, text } = config[status] || config.pending
    return (
      <Tag color={color} icon={icon}>
        {text}
      </Tag>
    )
  }

  const formatDate = (dateStr: string) => {
    return new Date(dateStr).toLocaleString('zh-CN', {
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
    })
  }

  // Filter remote repos by search
  const filteredRemoteRepos = displayRemoteRepos.filter(
    (repo) =>
      repo.name.toLowerCase().includes(searchText.toLowerCase()) ||
      repo.fullName.toLowerCase().includes(searchText.toLowerCase()) ||
      repo.description?.toLowerCase().includes(searchText.toLowerCase())
  )

  // Connection card component
  const ConnectionCard = ({ provider }: { provider: GitProvider }) => {
    const config = providerConfig[provider]
    const connection = displayConnections.find((c) => c.provider === provider)
    const isConnected = !!connection

    return (
      <Card
        hoverable
        className="text-center"
        style={{ borderColor: isConnected ? config.color : undefined }}
      >
        <div className="mb-4">
          <Avatar
            size={64}
            style={{ backgroundColor: config.color, fontSize: 28 }}
            icon={config.icon}
          />
        </div>
        <h3 className="text-lg font-medium mb-2">{config.name}</h3>
        {isConnected ? (
          <>
            <Tag color="success" className="mb-3">
              <CheckCircleOutlined /> 已连接
            </Tag>
            <div className="text-gray-500 text-sm mb-3">
              @{connection.providerUsername}
            </div>
            <Space>
              <Button
                type="primary"
                size="small"
                onClick={() => {
                  setSelectedProvider(provider)
                  setActiveTab('repositories')
                }}
              >
                查看仓库
              </Button>
              <Popconfirm
                title="确定要断开连接吗？"
                onConfirm={() => handleDisconnect(provider)}
                okText="确定"
                cancelText="取消"
              >
                <Button size="small" danger icon={<DisconnectOutlined />}>
                  断开
                </Button>
              </Popconfirm>
            </Space>
          </>
        ) : (
          <>
            <Tag color="default" className="mb-3">
              未连接
            </Tag>
            <div className="text-gray-500 text-sm mb-3">
              连接后可导入仓库
            </div>
            <Button
              type="primary"
              icon={<LinkOutlined />}
              onClick={() => handleConnect(provider)}
            >
              连接 {config.name}
            </Button>
          </>
        )}
      </Card>
    )
  }

  // Remote repositories table columns
  const remoteRepoColumns: ColumnsType<GitRepoInfo> = [
    {
      title: '仓库',
      key: 'repo',
      render: (_, record) => (
        <div>
          <div className="font-medium flex items-center gap-2">
            {record.isPrivate ? (
              <LockOutlined className="text-yellow-500" />
            ) : (
              <UnlockOutlined className="text-green-500" />
            )}
            {record.fullName}
          </div>
          {record.description && (
            <div className="text-gray-500 text-sm mt-1">{record.description}</div>
          )}
        </div>
      ),
    },
    {
      title: '默认分支',
      dataIndex: 'defaultBranch',
      key: 'defaultBranch',
      width: 120,
      render: (branch) => (
        <Tag icon={<BranchesOutlined />}>{branch}</Tag>
      ),
    },
    {
      title: '语言',
      dataIndex: 'language',
      key: 'language',
      width: 100,
      render: (lang) => lang || '-',
    },
    {
      title: '操作',
      key: 'action',
      width: 120,
      render: (_, record) => {
        const isLinked = displayLinkedRepos.some(
          (lr) => lr.repoId === record.id || lr.repoFullName === record.fullName
        )
        return isLinked ? (
          <Tag color="blue">已关联</Tag>
        ) : (
          <Button
            type="primary"
            size="small"
            icon={<PlusOutlined />}
            onClick={() => {
              setSelectedRepoToLink(record)
              setIsLinkModalOpen(true)
            }}
          >
            关联
          </Button>
        )
      },
    },
  ]

  // Linked repositories table columns
  const linkedRepoColumns: ColumnsType<GitRepository> = [
    {
      title: '仓库',
      key: 'repo',
      render: (_, record) => (
        <div>
          <div className="font-medium flex items-center gap-2">
            <Avatar
              size="small"
              style={{ backgroundColor: providerConfig[record.provider].color }}
              icon={providerConfig[record.provider].icon}
            />
            {record.repoFullName}
            {record.isPrivate && <LockOutlined className="text-yellow-500" />}
          </div>
        </div>
      ),
    },
    {
      title: '分支',
      dataIndex: 'defaultBranch',
      key: 'defaultBranch',
      width: 100,
      render: (branch) => <Tag icon={<BranchesOutlined />}>{branch}</Tag>,
    },
    {
      title: '同步状态',
      dataIndex: 'syncStatus',
      key: 'syncStatus',
      width: 120,
      render: (status) => getSyncStatusTag(status),
    },
    {
      title: '最后同步',
      dataIndex: 'lastSyncAt',
      key: 'lastSyncAt',
      width: 160,
      render: (date) => (date ? formatDate(date) : '-'),
    },
    {
      title: '操作',
      key: 'action',
      width: 180,
      render: (_, record) => (
        <Space>
          <Tooltip title="查看详情">
            <Button
              icon={<EyeOutlined />}
              size="small"
              type="primary"
              onClick={() => navigate(`/git/${record.id}`)}
            />
          </Tooltip>
          <Tooltip title="同步">
            <Button
              icon={<SyncOutlined />}
              size="small"
              onClick={() => handleSyncRepository(record)}
            />
          </Tooltip>
          <Popconfirm
            title="确定要取消关联吗？"
            onConfirm={() => handleUnlinkRepository(record.id)}
            okText="确定"
            cancelText="取消"
          >
            <Button icon={<DisconnectOutlined />} size="small" danger />
          </Popconfirm>
        </Space>
      ),
    },
  ]

  // Stats
  const stats = {
    totalConnections: displayConnections.length,
    totalLinked: displayLinkedRepos.length,
    synced: displayLinkedRepos.filter((r) => r.syncStatus === 'synced').length,
    pending: displayLinkedRepos.filter((r) => r.syncStatus !== 'synced').length,
  }

  return (
    <div>
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-2xl font-semibold">Git 仓库管理</h1>
        <Button
          type="primary"
          icon={<ReloadOutlined />}
          onClick={() => {
            loadConnections()
            loadLinkedRepositories()
            if (selectedProvider) loadRemoteRepositories(selectedProvider)
          }}
        >
          刷新
        </Button>
      </div>

      <Row gutter={16} className="mb-4">
        <Col span={6}>
          <Card size="small">
            <Statistic
              title="已连接平台"
              value={stats.totalConnections}
              prefix={<LinkOutlined />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card size="small">
            <Statistic
              title="已关联仓库"
              value={stats.totalLinked}
              prefix={<CloudServerOutlined />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card size="small">
            <Statistic
              title="已同步"
              value={stats.synced}
              valueStyle={{ color: '#52c41a' }}
              prefix={<CheckCircleOutlined />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card size="small">
            <Statistic
              title="待同步"
              value={stats.pending}
              valueStyle={{ color: '#faad14' }}
              prefix={<ClockCircleOutlined />}
            />
          </Card>
        </Col>
      </Row>

      <Card>
        <Tabs
          activeKey={activeTab}
          onChange={setActiveTab}
          items={[
            {
              key: 'connections',
              label: '平台连接',
              children: (
                <Spin spinning={isLoadingConnections}>
                  <Row gutter={[16, 16]}>
                    {(Object.keys(providerConfig) as GitProvider[]).map((provider) => (
                      <Col xs={24} sm={12} md={6} key={provider}>
                        <ConnectionCard provider={provider} />
                      </Col>
                    ))}
                  </Row>
                </Spin>
              ),
            },
            {
              key: 'repositories',
              label: '远程仓库',
              children: (
                <div>
                  <div className="flex justify-between items-center mb-4">
                    <Space>
                      <Select
                        placeholder="选择平台"
                        value={selectedProvider}
                        onChange={setSelectedProvider}
                        style={{ width: 150 }}
                        options={displayConnections.map((c) => ({
                          value: c.provider,
                          label: (
                            <span>
                              {providerConfig[c.provider].icon}{' '}
                              {providerConfig[c.provider].name}
                            </span>
                          ),
                        }))}
                      />
                      <Input
                        placeholder="搜索仓库..."
                        prefix={<SearchOutlined />}
                        value={searchText}
                        onChange={(e) => setSearchText(e.target.value)}
                        style={{ width: 250 }}
                        allowClear
                        onPressEnter={() =>
                          selectedProvider && loadRemoteRepositories(selectedProvider)
                        }
                      />
                    </Space>
                    <Button
                      icon={<ReloadOutlined />}
                      onClick={() =>
                        selectedProvider && loadRemoteRepositories(selectedProvider)
                      }
                      disabled={!selectedProvider}
                    >
                      刷新
                    </Button>
                  </div>
                  {selectedProvider ? (
                    <Table
                      columns={remoteRepoColumns}
                      dataSource={filteredRemoteRepos}
                      rowKey="id"
                      loading={isLoadingRemoteRepos}
                      pagination={{
                        showSizeChanger: true,
                        showTotal: (total) => `共 ${total} 个仓库`,
                      }}
                    />
                  ) : (
                    <Empty description="请先选择一个已连接的 Git 平台" />
                  )}
                </div>
              ),
            },
            {
              key: 'linked',
              label: (
                <Badge count={stats.totalLinked} offset={[10, 0]}>
                  已关联仓库
                </Badge>
              ),
              children: (
                <Table
                  columns={linkedRepoColumns}
                  dataSource={displayLinkedRepos}
                  rowKey="id"
                  loading={isLoadingLinkedRepos}
                  pagination={{
                    showSizeChanger: true,
                    showTotal: (total) => `共 ${total} 个仓库`,
                  }}
                  locale={{
                    emptyText: <Empty description="暂无关联仓库" />,
                  }}
                />
              ),
            },
          ]}
        />
      </Card>

      {/* Link Repository Modal */}
      <Modal
        title="关联仓库到环境"
        open={isLinkModalOpen}
        onOk={handleLinkRepository}
        onCancel={() => {
          setIsLinkModalOpen(false)
          setSelectedRepoToLink(null)
          setSelectedEnvironmentId('')
        }}
        okText="关联"
        cancelText="取消"
        confirmLoading={isLinking}
      >
        {selectedRepoToLink && (
          <div className="mb-4">
            <Text strong>选中仓库：</Text>
            <div className="mt-2 p-3 bg-gray-50 rounded">
              <div className="font-medium">{selectedRepoToLink.fullName}</div>
              {selectedRepoToLink.description && (
                <div className="text-gray-500 text-sm mt-1">
                  {selectedRepoToLink.description}
                </div>
              )}
            </div>
          </div>
        )}
        <div>
          <Text strong>选择环境：</Text>
          <Select
            placeholder="选择要关联的开发环境"
            value={selectedEnvironmentId}
            onChange={setSelectedEnvironmentId}
            style={{ width: '100%', marginTop: 8 }}
            options={[
              { value: 'env1', label: 'react-app (运行中)' },
              { value: 'env2', label: 'node-api (已停止)' },
              { value: 'env3', label: 'python-ml (运行中)' },
            ]}
          />
        </div>
      </Modal>
    </div>
  )
}
