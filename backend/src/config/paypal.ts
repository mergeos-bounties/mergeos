// backend/src/config/paypal.ts

/**
 * PayPal Sandbox Configuration
 * 
 * Setup instructions:
 * 1. Go to https://developer.paypal.com/dashboard/applications/sandbox
 * 2. Create a sandbox business account
 * 3. Generate REST API credentials (Client ID + Secret)
 * 4. Set the environment variables below
 */
export const paypalConfig = {
  // Sandbox mode - set to false for production
  sandbox: process.env.PAYPAL_SANDBOX === 'true' || process.env.NODE_ENV !== 'production',

  // API credentials
  clientId: process.env.PAYPAL_CLIENT_ID || '',
  clientSecret: process.env.PAYPAL_CLIENT_SECRET || '',

  // Webhook configuration
  webhookUrl: process.env.PAYPAL_WEBHOOK_URL || 'https://localhost:3000/api/webhook/paypal',

  // API base URLs
  baseUrl: process.env.PAYPAL_SANDBOX === 'true'
    ? 'https://api-m.sandbox.paypal.com'
    : 'https://api-m.paypal.com',

  // Payment capture settings
  currency: process.env.PAYPAL_CURRENCY || 'USD',
  returnUrl: process.env.PAYPAL_RETURN_URL || 'http://localhost:5173/payment/success',
  cancelUrl: process.env.PAYPAL_CANCEL_URL || 'http://localhost:5173/payment/cancel',
};

/**
 * Validate that PayPal configuration is properly set
 */
export function validatePayPalConfig(): boolean {
  if (!paypalConfig.clientId || !paypalConfig.clientSecret) {
    console.warn('PayPal credentials not configured. Set PAYPAL_CLIENT_ID and PAYPAL_CLIENT_SECRET.');
    return false;
  }
  return true;
}
