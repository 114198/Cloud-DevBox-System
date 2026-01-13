// Package handlers provides HTTP request handlers for the core service.
package handlers

import (
	"net/http"
	"strings"

	"github.com/cloud-devbox/services/core/internal/models"
	"github.com/cloud-devbox/services/core/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// AuthHandler handles authentication-related HTTP requests
type AuthHandler struct {
	authService *services.AuthService
}

// NewAuthHandler creates a new AuthHandler instance
func NewAuthHandler(authService *services.AuthService) *AuthHandler {
	return &AuthHandler{
		authService: authService,
	}
}

// RegisterResponse represents the response for user registration
type RegisterResponse struct {
	User   *UserResponse `json:"user"`
	Tokens *TokenResponse `json:"tokens"`
}

// LoginResponse represents the response for user login
type LoginResponse struct {
	User   *UserResponse  `json:"user"`
	Tokens *TokenResponse `json:"tokens"`
}

// TokenResponse represents the token response
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresAt    string `json:"expires_at"`
	TokenType    string `json:"token_type"`
}

// UserResponse represents the user response (without sensitive data)
type UserResponse struct {
	ID            string                 `json:"id"`
	Email         string                 `json:"email"`
	Username      string                 `json:"username"`
	DisplayName   string                 `json:"display_name"`
	Avatar        string                 `json:"avatar,omitempty"`
	Role          string                 `json:"role"`
	EmailVerified bool                   `json:"email_verified"`
	MFAEnabled    bool                   `json:"mfa_enabled"`
	Preferences   map[string]interface{} `json:"preferences"`
	CreatedAt     string                 `json:"created_at"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Code    string `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}

// Register handles user registration
func (h *AuthHandler) Register(c *gin.Context) {
	var req services.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_request",
			Message: err.Error(),
		})
		return
	}

	user, tokens, err := h.authService.Register(&req)
	if err != nil {
		switch err {
		case services.ErrUserExists:
			c.JSON(http.StatusConflict, ErrorResponse{
				Error:   "user_exists",
				Message: "A user with this email or username already exists",
			})
		default:
			c.JSON(http.StatusInternalServerError, ErrorResponse{
				Error:   "registration_failed",
				Message: err.Error(),
			})
		}
		return
	}

	c.JSON(http.StatusCreated, RegisterResponse{
		User: toUserResponse(user),
		Tokens: &TokenResponse{
			AccessToken:  tokens.AccessToken,
			RefreshToken: tokens.RefreshToken,
			ExpiresAt:    tokens.ExpiresAt.Format("2006-01-02T15:04:05Z07:00"),
			TokenType:    tokens.TokenType,
		},
	})
}


// Login handles user login
func (h *AuthHandler) Login(c *gin.Context) {
	var req services.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_request",
			Message: err.Error(),
		})
		return
	}

	user, tokens, err := h.authService.Login(&req)
	if err != nil {
		switch err {
		case services.ErrInvalidCredentials:
			c.JSON(http.StatusUnauthorized, ErrorResponse{
				Error:   "invalid_credentials",
				Message: "Invalid email or password",
			})
		case services.ErrMFARequired:
			c.JSON(http.StatusForbidden, ErrorResponse{
				Error:   "mfa_required",
				Code:    "MFA_REQUIRED",
				Message: "Multi-factor authentication is required",
			})
		default:
			c.JSON(http.StatusInternalServerError, ErrorResponse{
				Error:   "login_failed",
				Message: err.Error(),
			})
		}
		return
	}

	c.JSON(http.StatusOK, LoginResponse{
		User: toUserResponse(user),
		Tokens: &TokenResponse{
			AccessToken:  tokens.AccessToken,
			RefreshToken: tokens.RefreshToken,
			ExpiresAt:    tokens.ExpiresAt.Format("2006-01-02T15:04:05Z07:00"),
			TokenType:    tokens.TokenType,
		},
	})
}

// RefreshToken handles token refresh
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req services.RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_request",
			Message: err.Error(),
		})
		return
	}

	tokens, err := h.authService.RefreshToken(req.RefreshToken)
	if err != nil {
		switch err {
		case services.ErrInvalidToken:
			c.JSON(http.StatusUnauthorized, ErrorResponse{
				Error:   "invalid_token",
				Message: "Invalid refresh token",
			})
		case services.ErrTokenExpired:
			c.JSON(http.StatusUnauthorized, ErrorResponse{
				Error:   "token_expired",
				Message: "Refresh token has expired",
			})
		case services.ErrTokenRevoked:
			c.JSON(http.StatusUnauthorized, ErrorResponse{
				Error:   "token_revoked",
				Message: "Refresh token has been revoked",
			})
		default:
			c.JSON(http.StatusInternalServerError, ErrorResponse{
				Error:   "refresh_failed",
				Message: err.Error(),
			})
		}
		return
	}

	c.JSON(http.StatusOK, TokenResponse{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		ExpiresAt:    tokens.ExpiresAt.Format("2006-01-02T15:04:05Z07:00"),
		TokenType:    tokens.TokenType,
	})
}

// Logout handles user logout
func (h *AuthHandler) Logout(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "User not authenticated",
		})
		return
	}

	uid, err := uuid.Parse(userID.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_user_id",
			Message: "Invalid user ID",
		})
		return
	}

	if err := h.authService.Logout(uid); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "logout_failed",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Successfully logged out"})
}

// GetCurrentUser returns the current authenticated user
func (h *AuthHandler) GetCurrentUser(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "User not authenticated",
		})
		return
	}

	uid, err := uuid.Parse(userID.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_user_id",
			Message: "Invalid user ID",
		})
		return
	}

	user, err := h.authService.GetUserByID(uid)
	if err != nil {
		if err == services.ErrUserNotFound {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "user_not_found",
				Message: "User not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "fetch_failed",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, toUserResponse(user))
}

// toUserResponse converts a User model to UserResponse
func toUserResponse(user *models.User) *UserResponse {
	return &UserResponse{
		ID:            user.ID.String(),
		Email:         user.Email,
		Username:      user.Username,
		DisplayName:   user.DisplayName,
		Avatar:        user.Avatar,
		Role:          user.Role,
		EmailVerified: user.EmailVerified,
		MFAEnabled:    user.MFAEnabled,
		Preferences:   user.Preferences,
		CreatedAt:     user.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

// AuthMiddleware creates a middleware for JWT authentication
func (h *AuthHandler) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, ErrorResponse{
				Error:   "missing_token",
				Message: "Authorization header is required",
			})
			c.Abort()
			return
		}

		// Extract token from "Bearer <token>"
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			c.JSON(http.StatusUnauthorized, ErrorResponse{
				Error:   "invalid_token_format",
				Message: "Authorization header must be in format: Bearer <token>",
			})
			c.Abort()
			return
		}

		tokenStr := parts[1]
		claims, err := h.authService.ValidateAccessToken(tokenStr)
		if err != nil {
			switch err {
			case services.ErrTokenExpired:
				c.JSON(http.StatusUnauthorized, ErrorResponse{
					Error:   "token_expired",
					Message: "Access token has expired",
				})
			default:
				c.JSON(http.StatusUnauthorized, ErrorResponse{
					Error:   "invalid_token",
					Message: "Invalid access token",
				})
			}
			c.Abort()
			return
		}

		// Set user info in context
		c.Set("user_id", claims.UserID)
		c.Set("email", claims.Email)
		c.Set("username", claims.Username)
		c.Set("role", claims.Role)

		c.Next()
	}
}
