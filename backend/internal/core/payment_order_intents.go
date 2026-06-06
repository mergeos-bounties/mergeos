package core

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

var errPaymentOrderIntentPersistence = errors.New("payment order intent persistence failed")

func normalizePaymentOrderFlow(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "project", "project_funding":
		return PaymentOrderFlowProjectFunding
	case "repo_task", "repo-task", "repository_task", "repository-task", "repo_task_funding":
		return PaymentOrderFlowRepositoryTaskFunding
	default:
		return strings.ToLower(strings.TrimSpace(value))
	}
}

func validatePaymentOrderFlow(value string) (string, error) {
	flow := normalizePaymentOrderFlow(value)
	switch flow {
	case PaymentOrderFlowProjectFunding, PaymentOrderFlowRepositoryTaskFunding:
		return flow, nil
	default:
		return "", fmt.Errorf("unsupported PayPal funding flow %q", strings.TrimSpace(value))
	}
}

func (s *Store) RecordPayPalOrderIntent(userID string, req CreatePayPalOrderRequest, order *CreatePayPalOrderResponse) (*PaymentOrderIntent, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, errors.New("login is required to create a PayPal order")
	}
	if order == nil {
		return nil, errors.New("paypal order response is required")
	}
	orderID := strings.TrimSpace(order.OrderID)
	if orderID == "" {
		return nil, errors.New("paypal order id is required")
	}
	flow, err := validatePaymentOrderFlow(req.Flow)
	if err != nil {
		return nil, err
	}
	if req.AmountCents < 10000 {
		return nil, errors.New("amount must be at least 100 USD")
	}
	now := time.Now().UTC()
	intent := &PaymentOrderIntent{
		OrderID:         orderID,
		Provider:        "paypal",
		Flow:            flow,
		UserID:          userID,
		ProjectID:       strings.TrimSpace(req.ProjectID),
		SuggestedTaskID: strings.TrimSpace(req.SuggestedTaskID),
		AmountCents:     req.AmountCents,
		Currency:        "USD",
		Description:     strings.TrimSpace(req.Description),
		Status:          normalizePaymentOrderStatus(order.Status),
		ApprovalURL:     strings.TrimSpace(order.ApprovalURL),
		ReturnURL:       strings.TrimSpace(req.ReturnURL),
		CancelURL:       strings.TrimSpace(req.CancelURL),
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.paymentOrders == nil {
		s.paymentOrders = map[string]*PaymentOrderIntent{}
	}
	if _, exists := s.paymentOrders[orderID]; exists {
		return nil, fmt.Errorf("paypal order intent %s is already recorded", orderID)
	}
	s.paymentOrders[orderID] = intent
	if err := s.saveLocked(); err != nil {
		delete(s.paymentOrders, orderID)
		return nil, fmt.Errorf("%w: %v", errPaymentOrderIntentPersistence, err)
	}
	return clonePaymentOrderIntent(intent), nil
}

func isPaymentOrderIntentPersistenceError(err error) bool {
	return errors.Is(err, errPaymentOrderIntentPersistence)
}

func (s *Store) PayPalOrderIntent(orderID string) (*PaymentOrderIntent, bool) {
	orderID = strings.TrimSpace(orderID)
	if orderID == "" {
		return nil, false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	intent, ok := s.paymentOrders[orderID]
	if !ok || intent == nil {
		return nil, false
	}
	return clonePaymentOrderIntent(intent), true
}

func (s *Store) ValidatePendingPayPalOrderIntent(userID, orderID, flow, projectID, suggestedTaskID string, amountCents int64) error {
	orderID = strings.TrimSpace(orderID)
	if !s.requiresPayPalOrderIntent(orderID) {
		return nil
	}
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return errors.New("login is required to validate PayPal order intent")
	}
	flow, err := validatePaymentOrderFlow(flow)
	if err != nil {
		return err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	intent, ok := s.paymentOrders[orderID]
	if !ok || intent == nil {
		return errors.New("paypal order intent was not created by MergeOS")
	}
	if !strings.EqualFold(intent.Provider, "paypal") {
		return errors.New("payment order intent is not a PayPal order")
	}
	if intent.UserID != userID {
		return errors.New("paypal order intent belongs to a different user")
	}
	if intent.Flow != flow {
		return fmt.Errorf("paypal order intent is for %s, not %s", intent.Flow, flow)
	}
	if intent.AmountCents != amountCents {
		return fmt.Errorf("paypal order intent amount mismatch: got %s, expected %s", centsToPayPalValue(intent.AmountCents), centsToPayPalValue(amountCents))
	}
	if projectID = strings.TrimSpace(projectID); projectID != "" && intent.ProjectID != "" && intent.ProjectID != projectID {
		return errors.New("paypal order intent is attached to a different project")
	}
	if suggestedTaskID = strings.TrimSpace(suggestedTaskID); suggestedTaskID != "" && intent.SuggestedTaskID != "" && intent.SuggestedTaskID != suggestedTaskID {
		return errors.New("paypal order intent is attached to a different suggested task")
	}
	return nil
}

func (s *Store) requiresPayPalOrderIntent(orderID string) bool {
	orderID = strings.TrimSpace(orderID)
	if orderID == "" {
		return false
	}
	if !s.cfg.PayPalReady() {
		return false
	}
	return !strings.EqualFold(orderID, strings.TrimSpace(s.cfg.DevPaymentCode))
}

func (s *Store) attachPayPalOrderIntentLocked(userID, orderID, flow, projectID, suggestedTaskID string, amountCents int64) error {
	orderID = strings.TrimSpace(orderID)
	if !s.requiresPayPalOrderIntent(orderID) {
		return nil
	}
	intent, ok := s.paymentOrders[orderID]
	if !ok || intent == nil {
		return errors.New("paypal order intent was not created by MergeOS")
	}
	flow, err := validatePaymentOrderFlow(flow)
	if err != nil {
		return err
	}
	userID = strings.TrimSpace(userID)
	if userID != "" && intent.UserID != userID {
		return errors.New("paypal order intent belongs to a different user")
	}
	if intent.Flow != flow {
		return fmt.Errorf("paypal order intent is for %s, not %s", intent.Flow, flow)
	}
	if intent.AmountCents != amountCents {
		return fmt.Errorf("paypal order intent amount mismatch: got %s, expected %s", centsToPayPalValue(intent.AmountCents), centsToPayPalValue(amountCents))
	}
	if projectID = strings.TrimSpace(projectID); projectID != "" {
		if intent.ProjectID != "" && intent.ProjectID != projectID {
			return errors.New("paypal order intent is attached to a different project")
		}
		intent.ProjectID = projectID
	}
	if suggestedTaskID = strings.TrimSpace(suggestedTaskID); suggestedTaskID != "" {
		if intent.SuggestedTaskID != "" && intent.SuggestedTaskID != suggestedTaskID {
			return errors.New("paypal order intent is attached to a different suggested task")
		}
		intent.SuggestedTaskID = suggestedTaskID
	}
	if strings.TrimSpace(intent.Status) == "" || strings.EqualFold(intent.Status, "created") {
		intent.Status = "verified"
	}
	intent.UpdatedAt = time.Now().UTC()
	return nil
}

func (s *Store) recordPayPalOrderIntentSettlementLocked(eventID string, payment paypalWebhookPayment) (*PaymentOrderIntent, bool, error) {
	intent := s.payPalOrderIntentForPaymentLocked(payment)
	if intent == nil {
		return nil, false, nil
	}
	if intent.AmountCents > 0 && intent.AmountCents != payment.AmountCents {
		return nil, false, fmt.Errorf("paypal order intent amount mismatch: got %s, expected %s", centsToPayPalValue(payment.AmountCents), centsToPayPalValue(intent.AmountCents))
	}
	now := time.Now().UTC()
	changed := false
	if intent.Status != "verified" {
		intent.Status = "verified"
		changed = true
	}
	if captureID := strings.TrimSpace(payment.CaptureID); captureID != "" && intent.CaptureID != captureID {
		intent.CaptureID = captureID
		changed = true
	}
	if eventID = strings.TrimSpace(eventID); eventID != "" && intent.WebhookEventID != eventID {
		intent.WebhookEventID = eventID
		changed = true
	}
	if intent.CapturedAt == nil {
		intent.CapturedAt = &now
		changed = true
	}
	if changed {
		intent.UpdatedAt = now
	}
	return intent, changed, nil
}

func (s *Store) payPalOrderIntentForPaymentLocked(payment paypalWebhookPayment) *PaymentOrderIntent {
	if s.paymentOrders == nil {
		return nil
	}
	for _, id := range payPalWebhookPaymentIDs(payment) {
		if intent := s.paymentOrders[strings.TrimSpace(id)]; intent != nil {
			return intent
		}
	}
	return nil
}

func normalizePaymentOrderStatus(value string) string {
	status := strings.ToLower(strings.TrimSpace(value))
	if status == "" {
		return "created"
	}
	return status
}

func clonePaymentOrderIntent(intent *PaymentOrderIntent) *PaymentOrderIntent {
	if intent == nil {
		return nil
	}
	copyIntent := *intent
	copyIntent.CapturedAt = cloneTimePtr(intent.CapturedAt)
	return &copyIntent
}
