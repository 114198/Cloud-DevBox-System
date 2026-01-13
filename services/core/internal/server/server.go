// Package server provides the HTTP server for the core service.
package server

import (
	"github.com/cloud-devbox/services/core/internal/config"
	"github.com/cloud-devbox/services/core/internal/database"
	"github.com/cloud-devbox/services/core/internal/handlers"
	"github.com/cloud-devbox/services/core/internal/services"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Server represents the HTTP server
type Server struct {
	router          *gin.Engine
	config          *config.Config
	db              *gorm.DB
	authHandler     *handlers.AuthHandler
	oauthHandler    *handlers.OAuthHandler
	mfaHandler      *handlers.MFAHandler
	rbacHandler     *handlers.RBACHandler
	templateHandler *handlers.TemplateHandler
}

// New creates a new server instance
func New(cfg *config.Config) (*Server, error) {
	router := gin.Default()

	// Connect to database
	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		return nil, err
	}

	// Initialize services
	authService, err := services.NewAuthService(db)
	if err != nil {
		return nil, err
	}

	mfaService := services.NewMFAService(db)

	rbacService, err := services.NewRBACService(db, cfg.RedisURL)
	if err != nil {
		return nil, err
	}

	// Initialize template service
	templateService, err := services.NewTemplateService(db, cfg.RedisURL)
	if err != nil {
		return nil, err
	}
	templateVersionService := services.NewTemplateVersionService(db, templateService)

	// Initialize OAuth service with provider configs
	oauthConfigs := map[services.OAuthProvider]*services.OAuthConfig{
		services.ProviderGitHub: {
			ClientID:     cfg.GitHubClientID,
			ClientSecret: cfg.GitHubClientSecret,
			RedirectURL:  cfg.BaseURL + "/api/v1/oauth/github/callback",
		},
		services.ProviderGitLab: {
			ClientID:     cfg.GitLabClientID,
			ClientSecret: cfg.GitLabClientSecret,
			RedirectURL:  cfg.BaseURL + "/api/v1/oauth/gitlab/callback",
		},
		services.ProviderGitee: {
			ClientID:     cfg.GiteeClientID,
			ClientSecret: cfg.GiteeClientSecret,
			RedirectURL:  cfg.BaseURL + "/api/v1/oauth/gitee/callback",
		},
	}
	oauthService := services.NewOAuthService(db, authService, oauthConfigs)

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(authService)
	oauthHandler := handlers.NewOAuthHandler(oauthService)
	mfaHandler := handlers.NewMFAHandler(mfaService, authService)
	rbacHandler := handlers.NewRBACHandler(rbacService)
	templateHandler := handlers.NewTemplateHandler(templateService, templateVersionService)

	s := &Server{
		router:          router,
		config:          cfg,
		db:              db,
		authHandler:     authHandler,
		oauthHandler:    oauthHandler,
		mfaHandler:      mfaHandler,
		rbacHandler:     rbacHandler,
		templateHandler: templateHandler,
	}

	s.setupRoutes()
	return s, nil
}

// Run starts the HTTP server
func (s *Server) Run(addr string) error {
	return s.router.Run(addr)
}

func (s *Server) setupRoutes() {
	// Health check
	s.router.GET("/health", handlers.HealthCheck)

	// API v1 routes
	v1 := s.router.Group("/api/v1")
	{
		// Auth routes (public)
		auth := v1.Group("/auth")
		{
			auth.POST("/register", s.authHandler.Register)
			auth.POST("/login", s.authHandler.Login)
			auth.POST("/refresh", s.authHandler.RefreshToken)
			auth.POST("/mfa/verify", s.mfaHandler.VerifyMFAForLogin)
		}

		// OAuth routes (public)
		oauth := v1.Group("/oauth")
		{
			oauth.GET("/:provider/authorize", s.oauthHandler.GetAuthURL)
			oauth.GET("/:provider/callback", s.oauthHandler.Callback)
		}

		// Protected routes
		protected := v1.Group("")
		protected.Use(s.authHandler.AuthMiddleware())
		{
			// Auth routes (protected)
			protected.POST("/auth/logout", s.authHandler.Logout)

			// MFA routes (protected)
			mfa := protected.Group("/mfa")
			{
				mfa.GET("/status", s.mfaHandler.GetMFAStatus)
				mfa.POST("/setup", s.mfaHandler.SetupMFA)
				mfa.POST("/enable", s.mfaHandler.VerifyAndEnableMFA)
				mfa.POST("/disable", s.mfaHandler.DisableMFA)
				mfa.POST("/recovery-codes/regenerate", s.mfaHandler.RegenerateRecoveryCodes)
			}

			// OAuth account management (protected)
			protectedOAuth := protected.Group("/oauth")
			{
				protectedOAuth.GET("/accounts", s.oauthHandler.GetLinkedAccounts)
				protectedOAuth.POST("/:provider/link", s.oauthHandler.LinkAccount)
				protectedOAuth.DELETE("/:provider/unlink", s.oauthHandler.UnlinkAccount)
			}

			// User routes
			users := protected.Group("/users")
			{
				users.GET("/me", s.authHandler.GetCurrentUser)
				users.PUT("/me", handlers.UpdateCurrentUser)
				users.GET("/me/permissions", s.rbacHandler.GetMyPermissions)
			}

			// Admin routes (admin only)
			admin := protected.Group("/admin")
			admin.Use(s.rbacHandler.RequireRole(services.RoleAdmin))
			{
				admin.GET("/roles", s.rbacHandler.GetRoles)
				admin.PUT("/users/:id/role", s.rbacHandler.UpdateUserRole)
			}

			// Environment routes
			envs := protected.Group("/environments")
			{
				envs.GET("", handlers.ListEnvironments)
				envs.POST("", handlers.CreateEnvironment)
				envs.GET("/:id", handlers.GetEnvironment)
				envs.PUT("/:id", handlers.UpdateEnvironment)
				envs.DELETE("/:id", handlers.DeleteEnvironment)
				envs.POST("/:id/start", handlers.StartEnvironment)
				envs.POST("/:id/stop", handlers.StopEnvironment)
			}

			// Template routes
			templates := protected.Group("/templates")
			{
				templates.GET("", s.templateHandler.ListTemplates)
				templates.GET("/categories", s.templateHandler.GetCategories)
				templates.GET("/tags", s.templateHandler.GetTags)
				templates.GET("/valid-categories", s.templateHandler.GetValidCategories)
				templates.POST("", s.templateHandler.CreateTemplate)
				templates.GET("/:id", s.templateHandler.GetTemplate)
				templates.PUT("/:id", s.templateHandler.UpdateTemplate)
				templates.DELETE("/:id", s.templateHandler.DeleteTemplate)
				templates.GET("/:id/versions", s.templateHandler.ListVersions)
				templates.POST("/:id/versions", s.templateHandler.CreateVersion)
				templates.GET("/:id/versions/:versionId", s.templateHandler.GetVersion)
				templates.POST("/:id/versions/:versionId/rollback", s.templateHandler.RollbackVersion)
			}

			// Project routes
			projects := protected.Group("/projects")
			{
				projects.GET("", handlers.ListProjects)
				projects.POST("", handlers.CreateProject)
				projects.GET("/:id", handlers.GetProject)
				projects.PUT("/:id", handlers.UpdateProject)
				projects.DELETE("/:id", handlers.DeleteProject)
			}
		}
	}
}

// Close closes the server and its resources
func (s *Server) Close() error {
	return database.Close(s.db)
}
