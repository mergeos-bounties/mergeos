package core

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
)

func (s *Server) createAdminLedgerCredit(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	var req AdminManualCreditRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	rewardMRG := selectedManualCreditRewardMRG(req)
	if rewardMRG <= 0 {
		writeError(w, http.StatusBadRequest, "reward_mrg is required")
		return
	}
	bountyType, err := normalizeAdminBountyType(req.BountyType)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	workerID := normalizeAdminCreditWorkerID(req.WorkerID)
	if workerID == "" {
		writeError(w, http.StatusBadRequest, "worker_id is required")
		return
	}
	reference, err := adminManualCreditReference(req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	entry, err := s.store.AddManualCredit(workerID, rewardMRG, reference)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, AdminManualCreditResponse{
		LedgerEntry: entry,
		WorkerID:    workerID,
		RewardMRG:   rewardMRG,
		BountyType:  bountyType,
		CreditURL:   scanAccountURL(s.cfg, entry.ToAccount),
	})
}

func selectedManualCreditRewardMRG(req AdminManualCreditRequest) int64 {
	if req.RewardMRG > 0 {
		return req.RewardMRG
	}
	if req.AmountMRG > 0 {
		return req.AmountMRG
	}
	return req.RewardCents
}

func normalizeAdminCreditWorkerID(value string) string {
	value = strings.TrimSpace(normalizeWorkerID(value))
	if value == "" {
		return ""
	}
	lower := strings.ToLower(value)
	if strings.HasPrefix(lower, "github:") {
		return githubWorkerAccount(value)
	}
	if validWalletAddress(value) || strings.HasPrefix(lower, "wallet:") || strings.HasPrefix(lower, "worker:") {
		return value
	}
	username := normalizeGitHubUsername(value)
	if isGitHubUsername(username) {
		return githubWorkerAccount(username)
	}
	return value
}

func adminManualCreditReference(req AdminManualCreditRequest) (string, error) {
	taskID := sanitizeLedgerReferenceValue(req.TaskID)
	if strings.TrimSpace(req.PRURL) != "" {
		if normalizeLedgerPullURL(req.PRURL) == "" {
			return "", errors.New("pr_url must be a GitHub pull request URL")
		}
		return buildPullLedgerReference(taskID, req.PRURL, req.PRTitle), nil
	}
	reference := sanitizeLedgerReferenceValue(req.Reference)
	if reference == "" {
		reference = sanitizeLedgerReferenceValue(req.Note)
	}
	if reference == "" {
		return "", errors.New("pr_url or reference is required")
	}
	if taskID != "" {
		return "task:" + taskID + ";manual:" + reference, nil
	}
	return "manual:" + reference, nil
}

func isGitHubUsername(value string) bool {
	if value == "" || len(value) > 39 || strings.HasPrefix(value, "-") || strings.HasSuffix(value, "-") {
		return false
	}
	for _, char := range value {
		if (char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') || char == '-' {
			continue
		}
		return false
	}
	return true
}
