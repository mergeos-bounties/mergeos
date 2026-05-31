package core

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// mockPayPalVerifyServer creates a mock PayPal verification server for testing
func mockPayPalVerifyServer(shouldSucceed bool) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/notifications/verify-webhook-signature" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		var reqBody map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		status := "FAILURE"
		if shouldSucceed {
			status = "SUCCESS"
		}

		resp := map[string]string{
			"verification_status": status,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
}

func TestHandlePayPalWebhook_MissingSignature(t *testing.T) {
	cfg := Config{
		PayPalClientID:     "test-client-id",
		PayPalClientSecret: "test-client-secret",
		PayPalWebhookID:    "test-webhook-id",
	}
	srv := &Server{cfg: cfg}

	body := map[string]interface{}{
		"id":         "WH-TEST",
		"event_type": "PAYMENT.CAPTURE.COMPLETED",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/payments/paypal/webhook", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	// No paypal-transmission-sig header

	rr := httptest.NewRecorder()
	srv.handlePayPalWebhook(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, rr.Code)
	}

	if !bytes.Contains(rr.Body.Bytes(), []byte("missing")) {
		t.Errorf("Expected error message about missing signature")
	}
}

func TestHandlePayPalWebhook_InvalidSignature(t *testing.T) {
	// Create mock PayPal server that returns FAILURE
	mockServer := mockPayPalVerifyServer(false)
	defer mockServer.Close()

	cfg := Config{
		PayPalClientID:     "test-client-id",
		PayPalClientSecret: "test-client-secret",
		PayPalWebhookID:    "test-webhook-id",
		PayPalEnvironment:  "sandbox",
	}

	srv := &Server{
		cfg:           cfg,
		paypalBaseURL: mockServer.URL,
	}

	body := map[string]interface{}{
		"id":         "WH-TEST",
		"event_type": "PAYMENT.CAPTURE.COMPLETED",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/payments/paypal/webhook", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("paypal-transmission-sig", "invalid-sig")
	req.Header.Set("paypal-auth-algo", "SHA256withRSA")
	req.Header.Set("paypal-cert-url", "https://api.paypal.com/cert")
	req.Header.Set("paypal-transmission-id", "test-transmission-id")
	req.Header.Set("paypal-transmission-time", "2024-01-01T00:00:00Z")

	rr := httptest.NewRecorder()
	srv.handlePayPalWebhook(rr, req)

	// Should return 403 for invalid signature
	if rr.Code != http.StatusForbidden {
		t.Errorf("Expected status %d, got %d", http.StatusForbidden, rr.Code)
	}
}

func TestHandlePayPalWebhook_ValidEvent(t *testing.T) {
	// Create mock PayPal server that returns SUCCESS
	mockServer := mockPayPalVerifyServer(true)
	defer mockServer.Close()

	cfg := Config{
		PayPalClientID:     "test-client-id",
		PayPalClientSecret: "test-client-secret",
		PayPalWebhookID:    "test-webhook-id",
		PayPalEnvironment:  "sandbox",
	}

	payments := NewPaymentManager(cfg)
	payments.client = mockServer.Client()

	srv := &Server{
		cfg:           cfg,
		payments:      payments,
		paypalBaseURL: mockServer.URL,
		store:         newStore(cfg),
	}

	body := map[string]interface{}{
		"id":            "WH-TEST",
		"event_type":    "PAYMENT.CAPTURE.COMPLETED",
		"event_version": "1.0",
		"resource": map[string]interface{}{
			"id":     "ORDER-123",
			"status": "COMPLETED",
			"purchase_units": []map[string]interface{}{
				{
					"payments": map[string]interface{}{
						"captures": []map[string]interface{}{
							{
								"id":     "CAPTURE-123",
								"status": "COMPLETED",
								"amount": map[string]interface{}{
									"currency_code": "USD",
									"value":         "100.00",
								},
							},
						},
					},
				},
			},
		},
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/payments/paypal/webhook", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("paypal-transmission-sig", "valid-sig")
	req.Header.Set("paypal-auth-algo", "SHA256withRSA")
	req.Header.Set("paypal-cert-url", "https://api.paypal.com/cert")
	req.Header.Set("paypal-transmission-id", "test-transmission-id")
	req.Header.Set("paypal-transmission-time", "2024-01-01T00:00:00Z")

	rr := httptest.NewRecorder()
	srv.handlePayPalWebhook(rr, req)

	// Should return 200 for valid event
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rr.Code)
	}

	var resp map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if resp["status"] != "received" {
		t.Errorf("Expected status 'received', got '%s'", resp["status"])
	}
}

func TestHandlePayPalWebhook_PaymentDenied(t *testing.T) {
	mockServer := mockPayPalVerifyServer(true)
	defer mockServer.Close()

	cfg := Config{
		PayPalClientID:     "test-client-id",
		PayPalClientSecret: "test-client-secret",
		PayPalWebhookID:    "test-webhook-id",
		PayPalEnvironment:  "sandbox",
	}

	srv := &Server{
		cfg:           cfg,
		paypalBaseURL: mockServer.URL,
		store:         newStore(cfg),
	}

	body := map[string]interface{}{
		"id":         "WH-TEST",
		"event_type": "PAYMENT.CAPTURE.DENIED",
		"resource": map[string]interface{}{
			"id":     "ORDER-123",
			"status": "DENIED",
		},
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/payments/paypal/webhook", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("paypal-transmission-sig", "valid-sig")
	req.Header.Set("paypal-auth-algo", "SHA256withRSA")
	req.Header.Set("paypal-cert-url", "https://api.paypal.com/cert")
	req.Header.Set("paypal-transmission-id", "test-transmission-id")
	req.Header.Set("paypal-transmission-time", "2024-01-01T00:00:00Z")

	rr := httptest.NewRecorder()
	srv.handlePayPalWebhook(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rr.Code)
	}
}

func TestHandlePayPalWebhook_PaymentDeclined(t *testing.T) {
	mockServer := mockPayPalVerifyServer(true)
	defer mockServer.Close()

	cfg := Config{
		PayPalClientID:     "test-client-id",
		PayPalClientSecret: "test-client-secret",
		PayPalWebhookID:    "test-webhook-id",
		PayPalEnvironment:  "sandbox",
	}

	srv := &Server{
		cfg:           cfg,
		paypalBaseURL: mockServer.URL,
		store:         newStore(cfg),
	}

	body := map[string]interface{}{
		"id":         "WH-TEST",
		"event_type": "PAYMENT.CAPTURE.DECLINED",
		"resource": map[string]interface{}{
			"id":     "ORDER-123",
			"status": "DECLINED",
		},
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/payments/paypal/webhook", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("paypal-transmission-sig", "valid-sig")
	req.Header.Set("paypal-auth-algo", "SHA256withRSA")
	req.Header.Set("paypal-cert-url", "https://api.paypal.com/cert")
	req.Header.Set("paypal-transmission-id", "test-transmission-id")
	req.Header.Set("paypal-transmission-time", "2024-01-01T00:00:00Z")

	rr := httptest.NewRecorder()
	srv.handlePayPalWebhook(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rr.Code)
	}
}

func TestHandlePayPalWebhook_PaymentRefunded(t *testing.T) {
	mockServer := mockPayPalVerifyServer(true)
	defer mockServer.Close()

	cfg := Config{
		PayPalClientID:     "test-client-id",
		PayPalClientSecret: "test-client-secret",
		PayPalWebhookID:    "test-webhook-id",
		PayPalEnvironment:  "sandbox",
	}

	srv := &Server{
		cfg:           cfg,
		paypalBaseURL: mockServer.URL,
		store:         newStore(cfg),
	}

	body := map[string]interface{}{
		"id":         "WH-TEST",
		"event_type": "PAYMENT.CAPTURE.REFUNDED",
		"resource": map[string]interface{}{
			"id":     "ORDER-123",
			"status": "REFUNDED",
		},
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/api/payments/paypal/webhook", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("paypal-transmission-sig", "valid-sig")
	req.Header.Set("paypal-auth-algo", "SHA256withRSA")
	req.Header.Set("paypal-cert-url", "https://api.paypal.com/cert")
	req.Header.Set("paypal-transmission-id", "test-transmission-id")
	req.Header.Set("paypal-transmission-time", "2024-01-01T00:00:00Z")

	rr := httptest.NewRecorder()
	srv.handlePayPalWebhook(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, rr.Code)
	}
}

func TestVerifyPayPalSignature_NotConfigured(t *testing.T) {
	cfg := Config{
		PayPalClientID:     "test-client-id",
		PayPalClientSecret: "test-client-secret",
		// PayPalWebhookID is empty - webhook not ready
	}
	srv := &Server{cfg: cfg}

	headers := http.Header{}
	headers.Set("paypal-transmission-sig", "test-sig")

	valid, err := srv.verifyPayPalSignature(nil, headers, []byte("{}"))
	if err == nil {
		t.Error("Expected error when webhook not configured")
	}
	if valid {
		t.Error("Expected verification to fail when not configured")
	}
}
