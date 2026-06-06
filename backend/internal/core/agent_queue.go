package core

import (
	"fmt"
	"sort"
	"strings"
)

const (
	ceoAgentType          = "ceo-strategy-agent"
	designReviewAgentType = "design-review-agent"
	agentQueueEndpoint    = "/api/public/protocol/agent-queue"
	agentLeaseEndpoint    = "/api/agent-queue/leases"
	agentLeaseTTLSeconds  = 900
	agentHeartbeatSeconds = 120
)

func (s *Store) PublicAgentQueue(limit int) AgentQueueResponse {
	marketplace := s.Marketplace()
	limit = normalizePublicLiveFeedLimit(limit)

	tasks := []AgentQueueTask{}
	queueDepth := map[string]int{}
	for _, bounty := range marketplace.Bounties {
		if !bountyNeedsAgent(bounty) {
			continue
		}
		row := agentQueueTaskRow(bounty)
		tasks = append(tasks, row)
		queueDepth[ceoAgentType]++
		queueDepth[designReviewAgentType]++
		if row.AgentType != "" {
			queueDepth[row.AgentType]++
		}
		if len(tasks) >= limit {
			break
		}
	}

	agents := make([]AgentQueueAgent, 0, len(marketplace.Agents))
	for _, agent := range marketplace.Agents {
		agentType := strings.TrimSpace(agent.Type)
		document := publicAgentProtocolDocument(agent, nil, marketplace.Stats.TokenSymbol)
		status := "standby"
		if agent.OpenTaskCount > 0 || queueDepth[agentType] > 0 {
			status = "active"
		}
		agents = append(agents, AgentQueueAgent{
			Type:               agentType,
			Title:              agent.Title,
			WorkerKind:         agent.WorkerKind,
			Role:               agent.Role,
			ParentAgentType:    agent.ParentAgentType,
			SubagentTypes:      append([]string(nil), agent.SubagentTypes...),
			DelegationEndpoint: agent.DelegationEndpoint,
			Focus:              append([]string(nil), agent.Focus...),
			TaskCount:          agent.TaskCount,
			OpenTaskCount:      agent.OpenTaskCount,
			BudgetCents:        agent.BudgetCents,
			Status:             status,
			SupportedActions:   document.SupportedActions,
			QueueDepth:         queueDepth[agentType],
		})
	}
	sort.Slice(agents, func(i, j int) bool {
		if agents[i].QueueDepth == agents[j].QueueDepth {
			return agents[i].Type < agents[j].Type
		}
		return agents[i].QueueDepth > agents[j].QueueDepth
	})

	stats := AgentQueueStats{
		TotalCount:  len(tasks),
		AgentCount:  len(agents),
		TokenSymbol: marketplace.Stats.TokenSymbol,
		UpdatedAt:   marketplace.Stats.UpdatedAt,
	}
	for _, task := range tasks {
		stats.ReadyCount++
		stats.RewardCents += task.RewardCents
	}
	return AgentQueueResponse{
		ProtocolVersion: "mergeos.agent-queue.v1",
		Kind:            "agent_queue",
		Stats:           stats,
		Agents:          agents,
		Tasks:           tasks,
	}
}

func bountyNeedsAgent(bounty *MarketplaceBounty) bool {
	if bounty == nil {
		return false
	}
	return strings.TrimSpace(bounty.SuggestedAgentType) != "" || bounty.RequiredWorkerKind == WorkerAgent || bounty.RequiredWorkerKind == WorkerHybrid
}

func agentQueueTaskRow(bounty *MarketplaceBounty) AgentQueueTask {
	bountyID := strings.TrimSpace(bounty.ClaimID)
	if bountyID == "" {
		bountyID = strings.TrimSpace(bounty.ID)
	}
	agentType := strings.TrimSpace(bounty.SuggestedAgentType)
	if agentType == "" && bounty.RequiredWorkerKind == WorkerAgent {
		agentType = "general-ai-agent"
	}
	claimEndpoint := "/api/tasks/" + bountyID + "/claim"
	submitEndpoint := "/api/tasks/" + bountyID + "/submit"
	actionEndpoint := "/api/projects/" + bounty.ProjectID + "/agent-actions"
	protocolURL := "/api/public/protocol/tasks?task_id=" + bountyID
	contextURLs := map[string]string{
		"task_protocol":     protocolURL,
		"agent_queue":       agentQueueEndpoint,
		"workflow_protocol": "/api/public/projects/" + bounty.ProjectID + "/workflow",
		"workflow_pulse":    "/api/public/projects/" + bounty.ProjectID + "/ai-workflow",
		"pr_monitor":        "/api/public/projects/" + bounty.ProjectID + "/pull-requests",
		"ceo_agent":         "/api/public/protocol/agents",
		"design_review":     agentQueueEndpoint + "#design-review-agent",
	}
	if bounty.IssueURL != "" {
		contextURLs["issue"] = bounty.IssueURL
	}
	if bounty.SourceRepository != "" {
		contextURLs["repository"] = bounty.SourceRepository
	}
	workPacket := AgentWorkPacket{
		ClaimEndpoint:       claimEndpoint,
		ActionEndpoint:      actionEndpoint,
		SubmitEndpoint:      submitEndpoint,
		LeasePacket:         agentLeasePacket(bountyID, agentType),
		SupervisorAgentType: ceoAgentType,
		SubagentType:        agentType,
		DesignReviewAgent:   designReviewAgentType,
		DelegationChain:     agentDelegationChain(agentType),
		ContextURLs:         contextURLs,
		Runbook: []AgentRunbookStep{
			{Step: 1, Action: "fetch_context", Label: "Fetch task protocol", Method: "GET", Endpoint: protocolURL},
			{Step: 2, Action: "plan_scope", Label: "CEO agent decomposes work and delegates subagents", Method: "GET", Endpoint: agentQueueEndpoint},
			{Step: 3, Action: "design_review", Label: "Design Review Agent checks UX, responsive layout, and visual quality", Method: "POST", Endpoint: actionEndpoint},
			{Step: 4, Action: "claim_task", Label: "Claim bounty lane", Method: "POST", Endpoint: claimEndpoint},
			{Step: 5, Action: "run_checks", Label: "Run review, test, generation, or deployment checks", Method: "POST", Endpoint: actionEndpoint},
			{Step: 6, Action: "attach_evidence", Label: "Attach agent check evidence to the live log", Method: "POST", Endpoint: actionEndpoint},
			{Step: 7, Action: "submit_review", Label: "Submit final PR and review evidence", Method: "POST", Endpoint: submitEndpoint},
		},
		ActionPayloads:  agentQueueActionPayloads(bounty, actionEndpoint, contextURLs),
		OutputContracts: agentQueueOutputContracts(bounty, actionEndpoint, submitEndpoint, contextURLs),
	}
	return AgentQueueTask{
		ID:               bountyID,
		BountyID:         bountyID,
		ProjectID:        bounty.ProjectID,
		ProjectTitle:     bounty.ProjectTitle,
		IssueNumber:      bounty.IssueNumber,
		Title:            bounty.Title,
		Summary:          protocolText(bounty.Acceptance, 320, "Protocol-ready task for an AI agent lane."),
		RewardCents:      bounty.RewardCents,
		WorkerKind:       bounty.RequiredWorkerKind,
		AgentType:        agentType,
		Readiness:        "agent_ready",
		EvidenceRequired: publicTaskEvidenceRequired(bounty),
		ClaimEndpoint:    claimEndpoint,
		ActionEndpoint:   actionEndpoint,
		ProtocolURL:      protocolURL,
		WorkPacket:       workPacket,
	}
}

func agentLeasePacket(claimID, agentType string) AgentLeasePacket {
	claimID = strings.TrimSpace(claimID)
	agentType = strings.TrimSpace(agentType)
	if agentType == "" {
		agentType = "general-ai-agent"
	}
	return AgentLeasePacket{
		LeaseEndpoint:     agentLeaseEndpoint,
		HeartbeatEndpoint: agentLeaseEndpoint,
		Method:            "POST",
		TTLSeconds:        agentLeaseTTLSeconds,
		HeartbeatSeconds:  agentHeartbeatSeconds,
		Payload: map[string]any{
			"claim_id":   claimID,
			"bounty_id":  claimID,
			"agent_type": agentType,
			"status":     "leased",
		},
	}
}

func agentDelegationChain(agentType string) []string {
	chain := []string{ceoAgentType, designReviewAgentType}
	agentType = strings.TrimSpace(agentType)
	if agentType != "" && agentType != ceoAgentType && agentType != designReviewAgentType {
		chain = append(chain, agentType)
	}
	return chain
}

func agentQueueActionPayloads(bounty *MarketplaceBounty, endpoint string, contextURLs map[string]string) []AgentActionPayload {
	actions := agentQueueActions(bounty)
	rows := make([]AgentActionPayload, 0, len(actions))
	for _, action := range actions {
		agentType := protocolText(bounty.SuggestedAgentType, 120, "general-ai-agent")
		rows = append(rows, AgentActionPayload{
			Action:   action,
			Label:    marketplaceTitle(action),
			Method:   "POST",
			Endpoint: endpoint,
			Body: map[string]any{
				"action":           action,
				"status":           "queued",
				"project_id":       bounty.ProjectID,
				"claim_id":         bounty.ClaimID,
				"bounty_id":        bounty.ClaimID,
				"agent_type":       agentType,
				"delegated_by":     ceoAgentType,
				"design_agent":     designReviewAgentType,
				"subagent_type":    agentType,
				"delegation_chain": agentDelegationChain(agentType),
				"reference_url":    bounty.IssueURL,
				"context_urls": []string{
					contextURLs["task_protocol"],
					contextURLs["workflow_protocol"],
					contextURLs["workflow_pulse"],
					contextURLs["design_review"],
				},
				"evidence": publicTaskEvidenceRequired(bounty),
				"runbook": []string{
					"Fetch task protocol",
					"Let CEO Strategy Agent split scope and select subagents",
					"Run Design Review Agent for UX, responsive, and visual quality checks",
					"Claim or reserve the bounty lane",
					"Run scoped checks",
					"Record evidence",
				},
			},
		})
	}
	return rows
}

func agentQueueActions(bounty *MarketplaceBounty) []string {
	haystack := strings.ToLower(strings.Join([]string{
		bounty.Title,
		bounty.Acceptance,
		bounty.BountyType,
		bounty.SuggestedAgentType,
	}, " "))
	actions := []string{"review", "test"}
	if containsAny(haystack, []string{"build", "generate", "frontend", "backend", "fix", "implementation", "page", "code"}) {
		actions = append(actions, "generate")
	}
	if containsAny(haystack, []string{"deploy", "pipeline", "release"}) {
		actions = append(actions, "deploy")
	}
	if containsAny(haystack, []string{"scan", "dependency", "secret", "security"}) {
		actions = append(actions, "scan")
	}
	return stableStrings(actions)
}

func agentActionsForTask(task *Task) []string {
	if task == nil {
		return []string{"review", "test"}
	}
	haystack := strings.ToLower(strings.Join([]string{
		task.Title,
		task.Acceptance,
		task.BountyType,
		task.SuggestedAgentType,
		task.AgentType,
	}, " "))
	actions := []string{"review", "test"}
	if containsAny(haystack, []string{"build", "generate", "frontend", "backend", "fix", "implementation", "page", "code"}) {
		actions = append(actions, "generate")
	}
	if containsAny(haystack, []string{"deploy", "pipeline", "release"}) {
		actions = append(actions, "deploy")
	}
	if containsAny(haystack, []string{"scan", "dependency", "secret", "security"}) {
		actions = append(actions, "scan")
	}
	return stableStrings(actions)
}

func agentQueueOutputContracts(bounty *MarketplaceBounty, actionEndpoint, submitEndpoint string, contextURLs map[string]string) []AgentOutputContract {
	actions := agentQueueActions(bounty)
	rows := make([]AgentOutputContract, 0, len(actions)+1)
	for _, action := range actions {
		rows = append(rows, agentQueueOutputContract(action, bounty.ProjectID, actionEndpoint, contextURLs))
	}
	rows = append(rows, AgentOutputContract{
		Action:            "submit",
		ArtifactKind:      "task_submission",
		OutputEndpoint:    submitEndpoint,
		OutputProtocol:    "mergeos.task-submission.v1",
		OutputProtocolURL: "/protocol/task-submission.v1.schema.json",
		PublicURL:         contextURLs["task_protocol"],
	})
	return rows
}

func agentQueueOutputContract(action, projectID, actionEndpoint string, contextURLs map[string]string) AgentOutputContract {
	contract := AgentOutputContract{
		Action:            action,
		ArtifactKind:      "agent_evidence",
		OutputEndpoint:    actionEndpoint,
		OutputProtocol:    "mergeos.agent-action.v1",
		OutputProtocolURL: "/protocol/agent-action.v1.schema.json",
		PublicURL:         "/api/public/live-feed",
	}
	switch action {
	case "review":
		contract.ArtifactKind = "pr_review"
		contract.PublicURL = contextURLs["pr_monitor"]
	case "test":
		contract.ArtifactKind = "test_evidence"
		contract.PublicURL = contextURLs["workflow_pulse"]
	case "generate":
		contract.ArtifactKind = "generated_work"
		contract.PublicURL = contextURLs["workflow_protocol"]
	case "deploy":
		contract.ArtifactKind = "deployment_evidence"
		contract.PublicURL = "/api/public/projects/" + strings.TrimSpace(projectID) + "/deployment"
	case "scan":
		contract.ArtifactKind = "repository_scan"
		if contextURLs["repository_scan"] != "" {
			contract.PublicURL = contextURLs["repository_scan"]
		}
	}
	return contract
}

func (s *Store) ProjectRouting(projectID string) (ProjectRoutingResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	project, ok := s.projects[strings.TrimSpace(projectID)]
	if !ok {
		return ProjectRoutingResponse{}, fmt.Errorf("project not found")
	}
	response := ProjectRoutingResponse{
		ProtocolVersion: "mergeos.routing.v1",
		Kind:            "project_routing",
		ProjectID:       project.ID,
		ProjectTitle:    publicLiveFeedProjectTitle(project),
		Status:          "routing",
		UpdatedAt:       project.CreatedAt,
		Lanes:           []ProjectRoutingLane{},
		Routes:          []ProjectRoutingRoute{},
	}
	agentDepth := projectRoutingAgentDepthLocked(s.tasks)
	contributor := projectRoutingTopContributorLocked(s.tasks)
	lanes := map[string]*ProjectRoutingLane{}
	for _, task := range project.Tasks {
		if task == nil {
			continue
		}
		if task.CreatedAt.After(response.UpdatedAt) {
			response.UpdatedAt = task.CreatedAt
		}
		ready, blockedBy := projectRoutingReadiness(task)
		route := projectRoutingRoute(task, ready, blockedBy, agentDepth, contributor)
		response.Routes = append(response.Routes, route)
		response.Stats.TaskCount++
		if ready {
			response.Stats.ReadyCount++
		} else if len(blockedBy) > 0 {
			response.Stats.BlockedCount++
		}
		switch task.RequiredWorkerKind {
		case WorkerAgent:
			response.Stats.AgentLaneCount++
			if ready {
				response.Stats.AgentCandidateCount++
			}
		case WorkerHybrid:
			response.Stats.HybridLaneCount++
			if ready {
				response.Stats.AgentCandidateCount++
				response.Stats.ContributorCandidateCount++
			}
		default:
			response.Stats.HumanLaneCount++
			if ready {
				response.Stats.ContributorCandidateCount++
			}
		}
		lane := projectRoutingLaneForTask(lanes, task)
		lane.TaskCount++
		lane.RewardCents += task.RewardCents
		if ready {
			lane.ReadyCount++
		} else if len(blockedBy) > 0 {
			lane.BlockedCount++
		}
		if taskIsReleased(task) && task.AcceptedAt != nil && task.AcceptedAt.After(response.UpdatedAt) {
			response.UpdatedAt = *task.AcceptedAt
		}
	}
	for _, lane := range lanes {
		if lane.ReadyCount > 0 {
			lane.Status = "ready"
		} else if lane.BlockedCount > 0 {
			lane.Status = "blocked"
		} else {
			lane.Status = "complete"
		}
		response.Lanes = append(response.Lanes, *lane)
	}
	sort.Slice(response.Lanes, func(i, j int) bool {
		if response.Lanes[i].ReadyCount == response.Lanes[j].ReadyCount {
			return response.Lanes[i].ID < response.Lanes[j].ID
		}
		return response.Lanes[i].ReadyCount > response.Lanes[j].ReadyCount
	})
	sort.Slice(response.Routes, func(i, j int) bool {
		if response.Routes[i].Ready == response.Routes[j].Ready {
			return response.Routes[i].IssueNumber < response.Routes[j].IssueNumber
		}
		return response.Routes[i].Ready
	})
	if response.Stats.TaskCount == 0 {
		response.Status = "waiting"
		response.Summary = "Task routing will appear once the project has funded work."
	} else if response.Stats.ReadyCount == 0 && response.Stats.BlockedCount == 0 {
		response.Status = "complete"
		response.Summary = "All funded tasks have been accepted or paid."
	} else {
		response.Status = "ready"
		response.Summary = fmt.Sprintf("%d tasks are ready across %d routing lanes.", response.Stats.ReadyCount, len(response.Lanes))
	}
	return response, nil
}

func projectRoutingReadiness(task *Task) (bool, []string) {
	if !taskIsOpenForClaim(task) {
		return false, nil
	}
	blockedBy := []string{}
	if task.RewardCents <= 0 {
		blockedBy = append(blockedBy, "missing_reward")
	}
	if task.IssueNumber <= 0 {
		blockedBy = append(blockedBy, "missing_issue_number")
	}
	return len(blockedBy) == 0, blockedBy
}

func projectRoutingRoute(task *Task, ready bool, blockedBy []string, agentDepth map[string]int, contributor *ProjectRoutingContributor) ProjectRoutingRoute {
	lane := string(task.RequiredWorkerKind)
	if task.SuggestedAgentType != "" {
		lane = task.SuggestedAgentType
	}
	claimID := marketplaceBountyID(task.ProjectID, task.IssueNumber)
	protocolURL := "/api/public/protocol/tasks?task_id=" + claimID
	action := "publish_bounty"
	reasons := []string{"Escrow-backed task is visible in the marketplace."}
	score := 70
	var agent *ProjectRoutingAgent
	var worker *ProjectRoutingContributor
	if taskIsReleased(task) {
		action = "paid"
		score = 100
		reasons = []string{"Task has already been released and paid."}
	} else if taskHasWorker(task) {
		action = "review_evidence"
		score = 92
		reasons = []string{"Task is claimed and waiting for review evidence or payout release."}
	} else if !ready {
		action = "wait_for_dependencies"
		score = 30
		reasons = append([]string{"Resolve blockers before routing."}, blockedBy...)
	} else {
		switch task.RequiredWorkerKind {
		case WorkerAgent:
			action = "route_to_agent"
			score = 88
			agent = &ProjectRoutingAgent{Type: task.SuggestedAgentType, Title: marketplaceTitle(task.SuggestedAgentType), Status: "active", QueueDepth: agentDepth[task.SuggestedAgentType]}
			reasons = append(reasons, "Agent lane has a scoped work packet.")
		case WorkerHybrid:
			action = "route_hybrid_pair"
			score = 82
			agent = &ProjectRoutingAgent{Type: task.SuggestedAgentType, Title: marketplaceTitle(task.SuggestedAgentType), Status: "active", QueueDepth: agentDepth[task.SuggestedAgentType]}
			if contributor != nil {
				workerCopy := *contributor
				worker = &workerCopy
				score += 5
			}
			reasons = append(reasons, "Hybrid task benefits from human approval plus AI execution.")
		default:
			action = "invite_contributor"
			if contributor != nil {
				workerCopy := *contributor
				worker = &workerCopy
				score = 78 + minInt(contributor.ReputationScore/20, 10)
				reasons = append(reasons, "Contributor reputation history is available.")
			}
		}
	}
	return ProjectRoutingRoute{
		ID:                    "route:" + task.ID,
		TaskID:                task.ID,
		ClaimID:               claimID,
		IssueNumber:           task.IssueNumber,
		Title:                 task.Title,
		Lane:                  lane,
		Status:                string(task.Status),
		Ready:                 ready,
		BlockedBy:             blockedBy,
		RewardCents:           task.RewardCents,
		RequiredWorkerKind:    task.RequiredWorkerKind,
		SuggestedAgentType:    task.SuggestedAgentType,
		ProtocolURL:           protocolURL,
		RecommendedNextAction: action,
		MatchScore:            score,
		RoutingReason:         stableStrings(reasons),
		RecommendedAgent:      agent,
		RecommendedWorker:     worker,
		RoutingPacket:         projectRoutingPacket(task, action, claimID, protocolURL),
	}
}

func projectRoutingPacket(task *Task, action, claimID, protocolURL string) ProjectRoutingPacket {
	projectID := strings.TrimSpace(task.ProjectID)
	agentType := strings.TrimSpace(task.SuggestedAgentType)
	if agentType == "" && task.WorkerKind != WorkerHuman {
		agentType = strings.TrimSpace(task.AgentType)
	}
	if agentType == "" && task.RequiredWorkerKind != WorkerHuman {
		agentType = "general-ai-agent"
	}
	submitEndpoint := "/api/tasks/" + claimID + "/submit"
	actionEndpoint := "/api/projects/" + projectID + "/agent-actions"
	contextURLs := map[string]string{
		"task_protocol":     protocolURL,
		"marketplace":       "/api/public/marketplace",
		"agent_queue":       agentQueueEndpoint,
		"contributors":      "/api/public/protocol/contributors",
		"workflow_protocol": "/api/public/projects/" + projectID + "/workflow",
		"workflow_pulse":    "/api/public/projects/" + projectID + "/ai-workflow",
		"pr_monitor":        "/api/public/projects/" + projectID + "/pull-requests",
	}
	if task.IssueURL != "" {
		contextURLs["issue"] = marketplacePublicRepoURL(task.IssueURL)
	}
	packet := ProjectRoutingPacket{
		Action:      action,
		Method:      "GET",
		Endpoint:    protocolURL,
		ContextURLs: contextURLs,
		Runbook: []AgentRunbookStep{
			{Step: 1, Action: "fetch_context", Label: "Read the public task protocol and acceptance criteria", Method: "GET", Endpoint: protocolURL},
			{Step: 2, Action: "inspect_routing", Label: "Compare route lane, worker kind, score, and readiness blockers", Method: "GET", Endpoint: "/api/projects/" + projectID + "/routing"},
		},
	}
	switch action {
	case "route_to_agent", "route_hybrid_pair":
		lease := agentLeasePacket(claimID, agentType)
		packet.Method = "POST"
		packet.Endpoint = lease.LeaseEndpoint
		packet.Payload = lease.Payload
		packet.Runbook = append(packet.Runbook,
			AgentRunbookStep{Step: 3, Action: "lease_work", Label: "Lease the agent work packet before running checks", Method: "POST", Endpoint: lease.LeaseEndpoint},
			AgentRunbookStep{Step: 4, Action: "record_evidence", Label: "Record scoped agent evidence", Method: "POST", Endpoint: actionEndpoint},
			AgentRunbookStep{Step: 5, Action: "submit_review", Label: "Submit pull request and review evidence", Method: "POST", Endpoint: submitEndpoint},
		)
		packet.OutputContracts = []AgentOutputContract{
			{Action: "lease", ArtifactKind: "agent_lease", OutputEndpoint: lease.LeaseEndpoint, OutputProtocol: "mergeos.agent-lease.v1", OutputProtocolURL: "/protocol/agent-lease.v1.schema.json", PublicURL: "/api/public/live-feed"},
		}
		for _, agentAction := range agentActionsForTask(task) {
			packet.OutputContracts = append(packet.OutputContracts, agentQueueOutputContract(agentAction, projectID, actionEndpoint, contextURLs))
		}
		packet.OutputContracts = append(packet.OutputContracts, AgentOutputContract{
			Action:            "submit",
			ArtifactKind:      "task_submission",
			OutputEndpoint:    submitEndpoint,
			OutputProtocol:    "mergeos.task-submission.v1",
			OutputProtocolURL: "/protocol/task-submission.v1.schema.json",
			PublicURL:         protocolURL,
		})
	case "invite_contributor", "publish_bounty":
		packet.Method = "POST"
		packet.Endpoint = "/api/proposals"
		packet.Payload = map[string]any{
			"task_id":         claimID,
			"cover_letter":    proposalPacketCoverLetter(nil, task),
			"bid_cents":       task.RewardCents,
			"estimated_hours": marketplaceEstimatedHours(task),
			"availability":    "Available after customer approval",
		}
		packet.Runbook = append(packet.Runbook,
			AgentRunbookStep{Step: 3, Action: "prepare_proposal", Label: "Attach bid, availability, and identity proof", Method: "POST", Endpoint: "/api/proposals"},
			AgentRunbookStep{Step: 4, Action: "wait_customer_review", Label: "Customer reviews the proposal before claim", Method: "GET", Endpoint: "/api/workers/me"},
		)
		packet.OutputContracts = []AgentOutputContract{
			{Action: "propose", ArtifactKind: "worker_proposal", OutputEndpoint: "/api/proposals", OutputProtocol: "mergeos.proposal.v1", OutputProtocolURL: "/protocol/proposal.v1.schema.json", PublicURL: "/api/public/live-feed"},
		}
	case "review_evidence":
		packet.Method = "POST"
		packet.Endpoint = submitEndpoint
		packet.Payload = map[string]any{
			"pull_request_url":    "",
			"review_evidence_url": "",
			"review_notes":        "Attach PR, tests, deployment preview, or agent evidence for review.",
		}
		packet.Runbook = append(packet.Runbook,
			AgentRunbookStep{Step: 3, Action: "submit_review", Label: "Submit PR and review evidence for the claimed task", Method: "POST", Endpoint: submitEndpoint},
		)
		packet.OutputContracts = []AgentOutputContract{
			{Action: "submit", ArtifactKind: "task_submission", OutputEndpoint: submitEndpoint, OutputProtocol: "mergeos.task-submission.v1", OutputProtocolURL: "/protocol/task-submission.v1.schema.json", PublicURL: protocolURL},
		}
	case "paid":
		packet.Endpoint = "/api/public/ledger/proof"
		packet.Runbook = append(packet.Runbook,
			AgentRunbookStep{Step: 3, Action: "verify_payout", Label: "Verify payout proof through the public ledger", Method: "GET", Endpoint: "/api/public/ledger/proof"},
		)
		packet.OutputContracts = []AgentOutputContract{
			{Action: "verify_payout", ArtifactKind: "ledger_proof", OutputEndpoint: "/api/public/ledger/proof", OutputProtocol: "mergeos.ledger-proof.v1", OutputProtocolURL: "/protocol/ledger-proof.v1.schema.json", PublicURL: "/api/public/ledger/proof"},
		}
	default:
		packet.Runbook = append(packet.Runbook,
			AgentRunbookStep{Step: 3, Action: "resolve_blockers", Label: "Resolve readiness blockers before claim or proposal", Method: "GET", Endpoint: protocolURL},
		)
	}
	return packet
}

func projectRoutingLaneForTask(lanes map[string]*ProjectRoutingLane, task *Task) *ProjectRoutingLane {
	key := string(task.RequiredWorkerKind)
	if task.SuggestedAgentType != "" {
		key += ":" + task.SuggestedAgentType
	}
	lane := lanes[key]
	if lane != nil {
		return lane
	}
	title := marketplaceTitle(string(task.RequiredWorkerKind))
	if task.SuggestedAgentType != "" {
		title = marketplaceTitle(task.SuggestedAgentType)
	}
	lane = &ProjectRoutingLane{
		ID:             key,
		Title:          title,
		WorkerKind:     task.RequiredWorkerKind,
		AgentType:      task.SuggestedAgentType,
		RecommendedFor: projectRoutingRecommendedFor(task),
	}
	lanes[key] = lane
	return lane
}

func projectRoutingRecommendedFor(task *Task) string {
	switch task.RequiredWorkerKind {
	case WorkerAgent:
		return "automated execution"
	case WorkerHybrid:
		return "human plus ai"
	default:
		return "contributor delivery"
	}
}

func projectRoutingAgentDepthLocked(tasks map[string]*Task) map[string]int {
	depth := map[string]int{}
	for _, task := range tasks {
		if !taskIsOpenForClaim(task) || strings.TrimSpace(task.SuggestedAgentType) == "" {
			continue
		}
		depth[task.SuggestedAgentType]++
	}
	return depth
}

func projectRoutingTopContributorLocked(tasks map[string]*Task) *ProjectRoutingContributor {
	byWorker := map[string]*ProjectRoutingContributor{}
	for _, task := range tasks {
		if !taskIsReleased(task) || strings.TrimSpace(task.WorkerID) == "" {
			continue
		}
		workerID := task.WorkerID
		row := byWorker[workerID]
		if row == nil {
			row = &ProjectRoutingContributor{
				WorkerID:  workerID,
				Name:      marketplaceWorkerName(workerID, task.AgentType),
				Kind:      task.WorkerKind,
				RiskLevel: "low",
			}
			byWorker[workerID] = row
		}
		row.ReputationScore += 8
	}
	var best *ProjectRoutingContributor
	for _, row := range byWorker {
		if row.ReputationScore > 100 {
			row.ReputationScore = 100
		}
		if best == nil || row.ReputationScore > best.ReputationScore || (row.ReputationScore == best.ReputationScore && row.WorkerID < best.WorkerID) {
			copyRow := *row
			best = &copyRow
		}
	}
	return best
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
