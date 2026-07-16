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
)

func TestUSDTWebhookHandler_ValidSignature(t *testing.T) {
	cfg := Config{
		USDTWebhookSecret: "test-secret-key",
	}
	store, err := NewStore(cfg, nil, nil, nil)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	handler := NewUSDTWebhookHandler(cfg, store)

	event := USDTWebhookEvent{
		ID:            "tx-001",
		Status:        "COMPLETED",
		Currency:      "USDT",
		Amount:        "100.00",
		PaymentIntent: "intent-001",
		TxHash:        "0xabc123",
		Sender:        "0xSenderAddress",
	}

	body, _ := json.Marshal(event)

	req := httptest.NewRequest(http.MethodPost, "/api/payments/usdt/webhook", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	mac := hmac.New(sha256.New, []byte(cfg.USDTWebhookSecret))
	mac.Write(body)
	req.Header.Set("X-Crypto-Signature", hex.EncodeToString(mac.Sum(nil)))

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUSDTWebhookHandler_InvalidSignature(t *testing.T) {
	cfg := Config{
		USDTWebhookSecret: "test-secret-key",
	}
	store, err := NewStore(cfg, nil, nil, nil)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	handler := NewUSDTWebhookHandler(cfg, store)

	body, _ := json.Marshal(USDTWebhookEvent{ID: "tx-002"})

	req := httptest.NewRequest(http.MethodPost, "/api/payments/usdt/webhook", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Crypto-Signature", "invalid-signature")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", w.Code)
	}
}

func TestUSDTWebhookHandler_NoSignatureWithDevMode(t *testing.T) {
	cfg := Config{
		USDTWebhookSecret: "", // dev mode — no secret configured
	}
	store, err := NewStore(cfg, nil, nil, nil)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	handler := NewUSDTWebhookHandler(cfg, store)

	body, _ := json.Marshal(USDTWebhookEvent{
		ID:       "tx-003",
		Status:   "COMPLETED",
		Currency: "USDT",
		Amount:   "50.00",
		TxHash:   "0xdev123",
		Sender:   "0xDevSender",
	})

	req := httptest.NewRequest(http.MethodPost, "/api/payments/usdt/webhook", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200 in dev mode, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRecordUSDTWebhookPayment_MatchAndSettleProject(t *testing.T) {
	cfg := Config{
		USDTWebhookSecret: "test-secret-key",
	}
	store, err := NewStore(cfg, nil, nil, nil)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	// Create a project with PaymentUSDT
	projectReq := CreateProjectRequest{
		Title:            "Test crypto funding",
		ClientName:       "Alice",
		ClientEmail:      "alice@example.com",
		PaymentMethod:    PaymentUSDT,
		PaymentReference: "0xtx_test",
		BudgetCents:      50000,
	}
	_, err = store.CreateProject(context.Background(), "user-1", projectReq)
	if err != nil {
		t.Fatalf("failed to create project: %v", err)
	}

	// Send webhook
	settlement, err := store.RecordUSDTWebhookPayment(USDTWebhookEvent{
		Status:   "COMPLETED",
		Currency: "USDT",
		Amount:   "500.00",
		TxHash:   "0xtx_test",
		Sender:   "0xfunder",
	})
	if err != nil {
		t.Fatalf("RecordUSDTWebhookPayment failed: %v", err)
	}

	if settlement.Status != "settled" {
		t.Errorf("expected status 'settled', got '%s'", settlement.Status)
	}
	if settlement.ProjectID == "" {
		t.Error("expected non-empty ProjectID")
	}
	if settlement.LedgerEntry == nil {
		t.Error("expected LedgerEntry to be created")
	}
}

func TestRecordUSDTWebhookPayment_DuplicateSafe(t *testing.T) {
	cfg := Config{
		USDTWebhookSecret: "test-secret-key",
	}
	store, err := NewStore(cfg, nil, nil, nil)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer store.Close()

	event := USDTWebhookEvent{
		Status:   "COMPLETED",
		Currency: "USDT",
		Amount:   "100.00",
		TxHash:   "0xdup",
		Sender:   "0xsender",
	}

	// First call
	_, _ = store.RecordUSDTWebhookPayment(event)

	// Duplicate call should be safe
	settlement2, err := store.RecordUSDTWebhookPayment(event)
	if err != nil {
		t.Fatalf("duplicate call failed: %v", err)
	}
	if !settlement2.Duplicate {
		t.Error("expected duplicate flag on second call")
	}
	if settlement2.Status != "duplicate" {
		t.Errorf("expected status 'duplicate', got '%s'", settlement2.Status)
	}
}
