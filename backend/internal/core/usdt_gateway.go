package core

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"sync"
)

// USDTGatewayProvider defines the abstraction for a USDT crypto payment gateway
type USDTGatewayProvider interface {
	VerifyWebhookSignature(payload []byte, signature string) bool
	ProcessEvent(event USDTWebhookEvent) error
}

// USDTWebhookEvent represents the payload coming from the USDT gateway
type USDTWebhookEvent struct {
	EventID       string `json:"event_id"`
	TransactionID string `json:"transaction_id"`
	Status        string `json:"status"` // e.g., "completed", "failed"
	Amount        string `json:"amount"`
	Currency      string `json:"currency"`
	WalletAddress string `json:"wallet_address"`
}

type defaultUSDTGateway struct {
	secretKey     string
	processedIDs  map[string]bool
	processedLock sync.RWMutex
}

// NewUSDTGateway creates a new instance of the default USDT gateway provider
func NewUSDTGateway(secretKey string) USDTGatewayProvider {
	return &defaultUSDTGateway{
		secretKey:    secretKey,
		processedIDs: make(map[string]bool),
	}
}

// VerifyWebhookSignature verifies the HMAC-SHA256 signature of the webhook payload
func (g *defaultUSDTGateway) VerifyWebhookSignature(payload []byte, signature string) bool {
	mac := hmac.New(sha256.New, []byte(g.secretKey))
	mac.Write(payload)
	expectedMAC := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expectedMAC), []byte(signature))
}

// ProcessEvent processes the USDT event idempotently
func (g *defaultUSDTGateway) ProcessEvent(event USDTWebhookEvent) error {
	g.processedLock.Lock()
	defer g.processedLock.Unlock()

	// Idempotency check: Have we already processed this event?
	if g.processedIDs[event.EventID] {
		log.Printf("USDT Gateway: Event %s already processed. Ignoring.", event.EventID)
		return nil
	}

	// Process the event based on its status
	switch event.Status {
	case "completed":
		log.Printf("USDT Gateway: Payment of %s %s completed for TX %s.", event.Amount, event.Currency, event.TransactionID)
		// TODO: Hook into core ledger/database to mark payment as paid
	case "failed":
		log.Printf("USDT Gateway: Payment failed for TX %s.", event.TransactionID)
	default:
		log.Printf("USDT Gateway: Unknown status %s for TX %s.", event.Status, event.TransactionID)
	}

	// Mark as processed
	g.processedIDs[event.EventID] = true
	return nil
}

// USDTWebhookHandler returns an HTTP handler for receiving USDT webhooks
func USDTWebhookHandler(provider USDTGatewayProvider) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		payload, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Error reading request body", http.StatusInternalServerError)
			return
		}
		defer r.Body.Close()

		signature := r.Header.Get("X-USDT-Signature")
		if signature == "" {
			http.Error(w, "Missing signature header", http.StatusUnauthorized)
			return
		}

		if !provider.VerifyWebhookSignature(payload, signature) {
			http.Error(w, "Invalid signature", http.StatusUnauthorized)
			return
		}

		var event USDTWebhookEvent
		if err := json.Unmarshal(payload, &event); err != nil {
			http.Error(w, "Invalid payload", http.StatusBadRequest)
			return
		}

		if err := provider.ProcessEvent(event); err != nil {
			http.Error(w, "Error processing event", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("Webhook received successfully"))
	}
}
