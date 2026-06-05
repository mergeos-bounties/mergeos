package core

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"
)

const submittedTaskStatus = "submitted"
const taskReviewChangesRequested = "changes_requested"

func (s *Store) SubmitTaskReview(userID string, role UserRole, taskRef string, req TaskSubmissionRequest) (TaskSubmissionResponse, error) {
	submission, err := normalizeTaskSubmissionRequest(req)
	if err != nil {
		return TaskSubmissionResponse{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	user := s.users[strings.TrimSpace(userID)]
	if user == nil {
		return TaskSubmissionResponse{}, errors.New("login is required")
	}
	taskID, err := s.resolveTaskClaimIDLocked(taskRef)
	if err != nil {
		return TaskSubmissionResponse{}, err
	}
	task, ok := s.tasks[taskID]
	if !ok || task == nil {
		return TaskSubmissionResponse{}, errors.New("task not found")
	}
	project, ok := s.projects[task.ProjectID]
	if !ok || project == nil {
		return TaskSubmissionResponse{}, errors.New("project not found")
	}
	if !taskCanSubmitReview(task) || strings.TrimSpace(task.WorkerID) == "" {
		return TaskSubmissionResponse{}, errors.New("task must be claimed before review evidence can be submitted")
	}
	if !canSubmitTaskEvidenceLocked(user, role, project, task) {
		return TaskSubmissionResponse{}, errors.New("assigned worker or project owner access is required")
	}

	now := time.Now().UTC()
	task.PullRequestURL = submission.PullRequestURL
	task.ReviewEvidenceURL = submission.ReviewEvidenceURL
	task.ReviewNotes = submission.ReviewNotes
	task.SubmittedAt = &now
	if task.Status != TaskAccepted {
		task.Status = TaskSubmitted
	}
	updateProjectTaskLocked(project, task)

	subject := "MergeOS task submitted: " + task.Title
	body := taskSubmissionNotificationBody(task)
	status := s.emailer.Send(project.ClientEmail, subject, body)
	s.addNotificationLocked(project.ClientUserID, project.ID, "task", subject, body, status)

	if err := s.saveLocked(); err != nil {
		return TaskSubmissionResponse{}, err
	}
	return taskSubmissionProtocolDocument(marketplaceBountyID(task.ProjectID, task.IssueNumber), task), nil
}

func taskCanSubmitReview(task *Task) bool {
	if task == nil {
		return false
	}
	return task.Status == TaskClaimed || task.Status == TaskSubmitted || task.Status == TaskAccepted
}

func (s *Store) RequestTaskChanges(userID string, role UserRole, taskRef string, req TaskReviewRequest) (TaskReviewResponse, error) {
	notes, err := normalizeTaskReviewNotes(req)
	if err != nil {
		return TaskReviewResponse{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	user := s.users[strings.TrimSpace(userID)]
	if user == nil {
		return TaskReviewResponse{}, errors.New("login is required")
	}
	taskID, err := s.resolveTaskClaimIDLocked(taskRef)
	if err != nil {
		return TaskReviewResponse{}, err
	}
	task, ok := s.tasks[taskID]
	if !ok || task == nil {
		return TaskReviewResponse{}, errors.New("task not found")
	}
	project, ok := s.projects[task.ProjectID]
	if !ok || project == nil {
		return TaskReviewResponse{}, errors.New("project not found")
	}
	if normalizeRole(role) != RoleAdmin && project.ClientUserID != user.ID {
		return TaskReviewResponse{}, errors.New("project owner or admin access is required")
	}
	if task.Status != TaskSubmitted {
		return TaskReviewResponse{}, errors.New("task must be submitted before changes can be requested")
	}

	now := time.Now().UTC()
	task.Status = TaskClaimed
	task.ReviewNotes = notes
	updateProjectTaskLocked(project, task)

	claimID := marketplaceBountyID(task.ProjectID, task.IssueNumber)
	reference := taskReviewReference(taskReviewChangesRequested, claimID, task.WorkerID)
	subject := "MergeOS changes requested: " + task.Title
	body := taskReviewChangesRequestedBody(task, notes)
	s.addNotificationLocked(project.ClientUserID, project.ID, "task_review", subject, body, reference)
	if worker := s.userForWorkerIDLocked(task.WorkerID); worker != nil && worker.ID != project.ClientUserID {
		s.addNotificationLocked(worker.ID, project.ID, "task_review", subject, body, reference)
	}

	if err := s.saveLocked(); err != nil {
		return TaskReviewResponse{}, err
	}
	return taskReviewProtocolDocument(claimID, taskReviewChangesRequested, now, task), nil
}

func normalizeTaskReviewNotes(req TaskReviewRequest) (string, error) {
	notes := protocolText(req.ReviewNotes, 2000, "")
	if notes == "" {
		notes = protocolText(req.Notes, 2000, "")
	}
	if notes == "" {
		notes = protocolText(req.Reason, 2000, "")
	}
	if len([]rune(notes)) < 12 {
		return "", errors.New("review_notes must be at least 12 characters")
	}
	return notes, nil
}

func normalizeTaskSubmissionRequest(req TaskSubmissionRequest) (TaskSubmissionRequest, error) {
	rawPullRequestURL := strings.TrimSpace(req.PullRequestURL)
	pullRequestURL := normalizeTaskSubmissionURL(req.PullRequestURL)
	if rawPullRequestURL != "" && pullRequestURL == "" {
		return TaskSubmissionRequest{}, errors.New("pull_request_url must be an http or https URL")
	}
	if pullRequestURL != "" && !isGitHubPullRequestURL(pullRequestURL) {
		return TaskSubmissionRequest{}, errors.New("pull_request_url must be a GitHub pull request URL")
	}

	rawEvidenceURL := strings.TrimSpace(req.ReviewEvidenceURL)
	evidenceURL := normalizeTaskSubmissionURL(rawEvidenceURL)
	if evidenceURL == "" {
		rawEvidenceURL = strings.TrimSpace(req.EvidenceURL)
		evidenceURL = normalizeTaskSubmissionURL(rawEvidenceURL)
	}
	if rawEvidenceURL != "" && evidenceURL == "" {
		return TaskSubmissionRequest{}, errors.New("evidence_url must be an http or https URL")
	}
	if evidenceURL != "" && !isHTTPURL(evidenceURL) {
		return TaskSubmissionRequest{}, errors.New("evidence_url must be an http or https URL")
	}

	notes := protocolText(req.ReviewNotes, 2000, "")
	if notes == "" {
		notes = protocolText(req.Notes, 2000, "")
	}
	if pullRequestURL == "" && evidenceURL == "" && len([]rune(notes)) < 12 {
		return TaskSubmissionRequest{}, errors.New("review notes, pull_request_url, or evidence_url is required")
	}
	if notes != "" && len([]rune(notes)) < 12 {
		return TaskSubmissionRequest{}, errors.New("review_notes must be at least 12 characters")
	}
	return TaskSubmissionRequest{
		PullRequestURL:    pullRequestURL,
		EvidenceURL:       evidenceURL,
		ReviewEvidenceURL: evidenceURL,
		Notes:             notes,
		ReviewNotes:       notes,
	}, nil
}

func normalizeTaskSubmissionURL(value string) string {
	value = publicLiveFeedURL(value)
	if value == "" {
		return ""
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return ""
	}
	parsed.Fragment = ""
	return parsed.String()
}

func isHTTPURL(value string) bool {
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil {
		return false
	}
	return (parsed.Scheme == "http" || parsed.Scheme == "https") && parsed.Host != ""
}

func isGitHubPullRequestURL(value string) bool {
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil {
		return false
	}
	if !strings.EqualFold(parsed.Host, "github.com") || parsed.Scheme != "https" {
		return false
	}
	parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	if len(parts) != 4 || parts[2] != "pull" {
		return false
	}
	for _, char := range parts[3] {
		if char < '0' || char > '9' {
			return false
		}
	}
	return parts[0] != "" && parts[1] != "" && parts[3] != ""
}

func canSubmitTaskEvidenceLocked(user *User, role UserRole, project *Project, task *Task) bool {
	if user == nil || project == nil || task == nil {
		return false
	}
	if normalizeRole(role) == RoleAdmin || project.ClientUserID == user.ID {
		return true
	}
	workerIDs, _ := workerIdentitySets(user)
	return workerIDs[workerIdentityKey(task.WorkerID)]
}

func (s *Store) userForWorkerIDLocked(workerID string) *User {
	key := workerIdentityKey(workerID)
	if key == "" {
		return nil
	}
	for _, user := range s.users {
		if user == nil {
			continue
		}
		workerIDs, _ := workerIdentitySets(user)
		if workerIDs[key] {
			return user
		}
	}
	return nil
}

func updateProjectTaskLocked(project *Project, task *Task) {
	if project == nil || task == nil {
		return
	}
	for index, projectTask := range project.Tasks {
		if projectTask != nil && projectTask.ID == task.ID {
			taskCopy := *task
			project.Tasks[index] = &taskCopy
			return
		}
	}
	taskCopy := *task
	project.Tasks = append(project.Tasks, &taskCopy)
}

func taskSubmissionNotificationBody(task *Task) string {
	if task == nil {
		return "A task was submitted for MergeOS review."
	}
	parts := []string{fmt.Sprintf("Task #%d was submitted for review.", task.IssueNumber)}
	if task.PullRequestURL != "" {
		parts = append(parts, "Pull request: "+task.PullRequestURL)
	}
	if task.ReviewEvidenceURL != "" {
		parts = append(parts, "Evidence: "+task.ReviewEvidenceURL)
	}
	if task.ReviewNotes != "" {
		parts = append(parts, "Notes: "+task.ReviewNotes)
	}
	return strings.Join(parts, "\n")
}

func taskReviewChangesRequestedBody(task *Task, notes string) string {
	if task == nil {
		return "Changes were requested before payout release."
	}
	parts := []string{fmt.Sprintf("Task #%d needs changes before payout release.", task.IssueNumber)}
	if strings.TrimSpace(notes) != "" {
		parts = append(parts, "Requested changes: "+notes)
	}
	if task.PullRequestURL != "" {
		parts = append(parts, "Pull request: "+task.PullRequestURL)
	}
	if task.ReviewEvidenceURL != "" {
		parts = append(parts, "Evidence: "+task.ReviewEvidenceURL)
	}
	return strings.Join(parts, "\n")
}

func taskSubmissionProtocolDocument(claimID string, task *Task) TaskSubmissionResponse {
	if task == nil {
		return TaskSubmissionResponse{
			ProtocolVersion: "mergeos.task-submission.v1",
			Kind:            "task_submission",
			Status:          submittedTaskStatus,
		}
	}
	claimID = strings.TrimSpace(claimID)
	if claimID == "" {
		claimID = marketplaceBountyID(task.ProjectID, task.IssueNumber)
	}
	taskCopy := *task
	taskCopy.IssueURL = marketplacePublicRepoURL(taskCopy.IssueURL)
	submittedAt := time.Now().UTC()
	if task.SubmittedAt != nil {
		submittedAt = *task.SubmittedAt
	}
	return TaskSubmissionResponse{
		ProtocolVersion:   "mergeos.task-submission.v1",
		Kind:              "task_submission",
		ID:                "submission:" + claimID,
		ClaimID:           claimID,
		TaskID:            task.ID,
		ProjectID:         task.ProjectID,
		IssueNumber:       task.IssueNumber,
		Title:             task.Title,
		Status:            submittedTaskStatus,
		WorkerKind:        task.WorkerKind,
		WorkerID:          task.WorkerID,
		AgentType:         task.AgentType,
		PullRequestURL:    task.PullRequestURL,
		ReviewEvidenceURL: task.ReviewEvidenceURL,
		ReviewNotes:       task.ReviewNotes,
		SubmittedAt:       submittedAt,
		Task:              taskCopy,
	}
}

func taskReviewProtocolDocument(claimID, decision string, requestedAt time.Time, task *Task) TaskReviewResponse {
	if task == nil {
		return TaskReviewResponse{
			ProtocolVersion: "mergeos.task-review.v1",
			Kind:            "task_review",
		}
	}
	claimID = strings.TrimSpace(claimID)
	if claimID == "" {
		claimID = marketplaceBountyID(task.ProjectID, task.IssueNumber)
	}
	taskCopy := *task
	taskCopy.IssueURL = marketplacePublicRepoURL(taskCopy.IssueURL)
	if requestedAt.IsZero() {
		requestedAt = time.Now().UTC()
	}
	return TaskReviewResponse{
		ProtocolVersion: "mergeos.task-review.v1",
		Kind:            "task_review",
		ID:              "review:" + claimID,
		ClaimID:         claimID,
		TaskID:          task.ID,
		ProjectID:       task.ProjectID,
		IssueNumber:     task.IssueNumber,
		Title:           task.Title,
		Decision:        strings.TrimSpace(decision),
		Status:          task.Status,
		WorkerKind:      task.WorkerKind,
		WorkerID:        task.WorkerID,
		AgentType:       task.AgentType,
		ReviewNotes:     task.ReviewNotes,
		RequestedAt:     requestedAt,
		Task:            taskCopy,
	}
}

func taskReviewReference(decision, taskID, workerID string) string {
	return strings.Join([]string{
		"task_review:" + sanitizeLedgerReferenceValue(decision),
		"task:" + sanitizeLedgerReferenceValue(taskID),
		"worker:" + sanitizeLedgerReferenceValue(workerID),
	}, ";")
}
