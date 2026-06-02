package core

import (
	"errors"
	"strings"
	"time"
)

const defaultAgentActionActor = "ai-agent"

func (s *Store) RecordProjectAgentAction(projectID string, req AgentActionRequest) (AgentActionResponse, error) {
	action, err := normalizeAgentAction(req.Action)
	if err != nil {
		return AgentActionResponse{}, err
	}
	status, err := normalizeAgentActionStatus(req.Status)
	if err != nil {
		return AgentActionResponse{}, err
	}
	agentType := sanitizeLedgerReferenceValue(req.AgentType)
	if agentType == "" {
		agentType = defaultAgentActionActor
	}
	durationMillis := req.DurationMillis
	if durationMillis < 0 {
		durationMillis = 0
	}
	pullNumber := req.PullNumber
	if pullNumber < 0 {
		pullNumber = 0
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	project, ok := s.projects[strings.TrimSpace(projectID)]
	if !ok {
		return AgentActionResponse{}, errors.New("project not found")
	}
	if s.geminiWebhookLogs == nil {
		s.geminiWebhookLogs = map[string]*GeminiWebhookLog{}
	}

	now := time.Now().UTC()
	log := GeminiWebhookLog{
		ID:             geminiWebhookLogID(),
		EventName:      "agent_action",
		Action:         action,
		Repository:     projectAgentActionRepository(project),
		PullNumber:     pullNumber,
		Sender:         "agent:" + agentType,
		Status:         status,
		StatusCode:     agentActionStatusCode(status),
		CommentURL:     publicLiveFeedURL(req.ReferenceURL),
		Labels:         normalizeAgentActionLabels(req.Labels),
		DurationMillis: durationMillis,
		ReceivedAt:     now,
	}
	if durationMillis > 0 || status == "processed" || status == "failed" {
		completedAt := now
		log.CompletedAt = &completedAt
	}
	s.geminiWebhookLogs[log.ID] = &log
	s.trimGeminiWebhookLogsLocked()
	if err := s.saveLocked(); err != nil {
		return AgentActionResponse{}, err
	}
	return AgentActionResponse{Log: log}, nil
}

func normalizeAgentAction(value string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "review":
		return "review", nil
	case "test":
		return "test", nil
	case "generate", "gen":
		return "generate", nil
	case "deploy":
		return "deploy", nil
	case "scan":
		return "scan", nil
	default:
		return "", errors.New("action must be review, test, generate, deploy, or scan")
	}
}

func normalizeAgentActionStatus(value string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "":
		return "processed", nil
	case "received", "queued":
		return "received", nil
	case "running", "in_progress":
		return "running", nil
	case "processed", "complete", "completed", "success":
		return "processed", nil
	case "needs_review", "needs-review":
		return "needs_review", nil
	case "failed", "error":
		return "failed", nil
	default:
		return "", errors.New("status must be received, running, processed, needs_review, or failed")
	}
}

func agentActionStatusCode(status string) int {
	switch status {
	case "failed":
		return 500
	case "needs_review":
		return 202
	default:
		return 200
	}
}

func normalizeAgentActionLabels(values []string) []string {
	values = cleanStrings(values)
	labels := make([]string, 0, len(values))
	for _, value := range values {
		value = sanitizeLedgerReferenceValue(value)
		if value != "" {
			labels = append(labels, value)
		}
	}
	if len(labels) > 12 {
		return labels[:12]
	}
	return labels
}

func projectAgentActionRepository(project *Project) string {
	if project == nil {
		return ""
	}
	repository := sanitizeLedgerReferenceValue(project.BountyRepoName)
	if repository != "" {
		return repository
	}
	return sanitizeLedgerReferenceValue(project.RepoURL)
}
