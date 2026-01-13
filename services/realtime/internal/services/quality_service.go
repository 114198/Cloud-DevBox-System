// Package services provides quality adaptation functionality for meetings.
package services

import (
	"sync"
	"time"

	"github.com/cloud-devbox/services/realtime/internal/models"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// QualityService handles adaptive video quality based on network conditions
type QualityService struct {
	logger           *zap.Logger
	signalingService *SignalingService
	participants     map[string]*ParticipantQualityState
	mu               sync.RWMutex
	config           QualityConfig
}

// QualityConfig defines quality adaptation thresholds
type QualityConfig struct {
	// Bandwidth thresholds in kbps
	Bandwidth1080p int // Minimum bandwidth for 1080p
	Bandwidth720p  int // Minimum bandwidth for 720p
	Bandwidth480p  int // Minimum bandwidth for 480p
	Bandwidth360p  int // Minimum bandwidth for 360p
	BandwidthAudio int // Minimum bandwidth for audio only

	// Packet loss thresholds (percentage)
	PacketLossWarning  float64 // Warning threshold
	PacketLossCritical float64 // Critical threshold - force downgrade

	// RTT thresholds in milliseconds
	RTTWarning  int // Warning threshold
	RTTCritical int // Critical threshold

	// Adaptation settings
	UpgradeDelay   time.Duration // Wait time before upgrading quality
	DowngradeDelay time.Duration // Wait time before downgrading quality
	SampleWindow   time.Duration // Window for averaging metrics
}

// DefaultQualityConfig returns default quality configuration
func DefaultQualityConfig() QualityConfig {
	return QualityConfig{
		Bandwidth1080p: 2500, // 2.5 Mbps
		Bandwidth720p:  1500, // 1.5 Mbps
		Bandwidth480p:  800,  // 800 Kbps
		Bandwidth360p:  400,  // 400 Kbps
		BandwidthAudio: 100,  // 100 Kbps

		PacketLossWarning:  2.0,  // 2%
		PacketLossCritical: 5.0,  // 5%

		RTTWarning:  150, // 150ms
		RTTCritical: 300, // 300ms

		UpgradeDelay:   10 * time.Second,
		DowngradeDelay: 3 * time.Second,
		SampleWindow:   5 * time.Second,
	}
}

// ParticipantQualityState tracks quality state for a participant
type ParticipantQualityState struct {
	ParticipantID   uuid.UUID
	MeetingID       uuid.UUID
	CurrentQuality  models.VideoQuality
	TargetQuality   models.VideoQuality
	Metrics         []NetworkMetrics
	LastUpgrade     time.Time
	LastDowngrade   time.Time
	StableCount     int
	UnstableCount   int
}

// NetworkMetrics represents network quality metrics
type NetworkMetrics struct {
	Timestamp       time.Time `json:"timestamp"`
	BandwidthKbps   int       `json:"bandwidthKbps"`
	PacketLoss      float64   `json:"packetLoss"`
	RTTMs           int       `json:"rttMs"`
	JitterMs        int       `json:"jitterMs"`
}

// QualityChangeEvent represents a quality change event
type QualityChangeEvent struct {
	ParticipantID   uuid.UUID           `json:"participantId"`
	MeetingID       uuid.UUID           `json:"meetingId"`
	PreviousQuality models.VideoQuality `json:"previousQuality"`
	NewQuality      models.VideoQuality `json:"newQuality"`
	Reason          string              `json:"reason"`
	Metrics         NetworkMetrics      `json:"metrics"`
	Timestamp       time.Time           `json:"timestamp"`
}

// NewQualityService creates a new quality service
func NewQualityService(logger *zap.Logger, signalingService *SignalingService) *QualityService {
	return &QualityService{
		logger:           logger,
		signalingService: signalingService,
		participants:     make(map[string]*ParticipantQualityState),
		config:           DefaultQualityConfig(),
	}
}

// SetConfig updates the quality configuration
func (s *QualityService) SetConfig(config QualityConfig) {
	s.config = config
}

// RegisterParticipant registers a participant for quality monitoring
func (s *QualityService) RegisterParticipant(meetingID string, participantID string, initialQuality models.VideoQuality) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := meetingID + ":" + participantID
	meetingUUID, _ := uuid.Parse(meetingID)
	participantUUID, _ := uuid.Parse(participantID)

	s.participants[key] = &ParticipantQualityState{
		ParticipantID:  participantUUID,
		MeetingID:      meetingUUID,
		CurrentQuality: initialQuality,
		TargetQuality:  initialQuality,
		Metrics:        make([]NetworkMetrics, 0),
		LastUpgrade:    time.Now(),
		LastDowngrade:  time.Now(),
	}

	s.logger.Debug("Registered participant for quality monitoring",
		zap.String("meetingId", meetingID),
		zap.String("participantId", participantID),
		zap.String("initialQuality", string(initialQuality)),
	)
}

// UnregisterParticipant removes a participant from quality monitoring
func (s *QualityService) UnregisterParticipant(meetingID string, participantID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := meetingID + ":" + participantID
	delete(s.participants, key)
}

// ProcessMetrics processes network metrics and determines quality adaptation
func (s *QualityService) ProcessMetrics(meetingID string, participantID string, metrics NetworkMetrics) (*QualityChangeEvent, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := meetingID + ":" + participantID
	state, exists := s.participants[key]
	if !exists {
		return nil, &models.MeetingError{
			Code:    models.ErrMeetingNotFound,
			Message: "Participant not registered for quality monitoring",
		}
	}

	// Add metrics to history
	metrics.Timestamp = time.Now()
	state.Metrics = append(state.Metrics, metrics)

	// Keep only metrics within sample window
	cutoff := time.Now().Add(-s.config.SampleWindow)
	filtered := make([]NetworkMetrics, 0)
	for _, m := range state.Metrics {
		if m.Timestamp.After(cutoff) {
			filtered = append(filtered, m)
		}
	}
	state.Metrics = filtered

	// Calculate average metrics
	avgMetrics := s.calculateAverageMetrics(state.Metrics)

	// Determine target quality based on metrics
	targetQuality := s.determineTargetQuality(avgMetrics)
	reason := ""

	// Check if quality change is needed
	if targetQuality != state.CurrentQuality {
		if s.isQualityHigher(targetQuality, state.CurrentQuality) {
			// Upgrading - check delay
			if time.Since(state.LastUpgrade) < s.config.UpgradeDelay {
				return nil, nil // Not enough time since last upgrade
			}
			state.StableCount++
			if state.StableCount < 3 {
				return nil, nil // Need more stable samples
			}
			reason = "bandwidth_recovered"
		} else {
			// Downgrading - check delay
			if time.Since(state.LastDowngrade) < s.config.DowngradeDelay {
				return nil, nil // Not enough time since last downgrade
			}
			reason = s.determineDowngradeReason(avgMetrics)
		}

		// Apply quality change
		previousQuality := state.CurrentQuality
		state.CurrentQuality = targetQuality
		state.TargetQuality = targetQuality
		state.StableCount = 0
		state.UnstableCount = 0

		if s.isQualityHigher(targetQuality, previousQuality) {
			state.LastUpgrade = time.Now()
		} else {
			state.LastDowngrade = time.Now()
		}

		event := &QualityChangeEvent{
			ParticipantID:   state.ParticipantID,
			MeetingID:       state.MeetingID,
			PreviousQuality: previousQuality,
			NewQuality:      targetQuality,
			Reason:          reason,
			Metrics:         avgMetrics,
			Timestamp:       time.Now(),
		}

		s.logger.Info("Quality changed",
			zap.String("meetingId", meetingID),
			zap.String("participantId", participantID),
			zap.String("from", string(previousQuality)),
			zap.String("to", string(targetQuality)),
			zap.String("reason", reason),
		)

		return event, nil
	}

	return nil, nil
}

// calculateAverageMetrics calculates average metrics from samples
func (s *QualityService) calculateAverageMetrics(metrics []NetworkMetrics) NetworkMetrics {
	if len(metrics) == 0 {
		return NetworkMetrics{}
	}

	var totalBandwidth int
	var totalPacketLoss float64
	var totalRTT int
	var totalJitter int

	for _, m := range metrics {
		totalBandwidth += m.BandwidthKbps
		totalPacketLoss += m.PacketLoss
		totalRTT += m.RTTMs
		totalJitter += m.JitterMs
	}

	count := len(metrics)
	return NetworkMetrics{
		Timestamp:     time.Now(),
		BandwidthKbps: totalBandwidth / count,
		PacketLoss:    totalPacketLoss / float64(count),
		RTTMs:         totalRTT / count,
		JitterMs:      totalJitter / count,
	}
}

// determineTargetQuality determines the target quality based on metrics
func (s *QualityService) determineTargetQuality(metrics NetworkMetrics) models.VideoQuality {
	// Check for critical conditions first
	if metrics.PacketLoss >= s.config.PacketLossCritical || metrics.RTTMs >= s.config.RTTCritical {
		if metrics.BandwidthKbps < s.config.BandwidthAudio {
			return models.VideoQualityAudioOnly
		}
		return models.VideoQuality360p
	}

	// Determine quality based on bandwidth
	if metrics.BandwidthKbps >= s.config.Bandwidth1080p {
		return models.VideoQuality1080p
	} else if metrics.BandwidthKbps >= s.config.Bandwidth720p {
		return models.VideoQuality720p
	} else if metrics.BandwidthKbps >= s.config.Bandwidth480p {
		return models.VideoQuality480p
	} else if metrics.BandwidthKbps >= s.config.Bandwidth360p {
		return models.VideoQuality360p
	}

	return models.VideoQualityAudioOnly
}

// determineDowngradeReason determines the reason for quality downgrade
func (s *QualityService) determineDowngradeReason(metrics NetworkMetrics) string {
	if metrics.PacketLoss >= s.config.PacketLossCritical {
		return "packet_loss_critical"
	}
	if metrics.PacketLoss >= s.config.PacketLossWarning {
		return "packet_loss_high"
	}
	if metrics.RTTMs >= s.config.RTTCritical {
		return "latency_critical"
	}
	if metrics.RTTMs >= s.config.RTTWarning {
		return "latency_high"
	}
	return "bandwidth_low"
}

// isQualityHigher returns true if q1 is higher quality than q2
func (s *QualityService) isQualityHigher(q1, q2 models.VideoQuality) bool {
	order := map[models.VideoQuality]int{
		models.VideoQualityAudioOnly: 0,
		models.VideoQuality360p:      1,
		models.VideoQuality480p:      2,
		models.VideoQuality720p:      3,
		models.VideoQuality1080p:     4,
	}
	return order[q1] > order[q2]
}

// GetParticipantQuality returns the current quality for a participant
func (s *QualityService) GetParticipantQuality(meetingID string, participantID string) (models.VideoQuality, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key := meetingID + ":" + participantID
	state, exists := s.participants[key]
	if !exists {
		return models.VideoQuality720p, &models.MeetingError{
			Code:    models.ErrMeetingNotFound,
			Message: "Participant not found",
		}
	}

	return state.CurrentQuality, nil
}

// ForceQuality forces a specific quality for a participant
func (s *QualityService) ForceQuality(meetingID string, participantID string, quality models.VideoQuality) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := meetingID + ":" + participantID
	state, exists := s.participants[key]
	if !exists {
		return &models.MeetingError{
			Code:    models.ErrMeetingNotFound,
			Message: "Participant not found",
		}
	}

	state.CurrentQuality = quality
	state.TargetQuality = quality
	state.StableCount = 0
	state.UnstableCount = 0

	s.logger.Info("Quality forced",
		zap.String("meetingId", meetingID),
		zap.String("participantId", participantID),
		zap.String("quality", string(quality)),
	)

	return nil
}

// GetQualityStats returns quality statistics for a participant
func (s *QualityService) GetQualityStats(meetingID string, participantID string) (*QualityStats, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key := meetingID + ":" + participantID
	state, exists := s.participants[key]
	if !exists {
		return nil, &models.MeetingError{
			Code:    models.ErrMeetingNotFound,
			Message: "Participant not found",
		}
	}

	avgMetrics := s.calculateAverageMetrics(state.Metrics)

	return &QualityStats{
		ParticipantID:  state.ParticipantID,
		MeetingID:      state.MeetingID,
		CurrentQuality: state.CurrentQuality,
		TargetQuality:  state.TargetQuality,
		AverageMetrics: avgMetrics,
		SampleCount:    len(state.Metrics),
		LastUpgrade:    state.LastUpgrade,
		LastDowngrade:  state.LastDowngrade,
	}, nil
}

// QualityStats represents quality statistics for a participant
type QualityStats struct {
	ParticipantID  uuid.UUID           `json:"participantId"`
	MeetingID      uuid.UUID           `json:"meetingId"`
	CurrentQuality models.VideoQuality `json:"currentQuality"`
	TargetQuality  models.VideoQuality `json:"targetQuality"`
	AverageMetrics NetworkMetrics      `json:"averageMetrics"`
	SampleCount    int                 `json:"sampleCount"`
	LastUpgrade    time.Time           `json:"lastUpgrade"`
	LastDowngrade  time.Time           `json:"lastDowngrade"`
}
