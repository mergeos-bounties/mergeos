package core

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log"
	"net/http"
)

// USDTWebhookEvent represents a generic crypto gateway webhook
type USDTWebhookEvent struct {
	ID            string `json:"transaction_id"`
	Status        string `json:"status"`
	Currency      string `json:"currency"`
	Amount        string `json:"amount"`
	PaymentIntent string `json:"metadata_intent"`
	TxHash        string `json:"tx_hash"`
	Sender        string `json:"sender_address"`
}

type USDTWebhookHandler struct {
	cfg   Config
	store *Store
}

func NewUSDTWebhookHandler(cfg Config, store *Store) *USDTWebhookHandler {
	return &USDTWebhookHandler{cfg: cfg, store: store}
}

// VerifySignature checks HMAC SHA256 signature if a webhook secret is configured
func (h *USDTWebhookHandler) VerifySignature(payload []byte, sigHeader string) bool {
	if h.cfg.USDTWebhookSecret == "" {
		return true // Sandbox / dev mode
	}

	mac := hmac.New(sha256.New, []byte(h.cfg.USDTWebhookSecret))
	mac.Write(payload)
	expectedMAC := hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(expectedMAC), []byte(sigHeader))
}

func (h *USDTWebhookHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("USDT Webhook error reading body: %v", err)
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	sigHeader := r.Header.Get("X-Crypto-Signature")
	if !h.VerifySignature(body, sigHeader) {
		log.Printf("USDT Webhook invalid signature")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var event USDTWebhookEvent
	if err := json.Unmarshal(body, &event); err != nil {
		log.Printf("USDT Webhook error parsing body: %v", err)
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	if event.Status == "COMPLETED" || event.Status == "SUCCESS" {
		settlement, err := h.store.RecordUSDTWebhookPayment(event)
		if err != nil {
			log.Printf("USDT Webhook settlement error: %v", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		log.Printf("USDT Webhook processed: status=%s, msg=%s", settlement.Status, settlement.Message)
	}

	w.WriteHeader(http.StatusOK)
}
