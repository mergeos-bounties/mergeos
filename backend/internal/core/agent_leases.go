package core

import (
	"errors"
	"strings"
	"time"
)

func (s *Store) CreateOrRefreshAgentLease(user *User, req AgentLeaseRequest) (AgentLeaseResponse, error) {
	if user == nil || strings.TrimSpace(user.ID) == "" {
		return AgentLeaseResponse{}, errors.New("login is required")
	}
	claimID := agentLeaseClaimRef(req)
	if claimID == "" {
		return AgentLeaseResponse{}, errors.New("claim_id or bounty_id is required")
	}
	status, err := normalizeAgentLeaseStatus(req.Status)
	if err != nil {
		return AgentLeaseResponse{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	task, project, err := s.agentLeaseTaskLocked(claimID)
	if err != nil {
		return AgentLeaseResponse{}, err
	}
	if task.Status == TaskAccepted {
		return AgentLeaseResponse{}, errors.New("task is already accepted")
	}

	publicClaimID := marketplaceBountyID(project.ID, task.IssueNumber)
	agentType := strings.TrimSpace(req.AgentType)
	if agentType == "" {
		agentType = strings.TrimSpace(task.AgentType)
	}
	if agentType == "" {
		agentType = strings.TrimSpace(task.SuggestedAgentType)
	}
	if agentType == "" {
		agentType = "general-ai-agent"
	}

	leaseID := strings.TrimSpace(req.LeaseID)
	if leaseID == "" {
		leaseID = s.newID("agl")
	}

	now := time.Now().UTC()
	actionEndpoint := "/api/projects/" + project.ID + "/agent-actions"
	submitEndpoint := "/api/tasks/" + publicClaimID + "/submit"
	contextURLs := map[string]string{
		"task_protocol":     "/api/public/protocol/tasks?task_id=" + publicClaimID,
		"workflow_protocol": "/api/public/projects/" + project.ID + "/workflow",
		"workflow_pulse":    "/api/public/projects/" + project.ID + "/ai-workflow",
		"pr_monitor":        "/api/public/projects/" + project.ID + "/pull-requests",
	}
	response := AgentLeaseResponse{
		ProtocolVersion:   "mergeos.agent-lease.v1",
		Kind:              "agent_lease",
		LeaseID:           leaseID,
		Status:            status,
		ClaimID:           publicClaimID,
		BountyID:          publicClaimID,
		ProjectID:         project.ID,
		ProjectTitle:      marketplaceProjectTitle(project),
		TaskTitle:         task.Title,
		IssueNumber:       task.IssueNumber,
		AgentType:         agentType,
		WorkerID:          agentLeaseWorkerID(user),
		LeaseEndpoint:     agentLeaseEndpoint,
		HeartbeatEndpoint: agentLeaseEndpoint,
		ActionEndpoint:    actionEndpoint,
		SubmitEndpoint:    submitEndpoint,
		HeartbeatSeconds:  agentHeartbeatSeconds,
		LeaseTTLSeconds:   agentLeaseTTLSeconds,
		LeasedAt:          now,
		HeartbeatDueAt:    now.Add(time.Duration(agentHeartbeatSeconds) * time.Second),
		ExpiresAt:         now.Add(time.Duration(agentLeaseTTLSeconds) * time.Second),
		OutputContracts: append([]AgentOutputContract{
			{
				Action:            "heartbeat",
				ArtifactKind:      "agent_lease",
				OutputEndpoint:    agentLeaseEndpoint,
				OutputProtocol:    "mergeos.agent-lease.v1",
				OutputProtocolURL: "/protocol/agent-lease.v1.schema.json",
			},
		}, agentLeaseOutputContracts(project.ID, actionEndpoint, submitEndpoint, contextURLs)...),
		NextActions: []AgentRunbookStep{
			{Step: 1, Action: "heartbeat", Label: "Refresh the lease before heartbeat_due_at", Method: "POST", Endpoint: agentLeaseEndpoint},
			{Step: 2, Action: "record_evidence", Label: "Record scoped agent evidence on the project live log", Method: "POST", Endpoint: actionEndpoint},
			{Step: 3, Action: "submit_review", Label: "Submit final pull request and review evidence for payout review", Method: "POST", Endpoint: submitEndpoint},
		},
	}
	if s.agentLeases == nil {
		s.agentLeases = map[string]*AgentLeaseResponse{}
	}
	copyResponse := response
	s.agentLeases[leaseID] = &copyResponse
	if s.geminiWebhookLogs == nil {
		s.geminiWebhookLogs = map[string]*GeminiWebhookLog{}
	}
	completedAt := now
	log := GeminiWebhookLog{
		ID:              geminiWebhookLogID(),
		EventName:       "agent_lease",
		Action:          status,
		Repository:      projectAgentActionRepository(project),
		Sender:          "agent:" + agentType,
		Status:          status,
		StatusCode:      200,
		CommentURL:      contextURLs["task_protocol"],
		KeyID:           leaseID,
		Labels:          []string{"claim:" + publicClaimID, "lease:" + status},
		ContextURLs:     []string{contextURLs["task_protocol"], contextURLs["workflow_protocol"], contextURLs["workflow_pulse"], contextURLs["pr_monitor"], agentQueueEndpoint},
		Evidence:        []string{"lease_id:" + leaseID, "claim_id:" + publicClaimID, "worker_id:" + response.WorkerID},
		Runbook:         []string{"Lease public agent work packet", "Heartbeat before due time", "Record evidence before submission"},
		Checks:          []AgentActionCheck{{Name: "agent_lease", Status: "passed", Summary: "Lease accepted for public claim " + publicClaimID, ReferenceURL: contextURLs["task_protocol"]}},
		SourceFindingID: publicClaimID,
		Signal:          "agent_lease_" + status,
		Path:            agentLeaseEndpoint,
		DelegatedBy:     ceoAgentType,
		DesignAgent:     designReviewAgentType,
		SubagentType:    agentType,
		DelegationChain: agentDelegationChain(agentType),
		ReceivedAt:      now,
		CompletedAt:     &completedAt,
	}
	s.geminiWebhookLogs[log.ID] = &log
	s.trimGeminiWebhookLogsLocked()
	if err := s.saveLocked(); err != nil {
		return AgentLeaseResponse{}, err
	}
	return response, nil
}

func (s *Store) agentLeaseTaskLocked(claimID string) (*Task, *Project, error) {
	taskID, err := s.resolveTaskClaimIDLocked(claimID)
	if err != nil {
		return nil, nil, err
	}
	task, ok := s.tasks[taskID]
	if !ok || task == nil {
		return nil, nil, errors.New("task not found")
	}
	project, ok := s.projects[task.ProjectID]
	if !ok || project == nil {
		return nil, nil, errors.New("project not found")
	}
	return task, project, nil
}

func agentLeaseClaimRef(req AgentLeaseRequest) string {
	for _, value := range []string{req.ClaimID, req.BountyID} {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func normalizeAgentLeaseStatus(value string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "lease", "leased", "reserve", "reserved":
		return "leased", nil
	case "heartbeat", "refresh", "running":
		return "heartbeat", nil
	case "released", "release", "complete", "completed":
		return "released", nil
	default:
		return "", errors.New("status must be leased, heartbeat, or released")
	}
}

func agentLeaseWorkerID(user *User) string {
	if github := normalizeGitHubUsername(user.GitHubUsername); github != "" {
		return githubWorkerAccount(github)
	}
	if wallet := normalizeWalletAddress(user.WalletAddress); wallet != "" {
		return walletAccount(wallet)
	}
	return strings.TrimSpace(user.ID)
}

func agentLeaseOutputContracts(projectID, actionEndpoint, submitEndpoint string, contextURLs map[string]string) []AgentOutputContract {
	return []AgentOutputContract{
		agentQueueOutputContract("review", projectID, actionEndpoint, contextURLs),
		agentQueueOutputContract("test", projectID, actionEndpoint, contextURLs),
		{
			Action:            "submit",
			ArtifactKind:      "task_submission",
			OutputEndpoint:    submitEndpoint,
			OutputProtocol:    "mergeos.task-submission.v1",
			OutputProtocolURL: "/protocol/task-submission.v1.schema.json",
			PublicURL:         contextURLs["task_protocol"],
		},
	}
}
