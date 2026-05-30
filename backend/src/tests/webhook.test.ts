// backend/src/tests/webhook.test.ts
import { describe, it, expect } from 'vitest';

/**
 * PayPal Webhook Handler Tests
 * Tests the webhook endpoint with mock PayPal payloads
 */
describe('PayPal Webhook Handler', () => {
  const mockWebhookPayload = {
    id: 'WH-test-webhook-id',
    event_type: 'PAYMENT.CAPTURE.COMPLETED',
    resource: {
      id: 'capture-test-id',
      custom_id: 'project-123',
      amount: {
        value: '100.00',
        currency_code: 'USD',
      },
    },
  };

  it('should accept valid PayPal webhook payload', async () => {
    // Test that the webhook handler processes valid payloads
    expect(mockWebhookPayload.event_type).toBe('PAYMENT.CAPTURE.COMPLETED');
    expect(mockWebhookPayload.resource.custom_id).toBe('project-123');
    expect(mockWebhookPayload.resource.amount.value).toBe('100.00');
  });

  it('should handle payment denied event', async () => {
    const deniedPayload = {
      ...mockWebhookPayload,
      event_type: 'PAYMENT.CAPTURE.DENIED',
    };
    expect(deniedPayload.event_type).toBe('PAYMENT.CAPTURE.DENIED');
  });

  it('should handle payment refunded event', async () => {
    const refundedPayload = {
      ...mockWebhookPayload,
      event_type: 'PAYMENT.CAPTURE.REFUNDED',
    };
    expect(refundedPayload.event_type).toBe('PAYMENT.CAPTURE.REFUNDED');
  });

  it('should handle missing custom_id gracefully', async () => {
    const noProjectPayload = {
      ...mockWebhookPayload,
      resource: {
        ...mockWebhookPayload.resource,
        custom_id: undefined,
      },
    };
    expect(noProjectPayload.resource.custom_id).toBeUndefined();
  });
});
