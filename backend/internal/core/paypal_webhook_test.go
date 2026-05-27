package core

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"
)

func TestPayPalWebhookEventUnmarshal(t *testing.T) {
	testData := `{
		"id": "WH-TEST-123",
		"event_type": "PAYMENT.CAPTURE.COMPLETED",
		"event_version": "1.0",
		"create_time": "2026-05-27T01:00:00Z",
		"resource_type": "capture",
		"resource": {
			"id": "ORDER-123",
			"status": "COMPLETED",
			"intent": "CAPTURE",
			"purchase_units": [{
				"reference_id": "default",
				"amount": {
					"currency_code": "USD",
					"value": "150.00"
				},
				"payments": {
					"captures": [{
						"id": "CAPTURE-456",
						"status": "COMPLETED",
						"amount": {
							"currency_code": "USD",
							"value": "150.00"
						}
					}]
				}
			}],
			"payer": {
				"payer_id": "PAYER-789",
				"email_address": "buyer@example.com"
			}
		}
	}`

	var event PayPalWebhookEvent
	if err := json.Unmarshal([]byte(testData), &event); err != nil {
		t.Fatalf("failed to unmarshal webhook event: %v", err)
	}

	if event.ID != "WH-TEST-123" {
		t.Errorf("expected event ID WH-TEST-123, got %s", event.ID)
	}
	if event.EventType != "PAYMENT.CAPTURE.COMPLETED" {
		t.Errorf("expected event type PAYMENT.CAPTURE.COMPLETED, got %s", event.EventType)
	}
	if event.Resource == nil {
		t.Fatal("expected resource to be non-nil")
	}
	if event.Resource.ID != "ORDER-123" {
		t.Errorf("expected order ID ORDER-123, got %s", event.Resource.ID)
	}
	if event.Resource.Status != "COMPLETED" {
		t.Errorf("expected status COMPLETED, got %s", event.Resource.Status)
	}
	if len(event.Resource.PurchaseUnits) == 0 {
		t.Fatal("expected at least one purchase unit")
	}
	pu := event.Resource.PurchaseUnits[0]
	if pu.Payments == nil || len(pu.Payments.Captures) == 0 {
		t.Fatal("expected captures in purchase unit")
	}
	capture := pu.Payments.Captures[0]
	if capture.Status != "COMPLETED" {
		t.Errorf("expected capture status COMPLETED, got %s", capture.Status)
	}
	if capture.Amount.CurrencyCode != "USD" {
		t.Errorf("expected currency USD, got %s", capture.Amount.CurrencyCode)
	}
	if capture.Amount.Value != "150.00" {
		t.Errorf("expected value 150.00, got %s", capture.Amount.Value)
	}
	if event.Resource.Payer == nil {
		t.Fatal("expected payer info")
	}
	if event.Resource.Payer.EmailAddress != "buyer@example.com" {
		t.Errorf("expected buyer@example.com, got %s", event.Resource.Payer.EmailAddress)
	}
}

func TestPayPalWebhookEventTypes(t *testing.T) {
	eventTypes := []string{
		"PAYMENT.CAPTURE.COMPLETED",
		"PAYMENT.CAPTURE.DENIED",
		"PAYMENT.CAPTURE.DECLINED",
		"PAYMENT.CAPTURE.REFUNDED",
		"CHECKOUT.ORDER.APPROVED",
		"CHECKOUT.ORDER.COMPLETED",
	}

	for _, eventType := range eventTypes {
		testData := fmt.Sprintf(`{"id": "test-1", "event_type": "%s", "resource": {"id": "ORDER-1", "status": "COMPLETED"}}`, eventType)
		var event PayPalWebhookEvent
		if err := json.Unmarshal([]byte(testData), &event); err != nil {
			t.Errorf("failed to unmarshal event type %s: %v", eventType, err)
		}
		if event.EventType != eventType {
			t.Errorf("expected event type %s, got %s", eventType, event.EventType)
		}
	}
}

func TestPayPalWebhookLog(t *testing.T) {
	log := PayPalWebhookLog{
		EventID:    "WH-LOG-1",
		EventType:  "PAYMENT.CAPTURE.COMPLETED",
		OrderID:    "ORDER-1",
		Currency:   "USD",
		Value:      "150.00",
		RawPayload: `{"id": "test"}`,
		ReceivedAt: time.Now(),
		Processed:  true,
	}

	if log.EventID != "WH-LOG-1" {
		t.Errorf("expected event ID WH-LOG-1, got %s", log.EventID)
	}
	if !log.Processed {
		t.Error("expected log to be processed")
	}
}

func TestPayPalWebhookResourceExtraction(t *testing.T) {
	testData := `{
		"id": "WH-EXTRACT-1",
		"event_type": "PAYMENT.CAPTURE.COMPLETED",
		"resource": {
			"id": "ORDER-EXTRACT-1",
			"status": "COMPLETED",
			"purchase_units": [{
				"payments": {
					"captures": [{
						"status": "COMPLETED",
						"amount": {
							"currency_code": "USD",
							"value": "250.00"
						}
					}]
				}
			}]
		}
	}`

	var event PayPalWebhookEvent
	if err := json.Unmarshal([]byte(testData), &event); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	orderID := ""
	currency := ""
	value := ""
	if event.Resource != nil {
		orderID = event.Resource.ID
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

	if orderID != "ORDER-EXTRACT-1" {
		t.Errorf("expected ORDER-EXTRACT-1, got %s", orderID)
	}
	if currency != "USD" {
		t.Errorf("expected USD, got %s", currency)
	}
	if value != "250.00" {
		t.Errorf("expected 250.00, got %s", value)
	}
}
