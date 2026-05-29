package core

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// ---------- Models ----------

type TestSettingsConfig struct {
	TestModeEnabled  bool      `json:"test_mode_enabled"`
	TestPasswordHash string    `json:"-"`
	UpdatedAt        time.Time `json:"updated_at"`
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

// ---------- API request/response types ----------

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

// ---------- Known env names for collision detection ----------

var knownEnvNames = []string{
	"GITHUB_TOKEN",
	"MERGEOS_GITHUB_TOKEN",
	"TOKEN_SYMBOL",
	"ADMIN_EMAIL",
	"ADMIN_PASSWORD",
	"PAYPAL_CLIENT_ID",
	"PAYPAL_CLIENT_SECRET",
	"PAYPAL_ENVIRONMENT",
	"CRYPTO_WEBHOOK_SECRET",
	"CRYPTO_TOKEN_CONTRACT",
	"USDT_RECEIVER_ADDRESS",
	"GEMINI_API_KEYS",
}

// SettingValueMask returns a masked version of the value showing first 4 + last 4 characters.
func SettingValueMask(value string) string {
	if len(value) <= 8 {
		if len(value) == 0 {
			return "****"
		}
		return value[:1] + "****" + value[len(value)-1:]
	}
	return value[:4] + "****" + value[len(value)-4:]
}

// checkForEnvCollision checks if a setting_key collides with known env variable names.
func checkForEnvCollision(key string) error {
	upper := strings.ToUpper(strings.TrimSpace(key))
	if upper == "" {
		return nil
	}
	for _, known := range knownEnvNames {
		if upper == known {
			return fmt.Errorf("setting key %q collides with known environment variable %s", key, known)
		}
	}
	if strings.HasPrefix(upper, "MERGEOS_") {
		return fmt.Errorf("setting key %q collides with MERGEOS_* environment variable prefix", key)
	}
	return nil
}

// ---------- Store methods ----------

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

// VerifyTestPassword performs constant-time comparison of the given password
// against the stored hash salt:hash format.
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
			ID:               entry.ID,
			IntegrationType:  entry.IntegrationType,
			DisplayName:      entry.DisplayName,
			SettingKey:       entry.SettingKey,
			SettingValueHint: SettingValueMask(entry.SettingValue),
			KeyValueMap:      entry.KeyValueMap,
			Status:           entry.Status,
			LastUsedAt:       entry.LastUsedAt,
			CreatedAt:        entry.CreatedAt,
			UpdatedAt:        entry.UpdatedAt,
		})
	}
	return responses
}

func (s *Store) AddTestSettingsEntry(req AddTestEntryRequest) (*TestSettingsEntryResponse, error) {
	if strings.TrimSpace(req.IntegrationType) == "" {
		return nil, errors.New("integration_type is required")
	}
	if strings.TrimSpace(req.SettingKey) == "" {
		return nil, errors.New("setting_key is required")
	}
	if strings.TrimSpace(req.SettingValue) == "" {
		return nil, errors.New("setting_value is required")
	}

	// Check for env name collision.
	if err := checkForEnvCollision(req.SettingKey); err != nil {
		return nil, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().UTC()
	entry := &TestSettingsEntry{
		ID:              s.newID("tse"),
		IntegrationType: strings.TrimSpace(req.IntegrationType),
		DisplayName:     strings.TrimSpace(req.DisplayName),
		SettingKey:      strings.TrimSpace(req.SettingKey),
		SettingValue:    req.SettingValue,
		KeyValueMap:     req.KeyValueMap,
		Status:          "active",
		CreatedAt:       now,
		UpdatedAt:       now,
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
		if err := checkForEnvCollision(req.SettingKey); err != nil {
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
		for k, v := range req.KeyValueMap {
			entry.KeyValueMap[k] = v
		}
	}
	if strings.TrimSpace(req.Status) != "" {
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

func entryToResponse(entry *TestSettingsEntry) *TestSettingsEntryResponse {
	return &TestSettingsEntryResponse{
		ID:               entry.ID,
		IntegrationType:  entry.IntegrationType,
		DisplayName:      entry.DisplayName,
		SettingKey:       entry.SettingKey,
		SettingValueHint: SettingValueMask(entry.SettingValue),
		KeyValueMap:      entry.KeyValueMap,
		Status:           entry.Status,
		LastUsedAt:       entry.LastUsedAt,
		CreatedAt:        entry.CreatedAt,
		UpdatedAt:        entry.UpdatedAt,
	}
}

// ---------- Server handlers ----------

// Admin endpoints

func (s *Server) adminGetTestSettings(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	writeJSON(w, http.StatusOK, s.store.GetTestSettingsConfig())
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
	writeJSON(w, http.StatusOK, config)
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

// Public endpoints (password-gated)

func (s *Server) requireTestAuthBody(w http.ResponseWriter, r *http.Request) bool {
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
	// Re-inject body bytes so downstream handlers can decode the full payload.
	r.Body = io.NopCloser(strings.NewReader(string(bodyBytes)))
	if !s.store.VerifyTestPassword(req.Password) {
		writeError(w, http.StatusUnauthorized, "invalid password")
		return false
	}
	return true
}

func (s *Server) publicTestSettingsAuth(w http.ResponseWriter, r *http.Request) {
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
	if !s.store.VerifyTestPassword(req.Password) {
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

func (s *Server) publicTestStatus(w http.ResponseWriter, r *http.Request) {
	cfg := s.store.GetTestSettingsConfig()
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"test_mode_enabled": cfg.TestModeEnabled,
	})
}
