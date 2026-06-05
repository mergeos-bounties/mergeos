package core

import (
	"errors"
	"fmt"
	"strings"
)

func (s *Store) CreateDispute(userID string, role UserRole, req CreateDisputeRequest) (CreateDisputeResponse, error) {
	userID = strings.TrimSpace(userID)
	subject := disputeText(req.Subject, 160)
	body := disputeText(req.Body, 2000)
	if body == "" {
		return CreateDisputeResponse{}, errors.New("dispute body is required")
	}
	severity := normalizeDisputeSeverity(req.Severity)

	s.mu.Lock()
	defer s.mu.Unlock()

	user := s.users[userID]
	if user == nil {
		return CreateDisputeResponse{}, errors.New("login is required")
	}

	projectID := strings.TrimSpace(req.ProjectID)
	taskID := strings.TrimSpace(req.TaskID)
	var task *Task
	var project *Project
	if taskID != "" {
		task = s.tasks[taskID]
		if task == nil {
			return CreateDisputeResponse{}, errors.New("task not found")
		}
		if projectID != "" && task.ProjectID != projectID {
			return CreateDisputeResponse{}, errors.New("task does not belong to project")
		}
		projectID = task.ProjectID
		project = s.projects[projectID]
	} else if projectID != "" {
		project = s.projects[projectID]
	} else {
		return CreateDisputeResponse{}, errors.New("project_id or task_id is required")
	}
	if project == nil {
		return CreateDisputeResponse{}, errors.New("project not found")
	}
	if !s.canCreateDisputeLocked(user, role, project, task) {
		return CreateDisputeResponse{}, errors.New("project access is required")
	}

	if subject == "" {
		subject = "Delivery dispute"
		if task != nil && task.IssueNumber > 0 {
			subject = fmt.Sprintf("Delivery dispute for issue #%d", task.IssueNumber)
		}
	}
	if task != nil {
		body = fmt.Sprintf("Task #%d: %s", task.IssueNumber, body)
	}
	note := s.addNotificationLocked(user.ID, project.ID, "dispute", subject, body, "dispute:"+severity)
	if err := s.saveLocked(); err != nil {
		return CreateDisputeResponse{}, err
	}
	responseTaskID := ""
	if task != nil {
		responseTaskID = task.ID
	}
	return CreateDisputeResponse{
		ProtocolVersion: "mergeos.dispute.v1",
		Kind:            "dispute",
		DisputeID:       note.ID,
		ProjectID:       project.ID,
		TaskID:          responseTaskID,
		UserID:          user.ID,
		Severity:        severity,
		Status:          note.Status,
		Subject:         note.Subject,
		Body:            note.Body,
		Notification:    *note,
		CreatedAt:       note.CreatedAt,
	}, nil
}

func (s *Store) canCreateDisputeLocked(user *User, role UserRole, project *Project, task *Task) bool {
	if normalizeRole(role) == RoleAdmin {
		return true
	}
	if project != nil && project.ClientUserID == user.ID {
		return true
	}
	if task == nil || task.Status != TaskAccepted {
		return false
	}
	workerID := strings.ToLower(strings.TrimSpace(normalizeWorkerID(task.WorkerID)))
	if workerID == "" {
		return false
	}
	if github := normalizeGitHubUsername(user.GitHubUsername); github != "" && workerID == strings.ToLower(githubWorkerAccount(github)) {
		return true
	}
	if wallet := normalizeWorkerID(user.WalletAddress); wallet != "" && workerID == strings.ToLower(wallet) {
		return true
	}
	return false
}

func normalizeDisputeSeverity(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "critical", "high", "medium", "low":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return "high"
	}
}

func disputeText(value string, maxLength int) string {
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
