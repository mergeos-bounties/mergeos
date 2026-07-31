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

// stripeSignatureTolerance is the maximum age (in either direction) we accept
// for a Stripe webhook signature timestamp. Stripe defaults to a 5 minute
// tolerance to prevent replay attacks.
const stripeSignatureTolerance = 5 * time.Minute

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
	PaymentIntent   string            `json:"payment_intent,omitempty"`
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
	RefundID        string
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

	timestamp, signatures, err := parseStripeSignatureHeader(signatureHeader)
	if err != nil {
		return err
	}

	// Reject stale or future timestamps to prevent replay attacks. Stripe's
	// default tolerance is 5 minutes; accept anything within that window.
	now := time.Now().Unix()
	tolerance := int64(stripeSignatureTolerance / time.Second)
	if now-timestamp > tolerance {
		return errors.New("stripe webhook signature timestamp is too old")
	}
	if timestamp-now > tolerance {
		return errors.New("stripe webhook signature timestamp is in the future")
	}

	// Build the signed payload: timestamp + "." + raw body. Accept any valid
	// v1 signature in the header so webhook-secret key rotation works.
	signedPayload := strconv.FormatInt(timestamp, 10) + "." + string(payload)
	for _, signature := range signatures {
		mac := hmac.New(sha256.New, []byte(webhookSecret))
		mac.Write([]byte(signedPayload))
		expected := hex.EncodeToString(mac.Sum(nil))
		if hmac.Equal([]byte(signature), []byte(expected)) {
			return nil
		}
	}

	return errors.New("stripe webhook signature mismatch")
}

// parseStripeSignatureHeader parses the Stripe-Signature header value.
// Format: t=timestamp,v1=signature[,v1=signature2,...]. Returns the shared
// timestamp and every v1 signature found (Stripe may send multiple signatures
// during webhook-secret rotation).
func parseStripeSignatureHeader(header string) (int64, []string, error) {
	var timestamp int64
	signatures := []string{}
	seen := map[string]bool{}
	pairs := strings.Split(header, ",")
	for _, pair := range pairs {
		pair = strings.TrimSpace(pair)
		switch {
		case strings.HasPrefix(pair, "t="):
			parsed, err := strconv.ParseInt(strings.TrimPrefix(pair, "t="), 10, 64)
			if err != nil {
				return 0, nil, errors.New("invalid stripe signature timestamp")
			}
			timestamp = parsed
		case strings.HasPrefix(pair, "v1="):
			sig := strings.TrimSpace(strings.TrimPrefix(pair, "v1="))
			if sig != "" && !seen[sig] {
				seen[sig] = true
				signatures = append(signatures, sig)
			}
		}
	}
	if timestamp == 0 {
		return 0, nil, errors.New("stripe signature missing timestamp")
	}
	if len(signatures) == 0 {
		return 0, nil, errors.New("stripe signature missing v1 signature")
	}
	return timestamp, signatures, nil
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

	case "charge.refunded", "refund.created", "payment_intent.refunded":
		return stripeRefundPaymentFromEvent(event.Type, event.Data.Object, event.Data.Object.ID)

	default:
		return stripeWebhookPayment{}, errors.New("event type is not a supported Stripe payment event")
	}
}

// stripeRefundPaymentFromEvent extracts refund payment data from a Stripe
// webhook event. Stripe emits refunds as "charge.refunded" (object is a Charge)
// or "refund.created" (object is a Refund); the related PaymentIntent id lives
// on the "payment_intent" field. "payment_intent.refunded" is kept as a legacy
// alias that identifies the intent by the object id directly. The refund amount
// source and fallback precedence depend on the event type.
func stripeRefundPaymentFromEvent(eventType string, obj stripeEventObject, fallbackIntent string) (stripeWebhookPayment, error) {
	intentID := strings.TrimSpace(obj.PaymentIntent)
	if intentID == "" {
		intentID = strings.TrimSpace(fallbackIntent)
	}
	if intentID == "" {
		return stripeWebhookPayment{}, errors.New("missing stripe payment intent id")
	}
	currency := strings.ToLower(strings.TrimSpace(obj.Currency))
	if currency != "usd" {
		return stripeWebhookPayment{}, fmt.Errorf("stripe currency %s is not USD", obj.Currency)
	}

	refundedAmount := int64(0)
	switch strings.ToLower(strings.TrimSpace(eventType)) {
	case "refund.created":
		// Object is a Refund: amount is the refunded amount.
		refundedAmount = firstPositive(obj.Amount, obj.AmountRefunded, obj.AmountReceived)
	case "charge.refunded":
		// Object is a Charge: amount_refunded is the aggregate refunded amount.
		refundedAmount = firstPositive(obj.AmountRefunded, obj.Amount, obj.AmountReceived)
	default:
		// Legacy payment_intent.refunded: object is a PaymentIntent.
		refundedAmount = firstPositive(obj.AmountRefunded, obj.AmountReceived, obj.Amount)
	}
	if refundedAmount <= 0 {
		return stripeWebhookPayment{}, errors.New("stripe refund amount must be positive")
	}
	return stripeWebhookPayment{
		PaymentIntentID: intentID,
		RefundID:        strings.TrimSpace(obj.ID),
		AmountCents:     refundedAmount,
		Currency:        currency,
		Status:          "refunded",
	}, nil
}

// firstPositive returns the first positive value among the candidates.
func firstPositive(values ...int64) int64 {
	for _, v := range values {
		if v > 0 {
			return v
		}
	}
	return 0
}

// stripeSettlementResult is the result of recording a Stripe webhook payment.
type stripeSettlementResult struct {
	EventID   string `json:"event_id,omitempty"`
	Status    string `json:"status"`
	Duplicate bool   `json:"duplicate"`
}

// RecordStripeSettlement records a Stripe payment intent settlement event.
// It deduplicates by event ID, maps succeeded/failed/refunded into ledger
// and project status, and mints MRG credits for successful payments exactly
// once per PaymentIntent (the synchronous CreateProject verifier and this
// webhook share the same settlement, never double-minting).
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
		return &stripeSettlementResult{EventID: eventID, Status: existing.Status, Duplicate: true}, nil
	}

	switch payment.Status {
	case "succeeded":
		return s.recordStripeSettlementSucceededLocked(eventID, payment)
	case "failed":
		return s.recordStripeSettlementFailedLocked(eventID, payment)
	case "refunded":
		return s.recordStripeSettlementRefundedLocked(eventID, payment)
	default:
		return nil, fmt.Errorf("unknown stripe payment status %q", payment.Status)
	}
}

func (s *Store) recordStripeSettlementSucceededLocked(eventID string, payment stripeWebhookPayment) (*stripeSettlementResult, error) {
	// Find the project associated with this PaymentIntent.
	target := s.stripeProjectLocked(payment.PaymentIntentID)
	if target == nil {
		return nil, fmt.Errorf("no project found for stripe payment intent %s", payment.PaymentIntentID)
	}
	// Problem 2: enforce the settled-amount/currency/ownership invariants
	// BEFORE any ledger change, matching the synchronous verifier.
	if err := validateStripeSettlementLocked(target, payment); err != nil {
		return nil, err
	}

	clientProjectAccount := "client:" + target.ClientUserID + ":project:" + target.ID
	original := captureStripeProjectStateLocked(target)
	ledgerStart := len(s.ledger)

	// Problem 1: idempotent settlement. If this intent was already credited by
	// CreateProject's synchronous verifier (payment_verified) or a prior
	// webhook (stripe_payment_verified), do NOT mint a second time.
	if s.stripeSettlementAlreadyRecordedLocked(payment.PaymentIntentID) {
		applyStripeCardInfoLocked(target, payment)
		target.PaymentStatus = "verified"
		result := &stripeSettlementResult{EventID: eventID, Status: "verified", Duplicate: true}
		if err := s.saveLocked(); err != nil {
			s.restoreStripeProjectStateLocked(target, original)
			return nil, fmt.Errorf("%w: %v", errPaymentOrderIntentPersistence, err)
		}
		return result, nil
	}

	// Mint MRG credit + write ledger proof.
	s.addLedger("stripe_payment_verified", "payment:stripe:"+payment.PaymentIntentID, clientProjectAccount, payment.AmountCents, payment.PaymentIntentID)
	s.addLedger("token_mint", "issuer:mergeos", clientProjectAccount, payment.AmountCents, "mint:"+target.ID)

	// Update project status and record card brand info.
	applyStripeCardInfoLocked(target, payment)
	target.PaymentStatus = "verified"

	result := &stripeSettlementResult{EventID: eventID, Status: "verified"}
	s.paymentSettlements[eventID] = result
	if err := s.saveLocked(); err != nil {
		delete(s.paymentSettlements, eventID)
		s.ledger = s.ledger[:ledgerStart]
		s.restoreStripeProjectStateLocked(target, original)
		return nil, fmt.Errorf("%w: %v", errPaymentOrderIntentPersistence, err)
	}
	return result, nil
}

func (s *Store) recordStripeSettlementFailedLocked(eventID string, payment stripeWebhookPayment) (*stripeSettlementResult, error) {
	result := &stripeSettlementResult{EventID: eventID, Status: "failed"}
	if target := s.stripeProjectLocked(payment.PaymentIntentID); target != nil {
		original := captureStripeProjectStateLocked(target)
		target.PaymentStatus = "failed"
		s.paymentSettlements[eventID] = result
		if err := s.saveLocked(); err != nil {
			delete(s.paymentSettlements, eventID)
			s.restoreStripeProjectStateLocked(target, original)
			return nil, fmt.Errorf("%w: %v", errPaymentOrderIntentPersistence, err)
		}
		return result, nil
	}
	s.paymentSettlements[eventID] = result
	if err := s.saveLocked(); err != nil {
		delete(s.paymentSettlements, eventID)
		return nil, fmt.Errorf("%w: %v", errPaymentOrderIntentPersistence, err)
	}
	return result, nil
}

// recordStripeSettlementRefundedLocked implements an auditable, idempotent
// reversal for full and partial Stripe refunds. It burns the previously minted
// MRG back to the issuer and releases the held project reserve, recording a
// token_burn ledger entry that doubles as the reversal audit trail and the
// idempotency key (reversals are summed per PaymentIntent).
func (s *Store) recordStripeSettlementRefundedLocked(eventID string, payment stripeWebhookPayment) (*stripeSettlementResult, error) {
	target := s.stripeProjectLocked(payment.PaymentIntentID)
	if target == nil {
		return nil, fmt.Errorf("no project found for stripe payment intent %s", payment.PaymentIntentID)
	}
	if payment.Currency != "" && !strings.EqualFold(strings.TrimSpace(payment.Currency), "usd") {
		return nil, fmt.Errorf("stripe currency %q is not USD", payment.Currency)
	}
	settledCents := target.BudgetCents
	if settledCents <= 0 {
		return nil, fmt.Errorf("cannot reverse stripe payment for project %s with zero budget", target.ID)
	}
	refundCents := payment.AmountCents
	if refundCents <= 0 {
		return nil, errors.New("stripe refund amount must be positive")
	}
	if refundCents > settledCents {
		return nil, fmt.Errorf("stripe refund %d exceeds settled %d cents", refundCents, settledCents)
	}

	// Idempotency: only reverse the delta between what was already reversed for
	// this intent and the refunded amount. A replay (same or lower refund) is a
	// no-op, so duplicate events never double-reverse.
	alreadyReversed := s.stripeReversedCentsLocked(payment.PaymentIntentID)
	if refundCents <= alreadyReversed {
		result := &stripeSettlementResult{EventID: eventID, Status: "refunded", Duplicate: true}
		s.paymentSettlements[eventID] = result
		if err := s.saveLocked(); err != nil {
			delete(s.paymentSettlements, eventID)
			return nil, fmt.Errorf("%w: %v", errPaymentOrderIntentPersistence, err)
		}
		return result, nil
	}
	increment := refundCents - alreadyReversed

	clientProjectAccount := "client:" + target.ClientUserID + ":project:" + target.ID
	original := captureStripeProjectStateLocked(target)
	ledgerStart := len(s.ledger)
	ref := tokenWorkflowReference([]string{"intent:" + payment.PaymentIntentID, "refund:" + payment.RefundID, "event:" + eventID})

	// Burn the previously minted MRG back to the issuer.
	s.addLedger("token_burn", clientProjectAccount, "issuer:mergeos", increment, ref)
	// Release the held project reserve back to the client.
	s.addLedger("project_reserve_release", "reserve:project:"+target.ID, clientProjectAccount, increment, ref)

	if refundCents >= settledCents {
		target.PaymentStatus = "refunded"
	} else {
		target.PaymentStatus = "partially_refunded"
	}
	result := &stripeSettlementResult{EventID: eventID, Status: "refunded"}
	s.paymentSettlements[eventID] = result
	if err := s.saveLocked(); err != nil {
		delete(s.paymentSettlements, eventID)
		s.ledger = s.ledger[:ledgerStart]
		s.restoreStripeProjectStateLocked(target, original)
		return nil, fmt.Errorf("%w: %v", errPaymentOrderIntentPersistence, err)
	}
	return result, nil
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

// validateStripeSettlementLocked rejects a settlement whose currency, amount, or
// PaymentIntent ownership does not match the target project, before any ledger
// entry or status mutation is applied.
func validateStripeSettlementLocked(target *Project, payment stripeWebhookPayment) error {
	if !strings.EqualFold(strings.TrimSpace(payment.Currency), "usd") {
		return fmt.Errorf("stripe currency %q is not USD", payment.Currency)
	}
	if target.PaymentReference != "" && target.PaymentReference != payment.PaymentIntentID {
		return fmt.Errorf("stripe payment intent %s does not own project reference %s", payment.PaymentIntentID, target.PaymentReference)
	}
	if target.BudgetCents > 0 && payment.AmountCents != target.BudgetCents {
		return fmt.Errorf("stripe amount mismatch: got %d cents, expected %d cents", payment.AmountCents, target.BudgetCents)
	}
	return nil
}

// stripeSettlementAlreadyRecordedLocked reports whether a PaymentIntent has
// already been credited to the ledger, either by CreateProject's synchronous
// verifier (payment_verified from payment:stripe) or by this webhook
// (stripe_payment_verified). It guards against a double mint.
func (s *Store) stripeSettlementAlreadyRecordedLocked(intentID string) bool {
	for _, entry := range s.ledger {
		if entry.Type != "payment_verified" && entry.Type != "stripe_payment_verified" {
			continue
		}
		if !strings.HasPrefix(entry.FromAccount, "payment:stripe") {
			continue
		}
		if ledgerValueReferencesID(entry.Reference, intentID) {
			return true
		}
	}
	return false
}

// stripeReversedCentsLocked sums the amount already reversed (burned) for a
// PaymentIntent. It is the idempotency basis for refund reversals.
func (s *Store) stripeReversedCentsLocked(intentID string) int64 {
	var total int64
	for _, entry := range s.ledger {
		if entry.Type != "token_burn" {
			continue
		}
		if ledgerValueReferencesID(entry.Reference, intentID) {
			total += entry.AmountCents
		}
	}
	return total
}

func applyStripeCardInfoLocked(target *Project, payment stripeWebhookPayment) {
	if payment.Brand != "" {
		target.PaymentProvider = "stripe:" + payment.Brand
	}
	if payment.Last4 != "" && target.Phone == "" {
		target.Phone = "card:****" + payment.Last4
	}
}

// stripeProjectState captures the mutable project fields that a settlement
// handler may change, so they can be restored if persistence fails.
type stripeProjectState struct {
	PaymentStatus   string
	PaymentProvider string
	Phone           string
}

func captureStripeProjectStateLocked(project *Project) stripeProjectState {
	return stripeProjectState{
		PaymentStatus:   project.PaymentStatus,
		PaymentProvider: project.PaymentProvider,
		Phone:           project.Phone,
	}
}

func (s *Store) restoreStripeProjectStateLocked(project *Project, state stripeProjectState) {
	project.PaymentStatus = state.PaymentStatus
	project.PaymentProvider = state.PaymentProvider
	project.Phone = state.Phone
}
