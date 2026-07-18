package core

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
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
	s.broadcastLiveFeedEvent("ledger_manual_credit")

	creditURL := scanAccountURL(s.cfg, entry.ToAccount)
	scanURL := scanBaseURL(s.cfg) + "/address/" + url.PathEscape(entry.ToAccount)
	commentBody := renderManualCreditComment(entry, workerID, rewardMRG, bountyType, creditURL, req.PRURL)

	// Optionally post comment to GitHub PR if a PR URL was provided
	var commentURL string
	var commentError string
	if prURL := strings.TrimSpace(req.PRURL); prURL != "" {
		if target, err := parseGitHubPullURL(prURL); err == nil {
			client, ghErr := newAdminGitHubClient(s.cfg, false)
			if ghErr == nil {
				pullNumber := target.IssueNumber
				cURL, cErr := client.commentPullRequest(r.Context(), target, pullNumber, commentBody)
				if cErr == nil {
					commentURL = cURL
				} else {
					commentError = cErr.Error()
				}
			}
		}
	}

	writeJSON(w, http.StatusCreated, AdminManualCreditResponse{
		LedgerEntry:    entry,
		WorkerID:       workerID,
		RewardMRG:      rewardMRG,
		BountyType:     bountyType,
		CreditURL:      creditURL,
		LedgerSequence: int64(entry.Sequence),
		ProofHash:      entry.EntryHash,
		ScanURL:        scanURL,
		CommentURL:     commentURL,
		CommentError:   commentError,
		CommentBody:    commentBody,
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
	if strings.HasPrefix(lower, "worker:github:") {
		return githubWorkerAccount(strings.TrimPrefix(lower, "worker:"))
	}
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

// renderManualCreditComment generates a standard bounty credit comment body
// matching the maintainer comment format used on merged bounty PRs.
func renderManualCreditComment(entry LedgerEntry, workerID string, rewardMRG int64, bountyType string, creditURL string, prURL string) string {
	proofHash := strings.TrimSpace(entry.EntryHash)
	if proofHash == "" {
		proofHash = entry.EntryHash
	}
	return fmt.Sprintf(`MergeOS bounty credit approved.

- Bounty type: %s
- MRG credited: %d MRG
- Credited worker: %s
- MRG credit URL: %s
- Proof hash: %s
- Ledger sequence: %d
- PR: %s
`, adminBountyTitle(bountyType), rewardMRG, workerID, creditURL, proofHash, entry.Sequence, prURL)
}

// parseGitHubPullURL parses a GitHub pull request URL into owner, repo, and pull number.
func parseGitHubPullURL(value string) (githubIssueTarget, error) {
	raw := strings.TrimSpace(value)
	if raw == "" {
		return githubIssueTarget{}, errors.New("pull request URL is required")
	}
	parsed, err := url.Parse(raw)
	if err != nil || !strings.EqualFold(parsed.Hostname(), "github.com") {
		return githubIssueTarget{}, errors.New("pull request URL must be a GitHub URL")
	}
	parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	if len(parts) < 4 || !strings.EqualFold(parts[2], "pull") {
		return githubIssueTarget{}, errors.New("pull request URL must look like https://github.com/owner/repo/pull/123")
	}
	number, err := strconv.Atoi(parts[3])
	if err != nil || number <= 0 {
		return githubIssueTarget{}, errors.New("pull request number is invalid")
	}
	owner, repo, err := cleanRepoParts(parts[:2])
	if err != nil {
		return githubIssueTarget{}, err
	}
	return githubIssueTarget{Owner: owner, Repo: repo, IssueNumber: number}, nil
}
