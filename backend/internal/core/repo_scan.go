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
	maxRepositoryScanFiles    = 400
	maxRepositoryScanBytes    = 256 * 1024
	maxRepositoryScanFindings = 80
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
			if !dependency.HasLockfile && dependency.Ecosystem == "npm" {
				addFinding("medium", "dependency", "Missing npm lockfile", "package.json exists without a package-lock.json, pnpm-lock.yaml, yarn.lock, or bun.lockb in the same folder.", relativePath, 0, "lockfile_missing")
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
	response.Summary = fmt.Sprintf("Scanned %d text files across %d repository files.", response.Stats.ScannedFiles, response.Stats.FileCount)
	return response
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
	default:
		return RepositoryDependencyFile{}
	}
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
