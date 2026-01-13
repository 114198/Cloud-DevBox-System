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

// PaymentProvider defines the interface for payment providers
type PaymentProvider interface {
	CreatePayment(ctx context.Context, req *CreatePaymentRequest) (*PaymentResponse, error)
	QueryPayment(ctx context.Context, paymentID string) (*PaymentQueryResponse, error)
	HandleCallback(ctx context.Context, data map[string]string) (*PaymentCallbackResult, error)
	RefundPayment(ctx context.Context, paymentID string, amount decimal.Decimal, reason string) error
}

// CreatePaymentRequest represents a payment creation request
type CreatePaymentRequest struct {
	OrderID     string
	Amount      decimal.Decimal
	Currency    string
	Subject     string
	Description string
	ReturnURL   string
	NotifyURL   string
	ClientIP    string
	UserID      string
}

// PaymentResponse represents a payment creation response
type PaymentResponse struct {
	PaymentID   string
	ProviderID  string
	PaymentURL  string // For redirect-based payments
	QRCodeURL   string // For QR code payments
	PrepayID    string // For WeChat mini-program payments
	ExpiresAt   time.Time
	RawResponse map[string]interface{}
}

// PaymentQueryResponse represents a payment query response
type PaymentQueryResponse struct {
	PaymentID     string
	ProviderID    string
	Status        models.PaymentStatus
	Amount        decimal.Decimal
	PaidAt        *time.Time
	FailureReason string
}

// PaymentCallbackResult represents the result of processing a callback
type PaymentCallbackResult struct {
	PaymentID string
	Success   bool
	Amount    decimal.Decimal
	PaidAt    time.Time
}

// PaymentService handles payment operations
type PaymentService struct {
	db             *gorm.DB
	invoiceService *InvoiceService
	alipay         PaymentProvider
	wechat         PaymentProvider
}

// PaymentConfig holds payment provider configuration
type PaymentConfig struct {
	Alipay AlipayConfig
	Wechat WechatConfig
}

// AlipayConfig holds Alipay configuration
type AlipayConfig struct {
	AppID           string
	PrivateKey      string
	AlipayPublicKey string
	NotifyURL       string
	ReturnURL       string
	IsSandbox       bool
}

// WechatConfig holds WeChat Pay configuration
type WechatConfig struct {
	AppID       string
	MchID       string
	APIKey      string
	CertPath    string
	KeyPath     string
	NotifyURL   string
	IsSandbox   bool
}

// NewPaymentService creates a new PaymentService
func NewPaymentService(db *gorm.DB, invoiceService *InvoiceService, config PaymentConfig) *PaymentService {
	return &PaymentService{
		db:             db,
		invoiceService: invoiceService,
		alipay:         NewAlipayProvider(config.Alipay),
		wechat:         NewWechatProvider(config.Wechat),
	}
}

// CreatePayment creates a new payment
func (s *PaymentService) CreatePayment(ctx context.Context, userID uuid.UUID, invoiceID uuid.UUID, provider string, clientIP string) (*models.Payment, *PaymentResponse, error) {
	// Get invoice
	invoice, err := s.invoiceService.GetInvoice(ctx, userID, invoiceID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get invoice: %w", err)
	}

	if invoice.Status == models.InvoiceStatusPaid {
		return nil, nil, errors.New("invoice is already paid")
	}

	if invoice.AmountDue.LessThanOrEqual(decimal.Zero) {
		return nil, nil, errors.New("no amount due")
	}

	// Create payment record
	payment := &models.Payment{
		ID:        uuid.New(),
		UserID:    userID,
		InvoiceID: &invoiceID,
		Amount:    invoice.AmountDue,
		Currency:  invoice.Currency,
		Status:    models.PaymentStatusPending,
		Provider:  provider,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.db.WithContext(ctx).Create(payment).Error; err != nil {
		return nil, nil, fmt.Errorf("failed to create payment: %w", err)
	}

	// Get payment provider
	var paymentProvider PaymentProvider
	switch provider {
	case "alipay":
		paymentProvider = s.alipay
	case "wechat":
		paymentProvider = s.wechat
	default:
		return nil, nil, fmt.Errorf("unsupported payment provider: %s", provider)
	}

	// Create payment with provider
	req := &CreatePaymentRequest{
		OrderID:     payment.ID.String(),
		Amount:      payment.Amount,
		Currency:    payment.Currency,
		Subject:     fmt.Sprintf("Cloud DevBox - Invoice %s", invoice.InvoiceNumber),
		Description: fmt.Sprintf("Payment for invoice %s", invoice.InvoiceNumber),
		ClientIP:    clientIP,
		UserID:      userID.String(),
	}

	resp, err := paymentProvider.CreatePayment(ctx, req)
	if err != nil {
		// Update payment status to failed
		payment.Status = models.PaymentStatusFailed
		payment.FailureReason = err.Error()
		s.db.WithContext(ctx).Save(payment)
		return nil, nil, fmt.Errorf("failed to create payment with provider: %w", err)
	}

	// Update payment with provider ID
	payment.ProviderPaymentID = resp.ProviderID
	payment.Status = models.PaymentStatusProcessing
	payment.UpdatedAt = time.Now()
	if err := s.db.WithContext(ctx).Save(payment).Error; err != nil {
		return nil, nil, fmt.Errorf("failed to update payment: %w", err)
	}

	return payment, resp, nil
}

// HandlePaymentCallback handles payment callback from provider
func (s *PaymentService) HandlePaymentCallback(ctx context.Context, provider string, data map[string]string) error {
	var paymentProvider PaymentProvider
	switch provider {
	case "alipay":
		paymentProvider = s.alipay
	case "wechat":
		paymentProvider = s.wechat
	default:
		return fmt.Errorf("unsupported payment provider: %s", provider)
	}

	result, err := paymentProvider.HandleCallback(ctx, data)
	if err != nil {
		return fmt.Errorf("failed to handle callback: %w", err)
	}

	// Get payment
	paymentID, err := uuid.Parse(result.PaymentID)
	if err != nil {
		return fmt.Errorf("invalid payment ID: %w", err)
	}

	var payment models.Payment
	if err := s.db.WithContext(ctx).First(&payment, paymentID).Error; err != nil {
		return fmt.Errorf("payment not found: %w", err)
	}

	if payment.Status == models.PaymentStatusSucceeded {
		return nil // Already processed
	}

	if result.Success {
		payment.Status = models.PaymentStatusSucceeded
		payment.UpdatedAt = time.Now()

		if err := s.db.WithContext(ctx).Save(&payment).Error; err != nil {
			return fmt.Errorf("failed to update payment: %w", err)
		}

		// Mark invoice as paid
		if payment.InvoiceID != nil {
			if err := s.invoiceService.MarkInvoicePaid(ctx, *payment.InvoiceID, result.Amount); err != nil {
				return fmt.Errorf("failed to mark invoice as paid: %w", err)
			}
		}
	}

	return nil
}

// GetPayment returns a payment by ID
func (s *PaymentService) GetPayment(ctx context.Context, userID, paymentID uuid.UUID) (*models.Payment, error) {
	var payment models.Payment
	if err := s.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", paymentID, userID).
		First(&payment).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("payment not found")
		}
		return nil, fmt.Errorf("failed to get payment: %w", err)
	}
	return &payment, nil
}

// ListPayments returns user's payments
func (s *PaymentService) ListPayments(ctx context.Context, userID uuid.UUID, page, pageSize int) (*models.PaginatedResponse, error) {
	var payments []models.Payment
	var total int64

	query := s.db.WithContext(ctx).Model(&models.Payment{}).Where("user_id = ?", userID)

	if err := query.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("failed to count payments: %w", err)
	}

	offset := (page - 1) * pageSize
	if err := query.
		Order("created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&payments).Error; err != nil {
		return nil, fmt.Errorf("failed to list payments: %w", err)
	}

	return &models.PaginatedResponse{
		Data:       payments,
		Pagination: models.NewPagination(page, pageSize, total),
	}, nil
}

// QueryPaymentStatus queries payment status from provider
func (s *PaymentService) QueryPaymentStatus(ctx context.Context, userID, paymentID uuid.UUID) (*models.Payment, error) {
	payment, err := s.GetPayment(ctx, userID, paymentID)
	if err != nil {
		return nil, err
	}

	if payment.Status == models.PaymentStatusSucceeded || payment.Status == models.PaymentStatusFailed {
		return payment, nil
	}

	var paymentProvider PaymentProvider
	switch payment.Provider {
	case "alipay":
		paymentProvider = s.alipay
	case "wechat":
		paymentProvider = s.wechat
	default:
		return payment, nil
	}

	resp, err := paymentProvider.QueryPayment(ctx, payment.ProviderPaymentID)
	if err != nil {
		return payment, nil // Return current status on error
	}

	if resp.Status != payment.Status {
		payment.Status = resp.Status
		payment.UpdatedAt = time.Now()
		if resp.FailureReason != "" {
			payment.FailureReason = resp.FailureReason
		}
		s.db.WithContext(ctx).Save(payment)

		if resp.Status == models.PaymentStatusSucceeded && payment.InvoiceID != nil {
			s.invoiceService.MarkInvoicePaid(ctx, *payment.InvoiceID, resp.Amount)
		}
	}

	return payment, nil
}

// GetPaymentMethods returns user's saved payment methods
func (s *PaymentService) GetPaymentMethods(ctx context.Context, userID uuid.UUID) ([]models.PaymentMethod, error) {
	var methods []models.PaymentMethod
	if err := s.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("is_default DESC, created_at DESC").
		Find(&methods).Error; err != nil {
		return nil, fmt.Errorf("failed to get payment methods: %w", err)
	}
	return methods, nil
}

// SetDefaultPaymentMethod sets a payment method as default
func (s *PaymentService) SetDefaultPaymentMethod(ctx context.Context, userID, methodID uuid.UUID) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Unset all defaults
		if err := tx.Model(&models.PaymentMethod{}).
			Where("user_id = ?", userID).
			Update("is_default", false).Error; err != nil {
			return err
		}

		// Set new default
		result := tx.Model(&models.PaymentMethod{}).
			Where("id = ? AND user_id = ?", methodID, userID).
			Update("is_default", true)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return errors.New("payment method not found")
		}
		return nil
	})
}

// DeletePaymentMethod deletes a payment method
func (s *PaymentService) DeletePaymentMethod(ctx context.Context, userID, methodID uuid.UUID) error {
	result := s.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", methodID, userID).
		Delete(&models.PaymentMethod{})
	if result.Error != nil {
		return fmt.Errorf("failed to delete payment method: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return errors.New("payment method not found")
	}
	return nil
}
