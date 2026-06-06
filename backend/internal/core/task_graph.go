package core

import (
	"errors"
	"fmt"
	"net/http"
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
	return workflowProtocolDocument(project, s.projectTaskGraphLocked(project), s.projectAIWorkflowLocked(project)), nil
}

func (s *Store) PublicProjectWorkflowProtocol(projectID string) (WorkflowProtocolDocument, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	project, ok := s.projects[strings.TrimSpace(projectID)]
	if !ok {
		return WorkflowProtocolDocument{}, errors.New("project not found")
	}
	graph := s.projectTaskGraphLocked(project)
	document := workflowProtocolDocument(project, graph, s.projectAIWorkflowLocked(project))
	return publicWorkflowProtocolDocument(document, graph), nil
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

func workflowProtocolDocument(project *Project, graph ProjectTaskGraphResponse, aiWorkflow ProjectAIWorkflowResponse) WorkflowProtocolDocument {
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

	currentStep := workflowProtocolCurrentStepWithAI(graph, aiWorkflow)
	stages := workflowProtocolStages(aiWorkflow)
	return WorkflowProtocolDocument{
		ProtocolVersion: "mergeos.workflow.v1",
		Kind:            "workflow",
		ID:              project.ID + ":workflow",
		ProjectID:       project.ID,
		Status:          workflowProtocolStatus(graph),
		Progress:        graph.Progress,
		CurrentStep:     currentStep,
		Nodes:           nodes,
		Edges:           edges,
		Stages:          stages,
		Checks:          workflowProtocolChecks(stages, graph, currentStep),
		NextActions:     workflowProtocolNextActions(project.ID, currentStep, nodes),
		Evidence:        workflowProtocolEvidence(aiWorkflow.Signals),
		Metadata: map[string]any{
			"project_title":  graph.ProjectTitle,
			"workflow_steps": workflowProtocolSteps(),
			"current_step":   currentStep,
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

func publicWorkflowProtocolDocument(document WorkflowProtocolDocument, graph ProjectTaskGraphResponse) WorkflowProtocolDocument {
	idMap := map[string]string{}
	for index, node := range graph.Nodes {
		publicID := publicWorkflowNodeID(document.ProjectID, node, index)
		for _, value := range []string{node.ID, node.TaskID} {
			value = strings.TrimSpace(value)
			if value != "" {
				idMap[value] = publicID
			}
		}
	}

	for index := range document.Nodes {
		node := &document.Nodes[index]
		publicID := idMap[strings.TrimSpace(node.ID)]
		if publicID == "" {
			publicID = idMap[strings.TrimSpace(node.TaskID)]
		}
		if publicID == "" {
			publicID = publicTaskProtocolID(fmt.Sprintf("%s:node:%d", document.ProjectID, index+1))
		}
		node.ID = publicID
		node.TaskID = publicID
		node.Dependencies = publicWorkflowIDList(node.Dependencies, idMap)
	}
	for index := range document.NextActions {
		action := &document.NextActions[index]
		if publicID := idMap[strings.TrimSpace(action.TaskID)]; publicID != "" {
			action.TaskID = publicID
		}
		if publicID := idMap[strings.TrimSpace(action.TargetNodeID)]; publicID != "" {
			action.TargetNodeID = publicID
		}
	}

	edges := make([]WorkflowProtocolEdge, 0, len(document.Edges))
	for _, edge := range document.Edges {
		from := idMap[strings.TrimSpace(edge.From)]
		to := idMap[strings.TrimSpace(edge.To)]
		if from == "" || to == "" || from == to {
			continue
		}
		edges = append(edges, WorkflowProtocolEdge{
			From:     from,
			To:       to,
			Relation: edge.Relation,
		})
	}
	document.ID = document.ProjectID + ":public-workflow"
	document.Edges = edges
	if document.Metadata == nil {
		document.Metadata = map[string]any{}
	}
	document.Metadata["public"] = true
	document.Metadata["task_protocol_endpoint"] = "/api/public/protocol/tasks"
	document.Metadata["ai_workflow_endpoint"] = "/api/public/projects/{id}/ai-workflow"
	document.Metadata["pr_monitor_endpoint"] = "/api/public/projects/{id}/pull-requests"
	document.Metadata["workflow_endpoint"] = "/api/public/projects/{id}/workflow"
	return document
}

func workflowProtocolStages(aiWorkflow ProjectAIWorkflowResponse) []WorkflowProtocolStage {
	if len(aiWorkflow.Stages) == 0 {
		return nil
	}
	stages := make([]WorkflowProtocolStage, 0, len(aiWorkflow.Stages))
	for _, stage := range aiWorkflow.Stages {
		stages = append(stages, WorkflowProtocolStage{
			ID:                stage.ID,
			Title:             stage.Title,
			Summary:           stage.Body,
			Status:            stage.Status,
			Tone:              stage.Tone,
			ArtifactKind:      stage.ArtifactKind,
			InputEndpoint:     stage.InputEndpoint,
			OutputEndpoint:    stage.OutputEndpoint,
			OutputProtocol:    stage.OutputProtocol,
			OutputProtocolURL: stage.OutputProtocolURL,
			ActionEndpoint:    stage.ActionEndpoint,
			ContextURLs:       stage.ContextURLs,
			Checklist:         append([]string(nil), stage.Checklist...),
			OutputIDs:         stage.OutputIDs,
			ProducedCount:     stage.ProducedCount,
			Reference:         stage.Reference,
			URL:               stage.URL,
			UpdatedAt:         stage.UpdatedAt,
		})
	}
	return stages
}

func workflowProtocolChecks(stages []WorkflowProtocolStage, graph ProjectTaskGraphResponse, currentStep string) []WorkflowProtocolCheck {
	if len(stages) == 0 {
		return nil
	}
	checks := make([]WorkflowProtocolCheck, 0, len(stages))
	workflowBlocked := workflowProtocolStatus(graph) == "blocked"
	for _, stage := range stages {
		status := workflowProtocolCheckStatus(stage.Status)
		if workflowBlocked && stage.ID == currentStep {
			status = "blocked"
		}
		checks = append(checks, WorkflowProtocolCheck{
			ID:       "check:" + stage.ID,
			StageID:  stage.ID,
			Title:    stage.Title,
			Status:   status,
			Required: true,
			Summary:  stage.Summary,
		})
	}
	return checks
}

func workflowProtocolCheckStatus(stageStatus string) string {
	switch stageStatus {
	case deploymentStageComplete:
		return "passed"
	case deploymentStageInProgress:
		return "running"
	default:
		return "pending"
	}
}

func workflowProtocolNextActions(projectID, currentStep string, nodes []WorkflowProtocolNode) []WorkflowProtocolAction {
	actions := []WorkflowProtocolAction{}
	add := func(action WorkflowProtocolAction) {
		if strings.TrimSpace(action.ID) == "" || strings.TrimSpace(action.Type) == "" {
			return
		}
		actions = append(actions, action)
	}

	switch currentStep {
	case "repo_import":
		add(WorkflowProtocolAction{
			ID:         "next:repo-import",
			Type:       "import_repository",
			Label:      "Import repository context",
			TargetStep: currentStep,
			Method:     http.MethodPost,
			Endpoint:   "/api/public/repo/issues",
		})
	case "issue_scan":
		add(WorkflowProtocolAction{
			ID:         "next:repo-sync",
			Type:       "sync_repository_issues",
			Label:      "Sync repository issues",
			TargetStep: currentStep,
			Method:     http.MethodPost,
			Endpoint:   fmt.Sprintf("/api/projects/%s/repo-sync", projectID),
		})
	case "task_generation", "reward_estimation":
		add(WorkflowProtocolAction{
			ID:         "next:estimate-scope",
			Type:       "evaluate_scope",
			Label:      "Evaluate scope and reward allocation",
			TargetStep: currentStep,
			Method:     http.MethodPost,
			Endpoint:   "/api/projects/evaluate-price",
		})
	case "contributor_routing":
		for _, node := range nodes {
			if node.Status != "ready" {
				continue
			}
			actionType := "submit_proposal"
			label := "Submit worker proposal"
			endpoint := "/api/proposals"
			if node.RequiredWorkerKind == WorkerAgent {
				actionType = "claim_with_agent"
				label = "Claim with AI agent"
				endpoint = "/api/tasks/{task_id}/accept"
			}
			add(WorkflowProtocolAction{
				ID:           fmt.Sprintf("next:route-%d", len(actions)+1),
				Type:         actionType,
				Label:        label,
				TargetStep:   currentStep,
				TargetNodeID: node.ID,
				TaskID:       node.TaskID,
				WorkerKind:   node.RequiredWorkerKind,
				Method:       http.MethodPost,
				Endpoint:     endpoint,
			})
			if len(actions) >= 4 {
				break
			}
		}
	case "pr_review":
		for _, node := range nodes {
			if node.Status != "accepted" {
				continue
			}
			add(WorkflowProtocolAction{
				ID:           fmt.Sprintf("next:review-%d", len(actions)+1),
				Type:         "record_agent_review",
				Label:        "Record AI review evidence",
				TargetStep:   currentStep,
				TargetNodeID: node.ID,
				TaskID:       node.TaskID,
				WorkerKind:   WorkerAgent,
				Method:       http.MethodPost,
				Endpoint:     fmt.Sprintf("/api/projects/%s/agent-actions", projectID),
			})
			if len(actions) >= 3 {
				break
			}
		}
		if len(actions) == 0 {
			for _, node := range nodes {
				if node.Status != "ready" {
					continue
				}
				actionType := "submit_proposal"
				label := "Submit worker proposal"
				endpoint := "/api/proposals"
				if node.RequiredWorkerKind == WorkerAgent {
					actionType = "claim_with_agent"
					label = "Claim with AI agent"
					endpoint = "/api/tasks/{task_id}/accept"
				}
				add(WorkflowProtocolAction{
					ID:           fmt.Sprintf("next:route-%d", len(actions)+1),
					Type:         actionType,
					Label:        label,
					TargetStep:   "contributor_routing",
					TargetNodeID: node.ID,
					TaskID:       node.TaskID,
					WorkerKind:   node.RequiredWorkerKind,
					Method:       http.MethodPost,
					Endpoint:     endpoint,
				})
				if len(actions) >= 4 {
					break
				}
			}
		}
	case "deployment_validation":
		add(WorkflowProtocolAction{
			ID:         "next:deployment-evidence",
			Type:       "record_deployment_evidence",
			Label:      "Record deployment validation",
			TargetStep: currentStep,
			WorkerKind: WorkerAgent,
			Method:     http.MethodPost,
			Endpoint:   fmt.Sprintf("/api/projects/%s/agent-actions", projectID),
		})
	}
	return actions
}

func workflowProtocolEvidence(signals []AIWorkflowSignal) []WorkflowProtocolEvidence {
	if len(signals) == 0 {
		return nil
	}
	evidence := make([]WorkflowProtocolEvidence, 0, len(signals))
	for _, signal := range signals {
		evidence = append(evidence, WorkflowProtocolEvidence{
			ID:        signal.ID,
			Type:      signal.Type,
			Title:     signal.Title,
			Status:    signal.Status,
			Reference: signal.Reference,
			URL:       signal.URL,
			CreatedAt: signal.CreatedAt,
		})
	}
	return evidence
}

func publicWorkflowNodeID(projectID string, node TaskGraphNode, index int) string {
	if node.IssueNumber > 0 {
		return publicTaskProtocolID(marketplaceBountyID(projectID, node.IssueNumber))
	}
	return publicTaskProtocolID(fmt.Sprintf("%s:node:%d", projectID, index+1))
}

func publicWorkflowIDList(values []string, idMap map[string]string) []string {
	if len(values) == 0 {
		return nil
	}
	ids := []string{}
	seen := map[string]bool{}
	for _, value := range values {
		id := idMap[strings.TrimSpace(value)]
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		ids = append(ids, id)
	}
	return ids
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

func workflowProtocolCurrentStepWithAI(graph ProjectTaskGraphResponse, aiWorkflow ProjectAIWorkflowResponse) string {
	graphStep := workflowProtocolCurrentStep(graph)
	aiStep := strings.TrimSpace(aiWorkflow.CurrentStep)
	if aiStep == "" {
		return graphStep
	}
	if aiStep == "deployment_validation" && graph.Stats.NodeCount > 0 && graph.Stats.CompleteCount < graph.Stats.NodeCount {
		return graphStep
	}
	return aiStep
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
