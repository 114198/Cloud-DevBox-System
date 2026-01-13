import { useState, useEffect } from 'react'
import {
  Card,
  Table,
  Button,
  Space,
  Tag,
  Tabs,
  Modal,
  Select,
  message,
  Tooltip,
  Row,
  Col,
  Statistic,
  Progress,
  Upload,
  Empty,
  Popconfirm,
  Spin,
  Typography,
  Alert,
} from 'antd'
import {
  CloudUploadOutlined,
  CloudDownloadOutlined,
  DeleteOutlined,
  ReloadOutlined,
  HistoryOutlined,
  DatabaseOutlined,
  FileOutlined,
  SettingOutlined,
  ExportOutlined,
  ImportOutlined,
  CheckCircleOutlined,
  ClockCircleOutlined,
  SyncOutlined,
  CloseCircleOutlined,
  InfoCircleOutlined,
  DownloadOutlined,
  UploadOutlined,
} from '@ant-design/icons'
import type { ColumnsType } from 'antd/es/table'
import type { UploadProps } from 'antd'
import { useTranslation } from 'react-i18next'
import {
  backupService,
  Backup,
  BackupStats,
  BackupType,
  BackupStatus,
  ConfigVersion,
  EnvironmentExportConfig,
} from '@/services/backup'
import { environmentService } from '@/services/environment'
import { Environment } from '@/stores/environment'

const { Text } = Typography

// Mock data for development
const mockBackups: Backup[] = [
  {
    id: 'bk-001',
    environmentId: 'env-001',
    userId: 'user-001',
    type: 'code',
    mode: 'incremental',
    status: 'completed',
    size: 52428800,
    storagePath: '/backups/code/bk-001.tar.gz',
    storageRegion: 'cn-north-1',
    checksum: 'sha256:abc123...',
    metadata: { fileCount: 1250, totalSize: 52428800, changedFiles: 45 },
    createdAt: '2026-01-13T10:30:00Z',
    completedAt: '2026-01-13T10:31:00Z',
  },
  {
    id: 'bk-002',
    environmentId: 'env-002',
    userId: 'user-001',
    type: 'code',
    mode: 'full',
    status: 'completed',
    size: 104857600,
    storagePath: '/backups/code/bk-002.tar.gz',
    storageRegion: 'cn-north-1',
    checksum: 'sha256:def456...',
    metadata: { fileCount: 2500, totalSize: 104857600 },
    createdAt: '2026-01-12T08:00:00Z',
    completedAt: '2026-01-12T08:05:00Z',
  },
  {
    id: 'bk-003',
    environmentId: 'env-001',
    userId: 'user-001',
    type: 'config',
    mode: 'full',
    status: 'completed',
    size: 2048,
    storagePath: '/backups/config/bk-003.json',
    storageRegion: 'cn-north-1',
    checksum: 'sha256:ghi789...',
    metadata: {},
    createdAt: '2026-01-13T09:00:00Z',
    completedAt: '2026-01-13T09:00:01Z',
  },
  {
    id: 'bk-004',
    environmentId: '',
    userId: 'user-001',
    type: 'database',
    mode: 'full',
    status: 'in_progress',
    size: 0,
    storagePath: '',
    storageRegion: 'cn-north-1',
    checksum: '',
    metadata: {},
    createdAt: '2026-01-13T11:00:00Z',
  },
]

const mockStats: BackupStats = {
  totalBackups: 45,
  totalSize: 5368709120,
  codeBackups: 30,
  configBackups: 10,
  databaseBackups: 5,
  lastBackupAt: '2026-01-13T10:30:00Z',
  oldestBackupAt: '2025-12-15T08:00:00Z',
}

const mockConfigVersions: ConfigVersion[] = [
  {
    id: 'cv-001',
    environmentId: 'env-001',
    version: 5,
    config: { cpu: '4', memory: '8GB', storage: '50GB' },
    message: '升级资源配置',
    createdAt: '2026-01-13T10:00:00Z',
    createdBy: 'user-001',
  },
  {
    id: 'cv-002',
    environmentId: 'env-001',
    version: 4,
    config: { cpu: '2', memory: '4GB', storage: '50GB' },
    message: '调整内存配置',
    createdAt: '2026-01-12T15:00:00Z',
    createdBy: 'user-001',
  },
  {
    id: 'cv-003',
    environmentId: 'env-001',
    version: 3,
    config: { cpu: '2', memory: '4GB', storage: '30GB' },
    message: '增加存储空间',
    createdAt: '2026-01-11T09:00:00Z',
    createdBy: 'user-001',
  },
]

const mockEnvironments: Environment[] = [
  { id: 'env-001', name: 'react-app', description: 'React 前端项目', templateId: 't1', templateName: 'React 18', status: 'running', resources: { cpu: '2 核', memory: '4 GB', storage: '20 GB' }, createdAt: '2026-01-10T10:00:00Z' },
  { id: 'env-002', name: 'node-api', description: 'Node.js API 服务', templateId: 't2', templateName: 'Node.js 18', status: 'stopped', resources: { cpu: '1 核', memory: '2 GB', storage: '10 GB' }, createdAt: '2026-01-09T08:00:00Z' },
]

export default function BackupPage() {
  const { t } = useTranslation()
  const [activeTab, setActiveTab] = useState('backups')
  const [backups, setBackups] = useState<Backup[]>(mockBackups)
  const [stats, setStats] = useState<BackupStats>(mockStats)
  const [configVersions, setConfigVersions] = useState<ConfigVersion[]>(mockConfigVersions)
  const [environments, setEnvironments] = useState<Environment[]>(mockEnvironments)
  const [loading, setLoading] = useState(false)
  const [selectedEnvId, setSelectedEnvId] = useState<string>('')
  const [backupTypeFilter, setBackupTypeFilter] = useState<BackupType | 'all'>('all')
  const [isCreateModalOpen, setIsCreateModalOpen] = useState(false)
  const [isImportModalOpen, setIsImportModalOpen] = useState(false)
  const [isDetailModalOpen, setIsDetailModalOpen] = useState(false)
  const [selectedBackup, setSelectedBackup] = useState<Backup | null>(null)
  const [restoreInProgress, setRestoreInProgress] = useState<string | null>(null)
  const [createBackupType, setCreateBackupType] = useState<BackupType>('code')
  const [createBackupEnvId, setCreateBackupEnvId] = useState<string>('')
  const [importConfig, setImportConfig] = useState<EnvironmentExportConfig | null>(null)

  useEffect(() => {
    loadData()
  }, [])

  const loadData = async () => {
    setLoading(true)
    try {
      // Load environments
      const envList = await environmentService.list().catch(() => mockEnvironments)
      setEnvironments(envList.length > 0 ? envList : mockEnvironments)

      // Load backups from API
      const [codeBackupsRes, dbBackupsRes, codeStats] = await Promise.all([
        backupService.listCodeBackups({ page: 1, pageSize: 100 }).catch(() => ({ backups: [], total: 0 })),
        backupService.listDatabaseBackups(1, 100).catch(() => ({ backups: [], total: 0 })),
        backupService.getCodeBackupStats().catch(() => null),
      ])

      // Combine all backups
      const allBackups = [
        ...codeBackupsRes.backups,
        ...dbBackupsRes.backups,
      ]
      
      if (allBackups.length > 0) {
        setBackups(allBackups)
      }

      // Update stats if available
      if (codeStats) {
        setStats(codeStats)
      }

      // Load config versions for the first environment if available
      if (envList.length > 0) {
        const configVersionsRes = await backupService.listConfigVersions(envList[0].id, 1, 50).catch(() => ({ versions: [] }))
        if (configVersionsRes.versions && configVersionsRes.versions.length > 0) {
          setConfigVersions(configVersionsRes.versions)
        }
      }
    } catch (error) {
      console.log('Using mock data due to API error:', error)
    } finally {
      setLoading(false)
    }
  }

  const loadConfigVersionsForEnv = async (envId: string) => {
    if (!envId) return
    try {
      const res = await backupService.listConfigVersions(envId, 1, 50)
      if (res.versions && res.versions.length > 0) {
        setConfigVersions(res.versions)
      }
    } catch {
      // Keep existing mock data
    }
  }

  const formatBytes = (bytes: number): string => {
    if (bytes === 0) return '0 B'
    const k = 1024
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
    const i = Math.floor(Math.log(bytes) / Math.log(k))
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
  }

  const formatDate = (dateStr: string): string => {
    return new Date(dateStr).toLocaleString('zh-CN', {
      year: 'numeric', month: '2-digit', day: '2-digit',
      hour: '2-digit', minute: '2-digit',
    })
  }

  const getStatusTag = (status: BackupStatus) => {
    const config: Record<BackupStatus, { color: string; icon: React.ReactNode; text: string }> = {
      pending: { color: 'default', icon: <ClockCircleOutlined />, text: t('backup.status.pending', '等待中') },
      in_progress: { color: 'processing', icon: <SyncOutlined spin />, text: t('backup.status.inProgress', '进行中') },
      completed: { color: 'success', icon: <CheckCircleOutlined />, text: t('backup.status.completed', '已完成') },
      failed: { color: 'error', icon: <CloseCircleOutlined />, text: t('backup.status.failed', '失败') },
    }
    const { color, icon, text } = config[status]
    return <Tag color={color} icon={icon}>{text}</Tag>
  }

  const getTypeTag = (type: BackupType) => {
    const config: Record<BackupType, { color: string; icon: React.ReactNode; text: string }> = {
      code: { color: 'blue', icon: <FileOutlined />, text: t('backup.type.code', '代码') },
      config: { color: 'green', icon: <SettingOutlined />, text: t('backup.type.config', '配置') },
      database: { color: 'purple', icon: <DatabaseOutlined />, text: t('backup.type.database', '数据库') },
      full: { color: 'orange', icon: <CloudUploadOutlined />, text: t('backup.type.full', '完整') },
    }
    const { color, icon, text } = config[type]
    return <Tag color={color} icon={icon}>{text}</Tag>
  }

  const handleCreateBackup = async () => {
    if (!createBackupEnvId && createBackupType !== 'database') {
      message.error(t('backup.selectEnvironment', '请选择环境'))
      return
    }
    setLoading(true)
    try {
      if (createBackupType === 'code') {
        await backupService.createCodeBackup(createBackupEnvId)
      } else if (createBackupType === 'database') {
        await backupService.createDatabaseBackup('full')
      }
      message.success(t('backup.createSuccess', '备份创建成功'))
      setIsCreateModalOpen(false)
      loadData()
    } catch {
      message.error(t('backup.createFailed', '备份创建失败'))
    } finally {
      setLoading(false)
    }
  }

  const handleDeleteBackup = async (backup: Backup) => {
    try {
      if (backup.type === 'code') {
        await backupService.deleteCodeBackup(backup.id)
      } else if (backup.type === 'database') {
        await backupService.deleteDatabaseBackup(backup.id)
      }
      message.success(t('backup.deleteSuccess', '备份删除成功'))
      setBackups(backups.filter(b => b.id !== backup.id))
    } catch {
      message.error(t('backup.deleteFailed', '备份删除失败'))
    }
  }

  const handleRestoreBackup = async (backup: Backup) => {
    if (!backup.environmentId && backup.type !== 'database') {
      message.error(t('backup.noEnvironment', '无法恢复：缺少环境信息'))
      return
    }
    setRestoreInProgress(backup.id)
    try {
      if (backup.type === 'database') {
        await backupService.restoreDatabaseBackup(backup.id)
        message.success(t('backup.restoreStarted', '数据库恢复操作已开始'))
      } else {
        await backupService.restoreFromBackup(backup.id, backup.environmentId, true)
        message.success(t('backup.restoreStarted', '恢复操作已开始'))
      }
      // Refresh the backup list to show updated status
      loadData()
    } catch {
      message.error(t('backup.restoreFailed', '恢复失败'))
    } finally {
      setRestoreInProgress(null)
    }
  }

  const handleViewBackupDetails = (backup: Backup) => {
    setSelectedBackup(backup)
    setIsDetailModalOpen(true)
  }

  const handleRestoreConfigVersion = async (version: ConfigVersion) => {
    setLoading(true)
    try {
      await backupService.restoreConfigVersion(version.environmentId, version.version)
      message.success(t('backup.configRestoreSuccess', '配置恢复成功'))
      // Refresh config versions to show the new version created
      loadConfigVersionsForEnv(version.environmentId)
    } catch {
      message.error(t('backup.configRestoreFailed', '配置恢复失败'))
    } finally {
      setLoading(false)
    }
  }

  const handleExportConfig = async (envId: string) => {
    try {
      const yaml = await backupService.downloadEnvironmentConfigYaml(envId)
      const env = environments.find(e => e.id === envId)
      const blob = new Blob([yaml], { type: 'text/yaml' })
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = `${env?.name || 'environment'}-config.yaml`
      document.body.appendChild(a)
      a.click()
      document.body.removeChild(a)
      URL.revokeObjectURL(url)
      message.success(t('backup.exportSuccess', '配置导出成功'))
    } catch {
      // Mock export for development
      const env = environments.find(e => e.id === envId)
      const mockYaml = `# Cloud DevBox Environment Configuration
name: ${env?.name || 'environment'}
description: ${env?.description || ''}
template: ${env?.templateName || ''}
resources:
  cpu: ${env?.resources.cpu || '2 核'}
  memory: ${env?.resources.memory || '4 GB'}
  storage: ${env?.resources.storage || '20 GB'}
exportedAt: ${new Date().toISOString()}
version: "1.0"
`
      const blob = new Blob([mockYaml], { type: 'text/yaml' })
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = `${env?.name || 'environment'}-config.yaml`
      document.body.appendChild(a)
      a.click()
      document.body.removeChild(a)
      URL.revokeObjectURL(url)
      message.success(t('backup.exportSuccess', '配置导出成功'))
    }
  }

  const handleDownloadBackup = async (backup: Backup) => {
    if (backup.status !== 'completed') {
      message.warning(t('backup.notCompleted', '备份尚未完成'))
      return
    }
    try {
      // In production, this would download from the storage path
      // For now, we'll show a message about the storage location
      message.info(`${t('backup.downloadInfo', '备份文件位置')}: ${backup.storagePath}`)
    } catch {
      message.error(t('backup.downloadFailed', '下载失败'))
    }
  }

  const handleImportConfig = async () => {
    if (!importConfig) {
      message.error(t('backup.noConfigFile', '请先上传配置文件'))
      return
    }
    try {
      await backupService.importEnvironmentConfig(importConfig)
      message.success(t('backup.importSuccess', '配置导入成功，环境创建中'))
      setIsImportModalOpen(false)
      setImportConfig(null)
    } catch {
      message.error(t('backup.importFailed', '配置导入失败'))
    }
  }

  // Simple YAML parser for environment config files
  const parseYamlConfig = (content: string): EnvironmentExportConfig | null => {
    try {
      // First try JSON parsing
      return JSON.parse(content) as EnvironmentExportConfig
    } catch {
      // Simple YAML parsing for common config format
      try {
        const lines = content.split('\n')
        const config: Record<string, unknown> = {}
        let currentSection = ''
        let currentSubSection = ''
        
        for (const line of lines) {
          const trimmed = line.trim()
          if (!trimmed || trimmed.startsWith('#')) continue
          
          const indent = line.search(/\S/)
          const [key, ...valueParts] = trimmed.split(':')
          const value = valueParts.join(':').trim()
          
          if (indent === 0 && value === '') {
            currentSection = key
            config[currentSection] = {}
          } else if (indent === 2 && value === '') {
            currentSubSection = key
            if (typeof config[currentSection] === 'object') {
              (config[currentSection] as Record<string, unknown>)[currentSubSection] = {}
            }
          } else if (indent === 0 && value) {
            config[key] = value.replace(/^["']|["']$/g, '')
          } else if (indent === 2 && value && currentSection) {
            if (typeof config[currentSection] === 'object') {
              (config[currentSection] as Record<string, unknown>)[key] = value.replace(/^["']|["']$/g, '')
            }
          } else if (indent === 4 && value && currentSection && currentSubSection) {
            const section = config[currentSection] as Record<string, unknown>
            if (section && typeof section[currentSubSection] === 'object') {
              (section[currentSubSection] as Record<string, unknown>)[key] = value.replace(/^["']|["']$/g, '')
            }
          }
        }
        
        // Validate required fields
        if (!config.name) {
          return null
        }
        
        return {
          name: config.name as string,
          description: config.description as string || '',
          templateId: config.templateId as string || config.template as string || '',
          resources: (config.resources as { cpu: string; memory: string; storage: string }) || { cpu: '2', memory: '4GB', storage: '20GB' },
          environment: (config.environment as Record<string, string>) || {},
          ports: (config.ports as Array<{ containerPort: number; protocol: string; name?: string }>) || [],
          volumes: (config.volumes as Array<{ name: string; mountPath: string; size: string }>) || [],
          runtime: (config.runtime as { language: string; version: string; framework?: string }) || { language: '', version: '' },
          exportedAt: config.exportedAt as string || new Date().toISOString(),
          exportedBy: config.exportedBy as string || '',
          version: config.version as string || '1.0',
        }
      } catch {
        return null
      }
    }
  }

  const uploadProps: UploadProps = {
    accept: '.yaml,.yml,.json',
    maxCount: 1,
    beforeUpload: (file) => {
      const reader = new FileReader()
      reader.onload = (e) => {
        const content = e.target?.result as string
        const config = parseYamlConfig(content)
        if (config) {
          setImportConfig(config)
          message.success(t('backup.fileLoaded', '配置文件加载成功'))
        } else {
          message.error(t('backup.invalidFile', '无效的配置文件格式'))
        }
      }
      reader.readAsText(file)
      return false
    },
  }

  const filteredBackups = backups.filter(b => {
    if (backupTypeFilter !== 'all' && b.type !== backupTypeFilter) return false
    if (selectedEnvId && b.environmentId !== selectedEnvId) return false
    return true
  })

  const backupColumns: ColumnsType<Backup> = [
    {
      title: t('backup.columns.type', '类型'),
      dataIndex: 'type',
      key: 'type',
      width: 100,
      render: (type: BackupType) => getTypeTag(type),
    },
    {
      title: t('backup.columns.environment', '环境'),
      dataIndex: 'environmentId',
      key: 'environmentId',
      render: (envId: string) => {
        const env = environments.find(e => e.id === envId)
        return env?.name || (envId ? envId.slice(0, 8) : '-')
      },
    },
    {
      title: t('backup.columns.mode', '模式'),
      dataIndex: 'mode',
      key: 'mode',
      width: 100,
      render: (mode: string) => (
        <Tag>{mode === 'full' ? t('backup.mode.full', '全量') : t('backup.mode.incremental', '增量')}</Tag>
      ),
    },
    {
      title: t('backup.columns.status', '状态'),
      dataIndex: 'status',
      key: 'status',
      width: 120,
      render: (status: BackupStatus) => getStatusTag(status),
    },
    {
      title: t('backup.columns.size', '大小'),
      dataIndex: 'size',
      key: 'size',
      width: 100,
      render: (size: number) => formatBytes(size),
    },
    {
      title: t('backup.columns.createdAt', '创建时间'),
      dataIndex: 'createdAt',
      key: 'createdAt',
      width: 160,
      render: (date: string) => formatDate(date),
      sorter: (a, b) => new Date(a.createdAt).getTime() - new Date(b.createdAt).getTime(),
      defaultSortOrder: 'descend',
    },
    {
      title: t('common.actions', '操作'),
      key: 'actions',
      width: 220,
      render: (_, record) => (
        <Space size="small">
          <Tooltip title={t('backup.viewDetails', '查看详情')}>
            <Button
              icon={<InfoCircleOutlined />}
              size="small"
              onClick={() => handleViewBackupDetails(record)}
            />
          </Tooltip>
          <Tooltip title={t('backup.restore', '恢复')}>
            <Button
              icon={<CloudDownloadOutlined />}
              size="small"
              disabled={record.status !== 'completed' || restoreInProgress === record.id}
              loading={restoreInProgress === record.id}
              onClick={() => handleRestoreBackup(record)}
            />
          </Tooltip>
          <Tooltip title={t('common.download', '下载')}>
            <Button
              icon={<DownloadOutlined />}
              size="small"
              disabled={record.status !== 'completed'}
              onClick={() => handleDownloadBackup(record)}
            />
          </Tooltip>
          <Popconfirm
            title={t('backup.confirmDelete', '确定要删除此备份吗？')}
            onConfirm={() => handleDeleteBackup(record)}
            okText={t('common.yes', '是')}
            cancelText={t('common.no', '否')}
          >
            <Button icon={<DeleteOutlined />} size="small" danger />
          </Popconfirm>
        </Space>
      ),
    },
  ]

  const configVersionColumns: ColumnsType<ConfigVersion> = [
    {
      title: t('backup.columns.version', '版本'),
      dataIndex: 'version',
      key: 'version',
      width: 80,
      render: (v: number) => <Tag color="blue">v{v}</Tag>,
    },
    {
      title: t('backup.columns.environment', '环境'),
      dataIndex: 'environmentId',
      key: 'environmentId',
      render: (envId: string) => {
        const env = environments.find(e => e.id === envId)
        return env?.name || envId.slice(0, 8)
      },
    },
    {
      title: t('backup.columns.message', '说明'),
      dataIndex: 'message',
      key: 'message',
      render: (msg: string) => msg || '-',
    },
    {
      title: t('backup.columns.createdAt', '创建时间'),
      dataIndex: 'createdAt',
      key: 'createdAt',
      width: 160,
      render: (date: string) => formatDate(date),
    },
    {
      title: t('common.actions', '操作'),
      key: 'actions',
      width: 120,
      render: (_, record) => (
        <Space size="small">
          <Tooltip title={t('backup.restoreVersion', '恢复到此版本')}>
            <Popconfirm
              title={t('backup.confirmRestoreVersion', '确定要恢复到此版本吗？')}
              onConfirm={() => handleRestoreConfigVersion(record)}
              okText={t('common.yes', '是')}
              cancelText={t('common.no', '否')}
            >
              <Button icon={<HistoryOutlined />} size="small" type="primary" />
            </Popconfirm>
          </Tooltip>
        </Space>
      ),
    },
  ]

  const storageUsagePercent = Math.round((stats.totalSize / (10 * 1024 * 1024 * 1024)) * 100)

  return (
    <div>
      <div className="flex justify-between items-center mb-6">
        <h1 className="text-2xl font-semibold">{t('backup.title', '备份管理')}</h1>
        <Space>
          <Button icon={<ImportOutlined />} onClick={() => setIsImportModalOpen(true)}>
            {t('backup.import', '导入配置')}
          </Button>
          <Button type="primary" icon={<CloudUploadOutlined />} onClick={() => setIsCreateModalOpen(true)}>
            {t('backup.create', '创建备份')}
          </Button>
        </Space>
      </div>

      {/* Statistics Cards */}
      <Row gutter={16} className="mb-4">
        <Col span={6}>
          <Card size="small">
            <Statistic
              title={t('backup.stats.total', '总备份数')}
              value={stats.totalBackups}
              prefix={<CloudUploadOutlined />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card size="small">
            <Statistic
              title={t('backup.stats.totalSize', '总大小')}
              value={formatBytes(stats.totalSize)}
              prefix={<DatabaseOutlined />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card size="small">
            <Statistic
              title={t('backup.stats.lastBackup', '最近备份')}
              value={stats.lastBackupAt ? formatDate(stats.lastBackupAt) : '-'}
              prefix={<ClockCircleOutlined />}
              valueStyle={{ fontSize: '14px' }}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card size="small">
            <div className="mb-2">
              <Text type="secondary">{t('backup.stats.storageUsage', '存储使用')}</Text>
            </div>
            <Progress percent={storageUsagePercent} status={storageUsagePercent > 80 ? 'exception' : 'active'} />
            <Text type="secondary" className="text-xs">
              {formatBytes(stats.totalSize)} / 10 GB
            </Text>
          </Card>
        </Col>
      </Row>

      {/* Main Content Tabs */}
      <Card>
        <Tabs
          activeKey={activeTab}
          onChange={setActiveTab}
          items={[
            {
              key: 'backups',
              label: (
                <span>
                  <HistoryOutlined />
                  {t('backup.tabs.backups', '备份历史')}
                </span>
              ),
              children: (
                <div>
                  <div className="flex justify-between items-center mb-4">
                    <Space>
                      <Select
                        placeholder={t('backup.filterByType', '按类型筛选')}
                        value={backupTypeFilter}
                        onChange={setBackupTypeFilter}
                        style={{ width: 120 }}
                        options={[
                          { label: t('common.all', '全部'), value: 'all' },
                          { label: t('backup.type.code', '代码'), value: 'code' },
                          { label: t('backup.type.config', '配置'), value: 'config' },
                          { label: t('backup.type.database', '数据库'), value: 'database' },
                        ]}
                      />
                      <Select
                        placeholder={t('backup.filterByEnv', '按环境筛选')}
                        value={selectedEnvId}
                        onChange={setSelectedEnvId}
                        style={{ width: 160 }}
                        allowClear
                        options={[
                          { label: t('common.all', '全部环境'), value: '' },
                          ...environments.map(e => ({ label: e.name, value: e.id })),
                        ]}
                      />
                    </Space>
                    <Button icon={<ReloadOutlined />} onClick={loadData}>
                      {t('common.refresh', '刷新')}
                    </Button>
                  </div>
                  <Table
                    columns={backupColumns}
                    dataSource={filteredBackups}
                    rowKey="id"
                    loading={loading}
                    pagination={{
                      showSizeChanger: true,
                      showQuickJumper: true,
                      showTotal: (total) => t('backup.totalBackups', `共 ${total} 个备份`, { total }),
                    }}
                  />
                </div>
              ),
            },

            {
              key: 'configVersions',
              label: (
                <span>
                  <SettingOutlined />
                  {t('backup.tabs.configVersions', '配置版本')}
                </span>
              ),
              children: (
                <div>
                  <Alert
                    message={t('backup.configVersionInfo', '配置版本会在每次修改环境配置时自动保存，您可以随时恢复到之前的版本。')}
                    type="info"
                    showIcon
                    icon={<InfoCircleOutlined />}
                    className="mb-4"
                  />
                  <Table
                    columns={configVersionColumns}
                    dataSource={configVersions}
                    rowKey="id"
                    loading={loading}
                    pagination={{
                      showSizeChanger: true,
                      showQuickJumper: true,
                      showTotal: (total) => t('backup.totalVersions', `共 ${total} 个版本`, { total }),
                    }}
                  />
                </div>
              ),
            },
            {
              key: 'export',
              label: (
                <span>
                  <ExportOutlined />
                  {t('backup.tabs.export', '导入/导出')}
                </span>
              ),
              children: (
                <div>
                  <Alert
                    message={t('backup.exportInfo', '您可以将环境配置导出为 YAML 文件，方便迁移或分享。导入配置文件可以快速创建新环境。')}
                    type="info"
                    showIcon
                    icon={<InfoCircleOutlined />}
                    className="mb-4"
                  />
                  <Row gutter={16}>
                    <Col span={12}>
                      <Card title={t('backup.exportConfig', '导出配置')} size="small">
                        {environments.length > 0 ? (
                          <div className="space-y-3">
                            {environments.map(env => (
                              <div key={env.id} className="flex justify-between items-center p-3 bg-gray-50 rounded">
                                <div>
                                  <div className="font-medium">{env.name}</div>
                                  <div className="text-gray-500 text-sm">{env.templateName}</div>
                                </div>
                                <Button
                                  icon={<DownloadOutlined />}
                                  onClick={() => handleExportConfig(env.id)}
                                >
                                  {t('backup.export', '导出')}
                                </Button>
                              </div>
                            ))}
                          </div>
                        ) : (
                          <Empty description={t('backup.noEnvironments', '暂无环境')} />
                        )}
                      </Card>
                    </Col>

                    <Col span={12}>
                      <Card title={t('backup.importConfig', '导入配置')} size="small">
                        <Upload.Dragger {...uploadProps} className="mb-4">
                          <p className="ant-upload-drag-icon">
                            <UploadOutlined />
                          </p>
                          <p className="ant-upload-text">
                            {t('backup.uploadHint', '点击或拖拽文件到此区域上传')}
                          </p>
                          <p className="ant-upload-hint">
                            {t('backup.uploadFormat', '支持 YAML 或 JSON 格式的配置文件')}
                          </p>
                        </Upload.Dragger>
                        {importConfig && (
                          <div className="p-3 bg-green-50 rounded mb-4">
                            <div className="flex items-center gap-2 text-green-600">
                              <CheckCircleOutlined />
                              <span>{t('backup.configLoaded', '配置已加载')}: {importConfig.name}</span>
                            </div>
                          </div>
                        )}
                        <Button
                          type="primary"
                          icon={<ImportOutlined />}
                          onClick={handleImportConfig}
                          disabled={!importConfig}
                          block
                        >
                          {t('backup.createFromConfig', '从配置创建环境')}
                        </Button>
                      </Card>
                    </Col>
                  </Row>
                </div>
              ),
            },
          ]}
        />
      </Card>

      {/* Create Backup Modal */}
      <Modal
        title={t('backup.createBackup', '创建备份')}
        open={isCreateModalOpen}
        onOk={handleCreateBackup}
        onCancel={() => setIsCreateModalOpen(false)}
        okText={t('common.create', '创建')}
        cancelText={t('common.cancel', '取消')}
        confirmLoading={loading}
      >
        <div className="space-y-4">
          <div>
            <label className="block text-sm font-medium mb-2">
              {t('backup.backupType', '备份类型')}
            </label>
            <Select
              value={createBackupType}
              onChange={setCreateBackupType}
              style={{ width: '100%' }}
              options={[
                { label: t('backup.type.code', '代码备份'), value: 'code' },
                { label: t('backup.type.database', '数据库备份'), value: 'database' },
              ]}
            />
          </div>
          {createBackupType !== 'database' && (
            <div>
              <label className="block text-sm font-medium mb-2">
                {t('backup.selectEnvironment', '选择环境')}
              </label>
              <Select
                value={createBackupEnvId}
                onChange={setCreateBackupEnvId}
                style={{ width: '100%' }}
                placeholder={t('backup.selectEnvironmentPlaceholder', '请选择要备份的环境')}
                options={environments.map(e => ({ label: e.name, value: e.id }))}
              />
            </div>
          )}
        </div>
      </Modal>

      {/* Import Config Modal */}
      <Modal
        title={t('backup.importConfig', '导入配置')}
        open={isImportModalOpen}
        onOk={handleImportConfig}
        onCancel={() => {
          setIsImportModalOpen(false)
          setImportConfig(null)
        }}
        okText={t('backup.createEnvironment', '创建环境')}
        cancelText={t('common.cancel', '取消')}
        okButtonProps={{ disabled: !importConfig }}
      >
        <Upload.Dragger {...uploadProps}>
          <p className="ant-upload-drag-icon">
            <UploadOutlined />
          </p>
          <p className="ant-upload-text">
            {t('backup.uploadHint', '点击或拖拽文件到此区域上传')}
          </p>
          <p className="ant-upload-hint">
            {t('backup.uploadFormat', '支持 YAML 或 JSON 格式的配置文件')}
          </p>
        </Upload.Dragger>
        {importConfig && (
          <div className="mt-4 p-3 bg-green-50 rounded">
            <div className="flex items-center gap-2 text-green-600 mb-2">
              <CheckCircleOutlined />
              <span className="font-medium">{t('backup.configLoaded', '配置已加载')}</span>
            </div>
            <div className="text-sm text-gray-600">
              <div>{t('common.name', '名称')}: {importConfig.name}</div>
              {importConfig.description && (
                <div>{t('common.description', '描述')}: {importConfig.description}</div>
              )}
            </div>
          </div>
        )}
      </Modal>

      {/* Backup Details Modal */}
      <Modal
        title={t('backup.backupDetails', '备份详情')}
        open={isDetailModalOpen}
        onCancel={() => {
          setIsDetailModalOpen(false)
          setSelectedBackup(null)
        }}
        footer={[
          <Button key="close" onClick={() => setIsDetailModalOpen(false)}>
            {t('common.close', '关闭')}
          </Button>,
          <Button
            key="restore"
            type="primary"
            icon={<CloudDownloadOutlined />}
            disabled={selectedBackup?.status !== 'completed'}
            loading={restoreInProgress === selectedBackup?.id}
            onClick={() => selectedBackup && handleRestoreBackup(selectedBackup)}
          >
            {t('backup.restore', '恢复')}
          </Button>,
        ]}
        width={600}
      >
        {selectedBackup && (
          <div className="space-y-4">
            <Row gutter={16}>
              <Col span={12}>
                <div className="text-gray-500 text-sm">{t('backup.columns.type', '类型')}</div>
                <div className="mt-1">{getTypeTag(selectedBackup.type)}</div>
              </Col>
              <Col span={12}>
                <div className="text-gray-500 text-sm">{t('backup.columns.status', '状态')}</div>
                <div className="mt-1">{getStatusTag(selectedBackup.status)}</div>
              </Col>
            </Row>
            <Row gutter={16}>
              <Col span={12}>
                <div className="text-gray-500 text-sm">{t('backup.columns.environment', '环境')}</div>
                <div className="mt-1 font-medium">
                  {environments.find(e => e.id === selectedBackup.environmentId)?.name || selectedBackup.environmentId || '-'}
                </div>
              </Col>
              <Col span={12}>
                <div className="text-gray-500 text-sm">{t('backup.columns.mode', '模式')}</div>
                <div className="mt-1">
                  <Tag>{selectedBackup.mode === 'full' ? t('backup.mode.full', '全量') : t('backup.mode.incremental', '增量')}</Tag>
                </div>
              </Col>
            </Row>
            <Row gutter={16}>
              <Col span={12}>
                <div className="text-gray-500 text-sm">{t('backup.columns.size', '大小')}</div>
                <div className="mt-1 font-medium">{formatBytes(selectedBackup.size)}</div>
              </Col>
              <Col span={12}>
                <div className="text-gray-500 text-sm">{t('backup.columns.createdAt', '创建时间')}</div>
                <div className="mt-1">{formatDate(selectedBackup.createdAt)}</div>
              </Col>
            </Row>
            {selectedBackup.completedAt && (
              <Row gutter={16}>
                <Col span={12}>
                  <div className="text-gray-500 text-sm">{t('backup.completedAt', '完成时间')}</div>
                  <div className="mt-1">{formatDate(selectedBackup.completedAt)}</div>
                </Col>
                <Col span={12}>
                  <div className="text-gray-500 text-sm">{t('backup.duration', '耗时')}</div>
                  <div className="mt-1">
                    {Math.round((new Date(selectedBackup.completedAt).getTime() - new Date(selectedBackup.createdAt).getTime()) / 1000)} {t('backup.seconds', '秒')}
                  </div>
                </Col>
              </Row>
            )}
            <div>
              <div className="text-gray-500 text-sm">{t('backup.storagePath', '存储路径')}</div>
              <div className="mt-1 font-mono text-sm bg-gray-50 p-2 rounded break-all">
                {selectedBackup.storagePath || '-'}
              </div>
            </div>
            {selectedBackup.checksum && (
              <div>
                <div className="text-gray-500 text-sm">{t('backup.checksum', '校验和')}</div>
                <div className="mt-1 font-mono text-xs bg-gray-50 p-2 rounded break-all">
                  {selectedBackup.checksum}
                </div>
              </div>
            )}
            {selectedBackup.metadata && Object.keys(selectedBackup.metadata).length > 0 && (
              <div>
                <div className="text-gray-500 text-sm mb-2">{t('backup.metadata', '元数据')}</div>
                <div className="bg-gray-50 p-3 rounded space-y-1">
                  {selectedBackup.metadata.fileCount !== undefined && (
                    <div className="flex justify-between">
                      <span className="text-gray-600">{t('backup.fileCount', '文件数量')}</span>
                      <span className="font-medium">{selectedBackup.metadata.fileCount.toLocaleString()}</span>
                    </div>
                  )}
                  {selectedBackup.metadata.changedFiles !== undefined && (
                    <div className="flex justify-between">
                      <span className="text-gray-600">{t('backup.changedFiles', '变更文件')}</span>
                      <span className="font-medium">{selectedBackup.metadata.changedFiles.toLocaleString()}</span>
                    </div>
                  )}
                  {selectedBackup.metadata.totalSize !== undefined && (
                    <div className="flex justify-between">
                      <span className="text-gray-600">{t('backup.originalSize', '原始大小')}</span>
                      <span className="font-medium">{formatBytes(selectedBackup.metadata.totalSize)}</span>
                    </div>
                  )}
                </div>
              </div>
            )}
          </div>
        )}
      </Modal>
    </div>
  )
}
