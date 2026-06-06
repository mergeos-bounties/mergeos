package core

import (
	"errors"
	"fmt"
	"strings"
)

type payPalWebhookSettlement struct {
	Status      string       `json:"status"`
	EventID     string       `json:"event_id"`
	OrderID     string       `json:"order_id,omitempty"`
	CaptureID   string       `json:"capture_id,omitempty"`
	ProjectID   string       `json:"project_id,omitempty"`
	AmountCents int64        `json:"amount_cents,omitempty"`
	LedgerEntry *LedgerEntry `json:"ledger_entry,omitempty"`
	Duplicate   bool         `json:"duplicate"`
	Message     string       `json:"message,omitempty"`
}

func (s *Store) RecordPayPalWebhookPayment(eventID string, payment paypalWebhookPayment) (payPalWebhookSettlement, error) {
	eventID = strings.TrimSpace(eventID)
	if eventID == "" {
		return payPalWebhookSettlement{}, errors.New("paypal webhook event id is required")
	}
	payment, err := validatePayPalWebhookPayment(payment)
	if err != nil {
		return payPalWebhookSettlement{}, err
	}
	settlement := payPalWebhookSettlement{
		Status:      "unmatched",
		EventID:     eventID,
		OrderID:     strings.TrimSpace(payment.OrderID),
		CaptureID:   strings.TrimSpace(payment.CaptureID),
		AmountCents: payment.AmountCents,
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if existing, projectID := s.payPalWebhookDuplicateLedgerLocked(eventID, payment); existing != nil {
		if _, changed, err := s.recordPayPalOrderIntentSettlementLocked(eventID, payment); err != nil {
			return payPalWebhookSettlement{}, err
		} else if changed {
			if err := s.saveLocked(); err != nil {
				return payPalWebhookSettlement{}, err
			}
		}
		settlement.Status = "duplicate"
		settlement.ProjectID = projectID
		settlement.LedgerEntry = existing
		settlement.Duplicate = true
		settlement.Message = "paypal payment was already recorded"
		return settlement, nil
	}

	project := s.payPalWebhookProjectLocked(payment)
	if project == nil {
		intent, changed, err := s.recordPayPalOrderIntentSettlementLocked(eventID, payment)
		if err != nil {
			return payPalWebhookSettlement{}, err
		}
		if intent != nil {
			if changed {
				if err := s.saveLocked(); err != nil {
					return payPalWebhookSettlement{}, err
				}
			}
			settlement.Status = "intent_verified"
			settlement.ProjectID = intent.ProjectID
			settlement.Message = "paypal order intent verified; waiting for project funding attachment"
			return settlement, nil
		}
		settlement.Message = "no project matched the paypal order or capture"
		return settlement, nil
	}
	settlement.ProjectID = project.ID
	if project.BudgetCents > 0 && project.BudgetCents != payment.AmountCents {
		return payPalWebhookSettlement{}, fmt.Errorf("paypal amount mismatch: got %s, expected %s", centsToPayPalValue(payment.AmountCents), centsToPayPalValue(project.BudgetCents))
	}

	if existing, _ := s.payPalWebhookDuplicateLedgerForProjectLocked(project, eventID, payment); existing != nil {
		project.PaymentStatus = "verified"
		project.PaymentProvider = "paypal"
		if strings.TrimSpace(project.PaymentReference) == "" {
			project.PaymentReference = payPalWebhookPreferredReference(payment)
		}
		if _, _, err := s.recordPayPalOrderIntentSettlementLocked(eventID, payment); err != nil {
			return payPalWebhookSettlement{}, err
		}
		if err := s.saveLocked(); err != nil {
			return payPalWebhookSettlement{}, err
		}
		settlement.Status = "duplicate"
		settlement.LedgerEntry = existing
		settlement.Duplicate = true
		settlement.Message = "paypal payment was already recorded for this project"
		return settlement, nil
	}

	project.PaymentStatus = "verified"
	project.PaymentProvider = "paypal"
	project.PaymentReference = payPalWebhookPreferredReference(payment)
	if _, _, err := s.recordPayPalOrderIntentSettlementLocked(eventID, payment); err != nil {
		return payPalWebhookSettlement{}, err
	}
	entry := s.addLedger(
		"payment_verified",
		"payment:paypal",
		"client:"+project.ClientUserID+":project:"+project.ID,
		payment.AmountCents,
		payPalWebhookLedgerReference(eventID, payment),
	)
	s.addNotificationLocked(
		project.ClientUserID,
		project.ID,
		"payment_verified",
		"PayPal payment verified",
		fmt.Sprintf("PayPal confirmed %s %s for %s.", formatTokenAmount(payment.AmountCents), normalizedTokenSymbol(s.cfg.TokenSymbol), project.Title),
		"logged:paypal-payment-verified",
	)
	if err := s.saveLocked(); err != nil {
		return payPalWebhookSettlement{}, err
	}
	settlement.Status = "verified"
	settlement.LedgerEntry = &entry
	return settlement, nil
}

func (s *Store) payPalWebhookProjectLocked(payment paypalWebhookPayment) *Project {
	ids := payPalWebhookPaymentIDs(payment)
	for _, project := range s.projects {
		if project == nil {
			continue
		}
		if project.PaymentMethod != PaymentPayPal && !strings.EqualFold(project.PaymentProvider, "paypal") {
			continue
		}
		reference := strings.TrimSpace(project.PaymentReference)
		if reference == "" {
			continue
		}
		for _, id := range ids {
			if strings.EqualFold(reference, id) || ledgerValueReferencesID(reference, id) {
				return project
			}
		}
	}
	return nil
}

func (s *Store) payPalWebhookDuplicateLedgerLocked(eventID string, payment paypalWebhookPayment) (*LedgerEntry, string) {
	for i := range s.ledger {
		entry := s.ledger[i]
		if !payPalWebhookLedgerEntryMatches(entry, eventID, payment) {
			continue
		}
		copyEntry := entry
		return &copyEntry, s.projectIDForLedgerEntryLocked(entry)
	}
	return nil, ""
}

func (s *Store) payPalWebhookDuplicateLedgerForProjectLocked(project *Project, eventID string, payment paypalWebhookPayment) (*LedgerEntry, string) {
	if project == nil {
		return nil, ""
	}
	for i := range s.ledger {
		entry := s.ledger[i]
		if !ledgerEntryReferencesID(entry, project.ID) {
			continue
		}
		if !payPalWebhookLedgerEntryMatches(entry, eventID, payment) {
			continue
		}
		copyEntry := entry
		return &copyEntry, project.ID
	}
	return nil, ""
}

func payPalWebhookLedgerEntryMatches(entry LedgerEntry, eventID string, payment paypalWebhookPayment) bool {
	if entry.Type != "payment_verified" || !strings.EqualFold(entry.FromAccount, "payment:paypal") {
		return false
	}
	for _, id := range append([]string{strings.TrimSpace(eventID)}, payPalWebhookPaymentIDs(payment)...) {
		if id != "" && ledgerValueReferencesID(entry.Reference, id) {
			return true
		}
	}
	return false
}

func (s *Store) projectIDForLedgerEntryLocked(entry LedgerEntry) string {
	for projectID := range s.projects {
		if ledgerEntryReferencesID(entry, projectID) {
			return projectID
		}
	}
	return ""
}

func payPalWebhookPaymentIDs(payment paypalWebhookPayment) []string {
	seen := map[string]bool{}
	ids := []string{}
	for _, id := range []string{payment.OrderID, payment.CaptureID} {
		id = strings.TrimSpace(id)
		if id == "" || seen[strings.ToLower(id)] {
			continue
		}
		seen[strings.ToLower(id)] = true
		ids = append(ids, id)
	}
	return ids
}

func payPalWebhookPreferredReference(payment paypalWebhookPayment) string {
	if ref := strings.TrimSpace(payment.OrderID); ref != "" {
		return ref
	}
	return strings.TrimSpace(payment.CaptureID)
}

func payPalWebhookLedgerReference(eventID string, payment paypalWebhookPayment) string {
	return tokenWorkflowReference([]string{
		"paypal_order:" + payment.OrderID,
		"capture:" + payment.CaptureID,
		"event:" + eventID,
	})
}
