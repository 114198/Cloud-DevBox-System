// Package services provides business logic services for the core service.
package services

import (
	"context"
	"testing"

	"github.com/cloud-devbox/services/core/internal/models"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupRBACTestDB creates an in-memory SQLite database for RBAC testing
func setupRBACTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)

	// Auto-migrate models
	err = db.AutoMigrate(&models.User{})
	require.NoError(t, err)

	return db
}

// createRBACTestUser creates a test user with a specific role
func createRBACTestUser(t *testing.T, db *gorm.DB, email, username, role string) *models.User {
	user := &models.User{
		ID:          uuid.New(),
		Email:       email,
		Username:    username,
		DisplayName: username,
		Role:        role,
	}

	err := db.Create(user).Error
	require.NoError(t, err)

	return user
}

func TestRBACService_HasPermission(t *testing.T) {
	db := setupRBACTestDB(t)
	rbacService, err := NewRBACService(db, "")
	require.NoError(t, err)

	t.Run("developer permissions", func(t *testing.T) {
		// Developer should have environment CRUD
		assert.True(t, rbacService.HasPermission(RoleDeveloper, ResourceEnvironment, ActionCreate))
		assert.True(t, rbacService.HasPermission(RoleDeveloper, ResourceEnvironment, ActionRead))
		assert.True(t, rbacService.HasPermission(RoleDeveloper, ResourceEnvironment, ActionUpdate))
		assert.True(t, rbacService.HasPermission(RoleDeveloper, ResourceEnvironment, ActionDelete))

		// Developer should have template read-only
		assert.True(t, rbacService.HasPermission(RoleDeveloper, ResourceTemplate, ActionRead))
		assert.True(t, rbacService.HasPermission(RoleDeveloper, ResourceTemplate, ActionList))
		assert.False(t, rbacService.HasPermission(RoleDeveloper, ResourceTemplate, ActionCreate))
		assert.False(t, rbacService.HasPermission(RoleDeveloper, ResourceTemplate, ActionDelete))

		// Developer should not have admin permissions
		assert.False(t, rbacService.HasPermission(RoleDeveloper, ResourceAuditLog, ActionRead))
		assert.False(t, rbacService.HasPermission(RoleDeveloper, ResourceOrganization, ActionManage))
	})

	t.Run("org_admin permissions", func(t *testing.T) {
		// OrgAdmin should have template management
		assert.True(t, rbacService.HasPermission(RoleOrgAdmin, ResourceTemplate, ActionCreate))
		assert.True(t, rbacService.HasPermission(RoleOrgAdmin, ResourceTemplate, ActionUpdate))
		assert.True(t, rbacService.HasPermission(RoleOrgAdmin, ResourceTemplate, ActionDelete))

		// OrgAdmin should have user management
		assert.True(t, rbacService.HasPermission(RoleOrgAdmin, ResourceUser, ActionList))
		assert.True(t, rbacService.HasPermission(RoleOrgAdmin, ResourceUser, ActionManage))

		// OrgAdmin should have billing access
		assert.True(t, rbacService.HasPermission(RoleOrgAdmin, ResourceBilling, ActionRead))
		assert.True(t, rbacService.HasPermission(RoleOrgAdmin, ResourceBilling, ActionManage))

		// OrgAdmin inherits developer permissions
		assert.True(t, rbacService.HasPermission(RoleOrgAdmin, ResourceEnvironment, ActionCreate))
		assert.True(t, rbacService.HasPermission(RoleOrgAdmin, ResourceProject, ActionCreate))
	})

	t.Run("admin permissions", func(t *testing.T) {
		// Admin should have all permissions
		assert.True(t, rbacService.HasPermission(RoleAdmin, ResourceEnvironment, ActionManage))
		assert.True(t, rbacService.HasPermission(RoleAdmin, ResourceTemplate, ActionManage))
		assert.True(t, rbacService.HasPermission(RoleAdmin, ResourceProject, ActionManage))
		assert.True(t, rbacService.HasPermission(RoleAdmin, ResourceUser, ActionManage))
		assert.True(t, rbacService.HasPermission(RoleAdmin, ResourceOrganization, ActionManage))
		assert.True(t, rbacService.HasPermission(RoleAdmin, ResourceBilling, ActionManage))
		assert.True(t, rbacService.HasPermission(RoleAdmin, ResourceAuditLog, ActionRead))
	})

	t.Run("invalid role", func(t *testing.T) {
		assert.False(t, rbacService.HasPermission(Role("invalid"), ResourceEnvironment, ActionRead))
	})
}

func TestRBACService_GetUserRole(t *testing.T) {
	db := setupRBACTestDB(t)
	rbacService, err := NewRBACService(db, "")
	require.NoError(t, err)

	ctx := context.Background()

	t.Run("get developer role", func(t *testing.T) {
		user := createRBACTestUser(t, db, "dev@example.com", "devuser", "developer")

		role, err := rbacService.GetUserRole(ctx, user.ID)

		assert.NoError(t, err)
		assert.Equal(t, RoleDeveloper, role)
	})

	t.Run("get admin role", func(t *testing.T) {
		user := createRBACTestUser(t, db, "admin@example.com", "adminuser", "admin")

		role, err := rbacService.GetUserRole(ctx, user.ID)

		assert.NoError(t, err)
		assert.Equal(t, RoleAdmin, role)
	})

	t.Run("get org_admin role", func(t *testing.T) {
		user := createRBACTestUser(t, db, "orgadmin@example.com", "orgadminuser", "org_admin")

		role, err := rbacService.GetUserRole(ctx, user.ID)

		assert.NoError(t, err)
		assert.Equal(t, RoleOrgAdmin, role)
	})

	t.Run("non-existent user", func(t *testing.T) {
		role, err := rbacService.GetUserRole(ctx, uuid.New())

		assert.Error(t, err)
		assert.Equal(t, ErrUserNotFound, err)
		assert.Empty(t, role)
	})
}

func TestRBACService_SetUserRole(t *testing.T) {
	db := setupRBACTestDB(t)
	rbacService, err := NewRBACService(db, "")
	require.NoError(t, err)

	ctx := context.Background()

	t.Run("update role successfully", func(t *testing.T) {
		user := createRBACTestUser(t, db, "update@example.com", "updateuser", "developer")

		err := rbacService.SetUserRole(ctx, user.ID, RoleOrgAdmin)
		assert.NoError(t, err)

		// Verify role was updated
		role, err := rbacService.GetUserRole(ctx, user.ID)
		assert.NoError(t, err)
		assert.Equal(t, RoleOrgAdmin, role)
	})

	t.Run("invalid role", func(t *testing.T) {
		user := createRBACTestUser(t, db, "invalid@example.com", "invaliduser", "developer")

		err := rbacService.SetUserRole(ctx, user.ID, Role("invalid_role"))

		assert.Error(t, err)
		assert.Equal(t, ErrRoleNotFound, err)
	})
}

func TestRBACService_CheckPermission(t *testing.T) {
	db := setupRBACTestDB(t)
	rbacService, err := NewRBACService(db, "")
	require.NoError(t, err)

	ctx := context.Background()

	t.Run("developer has environment permission", func(t *testing.T) {
		user := createRBACTestUser(t, db, "devperm@example.com", "devpermuser", "developer")

		err := rbacService.CheckPermission(ctx, user.ID, ResourceEnvironment, ActionCreate)
		assert.NoError(t, err)
	})

	t.Run("developer denied template create", func(t *testing.T) {
		user := createRBACTestUser(t, db, "devdenied@example.com", "devdenieduser", "developer")

		err := rbacService.CheckPermission(ctx, user.ID, ResourceTemplate, ActionCreate)
		assert.Error(t, err)
		assert.Equal(t, ErrPermissionDenied, err)
	})

	t.Run("admin has all permissions", func(t *testing.T) {
		user := createRBACTestUser(t, db, "adminperm@example.com", "adminpermuser", "admin")

		// Admin should have access to everything
		assert.NoError(t, rbacService.CheckPermission(ctx, user.ID, ResourceEnvironment, ActionManage))
		assert.NoError(t, rbacService.CheckPermission(ctx, user.ID, ResourceTemplate, ActionManage))
		assert.NoError(t, rbacService.CheckPermission(ctx, user.ID, ResourceAuditLog, ActionRead))
	})
}

func TestRBACService_GetUserPermissions(t *testing.T) {
	db := setupRBACTestDB(t)
	rbacService, err := NewRBACService(db, "")
	require.NoError(t, err)

	ctx := context.Background()

	t.Run("get developer permissions", func(t *testing.T) {
		user := createRBACTestUser(t, db, "devperms@example.com", "devpermsuser", "developer")

		perms, err := rbacService.GetUserPermissions(ctx, user.ID)

		assert.NoError(t, err)
		assert.NotNil(t, perms)
		assert.Equal(t, RoleDeveloper, perms.Role)
		assert.NotEmpty(t, perms.Permissions)

		// Check that developer has expected permissions
		hasEnvCreate := false
		for _, p := range perms.Permissions {
			if p.Resource == ResourceEnvironment && p.Action == ActionCreate {
				hasEnvCreate = true
				break
			}
		}
		assert.True(t, hasEnvCreate)
	})

	t.Run("get admin permissions", func(t *testing.T) {
		user := createRBACTestUser(t, db, "adminperms@example.com", "adminpermsuser", "admin")

		perms, err := rbacService.GetUserPermissions(ctx, user.ID)

		assert.NoError(t, err)
		assert.NotNil(t, perms)
		assert.Equal(t, RoleAdmin, perms.Role)
		// Admin should have many permissions
		assert.Greater(t, len(perms.Permissions), 10)
	})
}

func TestRBACService_GetAllRoles(t *testing.T) {
	db := setupRBACTestDB(t)
	rbacService, err := NewRBACService(db, "")
	require.NoError(t, err)

	roles := rbacService.GetAllRoles()

	assert.Len(t, roles, 3)
	assert.Contains(t, roles, RoleDeveloper)
	assert.Contains(t, roles, RoleOrgAdmin)
	assert.Contains(t, roles, RoleAdmin)
}

func TestRBACService_GetRolePermissions(t *testing.T) {
	db := setupRBACTestDB(t)
	rbacService, err := NewRBACService(db, "")
	require.NoError(t, err)

	t.Run("valid role", func(t *testing.T) {
		perms, err := rbacService.GetRolePermissions(RoleDeveloper)

		assert.NoError(t, err)
		assert.NotEmpty(t, perms)
	})

	t.Run("invalid role", func(t *testing.T) {
		perms, err := rbacService.GetRolePermissions(Role("invalid"))

		assert.Error(t, err)
		assert.Equal(t, ErrRoleNotFound, err)
		assert.Nil(t, perms)
	})
}
