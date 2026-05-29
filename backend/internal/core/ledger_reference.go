package core

import (
	"net/url"
	"strconv"
	"strings"
)

const maxLedgerReferenceValueLength = 160

func buildPullLedgerReference(taskID, pullURL, pullTitle string) string {
	parts := []string{}
	if taskID = sanitizeLedgerReferenceValue(taskID); taskID != "" {
		parts = append(parts, "task:"+taskID)
	}
	if pullURL = normalizeLedgerPullURL(pullURL); pullURL != "" {
		parts = append(parts, "pr:"+pullURL)
	}
	if pullTitle = sanitizeLedgerReferenceValue(pullTitle); pullTitle != "" {
		parts = append(parts, "title:"+pullTitle)
	}
	return strings.Join(parts, ";")
}

func ensureTaskLedgerReference(taskID, reference string) string {
	taskID = strings.TrimSpace(taskID)
	taskReference := "task:" + taskID
	reference = strings.TrimSpace(reference)
	if reference == "" {
		return taskReference
	}
	if ledgerReferenceTaskID(reference) == taskID {
		return reference
	}
	return taskReference + ";" + reference
}

func ledgerReferenceTaskID(reference string) string {
	fields := splitLedgerReference(reference)
	if taskID := strings.TrimSpace(fields["task"]); taskID != "" {
		return taskID
	}
	reference = strings.TrimSpace(reference)
	if strings.HasPrefix(reference, "task:") && !strings.Contains(reference, ";") {
		return strings.TrimSpace(strings.TrimPrefix(reference, "task:"))
	}
	return ""
}

func publicPullLedgerReference(reference string) string {
	fields := splitLedgerReference(reference)
	pullURL := normalizeLedgerPullURL(fields["pr"])
	if pullURL == "" {
		return ""
	}
	if title := sanitizeLedgerReferenceValue(fields["title"]); title != "" {
		return "pr:" + pullURL + ";title:" + title
	}
	return "pr:" + pullURL
}

func splitLedgerReference(reference string) map[string]string {
	fields := map[string]string{}
	for _, part := range strings.Split(reference, ";") {
		key, value, ok := strings.Cut(strings.TrimSpace(part), ":")
		if !ok {
			continue
		}
		key = strings.ToLower(strings.TrimSpace(key))
		value = strings.TrimSpace(value)
		if key == "" || value == "" || fields[key] != "" {
			continue
		}
		fields[key] = value
	}
	return fields
}

func sanitizeLedgerReferenceValue(value string) string {
	value = strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
	value = strings.ReplaceAll(value, ";", ",")
	runes := []rune(value)
	if len(runes) > maxLedgerReferenceValueLength {
		value = string(runes[:maxLedgerReferenceValueLength])
	}
	return value
}

func normalizeLedgerPullURL(value string) string {
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil {
		return ""
	}
	if parsed.Scheme != "https" && parsed.Scheme != "http" {
		return ""
	}
	if !strings.EqualFold(parsed.Hostname(), "github.com") {
		return ""
	}
	parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	if len(parts) < 4 || !strings.EqualFold(parts[2], "pull") {
		return ""
	}
	number, err := strconv.Atoi(parts[3])
	if err != nil || number <= 0 {
		return ""
	}
	owner := sanitizeLedgerReferenceValue(parts[0])
	repo := sanitizeLedgerReferenceValue(parts[1])
	if owner == "" || repo == "" || strings.Contains(owner, "/") || strings.Contains(repo, "/") {
		return ""
	}
	return "https://github.com/" + owner + "/" + repo + "/pull/" + strconv.Itoa(number)
}
