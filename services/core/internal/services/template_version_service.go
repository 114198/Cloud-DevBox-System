// Package services provides business logic for the core service.
package services

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/cloud-devbox/services/core/internal/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// TemplateVersionService handles template version-related business logic
type TemplateVersionService struct {
	db              *gorm.DB
	templateService *TemplateService
}

// NewTemplateVersionService creates a new template version service
func NewTemplateVersionService(db *gorm.DB, templateService *TemplateService) *TemplateVersionService {
	return &TemplateVersionService{
		db:              db,
		templateService: templateService,
	}
}

// CreateVersionInput represents input for creating a new template version
type CreateVersionInput struct {
	Runtime       map[string]interface{} `json:"runtime"`
	DefaultConfig map[string]interface{} `json:"default_config"`
	Dockerfile    string                 `json:"dockerfile"`
	InitScript    string                 `json:"init_script"`
	Extensions    []interface{}          `json:"extensions"`
	ChangeLog     string                 `json:"change_log" binding:"required"`
	VersionBump   string                 `json:"version_bump"` // "major", "minor", "patch"
}

// Common errors
var (
	ErrVersionNotFound    = errors.New("template version not found")
	ErrInvalidVersionBump = errors.New("invalid version bump type, must be 'major', 'minor', or 'patch'")
)

// ListVersions returns all versions of a template
func (s *TemplateVersionService) ListVersions(ctx context.Context, templateID uuid.UUID) ([]models.TemplateVersion, error) {
	var versions []models.TemplateVersion
	if err := s.db.WithContext(ctx).
		Where("template_id = ?", templateID).
		Order("created_at DESC").
		Find(&versions).Error; err != nil {
		return nil, fmt.Errorf("failed to list template versions: %w", err)
	}
	return versions, nil
}

// GetVersion retrieves a specific version
func (s *TemplateVersionService) GetVersion(ctx context.Context, versionID uuid.UUID) (*models.TemplateVersion, error) {
	var version models.TemplateVersion
	if err := s.db.WithContext(ctx).
		Preload("Template").
		Preload("Creator").
		First(&version, "id = ?", versionID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrVersionNotFound
		}
		return nil, fmt.Errorf("failed to get template version: %w", err)
	}
	return &version, nil
}

// GetVersionByNumber retrieves a specific version by version number
func (s *TemplateVersionService) GetVersionByNumber(ctx context.Context, templateID uuid.UUID, versionNumber string) (*models.TemplateVersion, error) {
	var version models.TemplateVersion
	if err := s.db.WithContext(ctx).
		Preload("Template").
		Preload("Creator").
		First(&version, "template_id = ? AND version = ?", templateID, versionNumber).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrVersionNotFound
		}
		return nil, fmt.Errorf("failed to get template version: %w", err)
	}
	return &version, nil
}

// CreateVersion creates a new version of a template
func (s *TemplateVersionService) CreateVersion(ctx context.Context, templateID uuid.UUID, userID uuid.UUID, isAdmin bool, input CreateVersionInput) (*models.TemplateVersion, error) {
	// Get the template
	template, err := s.templateService.GetByID(ctx, templateID)
	if err != nil {
		return nil, err
	}

	// Check authorization
	if template.CreatedBy != userID && !isAdmin {
		return nil, ErrUnauthorized
	}

	// Calculate new version
	versionBump := input.VersionBump
	if versionBump == "" {
		versionBump = "patch"
	}

	newVersion, err := bumpVersion(template.Version, versionBump)
	if err != nil {
		return nil, err
	}

	// Use template's current values if not provided
	runtime := template.Runtime
	if input.Runtime != nil {
		runtime = models.JSONMap(input.Runtime)
	}

	defaultConfig := template.DefaultConfig
	if input.DefaultConfig != nil {
		defaultConfig = models.JSONMap(input.DefaultConfig)
	}

	dockerfile := template.Dockerfile
	if input.Dockerfile != "" {
		dockerfile = input.Dockerfile
	}

	initScript := template.InitScript
	if input.InitScript != "" {
		initScript = input.InitScript
	}

	extensions := template.Extensions
	if input.Extensions != nil {
		extensions = models.JSONArray(input.Extensions)
	}

	// Create version record
	version := &models.TemplateVersion{
		TemplateID:    templateID,
		Version:       newVersion,
		Runtime:       runtime,
		DefaultConfig: defaultConfig,
		Dockerfile:    dockerfile,
		InitScript:    initScript,
		Extensions:    extensions,
		ChangeLog:     input.ChangeLog,
		CreatedBy:     userID,
	}

	// Start transaction
	tx := s.db.WithContext(ctx).Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Create version
	if err := tx.Create(version).Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to create template version: %w", err)
	}

	// Update template with new version and config
	updates := map[string]interface{}{
		"version":        newVersion,
		"runtime":        runtime,
		"default_config": defaultConfig,
		"dockerfile":     dockerfile,
		"init_script":    initScript,
		"extensions":     extensions,
	}

	if err := tx.Model(template).Updates(updates).Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to update template version: %w", err)
	}

	if err := tx.Commit().Error; err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Invalidate cache
	s.templateService.invalidateCache(ctx)

	return version, nil
}

// RollbackToVersion rolls back a template to a specific version
func (s *TemplateVersionService) RollbackToVersion(ctx context.Context, templateID uuid.UUID, versionID uuid.UUID, userID uuid.UUID, isAdmin bool) (*models.Template, error) {
	// Get the template
	template, err := s.templateService.GetByID(ctx, templateID)
	if err != nil {
		return nil, err
	}

	// Check authorization
	if template.CreatedBy != userID && !isAdmin {
		return nil, ErrUnauthorized
	}

	// Get the version to rollback to
	version, err := s.GetVersion(ctx, versionID)
	if err != nil {
		return nil, err
	}

	// Verify version belongs to this template
	if version.TemplateID != templateID {
		return nil, ErrVersionNotFound
	}

	// Create a new version for the rollback
	newVersion, err := bumpVersion(template.Version, "patch")
	if err != nil {
		return nil, err
	}

	// Start transaction
	tx := s.db.WithContext(ctx).Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// Create rollback version record
	rollbackVersion := &models.TemplateVersion{
		TemplateID:    templateID,
		Version:       newVersion,
		Runtime:       version.Runtime,
		DefaultConfig: version.DefaultConfig,
		Dockerfile:    version.Dockerfile,
		InitScript:    version.InitScript,
		Extensions:    version.Extensions,
		ChangeLog:     fmt.Sprintf("Rollback to version %s", version.Version),
		CreatedBy:     userID,
	}

	if err := tx.Create(rollbackVersion).Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to create rollback version: %w", err)
	}

	// Update template
	updates := map[string]interface{}{
		"version":        newVersion,
		"runtime":        version.Runtime,
		"default_config": version.DefaultConfig,
		"dockerfile":     version.Dockerfile,
		"init_script":    version.InitScript,
		"extensions":     version.Extensions,
	}

	if err := tx.Model(template).Updates(updates).Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to update template: %w", err)
	}

	if err := tx.Commit().Error; err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Invalidate cache
	s.templateService.invalidateCache(ctx)

	return s.templateService.GetByID(ctx, templateID)
}

// bumpVersion increments a semantic version
func bumpVersion(currentVersion string, bumpType string) (string, error) {
	// Parse version (e.g., "1.2.3")
	re := regexp.MustCompile(`^(\d+)\.(\d+)\.(\d+)$`)
	matches := re.FindStringSubmatch(currentVersion)

	if len(matches) != 4 {
		// If version doesn't match semver, start fresh
		return "1.0.0", nil
	}

	major, _ := strconv.Atoi(matches[1])
	minor, _ := strconv.Atoi(matches[2])
	patch, _ := strconv.Atoi(matches[3])

	switch strings.ToLower(bumpType) {
	case "major":
		major++
		minor = 0
		patch = 0
	case "minor":
		minor++
		patch = 0
	case "patch":
		patch++
	default:
		return "", ErrInvalidVersionBump
	}

	return fmt.Sprintf("%d.%d.%d", major, minor, patch), nil
}
