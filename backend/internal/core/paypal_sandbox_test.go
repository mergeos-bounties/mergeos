package core

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// mockPayPalSandboxServer creates a comprehensive mock PayPal sandbox server
func mockPayPalSandboxServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/v1/oauth2/token":
			// Token endpoint
			json.NewEncoder(w).Encode(map[string]interface{}{
				"access_token": "test-access-token",
				"expires_in":   3600,
				"token_type":   "Bearer",
			})

		case "/v2/checkout/orders":
			// Create order endpoint
			var reqBody map[string]interface{}
			json.NewDecoder(r.Body).Decode(&reqBody)

			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":     "test-order-123",
				"status": "CREATED",
				"links": []map[string]string{
					{
						"href":   "https://www.sandbox.paypal.com/checkoutnow?token=test-order-123",
						"rel":    "approve",
						"method": "GET",
					},
				},
			})

		case "/v2/checkout/orders/test-order-123/capture":
			// Capture order endpoint
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":     "test-capture-456",
				"status": "COMPLETED",
				"payer": map[string]string{
					"payer_id":      "test-payer-789",
					"email_address": "buyer@example.com",
				},
				"purchase_units": []map[string]interface{}{
					{
						"payments": map[string]interface{}{
							"captures": []map[string]interface{}{
								{
									"id":     "test-capture-456",
									"status": "COMPLETED",
									"amount": map[string]string{
										"currency_code": "USD",
										"value":         "150.00",
									},
								},
							},
						},
					},
				},
			})

		default:
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "not found",
			})
		}
	}))
}

// min returns the smaller of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// TestPayPalSandboxConfig_IsReady tests config validation
func TestPayPalSandboxConfig_IsReady(t *testing.T) {
	tests := []struct {
		name     string
		config   PayPalSandboxConfig
		expected bool
	}{
		{
			name: "ready with all fields",
			config: PayPalSandboxConfig{
				ClientID:     "test-client-id",
				ClientSecret: "test-client-secret",
				Enabled:      true,
			},
			expected: true,
		},
		{
			name: "not ready - disabled",
			config: PayPalSandboxConfig{
				ClientID:     "test-client-id",
				ClientSecret: "test-client-secret",
				Enabled:      false,
			},
			expected: false,
		},
		{
			name: "not ready - missing client ID",
			config: PayPalSandboxConfig{
				ClientSecret: "test-client-secret",
				Enabled:      true,
			},
			expected: false,
		},
		{
			name: "not ready - missing client secret",
			config: PayPalSandboxConfig{
				ClientID: "test-client-id",
				Enabled:  true,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.config.IsReady(); got != tt.expected {
				t.Errorf("IsReady() = %v, want %v", got, tt.expected)
			}
		})
	}
}

// TestPayPalSandboxPayment_GetAccessToken tests token retrieval
func TestPayPalSandboxPayment_GetAccessToken(t *testing.T) {
	mockServer := mockPayPalSandboxServer(t)
	defer mockServer.Close()

	cfg := PayPalSandboxConfig{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		Enabled:      true,
		BaseURL:      mockServer.URL,
	}

	storeCfg := Config{TokenSymbol: "MRG", Environment: "sandbox"}
	store, _ := NewStore(storeCfg, nil, nil, nil)

	payment := NewPayPalSandboxPayment(cfg, store)

	token, err := payment.GetAccessToken(context.Background())
	if err != nil {
		t.Fatalf("GetAccessToken() error = %v", err)
	}

	if token != "test-access-token" {
		t.Errorf("GetAccessToken() = %v, want test-access-token", token)
	}
}

// TestPayPalSandboxPayment_GetAccessToken_NotConfigured tests error when not configured
func TestPayPalSandboxPayment_GetAccessToken_NotConfigured(t *testing.T) {
	cfg := PayPalSandboxConfig{
		Enabled: false,
	}

	storeCfg := Config{TokenSymbol: "MRG", Environment: "sandbox"}
	store, _ := NewStore(storeCfg, nil, nil, nil)

	payment := NewPayPalSandboxPayment(cfg, store)

	_, err := payment.GetAccessToken(context.Background())
	if err == nil {
		t.Error("GetAccessToken() expected error when not configured")
	}
}

// TestPayPalSandboxPayment_CreateOrder tests order creation
func TestPayPalSandboxPayment_CreateOrder(t *testing.T) {
	mockServer := mockPayPalSandboxServer(t)
	defer mockServer.Close()

	cfg := PayPalSandboxConfig{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		Enabled:      true,
		BaseURL:      mockServer.URL,
	}

	storeCfg := Config{TokenSymbol: "MRG", Environment: "sandbox"}
	store, _ := NewStore(storeCfg, nil, nil, nil)

	payment := NewPayPalSandboxPayment(cfg, store)

	req := CreatePayPalOrderRequest{
		AmountCents: 15000,
		Description: "Test project funding",
		ReturnURL:   "http://localhost:3000/return",
		CancelURL:   "http://localhost:3000/cancel",
	}

	resp, err := payment.CreateOrder(context.Background(), req)
	if err != nil {
		t.Fatalf("CreateOrder() error = %v", err)
	}

	if resp.OrderID != "test-order-123" {
		t.Errorf("CreateOrder() OrderID = %v, want test-order-123", resp.OrderID)
	}
	if resp.Status != "CREATED" {
		t.Errorf("CreateOrder() Status = %v, want CREATED", resp.Status)
	}
	if resp.ApprovalURL == "" {
		t.Error("CreateOrder() ApprovalURL is empty")
	}

	t.Logf("Created order: %s", resp.OrderID)
	t.Logf("Approval URL: %s", resp.ApprovalURL)
}

// TestPayPalSandboxPayment_CaptureOrder tests order capture
func TestPayPalSandboxPayment_CaptureOrder(t *testing.T) {
	mockServer := mockPayPalSandboxServer(t)
	defer mockServer.Close()

	cfg := PayPalSandboxConfig{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		Enabled:      true,
		BaseURL:      mockServer.URL,
	}

	storeCfg := Config{TokenSymbol: "MRG", Environment: "sandbox"}
	store, _ := NewStore(storeCfg, nil, nil, nil)

	payment := NewPayPalSandboxPayment(cfg, store)

	verification, err := payment.CaptureOrder(context.Background(), "test-order-123")
	if err != nil {
		t.Fatalf("CaptureOrder() error = %v", err)
	}

	if verification.Provider != "paypal-sandbox" {
		t.Errorf("CaptureOrder() Provider = %v, want paypal-sandbox", verification.Provider)
	}
	if verification.Reference != "test-capture-456" {
		t.Errorf("CaptureOrder() Reference = %v, want test-capture-456", verification.Reference)
	}

	t.Logf("Captured payment: %s", verification.Reference)
}

// TestPayPalSandboxPayment_CreateOrder_AmountValidation tests amount validation
func TestPayPalSandboxPayment_CreateOrder_AmountValidation(t *testing.T) {
	mockServer := mockPayPalSandboxServer(t)
	defer mockServer.Close()

	cfg := PayPalSandboxConfig{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		Enabled:      true,
		BaseURL:      mockServer.URL,
	}

	storeCfg := Config{TokenSymbol: "MRG", Environment: "sandbox"}
	store, _ := NewStore(storeCfg, nil, nil, nil)

	payment := NewPayPalSandboxPayment(cfg, store)

	tests := []struct {
		name        string
		amountCents int64
		wantErr     bool
	}{
		{
			name:        "valid amount",
			amountCents: 15000,
			wantErr:     false,
		},
		{
			name:        "minimum amount",
			amountCents: 10000,
			wantErr:     false,
		},
		{
			name:        "below minimum",
			amountCents: 5000,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := CreatePayPalOrderRequest{
				AmountCents: tt.amountCents,
				Description: "Test",
				ReturnURL:   "http://localhost/return",
				CancelURL:   "http://localhost/cancel",
			}

			_, err := payment.CreateOrder(context.Background(), req)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateOrder() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestPayPalSandboxEndToEnd simulates a complete user flow
func TestPayPalSandboxEndToEnd(t *testing.T) {
	t.Log("=== PayPal Sandbox End-to-End Test ===")

	mockServer := mockPayPalSandboxServer(t)
	defer mockServer.Close()

	// Step 1: Create PayPal Sandbox Config
	cfg := PayPalSandboxConfig{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		Enabled:      true,
		BaseURL:      mockServer.URL,
	}
	t.Logf("Step 1: Config created - Enabled: %v", cfg.IsReady())

	// Step 2: Create Store
	storeCfg := Config{TokenSymbol: "MRG", Environment: "sandbox"}
	store, _ := NewStore(storeCfg, nil, nil, nil)
	t.Log("Step 2: Store created")

	// Step 3: Create Payment Handler
	payment := NewPayPalSandboxPayment(cfg, store)
	t.Log("Step 3: Payment handler created")

	// Step 4: Get Access Token
	token, err := payment.GetAccessToken(context.Background())
	if err != nil {
		t.Fatalf("GetAccessToken() error = %v", err)
	}
	t.Logf("Step 4: Got access token: %s", token[:min(len(token), 20)])

	// Step 5: Create Order
	orderReq := CreatePayPalOrderRequest{
		AmountCents: 15000,
		Description: "Test project funding",
		ReturnURL:   "http://localhost:3000/return",
		CancelURL:   "http://localhost:3000/cancel",
	}
	orderResp, err := payment.CreateOrder(context.Background(), orderReq)
	if err != nil {
		t.Fatalf("CreateOrder() error = %v", err)
	}
	t.Logf("Step 5: Order created: %s (status: %s)", orderResp.OrderID, orderResp.Status)

	// Step 6: Capture Order
	verification, err := payment.CaptureOrder(context.Background(), orderResp.OrderID)
	if err != nil {
		t.Fatalf("CaptureOrder() error = %v", err)
	}
	t.Logf("Step 6: Payment captured: %s", verification.Reference)

	t.Log("=== End-to-End Test Complete ===")
}

// TestPayPalSandboxWebhookFlow tests webhook handling
func TestPayPalSandboxWebhookFlow(t *testing.T) {
	mockVerifyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"verification_status": "SUCCESS",
		})
	}))
	defer mockVerifyServer.Close()

	storeCfg := Config{TokenSymbol: "MRG", Environment: "sandbox"}
	store, _ := NewStore(storeCfg, nil, nil, nil)

	cfg := Config{
		PayPalClientID:    "test-client-id",
		PayPalClientSecret: "test-client-secret",
		PayPalWebhookID:   "test-webhook-id",
		PayPalEnvironment: "sandbox",
	}

	payments := NewPaymentManager(cfg)

	srv := &Server{
		cfg:           cfg,
		payments:      payments,
		store:         store,
		paypalBaseURL: mockVerifyServer.URL,
	}

	webhookPayload := map[string]interface{}{
		"id":         "WH-TEST-123",
		"event_type": "PAYMENT.CAPTURE.COMPLETED",
		"resource": map[string]interface{}{
			"id":     "test-order-123",
			"status": "COMPLETED",
			"payer": map[string]string{
				"payer_id":      "test-payer-789",
				"email_address": "buyer@example.com",
			},
			"purchase_units": []map[string]interface{}{
				{
					"payments": map[string]interface{}{
						"captures": []map[string]interface{}{
							{
								"id":     "test-capture-456",
								"status": "COMPLETED",
								"amount": map[string]string{
									"currency_code": "USD",
									"value":         "150.00",
								},
							},
						},
					},
				},
			},
		},
	}

	payloadBytes, _ := json.Marshal(webhookPayload)

	req := httptest.NewRequest(http.MethodPost, "/api/payments/paypal/webhook", bytes.NewReader(payloadBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("paypal-transmission-sig", "test-signature")
	req.Header.Set("paypal-auth-algo", "SHA256withRSA")
	req.Header.Set("paypal-cert-url", "https://api.paypal.com/cert")
	req.Header.Set("paypal-transmission-id", "test-transmission-id")
	req.Header.Set("paypal-transmission-time", "2024-01-01T00:00:00Z")

	rr := httptest.NewRecorder()
	srv.handlePayPalWebhook(rr, req)

	t.Logf("Webhook response status: %d", rr.Code)
	t.Logf("Webhook response body: %s", rr.Body.String())
}
