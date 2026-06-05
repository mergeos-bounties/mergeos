package core

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

func (s *Store) ProjectPayouts(projectID string) (ProjectPayoutsResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	project, ok := s.projects[strings.TrimSpace(projectID)]
	if !ok {
		return ProjectPayoutsResponse{}, errors.New("project not found")
	}
	return s.projectPayoutsLocked(project), nil
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
