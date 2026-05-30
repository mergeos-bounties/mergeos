package core

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

// paypalWebhookEvent represents a PayPal webhook notification
type paypalWebhookEvent struct {
	ID           string          `json:"id"`
	EventType    string          `json:"event_type"`
	EventVersion string          `json:"event_version"`
	CreateTime   string          `json:"create_time"`
	ResourceType string          `json:"resource_type"`
	Resource     json.RawMessage `json:"resource"`
}

// paypalOrderResource represents the resource payload for order events
type paypalOrderResource struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	Intent string `json:"intent"`
	Payer  *struct {
		PayerID       string `json:"payer_id"`
		EmailAddress  string `json:"email_address"`
	} `json:"payer"`
	PurchaseUnits []struct {
		Payments *struct {
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

// handlePayPalWebhook processes incoming PayPal webhook notifications.
func (s *Server) handlePayPalWebhook(w http.ResponseWriter, r *http.Request) {
	bodyBytes, err := io.ReadAll(io.LimitReader(r.Body, 256*1024))
	defer r.Body.Close()
	if err != nil {
		log.Printf("[paypal-webhook] read error: %v", err)
		writeError(w, http.StatusBadRequest, "failed to read body")
		return
	}

	var event paypalWebhookEvent
	if err := json.Unmarshal(bodyBytes, &event); err != nil {
		log.Printf("[paypal-webhook] parse error: %v", err)
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	var res paypalOrderResource
	json.Unmarshal(event.Resource, &res)

	orderID := res.ID
	status := res.Status
	currency := ""
	value := ""
	captureID := ""
	payerID := ""
	payerEmail := ""
	if res.Payer != nil {
		payerID = res.Payer.PayerID
		payerEmail = res.Payer.EmailAddress
	}
	for _, pu := range res.PurchaseUnits {
		if pu.Payments != nil && len(pu.Payments.Captures) > 0 {
			c := pu.Payments.Captures[0]
			captureID = c.ID
			if c.Status == "COMPLETED" {
				currency = c.Amount.CurrencyCode
				value = c.Amount.Value
			}
		}
	}

	cents := int64(0)
	if value != "" && currency == "USD" {
		cents, _ = payPalValueToCents(value)
	}

	if event.EventType == "PAYMENT.CAPTURE.COMPLETED" && s.cfg.PayPalReady() && cents > 0 {
		if s.store.IsPaymentReferenceUsed(orderID) {
			log.Printf("[paypal-webhook] duplicate order ignored: %s", orderID)
		} else {
			s.processPayPalPayment(r.Context(), orderID, captureID, payerID, payerEmail, cents, value, currency)
		}
	}

	log.Printf("[paypal-webhook] event=%s order=%s status=%s", event.EventType, orderID, status)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status":   "received",
		"event_id": event.ID,
	})
}

func (s *Server) processPayPalPayment(ctxReq interface{}, orderID, captureID, payerID, payerEmail string, cents int64, value, currency string) {
	tokenSymbol := normalizedTokenSymbol(s.cfg.TokenSymbol)

	ledgerRef := "paypal:" + orderID + ":" + captureID
	entry := s.store.addLedger("payment_verified", "payment:paypal", "project:reserve", cents, ledgerRef)
	log.Printf("[paypal-webhook] payment ledger created: seq=%d hash=%s", entry.Sequence, entry.EntryHash)

	mintCents := cents * 8 / 10
	mintRef := "token_mint:" + orderID
	s.store.addLedger("token_mint", "payment:paypal", "customer:mint", mintCents, mintRef)
	log.Printf("[paypal-webhook] MRG minted: %d cents (%s)", mintCents, tokenSymbol)

	s.store.addNotificationLocked("", "", "payment",
		"PayPal Payment Completed - MRG Minted",
		fmt.Sprintf("Order %s completed: %s %s. %s minted: %s", orderID, value, currency, tokenSymbol, value),
		"confirmed")

	s.store.saveLocked()
	log.Printf("[paypal-webhook] MRG minting and ledger complete for order %s", orderID)
}
