package core

import (
	"errors"
	"strings"
	"time"
)

func (s *Store) ProjectDashboard(projectID string) (ProjectDashboardResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	project, ok := s.projects[strings.TrimSpace(projectID)]
	if !ok {
		return ProjectDashboardResponse{}, errors.New("project not found")
	}

	escrow := s.projectEscrowLocked(project)
	payouts := s.projectPayoutsLocked(project)
	deployment := s.projectDeploymentLocked(project)
	aiWorkflow := s.projectAIWorkflowLocked(project)
	taskGraph := s.projectTaskGraphLocked(project)
	repositoryScan := s.projectRepositoryScanLocked(project)
	updatedAt := latestTime(
		project.CreatedAt,
		escrow.UpdatedAt,
		payouts.UpdatedAt,
		deployment.UpdatedAt,
		aiWorkflow.UpdatedAt,
		taskGraph.UpdatedAt,
		repositoryScan.UpdatedAt,
	)

	return ProjectDashboardResponse{
		ProtocolVersion: "mergeos.customer-dashboard.v1",
		Kind:            "customer_dashboard",
		Project:         projectDashboardOverview(project, updatedAt),
		Escrow:          escrow,
		Payouts:         payouts,
		Deployment:      deployment,
		AIWorkflow:      aiWorkflow,
		TaskGraph:       taskGraph,
		RepositoryScan:  repositoryScan,
		PullRequests: ProjectPullRequestsResponse{
			ProjectID:    project.ID,
			ProjectTitle: publicLiveFeedProjectTitle(project),
			Tasks:        []ProjectTaskPullRequests{},
			UpdatedAt:    updatedAt,
		},
		UpdatedAt: updatedAt,
	}, nil
}

func projectDashboardOverview(project *Project, updatedAt time.Time) ProjectDashboardOverview {
	overview := ProjectDashboardOverview{
		ProjectID:      project.ID,
		Title:          publicLiveFeedProjectTitle(project),
		Brief:          compactText(project.Brief),
		SiteType:       project.SiteType,
		PackageTier:    project.PackageTier,
		Timeline:       project.Timeline,
		Status:         project.Status,
		RepoProvider:   project.RepoProvider,
		RepoURL:        marketplacePublicRepoURL(project.RepoURL),
		BountyRepoName: project.BountyRepoName,
		BudgetCents:    project.BudgetCents,
		FeeCents:       project.FeeCents,
		WorkPoolCents:  project.WorkPoolCents,
		CreatedAt:      project.CreatedAt,
		UpdatedAt:      updatedAt,
	}
	for _, task := range project.Tasks {
		if task == nil {
			continue
		}
		overview.TaskCount++
		switch task.Status {
		case TaskAccepted:
			overview.AcceptedTaskCount++
		default:
			overview.OpenTaskCount++
		}
		switch task.RequiredWorkerKind {
		case WorkerAgent:
			overview.AgentTaskCount++
		case WorkerHybrid:
			overview.HybridTaskCount++
		default:
			overview.HumanTaskCount++
		}
		if taskUpdatedAt := deploymentTaskUpdatedAt(task); taskUpdatedAt.After(overview.UpdatedAt) {
			overview.UpdatedAt = taskUpdatedAt
		}
	}
	return overview
}

func latestTime(values ...time.Time) time.Time {
	latest := time.Time{}
	for _, value := range values {
		if value.After(latest) {
			latest = value
		}
	}
	return latest
}
