// Package services provides business logic services for the core service.
package services

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/cloud-devbox/services/core/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupTemplateTestDB creates an in-memory SQLite database for template testing
func setupTemplateTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// Auto-migrate models
	err = db.AutoMigrate(
		&models.User{},
		&models.Template{},
		&models.TemplateVersion{},
	)
	require.NoError(t, err)

	return db
}

// createSystemUser creates the system user for templates
func createSystemUser(t *testing.T, db *gorm.DB) *models.User {
	user := &models.User{
		ID:          uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		Email:       "system@devbox.io",
		Username:    "system",
		DisplayName: "System",
		Role:        "admin",
	}
	err := db.Create(user).Error
	require.NoError(t, err)
	return user
}

// createTestTemplateUser creates a regular test user
func createTestTemplateUser(t *testing.T, db *gorm.DB) *models.User {
	user := &models.User{
		ID:          uuid.New(),
		Email:       "testuser@example.com",
		Username:    "testuser",
		DisplayName: "Test User",
		Role:        "developer",
	}
	err := db.Create(user).Error
	require.NoError(t, err)
	return user
}

// seedTemplates creates multiple templates for testing
func seedTemplates(t *testing.T, db *gorm.DB, userID uuid.UUID, count int) {
	categories := []string{"前端", "后端", "全栈", "数据库", "DevOps", "其他"}
	
	for i := 0; i < count; i++ {
		template := &models.Template{
			Name:        fmt.Sprintf("template-%d", i),
			DisplayName: fmt.Sprintf("Template %d", i),
			Description: fmt.Sprintf("Description for template %d", i),
			Category:    categories[i%len(categories)],
			Tags:        []string{"test", fmt.Sprintf("tag-%d", i%5)},
			Runtime: models.JSONMap{
				"language": "javascript",
				"version":  "18",
			},
			DefaultConfig: models.JSONMap{
				"cpu":    "1",
				"memory": "2Gi",
			},
			Dockerfile: "FROM node:18-alpine\nWORKDIR /app",
			IsPublic:   true,
			CreatedBy:  userID,
			Version:    "1.0.0",
		}
		err := db.Create(template).Error
		require.NoError(t, err)
	}
}


func TestTemplateService_Create(t *testing.T) {
	db := setupTemplateTestDB(t)
	user := createTestTemplateUser(t, db)
	
	templateService, err := NewTemplateService(db, "")
	require.NoError(t, err)

	t.Run("successful creation", func(t *testing.T) {
		input := CreateTemplateInput{
			Name:        "test-template",
			DisplayName: "Test Template",
			Description: "A test template",
			Category:    "前端",
			Tags:        []string{"React", "TypeScript"},
			Runtime: map[string]interface{}{
				"language": "javascript",
				"version":  "18",
			},
			DefaultConfig: map[string]interface{}{
				"cpu":    "1",
				"memory": "2Gi",
			},
			Dockerfile: "FROM node:18-alpine",
			IsPublic:   true,
		}

		template, err := templateService.Create(context.Background(), user.ID, input)

		assert.NoError(t, err)
		assert.NotNil(t, template)
		assert.Equal(t, input.Name, template.Name)
		assert.Equal(t, input.DisplayName, template.DisplayName)
		assert.Equal(t, input.Category, template.Category)
		assert.Equal(t, "1.0.0", template.Version)
	})

	t.Run("duplicate name", func(t *testing.T) {
		input := CreateTemplateInput{
			Name:        "test-template",
			DisplayName: "Another Template",
			Category:    "后端",
			Runtime:     map[string]interface{}{"language": "go"},
			DefaultConfig: map[string]interface{}{"cpu": "1"},
			Dockerfile:  "FROM golang:1.21",
		}

		template, err := templateService.Create(context.Background(), user.ID, input)

		assert.Error(t, err)
		assert.Equal(t, ErrTemplateNameExists, err)
		assert.Nil(t, template)
	})

	t.Run("invalid category", func(t *testing.T) {
		input := CreateTemplateInput{
			Name:        "invalid-category-template",
			DisplayName: "Invalid Category",
			Category:    "InvalidCategory",
			Runtime:     map[string]interface{}{"language": "go"},
			DefaultConfig: map[string]interface{}{"cpu": "1"},
			Dockerfile:  "FROM golang:1.21",
		}

		template, err := templateService.Create(context.Background(), user.ID, input)

		assert.Error(t, err)
		assert.Equal(t, ErrInvalidCategory, err)
		assert.Nil(t, template)
	})
}

func TestTemplateService_GetByID(t *testing.T) {
	db := setupTemplateTestDB(t)
	user := createTestTemplateUser(t, db)
	
	templateService, err := NewTemplateService(db, "")
	require.NoError(t, err)

	// Create a template
	input := CreateTemplateInput{
		Name:        "get-test",
		DisplayName: "Get Test",
		Category:    "前端",
		Runtime:     map[string]interface{}{"language": "javascript"},
		DefaultConfig: map[string]interface{}{"cpu": "1"},
		Dockerfile:  "FROM node:18",
	}
	created, err := templateService.Create(context.Background(), user.ID, input)
	require.NoError(t, err)

	t.Run("existing template", func(t *testing.T) {
		template, err := templateService.GetByID(context.Background(), created.ID)

		assert.NoError(t, err)
		assert.NotNil(t, template)
		assert.Equal(t, created.ID, template.ID)
		assert.Equal(t, created.Name, template.Name)
	})

	t.Run("non-existent template", func(t *testing.T) {
		template, err := templateService.GetByID(context.Background(), uuid.New())

		assert.Error(t, err)
		assert.Equal(t, ErrTemplateNotFound, err)
		assert.Nil(t, template)
	})
}

func TestTemplateService_List(t *testing.T) {
	db := setupTemplateTestDB(t)
	user := createSystemUser(t, db)
	seedTemplates(t, db, user.ID, 50)
	
	templateService, err := NewTemplateService(db, "")
	require.NoError(t, err)

	t.Run("list all templates", func(t *testing.T) {
		result, err := templateService.List(context.Background(), TemplateFilter{})

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, int64(50), result.Total)
		assert.Equal(t, 20, len(result.Templates)) // Default page size
		assert.Equal(t, 1, result.Page)
		assert.Equal(t, 3, result.TotalPages)
	})

	t.Run("filter by category", func(t *testing.T) {
		result, err := templateService.List(context.Background(), TemplateFilter{
			Category: "前端",
		})

		assert.NoError(t, err)
		assert.NotNil(t, result)
		for _, tpl := range result.Templates {
			assert.Equal(t, "前端", tpl.Category)
		}
	})

	t.Run("search by name", func(t *testing.T) {
		result, err := templateService.List(context.Background(), TemplateFilter{
			Search: "Template 1",
		})

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Greater(t, result.Total, int64(0))
	})

	t.Run("pagination", func(t *testing.T) {
		result, err := templateService.List(context.Background(), TemplateFilter{
			Page:     2,
			PageSize: 10,
		})

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, 2, result.Page)
		assert.Equal(t, 10, result.PageSize)
		assert.Equal(t, 10, len(result.Templates))
	})
}


func TestTemplateService_Update(t *testing.T) {
	db := setupTemplateTestDB(t)
	user := createTestTemplateUser(t, db)
	
	templateService, err := NewTemplateService(db, "")
	require.NoError(t, err)

	// Create a template
	input := CreateTemplateInput{
		Name:        "update-test",
		DisplayName: "Update Test",
		Category:    "前端",
		Runtime:     map[string]interface{}{"language": "javascript"},
		DefaultConfig: map[string]interface{}{"cpu": "1"},
		Dockerfile:  "FROM node:18",
	}
	created, err := templateService.Create(context.Background(), user.ID, input)
	require.NoError(t, err)

	t.Run("successful update by owner", func(t *testing.T) {
		newDisplayName := "Updated Display Name"
		updateInput := UpdateTemplateInput{
			DisplayName: &newDisplayName,
		}

		updated, err := templateService.Update(context.Background(), created.ID, user.ID, false, updateInput)

		assert.NoError(t, err)
		assert.NotNil(t, updated)
		assert.Equal(t, newDisplayName, updated.DisplayName)
	})

	t.Run("unauthorized update", func(t *testing.T) {
		otherUser := uuid.New()
		newDisplayName := "Unauthorized Update"
		updateInput := UpdateTemplateInput{
			DisplayName: &newDisplayName,
		}

		updated, err := templateService.Update(context.Background(), created.ID, otherUser, false, updateInput)

		assert.Error(t, err)
		assert.Equal(t, ErrUnauthorized, err)
		assert.Nil(t, updated)
	})

	t.Run("admin can update any template", func(t *testing.T) {
		otherUser := uuid.New()
		newDescription := "Admin updated description"
		updateInput := UpdateTemplateInput{
			Description: &newDescription,
		}

		updated, err := templateService.Update(context.Background(), created.ID, otherUser, true, updateInput)

		assert.NoError(t, err)
		assert.NotNil(t, updated)
		assert.Equal(t, newDescription, updated.Description)
	})
}

func TestTemplateService_Delete(t *testing.T) {
	db := setupTemplateTestDB(t)
	user := createTestTemplateUser(t, db)
	
	templateService, err := NewTemplateService(db, "")
	require.NoError(t, err)

	t.Run("successful delete by owner", func(t *testing.T) {
		input := CreateTemplateInput{
			Name:        "delete-test",
			DisplayName: "Delete Test",
			Category:    "前端",
			Runtime:     map[string]interface{}{"language": "javascript"},
			DefaultConfig: map[string]interface{}{"cpu": "1"},
			Dockerfile:  "FROM node:18",
		}
		created, err := templateService.Create(context.Background(), user.ID, input)
		require.NoError(t, err)

		err = templateService.Delete(context.Background(), created.ID, user.ID, false)

		assert.NoError(t, err)

		// Verify deletion
		_, err = templateService.GetByID(context.Background(), created.ID)
		assert.Equal(t, ErrTemplateNotFound, err)
	})

	t.Run("unauthorized delete", func(t *testing.T) {
		input := CreateTemplateInput{
			Name:        "delete-test-2",
			DisplayName: "Delete Test 2",
			Category:    "前端",
			Runtime:     map[string]interface{}{"language": "javascript"},
			DefaultConfig: map[string]interface{}{"cpu": "1"},
			Dockerfile:  "FROM node:18",
		}
		created, err := templateService.Create(context.Background(), user.ID, input)
		require.NoError(t, err)

		otherUser := uuid.New()
		err = templateService.Delete(context.Background(), created.ID, otherUser, false)

		assert.Error(t, err)
		assert.Equal(t, ErrUnauthorized, err)
	})
}

func TestTemplateService_GetCategories(t *testing.T) {
	db := setupTemplateTestDB(t)
	user := createSystemUser(t, db)
	seedTemplates(t, db, user.ID, 30)
	
	templateService, err := NewTemplateService(db, "")
	require.NoError(t, err)

	categories, err := templateService.GetCategories(context.Background())

	assert.NoError(t, err)
	assert.NotNil(t, categories)
	assert.Greater(t, len(categories), 0)

	// Each category should have name and count
	for _, cat := range categories {
		assert.Contains(t, cat, "name")
		assert.Contains(t, cat, "count")
	}
}

// ============================================
// PROPERTY-BASED TESTS
// ============================================

// TestProperty1_TemplateListResponseTime tests that template list responds within 2 seconds
// **Feature: cloud-devbox, Property 1: 模板列表响应 < 2秒**
// **Validates: Requirements 1.1**
func TestProperty1_TemplateListResponseTime(t *testing.T) {
	db := setupTemplateTestDB(t)
	user := createSystemUser(t, db)
	
	// Seed a large number of templates to simulate production load
	seedTemplates(t, db, user.ID, 100)
	
	templateService, err := NewTemplateService(db, "")
	require.NoError(t, err)

	// Test various filter combinations
	testCases := []struct {
		name   string
		filter TemplateFilter
	}{
		{"no filter", TemplateFilter{}},
		{"category filter", TemplateFilter{Category: "前端"}},
		{"search filter", TemplateFilter{Search: "Template"}},
		{"pagination", TemplateFilter{Page: 1, PageSize: 50}},
		{"combined filters", TemplateFilter{Category: "后端", Search: "Template", Page: 1, PageSize: 20}},
	}

	maxResponseTime := 2 * time.Second

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			start := time.Now()
			
			result, err := templateService.List(context.Background(), tc.filter)
			
			elapsed := time.Since(start)

			assert.NoError(t, err)
			assert.NotNil(t, result)
			assert.Less(t, elapsed, maxResponseTime, 
				"Template list response time (%v) exceeded 2 seconds for filter: %s", elapsed, tc.name)
			
			t.Logf("Filter '%s': response time = %v, results = %d", tc.name, elapsed, len(result.Templates))
		})
	}
}

// TestProperty1_TemplateListResponseTimeWithLoad performs stress testing
// **Feature: cloud-devbox, Property 1: 模板列表响应 < 2秒**
// **Validates: Requirements 1.1**
func TestProperty1_TemplateListResponseTimeWithLoad(t *testing.T) {
	db := setupTemplateTestDB(t)
	user := createSystemUser(t, db)
	
	// Seed 500 templates to simulate heavy load
	seedTemplates(t, db, user.ID, 500)
	
	templateService, err := NewTemplateService(db, "")
	require.NoError(t, err)

	maxResponseTime := 2 * time.Second
	iterations := 10

	var totalTime time.Duration
	var maxTime time.Duration

	for i := 0; i < iterations; i++ {
		start := time.Now()
		
		result, err := templateService.List(context.Background(), TemplateFilter{
			Page:     (i % 5) + 1,
			PageSize: 20,
		})
		
		elapsed := time.Since(start)
		totalTime += elapsed
		if elapsed > maxTime {
			maxTime = elapsed
		}

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Less(t, elapsed, maxResponseTime,
			"Iteration %d: response time (%v) exceeded 2 seconds", i, elapsed)
	}

	avgTime := totalTime / time.Duration(iterations)
	t.Logf("Performance summary: avg=%v, max=%v over %d iterations", avgTime, maxTime, iterations)
	
	// Average should be well under 2 seconds
	assert.Less(t, avgTime, maxResponseTime,
		"Average response time (%v) exceeded 2 seconds", avgTime)
}
