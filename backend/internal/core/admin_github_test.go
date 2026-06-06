package core

import (
	"strings"
	"testing"
)

func containsString(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}

func TestParseGitHubIssueURL(t *testing.T) {
	target, err := parseGitHubIssueURL("https://github.com/mergeos-bounties/mergeos/issues/42")
	if err != nil {
		t.Fatal(err)
	}
	if target.Owner != "mergeos-bounties" || target.Repo != "mergeos" || target.IssueNumber != 42 {
		t.Fatalf("target = %#v", target)
	}
}

func TestGitHubIssueTargetForTaskUsesImportedIssueURL(t *testing.T) {
	task := &Task{
		IssueNumber: 7,
		IssueURL:    "https://github.com/source-org/source-repo/issues/9",
	}
	project := &Project{
		RepoProvider:   "local-git",
		BountyRepoName: "mergeos-bounties/local-child",
	}
	target, err := githubIssueTargetForTask(task, project)
	if err != nil {
		t.Fatal(err)
	}
	if target.fullName() != "source-org/source-repo" || target.IssueNumber != 9 {
		t.Fatalf("target = %#v", target)
	}
}

func TestGitHubIssueTargetForTaskUsesBountyRepo(t *testing.T) {
	task := &Task{IssueNumber: 5}
	project := &Project{
		RepoProvider:   "github",
		BountyRepoName: "mergeos-bounties/private-child",
	}
	target, err := githubIssueTargetForTask(task, project)
	if err != nil {
		t.Fatal(err)
	}
	if target.fullName() != "mergeos-bounties/private-child" || target.IssueNumber != 5 {
		t.Fatalf("target = %#v", target)
	}
}

func TestAcceptRequestForPullAuthorCreditsGitHubWorker(t *testing.T) {
	req, err := acceptRequestForPullAuthor(&Task{RequiredWorkerKind: WorkerHuman}, "@maya-dev")
	if err != nil {
		t.Fatal(err)
	}
	if req.WorkerKind != WorkerHuman || req.WorkerID != "github:maya-dev" || req.AgentType != "" {
		t.Fatalf("human req = %#v", req)
	}

	agentReq, err := acceptRequestForPullAuthor(&Task{
		RequiredWorkerKind: WorkerAgent,
		SuggestedAgentType: "go-ledger-agent",
	}, "octo")
	if err != nil {
		t.Fatal(err)
	}
	if agentReq.WorkerKind != WorkerAgent || agentReq.WorkerID != "github:octo" || agentReq.AgentType != "go-ledger-agent" {
		t.Fatalf("agent req = %#v", agentReq)
	}
}

func TestNormalizeAdminBountyType(t *testing.T) {
	bountyType, err := normalizeAdminBountyType("Bug-Large")
	if err != nil {
		t.Fatal(err)
	}
	if bountyType != "bug-large" {
		t.Fatalf("bounty type = %q", bountyType)
	}
	if _, err := normalizeAdminBountyType("tiny"); err == nil {
		t.Fatal("expected unsupported bounty type error")
	}
}

func TestRenderMergeOSPullCommentLinksScanCreditAccount(t *testing.T) {
	comment := renderMergeOSPullComment(
		&Task{ProofHash: "proof123"},
		AdminTaskPullRequest{
			HTMLURL:  "https://github.com/mergeos-bounties/demo/pull/4",
			MergeURL: "4406a84",
		},
		"github:hummusonrails",
		50,
		"future-medium",
		scanAccountURL(Config{ScanDomain: "scan.mergeos.shop"}, "github:hummusonrails"),
	)
	if !strings.Contains(comment, "Merge URL: https://github.com/mergeos-bounties/demo/pull/4") {
		t.Fatalf("comment used non-url merge value: %s", comment)
	}
	if !strings.Contains(comment, "MRG credit URL: https://scan.mergeos.shop/address/github:hummusonrails") {
		t.Fatalf("comment missing scan credit URL: %s", comment)
	}
	if !strings.Contains(comment, "Credited worker: github:hummusonrails") {
		t.Fatalf("comment missing github worker: %s", comment)
	}
}

func TestPullLedgerReferencePublishesPullLinkAndTitle(t *testing.T) {
	reference := buildPullLedgerReference(
		"tsk_0041",
		"https://github.com/mergeos-bounties/mergeos/pull/120?utm=admin",
		"Timeline correction; reviewer bonus",
	)
	if !strings.Contains(reference, "task:tsk_0041") {
		t.Fatalf("reference missing task id: %s", reference)
	}
	publicReference := publicPullLedgerReference(reference)
	if publicReference != "pr:https://github.com/mergeos-bounties/mergeos/pull/120;title:Timeline correction, reviewer bonus" {
		t.Fatalf("public reference = %q", publicReference)
	}
	if strings.Contains(publicReference, "tsk_0041") {
		t.Fatalf("public reference leaked task id: %s", publicReference)
	}
}

func TestNeutralizeClosingIssueKeywords(t *testing.T) {
	body, changed := neutralizeClosingIssueKeywords("Closes #3\nFixes mergeos-bounties/mergeos#4\nResolves: https://github.com/mergeos-bounties/mergeos/issues/5")
	if !changed {
		t.Fatal("expected closing keywords to change")
	}
	for _, blocked := range []string{"Closes #3", "Fixes mergeos-bounties/mergeos#4", "Resolves:"} {
		if strings.Contains(body, blocked) {
			t.Fatalf("body still contains closing keyword %q: %s", blocked, body)
		}
	}
	for _, expected := range []string{"Related to #3", "Related to mergeos-bounties/mergeos#4", "Related to https://github.com/mergeos-bounties/mergeos/issues/5"} {
		if !strings.Contains(body, expected) {
			t.Fatalf("body missing neutral reference %q: %s", expected, body)
		}
	}

	safe, changed := neutralizeClosingIssueKeywords("Related to #3")
	if changed || safe != "Related to #3" {
		t.Fatalf("safe body changed to %q", safe)
	}
}

func TestAdminPullRequestReadinessBlocksMissingEvidenceAndWorkflowDeletion(t *testing.T) {
	readiness := adminPullRequestReadiness(
		&Task{Title: "PayPal payment flow", Acceptance: "Provider evidence required"},
		AdminTaskPullRequest{
			State:          "open",
			MergeableState: "clean",
			Labels:         []string{"star: verified", "evidence: missing"},
			ChangedFiles: []AdminPullRequestFile{
				{Path: ".github/workflows/deploy.yml", Status: "removed", Deletions: 30},
			},
		},
	)
	if readiness.CanMerge || readiness.Status != "blocked" || readiness.RiskLevel != "high" {
		t.Fatalf("readiness should block merge: %#v", readiness)
	}
	for _, expected := range []string{"evidence: provided label is required", "workflow file deletion requires separate maintainer approval"} {
		found := false
		for _, blocker := range readiness.Blockers {
			if blocker == expected {
				found = true
			}
		}
		if !found {
			t.Fatalf("missing blocker %q in %#v", expected, readiness.Blockers)
		}
	}
}

func TestAdminPullRequestReadinessBlocksSecretFilesAndWarnsSensitiveCode(t *testing.T) {
	readiness := adminPullRequestReadiness(
		&Task{Title: "Auth payment hardening", Acceptance: "Update admin auth and PayPal validation"},
		AdminTaskPullRequest{
			State:          "open",
			MergeableState: "clean",
			Labels:         []string{"star: verified", "evidence: provided"},
			ChangedFiles: []AdminPullRequestFile{
				{Path: "backend/internal/core/auth.go", Status: "modified", Additions: 12, Deletions: 2},
				{Path: "backend/secrets/paypal.pem", Status: "added", Additions: 20},
			},
		},
	)
	if readiness.CanMerge || readiness.Status != "blocked" || readiness.RiskLevel != "high" {
		t.Fatalf("readiness should block secret file change: %#v", readiness)
	}
	if !containsString(readiness.Blockers, "secret or credential file changes are not allowed in bounty PRs") {
		t.Fatalf("missing secret blocker: %#v", readiness.Blockers)
	}
	if !containsString(readiness.Warnings, "security-sensitive code paths changed; maintainer review required") {
		t.Fatalf("missing sensitive-code warning: %#v", readiness.Warnings)
	}
	if !containsString(readiness.Signals, "security-sensitive-path") {
		t.Fatalf("missing sensitive-code signal: %#v", readiness.Signals)
	}
}

func TestAdminPullRequestReadinessBlocksBroadDeletionHeavyDiff(t *testing.T) {
	readiness := adminPullRequestReadiness(
		&Task{Title: "Small frontend fix", Acceptance: "Keep changes scoped to the issue"},
		AdminTaskPullRequest{
			State:          "open",
			MergeableState: "clean",
			Labels:         []string{"star: verified", "evidence: provided"},
			ChangedFiles: []AdminPullRequestFile{
				{Path: "frontend/src/App.vue", Status: "modified", Additions: 4, Deletions: 80},
				{Path: "frontend/src/styles.css", Status: "modified", Additions: 2, Deletions: 60},
				{Path: "README.md", Status: "modified", Deletions: 50},
				{Path: "sdk/README.md", Status: "modified", Deletions: 30},
				{Path: "protocol/README.md", Status: "modified", Deletions: 20},
				{Path: "contracts/README.md", Status: "modified", Deletions: 20},
			},
		},
	)
	if readiness.CanMerge || readiness.Status != "blocked" || readiness.RiskLevel != "high" {
		t.Fatalf("readiness should block broad deletion diff: %#v", readiness)
	}
	if !containsString(readiness.Blockers, "broad deletion-heavy diff requires separate maintainer approval") {
		t.Fatalf("missing deletion-heavy blocker: %#v", readiness.Blockers)
	}
}

func TestAdminPullRequestReadinessBlocksSpamLabel(t *testing.T) {
	readiness := adminPullRequestReadiness(
		&Task{Title: "Dashboard copy fix"},
		AdminTaskPullRequest{
			State:          "open",
			MergeableState: "clean",
			Labels:         []string{"star: verified", "evidence: provided", "spam"},
			ChangedFiles:   []AdminPullRequestFile{{Path: "frontend/src/App.vue", Status: "modified", Additions: 3}},
		},
	)
	if readiness.CanMerge || readiness.Status != "blocked" || readiness.RiskLevel != "high" {
		t.Fatalf("readiness should block spam label: %#v", readiness)
	}
	if !containsString(readiness.Blockers, "pull request is marked spam or invalid") {
		t.Fatalf("missing spam blocker: %#v", readiness.Blockers)
	}
}

func TestAdminPullRequestReadinessRequiresDeploymentValidationForAutoRelease(t *testing.T) {
	readiness := adminPullRequestReadiness(
		&Task{Title: "Deployment handoff", Acceptance: "Preview rollout must be validated before release."},
		AdminTaskPullRequest{
			State:          "open",
			MergeableState: "clean",
			Labels:         []string{"star: verified", "evidence: provided"},
			ChangedFiles: []AdminPullRequestFile{
				{Path: "frontend/src/App.vue", Status: "modified", Additions: 12, Deletions: 2},
			},
		},
	)
	if !readiness.CanMerge || readiness.Status != "needs_review" || readiness.RiskLevel != "medium" {
		t.Fatalf("deployment PR should need review before auto-release: %#v", readiness)
	}
	if !containsString(readiness.Signals, "deployment-sensitive") {
		t.Fatalf("missing deployment-sensitive signal: %#v", readiness.Signals)
	}
	if !containsString(readiness.Warnings, "deployment validation is required before auto-release") {
		t.Fatalf("missing deployment validation warning: %#v", readiness.Warnings)
	}

	verified := adminPullRequestReadiness(
		&Task{Title: "Deployment handoff", Acceptance: "Preview rollout must be validated before release."},
		AdminTaskPullRequest{
			State:          "open",
			MergeableState: "clean",
			Labels:         []string{"star: verified", "evidence: provided", "deployment: verified"},
			ChangedFiles: []AdminPullRequestFile{
				{Path: "frontend/src/App.vue", Status: "modified", Additions: 12, Deletions: 2},
			},
		},
	)
	if !verified.CanMerge || verified.Status != "ready" || verified.RiskLevel != "low" {
		t.Fatalf("verified deployment PR should be ready: %#v", verified)
	}
	if !containsString(verified.Signals, "deployment: verified") {
		t.Fatalf("missing deployment proof signal: %#v", verified.Signals)
	}
}

func TestAdminPullRequestReadinessAllowsVerifiedLowRiskPR(t *testing.T) {
	readiness := adminPullRequestReadiness(
		&Task{Title: "Dashboard copy fix"},
		AdminTaskPullRequest{
			State:          "open",
			MergeableState: "clean",
			Labels:         []string{"star: verified", "evidence: provided"},
			ChangedFiles: []AdminPullRequestFile{
				{Path: "frontend/src/App.vue", Status: "modified", Additions: 12, Deletions: 2},
			},
		},
	)
	if !readiness.CanMerge || readiness.Status != "ready" || readiness.RiskLevel != "low" {
		t.Fatalf("readiness should allow verified low-risk PR: %#v", readiness)
	}
	if len(readiness.Blockers) != 0 || len(readiness.Warnings) != 0 {
		t.Fatalf("unexpected readiness notes: %#v", readiness)
	}
}
