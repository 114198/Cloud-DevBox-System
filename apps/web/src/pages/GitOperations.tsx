import { useState, useEffect } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import {
  Card,
  Button,
  Table,
  Tag,
  Space,
  Select,
  Spin,
  Empty,
  Timeline,
  Tooltip,
  message,
  Drawer,
  Typography,
  Breadcrumb,
  Row,
  Col,
  Statistic,
  Divider,
  Badge,
  Avatar,
} from 'antd'
import {
  BranchesOutlined,
  HistoryOutlined,
  SyncOutlined,
  ArrowLeftOutlined,
  FileTextOutlined,
  PlusOutlined,
  MinusOutlined,
  EditOutlined,
  CheckCircleOutlined,
  ClockCircleOutlined,
  UserOutlined,
  CalendarOutlined,
  CodeOutlined,
  DiffOutlined,
  CopyOutlined,
} from '@ant-design/icons'
import { useGitStore } from '@/stores/git'
import {
  gitService,
  GitBranch,
  GitCommit,
  GitRepository,
  GitProvider,
} from '@/services/git'

const { Text, Paragraph } = Typography

// Mock data for development
const mockBranches: GitBranch[] = [
  { name: 'main', commitSha: 'abc123', isDefault: true, protected: true },
  { name: 'develop', commitSha: 'def456', isDefault: false, protected: false },
  { name: 'feature/user-auth', commitSha: 'ghi789', isDefault: false, protected: false },
  { name: 'feature/dashboard', commitSha: 'jkl012', isDefault: false, protected: false },
  { name: 'hotfix/login-bug', commitSha: 'mno345', isDefault: false, protected: false },
]

const mockCommits: GitCommit[] = [
  {
    sha: 'abc123def456789012345678901234567890abcd',
    message: 'feat: Add user authentication module\n\nImplemented JWT-based authentication with refresh tokens.',
    author: 'John Doe',
    email: 'john@example.com',
    timestamp: '2026-01-12T10:30:00Z',
  },
  {
    sha: 'bcd234ef5678901234567890123456789012bcde',
    message: 'fix: Resolve login redirect issue',
    author: 'Jane Smith',
    email: 'jane@example.com',
    timestamp: '2026-01-12T09:15:00Z',
  },
  {
    sha: 'cde345fg6789012345678901234567890123cdef',
    message: 'docs: Update README with setup instructions',
    author: 'John Doe',
    email: 'john@example.com',
    timestamp: '2026-01-11T16:45:00Z',
  },
  {
    sha: 'def456gh7890123456789012345678901234defg',
    message: 'refactor: Improve database connection handling\n\nAdded connection pooling and retry logic.',
    author: 'Alice Wang',
    email: 'alice@example.com',
    timestamp: '2026-01-11T14:20:00Z',
  },
  {
    sha: 'efg567hi8901234567890123456789012345efgh',
    message: 'style: Format code with prettier',
    author: 'Bob Chen',
    email: 'bob@example.com',
    timestamp: '2026-01-11T11:00:00Z',
  },
  {
    sha: 'fgh678ij9012345678901234567890123456fghi',
    message: 'test: Add unit tests for auth service',
    author: 'Jane Smith',
    email: 'jane@example.com',
    timestamp: '2026-01-10T17:30:00Z',
  },
  {
    sha: 'ghi789jk0123456789012345678901234567ghij',
    message: 'chore: Update dependencies',
    author: 'John Doe',
    email: 'john@example.com',
    timestamp: '2026-01-10T10:00:00Z',
  },
]

const mockFileDiff = {
  files: [
    {
      filename: 'src/services/auth.ts',
      status: 'modified',
      additions: 45,
      deletions: 12,
      patch: `@@ -1,10 +1,15 @@
 import { api } from './api'
+import { TokenManager } from './tokenManager'
 
 export class AuthService {
-  private token: string | null = null
+  private tokenManager: TokenManager
+  
+  constructor() {
+    this.tokenManager = new TokenManager()
+  }
 
   async login(email: string, password: string) {
-    const response = await api.post('/auth/login', { email, password })
-    this.token = response.data.token
+    const response = await api.post('/auth/login', { email, password })
+    this.tokenManager.setTokens(response.data.accessToken, response.data.refreshToken)
     return response.data
   }
 }`,
    },
    {
      filename: 'src/services/tokenManager.ts',
      status: 'added',
      additions: 35,
      deletions: 0,
      patch: `@@ -0,0 +1,35 @@
+export class TokenManager {
+  private accessToken: string | null = null
+  private refreshToken: string | null = null
+
+  setTokens(access: string, refresh: string) {
+    this.accessToken = access
+    this.refreshToken = refresh
+    localStorage.setItem('accessToken', access)
+    localStorage.setItem('refreshToken', refresh)
+  }
+
+  getAccessToken(): string | null {
+    return this.accessToken || localStorage.getItem('accessToken')
+  }
+
+  clearTokens() {
+    this.accessToken = null
+    this.refreshToken = null
+    localStorage.removeItem('accessToken')
+    localStorage.removeItem('refreshToken')
+  }
+}`,
    },
    {
      filename: 'src/utils/deprecated.ts',
      status: 'removed',
      additions: 0,
      deletions: 20,
      patch: `@@ -1,20 +0,0 @@
-// This file is deprecated
-export function oldFunction() {
-  console.log('deprecated')
-}`,
    },
  ],
}

// Mock repository for standalone page
const mockRepository: GitRepository = {
  id: 'repo1',
  userId: 'user1',
  connectionId: 'conn1',
  environmentId: 'env1',
  provider: 'github',
  repoId: '12345',
  repoName: 'react-app',
  repoFullName: 'developer/react-app',
  cloneUrl: 'https://github.com/developer/react-app.git',
  defaultBranch: 'main',
  isPrivate: false,
  syncStatus: 'synced',
  lastSyncAt: '2026-01-12T10:30:00Z',
  createdAt: '2026-01-10T10:00:00Z',
  updatedAt: '2026-01-12T10:30:00Z',
}

interface GitOperationsProps {
  repositoryId?: string
  repository?: GitRepository
  embedded?: boolean
}

export default function GitOperations({ repositoryId: propRepoId, repository: propRepo, embedded = false }: GitOperationsProps) {
  const { id: paramRepoId } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const repoId = propRepoId || paramRepoId

  const [repository, setRepository] = useState<GitRepository | null>(propRepo || null)
  const [selectedBranch, setSelectedBranch] = useState<string>('')
  const [selectedCommit, setSelectedCommit] = useState<GitCommit | null>(null)
  const [isDiffDrawerOpen, setIsDiffDrawerOpen] = useState(false)
  const [fileDiff, setFileDiff] = useState<typeof mockFileDiff | null>(null)
  const [isLoadingDiff, setIsLoadingDiff] = useState(false)
  const [isSyncing, setIsSyncing] = useState(false)

  const {
    branches,
    commits,
    isLoadingBranches,
    isLoadingCommits,
    setBranches,
    setCommits,
    setLoadingBranches,
    setLoadingCommits,
  } = useGitStore()

  // Use mock data if no real data
  const displayBranches = branches.length > 0 ? branches : mockBranches
  const displayCommits = commits.length > 0 ? commits : mockCommits
  const displayRepo = repository || mockRepository

  useEffect(() => {
    if (displayRepo) {
      setSelectedBranch(displayRepo.defaultBranch)
      loadBranches()
      loadCommits(displayRepo.defaultBranch)
    }
  }, [displayRepo])

  const loadBranches = async () => {
    if (!displayRepo) return
    setLoadingBranches(true)
    try {
      const branchList = await gitService.getBranches(
        displayRepo.provider,
        displayRepo.repoFullName
      )
      setBranches(branchList)
    } catch {
      console.log('Using mock branches')
    } finally {
      setLoadingBranches(false)
    }
  }

  const loadCommits = async (branch: string) => {
    if (!displayRepo) return
    setLoadingCommits(true)
    try {
      const commitList = await gitService.getCommits(
        displayRepo.provider,
        displayRepo.repoFullName,
        branch,
        20
      )
      setCommits(commitList)
    } catch {
      console.log('Using mock commits')
    } finally {
      setLoadingCommits(false)
    }
  }

  const handleBranchChange = (branch: string) => {
    setSelectedBranch(branch)
    loadCommits(branch)
  }

  const handleViewDiff = async (commit: GitCommit) => {
    setSelectedCommit(commit)
    setIsDiffDrawerOpen(true)
    setIsLoadingDiff(true)
    try {
      const diff = await gitService.getFileDiff(
        displayRepo.provider,
        displayRepo.repoFullName,
        commit.sha
      )
      setFileDiff(diff)
    } catch {
      setFileDiff(mockFileDiff)
    } finally {
      setIsLoadingDiff(false)
    }
  }

  const handleSync = async () => {
    if (!displayRepo || !displayRepo.environmentId) {
      message.warning('请先关联环境')
      return
    }
    setIsSyncing(true)
    try {
      await gitService.syncRepository({
        repositoryId: displayRepo.id,
        environmentId: displayRepo.environmentId,
        branch: selectedBranch,
      })
      message.success('同步成功')
      loadCommits(selectedBranch)
    } catch {
      message.error('同步失败')
    } finally {
      setIsSyncing(false)
    }
  }

  const copyToClipboard = (text: string) => {
    navigator.clipboard.writeText(text)
    message.success('已复制到剪贴板')
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

  const formatRelativeTime = (dateStr: string) => {
    const date = new Date(dateStr)
    const now = new Date()
    const diffMs = now.getTime() - date.getTime()
    const diffMins = Math.floor(diffMs / 60000)
    const diffHours = Math.floor(diffMs / 3600000)
    const diffDays = Math.floor(diffMs / 86400000)

    if (diffMins < 60) return `${diffMins} 分钟前`
    if (diffHours < 24) return `${diffHours} 小时前`
    if (diffDays < 7) return `${diffDays} 天前`
    return formatDate(dateStr)
  }

  const getCommitTypeTag = (message: string) => {
    const type = message.split(':')[0]?.toLowerCase()
    const config: Record<string, { color: string; text: string }> = {
      feat: { color: 'green', text: '功能' },
      fix: { color: 'red', text: '修复' },
      docs: { color: 'blue', text: '文档' },
      style: { color: 'purple', text: '样式' },
      refactor: { color: 'orange', text: '重构' },
      test: { color: 'cyan', text: '测试' },
      chore: { color: 'default', text: '杂项' },
    }
    const { color, text } = config[type] || { color: 'default', text: '提交' }
    return <Tag color={color}>{text}</Tag>
  }

  const getFileStatusTag = (status: string) => {
    const config: Record<string, { color: string; icon: React.ReactNode }> = {
      added: { color: 'success', icon: <PlusOutlined /> },
      modified: { color: 'processing', icon: <EditOutlined /> },
      removed: { color: 'error', icon: <MinusOutlined /> },
      renamed: { color: 'warning', icon: <EditOutlined /> },
    }
    const { color, icon } = config[status] || { color: 'default', icon: <FileTextOutlined /> }
    return (
      <Tag color={color} icon={icon}>
        {status}
      </Tag>
    )
  }

  // Stats
  const stats = {
    totalBranches: displayBranches.length,
    protectedBranches: displayBranches.filter((b) => b.protected).length,
    totalCommits: displayCommits.length,
  }

  const content = (
    <div>
      {!embedded && (
        <>
          <div className="flex justify-between items-center mb-6">
            <div className="flex items-center gap-4">
              <Button icon={<ArrowLeftOutlined />} onClick={() => navigate('/git')}>
                返回
              </Button>
              <Breadcrumb
                items={[
                  { title: 'Git 仓库' },
                  { title: displayRepo.repoFullName },
                ]}
              />
            </div>
            <Button
              type="primary"
              icon={<SyncOutlined spin={isSyncing} />}
              onClick={handleSync}
              loading={isSyncing}
            >
              同步到环境
            </Button>
          </div>

          <Row gutter={16} className="mb-4">
            <Col span={8}>
              <Card size="small">
                <Statistic
                  title="分支数量"
                  value={stats.totalBranches}
                  prefix={<BranchesOutlined />}
                />
              </Card>
            </Col>
            <Col span={8}>
              <Card size="small">
                <Statistic
                  title="受保护分支"
                  value={stats.protectedBranches}
                  valueStyle={{ color: '#faad14' }}
                />
              </Card>
            </Col>
            <Col span={8}>
              <Card size="small">
                <Statistic
                  title="最近提交"
                  value={stats.totalCommits}
                  prefix={<HistoryOutlined />}
                />
              </Card>
            </Col>
          </Row>
        </>
      )}

      <Row gutter={16}>
        {/* Branches Panel */}
        <Col span={8}>
          <Card
            title={
              <span>
                <BranchesOutlined /> 分支
              </span>
            }
            extra={
              <Badge count={displayBranches.length} showZero color="#1890ff" />
            }
            loading={isLoadingBranches}
          >
            <div className="mb-4">
              <Select
                value={selectedBranch}
                onChange={handleBranchChange}
                style={{ width: '100%' }}
                placeholder="选择分支"
              >
                {displayBranches.map((branch) => (
                  <Select.Option key={branch.name} value={branch.name}>
                    <div className="flex items-center justify-between">
                      <span>
                        <BranchesOutlined className="mr-2" />
                        {branch.name}
                      </span>
                      {branch.isDefault && (
                        <Tag color="blue" className="ml-2">
                          默认
                        </Tag>
                      )}
                      {branch.protected && (
                        <Tag color="orange" className="ml-1">
                          保护
                        </Tag>
                      )}
                    </div>
                  </Select.Option>
                ))}
              </Select>
            </div>

            <Divider orientation="left" plain>
              所有分支
            </Divider>

            <div className="max-h-80 overflow-y-auto">
              {displayBranches.map((branch) => (
                <div
                  key={branch.name}
                  className={`p-2 rounded cursor-pointer hover:bg-gray-50 mb-1 ${
                    selectedBranch === branch.name ? 'bg-blue-50 border-l-2 border-blue-500' : ''
                  }`}
                  onClick={() => handleBranchChange(branch.name)}
                >
                  <div className="flex items-center justify-between">
                    <span className="font-medium">
                      <BranchesOutlined className="mr-2 text-gray-400" />
                      {branch.name}
                    </span>
                    <Space size="small">
                      {branch.isDefault && <Tag color="blue">默认</Tag>}
                      {branch.protected && <Tag color="orange">保护</Tag>}
                    </Space>
                  </div>
                  <div className="text-xs text-gray-400 mt-1 ml-5">
                    {branch.commitSha.substring(0, 7)}
                  </div>
                </div>
              ))}
            </div>
          </Card>
        </Col>

        {/* Commits Panel */}
        <Col span={16}>
          <Card
            title={
              <span>
                <HistoryOutlined /> 提交历史
                {selectedBranch && (
                  <Tag color="blue" className="ml-2">
                    {selectedBranch}
                  </Tag>
                )}
              </span>
            }
            extra={
              <Button
                icon={<SyncOutlined />}
                onClick={() => loadCommits(selectedBranch)}
                loading={isLoadingCommits}
              >
                刷新
              </Button>
            }
            loading={isLoadingCommits}
          >
            {displayCommits.length > 0 ? (
              <Timeline
                items={displayCommits.map((commit) => ({
                  color: 'blue',
                  children: (
                    <div
                      className="p-3 rounded border hover:border-blue-300 hover:bg-blue-50 cursor-pointer transition-all"
                      onClick={() => handleViewDiff(commit)}
                    >
                      <div className="flex items-start justify-between">
                        <div className="flex-1">
                          <div className="flex items-center gap-2 mb-1">
                            {getCommitTypeTag(commit.message)}
                            <Text strong className="text-base">
                              {commit.message.split('\n')[0]}
                            </Text>
                          </div>
                          {commit.message.includes('\n') && (
                            <Paragraph
                              className="text-gray-500 text-sm mb-2"
                              ellipsis={{ rows: 2 }}
                            >
                              {commit.message.split('\n').slice(1).join('\n').trim()}
                            </Paragraph>
                          )}
                          <div className="flex items-center gap-4 text-xs text-gray-400">
                            <span>
                              <UserOutlined className="mr-1" />
                              {commit.author}
                            </span>
                            <span>
                              <CalendarOutlined className="mr-1" />
                              {formatRelativeTime(commit.timestamp)}
                            </span>
                            <Tooltip title="点击复制">
                              <span
                                className="font-mono hover:text-blue-500"
                                onClick={(e) => {
                                  e.stopPropagation()
                                  copyToClipboard(commit.sha)
                                }}
                              >
                                <CodeOutlined className="mr-1" />
                                {commit.sha.substring(0, 7)}
                              </span>
                            </Tooltip>
                          </div>
                        </div>
                        <Button
                          type="link"
                          icon={<DiffOutlined />}
                          onClick={(e) => {
                            e.stopPropagation()
                            handleViewDiff(commit)
                          }}
                        >
                          查看变更
                        </Button>
                      </div>
                    </div>
                  ),
                }))}
              />
            ) : (
              <Empty description="暂无提交记录" />
            )}
          </Card>
        </Col>
      </Row>

      {/* Diff Drawer */}
      <Drawer
        title={
          <div>
            <DiffOutlined className="mr-2" />
            提交详情
            {selectedCommit && (
              <Tag color="blue" className="ml-2">
                {selectedCommit.sha.substring(0, 7)}
              </Tag>
            )}
          </div>
        }
        placement="right"
        width={800}
        open={isDiffDrawerOpen}
        onClose={() => {
          setIsDiffDrawerOpen(false)
          setSelectedCommit(null)
          setFileDiff(null)
        }}
      >
        {selectedCommit && (
          <div>
            {/* Commit Info */}
            <Card size="small" className="mb-4">
              <div className="mb-2">
                {getCommitTypeTag(selectedCommit.message)}
                <Text strong className="text-lg ml-2">
                  {selectedCommit.message.split('\n')[0]}
                </Text>
              </div>
              {selectedCommit.message.includes('\n') && (
                <Paragraph className="text-gray-600 mb-3 whitespace-pre-wrap">
                  {selectedCommit.message.split('\n').slice(1).join('\n').trim()}
                </Paragraph>
              )}
              <div className="flex items-center gap-4 text-sm text-gray-500">
                <span>
                  <Avatar size="small" icon={<UserOutlined />} className="mr-1" />
                  {selectedCommit.author}
                </span>
                <span>
                  <CalendarOutlined className="mr-1" />
                  {formatDate(selectedCommit.timestamp)}
                </span>
                <Tooltip title="复制完整 SHA">
                  <span
                    className="font-mono cursor-pointer hover:text-blue-500"
                    onClick={() => copyToClipboard(selectedCommit.sha)}
                  >
                    <CopyOutlined className="mr-1" />
                    {selectedCommit.sha}
                  </span>
                </Tooltip>
              </div>
            </Card>

            {/* File Changes */}
            <Spin spinning={isLoadingDiff}>
              {fileDiff ? (
                <div>
                  <div className="mb-3 flex items-center justify-between">
                    <Text strong>
                      变更文件 ({fileDiff.files.length})
                    </Text>
                    <Space>
                      <Tag color="green">
                        +{fileDiff.files.reduce((sum, f) => sum + f.additions, 0)}
                      </Tag>
                      <Tag color="red">
                        -{fileDiff.files.reduce((sum, f) => sum + f.deletions, 0)}
                      </Tag>
                    </Space>
                  </div>

                  {fileDiff.files.map((file, index) => (
                    <Card
                      key={index}
                      size="small"
                      className="mb-3"
                      title={
                        <div className="flex items-center justify-between">
                          <span>
                            <FileTextOutlined className="mr-2" />
                            {file.filename}
                          </span>
                          <Space>
                            {getFileStatusTag(file.status)}
                            <Tag color="green">+{file.additions}</Tag>
                            <Tag color="red">-{file.deletions}</Tag>
                          </Space>
                        </div>
                      }
                    >
                      {file.patch && (
                        <pre className="bg-gray-900 text-gray-100 p-3 rounded text-xs overflow-x-auto">
                          {file.patch.split('\n').map((line, lineIndex) => {
                            let className = ''
                            if (line.startsWith('+') && !line.startsWith('+++')) {
                              className = 'bg-green-900 text-green-300'
                            } else if (line.startsWith('-') && !line.startsWith('---')) {
                              className = 'bg-red-900 text-red-300'
                            } else if (line.startsWith('@@')) {
                              className = 'text-blue-400'
                            }
                            return (
                              <div key={lineIndex} className={className}>
                                {line}
                              </div>
                            )
                          })}
                        </pre>
                      )}
                    </Card>
                  ))}
                </div>
              ) : (
                <Empty description="加载中..." />
              )}
            </Spin>
          </div>
        )}
      </Drawer>
    </div>
  )

  return embedded ? content : <div>{content}</div>
}
