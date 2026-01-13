// Package services provides business logic services for the core service.
package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

var (
	ErrPermissionDenied    = errors.New("permission denied")
	ErrRoleNotFound        = errors.New("role not found")
	ErrInvalidResource     = errors.New("invalid resource")
	ErrInvalidAction       = errors.New("invalid action")
)

// Role represents a user role in the system
type Role string

const (
	RoleDeveloper Role = "developer"
	RoleAdmin     Role = "admin"
	RoleOrgAdmin  Role = "org_admin"
)

// Resource represents a resource type in the system
type Resource string

const (
	ResourceEnvironment   Resource = "environment"
	ResourceTemplate      Resource = "template"
	ResourceProject       Resource = "project"
	ResourceUser          Resource = "user"
	ResourceOrganization  Resource = "organization"
	ResourceBilling       Resource = "billing"
	ResourceAuditLog      Resource = "audit_log"
)

// Action represents an action that can be performed on a resource
type Action string

const (
	ActionCreate Action = "create"
	ActionRead   Action = "read"
	ActionUpdate Action = "update"
	ActionDelete Action = "delete"
	ActionList   Action = "list"
	ActionManage Action = "manage" // Full control
)

// Permission represents a permission entry
type Permission struct {
	Resource Resource `json:"resource"`
	Action   Action   `json:"action"`
}

// RBACService handles role-based access control
type RBACService struct {
	db          *gorm.DB
	redis       *redis.Client
	cacheTTL    time.Duration
	permissions map[Role][]Permission
}

// NewRBACService creates a new RBACService instance
func NewRBACService(db *gorm.DB, redisURL string) (*RBACService, error) {
	// Parse Redis URL and create client
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		// If Redis is not available, continue without caching
		return &RBACService{
			db:          db,
			redis:       nil,
			cacheTTL:    5 * time.Minute,
			permissions: initializePermissions(),
		}, nil
	}

	client := redis.NewClient(opt)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		// Redis not available, continue without caching
		return &RBACService{
			db:          db,
			redis:       nil,
			cacheTTL:    5 * time.Minute,
			permissions: initializePermissions(),
		}, nil
	}

	return &RBACService{
		db:          db,
		redis:       client,
		cacheTTL:    5 * time.Minute,
		permissions: initializePermissions(),
	}, nil
}

// initializePermissions sets up the default role permissions
func initializePermissions() map[Role][]Permission {
	return map[Role][]Permission{
		RoleDeveloper: {
			// Environment permissions
			{Resource: ResourceEnvironment, Action: ActionCreate},
			{Resource: ResourceEnvironment, Action: ActionRead},
			{Resource: ResourceEnvironment, Action: ActionUpdate},
			{Resource: ResourceEnvironment, Action: ActionDelete},
			{Resource: ResourceEnvironment, Action: ActionList},
			// Template permissions (read-only)
			{Resource: ResourceTemplate, Action: ActionRead},
			{Resource: ResourceTemplate, Action: ActionList},
			// Project permissions
			{Resource: ResourceProject, Action: ActionCreate},
			{Resource: ResourceProject, Action: ActionRead},
			{Resource: ResourceProject, Action: ActionUpdate},
			{Resource: ResourceProject, Action: ActionDelete},
			{Resource: ResourceProject, Action: ActionList},
			// User permissions (self only)
			{Resource: ResourceUser, Action: ActionRead},
			{Resource: ResourceUser, Action: ActionUpdate},
		},
		RoleOrgAdmin: {
			// All developer permissions plus...
			{Resource: ResourceEnvironment, Action: ActionManage},
			{Resource: ResourceTemplate, Action: ActionCreate},
			{Resource: ResourceTemplate, Action: ActionUpdate},
			{Resource: ResourceTemplate, Action: ActionDelete},
			{Resource: ResourceTemplate, Action: ActionManage},
			{Resource: ResourceProject, Action: ActionManage},
			{Resource: ResourceUser, Action: ActionList},
			{Resource: ResourceUser, Action: ActionManage},
			{Resource: ResourceOrganization, Action: ActionRead},
			{Resource: ResourceOrganization, Action: ActionUpdate},
			{Resource: ResourceBilling, Action: ActionRead},
			{Resource: ResourceBilling, Action: ActionManage},
		},
		RoleAdmin: {
			// Full system access
			{Resource: ResourceEnvironment, Action: ActionManage},
			{Resource: ResourceTemplate, Action: ActionManage},
			{Resource: ResourceProject, Action: ActionManage},
			{Resource: ResourceUser, Action: ActionManage},
			{Resource: ResourceOrganization, Action: ActionManage},
			{Resource: ResourceBilling, Action: ActionManage},
			{Resource: ResourceAuditLog, Action: ActionRead},
			{Resource: ResourceAuditLog, Action: ActionList},
		},
	}
}


// HasPermission checks if a role has permission to perform an action on a resource
func (s *RBACService) HasPermission(role Role, resource Resource, action Action) bool {
	permissions, ok := s.permissions[role]
	if !ok {
		return false
	}

	for _, p := range permissions {
		// Check for exact match or manage permission
		if p.Resource == resource && (p.Action == action || p.Action == ActionManage) {
			return true
		}
	}

	// Admin has all permissions
	if role == RoleAdmin {
		return true
	}

	// OrgAdmin inherits developer permissions
	if role == RoleOrgAdmin {
		return s.HasPermission(RoleDeveloper, resource, action)
	}

	return false
}

// CheckPermission checks if a user has permission and returns an error if not
func (s *RBACService) CheckPermission(ctx context.Context, userID uuid.UUID, resource Resource, action Action) error {
	role, err := s.GetUserRole(ctx, userID)
	if err != nil {
		return err
	}

	if !s.HasPermission(role, resource, action) {
		return ErrPermissionDenied
	}

	return nil
}

// CheckResourcePermission checks if a user has permission on a specific resource instance
func (s *RBACService) CheckResourcePermission(ctx context.Context, userID uuid.UUID, resource Resource, resourceID uuid.UUID, action Action) error {
	role, err := s.GetUserRole(ctx, userID)
	if err != nil {
		return err
	}

	// Admin has all permissions
	if role == RoleAdmin {
		return nil
	}

	// Check basic permission first
	if !s.HasPermission(role, resource, action) {
		return ErrPermissionDenied
	}

	// For non-admin users, check resource ownership
	isOwner, err := s.isResourceOwner(ctx, userID, resource, resourceID)
	if err != nil {
		return err
	}

	if !isOwner {
		// Check if user has org-level access
		hasOrgAccess, err := s.hasOrganizationAccess(ctx, userID, resource, resourceID)
		if err != nil {
			return err
		}
		if !hasOrgAccess {
			return ErrPermissionDenied
		}
	}

	return nil
}

// GetUserRole retrieves the role for a user (with caching)
func (s *RBACService) GetUserRole(ctx context.Context, userID uuid.UUID) (Role, error) {
	cacheKey := fmt.Sprintf("user:role:%s", userID.String())

	// Try cache first
	if s.redis != nil {
		cached, err := s.redis.Get(ctx, cacheKey).Result()
		if err == nil {
			return Role(cached), nil
		}
	}

	// Query database
	var user struct {
		Role string
	}
	if err := s.db.Table("users").Select("role").Where("id = ?", userID).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", ErrUserNotFound
		}
		return "", err
	}

	role := Role(user.Role)

	// Cache the result
	if s.redis != nil {
		s.redis.Set(ctx, cacheKey, string(role), s.cacheTTL)
	}

	return role, nil
}

// SetUserRole updates a user's role
func (s *RBACService) SetUserRole(ctx context.Context, userID uuid.UUID, role Role) error {
	// Validate role
	if _, ok := s.permissions[role]; !ok {
		return ErrRoleNotFound
	}

	// Update database
	if err := s.db.Table("users").Where("id = ?", userID).Update("role", string(role)).Error; err != nil {
		return err
	}

	// Invalidate cache
	if s.redis != nil {
		cacheKey := fmt.Sprintf("user:role:%s", userID.String())
		s.redis.Del(ctx, cacheKey)
	}

	return nil
}

// isResourceOwner checks if a user owns a specific resource
func (s *RBACService) isResourceOwner(ctx context.Context, userID uuid.UUID, resource Resource, resourceID uuid.UUID) (bool, error) {
	var count int64

	switch resource {
	case ResourceEnvironment:
		s.db.Table("environments").Where("id = ? AND user_id = ?", resourceID, userID).Count(&count)
	case ResourceProject:
		s.db.Table("projects").Where("id = ? AND owner_id = ?", resourceID, userID).Count(&count)
	case ResourceTemplate:
		s.db.Table("templates").Where("id = ? AND created_by = ?", resourceID, userID).Count(&count)
	case ResourceUser:
		// Users can only access their own data
		return resourceID == userID, nil
	default:
		return false, nil
	}

	return count > 0, nil
}

// hasOrganizationAccess checks if a user has organization-level access to a resource
func (s *RBACService) hasOrganizationAccess(ctx context.Context, userID uuid.UUID, resource Resource, resourceID uuid.UUID) (bool, error) {
	// Get user's organization
	var user struct {
		OrganizationID *uuid.UUID
		Role           string
	}
	if err := s.db.Table("users").Select("organization_id, role").Where("id = ?", userID).First(&user).Error; err != nil {
		return false, err
	}

	// If user is not in an organization, no org-level access
	if user.OrganizationID == nil {
		return false, nil
	}

	// Only org_admin can access org resources
	if user.Role != string(RoleOrgAdmin) {
		return false, nil
	}

	// Check if resource belongs to the same organization
	var count int64
	switch resource {
	case ResourceEnvironment:
		s.db.Table("environments e").
			Joins("JOIN users u ON e.user_id = u.id").
			Where("e.id = ? AND u.organization_id = ?", resourceID, user.OrganizationID).
			Count(&count)
	case ResourceProject:
		s.db.Table("projects").
			Where("id = ? AND organization_id = ?", resourceID, user.OrganizationID).
			Count(&count)
	case ResourceUser:
		s.db.Table("users").
			Where("id = ? AND organization_id = ?", resourceID, user.OrganizationID).
			Count(&count)
	default:
		return false, nil
	}

	return count > 0, nil
}


// UserPermissions represents cached user permissions
type UserPermissions struct {
	Role        Role         `json:"role"`
	Permissions []Permission `json:"permissions"`
	CachedAt    time.Time    `json:"cached_at"`
}

// GetUserPermissions retrieves all permissions for a user (with caching)
func (s *RBACService) GetUserPermissions(ctx context.Context, userID uuid.UUID) (*UserPermissions, error) {
	cacheKey := fmt.Sprintf("user:permissions:%s", userID.String())

	// Try cache first
	if s.redis != nil {
		cached, err := s.redis.Get(ctx, cacheKey).Result()
		if err == nil {
			var perms UserPermissions
			if json.Unmarshal([]byte(cached), &perms) == nil {
				return &perms, nil
			}
		}
	}

	// Get user role
	role, err := s.GetUserRole(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Build permissions list
	permissions := s.permissions[role]

	// For org_admin, include developer permissions
	if role == RoleOrgAdmin {
		permissions = append(permissions, s.permissions[RoleDeveloper]...)
	}

	// For admin, include all permissions
	if role == RoleAdmin {
		permissions = getAllPermissions()
	}

	userPerms := &UserPermissions{
		Role:        role,
		Permissions: permissions,
		CachedAt:    time.Now(),
	}

	// Cache the result
	if s.redis != nil {
		data, _ := json.Marshal(userPerms)
		s.redis.Set(ctx, cacheKey, string(data), s.cacheTTL)
	}

	return userPerms, nil
}

// getAllPermissions returns all possible permissions
func getAllPermissions() []Permission {
	resources := []Resource{
		ResourceEnvironment,
		ResourceTemplate,
		ResourceProject,
		ResourceUser,
		ResourceOrganization,
		ResourceBilling,
		ResourceAuditLog,
	}

	actions := []Action{
		ActionCreate,
		ActionRead,
		ActionUpdate,
		ActionDelete,
		ActionList,
		ActionManage,
	}

	var permissions []Permission
	for _, r := range resources {
		for _, a := range actions {
			permissions = append(permissions, Permission{Resource: r, Action: a})
		}
	}

	return permissions
}

// InvalidateUserPermissions invalidates cached permissions for a user
func (s *RBACService) InvalidateUserPermissions(ctx context.Context, userID uuid.UUID) error {
	if s.redis == nil {
		return nil
	}

	keys := []string{
		fmt.Sprintf("user:role:%s", userID.String()),
		fmt.Sprintf("user:permissions:%s", userID.String()),
	}

	return s.redis.Del(ctx, keys...).Err()
}

// GetRolePermissions returns all permissions for a role
func (s *RBACService) GetRolePermissions(role Role) ([]Permission, error) {
	permissions, ok := s.permissions[role]
	if !ok {
		return nil, ErrRoleNotFound
	}
	return permissions, nil
}

// GetAllRoles returns all available roles
func (s *RBACService) GetAllRoles() []Role {
	return []Role{RoleDeveloper, RoleOrgAdmin, RoleAdmin}
}

// Close closes the Redis connection
func (s *RBACService) Close() error {
	if s.redis != nil {
		return s.redis.Close()
	}
	return nil
}
