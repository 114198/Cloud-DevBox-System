/**
 * Meeting Code Integration Component
 * Integrates video meetings with collaborative code editing
 * Requirements: 4.5.4 - Code collaboration integration with meetings
 */

import { useState, useEffect, useCallback } from 'react'
import {
  Button,
  Tooltip,
  Badge,
  Popover,
  Switch,
  Space,
  Tag,
  Avatar,
  List,
  message,
} from 'antd'
import {
  VideoCameraOutlined,
  TeamOutlined,
  EyeOutlined,
  HighlightOutlined,
  AimOutlined,
  UserOutlined,
} from '@ant-design/icons'
import { useMeetingStore } from '@/stores/meeting'
import { useCollaborationStore } from '@/stores/collaboration'

// Code highlight data structure
export interface CodeHighlight {
  id: string
  userId: string
  userName: string
  userColor: string
  filePath: string
  startLine: number
  endLine: number
  comment?: string
  timestamp: number
}

// Follow mode state
export interface FollowModeState {
  enabled: boolean
  hostUserId: string | null
  hostUserName: string | null
}

interface MeetingCodeIntegrationProps {
  environmentId: string
  currentFilePath: string
  currentLine?: number
  onHighlightCode?: (highlight: CodeHighlight) => void
  onFollowUser?: (userId: string, filePath: string, line: number) => void
  onJumpToLine?: (line: number) => void
}

export default function MeetingCodeIntegration({
  currentFilePath,
  currentLine,
  onHighlightCode,
  onFollowUser,
  onJumpToLine,
}: MeetingCodeIntegrationProps) {
  const { isInMeeting, meeting, participants, signaling } = useMeetingStore()
  const { isConnected: isCollaborating, participants: collaborators } = useCollaborationStore()

  const [followMode, setFollowMode] = useState<FollowModeState>({
    enabled: false,
    hostUserId: null,
    hostUserName: null,
  })
  const [highlights, setHighlights] = useState<CodeHighlight[]>([])
  const [showHighlightPopover, setShowHighlightPopover] = useState(false)

  // Listen for code highlight messages from meeting
  useEffect(() => {
    if (!signaling) return

    const handleCodeHighlight = (data: CodeHighlight) => {
      setHighlights((prev) => [...prev.filter((h) => h.id !== data.id), data])
      onHighlightCode?.(data)
      
      // Auto-remove highlight after 30 seconds
      setTimeout(() => {
        setHighlights((prev) => prev.filter((h) => h.id !== data.id))
      }, 30000)
    }

    const handleFollowMode = (data: { enabled: boolean; hostUserId: string; hostUserName: string }) => {
      setFollowMode({
        enabled: data.enabled,
        hostUserId: data.hostUserId,
        hostUserName: data.hostUserName,
      })
      
      if (data.enabled) {
        message.info(`${data.hostUserName} 开启了跟随模式`)
      }
    }

    // These would be actual WebSocket message handlers
    // For now, we'll simulate with custom events
    const handleMessage = (event: CustomEvent) => {
      const { type, data } = event.detail
      if (type === 'code-highlight') {
        handleCodeHighlight(data)
      } else if (type === 'follow-mode') {
        handleFollowMode(data)
      } else if (type === 'follow-location') {
        if (followMode.enabled && data.userId === followMode.hostUserId) {
          onFollowUser?.(data.userId, data.filePath, data.line)
        }
      }
    }

    window.addEventListener('meeting-code-event' as any, handleMessage as EventListener)
    return () => {
      window.removeEventListener('meeting-code-event' as any, handleMessage as EventListener)
    }
  }, [signaling, followMode, onHighlightCode, onFollowUser])

  // Broadcast current location when in follow mode as host
  useEffect(() => {
    if (followMode.enabled && followMode.hostUserId === meeting?.hostUserId && currentLine) {
      broadcastLocation(currentFilePath, currentLine)
    }
  }, [currentFilePath, currentLine, followMode])

  const broadcastLocation = useCallback((filePath: string, line: number) => {
    if (!signaling) return
    
    signaling.send({
      type: 'media-state' as any, // Using existing message type for simplicity
      payload: {
        type: 'follow-location',
        filePath,
        line,
      },
    })
  }, [signaling])

  const highlightCurrentCode = useCallback((comment?: string) => {
    if (!currentLine || !meeting) return

    const highlight: CodeHighlight = {
      id: `${Date.now()}-${Math.random().toString(36).substr(2, 9)}`,
      userId: meeting.hostUserId,
      userName: participants.find((p) => p.userId === meeting.hostUserId)?.displayName || '未知用户',
      userColor: '#1890ff',
      filePath: currentFilePath,
      startLine: currentLine,
      endLine: currentLine,
      comment,
      timestamp: Date.now(),
    }

    // Broadcast to meeting participants
    if (signaling) {
      signaling.send({
        type: 'media-state' as any,
        payload: {
          type: 'code-highlight',
          ...highlight,
        },
      })
    }

    setHighlights((prev) => [...prev, highlight])
    onHighlightCode?.(highlight)
    message.success('已高亮当前代码行')
  }, [currentLine, currentFilePath, meeting, participants, signaling, onHighlightCode])

  const toggleFollowMode = useCallback(() => {
    if (!meeting) return

    const newState = !followMode.enabled
    const hostUser = participants.find((p) => p.userId === meeting.hostUserId)

    setFollowMode({
      enabled: newState,
      hostUserId: newState ? meeting.hostUserId : null,
      hostUserName: newState ? hostUser?.displayName || null : null,
    })

    // Broadcast follow mode change
    if (signaling) {
      signaling.send({
        type: 'media-state' as any,
        payload: {
          type: 'follow-mode',
          enabled: newState,
          hostUserId: meeting.hostUserId,
          hostUserName: hostUser?.displayName,
        },
      })
    }

    message.info(newState ? '跟随模式已开启' : '跟随模式已关闭')
  }, [followMode.enabled, meeting, participants, signaling])

  const jumpToHighlight = useCallback((highlight: CodeHighlight) => {
    onJumpToLine?.(highlight.startLine)
    message.info(`跳转到 ${highlight.userName} 高亮的代码`)
  }, [onJumpToLine])

  // Don't render if not in meeting or collaboration
  if (!isInMeeting && !isCollaborating) {
    return null
  }

  const isHost = meeting?.hostUserId === participants.find((p) => p.role === 'host')?.userId

  // Highlight popover content
  const highlightContent = (
    <div style={{ width: 280 }}>
      <div style={{ marginBottom: 12 }}>
        <Button
          type="primary"
          icon={<HighlightOutlined />}
          onClick={() => highlightCurrentCode()}
          disabled={!currentLine}
          block
        >
          高亮当前行
        </Button>
      </div>

      {highlights.length > 0 ? (
        <List
          size="small"
          dataSource={highlights}
          renderItem={(highlight) => (
            <List.Item
              style={{ cursor: 'pointer', padding: '8px 0' }}
              onClick={() => jumpToHighlight(highlight)}
            >
              <List.Item.Meta
                avatar={
                  <Avatar
                    size="small"
                    style={{ background: highlight.userColor }}
                    icon={<UserOutlined />}
                  />
                }
                title={
                  <span style={{ fontSize: 12 }}>
                    {highlight.userName} - 第 {highlight.startLine} 行
                  </span>
                }
                description={
                  <span style={{ fontSize: 11, color: '#999' }}>
                    {highlight.filePath.split('/').pop()}
                  </span>
                }
              />
            </List.Item>
          )}
        />
      ) : (
        <div style={{ textAlign: 'center', color: '#999', padding: '12px 0' }}>
          暂无代码高亮
        </div>
      )}
    </div>
  )

  return (
    <div
      style={{
        display: 'flex',
        alignItems: 'center',
        gap: 8,
        padding: '4px 8px',
        background: '#f5f5f5',
        borderRadius: 4,
      }}
    >
      {/* Meeting indicator */}
      {isInMeeting && (
        <Tooltip title={`会议中: ${meeting?.title}`}>
          <Badge status="processing" color="green">
            <Tag color="green" icon={<VideoCameraOutlined />}>
              会议中 ({participants.length})
            </Tag>
          </Badge>
        </Tooltip>
      )}

      {/* Collaboration indicator */}
      {isCollaborating && !isInMeeting && (
        <Tooltip title="协作编辑中">
          <Tag color="blue" icon={<TeamOutlined />}>
            协作中 ({collaborators.length})
          </Tag>
        </Tooltip>
      )}

      {/* Code highlight button */}
      {isInMeeting && (
        <Popover
          content={highlightContent}
          title="代码高亮"
          trigger="click"
          open={showHighlightPopover}
          onOpenChange={setShowHighlightPopover}
          placement="bottomRight"
        >
          <Tooltip title="代码高亮">
            <Badge count={highlights.length} size="small">
              <Button
                type="text"
                size="small"
                icon={<HighlightOutlined />}
              />
            </Badge>
          </Tooltip>
        </Popover>
      )}

      {/* Follow mode toggle (host only) */}
      {isInMeeting && isHost && (
        <Tooltip title={followMode.enabled ? '关闭跟随模式' : '开启跟随模式'}>
          <Space size={4}>
            <AimOutlined style={{ color: followMode.enabled ? '#52c41a' : '#999' }} />
            <Switch
              size="small"
              checked={followMode.enabled}
              onChange={toggleFollowMode}
            />
          </Space>
        </Tooltip>
      )}

      {/* Follow mode indicator (non-host) */}
      {isInMeeting && !isHost && followMode.enabled && (
        <Tooltip title={`正在跟随 ${followMode.hostUserName}`}>
          <Tag color="purple" icon={<EyeOutlined />}>
            跟随 {followMode.hostUserName}
          </Tag>
        </Tooltip>
      )}
    </div>
  )
}

// Code Highlight Decoration Component
interface CodeHighlightDecorationProps {
  highlights: CodeHighlight[]
  currentFilePath: string
  onRemoveHighlight?: (id: string) => void
}

export function CodeHighlightDecoration({
  highlights,
  currentFilePath,
  onRemoveHighlight,
}: CodeHighlightDecorationProps) {
  const fileHighlights = highlights.filter((h) => h.filePath === currentFilePath)

  if (fileHighlights.length === 0) return null

  return (
    <div className="code-highlight-decorations">
      {fileHighlights.map((highlight) => (
        <div
          key={highlight.id}
          className="code-highlight-marker"
          style={{
            position: 'absolute',
            left: 0,
            top: `${(highlight.startLine - 1) * 20}px`, // Assuming 20px line height
            height: `${(highlight.endLine - highlight.startLine + 1) * 20}px`,
            width: 4,
            background: highlight.userColor,
            borderRadius: 2,
            cursor: 'pointer',
          }}
          title={`${highlight.userName}: ${highlight.comment || '高亮代码'}`}
          onClick={() => onRemoveHighlight?.(highlight.id)}
        />
      ))}
    </div>
  )
}

// Meeting Quick Actions for Editor
interface MeetingQuickActionsProps {
  onStartMeeting: () => void
  onJoinMeeting: (meetingId: string) => void
  activeMeetingId?: string | null
}

export function MeetingQuickActions({
  onStartMeeting,
  onJoinMeeting,
  activeMeetingId,
}: MeetingQuickActionsProps) {
  const { isInMeeting } = useMeetingStore()

  if (isInMeeting) {
    return null
  }

  return (
    <Space>
      {activeMeetingId ? (
        <Button
          type="primary"
          icon={<VideoCameraOutlined />}
          onClick={() => onJoinMeeting(activeMeetingId)}
        >
          加入会议
        </Button>
      ) : (
        <Button
          icon={<VideoCameraOutlined />}
          onClick={onStartMeeting}
        >
          发起会议
        </Button>
      )}
    </Space>
  )
}
