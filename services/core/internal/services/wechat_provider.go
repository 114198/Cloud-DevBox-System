// Package services provides business logic for the core service.
package services

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/cloud-devbox/services/core/internal/models"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// WechatProvider implements PaymentProvider for WeChat Pay
type WechatProvider struct {
	config     WechatConfig
	gatewayURL string
	httpClient *http.Client
}

// WechatPayRequest represents a WeChat Pay unified order request
type WechatPayRequest struct {
	XMLName        xml.Name `xml:"xml"`
	AppID          string   `xml:"appid"`
	MchID          string   `xml:"mch_id"`
	NonceStr       string   `xml:"nonce_str"`
	Sign           string   `xml:"sign"`
	SignType       string   `xml:"sign_type,omitempty"`
	Body           string   `xml:"body"`
	OutTradeNo     string   `xml:"out_trade_no"`
	TotalFee       int64    `xml:"total_fee"`
	SpbillCreateIP string   `xml:"spbill_create_ip"`
	NotifyURL      string   `xml:"notify_url"`
	TradeType      string   `xml:"trade_type"`
	ProductID      string   `xml:"product_id,omitempty"`
}

// WechatPayResponse represents a WeChat Pay unified order response
type WechatPayResponse struct {
	XMLName    xml.Name `xml:"xml"`
	ReturnCode string   `xml:"return_code"`
	ReturnMsg  string   `xml:"return_msg"`
	ResultCode string   `xml:"result_code"`
	ErrCode    string   `xml:"err_code"`
	ErrCodeDes string   `xml:"err_code_des"`
	AppID      string   `xml:"appid"`
	MchID      string   `xml:"mch_id"`
	NonceStr   string   `xml:"nonce_str"`
	Sign       string   `xml:"sign"`
	PrepayID   string   `xml:"prepay_id"`
	TradeType  string   `xml:"trade_type"`
	CodeURL    string   `xml:"code_url"`
}

// WechatQueryRequest represents a WeChat Pay order query request
type WechatQueryRequest struct {
	XMLName    xml.Name `xml:"xml"`
	AppID      string   `xml:"appid"`
	MchID      string   `xml:"mch_id"`
	OutTradeNo string   `xml:"out_trade_no,omitempty"`
	NonceStr   string   `xml:"nonce_str"`
	Sign       string   `xml:"sign"`
	SignType   string   `xml:"sign_type,omitempty"`
}

// WechatQueryResponse represents a WeChat Pay order query response
type WechatQueryResponse struct {
	XMLName        xml.Name `xml:"xml"`
	ReturnCode     string   `xml:"return_code"`
	ReturnMsg      string   `xml:"return_msg"`
	ResultCode     string   `xml:"result_code"`
	ErrCode        string   `xml:"err_code"`
	ErrCodeDes     string   `xml:"err_code_des"`
	AppID          string   `xml:"appid"`
	MchID          string   `xml:"mch_id"`
	NonceStr       string   `xml:"nonce_str"`
	Sign           string   `xml:"sign"`
	OutTradeNo     string   `xml:"out_trade_no"`
	TransactionID  string   `xml:"transaction_id"`
	TradeState     string   `xml:"trade_state"`
	TradeStateDesc string   `xml:"trade_state_desc"`
	TotalFee       int64    `xml:"total_fee"`
	TimeEnd        string   `xml:"time_end"`
}

// WechatNotifyRequest represents a WeChat Pay notification
type WechatNotifyRequest struct {
	XMLName       xml.Name `xml:"xml"`
	ReturnCode    string   `xml:"return_code"`
	ReturnMsg     string   `xml:"return_msg"`
	ResultCode    string   `xml:"result_code"`
	ErrCode       string   `xml:"err_code"`
	ErrCodeDes    string   `xml:"err_code_des"`
	AppID         string   `xml:"appid"`
	MchID         string   `xml:"mch_id"`
	NonceStr      string   `xml:"nonce_str"`
	Sign          string   `xml:"sign"`
	SignType      string   `xml:"sign_type"`
	OutTradeNo    string   `xml:"out_trade_no"`
	TransactionID string   `xml:"transaction_id"`
	TradeType     string   `xml:"trade_type"`
	TotalFee      int64    `xml:"total_fee"`
	TimeEnd       string   `xml:"time_end"`
}

// WechatRefundRequest represents a WeChat Pay refund request
type WechatRefundRequest struct {
	XMLName       xml.Name `xml:"xml"`
	AppID         string   `xml:"appid"`
	MchID         string   `xml:"mch_id"`
	NonceStr      string   `xml:"nonce_str"`
	Sign          string   `xml:"sign"`
	SignType      string   `xml:"sign_type,omitempty"`
	OutTradeNo    string   `xml:"out_trade_no"`
	OutRefundNo   string   `xml:"out_refund_no"`
	TotalFee      int64    `xml:"total_fee"`
	RefundFee     int64    `xml:"refund_fee"`
	RefundDesc    string   `xml:"refund_desc,omitempty"`
}

// WechatRefundResponse represents a WeChat Pay refund response
type WechatRefundResponse struct {
	XMLName    xml.Name `xml:"xml"`
	ReturnCode string   `xml:"return_code"`
	ReturnMsg  string   `xml:"return_msg"`
	ResultCode string   `xml:"result_code"`
	ErrCode    string   `xml:"err_code"`
	ErrCodeDes string   `xml:"err_code_des"`
}

// NewWechatProvider creates a new WechatProvider
func NewWechatProvider(config WechatConfig) *WechatProvider {
	provider := &WechatProvider{
		config: config,
	}

	if config.IsSandbox {
		provider.gatewayURL = "https://api.mch.weixin.qq.com/sandboxnew"
	} else {
		provider.gatewayURL = "https://api.mch.weixin.qq.com"
	}

	// Create HTTP client with TLS config for certificate-based auth
	tlsConfig := &tls.Config{}
	if config.CertPath != "" && config.KeyPath != "" {
		cert, err := tls.LoadX509KeyPair(config.CertPath, config.KeyPath)
		if err == nil {
			tlsConfig.Certificates = []tls.Certificate{cert}
		}
	}

	provider.httpClient = &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}

	return provider
}

// CreatePayment creates a payment with WeChat Pay
func (p *WechatProvider) CreatePayment(ctx context.Context, req *CreatePaymentRequest) (*PaymentResponse, error) {
	// Convert amount to cents (WeChat uses cents)
	totalFee := req.Amount.Mul(decimal.NewFromInt(100)).IntPart()

	nonceStr := generateNonceStr()

	wxReq := &WechatPayRequest{
		AppID:          p.config.AppID,
		MchID:          p.config.MchID,
		NonceStr:       nonceStr,
		Body:           req.Subject,
		OutTradeNo:     req.OrderID,
		TotalFee:       totalFee,
		SpbillCreateIP: req.ClientIP,
		NotifyURL:      p.config.NotifyURL,
		TradeType:      "NATIVE", // QR code payment
		ProductID:      req.OrderID,
	}

	// Sign request
	wxReq.Sign = p.signRequest(wxReq)

	// Make request
	xmlData, err := xml.Marshal(wxReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.gatewayURL+"/pay/unifiedorder", bytes.NewReader(xmlData))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/xml")

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var wxResp WechatPayResponse
	if err := xml.Unmarshal(body, &wxResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if wxResp.ReturnCode != "SUCCESS" {
		return nil, fmt.Errorf("wechat error: %s", wxResp.ReturnMsg)
	}

	if wxResp.ResultCode != "SUCCESS" {
		return nil, fmt.Errorf("wechat error: %s - %s", wxResp.ErrCode, wxResp.ErrCodeDes)
	}

	return &PaymentResponse{
		PaymentID:  req.OrderID,
		ProviderID: wxResp.PrepayID,
		QRCodeURL:  wxResp.CodeURL,
		PrepayID:   wxResp.PrepayID,
		ExpiresAt:  time.Now().Add(2 * time.Hour),
	}, nil
}

// QueryPayment queries payment status from WeChat Pay
func (p *WechatProvider) QueryPayment(ctx context.Context, paymentID string) (*PaymentQueryResponse, error) {
	nonceStr := generateNonceStr()

	wxReq := &WechatQueryRequest{
		AppID:      p.config.AppID,
		MchID:      p.config.MchID,
		OutTradeNo: paymentID,
		NonceStr:   nonceStr,
	}

	wxReq.Sign = p.signQueryRequest(wxReq)

	xmlData, err := xml.Marshal(wxReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.gatewayURL+"/pay/orderquery", bytes.NewReader(xmlData))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/xml")

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var wxResp WechatQueryResponse
	if err := xml.Unmarshal(body, &wxResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if wxResp.ReturnCode != "SUCCESS" {
		return nil, fmt.Errorf("wechat error: %s", wxResp.ReturnMsg)
	}

	if wxResp.ResultCode != "SUCCESS" {
		return nil, fmt.Errorf("wechat error: %s - %s", wxResp.ErrCode, wxResp.ErrCodeDes)
	}

	status := p.mapTradeState(wxResp.TradeState)
	amount := decimal.NewFromInt(wxResp.TotalFee).Div(decimal.NewFromInt(100))

	result := &PaymentQueryResponse{
		PaymentID:  paymentID,
		ProviderID: wxResp.TransactionID,
		Status:     status,
		Amount:     amount,
	}

	if wxResp.TimeEnd != "" {
		paidAt, _ := time.Parse("20060102150405", wxResp.TimeEnd)
		result.PaidAt = &paidAt
	}

	if status == models.PaymentStatusFailed {
		result.FailureReason = wxResp.TradeStateDesc
	}

	return result, nil
}

// HandleCallback handles WeChat Pay notification
func (p *WechatProvider) HandleCallback(ctx context.Context, data map[string]string) (*PaymentCallbackResult, error) {
	// Verify signature
	if err := p.verifyCallback(data); err != nil {
		return nil, fmt.Errorf("signature verification failed: %w", err)
	}

	if data["return_code"] != "SUCCESS" || data["result_code"] != "SUCCESS" {
		return nil, fmt.Errorf("payment not successful: %s", data["err_code_des"])
	}

	totalFee, _ := decimal.NewFromString(data["total_fee"])
	amount := totalFee.Div(decimal.NewFromInt(100))

	paidAt, _ := time.Parse("20060102150405", data["time_end"])

	return &PaymentCallbackResult{
		PaymentID: data["out_trade_no"],
		Success:   true,
		Amount:    amount,
		PaidAt:    paidAt,
	}, nil
}

// RefundPayment refunds a payment
func (p *WechatProvider) RefundPayment(ctx context.Context, paymentID string, amount decimal.Decimal, reason string) error {
	// First query the original payment to get total fee
	queryResp, err := p.QueryPayment(ctx, paymentID)
	if err != nil {
		return fmt.Errorf("failed to query payment: %w", err)
	}

	totalFee := queryResp.Amount.Mul(decimal.NewFromInt(100)).IntPart()
	refundFee := amount.Mul(decimal.NewFromInt(100)).IntPart()

	nonceStr := generateNonceStr()

	wxReq := &WechatRefundRequest{
		AppID:       p.config.AppID,
		MchID:       p.config.MchID,
		NonceStr:    nonceStr,
		OutTradeNo:  paymentID,
		OutRefundNo: fmt.Sprintf("%s_%d", paymentID, time.Now().Unix()),
		TotalFee:    totalFee,
		RefundFee:   refundFee,
		RefundDesc:  reason,
	}

	wxReq.Sign = p.signRefundRequest(wxReq)

	xmlData, err := xml.Marshal(wxReq)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.gatewayURL+"/secapi/pay/refund", bytes.NewReader(xmlData))
	if err != nil {
		return err
	}
	httpReq.Header.Set("Content-Type", "application/xml")

	resp, err := p.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	var wxResp WechatRefundResponse
	if err := xml.Unmarshal(body, &wxResp); err != nil {
		return fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if wxResp.ReturnCode != "SUCCESS" {
		return fmt.Errorf("wechat error: %s", wxResp.ReturnMsg)
	}

	if wxResp.ResultCode != "SUCCESS" {
		return fmt.Errorf("refund failed: %s - %s", wxResp.ErrCode, wxResp.ErrCodeDes)
	}

	return nil
}

// signRequest signs a unified order request
func (p *WechatProvider) signRequest(req *WechatPayRequest) string {
	params := map[string]string{
		"appid":            req.AppID,
		"mch_id":           req.MchID,
		"nonce_str":        req.NonceStr,
		"body":             req.Body,
		"out_trade_no":     req.OutTradeNo,
		"total_fee":        fmt.Sprintf("%d", req.TotalFee),
		"spbill_create_ip": req.SpbillCreateIP,
		"notify_url":       req.NotifyURL,
		"trade_type":       req.TradeType,
	}
	if req.ProductID != "" {
		params["product_id"] = req.ProductID
	}
	return p.sign(params)
}

// signQueryRequest signs an order query request
func (p *WechatProvider) signQueryRequest(req *WechatQueryRequest) string {
	params := map[string]string{
		"appid":        req.AppID,
		"mch_id":       req.MchID,
		"nonce_str":    req.NonceStr,
		"out_trade_no": req.OutTradeNo,
	}
	return p.sign(params)
}

// signRefundRequest signs a refund request
func (p *WechatProvider) signRefundRequest(req *WechatRefundRequest) string {
	params := map[string]string{
		"appid":         req.AppID,
		"mch_id":        req.MchID,
		"nonce_str":     req.NonceStr,
		"out_trade_no":  req.OutTradeNo,
		"out_refund_no": req.OutRefundNo,
		"total_fee":     fmt.Sprintf("%d", req.TotalFee),
		"refund_fee":    fmt.Sprintf("%d", req.RefundFee),
	}
	if req.RefundDesc != "" {
		params["refund_desc"] = req.RefundDesc
	}
	return p.sign(params)
}

// sign generates MD5 signature for WeChat Pay
func (p *WechatProvider) sign(params map[string]string) string {
	// Sort keys
	keys := make([]string, 0, len(params))
	for k := range params {
		if params[k] != "" {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)

	// Build sign string
	var pairs []string
	for _, k := range keys {
		pairs = append(pairs, fmt.Sprintf("%s=%s", k, params[k]))
	}
	signStr := strings.Join(pairs, "&") + "&key=" + p.config.APIKey

	// MD5 hash
	hash := md5.Sum([]byte(signStr))
	return strings.ToUpper(hex.EncodeToString(hash[:]))
}

// signHMACSHA256 generates HMAC-SHA256 signature
func (p *WechatProvider) signHMACSHA256(params map[string]string) string {
	keys := make([]string, 0, len(params))
	for k := range params {
		if params[k] != "" {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)

	var pairs []string
	for _, k := range keys {
		pairs = append(pairs, fmt.Sprintf("%s=%s", k, params[k]))
	}
	signStr := strings.Join(pairs, "&") + "&key=" + p.config.APIKey

	h := hmac.New(sha256.New, []byte(p.config.APIKey))
	h.Write([]byte(signStr))
	return strings.ToUpper(hex.EncodeToString(h.Sum(nil)))
}

// verifyCallback verifies the callback signature
func (p *WechatProvider) verifyCallback(data map[string]string) error {
	sign := data["sign"]
	signType := data["sign_type"]

	// Remove sign and sign_type from params
	params := make(map[string]string)
	for k, v := range data {
		if k != "sign" && k != "sign_type" {
			params[k] = v
		}
	}

	var expectedSign string
	if signType == "HMAC-SHA256" {
		expectedSign = p.signHMACSHA256(params)
	} else {
		expectedSign = p.sign(params)
	}

	if sign != expectedSign {
		return errors.New("signature mismatch")
	}

	return nil
}

// mapTradeState maps WeChat trade state to PaymentStatus
func (p *WechatProvider) mapTradeState(state string) models.PaymentStatus {
	switch state {
	case "SUCCESS":
		return models.PaymentStatusSucceeded
	case "NOTPAY", "USERPAYING":
		return models.PaymentStatusPending
	case "CLOSED", "REVOKED", "PAYERROR":
		return models.PaymentStatusFailed
	case "REFUND":
		return models.PaymentStatusRefunded
	default:
		return models.PaymentStatusPending
	}
}

// generateNonceStr generates a random nonce string
func generateNonceStr() string {
	return strings.ReplaceAll(uuid.New().String(), "-", "")
}
