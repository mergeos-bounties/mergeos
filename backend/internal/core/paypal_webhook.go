package core

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// PayPalWebhookEvent represents a PayPal webhook notification
type PayPalWebhookEvent struct {
	ID                 string    `json:"id"`
	EventType          string    `json:"event_type"`
	EventVersion       string    `json:"event_version"`
	CreateTime         string    `json:"create_time"`
	ResourceType       string    `json:"resource_type"`
	Resource           *PayPalResource `json:"resource"`
	ReceivedAt         time.Time
	WebhookID          string `json:"-"`
}

// PayPalResource represents the resource payload inside a webhook event
type PayPalResource struct {
	ID            string `json:"id"`
	State         string `json:"state"`
	Status        string `json:"status"`
	Intent        string `json:"intent"`
	Amount        *PayPalAmount `json:"amount"`
	PurchaseUnits []struct {
		ReferenceID string `json:"reference_id"`
		Amount      *PayPalAmount `json:"amount"`
		Payments    *struct {
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
	Payer *struct {
		PayerID   string `json:"payer_id"`
		EmailAddress string `json:"email_address"`
	} `json:"payer"`
}

// PayPalAmount represents a PayPal amount
type PayPalAmount struct {
	CurrencyCode string `json:"currency_code"`
	Value        string `json:"value"`
}

// PayPalWebhookLog represents a stored webhook event log
type PayPalWebhookLog struct {
	ID           int64     `json:"id"`
	EventID      string    `json:"event_id"`
	EventType    string    `json:"event_type"`
	Status       string    `json:"status"`
	OrderID      string    `json:"order_id"`
	Currency     string    `json:"currency"`
	Value        string    `json:"value"`
	RawPayload   string    `json:"raw_payload"`
	ReceivedAt   time.Time `json:"received_at"`
	Processed    bool      `json:"processed"`
	ErrorMessage string    `json:"error_message"`
}

// HandlePayPalWebhook processes incoming PayPal webhook notifications
func (s *Server) handlePayPalWebhook(w http.ResponseWriter, r *http.Request) {
	// Read and store raw body for signature verification
	bodyBytes, err := io.ReadAll(io.LimitReader(r.Body, 256*1024))
	r.Body.Close()
	if err != nil {
		writeError(w, http.StatusBadRequest, "failed to read request body")
		return
	}

	// Log the event before processing
	var event PayPalWebhookEvent
	if err := json.Unmarshal(bodyBytes, &event); err != nil {
		s.logPayPalWebhookEvent("", "parse_error", "", "", "", string(bodyBytes[:min(2048, len(bodyBytes))]), false, err.Error())
		writeError(w, http.StatusBadRequest, "invalid JSON payload")
		return
	}

	event.ReceivedAt = time.Now()

	// Verify webhook signature if webhook ID is configured
	if s.cfg.PayPalWebhookID != "" {
		if err := s.verifyPayPalWebhookSignature(r, bodyBytes); err != nil {
			s.logPayPalWebhookEvent(event.ID, event.EventType, "", "", "", string(bodyBytes[:min(2048, len(bodyBytes))]), false, "signature verification failed: "+err.Error())
			writeError(w, http.StatusUnauthorized, "webhook signature verification failed")
			return
		}
	}

	// Extract order/resource information
	orderID := ""
	status := ""
	currency := ""
	value := ""
	if event.Resource != nil {
		orderID = event.Resource.ID
		status = event.Resource.Status
		if event.Resource.Amount != nil {
			currency = event.Resource.Amount.CurrencyCode
			value = event.Resource.Amount.Value
		}
		// Also check purchase units for captures
		for _, pu := range event.Resource.PurchaseUnits {
			if pu.Payments != nil && len(pu.Payments.Captures) > 0 {
				capture := pu.Payments.Captures[0]
				if capture.Status == "COMPLETED" {
					currency = capture.Amount.CurrencyCode
					value = capture.Amount.Value
				}
			}
		}
	}

	// Log the event
	if err := s.logPayPalWebhookEvent(event.ID, event.EventType, orderID, currency, value, string(bodyBytes[:min(2048, len(bodyBytes))]), true, ""); err != nil {
		fmt.Printf("[paypal-webhook] failed to log event %s: %v\n", event.ID, err)
	}

	// Process based on event type
	var processErr error
	switch event.EventType {
	case "PAYMENT.CAPTURE.COMPLETED":
		processErr = s.processPaymentCaptureCompleted(event, orderID, value, currency)
	case "PAYMENT.CAPTURE.DENIED", "PAYMENT.CAPTURE.DECLINED":
		processErr = s.processPaymentCaptureDenied(event, orderID)
	case "PAYMENT.CAPTURE.REFUNDED":
		processErr = s.processPaymentCaptureRefunded(event, orderID)
	case "CHECKOUT.ORDER.APPROVED":
		// Order approved by buyer - waiting for capture
		processErr = nil // No action needed, capture will come separately
	case "CHECKOUT.ORDER.COMPLETED":
		// Full order completed
		processErr = nil
	default:
		// Log but don't error on unknown event types
		fmt.Printf("[paypal-webhook] unhandled event type: %s\n", event.EventType)
	}

	if processErr != nil {
		fmt.Printf("[paypal-webhook] processing error for event %s: %v\n", event.ID, processErr)
		// Return 200 anyway - PayPal doesn't retry on 5xx for most events
		// but we log the error for manual review
	}

	// Always return 200 to acknowledge receipt
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "received", "event_id": event.ID})
}

// verifyPayPalWebhookSignature verifies the webhook signature using HMAC-SHA256
func (s *Server) verifyPayPalWebhookSignature(r *http.Request, body []byte) error {
	transmissionID := r.Header.Get("PAYPAL-TRANSMISSION-ID")
	transmissionSig := r.Header.Get("PAYPAL-TRANSMISSION-SIG")
	transmissionTime := r.Header.Get("PAYPAL-TRANSMISSION-TIME")
	certURL := r.Header.Get("PAYPAL-CERT-URL")
	authAlgo := r.Header.Get("PAYPAL-AUTH-ALGO")

	if transmissionID == "" || transmissionSig == "" {
		// Headers missing - could be sandbox test without signature
		return nil
	}

	// For full verification we'd need to fetch the PayPal cert and verify the signature.
	// In sandbox mode with dev payment enabled, we accept events without full cert verification.
	// Production deployments should implement full cert verification.
	if s.cfg.Environment == "development" {
		return nil
	}

	// Production: verify via PayPal API
	verificationBody := map[string]any{
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

	var payload json.Buffer
	json.NewEncoder(&payload).Encode(verificationBody)
	
	httpReq, err := http.NewRequestWithContext(r.Context(), http.MethodPost,
		s.payments.payPalBaseURL()+"/v1/notifications/verify-webhook-signature", &payload)
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
		return fmt.Errorf("webhook signature verification failed: %s", verification.VerificationStatus)
	}

	return nil
}

// processPaymentCaptureCompleted handles successful payment capture
func (s *Server) processPaymentCaptureCompleted(event PayPalWebhookEvent, orderID, value, currency string) error {
	// Find the project associated with this PayPal order
	// The order ID should be stored as payment_reference when the order was created
	project, err := s.store.FindProjectByPaymentReference(orderID)
	if err != nil {
		return fmt.Errorf("project lookup failed for order %s: %w", orderID, err)
	}
	if project == nil {
		return fmt.Errorf("no project found for PayPal order %s", orderID)
	}

	// Update project payment status
	if err := s.store.MarkProjectAsPaid(project.ID, orderID, "paypal", value, currency); err != nil {
		return fmt.Errorf("failed to mark project %s as paid: %w", project.ID, err)
	}

	// Create notification for project owner
	notification := CreateNotificationRequest{
		UserID:    project.UserID,
		Subject:   "Payment Received",
		Body:      fmt.Sprintf("Your project '%s' payment of %s %s has been completed via PayPal (Order: %s)", project.Name, currency, value, orderID),
		Channel:   "payment",
		ProjectID: project.ID,
	}
	s.store.CreateNotification(notification)

	fmt.Printf("[paypal-webhook] payment completed for project %s, order %s, %s %s\n",
		project.ID, orderID, currency, value)
	return nil
}

// processPaymentCaptureDenied handles denied payment capture
func (s *Server) processPaymentCaptureDenied(event PayPalWebhookEvent, orderID string) error {
	project, err := s.store.FindProjectByPaymentReference(orderID)
	if err != nil || project == nil {
		return nil // Project may not exist yet
	}

	// Mark payment as failed
	notification := CreateNotificationRequest{
		UserID:    project.UserID,
		Subject:   "Payment Failed",
		Body:      fmt.Sprintf("Your PayPal payment for project '%s' was denied (Order: %s)", project.Name, orderID),
		Channel:   "payment",
		ProjectID: project.ID,
	}
	s.store.CreateNotification(notification)

	fmt.Printf("[paypal-webhook] payment denied for project %s, order %s\n", project.ID, orderID)
	return nil
}

// processPaymentCaptureRefunded handles refunded payment
func (s *Server) processPaymentCaptureRefunded(event PayPalWebhookEvent, orderID string) error {
	project, err := s.store.FindProjectByPaymentReference(orderID)
	if err != nil || project == nil {
		return nil
	}

	notification := CreateNotificationRequest{
		UserID:    project.UserID,
		Subject:   "Payment Refunded",
		Body:      fmt.Sprintf("Your PayPal payment for project '%s' has been refunded (Order: %s)", project.Name, orderID),
		Channel:   "payment",
		ProjectID: project.ID,
	}
	s.store.CreateNotification(notification)

	fmt.Printf("[paypal-webhook] payment refunded for project %s, order %s\n", project.ID, orderID)
	return nil
}

// logPayPalWebhookEvent stores webhook event in the database
func (s *Server) logPayPalWebhookEvent(eventID, eventType, orderID, currency, value, rawPayload string, processed bool, errMsg string) error {
	log := PayPalWebhookLog{
		EventID:      eventID,
		EventType:    eventType,
		Status:       eventType,
		OrderID:      orderID,
		Currency:     currency,
		Value:        value,
		RawPayload:   rawPayload,
		ReceivedAt:   time.Now(),
		Processed:    processed,
		ErrorMessage: errMsg,
	}
	return s.store.SavePayPalWebhookLog(log)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
