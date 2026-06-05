package core

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

func (s *Store) PublicContributorProtocol(limit int) PublicContributorProtocolResponse {
	marketplace := s.Marketplace()
	limit = normalizePublicLiveFeedLimit(limit)
	if len(marketplace.Contributors) > limit {
		marketplace.Contributors = marketplace.Contributors[:limit]
	}

	matchedTaskIDs := map[string][]string{}
	for _, contributor := range marketplace.Contributors {
		for _, bounty := range marketplace.Bounties {
			if !publicContributorMatchesBounty(contributor, bounty) {
				continue
			}
			if len(matchedTaskIDs[contributor.WorkerID]) >= 5 {
				break
			}
			matchedTaskIDs[contributor.WorkerID] = append(matchedTaskIDs[contributor.WorkerID], publicTaskProtocolID(bounty.ClaimID))
		}
	}

	contributors := make([]ContributorProtocolDocument, 0, len(marketplace.Contributors))
	for _, contributor := range marketplace.Contributors {
		contributors = append(contributors, publicContributorProtocolDocument(contributor, matchedTaskIDs[contributor.WorkerID], marketplace.Stats.TokenSymbol))
	}
	return PublicContributorProtocolResponse{
		Stats:        marketplace.Stats,
		Contributors: contributors,
	}
}

func publicContributorProtocolDocument(contributor *MarketplaceContributor, matchedTaskIDs []string, tokenSymbol string) ContributorProtocolDocument {
	workerKind := contributor.Kind
	if workerKind == "" {
		workerKind = WorkerHuman
	}
	capabilities := publicContributorCapabilities(contributor)
	return ContributorProtocolDocument{
		ProtocolVersion:    "mergeos.contributor.v1",
		Kind:               "contributor",
		ID:                 publicContributorProtocolID(contributor.WorkerID),
		WorkerID:           protocolText(contributor.WorkerID, 160, ""),
		DisplayName:        protocolText(contributor.Name, 160, marketplaceTitle(contributor.WorkerID)),
		WorkerKind:         workerKind,
		AgentType:          protocolText(contributor.AgentType, 120, ""),
		CompletedTaskCount: contributor.TaskCount,
		EarnedMRG:          float64(contributor.EarnedCents) / 100,
		ReputationScore:    contributor.ReputationScore,
		ReputationLevel:    protocolText(contributor.ReputationLevel, 80, "new"),
		RiskLevel:          protocolText(contributor.RiskLevel, 40, "unknown"),
		LastPaidAt:         contributor.LastPaidAt,
		MatchedTaskIDs:     matchedTaskIDs,
		Capabilities:       capabilities,
		Flags:              stableStrings(contributor.Flags),
		Tags:               publicContributorProtocolTags(contributor, capabilities),
		Metadata: map[string]any{
			"token_symbol":              normalizedTokenSymbol(tokenSymbol),
			"marketplace_endpoint":      "GET /api/public/marketplace",
			"task_protocol_endpoint":    "GET /api/public/protocol/tasks",
			"event_stream_endpoint":     "WS /api/ws",
			"matched_open_task_count":   len(matchedTaskIDs),
			"public_reputation_version": "mergeos.contributor.v1",
		},
	}
}

func publicContributorProtocolID(workerID string) string {
	workerID = strings.ToLower(strings.TrimSpace(workerID))
	var normalized strings.Builder
	lastUnderscore := false
	for _, r := range workerID {
		isAlpha := r >= 'a' && r <= 'z'
		isDigit := r >= '0' && r <= '9'
		if isAlpha || isDigit {
			normalized.WriteRune(r)
			lastUnderscore = false
			continue
		}
		if !lastUnderscore {
			normalized.WriteRune('_')
			lastUnderscore = true
		}
	}
	id := strings.Trim(normalized.String(), "_")
	if len(id) >= 3 && len(id) <= 80 {
		return "ctr_" + id
	}
	sum := sha256.Sum256([]byte(workerID))
	return "ctr_" + hex.EncodeToString(sum[:14])
}

func publicContributorMatchesBounty(contributor *MarketplaceContributor, bounty *MarketplaceBounty) bool {
	if contributor == nil || bounty == nil {
		return false
	}
	kind := contributor.Kind
	bountyKind := bounty.RequiredWorkerKind
	if strings.TrimSpace(contributor.AgentType) != "" && strings.EqualFold(strings.TrimSpace(contributor.AgentType), strings.TrimSpace(bounty.SuggestedAgentType)) {
		return true
	}
	return kind == WorkerHybrid || bountyKind == WorkerHybrid || kind == bountyKind
}

func publicContributorCapabilities(contributor *MarketplaceContributor) []string {
	values := []string{"bounty_delivery", "evidence_reporting"}
	switch contributor.Kind {
	case WorkerAgent:
		values = append(values, "automated_execution")
	case WorkerHybrid:
		values = append(values, "human_agent_collaboration")
	default:
		values = append(values, "human_delivery")
	}
	haystack := strings.ToLower(strings.Join([]string{contributor.Name, contributor.AgentType, contributor.ReputationLevel}, " "))
	if containsAny(haystack, []string{"security", "audit", "review"}) {
		values = append(values, "security_review", "code_review")
	}
	if containsAny(haystack, []string{"frontend", "design", "ui", "ux"}) {
		values = append(values, "frontend_delivery")
	}
	if containsAny(haystack, []string{"backend", "api", "database"}) {
		values = append(values, "backend_delivery")
	}
	if contributor.ReputationScore >= 85 && strings.EqualFold(contributor.RiskLevel, "low") {
		values = append(values, "trusted_release_candidate")
	}
	return stableStrings(values)
}

func publicContributorProtocolTags(contributor *MarketplaceContributor, capabilities []string) []string {
	values := []string{
		"contributor",
		string(contributor.Kind),
		contributor.AgentType,
		contributor.ReputationLevel,
		contributor.RiskLevel,
	}
	values = append(values, capabilities...)
	values = append(values, contributor.Flags...)
	return stableStrings(values)
}
