package core

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestPayPalWebhookEventParse(t *testing.T) {
	data := completedPayPalCaptureWebhook("WH-1", "ORD-1", "CAP-1", "100.00")
	var event paypalWebhookEvent
	if err := json.Unmarshal([]byte(data), &event); err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if event.ID != "WH-1" {
		t.Errorf("expected WH-1, got %s", event.ID)
	}
	if event.EventType != "PAYMENT.CAPTURE.COMPLETED" {
		t.Errorf("expected PAYMENT.CAPTURE.COMPLETED, got %s", event.EventType)
	}
	payment, err := paypalWebhookPaymentFromEvent(event)
	if err != nil {
		t.Fatalf("payment parse error: %v", err)
	}
	if payment.OrderID != "ORD-1" || payment.CaptureID != "CAP-1" || payment.AmountCents != 10000 {
		t.Fatalf("payment = %#v", payment)
	}
}

func TestPayPalWebhookRejectsInvalidSignature(t *testing.T) {
	paypal, _ := newPayPalWebhookVerifier(t, "FAILURE")
	defer paypal.Close()
	cfg := testPayPalWebhookConfig(t, paypal.URL)
	store, err := NewStore(cfg, NewPaymentManager(cfg), NewRepoFactory(cfg), NewEmailSender(cfg))
	if err != nil {
		t.Fatal(err)
	}
	server := NewServer(cfg, store, NewPaymentManager(cfg))

	req := httptest.NewRequest(http.MethodPost, "/api/payments/paypal/webhook", strings.NewReader(completedPayPalCaptureWebhook("WH-1", "ORDER-1", "CAP-1", "100.00")))
	addPayPalWebhookHeaders(req)
	rr := httptest.NewRecorder()

	server.Routes().ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, body = %s", rr.Code, rr.Body.String())
	}
	if got := countPayPalPaymentVerified(store); got != 0 {
		t.Fatalf("paypal payment ledger entries = %d", got)
	}
}

func TestPayPalWebhookRecordsPaymentOnce(t *testing.T) {
	paypal, verifyCalls := newPayPalWebhookVerifier(t, "SUCCESS")
	defer paypal.Close()
	cfg := testPayPalWebhookConfig(t, paypal.URL)
	payments := NewPaymentManager(cfg)
	store, err := NewStore(cfg, payments, NewRepoFactory(cfg), NewEmailSender(cfg))
	if err != nil {
		t.Fatal(err)
	}
	auth, err := store.Register(RegisterRequest{
		Name:     "Webhook Client",
		Email:    "paypal-webhook@example.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatal(err)
	}
	projectID := "prj_paypal_webhook"
	store.mu.Lock()
	store.projects[projectID] = &Project{
		ID:               projectID,
		ClientUserID:     auth.User.ID,
		Title:            "Webhook funded project",
		ClientName:       auth.User.Name,
		ClientEmail:      auth.User.Email,
		PaymentMethod:    PaymentPayPal,
		PaymentStatus:    "pending",
		PaymentProvider:  "paypal",
		PaymentReference: "ORDER-1",
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
	body := completedPayPalCaptureWebhook("WH-1", "ORDER-1", "CAP-1", "100.00")
	first := postPayPalWebhook(t, server, body)
	if first.Code != http.StatusOK {
		t.Fatalf("first status = %d, body = %s", first.Code, first.Body.String())
	}
	var firstSettlement payPalWebhookSettlement
	if err := json.Unmarshal(first.Body.Bytes(), &firstSettlement); err != nil {
		t.Fatal(err)
	}
	if firstSettlement.Status != "verified" || firstSettlement.ProjectID != projectID || firstSettlement.Duplicate {
		t.Fatalf("first settlement = %#v", firstSettlement)
	}
	if got := countPayPalPaymentVerified(store); got != 1 {
		t.Fatalf("paypal payment ledger entries after first webhook = %d", got)
	}

	replay := postPayPalWebhook(t, server, body)
	if replay.Code != http.StatusOK {
		t.Fatalf("replay status = %d, body = %s", replay.Code, replay.Body.String())
	}
	var replaySettlement payPalWebhookSettlement
	if err := json.Unmarshal(replay.Body.Bytes(), &replaySettlement); err != nil {
		t.Fatal(err)
	}
	if replaySettlement.Status != "duplicate" || !replaySettlement.Duplicate {
		t.Fatalf("replay settlement = %#v", replaySettlement)
	}
	if got := countPayPalPaymentVerified(store); got != 1 {
		t.Fatalf("paypal payment ledger entries after replay = %d", got)
	}
	if *verifyCalls != 2 {
		t.Fatalf("verify calls = %d", *verifyCalls)
	}
}

func TestPayPalWebhookAllEventTypesParseEnvelope(t *testing.T) {
	types := []string{
		"PAYMENT.CAPTURE.COMPLETED",
		"PAYMENT.CAPTURE.DENIED",
		"PAYMENT.CAPTURE.DECLINED",
		"PAYMENT.CAPTURE.REFUNDED",
		"CHECKOUT.ORDER.APPROVED",
		"CHECKOUT.ORDER.COMPLETED",
	}
	for _, et := range types {
		data := `{"id":"test","event_type":"` + et + `","resource":{"id":"ORD-1"}}`
		var event paypalWebhookEvent
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			t.Errorf("parse error for %s: %v", et, err)
		}
		if event.EventType != et {
			t.Errorf("expected %s, got %s", et, event.EventType)
		}
	}
}

func newPayPalWebhookVerifier(t *testing.T, status string) (*httptest.Server, *int) {
	t.Helper()
	verifyCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v1/oauth2/token":
			if got := r.Header.Get("Authorization"); !strings.HasPrefix(got, "Basic ") {
				t.Fatalf("missing basic auth header")
			}
			_, _ = w.Write([]byte(`{"access_token":"test-paypal-token"}`))
		case "/v1/notifications/verify-webhook-signature":
			verifyCalls++
			if got := r.Header.Get("Authorization"); got != "Bearer test-paypal-token" {
				t.Fatalf("authorization header = %q", got)
			}
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode verify request: %v", err)
			}
			if body["webhook_id"] != "WH-MERGEOS" {
				t.Fatalf("webhook_id = %#v", body["webhook_id"])
			}
			if body["webhook_event"] == nil {
				t.Fatal("missing webhook_event")
			}
			_, _ = w.Write([]byte(`{"verification_status":"` + status + `"}`))
		default:
			t.Fatalf("unexpected PayPal verifier path %s", r.URL.Path)
		}
	}))
	return server, &verifyCalls
}

func testPayPalWebhookConfig(t *testing.T, paypalURL string) Config {
	t.Helper()
	tempDir := t.TempDir()
	return Config{
		Environment:        "production",
		StatePath:          filepath.Join(tempDir, "state.json"),
		TokenSymbol:        defaultTokenSymbol,
		PayPalEnvironment:  paypalURL,
		PayPalClientID:     "paypal-client",
		PayPalClientSecret: "paypal-secret",
		PayPalWebhookID:    "WH-MERGEOS",
	}
}

func postPayPalWebhook(t *testing.T, server *Server, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/payments/paypal/webhook", strings.NewReader(body))
	addPayPalWebhookHeaders(req)
	rr := httptest.NewRecorder()
	server.Routes().ServeHTTP(rr, req)
	return rr
}

func addPayPalWebhookHeaders(req *http.Request) {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("PayPal-Auth-Algo", "SHA256withRSA")
	req.Header.Set("PayPal-Cert-Url", "https://api-m.sandbox.paypal.com/certs/test")
	req.Header.Set("PayPal-Transmission-Id", "transmission-1")
	req.Header.Set("PayPal-Transmission-Sig", "signature")
	req.Header.Set("PayPal-Transmission-Time", "2026-06-06T00:00:00Z")
}

func completedPayPalCaptureWebhook(eventID, orderID, captureID, value string) string {
	return `{"id":"` + eventID + `","event_type":"PAYMENT.CAPTURE.COMPLETED","resource":{"id":"` + captureID + `","status":"COMPLETED","amount":{"currency_code":"USD","value":"` + value + `"},"supplementary_data":{"related_ids":{"order_id":"` + orderID + `"}}}}`
}

func countPayPalPaymentVerified(store *Store) int {
	count := 0
	for _, entry := range store.ListLedger() {
		if entry.Type == "payment_verified" && strings.EqualFold(entry.FromAccount, "payment:paypal") {
			count++
		}
	}
	return count
}
