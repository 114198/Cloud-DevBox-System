// Package handlers provides HTTP request handlers for the core service.
package handlers

import (
	"bytes"
	"encoding/xml"
	"io"
	"net/http"
	"strconv"

	"github.com/cloud-devbox/services/core/internal/models"
	"github.com/cloud-devbox/services/core/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// PaymentHandler handles payment related requests
type PaymentHandler struct {
	paymentService *services.PaymentService
}

// NewPaymentHandler creates a new PaymentHandler
func NewPaymentHandler(paymentService *services.PaymentService) *PaymentHandler {
	return &PaymentHandler{
		paymentService: paymentService,
	}
}

// CreatePayment creates a new payment
// @Summary Create payment
// @Description Create a new payment for an invoice
// @Tags Payments
// @Accept json
// @Produce json
// @Param request body CreatePaymentRequest true "Payment request"
// @Success 201 {object} CreatePaymentResponse
// @Failure 400 {object} models.APIError
// @Failure 401 {object} models.APIError
// @Router /api/v1/payments [post]
func (h *PaymentHandler) CreatePayment(c *gin.Context) {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, models.NewAPIError(models.ErrCodeUnauthorized, "unauthorized"))
		return
	}

	var req CreatePaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.NewAPIError(models.ErrCodeValidationFailed, err.Error()))
		return
	}

	invoiceID, err := uuid.Parse(req.InvoiceID)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.NewAPIError(models.ErrCodeBadRequest, "invalid invoice ID"))
		return
	}

	// Validate provider
	if req.Provider != "alipay" && req.Provider != "wechat" {
		c.JSON(http.StatusBadRequest, models.NewAPIError(models.ErrCodeBadRequest, "invalid payment provider, must be 'alipay' or 'wechat'"))
		return
	}

	clientIP := c.ClientIP()
	if clientIP == "" {
		clientIP = "127.0.0.1"
	}

	payment, paymentResp, err := h.paymentService.CreatePayment(c.Request.Context(), userID, invoiceID, req.Provider, clientIP)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.NewAPIError(models.ErrCodeBadRequest, err.Error()))
		return
	}

	c.JSON(http.StatusCreated, CreatePaymentResponse{
		Payment:    payment,
		PaymentURL: paymentResp.PaymentURL,
		QRCodeURL:  paymentResp.QRCodeURL,
		ExpiresAt:  paymentResp.ExpiresAt.Format("2006-01-02T15:04:05Z07:00"),
	})
}

// GetPayment returns a payment by ID
// @Summary Get payment
// @Description Get a payment by ID
// @Tags Payments
// @Produce json
// @Param id path string true "Payment ID"
// @Success 200 {object} models.Payment
// @Failure 401 {object} models.APIError
// @Failure 404 {object} models.APIError
// @Router /api/v1/payments/{id} [get]
func (h *PaymentHandler) GetPayment(c *gin.Context) {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, models.NewAPIError(models.ErrCodeUnauthorized, "unauthorized"))
		return
	}

	paymentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, models.NewAPIError(models.ErrCodeBadRequest, "invalid payment ID"))
		return
	}

	payment, err := h.paymentService.GetPayment(c.Request.Context(), userID, paymentID)
	if err != nil {
		c.JSON(http.StatusNotFound, models.NewAPIError(models.ErrCodeNotFound, err.Error()))
		return
	}

	c.JSON(http.StatusOK, payment)
}

// ListPayments returns user's payments
// @Summary List payments
// @Description List user's payments
// @Tags Payments
// @Produce json
// @Param page query int false "Page number" default(1)
// @Param page_size query int false "Page size" default(20)
// @Success 200 {object} models.PaginatedResponse
// @Failure 401 {object} models.APIError
// @Router /api/v1/payments [get]
func (h *PaymentHandler) ListPayments(c *gin.Context) {
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

	resp, err := h.paymentService.ListPayments(c.Request.Context(), userID, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.NewAPIError(models.ErrCodeInternalError, err.Error()))
		return
	}

	c.JSON(http.StatusOK, resp)
}

// QueryPaymentStatus queries payment status from provider
// @Summary Query payment status
// @Description Query payment status from payment provider
// @Tags Payments
// @Produce json
// @Param id path string true "Payment ID"
// @Success 200 {object} models.Payment
// @Failure 401 {object} models.APIError
// @Failure 404 {object} models.APIError
// @Router /api/v1/payments/{id}/status [get]
func (h *PaymentHandler) QueryPaymentStatus(c *gin.Context) {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, models.NewAPIError(models.ErrCodeUnauthorized, "unauthorized"))
		return
	}

	paymentID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, models.NewAPIError(models.ErrCodeBadRequest, "invalid payment ID"))
		return
	}

	payment, err := h.paymentService.QueryPaymentStatus(c.Request.Context(), userID, paymentID)
	if err != nil {
		c.JSON(http.StatusNotFound, models.NewAPIError(models.ErrCodeNotFound, err.Error()))
		return
	}

	c.JSON(http.StatusOK, payment)
}

// AlipayCallback handles Alipay async notification
// @Summary Alipay callback
// @Description Handle Alipay payment notification
// @Tags Payments
// @Accept x-www-form-urlencoded
// @Produce text/plain
// @Success 200 {string} string "success"
// @Failure 400 {string} string "fail"
// @Router /api/v1/payments/callback/alipay [post]
func (h *PaymentHandler) AlipayCallback(c *gin.Context) {
	if err := c.Request.ParseForm(); err != nil {
		c.String(http.StatusBadRequest, "fail")
		return
	}

	data := make(map[string]string)
	for k, v := range c.Request.Form {
		if len(v) > 0 {
			data[k] = v[0]
		}
	}

	if err := h.paymentService.HandlePaymentCallback(c.Request.Context(), "alipay", data); err != nil {
		c.String(http.StatusBadRequest, "fail")
		return
	}

	c.String(http.StatusOK, "success")
}

// WechatCallback handles WeChat Pay notification
// @Summary WeChat callback
// @Description Handle WeChat Pay notification
// @Tags Payments
// @Accept xml
// @Produce xml
// @Success 200 {object} WechatCallbackResponse
// @Failure 400 {object} WechatCallbackResponse
// @Router /api/v1/payments/callback/wechat [post]
func (h *PaymentHandler) WechatCallback(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.XML(http.StatusBadRequest, WechatCallbackResponse{
			ReturnCode: "FAIL",
			ReturnMsg:  "read body failed",
		})
		return
	}

	// Parse XML to map
	data, err := parseWechatXML(body)
	if err != nil {
		c.XML(http.StatusBadRequest, WechatCallbackResponse{
			ReturnCode: "FAIL",
			ReturnMsg:  "parse xml failed",
		})
		return
	}

	if err := h.paymentService.HandlePaymentCallback(c.Request.Context(), "wechat", data); err != nil {
		c.XML(http.StatusBadRequest, WechatCallbackResponse{
			ReturnCode: "FAIL",
			ReturnMsg:  err.Error(),
		})
		return
	}

	c.XML(http.StatusOK, WechatCallbackResponse{
		ReturnCode: "SUCCESS",
		ReturnMsg:  "OK",
	})
}

// GetPaymentMethods returns user's saved payment methods
// @Summary Get payment methods
// @Description Get user's saved payment methods
// @Tags Payments
// @Produce json
// @Success 200 {array} models.PaymentMethod
// @Failure 401 {object} models.APIError
// @Router /api/v1/payments/methods [get]
func (h *PaymentHandler) GetPaymentMethods(c *gin.Context) {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, models.NewAPIError(models.ErrCodeUnauthorized, "unauthorized"))
		return
	}

	methods, err := h.paymentService.GetPaymentMethods(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.NewAPIError(models.ErrCodeInternalError, err.Error()))
		return
	}

	c.JSON(http.StatusOK, methods)
}

// SetDefaultPaymentMethod sets a payment method as default
// @Summary Set default payment method
// @Description Set a payment method as the default
// @Tags Payments
// @Param id path string true "Payment Method ID"
// @Success 200 {object} object{message=string}
// @Failure 400 {object} models.APIError
// @Failure 401 {object} models.APIError
// @Router /api/v1/payments/methods/{id}/default [post]
func (h *PaymentHandler) SetDefaultPaymentMethod(c *gin.Context) {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, models.NewAPIError(models.ErrCodeUnauthorized, "unauthorized"))
		return
	}

	methodID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, models.NewAPIError(models.ErrCodeBadRequest, "invalid payment method ID"))
		return
	}

	if err := h.paymentService.SetDefaultPaymentMethod(c.Request.Context(), userID, methodID); err != nil {
		c.JSON(http.StatusBadRequest, models.NewAPIError(models.ErrCodeBadRequest, err.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Default payment method updated"})
}

// DeletePaymentMethod deletes a payment method
// @Summary Delete payment method
// @Description Delete a saved payment method
// @Tags Payments
// @Param id path string true "Payment Method ID"
// @Success 200 {object} object{message=string}
// @Failure 400 {object} models.APIError
// @Failure 401 {object} models.APIError
// @Router /api/v1/payments/methods/{id} [delete]
func (h *PaymentHandler) DeletePaymentMethod(c *gin.Context) {
	userID, err := getUserIDFromContext(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, models.NewAPIError(models.ErrCodeUnauthorized, "unauthorized"))
		return
	}

	methodID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, models.NewAPIError(models.ErrCodeBadRequest, "invalid payment method ID"))
		return
	}

	if err := h.paymentService.DeletePaymentMethod(c.Request.Context(), userID, methodID); err != nil {
		c.JSON(http.StatusBadRequest, models.NewAPIError(models.ErrCodeBadRequest, err.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Payment method deleted"})
}

// Request/Response types

// CreatePaymentRequest represents a payment creation request
type CreatePaymentRequest struct {
	InvoiceID string `json:"invoice_id" binding:"required"`
	Provider  string `json:"provider" binding:"required"` // alipay or wechat
}

// CreatePaymentResponse represents a payment creation response
type CreatePaymentResponse struct {
	Payment    *models.Payment `json:"payment"`
	PaymentURL string          `json:"payment_url,omitempty"` // For redirect-based payments (Alipay)
	QRCodeURL  string          `json:"qr_code_url,omitempty"` // For QR code payments (WeChat)
	ExpiresAt  string          `json:"expires_at"`
}

// WechatCallbackResponse represents WeChat callback response
type WechatCallbackResponse struct {
	XMLName    xml.Name `xml:"xml"`
	ReturnCode string   `xml:"return_code"`
	ReturnMsg  string   `xml:"return_msg"`
}

// parseWechatXML parses WeChat XML notification to map
func parseWechatXML(data []byte) (map[string]string, error) {
	result := make(map[string]string)
	decoder := xml.NewDecoder(bytes.NewReader(data))

	var currentKey string
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		switch t := token.(type) {
		case xml.StartElement:
			currentKey = t.Name.Local
		case xml.CharData:
			if currentKey != "" && currentKey != "xml" {
				result[currentKey] = string(t)
			}
		case xml.EndElement:
			currentKey = ""
		}
	}

	return result, nil
}


