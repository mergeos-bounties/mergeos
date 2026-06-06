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
	if payload.Stats.PullRequestCount != 2 || payload.Stats.OpenPullRequestCount != 2 || payload.Stats.ReadyCount != 1 || payload.Stats.BlockedCount != 1 || payload.Stats.ErrorCount != 0 || payload.Stats.AutoReleaseReadyCount != 1 {
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
	if payload.Tasks[0].RewardCents != 50 || payload.Tasks[0].WorkerKind != WorkerHuman {
		t.Fatalf("expected task release metadata: %#v", payload.Tasks[0])
	}
	if payload.Tasks[0].ReviewPacket == nil || payload.Tasks[0].ReviewPacket["review_endpoint"] != "/api/projects/prj_1/agent-actions" {
		t.Fatalf("expected PR review packet: %#v", payload.Tasks[0].ReviewPacket)
	}
	reviewPayload, ok := payload.Tasks[0].ReviewPacket["payload"].(AgentActionRequest)
	if !ok {
		t.Fatalf("expected review packet agent action payload: %#v", payload.Tasks[0].ReviewPacket)
	}
	if reviewPayload.Action != "review" || reviewPayload.ClaimID != "tsk_1" || reviewPayload.BountyID != "prj_1:7" || reviewPayload.PullNumber != 13 || reviewPayload.ReferenceURL != "https://github.com/mergeos-bounties/mergeos/pull/13" {
		t.Fatalf("unexpected review packet payload: %#v", reviewPayload)
	}
	if reviewPayload.AgentType != "review-agent" || len(reviewPayload.Runbook) < 3 || !containsString(reviewPayload.ContextURLs, "/api/projects/prj_1/pull-requests") {
		t.Fatalf("review packet missing agent context: %#v", reviewPayload)
	}
	if payload.Tasks[0].ReviewPacket["status"] != "blocked" || len(reviewPayload.Checks) == 0 {
		t.Fatalf("expected latest review packet status to carry blocker checks: %#v", payload.Tasks[0].ReviewPacket)
	}
	if payload.Tasks[0].ReleasePacket == nil || payload.Tasks[0].ReleasePacket["can_release"] != true {
		t.Fatalf("expected single release packet: %#v", payload.Tasks[0].ReleasePacket)
	}
	if payload.Tasks[0].AutoReleasePacket == nil || payload.Tasks[0].AutoReleasePacket["can_auto_release"] != true {
		t.Fatalf("expected auto-release packet: %#v", payload.Tasks[0].AutoReleasePacket)
	}
	payloadMap, ok := payload.Tasks[0].AutoReleasePacket["payload"].(map[string]any)
	if !ok {
		t.Fatalf("expected auto-release payload map: %#v", payload.Tasks[0].AutoReleasePacket)
	}
	if payloadMap["policy"] != defaultAutoReleasePolicy {
		t.Fatalf("unexpected auto-release policy: %#v", payloadMap)
	}
	candidates, ok := payloadMap["candidates"].([]ProjectAutoReleaseCandidate)
	if !ok || len(candidates) != 1 {
		t.Fatalf("unexpected auto-release candidates: %#v", payloadMap["candidates"])
	}
	if candidates[0].TaskID != "tsk_1" || candidates[0].WorkerID != "github:builder" || candidates[0].PullRequestURL != "https://github.com/mergeos-bounties/mergeos/pull/12" {
		t.Fatalf("unexpected auto-release candidate: %#v", candidates[0])
	}
	if candidates[0].RewardCents != 50 || candidates[0].Repository != "mergeos-bounties/mergeos" || candidates[0].PullRequestNumber != 12 || candidates[0].ReadinessStatus != "ready" || !candidates[0].CanMerge || candidates[0].RiskLevel != "low" || candidates[0].Draft || !candidates[0].CanRelease {
		t.Fatalf("unexpected auto-release release gate: %#v", candidates[0])
	}
}

func TestProjectPullRequestsMonitorRequiresDeploymentValidationForAutoReleasePacket(t *testing.T) {
	now := time.Now().UTC()
	project := &Project{
		ID:           "prj_deploy",
		Title:        "Deployment proof",
		RepoProvider: "local-git",
		Tasks: []*Task{
			{
				ID:                 "tsk_deploy",
				ProjectID:          "prj_deploy",
				IssueNumber:        31,
				Title:              "Deployment handoff",
				Acceptance:         "Preview rollout must be validated before completion.",
				RewardCents:        80,
				RequiredWorkerKind: WorkerHuman,
				Status:             TaskOpen,
				IssueURL:           "https://github.com/mergeos-bounties/mergeos/issues/31",
				CreatedAt:          now.Add(-2 * time.Hour),
			},
		},
		CreatedAt: now.Add(-3 * time.Hour),
	}

	unverified := projectPullRequestsMonitor(context.Background(), fakeProjectPullLister{
		pulls: map[int][]AdminTaskPullRequest{
			31: {
				{
					Number:         310,
					Title:          "Deploy preview",
					State:          "open",
					HTMLURL:        "https://github.com/mergeos-bounties/mergeos/pull/310",
					Author:         "builder",
					MergeableState: "clean",
					Labels:         []string{"evidence: provided", "star: verified"},
					CreatedAt:      now.Add(-30 * time.Minute),
					UpdatedAt:      now.Add(-20 * time.Minute),
				},
			},
		},
	}, project)
	if unverified.Stats.AutoReleaseReadyCount != 0 || unverified.Tasks[0].AutoReleasePacket != nil {
		t.Fatalf("unverified deployment PR should not expose auto-release: %#v", unverified.Tasks[0])
	}
	if unverified.Tasks[0].PullRequests[0].Readiness.Status != "needs_review" {
		t.Fatalf("deployment PR should need review without validation: %#v", unverified.Tasks[0].PullRequests[0].Readiness)
	}

	verified := projectPullRequestsMonitor(context.Background(), fakeProjectPullLister{
		pulls: map[int][]AdminTaskPullRequest{
			31: {
				{
					Number:         311,
					Title:          "Deploy preview",
					State:          "open",
					HTMLURL:        "https://github.com/mergeos-bounties/mergeos/pull/311",
					Author:         "builder",
					MergeableState: "clean",
					Labels:         []string{"evidence: provided", "star: verified", "deployment: verified"},
					CreatedAt:      now.Add(-10 * time.Minute),
					UpdatedAt:      now.Add(-5 * time.Minute),
				},
			},
		},
	}, project)
	if verified.Stats.AutoReleaseReadyCount != 1 || verified.Tasks[0].AutoReleasePacket == nil {
		t.Fatalf("verified deployment PR should expose auto-release: %#v", verified.Tasks[0])
	}
	payloadMap, ok := verified.Tasks[0].AutoReleasePacket["payload"].(map[string]any)
	if !ok {
		t.Fatalf("expected verified auto-release payload map: %#v", verified.Tasks[0].AutoReleasePacket)
	}
	candidates, ok := payloadMap["candidates"].([]ProjectAutoReleaseCandidate)
	if !ok || len(candidates) != 1 {
		t.Fatalf("unexpected verified auto-release candidates: %#v", payloadMap["candidates"])
	}
	if candidates[0].DeploymentStatus != "validated" || !containsString(candidates[0].ValidationSignals, "deployment: verified") {
		t.Fatalf("candidate missing deployment validation proof: %#v", candidates[0])
	}
}

func TestAutoReleaseAcceptRequestRequiresDeploymentValidationForDeployTask(t *testing.T) {
	task := &Task{
		ID:                 "tsk_deploy_gate",
		Title:              "Deployment handoff",
		Acceptance:         "Preview rollout must be validated before completion.",
		RewardCents:        120,
		RequiredWorkerKind: WorkerHuman,
	}
	candidate := ProjectAutoReleaseCandidate{
		TaskID:            task.ID,
		WorkerKind:        WorkerHuman,
		WorkerID:          "github:builder",
		RewardCents:       task.RewardCents,
		PullRequestNumber: 411,
		PullRequestURL:    "https://github.com/mergeos-bounties/mergeos/pull/411",
		PullRequestTitle:  "Deploy handoff",
		ReadinessStatus:   "ready",
		CanMerge:          true,
		RiskLevel:         "low",
		CanRelease:        true,
	}

	if _, _, err := autoReleaseAcceptRequest(task, candidate, defaultAutoReleasePolicy); err == nil || !strings.Contains(err.Error(), "deployment validation") {
		t.Fatalf("expected missing deployment validation to block auto-release, got %v", err)
	}

	candidate.DeploymentStatus = "validated"
	req, reference, err := autoReleaseAcceptRequest(task, candidate, defaultAutoReleasePolicy)
	if err != nil {
		t.Fatalf("validated deployment candidate should pass: %v", err)
	}
	if req.WorkerID != "github:builder" || req.WorkerKind != WorkerHuman {
		t.Fatalf("unexpected accept request: %#v", req)
	}
	if !strings.Contains(reference, "deployment_validation:validated") || !strings.Contains(reference, "auto_release:"+defaultAutoReleasePolicy) {
		t.Fatalf("reference missing deployment proof: %s", reference)
	}
}

func TestAutoReleaseAcceptRequestRequiresDeploymentValidationForCandidateSignal(t *testing.T) {
	task := &Task{
		ID:                 "tsk_signal_gate",
		Title:              "Runtime handoff",
		Acceptance:         "Attach acceptance evidence.",
		RewardCents:        90,
		RequiredWorkerKind: WorkerHuman,
	}
	candidate := ProjectAutoReleaseCandidate{
		TaskID:            task.ID,
		WorkerKind:        WorkerHuman,
		WorkerID:          "github:builder",
		RewardCents:       task.RewardCents,
		PullRequestNumber: 412,
		PullRequestURL:    "https://github.com/mergeos-bounties/mergeos/pull/412",
		PullRequestTitle:  "Update preview workflow",
		ReadinessStatus:   "ready",
		CanMerge:          true,
		RiskLevel:         "low",
		ValidationSignals: []string{"evidence: provided", "star: verified", "deployment-sensitive"},
		CanRelease:        true,
	}

	if _, _, err := autoReleaseAcceptRequest(task, candidate, defaultAutoReleasePolicy); err == nil || !strings.Contains(err.Error(), "deployment validation") {
		t.Fatalf("expected deployment-sensitive candidate signal to block auto-release, got %v", err)
	}

	candidate.DeploymentStatus = "validated"
	candidate.ValidationSignals = append(candidate.ValidationSignals, "deployment: verified")
	if _, reference, err := autoReleaseAcceptRequest(task, candidate, defaultAutoReleasePolicy); err != nil {
		t.Fatalf("validated deployment signal should pass: %v", err)
	} else if !strings.Contains(reference, "deployment_validation:validated") {
		t.Fatalf("reference missing deployment validation marker: %s", reference)
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
	for _, value := range []string{"tsk_public", `"task_id"`, `"review_packet"`, `"release_packet"`, `"auto_release_packet"`, "github:builder"} {
		if strings.Contains(string(body), value) {
			t.Fatalf("public PR monitor leaked private release value %q: %s", value, string(body))
		}
	}
}
