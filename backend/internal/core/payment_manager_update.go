// ---------------------------------------------------------------
// Integration: PaymentManager + CryptoProvider
// Add these methods to the existing PaymentManager in payments.go
// ---------------------------------------------------------------

// CreateCryptoInvoice uses the configured crypto provider to create a payment
// invoice for the user.  Returns the invoice URL or address the buyer uses
// to send crypto.
func (p *PaymentManager) CreateCryptoInvoice(ctx context.Context, req CryptoInvoiceRequest, providerName string) (*CryptoInvoice, error) {
	provider := GetCryptoProvider(providerName)
	if provider == nil {
		return nil, fmt.Errorf("unknown crypto provider %q (available: %v)", providerName, ListCryptoProviders())
	}

	// Inject API key into context
	cfgMap := ProviderConfigFromEnv(p.cfg, providerName)
	apiKey := cfgMap["np_api_key"]
	if apiKey != "" {
		ctx = ContextWithAPIKey(ctx, providerName, apiKey)
	}

	return provider.CreateInvoice(ctx, req)
}

// VerifyCryptoWebhook delegates webhook verification to the named provider.
// providerName should match the gateway that sent the webhook (e.g. "nowpayments").
func (p *PaymentManager) VerifyCryptoWebhook(r *http.Request, body []byte, providerName string) (*CryptoPaymentUpdate, error) {
	provider := GetCryptoProvider(providerName)
	if provider == nil {
		return nil, fmt.Errorf("unknown crypto provider %q", providerName)
	}
	cfgMap := ProviderConfigFromEnv(p.cfg, providerName)
	return provider.VerifyWebhook(r, body, cfgMap)
}

// VerifyCryptoPaymentUpdate persists a payment update and updates project
// state.  This is called from the provider webhook handler.
func (s *Server) VerifyCryptoPaymentUpdate(update *CryptoPaymentUpdate) error {
	if s.store.IsPaymentReferenceUsed(update.InvoiceID) {
		return nil // idempotent — already processed
	}

	// Log to notification system
	statusMsg := fmt.Sprintf("USDT payment %s for invoice %s: %s (tx: %s)",
		update.Currency, update.InvoiceID, update.Status, update.TransactionID)
	s.store.addNotificationLocked("", "", "payment", statusMsg, update.RawPayload, update.Status)
	s.store.saveLocked()

	// Record in proof ledger (sanitized — no secrets)
	ledgerEntry := LedgerEntry{
		Provider:       "nowpayments",
		InvoiceID:      update.InvoiceID,
		TransactionID:  update.TransactionID,
		Status:         update.Status,
		AmountReceived: update.AmountReceived,
		Currency:       update.Currency,
		USDEquivalent:  update.USDEquivalent,
		ConfirmedAt:    update.ConfirmedAt,
		RawPayloadHash: sha256Hash(update.RawPayload), // store hash only, not raw payload
	}
	s.store.AddLedgerEntry(ledgerEntry)
	s.store.saveLocked()

	return nil
}

// sha256Hash returns the hex SHA-256 digest of a string.
func sha256Hash(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}
