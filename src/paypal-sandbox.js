// PayPal Sandbox Payment Flow for MergeOS (#7, 1000 MRG)
const PAYPAL_SANDBOX='https://api-m.sandbox.paypal.com';
async function createOrder(amount,currency='USD'){
  const auth=Buffer.from(`${process.env.PAYPAL_CLIENT_ID}:${process.env.PAYPAL_SECRET}`).toString('base64');
  const token=await fetch(`${PAYPAL_SANDBOX}/v1/oauth2/token`,{method:'POST',headers:{Authorization:`Basic ${auth}`,'Content-Type':'application/x-www-form-urlencoded'},body:'grant_type=client_credentials'}).then(r=>r.json());
  const order=await fetch(`${PAYPAL_SANDBOX}/v2/checkout/orders`,{method:'POST',headers:{Authorization:`Bearer ${token.access_token}`,'Content-Type':'application/json'},body:JSON.stringify({intent:'CAPTURE',purchase_units:[{amount:{currency_code:currency,value:amount.toFixed(2)}}]})}).then(r=>r.json());
  return order;
}
async function captureOrder(orderId){
  const auth=Buffer.from(`${process.env.PAYPAL_CLIENT_ID}:${process.env.PAYPAL_SECRET}`).toString('base64');
  const token=await fetch(`${PAYPAL_SANDBOX}/v1/oauth2/token`,{method:'POST',headers:{Authorization:`Basic ${auth}`,'Content-Type':'application/x-www-form-urlencoded'},body:'grant_type=client_credentials'}).then(r=>r.json());
  return fetch(`${PAYPAL_SANDBOX}/v2/checkout/orders/${orderId}/capture`,{method:'POST',headers:{Authorization:`Bearer ${token.access_token}`,'Content-Type':'application/json'}}).then(r=>r.json());
}
module.exports={createOrder,captureOrder};
