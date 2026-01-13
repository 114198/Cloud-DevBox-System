// Package server provides the HTTP/WebSocket server for the realtime service.
package server

import (
	"context"

	"github.com/cloud-devbox/services/realtime/internal/config"
	"github.com/cloud-devbox/services/realtime/internal/handlers"
	"github.com/cloud-devbox/services/realtime/internal/services"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Server represents the HTTP/WebSocket server
type Server struct {
	router            *gin.Engine
	config            *config.Config
	logger            *zap.Logger
	hub               *services.Hub
	collabService     *services.CollaborationService
	collabHandler     *handlers.CollaborationHandler
	signalingService  *services.SignalingService
	meetingHandler    *handlers.MeetingHandler
}

// New creates a new server instance
func New(cfg *config.Config, logger *zap.Logger) (*Server, error) {
	router := gin.Default()

	// Create hub and services
	hub := services.NewHub(logger)
	collabService := services.NewCollaborationService(logger, hub)
	collabHandler := handlers.NewCollaborationHandler(collabService, hub, logger)
	
	// Create meeting services
	signalingService := services.NewSignalingService(logger)
	meetingHandler := handlers.NewMeetingHandler(signalingService, logger)

	s := &Server{
		router:           router,
		config:           cfg,
		logger:           logger,
		hub:              hub,
		collabService:    collabService,
		collabHandler:    collabHandler,
		signalingService: signalingService,
		meetingHandler:   meetingHandler,
	}

	s.setupRoutes()
	return s, nil
}

// Start starts the server and background services
func (s *Server) Start(ctx context.Context) error {
	// Start collaboration service
	s.collabService.Start(ctx)

	s.logger.Info("Server started", zap.String("port", s.config.Port))
	return nil
}

// Run starts the HTTP server
func (s *Server) Run(addr string) error {
	return s.router.Run(addr)
}

func (s *Server) setupRoutes() {
	// Health check
	s.router.GET("/health", handlers.HealthCheck)

	// WebSocket endpoints
	s.router.GET("/ws/collaboration/:environmentId", s.collabHandler.HandleWebSocket)
	s.router.GET("/ws/terminal/:environmentId", handlers.TerminalWebSocket)
	s.router.GET("/ws/meeting/:meetingId", s.meetingHandler.HandleSignalingWebSocket)

	// REST API endpoints for collaboration
	api := s.router.Group("/api/v1")
	{
		collab := api.Group("/collaboration")
		{
			collab.POST("/sessions", s.collabHandler.CreateSession)
			collab.GET("/sessions/:sessionId", s.collabHandler.GetSession)
			collab.DELETE("/sessions/:sessionId", s.collabHandler.EndSession)
			collab.GET("/sessions/:sessionId/participants", s.collabHandler.GetSessionParticipants)
			collab.GET("/sessions/:sessionId/history", s.collabHandler.GetSessionHistory)
			collab.GET("/sessions/:sessionId/stats", s.collabHandler.GetSessionStats)
		}

		// Meeting endpoints
		meetings := api.Group("/meetings")
		{
			meetings.POST("", s.meetingHandler.CreateMeeting)
			meetings.GET("", s.meetingHandler.ListMeetings)
			meetings.GET("/:meetingId", s.meetingHandler.GetMeeting)
			meetings.DELETE("/:meetingId", s.meetingHandler.EndMeeting)
			meetings.GET("/:meetingId/participants", s.meetingHandler.GetMeetingParticipants)
			meetings.PUT("/:meetingId/media-state", s.meetingHandler.UpdateMediaState)
			meetings.POST("/:meetingId/screen-share/start", s.meetingHandler.StartScreenShare)
			meetings.POST("/:meetingId/screen-share/stop", s.meetingHandler.StopScreenShare)
		}
	}
}
