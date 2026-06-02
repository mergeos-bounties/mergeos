package core

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

func TestVerifyMergeOSSignature(t *testing.T) {
	body := []byte(`{"pull_number":14}`)
	mac := hmac.New(sha256.New, []byte("secret"))
	mac.Write(body)
	signature := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	if !verifyMergeOSSignature("secret", signature, body) {
		t.Fatal("expected valid signature")
	}
	if verifyMergeOSSignature("secret", signature, []byte(`{"pull_number":15}`)) {
		t.Fatal("expected changed body to fail")
	}
	if verifyMergeOSSignature("", signature, body) {
		t.Fatal("expected empty secret to fail")
	}
}

func TestReviewEvidenceProvided(t *testing.T) {
	var pr geminiReviewPullRequest
	pr.User.Login = "alice"
	pr.Body = "## Verification\n- npm test\n- Browser check at 390x664"

	if !reviewEvidenceProvided(pr, nil) {
		t.Fatal("expected verification notes to count as evidence")
	}

	pr.Body = "Fixes #13"
	if reviewEvidenceProvided(pr, nil) {
		t.Fatal("expected empty evidence to be missing")
	}

	comments := []geminiReviewComment{{Body: "After screenshot: https://github.com/user-attachments/assets/abc", User: struct {
		Login string `json:"login"`
		Type  string `json:"type"`
	}{Login: "alice"}}}
	if !reviewEvidenceProvided(pr, comments) {
		t.Fatal("expected author attachment comment to count as evidence")
	}
}

func TestSplitEnvList(t *testing.T) {
	keys := splitEnvList("one,two;\nthree\r\n")
	if len(keys) != 3 || keys[0] != "one" || keys[1] != "two" || keys[2] != "three" {
		t.Fatalf("unexpected keys: %#v", keys)
	}
}

func TestGeminiAPIKeyStats(t *testing.T) {
	store := &Store{
		cfg:           Config{GeminiAPIKeys: []string{"first-key", "second-key"}},
		geminiAPIKeys: map[string]*GeminiAPIKey{},
	}
	if err := store.SeedGeminiAPIKeysFromConfig(); err != nil {
		t.Fatalf("seed keys: %v", err)
	}
	candidates := store.GeminiAPIKeyCandidates()
	if len(candidates) != 2 {
		t.Fatalf("expected two candidates, got %d", len(candidates))
	}
	firstID := candidates[0].ID
	if err := store.MarkGeminiAPIKeyAttempt(firstID); err != nil {
		t.Fatalf("mark attempt: %v", err)
	}
	if err := store.MarkGeminiAPIKeyQuotaLimited(firstID, 429, "quota exceeded"); err != nil {
		t.Fatalf("mark quota: %v", err)
	}
	stats := store.ListGeminiAPIKeyStats()
	if len(stats) != 2 {
		t.Fatalf("expected two stats rows, got %d", len(stats))
	}
	var quotaFound bool
	for _, item := range stats {
		if item.ID == firstID && item.Status == GeminiAPIKeyStatusQuotaLimited && item.RequestCount == 1 {
			quotaFound = true
		}
	}
	if !quotaFound {
		t.Fatalf("expected quota status for first key: %#v", stats)
	}
	next := store.GeminiAPIKeyCandidates()
	if len(next) != 1 || next[0].ID == firstID {
		t.Fatalf("expected quota-limited key to be skipped: %#v", next)
	}
	if err := store.MarkGeminiAPIKeyAttempt(next[0].ID); err != nil {
		t.Fatalf("mark second attempt: %v", err)
	}
	if err := store.MarkGeminiAPIKeySuccess(next[0].ID, 200); err != nil {
		t.Fatalf("mark second success: %v", err)
	}
	stats = store.ListGeminiAPIKeyStats()
	var successFound bool
	for _, item := range stats {
		if item.SuccessCount == 1 && item.Status == GeminiAPIKeyStatusActive {
			successFound = true
		}
	}
	if !successFound {
		t.Fatalf("expected successful key stats: %#v", stats)
	}
}

func TestGeminiAPIKeyAdminUpdates(t *testing.T) {
	store := &Store{geminiAPIKeys: map[string]*GeminiAPIKey{}}
	added, err := store.AddGeminiAPIKey("admin-added-key")
	if err != nil {
		t.Fatalf("add key: %v", err)
	}
	if added.Status != GeminiAPIKeyStatusActive || added.KeyHint == "" {
		t.Fatalf("unexpected added key: %#v", added)
	}
	if _, err := store.AddGeminiAPIKey("admin-added-key"); err == nil {
		t.Fatal("expected duplicate key to fail")
	}
	updated, err := store.UpdateGeminiAPIKey(added.ID, GeminiAPIKeyStatusDisabled, true)
	if err != nil {
		t.Fatalf("disable key: %v", err)
	}
	if updated.Status != GeminiAPIKeyStatusDisabled || updated.RequestCount != 0 {
		t.Fatalf("unexpected updated key: %#v", updated)
	}
	if store.HasRunnableGeminiAPIKey() {
		t.Fatal("expected disabled key to stop being runnable")
	}
}

func TestGeminiAPIKeyTestResultPreservesDisabledStatus(t *testing.T) {
	store := &Store{geminiAPIKeys: map[string]*GeminiAPIKey{}}
	added, err := store.AddGeminiAPIKey("admin-test-key")
	if err != nil {
		t.Fatalf("add key: %v", err)
	}
	if _, err := store.UpdateGeminiAPIKey(added.ID, GeminiAPIKeyStatusDisabled, false); err != nil {
		t.Fatalf("disable key: %v", err)
	}
	stats, err := store.RecordGeminiAPIKeyTestResult(added.ID, GeminiAPIKeyStatusActive, 200, "")
	if err != nil {
		t.Fatalf("record test: %v", err)
	}
	if stats.Status != GeminiAPIKeyStatusDisabled {
		t.Fatalf("test should not re-enable disabled key: %#v", stats)
	}
	if stats.RequestCount != 1 || stats.SuccessCount != 1 || stats.LastStatusCode != 200 || stats.LastUsedAt == nil {
		t.Fatalf("test counters not updated: %#v", stats)
	}
}

func TestGeminiWebhookLogs(t *testing.T) {
	store := &Store{geminiWebhookLogs: map[string]*GeminiWebhookLog{}}
	if err := store.AddGeminiWebhookLog(GeminiWebhookLog{
		DeliveryID: "delivery-1",
		EventName:  "pull_request",
		Action:     "opened",
		Repository: "mergeos-bounties/mergeos",
		PullNumber: 14,
		Status:     "processed",
		StatusCode: 200,
		Labels:     []string{"evidence: provided"},
	}); err != nil {
		t.Fatalf("add webhook log: %v", err)
	}
	logs := store.ListGeminiWebhookLogs(10)
	if len(logs) != 1 {
		t.Fatalf("expected one log, got %d", len(logs))
	}
	if logs[0].DeliveryID != "delivery-1" || logs[0].Labels[0] != "evidence: provided" {
		t.Fatalf("unexpected log: %#v", logs[0])
	}
}

func TestGeminiWebhookBroadcastType(t *testing.T) {
	cases := []struct {
		name string
		log  GeminiWebhookLog
		want string
	}{
		{
			name: "pull opened",
			log:  GeminiWebhookLog{EventName: "pull_request", Action: "opened", Repository: "mergeos-bounties/mergeos", PullNumber: 14},
			want: "pr_opened",
		},
		{
			name: "pull reviewed",
			log:  GeminiWebhookLog{EventName: "pull_request", Action: "synchronize", Repository: "mergeos-bounties/mergeos", PullNumber: 14},
			want: "ai_review",
		},
		{
			name: "issue comment reviewed",
			log:  GeminiWebhookLog{EventName: "issue_comment", Action: "created", Repository: "mergeos-bounties/mergeos", PullNumber: 14},
			want: "ai_review",
		},
		{
			name: "agent action",
			log:  GeminiWebhookLog{EventName: "agent_action", Action: "test", Repository: "mergeos-bounties/mergeos"},
			want: "agent_action",
		},
		{
			name: "missing context",
			log:  GeminiWebhookLog{EventName: "pull_request", Action: "opened"},
			want: "",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := geminiWebhookBroadcastType(tc.log); got != tc.want {
				t.Fatalf("broadcast type = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestGeminiReviewRequestFromGitHubWebhook(t *testing.T) {
	body := []byte(`{
		"action":"opened",
		"number":14,
		"repository":{"full_name":"mergeos-bounties/mergeos"},
		"sender":{"login":"maintainer"},
		"pull_request":{
			"number":14,
			"title":"Fix auth modal",
			"body":"Fixes #13",
			"html_url":"https://github.com/mergeos-bounties/mergeos/pull/14",
			"user":{"login":"alice"},
			"base":{"ref":"master"},
			"head":{"ref":"fix","sha":"abc"},
			"draft":false
		}
	}`)
	req, ok, err := geminiReviewRequestFromGitHubWebhook("pull_request", body)
	if err != nil || !ok {
		t.Fatalf("expected request, ok=%t err=%v", ok, err)
	}
	if req.Repository != "mergeos-bounties/mergeos" || req.PullNumber != 14 || req.PullRequest.Author != "alice" {
		t.Fatalf("unexpected request: %#v", req)
	}
}

func TestGeminiReviewRequestSkipsOwnComment(t *testing.T) {
	body := []byte(`{
		"action":"edited",
		"repository":{"full_name":"mergeos-bounties/mergeos"},
		"sender":{"login":"maintainer"},
		"comment":{"body":"<!-- mergeos-gemini-pr-review -->\nreview"},
		"issue":{"number":14,"pull_request":{}}
	}`)
	_, ok, err := geminiReviewRequestFromGitHubWebhook("issue_comment", body)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ok {
		t.Fatal("expected own Gemini comment to be skipped")
	}
}
