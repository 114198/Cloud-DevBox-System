/**
 * CollaboratorsPanel Component
 * Displays online collaborators with their status and cursor colors
 * Requirements: 4.2 - Display other users' cursor positions and editing status
 */

import { useState } from 'react'
import {
  Card,
  List,
  Avatar,
  Badge,
  Tag,
  Tooltip,
  Button,
  Dropdown,
  Space,
  Empty,
  Typography,
  Popconfirm,
  message,
} from 'antd'
import {
  UserOutlined,
  CrownOutlined,
  EyeOutlined,
  EditOutlined,
  MoreOutlined,
  UserDeleteOutlined,
  SwapOutlined,
  TeamOutlined,
} from '@ant-design/icons'
import { useCollaborationStore, useOnlineParticipants } from '@/stores/collaboration'
import { collaborationService, SessionParticipant } from '@/services/collaboration'
import { useAuthStore } from '@/stores/auth'

const { Text } = Typography

interface CollaboratorsPanelProps {
  sessionId?: string
  showOffline?: boolean
  compact?: boolean
  onInviteClick?: () => void
}

// Role icons and labels
const roleConfig: Record<string, { icon: React.ReactNode; label: string; color: string }> = {
  host: { icon: <CrownOutlined />, label: '主持人', color: 'gold' },
  participant: { icon: <EditOutlined />, label: '编辑者', color: 'blue' },
  viewer: { icon: <EyeOutlined />, label: '查看者', color: 'default' },
}

export default function CollaboratorsPanel({
  sessionId,
  showOffline = false,
  compact = false,
  onInviteClick,
}: CollaboratorsPanelProps) {
  const { participants, currentUserId, isConnected } = useCollaborationStore()
  const onlineParticipants = useOnlineParticipants()
  const { user } = useAuthStore()
  const [loading, setLoading] = useState<string | null>(null)

  // Filter participants based on showOffline prop
  const displayParticipants = showOffline ? participants : onlineParticipants

  // Check if current user is host
  const isHost = participants.find(p => p.userId === currentUserId)?.role === 'host'

  // Handle role change
  const handleRoleChange = async (userId: string, newRole: 'participant' | 'viewer') => {
    if (!sessionId) return

    setLoading(userId)
    try {
      await collaborationService.updateParticipantRole(sessionId, userId, newRole)
      message.success('角色已更新')
    } catch (err) {
      message.error('更新角色失败')
    } finally {
      setLoading(null)
    }
  }

  // Handle remove participant
  const handleRemoveParticipant = async (userId: string) => {
    if (!sessionId) return

    setLoading(userId)
    try {
      await collaborationService.removeParticipant(sessionId, userId)
      message.success('已移除协作者')
    } catch (err) {
      message.error('移除失败')
    } finally {
      setLoading(null)
    }
  }

  // Get dropdown menu items for a participant
  const getMenuItems = (participant: SessionParticipant) => {
    if (!isHost || participant.userId === currentUserId || participant.role === 'host') {
      return []
    }

    return [
      {
        key: 'role',
        label: '更改角色',
        icon: <SwapOutlined />,
        children: [
          {
            key: 'participant',
            label: '编辑者',
            disabled: participant.role === 'participant',
            onClick: () => handleRoleChange(participant.userId, 'participant'),
          },
          {
            key: 'viewer',
            label: '查看者',
            disabled: participant.role === 'viewer',
            onClick: () => handleRoleChange(participant.userId, 'viewer'),
          },
        ],
      },
      {
        type: 'divider' as const,
      },
      {
        key: 'remove',
        label: '移除',
        icon: <UserDeleteOutlined />,
        danger: true,
        onClick: () => {}, // Handled by Popconfirm
      },
    ]
  }

  // Render participant item
  const renderParticipant = (participant: SessionParticipant) => {
    const isCurrentUser = participant.userId === currentUserId
    const role = roleConfig[participant.role] || roleConfig.participant
    const menuItems = getMenuItems(participant)

    return (
      <List.Item
        key={participant.userId}
        className={`collaborator-item ${!participant.isOnline ? 'offline' : ''}`}
        style={{
          padding: compact ? '8px 12px' : '12px 16px',
          opacity: participant.isOnline ? 1 : 0.5,
        }}
        actions={
          !compact && menuItems.length > 0
            ? [
                <Dropdown
                  key="actions"
                  menu={{ items: menuItems }}
                  trigger={['click']}
                  disabled={loading === participant.userId}
                >
                  <Button
                    type="text"
                    size="small"
                    icon={<MoreOutlined />}
                    loading={loading === participant.userId}
                  />
                </Dropdown>,
              ]
            : undefined
        }
      >
        <List.Item.Meta
          avatar={
            <Badge
              dot
              status={participant.isOnline ? 'success' : 'default'}
              offset={[-4, 28]}
            >
              <Avatar
                src={participant.avatarUrl}
                icon={<UserOutlined />}
                style={{
                  backgroundColor: participant.color,
                  border: isCurrentUser ? '2px solid #1890ff' : 'none',
                }}
                size={compact ? 'small' : 'default'}
              >
                {participant.displayName?.[0] || participant.username?.[0]}
              </Avatar>
            </Badge>
          }
          title={
            <Space size={4}>
              <Text
                strong={isCurrentUser}
                style={{ fontSize: compact ? 12 : 14 }}
              >
                {participant.displayName || participant.username}
                {isCurrentUser && <Text type="secondary"> (你)</Text>}
              </Text>
              {!compact && (
                <Tooltip title={role.label}>
                  <Tag
                    color={role.color}
                    icon={role.icon}
                    style={{ marginLeft: 4, fontSize: 10 }}
                  >
                    {compact ? '' : role.label}
                  </Tag>
                </Tooltip>
              )}
            </Space>
          }
          description={
            !compact && (
              <Text type="secondary" style={{ fontSize: 12 }}>
                {participant.isOnline
                  ? participant.cursorPosition
                    ? `正在编辑 ${participant.cursorPosition.filePath.split('/').pop()}`
                    : '在线'
                  : `离开于 ${formatTime(participant.leftAt)}`}
              </Text>
            )
          }
        />
        {/* Color indicator */}
        <div
          style={{
            width: 4,
            height: compact ? 24 : 32,
            backgroundColor: participant.color,
            borderRadius: 2,
            marginLeft: 8,
          }}
        />
      </List.Item>
    )
  }

  // Format time helper
  const formatTime = (dateStr?: string) => {
    if (!dateStr) return ''
    const date = new Date(dateStr)
    const now = new Date()
    const diff = now.getTime() - date.getTime()

    if (diff < 60000) return '刚刚'
    if (diff < 3600000) return `${Math.floor(diff / 60000)} 分钟前`
    if (diff < 86400000) return `${Math.floor(diff / 3600000)} 小时前`
    return date.toLocaleDateString('zh-CN')
  }

  if (!isConnected && displayParticipants.length === 0) {
    return (
      <Card
        title={
          <Space>
            <TeamOutlined />
            协作者
          </Space>
        }
        size="small"
        extra={
          onInviteClick && (
            <Button type="link" size="small" onClick={onInviteClick}>
              邀请
            </Button>
          )
        }
      >
        <Empty
          image={Empty.PRESENTED_IMAGE_SIMPLE}
          description="未连接到协作会话"
        />
      </Card>
    )
  }

  return (
    <Card
      title={
        <Space>
          <TeamOutlined />
          协作者
          <Badge
            count={onlineParticipants.length}
            style={{ backgroundColor: '#52c41a' }}
            size="small"
          />
        </Space>
      }
      size="small"
      extra={
        <Space>
          {showOffline && participants.length > onlineParticipants.length && (
            <Text type="secondary" style={{ fontSize: 12 }}>
              {participants.length - onlineParticipants.length} 离线
            </Text>
          )}
          {onInviteClick && (
            <Button type="link" size="small" onClick={onInviteClick}>
              邀请
            </Button>
          )}
        </Space>
      }
      styles={{ body: { padding: 0 } }}
    >
      <List
        dataSource={displayParticipants}
        renderItem={renderParticipant}
        locale={{ emptyText: '暂无协作者' }}
        size="small"
      />
    </Card>
  )
}

// Compact version for inline display
export function CollaboratorsAvatarGroup({
  maxCount = 5,
  onClick,
}: {
  maxCount?: number
  onClick?: () => void
}) {
  const onlineParticipants = useOnlineParticipants()
  const { currentUserId } = useCollaborationStore()

  if (onlineParticipants.length === 0) {
    return null
  }

  return (
    <div
      onClick={onClick}
      style={{ cursor: onClick ? 'pointer' : 'default' }}
    >
      <Avatar.Group
        maxCount={maxCount}
        maxStyle={{
          color: '#f56a00',
          backgroundColor: '#fde3cf',
        }}
      >
        {onlineParticipants.map((participant) => (
          <Tooltip
            key={participant.userId}
            title={`${participant.displayName || participant.username}${
              participant.userId === currentUserId ? ' (你)' : ''
            }`}
          >
            <Badge
              dot
              status="success"
              offset={[-4, 28]}
            >
              <Avatar
                src={participant.avatarUrl}
                style={{
                  backgroundColor: participant.color,
                  border:
                    participant.userId === currentUserId
                      ? '2px solid #1890ff'
                      : '2px solid #fff',
                }}
              >
                {participant.displayName?.[0] || participant.username?.[0]}
              </Avatar>
            </Badge>
          </Tooltip>
        ))}
      </Avatar.Group>
    </div>
  )
}
