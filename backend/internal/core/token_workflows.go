package core

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	airdropClaimProtocolVersion             = "mergeos.airdrop-claim.v1"
	airdropMissionsProtocolVersion          = "mergeos.airdrop-missions.v1"
	presaleReservationProtocolVersion       = "mergeos.presale-reservation.v1"
	tokenLaunchBriefProtocolVersion         = "mergeos.token-launch-brief.v1"
	defaultAirdropAllocationMRG       int64 = 250
	maxAirdropAllocationMRG           int64 = 100000
	minPresaleReserveMRG              int64 = 100
	maxPresaleReserveMRG              int64 = 1000000
)

type AirdropMission struct {
	ID                   string   `json:"id"`
	Title                string   `json:"title"`
	Description          string   `json:"description"`
	ProofRequirement     string   `json:"proof_requirement"`
	RequiredReference    string   `json:"required_reference"`
	DefaultAllocationMRG int64    `json:"default_allocation_mrg"`
	MaxAllocationMRG     int64    `json:"max_allocation_mrg"`
	MissionScore         int64    `json:"mission_score"`
	ProofSignals         []string `json:"proof_signals"`
}

type AirdropMissionsResponse struct {
	ProtocolVersion string           `json:"protocol_version"`
	Kind            string           `json:"kind"`
	Missions        []AirdropMission `json:"missions"`
	Stats           map[string]int64 `json:"stats"`
}

type AirdropClaimRequest struct {
	MissionID          string   `json:"mission_id"`
	WorkerID           string   `json:"worker_id,omitempty"`
	WalletAddress      string   `json:"wallet_address"`
	TaskReference      string   `json:"task_reference,omitempty"`
	ProofURL           string   `json:"proof_url,omitempty"`
	ProofSignals       []string `json:"proof_signals,omitempty"`
	Notes              string   `json:"notes,omitempty"`
	AllocationMRG      int64    `json:"allocation_mrg,omitempty"`
	AllocationMRGCents int64    `json:"allocation_mrg_cents,omitempty"`
}

type AirdropClaimResponse struct {
	ProtocolVersion  string      `json:"protocol_version"`
	Kind             string      `json:"kind"`
	ClaimID          string      `json:"claim_id"`
	Status           string      `json:"status"`
	MissionID        string      `json:"mission_id"`
	WorkerID         string      `json:"worker_id"`
	WalletAddress    string      `json:"wallet_address"`
	TaskReference    string      `json:"task_reference,omitempty"`
	ProofURL         string      `json:"proof_url,omitempty"`
	ProofRequirement string      `json:"proof_requirement"`
	MissionScore     int64       `json:"mission_score"`
	MaxAllocationMRG int64       `json:"max_allocation_mrg"`
	ProofSignals     []string    `json:"proof_signals"`
	Notes            string      `json:"notes,omitempty"`
	AllocationMRG    int64       `json:"allocation_mrg"`
	LedgerEntry      LedgerEntry `json:"ledger_entry"`
	LedgerProofURL   string      `json:"ledger_proof_url"`
	LiveFeedURL      string      `json:"live_feed_url"`
	CreatedAt        time.Time   `json:"created_at"`
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

type TokenLaunchBriefRequest struct {
	LaunchType       string   `json:"launch_type"`
	ProjectTitle     string   `json:"project_title"`
	ProjectSummary   string   `json:"project_summary"`
	RepositoryURL    string   `json:"repository_url,omitempty"`
	AllocationPolicy string   `json:"allocation_policy,omitempty"`
	ProofPolicy      string   `json:"proof_policy,omitempty"`
	WalletPolicy     string   `json:"wallet_policy,omitempty"`
	RiskNotes        string   `json:"risk_notes,omitempty"`
	ResearchSignals  []string `json:"research_signals,omitempty"`
}

type TokenLaunchBriefResponse struct {
	ProtocolVersion  string      `json:"protocol_version"`
	Kind             string      `json:"kind"`
	BriefID          string      `json:"brief_id"`
	Status           string      `json:"status"`
	LaunchType       string      `json:"launch_type"`
	ProjectTitle     string      `json:"project_title"`
	ProjectSummary   string      `json:"project_summary"`
	RepositoryURL    string      `json:"repository_url,omitempty"`
	AllocationPolicy string      `json:"allocation_policy,omitempty"`
	ProofPolicy      string      `json:"proof_policy,omitempty"`
	WalletPolicy     string      `json:"wallet_policy,omitempty"`
	RiskNotes        string      `json:"risk_notes,omitempty"`
	ResearchSignals  []string    `json:"research_signals"`
	LedgerEntry      LedgerEntry `json:"ledger_entry"`
	LedgerProofURL   string      `json:"ledger_proof_url"`
	LiveFeedURL      string      `json:"live_feed_url"`
	CreatedAt        time.Time   `json:"created_at"`
}

var airdropMissionCatalog = []AirdropMission{
	{
		ID:                   "repo-import",
		Title:                "Repository import",
		Description:          "Import a GitHub repository or issue set so MergeOS can score real software work.",
		ProofRequirement:     "Attach an imported repository report, issue scan, or public task reference.",
		RequiredReference:    "task_or_url",
		DefaultAllocationMRG: 250,
		MaxAllocationMRG:     1000,
		MissionScore:         45,
		ProofSignals:         []string{"repo_import", "issue_scan"},
	},
	{
		ID:                   "bounty-work",
		Title:                "Bounty delivery",
		Description:          "Claim or complete an escrow-backed bounty with acceptance criteria.",
		ProofRequirement:     "Attach a bounty or task reference and public delivery evidence.",
		RequiredReference:    "task_reference",
		DefaultAllocationMRG: 500,
		MaxAllocationMRG:     2500,
		MissionScore:         65,
		ProofSignals:         []string{"bounty_claim", "task_submission"},
	},
	{
		ID:                   "pr-review",
		Title:                "Pull request review",
		Description:          "Review a pull request with public feedback or accepted review evidence.",
		ProofRequirement:     "Attach a GitHub pull request or review URL.",
		RequiredReference:    "proof_url",
		DefaultAllocationMRG: 350,
		MaxAllocationMRG:     1500,
		MissionScore:         55,
		ProofSignals:         []string{"pull_request", "review"},
	},
	{
		ID:                   "qa-check",
		Title:                "QA evidence",
		Description:          "Provide test, accessibility, regression, or smoke evidence for a funded work packet.",
		ProofRequirement:     "Attach a QA evidence URL, test run, or issue/task reference.",
		RequiredReference:    "task_or_url",
		DefaultAllocationMRG: 300,
		MaxAllocationMRG:     1200,
		MissionScore:         50,
		ProofSignals:         []string{"qa", "test_evidence"},
	},
	{
		ID:                   "agent-review",
		Title:                "AI agent review",
		Description:          "Record AI review, test, scan, or generation evidence linked to MergeOS agent workflow.",
		ProofRequirement:     "Attach an agent action, live feed, or workflow proof URL.",
		RequiredReference:    "proof_url",
		DefaultAllocationMRG: 400,
		MaxAllocationMRG:     1800,
		MissionScore:         60,
		ProofSignals:         []string{"agent_action", "ai_review"},
	},
	{
		ID:                   "deployment-proof",
		Title:                "Deployment proof",
		Description:          "Attach deployment, release, or rollout evidence for a delivered task.",
		ProofRequirement:     "Attach a deployment or release proof URL.",
		RequiredReference:    "proof_url",
		DefaultAllocationMRG: 450,
		MaxAllocationMRG:     2000,
		MissionScore:         62,
		ProofSignals:         []string{"deployment", "release"},
	},
}

func (s *Server) publicAirdropMissions(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, PublicAirdropMissions())
}

func PublicAirdropMissions() AirdropMissionsResponse {
	missions := make([]AirdropMission, 0, len(airdropMissionCatalog))
	var totalDefault int64
	var totalMax int64
	for _, mission := range airdropMissionCatalog {
		mission.ProofSignals = append([]string{}, mission.ProofSignals...)
		missions = append(missions, mission)
		totalDefault += mission.DefaultAllocationMRG
		totalMax += mission.MaxAllocationMRG
	}
	return AirdropMissionsResponse{
		ProtocolVersion: airdropMissionsProtocolVersion,
		Kind:            "airdrop_missions",
		Missions:        missions,
		Stats: map[string]int64{
			"mission_count":          int64(len(missions)),
			"default_allocation_mrg": totalDefault,
			"max_allocation_mrg":     totalMax,
			"average_mission_score":  averageAirdropMissionScore(missions),
		},
	}
}

func (s *Server) createAirdropClaim(w http.ResponseWriter, r *http.Request) {
	user, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	var req AirdropClaimRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	response, err := s.store.RecordAirdropClaimForUser(user.ID, req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	s.broadcastLiveFeedEvent("ledger_airdrop_claim")
	s.broadcastAdminOpsUpdated()
	writeJSON(w, http.StatusCreated, response)
}

func (s *Server) createPresaleReservation(w http.ResponseWriter, r *http.Request) {
	user, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	var req PresaleReservationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	response, err := s.store.RecordPresaleReservationForUser(user.ID, req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	s.broadcastLiveFeedEvent("ledger_presale_reservation")
	s.broadcastAdminOpsUpdated()
	writeJSON(w, http.StatusCreated, response)
}

func (s *Server) createTokenLaunchBrief(w http.ResponseWriter, r *http.Request) {
	user, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	var req TokenLaunchBriefRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	response, err := s.store.RecordTokenLaunchBriefForUser(user.ID, req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	s.broadcastLiveFeedEvent("ledger_token_launch_brief")
	s.broadcastAdminOpsUpdated()
	writeJSON(w, http.StatusCreated, response)
}

func (s *Store) RecordAirdropClaim(req AirdropClaimRequest) (AirdropClaimResponse, error) {
	return s.RecordAirdropClaimForUser("", req)
}

func (s *Store) RecordAirdropClaimForUser(userID string, req AirdropClaimRequest) (AirdropClaimResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	missionID := normalizeTokenWorkflowID(req.MissionID)
	if missionID == "" {
		return AirdropClaimResponse{}, errors.New("mission_id is required")
	}
	mission, ok := airdropMissionByID(missionID)
	if !ok {
		return AirdropClaimResponse{}, errors.New("mission_id must be one of: " + strings.Join(airdropMissionIDs(), ", "))
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
	if err := validateAirdropMissionProof(mission, taskReference, proofURL); err != nil {
		return AirdropClaimResponse{}, err
	}
	allocationMRG := selectedAirdropAllocationMRG(req)
	if allocationMRG <= 0 {
		allocationMRG = mission.DefaultAllocationMRG
	}
	if allocationMRG > maxAirdropAllocationMRG {
		return AirdropClaimResponse{}, errors.New("allocation_mrg is too large for an airdrop claim")
	}
	if allocationMRG > mission.MaxAllocationMRG {
		return AirdropClaimResponse{}, errors.New("allocation_mrg exceeds max allocation for mission " + mission.ID)
	}
	proofSignals := normalizeAirdropProofSignals(req.ProofSignals, mission, taskReference, proofURL)
	missionScore := airdropMissionScore(mission, taskReference, proofURL, proofSignals)
	notes := sanitizeLedgerReferenceValue(req.Notes)
	claimID := s.newID("adc")
	reference := tokenWorkflowReference([]string{
		"airdrop:" + claimID,
		"mission:" + missionID,
		"score:" + int64String(missionScore),
		"cap:" + int64String(mission.MaxAllocationMRG),
		"signals:" + strings.Join(proofSignals, ","),
		"task:" + taskReference,
		"proof:" + proofURL,
		"note:" + notes,
	})
	entry := s.addLedger("airdrop_claim", "airdrop:pool", walletAccount(walletAddress), allocationMRG, reference)
	if strings.TrimSpace(userID) != "" {
		s.addNotificationLocked(
			userID,
			"",
			"token_workflow",
			"Airdrop claim pending review",
			fmt.Sprintf("%s claim for %s is recorded on the public ledger and waiting for operator review.", mission.Title, formatTokenAmount(allocationMRG)),
			tokenWorkflowReviewStatus("airdrop", claimID, entry.Sequence),
		)
	}
	if err := s.saveLocked(); err != nil {
		return AirdropClaimResponse{}, err
	}
	return AirdropClaimResponse{
		ProtocolVersion:  airdropClaimProtocolVersion,
		Kind:             "airdrop_claim",
		ClaimID:          claimID,
		Status:           "claimed_pending_review",
		MissionID:        missionID,
		WorkerID:         workerID,
		WalletAddress:    walletAddress,
		TaskReference:    taskReference,
		ProofURL:         proofURL,
		ProofRequirement: mission.ProofRequirement,
		MissionScore:     missionScore,
		MaxAllocationMRG: mission.MaxAllocationMRG,
		ProofSignals:     proofSignals,
		Notes:            notes,
		AllocationMRG:    allocationMRG,
		LedgerEntry:      entry,
		LedgerProofURL:   "/api/public/ledger/proof",
		LiveFeedURL:      "/api/public/live-feed",
		CreatedAt:        entry.CreatedAt,
	}, nil
}

func (s *Store) RecordPresaleReservation(req PresaleReservationRequest) (PresaleReservationResponse, error) {
	return s.RecordPresaleReservationForUser("", req)
}

func (s *Store) RecordTokenLaunchBrief(req TokenLaunchBriefRequest) (TokenLaunchBriefResponse, error) {
	return s.RecordTokenLaunchBriefForUser("", req)
}

func (s *Store) RecordTokenLaunchBriefForUser(userID string, req TokenLaunchBriefRequest) (TokenLaunchBriefResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	launchType, err := normalizeTokenLaunchType(req.LaunchType)
	if err != nil {
		return TokenLaunchBriefResponse{}, err
	}
	projectTitle := sanitizeTokenLaunchText(req.ProjectTitle, 140)
	if len(projectTitle) < 6 {
		return TokenLaunchBriefResponse{}, errors.New("project_title must be at least 6 characters")
	}
	projectSummary := sanitizeTokenLaunchText(req.ProjectSummary, 1000)
	if len(projectSummary) < 24 {
		return TokenLaunchBriefResponse{}, errors.New("project_summary must be at least 24 characters")
	}
	repositoryURL := normalizeTokenWorkflowURL(req.RepositoryURL)
	if repositoryURL == "" && strings.TrimSpace(req.RepositoryURL) != "" {
		return TokenLaunchBriefResponse{}, errors.New("repository_url must be an http(s) URL")
	}
	allocationPolicy := sanitizeTokenLaunchText(req.AllocationPolicy, 260)
	proofPolicy := sanitizeTokenLaunchText(req.ProofPolicy, 260)
	walletPolicy := sanitizeTokenLaunchText(req.WalletPolicy, 260)
	riskNotes := sanitizeTokenLaunchText(req.RiskNotes, 260)
	researchSignals := normalizeTokenLaunchResearchSignals(req.ResearchSignals, launchType, repositoryURL, proofPolicy, walletPolicy)
	briefID := s.newID("tlb")
	reference := tokenWorkflowReference([]string{
		"launch_brief:" + briefID,
		"type:" + launchType,
		"title:" + projectTitle,
		"repo:" + repositoryURL,
		"signals:" + strings.Join(researchSignals, ","),
	})
	entry := s.addLedger("token_launch_brief", "token:launch-intake", "ceo:research", 0, reference)
	if strings.TrimSpace(userID) != "" {
		s.addNotificationLocked(
			userID,
			"",
			"token_workflow",
			"CEO token launch brief received",
			fmt.Sprintf("CEO research brief for %s is recorded and waiting for launch decision review.", marketplaceTitle(launchType)),
			tokenWorkflowReviewStatus("launch_brief", briefID, entry.Sequence),
		)
	}
	if err := s.saveLocked(); err != nil {
		return TokenLaunchBriefResponse{}, err
	}
	return TokenLaunchBriefResponse{
		ProtocolVersion:  tokenLaunchBriefProtocolVersion,
		Kind:             "token_launch_brief",
		BriefID:          briefID,
		Status:           "research_pending",
		LaunchType:       launchType,
		ProjectTitle:     projectTitle,
		ProjectSummary:   projectSummary,
		RepositoryURL:    repositoryURL,
		AllocationPolicy: allocationPolicy,
		ProofPolicy:      proofPolicy,
		WalletPolicy:     walletPolicy,
		RiskNotes:        riskNotes,
		ResearchSignals:  researchSignals,
		LedgerEntry:      entry,
		LedgerProofURL:   "/api/public/ledger/proof",
		LiveFeedURL:      "/api/public/live-feed",
		CreatedAt:        entry.CreatedAt,
	}, nil
}

func (s *Store) RecordPresaleReservationForUser(userID string, req PresaleReservationRequest) (PresaleReservationResponse, error) {
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
	if strings.TrimSpace(userID) != "" {
		s.addNotificationLocked(
			userID,
			"",
			"token_workflow",
			"Presale reservation pending review",
			fmt.Sprintf("%s reservation for %s is recorded on the public ledger and waiting for operator review.", marketplaceTitle(tier), formatTokenAmount(reserveMRG)),
			tokenWorkflowReviewStatus("presale", reservationID, entry.Sequence),
		)
	}
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

func tokenWorkflowReviewStatus(kind, id string, sequence int) string {
	parts := []string{
		"token_workflow:" + sanitizeLedgerReferenceValue(kind),
		"id:" + sanitizeLedgerReferenceValue(id),
	}
	if sequence > 0 {
		parts = append(parts, "ledger:"+strconv.Itoa(sequence))
	}
	parts = append(parts, "status:pending_review")
	return strings.Join(parts, ";")
}

func normalizeTokenLaunchType(value string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(value))
	switch normalized {
	case "airdrop", "earned_airdrop", "task_airdrop":
		return "airdrop", nil
	case "presale", "reserve", "reservation":
		return "presale", nil
	default:
		return "", errors.New("launch_type must be airdrop or presale")
	}
}

func sanitizeTokenLaunchText(value string, maxLen int) string {
	text := strings.TrimSpace(value)
	text = strings.Join(strings.Fields(text), " ")
	if maxLen > 0 && len(text) > maxLen {
		text = text[:maxLen]
	}
	return sanitizeLedgerReferenceValue(text)
}

func normalizeTokenLaunchResearchSignals(values []string, launchType, repositoryURL, proofPolicy, walletPolicy string) []string {
	seen := map[string]bool{}
	signals := []string{}
	add := func(value string) {
		normalized := normalizeTokenWorkflowID(value)
		if normalized == "" || seen[normalized] {
			return
		}
		seen[normalized] = true
		signals = append(signals, normalized)
	}
	add(launchType + "_launch")
	for _, value := range values {
		add(value)
	}
	if repositoryURL != "" {
		add("repository_context")
	}
	if proofPolicy != "" {
		add("proof_policy")
	}
	if walletPolicy != "" {
		add("wallet_policy")
	}
	if len(signals) == 0 {
		add("ceo_research")
	}
	if len(signals) > 12 {
		signals = signals[:12]
	}
	return signals
}

func selectedAirdropAllocationMRG(req AirdropClaimRequest) int64 {
	if req.AllocationMRG > 0 {
		return req.AllocationMRG
	}
	return req.AllocationMRGCents
}

func averageAirdropMissionScore(missions []AirdropMission) int64 {
	if len(missions) == 0 {
		return 0
	}
	var total int64
	for _, mission := range missions {
		total += mission.MissionScore
	}
	return total / int64(len(missions))
}

func airdropMissionByID(id string) (AirdropMission, bool) {
	id = normalizeTokenWorkflowID(id)
	for _, mission := range airdropMissionCatalog {
		if mission.ID == id {
			mission.ProofSignals = append([]string{}, mission.ProofSignals...)
			return mission, true
		}
	}
	return AirdropMission{}, false
}

func airdropMissionIDs() []string {
	ids := make([]string, 0, len(airdropMissionCatalog))
	for _, mission := range airdropMissionCatalog {
		ids = append(ids, mission.ID)
	}
	return ids
}

func validateAirdropMissionProof(mission AirdropMission, taskReference, proofURL string) error {
	switch mission.RequiredReference {
	case "proof_url":
		if proofURL == "" {
			return errors.New("proof_url is required for mission " + mission.ID)
		}
	case "task_reference":
		if taskReference == "" {
			return errors.New("task_reference is required for mission " + mission.ID)
		}
	default:
		if proofURL == "" && taskReference == "" {
			return errors.New("proof_url or task_reference is required for mission " + mission.ID)
		}
	}
	return nil
}

func normalizeAirdropProofSignals(values []string, mission AirdropMission, taskReference, proofURL string) []string {
	seen := map[string]bool{}
	signals := []string{}
	add := func(value string) {
		value = normalizeTokenWorkflowID(value)
		if value == "" || seen[value] {
			return
		}
		seen[value] = true
		signals = append(signals, value)
	}
	for _, value := range mission.ProofSignals {
		add(value)
	}
	for _, value := range values {
		add(value)
	}
	if taskReference != "" {
		add("task_reference")
	}
	if proofURL != "" {
		add("proof_url")
	}
	return signals
}

func airdropMissionScore(mission AirdropMission, taskReference, proofURL string, proofSignals []string) int64 {
	score := mission.MissionScore
	if taskReference != "" {
		score += 5
	}
	if proofURL != "" {
		score += 5
	}
	if extra := int64(len(proofSignals) - len(mission.ProofSignals)); extra > 0 {
		score += extra * 2
	}
	if score > 100 {
		return 100
	}
	if score < 1 {
		return 1
	}
	return score
}

func int64String(value int64) string {
	return strconv.FormatInt(value, 10)
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
