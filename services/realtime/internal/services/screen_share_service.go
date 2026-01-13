// Package services provides screen sharing functionality for meetings.
package services

import (
	"sync"
	"time"

	"github.com/cloud-devbox/services/realtime/internal/models"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// ScreenShareService handles screen sharing in meetings
type ScreenShareService struct {
	logger           *zap.Logger
	signalingService *SignalingService
	activeSessions   map[string]*ScreenShareSession
	mu               sync.RWMutex
}

// ScreenShareSession represents an active screen share session
type ScreenShareSession struct {
	ID            uuid.UUID         `json:"id"`
	MeetingID     uuid.UUID         `json:"meetingId"`
	ParticipantID uuid.UUID         `json:"participantId"`
	UserID        uuid.UUID         `json:"userId"`
	Config        ScreenShareConfig `json:"config"`
	StartedAt     time.Time         `json:"startedAt"`
	EndedAt       *time.Time        `json:"endedAt,omitempty"`
}

// ScreenShareConfig represents screen share configuration
type ScreenShareConfig struct {
	ShareType  string `json:"shareType"`  // "screen", "window", "tab"
	Resolution string `json:"resolution"` // "1080p", "720p", "480p"
	FrameRate  int    `json:"frameRate"`  // 30, 15, 5
	Audio      bool   `json:"audio"`      // share system audio
}

// DefaultScreenShareConfig returns default screen share configuration
func DefaultScreenShareConfig() ScreenShareConfig {
	return ScreenShareConfig{
		ShareType:  "screen",
		Resolution: "1080p",
		FrameRate:  30,
		Audio:      false,
	}
}

// NewScreenShareService creates a new screen share service
func NewScreenShareService(logger *zap.Logger, signalingService *SignalingService) *ScreenShareService {
	return &ScreenShareService{
		logger:           logger,
		signalingService: signalingService,
		activeSessions:   make(map[string]*ScreenShareSession),
	}
}

// StartScreenShare starts a screen share session
func (s *ScreenShareService) StartScreenShare(meetingID string, participantID string, userID string, config *ScreenShareConfig) (*ScreenShareSession, error) {
	meeting, err := s.signalingService.GetMeeting(meetingID)
	if err != nil {
		return nil, err
	}

	// Check if screen sharing is allowed
	if !meeting.Settings.AllowScreenShare {
		return nil, &models.MeetingError{
			Code:    models.ErrUnauthorized,
			Message: "Screen sharing is not allowed in this meeting",
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if someone else is already sharing
	for _, session := range s.activeSessions {
		if session.MeetingID.String() == meetingID && session.EndedAt == nil {
			if session.ParticipantID.String() != participantID {
				return nil, &models.MeetingError{
					Code:    models.ErrScreenShareInUse,
					Message: "Another participant is already sharing their screen",
				}
			}
			// Same participant, return existing session
			return session, nil
		}
	}

	// Use default config if not provided
	shareConfig := DefaultScreenShareConfig()
	if config != nil {
		if config.ShareType != "" {
			shareConfig.ShareType = config.ShareType
		}
		if config.Resolution != "" {
			shareConfig.Resolution = config.Resolution
		}
		if config.FrameRate > 0 {
			shareConfig.FrameRate = config.FrameRate
		}
		shareConfig.Audio = config.Audio
	}

	// Validate resolution for 1080p/30fps requirement
	if shareConfig.Resolution == "1080p" && shareConfig.FrameRate > 30 {
		shareConfig.FrameRate = 30
	}

	participantUUID, _ := uuid.Parse(participantID)
	userUUID, _ := uuid.Parse(userID)
	meetingUUID, _ := uuid.Parse(meetingID)

	session := &ScreenShareSession{
		ID:            uuid.New(),
		MeetingID:     meetingUUID,
		ParticipantID: participantUUID,
		UserID:        userUUID,
		Config:        shareConfig,
		StartedAt:     time.Now(),
	}

	s.activeSessions[session.ID.String()] = session

	s.logger.Info("Screen share started",
		zap.String("sessionId", session.ID.String()),
		zap.String("meetingId", meetingID),
		zap.String("participantId", participantID),
		zap.String("shareType", shareConfig.ShareType),
		zap.String("resolution", shareConfig.Resolution),
		zap.Int("frameRate", shareConfig.FrameRate),
	)

	return session, nil
}

// StopScreenShare stops a screen share session
func (s *ScreenShareService) StopScreenShare(meetingID string, participantID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for id, session := range s.activeSessions {
		if session.MeetingID.String() == meetingID && 
		   session.ParticipantID.String() == participantID && 
		   session.EndedAt == nil {
			now := time.Now()
			session.EndedAt = &now

			s.logger.Info("Screen share stopped",
				zap.String("sessionId", id),
				zap.String("meetingId", meetingID),
				zap.String("participantId", participantID),
				zap.Duration("duration", now.Sub(session.StartedAt)),
			)

			return nil
		}
	}

	return &models.MeetingError{
		Code:    models.ErrMeetingNotFound,
		Message: "No active screen share session found",
	}
}

// GetActiveScreenShare returns the active screen share session for a meeting
func (s *ScreenShareService) GetActiveScreenShare(meetingID string) *ScreenShareSession {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, session := range s.activeSessions {
		if session.MeetingID.String() == meetingID && session.EndedAt == nil {
			return session
		}
	}

	return nil
}

// UpdateScreenShareConfig updates the screen share configuration
func (s *ScreenShareService) UpdateScreenShareConfig(meetingID string, participantID string, config ScreenShareConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, session := range s.activeSessions {
		if session.MeetingID.String() == meetingID && 
		   session.ParticipantID.String() == participantID && 
		   session.EndedAt == nil {
			
			// Validate and apply config
			if config.Resolution != "" {
				session.Config.Resolution = config.Resolution
			}
			if config.FrameRate > 0 {
				session.Config.FrameRate = config.FrameRate
			}
			if config.ShareType != "" {
				session.Config.ShareType = config.ShareType
			}
			session.Config.Audio = config.Audio

			s.logger.Info("Screen share config updated",
				zap.String("meetingId", meetingID),
				zap.String("participantId", participantID),
				zap.String("resolution", session.Config.Resolution),
				zap.Int("frameRate", session.Config.FrameRate),
			)

			return nil
		}
	}

	return &models.MeetingError{
		Code:    models.ErrMeetingNotFound,
		Message: "No active screen share session found",
	}
}

// GetScreenShareStats returns statistics for a screen share session
func (s *ScreenShareService) GetScreenShareStats(sessionID string) (*ScreenShareStats, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	session, exists := s.activeSessions[sessionID]
	if !exists {
		return nil, &models.MeetingError{
			Code:    models.ErrMeetingNotFound,
			Message: "Screen share session not found",
		}
	}

	var duration time.Duration
	if session.EndedAt != nil {
		duration = session.EndedAt.Sub(session.StartedAt)
	} else {
		duration = time.Since(session.StartedAt)
	}

	return &ScreenShareStats{
		SessionID:  session.ID,
		MeetingID:  session.MeetingID,
		UserID:     session.UserID,
		Config:     session.Config,
		Duration:   duration,
		IsActive:   session.EndedAt == nil,
		StartedAt:  session.StartedAt,
		EndedAt:    session.EndedAt,
	}, nil
}

// ScreenShareStats represents screen share statistics
type ScreenShareStats struct {
	SessionID  uuid.UUID         `json:"sessionId"`
	MeetingID  uuid.UUID         `json:"meetingId"`
	UserID     uuid.UUID         `json:"userId"`
	Config     ScreenShareConfig `json:"config"`
	Duration   time.Duration     `json:"duration"`
	IsActive   bool              `json:"isActive"`
	StartedAt  time.Time         `json:"startedAt"`
	EndedAt    *time.Time        `json:"endedAt,omitempty"`
}

// CleanupEndedSessions removes ended sessions older than 1 hour
func (s *ScreenShareService) CleanupEndedSessions() {
	s.mu.Lock()
	defer s.mu.Unlock()

	cutoff := time.Now().Add(-1 * time.Hour)
	for id, session := range s.activeSessions {
		if session.EndedAt != nil && session.EndedAt.Before(cutoff) {
			delete(s.activeSessions, id)
			s.logger.Debug("Cleaned up ended screen share session",
				zap.String("sessionId", id),
			)
		}
	}
}

// ForceStopScreenShare forcefully stops a screen share (for host/co-host)
func (s *ScreenShareService) ForceStopScreenShare(meetingID string, requestingUserID string) error {
	meeting, err := s.signalingService.GetMeeting(meetingID)
	if err != nil {
		return err
	}

	// Check if requesting user is host or co-host
	isAuthorized := meeting.HostUserID.String() == requestingUserID
	if !isAuthorized {
		for _, p := range meeting.Participants {
			if p.UserID.String() == requestingUserID && p.Role == models.MeetingRoleCoHost {
				isAuthorized = true
				break
			}
		}
	}

	if !isAuthorized {
		return &models.MeetingError{
			Code:    models.ErrUnauthorized,
			Message: "Only host or co-host can force stop screen share",
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for id, session := range s.activeSessions {
		if session.MeetingID.String() == meetingID && session.EndedAt == nil {
			now := time.Now()
			session.EndedAt = &now

			s.logger.Info("Screen share force stopped",
				zap.String("sessionId", id),
				zap.String("meetingId", meetingID),
				zap.String("stoppedBy", requestingUserID),
			)

			return nil
		}
	}

	return nil // No active session to stop
}
