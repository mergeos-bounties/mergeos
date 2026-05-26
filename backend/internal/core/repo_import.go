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
	for _, row := range rows {
		if row.PullRequest != nil {
			continue
		}
		issue := scoreRepoIssue(row)
		issues = append(issues, issue)
		total += issue.EstimatedCents
	}
	sort.SliceStable(issues, func(i, j int) bool {
		if issues[i].Score != issues[j].Score {
			return issues[i].Score > issues[j].Score
		}
		return issues[i].UpdatedAt.After(issues[j].UpdatedAt)
	})

	return &ImportRepoIssuesResponse{
		Owner:               owner,
		Name:                name,
		RepoURL:             "https://github.com/" + owner + "/" + name,
		IssueCount:          len(issues),
		TotalEstimatedCents: total,
		Issues:              issues,
	}, nil
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
		RequiredWorkerKind: kind,
		SuggestedAgentType: agent,
		Reasons:            reasons,
		CreatedAt:          row.CreatedAt,
		UpdatedAt:          row.UpdatedAt,
	}
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
