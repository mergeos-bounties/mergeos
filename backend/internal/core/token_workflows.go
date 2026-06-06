package core

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	airdropClaimProtocolVersion             = "mergeos.airdrop-claim.v1"
	presaleReservationProtocolVersion       = "mergeos.presale-reservation.v1"
	defaultAirdropAllocationMRG       int64 = 250
	maxAirdropAllocationMRG           int64 = 100000
	minPresaleReserveMRG              int64 = 100
	maxPresaleReserveMRG              int64 = 1000000
)

type AirdropClaimRequest struct {
	MissionID          string `json:"mission_id"`
	WorkerID           string `json:"worker_id,omitempty"`
	WalletAddress      string `json:"wallet_address"`
	TaskReference      string `json:"task_reference,omitempty"`
	ProofURL           string `json:"proof_url,omitempty"`
	Notes              string `json:"notes,omitempty"`
	AllocationMRG      int64  `json:"allocation_mrg,omitempty"`
	AllocationMRGCents int64  `json:"allocation_mrg_cents,omitempty"`
}

type AirdropClaimResponse struct {
	ProtocolVersion string      `json:"protocol_version"`
	Kind            string      `json:"kind"`
	ClaimID         string      `json:"claim_id"`
	Status          string      `json:"status"`
	MissionID       string      `json:"mission_id"`
	WorkerID        string      `json:"worker_id"`
	WalletAddress   string      `json:"wallet_address"`
	TaskReference   string      `json:"task_reference,omitempty"`
	ProofURL        string      `json:"proof_url,omitempty"`
	Notes           string      `json:"notes,omitempty"`
	AllocationMRG   int64       `json:"allocation_mrg"`
	LedgerEntry     LedgerEntry `json:"ledger_entry"`
	LedgerProofURL  string      `json:"ledger_proof_url"`
	LiveFeedURL     string      `json:"live_feed_url"`
	CreatedAt       time.Time   `json:"created_at"`
}

type PresaleReservationRequest struct {
	WalletAddress    string `json:"wallet_address"`
	ReserveMRG       int64  `json:"reserve_mrg,omitempty"`
	ReserveMRGCents  int64  `json:"reserve_mrg_cents,omitempty"`
	FundingRail      string `json:"funding_rail"`
	FundingReference string `json:"funding_reference,omitempty"`
	Tier             string `json:"tier,omitempty"`
	Notes            string `json:"notes,omitempty"`
}

type PresaleReservationResponse struct {
	ProtocolVersion  string      `json:"protocol_version"`
	Kind             string      `json:"kind"`
	ReservationID    string      `json:"reservation_id"`
	Status           string      `json:"status"`
	WalletAddress    string      `json:"wallet_address"`
	ReserveMRG       int64       `json:"reserve_mrg"`
	FundingRail      string      `json:"funding_rail"`
	FundingReference string      `json:"funding_reference,omitempty"`
	Tier             string      `json:"tier"`
	Notes            string      `json:"notes,omitempty"`
	LedgerEntry      LedgerEntry `json:"ledger_entry"`
	LedgerProofURL   string      `json:"ledger_proof_url"`
	LiveFeedURL      string      `json:"live_feed_url"`
	CreatedAt        time.Time   `json:"created_at"`
}

func (s *Server) createAirdropClaim(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireUser(w, r); !ok {
		return
	}
	var req AirdropClaimRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	response, err := s.store.RecordAirdropClaim(req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	s.broadcastLiveFeedEvent("ledger_airdrop_claim")
	writeJSON(w, http.StatusCreated, response)
}

func (s *Server) createPresaleReservation(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireUser(w, r); !ok {
		return
	}
	var req PresaleReservationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	response, err := s.store.RecordPresaleReservation(req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	s.broadcastLiveFeedEvent("ledger_presale_reservation")
	writeJSON(w, http.StatusCreated, response)
}

func (s *Store) RecordAirdropClaim(req AirdropClaimRequest) (AirdropClaimResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	missionID := normalizeTokenWorkflowID(req.MissionID)
	if missionID == "" {
		return AirdropClaimResponse{}, errors.New("mission_id is required")
	}
	walletAddress := normalizeWalletAddress(req.WalletAddress)
	if !validWalletAddress(walletAddress) {
		return AirdropClaimResponse{}, errors.New("valid Solana wallet_address is required")
	}
	workerID := normalizeAdminCreditWorkerID(req.WorkerID)
	if strings.TrimSpace(workerID) == "" {
		workerID = walletAccount(walletAddress)
	}
	taskReference := sanitizeLedgerReferenceValue(req.TaskReference)
	proofURL := normalizeTokenWorkflowURL(req.ProofURL)
	if proofURL == "" && strings.TrimSpace(req.ProofURL) != "" {
		return AirdropClaimResponse{}, errors.New("proof_url must be an http(s) URL")
	}
	if proofURL == "" && taskReference == "" {
		return AirdropClaimResponse{}, errors.New("proof_url or task_reference is required")
	}
	allocationMRG := selectedAirdropAllocationMRG(req)
	if allocationMRG <= 0 {
		allocationMRG = defaultAirdropAllocationMRG
	}
	if allocationMRG > maxAirdropAllocationMRG {
		return AirdropClaimResponse{}, errors.New("allocation_mrg is too large for an airdrop claim")
	}
	notes := sanitizeLedgerReferenceValue(req.Notes)
	claimID := s.newID("adc")
	reference := tokenWorkflowReference([]string{
		"airdrop:" + claimID,
		"mission:" + missionID,
		"task:" + taskReference,
		"proof:" + proofURL,
		"note:" + notes,
	})
	entry := s.addLedger("airdrop_claim", "airdrop:pool", walletAccount(walletAddress), allocationMRG, reference)
	if err := s.saveLocked(); err != nil {
		return AirdropClaimResponse{}, err
	}
	return AirdropClaimResponse{
		ProtocolVersion: airdropClaimProtocolVersion,
		Kind:            "airdrop_claim",
		ClaimID:         claimID,
		Status:          "claimed_pending_review",
		MissionID:       missionID,
		WorkerID:        workerID,
		WalletAddress:   walletAddress,
		TaskReference:   taskReference,
		ProofURL:        proofURL,
		Notes:           notes,
		AllocationMRG:   allocationMRG,
		LedgerEntry:     entry,
		LedgerProofURL:  "/api/public/ledger/proof",
		LiveFeedURL:     "/api/public/live-feed",
		CreatedAt:       entry.CreatedAt,
	}, nil
}

func (s *Store) RecordPresaleReservation(req PresaleReservationRequest) (PresaleReservationResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	walletAddress := normalizeWalletAddress(req.WalletAddress)
	if !validWalletAddress(walletAddress) {
		return PresaleReservationResponse{}, errors.New("valid Solana wallet_address is required")
	}
	reserveMRG := selectedPresaleReserveMRG(req)
	if reserveMRG < minPresaleReserveMRG {
		return PresaleReservationResponse{}, errors.New("reserve_mrg must be at least 100")
	}
	if reserveMRG > maxPresaleReserveMRG {
		return PresaleReservationResponse{}, errors.New("reserve_mrg is too large for one reservation")
	}
	fundingRail, err := normalizePresaleFundingRail(req.FundingRail)
	if err != nil {
		return PresaleReservationResponse{}, err
	}
	tier := normalizePresaleTier(req.Tier)
	fundingReference := sanitizeLedgerReferenceValue(req.FundingReference)
	if fundingReference == "" {
		fundingReference = "pending_review"
	}
	notes := sanitizeLedgerReferenceValue(req.Notes)
	reservationID := s.newID("psr")
	reference := tokenWorkflowReference([]string{
		"presale:" + reservationID,
		"tier:" + tier,
		"rail:" + fundingRail,
		"funding:" + fundingReference,
		"note:" + notes,
	})
	entry := s.addLedger("presale_reservation", walletAccount(walletAddress), "presale:reserve", reserveMRG, reference)
	if err := s.saveLocked(); err != nil {
		return PresaleReservationResponse{}, err
	}
	return PresaleReservationResponse{
		ProtocolVersion:  presaleReservationProtocolVersion,
		Kind:             "presale_reservation",
		ReservationID:    reservationID,
		Status:           "reserved_pending_review",
		WalletAddress:    walletAddress,
		ReserveMRG:       reserveMRG,
		FundingRail:      fundingRail,
		FundingReference: fundingReference,
		Tier:             tier,
		Notes:            notes,
		LedgerEntry:      entry,
		LedgerProofURL:   "/api/public/ledger/proof",
		LiveFeedURL:      "/api/public/live-feed",
		CreatedAt:        entry.CreatedAt,
	}, nil
}

func selectedAirdropAllocationMRG(req AirdropClaimRequest) int64 {
	if req.AllocationMRG > 0 {
		return req.AllocationMRG
	}
	return req.AllocationMRGCents
}

func selectedPresaleReserveMRG(req PresaleReservationRequest) int64 {
	if req.ReserveMRG > 0 {
		return req.ReserveMRG
	}
	return req.ReserveMRGCents
}

func normalizeTokenWorkflowID(value string) string {
	value = strings.ToLower(sanitizeLedgerReferenceValue(value))
	value = strings.ReplaceAll(value, " ", "-")
	value = strings.Trim(value, "-")
	return value
}

func normalizeTokenWorkflowURL(value string) string {
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return ""
	}
	if parsed.Scheme != "https" && parsed.Scheme != "http" {
		return ""
	}
	return parsed.String()
}

func normalizePresaleFundingRail(value string) (string, error) {
	rail := strings.ToLower(strings.TrimSpace(value))
	switch rail {
	case "solana", "usdc", "paypal", "card", "bank", "manual_review":
		return rail, nil
	case "":
		return "", errors.New("funding_rail is required")
	default:
		return "", errors.New("funding_rail must be solana, usdc, paypal, card, bank, or manual_review")
	}
}

func normalizePresaleTier(value string) string {
	tier := strings.ToLower(strings.TrimSpace(sanitizeLedgerReferenceValue(value)))
	switch tier {
	case "builder", "founder", "protocol", "strategic":
		return tier
	default:
		return "builder"
	}
}

func tokenWorkflowReference(parts []string) string {
	clean := make([]string, 0, len(parts))
	for _, part := range parts {
		key, value, ok := strings.Cut(part, ":")
		value = sanitizeLedgerReferenceValue(value)
		if !ok || strings.TrimSpace(key) == "" || value == "" {
			continue
		}
		clean = append(clean, strings.TrimSpace(key)+":"+value)
	}
	return strings.Join(clean, ";")
}
