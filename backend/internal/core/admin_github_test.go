package core

import (
	"strings"
	"testing"
)

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
		scanAccountURL(Config{ScanDomain: "scan.mergeos.shop"}, "worker:github:hummusonrails"),
	)
	if !strings.Contains(comment, "Merge URL: https://github.com/mergeos-bounties/demo/pull/4") {
		t.Fatalf("comment used non-url merge value: %s", comment)
	}
	if !strings.Contains(comment, "MRG credit URL: https://scan.mergeos.shop/address/worker:github:hummusonrails") {
		t.Fatalf("comment missing scan credit URL: %s", comment)
	}
	if !strings.Contains(comment, "Credited worker: github:hummusonrails") {
		t.Fatalf("comment missing github worker: %s", comment)
	}
}
