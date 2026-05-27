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

	// Extract order info
	var res paypalOrderResource
	json.Unmarshal(event.Resource, &res)

	orderID := res.ID
	status := res.Status
	currency := ""
	value := ""
	for _, pu := range res.PurchaseUnits {
		if pu.Payments != nil && len(pu.Payments.Captures) > 0 {
			c := pu.Payments.Captures[0]
			if c.Status == "COMPLETED" {
				currency = c.Amount.CurrencyCode
				value = c.Amount.Value
			}
		}
	}

	// Process payment completions via existing PayPal API
	if event.EventType == "PAYMENT.CAPTURE.COMPLETED" && s.cfg.PayPalReady() {
		token, tokenErr := s.payments.payPalAccessToken(r.Context())
		if tokenErr != nil {
			log.Printf("[paypal-webhook] token error: %v", tokenErr)
		} else {
			httpReq, reqErr := http.NewRequestWithContext(r.Context(), http.MethodPost,
				s.payments.payPalBaseURL()+"/v2/checkout/orders/"+orderID+"/capture", nil)
			if reqErr == nil {
				httpReq.Header.Set("Authorization", "Bearer "+token)
				httpReq.Header.Set("Content-Type", "application/json")
				httpReq.Header.Set("PayPal-Request-Id", "mergeos-webhook-"+orderID)
				resp, doErr := s.payments.client.Do(httpReq)
				if doErr == nil {
					defer resp.Body.Close()
					if resp.StatusCode >= 200 && resp.StatusCode < 300 {
						log.Printf("[paypal-webhook] payment verified: order %s, %s %s", orderID, currency, value)
						// Record notification using existing store API
						s.store.addNotificationLocked("", "", "payment",
							"PayPal Payment Completed",
							fmt.Sprintf("Order %s completed: %s %s", orderID, currency, value),
							"confirmed")
						s.store.saveLocked()
					} else {
						body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
						log.Printf("[paypal-webhook] capture verify returned %d: %s", resp.StatusCode, string(body))
					}
				}
			}
		}
	}

	log.Printf("[paypal-webhook] event=%s order=%s status=%s", event.EventType, orderID, status)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status":   "received",
		"event_id": event.ID,
	})
}
