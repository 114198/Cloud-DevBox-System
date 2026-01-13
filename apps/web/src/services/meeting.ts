/**
 * Meeting Service
 * Handles API calls and WebRTC signaling for video/audio meetings
 * Requirements: 4.5.1, 4.5.2, 4.5.3 - Real-time meeting with video/audio support
 */

import api from './api'
import { useAuthStore } from '@/stores/auth'

// Types
export type MeetingType = 'audio' | 'video' | 'screen_share'
export type MeetingStatus = 'scheduled' | 'active' | 'ended' | 'cancelled'
export type ParticipantRole = 'host' | 'co-host' | 'participant'
export type VideoQuality = '1080p' | '720p' | '480p' | '360p' | 'audio_only'

export interface MeetingSettings {
  maxParticipants: number
  allowRecording: boolean
  allowScreenShare: boolean
  enableTranscription: boolean
  enableVirtualBackground: boolean
  waitingRoom: boolean
  muteOnJoin: boolean
  videoOnJoin: boolean
}

export interface MeetingParticipant {
  id: string
  meetingId: string
  userId: string
  displayName: string
  avatarUrl?: string
  role: ParticipantRole
  joinedAt: string
  leftAt?: string
  audioEnabled: boolean
  videoEnabled: boolean
  screenSharing: boolean
  videoQuality: VideoQuality
  connectionId?: string
}

export interface MeetingRecording {
  id: string
  meetingId: string
  duration: number
  fileSize: number
  format: string
  storageUrl: string
  transcription?: string
  createdAt: string
}

export interface Meeting {
  id: string
  environmentId: string
  hostUserId: string
  title: string
  type: MeetingType
  status: MeetingStatus
  settings: MeetingSettings
  participants: MeetingParticipant[]
  recording?: MeetingRecording
  createdAt: string
  startedAt?: string
  endedAt?: string
}

export interface ICEServer {
  urls: string[]
  username?: string
  credential?: string
}

export interface RTCConfiguration {
  iceServers: ICEServer[]
  iceTransportPolicy?: string
  bundlePolicy?: string
}

export interface JoinMeetingResponse {
  meeting: Meeting
  participant: MeetingParticipant
  rtcConfig: RTCConfiguration
  participants: MeetingParticipant[]
}

export interface CreateMeetingRequest {
  environmentId: string
  title: string
  type?: MeetingType
  settings?: Partial<MeetingSettings>
  invitees?: string[]
}

// Signaling message types
export type SignalingMessageType =
  | 'offer'
  | 'answer'
  | 'ice-candidate'
  | 'join'
  | 'leave'
  | 'mute'
  | 'unmute'
  | 'screen-share'
  | 'stop-share'
  | 'quality-change'
  | 'record-start'
  | 'record-stop'
  | 'error'
  | 'user-joined'
  | 'user-left'
  | 'media-state'

export interface SignalingMessage {
  type: SignalingMessageType
  meetingId: string
  userId: string
  targetId?: string
  payload: unknown
  timestamp: number
}

export interface MediaState {
  audioEnabled: boolean
  videoEnabled: boolean
  screenSharing: boolean
  videoQuality?: VideoQuality
}

// Default settings
export const DEFAULT_MEETING_SETTINGS: MeetingSettings = {
  maxParticipants: 10,
  allowRecording: true,
  allowScreenShare: true,
  enableTranscription: false,
  enableVirtualBackground: true,
  waitingRoom: false,
  muteOnJoin: false,
  videoOnJoin: true,
}

// WebRTC Signaling WebSocket Manager
export class MeetingSignaling {
  private ws: WebSocket | null = null
  private meetingId: string
  private reconnectAttempts = 0
  private maxReconnectAttempts = 5
  private reconnectDelay = 1000
  private pingInterval: ReturnType<typeof setInterval> | null = null

  // Event handlers
  public onConnect?: () => void
  public onDisconnect?: () => void
  public onError?: (error: Error) => void
  public onOffer?: (userId: string, sdp: RTCSessionDescriptionInit) => void
  public onAnswer?: (userId: string, sdp: RTCSessionDescriptionInit) => void
  public onIceCandidate?: (userId: string, candidate: RTCIceCandidateInit) => void
  public onUserJoined?: (participant: MeetingParticipant) => void
  public onUserLeft?: (userId: string) => void
  public onMediaStateChange?: (userId: string, state: MediaState) => void
  public onScreenShareStart?: (userId: string) => void
  public onScreenShareStop?: (userId: string) => void
  public onRecordingStart?: () => void
  public onRecordingStop?: () => void

  constructor(meetingId: string) {
    this.meetingId = meetingId
  }

  connect(): Promise<void> {
    return new Promise((resolve, reject) => {
      const { user, accessToken } = useAuthStore.getState()
      if (!user || !accessToken) {
        reject(new Error('User not authenticated'))
        return
      }

      const wsUrl = this.buildWebSocketUrl(accessToken)
      this.ws = new WebSocket(wsUrl)

      this.ws.onopen = () => {
        this.reconnectAttempts = 0
        this.startPingInterval()
        this.onConnect?.()
        resolve()
      }

      this.ws.onmessage = (event) => {
        try {
          const message = JSON.parse(event.data) as SignalingMessage
          this.handleMessage(message)
        } catch (err) {
          console.error('Failed to parse signaling message:', err)
        }
      }

      this.ws.onerror = () => {
        this.onError?.(new Error('WebSocket connection error'))
      }

      this.ws.onclose = () => {
        this.stopPingInterval()
        this.onDisconnect?.()
        this.attemptReconnect()
      }

      setTimeout(() => {
        if (this.ws?.readyState !== WebSocket.OPEN) {
          reject(new Error('Connection timeout'))
        }
      }, 10000)
    })
  }

  private buildWebSocketUrl(token: string): string {
    const baseUrl = import.meta.env.VITE_WS_URL || `ws://${window.location.host}`
    return `${baseUrl}/api/v1/meetings/${this.meetingId}/signal?token=${token}`
  }

  private handleMessage(message: SignalingMessage) {
    switch (message.type) {
      case 'offer':
        this.onOffer?.(message.userId, message.payload as RTCSessionDescriptionInit)
        break
      case 'answer':
        this.onAnswer?.(message.userId, message.payload as RTCSessionDescriptionInit)
        break
      case 'ice-candidate':
        this.onIceCandidate?.(message.userId, message.payload as RTCIceCandidateInit)
        break
      case 'user-joined':
        this.onUserJoined?.(message.payload as MeetingParticipant)
        break
      case 'user-left':
        this.onUserLeft?.(message.userId)
        break
      case 'media-state':
        this.onMediaStateChange?.(message.userId, message.payload as MediaState)
        break
      case 'screen-share':
        this.onScreenShareStart?.(message.userId)
        break
      case 'stop-share':
        this.onScreenShareStop?.(message.userId)
        break
      case 'record-start':
        this.onRecordingStart?.()
        break
      case 'record-stop':
        this.onRecordingStop?.()
        break
      case 'error':
        const errorPayload = message.payload as { message: string }
        this.onError?.(new Error(errorPayload?.message || 'Unknown error'))
        break
    }
  }

  disconnect() {
    this.stopPingInterval()
    if (this.ws) {
      this.ws.close()
      this.ws = null
    }
  }

  private startPingInterval() {
    this.pingInterval = setInterval(() => {
      this.send({ type: 'ping' as SignalingMessageType, payload: {} })
    }, 30000)
  }

  private stopPingInterval() {
    if (this.pingInterval) {
      clearInterval(this.pingInterval)
      this.pingInterval = null
    }
  }

  private attemptReconnect() {
    if (this.reconnectAttempts >= this.maxReconnectAttempts) {
      this.onError?.(new Error('Max reconnection attempts reached'))
      return
    }

    this.reconnectAttempts++
    const delay = this.reconnectDelay * Math.pow(2, this.reconnectAttempts - 1)

    setTimeout(() => {
      this.connect().catch(console.error)
    }, delay)
  }

  send(message: Partial<SignalingMessage>) {
    if (this.ws?.readyState === WebSocket.OPEN) {
      const fullMessage: SignalingMessage = {
        type: message.type!,
        meetingId: this.meetingId,
        userId: useAuthStore.getState().user?.id || '',
        targetId: message.targetId,
        payload: message.payload,
        timestamp: Date.now(),
      }
      this.ws.send(JSON.stringify(fullMessage))
    }
  }

  sendOffer(targetId: string, sdp: RTCSessionDescriptionInit) {
    this.send({ type: 'offer', targetId, payload: sdp })
  }

  sendAnswer(targetId: string, sdp: RTCSessionDescriptionInit) {
    this.send({ type: 'answer', targetId, payload: sdp })
  }

  sendIceCandidate(targetId: string, candidate: RTCIceCandidateInit) {
    this.send({ type: 'ice-candidate', targetId, payload: candidate })
  }

  sendMediaState(state: MediaState) {
    this.send({ type: 'media-state', payload: state })
  }

  sendScreenShare(action: 'start' | 'stop') {
    this.send({ type: action === 'start' ? 'screen-share' : 'stop-share', payload: { action } })
  }

  get isConnected(): boolean {
    return this.ws?.readyState === WebSocket.OPEN
  }
}


// REST API functions
export const meetingService = {
  // Create a new meeting
  async createMeeting(request: CreateMeetingRequest): Promise<Meeting> {
    const { user } = useAuthStore.getState()
    const response = await api.post('/meetings', {
      ...request,
      hostUserId: user?.id,
      type: request.type || 'video',
      settings: { ...DEFAULT_MEETING_SETTINGS, ...request.settings },
    })
    return response.data
  },

  // Get meeting by ID
  async getMeeting(meetingId: string): Promise<Meeting> {
    const response = await api.get(`/meetings/${meetingId}`)
    return response.data
  },

  // Get active meeting for environment
  async getActiveMeetingForEnvironment(environmentId: string): Promise<Meeting | null> {
    try {
      const response = await api.get(`/meetings/environment/${environmentId}/active`)
      return response.data
    } catch {
      return null
    }
  },

  // Join a meeting
  async joinMeeting(meetingId: string, displayName: string): Promise<JoinMeetingResponse> {
    const { user } = useAuthStore.getState()
    const response = await api.post(`/meetings/${meetingId}/join`, {
      meetingId,
      userId: user?.id,
      displayName,
      avatarUrl: user?.avatar,
    })
    return response.data
  },

  // Leave a meeting
  async leaveMeeting(meetingId: string): Promise<void> {
    await api.post(`/meetings/${meetingId}/leave`)
  },

  // End a meeting (host only)
  async endMeeting(meetingId: string): Promise<void> {
    await api.post(`/meetings/${meetingId}/end`)
  },

  // Update participant media state
  async updateMediaState(meetingId: string, state: MediaState): Promise<void> {
    await api.patch(`/meetings/${meetingId}/media`, state)
  },

  // Start screen sharing
  async startScreenShare(meetingId: string): Promise<void> {
    await api.post(`/meetings/${meetingId}/screen-share/start`)
  },

  // Stop screen sharing
  async stopScreenShare(meetingId: string): Promise<void> {
    await api.post(`/meetings/${meetingId}/screen-share/stop`)
  },

  // Start recording (host only)
  async startRecording(meetingId: string): Promise<void> {
    await api.post(`/meetings/${meetingId}/recording/start`)
  },

  // Stop recording (host only)
  async stopRecording(meetingId: string): Promise<void> {
    await api.post(`/meetings/${meetingId}/recording/stop`)
  },

  // Get meeting recordings
  async getRecordings(meetingId: string): Promise<MeetingRecording[]> {
    const response = await api.get(`/meetings/${meetingId}/recordings`)
    return response.data
  },

  // Get meeting participants
  async getParticipants(meetingId: string): Promise<MeetingParticipant[]> {
    const response = await api.get(`/meetings/${meetingId}/participants`)
    return response.data
  },

  // Kick participant (host only)
  async kickParticipant(meetingId: string, userId: string): Promise<void> {
    await api.delete(`/meetings/${meetingId}/participants/${userId}`)
  },

  // Mute participant (host only)
  async muteParticipant(meetingId: string, userId: string): Promise<void> {
    await api.post(`/meetings/${meetingId}/participants/${userId}/mute`)
  },

  // Promote to co-host
  async promoteToCoHost(meetingId: string, userId: string): Promise<void> {
    await api.patch(`/meetings/${meetingId}/participants/${userId}`, { role: 'co-host' })
  },
}

export default meetingService
