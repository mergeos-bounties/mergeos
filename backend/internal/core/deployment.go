package core

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

const (
	deploymentStageComplete   = "complete"
	deploymentStageInProgress = "in_progress"
	deploymentStagePending    = "pending"
)

func (s *Store) CanAccessProject(userID string, role UserRole, projectID string) bool {
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return false
	}
	if normalizeRole(role) == RoleAdmin {
		return true
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	project, ok := s.projects[projectID]
	return ok && project.ClientUserID == strings.TrimSpace(userID)
}

func (s *Store) ProjectDeployment(projectID string) (ProjectDeploymentResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	project, ok := s.projects[strings.TrimSpace(projectID)]
	if !ok {
		return ProjectDeploymentResponse{}, errors.New("project not found")
	}
	return s.projectDeploymentLocked(project), nil
}

func (s *Store) projectDeploymentLocked(project *Project) ProjectDeploymentResponse {
	tasks := s.projectDeploymentTasksLocked(project)
	stages := []DeploymentStage{
		deploymentRepoStage(project),
		deploymentTaskPlanStage(project, tasks),
		deploymentTaskCategoryStage(
			"qa_validation",
			"QA validation",
			tasks,
			project.CreatedAt,
			[]string{"qa", "quality", "accessibility", "a11y", "test", "smoke", "preview", "validation"},
			"QA and customer preview evidence has been accepted.",
			"QA and customer preview work is open for validation.",
			"QA validation will start after a matching task is created.",
		),
		deploymentTaskCategoryStage(
			"deployment_handoff",
			"Deployment handoff",
			tasks,
			project.CreatedAt,
			[]string{"deploy", "deployment", "devops", "handoff", "release", "pipeline", "environment"},
			"Deployment pipeline and handoff notes have been accepted.",
			"Deployment handoff is open and waiting on delivery proof.",
			"Deployment handoff will start after a matching task is created.",
		),
		deploymentReleaseGateStage(project, tasks),
	}
	signals := s.deploymentSignalsLocked(project, tasks)

	updatedAt := project.CreatedAt
	completed := 0
	inProgress := 0
	for _, stage := range stages {
		if stage.UpdatedAt.After(updatedAt) {
			updatedAt = stage.UpdatedAt
		}
		switch stage.Status {
		case deploymentStageComplete:
			completed++
		case deploymentStageInProgress:
			inProgress++
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
	} else if inProgress > 0 || completed > 0 {
		status = "validating"
	}

	return ProjectDeploymentResponse{
		ProjectID:    project.ID,
		ProjectTitle: publicLiveFeedProjectTitle(project),
		Status:       status,
		Progress:     progress,
		UpdatedAt:    updatedAt,
		Stages:       stages,
		Signals:      signals,
	}
}

func (s *Store) projectDeploymentTasksLocked(project *Project) []*Task {
	if project == nil {
		return []*Task{}
	}
	tasksByID := map[string]*Task{}
	for _, task := range project.Tasks {
		if task != nil && strings.TrimSpace(task.ID) != "" {
			tasksByID[task.ID] = task
		}
	}
	for _, task := range s.tasks {
		if task != nil && task.ProjectID == project.ID && strings.TrimSpace(task.ID) != "" {
			tasksByID[task.ID] = task
		}
	}

	tasks := make([]*Task, 0, len(tasksByID))
	for _, task := range tasksByID {
		tasks = append(tasks, task)
	}
	sortTasks(tasks)
	return tasks
}

func deploymentRepoStage(project *Project) DeploymentStage {
	body := "Repository and bounty workspace are available for delivery."
	status := deploymentStageComplete
	reference := strings.TrimSpace(project.BountyRepoName)
	url := marketplacePublicRepoURL(project.RepoURL)
	if reference == "" && url == "" {
		status = deploymentStagePending
		body = "Repository handoff is waiting for project repo context."
	}
	if reference == "" {
		reference = url
	}
	return DeploymentStage{
		ID:        "repo_handoff",
		Title:     "Repository handoff",
		Body:      body,
		Status:    status,
		Tone:      deploymentStageTone(status),
		Reference: sanitizeLedgerReferenceValue(reference),
		URL:       url,
		UpdatedAt: project.CreatedAt,
	}
}

func deploymentTaskPlanStage(project *Project, tasks []*Task) DeploymentStage {
	status := deploymentStagePending
	body := "Task routing will appear once MergeOS splits the funded scope."
	if len(tasks) > 0 {
		status = deploymentStageComplete
		body = fmt.Sprintf("%d payable tasks are routed across human, agent, and hybrid lanes.", len(tasks))
	}
	return DeploymentStage{
		ID:        "task_routing",
		Title:     "Task routing",
		Body:      body,
		Status:    status,
		Tone:      deploymentStageTone(status),
		Reference: "project:" + project.ID,
		UpdatedAt: project.CreatedAt,
	}
}

func deploymentTaskCategoryStage(id, title string, tasks []*Task, fallbackUpdatedAt time.Time, keywords []string, completeBody, inProgressBody, pendingBody string) DeploymentStage {
	task := deploymentBestTask(tasks, keywords)
	if task == nil {
		return DeploymentStage{
			ID:        id,
			Title:     title,
			Body:      pendingBody,
			Status:    deploymentStagePending,
			Tone:      deploymentStageTone(deploymentStagePending),
			UpdatedAt: fallbackUpdatedAt,
		}
	}

	status := deploymentStageInProgress
	body := inProgressBody
	if task.Status == TaskAccepted {
		status = deploymentStageComplete
		body = completeBody
	}
	updatedAt := deploymentTaskUpdatedAt(task)
	return DeploymentStage{
		ID:                    id,
		Title:                 title,
		Body:                  body,
		Status:                status,
		Tone:                  deploymentStageTone(status),
		SourceTaskIssueNumber: task.IssueNumber,
		Reference:             deploymentTaskReference(task),
		URL:                   marketplacePublicRepoURL(task.IssueURL),
		UpdatedAt:             updatedAt,
	}
}

func deploymentReleaseGateStage(project *Project, tasks []*Task) DeploymentStage {
	accepted := 0
	updatedAt := project.CreatedAt
	for _, task := range tasks {
		if task.Status == TaskAccepted {
			accepted++
		}
		if taskUpdated := deploymentTaskUpdatedAt(task); taskUpdated.After(updatedAt) {
			updatedAt = taskUpdated
		}
	}

	status := deploymentStagePending
	if len(tasks) > 0 && accepted == len(tasks) {
		status = deploymentStageComplete
	} else if accepted > 0 {
		status = deploymentStageInProgress
	}
	return DeploymentStage{
		ID:        "release_gate",
		Title:     "Release gate",
		Body:      fmt.Sprintf("%d of %d delivery tasks are accepted and paid.", accepted, len(tasks)),
		Status:    status,
		Tone:      deploymentStageTone(status),
		Reference: "project:" + project.ID,
		UpdatedAt: updatedAt,
	}
}

func deploymentBestTask(tasks []*Task, keywords []string) *Task {
	var best *Task
	for _, task := range tasks {
		if task == nil || !deploymentTaskMatches(task, keywords) {
			continue
		}
		if best == nil || deploymentTaskScore(task) > deploymentTaskScore(best) {
			best = task
		}
	}
	return best
}

func deploymentTaskMatches(task *Task, keywords []string) bool {
	haystack := strings.ToLower(strings.Join([]string{
		task.Title,
		task.Acceptance,
		string(task.RequiredWorkerKind),
		task.SuggestedAgentType,
		task.AgentType,
	}, " "))
	for _, keyword := range keywords {
		if strings.Contains(haystack, strings.ToLower(keyword)) {
			return true
		}
	}
	return false
}

func deploymentTaskScore(task *Task) int64 {
	score := int64(task.IssueNumber)
	if task.Status == TaskAccepted {
		score += 1_000_000
	}
	return score
}

func deploymentTaskUpdatedAt(task *Task) time.Time {
	if task == nil {
		return time.Time{}
	}
	if task.AcceptedAt != nil {
		return *task.AcceptedAt
	}
	return task.CreatedAt
}

func deploymentTaskReference(task *Task) string {
	if task == nil {
		return ""
	}
	if url := publicLiveFeedURL(task.IssueURL); url != "" {
		return url
	}
	if task.IssueNumber > 0 {
		return fmt.Sprintf("issue:%d", task.IssueNumber)
	}
	return "task"
}

func (s *Store) deploymentSignalsLocked(project *Project, tasks []*Task) []DeploymentSignal {
	projectIDs := map[string]bool{project.ID: true}
	taskProjectIDs := map[string]string{}
	taskIDs := map[string]bool{}
	taskIssueNumbers := map[string]int{}
	for _, task := range tasks {
		taskIDs[task.ID] = true
		taskProjectIDs[task.ID] = project.ID
		taskIssueNumbers[task.ID] = task.IssueNumber
	}

	signals := []DeploymentSignal{}
	for _, entry := range s.ledger {
		if !ledgerEntryMatches(entry, projectIDs, taskIDs) {
			continue
		}
		projectID, taskID := publicLedgerScope(entry, projectIDs, taskProjectIDs)
		reference := deploymentLedgerReference(projectID, taskID, entry.Sequence, entry.Reference, taskIssueNumbers)
		signals = append(signals, DeploymentSignal{
			ID:        fmt.Sprintf("ledger:%d", entry.Sequence),
			Type:      "ledger_" + entry.Type,
			Title:     publicLiveFeedLedgerTitle(entry.Type),
			Body:      publicLiveFeedLedgerBody(entry, publicLiveFeedProjectTitle(project)),
			Status:    "verified",
			Reference: reference,
			URL:       publicLiveFeedReferenceURL(reference),
			CreatedAt: entry.CreatedAt,
		})
	}

	for _, log := range s.geminiWebhookLogs {
		if !deploymentLogMatchesProject(log, project) {
			continue
		}
		reference := publicLiveFeedAIReference(log)
		signals = append(signals, DeploymentSignal{
			ID:        "ai:" + log.ID,
			Type:      publicLiveFeedAIType(log),
			Title:     publicLiveFeedAITitle(log),
			Body:      publicLiveFeedAIBody(log),
			Status:    publicLiveFeedStatus(log.Status),
			Reference: reference,
			URL:       publicLiveFeedURL(log.CommentURL),
			CreatedAt: log.ReceivedAt,
		})
	}

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

func deploymentLedgerReference(projectID, taskID string, sequence int, reference string, taskIssueNumbers map[string]int) string {
	if pullReference := publicPullLedgerReference(reference); pullReference != "" {
		return pullReference
	}
	if projectID == "" {
		return fmt.Sprintf("ledger:%d", sequence)
	}
	if taskID != "" {
		if issueNumber := taskIssueNumbers[taskID]; issueNumber > 0 {
			return fmt.Sprintf("project:%s;issue:%d", projectID, issueNumber)
		}
		return "project:" + projectID + ";task"
	}
	return "project:" + projectID
}

func deploymentLogMatchesProject(log *GeminiWebhookLog, project *Project) bool {
	if log == nil || project == nil {
		return false
	}
	repo := strings.ToLower(strings.TrimSpace(log.Repository))
	if repo == "" {
		return false
	}
	candidates := []string{
		project.BountyRepoName,
		project.RepoURL,
	}
	for _, candidate := range candidates {
		if strings.Contains(strings.ToLower(candidate), repo) {
			return true
		}
	}
	return false
}

func deploymentStageTone(status string) string {
	switch status {
	case deploymentStageComplete:
		return "green"
	case deploymentStageInProgress:
		return "blue"
	default:
		return "amber"
	}
}
