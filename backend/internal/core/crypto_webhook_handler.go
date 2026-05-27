package core

import (
	"encoding/json"
	"fmt"
	"net/http"
)

func (s *Server) createCryptoPayment(w http.ResponseWriter, r *http.Request) {
	user, ok := s.requireUser(w, r)
	if !ok {
		return
	}

	var req struct {
		Provider    string `json:"provider"`
		Title       string `json:"title"`
		Currency    string `json:"currency"`
		AmountCents int64  `json:"amountCents"`
		CallbackURL string `json:"callbackUrl"`
		CancelURL   string `json:"cancelUrl"`
		SuccessURL  string `json:"successUrl"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	if req.Provider == "" {
		req.Provider = "nowpayments"
	}
	if req.Currency == "" {
		req.Currency = "USDT"
	}
	if req.AmountCents < 100 {
		writeError(w, http.StatusBadRequest, "amount must be at least 100 cents")
		return
	}

	invoiceReq := CryptoInvoiceRequest{
		OrderID:     req.Title + "-" + user.ID,
		Title:       req.Title,
		AmountCents: req.AmountCents,
		Currency:    req.Currency,
		CallbackURL: req.CallbackURL,
		CancelURL:   req.CancelURL,
		SuccessURL:  req.SuccessURL,
	}

	invoice, err := s.payments.CreateCryptoInvoice(r.Context(), invoiceReq, req.Provider)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, invoice)
}

func (s *Server) verifyCryptoPaymentUpdate(update *CryptoPaymentUpdate) error {
	if s.store.IsPaymentReferenceUsed(update.InvoiceID) {
		return nil
	}

	statusMsg := fmt.Sprintf("USDT payment %s for invoice %s: %s (tx: %s)",
		update.Currency, update.InvoiceID, update.Status, update.TransactionID)
	s.store.addNotificationLocked("", "", "payment", statusMsg, "", update.Status)
	s.store.saveLocked()

	return nil
}
