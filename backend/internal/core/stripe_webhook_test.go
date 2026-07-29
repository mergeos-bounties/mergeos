package core

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// stripeTestWebhookSecret is the test webhook secret used in tests.
const stripeTestWebhookSecret = "whsec_test_secret_key_12345"

// stripeTestSignatureHeader returns a valid Stripe-Signature header value for the given payload.
func stripeTestSignatureHeader(payload []byte, secret string) string {
	timestamp := time.Now().Unix()
	signedPayload := fmt.Sprintf("%d.%s", timestamp, string(payload))
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(signedPayload))
	sig := hex.EncodeToString(mac.Sum(nil))
	return fmt.Sprintf("t=%d,v1=%s", timestamp, sig)
}

func stripeTestConfig(t *testing.T) Config {
	t.Helper()
	tempDir := t.TempDir()
	return Config{
		Environment:        "local",
		StatePath:          filepath.Join(tempDir, "state.json"),
		TokenSymbol:        defaultTokenSymbol,
		DevPaymentEnabled:  true,
		DevPaymentCode:     "LOCAL-PAID",
		GitHubOwner:        defaultGitHubOwner,
		BountyRoot:         filepath.Join(tempDir, "bounties"),
		SMTPFrom:           "noreply@mergeos.local",
		StripeWebhookSecret: stripeTestWebhookSecret,
	}
}

func succeededStripePaymentIntentEvent(eventID, piID string) string {
	return fmt.Sprintf(`{
		"id": "%s",
		"type": "payment_intent.succeeded",
		"data": {
			"object": {
				"id": "%s",
				"amount": 10000,
				"amount_received": 10000,
				"currency": "usd",
				"status": "succeeded",
				"description": "test payment"
			}
		}
	}`, eventID, piID)
}

func TestParseStripeSignatureHeader(t *testing.T) {
	header := "t=1234567890,v1=sig1,v1=sig2"
	params := parseStripeSignatureHeader(header)
	if params["t"] != "1234567890" {
		t.Errorf("expected t=1234567890, got t=%s", params["t"])
	}
	if params["v1"] != "sig1" {
		t.Errorf("expected v1=sig1, got v1=%s", params["v1"])
	}
}

func TestVerifyStripeWebhookSignatureValid(t *testing.T) {
	payload := []byte(`{"test": "data"}`)
	header := stripeTestSignatureHeader(payload, stripeTestWebhookSecret)
	if err := verifyStripeWebhookSignature(payload, header, stripeTestWebhookSecret); err != nil {
		t.Fatalf("expected valid signature, got error: %v", err)
	}
}

func TestVerifyStripeWebhookSignatureInvalid(t *testing.T) {
	payload := []byte(`{"test": "data"}`)
	header := stripeTestSignatureHeader(payload, "wrong-secret")
	if err := verifyStripeWebhookSignature(payload, header, stripeTestWebhookSecret); err == nil {
		t.Fatal("expected error for invalid signature")
	}
}

func TestVerifyStripeWebhookSignatureMissingHeader(t *testing.T) {
	payload := []byte(`{"test": "data"}`)
	if err := verifyStripeWebhookSignature(payload, "", stripeTestWebhookSecret); err == nil {
		t.Fatal("expected error for missing header")
	}
}

func TestVerifyStripeWebhookSignatureDevMode(t *testing.T) {
	payload := []byte(`{"test": "data"}`)
	// Empty secret means dev mode — skip verification
	if err := verifyStripeWebhookSignature(payload, "", ""); err != nil {
		t.Fatalf("expected dev mode to skip verification, got error: %v", err)
	}
}

func TestStripeWebhookEventParse(t *testing.T) {
	data := succeededStripePaymentIntentEvent("evt_1", "pi_3")
	var event stripeWebhookEvent
	if err := json.Unmarshal([]byte(data), &event); err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if event.ID != "evt_1" {
		t.Errorf("expected evt_1, got %s", event.ID)
	}
	if event.Type != "payment_intent.succeeded" {
		t.Errorf("expected payment_intent.succeeded, got %s", event.Type)
	}
	payment, err := stripeWebhookPaymentFromEvent(event)
	if err != nil {
		t.Fatalf("payment parse error: %v", err)
	}
	if payment.PaymentIntentID != "pi_3" {
		t.Errorf("expected pi_3, got %s", payment.PaymentIntentID)
	}
	if payment.Status != "succeeded" {
		t.Errorf("expected succeeded, got %s", payment.Status)
	}
	if payment.AmountCents != 10000 {
		t.Errorf("expected 10000 cents, got %d", payment.AmountCents)
	}
	if payment.Currency != "USD" {
		t.Errorf("expected USD, got %s", payment.Currency)
	}
}

func TestStripeWebhookRejectsInvalidSignature(t *testing.T) {
	cfg := stripeTestConfig(t)
	payments := NewPaymentManager(cfg)
	store, err := NewStore(cfg, payments, NewRepoFactory(cfg), NewEmailSender(cfg))
	if err != nil {
		t.Fatal(err)
	}
	server := NewServer(cfg, store, payments)

	body := succeededStripePaymentIntentEvent("evt_bad_sig", "pi_bad")
	req := httptest.NewRequest(http.MethodPost, "/api/payments/stripe/webhook", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Stripe-Signature", "t=0,v1=invalid_signature")
	rr := httptest.NewRecorder()
	server.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestStripeWebhookIgnoresNonPaymentIntentEvents(t *testing.T) {
	cfg := stripeTestConfig(t)
	payments := NewPaymentManager(cfg)
	store, err := NewStore(cfg, payments, NewRepoFactory(cfg), NewEmailSender(cfg))
	if err != nil {
		t.Fatal(err)
	}
	server := NewServer(cfg, store, payments)

	body := `{"id":"evt_ignored","type":"charge.succeeded","data":{"object":{"id":"ch_3"}}}`
	req := httptest.NewRequest(http.MethodPost, "/api/payments/stripe/webhook", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Stripe-Signature", stripeTestSignatureHeader([]byte(body), stripeTestWebhookSecret))
	rr := httptest.NewRecorder()
	server.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200 for ignored event, got %d: %s", rr.Code, rr.Body.String())
	}
	var result map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["status"] != "ignored" {
		t.Fatalf("expected ignored status, got %#v", result)
	}
}

func TestStripeWebhookRecordsPaymentOnce(t *testing.T) {
	cfg := stripeTestConfig(t)
	payments := NewPaymentManager(cfg)
	store, err := NewStore(cfg, payments, NewRepoFactory(cfg), NewEmailSender(cfg))
	if err != nil {
		t.Fatal(err)
	}
	auth, err := store.Register(RegisterRequest{
		Name:     "Stripe Client",
		Email:    "stripe-webhook@example.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatal(err)
	}

	projectID := "prj_stripe_webhook"
	store.mu.Lock()
	store.projects[projectID] = &Project{
		ID:               projectID,
		ClientUserID:     auth.User.ID,
		Title:            "Stripe funded project",
		ClientName:       auth.User.Name,
		ClientEmail:      auth.User.Email,
		PaymentMethod:    PaymentStripe,
		PaymentStatus:    "pending",
		PaymentProvider:  "stripe",
		PaymentReference: "pi_webhook_1",
		BudgetCents:      10000,
		Status:           ProjectFunded,
		CreatedAt:        time.Now().UTC(),
	}
	if err := store.saveLocked(); err != nil {
		store.mu.Unlock()
		t.Fatal(err)
	}
	store.mu.Unlock()

	server := NewServer(cfg, store, payments)

	// First webhook call
	body := succeededStripePaymentIntentEvent("evt_first", "pi_webhook_1")
	req := httptest.NewRequest(http.MethodPost, "/api/payments/stripe/webhook", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Stripe-Signature", stripeTestSignatureHeader([]byte(body), stripeTestWebhookSecret))
	rr := httptest.NewRecorder()
	server.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("first call: expected 200, got %d: %s", rr.Code, rr.Body.String())
	}
	var firstSettlement stripeWebhookSettlement
	if err := json.Unmarshal(rr.Body.Bytes(), &firstSettlement); err != nil {
		t.Fatal(err)
	}
	if firstSettlement.Status != "verified" || firstSettlement.ProjectID != projectID || firstSettlement.Duplicate {
		t.Fatalf("first settlement = %#v", firstSettlement)
	}
	if got := countStripePaymentVerified(store); got != 1 {
		t.Fatalf("stripe payment ledger entries after first webhook = %d", got)
	}

	// Replay the same event
	replay := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodPost, "/api/payments/stripe/webhook", bytes.NewReader([]byte(body)))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Stripe-Signature", stripeTestSignatureHeader([]byte(body), stripeTestWebhookSecret))
	server.Routes().ServeHTTP(replay, req2)

	if replay.Code != http.StatusOK {
		t.Fatalf("replay: expected 200, got %d: %s", replay.Code, replay.Body.String())
	}
	var replaySettlement stripeWebhookSettlement
	if err := json.Unmarshal(replay.Body.Bytes(), &replaySettlement); err != nil {
		t.Fatal(err)
	}
	if replaySettlement.Status != "duplicate" || !replaySettlement.Duplicate {
		t.Fatalf("replay settlement = %#v", replaySettlement)
	}
	if got := countStripePaymentVerified(store); got != 1 {
		t.Fatalf("stripe payment ledger entries after replay = %d", got)
	}
}

func TestStripeWebhookRejectsAmountMismatch(t *testing.T) {
	cfg := stripeTestConfig(t)
	payments := NewPaymentManager(cfg)
	store, err := NewStore(cfg, payments, NewRepoFactory(cfg), NewEmailSender(cfg))
	if err != nil {
		t.Fatal(err)
	}
	auth, err := store.Register(RegisterRequest{
		Name:     "Stripe Mismatch",
		Email:    "stripe-mismatch@example.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatal(err)
	}

	projectID := "prj_stripe_mismatch"
	store.mu.Lock()
	store.projects[projectID] = &Project{
		ID:               projectID,
		ClientUserID:     auth.User.ID,
		Title:            "Stripe amount mismatch",
		ClientName:       auth.User.Name,
		ClientEmail:      auth.User.Email,
		PaymentMethod:    PaymentStripe,
		PaymentStatus:    "pending",
		PaymentProvider:  "stripe",
		PaymentReference: "pi_mismatch_1",
		BudgetCents:      50000, // Project expects 500 USD
		Status:           ProjectFunded,
		CreatedAt:        time.Now().UTC(),
	}
	if err := store.saveLocked(); err != nil {
		store.mu.Unlock()
		t.Fatal(err)
	}
	store.mu.Unlock()

	server := NewServer(cfg, store, payments)

	// PaymentIntent has 100 USD, but project expects 500 USD
	body := succeededStripePaymentIntentEvent("evt_mismatch", "pi_mismatch_1")
	req := httptest.NewRequest(http.MethodPost, "/api/payments/stripe/webhook", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Stripe-Signature", stripeTestSignatureHeader([]byte(body), stripeTestWebhookSecret))
	rr := httptest.NewRecorder()
	server.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for amount mismatch, got %d: %s", rr.Code, rr.Body.String())
	}
}

func countStripePaymentVerified(store *Store) int {
	count := 0
	for _, entry := range store.ListLedger() {
		if entry.Type == "payment_verified" && strings.EqualFold(entry.FromAccount, "payment:stripe") {
			count++
		}
	}
	return count
}