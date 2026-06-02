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
			verification, err := p.verifyCrypto(ctx, reference, req.BudgetCents)
			if err != nil {
				return PaymentVerification{}, err
			}
			verification.Provider = "usdt-" + verification.Provider
			return verification, nil
		}
		return p.verifyDev(reference, "dev-usdt")
	default:
		return PaymentVerification{}, errors.New("payment method must be paypal, crypto, or usdt")
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
	if p.cfg.PayPalEnvironment == "live" {
		return "https://api-m.paypal.com"
	}
	return "https://api-m.sandbox.paypal.com"
}

func (p *PaymentManager) verifyCrypto(ctx context.Context, txHash string, expectedCents int64) (PaymentVerification, error) {
	txHash = strings.TrimSpace(txHash)
	if !strings.HasPrefix(txHash, "0x") || len(txHash) != 66 {
		return PaymentVerification{}, errors.New("crypto payment reference must be a transaction hash")
	}

	var receipt evmReceipt
	if err := p.rpcCall(ctx, "eth_getTransactionReceipt", []any{txHash}, &receipt); err != nil {
		return PaymentVerification{}, err
	}
	if receipt.Status != "0x1" {
		return PaymentVerification{}, errors.New("crypto transaction is not successful")
	}
	if p.cfg.CryptoMinConfirmations > 0 {
		confirmations, err := p.confirmations(ctx, receipt.BlockNumber)
		if err != nil {
			return PaymentVerification{}, err
		}
		if confirmations < p.cfg.CryptoMinConfirmations {
			return PaymentVerification{}, fmt.Errorf("crypto transaction has %d confirmations, need %d", confirmations, p.cfg.CryptoMinConfirmations)
		}
	}

	switch p.cfg.CryptoAsset {
	case "erc20":
		if err := p.verifyERC20Receipt(receipt, expectedCents); err != nil {
			return PaymentVerification{}, err
		}
	default:
		if err := p.verifyNativePayment(ctx, txHash, expectedCents); err != nil {
			return PaymentVerification{}, err
		}
	}

	return PaymentVerification{
		Provider:  "evm-" + p.cfg.CryptoAsset,
		Reference: txHash,
	}, nil
}

func (p *PaymentManager) verifyNativePayment(ctx context.Context, txHash string, expectedCents int64) error {
	var tx evmTransaction
	if err := p.rpcCall(ctx, "eth_getTransactionByHash", []any{txHash}, &tx); err != nil {
		return err
	}
	if strings.ToLower(tx.To) != p.cfg.CryptoReceiver {
		return fmt.Errorf("crypto receiver mismatch: got %s", tx.To)
	}
	required := new(big.Int)
	if _, ok := required.SetString(p.cfg.CryptoWeiPerUSDCent, 10); !ok {
		return errors.New("CRYPTO_WEI_PER_USD_CENT must be a base-10 integer")
	}
	required.Mul(required, big.NewInt(expectedCents))
	value, err := hexBig(tx.Value)
	if err != nil {
		return err
	}
	if value.Cmp(required) < 0 {
		return fmt.Errorf("native payment too small: got %s wei, need %s wei", value.String(), required.String())
	}
	return nil
}

func (p *PaymentManager) verifyERC20Receipt(receipt evmReceipt, expectedCents int64) error {
	required := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(p.cfg.CryptoTokenDecimals)), nil)
	required.Mul(required, big.NewInt(expectedCents))
	required.Div(required, big.NewInt(100))

	receiver := strings.TrimPrefix(strings.ToLower(p.cfg.CryptoReceiver), "0x")
	token := strings.ToLower(p.cfg.CryptoTokenContract)
	for _, log := range receipt.Logs {
		if strings.ToLower(log.Address) != token || len(log.Topics) < 3 {
			continue
		}
		if strings.ToLower(log.Topics[0]) != erc20TransferTopic {
			continue
		}
		toTopic := strings.TrimPrefix(strings.ToLower(log.Topics[2]), "0x")
		if !strings.HasSuffix(toTopic, receiver) {
			continue
		}
		amount, err := hexBig(log.Data)
		if err != nil {
			return err
		}
		if amount.Cmp(required) >= 0 {
			return nil
		}
	}
	return errors.New("erc20 transfer to configured receiver with required amount was not found")
}

func (p *PaymentManager) confirmations(ctx context.Context, txBlockHex string) (int64, error) {
	txBlock, err := hexBig(txBlockHex)
	if err != nil {
		return 0, err
	}
	var latestHex string
	if err := p.rpcCall(ctx, "eth_blockNumber", []any{}, &latestHex); err != nil {
		return 0, err
	}
	latest, err := hexBig(latestHex)
	if err != nil {
		return 0, err
	}
	diff := new(big.Int).Sub(latest, txBlock)
	diff.Add(diff, big.NewInt(1))
	if !diff.IsInt64() {
		return 0, errors.New("confirmation count is too large")
	}
	return diff.Int64(), nil
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

type evmReceipt struct {
	Status      string   `json:"status"`
	BlockNumber string   `json:"blockNumber"`
	Logs        []evmLog `json:"logs"`
}

type evmLog struct {
	Address string   `json:"address"`
	Topics  []string `json:"topics"`
	Data    string   `json:"data"`
}

type evmTransaction struct {
	To    string `json:"to"`
	Value string `json:"value"`
}

const erc20TransferTopic = "0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"

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

func hexBig(value string) (*big.Int, error) {
	value = strings.TrimPrefix(strings.ToLower(strings.TrimSpace(value)), "0x")
	if value == "" {
		return big.NewInt(0), nil
	}
	parsed := new(big.Int)
	if _, ok := parsed.SetString(value, 16); !ok {
		return nil, fmt.Errorf("invalid hex integer %q", value)
	}
	return parsed, nil
}

func readBody(body io.Reader) string {
	bytes, _ := io.ReadAll(io.LimitReader(body, 4096))
	return string(bytes)
}
