// Package services provides business logic services for the core service.
package services

import (
	"crypto/rand"
	"encoding/base32"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/cloud-devbox/services/core/internal/models"
	"github.com/google/uuid"
	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var (
	ErrMFAAlreadyEnabled  = errors.New("MFA is already enabled")
	ErrMFANotEnabled      = errors.New("MFA is not enabled")
	ErrInvalidRecoveryCode = errors.New("invalid recovery code")
	ErrNoRecoveryCodesLeft = errors.New("no recovery codes left")
)

const (
	// Number of recovery codes to generate
	RecoveryCodeCount = 10
	// Length of each recovery code segment
	RecoveryCodeSegmentLength = 4
	// Number of segments in a recovery code
	RecoveryCodeSegments = 2
	// TOTP issuer name
	TOTPIssuer = "CloudDevBox"
)

// MFAService handles multi-factor authentication operations
type MFAService struct {
	db *gorm.DB
}

// NewMFAService creates a new MFAService instance
func NewMFAService(db *gorm.DB) *MFAService {
	return &MFAService{db: db}
}

// MFASetupResponse represents the response for MFA setup
type MFASetupResponse struct {
	Secret        string   `json:"secret"`
	QRCodeURL     string   `json:"qr_code_url"`
	RecoveryCodes []string `json:"recovery_codes"`
}

// SetupMFA initiates MFA setup for a user
func (s *MFAService) SetupMFA(userID uuid.UUID) (*MFASetupResponse, error) {
	var user models.User
	if err := s.db.First(&user, "id = ?", userID).Error; err != nil {
		return nil, ErrUserNotFound
	}

	if user.MFAEnabled {
		return nil, ErrMFAAlreadyEnabled
	}

	// Generate TOTP key
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      TOTPIssuer,
		AccountName: user.Email,
		Period:      30,
		SecretSize:  32,
		Digits:      otp.DigitsSix,
		Algorithm:   otp.AlgorithmSHA1,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to generate TOTP key: %w", err)
	}

	// Store secret temporarily (will be confirmed when user verifies)
	user.MFASecret = key.Secret()
	if err := s.db.Save(&user).Error; err != nil {
		return nil, fmt.Errorf("failed to save MFA secret: %w", err)
	}

	// Generate recovery codes
	recoveryCodes, err := s.generateRecoveryCodes(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate recovery codes: %w", err)
	}

	return &MFASetupResponse{
		Secret:        key.Secret(),
		QRCodeURL:     key.URL(),
		RecoveryCodes: recoveryCodes,
	}, nil
}

// VerifyAndEnableMFA verifies the TOTP code and enables MFA
func (s *MFAService) VerifyAndEnableMFA(userID uuid.UUID, code string) error {
	var user models.User
	if err := s.db.First(&user, "id = ?", userID).Error; err != nil {
		return ErrUserNotFound
	}

	if user.MFAEnabled {
		return ErrMFAAlreadyEnabled
	}

	if user.MFASecret == "" {
		return errors.New("MFA setup not initiated")
	}

	// Verify TOTP code
	valid := totp.Validate(code, user.MFASecret)
	if !valid {
		return ErrInvalidMFACode
	}

	// Enable MFA
	user.MFAEnabled = true
	if err := s.db.Save(&user).Error; err != nil {
		return fmt.Errorf("failed to enable MFA: %w", err)
	}

	return nil
}

// DisableMFA disables MFA for a user
func (s *MFAService) DisableMFA(userID uuid.UUID, code string) error {
	var user models.User
	if err := s.db.First(&user, "id = ?", userID).Error; err != nil {
		return ErrUserNotFound
	}

	if !user.MFAEnabled {
		return ErrMFANotEnabled
	}

	// Verify TOTP code
	valid := totp.Validate(code, user.MFASecret)
	if !valid {
		return ErrInvalidMFACode
	}

	// Disable MFA and clear secret
	user.MFAEnabled = false
	user.MFASecret = ""
	if err := s.db.Save(&user).Error; err != nil {
		return fmt.Errorf("failed to disable MFA: %w", err)
	}

	// Delete recovery codes
	s.db.Where("user_id = ?", userID).Delete(&models.RecoveryCode{})

	return nil
}

// VerifyTOTP verifies a TOTP code for a user
func (s *MFAService) VerifyTOTP(userID uuid.UUID, code string) (bool, error) {
	var user models.User
	if err := s.db.First(&user, "id = ?", userID).Error; err != nil {
		return false, ErrUserNotFound
	}

	if !user.MFAEnabled {
		return false, ErrMFANotEnabled
	}

	return totp.Validate(code, user.MFASecret), nil
}


// VerifyRecoveryCode verifies and consumes a recovery code
func (s *MFAService) VerifyRecoveryCode(userID uuid.UUID, code string) (bool, error) {
	var user models.User
	if err := s.db.First(&user, "id = ?", userID).Error; err != nil {
		return false, ErrUserNotFound
	}

	if !user.MFAEnabled {
		return false, ErrMFANotEnabled
	}

	// Normalize code (remove dashes, uppercase)
	normalizedCode := strings.ToUpper(strings.ReplaceAll(code, "-", ""))

	// Find unused recovery codes
	var recoveryCodes []models.RecoveryCode
	if err := s.db.Where("user_id = ? AND used = false", userID).Find(&recoveryCodes).Error; err != nil {
		return false, err
	}

	// Check each code
	for _, rc := range recoveryCodes {
		if err := bcrypt.CompareHashAndPassword([]byte(rc.Code), []byte(normalizedCode)); err == nil {
			// Mark code as used
			now := time.Now()
			rc.Used = true
			rc.UsedAt = &now
			s.db.Save(&rc)
			return true, nil
		}
	}

	return false, ErrInvalidRecoveryCode
}

// generateRecoveryCodes generates new recovery codes for a user
func (s *MFAService) generateRecoveryCodes(userID uuid.UUID) ([]string, error) {
	// Delete existing recovery codes
	s.db.Where("user_id = ?", userID).Delete(&models.RecoveryCode{})

	codes := make([]string, RecoveryCodeCount)
	for i := 0; i < RecoveryCodeCount; i++ {
		code, err := generateRecoveryCode()
		if err != nil {
			return nil, err
		}
		codes[i] = code

		// Hash the code for storage
		normalizedCode := strings.ToUpper(strings.ReplaceAll(code, "-", ""))
		hashedCode, err := bcrypt.GenerateFromPassword([]byte(normalizedCode), bcrypt.DefaultCost)
		if err != nil {
			return nil, err
		}

		recoveryCode := &models.RecoveryCode{
			UserID: userID,
			Code:   string(hashedCode),
		}
		if err := s.db.Create(recoveryCode).Error; err != nil {
			return nil, err
		}
	}

	return codes, nil
}

// generateRecoveryCode generates a single recovery code
func generateRecoveryCode() (string, error) {
	segments := make([]string, RecoveryCodeSegments)
	for i := 0; i < RecoveryCodeSegments; i++ {
		bytes := make([]byte, RecoveryCodeSegmentLength)
		if _, err := rand.Read(bytes); err != nil {
			return "", err
		}
		// Use base32 encoding for readability (no 0, 1, O, I)
		encoded := base32.StdEncoding.EncodeToString(bytes)
		segments[i] = encoded[:RecoveryCodeSegmentLength]
	}
	return strings.Join(segments, "-"), nil
}

// RegenerateRecoveryCodes regenerates recovery codes for a user
func (s *MFAService) RegenerateRecoveryCodes(userID uuid.UUID, code string) ([]string, error) {
	var user models.User
	if err := s.db.First(&user, "id = ?", userID).Error; err != nil {
		return nil, ErrUserNotFound
	}

	if !user.MFAEnabled {
		return nil, ErrMFANotEnabled
	}

	// Verify TOTP code
	valid := totp.Validate(code, user.MFASecret)
	if !valid {
		return nil, ErrInvalidMFACode
	}

	return s.generateRecoveryCodes(userID)
}

// GetRecoveryCodeCount returns the number of unused recovery codes
func (s *MFAService) GetRecoveryCodeCount(userID uuid.UUID) (int64, error) {
	var count int64
	if err := s.db.Model(&models.RecoveryCode{}).
		Where("user_id = ? AND used = false", userID).
		Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// IsMFAEnabled checks if MFA is enabled for a user
func (s *MFAService) IsMFAEnabled(userID uuid.UUID) (bool, error) {
	var user models.User
	if err := s.db.First(&user, "id = ?", userID).Error; err != nil {
		return false, ErrUserNotFound
	}
	return user.MFAEnabled, nil
}
