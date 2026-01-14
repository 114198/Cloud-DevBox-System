// Package models provides data models for the realtime meeting service.
package models

import (
	"time"

	"github.com/google/uuid"
)

// MeetingType represents the type of meeting
type MeetingType string

const (
	MeetingTypeAudio       MeetingType = "audio"
	MeetingTypeVideo       MeetingType = "video"
	MeetingTypeScreenShare MeetingType = "screen_share"
)

// MeetingStatus represents the status of a meeting
type MeetingStatus string

const (
	MeetingStatusScheduled MeetingStatus = "scheduled"
	MeetingStatusActive    MeetingStatus = "active"
	MeetingStatusEnded     MeetingStatus = "ended"
	MeetingStatusCancelled MeetingStatus = "cancelled"
)

// MeetingParticipantRole represents the role of a meeting participant
type MeetingParticipantRole string

const (
	MeetingRoleHost        MeetingParticipantRole = "host"
	MeetingRoleCoHost      MeetingParticipantRole = "co-host"
	MeetingRoleParticipant MeetingParticipantRole = "participant"
)

// SignalingMessageType represents the type of WebRTC signaling message
type SignalingMessageType string

const (
	SignalTypeOffer         SignalingMessageType = "offer"
	SignalTypeAnswer        SignalingMessageType = "answer"
	SignalTypeCandidate     SignalingMessageType = "ice-candidate"
	SignalTypeJoin          SignalingMessageType = "join"
	SignalTypeLeave         SignalingMessageType = "leave"
	SignalTypeMute          SignalingMessageType = "mute"
	SignalTypeUnmute        SignalingMessageType = "unmute"
	SignalTypeScreenShare   SignalingMessageType = "screen-share"
	SignalTypeStopShare     SignalingMessageType = "stop-share"
	SignalTypeQualityChange SignalingMessageType = "quality-change"
	SignalTypeRecordStart   SignalingMessageType = "record-start"
	SignalTypeRecordStop    SignalingMessageType = "record-stop"
	SignalTypeError         SignalingMessageType = "error"
	SignalTypeUserJoined    SignalingMessageType = "user-joined"
	SignalTypeUserLeft      SignalingMessageType = "user-left"
	SignalTypeMediaState    SignalingMessageType = "media-state"
)

// VideoQuality represents video quality levels
type VideoQuality string

const (
	VideoQuality1080p     VideoQuality = "1080p"
	VideoQuality720p      VideoQuality = "720p"
	VideoQuality480p      VideoQuality = "480p"
	VideoQuality360p      VideoQuality = "360p"
	VideoQualityAudioOnly VideoQuality = "audio_only"
)

// Meeting represents a video/audio meeting session
type Meeting struct {
	ID            uuid.UUID            `json:"id"`
	EnvironmentID uuid.UUID            `json:"environmentId"`
	HostUserID    uuid.UUID            `json:"hostUserId"`
	Title         string               `json:"title"`
	Type          MeetingType          `json:"type"`
	Status        MeetingStatus        `json:"status"`
	Settings      MeetingSettings      `json:"settings"`
	Participants  []MeetingParticipant `json:"participants"`
	Recording     *MeetingRecording    `json:"recording,omitempty"`
	CreatedAt     time.Time            `json:"createdAt"`
	StartedAt     *time.Time           `json:"startedAt,omitempty"`
	EndedAt       *time.Time           `json:"endedAt,omitempty"`
}

// MeetingSettings represents meeting configuration settings
type MeetingSettings struct {
	MaxParticipants         int  `json:"maxParticipants"`
	AllowRecording          bool `json:"allowRecording"`
	AllowScreenShare        bool `json:"allowScreenShare"`
	EnableTranscription     bool `json:"enableTranscription"`
	EnableVirtualBackground bool `json:"enableVirtualBackground"`
	WaitingRoom             bool `json:"waitingRoom"`
	MuteOnJoin              bool `json:"muteOnJoin"`
	VideoOnJoin             bool `json:"videoOnJoin"`
}

// DefaultMeetingSettings returns default meeting settings
func DefaultMeetingSettings() MeetingSettings {
	return MeetingSettings{
		MaxParticipants:         10,
		AllowRecording:          true,
		AllowScreenShare:        true,
		EnableTranscription:     false,
		EnableVirtualBackground: true,
		WaitingRoom:             false,
		MuteOnJoin:              false,
		VideoOnJoin:             true,
	}
}

// MeetingParticipant represents a participant in a meeting
type MeetingParticipant struct {
	ID            uuid.UUID              `json:"id"`
	MeetingID     uuid.UUID              `json:"meetingId"`
	UserID        uuid.UUID              `json:"userId"`
	DisplayName   string                 `json:"displayName"`
	AvatarURL     string                 `json:"avatarUrl,omitempty"`
	Role          MeetingParticipantRole `json:"role"`
	JoinedAt      time.Time              `json:"joinedAt"`
	LeftAt        *time.Time             `json:"leftAt,omitempty"`
	AudioEnabled  bool                   `json:"audioEnabled"`
	VideoEnabled  bool                   `json:"videoEnabled"`
	ScreenSharing bool                   `json:"screenSharing"`
	VideoQuality  VideoQuality           `json:"videoQuality"`
	ConnectionID  string                 `json:"connectionId,omitempty"`
}

// MeetingRecording represents a meeting recording
type MeetingRecording struct {
	ID            uuid.UUID `json:"id"`
	MeetingID     uuid.UUID `json:"meetingId"`
	Duration      int       `json:"duration"` // seconds
	FileSize      int64     `json:"fileSize"` // bytes
	Format        string    `json:"format"`
	StorageURL    string    `json:"storageUrl"`
	Transcription string    `json:"transcription,omitempty"`
	CreatedAt     time.Time `json:"createdAt"`
}

// SignalingMessage represents a WebRTC signaling message
type SignalingMessage struct {
	Type      SignalingMessageType `json:"type"`
	MeetingID string               `json:"meetingId"`
	UserID    string               `json:"userId"`
	TargetID  string               `json:"targetId,omitempty"`
	Payload   interface{}          `json:"payload"`
	Timestamp int64                `json:"timestamp"`
}

// SDPPayload represents SDP offer/answer payload
type SDPPayload struct {
	SDP  string `json:"sdp"`
	Type string `json:"type"` // "offer" or "answer"
}

// ICECandidatePayload represents ICE candidate payload
type ICECandidatePayload struct {
	Candidate     string `json:"candidate"`
	SDPMid        string `json:"sdpMid"`
	SDPMLineIndex int    `json:"sdpMLineIndex"`
}

// MediaStatePayload represents media state change payload
type MediaStatePayload struct {
	AudioEnabled  bool         `json:"audioEnabled"`
	VideoEnabled  bool         `json:"videoEnabled"`
	ScreenSharing bool         `json:"screenSharing"`
	VideoQuality  VideoQuality `json:"videoQuality,omitempty"`
}

// QualityChangePayload represents quality change payload
type QualityChangePayload struct {
	Quality   VideoQuality `json:"quality"`
	Bandwidth int          `json:"bandwidth"` // kbps
}

// ScreenSharePayload represents screen share payload
type ScreenSharePayload struct {
	Action     string `json:"action"`               // "start" or "stop"
	ShareType  string `json:"shareType,omitempty"`  // "screen", "window", "tab"
	Resolution string `json:"resolution,omitempty"` // "1080p", "720p"
	FrameRate  int    `json:"frameRate,omitempty"`
}

// CreateMeetingRequest represents a request to create a meeting
type CreateMeetingRequest struct {
	EnvironmentID string           `json:"environmentId" binding:"required"`
	HostUserID    string           `json:"hostUserId" binding:"required"`
	Title         string           `json:"title" binding:"required"`
	Type          MeetingType      `json:"type"`
	Settings      *MeetingSettings `json:"settings,omitempty"`
	Invitees      []string         `json:"invitees,omitempty"`
}

// JoinMeetingRequest represents a request to join a meeting
type JoinMeetingRequest struct {
	MeetingID   string `json:"meetingId" binding:"required"`
	UserID      string `json:"userId" binding:"required"`
	DisplayName string `json:"displayName" binding:"required"`
	AvatarURL   string `json:"avatarUrl,omitempty"`
}

// JoinMeetingResponse represents the response to a join meeting request
type JoinMeetingResponse struct {
	Meeting      *Meeting             `json:"meeting"`
	Participant  *MeetingParticipant  `json:"participant"`
	RTCConfig    *RTCConfiguration    `json:"rtcConfig"`
	Participants []MeetingParticipant `json:"participants"`
}

// RTCConfiguration represents WebRTC configuration
type RTCConfiguration struct {
	ICEServers         []ICEServer `json:"iceServers"`
	ICETransportPolicy string      `json:"iceTransportPolicy,omitempty"`
	BundlePolicy       string      `json:"bundlePolicy,omitempty"`
}

// ICEServer represents an ICE server configuration
type ICEServer struct {
	URLs       []string `json:"urls"`
	Username   string   `json:"username,omitempty"`
	Credential string   `json:"credential,omitempty"`
}

// DefaultRTCConfiguration returns default WebRTC configuration
func DefaultRTCConfiguration() *RTCConfiguration {
	return &RTCConfiguration{
		ICEServers: []ICEServer{
			{URLs: []string{"stun:stun.l.google.com:19302"}},
			{URLs: []string{"stun:stun1.l.google.com:19302"}},
		},
		ICETransportPolicy: "all",
		BundlePolicy:       "max-bundle",
	}
}

// MeetingError represents a meeting-related error
type MeetingError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

func (e *MeetingError) Error() string {
	if e == nil {
		return ""
	}
	if e.Details != "" {
		return e.Message + ": " + e.Details
	}
	return e.Message
}

// Common meeting error codes
const (
	ErrMeetingNotFound   = "MEETING_NOT_FOUND"
	ErrMeetingFull       = "MEETING_FULL"
	ErrMeetingEnded      = "MEETING_ENDED"
	ErrUnauthorized      = "UNAUTHORIZED"
	ErrConnectionFailed  = "CONNECTION_FAILED"
	ErrMediaAccessDenied = "MEDIA_ACCESS_DENIED"
	ErrScreenShareInUse  = "SCREEN_SHARE_IN_USE"
	ErrRecordingFailed   = "RECORDING_FAILED"
	ErrInvalidSignal     = "INVALID_SIGNAL"
)
