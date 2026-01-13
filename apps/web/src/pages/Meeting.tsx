/**
 * Meeting Page
 * Main page for video/audio meetings with screen sharing
 * Requirements: 4.5.1, 4.5.2, 4.5.3 - Real-time meeting with video/audio support
 */

import { useState, useEffect, useCallback } from 'react'
import { useParams, useNavigate, useSearchParams } from 'react-router-dom'
import {
  Layout,
  Spin,
  Alert,
  Button,
  Modal,
  Form,
  Input,
  message,
  Drawer,
  List,
  Avatar,
  Tag,
  Dropdown,
  Empty,
} from 'antd'
import type { MenuProps } from 'antd'
import {
  UserOutlined,
  CrownOutlined,
  AudioMutedOutlined,
  MoreOutlined,
  CloseOutlined,
} from '@ant-design/icons'
import MeetingVideoGrid from '@/components/MeetingVideoGrid'
import MeetingControlBar from '@/components/MeetingControlBar'
import { useMeetingStore } from '@/stores/meeting'
import { useAuthStore } from '@/stores/auth'
import { meetingService, Meeting } from '@/services/meeting'

const { Content } = Layout

export default function MeetingPage() {
  const { id: meetingId } = useParams<{ id: string }>()
  const [searchParams] = useSearchParams()
  const navigate = useNavigate()
  const { user } = useAuthStore()

  // Meeting store
  const {
    meeting,
    isInMeeting,
    isJoining,
    error,
    localStream,
    remoteStreams,
    participants,
    audioEnabled,
    videoEnabled,
    isScreenSharing,
    isRecording,
    screenStream,
    joinMeeting,
    leaveMeeting,
    endMeeting,
    toggleAudio,
    toggleVideo,
    startScreenShare,
    stopScreenShare,
    startRecording,
    stopRecording,
    setError,
  } = useMeetingStore()

  // Local state
  const [isLoading, setIsLoading] = useState(true)
  const [meetingInfo, setMeetingInfo] = useState<Meeting | null>(null)
  const [showJoinModal, setShowJoinModal] = useState(false)
  const [showParticipants, setShowParticipants] = useState(false)
  const [showChat, setShowChat] = useState(false)
  const [isFullscreen, setIsFullscreen] = useState(false)
  const [duration, setDuration] = useState(0)
  const [layout, setLayout] = useState<'grid' | 'spotlight' | 'sidebar'>('grid')

  const [joinForm] = Form.useForm()

  // Load meeting info
  useEffect(() => {
    if (meetingId) {
      loadMeetingInfo()
    }
  }, [meetingId])

  // Duration timer
  useEffect(() => {
    let interval: ReturnType<typeof setInterval>
    if (isInMeeting && meeting?.startedAt) {
      interval = setInterval(() => {
        const start = new Date(meeting.startedAt!).getTime()
        const now = Date.now()
        setDuration(Math.floor((now - start) / 1000))
      }, 1000)
    }
    return () => clearInterval(interval)
  }, [isInMeeting, meeting?.startedAt])

  // Auto-switch to spotlight layout when screen sharing
  useEffect(() => {
    const screenSharer = participants.find((p) => p.screenSharing)
    if (screenSharer) {
      setLayout('spotlight')
    } else if (layout === 'spotlight') {
      setLayout('grid')
    }
  }, [participants])

  // Handle fullscreen
  useEffect(() => {
    const handleFullscreenChange = () => {
      setIsFullscreen(!!document.fullscreenElement)
    }
    document.addEventListener('fullscreenchange', handleFullscreenChange)
    return () => document.removeEventListener('fullscreenchange', handleFullscreenChange)
  }, [])

  const loadMeetingInfo = async () => {
    setIsLoading(true)
    try {
      if (meetingId) {
        const info = await meetingService.getMeeting(meetingId)
        setMeetingInfo(info)

        if (info.status === 'ended') {
          message.error('会议已结束')
          navigate('/')
          return
        }

        // Check if auto-join from URL
        const autoJoin = searchParams.get('join') === 'true'
        if (autoJoin && user) {
          handleJoinMeeting(user.displayName || user.username)
        } else {
          setShowJoinModal(true)
        }
      }
    } catch {
      // Mock data for development
      setMeetingInfo({
        id: meetingId || '1',
        environmentId: 'env-1',
        hostUserId: 'user-1',
        title: '代码评审会议',
        type: 'video',
        status: 'active',
        settings: {
          maxParticipants: 10,
          allowRecording: true,
          allowScreenShare: true,
          enableTranscription: false,
          enableVirtualBackground: true,
          waitingRoom: false,
          muteOnJoin: false,
          videoOnJoin: true,
        },
        participants: [],
        createdAt: new Date().toISOString(),
        startedAt: new Date().toISOString(),
      })
      setShowJoinModal(true)
    } finally {
      setIsLoading(false)
    }
  }

  const handleJoinMeeting = async (displayName: string) => {
    if (!meetingId) return

    try {
      await joinMeeting(meetingId, displayName)
      setShowJoinModal(false)
      message.success('已加入会议')
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : '加入会议失败'
      message.error(errorMessage)
    }
  }

  const handleLeaveMeeting = async () => {
    await leaveMeeting()
    message.info('已离开会议')
    navigate(-1)
  }

  const handleEndMeeting = async () => {
    await endMeeting()
    message.info('会议已结束')
    navigate(-1)
  }

  const toggleFullscreen = useCallback(() => {
    if (!document.fullscreenElement) {
      document.documentElement.requestFullscreen()
    } else {
      document.exitFullscreen()
    }
  }, [])

  const isHost = meeting?.hostUserId === user?.id

  // Find screen sharer
  const screenSharer = participants.find((p) => p.screenSharing)
  const screenShareUserId = screenSharer?.userId || null
  const activeScreenStream = screenSharer?.userId === user?.id ? screenStream : remoteStreams.get(screenSharer?.userId || '')

  // Participant menu items
  const getParticipantMenuItems = (participant: typeof participants[0]): MenuProps['items'] => {
    if (!isHost || participant.userId === user?.id) return []
    
    return [
      {
        key: 'mute',
        label: '静音',
        onClick: () => meetingService.muteParticipant(meeting!.id, participant.userId),
      },
      {
        key: 'promote',
        label: '设为联合主持人',
        onClick: () => meetingService.promoteToCoHost(meeting!.id, participant.userId),
      },
      { type: 'divider' },
      {
        key: 'kick',
        label: '移出会议',
        danger: true,
        onClick: () => meetingService.kickParticipant(meeting!.id, participant.userId),
      },
    ]
  }

  if (isLoading) {
    return (
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          height: '100vh',
          background: '#0d0d1a',
        }}
      >
        <Spin size="large" tip="加载会议信息..." />
      </div>
    )
  }

  if (error && !isInMeeting) {
    return (
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          height: '100vh',
          background: '#0d0d1a',
        }}
      >
        <Alert
          type="error"
          message="无法加入会议"
          description={error}
          action={
            <Button onClick={() => navigate(-1)}>返回</Button>
          }
        />
      </div>
    )
  }

  return (
    <Layout style={{ height: '100vh', background: '#0d0d1a' }}>
      {/* Main content */}
      <Content
        style={{
          display: 'flex',
          flexDirection: 'column',
          height: '100%',
        }}
      >
        {/* Video grid area */}
        <div style={{ flex: 1, overflow: 'hidden' }}>
          {isInMeeting ? (
            <MeetingVideoGrid
              localStream={localStream}
              remoteStreams={remoteStreams}
              participants={participants}
              currentUserId={user?.id || ''}
              screenShareStream={activeScreenStream}
              screenShareUserId={screenShareUserId}
              layout={layout}
            />
          ) : (
            <div
              style={{
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                height: '100%',
              }}
            >
              {isJoining ? (
                <Spin size="large" tip="正在加入会议..." />
              ) : (
                <Empty
                  description="等待加入会议"
                  image={Empty.PRESENTED_IMAGE_SIMPLE}
                />
              )}
            </div>
          )}
        </div>

        {/* Control bar */}
        {isInMeeting && (
          <MeetingControlBar
            audioEnabled={audioEnabled}
            videoEnabled={videoEnabled}
            isScreenSharing={isScreenSharing}
            isRecording={isRecording}
            meetingId={meeting?.id || ''}
            meetingTitle={meeting?.title || '会议'}
            participants={participants}
            isHost={isHost}
            duration={duration}
            onToggleAudio={toggleAudio}
            onToggleVideo={toggleVideo}
            onStartScreenShare={startScreenShare}
            onStopScreenShare={stopScreenShare}
            onStartRecording={startRecording}
            onStopRecording={stopRecording}
            onLeaveMeeting={handleLeaveMeeting}
            onEndMeeting={handleEndMeeting}
            onToggleParticipants={() => setShowParticipants(!showParticipants)}
            onToggleChat={() => setShowChat(!showChat)}
            onToggleFullscreen={toggleFullscreen}
            onOpenSettings={() => {}}
            isFullscreen={isFullscreen}
            showParticipants={showParticipants}
            showChat={showChat}
          />
        )}
      </Content>

      {/* Participants drawer */}
      <Drawer
        title={`参与者 (${participants.length})`}
        placement="right"
        open={showParticipants}
        onClose={() => setShowParticipants(false)}
        width={320}
        styles={{ body: { padding: 0 } }}
      >
        <List
          dataSource={participants}
          renderItem={(participant) => (
            <List.Item
              style={{ padding: '12px 16px' }}
              actions={
                isHost && participant.userId !== user?.id
                  ? [
                      <Dropdown
                        key="more"
                        menu={{ items: getParticipantMenuItems(participant) }}
                      >
                        <Button type="text" icon={<MoreOutlined />} />
                      </Dropdown>,
                    ]
                  : undefined
              }
            >
              <List.Item.Meta
                avatar={
                  <Avatar src={participant.avatarUrl} icon={<UserOutlined />} />
                }
                title={
                  <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                    {participant.displayName}
                    {participant.userId === user?.id && (
                      <Tag color="blue" style={{ marginLeft: 4 }}>
                        你
                      </Tag>
                    )}
                    {participant.role === 'host' && (
                      <CrownOutlined style={{ color: '#faad14' }} />
                    )}
                  </div>
                }
                description={
                  <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                    {!participant.audioEnabled && (
                      <AudioMutedOutlined style={{ color: '#ff4d4f' }} />
                    )}
                    {participant.screenSharing && (
                      <Tag color="green">共享中</Tag>
                    )}
                  </div>
                }
              />
            </List.Item>
          )}
        />
      </Drawer>

      {/* Chat drawer */}
      <Drawer
        title="聊天"
        placement="right"
        open={showChat}
        onClose={() => setShowChat(false)}
        width={360}
      >
        <Empty description="聊天功能即将推出" />
      </Drawer>

      {/* Join meeting modal */}
      <Modal
        title="加入会议"
        open={showJoinModal}
        onCancel={() => navigate(-1)}
        footer={null}
        closable
        maskClosable={false}
      >
        <div style={{ marginBottom: 16 }}>
          <div style={{ fontSize: 16, fontWeight: 500, marginBottom: 4 }}>
            {meetingInfo?.title}
          </div>
          <div style={{ color: '#666' }}>
            会议 ID: {meetingId}
          </div>
        </div>

        <Form
          form={joinForm}
          layout="vertical"
          onFinish={(values) => handleJoinMeeting(values.displayName)}
          initialValues={{
            displayName: user?.displayName || user?.username || '',
          }}
        >
          <Form.Item
            name="displayName"
            label="显示名称"
            rules={[{ required: true, message: '请输入显示名称' }]}
          >
            <Input placeholder="输入你在会议中显示的名称" />
          </Form.Item>

          <Form.Item style={{ marginBottom: 0 }}>
            <Button
              type="primary"
              htmlType="submit"
              loading={isJoining}
              block
              size="large"
            >
              加入会议
            </Button>
          </Form.Item>
        </Form>
      </Modal>
    </Layout>
  )
}
