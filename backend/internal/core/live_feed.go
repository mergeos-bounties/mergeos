package core

import (
	"crypto/sha256"
	"encoding/hex"
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
	activeContributors := map[string]bool{}
	activeAgents := map[string]bool{}
	response := PublicLiveFeedResponse{
		ProtocolVersion: "mergeos.live-feed.v1",
		Kind:            "live_feed",
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
		publicLiveFeedTrackAgent(activeAgents, task.RequiredWorkerKind, task.SuggestedAgentType, task.AgentType)
		project := s.projects[task.ProjectID]
		if taskIsReleased(task) {
			response.Stats.AcceptedTaskCount++
			if workerID := strings.ToLower(strings.TrimSpace(normalizeWorkerID(task.WorkerID))); workerID != "" {
				activeContributors[workerID] = true
			}
			if task.SubmittedAt != nil {
				touch(*task.SubmittedAt)
				response.Items = append(response.Items, publicTaskSubmittedLiveFeedItem(task, project))
			}
			if task.AcceptedAt != nil {
				touch(*task.AcceptedAt)
			}
			response.Items = append(response.Items, publicTaskAcceptedLiveFeedItem(task, project))
			continue
		}
		if taskHasWorker(task) {
			if workerID := strings.ToLower(strings.TrimSpace(normalizeWorkerID(task.WorkerID))); workerID != "" {
				activeContributors[workerID] = true
			}
			if task.SubmittedAt != nil {
				touch(*task.SubmittedAt)
				response.Items = append(response.Items, publicTaskSubmittedLiveFeedItem(task, project))
				continue
			}
			if task.AcceptedAt != nil {
				touch(*task.AcceptedAt)
			}
			response.Items = append(response.Items, publicTaskClaimedLiveFeedItem(task, project))
			continue
		}
		if taskIsOpenForClaim(task) {
			response.Stats.OpenTaskCount++
			touch(task.CreatedAt)
			response.Items = append(response.Items, publicTaskOpenLiveFeedItem(task, project))
		}
	}

	for _, entry := range s.ledger {
		touch(entry.CreatedAt)
		if entry.Type == "task_payment" || entry.Type == "manual_credit" {
			if account := strings.ToLower(strings.TrimSpace(entry.ToAccount)); account != "" {
				activeContributors[account] = true
			}
		}
		response.Items = append(response.Items, publicLedgerLiveFeedItem(entry, projectIDs, taskProjectIDs, s.projects))
	}

	proposalKeys := map[string]bool{}
	for _, note := range s.notifications {
		if note == nil || note.Channel != "proposal" {
			continue
		}
		project := s.projects[note.ProjectID]
		if project != nil && note.UserID == project.ClientUserID {
			continue
		}
		proposal := s.workerSubmittedProposalFromNotificationLocked(note)
		if strings.TrimSpace(proposal.WorkerID) == "" || strings.TrimSpace(proposal.TaskID) == "" {
			continue
		}
		key := strings.TrimSpace(proposal.TaskID) + "|" + strings.TrimSpace(proposal.WorkerID) + "|" + strings.TrimSpace(proposal.Status)
		if proposalKeys[key] {
			continue
		}
		proposalKeys[key] = true
		response.Stats.ProposalCount++
		touch(proposal.UpdatedAt)
		if workerID := strings.ToLower(strings.TrimSpace(normalizeWorkerID(proposal.WorkerID))); workerID != "" {
			activeContributors[workerID] = true
		}
		response.Items = append(response.Items, publicProposalLiveFeedItem(proposal))
	}

	for _, note := range s.notifications {
		if note == nil || note.Channel != "task_review" {
			continue
		}
		fields := splitLedgerReference(note.Status)
		if strings.TrimSpace(fields["task_review"]) != taskReviewChangesRequested || strings.TrimSpace(fields["task"]) == "" {
			continue
		}
		taskID, err := s.resolveTaskClaimIDLocked(fields["task"])
		if err != nil {
			continue
		}
		task := s.tasks[taskID]
		if task == nil {
			continue
		}
		project := s.projects[task.ProjectID]
		if project != nil && note.UserID != project.ClientUserID {
			continue
		}
		touch(note.CreatedAt)
		if workerID := strings.ToLower(strings.TrimSpace(normalizeWorkerID(task.WorkerID))); workerID != "" {
			activeContributors[workerID] = true
		}
		response.Items = append(response.Items, publicTaskChangesRequestedLiveFeedItem(note, task, project))
	}

	for _, log := range s.geminiWebhookLogs {
		touch(log.ReceivedAt)
		publicLiveFeedTrackAgentLog(activeAgents, log)
		response.Items = append(response.Items, publicAILiveFeedItem(log))
	}
	response.Stats.ActiveContributorCount = len(activeContributors)
	response.Stats.ActiveAgentCount = len(activeAgents)

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

func publicLiveFeedTrackAgent(activeAgents map[string]bool, kinds ...any) {
	for _, value := range kinds {
		switch typed := value.(type) {
		case WorkerKind:
			if typed == WorkerAgent || typed == WorkerHybrid {
				activeAgents[string(typed)] = true
			}
		case string:
			normalized := strings.ToLower(strings.TrimSpace(typed))
			if normalized != "" {
				activeAgents[normalized] = true
			}
		}
	}
}

func publicLiveFeedTrackAgentLog(activeAgents map[string]bool, log *GeminiWebhookLog) {
	if log == nil {
		return
	}
	if strings.EqualFold(log.EventName, "agent_action") {
		agent := strings.ToLower(strings.TrimSpace(strings.TrimPrefix(log.Sender, "agent:")))
		if agent == "" {
			agent = defaultAgentActionActor
		}
		activeAgents[agent] = true
		return
	}
	if strings.TrimSpace(log.Action) != "" {
		activeAgents[defaultGeminiReviewModel] = true
	}
}

func (s *Store) PublicEventProtocol(limit int) PublicEventProtocolResponse {
	feed := s.PublicLiveFeed(limit)
	events := make([]EventProtocolDocument, 0, len(feed.Items))
	for _, item := range feed.Items {
		events = append(events, publicLiveFeedProtocolEvent(item))
	}
	return PublicEventProtocolResponse{
		Stats:  feed.Stats,
		Events: events,
	}
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
	claimID := marketplaceBountyID(projectID, task.IssueNumber)
	return PublicLiveFeedItem{
		ID:               publicLiveFeedTaskID("task-open", task),
		Type:             "task_opened",
		Title:            fmt.Sprintf("Task #%d opened", task.IssueNumber),
		Body:             publicLiveFeedTaskBody(task),
		ProjectID:        projectID,
		ProjectTitle:     projectTitle,
		TaskID:           claimID,
		Actor:            publicLiveFeedWorkerKind(task.RequiredWorkerKind, task.SuggestedAgentType),
		AmountCents:      task.RewardCents,
		Reference:        publicTaskReference(task),
		EvidenceRequired: publicTaskEvidenceRequiredForTask(task),
		URL:              marketplacePublicRepoURL(task.IssueURL),
		Status:           string(task.Status),
		CreatedAt:        task.CreatedAt,
	}
}

func publicTaskAcceptedLiveFeedItem(task *Task, project *Project) PublicLiveFeedItem {
	projectID, projectTitle := publicLiveFeedProjectScope(task, project)
	claimID := marketplaceBountyID(projectID, task.IssueNumber)
	createdAt := task.CreatedAt
	if task.AcceptedAt != nil {
		createdAt = *task.AcceptedAt
	}
	return PublicLiveFeedItem{
		ID:               publicLiveFeedTaskID("task-accepted", task),
		Type:             "task_accepted",
		Title:            fmt.Sprintf("Task #%d accepted", task.IssueNumber),
		Body:             publicLiveFeedTaskBody(task),
		ProjectID:        projectID,
		ProjectTitle:     projectTitle,
		TaskID:           claimID,
		Actor:            publicLiveFeedActor(task.WorkerID, task.AgentType),
		AmountCents:      task.RewardCents,
		Reference:        publicTaskReference(task),
		EvidenceRequired: publicTaskEvidenceRequiredForTask(task),
		URL:              marketplacePublicRepoURL(task.IssueURL),
		Status:           string(task.Status),
		CreatedAt:        createdAt,
	}
}

func publicTaskClaimedLiveFeedItem(task *Task, project *Project) PublicLiveFeedItem {
	projectID, projectTitle := publicLiveFeedProjectScope(task, project)
	claimID := marketplaceBountyID(projectID, task.IssueNumber)
	createdAt := task.CreatedAt
	if task.AcceptedAt != nil {
		createdAt = *task.AcceptedAt
	}
	return PublicLiveFeedItem{
		ID:               publicLiveFeedTaskID("task-claimed", task),
		Type:             "task_claimed",
		Title:            fmt.Sprintf("Task #%d claimed", task.IssueNumber),
		Body:             publicLiveFeedTaskBody(task),
		ProjectID:        projectID,
		ProjectTitle:     projectTitle,
		TaskID:           claimID,
		Actor:            publicLiveFeedActor(task.WorkerID, task.AgentType),
		AmountCents:      task.RewardCents,
		Reference:        publicTaskReference(task),
		EvidenceRequired: publicTaskEvidenceRequiredForTask(task),
		URL:              marketplacePublicRepoURL(task.IssueURL),
		Status:           string(task.Status),
		CreatedAt:        createdAt,
	}
}

func publicTaskSubmittedLiveFeedItem(task *Task, project *Project) PublicLiveFeedItem {
	projectID, projectTitle := publicLiveFeedProjectScope(task, project)
	claimID := marketplaceBountyID(projectID, task.IssueNumber)
	createdAt := task.CreatedAt
	if task.SubmittedAt != nil {
		createdAt = *task.SubmittedAt
	}
	return PublicLiveFeedItem{
		ID:               publicLiveFeedTaskID("task-submitted", task),
		Type:             "task_submitted",
		Title:            fmt.Sprintf("Task #%d submitted", task.IssueNumber),
		Body:             publicTaskSubmittedBody(task),
		ProjectID:        projectID,
		ProjectTitle:     projectTitle,
		TaskID:           claimID,
		Actor:            publicLiveFeedActor(task.WorkerID, task.AgentType),
		AmountCents:      task.RewardCents,
		Reference:        publicTaskSubmissionReference(task),
		EvidenceRequired: publicTaskEvidenceRequiredForTask(task),
		Evidence:         publicTaskSubmissionEvidence(task),
		URL:              publicTaskSubmissionURL(task),
		Status:           submittedTaskStatus,
		CreatedAt:        createdAt,
	}
}

func publicTaskChangesRequestedLiveFeedItem(note *Notification, task *Task, project *Project) PublicLiveFeedItem {
	projectID, projectTitle := publicLiveFeedProjectScope(task, project)
	claimID := marketplaceBountyID(projectID, task.IssueNumber)
	body := compactText(note.Body)
	if body == "" {
		body = fmt.Sprintf("Task #%d needs changes before payout release.", task.IssueNumber)
	}
	actor := marketplaceClientDisplayName(project)
	if actor == "" {
		actor = projectTitle
	}
	createdAt := time.Now().UTC()
	itemID := publicLiveFeedTaskID("task-changes-requested", task)
	if note != nil && !note.CreatedAt.IsZero() {
		createdAt = note.CreatedAt
	}
	if note != nil && strings.TrimSpace(note.ID) != "" {
		itemID = "task-changes-requested:" + sanitizeLedgerReferenceValue(note.ID)
	}
	return PublicLiveFeedItem{
		ID:               itemID,
		Type:             "task_changes_requested",
		Title:            fmt.Sprintf("Task #%d changes requested", task.IssueNumber),
		Body:             body,
		ProjectID:        projectID,
		ProjectTitle:     projectTitle,
		TaskID:           claimID,
		Actor:            actor,
		Action:           taskReviewChangesRequested,
		AmountCents:      task.RewardCents,
		Reference:        taskReviewReference(taskReviewChangesRequested, claimID, task.WorkerID),
		EvidenceRequired: publicTaskEvidenceRequiredForTask(task),
		URL:              publicTaskSubmissionURL(task),
		Status:           taskReviewChangesRequested,
		CreatedAt:        createdAt,
	}
}

func publicLedgerLiveFeedItem(entry LedgerEntry, projectIDs map[string]bool, taskProjectIDs map[string]string, projects map[string]*Project) PublicLiveFeedItem {
	projectID, taskID := publicLedgerScope(entry, projectIDs, taskProjectIDs)
	projectTitle := ""
	publicTaskID := ""
	if projectID != "" {
		project := projects[projectID]
		projectTitle = publicLiveFeedProjectTitle(project)
		publicTaskID = publicLedgerTaskReferenceID(project, taskID)
	}
	reference := publicLedgerReference(projectID, publicTaskID, entry.Sequence, entry.Reference)
	return PublicLiveFeedItem{
		ID:             fmt.Sprintf("ledger:%d", entry.Sequence),
		Type:           "ledger_" + entry.Type,
		Title:          publicLiveFeedLedgerTitle(entry.Type),
		Body:           publicLiveFeedLedgerBody(entry, projectTitle),
		ProjectID:      projectID,
		ProjectTitle:   projectTitle,
		TaskID:         publicTaskID,
		Actor:          publicLedgerAccount(entry.ToAccount, projectID, publicTaskID),
		AmountCents:    entry.AmountCents,
		LedgerSequence: entry.Sequence,
		EntryHash:      entry.EntryHash,
		Reference:      reference,
		URL:            publicLiveFeedReferenceURL(reference),
		Status:         "verified",
		CreatedAt:      entry.CreatedAt,
	}
}

func publicLedgerTaskReferenceID(project *Project, taskID string) string {
	taskID = strings.TrimSpace(taskID)
	if project == nil || taskID == "" {
		return ""
	}
	for _, task := range project.Tasks {
		if task == nil || strings.TrimSpace(task.ID) != taskID {
			continue
		}
		return marketplaceBountyID(project.ID, task.IssueNumber)
	}
	return ""
}

func publicProposalLiveFeedItem(proposal WorkerSubmittedProposal) PublicLiveFeedItem {
	status := normalizeProposalStatus(proposal.Status)
	if status == "" {
		status = "submitted"
	}
	issueLabel := "task"
	if proposal.IssueNumber > 0 {
		issueLabel = fmt.Sprintf("issue #%d", proposal.IssueNumber)
	}
	projectTitle := strings.TrimSpace(proposal.ProjectTitle)
	if projectTitle == "" {
		projectTitle = "MergeOS project"
	}
	actor := publicLedgerAccount(proposal.WorkerID, proposal.ProjectID, proposal.TaskID)
	if actor == "" {
		actor = "worker:contributor"
	}
	title := "Worker proposal submitted"
	body := fmt.Sprintf("%s proposed %s for %s in %s.", actor, formatTokenAmount(proposal.BidCents), issueLabel, projectTitle)
	switch status {
	case "accepted":
		title = "Worker proposal accepted"
		body = fmt.Sprintf("%s was accepted for %s in %s.", actor, issueLabel, projectTitle)
	case "declined":
		title = "Worker proposal declined"
		body = fmt.Sprintf("%s was declined for %s in %s.", actor, issueLabel, projectTitle)
	case "reviewing":
		title = "Worker proposal under review"
	}
	createdAt := proposal.UpdatedAt
	if createdAt.IsZero() {
		createdAt = proposal.CreatedAt
	}
	return PublicLiveFeedItem{
		ID:           "proposal:" + proposal.ID,
		Type:         publicProposalLiveFeedType(status),
		Title:        title,
		Body:         body,
		ProjectID:    proposal.ProjectID,
		ProjectTitle: projectTitle,
		TaskID:       proposal.TaskID,
		Actor:        actor,
		Action:       status,
		AmountCents:  proposal.BidCents,
		Reference:    publicProposalReference(proposal),
		Status:       status,
		CreatedAt:    createdAt,
	}
}

func publicProposalReference(proposal WorkerSubmittedProposal) string {
	status := normalizeProposalStatus(proposal.Status)
	if status == "" {
		status = "submitted"
	}
	proposalID := strings.TrimSpace(proposal.ID)
	if proposalID == "" {
		proposalID = "unknown"
	}
	parts := []string{"proposal:" + proposalID}
	if projectID := strings.TrimSpace(proposal.ProjectID); projectID != "" {
		parts = append(parts, "project:"+projectID)
	}
	taskID := strings.TrimSpace(proposal.ClaimID)
	if taskID == "" {
		taskID = strings.TrimSpace(proposal.TaskID)
	}
	if taskID != "" {
		parts = append(parts, "task:"+taskID)
	}
	if proposal.IssueNumber > 0 {
		parts = append(parts, fmt.Sprintf("issue:%d", proposal.IssueNumber))
	}
	parts = append(parts, "status:"+status)
	return strings.Join(parts, ";")
}

func publicProposalLiveFeedType(status string) string {
	switch normalizeProposalStatus(status) {
	case "accepted":
		return "proposal_accepted"
	case "declined":
		return "proposal_declined"
	default:
		return "proposal_submitted"
	}
}

func publicAILiveFeedItem(log *GeminiWebhookLog) PublicLiveFeedItem {
	reference := publicLiveFeedAIReference(log)
	return PublicLiveFeedItem{
		ID:              "ai:" + log.ID,
		Type:            publicLiveFeedAIType(log),
		Title:           publicLiveFeedAITitle(log),
		Body:            publicLiveFeedAIBody(log),
		Actor:           publicLiveFeedAIActor(log),
		Action:          publicLiveFeedAIAction(log),
		Reference:       reference,
		ContextURLs:     publicAgentActionContextURLs(log),
		Evidence:        normalizeAgentActionTextList(log.Evidence, 12, 220),
		Runbook:         normalizeAgentActionTextList(log.Runbook, 12, 220),
		Checks:          normalizeAgentActionChecks(log.Checks),
		DelegatedBy:     log.DelegatedBy,
		DesignAgent:     log.DesignAgent,
		SubagentType:    log.SubagentType,
		DelegationChain: normalizeAgentDelegationChain(log.DelegationChain, log.DelegatedBy, log.DesignAgent, log.SubagentType),
		URL:             publicLiveFeedURL(log.CommentURL),
		Status:          publicLiveFeedStatus(log.Status),
		CreatedAt:       log.ReceivedAt,
	}
}

func publicAgentActionContextURLs(log *GeminiWebhookLog) []string {
	if log == nil {
		return []string{}
	}
	values := make([]string, 0, len(log.ContextURLs)+1)
	values = append(values, log.ContextURLs...)
	if log.CommentURL != "" {
		values = append(values, log.CommentURL)
	}
	return normalizeAgentActionURLs(values)
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

func publicTaskSubmissionReference(task *Task) string {
	if url := publicTaskSubmissionURL(task); url != "" {
		return url
	}
	return publicTaskReference(task)
}

func publicTaskSubmissionURL(task *Task) string {
	if task == nil {
		return ""
	}
	if url := publicLiveFeedURL(task.PullRequestURL); url != "" {
		return url
	}
	return publicLiveFeedURL(task.ReviewEvidenceURL)
}

func publicTaskSubmittedBody(task *Task) string {
	body := publicLiveFeedTaskBody(task)
	if task == nil {
		return body
	}
	switch {
	case task.PullRequestURL != "" && task.ReviewEvidenceURL != "":
		return body + " - PR and review evidence submitted."
	case task.PullRequestURL != "":
		return body + " - Pull request submitted for review."
	case task.ReviewEvidenceURL != "":
		return body + " - Evidence URL submitted for review."
	default:
		return body + " - Review notes submitted."
	}
}

func publicTaskSubmissionEvidence(task *Task) []string {
	if task == nil {
		return []string{}
	}
	evidence := []string{}
	if task.PullRequestURL != "" {
		evidence = append(evidence, "pull_request:"+task.PullRequestURL)
	}
	if task.ReviewEvidenceURL != "" {
		evidence = append(evidence, "evidence:"+task.ReviewEvidenceURL)
	}
	if task.ReviewNotes != "" {
		evidence = append(evidence, "notes:"+protocolText(task.ReviewNotes, 180, ""))
	}
	return normalizeAgentActionTextList(evidence, 6, 220)
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

func publicLiveFeedAIType(log *GeminiWebhookLog) string {
	if log != nil && strings.EqualFold(log.EventName, "repo_issues_synced") {
		return "repo_issues_synced"
	}
	if log != nil && strings.EqualFold(log.EventName, "agent_action") {
		return "agent_action"
	}
	if publicLiveFeedIsPullRequestOpened(log) {
		return "pr_opened"
	}
	return "ai_review"
}

func publicLiveFeedIsPullRequestOpened(log *GeminiWebhookLog) bool {
	return log != nil &&
		strings.EqualFold(log.EventName, "pull_request") &&
		strings.EqualFold(strings.TrimSpace(log.Action), "opened")
}

func publicLiveFeedAIAction(log *GeminiWebhookLog) string {
	if log == nil {
		return ""
	}
	action := sanitizeLedgerReferenceValue(log.Action)
	if action == "" {
		action = sanitizeLedgerReferenceValue(log.EventName)
	}
	return strings.ToLower(strings.TrimSpace(action))
}

func publicLiveFeedAIActor(log *GeminiWebhookLog) string {
	if log == nil {
		return ""
	}
	if strings.EqualFold(log.EventName, "repo_issues_synced") {
		return "mergeos-repo-sync"
	}
	if log != nil && strings.EqualFold(log.EventName, "agent_action") {
		actor := sanitizeLedgerReferenceValue(strings.TrimPrefix(log.Sender, "agent:"))
		if actor == "" {
			actor = defaultAgentActionActor
		}
		return marketplaceTitle(actor)
	}
	return publicLiveFeedGitHubActor(log.Sender)
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
	case "airdrop_claim":
		return "Airdrop claim recorded"
	case "presale_reservation":
		return "Presale reservation recorded"
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
	if strings.EqualFold(log.EventName, "repo_issues_synced") {
		return "Repository issues synced"
	}
	if strings.EqualFold(log.EventName, "agent_action") {
		action := sanitizeLedgerReferenceValue(log.Action)
		if action == "" {
			action = "action"
		}
		if log.PullNumber > 0 {
			return fmt.Sprintf("AI agent %s PR #%d", agentActionTitleVerb(action), log.PullNumber)
		}
		return fmt.Sprintf("AI agent %s", agentActionTitleVerb(action))
	}
	if publicLiveFeedIsPullRequestOpened(log) {
		if log.PullNumber > 0 {
			return fmt.Sprintf("PR #%d opened", log.PullNumber)
		}
		return "Pull request opened"
	}
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
	if strings.EqualFold(log.EventName, "repo_issues_synced") {
		imported := webhookLogLabelInt(log, "imported")
		added := webhookLogLabelInt(log, "added")
		updated := webhookLogLabelInt(log, "updated")
		return fmt.Sprintf("MergeOS synced %d issues for %s and routed %d new tasks plus %d updates.", imported, repo, added, updated)
	}
	if strings.EqualFold(log.EventName, "agent_action") {
		agent := sanitizeLedgerReferenceValue(strings.TrimPrefix(log.Sender, "agent:"))
		if agent == "" {
			agent = defaultAgentActionActor
		}
		action := sanitizeLedgerReferenceValue(log.Action)
		if action == "" {
			action = "action"
		}
		if log.PullNumber > 0 {
			return fmt.Sprintf("%s ran %s for %s PR #%d.", marketplaceTitle(agent), action, repo, log.PullNumber)
		}
		return fmt.Sprintf("%s ran %s for %s.", marketplaceTitle(agent), action, repo)
	}
	if publicLiveFeedIsPullRequestOpened(log) {
		actor := publicLiveFeedGitHubActor(log.Sender)
		if actor == "" {
			actor = "GitHub"
		}
		if log.PullNumber > 0 {
			return fmt.Sprintf("%s opened PR #%d in %s for MergeOS review.", actor, log.PullNumber, repo)
		}
		return fmt.Sprintf("%s opened a pull request in %s for MergeOS review.", actor, repo)
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

func agentActionTitleVerb(action string) string {
	switch strings.ToLower(strings.TrimSpace(action)) {
	case "review":
		return "reviewed"
	case "test":
		return "tested"
	case "generate":
		return "generated"
	case "deploy":
		return "validated deployment"
	case "scan":
		return "scanned"
	default:
		return "recorded action"
	}
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

func publicLiveFeedProtocolEvent(item PublicLiveFeedItem) EventProtocolDocument {
	actor := strings.TrimSpace(item.Actor)
	if actor == "" {
		actor = "mergeos"
	}
	occurredAt := item.CreatedAt
	if occurredAt.IsZero() {
		occurredAt = time.Unix(0, 0).UTC()
	}
	reference := publicEventReference(item)
	payload := map[string]any{
		"feed_type": item.Type,
		"title":     item.Title,
		"status":    item.Status,
	}
	if item.ProjectTitle != "" {
		payload["project_title"] = item.ProjectTitle
	}
	if item.URL != "" {
		payload["url"] = item.URL
	}
	if item.LedgerSequence > 0 {
		payload["ledger_sequence"] = item.LedgerSequence
	}
	if item.EntryHash != "" {
		payload["entry_hash"] = item.EntryHash
	}
	if len(item.EvidenceRequired) > 0 {
		payload["evidence_required"] = stableStrings(item.EvidenceRequired)
	}
	if len(item.ContextURLs) > 0 {
		payload["context_urls"] = normalizeAgentActionURLs(item.ContextURLs)
	}
	if len(item.Evidence) > 0 {
		payload["evidence"] = normalizeAgentActionTextList(item.Evidence, 12, 220)
	}
	if len(item.Runbook) > 0 {
		payload["runbook"] = normalizeAgentActionTextList(item.Runbook, 12, 220)
	}
	if len(item.Checks) > 0 {
		payload["checks"] = normalizeAgentActionChecks(item.Checks)
	}
	if item.DelegatedBy != "" {
		payload["delegated_by"] = item.DelegatedBy
	}
	if item.DesignAgent != "" {
		payload["design_agent"] = item.DesignAgent
	}
	if item.SubagentType != "" {
		payload["subagent_type"] = item.SubagentType
	}
	if len(item.DelegationChain) > 0 {
		payload["delegation_chain"] = item.DelegationChain
	}

	event := EventProtocolDocument{
		ProtocolVersion: "mergeos.event.v1",
		Kind:            "event",
		ID:              publicEventID(item.ID),
		Type:            publicEventType(item),
		OccurredAt:      occurredAt,
		Actor:           actor,
		ProjectID:       strings.TrimSpace(item.ProjectID),
		TaskID:          strings.TrimSpace(item.TaskID),
		Reference:       reference,
		Payload:         payload,
	}
	if item.Action != "" {
		payload["action"] = item.Action
	}
	if item.AmountCents > 0 {
		amount := float64(item.AmountCents) / 100
		event.AmountMRG = &amount
	}
	return event
}

func publicEventType(item PublicLiveFeedItem) string {
	feedType := item.Type
	switch feedType {
	case "project_funded":
		return "project.funded"
	case "task_opened":
		return "task.created"
	case "task_claimed":
		return "task.claimed"
	case "task_submitted":
		return "task.submitted"
	case "task_changes_requested":
		return "task.changes_requested"
	case "task_accepted":
		return "task.accepted"
	case "deployment_validation":
		return "deployment.updated"
	case "pr_opened":
		return "pr.opened"
	case "ai_review":
		return "pr.reviewed"
	case "repo_issues_synced":
		return "repo.issues.synced"
	case "proposal_submitted":
		return "proposal.submitted"
	case "proposal_accepted":
		return "proposal.accepted"
	case "proposal_declined":
		return "proposal.declined"
	case "agent_action":
		return publicAgentActionEventType(item.Action)
	case "ledger_airdrop_claim":
		return "airdrop.claimed"
	case "ledger_presale_reservation":
		return "presale.reserved"
	}
	if strings.HasPrefix(feedType, "ledger_task_payment") {
		return "task.paid"
	}
	if strings.HasPrefix(feedType, "ledger_") {
		return "ledger.recorded"
	}
	return "agent.action"
}

func publicAgentActionEventType(action string) string {
	switch strings.ToLower(strings.TrimSpace(action)) {
	case "review":
		return "agent.reviewed"
	case "test":
		return "agent.tested"
	case "generate":
		return "agent.generated"
	case "deploy":
		return "agent.deployed"
	case "scan":
		return "agent.scanned"
	default:
		return "agent.action"
	}
}

func publicEventID(feedID string) string {
	id := "evt:" + strings.TrimSpace(feedID)
	if len(id) >= 3 && len(id) <= 120 {
		return id
	}
	sum := sha256.Sum256([]byte(id))
	return "evt:" + hex.EncodeToString(sum[:16])
}

func publicEventReference(item PublicLiveFeedItem) string {
	reference := strings.TrimSpace(item.Reference)
	if reference == "" {
		reference = strings.TrimSpace(item.URL)
	}
	if len(reference) <= 512 {
		return reference
	}
	return reference[:512]
}
