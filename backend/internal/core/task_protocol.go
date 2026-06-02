package core

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

func (s *Store) PublicTaskProtocol(limit int) PublicTaskProtocolResponse {
	marketplace := s.Marketplace()
	limit = normalizePublicLiveFeedLimit(limit)
	if len(marketplace.Bounties) > limit {
		marketplace.Bounties = marketplace.Bounties[:limit]
	}

	tasks := make([]TaskProtocolDocument, 0, len(marketplace.Bounties))
	for _, bounty := range marketplace.Bounties {
		tasks = append(tasks, publicTaskProtocolDocument(bounty))
	}
	return PublicTaskProtocolResponse{
		Stats: marketplace.Stats,
		Tasks: tasks,
	}
}

func publicTaskProtocolDocument(bounty *MarketplaceBounty) TaskProtocolDocument {
	workerKind := bounty.RequiredWorkerKind
	if workerKind == "" {
		workerKind = WorkerHuman
	}
	complexity := publicTaskComplexity(bounty)
	riskLevel := publicTaskRiskLevel(bounty, complexity)
	metadata := map[string]any{
		"claim_id":      bounty.ClaimID,
		"issue_number":  bounty.IssueNumber,
		"project_title": bounty.ProjectTitle,
		"created_at":    bounty.CreatedAt,
		"analysis": map[string]any{
			"complexity": complexity,
			"risk_level": riskLevel,
		},
	}
	if bounty.EstimatedHours > 0 {
		metadata["estimated_hours"] = bounty.EstimatedHours
	}
	return TaskProtocolDocument{
		ProtocolVersion:    "mergeos.task.v1",
		Kind:               "task",
		ID:                 publicTaskProtocolID(bounty.ClaimID),
		ProjectID:          strings.TrimSpace(bounty.ProjectID),
		Title:              protocolText(bounty.Title, 240, "Untitled bounty"),
		Summary:            protocolText(bounty.Acceptance, 2000, ""),
		IssueURL:           protocolText(bounty.IssueURL, 512, ""),
		RewardMRG:          float64(bounty.RewardCents) / 100,
		EstimatedHours:     bounty.EstimatedHours,
		Complexity:         complexity,
		RiskLevel:          riskLevel,
		BountyType:         protocolText(bounty.BountyType, 80, ""),
		WorkerKind:         workerKind,
		AgentType:          protocolText(bounty.SuggestedAgentType, 120, ""),
		AcceptanceCriteria: publicTaskAcceptanceCriteria(bounty.Acceptance),
		EvidenceRequired:   publicTaskEvidenceRequired(bounty),
		Tags:               publicTaskProtocolTags(bounty),
		Metadata:           metadata,
	}
}

func publicTaskComplexity(bounty *MarketplaceBounty) string {
	haystack := strings.ToLower(strings.Join([]string{
		bounty.Title,
		bounty.Acceptance,
		bounty.BountyType,
		bounty.SuggestedAgentType,
	}, " "))
	for _, value := range []string{"high", "medium", "low"} {
		if strings.Contains(haystack, "complexity: "+value) || strings.Contains(haystack, value+" complexity") {
			return value
		}
	}
	if bounty.EstimatedHours >= 12 || bounty.RewardCents >= 100000 {
		return "high"
	}
	if bounty.EstimatedHours >= 5 || bounty.RewardCents >= 40000 {
		return "medium"
	}
	return "low"
}

func publicTaskRiskLevel(bounty *MarketplaceBounty, complexity string) string {
	haystack := strings.ToLower(strings.Join([]string{
		bounty.Title,
		bounty.Acceptance,
		bounty.BountyType,
		bounty.SuggestedAgentType,
	}, " "))
	switch {
	case containsAny(haystack, []string{"payment", "paypal", "usdt", "crypto", "escrow", "payout", "webhook", "secret", "token", "auth", "security"}):
		return "high"
	case complexity == "high" || containsAny(haystack, []string{"deploy", "deployment", "admin", "database", "ledger", "api"}):
		return "medium"
	default:
		return "low"
	}
}

func publicTaskProtocolID(claimID string) string {
	id := strings.TrimSpace(claimID)
	if len(id) >= 3 && len(id) <= 96 {
		return id
	}
	sum := sha256.Sum256([]byte(id))
	return "tsk_" + hex.EncodeToString(sum[:14])
}

func publicTaskAcceptanceCriteria(acceptance string) []string {
	acceptance = protocolText(acceptance, 500, "")
	if acceptance == "" {
		return []string{"Submit implementation evidence and passing checks."}
	}
	parts := strings.FieldsFunc(acceptance, func(r rune) bool {
		return r == '\n' || r == '\r' || r == ';'
	})
	criteria := []string{}
	for _, part := range parts {
		value := protocolText(part, 500, "")
		if value != "" {
			criteria = append(criteria, value)
		}
		if len(criteria) == 5 {
			break
		}
	}
	if len(criteria) == 0 {
		return []string{"Submit implementation evidence and passing checks."}
	}
	return criteria
}

func publicTaskEvidenceRequired(bounty *MarketplaceBounty) []string {
	haystack := strings.ToLower(strings.Join([]string{
		bounty.Title,
		bounty.Acceptance,
		bounty.BountyType,
		bounty.SuggestedAgentType,
	}, " "))
	required := []string{"tests"}
	if containsAny(haystack, []string{"ui", "ux", "design", "frontend", "page", "screen", "layout"}) {
		required = append(required, "screenshot")
	}
	if containsAny(haystack, []string{"deploy", "preview", "release", "vercel", "pipeline"}) {
		required = append(required, "deploy_preview")
	}
	if containsAny(haystack, []string{"payment", "paypal", "usdt", "crypto", "transaction", "ledger", "escrow", "payout"}) {
		required = append(required, "transaction_proof", "security_review")
	}
	if containsAny(haystack, []string{"security", "auth", "password", "secret", "env", "admin", "webhook"}) {
		required = append(required, "security_review")
	}
	return stableStrings(required)
}

func publicTaskProtocolTags(bounty *MarketplaceBounty) []string {
	values := []string{
		string(bounty.RequiredWorkerKind),
		bounty.SuggestedAgentType,
		bounty.BountyType,
	}
	haystack := strings.ToLower(strings.Join([]string{bounty.Title, bounty.Acceptance}, " "))
	for _, candidate := range []string{"frontend", "backend", "design", "security", "payment", "deployment", "ai", "ledger", "marketplace"} {
		if strings.Contains(haystack, candidate) {
			values = append(values, candidate)
		}
	}
	return stableStrings(values)
}

func protocolText(value string, maxLength int, fallback string) string {
	value = compactText(value)
	if value == "" {
		value = fallback
	}
	if maxLength > 0 && len(value) > maxLength {
		runes := []rune(value)
		if len(runes) > maxLength {
			return strings.TrimSpace(string(runes[:maxLength]))
		}
	}
	return value
}

func stableStrings(values []string) []string {
	seen := map[string]bool{}
	result := []string{}
	for _, value := range values {
		value = strings.ToLower(strings.TrimSpace(value))
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		result = append(result, value)
	}
	return result
}
