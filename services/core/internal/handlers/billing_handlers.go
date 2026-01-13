// Package handlers provides HTTP request handlers for the core service.
package handlers

import (
	"net/http"
	"strconv"

	"github.com/cloud-devbox/services/core/internal/models"
	"github.com/cloud-devbox/services/core/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// BillingHandler handles billing related requests
type BillingHandler struct {
	billingService *services.BillingService
	invoiceService *services.InvoiceService
}

// NewBillingHandler creates a new BillingHandler
func NewBillingHandler(billingService *services.BillingService, invoiceService *services.InvoiceService) *BillingHandler {
	return &BillingHandler{
		billingService: billingService,
		invoiceService: invoiceService,
	}
}

// GetPlans returns all available plans
// @Summary Get plans
// @Description Get all available subscription plans
// @Tags Billing
// @Produce json
// @Success 200 {array} models.Plan
// @Router /api/v1/billing/plans [get]
func (h *BillingHandler) GetPlans(c *gin.Context) {
	plans, err := h.billingService.GetPlans(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.NewAPIError(models.ErrCodeInternalError, err.Error()))
		return
	}
	c.JSON(http.StatusOK, plans)
}

// GetSubscription returns user's current subscription
// @Summary Get subscription
// @Description Get current user's subscription
// @Tags Billing
// @Produce json
// @Success 200 {object} models.Subscription
// @Failure 401 {object} models.APIError
// @Router /api/v1/billing/subscription [get]
func (h *BillingHandler) GetSubscription(c *gin.Context) {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, models.NewAPIError(models.ErrCodeUnauthorized, "unauthorized"))
		return
	}

	sub, err := h.billingService.GetSubscription(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.NewAPIError(models.ErrCodeInternalError, err.Error()))
		return
	}

	if sub == nil {
		c.JSON(http.StatusOK, gin.H{"subscription": nil, "message": "No active subscription"})
		return
	}

	c.JSON(http.StatusOK, sub)
}

// CreateSubscription creates a new subscription
// @Summary Create subscription
// @Description Create a new subscription
// @Tags Billing
// @Accept json
// @Produce json
// @Param request body object{plan_id=string,billing_cycle=string} true "Subscription request"
// @Success 201 {object} models.Subscription
// @Failure 400 {object} models.APIError
// @Failure 401 {object} models.APIError
// @Router /api/v1/billing/subscription [post]
func (h *BillingHandler) CreateSubscription(c *gin.Context) {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, models.NewAPIError(models.ErrCodeUnauthorized, "unauthorized"))
		return
	}

	var req struct {
		PlanID       string `json:"plan_id" binding:"required"`
		BillingCycle string `json:"billing_cycle"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.NewAPIError(models.ErrCodeValidationFailed, err.Error()))
		return
	}

	planID, err := uuid.Parse(req.PlanID)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.NewAPIError(models.ErrCodeBadRequest, "invalid plan ID"))
		return
	}

	cycle := models.BillingCycleMonthly
	if req.BillingCycle == "yearly" {
		cycle = models.BillingCycleYearly
	}

	sub, err := h.billingService.CreateSubscription(c.Request.Context(), userID, planID, cycle)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.NewAPIError(models.ErrCodeBadRequest, err.Error()))
		return
	}

	c.JSON(http.StatusCreated, sub)
}

// CancelSubscription cancels the subscription
// @Summary Cancel subscription
// @Description Cancel current subscription
// @Tags Billing
// @Accept json
// @Param request body object{immediate=bool} false "Cancel options"
// @Success 200 {object} object{message=string}
// @Failure 400 {object} models.APIError
// @Failure 401 {object} models.APIError
// @Router /api/v1/billing/subscription/cancel [post]
func (h *BillingHandler) CancelSubscription(c *gin.Context) {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, models.NewAPIError(models.ErrCodeUnauthorized, "unauthorized"))
		return
	}

	var req struct {
		Immediate bool `json:"immediate"`
	}
	c.ShouldBindJSON(&req)

	if err := h.billingService.CancelSubscription(c.Request.Context(), userID, req.Immediate); err != nil {
		c.JSON(http.StatusBadRequest, models.NewAPIError(models.ErrCodeBadRequest, err.Error()))
		return
	}

	message := "Subscription will be canceled at the end of the billing period"
	if req.Immediate {
		message = "Subscription canceled immediately"
	}
	c.JSON(http.StatusOK, gin.H{"message": message})
}

// ChangePlan changes the subscription plan
// @Summary Change plan
// @Description Change subscription plan
// @Tags Billing
// @Accept json
// @Produce json
// @Param request body object{plan_id=string} true "New plan"
// @Success 200 {object} models.Subscription
// @Failure 400 {object} models.APIError
// @Failure 401 {object} models.APIError
// @Router /api/v1/billing/subscription/change-plan [post]
func (h *BillingHandler) ChangePlan(c *gin.Context) {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, models.NewAPIError(models.ErrCodeUnauthorized, "unauthorized"))
		return
	}

	var req struct {
		PlanID string `json:"plan_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.NewAPIError(models.ErrCodeValidationFailed, err.Error()))
		return
	}

	planID, err := uuid.Parse(req.PlanID)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.NewAPIError(models.ErrCodeBadRequest, "invalid plan ID"))
		return
	}

	sub, err := h.billingService.ChangePlan(c.Request.Context(), userID, planID)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.NewAPIError(models.ErrCodeBadRequest, err.Error()))
		return
	}

	c.JSON(http.StatusOK, sub)
}

// GetQuotas returns user's quotas
// @Summary Get quotas
// @Description Get current user's resource quotas
// @Tags Billing
// @Produce json
// @Success 200 {array} models.Quota
// @Failure 401 {object} models.APIError
// @Router /api/v1/billing/quotas [get]
func (h *BillingHandler) GetQuotas(c *gin.Context) {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, models.NewAPIError(models.ErrCodeUnauthorized, "unauthorized"))
		return
	}

	quotas, err := h.billingService.GetQuotas(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.NewAPIError(models.ErrCodeInternalError, err.Error()))
		return
	}

	c.JSON(http.StatusOK, quotas)
}

// GetInvoices returns user's invoices
// @Summary Get invoices
// @Description Get current user's invoices
// @Tags Billing
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Page size" default(20)
// @Success 200 {object} models.PaginatedResponse
// @Failure 401 {object} models.APIError
// @Router /api/v1/billing/invoices [get]
func (h *BillingHandler) GetInvoices(c *gin.Context) {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, models.NewAPIError(models.ErrCodeUnauthorized, "unauthorized"))
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	resp, err := h.invoiceService.ListInvoices(c.Request.Context(), userID, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.NewAPIError(models.ErrCodeInternalError, err.Error()))
		return
	}

	c.JSON(http.StatusOK, resp)
}

// GetInvoice returns a specific invoice
// @Summary Get invoice
// @Description Get a specific invoice
// @Tags Billing
// @Produce json
// @Param id path string true "Invoice ID"
// @Success 200 {object} models.Invoice
// @Failure 401 {object} models.APIError
// @Failure 404 {object} models.APIError
// @Router /api/v1/billing/invoices/{id} [get]
func (h *BillingHandler) GetInvoice(c *gin.Context) {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, models.NewAPIError(models.ErrCodeUnauthorized, "unauthorized"))
		return
	}

	invoiceID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, models.NewAPIError(models.ErrCodeBadRequest, "invalid invoice ID"))
		return
	}

	invoice, err := h.invoiceService.GetInvoice(c.Request.Context(), userID, invoiceID)
	if err != nil {
		c.JSON(http.StatusNotFound, models.NewAPIError(models.ErrCodeNotFound, err.Error()))
		return
	}

	c.JSON(http.StatusOK, invoice)
}
