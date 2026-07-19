package core

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// ---------------------------------------------------------------------------
// Unit tests: parseStripeSignatureHeader
// ---------------------------------------------------------------------------

func TestParseStripeSignatureHeader(t *testing.T) {
	t.Run("valid header", func(t *testing.T) {
		header := "t=1234567890,v1=abcdef1234567890abcdef1234567890abcdef12"
		ts, sig, err := parseStripeSignatureHeader(header)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ts != 1234567890 {
			t.Fatalf("timestamp = %d, want 1234567890", ts)
		}
		if sig != "abcdef1234567890abcdef1234567890abcdef12" {
			t.Fatalf("signature = %q", sig)
		}
	})

	t.Run("header with multiple v1 signatures picks first", func(t *testing.T) {
		header := "t=987654321,v1=firstsig,v1=secondsig"
		ts, sig, err := parseStripeSignatureHeader(header)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ts != 987654321 {
			t.Fatalf("timestamp = %d", ts)
		}
		if sig != "firstsig" {
			t.Fatalf("signature = %q, want firstsig", sig)
		}
	})

	t.Run("missing timestamp", func(t *testing.T) {
		header := "v1=somesignature"
		_, _, err := parseStripeSignatureHeader(header)
		if err == nil {
			t.Fatal("expected error for missing timestamp")
		}
	})

	t.Run("missing v1 signature", func(t *testing.T) {
		header := "t=12345"
		_, _, err := parseStripeSignatureHeader(header)
		if err == nil {
			t.Fatal("expected error for missing v1 signature")
		}
	})

	t.Run("invalid timestamp value", func(t *testing.T) {
		header := "t=notanumber,v1=sig"
		_, _, err := parseStripeSignatureHeader(header)
		if err == nil {
			t.Fatal("expected error for invalid timestamp")
		}
	})

	t.Run("empty header", func(t *testing.T) {
		_, _, err := parseStripeSignatureHeader("")
		if err == nil {
			t.Fatal("expected error for empty header")
		}
	})
}

// ---------------------------------------------------------------------------
// Unit tests: verifyStripeWebhookSignature
// ---------------------------------------------------------------------------

// computeStripeSignature replicates the server's signature calculation.
func computeStripeSignature(timestamp int64, payload, secret string) string {
	signedPayload := strconv.FormatInt(timestamp, 10) + "." + payload
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(signedPayload))
	return hex.EncodeToString(mac.Sum(nil))
}

func TestVerifyStripeWebhookSignature(t *testing.T) {
	secret := "whsec_test_secret_key"
	payload := `{"id":"evt_test","type":"payment_intent.succeeded","data":{"object":{"id":"pi_test","amount":1000}}}`
	timestamp := int64(1700000000)
	correctSig := computeStripeSignature(timestamp, payload, secret)
	header := fmt.Sprintf("t=%d,v1=%s", timestamp, correctSig)

	t.Run("valid signature passes", func(t *testing.T) {
		server := &Server{cfg: Config{StripeWebhookSecret: secret}}
		err := server.verifyStripeWebhookSignature([]byte(payload), header)
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
	})

	t.Run("wrong signature fails", func(t *testing.T) {
		server := &Server{cfg: Config{StripeWebhookSecret: secret}}
		wrongHeader := fmt.Sprintf("t=%d,v1=0000000000000000000000000000000000000000", timestamp)
		err := server.verifyStripeWebhookSignature([]byte(payload), wrongHeader)
		if err == nil {
			t.Fatal("expected error for wrong signature")
		}
	})

	t.Run("empty secret in dev skips verification", func(t *testing.T) {
		server := &Server{cfg: Config{Environment: "development", StripeWebhookSecret: ""}}
		err := server.verifyStripeWebhookSignature([]byte(payload), header)
		if err != nil {
			t.Fatalf("expected dev skip, got: %v", err)
		}
	})

	t.Run("empty secret in production returns error", func(t *testing.T) {
		server := &Server{cfg: Config{Environment: "production", StripeWebhookSecret: ""}}
		err := server.verifyStripeWebhookSignature([]byte(payload), header)
		if err == nil {
			t.Fatal("expected error for empty secret in production")
		}
	})
}

// ---------------------------------------------------------------------------
// Unit tests: stripeWebhookPaymentFromEvent (status mapping)
// ---------------------------------------------------------------------------

func TestStripeWebhookPaymentFromEvent_Succeeded(t *testing.T) {
	t.Run("full card details", func(t *testing.T) {
		event := stripeWebhookEvent{
			ID:   "evt_success_1",
			Type: "payment_intent.succeeded",
			Data: stripeEventData{
				Object: stripeEventObject{
					ID:             "pi_123456",
					Amount:         2000,
					AmountReceived: 2000,
					Currency:       "usd",
					Status:         "succeeded",
					Description:    "Test payment",
					Charges: stripeCharges{
						Data: []stripeCharge{{
							ID: "ch_789",
							PaymentMethodDetails: &stripePaymentMethodDetails{
								Card: &stripeCardDetails{
									Brand: "visa",
									Last4: "4242",
								},
							},
						}},
					},
				},
			},
		}
		payment, err := stripeWebhookPaymentFromEvent(event)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if payment.PaymentIntentID != "pi_123456" {
			t.Fatalf("PaymentIntentID = %q", payment.PaymentIntentID)
		}
		if payment.AmountCents != 2000 {
			t.Fatalf("AmountCents = %d", payment.AmountCents)
		}
		if payment.Currency != "usd" {
			t.Fatalf("Currency = %q", payment.Currency)
		}
		if payment.Status != "succeeded" {
			t.Fatalf("Status = %q", payment.Status)
		}
		if payment.Brand != "visa" {
			t.Fatalf("Brand = %q", payment.Brand)
		}
		if payment.Last4 != "4242" {
			t.Fatalf("Last4 = %q", payment.Last4)
		}
	})

	t.Run("missing charge details", func(t *testing.T) {
		event := stripeWebhookEvent{
			ID:   "evt_success_2",
			Type: "payment_intent.succeeded",
			Data: stripeEventData{
				Object: stripeEventObject{
					ID:             "pi_789",
					Amount:         5000,
					AmountReceived: 5000,
					Currency:       "usd",
					Status:         "succeeded",
				},
			},
		}
		payment, err := stripeWebhookPaymentFromEvent(event)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if payment.PaymentIntentID != "pi_789" {
			t.Fatalf("PaymentIntentID = %q", payment.PaymentIntentID)
		}
		if payment.Brand != "" || payment.Last4 != "" {
			t.Fatalf("expected empty card fields, got brand=%q last4=%q", payment.Brand, payment.Last4)
		}
	})

	t.Run("non-succeeded status rejected", func(t *testing.T) {
		event := stripeWebhookEvent{
			ID:   "evt_success_3",
			Type: "payment_intent.succeeded",
			Data: stripeEventData{
				Object: stripeEventObject{
					ID:             "pi_processing",
					Amount:         1000,
					AmountReceived: 0,
					Currency:       "usd",
					Status:         "processing",
				},
			},
		}
		_, err := stripeWebhookPaymentFromEvent(event)
		if err == nil {
			t.Fatal("expected error for non-succeeded status")
		}
	})

	t.Run("non-USD currency rejected", func(t *testing.T) {
		event := stripeWebhookEvent{
			ID:   "evt_success_4",
			Type: "payment_intent.succeeded",
			Data: stripeEventData{
				Object: stripeEventObject{
					ID:             "pi_eur",
					Amount:         1000,
					AmountReceived: 1000,
					Currency:       "eur",
					Status:         "succeeded",
				},
			},
		}
		_, err := stripeWebhookPaymentFromEvent(event)
		if err == nil {
			t.Fatal("expected error for non-USD currency")
		}
	})

	t.Run("zero amount received rejected", func(t *testing.T) {
		event := stripeWebhookEvent{
			ID:   "evt_success_5",
			Type: "payment_intent.succeeded",
			Data: stripeEventData{
				Object: stripeEventObject{
					ID:             "pi_zero",
					Amount:         0,
					AmountReceived: 0,
					Currency:       "usd",
					Status:         "succeeded",
				},
			},
		}
		_, err := stripeWebhookPaymentFromEvent(event)
		if err == nil {
			t.Fatal("expected error for zero amount")
		}
	})

	t.Run("missing payment intent ID rejected", func(t *testing.T) {
		event := stripeWebhookEvent{
			ID:   "evt_success_6",
			Type: "payment_intent.succeeded",
			Data: stripeEventData{
				Object: stripeEventObject{
					Amount:         1000,
					AmountReceived: 1000,
					Currency:       "usd",
					Status:         "succeeded",
				},
			},
		}
		_, err := stripeWebhookPaymentFromEvent(event)
		if err == nil {
			t.Fatal("expected error for missing payment intent ID")
		}
	})
}

func TestStripeWebhookPaymentFromEvent_Failed(t *testing.T) {
	t.Run("payment_failed maps correctly", func(t *testing.T) {
		event := stripeWebhookEvent{
			ID:   "evt_fail_1",
			Type: "payment_intent.payment_failed",
			Data: stripeEventData{
				Object: stripeEventObject{
					ID: "pi_failed",
					LastPaymentError: &struct {
						Message string `json:"message"`
					}{Message: "card_declined"},
				},
			},
		}
		payment, err := stripeWebhookPaymentFromEvent(event)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if payment.PaymentIntentID != "pi_failed" {
			t.Fatalf("PaymentIntentID = %q", payment.PaymentIntentID)
		}
		if payment.Status != "failed" {
			t.Fatalf("Status = %q, want failed", payment.Status)
		}
	})

	t.Run("payment_failed missing ID rejected", func(t *testing.T) {
		event := stripeWebhookEvent{
			ID:   "evt_fail_2",
			Type: "payment_intent.payment_failed",
			Data: stripeEventData{
				Object: stripeEventObject{},
			},
		}
		_, err := stripeWebhookPaymentFromEvent(event)
		if err == nil {
			t.Fatal("expected error for missing payment intent ID")
		}
	})
}

func TestStripeWebhookPaymentFromEvent_Refunded(t *testing.T) {
	t.Run("refunded with amount_refunded", func(t *testing.T) {
		event := stripeWebhookEvent{
			ID:   "evt_refund_1",
			Type: "payment_intent.refunded",
			Data: stripeEventData{
				Object: stripeEventObject{
					ID:             "pi_refunded",
					Amount:         5000,
					AmountRefunded: 5000,
					Currency:       "usd",
				},
			},
		}
		payment, err := stripeWebhookPaymentFromEvent(event)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if payment.PaymentIntentID != "pi_refunded" {
			t.Fatalf("PaymentIntentID = %q", payment.PaymentIntentID)
		}
		if payment.AmountCents != 5000 {
			t.Fatalf("AmountCents = %d, want 5000", payment.AmountCents)
		}
		if payment.Currency != "usd" {
			t.Fatalf("Currency = %q", payment.Currency)
		}
		if payment.Status != "refunded" {
			t.Fatalf("Status = %q, want refunded", payment.Status)
		}
	})

	t.Run("refunded falls back to amount_received when amount_refunded is zero", func(t *testing.T) {
		event := stripeWebhookEvent{
			ID:   "evt_refund_2",
			Type: "payment_intent.refunded",
			Data: stripeEventData{
				Object: stripeEventObject{
					ID:             "pi_partial_refund",
					Amount:         10000,
					AmountReceived: 8000,
					AmountRefunded: 0,
					Currency:       "usd",
				},
			},
		}
		payment, err := stripeWebhookPaymentFromEvent(event)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if payment.AmountCents != 8000 {
			t.Fatalf("AmountCents = %d, want 8000 (fallback to amount_received)", payment.AmountCents)
		}
	})

	t.Run("refunded missing ID rejected", func(t *testing.T) {
		event := stripeWebhookEvent{
			ID:   "evt_refund_3",
			Type: "payment_intent.refunded",
			Data: stripeEventData{
				Object: stripeEventObject{
					Amount:         1000,
					AmountRefunded: 1000,
					Currency:       "usd",
				},
			},
		}
		_, err := stripeWebhookPaymentFromEvent(event)
		if err == nil {
			t.Fatal("expected error for missing payment intent ID")
		}
	})
}

func TestStripeWebhookPaymentFromEvent_Unsupported(t *testing.T) {
	unsupportedTypes := []string{
		"payment_intent.created",
		"payment_intent.canceled",
		"charge.succeeded",
		"checkout.session.completed",
		"customer.subscription.updated",
		"",
		"random.nonexistent.event",
	}
	for _, et := range unsupportedTypes {
		event := stripeWebhookEvent{
			ID:   "evt_unsupported",
			Type: et,
			Data: stripeEventData{
				Object: stripeEventObject{ID: "pi_ignored"},
			},
		}
		_, err := stripeWebhookPaymentFromEvent(event)
		if err == nil {
			t.Errorf("expected error for unsupported event type %q", et)
		}
	}
}

// ---------------------------------------------------------------------------
// Integration tests: Stripe webhook HTTP handler
// ---------------------------------------------------------------------------

// testStripeConfig returns a Config suitable for Stripe webhook tests.
func testStripeConfig(t *testing.T) Config {
	t.Helper()
	tempDir := t.TempDir()
	return Config{
		Environment:         "production",
		StatePath:           filepath.Join(tempDir, "state.json"),
		TokenSymbol:         defaultTokenSymbol,
		StripeWebhookSecret: "whsec_test_webhook_secret",
	}
}

// stripeSucceededPayload returns a minimal valid payment_intent.succeeded body.
func stripeSucceededPayload(eventID, intentID string) string {
	return fmt.Sprintf(`{
		"id": "%s",
		"type": "payment_intent.succeeded",
		"created": 1700000000,
		"data": {
			"object": {
				"id": "%s",
				"object": "payment_intent",
				"amount": 2000,
				"amount_received": 2000,
				"currency": "usd",
				"status": "succeeded",
				"charges": {
					"data": [{
						"id": "ch_test",
						"payment_method_details": {
							"card": {
								"brand": "visa",
								"last4": "4242"
							}
						}
					}]
				}
			}
		}
	}`, eventID, intentID)
}

// stripeFailedPayload returns a minimal payment_intent.payment_failed body.
func stripeFailedPayload(eventID, intentID string) string {
	return fmt.Sprintf(`{
		"id": "%s",
		"type": "payment_intent.payment_failed",
		"data": {
			"object": {
				"id": "%s",
				"object": "payment_intent",
				"last_payment_error": {
					"message": "Your card was declined."
				}
			}
		}
	}`, eventID, intentID)
}

// stripeRefundedPayload returns a minimal payment_intent.refunded body.
func stripeRefundedPayload(eventID, intentID string) string {
	return fmt.Sprintf(`{
		"id": "%s",
		"type": "payment_intent.refunded",
		"data": {
			"object": {
				"id": "%s",
				"object": "payment_intent",
				"amount": 2000,
				"amount_refunded": 2000,
				"currency": "usd"
			}
		}
	}`, eventID, intentID)
}

// computeStripeHeader computes a valid Stripe-Signature header for the given payload and secret.
func computeStripeHeader(payload, secret string) string {
	timestamp := time.Now().Unix()
	sig := computeStripeSignature(timestamp, payload, secret)
	return fmt.Sprintf("t=%d,v1=%s", timestamp, sig)
}

func TestStripeWebhook_MissingSignatureHeader(t *testing.T) {
	cfg := testStripeConfig(t)
	payments := NewPaymentManager(cfg)
	store, err := NewStore(cfg, payments, NewRepoFactory(cfg), NewEmailSender(cfg))
	if err != nil {
		t.Fatal(err)
	}
	server := NewServer(cfg, store, payments)

	body := stripeSucceededPayload("evt_no_sig", "pi_no_sig")
	req := httptest.NewRequest(http.MethodPost, "/api/payments/stripe/webhook", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	server.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401; body = %s", rr.Code, rr.Body.String())
	}
}

func TestStripeWebhook_InvalidSignature(t *testing.T) {
	cfg := testStripeConfig(t)
	payments := NewPaymentManager(cfg)
	store, err := NewStore(cfg, payments, NewRepoFactory(cfg), NewEmailSender(cfg))
	if err != nil {
		t.Fatal(err)
	}
	server := NewServer(cfg, store, payments)

	body := stripeSucceededPayload("evt_bad_sig", "pi_bad_sig")
	req := httptest.NewRequest(http.MethodPost, "/api/payments/stripe/webhook", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Stripe-Signature", "t=1700000000,v1=0000000000000000000000000000000000000000")
	rr := httptest.NewRecorder()
	server.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401; body = %s", rr.Code, rr.Body.String())
	}
}

func TestStripeWebhook_SucceededRecordsSettlement(t *testing.T) {
	cfg := testStripeConfig(t)
	payments := NewPaymentManager(cfg)
	store, err := NewStore(cfg, payments, NewRepoFactory(cfg), NewEmailSender(cfg))
	if err != nil {
		t.Fatal(err)
	}

	// Register a user and create a project that references the Stripe PaymentIntent.
	auth, err := store.Register(RegisterRequest{
		Name:     "Stripe Webhook Client",
		Email:    "stripe-webhook@example.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatal(err)
	}
	projectID := "prj_stripe_webhook"
	intentID := "pi_settlement_test"
	store.mu.Lock()
	store.projects[projectID] = &Project{
		ID:               projectID,
		ClientUserID:     auth.User.ID,
		Title:            "Stripe webhook funded project",
		ClientName:       auth.User.Name,
		ClientEmail:      auth.User.Email,
		PaymentMethod:    PaymentStripe,
		PaymentStatus:    "pending",
		PaymentProvider:  "stripe",
		PaymentReference: intentID,
		BudgetCents:      2000,
		Status:           ProjectFunded,
		CreatedAt:        time.Now().UTC(),
	}
	if err := store.saveLocked(); err != nil {
		store.mu.Unlock()
		t.Fatal(err)
	}
	store.mu.Unlock()

	server := NewServer(cfg, store, payments)

	eventID := "evt_settlement_1"
	body := stripeSucceededPayload(eventID, intentID)
	sigHeader := computeStripeHeader(body, cfg.StripeWebhookSecret)

	req := httptest.NewRequest(http.MethodPost, "/api/payments/stripe/webhook", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Stripe-Signature", sigHeader)
	rr := httptest.NewRecorder()
	server.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body = %s", rr.Code, rr.Body.String())
	}

	var result stripeSettlementResult
	if err := json.Unmarshal(rr.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	if result.Status != "verified" {
		t.Fatalf("Status = %q, want verified", result.Status)
	}
	if result.Duplicate {
		t.Fatal("expected non-duplicate")
	}

	// Verify the project payment status was updated.
	store.mu.Lock()
	project := store.projects[projectID]
	store.mu.Unlock()
	if project == nil {
		t.Fatal("project not found")
	}
	if project.PaymentStatus != "verified" {
		t.Fatalf("project PaymentStatus = %q, want verified", project.PaymentStatus)
	}
	if project.PaymentProvider != "stripe:visa" {
		t.Fatalf("project PaymentProvider = %q, want stripe:visa", project.PaymentProvider)
	}

	// Verify ledger entries were created.
	ledgerEntries := 0
	for _, entry := range store.ListLedger() {
		if entry.Type == "stripe_payment_verified" || entry.Type == "token_mint" {
			ledgerEntries++
		}
	}
	if ledgerEntries != 2 {
		t.Fatalf("expected 2 ledger entries (payment_verified + token_mint), got %d", ledgerEntries)
	}
}

func TestStripeWebhook_Deduplication(t *testing.T) {
	cfg := testStripeConfig(t)
	payments := NewPaymentManager(cfg)
	store, err := NewStore(cfg, payments, NewRepoFactory(cfg), NewEmailSender(cfg))
	if err != nil {
		t.Fatal(err)
	}

	auth, err := store.Register(RegisterRequest{
		Name:     "Stripe Dedup Client",
		Email:    "stripe-dedup@example.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatal(err)
	}
	projectID := "prj_stripe_dedup"
	intentID := "pi_dedup_test"
	store.mu.Lock()
	store.projects[projectID] = &Project{
		ID:               projectID,
		ClientUserID:     auth.User.ID,
		Title:            "Stripe dedup test",
		ClientName:       auth.User.Name,
		ClientEmail:      auth.User.Email,
		PaymentMethod:    PaymentStripe,
		PaymentStatus:    "pending",
		PaymentProvider:  "stripe",
		PaymentReference: intentID,
		BudgetCents:      2000,
		Status:           ProjectFunded,
		CreatedAt:        time.Now().UTC(),
	}
	if err := store.saveLocked(); err != nil {
		store.mu.Unlock()
		t.Fatal(err)
	}
	store.mu.Unlock()

	server := NewServer(cfg, store, payments)

	eventID := "evt_dedup_1"
	body := stripeSucceededPayload(eventID, intentID)
	sigHeader := computeStripeHeader(body, cfg.StripeWebhookSecret)

	// First call — should succeed as verified.
	req1 := httptest.NewRequest(http.MethodPost, "/api/payments/stripe/webhook", strings.NewReader(body))
	req1.Header.Set("Content-Type", "application/json")
	req1.Header.Set("Stripe-Signature", sigHeader)
	rr1 := httptest.NewRecorder()
	server.Routes().ServeHTTP(rr1, req1)

	if rr1.Code != http.StatusOK {
		t.Fatalf("first call status = %d, body = %s", rr1.Code, rr1.Body.String())
	}
	var firstRes stripeSettlementResult
	json.Unmarshal(rr1.Body.Bytes(), &firstRes)
	if firstRes.Status != "verified" || firstRes.Duplicate {
		t.Fatalf("first call = {status:%q, duplicate:%t}", firstRes.Status, firstRes.Duplicate)
	}

	// Replay same event ID — should be deduplicated.
	req2 := httptest.NewRequest(http.MethodPost, "/api/payments/stripe/webhook", strings.NewReader(body))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Stripe-Signature", sigHeader)
	rr2 := httptest.NewRecorder()
	server.Routes().ServeHTTP(rr2, req2)

	if rr2.Code != http.StatusOK {
		t.Fatalf("replay status = %d, body = %s", rr2.Code, rr2.Body.String())
	}
	var replayRes stripeSettlementResult
	json.Unmarshal(rr2.Body.Bytes(), &replayRes)
	if !replayRes.Duplicate {
		t.Fatalf("replay = {status:%q, duplicate:%t}, expected duplicate=true", replayRes.Status, replayRes.Duplicate)
	}
}

func TestStripeWebhook_FailedUpdatesProjectStatus(t *testing.T) {
	cfg := testStripeConfig(t)
	payments := NewPaymentManager(cfg)
	store, err := NewStore(cfg, payments, NewRepoFactory(cfg), NewEmailSender(cfg))
	if err != nil {
		t.Fatal(err)
	}

	auth, err := store.Register(RegisterRequest{
		Name:     "Stripe Failed Client",
		Email:    "stripe-failed@example.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatal(err)
	}
	projectID := "prj_stripe_failed"
	intentID := "pi_fail_test"
	store.mu.Lock()
	store.projects[projectID] = &Project{
		ID:               projectID,
		ClientUserID:     auth.User.ID,
		Title:            "Stripe failed test",
		ClientName:       auth.User.Name,
		ClientEmail:      auth.User.Email,
		PaymentMethod:    PaymentStripe,
		PaymentStatus:    "pending",
		PaymentProvider:  "stripe",
		PaymentReference: intentID,
		BudgetCents:      2000,
		Status:           ProjectFunded,
		CreatedAt:        time.Now().UTC(),
	}
	if err := store.saveLocked(); err != nil {
		store.mu.Unlock()
		t.Fatal(err)
	}
	store.mu.Unlock()

	server := NewServer(cfg, store, payments)

	eventID := "evt_fail_test_1"
	body := stripeFailedPayload(eventID, intentID)
	sigHeader := computeStripeHeader(body, cfg.StripeWebhookSecret)

	req := httptest.NewRequest(http.MethodPost, "/api/payments/stripe/webhook", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Stripe-Signature", sigHeader)
	rr := httptest.NewRecorder()
	server.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}

	var result stripeSettlementResult
	json.Unmarshal(rr.Body.Bytes(), &result)
	if result.Status != "failed" {
		t.Fatalf("Status = %q, want failed", result.Status)
	}

	// Verify project payment status was updated.
	store.mu.Lock()
	project := store.projects[projectID]
	store.mu.Unlock()
	if project.PaymentStatus != "failed" {
		t.Fatalf("project PaymentStatus = %q, want failed", project.PaymentStatus)
	}
}

func TestStripeWebhook_RefundedUpdatesProjectStatus(t *testing.T) {
	cfg := testStripeConfig(t)
	payments := NewPaymentManager(cfg)
	store, err := NewStore(cfg, payments, NewRepoFactory(cfg), NewEmailSender(cfg))
	if err != nil {
		t.Fatal(err)
	}

	auth, err := store.Register(RegisterRequest{
		Name:     "Stripe Refund Client",
		Email:    "stripe-refund@example.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatal(err)
	}
	projectID := "prj_stripe_refund"
	intentID := "pi_refund_test"
	store.mu.Lock()
	store.projects[projectID] = &Project{
		ID:               projectID,
		ClientUserID:     auth.User.ID,
		Title:            "Stripe refund test",
		ClientName:       auth.User.Name,
		ClientEmail:      auth.User.Email,
		PaymentMethod:    PaymentStripe,
		PaymentStatus:    "pending",
		PaymentProvider:  "stripe",
		PaymentReference: intentID,
		BudgetCents:      2000,
		Status:           ProjectFunded,
		CreatedAt:        time.Now().UTC(),
	}
	if err := store.saveLocked(); err != nil {
		store.mu.Unlock()
		t.Fatal(err)
	}
	store.mu.Unlock()

	server := NewServer(cfg, store, payments)

	eventID := "evt_refund_test_1"
	body := stripeRefundedPayload(eventID, intentID)
	sigHeader := computeStripeHeader(body, cfg.StripeWebhookSecret)

	req := httptest.NewRequest(http.MethodPost, "/api/payments/stripe/webhook", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Stripe-Signature", sigHeader)
	rr := httptest.NewRecorder()
	server.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}

	var result stripeSettlementResult
	json.Unmarshal(rr.Body.Bytes(), &result)
	if result.Status != "refunded" {
		t.Fatalf("Status = %q, want refunded", result.Status)
	}

	store.mu.Lock()
	project := store.projects[projectID]
	store.mu.Unlock()
	if project.PaymentStatus != "refunded" {
		t.Fatalf("project PaymentStatus = %q, want refunded", project.PaymentStatus)
	}
}

func TestStripeWebhook_NoProjectForIntent(t *testing.T) {
	cfg := testStripeConfig(t)
	payments := NewPaymentManager(cfg)
	store, err := NewStore(cfg, payments, NewRepoFactory(cfg), NewEmailSender(cfg))
	if err != nil {
		t.Fatal(err)
	}
	server := NewServer(cfg, store, payments)

	// Send a succeeded event for a PaymentIntent that has no project.
	eventID := "evt_no_project_1"
	intentID := "pi_no_project"
	body := stripeSucceededPayload(eventID, intentID)
	sigHeader := computeStripeHeader(body, cfg.StripeWebhookSecret)

	req := httptest.NewRequest(http.MethodPost, "/api/payments/stripe/webhook", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Stripe-Signature", sigHeader)
	rr := httptest.NewRecorder()
	server.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body = %s", rr.Code, rr.Body.String())
	}
}

// ---------------------------------------------------------------------------
// Dedup persistence test: verifies that processed webhook event IDs survive
// a full store reload (save → close → reopen → dedup still works).
// ---------------------------------------------------------------------------

func TestStripeWebhook_DedupPersistenceAcrossReload(t *testing.T) {
	cfg := testStripeConfig(t)
	payments := NewPaymentManager(cfg)

	store1, err := NewStore(cfg, payments, NewRepoFactory(cfg), NewEmailSender(cfg))
	if err != nil {
		t.Fatal(err)
	}

	// Register a user and create a project.
	auth, err := store1.Register(RegisterRequest{
		Name:     "Stripe Persist Client",
		Email:    "stripe-persist@example.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatal(err)
	}
	projectID := "prj_stripe_persist"
	intentID := "pi_persist_test"
	store1.mu.Lock()
	store1.projects[projectID] = &Project{
		ID:               projectID,
		ClientUserID:     auth.User.ID,
		Title:            "Stripe persistence test",
		ClientName:       auth.User.Name,
		ClientEmail:      auth.User.Email,
		PaymentMethod:    PaymentStripe,
		PaymentStatus:    "pending",
		PaymentProvider:  "stripe",
		PaymentReference: intentID,
		BudgetCents:      2000,
		Status:           ProjectFunded,
		CreatedAt:        time.Now().UTC(),
	}
	if err := store1.saveLocked(); err != nil {
		store1.mu.Unlock()
		t.Fatal(err)
	}
	store1.mu.Unlock()

	// Process a webhook event via the HTTP handler.
	server1 := NewServer(cfg, store1, payments)
	eventID := "evt_persist_1"
	body := stripeSucceededPayload(eventID, intentID)
	sigHeader := computeStripeHeader(body, cfg.StripeWebhookSecret)

	req := httptest.NewRequest(http.MethodPost, "/api/payments/stripe/webhook", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Stripe-Signature", sigHeader)
	rr := httptest.NewRecorder()
	server1.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("first call status = %d, body = %s", rr.Code, rr.Body.String())
	}
	var firstRes stripeSettlementResult
	json.Unmarshal(rr.Body.Bytes(), &firstRes)
	if firstRes.Status != "verified" || firstRes.Duplicate {
		t.Fatalf("first call = {status:%q, duplicate:%t}", firstRes.Status, firstRes.Duplicate)
	}

	// Close store1 — the state is now on disk (JSON at cfg.StatePath).
	store1.Close()

	// Reopen from the same state file — simulates a server restart.
	store2, err := NewStore(cfg, payments, NewRepoFactory(cfg), NewEmailSender(cfg))
	if err != nil {
		t.Fatal(err)
	}
	defer store2.Close()

	// Verify dedup map was loaded from disk.
	store2.mu.RLock()
	settlement, exists := store2.paymentSettlements[eventID]
	store2.mu.RUnlock()
	if !exists {
		t.Fatal("CRITICAL: paymentSettlement not found after reload — dedup persistence is broken")
	}
	if settlement.EventID != eventID {
		t.Fatalf("settlement.EventID = %q, want %q", settlement.EventID, eventID)
	}
	if settlement.Status != "verified" {
		t.Fatalf("settlement.Status = %q, want verified", settlement.Status)
	}

	// Direct RecordStripeSettlement call with the same event ID — must return duplicate=true.
	result, err := store2.RecordStripeSettlement(eventID, stripeWebhookPayment{
		PaymentIntentID: intentID,
		AmountCents:     2000,
		Currency:        "usd",
		Status:          "succeeded",
	})
	if err != nil {
		t.Fatalf("RecordStripeSettlement on reloaded store: %v", err)
	}
	if !result.Duplicate {
		t.Fatalf("result.Duplicate = false after reload — expected true; settlement map says status=%q", result.Status)
	}
	if result.Status != "verified" {
		t.Fatalf("result.Status = %q, want verified (original status)", result.Status)
	}
	if result.EventID != eventID {
		t.Fatalf("result.EventID = %q, want %q", result.EventID, eventID)
	}
}

// ---------------------------------------------------------------------------
// RecordStripeSettlement unit tests (direct store-level, no HTTP)
// ---------------------------------------------------------------------------

func TestRecordStripeSettlement_EmptyEventID(t *testing.T) {
	cfg := testStripeConfig(t)
	store, err := NewStore(cfg, NewPaymentManager(cfg), NewRepoFactory(cfg), NewEmailSender(cfg))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	_, err = store.RecordStripeSettlement("", stripeWebhookPayment{PaymentIntentID: "pi_test", Status: "succeeded"})
	if err == nil {
		t.Fatal("expected error for empty event ID")
	}
}

func TestRecordStripeSettlement_EmptyPaymentIntentID(t *testing.T) {
	cfg := testStripeConfig(t)
	store, err := NewStore(cfg, NewPaymentManager(cfg), NewRepoFactory(cfg), NewEmailSender(cfg))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	_, err = store.RecordStripeSettlement("evt_test", stripeWebhookPayment{Status: "succeeded"})
	if err == nil {
		t.Fatal("expected error for empty PaymentIntentID")
	}
}

func TestRecordStripeSettlement_UnknownStatus(t *testing.T) {
	cfg := testStripeConfig(t)
	store, err := NewStore(cfg, NewPaymentManager(cfg), NewRepoFactory(cfg), NewEmailSender(cfg))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	_, err = store.RecordStripeSettlement("evt_test", stripeWebhookPayment{
		PaymentIntentID: "pi_test",
		Status:          "processing",
	})
	if err == nil {
		t.Fatal("expected error for unknown status 'processing'")
	}
}

func TestRecordStripeSettlement_FailedDedup(t *testing.T) {
	cfg := testStripeConfig(t)
	store, err := NewStore(cfg, NewPaymentManager(cfg), NewRepoFactory(cfg), NewEmailSender(cfg))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	auth, err := store.Register(RegisterRequest{
		Name:     "Stripe Failed Dedup",
		Email:    "stripe-failed-dedup@example.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatal(err)
	}

	intentID := "pi_failed_dedup"
	store.mu.Lock()
	store.projects["prj_fail_dedup"] = &Project{
		ID:               "prj_fail_dedup",
		ClientUserID:     auth.User.ID,
		Title:            "Failed dedup test",
		PaymentStatus:    "pending",
		PaymentProvider:  "stripe",
		PaymentReference: intentID,
		BudgetCents:      2000,
		Status:           ProjectFunded,
		CreatedAt:        time.Now().UTC(),
	}
	store.mu.Unlock()

	result1, err := store.RecordStripeSettlement("evt_fail_dedup", stripeWebhookPayment{
		PaymentIntentID: intentID,
		Status:          "failed",
	})
	if err != nil {
		t.Fatalf("first call: %v", err)
	}
	if result1.Duplicate {
		t.Fatal("first call should not be duplicate")
	}
	if result1.Status != "failed" {
		t.Fatalf("status = %q, want failed", result1.Status)
	}

	// Replay — must be duplicate.
	result2, err := store.RecordStripeSettlement("evt_fail_dedup", stripeWebhookPayment{
		PaymentIntentID: intentID,
		Status:          "failed",
	})
	if err != nil {
		t.Fatalf("replay: %v", err)
	}
	if !result2.Duplicate {
		t.Fatal("replay should be duplicate")
	}
	if result2.Status != "failed" {
		t.Fatalf("replay status = %q, want failed", result2.Status)
	}
}

func TestRecordStripeSettlement_RefundedDedup(t *testing.T) {
	cfg := testStripeConfig(t)
	store, err := NewStore(cfg, NewPaymentManager(cfg), NewRepoFactory(cfg), NewEmailSender(cfg))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	auth, err := store.Register(RegisterRequest{
		Name:     "Stripe Refund Dedup",
		Email:    "stripe-refund-dedup@example.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatal(err)
	}

	intentID := "pi_refund_dedup"
	store.mu.Lock()
	store.projects["prj_refund_dedup"] = &Project{
		ID:               "prj_refund_dedup",
		ClientUserID:     auth.User.ID,
		Title:            "Refund dedup test",
		PaymentStatus:    "pending",
		PaymentProvider:  "stripe",
		PaymentReference: intentID,
		BudgetCents:      2000,
		Status:           ProjectFunded,
		CreatedAt:        time.Now().UTC(),
	}
	store.mu.Unlock()

	result1, err := store.RecordStripeSettlement("evt_refund_dedup", stripeWebhookPayment{
		PaymentIntentID: intentID,
		Status:          "refunded",
		AmountCents:     2000,
	})
	if err != nil {
		t.Fatalf("first call: %v", err)
	}
	if result1.Duplicate {
		t.Fatal("first call should not be duplicate")
	}

	result2, err := store.RecordStripeSettlement("evt_refund_dedup", stripeWebhookPayment{
		PaymentIntentID: intentID,
		Status:          "refunded",
	})
	if err != nil {
		t.Fatalf("replay: %v", err)
	}
	if !result2.Duplicate {
		t.Fatal("replay should be duplicate")
	}
	if result2.Status != "refunded" {
		t.Fatalf("replay status = %q, want refunded", result2.Status)
	}
}
