package core

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

func (s *Store) ProjectTaskGraph(projectID string) (ProjectTaskGraphResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	project, ok := s.projects[strings.TrimSpace(projectID)]
	if !ok {
		return ProjectTaskGraphResponse{}, errors.New("project not found")
	}
	return s.projectTaskGraphLocked(project), nil
}

func (s *Store) ProjectWorkflowProtocol(projectID string) (WorkflowProtocolDocument, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	project, ok := s.projects[strings.TrimSpace(projectID)]
	if !ok {
		return WorkflowProtocolDocument{}, errors.New("project not found")
	}
	return workflowProtocolDocument(project, s.projectTaskGraphLocked(project)), nil
}

func (s *Store) projectTaskGraphLocked(project *Project) ProjectTaskGraphResponse {
	tasks := s.projectDeploymentTasksLocked(project)
	nodes := make([]TaskGraphNode, 0, len(tasks))
	nodeIndex := map[string]int{}
	for _, task := range tasks {
		node := taskGraphNode(task)
		nodeIndex[task.ID] = len(nodes)
		nodes = append(nodes, node)
	}

	edges := taskGraphEdges(tasks)
	blockedBy := map[string][]string{}
	accepted := map[string]bool{}
	for _, task := range tasks {
		accepted[task.ID] = task.Status == TaskAccepted
	}
	for _, edge := range edges {
		if !accepted[edge.From] {
			blockedBy[edge.To] = append(blockedBy[edge.To], edge.From)
		}
	}

	updatedAt := project.CreatedAt
	stats := TaskGraphStats{
		NodeCount: len(nodes),
		EdgeCount: len(edges),
	}
	for index := range nodes {
		node := &nodes[index]
		if node.UpdatedAt.After(updatedAt) {
			updatedAt = node.UpdatedAt
		}
		node.BlockedBy = stableTaskIDs(blockedBy[node.TaskID])
		node.Ready = len(node.BlockedBy) == 0
		if node.Status == string(TaskAccepted) {
			stats.CompleteCount++
			continue
		}
		stats.OpenCount++
		if node.Ready {
			stats.ReadyCount++
		} else {
			stats.BlockedCount++
		}
	}

	progress := 0
	if stats.NodeCount > 0 {
		progress = stats.CompleteCount * 100 / stats.NodeCount
	}
	status := "queued"
	if stats.NodeCount > 0 && stats.CompleteCount == stats.NodeCount {
		status = "ready"
	} else if stats.CompleteCount > 0 || stats.ReadyCount > 0 {
		status = "planning"
	}

	sort.Slice(edges, func(i, j int) bool {
		leftFrom := nodeIndex[edges[i].From]
		rightFrom := nodeIndex[edges[j].From]
		if leftFrom != rightFrom {
			return leftFrom < rightFrom
		}
		return nodeIndex[edges[i].To] < nodeIndex[edges[j].To]
	})

	return ProjectTaskGraphResponse{
		ProjectID:    project.ID,
		ProjectTitle: publicLiveFeedProjectTitle(project),
		Status:       status,
		Progress:     progress,
		Stats:        stats,
		Nodes:        nodes,
		Edges:        edges,
		UpdatedAt:    updatedAt,
	}
}

func taskGraphNode(task *Task) TaskGraphNode {
	status := string(task.Status)
	if status == "" {
		status = string(TaskOpen)
	}
	return TaskGraphNode{
		ID:                 task.ID,
		TaskID:             task.ID,
		IssueNumber:        task.IssueNumber,
		Title:              task.Title,
		Lane:               taskGraphLane(task),
		Status:             status,
		RewardCents:        task.RewardCents,
		EstimatedHours:     marketplaceEstimatedHours(task),
		RequiredWorkerKind: task.RequiredWorkerKind,
		SuggestedAgentType: strings.TrimSpace(task.SuggestedAgentType),
		IssueURL:           marketplacePublicRepoURL(task.IssueURL),
		UpdatedAt:          deploymentTaskUpdatedAt(task),
	}
}

func taskGraphEdges(tasks []*Task) []TaskGraphEdge {
	edges := []TaskGraphEdge{}
	seen := map[string]bool{}
	add := func(from, to, relation string) {
		if strings.TrimSpace(from) == "" || strings.TrimSpace(to) == "" || from == to {
			return
		}
		id := fmt.Sprintf("%s>%s:%s", from, to, relation)
		if seen[id] {
			return
		}
		seen[id] = true
		edges = append(edges, TaskGraphEdge{
			ID:       id,
			From:     from,
			To:       to,
			Relation: relation,
		})
	}

	for index, task := range tasks {
		if task == nil || index == 0 {
			continue
		}
		previous := tasks[index-1]
		if previous != nil {
			add(previous.ID, task.ID, "sequence")
		}
		lane := taskGraphLane(task)
		if lane != "validation" && lane != "deployment" {
			continue
		}
		for _, dependency := range tasks[:index] {
			if dependency == nil {
				continue
			}
			dependencyLane := taskGraphLane(dependency)
			if dependencyLane == "implementation" || dependencyLane == "backend" || dependencyLane == "design" || dependencyLane == "agent" {
				add(dependency.ID, task.ID, lane+"_dependency")
			}
		}
	}
	return edges
}

func taskGraphLane(task *Task) string {
	haystack := strings.ToLower(strings.Join([]string{
		task.Title,
		task.Acceptance,
		string(task.RequiredWorkerKind),
		task.SuggestedAgentType,
		task.AgentType,
	}, " "))
	switch {
	case containsAny(haystack, []string{"qa", "quality", "test", "review", "a11y", "accessibility", "validation"}):
		return "validation"
	case containsAny(haystack, []string{"deploy", "deployment", "devops", "release", "handoff", "pipeline"}):
		return "deployment"
	case containsAny(haystack, []string{"design", "ui", "ux", "layout", "visual"}):
		return "design"
	case containsAny(haystack, []string{"api", "backend", "database", "postgres", "redis", "auth"}):
		return "backend"
	case task.RequiredWorkerKind == WorkerAgent:
		return "agent"
	default:
		return "implementation"
	}
}

func containsAny(value string, keywords []string) bool {
	for _, keyword := range keywords {
		if strings.Contains(value, keyword) {
			return true
		}
	}
	return false
}

func stableTaskIDs(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	rows := append([]string(nil), values...)
	sort.Strings(rows)
	return rows
}

func taskGraphUpdatedAt(tasks []*Task, fallback time.Time) time.Time {
	updatedAt := fallback
	for _, task := range tasks {
		if task == nil {
			continue
		}
		if taskUpdatedAt := deploymentTaskUpdatedAt(task); taskUpdatedAt.After(updatedAt) {
			updatedAt = taskUpdatedAt
		}
	}
	return updatedAt
}

func workflowProtocolDocument(project *Project, graph ProjectTaskGraphResponse) WorkflowProtocolDocument {
	dependenciesByTaskID := map[string][]string{}
	for _, edge := range graph.Edges {
		dependenciesByTaskID[edge.To] = append(dependenciesByTaskID[edge.To], edge.From)
	}

	nodes := make([]WorkflowProtocolNode, 0, len(graph.Nodes))
	for _, node := range graph.Nodes {
		nodes = append(nodes, WorkflowProtocolNode{
			ID:                 node.ID,
			TaskID:             node.TaskID,
			IssueNumber:        node.IssueNumber,
			Title:              node.Title,
			Lane:               node.Lane,
			Status:             workflowProtocolNodeStatus(node),
			RewardMRG:          float64(node.RewardCents) / 100,
			EstimatedHours:     node.EstimatedHours,
			RequiredWorkerKind: node.RequiredWorkerKind,
			SuggestedAgentType: node.SuggestedAgentType,
			IssueURL:           node.IssueURL,
			Dependencies:       stableTaskIDs(dependenciesByTaskID[node.TaskID]),
		})
	}

	edges := make([]WorkflowProtocolEdge, 0, len(graph.Edges))
	for _, edge := range graph.Edges {
		edges = append(edges, WorkflowProtocolEdge{
			From:     edge.From,
			To:       edge.To,
			Relation: edge.Relation,
		})
	}

	return WorkflowProtocolDocument{
		ProtocolVersion: "mergeos.workflow.v1",
		Kind:            "workflow",
		ID:              project.ID + ":workflow",
		ProjectID:       project.ID,
		Status:          workflowProtocolStatus(graph),
		Nodes:           nodes,
		Edges:           edges,
		Metadata: map[string]any{
			"project_title":  graph.ProjectTitle,
			"workflow_steps": workflowProtocolSteps(),
			"current_step":   workflowProtocolCurrentStep(graph),
			"progress":       graph.Progress,
			"node_count":     graph.Stats.NodeCount,
			"edge_count":     graph.Stats.EdgeCount,
			"ready_count":    graph.Stats.ReadyCount,
			"blocked_count":  graph.Stats.BlockedCount,
			"complete_count": graph.Stats.CompleteCount,
			"updated_at":     graph.UpdatedAt,
		},
	}
}

func workflowProtocolSteps() []string {
	return []string{
		"repo_import",
		"issue_scan",
		"task_generation",
		"reward_estimation",
		"contributor_routing",
		"pr_review",
		"deployment_validation",
	}
}

func workflowProtocolCurrentStep(graph ProjectTaskGraphResponse) string {
	if graph.Stats.NodeCount == 0 {
		return "repo_import"
	}
	if graph.Stats.CompleteCount == graph.Stats.NodeCount {
		return "deployment_validation"
	}
	if graph.Stats.CompleteCount > 0 {
		return "pr_review"
	}
	if graph.Stats.ReadyCount > 0 || graph.Stats.BlockedCount > 0 {
		return "contributor_routing"
	}
	return "task_generation"
}

func workflowProtocolStatus(graph ProjectTaskGraphResponse) string {
	if graph.Stats.NodeCount > 0 && graph.Stats.CompleteCount == graph.Stats.NodeCount {
		return "ready"
	}
	if graph.Stats.BlockedCount > 0 && graph.Stats.ReadyCount == 0 && graph.Stats.CompleteCount == 0 {
		return "blocked"
	}
	switch graph.Status {
	case "ready":
		return "ready"
	case "planning":
		return "active"
	default:
		return "planned"
	}
}

func workflowProtocolNodeStatus(node TaskGraphNode) string {
	if node.Status == string(TaskAccepted) {
		return "accepted"
	}
	if node.Ready {
		return "ready"
	}
	if len(node.BlockedBy) > 0 {
		return "blocked"
	}
	return "open"
}
