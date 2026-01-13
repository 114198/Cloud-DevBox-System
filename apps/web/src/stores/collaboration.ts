/**
 * Collaboration Store
 * Manages real-time collaboration state using Zustand
 */

import { create } from 'zustand'
import {
  CollaborationWebSocket,
  SessionParticipant,
  CursorPosition,
  Selection,
  EditOperation,
  DocumentState,
  JoinResponse,
} from '@/services/collaboration'

interface RemoteCursor {
  userId: string
  username: string
  displayName: string
  color: string
  position: CursorPosition
  selection?: Selection
  lastUpdated: number
}

interface CollaborationState {
  // Connection state
  isConnected: boolean
  isConnecting: boolean
  connectionError: string | null
  
  // Session state
  sessionId: string | null
  environmentId: string | null
  currentUserId: string | null
  currentUserColor: string | null
  
  // Participants
  participants: SessionParticipant[]
  
  // Remote cursors and selections
  remoteCursors: Map<string, RemoteCursor>
  
  // Document state
  documents: Map<string, DocumentState>
  pendingOperations: EditOperation[]
  
  // WebSocket instance
  ws: CollaborationWebSocket | null
  
  // Actions
  connect: (environmentId: string) => Promise<JoinResponse>
  disconnect: () => void
  
  // Participant actions
  setParticipants: (participants: SessionParticipant[]) => void
  addParticipant: (participant: SessionParticipant) => void
  removeParticipant: (userId: string) => void
  updateParticipantStatus: (userId: string, isOnline: boolean) => void
  
  // Cursor actions
  updateRemoteCursor: (userId: string, position: CursorPosition) => void
  updateRemoteSelection: (userId: string, selection: Selection) => void
  removeRemoteCursor: (userId: string) => void
  sendCursorUpdate: (position: CursorPosition) => void
  sendSelectionUpdate: (selection: Selection) => void
  
  // Edit actions
  sendEdit: (filePath: string, type: 'insert' | 'delete', position: number, content?: string, length?: number) => Promise<void>
  applyRemoteEdit: (operation: EditOperation) => void
  
  // Document actions
  setDocument: (filePath: string, state: DocumentState) => void
  requestSync: (filePath: string) => void
  
  // Error handling
  setError: (error: string | null) => void
  clearError: () => void
}

export const useCollaborationStore = create<CollaborationState>((set, get) => ({
  // Initial state
  isConnected: false,
  isConnecting: false,
  connectionError: null,
  sessionId: null,
  environmentId: null,
  currentUserId: null,
  currentUserColor: null,
  participants: [],
  remoteCursors: new Map(),
  documents: new Map(),
  pendingOperations: [],
  ws: null,

  // Connect to collaboration session
  connect: async (environmentId: string) => {
    const { ws: existingWs, disconnect } = get()
    
    // Disconnect existing connection if any
    if (existingWs) {
      disconnect()
    }

    set({ isConnecting: true, connectionError: null, environmentId })

    const ws = new CollaborationWebSocket(environmentId)

    // Set up event handlers
    ws.onConnect = (response: JoinResponse) => {
      set({
        isConnected: true,
        isConnecting: false,
        sessionId: response.sessionId,
        currentUserId: response.userId,
        currentUserColor: response.color,
        participants: response.participants,
      })
    }

    ws.onDisconnect = () => {
      set({ isConnected: false })
    }

    ws.onError = (error: Error) => {
      set({ connectionError: error.message, isConnecting: false })
    }

    ws.onEdit = (operation: EditOperation) => {
      get().applyRemoteEdit(operation)
    }

    ws.onCursor = (userId: string, position: CursorPosition) => {
      get().updateRemoteCursor(userId, position)
    }

    ws.onSelection = (userId: string, selection: Selection) => {
      get().updateRemoteSelection(userId, selection)
    }

    ws.onUserJoin = (participant: SessionParticipant) => {
      get().addParticipant(participant)
    }

    ws.onUserLeave = (userId: string) => {
      get().removeParticipant(userId)
      get().removeRemoteCursor(userId)
    }

    ws.onSync = (document: DocumentState, cursors: Record<string, CursorPosition>) => {
      get().setDocument(document.filePath, document)
      
      // Update cursors from sync
      Object.entries(cursors).forEach(([userId, position]) => {
        if (userId !== get().currentUserId) {
          get().updateRemoteCursor(userId, position)
        }
      })
    }

    set({ ws })

    try {
      const response = await ws.connect()
      return response
    } catch (error) {
      set({
        isConnecting: false,
        connectionError: error instanceof Error ? error.message : 'Connection failed',
      })
      throw error
    }
  },

  // Disconnect from collaboration session
  disconnect: () => {
    const { ws } = get()
    if (ws) {
      ws.disconnect()
    }
    set({
      isConnected: false,
      isConnecting: false,
      sessionId: null,
      currentUserId: null,
      currentUserColor: null,
      participants: [],
      remoteCursors: new Map(),
      documents: new Map(),
      pendingOperations: [],
      ws: null,
    })
  },

  // Participant management
  setParticipants: (participants: SessionParticipant[]) => {
    set({ participants })
  },

  addParticipant: (participant: SessionParticipant) => {
    set((state) => ({
      participants: [...state.participants.filter(p => p.userId !== participant.userId), participant],
    }))
  },

  removeParticipant: (userId: string) => {
    set((state) => ({
      participants: state.participants.map(p =>
        p.userId === userId ? { ...p, isOnline: false, leftAt: new Date().toISOString() } : p
      ),
    }))
  },

  updateParticipantStatus: (userId: string, isOnline: boolean) => {
    set((state) => ({
      participants: state.participants.map(p =>
        p.userId === userId ? { ...p, isOnline } : p
      ),
    }))
  },

  // Cursor management
  updateRemoteCursor: (userId: string, position: CursorPosition) => {
    set((state) => {
      const participant = state.participants.find(p => p.userId === userId)
      if (!participant) return state

      const newCursors = new Map(state.remoteCursors)
      const existing = newCursors.get(userId)
      
      newCursors.set(userId, {
        userId,
        username: participant.username,
        displayName: participant.displayName,
        color: participant.color,
        position,
        selection: existing?.selection,
        lastUpdated: Date.now(),
      })

      return { remoteCursors: newCursors }
    })
  },

  updateRemoteSelection: (userId: string, selection: Selection) => {
    set((state) => {
      const participant = state.participants.find(p => p.userId === userId)
      if (!participant) return state

      const newCursors = new Map(state.remoteCursors)
      const existing = newCursors.get(userId)
      
      newCursors.set(userId, {
        userId,
        username: participant.username,
        displayName: participant.displayName,
        color: participant.color,
        position: existing?.position || { filePath: selection.filePath, line: selection.startLine, column: selection.startColumn },
        selection,
        lastUpdated: Date.now(),
      })

      return { remoteCursors: newCursors }
    })
  },

  removeRemoteCursor: (userId: string) => {
    set((state) => {
      const newCursors = new Map(state.remoteCursors)
      newCursors.delete(userId)
      return { remoteCursors: newCursors }
    })
  },

  sendCursorUpdate: (position: CursorPosition) => {
    const { ws, isConnected } = get()
    if (ws && isConnected) {
      ws.sendCursor(position)
    }
  },

  sendSelectionUpdate: (selection: Selection) => {
    const { ws, isConnected } = get()
    if (ws && isConnected) {
      ws.sendSelection(selection)
    }
  },

  // Edit operations
  sendEdit: async (filePath: string, type: 'insert' | 'delete', position: number, content?: string, length?: number) => {
    const { ws, isConnected } = get()
    if (ws && isConnected) {
      await ws.sendEdit(filePath, type, position, content, length)
    }
  },

  applyRemoteEdit: (operation: EditOperation) => {
    set((state) => ({
      pendingOperations: [...state.pendingOperations, operation],
    }))
  },

  // Document management
  setDocument: (filePath: string, state: DocumentState) => {
    set((currentState) => {
      const newDocs = new Map(currentState.documents)
      newDocs.set(filePath, state)
      return { documents: newDocs }
    })
  },

  requestSync: (filePath: string) => {
    const { ws, isConnected } = get()
    if (ws && isConnected) {
      ws.requestSync(filePath)
    }
  },

  // Error handling
  setError: (error: string | null) => {
    set({ connectionError: error })
  },

  clearError: () => {
    set({ connectionError: null })
  },
}))

// Selector hooks
export const useIsCollaborating = () => useCollaborationStore((state) => state.isConnected)
export const useParticipants = () => useCollaborationStore((state) => state.participants)
export const useOnlineParticipants = () => useCollaborationStore((state) => 
  state.participants.filter(p => p.isOnline)
)
export const useRemoteCursors = () => useCollaborationStore((state) => state.remoteCursors)
export const useCurrentUserColor = () => useCollaborationStore((state) => state.currentUserColor)
export const useCollaborationError = () => useCollaborationStore((state) => state.connectionError)
