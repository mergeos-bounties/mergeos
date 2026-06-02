package core

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

func newTestStore(t testing.TB) *Store {
	t.Helper()
	cfg := Config{
		Environment: "test",
		StatePath:   "",
	}
	store := &Store{
		cfg:                 cfg,
		nextID:              1,
		projects:            map[string]*Project{},
		tasks:               map[string]*Task{},
		users:               map[string]*User{},
		wallets:             map[string]*Wallet{},
		sessions:            map[string]*Session{},
		notifications:       map[string]*Notification{},
		attachments:         map[string]*Attachment{},
		sslReviews:          map[string]*SSLReviewStatus{},
		geminiAPIKeys:       map[string]*GeminiAPIKey{},
		geminiWebhookLogs:   map[string]*GeminiWebhookLog{},
		testSettingsConfig:  TestSettingsConfig{},
		testSettingsEntries: map[string]*TestSettingsEntry{},
		adminSettings:       defaultAdminSettings(cfg),
		ledger:              []LedgerEntry{},
	}
	return store
}

func TestSettingValueMask(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"", "****"},
		{"a", "a****a"},
		{"abcdefghi", "abcd****fghi"},
		{"sec_api_key_abc123xy", "sec_****23xy"},
	}
	for _, tt := range tests {
		got := SettingValueMask(tt.input)
		if got != tt.expected {
			t.Errorf("SettingValueMask(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestCheckForEnvCollision(t *testing.T) {
	t.Setenv("runtime_secret_name", "secret")
	runtimeEnvNames = nil
	runtimeEnvOnce = sync.Once{}

	for _, name := range []string{"GITHUB_TOKEN", "ADMIN_EMAIL", "ADMIN_PASSWORD", "PAYPAL_CLIENT_ID", "PAYPAL_ENV", "CRYPTO_RECEIVER", "GEMINI_API_KEYS", "runtime_secret_name"} {
		if err := checkForEnvCollision(name, nil); err == nil {
			t.Errorf("expected collision for %q", name)
		}
	}
	if err := checkForEnvCollision("MERGEOS_FOO", nil); err == nil {
		t.Error("expected collision for MERGEOS_* prefix")
	}
	if err := checkForEnvCollision("my-custom-key", nil); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	// Test nested key collision.
	if err := checkForEnvCollision("my-key", []string{"GITHUB_TOKEN"}); err == nil {
		t.Error("expected collision for nested key GITHUB_TOKEN")
	}
	if err := checkForEnvCollision("my-key", []string{"my-safe-key"}); err != nil {
		t.Errorf("unexpected error for safe nested key: %v", err)
	}
}

func TestTestSettingsStore(t *testing.T) {
	store := newTestStore(t)
	defer store.Close()

	enabled := true
	_, err := store.UpdateTestSettingsConfig(UpdateTestSettingsRequest{
		TestModeEnabled: &enabled,
		TestPassword:    "test12345",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !store.GetTestSettingsConfig().TestModeEnabled {
		t.Error("expected enabled")
	}
	if !store.VerifyTestPassword("test12345") {
		t.Error("password should match")
	}
	if store.VerifyTestPassword("wrong") {
		t.Error("wrong pw should fail")
	}

	entry, err := store.AddTestSettingsEntry(AddTestEntryRequest{
		IntegrationType: "llm", SettingKey: "OPENAI_KEY", SettingValue: "sk-proj-abc",
	})
	if err != nil {
		t.Fatal(err)
	}
	if entry.SettingValueHint != SettingValueMask("sk-proj-abc") {
		t.Errorf("bad mask: %q", entry.SettingValueHint)
	}

	_, err = store.AddTestSettingsEntry(AddTestEntryRequest{
		IntegrationType: "env", SettingKey: "GITHUB_TOKEN", SettingValue: "xxx",
	})
	if err == nil {
		t.Error("expected env collision error")
	}
}

func TestRevealTestSettingsEntryReturnsSecretAndTracksUsage(t *testing.T) {
	store := newTestStore(t)
	defer store.Close()

	entry, err := store.AddTestSettingsEntry(AddTestEntryRequest{
		IntegrationType: "paypal",
		DisplayName:     "PayPal Sandbox",
		SettingKey:      "TASK_PAYPAL_SANDBOX",
		SettingValue:    "sandbox-secret",
		KeyValueMap: map[string]string{
			"client_id": "client-123",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if entry.SettingValueHint == "sandbox-secret" {
		t.Fatal("normal list response should not expose the primary secret")
	}

	revealed, err := store.RevealTestSettingsEntry(entry.ID)
	if err != nil {
		t.Fatal(err)
	}
	if revealed.SettingValue != "sandbox-secret" || revealed.KeyValueMap["client_id"] != "client-123" {
		t.Fatalf("reveal did not return stored secret values: %#v", revealed)
	}
	if revealed.LastUsedAt == nil {
		t.Fatal("expected reveal to update last_used_at")
	}

	rows := store.ListTestSettingsEntries()
	if len(rows) != 1 || rows[0].SettingValueHint == "sandbox-secret" || rows[0].KeyValueMap["client_id"] == "client-123" {
		t.Fatalf("masked list response leaked secret values: %#v", rows)
	}
}

func TestResolveActiveTestSettingsRequiresTestModeAndFiltersEntries(t *testing.T) {
	store := newTestStore(t)
	defer store.Close()

	if _, err := store.ResolveActiveTestSettings("llm"); err == nil {
		t.Fatal("expected disabled test mode to block runtime resolution")
	}

	enabled := true
	if _, err := store.UpdateTestSettingsConfig(UpdateTestSettingsRequest{
		TestModeEnabled: &enabled,
		TestPassword:    "runtime-test-123",
	}); err != nil {
		t.Fatal(err)
	}

	llmEntry, err := store.AddTestSettingsEntry(AddTestEntryRequest{
		IntegrationType: "llm",
		DisplayName:     "LLM runtime",
		SettingKey:      "TASK_LLM_RUNTIME_KEY",
		SettingValue:    "llm-secret",
		KeyValueMap: map[string]string{
			"model": "test-model",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	paypalEntry, err := store.AddTestSettingsEntry(AddTestEntryRequest{
		IntegrationType: "paypal",
		DisplayName:     "PayPal runtime",
		SettingKey:      "TASK_PAYPAL_RUNTIME_KEY",
		SettingValue:    "paypal-secret",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.UpdateTestSettingsEntry(paypalEntry.ID, UpdateTestEntryRequest{Status: "disabled"}); err != nil {
		t.Fatal(err)
	}

	resolved, err := store.ResolveActiveTestSettings("LLM")
	if err != nil {
		t.Fatal(err)
	}
	if len(resolved) != 1 || resolved[0].ID != llmEntry.ID || resolved[0].SettingValue != "llm-secret" || resolved[0].KeyValueMap["model"] != "test-model" {
		t.Fatalf("unexpected resolved entries: %#v", resolved)
	}
	resolved[0].KeyValueMap["model"] = "mutated"
	again, err := store.ResolveActiveTestSettings("llm")
	if err != nil {
		t.Fatal(err)
	}
	if again[0].KeyValueMap["model"] != "test-model" {
		t.Fatalf("runtime resolver leaked mutable map reference: %#v", again[0].KeyValueMap)
	}

	paypalEntries, err := store.ResolveActiveTestSettings("paypal")
	if err != nil {
		t.Fatal(err)
	}
	if len(paypalEntries) != 0 {
		t.Fatalf("disabled paypal entry should not resolve: %#v", paypalEntries)
	}
}

func TestTestSettingsConfigPersistencePreservesPasswordHash(t *testing.T) {
	store := newTestStore(t)
	defer store.Close()

	enabled := true
	if _, err := store.UpdateTestSettingsConfig(UpdateTestSettingsRequest{
		TestModeEnabled: &enabled,
		TestPassword:    "persist-me-123",
	}); err != nil {
		t.Fatal(err)
	}

	state := store.snapshotLocked()
	raw, err := json.Marshal(state)
	if err != nil {
		t.Fatal(err)
	}
	var reloadedState persistedState
	if err := json.Unmarshal(raw, &reloadedState); err != nil {
		t.Fatal(err)
	}

	reloaded := newTestStore(t)
	defer reloaded.Close()
	reloaded.applyState(reloadedState)
	if !reloaded.GetTestSettingsConfig().TestModeEnabled {
		t.Fatal("expected test mode to stay enabled after persistence reload")
	}
	if !reloaded.VerifyTestPassword("persist-me-123") {
		t.Fatal("expected public test password to survive persistence reload")
	}
}

func TestTestSettingsConfigResponseDoesNotExposePasswordHash(t *testing.T) {
	response := testSettingsConfigResponse(TestSettingsConfig{
		TestModeEnabled:  true,
		TestPasswordHash: "salt:hash",
	})
	raw, err := json.Marshal(response)
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) == "" || !json.Valid(raw) {
		t.Fatalf("invalid response JSON: %s", raw)
	}
	if strings.Contains(string(raw), "salt:hash") || strings.Contains(string(raw), "test_password_hash") {
		t.Fatalf("response leaked password hash: %s", raw)
	}
}

func TestTestSettingsRateLimitKeyNormalizesRemotePort(t *testing.T) {
	reqA := httptest.NewRequest("POST", "/api/public/test-settings/auth", nil)
	reqA.RemoteAddr = "203.0.113.10:51000"
	reqB := httptest.NewRequest("POST", "/api/public/test-settings/auth", nil)
	reqB.RemoteAddr = "203.0.113.10:51001"

	if testSettingsRateLimitKey(reqA) != "203.0.113.10" || testSettingsRateLimitKey(reqB) != "203.0.113.10" {
		t.Fatalf("rate limit key should ignore remote port: %q / %q", testSettingsRateLimitKey(reqA), testSettingsRateLimitKey(reqB))
	}
}

func TestTestSettingsRateLimitKeyUsesTrustedProxyForwardedFor(t *testing.T) {
	req := httptest.NewRequest("POST", "/api/public/test-settings/auth", nil)
	req.RemoteAddr = "127.0.0.1:42000"
	req.Header.Set("X-Forwarded-For", "198.51.100.7, 127.0.0.1")

	if got := testSettingsRateLimitKey(req); got != "198.51.100.7" {
		t.Fatalf("rate limit key = %q, want forwarded client IP", got)
	}
}
