package core

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type RepoResult struct {
	Provider  string
	Name      string
	URL       string
	LocalPath string
	Issues    map[string]RepoIssue
}

type RepoIssue struct {
	Number int
	URL    string
}

type RepoFactory interface {
	CreateProjectRepo(ctx context.Context, project *Project, tasks []*Task) (*RepoResult, error)
}

func NewRepoFactory(cfg Config) RepoFactory {
	if cfg.GitHubReady() {
		return &GitHubRepoFactory{
			cfg: cfg,
			client: &http.Client{
				Timeout: 25 * time.Second,
			},
		}
	}
	return LocalRepoFactory{cfg: cfg}
}

type LocalRepoFactory struct {
	cfg Config
}

func (f LocalRepoFactory) CreateProjectRepo(_ context.Context, project *Project, tasks []*Task) (*RepoResult, error) {
	root, err := filepath.Abs(f.cfg.BountyRoot)
	if err != nil {
		return nil, err
	}
	owner := strings.TrimSpace(f.cfg.GitHubOwner)
	if owner == "" {
		owner = defaultGitHubOwner
	}
	repoSlug := fmt.Sprintf("%s-%s", slug(project.ClientName), project.ID)
	repoPath := filepath.Join(root, repoSlug)
	if err := os.MkdirAll(filepath.Join(repoPath, "tasks"), 0755); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Join(repoPath, "attachments"), 0755); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Join(repoPath, ".mergeos"), 0755); err != nil {
		return nil, err
	}

	if err := os.WriteFile(filepath.Join(repoPath, "README.md"), []byte(renderRepoReadme(project, tasks)), 0644); err != nil {
		return nil, err
	}
	for _, task := range tasks {
		fileName := fmt.Sprintf("%03d-%s.md", task.IssueNumber, slug(task.Title))
		taskPath := filepath.Join(repoPath, "tasks", fileName)
		if err := os.WriteFile(taskPath, []byte(renderTaskMarkdown(project, task)), 0644); err != nil {
			return nil, err
		}
	}
	for _, attachment := range project.Attachments {
		if attachment.StoredPath == "" {
			continue
		}
		attachmentName := attachmentRepoName(attachment)
		targetPath := filepath.Join(repoPath, "attachments", attachmentName)
		if err := copyFile(attachment.StoredPath, targetPath); err != nil {
			return nil, err
		}
	}

	manifest, err := json.MarshalIndent(project, "", "  ")
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(filepath.Join(repoPath, ".mergeos", "project.json"), manifest, 0644); err != nil {
		return nil, err
	}

	initLocalGit(repoPath)

	issues := map[string]RepoIssue{}
	for _, task := range tasks {
		issuePath := filepath.Join(repoPath, "tasks", fmt.Sprintf("%03d-%s.md", task.IssueNumber, slug(task.Title)))
		issues[task.ID] = RepoIssue{
			Number: task.IssueNumber,
			URL:    issuePath,
		}
	}

	return &RepoResult{
		Provider:  "local-git",
		Name:      owner + "/" + repoSlug,
		URL:       repoPath,
		LocalPath: repoPath,
		Issues:    issues,
	}, nil
}

func initLocalGit(repoPath string) {
	if _, err := os.Stat(filepath.Join(repoPath, ".git")); err == nil {
		return
	}
	commands := [][]string{
		{"git", "init"},
		{"git", "add", "."},
		{"git", "-c", "user.name=MergeOS", "-c", "user.email=mergeos@local", "commit", "-m", "Initialize MergeOS bounty repo"},
	}
	for _, parts := range commands {
		cmd := exec.Command(parts[0], parts[1:]...)
		cmd.Dir = repoPath
		_ = cmd.Run()
	}
}

type GitHubRepoFactory struct {
	cfg    Config
	client *http.Client
}

func (f *GitHubRepoFactory) CreateProjectRepo(ctx context.Context, project *Project, tasks []*Task) (*RepoResult, error) {
	repoName := fmt.Sprintf("%s-%s", slug(project.ClientName), project.ID)
	createURL := "https://api.github.com/user/repos"
	if f.cfg.GitHubOwnerType == "org" {
		createURL = "https://api.github.com/orgs/" + f.cfg.GitHubOwner + "/repos"
	}
	repoPayload := map[string]any{
		"name":        repoName,
		"private":     true,
		"has_issues":  true,
		"description": "MergeOS child bounty repo for " + project.Title,
	}
	var created struct {
		Name     string `json:"name"`
		FullName string `json:"full_name"`
		HTMLURL  string `json:"html_url"`
	}
	if err := f.githubJSON(ctx, http.MethodPost, createURL, repoPayload, &created); err != nil {
		return nil, err
	}
	if created.Name == "" {
		created.Name = repoName
	}
	if created.FullName == "" {
		created.FullName = f.cfg.GitHubOwner + "/" + created.Name
	}

	readmePayload := map[string]any{
		"message": "Initialize MergeOS bounty repo",
		"content": base64.StdEncoding.EncodeToString([]byte(renderRepoReadme(project, tasks))),
	}
	contentsURL := "https://api.github.com/repos/" + created.FullName + "/contents/README.md"
	if err := f.githubJSON(ctx, http.MethodPut, contentsURL, readmePayload, nil); err != nil {
		return nil, err
	}
	for _, attachment := range project.Attachments {
		if attachment.StoredPath == "" {
			continue
		}
		data, err := os.ReadFile(attachment.StoredPath)
		if err != nil {
			return nil, err
		}
		attachmentPayload := map[string]any{
			"message": "Add client attachment " + attachment.OriginalName,
			"content": base64.StdEncoding.EncodeToString(data),
		}
		attachmentURL := "https://api.github.com/repos/" + created.FullName + "/contents/attachments/" + attachmentRepoName(attachment)
		if err := f.githubJSON(ctx, http.MethodPut, attachmentURL, attachmentPayload, nil); err != nil {
			return nil, err
		}
	}

	issues := map[string]RepoIssue{}
	for _, task := range tasks {
		issuePayload := map[string]any{
			"title": task.Title,
			"body":  renderTaskMarkdown(project, task),
			"labels": []string{
				"mergeos",
				"worker:" + string(task.RequiredWorkerKind),
			},
		}
		var issue struct {
			Number  int    `json:"number"`
			HTMLURL string `json:"html_url"`
		}
		issueURL := "https://api.github.com/repos/" + created.FullName + "/issues"
		if err := f.githubJSON(ctx, http.MethodPost, issueURL, issuePayload, &issue); err != nil {
			return nil, err
		}
		issues[task.ID] = RepoIssue{
			Number: issue.Number,
			URL:    issue.HTMLURL,
		}
	}

	return &RepoResult{
		Provider: "github",
		Name:     created.FullName,
		URL:      created.HTMLURL,
		Issues:   issues,
	}, nil
}

func (f *GitHubRepoFactory) githubJSON(ctx context.Context, method, endpoint string, body any, out any) error {
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
	req.Header.Set("Authorization", "Bearer "+f.cfg.GitHubToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := f.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("github request failed: %s", readBody(resp.Body))
	}
	if out == nil {
		return nil
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func renderRepoReadme(project *Project, tasks []*Task) string {
	var builder strings.Builder
	builder.WriteString("# " + project.Title + "\n\n")
	builder.WriteString("Private child bounty repo generated by MergeOS.\n\n")
	builder.WriteString("Client: " + project.ClientName + "\n\n")
	if project.CompanyName != "" {
		builder.WriteString("Company: " + project.CompanyName + "\n\n")
	}
	builder.WriteString("Contact email: " + project.ClientEmail + "\n\n")
	if project.Timeline != "" {
		builder.WriteString("Timeline: " + project.Timeline + "\n\n")
	}
	if project.PackageTier != "" {
		builder.WriteString("Package: " + project.PackageTier + "\n\n")
	}
	builder.WriteString("Budget: " + centsToPayPalValue(project.BudgetCents) + " USD\n\n")
	builder.WriteString("## Brief\n\n")
	builder.WriteString(project.Brief + "\n\n")
	if len(project.Attachments) > 0 {
		builder.WriteString("## Client Attachments\n\n")
		for _, attachment := range project.Attachments {
			builder.WriteString(fmt.Sprintf("- [%s](attachments/%s) - %s - %d bytes\n", attachment.OriginalName, attachmentRepoName(attachment), attachment.ContentType, attachment.SizeBytes))
		}
		builder.WriteString("\n")
	}
	builder.WriteString("## Bounty Tasks\n\n")
	for _, task := range tasks {
		builder.WriteString(fmt.Sprintf("- #%d %s - %s - %s MERGE\n", task.IssueNumber, task.Title, task.RequiredWorkerKind, centsToPayPalValue(task.RewardCents)))
	}
	return builder.String()
}

func renderTaskMarkdown(project *Project, task *Task) string {
	var builder strings.Builder
	builder.WriteString("## MergeOS Task\n\n")
	builder.WriteString("Project: " + project.Title + "\n\n")
	builder.WriteString("Acceptance: " + task.Acceptance + "\n\n")
	builder.WriteString("Required worker kind: `" + string(task.RequiredWorkerKind) + "`\n\n")
	if task.SuggestedAgentType != "" {
		builder.WriteString("Suggested agent type: `" + task.SuggestedAgentType + "`\n\n")
	}
	if len(project.Attachments) > 0 {
		builder.WriteString("Client attachments are available in the repo `attachments/` directory and should be used as visual/content references.\n\n")
	}
	builder.WriteString("Reward: " + centsToPayPalValue(task.RewardCents) + " MERGE\n\n")
	builder.WriteString("A paid submission must include a worker manifest with worker kind, worker id, and agent type when the work is agent or hybrid.\n")
	return builder.String()
}

var _ RepoFactory = LocalRepoFactory{}
var _ RepoFactory = (*GitHubRepoFactory)(nil)

func attachmentRepoName(attachment *Attachment) string {
	name := slug(strings.TrimSuffix(attachment.OriginalName, filepath.Ext(attachment.OriginalName)))
	if name == "" {
		name = attachment.ID
	}
	ext := strings.ToLower(filepath.Ext(attachment.OriginalName))
	if ext == "" {
		ext = filepath.Ext(attachment.StoredName)
	}
	return attachment.ID + "-" + name + ext
}

func copyFile(sourcePath, targetPath string) error {
	data, err := os.ReadFile(sourcePath)
	if err != nil {
		return err
	}
	return os.WriteFile(targetPath, data, 0644)
}
