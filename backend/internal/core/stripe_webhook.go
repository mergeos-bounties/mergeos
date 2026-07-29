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

// stripeWebhookEvent represents a Stripe webhook notification.
type stripeWebhookEvent struct {
	ID         string          `json:"id"`
	Type       string          `json:"type"`
	APIVersion string          `json:"api_version,omitempty"`
	Created    int64           `json:"created,omitempty"`
	Data       struct {
		Object json.RawMessage `json:"object"`
	} `json:"data"`
}

// stripePaymentIntent represents the payment_intent object in a Stripe event.
type stripePaymentIntent struct {
	ID              string `json:"id"`
	Amount          int64  `json:"amount"`
	AmountReceived  int64  `json:"amount_received,omitempty"`
	AmountCaptured  int64  `json:"amount_captured,omitempty"`
	Currency        string `json:"currency"`
	Status          string `json:"status"`
	Description     string `json:"description,omitempty"`
	Metadata        map[string]string `json:"metadata,omitempty"`
	ClientSecret    string `json:"client_secret,omitempty"`
	PaymentMethod   string `json:"payment_method,omitempty"`
	LatestCharge    string `json:"latest_charge,omitempty"`
}

// stripeWebhookPayment is the normalized payment info extracted from a Stripe event.
type stripeWebhookPayment struct {
	PaymentIntentID string
	Status          string
	Currency        string
	AmountCents     int64
	Description     string
	Metadata        map[string]string
	ChargeID        string
}

// stripeSignatureTTL is the maximum age of a Stripe webhook timestamp (5 minutes).
const stripeSignatureTTL = 5 * time.Minute

// stripeSignatureHeader is the Stripe webhook signature header name.
const stripeSignatureHeader = "Stripe-Signature"

// verifyStripeWebhookSignature verifies the Stripe webhook signature.
// Stripe sends: t=timestamp,v1=signature[,v1=additional_sig,...]
// Verification: HMAC-SHA256(webhook_secret, timestamp.payload)
func verifyStripeWebhookSignature(payload []byte, sigHeader string, webhookSecret string) error {
	if webhookSecret == "" {
		return nil // Dev mode: skip verification
	}

	// Parse the signature header
	params := parseStripeSignatureHeader(sigHeader)
	if params == nil {
		return errors.New("stripe webhook: missing or malformed signature header")
	}

	timestampStr, ok := params["t"]
	if !ok || timestampStr == "" {
		return errors.New("stripe webhook: missing timestamp in signature header")
	}

	timestamp, err := strconv.ParseInt(timestampStr, 10, 64)
	if err != nil {
		return fmt.Errorf("stripe webhook: invalid timestamp: %v", err)
	}

	// Check timestamp is within tolerance (5 minutes)
	now := time.Now().Unix()
	if now-timestamp > int64(stripeSignatureTTL.Seconds()) {
		return errors.New("stripe webhook: timestamp is too old")
	}
	if timestamp-now > int64(stripeSignatureTTL.Seconds()) {
		return errors.New("stripe webhook: timestamp is in the future")
	}

	// Build the signed payload: timestamp + "." + payload
	signedPayload := fmt.Sprintf("%d.%s", timestamp, string(payload))

	// Find the first v1 signature
	signature, ok := params["v1"]
	if !ok || signature == "" {
		return errors.New("stripe webhook: missing v1 signature")
	}

	// Compute expected signature
	mac := hmac.New(sha256.New, []byte(webhookSecret))
	mac.Write([]byte(signedPayload))
	expected := hex.EncodeToString(mac.Sum(nil))

	// Constant-time comparison
	if !hmac.Equal([]byte(signature), []byte(expected)) {
		return errors.New("stripe webhook: invalid signature")
	}

	return nil
}

// parseStripeSignatureHeader parses the Stripe-Signature header value.
// Format: t=timestamp,v1=sig1,v1=sig2,...
func parseStripeSignatureHeader(header string) map[string]string {
	params := make(map[string]string)
	for _, pair := range strings.Split(header, ",") {
		pair = strings.TrimSpace(pair)
		if idx := strings.IndexByte(pair, '='); idx > 0 {
			key := strings.TrimSpace(pair[:idx])
			value := strings.TrimSpace(pair[idx+1:])
			if key == "v1" {
				// For v1, keep the last one (or first — Stripe may send multiple)
				if _, exists := params[key]; !exists {
					params[key] = value
				}
			} else {
				params[key] = value
			}
		}
	}
	return params
}

// handleStripeWebhook processes incoming Stripe webhook notifications.
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

	// Verify signature
	sigHeader := r.Header.Get(stripeSignatureHeader)
	if err := verifyStripeWebhookSignature(bodyBytes, sigHeader, s.cfg.StripeWebhookSecret); err != nil {
		log.Printf("[stripe-webhook] signature verification error: %v", err)
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
		log.Printf("[stripe-webhook] ignored event=%s: %v", event.Type, err)
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

// stripeWebhookPaymentFromEvent extracts payment info from a Stripe webhook event.
func stripeWebhookPaymentFromEvent(event stripeWebhookEvent) (stripeWebhookPayment, error) {
	eventType := strings.TrimSpace(event.Type)

	switch {
	case eventType == "payment_intent.succeeded":
		return stripePaymentIntentFromEvent(event, "succeeded")
	case eventType == "payment_intent.payment_failed":
		return stripePaymentIntentFromEvent(event, "failed")
	case eventType == "payment_intent.canceled":
		return stripePaymentIntentFromEvent(event, "canceled")
	case strings.HasPrefix(eventType, "payment_intent."):
		// Accept other payment_intent events but skip them
		return stripeWebhookPayment{}, errors.New("unhandled payment_intent event type: " + eventType)
	default:
		return stripeWebhookPayment{}, errors.New("event type is not a payment_intent event")
	}
}

func stripePaymentIntentFromEvent(event stripeWebhookEvent, expectedStatus string) (stripeWebhookPayment, error) {
	var pi stripePaymentIntent
	if err := json.Unmarshal(event.Data.Object, &pi); err != nil {
		return stripeWebhookPayment{}, fmt.Errorf("failed to parse payment_intent: %v", err)
	}

	piID := strings.TrimSpace(pi.ID)
	if piID == "" {
		return stripeWebhookPayment{}, errors.New("payment_intent id is required")
	}

	// Normalize currency from lowercase (Stripe) to uppercase
	currency := strings.ToUpper(strings.TrimSpace(pi.Currency))
	if currency == "" {
		currency = "USD"
	}

	// Use amount_received for succeeded, amount for other states
	amount := pi.AmountReceived
	if amount == 0 {
		amount = pi.Amount
	}

	return stripeWebhookPayment{
		PaymentIntentID: piID,
		Status:          strings.TrimSpace(pi.Status),
		Currency:        currency,
		AmountCents:     amount,
		Description:     strings.TrimSpace(pi.Description),
		Metadata:        pi.Metadata,
		ChargeID:        strings.TrimSpace(pi.LatestCharge),
	}, nil
}