// Package services provides business logic for the core service.
package services

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/cloud-devbox/services/core/internal/models"
	"github.com/shopspring/decimal"
)

// AlipayProvider implements PaymentProvider for Alipay
type AlipayProvider struct {
	config          AlipayConfig
	privateKey      *rsa.PrivateKey
	alipayPublicKey *rsa.PublicKey
	gatewayURL      string
}

// NewAlipayProvider creates a new AlipayProvider
func NewAlipayProvider(config AlipayConfig) *AlipayProvider {
	provider := &AlipayProvider{
		config: config,
	}

	if config.IsSandbox {
		provider.gatewayURL = "https://openapi-sandbox.dl.alipaydev.com/gateway.do"
	} else {
		provider.gatewayURL = "https://openapi.alipay.com/gateway.do"
	}

	// Parse private key
	if config.PrivateKey != "" {
		block, _ := pem.Decode([]byte(config.PrivateKey))
		if block != nil {
			key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
			if err != nil {
				// Try PKCS1
				key, err = x509.ParsePKCS1PrivateKey(block.Bytes)
			}
			if err == nil {
				if rsaKey, ok := key.(*rsa.PrivateKey); ok {
					provider.privateKey = rsaKey
				}
			}
		}
	}

	// Parse Alipay public key
	if config.AlipayPublicKey != "" {
		block, _ := pem.Decode([]byte(config.AlipayPublicKey))
		if block != nil {
			pub, err := x509.ParsePKIXPublicKey(block.Bytes)
			if err == nil {
				if rsaPub, ok := pub.(*rsa.PublicKey); ok {
					provider.alipayPublicKey = rsaPub
				}
			}
		}
	}

	return provider
}

// CreatePayment creates a payment with Alipay
func (p *AlipayProvider) CreatePayment(ctx context.Context, req *CreatePaymentRequest) (*PaymentResponse, error) {
	if p.privateKey == nil {
		return nil, errors.New("alipay private key not configured")
	}

	// Build biz content
	bizContent := map[string]interface{}{
		"out_trade_no": req.OrderID,
		"total_amount": req.Amount.StringFixed(2),
		"subject":      req.Subject,
		"product_code": "FAST_INSTANT_TRADE_PAY",
	}
	if req.Description != "" {
		bizContent["body"] = req.Description
	}

	bizContentJSON, _ := json.Marshal(bizContent)

	// Build request parameters
	params := map[string]string{
		"app_id":      p.config.AppID,
		"method":      "alipay.trade.page.pay",
		"format":      "JSON",
		"charset":     "utf-8",
		"sign_type":   "RSA2",
		"timestamp":   time.Now().Format("2006-01-02 15:04:05"),
		"version":     "1.0",
		"biz_content": string(bizContentJSON),
	}

	if p.config.NotifyURL != "" {
		params["notify_url"] = p.config.NotifyURL
	}
	if p.config.ReturnURL != "" {
		params["return_url"] = p.config.ReturnURL
	}

	// Sign request
	sign, err := p.signParams(params)
	if err != nil {
		return nil, fmt.Errorf("failed to sign request: %w", err)
	}
	params["sign"] = sign

	// Build payment URL
	paymentURL := p.buildPaymentURL(params)

	return &PaymentResponse{
		PaymentID:  req.OrderID,
		ProviderID: req.OrderID,
		PaymentURL: paymentURL,
		ExpiresAt:  time.Now().Add(30 * time.Minute),
	}, nil
}

// QueryPayment queries payment status from Alipay
func (p *AlipayProvider) QueryPayment(ctx context.Context, paymentID string) (*PaymentQueryResponse, error) {
	if p.privateKey == nil {
		return nil, errors.New("alipay private key not configured")
	}

	bizContent := map[string]interface{}{
		"out_trade_no": paymentID,
	}
	bizContentJSON, _ := json.Marshal(bizContent)

	params := map[string]string{
		"app_id":      p.config.AppID,
		"method":      "alipay.trade.query",
		"format":      "JSON",
		"charset":     "utf-8",
		"sign_type":   "RSA2",
		"timestamp":   time.Now().Format("2006-01-02 15:04:05"),
		"version":     "1.0",
		"biz_content": string(bizContentJSON),
	}

	sign, err := p.signParams(params)
	if err != nil {
		return nil, fmt.Errorf("failed to sign request: %w", err)
	}
	params["sign"] = sign

	// Make API request
	resp, err := p.doRequest(ctx, params)
	if err != nil {
		return nil, err
	}

	// Parse response
	queryResp, ok := resp["alipay_trade_query_response"].(map[string]interface{})
	if !ok {
		return nil, errors.New("invalid response format")
	}

	code, _ := queryResp["code"].(string)
	if code != "10000" {
		msg, _ := queryResp["sub_msg"].(string)
		return nil, fmt.Errorf("alipay error: %s", msg)
	}

	status := p.mapTradeStatus(queryResp["trade_status"].(string))
	amount, _ := decimal.NewFromString(queryResp["total_amount"].(string))

	result := &PaymentQueryResponse{
		PaymentID:  paymentID,
		ProviderID: queryResp["trade_no"].(string),
		Status:     status,
		Amount:     amount,
	}

	if gmtPayment, ok := queryResp["gmt_payment"].(string); ok {
		paidAt, _ := time.Parse("2006-01-02 15:04:05", gmtPayment)
		result.PaidAt = &paidAt
	}

	return result, nil
}

// HandleCallback handles Alipay async notification
func (p *AlipayProvider) HandleCallback(ctx context.Context, data map[string]string) (*PaymentCallbackResult, error) {
	// Verify signature
	if err := p.verifyCallback(data); err != nil {
		return nil, fmt.Errorf("signature verification failed: %w", err)
	}

	tradeStatus := data["trade_status"]
	if tradeStatus != "TRADE_SUCCESS" && tradeStatus != "TRADE_FINISHED" {
		return nil, fmt.Errorf("trade not successful: %s", tradeStatus)
	}

	amount, _ := decimal.NewFromString(data["total_amount"])
	paidAt, _ := time.Parse("2006-01-02 15:04:05", data["gmt_payment"])

	return &PaymentCallbackResult{
		PaymentID: data["out_trade_no"],
		Success:   true,
		Amount:    amount,
		PaidAt:    paidAt,
	}, nil
}

// RefundPayment refunds a payment
func (p *AlipayProvider) RefundPayment(ctx context.Context, paymentID string, amount decimal.Decimal, reason string) error {
	if p.privateKey == nil {
		return errors.New("alipay private key not configured")
	}

	bizContent := map[string]interface{}{
		"out_trade_no":   paymentID,
		"refund_amount":  amount.StringFixed(2),
		"refund_reason":  reason,
		"out_request_no": fmt.Sprintf("%s_%d", paymentID, time.Now().Unix()),
	}
	bizContentJSON, _ := json.Marshal(bizContent)

	params := map[string]string{
		"app_id":      p.config.AppID,
		"method":      "alipay.trade.refund",
		"format":      "JSON",
		"charset":     "utf-8",
		"sign_type":   "RSA2",
		"timestamp":   time.Now().Format("2006-01-02 15:04:05"),
		"version":     "1.0",
		"biz_content": string(bizContentJSON),
	}

	sign, err := p.signParams(params)
	if err != nil {
		return fmt.Errorf("failed to sign request: %w", err)
	}
	params["sign"] = sign

	resp, err := p.doRequest(ctx, params)
	if err != nil {
		return err
	}

	refundResp, ok := resp["alipay_trade_refund_response"].(map[string]interface{})
	if !ok {
		return errors.New("invalid response format")
	}

	code, _ := refundResp["code"].(string)
	if code != "10000" {
		msg, _ := refundResp["sub_msg"].(string)
		return fmt.Errorf("refund failed: %s", msg)
	}

	return nil
}

// signParams signs the request parameters
func (p *AlipayProvider) signParams(params map[string]string) (string, error) {
	// Sort keys
	keys := make([]string, 0, len(params))
	for k := range params {
		if k != "sign" && params[k] != "" {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)

	// Build sign string
	var pairs []string
	for _, k := range keys {
		pairs = append(pairs, fmt.Sprintf("%s=%s", k, params[k]))
	}
	signStr := strings.Join(pairs, "&")

	// Sign with RSA2 (SHA256)
	hash := sha256.Sum256([]byte(signStr))
	signature, err := rsa.SignPKCS1v15(rand.Reader, p.privateKey, crypto.SHA256, hash[:])
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(signature), nil
}

// verifyCallback verifies the callback signature
func (p *AlipayProvider) verifyCallback(data map[string]string) error {
	if p.alipayPublicKey == nil {
		return errors.New("alipay public key not configured")
	}

	sign := data["sign"]
	signType := data["sign_type"]

	// Build verify string
	keys := make([]string, 0, len(data))
	for k := range data {
		if k != "sign" && k != "sign_type" && data[k] != "" {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)

	var pairs []string
	for _, k := range keys {
		pairs = append(pairs, fmt.Sprintf("%s=%s", k, data[k]))
	}
	verifyStr := strings.Join(pairs, "&")

	// Decode signature
	signBytes, err := base64.StdEncoding.DecodeString(sign)
	if err != nil {
		return fmt.Errorf("failed to decode signature: %w", err)
	}

	// Verify based on sign type
	if signType == "RSA2" {
		hash := sha256.Sum256([]byte(verifyStr))
		return rsa.VerifyPKCS1v15(p.alipayPublicKey, crypto.SHA256, hash[:], signBytes)
	}

	return errors.New("unsupported sign type")
}

// buildPaymentURL builds the payment redirect URL
func (p *AlipayProvider) buildPaymentURL(params map[string]string) string {
	values := url.Values{}
	for k, v := range params {
		values.Set(k, v)
	}
	return p.gatewayURL + "?" + values.Encode()
}

// doRequest makes an API request to Alipay
func (p *AlipayProvider) doRequest(ctx context.Context, params map[string]string) (map[string]interface{}, error) {
	values := url.Values{}
	for k, v := range params {
		values.Set(k, v)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", p.gatewayURL, strings.NewReader(values.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// mapTradeStatus maps Alipay trade status to PaymentStatus
func (p *AlipayProvider) mapTradeStatus(status string) models.PaymentStatus {
	switch status {
	case "WAIT_BUYER_PAY":
		return models.PaymentStatusPending
	case "TRADE_SUCCESS", "TRADE_FINISHED":
		return models.PaymentStatusSucceeded
	case "TRADE_CLOSED":
		return models.PaymentStatusFailed
	default:
		return models.PaymentStatusPending
	}
}
