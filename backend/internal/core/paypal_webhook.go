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
		PayerID      string `json:"payer_id"`
		EmailAddress string `json:"email_address"`
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

// paypalVerifySignatureRequest represents the request body for PayPal's Verify Webhook Signature API
type paypalVerifySignatureRequest struct {
	AuthAlgo         string          `json:"auth_algo"`
	CertURL          string          `json:"cert_url"`
	TransmissionID   string          `json:"transmission_id"`
	TransmissionSig  string          `json:"transmission_sig"`
	TransmissionTime string          `json:"transmission_time"`
	WebhookID        string          `json:"webhook_id"`
	WebhookEvent     json.RawMessage `json:"webhook_event"`
}

// paypalVerifySignatureResponse represents the response from PayPal's Verify Webhook Signature API
type paypalVerifySignatureResponse struct {
	VerificationStatus string `json:"verification_status"`
}

// verifyPayPalSignature calls PayPal's Verify Webhook Signature API to verify the webhook signature
func (s *Server) verifyPayPalSignature(ctx context.Context, headers http.Header, body []byte) (bool, error) {
	if !s.cfg.PayPalWebhookReady() {
		return false, fmt.Errorf("PayPal webhook not configured: missing webhook ID")
	}

	// Extract required headers
	authAlgo := headers.Get("paypal-auth-algo")
	certURL := headers.Get("paypal-cert-url")
	transmissionID := headers.Get("paypal-transmission-id")
	transmissionSig := headers.Get("paypal-transmission-sig")
	transmissionTime := headers.Get("paypal-transmission-time")

	// Build verification request
	verifyReq := paypalVerifySignatureRequest{
		AuthAlgo:         authAlgo,
		CertURL:          certURL,
		TransmissionID:   transmissionID,
		TransmissionSig:  transmissionSig,
		TransmissionTime: transmissionTime,
		WebhookID:        s.cfg.PayPalWebhookID,
		WebhookEvent:     body,
	}

	reqBody, err := json.Marshal(verifyReq)
	if err != nil {
		return false, fmt.Errorf("failed to marshal verification request: %w", err)
	}

	// Call PayPal Verify Webhook Signature API
	apiURL := s.payments.payPalBaseURL() + "/v1/notifications/verify-webhook-signature"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewReader(reqBody))
	if err != nil {
		return false, fmt.Errorf("failed to create verification request: %w", err)
	}

	// Set Basic auth header
	auth := base64.StdEncoding.EncodeToString([]byte(s.cfg.PayPalClientID + ":" + s.cfg.PayPalClientSecret))
	req.Header.Set("Authorization", "Basic "+auth)
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.payments.client.Do(req)
	if err != nil {
		return false, fmt.Errorf("failed to call PayPal verification API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return false, fmt.Errorf("PayPal verification API returned %d: %s", resp.StatusCode, string(body))
	}

	var verifyResp paypalVerifySignatureResponse
	if err := json.NewDecoder(resp.Body).Decode(&verifyResp); err != nil {
		return false, fmt.Errorf("failed to decode verification response: %w", err)
	}

	return verifyResp.VerificationStatus == "SUCCESS", nil
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

	// Check for required signature header
	transmissionSig := r.Header.Get("paypal-transmission-sig")
	if transmissionSig == "" {
		log.Printf("[paypal-webhook] missing signature header")
		writeError(w, http.StatusBadRequest, "missing PayPal signature header")
		return
	}

	// Verify signature using PayPal's API
	isValid, err := s.verifyPayPalSignature(r.Context(), r.Header, bodyBytes)
	if err != nil {
		log.Printf("[paypal-webhook] signature verification error: %v", err)
		writeError(w, http.StatusInternalServerError, "signature verification failed")
		return
	}
	if !isValid {
		log.Printf("[paypal-webhook] invalid signature rejected")
		writeError(w, http.StatusForbidden, "invalid webhook signature")
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

	// Process payment events
	switch event.EventType {
	case "PAYMENT.CAPTURE.COMPLETED":
		if s.cfg.PayPalReady() {
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

	case "PAYMENT.CAPTURE.DENIED", "PAYMENT.CAPTURE.DECLINED":
		log.Printf("[paypal-webhook] payment denied/declined: order %s", orderID)
		s.store.addNotificationLocked("", "", "payment",
			"PayPal Payment Denied",
			fmt.Sprintf("Order %s was denied/declined", orderID),
			"denied")
		s.store.saveLocked()

	case "PAYMENT.CAPTURE.REFUNDED":
		log.Printf("[paypal-webhook] payment refunded: order %s", orderID)
		s.store.addNotificationLocked("", "", "payment",
			"PayPal Payment Refunded",
			fmt.Sprintf("Order %s was refunded", orderID),
			"refunded")
		s.store.saveLocked()

	default:
		log.Printf("[paypal-webhook] unhandled event type: %s", event.EventType)
	}

	log.Printf("[paypal-webhook] event=%s order=%s status=%s", event.EventType, orderID, status)

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status":   "received",
		"event_id": event.ID,
	})
}
