/**
 * Meeting Store
 * Manages video/audio meeting state using Zustand
 * Requirements: 4.5.1, 4.5.2, 4.5.3 - Real-time meeting state management
 */

import { create } from 'zustand'
import {
  Meeting,
  MeetingParticipant,
  MeetingSignaling,
  MediaState,
  RTCConfiguration,
  JoinMeetingResponse,
  meetingService,
} from '@/services/meeting'

interface PeerConnection {
  peerId: string
  connection: RTCPeerConnection
  stream?: MediaStream
}

interface MeetingState {
  // Meeting state
  meeting: Meeting | null
  isInMeeting: boolean
  isJoining: boolean
  isLeaving: boolean
  error: string | null

  // Local media
  localStream: MediaStream | null
  screenStream: MediaStream | null
  audioEnabled: boolean
  videoEnabled: boolean
  isScreenSharing: boolean

  // Remote participants
  participants: MeetingParticipant[]
  remoteStreams: Map<string, MediaStream>

  // WebRTC
  peerConnections: Map<string, PeerConnection>
  rtcConfig: RTCConfiguration | null

  // Signaling
  signaling: MeetingSignaling | null

  // Recording
  isRecording: boolean

  // Actions
  createMeeting: (environmentId: string, title: string) => Promise<Meeting>
  joinMeeting: (meetingId: string, displayName: string) => Promise<void>
  leaveMeeting: () => Promise<void>
  endMeeting: () => Promise<void>

  // Media controls
  toggleAudio: () => void
  toggleVideo: () => void
  startScreenShare: () => Promise<void>
  stopScreenShare: () => void

  // Recording
  startRecording: () => Promise<void>
  stopRecording: () => Promise<void>

  // Internal actions
  setLocalStream: (stream: MediaStream | null) => void
  addRemoteStream: (peerId: string, stream: MediaStream) => void
  removeRemoteStream: (peerId: string) => void
  addParticipant: (participant: MeetingParticipant) => void
  removeParticipant: (userId: string) => void
  updateParticipantMedia: (userId: string, state: MediaState) => void
  setError: (error: string | null) => void
  reset: () => void
}

const initialState = {
  meeting: null,
  isInMeeting: false,
  isJoining: false,
  isLeaving: false,
  error: null,
  localStream: null,
  screenStream: null,
  audioEnabled: true,
  videoEnabled: true,
  isScreenSharing: false,
  participants: [],
  remoteStreams: new Map<string, MediaStream>(),
  peerConnections: new Map<string, PeerConnection>(),
  rtcConfig: null,
  signaling: null,
  isRecording: false,
}

export const useMeetingStore = create<MeetingState>((set, get) => ({
  ...initialState,

  createMeeting: async (environmentId: string, title: string) => {
    try {
      const meeting = await meetingService.createMeeting({
        environmentId,
        title,
        type: 'video',
      })
      set({ meeting })
      return meeting
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Failed to create meeting'
      set({ error: message })
      throw error
    }
  },

  joinMeeting: async (meetingId: string, displayName: string) => {
    const { reset } = get()
    reset()
    set({ isJoining: true, error: null })

    try {
      // Get local media stream
      const localStream = await navigator.mediaDevices.getUserMedia({
        audio: true,
        video: true,
      })
      set({ localStream, audioEnabled: true, videoEnabled: true })

      // Join meeting via API
      const response: JoinMeetingResponse = await meetingService.joinMeeting(meetingId, displayName)
      
      // Set up signaling
      const signaling = new MeetingSignaling(meetingId)
      
      signaling.onUserJoined = (participant) => {
        get().addParticipant(participant)
        // Create peer connection for new user
        createPeerConnection(participant.userId, get, set)
      }

      signaling.onUserLeft = (userId) => {
        get().removeParticipant(userId)
        get().removeRemoteStream(userId)
        closePeerConnection(userId, get)
      }

      signaling.onOffer = async (userId, sdp) => {
        await handleOffer(userId, sdp, get, set)
      }

      signaling.onAnswer = async (userId, sdp) => {
        await handleAnswer(userId, sdp, get)
      }

      signaling.onIceCandidate = async (userId, candidate) => {
        await handleIceCandidate(userId, candidate, get)
      }

      signaling.onMediaStateChange = (userId, state) => {
        get().updateParticipantMedia(userId, state)
      }

      signaling.onScreenShareStart = (userId) => {
        set((state) => ({
          participants: state.participants.map((p) =>
            p.userId === userId ? { ...p, screenSharing: true } : p
          ),
        }))
      }

      signaling.onScreenShareStop = (userId) => {
        set((state) => ({
          participants: state.participants.map((p) =>
            p.userId === userId ? { ...p, screenSharing: false } : p
          ),
        }))
      }

      signaling.onRecordingStart = () => set({ isRecording: true })
      signaling.onRecordingStop = () => set({ isRecording: false })

      signaling.onError = (error) => {
        set({ error: error.message })
      }

      await signaling.connect()

      set({
        meeting: response.meeting,
        participants: response.participants,
        rtcConfig: response.rtcConfig,
        signaling,
        isInMeeting: true,
        isJoining: false,
      })

      // Create peer connections for existing participants
      for (const participant of response.participants) {
        if (participant.userId !== response.participant.userId) {
          await createPeerConnection(participant.userId, get, set)
        }
      }
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Failed to join meeting'
      set({ error: message, isJoining: false })
      throw error
    }
  },

  leaveMeeting: async () => {
    const { meeting, signaling, localStream, screenStream, peerConnections } = get()
    set({ isLeaving: true })

    try {
      if (meeting) {
        await meetingService.leaveMeeting(meeting.id)
      }

      // Clean up
      signaling?.disconnect()
      localStream?.getTracks().forEach((track) => track.stop())
      screenStream?.getTracks().forEach((track) => track.stop())
      peerConnections.forEach((pc) => pc.connection.close())

      get().reset()
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Failed to leave meeting'
      set({ error: message, isLeaving: false })
    }
  },

  endMeeting: async () => {
    const { meeting } = get()
    if (!meeting) return

    try {
      await meetingService.endMeeting(meeting.id)
      await get().leaveMeeting()
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Failed to end meeting'
      set({ error: message })
    }
  },

  toggleAudio: () => {
    const { localStream, audioEnabled, signaling, videoEnabled, isScreenSharing } = get()
    if (localStream) {
      const audioTrack = localStream.getAudioTracks()[0]
      if (audioTrack) {
        audioTrack.enabled = !audioEnabled
        set({ audioEnabled: !audioEnabled })
        signaling?.sendMediaState({
          audioEnabled: !audioEnabled,
          videoEnabled,
          screenSharing: isScreenSharing,
        })
      }
    }
  },

  toggleVideo: () => {
    const { localStream, videoEnabled, signaling, audioEnabled, isScreenSharing } = get()
    if (localStream) {
      const videoTrack = localStream.getVideoTracks()[0]
      if (videoTrack) {
        videoTrack.enabled = !videoEnabled
        set({ videoEnabled: !videoEnabled })
        signaling?.sendMediaState({
          audioEnabled,
          videoEnabled: !videoEnabled,
          screenSharing: isScreenSharing,
        })
      }
    }
  },

  startScreenShare: async () => {
    const { signaling, meeting, peerConnections } = get()
    
    try {
      const screenStream = await navigator.mediaDevices.getDisplayMedia({
        video: { width: 1920, height: 1080, frameRate: 30 },
        audio: true,
      })

      set({ screenStream, isScreenSharing: true })

      // Replace video track in all peer connections
      const videoTrack = screenStream.getVideoTracks()[0]
      peerConnections.forEach((pc) => {
        const sender = pc.connection.getSenders().find((s) => s.track?.kind === 'video')
        if (sender) {
          sender.replaceTrack(videoTrack)
        }
      })

      // Handle screen share stop
      videoTrack.onended = () => {
        get().stopScreenShare()
      }

      signaling?.sendScreenShare('start')
      if (meeting) {
        await meetingService.startScreenShare(meeting.id)
      }
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Failed to start screen share'
      set({ error: message })
    }
  },

  stopScreenShare: () => {
    const { screenStream, localStream, signaling, meeting, peerConnections } = get()

    screenStream?.getTracks().forEach((track) => track.stop())

    // Restore camera video track
    if (localStream) {
      const videoTrack = localStream.getVideoTracks()[0]
      peerConnections.forEach((pc) => {
        const sender = pc.connection.getSenders().find((s) => s.track?.kind === 'video')
        if (sender && videoTrack) {
          sender.replaceTrack(videoTrack)
        }
      })
    }

    set({ screenStream: null, isScreenSharing: false })
    signaling?.sendScreenShare('stop')
    
    if (meeting) {
      meetingService.stopScreenShare(meeting.id).catch(console.error)
    }
  },

  startRecording: async () => {
    const { meeting } = get()
    if (!meeting) return

    try {
      await meetingService.startRecording(meeting.id)
      set({ isRecording: true })
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Failed to start recording'
      set({ error: message })
    }
  },

  stopRecording: async () => {
    const { meeting } = get()
    if (!meeting) return

    try {
      await meetingService.stopRecording(meeting.id)
      set({ isRecording: false })
    } catch (error) {
      const message = error instanceof Error ? error.message : 'Failed to stop recording'
      set({ error: message })
    }
  },

  setLocalStream: (stream) => set({ localStream: stream }),

  addRemoteStream: (peerId, stream) => {
    set((state) => {
      const newStreams = new Map(state.remoteStreams)
      newStreams.set(peerId, stream)
      return { remoteStreams: newStreams }
    })
  },

  removeRemoteStream: (peerId) => {
    set((state) => {
      const newStreams = new Map(state.remoteStreams)
      newStreams.delete(peerId)
      return { remoteStreams: newStreams }
    })
  },

  addParticipant: (participant) => {
    set((state) => ({
      participants: [...state.participants.filter((p) => p.userId !== participant.userId), participant],
    }))
  },

  removeParticipant: (userId) => {
    set((state) => ({
      participants: state.participants.filter((p) => p.userId !== userId),
    }))
  },

  updateParticipantMedia: (userId, mediaState) => {
    set((state) => ({
      participants: state.participants.map((p) =>
        p.userId === userId
          ? { ...p, audioEnabled: mediaState.audioEnabled, videoEnabled: mediaState.videoEnabled, screenSharing: mediaState.screenSharing }
          : p
      ),
    }))
  },

  setError: (error) => set({ error }),

  reset: () => {
    const { signaling, localStream, screenStream, peerConnections } = get()
    signaling?.disconnect()
    localStream?.getTracks().forEach((track) => track.stop())
    screenStream?.getTracks().forEach((track) => track.stop())
    peerConnections.forEach((pc) => pc.connection.close())
    set({ ...initialState, remoteStreams: new Map(), peerConnections: new Map() })
  },
}))


// Helper functions for WebRTC
async function createPeerConnection(
  peerId: string,
  get: () => MeetingState,
  set: (partial: Partial<MeetingState> | ((state: MeetingState) => Partial<MeetingState>)) => void
) {
  const { rtcConfig, localStream, signaling, peerConnections } = get()
  if (!rtcConfig || !localStream || !signaling) return

  const pc = new RTCPeerConnection(rtcConfig)

  // Add local tracks
  localStream.getTracks().forEach((track) => {
    pc.addTrack(track, localStream)
  })

  // Handle ICE candidates
  pc.onicecandidate = (event) => {
    if (event.candidate) {
      signaling.sendIceCandidate(peerId, event.candidate.toJSON())
    }
  }

  // Handle remote stream
  pc.ontrack = (event) => {
    const [remoteStream] = event.streams
    if (remoteStream) {
      get().addRemoteStream(peerId, remoteStream)
    }
  }

  // Store peer connection
  const newConnections = new Map(peerConnections)
  newConnections.set(peerId, { peerId, connection: pc })
  set({ peerConnections: newConnections })

  // Create and send offer
  const offer = await pc.createOffer()
  await pc.setLocalDescription(offer)
  signaling.sendOffer(peerId, offer)
}

async function handleOffer(
  userId: string,
  sdp: RTCSessionDescriptionInit,
  get: () => MeetingState,
  set: (partial: Partial<MeetingState> | ((state: MeetingState) => Partial<MeetingState>)) => void
) {
  const { rtcConfig, localStream, signaling, peerConnections } = get()
  if (!rtcConfig || !localStream || !signaling) return

  let pc = peerConnections.get(userId)?.connection

  if (!pc) {
    pc = new RTCPeerConnection(rtcConfig)

    localStream.getTracks().forEach((track) => {
      pc!.addTrack(track, localStream)
    })

    pc.onicecandidate = (event) => {
      if (event.candidate) {
        signaling.sendIceCandidate(userId, event.candidate.toJSON())
      }
    }

    pc.ontrack = (event) => {
      const [remoteStream] = event.streams
      if (remoteStream) {
        get().addRemoteStream(userId, remoteStream)
      }
    }

    const newConnections = new Map(peerConnections)
    newConnections.set(userId, { peerId: userId, connection: pc })
    set({ peerConnections: newConnections })
  }

  await pc.setRemoteDescription(new RTCSessionDescription(sdp))
  const answer = await pc.createAnswer()
  await pc.setLocalDescription(answer)
  signaling.sendAnswer(userId, answer)
}

async function handleAnswer(
  userId: string,
  sdp: RTCSessionDescriptionInit,
  get: () => MeetingState
) {
  const { peerConnections } = get()
  const pc = peerConnections.get(userId)?.connection
  if (pc) {
    await pc.setRemoteDescription(new RTCSessionDescription(sdp))
  }
}

async function handleIceCandidate(
  userId: string,
  candidate: RTCIceCandidateInit,
  get: () => MeetingState
) {
  const { peerConnections } = get()
  const pc = peerConnections.get(userId)?.connection
  if (pc) {
    await pc.addIceCandidate(new RTCIceCandidate(candidate))
  }
}

function closePeerConnection(userId: string, get: () => MeetingState) {
  const { peerConnections } = get()
  const pc = peerConnections.get(userId)
  if (pc) {
    pc.connection.close()
    peerConnections.delete(userId)
  }
}

// Selector hooks
export const useIsInMeeting = () => useMeetingStore((state) => state.isInMeeting)
export const useMeetingParticipants = () => useMeetingStore((state) => state.participants)
export const useLocalStream = () => useMeetingStore((state) => state.localStream)
export const useRemoteStreams = () => useMeetingStore((state) => state.remoteStreams)
export const useAudioEnabled = () => useMeetingStore((state) => state.audioEnabled)
export const useVideoEnabled = () => useMeetingStore((state) => state.videoEnabled)
export const useIsScreenSharing = () => useMeetingStore((state) => state.isScreenSharing)
export const useIsRecording = () => useMeetingStore((state) => state.isRecording)
export const useMeetingError = () => useMeetingStore((state) => state.error)
