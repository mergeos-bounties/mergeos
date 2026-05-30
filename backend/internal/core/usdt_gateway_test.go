package core

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

func TestUSDTMockProviderCreateInvoice(t *testing.T) {
	provider := NewUSDTMockProvider()
	if provider.Name() != "usdt-mock" {
		t.Fatalf("expected usdt-mock, got %s", provider.Name())
	}

	req := USDTInvoiceRequest{
		AmountUSDCents: 10000,
		OrderID:        "test-order-1",
		Description:    "Test project funding",
		CustomerEmail:  "test@example.com",
		Network:        "trc20",
	}
	invoice, err := provider.CreateInvoice(req)
	if err != nil {
		t.Fatalf("create invoice failed: %v", err)
	}
	if invoice.InvoiceID == "" {
		t.Fatal("expected invoice ID")
	}
	if invoice.PayAddress == "" {
		t.Fatal("expected pay address")
	}
	if invoice.PayCurrency != "USDT" {
		t.Fatalf("expected USDT, got %s", invoice.PayCurrency)
	}
}

func TestUSDTWebhookValidSignature(t *testing.T) {
	tempDir := t.TempDir()
	cfg := Config{
		Environment:         "local",
		TokenSymbol:         defaultTokenSymbol,
		StatePath:           filepath.Join(tempDir, "state.json"),
		PlatformFeeBps:      1000,
		DevPaymentEnabled:   true,
		DevPaymentCode:      "LOCAL-PAID",
		GitHubOwner:         defaultGitHubOwner,
		BountyRoot:          filepath.Join(tempDir, "bounties"),
		SMTPFrom:            "noreply@mergeos.local",
		CryptoWebhookSecret: "usdt-secret-key",
	}
	payments := NewPaymentManager(cfg)
	store, err := NewStore(cfg, payments, NewRepoFactory(cfg), NewEmailSender(cfg))
	if err != nil {
		t.Fatal(err)
	}
	server := NewServer(cfg, store, payments)

	payload := USDTWebhookPayload{
		EventType:   "payment.confirmed",
		InvoiceID:   "mock_usdt_20250101120000",
		OrderID:     "mrg_test_123",
		TxHash:      "0xabcdef1234567890abcdef1234567890abcdef12",
		Status:      USDTStatusConfirmed,
		AmountPaid:  "100.00",
		Currency:    "USDT",
		Network:     "trc20",
		FromAddress: "TFromAddress123",
		ToAddress:   "TToAddress456",
	}
	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}

	mac := hmac.New(sha256.New, []byte(cfg.CryptoWebhookSecret))
	mac.Write(bodyBytes)
	signature := hex.EncodeToString(mac.Sum(nil))

	req := httptest.NewRequest(http.MethodPost, "/api/payments/usdt/webhook", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-MergeOS-Signature", signature)

	resp := httptest.NewRecorder()
	server.Routes().ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
	}
}

func TestUSDTWebhookRejectsInvalidSignature(t *testing.T) {
	tempDir := t.TempDir()
	cfg := Config{
		Environment:         "local",
		TokenSymbol:         defaultTokenSymbol,
		StatePath:           filepath.Join(tempDir, "state.json"),
		PlatformFeeBps:      1000,
		DevPaymentEnabled:   true,
		DevPaymentCode:      "LOCAL-PAID",
		GitHubOwner:         defaultGitHubOwner,
		BountyRoot:          filepath.Join(tempDir, "bounties"),
		SMTPFrom:            "noreply@mergeos.local",
		CryptoWebhookSecret: "secret-key",
	}
	payments := NewPaymentManager(cfg)
	store, err := NewStore(cfg, payments, NewRepoFactory(cfg), NewEmailSender(cfg))
	if err != nil {
		t.Fatal(err)
	}
	server := NewServer(cfg, store, payments)

	payload := USDTWebhookPayload{
		EventType: "payment.confirmed",
		InvoiceID: "mock_usdt_001",
		Status:    USDTStatusConfirmed,
		TxHash:    "0xdeadbeef",
	}
	bodyBytes, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/payments/usdt/webhook", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-MergeOS-Signature", "invalid-signature")

	resp := httptest.NewRecorder()
	server.Routes().ServeHTTP(resp, req)

	if resp.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", resp.Code, resp.Body.String())
	}
}

func TestUSDTWebhookDuplicateCallback(t *testing.T) {
	tempDir := t.TempDir()
	cfg := Config{
		Environment:         "local",
		TokenSymbol:         defaultTokenSymbol,
		StatePath:           filepath.Join(tempDir, "state.json"),
		PlatformFeeBps:      1000,
		DevPaymentEnabled:   true,
		DevPaymentCode:      "LOCAL-PAID",
		GitHubOwner:         defaultGitHubOwner,
		BountyRoot:          filepath.Join(tempDir, "bounties"),
		SMTPFrom:            "noreply@mergeos.local",
		CryptoWebhookSecret: "dup-secret",
	}
	payments := NewPaymentManager(cfg)
	store, err := NewStore(cfg, payments, NewRepoFactory(cfg), NewEmailSender(cfg))
	if err != nil {
		t.Fatal(err)
	}
	server := NewServer(cfg, store, payments)

	payload := USDTWebhookPayload{
		EventType: "payment.confirmed",
		InvoiceID: "dup-invoice-1",
		TxHash:    "0xduptxhash123456789",
		Status:    USDTStatusConfirmed,
		AmountPaid: "50.00",
		Currency:  "USDT",
	}
	bodyBytes, _ := json.Marshal(payload)

	mac := hmac.New(sha256.New, []byte(cfg.CryptoWebhookSecret))
	mac.Write(bodyBytes)
	sig := hex.EncodeToString(mac.Sum(nil))

	req1 := httptest.NewRequest(http.MethodPost, "/api/payments/usdt/webhook", bytes.NewReader(bodyBytes))
	req1.Header.Set("Content-Type", "application/json")
	req1.Header.Set("X-MergeOS-Signature", sig)
	resp1 := httptest.NewRecorder()
	server.Routes().ServeHTTP(resp1, req1)
	if resp1.Code != http.StatusOK {
		t.Fatalf("first call: expected 200, got %d", resp1.Code)
	}

	req2 := httptest.NewRequest(http.MethodPost, "/api/payments/usdt/webhook", bytes.NewReader(bodyBytes))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("X-MergeOS-Signature", sig)
	resp2 := httptest.NewRecorder()
	server.Routes().ServeHTTP(resp2, req2)
	if resp2.Code != http.StatusOK {
		t.Fatalf("duplicate call: expected 200, got %d", resp2.Code)
	}

	var result map[string]string
	json.Unmarshal(resp2.Body.Bytes(), &result)
	if result["status"] != "duplicate_ignored" {
		t.Fatalf("expected duplicate_ignored, got %s", result["status"])
	}
}
