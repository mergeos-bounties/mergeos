package core

import "testing"

func TestParseGitHubRepoURL(t *testing.T) {
	fullName, htmlURL := parseGitHubRepoURL("https://github.com/mergeos-bounties/NeraJob")
	if fullName != "mergeos-bounties/NeraJob" || htmlURL != "https://github.com/mergeos-bounties/NeraJob" {
		t.Fatalf("got fullName=%q htmlURL=%q", fullName, htmlURL)
	}
	fullName, htmlURL = parseGitHubRepoURL("mergeos-bounties/Loru")
	if fullName != "mergeos-bounties/Loru" || htmlURL != "https://github.com/mergeos-bounties/Loru" {
		t.Fatalf("short form got fullName=%q htmlURL=%q", fullName, htmlURL)
	}
	fullName, htmlURL = parseGitHubRepoURL("https://github.com/mergeos-bounties/PoseGuide.git")
	if fullName != "mergeos-bounties/PoseGuide" {
		t.Fatalf(".git suffix got %q", fullName)
	}
	if fullName, _ = parseGitHubRepoURL("https://example.com/not-github"); fullName != "" {
		t.Fatalf("expected empty for non-github, got %q", fullName)
	}
}

func TestBindExistingGitHubRepoKeepsImportedIssues(t *testing.T) {
	project := &Project{
		Title:          "NeraJob",
		BountyRepoName: "mergeos-bounties/NeraJob",
		RepoURL:        "https://github.com/mergeos-bounties/NeraJob",
	}
	tasks := []*Task{
		{
			ID:          "tsk_1",
			IssueNumber: 22,
			IssueURL:    "https://github.com/mergeos-bounties/NeraJob/issues/22",
			Title:       "Fix #22",
		},
		{
			ID:          "tsk_2",
			IssueNumber: 5,
			Title:       "Fix #5",
		},
	}
	result := bindExistingGitHubRepo(project, tasks, "mergeos-bounties/NeraJob")
	if result.Provider != "github" || result.Name != "mergeos-bounties/NeraJob" {
		t.Fatalf("unexpected result: %#v", result)
	}
	if result.Issues["tsk_1"].URL != "https://github.com/mergeos-bounties/NeraJob/issues/22" {
		t.Fatalf("issue 1 url = %q", result.Issues["tsk_1"].URL)
	}
	if result.Issues["tsk_2"].URL != "https://github.com/mergeos-bounties/NeraJob/issues/5" {
		t.Fatalf("issue 2 url = %q", result.Issues["tsk_2"].URL)
	}
	if result.Issues["tsk_2"].Number != 5 {
		t.Fatalf("issue 2 number = %d", result.Issues["tsk_2"].Number)
	}
}

func TestGitHubRepoFactoryNeverCreatesPrivateChildWithoutSource(t *testing.T) {
	tempDir := t.TempDir()
	cfg := Config{
		TokenSymbol:    defaultTokenSymbol,
		GitHubOwner:    defaultGitHubOwner,
		GitHubToken:    "fake-token-must-not-be-used-for-repo-create",
		GitHubOwnerType: "org",
		BountyRoot:     tempDir,
	}
	factory := &GitHubRepoFactory{cfg: cfg}
	project := &Project{
		ID:         "prj_test",
		Title:      "No source project",
		ClientName: "MergeOS",
		Brief:      "Should use local workspace only",
	}
	tasks := []*Task{
		{ID: "tsk_a", IssueNumber: 1, Title: "Task A", RequiredWorkerKind: WorkerHuman},
	}
	result, err := factory.CreateProjectRepo(t.Context(), project, tasks)
	if err != nil {
		t.Fatal(err)
	}
	if result.Provider != "local-git" {
		t.Fatalf("provider = %q, want local-git (no private GitHub child)", result.Provider)
	}
	if result.LocalPath == "" {
		t.Fatal("expected local path")
	}
}
