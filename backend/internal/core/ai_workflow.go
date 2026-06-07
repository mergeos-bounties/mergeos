package core

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

func (s *Store) ProjectAIWorkflow(projectID string) (ProjectAIWorkflowResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	project, ok := s.projects[strings.TrimSpace(projectID)]
	if !ok {
		return ProjectAIWorkflowResponse{}, errors.New("project not found")
	}
	return s.projectAIWorkflowLocked(project), nil
}

func (s *Store) PublicProjectAIWorkflow(projectID string) (ProjectAIWorkflowResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	project, ok := s.projects[strings.TrimSpace(projectID)]
	if !ok {
		return ProjectAIWorkflowResponse{}, errors.New("project not found")
	}
	return s.projectAIWorkflowLocked(project), nil
}

func (s *Store) projectAIWorkflowLocked(project *Project) ProjectAIWorkflowResponse {
	tasks := s.projectDeploymentTasksLocked(project)
	logs := s.aiWorkflowLogsLocked(project)
	deployment := s.projectDeploymentLocked(project)
	stages := []AIWorkflowStage{
		aiWorkflowRepoStage(project),
		aiWorkflowIssueScanStage(project, tasks),
		aiWorkflowTaskGenerationStage(project, tasks),
		aiWorkflowEstimateStage(project, tasks, s.cfg.TokenSymbol),
		aiWorkflowRoutingStage(project, tasks),
		aiWorkflowPRReviewStage(project, logs),
		aiWorkflowDeploymentStage(project, deployment),
	}
	signals := aiWorkflowSignals(project, logs, deployment)

	updatedAt := project.CreatedAt
	completed := 0
	active := 0
	for _, stage := range stages {
		if stage.UpdatedAt.After(updatedAt) {
			updatedAt = stage.UpdatedAt
		}
		switch stage.Status {
		case deploymentStageComplete:
			completed++
		case deploymentStageInProgress:
			active++
		}
	}
	for _, signal := range signals {
		if signal.CreatedAt.After(updatedAt) {
			updatedAt = signal.CreatedAt
		}
	}

	progress := 0
	if len(stages) > 0 {
		progress = completed * 100 / len(stages)
	}
	status := "queued"
	if completed == len(stages) && len(stages) > 0 {
		status = "ready"
	} else if completed > 0 || active > 0 {
		status = "orchestrating"
	}

	response := ProjectAIWorkflowResponse{
		ProtocolVersion: "mergeos.ai-workflow.v1",
		Kind:            "ai_workflow",
		ProjectID:       project.ID,
		ProjectTitle:    publicLiveFeedProjectTitle(project),
		Status:          status,
		Progress:        progress,
		CurrentStep:     aiWorkflowCurrentStep(stages),
		TaskCount:       len(tasks),
		AIActionCount:   len(logs),
		UpdatedAt:       updatedAt,
		Stages:          stages,
		Signals:         signals,
	}
	for _, task := range tasks {
		switch task.RequiredWorkerKind {
		case WorkerAgent:
			response.AgentTaskCount++
		case WorkerHybrid:
			response.HybridTaskCount++
		default:
			response.HumanTaskCount++
		}
	}
	return response
}

func aiWorkflowCurrentStep(stages []AIWorkflowStage) string {
	for _, stage := range stages {
		if stage.Status == deploymentStageInProgress {
			return stage.ID
		}
	}
	for _, stage := range stages {
		if stage.Status == deploymentStagePending {
			return stage.ID
		}
	}
	if len(stages) > 0 {
		return stages[len(stages)-1].ID
	}
	return ""
}

func aiWorkflowRepoStage(project *Project) AIWorkflowStage {
	reference := aiWorkflowRepoReference(project)
	status := deploymentStagePending
	body := "Repo context is waiting for import or bounty workspace creation."
	producedCount := 0
	outputIDs := []string{}
	if reference != "" {
		status = deploymentStageComplete
		body = "Repository context is attached to the delivery workflow."
		producedCount = 1
		outputIDs = append(outputIDs, project.ID)
	}
	stage := AIWorkflowStage{
		ID:        "repo_import",
		Title:     "Repository context",
		Body:      body,
		Status:    status,
		Tone:      deploymentStageTone(status),
		Reference: sanitizeLedgerReferenceValue(reference),
		URL:       marketplacePublicRepoURL(project.RepoURL),
		UpdatedAt: project.CreatedAt,
	}
	return aiWorkflowStageWithContract(project, stage, producedCount, outputIDs)
}

func aiWorkflowIssueScanStage(project *Project, tasks []*Task) AIWorkflowStage {
	status := deploymentStagePending
	body := "Issue scan will complete when the project has task rows."
	outputIDs := aiWorkflowTaskOutputIDs(tasks)
	if len(tasks) > 0 {
		status = deploymentStageComplete
		body = fmt.Sprintf("Issue analysis produced %d payable task rows.", len(tasks))
	}
	stage := AIWorkflowStage{
		ID:        "issue_scan",
		Title:     "Issue scan",
		Body:      body,
		Status:    status,
		Tone:      deploymentStageTone(status),
		Reference: "project:" + project.ID,
		UpdatedAt: project.CreatedAt,
	}
	return aiWorkflowStageWithContract(project, stage, len(tasks), outputIDs)
}

func aiWorkflowTaskGenerationStage(project *Project, tasks []*Task) AIWorkflowStage {
	latest := project.CreatedAt
	for _, task := range tasks {
		if task.CreatedAt.After(latest) {
			latest = task.CreatedAt
		}
	}
	outputIDs := aiWorkflowTaskOutputIDs(tasks)
	status := deploymentStagePending
	body := "Task generation is waiting for parsed issues or project scope."
	if len(tasks) > 0 {
		status = deploymentStageComplete
		body = fmt.Sprintf("Task generation created %d delivery nodes for marketplace routing.", len(tasks))
	}
	stage := AIWorkflowStage{
		ID:        "task_generation",
		Title:     "Task generation",
		Body:      body,
		Status:    status,
		Tone:      deploymentStageTone(status),
		Reference: "project:" + project.ID,
		UpdatedAt: latest,
	}
	return aiWorkflowStageWithContract(project, stage, len(tasks), outputIDs)
}

func aiWorkflowEstimateStage(project *Project, tasks []*Task, tokenSymbol string) AIWorkflowStage {
	allocated := int64(0)
	latest := project.CreatedAt
	for _, task := range tasks {
		allocated += task.RewardCents
		if task.CreatedAt.After(latest) {
			latest = task.CreatedAt
		}
	}
	outputIDs := aiWorkflowTaskOutputIDs(tasks)
	status := deploymentStagePending
	body := "Reward estimation is waiting for task budget allocation."
	if allocated > 0 {
		status = deploymentStageComplete
		body = fmt.Sprintf("Reward estimation allocated %s %s across delivery tasks.", formatTokenAmount(allocated), normalizedTokenSymbol(tokenSymbol))
	}
	stage := AIWorkflowStage{
		ID:        "reward_estimation",
		Title:     "Reward estimation",
		Body:      body,
		Status:    status,
		Tone:      deploymentStageTone(status),
		Reference: "project:" + project.ID,
		UpdatedAt: latest,
	}
	return aiWorkflowStageWithContract(project, stage, len(tasks), outputIDs)
}

func aiWorkflowRoutingStage(project *Project, tasks []*Task) AIWorkflowStage {
	routed := 0
	latest := project.CreatedAt
	for _, task := range tasks {
		if task.RequiredWorkerKind == WorkerHuman || task.RequiredWorkerKind == WorkerAgent || task.RequiredWorkerKind == WorkerHybrid {
			routed++
		}
		if task.CreatedAt.After(latest) {
			latest = task.CreatedAt
		}
	}
	outputIDs := aiWorkflowTaskOutputIDs(tasks)
	status := deploymentStagePending
	body := "Contributor routing is waiting for worker lane assignment."
	if len(tasks) > 0 && routed == len(tasks) {
		status = deploymentStageComplete
		body = fmt.Sprintf("%d tasks are routed to human, agent, or hybrid lanes.", routed)
	} else if routed > 0 {
		status = deploymentStageInProgress
		body = fmt.Sprintf("%d of %d tasks have worker lane assignment.", routed, len(tasks))
	}
	stage := AIWorkflowStage{
		ID:        "contributor_routing",
		Title:     "Contributor routing",
		Body:      body,
		Status:    status,
		Tone:      deploymentStageTone(status),
		Reference: "project:" + project.ID,
		UpdatedAt: latest,
	}
	return aiWorkflowStageWithContract(project, stage, routed, outputIDs)
}

func aiWorkflowPRReviewStage(project *Project, logs []GeminiWebhookLog) AIWorkflowStage {
	status := deploymentStagePending
	body := "AI review and agent execution are waiting for matching workflow activity."
	updatedAt := project.CreatedAt
	reviewEvents := 0
	openedPulls := 0
	outputIDs := []string{}
	for _, log := range logs {
		if publicLiveFeedIsPullRequestOpened(&log) {
			openedPulls++
			if log.PullNumber > 0 {
				outputIDs = append(outputIDs, fmt.Sprintf("pr:%d", log.PullNumber))
			}
			if log.ReceivedAt.After(updatedAt) {
				updatedAt = log.ReceivedAt
			}
			continue
		}
		if strings.EqualFold(log.EventName, "repo_issues_synced") {
			continue
		}
		reviewEvents++
		if log.PullNumber > 0 {
			outputIDs = append(outputIDs, fmt.Sprintf("pr:%d", log.PullNumber))
		} else if action := strings.TrimSpace(log.Action); action != "" {
			outputIDs = append(outputIDs, "agent_action:"+action)
		}
		if log.ReceivedAt.After(updatedAt) {
			updatedAt = log.ReceivedAt
		}
	}
	if reviewEvents > 0 {
		status = deploymentStageComplete
		body = fmt.Sprintf("%d AI review or agent events are linked to this project.", reviewEvents)
	} else if openedPulls > 0 {
		status = deploymentStageInProgress
		body = fmt.Sprintf("%d opened PRs are waiting for AI review or agent execution.", openedPulls)
	}
	stage := AIWorkflowStage{
		ID:        "pr_review",
		Title:     "AI review and agent actions",
		Body:      body,
		Status:    status,
		Tone:      deploymentStageTone(status),
		Reference: aiWorkflowRepoReference(project),
		URL:       marketplacePublicRepoURL(project.RepoURL),
		UpdatedAt: updatedAt,
	}
	return aiWorkflowStageWithContract(project, stage, reviewEvents+openedPulls, stableStrings(outputIDs))
}

func aiWorkflowDeploymentStage(project *Project, deployment ProjectDeploymentResponse) AIWorkflowStage {
	status := deploymentStagePending
	if deployment.Status == "ready" {
		status = deploymentStageComplete
	} else if deployment.Status == "validating" {
		status = deploymentStageInProgress
	}
	producedCount := len(deployment.Signals)
	if producedCount == 0 && deployment.Progress > 0 {
		producedCount = 1
	}
	stage := AIWorkflowStage{
		ID:        "deployment_validation",
		Title:     "Deployment validation",
		Body:      fmt.Sprintf("Deployment validation is %d%% complete.", deployment.Progress),
		Status:    status,
		Tone:      deploymentStageTone(status),
		Reference: "project:" + project.ID,
		UpdatedAt: deployment.UpdatedAt,
	}
	return aiWorkflowStageWithContract(project, stage, producedCount, []string{"deployment:" + project.ID})
}

func aiWorkflowTaskOutputIDs(tasks []*Task) []string {
	outputIDs := []string{}
	for _, task := range tasks {
		if task == nil {
			continue
		}
		outputIDs = append(outputIDs, marketplaceBountyID(task.ProjectID, task.IssueNumber))
	}
	return stableStrings(outputIDs)
}

func aiWorkflowStageWithContract(project *Project, stage AIWorkflowStage, producedCount int, outputIDs []string) AIWorkflowStage {
	projectID := ""
	if project != nil {
		projectID = strings.TrimSpace(project.ID)
	}
	if producedCount < 0 {
		producedCount = 0
	}
	stage.ProducedCount = producedCount
	stage.OutputIDs = stableStrings(outputIDs)
	stage.ContextURLs = aiWorkflowStageContextURLs(project, stage.ID)
	stage.Checklist = aiWorkflowStageChecklist(stage.ID)
	stage.ActorLane = aiWorkflowStageActorLane(stage.ID)
	stage.InputEndpoint, stage.OutputEndpoint, stage.OutputProtocol, stage.ActionEndpoint, stage.ArtifactKind = aiWorkflowStageContract(projectID, stage.ID)
	stage.OutputProtocolURL = aiWorkflowProtocolSchemaURL(stage.OutputProtocol)
	return stage
}

func aiWorkflowStageActorLane(stageID string) string {
	switch stageID {
	case "repo_import":
		return "system"
	case "issue_scan", "task_generation", "reward_estimation":
		return "ai"
	case "contributor_routing", "pr_review":
		return "hybrid"
	case "deployment_validation":
		return "deployment_agent"
	default:
		return "system"
	}
}

func aiWorkflowStageChecklist(stageID string) []string {
	switch stageID {
	case "repo_import":
		return []string{
			"Attach source repository or funded bounty workspace.",
			"Resolve public repository context without private customer data.",
			"Expose repository context through the workflow protocol.",
		}
	case "issue_scan":
		return []string{
			"Scan issues, bugs, technical debt, dependencies, secrets, and security risk.",
			"Group findings into routeable delivery signals.",
			"Publish sanitized scan output for agents and contributors.",
		}
	case "task_generation":
		return []string{
			"Convert scanned findings or project scope into task protocol rows.",
			"Attach acceptance criteria, evidence requirements, and dependencies.",
			"Assign each task to human, agent, or hybrid worker lanes.",
		}
	case "reward_estimation":
		return []string{
			"Estimate complexity, delivery time, and reward allocation.",
			"Keep total task rewards inside the funded escrow budget.",
			"Expose estimate output through the public estimate protocol.",
		}
	case "contributor_routing":
		return []string{
			"Route tasks to contributors, AI agents, or hybrid review lanes.",
			"Attach claim, lease, proposal, and agent action endpoints.",
			"Publish routing reasons, readiness blockers, and output contracts.",
		}
	case "pr_review":
		return []string{
			"Validate pull requests, tests, security notes, and agent evidence.",
			"Record review, test, generate, deploy, or scan agent actions.",
			"Keep evidence linked to PR monitor, live feed, and ledger proof.",
		}
	case "deployment_validation":
		return []string{
			"Check deployment preview, environment health, release evidence, and rollback notes.",
			"Require deployment agent evidence for release-sensitive work.",
			"Publish deployment state before payout release.",
		}
	default:
		return []string{
			"Fetch context URLs.",
			"Run the required workflow checks.",
			"Attach sanitized evidence.",
		}
	}
}

func aiWorkflowStageContextURLs(project *Project, stageID string) map[string]string {
	projectID := ""
	if project != nil {
		projectID = strings.TrimSpace(project.ID)
	}
	context := map[string]string{
		"protocol_manifest": "/api/public/protocol",
		"agent_queue":       "/api/public/protocol/agent-queue",
	}
	if projectID != "" {
		context["task_protocol"] = "/api/public/protocol/tasks?project_id=" + projectID
		context["workflow"] = "/api/public/projects/" + projectID + "/workflow"
		context["ai_workflow"] = "/api/public/projects/" + projectID + "/ai-workflow"
		context["repo_scan"] = "/api/public/projects/" + projectID + "/repo-scan"
		context["pull_requests"] = "/api/public/projects/" + projectID + "/pull-requests"
		context["deployment"] = "/api/public/projects/" + projectID + "/deployment"
	}
	if project != nil {
		if repoURL := marketplacePublicRepoURL(projectSourceRepoURL(project)); repoURL != "" {
			context["repository"] = repoURL
		}
	}
	if projectID == "" {
		return context
	}
	switch stageID {
	case "issue_scan":
		context["repo_sync"] = "/api/projects/" + projectID + "/repo-sync"
	case "reward_estimation":
		context["price_estimate"] = "/api/projects/evaluate-price"
	case "contributor_routing":
		context["routing"] = "/api/projects/" + projectID + "/routing"
	case "pr_review":
		context["agent_action_template"] = "/api/projects/" + projectID + "/agent-actions"
	case "deployment_validation":
		context["deployment_evidence"] = "/api/public/projects/" + projectID + "/deployment"
	}
	return context
}

func aiWorkflowStageContract(projectID, stageID string) (inputEndpoint, outputEndpoint, outputProtocol, actionEndpoint, artifactKind string) {
	projectPath := func(path string) string {
		if projectID == "" {
			return strings.ReplaceAll(path, "{id}", "project")
		}
		return strings.ReplaceAll(path, "{id}", projectID)
	}
	switch stageID {
	case "repo_import":
		return "/api/public/repo/issues", "/api/public/repo/issues", "mergeos.repo-import.v1", "", "repository_context"
	case "issue_scan":
		return "/api/public/repo/issues", projectPath("/api/public/projects/{id}/repo-scan"), "mergeos.scan.v1", projectPath("/api/projects/{id}/repo-sync"), "repository_scan"
	case "task_generation":
		return projectPath("/api/public/projects/{id}/repo-scan"), "/api/public/protocol/tasks?project_id=" + projectPath("{id}"), "mergeos.task.v1", "", "task_protocol"
	case "reward_estimation":
		return "/api/public/protocol/tasks?project_id=" + projectPath("{id}"), "/api/projects/evaluate-price", "mergeos.estimate.v1", "/api/projects/evaluate-price", "project_estimate"
	case "contributor_routing":
		return projectPath("/api/public/projects/{id}/workflow"), projectPath("/api/projects/{id}/routing"), "mergeos.routing.v1", "", "routing_plan"
	case "pr_review":
		return projectPath("/api/public/projects/{id}/pull-requests"), projectPath("/api/projects/{id}/agent-actions"), "mergeos.agent-action.v1", projectPath("/api/projects/{id}/agent-actions"), "agent_action"
	case "deployment_validation":
		return projectPath("/api/public/projects/{id}/pull-requests"), projectPath("/api/public/projects/{id}/deployment"), "mergeos.deployment.v1", projectPath("/api/projects/{id}/agent-actions"), "deployment_evidence"
	default:
		return "/api/public/protocol", projectPath("/api/public/projects/{id}/ai-workflow"), "mergeos.ai-workflow.v1", "", "workflow_stage"
	}
}

func aiWorkflowProtocolSchemaURL(protocol string) string {
	switch protocol {
	case "mergeos.agent-action.v1":
		return "/protocol/agent-action.v1.schema.json"
	case "mergeos.ai-workflow.v1":
		return "/protocol/ai-workflow.v1.schema.json"
	case "mergeos.deployment.v1":
		return "/protocol/deployment.v1.schema.json"
	case "mergeos.estimate.v1":
		return "/protocol/estimate.v1.schema.json"
	case "mergeos.repo-import.v1":
		return "/protocol/repo-import.v1.schema.json"
	case "mergeos.routing.v1":
		return "/protocol/routing.v1.schema.json"
	case "mergeos.scan.v1":
		return "/protocol/scan.v1.schema.json"
	case "mergeos.task.v1":
		return "/protocol/task.v1.schema.json"
	case "mergeos.workflow.v1":
		return "/protocol/workflow.v1.schema.json"
	default:
		return "/protocol/protocol.v1.schema.json"
	}
}

func (s *Store) aiWorkflowLogsLocked(project *Project) []GeminiWebhookLog {
	logs := []GeminiWebhookLog{}
	for _, log := range s.geminiWebhookLogs {
		if !deploymentLogMatchesProject(log, project) {
			continue
		}
		logs = append(logs, *log)
	}
	sort.Slice(logs, func(i, j int) bool {
		return logs[i].ReceivedAt.After(logs[j].ReceivedAt)
	})
	if len(logs) > 8 {
		return logs[:8]
	}
	return logs
}

func aiWorkflowSignals(project *Project, logs []GeminiWebhookLog, deployment ProjectDeploymentResponse) []AIWorkflowSignal {
	signals := []AIWorkflowSignal{}
	for _, log := range logs {
		reference := publicLiveFeedAIReference(&log)
		signals = append(signals, AIWorkflowSignal{
			ID:              aiWorkflowSignalID(project.ID, log),
			Type:            publicLiveFeedAIType(&log),
			Title:           publicLiveFeedAITitle(&log),
			Body:            publicLiveFeedAIBody(&log),
			Status:          publicLiveFeedStatus(log.Status),
			Reference:       reference,
			URL:             publicLiveFeedURL(log.CommentURL),
			DelegatedBy:     log.DelegatedBy,
			DesignAgent:     log.DesignAgent,
			SubagentType:    log.SubagentType,
			DelegationChain: normalizeAgentDelegationChain(log.DelegationChain, log.DelegatedBy, log.DesignAgent, log.SubagentType),
			SourceFindingID: log.SourceFindingID,
			Signal:          log.Signal,
			Path:            log.Path,
			CreatedAt:       log.ReceivedAt,
		})
	}
	signals = append(signals, AIWorkflowSignal{
		ID:        "deployment:" + project.ID,
		Type:      "deployment_validation",
		Title:     "Deployment validation",
		Body:      fmt.Sprintf("Deployment validation is %d%% complete.", deployment.Progress),
		Status:    deployment.Status,
		Reference: "project:" + project.ID,
		CreatedAt: deployment.UpdatedAt,
	})
	sort.Slice(signals, func(i, j int) bool {
		if signals[i].CreatedAt.Equal(signals[j].CreatedAt) {
			return signals[i].ID > signals[j].ID
		}
		return signals[i].CreatedAt.After(signals[j].CreatedAt)
	})
	if len(signals) > 8 {
		return signals[:8]
	}
	return signals
}

func aiWorkflowSignalID(projectID string, log GeminiWebhookLog) string {
	action := strings.TrimSpace(log.Action)
	if action == "" {
		action = strings.TrimSpace(log.EventName)
	}
	if action == "" {
		action = strings.TrimSpace(log.Status)
	}
	parts := []string{
		strings.TrimSpace(projectID),
		publicLiveFeedAIType(&log),
		slug(action),
		log.ReceivedAt.UTC().Format("20060102T150405"),
	}
	if log.PullNumber > 0 {
		parts = append(parts, fmt.Sprintf("pr-%d", log.PullNumber))
	}
	return "ai:" + publicTaskProtocolID(strings.Join(parts, ":"))
}

func aiWorkflowRepoReference(project *Project) string {
	if project == nil {
		return ""
	}
	for _, line := range strings.Split(project.Brief, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(strings.ToLower(line), "source repository:") {
			_, value, _ := strings.Cut(line, ":")
			if reference := aiWorkflowSafeReference(strings.TrimSpace(value)); reference != "" {
				return reference
			}
		}
	}
	if reference := aiWorkflowSafeReference(project.RepoURL); reference != "" {
		return reference
	}
	if reference := aiWorkflowSafeReference(project.BountyRepoName); reference != "" {
		return reference
	}
	if strings.TrimSpace(project.ID) != "" {
		return "project:" + project.ID
	}
	return ""
}

func aiWorkflowSafeReference(value string) string {
	value = sanitizeLedgerReferenceValue(value)
	if value == "" {
		return ""
	}
	if publicLiveFeedURL(value) != "" {
		return value
	}
	lower := strings.ToLower(value)
	if strings.HasPrefix(value, "/") || strings.HasPrefix(value, ".") || strings.HasPrefix(lower, "file:") {
		return ""
	}
	if len(value) >= 2 && value[1] == ':' {
		return ""
	}
	if strings.Contains(value, "\\") {
		return ""
	}
	return value
}
