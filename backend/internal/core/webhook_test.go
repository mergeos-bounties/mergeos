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

func TestCryptoWebhookVerifiesSignatureAndCreatesProjectSuccessfully(t *testing.T) {
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

	// Register a test user
	userAuth, err := store.Register(RegisterRequest{
		Name:        "Test Client",
		CompanyName: "Test Co",
		Email:       "test-client@example.com",
		Password:    "password123",
	})
	if err != nil {
		t.Fatal(err)
	}

	server := NewServer(cfg, store, payments)

	// Create webhook payload
	payload := CryptoWebhookRequest{
		UserID:      userAuth.User.ID,
		Title:       "Solana SPL Payment Gateway",
		ClientName:  "Test Client",
		ClientEmail: "test-client@example.com",
		BudgetCents: 10000,        // 100 USD minimum
		TxHash:      "LOCAL-PAID", // Using dev verifier code as hash
	}
	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}

	// Calculate HMAC signature
	mac := hmac.New(sha256.New, []byte(cfg.CryptoWebhookSecret))
	mac.Write(bodyBytes)
	signature := hex.EncodeToString(mac.Sum(nil))

	req := httptest.NewRequest(http.MethodPost, "/api/payments/crypto/webhook", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-MergeOS-Signature", signature)

	resp := httptest.NewRecorder()
	server.Routes().ServeHTTP(resp, req)

	if resp.Code != http.StatusCreated {
		t.Fatalf("unexpected status code: got %d, body: %s", resp.Code, resp.Body.String())
	}

	var project Project
	if err := json.Unmarshal(resp.Body.Bytes(), &project); err != nil {
		t.Fatal(err)
	}

	if project.Title != "Solana SPL Payment Gateway" || project.PaymentStatus != "verified" {
		t.Fatalf("invalid project state: %#v", project)
	}
}

func TestCryptoWebhookRejectsReplayAttack(t *testing.T) {
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

	userAuth, err := store.Register(RegisterRequest{
		Name:        "Test Client",
		CompanyName: "Test Co",
		Email:       "test-client@example.com",
		Password:    "password123",
	})
	if err != nil {
		t.Fatal(err)
	}

	server := NewServer(cfg, store, payments)

	payload := CryptoWebhookRequest{
		UserID:      userAuth.User.ID,
		Title:       "Solana SPL Payment Gateway",
		ClientName:  "Test Client",
		ClientEmail: "test-client@example.com",
		BudgetCents: 10000,
		TxHash:      "LOCAL-PAID",
	}
	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}

	mac := hmac.New(sha256.New, []byte(cfg.CryptoWebhookSecret))
	mac.Write(bodyBytes)
	signature := hex.EncodeToString(mac.Sum(nil))

	// First call: succeeds
	req1 := httptest.NewRequest(http.MethodPost, "/api/payments/crypto/webhook", bytes.NewReader(bodyBytes))
	req1.Header.Set("Content-Type", "application/json")
	req1.Header.Set("X-MergeOS-Signature", signature)
	resp1 := httptest.NewRecorder()
	server.Routes().ServeHTTP(resp1, req1)

	if resp1.Code != http.StatusCreated {
		t.Fatalf("first call failed: got %d, body: %s", resp1.Code, resp1.Body.String())
	}

	// Second call with same TxHash: must fail with 409 Conflict (Replay Attack)
	req2 := httptest.NewRequest(http.MethodPost, "/api/payments/crypto/webhook", bytes.NewReader(bodyBytes))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("X-MergeOS-Signature", signature)
	resp2 := httptest.NewRecorder()
	server.Routes().ServeHTTP(resp2, req2)

	if resp2.Code != http.StatusConflict {
		t.Fatalf("replay attack succeeded or returned wrong status: got %d, body: %s", resp2.Code, resp2.Body.String())
	}
}
