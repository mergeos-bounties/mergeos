package core

import (
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

const maxAdminOpsQueueItems = 120
const adminOpsRapidPayoutThreshold = 3

const adminOpsRapidPayoutWindow = 10 * time.Minute

func (s *Store) AdminOpsQueue() AdminOpsQueueResponse {
	s.mu.RLock()
	defer s.mu.RUnlock()

	response := AdminOpsQueueResponse{
		ProtocolVersion: "mergeos.admin-ops.v1",
		Kind:            "admin_ops",
		Items:           []AdminOpsQueueItem{},
		OutputContracts: adminOpsQueueOutputContracts(),
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
		if task == nil || taskIsReleased(task) || normalizeIssueState(task.IssueState) != "closed" {
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
			Actions:      adminOpsActions("payout_review", task.ID, marketplacePublicRepoURL(task.IssueURL)),
			CreatedAt:    adminOpsTaskUpdatedAt(task),
		})
	}

	for _, note := range s.notifications {
		if note != nil && note.Channel == "proposal" {
			project := s.projects[note.ProjectID]
			if project == nil || note.UserID == project.ClientUserID {
				continue
			}
			fields := splitLedgerReference(note.Status)
			if !proposalOpenStatus(fields["proposal"]) {
				continue
			}
			taskID := fields["task"]
			task := s.tasks[taskID]
			add(AdminOpsQueueItem{
				ID:           adminOpsItemID("proposal", note.ID),
				Type:         "proposal_review",
				Severity:     "medium",
				Title:        note.Subject,
				Body:         compactText(note.Body),
				ProjectID:    note.ProjectID,
				ProjectTitle: publicLiveFeedProjectTitle(project),
				TaskID:       taskID,
				UserID:       note.UserID,
				Reference:    note.Status,
				URL:          marketplacePublicRepoURL(taskIssueURL(task)),
				Status:       "submitted",
				Actions:      adminOpsActions("proposal_review", taskID, marketplacePublicRepoURL(taskIssueURL(task))),
				CreatedAt:    note.CreatedAt,
			})
			continue
		}
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
			Actions:      adminOpsActions("dispute", "", ""),
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
			Actions:   adminOpsActions("moderation", "", publicLiveFeedURL(log.CommentURL)),
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
			Actions:   adminOpsActions("security_moderation", "", ""),
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
			Actions:   adminOpsActions("payout_audit", "", publicLiveFeedReferenceURL(entry.Reference)),
			CreatedAt: entry.CreatedAt,
		})
	}

	for _, entry := range s.ledger {
		if entry.Type != "airdrop_claim" && entry.Type != "presale_reservation" && entry.Type != "token_launch_brief" {
			continue
		}
		add(adminOpsTokenWorkflowItem(entry))
	}

	for _, item := range s.adminOpsFraudItemsLocked() {
		add(item)
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
	for i := range response.Items {
		response.Items[i].Actions = adminOpsExecutableActions(response.Items[i])
	}
	response.Stats = adminOpsQueueStats(response.Items)
	for _, item := range response.Items {
		response.Stats.BlockedPayoutCents += adminOpsItemBlockedPayoutCentsFromTasks(s.tasks, item)
	}
	return response
}

func (s *Store) AdminDisputes() AdminDisputesResponse {
	queue := s.AdminOpsQueue()
	response := AdminDisputesResponse{
		ProtocolVersion: "mergeos.admin-ops.v1",
		Kind:            "admin_disputes",
		Stats: AdminDisputesStats{
			TotalCount:        queue.Stats.TotalCount,
			CriticalCount:     queue.Stats.CriticalCount,
			DisputeCount:      queue.Stats.DisputeCount,
			ModerationCount:   queue.Stats.ModerationCount,
			PayoutReviewCount: queue.Stats.PayoutReviewCount,
			ProposalCount:     queue.Stats.ProposalCount,
			FraudCount:        queue.Stats.FraudCount,
			SecurityCount:     queue.Stats.SecurityCount,
			UpdatedAt:         queue.Stats.UpdatedAt,
		},
		Items: append([]AdminOpsQueueItem(nil), queue.Items...),
		OutputContracts: []AgentOutputContract{{
			Action:            "refresh_admin_disputes",
			ArtifactKind:      "admin_dispute_lanes",
			OutputEndpoint:    "/api/admin/disputes",
			OutputProtocol:    "mergeos.admin-ops.v1",
			OutputProtocolURL: "/protocol/admin-ops.v1.schema.json",
			PublicURL:         "/protocol/admin-ops.v1.schema.json",
		}},
	}

	lanes := map[string]*AdminDisputeLane{}
	for _, item := range queue.Items {
		laneMeta := adminDisputeLaneForType(item.Type)
		lane := lanes[laneMeta.ID]
		if lane == nil {
			lane = &AdminDisputeLane{
				ID:    laneMeta.ID,
				Title: laneMeta.Title,
				Body:  laneMeta.Body,
				Tone:  laneMeta.Tone,
				Items: []AdminOpsQueueItem{},
			}
			lanes[laneMeta.ID] = lane
		}
		lane.Count++
		if item.Severity == "critical" {
			lane.CriticalCount++
		}
		if item.Severity == "high" {
			lane.HighCount++
			response.Stats.HighCount++
		}
		if item.Type == "token_workflow_review" {
			response.Stats.TokenWorkflowCount++
		}
		reward := s.adminOpsItemBlockedPayoutCents(item)
		lane.RewardCents += reward
		response.Stats.BlockedPayoutCents += reward
		lane.Items = append(lane.Items, item)
	}

	order := []string{"disputes", "payouts", "moderation", "proposals", "fraud", "security", "token"}
	for _, id := range order {
		if lane := lanes[id]; lane != nil {
			response.Lanes = append(response.Lanes, *lane)
		}
	}
	return response
}

type adminDisputeLaneMeta struct {
	ID    string
	Title string
	Body  string
	Tone  string
}

func adminDisputeLaneForType(itemType string) adminDisputeLaneMeta {
	switch itemType {
	case "dispute":
		return adminDisputeLaneMeta{
			ID:    "disputes",
			Title: "Customer disputes",
			Body:  "Scope, acceptance, and delivery conflicts that need admin resolution before payout movement.",
			Tone:  "amber",
		}
	case "payout_review", "payout_audit":
		return adminDisputeLaneMeta{
			ID:    "payouts",
			Title: "Payout pressure",
			Body:  "Closed issues, manual credits, and payout evidence that need treasury review.",
			Tone:  "blue",
		}
	case "moderation":
		return adminDisputeLaneMeta{
			ID:    "moderation",
			Title: "AI moderation",
			Body:  "Webhook, review, and automation failures that need operator attention.",
			Tone:  "purple",
		}
	case "proposal_review":
		return adminDisputeLaneMeta{
			ID:    "proposals",
			Title: "Proposal decisions",
			Body:  "Contributor proposals waiting for accept, decline, or follow-up routing.",
			Tone:  "green",
		}
	case "fraud_review":
		return adminDisputeLaneMeta{
			ID:    "fraud",
			Title: "Fraud signals",
			Body:  "Duplicate payout references, rapid payout bursts, and repeated identities.",
			Tone:  "amber",
		}
	case "security_moderation":
		return adminDisputeLaneMeta{
			ID:    "security",
			Title: "Security checks",
			Body:  "SSL, webhook, and operational security exceptions that block confidence.",
			Tone:  "red",
		}
	case "token_workflow_review":
		return adminDisputeLaneMeta{
			ID:    "token",
			Title: "Token workflow review",
			Body:  "Airdrop and presale proofs that need operator approval before allocation moves forward.",
			Tone:  "green",
		}
	default:
		return adminDisputeLaneMeta{
			ID:    "moderation",
			Title: "Operations review",
			Body:  "Operational signals that need admin review.",
			Tone:  "blue",
		}
	}
}

func (s *Store) adminOpsItemBlockedPayoutCents(item AdminOpsQueueItem) int64 {
	if item.Type != "payout_review" && item.Type != "dispute" {
		return 0
	}
	taskID := strings.TrimSpace(item.TaskID)
	if taskID == "" {
		return 0
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if task := s.tasks[taskID]; task != nil {
		return task.RewardCents
	}
	return 0
}

func adminOpsItemBlockedPayoutCentsFromTasks(tasks map[string]*Task, item AdminOpsQueueItem) int64 {
	if item.Type != "payout_review" && item.Type != "dispute" {
		return 0
	}
	taskID := strings.TrimSpace(item.TaskID)
	if taskID == "" {
		return 0
	}
	if task := tasks[taskID]; task != nil {
		return task.RewardCents
	}
	return 0
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
		case "proposal_review":
			stats.ProposalCount++
		case "moderation":
			stats.ModerationCount++
		case "security_moderation":
			stats.ModerationCount++
			stats.SecurityCount++
		case "payout_review", "payout_audit":
			stats.PayoutReviewCount++
		case "token_workflow_review":
			stats.ModerationCount++
			stats.TokenWorkflowCount++
		case "fraud_review":
			stats.FraudCount++
		}
		if item.Severity == "critical" {
			stats.CriticalCount++
		}
		if item.Severity == "high" {
			stats.HighCount++
		}
	}
	return stats
}

func adminOpsQueueOutputContracts() []AgentOutputContract {
	return []AgentOutputContract{
		{
			Action:            "refresh_admin_ops",
			ArtifactKind:      "admin_ops_queue",
			OutputEndpoint:    "/api/admin/ops-queue",
			OutputProtocol:    "mergeos.admin-ops.v1",
			OutputProtocolURL: "/protocol/admin-ops.v1.schema.json",
			PublicURL:         "/protocol/admin-ops.v1.schema.json",
		},
		{
			Action:            "prove_ledger",
			ArtifactKind:      "ledger_proof",
			OutputEndpoint:    "/api/public/ledger/proof",
			OutputProtocol:    "mergeos.ledger-proof.v1",
			OutputProtocolURL: "/protocol/ledger-proof.v1.schema.json",
			PublicURL:         "/api/public/ledger/proof",
		},
	}
}

func adminOpsItemID(prefix, id string) string {
	id = sanitizeLedgerReferenceValue(id)
	if id == "" {
		id = "unknown"
	}
	return prefix + ":" + id
}

func adminOpsActions(itemType, taskID, url string) []AdminOpsQueueAction {
	actions := []AdminOpsQueueAction{}
	add := func(id, label, actionType, actionURL string) {
		actions = append(actions, AdminOpsQueueAction{
			ID:    id,
			Label: label,
			Type:  actionType,
			URL:   publicLiveFeedURL(actionURL),
		})
	}
	switch itemType {
	case "payout_review":
		if strings.TrimSpace(taskID) != "" {
			add("review-prs", "Review PRs", "review_task_pulls", "")
		}
		if publicLiveFeedURL(url) != "" {
			add("open-issue", "Open Issue", "open_url", url)
		}
	case "security_moderation":
		add("run-ssl-review", "Run SSL Review", "run_ssl_review", "")
	case "moderation", "payout_audit":
		if publicLiveFeedURL(url) != "" {
			add("open-proof", "Open Proof", "open_url", url)
		}
	case "dispute":
		add("refresh-queue", "Refresh Queue", "refresh_admin_ops", "")
	case "proposal_review":
		if publicLiveFeedURL(url) != "" {
			add("open-task", "Open Task", "open_url", url)
		}
		add("refresh-queue", "Refresh Queue", "refresh_admin_ops", "")
	case "token_workflow_review":
		if publicLiveFeedURL(url) != "" {
			add("open-proof", "Open Proof", "open_url", url)
		}
		add("refresh-queue", "Refresh Queue", "refresh_admin_ops", "")
	case "fraud_review":
		if publicLiveFeedURL(url) != "" {
			add("open-proof", "Open Proof", "open_url", url)
		}
		add("refresh-queue", "Refresh Queue", "refresh_admin_ops", "")
	}
	return actions
}

func adminOpsExecutableActions(item AdminOpsQueueItem) []AdminOpsQueueAction {
	actions := append([]AdminOpsQueueAction(nil), item.Actions...)
	for i := range actions {
		actions[i] = adminOpsExecutableAction(item, actions[i])
	}
	return actions
}

func adminOpsExecutableAction(item AdminOpsQueueItem, action AdminOpsQueueAction) AdminOpsQueueAction {
	switch action.Type {
	case "review_task_pulls":
		taskID := strings.TrimSpace(item.TaskID)
		if taskID != "" {
			action.Method = http.MethodGet
			action.Endpoint = "/api/admin/tasks/" + url.PathEscape(taskID) + "/pulls"
			action.Payload = map[string]any{"task_id": taskID}
		}
	case "run_ssl_review":
		action.Method = http.MethodPost
		action.Endpoint = "/api/admin/ssl/review"
		if reference := strings.TrimSpace(item.Reference); reference != "" {
			action.Payload = map[string]any{"domain": reference}
		}
	case "refresh_admin_ops":
		action.Method = http.MethodGet
		action.Endpoint = "/api/admin/ops-queue"
	case "open_url":
		if target := publicLiveFeedURL(action.URL); target != "" {
			action.Method = http.MethodGet
			action.Endpoint = target
		}
	}
	action.OutputContracts = adminOpsActionOutputContracts(item, action)
	return action
}

func adminOpsActionOutputContracts(item AdminOpsQueueItem, action AdminOpsQueueAction) []AgentOutputContract {
	endpoint := strings.TrimSpace(action.Endpoint)
	if endpoint == "" {
		endpoint = strings.TrimSpace(action.URL)
	}
	actionType := strings.TrimSpace(action.Type)
	switch actionType {
	case "review_task_pulls":
		return []AgentOutputContract{{
			Action:            actionType,
			ArtifactKind:      "admin_pr_review_context",
			OutputEndpoint:    endpoint,
			OutputProtocol:    "mergeos.pr-monitor.v1",
			OutputProtocolURL: "/protocol/pr-monitor.v1.schema.json",
			PublicURL:         publicLiveFeedURL(item.URL),
		}}
	case "run_ssl_review":
		return []AgentOutputContract{{
			Action:            actionType,
			ArtifactKind:      "security_review",
			OutputEndpoint:    endpoint,
			OutputProtocol:    "mergeos.admin-ops.v1",
			OutputProtocolURL: "/protocol/admin-ops.v1.schema.json",
			PublicURL:         "/protocol/admin-ops.v1.schema.json",
		}}
	case "refresh_admin_ops":
		return []AgentOutputContract{{
			Action:            actionType,
			ArtifactKind:      "admin_ops_queue",
			OutputEndpoint:    endpoint,
			OutputProtocol:    "mergeos.admin-ops.v1",
			OutputProtocolURL: "/protocol/admin-ops.v1.schema.json",
			PublicURL:         "/protocol/admin-ops.v1.schema.json",
		}}
	case "open_url":
		protocol := "mergeos.ledger-proof.v1"
		protocolURL := "/protocol/ledger-proof.v1.schema.json"
		artifact := "public_proof"
		if strings.Contains(strings.ToLower(endpoint), "github.com/") {
			protocol = "mergeos.event.v1"
			protocolURL = "/protocol/event.v1.schema.json"
			artifact = "external_reference"
		}
		return []AgentOutputContract{{
			Action:            actionType,
			ArtifactKind:      artifact,
			OutputEndpoint:    endpoint,
			OutputProtocol:    protocol,
			OutputProtocolURL: protocolURL,
			PublicURL:         publicLiveFeedURL(endpoint),
		}}
	default:
		return nil
	}
}

func adminOpsTokenWorkflowItem(entry LedgerEntry) AdminOpsQueueItem {
	fields := splitLedgerReference(entry.Reference)
	kind := "token workflow"
	title := "Token workflow needs review"
	status := "pending_review"
	severity := "medium"
	reference := publicTokenWorkflowReviewReference(entry)
	body := fmt.Sprintf("%s for %s needs operator review before allocation moves forward.", formatTokenAmount(entry.AmountCents), publicLedgerAccount(entry.FromAccount, "", ""))
	if entry.Type == "airdrop_claim" {
		kind = "airdrop"
		mission := sanitizeLedgerReferenceValue(fields["mission"])
		if mission == "" {
			mission = "mission"
		}
		title = "Airdrop claim needs review"
		body = fmt.Sprintf("%s claim for %s needs mission proof review.", formatTokenAmount(entry.AmountCents), mission)
		if sanitizeLedgerReferenceValue(fields["proof"]) != "" {
			severity = "low"
		}
	}
	if entry.Type == "presale_reservation" {
		kind = "presale"
		tier := sanitizeLedgerReferenceValue(fields["tier"])
		rail := sanitizeLedgerReferenceValue(fields["rail"])
		if tier == "" {
			tier = "standard"
		}
		if rail == "" {
			rail = "manual_review"
		}
		title = "Presale reservation needs review"
		body = fmt.Sprintf("%s %s reservation via %s needs funding review.", formatTokenAmount(entry.AmountCents), tier, rail)
	}
	if entry.Type == "token_launch_brief" {
		kind = "token-launch"
		launchType := sanitizeLedgerReferenceValue(fields["type"])
		if launchType == "" {
			launchType = "token"
		}
		title = "CEO token launch brief needs review"
		body = fmt.Sprintf("%s launch brief needs CEO research, risk review, and open/no-open decision.", marketplaceTitle(launchType))
		severity = "medium"
	}
	return AdminOpsQueueItem{
		ID:        fmt.Sprintf("%s:%d", kind, entry.Sequence),
		Type:      "token_workflow_review",
		Severity:  severity,
		Title:     title,
		Body:      compactText(body),
		Reference: reference,
		Status:    status,
		Actions:   adminOpsActions("token_workflow_review", "", publicLiveFeedReferenceURL(entry.Reference)),
		CreatedAt: entry.CreatedAt,
	}
}

func publicTokenWorkflowReviewReference(entry LedgerEntry) string {
	fields := splitLedgerReference(entry.Reference)
	parts := []string{}
	if entry.Type == "airdrop_claim" {
		if claimID := sanitizeLedgerReferenceValue(fields["airdrop"]); claimID != "" {
			parts = append(parts, "airdrop:"+claimID)
		}
		if mission := sanitizeLedgerReferenceValue(fields["mission"]); mission != "" {
			parts = append(parts, "mission:"+mission)
		}
	}
	if entry.Type == "presale_reservation" {
		if reservationID := sanitizeLedgerReferenceValue(fields["presale"]); reservationID != "" {
			parts = append(parts, "presale:"+reservationID)
		}
		if tier := sanitizeLedgerReferenceValue(fields["tier"]); tier != "" {
			parts = append(parts, "tier:"+tier)
		}
		if rail := sanitizeLedgerReferenceValue(fields["rail"]); rail != "" {
			parts = append(parts, "rail:"+rail)
		}
	}
	if entry.Type == "token_launch_brief" {
		if briefID := sanitizeLedgerReferenceValue(fields["launch_brief"]); briefID != "" {
			parts = append(parts, "launch_brief:"+briefID)
		}
		if launchType := sanitizeLedgerReferenceValue(fields["type"]); launchType != "" {
			parts = append(parts, "type:"+launchType)
		}
	}
	parts = append(parts, fmt.Sprintf("ledger:%d", entry.Sequence))
	return strings.Join(parts, ";")
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

func taskIssueURL(task *Task) string {
	if task == nil {
		return ""
	}
	return task.IssueURL
}

func adminOpsTaskUpdatedAt(task *Task) time.Time {
	if task.AcceptedAt != nil {
		return *task.AcceptedAt
	}
	return task.CreatedAt
}

func adminOpsNotificationNeedsAttention(status string) bool {
	status = strings.ToLower(strings.TrimSpace(status))
	return strings.HasPrefix(status, "error:") || strings.HasPrefix(status, "skipped:") || strings.HasPrefix(status, "dispute:")
}

func adminOpsNotificationSeverity(status string) string {
	status = strings.ToLower(strings.TrimSpace(status))
	if strings.HasPrefix(status, "dispute:") {
		return normalizeDisputeSeverity(strings.TrimPrefix(status, "dispute:"))
	}
	if strings.HasPrefix(status, "error:") {
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

func (s *Store) adminOpsFraudItemsLocked() []AdminOpsQueueItem {
	items := []AdminOpsQueueItem{}
	byReference := map[string][]LedgerEntry{}
	byAccount := map[string][]LedgerEntry{}

	for _, entry := range s.ledger {
		if !adminOpsLedgerEntryIsPayout(entry) {
			continue
		}
		account := strings.ToLower(strings.TrimSpace(entry.ToAccount))
		if account != "" {
			byAccount[account] = append(byAccount[account], entry)
		}
		if referenceKey := adminOpsFraudReferenceKey(entry.Reference); referenceKey != "" {
			byReference[referenceKey] = append(byReference[referenceKey], entry)
		}
	}

	for referenceKey, rows := range byReference {
		if len(rows) < 2 {
			continue
		}
		displayReference := adminOpsFraudDisplayReference(rows[0].Reference)
		accounts := adminOpsFraudAccounts(rows)
		severity := "high"
		if len(accounts) > 1 {
			severity = "critical"
		}
		items = append(items, AdminOpsQueueItem{
			ID:        adminOpsItemID("fraud-duplicate", referenceKey),
			Type:      "fraud_review",
			Severity:  severity,
			Title:     "Duplicate payout reference",
			Body:      fmt.Sprintf("%d payout ledger rows share %s across %s. Review before releasing more MRG.", len(rows), displayReference, strings.Join(accounts, ", ")),
			Reference: displayReference,
			URL:       publicLiveFeedReferenceURL(rows[0].Reference),
			Status:    "duplicate_payout_reference",
			Actions:   adminOpsActions("fraud_review", "", publicLiveFeedReferenceURL(rows[0].Reference)),
			CreatedAt: adminOpsLatestLedgerCreatedAt(rows),
		})
	}

	for account, rows := range byAccount {
		count, latest := adminOpsRapidPayoutBurst(rows)
		if count < adminOpsRapidPayoutThreshold {
			continue
		}
		accountLabel := publicLedgerAccount(account, "", "")
		items = append(items, AdminOpsQueueItem{
			ID:        adminOpsItemID("fraud-burst", account),
			Type:      "fraud_review",
			Severity:  "high",
			Title:     "Rapid payout burst",
			Body:      fmt.Sprintf("%s received %d payouts inside 10 minutes. Review payout intent and duplicate work before approving more credits.", accountLabel, count),
			Reference: accountLabel,
			Status:    "rapid_payout_burst",
			Actions:   adminOpsActions("fraud_review", "", ""),
			CreatedAt: latest,
		})
	}

	items = append(items, s.adminOpsDuplicateIdentityItemsLocked()...)
	sort.Slice(items, func(i, j int) bool {
		if items[i].CreatedAt.Equal(items[j].CreatedAt) {
			return items[i].ID < items[j].ID
		}
		return items[i].CreatedAt.After(items[j].CreatedAt)
	})
	return items
}

func adminOpsLedgerEntryIsPayout(entry LedgerEntry) bool {
	if entry.Type != "task_payment" && entry.Type != "manual_credit" {
		return false
	}
	return entry.AmountCents > 0 && strings.TrimSpace(entry.ToAccount) != ""
}

func adminOpsFraudReferenceKey(reference string) string {
	fields := splitLedgerReference(reference)
	if pullURL := normalizeLedgerPullURL(fields["pr"]); pullURL != "" {
		return "pr:" + strings.ToLower(pullURL)
	}
	if taskID := ledgerReferenceTaskID(reference); taskID != "" {
		return "task:" + strings.ToLower(sanitizeLedgerReferenceValue(taskID))
	}
	reference = strings.ToLower(sanitizeLedgerReferenceValue(reference))
	if reference == "" {
		return ""
	}
	return "ref:" + reference
}

func adminOpsFraudDisplayReference(reference string) string {
	if pullReference := publicPullLedgerReference(reference); pullReference != "" {
		return pullReference
	}
	if taskID := ledgerReferenceTaskID(reference); taskID != "" {
		return "task:" + sanitizeLedgerReferenceValue(taskID)
	}
	return sanitizeLedgerReferenceValue(reference)
}

func adminOpsFraudAccounts(rows []LedgerEntry) []string {
	seen := map[string]bool{}
	accounts := []string{}
	for _, row := range rows {
		account := strings.ToLower(strings.TrimSpace(row.ToAccount))
		if account == "" || seen[account] {
			continue
		}
		seen[account] = true
		accounts = append(accounts, publicLedgerAccount(account, "", ""))
	}
	sort.Strings(accounts)
	if len(accounts) > 3 {
		return append(accounts[:3], fmt.Sprintf("+%d more", len(accounts)-3))
	}
	return accounts
}

func adminOpsLatestLedgerCreatedAt(rows []LedgerEntry) time.Time {
	latest := time.Time{}
	for _, row := range rows {
		if latest.IsZero() || row.CreatedAt.After(latest) {
			latest = row.CreatedAt
		}
	}
	return latest
}

func adminOpsRapidPayoutBurst(rows []LedgerEntry) (int, time.Time) {
	if len(rows) < adminOpsRapidPayoutThreshold {
		return 0, time.Time{}
	}
	sort.Slice(rows, func(i, j int) bool {
		return rows[i].CreatedAt.Before(rows[j].CreatedAt)
	})
	for start := range rows {
		count := 1
		latest := rows[start].CreatedAt
		for end := start + 1; end < len(rows); end++ {
			if rows[end].CreatedAt.Sub(rows[start].CreatedAt) > adminOpsRapidPayoutWindow {
				break
			}
			count++
			if rows[end].CreatedAt.After(latest) {
				latest = rows[end].CreatedAt
			}
		}
		if count >= adminOpsRapidPayoutThreshold {
			return count, latest
		}
	}
	return 0, time.Time{}
}

func (s *Store) adminOpsDuplicateIdentityItemsLocked() []AdminOpsQueueItem {
	items := []AdminOpsQueueItem{}
	githubUsers := map[string][]*User{}
	walletUsers := map[string][]*User{}
	for _, user := range s.users {
		if user == nil {
			continue
		}
		if github := normalizeGitHubUsername(user.GitHubUsername); github != "" {
			githubUsers[github] = append(githubUsers[github], user)
		}
		if wallet := normalizeWalletAddress(user.WalletAddress); wallet != "" {
			walletUsers[wallet] = append(walletUsers[wallet], user)
		}
	}
	for github, users := range githubUsers {
		if len(users) < 2 {
			continue
		}
		reference := githubWorkerAccount(github)
		items = append(items, AdminOpsQueueItem{
			ID:        adminOpsItemID("fraud-github", github),
			Type:      "fraud_review",
			Severity:  "high",
			Title:     "Duplicate GitHub identity",
			Body:      fmt.Sprintf("%d user accounts share %s. Confirm ownership before paying this identity.", len(users), reference),
			UserID:    users[0].ID,
			Reference: reference,
			Status:    "duplicate_identity",
			Actions:   adminOpsActions("fraud_review", "", ""),
			CreatedAt: adminOpsLatestUserCreatedAt(users),
		})
	}
	for wallet, users := range walletUsers {
		if len(users) < 2 {
			continue
		}
		reference := walletAccount(wallet)
		items = append(items, AdminOpsQueueItem{
			ID:        adminOpsItemID("fraud-wallet", wallet),
			Type:      "fraud_review",
			Severity:  "high",
			Title:     "Duplicate wallet identity",
			Body:      fmt.Sprintf("%d user accounts share wallet %s. Confirm account ownership before releasing more MRG.", len(users), reference),
			UserID:    users[0].ID,
			Reference: reference,
			Status:    "duplicate_identity",
			Actions:   adminOpsActions("fraud_review", "", ""),
			CreatedAt: adminOpsLatestUserCreatedAt(users),
		})
	}
	return items
}

func adminOpsLatestUserCreatedAt(users []*User) time.Time {
	latest := time.Time{}
	for _, user := range users {
		if user == nil {
			continue
		}
		if latest.IsZero() || user.CreatedAt.After(latest) {
			latest = user.CreatedAt
		}
	}
	return latest
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
