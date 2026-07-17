package core

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

// PayPalSandboxConfig holds PayPal sandbox configuration
type PayPalSandboxConfig struct {
	ClientID     string
	ClientSecret string
	WebhookID    string
	BaseURL      string // https://api-m.sandbox.paypal.com
	Enabled      bool
}

// NewPayPalSandboxConfig creates config from environment variables
func NewPayPalSandboxConfig() PayPalSandboxConfig {
	return PayPalSandboxConfig{
		ClientID:     os.Getenv("PAYPAL_SANDBOX_CLIENT_ID"),
		ClientSecret: os.Getenv("PAYPAL_SANDBOX_CLIENT_SECRET"),
		WebhookID:    os.Getenv("PAYPAL_SANDBOX_WEBHOOK_ID"),
		BaseURL:      "https://api-m.sandbox.paypal.com",
		Enabled:      os.Getenv("PAYPAL_SANDBOX_ENABLED") == "true",
	}
}

// IsReady returns true if sandbox is properly configured
func (c PayPalSandboxConfig) IsReady() bool {
	return c.Enabled && c.ClientID != "" && c.ClientSecret != ""
}

// PayPalSandboxPayment handles sandbox payment flow
type PayPalSandboxPayment struct {
	cfg     PayPalSandboxConfig
	client  *http.Client
	store   *Store
}

// NewPayPalSandboxPayment creates a new sandbox payment handler
func NewPayPalSandboxPayment(cfg PayPalSandboxConfig, store *Store) *PayPalSandboxPayment {
	return &PayPalSandboxPayment{
		cfg: cfg,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		store: store,
	}
}

// GetAccessToken retrieves OAuth2 token from PayPal sandbox
func (p *PayPalSandboxPayment) GetAccessToken(ctx context.Context) (string, error) {
	if !p.cfg.IsReady() {
		return "", fmt.Errorf("PayPal sandbox not configured")
	}

	// Create Basic Auth header
	auth := base64Encode(p.cfg.ClientID + ":" + p.cfg.ClientSecret)

	body := "grant_type=client_credentials"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		p.cfg.BaseURL+"/v1/oauth2/token",
		strings.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Basic "+auth)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := p.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return "", fmt.Errorf("token request failed: %d %s", resp.StatusCode, string(respBody))
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", err
	}

	return tokenResp.AccessToken, nil
}

// CreateOrder creates a PayPal order in sandbox
func (p *PayPalSandboxPayment) CreateOrder(ctx context.Context, req CreatePayPalOrderRequest) (*CreatePayPalOrderResponse, error) {
	if !p.cfg.IsReady() {
		return nil, fmt.Errorf("PayPal sandbox not configured")
	}

	if req.AmountCents < 10000 {
		return nil, fmt.Errorf("amount must be at least 100 USD (got %d cents)", req.AmountCents)
	}

	token, err := p.GetAccessToken(ctx)
	if err != nil {
		return nil, err
	}

	amount := fmt.Sprintf("%.2f", float64(req.AmountCents)/100)

	orderBody := map[string]interface{}{
		"intent": "CAPTURE",
		"purchase_units": []map[string]interface{}{
			{
				"reference_id": fmt.Sprintf("mergeos-%d", time.Now().UnixNano()),
				"description":  req.Description,
				"amount": map[string]string{
					"currency_code": "USD",
					"value":         amount,
				},
			},
		},
		"application_context": map[string]string{
			"return_url": req.ReturnURL,
			"cancel_url": req.CancelURL,
		},
	}

	bodyBytes, _ := json.Marshal(orderBody)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		p.cfg.BaseURL+"/v2/checkout/orders",
		bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+token)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("PayPal-Request-Id", fmt.Sprintf("mergeos-%d", time.Now().UnixNano()))

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return nil, fmt.Errorf("create order failed: %d %s", resp.StatusCode, string(respBody))
	}

	var orderResp struct {
		ID     string `json:"id"`
		Status string `json:"status"`
		Links  []struct {
			Href string `json:"href"`
			Rel  string `json:"rel"`
		} `json:"links"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&orderResp); err != nil {
		return nil, err
	}

	approvalURL := ""
	for _, link := range orderResp.Links {
		if link.Rel == "approve" {
			approvalURL = link.Href
			break
		}
	}

	return &CreatePayPalOrderResponse{
		OrderID:     orderResp.ID,
		ApprovalURL: approvalURL,
		Status:      orderResp.Status,
	}, nil
}

// CaptureOrder captures/verifies a PayPal order
func (p *PayPalSandboxPayment) CaptureOrder(ctx context.Context, orderID string) (*PaymentVerification, error) {
	token, err := p.GetAccessToken(ctx)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		p.cfg.BaseURL+"/v2/checkout/orders/"+orderID+"/capture",
		nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("PayPal-Request-Id", fmt.Sprintf("mergeos-capture-%d", time.Now().UnixNano()))

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return nil, fmt.Errorf("capture order failed: %d %s", resp.StatusCode, string(respBody))
	}

	var captureResp struct {
		ID     string `json:"id"`
		Status string `json:"status"`
		Payer  struct {
			PayerID string `json:"payer_id"`
			Email   string `json:"email_address"`
		} `json:"payer"`
		PurchaseUnits []struct {
			Payments struct {
				Captures []struct {
					ID     string `json:"id"`
					Status string `json:"status"`
					Amount struct {
						CurrencyCode string `json:"currency_code"`
						Value        string `json:"value"`
					} `json:"amount"`
				} `json:"captures"`
			} `json:"payments"`
		} `json:"purchase_units"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&captureResp); err != nil {
		return nil, err
	}

	// Record in ledger
	if captureResp.Status == "COMPLETED" {
		p.recordPayment(ctx, captureResp)
	}

	return &PaymentVerification{
		Provider:  "paypal-sandbox",
		Reference: captureResp.ID,
	}, nil
}

// captureResponse represents PayPal capture response
type captureResponse struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	Payer  struct {
		PayerID string `json:"payer_id"`
		Email   string `json:"email_address"`
	} `json:"payer"`
	PurchaseUnits []struct {
		Payments struct {
			Captures []struct {
				ID     string `json:"id"`
				Status string `json:"status"`
				Amount struct {
					CurrencyCode string `json:"currency_code"`
					Value        string `json:"value"`
				} `json:"amount"`
			} `json:"captures"`
		} `json:"payments"`
	} `json:"purchase_units"`
}

// recordPayment records a successful payment in the ledger
func (p *PayPalSandboxPayment) recordPayment(ctx context.Context, captureResp captureResponse) {
	// Get capture amount
	currency := ""
	value := ""
	for _, pu := range captureResp.PurchaseUnits {
		for _, c := range pu.Payments.Captures {
			if c.Status == "COMPLETED" {
				currency = c.Amount.CurrencyCode
				value = c.Amount.Value
			}
		}
	}

	// Parse amount to cents
	amountCents := int64(0)
	if value != "" {
		// Simple parsing: "150.00" -> 15000
		var amount float64
		if _, err := fmt.Sscanf(value, "%f", &amount); err == nil {
			amountCents = int64(amount * 100)
		}
	}

	// Create ledger entry
	entry := LedgerEntry{
		Type:        "payment",
		FromAccount: captureResp.Payer.Email,
		ToAccount:   "mergeos",
		AmountCents: amountCents,
		Reference:   captureResp.ID,
		CreatedAt:   time.Now(),
	}

	// Add to store
	p.store.mu.Lock()
	p.store.addLedger(entry.Type, entry.FromAccount, entry.ToAccount, entry.AmountCents, entry.Reference)
	p.store.mu.Unlock()

	log.Printf("[paypal-sandbox] Payment recorded: %s %s (order: %s)", currency, value, captureResp.ID)
}

// base64Encode encodes string to base64
func base64Encode(s string) string {
	return base64.StdEncoding.EncodeToString([]byte(s))
}
