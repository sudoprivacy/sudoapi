// sudoapi: Fuiou Pay payment provider integration.

package provider

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/shopspring/decimal"

	"github.com/Wei-Shaw/sub2api/internal/payment"
)

// Fuiou (富友支付) constants.
const (
	fuiouHTTPTimeout      = 15 * time.Second
	fuiouMaxResponseSize  = 1 << 20
	fuiouMaxErrorSummary  = 512
	fuiouAPIVersion       = "1.0.0"
	fuiouRespCodeSuccess  = "0000"
	fuiouOrderPath        = "/aggpos/order.fuiou"
	fuiouQueryPath        = "/aggpos/query.fuiou"
	fuiouRefundPath       = "/aggpos/refund.fuiou"
	fuiouOrderPayAlipay   = "ALIPAY"
	fuiouOrderPayWechat   = "WECHAT"
	fuiouOrderStatusPaid  = "1"
	fuiouOrderStatusExpd  = "2"
	fuiouOrderStatusFaild = "3"
)

// fuiouMinorUnit is the divisor used to convert Fuiou's "cents" amounts back to
// major currency units. Fuiou always quotes order_amt in 分 (CNY minor units),
// regardless of the configured display currency.
var fuiouMinorUnit = decimal.NewFromInt(100)

// Fuiou implements payment.Provider for 富友聚合支付.
//
// Configuration keys (all lower-camelCase to match other providers):
//
//	apiBase            — Fuiou API root, e.g. https://hlwnets.fuioupay.com (prod)
//	                     or https://hlwnets-test.fuioupay.com (sandbox).
//	mchntCd            — 富友商户号 (used as instance identity for webhook routing).
//	fuiouPublicKey     — Base64 PKIX-encoded RSA public key issued by Fuiou.
//	merchantPrivateKey — Base64 PKCS8-encoded RSA private key of the merchant.
//	notifyUrl          — Async webhook URL (filled in by admin UI).
//	returnUrl          — Browser return URL after payment.
//	currency           — Display currency (Fuiou itself is CNY-only; this is the
//	                     currency the order is recorded under).
type Fuiou struct {
	instanceID string
	config     map[string]string
	httpClient *http.Client

	fuiouPub    *rsa.PublicKey
	merchantPri *rsa.PrivateKey
}

// NewFuiou constructs a new Fuiou provider from a config map.
func NewFuiou(instanceID string, config map[string]string) (*Fuiou, error) {
	required := []string{"apiBase", "mchntCd", "fuiouPublicKey", "merchantPrivateKey"}
	for _, k := range required {
		if strings.TrimSpace(config[k]) == "" {
			return nil, fmt.Errorf("fuiou config missing required key: %s", k)
		}
	}
	cfg := cloneStringMap(config)
	apiBase, err := normalizeFuiouAPIBase(cfg["apiBase"])
	if err != nil {
		return nil, err
	}
	cfg["apiBase"] = apiBase

	currency, err := payment.NormalizePaymentCurrency(cfg["currency"])
	if err != nil {
		return nil, fmt.Errorf("fuiou config currency: %w", err)
	}
	cfg["currency"] = currency

	pub, err := parseFuiouPublicKey(cfg["fuiouPublicKey"])
	if err != nil {
		return nil, fmt.Errorf("fuiou config fuiouPublicKey: %w", err)
	}
	pri, err := parseFuiouPrivateKey(cfg["merchantPrivateKey"])
	if err != nil {
		return nil, fmt.Errorf("fuiou config merchantPrivateKey: %w", err)
	}

	return &Fuiou{
		instanceID:  instanceID,
		config:      cfg,
		httpClient:  &http.Client{Timeout: fuiouHTTPTimeout},
		fuiouPub:    pub,
		merchantPri: pri,
	}, nil
}

func normalizeFuiouAPIBase(raw string) (string, error) {
	base := strings.TrimSpace(raw)
	if base == "" {
		return "", fmt.Errorf("fuiou apiBase is required")
	}
	parsed, err := url.Parse(base)
	if err != nil || (parsed.Scheme != "https" && parsed.Scheme != "http") || parsed.Host == "" {
		return "", fmt.Errorf("fuiou apiBase must be a valid HTTP(S) URL")
	}
	parsed.RawQuery = ""
	parsed.Fragment = ""
	parsed.RawPath = ""
	parsed.Path = strings.TrimRight(parsed.Path, "/")
	return parsed.String(), nil
}

func parseFuiouPublicKey(b64 string) (*rsa.PublicKey, error) {
	keyBytes, err := base64.StdEncoding.DecodeString(strings.TrimSpace(b64))
	if err != nil {
		return nil, fmt.Errorf("decode public key: %w", err)
	}
	pub, err := x509.ParsePKIXPublicKey(keyBytes)
	if err != nil {
		return nil, fmt.Errorf("parse PKIX public key: %w", err)
	}
	rsaPub, ok := pub.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("public key is not an RSA key")
	}
	return rsaPub, nil
}

func parseFuiouPrivateKey(b64 string) (*rsa.PrivateKey, error) {
	keyBytes, err := base64.StdEncoding.DecodeString(strings.TrimSpace(b64))
	if err != nil {
		return nil, fmt.Errorf("decode private key: %w", err)
	}
	pri, err := x509.ParsePKCS8PrivateKey(keyBytes)
	if err != nil {
		return nil, fmt.Errorf("parse PKCS8 private key: %w", err)
	}
	rsaPri, ok := pri.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("private key is not an RSA key")
	}
	return rsaPri, nil
}

func (f *Fuiou) Name() string        { return "富友支付" }
func (f *Fuiou) ProviderKey() string { return payment.TypeFuiou }
func (f *Fuiou) SupportedTypes() []payment.PaymentType {
	return []payment.PaymentType{payment.TypeAlipay, payment.TypeWxpay}
}

// MerchantIdentityMetadata exposes the merchant code so that order snapshots
// and webhook-instance lookup can identify the originating Fuiou account.
func (f *Fuiou) MerchantIdentityMetadata() map[string]string {
	if f == nil {
		return nil
	}
	mchnt := strings.TrimSpace(f.config["mchntCd"])
	if mchnt == "" {
		return nil
	}
	return map[string]string{"mchnt_cd": mchnt}
}

// fuiouOrderRequest mirrors the body Fuiou expects inside the encrypted message.
type fuiouOrderRequest struct {
	MchntCD       string `json:"mchnt_cd"`
	OrderDate     string `json:"order_date"`
	OrderID       string `json:"order_id"`
	OrderAmt      string `json:"order_amt"`
	OrderPayType  string `json:"order_pay_type"`
	BackNotifyURL string `json:"back_notify_url"`
	GoodsName     string `json:"goods_name"`
	GoodsDetail   string `json:"goods_detail"`
	Ver           string `json:"ver"`
}

// fuiouEnvelope is the outer Fuiou request/response wrapper.
type fuiouEnvelope struct {
	MchntCD  string `json:"mchnt_cd"`
	Message  []byte `json:"message"`
	RespCode string `json:"resp_code,omitempty"`
	RespDesc string `json:"resp_desc,omitempty"`
}

// fuiouOrderResponse describes the decrypted body Fuiou returns from the order
// endpoint. order_info carries the pay URL / QR payload — its exact shape
// depends on the channel (alipay/wechat) so we surface it raw to the caller.
type fuiouOrderResponse struct {
	OrderDate    string `json:"order_date"`
	OrderPayType string `json:"order_pay_type"`
	OrderAmt     string `json:"order_amt"`
	MchntCD      string `json:"mchnt_cd"`
	OrderID      string `json:"order_id"`
	OrderInfo    string `json:"order_info"`
}

// fuiouQueryResponse describes the decrypted body of the query endpoint.
type fuiouQueryResponse struct {
	OrderDate string `json:"order_date"`
	OrderID   string `json:"order_id"`
	OrderAmt  string `json:"order_amt"`
	OrderSt   string `json:"order_st"`
	MchntCD   string `json:"mchnt_cd"`
}

// fuiouRefundRequest is the decrypted body of the refund endpoint.
type fuiouRefundRequest struct {
	MchntCD     string `json:"mchnt_cd"`
	OrderID     string `json:"order_id"`
	OrderDate   string `json:"order_date"`
	RefundAmt   string `json:"refund_amt"`
	RefundID    string `json:"refund_id"`
	RefundDesc  string `json:"refund_desc,omitempty"`
	OrigOrderID string `json:"orig_order_id,omitempty"`
	Ver         string `json:"ver"`
}

type fuiouRefundResponse struct {
	OrderID    string `json:"order_id"`
	RefundID   string `json:"refund_id"`
	RefundSt   string `json:"refund_st"`
	RefundAmt  string `json:"refund_amt"`
	OrderDate  string `json:"order_date"`
	RefundDesc string `json:"refund_desc"`
}

// fuiouCallbackMessage is the decrypted body of an async webhook.
type fuiouCallbackMessage struct {
	OrderID   string `json:"order_id"`
	OrderSt   string `json:"order_st"`
	OrderAmt  string `json:"order_amt"`
	OrderDate string `json:"order_date"`
	MchntCD   string `json:"mchnt_cd"`
}

func (f *Fuiou) CreatePayment(ctx context.Context, req payment.CreatePaymentRequest) (*payment.CreatePaymentResponse, error) {
	payType, err := fuiouMapPaymentType(req.PaymentType)
	if err != nil {
		return nil, err
	}
	amountCents, err := fuiouAmountToCents(req.Amount)
	if err != nil {
		return nil, err
	}

	notifyURL := strings.TrimSpace(req.NotifyURL)
	if notifyURL == "" {
		notifyURL = strings.TrimSpace(f.config["notifyUrl"])
	}
	if notifyURL == "" {
		return nil, fmt.Errorf("fuiou create payment: notify URL is required")
	}

	body := fuiouOrderRequest{
		MchntCD:       f.config["mchntCd"],
		OrderDate:     time.Now().Format("20060102"),
		OrderID:       req.OrderID,
		OrderAmt:      strconv.FormatInt(amountCents, 10),
		OrderPayType:  payType,
		BackNotifyURL: notifyURL,
		GoodsName:     defaultIfBlank(req.Subject, req.OrderID),
		GoodsDetail:   defaultIfBlank(req.Subject, req.OrderID),
		Ver:           fuiouAPIVersion,
	}

	var resp fuiouOrderResponse
	if err := f.doEncryptedRequest(ctx, fuiouOrderPath, body, &resp); err != nil {
		return nil, fmt.Errorf("fuiou create payment: %w", err)
	}

	payURL, qrCode := fuiouSplitOrderInfo(resp.OrderInfo, payType)
	return &payment.CreatePaymentResponse{
		TradeNo:    resp.OrderID,
		PayURL:     payURL,
		QRCode:     qrCode,
		Currency:   f.currency(),
		ResultType: payment.CreatePaymentResultOrderCreated,
	}, nil
}

func (f *Fuiou) QueryOrder(ctx context.Context, tradeNo string) (*payment.QueryOrderResponse, error) {
	orderID := strings.TrimSpace(tradeNo)
	if orderID == "" {
		return nil, fmt.Errorf("fuiou query order: missing order id")
	}
	body := map[string]string{
		"mchnt_cd":   f.config["mchntCd"],
		"order_id":   orderID,
		"order_date": time.Now().Format("20060102"),
		"ver":        fuiouAPIVersion,
	}
	var resp fuiouQueryResponse
	if err := f.doEncryptedRequest(ctx, fuiouQueryPath, body, &resp); err != nil {
		return nil, fmt.Errorf("fuiou query order: %w", err)
	}
	amount, _ := strconv.ParseFloat(resp.OrderAmt, 64)
	return &payment.QueryOrderResponse{
		TradeNo:  resp.OrderID,
		Status:   fuiouMapStatus(resp.OrderSt),
		Amount:   amount / 100.0,
		Metadata: f.MerchantIdentityMetadata(),
	}, nil
}

// VerifyNotification decrypts the callback message, validates that the merchant
// code matches this instance, and maps the order status. Unknown / non-final
// statuses are surfaced as failed so callers can distinguish them from success.
func (f *Fuiou) VerifyNotification(_ context.Context, rawBody string, _ map[string]string) (*payment.PaymentNotification, error) {
	var envelope fuiouEnvelope
	if err := json.Unmarshal([]byte(rawBody), &envelope); err != nil {
		return nil, fmt.Errorf("fuiou parse webhook envelope: %w", err)
	}
	if envelope.RespCode != "" && envelope.RespCode != fuiouRespCodeSuccess {
		return nil, fmt.Errorf("fuiou webhook resp_code %s: %s", envelope.RespCode, envelope.RespDesc)
	}
	if expected := strings.TrimSpace(f.config["mchntCd"]); expected != "" &&
		strings.TrimSpace(envelope.MchntCD) != expected {
		return nil, fmt.Errorf("fuiou webhook mchnt_cd mismatch: expected %s, got %s", expected, envelope.MchntCD)
	}
	plain, err := f.decryptMessage(envelope.Message)
	if err != nil {
		return nil, fmt.Errorf("fuiou decrypt webhook: %w", err)
	}
	var msg fuiouCallbackMessage
	if err := json.Unmarshal(plain, &msg); err != nil {
		return nil, fmt.Errorf("fuiou parse webhook message: %w", err)
	}
	if strings.TrimSpace(msg.OrderID) == "" {
		return nil, fmt.Errorf("fuiou webhook missing order_id")
	}
	status := fuiouMapStatus(msg.OrderSt)
	amount, _ := strconv.ParseFloat(msg.OrderAmt, 64)
	metadata := f.MerchantIdentityMetadata()
	if metadata == nil {
		metadata = map[string]string{}
	}
	if mchnt := strings.TrimSpace(msg.MchntCD); mchnt != "" {
		metadata["mchnt_cd"] = mchnt
	}
	return &payment.PaymentNotification{
		TradeNo:  msg.OrderID,
		OrderID:  msg.OrderID,
		Amount:   amount / 100.0,
		Status:   status,
		RawData:  rawBody,
		Metadata: metadata,
	}, nil
}

func (f *Fuiou) Refund(ctx context.Context, req payment.RefundRequest) (*payment.RefundResponse, error) {
	orderID := strings.TrimSpace(req.OrderID)
	if orderID == "" {
		orderID = strings.TrimSpace(req.TradeNo)
	}
	if orderID == "" {
		return nil, fmt.Errorf("fuiou refund: missing order id")
	}
	amountCents, err := fuiouAmountToCents(req.Amount)
	if err != nil {
		return nil, fmt.Errorf("fuiou refund: %w", err)
	}
	refundID := fmt.Sprintf("R%s%d", orderID, time.Now().UnixNano())
	if len(refundID) > 32 {
		refundID = refundID[:32]
	}
	body := fuiouRefundRequest{
		MchntCD:     f.config["mchntCd"],
		OrderID:     refundID,
		OrderDate:   time.Now().Format("20060102"),
		RefundAmt:   strconv.FormatInt(amountCents, 10),
		RefundID:    refundID,
		RefundDesc:  defaultIfBlank(req.Reason, "refund"),
		OrigOrderID: orderID,
		Ver:         fuiouAPIVersion,
	}
	var resp fuiouRefundResponse
	if err := f.doEncryptedRequest(ctx, fuiouRefundPath, body, &resp); err != nil {
		return nil, fmt.Errorf("fuiou refund: %w", err)
	}
	rid := strings.TrimSpace(resp.RefundID)
	if rid == "" {
		rid = refundID
	}
	return &payment.RefundResponse{
		RefundID: rid,
		Status:   fuiouMapRefundStatus(resp.RefundSt),
	}, nil
}

// --- Helpers ---

func (f *Fuiou) currency() string {
	c, err := payment.NormalizePaymentCurrency(f.config["currency"])
	if err != nil {
		return payment.DefaultPaymentCurrency
	}
	return c
}

// doEncryptedRequest serialises body → encrypts with Fuiou public key → posts →
// decrypts response with merchant private key → unmarshals into out.
func (f *Fuiou) doEncryptedRequest(ctx context.Context, path string, body any, out any) error {
	plaintext, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}
	cipher, err := fuiouRSAEncrypt(f.fuiouPub, plaintext)
	if err != nil {
		return fmt.Errorf("encrypt request: %w", err)
	}
	reqEnvelope := fuiouEnvelope{
		MchntCD: f.config["mchntCd"],
		Message: cipher,
	}
	reqBytes, err := json.Marshal(reqEnvelope)
	if err != nil {
		return fmt.Errorf("marshal envelope: %w", err)
	}

	endpoint := f.config["apiBase"] + path
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(reqBytes))
	if err != nil {
		return err
	}
	httpReq.Header.Set("Content-Type", "application/json;charset=UTF-8")

	client := f.httpClient
	if client == nil {
		client = &http.Client{Timeout: fuiouHTTPTimeout}
	}
	httpResp, err := client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("HTTP request: %w", err)
	}
	defer func() { _ = httpResp.Body.Close() }()

	respBytes, err := io.ReadAll(io.LimitReader(httpResp.Body, fuiouMaxResponseSize))
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}
	if httpResp.StatusCode < http.StatusOK || httpResp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("HTTP %d: %s", httpResp.StatusCode, summarizeFuiouResponse(respBytes))
	}

	var respEnvelope fuiouEnvelope
	if err := json.Unmarshal(respBytes, &respEnvelope); err != nil {
		return fmt.Errorf("parse response envelope: %w: %s", err, summarizeFuiouResponse(respBytes))
	}
	if respEnvelope.RespCode != fuiouRespCodeSuccess {
		return fmt.Errorf("resp_code %s: %s", respEnvelope.RespCode, respEnvelope.RespDesc)
	}
	if len(respEnvelope.Message) == 0 {
		return nil
	}
	plain, err := f.decryptMessage(respEnvelope.Message)
	if err != nil {
		return fmt.Errorf("decrypt response: %w", err)
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(plain, out); err != nil {
		return fmt.Errorf("parse decrypted response: %w", err)
	}
	return nil
}

func (f *Fuiou) decryptMessage(data []byte) ([]byte, error) {
	if f.merchantPri == nil {
		return nil, fmt.Errorf("merchant private key not configured")
	}
	return fuiouRSADecrypt(f.merchantPri, data)
}

// fuiouRSAEncrypt encrypts data with PKCS1v15, chunked at (keySize-11) bytes
// because PKCS1v15 reserves 11 bytes of padding per block.
func fuiouRSAEncrypt(pub *rsa.PublicKey, data []byte) ([]byte, error) {
	if pub == nil {
		return nil, fmt.Errorf("nil public key")
	}
	chunk := pub.Size() - 11
	if chunk <= 0 {
		return nil, fmt.Errorf("invalid public key size: %d", pub.Size())
	}
	var buf bytes.Buffer
	for i := 0; i < len(data); i += chunk {
		end := i + chunk
		if end > len(data) {
			end = len(data)
		}
		block, err := rsa.EncryptPKCS1v15(rand.Reader, pub, data[i:end])
		if err != nil {
			return nil, err
		}
		_, _ = buf.Write(block)
	}
	return buf.Bytes(), nil
}

// fuiouRSADecrypt is the inverse of fuiouRSAEncrypt, chunked at keySize bytes.
func fuiouRSADecrypt(pri *rsa.PrivateKey, data []byte) ([]byte, error) {
	if pri == nil {
		return nil, fmt.Errorf("nil private key")
	}
	chunk := pri.Size()
	if chunk <= 0 || len(data)%chunk != 0 {
		return nil, fmt.Errorf("invalid ciphertext length %d for key size %d", len(data), chunk)
	}
	var buf bytes.Buffer
	for i := 0; i < len(data); i += chunk {
		end := i + chunk
		block, err := rsa.DecryptPKCS1v15(rand.Reader, pri, data[i:end])
		if err != nil {
			return nil, err
		}
		_, _ = buf.Write(block)
	}
	return buf.Bytes(), nil
}

func fuiouMapPaymentType(t string) (string, error) {
	switch payment.GetBasePaymentType(t) {
	case payment.TypeAlipay:
		return fuiouOrderPayAlipay, nil
	case payment.TypeWxpay:
		return fuiouOrderPayWechat, nil
	default:
		return "", fmt.Errorf("fuiou unsupported payment type: %s", t)
	}
}

func fuiouMapStatus(orderSt string) string {
	switch strings.TrimSpace(orderSt) {
	case fuiouOrderStatusPaid:
		return payment.ProviderStatusSuccess
	case fuiouOrderStatusExpd:
		return payment.ProviderStatusFailed
	case fuiouOrderStatusFaild:
		return payment.ProviderStatusFailed
	default:
		return payment.ProviderStatusPending
	}
}

// fuiouMapRefundStatus maps Fuiou refund status codes to provider statuses.
// Fuiou uses the same family of codes as the order endpoint: "1" = success,
// "0"/"2"/"3" = pending or failed depending on the gateway, so we treat
// anything that is not an explicit success as pending.
func fuiouMapRefundStatus(refundSt string) string {
	switch strings.TrimSpace(refundSt) {
	case fuiouOrderStatusPaid:
		return payment.ProviderStatusSuccess
	case fuiouOrderStatusExpd, fuiouOrderStatusFaild:
		return payment.ProviderStatusFailed
	default:
		return payment.ProviderStatusPending
	}
}

// fuiouAmountToCents normalises req.Amount (a major-unit string like "12.34")
// into Fuiou's integer 分 representation. Fuiou is CNY-only, so we always use
// the 2-decimal minor-unit conversion.
func fuiouAmountToCents(raw string) (int64, error) {
	d, err := decimal.NewFromString(strings.TrimSpace(raw))
	if err != nil {
		return 0, fmt.Errorf("invalid amount: %s", raw)
	}
	if d.LessThanOrEqual(decimal.Zero) {
		return 0, fmt.Errorf("amount must be greater than zero")
	}
	cents := d.Mul(fuiouMinorUnit)
	if !cents.Equal(cents.Truncate(0)) {
		return 0, fmt.Errorf("amount %s exceeds two decimal places", raw)
	}
	return cents.IntPart(), nil
}

// fuiouSplitOrderInfo classifies the order_info Fuiou returns. Alipay returns
// a full http(s) pay URL; WeChat usually returns a code_url (weixin://...). We
// keep behaviour simple and predictable: expose URLs that look like HTTP(S) as
// PayURL and the rest as QRCode payloads. Both fields are populated for HTTP(S)
// URLs so that frontends can choose to render a QR code or redirect.
func fuiouSplitOrderInfo(orderInfo, payType string) (payURL string, qrCode string) {
	info := strings.TrimSpace(orderInfo)
	if info == "" {
		return "", ""
	}
	lower := strings.ToLower(info)
	switch {
	case strings.HasPrefix(lower, "http://"), strings.HasPrefix(lower, "https://"):
		return info, info
	case strings.HasPrefix(lower, "weixin://"), payType == fuiouOrderPayWechat:
		return "", info
	default:
		return "", info
	}
}

func defaultIfBlank(s, fallback string) string {
	if strings.TrimSpace(s) == "" {
		return fallback
	}
	return s
}

func summarizeFuiouResponse(body []byte) string {
	summary := strings.Join(strings.Fields(string(body)), " ")
	if summary == "" {
		return "<empty>"
	}
	if len(summary) > fuiouMaxErrorSummary {
		return summary[:fuiouMaxErrorSummary] + "..."
	}
	return summary
}
