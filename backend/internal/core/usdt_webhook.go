package core

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// USDTWebhookEvent stores a processed crypto payment webhook event.
type USDTWebhookEvent struct {
	ID             string          `json:"id"`
	Provider       string          `json:"provider"`
	EventType      string          `json:"event_type"`
	Status         string          `json:"status"`
	GatewayID      string          `json:"gateway_id"`
	AmountCents    int64           `json:"amount_cents"`
	Currency       string          `json:"currency"`
	Network        string          `json:"network"`
	TxHash         string          `json:"tx_hash"`
	SenderAddress  string          `json:"sender_address"`
	ReceiverAddress string         `json:"receiver_address"`
	SignatureValid bool            `json:"signature_valid"`
	RawPayload     json.RawMessage `json:"raw_payload"`
	Error          string          `json:"error,omitempty"`
	ProjectID      string          `json:"project_id,omitempty"`
	IdempotencyKey string          `json:"idempotency_key"`
	ProcessedAt    *time.Time      `json:"processed_at,omitempty"`
	ReceivedAt     time.Time       `json:"received_at"`
}

// USDTWebhookPayload is the expected JSON structure from the crypto gateway.
type USDTWebhookPayload struct {
	EventID       string `json:"event_id"`
	EventType     string `json:"event_type"`
	GatewayID     string `json:"gateway_id"`
	IdempotencyKey string `json:"idempotency_key"`
	Amount        string `json:"amount"`
	Currency      string `json:"currency"`
	Network       string `json:"network"`
	TxHash        string `json:"tx_hash"`
	Sender        string `json:"sender_address"`
	Receiver      string `json:"receiver_address"`
	Status        string `json:"status"`
	Timestamp     string `json:"timestamp"`
	Signature     string `json:"signature"`
	ProjectID     string `json:"project_id,omitempty"`
}

// USDTWebhookResponse is returned to the gateway after processing.
type USDTWebhookResponse struct {
	Received bool   `json:"received"`
	EventID  string `json:"event_id"`
	Status   string `json:"status"`
	Message  string `json:"message,omitempty"`
}

// usdtWebhookHandler processes incoming USDT payment gateway webhooks.
type usdtWebhookHandler struct {
	cfg   Config
	store *Store
}

func newUSDTWebhookHandler(cfg Config, store *Store) *usdtWebhookHandler {
	return &usdtWebhookHandler{cfg: cfg, store: store}
}

// handleUSDTWebhook handles POST /api/payments/usdt/webhook
func (h *usdtWebhookHandler) handleUSDTWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "only POST is accepted")
		return
	}

	// Read body for signature verification, then re-decode
	bodyBytes, err := io.ReadAll(io.LimitReader(r.Body, 1<<20)) // 1 MB limit
	if err != nil {
		writeError(w, http.StatusBadRequest, "failed to read request body")
		return
	}
	defer r.Body.Close()

	// Verify signature before any processing
	signatureHeader := r.Header.Get("X-Webhook-Signature")
	if err := h.verifySignature(bodyBytes, signatureHeader); err != nil {
		h.logWebhookEvent(bodyBytes, "", "signature_invalid", "crypto-gateway", err.Error(), "", "")
		writeError(w, http.StatusUnauthorized, "invalid webhook signature")
		return
	}

	var payload USDTWebhookPayload
	if err := json.Unmarshal(bodyBytes, &payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON payload")
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
	if strings.TrimSpace(payload.EventType) == "" {
		writeError(w, http.StatusBadRequest, "event_type is required")
		return
	}

	// Map gateway status to internal payment status
	internalStatus := mapGatewayStatus(payload.Status)
	if internalStatus == "" {
		msg := fmt.Sprintf("unknown gateway status: %s", payload.Status)
		h.logWebhookEvent(bodyBytes, payload.EventID, "unknown_status", payload.Network, msg, payload.ProjectID, payload.IdempotencyKey)
		writeError(w, http.StatusBadRequest, msg)
		return
	}

	// Idempotency check: if we already processed this idempotency key, return cached result
	existing := h.store.FindUSDTWebhookByIdempotencyKey(payload.IdempotencyKey)
	if existing != nil {
		writeJSON(w, http.StatusOK, USDTWebhookResponse{
			Received: true,
			EventID:  existing.ID,
			Status:   existing.Status,
			Message:  "duplicate webhook already processed",
		})
		return
	}

	// Parse amount to cents
	amountCents, err := parseAmountToCents(payload.Amount)
	if err != nil {
		h.logWebhookEvent(bodyBytes, payload.EventID, "invalid_amount", payload.Network, err.Error(), payload.ProjectID, payload.IdempotencyKey)
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	now := time.Now().UTC()
	event := &USDTWebhookEvent{
		ID:              payload.EventID,
		Provider:        "crypto-gateway",
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

	// Verify the receiver matches our configured address
	if h.cfg.CryptoReceiver != "" && event.ReceiverAddress != "" {
		configured := strings.TrimPrefix(strings.ToLower(h.cfg.CryptoReceiver), "0x")
		received := strings.TrimPrefix(event.ReceiverAddress, "0x")
		if configured != received {
			event.Status = "receiver_mismatch"
			event.Error = fmt.Sprintf("receiver %s does not match configured %s", event.ReceiverAddress, h.cfg.CryptoReceiver)
			h.store.SaveUSDTWebhookEvent(event)
			writeError(w, http.StatusBadRequest, event.Error)
			return
		}
	}

	// If status is confirmed, update project payment state and mint MRG
	if event.Status == "confirmed" && event.ProjectID != "" {
		if err := h.store.ApplyUSDTWebhookPayment(event); err != nil {
			event.Error = err.Error()
			event.Status = "apply_failed"
			h.store.SaveUSDTWebhookEvent(event)
			writeError(w, http.StatusInternalServerError, "failed to apply payment")
			return
		}
	}

	h.store.SaveUSDTWebhookEvent(event)

	writeJSON(w, http.StatusOK, USDTWebhookResponse{
		Received: true,
		EventID:  event.ID,
		Status:   event.Status,
		Message:  "webhook processed successfully",
	})
}

// verifySignature validates the HMAC-SHA256 webhook signature.
func (h *usdtWebhookHandler) verifySignature(body []byte, signatureHeader string) error {
	secret := h.cfg.CryptoWebhookSecret
	if secret == "" {
		// In dev mode without a configured secret, skip verification
		if h.cfg.DevPaymentEnabled {
			return nil
		}
		return fmt.Errorf("CRYPTO_WEBHOOK_SECRET is not configured")
	}

	if strings.TrimSpace(signatureHeader) == "" {
		return fmt.Errorf("missing X-Webhook-Signature header")
	}

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expectedMAC := hex.EncodeToString(mac.Sum(nil))

	// Support both "sha256=<hex>" and bare "<hex>" formats
	received := strings.TrimPrefix(signatureHeader, "sha256=")
	received = strings.TrimSpace(received)

	if !hmac.Equal([]byte(strings.ToLower(received)), []byte(strings.ToLower(expectedMAC))) {
		return fmt.Errorf("webhook signature mismatch")
	}
	return nil
}

// logWebhookEvent records a failed webhook event for debugging.
func (h *usdtWebhookHandler) logWebhookEvent(raw json.RawMessage, eventID, status, provider, errMsg, projectID, idempotencyKey string) {
	event := &USDTWebhookEvent{
		ID:             eventID,
		Provider:       provider,
		EventType:      "webhook_callback",
		Status:         status,
		RawPayload:     raw,
		Error:          errMsg,
		ProjectID:      projectID,
		IdempotencyKey: idempotencyKey,
		SignatureValid: status != "signature_invalid",
		ReceivedAt:     time.Now().UTC(),
	}
	h.store.SaveUSDTWebhookEvent(event)
}

// mapGatewayStatus maps a gateway-specific status to an internal payment status.
// Supported internal statuses: pending, confirmed, expired, failed, refunded
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
