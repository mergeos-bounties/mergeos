package core

import (
	"errors"
	"fmt"
	"strings"
)

// stripeWebhookSettlement represents the result of processing a Stripe webhook event.
type stripeWebhookSettlement struct {
	Status          string       `json:"status"`
	EventID         string       `json:"event_id"`
	PaymentIntentID string       `json:"payment_intent_id,omitempty"`
	ProjectID       string       `json:"project_id,omitempty"`
	AmountCents     int64        `json:"amount_cents,omitempty"`
	LedgerEntry     *LedgerEntry `json:"ledger_entry,omitempty"`
	Duplicate       bool         `json:"duplicate"`
	Message         string       `json:"message,omitempty"`
}

// RecordStripeWebhookPayment records a Stripe webhook payment settlement.
// It follows the same pattern as RecordPayPalWebhookPayment.
// Handles: succeeded (verify + ledger), failed/canceled (project status update without ledger).
func (s *Store) RecordStripeWebhookPayment(eventID string, payment stripeWebhookPayment) (stripeWebhookSettlement, error) {
	eventID = strings.TrimSpace(eventID)
	if eventID == "" {
		return stripeWebhookSettlement{}, errors.New("stripe webhook event id is required")
	}

	settlement := stripeWebhookSettlement{
		Status:          "unmatched",
		EventID:         eventID,
		PaymentIntentID: payment.PaymentIntentID,
		AmountCents:     payment.AmountCents,
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Handle non-succeeded events (failed/canceled) — update project status without ledger
	if !strings.EqualFold(payment.Status, "succeeded") {
		return s.recordStripeNonSucceededPaymentLocked(eventID, payment, settlement)
	}

	payment, err := validateStripeWebhookPayment(payment)
	if err != nil {
		return stripeWebhookSettlement{}, err
	}

	// Check for duplicate event or PaymentIntent
	if existing, projectID := s.stripeWebhookDuplicateLedgerLocked(eventID, payment); existing != nil {
		settlement.Status = "duplicate"
		settlement.ProjectID = projectID
		settlement.LedgerEntry = existing
		settlement.Duplicate = true
		settlement.Message = "stripe payment was already recorded"
		return settlement, nil
	}

	// Find project by PaymentReference (PaymentIntent ID)
	project := s.stripeWebhookProjectLocked(payment)
	if project == nil {
		settlement.Message = "no project matched the stripe payment intent"
		return settlement, nil
	}
	settlement.ProjectID = project.ID

	// Check amount match
	if project.BudgetCents > 0 && project.BudgetCents != payment.AmountCents {
		return stripeWebhookSettlement{}, fmt.Errorf(
			"stripe amount mismatch: got %d cents, expected %d cents",
			payment.AmountCents, project.BudgetCents,
		)
	}

	// Check for duplicate project settlement
	if existing, _ := s.stripeWebhookDuplicateLedgerForProjectLocked(project, eventID, payment); existing != nil {
		project.PaymentStatus = "verified"
		project.PaymentProvider = "stripe"
		if strings.TrimSpace(project.PaymentReference) == "" {
			project.PaymentReference = payment.PaymentIntentID
		}
		if err := s.saveLocked(); err != nil {
			return stripeWebhookSettlement{}, err
		}
		settlement.Status = "duplicate"
		settlement.LedgerEntry = existing
		settlement.Duplicate = true
		settlement.Message = "stripe payment was already recorded for this project"
		return settlement, nil
	}

	// Record the payment
	project.PaymentStatus = "verified"
	project.PaymentProvider = "stripe"
	project.PaymentReference = payment.PaymentIntentID

	entry := s.addLedger(
		"payment_verified",
		"payment:stripe",
		"client:"+project.ClientUserID+":project:"+project.ID,
		payment.AmountCents,
		stripeWebhookLedgerReference(eventID, payment),
	)
	s.addNotificationLocked(
		project.ClientUserID,
		project.ID,
		"payment_verified",
		"Stripe payment verified",
		fmt.Sprintf("Stripe confirmed %s %s for %s.", formatTokenAmount(payment.AmountCents), normalizedTokenSymbol(s.cfg.TokenSymbol), project.Title),
		"logged:stripe-payment-verified",
	)
	if err := s.saveLocked(); err != nil {
		return stripeWebhookSettlement{}, err
	}

	settlement.Status = "verified"
	settlement.LedgerEntry = &entry
	return settlement, nil
}

// recordStripeNonSucceededPaymentLocked handles failed/canceled Stripe payment events.
// Updates the project payment status without creating a ledger entry.
func (s *Store) recordStripeNonSucceededPaymentLocked(eventID string, payment stripeWebhookPayment, settlement stripeWebhookSettlement) (stripeWebhookSettlement, error) {
	project := s.stripeWebhookProjectLocked(payment)
	if project == nil {
		settlement.Message = "no project matched the stripe payment intent"
		return settlement, nil
	}
	settlement.ProjectID = project.ID

	// Update project status to reflect the payment failure
	project.PaymentStatus = payment.Status // "failed" or "canceled"
	project.PaymentProvider = "stripe"

	s.addNotificationLocked(
		project.ClientUserID,
		project.ID,
		"payment_"+payment.Status,
		"Stripe payment "+payment.Status,
		fmt.Sprintf("Stripe payment %s for %s (PI: %s).", payment.Status, project.Title, payment.PaymentIntentID),
		"logged:stripe-payment-"+payment.Status,
	)
	if err := s.saveLocked(); err != nil {
		return stripeWebhookSettlement{}, err
	}

	settlement.Status = payment.Status + "_recorded"
	settlement.Message = fmt.Sprintf("stripe payment %s recorded for project", payment.Status)
	return settlement, nil
}

func validateStripeWebhookPayment(payment stripeWebhookPayment) (stripeWebhookPayment, error) {
	if strings.TrimSpace(payment.PaymentIntentID) == "" {
		return stripeWebhookPayment{}, errors.New("stripe payment intent id is required")
	}
	if !strings.EqualFold(payment.Status, "succeeded") {
		return stripeWebhookPayment{}, errors.New("stripe payment is not succeeded")
	}
	if payment.AmountCents <= 0 {
		return stripeWebhookPayment{}, errors.New("stripe amount must be positive")
	}
	return payment, nil
}

// stripeWebhookProjectLocked finds a project whose PaymentReference matches the PaymentIntent.
func (s *Store) stripeWebhookProjectLocked(payment stripeWebhookPayment) *Project {
	piID := strings.TrimSpace(payment.PaymentIntentID)
	if piID == "" {
		return nil
	}
	for _, project := range s.projects {
		if project == nil {
			continue
		}
		if project.PaymentMethod != PaymentStripe && !strings.EqualFold(project.PaymentProvider, "stripe") {
			continue
		}
		reference := strings.TrimSpace(project.PaymentReference)
		if reference == "" {
			continue
		}
		// Match by PaymentIntent ID or ledger reference
		if strings.EqualFold(reference, piID) || ledgerValueReferencesID(reference, piID) {
			return project
		}
	}
	return nil
}

// stripeWebhookDuplicateLedgerLocked checks if this event or PaymentIntent has already been recorded in the ledger.
func (s *Store) stripeWebhookDuplicateLedgerLocked(eventID string, payment stripeWebhookPayment) (*LedgerEntry, string) {
	for i := range s.ledger {
		entry := s.ledger[i]
		if !stripeWebhookLedgerEntryMatches(entry, eventID, payment) {
			continue
		}
		copyEntry := entry
		return &copyEntry, s.projectIDForLedgerEntryLocked(entry)
	}
	return nil, ""
}

// stripeWebhookDuplicateLedgerForProjectLocked checks if this project already has a Stripe payment recorded.
func (s *Store) stripeWebhookDuplicateLedgerForProjectLocked(project *Project, eventID string, payment stripeWebhookPayment) (*LedgerEntry, string) {
	if project == nil {
		return nil, ""
	}
	for i := range s.ledger {
		entry := s.ledger[i]
		if !ledgerEntryReferencesID(entry, project.ID) {
			continue
		}
		if !stripeWebhookLedgerEntryMatches(entry, eventID, payment) {
			continue
		}
		copyEntry := entry
		return &copyEntry, project.ID
	}
	return nil, ""
}

func stripeWebhookLedgerEntryMatches(entry LedgerEntry, eventID string, payment stripeWebhookPayment) bool {
	if entry.Type != "payment_verified" || !strings.EqualFold(entry.FromAccount, "payment:stripe") {
		return false
	}
	refs := []string{strings.TrimSpace(eventID), strings.TrimSpace(payment.PaymentIntentID)}
	for _, ref := range refs {
		if ref != "" && ledgerValueReferencesID(entry.Reference, ref) {
			return true
		}
	}
	return false
}

func stripeWebhookLedgerReference(eventID string, payment stripeWebhookPayment) string {
	return tokenWorkflowReference([]string{
		"stripe_pi:" + payment.PaymentIntentID,
		"event:" + eventID,
	})
}