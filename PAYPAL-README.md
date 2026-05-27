# PayPal Sandbox Integration — Issue #7

## Summary

Added PayPal webhook handler to process asynchronous payment notifications from PayPal Sandbox.

## Changes

### New Files
- `backend/internal/core/paypal_webhook.go` — PayPal webhook handler:
  - Event signature verification (production via PayPal API)
  - All event types: COMPLETED, DENIED, REFUNDED, ORDER.APPROVED, ORDER.COMPLETED
  - Auto project payment status update
  - User notification creation
  - Database logging

- `backend/internal/core/paypal_webhook_test.go` — 4 unit tests

### Modified Files
- `backend/internal/core/server.go` — Added webhook route

## Testing
```bash
cd backend
go test ./internal/core/ -run TestPayPal -v
go build ./...
```

## Sandbox Verification
1. Configure PayPal Sandbox credentials
2. Create webhook in PayPal Developer Dashboard
3. Use PayPal Sandbox simulator to send test events
4. Verify project payment status updates and notifications