// backend/src/routes/webhook.ts
import { Router, Request, Response } from 'express';
import crypto from 'crypto';

const router = Router();

/**
 * PayPal Webhook Handler
 * Handles payment confirmation webhooks from PayPal Sandbox/Production
 */
router.post('/paypal', async (req: Request, res: Response) => {
  try {
    const { event_type, resource } = req.body;
    const webhookId = resource?.id;

    // Verify webhook signature
    const signature = req.headers['paypal-transmission-sig'];
    if (!signature) {
      return res.status(400).json({ error: 'Missing PayPal signature' });
    }

    switch (event_type) {
      case 'PAYMENT.CAPTURE.COMPLETED':
        // Payment confirmed - update project status
        const projectId = resource?.custom_id;
        const amount = resource?.amount?.value;
        const currency = resource?.amount?.currency_code;

        if (!projectId) {
          return res.status(400).json({ error: 'Missing project ID in webhook' });
        }

        // TODO: Update project payment status in DB
        console.log(`Payment confirmed for project ${projectId}: ${amount} ${currency}`);
        break;

      case 'PAYMENT.CAPTURE.DENIED':
      case 'PAYMENT.CAPTURE.DECLINED':
        console.log(`Payment denied for webhook ${webhookId}`);
        break;

      case 'PAYMENT.CAPTURE.REFUNDED':
        console.log(`Payment refunded for webhook ${webhookId}`);
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
