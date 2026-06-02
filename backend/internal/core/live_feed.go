package core

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

const (
	defaultPublicLiveFeedLimit = 40
	maxPublicLiveFeedLimit     = 120
)

func (s *Store) PublicLiveFeed(limit int) PublicLiveFeedResponse {
	limit = normalizePublicLiveFeedLimit(limit)

	s.mu.RLock()
	defer s.mu.RUnlock()

	projectIDs := map[string]bool{}
	taskProjectIDs := map[string]string{}
	response := PublicLiveFeedResponse{
		Stats: PublicLiveFeedStats{
			ProjectCount:     len(s.projects),
			LedgerEntryCount: len(s.ledger),
			AIActionCount:    len(s.geminiWebhookLogs),
			TokenSymbol:      s.cfg.TokenSymbol,
		},
		Items: []PublicLiveFeedItem{},
	}
	touch := func(value time.Time) {
		if value.IsZero() {
			return
		}
		if response.Stats.UpdatedAt == nil || value.After(*response.Stats.UpdatedAt) {
			updatedAt := value
			response.Stats.UpdatedAt = &updatedAt
		}
	}

	for _, project := range s.projects {
		projectIDs[project.ID] = true
		response.Stats.TotalBudgetCents += project.BudgetCents
		touch(project.CreatedAt)
		for _, task := range project.Tasks {
			taskProjectIDs[task.ID] = project.ID
		}
		response.Items = append(response.Items, publicProjectLiveFeedItem(project))
		deployment := s.projectDeploymentLocked(project)
		touch(deployment.UpdatedAt)
		response.Items = append(response.Items, publicDeploymentLiveFeedItem(deployment))
	}

	for _, task := range s.tasks {
		project := s.projects[task.ProjectID]
		if task.Status == TaskAccepted {
			response.Stats.AcceptedTaskCount++
			if task.AcceptedAt != nil {
				touch(*task.AcceptedAt)
			}
			response.Items = append(response.Items, publicTaskAcceptedLiveFeedItem(task, project))
			continue
		}
		response.Stats.OpenTaskCount++
		touch(task.CreatedAt)
		response.Items = append(response.Items, publicTaskOpenLiveFeedItem(task, project))
	}

	for _, entry := range s.ledger {
		touch(entry.CreatedAt)
		response.Items = append(response.Items, publicLedgerLiveFeedItem(entry, projectIDs, taskProjectIDs, s.projects))
	}

	for _, log := range s.geminiWebhookLogs {
		touch(log.ReceivedAt)
		response.Items = append(response.Items, publicAILiveFeedItem(log))
	}

	sort.Slice(response.Items, func(i, j int) bool {
		if response.Items[i].CreatedAt.Equal(response.Items[j].CreatedAt) {
			return response.Items[i].ID > response.Items[j].ID
		}
		return response.Items[i].CreatedAt.After(response.Items[j].CreatedAt)
	})
	if len(response.Items) > limit {
		response.Items = response.Items[:limit]
	}
	return response
}

func normalizePublicLiveFeedLimit(limit int) int {
	if limit <= 0 {
		return defaultPublicLiveFeedLimit
	}
	if limit > maxPublicLiveFeedLimit {
		return maxPublicLiveFeedLimit
	}
	return limit
}

func publicProjectLiveFeedItem(project *Project) PublicLiveFeedItem {
	taskCount := len(project.Tasks)
	return PublicLiveFeedItem{
		ID:           "project:" + project.ID,
		Type:         "project_funded",
		Title:        "Project funded",
		Body:         fmt.Sprintf("%s opened with %d payable tasks and escrow-backed delivery.", publicLiveFeedProjectTitle(project), taskCount),
		ProjectID:    project.ID,
		ProjectTitle: publicLiveFeedProjectTitle(project),
		Actor:        marketplaceClientDisplayName(project),
		AmountCents:  project.BudgetCents,
		Reference:    "project:" + project.ID,
		URL:          marketplacePublicRepoURL(project.RepoURL),
		Status:       string(project.Status),
		CreatedAt:    project.CreatedAt,
	}
}

func publicDeploymentLiveFeedItem(deployment ProjectDeploymentResponse) PublicLiveFeedItem {
	status := strings.TrimSpace(deployment.Status)
	if status == "" {
		status = "queued"
	}
	title := "Deployment validation running"
	if status == "ready" {
		title = "Deployment validation ready"
	}
	projectTitle := strings.TrimSpace(deployment.ProjectTitle)
	if projectTitle == "" {
		projectTitle = "MergeOS project"
	}
	return PublicLiveFeedItem{
		ID:           "deployment:" + deployment.ProjectID,
		Type:         "deployment_validation",
		Title:        title,
		Body:         fmt.Sprintf("%s deployment gate is %d%% complete across QA, handoff, and release checks.", projectTitle, deployment.Progress),
		ProjectID:    deployment.ProjectID,
		ProjectTitle: projectTitle,
		Actor:        "mergeos-orchestrator",
		Reference:    "project:" + deployment.ProjectID,
		Status:       status,
		CreatedAt:    deployment.UpdatedAt,
	}
}

func publicTaskOpenLiveFeedItem(task *Task, project *Project) PublicLiveFeedItem {
	projectID, projectTitle := publicLiveFeedProjectScope(task, project)
	return PublicLiveFeedItem{
		ID:           publicLiveFeedTaskID("task-open", task),
		Type:         "task_opened",
		Title:        fmt.Sprintf("Task #%d opened", task.IssueNumber),
		Body:         publicLiveFeedTaskBody(task),
		ProjectID:    projectID,
		ProjectTitle: projectTitle,
		Actor:        publicLiveFeedWorkerKind(task.RequiredWorkerKind, task.SuggestedAgentType),
		AmountCents:  task.RewardCents,
		Reference:    publicTaskReference(task),
		URL:          marketplacePublicRepoURL(task.IssueURL),
		Status:       string(task.Status),
		CreatedAt:    task.CreatedAt,
	}
}

func publicTaskAcceptedLiveFeedItem(task *Task, project *Project) PublicLiveFeedItem {
	projectID, projectTitle := publicLiveFeedProjectScope(task, project)
	createdAt := task.CreatedAt
	if task.AcceptedAt != nil {
		createdAt = *task.AcceptedAt
	}
	return PublicLiveFeedItem{
		ID:           publicLiveFeedTaskID("task-accepted", task),
		Type:         "task_accepted",
		Title:        fmt.Sprintf("Task #%d accepted", task.IssueNumber),
		Body:         publicLiveFeedTaskBody(task),
		ProjectID:    projectID,
		ProjectTitle: projectTitle,
		Actor:        publicLiveFeedActor(task.WorkerID, task.AgentType),
		AmountCents:  task.RewardCents,
		Reference:    publicTaskReference(task),
		URL:          marketplacePublicRepoURL(task.IssueURL),
		Status:       string(task.Status),
		CreatedAt:    createdAt,
	}
}

func publicLedgerLiveFeedItem(entry LedgerEntry, projectIDs map[string]bool, taskProjectIDs map[string]string, projects map[string]*Project) PublicLiveFeedItem {
	projectID, taskID := publicLedgerScope(entry, projectIDs, taskProjectIDs)
	projectTitle := ""
	if projectID != "" {
		projectTitle = publicLiveFeedProjectTitle(projects[projectID])
	}
	reference := publicLedgerReference(projectID, taskID, entry.Sequence, entry.Reference)
	return PublicLiveFeedItem{
		ID:           fmt.Sprintf("ledger:%d", entry.Sequence),
		Type:         "ledger_" + entry.Type,
		Title:        publicLiveFeedLedgerTitle(entry.Type),
		Body:         publicLiveFeedLedgerBody(entry, projectTitle),
		ProjectID:    projectID,
		ProjectTitle: projectTitle,
		Actor:        publicLedgerAccount(entry.ToAccount, projectID, taskID),
		AmountCents:  entry.AmountCents,
		Reference:    reference,
		URL:          publicLiveFeedReferenceURL(reference),
		Status:       "verified",
		CreatedAt:    entry.CreatedAt,
	}
}

func publicAILiveFeedItem(log *GeminiWebhookLog) PublicLiveFeedItem {
	reference := publicLiveFeedAIReference(log)
	return PublicLiveFeedItem{
		ID:        "ai:" + log.ID,
		Type:      "ai_review",
		Title:     publicLiveFeedAITitle(log),
		Body:      publicLiveFeedAIBody(log),
		Actor:     publicLiveFeedGitHubActor(log.Sender),
		Reference: reference,
		URL:       publicLiveFeedURL(log.CommentURL),
		Status:    publicLiveFeedStatus(log.Status),
		CreatedAt: log.ReceivedAt,
	}
}

func publicLiveFeedProjectScope(task *Task, project *Project) (string, string) {
	if project != nil {
		return project.ID, publicLiveFeedProjectTitle(project)
	}
	return task.ProjectID, ""
}

func publicLiveFeedProjectTitle(project *Project) string {
	if project == nil {
		return "MergeOS project"
	}
	if title := strings.TrimSpace(project.Title); title != "" {
		return title
	}
	return "MergeOS project"
}

func publicLiveFeedTaskBody(task *Task) string {
	title := strings.TrimSpace(task.Title)
	if title == "" {
		title = "Untitled task"
	}
	acceptance := compactText(task.Acceptance)
	if acceptance == "" {
		return title
	}
	return title + " - " + acceptance
}

func publicTaskReference(task *Task) string {
	if url := publicLiveFeedURL(task.IssueURL); url != "" {
		return url
	}
	if task.IssueNumber > 0 {
		return fmt.Sprintf("issue:%d", task.IssueNumber)
	}
	return "task"
}

func publicLiveFeedTaskID(prefix string, task *Task) string {
	scope := strings.TrimSpace(task.ProjectID)
	if scope == "" {
		scope = "project"
	}
	if task.IssueNumber > 0 {
		return fmt.Sprintf("%s:%s:%d", prefix, scope, task.IssueNumber)
	}
	return prefix + ":" + scope
}

func publicLiveFeedWorkerKind(kind WorkerKind, agentType string) string {
	if strings.TrimSpace(agentType) != "" {
		return marketplaceTitle(agentType)
	}
	if kind == WorkerAgent || kind == WorkerHybrid {
		return string(kind)
	}
	return "human"
}

func publicLiveFeedActor(workerID, agentType string) string {
	workerID = strings.TrimSpace(workerID)
	if strings.TrimSpace(agentType) != "" {
		return marketplaceWorkerName(workerID, agentType)
	}
	if strings.HasPrefix(workerID, "github:") {
		return githubWorkerAccount(workerID)
	}
	if strings.HasPrefix(workerID, "worker:github:") {
		return githubWorkerAccount(strings.TrimPrefix(workerID, "worker:"))
	}
	if workerID == "" {
		return ""
	}
	return "worker:contributor"
}

func publicLiveFeedGitHubActor(sender string) string {
	sender = sanitizeLedgerReferenceValue(sender)
	if sender == "" {
		return ""
	}
	if strings.HasPrefix(sender, "github:") {
		return githubWorkerAccount(sender)
	}
	return githubWorkerAccount("github:" + strings.TrimPrefix(sender, "@"))
}

func publicLiveFeedLedgerTitle(entryType string) string {
	switch entryType {
	case "payment_verified":
		return "Payment verified"
	case "token_mint":
		return "MRG tokens minted"
	case "platform_fee":
		return "Platform fee recorded"
	case "project_reserve":
		return "Project escrow reserved"
	case "task_reserve":
		return "Task reward reserved"
	case "task_payment":
		return "Task payout released"
	case "manual_credit":
		return "Manual MRG credit"
	default:
		return marketplaceTitle(entryType)
	}
}

func publicLiveFeedLedgerBody(entry LedgerEntry, projectTitle string) string {
	scope := projectTitle
	if scope == "" {
		scope = "MergeOS public ledger"
	}
	return fmt.Sprintf("%s recorded %s.", scope, publicLiveFeedLedgerTitle(entry.Type))
}

func publicLiveFeedAITitle(log *GeminiWebhookLog) string {
	if log.PullNumber > 0 {
		return fmt.Sprintf("AI reviewed PR #%d", log.PullNumber)
	}
	return "AI review event"
}

func publicLiveFeedAIBody(log *GeminiWebhookLog) string {
	repo := sanitizeLedgerReferenceValue(log.Repository)
	if repo == "" {
		repo = "GitHub repository"
	}
	action := sanitizeLedgerReferenceValue(log.Action)
	if action == "" {
		action = sanitizeLedgerReferenceValue(log.EventName)
	}
	if action == "" {
		action = "review"
	}
	if log.PullNumber > 0 {
		return fmt.Sprintf("%s processed %s for %s PR #%d.", defaultGeminiReviewModel, action, repo, log.PullNumber)
	}
	return fmt.Sprintf("%s processed %s for %s.", defaultGeminiReviewModel, action, repo)
}

func publicLiveFeedAIReference(log *GeminiWebhookLog) string {
	repo := sanitizeLedgerReferenceValue(log.Repository)
	if repo == "" {
		return sanitizeLedgerReferenceValue(log.EventName)
	}
	if log.PullNumber > 0 {
		return fmt.Sprintf("%s#%d", repo, log.PullNumber)
	}
	return repo
}

func publicLiveFeedReferenceURL(reference string) string {
	fields := splitLedgerReference(reference)
	if pullURL := normalizeLedgerPullURL(fields["pr"]); pullURL != "" {
		return pullURL
	}
	return publicLiveFeedURL(reference)
}

func publicLiveFeedURL(value string) string {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(value, "https://") || strings.HasPrefix(value, "http://") {
		return value
	}
	return ""
}

func publicLiveFeedStatus(status string) string {
	status = sanitizeLedgerReferenceValue(status)
	if status == "" {
		return "received"
	}
	return strings.ToLower(status)
}
