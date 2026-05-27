package core

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

// paypalWebhookEvent represents a PayPal webhook notification
type paypalWebhookEvent struct {
	ID             string          `json:"id"`
	EventType      string          `json:"event_type"`
	EventVersion   string          `json:"event_version"`
	CreateTime     string          `json:"create_time"`
	ResourceType   string          `json:"resource_type"`
	Resource       json.RawMessage `json:"resource"`
	ReceivedAt     time.Time
}

// paypalResource represents the resource payload inside a webhook event
type paypalResource struct {
	ID            string             `json:"id"`
	State         string             `json:"state"`
	Status        string             `json:"status"`
	Amount        *paypalAmount      `json:"amount"`
	PurchaseUnits []paypalPurchaseUnit `json:"purchase_units"`
	Payer         *paypalPayer       `json:"payer"`
}

type paypalPurchaseUnit struct {
	Payments *struct {
		Captures []struct {
			ID     string         `json:"id"`
			Status string         `json:"status"`
			Amount paypalAmount   `json:"amount"`
		} `json:"captures"`
	} `json:"payments"`
}

type paypalAmount struct {
	CurrencyCode string `json:"currency_code"`
	Value        string `json:"value"`
}

type paypalPayer struct {
	PayerID       string `json:"payer_id"`
	EmailAddress  string `json:"email_address"`
}

// paypalWebhookLog represents a stored webhook event log
type paypalWebhookLog struct {
	ID          int64     `json:"id"`
	EventID     string    `json:"event_id"`
	EventType   string    `json:"event_type"`
	Status      string    `json:"status"`
	OrderID     string    `json:"order_id"`
	Currency    string    `json:"currency"`
	Value       string    `json:"value"`
	RawPayload  string    `json:"raw_payload"`
	ReceivedAt  time.Time `json:"received_at"`
	Processed   bool      `json:"processed"`
	Error       string    `json:"error"`
}

// handlePayPalWebhook processes incoming PayPal webhook notifications
func (s *Server) handlePayPalWebhook(w http.ResponseWriter, r *http.Request) {
	// Read raw body for signature verification
	bodyBytes, err := io.ReadAll(io.LimitReader(r.Body, 256*1024))
	defer r.Body.Close()
	if err != nil {
		log.Printf("[paypal-webhook] failed to read body: %v", err)
		writeError(w, http.StatusBadRequest, "failed to read request body")
		return
	}

	var event paypalWebhookEvent
	if err := json.Unmarshal(bodyBytes, &event); err != nil {
		s.logPayPalWebhook("", "parse_error", "", "", "", string(bodyBytes[:min(len(bodyBytes), 2048)]), false, err.Error())
		writeError(w, http.StatusBadRequest, "invalid JSON payload")
		return
	}

	event.ReceivedAt = time.Now()

	// Verify webhook signature if webhook ID is configured
	if s.cfg.PayPalWebhookID != "" {
		if err := s.verifyPayPalWebhookSig(r, bodyBytes); err != nil {
			s.logPayPalWebhook(event.ID, event.EventType, "", "", "", string(bodyBytes[:min(len(bodyBytes), 2048)]), false, "sig verify failed: "+err.Error())
			writeError(w, http.StatusUnauthorized, "webhook signature verification failed")
			return
		}
	}

	// Extract resource info
	var res paypalResource
	json.Unmarshal(event.Resource, &res)

	orderID := res.ID
	status := res.Status
	currency := ""
	value := ""

	// Extract amount from capture if available
	for _, pu := range res.PurchaseUnits {
		if pu.Payments != nil && len(pu.Payments.Captures) > 0 {
			cap := pu.Payments.Captures[0]
			if cap.Status == "COMPLETED" {
				currency = cap.Amount.CurrencyCode
				value = cap.Amount.Value
			}
		}
	}

	// Log the event
	s.logPayPalWebhook(event.ID, event.EventType, orderID, currency, value, string(bodyBytes[:min(len(bodyBytes), 2048)]), true, "")

	// Process based on event type
	var processErr error
	switch event.EventType {
	case "PAYMENT.CAPTURE.COMPLETED":
		processErr = s.processPaymentCompleted(event, orderID, value, currency)
	case "PAYMENT.CAPTURE.DENIED", "PAYMENT.CAPTURE.DECLINED":
		processErr = s.processPaymentDenied(event, orderID)
	case "PAYMENT.CAPTURE.REFUNDED":
		processErr = s.processPaymentRefunded(event, orderID)
	case "CHECKOUT.ORDER.APPROVED":
		// Order approved - no action needed, capture comes separately
		processErr = nil
	case "CHECKOUT.ORDER.COMPLETED":
		processErr = nil
	default:
		log.Printf("[paypal-webhook] unhandled event type: %s", event.EventType)
	}

	if processErr != nil {
		log.Printf("[paypal-webhook] processing error for event %s: %v", event.ID, processErr)
	}

	// Always return 200 to acknowledge receipt
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "received", "event_id": event.ID})
}

// verifyPayPalWebhookSig verifies webhook signature via PayPal API
func (s *Server) verifyPayPalWebhookSig(r *http.Request, body []byte) error {
	transmissionID := r.Header.Get("PAYPAL-TRANSMISSION-ID")
	transmissionSig := r.Header.Get("PAYPAL-TRANSMISSION-SIG")
	transmissionTime := r.Header.Get("PAYPAL-TRANSMISSION-TIME")
	certURL := r.Header.Get("PAYPAL-CERT-URL")
	authAlgo := r.Header.Get("PAYPAL-AUTH-ALGO")

	if transmissionID == "" || transmissionSig == "" {
		// Sandbox dev mode - accept without full verification
		if s.cfg.Environment == "development" || s.cfg.PayPalEnvironment == "sandbox" {
			return nil
		}
	}

	// Production: verify via PayPal API
	verificationReq := map[string]any{
		"transmission_id":   transmissionID,
		"transmission_sig":  transmissionSig,
		"transmission_time": transmissionTime,
		"cert_url":          certURL,
		"auth_algo":         authAlgo,
		"webhook_id":        s.cfg.PayPalWebhookID,
		"webhook_event":     json.RawMessage(body),
	}

	token, err := s.payments.payPalAccessToken(r.Context())
	if err != nil {
		return fmt.Errorf("failed to get PayPal access token: %w", err)
	}

	payload, _ := json.Marshal(verificationReq)
	httpReq, err := http.NewRequestWithContext(r.Context(), http.MethodPost,
		s.payments.payPalBaseURL()+"/v1/notifications/verify-webhook-signature",
		bytes.NewReader(payload))
	if err != nil {
		return err
	}
	httpReq.Header.Set("Authorization", "Bearer "+token)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := s.payments.client.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var verification struct {
		VerificationStatus string `json:"verification_status"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&verification); err != nil {
		return err
	}

	if verification.VerificationStatus != "SUCCESS" {
		return fmt.Errorf("webhook sig verification failed: %s", verification.VerificationStatus)
	}

	return nil
}

// processPaymentCompleted handles successful PayPal payment capture
func (s *Server) processPaymentCompleted(event paypalWebhookEvent, orderID, value, currency string) error {
	// Find project by payment reference (order ID)
	project, err := s.store.FindProjectByPaymentReference(orderID)
	if err != nil {
		return fmt.Errorf("project lookup failed for order %s: %w", orderID, err)
	}
	if project == nil {
		return fmt.Errorf("no project found for PayPal order %s", orderID)
	}

	// Mark project as paid
	if err := s.store.MarkProjectPaid(project.ID, orderID, "paypal", value, currency); err != nil {
		return fmt.Errorf("failed to mark project %s as paid: %w", project.ID, err)
	}

	// Create notification for project owner
	s.store.CreateNotification(Notification{
		UserID:    project.UserID,
		Subject:   "Payment Received",
		Body:      fmt.Sprintf("Your project '%s' payment of %s %s completed via PayPal (Order: %s)", project.Name, currency, value, orderID),
		Channel:   "payment",
		ProjectID: project.ID,
	})

	log.Printf("[paypal-webhook] payment completed: project %s, order %s, %s %s",
		project.ID, orderID, currency, value)
	return nil
}

// processPaymentDenied handles denied payment
func (s *Server) processPaymentDenied(event paypalWebhookEvent, orderID string) error {
	project, err := s.store.FindProjectByPaymentReference(orderID)
	if err != nil || project == nil {
		return nil // Project may not exist yet
	}

	s.store.CreateNotification(Notification{
		UserID:    project.UserID,
		Subject:   "Payment Failed",
		Body:      fmt.Sprintf("PayPal payment for project '%s' was denied (Order: %s)", project.Name, orderID),
		Channel:   "payment",
		ProjectID: project.ID,
	})

	log.Printf("[paypal-webhook] payment denied: project %s, order %s", project.ID, orderID)
	return nil
}

// processPaymentRefunded handles refunded payment
func (s *Server) processPaymentRefunded(event paypalWebhookEvent, orderID string) error {
	project, err := s.store.FindProjectByPaymentReference(orderID)
	if err != nil || project == nil {
		return nil
	}

	s.store.CreateNotification(Notification{
		UserID:    project.UserID,
		Subject:   "Payment Refunded",
		Body:      fmt.Sprintf("PayPal payment for project '%s' refunded (Order: %s)", project.Name, orderID),
		Channel:   "payment",
		ProjectID: project.ID,
	})

	log.Printf("[paypal-webhook] payment refunded: project %s, order %s", project.ID, orderID)
	return nil
}

// logPayPalWebhook stores webhook event in the database
func (s *Server) logPayPalWebhook(eventID, eventType, orderID, currency, value, rawPayload string, processed bool, errMsg string) error {
	log := paypalWebhookLog{
		EventID:    eventID,
		EventType:  eventType,
		Status:     eventType,
		OrderID:    orderID,
		Currency:   currency,
		Value:      value,
		RawPayload: rawPayload,
		ReceivedAt: time.Now(),
		Processed:  processed,
		Error:      errMsg,
	}
	return s.store.SavePayPalWebhookLog(log)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
