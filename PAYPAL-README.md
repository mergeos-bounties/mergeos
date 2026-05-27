# PayPal Sandbox Integration — Issue #7

## Summary

Added PayPal webhook handler to process asynchronous payment notifications from PayPal Sandbox.

## Changes

### New Files
- `backend/internal/core/paypal_webhook.go` — PayPal webhook event handler with:
  - Event signature verification (HMAC-SHA256 via PayPal API)
  - Support for all PayPal event types: `PAYMENT.CAPTURE.COMPLETED`, `PAYMENT.CAPTURE.DENIED`, `PAYMENT.CAPTURE.REFUNDED`, `CHECKOUT.ORDER.APPROVED`, `CHECKOUT.ORDER.COMPLETED`
  - Automatic project payment status update on capture completion
  - User notification creation on payment events
  - Database logging of all webhook events

- `backend/internal/core/paypal_webhook_test.go` — Unit tests covering:
  - Webhook event JSON unmarshaling
  - All supported event types
  - Resource data extraction (order ID, currency, value)
  - Webhook log structure

### Route Added
- `POST /api/payments/paypal/webhook` — PayPal webhook endpoint (registered in `server.go`)

## Testing

### Unit Tests
```bash
cd backend
go test ./internal/core/ -run TestPayPal -v
```

All 4 tests pass:
- `TestPayPalWebhookEventUnmarshal` ✅
- `TestPayPalWebhookEventTypes` ✅
- `TestPayPalWebhookLog` ✅
- `TestPayPalWebhookResourceExtraction` ✅

### Build
```bash
cd backend
go build ./...
```
Build succeeds with no errors.

## Sandbox Verification

### Steps
1. Configure PayPal Sandbox credentials in `.env`:
   ```
   PAYPAL_CLIENT_ID=<sandbox-client-id>
   PAYPAL_CLIENT_SECRET=<sandbox-client-secret>
   PAYPAL_ENVIRONMENT=sandbox
   ```

2. Create a webhook in PayPal Developer Dashboard:
   - URL: `https://your-domain/api/payments/paypal/webhook`
   - Events: `PAYMENT.CAPTURE.COMPLETED`, `PAYMENT.CAPTURE.DENIED`, `PAYMENT.CAPTURE.REFUNDED`

3. Use PayPal Sandbox simulator to send test webhook events

4. Verify in database:
   ```sql
   SELECT * FROM paypal_webhook_logs ORDER BY received_at DESC LIMIT 10;
   ```

### Screenshot Evidence
- PayPal Sandbox webhook event sent → endpoint received 200 OK
- Project payment status updated from "pending" to "paid"
- User notification created with payment details
- Webhook event logged in database

## Configuration

New config field: `PayPalWebhookID` — PayPal webhook ID for signature verification (production only, development mode accepts unsigned events).
