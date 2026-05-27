# USDT Crypto Payment Gateway — Bounty #8

## Summary

Implements a **pluggable crypto payment provider abstraction** and an initial
**NowPayments.io USDT gateway** for MergeOS.  Customers can fund projects by
paying with USDT (ERC-20 / TRC-20 / BEP-20) through NowPayments; MergeOS
receives verified webhook callbacks and updates project payment state, admin
review records, and the public proof ledger.

## Files Changed

### New files

| File | Description |
|------|-------------|
| `backend/internal/core/crypto_provider.go` | `CryptoProvider` interface, `NowPaymentsProvider`, provider registry, helper functions |
| `backend/internal/core/crypto_provider_test.go` | Unit tests for provider interface, mock provider, NowPayments signature validation, config helper |

### Modified files

| File | Change |
|------|--------|
| `backend/internal/core/config.go` | Added `NPAPIKey`, `NPIPNSecret`, `NPSandbox` config fields |
| `backend/internal/core/payments.go` | Added `CreateCryptoInvoice`, `VerifyCryptoWebhook` methods on `PaymentManager` |
| `backend/internal/core/server.go` | Added `POST /api/payments/crypto/create` route; updated `cryptoWebhook` to use provider abstraction; added `VerifyCryptoPaymentUpdate` |
| `backend/.env.example` | Added NowPayments environment variables |

## Provider abstraction

```go
type CryptoProvider interface {
    Name() string
    CreateInvoice(ctx context.Context, req CryptoInvoiceRequest) (*CryptoInvoice, error)
    VerifyWebhook(r *http.Request, body []byte, providerCfg map[string]string) (*CryptoPaymentUpdate, error)
    VerifyOnChain(ctx context.Context, txHash string, expectedCents int64, providerCfg map[string]string) (*CryptoPaymentUpdate, error)
}
```

New providers can be added by implementing this interface and registering:

```go
RegisterCryptoProvider("coinbase-commerce", &CoinbaseCommerceProvider{...})
```

## NowPayments USDT gateway

The `NowPaymentsProvider` implements all three methods:

1. **CreateInvoice** — calls `POST /v1/invoice` on NowPayments API
2. **VerifyWebhook** — validates `x-nowpayments-sig` HMAC-SHA512 header, parses IPN payload, maps status codes
3. **VerifyOnChain** — returns nil (falls back to generic EVM verifier)

### Status mapping

| NowPayments status | MergeOS status |
|--------------------|---------------|
| `waiting` | `pending` |
| `confirming`, `sending` | `confirming` |
| `confirmed`, `finished` | `confirmed` |
| `partially_paid` | `confirmed` (needs admin review) |
| `failed` | `failed` |
| `refunded` | `refunded` |
| `expired` | `expired` |

## Environment variables (new)

```
# NowPayments
NP_API_KEY=npk_...
NP_IPN_SECRET=your-ipn-secret
NP_SANDBOX=true
```

## API Routes

| Method | Path | Description |
|--------|------|-------------|
| `POST` | `/api/payments/crypto/create` | Create a crypto payment invoice |
| `POST` | `/api/payments/crypto/webhook` | Receive verified payment callbacks (NowPayments IPN + legacy crypto) |

## Testing

```bash
cd backend
go test ./internal/core/ -run TestNowPayments -v
go test ./internal/core/ -run TestMockProvider -v
go test ./internal/core/ -run TestProvider -v
```

## Sandbox verification

1. Sign up at [nowpayments.io](https://nowpayments.io) and get API keys
2. Set `NP_API_KEY`, `NP_IPN_SECRET`, `NP_SANDBOX=true` in `.env.local`
3. Configure the IPN callback URL in NowPayments dashboard → `{app_url}/api/payments/crypto/webhook`
4. Run the app and create a test project (crypto payment)
5. Use NowPayments sandbox to send a test IPN notification
6. Verify admin payment view and ledger show the USDT payment
