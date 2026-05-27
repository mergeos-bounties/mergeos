package core

import (
	"encoding/json"
	"testing"
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
			"purchase_units": [{
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

	var event paypalWebhookEvent
	if err := json.Unmarshal([]byte(testData), &event); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if event.ID != "WH-TEST-123" {
		t.Errorf("expected ID WH-TEST-123, got %s", event.ID)
	}
	if event.EventType != "PAYMENT.CAPTURE.COMPLETED" {
		t.Errorf("expected PAYMENT.CAPTURE.COMPLETED, got %s", event.EventType)
	}

	var res paypalResource
	if err := json.Unmarshal(event.Resource, &res); err != nil {
		t.Fatalf("failed to unmarshal resource: %v", err)
	}
	if res.ID != "ORDER-123" {
		t.Errorf("expected ORDER-123, got %s", res.ID)
	}
	if res.Status != "COMPLETED" {
		t.Errorf("expected COMPLETED, got %s", res.Status)
	}
	if len(res.PurchaseUnits) == 0 || res.PurchaseUnits[0].Payments == nil || len(res.PurchaseUnits[0].Payments.Captures) == 0 {
		t.Fatal("expected captures")
	}
	cap := res.PurchaseUnits[0].Payments.Captures[0]
	if cap.Status != "COMPLETED" {
		t.Errorf("expected capture COMPLETED, got %s", cap.Status)
	}
	if cap.Amount.CurrencyCode != "USD" {
		t.Errorf("expected USD, got %s", cap.Amount.CurrencyCode)
	}
	if cap.Amount.Value != "150.00" {
		t.Errorf("expected 150.00, got %s", cap.Amount.Value)
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

	for _, et := range eventTypes {
		data := `{"id": "test-1", "event_type": "` + et + `", "resource": {"id": "ORDER-1"}}`
		var event paypalWebhookEvent
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			t.Errorf("failed to unmarshal %s: %v", et, err)
		}
		if event.EventType != et {
			t.Errorf("expected %s, got %s", et, event.EventType)
		}
	}
}

func TestPayPalWebhookLog(t *testing.T) {
	log := paypalWebhookLog{
		EventID:   "WH-LOG-1",
		EventType: "PAYMENT.CAPTURE.COMPLETED",
		OrderID:   "ORDER-1",
		Currency:  "USD",
		Value:     "150.00",
		Processed: true,
	}

	if log.EventID != "WH-LOG-1" {
		t.Errorf("expected WH-LOG-1, got %s", log.EventID)
	}
	if !log.Processed {
		t.Error("expected processed=true")
	}
}

func TestPayPalWebhookResourceExtraction(t *testing.T) {
	data := `{
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

	var event paypalWebhookEvent
	if err := json.Unmarshal([]byte(data), &event); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	var res paypalResource
	json.Unmarshal(event.Resource, &res)

	orderID := res.ID
	currency := ""
	value := ""
	for _, pu := range res.PurchaseUnits {
		if pu.Payments != nil && len(pu.Payments.Captures) > 0 {
			cap := pu.Payments.Captures[0]
			if cap.Status == "COMPLETED" {
				currency = cap.Amount.CurrencyCode
				value = cap.Amount.Value
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
