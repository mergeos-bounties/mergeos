package core

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func TestPaymentManagerVerifyStripePaymentIntent(t *testing.T) {
	cfg := Config{
		StripePublishableKey: "pk_test_mergeos",
		StripeSecretKey:      "sk_test_secret",
		StripeWebhookSecret:  "whsec_secret",
	}
	payments := NewPaymentManager(cfg)
	payments.client = &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			if req.Method != http.MethodGet {
				t.Fatalf("method = %s", req.Method)
			}
			if req.URL.String() != "https://api.stripe.com/v1/payment_intents/pi_test_123" {
				t.Fatalf("url = %s", req.URL.String())
			}
			if req.Header.Get("Authorization") != "Bearer sk_test_secret" {
				t.Fatalf("authorization header = %q", req.Header.Get("Authorization"))
			}
			body := `{"id":"pi_test_123","status":"succeeded","currency":"usd","amount":120000,"amount_received":120000}`
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       io.NopCloser(strings.NewReader(body)),
			}, nil
		}),
	}

	verification, err := payments.Verify(context.Background(), CreateProjectRequest{
		PaymentMethod:    PaymentStripe,
		PaymentReference: "pi_test_123",
		BudgetCents:      120000,
	})
	if err != nil {
		t.Fatal(err)
	}
	if verification.Provider != "stripe" || verification.Reference != "pi_test_123" {
		t.Fatalf("verification = %#v", verification)
	}
}
