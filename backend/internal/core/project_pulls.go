package core

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type projectPullRequestLister interface {
	listPullRequestsLinkedToIssue(ctx context.Context, target githubIssueTarget) ([]AdminTaskPullRequest, error)
}

type projectRecentPullRequestLister interface {
	listRecentPullRequests(ctx context.Context, target githubIssueTarget, limit int) ([]AdminTaskPullRequest, error)
}

const publicProjectPullRequestLimit = 100

var publicPullIssueMentionPattern = regexp.MustCompile(`(?i)(?:^|[^A-Za-z0-9_])(?:#|issues/)(\d+)\b`)

func projectPullRequestsMonitor(ctx context.Context, lister projectPullRequestLister, project *Project) ProjectPullRequestsResponse {
	response := ProjectPullRequestsResponse{
		ProtocolVersion: "mergeos.pr-monitor.v1",
		Kind:            "pr_monitor",
		ProjectID:       project.ID,
		ProjectTitle:    publicLiveFeedProjectTitle(project),
		Tasks:           []ProjectTaskPullRequests{},
		UpdatedAt:       project.CreatedAt,
	}
	tasks := make([]*Task, 0, len(project.Tasks))
	for _, task := range project.Tasks {
		if task == nil {
			continue
		}
		taskCopy := *task
		tasks = append(tasks, &taskCopy)
	}
	sortTasks(tasks)
	response.Stats.TaskCount = len(tasks)
	autoReleaseRows := []projectAutoReleaseMonitorRow{}

	for _, task := range tasks {
		row := ProjectTaskPullRequests{
			TaskID:        task.ID,
			IssueNumber:   task.IssueNumber,
			Title:         task.Title,
			Status:        string(task.Status),
			RewardCents:   task.RewardCents,
			WorkerKind:    task.RequiredWorkerKind,
			WorkerID:      task.WorkerID,
			AgentType:     projectPullRequestTaskAgentType(task),
			IssueURL:      marketplacePublicRepoURL(task.IssueURL),
			MonitorStatus: "unlinked",
			PullRequests:  []ProjectPullRequestSummary{},
			UpdatedAt:     deploymentTaskUpdatedAt(task),
		}
		if row.UpdatedAt.After(response.UpdatedAt) {
			response.UpdatedAt = row.UpdatedAt
		}

		target, err := githubIssueTargetForTask(task, project)
		if err != nil {
			row.MonitorError = sanitizeLedgerReferenceValue(err.Error())
			response.Tasks = append(response.Tasks, row)
			continue
		}
		row.Repository = target.fullName()
		row.MonitorStatus = "linked"
		response.Stats.LinkedTaskCount++

		pulls, err := lister.listPullRequestsLinkedToIssue(ctx, target)
		if err != nil {
			row.MonitorStatus = "error"
			row.MonitorError = sanitizeLedgerReferenceValue(err.Error())
			response.Stats.ErrorCount++
			response.Tasks = append(response.Tasks, row)
			continue
		}
		row.MonitorStatus = "synced"
		for _, pull := range pulls {
			pull.Readiness = adminPullRequestReadiness(task, pull)
			summary := projectPullRequestSummary(pull)
			row.PullRequests = append(row.PullRequests, summary)
			projectPullRequestsAddStats(&response.Stats, summary)
			if summary.UpdatedAt.After(row.UpdatedAt) {
				row.UpdatedAt = summary.UpdatedAt
			}
		}
		if candidate, ok := projectAutoReleaseCandidateForTask(task, row); ok {
			autoReleaseRows = append(autoReleaseRows, projectAutoReleaseMonitorRow{
				RowIndex:   len(response.Tasks),
				Task:       task,
				Candidate:  candidate,
				Repository: row.Repository,
			})
		}
		if row.UpdatedAt.After(response.UpdatedAt) {
			response.UpdatedAt = row.UpdatedAt
		}
		response.Tasks = append(response.Tasks, row)
	}
	attachProjectAutoReleasePackets(&response, project.ID, autoReleaseRows)
	return response
}

func publicProjectPullRequestsMonitor(ctx context.Context, lister projectPullRequestLister, project *Project) ProjectPullRequestsResponse {
	if recentLister, ok := lister.(projectRecentPullRequestLister); ok {
		return publicProjectPullRequestsRecentMonitor(ctx, recentLister, project)
	}
	response := projectPullRequestsMonitor(ctx, lister, project)
	for i := range response.Tasks {
		response.Tasks[i].TaskID = ""
		response.Tasks[i].WorkerID = ""
		response.Tasks[i].ReleasePacket = nil
		response.Tasks[i].AutoReleasePacket = nil
	}
	return response
}

func publicProjectPullRequestsRecentMonitor(ctx context.Context, lister projectRecentPullRequestLister, project *Project) ProjectPullRequestsResponse {
	response := ProjectPullRequestsResponse{
		ProtocolVersion: "mergeos.pr-monitor.v1",
		Kind:            "pr_monitor",
		ProjectID:       project.ID,
		ProjectTitle:    publicLiveFeedProjectTitle(project),
		Tasks:           []ProjectTaskPullRequests{},
		UpdatedAt:       project.CreatedAt,
	}
	tasks := make([]*Task, 0, len(project.Tasks))
	for _, task := range project.Tasks {
		if task == nil {
			continue
		}
		taskCopy := *task
		tasks = append(tasks, &taskCopy)
	}
	sortTasks(tasks)
	response.Stats.TaskCount = len(tasks)

	rowTasks := []*Task{}
	repoKeys := []string{}
	repoTargets := map[string]githubIssueTarget{}
	repoIssueRows := map[string]map[int][]int{}
	for _, task := range tasks {
		row := ProjectTaskPullRequests{
			IssueNumber:   task.IssueNumber,
			Title:         task.Title,
			Status:        string(task.Status),
			IssueURL:      marketplacePublicRepoURL(task.IssueURL),
			MonitorStatus: "unlinked",
			PullRequests:  []ProjectPullRequestSummary{},
			UpdatedAt:     deploymentTaskUpdatedAt(task),
		}
		if row.UpdatedAt.After(response.UpdatedAt) {
			response.UpdatedAt = row.UpdatedAt
		}

		target, err := githubIssueTargetForTask(task, project)
		if err != nil {
			row.MonitorError = sanitizeLedgerReferenceValue(err.Error())
			response.Tasks = append(response.Tasks, row)
			rowTasks = append(rowTasks, task)
			continue
		}
		row.Repository = target.fullName()
		row.MonitorStatus = "linked"
		response.Stats.LinkedTaskCount++
		rowIndex := len(response.Tasks)
		response.Tasks = append(response.Tasks, row)
		rowTasks = append(rowTasks, task)

		repoKey := strings.ToLower(target.fullName())
		if _, ok := repoTargets[repoKey]; !ok {
			repoTargets[repoKey] = target
			repoKeys = append(repoKeys, repoKey)
		}
		if repoIssueRows[repoKey] == nil {
			repoIssueRows[repoKey] = map[int][]int{}
		}
		repoIssueRows[repoKey][target.IssueNumber] = append(repoIssueRows[repoKey][target.IssueNumber], rowIndex)
	}
	sort.Strings(repoKeys)

	for _, repoKey := range repoKeys {
		target := repoTargets[repoKey]
		pulls, err := lister.listRecentPullRequests(ctx, target, publicProjectPullRequestLimit)
		if err != nil {
			message := sanitizeLedgerReferenceValue(err.Error())
			for _, rows := range repoIssueRows[repoKey] {
				for _, rowIndex := range rows {
					response.Tasks[rowIndex].MonitorStatus = "error"
					response.Tasks[rowIndex].MonitorError = message
					response.Stats.ErrorCount++
				}
			}
			continue
		}
		for _, pull := range pulls {
			for _, issueNumber := range publicPullReferencedIssues(pull) {
				for _, rowIndex := range repoIssueRows[repoKey][issueNumber] {
					pull.Readiness = adminPullRequestReadiness(rowTasks[rowIndex], pull)
					summary := projectPullRequestSummary(pull)
					response.Tasks[rowIndex].PullRequests = append(response.Tasks[rowIndex].PullRequests, summary)
					projectPullRequestsAddStats(&response.Stats, summary)
					if summary.UpdatedAt.After(response.Tasks[rowIndex].UpdatedAt) {
						response.Tasks[rowIndex].UpdatedAt = summary.UpdatedAt
					}
					if summary.UpdatedAt.After(response.UpdatedAt) {
						response.UpdatedAt = summary.UpdatedAt
					}
				}
			}
		}
		for _, rows := range repoIssueRows[repoKey] {
			for _, rowIndex := range rows {
				if response.Tasks[rowIndex].MonitorStatus != "error" {
					response.Tasks[rowIndex].MonitorStatus = "synced"
				}
				sort.SliceStable(response.Tasks[rowIndex].PullRequests, func(i, j int) bool {
					return response.Tasks[rowIndex].PullRequests[i].UpdatedAt.After(response.Tasks[rowIndex].PullRequests[j].UpdatedAt)
				})
			}
		}
	}

	return response
}

func publicPullReferencedIssues(pull AdminTaskPullRequest) []int {
	seen := map[int]bool{}
	issues := []int{}
	text := strings.Join([]string{pull.Title, pull.Body, pull.HTMLURL, pull.HeadRef, pull.BaseRef}, "\n")
	for _, match := range publicPullIssueMentionPattern.FindAllStringSubmatch(text, -1) {
		if len(match) < 2 {
			continue
		}
		number, err := strconv.Atoi(match[1])
		if err != nil || number <= 0 || seen[number] {
			continue
		}
		seen[number] = true
		issues = append(issues, number)
	}
	sort.Ints(issues)
	return issues
}

func projectPullRequestSummary(pull AdminTaskPullRequest) ProjectPullRequestSummary {
	return ProjectPullRequestSummary{
		Number:         pull.Number,
		Title:          pull.Title,
		State:          pull.State,
		HTMLURL:        pull.HTMLURL,
		MergeURL:       pull.MergeURL,
		Author:         pull.Author,
		Draft:          pull.Draft,
		Merged:         pull.Merged,
		MergeableState: pull.MergeableState,
		BaseRef:        pull.BaseRef,
		HeadRef:        pull.HeadRef,
		Labels:         append([]string{}, pull.Labels...),
		Readiness:      pull.Readiness,
		CreatedAt:      pull.CreatedAt,
		UpdatedAt:      pull.UpdatedAt,
		MergedAt:       pull.MergedAt,
	}
}

func projectPullRequestsAddStats(stats *ProjectPullRequestStats, pull ProjectPullRequestSummary) {
	stats.PullRequestCount++
	if pull.Merged {
		stats.MergedPullRequestCount++
	} else if strings.EqualFold(pull.State, "open") {
		stats.OpenPullRequestCount++
	}
	switch strings.ToLower(strings.TrimSpace(pull.Readiness.Status)) {
	case "ready":
		stats.ReadyCount++
	case "needs_review":
		stats.NeedsReviewCount++
	case "blocked":
		stats.BlockedCount++
	}
}

type projectAutoReleaseMonitorRow struct {
	RowIndex   int
	Task       *Task
	Candidate  ProjectAutoReleaseCandidate
	Repository string
}

func projectPullRequestTaskAgentType(task *Task) string {
	if task == nil {
		return ""
	}
	if strings.TrimSpace(task.AgentType) != "" {
		return strings.TrimSpace(task.AgentType)
	}
	return strings.TrimSpace(task.SuggestedAgentType)
}

func projectAutoReleaseCandidateForTask(task *Task, row ProjectTaskPullRequests) (ProjectAutoReleaseCandidate, bool) {
	if task == nil || taskIsReleased(task) {
		return ProjectAutoReleaseCandidate{}, false
	}
	var selected *ProjectPullRequestSummary
	for index := range row.PullRequests {
		pull := &row.PullRequests[index]
		readiness := pull.Readiness
		if !readiness.CanMerge || !strings.EqualFold(readiness.Status, "ready") || !strings.EqualFold(readiness.RiskLevel, "low") {
			continue
		}
		if pull.Draft || normalizeLedgerPullURL(pull.HTMLURL) == "" {
			continue
		}
		if selected == nil || pull.UpdatedAt.After(selected.UpdatedAt) {
			selected = pull
		}
	}
	if selected == nil {
		return ProjectAutoReleaseCandidate{}, false
	}
	workerID, err := githubWorkerID(selected.Author)
	if err != nil {
		return ProjectAutoReleaseCandidate{}, false
	}
	candidate := ProjectAutoReleaseCandidate{
		TaskID:            task.ID,
		WorkerKind:        task.RequiredWorkerKind,
		WorkerID:          workerID,
		AgentType:         "",
		RewardCents:       task.RewardCents,
		Repository:        row.Repository,
		PullRequestNumber: selected.Number,
		PullRequestURL:    normalizeLedgerPullURL(selected.HTMLURL),
		PullRequestTitle:  selected.Title,
		ReadinessStatus:   selected.Readiness.Status,
		CanMerge:          selected.Readiness.CanMerge,
		RiskLevel:         selected.Readiness.RiskLevel,
		Draft:             selected.Draft,
		CanRelease:        true,
	}
	if candidate.WorkerKind != WorkerHuman {
		candidate.AgentType = projectPullRequestTaskAgentType(task)
		if candidate.AgentType == "" {
			candidate.AgentType = "github-pr"
		}
	}
	return candidate, true
}

func attachProjectAutoReleasePackets(response *ProjectPullRequestsResponse, projectID string, rows []projectAutoReleaseMonitorRow) {
	if response == nil || len(rows) == 0 {
		return
	}
	candidates := make([]ProjectAutoReleaseCandidate, 0, len(rows))
	for _, row := range rows {
		candidates = append(candidates, row.Candidate)
	}
	for _, row := range rows {
		if row.RowIndex < 0 || row.RowIndex >= len(response.Tasks) {
			continue
		}
		response.Tasks[row.RowIndex].ReleasePacket = projectAutoReleasePacket(projectID, []ProjectAutoReleaseCandidate{row.Candidate}, false, row.Repository)
		response.Tasks[row.RowIndex].AutoReleasePacket = projectAutoReleasePacket(projectID, candidates, true, row.Repository)
		response.Stats.AutoReleaseReadyCount++
	}
}

func projectAutoReleasePacket(projectID string, candidates []ProjectAutoReleaseCandidate, auto bool, repository string) map[string]any {
	taskIDs := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		if strings.TrimSpace(candidate.TaskID) != "" {
			taskIDs = append(taskIDs, candidate.TaskID)
		}
	}
	packet := map[string]any{
		"status":           "ready",
		"method":           "POST",
		"release_endpoint": fmt.Sprintf("/api/projects/%s/auto-release", projectID),
		"policy":           defaultAutoReleasePolicy,
		"payload": map[string]any{
			"task_ids":   taskIDs,
			"policy":     defaultAutoReleasePolicy,
			"candidates": candidates,
		},
		"context_urls": map[string]string{
			"workflow": fmt.Sprintf("/api/projects/%s/protocol/workflow", projectID),
			"payouts":  fmt.Sprintf("/api/projects/%s/payouts", projectID),
			"ledger":   "/api/public/ledger",
		},
		"runbook": []map[string]any{
			{"step": 1, "action": "verify_pr", "label": "Verify PR readiness", "purpose": "Evidence, repository star, and low-risk labels are present."},
			{"step": 2, "action": "release_payout", "label": "Release escrow payout", "purpose": "Accept the task and write a task_payment ledger row."},
			{"step": 3, "action": "prove_release", "label": "Record ledger proof", "purpose": "Expose payout, PR reference, and auto-release policy in payouts."},
		},
	}
	if strings.TrimSpace(repository) != "" {
		packet["repository"] = repository
	}
	if auto {
		packet["can_auto_release"] = true
		packet["mode"] = "batch"
	} else {
		packet["can_release"] = true
		packet["mode"] = "single"
	}
	return packet
}
