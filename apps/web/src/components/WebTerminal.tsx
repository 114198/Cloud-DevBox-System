import { useEffect, useRef, useState, useCallback } from 'react'
import { Card, Button, Space, Tooltip, Dropdown, message, Spin, Alert } from 'antd'
import {
  ExpandOutlined,
  CompressOutlined,
  ClearOutlined,
  CopyOutlined,
  SearchOutlined,
  ReloadOutlined,
  CloseOutlined,
  SettingOutlined,
} from '@ant-design/icons'
import type { MenuProps } from 'antd'

// Terminal types (xterm.js will be dynamically imported)
interface ITerminal {
  open: (container: HTMLElement) => void
  write: (data: string) => void
  writeln: (data: string) => void
  clear: () => void
  focus: () => void
  dispose: () => void
  onData: (callback: (data: string) => void) => { dispose: () => void }
  onResize: (callback: (size: { cols: number; rows: number }) => void) => { dispose: () => void }
  loadAddon: (addon: unknown) => void
  select: (column: number, row: number, length: number) => void
  getSelection: () => string
  cols: number
  rows: number
}

interface WebTerminalProps {
  environmentId: string
  environmentName: string
  isRunning: boolean
  onClose?: () => void
}

type ConnectionStatus = 'disconnected' | 'connecting' | 'connected' | 'error'

export default function WebTerminal({
  environmentId,
  environmentName,
  isRunning,
  onClose,
}: WebTerminalProps) {
  const terminalRef = useRef<HTMLDivElement>(null)
  const terminalInstance = useRef<ITerminal | null>(null)
  const wsRef = useRef<WebSocket | null>(null)
  const fitAddonRef = useRef<{ fit: () => void } | null>(null)
  const searchAddonRef = useRef<{ findNext: (term: string) => boolean } | null>(null)

  const [isFullscreen, setIsFullscreen] = useState(false)
  const [connectionStatus, setConnectionStatus] = useState<ConnectionStatus>('disconnected')
  const [reconnectAttempts, setReconnectAttempts] = useState(0)

  const MAX_RECONNECT_ATTEMPTS = 3
  const RECONNECT_DELAY = 2000

  // Initialize terminal
  const initTerminal = useCallback(async () => {
    if (!terminalRef.current || terminalInstance.current) return

    try {
      // Dynamic import of xterm.js
      const { Terminal } = await import('@xterm/xterm')
      const { FitAddon } = await import('@xterm/addon-fit')
      const { WebLinksAddon } = await import('@xterm/addon-web-links')
      const { SearchAddon } = await import('@xterm/addon-search')

      // Import CSS
      await import('@xterm/xterm/css/xterm.css')

      const terminal = new Terminal({
        cursorBlink: true,
        cursorStyle: 'block',
        fontSize: 14,
        fontFamily: 'Menlo, Monaco, "Courier New", monospace',
        theme: {
          background: '#1e1e1e',
          foreground: '#d4d4d4',
          cursor: '#d4d4d4',
          cursorAccent: '#1e1e1e',
          selectionBackground: '#264f78',
          black: '#000000',
          red: '#cd3131',
          green: '#0dbc79',
          yellow: '#e5e510',
          blue: '#2472c8',
          magenta: '#bc3fbc',
          cyan: '#11a8cd',
          white: '#e5e5e5',
          brightBlack: '#666666',
          brightRed: '#f14c4c',
          brightGreen: '#23d18b',
          brightYellow: '#f5f543',
          brightBlue: '#3b8eea',
          brightMagenta: '#d670d6',
          brightCyan: '#29b8db',
          brightWhite: '#ffffff',
        },
        scrollback: 10000,
        convertEol: true,
      })

      // Load addons
      const fitAddon = new FitAddon()
      const webLinksAddon = new WebLinksAddon()
      const searchAddon = new SearchAddon()

      terminal.loadAddon(fitAddon)
      terminal.loadAddon(webLinksAddon)
      terminal.loadAddon(searchAddon)

      fitAddonRef.current = fitAddon
      searchAddonRef.current = searchAddon

      // Open terminal in container
      terminal.open(terminalRef.current)
      fitAddon.fit()

      terminalInstance.current = terminal as unknown as ITerminal

      // Handle window resize
      const handleResize = () => {
        if (fitAddonRef.current) {
          fitAddonRef.current.fit()
        }
      }
      window.addEventListener('resize', handleResize)

      // Connect to WebSocket
      connectWebSocket()

      return () => {
        window.removeEventListener('resize', handleResize)
      }
    } catch (error) {
      console.error('Failed to initialize terminal:', error)
      message.error('终端初始化失败')
    }
  }, [environmentId])

  // Connect to WebSocket
  const connectWebSocket = useCallback(() => {
    if (!terminalInstance.current) return

    setConnectionStatus('connecting')

    const wsUrl = `${window.location.protocol === 'https:' ? 'wss:' : 'ws:'}//${window.location.host}/api/v1/environments/${environmentId}/terminal`

    try {
      const ws = new WebSocket(wsUrl)
      wsRef.current = ws

      ws.onopen = () => {
        setConnectionStatus('connected')
        setReconnectAttempts(0)
        terminalInstance.current?.writeln('\x1b[32m✓ 已连接到 DevBox 终端\x1b[0m')
        terminalInstance.current?.writeln(`\x1b[90m环境: ${environmentName}\x1b[0m`)
        terminalInstance.current?.writeln('')

        // Send terminal size
        if (terminalInstance.current) {
          ws.send(
            JSON.stringify({
              type: 'resize',
              cols: terminalInstance.current.cols,
              rows: terminalInstance.current.rows,
            })
          )
        }
      }

      ws.onmessage = (event) => {
        if (terminalInstance.current) {
          terminalInstance.current.write(event.data)
        }
      }

      ws.onerror = (error) => {
        console.error('WebSocket error:', error)
        setConnectionStatus('error')
      }

      ws.onclose = (event) => {
        setConnectionStatus('disconnected')
        if (!event.wasClean && reconnectAttempts < MAX_RECONNECT_ATTEMPTS) {
          terminalInstance.current?.writeln(
            `\x1b[33m⚠ 连接断开，${RECONNECT_DELAY / 1000}秒后重试...\x1b[0m`
          )
          setTimeout(() => {
            setReconnectAttempts((prev: number) => prev + 1)
            connectWebSocket()
          }, RECONNECT_DELAY)
        } else if (reconnectAttempts >= MAX_RECONNECT_ATTEMPTS) {
          terminalInstance.current?.writeln('\x1b[31m✗ 连接失败，请刷新页面重试\x1b[0m')
        }
      }

      // Handle terminal input
      terminalInstance.current?.onData((data: string) => {
        if (ws.readyState === WebSocket.OPEN) {
          ws.send(JSON.stringify({ type: 'input', data }))
        }
      })

      // Handle terminal resize
      terminalInstance.current?.onResize(({ cols, rows }: { cols: number; rows: number }) => {
        if (ws.readyState === WebSocket.OPEN) {
          ws.send(JSON.stringify({ type: 'resize', cols, rows }))
        }
      })
    } catch (error) {
      console.error('Failed to connect WebSocket:', error)
      setConnectionStatus('error')
      // Show demo mode
      showDemoMode()
    }
  }, [environmentId, environmentName, reconnectAttempts])

  // Show demo mode when WebSocket is not available
  const showDemoMode = useCallback(() => {
    if (!terminalInstance.current) return

    terminalInstance.current.writeln('\x1b[33m⚠ 演示模式 - WebSocket 连接不可用\x1b[0m')
    terminalInstance.current.writeln('')
    terminalInstance.current.writeln('\x1b[90m在生产环境中，此终端将连接到您的 DevBox 环境。\x1b[0m')
    terminalInstance.current.writeln('\x1b[90m您可以在此执行命令、查看日志等操作。\x1b[0m')
    terminalInstance.current.writeln('')
    terminalInstance.current.write('\x1b[32mdevbox@' + environmentName + '\x1b[0m:\x1b[34m~\x1b[0m$ ')

    // Simple echo for demo
    let inputBuffer = ''
    terminalInstance.current.onData((data: string) => {
      if (data === '\r') {
        // Enter key
        terminalInstance.current?.writeln('')
        if (inputBuffer.trim()) {
          handleDemoCommand(inputBuffer.trim())
        }
        inputBuffer = ''
        terminalInstance.current?.write(
          '\x1b[32mdevbox@' + environmentName + '\x1b[0m:\x1b[34m~\x1b[0m$ '
        )
      } else if (data === '\x7f') {
        // Backspace
        if (inputBuffer.length > 0) {
          inputBuffer = inputBuffer.slice(0, -1)
          terminalInstance.current?.write('\b \b')
        }
      } else if (data >= ' ') {
        inputBuffer += data
        terminalInstance.current?.write(data)
      }
    })
  }, [environmentName])

  // Handle demo commands
  const handleDemoCommand = (command: string) => {
    const terminal = terminalInstance.current
    if (!terminal) return

    const cmd = command.toLowerCase().split(' ')[0]

    switch (cmd) {
      case 'help':
        terminal.writeln('可用命令 (演示模式):')
        terminal.writeln('  help     - 显示帮助信息')
        terminal.writeln('  ls       - 列出文件')
        terminal.writeln('  pwd      - 显示当前目录')
        terminal.writeln('  whoami   - 显示当前用户')
        terminal.writeln('  date     - 显示日期时间')
        terminal.writeln('  clear    - 清屏')
        terminal.writeln('  echo     - 输出文本')
        break
      case 'ls':
        terminal.writeln('\x1b[34mnode_modules\x1b[0m  package.json  src  \x1b[34mpublic\x1b[0m  README.md')
        break
      case 'pwd':
        terminal.writeln('/home/devbox/project')
        break
      case 'whoami':
        terminal.writeln('devbox')
        break
      case 'date':
        terminal.writeln(new Date().toString())
        break
      case 'clear':
        terminal.clear()
        break
      case 'echo':
        terminal.writeln(command.slice(5))
        break
      default:
        terminal.writeln(`\x1b[31m命令未找到: ${cmd}\x1b[0m`)
        terminal.writeln('输入 "help" 查看可用命令')
    }
  }

  // Initialize on mount
  useEffect(() => {
    if (isRunning) {
      initTerminal()
    }

    return () => {
      wsRef.current?.close()
      terminalInstance.current?.dispose()
      terminalInstance.current = null
    }
  }, [isRunning, initTerminal])

  // Fit terminal when fullscreen changes
  useEffect(() => {
    setTimeout(() => {
      fitAddonRef.current?.fit()
    }, 100)
  }, [isFullscreen])

  // Actions
  const handleClear = () => {
    terminalInstance.current?.clear()
  }

  const handleCopy = () => {
    const selection = terminalInstance.current?.getSelection()
    if (selection) {
      navigator.clipboard.writeText(selection)
      message.success('已复制到剪贴板')
    } else {
      message.info('请先选择要复制的文本')
    }
  }

  const handleSearch = () => {
    const term = prompt('搜索:')
    if (term && searchAddonRef.current) {
      searchAddonRef.current.findNext(term)
    }
  }

  const handleReconnect = () => {
    wsRef.current?.close()
    setReconnectAttempts(0)
    terminalInstance.current?.clear()
    connectWebSocket()
  }

  const handleFullscreen = () => {
    setIsFullscreen(!isFullscreen)
  }

  const settingsMenu: MenuProps['items'] = [
    {
      key: 'fontSize',
      label: '字体大小',
      children: [
        { key: 'small', label: '小 (12px)' },
        { key: 'medium', label: '中 (14px)' },
        { key: 'large', label: '大 (16px)' },
      ],
    },
    {
      key: 'theme',
      label: '主题',
      children: [
        { key: 'dark', label: '深色' },
        { key: 'light', label: '浅色' },
      ],
    },
  ]

  if (!isRunning) {
    return (
      <Card>
        <Alert
          message="环境未运行"
          description="请先启动环境后再使用 Web 终端"
          type="warning"
          showIcon
        />
      </Card>
    )
  }

  const getStatusColor = () => {
    switch (connectionStatus) {
      case 'connected':
        return 'green'
      case 'connecting':
        return 'blue'
      case 'error':
        return 'red'
      default:
        return 'default'
    }
  }

  const getStatusText = () => {
    switch (connectionStatus) {
      case 'connected':
        return '已连接'
      case 'connecting':
        return '连接中...'
      case 'error':
        return '连接错误'
      default:
        return '未连接'
    }
  }

  return (
    <Card
      title={
        <Space>
          <span>Web 终端</span>
          <span
            className="inline-block w-2 h-2 rounded-full"
            style={{ backgroundColor: getStatusColor() }}
          />
          <span className="text-sm text-gray-500">{getStatusText()}</span>
        </Space>
      }
      extra={
        <Space>
          <Tooltip title="搜索">
            <Button icon={<SearchOutlined />} size="small" onClick={handleSearch} />
          </Tooltip>
          <Tooltip title="复制选中">
            <Button icon={<CopyOutlined />} size="small" onClick={handleCopy} />
          </Tooltip>
          <Tooltip title="清屏">
            <Button icon={<ClearOutlined />} size="small" onClick={handleClear} />
          </Tooltip>
          <Tooltip title="重新连接">
            <Button
              icon={<ReloadOutlined />}
              size="small"
              onClick={handleReconnect}
              loading={connectionStatus === 'connecting'}
            />
          </Tooltip>
          <Dropdown menu={{ items: settingsMenu }} placement="bottomRight">
            <Button icon={<SettingOutlined />} size="small" />
          </Dropdown>
          <Tooltip title={isFullscreen ? '退出全屏' : '全屏'}>
            <Button
              icon={isFullscreen ? <CompressOutlined /> : <ExpandOutlined />}
              size="small"
              onClick={handleFullscreen}
            />
          </Tooltip>
          {onClose && (
            <Tooltip title="关闭">
              <Button icon={<CloseOutlined />} size="small" onClick={onClose} />
            </Tooltip>
          )}
        </Space>
      }
      className={`web-terminal-card ${isFullscreen ? 'fixed inset-0 z-50 m-0 rounded-none' : ''}`}
      styles={{
        body: {
          padding: 0,
          height: isFullscreen ? 'calc(100vh - 57px)' : '400px',
          backgroundColor: '#1e1e1e',
        },
      }}
    >
      {connectionStatus === 'connecting' && (
        <div className="absolute inset-0 flex items-center justify-center bg-black bg-opacity-50 z-10">
          <Spin tip="正在连接..." />
        </div>
      )}
      <div
        ref={terminalRef}
        className="w-full h-full"
        style={{ padding: '8px' }}
      />
    </Card>
  )
}
