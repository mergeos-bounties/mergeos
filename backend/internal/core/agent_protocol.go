package core

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

func (s *Store) PublicAgentProtocol(limit int) PublicAgentProtocolResponse {
	marketplace := s.Marketplace()
	limit = normalizePublicLiveFeedLimit(limit)
	if len(marketplace.Agents) > limit {
		marketplace.Agents = marketplace.Agents[:limit]
	}

	openTaskIDs := map[string][]string{}
	for _, bounty := range marketplace.Bounties {
		agentType := strings.TrimSpace(bounty.SuggestedAgentType)
		if !bountyNeedsAgent(bounty) || strings.TrimSpace(bounty.ClaimID) == "" {
			continue
		}
		publicID := publicTaskProtocolID(bounty.ClaimID)
		appendOpenTaskID(openTaskIDs, ceoAgentType, publicID, 5)
		appendOpenTaskID(openTaskIDs, designReviewAgentType, publicID, 5)
		if agentType == "" {
			agentType = "general-ai-agent"
		}
		if len(openTaskIDs[agentType]) >= 5 {
			continue
		}
		openTaskIDs[agentType] = append(openTaskIDs[agentType], publicID)
	}

	agents := make([]AgentProtocolDocument, 0, len(marketplace.Agents))
	for _, agent := range marketplace.Agents {
		agents = append(agents, publicAgentProtocolDocument(agent, openTaskIDs[agent.Type], marketplace.Stats.TokenSymbol))
	}
	return PublicAgentProtocolResponse{
		Stats:  marketplace.Stats,
		Agents: agents,
	}
}

func appendOpenTaskID(rows map[string][]string, agentType, taskID string, limit int) {
	if strings.TrimSpace(agentType) == "" || strings.TrimSpace(taskID) == "" || len(rows[agentType]) >= limit {
		return
	}
	rows[agentType] = append(rows[agentType], taskID)
}

func publicAgentProtocolDocument(agent *MarketplaceAgent, openTaskIDs []string, tokenSymbol string) AgentProtocolDocument {
	workerKind := agent.WorkerKind
	if workerKind == "" {
		workerKind = WorkerAgent
	}
	status := "standby"
	if agent.OpenTaskCount > 0 || len(openTaskIDs) > 0 {
		status = "active"
	}
	actions := publicAgentSupportedActions(agent)
	capabilities := publicAgentCapabilities(agent, actions)
	return AgentProtocolDocument{
		ProtocolVersion: "mergeos.agent.v1",
		Kind:            "agent",
		ID:              publicAgentProtocolID(agent.Type),
		Type:            protocolText(agent.Type, 120, "ai-agent"),
		Title:           protocolText(agent.Title, 160, marketplaceTitle(agent.Type)),
		WorkerKind:      workerKind,
		Role:            protocolText(agent.Role, 80, ""),
		ParentAgentType: protocolText(agent.ParentAgentType, 120, ""),
		SubagentTypes:   append([]string(nil), agent.SubagentTypes...),
		DelegationEndpoint: protocolText(
			agent.DelegationEndpoint,
			240,
			"",
		),
		Focus:            append([]string(nil), agent.Focus...),
		SupportedActions: actions,
		Capabilities:     capabilities,
		TaskCount:        agent.TaskCount,
		OpenTaskCount:    agent.OpenTaskCount,
		BudgetMRG:        float64(agent.BudgetCents) / 100,
		Status:           status,
		OpenTaskIDs:      openTaskIDs,
		Tags:             publicAgentProtocolTags(agent, status, capabilities),
		Metadata: map[string]any{
			"event_protocol":         "mergeos.event.v1",
			"event_stream_endpoint":  "WS /api/ws",
			"task_protocol_endpoint": "GET /api/public/protocol/tasks",
			"agent_queue_endpoint":   agentQueueEndpoint,
			"agent_role":             protocolText(agent.Role, 80, "subagent"),
			"parent_agent_type":      protocolText(agent.ParentAgentType, 120, ""),
			"subagent_types":         append([]string(nil), agent.SubagentTypes...),
			"token_symbol":           normalizedTokenSymbol(tokenSymbol),
			"queue_depth":            len(openTaskIDs),
		},
	}
}

func publicAgentProtocolID(agentType string) string {
	agentType = strings.ToLower(strings.TrimSpace(agentType))
	var normalized strings.Builder
	lastUnderscore := false
	for _, r := range agentType {
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
		return "agt_" + id
	}
	sum := sha256.Sum256([]byte(agentType))
	return "agt_" + hex.EncodeToString(sum[:14])
}

func publicAgentSupportedActions(agent *MarketplaceAgent) []string {
	haystack := strings.ToLower(strings.Join([]string{
		agent.Type,
		agent.Title,
		string(agent.WorkerKind),
	}, " "))
	actions := []string{}
	if containsAny(haystack, []string{"review", "security", "audit"}) {
		actions = append(actions, "review")
	}
	if containsAny(haystack, []string{"qa", "test", "quality", "validation"}) {
		actions = append(actions, "test")
	}
	if containsAny(haystack, []string{"generate", "generator", "build", "code", "coding", "frontend", "backend", "design", "ui", "ux"}) {
		actions = append(actions, "generate")
	}
	if containsAny(haystack, []string{"deploy", "devops", "release", "pipeline"}) {
		actions = append(actions, "deploy")
	}
	if containsAny(haystack, []string{"scan", "repo", "dependency", "secret", "security", "debt"}) {
		actions = append(actions, "scan")
	}
	if containsAny(haystack, []string{"ceo", "strategy", "planner", "orchestration"}) {
		actions = append(actions, "review", "generate", "scan")
	}
	if len(actions) == 0 {
		actions = []string{"review", "test", "generate"}
	}
	return stableStrings(actions)
}

func publicAgentCapabilities(agent *MarketplaceAgent, actions []string) []string {
	values := []string{"task_intake", "evidence_reporting"}
	if strings.EqualFold(agent.Role, "ceo_planner") || strings.EqualFold(agent.Type, ceoAgentType) {
		values = append(values, "idea_generation", "task_decomposition", "subagent_delegation", "quality_gate")
	}
	if strings.EqualFold(agent.Type, designReviewAgentType) {
		values = append(values, "ux_review", "responsive_design", "visual_quality")
	}
	for _, action := range actions {
		switch action {
		case "review":
			values = append(values, "code_review", "security_review")
		case "test":
			values = append(values, "qa_validation", "smoke_testing")
		case "generate":
			values = append(values, "implementation_generation")
		case "deploy":
			values = append(values, "deployment_validation", "release_handoff")
		case "scan":
			values = append(values, "repository_scan", "dependency_scan")
		}
	}
	if agent.OpenTaskCount > 0 {
		values = append(values, "open_bounty_matching")
	}
	return stableStrings(values)
}

func publicAgentProtocolTags(agent *MarketplaceAgent, status string, capabilities []string) []string {
	values := []string{"ai", "agent", string(agent.WorkerKind), agent.Type, status}
	values = append(values, agent.Role, agent.ParentAgentType)
	values = append(values, agent.SubagentTypes...)
	values = append(values, agent.Focus...)
	values = append(values, capabilities...)
	return stableStrings(values)
}
