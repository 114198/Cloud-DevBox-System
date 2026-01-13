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

// InvoiceService handles invoice operations
type InvoiceService struct {
	db             *gorm.DB
	billingService *BillingService
}

// NewInvoiceService creates a new InvoiceService
func NewInvoiceService(db *gorm.DB, billingService *BillingService) *InvoiceService {
	return &InvoiceService{
		db:             db,
		billingService: billingService,
	}
}

// GenerateInvoiceNumber generates a unique invoice number
func (s *InvoiceService) GenerateInvoiceNumber() string {
	return fmt.Sprintf("INV-%s-%d", time.Now().Format("200601"), time.Now().UnixNano()%100000)
}

// CreateInvoice creates a new invoice
func (s *InvoiceService) CreateInvoice(ctx context.Context, userID uuid.UUID, items []models.InvoiceItem) (*models.Invoice, error) {
	// Calculate totals
	subtotal := decimal.Zero
	for _, item := range items {
		subtotal = subtotal.Add(item.Amount)
	}

	// Calculate tax (example: 0% for now)
	taxRate := decimal.Zero
	tax := subtotal.Mul(taxRate)
	total := subtotal.Add(tax)

	now := time.Now()
	dueDate := now.AddDate(0, 0, 30) // 30 days payment term

	invoice := &models.Invoice{
		ID:            uuid.New(),
		UserID:        userID,
		InvoiceNumber: s.GenerateInvoiceNumber(),
		Status:        models.InvoiceStatusDraft,
		Currency:      "USD",
		Subtotal:      subtotal,
		Tax:           tax,
		Total:         total,
		AmountDue:     total,
		DueDate:       &dueDate,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(invoice).Error; err != nil {
			return err
		}

		for i := range items {
			items[i].ID = uuid.New()
			items[i].InvoiceID = invoice.ID
			items[i].CreatedAt = now
		}

		if len(items) > 0 {
			if err := tx.Create(&items).Error; err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to create invoice: %w", err)
	}

	invoice.Items = items
	return invoice, nil
}

// GetInvoice returns an invoice by ID
func (s *InvoiceService) GetInvoice(ctx context.Context, userID, invoiceID uuid.UUID) (*models.Invoice, error) {
	var invoice models.Invoice
	if err := s.db.WithContext(ctx).
		Preload("Items").
		Where("id = ? AND user_id = ?", invoiceID, userID).
		First(&invoice).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("invoice not found")
		}
		return nil, fmt.Errorf("failed to get invoice: %w", err)
	}
	return &invoice, nil
}

// ListInvoices returns user's invoices
func (s *InvoiceService) ListInvoices(ctx context.Context, userID uuid.UUID, page, pageSize int) (*models.PaginatedResponse, error) {
	var invoices []models.Invoice
	var total int64

	query := s.db.WithContext(ctx).Model(&models.Invoice{}).Where("user_id = ?", userID)

	if err := query.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("failed to count invoices: %w", err)
	}

	offset := (page - 1) * pageSize
	if err := query.
		Order("created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&invoices).Error; err != nil {
		return nil, fmt.Errorf("failed to list invoices: %w", err)
	}

	return &models.PaginatedResponse{
		Data:       invoices,
		Pagination: models.NewPagination(page, pageSize, total),
	}, nil
}

// FinalizeInvoice finalizes a draft invoice
func (s *InvoiceService) FinalizeInvoice(ctx context.Context, userID, invoiceID uuid.UUID) (*models.Invoice, error) {
	invoice, err := s.GetInvoice(ctx, userID, invoiceID)
	if err != nil {
		return nil, err
	}

	if invoice.Status != models.InvoiceStatusDraft {
		return nil, errors.New("only draft invoices can be finalized")
	}

	invoice.Status = models.InvoiceStatusOpen
	invoice.UpdatedAt = time.Now()

	if err := s.db.WithContext(ctx).Save(invoice).Error; err != nil {
		return nil, fmt.Errorf("failed to finalize invoice: %w", err)
	}

	return invoice, nil
}

// MarkInvoicePaid marks an invoice as paid
func (s *InvoiceService) MarkInvoicePaid(ctx context.Context, invoiceID uuid.UUID, amount decimal.Decimal) error {
	var invoice models.Invoice
	if err := s.db.WithContext(ctx).First(&invoice, invoiceID).Error; err != nil {
		return fmt.Errorf("failed to get invoice: %w", err)
	}

	now := time.Now()
	invoice.AmountPaid = invoice.AmountPaid.Add(amount)
	invoice.AmountDue = invoice.Total.Sub(invoice.AmountPaid)

	if invoice.AmountDue.LessThanOrEqual(decimal.Zero) {
		invoice.Status = models.InvoiceStatusPaid
		invoice.PaidAt = &now
		invoice.AmountDue = decimal.Zero
	}

	invoice.UpdatedAt = now

	if err := s.db.WithContext(ctx).Save(&invoice).Error; err != nil {
		return fmt.Errorf("failed to update invoice: %w", err)
	}

	return nil
}

// VoidInvoice voids an invoice
func (s *InvoiceService) VoidInvoice(ctx context.Context, userID, invoiceID uuid.UUID) error {
	invoice, err := s.GetInvoice(ctx, userID, invoiceID)
	if err != nil {
		return err
	}

	if invoice.Status == models.InvoiceStatusPaid {
		return errors.New("cannot void a paid invoice")
	}

	invoice.Status = models.InvoiceStatusVoid
	invoice.UpdatedAt = time.Now()

	if err := s.db.WithContext(ctx).Save(invoice).Error; err != nil {
		return fmt.Errorf("failed to void invoice: %w", err)
	}

	return nil
}

// GenerateMonthlyInvoice generates monthly invoice for a user
func (s *InvoiceService) GenerateMonthlyInvoice(ctx context.Context, userID uuid.UUID, periodStart, periodEnd time.Time) (*models.Invoice, error) {
	// Get subscription
	sub, err := s.billingService.GetSubscription(ctx, userID)
	if err != nil {
		return nil, err
	}
	if sub == nil || sub.Plan == nil {
		return nil, errors.New("no active subscription")
	}

	// Get resource usage for the period
	var usages []models.ResourceUsage
	if err := s.db.WithContext(ctx).
		Where("user_id = ? AND billing_period_start >= ? AND billing_period_end <= ?", userID, periodStart, periodEnd).
		Find(&usages).Error; err != nil {
		return nil, fmt.Errorf("failed to get usage: %w", err)
	}

	// Create invoice items
	items := []models.InvoiceItem{}

	// Add subscription fee
	var price decimal.Decimal
	if sub.BillingCycle == models.BillingCycleYearly {
		price = sub.Plan.PriceYearly.Div(decimal.NewFromInt(12))
	} else {
		price = sub.Plan.PriceMonthly
	}

	if price.GreaterThan(decimal.Zero) {
		items = append(items, models.InvoiceItem{
			Description: fmt.Sprintf("%s Plan - %s", sub.Plan.DisplayName, periodStart.Format("January 2006")),
			Quantity:    decimal.NewFromInt(1),
			UnitPrice:   price,
			Amount:      price,
		})
	}

	// Add overage charges if any (simplified example)
	// In production, you would calculate based on actual usage vs plan limits

	if len(items) == 0 {
		return nil, nil // No charges
	}

	invoice, err := s.CreateInvoice(ctx, userID, items)
	if err != nil {
		return nil, err
	}

	// Set billing period
	invoice.BillingPeriodStart = &periodStart
	invoice.BillingPeriodEnd = &periodEnd
	invoice.SubscriptionID = &sub.ID

	if err := s.db.WithContext(ctx).Save(invoice).Error; err != nil {
		return nil, fmt.Errorf("failed to update invoice: %w", err)
	}

	return invoice, nil
}
