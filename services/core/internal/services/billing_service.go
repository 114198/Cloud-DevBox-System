// Package services provides business logic for the core service.
package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/cloud-devbox/services/core/internal/models"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// BillingService handles billing operations
type BillingService struct {
	db *gorm.DB
}

// NewBillingService creates a new BillingService
func NewBillingService(db *gorm.DB) *BillingService {
	return &BillingService{db: db}
}

// ============================================================================
// Plan Operations
// ============================================================================

// GetPlans returns all active plans
func (s *BillingService) GetPlans(ctx context.Context) ([]models.Plan, error) {
	var plans []models.Plan
	if err := s.db.WithContext(ctx).
		Where("is_active = ?", true).
		Order("sort_order ASC").
		Find(&plans).Error; err != nil {
		return nil, fmt.Errorf("failed to get plans: %w", err)
	}
	return plans, nil
}

// GetPlan returns a plan by ID
func (s *BillingService) GetPlan(ctx context.Context, planID uuid.UUID) (*models.Plan, error) {
	var plan models.Plan
	if err := s.db.WithContext(ctx).First(&plan, planID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("plan not found")
		}
		return nil, fmt.Errorf("failed to get plan: %w", err)
	}
	return &plan, nil
}

// GetPlanByName returns a plan by name
func (s *BillingService) GetPlanByName(ctx context.Context, name string) (*models.Plan, error) {
	var plan models.Plan
	if err := s.db.WithContext(ctx).Where("name = ?", name).First(&plan).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("plan not found")
		}
		return nil, fmt.Errorf("failed to get plan: %w", err)
	}
	return &plan, nil
}

// ============================================================================
// Subscription Operations
// ============================================================================

// GetSubscription returns user's active subscription
func (s *BillingService) GetSubscription(ctx context.Context, userID uuid.UUID) (*models.Subscription, error) {
	var sub models.Subscription
	if err := s.db.WithContext(ctx).
		Preload("Plan").
		Where("user_id = ? AND status IN ?", userID, []string{"active", "trialing"}).
		First(&sub).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil // No active subscription
		}
		return nil, fmt.Errorf("failed to get subscription: %w", err)
	}
	return &sub, nil
}

// CreateSubscription creates a new subscription
func (s *BillingService) CreateSubscription(ctx context.Context, userID, planID uuid.UUID, cycle models.BillingCycle) (*models.Subscription, error) {
	// Check if user already has an active subscription
	existing, err := s.GetSubscription(ctx, userID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, errors.New("user already has an active subscription")
	}

	// Get plan
	plan, err := s.GetPlan(ctx, planID)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	var periodEnd time.Time
	if cycle == models.BillingCycleYearly {
		periodEnd = now.AddDate(1, 0, 0)
	} else {
		periodEnd = now.AddDate(0, 1, 0)
	}

	sub := &models.Subscription{
		ID:                 uuid.New(),
		UserID:             userID,
		PlanID:             planID,
		Status:             models.SubscriptionStatusActive,
		BillingCycle:       cycle,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   periodEnd,
		CreatedAt:          now,
		UpdatedAt:          now,
	}

	if err := s.db.WithContext(ctx).Create(sub).Error; err != nil {
		return nil, fmt.Errorf("failed to create subscription: %w", err)
	}

	// Initialize quotas based on plan limits
	if err := s.initializeQuotas(ctx, userID, plan); err != nil {
		return nil, fmt.Errorf("failed to initialize quotas: %w", err)
	}

	sub.Plan = plan
	return sub, nil
}

// CancelSubscription cancels a subscription
func (s *BillingService) CancelSubscription(ctx context.Context, userID uuid.UUID, immediate bool) error {
	sub, err := s.GetSubscription(ctx, userID)
	if err != nil {
		return err
	}
	if sub == nil {
		return errors.New("no active subscription found")
	}

	now := time.Now()
	if immediate {
		sub.Status = models.SubscriptionStatusCanceled
		sub.CanceledAt = &now
	} else {
		sub.CancelAtPeriodEnd = true
	}
	sub.UpdatedAt = now

	if err := s.db.WithContext(ctx).Save(sub).Error; err != nil {
		return fmt.Errorf("failed to cancel subscription: %w", err)
	}

	return nil
}

// ChangePlan changes subscription plan
func (s *BillingService) ChangePlan(ctx context.Context, userID, newPlanID uuid.UUID) (*models.Subscription, error) {
	sub, err := s.GetSubscription(ctx, userID)
	if err != nil {
		return nil, err
	}
	if sub == nil {
		return nil, errors.New("no active subscription found")
	}

	newPlan, err := s.GetPlan(ctx, newPlanID)
	if err != nil {
		return nil, err
	}

	sub.PlanID = newPlanID
	sub.UpdatedAt = time.Now()

	if err := s.db.WithContext(ctx).Save(sub).Error; err != nil {
		return nil, fmt.Errorf("failed to change plan: %w", err)
	}

	// Update quotas
	if err := s.updateQuotas(ctx, userID, newPlan); err != nil {
		return nil, fmt.Errorf("failed to update quotas: %w", err)
	}

	sub.Plan = newPlan
	return sub, nil
}

// ============================================================================
// Quota Operations
// ============================================================================

// GetQuotas returns user's quotas
func (s *BillingService) GetQuotas(ctx context.Context, userID uuid.UUID) ([]models.Quota, error) {
	var quotas []models.Quota
	if err := s.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Find(&quotas).Error; err != nil {
		return nil, fmt.Errorf("failed to get quotas: %w", err)
	}
	return quotas, nil
}

// GetQuota returns a specific quota
func (s *BillingService) GetQuota(ctx context.Context, userID uuid.UUID, resourceType models.ResourceType) (*models.Quota, error) {
	var quota models.Quota
	if err := s.db.WithContext(ctx).
		Where("user_id = ? AND resource_type = ?", userID, resourceType).
		First(&quota).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get quota: %w", err)
	}
	return &quota, nil
}

// CheckQuota checks if user has enough quota
func (s *BillingService) CheckQuota(ctx context.Context, userID uuid.UUID, resourceType models.ResourceType, amount decimal.Decimal) (bool, error) {
	quota, err := s.GetQuota(ctx, userID, resourceType)
	if err != nil {
		return false, err
	}
	if quota == nil {
		return true, nil // No quota limit
	}

	// -1 means unlimited
	if quota.LimitValue.Equal(decimal.NewFromInt(-1)) {
		return true, nil
	}

	return quota.UsedValue.Add(amount).LessThanOrEqual(quota.LimitValue), nil
}

// IncrementQuotaUsage increments quota usage
func (s *BillingService) IncrementQuotaUsage(ctx context.Context, userID uuid.UUID, resourceType models.ResourceType, amount decimal.Decimal) error {
	result := s.db.WithContext(ctx).
		Model(&models.Quota{}).
		Where("user_id = ? AND resource_type = ?", userID, resourceType).
		Update("used_value", gorm.Expr("used_value + ?", amount))
	
	if result.Error != nil {
		return fmt.Errorf("failed to increment quota: %w", result.Error)
	}
	return nil
}

// initializeQuotas initializes quotas for a user based on plan
func (s *BillingService) initializeQuotas(ctx context.Context, userID uuid.UUID, plan *models.Plan) error {
	quotas := []models.Quota{
		{UserID: userID, ResourceType: models.ResourceTypeCPUHours, LimitValue: decimal.NewFromInt(int64(plan.Limits.CPUHoursMonthly))},
		{UserID: userID, ResourceType: models.ResourceTypeMemoryGBHours, LimitValue: decimal.NewFromInt(int64(plan.Limits.MemoryGBHoursMonthly))},
		{UserID: userID, ResourceType: models.ResourceTypeStorageGBDays, LimitValue: decimal.NewFromInt(int64(plan.Limits.StorageGB))},
		{UserID: userID, ResourceType: models.ResourceTypeBuildMinutes, LimitValue: decimal.NewFromInt(int64(plan.Limits.BuildMinutesMonthly))},
	}

	for _, q := range quotas {
		q.ID = uuid.New()
		q.CreatedAt = time.Now()
		q.UpdatedAt = time.Now()
		
		if err := s.db.WithContext(ctx).
			Clauses(gorm.OnConflict{
				Columns:   []gorm.Column{{Name: "user_id"}, {Name: "resource_type"}},
				DoUpdates: gorm.AssignmentColumns([]string{"limit_value", "updated_at"}),
			}).
			Create(&q).Error; err != nil {
			return err
		}
	}

	return nil
}

// updateQuotas updates quotas based on new plan
func (s *BillingService) updateQuotas(ctx context.Context, userID uuid.UUID, plan *models.Plan) error {
	return s.initializeQuotas(ctx, userID, plan)
}

// ResetMonthlyQuotas resets monthly quotas
func (s *BillingService) ResetMonthlyQuotas(ctx context.Context, userID uuid.UUID) error {
	now := time.Now()
	nextReset := now.AddDate(0, 1, 0)

	result := s.db.WithContext(ctx).
		Model(&models.Quota{}).
		Where("user_id = ? AND resource_type IN ?", userID, []string{
			string(models.ResourceTypeCPUHours),
			string(models.ResourceTypeMemoryGBHours),
			string(models.ResourceTypeBuildMinutes),
		}).
		Updates(map[string]interface{}{
			"used_value": 0,
			"reset_at":   nextReset,
			"updated_at": now,
		})

	if result.Error != nil {
		return fmt.Errorf("failed to reset quotas: %w", result.Error)
	}
	return nil
}
