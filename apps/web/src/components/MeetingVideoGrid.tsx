/**
 * Meeting Video Grid Component
 * Displays video streams in a responsive grid layout
 * Requirements: 4.5.3 - Video grid layout for meetings
 */

import { useRef, useEffect, useMemo } from 'react'
import { Avatar, Tooltip, Badge } from 'antd'
import {
  AudioMutedOutlined,
  AudioOutlined,
  VideoCameraOutlined,
  DesktopOutlined,
  UserOutlined,
} from '@ant-design/icons'
import { MeetingParticipant } from '@/services/meeting'

interface VideoTileProps {
  participant: MeetingParticipant
  stream?: MediaStream
  isLocal?: boolean
  isScreenShare?: boolean
  isSpeaking?: boolean
  size?: 'small' | 'medium' | 'large'
}

function VideoTile({
  participant,
  stream,
  isLocal = false,
  isScreenShare = false,
  isSpeaking = false,
  size = 'medium',
}: VideoTileProps) {
  const videoRef = useRef<HTMLVideoElement>(null)

  useEffect(() => {
    if (videoRef.current && stream) {
      videoRef.current.srcObject = stream
    }
  }, [stream])

  const sizeStyles = {
    small: { minHeight: 120, fontSize: 12 },
    medium: { minHeight: 180, fontSize: 14 },
    large: { minHeight: 280, fontSize: 16 },
  }

  const showVideo = stream && participant.videoEnabled && !isScreenShare

  return (
    <div
      className={`video-tile ${isSpeaking ? 'speaking' : ''} ${isLocal ? 'local' : ''}`}
      style={{
        position: 'relative',
        background: '#1a1a2e',
        borderRadius: 8,
        overflow: 'hidden',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        border: isSpeaking ? '2px solid #52c41a' : '2px solid transparent',
        transition: 'border-color 0.2s',
        ...sizeStyles[size],
      }}
    >
      {/* Video element */}
      {showVideo ? (
        <video
          ref={videoRef}
          autoPlay
          playsInline
          muted={isLocal}
          style={{
            width: '100%',
            height: '100%',
            objectFit: 'cover',
            transform: isLocal ? 'scaleX(-1)' : 'none',
          }}
        />
      ) : isScreenShare && stream ? (
        <video
          ref={videoRef}
          autoPlay
          playsInline
          muted
          style={{
            width: '100%',
            height: '100%',
            objectFit: 'contain',
            background: '#000',
          }}
        />
      ) : (
        <Avatar
          size={size === 'large' ? 80 : size === 'medium' ? 64 : 48}
          src={participant.avatarUrl}
          icon={<UserOutlined />}
          style={{ background: '#4a4a6a' }}
        />
      )}

      {/* Participant info overlay */}
      <div
        style={{
          position: 'absolute',
          bottom: 0,
          left: 0,
          right: 0,
          padding: '8px 12px',
          background: 'linear-gradient(transparent, rgba(0,0,0,0.7))',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
        }}
      >
        <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
          <span
            style={{
              color: '#fff',
              fontSize: sizeStyles[size].fontSize,
              fontWeight: 500,
              textShadow: '0 1px 2px rgba(0,0,0,0.5)',
            }}
          >
            {participant.displayName}
            {isLocal && ' (你)'}
          </span>
          {participant.role === 'host' && (
            <Badge
              count="主持人"
              style={{
                backgroundColor: '#1890ff',
                fontSize: 10,
              }}
            />
          )}
        </div>

        <div style={{ display: 'flex', alignItems: 'center', gap: 4 }}>
          {participant.screenSharing && (
            <Tooltip title="正在共享屏幕">
              <DesktopOutlined style={{ color: '#52c41a', fontSize: 14 }} />
            </Tooltip>
          )}
          <Tooltip title={participant.audioEnabled ? '麦克风已开启' : '麦克风已关闭'}>
            {participant.audioEnabled ? (
              <AudioOutlined style={{ color: '#fff', fontSize: 14 }} />
            ) : (
              <AudioMutedOutlined style={{ color: '#ff4d4f', fontSize: 14 }} />
            )}
          </Tooltip>
          <Tooltip title={participant.videoEnabled ? '摄像头已开启' : '摄像头已关闭'}>
            <VideoCameraOutlined
              style={{
                color: participant.videoEnabled ? '#fff' : '#ff4d4f',
                fontSize: 14,
              }}
            />
          </Tooltip>
        </div>
      </div>
    </div>
  )
}

interface MeetingVideoGridProps {
  localStream: MediaStream | null
  remoteStreams: Map<string, MediaStream>
  participants: MeetingParticipant[]
  currentUserId: string
  screenShareStream?: MediaStream | null
  screenShareUserId?: string | null
  layout?: 'grid' | 'spotlight' | 'sidebar'
}

export default function MeetingVideoGrid({
  localStream,
  remoteStreams,
  participants,
  currentUserId,
  screenShareStream,
  screenShareUserId,
  layout = 'grid',
}: MeetingVideoGridProps) {
  // Find local participant
  const localParticipant = participants.find((p) => p.userId === currentUserId)
  const remoteParticipants = participants.filter((p) => p.userId !== currentUserId)

  // Calculate grid layout
  const totalParticipants = participants.length
  const gridConfig = useMemo(() => {
    if (layout === 'spotlight' && screenShareUserId) {
      return { cols: 1, rows: 1, size: 'large' as const }
    }
    if (totalParticipants <= 1) return { cols: 1, rows: 1, size: 'large' as const }
    if (totalParticipants <= 2) return { cols: 2, rows: 1, size: 'large' as const }
    if (totalParticipants <= 4) return { cols: 2, rows: 2, size: 'medium' as const }
    if (totalParticipants <= 6) return { cols: 3, rows: 2, size: 'medium' as const }
    if (totalParticipants <= 9) return { cols: 3, rows: 3, size: 'small' as const }
    return { cols: 4, rows: Math.ceil(totalParticipants / 4), size: 'small' as const }
  }, [totalParticipants, layout, screenShareUserId])

  // Spotlight layout (screen share active)
  if (layout === 'spotlight' && screenShareStream && screenShareUserId) {
    const screenShareParticipant = participants.find((p) => p.userId === screenShareUserId)
    
    return (
      <div style={{ display: 'flex', height: '100%', gap: 16 }}>
        {/* Main screen share view */}
        <div style={{ flex: 1, display: 'flex', flexDirection: 'column' }}>
          {screenShareParticipant && (
            <VideoTile
              participant={screenShareParticipant}
              stream={screenShareStream}
              isScreenShare
              size="large"
            />
          )}
        </div>

        {/* Sidebar with participants */}
        <div
          style={{
            width: 200,
            display: 'flex',
            flexDirection: 'column',
            gap: 8,
            overflowY: 'auto',
          }}
        >
          {/* Local video */}
          {localParticipant && (
            <VideoTile
              participant={localParticipant}
              stream={localStream || undefined}
              isLocal
              size="small"
            />
          )}

          {/* Remote videos */}
          {remoteParticipants.map((participant) => (
            <VideoTile
              key={participant.userId}
              participant={participant}
              stream={remoteStreams.get(participant.userId)}
              size="small"
            />
          ))}
        </div>
      </div>
    )
  }

  // Sidebar layout
  if (layout === 'sidebar') {
    const mainParticipant = remoteParticipants[0] || localParticipant
    const sideParticipants = mainParticipant === localParticipant
      ? remoteParticipants
      : [localParticipant, ...remoteParticipants.slice(1)].filter(Boolean)

    return (
      <div style={{ display: 'flex', height: '100%', gap: 16 }}>
        {/* Main view */}
        <div style={{ flex: 1 }}>
          {mainParticipant && (
            <VideoTile
              participant={mainParticipant}
              stream={
                mainParticipant.userId === currentUserId
                  ? localStream || undefined
                  : remoteStreams.get(mainParticipant.userId)
              }
              isLocal={mainParticipant.userId === currentUserId}
              size="large"
            />
          )}
        </div>

        {/* Sidebar */}
        {sideParticipants.length > 0 && (
          <div
            style={{
              width: 200,
              display: 'flex',
              flexDirection: 'column',
              gap: 8,
              overflowY: 'auto',
            }}
          >
            {sideParticipants.map((participant) => participant && (
              <VideoTile
                key={participant.userId}
                participant={participant}
                stream={
                  participant.userId === currentUserId
                    ? localStream || undefined
                    : remoteStreams.get(participant.userId)
                }
                isLocal={participant.userId === currentUserId}
                size="small"
              />
            ))}
          </div>
        )}
      </div>
    )
  }

  // Grid layout (default)
  return (
    <div
      style={{
        display: 'grid',
        gridTemplateColumns: `repeat(${gridConfig.cols}, 1fr)`,
        gap: 12,
        height: '100%',
        padding: 12,
      }}
    >
      {/* Local video */}
      {localParticipant && (
        <VideoTile
          participant={localParticipant}
          stream={localStream || undefined}
          isLocal
          size={gridConfig.size}
        />
      )}

      {/* Remote videos */}
      {remoteParticipants.map((participant) => (
        <VideoTile
          key={participant.userId}
          participant={participant}
          stream={remoteStreams.get(participant.userId)}
          size={gridConfig.size}
        />
      ))}
    </div>
  )
}
