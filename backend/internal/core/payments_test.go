package core

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestVerifyCryptoAcceptsSolanaSPLTransfer(t *testing.T) {
	signature := base58Encode(bytes.Repeat([]byte{1}, 64))
	receiver := base58Encode(bytes.Repeat([]byte{2}, walletAddressBytes))
	mint := base58Encode(bytes.Repeat([]byte{3}, walletAddressBytes))
	calls := map[string]int{}

	rpc := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Method string          `json:"method"`
			Params json.RawMessage `json:"params"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode rpc request: %v", err)
		}
		calls[req.Method]++
		w.Header().Set("Content-Type", "application/json")
		switch req.Method {
		case "getSignatureStatuses":
			_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":{"value":[{"confirmations":null,"confirmationStatus":"finalized","err":null}]}}`))
		case "getTransaction":
			body := map[string]any{
				"jsonrpc": "2.0",
				"id":      1,
				"result": map[string]any{
					"slot": 100,
					"meta": map[string]any{
						"err":               nil,
						"preTokenBalances":  []any{},
						"postTokenBalances": []any{},
						"innerInstructions": []any{},
					},
					"transaction": map[string]any{
						"message": map[string]any{
							"accountKeys": []any{map[string]any{"pubkey": receiver}},
							"instructions": []any{
								map[string]any{
									"program":   "spl-token",
									"programId": "TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA",
									"parsed": map[string]any{
										"type": "transferChecked",
										"info": map[string]any{
											"mint":        mint,
											"destination": receiver,
											"tokenAmount": map[string]any{"amount": "100000000", "decimals": 6},
										},
									},
								},
							},
						},
					},
				},
			}
			if err := json.NewEncoder(w).Encode(body); err != nil {
				t.Fatalf("encode rpc response: %v", err)
			}
		default:
			t.Fatalf("unexpected rpc method %q", req.Method)
		}
	}))
	defer rpc.Close()

	payments := NewPaymentManager(Config{
		CryptoRPCURL:           rpc.URL,
		CryptoReceiver:         receiver,
		CryptoAsset:            "spl",
		CryptoTokenContract:    mint,
		CryptoTokenDecimals:    6,
		CryptoMinConfirmations: 1,
	})
	verification, err := payments.verifyCrypto(context.Background(), signature, 10000)
	if err != nil {
		t.Fatal(err)
	}
	if verification.Provider != "solana-spl" || verification.Reference != signature {
		t.Fatalf("verification = %#v", verification)
	}
	if calls["getSignatureStatuses"] != 1 || calls["getTransaction"] != 1 {
		t.Fatalf("rpc calls = %#v", calls)
	}
}

func TestCreatePayPalOrderPostsCheckoutOrder(t *testing.T) {
	paypal := newPayPalCreateOrderServer(t, "ORDER-UNIT-1", func(req *http.Request, body map[string]any) {
		if got := req.Header.Get("Authorization"); got != "Bearer test-paypal-token" {
			t.Fatalf("authorization header = %q", got)
		}
		if got := req.Header.Get("PayPal-Request-Id"); !strings.HasPrefix(got, "mergeos-order-") {
			t.Fatalf("paypal request id = %q", got)
		}
		if body["intent"] != "CAPTURE" {
			t.Fatalf("intent = %#v", body["intent"])
		}
		units, ok := body["purchase_units"].([]any)
		if !ok || len(units) != 1 {
			t.Fatalf("purchase_units = %#v", body["purchase_units"])
		}
		unit, ok := units[0].(map[string]any)
		if !ok {
			t.Fatalf("purchase unit = %#v", units[0])
		}
		if unit["description"] != "MergeOS project" {
			t.Fatalf("description = %#v", unit["description"])
		}
		amount, ok := unit["amount"].(map[string]any)
		if !ok || amount["currency_code"] != "USD" || amount["value"] != "1200.00" {
			t.Fatalf("amount = %#v", unit["amount"])
		}
		appContext, ok := body["application_context"].(map[string]any)
		if !ok || appContext["return_url"] != "https://mergeos.shop/paypal/return" || appContext["cancel_url"] != "https://mergeos.shop/paypal/cancel" {
			t.Fatalf("application_context = %#v", body["application_context"])
		}
	})
	defer paypal.Close()

	payments := NewPaymentManager(Config{
		PayPalEnvironment:  paypal.URL,
		PayPalClientID:     "paypal-client",
		PayPalClientSecret: "paypal-secret",
	})
	order, err := payments.CreatePayPalOrder(context.Background(), CreatePayPalOrderRequest{
		AmountCents: 120000,
		Description: "MergeOS project",
		Flow:        PaymentOrderFlowProjectFunding,
		ReturnURL:   "https://mergeos.shop/paypal/return",
		CancelURL:   "https://mergeos.shop/paypal/cancel",
	})
	if err != nil {
		t.Fatal(err)
	}
	if order.OrderID != "ORDER-UNIT-1" || order.PaymentReference != "ORDER-UNIT-1" || order.Provider != "paypal" || order.Flow != PaymentOrderFlowProjectFunding {
		t.Fatalf("order = %#v", order)
	}
	if order.ApprovalURL != "https://paypal.test/checkout/ORDER-UNIT-1" || order.AmountCents != 120000 || order.Currency != "USD" {
		t.Fatalf("order metadata = %#v", order)
	}
}

func TestCreateCardPaymentIntentUsesDevStripeWhenVerifierIsLocal(t *testing.T) {
	payments := NewPaymentManager(Config{
		DevPaymentEnabled: true,
		DevPaymentCode:    defaultDevPaymentCode,
	})
	intent, err := payments.CreateCardPaymentIntent(context.Background(), CreateCardPaymentIntentRequest{
		AmountCents: 120000,
		Description: "MergeOS test funding",
		Flow:        "project_funding",
	})
	if err != nil {
		t.Fatal(err)
	}
	if intent.PaymentReference != defaultDevPaymentCode || intent.Provider != "dev-stripe" || intent.Status != "succeeded" {
		t.Fatalf("intent = %#v", intent)
	}
}

func TestCreateCardPaymentIntentPostsStripePaymentIntent(t *testing.T) {
	payments := NewPaymentManager(Config{
		StripePublishableKey: "pk_test_mergeos",
		StripeSecretKey:      "sk_test_secret",
		StripeWebhookSecret:  "whsec_secret",
	})
	payments.client = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.Method != http.MethodPost || req.URL.String() != "https://api.stripe.com/v1/payment_intents" {
			t.Fatalf("unexpected stripe request %s %s", req.Method, req.URL.String())
		}
		if got := req.Header.Get("Authorization"); got != "Bearer sk_test_secret" {
			t.Fatalf("authorization header = %q", got)
		}
		body, err := io.ReadAll(req.Body)
		if err != nil {
			t.Fatal(err)
		}
		form := string(body)
		for _, expected := range []string{
			"amount=120000",
			"currency=usd",
			"automatic_payment_methods%5Benabled%5D=true",
			"description=MergeOS+project",
			"metadata%5Bmergeos_flow%5D=project_funding",
		} {
			if !strings.Contains(form, expected) {
				t.Fatalf("stripe form %q missing %q", form, expected)
			}
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(`{"id":"pi_mergeos_123","client_secret":"dummy","status":"requires_payment_method"}`)),
		}, nil
	})}

	intent, err := payments.CreateCardPaymentIntent(context.Background(), CreateCardPaymentIntentRequest{
		AmountCents: 120000,
		Description: "MergeOS project",
		Flow:        "project_funding",
	})
	if err != nil {
		t.Fatal(err)
	}
	if intent.PaymentReference != "pi_mergeos_123" || intent.ClientSecret == "" || intent.PublicKey != "pk_test_mergeos" || intent.Provider != "stripe" {
		t.Fatalf("intent = %#v", intent)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func newPayPalCreateOrderServer(t *testing.T, orderID string, onCreate func(*http.Request, map[string]any)) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v1/oauth2/token":
			username, password, ok := r.BasicAuth()
			if !ok || username != "paypal-client" || password != "paypal-secret" {
				t.Fatalf("paypal basic auth = %q/%q ok=%v", username, password, ok)
			}
			_, _ = w.Write([]byte(`{"access_token":"test-paypal-token"}`))
		case "/v2/checkout/orders":
			if r.Method != http.MethodPost {
				t.Fatalf("paypal order method = %s", r.Method)
			}
			var body map[string]any
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode paypal order body: %v", err)
			}
			if onCreate != nil {
				onCreate(r, body)
			}
			response := map[string]any{
				"id":     orderID,
				"status": "CREATED",
				"links": []map[string]string{
					{"rel": "approve", "href": "https://paypal.test/checkout/" + orderID},
				},
			}
			if err := json.NewEncoder(w).Encode(response); err != nil {
				t.Fatalf("encode paypal order response: %v", err)
			}
		default:
			t.Fatalf("unexpected PayPal path %s", r.URL.Path)
		}
	}))
}
