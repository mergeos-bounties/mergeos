package core

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
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
		{"successful", "confirmed"},
		{"paid", "confirmed"},
		{"expired", "expired"},
		{"timeout", "expired"},
		{"timed_out", "expired"},
		{"failed", "failed"},
		{"error", "failed"},
		{"cancelled", "failed"},
		{"canceled", "failed"},
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
		{"10.999", 0, true}, // rejects extra decimals
		{"-10.00", 0, true}, // rejects negative values
		{"00.50", 0, true},  // rejects leading zeros
		{"10.", 0, true},    // rejects trailing dot
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
	body := []byte(`{"event_id":"evt_123","event_type":"payment.confirmed"}`)

	// Correct signature via header
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	correctSig := hex.EncodeToString(mac.Sum(nil))

	t.Run("correct signature sha256= format", func(t *testing.T) {
		p := NewGenericHMACProvider("test", secret, false)
		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
		req.Header.Set("X-Webhook-Signature", "sha256="+correctSig)
		if err := p.VerifySignature(body, req); err != nil {
			t.Errorf("expected pass: %v", err)
		}
	})

	t.Run("correct signature bare format", func(t *testing.T) {
		p := NewGenericHMACProvider("test", secret, false)
		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
		req.Header.Set("X-Webhook-Signature", correctSig)
		if err := p.VerifySignature(body, req); err != nil {
			t.Errorf("expected pass: %v", err)
		}
	})

	t.Run("correct signature X-Signature header", func(t *testing.T) {
		p := NewGenericHMACProvider("test", secret, false)
		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
		req.Header.Set("X-Signature", "sha256="+correctSig)
		if err := p.VerifySignature(body, req); err != nil {
			t.Errorf("expected pass: %v", err)
		}
	})

	t.Run("wrong signature", func(t *testing.T) {
		p := NewGenericHMACProvider("test", secret, false)
		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
		req.Header.Set("X-Webhook-Signature", "sha256=badsignature")
		if err := p.VerifySignature(body, req); err == nil {
			t.Error("wrong signature should fail")
		}
	})

	t.Run("missing signature", func(t *testing.T) {
		p := NewGenericHMACProvider("test", secret, false)
		req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
		if err := p.VerifySignature(body, req); err == nil {
			t.Error("missing signature should fail")
		}
	})
}

func TestVerifySignatureDevMode(t *testing.T) {
	body := []byte(`{"event_id":"evt_123"}`)
	p := NewGenericHMACProvider("test", "", true) // dev mode: no secret
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	if err := p.VerifySignature(body, req); err != nil {
		t.Errorf("dev mode should skip verification: %v", err)
	}
}

func TestVerifySignatureNoSecretNotDev(t *testing.T) {
	body := []byte(`{}`)
	p := NewGenericHMACProvider("test", "", false) // no secret, no dev mode
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(body))
	if err := p.VerifySignature(body, req); err == nil {
		t.Error("should fail when no secret and not dev mode")
	}
}

func TestGenericHMACProviderParsePayload(t *testing.T) {
	p := NewGenericHMACProvider("test", "secret", false)
	payload := USDTWebhookPayload{
		EventID:        "evt_1",
		EventType:      "payment.confirmed",
		IdempotencyKey: "idem_1",
		Amount:         "1000.00",
		Currency:       "USDT",
		Network:        "ethereum",
		TxHash:         "0xabc",
		Status:         "confirmed",
	}
	body, _ := json.Marshal(payload)

	parsed, err := p.ParsePayload(body)
	if err != nil {
		t.Fatalf("ParsePayload failed: %v", err)
	}
	if parsed.EventID != "evt_1" {
		t.Errorf("expected evt_1, got %s", parsed.EventID)
	}
	if parsed.Amount != "1000.00" {
		t.Errorf("expected 1000.00, got %s", parsed.Amount)
	}
}

func TestGenericHMACProviderParsePayloadInvalid(t *testing.T) {
	p := NewGenericHMACProvider("test", "secret", false)
	_, err := p.ParsePayload([]byte(`not json`))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestUSDTWebhookHandlerBadMethod(t *testing.T) {
	cfg := Config{DevPaymentEnabled: true}
	store := newTestStore(t, cfg)
	gm := NewUSDTGatewayManager(cfg, store)

	req := httptest.NewRequest(http.MethodGet, "/api/payments/usdt/webhook", nil)
	w := httptest.NewRecorder()
	gm.handleWebhook(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

func TestUSDTWebhookHandlerMissingSignature(t *testing.T) {
	cfg := Config{
		CryptoWebhookSecret: "secret123",
		DevPaymentEnabled:   false,
	}
	store := newTestStore(t, cfg)
	gm := NewUSDTGatewayManager(cfg, store)

	body := []byte(`{"event_id":"evt_1"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/payments/usdt/webhook", bytes.NewReader(body))
	w := httptest.NewRecorder()
	gm.handleWebhook(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for missing signature, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUSDTWebhookHandlerInvalidJSON(t *testing.T) {
	cfg := Config{DevPaymentEnabled: true}
	store := newTestStore(t, cfg)
	gm := NewUSDTGatewayManager(cfg, store)

	body := []byte(`not json`)
	req := httptest.NewRequest(http.MethodPost, "/api/payments/usdt/webhook", bytes.NewReader(body))
	w := httptest.NewRecorder()
	gm.handleWebhook(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for invalid JSON, got %d", w.Code)
	}
}

func TestUSDTWebhookHandlerMissingEventID(t *testing.T) {
	cfg := Config{DevPaymentEnabled: true}
	store := newTestStore(t, cfg)
	gm := NewUSDTGatewayManager(cfg, store)

	payload := USDTWebhookPayload{
		IdempotencyKey: "idem_1",
		EventType:      "payment.confirmed",
		Status:         "confirmed",
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/payments/usdt/webhook", bytes.NewReader(body))
	w := httptest.NewRecorder()
	gm.handleWebhook(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing event_id, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUSDTWebhookHandlerValidConfirmed(t *testing.T) {
	cfg := Config{
		DevPaymentEnabled: true,
		PlatformFeeBps:    1000,
	}
	store := newTestStore(t, cfg)
	gm := NewUSDTGatewayManager(cfg, store)

	// First create a project for the webhook to reference
	userID := createTestUser(t, store)
	project, err := store.CreateProject(context.Background(), userID, CreateProjectRequest{
		Title:         "Test Project",
		ClientName:    "Test",
		CompanyName:   "TestCo",
		ClientEmail:   "test@test.com",
		Phone:         "123",
		SiteType:      "web",
		PackageTier:   "basic",
		Timeline:      "2 weeks",
		Brief:         "Test",
		BudgetCents:   100000,
		PaymentMethod: PaymentCrypto,
	})
	if err != nil {
		t.Fatalf("failed to create project: %v", err)
	}

	payload := USDTWebhookPayload{
		EventID:        "evt_confirmed_1",
		EventType:      "payment.confirmed",
		IdempotencyKey: "idem_confirmed_1",
		Amount:         "1000.00",
		Currency:       "USDT",
		Network:        "ethereum",
		TxHash:         "0xabc123",
		Sender:         "0xsender",
		Receiver:       "0xreceiver",
		Status:         "confirmed",
		ProjectID:      project.ID,
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/payments/usdt/webhook", bytes.NewReader(body))
	w := httptest.NewRecorder()
	gm.handleWebhook(w, req)

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

	// Verify project was updated
	projects := store.ListProjects("")
	var found bool
	for _, p := range projects {
		if p.ID == project.ID {
			found = true
			if p.PaymentStatus != "verified" {
				t.Errorf("expected payment_status verified, got %s", p.PaymentStatus)
			}
			if p.PaymentReference != "0xabc123" {
				t.Errorf("expected payment_reference 0xabc123, got %s", p.PaymentReference)
			}
		}
	}
	if !found {
		t.Error("project not found in listing")
	}
}

func TestUSDTWebhookHandlerIdempotencyDuplicate(t *testing.T) {
	cfg := Config{DevPaymentEnabled: true}
	store := newTestStore(t, cfg)
	gm := NewUSDTGatewayManager(cfg, store)

	userID := createTestUser(t, store)
	project, err := store.CreateProject(context.Background(), userID, CreateProjectRequest{
		Title:         "Test",
		ClientName:    "Test",
		CompanyName:   "TestCo",
		ClientEmail:   "t@t.com",
		Phone:         "123",
		SiteType:      "web",
		PackageTier:   "basic",
		Timeline:      "2w",
		Brief:         "T",
		BudgetCents:   100000,
		PaymentMethod: PaymentCrypto,
	})
	if err != nil {
		t.Fatalf("create project: %v", err)
	}

	payload := USDTWebhookPayload{
		EventID:        "evt_dup",
		EventType:      "payment.confirmed",
		IdempotencyKey: "idem_dup_1",
		Amount:         "1000.00",
		Currency:       "USDT",
		Network:        "ethereum",
		Receiver:       "0xreceiver",
		Status:         "confirmed",
		ProjectID:      project.ID,
	}
	body1, _ := json.Marshal(payload)
	req1 := httptest.NewRequest(http.MethodPost, "/api/payments/usdt/webhook", bytes.NewReader(body1))
	w1 := httptest.NewRecorder()
	gm.handleWebhook(w1, req1)

	if w1.Code != http.StatusOK {
		t.Fatalf("first request: expected 200, got %d", w1.Code)
	}

	// Duplicate with same idempotency key
	payload2 := payload
	payload2.EventID = "evt_dup_2"
	body2, _ := json.Marshal(payload2)
	req2 := httptest.NewRequest(http.MethodPost, "/api/payments/usdt/webhook", bytes.NewReader(body2))
	w2 := httptest.NewRecorder()
	gm.handleWebhook(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("duplicate: expected 200, got %d", w2.Code)
	}

	var resp USDTWebhookResponse
	if err := json.Unmarshal(w2.Body.Bytes(), &resp); err != nil {
		t.Fatalf("parse response: %v", err)
	}
	if resp.Message != "duplicate webhook already processed" {
		t.Errorf("expected duplicate message, got: %s", resp.Message)
	}
}

func TestListUSDTWebhookEvents(t *testing.T) {
	cfg := Config{DevPaymentEnabled: true}
	store := newTestStore(t, cfg)

	events := store.ListUSDTWebhookEvents()
	if events == nil {
		t.Error("expected non-nil slice")
	}
	if len(events) != 0 {
		t.Errorf("expected 0 events, got %d", len(events))
	}

	// Add one
	store.SaveUSDTWebhookEvent(&USDTWebhookEvent{
		ID:         "test-1",
		Status:     "confirmed",
		ReceivedAt: time.Now().UTC(),
	})

	events = store.ListUSDTWebhookEvents()
	if len(events) != 1 {
		t.Errorf("expected 1 event, got %d", len(events))
	}
}

func TestUSDTGatewayProviderRegistration(t *testing.T) {
	cfg := Config{DevPaymentEnabled: true}
	store := newTestStore(t, cfg)
	gm := NewUSDTGatewayManager(cfg, store)

	if len(gm.providers) != 1 {
		t.Errorf("expected 1 default provider, got %d", len(gm.providers))
	}

	customProvider := NewGenericHMACProvider("custom", "custom-secret", false)
	gm.RegisterProvider(customProvider)

	if len(gm.providers) != 2 {
		t.Errorf("expected 2 providers after registration, got %d", len(gm.providers))
	}
}

// createTestUser is a helper to create a user in the test store.
func createTestUser(t *testing.T, store *Store) string {
	t.Helper()
	auth, err := store.Register(RegisterRequest{
		Name:     "Test User",
		Email:    "test-" + t.Name() + "@example.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("failed to create test user: %v", err)
	}
	return auth.User.ID
}

// newTestStore creates a minimal in-memory store for tests.
func newTestStore(t *testing.T, cfg Config) *Store {
	t.Helper()
	store, err := NewStore(cfg, NewPaymentManager(cfg), NewRepoFactory(cfg), NewEmailSender(cfg))
	if err != nil {
		t.Fatalf("failed to create test store: %v", err)
	}
	return store
}
