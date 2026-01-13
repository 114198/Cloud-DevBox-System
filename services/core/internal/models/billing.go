// Package models provides data models for the core service.
package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// ============================================================================
// Plan Models
// ============================================================================

// Plan represents a subscription plan
type Plan struct {
	ID           uuid.UUID       `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	Name         string          `json:"name" gorm:"type:varchar(50);unique;not null"`
	DisplayName  string          `json:"display_name" gorm:"type:varchar(100);not null"`
	Description  string          `json:"description,omitempty" gorm:"type:text"`
	PriceMonthly decimal.Decimal `json:"price_monthly" gorm:"type:decimal(10,2);default:0"`
	PriceYearly  decimal.Decimal `json:"price_yearly" gorm:"type:decimal(10,2);default:0"`
	Currency     string          `json:"currency" gorm:"type:varchar(3);default:'USD'"`
	Features     PlanFeatures    `json:"features" gorm:"type:jsonb;default:'{}'"`
	Limits       PlanLimits      `json:"limits" gorm:"type:jsonb;default:'{}'"`
	IsActive     bool            `json:"is_active" gorm:"default:true"`
	SortOrder    int             `json:"sort_order" gorm:"default:0"`
	CreatedAt    time.Time       `json:"created_at" gorm:"type:timestamp with time zone;default:now()"`
	UpdatedAt    time.Time       `json:"updated_at" gorm:"type:timestamp with time zone;default:now()"`
}

// TableName returns the table name for Plan
func (Plan) TableName() string {
	return "plans"
}

// PlanFeatures defines plan features
type PlanFeatures struct {
	Environments  interface{} `json:"environments"` // int or "unlimited"
	Collaboration bool        `json:"collaboration"`
	Support       string      `json:"support"` // community, email, priority, dedicated
	CustomDomains bool        `json:"custom_domains"`
	SSO           bool        `json:"sso"`
	AuditLogs     bool        `json:"audit_logs"`
}

// PlanLimits defines plan resource limits
type PlanLimits struct {
	MaxEnvironments       int `json:"max_environments"`       // -1 for unlimited
	CPUHoursMonthly       int `json:"cpu_hours_monthly"`      // -1 for unlimited
	MemoryGBHoursMonthly  int `json:"memory_gb_hours_monthly"`
	StorageGB             int `json:"storage_gb"`
	BuildMinutesMonthly   int `json:"build_minutes_monthly"`
}

// ============================================================================
// Subscription Models
// ============================================================================

// SubscriptionStatus represents subscription status
type SubscriptionStatus string

const (
	SubscriptionStatusActive   SubscriptionStatus = "active"
	SubscriptionStatusCanceled SubscriptionStatus = "canceled"
	SubscriptionStatusPastDue  SubscriptionStatus = "past_due"
	SubscriptionStatusTrialing SubscriptionStatus = "trialing"
	SubscriptionStatusPaused   SubscriptionStatus = "paused"
)

// BillingCycle represents billing cycle
type BillingCycle string

const (
	BillingCycleMonthly BillingCycle = "monthly"
	BillingCycleYearly  BillingCycle = "yearly"
)

// Subscription represents a user subscription
type Subscription struct {
	ID                 uuid.UUID          `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	UserID             uuid.UUID          `json:"user_id" gorm:"type:uuid;not null"`
	PlanID             uuid.UUID          `json:"plan_id" gorm:"type:uuid;not null"`
	Plan               *Plan              `json:"plan,omitempty" gorm:"foreignKey:PlanID"`
	Status             SubscriptionStatus `json:"status" gorm:"type:varchar(20);default:'active'"`
	BillingCycle       BillingCycle       `json:"billing_cycle" gorm:"type:varchar(20);default:'monthly'"`
	CurrentPeriodStart time.Time          `json:"current_period_start" gorm:"type:timestamp with time zone;not null"`
	CurrentPeriodEnd   time.Time          `json:"current_period_end" gorm:"type:timestamp with time zone;not null"`
	CancelAtPeriodEnd  bool               `json:"cancel_at_period_end" gorm:"default:false"`
	CanceledAt         *time.Time         `json:"canceled_at,omitempty" gorm:"type:timestamp with time zone"`
	TrialStart         *time.Time         `json:"trial_start,omitempty" gorm:"type:timestamp with time zone"`
	TrialEnd           *time.Time         `json:"trial_end,omitempty" gorm:"type:timestamp with time zone"`
	Metadata           map[string]string  `json:"metadata,omitempty" gorm:"type:jsonb;default:'{}'"`
	CreatedAt          time.Time          `json:"created_at" gorm:"type:timestamp with time zone;default:now()"`
	UpdatedAt          time.Time          `json:"updated_at" gorm:"type:timestamp with time zone;default:now()"`
}

// TableName returns the table name for Subscription
func (Subscription) TableName() string {
	return "subscriptions"
}

// ============================================================================
// Resource Usage Models
// ============================================================================

// ResourceType represents resource type
type ResourceType string

const (
	ResourceTypeCPUHours      ResourceType = "cpu_hours"
	ResourceTypeMemoryGBHours ResourceType = "memory_gb_hours"
	ResourceTypeStorageGBDays ResourceType = "storage_gb_days"
	ResourceTypeNetworkGB     ResourceType = "network_gb"
	ResourceTypeBuildMinutes  ResourceType = "build_minutes"
)

// ResourceUsage represents resource usage record
type ResourceUsage struct {
	ID                 uuid.UUID         `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	UserID             uuid.UUID         `json:"user_id" gorm:"type:uuid;not null"`
	EnvironmentID      *uuid.UUID        `json:"environment_id,omitempty" gorm:"type:uuid"`
	ResourceType       ResourceType      `json:"resource_type" gorm:"type:varchar(50);not null"`
	Quantity           decimal.Decimal   `json:"quantity" gorm:"type:decimal(20,6);not null"`
	Unit               string            `json:"unit" gorm:"type:varchar(20);not null"`
	RecordedAt         time.Time         `json:"recorded_at" gorm:"type:timestamp with time zone;not null"`
	BillingPeriodStart time.Time         `json:"billing_period_start" gorm:"type:timestamp with time zone;not null"`
	BillingPeriodEnd   time.Time         `json:"billing_period_end" gorm:"type:timestamp with time zone;not null"`
	Metadata           map[string]string `json:"metadata,omitempty" gorm:"type:jsonb;default:'{}'"`
	CreatedAt          time.Time         `json:"created_at" gorm:"type:timestamp with time zone;default:now()"`
}

// TableName returns the table name for ResourceUsage
func (ResourceUsage) TableName() string {
	return "resource_usage"
}

// ============================================================================
// Quota Models
// ============================================================================

// Quota represents user resource quota
type Quota struct {
	ID           uuid.UUID       `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	UserID       uuid.UUID       `json:"user_id" gorm:"type:uuid;not null"`
	ResourceType ResourceType    `json:"resource_type" gorm:"type:varchar(50);not null"`
	LimitValue   decimal.Decimal `json:"limit_value" gorm:"type:decimal(20,6);not null"`
	UsedValue    decimal.Decimal `json:"used_value" gorm:"type:decimal(20,6);default:0"`
	ResetAt      *time.Time      `json:"reset_at,omitempty" gorm:"type:timestamp with time zone"`
	CreatedAt    time.Time       `json:"created_at" gorm:"type:timestamp with time zone;default:now()"`
	UpdatedAt    time.Time       `json:"updated_at" gorm:"type:timestamp with time zone;default:now()"`
}

// TableName returns the table name for Quota
func (Quota) TableName() string {
	return "quotas"
}

// IsExceeded checks if quota is exceeded
func (q *Quota) IsExceeded() bool {
	return q.UsedValue.GreaterThanOrEqual(q.LimitValue)
}

// RemainingValue returns remaining quota
func (q *Quota) RemainingValue() decimal.Decimal {
	remaining := q.LimitValue.Sub(q.UsedValue)
	if remaining.IsNegative() {
		return decimal.Zero
	}
	return remaining
}

// UsagePercent returns usage percentage
func (q *Quota) UsagePercent() float64 {
	if q.LimitValue.IsZero() {
		return 0
	}
	return q.UsedValue.Div(q.LimitValue).Mul(decimal.NewFromInt(100)).InexactFloat64()
}
