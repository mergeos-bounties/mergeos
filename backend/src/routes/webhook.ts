// backend/src/routes/webhook.ts
import { Router, Request, Response } from 'express';
import https from 'https';
import { URL } from 'url';
import { paypalConfig } from '../config/paypal';

const router = Router();

/**
 * Verify PayPal webhook signature using PayPal's Verify Webhook Signature API.
 * https://developer.paypal.com/docs/api/webhooks/v1/#webhooks_verify-webhook-signature
 */
function verifyPayPalSignature(req: Request): Promise<boolean> {
  return new Promise((resolve) => {
    const auth = Buffer.from(
      `${paypalConfig.clientId}:${paypalConfig.clientSecret}`
    ).toString('base64');

    const payload = JSON.stringify({
      auth_algo: req.headers['paypal-auth-algo'],
      cert_url: req.headers['paypal-cert-url'],
      transmission_id: req.headers['paypal-transmission-id'],
      transmission_sig: req.headers['paypal-transmission-sig'],
      transmission_time: req.headers['paypal-transmission-time'],
      webhook_id: process.env.PAYPAL_WEBHOOK_ID || '',
      webhook_event: req.body,
    });

    const apiUrl = `${paypalConfig.baseUrl}/v1/notifications/verify-webhook-signature`;
    const urlObj = new URL(apiUrl);

    const options = {
      hostname: urlObj.hostname,
      path: urlObj.pathname,
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Content-Length': Buffer.byteLength(payload),
        Authorization: `Basic ${auth}`,
      },
    };

    const request = https.request(options, (response) => {
      let data = '';
      response.on('data', (chunk) => { data += chunk; });
      response.on('end', () => {
        try {
          const result = JSON.parse(data);
          resolve(result.verification_status === 'SUCCESS');
        } catch {
          resolve(false);
        }
      });
    });

    request.on('error', () => resolve(false));
    request.write(payload);
    request.end();
  });
}

router.post('/paypal', async (req: Request, res: Response) => {
  try {
    const { event_type, resource } = req.body;

    // Step 1: Check signature header exists
    const signature = req.headers['paypal-transmission-sig'];
    if (!signature) {
      return res.status(400).json({ error: 'Missing PayPal signature' });
    }

    // Step 2: Verify signature via PayPal API
    const isValid = await verifyPayPalSignature(req);
    if (!isValid) {
      console.warn('Invalid webhook signature rejected');
      return res.status(403).json({ error: 'Invalid webhook signature' });
    }

    // Step 3: Process verified payment events
    switch (event_type) {
      case 'PAYMENT.CAPTURE.COMPLETED': {
        const projectId = resource?.custom_id;
        const amount = resource?.amount?.value;
        const currency = resource?.amount?.currency_code;

        if (!projectId) {
          return res.status(400).json({ error: 'Missing project ID in webhook' });
        }

        // TODO: Update project payment status in DB
        console.log(`Payment confirmed for project ${projectId}: ${amount} ${currency}`);
        break;
      }

      case 'PAYMENT.CAPTURE.DENIED':
      case 'PAYMENT.CAPTURE.DECLINED':
        console.log(`Payment denied for webhook ${resource?.id}`);
        break;

      case 'PAYMENT.CAPTURE.REFUNDED':
        console.log(`Payment refunded for webhook ${resource?.id}`);
        break;

      default:
        console.log(`Unhandled PayPal event: ${event_type}`);
    }

    res.status(200).json({ received: true });
  } catch (error) {
    console.error('Webhook processing error:', error);
    res.status(500).json({ error: 'Webhook processing failed' });
  }
});

export default router;
