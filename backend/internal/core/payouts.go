package core

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

const defaultAutoReleasePolicy = "mergeos.auto_release.low_risk_pr.v1"

func (s *Store) ProjectPayouts(projectID string) (ProjectPayoutsResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	project, ok := s.projects[strings.TrimSpace(projectID)]
	if !ok {
		return ProjectPayoutsResponse{}, errors.New("project not found")
	}
	return s.projectPayoutsLocked(project), nil
}

func (s *Store) AutoReleaseProjectPayouts(projectID string, req ProjectAutoReleaseRequest) (ProjectAutoReleaseResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	project, ok := s.projects[strings.TrimSpace(projectID)]
	if !ok {
		return ProjectAutoReleaseResponse{}, errors.New("project not found")
	}

	policy := normalizeAutoReleasePolicy(req.Policy)
	candidates := autoReleaseCandidateMap(project, req.Candidates)
	taskIDs := autoReleaseTaskIDs(project, req.TaskIDs, req.Candidates)
	response := ProjectAutoReleaseResponse{
		ProtocolVersion: "mergeos.payout-release.v1",
		Kind:            "auto_release",
		ProjectID:       project.ID,
		Policy:          policy,
		Released:        []TaskClaimResponse{},
		Skipped:         []ProjectAutoReleaseSkip{},
	}

	for _, requestedTaskID := range taskIDs {
		taskID, err := resolveProjectTaskReference(project, requestedTaskID)
		if err != nil {
			response.Skipped = append(response.Skipped, ProjectAutoReleaseSkip{TaskID: requestedTaskID, Reason: err.Error()})
			continue
		}
		task := s.tasks[taskID]
		if task == nil || task.ProjectID != project.ID {
			response.Skipped = append(response.Skipped, ProjectAutoReleaseSkip{TaskID: requestedTaskID, Reason: "task not found"})
			continue
		}
		candidate := candidates[taskID]
		if strings.TrimSpace(candidate.TaskID) == "" {
			response.Skipped = append(response.Skipped, ProjectAutoReleaseSkip{TaskID: requestedTaskID, Reason: "auto-release candidate is required"})
			continue
		}
		acceptReq, reference, err := autoReleaseAcceptRequest(task, candidate, policy)
		if err != nil {
			response.Skipped = append(response.Skipped, ProjectAutoReleaseSkip{TaskID: requestedTaskID, Reason: err.Error()})
			continue
		}
		released, _, err := s.acceptTaskWithReviewReferenceLocked(task.ID, acceptReq, 0, "", reference)
		if err != nil {
			response.Skipped = append(response.Skipped, ProjectAutoReleaseSkip{TaskID: requestedTaskID, Reason: err.Error()})
			continue
		}
		response.Released = append(response.Released, taskClaimProtocolDocument(marketplaceBountyID(released.ProjectID, released.IssueNumber), released))
	}

	response.ReleasedCount = len(response.Released)
	response.SkippedCount = len(response.Skipped)
	if response.ReleasedCount > 0 {
		if err := s.saveLocked(); err != nil {
			return ProjectAutoReleaseResponse{}, err
		}
	}
	response.Payouts = s.projectPayoutsLocked(project)
	return response, nil
}

func (s *Store) projectPayoutsLocked(project *Project) ProjectPayoutsResponse {
	escrow := s.projectEscrowLocked(project)
	tasks := s.projectDeploymentTasksLocked(project)
	taskIDs := map[string]bool{}
	for _, task := range tasks {
		if task != nil {
			taskIDs[task.ID] = true
		}
	}

	paidByTask := map[string]int64{}
	entryCountByTask := map[string]int{}
	lastEntryByTask := map[string]LedgerEntry{}
	for _, entry := range s.ledger {
		if entry.Type != "task_payment" && entry.Type != "manual_credit" {
			continue
		}
		taskID := ledgerReferenceTaskID(entry.Reference)
		if !taskIDs[taskID] {
			continue
		}
		paidByTask[taskID] += entry.AmountCents
		entryCountByTask[taskID]++
		lastEntry, ok := lastEntryByTask[taskID]
		if !ok || entry.CreatedAt.After(lastEntry.CreatedAt) || (entry.CreatedAt.Equal(lastEntry.CreatedAt) && entry.Sequence > lastEntry.Sequence) {
			lastEntryByTask[taskID] = entry
		}
	}

	rows := make([]ProjectPayoutRow, 0, len(tasks))
	for _, task := range tasks {
		if task == nil {
			continue
		}
		lastEntry, hasEntry := lastEntryByTask[task.ID]
		row := s.projectPayoutRowLocked(task, paidByTask[task.ID], entryCountByTask[task.ID], lastEntry, hasEntry)
		rows = append(rows, row)
	}
	sort.Slice(rows, func(i, j int) bool {
		left := rows[i].ReleasedAt
		right := rows[j].ReleasedAt
		if left != nil && right != nil && !left.Equal(*right) {
			return left.After(*right)
		}
		if left != nil && right == nil {
			return true
		}
		if left == nil && right != nil {
			return false
		}
		return rows[i].IssueNumber < rows[j].IssueNumber
	})

	return ProjectPayoutsResponse{
		ProtocolVersion: "mergeos.payouts.v1",
		Kind:            "payouts",
		ProjectID:       escrow.ProjectID,
		ProjectTitle:    escrow.ProjectTitle,
		TokenSymbol:     escrow.TokenSymbol,
		ReleaseStatus:   escrow.ReleaseStatus,
		WorkPoolCents:   escrow.WorkPoolCents,
		ReleasedCents:   escrow.ReleasedCents,
		RemainingCents:  escrow.RemainingCents,
		OverdrawnCents:  escrow.OverdrawnCents,
		TaskCount:       len(rows),
		PaidTaskCount:   escrow.PaidTaskCount,
		OpenTaskCount:   escrow.OpenTaskCount,
		ReleaseCount:    projectPayoutReleaseCount(entryCountByTask),
		UpdatedAt:       escrow.UpdatedAt,
		Payouts:         rows,
	}
}

func (s *Store) projectPayoutRowLocked(task *Task, paidCents int64, ledgerEntryCount int, lastEntry LedgerEntry, hasEntry bool) ProjectPayoutRow {
	remainingCents := task.RewardCents - paidCents
	overpaidCents := int64(0)
	if remainingCents < 0 {
		overpaidCents = -remainingCents
		remainingCents = 0
	}

	rowType := "reserved"
	payoutAccount := ""
	ledgerSequence := 0
	entryHash := ""
	reference := projectPayoutTaskReference(task)
	url := marketplacePublicRepoURL(task.IssueURL)
	var releasedAt *time.Time
	if hasEntry {
		rowType = lastEntry.Type
		payoutAccount = lastEntry.ToAccount
		ledgerSequence = lastEntry.Sequence
		entryHash = lastEntry.EntryHash
		reference = projectPayoutLedgerReference(lastEntry.Reference)
		url = publicLiveFeedReferenceURL(lastEntry.Reference)
		released := lastEntry.CreatedAt
		releasedAt = &released
	} else if strings.TrimSpace(task.WorkerID) != "" {
		payoutAccount = s.payoutAccountForWorkerLocked(task.WorkerID)
	}
	if url == "" {
		url = marketplacePublicRepoURL(task.IssueURL)
	}

	return ProjectPayoutRow{
		TaskID:           task.ID,
		IssueNumber:      task.IssueNumber,
		Title:            task.Title,
		Type:             rowType,
		Status:           string(task.Status),
		ReleaseStatus:    projectEscrowTaskReleaseStatus(task.RewardCents, paidCents),
		WorkerID:         task.WorkerID,
		PayoutAccount:    payoutAccount,
		RewardCents:      task.RewardCents,
		PaidCents:        paidCents,
		RemainingCents:   remainingCents,
		OverpaidCents:    overpaidCents,
		LedgerSequence:   ledgerSequence,
		LedgerEntryCount: ledgerEntryCount,
		EntryHash:        entryHash,
		ProofHash:        task.ProofHash,
		Reference:        reference,
		URL:              url,
		ReleasedAt:       releasedAt,
		UpdatedAt:        deploymentTaskUpdatedAt(task),
	}
}

func projectPayoutReleaseCount(counts map[string]int) int {
	total := 0
	for _, count := range counts {
		total += count
	}
	return total
}

func projectPayoutLedgerReference(reference string) string {
	if pullReference := publicPullLedgerReference(reference); pullReference != "" {
		if policy := sanitizeLedgerReferenceValue(splitLedgerReference(reference)["auto_release"]); policy != "" {
			return pullReference + ";auto_release:" + policy
		}
		return pullReference
	}
	return sanitizeLedgerReferenceValue(reference)
}

func projectPayoutTaskReference(task *Task) string {
	if task == nil {
		return ""
	}
	if url := marketplacePublicRepoURL(task.IssueURL); url != "" {
		return url
	}
	if task.IssueNumber > 0 {
		return fmt.Sprintf("issue:%d", task.IssueNumber)
	}
	return "task:" + sanitizeLedgerReferenceValue(task.ID)
}

func autoReleaseCandidateMap(project *Project, candidates []ProjectAutoReleaseCandidate) map[string]ProjectAutoReleaseCandidate {
	rows := map[string]ProjectAutoReleaseCandidate{}
	for _, candidate := range candidates {
		taskID, err := resolveProjectTaskReference(project, candidate.TaskID)
		if err != nil {
			continue
		}
		candidate.TaskID = taskID
		rows[taskID] = candidate
	}
	return rows
}

func autoReleaseTaskIDs(project *Project, requested []string, candidates []ProjectAutoReleaseCandidate) []string {
	seen := map[string]bool{}
	rows := []string{}
	add := func(value string) {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			return
		}
		seen[value] = true
		rows = append(rows, value)
	}
	for _, taskID := range requested {
		add(taskID)
	}
	if len(rows) == 0 {
		for _, candidate := range candidates {
			if taskID, err := resolveProjectTaskReference(project, candidate.TaskID); err == nil {
				add(taskID)
			} else {
				add(candidate.TaskID)
			}
		}
	}
	return rows
}

func resolveProjectTaskReference(project *Project, value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", errors.New("task id is required")
	}
	if project == nil {
		return "", errors.New("project not found")
	}
	for _, task := range project.Tasks {
		if task == nil {
			continue
		}
		if task.ID == value || marketplaceBountyID(project.ID, task.IssueNumber) == value {
			return task.ID, nil
		}
	}
	return "", errors.New("task not found")
}

func autoReleaseAcceptRequest(task *Task, candidate ProjectAutoReleaseCandidate, policy string) (AcceptTaskRequest, string, error) {
	if task == nil {
		return AcceptTaskRequest{}, "", errors.New("task not found")
	}
	if !candidate.CanRelease {
		return AcceptTaskRequest{}, "", errors.New("candidate is not marked release-ready")
	}
	if !strings.EqualFold(strings.TrimSpace(candidate.ReadinessStatus), "ready") {
		return AcceptTaskRequest{}, "", errors.New("pull request readiness must be ready")
	}
	if !candidate.CanMerge {
		return AcceptTaskRequest{}, "", errors.New("pull request must be merge-ready")
	}
	if !strings.EqualFold(strings.TrimSpace(candidate.RiskLevel), "low") {
		return AcceptTaskRequest{}, "", errors.New("pull request risk must be low")
	}
	if candidate.Draft {
		return AcceptTaskRequest{}, "", errors.New("draft pull requests cannot auto-release")
	}
	if task.RewardCents <= 0 {
		return AcceptTaskRequest{}, "", errors.New("funded reward is required")
	}
	if candidate.RewardCents != task.RewardCents {
		return AcceptTaskRequest{}, "", errors.New("candidate reward does not match task reward")
	}
	workerID := normalizeWorkerID(candidate.WorkerID)
	if workerID == "" {
		return AcceptTaskRequest{}, "", errors.New("worker id is required")
	}
	pullURL := normalizeLedgerPullURL(candidate.PullRequestURL)
	if pullURL == "" {
		return AcceptTaskRequest{}, "", errors.New("pull request evidence is required")
	}
	if candidate.PullRequestNumber <= 0 {
		return AcceptTaskRequest{}, "", errors.New("pull request number is required")
	}
	if pullNumberFromLedgerURL(pullURL) != candidate.PullRequestNumber {
		return AcceptTaskRequest{}, "", errors.New("pull request number does not match evidence URL")
	}
	workerKind := candidate.WorkerKind
	if workerKind == "" || workerKind != task.RequiredWorkerKind {
		return AcceptTaskRequest{}, "", fmt.Errorf("task requires %s work", task.RequiredWorkerKind)
	}
	agentType := strings.TrimSpace(candidate.AgentType)
	if workerKind != WorkerHuman && agentType == "" {
		agentType = strings.TrimSpace(task.SuggestedAgentType)
		if agentType == "" {
			agentType = "auto-release"
		}
	}
	req := AcceptTaskRequest{
		WorkerKind: workerKind,
		WorkerID:   workerID,
		AgentType:  agentType,
	}
	reference := buildPullLedgerReference(task.ID, pullURL, candidate.PullRequestTitle)
	reference = strings.TrimSpace(reference + ";auto_release:" + sanitizeLedgerReferenceValue(policy))
	return req, ensureTaskLedgerReference(task.ID, reference), nil
}

func pullNumberFromLedgerURL(value string) int {
	parts := strings.Split(strings.Trim(normalizeLedgerPullURL(value), "/"), "/")
	if len(parts) == 0 {
		return 0
	}
	number, err := strconv.Atoi(parts[len(parts)-1])
	if err != nil {
		return 0
	}
	return number
}

func normalizeAutoReleasePolicy(value string) string {
	value = sanitizeLedgerReferenceValue(value)
	if value == "" {
		return defaultAutoReleasePolicy
	}
	return value
}
