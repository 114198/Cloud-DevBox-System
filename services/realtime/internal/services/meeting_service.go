// Package services provides business logic for the meeting management service.
package services

import (
	"context"
	"sync"
	"time"

	"github.com/cloud-devbox/services/realtime/internal/models"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// MeetingService handles meeting lifecycle management
type MeetingService struct {
	logger           *zap.Logger
	signalingService *SignalingService
	meetings         map[string]*models.Meeting
	mu               sync.RWMutex
}

// NewMeetingService creates a new meeting service
func NewMeetingService(logger *zap.Logger, signalingService *SignalingService) *MeetingService {
	return &MeetingService{
		logger:           logger,
		signalingService: signalingService,
		meetings:         make(map[string]*models.Meeting),
	}
}

// CreateMeeting creates a new meeting
func (s *MeetingService) CreateMeeting(ctx context.Context, req *models.CreateMeetingRequest) (*models.Meeting, error) {
	startTime := time.Now()
	
	meeting, err := s.signalingService.CreateMeeting(ctx, req)
	if err != nil {
		return nil, err
	}

	s.mu.Lock()
	s.meetings[meeting.ID.String()] = meeting
	s.mu.Unlock()

	duration := time.Since(startTime)
	s.logger.Info("Meeting created",
		zap.String("meetingId", meeting.ID.String()),
		zap.Duration("duration", duration),
	)

	return meeting, nil
}

// GetMeeting retrieves a meeting by ID
func (s *MeetingService) GetMeeting(meetingID string) (*models.Meeting, error) {
	return s.signalingService.GetMeeting(meetingID)
}

// EndMeeting ends a meeting
func (s *MeetingService) EndMeeting(meetingID string, userID string) error {
	err := s.signalingService.EndMeeting(meetingID, userID)
	if err != nil {
		return err
	}

	s.mu.Lock()
	delete(s.meetings, meetingID)
	s.mu.Unlock()

	return nil
}

// GetMeetingParticipants returns the list of active participants
func (s *MeetingService) GetMeetingParticipants(meetingID string) ([]models.MeetingParticipant, error) {
	return s.signalingService.GetMeetingParticipants(meetingID)
}

// ListMeetings returns all active meetings for an environment
func (s *MeetingService) ListMeetings(environmentID string) ([]*models.Meeting, error) {
	return s.signalingService.ListMeetings(environmentID)
}

// UpdateParticipantRole updates a participant's role
func (s *MeetingService) UpdateParticipantRole(meetingID string, participantID string, role models.MeetingParticipantRole, requestingUserID string) error {
	meeting, err := s.signalingService.GetMeeting(meetingID)
	if err != nil {
		return err
	}

	// Check if requesting user is host
	if meeting.HostUserID.String() != requestingUserID {
		return &models.MeetingError{
			Code:    models.ErrUnauthorized,
			Message: "Only host can change participant roles",
		}
	}

	// Find and update participant
	for i := range meeting.Participants {
		if meeting.Participants[i].ID.String() == participantID {
			meeting.Participants[i].Role = role
			s.logger.Info("Participant role updated",
				zap.String("meetingId", meetingID),
				zap.String("participantId", participantID),
				zap.String("newRole", string(role)),
			)
			return nil
		}
	}

	return &models.MeetingError{
		Code:    models.ErrMeetingNotFound,
		Message: "Participant not found",
	}
}

// KickParticipant removes a participant from the meeting
func (s *MeetingService) KickParticipant(meetingID string, participantID string, requestingUserID string) error {
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
			Message: "Only host or co-host can kick participants",
		}
	}

	// Find participant's connection ID and leave
	for _, p := range meeting.Participants {
		if p.ID.String() == participantID {
			return s.signalingService.LeaveMeeting(p.ConnectionID)
		}
	}

	return &models.MeetingError{
		Code:    models.ErrMeetingNotFound,
		Message: "Participant not found",
	}
}

// MuteParticipant mutes a participant
func (s *MeetingService) MuteParticipant(meetingID string, participantID string, requestingUserID string) error {
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
			Message: "Only host or co-host can mute participants",
		}
	}

	// Find and mute participant
	for i := range meeting.Participants {
		if meeting.Participants[i].ID.String() == participantID {
			meeting.Participants[i].AudioEnabled = false
			s.logger.Info("Participant muted",
				zap.String("meetingId", meetingID),
				zap.String("participantId", participantID),
			)
			return nil
		}
	}

	return &models.MeetingError{
		Code:    models.ErrMeetingNotFound,
		Message: "Participant not found",
	}
}

// MuteAll mutes all participants except host
func (s *MeetingService) MuteAll(meetingID string, requestingUserID string) error {
	meeting, err := s.signalingService.GetMeeting(meetingID)
	if err != nil {
		return err
	}

	// Check if requesting user is host
	if meeting.HostUserID.String() != requestingUserID {
		return &models.MeetingError{
			Code:    models.ErrUnauthorized,
			Message: "Only host can mute all participants",
		}
	}

	// Mute all non-host participants
	for i := range meeting.Participants {
		if meeting.Participants[i].Role != models.MeetingRoleHost {
			meeting.Participants[i].AudioEnabled = false
		}
	}

	s.logger.Info("All participants muted",
		zap.String("meetingId", meetingID),
	)

	return nil
}

// UpdateMeetingSettings updates meeting settings
func (s *MeetingService) UpdateMeetingSettings(meetingID string, settings models.MeetingSettings, requestingUserID string) error {
	meeting, err := s.signalingService.GetMeeting(meetingID)
	if err != nil {
		return err
	}

	// Check if requesting user is host
	if meeting.HostUserID.String() != requestingUserID {
		return &models.MeetingError{
			Code:    models.ErrUnauthorized,
			Message: "Only host can update meeting settings",
		}
	}

	meeting.Settings = settings
	s.logger.Info("Meeting settings updated",
		zap.String("meetingId", meetingID),
	)

	return nil
}

// GetMeetingStats returns meeting statistics
func (s *MeetingService) GetMeetingStats(meetingID string) (*MeetingStats, error) {
	meeting, err := s.signalingService.GetMeeting(meetingID)
	if err != nil {
		return nil, err
	}

	activeParticipants := 0
	audioEnabled := 0
	videoEnabled := 0
	screenSharing := 0

	for _, p := range meeting.Participants {
		if p.LeftAt == nil {
			activeParticipants++
			if p.AudioEnabled {
				audioEnabled++
			}
			if p.VideoEnabled {
				videoEnabled++
			}
			if p.ScreenSharing {
				screenSharing++
			}
		}
	}

	var duration time.Duration
	if meeting.StartedAt != nil {
		if meeting.EndedAt != nil {
			duration = meeting.EndedAt.Sub(*meeting.StartedAt)
		} else {
			duration = time.Since(*meeting.StartedAt)
		}
	}

	return &MeetingStats{
		MeetingID:          meeting.ID,
		Status:             meeting.Status,
		ActiveParticipants: activeParticipants,
		TotalParticipants:  len(meeting.Participants),
		AudioEnabled:       audioEnabled,
		VideoEnabled:       videoEnabled,
		ScreenSharing:      screenSharing,
		Duration:           duration,
		CreatedAt:          meeting.CreatedAt,
		StartedAt:          meeting.StartedAt,
	}, nil
}

// MeetingStats represents meeting statistics
type MeetingStats struct {
	MeetingID          uuid.UUID            `json:"meetingId"`
	Status             models.MeetingStatus `json:"status"`
	ActiveParticipants int                  `json:"activeParticipants"`
	TotalParticipants  int                  `json:"totalParticipants"`
	AudioEnabled       int                  `json:"audioEnabled"`
	VideoEnabled       int                  `json:"videoEnabled"`
	ScreenSharing      int                  `json:"screenSharing"`
	Duration           time.Duration        `json:"duration"`
	CreatedAt          time.Time            `json:"createdAt"`
	StartedAt          *time.Time           `json:"startedAt,omitempty"`
}

// ScheduleMeeting schedules a meeting for a future time
func (s *MeetingService) ScheduleMeeting(ctx context.Context, req *models.CreateMeetingRequest, scheduledTime time.Time) (*models.Meeting, error) {
	meeting, err := s.CreateMeeting(ctx, req)
	if err != nil {
		return nil, err
	}

	// Meeting is created in scheduled status
	s.logger.Info("Meeting scheduled",
		zap.String("meetingId", meeting.ID.String()),
		zap.Time("scheduledTime", scheduledTime),
	)

	return meeting, nil
}

// CleanupExpiredMeetings removes meetings that have been ended for more than 24 hours
func (s *MeetingService) CleanupExpiredMeetings() {
	s.mu.Lock()
	defer s.mu.Unlock()

	cutoff := time.Now().Add(-24 * time.Hour)
	for id, meeting := range s.meetings {
		if meeting.EndedAt != nil && meeting.EndedAt.Before(cutoff) {
			delete(s.meetings, id)
			s.logger.Info("Cleaned up expired meeting",
				zap.String("meetingId", id),
			)
		}
	}
}
