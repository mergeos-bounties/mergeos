package core

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strings"
)

// paypalWebhookEvent represents a PayPal webhook notification
type paypalWebhookEvent struct {
	ID           string          `json:"id"`
	EventType    string          `json:"event_type"`
	EventVersion string          `json:"event_version"`
	CreateTime   string          `json:"create_time"`
	ResourceType string          `json:"resource_type"`
	Resource     json.RawMessage `json:"resource"`
}

type paypalOrderResource struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	Intent string `json:"intent"`
	Payer  *struct {
		PayerID      string `json:"payer_id"`
		EmailAddress string `json:"email_address"`
	} `json:"payer"`
	PurchaseUnits []struct {
		Payments *struct {
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
}

type paypalCaptureResource struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	Amount struct {
		CurrencyCode string `json:"currency_code"`
		Value        string `json:"value"`
	} `json:"amount"`
	SupplementaryData struct {
		RelatedIDs struct {
			OrderID string `json:"order_id"`
		} `json:"related_ids"`
	} `json:"supplementary_data"`
}

type paypalWebhookPayment struct {
	OrderID     string
	CaptureID   string
	Status      string
	Currency    string
	Value       string
	AmountCents int64
}

// handlePayPalWebhook processes incoming PayPal webhook notifications.
func (s *Server) handlePayPalWebhook(w http.ResponseWriter, r *http.Request) {
	bodyBytes, err := io.ReadAll(io.LimitReader(r.Body, 256*1024))
	if r.Body != nil {
		defer r.Body.Close()
	}
	if err != nil {
		log.Printf("[paypal-webhook] read error: %v", err)
		writeError(w, http.StatusBadRequest, "failed to read body")
		return
	}

	var event paypalWebhookEvent
	if err := json.Unmarshal(bodyBytes, &event); err != nil {
		log.Printf("[paypal-webhook] parse error: %v", err)
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	verified, err := s.payments.verifyPayPalWebhookSignature(r.Context(), r.Header, event)
	if err != nil {
		log.Printf("[paypal-webhook] signature verification error: %v", err)
		writeError(w, http.StatusUnauthorized, "invalid PayPal webhook signature")
		return
	}
	if !verified {
		writeError(w, http.StatusUnauthorized, "invalid PayPal webhook signature")
		return
	}

	payment, err := paypalWebhookPaymentFromEvent(event)
	if err != nil {
		log.Printf("[paypal-webhook] ignored event=%s id=%s: %v", event.EventType, event.ID, err)
		writeJSON(w, http.StatusOK, map[string]any{
			"status":     "ignored",
			"event_id":   event.ID,
			"event_type": event.EventType,
			"reason":     err.Error(),
		})
		return
	}
	settlement, err := s.store.RecordPayPalWebhookPayment(event.ID, payment)
	if err != nil {
		log.Printf("[paypal-webhook] settlement error: %v", err)
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if settlement.Status == "verified" && !settlement.Duplicate {
		s.broadcastLiveFeedEvent("payment_verified")
	}
	log.Printf("[paypal-webhook] event=%s order=%s capture=%s status=%s duplicate=%t", event.EventType, payment.OrderID, payment.CaptureID, settlement.Status, settlement.Duplicate)
	writeJSON(w, http.StatusOK, settlement)
}

func paypalWebhookPaymentFromEvent(event paypalWebhookEvent) (paypalWebhookPayment, error) {
	switch strings.ToUpper(strings.TrimSpace(event.EventType)) {
	case "PAYMENT.CAPTURE.COMPLETED":
		var capture paypalCaptureResource
		if err := json.Unmarshal(event.Resource, &capture); err != nil {
			return paypalWebhookPayment{}, err
		}
		payment := paypalWebhookPayment{
			OrderID:   strings.TrimSpace(capture.SupplementaryData.RelatedIDs.OrderID),
			CaptureID: strings.TrimSpace(capture.ID),
			Status:    strings.TrimSpace(capture.Status),
			Currency:  strings.TrimSpace(capture.Amount.CurrencyCode),
			Value:     strings.TrimSpace(capture.Amount.Value),
		}
		if payment.OrderID == "" {
			payment.OrderID = strings.TrimSpace(capture.ID)
		}
		return validatePayPalWebhookPayment(payment)
	case "CHECKOUT.ORDER.COMPLETED":
		var order paypalOrderResource
		if err := json.Unmarshal(event.Resource, &order); err != nil {
			return paypalWebhookPayment{}, err
		}
		payment := paypalWebhookPayment{
			OrderID: strings.TrimSpace(order.ID),
			Status:  strings.TrimSpace(order.Status),
		}
		for _, unit := range order.PurchaseUnits {
			if unit.Payments == nil || len(unit.Payments.Captures) == 0 {
				continue
			}
			capture := unit.Payments.Captures[0]
			payment.CaptureID = strings.TrimSpace(capture.ID)
			payment.Status = strings.TrimSpace(capture.Status)
			payment.Currency = strings.TrimSpace(capture.Amount.CurrencyCode)
			payment.Value = strings.TrimSpace(capture.Amount.Value)
			break
		}
		return validatePayPalWebhookPayment(payment)
	default:
		return paypalWebhookPayment{}, errors.New("event type is not a completed PayPal payment")
	}
}

func validatePayPalWebhookPayment(payment paypalWebhookPayment) (paypalWebhookPayment, error) {
	if strings.TrimSpace(payment.OrderID) == "" && strings.TrimSpace(payment.CaptureID) == "" {
		return paypalWebhookPayment{}, errors.New("paypal order or capture id is required")
	}
	if !strings.EqualFold(strings.TrimSpace(payment.Status), "COMPLETED") {
		return paypalWebhookPayment{}, errors.New("paypal payment is not completed")
	}
	if !strings.EqualFold(strings.TrimSpace(payment.Currency), "USD") {
		return paypalWebhookPayment{}, errors.New("paypal currency is not USD")
	}
	cents, err := payPalValueToCents(payment.Value)
	if err != nil {
		return paypalWebhookPayment{}, err
	}
	if cents <= 0 {
		return paypalWebhookPayment{}, errors.New("paypal amount must be positive")
	}
	payment.AmountCents = cents
	return payment, nil
}
