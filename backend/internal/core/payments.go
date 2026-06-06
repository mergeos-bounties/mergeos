package core

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type PaymentVerification struct {
	Provider  string
	Reference string
}

type PaymentManager struct {
	cfg    Config
	client *http.Client
}

func NewPaymentManager(cfg Config) *PaymentManager {
	return &PaymentManager{
		cfg: cfg,
		client: &http.Client{
			Timeout: 20 * time.Second,
		},
	}
}

func (p *PaymentManager) Verify(ctx context.Context, req CreateProjectRequest) (PaymentVerification, error) {
	reference := strings.TrimSpace(req.PaymentReference)
	switch req.PaymentMethod {
	case PaymentPayPal:
		if p.cfg.PayPalReady() && reference != p.cfg.DevPaymentCode {
			return p.verifyPayPal(ctx, reference, req.BudgetCents)
		}
		return p.verifyDev(reference, "dev-paypal")
	case PaymentCrypto:
		if p.cfg.CryptoReady() && reference != p.cfg.DevPaymentCode {
			return p.verifyCrypto(ctx, reference, req.BudgetCents)
		}
		return p.verifyDev(reference, "dev-crypto")
	case PaymentUSDT:
		if p.cfg.CryptoReady() && reference != p.cfg.DevPaymentCode {
			return p.verifyCrypto(ctx, reference, req.BudgetCents)
		}
		return p.verifyDev(reference, "dev-solana-spl")
	case PaymentStripe:
		if p.cfg.StripeReady() && reference != p.cfg.DevPaymentCode {
			return p.verifyStripe(ctx, reference, req.BudgetCents)
		}
		return p.verifyDev(reference, "dev-stripe")
	default:
		return PaymentVerification{}, errors.New("payment method must be paypal, crypto, usdt, or stripe")
	}
}

func (p *PaymentManager) CreatePayPalOrder(ctx context.Context, req CreatePayPalOrderRequest) (*CreatePayPalOrderResponse, error) {
	if !p.cfg.PayPalReady() {
		return nil, errors.New("paypal credentials are not configured")
	}
	if req.AmountCents < 10000 {
		return nil, errors.New("amount must be at least 100 USD")
	}

	token, err := p.payPalAccessToken(ctx)
	if err != nil {
		return nil, err
	}

	returnURL := strings.TrimSpace(req.ReturnURL)
	cancelURL := strings.TrimSpace(req.CancelURL)
	if returnURL == "" {
		returnURL = "http://127.0.0.1:5173/paypal/return"
	}
	if cancelURL == "" {
		cancelURL = "http://127.0.0.1:5173/paypal/cancel"
	}

	body := map[string]any{
		"intent": "CAPTURE",
		"purchase_units": []map[string]any{
			{
				"description": strings.TrimSpace(req.Description),
				"amount": map[string]string{
					"currency_code": "USD",
					"value":         centsToPayPalValue(req.AmountCents),
				},
			},
		},
		"application_context": map[string]string{
			"return_url": returnURL,
			"cancel_url": cancelURL,
		},
	}

	var payload bytes.Buffer
	if err := json.NewEncoder(&payload).Encode(body); err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.payPalBaseURL()+"/v2/checkout/orders", &payload)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+token)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("PayPal-Request-Id", fmt.Sprintf("mergeos-order-%d", time.Now().UnixNano()))

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("paypal create order failed: %s", readBody(resp.Body))
	}

	var decoded struct {
		ID     string `json:"id"`
		Status string `json:"status"`
		Links  []struct {
			Href string `json:"href"`
			Rel  string `json:"rel"`
		} `json:"links"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return nil, err
	}
	approvalURL := ""
	for _, link := range decoded.Links {
		if link.Rel == "approve" {
			approvalURL = link.Href
			break
		}
	}
	return &CreatePayPalOrderResponse{
		OrderID:     decoded.ID,
		ApprovalURL: approvalURL,
		Status:      decoded.Status,
	}, nil
}

func (p *PaymentManager) CreateCardPaymentIntent(ctx context.Context, req CreateCardPaymentIntentRequest) (*CreateCardPaymentIntentResponse, error) {
	if req.AmountCents < 10000 {
		return nil, errors.New("amount must be at least 100 USD")
	}
	if !p.cfg.StripeReady() {
		if !p.cfg.DevPaymentEnabled {
			return nil, errors.New("stripe payment intents are not configured")
		}
		return &CreateCardPaymentIntentResponse{
			PaymentReference: p.cfg.DevPaymentCode,
			Status:           "succeeded",
			Provider:         "dev-stripe",
			Mode:             "local-dev-verifier",
			Brand:            "test-card",
			Last4:            "4242",
		}, nil
	}

	form := url.Values{}
	form.Set("amount", fmt.Sprintf("%d", req.AmountCents))
	form.Set("currency", "usd")
	form.Set("automatic_payment_methods[enabled]", "true")
	description := strings.TrimSpace(req.Description)
	if description != "" {
		form.Set("description", description)
	}
	flow := strings.TrimSpace(req.Flow)
	if flow != "" {
		form.Set("metadata[mergeos_flow]", flow)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.stripe.com/v1/payment_intents", strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+strings.TrimSpace(p.cfg.StripeSecretKey))
	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	httpReq.Header.Set("Idempotency-Key", fmt.Sprintf("mergeos-card-intent-%d-%d", req.AmountCents, time.Now().UnixNano()))

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("stripe payment intent create failed: %s", readBody(resp.Body))
	}

	var decoded struct {
		ID           string `json:"id"`
		ClientSecret string `json:"client_secret"`
		Status       string `json:"status"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return nil, err
	}
	if decoded.ID == "" {
		return nil, errors.New("stripe returned an empty PaymentIntent id")
	}
	return &CreateCardPaymentIntentResponse{
		PaymentReference: decoded.ID,
		PaymentIntentID:  decoded.ID,
		ClientSecret:     decoded.ClientSecret,
		Status:           decoded.Status,
		Provider:         "stripe",
		Mode:             "stripe-payment-intent",
		PublicKey:        p.cfg.StripePublishableKey,
	}, nil
}

func (p *PaymentManager) verifyDev(reference, provider string) (PaymentVerification, error) {
	if !p.cfg.DevPaymentEnabled {
		return PaymentVerification{}, errors.New("dev payment verifier is disabled")
	}
	if strings.TrimSpace(reference) != p.cfg.DevPaymentCode {
		return PaymentVerification{}, fmt.Errorf("local verifier requires payment reference %q", p.cfg.DevPaymentCode)
	}
	return PaymentVerification{
		Provider:  provider,
		Reference: reference,
	}, nil
}

func (p *PaymentManager) verifyPayPal(ctx context.Context, orderID string, expectedCents int64) (PaymentVerification, error) {
	if strings.TrimSpace(orderID) == "" {
		return PaymentVerification{}, errors.New("paypal order id is required")
	}

	token, err := p.payPalAccessToken(ctx)
	if err != nil {
		return PaymentVerification{}, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.payPalBaseURL()+"/v2/checkout/orders/"+url.PathEscape(orderID)+"/capture", nil)
	if err != nil {
		return PaymentVerification{}, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+token)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("PayPal-Request-Id", "mergeos-capture-"+orderID)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return PaymentVerification{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return PaymentVerification{}, fmt.Errorf("paypal capture failed: %s", readBody(resp.Body))
	}

	var decoded struct {
		Status        string `json:"status"`
		PurchaseUnits []struct {
			Payments struct {
				Captures []struct {
					Status string `json:"status"`
					Amount struct {
						CurrencyCode string `json:"currency_code"`
						Value        string `json:"value"`
					} `json:"amount"`
				} `json:"captures"`
			} `json:"payments"`
		} `json:"purchase_units"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return PaymentVerification{}, err
	}
	if decoded.Status != "COMPLETED" {
		return PaymentVerification{}, fmt.Errorf("paypal order is %s, not COMPLETED", decoded.Status)
	}
	if len(decoded.PurchaseUnits) == 0 || len(decoded.PurchaseUnits[0].Payments.Captures) == 0 {
		return PaymentVerification{}, errors.New("paypal capture response has no capture amount")
	}
	capture := decoded.PurchaseUnits[0].Payments.Captures[0]
	if capture.Status != "COMPLETED" {
		return PaymentVerification{}, fmt.Errorf("paypal capture is %s, not COMPLETED", capture.Status)
	}
	if capture.Amount.CurrencyCode != "USD" {
		return PaymentVerification{}, fmt.Errorf("paypal currency %s is not USD", capture.Amount.CurrencyCode)
	}
	cents, err := payPalValueToCents(capture.Amount.Value)
	if err != nil {
		return PaymentVerification{}, err
	}
	if cents != expectedCents {
		return PaymentVerification{}, fmt.Errorf("paypal amount mismatch: got %s, expected %s", capture.Amount.Value, centsToPayPalValue(expectedCents))
	}
	return PaymentVerification{
		Provider:  "paypal",
		Reference: orderID,
	}, nil
}

func (p *PaymentManager) verifyStripe(ctx context.Context, paymentIntentID string, expectedCents int64) (PaymentVerification, error) {
	paymentIntentID = strings.TrimSpace(paymentIntentID)
	if paymentIntentID == "" {
		return PaymentVerification{}, errors.New("stripe payment intent id is required")
	}
	if !strings.HasPrefix(paymentIntentID, "pi_") {
		return PaymentVerification{}, errors.New("stripe payment reference must be a PaymentIntent id")
	}
	endpoint := "https://api.stripe.com/v1/payment_intents/" + url.PathEscape(paymentIntentID)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return PaymentVerification{}, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+strings.TrimSpace(p.cfg.StripeSecretKey))

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return PaymentVerification{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return PaymentVerification{}, fmt.Errorf("stripe payment intent lookup failed: %s", readBody(resp.Body))
	}

	var decoded struct {
		ID             string `json:"id"`
		Status         string `json:"status"`
		Currency       string `json:"currency"`
		Amount         int64  `json:"amount"`
		AmountReceived int64  `json:"amount_received"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return PaymentVerification{}, err
	}
	if decoded.ID != paymentIntentID {
		return PaymentVerification{}, fmt.Errorf("stripe payment intent mismatch: got %s", decoded.ID)
	}
	if decoded.Status != "succeeded" {
		return PaymentVerification{}, fmt.Errorf("stripe payment intent is %s, not succeeded", decoded.Status)
	}
	if strings.ToLower(strings.TrimSpace(decoded.Currency)) != "usd" {
		return PaymentVerification{}, fmt.Errorf("stripe currency %s is not USD", decoded.Currency)
	}
	if decoded.AmountReceived != expectedCents {
		return PaymentVerification{}, fmt.Errorf("stripe amount mismatch: got %d cents, expected %d cents", decoded.AmountReceived, expectedCents)
	}
	if decoded.Amount > 0 && decoded.Amount != expectedCents {
		return PaymentVerification{}, fmt.Errorf("stripe intent amount mismatch: got %d cents, expected %d cents", decoded.Amount, expectedCents)
	}
	return PaymentVerification{
		Provider:  "stripe",
		Reference: paymentIntentID,
	}, nil
}

func (p *PaymentManager) payPalAccessToken(ctx context.Context) (string, error) {
	form := strings.NewReader("grant_type=client_credentials")
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.payPalBaseURL()+"/v1/oauth2/token", form)
	if err != nil {
		return "", err
	}
	httpReq.SetBasicAuth(p.cfg.PayPalClientID, p.cfg.PayPalClientSecret)
	httpReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("paypal auth failed: %s", readBody(resp.Body))
	}

	var decoded struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return "", err
	}
	if decoded.AccessToken == "" {
		return "", errors.New("paypal returned empty access token")
	}
	return decoded.AccessToken, nil
}

func (p *PaymentManager) payPalBaseURL() string {
	if strings.HasPrefix(strings.TrimSpace(p.cfg.PayPalEnvironment), "http://") || strings.HasPrefix(strings.TrimSpace(p.cfg.PayPalEnvironment), "https://") {
		return strings.TrimRight(strings.TrimSpace(p.cfg.PayPalEnvironment), "/")
	}
	if p.cfg.PayPalEnvironment == "live" {
		return "https://api-m.paypal.com"
	}
	return "https://api-m.sandbox.paypal.com"
}

func (p *PaymentManager) verifyPayPalWebhookSignature(ctx context.Context, headers http.Header, event paypalWebhookEvent) (bool, error) {
	webhookID := strings.TrimSpace(p.cfg.PayPalWebhookID)
	if webhookID == "" {
		if p.cfg.Environment != "production" {
			return true, nil
		}
		return false, errors.New("PAYPAL_WEBHOOK_ID is required in production")
	}
	token, err := p.payPalAccessToken(ctx)
	if err != nil {
		return false, err
	}
	body := map[string]any{
		"auth_algo":         strings.TrimSpace(headers.Get("PayPal-Auth-Algo")),
		"cert_url":          strings.TrimSpace(headers.Get("PayPal-Cert-Url")),
		"transmission_id":   strings.TrimSpace(headers.Get("PayPal-Transmission-Id")),
		"transmission_sig":  strings.TrimSpace(headers.Get("PayPal-Transmission-Sig")),
		"transmission_time": strings.TrimSpace(headers.Get("PayPal-Transmission-Time")),
		"webhook_id":        webhookID,
		"webhook_event":     event,
	}
	for _, key := range []string{"auth_algo", "cert_url", "transmission_id", "transmission_sig", "transmission_time"} {
		if strings.TrimSpace(fmt.Sprint(body[key])) == "" {
			return false, errors.New("missing PayPal webhook signature headers")
		}
	}
	var payload bytes.Buffer
	if err := json.NewEncoder(&payload).Encode(body); err != nil {
		return false, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.payPalBaseURL()+"/v1/notifications/verify-webhook-signature", &payload)
	if err != nil {
		return false, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+token)
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := p.client.Do(httpReq)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return false, fmt.Errorf("paypal webhook signature verification failed: %s", readBody(resp.Body))
	}
	var decoded struct {
		VerificationStatus string `json:"verification_status"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return false, err
	}
	return strings.EqualFold(strings.TrimSpace(decoded.VerificationStatus), "SUCCESS"), nil
}

func (p *PaymentManager) verifyCrypto(ctx context.Context, reference string, expectedCents int64) (PaymentVerification, error) {
	signature := normalizeSolanaSignature(reference)
	if !validSolanaSignature(signature) {
		return PaymentVerification{}, errors.New("crypto payment reference must be a Solana transaction signature")
	}
	if p.cfg.CryptoMinConfirmations > 0 {
		if err := p.verifySolanaSignatureStatus(ctx, signature); err != nil {
			return PaymentVerification{}, err
		}
	}
	if err := p.verifySolanaSPLTransaction(ctx, signature, expectedCents); err != nil {
		return PaymentVerification{}, err
	}

	return PaymentVerification{
		Provider:  "solana-spl",
		Reference: signature,
	}, nil
}

func normalizeSolanaSignature(value string) string {
	value = strings.TrimSpace(value)
	value = trimAddressPrefix(value, "solana:")
	value = trimAddressPrefix(value, "sol:")
	return strings.TrimSpace(value)
}

func validSolanaSignature(value string) bool {
	decoded, ok := base58Decode(normalizeSolanaSignature(value))
	return ok && len(decoded) == 64
}

func (p *PaymentManager) verifySolanaSPLTransaction(ctx context.Context, signature string, expectedCents int64) error {
	var tx solanaTransaction
	if err := p.rpcCall(ctx, "getTransaction", []any{
		signature,
		map[string]any{
			"encoding":                       "jsonParsed",
			"commitment":                     "confirmed",
			"maxSupportedTransactionVersion": 0,
		},
	}, &tx); err != nil {
		return err
	}
	if tx.Meta == nil {
		return errors.New("solana transaction metadata is missing")
	}
	if len(tx.Meta.Err) > 0 && string(tx.Meta.Err) != "null" {
		return errors.New("solana transaction is not successful")
	}
	required := tokenUnitsForCents(expectedCents, p.cfg.CryptoTokenDecimals)
	if required.Sign() <= 0 {
		return errors.New("crypto payment amount must be positive")
	}
	mint := normalizeWalletAddress(p.cfg.CryptoTokenContract)
	receiver := normalizeWalletAddress(p.cfg.CryptoReceiver)
	if !validWalletAddress(mint) {
		return errors.New("CRYPTO_TOKEN_MINT must be a Solana mint address")
	}
	if !validWalletAddress(receiver) {
		return errors.New("CRYPTO_RECEIVER must be a Solana wallet or token account")
	}
	if solanaInstructionTransfers(tx.Transaction.Message.Instructions, mint, receiver, required) {
		return nil
	}
	for _, inner := range tx.Meta.InnerInstructions {
		if solanaInstructionTransfers(inner.Instructions, mint, receiver, required) {
			return nil
		}
	}
	if solanaTokenBalanceDelta(tx, mint, receiver).Cmp(required) >= 0 {
		return nil
	}
	return errors.New("spl transfer to configured receiver with required amount was not found")
}

func (p *PaymentManager) verifySolanaSignatureStatus(ctx context.Context, signature string) error {
	var statuses solanaSignatureStatuses
	if err := p.rpcCall(ctx, "getSignatureStatuses", []any{
		[]string{signature},
		map[string]any{"searchTransactionHistory": true},
	}, &statuses); err != nil {
		return err
	}
	if len(statuses.Value) == 0 || statuses.Value[0] == nil {
		return errors.New("solana signature was not found")
	}
	status := statuses.Value[0]
	if len(status.Err) > 0 && string(status.Err) != "null" {
		return errors.New("solana transaction is not successful")
	}
	if status.Confirmations == nil {
		return nil
	}
	if *status.Confirmations < p.cfg.CryptoMinConfirmations {
		return fmt.Errorf("solana transaction has %d confirmations, need %d", *status.Confirmations, p.cfg.CryptoMinConfirmations)
	}
	return nil
}

func solanaInstructionTransfers(instructions []solanaInstruction, mint, receiver string, required *big.Int) bool {
	for _, instruction := range instructions {
		if !strings.EqualFold(instruction.Program, "spl-token") && !strings.EqualFold(instruction.ProgramID, "TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA") {
			continue
		}
		parsed := instruction.parsedInfo()
		if parsed == nil {
			continue
		}
		instructionMint := normalizeWalletAddress(parsed.String("mint"))
		destination := normalizeWalletAddress(parsed.String("destination"))
		if instructionMint == "" || instructionMint != mint {
			continue
		}
		if destination != receiver && normalizeWalletAddress(parsed.String("account")) != receiver && normalizeWalletAddress(parsed.String("to")) != receiver {
			continue
		}
		amount := parsed.TokenAmount()
		if amount.Cmp(required) >= 0 {
			return true
		}
	}
	return false
}

func solanaTokenBalanceDelta(tx solanaTransaction, mint, receiver string) *big.Int {
	if tx.Meta == nil {
		return big.NewInt(0)
	}
	preBalances := map[int]*big.Int{}
	for _, balance := range tx.Meta.PreTokenBalances {
		if normalizeWalletAddress(balance.Mint) != mint {
			continue
		}
		preBalances[balance.AccountIndex] = balance.UIAmount.AmountInt()
	}
	largest := big.NewInt(0)
	for _, balance := range tx.Meta.PostTokenBalances {
		if normalizeWalletAddress(balance.Mint) != mint {
			continue
		}
		account := normalizeWalletAddress(tx.Transaction.Message.AccountKey(balance.AccountIndex))
		owner := normalizeWalletAddress(balance.Owner)
		if account != receiver && owner != receiver {
			continue
		}
		post := balance.UIAmount.AmountInt()
		pre := preBalances[balance.AccountIndex]
		if pre == nil {
			pre = big.NewInt(0)
		}
		delta := new(big.Int).Sub(post, pre)
		if delta.Cmp(largest) > 0 {
			largest = delta
		}
	}
	return largest
}

func tokenUnitsForCents(expectedCents int64, decimals int) *big.Int {
	if decimals < 0 {
		decimals = 0
	}
	required := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil)
	required.Mul(required, big.NewInt(expectedCents))
	required.Div(required, big.NewInt(100))
	return required
}

func (p *PaymentManager) rpcCall(ctx context.Context, method string, params []any, out any) error {
	body := map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  method,
		"params":  params,
	}
	var payload bytes.Buffer
	if err := json.NewEncoder(&payload).Encode(body); err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.cfg.CryptoRPCURL, &payload)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("rpc call failed: %s", readBody(resp.Body))
	}

	var decoded struct {
		Result json.RawMessage `json:"result"`
		Error  *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return err
	}
	if decoded.Error != nil {
		return fmt.Errorf("rpc error %d: %s", decoded.Error.Code, decoded.Error.Message)
	}
	if string(decoded.Result) == "null" || len(decoded.Result) == 0 {
		return fmt.Errorf("rpc method %s returned null", method)
	}
	return json.Unmarshal(decoded.Result, out)
}

type solanaTransaction struct {
	Slot int64 `json:"slot"`
	Meta *struct {
		Err               json.RawMessage           `json:"err"`
		PreTokenBalances  []solanaTokenBalance      `json:"preTokenBalances"`
		PostTokenBalances []solanaTokenBalance      `json:"postTokenBalances"`
		InnerInstructions []solanaInnerInstructions `json:"innerInstructions"`
	} `json:"meta"`
	Transaction struct {
		Message solanaMessage `json:"message"`
	} `json:"transaction"`
}

type solanaMessage struct {
	AccountKeys  []solanaAccountKey  `json:"accountKeys"`
	Instructions []solanaInstruction `json:"instructions"`
}

func (m solanaMessage) AccountKey(index int) string {
	if index < 0 || index >= len(m.AccountKeys) {
		return ""
	}
	return m.AccountKeys[index].Pubkey
}

type solanaAccountKey struct {
	Pubkey string `json:"pubkey"`
}

func (k *solanaAccountKey) UnmarshalJSON(data []byte) error {
	var object struct {
		Pubkey string `json:"pubkey"`
	}
	if err := json.Unmarshal(data, &object); err == nil && object.Pubkey != "" {
		k.Pubkey = object.Pubkey
		return nil
	}
	var pubkey string
	if err := json.Unmarshal(data, &pubkey); err != nil {
		return err
	}
	k.Pubkey = pubkey
	return nil
}

type solanaInstruction struct {
	Program   string          `json:"program"`
	ProgramID string          `json:"programId"`
	Parsed    json.RawMessage `json:"parsed"`
}

func (i solanaInstruction) parsedInfo() solanaParsedInfo {
	if len(i.Parsed) == 0 || string(i.Parsed) == "null" {
		return nil
	}
	var parsed struct {
		Type string         `json:"type"`
		Info map[string]any `json:"info"`
	}
	if err := json.Unmarshal(i.Parsed, &parsed); err != nil || parsed.Info == nil {
		return nil
	}
	return solanaParsedInfo(parsed.Info)
}

type solanaParsedInfo map[string]any

func (i solanaParsedInfo) String(key string) string {
	value, _ := i[key].(string)
	return strings.TrimSpace(value)
}

func (i solanaParsedInfo) TokenAmount() *big.Int {
	if tokenAmount, ok := i["tokenAmount"].(map[string]any); ok {
		if amount, ok := tokenAmount["amount"].(string); ok {
			if parsed, ok := new(big.Int).SetString(amount, 10); ok {
				return parsed
			}
		}
	}
	if amount, ok := i["amount"].(string); ok {
		if parsed, ok := new(big.Int).SetString(amount, 10); ok {
			return parsed
		}
	}
	return big.NewInt(0)
}

type solanaInnerInstructions struct {
	Index        int                 `json:"index"`
	Instructions []solanaInstruction `json:"instructions"`
}

type solanaTokenBalance struct {
	AccountIndex int                 `json:"accountIndex"`
	Mint         string              `json:"mint"`
	Owner        string              `json:"owner"`
	UIAmount     solanaUITokenAmount `json:"uiTokenAmount"`
}

type solanaUITokenAmount struct {
	Amount string `json:"amount"`
}

func (a solanaUITokenAmount) AmountInt() *big.Int {
	if parsed, ok := new(big.Int).SetString(strings.TrimSpace(a.Amount), 10); ok {
		return parsed
	}
	return big.NewInt(0)
}

type solanaSignatureStatuses struct {
	Value []*solanaSignatureStatus `json:"value"`
}

type solanaSignatureStatus struct {
	Confirmations *int64          `json:"confirmations"`
	Confirmation  string          `json:"confirmationStatus"`
	Err           json.RawMessage `json:"err"`
}

func centsToPayPalValue(cents int64) string {
	return fmt.Sprintf("%d.%02d", cents/100, cents%100)
}

func payPalValueToCents(value string) (int64, error) {
	parts := strings.Split(value, ".")
	if len(parts) == 0 || len(parts) > 2 {
		return 0, fmt.Errorf("invalid paypal amount %q", value)
	}
	dollars := new(big.Int)
	if _, ok := dollars.SetString(parts[0], 10); !ok {
		return 0, fmt.Errorf("invalid paypal amount %q", value)
	}
	dollars.Mul(dollars, big.NewInt(100))
	cents := int64(0)
	if len(parts) == 2 {
		fraction := parts[1]
		if len(fraction) == 1 {
			fraction += "0"
		}
		if len(fraction) > 2 {
			return 0, fmt.Errorf("paypal amount %q has more than two decimals", value)
		}
		parsed, ok := new(big.Int).SetString(fraction, 10)
		if !ok {
			return 0, fmt.Errorf("invalid paypal amount %q", value)
		}
		cents = parsed.Int64()
	}
	dollars.Add(dollars, big.NewInt(cents))
	if !dollars.IsInt64() {
		return 0, errors.New("paypal amount is too large")
	}
	return dollars.Int64(), nil
}

func readBody(body io.Reader) string {
	bytes, _ := io.ReadAll(io.LimitReader(body, 4096))
	return string(bytes)
}
