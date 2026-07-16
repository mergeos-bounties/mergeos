package core

import (
	"errors"
	"fmt"
	"strings"
)

type usdtWebhookSettlement struct {
	Status      string       `json:"status"`
	TxHash      string       `json:"tx_hash"`
	ProjectID   string       `json:"project_id,omitempty"`
	AmountCents int64        `json:"amount_cents,omitempty"`
	LedgerEntry *LedgerEntry `json:"ledger_entry,omitempty"`
	Duplicate   bool         `json:"duplicate"`
	Message     string       `json:"message,omitempty"`
}

func (s *Store) RecordUSDTWebhookPayment(event USDTWebhookEvent) (usdtWebhookSettlement, error) {
	txHash := strings.TrimSpace(event.TxHash)
	if txHash == "" {
		return usdtWebhookSettlement{}, errors.New("usdt webhook tx_hash is required")
	}

	settlement := usdtWebhookSettlement{
		Status:  "unmatched",
		TxHash:  txHash,
		Message: "checking payment reference",
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Find project by PaymentReference
	var project *Project
	for _, p := range s.projects {
		if p.PaymentMethod == PaymentUSDT && p.PaymentReference == txHash {
			project = p
			break
		}
	}

	if project == nil {
		// No project found with this tx_hash as payment reference
		settlement.Message = "no project matched the usdt tx_hash"
		return settlement, nil
	}

	settlement.ProjectID = project.ID

	// Check if already settled
	if project.PaymentStatus == "verified" {
		settlement.Status = "duplicate"
		settlement.Duplicate = true
		settlement.Message = "usdt payment was already recorded"
		return settlement, nil
	}

	// Update Project
	project.PaymentStatus = "verified"
	project.PaymentProvider = "usdt_crypto"

	// Create Ledger Entry
	ledgerEntry := s.addLedger("payment_received", event.Sender, "platform_usdt", project.BudgetCents, fmt.Sprintf("USDT %s", txHash))

	if err := s.saveLocked(); err != nil {
		return usdtWebhookSettlement{}, fmt.Errorf("failed to save state: %w", err)
	}

	settlement.Status = "settled"
	settlement.LedgerEntry = &ledgerEntry
	settlement.Message = "usdt payment verified and project funded"

	return settlement, nil
}
