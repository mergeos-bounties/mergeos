package core

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

const maxAdminOpsQueueItems = 120

func (s *Store) AdminOpsQueue() AdminOpsQueueResponse {
	s.mu.RLock()
	defer s.mu.RUnlock()

	response := AdminOpsQueueResponse{
		Items: []AdminOpsQueueItem{},
	}
	add := func(item AdminOpsQueueItem) {
		if item.CreatedAt.IsZero() {
			item.CreatedAt = time.Now().UTC()
		}
		if item.Status == "" {
			item.Status = "needs_review"
		}
		if item.Severity == "" {
			item.Severity = "medium"
		}
		response.Items = append(response.Items, item)
	}

	for _, task := range s.tasks {
		if task == nil || task.Status == TaskAccepted || normalizeIssueState(task.IssueState) != "closed" {
			continue
		}
		project := s.projects[task.ProjectID]
		projectTitle := publicLiveFeedProjectTitle(project)
		add(AdminOpsQueueItem{
			ID:           adminOpsItemID("payout", task.ID),
			Type:         "payout_review",
			Severity:     "high",
			Title:        fmt.Sprintf("Issue #%d needs payout review", task.IssueNumber),
			Body:         fmt.Sprintf("%s is closed while the MergeOS task is still open.", task.Title),
			ProjectID:    task.ProjectID,
			ProjectTitle: projectTitle,
			TaskID:       task.ID,
			IssueNumber:  task.IssueNumber,
			Reference:    adminOpsTaskReference(task),
			URL:          marketplacePublicRepoURL(task.IssueURL),
			Status:       "needs_payout_review",
			CreatedAt:    adminOpsTaskUpdatedAt(task),
		})
	}

	for _, note := range s.notifications {
		if note == nil || !adminOpsNotificationNeedsAttention(note.Status) {
			continue
		}
		project := s.projects[note.ProjectID]
		add(AdminOpsQueueItem{
			ID:           adminOpsItemID("dispute", note.ID),
			Type:         "dispute",
			Severity:     adminOpsNotificationSeverity(note.Status),
			Title:        "Delivery notification needs review",
			Body:         compactText(strings.TrimSpace(note.Subject + " - " + note.Body)),
			ProjectID:    note.ProjectID,
			ProjectTitle: publicLiveFeedProjectTitle(project),
			UserID:       note.UserID,
			Reference:    note.Channel,
			Status:       sanitizeLedgerReferenceValue(note.Status),
			CreatedAt:    note.CreatedAt,
		})
	}

	for _, log := range s.geminiWebhookLogs {
		if log == nil || !adminOpsGeminiLogNeedsModeration(log.Status) {
			continue
		}
		add(AdminOpsQueueItem{
			ID:        adminOpsItemID("automation", log.ID),
			Type:      "moderation",
			Severity:  adminOpsGeminiLogSeverity(log.Status),
			Title:     "AI review webhook needs moderation",
			Body:      publicLiveFeedAIBody(log),
			Reference: publicLiveFeedAIReference(log),
			URL:       publicLiveFeedURL(log.CommentURL),
			Status:    publicLiveFeedStatus(log.Status),
			CreatedAt: log.ReceivedAt,
		})
	}

	for _, review := range s.sslReviewRowsLocked() {
		if review == nil || !adminOpsSSLNeedsAttention(review) {
			continue
		}
		createdAt := time.Now().UTC()
		if review.LastCheckedAt != nil {
			createdAt = *review.LastCheckedAt
		}
		add(AdminOpsQueueItem{
			ID:        adminOpsItemID("security", review.Domain),
			Type:      "security_moderation",
			Severity:  adminOpsSSLSeverity(review),
			Title:     "SSL certificate needs review",
			Body:      adminOpsSSLBody(review),
			Reference: review.Domain,
			Status:    sanitizeLedgerReferenceValue(review.Status),
			CreatedAt: createdAt,
		})
	}

	for _, entry := range s.ledger {
		if entry.Type != "manual_credit" {
			continue
		}
		add(AdminOpsQueueItem{
			ID:        fmt.Sprintf("manual-credit:%d", entry.Sequence),
			Type:      "payout_audit",
			Severity:  "low",
			Title:     "Manual MRG credit audit",
			Body:      fmt.Sprintf("%s was credited to %s.", formatTokenAmount(entry.AmountCents), publicLedgerAccount(entry.ToAccount, "", "")),
			Reference: publicPullLedgerReference(entry.Reference),
			URL:       publicLiveFeedReferenceURL(entry.Reference),
			Status:    "recorded",
			CreatedAt: entry.CreatedAt,
		})
	}

	sort.Slice(response.Items, func(i, j int) bool {
		left, right := response.Items[i], response.Items[j]
		if adminOpsSeverityRank(left.Severity) != adminOpsSeverityRank(right.Severity) {
			return adminOpsSeverityRank(left.Severity) > adminOpsSeverityRank(right.Severity)
		}
		if left.CreatedAt.Equal(right.CreatedAt) {
			return left.ID > right.ID
		}
		return left.CreatedAt.After(right.CreatedAt)
	})
	if len(response.Items) > maxAdminOpsQueueItems {
		response.Items = response.Items[:maxAdminOpsQueueItems]
	}
	response.Stats = adminOpsQueueStats(response.Items)
	return response
}

func adminOpsQueueStats(items []AdminOpsQueueItem) AdminOpsQueueStats {
	stats := AdminOpsQueueStats{TotalCount: len(items)}
	for _, item := range items {
		if stats.UpdatedAt == nil || item.CreatedAt.After(*stats.UpdatedAt) {
			updatedAt := item.CreatedAt
			stats.UpdatedAt = &updatedAt
		}
		switch item.Type {
		case "dispute":
			stats.DisputeCount++
		case "moderation":
			stats.ModerationCount++
		case "security_moderation":
			stats.ModerationCount++
			stats.SecurityCount++
		case "payout_review", "payout_audit":
			stats.PayoutReviewCount++
		}
		if item.Severity == "critical" {
			stats.CriticalCount++
		}
	}
	return stats
}

func adminOpsItemID(prefix, id string) string {
	id = sanitizeLedgerReferenceValue(id)
	if id == "" {
		id = "unknown"
	}
	return prefix + ":" + id
}

func adminOpsTaskReference(task *Task) string {
	if task == nil {
		return ""
	}
	if url := publicLiveFeedURL(task.IssueURL); url != "" {
		return url
	}
	if task.IssueNumber > 0 {
		return fmt.Sprintf("issue:%d", task.IssueNumber)
	}
	return task.ID
}

func adminOpsTaskUpdatedAt(task *Task) time.Time {
	if task.AcceptedAt != nil {
		return *task.AcceptedAt
	}
	return task.CreatedAt
}

func adminOpsNotificationNeedsAttention(status string) bool {
	status = strings.ToLower(strings.TrimSpace(status))
	return strings.HasPrefix(status, "error:") || strings.HasPrefix(status, "skipped:")
}

func adminOpsNotificationSeverity(status string) string {
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(status)), "error:") {
		return "high"
	}
	return "medium"
}

func adminOpsGeminiLogNeedsModeration(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "failed", "unauthorized", "bad_request", "service_unavailable":
		return true
	default:
		return false
	}
}

func adminOpsGeminiLogSeverity(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "unauthorized":
		return "critical"
	case "failed", "service_unavailable":
		return "high"
	default:
		return "medium"
	}
}

func adminOpsSSLNeedsAttention(review *SSLReviewStatus) bool {
	status := strings.ToLower(strings.TrimSpace(review.Status))
	if status == "" || status == "ok" || status == "pending" {
		return false
	}
	return true
}

func adminOpsSSLSeverity(review *SSLReviewStatus) string {
	if strings.EqualFold(review.Status, "error") || review.DaysRemaining <= 0 {
		return "critical"
	}
	if review.DaysRemaining <= 14 {
		return "high"
	}
	return "medium"
}

func adminOpsSSLBody(review *SSLReviewStatus) string {
	if strings.TrimSpace(review.Error) != "" {
		return compactText(review.Error)
	}
	if review.DaysRemaining > 0 {
		return fmt.Sprintf("%s has %d certificate days remaining.", review.Domain, review.DaysRemaining)
	}
	return review.Domain + " certificate status is " + sanitizeLedgerReferenceValue(review.Status) + "."
}

func adminOpsSeverityRank(severity string) int {
	switch strings.ToLower(strings.TrimSpace(severity)) {
	case "critical":
		return 4
	case "high":
		return 3
	case "medium":
		return 2
	case "low":
		return 1
	default:
		return 0
	}
}
