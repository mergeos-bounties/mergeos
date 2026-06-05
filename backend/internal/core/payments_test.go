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
			Body:       io.NopCloser(strings.NewReader(`{"id":"pi_mergeos_123","client_secret":"pi_mergeos_123_secret_abc","status":"requires_payment_method"}`)),
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
