/**
 * Screen Share Picker Component
 * Provides UI for selecting screen/window/tab to share
 * Requirements: 4.5.2 - Screen sharing with selection options
 */

import { useState, useEffect, useRef } from 'react'
import {
  Modal,
  Tabs,
  Card,
  Button,
  Checkbox,
  Alert,
  Empty,
} from 'antd'
import {
  DesktopOutlined,
  AppstoreOutlined,
  ChromeOutlined,
  SoundOutlined,
} from '@ant-design/icons'

interface ScreenShareSource {
  id: string
  name: string
  thumbnail?: string
  type: 'screen' | 'window' | 'tab'
}

interface ScreenSharePickerProps {
  visible: boolean
  onClose: () => void
  onSelect: (stream: MediaStream, sourceType: string) => void
}

export default function ScreenSharePicker({
  visible,
  onClose,
  onSelect,
}: ScreenSharePickerProps) {
  const [activeTab, setActiveTab] = useState<'screen' | 'window' | 'tab'>('screen')
  const [selectedSource, setSelectedSource] = useState<string | null>(null)
  const [includeAudio, setIncludeAudio] = useState(true)
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [previewStream, setPreviewStream] = useState<MediaStream | null>(null)
  const previewRef = useRef<HTMLVideoElement>(null)

  // Clean up preview stream on unmount
  useEffect(() => {
    return () => {
      if (previewStream) {
        previewStream.getTracks().forEach((track) => track.stop())
      }
    }
  }, [previewStream])

  // Update preview video element
  useEffect(() => {
    if (previewRef.current && previewStream) {
      previewRef.current.srcObject = previewStream
    }
  }, [previewStream])

  const handleStartShare = async () => {
    setIsLoading(true)
    setError(null)

    try {
      // Request screen capture with appropriate constraints
      const constraints: DisplayMediaStreamOptions = {
        video: {
          width: { ideal: 1920 },
          height: { ideal: 1080 },
          frameRate: { ideal: 30 },
        },
        audio: includeAudio,
      }

      // For specific display surface types
      if (activeTab === 'window') {
        (constraints.video as MediaTrackConstraints).displaySurface = 'window'
      } else if (activeTab === 'tab') {
        (constraints.video as MediaTrackConstraints).displaySurface = 'browser'
      } else {
        (constraints.video as MediaTrackConstraints).displaySurface = 'monitor'
      }

      const stream = await navigator.mediaDevices.getDisplayMedia(constraints)

      // Clean up preview
      if (previewStream) {
        previewStream.getTracks().forEach((track) => track.stop())
      }

      onSelect(stream, activeTab)
      onClose()
    } catch (err) {
      if (err instanceof Error) {
        if (err.name === 'NotAllowedError') {
          setError('屏幕共享权限被拒绝')
        } else if (err.name === 'NotFoundError') {
          setError('未找到可共享的屏幕')
        } else {
          setError(err.message)
        }
      } else {
        setError('启动屏幕共享失败')
      }
    } finally {
      setIsLoading(false)
    }
  }

  const handlePreview = async () => {
    setIsLoading(true)
    setError(null)

    try {
      // Stop existing preview
      if (previewStream) {
        previewStream.getTracks().forEach((track) => track.stop())
      }

      const constraints: DisplayMediaStreamOptions = {
        video: {
          width: { ideal: 640 },
          height: { ideal: 360 },
          frameRate: { ideal: 15 },
        },
        audio: false,
      }

      const stream = await navigator.mediaDevices.getDisplayMedia(constraints)
      setPreviewStream(stream)
      setSelectedSource('preview')

      // Handle stream end
      stream.getVideoTracks()[0].onended = () => {
        setPreviewStream(null)
        setSelectedSource(null)
      }
    } catch (err) {
      if (err instanceof Error && err.name !== 'NotAllowedError') {
        setError('预览失败: ' + err.message)
      }
    } finally {
      setIsLoading(false)
    }
  }

  const handleClose = () => {
    if (previewStream) {
      previewStream.getTracks().forEach((track) => track.stop())
      setPreviewStream(null)
    }
    setSelectedSource(null)
    setError(null)
    onClose()
  }

  const tabItems = [
    {
      key: 'screen',
      label: (
        <span>
          <DesktopOutlined />
          整个屏幕
        </span>
      ),
      children: (
        <div style={{ padding: '16px 0' }}>
          <Alert
            message="共享整个屏幕"
            description="其他参与者将看到你屏幕上的所有内容，包括通知和其他应用程序。"
            type="info"
            showIcon
            style={{ marginBottom: 16 }}
          />
          
          {previewStream ? (
            <Card
              hoverable
              style={{
                border: selectedSource === 'preview' ? '2px solid #1890ff' : undefined,
              }}
              onClick={() => setSelectedSource('preview')}
            >
              <video
                ref={previewRef}
                autoPlay
                muted
                style={{
                  width: '100%',
                  maxHeight: 300,
                  objectFit: 'contain',
                  background: '#000',
                  borderRadius: 4,
                }}
              />
              <div style={{ marginTop: 8, textAlign: 'center', color: '#666' }}>
                当前预览
              </div>
            </Card>
          ) : (
            <Card
              hoverable
              style={{ textAlign: 'center', padding: '40px 0' }}
              onClick={handlePreview}
            >
              <DesktopOutlined style={{ fontSize: 48, color: '#1890ff', marginBottom: 16 }} />
              <div>点击选择要共享的屏幕</div>
            </Card>
          )}
        </div>
      ),
    },
    {
      key: 'window',
      label: (
        <span>
          <AppstoreOutlined />
          应用窗口
        </span>
      ),
      children: (
        <div style={{ padding: '16px 0' }}>
          <Alert
            message="共享应用窗口"
            description="只共享选定的应用程序窗口，其他内容不会被看到。"
            type="info"
            showIcon
            style={{ marginBottom: 16 }}
          />
          
          <Card
            hoverable
            style={{ textAlign: 'center', padding: '40px 0' }}
            onClick={handlePreview}
          >
            <AppstoreOutlined style={{ fontSize: 48, color: '#52c41a', marginBottom: 16 }} />
            <div>点击选择要共享的窗口</div>
          </Card>
        </div>
      ),
    },
    {
      key: 'tab',
      label: (
        <span>
          <ChromeOutlined />
          浏览器标签页
        </span>
      ),
      children: (
        <div style={{ padding: '16px 0' }}>
          <Alert
            message="共享浏览器标签页"
            description="只共享选定的浏览器标签页，适合展示网页内容。"
            type="info"
            showIcon
            style={{ marginBottom: 16 }}
          />
          
          <Card
            hoverable
            style={{ textAlign: 'center', padding: '40px 0' }}
            onClick={handlePreview}
          >
            <ChromeOutlined style={{ fontSize: 48, color: '#faad14', marginBottom: 16 }} />
            <div>点击选择要共享的标签页</div>
          </Card>
        </div>
      ),
    },
  ]

  return (
    <Modal
      title="共享屏幕"
      open={visible}
      onCancel={handleClose}
      width={600}
      footer={[
        <Checkbox
          key="audio"
          checked={includeAudio}
          onChange={(e) => setIncludeAudio(e.target.checked)}
          style={{ float: 'left', marginTop: 5 }}
        >
          <SoundOutlined /> 共享系统音频
        </Checkbox>,
        <Button key="cancel" onClick={handleClose}>
          取消
        </Button>,
        <Button
          key="share"
          type="primary"
          onClick={handleStartShare}
          loading={isLoading}
          icon={<DesktopOutlined />}
        >
          开始共享
        </Button>,
      ]}
    >
      {error && (
        <Alert
          message={error}
          type="error"
          showIcon
          closable
          onClose={() => setError(null)}
          style={{ marginBottom: 16 }}
        />
      )}

      <Tabs
        activeKey={activeTab}
        onChange={(key) => setActiveTab(key as 'screen' | 'window' | 'tab')}
        items={tabItems}
      />
    </Modal>
  )
}

// Screen Share Preview Component
interface ScreenSharePreviewProps {
  stream: MediaStream | null
  userName: string
  onStopShare?: () => void
  isLocal?: boolean
}

export function ScreenSharePreview({
  stream,
  userName,
  onStopShare,
  isLocal = false,
}: ScreenSharePreviewProps) {
  const videoRef = useRef<HTMLVideoElement>(null)

  useEffect(() => {
    if (videoRef.current && stream) {
      videoRef.current.srcObject = stream
    }
  }, [stream])

  if (!stream) {
    return (
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          height: '100%',
          background: '#1a1a2e',
          borderRadius: 8,
        }}
      >
        <Empty description="没有屏幕共享" />
      </div>
    )
  }

  return (
    <div
      style={{
        position: 'relative',
        height: '100%',
        background: '#000',
        borderRadius: 8,
        overflow: 'hidden',
      }}
    >
      <video
        ref={videoRef}
        autoPlay
        playsInline
        muted
        style={{
          width: '100%',
          height: '100%',
          objectFit: 'contain',
        }}
      />

      {/* Overlay info */}
      <div
        style={{
          position: 'absolute',
          top: 12,
          left: 12,
          padding: '4px 12px',
          background: 'rgba(0, 0, 0, 0.6)',
          borderRadius: 4,
          color: '#fff',
          fontSize: 12,
        }}
      >
        <DesktopOutlined style={{ marginRight: 8 }} />
        {userName} 正在共享屏幕
      </div>

      {/* Stop button for local share */}
      {isLocal && onStopShare && (
        <div
          style={{
            position: 'absolute',
            bottom: 12,
            left: '50%',
            transform: 'translateX(-50%)',
          }}
        >
          <Button
            type="primary"
            danger
            onClick={onStopShare}
            icon={<DesktopOutlined />}
          >
            停止共享
          </Button>
        </div>
      )}
    </div>
  )
}

// Floating Screen Share Indicator
interface ScreenShareIndicatorProps {
  isSharing: boolean
  onStopShare: () => void
}

export function ScreenShareIndicator({
  isSharing,
  onStopShare,
}: ScreenShareIndicatorProps) {
  if (!isSharing) return null

  return (
    <div
      style={{
        position: 'fixed',
        bottom: 100,
        left: '50%',
        transform: 'translateX(-50%)',
        padding: '8px 16px',
        background: '#52c41a',
        borderRadius: 20,
        color: '#fff',
        display: 'flex',
        alignItems: 'center',
        gap: 12,
        boxShadow: '0 4px 12px rgba(0, 0, 0, 0.3)',
        zIndex: 1000,
      }}
    >
      <DesktopOutlined />
      <span>你正在共享屏幕</span>
      <Button
        type="text"
        size="small"
        onClick={onStopShare}
        style={{ color: '#fff', padding: '0 8px' }}
      >
        停止共享
      </Button>
    </div>
  )
}
