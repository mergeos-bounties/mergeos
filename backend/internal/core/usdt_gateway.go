package core

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

type USDTGatewayProvider interface {
	Name() string
	CreateInvoice(ctx USDTInvoiceRequest) (*USDTInvoiceResponse, error)
	VerifyCallback(ctx USDTWebhookPayload, secret string) (bool, error)
}

type USDTInvoiceRequest struct {
	AmountUSDCents int64  `json:"amount_usd_cents"`
	OrderID        string `json:"order_id"`
	Description    string `json:"description"`
	CustomerEmail  string `json:"customer_email"`
	Network        string `json:"network"`
}

type USDTInvoiceResponse struct {
	InvoiceID    string `json:"invoice_id"`
	PayAddress   string `json:"pay_address"`
	PayAmount    string `json:"pay_amount"`
	PayCurrency  string `json:"pay_currency"`
	Network      string `json:"network"`
	ExpiresAt    string `json:"expires_at"`
	StatusURL    string `json:"status_url"`
	InvoiceURL   string `json:"invoice_url"`
}

type USDTWebhookPayload struct {
	EventType   string          `json:"event_type"`
	InvoiceID   string          `json:"invoice_id"`
	OrderID     string          `json:"order_id"`
	TxHash      string          `json:"tx_hash"`
	Status      string          `json:"status"`
	AmountPaid  string          `json:"amount_paid"`
	Currency    string          `json:"currency"`
	Network     string          `json:"network"`
	FromAddress string          `json:"from_address"`
	ToAddress   string          `json:"to_address"`
	Confirmations int           `json:"confirmations"`
	Raw         json.RawMessage `json:"raw,omitempty"`
}

const (
	USDTStatusPending   = "pending"
	USDTStatusConfirmed = "confirmed"
	USDTStatusExpired   = "expired"
	USDTStatusFailed    = "failed"
	USDTStatusRefunded  = "refunded"
)

type USDTMockProvider struct{}

func NewUSDTMockProvider() *USDTMockProvider {
	return &USDTMockProvider{}
}

func (m *USDTMockProvider) Name() string {
	return "usdt-mock"
}

func (m *USDTMockProvider) CreateInvoice(req USDTInvoiceRequest) (*USDTInvoiceResponse, error) {
	network := strings.TrimSpace(req.Network)
	if network == "" {
		network = "trc20"
	}
	return &USDTInvoiceResponse{
		InvoiceID:   "mock_usdt_" + time.Now().UTC().Format("20060102150405"),
		PayAddress:  "TXYZMockUSDTAddress123456789abcdef",
		PayAmount:   fmt.Sprintf("%.2f", float64(req.AmountUSDCents)/100.0),
		PayCurrency: "USDT",
		Network:     network,
		ExpiresAt:   time.Now().UTC().Add(30 * time.Minute).Format(time.RFC3339),
		StatusURL:   "https://mock.gateway/status",
		InvoiceURL:  "https://mock.gateway/invoice",
	}, nil
}

func (m *USDTMockProvider) VerifyCallback(payload USDTWebhookPayload, secret string) (bool, error) {
	return true, nil
}

type USDTCryptoWebhookRequest struct {
	UserID         string   `json:"userId"`
	Title          string   `json:"title"`
	ClientName     string   `json:"clientName"`
	CompanyName    string   `json:"companyName"`
	ClientEmail    string   `json:"clientEmail"`
	Phone          string   `json:"phone"`
	SiteType       string   `json:"siteType"`
	PackageTier    string   `json:"packageTier"`
	Timeline       string   `json:"timeline"`
	Brief          string   `json:"brief"`
	BudgetCents    int64    `json:"budgetCents"`
	AttachmentIDs  []string `json:"attachmentIds"`
	SourceRepoURL  string   `json:"sourceRepoURL"`
	TxHash         string   `json:"txHash"`
	Network        string   `json:"network"`
	Provider       string   `json:"provider"`
}

type USDTInvoiceCreateRequest struct {
	UserID      string `json:"userId"`
	AmountCents int64  `json:"amountCents"`
	Description string `json:"description"`
	Network     string `json:"network"`
}

func (s *Server) createUSDTInvoice(w http.ResponseWriter, r *http.Request) {
	user, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	var req USDTInvoiceCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.AmountCents < 10000 {
		writeError(w, http.StatusBadRequest, "amount must be at least 100 USD")
		return
	}
	provider := newUSDTProvider(s.cfg)
	invoiceReq := USDTInvoiceRequest{
		AmountUSDCents: req.AmountCents,
		OrderID:        "mrg_" + user.ID + "_" + fmt.Sprintf("%d", time.Now().UnixNano()),
		Description:    strings.TrimSpace(req.Description),
		CustomerEmail:  user.Email,
		Network:        strings.TrimSpace(req.Network),
	}
	if invoiceReq.Network == "" {
		invoiceReq.Network = "trc20"
	}
	invoice, err := provider.CreateInvoice(invoiceReq)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, invoice)
}

func newUSDTProvider(cfg Config) USDTGatewayProvider {
	return NewUSDTMockProvider()
}

func (s *Server) usdtWebhook(w http.ResponseWriter, r *http.Request) {
	bodyBytes, err := io.ReadAll(io.LimitReader(r.Body, 256*1024))
	defer r.Body.Close()
	if err != nil {
		log.Printf("[usdt-webhook] read error: %v", err)
		writeError(w, http.StatusBadRequest, "failed to read body")
		return
	}

	if s.cfg.CryptoWebhookSecret != "" {
		signatureHex := r.Header.Get("X-MergeOS-Signature")
		eventHeader := r.Header.Get("X-MergeOS-Event")
		if signatureHex == "" && eventHeader == "" {
			signatureHex = r.Header.Get("X-USDT-Signature")
		}
		if signatureHex != "" {
			expectedMac := hmac.New(sha256.New, []byte(s.cfg.CryptoWebhookSecret))
			expectedMac.Write(bodyBytes)
			expectedSignature := hex.EncodeToString(expectedMac.Sum(nil))
			if !hmac.Equal([]byte(signatureHex), []byte(expectedSignature)) {
				writeError(w, http.StatusUnauthorized, "invalid signature")
				return
			}
		}
	}

	var payload USDTWebhookPayload
	if err := json.Unmarshal(bodyBytes, &payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON payload")
		return
	}

	txHash := strings.TrimSpace(payload.TxHash)
	if txHash == "" {
		txHash = strings.TrimSpace(payload.InvoiceID)
	}
	if txHash == "" {
		writeError(w, http.StatusBadRequest, "transaction hash or invoice ID is required")
		return
	}

	if payload.Status == USDTStatusConfirmed {
		if s.store.IsPaymentReferenceUsed(txHash) {
			log.Printf("[usdt-webhook] duplicate callback ignored: %s", txHash)
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"status": "duplicate_ignored", "tx_hash": txHash})
			return
		}
		s.store.MarkPaymentReferenceUsed(txHash)
		s.store.addNotificationLocked("", "", "payment",
			"USDT Payment Confirmed",
			fmt.Sprintf("USDT payment confirmed: invoice %s, tx %s, amount %s %s", payload.InvoiceID, txHash, payload.AmountPaid, payload.Currency),
			"confirmed")
		s.store.saveLocked()
		log.Printf("[usdt-webhook] payment confirmed: invoice=%s tx=%s amount=%s", payload.InvoiceID, txHash, payload.AmountPaid)
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status":     "received",
		"invoice_id": payload.InvoiceID,
	})
}

func (s *Store) MarkPaymentReferenceUsed(reference string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.adminSettings.UsedPaymentReferences == nil {
		s.adminSettings.UsedPaymentReferences = map[string]bool{}
	}
	s.adminSettings.UsedPaymentReferences[reference] = true
	s.saveLocked()
}
