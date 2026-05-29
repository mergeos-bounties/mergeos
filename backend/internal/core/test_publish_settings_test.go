package core

import (
    "encoding/json"
    "strings"
    "testing"
    "time"
)

func newTestStore(t testing.TB) *Store {
    t.Helper()
    cfg := Config{
        Environment: "test",
        StatePath:   "",
    }
    store := &Store{
        cfg:                cfg,
        nextID:             1,
        projects:           map[string]*Project{},
        tasks:              map[string]*Task{},
        users:              map[string]*User{},
        wallets:            map[string]*Wallet{},
        sessions:           map[string]*Session{},
        notifications:      map[string]*Notification{},
        attachments:        map[string]*Attachment{},
        sslReviews:         map[string]*SSLReviewStatus{},
        geminiAPIKeys:      map[string]*GeminiAPIKey{},
        geminiWebhookLogs:  map[string]*GeminiWebhookLog{},
        testSettingsConfig: TestSettingsConfig{},
        testSettingsEntries: map[string]*TestSettingsEntry{},
        adminSettings:      defaultAdminSettings(cfg),
        ledger:             []LedgerEntry{},
    }
    return store
}

func TestSettingValueMask(t *testing.T) {
    tests := []struct { input string; expected string }{
        {"", "****"},
        {"a", "a****a"},
        {"abcdefghi", "abcd****fghi"},
        {"sec_api_key_abc123xy", "sk_t****45xyz"},
    }
    for _, tt := range tests {
        got := SettingValueMask(tt.input)
        if got != tt.expected {
            t.Errorf("SettingValueMask(%q) = %q, want %q", tt.input, got, tt.expected)
        }
    }
}

func TestCheckForEnvCollision(t *testing.T) {
    for _, name := range []string{"GITHUB_TOKEN","ADMIN_EMAIL","ADMIN_PASSWORD","PAYPAL_CLIENT_ID","GEMINI_API_KEYS"} {
        if err := checkForEnvCollision(name); err == nil {
            t.Errorf("expected collision for %q", name)
        }
    }
    if err := checkForEnvCollision("MERGEOS_FOO"); err == nil {
        t.Error("expected collision for MERGEOS_* prefix")
    }
    if err := checkForEnvCollision("my-custom-key"); err != nil {
        t.Errorf("unexpected error: %v", err)
    }
}

func TestTestSettingsStore(t *testing.T) {
    store := newTestStore(t)
    defer store.Close()
    
    enabled := true
    _, err := store.UpdateTestSettingsConfig(UpdateTestSettingsRequest{
        TestModeEnabled: &enabled,
        TestPassword:    "test123",
    })
    if err != nil { t.Fatal(err) }
    if !store.GetTestSettingsConfig().TestModeEnabled { t.Error("expected enabled") }
    if !store.VerifyTestPassword("test123") { t.Error("password should match") }
    if store.VerifyTestPassword("wrong") { t.Error("wrong pw should fail") }

    entry, err := store.AddTestSettingsEntry(AddTestEntryRequest{
        IntegrationType: "llm", SettingKey: "OPENAI_KEY", SettingValue: "sk-proj-abc",
    })
    if err != nil { t.Fatal(err) }
    if entry.SettingValueHint != SettingValueMask("sk-proj-abc") {
        t.Errorf("bad mask: %q", entry.SettingValueHint)
    }

    _, err = store.AddTestSettingsEntry(AddTestEntryRequest{
        IntegrationType: "env", SettingKey: "GITHUB_TOKEN", SettingValue: "xxx",
    })
    if err == nil { t.Error("expected env collision error") }
}
