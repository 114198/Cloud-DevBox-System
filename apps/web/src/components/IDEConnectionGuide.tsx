import { useState } from 'react'
import {
  Card,
  Tabs,
  Steps,
  Typography,
  Button,
  Space,
  Alert,
  Collapse,
  Tag,
} from 'antd'
import {
  CodeOutlined,
  RocketOutlined,
  CheckCircleOutlined,
  LinkOutlined,
} from '@ant-design/icons'

const { Title, Text, Paragraph } = Typography
const { Panel } = Collapse

interface IDEConnectionGuideProps {
  environmentId: string
  environmentName: string
  sshHost?: string
}

export default function IDEConnectionGuide({
  environmentId,
}: IDEConnectionGuideProps) {
  const hostAlias = `devbox-${environmentId.slice(0, 8)}`

  return (
    <Card title="IDE 连接指引" className="ide-connection-guide">
      <Tabs
        defaultActiveKey="vscode"
        items={[
          {
            key: 'vscode',
            label: (
              <span>
                <CodeOutlined /> VSCode
              </span>
            ),
            children: <VSCodeGuide hostAlias={hostAlias} environmentName={environmentName} />,
          },
          {
            key: 'jetbrains',
            label: (
              <span>
                <RocketOutlined /> JetBrains
              </span>
            ),
            children: <JetBrainsGuide hostAlias={hostAlias} environmentName={environmentName} />,
          },
        ]}
      />
    </Card>
  )
}

interface GuideProps {
  hostAlias: string
}

function VSCodeGuide({ hostAlias }: GuideProps) {
  return (
    <div className="vscode-guide">
      <Alert
        message="推荐使用 VSCode Remote SSH"
        description="VSCode Remote SSH 扩展提供了最佳的远程开发体验，支持完整的 IDE 功能。"
        type="info"
        showIcon
        className="mb-4"
      />

      <Title level={5}>安装步骤</Title>
      <Steps
        direction="vertical"
        current={-1}
        items={[
          {
            title: '安装 Remote SSH 扩展',
            description: (
              <div className="py-2">
                <Paragraph>
                  在 VSCode 中打开扩展市场 (Ctrl+Shift+X)，搜索并安装{' '}
                  <Text strong>"Remote - SSH"</Text> 扩展。
                </Paragraph>
                <Button
                  type="link"
                  icon={<LinkOutlined />}
                  href="vscode:extension/ms-vscode-remote.remote-ssh"
                  target="_blank"
                >
                  一键安装扩展
                </Button>
              </div>
            ),
          },
          {
            title: '配置 SSH',
            description: (
              <div className="py-2">
                <Paragraph>
                  将 SSH 配置添加到您的配置文件中：
                </Paragraph>
                <div className="bg-gray-100 p-3 rounded font-mono text-sm mb-2">
                  <div>
                    <Text type="secondary">Windows:</Text>{' '}
                    <code>C:\Users\用户名\.ssh\config</code>
                  </div>
                  <div>
                    <Text type="secondary">macOS/Linux:</Text> <code>~/.ssh/config</code>
                  </div>
                </div>
              </div>
            ),
          },
          {
            title: '连接到 DevBox',
            description: (
              <div className="py-2">
                <Paragraph>
                  按 <Tag>Ctrl+Shift+P</Tag> (Mac: <Tag>Cmd+Shift+P</Tag>) 打开命令面板
                </Paragraph>
                <Paragraph>
                  输入并选择 <Text code>Remote-SSH: Connect to Host</Text>
                </Paragraph>
                <Paragraph>
                  从列表中选择 <Text strong>{hostAlias}</Text>
                </Paragraph>
              </div>
            ),
          },
          {
            title: '开始开发',
            description: (
              <div className="py-2">
                <Paragraph>
                  连接成功后，VSCode 会在远程环境中安装服务器组件。
                  完成后即可像本地开发一样使用所有功能。
                </Paragraph>
                <Space>
                  <Tag color="green">
                    <CheckCircleOutlined /> 完整 IntelliSense
                  </Tag>
                  <Tag color="green">
                    <CheckCircleOutlined /> 调试支持
                  </Tag>
                  <Tag color="green">
                    <CheckCircleOutlined /> 终端访问
                  </Tag>
                </Space>
              </div>
            ),
          },
        ]}
      />

      <Collapse className="mt-4" ghost>
        <Panel header="常见问题" key="faq">
          <VSCodeFAQ hostAlias={hostAlias} />
        </Panel>
        <Panel header="快捷键参考" key="shortcuts">
          <VSCodeShortcuts />
        </Panel>
      </Collapse>
    </div>
  )
}

function JetBrainsGuide({ hostAlias }: GuideProps) {
  return (
    <div className="jetbrains-guide">
      <Alert
        message="JetBrains Gateway 远程开发"
        description="JetBrains Gateway 支持所有 JetBrains IDE，包括 IntelliJ IDEA、PyCharm、WebStorm 等。"
        type="info"
        showIcon
        className="mb-4"
      />

      <Title level={5}>支持的 IDE</Title>
      <Space wrap className="mb-4">
        <Tag color="blue">IntelliJ IDEA</Tag>
        <Tag color="green">PyCharm</Tag>
        <Tag color="orange">WebStorm</Tag>
        <Tag color="purple">GoLand</Tag>
        <Tag color="cyan">CLion</Tag>
        <Tag color="magenta">PhpStorm</Tag>
        <Tag color="gold">RubyMine</Tag>
        <Tag color="lime">Rider</Tag>
      </Space>

      <Title level={5}>安装步骤</Title>
      <Steps
        direction="vertical"
        current={-1}
        items={[
          {
            title: '下载 JetBrains Gateway',
            description: (
              <div className="py-2">
                <Paragraph>
                  从 JetBrains 官网下载并安装 Gateway 客户端。
                </Paragraph>
                <Button
                  type="primary"
                  icon={<LinkOutlined />}
                  href="https://www.jetbrains.com/remote-development/gateway/"
                  target="_blank"
                >
                  下载 JetBrains Gateway
                </Button>
              </div>
            ),
          },
          {
            title: '配置 SSH',
            description: (
              <div className="py-2">
                <Paragraph>
                  将 SSH 配置添加到配置文件，并创建 sockets 目录：
                </Paragraph>
                <div className="bg-gray-900 text-green-400 p-3 rounded font-mono text-sm mb-2">
                  mkdir -p ~/.ssh/sockets
                </div>
                <Paragraph type="secondary">
                  sockets 目录用于 SSH 连接复用，提高连接速度。
                </Paragraph>
              </div>
            ),
          },
          {
            title: '连接到 DevBox',
            description: (
              <div className="py-2">
                <Paragraph>打开 JetBrains Gateway</Paragraph>
                <Paragraph>
                  点击 <Text strong>"Connect via SSH"</Text>
                </Paragraph>
                <Paragraph>
                  选择 <Text strong>{hostAlias}</Text> 或手动输入连接信息
                </Paragraph>
              </div>
            ),
          },
          {
            title: '选择 IDE 和项目',
            description: (
              <div className="py-2">
                <Paragraph>选择您要使用的 JetBrains IDE</Paragraph>
                <Paragraph>Gateway 会自动在远程环境安装 IDE 后端</Paragraph>
                <Paragraph>选择或创建项目目录开始开发</Paragraph>
              </div>
            ),
          },
        ]}
      />

      <Collapse className="mt-4" ghost>
        <Panel header="常见问题" key="faq">
          <JetBrainsFAQ hostAlias={hostAlias} />
        </Panel>
        <Panel header="性能优化建议" key="performance">
          <JetBrainsPerformance />
        </Panel>
      </Collapse>
    </div>
  )
}

function VSCodeFAQ({ hostAlias }: { hostAlias: string }) {
  return (
    <div className="space-y-4">
      <div>
        <Text strong>Q: 连接超时怎么办？</Text>
        <Paragraph type="secondary">
          A: 检查网络连接，确保 SSH 密钥配置正确。可以先在终端测试：
          <code className="block bg-gray-100 p-2 mt-1 rounded">ssh {hostAlias}</code>
        </Paragraph>
      </div>
      <div>
        <Text strong>Q: 扩展在远程环境不工作？</Text>
        <Paragraph type="secondary">
          A: 部分扩展需要在远程环境重新安装。点击扩展图标，选择"在 SSH: {hostAlias} 中安装"。
        </Paragraph>
      </div>
      <div>
        <Text strong>Q: 如何转发端口？</Text>
        <Paragraph type="secondary">
          A: VSCode 会自动检测并转发端口。也可以在"端口"面板手动添加转发规则。
        </Paragraph>
      </div>
    </div>
  )
}

function VSCodeShortcuts() {
  return (
    <div className="grid grid-cols-2 gap-2">
      <div className="bg-gray-50 p-2 rounded">
        <Tag>Ctrl+Shift+P</Tag>
        <Text className="ml-2">命令面板</Text>
      </div>
      <div className="bg-gray-50 p-2 rounded">
        <Tag>Ctrl+`</Tag>
        <Text className="ml-2">打开终端</Text>
      </div>
      <div className="bg-gray-50 p-2 rounded">
        <Tag>Ctrl+Shift+E</Tag>
        <Text className="ml-2">文件浏览器</Text>
      </div>
      <div className="bg-gray-50 p-2 rounded">
        <Tag>Ctrl+Shift+G</Tag>
        <Text className="ml-2">Git 面板</Text>
      </div>
      <div className="bg-gray-50 p-2 rounded">
        <Tag>F5</Tag>
        <Text className="ml-2">开始调试</Text>
      </div>
      <div className="bg-gray-50 p-2 rounded">
        <Tag>Ctrl+Shift+F</Tag>
        <Text className="ml-2">全局搜索</Text>
      </div>
    </div>
  )
}

function JetBrainsFAQ({ hostAlias: _hostAlias }: { hostAlias: string }) {
  return (
    <div className="space-y-4">
      <div>
        <Text strong>Q: IDE 后端安装失败？</Text>
        <Paragraph type="secondary">
          A: 确保远程环境有足够的磁盘空间（至少 2GB）和内存（至少 2GB）。
        </Paragraph>
      </div>
      <div>
        <Text strong>Q: 连接断开后如何重连？</Text>
        <Paragraph type="secondary">
          A: Gateway 会保存连接历史，直接点击之前的连接即可重连。
        </Paragraph>
      </div>
      <div>
        <Text strong>Q: 如何更新远程 IDE？</Text>
        <Paragraph type="secondary">
          A: 在 Gateway 中选择连接，点击设置图标，选择"更新 IDE 后端"。
        </Paragraph>
      </div>
    </div>
  )
}

function JetBrainsPerformance() {
  return (
    <div className="space-y-2">
      <Alert
        message="性能优化建议"
        type="info"
        description={
          <ul className="list-disc list-inside mt-2">
            <li>启用 SSH 连接复用（已在配置中启用）</li>
            <li>使用 SSD 存储以提高索引速度</li>
            <li>为大型项目分配更多内存</li>
            <li>排除不需要索引的目录（node_modules、.git 等）</li>
            <li>使用有线网络连接以获得最佳体验</li>
          </ul>
        }
      />
    </div>
  )
}
