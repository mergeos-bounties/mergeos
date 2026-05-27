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
	GeminiAPIKeyStatusActive       = "active"
	GeminiAPIKeyStatusQuotaLimited = "quota_limited"
	GeminiAPIKeyStatusError        = "error"
	GeminiAPIKeyStatusDisabled     = "disabled"
)

const geminiAPIKeyRetryAfter = 24 * time.Hour
const maxGeminiWebhookLogs = 200

type GeminiAPIKeyCandidate struct {
	ID           string
	Provider     string
	Model        string
	KeyValue     string
	KeyHint      string
	Status       string
	RequestCount int64
	LastUsedAt   *time.Time
}

type GeminiAPIKeyStats struct {
	ID              string     `json:"id"`
	Provider        string     `json:"provider"`
	Model           string     `json:"model"`
	KeyHint         string     `json:"key_hint"`
	Status          string     `json:"status"`
	RequestCount    int64      `json:"request_count"`
	SuccessCount    int64      `json:"success_count"`
	QuotaErrorCount int64      `json:"quota_error_count"`
	LastStatusCode  int        `json:"last_status_code"`
	LastError       string     `json:"last_error,omitempty"`
	LastUsedAt      *time.Time `json:"last_used_at,omitempty"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

type AddGeminiAPIKeyRequest struct {
	KeyValue string `json:"key_value"`
	APIKey   string `json:"api_key"`
	Provider string `json:"provider"`
	Model    string `json:"model"`
}

type UpdateGeminiAPIKeyRequest struct {
	Status      string `json:"status"`
	ResetCounts bool   `json:"reset_counts"`
}

type TestGeminiAPIKeyRequest struct {
	Provider string `json:"provider"`
	Model    string `json:"model"`
}

type TestGeminiAPIKeyResponse struct {
	OK             bool              `json:"ok"`
	Provider       string            `json:"provider"`
	Model          string            `json:"model"`
	Key            GeminiAPIKeyStats `json:"key"`
	StatusCode     int               `json:"status_code"`
	DurationMillis int64             `json:"duration_millis"`
	Message        string            `json:"message,omitempty"`
	Error          string            `json:"error,omitempty"`
}

func (s *Store) SeedGeminiAPIKeysFromConfig() error {
	if len(s.cfg.GeminiAPIKeys) == 0 {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.geminiAPIKeys == nil {
		s.geminiAPIKeys = map[string]*GeminiAPIKey{}
	}

	now := time.Now().UTC()
	changed := false
	for _, raw := range s.cfg.GeminiAPIKeys {
		keyValue := strings.TrimSpace(raw)
		if keyValue == "" {
			continue
		}
		id := geminiAPIKeyID(keyValue)
		key, ok := s.geminiAPIKeys[id]
		if !ok {
			s.geminiAPIKeys[id] = &GeminiAPIKey{
				ID:        id,
				Provider:  "gemini",
				Model:     normalizedLLMModelOrDefault("gemini", ""),
				KeyValue:  keyValue,
				KeyHint:   geminiAPIKeyHint(keyValue),
				Status:    GeminiAPIKeyStatusActive,
				CreatedAt: now,
				UpdatedAt: now,
			}
			changed = true
			continue
		}
		if key.KeyValue != keyValue || key.KeyHint == "" || key.Status == "" || key.Provider == "" || key.Model == "" {
			key.KeyValue = keyValue
			key.Provider = "gemini"
			key.Model = normalizedLLMModelOrDefault("gemini", key.Model)
			key.KeyHint = geminiAPIKeyHint(keyValue)
			if key.Status == "" {
				key.Status = GeminiAPIKeyStatusActive
			}
			key.UpdatedAt = now
			changed = true
		}
	}
	if !changed {
		return nil
	}
	return s.saveLocked()
}

func (s *Store) HasRunnableGeminiAPIKey() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	now := time.Now().UTC()
	provider := normalizedLLMProviderOrDefault(s.adminSettings.LLMProvider)
	for _, key := range s.geminiAPIKeys {
		if normalizedLLMProviderOrDefault(key.Provider) == provider && geminiAPIKeyRunnable(key, now) {
			return true
		}
	}
	return false
}

func (s *Store) GeminiAPIKeyCandidates() []GeminiAPIKeyCandidate {
	return s.GeminiAPIKeyCandidatesForProvider("")
}

func (s *Store) GeminiAPIKeyCandidatesForProvider(provider string) []GeminiAPIKeyCandidate {
	s.mu.RLock()
	defer s.mu.RUnlock()
	provider = strings.TrimSpace(provider)
	if provider != "" {
		provider = normalizedLLMProviderOrDefault(provider)
	}
	now := time.Now().UTC()
	candidates := []GeminiAPIKeyCandidate{}
	for _, key := range s.geminiAPIKeys {
		if !geminiAPIKeyRunnable(key, now) {
			continue
		}
		keyProvider := normalizedLLMProviderOrDefault(key.Provider)
		if provider != "" && keyProvider != provider {
			continue
		}
		candidates = append(candidates, GeminiAPIKeyCandidate{
			ID:           key.ID,
			Provider:     keyProvider,
			Model:        normalizedLLMModelOrDefault(keyProvider, key.Model),
			KeyValue:     key.KeyValue,
			KeyHint:      key.KeyHint,
			Status:       key.Status,
			RequestCount: key.RequestCount,
			LastUsedAt:   cloneTimePtr(key.LastUsedAt),
		})
	}
	sort.Slice(candidates, func(i, j int) bool {
		left := candidates[i]
		right := candidates[j]
		if left.Status != right.Status {
			return left.Status == GeminiAPIKeyStatusActive
		}
		if left.RequestCount != right.RequestCount {
			return left.RequestCount < right.RequestCount
		}
		if left.LastUsedAt == nil && right.LastUsedAt != nil {
			return true
		}
		if left.LastUsedAt != nil && right.LastUsedAt == nil {
			return false
		}
		if left.LastUsedAt != nil && right.LastUsedAt != nil && !left.LastUsedAt.Equal(*right.LastUsedAt) {
			return left.LastUsedAt.Before(*right.LastUsedAt)
		}
		return left.ID < right.ID
	})
	return candidates
}

func (s *Store) ListGeminiAPIKeyStats() []GeminiAPIKeyStats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := make([]GeminiAPIKeyStats, 0, len(s.geminiAPIKeys))
	for _, key := range s.geminiAPIKeys {
		stats = append(stats, GeminiAPIKeyStats{
			ID:              key.ID,
			Provider:        normalizedLLMProviderOrDefault(key.Provider),
			Model:           normalizedLLMModelOrDefault(key.Provider, key.Model),
			KeyHint:         key.KeyHint,
			Status:          key.Status,
			RequestCount:    key.RequestCount,
			SuccessCount:    key.SuccessCount,
			QuotaErrorCount: key.QuotaErrorCount,
			LastStatusCode:  key.LastStatusCode,
			LastError:       key.LastError,
			LastUsedAt:      cloneTimePtr(key.LastUsedAt),
			UpdatedAt:       key.UpdatedAt,
		})
	}
	sort.Slice(stats, func(i, j int) bool {
		if stats[i].RequestCount != stats[j].RequestCount {
			return stats[i].RequestCount < stats[j].RequestCount
		}
		return stats[i].ID < stats[j].ID
	})
	return stats
}

func (s *Store) AddGeminiAPIKey(value string, options ...string) (GeminiAPIKeyStats, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return GeminiAPIKeyStats{}, errors.New("LLM API key is required")
	}
	if len(value) < 8 {
		return GeminiAPIKeyStats{}, errors.New("LLM API key is too short")
	}
	provider := defaultLLMProvider
	model := ""
	if len(options) > 0 {
		provider = options[0]
	}
	if len(options) > 1 {
		model = options[1]
	}
	provider, err := normalizeLLMProvider(provider)
	if err != nil {
		return GeminiAPIKeyStats{}, err
	}
	model = normalizedLLMModelOrDefault(provider, model)
	if _, err := normalizeLLMModel(provider, model); err != nil {
		return GeminiAPIKeyStats{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.geminiAPIKeys == nil {
		s.geminiAPIKeys = map[string]*GeminiAPIKey{}
	}

	id := geminiAPIKeyID(value)
	if _, ok := s.geminiAPIKeys[id]; ok {
		return GeminiAPIKeyStats{}, errors.New("LLM API token already exists")
	}
	now := time.Now().UTC()
	key := &GeminiAPIKey{
		ID:        id,
		Provider:  provider,
		Model:     model,
		KeyValue:  value,
		KeyHint:   geminiAPIKeyHint(value),
		Status:    GeminiAPIKeyStatusActive,
		CreatedAt: now,
		UpdatedAt: now,
	}
	s.geminiAPIKeys[id] = key
	if err := s.saveLocked(); err != nil {
		return GeminiAPIKeyStats{}, err
	}
	return geminiAPIKeyStatsFromKey(key), nil
}

func (s *Store) UpdateGeminiAPIKey(id, status string, resetCounts bool) (GeminiAPIKeyStats, error) {
	id = strings.TrimSpace(id)
	rawStatus := strings.TrimSpace(status)
	status = normalizeGeminiAPIKeyStatus(status)
	if id == "" {
		return GeminiAPIKeyStats{}, errors.New("LLM API token id is required")
	}
	if rawStatus != "" && status == "" {
		return GeminiAPIKeyStats{}, errors.New("unsupported LLM API token status")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.geminiAPIKeys == nil {
		s.geminiAPIKeys = map[string]*GeminiAPIKey{}
	}

	key := s.geminiAPIKeys[id]
	if key == nil {
		return GeminiAPIKeyStats{}, errors.New("LLM API token not found")
	}
	now := time.Now().UTC()
	if status != "" {
		key.Status = status
	}
	if resetCounts {
		key.RequestCount = 0
		key.SuccessCount = 0
		key.QuotaErrorCount = 0
		key.LastStatusCode = 0
		key.LastError = ""
		key.LastUsedAt = nil
		if key.Status == GeminiAPIKeyStatusQuotaLimited || key.Status == GeminiAPIKeyStatusError {
			key.Status = GeminiAPIKeyStatusActive
		}
	}
	key.UpdatedAt = now
	if err := s.saveLocked(); err != nil {
		return GeminiAPIKeyStats{}, err
	}
	return geminiAPIKeyStatsFromKey(key), nil
}

func (s *Store) GeminiAPIKeyCandidateByID(id string) (GeminiAPIKeyCandidate, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return GeminiAPIKeyCandidate{}, errors.New("LLM API token id is required")
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	key := s.geminiAPIKeys[id]
	if key == nil {
		return GeminiAPIKeyCandidate{}, errors.New("LLM API token not found")
	}
	return GeminiAPIKeyCandidate{
		ID:           key.ID,
		Provider:     normalizedLLMProviderOrDefault(key.Provider),
		Model:        normalizedLLMModelOrDefault(key.Provider, key.Model),
		KeyValue:     key.KeyValue,
		KeyHint:      key.KeyHint,
		Status:       key.Status,
		RequestCount: key.RequestCount,
		LastUsedAt:   cloneTimePtr(key.LastUsedAt),
	}, nil
}

func (s *Store) RecordGeminiAPIKeyTestResult(id, status string, statusCode int, message string) (GeminiAPIKeyStats, error) {
	id = strings.TrimSpace(id)
	status = normalizeGeminiAPIKeyStatus(status)
	if id == "" {
		return GeminiAPIKeyStats{}, errors.New("LLM API token id is required")
	}
	if status == "" {
		return GeminiAPIKeyStats{}, errors.New("LLM API token test status is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	key := s.geminiAPIKeys[id]
	if key == nil {
		return GeminiAPIKeyStats{}, errors.New("LLM API token not found")
	}
	now := time.Now().UTC()
	wasDisabled := key.Status == GeminiAPIKeyStatusDisabled
	key.RequestCount++
	key.LastStatusCode = statusCode
	key.LastUsedAt = &now
	key.UpdatedAt = now
	switch status {
	case GeminiAPIKeyStatusActive:
		key.SuccessCount++
		key.LastError = ""
		if !wasDisabled {
			key.Status = GeminiAPIKeyStatusActive
		}
	case GeminiAPIKeyStatusQuotaLimited:
		key.QuotaErrorCount++
		key.LastError = truncateGeminiKeyError(message)
		if !wasDisabled {
			key.Status = GeminiAPIKeyStatusQuotaLimited
		}
	case GeminiAPIKeyStatusError:
		key.LastError = truncateGeminiKeyError(message)
		if !wasDisabled {
			key.Status = GeminiAPIKeyStatusError
		}
	default:
		return GeminiAPIKeyStats{}, errors.New("unsupported LLM API token test status")
	}
	if err := s.saveLocked(); err != nil {
		return GeminiAPIKeyStats{}, err
	}
	return geminiAPIKeyStatsFromKey(key), nil
}

func (s *Store) MarkGeminiAPIKeyAttempt(id string) error {
	return s.updateGeminiAPIKey(id, func(key *GeminiAPIKey, now time.Time) {
		key.RequestCount++
		key.Status = GeminiAPIKeyStatusActive
		key.LastUsedAt = &now
		key.LastError = ""
		key.UpdatedAt = now
	})
}

func (s *Store) MarkGeminiAPIKeySuccess(id string, statusCode int) error {
	return s.updateGeminiAPIKey(id, func(key *GeminiAPIKey, now time.Time) {
		key.SuccessCount++
		key.Status = GeminiAPIKeyStatusActive
		key.LastStatusCode = statusCode
		key.LastError = ""
		key.UpdatedAt = now
	})
}

func (s *Store) MarkGeminiAPIKeyQuotaLimited(id string, statusCode int, message string) error {
	return s.updateGeminiAPIKey(id, func(key *GeminiAPIKey, now time.Time) {
		key.QuotaErrorCount++
		key.Status = GeminiAPIKeyStatusQuotaLimited
		key.LastStatusCode = statusCode
		key.LastError = truncateGeminiKeyError(message)
		key.UpdatedAt = now
	})
}

func (s *Store) MarkGeminiAPIKeyError(id string, statusCode int, message string) error {
	return s.updateGeminiAPIKey(id, func(key *GeminiAPIKey, now time.Time) {
		key.Status = GeminiAPIKeyStatusError
		key.LastStatusCode = statusCode
		key.LastError = truncateGeminiKeyError(message)
		key.UpdatedAt = now
	})
}

func (s *Store) updateGeminiAPIKey(id string, update func(*GeminiAPIKey, time.Time)) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := s.geminiAPIKeys[id]
	if key == nil {
		return nil
	}
	if key.Status == GeminiAPIKeyStatusDisabled {
		return nil
	}
	update(key, time.Now().UTC())
	return s.saveLocked()
}

func (s *Store) AddGeminiWebhookLog(log GeminiWebhookLog) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.geminiWebhookLogs == nil {
		s.geminiWebhookLogs = map[string]*GeminiWebhookLog{}
	}

	if strings.TrimSpace(log.ID) == "" {
		log.ID = geminiWebhookLogID()
	}
	if log.ReceivedAt.IsZero() {
		log.ReceivedAt = time.Now().UTC()
	}
	if log.CompletedAt != nil && log.DurationMillis <= 0 {
		log.DurationMillis = log.CompletedAt.Sub(log.ReceivedAt).Milliseconds()
	}
	if log.Status == "" {
		log.Status = "received"
	}
	log.Error = truncateGeminiWebhookError(log.Error)
	log.Labels = append([]string(nil), log.Labels...)
	s.geminiWebhookLogs[log.ID] = &log
	s.trimGeminiWebhookLogsLocked()
	return s.saveLocked()
}

func (s *Store) ListGeminiWebhookLogs(limit int) []GeminiWebhookLog {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if limit <= 0 || limit > maxGeminiWebhookLogs {
		limit = maxGeminiWebhookLogs
	}
	logs := make([]GeminiWebhookLog, 0, len(s.geminiWebhookLogs))
	for _, log := range s.geminiWebhookLogs {
		logCopy := *log
		logCopy.Labels = append([]string(nil), log.Labels...)
		logs = append(logs, logCopy)
	}
	sort.Slice(logs, func(i, j int) bool {
		return logs[i].ReceivedAt.After(logs[j].ReceivedAt)
	})
	if len(logs) > limit {
		logs = logs[:limit]
	}
	return logs
}

func (s *Store) trimGeminiWebhookLogsLocked() {
	if len(s.geminiWebhookLogs) <= maxGeminiWebhookLogs {
		return
	}
	logs := make([]*GeminiWebhookLog, 0, len(s.geminiWebhookLogs))
	for _, log := range s.geminiWebhookLogs {
		logs = append(logs, log)
	}
	sort.Slice(logs, func(i, j int) bool {
		return logs[i].ReceivedAt.After(logs[j].ReceivedAt)
	})
	for _, log := range logs[maxGeminiWebhookLogs:] {
		delete(s.geminiWebhookLogs, log.ID)
	}
}

func geminiAPIKeyRunnable(key *GeminiAPIKey, now time.Time) bool {
	if key == nil || strings.TrimSpace(key.KeyValue) == "" || key.Status == GeminiAPIKeyStatusDisabled {
		return false
	}
	if key.Status == "" || key.Status == GeminiAPIKeyStatusActive {
		return true
	}
	if key.LastUsedAt == nil {
		return false
	}
	return now.Sub(*key.LastUsedAt) >= geminiAPIKeyRetryAfter
}

func normalizeGeminiAPIKeyStatus(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "":
		return ""
	case GeminiAPIKeyStatusActive:
		return GeminiAPIKeyStatusActive
	case GeminiAPIKeyStatusQuotaLimited:
		return GeminiAPIKeyStatusQuotaLimited
	case GeminiAPIKeyStatusError:
		return GeminiAPIKeyStatusError
	case GeminiAPIKeyStatusDisabled:
		return GeminiAPIKeyStatusDisabled
	default:
		return ""
	}
}

func geminiAPIKeyStatsFromKey(key *GeminiAPIKey) GeminiAPIKeyStats {
	if key == nil {
		return GeminiAPIKeyStats{}
	}
	return GeminiAPIKeyStats{
		ID:              key.ID,
		Provider:        normalizedLLMProviderOrDefault(key.Provider),
		Model:           normalizedLLMModelOrDefault(key.Provider, key.Model),
		KeyHint:         key.KeyHint,
		Status:          key.Status,
		RequestCount:    key.RequestCount,
		SuccessCount:    key.SuccessCount,
		QuotaErrorCount: key.QuotaErrorCount,
		LastStatusCode:  key.LastStatusCode,
		LastError:       key.LastError,
		LastUsedAt:      cloneTimePtr(key.LastUsedAt),
		UpdatedAt:       key.UpdatedAt,
	}
}

func geminiAPIKeyID(value string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(value)))
	return hex.EncodeToString(sum[:])[:24]
}

func geminiAPIKeyHint(value string) string {
	value = strings.TrimSpace(value)
	if len(value) <= 8 {
		return "****"
	}
	return value[:4] + "..." + value[len(value)-4:]
}

func truncateGeminiKeyError(value string) string {
	value = strings.TrimSpace(value)
	if len(value) <= 500 {
		return value
	}
	return value[:500]
}

func truncateGeminiWebhookError(value string) string {
	value = strings.TrimSpace(value)
	if len(value) <= 1000 {
		return value
	}
	return value[:1000]
}

func geminiWebhookLogID() string {
	token, err := newToken()
	if err != nil || len(token) < 16 {
		return "gwh_" + time.Now().UTC().Format("20060102150405")
	}
	return "gwh_" + token[:16]
}

func cloneTimePtr(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}
