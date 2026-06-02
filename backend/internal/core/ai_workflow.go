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

func (s *Store) projectAIWorkflowLocked(project *Project) ProjectAIWorkflowResponse {
	tasks := s.projectDeploymentTasksLocked(project)
	logs := s.aiWorkflowLogsLocked(project)
	deployment := s.projectDeploymentLocked(project)
	stages := []AIWorkflowStage{
		aiWorkflowRepoStage(project),
		aiWorkflowIssueScanStage(project, tasks),
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
		ProjectID:     project.ID,
		ProjectTitle:  publicLiveFeedProjectTitle(project),
		Status:        status,
		Progress:      progress,
		TaskCount:     len(tasks),
		AIActionCount: len(logs),
		UpdatedAt:     updatedAt,
		Stages:        stages,
		Signals:       signals,
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

func aiWorkflowRepoStage(project *Project) AIWorkflowStage {
	reference := aiWorkflowRepoReference(project)
	status := deploymentStagePending
	body := "Repo context is waiting for import or bounty workspace creation."
	if reference != "" {
		status = deploymentStageComplete
		body = "Repository context is attached to the delivery workflow."
	}
	return AIWorkflowStage{
		ID:        "repo_import",
		Title:     "Repository context",
		Body:      body,
		Status:    status,
		Tone:      deploymentStageTone(status),
		Reference: sanitizeLedgerReferenceValue(reference),
		URL:       marketplacePublicRepoURL(project.RepoURL),
		UpdatedAt: project.CreatedAt,
	}
}

func aiWorkflowIssueScanStage(project *Project, tasks []*Task) AIWorkflowStage {
	status := deploymentStagePending
	body := "Issue scan will complete when the project has task rows."
	if len(tasks) > 0 {
		status = deploymentStageComplete
		body = fmt.Sprintf("Issue analysis produced %d payable task rows.", len(tasks))
	}
	return AIWorkflowStage{
		ID:        "issue_scan",
		Title:     "Issue scan",
		Body:      body,
		Status:    status,
		Tone:      deploymentStageTone(status),
		Reference: "project:" + project.ID,
		UpdatedAt: project.CreatedAt,
	}
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
	status := deploymentStagePending
	body := "Reward estimation is waiting for task budget allocation."
	if allocated > 0 {
		status = deploymentStageComplete
		body = fmt.Sprintf("Reward estimation allocated %s %s across delivery tasks.", formatTokenAmount(allocated), normalizedTokenSymbol(tokenSymbol))
	}
	return AIWorkflowStage{
		ID:        "reward_estimation",
		Title:     "Reward estimation",
		Body:      body,
		Status:    status,
		Tone:      deploymentStageTone(status),
		Reference: "project:" + project.ID,
		UpdatedAt: latest,
	}
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
	status := deploymentStagePending
	body := "Contributor routing is waiting for worker lane assignment."
	if len(tasks) > 0 && routed == len(tasks) {
		status = deploymentStageComplete
		body = fmt.Sprintf("%d tasks are routed to human, agent, or hybrid lanes.", routed)
	} else if routed > 0 {
		status = deploymentStageInProgress
		body = fmt.Sprintf("%d of %d tasks have worker lane assignment.", routed, len(tasks))
	}
	return AIWorkflowStage{
		ID:        "contributor_routing",
		Title:     "Contributor routing",
		Body:      body,
		Status:    status,
		Tone:      deploymentStageTone(status),
		Reference: "project:" + project.ID,
		UpdatedAt: latest,
	}
}

func aiWorkflowPRReviewStage(project *Project, logs []GeminiWebhookLog) AIWorkflowStage {
	status := deploymentStagePending
	body := "AI review and agent execution are waiting for matching workflow activity."
	updatedAt := project.CreatedAt
	reviewEvents := 0
	openedPulls := 0
	for _, log := range logs {
		if publicLiveFeedIsPullRequestOpened(&log) {
			openedPulls++
			if log.ReceivedAt.After(updatedAt) {
				updatedAt = log.ReceivedAt
			}
			continue
		}
		if strings.EqualFold(log.EventName, "repo_issues_synced") {
			continue
		}
		reviewEvents++
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
	return AIWorkflowStage{
		ID:        "pr_review",
		Title:     "AI review and agent actions",
		Body:      body,
		Status:    status,
		Tone:      deploymentStageTone(status),
		Reference: aiWorkflowRepoReference(project),
		URL:       marketplacePublicRepoURL(project.RepoURL),
		UpdatedAt: updatedAt,
	}
}

func aiWorkflowDeploymentStage(project *Project, deployment ProjectDeploymentResponse) AIWorkflowStage {
	status := deploymentStagePending
	if deployment.Status == "ready" {
		status = deploymentStageComplete
	} else if deployment.Status == "validating" {
		status = deploymentStageInProgress
	}
	return AIWorkflowStage{
		ID:        "deployment_validation",
		Title:     "Deployment validation",
		Body:      fmt.Sprintf("Deployment validation is %d%% complete.", deployment.Progress),
		Status:    status,
		Tone:      deploymentStageTone(status),
		Reference: "project:" + project.ID,
		UpdatedAt: deployment.UpdatedAt,
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
			ID:        "ai:" + log.ID,
			Type:      publicLiveFeedAIType(&log),
			Title:     publicLiveFeedAITitle(&log),
			Body:      publicLiveFeedAIBody(&log),
			Status:    publicLiveFeedStatus(log.Status),
			Reference: reference,
			URL:       publicLiveFeedURL(log.CommentURL),
			CreatedAt: log.ReceivedAt,
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

func aiWorkflowRepoReference(project *Project) string {
	if project == nil {
		return ""
	}
	for _, line := range strings.Split(project.Brief, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(strings.ToLower(line), "source repository:") {
			_, value, _ := strings.Cut(line, ":")
			return strings.TrimSpace(value)
		}
	}
	if strings.TrimSpace(project.RepoURL) != "" {
		return project.RepoURL
	}
	return project.BountyRepoName
}
