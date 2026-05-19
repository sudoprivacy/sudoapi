// sudoapi: Fuiou Pay payment provider integration.

package provider

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/payment"
)

// generateFuiouTestKeyPair returns a (merchant-private-key, fuiou-public-key)
// pair encoded the way the Fuiou onboarding portal hands them out: PKCS8 for
// the merchant private key, PKIX for the gateway public key, both base64-encoded.
func generateFuiouTestKeyPair(t *testing.T) (merchantPriPKCS8B64, fuiouPubPKIXB64 string, fuiouPri *rsa.PrivateKey, merchantPri *rsa.PrivateKey) {
	t.Helper()
	mPri, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate merchant key: %v", err)
	}
	fPri, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate fuiou key: %v", err)
	}
	mPKCS8, err := x509.MarshalPKCS8PrivateKey(mPri)
	if err != nil {
		t.Fatalf("marshal merchant pkcs8: %v", err)
	}
	fPKIX, err := x509.MarshalPKIXPublicKey(&fPri.PublicKey)
	if err != nil {
		t.Fatalf("marshal fuiou pkix: %v", err)
	}
	return base64.StdEncoding.EncodeToString(mPKCS8),
		base64.StdEncoding.EncodeToString(fPKIX),
		fPri,
		mPri
}

func newTestFuiou(t *testing.T, apiBase string) *Fuiou {
	t.Helper()
	merchantB64, fuiouB64, _, _ := generateFuiouTestKeyPair(t)
	p, err := NewFuiou("inst-1", map[string]string{
		"apiBase":            apiBase,
		"mchntCd":            "0008000F1234567",
		"fuiouPublicKey":     fuiouB64,
		"merchantPrivateKey": merchantB64,
		"notifyUrl":          "https://example.com/api/v1/payment/webhook/fuiou",
		"returnUrl":          "https://example.com/payment/result",
		"currency":           "CNY",
	})
	if err != nil {
		t.Fatalf("NewFuiou: %v", err)
	}
	return p
}

func TestFuiouRSAEncryptDecryptRoundTrip(t *testing.T) {
	t.Parallel()
	pri, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	plaintext := []byte(strings.Repeat("a", 600)) // exceeds single block
	cipher, err := fuiouRSAEncrypt(&pri.PublicKey, plaintext)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	if len(cipher)%pri.Size() != 0 {
		t.Fatalf("ciphertext length %d not a multiple of key size %d", len(cipher), pri.Size())
	}
	got, err := fuiouRSADecrypt(pri, cipher)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if string(got) != string(plaintext) {
		t.Fatalf("round-trip mismatch")
	}
}

func TestFuiouAmountToCents(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in      string
		want    int64
		wantErr bool
	}{
		{"1", 100, false},
		{"1.00", 100, false},
		{"12.34", 1234, false},
		{"0.01", 1, false},
		{"0", 0, true},
		{"-1", 0, true},
		{"abc", 0, true},
		{"1.234", 0, true},
	}
	for _, tc := range cases {
		got, err := fuiouAmountToCents(tc.in)
		if tc.wantErr && err == nil {
			t.Errorf("fuiouAmountToCents(%q) expected error, got %d", tc.in, got)
		}
		if !tc.wantErr {
			if err != nil {
				t.Errorf("fuiouAmountToCents(%q): %v", tc.in, err)
				continue
			}
			if got != tc.want {
				t.Errorf("fuiouAmountToCents(%q) = %d, want %d", tc.in, got, tc.want)
			}
		}
	}
}

func TestFuiouMapPaymentType(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		"alipay":        fuiouOrderPayAlipay,
		"alipay_direct": fuiouOrderPayAlipay,
		"wxpay":         fuiouOrderPayWechat,
		"wxpay_direct":  fuiouOrderPayWechat,
	}
	for in, want := range cases {
		got, err := fuiouMapPaymentType(in)
		if err != nil {
			t.Errorf("fuiouMapPaymentType(%q): %v", in, err)
			continue
		}
		if got != want {
			t.Errorf("fuiouMapPaymentType(%q) = %q, want %q", in, got, want)
		}
	}
	if _, err := fuiouMapPaymentType("stripe"); err == nil {
		t.Errorf("expected error for unsupported type")
	}
}

func TestFuiouMapStatus(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		fuiouOrderStatusPaid:  payment.ProviderStatusSuccess,
		fuiouOrderStatusExpd:  payment.ProviderStatusFailed,
		fuiouOrderStatusFaild: payment.ProviderStatusFailed,
		"0":                   payment.ProviderStatusPending,
		"":                    payment.ProviderStatusPending,
	}
	for in, want := range cases {
		if got := fuiouMapStatus(in); got != want {
			t.Errorf("fuiouMapStatus(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestFuiouSplitOrderInfo(t *testing.T) {
	t.Parallel()
	cases := []struct {
		info, payType string
		wantURL       string
		wantQR        string
	}{
		{"https://qr.alipay.com/abc", fuiouOrderPayAlipay, "https://qr.alipay.com/abc", "https://qr.alipay.com/abc"},
		{"http://example/pay", fuiouOrderPayAlipay, "http://example/pay", "http://example/pay"},
		{"weixin://wxpay/bizpayurl?abc=1", fuiouOrderPayWechat, "", "weixin://wxpay/bizpayurl?abc=1"},
		{"raw-code", fuiouOrderPayWechat, "", "raw-code"},
		{"", fuiouOrderPayAlipay, "", ""},
	}
	for _, tc := range cases {
		gotURL, gotQR := fuiouSplitOrderInfo(tc.info, tc.payType)
		if gotURL != tc.wantURL || gotQR != tc.wantQR {
			t.Errorf("split(%q,%q) = (%q,%q), want (%q,%q)", tc.info, tc.payType, gotURL, gotQR, tc.wantURL, tc.wantQR)
		}
	}
}

func TestFuiouMerchantIdentityMetadata(t *testing.T) {
	t.Parallel()
	f := &Fuiou{config: map[string]string{"mchntCd": "0008000F1234567"}}
	if got := f.MerchantIdentityMetadata(); got["mchnt_cd"] != "0008000F1234567" {
		t.Fatalf("MerchantIdentityMetadata = %v, want mchnt_cd=0008000F1234567", got)
	}
	empty := &Fuiou{config: map[string]string{}}
	if got := empty.MerchantIdentityMetadata(); got != nil {
		t.Fatalf("expected nil metadata for empty mchntCd, got %v", got)
	}
}

func TestFuiouCreatePaymentRoundTrip(t *testing.T) {
	t.Parallel()

	// Build the provider first so we can encrypt the canned gateway response
	// against the merchant public key it knows how to decrypt.
	merchantB64, fuiouB64, fuiouPri, merchantPri := generateFuiouTestKeyPair(t)

	respPlain := fuiouOrderResponse{
		OrderDate:    "20260519",
		OrderPayType: fuiouOrderPayAlipay,
		OrderAmt:     "100",
		MchntCD:      "0008000F1234567",
		OrderID:      "ORD123",
		OrderInfo:    "https://qr.alipay.com/abc",
	}
	respPlainBytes, _ := json.Marshal(&respPlain)
	respCipher, err := fuiouRSAEncrypt(&merchantPri.PublicKey, respPlainBytes)
	if err != nil {
		t.Fatalf("encrypt server response: %v", err)
	}
	respBytes, _ := json.Marshal(&fuiouEnvelope{
		MchntCD:  "0008000F1234567",
		Message:  respCipher,
		RespCode: fuiouRespCodeSuccess,
	})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != fuiouOrderPath {
			http.Error(w, "wrong path", http.StatusNotFound)
			return
		}
		body, _ := io.ReadAll(r.Body)
		var env fuiouEnvelope
		if err := json.Unmarshal(body, &env); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		// Validate the request shape — the provider must decrypt with the
		// fuiou private key (gateway side) we generated above.
		plain, err := fuiouRSADecrypt(fuiouPri, env.Message)
		if err != nil {
			http.Error(w, "decrypt: "+err.Error(), http.StatusBadRequest)
			return
		}
		var sentReq fuiouOrderRequest
		if err := json.Unmarshal(plain, &sentReq); err != nil {
			http.Error(w, "parse: "+err.Error(), http.StatusBadRequest)
			return
		}
		if sentReq.OrderID != "ORD123" || sentReq.OrderAmt != "100" || sentReq.OrderPayType != fuiouOrderPayAlipay {
			http.Error(w, "unexpected request payload", http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(respBytes)
	}))
	defer server.Close()

	p, err := NewFuiou("inst-1", map[string]string{
		"apiBase":            server.URL,
		"mchntCd":            "0008000F1234567",
		"fuiouPublicKey":     fuiouB64,
		"merchantPrivateKey": merchantB64,
		"notifyUrl":          "https://example.com/api/v1/payment/webhook/fuiou",
		"returnUrl":          "https://example.com/payment/result",
		"currency":           "CNY",
	})
	if err != nil {
		t.Fatalf("NewFuiou: %v", err)
	}

	got, err := p.CreatePayment(context.Background(), payment.CreatePaymentRequest{
		OrderID:     "ORD123",
		Amount:      "1.00",
		PaymentType: "alipay",
		Subject:     "Test",
		NotifyURL:   "https://example.com/api/v1/payment/webhook/fuiou",
		ReturnURL:   "https://example.com/payment/result",
	})
	if err != nil {
		t.Fatalf("CreatePayment: %v", err)
	}
	if got.TradeNo != "ORD123" {
		t.Errorf("TradeNo = %q, want ORD123", got.TradeNo)
	}
	if got.PayURL != "https://qr.alipay.com/abc" {
		t.Errorf("PayURL = %q", got.PayURL)
	}
	if got.QRCode != "https://qr.alipay.com/abc" {
		t.Errorf("QRCode = %q", got.QRCode)
	}
	if got.Currency != "CNY" {
		t.Errorf("Currency = %q", got.Currency)
	}
}

func TestFuiouVerifyNotification(t *testing.T) {
	t.Parallel()
	p := newTestFuiou(t, "https://example.com")

	// Build a callback envelope: server encrypts the message with merchant
	// public key, provider decrypts with merchant private key.
	msg := fuiouCallbackMessage{
		OrderID:   "ORD123",
		OrderSt:   fuiouOrderStatusPaid,
		OrderAmt:  "1234",
		OrderDate: "20260519",
		MchntCD:   "0008000F1234567",
	}
	plain, _ := json.Marshal(&msg)
	cipher, err := fuiouRSAEncrypt(&p.merchantPri.PublicKey, plain)
	if err != nil {
		t.Fatalf("encrypt cb: %v", err)
	}
	envelope := fuiouEnvelope{
		MchntCD:  "0008000F1234567",
		Message:  cipher,
		RespCode: fuiouRespCodeSuccess,
	}
	raw, _ := json.Marshal(&envelope)

	got, err := p.VerifyNotification(context.Background(), string(raw), nil)
	if err != nil {
		t.Fatalf("VerifyNotification: %v", err)
	}
	if got == nil {
		t.Fatal("nil notification")
	}
	if got.OrderID != "ORD123" || got.Status != payment.ProviderStatusSuccess {
		t.Errorf("notification = %+v", got)
	}
	if got.Amount != 12.34 {
		t.Errorf("Amount = %v, want 12.34", got.Amount)
	}
	if got.Metadata["mchnt_cd"] != "0008000F1234567" {
		t.Errorf("metadata = %+v", got.Metadata)
	}
}

func TestFuiouVerifyNotificationRejectsMchntMismatch(t *testing.T) {
	t.Parallel()
	p := newTestFuiou(t, "https://example.com")
	envelope := fuiouEnvelope{
		MchntCD:  "WRONG_MCHNT",
		Message:  []byte{0x01},
		RespCode: fuiouRespCodeSuccess,
	}
	raw, _ := json.Marshal(&envelope)
	if _, err := p.VerifyNotification(context.Background(), string(raw), nil); err == nil {
		t.Fatal("expected mchnt_cd mismatch error")
	}
}

func TestFuiouNewRejectsMissingFields(t *testing.T) {
	t.Parallel()
	if _, err := NewFuiou("x", map[string]string{}); err == nil {
		t.Fatal("expected error for empty config")
	}
	if _, err := NewFuiou("x", map[string]string{
		"apiBase":            "https://example.com",
		"mchntCd":            "M",
		"fuiouPublicKey":     "not-base64!",
		"merchantPrivateKey": "not-base64!",
	}); err == nil {
		t.Fatal("expected error for invalid keys")
	}
}

func TestFuiouNormalizeAPIBase(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in      string
		want    string
		wantErr bool
	}{
		{"https://hlwnets.fuioupay.com/", "https://hlwnets.fuioupay.com", false},
		{" https://hlwnets-test.fuioupay.com/aggpos/ ", "https://hlwnets-test.fuioupay.com/aggpos", false},
		{"", "", true},
		{"not-a-url", "", true},
	}
	for _, tc := range cases {
		got, err := normalizeFuiouAPIBase(tc.in)
		if tc.wantErr {
			if err == nil {
				t.Errorf("normalizeFuiouAPIBase(%q) expected error", tc.in)
			}
			continue
		}
		if err != nil {
			t.Errorf("normalizeFuiouAPIBase(%q): %v", tc.in, err)
			continue
		}
		if got != tc.want {
			t.Errorf("normalizeFuiouAPIBase(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
