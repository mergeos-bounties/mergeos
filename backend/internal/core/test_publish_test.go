package core

import (
	"strings"
	"testing"
)

func TestIsBlockedKeyName(t *testing.T) {
	blocked := []string{
		"GITHUB_TOKEN", "github_token",
		"ADMIN_EMAIL", "admin_email",
		"PAYPAL_CLIENT_ID",
		"USDT_RECEIVER_ADDRESS",
		"GEMINI_API_KEYS",
		"MERGEOS_GITHUB_TOKEN",
		"mergeos_anything",
		"MERGEOS_FOO_BAR",
	}
	for _, name := range blocked {
		if !isBlockedKeyName(name) {
			t.Errorf("expected %q to be blocked", name)
		}
	}

	allowed := []string{
		"llm_test_keys",
		"paypal_sandbox_test_accounts",
		"usdt_test_receivers",
		"my_custom_llm_key",
		"openai_test_key",
	}
	for _, name := range allowed {
		if isBlockedKeyName(name) {
			t.Errorf("expected %q to be allowed", name)
		}
	}
}

func TestNormalizeIntegrationType(t *testing.T) {
	cases := map[string]string{
		"llm":            "llm",
		"LLM":            "llm",
		"paypal_sandbox": "paypal_sandbox",
		"PAYPAL_SANDBOX": "paypal_sandbox",
		"usdt_receiver":  "usdt_receiver",
		"invalid":        "",
		"":               "",
	}
	for input, want := range cases {
		got := normalizeIntegrationType(input)
		if got != want {
			t.Errorf("normalizeIntegrationType(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestTestPublishSettingHint(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"short", "****"},
		{"12345678", "1234...5678"},
		{"sk-ant-api-key-value-here", "sk-a...here"},
	}
	for _, c := range cases {
		got := testPublishSettingHint(c.input)
		if got != c.want {
			t.Errorf("testPublishSettingHint(%q) = %q, want %q", c.input, got, c.want)
		}
	}
}

func TestStoreTestPublishSettings(t *testing.T) {
	store := newTestStore(t)

	// Test mode disabled by default
	status := store.GetTestModeStatus()
	if status.Enabled {
		t.Fatal("test mode should be disabled by default")
	}

	// Set password
	_, err := store.SetTestModePassword("testpass123")
	if err != nil {
		t.Fatalf("SetTestModePassword: %v", err)
	}

	// Verify wrong password fails
	err = store.VerifyTestModePassword("wrongpassword")
	if err == nil {
		t.Fatal("expected error for wrong password")
	}

	// Enable test mode
	resp, err := store.SetTestModeEnabled(true)
	if err != nil {
		t.Fatalf("SetTestModeEnabled: %v", err)
	}
	if !resp.Enabled {
		t.Fatal("test mode should be enabled")
	}

	// Verify correct password succeeds
	err = store.VerifyTestModePassword("testpass123")
	if err != nil {
		t.Fatalf("VerifyTestModePassword: %v", err)
	}

	// Disable test mode — verify password now fails
	_, err = store.SetTestModeEnabled(false)
	if err != nil {
		t.Fatalf("SetTestModeEnabled(false): %v", err)
	}
	err = store.VerifyTestModePassword("testpass123")
	if err == nil {
		t.Fatal("expected error when test mode is disabled")
	}

	// Re-enable for CRUD tests
	store.SetTestModeEnabled(true)

	// Add LLM key
	llmReq := AddTestPublishSettingRequest{
		IntegrationType: "llm",
		DisplayName:     "OpenAI Test Key",
		KeyName:         "openai_test_key",
		KeyValue:        "sk-test-openai-key-value-here",
		Provider:        "openai",
	}
	llmSetting, err := store.AddTestPublishSetting(llmReq)
	if err != nil {
		t.Fatalf("AddTestPublishSetting LLM: %v", err)
	}
	if llmSetting.KeyHint == llmSetting.KeyValue {
		t.Fatal("key hint should not equal key value (secret must be masked)")
	}
	if strings.Contains(llmSetting.KeyHint, "sk-test-openai-key-value-here") {
		t.Fatal("full key value must not appear in stats response")
	}

	// Add PayPal sandbox entry
	paypalReq := AddTestPublishSettingRequest{
		IntegrationType: "paypal_sandbox",
		DisplayName:     "PayPal Sandbox Account 1",
		KeyName:         "paypal_sandbox_test_accounts",
		KeyValue:        "sandbox-secret-value",
		ClientID:        "AXy_sandbox_client_id",
	}
	paypalSetting, err := store.AddTestPublishSetting(paypalReq)
	if err != nil {
		t.Fatalf("AddTestPublishSetting PayPal: %v", err)
	}
	if paypalSetting.IntegrationType != "paypal_sandbox" {
		t.Fatal("wrong integration type stored")
	}

	// Add USDT receiver
	usdtReq := AddTestPublishSettingRequest{
		IntegrationType: "usdt_receiver",
		DisplayName:     "USDT Test Receiver",
		KeyName:         "usdt_test_receivers",
		KeyValue:        "0xTestReceiverAddress",
		ReceiverAddress: "0xTestReceiverAddress",
	}
	usdtSetting, err := store.AddTestPublishSetting(usdtReq)
	if err != nil {
		t.Fatalf("AddTestPublishSetting USDT: %v", err)
	}

	// List all settings
	all := store.ListTestPublishSettings("")
	if len(all) != 3 {
		t.Fatalf("expected 3 settings, got %d", len(all))
	}

	// List by type
	llmOnly := store.ListTestPublishSettings("llm")
	if len(llmOnly) != 1 {
		t.Fatalf("expected 1 LLM setting, got %d", len(llmOnly))
	}

	// Reject blocked key name
	_, err = store.AddTestPublishSetting(AddTestPublishSettingRequest{
		IntegrationType: "llm",
		KeyName:         "GITHUB_TOKEN",
		KeyValue:        "some-value",
		Provider:        "openai",
	})
	if err == nil {
		t.Fatal("expected error for blocked key name GITHUB_TOKEN")
	}

	// Reject mergeos_ prefix
	_, err = store.AddTestPublishSetting(AddTestPublishSettingRequest{
		IntegrationType: "llm",
		KeyName:         "MERGEOS_CUSTOM_KEY",
		KeyValue:        "some-value",
		Provider:        "openai",
	})
	if err == nil {
		t.Fatal("expected error for MERGEOS_* key name")
	}

	// Disable a setting
	updated, err := store.UpdateTestPublishSetting(llmSetting.ID, TestPublishSettingStatusDisabled)
	if err != nil {
		t.Fatalf("UpdateTestPublishSetting: %v", err)
	}
	if updated.Status != TestPublishSettingStatusDisabled {
		t.Fatal("setting should be disabled")
	}

	// Runtime resolution excludes disabled
	runtime := store.GetTestPublishSettingsForRuntime("llm")
	if len(runtime) != 0 {
		t.Fatal("disabled setting should not appear in runtime resolution")
	}

	// Delete
	if err := store.DeleteTestPublishSetting(paypalSetting.ID); err != nil {
		t.Fatalf("DeleteTestPublishSetting: %v", err)
	}
	if err := store.DeleteTestPublishSetting(usdtSetting.ID); err != nil {
		t.Fatalf("DeleteTestPublishSetting: %v", err)
	}
	remaining := store.ListTestPublishSettings("")
	if len(remaining) != 1 {
		t.Fatalf("expected 1 remaining setting, got %d", len(remaining))
	}
}

func TestTestPublishSettingPasswordValidation(t *testing.T) {
	store := newTestStore(t)

	// Too short
	_, err := store.SetTestModePassword("short")
	if err == nil {
		t.Fatal("expected error for short password")
	}

	// Empty
	_, err = store.SetTestModePassword("")
	if err == nil {
		t.Fatal("expected error for empty password")
	}

	// Valid
	_, err = store.SetTestModePassword("validpassword123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestTestPublishSettingArrayPersistence(t *testing.T) {
	store := newTestStore(t)
	store.SetTestModeEnabled(true)
	store.SetTestModePassword("testpass123")

	// Add 2 LLM keys
	for i, name := range []string{"openai_key_one", "anthropic_key_two"} {
		provider := "openai"
		if i == 1 {
			provider = "anthropic"
		}
		_, err := store.AddTestPublishSetting(AddTestPublishSettingRequest{
			IntegrationType: "llm",
			KeyName:         name,
			KeyValue:        "sk-test-value-" + name,
			Provider:        provider,
		})
		if err != nil {
			t.Fatalf("AddTestPublishSetting: %v", err)
		}
	}

	settings := store.ListTestPublishSettings("llm")
	if len(settings) != 2 {
		t.Fatalf("expected 2 LLM settings, got %d", len(settings))
	}

	// Verify secrets are masked in list response
	for _, s := range settings {
		if s.KeyHint == "" || len(s.KeyHint) > 20 {
			t.Errorf("key hint looks wrong: %q", s.KeyHint)
		}
	}
}
