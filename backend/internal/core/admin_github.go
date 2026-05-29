package core

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

var closingIssueKeywordPattern = regexp.MustCompile(`(?i)\b(?:close[sd]?|fix(?:e[sd])?|resolve[sd]?)\s*:?\s+((?:[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+)?#\d+|https://github\.com/[^\s/]+/[^\s/]+/issues/\d+)`)

type githubIssueTarget struct {
	Owner       string
	Repo        string
	IssueNumber int
}

func (t githubIssueTarget) fullName() string {
	return t.Owner + "/" + t.Repo
}

type adminGitHubClient struct {
	token  string
	client *http.Client
}

func newAdminGitHubClient(cfg Config, requireToken bool) (*adminGitHubClient, error) {
	token := strings.TrimSpace(cfg.GitHubToken)
	if requireToken && token == "" {
		return nil, errors.New("GITHUB_TOKEN is required to merge pull requests")
	}
	return &adminGitHubClient{
		token: token,
		client: &http.Client{
			Timeout: 25 * time.Second,
		},
	}, nil
}

func (s *Server) adminTaskPullRequests(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	task, project, ok := s.store.TaskWithProject(r.PathValue("id"))
	if !ok {
		writeError(w, http.StatusNotFound, "task not found")
		return
	}
	target, err := githubIssueTargetForTask(task, project)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	client, err := newAdminGitHubClient(s.cfg, false)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	pulls, err := client.listPullRequestsLinkedToIssue(r.Context(), target)
	if err != nil {
		writeGitHubAdminError(w, err, http.StatusBadGateway)
		return
	}
	writeJSON(w, http.StatusOK, AdminTaskPullRequestsResponse{
		TaskID:       task.ID,
		IssueNumber:  target.IssueNumber,
		IssueURL:     task.IssueURL,
		Repository:   target.fullName(),
		PullRequests: pulls,
	})
}

func (s *Server) mergeAdminTaskPullRequest(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	var review AdminMergeTaskPullRequestRequest
	if err := json.NewDecoder(r.Body).Decode(&review); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	rewardMRG := selectedReviewRewardMRG(review)
	if rewardMRG <= 0 {
		writeError(w, http.StatusBadRequest, "reward_mrg is required")
		return
	}
	bountyType, err := normalizeAdminBountyType(review.BountyType)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	task, project, ok := s.store.TaskWithProject(r.PathValue("id"))
	if !ok {
		writeError(w, http.StatusNotFound, "task not found")
		return
	}
	pullNumber, err := strconv.Atoi(strings.TrimSpace(r.PathValue("number")))
	if err != nil || pullNumber <= 0 {
		writeError(w, http.StatusBadRequest, "pull request number is invalid")
		return
	}
	target, err := githubIssueTargetForTask(task, project)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	client, err := newAdminGitHubClient(s.cfg, true)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	pull, err := client.pullRequest(r.Context(), target, pullNumber)
	if err != nil {
		writeGitHubAdminError(w, err, http.StatusBadGateway)
		return
	}
	if pull.Draft {
		writeError(w, http.StatusConflict, "draft pull requests cannot be merged")
		return
	}
	var mergeSHA string
	if !pull.Merged {
		if !strings.EqualFold(pull.State, "open") {
			writeError(w, http.StatusConflict, "pull request is closed without being merged")
			return
		}
		if err := client.neutralizePullRequestClosingKeywords(r.Context(), target, pullNumber, pull.Body); err != nil {
			writeGitHubAdminError(w, fmt.Errorf("GitHub refused to prepare PR #%d for non-closing merge: %w", pullNumber, err), http.StatusConflict)
			return
		}
		mergeSHA, err = client.mergePullRequest(r.Context(), target, pullNumber)
		if err != nil {
			writeGitHubAdminError(w, fmt.Errorf("GitHub refused to merge PR #%d: %w", pullNumber, err), http.StatusConflict)
			return
		}
		if refreshed, err := client.pullRequest(r.Context(), target, pullNumber); err == nil {
			pull = refreshed
		}
	}

	req, err := acceptRequestForPullAuthor(task, pull.Author)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	accepted, err := s.store.AcceptTaskWithReviewReference(task.ID, req, rewardMRG, bountyType, buildPullLedgerReference(task.ID, pull.HTMLURL, pull.Title))
	if err != nil {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	if mergeSHA != "" && pull.MergeURL == "" {
		pull.MergeURL = githubCommitURL(target, mergeSHA)
	}
	adminURL := adminTasksURL(s.cfg)
	creditAccount, _ := s.store.TaskPayoutAccount(accepted.ID)
	creditURL := scanAccountURL(s.cfg, creditAccount)
	commentURL, commentErr := client.commentPullRequest(r.Context(), target, pullNumber, renderMergeOSPullComment(accepted, pull, req.WorkerID, rewardMRG, bountyType, creditURL))
	commentError := ""
	if commentErr != nil {
		commentError = commentErr.Error()
	}
	writeJSON(w, http.StatusOK, AdminMergeTaskPullRequestResponse{
		Task:         accepted,
		PullRequest:  pull,
		WorkerID:     req.WorkerID,
		RewardMRG:    rewardMRG,
		BountyType:   bountyType,
		AdminURL:     adminURL,
		CreditURL:    creditURL,
		CommentURL:   commentURL,
		CommentError: commentError,
	})
}

func selectedReviewRewardMRG(review AdminMergeTaskPullRequestRequest) int64 {
	if review.RewardMRG > 0 {
		return review.RewardMRG
	}
	return review.RewardCents
}

func normalizeAdminBountyType(value string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(value))
	switch normalized {
	case "future-small", "future-medium", "bug-large", "major-feature":
		return normalized, nil
	case "":
		return "", errors.New("bounty_type is required")
	default:
		return "", fmt.Errorf("unsupported bounty_type %q", value)
	}
}

func adminBountyTitle(value string) string {
	switch value {
	case "future-small":
		return "Future bounty - small"
	case "future-medium":
		return "Future bounty - medium"
	case "bug-large":
		return "Bug bounty - large"
	case "major-feature":
		return "Major feature bounty"
	default:
		return value
	}
}

func adminTasksURL(cfg Config) string {
	domain := strings.TrimSpace(cfg.AdminDomain)
	if domain == "" {
		return "https://uta.mergeos.shop/tasks"
	}
	domain = strings.TrimPrefix(strings.TrimPrefix(domain, "https://"), "http://")
	domain = strings.Trim(domain, "/")
	return "https://" + domain + "/tasks"
}

func (s *Server) syncAdminProjectIssues(ctx context.Context) {
	for _, project := range s.store.ListProjects("") {
		repoURL := projectSourceRepoURL(project)
		if repoURL == "" {
			continue
		}
		imported, err := ImportRepoIssues(ctx, s.cfg, ImportRepoIssuesRequest{RepoURL: repoURL})
		if err != nil || imported == nil {
			continue
		}
		_ = s.store.SyncProjectImportedIssues(project.ID, imported.Issues)
	}
}

func projectSourceRepoURL(project *Project) string {
	if project == nil {
		return ""
	}
	for _, task := range project.Tasks {
		if target, err := parseGitHubIssueURL(task.IssueURL); err == nil {
			return "https://github.com/" + target.fullName()
		}
	}
	if repoURL := sourceRepoURLFromBrief(project.Brief); repoURL != "" {
		return repoURL
	}
	for _, candidate := range []string{project.RepoURL, project.BountyRepoName} {
		owner, repo, err := parseGitHubRepo(candidate)
		if err == nil {
			return "https://github.com/" + owner + "/" + repo
		}
	}
	return ""
}

func sourceRepoURLFromBrief(brief string) string {
	for _, line := range strings.Split(brief, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(strings.ToLower(line), "source repository:") {
			continue
		}
		value := strings.TrimSpace(line[len("source repository:"):])
		owner, repo, err := parseGitHubRepo(value)
		if err != nil {
			return ""
		}
		return "https://github.com/" + owner + "/" + repo
	}
	return ""
}

func scanBaseURL(cfg Config) string {
	domain := strings.TrimSpace(cfg.ScanDomain)
	if domain == "" {
		return "https://scan.mergeos.shop"
	}
	domain = strings.TrimPrefix(strings.TrimPrefix(domain, "https://"), "http://")
	domain = strings.Trim(domain, "/")
	return "https://" + domain
}

func scanAccountURL(cfg Config, account string) string {
	account = strings.TrimSpace(account)
	if account == "" {
		return scanBaseURL(cfg) + "/"
	}
	return scanBaseURL(cfg) + "/address/" + url.PathEscape(account)
}

func httpURLOrFallback(value, fallback string) string {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(value, "https://") || strings.HasPrefix(value, "http://") {
		return value
	}
	return strings.TrimSpace(fallback)
}

func neutralizeClosingIssueKeywords(body string) (string, bool) {
	updated := closingIssueKeywordPattern.ReplaceAllString(body, "Related to $1")
	return updated, updated != body
}

func renderMergeOSPullComment(task *Task, pull AdminTaskPullRequest, workerID string, rewardMRG int64, bountyType string, creditURL string) string {
	mergeURL := httpURLOrFallback(pull.MergeURL, pull.HTMLURL)
	return fmt.Sprintf(`MergeOS approved and merged this PR.

- Merge URL: %s
- MRG credit URL: %s
- Credited worker: %s
- Bounty type: %s
- MRG credited: %d MRG
- Proof hash: %s
`, mergeURL, creditURL, workerID, adminBountyTitle(bountyType), rewardMRG, task.ProofHash)
}

func writeGitHubAdminError(w http.ResponseWriter, err error, fallbackStatus int) {
	status := fallbackStatus
	var apiErr githubAPIError
	if errors.As(err, &apiErr) {
		switch apiErr.StatusCode {
		case http.StatusForbidden:
			status = http.StatusForbidden
		case http.StatusNotFound:
			status = http.StatusNotFound
		case http.StatusConflict, http.StatusUnprocessableEntity, http.StatusMethodNotAllowed:
			status = http.StatusConflict
		case http.StatusUnauthorized:
			status = http.StatusUnauthorized
		}
	}
	writeError(w, status, err.Error())
}

func githubIssueTargetForTask(task *Task, project *Project) (githubIssueTarget, error) {
	if task == nil {
		return githubIssueTarget{}, errors.New("task not found")
	}
	if target, err := parseGitHubIssueURL(task.IssueURL); err == nil {
		return target, nil
	}
	if project != nil && strings.EqualFold(project.RepoProvider, "github") && task.IssueNumber > 0 {
		for _, candidate := range []string{project.RepoURL, project.BountyRepoName} {
			owner, repo, err := parseGitHubRepo(candidate)
			if err == nil {
				return githubIssueTarget{Owner: owner, Repo: repo, IssueNumber: task.IssueNumber}, nil
			}
		}
	}
	return githubIssueTarget{}, errors.New("task is not tied to a GitHub issue")
}

func parseGitHubIssueURL(value string) (githubIssueTarget, error) {
	raw := strings.TrimSpace(value)
	if raw == "" {
		return githubIssueTarget{}, errors.New("issue url is required")
	}
	parsed, err := url.Parse(raw)
	if err != nil || !strings.EqualFold(parsed.Hostname(), "github.com") {
		return githubIssueTarget{}, errors.New("issue url must be a GitHub URL")
	}
	parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	if len(parts) < 4 || !strings.EqualFold(parts[2], "issues") {
		return githubIssueTarget{}, errors.New("issue url must look like https://github.com/owner/repo/issues/123")
	}
	number, err := strconv.Atoi(parts[3])
	if err != nil || number <= 0 {
		return githubIssueTarget{}, errors.New("issue number is invalid")
	}
	owner, repo, err := cleanRepoParts(parts[:2])
	if err != nil {
		return githubIssueTarget{}, err
	}
	return githubIssueTarget{Owner: owner, Repo: repo, IssueNumber: number}, nil
}

func acceptRequestForPullAuthor(task *Task, author string) (AcceptTaskRequest, error) {
	workerID, err := githubWorkerID(author)
	if err != nil {
		return AcceptTaskRequest{}, err
	}
	req := AcceptTaskRequest{
		WorkerKind: task.RequiredWorkerKind,
		WorkerID:   workerID,
	}
	if req.WorkerKind != WorkerHuman {
		req.AgentType = strings.TrimSpace(task.SuggestedAgentType)
		if req.AgentType == "" {
			req.AgentType = "github-pr"
		}
	}
	return req, nil
}

func githubWorkerID(login string) (string, error) {
	login = strings.TrimPrefix(strings.TrimSpace(login), "@")
	if login == "" {
		return "", errors.New("pull request author is required")
	}
	return "github:" + login, nil
}

func (c *adminGitHubClient) listPullRequestsLinkedToIssue(ctx context.Context, target githubIssueTarget) ([]AdminTaskPullRequest, error) {
	seen := map[int]bool{}
	numbers := []int{}
	collect := func(number int) {
		if number <= 0 || seen[number] {
			return
		}
		seen[number] = true
		numbers = append(numbers, number)
	}

	var firstErr error
	if timelineNumbers, err := c.timelinePullNumbers(ctx, target); err == nil {
		for _, number := range timelineNumbers {
			collect(number)
		}
	} else {
		firstErr = err
	}
	if searchNumbers, err := c.searchPullNumbers(ctx, target); err == nil {
		for _, number := range searchNumbers {
			collect(number)
		}
	} else if firstErr == nil {
		firstErr = err
	}
	if len(numbers) == 0 && firstErr != nil {
		return nil, firstErr
	}

	pulls := make([]AdminTaskPullRequest, 0, len(numbers))
	for _, number := range numbers {
		pull, err := c.pullRequest(ctx, target, number)
		if err != nil {
			return nil, err
		}
		pulls = append(pulls, pull)
	}
	sort.SliceStable(pulls, func(i, j int) bool {
		return pulls[i].UpdatedAt.After(pulls[j].UpdatedAt)
	})
	return pulls, nil
}

func (c *adminGitHubClient) timelinePullNumbers(ctx context.Context, target githubIssueTarget) ([]int, error) {
	endpoint := fmt.Sprintf(
		"https://api.github.com/repos/%s/%s/issues/%d/timeline?per_page=100",
		url.PathEscape(target.Owner),
		url.PathEscape(target.Repo),
		target.IssueNumber,
	)
	var rows []githubTimelineEvent
	if err := c.githubJSON(ctx, http.MethodGet, endpoint, nil, &rows); err != nil {
		return nil, err
	}
	numbers := []int{}
	for _, row := range rows {
		if row.Source == nil {
			continue
		}
		if row.Source.PullRequest != nil && row.Source.PullRequest.Number > 0 {
			numbers = append(numbers, row.Source.PullRequest.Number)
			continue
		}
		if row.Source.Issue != nil && row.Source.Issue.PullRequest != nil {
			numbers = append(numbers, row.Source.Issue.Number)
		}
	}
	return numbers, nil
}

func (c *adminGitHubClient) searchPullNumbers(ctx context.Context, target githubIssueTarget) ([]int, error) {
	query := fmt.Sprintf("repo:%s/%s type:pr linked:issue #%d", target.Owner, target.Repo, target.IssueNumber)
	endpoint := "https://api.github.com/search/issues?q=" + url.QueryEscape(query) + "&per_page=50"
	var response githubIssueSearchResponse
	if err := c.githubJSON(ctx, http.MethodGet, endpoint, nil, &response); err != nil {
		return nil, err
	}
	numbers := []int{}
	for _, item := range response.Items {
		if item.PullRequest == nil {
			continue
		}
		numbers = append(numbers, item.Number)
	}
	return numbers, nil
}

func (c *adminGitHubClient) pullRequest(ctx context.Context, target githubIssueTarget, number int) (AdminTaskPullRequest, error) {
	endpoint := fmt.Sprintf(
		"https://api.github.com/repos/%s/%s/pulls/%d",
		url.PathEscape(target.Owner),
		url.PathEscape(target.Repo),
		number,
	)
	var row githubPullRequestRow
	if err := c.githubJSON(ctx, http.MethodGet, endpoint, nil, &row); err != nil {
		return AdminTaskPullRequest{}, err
	}
	return row.adminRow(target), nil
}

func (c *adminGitHubClient) mergePullRequest(ctx context.Context, target githubIssueTarget, number int) (string, error) {
	endpoint := fmt.Sprintf(
		"https://api.github.com/repos/%s/%s/pulls/%d/merge",
		url.PathEscape(target.Owner),
		url.PathEscape(target.Repo),
		number,
	)
	payload := map[string]any{
		"merge_method":   "squash",
		"commit_title":   fmt.Sprintf("Merge PR #%d through MergeOS admin", number),
		"commit_message": "Merged by MergeOS admin.\n\nReward and proof details are posted as a pull request comment. Linked issues remain open for bounty tracking.",
	}
	var result struct {
		Merged  bool   `json:"merged"`
		Message string `json:"message"`
		SHA     string `json:"sha"`
	}
	if err := c.githubJSON(ctx, http.MethodPut, endpoint, payload, &result); err != nil {
		return "", err
	}
	if !result.Merged {
		message := strings.TrimSpace(result.Message)
		if message == "" {
			message = "GitHub did not merge the pull request"
		}
		return "", errors.New(message)
	}
	return result.SHA, nil
}

func (c *adminGitHubClient) neutralizePullRequestClosingKeywords(ctx context.Context, target githubIssueTarget, number int, body string) error {
	updated, changed := neutralizeClosingIssueKeywords(body)
	if !changed {
		return nil
	}
	endpoint := fmt.Sprintf(
		"https://api.github.com/repos/%s/%s/pulls/%d",
		url.PathEscape(target.Owner),
		url.PathEscape(target.Repo),
		number,
	)
	return c.githubJSON(ctx, http.MethodPatch, endpoint, map[string]string{"body": updated}, nil)
}

func (c *adminGitHubClient) commentPullRequest(ctx context.Context, target githubIssueTarget, number int, body string) (string, error) {
	endpoint := fmt.Sprintf(
		"https://api.github.com/repos/%s/%s/issues/%d/comments",
		url.PathEscape(target.Owner),
		url.PathEscape(target.Repo),
		number,
	)
	var result struct {
		HTMLURL string `json:"html_url"`
	}
	if err := c.githubJSON(ctx, http.MethodPost, endpoint, map[string]string{"body": body}, &result); err != nil {
		return "", err
	}
	return result.HTMLURL, nil
}

func githubCommitURL(target githubIssueTarget, sha string) string {
	sha = strings.TrimSpace(sha)
	if sha == "" {
		return ""
	}
	return "https://github.com/" + target.fullName() + "/commit/" + sha
}

func (c *adminGitHubClient) githubJSON(ctx context.Context, method, endpoint string, body any, out any) error {
	var payload bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&payload).Encode(body); err != nil {
			return err
		}
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint, &payload)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return githubAPIError{StatusCode: resp.StatusCode, Body: readBody(resp.Body)}
	}
	if out == nil {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

type githubAPIError struct {
	StatusCode int
	Body       string
}

func (e githubAPIError) Error() string {
	body := strings.TrimSpace(e.Body)
	if body == "" {
		return fmt.Sprintf("github request failed with status %d", e.StatusCode)
	}
	var decoded struct {
		Message string `json:"message"`
		Errors  []struct {
			Message string `json:"message"`
			Code    string `json:"code"`
			Field   string `json:"field"`
		} `json:"errors"`
	}
	if err := json.Unmarshal([]byte(body), &decoded); err == nil {
		parts := []string{}
		if decoded.Message != "" {
			parts = append(parts, decoded.Message)
		}
		for _, item := range decoded.Errors {
			detail := strings.TrimSpace(item.Message)
			if detail == "" {
				detail = strings.TrimSpace(strings.Join([]string{item.Field, item.Code}, " "))
			}
			if detail != "" {
				parts = append(parts, detail)
			}
		}
		if len(parts) > 0 {
			return fmt.Sprintf("github request failed (%d): %s", e.StatusCode, strings.Join(parts, "; "))
		}
	}
	return fmt.Sprintf("github request failed (%d): %s", e.StatusCode, body)
}

type githubTimelineEvent struct {
	Event  string `json:"event"`
	Source *struct {
		Type        string             `json:"type"`
		Issue       *githubLinkedIssue `json:"issue"`
		PullRequest *githubLinkedIssue `json:"pull_request"`
	} `json:"source"`
}

type githubIssueSearchResponse struct {
	Items []githubLinkedIssue `json:"items"`
}

type githubLinkedIssue struct {
	Number      int         `json:"number"`
	HTMLURL     string      `json:"html_url"`
	PullRequest interface{} `json:"pull_request"`
}

type githubPullRequestRow struct {
	Number         int        `json:"number"`
	Title          string     `json:"title"`
	Body           string     `json:"body"`
	State          string     `json:"state"`
	HTMLURL        string     `json:"html_url"`
	MergeCommitSHA string     `json:"merge_commit_sha"`
	Draft          bool       `json:"draft"`
	Merged         bool       `json:"merged"`
	MergeableState string     `json:"mergeable_state"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	MergedAt       *time.Time `json:"merged_at"`
	User           *struct {
		Login string `json:"login"`
	} `json:"user"`
	Base *struct {
		Ref string `json:"ref"`
	} `json:"base"`
	Head *struct {
		Ref string `json:"ref"`
	} `json:"head"`
}

func (row githubPullRequestRow) adminRow(target githubIssueTarget) AdminTaskPullRequest {
	result := AdminTaskPullRequest{
		Number:         row.Number,
		Title:          row.Title,
		Body:           row.Body,
		State:          row.State,
		HTMLURL:        row.HTMLURL,
		MergeURL:       githubCommitURL(target, row.MergeCommitSHA),
		Draft:          row.Draft,
		Merged:         row.Merged,
		MergeableState: row.MergeableState,
		CreatedAt:      row.CreatedAt,
		UpdatedAt:      row.UpdatedAt,
		MergedAt:       row.MergedAt,
	}
	if row.User != nil {
		result.Author = row.User.Login
	}
	if row.Base != nil {
		result.BaseRef = row.Base.Ref
	}
	if row.Head != nil {
		result.HeadRef = row.Head.Ref
	}
	return result
}
