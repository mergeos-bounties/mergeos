package core

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"
)

func (s *Store) ProjectAgentRun(projectID string, req AgentRunRequest) (AgentRunResponse, error) {
	action, err := normalizeAgentAction(req.Action)
	if err != nil {
		if strings.TrimSpace(req.Action) == "" {
			action = "generate"
		} else {
			return AgentRunResponse{}, err
		}
	}
	claimRef := strings.TrimSpace(req.ClaimID)
	if claimRef == "" {
		claimRef = strings.TrimSpace(req.BountyID)
	}
	if claimRef == "" {
		return AgentRunResponse{}, errors.New("claim_id or bounty_id is required")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	project, ok := s.projects[strings.TrimSpace(projectID)]
	if !ok || project == nil {
		return AgentRunResponse{}, errors.New("project not found")
	}
	taskID, err := s.resolveTaskClaimIDLocked(claimRef)
	if err != nil {
		return AgentRunResponse{}, err
	}
	task, ok := s.tasks[taskID]
	if !ok || task == nil {
		return AgentRunResponse{}, errors.New("task not found")
	}
	if task.ProjectID != project.ID {
		return AgentRunResponse{}, errors.New("claim_id does not belong to this project")
	}

	claimID := marketplaceBountyID(project.ID, task.IssueNumber)
	agentType := sanitizeLedgerReferenceValue(req.AgentType)
	if agentType == "" {
		agentType = strings.TrimSpace(task.AgentType)
	}
	if agentType == "" {
		agentType = strings.TrimSpace(task.SuggestedAgentType)
	}
	if agentType == "" {
		agentType = defaultAgentActionActor
	}
	baseBranch := cleanAgentRunBranch(req.BaseBranch, "main")
	branchName := cleanAgentRunBranch(fmt.Sprintf("mergeos-%s-%s", action, strings.ToLower(claimID)), "mergeos-agent-run")
	repository := marketplacePublicRepoURL(projectSourceRepoURL(project))
	if repository == "" {
		repository = marketplacePublicRepoURL(project.RepoURL)
	}
	if repository == "" {
		repository = sanitizeLedgerReferenceValue(project.BountyRepoName)
	}
	actionEndpoint := "/api/projects/" + project.ID + "/agent-actions"
	submitEndpoint := "/api/tasks/" + claimID + "/submit"
	contextURLs := agentRunContextURLs(project.ID, claimID, task, repository, req.ContextURLs)
	compareURL := agentRunCompareURL(repository, baseBranch, branchName)
	objective := sanitizeLedgerReferenceValue(req.Objective)
	if objective == "" {
		objective = task.Title
	}

	return AgentRunResponse{
		ProtocolVersion:  "mergeos.agent-run.v1",
		Kind:             "agent_run",
		RunID:            "run_" + claimID + "_" + action,
		ProjectID:        project.ID,
		ClaimID:          claimID,
		BountyID:         claimID,
		Action:           action,
		AgentType:        agentType,
		Repository:       repository,
		BaseBranch:       baseBranch,
		BranchName:       branchName,
		PRTitle:          fmt.Sprintf("%s: %s", strings.ToUpper(action[:1])+action[1:], sanitizeLedgerReferenceValue(task.Title)),
		PRBody:           agentRunPRBody(project, task, claimID, objective, contextURLs),
		GitHubCompareURL: compareURL,
		ActionEndpoint:   actionEndpoint,
		SubmitEndpoint:   submitEndpoint,
		ContextURLs:      contextURLs,
		Runbook:          agentRunbook(action, actionEndpoint, submitEndpoint, compareURL),
		ActionPayload: AgentActionRequest{
			Action:       action,
			ClaimID:      claimID,
			BountyID:     claimID,
			AgentType:    agentType,
			Status:       "running",
			ReferenceURL: compareURL,
			ContextURLs:  agentRunContextURLList(contextURLs),
			Runbook:      []string{"Create branch " + branchName, "Open pull request evidence", "Record agent action proof"},
		},
		OutputContracts: agentRunOutputContracts(project.ID, claimID, actionEndpoint, submitEndpoint, contextURLs),
		CreatedAt:       time.Now().UTC(),
	}, nil
}

func cleanAgentRunBranch(value, fallback string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		value = fallback
	}
	var b strings.Builder
	lastDash := false
	for _, r := range value {
		allowed := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
		if allowed {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			b.WriteByte('-')
			lastDash = true
		}
	}
	cleaned := strings.Trim(b.String(), "-")
	if cleaned == "" {
		return fallback
	}
	if len(cleaned) > 72 {
		cleaned = strings.Trim(cleaned[:72], "-")
	}
	return cleaned
}

func agentRunContextURLs(projectID, claimID string, task *Task, repository string, extra []string) map[string]string {
	context := map[string]string{
		"task_protocol":     "/api/public/protocol/tasks?task_id=" + url.QueryEscape(claimID),
		"workflow_protocol": "/api/public/projects/" + projectID + "/workflow",
		"ai_workflow":       "/api/public/projects/" + projectID + "/ai-workflow",
		"agent_action":      "/api/projects/" + projectID + "/agent-actions",
	}
	if repository != "" {
		context["repository"] = repository
	}
	if task != nil {
		if issueURL := marketplacePublicRepoURL(task.IssueURL); issueURL != "" {
			context["issue"] = issueURL
		}
	}
	for index, value := range normalizeAgentActionURLs(extra) {
		context[fmt.Sprintf("external_%d", index+1)] = value
	}
	return context
}

func agentRunContextURLList(context map[string]string) []string {
	keys := []string{"task_protocol", "workflow_protocol", "ai_workflow", "repository", "issue"}
	result := []string{}
	for _, key := range keys {
		if value := strings.TrimSpace(context[key]); value != "" {
			result = append(result, value)
		}
	}
	return result
}

func agentRunCompareURL(repository, baseBranch, branchName string) string {
	if !strings.HasPrefix(repository, "https://github.com/") {
		return ""
	}
	return strings.TrimRight(repository, "/") + "/compare/" + url.PathEscape(baseBranch) + "..." + url.PathEscape(branchName)
}

func agentRunPRBody(project *Project, task *Task, claimID, objective string, context map[string]string) string {
	lines := []string{
		"MergeOS agent run plan",
		"",
		"Project: " + sanitizeLedgerReferenceValue(project.Title),
		"Claim: " + claimID,
		"Objective: " + objective,
		"Acceptance: " + sanitizeLedgerReferenceValue(task.Acceptance),
		"",
		"Context:",
		"- Task protocol: " + context["task_protocol"],
		"- Workflow: " + context["workflow_protocol"],
		"- AI workflow: " + context["ai_workflow"],
		"",
		"Required proof:",
		"- Pull request URL",
		"- Test or review evidence",
		"- Deployment or rollback note when applicable",
		"- Agent action record through " + context["agent_action"],
	}
	return strings.Join(lines, "\n")
}

func agentRunbook(action, actionEndpoint, submitEndpoint, compareURL string) []AgentRunbookStep {
	return []AgentRunbookStep{
		{Step: 1, Action: "create_branch", Label: "Create a scoped branch for the " + action + " agent run", Method: "GIT", Endpoint: compareURL},
		{Step: 2, Action: "open_pull_request", Label: "Open PR with task protocol, workflow, and acceptance context", Method: "POST", Endpoint: compareURL},
		{Step: 3, Action: "record_agent_action", Label: "Record agent evidence, checks, and context URLs", Method: "POST", Endpoint: actionEndpoint},
		{Step: 4, Action: "submit_task_evidence", Label: "Submit PR, test, and deployment evidence for customer review", Method: "POST", Endpoint: submitEndpoint},
	}
}

func agentRunOutputContracts(projectID, claimID, actionEndpoint, submitEndpoint string, context map[string]string) []AgentOutputContract {
	return []AgentOutputContract{
		{Action: "record_agent_action", ArtifactKind: "agent_action", OutputEndpoint: actionEndpoint, OutputProtocol: "mergeos.agent-action.v1", OutputProtocolURL: "/protocol/agent-action.v1.schema.json", PublicURL: context["ai_workflow"]},
		{Action: "open_pull_request", ArtifactKind: "pull_request", OutputEndpoint: "/api/public/projects/" + projectID + "/pull-requests", OutputProtocol: "mergeos.pr-monitor.v1", OutputProtocolURL: "/protocol/pr-monitor.v1.schema.json"},
		{Action: "submit_task_evidence", ArtifactKind: "task_submission", OutputEndpoint: submitEndpoint, OutputProtocol: "mergeos.task-submission.v1", OutputProtocolURL: "/protocol/task-submission.v1.schema.json", PublicURL: context["task_protocol"]},
		{Action: "verify_ledger", ArtifactKind: "ledger_proof", OutputEndpoint: "/api/public/ledger/proof", OutputProtocol: "mergeos.ledger-proof.v1", OutputProtocolURL: "/protocol/ledger-proof.v1.schema.json"},
	}
}
