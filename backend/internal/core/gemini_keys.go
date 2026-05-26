package core

import (
	"crypto/sha256"
	"encoding/hex"
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

type GeminiAPIKeyCandidate struct {
	ID           string
	KeyValue     string
	KeyHint      string
	Status       string
	RequestCount int64
	LastUsedAt   *time.Time
}

type GeminiAPIKeyStats struct {
	ID              string     `json:"id"`
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

func (s *Store) SeedGeminiAPIKeysFromConfig() error {
	if len(s.cfg.GeminiAPIKeys) == 0 {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()

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
				KeyValue:  keyValue,
				KeyHint:   geminiAPIKeyHint(keyValue),
				Status:    GeminiAPIKeyStatusActive,
				CreatedAt: now,
				UpdatedAt: now,
			}
			changed = true
			continue
		}
		if key.KeyValue != keyValue || key.KeyHint == "" || key.Status == "" {
			key.KeyValue = keyValue
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
	for _, key := range s.geminiAPIKeys {
		if geminiAPIKeyRunnable(key, now) {
			return true
		}
	}
	return false
}

func (s *Store) GeminiAPIKeyCandidates() []GeminiAPIKeyCandidate {
	s.mu.RLock()
	defer s.mu.RUnlock()
	now := time.Now().UTC()
	candidates := []GeminiAPIKeyCandidate{}
	for _, key := range s.geminiAPIKeys {
		if !geminiAPIKeyRunnable(key, now) {
			continue
		}
		candidates = append(candidates, GeminiAPIKeyCandidate{
			ID:           key.ID,
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

func cloneTimePtr(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}
