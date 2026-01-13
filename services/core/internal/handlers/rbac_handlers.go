// Package handlers provides HTTP request handlers for the core service.
package handlers

import (
	"net/http"

	"github.com/cloud-devbox/services/core/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// RBACHandler handles RBAC-related HTTP requests
type RBACHandler struct {
	rbacService *services.RBACService
}

// NewRBACHandler creates a new RBACHandler instance
func NewRBACHandler(rbacService *services.RBACService) *RBACHandler {
	return &RBACHandler{
		rbacService: rbacService,
	}
}

// PermissionResponse represents a permission in the response
type PermissionResponse struct {
	Resource string `json:"resource"`
	Action   string `json:"action"`
}

// UserPermissionsResponse represents the user permissions response
type UserPermissionsResponse struct {
	Role        string               `json:"role"`
	Permissions []PermissionResponse `json:"permissions"`
}

// GetMyPermissions returns the current user's permissions
func (h *RBACHandler) GetMyPermissions(c *gin.Context) {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "User not authenticated",
		})
		return
	}

	perms, err := h.rbacService.GetUserPermissions(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "fetch_failed",
			Message: err.Error(),
		})
		return
	}

	response := UserPermissionsResponse{
		Role:        string(perms.Role),
		Permissions: make([]PermissionResponse, len(perms.Permissions)),
	}

	for i, p := range perms.Permissions {
		response.Permissions[i] = PermissionResponse{
			Resource: string(p.Resource),
			Action:   string(p.Action),
		}
	}

	c.JSON(http.StatusOK, response)
}

// UpdateUserRoleRequest represents the request to update a user's role
type UpdateUserRoleRequest struct {
	Role string `json:"role" binding:"required,oneof=developer admin org_admin"`
}

// UpdateUserRole updates a user's role (admin only)
func (h *RBACHandler) UpdateUserRole(c *gin.Context) {
	// Get target user ID from path
	targetUserIDStr := c.Param("id")
	targetUserID, err := uuid.Parse(targetUserIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_user_id",
			Message: "Invalid user ID",
		})
		return
	}

	var req UpdateUserRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_request",
			Message: err.Error(),
		})
		return
	}

	if err := h.rbacService.SetUserRole(c.Request.Context(), targetUserID, services.Role(req.Role)); err != nil {
		switch err {
		case services.ErrRoleNotFound:
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:   "invalid_role",
				Message: "Invalid role specified",
			})
		case services.ErrUserNotFound:
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "user_not_found",
				Message: "User not found",
			})
		default:
			c.JSON(http.StatusInternalServerError, ErrorResponse{
				Error:   "update_failed",
				Message: err.Error(),
			})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User role updated successfully"})
}

// GetRoles returns all available roles
func (h *RBACHandler) GetRoles(c *gin.Context) {
	roles := h.rbacService.GetAllRoles()
	
	response := make([]gin.H, len(roles))
	for i, role := range roles {
		perms, _ := h.rbacService.GetRolePermissions(role)
		permResponses := make([]PermissionResponse, len(perms))
		for j, p := range perms {
			permResponses[j] = PermissionResponse{
				Resource: string(p.Resource),
				Action:   string(p.Action),
			}
		}
		response[i] = gin.H{
			"name":        string(role),
			"permissions": permResponses,
		}
	}

	c.JSON(http.StatusOK, gin.H{"roles": response})
}

// RequirePermission creates a middleware that checks for a specific permission
func (h *RBACHandler) RequirePermission(resource services.Resource, action services.Action) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, err := getUserIDFromContext(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, ErrorResponse{
				Error:   "unauthorized",
				Message: "User not authenticated",
			})
			c.Abort()
			return
		}

		if err := h.rbacService.CheckPermission(c.Request.Context(), userID, resource, action); err != nil {
			if err == services.ErrPermissionDenied {
				c.JSON(http.StatusForbidden, ErrorResponse{
					Error:   "permission_denied",
					Message: "You don't have permission to perform this action",
				})
			} else {
				c.JSON(http.StatusInternalServerError, ErrorResponse{
					Error:   "permission_check_failed",
					Message: err.Error(),
				})
			}
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireResourcePermission creates a middleware that checks for permission on a specific resource
func (h *RBACHandler) RequireResourcePermission(resource services.Resource, action services.Action, resourceIDParam string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, err := getUserIDFromContext(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, ErrorResponse{
				Error:   "unauthorized",
				Message: "User not authenticated",
			})
			c.Abort()
			return
		}

		resourceIDStr := c.Param(resourceIDParam)
		resourceID, err := uuid.Parse(resourceIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:   "invalid_resource_id",
				Message: "Invalid resource ID",
			})
			c.Abort()
			return
		}

		if err := h.rbacService.CheckResourcePermission(c.Request.Context(), userID, resource, resourceID, action); err != nil {
			if err == services.ErrPermissionDenied {
				c.JSON(http.StatusForbidden, ErrorResponse{
					Error:   "permission_denied",
					Message: "You don't have permission to access this resource",
				})
			} else {
				c.JSON(http.StatusInternalServerError, ErrorResponse{
					Error:   "permission_check_failed",
					Message: err.Error(),
				})
			}
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireRole creates a middleware that checks for a specific role
func (h *RBACHandler) RequireRole(roles ...services.Role) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, err := getUserIDFromContext(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, ErrorResponse{
				Error:   "unauthorized",
				Message: "User not authenticated",
			})
			c.Abort()
			return
		}

		userRole, err := h.rbacService.GetUserRole(c.Request.Context(), userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, ErrorResponse{
				Error:   "role_check_failed",
				Message: err.Error(),
			})
			c.Abort()
			return
		}

		// Check if user has one of the required roles
		hasRole := false
		for _, role := range roles {
			if userRole == role {
				hasRole = true
				break
			}
		}

		// Admin always has access
		if userRole == services.RoleAdmin {
			hasRole = true
		}

		if !hasRole {
			c.JSON(http.StatusForbidden, ErrorResponse{
				Error:   "insufficient_role",
				Message: "You don't have the required role to perform this action",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
