// Package services provides meeting recording functionality.
package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/cloud-devbox/services/realtime/internal/models"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// RecordingService handles meeting recording
type RecordingService struct {
	logger           *zap.Logger
	signalingService *SignalingService
	recordings       map[string]*RecordingSession
	mu               sync.RWMutex
	config           RecordingConfig
	storageService   StorageService
}

// RecordingConfig defines recording configuration
type RecordingConfig struct {
	MaxDuration     time.Duration // Maximum recording duration
	Format          string        // Recording format (mp4, webm)
	VideoCodec      string        // Video codec (h264, vp8, vp9)
	AudioCodec      string        // Audio codec (opus, aac)
	Resolution      string        // Recording resolution
	FrameRate       int           // Recording frame rate
	StoragePath     string        // Base storage path
	RetentionDays   int           // Days to retain recordings
}

// DefaultRecordingConfig returns default recording configuration
func DefaultRecordingConfig() RecordingConfig {
	return RecordingConfig{
		MaxDuration:   4 * time.Hour,
		Format:        "mp4",
		VideoCodec:    "h264",
		AudioCodec:    "aac",
		Resolution:    "1080p",
		FrameRate:     30,
		StoragePath:   "/var/lib/devbox/recordings",
		RetentionDays: 30,
	}
}

// RecordingSession represents an active recording session
type RecordingSession struct {
	ID            uuid.UUID       `json:"id"`
	MeetingID     uuid.UUID       `json:"meetingId"`
	StartedBy     uuid.UUID       `json:"startedBy"`
	Status        RecordingStatus `json:"status"`
	Config        RecordingConfig `json:"config"`
	StartedAt     time.Time       `json:"startedAt"`
	StoppedAt     *time.Time      `json:"stoppedAt,omitempty"`
	Duration      time.Duration   `json:"duration"`
	FileSize      int64           `json:"fileSize"`
	FilePath      string          `json:"filePath"`
	StorageURL    string          `json:"storageUrl,omitempty"`
	Transcription string          `json:"transcription,omitempty"`
	Error         string          `json:"error,omitempty"`
}

// RecordingStatus represents the status of a recording
type RecordingStatus string

const (
	RecordingStatusPending    RecordingStatus = "pending"
	RecordingStatusRecording  RecordingStatus = "recording"
	RecordingStatusProcessing RecordingStatus = "processing"
	RecordingStatusCompleted  RecordingStatus = "completed"
	RecordingStatusFailed     RecordingStatus = "failed"
)

// StorageService interface for storing recordings
type StorageService interface {
	Upload(ctx context.Context, filePath string, destPath string) (string, error)
	Delete(ctx context.Context, path string) error
	GetURL(path string) string
}

// LocalStorageService implements StorageService for local storage
type LocalStorageService struct {
	basePath string
	baseURL  string
}

// NewLocalStorageService creates a new local storage service
func NewLocalStorageService(basePath string, baseURL string) *LocalStorageService {
	return &LocalStorageService{
		basePath: basePath,
		baseURL:  baseURL,
	}
}

// Upload uploads a file to local storage
func (s *LocalStorageService) Upload(ctx context.Context, filePath string, destPath string) (string, error) {
	// In production, this would copy the file to the storage location
	return s.GetURL(destPath), nil
}

// Delete deletes a file from local storage
func (s *LocalStorageService) Delete(ctx context.Context, path string) error {
	// In production, this would delete the file
	return nil
}

// GetURL returns the URL for a file
func (s *LocalStorageService) GetURL(path string) string {
	return fmt.Sprintf("%s/%s", s.baseURL, path)
}

// NewRecordingService creates a new recording service
func NewRecordingService(logger *zap.Logger, signalingService *SignalingService, storageService StorageService) *RecordingService {
	return &RecordingService{
		logger:           logger,
		signalingService: signalingService,
		recordings:       make(map[string]*RecordingSession),
		config:           DefaultRecordingConfig(),
		storageService:   storageService,
	}
}

// SetConfig updates the recording configuration
func (s *RecordingService) SetConfig(config RecordingConfig) {
	s.config = config
}

// StartRecording starts recording a meeting
func (s *RecordingService) StartRecording(ctx context.Context, meetingID string, userID string) (*RecordingSession, error) {
	meeting, err := s.signalingService.GetMeeting(meetingID)
	if err != nil {
		return nil, err
	}

	// Check if recording is allowed
	if !meeting.Settings.AllowRecording {
		return nil, &models.MeetingError{
			Code:    models.ErrUnauthorized,
			Message: "Recording is not allowed in this meeting",
		}
	}

	// Check if user is host or co-host
	isAuthorized := meeting.HostUserID.String() == userID
	if !isAuthorized {
		for _, p := range meeting.Participants {
			if p.UserID.String() == userID && p.Role == models.MeetingRoleCoHost {
				isAuthorized = true
				break
			}
		}
	}

	if !isAuthorized {
		return nil, &models.MeetingError{
			Code:    models.ErrUnauthorized,
			Message: "Only host or co-host can start recording",
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if already recording
	for _, session := range s.recordings {
		if session.MeetingID.String() == meetingID && 
		   (session.Status == RecordingStatusRecording || session.Status == RecordingStatusPending) {
			return nil, &models.MeetingError{
				Code:    models.ErrRecordingFailed,
				Message: "Meeting is already being recorded",
			}
		}
	}

	meetingUUID, _ := uuid.Parse(meetingID)
	userUUID, _ := uuid.Parse(userID)

	session := &RecordingSession{
		ID:        uuid.New(),
		MeetingID: meetingUUID,
		StartedBy: userUUID,
		Status:    RecordingStatusRecording,
		Config:    s.config,
		StartedAt: time.Now(),
		FilePath:  fmt.Sprintf("%s/%s/%s.%s", s.config.StoragePath, meetingID, uuid.New().String(), s.config.Format),
	}

	s.recordings[session.ID.String()] = session

	s.logger.Info("Recording started",
		zap.String("recordingId", session.ID.String()),
		zap.String("meetingId", meetingID),
		zap.String("startedBy", userID),
	)

	return session, nil
}

// StopRecording stops recording a meeting
func (s *RecordingService) StopRecording(ctx context.Context, meetingID string, userID string) (*RecordingSession, error) {
	meeting, err := s.signalingService.GetMeeting(meetingID)
	if err != nil {
		return nil, err
	}

	// Check if user is host or co-host
	isAuthorized := meeting.HostUserID.String() == userID
	if !isAuthorized {
		for _, p := range meeting.Participants {
			if p.UserID.String() == userID && p.Role == models.MeetingRoleCoHost {
				isAuthorized = true
				break
			}
		}
	}

	if !isAuthorized {
		return nil, &models.MeetingError{
			Code:    models.ErrUnauthorized,
			Message: "Only host or co-host can stop recording",
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Find active recording
	var activeSession *RecordingSession
	for _, session := range s.recordings {
		if session.MeetingID.String() == meetingID && session.Status == RecordingStatusRecording {
			activeSession = session
			break
		}
	}

	if activeSession == nil {
		return nil, &models.MeetingError{
			Code:    models.ErrRecordingFailed,
			Message: "No active recording found",
		}
	}

	now := time.Now()
	activeSession.StoppedAt = &now
	activeSession.Duration = now.Sub(activeSession.StartedAt)
	activeSession.Status = RecordingStatusProcessing

	s.logger.Info("Recording stopped",
		zap.String("recordingId", activeSession.ID.String()),
		zap.String("meetingId", meetingID),
		zap.Duration("duration", activeSession.Duration),
	)

	// Start async processing
	go s.processRecording(ctx, activeSession)

	return activeSession, nil
}

// processRecording processes a completed recording
func (s *RecordingService) processRecording(ctx context.Context, session *RecordingSession) {
	s.logger.Info("Processing recording",
		zap.String("recordingId", session.ID.String()),
	)

	// Simulate processing time
	time.Sleep(2 * time.Second)

	s.mu.Lock()
	defer s.mu.Unlock()

	// In production, this would:
	// 1. Transcode the recording if needed
	// 2. Generate thumbnails
	// 3. Upload to storage
	// 4. Generate transcription if enabled

	// Simulate file size (based on duration and quality)
	bytesPerSecond := 500000 // ~500KB/s for 720p
	session.FileSize = int64(session.Duration.Seconds()) * int64(bytesPerSecond)

	// Upload to storage
	if s.storageService != nil {
		destPath := fmt.Sprintf("recordings/%s/%s.%s", 
			session.MeetingID.String(), 
			session.ID.String(), 
			session.Config.Format)
		
		url, err := s.storageService.Upload(ctx, session.FilePath, destPath)
		if err != nil {
			session.Status = RecordingStatusFailed
			session.Error = err.Error()
			s.logger.Error("Failed to upload recording",
				zap.String("recordingId", session.ID.String()),
				zap.Error(err),
			)
			return
		}
		session.StorageURL = url
	}

	session.Status = RecordingStatusCompleted

	s.logger.Info("Recording processed",
		zap.String("recordingId", session.ID.String()),
		zap.Int64("fileSize", session.FileSize),
		zap.String("storageUrl", session.StorageURL),
	)
}

// GetRecording retrieves a recording by ID
func (s *RecordingService) GetRecording(recordingID string) (*RecordingSession, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	session, exists := s.recordings[recordingID]
	if !exists {
		return nil, &models.MeetingError{
			Code:    models.ErrRecordingFailed,
			Message: "Recording not found",
		}
	}

	return session, nil
}

// GetMeetingRecordings retrieves all recordings for a meeting
func (s *RecordingService) GetMeetingRecordings(meetingID string) ([]*RecordingSession, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	recordings := make([]*RecordingSession, 0)
	for _, session := range s.recordings {
		if session.MeetingID.String() == meetingID {
			recordings = append(recordings, session)
		}
	}

	return recordings, nil
}

// DeleteRecording deletes a recording
func (s *RecordingService) DeleteRecording(ctx context.Context, recordingID string, userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, exists := s.recordings[recordingID]
	if !exists {
		return &models.MeetingError{
			Code:    models.ErrRecordingFailed,
			Message: "Recording not found",
		}
	}

	// Check authorization
	meeting, err := s.signalingService.GetMeeting(session.MeetingID.String())
	if err != nil {
		return err
	}

	if meeting.HostUserID.String() != userID && session.StartedBy.String() != userID {
		return &models.MeetingError{
			Code:    models.ErrUnauthorized,
			Message: "Only host or recording owner can delete recording",
		}
	}

	// Delete from storage
	if s.storageService != nil && session.StorageURL != "" {
		if err := s.storageService.Delete(ctx, session.StorageURL); err != nil {
			s.logger.Error("Failed to delete recording from storage",
				zap.String("recordingId", recordingID),
				zap.Error(err),
			)
		}
	}

	delete(s.recordings, recordingID)

	s.logger.Info("Recording deleted",
		zap.String("recordingId", recordingID),
		zap.String("deletedBy", userID),
	)

	return nil
}

// GetActiveRecording returns the active recording for a meeting
func (s *RecordingService) GetActiveRecording(meetingID string) *RecordingSession {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, session := range s.recordings {
		if session.MeetingID.String() == meetingID && session.Status == RecordingStatusRecording {
			return session
		}
	}

	return nil
}

// IsRecording checks if a meeting is being recorded
func (s *RecordingService) IsRecording(meetingID string) bool {
	return s.GetActiveRecording(meetingID) != nil
}

// CleanupOldRecordings removes recordings older than retention period
func (s *RecordingService) CleanupOldRecordings(ctx context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cutoff := time.Now().AddDate(0, 0, -s.config.RetentionDays)
	
	for id, session := range s.recordings {
		if session.Status == RecordingStatusCompleted && session.StartedAt.Before(cutoff) {
			// Delete from storage
			if s.storageService != nil && session.StorageURL != "" {
				if err := s.storageService.Delete(ctx, session.StorageURL); err != nil {
					s.logger.Error("Failed to delete old recording",
						zap.String("recordingId", id),
						zap.Error(err),
					)
					continue
				}
			}

			delete(s.recordings, id)
			s.logger.Info("Cleaned up old recording",
				zap.String("recordingId", id),
				zap.Time("startedAt", session.StartedAt),
			)
		}
	}
}

// ConvertToMeetingRecording converts a RecordingSession to models.MeetingRecording
func (s *RecordingSession) ConvertToMeetingRecording() *models.MeetingRecording {
	return &models.MeetingRecording{
		ID:            s.ID,
		MeetingID:     s.MeetingID,
		Duration:      int(s.Duration.Seconds()),
		FileSize:      s.FileSize,
		Format:        s.Config.Format,
		StorageURL:    s.StorageURL,
		Transcription: s.Transcription,
		CreatedAt:     s.StartedAt,
	}
}
