# PayPal Sandbox Integration — Issue #7

## Summary

PayPal Sandbox payment flow for MergeOS project funding. Includes MRG token minting on confirmed payments, admin review tracking, and public proof ledger entries.

## Changes

### Files
- `backend/internal/core/paypal_webhook.go` — PayPal webhook handler:
  - Event signature verification (production via PayPal API)
  - MRG token minting on PAYMENT.CAPTURE.COMPLETED (80% of payment value minted as MRG credit)
  - Public ledger entries for payment verification and token minting
  - Admin review notifications
  - Duplicate order idempotency protection

- `backend/internal/core/paypal_webhook_test.go` — Unit tests
- `backend/internal/core/payments.go` — PayPal order creation (CreatePayPalOrder)
- `frontend/src/App.vue` — PayPal Sandbox checkout button in funding flow

## MRG Token Minting
- When `PAYMENT.CAPTURE.COMPLETED` is received, the system:
  1. Creates a `payment_verified` ledger entry
  2. Mints MRG tokens (80% of USD value) as a `token_mint` ledger entry
  3. Creates a notification for admin review

## Admin Review
- All PayPal payments appear in the admin ledger view
- Each payment creates a notification with subject "PayPal Payment Completed - MRG Minted"
- Ledger entries include the order ID and capture ID for audit

## Public Proof Ledger
- Payment verification and token mint entries are visible in the public ledger
- Sanitized: no customer secrets, PII, or raw PayPal payloads
- Each entry shows: sequence, type, amount, reference hash

## Testing
```bash
cd backend
go test ./internal/core/ -run TestPayPal -v
go build ./...
```

## Sandbox Verification
1. Configure PayPal Sandbox credentials in `.env.local`:
   ```
   PAYPAL_ENV=sandbox
   PAYPAL_CLIENT_ID=your-sandbox-client-id
   PAYPAL_CLIENT_SECRET=your-sandbox-secret
   ```
2. Create webhook in PayPal Developer Dashboard pointing to `/api/payments/paypal/webhook`
3. Use PayPal Sandbox simulator to send PAYMENT.CAPTURE.COMPLETED events
4. Verify: ledger entries created, MRG tokens minted, admin notifications received
5. Check public ledger at `/ledger` for sanitized proof