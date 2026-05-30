package core

import (
	"path/filepath"
	"testing"
)

func TestBlockedSettingNames(t *testing.T) {
	blocked := []string{"GITHUB_TOKEN", "PAYPAL_CLIENT_ID", "MERGEOS_GITHUB_TOKEN", "ADMIN_EMAIL", "TOKEN_SYMBOL"}
	allowed := []string{"llm_test_keys", "paypal_sandbox_test_accounts", "usdt_test_receivers", "my_custom_key"}

	for _, name := range blocked {
		if !isBlockedSettingName(name) {
			t.Errorf("expected %q to be blocked", name)
		}
	}
	for _, name := range allowed {
		if isBlockedSettingName(name) {
			t.Errorf("expected %q to be allowed", name)
		}
	}
}

func TestTestModeEnabledDisabled(t *testing.T) {
	tempDir := t.TempDir()
	cfg := Config{
		Environment:       "local",
		TokenSymbol:       defaultTokenSymbol,
		StatePath:         filepath.Join(tempDir, "state.json"),
		PlatformFeeBps:    1000,
		DevPaymentEnabled: true,
		DevPaymentCode:    "LOCAL-PAID",
		GitHubOwner:       defaultGitHubOwner,
		BountyRoot:        filepath.Join(tempDir, "bounties"),
		SMTPFrom:          "noreply@mergeos.local",
	}
	payments := NewPaymentManager(cfg)
	store, err := NewStore(cfg, payments, NewRepoFactory(cfg), NewEmailSender(cfg))
	if err != nil {
		t.Fatal(err)
	}

	status := store.TestModeStatus()
	if status.TestModeEnabled {
		t.Fatal("expected test mode disabled by default")
	}

	_, err = store.UpdateTestModeSettings(AdminTestModeSettingsRequest{
		Enabled:  true,
		Password: "test-password-123",
	})
	if err != nil {
		t.Fatal(err)
	}

	status = store.TestModeStatus()
	if !status.TestModeEnabled {
		t.Fatal("expected test mode enabled after update")
	}

	if !store.VerifyTestModePassword("test-password-123") {
		t.Fatal("expected password to verify")
	}
	if store.VerifyTestModePassword("wrong-password") {
		t.Fatal("expected wrong password to fail")
	}

	_, err = store.UpdateTestModeSettings(AdminTestModeSettingsRequest{
		Enabled: false,
	})
	if err != nil {
		t.Fatal(err)
	}

	status = store.TestModeStatus()
	if status.TestModeEnabled {
		t.Fatal("expected test mode disabled after turning off")
	}
	if store.VerifyTestModePassword("test-password-123") {
		t.Fatal("expected verification to fail when test mode is off")
	}
}

func TestAddTestIntegrationKey(t *testing.T) {
	tempDir := t.TempDir()
	cfg := Config{
		Environment:       "local",
		TokenSymbol:       defaultTokenSymbol,
		StatePath:         filepath.Join(tempDir, "state.json"),
		PlatformFeeBps:    1000,
		DevPaymentEnabled: true,
		DevPaymentCode:    "LOCAL-PAID",
		GitHubOwner:       defaultGitHubOwner,
		BountyRoot:        filepath.Join(tempDir, "bounties"),
		SMTPFrom:          "noreply@mergeos.local",
	}
	payments := NewPaymentManager(cfg)
	store, err := NewStore(cfg, payments, NewRepoFactory(cfg), NewEmailSender(cfg))
	if err != nil {
		t.Fatal(err)
	}

	key, err := store.AddTestIntegrationKey(AddTestIntegrationKeyRequest{
		Group:       "llm",
		DisplayName: "Test Gemini Key",
		KeyValues:   []KeyValuePair{{Name: "gemini_api_key", Value: "AIzaSyTest123456"}},
	})
	if err != nil {
		t.Fatalf("expected add to succeed: %v", err)
	}
	if key.ID == "" {
		t.Fatal("expected non-empty key ID")
	}

	keys := store.ListTestIntegrationKeys("llm")
	if len(keys) != 1 {
		t.Fatalf("expected 1 key, got %d", len(keys))
	}

	for _, kv := range keys[0].KeyValues {
		if kv.Value != "" && kv.Value != "****" && len(kv.Value) > 8 {
			if kv.Value[:4] != "AIza" {
				t.Fatalf("expected masked value, got %q", kv.Value)
			}
		}
	}
}

func TestAddIntegrationKeyRejectsBlockedName(t *testing.T) {
	tempDir := t.TempDir()
	cfg := Config{
		Environment:       "local",
		TokenSymbol:       defaultTokenSymbol,
		StatePath:         filepath.Join(tempDir, "state.json"),
		PlatformFeeBps:    1000,
		DevPaymentEnabled: true,
		DevPaymentCode:    "LOCAL-PAID",
		GitHubOwner:       defaultGitHubOwner,
		BountyRoot:        filepath.Join(tempDir, "bounties"),
		SMTPFrom:          "noreply@mergeos.local",
	}
	payments := NewPaymentManager(cfg)
	store, err := NewStore(cfg, payments, NewRepoFactory(cfg), NewEmailSender(cfg))
	if err != nil {
		t.Fatal(err)
	}

	_, err = store.AddTestIntegrationKey(AddTestIntegrationKeyRequest{
		Group:       "llm",
		DisplayName: "Blocked Key",
		KeyValues:   []KeyValuePair{{Name: "GITHUB_TOKEN", Value: "some-token"}},
	})
	if err == nil {
		t.Fatal("expected error for blocked ENV name")
	}
}
