package core

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type stripeWebhookEvent struct {
	ID        string          `json:"id"`
	Type      string          `json:"type"`
	Created   int64           `json:"created"`
	Data      stripeEventData `json:"data"`
}

type stripeEventData struct {
	Object json.RawMessage `json:"object"`
}

type stripePaymentIntent struct {
	ID               string `json:"id"`
	Amount           int64  `json:"amount"`
	AmountReceived   int64  `json:"amount_received"`
	AmountCaptured   int64  `json:"amount_captured"`
	Currency         string `json:"currency"`
	Status           string `json:"status"`
	Description      string `json:"description"`
}

type stripeWebhookPayment struct {
	PaymentIntentID string
	Status          string
	Currency        string
	AmountCents     int64
	EventType       string
}

func (s *Server) handleStripeWebhook(w http.ResponseWriter, r *http.Request) {
	bodyBytes, err := io.ReadAll(io.LimitReader(r.Body, 256*1024))
	if r.Body != nil {
		defer r.Body.Close()
	}
	if err != nil {
		log.Printf("[stripe-webhook] read error: %v", err)
		writeError(w, http.StatusBadRequest, "failed to read body")
		return
	}

	verified := s.verifyStripeWebhookSignature(r.Header, bodyBytes)
	if !verified {
		log.Printf("[stripe-webhook] signature verification failed")
		writeError(w, http.StatusUnauthorized, "invalid stripe webhook signature")
		return
	}

	var event stripeWebhookEvent
	if err := json.Unmarshal(bodyBytes, &event); err != nil {
		log.Printf("[stripe-webhook] parse error: %v", err)
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	payment, err := stripeWebhookPaymentFromEvent(event)
	if err != nil {
		log.Printf("[stripe-webhook] ignored event=%s id=%s: %v", event.Type, event.ID, err)
		writeJSON(w, http.StatusOK, map[string]any{
			"status":     "ignored",
			"event_id":   event.ID,
			"event_type": event.Type,
			"reason":     err.Error(),
		})
		return
	}

	settlement, err := s.store.RecordStripeWebhookPayment(event.ID, payment)
	if err != nil {
		log.Printf("[stripe-webhook] settlement error: %v", err)
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if settlement.Status == "verified" && !settlement.Duplicate {
		s.broadcastLiveFeedEvent("payment_verified")
	}

	log.Printf("[stripe-webhook] event=%s pi=%s status=%s duplicate=%t", event.Type, payment.PaymentIntentID, settlement.Status, settlement.Duplicate)
	writeJSON(w, http.StatusOK, settlement)
}

func (s *Server) verifyStripeWebhookSignature(headers http.Header, body []byte) bool {
	secret := strings.TrimSpace(s.cfg.StripeWebhookSecret)
	if secret == "" {
		if s.cfg.Environment != "production" {
			return true
		}
		log.Printf("[stripe-webhook] STRIPE_WEBHOOK_SECRET is required in production")
		return false
	}

	sigHeader := strings.TrimSpace(headers.Get("stripe-signature"))
	if sigHeader == "" {
		return false
	}

	var timestamp int64
	var signatures []string

	for _, part := range strings.Split(sigHeader, ",") {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, "t=") {
			ts, err := strconv.ParseInt(strings.TrimPrefix(part, "t="), 10, 64)
			if err != nil {
				continue
			}
			timestamp = ts
		} else if strings.HasPrefix(part, "v1=") {
			signatures = append(signatures, strings.TrimPrefix(part, "v1="))
		}
	}

	if timestamp == 0 || len(signatures) == 0 {
		return false
	}

	now := time.Now().Unix()
	if now-timestamp > 300 {
		log.Printf("[stripe-webhook] timestamp is too old: %d", timestamp)
		return false
	}

	signedPayload := fmt.Sprintf("%d.%s", timestamp, string(body))
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(signedPayload))
	expected := hex.EncodeToString(mac.Sum(nil))

	for _, sig := range signatures {
		if hmac.Equal([]byte(sig), []byte(expected)) {
			return true
		}
	}
	return false
}

func stripeWebhookPaymentFromEvent(event stripeWebhookEvent) (stripeWebhookPayment, error) {
	eventType := strings.TrimSpace(event.Type)
	switch eventType {
	case "payment_intent.succeeded", "payment_intent.payment_failed", "payment_intent.refunded", "payment_intent.canceled":
		var pi stripePaymentIntent
		if err := json.Unmarshal(event.Data.Object, &pi); err != nil {
			return stripeWebhookPayment{}, fmt.Errorf("failed to parse payment intent: %w", err)
		}
		payment := stripeWebhookPayment{
			PaymentIntentID: strings.TrimSpace(pi.ID),
			Status:          strings.TrimSpace(pi.Status),
			Currency:        strings.TrimSpace(pi.Currency),
			AmountCents:     pi.Amount,
			EventType:       eventType,
		}
		return validateStripeWebhookPayment(payment)
	default:
		return stripeWebhookPayment{}, errors.New("event type is not a supported stripe payment event")
	}
}

func validateStripeWebhookPayment(payment stripeWebhookPayment) (stripeWebhookPayment, error) {
	if !strings.HasPrefix(payment.PaymentIntentID, "pi_") {
		return stripeWebhookPayment{}, errors.New("payment intent id must start with pi_")
	}
	if payment.PaymentIntentID == "" {
		return stripeWebhookPayment{}, errors.New("payment intent id is required")
	}
	if payment.AmountCents <= 0 {
		return stripeWebhookPayment{}, errors.New("amount must be positive")
	}
	currency := strings.ToLower(strings.TrimSpace(payment.Currency))
	if currency != "usd" {
		return stripeWebhookPayment{}, errors.New("currency must be usd")
	}
	payment.Currency = "usd"
	return payment, nil
}
