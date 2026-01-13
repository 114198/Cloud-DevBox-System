// Package services provides business logic for the core service.
package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/cloud-devbox/services/core/internal/models"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// TemplateService handles template-related business logic
type TemplateService struct {
	db    *gorm.DB
	redis *redis.Client
}

// NewTemplateService creates a new template service
func NewTemplateService(db *gorm.DB, redisURL string) (*TemplateService, error) {
	var redisClient *redis.Client
	if redisURL != "" {
		opt, err := redis.ParseURL(redisURL)
		if err != nil {
			return nil, fmt.Errorf("failed to parse redis URL: %w", err)
		}
		redisClient = redis.NewClient(opt)
	}

	return &TemplateService{
		db:    db,
		redis: redisClient,
	}, nil
}

// TemplateFilter represents filter options for listing templates
type TemplateFilter struct {
	Category string
	Tags     []string
	Search   string
	IsPublic *bool
	Page     int
	PageSize int
}

// TemplateListResult represents the result of listing templates
type TemplateListResult struct {
	Templates  []models.Template `json:"templates"`
	Total      int64             `json:"total"`
	Page       int               `json:"page"`
	PageSize   int               `json:"page_size"`
	TotalPages int               `json:"total_pages"`
}

// CreateTemplateInput represents input for creating a template
type CreateTemplateInput struct {
	Name          string                 `json:"name" binding:"required"`
	DisplayName   string                 `json:"display_name" binding:"required"`
	Description   string                 `json:"description"`
	Category      string                 `json:"category" binding:"required"`
	Tags          []string               `json:"tags"`
	Icon          string                 `json:"icon"`
	Runtime       map[string]interface{} `json:"runtime" binding:"required"`
	DefaultConfig map[string]interface{} `json:"default_config" binding:"required"`
	Dockerfile    string                 `json:"dockerfile" binding:"required"`
	InitScript    string                 `json:"init_script"`
	Extensions    []interface{}          `json:"extensions"`
	IsPublic      bool                   `json:"is_public"`
	ParentID      *uuid.UUID             `json:"parent_id"`
}

// UpdateTemplateInput represents input for updating a template
type UpdateTemplateInput struct {
	DisplayName   *string                `json:"display_name"`
	Description   *string                `json:"description"`
	Category      *string                `json:"category"`
	Tags          []string               `json:"tags"`
	Icon          *string                `json:"icon"`
	Runtime       map[string]interface{} `json:"runtime"`
	DefaultConfig map[string]interface{} `json:"default_config"`
	Dockerfile    *string                `json:"dockerfile"`
	InitScript    *string                `json:"init_script"`
	Extensions    []interface{}          `json:"extensions"`
	IsPublic      *bool                  `json:"is_public"`
}

// Common errors
var (
	ErrTemplateNotFound      = errors.New("template not found")
	ErrTemplateNameExists    = errors.New("template name already exists")
	ErrInvalidCategory       = errors.New("invalid template category")
	ErrUnauthorized          = errors.New("unauthorized to perform this action")
	ErrCannotDeleteSystemTpl = errors.New("cannot delete system template")
)

// List returns a paginated list of templates with optional filters
func (s *TemplateService) List(ctx context.Context, filter TemplateFilter) (*TemplateListResult, error) {
	// Set defaults
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PageSize < 1 || filter.PageSize > 100 {
		filter.PageSize = 20
	}

	query := s.db.WithContext(ctx).Model(&models.Template{})

	// Apply filters
	if filter.Category != "" {
		query = query.Where("category = ?", filter.Category)
	}

	if len(filter.Tags) > 0 {
		query = query.Where("tags && ?", "{"+strings.Join(filter.Tags, ",")+"}")
	}

	if filter.Search != "" {
		searchPattern := "%" + filter.Search + "%"
		query = query.Where("display_name ILIKE ? OR description ILIKE ? OR name ILIKE ?",
			searchPattern, searchPattern, searchPattern)
	}

	if filter.IsPublic != nil {
		query = query.Where("is_public = ?", *filter.IsPublic)
	}

	// Count total
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("failed to count templates: %w", err)
	}

	// Calculate pagination
	offset := (filter.Page - 1) * filter.PageSize
	totalPages := int((total + int64(filter.PageSize) - 1) / int64(filter.PageSize))

	// Fetch templates
	var templates []models.Template
	if err := query.
		Preload("Creator").
		Order("category ASC, display_name ASC").
		Offset(offset).
		Limit(filter.PageSize).
		Find(&templates).Error; err != nil {
		return nil, fmt.Errorf("failed to list templates: %w", err)
	}

	return &TemplateListResult{
		Templates:  templates,
		Total:      total,
		Page:       filter.Page,
		PageSize:   filter.PageSize,
		TotalPages: totalPages,
	}, nil
}

// GetByID retrieves a template by its ID
func (s *TemplateService) GetByID(ctx context.Context, id uuid.UUID) (*models.Template, error) {
	var template models.Template
	if err := s.db.WithContext(ctx).
		Preload("Creator").
		Preload("Parent").
		First(&template, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTemplateNotFound
		}
		return nil, fmt.Errorf("failed to get template: %w", err)
	}
	return &template, nil
}

// GetByName retrieves a template by its name
func (s *TemplateService) GetByName(ctx context.Context, name string) (*models.Template, error) {
	var template models.Template
	if err := s.db.WithContext(ctx).
		Preload("Creator").
		Preload("Parent").
		First(&template, "name = ?", name).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTemplateNotFound
		}
		return nil, fmt.Errorf("failed to get template: %w", err)
	}
	return &template, nil
}

// Create creates a new template
func (s *TemplateService) Create(ctx context.Context, userID uuid.UUID, input CreateTemplateInput) (*models.Template, error) {
	// Validate category
	if !models.IsValidCategory(input.Category) {
		return nil, ErrInvalidCategory
	}

	// Check if name already exists
	var count int64
	if err := s.db.WithContext(ctx).Model(&models.Template{}).
		Where("name = ?", input.Name).Count(&count).Error; err != nil {
		return nil, fmt.Errorf("failed to check template name: %w", err)
	}
	if count > 0 {
		return nil, ErrTemplateNameExists
	}

	// Create template
	template := &models.Template{
		Name:          input.Name,
		DisplayName:   input.DisplayName,
		Description:   input.Description,
		Category:      input.Category,
		Tags:          input.Tags,
		Icon:          input.Icon,
		Runtime:       models.JSONMap(input.Runtime),
		DefaultConfig: models.JSONMap(input.DefaultConfig),
		Dockerfile:    input.Dockerfile,
		InitScript:    input.InitScript,
		Extensions:    models.JSONArray(input.Extensions),
		IsPublic:      input.IsPublic,
		CreatedBy:     userID,
		ParentID:      input.ParentID,
		Version:       "1.0.0",
	}

	if err := s.db.WithContext(ctx).Create(template).Error; err != nil {
		return nil, fmt.Errorf("failed to create template: %w", err)
	}

	// Create initial version
	version := &models.TemplateVersion{
		TemplateID:    template.ID,
		Version:       template.Version,
		Runtime:       template.Runtime,
		DefaultConfig: template.DefaultConfig,
		Dockerfile:    template.Dockerfile,
		InitScript:    template.InitScript,
		Extensions:    template.Extensions,
		ChangeLog:     "Initial version",
		CreatedBy:     userID,
	}

	if err := s.db.WithContext(ctx).Create(version).Error; err != nil {
		// Log error but don't fail the template creation
		fmt.Printf("Warning: failed to create initial version: %v\n", err)
	}

	// Invalidate cache
	s.invalidateCache(ctx)

	return template, nil
}

// Update updates an existing template
func (s *TemplateService) Update(ctx context.Context, id uuid.UUID, userID uuid.UUID, isAdmin bool, input UpdateTemplateInput) (*models.Template, error) {
	// Get existing template
	template, err := s.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Check authorization (only creator or admin can update)
	if template.CreatedBy != userID && !isAdmin {
		return nil, ErrUnauthorized
	}

	// Validate category if provided
	if input.Category != nil && !models.IsValidCategory(*input.Category) {
		return nil, ErrInvalidCategory
	}

	// Build updates map
	updates := make(map[string]interface{})

	if input.DisplayName != nil {
		updates["display_name"] = *input.DisplayName
	}
	if input.Description != nil {
		updates["description"] = *input.Description
	}
	if input.Category != nil {
		updates["category"] = *input.Category
	}
	if input.Tags != nil {
		updates["tags"] = input.Tags
	}
	if input.Icon != nil {
		updates["icon"] = *input.Icon
	}
	if input.Runtime != nil {
		updates["runtime"] = models.JSONMap(input.Runtime)
	}
	if input.DefaultConfig != nil {
		updates["default_config"] = models.JSONMap(input.DefaultConfig)
	}
	if input.Dockerfile != nil {
		updates["dockerfile"] = *input.Dockerfile
	}
	if input.InitScript != nil {
		updates["init_script"] = *input.InitScript
	}
	if input.Extensions != nil {
		updates["extensions"] = models.JSONArray(input.Extensions)
	}
	if input.IsPublic != nil {
		updates["is_public"] = *input.IsPublic
	}

	updates["updated_at"] = time.Now()

	if err := s.db.WithContext(ctx).Model(template).Updates(updates).Error; err != nil {
		return nil, fmt.Errorf("failed to update template: %w", err)
	}

	// Invalidate cache
	s.invalidateCache(ctx)

	// Reload template
	return s.GetByID(ctx, id)
}

// Delete deletes a template
func (s *TemplateService) Delete(ctx context.Context, id uuid.UUID, userID uuid.UUID, isAdmin bool) error {
	// Get existing template
	template, err := s.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// Check if it's a system template
	systemUserID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	if template.CreatedBy == systemUserID && !isAdmin {
		return ErrCannotDeleteSystemTpl
	}

	// Check authorization (only creator or admin can delete)
	if template.CreatedBy != userID && !isAdmin {
		return ErrUnauthorized
	}

	// Delete template (cascade will delete versions)
	if err := s.db.WithContext(ctx).Delete(template).Error; err != nil {
		return fmt.Errorf("failed to delete template: %w", err)
	}

	// Invalidate cache
	s.invalidateCache(ctx)

	return nil
}

// GetCategories returns all available categories with template counts
func (s *TemplateService) GetCategories(ctx context.Context) ([]map[string]interface{}, error) {
	type CategoryCount struct {
		Category string
		Count    int64
	}

	var results []CategoryCount
	if err := s.db.WithContext(ctx).
		Model(&models.Template{}).
		Select("category, count(*) as count").
		Where("is_public = ?", true).
		Group("category").
		Order("category ASC").
		Scan(&results).Error; err != nil {
		return nil, fmt.Errorf("failed to get categories: %w", err)
	}

	categories := make([]map[string]interface{}, len(results))
	for i, r := range results {
		categories[i] = map[string]interface{}{
			"name":  r.Category,
			"count": r.Count,
		}
	}

	return categories, nil
}

// GetTags returns all unique tags with usage counts
func (s *TemplateService) GetTags(ctx context.Context) ([]map[string]interface{}, error) {
	type TagCount struct {
		Tag   string
		Count int64
	}

	var results []TagCount
	if err := s.db.WithContext(ctx).Raw(`
		SELECT unnest(tags) as tag, count(*) as count
		FROM templates
		WHERE is_public = true
		GROUP BY tag
		ORDER BY count DESC, tag ASC
	`).Scan(&results).Error; err != nil {
		return nil, fmt.Errorf("failed to get tags: %w", err)
	}

	tags := make([]map[string]interface{}, len(results))
	for i, r := range results {
		tags[i] = map[string]interface{}{
			"name":  r.Tag,
			"count": r.Count,
		}
	}

	return tags, nil
}

// invalidateCache invalidates template-related cache
func (s *TemplateService) invalidateCache(ctx context.Context) {
	if s.redis == nil {
		return
	}

	// Delete all template cache keys
	keys, err := s.redis.Keys(ctx, "templates:*").Result()
	if err != nil {
		return
	}

	if len(keys) > 0 {
		s.redis.Del(ctx, keys...)
	}
}
