package core

import (
	"testing"
	"time"
)

func TestParseGitHubRepo(t *testing.T) {
	tests := map[string]string{
		"https://github.com/openai/codex": "openai/codex",
		"git@github.com:owner/app.git":    "owner/app",
		"mergeos/platform":                "mergeos/platform",
	}
	for input, expected := range tests {
		owner, repo, err := parseGitHubRepo(input)
		if err != nil {
			t.Fatalf("parseGitHubRepo(%q): %v", input, err)
		}
		got := owner + "/" + repo
		if got != expected {
			t.Fatalf("parseGitHubRepo(%q) = %q, want %q", input, got, expected)
		}
	}
}

func TestScoreRepoIssue(t *testing.T) {
	issue := scoreRepoIssue(githubIssueRow{
		Number:    42,
		Title:     "Payment checkout crashes after auth token refresh",
		Body:      "Users report a production crash during checkout after auth token refresh. Needs backend and frontend verification.",
		State:     "open",
		HTMLURL:   "https://github.com/acme/app/issues/42",
		Comments:  4,
		CreatedAt: time.Now().Add(-24 * time.Hour),
		UpdatedAt: time.Now(),
		Labels: []struct {
			Name string `json:"name"`
		}{
			{Name: "bug"},
			{Name: "checkout"},
		},
	})
	if issue.Score < 75 {
		t.Fatalf("score = %d, want high-priority score", issue.Score)
	}
	if issue.RequiredWorkerKind != WorkerHybrid {
		t.Fatalf("worker kind = %q, want %q", issue.RequiredWorkerKind, WorkerHybrid)
	}
	if issue.EstimatedCents <= 0 {
		t.Fatalf("estimated cents = %d", issue.EstimatedCents)
	}
	if issue.EstimatedHours <= 0 {
		t.Fatalf("estimated hours = %.1f", issue.EstimatedHours)
	}
}
