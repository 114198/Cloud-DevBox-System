// Package models provides data models for the core service.
package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/gorm"
)

// Template represents a development environment template
type Template struct {
	ID            uuid.UUID      `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	Name          string         `gorm:"uniqueIndex;not null" json:"name"`
	DisplayName   string         `gorm:"not null" json:"display_name"`
	Description   string         `json:"description,omitempty"`
	Category      string         `gorm:"not null" json:"category"`
	Tags          pq.StringArray `gorm:"type:text[]" json:"tags"`
	Icon          string         `json:"icon,omitempty"`
	Runtime       JSONMap        `gorm:"type:jsonb;not null" json:"runtime"`
	DefaultConfig JSONMap        `gorm:"type:jsonb;not null" json:"default_config"`
	Dockerfile    string         `gorm:"not null" json:"dockerfile"`
	InitScript    string         `json:"init_script,omitempty"`
	Extensions    JSONArray      `gorm:"type:jsonb;default:'[]'" json:"extensions"`
	IsPublic      bool           `gorm:"default:true" json:"is_public"`
	CreatedBy     uuid.UUID      `gorm:"type:uuid;not null" json:"created_by"`
	Version       string         `gorm:"not null;default:'1.0.0'" json:"version"`
	ParentID      *uuid.UUID     `gorm:"type:uuid" json:"parent_id,omitempty"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`

	// Relations
	Creator  *User              `gorm:"foreignKey:CreatedBy" json:"creator,omitempty"`
	Parent   *Template          `gorm:"foreignKey:ParentID" json:"parent,omitempty"`
	Versions []TemplateVersion  `gorm:"foreignKey:TemplateID" json:"versions,omitempty"`
}

// TableName returns the table name for Template
func (Template) TableName() string {
	return "templates"
}

// BeforeCreate sets default values before creating a template
func (t *Template) BeforeCreate(tx *gorm.DB) error {
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}
	if t.Version == "" {
		t.Version = "1.0.0"
	}
	if t.Tags == nil {
		t.Tags = pq.StringArray{}
	}
	if t.Extensions == nil {
		t.Extensions = JSONArray{}
	}
	return nil
}

// TemplateVersion represents a version history entry for a template
type TemplateVersion struct {
	ID            uuid.UUID `gorm:"type:uuid;primary_key;default:uuid_generate_v4()" json:"id"`
	TemplateID    uuid.UUID `gorm:"type:uuid;not null" json:"template_id"`
	Version       string    `gorm:"not null" json:"version"`
	Runtime       JSONMap   `gorm:"type:jsonb;not null" json:"runtime"`
	DefaultConfig JSONMap   `gorm:"type:jsonb;not null" json:"default_config"`
	Dockerfile    string    `gorm:"not null" json:"dockerfile"`
	InitScript    string    `json:"init_script,omitempty"`
	Extensions    JSONArray `gorm:"type:jsonb;default:'[]'" json:"extensions"`
	ChangeLog     string    `json:"change_log,omitempty"`
	CreatedBy     uuid.UUID `gorm:"type:uuid;not null" json:"created_by"`
	CreatedAt     time.Time `json:"created_at"`

	// Relations
	Template *Template `gorm:"foreignKey:TemplateID" json:"template,omitempty"`
	Creator  *User     `gorm:"foreignKey:CreatedBy" json:"creator,omitempty"`
}

// TableName returns the table name for TemplateVersion
func (TemplateVersion) TableName() string {
	return "template_versions"
}

// BeforeCreate sets default values before creating a template version
func (tv *TemplateVersion) BeforeCreate(tx *gorm.DB) error {
	if tv.ID == uuid.Nil {
		tv.ID = uuid.New()
	}
	if tv.Extensions == nil {
		tv.Extensions = JSONArray{}
	}
	return nil
}

// TemplateCategory represents available template categories
type TemplateCategory string

const (
	CategoryFrontend  TemplateCategory = "前端"
	CategoryBackend   TemplateCategory = "后端"
	CategoryFullStack TemplateCategory = "全栈"
	CategoryDatabase  TemplateCategory = "数据库"
	CategoryDevOps    TemplateCategory = "DevOps"
	CategoryOther     TemplateCategory = "其他"
)

// ValidCategories returns all valid template categories
func ValidCategories() []TemplateCategory {
	return []TemplateCategory{
		CategoryFrontend,
		CategoryBackend,
		CategoryFullStack,
		CategoryDatabase,
		CategoryDevOps,
		CategoryOther,
	}
}

// IsValidCategory checks if a category is valid
func IsValidCategory(category string) bool {
	for _, c := range ValidCategories() {
		if string(c) == category {
			return true
		}
	}
	return false
}
