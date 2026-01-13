// Package models provides data models for the core service.
package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// ============================================================================
// Invoice Models
// ============================================================================

// InvoiceStatus represents invoice status
type InvoiceStatus string

const (
	InvoiceStatusDraft         InvoiceStatus = "draft"
	InvoiceStatusOpen          InvoiceStatus = "open"
	InvoiceStatusPaid          InvoiceStatus = "paid"
	InvoiceStatusVoid          InvoiceStatus = "void"
	InvoiceStatusUncollectible InvoiceStatus = "uncollectible"
)

// Invoice represents an invoice
type Invoice struct {
	ID                 uuid.UUID         `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	UserID             uuid.UUID         `json:"user_id" gorm:"type:uuid;not null"`
	SubscriptionID     *uuid.UUID        `json:"subscription_id,omitempty" gorm:"type:uuid"`
	InvoiceNumber      string            `json:"invoice_number" gorm:"type:varchar(50);unique;not null"`
	Status             InvoiceStatus     `json:"status" gorm:"type:varchar(20);default:'draft'"`
	Currency           string            `json:"currency" gorm:"type:varchar(3);default:'USD'"`
	Subtotal           decimal.Decimal   `json:"subtotal" gorm:"type:decimal(10,2);default:0"`
	Tax                decimal.Decimal   `json:"tax" gorm:"type:decimal(10,2);default:0"`
	Total              decimal.Decimal   `json:"total" gorm:"type:decimal(10,2);default:0"`
	AmountPaid         decimal.Decimal   `json:"amount_paid" gorm:"type:decimal(10,2);default:0"`
	AmountDue          decimal.Decimal   `json:"amount_due" gorm:"type:decimal(10,2);default:0"`
	BillingPeriodStart *time.Time        `json:"billing_period_start,omitempty" gorm:"type:timestamp with time zone"`
	BillingPeriodEnd   *time.Time        `json:"billing_period_end,omitempty" gorm:"type:timestamp with time zone"`
	DueDate            *time.Time        `json:"due_date,omitempty" gorm:"type:timestamp with time zone"`
	PaidAt             *time.Time        `json:"paid_at,omitempty" gorm:"type:timestamp with time zone"`
	PDFURL             string            `json:"pdf_url,omitempty" gorm:"type:text"`
	Metadata           map[string]string `json:"metadata,omitempty" gorm:"type:jsonb;default:'{}'"`
	Items              []InvoiceItem     `json:"items,omitempty" gorm:"foreignKey:InvoiceID"`
	CreatedAt          time.Time         `json:"created_at" gorm:"type:timestamp with time zone;default:now()"`
	UpdatedAt          time.Time         `json:"updated_at" gorm:"type:timestamp with time zone;default:now()"`
}

// TableName returns the table name for Invoice
func (Invoice) TableName() string {
	return "invoices"
}

// InvoiceItem represents an invoice line item
type InvoiceItem struct {
	ID           uuid.UUID         `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	InvoiceID    uuid.UUID         `json:"invoice_id" gorm:"type:uuid;not null"`
	Description  string            `json:"description" gorm:"type:text;not null"`
	Quantity     decimal.Decimal   `json:"quantity" gorm:"type:decimal(20,6);default:1"`
	UnitPrice    decimal.Decimal   `json:"unit_price" gorm:"type:decimal(10,4);not null"`
	Amount       decimal.Decimal   `json:"amount" gorm:"type:decimal(10,2);not null"`
	ResourceType *ResourceType     `json:"resource_type,omitempty" gorm:"type:varchar(50)"`
	Metadata     map[string]string `json:"metadata,omitempty" gorm:"type:jsonb;default:'{}'"`
	CreatedAt    time.Time         `json:"created_at" gorm:"type:timestamp with time zone;default:now()"`
}

// TableName returns the table name for InvoiceItem
func (InvoiceItem) TableName() string {
	return "invoice_items"
}

// ============================================================================
// Payment Models
// ============================================================================

// PaymentMethodType represents payment method type
type PaymentMethodType string

const (
	PaymentMethodTypeCard         PaymentMethodType = "card"
	PaymentMethodTypeAlipay       PaymentMethodType = "alipay"
	PaymentMethodTypeWechat       PaymentMethodType = "wechat"
	PaymentMethodTypeBankTransfer PaymentMethodType = "bank_transfer"
)

// PaymentMethod represents a payment method
type PaymentMethod struct {
	ID             uuid.UUID         `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	UserID         uuid.UUID         `json:"user_id" gorm:"type:uuid;not null"`
	Type           PaymentMethodType `json:"type" gorm:"type:varchar(20);not null"`
	Provider       string            `json:"provider" gorm:"type:varchar(50);not null"`
	ProviderID     string            `json:"-" gorm:"type:varchar(255)"`
	IsDefault      bool              `json:"is_default" gorm:"default:false"`
	LastFour       string            `json:"last_four,omitempty" gorm:"type:varchar(4)"`
	Brand          string            `json:"brand,omitempty" gorm:"type:varchar(50)"`
	ExpMonth       *int              `json:"exp_month,omitempty"`
	ExpYear        *int              `json:"exp_year,omitempty"`
	BillingAddress map[string]string `json:"billing_address,omitempty" gorm:"type:jsonb;default:'{}'"`
	Metadata       map[string]string `json:"metadata,omitempty" gorm:"type:jsonb;default:'{}'"`
	CreatedAt      time.Time         `json:"created_at" gorm:"type:timestamp with time zone;default:now()"`
	UpdatedAt      time.Time         `json:"updated_at" gorm:"type:timestamp with time zone;default:now()"`
}

// TableName returns the table name for PaymentMethod
func (PaymentMethod) TableName() string {
	return "payment_methods"
}

// PaymentStatus represents payment status
type PaymentStatus string

const (
	PaymentStatusPending    PaymentStatus = "pending"
	PaymentStatusProcessing PaymentStatus = "processing"
	PaymentStatusSucceeded  PaymentStatus = "succeeded"
	PaymentStatusFailed     PaymentStatus = "failed"
	PaymentStatusRefunded   PaymentStatus = "refunded"
)

// Payment represents a payment
type Payment struct {
	ID                uuid.UUID         `json:"id" gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	UserID            uuid.UUID         `json:"user_id" gorm:"type:uuid;not null"`
	InvoiceID         *uuid.UUID        `json:"invoice_id,omitempty" gorm:"type:uuid"`
	PaymentMethodID   *uuid.UUID        `json:"payment_method_id,omitempty" gorm:"type:uuid"`
	Amount            decimal.Decimal   `json:"amount" gorm:"type:decimal(10,2);not null"`
	Currency          string            `json:"currency" gorm:"type:varchar(3);default:'USD'"`
	Status            PaymentStatus     `json:"status" gorm:"type:varchar(20);default:'pending'"`
	Provider          string            `json:"provider" gorm:"type:varchar(50);not null"`
	ProviderPaymentID string            `json:"-" gorm:"type:varchar(255)"`
	FailureReason     string            `json:"failure_reason,omitempty" gorm:"type:text"`
	Metadata          map[string]string `json:"metadata,omitempty" gorm:"type:jsonb;default:'{}'"`
	CreatedAt         time.Time         `json:"created_at" gorm:"type:timestamp with time zone;default:now()"`
	UpdatedAt         time.Time         `json:"updated_at" gorm:"type:timestamp with time zone;default:now()"`
}

// TableName returns the table name for Payment
func (Payment) TableName() string {
	return "payments"
}
