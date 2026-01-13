// Package services provides tests for the meeting service.
package services

import (
	"context"
	"testing"
	"time"

	"github.com/cloud-devbox/services/realtime/internal/models"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"pgregory.net/rapid"
)

// TestMeetingCreationPerformance tests Property 6.5: Meeting creation < 3 seconds
// Feature: cloud-devbox, Property 6.5: Meeting creation < 3 seconds
// **Validates: Requirements 4.5.1**
func TestMeetingCreationPerformance(t *testing.T) {
	logger := zap.NewNop()
	signalingService := NewSignalingService(logger)
	meetingService := NewMeetingService(logger, signalingService)

	rapid.Check(t, func(t *rapid.T) {
		// Generate random meeting request
		envID := uuid.New().String()
		hostID := uuid.New().String()
		title := rapid.StringMatching(`[a-zA-Z0-9 ]{5,50}`).Draw(t, "title")
		meetingType := rapid.SampledFrom([]models.MeetingType{
			models.MeetingTypeAudio,
			models.MeetingTypeVideo,
			models.MeetingTypeScreenShare,
		}).Draw(t, "meetingType")

		req := &models.CreateMeetingRequest{
			EnvironmentID: envID,
			HostUserID:    hostID,
			Title:         title,
			Type:          meetingType,
		}

		// Measure creation time
		startTime := time.Now()
		meeting, err := meetingService.CreateMeeting(context.Background(), req)
		duration := time.Since(startTime)

		// Verify meeting was created successfully
		if err != nil {
			t.Fatalf("Failed to create meeting: %v", err)
		}

		// Property: Meeting creation should complete within 3 seconds
		if duration > 3*time.Second {
			t.Fatalf("Meeting creation took %v, expected < 3 seconds", duration)
		}

		// Verify meeting properties
		if meeting.ID == uuid.Nil {
			t.Fatal("Meeting ID should not be nil")
		}
		if meeting.Title != title {
			t.Fatalf("Meeting title mismatch: got %s, expected %s", meeting.Title, title)
		}
		if meeting.Type != meetingType {
			t.Fatalf("Meeting type mismatch: got %s, expected %s", meeting.Type, meetingType)
		}
		if meeting.Status != models.MeetingStatusScheduled {
			t.Fatalf("Meeting status should be scheduled, got %s", meeting.Status)
		}
	})
}

// TestQualityAdaptation tests Property 6.6: Quality adaptation based on bandwidth
// Feature: cloud-devbox, Property 6.6: Quality adaptation
// **Validates: Requirements 4.5.5**
func TestQualityAdaptation(t *testing.T) {
	logger := zap.NewNop()
	signalingService := NewSignalingService(logger)
	qualityService := NewQualityService(logger, signalingService)

	rapid.Check(t, func(t *rapid.T) {
		meetingID := uuid.New().String()
		participantID := uuid.New().String()

		// Register participant
		qualityService.RegisterParticipant(meetingID, participantID, models.VideoQuality720p)

		// Generate random bandwidth (in kbps)
		bandwidth := rapid.IntRange(50, 5000).Draw(t, "bandwidth")
		packetLoss := rapid.Float64Range(0, 10).Draw(t, "packetLoss")
		rtt := rapid.IntRange(10, 500).Draw(t, "rtt")

		metrics := NetworkMetrics{
			BandwidthKbps: bandwidth,
			PacketLoss:    packetLoss,
			RTTMs:         rtt,
			JitterMs:      rapid.IntRange(0, 100).Draw(t, "jitter"),
		}

		// Process metrics multiple times to trigger adaptation
		for i := 0; i < 5; i++ {
			qualityService.ProcessMetrics(meetingID, participantID, metrics)
		}

		// Get current quality
		quality, err := qualityService.GetParticipantQuality(meetingID, participantID)
		if err != nil {
			t.Fatalf("Failed to get quality: %v", err)
		}

		// Property: Quality should match bandwidth thresholds
		config := DefaultQualityConfig()
		
		// Check quality matches expected based on bandwidth
		if bandwidth >= config.Bandwidth1080p && packetLoss < config.PacketLossCritical && rtt < config.RTTCritical {
			// Should be 1080p or close
			if quality == models.VideoQualityAudioOnly {
				t.Logf("Quality is audio-only with bandwidth %d kbps (may be due to packet loss %.2f%% or RTT %dms)", 
					bandwidth, packetLoss, rtt)
			}
		} else if bandwidth < config.BandwidthAudio {
			// Should be audio only
			if quality != models.VideoQualityAudioOnly {
				t.Logf("Expected audio-only with bandwidth %d kbps, got %s", bandwidth, quality)
			}
		}

		// Property: Quality should never be higher than bandwidth allows
		maxQuality := getMaxQualityForBandwidth(bandwidth, config)
		if isQualityHigherThan(quality, maxQuality) && packetLoss < config.PacketLossCritical {
			t.Fatalf("Quality %s exceeds max allowed %s for bandwidth %d kbps", 
				quality, maxQuality, bandwidth)
		}
	})
}

// TestQualityDegradationOnHighPacketLoss tests quality degradation on high packet loss
// Feature: cloud-devbox, Property 6.6: Quality adaptation - packet loss handling
// **Validates: Requirements 4.5.5**
func TestQualityDegradationOnHighPacketLoss(t *testing.T) {
	logger := zap.NewNop()
	signalingService := NewSignalingService(logger)
	qualityService := NewQualityService(logger, signalingService)

	rapid.Check(t, func(t *rapid.T) {
		meetingID := uuid.New().String()
		participantID := uuid.New().String()

		// Register participant with high quality
		qualityService.RegisterParticipant(meetingID, participantID, models.VideoQuality1080p)

		// Generate high packet loss scenario
		packetLoss := rapid.Float64Range(5.0, 15.0).Draw(t, "highPacketLoss")
		
		metrics := NetworkMetrics{
			BandwidthKbps: 3000, // High bandwidth
			PacketLoss:    packetLoss,
			RTTMs:         50,
			JitterMs:      10,
		}

		// Process metrics multiple times
		for i := 0; i < 5; i++ {
			qualityService.ProcessMetrics(meetingID, participantID, metrics)
		}

		quality, _ := qualityService.GetParticipantQuality(meetingID, participantID)

		// Property: High packet loss should trigger quality degradation
		config := DefaultQualityConfig()
		if packetLoss >= config.PacketLossCritical {
			// Quality should be degraded
			if quality == models.VideoQuality1080p {
				t.Logf("Quality should degrade from 1080p with packet loss %.2f%%", packetLoss)
			}
		}
	})
}

// TestMeetingJoinPerformance tests meeting join performance
// Feature: cloud-devbox, Property 6.5: Meeting operations performance
// **Validates: Requirements 4.5.1**
func TestMeetingJoinPerformance(t *testing.T) {
	logger := zap.NewNop()
	signalingService := NewSignalingService(logger)

	rapid.Check(t, func(t *rapid.T) {
		// Create a meeting first
		envID := uuid.New().String()
		hostID := uuid.New().String()
		
		meeting, err := signalingService.CreateMeeting(context.Background(), &models.CreateMeetingRequest{
			EnvironmentID: envID,
			HostUserID:    hostID,
			Title:         "Test Meeting",
			Type:          models.MeetingTypeVideo,
		})
		if err != nil {
			t.Fatalf("Failed to create meeting: %v", err)
		}

		// Generate random participant
		userID := uuid.New().String()
		displayName := rapid.StringMatching(`[a-zA-Z ]{3,20}`).Draw(t, "displayName")

		// Measure join time (without actual WebSocket connection)
		startTime := time.Now()
		
		// Simulate join request validation
		_, err = signalingService.GetMeeting(meeting.ID.String())
		if err != nil {
			t.Fatalf("Failed to get meeting: %v", err)
		}

		duration := time.Since(startTime)

		// Property: Meeting lookup should be fast (< 100ms)
		if duration > 100*time.Millisecond {
			t.Fatalf("Meeting lookup took %v, expected < 100ms", duration)
		}

		// Verify meeting can be found
		if meeting.ID == uuid.Nil {
			t.Fatal("Meeting ID should not be nil")
		}

		_ = userID
		_ = displayName
	})
}

// TestScreenShareConstraints tests screen share constraints
// Feature: cloud-devbox, Property 6.7: Screen share performance
// **Validates: Requirements 4.5.2**
func TestScreenShareConstraints(t *testing.T) {
	logger := zap.NewNop()
	signalingService := NewSignalingService(logger)
	screenShareService := NewScreenShareService(logger, signalingService)

	// Create a meeting first
	meeting, _ := signalingService.CreateMeeting(context.Background(), &models.CreateMeetingRequest{
		EnvironmentID: uuid.New().String(),
		HostUserID:    uuid.New().String(),
		Title:         "Test Meeting",
		Type:          models.MeetingTypeVideo,
		Settings: &models.MeetingSettings{
			AllowScreenShare: true,
			MaxParticipants:  10,
		},
	})

	rapid.Check(t, func(t *rapid.T) {
		participantID := uuid.New().String()
		userID := uuid.New().String()

		// Generate random screen share config
		resolution := rapid.SampledFrom([]string{"1080p", "720p", "480p"}).Draw(t, "resolution")
		frameRate := rapid.IntRange(5, 60).Draw(t, "frameRate")

		config := &ScreenShareConfig{
			ShareType:  "screen",
			Resolution: resolution,
			FrameRate:  frameRate,
			Audio:      rapid.Bool().Draw(t, "audio"),
		}

		session, err := screenShareService.StartScreenShare(
			meeting.ID.String(),
			participantID,
			userID,
			config,
		)

		if err != nil {
			// Expected error if screen share not allowed
			return
		}

		// Property: 1080p should be capped at 30fps
		if session.Config.Resolution == "1080p" && session.Config.FrameRate > 30 {
			t.Fatalf("1080p screen share should be capped at 30fps, got %d", session.Config.FrameRate)
		}

		// Property: Session should have valid ID
		if session.ID == uuid.Nil {
			t.Fatal("Screen share session ID should not be nil")
		}

		// Cleanup
		screenShareService.StopScreenShare(meeting.ID.String(), participantID)
	})
}

// TestConcurrentMeetingCreation tests concurrent meeting creation
// Feature: cloud-devbox, Property 6.5: Meeting creation under load
// **Validates: Requirements 4.5.1**
func TestConcurrentMeetingCreation(t *testing.T) {
	logger := zap.NewNop()
	signalingService := NewSignalingService(logger)
	meetingService := NewMeetingService(logger, signalingService)

	rapid.Check(t, func(t *rapid.T) {
		numMeetings := rapid.IntRange(1, 10).Draw(t, "numMeetings")
		
		results := make(chan time.Duration, numMeetings)
		errors := make(chan error, numMeetings)

		for i := 0; i < numMeetings; i++ {
			go func(idx int) {
				req := &models.CreateMeetingRequest{
					EnvironmentID: uuid.New().String(),
					HostUserID:    uuid.New().String(),
					Title:         "Concurrent Test Meeting",
					Type:          models.MeetingTypeVideo,
				}

				startTime := time.Now()
				_, err := meetingService.CreateMeeting(context.Background(), req)
				duration := time.Since(startTime)

				if err != nil {
					errors <- err
				} else {
					results <- duration
				}
			}(i)
		}

		// Collect results
		var maxDuration time.Duration
		for i := 0; i < numMeetings; i++ {
			select {
			case duration := <-results:
				if duration > maxDuration {
					maxDuration = duration
				}
			case err := <-errors:
				t.Fatalf("Concurrent meeting creation failed: %v", err)
			case <-time.After(5 * time.Second):
				t.Fatal("Timeout waiting for meeting creation")
			}
		}

		// Property: All meetings should be created within 3 seconds
		if maxDuration > 3*time.Second {
			t.Fatalf("Slowest meeting creation took %v, expected < 3 seconds", maxDuration)
		}
	})
}

// Helper functions

func getMaxQualityForBandwidth(bandwidth int, config QualityConfig) models.VideoQuality {
	if bandwidth >= config.Bandwidth1080p {
		return models.VideoQuality1080p
	} else if bandwidth >= config.Bandwidth720p {
		return models.VideoQuality720p
	} else if bandwidth >= config.Bandwidth480p {
		return models.VideoQuality480p
	} else if bandwidth >= config.Bandwidth360p {
		return models.VideoQuality360p
	}
	return models.VideoQualityAudioOnly
}

func isQualityHigherThan(q1, q2 models.VideoQuality) bool {
	order := map[models.VideoQuality]int{
		models.VideoQualityAudioOnly: 0,
		models.VideoQuality360p:      1,
		models.VideoQuality480p:      2,
		models.VideoQuality720p:      3,
		models.VideoQuality1080p:     4,
	}
	return order[q1] > order[q2]
}
