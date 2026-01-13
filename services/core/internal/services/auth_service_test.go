// Package services provides business logic services for the core service.
package services

import (
	"testing"
	"time"

	"github.com/cloud-devbox/services/core/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupTestDB creates an in-memory SQLite database for testing
func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// Auto-migrate models
	err = db.AutoMigrate(
		&models.User{},
		&models.OAuthAccount{},
		&models.RefreshToken{},
		&models.RecoveryCode{},
	)
	require.NoError(t, err)

	return db
}

// createTestUser creates a test user in the database
func createTestUser(t *testing.T, db *gorm.DB, email, username, password string) *models.User {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	require.NoError(t, err)

	user := &models.User{
		ID:           uuid.New(),
		Email:        email,
		Username:     username,
		PasswordHash: string(hashedPassword),
		DisplayName:  username,
		Role:         "developer",
	}

	err = db.Create(user).Error
	require.NoError(t, err)

	return user
}

func TestAuthService_Register(t *testing.T) {
	db := setupTestDB(t)
	authService, err := NewAuthService(db)
	require.NoError(t, err)

	t.Run("successful registration", func(t *testing.T) {
		req := &RegisterRequest{
			Email:       "test@example.com",
			Username:    "testuser",
			Password:    "password123",
			DisplayName: "Test User",
		}

		user, tokens, err := authService.Register(req)

		assert.NoError(t, err)
		assert.NotNil(t, user)
		assert.NotNil(t, tokens)
		assert.Equal(t, req.Email, user.Email)
		assert.Equal(t, req.Username, user.Username)
		assert.Equal(t, req.DisplayName, user.DisplayName)
		assert.Equal(t, "developer", user.Role)
		assert.NotEmpty(t, tokens.AccessToken)
		assert.NotEmpty(t, tokens.RefreshToken)
		assert.Equal(t, "Bearer", tokens.TokenType)
	})

	t.Run("duplicate email", func(t *testing.T) {
		req := &RegisterRequest{
			Email:    "test@example.com",
			Username: "anotheruser",
			Password: "password123",
		}

		user, tokens, err := authService.Register(req)

		assert.Error(t, err)
		assert.Equal(t, ErrUserExists, err)
		assert.Nil(t, user)
		assert.Nil(t, tokens)
	})

	t.Run("duplicate username", func(t *testing.T) {
		req := &RegisterRequest{
			Email:    "another@example.com",
			Username: "testuser",
			Password: "password123",
		}

		user, tokens, err := authService.Register(req)

		assert.Error(t, err)
		assert.Equal(t, ErrUserExists, err)
		assert.Nil(t, user)
		assert.Nil(t, tokens)
	})
}

func TestAuthService_Login(t *testing.T) {
	db := setupTestDB(t)
	authService, err := NewAuthService(db)
	require.NoError(t, err)

	// Create a test user
	createTestUser(t, db, "login@example.com", "loginuser", "correctpassword")

	t.Run("successful login", func(t *testing.T) {
		req := &LoginRequest{
			Email:    "login@example.com",
			Password: "correctpassword",
		}

		user, tokens, err := authService.Login(req)

		assert.NoError(t, err)
		assert.NotNil(t, user)
		assert.NotNil(t, tokens)
		assert.Equal(t, "login@example.com", user.Email)
		assert.NotEmpty(t, tokens.AccessToken)
		assert.NotEmpty(t, tokens.RefreshToken)
	})

	t.Run("wrong password", func(t *testing.T) {
		req := &LoginRequest{
			Email:    "login@example.com",
			Password: "wrongpassword",
		}

		user, tokens, err := authService.Login(req)

		assert.Error(t, err)
		assert.Equal(t, ErrInvalidCredentials, err)
		assert.Nil(t, user)
		assert.Nil(t, tokens)
	})

	t.Run("non-existent user", func(t *testing.T) {
		req := &LoginRequest{
			Email:    "nonexistent@example.com",
			Password: "password123",
		}

		user, tokens, err := authService.Login(req)

		assert.Error(t, err)
		assert.Equal(t, ErrInvalidCredentials, err)
		assert.Nil(t, user)
		assert.Nil(t, tokens)
	})
}

func TestAuthService_ValidateAccessToken(t *testing.T) {
	db := setupTestDB(t)
	authService, err := NewAuthService(db)
	require.NoError(t, err)

	// Register a user to get tokens
	req := &RegisterRequest{
		Email:    "token@example.com",
		Username: "tokenuser",
		Password: "password123",
	}
	user, tokens, err := authService.Register(req)
	require.NoError(t, err)

	t.Run("valid token", func(t *testing.T) {
		claims, err := authService.ValidateAccessToken(tokens.AccessToken)

		assert.NoError(t, err)
		assert.NotNil(t, claims)
		assert.Equal(t, user.ID.String(), claims.UserID)
		assert.Equal(t, user.Email, claims.Email)
		assert.Equal(t, user.Username, claims.Username)
		assert.Equal(t, user.Role, claims.Role)
	})

	t.Run("invalid token", func(t *testing.T) {
		claims, err := authService.ValidateAccessToken("invalid.token.here")

		assert.Error(t, err)
		assert.Equal(t, ErrInvalidToken, err)
		assert.Nil(t, claims)
	})

	t.Run("empty token", func(t *testing.T) {
		claims, err := authService.ValidateAccessToken("")

		assert.Error(t, err)
		assert.Nil(t, claims)
	})
}

func TestAuthService_RefreshToken(t *testing.T) {
	db := setupTestDB(t)
	authService, err := NewAuthService(db)
	require.NoError(t, err)

	// Register a user to get tokens
	req := &RegisterRequest{
		Email:    "refresh@example.com",
		Username: "refreshuser",
		Password: "password123",
	}
	_, tokens, err := authService.Register(req)
	require.NoError(t, err)

	t.Run("valid refresh token", func(t *testing.T) {
		newTokens, err := authService.RefreshToken(tokens.RefreshToken)

		assert.NoError(t, err)
		assert.NotNil(t, newTokens)
		assert.NotEmpty(t, newTokens.AccessToken)
		assert.NotEmpty(t, newTokens.RefreshToken)
		// New refresh token should be different
		assert.NotEqual(t, tokens.RefreshToken, newTokens.RefreshToken)
	})

	t.Run("invalid refresh token", func(t *testing.T) {
		newTokens, err := authService.RefreshToken("invalid-refresh-token")

		assert.Error(t, err)
		assert.Equal(t, ErrInvalidToken, err)
		assert.Nil(t, newTokens)
	})

	t.Run("revoked refresh token", func(t *testing.T) {
		// The original token should be revoked after refresh
		newTokens, err := authService.RefreshToken(tokens.RefreshToken)

		assert.Error(t, err)
		assert.Equal(t, ErrTokenRevoked, err)
		assert.Nil(t, newTokens)
	})
}

func TestAuthService_Logout(t *testing.T) {
	db := setupTestDB(t)
	authService, err := NewAuthService(db)
	require.NoError(t, err)

	// Register a user
	req := &RegisterRequest{
		Email:    "logout@example.com",
		Username: "logoutuser",
		Password: "password123",
	}
	user, tokens, err := authService.Register(req)
	require.NoError(t, err)

	t.Run("successful logout", func(t *testing.T) {
		err := authService.Logout(user.ID)
		assert.NoError(t, err)

		// Refresh token should be revoked
		_, err = authService.RefreshToken(tokens.RefreshToken)
		assert.Error(t, err)
		assert.Equal(t, ErrTokenRevoked, err)
	})
}

func TestAuthService_GetUserByID(t *testing.T) {
	db := setupTestDB(t)
	authService, err := NewAuthService(db)
	require.NoError(t, err)

	// Create a test user
	testUser := createTestUser(t, db, "getuser@example.com", "getuser", "password123")

	t.Run("existing user", func(t *testing.T) {
		user, err := authService.GetUserByID(testUser.ID)

		assert.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, testUser.ID, user.ID)
		assert.Equal(t, testUser.Email, user.Email)
	})

	t.Run("non-existent user", func(t *testing.T) {
		user, err := authService.GetUserByID(uuid.New())

		assert.Error(t, err)
		assert.Equal(t, ErrUserNotFound, err)
		assert.Nil(t, user)
	})
}

func TestJWTClaims_Expiration(t *testing.T) {
	db := setupTestDB(t)
	authService, err := NewAuthService(db)
	require.NoError(t, err)

	// Register a user
	req := &RegisterRequest{
		Email:    "expiry@example.com",
		Username: "expiryuser",
		Password: "password123",
	}
	_, tokens, err := authService.Register(req)
	require.NoError(t, err)

	t.Run("token expiry time is set correctly", func(t *testing.T) {
		claims, err := authService.ValidateAccessToken(tokens.AccessToken)
		require.NoError(t, err)

		// Token should expire in approximately 15 minutes
		expiryTime := claims.ExpiresAt.Time
		expectedExpiry := time.Now().Add(15 * time.Minute)

		// Allow 1 minute tolerance
		assert.WithinDuration(t, expectedExpiry, expiryTime, time.Minute)
	})
}
