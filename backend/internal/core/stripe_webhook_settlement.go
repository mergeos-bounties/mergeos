package core

import (
	"errors"
	"fmt"
	"strings"
)

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

func (s *Store) RecordStripeWebhookPayment(eventID string, payment stripeWebhookPayment) (stripeWebhookSettlement, error) {
	eventID = strings.TrimSpace(eventID)
	if eventID == "" {
		return stripeWebhookSettlement{}, errors.New("stripe webhook event id is required")
	}
	payment, err := validateStripeWebhookPayment(payment)
	if err != nil {
		return stripeWebhookSettlement{}, err
	}

	settlement := stripeWebhookSettlement{
		Status:          "unmatched",
		EventID:         eventID,
		PaymentIntentID: payment.PaymentIntentID,
		AmountCents:     payment.AmountCents,
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if existing, projectID := s.stripeWebhookDuplicateLedgerLocked(eventID, payment); existing != nil {
		settlement.Status = "duplicate"
		settlement.ProjectID = projectID
		settlement.LedgerEntry = existing
		settlement.Duplicate = true
		settlement.Message = "stripe payment was already recorded"
		return settlement, nil
	}

	project := s.stripeWebhookProjectLocked(payment)
	if project == nil {
		settlement.Message = "no project matched the stripe payment intent"
		return settlement, nil
	}

	settlement.ProjectID = project.ID

	if existing, _ := s.stripeWebhookDuplicateLedgerForProjectLocked(project, eventID, payment); existing != nil {
		project.PaymentStatus = s.stripeWebhookStatusMapping(payment.EventType)
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

	project.PaymentStatus = s.stripeWebhookStatusMapping(payment.EventType)
	project.PaymentProvider = "stripe"
	project.PaymentReference = payment.PaymentIntentID

	switch payment.EventType {
	case "payment_intent.succeeded":
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

	case "payment_intent.payment_failed":
		s.addNotificationLocked(
			project.ClientUserID,
			project.ID,
			"payment_failed",
			"Stripe payment failed",
			fmt.Sprintf("Stripe payment of %s %s for %s failed.", formatTokenAmount(payment.AmountCents), normalizedTokenSymbol(s.cfg.TokenSymbol), project.Title),
			"logged:stripe-payment-failed",
		)
		if err := s.saveLocked(); err != nil {
			return stripeWebhookSettlement{}, err
		}
		settlement.Status = "failed"

	case "payment_intent.refunded", "payment_intent.canceled":
		s.addNotificationLocked(
			project.ClientUserID,
			project.ID,
			"payment_refunded",
			"Stripe payment refunded",
			fmt.Sprintf("Stripe payment of %s %s for %s was refunded.", formatTokenAmount(payment.AmountCents), normalizedTokenSymbol(s.cfg.TokenSymbol), project.Title),
			"logged:stripe-payment-refunded",
		)
		if err := s.saveLocked(); err != nil {
			return stripeWebhookSettlement{}, err
		}
		settlement.Status = "refunded"
	}

	return settlement, nil
}

func (s *Store) stripeWebhookProjectLocked(payment stripeWebhookPayment) *Project {
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
		if strings.EqualFold(reference, payment.PaymentIntentID) || ledgerValueReferencesID(reference, payment.PaymentIntentID) {
			return project
		}
	}
	return nil
}

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

func (s *Store) stripeWebhookStatusMapping(eventType string) string {
	switch eventType {
	case "payment_intent.succeeded":
		return "verified"
	case "payment_intent.payment_failed":
		return "failed"
	case "payment_intent.refunded", "payment_intent.canceled":
		return "refunded"
	default:
		return "unmatched"
	}
}

func stripeWebhookLedgerEntryMatches(entry LedgerEntry, eventID string, payment stripeWebhookPayment) bool {
	if entry.Type != "payment_verified" || !strings.EqualFold(entry.FromAccount, "payment:stripe") {
		return false
	}
	for _, id := range []string{strings.TrimSpace(eventID), payment.PaymentIntentID} {
		if id != "" && ledgerValueReferencesID(entry.Reference, id) {
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
