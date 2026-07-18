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
	ID      string          `json:"id"`
	Type    string          `json:"type"`
	Created int64           `json:"created"`
	Data    stripeEventData `json:"data"`
}

type stripeEventData struct {
	Object stripeEventObject `json:"object"`
}

type stripeEventObject struct {
	ID              string            `json:"id"`
	Object          string            `json:"object"`
	Amount          int64             `json:"amount"`
	AmountReceived  int64             `json:"amount_received"`
	AmountRefunded  int64             `json:"amount_refunded"`
	Currency        string            `json:"currency"`
	Status          string            `json:"status"`
	Description     string            `json:"description"`
	Metadata        map[string]string `json:"metadata,omitempty"`
	LastPaymentError *struct {
		Message string `json:"message"`
	} `json:"last_payment_error,omitempty"`
	Charges stripeCharges `json:"charges,omitempty"`
}

type stripeCharges struct {
	Data []stripeCharge `json:"data"`
}

type stripeCharge struct {
	ID                    string                    `json:"id"`
	PaymentMethod         string                    `json:"payment_method"`
	PaymentMethodDetails  *stripePaymentMethodDetails `json:"payment_method_details,omitempty"`
}

type stripePaymentMethodDetails struct {
	Card *stripeCardDetails `json:"card,omitempty"`
}

type stripeCardDetails struct {
	Brand   string `json:"brand"`
	Last4   string `json:"last4"`
	Network string `json:"network,omitempty"`
}

// stripeWebhookPayment represents a verified Stripe webhook payment.
type stripeWebhookPayment struct {
	PaymentIntentID string
	AmountCents     int64
	Currency        string
	Status          string
	Brand           string
	Last4           string
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

	// Verify Stripe webhook signature using the configured webhook secret.
	signatureHeader := r.Header.Get("Stripe-Signature")
	if signatureHeader == "" {
		writeError(w, http.StatusUnauthorized, "missing Stripe-Signature header")
		return
	}
	if err := s.verifyStripeWebhookSignature(bodyBytes, signatureHeader); err != nil {
		log.Printf("[stripe-webhook] signature verification error: %v", err)
		writeError(w, http.StatusUnauthorized, "invalid Stripe webhook signature")
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

	result, err := s.store.RecordStripeSettlement(event.ID, payment)
	if err != nil {
		log.Printf("[stripe-webhook] settlement error: %v", err)
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if result.Status == "verified" && !result.Duplicate {
		s.broadcastLiveFeedEvent("payment_verified")
	}
	log.Printf("[stripe-webhook] event=%s intent=%s status=%s duplicate=%t",
		event.Type, payment.PaymentIntentID, result.Status, result.Duplicate)
	writeJSON(w, http.StatusOK, result)
}

// verifyStripeWebhookSignature verifies the Stripe webhook signature.
// Stripe signs webhooks with HMAC-SHA256 using the webhook secret.
// The Stripe-Signature header contains t=<timestamp>,v1=<signature>.
// The signed payload is the timestamp concatenated with the body (separated by a dot).
func (s *Server) verifyStripeWebhookSignature(payload []byte, signatureHeader string) error {
	webhookSecret := strings.TrimSpace(s.cfg.StripeWebhookSecret)
	if webhookSecret == "" {
		if s.cfg.Environment != "production" {
			return nil // Skip verification in development when secret is not set.
		}
		return errors.New("STRIPE_WEBHOOK_SECRET is required in production")
	}

	timestamp, signature, err := parseStripeSignatureHeader(signatureHeader)
	if err != nil {
		return err
	}

	// Build the signed payload: timestamp + "." + raw body
	signedPayload := strconv.FormatInt(timestamp, 10) + "." + string(payload)

	mac := hmac.New(sha256.New, []byte(webhookSecret))
	mac.Write([]byte(signedPayload))
	expected := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(signature), []byte(expected)) {
		return errors.New("stripe webhook signature mismatch")
	}

	return nil
}

// parseStripeSignatureHeader parses the Stripe-Signature header value.
// Format: t=timestamp,v1=signature[,v1=signature2,...]
func parseStripeSignatureHeader(header string) (int64, string, error) {
	var timestamp int64
	var primarySignature string
	pairs := strings.Split(header, ",")
	for _, pair := range pairs {
		pair = strings.TrimSpace(pair)
		if strings.HasPrefix(pair, "t=") {
			parsed, err := strconv.ParseInt(strings.TrimPrefix(pair, "t="), 10, 64)
			if err != nil {
				return 0, "", errors.New("invalid stripe signature timestamp")
			}
			timestamp = parsed
		} else if strings.HasPrefix(pair, "v1=") {
			sig := strings.TrimPrefix(pair, "v1=")
			if primarySignature == "" {
				primarySignature = sig
			}
		}
	}
	if timestamp == 0 {
		return 0, "", errors.New("stripe signature missing timestamp")
	}
	if primarySignature == "" {
		return 0, "", errors.New("stripe signature missing v1 signature")
	}
	return timestamp, primarySignature, nil
}

// stripeWebhookPaymentFromEvent extracts payment data from a Stripe webhook event.
func stripeWebhookPaymentFromEvent(event stripeWebhookEvent) (stripeWebhookPayment, error) {
	switch strings.ToLower(strings.TrimSpace(event.Type)) {
	case "payment_intent.succeeded":
		obj := event.Data.Object
		if obj.ID == "" {
			return stripeWebhookPayment{}, errors.New("missing payment intent id")
		}
		if obj.Status != "succeeded" {
			return stripeWebhookPayment{}, fmt.Errorf("payment intent status is %s, not succeeded", obj.Status)
		}
		currency := strings.ToLower(strings.TrimSpace(obj.Currency))
		if currency != "usd" {
			return stripeWebhookPayment{}, fmt.Errorf("stripe currency %s is not USD", obj.Currency)
		}
		if obj.AmountReceived <= 0 {
			return stripeWebhookPayment{}, errors.New("stripe amount received must be positive")
		}
		payment := stripeWebhookPayment{
			PaymentIntentID: obj.ID,
			AmountCents:     obj.AmountReceived,
			Currency:        currency,
			Status:          "succeeded",
		}
		// Extract card brand and last4 from charges if available.
		if len(obj.Charges.Data) > 0 {
			charge := obj.Charges.Data[0]
			if charge.PaymentMethodDetails != nil && charge.PaymentMethodDetails.Card != nil {
				payment.Brand = charge.PaymentMethodDetails.Card.Brand
				payment.Last4 = charge.PaymentMethodDetails.Card.Last4
			}
		}
		return payment, nil

	case "payment_intent.payment_failed":
		obj := event.Data.Object
		if obj.ID == "" {
			return stripeWebhookPayment{}, errors.New("missing payment intent id")
		}
		return stripeWebhookPayment{
			PaymentIntentID: obj.ID,
			Status:          "failed",
		}, nil

	case "payment_intent.refunded":
		obj := event.Data.Object
		if obj.ID == "" {
			return stripeWebhookPayment{}, errors.New("missing payment intent id")
		}
		currency := strings.ToLower(strings.TrimSpace(obj.Currency))
		refundedAmount := obj.AmountRefunded
		if refundedAmount <= 0 {
			refundedAmount = obj.AmountReceived
		}
		return stripeWebhookPayment{
			PaymentIntentID: obj.ID,
			AmountCents:     refundedAmount,
			Currency:        currency,
			Status:          "refunded",
		}, nil

	default:
		return stripeWebhookPayment{}, errors.New("event type is not a supported Stripe payment event")
	}
}

// stripeSettlementResult is the result of recording a Stripe webhook payment.
type stripeSettlementResult struct {
	Status    string `json:"status"`
	Duplicate bool   `json:"duplicate"`
}

// RecordStripeSettlement records a Stripe payment intent settlement event.
// It deduplicates by event ID, maps succeeded/failed/refunded into ledger
// and project status, and mints MRG credits for successful payments.
func (s *Store) RecordStripeSettlement(eventID string, payment stripeWebhookPayment) (*stripeSettlementResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if eventID == "" {
		return nil, errors.New("stripe event id is required")
	}
	if payment.PaymentIntentID == "" {
		return nil, errors.New("stripe payment intent id is required")
	}

	// Deduplicate by event ID.
	if s.paymentSettlements == nil {
		s.paymentSettlements = map[string]*stripeSettlementResult{}
	}
	if existing, ok := s.paymentSettlements[eventID]; ok {
		return &stripeSettlementResult{Status: existing.Status, Duplicate: true}, nil
	}

	switch payment.Status {
	case "succeeded":
		// Find the project associated with this PaymentIntent.
		target := s.stripeProjectLocked(payment.PaymentIntentID)
		if target == nil {
			return nil, fmt.Errorf("no project found for stripe payment intent %s", payment.PaymentIntentID)
		}

		// Mint MRG credit + write ledger proof.
		tokenSymbol := normalizedTokenSymbol(s.cfg.TokenSymbol)
		clientProjectAccount := "client:" + target.ClientUserID + ":project:" + target.ID
		s.addLedger("stripe_payment_verified", "payment:stripe:"+payment.PaymentIntentID, clientProjectAccount, payment.AmountCents, payment.PaymentIntentID)
		s.addLedger("token_mint", "issuer:mergeos", clientProjectAccount, payment.AmountCents, "mint:"+target.ID)

		// Update project status and record card brand info.
		target.PaymentStatus = "verified"
		if payment.Brand != "" {
			target.PaymentProvider = "stripe:" + payment.Brand
		}
		if payment.Last4 != "" {
			if existingNote := target.Phone; existingNote == "" {
				target.Phone = "card:****" + payment.Last4
			}
		}

		result := &stripeSettlementResult{Status: "verified"}
		s.paymentSettlements[eventID] = result
		if err := s.saveLocked(); err != nil {
			delete(s.paymentSettlements, eventID)
			return nil, fmt.Errorf("%w: %v", errPaymentOrderIntentPersistence, err)
		}
		return result, nil

	case "failed":
		result := &stripeSettlementResult{Status: "failed"}
		s.paymentSettlements[eventID] = result
		if err := s.saveLocked(); err != nil {
			delete(s.paymentSettlements, eventID)
			return nil, fmt.Errorf("%w: %v", errPaymentOrderIntentPersistence, err)
		}
		s.updateStripeProjectStatusLocked(payment.PaymentIntentID, "failed")
		return result, nil

	case "refunded":
		result := &stripeSettlementResult{Status: "refunded"}
		s.paymentSettlements[eventID] = result
		if err := s.saveLocked(); err != nil {
			delete(s.paymentSettlements, eventID)
			return nil, fmt.Errorf("%w: %v", errPaymentOrderIntentPersistence, err)
		}
		s.updateStripeProjectStatusLocked(payment.PaymentIntentID, "refunded")
		return result, nil

	default:
		return nil, fmt.Errorf("unknown stripe payment status %q", payment.Status)
	}
}

// stripeProjectLocked finds a project by Stripe PaymentIntent reference.
func (s *Store) stripeProjectLocked(paymentIntentID string) *Project {
	for _, project := range s.projects {
		if project == nil {
			continue
		}
		if project.PaymentReference == paymentIntentID &&
			(project.PaymentProvider == "stripe" || project.PaymentProvider == "dev-stripe" || strings.HasPrefix(project.PaymentProvider, "stripe:")) {
			return project
		}
	}
	return nil
}

// updateStripeProjectStatusLocked updates the payment status on a project by PaymentIntent ID.
func (s *Store) updateStripeProjectStatusLocked(paymentIntentID, status string) {
	for _, project := range s.projects {
		if project == nil {
			continue
		}
		if project.PaymentReference == paymentIntentID {
			project.PaymentStatus = status
			return
		}
	}
}
