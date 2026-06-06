package core

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

type githubIssueRow struct {
	Number      int       `json:"number"`
	Title       string    `json:"title"`
	Body        string    `json:"body"`
	State       string    `json:"state"`
	HTMLURL     string    `json:"html_url"`
	Comments    int       `json:"comments"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	PullRequest *struct{} `json:"pull_request"`
	Labels      []struct {
		Name string `json:"name"`
	} `json:"labels"`
}

func ImportRepoIssues(ctx context.Context, cfg Config, req ImportRepoIssuesRequest) (*ImportRepoIssuesResponse, error) {
	owner, name, err := parseGitHubRepo(req.RepoURL)
	if err != nil {
		return nil, err
	}

	rows, err := fetchRepoIssueRows(ctx, cfg, owner, name)
	if err != nil {
		return nil, err
	}

	issues := make([]*ImportedRepoIssue, 0, len(rows))
	var total int64
	var totalHours float64
	for _, row := range rows {
		if row.PullRequest != nil {
			continue
		}
		issue := scoreRepoIssue(row)
		issues = append(issues, issue)
		total += issue.EstimatedCents
		totalHours += issue.EstimatedHours
	}
	sort.SliceStable(issues, func(i, j int) bool {
		if issues[i].Score != issues[j].Score {
			return issues[i].Score > issues[j].Score
		}
		return issues[i].UpdatedAt.After(issues[j].UpdatedAt)
	})

	return &ImportRepoIssuesResponse{
		ProtocolVersion:     "mergeos.repo-import.v1",
		Kind:                "repo_import",
		Owner:               owner,
		Name:                name,
		RepoURL:             "https://github.com/" + owner + "/" + name,
		IssueCount:          len(issues),
		TotalEstimatedCents: total,
		TotalEstimatedHours: roundHalfHour(totalHours),
		PlanningPacket:      repoImportPlanningPacket("https://github.com/"+owner+"/"+name, issues),
		Issues:              issues,
	}, nil
}

func repoImportPlanningPacket(repoURL string, issues []*ImportedRepoIssue) AIPlanningPacket {
	summary := AIPlanningSummary{
		IssueCount:          len(issues),
		TaskCount:           len(issues),
		TotalEstimatedHours: 0,
	}
	for _, issue := range issues {
		if issue == nil {
			continue
		}
		summary.TotalRewardCents += issue.EstimatedCents
		summary.TotalEstimatedHours += issue.EstimatedHours
		switch issue.RequiredWorkerKind {
		case WorkerAgent:
			summary.AgentTaskCount++
		case WorkerHybrid:
			summary.HybridTaskCount++
		default:
			summary.HumanTaskCount++
		}
	}
	summary.TotalEstimatedHours = roundHalfHour(summary.TotalEstimatedHours)
	contextURLs := map[string]string{
		"repo_import": "/api/public/repo/issues",
		"repository":  strings.TrimSpace(repoURL),
		"agent_queue": agentQueueEndpoint,
		"agents":      "/api/public/protocol/agents",
	}
	steps := []AIPlanningStep{
		aiPlanningStep("issue_scan", "Issue scan", "complete", "repo_issue_scan", "/api/public/repo/issues", "mergeos.repo-import.v1", "/protocol/repo-import.v1.schema.json"),
		aiPlanningStep("task_generation", "Task generation", "ready", "task_plan", "/api/projects/{id}/repo-sync", "mergeos.repo-sync.v1", "/protocol/repo-sync.v1.schema.json"),
		aiPlanningStep("reward_estimation", "Reward estimation", "ready", "reward_plan", "/api/projects/{id}/repo-sync", "mergeos.repo-sync.v1", "/protocol/repo-sync.v1.schema.json"),
		aiPlanningStep("contributor_routing", "Contributor routing", "queued", "routing_plan", "/api/projects/{id}/routing", "mergeos.routing.v1", "/protocol/routing.v1.schema.json"),
	}
	return AIPlanningPacket{
		Status:              planningStatus(summary.TaskCount, 0),
		SupervisorAgentType: ceoAgentType,
		ContextURLs:         contextURLs,
		Runbook: []AgentRunbookStep{
			{Step: 1, Action: "scan_issues", Label: "Score GitHub issues by risk, complexity, and work type", Method: "GET", Endpoint: "/api/public/repo/issues"},
			{Step: 2, Action: "generate_tasks", Label: "Convert issues into funded task candidates", Method: "POST", Endpoint: "/api/projects/{id}/repo-sync"},
			{Step: 3, Action: "estimate_rewards", Label: "Calibrate task rewards and estimated hours", Method: "POST", Endpoint: "/api/projects/{id}/repo-sync"},
			{Step: 4, Action: "route_contributors", Label: "Route each generated task to human, agent, or hybrid lanes", Method: "GET", Endpoint: "/api/projects/{id}/routing"},
		},
		Steps: steps,
		OutputContracts: []AgentOutputContract{
			{Action: "scan_issues", ArtifactKind: "repo_issue_scan", OutputEndpoint: "/api/public/repo/issues", OutputProtocol: "mergeos.repo-import.v1", OutputProtocolURL: "/protocol/repo-import.v1.schema.json", PublicURL: "/api/public/repo/issues"},
			{Action: "generate_tasks", ArtifactKind: "repo_sync", OutputEndpoint: "/api/projects/{id}/repo-sync", OutputProtocol: "mergeos.repo-sync.v1", OutputProtocolURL: "/protocol/repo-sync.v1.schema.json"},
			{Action: "route_contributors", ArtifactKind: "routing_plan", OutputEndpoint: "/api/projects/{id}/routing", OutputProtocol: "mergeos.routing.v1", OutputProtocolURL: "/protocol/routing.v1.schema.json"},
		},
		Summary: summary,
	}
}

func repoSyncPlanningPacket(projectID, repoURL string, mappings []ProjectIssueSyncMapping) AIPlanningPacket {
	summary := AIPlanningSummary{
		IssueCount: len(mappings),
		TaskCount:  len(mappings),
	}
	for _, mapping := range mappings {
		summary.TotalRewardCents += mapping.RewardCents
		summary.TotalEstimatedHours += mapping.EstimatedHours
		switch mapping.RequiredWorkerKind {
		case WorkerAgent:
			summary.AgentTaskCount++
		case WorkerHybrid:
			summary.HybridTaskCount++
		default:
			summary.HumanTaskCount++
		}
	}
	summary.TotalEstimatedHours = roundHalfHour(summary.TotalEstimatedHours)
	projectPath := "/api/projects/" + strings.TrimSpace(projectID)
	contextURLs := map[string]string{
		"repo_sync":       projectPath + "/repo-sync",
		"routing":         projectPath + "/routing",
		"workflow":        projectPath + "/protocol/workflow",
		"ai_workflow":     projectPath + "/ai-workflow",
		"agent_queue":     agentQueueEndpoint,
		"public_workflow": "/api/public/projects/" + strings.TrimSpace(projectID) + "/workflow",
	}
	if strings.TrimSpace(repoURL) != "" {
		contextURLs["repository"] = strings.TrimSpace(repoURL)
	}
	steps := []AIPlanningStep{
		aiPlanningStep("issue_scan", "Issue scan", "complete", "repo_issue_scan", "/api/public/repo/issues", "mergeos.repo-import.v1", "/protocol/repo-import.v1.schema.json"),
		aiPlanningStep("task_generation", "Task generation", "complete", "repo_sync", projectPath+"/repo-sync", "mergeos.repo-sync.v1", "/protocol/repo-sync.v1.schema.json"),
		aiPlanningStep("reward_estimation", "Reward estimation", "complete", "reward_plan", projectPath+"/repo-sync", "mergeos.repo-sync.v1", "/protocol/repo-sync.v1.schema.json"),
		aiPlanningStep("contributor_routing", "Contributor routing", "ready", "routing_plan", projectPath+"/routing", "mergeos.routing.v1", "/protocol/routing.v1.schema.json"),
	}
	return AIPlanningPacket{
		Status:              planningStatus(summary.TaskCount, summary.TaskCount),
		SupervisorAgentType: ceoAgentType,
		ContextURLs:         contextURLs,
		Runbook: []AgentRunbookStep{
			{Step: 1, Action: "review_generated_tasks", Label: "Review imported issues mapped to MergeOS tasks", Method: "POST", Endpoint: projectPath + "/repo-sync"},
			{Step: 2, Action: "inspect_rewards", Label: "Inspect reward estimates, hours, and worker lane mix", Method: "POST", Endpoint: projectPath + "/repo-sync"},
			{Step: 3, Action: "route_contributors", Label: "Open routing plan for contributors and agents", Method: "GET", Endpoint: projectPath + "/routing"},
			{Step: 4, Action: "publish_bounties", Label: "Publish ready task claims to marketplace and agent queue", Method: "GET", Endpoint: agentQueueEndpoint},
		},
		Steps: steps,
		OutputContracts: []AgentOutputContract{
			{Action: "generate_tasks", ArtifactKind: "repo_sync", OutputEndpoint: projectPath + "/repo-sync", OutputProtocol: "mergeos.repo-sync.v1", OutputProtocolURL: "/protocol/repo-sync.v1.schema.json"},
			{Action: "route_contributors", ArtifactKind: "routing_plan", OutputEndpoint: projectPath + "/routing", OutputProtocol: "mergeos.routing.v1", OutputProtocolURL: "/protocol/routing.v1.schema.json"},
			{Action: "publish_bounties", ArtifactKind: "marketplace_bounties", OutputEndpoint: "/api/marketplace", OutputProtocol: "mergeos.marketplace.v1", OutputProtocolURL: "/protocol/marketplace.v1.schema.json", PublicURL: "/marketplace"},
		},
		Summary: summary,
	}
}

func aiPlanningStep(id, title, status, artifactKind, endpoint, protocol, protocolURL string) AIPlanningStep {
	return AIPlanningStep{ID: id, Title: title, Status: status, ArtifactKind: artifactKind, OutputEndpoint: endpoint, OutputProtocol: protocol, OutputProtocolURL: protocolURL}
}

func planningStatus(taskCount, completedCount int) string {
	if taskCount <= 0 {
		return "waiting"
	}
	if completedCount >= taskCount {
		return "ready"
	}
	return "planning"
}

func fetchRepoIssueRows(ctx context.Context, cfg Config, owner, name string) ([]githubIssueRow, error) {
	client := &http.Client{Timeout: 20 * time.Second}
	rows := []githubIssueRow{}
	for page := 1; page <= 10; page++ {
		endpoint := fmt.Sprintf(
			"https://api.github.com/repos/%s/%s/issues?state=all&per_page=100&page=%d&sort=updated&direction=desc",
			url.PathEscape(owner),
			url.PathEscape(name),
			page,
		)
		httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
		if err != nil {
			return nil, err
		}
		httpReq.Header.Set("Accept", "application/vnd.github+json")
		httpReq.Header.Set("X-GitHub-Api-Version", "2022-11-28")
		if strings.TrimSpace(cfg.GitHubToken) != "" {
			httpReq.Header.Set("Authorization", "Bearer "+cfg.GitHubToken)
		}

		resp, err := client.Do(httpReq)
		if err != nil {
			return nil, err
		}
		var pageRows []githubIssueRow
		decodeErr := func() error {
			defer resp.Body.Close()
			if resp.StatusCode == http.StatusNotFound {
				return errors.New("repo was not found or is private; connect GitHub before importing private repos")
			}
			if resp.StatusCode < 200 || resp.StatusCode >= 300 {
				return fmt.Errorf("github issue import failed: %s", readBody(resp.Body))
			}
			return json.NewDecoder(resp.Body).Decode(&pageRows)
		}()
		if decodeErr != nil {
			return nil, decodeErr
		}
		rows = append(rows, pageRows...)
		if len(pageRows) < 100 {
			break
		}
	}
	return rows, nil
}

func parseGitHubRepo(value string) (string, string, error) {
	raw := strings.TrimSpace(value)
	raw = strings.TrimSuffix(raw, ".git")
	if raw == "" {
		return "", "", errors.New("repo url is required")
	}
	if strings.HasPrefix(raw, "git@github.com:") {
		raw = strings.TrimPrefix(raw, "git@github.com:")
		parts := strings.Split(strings.Trim(raw, "/"), "/")
		return cleanRepoParts(parts)
	}
	if strings.Contains(raw, "://") {
		parsed, err := url.Parse(raw)
		if err != nil {
			return "", "", errors.New("repo url is invalid")
		}
		if !strings.EqualFold(parsed.Hostname(), "github.com") {
			return "", "", errors.New("only GitHub repos are supported right now")
		}
		parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
		return cleanRepoParts(parts)
	}
	return cleanRepoParts(strings.Split(strings.Trim(raw, "/"), "/"))
}

func cleanRepoParts(parts []string) (string, string, error) {
	if len(parts) < 2 {
		return "", "", errors.New("repo url must look like owner/name")
	}
	owner := strings.TrimSpace(parts[0])
	name := strings.TrimSuffix(strings.TrimSpace(parts[1]), ".git")
	if owner == "" || name == "" {
		return "", "", errors.New("repo url must include owner and repo name")
	}
	return owner, name, nil
}

func scoreRepoIssue(row githubIssueRow) *ImportedRepoIssue {
	labels := make([]string, 0, len(row.Labels))
	for _, label := range row.Labels {
		if strings.TrimSpace(label.Name) != "" {
			labels = append(labels, label.Name)
		}
	}

	score := 25
	reasons := []string{"GitHub issue"}
	text := strings.ToLower(row.Title + " " + row.Body + " " + strings.Join(labels, " "))
	bodyLength := len(strings.TrimSpace(row.Body))

	if bodyLength > 1500 {
		score += 14
		reasons = append(reasons, "detailed issue body")
	} else if bodyLength > 500 {
		score += 8
		reasons = append(reasons, "clear reproduction context")
	}
	if row.Comments > 0 {
		added := row.Comments * 3
		if added > 15 {
			added = 15
		}
		score += added
		reasons = append(reasons, "active discussion")
	}

	applyKeywordScores(text, &score, &reasons)
	if score < 10 {
		score = 10
	}
	if score > 100 {
		score = 100
	}

	complexity := "low"
	if score >= 75 {
		complexity = "high"
	} else if score >= 45 {
		complexity = "medium"
	}

	estimated := int64(6000 + score*450)
	estimated = ((estimated + 999) / 1000) * 1000
	estimatedHours := estimatedIssueHours(score, complexity)
	kind, agent := workerForIssue(text)

	return &ImportedRepoIssue{
		Number:             row.Number,
		Title:              row.Title,
		State:              row.State,
		URL:                row.HTMLURL,
		Labels:             labels,
		Comments:           row.Comments,
		Score:              score,
		Complexity:         complexity,
		EstimatedCents:     estimated,
		EstimatedHours:     estimatedHours,
		RequiredWorkerKind: kind,
		SuggestedAgentType: agent,
		Reasons:            reasons,
		CreatedAt:          row.CreatedAt,
		UpdatedAt:          row.UpdatedAt,
	}
}

func estimatedIssueHours(score int, complexity string) float64 {
	hours := 2 + float64(score)/12
	switch strings.ToLower(strings.TrimSpace(complexity)) {
	case "high":
		hours += 4
	case "medium":
		hours += 1.5
	}
	if hours < 1 {
		hours = 1
	}
	if hours > 24 {
		hours = 24
	}
	return roundHalfHour(hours)
}

func roundHalfHour(value float64) float64 {
	if value <= 0 {
		return 0
	}
	return float64(int(value*2+0.5)) / 2
}

func applyKeywordScores(text string, score *int, reasons *[]string) {
	keywordScores := []struct {
		terms  []string
		points int
		reason string
	}{
		{[]string{"security", "auth", "token", "permission", "xss", "csrf"}, 18, "security or auth risk"},
		{[]string{"crash", "panic", "fatal", "data loss", "payment", "checkout"}, 16, "production risk"},
		{[]string{"bug", "regression", "broken", "error", "failing"}, 12, "bug fix"},
		{[]string{"api", "backend", "database", "migration", "webhook"}, 10, "backend surface"},
		{[]string{"frontend", "ui", "css", "responsive", "layout", "accessibility"}, 8, "frontend surface"},
		{[]string{"enhancement", "feature", "refactor"}, 6, "scope expansion"},
		{[]string{"documentation", "docs", "copy", "typo"}, -8, "small editorial task"},
		{[]string{"good first issue", "beginner", "easy"}, -10, "low complexity label"},
	}
	for _, item := range keywordScores {
		for _, term := range item.terms {
			if containsIssueTerm(text, term) {
				*score += item.points
				*reasons = append(*reasons, item.reason)
				break
			}
		}
	}
}

func containsIssueTerm(text, term string) bool {
	if strings.Contains(term, " ") || strings.ContainsAny(term, "-/_") {
		return strings.Contains(text, term)
	}
	for _, token := range strings.FieldsFunc(text, func(r rune) bool {
		return !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'))
	}) {
		if token == term {
			return true
		}
	}
	return false
}

func workerForIssue(text string) (WorkerKind, string) {
	if containsIssueTerm(text, "docs") || containsIssueTerm(text, "documentation") || containsIssueTerm(text, "copy") || containsIssueTerm(text, "typo") {
		return WorkerHuman, ""
	}
	if containsIssueTerm(text, "security") || containsIssueTerm(text, "auth") || containsIssueTerm(text, "payment") || containsIssueTerm(text, "checkout") {
		return WorkerHybrid, "security-review-agent"
	}
	if containsIssueTerm(text, "api") || containsIssueTerm(text, "backend") || containsIssueTerm(text, "database") || containsIssueTerm(text, "webhook") {
		return WorkerAgent, "backend-agent"
	}
	if containsIssueTerm(text, "ui") || containsIssueTerm(text, "css") || containsIssueTerm(text, "responsive") || containsIssueTerm(text, "layout") || containsIssueTerm(text, "frontend") {
		return WorkerAgent, "frontend-agent"
	}
	return WorkerHybrid, "repo-fix-agent"
}
