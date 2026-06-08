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
	airdropClaimProtocolVersion                = "mergeos.airdrop-claim.v1"
	airdropMissionsProtocolVersion             = "mergeos.airdrop-missions.v1"
	presaleReservationProtocolVersion          = "mergeos.presale-reservation.v1"
	tokenLaunchBriefProtocolVersion            = "mergeos.token-launch-brief.v1"
	tokenLaunchBriefsProtocolVersion           = "mergeos.token-launch-briefs.v1"
	tokenLaunchCandidatesProtocolVersion       = "mergeos.token-launch-candidates.v1"
	defaultAirdropAllocationMRG          int64 = 250
	maxAirdropAllocationMRG              int64 = 100000
	minPresaleReserveMRG                 int64 = 100
	maxPresaleReserveMRG                 int64 = 1000000
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
	CEOMemo          CEOMemo     `json:"ceo_memo"`
	LedgerEntry      LedgerEntry `json:"ledger_entry"`
	LedgerProofURL   string      `json:"ledger_proof_url"`
	LiveFeedURL      string      `json:"live_feed_url"`
	CreatedAt        time.Time   `json:"created_at"`
}

type PublicTokenLaunchBriefsResponse struct {
	ProtocolVersion string                         `json:"protocol_version"`
	Kind            string                         `json:"kind"`
	Stats           PublicTokenLaunchBriefsStats   `json:"stats"`
	Briefs          []PublicTokenLaunchBriefRecord `json:"briefs"`
}

type PublicTokenLaunchBriefsStats struct {
	BriefCount   int        `json:"brief_count"`
	AirdropCount int        `json:"airdrop_count"`
	PresaleCount int        `json:"presale_count"`
	UpdatedAt    *time.Time `json:"updated_at,omitempty"`
}

type PublicTokenLaunchBriefRecord struct {
	BriefID          string    `json:"brief_id"`
	LaunchType       string    `json:"launch_type"`
	ProjectTitle     string    `json:"project_title"`
	ProjectSummary   string    `json:"project_summary,omitempty"`
	AllocationPolicy string    `json:"allocation_policy,omitempty"`
	ProofPolicy      string    `json:"proof_policy,omitempty"`
	WalletPolicy     string    `json:"wallet_policy,omitempty"`
	RiskNotes        string    `json:"risk_notes,omitempty"`
	Decision         string    `json:"decision"`
	GateSummary      string    `json:"gate_summary"`
	GatesReference   string    `json:"gates_reference"`
	ResearchSource   string    `json:"research_source"`
	ResearchSignals  []string  `json:"research_signals"`
	LedgerSequence   int       `json:"ledger_sequence"`
	EntryHash        string    `json:"entry_hash"`
	LedgerProofURL   string    `json:"ledger_proof_url"`
	CreatedAt        time.Time `json:"created_at"`
}

type PublicTokenLaunchCandidatesResponse struct {
	ProtocolVersion  string                           `json:"protocol_version"`
	Kind             string                           `json:"kind"`
	LaunchTypeFilter string                           `json:"launch_type_filter"`
	Stats            PublicTokenLaunchCandidatesStats `json:"stats"`
	Candidates       []PublicTokenLaunchCandidate     `json:"candidates"`
}

type PublicTokenLaunchCandidatesStats struct {
	CandidateCount int        `json:"candidate_count"`
	AirdropCount   int        `json:"airdrop_count"`
	PresaleCount   int        `json:"presale_count"`
	ReadyCount     int        `json:"ready_count"`
	ReviewCount    int        `json:"review_count"`
	HoldCount      int        `json:"hold_count"`
	UpdatedAt      *time.Time `json:"updated_at,omitempty"`
}

type PublicTokenLaunchCandidate struct {
	CandidateID            string                               `json:"candidate_id"`
	ProjectID              string                               `json:"project_id"`
	ProjectTitle           string                               `json:"project_title"`
	RecommendedLaunchTypes []string                             `json:"recommended_launch_types"`
	ResearchSource         string                               `json:"research_source"`
	Brief                  string                               `json:"brief"`
	WorkPoolMRG            int64                                `json:"work_pool_mrg"`
	OpenTaskCount          int                                  `json:"open_task_count"`
	AcceptedTaskCount      int                                  `json:"accepted_task_count"`
	ResearchScore          int                                  `json:"research_score"`
	ProofSignals           []string                             `json:"proof_signals"`
	DecisionOptions        []TokenLaunchCandidateDecisionOption `json:"decision_options"`
	ReadinessGates         []TokenLaunchCandidateReadinessGate  `json:"readiness_gates"`
	NextAction             string                               `json:"next_action"`
	GateSummary            string                               `json:"gate_summary"`
	ProofPolicy            string                               `json:"proof_policy"`
	MarketplaceURL         string                               `json:"marketplace_url"`
	CreatedAt              time.Time                            `json:"created_at"`
}

type TokenLaunchCandidateReadinessGate struct {
	Key      string `json:"key"`
	Label    string `json:"label"`
	State    string `json:"state"`
	Value    string `json:"value"`
	Evidence string `json:"evidence"`
}

type TokenLaunchCandidateDecisionOption struct {
	Key         string `json:"key"`
	Label       string `json:"label"`
	Tone        string `json:"tone"`
	ProofPolicy string `json:"proof_policy"`
	RiskNotes   string `json:"risk_notes"`
}

type CEOMemoGate struct {
	Key      string `json:"key"`
	Label    string `json:"label"`
	Status   string `json:"status"`
	Required bool   `json:"required"`
	Evidence string `json:"evidence"`
}

type CEOMemo struct {
	Decision      string        `json:"decision"`
	DecisionLabel string        `json:"decision_label"`
	ReviewOwner   string        `json:"review_owner"`
	NextAction    string        `json:"next_action"`
	Gates         []CEOMemoGate `json:"gates"`
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

func (s *Server) publicTokenLaunchBriefs(w http.ResponseWriter, r *http.Request) {
	launchType := strings.TrimSpace(r.URL.Query().Get("launch_type"))
	if launchType != "" {
		normalized, err := normalizeTokenLaunchType(launchType)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		launchType = normalized
	}
	writeJSON(w, http.StatusOK, s.store.PublicTokenLaunchBriefs(launchType))
}

func (s *Server) publicTokenLaunchCandidates(w http.ResponseWriter, r *http.Request) {
	launchType := strings.TrimSpace(r.URL.Query().Get("launch_type"))
	if launchType != "" {
		normalized, err := normalizeTokenLaunchType(launchType)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		launchType = normalized
	}
	writeJSON(w, http.StatusOK, s.store.PublicTokenLaunchCandidates(launchType))
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

func (s *Store) PublicTokenLaunchBriefs(launchTypeFilter string) PublicTokenLaunchBriefsResponse {
	s.mu.RLock()
	defer s.mu.RUnlock()

	response := PublicTokenLaunchBriefsResponse{
		ProtocolVersion: tokenLaunchBriefsProtocolVersion,
		Kind:            "token_launch_briefs",
		Briefs:          []PublicTokenLaunchBriefRecord{},
	}
	for _, entry := range s.ledger {
		if entry.Type != "token_launch_brief" {
			continue
		}
		fields := tokenWorkflowReferenceFields(entry.Reference)
		launchType := fields["type"]
		switch launchType {
		case "airdrop", "presale":
		default:
			launchType = "token_launch"
		}
		if launchTypeFilter != "" && launchType != launchTypeFilter {
			continue
		}
		switch launchType {
		case "airdrop":
			response.Stats.AirdropCount++
		case "presale":
			response.Stats.PresaleCount++
		}
		if response.Stats.UpdatedAt == nil || entry.CreatedAt.After(*response.Stats.UpdatedAt) {
			updatedAt := entry.CreatedAt
			response.Stats.UpdatedAt = &updatedAt
		}
		record := PublicTokenLaunchBriefRecord{
			BriefID:          fields["launch_brief"],
			LaunchType:       launchType,
			ProjectTitle:     fields["title"],
			ProjectSummary:   fields["summary"],
			AllocationPolicy: fields["allocation_policy"],
			ProofPolicy:      fields["proof_policy"],
			WalletPolicy:     fields["wallet_policy"],
			RiskNotes:        fields["risk_notes"],
			Decision:         fields["decision"],
			GateSummary:      fields["gate_summary"],
			GatesReference:   fields["gates"],
			ResearchSource:   fields["source"],
			ResearchSignals:  tokenWorkflowReferenceList(fields["signals"]),
			LedgerSequence:   entry.Sequence,
			EntryHash:        entry.EntryHash,
			LedgerProofURL:   "/api/public/ledger/proof",
			CreatedAt:        entry.CreatedAt,
		}
		if record.ResearchSource == "" {
			record.ResearchSource = fields["repo"]
		}
		response.Briefs = append(response.Briefs, record)
	}
	for i, j := 0, len(response.Briefs)-1; i < j; i, j = i+1, j-1 {
		response.Briefs[i], response.Briefs[j] = response.Briefs[j], response.Briefs[i]
	}
	response.Stats.BriefCount = len(response.Briefs)
	return response
}

func (s *Store) PublicTokenLaunchCandidates(launchTypeFilter string) PublicTokenLaunchCandidatesResponse {
	marketplace := s.Marketplace()
	appliedFilter := launchTypeFilter
	if appliedFilter == "" {
		appliedFilter = "all"
	}
	response := PublicTokenLaunchCandidatesResponse{
		ProtocolVersion:  tokenLaunchCandidatesProtocolVersion,
		Kind:             "token_launch_candidates",
		LaunchTypeFilter: appliedFilter,
		Stats: PublicTokenLaunchCandidatesStats{
			UpdatedAt: marketplace.Stats.UpdatedAt,
		},
		Candidates: []PublicTokenLaunchCandidate{},
	}
	seenSources := map[string]bool{}
	addCandidate := func(candidate PublicTokenLaunchCandidate) {
		if strings.TrimSpace(candidate.CandidateID) == "" {
			return
		}
		response.Candidates = append(response.Candidates, candidate)
		statsLaunchTypes := candidate.RecommendedLaunchTypes
		if launchTypeFilter != "" {
			statsLaunchTypes = []string{launchTypeFilter}
		}
		for _, launchType := range statsLaunchTypes {
			switch launchType {
			case "airdrop":
				response.Stats.AirdropCount++
			case "presale":
				response.Stats.PresaleCount++
			}
		}
		switch tokenLaunchCandidateReadinessState(candidate.ReadinessGates) {
		case "ready":
			response.Stats.ReadyCount++
		case "hold":
			response.Stats.HoldCount++
		default:
			response.Stats.ReviewCount++
		}
		if candidate.CreatedAt.After(time.Time{}) && (response.Stats.UpdatedAt == nil || candidate.CreatedAt.After(*response.Stats.UpdatedAt)) {
			updatedAt := candidate.CreatedAt
			response.Stats.UpdatedAt = &updatedAt
		}
	}
	bountiesByProject := map[string][]*MarketplaceBounty{}
	for _, bounty := range marketplace.Bounties {
		bountiesByProject[bounty.ProjectID] = append(bountiesByProject[bounty.ProjectID], bounty)
	}
	for _, project := range marketplace.Projects {
		recommendedTypes := tokenLaunchCandidateTypes(project)
		if launchTypeFilter != "" && !stringSliceContains(recommendedTypes, launchTypeFilter) {
			continue
		}
		projectBounties := bountiesByProject[project.ID]
		source := project.RepoURL
		if source == "" && len(projectBounties) > 0 {
			source = projectBounties[0].SourceRepository
			if source == "" {
				source = projectBounties[0].IssueURL
			}
		}
		if source != "" {
			seenSources[strings.ToLower(strings.TrimSpace(source))] = true
		}
		proofSignals := tokenLaunchCandidateSignals(project, projectBounties)
		researchScore := tokenLaunchCandidateResearchScore(project, proofSignals)
		decisionLaunchType := tokenLaunchCandidateDecisionLaunchType(recommendedTypes, launchTypeFilter)
		readinessGates := tokenLaunchCandidateReadinessGates(decisionLaunchType, researchScore, project, proofSignals)
		readyToOpen := tokenLaunchCandidateReadinessState(readinessGates) == "ready"
		nextAction := tokenLaunchCandidateNextAction(decisionLaunchType, researchScore, readyToOpen)
		candidate := PublicTokenLaunchCandidate{
			CandidateID:            "tlc_" + project.ID,
			ProjectID:              project.ID,
			ProjectTitle:           project.Title,
			RecommendedLaunchTypes: recommendedTypes,
			ResearchSource:         source,
			Brief:                  project.Brief,
			WorkPoolMRG:            project.WorkPoolCents,
			OpenTaskCount:          project.OpenTaskCount,
			AcceptedTaskCount:      project.AcceptedTaskCount,
			ResearchScore:          researchScore,
			ProofSignals:           proofSignals,
			DecisionOptions:        tokenLaunchCandidateDecisionOptions(decisionLaunchType, researchScore, readyToOpen),
			ReadinessGates:         readinessGates,
			NextAction:             nextAction,
			GateSummary:            fmt.Sprintf("%d open tasks, %d accepted tasks, %d proof signals", project.OpenTaskCount, project.AcceptedTaskCount, len(proofSignals)),
			ProofPolicy:            tokenLaunchCandidateProofPolicy(projectBounties),
			MarketplaceURL:         "/marketplace",
			CreatedAt:              project.CreatedAt,
		}
		addCandidate(candidate)
	}
	for _, brief := range s.PublicTokenLaunchBriefs(launchTypeFilter).Briefs {
		sourceKey := strings.ToLower(strings.TrimSpace(brief.ResearchSource))
		if sourceKey != "" && seenSources[sourceKey] {
			continue
		}
		if sourceKey != "" {
			seenSources[sourceKey] = true
		}
		launchType := brief.LaunchType
		if launchType != "presale" {
			launchType = "airdrop"
		}
		proofSignals := tokenLaunchBriefCandidateSignals(brief)
		researchScore := tokenLaunchBriefCandidateScore(brief, proofSignals)
		readinessGates := tokenLaunchBriefCandidateReadinessGates(launchType, brief, proofSignals)
		readyToOpen := tokenLaunchCandidateReadinessState(readinessGates) == "ready"
		addCandidate(PublicTokenLaunchCandidate{
			CandidateID:            "tlb_" + brief.BriefID,
			ProjectID:              "launch_brief:" + brief.BriefID,
			ProjectTitle:           brief.ProjectTitle,
			RecommendedLaunchTypes: []string{launchType},
			ResearchSource:         brief.ResearchSource,
			Brief:                  tokenLaunchBriefCandidateSummary(brief),
			ResearchScore:          researchScore,
			ProofSignals:           proofSignals,
			DecisionOptions:        tokenLaunchCandidateDecisionOptions(launchType, researchScore, readyToOpen),
			ReadinessGates:         readinessGates,
			NextAction:             tokenLaunchCandidateBriefNextAction(launchType),
			GateSummary:            brief.GateSummary,
			ProofPolicy:            tokenLaunchBriefCandidateProofPolicy(launchType, brief),
			MarketplaceURL:         "/marketplace",
			CreatedAt:              brief.CreatedAt,
		})
	}
	response.Stats.CandidateCount = len(response.Candidates)
	return response
}

func tokenLaunchBriefCandidateSignals(brief PublicTokenLaunchBriefRecord) []string {
	signals := []string{"ceo_submitted_brief", "ceo_research_candidate"}
	if strings.TrimSpace(brief.ResearchSource) != "" {
		signals = append(signals, "research_source")
		if strings.Contains(strings.ToLower(brief.ResearchSource), "github.com") {
			signals = append(signals, "repository_context")
		}
	}
	for _, signal := range brief.ResearchSignals {
		normalized := normalizeTokenWorkflowID(signal)
		if normalized != "" && !stringSliceContains(signals, normalized) {
			signals = append(signals, normalized)
		}
	}
	return signals
}

func tokenLaunchBriefCandidateSummary(brief PublicTokenLaunchBriefRecord) string {
	if summary := strings.TrimSpace(brief.ProjectSummary); summary != "" {
		return summary
	}
	return "CEO-submitted launch brief waiting for candidate review."
}

func tokenLaunchBriefCandidateScore(brief PublicTokenLaunchBriefRecord, proofSignals []string) int {
	score := 50 + (minInt(len(proofSignals), 8) * 5)
	if strings.Contains(brief.GateSummary, "4/4") || strings.Contains(brief.GateSummary, "3/3") {
		score += 14
	}
	if strings.TrimSpace(brief.ResearchSource) != "" {
		score += 6
	}
	if score > 88 {
		return 88
	}
	if score < 50 {
		return 50
	}
	return score
}

func tokenLaunchBriefCandidateReadinessGates(launchType string, brief PublicTokenLaunchBriefRecord, proofSignals []string) []TokenLaunchCandidateReadinessGate {
	sourceEvidence := strings.TrimSpace(brief.ResearchSource)
	if sourceEvidence == "" {
		sourceEvidence = "Research source missing from CEO brief."
	}
	gateSummary := strings.TrimSpace(brief.GateSummary)
	if gateSummary == "" {
		gateSummary = "CEO gate summary pending."
	}
	signalEvidence := strings.Join(proofSignals[:minInt(len(proofSignals), 3)], ", ")
	if signalEvidence == "" {
		signalEvidence = "No proof signals attached yet."
	}
	allocationEvidence := tokenLaunchBriefPolicyEvidence(brief.AllocationPolicy, gateSummary)
	proofEvidence := tokenLaunchBriefPolicyEvidence(brief.ProofPolicy, signalEvidence)
	walletEvidence := tokenLaunchBriefPolicyEvidence(brief.WalletPolicy, sourceEvidence)
	riskEvidence := tokenLaunchBriefPolicyEvidence(brief.RiskNotes, "")
	if launchType == "presale" {
		contractEvidence := tokenLaunchBriefJoinEvidence(proofEvidence, riskEvidence)
		return []TokenLaunchCandidateReadinessGate{
			{Key: "utility", Label: "Utility", State: "review", Value: gateSummary, Evidence: allocationEvidence},
			{Key: "funding", Label: "Funding", State: "review", Value: "CEO brief queued", Evidence: walletEvidence},
			{Key: "contract", Label: "Contract", State: "review", Value: "Needs signoff", Evidence: contractEvidence},
		}
	}
	antiBotEvidence := tokenLaunchBriefJoinEvidence(walletEvidence, riskEvidence)
	return []TokenLaunchCandidateReadinessGate{
		{Key: "demand", Label: "Demand", State: "review", Value: gateSummary, Evidence: sourceEvidence},
		{Key: "proof", Label: "Proof", State: "review", Value: fmt.Sprintf("%d signals attached", len(proofSignals)), Evidence: proofEvidence},
		{Key: "anti_bot", Label: "Anti-bot", State: "review", Value: "Needs signoff", Evidence: antiBotEvidence},
	}
}

func tokenLaunchBriefPolicyEvidence(primary, fallback string) string {
	if trimmed := strings.TrimSpace(primary); trimmed != "" {
		return trimmed
	}
	if trimmed := strings.TrimSpace(fallback); trimmed != "" {
		return trimmed
	}
	return "CEO policy evidence pending."
}

func tokenLaunchBriefJoinEvidence(first, second string) string {
	first = strings.TrimSpace(first)
	second = strings.TrimSpace(second)
	if first == "" {
		return tokenLaunchBriefPolicyEvidence(second, "")
	}
	if second == "" || second == first {
		return first
	}
	return first + " " + second
}

func tokenLaunchCandidateBriefNextAction(launchType string) string {
	if launchType == "presale" {
		return "Review the submitted presale brief, utility proof, wallet path, funding rail, and Solana contract receipt before opening."
	}
	return "Review the submitted airdrop brief, mission demand, anti-bot policy, wallet uniqueness, and proof gates before opening missions."
}

func tokenLaunchBriefCandidateProofPolicy(launchType string, brief PublicTokenLaunchBriefRecord) string {
	source := strings.TrimSpace(brief.ResearchSource)
	if source == "" {
		source = "the submitted research source"
	}
	policyEvidence := tokenLaunchBriefJoinEvidence(brief.ProofPolicy, tokenLaunchBriefJoinEvidence(brief.WalletPolicy, tokenLaunchBriefJoinEvidence(brief.AllocationPolicy, brief.RiskNotes)))
	if launchType == "presale" {
		if policyEvidence != "" {
			return policyEvidence + " CEO must verify Solana contract proof and public ledger receipt from " + source + "."
		}
		return "CEO must verify utility, reserve cap, wallet ownership, funding reference, Solana contract proof, and public ledger receipt from " + source + "."
	}
	if policyEvidence != "" {
		return policyEvidence + " CEO must verify mission demand, anti-bot checks, wallet uniqueness, and public ledger receipt from " + source + "."
	}
	return "CEO must verify mission demand, useful work proof, anti-bot checks, wallet uniqueness, and public ledger receipt from " + source + "."
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
	rawRepositoryURL := strings.TrimSpace(req.RepositoryURL)
	repositoryURL := normalizeTokenWorkflowURL(rawRepositoryURL)
	if rawRepositoryURL == "" {
		return TokenLaunchBriefResponse{}, errors.New("repository_url is required for CEO launch research")
	}
	if repositoryURL == "" {
		return TokenLaunchBriefResponse{}, errors.New("repository_url must be an http(s) URL")
	}
	allocationPolicy := sanitizeTokenLaunchText(req.AllocationPolicy, 260)
	proofPolicy := sanitizeTokenLaunchText(req.ProofPolicy, 260)
	walletPolicy := sanitizeTokenLaunchText(req.WalletPolicy, 260)
	riskNotes := sanitizeTokenLaunchText(req.RiskNotes, 260)
	researchSignals := normalizeTokenLaunchResearchSignals(req.ResearchSignals, launchType, repositoryURL, proofPolicy, walletPolicy)
	ceoMemo := tokenLaunchCEOMemo(launchType, repositoryURL, allocationPolicy, proofPolicy, walletPolicy, riskNotes)
	briefID := s.newID("tlb")
	reference := tokenWorkflowReference([]string{
		"launch_brief:" + briefID,
		"type:" + launchType,
		"decision:" + ceoMemo.Decision,
		"gates:" + tokenLaunchGateReference(ceoMemo.Gates),
		"gate_summary:" + tokenLaunchGateSummary(ceoMemo.Gates),
		"title:" + projectTitle,
		"summary:" + projectSummary,
		"allocation_policy:" + allocationPolicy,
		"proof_policy:" + proofPolicy,
		"wallet_policy:" + walletPolicy,
		"risk_notes:" + riskNotes,
		"source:" + repositoryURL,
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
		CEOMemo:          ceoMemo,
		LedgerEntry:      entry,
		LedgerProofURL:   "/api/public/ledger/proof",
		LiveFeedURL:      "/api/public/live-feed",
		CreatedAt:        entry.CreatedAt,
	}, nil
}

func tokenLaunchCEOMemo(launchType, repositoryURL, allocationPolicy, proofPolicy, walletPolicy, riskNotes string) CEOMemo {
	if launchType == "presale" {
		return CEOMemo{
			Decision:      "pending_open_decision",
			DecisionLabel: "Presale window not open until CEO signs utility, reserve, wallet, contract, and ledger gates.",
			ReviewOwner:   "CEO token launch reviewer",
			NextAction:    "Write an open/no-open memo before accepting reservations as approved allocation.",
			Gates: []CEOMemoGate{
				{Key: "utility", Label: "Utility and reserve cap", Status: gateStatus(allocationPolicy), Required: true, Evidence: fallbackGateEvidence(allocationPolicy, "MRG utility, tier caps, and reserve limits")},
				{Key: "wallet", Label: "Wallet and funding rail", Status: gateStatus(walletPolicy), Required: true, Evidence: fallbackGateEvidence(walletPolicy, "Solana wallet, funding reference, and payer review")},
				{Key: "contract", Label: "Contract and ledger proof", Status: gateStatus(proofPolicy), Required: true, Evidence: fallbackGateEvidence(proofPolicy, "Solana contract reference, receipt hash, and public ledger proof")},
				{Key: "risk", Label: "Compliance and reversal risk", Status: gateStatus(riskNotes), Required: true, Evidence: fallbackGateEvidence(riskNotes, "Reserve caps, payment reversal, and compliance language review")},
			},
		}
	}
	return CEOMemo{
		Decision:      "pending_open_decision",
		DecisionLabel: "Airdrop missions not open until CEO signs demand, proof, anti-bot, wallet, and allocation gates.",
		ReviewOwner:   "CEO token launch reviewer",
		NextAction:    "Write an open/no-open memo before publishing claimable earned missions.",
		Gates: []CEOMemoGate{
			{Key: "source", Label: "Research source and mission demand", Status: gateStatus(repositoryURL), Required: true, Evidence: fallbackGateEvidence(repositoryURL, "Research URL, task backlog, mission-market fit")},
			{Key: "proof", Label: "Proof policy", Status: gateStatus(proofPolicy), Required: true, Evidence: fallbackGateEvidence(proofPolicy, "PR, task, QA, deployment, or agent evidence")},
			{Key: "wallet", Label: "Wallet uniqueness", Status: gateStatus(walletPolicy), Required: true, Evidence: fallbackGateEvidence(walletPolicy, "Solana wallet uniqueness and duplicate review")},
			{Key: "risk", Label: "Anti-bot and allocation risk", Status: gateStatus(riskNotes), Required: true, Evidence: fallbackGateEvidence(riskNotes, "Bot farming, duplicate wallets, and allocation cap review")},
		},
	}
}

func gateStatus(value string) string {
	if strings.TrimSpace(value) == "" {
		return "needs_evidence"
	}
	return "ready_for_review"
}

func fallbackGateEvidence(value, fallback string) string {
	if trimmed := strings.TrimSpace(value); trimmed != "" {
		return trimmed
	}
	return fallback
}

func tokenLaunchGateReference(gates []CEOMemoGate) string {
	parts := make([]string, 0, len(gates))
	for _, gate := range gates {
		if gate.Key == "" {
			continue
		}
		parts = append(parts, gate.Key+"="+gate.Status)
	}
	return strings.Join(parts, ",")
}

func tokenLaunchGateSummary(gates []CEOMemoGate) string {
	total := 0
	ready := 0
	needsEvidence := 0
	for _, gate := range gates {
		if gate.Key == "" {
			continue
		}
		total++
		switch gate.Status {
		case "ready_for_review":
			ready++
		case "needs_evidence":
			needsEvidence++
		}
	}
	if total == 0 {
		return "no gates recorded"
	}
	if needsEvidence == 0 {
		return fmt.Sprintf("%d/%d gates ready for CEO review", ready, total)
	}
	return fmt.Sprintf("%d/%d gates ready, %d need evidence", ready, total, needsEvidence)
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
		add("research_source")
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

func tokenWorkflowReferenceFields(reference string) map[string]string {
	fields := map[string]string{}
	for _, part := range strings.Split(reference, ";") {
		key, value, ok := strings.Cut(part, ":")
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if !ok || key == "" || value == "" {
			continue
		}
		fields[key] = value
	}
	return fields
}

func tokenWorkflowReferenceList(value string) []string {
	if strings.TrimSpace(value) == "" {
		return []string{}
	}
	parts := strings.Split(value, ",")
	items := make([]string, 0, len(parts))
	for _, part := range parts {
		item := strings.TrimSpace(part)
		if item != "" {
			items = append(items, item)
		}
	}
	return items
}

func tokenLaunchCandidateTypes(project *MarketplaceProject) []string {
	if project == nil {
		return []string{"airdrop"}
	}
	types := []string{"airdrop"}
	if project.WorkPoolCents >= minPresaleReserveMRG || project.AcceptedTaskCount > 0 {
		types = append(types, "presale")
	}
	return types
}

func tokenLaunchCandidateSignals(project *MarketplaceProject, bounties []*MarketplaceBounty) []string {
	signals := []string{"marketplace_project", "ceo_research_candidate"}
	if project != nil {
		if project.OpenTaskCount > 0 {
			signals = append(signals, "open_bounty_demand")
		}
		if project.AcceptedTaskCount > 0 {
			signals = append(signals, "accepted_delivery")
		}
		if strings.TrimSpace(project.RepoURL) != "" {
			signals = append(signals, "repository_context")
		}
	}
	seen := map[string]bool{}
	for _, bounty := range bounties {
		if strings.TrimSpace(bounty.SourceRepository) != "" && !seen["repository_context"] && !stringSliceContains(signals, "repository_context") {
			seen["repository_context"] = true
			signals = append(signals, "repository_context")
		}
		for _, evidence := range bounty.EvidenceRequired {
			normalized := normalizeTokenWorkflowID(evidence)
			if normalized != "" && !seen[normalized] {
				seen[normalized] = true
				signals = append(signals, normalized)
			}
		}
	}
	return signals
}

func tokenLaunchCandidateResearchScore(project *MarketplaceProject, proofSignals []string) int {
	if project == nil {
		return 42
	}
	openTaskCount := project.OpenTaskCount
	if openTaskCount > 8 {
		openTaskCount = 8
	}
	acceptedTaskCount := project.AcceptedTaskCount
	if acceptedTaskCount > 12 {
		acceptedTaskCount = 12
	}
	signalCount := len(proofSignals)
	if signalCount > 8 {
		signalCount = 8
	}
	poolScore := int(project.WorkPoolCents / 10000)
	if poolScore > 8 {
		poolScore = 8
	}
	score := 42 + (openTaskCount * 6) + (acceptedTaskCount * 3) + (signalCount * 5) + (poolScore * 2)
	if score > 98 {
		return 98
	}
	if score < 42 {
		return 42
	}
	return score
}

func tokenLaunchCandidateDecisionLaunchType(recommendedTypes []string, launchTypeFilter string) string {
	if launchTypeFilter == "airdrop" || launchTypeFilter == "presale" {
		return launchTypeFilter
	}
	launchType := "airdrop"
	if stringSliceContains(recommendedTypes, "presale") {
		launchType = "presale"
	}
	return launchType
}

func tokenLaunchCandidateNextAction(launchType string, score int, readyToOpen bool) string {
	if launchType == "presale" {
		if score >= 82 && readyToOpen {
			return "Open presale after CEO confirms utility, wallet, funding, contract, and receipt gates."
		}
		if score >= 82 {
			return "Draft a CEO presale memo and attach utility, wallet, funding, Solana contract, and receipt proof before opening."
		}
		return "Collect utility, reserve cap, wallet, funding, and Solana contract evidence before presale opens."
	}
	if score >= 82 && readyToOpen {
		return "Open earned missions after CEO confirms repo demand, anti-bot checks, wallet uniqueness, and proof gates."
	}
	if score >= 82 {
		return "Draft a CEO airdrop memo and attach repo demand, anti-bot, wallet uniqueness, and ledger proof before opening missions."
	}
	return "Collect repo demand, useful work proof, anti-bot policy, wallet uniqueness, and ledger evidence before missions open."
}

func tokenLaunchCandidateReadinessGates(launchType string, score int, project *MarketplaceProject, proofSignals []string) []TokenLaunchCandidateReadinessGate {
	openTasks := 0
	acceptedTasks := 0
	workPoolMRG := int64(0)
	if project != nil {
		openTasks = project.OpenTaskCount
		acceptedTasks = project.AcceptedTaskCount
		workPoolMRG = project.WorkPoolCents
	}
	signalCount := len(proofSignals)
	strong := score >= 82
	stateFrom := func(ok bool, review bool) string {
		if ok {
			return "ready"
		}
		if review {
			return "review"
		}
		return "hold"
	}
	signalEvidence := strings.Join(proofSignals[:minInt(len(proofSignals), 3)], ", ")
	if signalEvidence == "" {
		signalEvidence = "No public proof signal attached yet."
	}
	if launchType == "presale" {
		return []TokenLaunchCandidateReadinessGate{
			{
				Key:      "utility",
				Label:    "Utility",
				State:    stateFrom(signalCount >= 3, signalCount > 0),
				Value:    fmt.Sprintf("%d proof signals", signalCount),
				Evidence: signalEvidence,
			},
			{
				Key:      "reserve",
				Label:    "Reserve",
				State:    stateFrom(workPoolMRG >= 50000, workPoolMRG > 0),
				Value:    fmt.Sprintf("%s MRG pool", compactInt64(workPoolMRG)),
				Evidence: fmt.Sprintf("%d open tasks and %d accepted tasks", openTasks, acceptedTasks),
			},
			{
				Key:      "ceo_memo",
				Label:    "CEO memo",
				State:    stateFrom(false, strong || signalCount >= 3),
				Value:    ternaryString(strong, "Memo required", "Needs CEO proof"),
				Evidence: "CEO must write the presale memo after confirming Solana wallet, funding receipt, contract reference, and ledger proof.",
			},
		}
	}
	return []TokenLaunchCandidateReadinessGate{
		{
			Key:      "demand",
			Label:    "Demand",
			State:    stateFrom(openTasks > 0 || acceptedTasks > 0, signalCount > 0),
			Value:    fmt.Sprintf("%d open / %d accepted", openTasks, acceptedTasks),
			Evidence: fmt.Sprintf("%d proof signals attached", signalCount),
		},
		{
			Key:      "proof",
			Label:    "Proof",
			State:    stateFrom(signalCount >= 3, signalCount > 0),
			Value:    fmt.Sprintf("%d signals attached", signalCount),
			Evidence: signalEvidence,
		},
		{
			Key:      "ceo_memo",
			Label:    "CEO memo",
			State:    stateFrom(false, strong || signalCount >= 3),
			Value:    ternaryString(strong, "Memo required", "Needs policy"),
			Evidence: "CEO must write the airdrop memo after confirming wallet uniqueness, claim limits, and proof review.",
		},
	}
}

func tokenLaunchCandidateReadinessState(gates []TokenLaunchCandidateReadinessGate) string {
	if len(gates) == 0 {
		return "review"
	}
	allReady := true
	for _, gate := range gates {
		if gate.State == "hold" {
			return "hold"
		}
		if gate.State != "ready" {
			allReady = false
		}
	}
	if allReady {
		return "ready"
	}
	return "review"
}

func tokenLaunchCandidateDecisionOptions(launchType string, score int, readyToOpen bool) []TokenLaunchCandidateDecisionOption {
	if launchType != "presale" {
		launchType = "airdrop"
	}
	launchLabel := launchType
	approveLabel := "Draft missions"
	approveProof := "Approve only with repo task evidence, useful work proof, anti-bot review, wallet uniqueness, and public ledger receipt."
	needsEvidenceProof := "Hold airdrop until repo task, PR/deploy proof, QA evidence, and wallet uniqueness are attached."
	if launchType == "presale" {
		approveLabel = "Draft presale"
		approveProof = "Approve only with utility proof, reserve cap, Solana wallet path, funding reference, contract proof, and public ledger receipt."
		needsEvidenceProof = "Hold presale until utility, funding, wallet, contract, and receipt evidence are attached."
	}
	if score >= 82 && readyToOpen {
		if launchType == "presale" {
			approveLabel = "Open presale"
		} else {
			approveLabel = "Open missions"
		}
	}
	approveRisk := fmt.Sprintf("CEO %s decision: draft memo first; score %d%% fit but launch is not open until CEO memo and ledger proof are attached.", launchLabel, score)
	if readyToOpen {
		approveRisk = fmt.Sprintf("CEO %s decision: ready to open after final proof review; score %d%% fit.", launchLabel, score)
	}
	return []TokenLaunchCandidateDecisionOption{
		{
			Key:         "approve",
			Label:       approveLabel,
			Tone:        "approve",
			ProofPolicy: approveProof,
			RiskNotes:   approveRisk,
		},
		{
			Key:         "needs_evidence",
			Label:       "Needs evidence",
			Tone:        "evidence",
			ProofPolicy: needsEvidenceProof,
			RiskNotes:   fmt.Sprintf("CEO %s decision: request more evidence before opening; score %d%% fit.", launchLabel, score),
		},
		{
			Key:         "reject",
			Label:       "Reject",
			Tone:        "reject",
			ProofPolicy: "Do not open token workflow until source, demand, proof quality, wallet policy, and ledger evidence are remediated.",
			RiskNotes:   fmt.Sprintf("CEO %s decision: reject for now; score %d%% fit and proof is not launch-ready.", launchLabel, score),
		},
	}
}

func tokenLaunchCandidateProofPolicy(bounties []*MarketplaceBounty) string {
	evidence := []string{}
	seen := map[string]bool{}
	for _, bounty := range bounties {
		for _, item := range bounty.EvidenceRequired {
			normalized := strings.TrimSpace(item)
			if normalized != "" && !seen[normalized] {
				seen[normalized] = true
				evidence = append(evidence, normalized)
			}
		}
	}
	if len(evidence) == 0 {
		return "Require task evidence, repository context, review notes, and public ledger proof before opening token workflows."
	}
	return "Require " + strings.Join(evidence, ", ") + " plus public ledger proof before opening token workflows."
}

func compactInt64(value int64) string {
	if value >= 1000000 {
		return fmt.Sprintf("%dM", value/1000000)
	}
	if value >= 1000 {
		return fmt.Sprintf("%dK", value/1000)
	}
	return fmt.Sprintf("%d", value)
}

func ternaryString(condition bool, yes string, no string) string {
	if condition {
		return yes
	}
	return no
}
