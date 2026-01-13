/**
 * Meeting Control Bar Component
 * Provides controls for audio, video, screen share, and meeting actions
 * Requirements: 4.5.3 - Meeting control bar with media controls
 */

import { useState } from 'react'
import {
  Button,
  Tooltip,
  Dropdown,
  Space,
  Badge,
  Modal,
  message,
  Popconfirm,
} from 'antd'
import type { MenuProps } from 'antd'
import {
  AudioOutlined,
  AudioMutedOutlined,
  VideoCameraOutlined,
  VideoCameraAddOutlined,
  DesktopOutlined,
  PhoneOutlined,
  MoreOutlined,
  TeamOutlined,
  MessageOutlined,
  SettingOutlined,
  FullscreenOutlined,
  FullscreenExitOutlined,
  PlayCircleOutlined,
  PauseCircleOutlined,
  CopyOutlined,
  ShareAltOutlined,
} from '@ant-design/icons'
import { MeetingParticipant } from '@/services/meeting'

interface MeetingControlBarProps {
  // Media state
  audioEnabled: boolean
  videoEnabled: boolean
  isScreenSharing: boolean
  isRecording: boolean

  // Meeting info
  meetingId: string
  meetingTitle: string
  participants: MeetingParticipant[]
  isHost: boolean
  duration: number

  // Callbacks
  onToggleAudio: () => void
  onToggleVideo: () => void
  onStartScreenShare: () => void
  onStopScreenShare: () => void
  onStartRecording: () => void
  onStopRecording: () => void
  onLeaveMeeting: () => void
  onEndMeeting: () => void
  onToggleParticipants: () => void
  onToggleChat: () => void
  onToggleFullscreen: () => void
  onOpenSettings: () => void

  // UI state
  isFullscreen?: boolean
  showParticipants?: boolean
  showChat?: boolean
  unreadMessages?: number
}

export default function MeetingControlBar({
  audioEnabled,
  videoEnabled,
  isScreenSharing,
  isRecording,
  meetingId,
  meetingTitle,
  participants,
  isHost,
  duration,
  onToggleAudio,
  onToggleVideo,
  onStartScreenShare,
  onStopScreenShare,
  onStartRecording,
  onStopRecording,
  onLeaveMeeting,
  onEndMeeting,
  onToggleParticipants,
  onToggleChat,
  onToggleFullscreen,
  onOpenSettings,
  isFullscreen = false,
  showParticipants = false,
  showChat = false,
  unreadMessages = 0,
}: MeetingControlBarProps) {
  const [showInviteModal, setShowInviteModal] = useState(false)

  // Format duration as HH:MM:SS
  const formatDuration = (seconds: number) => {
    const hrs = Math.floor(seconds / 3600)
    const mins = Math.floor((seconds % 3600) / 60)
    const secs = seconds % 60
    if (hrs > 0) {
      return `${hrs.toString().padStart(2, '0')}:${mins.toString().padStart(2, '0')}:${secs.toString().padStart(2, '0')}`
    }
    return `${mins.toString().padStart(2, '0')}:${secs.toString().padStart(2, '0')}`
  }

  // Copy meeting link
  const copyMeetingLink = () => {
    const link = `${window.location.origin}/meeting/${meetingId}`
    navigator.clipboard.writeText(link)
    message.success('会议链接已复制')
  }

  // More options menu
  const moreMenuItems: MenuProps['items'] = [
    {
      key: 'settings',
      icon: <SettingOutlined />,
      label: '设置',
      onClick: onOpenSettings,
    },
    {
      key: 'fullscreen',
      icon: isFullscreen ? <FullscreenExitOutlined /> : <FullscreenOutlined />,
      label: isFullscreen ? '退出全屏' : '全屏',
      onClick: onToggleFullscreen,
    },
    { type: 'divider' },
    {
      key: 'invite',
      icon: <ShareAltOutlined />,
      label: '邀请参与者',
      onClick: () => setShowInviteModal(true),
    },
    {
      key: 'copy-link',
      icon: <CopyOutlined />,
      label: '复制会议链接',
      onClick: copyMeetingLink,
    },
  ]

  // Add host-only options
  if (isHost) {
    moreMenuItems.push(
      { type: 'divider' },
      {
        key: 'recording',
        icon: isRecording ? <PauseCircleOutlined /> : <PlayCircleOutlined />,
        label: isRecording ? '停止录制' : '开始录制',
        onClick: isRecording ? onStopRecording : onStartRecording,
      }
    )
  }

  return (
    <>
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          padding: '12px 24px',
          background: '#1a1a2e',
          borderTop: '1px solid #2a2a4e',
        }}
      >
        {/* Left section - Meeting info */}
        <div style={{ display: 'flex', alignItems: 'center', gap: 16 }}>
          <div>
            <div style={{ color: '#fff', fontWeight: 500, fontSize: 14 }}>
              {meetingTitle}
            </div>
            <div style={{ color: '#8c8c8c', fontSize: 12 }}>
              {formatDuration(duration)}
              {isRecording && (
                <Badge
                  status="processing"
                  text={<span style={{ color: '#ff4d4f', marginLeft: 8 }}>录制中</span>}
                />
              )}
            </div>
          </div>
        </div>

        {/* Center section - Main controls */}
        <Space size="middle">
          {/* Audio toggle */}
          <Tooltip title={audioEnabled ? '关闭麦克风' : '开启麦克风'}>
            <Button
              type={audioEnabled ? 'default' : 'primary'}
              danger={!audioEnabled}
              shape="circle"
              size="large"
              icon={audioEnabled ? <AudioOutlined /> : <AudioMutedOutlined />}
              onClick={onToggleAudio}
              style={{
                background: audioEnabled ? '#3a3a5e' : '#ff4d4f',
                borderColor: audioEnabled ? '#3a3a5e' : '#ff4d4f',
                color: '#fff',
              }}
            />
          </Tooltip>

          {/* Video toggle */}
          <Tooltip title={videoEnabled ? '关闭摄像头' : '开启摄像头'}>
            <Button
              type={videoEnabled ? 'default' : 'primary'}
              danger={!videoEnabled}
              shape="circle"
              size="large"
              icon={videoEnabled ? <VideoCameraOutlined /> : <VideoCameraAddOutlined />}
              onClick={onToggleVideo}
              style={{
                background: videoEnabled ? '#3a3a5e' : '#ff4d4f',
                borderColor: videoEnabled ? '#3a3a5e' : '#ff4d4f',
                color: '#fff',
              }}
            />
          </Tooltip>

          {/* Screen share */}
          <Tooltip title={isScreenSharing ? '停止共享' : '共享屏幕'}>
            <Button
              type={isScreenSharing ? 'primary' : 'default'}
              shape="circle"
              size="large"
              icon={<DesktopOutlined />}
              onClick={isScreenSharing ? onStopScreenShare : onStartScreenShare}
              style={{
                background: isScreenSharing ? '#52c41a' : '#3a3a5e',
                borderColor: isScreenSharing ? '#52c41a' : '#3a3a5e',
                color: '#fff',
              }}
            />
          </Tooltip>

          {/* Participants */}
          <Tooltip title="参与者">
            <Badge count={participants.length} size="small" offset={[-5, 5]}>
              <Button
                type={showParticipants ? 'primary' : 'default'}
                shape="circle"
                size="large"
                icon={<TeamOutlined />}
                onClick={onToggleParticipants}
                style={{
                  background: showParticipants ? '#1890ff' : '#3a3a5e',
                  borderColor: showParticipants ? '#1890ff' : '#3a3a5e',
                  color: '#fff',
                }}
              />
            </Badge>
          </Tooltip>

          {/* Chat */}
          <Tooltip title="聊天">
            <Badge count={unreadMessages} size="small" offset={[-5, 5]}>
              <Button
                type={showChat ? 'primary' : 'default'}
                shape="circle"
                size="large"
                icon={<MessageOutlined />}
                onClick={onToggleChat}
                style={{
                  background: showChat ? '#1890ff' : '#3a3a5e',
                  borderColor: showChat ? '#1890ff' : '#3a3a5e',
                  color: '#fff',
                }}
              />
            </Badge>
          </Tooltip>

          {/* More options */}
          <Dropdown menu={{ items: moreMenuItems }} placement="topRight">
            <Button
              shape="circle"
              size="large"
              icon={<MoreOutlined />}
              style={{
                background: '#3a3a5e',
                borderColor: '#3a3a5e',
                color: '#fff',
              }}
            />
          </Dropdown>

          {/* Leave/End meeting */}
          {isHost ? (
            <Popconfirm
              title="结束会议"
              description="确定要结束会议吗？所有参与者将被移出会议。"
              onConfirm={onEndMeeting}
              okText="结束"
              cancelText="取消"
              okButtonProps={{ danger: true }}
            >
              <Button
                type="primary"
                danger
                size="large"
                icon={<PhoneOutlined style={{ transform: 'rotate(135deg)' }} />}
                style={{ borderRadius: 20, paddingLeft: 20, paddingRight: 20 }}
              >
                结束会议
              </Button>
            </Popconfirm>
          ) : (
            <Popconfirm
              title="离开会议"
              description="确定要离开会议吗？"
              onConfirm={onLeaveMeeting}
              okText="离开"
              cancelText="取消"
              okButtonProps={{ danger: true }}
            >
              <Button
                type="primary"
                danger
                size="large"
                icon={<PhoneOutlined style={{ transform: 'rotate(135deg)' }} />}
                style={{ borderRadius: 20, paddingLeft: 20, paddingRight: 20 }}
              >
                离开
              </Button>
            </Popconfirm>
          )}
        </Space>

        {/* Right section - Empty for balance */}
        <div style={{ width: 150 }} />
      </div>

      {/* Invite Modal */}
      <Modal
        title="邀请参与者"
        open={showInviteModal}
        onCancel={() => setShowInviteModal(false)}
        footer={null}
      >
        <div style={{ marginBottom: 16 }}>
          <div style={{ marginBottom: 8, color: '#666' }}>会议链接</div>
          <div
            style={{
              display: 'flex',
              gap: 8,
              padding: '8px 12px',
              background: '#f5f5f5',
              borderRadius: 4,
            }}
          >
            <span style={{ flex: 1, wordBreak: 'break-all' }}>
              {`${window.location.origin}/meeting/${meetingId}`}
            </span>
            <Button
              type="link"
              icon={<CopyOutlined />}
              onClick={copyMeetingLink}
            >
              复制
            </Button>
          </div>
        </div>
        <div style={{ marginBottom: 16 }}>
          <div style={{ marginBottom: 8, color: '#666' }}>会议 ID</div>
          <div
            style={{
              padding: '8px 12px',
              background: '#f5f5f5',
              borderRadius: 4,
              fontFamily: 'monospace',
            }}
          >
            {meetingId}
          </div>
        </div>
      </Modal>
    </>
  )
}
