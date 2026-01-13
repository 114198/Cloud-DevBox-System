// Package handlers provides HTTP request handlers for the core service.
package handlers

import (
	"net/http"

	"github.com/cloud-devbox/services/core/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// MFAHandler handles MFA-related HTTP requests
type MFAHandler struct {
	mfaService  *services.MFAService
	authService *services.AuthService
}

// NewMFAHandler creates a new MFAHandler instance
func NewMFAHandler(mfaService *services.MFAService, authService *services.AuthService) *MFAHandler {
	return &MFAHandler{
		mfaService:  mfaService,
		authService: authService,
	}
}

// MFASetupResponse represents the MFA setup response
type MFASetupResponseDTO struct {
	Secret        string   `json:"secret"`
	QRCodeURL     string   `json:"qr_code_url"`
	RecoveryCodes []string `json:"recovery_codes"`
}

// MFAVerifyRequest represents the MFA verification request
type MFAVerifyRequest struct {
	Code string `json:"code" binding:"required"`
}

// MFAStatusResponse represents the MFA status response
type MFAStatusResponse struct {
	Enabled            bool  `json:"enabled"`
	RecoveryCodesLeft  int64 `json:"recovery_codes_left"`
}

// SetupMFA initiates MFA setup for the current user
func (h *MFAHandler) SetupMFA(c *gin.Context) {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "User not authenticated",
		})
		return
	}

	response, err := h.mfaService.SetupMFA(userID)
	if err != nil {
		switch err {
		case services.ErrMFAAlreadyEnabled:
			c.JSON(http.StatusConflict, ErrorResponse{
				Error:   "mfa_already_enabled",
				Message: "MFA is already enabled for this account",
			})
		case services.ErrUserNotFound:
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "user_not_found",
				Message: "User not found",
			})
		default:
			c.JSON(http.StatusInternalServerError, ErrorResponse{
				Error:   "setup_failed",
				Message: err.Error(),
			})
		}
		return
	}

	c.JSON(http.StatusOK, MFASetupResponseDTO{
		Secret:        response.Secret,
		QRCodeURL:     response.QRCodeURL,
		RecoveryCodes: response.RecoveryCodes,
	})
}

// VerifyAndEnableMFA verifies the TOTP code and enables MFA
func (h *MFAHandler) VerifyAndEnableMFA(c *gin.Context) {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "User not authenticated",
		})
		return
	}

	var req MFAVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_request",
			Message: err.Error(),
		})
		return
	}

	if err := h.mfaService.VerifyAndEnableMFA(userID, req.Code); err != nil {
		switch err {
		case services.ErrMFAAlreadyEnabled:
			c.JSON(http.StatusConflict, ErrorResponse{
				Error:   "mfa_already_enabled",
				Message: "MFA is already enabled",
			})
		case services.ErrInvalidMFACode:
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:   "invalid_code",
				Message: "Invalid verification code",
			})
		default:
			c.JSON(http.StatusInternalServerError, ErrorResponse{
				Error:   "verification_failed",
				Message: err.Error(),
			})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "MFA enabled successfully"})
}

// DisableMFA disables MFA for the current user
func (h *MFAHandler) DisableMFA(c *gin.Context) {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "User not authenticated",
		})
		return
	}

	var req MFAVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_request",
			Message: err.Error(),
		})
		return
	}

	if err := h.mfaService.DisableMFA(userID, req.Code); err != nil {
		switch err {
		case services.ErrMFANotEnabled:
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:   "mfa_not_enabled",
				Message: "MFA is not enabled",
			})
		case services.ErrInvalidMFACode:
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:   "invalid_code",
				Message: "Invalid verification code",
			})
		default:
			c.JSON(http.StatusInternalServerError, ErrorResponse{
				Error:   "disable_failed",
				Message: err.Error(),
			})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "MFA disabled successfully"})
}

// GetMFAStatus returns the MFA status for the current user
func (h *MFAHandler) GetMFAStatus(c *gin.Context) {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "User not authenticated",
		})
		return
	}

	enabled, err := h.mfaService.IsMFAEnabled(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "fetch_failed",
			Message: err.Error(),
		})
		return
	}

	var recoveryCodesLeft int64
	if enabled {
		recoveryCodesLeft, _ = h.mfaService.GetRecoveryCodeCount(userID)
	}

	c.JSON(http.StatusOK, MFAStatusResponse{
		Enabled:           enabled,
		RecoveryCodesLeft: recoveryCodesLeft,
	})
}

// RegenerateRecoveryCodes regenerates recovery codes
func (h *MFAHandler) RegenerateRecoveryCodes(c *gin.Context) {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "User not authenticated",
		})
		return
	}

	var req MFAVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_request",
			Message: err.Error(),
		})
		return
	}

	codes, err := h.mfaService.RegenerateRecoveryCodes(userID, req.Code)
	if err != nil {
		switch err {
		case services.ErrMFANotEnabled:
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:   "mfa_not_enabled",
				Message: "MFA is not enabled",
			})
		case services.ErrInvalidMFACode:
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:   "invalid_code",
				Message: "Invalid verification code",
			})
		default:
			c.JSON(http.StatusInternalServerError, ErrorResponse{
				Error:   "regenerate_failed",
				Message: err.Error(),
			})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"recovery_codes": codes})
}

// VerifyMFAForLogin verifies MFA code during login
func (h *MFAHandler) VerifyMFAForLogin(c *gin.Context) {
	var req struct {
		Email        string `json:"email" binding:"required,email"`
		Code         string `json:"code" binding:"required"`
		IsRecovery   bool   `json:"is_recovery"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_request",
			Message: err.Error(),
		})
		return
	}

	// Find user by email
	var user models.User
	if err := h.authService.DB().Where("email = ?", req.Email).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "invalid_credentials",
			Message: "Invalid email or code",
		})
		return
	}

	var valid bool
	var err error

	if req.IsRecovery {
		valid, err = h.mfaService.VerifyRecoveryCode(user.ID, req.Code)
	} else {
		valid, err = h.mfaService.VerifyTOTP(user.ID, req.Code)
	}

	if err != nil || !valid {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "invalid_code",
			Message: "Invalid verification code",
		})
		return
	}

	// Generate tokens
	tokens, err := h.authService.GenerateTokenPair(&user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "token_generation_failed",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, LoginResponse{
		User: toUserResponse(&user),
		Tokens: &TokenResponse{
			AccessToken:  tokens.AccessToken,
			RefreshToken: tokens.RefreshToken,
			ExpiresAt:    tokens.ExpiresAt.Format("2006-01-02T15:04:05Z07:00"),
			TokenType:    tokens.TokenType,
		},
	})
}

// getUserIDFromContext extracts user ID from gin context
func getUserIDFromContext(c *gin.Context) (uuid.UUID, error) {
	userIDStr, exists := c.Get("user_id")
	if !exists {
		return uuid.Nil, services.ErrUserNotFound
	}
	return uuid.Parse(userIDStr.(string))
}
