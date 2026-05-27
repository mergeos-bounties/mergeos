package core

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMockCryptoProviderCreateInvoice(t *testing.T) {
	p := NewMockCryptoProvider()
	req := CryptoInvoiceRequest{
		AmountCents: 10000,
		Currency:    "usdt",
		Network:     "erc20",
		ProjectID:   "test-project",
		Description: "Test payment",
	}

	resp, err := p.CreateInvoice(req)
	if err != nil {
		t.Fatalf("CreateInvoice failed: %v", err)
	}
	if resp.InvoiceID == "" {
		t.Error("expected non-empty invoice ID")
	}
	if resp.PayAddress == "" {
		t.Error("expected non-empty pay address")
	}
	if resp.Currency != "usdt" {
		t.Errorf("expected currency usdt, got %s", resp.Currency)
	}
	if resp.Status != "pending" {
		t.Errorf("expected status pending, got %s", resp.Status)
	}
}

func TestMockProviderParseWebhookEvent(t *testing.T) {
	p := NewMockCryptoProvider()

	event := CryptoWebhookEvent{
		InvoiceID: "test-inv-1",
		Status:    "confirmed",
		TxHash:    "0xabc123",
	}
	body, _ := json.Marshal(event)

	parsed, err := p.ParseWebhookEvent(body)
	if err != nil {
		t.Fatalf("ParseWebhookEvent failed: %v", err)
	}
	if parsed.InvoiceID != "test-inv-1" {
		t.Errorf("expected invoice ID test-inv-1, got %s", parsed.InvoiceID)
	}
	if parsed.Status != "confirmed" {
		t.Errorf("expected status confirmed, got %s", parsed.Status)
	}
}

func TestMockProviderParseWebhookEventMissingInvoiceID(t *testing.T) {
	p := NewMockCryptoProvider()
	body := []byte(`{"status":"confirmed"}`)

	_, err := p.ParseWebhookEvent(body)
	if err == nil {
		t.Error("expected error for missing invoice_id")
	}
}

func TestMapProviderStatus(t *testing.T) {
	tests := []struct {
		input    string
		expected CryptoPaymentStatus
	}{
		{"pending", CryptoStatusPending},
		{"waiting", CryptoStatusPending},
		{"confirming", CryptoStatusConfirming},
		{"confirmed", CryptoStatusConfirmed},
		{"completed", CryptoStatusConfirmed},
		{"failed", CryptoStatusFailed},
		{"expired", CryptoStatusExpired},
		{"refunded", CryptoStatusRefunded},
		{"unknown", CryptoStatusPending},
	}
	for _, tt := range tests {
		got := MapProviderStatus(tt.input)
		if got != tt.expected {
			t.Errorf("MapProviderStatus(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestSimulatePayment(t *testing.T) {
	p := NewMockCryptoProvider()

	// Create invoice first
	resp, _ := p.CreateInvoice(CryptoInvoiceRequest{
		AmountCents: 5000,
		Currency:    "usdt",
		Network:     "erc20",
	})

	// Simulate payment
	body, err := p.SimulatePayment(resp.InvoiceID, "confirmed", "0xdeadbeef")
	if err != nil {
		t.Fatalf("SimulatePayment failed: %v", err)
	}

	// Parse the simulated webhook
	event, err := p.ParseWebhookEvent(body)
	if err != nil {
		t.Fatalf("ParseWebhookEvent failed: %v", err)
	}
	if event.InvoiceID != resp.InvoiceID {
		t.Errorf("invoice ID mismatch: got %s", event.InvoiceID)
	}
	if event.TxHash != "0xdeadbeef" {
		t.Errorf("tx hash mismatch: got %s", event.TxHash)
	}
}

func TestCryptoWebhookEndpointMethodNotAllowed(t *testing.T) {
	cfg := Config{Environment: "development"}
	store := &Store{cryptoInvoices: make(map[string]*CryptoInvoice)}
	payments := NewPaymentManager(cfg)
	srv := NewServer(cfg, store, payments)

	req := httptest.NewRequest(http.MethodGet, "/api/payments/crypto/webhook", nil)
	w := httptest.NewRecorder()
	srv.cryptoWebhook(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected 405, got %d", w.Code)
	}
}

func TestCryptoWebhookEndpointNoGateway(t *testing.T) {
	cfg := Config{Environment: "development"}
	store := &Store{cryptoInvoices: make(map[string]*CryptoInvoice)}
	payments := NewPaymentManager(cfg)
	srv := NewServer(cfg, store, payments)
	srv.cryptoGateway = nil

	req := httptest.NewRequest(http.MethodPost, "/api/payments/crypto/webhook", bytes.NewReader([]byte("{}")))
	w := httptest.NewRecorder()
	srv.cryptoWebhook(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected 503, got %d", w.Code)
	}
}
