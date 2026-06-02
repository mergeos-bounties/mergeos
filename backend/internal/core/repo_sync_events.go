package core

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func (s *Store) RecordRepoIssueSyncEvent(report ProjectIssueSyncResponse) error {
	projectID := strings.TrimSpace(report.ProjectID)
	if projectID == "" {
		return nil
	}

	s.mu.RLock()
	project := s.projects[projectID]
	repository := strings.TrimSpace(report.SourceRepoURL)
	if repository == "" && project != nil {
		repository = project.BountyRepoName
		if repository == "" {
			repository = project.RepoURL
		}
	}
	s.mu.RUnlock()

	receivedAt := report.SyncedAt
	if receivedAt.IsZero() {
		receivedAt = time.Now().UTC()
	}
	return s.AddGeminiWebhookLog(GeminiWebhookLog{
		EventName:  "repo_issues_synced",
		Action:     "sync",
		Repository: sanitizeLedgerReferenceValue(repository),
		Sender:     "mergeos-repo-sync",
		Status:     "processed",
		StatusCode: http.StatusOK,
		CommentURL: publicLiveFeedURL(report.SourceRepoURL),
		Labels: []string{
			"project:" + projectID,
			fmt.Sprintf("imported:%d", report.ImportedIssueCount),
			fmt.Sprintf("added:%d", report.AddedTaskCount),
			fmt.Sprintf("updated:%d", report.UpdatedTaskCount),
			fmt.Sprintf("open:%d", report.OpenIssueCount),
			fmt.Sprintf("closed:%d", report.ClosedIssueCount),
		},
		ReceivedAt:  receivedAt,
		CompletedAt: &receivedAt,
	})
}

func webhookLogLabelInt(log *GeminiWebhookLog, key string) int {
	if log == nil {
		return 0
	}
	prefix := strings.TrimSpace(key) + ":"
	for _, label := range log.Labels {
		label = strings.TrimSpace(label)
		if !strings.HasPrefix(label, prefix) {
			continue
		}
		value, err := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(label, prefix)))
		if err == nil && value > 0 {
			return value
		}
		return 0
	}
	return 0
}
