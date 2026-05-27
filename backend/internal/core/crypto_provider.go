package core

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// ---------------------------------------------------------------
// Provider abstraction — allows adding new crypto gateways
// ---------------------------------------------------------------

// CryptoProvider defines the interface for a crypto payment gateway.
type CryptoProvider interface {
	// Name returns a short identifier, e.g. "nowpayments", "coinbase-commerce".
	Name() string

	// CreateInvoice creates a payment invoice/order and returns a charge URL
	// or payment address for the buyer.
	CreateInvoice(ctx context.Context, req CryptoInvoiceRequest) (*CryptoInvoice, error)

	// VerifyWebhook parses and authenticates an incoming webhook payload,
	// returning a payment status update.  providerCfg holds provider-specific
	// settings (API keys, secrets, sandbox flag, etc.) already resolved by
	// the caller.
	VerifyWebhook(r *http.Request, body []byte, providerCfg map[string]string) (*CryptoPaymentUpdate, error)

	// VerifyOnChain verifies an on-chain transaction (used as a fallback /
	// admin reconciliation).  Returns nil when the provider does not support
	// on-chain lookup (the caller falls back to the generic EVM verifier).
	VerifyOnChain(ctx context.Context, txHash string, expectedCents int64, providerCfg map[string]string) (*CryptoPaymentUpdate, error)
}

// CryptoInvoiceRequest is the input for creating a payment invoice.
type CryptoInvoiceRequest struct {
	OrderID      string // internal MergeOS order / project id
	Title        string // short description
	AmountCents  int64  // amount in USD cents
	Currency     string // "USDT", "USDC", "DAI", etc.
	CallbackURL  string // URL the gateway should POST to
	CancelURL    string // redirect on cancellation
	SuccessURL   string // redirect on success
}

// CryptoInvoice is the result of creating an invoice.
type CryptoInvoice struct {
	InvoiceID    string            `json:"invoiceId"`    // gateway invoice id
	PaymentURL   string            `json:"paymentUrl"`   // hosted payment page URL
	Address      string            `json:"address"`      // direct wallet address (when supported)
	ExpectedUSD  string            `json:"expectedUsd"`  // expected USD amount
	Status       string            `json:"status"`
	Extra        map[string]string `json:"extra,omitempty"` // provider-specific data
}

// CryptoPaymentUpdate is a normalised payment-status event from any provider.
type CryptoPaymentUpdate struct {
	InvoiceID      string `json:"invoiceId"`
	TransactionID  string `json:"transactionId"`  // blockchain tx hash when available
	Status         string `json:"status"`          // pending | confirmed | expired | failed | refunded
	AmountReceived string `json:"amountReceived"`  // decimal string in the crypto currency
	Currency       string `json:"currency"`        // "USDT", "USDC", etc.
	USDEquivalent  int64  `json:"usdEquivalent"`   // USD cents (estimated / reported by gateway)
	RawPayload     string `json:"rawPayload"`      // full verified payload for the proof ledger
	ConfirmedAt    int64  `json:"confirmedAt"`     // unix timestamp
}

// ---------------------------------------------------------------------------
// NowPayments provider — USDT/ERC20 via nowpayments.io
// ---------------------------------------------------------------------------

type NowPaymentsProvider struct {
	httpClient *http.Client
}

func NewNowPaymentsProvider() *NowPaymentsProvider {
	return &NowPaymentsProvider{
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func (np *NowPaymentsProvider) Name() string { return "nowpayments" }

// NowPayments API structures (partial — we only need invoice creation + webhook).

type npCreateInvoiceReq struct {
	PriceAmount    float64 `json:"price_amount"`
	PriceCurrency  string  `json:"price_currency"`
	PayCurrency    string  `json:"pay_currency"`
	IPNCallbackURL string  `json:"ipn_callback_url,omitempty"`
	OrderID        string  `json:"order_id,omitempty"`
	OrderDesc      string  `json:"order_description,omitempty"`
	CancelURL      string  `json:"cancel_url,omitempty"`
	SuccessURL     string  `json:"success_url,omitempty"`
	IsFeePaidByUser bool   `json:"is_fee_paid_by_user,omitempty"`
}

type npCreateInvoiceResp struct {
	InvoiceID       string  `json:"invoice_id"`
	PaymentID       string  `json:"payment_id"`
	PaymentStatus   string  `json:"payment_status"`
	PayAddress      string  `json:"pay_address"`
	PriceAmount     float64 `json:"price_amount"`
	PriceCurrency   string  `json:"price_currency"`
	PayAmount       float64 `json:"pay_amount"`
	PayCurrency     string  `json:"pay_currency"`
	CreatedAt       string  `json:"created_at"`
	ExpirationEstimate string `json:"expiration_estimate"`
	InvoiceURL      string  `json:"invoice_url"`
}

func (np *NowPaymentsProvider) CreateInvoice(ctx context.Context, req CryptoInvoiceRequest) (*CryptoInvoice, error) {
	// Resolve API key from context or passed via caller — this is called through
	// PaymentManager which reads config from the providerCfg map.
	apiKey := extractFromContext(ctx, "np-api-key")
	if apiKey == "" {
		return nil, errors.New("nowpayments: missing API key (set NP_API_KEY)")
	}

	createReq := npCreateInvoiceReq{
		PriceAmount:    float64(req.AmountCents) / 100.0,
		PriceCurrency:  "usd",
		PayCurrency:    strings.ToLower(req.Currency),
		IPNCallbackURL: req.CallbackURL,
		OrderID:        req.OrderID,
		OrderDesc:      truncate(req.Title, 200),
		CancelURL:      req.CancelURL,
		SuccessURL:     req.SuccessURL,
		IsFeePaidByUser: false,
	}

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(createReq); err != nil {
		return nil, fmt.Errorf("nowpayments: encode error: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://api.nowpayments.io/v1/invoice", &buf)
	if err != nil {
		return nil, fmt.Errorf("nowpayments: request error: %w", err)
	}
	httpReq.Header.Set("x-api-key", apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := np.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("nowpayments: http error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("nowpayments: api error %d: %s", resp.StatusCode, string(body))
	}

	var created npCreateInvoiceResp
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		return nil, fmt.Errorf("nowpayments: decode error: %w", err)
	}

	return &CryptoInvoice{
		InvoiceID:   created.InvoiceID,
		PaymentURL:  created.InvoiceURL,
		Address:     created.PayAddress,
		ExpectedUSD: fmt.Sprintf("%.2f", created.PriceAmount),
		Status:      created.PaymentStatus,
		Extra: map[string]string{
			"payment_id":   created.PaymentID,
			"pay_amount":   fmt.Sprintf("%f", created.PayAmount),
			"pay_currency": created.PayCurrency,
		},
	}, nil
}

// NowPayments IPN webhook verification.
// Ref: https://documenter.getpostman.com/view/7907941/2s93Y3vHfT#ae4a92dc-c83a-4c77-86e0-4cb8519eea34
func (np *NowPaymentsProvider) VerifyWebhook(r *http.Request, body []byte, providerCfg map[string]string) (*CryptoPaymentUpdate, error) {
	// 1. Signature verification (SHA-512 HMAC)
	secret := providerCfg["np_ipn_secret"]
	if secret == "" {
		return nil, errors.New("nowpayments: missing IPN secret (set NP_IPN_SECRET)")
	}
	sigHeader := r.Header.Get("x-nowpayments-sig")
	if sigHeader == "" {
		return nil, errors.New("nowpayments: missing x-nowpayments-sig header")
	}
	mac := hmac.New(sha512.New, []byte(secret))
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(sigHeader), []byte(expected)) {
		return nil, errors.New("nowpayments: invalid IPN signature")
	}

	// 2. Parse payload
	var ipn struct {
		PaymentID       string `json:"payment_id"`
		InvoiceID       string `json:"invoice_id"`
		PaymentStatus   string `json:"payment_status"`
		PayAddress      string `json:"pay_address"`
		PriceAmount     float64 `json:"price_amount"`
		PriceCurrency   string `json:"price_currency"`
		PayAmount       float64 `json:"pay_amount"`
		PayCurrency     string `json:"pay_currency"`
		ActuallyPaid    float64 `json:"actually_paid"`
		ActuallyPaidUSD float64 `json:"actually_paid_usd"`
		TxID            string  `json:"txid"`     // might be tx_id in some versions
		TxIDAlt         string  `json:"tx_id"`
		PurchaseID      string  `json:"purchase_id"`
		CreatedAt       string  `json:"created_at"`
		UpdatedAt       string  `json:"updated_at"`
	}
	if err := json.Unmarshal(body, &ipn); err != nil {
		return nil, fmt.Errorf("nowpayments: parse error: %w", err)
	}

	// 3. Map NowPayments status codes to MergeOS statuses
	//    waiting | confirming | confirmed | sending | partially_paid |
	//    finished | failed | refunded | expired
	var mappedStatus string
	switch ipn.PaymentStatus {
	case "waiting":
		mappedStatus = "pending"
	case "confirming", "sending":
		mappedStatus = "confirming"
	case "confirmed", "finished":
		mappedStatus = "confirmed"
	case "partially_paid":
		mappedStatus = "confirmed" // treat partial as confirmed (admin review)
	case "failed":
		mappedStatus = "failed"
	case "refunded":
		mappedStatus = "refunded"
	case "expired":
		mappedStatus = "expired"
	default:
		mappedStatus = "pending"
	}

	txID := ipn.TxID
	if txID == "" {
		txID = ipn.TxIDAlt
	}

	usdEquivalent := int64(ipn.ActuallyPaidUSD * 100)
	if usdEquivalent <= 0 {
		usdEquivalent = int64(ipn.PriceAmount * 100)
	}

	bodyStr := string(body)
	update := &CryptoPaymentUpdate{
		InvoiceID:      ipn.InvoiceID,
		TransactionID:  txID,
		Status:         mappedStatus,
		AmountReceived: fmt.Sprintf("%f", ipn.ActuallyPaid),
		Currency:       strings.ToUpper(ipn.PayCurrency),
		USDEquivalent:  usdEquivalent,
		RawPayload:     bodyStr,
		ConfirmedAt:    time.Now().Unix(),
	}

	return update, nil
}

func (np *NowPaymentsProvider) VerifyOnChain(ctx context.Context, txHash string, expectedCents int64, providerCfg map[string]string) (*CryptoPaymentUpdate, error) {
	// NowPayments does not expose an on-chain lookup API; return nil so the
	// caller uses the generic EVM verifier.
	return nil, nil
}

// ---------------------------------------------------------------------------
// Provider registry
// ---------------------------------------------------------------------------

var registeredProviders = map[string]CryptoProvider{
	"nowpayments": NewNowPaymentsProvider(),
}

// RegisterCryptoProvider lets callers add custom providers at init time.
func RegisterCryptoProvider(name string, p CryptoProvider) {
	registeredProviders[name] = p
}

// GetCryptoProvider returns a provider by name, or nil.
func GetCryptoProvider(name string) CryptoProvider {
	return registeredProviders[name]
}

// ListCryptoProviders returns all registered provider names.
func ListCryptoProviders() []string {
	names := make([]string, 0, len(registeredProviders))
	for n := range registeredProviders {
		names = append(names, n)
	}
	return names
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

type ctxKey string

func extractFromContext(ctx context.Context, key string) string {
	v, _ := ctx.Value(ctxKey(key)).(string)
	return v
}

func ContextWithAPIKey(ctx context.Context, provider, apiKey string) context.Context {
	return context.WithValue(ctx, ctxKey(provider+"-api-key"), apiKey)
}

func truncate(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "..."
}

// ProviderConfigFromEnv builds a provider config map from the MergeOS Config.
// This is called by PaymentManager when it needs to call a provider.
func ProviderConfigFromEnv(cfg Config, providerName string) map[string]string {
	m := make(map[string]string)
	switch providerName {
	case "nowpayments":
		if cfg.NPAPIKey != "" {
			m["np_api_key"] = cfg.NPAPIKey
		}
		if cfg.NPIPNSecret != "" {
			m["np_ipn_secret"] = cfg.NPIPNSecret
		}
		m["sandbox"] = fmt.Sprintf("%t", cfg.NPSandbox)
	}
	return m
}
