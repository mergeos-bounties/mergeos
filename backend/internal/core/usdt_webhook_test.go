package core

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMapGatewayStatus(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"pending", "pending"},
		{"waiting", "pending"},
		{"processing", "pending"},
		{"confirming", "pending"},
		{"confirmed", "confirmed"},
		{"completed", "confirmed"},
		{"success", "confirmed"},
		{"paid", "confirmed"},
		{"expired", "expired"},
		{"timeout", "expired"},
		{"failed", "failed"},
		{"error", "failed"},
		{"cancelled", "failed"},
		{"refunded", "refunded"},
		{"reversed", "refunded"},
		{"unknown_status", ""},
	}

	for _, tc := range tests {
		got := mapGatewayStatus(tc.input)
		if got != tc.expected {
			t.Errorf("mapGatewayStatus(%q) = %q, want %q", tc.input, got, tc.expected)
		}
	}
}

func TestParseAmountToCents(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
		hasError bool
	}{
		{"100.00", 10000, false},
		{"50.50", 5050, false},
		{"1.99", 199, false},
		{"0.01", 1, false},
		{"1000", 100000, false},
		{"0", 0, false},
		{"", 0, true},
		{"abc", 0, true},
	}

	for _, tc := range tests {
		got, err := parseAmountToCents(tc.input)
		if tc.hasError && err == nil {
			t.Errorf("parseAmountToCents(%q): expected error, got none", tc.input)
		}
		if !tc.hasError && err != nil {
			t.Errorf("parseAmountToCents(%q): unexpected error: %v", tc.input, err)
		}
		if got != tc.expected {
			t.Errorf("parseAmountToCents(%q) = %d, want %d", tc.input, got, tc.expected)
		}
	}
}

func TestVerifySignature(t *testing.T) {
	secret := "test-webhook-secret"
	cfg := Config{
		CryptoWebhookSecret: secret,
		DevPaymentEnabled: false,
	}
	h := &usdtWebhookHandler{cfg: cfg}

	body := []byte(`{"event_id":"evt_123","event_type":"payment.confirmed"}`)

	// Correct signature
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	correctSig := hex.EncodeToString(mac.Sum(nil))

	if err := h.verifySignature(body, "sha256="+correctSig); err != nil {
		t.Errorf("correct signature should pass: %v", err)
	}
	if err := h.verifySignature(body, correctSig); err != nil {
		t.Errorf("bare hex signature should pass: %v", err)
	}

	// Wrong signature
	if err := h.verifySignature(body, "sha256=badsignature"); err == nil {
		t.Error("wrong signature should fail")
	}

	// Missing signature
	if err := h.verifySignature(body, ""); err == nil {
		t.Error("missing signature should fail")
	}
}

func TestVerifySignatureDevMode(t *testing.T) {
	cfg := Config{
		CryptoWebhookSecret: "",
		DevPaymentEnabled: true,
	}
	h := &usdtWebhookHandler{cfg: cfg}

	body := []byte(`{"event_id":"evt_123"}`)
	// In dev mode with no secret, signature check should pass
	if err := h.verifySignature(body, ""); err != nil {
		t.Errorf("dev mode should skip signature verification: %v", err)
	}
}

func TestUSDTWebhookHandler_BadMethod(t *testing.T) {
	cfg := Config{DevPaymentEnabled: true}
	store := newTestStore(t, cfg)
	h := newUSDTWebhookHandler(cfg, store)

	req := httptest.NewRequest(http.MethodGet, "/api/payments/usdt/webhook", nil)
	w := httptest.NewRecorder()
	h.handleUSDTWebhook(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

func TestUSDTWebhookHandler_MissingSignature(t *testing.T) {
	cfg := Config{
		CryptoWebhookSecret: "secret123",
		DevPaymentEnabled: false,
	}
	store := newTestStore(t, cfg)
	h := newUSDTWebhookHandler(cfg, store)

	body := []byte(`{"event_id":"evt_1"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/payments/usdt/webhook", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.handleUSDTWebhook(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for missing signature, got %d", w.Code)
	}
}

func TestUSDTWebhookHandler_InvalidJSON(t *testing.T) {
	cfg := Config{DevPaymentEnabled: true}
	store := newTestStore(t, cfg)
	h := newUSDTWebhookHandler(cfg, store)

	body := []byte(`not json`)
	req := httptest.NewRequest(http.MethodPost, "/api/payments/usdt/webhook", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.handleUSDTWebhook(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid JSON, got %d", w.Code)
	}
}

func TestUSDTWebhookHandler_MissingEventID(t *testing.T) {
	cfg := Config{DevPaymentEnabled: true}
	store := newTestStore(t, cfg)
	h := newUSDTWebhookHandler(cfg, store)

	payload := USDTWebhookPayload{
		IdempotencyKey: "idem_1",
		EventType:      "payment.confirmed",
		Status:         "confirmed",
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/payments/usdt/webhook", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.handleUSDTWebhook(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing event_id, got %d", w.Code)
	}
}

func TestUSDTWebhookHandler_ValidConfirmed(t *testing.T) {
	cfg := Config{
		DevPaymentEnabled: true,
		TokenSymbol:       "MRG",
		PlatformFeeBps:    1000,
		CryptoReceiver:    "0xabc123",
	}
	store := newTestStore(t, cfg)
	h := newUSDTWebhookHandler(cfg, store)

	payload := USDTWebhookPayload{
		EventID:        "evt_confirmed_1",
		EventType:      "payment.confirmed",
		IdempotencyKey: "idem_confirmed_1",
		Amount:         "100.00",
		Currency:       "USD",
		Network:        "ethereum",
		TxHash:         "0xabc",
		Sender:         "0xsender",
		Receiver:       "0xabc123",
		Status:         "confirmed",
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/payments/usdt/webhook", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.handleUSDTWebhook(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp USDTWebhookResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if !resp.Received {
		t.Error("expected Received=true")
	}
	if resp.Status != "confirmed" {
		t.Errorf("expected status confirmed, got %s", resp.Status)
	}
}

func TestUSDTWebhookHandler_ReceiverMismatch(t *testing.T) {
	cfg := Config{
		DevPaymentEnabled: true,
		CryptoReceiver:    "0xexpected",
	}
	store := newTestStore(t, cfg)
	h := newUSDTWebhookHandler(cfg, store)

	payload := USDTWebhookPayload{
		EventID:        "evt_mismatch",
		EventType:      "payment.confirmed",
		IdempotencyKey: "idem_mismatch_1",
		Amount:         "100.00",
		Currency:       "USD",
		Network:        "ethereum",
		Receiver:       "0xwrong",
		Status:         "confirmed",
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/payments/usdt/webhook", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.handleUSDTWebhook(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for receiver mismatch, got %d", w.Code)
	}
}

func TestUSDTWebhookHandler_IdempotencyDuplicate(t *testing.T) {
	cfg := Config{
		DevPaymentEnabled: true,
		CryptoReceiver:    "0xabc123",
	}
	store := newTestStore(t, cfg)
	h := newUSDTWebhookHandler(cfg, store)

	// First request
	payload := USDTWebhookPayload{
		EventID:        "evt_dup",
		EventType:      "payment.confirmed",
		IdempotencyKey: "idem_dup_1",
		Amount:         "50.00",
		Currency:       "USD",
		Network:        "ethereum",
		Receiver:       "0xabc123",
		Status:         "confirmed",
	}
	body1, _ := json.Marshal(payload)
	req1 := httptest.NewRequest(http.MethodPost, "/api/payments/usdt/webhook", bytes.NewReader(body1))
	req1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	h.handleUSDTWebhook(w1, req1)

	if w1.Code != http.StatusOK {
		t.Fatalf("first request: expected 200, got %d", w1.Code)
	}

	// Duplicate request with same idempotency key
	payload2 := payload
	payload2.EventID = "evt_dup_2"
	body2, _ := json.Marshal(payload2)
	req2 := httptest.NewRequest(http.MethodPost, "/api/payments/usdt/webhook", bytes.NewReader(body2))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	h.handleUSDTWebhook(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("duplicate request: expected 200, got %d", w2.Code)
	}

	var resp USDTWebhookResponse
	if err := json.Unmarshal(w2.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp.Message != "duplicate webhook already processed" {
		t.Errorf("expected duplicate message, got: %s", resp.Message)
	}
}

// newTestStore creates a minimal in-memory store for webhook tests.
func newTestStore(t *testing.T, cfg Config) *Store {
	t.Helper()
	store, err := NewStore(cfg, NewPaymentManager(cfg), nil, NewEmailSender(cfg))
	if err != nil {
		t.Fatalf("failed to create test store: %v", err)
	}
	return store
}
