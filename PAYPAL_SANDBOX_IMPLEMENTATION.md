# PayPal Sandbox Payment Flow - Implementation Guide

## Overview

This document describes the PayPal Sandbox payment flow implementation for MergeOS project funding. The implementation enables test customers to fund projects using PayPal Sandbox, with proper payment verification, MRG token minting, and ledger recording.

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                        Frontend (Vue.js)                        │
├─────────────────────────────────────────────────────────────────┤
│  PayPal Button → Create Order → Redirect → Capture → Verify     │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                      Backend (Go/Gin)                           │
├─────────────────────────────────────────────────────────────────┤
│  /api/payments/paypal/orders  →  Create PayPal Order            │
│  /api/payments/paypal/webhook →  Handle Webhook Events         │
│  /api/payments/paypal/capture →  Capture/Verify Payment        │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                    PayPal Sandbox API                           │
├─────────────────────────────────────────────────────────────────┤
│  POST /v2/checkout/orders        →  Create Order                │
│  POST /v2/checkout/orders/:id/capture  →  Capture Payment      │
│  POST /v1/notifications/verify-webhook-signature →  Verify     │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                     MergeOS Ledger                              │
├─────────────────────────────────────────────────────────────────┤
│  Record Payment → Mint MRG Tokens → Update Admin View           │
└─────────────────────────────────────────────────────────────────┘
```

## Configuration

### Environment Variables

```bash
# PayPal Sandbox Credentials
PAYPAL_SANDBOX_CLIENT_ID=your-client-id
PAYPAL_SANDBOX_CLIENT_SECRET=your-client-secret
PAYPAL_SANDBOX_WEBHOOK_ID=your-webhook-id
PAYPAL_SANDBOX_ENABLED=true

# PayPal API Base URL (for sandbox)
PAYPAL_BASE_URL=https://api-m.sandbox.paypal.com
```

### PayPal Developer Dashboard Setup

1. Go to [PayPal Developer Dashboard](https://developer.paypal.com/)
2. Create a Sandbox account
3. Create an App to get Client ID and Secret
4. Set up Webhooks for your app:
   - Event types: `PAYMENT.CAPTURE.COMPLETED`, `PAYMENT.CAPTURE.DENIED`, `PAYMENT.CAPTURE.REFUNDED`
   - Webhook URL: `https://your-domain.com/api/payments/paypal/webhook`

## Implementation Details

### 1. Create Order Flow

```go
// POST /api/payments/paypal/orders
func (s *Server) createPayPalOrder(w http.ResponseWriter, r *http.Request) {
    // 1. Validate request
    // 2. Create PayPal order via API
    // 3. Return approval URL for redirect
}
```

**Request:**
```json
{
  "amount_cents": 15000,
  "description": "Project funding",
  "return_url": "https://your-domain.com/paypal/return",
  "cancel_url": "https://your-domain.com/paypal/cancel"
}
```

**Response:**
```json
{
  "order_id": "5O190127TN364715T",
  "approval_url": "https://www.sandbox.paypal.com/checkoutnow?token=5O190127TN364715T",
  "status": "CREATED"
}
```

### 2. Capture Order Flow

```go
// POST /api/payments/paypal/capture
func (s *Server) capturePayPalOrder(w http.ResponseWriter, r *http.Request) {
    // 1. Get order ID from request
    // 2. Call PayPal Capture API
    // 3. Verify payment status
    // 4. Record in ledger
    // 5. Return verification result
}
```

**Request:**
```json
{
  "order_id": "5O190127TN364715T"
}
```

**Response:**
```json
{
  "verification": {
    "provider": "paypal-sandbox",
    "reference": "0V463155YN3267534"
  },
  "status": "COMPLETED",
  "amount": {
    "currency_code": "USD",
    "value": "150.00"
  }
}
```

### 3. Webhook Handler

```go
// POST /api/payments/paypal/webhook
func (s *Server) handlePayPalWebhook(w http.ResponseWriter, r *http.Request) {
    // 1. Verify webhook signature
    // 2. Parse event type
    // 3. Handle PAYMENT.CAPTURE.COMPLETED
    // 4. Record in ledger
    // 5. Return 200 OK
}
```

## Testing

### Unit Tests

Run unit tests:
```bash
cd backend
go test ./internal/core/... -v -run TestPayPalSandbox
```

### Integration Tests

1. Set up sandbox environment variables
2. Run integration test suite:
```bash
cd backend
go test ./internal/core/... -v -run TestPayPalSandboxIntegration
```

### Manual Testing

1. Start the backend server
2. Create a project with PayPal payment method
3. Complete the PayPal checkout flow
4. Verify payment appears in admin ledger
5. Verify public ledger shows sanitized payment proof

## Security Considerations

1. **Sandbox Only**: This implementation only uses PayPal Sandbox environment
2. **Webhook Verification**: All webhooks are verified using PayPal's signature verification API
3. **No Credentials in Code**: All credentials are loaded from environment variables
4. **HTTPS Required**: Production deployment must use HTTPS for webhook endpoints

## MRG Token Minting

When a payment is successfully captured:

1. Payment is recorded in the ledger
2. MRG tokens are minted based on the payment amount
3. Token balance is updated in the user's account
4. Transaction appears in the public ledger

## Admin View

The admin dashboard shows:
- All PayPal payments with status
- MRG tokens minted per payment
- Payment verification details
- Webhook event history

## Public Ledger

The public ledger shows:
- Sanitized payment proofs (no private user data)
- Payment amount and currency
- Timestamp
- Transaction reference (order ID only)

## Troubleshooting

### Common Issues

1. **"PayPal credentials not configured"**
   - Ensure environment variables are set correctly
   - Check that `PAYPAL_SANDBOX_ENABLED=true`

2. **"Invalid webhook signature"**
   - Verify webhook ID matches the one in PayPal Developer Dashboard
   - Ensure the webhook URL is correctly configured

3. **"Payment not appearing in ledger"**
   - Check webhook events in PayPal Developer Dashboard
   - Verify the webhook endpoint is accessible

## References

- [PayPal Sandbox Documentation](https://developer.paypal.com/tools/sandbox/)
- [PayPal Checkout Integration](https://developer.paypal.com/docs/checkout/)
- [PayPal Webhooks](https://developer.paypal.com/docs/api-basics/notifications/webhooks/)
- [MergeOS Bounty Policy](https://github.com/mergeos-bounties/mergeos/blob/master/BOUNTY-POLICY.md)
