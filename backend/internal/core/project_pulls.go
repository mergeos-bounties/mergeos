package core

import (
	"context"
	"strings"
)

type projectPullRequestLister interface {
	listPullRequestsLinkedToIssue(ctx context.Context, target githubIssueTarget) ([]AdminTaskPullRequest, error)
}

func projectPullRequestsMonitor(ctx context.Context, lister projectPullRequestLister, project *Project) ProjectPullRequestsResponse {
	response := ProjectPullRequestsResponse{
		ProjectID:    project.ID,
		ProjectTitle: publicLiveFeedProjectTitle(project),
		Tasks:        []ProjectTaskPullRequests{},
		UpdatedAt:    project.CreatedAt,
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

	for _, task := range tasks {
		row := ProjectTaskPullRequests{
			TaskID:        task.ID,
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
		if row.UpdatedAt.After(response.UpdatedAt) {
			response.UpdatedAt = row.UpdatedAt
		}
		response.Tasks = append(response.Tasks, row)
	}
	return response
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
