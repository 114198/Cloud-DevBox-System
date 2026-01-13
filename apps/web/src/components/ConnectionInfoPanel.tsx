import { useState, useEffect } from 'react'
import {
  Card,
  Tabs,
  Button,
  Input,
  Select,
  Space,
  Typography,
  message,
  Tooltip,
  Tag,
  Spin,
  Alert,
  Divider,
} from 'antd'
import {
  CopyOutlined,
  DownloadOutlined,
  CodeOutlined,
  DesktopOutlined,
  CloudServerOutlined,
  CheckCircleOutlined,
  InfoCircleOutlined,
} from '@ant-design/icons'
import { connectionService, SSHConfigResponse, IDEType } from '@/services/connection'

const { Text, Paragraph, Title } = Typography
const { TextArea } = Input

interface ConnectionInfoPanelProps {
  environmentId: string
  environmentName: string
  isRunning: boolean
}

export default function ConnectionInfoPanel({
  environmentId,
  environmentName,
  isRunning,
}: ConnectionInfoPanelProps) {
  const [loading, setLoading] = useState(false)
  const [sshConfig, setSSHConfig] = useState<SSHConfigResponse | null>(null)
  const [sshCommand, setSSHCommand] = useState<string>('')
  const [selectedIDE, setSelectedIDE] = useState<IDEType>('vscode')
  const [keyPath, setKeyPath] = useState('~/.ssh/devbox_key')
  const [connectionStats, setConnectionStats] = useState<{
    canConnect: boolean
    currentConnections: number
    maxConnections: number
  } | null>(null)

  useEffect(() => {
    if (isRunning && environmentId) {
      loadConnectionInfo()
    }
  }, [environmentId, isRunning, selectedIDE, keyPath])

  const loadConnectionInfo = async () => {
    setLoading(true)
    try {
      const [configRes, commandRes, statsRes] = await Promise.all([
        connectionService.getSSHConfig(environmentId, selectedIDE, keyPath),
        connectionService.getSSHCommand(environmentId, keyPath),
        connectionService.canConnect(environmentId),
      ])
      setSSHConfig(configRes)
      setSSHCommand(commandRes)
      setConnectionStats(statsRes)
    } catch (error) {
      console.error('Failed to load connection info:', error)
      // Use mock data for demo
      setSSHConfig({
        config: {
          host: `devbox-${environmentId.slice(0, 8)}`,
          port: 22,
          user: 'devbox',
          identityFile: keyPath,
          strictHostKeyChecking: 'no',
          userKnownHostsFile: '/dev/null',
          serverAliveInterval: 60,
          serverAliveCountMax: 3,
        },
        configString: generateMockConfig(environmentId, environmentName, selectedIDE, keyPath),
        ideType: selectedIDE,
        instructions: generateMockInstructions(environmentId, selectedIDE),
      })
      setSSHCommand(`ssh -p 22 -i ${keyPath} -o StrictHostKeyChecking=no devbox@devbox-${environmentId.slice(0, 8)}.devbox.com`)
      setConnectionStats({ canConnect: true, currentConnections: 1, maxConnections: 5 })
    } finally {
      setLoading(false)
    }
  }

  const copyToClipboard = async (text: string, label: string) => {
    try {
      await navigator.clipboard.writeText(text)
      message.success(`${label} 已复制到剪贴板`)
    } catch {
      message.error('复制失败，请手动复制')
    }
  }

  const downloadConfig = () => {
    if (!sshConfig) return
    const blob = new Blob([sshConfig.configString], { type: 'text/plain' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `devbox-${environmentId.slice(0, 8)}-ssh-config`
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
    URL.revokeObjectURL(url)
    message.success('SSH 配置文件已下载')
  }

  if (!isRunning) {
    return (
      <Card>
        <Alert
          message="环境未运行"
          description="请先启动环境后再查看连接信息"
          type="warning"
          showIcon
        />
      </Card>
    )
  }

  return (
    <Card title="连接信息" className="connection-info-panel">
      <Spin spinning={loading}>
        {/* Connection Status */}
        {connectionStats && (
          <div className="mb-4">
            <Space>
              <Tag color={connectionStats.canConnect ? 'green' : 'red'}>
                {connectionStats.canConnect ? '可连接' : '连接已满'}
              </Tag>
              <Text type="secondary">
                当前连接: {connectionStats.currentConnections}/{connectionStats.maxConnections}
              </Text>
            </Space>
          </div>
        )}

        {/* IDE Selection */}
        <div className="mb-4">
          <Text strong>选择 IDE 类型：</Text>
          <Select
            value={selectedIDE}
            onChange={setSelectedIDE}
            style={{ width: 200, marginLeft: 12 }}
            options={[
              { value: 'vscode', label: 'VSCode Remote SSH' },
              { value: 'jetbrains', label: 'JetBrains Gateway' },
              { value: 'generic', label: '通用 SSH' },
            ]}
          />
        </div>

        {/* Key Path */}
        <div className="mb-4">
          <Text strong>SSH 密钥路径：</Text>
          <Input
            value={keyPath}
            onChange={(e) => setKeyPath(e.target.value)}
            style={{ width: 300, marginLeft: 12 }}
            placeholder="~/.ssh/devbox_key"
          />
        </div>

        <Divider />

        <Tabs
          defaultActiveKey="quick"
          items={[
            {
              key: 'quick',
              label: (
                <span>
                  <CodeOutlined /> 快速连接
                </span>
              ),
              children: (
                <div>
                  <Title level={5}>一键复制 SSH 命令</Title>
                  <div className="bg-gray-900 text-green-400 p-4 rounded-lg font-mono text-sm mb-4">
                    <div className="flex justify-between items-center">
                      <code>{sshCommand}</code>
                      <Tooltip title="复制命令">
                        <Button
                          type="text"
                          icon={<CopyOutlined className="text-white" />}
                          onClick={() => copyToClipboard(sshCommand, 'SSH 命令')}
                        />
                      </Tooltip>
                    </div>
                  </div>

                  <Alert
                    message="使用说明"
                    description={
                      <ol className="list-decimal list-inside space-y-1">
                        <li>确保已生成 SSH 密钥并保存到指定路径</li>
                        <li>复制上方命令到终端执行</li>
                        <li>首次连接可能需要确认主机指纹</li>
                      </ol>
                    }
                    type="info"
                    showIcon
                  />
                </div>
              ),
            },
            {
              key: 'config',
              label: (
                <span>
                  <DesktopOutlined /> SSH 配置
                </span>
              ),
              children: (
                <div>
                  <div className="flex justify-between items-center mb-2">
                    <Title level={5}>SSH 配置文件</Title>
                    <Space>
                      <Button
                        icon={<CopyOutlined />}
                        onClick={() => copyToClipboard(sshConfig?.configString || '', 'SSH 配置')}
                      >
                        复制配置
                      </Button>
                      <Button icon={<DownloadOutlined />} onClick={downloadConfig}>
                        下载配置
                      </Button>
                    </Space>
                  </div>

                  <TextArea
                    value={sshConfig?.configString || ''}
                    readOnly
                    rows={12}
                    className="font-mono text-sm"
                    style={{ backgroundColor: '#1e1e1e', color: '#d4d4d4' }}
                  />

                  <Alert
                    className="mt-4"
                    message="配置说明"
                    description={
                      <div>
                        <p>将上方配置添加到 <code>~/.ssh/config</code> 文件中</p>
                        <p className="mt-2">
                          <strong>Windows 用户：</strong> 配置文件位于{' '}
                          <code>C:\Users\用户名\.ssh\config</code>
                        </p>
                      </div>
                    }
                    type="info"
                    showIcon
                  />
                </div>
              ),
            },
            {
              key: 'details',
              label: (
                <span>
                  <CloudServerOutlined /> 连接详情
                </span>
              ),
              children: sshConfig?.config && (
                <div>
                  <Title level={5}>连接参数</Title>
                  <div className="grid grid-cols-2 gap-4">
                    <div className="bg-gray-50 p-3 rounded">
                      <Text type="secondary">主机地址</Text>
                      <div className="flex items-center justify-between">
                        <Text strong>{sshConfig.config.host}</Text>
                        <Button
                          type="text"
                          size="small"
                          icon={<CopyOutlined />}
                          onClick={() => copyToClipboard(sshConfig.config.host, '主机地址')}
                        />
                      </div>
                    </div>
                    <div className="bg-gray-50 p-3 rounded">
                      <Text type="secondary">端口</Text>
                      <div className="flex items-center justify-between">
                        <Text strong>{sshConfig.config.port}</Text>
                        <Button
                          type="text"
                          size="small"
                          icon={<CopyOutlined />}
                          onClick={() => copyToClipboard(String(sshConfig.config.port), '端口')}
                        />
                      </div>
                    </div>
                    <div className="bg-gray-50 p-3 rounded">
                      <Text type="secondary">用户名</Text>
                      <div className="flex items-center justify-between">
                        <Text strong>{sshConfig.config.user}</Text>
                        <Button
                          type="text"
                          size="small"
                          icon={<CopyOutlined />}
                          onClick={() => copyToClipboard(sshConfig.config.user, '用户名')}
                        />
                      </div>
                    </div>
                    <div className="bg-gray-50 p-3 rounded">
                      <Text type="secondary">密钥文件</Text>
                      <div className="flex items-center justify-between">
                        <Text strong>{sshConfig.config.identityFile || '未指定'}</Text>
                        {sshConfig.config.identityFile && (
                          <Button
                            type="text"
                            size="small"
                            icon={<CopyOutlined />}
                            onClick={() =>
                              copyToClipboard(sshConfig.config.identityFile || '', '密钥路径')
                            }
                          />
                        )}
                      </div>
                    </div>
                  </div>

                  {sshConfig.config.proxyCommand && (
                    <div className="mt-4">
                      <Text type="secondary">代理命令</Text>
                      <div className="bg-gray-100 p-2 rounded mt-1 font-mono text-sm">
                        {sshConfig.config.proxyCommand}
                      </div>
                    </div>
                  )}
                </div>
              ),
            },
          ]}
        />
      </Spin>
    </Card>
  )
}

// Helper functions for mock data
function generateMockConfig(envId: string, envName: string, ideType: IDEType, keyPath: string): string {
  const hostAlias = `devbox-${envId.slice(0, 8)}`
  const hostname = `${hostAlias}.devbox.com`

  let config = `# DevBox Environment: ${envName}\n`
  config += `Host ${hostAlias}\n`
  config += `    HostName ${hostname}\n`
  config += `    Port 22\n`
  config += `    User devbox\n`
  config += `    IdentityFile ${keyPath}\n`
  config += `    StrictHostKeyChecking no\n`
  config += `    UserKnownHostsFile /dev/null\n`
  config += `    ServerAliveInterval 60\n`
  config += `    ServerAliveCountMax 3\n`

  if (ideType === 'vscode') {
    config += `    ForwardAgent yes\n`
    config += `    AddKeysToAgent yes\n`
  } else if (ideType === 'jetbrains') {
    config += `    Compression yes\n`
    config += `    TCPKeepAlive yes\n`
    config += `    ControlMaster auto\n`
    config += `    ControlPath ~/.ssh/sockets/%r@%h-%p\n`
    config += `    ControlPersist 600\n`
  }

  return config
}

function generateMockInstructions(envId: string, ideType: IDEType): string {
  const hostAlias = `devbox-${envId.slice(0, 8)}`

  if (ideType === 'vscode') {
    return `## VSCode Remote SSH 设置

1. 在 VSCode 中安装 "Remote - SSH" 扩展
2. 将上方 SSH 配置复制到 ~/.ssh/config 文件
3. 按 Ctrl+Shift+P (Mac: Cmd+Shift+P) 选择 "Remote-SSH: Connect to Host"
4. 从列表中选择 "${hostAlias}"
5. VSCode 将连接到您的 DevBox 环境`
  }

  if (ideType === 'jetbrains') {
    return `## JetBrains Gateway 设置

1. 下载并安装 JetBrains Gateway
2. 将上方 SSH 配置复制到 ~/.ssh/config 文件
3. 创建 sockets 目录: mkdir -p ~/.ssh/sockets
4. 打开 JetBrains Gateway
5. 点击 "Connect via SSH"
6. 选择 "${hostAlias}"`
  }

  return `## SSH 连接说明

1. 将上方 SSH 配置复制到 ~/.ssh/config 文件
2. 使用命令连接: ssh ${hostAlias}`
}
