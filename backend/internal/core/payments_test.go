package core

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
