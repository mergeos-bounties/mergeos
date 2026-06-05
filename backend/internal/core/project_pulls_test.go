package core

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeProjectPullLister struct {
	pulls map[int][]AdminTaskPullRequest
	errs  map[int]error
}

func (f fakeProjectPullLister) listPullRequestsLinkedToIssue(_ context.Context, target githubIssueTarget) ([]AdminTaskPullRequest, error) {
	if err := f.errs[target.IssueNumber]; err != nil {
		return nil, err
	}
	rows := f.pulls[target.IssueNumber]
	copied := make([]AdminTaskPullRequest, len(rows))
	copy(copied, rows)
	return copied, nil
}

func TestProjectPullRequestsMonitorSummarizesReadinessWithoutAdminFields(t *testing.T) {
	now := time.Now().UTC()
	project := &Project{
		ID:           "prj_1",
		Title:        "Live PR proof",
		RepoProvider: "local-git",
		Tasks: []*Task{
			{
				ID:                 "tsk_1",
				ProjectID:          "prj_1",
				IssueNumber:        7,
				Title:              "Dashboard PR monitor",
				Acceptance:         "Show live pull requests.",
				RewardCents:        50,
				RequiredWorkerKind: WorkerHuman,
				Status:             TaskOpen,
				IssueURL:           "https://github.com/mergeos-bounties/mergeos/issues/7",
				CreatedAt:          now.Add(-2 * time.Hour),
			},
			{
				ID:                 "tsk_2",
				ProjectID:          "prj_1",
				IssueNumber:        8,
				Title:              "Unlinked backlog task",
				RewardCents:        25,
				RequiredWorkerKind: WorkerAgent,
				Status:             TaskOpen,
				CreatedAt:          now.Add(-time.Hour),
			},
		},
		CreatedAt: now.Add(-3 * time.Hour),
	}
	lister := fakeProjectPullLister{
		pulls: map[int][]AdminTaskPullRequest{
			7: {
				{
					Number:         12,
					Title:          "Add PR monitor",
					Body:           "private implementation detail",
					State:          "open",
					HTMLURL:        "https://github.com/mergeos-bounties/mergeos/pull/12",
					Author:         "builder",
					MergeableState: "clean",
					Labels:         []string{"evidence: provided", "star: verified"},
					ChangedFiles:   []AdminPullRequestFile{{Path: "frontend/src/App.vue", Additions: 20}},
					CreatedAt:      now.Add(-30 * time.Minute),
					UpdatedAt:      now.Add(-20 * time.Minute),
				},
				{
					Number:         13,
					Title:          "Delete workflow",
					State:          "open",
					HTMLURL:        "https://github.com/mergeos-bounties/mergeos/pull/13",
					Author:         "risky-builder",
					MergeableState: "dirty",
					Labels:         []string{"evidence: missing", "star: verified"},
					ChangedFiles:   []AdminPullRequestFile{{Path: ".github/workflows/deploy.yml", Status: "removed", Deletions: 40}},
					CreatedAt:      now.Add(-25 * time.Minute),
					UpdatedAt:      now.Add(-10 * time.Minute),
				},
			},
		},
	}

	payload := projectPullRequestsMonitor(context.Background(), lister, project)
	if payload.ProtocolVersion != "mergeos.pr-monitor.v1" || payload.Kind != "pr_monitor" {
		t.Fatalf("unexpected project PR protocol header: %#v", payload)
	}
	if payload.ProjectID != project.ID || payload.Stats.TaskCount != 2 || payload.Stats.LinkedTaskCount != 1 {
		t.Fatalf("unexpected project PR summary: %#v", payload)
	}
	if payload.Stats.PullRequestCount != 2 || payload.Stats.OpenPullRequestCount != 2 || payload.Stats.ReadyCount != 1 || payload.Stats.BlockedCount != 1 || payload.Stats.ErrorCount != 0 {
		t.Fatalf("unexpected project PR stats: %#v", payload.Stats)
	}
	if len(payload.Tasks) != 2 || payload.Tasks[0].MonitorStatus != "synced" || payload.Tasks[1].MonitorStatus != "unlinked" {
		t.Fatalf("unexpected task monitor rows: %#v", payload.Tasks)
	}
	if len(payload.Tasks[0].PullRequests) != 2 {
		t.Fatalf("missing pull request summaries: %#v", payload.Tasks[0])
	}
	if payload.Tasks[0].PullRequests[0].Readiness.Status != "ready" || !payload.Tasks[0].PullRequests[0].Readiness.CanMerge {
		t.Fatalf("expected first PR to be ready: %#v", payload.Tasks[0].PullRequests[0])
	}
	if payload.Tasks[0].PullRequests[1].Readiness.Status != "blocked" || payload.Tasks[0].PullRequests[1].Readiness.CanMerge {
		t.Fatalf("expected second PR to be blocked: %#v", payload.Tasks[0].PullRequests[1])
	}
}

func TestProjectPullRequestsMonitorSurfacesGitHubErrorsPerTask(t *testing.T) {
	now := time.Now().UTC()
	project := &Project{
		ID:           "prj_2",
		Title:        "Error proof",
		RepoProvider: "local-git",
		Tasks: []*Task{
			{
				ID:                 "tsk_error",
				ProjectID:          "prj_2",
				IssueNumber:        9,
				Title:              "Fetch PRs",
				RewardCents:        25,
				RequiredWorkerKind: WorkerHuman,
				Status:             TaskOpen,
				IssueURL:           "https://github.com/mergeos-bounties/mergeos/issues/9",
				CreatedAt:          now,
			},
		},
		CreatedAt: now,
	}
	payload := projectPullRequestsMonitor(context.Background(), fakeProjectPullLister{
		errs: map[int]error{9: errors.New("github unavailable")},
	}, project)
	if payload.Stats.ErrorCount != 1 || payload.Tasks[0].MonitorStatus != "error" || payload.Tasks[0].MonitorError != "github unavailable" {
		t.Fatalf("expected per-task github error: %#v", payload)
	}
}
