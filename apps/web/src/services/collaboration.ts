/**
 * Collaboration Service
 * Handles WebSocket connections and API calls for real-time collaboration
 */

import api from './api'
import { useAuthStore } from '@/stores/auth'

// Types
export interface CollaborationSession {
  id: string
  environmentId: string
  createdBy: string
  status: 'active' | 'ended' | 'expired'
  settings: SessionSettings
  participants: SessionParticipant[]
  createdAt: string
  updatedAt: string
  expiresAt?: string
}

export interface SessionSettings {
  maxParticipants: number
  allowEditing: boolean
  allowCursors: boolean
  autoSave: boolean
  autoSaveInterval: number
}

export interface SessionParticipant {
  id: string
  sessionId: string
  userId: string
  username: string
  displayName: string
  avatarUrl?: string
  role: 'host' | 'participant' | 'viewer'
  cursorPosition?: CursorPosition
  selection?: Selection
  color: string
  joinedAt: string
  leftAt?: string
  isOnline: boolean
}

export interface CursorPosition {
  filePath: string
  line: number
  column: number
  offset?: number
}

export interface Selection {
  filePath: string
  startLine: number
  startColumn: number
  endLine: number
  endColumn: number
  startOffset?: number
  endOffset?: number
}

export interface EditOperation {
  id: string
  userId: string
  filePath: string
  type: 'insert' | 'delete' | 'retain'
  position: number
  content?: string
  length?: number
  version: number
  timestamp: number
}

export interface WebSocketMessage {
  type: MessageType
  sessionId?: string
  environmentId?: string
  userId?: string
  timestamp: number
  data?: unknown
  sequence?: number
  ackId?: string
}

export type MessageType =
  | 'connect'
  | 'disconnect'
  | 'ping'
  | 'pong'
  | 'join'
  | 'leave'
  | 'edit'
  | 'cursor'
  | 'selection'
  | 'sync'
  | 'ack'
  | 'error'
  | 'user_list'
  | 'history'
  | 'conflict'
  | 'resolution'

export interface JoinResponse {
  sessionId: string
  userId: string
  color: string
  participants: SessionParticipant[]
  document?: DocumentState
}

export interface DocumentState {
  filePath: string
  content: string
  version: number
  lastUpdated: number
}

export interface InviteLink {
  id: string
  sessionId: string
  environmentId: string
  token: string
  role: 'participant' | 'viewer'
  expiresAt: string
  maxUses: number
  usedCount: number
  createdBy: string
  createdAt: string
}

// User colors for collaboration
export const USER_COLORS = [
  '#FF6B6B', '#4ECDC4', '#45B7D1', '#96CEB4',
  '#FFEAA7', '#DDA0DD', '#98D8C8', '#F7DC6F',
  '#BB8FCE', '#85C1E9', '#F8B500', '#00CED1',
]

export function getUserColor(index: number): string {
  return USER_COLORS[index % USER_COLORS.length]
}

// WebSocket connection manager
export class CollaborationWebSocket {
  private ws: WebSocket | null = null
  private sessionId: string | null = null
  private environmentId: string
  private reconnectAttempts = 0
  private maxReconnectAttempts = 5
  private reconnectDelay = 1000
  private pingInterval: ReturnType<typeof setInterval> | null = null
  private messageQueue: WebSocketMessage[] = []
  private sequence = 0
  private pendingAcks = new Map<string, { resolve: () => void; reject: (err: Error) => void }>()

  // Event handlers
  public onConnect?: (response: JoinResponse) => void
  public onDisconnect?: () => void
  public onError?: (error: Error) => void
  public onEdit?: (data: EditOperation) => void
  public onCursor?: (userId: string, position: CursorPosition) => void
  public onSelection?: (userId: string, selection: Selection) => void
  public onUserJoin?: (participant: SessionParticipant) => void
  public onUserLeave?: (userId: string) => void
  public onSync?: (document: DocumentState, cursors: Record<string, CursorPosition>) => void
  public onConflict?: (conflict: unknown) => void

  constructor(environmentId: string) {
    this.environmentId = environmentId
  }

  connect(): Promise<JoinResponse> {
    return new Promise((resolve, reject) => {
      const { user, accessToken } = useAuthStore.getState()
      if (!user || !accessToken) {
        reject(new Error('User not authenticated'))
        return
      }

      const wsUrl = this.buildWebSocketUrl(user, accessToken)
      this.ws = new WebSocket(wsUrl)

      this.ws.onopen = () => {
        this.reconnectAttempts = 0
        this.startPingInterval()
        this.flushMessageQueue()
      }

      this.ws.onmessage = (event) => {
        try {
          const message = JSON.parse(event.data) as WebSocketMessage
          this.handleMessage(message, resolve)
        } catch (err) {
          console.error('Failed to parse WebSocket message:', err)
        }
      }

      this.ws.onerror = (event) => {
        console.error('WebSocket error:', event)
        this.onError?.(new Error('WebSocket connection error'))
      }

      this.ws.onclose = () => {
        this.stopPingInterval()
        this.onDisconnect?.()
        this.attemptReconnect()
      }

      // Timeout for initial connection
      setTimeout(() => {
        if (this.ws?.readyState !== WebSocket.OPEN) {
          reject(new Error('Connection timeout'))
        }
      }, 10000)
    })
  }

  private buildWebSocketUrl(user: { id: string; username: string; displayName: string; avatar?: string }, token: string): string {
    const baseUrl = import.meta.env.VITE_WS_URL || `ws://${window.location.host}`
    const params = new URLSearchParams({
      userId: user.id,
      username: user.username,
      displayName: user.displayName,
      avatarUrl: user.avatar || '',
      token,
    })
    return `${baseUrl}/api/v1/collaboration/${this.environmentId}/ws?${params.toString()}`
  }

  private handleMessage(message: WebSocketMessage, connectResolve?: (response: JoinResponse) => void) {
    switch (message.type) {
      case 'connect':
        // Handle legacy connect message
        break

      case 'pong':
        // Pong received, connection is alive
        break

      case 'ack':
        if (message.ackId) {
          const pending = this.pendingAcks.get(message.ackId)
          if (pending) {
            pending.resolve()
            this.pendingAcks.delete(message.ackId)
          }
        }
        break

      case 'edit':
        if (message.data) {
          this.onEdit?.(message.data as EditOperation)
        }
        break

      case 'cursor':
        if (message.userId && message.data) {
          this.onCursor?.(message.userId, message.data as CursorPosition)
        }
        break

      case 'selection':
        if (message.userId && message.data) {
          this.onSelection?.(message.userId, message.data as Selection)
        }
        break

      case 'join':
        if (message.data) {
          this.onUserJoin?.(message.data as SessionParticipant)
        }
        break

      case 'leave':
        if (message.userId) {
          this.onUserLeave?.(message.userId)
        }
        break

      case 'sync':
        if (message.data) {
          const syncData = message.data as { document: DocumentState; cursors: Record<string, CursorPosition> }
          this.onSync?.(syncData.document, syncData.cursors)
        }
        break

      case 'conflict':
        this.onConflict?.(message.data)
        break

      case 'error':
        const errorData = message.data as { code: string; message: string }
        this.onError?.(new Error(errorData?.message || 'Unknown error'))
        break

      default:
        // Handle connected message (initial join response)
        if ((message as unknown as { type: string }).type === 'connected') {
          const response = message as unknown as JoinResponse & { type: string }
          this.sessionId = response.sessionId
          this.onConnect?.(response)
          connectResolve?.(response)
        }
    }
  }

  disconnect() {
    this.stopPingInterval()
    if (this.ws) {
      this.ws.close()
      this.ws = null
    }
    this.sessionId = null
  }

  private startPingInterval() {
    this.pingInterval = setInterval(() => {
      this.send({ type: 'ping', timestamp: Date.now() })
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

  private flushMessageQueue() {
    while (this.messageQueue.length > 0) {
      const message = this.messageQueue.shift()
      if (message) {
        this.sendImmediate(message)
      }
    }
  }

  private sendImmediate(message: WebSocketMessage) {
    if (this.ws?.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify(message))
    }
  }

  send(message: Omit<WebSocketMessage, 'sequence' | 'timestamp'>): Promise<void> {
    return new Promise((resolve, reject) => {
      const fullMessage: WebSocketMessage = {
        ...message,
        sessionId: this.sessionId || undefined,
        environmentId: this.environmentId,
        sequence: ++this.sequence,
        timestamp: Date.now(),
      }

      if (message.type === 'edit') {
        // Generate ack ID for edit operations
        const ackId = `${this.sequence}-${Date.now()}`
        fullMessage.ackId = ackId
        this.pendingAcks.set(ackId, { resolve, reject })

        // Timeout for ack
        setTimeout(() => {
          if (this.pendingAcks.has(ackId)) {
            this.pendingAcks.delete(ackId)
            reject(new Error('Edit operation timeout'))
          }
        }, 5000)
      } else {
        resolve()
      }

      if (this.ws?.readyState === WebSocket.OPEN) {
        this.sendImmediate(fullMessage)
      } else {
        this.messageQueue.push(fullMessage)
      }
    })
  }

  sendEdit(filePath: string, type: 'insert' | 'delete', position: number, content?: string, length?: number) {
    return this.send({
      type: 'edit',
      data: { filePath, type, position, content, length },
    })
  }

  sendCursor(position: CursorPosition) {
    return this.send({
      type: 'cursor',
      data: position,
    })
  }

  sendSelection(selection: Selection) {
    return this.send({
      type: 'selection',
      data: selection,
    })
  }

  requestSync(filePath: string) {
    return this.send({
      type: 'sync',
      data: { filePath },
    })
  }

  get isConnected(): boolean {
    return this.ws?.readyState === WebSocket.OPEN
  }

  get currentSessionId(): string | null {
    return this.sessionId
  }
}

// REST API functions
export const collaborationService = {
  // Create a new collaboration session
  async createSession(environmentId: string, settings?: Partial<SessionSettings>): Promise<CollaborationSession> {
    const { user } = useAuthStore.getState()
    const response = await api.post('/collaboration/sessions', {
      environmentId,
      userId: user?.id,
      settings,
    })
    return response.data
  },

  // Get session by ID
  async getSession(sessionId: string): Promise<CollaborationSession> {
    const response = await api.get(`/collaboration/sessions/${sessionId}`)
    return response.data
  },

  // Get session for environment
  async getSessionForEnvironment(environmentId: string): Promise<CollaborationSession | null> {
    try {
      const response = await api.get(`/collaboration/environments/${environmentId}/session`)
      return response.data
    } catch {
      return null
    }
  },

  // End a session
  async endSession(sessionId: string): Promise<void> {
    await api.post(`/collaboration/sessions/${sessionId}/end`)
  },

  // Get session participants
  async getParticipants(sessionId: string): Promise<SessionParticipant[]> {
    const response = await api.get(`/collaboration/sessions/${sessionId}/participants`)
    return response.data
  },

  // Get session history
  async getHistory(sessionId: string, limit = 100, offset = 0): Promise<unknown[]> {
    const response = await api.get(`/collaboration/sessions/${sessionId}/history`, {
      params: { limit, offset },
    })
    return response.data
  },

  // Create invite link
  async createInviteLink(
    sessionId: string,
    environmentId: string,
    role: 'participant' | 'viewer' = 'participant',
    expiresInHours = 24,
    maxUses = 10
  ): Promise<InviteLink> {
    const response = await api.post('/collaboration/invites', {
      sessionId,
      environmentId,
      role,
      expiresInHours,
      maxUses,
    })
    return response.data
  },

  // Get invite link details
  async getInviteLink(token: string): Promise<InviteLink> {
    const response = await api.get(`/collaboration/invites/${token}`)
    return response.data
  },

  // Accept invite
  async acceptInvite(token: string): Promise<{ sessionId: string; environmentId: string }> {
    const response = await api.post(`/collaboration/invites/${token}/accept`)
    return response.data
  },

  // Revoke invite link
  async revokeInviteLink(inviteId: string): Promise<void> {
    await api.delete(`/collaboration/invites/${inviteId}`)
  },

  // Get active invites for session
  async getSessionInvites(sessionId: string): Promise<InviteLink[]> {
    const response = await api.get(`/collaboration/sessions/${sessionId}/invites`)
    return response.data
  },

  // Update participant role
  async updateParticipantRole(sessionId: string, userId: string, role: 'participant' | 'viewer'): Promise<void> {
    await api.patch(`/collaboration/sessions/${sessionId}/participants/${userId}`, { role })
  },

  // Remove participant
  async removeParticipant(sessionId: string, userId: string): Promise<void> {
    await api.delete(`/collaboration/sessions/${sessionId}/participants/${userId}`)
  },
}

export default collaborationService
