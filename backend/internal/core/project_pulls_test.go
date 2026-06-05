package core

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
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

type fakeRecentProjectPullLister struct {
	recent        map[string][]AdminTaskPullRequest
	recentCalls   int
	perIssueCalls int
	err           error
}

func (f *fakeRecentProjectPullLister) listPullRequestsLinkedToIssue(_ context.Context, _ githubIssueTarget) ([]AdminTaskPullRequest, error) {
	f.perIssueCalls++
	return nil, errors.New("per-issue lookup should not be used by public recent monitor")
}

func (f *fakeRecentProjectPullLister) listRecentPullRequests(_ context.Context, target githubIssueTarget, _ int) ([]AdminTaskPullRequest, error) {
	f.recentCalls++
	if f.err != nil {
		return nil, f.err
	}
	rows := f.recent[target.fullName()]
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

func TestPublicProjectPullRequestsMonitorUsesRecentRepoSnapshot(t *testing.T) {
	now := time.Now().UTC()
	project := &Project{
		ID:           "prj_recent",
		Title:        "Recent PR proof",
		RepoProvider: "local-git",
		Tasks: []*Task{
			{
				ID:                 "tsk_10",
				ProjectID:          "prj_recent",
				IssueNumber:        10,
				Title:              "Wire public PR monitor",
				RewardCents:        50,
				RequiredWorkerKind: WorkerHuman,
				Status:             TaskOpen,
				IssueURL:           "https://github.com/mergeos-bounties/mergeos/issues/10",
				CreatedAt:          now.Add(-2 * time.Hour),
			},
			{
				ID:                 "tsk_11",
				ProjectID:          "prj_recent",
				IssueNumber:        11,
				Title:              "Show PR readiness",
				RewardCents:        50,
				RequiredWorkerKind: WorkerHuman,
				Status:             TaskOpen,
				IssueURL:           "https://github.com/mergeos-bounties/mergeos/issues/11",
				CreatedAt:          now.Add(-time.Hour),
			},
		},
		CreatedAt: now.Add(-3 * time.Hour),
	}
	lister := &fakeRecentProjectPullLister{
		recent: map[string][]AdminTaskPullRequest{
			"mergeos-bounties/mergeos": {
				{
					Number:         91,
					Title:          "Fix #10 public PR monitor",
					Body:           "Adds a recent snapshot for marketplace PR proof.",
					State:          "open",
					HTMLURL:        "https://github.com/mergeos-bounties/mergeos/pull/91",
					Author:         "builder",
					MergeableState: "clean",
					Labels:         []string{"evidence: provided", "star: verified"},
					CreatedAt:      now.Add(-40 * time.Minute),
					UpdatedAt:      now.Add(-30 * time.Minute),
				},
				{
					Number:         92,
					Title:          "Route PR readiness",
					Body:           "Fixes #11 with public readiness evidence.",
					State:          "open",
					HTMLURL:        "https://github.com/mergeos-bounties/mergeos/pull/92",
					Author:         "review-agent",
					MergeableState: "clean",
					Labels:         []string{"evidence: provided", "star: verified"},
					CreatedAt:      now.Add(-20 * time.Minute),
					UpdatedAt:      now.Add(-10 * time.Minute),
				},
			},
		},
	}

	payload := publicProjectPullRequestsMonitor(context.Background(), lister, project)
	if lister.recentCalls != 1 || lister.perIssueCalls != 0 {
		t.Fatalf("expected one repo snapshot call and no per-issue calls, got recent=%d per_issue=%d", lister.recentCalls, lister.perIssueCalls)
	}
	if payload.Stats.LinkedTaskCount != 2 || payload.Stats.PullRequestCount != 2 || payload.Stats.ErrorCount != 0 {
		t.Fatalf("unexpected public recent PR stats: %#v", payload.Stats)
	}
	if len(payload.Tasks) != 2 || payload.Tasks[0].MonitorStatus != "synced" || payload.Tasks[1].MonitorStatus != "synced" {
		t.Fatalf("expected synced public task rows: %#v", payload.Tasks)
	}
	if len(payload.Tasks[0].PullRequests) != 1 || payload.Tasks[0].PullRequests[0].Number != 91 {
		t.Fatalf("expected issue 10 to receive PR 91: %#v", payload.Tasks[0])
	}
	if len(payload.Tasks[1].PullRequests) != 1 || payload.Tasks[1].PullRequests[0].Number != 92 {
		t.Fatalf("expected issue 11 to receive PR 92: %#v", payload.Tasks[1])
	}
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal public recent PR monitor: %v", err)
	}
	if strings.Contains(string(body), `"task_id"`) || strings.Contains(string(body), "tsk_10") || strings.Contains(string(body), "tsk_11") {
		t.Fatalf("public recent PR monitor leaked internal task ids: %s", string(body))
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

func TestPublicProjectPullRequestsMonitorOmitsInternalTaskIDs(t *testing.T) {
	now := time.Now().UTC()
	project := &Project{
		ID:           "prj_public",
		Title:        "Public PR lane",
		RepoProvider: "local-git",
		Tasks: []*Task{
			{
				ID:                 "tsk_public",
				ProjectID:          "prj_public",
				IssueNumber:        21,
				Title:              "Expose public PR monitor",
				RewardCents:        75,
				RequiredWorkerKind: WorkerHuman,
				Status:             TaskOpen,
				IssueURL:           "https://github.com/mergeos-bounties/mergeos/issues/21",
				CreatedAt:          now,
			},
		},
		CreatedAt: now.Add(-time.Hour),
	}
	payload := publicProjectPullRequestsMonitor(context.Background(), fakeProjectPullLister{
		pulls: map[int][]AdminTaskPullRequest{
			21: {
				{
					Number:         42,
					Title:          "Add public PR protocol",
					State:          "open",
					HTMLURL:        "https://github.com/mergeos-bounties/mergeos/pull/42",
					Author:         "builder",
					MergeableState: "clean",
					Labels:         []string{"evidence: provided", "star: verified"},
					CreatedAt:      now,
					UpdatedAt:      now,
				},
			},
		},
	}, project)

	if payload.ProtocolVersion != "mergeos.pr-monitor.v1" || payload.Stats.PullRequestCount != 1 {
		t.Fatalf("unexpected public PR monitor payload: %#v", payload)
	}
	if len(payload.Tasks) != 1 || payload.Tasks[0].TaskID != "" {
		t.Fatalf("expected public task id to be omitted before serialization: %#v", payload.Tasks)
	}
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal public PR monitor: %v", err)
	}
	if strings.Contains(string(body), "tsk_public") || strings.Contains(string(body), `"task_id"`) {
		t.Fatalf("public PR monitor leaked internal task id: %s", string(body))
	}
}
