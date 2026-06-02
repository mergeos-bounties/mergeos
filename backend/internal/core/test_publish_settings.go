package core

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

type TestSettingsConfig struct {
	TestModeEnabled  bool      `json:"test_mode_enabled"`
	TestPasswordHash string    `json:"test_password_hash,omitempty"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type TestSettingsConfigResponse struct {
	TestModeEnabled bool      `json:"test_mode_enabled"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type TestSettingsEntry struct {
	ID              string            `json:"id"`
	IntegrationType string            `json:"integration_type"`
	DisplayName     string            `json:"display_name"`
	SettingKey      string            `json:"setting_key"`
	SettingValue    string            `json:"-"`
	KeyValueMap     map[string]string `json:"key_value_map"`
	Status          string            `json:"status"`
	LastUsedAt      *time.Time        `json:"last_used_at,omitempty"`
	CreatedAt       time.Time         `json:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
}

type UpdateTestSettingsRequest struct {
	TestModeEnabled *bool  `json:"test_mode_enabled,omitempty"`
	TestPassword    string `json:"test_password,omitempty"`
}

type AddTestEntryRequest struct {
	IntegrationType string            `json:"integration_type"`
	DisplayName     string            `json:"display_name"`
	SettingKey      string            `json:"setting_key"`
	SettingValue    string            `json:"setting_value"`
	KeyValueMap     map[string]string `json:"key_value_map"`
}

type UpdateTestEntryRequest struct {
	DisplayName  string            `json:"display_name,omitempty"`
	SettingKey   string            `json:"setting_key,omitempty"`
	SettingValue string            `json:"setting_value,omitempty"`
	KeyValueMap  map[string]string `json:"key_value_map,omitempty"`
	Status       string            `json:"status,omitempty"`
}

type PublicTestSettingsRequest struct {
	Password string `json:"password"`
}

type TestSettingsEntryResponse struct {
	ID               string            `json:"id"`
	IntegrationType  string            `json:"integration_type"`
	DisplayName      string            `json:"display_name"`
	SettingKey       string            `json:"setting_key"`
	SettingValueHint string            `json:"setting_value_hint"`
	KeyValueMap      map[string]string `json:"key_value_map"`
	Status           string            `json:"status"`
	LastUsedAt       *time.Time        `json:"last_used_at,omitempty"`
	CreatedAt        time.Time         `json:"created_at"`
	UpdatedAt        time.Time         `json:"updated_at"`
}

type TestSettingsEntrySecretResponse struct {
	ID              string            `json:"id"`
	IntegrationType string            `json:"integration_type"`
	DisplayName     string            `json:"display_name"`
	SettingKey      string            `json:"setting_key"`
	SettingValue    string            `json:"setting_value"`
	KeyValueMap     map[string]string `json:"key_value_map"`
	Status          string            `json:"status"`
	LastUsedAt      *time.Time        `json:"last_used_at,omitempty"`
}

func SettingValueMask(value string) string {
	if len(value) <= 8 {
		if len(value) == 0 {
			return "****"
		}
		return value[:1] + "****" + value[len(value)-1:]
	}
	return value[:4] + "****" + value[len(value)-4:]
}

func maskKeyValueMap(m map[string]string) map[string]string {
	if m == nil {
		return nil
	}
	masked := make(map[string]string, len(m))
	for k, v := range m {
		masked[k] = SettingValueMask(v)
	}
	return masked
}

type passwordAttemptTracker struct {
	mu       sync.Mutex
	attempts map[string]*attemptRecord
}

type attemptRecord struct {
	count      int
	firstAt    time.Time
	unlockedAt time.Time
}

const (
	maxPasswordAttempts     = 5
	passwordWindow          = 15 * time.Minute
	passwordLockoutDuration = 30 * time.Minute
)

var globalPasswordTracker = &passwordAttemptTracker{
	attempts: make(map[string]*attemptRecord),
}

func checkPasswordRateLimit(ip string) bool {
	globalPasswordTracker.mu.Lock()
	defer globalPasswordTracker.mu.Unlock()
	now := time.Now().UTC()
	rec, ok := globalPasswordTracker.attempts[ip]
	if !ok {
		return true
	}
	if now.Before(rec.unlockedAt) {
		return false
	}
	if now.Sub(rec.firstAt) > passwordWindow {
		delete(globalPasswordTracker.attempts, ip)
		return true
	}
	if rec.count >= maxPasswordAttempts {
		rec.unlockedAt = now.Add(passwordLockoutDuration)
		return false
	}
	return true
}

func recordPasswordAttempt(ip string, success bool) {
	globalPasswordTracker.mu.Lock()
	defer globalPasswordTracker.mu.Unlock()
	now := time.Now().UTC()
	if success {
		delete(globalPasswordTracker.attempts, ip)
		return
	}
	rec, ok := globalPasswordTracker.attempts[ip]
	if !ok {
		globalPasswordTracker.attempts[ip] = &attemptRecord{count: 1, firstAt: now}
		return
	}
	if now.Sub(rec.firstAt) > passwordWindow {
		globalPasswordTracker.attempts[ip] = &attemptRecord{count: 1, firstAt: now}
		return
	}
	rec.count++
	if rec.count >= maxPasswordAttempts {
		rec.unlockedAt = now.Add(passwordLockoutDuration)
	}
}

func testSettingsConfigResponse(config TestSettingsConfig) TestSettingsConfigResponse {
	return TestSettingsConfigResponse{
		TestModeEnabled: config.TestModeEnabled,
		UpdatedAt:       config.UpdatedAt,
	}
}

func testSettingsRateLimitKey(r *http.Request) string {
	remoteIP := normalizedClientIP(r.RemoteAddr)
	if trustedProxyIP(remoteIP) {
		if forwarded := firstForwardedClientIP(r.Header.Get("X-Forwarded-For")); forwarded != "" {
			return forwarded
		}
		if realIP := normalizedClientIP(r.Header.Get("X-Real-IP")); realIP != "" {
			return realIP
		}
	}
	return remoteIP
}

func firstForwardedClientIP(value string) string {
	for _, part := range strings.Split(value, ",") {
		if ip := normalizedClientIP(part); ip != "" {
			return ip
		}
	}
	return ""
}

func normalizedClientIP(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if host, _, err := net.SplitHostPort(value); err == nil {
		value = host
	}
	value = strings.Trim(value, "[]")
	ip := net.ParseIP(value)
	if ip == nil {
		return ""
	}
	return ip.String()
}

func trustedProxyIP(value string) bool {
	ip := net.ParseIP(value)
	return ip != nil && (ip.IsLoopback() || ip.IsPrivate())
}

var knownEnvNames = []string{
	"ADMIN_AUTO_PROMOTE_FIRST_USER", "ADMIN_COMPANY_NAME", "ADMIN_DOMAIN", "ADMIN_EMAIL", "ADMIN_NAME", "ADMIN_PASSWORD",
	"BOUNTY_ROOT", "DATABASE_URL", "DEV_PAYMENT_CODE", "DEV_PAYMENT_ENABLED",
	"GEMINI_API_KEY", "GEMINI_API_KEYS", "GEMINI_REVIEW_MAX_PATCH_BYTES", "GEMINI_REVIEW_MODEL", "GEMINI_REVIEW_WEBHOOK_SECRET",
	"GITHUB_APP_CLIENT_ID", "GITHUB_APP_CLIENT_SECRET", "GITHUB_APP_ID", "GITHUB_CLIENT_ID", "GITHUB_CLIENT_SECRET",
	"GITHUB_OAUTH_CLIENT_ID", "GITHUB_OAUTH_CLIENT_SECRET", "GITHUB_OWNER", "GITHUB_OWNER_TYPE", "GITHUB_TOKEN",
	"GOOGLE_CLIENT_ID", "GOOGLE_CLIENT_SECRET",
	"MERGEOS_ENV", "MERGEOS_GEMINI_API_KEY", "MERGEOS_GEMINI_API_KEYS", "MERGEOS_GEMINI_REVIEW_WEBHOOK_SECRET",
	"MERGEOS_GITHUB_APP_CLIENT_ID", "MERGEOS_GITHUB_APP_CLIENT_SECRET", "MERGEOS_GITHUB_APP_ID",
	"MERGEOS_GITHUB_OAUTH_CLIENT_ID", "MERGEOS_GITHUB_OAUTH_CLIENT_SECRET", "MERGEOS_GITHUB_TOKEN",
	"MERGEOS_GOOGLE_CLIENT_ID", "MERGEOS_GOOGLE_CLIENT_SECRET", "MERGEOS_STATE_PATH",
	"PAYPAL_CLIENT_ID", "PAYPAL_CLIENT_SECRET", "PAYPAL_ENV", "PAYPAL_ENVIRONMENT",
	"PLATFORM_FEE_BPS", "PRIMARY_DOMAIN", "SCAN_DOMAIN",
	"CRYPTO_ASSET", "CRYPTO_MIN_CONFIRMATIONS", "CRYPTO_RECEIVER", "CRYPTO_RPC_URL", "CRYPTO_TOKEN_CONTRACT",
	"CRYPTO_TOKEN_DECIMALS", "CRYPTO_WEBHOOK_SECRET", "CRYPTO_WEI_PER_USD_CENT",
	"SMTP_FROM", "SMTP_HOST", "SMTP_PASSWORD", "SMTP_PORT", "SMTP_USERNAME",
	"SSL_EXPIRY_WARN_DAYS", "SSL_REVIEW_DOMAINS", "SSL_REVIEW_ENABLED", "SSL_REVIEW_INTERVAL_MINUTES",
	"TOKEN_SYMBOL", "UPLOAD_ROOT",
	"USDT_NETWORK", "USDT_PROVIDER_API_KEY", "USDT_PROVIDER_SECRET", "USDT_PROVIDER_URL", "USDT_RECEIVER", "USDT_RECEIVER_ADDRESS", "USDT_WEBHOOK_SECRET",
}

var runtimeEnvNames map[string]bool
var runtimeEnvOnce sync.Once

func getRuntimeEnvNames() map[string]bool {
	runtimeEnvOnce.Do(func() {
		runtimeEnvNames = make(map[string]bool)
		for _, e := range os.Environ() {
			parts := strings.SplitN(e, "=", 2)
			if len(parts) > 0 && strings.TrimSpace(parts[0]) != "" {
				runtimeEnvNames[strings.ToUpper(strings.TrimSpace(parts[0]))] = true
			}
		}
	})
	return runtimeEnvNames
}

func checkForEnvCollision(key string, keyValueMapKeys []string) error {
	upper := strings.ToUpper(strings.TrimSpace(key))
	if upper == "" {
		return nil
	}
	envNames := getRuntimeEnvNames()
	if envNames[upper] {
		return fmt.Errorf("setting key %q collides with actual runtime environment variable %s", key, upper)
	}
	for _, known := range knownEnvNames {
		if upper == known {
			return fmt.Errorf("setting key %q collides with known environment variable %s", key, known)
		}
	}
	if strings.HasPrefix(upper, "MERGEOS_") {
		return fmt.Errorf("setting key %q collides with MERGEOS_* environment variable prefix", key)
	}
	for _, nestedKey := range keyValueMapKeys {
		nestedUpper := strings.ToUpper(strings.TrimSpace(nestedKey))
		if nestedUpper == "" {
			continue
		}
		if envNames[nestedUpper] {
			return fmt.Errorf("key_value_map key %q collides with actual runtime environment variable %s", nestedKey, nestedUpper)
		}
		for _, known := range knownEnvNames {
			if nestedUpper == known {
				return fmt.Errorf("key_value_map key %q collides with known environment variable %s", nestedKey, known)
			}
		}
		if strings.HasPrefix(nestedUpper, "MERGEOS_") {
			return fmt.Errorf("key_value_map key %q collides with MERGEOS_* environment variable prefix", nestedKey)
		}
	}
	return nil
}

func (s *Store) GetTestSettingsConfig() TestSettingsConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.testSettingsConfig
}

func (s *Store) UpdateTestSettingsConfig(req UpdateTestSettingsRequest) (TestSettingsConfig, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if req.TestModeEnabled != nil {
		s.testSettingsConfig.TestModeEnabled = *req.TestModeEnabled
	}
	if strings.TrimSpace(req.TestPassword) != "" {
		salt, hash, err := hashPassword(req.TestPassword)
		if err != nil {
			return TestSettingsConfig{}, err
		}
		s.testSettingsConfig.TestPasswordHash = salt + ":" + hash
	}
	s.testSettingsConfig.UpdatedAt = time.Now().UTC()
	if err := s.saveLocked(); err != nil {
		return TestSettingsConfig{}, err
	}
	return s.testSettingsConfig, nil
}

func (s *Store) VerifyTestPassword(password string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.testSettingsConfig.TestPasswordHash == "" {
		return false
	}
	parts := strings.SplitN(s.testSettingsConfig.TestPasswordHash, ":", 2)
	if len(parts) != 2 {
		return false
	}
	return verifyPassword(password, parts[0], parts[1])
}

func (s *Store) ListTestSettingsEntries() []*TestSettingsEntryResponse {
	s.mu.RLock()
	defer s.mu.RUnlock()
	responses := make([]*TestSettingsEntryResponse, 0, len(s.testSettingsEntries))
	for _, entry := range s.testSettingsEntries {
		responses = append(responses, &TestSettingsEntryResponse{
			ID: entry.ID, IntegrationType: entry.IntegrationType,
			DisplayName: entry.DisplayName, SettingKey: entry.SettingKey,
			SettingValueHint: SettingValueMask(entry.SettingValue),
			KeyValueMap:      maskKeyValueMap(entry.KeyValueMap),
			Status:           entry.Status, LastUsedAt: entry.LastUsedAt,
			CreatedAt: entry.CreatedAt, UpdatedAt: entry.UpdatedAt,
		})
	}
	return responses
}

var allowedIntegrationTypes = map[string]bool{
	"llm":    true,
	"paypal": true,
	"usdt":   true,
}

var allowedEntryStatuses = map[string]bool{
	"active":   true,
	"disabled": true,
}

func (s *Store) AddTestSettingsEntry(req AddTestEntryRequest) (*TestSettingsEntryResponse, error) {
	if strings.TrimSpace(req.IntegrationType) == "" {
		return nil, errors.New("integration_type is required")
	}
	if !allowedIntegrationTypes[strings.ToLower(strings.TrimSpace(req.IntegrationType))] {
		return nil, fmt.Errorf("invalid integration_type %q: must be one of llm, paypal, usdt", req.IntegrationType)
	}
	if strings.TrimSpace(req.SettingKey) == "" {
		return nil, errors.New("setting_key is required")
	}
	if strings.TrimSpace(req.SettingValue) == "" {
		return nil, errors.New("setting_value is required")
	}
	var kvMapKeys []string
	for k := range req.KeyValueMap {
		kvMapKeys = append(kvMapKeys, k)
	}
	if err := checkForEnvCollision(req.SettingKey, kvMapKeys); err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	entry := &TestSettingsEntry{
		ID: s.newID("tse"), IntegrationType: strings.TrimSpace(req.IntegrationType),
		DisplayName: strings.TrimSpace(req.DisplayName), SettingKey: strings.TrimSpace(req.SettingKey),
		SettingValue: req.SettingValue, KeyValueMap: req.KeyValueMap,
		Status: "active", CreatedAt: now, UpdatedAt: now,
	}
	if entry.KeyValueMap == nil {
		entry.KeyValueMap = map[string]string{}
	}
	s.testSettingsEntries[entry.ID] = entry
	if err := s.saveLocked(); err != nil {
		return nil, err
	}
	return entryToResponse(entry), nil
}

func (s *Store) UpdateTestSettingsEntry(id string, req UpdateTestEntryRequest) (*TestSettingsEntryResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.testSettingsEntries[strings.TrimSpace(id)]
	if !ok {
		return nil, errors.New("test settings entry not found")
	}
	if strings.TrimSpace(req.DisplayName) != "" {
		entry.DisplayName = strings.TrimSpace(req.DisplayName)
	}
	if strings.TrimSpace(req.SettingKey) != "" {
		var kvMapKeys []string
		for k := range entry.KeyValueMap {
			kvMapKeys = append(kvMapKeys, k)
		}
		for k := range req.KeyValueMap {
			kvMapKeys = append(kvMapKeys, k)
		}
		if err := checkForEnvCollision(req.SettingKey, kvMapKeys); err != nil {
			return nil, err
		}
		entry.SettingKey = strings.TrimSpace(req.SettingKey)
	}
	if strings.TrimSpace(req.SettingValue) != "" {
		entry.SettingValue = req.SettingValue
	}
	if req.KeyValueMap != nil {
		if entry.KeyValueMap == nil {
			entry.KeyValueMap = map[string]string{}
		}
		// Check nested key_value_map keys for ENV collision even when setting_key unchanged
		var mergedKeys []string
		for k := range entry.KeyValueMap {
			mergedKeys = append(mergedKeys, k)
		}
		for k := range req.KeyValueMap {
			mergedKeys = append(mergedKeys, k)
		}
		if err := checkForEnvCollision(entry.SettingKey, mergedKeys); err != nil {
			return nil, err
		}
		for k, v := range req.KeyValueMap {
			entry.KeyValueMap[k] = v
		}
	}
	if strings.TrimSpace(req.Status) != "" {
		if !allowedEntryStatuses[strings.ToLower(strings.TrimSpace(req.Status))] {
			return nil, fmt.Errorf("invalid status %q: must be one of active, disabled", req.Status)
		}
		entry.Status = strings.TrimSpace(req.Status)
	}
	entry.UpdatedAt = time.Now().UTC()
	if err := s.saveLocked(); err != nil {
		return nil, err
	}
	return entryToResponse(entry), nil
}

func (s *Store) DeleteTestSettingsEntry(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	id = strings.TrimSpace(id)
	if _, ok := s.testSettingsEntries[id]; !ok {
		return errors.New("test settings entry not found")
	}
	delete(s.testSettingsEntries, id)
	return s.saveLocked()
}

func (s *Store) RevealTestSettingsEntry(id string) (*TestSettingsEntrySecretResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.testSettingsEntries[strings.TrimSpace(id)]
	if !ok {
		return nil, errors.New("test settings entry not found")
	}
	if strings.EqualFold(entry.Status, "disabled") {
		return nil, errors.New("test settings entry is disabled")
	}
	now := time.Now().UTC()
	entry.LastUsedAt = &now
	entry.UpdatedAt = now
	if err := s.saveLocked(); err != nil {
		return nil, err
	}
	kv := make(map[string]string, len(entry.KeyValueMap))
	for k, v := range entry.KeyValueMap {
		kv[k] = v
	}
	return &TestSettingsEntrySecretResponse{
		ID: entry.ID, IntegrationType: entry.IntegrationType,
		DisplayName: entry.DisplayName, SettingKey: entry.SettingKey,
		SettingValue: entry.SettingValue, KeyValueMap: kv,
		Status: entry.Status, LastUsedAt: entry.LastUsedAt,
	}, nil
}

func entryToResponse(entry *TestSettingsEntry) *TestSettingsEntryResponse {
	return &TestSettingsEntryResponse{
		ID: entry.ID, IntegrationType: entry.IntegrationType,
		DisplayName: entry.DisplayName, SettingKey: entry.SettingKey,
		SettingValueHint: SettingValueMask(entry.SettingValue),
		KeyValueMap:      maskKeyValueMap(entry.KeyValueMap),
		Status:           entry.Status, LastUsedAt: entry.LastUsedAt,
		CreatedAt: entry.CreatedAt, UpdatedAt: entry.UpdatedAt,
	}
}

func (s *Server) adminGetTestSettings(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	writeJSON(w, http.StatusOK, testSettingsConfigResponse(s.store.GetTestSettingsConfig()))
}

func (s *Server) adminUpdateTestSettings(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	var req UpdateTestSettingsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	config, err := s.store.UpdateTestSettingsConfig(req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, testSettingsConfigResponse(config))
}

func (s *Server) adminListTestEntries(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	writeJSON(w, http.StatusOK, s.store.ListTestSettingsEntries())
}

func (s *Server) adminAddTestEntry(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	var req AddTestEntryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	entry, err := s.store.AddTestSettingsEntry(req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, entry)
}

func (s *Server) adminUpdateTestEntry(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	var req UpdateTestEntryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	entry, err := s.store.UpdateTestSettingsEntry(r.PathValue("id"), req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, entry)
}

func (s *Server) adminDeleteTestEntry(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	if err := s.store.DeleteTestSettingsEntry(r.PathValue("id")); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) requireTestAuthBody(w http.ResponseWriter, r *http.Request) bool {
	rateLimitKey := testSettingsRateLimitKey(r)
	if !checkPasswordRateLimit(rateLimitKey) {
		writeError(w, http.StatusTooManyRequests, "too many failed password attempts; try again later")
		return false
	}
	cfg := s.store.GetTestSettingsConfig()
	if !cfg.TestModeEnabled {
		writeError(w, http.StatusForbidden, "test mode is disabled")
		return false
	}
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "failed to read request body")
		return false
	}
	var req PublicTestSettingsRequest
	if err := json.Unmarshal(bodyBytes, &req); err != nil {
		writeError(w, http.StatusUnauthorized, "password is required")
		return false
	}
	r.Body = io.NopCloser(strings.NewReader(string(bodyBytes)))
	valid := s.store.VerifyTestPassword(req.Password)
	recordPasswordAttempt(rateLimitKey, valid)
	if !valid {
		writeError(w, http.StatusUnauthorized, "invalid password")
		return false
	}
	return true
}

func (s *Server) publicTestSettingsAuth(w http.ResponseWriter, r *http.Request) {
	rateLimitKey := testSettingsRateLimitKey(r)
	if !checkPasswordRateLimit(rateLimitKey) {
		writeError(w, http.StatusTooManyRequests, "too many failed password attempts; try again later")
		return
	}
	cfg := s.store.GetTestSettingsConfig()
	if !cfg.TestModeEnabled {
		writeError(w, http.StatusForbidden, "test mode is disabled")
		return
	}
	var req PublicTestSettingsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusUnauthorized, "password is required")
		return
	}
	valid := s.store.VerifyTestPassword(req.Password)
	recordPasswordAttempt(rateLimitKey, valid)
	if !valid {
		writeError(w, http.StatusUnauthorized, "invalid password")
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"authenticated": true})
}

func (s *Server) publicListTestEntries(w http.ResponseWriter, r *http.Request) {
	if !s.requireTestAuthBody(w, r) {
		return
	}
	writeJSON(w, http.StatusOK, s.store.ListTestSettingsEntries())
}

func (s *Server) publicAddTestEntry(w http.ResponseWriter, r *http.Request) {
	if !s.requireTestAuthBody(w, r) {
		return
	}
	var req AddTestEntryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	entry, err := s.store.AddTestSettingsEntry(req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, entry)
}

func (s *Server) publicUpdateTestEntry(w http.ResponseWriter, r *http.Request) {
	if !s.requireTestAuthBody(w, r) {
		return
	}
	var req UpdateTestEntryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	entry, err := s.store.UpdateTestSettingsEntry(r.PathValue("id"), req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, entry)
}

func (s *Server) publicDeleteTestEntry(w http.ResponseWriter, r *http.Request) {
	if !s.requireTestAuthBody(w, r) {
		return
	}
	if err := s.store.DeleteTestSettingsEntry(r.PathValue("id")); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) publicRevealTestEntry(w http.ResponseWriter, r *http.Request) {
	if !s.requireTestAuthBody(w, r) {
		return
	}
	entry, err := s.store.RevealTestSettingsEntry(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, entry)
}

func (s *Server) publicTestStatus(w http.ResponseWriter, r *http.Request) {
	cfg := s.store.GetTestSettingsConfig()
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"test_mode_enabled": cfg.TestModeEnabled,
	})
}
