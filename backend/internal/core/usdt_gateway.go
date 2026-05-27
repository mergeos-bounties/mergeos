package core

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sort"
	"strings"
	"time"
)

// --- Models ---

// USDTWebhookEvent represents a persisted USDT payment webhook event.
type USDTWebhookEvent struct {
	ID              string          `json:"id"`
	Provider        string          `json:"provider"`
	EventType       string          `json:"event_type"`
	Status          string          `json:"status"`
	GatewayID       string          `json:"gateway_id"`
	AmountCents     int64           `json:"amount_cents"`
	Currency        string          `json:"currency"`
	Network         string          `json:"network"`
	TxHash          string          `json:"tx_hash"`
	SenderAddress   string          `json:"sender_address"`
	ReceiverAddress string          `json:"receiver_address"`
	SignatureValid  bool            `json:"signature_valid"`
	RawPayload      json.RawMessage `json:"raw_payload"`
	Error           string          `json:"error,omitempty"`
	ProjectID       string          `json:"project_id,omitempty"`
	IdempotencyKey  string          `json:"idempotency_key"`
	ProcessedAt     *time.Time      `json:"processed_at,omitempty"`
	ReceivedAt      time.Time       `json:"received_at"`
}

// USDTWebhookPayload is the expected JSON structure from the crypto gateway callback.
type USDTWebhookPayload struct {
	EventID        string `json:"event_id"`
	EventType      string `json:"event_type"`
	GatewayID      string `json:"gateway_id"`
	IdempotencyKey string `json:"idempotency_key"`
	Amount         string `json:"amount"`
	Currency       string `json:"currency"`
	Network        string `json:"network"`
	TxHash         string `json:"tx_hash"`
	Sender         string `json:"sender_address"`
	Receiver       string `json:"receiver_address"`
	Status         string `json:"status"`
	Timestamp      string `json:"timestamp"`
	Signature      string `json:"signature"`
	ProjectID      string `json:"project_id,omitempty"`
}

// USDTWebhookResponse is returned to the gateway after processing.
type USDTWebhookResponse struct {
	Received bool   `json:"received"`
	EventID  string `json:"event_id"`
	Status   string `json:"status"`
	Message  string `json:"message,omitempty"`
}

// USDTGatewayProvider defines the interface for pluggable USDT payment gateway providers.
// Each provider handles its own webhook payload parsing, signature verification, and status mapping.
type USDTGatewayProvider interface {
	// Name returns the provider identifier.
	Name() string

	// VerifySignature validates the webhook request authenticity.
	VerifySignature(body []byte, r *http.Request) error

	// ParsePayload extracts the standard USDTWebhookPayload from the provider-specific body.
	ParsePayload(body []byte) (*USDTWebhookPayload, error)
}

// --- Generic HMAC Provider ---

// GenericHMACProvider implements USDTGatewayProvider using a shared HMAC-SHA256 secret.
// It supports both "sha256=<hex>" and bare "<hex>" signature formats.
// In dev mode (no secret + DevPaymentEnabled), signature verification is skipped.
type GenericHMACProvider struct {
	name   string
	secret string
	devBypass bool
}

// NewGenericHMACProvider creates a provider that verifies webhooks via HMAC-SHA256.
func NewGenericHMACProvider(name, secret string, devBypass bool) *GenericHMACProvider {
	return &GenericHMACProvider{name: name, secret: secret, devBypass: devBypass}
}

func (p *GenericHMACProvider) Name() string { return p.name }

func (p *GenericHMACProvider) VerifySignature(body []byte, r *http.Request) error {
	if p.secret == "" {
		if p.devBypass {
			return nil // dev mode: skip verification
		}
		return fmt.Errorf("CRYPTO_WEBHOOK_SECRET is not configured")
	}

	rawSig := strings.TrimSpace(r.Header.Get("X-Webhook-Signature"))
	if rawSig == "" {
		rawSig = strings.TrimSpace(r.Header.Get("X-Signature"))
	}
	if rawSig == "" {
		return fmt.Errorf("missing webhook signature header")
	}

	// Support both "sha256=<hex>" and bare "<hex>"
	rawSig = strings.TrimPrefix(rawSig, "sha256=")
	rawSig = strings.TrimSpace(rawSig)

	mac := hmac.New(sha256.New, []byte(p.secret))
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(strings.ToLower(rawSig)), []byte(strings.ToLower(expected))) {
		// Constant-time compare so no leak
		return fmt.Errorf("webhook signature mismatch")
	}
	return nil
}

func (p *GenericHMACProvider) ParsePayload(body []byte) (*USDTWebhookPayload, error) {
	var payload USDTWebhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("invalid JSON payload: %w", err)
	}
	return &payload, nil
}

// --- USDT Gateway Manager ---

// USDTGatewayManager manages USDT payment gateway providers and webhook processing.
type USDTGatewayManager struct {
	providers []USDTGatewayProvider
	store     *Store
}

// NewUSDTGatewayManager creates a manager with providers registered from config.
func NewUSDTGatewayManager(cfg Config, store *Store) *USDTGatewayManager {
	m := &USDTGatewayManager{store: store}

	// Register the default generic HMAC provider
	devMode := cfg.DevPaymentEnabled
	m.providers = append(m.providers, NewGenericHMACProvider("crypto-gateway", cfg.CryptoWebhookSecret, devMode))

	return m
}

// RegisterProvider adds an additional gateway provider.
func (m *USDTGatewayManager) RegisterProvider(p USDTGatewayProvider) {
	m.providers = append(m.providers, p)
}

// handleWebhook processes an incoming USDT webhook callback.
func (m *USDTGatewayManager) handleWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "only POST is accepted")
		return
	}

	bodyBytes, err := io.ReadAll(io.LimitReader(r.Body, 1<<20)) // 1 MB limit
	if err != nil {
		writeError(w, http.StatusBadRequest, "failed to read request body")
		return
	}
	defer r.Body.Close()

	// Try each registered provider until one verifies the signature and parses the payload
	var payload *USDTWebhookPayload
	var providerName string
	var parseErr error

	for _, prov := range m.providers {
		if sigErr := prov.VerifySignature(bodyBytes, r); sigErr != nil {
			parseErr = sigErr
			continue
		}
		p, pErr := prov.ParsePayload(bodyBytes)
		if pErr != nil {
			parseErr = pErr
			continue
		}
		payload = p
		providerName = prov.Name()
		parseErr = nil
		break
	}

	if payload == nil {
		msg := "webhook verification failed"
		if parseErr != nil {
			msg = parseErr.Error()
		}
		// Still log the event for debugging
		m.logWebhookEvent(bodyBytes, "", "verification_failed", providerName, msg, "", "")
		writeError(w, http.StatusUnauthorized, msg)
		return
	}

	// Validate required fields
	if strings.TrimSpace(payload.EventID) == "" {
		writeError(w, http.StatusBadRequest, "event_id is required")
		return
	}
	if strings.TrimSpace(payload.IdempotencyKey) == "" {
		writeError(w, http.StatusBadRequest, "idempotency_key is required")
		return
	}

	// Idempotency check
	existing := m.store.FindUSDTWebhookByIdempotencyKey(payload.IdempotencyKey)
	if existing != nil {
		writeJSON(w, http.StatusOK, USDTWebhookResponse{
			Received: true,
			EventID:  existing.ID,
			Status:   existing.Status,
			Message:  "duplicate webhook already processed",
		})
		return
	}

	// Map gateway status to internal status
	internalStatus := mapGatewayStatus(payload.Status)
	if internalStatus == "" {
		msg := fmt.Sprintf("unknown gateway status: %s", payload.Status)
		m.logWebhookEvent(bodyBytes, payload.EventID, "unknown_status", providerName, msg, payload.ProjectID, payload.IdempotencyKey)
		writeError(w, http.StatusBadRequest, msg)
		return
	}

	// Parse amount to cents
	amountCents, err := parseAmountToCents(payload.Amount)
	if err != nil {
		m.logWebhookEvent(bodyBytes, payload.EventID, "invalid_amount", providerName, err.Error(), payload.ProjectID, payload.IdempotencyKey)
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	now := time.Now().UTC()
	event := &USDTWebhookEvent{
		ID:              payload.EventID,
		Provider:        providerName,
		EventType:       payload.EventType,
		Status:          internalStatus,
		GatewayID:       payload.GatewayID,
		AmountCents:     amountCents,
		Currency:        strings.ToUpper(strings.TrimSpace(payload.Currency)),
		Network:         strings.TrimSpace(payload.Network),
		TxHash:          strings.TrimSpace(payload.TxHash),
		SenderAddress:   strings.ToLower(strings.TrimSpace(payload.Sender)),
		ReceiverAddress: strings.ToLower(strings.TrimSpace(payload.Receiver)),
		SignatureValid:  true,
		RawPayload:      json.RawMessage(bodyBytes),
		ProjectID:       strings.TrimSpace(payload.ProjectID),
		IdempotencyKey:  payload.IdempotencyKey,
		ProcessedAt:     &now,
		ReceivedAt:      now,
	}

	// Apply payment if confirmed
	if event.Status == "confirmed" && event.ProjectID != "" {
		if appErr := m.store.ApplyUSDTWebhookPayment(event); appErr != nil {
			event.Error = appErr.Error()
			event.Status = "apply_failed"
			if saveErr := m.store.SaveUSDTWebhookEvent(event); saveErr != nil {
				log.Printf("[usdt-gateway] failed to save apply_failed event: %v", saveErr)
			}
			writeError(w, http.StatusInternalServerError, "failed to apply payment")
			return
		}
	}

	if err := m.store.SaveUSDTWebhookEvent(event); err != nil {
		log.Printf("[usdt-gateway] failed to save event %s: %v", event.ID, err)
	}

	writeJSON(w, http.StatusOK, USDTWebhookResponse{
		Received: true,
		EventID:  event.ID,
		Status:   event.Status,
		Message:  "webhook processed successfully",
	})
}

func (m *USDTGatewayManager) logWebhookEvent(raw json.RawMessage, eventID, status, provider, errMsg, projectID, idempotencyKey string) {
	event := &USDTWebhookEvent{
		ID:             eventID,
		Provider:       provider,
		EventType:      "webhook_callback",
		Status:         status,
		RawPayload:     raw,
		Error:          errMsg,
		ProjectID:      projectID,
		IdempotencyKey: idempotencyKey,
		SignatureValid: status != "verification_failed",
		ReceivedAt:     time.Now().UTC(),
	}
	if err := m.store.SaveUSDTWebhookEvent(event); err != nil {
		log.Printf("[usdt-gateway] failed to save log event: %v", err)
	}
}

// --- Status Mapping ---

// mapGatewayStatus maps a gateway-specific status to an internal payment status.
// Supported: pending, confirmed, expired, failed, refunded
func mapGatewayStatus(gatewayStatus string) string {
	switch strings.ToLower(strings.TrimSpace(gatewayStatus)) {
	case "pending", "waiting", "processing", "confirming":
		return "pending"
	case "confirmed", "completed", "success", "successful", "paid":
		return "confirmed"
	case "expired", "timeout", "timed_out":
		return "expired"
	case "failed", "error", "rejected", "cancelled", "canceled":
		return "failed"
	case "refunded", "reversed":
		return "refunded"
	default:
		return ""
	}
}

// --- Amount Parsing ---

// parseAmountToCents converts a string amount (e.g. "100.00") to cents.
func parseAmountToCents(amountStr string) (int64, error) {
	amountStr = strings.TrimSpace(amountStr)
	if amountStr == "" {
		return 0, fmt.Errorf("amount is empty")
	}

	parts := strings.Split(amountStr, ".")
	dollarsStr := parts[0]
	if dollarsStr == "" {
		dollarsStr = "0"
	}

	var dollars, cents int64
	if _, err := fmt.Sscanf(dollarsStr, "%d", &dollars); err != nil {
		return 0, fmt.Errorf("invalid dollar amount: %q", amountStr)
	}

	if len(parts) > 1 {
		fraction := parts[1]
		if len(fraction) > 2 {
			fraction = fraction[:2]
		}
		for len(fraction) < 2 {
			fraction += "0"
		}
		if _, err := fmt.Sscanf(fraction, "%d", &cents); err != nil {
			return 0, fmt.Errorf("invalid cent amount: %q", amountStr)
		}
	}

	return dollars*100 + cents, nil
}

// --- Store helpers for USDTWebhookEvent ---

// SaveUSDTWebhookEvent persists a USDT webhook event.
func (s *Store) SaveUSDTWebhookEvent(event *USDTWebhookEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.usdtWebhookEvents == nil {
		s.usdtWebhookEvents = make(map[string]*USDTWebhookEvent)
	}
	s.usdtWebhookEvents[event.ID] = event
	return s.saveLocked()
}

// FindUSDTWebhookByIdempotencyKey looks up a webhook event by its idempotency key.
func (s *Store) FindUSDTWebhookByIdempotencyKey(key string) *USDTWebhookEvent {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.usdtWebhookEvents == nil {
		return nil
	}
	for _, event := range s.usdtWebhookEvents {
		if event.IdempotencyKey == key {
			return event
		}
	}
	return nil
}

// ListUSDTWebhookEvents returns all USDT webhook events in reverse chronological order.
func (s *Store) ListUSDTWebhookEvents() []*USDTWebhookEvent {
	s.mu.RLock()
	defer s.mu.RUnlock()
	events := make([]*USDTWebhookEvent, 0, len(s.usdtWebhookEvents))
	for _, event := range s.usdtWebhookEvents {
		events = append(events, event)
	}
	sort.Slice(events, func(i, j int) bool {
		return events[i].ReceivedAt.After(events[j].ReceivedAt)
	})
	return events
}

// ApplyUSDTWebhookPayment credits a project from a confirmed USDT webhook payment.
func (s *Store) ApplyUSDTWebhookPayment(event *USDTWebhookEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	project, ok := s.projects[event.ProjectID]
	if !ok {
		return fmt.Errorf("project %s not found", event.ProjectID)
	}

	// Verify the webhook amount matches the project budget
	if event.AmountCents != project.BudgetCents {
		return fmt.Errorf("webhook amount %d cents does not match project budget %d cents",
			event.AmountCents, project.BudgetCents)
	}

	// Update project payment state
	project.PaymentStatus = "verified"
	project.PaymentProvider = "usdt-webhook:" + event.Provider
	project.PaymentReference = event.TxHash

	// Add ledger entries
	clientProjectAccount := "client:" + project.ClientUserID + ":project:" + project.ID
	s.addLedger("usdt_payment_confirmed", "payment:usdt-webhook", clientProjectAccount,
		project.BudgetCents, event.TxHash)

	return s.saveLocked()
}
