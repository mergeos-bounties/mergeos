package core

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

func (s *Store) CreateProposal(userID string, req CreateProposalRequest) (CreateProposalResponse, error) {
	userID = strings.TrimSpace(userID)
	cover := proposalText(req.CoverLetter, 2000)
	if cover == "" {
		return CreateProposalResponse{}, errors.New("cover letter is required")
	}
	if req.EstimatedHours < 0 {
		return CreateProposalResponse{}, errors.New("estimated_hours must be positive")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	user := s.users[userID]
	if user == nil {
		return CreateProposalResponse{}, errors.New("login is required")
	}
	workerID := proposalWorkerID(user)
	if workerID == "" {
		return CreateProposalResponse{}, errors.New("GitHub or wallet identity is required to send proposals")
	}
	taskID, err := s.resolveTaskClaimIDLocked(req.TaskID)
	if err != nil {
		return CreateProposalResponse{}, err
	}
	task := s.tasks[taskID]
	if task == nil {
		return CreateProposalResponse{}, errors.New("task not found")
	}
	if task.Status == TaskAccepted {
		return CreateProposalResponse{}, errors.New("task is already accepted")
	}
	if task.RequiredWorkerKind == WorkerAgent {
		return CreateProposalResponse{}, errors.New("agent-only tasks cannot receive worker proposals")
	}
	project := s.projects[task.ProjectID]
	if project == nil {
		return CreateProposalResponse{}, errors.New("project not found")
	}
	if s.hasOpenProposalLocked(user.ID, task.ID) {
		return CreateProposalResponse{}, errors.New("proposal already submitted for this task")
	}

	bidCents := req.BidCents
	if bidCents <= 0 {
		bidCents = task.RewardCents
	}
	if bidCents <= 0 {
		return CreateProposalResponse{}, errors.New("bid_cents must be positive")
	}
	estimatedHours := req.EstimatedHours
	if estimatedHours == 0 {
		estimatedHours = marketplaceEstimatedHours(task)
	}
	availability := proposalText(req.Availability, 160)
	if availability == "" {
		availability = "Available after customer approval"
	}

	reference := proposalReference(task.ID, workerID, bidCents, estimatedHours, availability)
	subject := fmt.Sprintf("Proposal for issue #%d: %s", task.IssueNumber, sanitizeLedgerReferenceValue(task.Title))
	workerNote := s.addNotificationLocked(user.ID, project.ID, "proposal", subject, cover, reference)
	customerBody := fmt.Sprintf("%s proposed %s for issue #%d. %s", publicLedgerAccount(workerID, "", ""), formatTokenAmount(bidCents), task.IssueNumber, cover)
	customerNote := s.addNotificationLocked(project.ClientUserID, project.ID, "proposal", subject, proposalText(customerBody, 2000), reference)
	if err := s.saveLocked(); err != nil {
		return CreateProposalResponse{}, err
	}

	proposal := s.workerSubmittedProposalFromNotificationLocked(workerNote)
	return CreateProposalResponse{
		ProtocolVersion:      "mergeos.proposal.v1",
		Kind:                 "proposal",
		Proposal:             proposal,
		WorkerNotification:   publicProposalNotification(workerNote, proposal.Reference),
		CustomerNotification: publicProposalNotification(customerNote, proposal.Reference),
	}, nil
}

func (s *Store) resolveTaskClaimIDLocked(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", errors.New("task id is required")
	}
	if task, ok := s.tasks[value]; ok && task != nil {
		return task.ID, nil
	}
	separator := strings.LastIndex(value, ":")
	if separator <= 0 || separator >= len(value)-1 {
		return "", errors.New("task not found")
	}
	projectID := strings.TrimSpace(value[:separator])
	issueNumber, err := strconv.Atoi(strings.TrimSpace(value[separator+1:]))
	if err != nil || issueNumber <= 0 {
		return "", errors.New("task not found")
	}
	for _, task := range s.tasks {
		if task != nil && task.ProjectID == projectID && task.IssueNumber == issueNumber {
			return task.ID, nil
		}
	}
	return "", errors.New("task not found")
}

func (s *Store) hasOpenProposalLocked(userID, taskID string) bool {
	for _, note := range s.notifications {
		if note == nil || note.UserID != userID || note.Channel != "proposal" {
			continue
		}
		fields := splitLedgerReference(note.Status)
		if fields["proposal"] == "submitted" && fields["task"] == taskID {
			return true
		}
	}
	return false
}

func (s *Store) workerSubmittedProposalsLocked(userID string) []WorkerSubmittedProposal {
	rows := []WorkerSubmittedProposal{}
	for _, note := range s.notifications {
		if note == nil || note.Channel != "proposal" || note.UserID != userID {
			continue
		}
		rows = append(rows, s.workerSubmittedProposalFromNotificationLocked(note))
	}
	sort.Slice(rows, func(i, j int) bool {
		return rows[i].CreatedAt.After(rows[j].CreatedAt)
	})
	return rows
}

func (s *Store) projectSubmittedProposalsLocked(projectID string) []WorkerSubmittedProposal {
	rows := []WorkerSubmittedProposal{}
	seen := map[string]bool{}
	for _, note := range s.notifications {
		if note == nil || note.Channel != "proposal" || note.ProjectID != projectID {
			continue
		}
		fields := splitLedgerReference(note.Status)
		key := fields["task"] + "|" + fields["worker"]
		if seen[key] || fields["proposal"] != "submitted" {
			continue
		}
		seen[key] = true
		rows = append(rows, s.workerSubmittedProposalFromNotificationLocked(note))
	}
	sort.Slice(rows, func(i, j int) bool {
		return rows[i].CreatedAt.After(rows[j].CreatedAt)
	})
	return rows
}

func (s *Store) workerSubmittedProposalFromNotificationLocked(note *Notification) WorkerSubmittedProposal {
	if note == nil {
		return WorkerSubmittedProposal{}
	}
	fields := splitLedgerReference(note.Status)
	task := s.tasks[fields["task"]]
	project := s.projects[note.ProjectID]
	bidCents, _ := strconv.ParseInt(fields["bid"], 10, 64)
	estimatedHours, _ := strconv.ParseFloat(fields["hours"], 64)
	updatedAt := note.CreatedAt
	row := WorkerSubmittedProposal{
		ID:             note.ID,
		ProjectID:      note.ProjectID,
		TaskID:         fields["task"],
		WorkerID:       fields["worker"],
		CoverLetter:    note.Body,
		BidCents:       bidCents,
		EstimatedHours: estimatedHours,
		Availability:   fields["availability"],
		Status:         fields["proposal"],
		Reference:      note.Status,
		CreatedAt:      note.CreatedAt,
		UpdatedAt:      updatedAt,
	}
	if row.Status == "" {
		row.Status = "submitted"
	}
	if project != nil {
		row.ProjectTitle = publicLiveFeedProjectTitle(project)
	}
	if task != nil {
		claimID := marketplaceBountyID(task.ProjectID, task.IssueNumber)
		row.TaskID = claimID
		row.ClaimID = claimID
		row.IssueNumber = task.IssueNumber
		row.Title = task.Title
		if row.BidCents <= 0 {
			row.BidCents = task.RewardCents
		}
		if row.EstimatedHours <= 0 {
			row.EstimatedHours = marketplaceEstimatedHours(task)
		}
		row.Reference = proposalReference(claimID, row.WorkerID, row.BidCents, row.EstimatedHours, row.Availability)
	}
	return row
}

func publicProposalNotification(note *Notification, reference string) Notification {
	if note == nil {
		return Notification{}
	}
	copyNote := *note
	copyNote.UserID = ""
	copyNote.Status = reference
	return copyNote
}

func proposalReference(taskID, workerID string, bidCents int64, estimatedHours float64, availability string) string {
	parts := []string{
		"proposal:submitted",
		"task:" + sanitizeLedgerReferenceValue(taskID),
		"worker:" + sanitizeLedgerReferenceValue(workerID),
		"bid:" + strconv.FormatInt(bidCents, 10),
	}
	if estimatedHours > 0 {
		parts = append(parts, "hours:"+sanitizeLedgerReferenceValue(strconv.FormatFloat(estimatedHours, 'f', 2, 64)))
	}
	if availability = sanitizeLedgerReferenceValue(availability); availability != "" {
		parts = append(parts, "availability:"+availability)
	}
	return strings.Join(parts, ";")
}

func proposalWorkerID(user *User) string {
	if user == nil {
		return ""
	}
	if github := normalizeGitHubUsername(user.GitHubUsername); github != "" {
		return githubWorkerAccount(github)
	}
	if wallet := normalizeWalletAddress(user.WalletAddress); validWalletAddress(wallet) {
		return walletAccount(wallet)
	}
	return ""
}

func proposalText(value string, maxLength int) string {
	value = compactText(value)
	if maxLength <= 0 {
		return value
	}
	runes := []rune(value)
	if len(runes) > maxLength {
		return strings.TrimSpace(string(runes[:maxLength]))
	}
	return value
}

func proposalUpdatedAt(rows []WorkerSubmittedProposal, fallback time.Time) time.Time {
	updatedAt := fallback
	for _, row := range rows {
		if row.UpdatedAt.After(updatedAt) {
			updatedAt = row.UpdatedAt
		}
	}
	return updatedAt
}
