// Package services provides business logic services for the core service.
package services

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/cloud-devbox/services/core/internal/models"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserNotFound       = errors.New("user not found")
	ErrUserExists         = errors.New("user already exists")
	ErrInvalidToken       = errors.New("invalid token")
	ErrTokenExpired       = errors.New("token expired")
	ErrTokenRevoked       = errors.New("token revoked")
	ErrMFARequired        = errors.New("MFA verification required")
	ErrInvalidMFACode     = errors.New("invalid MFA code")
)

// JWTClaims represents the claims in a JWT token
type JWTClaims struct {
	UserID   string `json:"user_id"`
	Email    string `json:"email"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

// TokenPair represents an access token and refresh token pair
type TokenPair struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
	TokenType    string    `json:"token_type"`
}

// AuthService handles authentication operations
type AuthService struct {
	db             *gorm.DB
	privateKey     *rsa.PrivateKey
	publicKey      *rsa.PublicKey
	accessTokenTTL time.Duration
	refreshTokenTTL time.Duration
}

// DB returns the database connection
func (s *AuthService) DB() *gorm.DB {
	return s.db
}

// GenerateTokenPair generates a new access token and refresh token pair (public method)
func (s *AuthService) GenerateTokenPair(user *models.User) (*TokenPair, error) {
	return s.generateTokenPair(user)
}

// NewAuthService creates a new AuthService instance
func NewAuthService(db *gorm.DB) (*AuthService, error) {
	privateKey, publicKey, err := loadOrGenerateRSAKeys()
	if err != nil {
		return nil, fmt.Errorf("failed to load RSA keys: %w", err)
	}

	return &AuthService{
		db:              db,
		privateKey:      privateKey,
		publicKey:       publicKey,
		accessTokenTTL:  15 * time.Minute,
		refreshTokenTTL: 7 * 24 * time.Hour,
	}, nil
}

// loadOrGenerateRSAKeys loads RSA keys from environment or generates new ones
func loadOrGenerateRSAKeys() (*rsa.PrivateKey, *rsa.PublicKey, error) {
	privateKeyPEM := os.Getenv("JWT_PRIVATE_KEY")
	if privateKeyPEM != "" {
		block, _ := pem.Decode([]byte(privateKeyPEM))
		if block == nil {
			return nil, nil, errors.New("failed to decode private key PEM")
		}
		privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to parse private key: %w", err)
		}
		return privateKey, &privateKey.PublicKey, nil
	}

	// Generate new RSA key pair for development
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate RSA key: %w", err)
	}
	return privateKey, &privateKey.PublicKey, nil
}


// RegisterRequest represents a user registration request
type RegisterRequest struct {
	Email       string `json:"email" binding:"required,email"`
	Username    string `json:"username" binding:"required,min=3,max=50"`
	Password    string `json:"password" binding:"required,min=8"`
	DisplayName string `json:"display_name"`
}

// LoginRequest represents a user login request
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
	MFACode  string `json:"mfa_code,omitempty"`
}

// Register creates a new user account
func (s *AuthService) Register(req *RegisterRequest) (*models.User, *TokenPair, error) {
	// Check if user already exists
	var existingUser models.User
	if err := s.db.Where("email = ? OR username = ?", req.Email, req.Username).First(&existingUser).Error; err == nil {
		return nil, nil, ErrUserExists
	}

	// Hash password using bcrypt
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Create user
	user := &models.User{
		Email:        req.Email,
		Username:     req.Username,
		PasswordHash: string(hashedPassword),
		DisplayName:  req.DisplayName,
		Role:         "developer",
	}

	if user.DisplayName == "" {
		user.DisplayName = req.Username
	}

	if err := s.db.Create(user).Error; err != nil {
		return nil, nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Generate tokens
	tokens, err := s.generateTokenPair(user)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	return user, tokens, nil
}

// Login authenticates a user and returns tokens
func (s *AuthService) Login(req *LoginRequest) (*models.User, *TokenPair, error) {
	var user models.User
	if err := s.db.Where("email = ?", req.Email).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, ErrInvalidCredentials
		}
		return nil, nil, fmt.Errorf("failed to find user: %w", err)
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, nil, ErrInvalidCredentials
	}

	// Check if MFA is enabled
	if user.MFAEnabled {
		if req.MFACode == "" {
			return nil, nil, ErrMFARequired
		}
		// MFA verification will be implemented in task 5.3
	}

	// Update last login time
	now := time.Now()
	user.LastLoginAt = &now
	s.db.Model(&user).Update("last_login_at", now)

	// Generate tokens
	tokens, err := s.generateTokenPair(&user)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	return &user, tokens, nil
}

// RefreshTokenRequest represents a token refresh request
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// RefreshToken generates new tokens using a refresh token
func (s *AuthService) RefreshToken(refreshTokenStr string) (*TokenPair, error) {
	// Find the refresh token in database
	var refreshToken models.RefreshToken
	if err := s.db.Where("token = ?", refreshTokenStr).Preload("User").First(&refreshToken).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvalidToken
		}
		return nil, fmt.Errorf("failed to find refresh token: %w", err)
	}

	// Check if token is revoked
	if refreshToken.Revoked {
		return nil, ErrTokenRevoked
	}

	// Check if token is expired
	if time.Now().After(refreshToken.ExpiresAt) {
		return nil, ErrTokenExpired
	}

	// Revoke old refresh token
	s.db.Model(&refreshToken).Update("revoked", true)

	// Generate new token pair
	return s.generateTokenPair(&refreshToken.User)
}

// generateTokenPair generates a new access token and refresh token pair
func (s *AuthService) generateTokenPair(user *models.User) (*TokenPair, error) {
	now := time.Now()
	accessTokenExpiry := now.Add(s.accessTokenTTL)

	// Create access token claims
	claims := &JWTClaims{
		UserID:   user.ID.String(),
		Email:    user.Email,
		Username: user.Username,
		Role:     user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(accessTokenExpiry),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "cloud-devbox",
			Subject:   user.ID.String(),
		},
	}

	// Sign access token with RS256
	accessToken := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	accessTokenStr, err := accessToken.SignedString(s.privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to sign access token: %w", err)
	}

	// Generate refresh token
	refreshTokenBytes := make([]byte, 32)
	if _, err := rand.Read(refreshTokenBytes); err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}
	refreshTokenStr := base64.URLEncoding.EncodeToString(refreshTokenBytes)

	// Store refresh token in database
	refreshToken := &models.RefreshToken{
		UserID:    user.ID,
		Token:     refreshTokenStr,
		ExpiresAt: now.Add(s.refreshTokenTTL),
	}
	if err := s.db.Create(refreshToken).Error; err != nil {
		return nil, fmt.Errorf("failed to store refresh token: %w", err)
	}

	return &TokenPair{
		AccessToken:  accessTokenStr,
		RefreshToken: refreshTokenStr,
		ExpiresAt:    accessTokenExpiry,
		TokenType:    "Bearer",
	}, nil
}

// ValidateAccessToken validates an access token and returns the claims
func (s *AuthService) ValidateAccessToken(tokenStr string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.publicKey, nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*JWTClaims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

// GetUserByID retrieves a user by ID
func (s *AuthService) GetUserByID(userID uuid.UUID) (*models.User, error) {
	var user models.User
	if err := s.db.First(&user, "id = ?", userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to find user: %w", err)
	}
	return &user, nil
}

// Logout revokes all refresh tokens for a user
func (s *AuthService) Logout(userID uuid.UUID) error {
	return s.db.Model(&models.RefreshToken{}).
		Where("user_id = ? AND revoked = false", userID).
		Update("revoked", true).Error
}

// GetPublicKey returns the public key for token verification
func (s *AuthService) GetPublicKey() *rsa.PublicKey {
	return s.publicKey
}
