package core

import (
	"errors"
	"strings"
)

func (s *Store) ProjectEscrow(projectID string) (ProjectEscrowResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	project, ok := s.projects[strings.TrimSpace(projectID)]
	if !ok {
		return ProjectEscrowResponse{}, errors.New("project not found")
	}
	return s.projectEscrowLocked(project), nil
}

func (s *Store) projectEscrowLocked(project *Project) ProjectEscrowResponse {
	tasks := s.projectDeploymentTasksLocked(project)
	taskIDs := map[string]bool{}
	taskPaid := map[string]int64{}
	for _, task := range tasks {
		if task != nil {
			taskIDs[task.ID] = true
		}
	}

	projectReserveCents := int64(0)
	taskReserveCents := int64(0)
	taskPaymentCents := int64(0)
	manualCreditCents := int64(0)
	updatedAt := project.CreatedAt

	for _, entry := range s.ledger {
		if !projectEscrowLedgerApplies(project, taskIDs, entry) {
			continue
		}
		if entry.CreatedAt.After(updatedAt) {
			updatedAt = entry.CreatedAt
		}
		switch entry.Type {
		case "project_reserve":
			projectReserveCents += entry.AmountCents
		case "task_reserve":
			taskReserveCents += entry.AmountCents
		case "task_payment":
			taskID := ledgerReferenceTaskID(entry.Reference)
			taskPaymentCents += entry.AmountCents
			taskPaid[taskID] += entry.AmountCents
		case "manual_credit":
			taskID := ledgerReferenceTaskID(entry.Reference)
			manualCreditCents += entry.AmountCents
			taskPaid[taskID] += entry.AmountCents
		}
	}

	if projectReserveCents == 0 {
		projectReserveCents = project.WorkPoolCents
	}
	if taskReserveCents == 0 {
		for _, task := range tasks {
			if task != nil {
				taskReserveCents += task.RewardCents
			}
		}
	}

	taskRows := make([]ProjectEscrowTask, 0, len(tasks))
	paidTaskCount := 0
	openTaskCount := 0
	for _, task := range tasks {
		if task == nil {
			continue
		}
		if taskUpdated := deploymentTaskUpdatedAt(task); taskUpdated.After(updatedAt) {
			updatedAt = taskUpdated
		}
		paid := taskPaid[task.ID]
		if taskIsReleased(task) || paid > 0 {
			paidTaskCount++
		} else if taskIsOpenForClaim(task) {
			openTaskCount++
		}
		taskRows = append(taskRows, projectEscrowTaskRow(task, paid))
	}

	releasedCents := taskPaymentCents + manualCreditCents
	remainingCents := project.WorkPoolCents - releasedCents
	overdrawnCents := int64(0)
	if remainingCents < 0 {
		overdrawnCents = -remainingCents
		remainingCents = 0
	}
	unallocatedCents := project.WorkPoolCents - taskReserveCents
	if unallocatedCents < 0 {
		unallocatedCents = 0
	}

	return ProjectEscrowResponse{
		ProtocolVersion:     "mergeos.escrow.v1",
		Kind:                "escrow",
		ProjectID:           project.ID,
		ProjectTitle:        publicLiveFeedProjectTitle(project),
		TokenSymbol:         normalizedTokenSymbol(s.cfg.TokenSymbol),
		ReleaseStatus:       projectEscrowReleaseStatus(project.WorkPoolCents, releasedCents),
		BudgetCents:         project.BudgetCents,
		FeeCents:            project.FeeCents,
		WorkPoolCents:       project.WorkPoolCents,
		ProjectReserveCents: projectReserveCents,
		TaskReserveCents:    taskReserveCents,
		TaskPaymentCents:    taskPaymentCents,
		ManualCreditCents:   manualCreditCents,
		ReleasedCents:       releasedCents,
		RemainingCents:      remainingCents,
		OverdrawnCents:      overdrawnCents,
		UnallocatedCents:    unallocatedCents,
		PaidTaskCount:       paidTaskCount,
		OpenTaskCount:       openTaskCount,
		UpdatedAt:           updatedAt,
		Tasks:               taskRows,
	}
}

func projectEscrowLedgerApplies(project *Project, taskIDs map[string]bool, entry LedgerEntry) bool {
	if project == nil {
		return false
	}
	if entry.Type == "task_payment" || entry.Type == "manual_credit" {
		return taskIDs[ledgerReferenceTaskID(entry.Reference)]
	}
	projectID := strings.TrimSpace(project.ID)
	if projectID == "" {
		return false
	}
	haystack := strings.Join([]string{entry.FromAccount, entry.ToAccount, entry.Reference}, "|")
	return strings.Contains(haystack, projectID)
}

func projectEscrowTaskRow(task *Task, paidCents int64) ProjectEscrowTask {
	remainingCents := task.RewardCents - paidCents
	overpaidCents := int64(0)
	if remainingCents < 0 {
		overpaidCents = -remainingCents
		remainingCents = 0
	}
	return ProjectEscrowTask{
		TaskID:         task.ID,
		IssueNumber:    task.IssueNumber,
		Title:          task.Title,
		Status:         string(task.Status),
		ReleaseStatus:  projectEscrowTaskReleaseStatus(task.RewardCents, paidCents),
		RewardCents:    task.RewardCents,
		PaidCents:      paidCents,
		RemainingCents: remainingCents,
		OverpaidCents:  overpaidCents,
		WorkerID:       task.WorkerID,
		ProofHash:      task.ProofHash,
		IssueURL:       marketplacePublicRepoURL(task.IssueURL),
		UpdatedAt:      deploymentTaskUpdatedAt(task),
	}
}

func projectEscrowReleaseStatus(workPoolCents, releasedCents int64) string {
	switch {
	case workPoolCents > 0 && releasedCents > workPoolCents:
		return "overdrawn"
	case workPoolCents > 0 && releasedCents >= workPoolCents:
		return "released"
	case releasedCents > 0:
		return "releasing"
	default:
		return "funded"
	}
}

func projectEscrowTaskReleaseStatus(rewardCents, paidCents int64) string {
	switch {
	case rewardCents > 0 && paidCents > rewardCents:
		return "overpaid"
	case rewardCents > 0 && paidCents >= rewardCents:
		return "released"
	case paidCents > 0:
		return "partial"
	default:
		return "reserved"
	}
}
