package core

import (
	"errors"
	"strings"
	"time"
)

const defaultAgentActionActor = "ai-agent"

func (s *Store) RecordProjectAgentAction(projectID string, req AgentActionRequest) (AgentActionResponse, error) {
	action, err := normalizeAgentAction(req.Action)
	if err != nil {
		return AgentActionResponse{}, err
	}
	status, err := normalizeAgentActionStatus(req.Status)
	if err != nil {
		return AgentActionResponse{}, err
	}
	agentType := sanitizeLedgerReferenceValue(req.AgentType)
	if agentType == "" {
		agentType = defaultAgentActionActor
	}
	delegatedBy := normalizeAgentDelegate(req.DelegatedBy, ceoAgentType)
	designAgent := normalizeAgentDelegate(req.DesignAgent, designReviewAgentType)
	subagentType := normalizeAgentDelegate(req.SubagentType, agentType)
	delegationChain := normalizeAgentDelegationChain(req.DelegationChain, delegatedBy, designAgent, subagentType)
	durationMillis := req.DurationMillis
	if durationMillis < 0 {
		durationMillis = 0
	}
	pullNumber := req.PullNumber
	if pullNumber < 0 {
		pullNumber = 0
	}
	contextURLs := normalizeAgentActionURLs(req.ContextURLs)
	evidence := normalizeAgentActionTextList(req.Evidence, 12, 220)
	runbook := normalizeAgentActionTextList(req.Runbook, 12, 220)
	checks := normalizeAgentActionChecks(req.Checks)

	s.mu.Lock()
	defer s.mu.Unlock()

	project, ok := s.projects[strings.TrimSpace(projectID)]
	if !ok {
		return AgentActionResponse{}, errors.New("project not found")
	}
	claimID, err := s.publicAgentActionClaimIDLocked(project, req)
	if err != nil {
		return AgentActionResponse{}, err
	}
	if s.geminiWebhookLogs == nil {
		s.geminiWebhookLogs = map[string]*GeminiWebhookLog{}
	}

	now := time.Now().UTC()
	log := GeminiWebhookLog{
		ID:              geminiWebhookLogID(),
		EventName:       "agent_action",
		Action:          action,
		Repository:      projectAgentActionRepository(project),
		PullNumber:      pullNumber,
		Sender:          "agent:" + agentType,
		Status:          status,
		StatusCode:      agentActionStatusCode(status),
		CommentURL:      publicLiveFeedURL(req.ReferenceURL),
		Labels:          normalizeAgentActionLabels(req.Labels),
		ContextURLs:     contextURLs,
		Evidence:        evidence,
		Runbook:         runbook,
		Checks:          checks,
		DelegatedBy:     delegatedBy,
		DesignAgent:     designAgent,
		SubagentType:    subagentType,
		DelegationChain: delegationChain,
		DurationMillis:  durationMillis,
		ReceivedAt:      now,
	}
	if durationMillis > 0 || status == "processed" || status == "failed" {
		completedAt := now
		log.CompletedAt = &completedAt
	}
	s.geminiWebhookLogs[log.ID] = &log
	s.trimGeminiWebhookLogsLocked()
	if err := s.saveLocked(); err != nil {
		return AgentActionResponse{}, err
	}
	return AgentActionResponse{
		ProtocolVersion: "mergeos.agent-action.v1",
		Kind:            "agent_action",
		ActionID:        log.ID,
		ProjectID:       project.ID,
		ClaimID:         claimID,
		BountyID:        claimID,
		Action:          log.Action,
		AgentType:       agentType,
		Status:          log.Status,
		Repository:      log.Repository,
		PullNumber:      log.PullNumber,
		ReferenceURL:    log.CommentURL,
		Labels:          log.Labels,
		ContextURLs:     log.ContextURLs,
		Evidence:        log.Evidence,
		Runbook:         log.Runbook,
		Checks:          log.Checks,
		DelegatedBy:     log.DelegatedBy,
		DesignAgent:     log.DesignAgent,
		SubagentType:    log.SubagentType,
		DelegationChain: log.DelegationChain,
		DurationMillis:  log.DurationMillis,
		ReceivedAt:      log.ReceivedAt,
		CompletedAt:     log.CompletedAt,
		Log:             log,
	}, nil
}

func (s *Store) AuthorizeAssignedWorkerAgentAction(userID, projectID string, req AgentActionRequest) (AgentActionRequest, error) {
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return AgentActionRequest{}, errors.New("project not found")
	}
	claimRef := agentActionClaimRef(req)
	if claimRef == "" {
		return AgentActionRequest{}, errors.New("claim_id or bounty_id is required for assigned worker evidence")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	user := s.users[strings.TrimSpace(userID)]
	if user == nil {
		return AgentActionRequest{}, errors.New("login is required")
	}
	taskID, err := s.resolveTaskClaimIDLocked(claimRef)
	if err != nil {
		return AgentActionRequest{}, err
	}
	task, ok := s.tasks[taskID]
	if !ok || task == nil {
		return AgentActionRequest{}, errors.New("task not found")
	}
	if task.ProjectID != projectID {
		return AgentActionRequest{}, errors.New("claim_id does not belong to this project")
	}
	if _, ok := s.projects[projectID]; !ok {
		return AgentActionRequest{}, errors.New("project not found")
	}
	if !taskHasWorker(task) {
		return AgentActionRequest{}, errors.New("task must be claimed before evidence can be recorded")
	}
	workerIDs, _ := workerIdentitySets(user)
	if !workerIDs[workerIdentityKey(task.WorkerID)] {
		return AgentActionRequest{}, errors.New("claimed task worker identity is required")
	}

	agentType := strings.TrimSpace(task.AgentType)
	if agentType == "" {
		agentType = strings.TrimSpace(task.SuggestedAgentType)
	}
	if task.WorkerKind != WorkerHuman && agentType != "" {
		requestedType := strings.TrimSpace(req.AgentType)
		if requestedType == "" {
			req.AgentType = agentType
		} else if !strings.EqualFold(requestedType, agentType) {
			return AgentActionRequest{}, errors.New("agent_type must match the claimed task")
		} else {
			req.AgentType = agentType
		}
	}

	publicClaimID := marketplaceBountyID(task.ProjectID, task.IssueNumber)
	req.ClaimID = publicClaimID
	req.BountyID = publicClaimID
	return req, nil
}

func (s *Store) publicAgentActionClaimIDLocked(project *Project, req AgentActionRequest) (string, error) {
	claimRef := agentActionClaimRef(req)
	if claimRef == "" {
		return "", nil
	}
	taskID, err := s.resolveTaskClaimIDLocked(claimRef)
	if err != nil {
		return "", err
	}
	task, ok := s.tasks[taskID]
	if !ok || task == nil {
		return "", errors.New("task not found")
	}
	if project == nil || task.ProjectID != project.ID {
		return "", errors.New("claim_id does not belong to this project")
	}
	return marketplaceBountyID(task.ProjectID, task.IssueNumber), nil
}

func agentActionClaimRef(req AgentActionRequest) string {
	for _, value := range []string{req.ClaimID, req.BountyID} {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func normalizeAgentAction(value string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "review":
		return "review", nil
	case "test":
		return "test", nil
	case "generate", "gen":
		return "generate", nil
	case "deploy":
		return "deploy", nil
	case "scan":
		return "scan", nil
	default:
		return "", errors.New("action must be review, test, generate, deploy, or scan")
	}
}

func normalizeAgentActionStatus(value string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "":
		return "processed", nil
	case "received", "queued":
		return "received", nil
	case "running", "in_progress":
		return "running", nil
	case "processed", "complete", "completed", "success":
		return "processed", nil
	case "needs_review", "needs-review":
		return "needs_review", nil
	case "failed", "error":
		return "failed", nil
	default:
		return "", errors.New("status must be received, running, processed, needs_review, or failed")
	}
}

func agentActionStatusCode(status string) int {
	switch status {
	case "failed":
		return 500
	case "needs_review":
		return 202
	default:
		return 200
	}
}

func normalizeAgentActionLabels(values []string) []string {
	values = cleanStrings(values)
	labels := make([]string, 0, len(values))
	for _, value := range values {
		value = sanitizeLedgerReferenceValue(value)
		if value != "" {
			labels = append(labels, value)
		}
	}
	if len(labels) > 12 {
		return labels[:12]
	}
	return labels
}

func normalizeAgentDelegate(value string, fallback string) string {
	normalized := sanitizeLedgerReferenceValue(value)
	if normalized != "" {
		return normalized
	}
	return sanitizeLedgerReferenceValue(fallback)
}

func normalizeAgentDelegationChain(values []string, delegatedBy string, designAgent string, subagentType string) []string {
	chain := make([]string, 0, len(values)+3)
	seen := map[string]bool{}
	add := func(value string) {
		value = sanitizeLedgerReferenceValue(value)
		if value == "" {
			return
		}
		key := strings.ToLower(value)
		if seen[key] {
			return
		}
		seen[key] = true
		chain = append(chain, value)
	}
	if len(values) == 0 {
		add(delegatedBy)
		add(designAgent)
		add(subagentType)
	} else {
		for _, value := range values {
			add(value)
		}
		add(delegatedBy)
		add(designAgent)
		add(subagentType)
	}
	if len(chain) > 8 {
		return chain[:8]
	}
	return chain
}

func normalizeAgentActionURLs(values []string) []string {
	values = cleanStrings(values)
	result := make([]string, 0, len(values))
	seen := map[string]bool{}
	for _, value := range values {
		value = publicLiveFeedURL(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		result = append(result, protocolText(value, 512, ""))
		if len(result) >= 8 {
			break
		}
	}
	return result
}

func normalizeAgentActionTextList(values []string, maxItems int, maxLength int) []string {
	values = cleanStrings(values)
	result := make([]string, 0, len(values))
	seen := map[string]bool{}
	for _, value := range values {
		value = protocolText(value, maxLength, "")
		if value == "" || seen[strings.ToLower(value)] {
			continue
		}
		seen[strings.ToLower(value)] = true
		result = append(result, value)
		if maxItems > 0 && len(result) >= maxItems {
			break
		}
	}
	return result
}

func normalizeAgentActionChecks(values []AgentActionCheck) []AgentActionCheck {
	result := make([]AgentActionCheck, 0, len(values))
	seen := map[string]bool{}
	for _, value := range values {
		name := protocolText(value.Name, 120, "")
		if name == "" {
			continue
		}
		key := strings.ToLower(name)
		if seen[key] {
			continue
		}
		status := normalizeAgentActionCheckStatus(value.Status)
		check := AgentActionCheck{
			Name:         name,
			Status:       status,
			Summary:      protocolText(value.Summary, 260, ""),
			ReferenceURL: protocolText(publicLiveFeedURL(value.ReferenceURL), 512, ""),
		}
		seen[key] = true
		result = append(result, check)
		if len(result) >= 12 {
			break
		}
	}
	return result
}

func normalizeAgentActionCheckStatus(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "pass", "passed", "success", "processed", "ok":
		return "passed"
	case "fail", "failed", "error":
		return "failed"
	case "warning", "warn", "needs_review", "needs-review":
		return "warning"
	case "running", "in_progress":
		return "running"
	case "skipped", "skip":
		return "skipped"
	default:
		return "passed"
	}
}

func projectAgentActionRepository(project *Project) string {
	if project == nil {
		return ""
	}
	repository := sanitizeLedgerReferenceValue(project.BountyRepoName)
	if repository != "" {
		return repository
	}
	return sanitizeLedgerReferenceValue(project.RepoURL)
}
