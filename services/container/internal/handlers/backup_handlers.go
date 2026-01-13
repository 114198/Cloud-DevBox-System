// Package handlers provides HTTP request handlers for the container service.
package handlers

import (
	"net/http"
	"strconv"

	"github.com/cloud-devbox/services/container/internal/models"
	"github.com/cloud-devbox/services/container/internal/services"
	"github.com/gin-gonic/gin"
)

// BackupHandler handles backup-related HTTP requests
type BackupHandler struct {
	codeBackupSvc     *services.CodeBackupService
	configBackupSvc   *services.ConfigBackupService
	databaseBackupSvc *services.DatabaseBackupService
	drSvc             *services.DisasterRecoveryService
}

// NewBackupHandler creates a new backup handler
func NewBackupHandler(
	codeBackupSvc *services.CodeBackupService,
	configBackupSvc *services.ConfigBackupService,
	databaseBackupSvc *services.DatabaseBackupService,
	drSvc *services.DisasterRecoveryService,
) *BackupHandler {
	return &BackupHandler{
		codeBackupSvc:     codeBackupSvc,
		configBackupSvc:   configBackupSvc,
		databaseBackupSvc: databaseBackupSvc,
		drSvc:             drSvc,
	}
}

// getUserID extracts user ID from context (in production, from JWT)
func getUserID(c *gin.Context) string {
	userID := c.GetHeader("X-User-ID")
	if userID == "" {
		userID = "anonymous"
	}
	return userID
}

// CreateCodeBackup creates a new code backup
func (h *BackupHandler) CreateCodeBackup(c *gin.Context) {
	var req models.CreateBackupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := getUserID(c)
	backup, err := h.codeBackupSvc.CreateBackup(c.Request.Context(), userID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, backup)
}

// GetCodeBackup retrieves a code backup by ID
func (h *BackupHandler) GetCodeBackup(c *gin.Context) {
	id := c.Param("id")
	userID := getUserID(c)

	backup, err := h.codeBackupSvc.GetBackup(c.Request.Context(), userID, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, backup)
}

// ListCodeBackups lists code backups
func (h *BackupHandler) ListCodeBackups(c *gin.Context) {
	var req models.ListBackupsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Page == 0 {
		req.Page = 1
	}
	if req.PageSize == 0 {
		req.PageSize = 20
	}

	req.UserID = getUserID(c)
	resp, err := h.codeBackupSvc.ListBackups(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// DeleteCodeBackup deletes a code backup
func (h *BackupHandler) DeleteCodeBackup(c *gin.Context) {
	id := c.Param("id")
	userID := getUserID(c)

	if err := h.codeBackupSvc.DeleteBackup(c.Request.Context(), userID, id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

// GetCodeBackupStats retrieves code backup statistics
func (h *BackupHandler) GetCodeBackupStats(c *gin.Context) {
	userID := getUserID(c)

	stats, err := h.codeBackupSvc.GetStats(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// SaveConfig saves a configuration version
func (h *BackupHandler) SaveConfig(c *gin.Context) {
	environmentID := c.Param("environmentId")
	userID := getUserID(c)

	var req struct {
		Config  map[string]interface{} `json:"config" binding:"required"`
		Message string                 `json:"message"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	version, err := h.configBackupSvc.SaveConfig(c.Request.Context(), userID, environmentID, req.Config, req.Message)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, version)
}

// GetConfigVersion retrieves a specific config version
func (h *BackupHandler) GetConfigVersion(c *gin.Context) {
	environmentID := c.Param("environmentId")
	versionStr := c.Param("version")
	userID := getUserID(c)

	versionNum, err := strconv.Atoi(versionStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid version number"})
		return
	}

	version, err := h.configBackupSvc.GetVersion(c.Request.Context(), userID, environmentID, versionNum)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, version)
}

// GetLatestConfigVersion retrieves the latest config version
func (h *BackupHandler) GetLatestConfigVersion(c *gin.Context) {
	environmentID := c.Param("environmentId")
	userID := getUserID(c)

	version, err := h.configBackupSvc.GetLatestVersion(c.Request.Context(), userID, environmentID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, version)
}

// ListConfigVersions lists config versions for an environment
func (h *BackupHandler) ListConfigVersions(c *gin.Context) {
	environmentID := c.Param("environmentId")
	userID := getUserID(c)

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))

	versions, total, err := h.configBackupSvc.ListVersions(c.Request.Context(), userID, environmentID, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"versions": versions,
		"total":    total,
		"page":     page,
		"pageSize": pageSize,
	})
}

// CompareConfigVersions compares two config versions
func (h *BackupHandler) CompareConfigVersions(c *gin.Context) {
	environmentID := c.Param("environmentId")
	userID := getUserID(c)

	fromVersion, err := strconv.Atoi(c.Query("from"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid from version"})
		return
	}

	toVersion, err := strconv.Atoi(c.Query("to"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid to version"})
		return
	}

	diff, err := h.configBackupSvc.CompareVersions(c.Request.Context(), userID, environmentID, fromVersion, toVersion)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, diff)
}

// RestoreConfigVersion restores a config to a specific version
func (h *BackupHandler) RestoreConfigVersion(c *gin.Context) {
	environmentID := c.Param("environmentId")
	versionStr := c.Param("version")
	userID := getUserID(c)

	versionNum, err := strconv.Atoi(versionStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid version number"})
		return
	}

	version, err := h.configBackupSvc.RestoreVersion(c.Request.Context(), userID, environmentID, versionNum)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, version)
}

// GetConfigBackupStats retrieves config backup statistics
func (h *BackupHandler) GetConfigBackupStats(c *gin.Context) {
	userID := getUserID(c)

	stats, err := h.configBackupSvc.GetStats(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, stats)
}


// CreateDatabaseBackup creates a database backup
func (h *BackupHandler) CreateDatabaseBackup(c *gin.Context) {
	userID := getUserID(c)

	var req struct {
		Mode string `json:"mode"` // "full" or "incremental"
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		req.Mode = "incremental"
	}

	var backup *models.Backup
	var err error

	if req.Mode == "full" {
		backup, err = h.databaseBackupSvc.CreateFullBackup(c.Request.Context(), userID)
	} else {
		backup, err = h.databaseBackupSvc.CreateIncrementalBackup(c.Request.Context(), userID)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, backup)
}

// GetDatabaseBackup retrieves a database backup by ID
func (h *BackupHandler) GetDatabaseBackup(c *gin.Context) {
	id := c.Param("id")

	backup, err := h.databaseBackupSvc.GetBackup(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, backup)
}

// ListDatabaseBackups lists database backups
func (h *BackupHandler) ListDatabaseBackups(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))

	resp, err := h.databaseBackupSvc.ListBackups(c.Request.Context(), page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// DeleteDatabaseBackup deletes a database backup
func (h *BackupHandler) DeleteDatabaseBackup(c *gin.Context) {
	id := c.Param("id")

	if err := h.databaseBackupSvc.DeleteBackup(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

// RestoreDatabaseBackup restores a database from backup
func (h *BackupHandler) RestoreDatabaseBackup(c *gin.Context) {
	id := c.Param("id")

	if err := h.databaseBackupSvc.RestoreBackup(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "restore initiated"})
}

// VerifyDatabaseBackup verifies a database backup integrity
func (h *BackupHandler) VerifyDatabaseBackup(c *gin.Context) {
	id := c.Param("id")

	valid, err := h.databaseBackupSvc.VerifyBackup(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "valid": false})
		return
	}

	c.JSON(http.StatusOK, gin.H{"valid": valid})
}

// GetDatabaseBackupStats retrieves database backup statistics
func (h *BackupHandler) GetDatabaseBackupStats(c *gin.Context) {
	stats, err := h.databaseBackupSvc.GetStats(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// InitiateFailover initiates a disaster recovery failover
func (h *BackupHandler) InitiateFailover(c *gin.Context) {
	var req struct {
		TargetRegion string `json:"targetRegion" binding:"required"`
		Reason       string `json:"reason" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	record, err := h.drSvc.InitiateFailover(c.Request.Context(), req.TargetRegion, req.Reason)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusAccepted, record)
}

// GetFailoverStatus retrieves the status of a failover operation
func (h *BackupHandler) GetFailoverStatus(c *gin.Context) {
	id := c.Param("id")

	record, err := h.drSvc.GetFailoverStatus(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, record)
}

// ListFailovers lists all failover records
func (h *BackupHandler) ListFailovers(c *gin.Context) {
	records, err := h.drSvc.ListFailovers(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"failovers": records})
}

// GetReplicationStatus retrieves the status of a replication operation
func (h *BackupHandler) GetReplicationStatus(c *gin.Context) {
	id := c.Param("id")

	record, err := h.drSvc.GetReplicationStatus(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, record)
}

// ListReplications lists all replication records
func (h *BackupHandler) ListReplications(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))

	records, total, err := h.drSvc.ListReplications(c.Request.Context(), page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"replications": records,
		"total":        total,
		"page":         page,
		"pageSize":     pageSize,
	})
}

// GetRegionHealth retrieves the health status of all regions
func (h *BackupHandler) GetRegionHealth(c *gin.Context) {
	health := h.drSvc.GetRegionHealth(c.Request.Context())
	c.JSON(http.StatusOK, gin.H{"regions": health})
}

// GetDRStatus retrieves the overall disaster recovery status
func (h *BackupHandler) GetDRStatus(c *gin.Context) {
	status := h.drSvc.GetDRStatus(c.Request.Context())
	c.JSON(http.StatusOK, status)
}

// GetRecoveryObjectives retrieves the current RTO/RPO objectives
func (h *BackupHandler) GetRecoveryObjectives(c *gin.Context) {
	objectives := h.drSvc.GetRecoveryObjectives()
	c.JSON(http.StatusOK, objectives)
}

// RegisterBackupRoutes registers backup-related routes
func RegisterBackupRoutes(router *gin.RouterGroup, handler *BackupHandler) {
	// Code backup routes
	codeBackup := router.Group("/backups/code")
	{
		codeBackup.POST("", handler.CreateCodeBackup)
		codeBackup.GET("", handler.ListCodeBackups)
		codeBackup.GET("/stats", handler.GetCodeBackupStats)
		codeBackup.GET("/:id", handler.GetCodeBackup)
		codeBackup.DELETE("/:id", handler.DeleteCodeBackup)
	}

	// Config backup routes
	configBackup := router.Group("/environments/:environmentId/config")
	{
		configBackup.POST("/versions", handler.SaveConfig)
		configBackup.GET("/versions", handler.ListConfigVersions)
		configBackup.GET("/versions/latest", handler.GetLatestConfigVersion)
		configBackup.GET("/versions/:version", handler.GetConfigVersion)
		configBackup.POST("/versions/:version/restore", handler.RestoreConfigVersion)
		configBackup.GET("/compare", handler.CompareConfigVersions)
	}
	router.GET("/backups/config/stats", handler.GetConfigBackupStats)

	// Database backup routes
	dbBackup := router.Group("/backups/database")
	{
		dbBackup.POST("", handler.CreateDatabaseBackup)
		dbBackup.GET("", handler.ListDatabaseBackups)
		dbBackup.GET("/stats", handler.GetDatabaseBackupStats)
		dbBackup.GET("/:id", handler.GetDatabaseBackup)
		dbBackup.DELETE("/:id", handler.DeleteDatabaseBackup)
		dbBackup.POST("/:id/restore", handler.RestoreDatabaseBackup)
		dbBackup.GET("/:id/verify", handler.VerifyDatabaseBackup)
	}

	// Disaster recovery routes
	dr := router.Group("/disaster-recovery")
	{
		dr.GET("/status", handler.GetDRStatus)
		dr.GET("/objectives", handler.GetRecoveryObjectives)
		dr.GET("/regions/health", handler.GetRegionHealth)
		dr.POST("/failover", handler.InitiateFailover)
		dr.GET("/failover", handler.ListFailovers)
		dr.GET("/failover/:id", handler.GetFailoverStatus)
		dr.GET("/replications", handler.ListReplications)
		dr.GET("/replications/:id", handler.GetReplicationStatus)
	}
}
