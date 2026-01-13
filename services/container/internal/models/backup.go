// Package models defines the data models for the container service.
package models

import (
	"time"
)

// BackupType represents the type of backup
type BackupType string

const (
	BackupTypeCode     BackupType = "code"
	BackupTypeConfig   BackupType = "config"
	BackupTypeDatabase BackupType = "database"
	BackupTypeFull     BackupType = "full"
)

// BackupStatus represents the status of a backup
type BackupStatus string

const (
	BackupStatusPending    BackupStatus = "pending"
	BackupStatusInProgress BackupStatus = "in_progress"
	BackupStatusCompleted  BackupStatus = "completed"
	BackupStatusFailed     BackupStatus = "failed"
)

// BackupMode represents the backup mode
type BackupMode string

const (
	BackupModeFull        BackupMode = "full"
	BackupModeIncremental BackupMode = "incremental"
)

// Backup represents a backup record
type Backup struct {
	ID            string       `json:"id"`
	EnvironmentID string       `json:"environmentId"`
	UserID        string       `json:"userId"`
	Type          BackupType   `json:"type"`
	Mode          BackupMode   `json:"mode"`
	Status        BackupStatus `json:"status"`
	Size          int64        `json:"size"`          // Size in bytes
	StoragePath   string       `json:"storagePath"`   // Path in storage (S3/local)
	StorageRegion string       `json:"storageRegion"` // Storage region for cross-region sync
	Checksum      string       `json:"checksum"`      // SHA256 checksum
	ParentID      string       `json:"parentId,omitempty"` // Parent backup ID for incremental
	Message       string       `json:"message,omitempty"`
	Metadata      BackupMetadata `json:"metadata"`
	CreatedAt     time.Time    `json:"createdAt"`
	CompletedAt   *time.Time   `json:"completedAt,omitempty"`
	ExpiresAt     *time.Time   `json:"expiresAt,omitempty"`
}

// BackupMetadata contains additional backup information
type BackupMetadata struct {
	FileCount     int64             `json:"fileCount,omitempty"`
	TotalSize     int64             `json:"totalSize,omitempty"`
	ChangedFiles  int64             `json:"changedFiles,omitempty"`
	DeletedFiles  int64             `json:"deletedFiles,omitempty"`
	SourcePath    string            `json:"sourcePath,omitempty"`
	Compression   string            `json:"compression,omitempty"`
	Encryption    string            `json:"encryption,omitempty"`
	Labels        map[string]string `json:"labels,omitempty"`
}


// BackupSchedule represents a backup schedule configuration
type BackupSchedule struct {
	ID            string     `json:"id"`
	EnvironmentID string     `json:"environmentId"`
	UserID        string     `json:"userId"`
	Type          BackupType `json:"type"`
	Mode          BackupMode `json:"mode"`
	CronExpr      string     `json:"cronExpr"`      // Cron expression for scheduling
	Enabled       bool       `json:"enabled"`
	RetentionDays int        `json:"retentionDays"` // How long to keep backups
	LastRunAt     *time.Time `json:"lastRunAt,omitempty"`
	NextRunAt     *time.Time `json:"nextRunAt,omitempty"`
	CreatedAt     time.Time  `json:"createdAt"`
	UpdatedAt     time.Time  `json:"updatedAt"`
}

// BackupPolicy defines backup retention and scheduling policies
type BackupPolicy struct {
	// Code backup settings
	CodeBackupInterval    time.Duration `json:"codeBackupInterval"`    // Default: 1 hour
	CodeRetentionDays     int           `json:"codeRetentionDays"`     // Default: 30 days
	
	// Config backup settings
	ConfigBackupRealtime  bool          `json:"configBackupRealtime"`  // Default: true
	ConfigRetentionDays   int           `json:"configRetentionDays"`   // Default: 30 days
	ConfigVersionLimit    int           `json:"configVersionLimit"`    // Max versions to keep
	
	// Database backup settings
	DBFullBackupInterval  time.Duration `json:"dbFullBackupInterval"`  // Default: 24 hours
	DBIncrBackupInterval  time.Duration `json:"dbIncrBackupInterval"`  // Default: 1 hour
	DBRetentionDays       int           `json:"dbRetentionDays"`       // Default: 30 days
	
	// Cross-region settings
	CrossRegionEnabled    bool          `json:"crossRegionEnabled"`
	TargetRegions         []string      `json:"targetRegions"`
}

// DefaultBackupPolicy returns the default backup policy
func DefaultBackupPolicy() *BackupPolicy {
	return &BackupPolicy{
		CodeBackupInterval:   time.Hour,
		CodeRetentionDays:    30,
		ConfigBackupRealtime: true,
		ConfigRetentionDays:  30,
		ConfigVersionLimit:   100,
		DBFullBackupInterval: 24 * time.Hour,
		DBIncrBackupInterval: time.Hour,
		DBRetentionDays:      30,
		CrossRegionEnabled:   false,
		TargetRegions:        []string{},
	}
}

// RestoreRequest represents a request to restore from backup
type RestoreRequest struct {
	BackupID      string `json:"backupId" binding:"required"`
	EnvironmentID string `json:"environmentId" binding:"required"`
	TargetPath    string `json:"targetPath,omitempty"`
	Overwrite     bool   `json:"overwrite"`
}

// RestoreStatus represents the status of a restore operation
type RestoreStatus string

const (
	RestoreStatusPending    RestoreStatus = "pending"
	RestoreStatusInProgress RestoreStatus = "in_progress"
	RestoreStatusCompleted  RestoreStatus = "completed"
	RestoreStatusFailed     RestoreStatus = "failed"
)

// RestoreRecord represents a restore operation record
type RestoreRecord struct {
	ID            string        `json:"id"`
	BackupID      string        `json:"backupId"`
	EnvironmentID string        `json:"environmentId"`
	UserID        string        `json:"userId"`
	Status        RestoreStatus `json:"status"`
	Message       string        `json:"message,omitempty"`
	RestoredFiles int64         `json:"restoredFiles"`
	RestoredSize  int64         `json:"restoredSize"`
	StartedAt     time.Time     `json:"startedAt"`
	CompletedAt   *time.Time    `json:"completedAt,omitempty"`
}

// ListBackupsRequest represents a request to list backups
type ListBackupsRequest struct {
	EnvironmentID string     `form:"environmentId"`
	UserID        string     `form:"userId"`
	Type          BackupType `form:"type"`
	Status        BackupStatus `form:"status"`
	Page          int        `form:"page,default=1"`
	PageSize      int        `form:"pageSize,default=20"`
}

// ListBackupsResponse represents a response containing a list of backups
type ListBackupsResponse struct {
	Backups  []Backup `json:"backups"`
	Total    int64    `json:"total"`
	Page     int      `json:"page"`
	PageSize int      `json:"pageSize"`
}

// CreateBackupRequest represents a request to create a backup
type CreateBackupRequest struct {
	EnvironmentID string     `json:"environmentId" binding:"required"`
	Type          BackupType `json:"type" binding:"required"`
	Mode          BackupMode `json:"mode,omitempty"`
	Description   string     `json:"description,omitempty"`
}

// BackupStats represents backup statistics
type BackupStats struct {
	TotalBackups      int64 `json:"totalBackups"`
	TotalSize         int64 `json:"totalSize"`
	CodeBackups       int64 `json:"codeBackups"`
	ConfigBackups     int64 `json:"configBackups"`
	DatabaseBackups   int64 `json:"databaseBackups"`
	LastBackupAt      *time.Time `json:"lastBackupAt,omitempty"`
	OldestBackupAt    *time.Time `json:"oldestBackupAt,omitempty"`
}
