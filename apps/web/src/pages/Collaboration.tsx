/**
 * Collaboration Page
 * Main page for real-time collaborative editing
 * Requirements: 4.1, 4.2, 4.4, 4.5.4 - Real-time collaboration with multi-user support and meeting integration
 */

import { useState, useEffect } from 'react'
import { useParams, useNavigate, useSearchParams } from 'react-router-dom'
import {
  Layout,
  Card,
  Button,
  Space,
  Spin,
  Alert,
  message,
  Breadcrumb,
  Tooltip,
  Dropdown,
  Tree,
  Empty,
  Modal,
  Form,
  Input,
} from 'antd'
import {
  ArrowLeftOutlined,
  UserAddOutlined,
  SettingOutlined,
  FolderOutlined,
  FileOutlined,
  FileTextOutlined,
  CodeOutlined,
  SaveOutlined,
  FullscreenOutlined,
  FullscreenExitOutlined,
  MoreOutlined,
  VideoCameraOutlined,
  PhoneOutlined,
} from '@ant-design/icons'
import CollaborativeEditor from '@/components/CollaborativeEditor'
import CollaboratorsPanel, { CollaboratorsAvatarGroup } from '@/components/CollaboratorsPanel'
import CollaborationInvite from '@/components/CollaborationInvite'
import MeetingCodeIntegration from '@/components/MeetingCodeIntegration'
import { useCollaborationStore } from '@/stores/collaboration'
import { useMeetingStore } from '@/stores/meeting'
import { environmentService } from '@/services/environment'
import { meetingService } from '@/services/meeting'
import type { Environment } from '@/stores/environment'
import type { DataNode } from 'antd/es/tree'

const { Sider, Content } = Layout

// Mock file tree data
const mockFileTree: DataNode[] = [
  {
    title: 'src',
    key: 'src',
    icon: <FolderOutlined />,
    children: [
      {
        title: 'components',
        key: 'src/components',
        icon: <FolderOutlined />,
        children: [
          { title: 'App.tsx', key: 'src/components/App.tsx', icon: <CodeOutlined />, isLeaf: true },
          { title: 'Header.tsx', key: 'src/components/Header.tsx', icon: <CodeOutlined />, isLeaf: true },
          { title: 'Footer.tsx', key: 'src/components/Footer.tsx', icon: <CodeOutlined />, isLeaf: true },
        ],
      },
      {
        title: 'pages',
        key: 'src/pages',
        icon: <FolderOutlined />,
        children: [
          { title: 'Home.tsx', key: 'src/pages/Home.tsx', icon: <CodeOutlined />, isLeaf: true },
          { title: 'About.tsx', key: 'src/pages/About.tsx', icon: <CodeOutlined />, isLeaf: true },
        ],
      },
      { title: 'index.tsx', key: 'src/index.tsx', icon: <CodeOutlined />, isLeaf: true },
      { title: 'styles.css', key: 'src/styles.css', icon: <FileTextOutlined />, isLeaf: true },
    ],
  },
  {
    title: 'public',
    key: 'public',
    icon: <FolderOutlined />,
    children: [
      { title: 'index.html', key: 'public/index.html', icon: <FileOutlined />, isLeaf: true },
      { title: 'favicon.ico', key: 'public/favicon.ico', icon: <FileOutlined />, isLeaf: true },
    ],
  },
  { title: 'package.json', key: 'package.json', icon: <FileTextOutlined />, isLeaf: true },
  { title: 'tsconfig.json', key: 'tsconfig.json', icon: <FileTextOutlined />, isLeaf: true },
  { title: 'README.md', key: 'README.md', icon: <FileTextOutlined />, isLeaf: true },
]

// Mock file content
const mockFileContents: Record<string, string> = {
  'src/components/App.tsx': `import React from 'react';
import Header from './Header';
import Footer from './Footer';

function App() {
  return (
    <div className="app">
      <Header />
      <main>
        <h1>Welcome to Cloud DevBox</h1>
        <p>Start editing to see real-time collaboration!</p>
      </main>
      <Footer />
    </div>
  );
}

export default App;
`,
  'src/index.tsx': `import React from 'react';
import ReactDOM from 'react-dom/client';
import App from './components/App';
import './styles.css';

const root = ReactDOM.createRoot(
  document.getElementById('root') as HTMLElement
);

root.render(
  <React.StrictMode>
    <App />
  </React.StrictMode>
);
`,
  'package.json': `{
  "name": "my-react-app",
  "version": "1.0.0",
  "private": true,
  "dependencies": {
    "react": "^18.2.0",
    "react-dom": "^18.2.0",
    "typescript": "^5.0.0"
  },
  "scripts": {
    "start": "react-scripts start",
    "build": "react-scripts build",
    "test": "react-scripts test"
  }
}
`,
}

export default function Collaboration() {
  const { id: environmentId } = useParams<{ id: string }>()
  const [searchParams] = useSearchParams()
  const navigate = useNavigate()

  const [environment, setEnvironment] = useState<Environment | null>(null)
  const [isLoading, setIsLoading] = useState(true)
  const [selectedFile, setSelectedFile] = useState<string | null>(null)
  const [fileContent, setFileContent] = useState<string>('')
  const [isInviteModalOpen, setIsInviteModalOpen] = useState(false)
  const [isFullscreen, setIsFullscreen] = useState(false)
  const [expandedKeys, setExpandedKeys] = useState<string[]>(['src', 'src/components'])
  const [currentLine, setCurrentLine] = useState<number | undefined>(undefined)
  const [showStartMeetingModal, setShowStartMeetingModal] = useState(false)
  const [activeMeetingId, setActiveMeetingId] = useState<string | null>(null)

  const [meetingForm] = Form.useForm()

  const {
    isConnected,
    sessionId,
  } = useCollaborationStore()

  const {
    isInMeeting,
  } = useMeetingStore()

  // Load environment data
  useEffect(() => {
    if (environmentId) {
      loadEnvironment()
      checkActiveMeeting()
    }
  }, [environmentId])

  // Check for active meeting in this environment
  const checkActiveMeeting = async () => {
    if (!environmentId) return
    try {
      const activeMeeting = await meetingService.getActiveMeetingForEnvironment(environmentId)
      if (activeMeeting) {
        setActiveMeetingId(activeMeeting.id)
      }
    } catch {
      // No active meeting
    }
  }

  // Load environment data
  useEffect(() => {
    if (environmentId) {
      loadEnvironment()
    }
  }, [environmentId])

  // Auto-select file from URL params
  useEffect(() => {
    const file = searchParams.get('file')
    if (file) {
      setSelectedFile(file)
      loadFileContent(file)
    } else {
      // Default to App.tsx
      setSelectedFile('src/components/App.tsx')
      loadFileContent('src/components/App.tsx')
    }
  }, [searchParams])

  const loadEnvironment = async () => {
    setIsLoading(true)
    try {
      if (environmentId) {
        const env = await environmentService.get(environmentId)
        setEnvironment(env)
      }
    } catch {
      // Use mock data
      setEnvironment({
        id: environmentId || '1',
        name: 'react-app',
        description: 'React 前端项目',
        templateId: 't1',
        templateName: 'React 18',
        status: 'running',
        resources: { cpu: '2 核', memory: '4 GB', storage: '20 GB' },
        createdAt: new Date().toISOString(),
      })
    } finally {
      setIsLoading(false)
    }
  }

  const loadFileContent = (filePath: string) => {
    // In production, this would fetch from the backend
    const content = mockFileContents[filePath] || `// File: ${filePath}\n// Content not available`
    setFileContent(content)
  }

  const handleFileSelect = (selectedKeys: React.Key[]) => {
    const key = selectedKeys[0] as string
    if (key && (!key.includes('/') || mockFileContents[key] !== undefined)) {
      setSelectedFile(key)
      loadFileContent(key)
    }
  }

  const onTreeExpand = (keys: React.Key[]) => {
    setExpandedKeys(keys as string[])
  }

  const handleContentChange = (content: string) => {
    setFileContent(content)
  }

  const handleSave = async (content: string) => {
    // In production, this would save to the backend
    console.log('Saving file:', selectedFile, content)
    message.success('文件已保存')
  }

  const toggleFullscreen = () => {
    setIsFullscreen(!isFullscreen)
  }

  // Meeting functions
  const handleStartMeeting = async (values: { title: string }) => {
    if (!environmentId) return
    try {
      const newMeeting = await meetingService.createMeeting({
        environmentId,
        title: values.title,
        type: 'video',
      })
      setActiveMeetingId(newMeeting.id)
      setShowStartMeetingModal(false)
      message.success('会议已创建')
      // Navigate to meeting page
      navigate(`/meeting/${newMeeting.id}?join=true`)
    } catch {
      message.error('创建会议失败')
    }
  }

  const handleJoinMeeting = () => {
    if (activeMeetingId) {
      navigate(`/meeting/${activeMeetingId}?join=true`)
    }
  }

  const handleCodeHighlight = (highlight: { startLine: number }) => {
    // Handle code highlight from meeting
    setCurrentLine(highlight.startLine)
  }

  const handleFollowUser = (_userId: string, filePath: string, line: number) => {
    // Follow user to their location
    if (filePath !== selectedFile) {
      setSelectedFile(filePath)
      loadFileContent(filePath)
    }
    setCurrentLine(line)
  }

  const getLanguageFromFile = (filePath: string): string => {
    const ext = filePath.split('.').pop()?.toLowerCase()
    const languageMap: Record<string, string> = {
      ts: 'typescript',
      tsx: 'typescript',
      js: 'javascript',
      jsx: 'javascript',
      json: 'json',
      css: 'css',
      scss: 'scss',
      html: 'html',
      md: 'markdown',
      py: 'python',
      go: 'go',
      rs: 'rust',
      java: 'java',
    }
    return languageMap[ext || ''] || 'plaintext'
  }

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-screen">
        <Spin size="large" tip="加载中..." />
      </div>
    )
  }

  if (!environment) {
    return (
      <div className="flex items-center justify-center h-screen">
        <Alert
          type="error"
          message="环境不存在"
          description="请检查环境 ID 是否正确"
          action={
            <Button onClick={() => navigate('/environments')}>
              返回环境列表
            </Button>
          }
        />
      </div>
    )
  }

  return (
    <Layout style={{ height: '100vh', background: '#f0f2f5' }}>
      {/* Header */}
      <div
        style={{
          height: 48,
          background: '#fff',
          borderBottom: '1px solid #f0f0f0',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          padding: '0 16px',
        }}
      >
        <Space>
          <Button
            type="text"
            icon={<ArrowLeftOutlined />}
            onClick={() => navigate(`/environments/${environmentId}`)}
          >
            返回
          </Button>
          <Breadcrumb
            items={[
              { title: '环境' },
              { title: environment.name },
              { title: '协作编辑' },
            ]}
          />
        </Space>

        <Space>
          {/* Meeting integration */}
          <MeetingCodeIntegration
            environmentId={environmentId || ''}
            currentFilePath={selectedFile || ''}
            currentLine={currentLine}
            onHighlightCode={handleCodeHighlight}
            onFollowUser={handleFollowUser}
            onJumpToLine={(line) => setCurrentLine(line)}
          />

          {/* Meeting buttons */}
          {!isInMeeting && (
            activeMeetingId ? (
              <Button
                type="primary"
                icon={<PhoneOutlined />}
                onClick={handleJoinMeeting}
              >
                加入会议
              </Button>
            ) : (
              <Button
                icon={<VideoCameraOutlined />}
                onClick={() => setShowStartMeetingModal(true)}
              >
                发起会议
              </Button>
            )
          )}

          {/* Online collaborators */}
          <CollaboratorsAvatarGroup maxCount={4} />

          {/* Connection status */}
          <Tooltip title={isConnected ? '已连接' : '未连接'}>
            <span
              style={{
                display: 'inline-block',
                width: 8,
                height: 8,
                borderRadius: '50%',
                background: isConnected ? '#52c41a' : '#ff4d4f',
              }}
            />
          </Tooltip>

          {/* Invite button */}
          <Button
            icon={<UserAddOutlined />}
            onClick={() => setIsInviteModalOpen(true)}
          >
            邀请
          </Button>

          {/* Fullscreen toggle */}
          <Tooltip title={isFullscreen ? '退出全屏' : '全屏'}>
            <Button
              type="text"
              icon={isFullscreen ? <FullscreenExitOutlined /> : <FullscreenOutlined />}
              onClick={toggleFullscreen}
            />
          </Tooltip>

          {/* More options */}
          <Dropdown
            menu={{
              items: [
                { key: 'settings', label: '协作设置', icon: <SettingOutlined /> },
                { key: 'save', label: '保存所有', icon: <SaveOutlined /> },
              ],
            }}
          >
            <Button type="text" icon={<MoreOutlined />} />
          </Dropdown>
        </Space>
      </div>

      <Layout style={{ height: 'calc(100vh - 48px)' }}>
        {/* File tree sidebar */}
        {!isFullscreen && (
          <Sider
            width={250}
            style={{
              background: '#fff',
              borderRight: '1px solid #f0f0f0',
              overflow: 'auto',
            }}
          >
            <div style={{ padding: '12px 16px', borderBottom: '1px solid #f0f0f0' }}>
              <Space>
                <FolderOutlined />
                <span style={{ fontWeight: 500 }}>文件</span>
              </Space>
            </div>
            <Tree
              showIcon
              defaultExpandAll={false}
              expandedKeys={expandedKeys}
              onExpand={onTreeExpand}
              selectedKeys={selectedFile ? [selectedFile] : []}
              onSelect={handleFileSelect}
              treeData={mockFileTree}
              style={{ padding: '8px 0' }}
            />
          </Sider>
        )}

        {/* Main editor area */}
        <Content style={{ display: 'flex', flexDirection: 'column' }}>
          {/* File tabs / current file indicator */}
          {selectedFile && (
            <div
              style={{
                height: 36,
                background: '#fff',
                borderBottom: '1px solid #f0f0f0',
                display: 'flex',
                alignItems: 'center',
                padding: '0 16px',
              }}
            >
              <Space>
                <CodeOutlined />
                <span>{selectedFile}</span>
              </Space>
            </div>
          )}

          {/* Editor */}
          <div style={{ flex: 1, padding: isFullscreen ? 0 : 16 }}>
            {selectedFile ? (
              <CollaborativeEditor
                environmentId={environmentId || ''}
                filePath={selectedFile}
                initialContent={fileContent}
                language={getLanguageFromFile(selectedFile)}
                onContentChange={handleContentChange}
                onSave={handleSave}
                height="100%"
              />
            ) : (
              <Card style={{ height: '100%' }}>
                <Empty description="选择一个文件开始编辑" />
              </Card>
            )}
          </div>
        </Content>

        {/* Collaborators sidebar */}
        {!isFullscreen && (
          <Sider
            width={280}
            style={{
              background: '#fff',
              borderLeft: '1px solid #f0f0f0',
              overflow: 'auto',
              padding: 16,
            }}
          >
            <CollaboratorsPanel
              sessionId={sessionId || undefined}
              showOffline
              onInviteClick={() => setIsInviteModalOpen(true)}
            />
          </Sider>
        )}
      </Layout>

      {/* Invite modal */}
      <CollaborationInvite
        visible={isInviteModalOpen}
        onClose={() => setIsInviteModalOpen(false)}
        environmentId={environmentId || ''}
        environmentName={environment.name}
      />

      {/* Start meeting modal */}
      <Modal
        title="发起会议"
        open={showStartMeetingModal}
        onCancel={() => setShowStartMeetingModal(false)}
        footer={null}
      >
        <Form
          form={meetingForm}
          layout="vertical"
          onFinish={handleStartMeeting}
          initialValues={{ title: `${environment.name} 代码评审` }}
        >
          <Form.Item
            name="title"
            label="会议标题"
            rules={[{ required: true, message: '请输入会议标题' }]}
          >
            <Input placeholder="输入会议标题" />
          </Form.Item>
          <Form.Item style={{ marginBottom: 0 }}>
            <Space style={{ width: '100%', justifyContent: 'flex-end' }}>
              <Button onClick={() => setShowStartMeetingModal(false)}>
                取消
              </Button>
              <Button type="primary" htmlType="submit" icon={<VideoCameraOutlined />}>
                开始会议
              </Button>
            </Space>
          </Form.Item>
        </Form>
      </Modal>
    </Layout>
  )
}
