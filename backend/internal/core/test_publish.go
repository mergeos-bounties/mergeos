package core

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"sort"
	"strings"
	"time"
)

const (
	TestPublishSettingStatusActive   = "active"
	TestPublishSettingStatusDisabled = "disabled"

	IntegrationTypeLLM          = "llm"
	IntegrationTypePayPalSandbox = "paypal_sandbox"
	IntegrationTypeUSDTReceiver  = "usdt_receiver"
)

// blockedKeyNames lists ENV/config key names that must never be used as
// database-backed test setting names to prevent collision with production config.
var blockedKeyNames = map[string]struct{}{
	"github_token":                {},
	"mergeos_github_token":        {},
	"token_symbol":                {},
	"admin_email":                 {},
	"admin_password":              {},
	"paypal_client_id":            {},
	"paypal_client_secret":        {},
	"paypal_environment":          {},
	"crypto_webhook_secret":       {},
	"crypto_token_contract":       {},
	"usdt_receiver_address":       {},
	"gemini_api_keys":             {},
	"database_url":                {},
	"platform_fee_bps":            {},
	"dev_payment_enabled":         {},
	"dev_payment_code":            {},
	"admin_name":                  {},
	"admin_company_name":          {},
	"admin_auto_promote":          {},
	"primary_domain":              {},
	"admin_domain":                {},
	"scan_domain":                 {},
	"ssl_review_enabled":          {},
	"ssl_review_domains":          {},
	"crypto_rpc_url":              {},
	"crypto_receiver":             {},
	"crypto_asset":                {},
	"crypto_token_decimals":       {},
	"crypto_wei_per_usd_cent":     {},
	"crypto_min_confirmations":    {},
	"gemini_review_model":         {},
	"gemini_review_webhook_secret": {},
	"gemini_review_max_patch_bytes": {},
	"github_app_id":               {},
	"github_oauth_client_id":      {},
	"github_oauth_client_secret":  {},
	"bounty_root":                 {},
	"upload_root":                 {},
	"smtp_host":                   {},
	"smtp_port":                   {},
	"smtp_username":               {},
	"smtp_password":               {},
	"smtp_from":                   {},
	"google_client_id":            {},
	"google_client_secret":        {},
	"github_client_id":            {},
	"github_client_secret":        {},
}

var validIntegrationTypes = map[string]struct{}{
	IntegrationTypeLLM:          {},
	IntegrationTypePayPalSandbox: {},
	IntegrationTypeUSDTReceiver:  {},
}

func isBlockedKeyName(name string) bool {
	normalized := strings.ToLower(strings.TrimSpace(name))
	if _, blocked := blockedKeyNames[normalized]; blocked {
		return true
	}
	// Block anything that starts with mergeos_ to protect all MERGEOS_* env vars
	if strings.HasPrefix(normalized, "mergeos_") {
		return true
	}
	return false
}

func normalizeIntegrationType(t string) string {
	t = strings.ToLower(strings.TrimSpace(t))
	if _, ok := validIntegrationTypes[t]; ok {
		return t
	}
	return ""
}

func testPublishSettingID(integrationType, keyName string) string {
	sum := sha256.Sum256([]byte(integrationType + ":" + strings.TrimSpace(keyName) + ":" + time.Now().UTC().Format(time.RFC3339Nano)))
	return hex.EncodeToString(sum[:])[:24]
}

func testPublishSettingHint(value string) string {
	value = strings.TrimSpace(value)
	if len(value) <= 8 {
		return "****"
	}
	return value[:4] + "..." + value[len(value)-4:]
}

// Admin test mode control

func (s *Store) GetTestModeStatus() TestModeStatusResponse {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return TestModeStatusResponse{
		Enabled:         s.adminSettings.TestModeEnabled,
		PasswordIsSet:   s.adminSettings.TestModePasswordHash != "",
		UpdatedAt:       s.adminSettings.UpdatedAt,
	}
}

func (s *Store) SetTestModeEnabled(enabled bool) (TestModeStatusResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.adminSettings.TestModeEnabled = enabled
	s.adminSettings.UpdatedAt = time.Now().UTC()
	if err := s.saveLocked(); err != nil {
		return TestModeStatusResponse{}, err
	}
	return TestModeStatusResponse{
		Enabled:       s.adminSettings.TestModeEnabled,
		PasswordIsSet: s.adminSettings.TestModePasswordHash != "",
		UpdatedAt:     s.adminSettings.UpdatedAt,
	}, nil
}

func (s *Store) SetTestModePassword(password string) (TestModeStatusResponse, error) {
	password = strings.TrimSpace(password)
	if password == "" {
		return TestModeStatusResponse{}, errors.New("test mode password is required")
	}
	if len(password) < 8 {
		return TestModeStatusResponse{}, errors.New("test mode password must be at least 8 characters")
	}
	salt, hash, err := hashPassword(password)
	if err != nil {
		return TestModeStatusResponse{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.adminSettings.TestModePasswordHash = hash
	s.adminSettings.TestModePasswordSalt = salt
	s.adminSettings.UpdatedAt = time.Now().UTC()
	if err := s.saveLocked(); err != nil {
		return TestModeStatusResponse{}, err
	}
	return TestModeStatusResponse{
		Enabled:       s.adminSettings.TestModeEnabled,
		PasswordIsSet: true,
		UpdatedAt:     s.adminSettings.UpdatedAt,
	}, nil
}

// VerifyTestModePassword checks the supplied password against the stored hash.
// Returns an error if test mode is disabled or the password is wrong.
func (s *Store) VerifyTestModePassword(password string) error {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if !s.adminSettings.TestModeEnabled {
		return errors.New("test mode is not enabled")
	}
	if s.adminSettings.TestModePasswordHash == "" {
		return errors.New("test mode password has not been set")
	}
	if !verifyPassword(password, s.adminSettings.TestModePasswordSalt, s.adminSettings.TestModePasswordHash) {
		return errors.New("invalid test mode password")
	}
	return nil
}

// Test publish settings CRUD

func (s *Store) ListTestPublishSettings(integrationType string) []TestPublishSettingStats {
	s.mu.RLock()
	defer s.mu.RUnlock()
	integrationType = normalizeIntegrationType(integrationType)
	result := make([]TestPublishSettingStats, 0)
	for _, setting := range s.testPublishSettings {
		if integrationType != "" && setting.IntegrationType != integrationType {
			continue
		}
		result = append(result, testPublishSettingStats(setting))
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].IntegrationType != result[j].IntegrationType {
			return result[i].IntegrationType < result[j].IntegrationType
		}
		return result[i].CreatedAt.Before(result[j].CreatedAt)
	})
	return result
}

func (s *Store) AddTestPublishSetting(req AddTestPublishSettingRequest) (TestPublishSettingStats, error) {
	integrationType := normalizeIntegrationType(req.IntegrationType)
	if integrationType == "" {
		return TestPublishSettingStats{}, errors.New("invalid integration type: must be llm, paypal_sandbox, or usdt_receiver")
	}
	keyName := strings.TrimSpace(req.KeyName)
	if keyName == "" {
		return TestPublishSettingStats{}, errors.New("key name is required")
	}
	if isBlockedKeyName(keyName) {
		return TestPublishSettingStats{}, errors.New("key name conflicts with a reserved environment variable name")
	}
	keyValue := strings.TrimSpace(req.KeyValue)
	if keyValue == "" {
		return TestPublishSettingStats{}, errors.New("key value is required")
	}
	displayName := strings.TrimSpace(req.DisplayName)
	if displayName == "" {
		displayName = keyName
	}

	if err := validateTestPublishSettingFields(integrationType, req); err != nil {
		return TestPublishSettingStats{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.testPublishSettings == nil {
		s.testPublishSettings = map[string]*TestPublishSetting{}
	}
	// Check for duplicate key name within same integration type
	for _, existing := range s.testPublishSettings {
		if existing.IntegrationType == integrationType && strings.EqualFold(existing.KeyName, keyName) {
			return TestPublishSettingStats{}, errors.New("a setting with this key name already exists for this integration type")
		}
	}
	now := time.Now().UTC()
	id := testPublishSettingID(integrationType, keyName)
	setting := &TestPublishSetting{
		ID:              id,
		IntegrationType: integrationType,
		DisplayName:     displayName,
		KeyName:         keyName,
		KeyValue:        keyValue,
		KeyHint:         testPublishSettingHint(keyValue),
		Status:          TestPublishSettingStatusActive,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	s.testPublishSettings[id] = setting
	if err := s.saveLocked(); err != nil {
		return TestPublishSettingStats{}, err
	}
	return testPublishSettingStats(setting), nil
}

func (s *Store) UpdateTestPublishSetting(id, status string) (TestPublishSettingStats, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return TestPublishSettingStats{}, errors.New("setting id is required")
	}
	normalizedStatus := strings.ToLower(strings.TrimSpace(status))
	if normalizedStatus != "" && normalizedStatus != TestPublishSettingStatusActive && normalizedStatus != TestPublishSettingStatusDisabled {
		return TestPublishSettingStats{}, errors.New("invalid status: must be active or disabled")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	setting := s.testPublishSettings[id]
	if setting == nil {
		return TestPublishSettingStats{}, errors.New("setting not found")
	}
	if normalizedStatus != "" {
		setting.Status = normalizedStatus
	}
	setting.UpdatedAt = time.Now().UTC()
	if err := s.saveLocked(); err != nil {
		return TestPublishSettingStats{}, err
	}
	return testPublishSettingStats(setting), nil
}

func (s *Store) DeleteTestPublishSetting(id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return errors.New("setting id is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.testPublishSettings == nil || s.testPublishSettings[id] == nil {
		return errors.New("setting not found")
	}
	delete(s.testPublishSettings, id)
	return s.saveLocked()
}

// GetTestPublishSettingsForRuntime returns active test settings for use by
// test-mode code paths. Raw key values are returned only here (never in API responses).
func (s *Store) GetTestPublishSettingsForRuntime(integrationType string) []TestPublishSetting {
	s.mu.RLock()
	defer s.mu.RUnlock()
	integrationType = normalizeIntegrationType(integrationType)
	result := make([]TestPublishSetting, 0)
	for _, setting := range s.testPublishSettings {
		if setting.Status != TestPublishSettingStatusActive {
			continue
		}
		if integrationType != "" && setting.IntegrationType != integrationType {
			continue
		}
		result = append(result, *setting)
	}
	return result
}

func validateTestPublishSettingFields(integrationType string, req AddTestPublishSettingRequest) error {
	switch integrationType {
	case IntegrationTypeLLM:
		if strings.TrimSpace(req.Provider) == "" {
			return errors.New("provider is required for LLM keys")
		}
	case IntegrationTypePayPalSandbox:
		if strings.TrimSpace(req.ClientID) == "" {
			return errors.New("client_id is required for PayPal sandbox entries")
		}
	case IntegrationTypeUSDTReceiver:
		if strings.TrimSpace(req.ReceiverAddress) == "" {
			return errors.New("receiver_address is required for USDT receiver entries")
		}
	}
	return nil
}

func testPublishSettingStats(s *TestPublishSetting) TestPublishSettingStats {
	if s == nil {
		return TestPublishSettingStats{}
	}
	return TestPublishSettingStats{
		ID:              s.ID,
		IntegrationType: s.IntegrationType,
		DisplayName:     s.DisplayName,
		KeyName:         s.KeyName,
		KeyHint:         s.KeyHint,
		Status:          s.Status,
		CreatedAt:       s.CreatedAt,
		UpdatedAt:       s.UpdatedAt,
		LastUsedAt:      cloneTimePtr(s.LastUsedAt),
	}
}
