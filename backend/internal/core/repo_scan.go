package core

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const (
	maxRepositoryScanFiles             = 400
	maxRepositoryScanBytes             = 256 * 1024
	maxRepositoryScanFindings          = 80
	maxRepositorySuggestedTasks        = 16
	repositoryScanSuggestionBountyType = "repo_scan_suggestion"
)

var repositorySecretPattern = regexp.MustCompile(`(?i)\b[A-Z0-9_.-]*(api[_-]?key|secret|password|token|private[_-]?key)[A-Z0-9_.-]*\b\s*[:=]\s*['"]?[A-Za-z0-9_./+=-]{8,}`)

func (s *Store) ProjectRepositoryScan(projectID string) (ProjectRepositoryScanResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	project, ok := s.projects[strings.TrimSpace(projectID)]
	if !ok {
		return ProjectRepositoryScanResponse{}, errors.New("project not found")
	}
	return s.projectRepositoryScanLocked(project), nil
}

func (s *Store) ProjectRepositoryScanProtocol(projectID string) (RepositoryScanProtocolDocument, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	project, ok := s.projects[strings.TrimSpace(projectID)]
	if !ok {
		return RepositoryScanProtocolDocument{}, errors.New("project not found")
	}
	scan := s.projectRepositoryScanLocked(project)
	return repositoryScanProtocolDocument(project, scan), nil
}

func (s *Store) PublicProjectRepositoryScanProtocol(projectID string) (RepositoryScanProtocolDocument, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	project, ok := s.projects[strings.TrimSpace(projectID)]
	if !ok {
		return RepositoryScanProtocolDocument{}, errors.New("project not found")
	}
	scan := s.projectRepositoryScanLocked(project)
	return repositoryScanProtocolDocument(project, scan), nil
}

func (s *Store) projectRepositoryScanLocked(project *Project) ProjectRepositoryScanResponse {
	response := ProjectRepositoryScanResponse{
		ProjectID:    project.ID,
		ProjectTitle: publicLiveFeedProjectTitle(project),
		Status:       "unavailable",
		Summary:      "Repository files are not available for static scanning.",
		UpdatedAt:    project.CreatedAt,
	}

	root := strings.TrimSpace(project.RepoLocalPath)
	if root == "" {
		return response
	}
	absRoot, err := filepath.Abs(root)
	if err != nil || !repositoryScanRootAllowed(absRoot, s.cfg.BountyRoot) {
		response.Summary = "Repository path is outside the configured bounty workspace."
		return response
	}
	info, err := os.Stat(absRoot)
	if err != nil || !info.IsDir() {
		return response
	}

	languages := map[string]*RepositoryLanguage{}
	dependencies := []RepositoryDependencyFile{}
	findings := []RepositoryScanFinding{}
	addFinding := func(severity, category, title, body, path string, line int, signal string) {
		if len(findings) >= maxRepositoryScanFindings {
			return
		}
		findings = append(findings, RepositoryScanFinding{
			ID:       fmt.Sprintf("repo-finding-%03d", len(findings)+1),
			Severity: severity,
			Category: category,
			Title:    title,
			Body:     body,
			Path:     path,
			Line:     line,
			Signal:   signal,
		})
	}

	err = filepath.WalkDir(absRoot, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			response.Stats.SkippedFiles++
			return nil
		}
		if path == absRoot {
			return nil
		}
		if entry.IsDir() {
			if repositoryScanSkipDir(entry.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		response.Stats.FileCount++
		if response.Stats.FileCount > maxRepositoryScanFiles {
			response.Stats.SkippedFiles++
			return nil
		}

		relativePath := repositoryRelativePath(absRoot, path)
		info, err := entry.Info()
		if err != nil {
			response.Stats.SkippedFiles++
			return nil
		}
		if info.ModTime().After(response.UpdatedAt) {
			response.UpdatedAt = info.ModTime()
		}
		trackRepositoryLanguage(languages, relativePath)
		if !repositoryScanTextFile(relativePath) || info.Size() > maxRepositoryScanBytes {
			response.Stats.SkippedFiles++
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			response.Stats.SkippedFiles++
			return nil
		}
		response.Stats.ScannedFiles++
		if dependency := repositoryDependencyFile(absRoot, relativePath, content); dependency.Path != "" {
			dependencies = append(dependencies, dependency)
			if title, body := repositoryMissingLockfileFinding(dependency); !dependency.HasLockfile && title != "" {
				addFinding("medium", "dependency", title, body, relativePath, 0, "lockfile_missing")
			}
		}
		repositoryDependencyFindings(relativePath, content, addFinding)
		repositoryContentFindings(relativePath, string(content), addFinding)
		return nil
	})
	if err != nil {
		response.Summary = "Repository scan failed before completion."
		return response
	}

	response.Status = "ready"
	response.Dependencies = dependencies
	response.Languages = repositoryLanguageRows(languages)
	response.Findings = findings
	response.Stats.DependencyFiles = len(dependencies)
	response.Stats.FindingCount = len(findings)
	response.SuggestedTasks = repositorySuggestedTasks(project, findings, s.cfg.PlatformFeeBps)
	response.Stats.SuggestedTaskCount = len(response.SuggestedTasks)
	response.Summary = fmt.Sprintf("Scanned %d text files across %d repository files.", response.Stats.ScannedFiles, response.Stats.FileCount)
	return response
}

func repositoryScanProtocolDocument(project *Project, scan ProjectRepositoryScanResponse) RepositoryScanProtocolDocument {
	findings := scan.Findings
	if findings == nil {
		findings = []RepositoryScanFinding{}
	}
	suggestedTasks := scan.SuggestedTasks
	if suggestedTasks == nil {
		suggestedTasks = []RepositorySuggestedTask{}
	}
	return RepositoryScanProtocolDocument{
		ProtocolVersion: "mergeos.scan.v1",
		Kind:            "repository_scan",
		ID:              "scan:" + scan.ProjectID,
		ProjectID:       scan.ProjectID,
		ProjectTitle:    scan.ProjectTitle,
		Status:          scan.Status,
		Summary:         scan.Summary,
		SourceRepo:      projectSourceRepoURL(project),
		UpdatedAt:       scan.UpdatedAt,
		Stats:           scan.Stats,
		Languages:       scan.Languages,
		Dependencies:    scan.Dependencies,
		Findings:        findings,
		SuggestedTasks:  suggestedTasks,
		Metadata: map[string]any{
			"finding_count":        scan.Stats.FindingCount,
			"dependency_files":     scan.Stats.DependencyFiles,
			"suggested_task_count": scan.Stats.SuggestedTaskCount,
		},
	}
}

func repositorySuggestedTasks(project *Project, findings []RepositoryScanFinding, platformFeeBps int64) []RepositorySuggestedTask {
	if project == nil || len(findings) == 0 {
		return []RepositorySuggestedTask{}
	}
	alreadyFunded := repositoryFundedSuggestionIDs(project)
	capacity := len(findings)
	if capacity > maxRepositorySuggestedTasks {
		capacity = maxRepositorySuggestedTasks
	}
	tasks := make([]RepositorySuggestedTask, 0, capacity)
	for _, finding := range findings {
		if len(tasks) >= maxRepositorySuggestedTasks {
			break
		}
		if strings.TrimSpace(finding.ID) == "" || strings.TrimSpace(finding.Signal) == "" {
			continue
		}
		task := repositorySuggestedTaskFromFinding(project, finding, platformFeeBps)
		if alreadyFunded[finding.ID] {
			task.ReadyForBounty = false
			task.FundingPacket.Status = "already_funded"
			task.FundingPacket.CanFund = false
		}
		tasks = append(tasks, task)
	}
	return tasks
}

func repositorySuggestedTaskFromFinding(project *Project, finding RepositoryScanFinding, platformFeeBps int64) RepositorySuggestedTask {
	lane, workerKind, agentType := repositorySuggestedTaskRouting(finding)
	if !projectAllowsAgents(project) {
		workerKind = WorkerHuman
		agentType = ""
	}
	rewardCents := repositorySuggestedTaskRewardCents(finding)
	fundingCents := repositoryFundingCentsForReward(rewardCents, platformFeeBps)
	taskID := repositorySuggestedTaskID(finding)
	fundEndpoint := fmt.Sprintf("/api/projects/%s/repo-scan/suggested-tasks/%s/fund", project.ID, taskID)
	payPalOrderEndpoint := fmt.Sprintf("/api/projects/%s/repo-scan/suggested-tasks/%s/paypal-order", project.ID, taskID)
	criteria := repositorySuggestedTaskAcceptanceCriteria(finding)
	evidence := repositorySuggestedTaskEvidenceChecklist(finding)
	return RepositorySuggestedTask{
		ID:                   taskID,
		SourceFindingID:      finding.ID,
		Signal:               finding.Signal,
		Title:                repositorySuggestedTaskTitle(finding),
		Body:                 finding.Body,
		Severity:             finding.Severity,
		Lane:                 lane,
		Path:                 finding.Path,
		EstimatedRewardCents: rewardCents,
		EstimatedHours:       repositorySuggestedTaskHours(finding.Severity),
		WorkerKind:           workerKind,
		SuggestedAgentType:   agentType,
		ReadyForBounty:       true,
		AcceptanceCriteria:   criteria,
		EvidenceRequired:     evidence,
		FundingPacket: RepositoryFundingPacket{
			Status:                  "ready",
			CanFund:                 true,
			RecommendedRewardCents:  rewardCents,
			RecommendedFundingCents: fundingCents,
			FundEndpoint:            fundEndpoint,
			PayPalOrderEndpoint:     payPalOrderEndpoint,
			FundPayload: map[string]any{
				"suggested_task_id": taskID,
				"source_finding_id": finding.ID,
				"signal":            finding.Signal,
				"reward_cents":      rewardCents,
				"budget_cents":      fundingCents,
			},
			PayPalOrderPayload: map[string]any{
				"suggested_task_id": taskID,
				"source_finding_id": finding.ID,
				"reward_cents":      rewardCents,
				"budget_cents":      fundingCents,
				"flow":              PaymentOrderFlowRepositoryTaskFunding,
			},
			EvidenceChecklist: evidence,
		},
		RoutingPacket: repositorySuggestedTaskRoutingPacket(project, finding, taskID, lane, workerKind, agentType, fundEndpoint, criteria, evidence),
	}
}

func repositorySuggestedTaskRoutingPacket(project *Project, finding RepositoryScanFinding, taskID, lane string, workerKind WorkerKind, agentType, fundEndpoint string, criteria, evidence []string) ProjectRoutingPacket {
	projectID := ""
	if project != nil {
		projectID = strings.TrimSpace(project.ID)
	}
	contextURLs := map[string]string{
		"scan_protocol":     "/api/public/projects/" + projectID + "/repo-scan",
		"workflow_protocol": "/api/public/projects/" + projectID + "/workflow",
		"workflow_pulse":    "/api/public/projects/" + projectID + "/ai-workflow",
		"marketplace":       "/api/public/marketplace",
		"agent_queue":       agentQueueEndpoint,
	}
	if repo := projectSourceRepoURL(project); strings.TrimSpace(repo) != "" {
		contextURLs["source_repository"] = repo
	}
	payload := map[string]any{
		"suggested_task_id": taskID,
		"source_finding_id": finding.ID,
		"signal":            finding.Signal,
		"lane":              lane,
		"worker_kind":       workerKind,
		"acceptance":        append([]string(nil), criteria...),
		"evidence_required": append([]string(nil), evidence...),
	}
	if agentType != "" {
		payload["agent_type"] = agentType
	}
	packet := ProjectRoutingPacket{
		Action:      repositorySuggestedTaskRoutingAction(workerKind),
		Method:      "POST",
		Endpoint:    fundEndpoint,
		Payload:     payload,
		ContextURLs: contextURLs,
		Runbook: []AgentRunbookStep{
			{Step: 1, Action: "fetch_scan", Label: "Read the repository scan signal, suggested task, and evidence requirements", Method: "GET", Endpoint: contextURLs["scan_protocol"]},
			{Step: 2, Action: "fund_bounty", Label: "Fund the suggested task to create a public bounty and escrow record", Method: "POST", Endpoint: fundEndpoint},
			{Step: 3, Action: "route_work", Label: "Route the funded bounty through marketplace, agent queue, or hybrid delivery", Method: "GET", Endpoint: contextURLs["workflow_protocol"]},
		},
		OutputContracts: []AgentOutputContract{
			{Action: "fund_bounty", ArtifactKind: "repo_task_funding", OutputEndpoint: fundEndpoint, OutputProtocol: "mergeos.repo-task-funding.v1", OutputProtocolURL: "/protocol/repo-task-funding.v1.schema.json", PublicURL: contextURLs["scan_protocol"]},
			{Action: "publish_workflow", ArtifactKind: "workflow", OutputEndpoint: contextURLs["workflow_protocol"], OutputProtocol: "mergeos.workflow.v1", OutputProtocolURL: "/protocol/workflow.v1.schema.json", PublicURL: contextURLs["workflow_protocol"]},
		},
	}
	if workerKind != WorkerHuman {
		actionEndpoint := "/api/projects/" + projectID + "/agent-actions"
		for _, action := range repositorySuggestedTaskAgentActions(finding.Signal, lane) {
			packet.OutputContracts = append(packet.OutputContracts, agentQueueOutputContract(action, projectID, actionEndpoint, contextURLs))
		}
	}
	return packet
}

func repositorySuggestedTaskRoutingAction(workerKind WorkerKind) string {
	switch workerKind {
	case WorkerAgent:
		return "fund_and_route_agent"
	case WorkerHybrid:
		return "fund_and_pair_hybrid"
	default:
		return "fund_and_publish_bounty"
	}
}

func repositorySuggestedTaskAgentActions(signal, lane string) []string {
	switch strings.TrimSpace(signal) {
	case "lockfile_missing", "dependency_unpinned":
		return []string{"scan", "test"}
	case "secret_pattern", "env_file", "dangerous_js_execution", "direct_inner_html":
		return []string{"scan", "review", "test"}
	case "production_panic":
		return []string{"review", "test"}
	default:
		if lane == "security" {
			return []string{"scan", "review", "test"}
		}
		return []string{"review", "test"}
	}
}

func repositoryFundedSuggestionIDs(project *Project) map[string]bool {
	funded := map[string]bool{}
	if project == nil {
		return funded
	}
	for _, task := range project.Tasks {
		if task == nil || task.BountyType != repositoryScanSuggestionBountyType {
			continue
		}
		for _, line := range strings.Split(task.Acceptance, "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "Source finding: ") {
				funded[strings.TrimSpace(strings.TrimPrefix(line, "Source finding: "))] = true
			}
		}
	}
	return funded
}

func repositorySuggestedTaskID(finding RepositoryScanFinding) string {
	suffix := strings.TrimSpace(strings.TrimPrefix(finding.ID, "repo-finding-"))
	if suffix == "" || suffix == finding.ID {
		suffix = slug(finding.Signal)
	}
	if suffix == "" {
		suffix = "scan"
	}
	return "repo-task-" + suffix
}

func repositorySuggestedTaskTitle(finding RepositoryScanFinding) string {
	title := strings.TrimSpace(finding.Title)
	if title == "" {
		title = "Repository scan finding"
	}
	return "Fix: " + title
}

func repositorySuggestedTaskRouting(finding RepositoryScanFinding) (string, WorkerKind, string) {
	switch strings.TrimSpace(finding.Signal) {
	case "env_file", "secret_pattern", "dangerous_js_execution", "direct_inner_html":
		return "security", WorkerHybrid, "security-review-agent"
	case "lockfile_missing", "dependency_unpinned":
		return "dependencies", WorkerAgent, "dependency-scan-agent"
	case "production_panic":
		return "backend", WorkerHybrid, "go-reliability-agent"
	case "todo_fixme":
		return "implementation", WorkerHuman, ""
	default:
		if strings.EqualFold(finding.Category, "security") {
			return "security", WorkerHybrid, "security-review-agent"
		}
		return "implementation", WorkerHybrid, "code-review-agent"
	}
}

func repositorySuggestedTaskRewardCents(finding RepositoryScanFinding) int64 {
	switch strings.ToLower(strings.TrimSpace(finding.Severity)) {
	case "critical":
		return 60000
	case "high":
		return 35000
	case "medium":
		return 20000
	default:
		return 10000
	}
}

func repositorySuggestedTaskHours(severity string) float64 {
	switch strings.ToLower(strings.TrimSpace(severity)) {
	case "critical":
		return 8
	case "high":
		return 6
	case "medium":
		return 3.5
	default:
		return 2
	}
}

func repositoryFundingCentsForReward(rewardCents, platformFeeBps int64) int64 {
	if rewardCents <= 0 {
		return 0
	}
	if platformFeeBps <= 0 || platformFeeBps >= 10000 {
		return rewardCents
	}
	divisor := 10000 - platformFeeBps
	return (rewardCents*10000 + divisor - 1) / divisor
}

func repositorySuggestedTaskAcceptanceCriteria(finding RepositoryScanFinding) []string {
	location := repositorySuggestedTaskLocation(finding)
	title := strings.TrimSpace(finding.Title)
	if title == "" {
		title = "the repository scan signal"
	}
	criteria := []string{
		fmt.Sprintf("Resolve %s without introducing regressions.", title),
		"Attach a pull request, commit, or evidence URL that references the changed files.",
	}
	switch finding.Signal {
	case "lockfile_missing", "dependency_unpinned":
		criteria = append(criteria, "Lock dependency versions and include the generated lockfile or dependency diff.")
	case "secret_pattern", "env_file":
		criteria = append(criteria, "Rotate or remove exposed secret material and document the safe replacement path.")
	case "dangerous_js_execution", "direct_inner_html":
		criteria = append(criteria, "Replace unsafe rendering or execution with a sanitized implementation and add a regression test.")
	case "production_panic":
		criteria = append(criteria, "Replace the production panic path with handled error flow and coverage.")
	default:
		criteria = append(criteria, "Include test, lint, or review notes proving the scan signal is resolved.")
	}
	if location != "" {
		criteria = append(criteria, "Reference location: "+location+".")
	}
	return criteria
}

func repositorySuggestedTaskEvidenceChecklist(finding RepositoryScanFinding) []string {
	switch finding.Signal {
	case "lockfile_missing", "dependency_unpinned":
		return []string{"dependency_diff", "lockfile_or_pin", "tests_or_install"}
	case "secret_pattern", "env_file":
		return []string{"secret_removed", "rotation_note", "scan_clean"}
	case "dangerous_js_execution", "direct_inner_html":
		return []string{"pull_request", "security_review", "regression_test"}
	case "production_panic":
		return []string{"pull_request", "error_handling_test", "runtime_evidence"}
	default:
		return []string{"pull_request", "tests_or_review", "scan_clean"}
	}
}

func repositorySuggestedTaskLocation(finding RepositoryScanFinding) string {
	path := strings.TrimSpace(finding.Path)
	if path == "" {
		return ""
	}
	if finding.Line > 0 {
		return fmt.Sprintf("%s:%d", path, finding.Line)
	}
	return path
}

func repositoryScanRootAllowed(root, bountyRoot string) bool {
	if strings.TrimSpace(bountyRoot) == "" {
		return true
	}
	absBountyRoot, err := filepath.Abs(bountyRoot)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(absBountyRoot, root)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}

func repositoryScanSkipDir(name string) bool {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case ".git", "node_modules", "vendor", "dist", "build", ".next", ".nuxt", "coverage", ".cache":
		return true
	default:
		return false
	}
}

func repositoryScanTextFile(path string) bool {
	name := strings.ToLower(filepath.Base(path))
	switch name {
	case "package.json", "go.mod", "go.sum", "requirements.txt", "pyproject.toml", "cargo.toml", "composer.json", ".env", ".env.local":
		return true
	}
	switch strings.ToLower(filepath.Ext(path)) {
	case ".go", ".js", ".jsx", ".ts", ".tsx", ".vue", ".css", ".html", ".json", ".md", ".yml", ".yaml", ".toml", ".txt", ".env":
		return true
	default:
		return false
	}
}

func repositoryRelativePath(root, path string) string {
	relativePath, err := filepath.Rel(root, path)
	if err != nil {
		return filepath.ToSlash(filepath.Base(path))
	}
	return filepath.ToSlash(relativePath)
}

func trackRepositoryLanguage(rows map[string]*RepositoryLanguage, path string) {
	extension := strings.ToLower(filepath.Ext(path))
	language := repositoryLanguage(extension, filepath.Base(path))
	if language == "" {
		return
	}
	row := rows[extension]
	if row == nil {
		row = &RepositoryLanguage{Language: language, Extension: extension}
		rows[extension] = row
	}
	row.FileCount++
}

func repositoryLanguage(extension, name string) string {
	switch strings.ToLower(name) {
	case "go.mod", "go.sum":
		return "Go"
	case "package.json":
		return "JavaScript"
	}
	switch extension {
	case ".go":
		return "Go"
	case ".js", ".jsx", ".ts", ".tsx", ".vue":
		return "JavaScript"
	case ".css", ".html":
		return "Frontend"
	case ".json", ".yml", ".yaml", ".toml":
		return "Config"
	case ".md", ".txt":
		return "Docs"
	default:
		return ""
	}
}

func repositoryDependencyFile(root, relativePath string, content []byte) RepositoryDependencyFile {
	name := strings.ToLower(filepath.Base(relativePath))
	switch name {
	case "package.json":
		return RepositoryDependencyFile{
			Path:         relativePath,
			Ecosystem:    "npm",
			PackageCount: npmDependencyCount(content),
			HasLockfile:  repositoryHasAnyFile(filepath.Join(root, filepath.Dir(relativePath)), []string{"package-lock.json", "pnpm-lock.yaml", "yarn.lock", "bun.lockb"}),
		}
	case "go.mod":
		return RepositoryDependencyFile{
			Path:         relativePath,
			Ecosystem:    "go",
			PackageCount: goModDependencyCount(string(content)),
			HasLockfile:  repositoryHasAnyFile(filepath.Join(root, filepath.Dir(relativePath)), []string{"go.sum"}),
		}
	case "requirements.txt":
		return RepositoryDependencyFile{
			Path:         relativePath,
			Ecosystem:    "python",
			PackageCount: requirementsDependencyCount(string(content)),
			HasLockfile:  false,
		}
	case "pyproject.toml":
		return RepositoryDependencyFile{
			Path:         relativePath,
			Ecosystem:    "python",
			PackageCount: pyProjectDependencyCount(string(content)),
			HasLockfile:  repositoryHasAnyFile(filepath.Join(root, filepath.Dir(relativePath)), []string{"poetry.lock", "uv.lock", "pdm.lock"}),
		}
	case "cargo.toml":
		return RepositoryDependencyFile{
			Path:         relativePath,
			Ecosystem:    "rust",
			PackageCount: cargoTomlDependencyCount(string(content)),
			HasLockfile:  repositoryHasAnyFile(filepath.Join(root, filepath.Dir(relativePath)), []string{"Cargo.lock"}),
		}
	case "composer.json":
		return RepositoryDependencyFile{
			Path:         relativePath,
			Ecosystem:    "composer",
			PackageCount: composerDependencyCount(content),
			HasLockfile:  repositoryHasAnyFile(filepath.Join(root, filepath.Dir(relativePath)), []string{"composer.lock"}),
		}
	default:
		return RepositoryDependencyFile{}
	}
}

func repositoryMissingLockfileFinding(dependency RepositoryDependencyFile) (string, string) {
	switch dependency.Ecosystem {
	case "npm":
		return "Missing npm lockfile", "package.json exists without a package-lock.json, pnpm-lock.yaml, yarn.lock, or bun.lockb in the same folder."
	case "python":
		if strings.EqualFold(filepath.Base(dependency.Path), "pyproject.toml") {
			return "Missing Python lockfile", "pyproject.toml exists without a poetry.lock, uv.lock, or pdm.lock in the same folder."
		}
	case "rust":
		return "Missing Cargo lockfile", "Cargo.toml exists without a Cargo.lock in the same folder."
	case "composer":
		return "Missing Composer lockfile", "composer.json exists without a composer.lock in the same folder."
	}
	return "", ""
}

func repositoryHasAnyFile(root string, names []string) bool {
	for _, name := range names {
		if _, err := os.Stat(filepath.Join(root, name)); err == nil {
			return true
		}
	}
	return false
}

func npmDependencyCount(content []byte) int {
	var parsed struct {
		Dependencies     map[string]any `json:"dependencies"`
		DevDependencies  map[string]any `json:"devDependencies"`
		PeerDependencies map[string]any `json:"peerDependencies"`
	}
	if err := json.Unmarshal(content, &parsed); err != nil {
		return 0
	}
	return len(parsed.Dependencies) + len(parsed.DevDependencies) + len(parsed.PeerDependencies)
}

func goModDependencyCount(content string) int {
	count := 0
	inBlock := false
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}
		if strings.HasPrefix(line, "require (") {
			inBlock = true
			continue
		}
		if inBlock && line == ")" {
			inBlock = false
			continue
		}
		if inBlock || strings.HasPrefix(line, "require ") {
			count++
		}
	}
	return count
}

func requirementsDependencyCount(content string) int {
	count := 0
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			count++
		}
	}
	return count
}

func pyProjectDependencyCount(content string) int {
	count := 0
	inProjectDependencies := false
	inPoetryDependencies := false
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(strings.Split(line, "#")[0])
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "[") {
			section := strings.ToLower(trimmed)
			inProjectDependencies = false
			inPoetryDependencies = section == "[tool.poetry.dependencies]" || section == "[tool.pdm.dependencies]"
			continue
		}
		if strings.HasPrefix(strings.ToLower(trimmed), "dependencies") && strings.Contains(trimmed, "[") {
			inProjectDependencies = true
			count += strings.Count(trimmed, `"`) / 2
			if strings.Contains(trimmed, "]") {
				inProjectDependencies = false
			}
			continue
		}
		if inProjectDependencies {
			count += strings.Count(trimmed, `"`) / 2
			if strings.Contains(trimmed, "]") {
				inProjectDependencies = false
			}
			continue
		}
		if inPoetryDependencies && strings.Contains(trimmed, "=") && !strings.HasPrefix(strings.ToLower(trimmed), "python") {
			count++
		}
	}
	return count
}

func cargoTomlDependencyCount(content string) int {
	count := 0
	inDependencies := false
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(strings.Split(line, "#")[0])
		if trimmed == "" {
			continue
		}
		if strings.HasPrefix(trimmed, "[") {
			section := strings.ToLower(trimmed)
			inDependencies = section == "[dependencies]" || section == "[dev-dependencies]" || section == "[build-dependencies]"
			continue
		}
		if inDependencies && strings.Contains(trimmed, "=") {
			count++
		}
	}
	return count
}

func composerDependencyCount(content []byte) int {
	var parsed struct {
		Require    map[string]any `json:"require"`
		RequireDev map[string]any `json:"require-dev"`
	}
	if err := json.Unmarshal(content, &parsed); err != nil {
		return 0
	}
	return len(parsed.Require) + len(parsed.RequireDev)
}

func repositoryDependencyFindings(path string, content []byte, add func(string, string, string, string, string, int, string)) {
	switch strings.ToLower(filepath.Base(path)) {
	case "package.json":
		var parsed struct {
			Dependencies     map[string]string `json:"dependencies"`
			DevDependencies  map[string]string `json:"devDependencies"`
			PeerDependencies map[string]string `json:"peerDependencies"`
		}
		if err := json.Unmarshal(content, &parsed); err != nil {
			return
		}
		if npmHasFloatingDependencyVersion(parsed.Dependencies) || npmHasFloatingDependencyVersion(parsed.DevDependencies) || npmHasFloatingDependencyVersion(parsed.PeerDependencies) {
			add("medium", "dependency", "Floating npm dependency version", "A dependency uses an unpinned or latest-style version. Pin versions before release to reduce supply-chain drift.", path, 0, "dependency_unpinned")
		}
	case "requirements.txt":
		for index, line := range strings.Split(string(content), "\n") {
			if pythonRequirementUnpinned(line) {
				add("medium", "dependency", "Unpinned Python dependency", "A Python requirement is not pinned to an exact version. Pin versions before release to reduce supply-chain drift.", path, index+1, "dependency_unpinned")
			}
		}
	}
}

func npmHasFloatingDependencyVersion(dependencies map[string]string) bool {
	for _, version := range dependencies {
		version = strings.ToLower(strings.TrimSpace(version))
		if version == "" || version == "*" || version == "latest" || strings.HasPrefix(version, "file:") || strings.HasPrefix(version, "git+") {
			return true
		}
	}
	return false
}

func pythonRequirementUnpinned(line string) bool {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "-") {
		return false
	}
	return !strings.Contains(line, "==")
}

func repositoryContentFindings(path, content string, add func(string, string, string, string, string, int, string)) {
	name := strings.ToLower(filepath.Base(path))
	if strings.HasPrefix(name, ".env") {
		add("high", "secret_hygiene", "Environment file committed", "Environment-style files should not be committed unless they are sanitized examples.", path, 0, "env_file")
	}
	for index, line := range strings.Split(content, "\n") {
		lineNumber := index + 1
		lower := strings.ToLower(line)
		if strings.Contains(lower, "todo") || strings.Contains(lower, "fixme") {
			add("low", "technical_debt", "Open TODO/FIXME marker", "A TODO or FIXME marker needs triage before release planning.", path, lineNumber, "todo_fixme")
		}
		if repositorySecretPattern.MatchString(line) && !strings.Contains(lower, "example") && !strings.Contains(lower, "placeholder") {
			add("high", "secret_hygiene", "Potential hardcoded secret", "A secret-like assignment was found. The response intentionally omits the raw value.", path, lineNumber, "secret_pattern")
		}
		if repositoryDangerousJavaScriptPattern(path, lower) {
			add("high", "security", "Dangerous dynamic JavaScript execution", "Dynamic code execution was detected. Review this before release because it can turn user input into executable code.", path, lineNumber, "dangerous_js_execution")
		}
		if repositoryInnerHTMLPattern(path, lower) {
			add("medium", "security", "Direct innerHTML assignment", "Direct innerHTML writes should be reviewed for sanitization before release.", path, lineNumber, "direct_inner_html")
		}
		if repositoryProductionPanicPattern(path, lower) {
			add("medium", "bug_risk", "Production panic path", "A panic call was found outside a Go test file. Confirm this cannot crash a user-facing workflow.", path, lineNumber, "production_panic")
		}
	}
}

func repositoryDangerousJavaScriptPattern(path, lowerLine string) bool {
	extension := strings.ToLower(filepath.Ext(path))
	if extension != ".js" && extension != ".jsx" && extension != ".ts" && extension != ".tsx" && extension != ".vue" {
		return false
	}
	return strings.Contains(lowerLine, "eval(") || strings.Contains(lowerLine, "new function(")
}

func repositoryInnerHTMLPattern(path, lowerLine string) bool {
	extension := strings.ToLower(filepath.Ext(path))
	if extension != ".js" && extension != ".jsx" && extension != ".ts" && extension != ".tsx" && extension != ".vue" && extension != ".html" {
		return false
	}
	return strings.Contains(lowerLine, ".innerhtml") && strings.Contains(lowerLine, "=")
}

func repositoryProductionPanicPattern(path, lowerLine string) bool {
	if strings.ToLower(filepath.Ext(path)) != ".go" || strings.HasSuffix(strings.ToLower(path), "_test.go") {
		return false
	}
	return strings.Contains(lowerLine, "panic(")
}

func repositoryLanguageRows(rows map[string]*RepositoryLanguage) []RepositoryLanguage {
	languages := make([]RepositoryLanguage, 0, len(rows))
	for _, row := range rows {
		languages = append(languages, *row)
	}
	sort.Slice(languages, func(i, j int) bool {
		if languages[i].FileCount == languages[j].FileCount {
			return languages[i].Language < languages[j].Language
		}
		return languages[i].FileCount > languages[j].FileCount
	})
	return languages
}
