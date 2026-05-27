package core

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// CryptoGatewayProvider defines the interface for crypto payment providers.
// Implementations handle invoice creation, webhook verification, and status mapping.
type CryptoGatewayProvider interface {
	// Name returns the provider identifier (e.g., "nowpayments", "mock").
	Name() string

	// CreateInvoice creates a payment invoice for the given amount in cents.
	// Returns invoice details including payment address and ID.
	CreateInvoice(req CryptoInvoiceRequest) (*CryptoInvoiceResponse, error)

	// VerifyWebhook validates the webhook signature/body authenticity.
	VerifyWebhook(r *http.Request, body []byte) error

	// ParseWebhookEvent extracts the payment event from the webhook payload.
	ParseWebhookEvent(body []byte) (*CryptoWebhookEvent, error)
}

// CryptoInvoiceRequest holds the parameters for creating a crypto invoice.
type CryptoInvoiceRequest struct {
	AmountCents int64  `json:"amount_cents"`
	Currency    string `json:"currency"`     // e.g., "usdt"
	Network     string `json:"network"`      // e.g., "erc20", "trc20", "bep20"
	ProjectID   string `json:"project_id"`
	Description string `json:"description"`
	CallbackURL string `json:"callback_url"`
}

// CryptoInvoiceResponse holds the invoice details returned by the provider.
type CryptoInvoiceResponse struct {
	InvoiceID    string `json:"invoice_id"`
	PayAddress   string `json:"pay_address"`
	AmountCrypto string `json:"amount_crypto"` // amount in crypto units
	Currency     string `json:"currency"`
	Network      string `json:"network"`
	ExpiresAt    string `json:"expires_at"`
	Status       string `json:"status"`
}

// CryptoWebhookEvent represents a parsed webhook event from the payment provider.
type CryptoWebhookEvent struct {
	InvoiceID     string `json:"invoice_id"`
	Status        string `json:"status"` // pending, confirming, confirmed, failed, expired, refunded
	TxHash        string `json:"tx_hash"`
	AmountCents   int64  `json:"amount_cents"`
	AmountCrypto  string `json:"amount_crypto"`
	Currency      string `json:"currency"`
	Network       string `json:"network"`
	Confirmations int    `json:"confirmations"`
}

// CryptoPaymentStatus maps provider-specific statuses to MergeOS payment states.
type CryptoPaymentStatus string

const (
	CryptoStatusPending   CryptoPaymentStatus = "pending"
	CryptoStatusConfirming CryptoPaymentStatus = "confirming"
	CryptoStatusConfirmed CryptoPaymentStatus = "confirmed"
	CryptoStatusFailed    CryptoPaymentStatus = "failed"
	CryptoStatusExpired   CryptoPaymentStatus = "expired"
	CryptoStatusRefunded  CryptoPaymentStatus = "refunded"
)

// MapProviderStatus converts a provider-specific status to a MergeOS status.
func MapProviderStatus(providerStatus string) CryptoPaymentStatus {
	switch strings.ToLower(providerStatus) {
	case "pending", "waiting", "new":
		return CryptoStatusPending
	case "confirming", "processing", "partially_paid":
		return CryptoStatusConfirming
	case "confirmed", "completed", "finished", "paid":
		return CryptoStatusConfirmed
	case "failed", "error", "cancelled":
		return CryptoStatusFailed
	case "expired", "timeout":
		return CryptoStatusExpired
	case "refunded":
		return CryptoStatusRefunded
	default:
		return CryptoStatusPending
	}
}

// CryptoGatewayManager manages crypto payment providers and webhook processing.
type CryptoGatewayManager struct {
	cfg          Config
	providers    map[string]CryptoGatewayProvider
	webhookSecret string
	mu           sync.RWMutex
	// processedEvents tracks webhook event IDs for idempotency
	processedEvents map[string]time.Time
}

// NewCryptoGatewayManager creates a new gateway manager with configured providers.
func NewCryptoGatewayManager(cfg Config) *CryptoGatewayManager {
	m := &CryptoGatewayManager{
		cfg:             cfg,
		providers:       make(map[string]CryptoGatewayProvider),
		webhookSecret:   cfg.CryptoWebhookSecret,
		processedEvents: make(map[string]time.Time),
	}

	// Register mock provider for sandbox/test mode
	if cfg.CryptoGatewayProvider == "mock" || cfg.Environment != "production" {
		m.providers["mock"] = NewMockCryptoProvider()
	}

	return m
}

// RegisterProvider adds a new crypto gateway provider.
func (m *CryptoGatewayManager) RegisterProvider(name string, p CryptoGatewayProvider) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.providers[name] = p
}

// GetProvider returns the configured crypto gateway provider.
func (m *CryptoGatewayManager) GetProvider() (CryptoGatewayProvider, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	providerName := m.cfg.CryptoGatewayProvider
	if providerName == "" {
		providerName = "mock"
	}

	p, ok := m.providers[providerName]
	if !ok {
		return nil, fmt.Errorf("crypto gateway provider %q not registered", providerName)
	}
	return p, nil
}

// IsProcessed checks if a webhook event has already been processed (idempotency).
func (m *CryptoGatewayManager) IsProcessed(eventID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, exists := m.processedEvents[eventID]
	return exists
}

// MarkProcessed records a webhook event as processed.
func (m *CryptoGatewayManager) MarkProcessed(eventID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.processedEvents[eventID] = time.Now()

	// Cleanup old events (keep last hour)
	cutoff := time.Now().Add(-1 * time.Hour)
	for id, t := range m.processedEvents {
		if t.Before(cutoff) {
			delete(m.processedEvents, id)
		}
	}
}

// VerifyWebhookSecret verifies the webhook signature using HMAC-SHA256.
func (m *CryptoGatewayManager) VerifyWebhookSecret(body []byte, signature string) error {
	if m.webhookSecret == "" {
		return nil // no secret configured, skip verification
	}
	signature = strings.TrimPrefix(signature, "sha256=")
	mac := hmac.New(sha256.Sum256, []byte(m.webhookSecret))
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(signature), []byte(expected)) {
		return errors.New("invalid webhook signature")
	}
	return nil
}

// CleanupProcessedEvents removes old processed event records.
func (m *CryptoGatewayManager) CleanupProcessedEvents() {
	m.mu.Lock()
	defer m.mu.Unlock()
	cutoff := time.Now().Add(-24 * time.Hour)
	for id, t := range m.processedEvents {
		if t.Before(cutoff) {
			delete(m.processedEvents, id)
		}
	}
}

// --- Mock Provider for Sandbox/Testing ---

// MockCryptoProvider is a deterministic test provider that simulates crypto payments.
type MockCryptoProvider struct {
	mu       sync.Mutex
	invoices map[string]*CryptoInvoiceResponse
}

// NewMockCryptoProvider creates a new mock crypto provider.
func NewMockCryptoProvider() *MockCryptoProvider {
	return &MockCryptoProvider{
		invoices: make(map[string]*CryptoInvoiceResponse),
	}
}

func (m *MockCryptoProvider) Name() string { return "mock" }

func (m *MockCryptoProvider) CreateInvoice(req CryptoInvoiceRequest) (*CryptoInvoiceResponse, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	invoiceID := fmt.Sprintf("mock-inv-%d", time.Now().UnixNano())
	resp := &CryptoInvoiceResponse{
		InvoiceID:    invoiceID,
		PayAddress:   "0x0000000000000000000000000000000000000001",
		AmountCrypto: fmt.Sprintf("%.6f", float64(req.AmountCents)/100.0),
		Currency:     req.Currency,
		Network:      req.Network,
		ExpiresAt:    time.Now().Add(30 * time.Minute).Format(time.RFC3339),
		Status:       "pending",
	}
	m.invoices[invoiceID] = resp
	return resp, nil
}

func (m *MockCryptoProvider) VerifyWebhook(r *http.Request, body []byte) error {
	// Mock provider accepts all webhooks in test mode
	return nil
}

func (m *MockCryptoProvider) ParseWebhookEvent(body []byte) (*CryptoWebhookEvent, error) {
	var event CryptoWebhookEvent
	if err := json.Unmarshal(body, &event); err != nil {
		return nil, fmt.Errorf("invalid webhook payload: %w", err)
	}
	if event.InvoiceID == "" {
		return nil, errors.New("webhook event missing invoice_id")
	}
	return &event, nil
}

// SimulatePayment is a test helper that creates a mock webhook payload.
func (m *MockCryptoProvider) SimulatePayment(invoiceID string, status string, txHash string) ([]byte, error) {
	m.mu.Lock()
	inv, ok := m.invoices[invoiceID]
	m.mu.Unlock()
	if !ok {
		return nil, fmt.Errorf("invoice %s not found", invoiceID)
	}

	event := CryptoWebhookEvent{
		InvoiceID:    invoiceID,
		Status:       status,
		TxHash:       txHash,
		AmountCrypto: inv.AmountCrypto,
		Currency:     inv.Currency,
		Network:      inv.Network,
	}
	return json.Marshal(event)
}

// --- Webhook Handler ---

// cryptoWebhook handles incoming crypto payment webhook callbacks.
func (s *Server) cryptoWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20)) // 1MB limit
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "failed to read body"})
		return
	}

	if s.cryptoGateway == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "crypto gateway not configured"})
		return
	}

	// Verify webhook signature
	signature := r.Header.Get("X-Webhook-Signature")
	if signature == "" {
		signature = r.Header.Get("X-Signature")
	}
	if err := s.cryptoGateway.VerifyWebhookSecret(body, signature); err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid signature"})
		return
	}

	// Parse the event
	provider, err := s.cryptoGateway.GetProvider()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if err := provider.VerifyWebhook(r, body); err != nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "provider verification failed"})
		return
	}

	event, err := provider.ParseWebhookEvent(body)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	// Idempotency check
	eventID := fmt.Sprintf("%s-%s-%s", event.InvoiceID, event.Status, event.TxHash)
	if s.cryptoGateway.IsProcessed(eventID) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "already_processed"})
		return
	}

	// Map status and update payment state
	status := MapProviderStatus(event.Status)
	switch status {
	case CryptoStatusConfirmed:
		// Update project payment state, mint MRG credit, add ledger entry
		if err := s.processCryptoPayment(event); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
	case CryptoStatusFailed, CryptoStatusExpired:
		// Mark payment as failed/expired
		if err := s.processCryptoPaymentFailure(event); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
	}

	s.cryptoGateway.MarkProcessed(eventID)
	writeJSON(w, http.StatusOK, map[string]string{
		"status":  "ok",
		"invoice": event.InvoiceID,
		"mapped":  string(status),
	})
}

// processCryptoPayment handles a confirmed crypto payment.
func (s *Server) processCryptoPayment(event *CryptoWebhookEvent) error {
	// Find project by invoice ID and update payment state
	// This integrates with the existing store/payment flow
	return s.store.ConfirmCryptoPayment(event.InvoiceID, event.TxHash, event.AmountCents)
}

// processCryptoPaymentFailure handles a failed/expired crypto payment.
func (s *Server) processCryptoPaymentFailure(event *CryptoWebhookEvent) error {
	return s.store.FailCryptoPayment(event.InvoiceID, string(MapProviderStatus(event.Status)))
}

// createCryptoInvoice handles creating a new crypto payment invoice.
func (s *Server) createCryptoInvoice(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}

	if s.cryptoGateway == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "crypto gateway not configured"})
		return
	}

	var req CryptoInvoiceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		return
	}

	if req.AmountCents < 100 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "minimum amount is $1.00"})
		return
	}

	if req.Currency == "" {
		req.Currency = "usdt"
	}
	if req.Network == "" {
		req.Network = s.cfg.CryptoDefaultNetwork
		if req.Network == "" {
			req.Network = "erc20"
		}
	}

	provider, err := s.cryptoGateway.GetProvider()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	resp, err := provider.CreateInvoice(req)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, resp)
}
