package core

import (
	"encoding/json"
	"testing"
)

func TestPayPalWebhookEventParse(t *testing.T) {
	data := `{"id":"WH-1","event_type":"PAYMENT.CAPTURE.COMPLETED","resource":{"id":"ORD-1","status":"COMPLETED","purchase_units":[{"payments":{"captures":[{"id":"CAP-1","status":"COMPLETED","amount":{"currency_code":"USD","value":"10.00"}}]}}]}}`
	var event paypalWebhookEvent
	if err := json.Unmarshal([]byte(data), &event); err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if event.ID != "WH-1" {
		t.Errorf("expected WH-1, got %s", event.ID)
	}
	if event.EventType != "PAYMENT.CAPTURE.COMPLETED" {
		t.Errorf("expected PAYMENT.CAPTURE.COMPLETED, got %s", event.EventType)
	}
	var res paypalOrderResource
	if err := json.Unmarshal(event.Resource, &res); err != nil {
		t.Fatalf("resource parse error: %v", err)
	}
	if res.ID != "ORD-1" {
		t.Errorf("expected ORD-1, got %s", res.ID)
	}
	if len(res.PurchaseUnits) == 0 {
		t.Fatal("expected purchase units")
	}
}

func TestPayPalWebhookAllEventTypes(t *testing.T) {
	types := []string{
		"PAYMENT.CAPTURE.COMPLETED",
		"PAYMENT.CAPTURE.DENIED",
		"PAYMENT.CAPTURE.DECLINED",
		"PAYMENT.CAPTURE.REFUNDED",
		"CHECKOUT.ORDER.APPROVED",
		"CHECKOUT.ORDER.COMPLETED",
	}
	for _, et := range types {
		data := `{"id":"test","event_type":"` + et + `","resource":{"id":"ORD-1"}}`
		var event paypalWebhookEvent
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			t.Errorf("parse error for %s: %v", et, err)
		}
		if event.EventType != et {
			t.Errorf("expected %s, got %s", et, event.EventType)
		}
	}
}
