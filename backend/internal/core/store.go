package core

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

var slugClean = regexp.MustCompile(`[^a-z0-9-]+`)

const defaultGeminiReviewModel = "gemini-2.5-flash"
const defaultLLMProvider = "gemini"

type llmProviderDefinition struct {
	ID     string
	Label  string
	Models []string
}

var llmProviderDefinitions = []llmProviderDefinition{
	{
		ID:    "gemini",
		Label: "Google Gemini",
		Models: []string{
			"gemini-2.5-flash",
			"gemini-2.5-pro",
			"gemini-2.5-flash-lite",
			"gemini-2.0-flash",
			"gemini-2.0-flash-lite",
		},
	},
	{
		ID:    "openai",
		Label: "OpenAI",
		Models: []string{
			"gpt-4.1",
			"gpt-4.1-mini",
			"gpt-4o",
			"gpt-4o-mini",
			"o3-mini",
		},
	},
	{
		ID:    "anthropic",
		Label: "Anthropic Claude",
		Models: []string{
			"claude-3-5-sonnet-latest",
			"claude-3-5-haiku-latest",
			"claude-3-opus-latest",
		},
	},
	{
		ID:    "groq",
		Label: "Groq",
		Models: []string{
			"llama-3.3-70b-versatile",
			"llama-3.1-8b-instant",
			"mixtral-8x7b-32768",
		},
	},
	{
		ID:    "openrouter",
		Label: "OpenRouter",
		Models: []string{
			"openai/gpt-4o-mini",
			"anthropic/claude-3.5-sonnet",
			"google/gemini-2.0-flash-001",
			"meta-llama/llama-3.1-70b-instruct",
		},
	},
	{
		ID:    "deepseek",
		Label: "DeepSeek",
		Models: []string{
			"deepseek-chat",
			"deepseek-reasoner",
		},
	},
	{
		ID:    "mistral",
		Label: "Mistral AI",
		Models: []string{
			"mistral-large-latest",
			"mistral-small-latest",
			"codestral-latest",
		},
	},
}

type Store struct {
	mu       sync.RWMutex
	cfg      Config
	payments *PaymentManager
	repos    RepoFactory
	emailer  *EmailSender
	storage  statePersistence

	nextID            int
	projects          map[string]*Project
	tasks             map[string]*Task
	users             map[string]*User
	wallets           map[string]*Wallet
	sessions          map[string]*Session
	notifications     map[string]*Notification
	attachments       map[string]*Attachment
	sslReviews        map[string]*SSLReviewStatus
	geminiAPIKeys     map[string]*GeminiAPIKey
	geminiWebhookLogs map[string]*GeminiWebhookLog
	adminSettings     AdminSettings
	ledger            []LedgerEntry
}

type persistedState struct {
	NextID            int                 `json:"next_id"`
	Projects          []*Project          `json:"projects"`
	Tasks             []*Task             `json:"tasks"`
	Users             []*User             `json:"users"`
	Wallets           []*Wallet           `json:"wallets"`
	Sessions          []*Session          `json:"sessions"`
	Notifications     []*Notification     `json:"notifications"`
	Attachments       []*Attachment       `json:"attachments"`
	SSLReviews        []*SSLReviewStatus  `json:"ssl_reviews"`
	GeminiAPIKeys     []*GeminiAPIKey     `json:"gemini_api_keys"`
	GeminiWebhookLogs []*GeminiWebhookLog `json:"gemini_webhook_logs"`
	AdminSettings     *AdminSettings      `json:"admin_settings,omitempty"`
	Ledger            []LedgerEntry       `json:"ledger"`
}

type statePersistence interface {
	Load(ctx context.Context) (persistedState, bool, error)
	Save(ctx context.Context, state persistedState) error
	Close() error
}

func NewStore(cfg Config, payments *PaymentManager, repos RepoFactory, emailer *EmailSender) (*Store, error) {
	store := &Store{
		cfg:               cfg,
		payments:          payments,
		repos:             repos,
		emailer:           emailer,
		nextID:            1,
		projects:          map[string]*Project{},
		tasks:             map[string]*Task{},
		users:             map[string]*User{},
		wallets:           map[string]*Wallet{},
		sessions:          map[string]*Session{},
		notifications:     map[string]*Notification{},
		attachments:       map[string]*Attachment{},
		sslReviews:        map[string]*SSLReviewStatus{},
		geminiAPIKeys:     map[string]*GeminiAPIKey{},
		geminiWebhookLogs: map[string]*GeminiWebhookLog{},
		adminSettings:     defaultAdminSettings(cfg),
		ledger:            []LedgerEntry{},
	}
	if strings.TrimSpace(cfg.DatabaseURL) != "" {
		storage, err := newPostgresPersistence(context.Background(), cfg)
		if err != nil {
			return nil, err
		}
		store.storage = storage
	}
	if err := store.load(); err != nil {
		_ = store.Close()
		return nil, err
	}
	if err := store.ensureAdmin(); err != nil {
		_ = store.Close()
		return nil, err
	}
	if err := store.SeedGeminiAPIKeysFromConfig(); err != nil {
		_ = store.Close()
		return nil, err
	}
	return store, nil
}

func (s *Store) Close() error {
	if s.storage == nil {
		return nil
	}
	return s.storage.Close()
}

func (s *Store) AdminSettings() AdminSettingsResponse {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return adminSettingsResponse(s.adminSettings)
}

func (s *Store) UpdateAdminSettings(req UpdateAdminSettingsRequest) (AdminSettingsResponse, error) {
	provider := strings.TrimSpace(req.LLMProvider)
	modelValue := strings.TrimSpace(req.LLMModel)
	if provider == "" && modelValue == "" && strings.TrimSpace(req.GeminiReviewModel) != "" {
		provider = "gemini"
		modelValue = req.GeminiReviewModel
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if provider == "" {
		provider = s.adminSettings.LLMProvider
	}
	provider = normalizedLLMProviderOrDefault(provider)
	if modelValue == "" {
		if provider == s.adminSettings.LLMProvider {
			modelValue = s.adminSettings.LLMModel
		} else {
			modelValue = normalizedLLMModelOrDefault(provider, "")
		}
	}
	model, err := normalizeLLMModel(provider, modelValue)
	if err != nil {
		return AdminSettingsResponse{}, err
	}

	s.adminSettings.LLMProvider = provider
	s.adminSettings.LLMModel = model
	if provider == "gemini" {
		s.adminSettings.GeminiReviewModel = model
	}
	s.adminSettings.UpdatedAt = time.Now().UTC()
	if err := s.saveLocked(); err != nil {
		return AdminSettingsResponse{}, err
	}
	return adminSettingsResponse(s.adminSettings), nil
}

func (s *Store) GeminiReviewModel() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return normalizedGeminiReviewModelOrDefault(s.adminSettings.GeminiReviewModel)
}

func (s *Store) LLMReviewProviderModel() (string, string) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	provider := normalizedLLMProviderOrDefault(s.adminSettings.LLMProvider)
	model := normalizedLLMModelOrDefault(provider, s.adminSettings.LLMModel)
	return provider, model
}

func (s *Store) Register(req RegisterRequest) (*AuthResponse, error) {
	email, err := normalizeEmail(req.Email)
	if err != nil {
		return nil, err
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, errors.New("name is required")
	}
	salt, hash, err := hashPassword(req.Password)
	if err != nil {
		return nil, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.userByEmailLocked(email) != nil {
		return nil, errors.New("email is already registered")
	}
	now := time.Now().UTC()
	role := RoleClient
	if s.cfg.AdminAutoPromote && !s.hasAdminLocked() && len(s.users) == 0 {
		role = RoleAdmin
	}
	user := &User{
		ID:           s.newID("usr"),
		Name:         name,
		CompanyName:  strings.TrimSpace(req.CompanyName),
		Email:        email,
		Role:         role,
		PasswordSalt: salt,
		PasswordHash: hash,
		CreatedAt:    now,
		LastLoginAt:  &now,
	}
	if _, err := s.ensureWalletForUserLocked(user, "", ""); err != nil {
		return nil, err
	}
	token, err := newToken()
	if err != nil {
		return nil, err
	}
	s.users[user.ID] = user
	s.sessions[token] = &Session{
		Token:     token,
		UserID:    user.ID,
		CreatedAt: now,
		ExpiresAt: now.Add(30 * 24 * time.Hour),
	}
	s.addNotificationLocked(user.ID, "", "email", "Welcome to MergeOS", "Your client workspace is ready. Submit a funded website project whenever you are ready.", "logged:welcome")
	if err := s.saveLocked(); err != nil {
		return nil, err
	}
	return &AuthResponse{Token: token, User: publicUser(user)}, nil
}

func (s *Store) Login(req LoginRequest) (*AuthResponse, error) {
	email, err := normalizeEmail(req.Email)
	if err != nil {
		return nil, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	user := s.userByEmailLocked(email)
	if user == nil || !verifyPassword(req.Password, user.PasswordSalt, user.PasswordHash) {
		return nil, errors.New("invalid email or password")
	}
	now := time.Now().UTC()
	token, err := newToken()
	if err != nil {
		return nil, err
	}
	user.LastLoginAt = &now
	if _, err := s.ensureWalletForUserLocked(user, "", ""); err != nil {
		return nil, err
	}
	s.sessions[token] = &Session{
		Token:     token,
		UserID:    user.ID,
		CreatedAt: now,
		ExpiresAt: now.Add(30 * 24 * time.Hour),
	}
	if err := s.saveLocked(); err != nil {
		return nil, err
	}
	return &AuthResponse{Token: token, User: publicUser(user)}, nil
}

func (s *Store) LoginOrRegisterOAuth(email, name, provider string) (*AuthResponse, error) {
	email, err := normalizeEmail(email)
	if err != nil {
		return nil, err
	}
	name = strings.TrimSpace(name)
	if name == "" {
		name = strings.Split(email, "@")[0]
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	user := s.userByEmailLocked(email)
	now := time.Now().UTC()

	if user == nil {
		role := RoleClient
		if s.cfg.AdminAutoPromote && !s.hasAdminLocked() && len(s.users) == 0 {
			role = RoleAdmin
		}

		saltBytes := make([]byte, 16)
		if _, err := rand.Read(saltBytes); err != nil {
			return nil, err
		}
		salt := hex.EncodeToString(saltBytes)

		randPassBytes := make([]byte, 32)
		if _, err := rand.Read(randPassBytes); err != nil {
			return nil, err
		}
		hash := hex.EncodeToString(randPassBytes)

		user = &User{
			ID:           s.newID("usr"),
			Name:         name,
			CompanyName:  "",
			Email:        email,
			Role:         role,
			PasswordSalt: salt,
			PasswordHash: hash,
			CreatedAt:    now,
			LastLoginAt:  &now,
		}
		s.users[user.ID] = user
		s.addNotificationLocked(user.ID, "", "email", "Welcome to MergeOS via OAuth", "Your client workspace is ready. You signed up using "+provider+".", "logged:welcome")
	} else {
		user.LastLoginAt = &now
	}

	token, err := newToken()
	if err != nil {
		return nil, err
	}
	s.sessions[token] = &Session{
		Token:     token,
		UserID:    user.ID,
		CreatedAt: now,
		ExpiresAt: now.Add(30 * 24 * time.Hour),
	}

	if err := s.saveLocked(); err != nil {
		return nil, err
	}
	return &AuthResponse{Token: token, User: publicUser(user)}, nil
}

func (s *Store) UserByToken(token string) (*User, bool) {
	token = strings.TrimSpace(strings.TrimPrefix(token, "Bearer "))
	if token == "" {
		return nil, false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	session, ok := s.sessions[token]
	if !ok || time.Now().UTC().After(session.ExpiresAt) {
		delete(s.sessions, token)
		_ = s.saveLocked()
		return nil, false
	}
	user, ok := s.users[session.UserID]
	if !ok {
		return nil, false
	}
	copyUser := *user
	return &copyUser, true
}

func (s *Store) Logout(token string) {
	token = strings.TrimSpace(strings.TrimPrefix(token, "Bearer "))
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, token)
	_ = s.saveLocked()
}

func (s *Store) ensureAdmin() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	changed := false
	for _, user := range s.users {
		role := normalizeRole(user.Role)
		if user.Role != role {
			user.Role = role
			changed = true
		}
		previousWallet := user.WalletAddress
		previousWalletCount := len(s.wallets)
		if _, err := s.ensureWalletForUserLocked(user, "", ""); err != nil {
			return err
		}
		if previousWallet != user.WalletAddress || previousWalletCount != len(s.wallets) {
			changed = true
		}
	}

	if strings.TrimSpace(s.cfg.AdminEmail) != "" {
		adminChanged, err := s.ensureConfiguredAdminLocked()
		if err != nil {
			return err
		}
		changed = changed || adminChanged
	}

	if !s.hasAdminLocked() && s.cfg.AdminAutoPromote {
		var first *User
		for _, user := range s.users {
			if first == nil || user.CreatedAt.Before(first.CreatedAt) || (user.CreatedAt.Equal(first.CreatedAt) && user.ID < first.ID) {
				first = user
			}
		}
		if first != nil {
			first.Role = RoleAdmin
			changed = true
		}
	}

	if changed {
		return s.saveLocked()
	}
	return nil
}

func (s *Store) ensureConfiguredAdminLocked() (bool, error) {
	email, err := normalizeEmail(s.cfg.AdminEmail)
	if err != nil {
		return false, fmt.Errorf("ADMIN_EMAIL is invalid: %w", err)
	}
	name := strings.TrimSpace(s.cfg.AdminName)
	if name == "" {
		name = "MergeOS Admin"
	}
	companyName := strings.TrimSpace(s.cfg.AdminCompanyName)
	if companyName == "" {
		companyName = "MergeOS"
	}

	if user := s.userByEmailLocked(email); user != nil {
		changed := false
		if user.Role != RoleAdmin {
			user.Role = RoleAdmin
			changed = true
		}
		if user.Name == "" {
			user.Name = name
			changed = true
		}
		if user.CompanyName == "" {
			user.CompanyName = companyName
			changed = true
		}
		previousWallet := user.WalletAddress
		previousWalletCount := len(s.wallets)
		if _, err := s.ensureWalletForUserLocked(user, "", ""); err != nil {
			return false, err
		}
		if previousWallet != user.WalletAddress || previousWalletCount != len(s.wallets) {
			changed = true
		}
		password := strings.TrimSpace(s.cfg.AdminPassword)
		if password != "" && !verifyPassword(password, user.PasswordSalt, user.PasswordHash) {
			salt, hash, err := hashPassword(password)
			if err != nil {
				return false, err
			}
			user.PasswordSalt = salt
			user.PasswordHash = hash
			changed = true
		}
		return changed, nil
	}

	if strings.TrimSpace(s.cfg.AdminPassword) == "" {
		return false, errors.New("ADMIN_PASSWORD is required when ADMIN_EMAIL does not match an existing user")
	}
	salt, hash, err := hashPassword(s.cfg.AdminPassword)
	if err != nil {
		return false, err
	}
	now := time.Now().UTC()
	admin := &User{
		ID:           s.newID("usr"),
		Name:         name,
		CompanyName:  companyName,
		Email:        email,
		Role:         RoleAdmin,
		PasswordSalt: salt,
		PasswordHash: hash,
		CreatedAt:    now,
	}
	if _, err := s.ensureWalletForUserLocked(admin, "", ""); err != nil {
		return false, err
	}
	s.users[admin.ID] = admin
	s.addNotificationLocked(admin.ID, "", "email", "MergeOS admin enabled", "Your admin workspace can manage customers, funded projects, task payouts, ledger entries and delivery notifications.", "logged:admin-bootstrap")
	return true, nil
}

func (s *Store) hasAdminLocked() bool {
	for _, user := range s.users {
		if normalizeRole(user.Role) == RoleAdmin {
			return true
		}
	}
	return false
}

func (s *Store) CreateProject(ctx context.Context, userID string, req CreateProjectRequest) (*Project, error) {
	if strings.TrimSpace(userID) == "" {
		return nil, errors.New("login is required")
	}
	if strings.TrimSpace(req.Title) == "" {
		return nil, errors.New("title is required")
	}
	if req.BudgetCents < 10000 {
		return nil, errors.New("funding payment must be at least 100 USD")
	}
	if req.PaymentMethod != PaymentPayPal && req.PaymentMethod != PaymentCrypto {
		return nil, errors.New("payment method must be paypal or crypto")
	}
	tokenSymbol := normalizedTokenSymbol(s.cfg.TokenSymbol)
	sourceRepoURL := strings.TrimSpace(req.SourceRepoURL)
	var importedIssues []*ImportedRepoIssue
	if sourceRepoURL != "" {
		imported, err := ImportRepoIssues(ctx, s.cfg, ImportRepoIssuesRequest{RepoURL: sourceRepoURL})
		if err != nil {
			return nil, err
		}
		if len(imported.Issues) == 0 {
			return nil, errors.New("repo has no open issues to fund")
		}
		importedIssues = imported.Issues
		sourceRepoURL = imported.RepoURL
	}

	verification, err := s.payments.Verify(ctx, req)
	if err != nil {
		return nil, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	user, ok := s.users[userID]
	if !ok {
		return nil, errors.New("user not found")
	}

	clientName := strings.TrimSpace(req.ClientName)
	if clientName == "" {
		clientName = user.Name
	}
	companyName := strings.TrimSpace(req.CompanyName)
	if companyName == "" {
		companyName = user.CompanyName
	}
	clientEmail := strings.TrimSpace(req.ClientEmail)
	if clientEmail == "" {
		clientEmail = user.Email
	}
	clientEmail, err = normalizeEmail(clientEmail)
	if err != nil {
		return nil, err
	}

	projectID := s.newID("prj")
	fee := req.BudgetCents * s.cfg.PlatformFeeBps / 10000
	workPool := req.BudgetCents - fee
	now := time.Now().UTC()
	project := &Project{
		ID:               projectID,
		ClientUserID:     user.ID,
		Title:            strings.TrimSpace(req.Title),
		ClientName:       clientName,
		CompanyName:      companyName,
		ClientEmail:      clientEmail,
		Phone:            strings.TrimSpace(req.Phone),
		SiteType:         strings.TrimSpace(req.SiteType),
		PackageTier:      strings.TrimSpace(req.PackageTier),
		Timeline:         strings.TrimSpace(req.Timeline),
		Brief:            strings.TrimSpace(req.Brief),
		PaymentMethod:    req.PaymentMethod,
		PaymentStatus:    "verified",
		PaymentProvider:  verification.Provider,
		PaymentReference: verification.Reference,
		RepoVisibility:   "private-child-bounty-repo",
		BudgetCents:      req.BudgetCents,
		FeeCents:         fee,
		WorkPoolCents:    workPool,
		Status:           ProjectFunded,
		CreatedAt:        now,
	}
	if sourceRepoURL != "" && !strings.Contains(project.Brief, sourceRepoURL) {
		project.Brief = "Source repository: " + sourceRepoURL + "\n\n" + project.Brief
	}
	for _, attachmentID := range req.AttachmentIDs {
		attachment, ok := s.attachments[attachmentID]
		if !ok {
			return nil, fmt.Errorf("attachment %s not found", attachmentID)
		}
		if attachment.UserID != user.ID {
			return nil, fmt.Errorf("attachment %s does not belong to this user", attachmentID)
		}
		if attachment.ProjectID != "" {
			return nil, fmt.Errorf("attachment %s is already attached to a project", attachmentID)
		}
		attachment.ProjectID = project.ID
		project.Attachments = append(project.Attachments, cloneAttachment(attachment))
	}
	if len(importedIssues) > 0 {
		project.Tasks = s.tasksFromImportedIssues(project, importedIssues)
	} else {
		project.Tasks = s.splitProjectTasks(project)
	}

	repo, err := s.repos.CreateProjectRepo(ctx, project, project.Tasks)
	if err != nil {
		return nil, err
	}
	project.BountyRepoName = repo.Name
	project.RepoProvider = repo.Provider
	project.RepoURL = repo.URL
	project.RepoLocalPath = repo.LocalPath
	for _, task := range project.Tasks {
		if issue, ok := repo.Issues[task.ID]; ok {
			task.IssueNumber = issue.Number
			if strings.TrimSpace(task.IssueURL) == "" {
				task.IssueURL = issue.URL
			}
		}
	}

	s.projects[projectID] = project
	clientProjectAccount := "client:" + user.ID + ":project:" + projectID
	s.addLedger("payment_verified", "payment:"+verification.Provider, clientProjectAccount, req.BudgetCents, verification.Reference)
	s.addLedger("token_mint", "issuer:mergeos", clientProjectAccount, req.BudgetCents, "mint:"+projectID)
	s.addLedger("platform_fee", "client:"+projectID, "treasury:mergeos", fee, "fee:"+projectID)
	s.addLedger("project_reserve", "client:"+projectID, "reserve:project:"+projectID, workPool, "repo:"+project.BountyRepoName)

	for _, task := range project.Tasks {
		s.tasks[task.ID] = task
		reference := fmt.Sprintf("%s/issues/%d", project.BountyRepoName, task.IssueNumber)
		if task.IssueURL != "" {
			reference = task.IssueURL
		}
		s.addLedger("task_reserve", "reserve:project:"+projectID, taskReserveAccount(), task.RewardCents, reference)
	}
	subject := "MergeOS project funded: " + project.Title
	body := fmt.Sprintf("Hi %s,\n\nYour project %q is funded. MergeOS created bounty repo %s and split it into %d payable tasks.\n\nBudget: %s %s\nWork pool: %s %s\nAttachments: %d\n\nWe will notify you as tasks are accepted.", project.ClientName, project.Title, project.BountyRepoName, len(project.Tasks), formatTokenAmount(project.BudgetCents), tokenSymbol, formatTokenAmount(project.WorkPoolCents), tokenSymbol, len(project.Attachments))
	status := s.emailer.Send(project.ClientEmail, subject, body)
	s.addNotificationLocked(user.ID, project.ID, "email", subject, body, status)
	if err := s.saveLocked(); err != nil {
		return nil, err
	}

	return cloneProject(project), nil
}

func (s *Store) ListProjects(userID string) []*Project {
	s.mu.RLock()
	defer s.mu.RUnlock()

	projects := make([]*Project, 0, len(s.projects))
	for _, project := range s.projects {
		if userID != "" && project.ClientUserID != userID {
			continue
		}
		projects = append(projects, cloneProject(project))
	}
	return projects
}

func (s *Store) ListTasks(userID string) []*Task {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tasks := make([]*Task, 0, len(s.tasks))
	for _, task := range s.tasks {
		if userID != "" {
			project, ok := s.projects[task.ProjectID]
			if !ok || project.ClientUserID != userID {
				continue
			}
		}
		copyTask := *task
		tasks = append(tasks, &copyTask)
	}
	sortTasks(tasks)
	return tasks
}

func (s *Store) SyncProjectImportedIssues(projectID string, issues []*ImportedRepoIssue) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	project, ok := s.projects[strings.TrimSpace(projectID)]
	if !ok {
		return errors.New("project not found")
	}

	existing := map[int]*Task{}
	for _, task := range s.tasks {
		if task.ProjectID == project.ID && task.IssueNumber > 0 {
			existing[task.IssueNumber] = task
		}
	}

	changed := false
	now := time.Now().UTC()
	for _, issue := range issues {
		if issue == nil || issue.Number <= 0 {
			continue
		}
		state := normalizeIssueState(issue.State)
		if task, ok := existing[issue.Number]; ok {
			taskChanged := false
			if task.IssueState != state {
				task.IssueState = state
				taskChanged = true
			}
			if strings.TrimSpace(task.IssueURL) == "" && strings.TrimSpace(issue.URL) != "" {
				task.IssueURL = strings.TrimSpace(issue.URL)
				taskChanged = true
			}
			if taskChanged {
				s.syncProjectTaskSnapshotLocked(project, task)
				changed = true
			}
			continue
		}

		task := &Task{
			ID:                 s.newID("tsk"),
			ProjectID:          project.ID,
			IssueNumber:        issue.Number,
			Title:              fmt.Sprintf("Fix #%d: %s", issue.Number, strings.TrimSpace(issue.Title)),
			Acceptance:         importedIssueAcceptance(issue),
			RewardCents:        importedIssueReward(issue),
			RequiredWorkerKind: issue.RequiredWorkerKind,
			SuggestedAgentType: strings.TrimSpace(issue.SuggestedAgentType),
			Status:             TaskOpen,
			IssueURL:           strings.TrimSpace(issue.URL),
			IssueState:         state,
			CreatedAt:          now,
		}
		s.tasks[task.ID] = task
		existing[issue.Number] = task
		s.syncProjectTaskSnapshotLocked(project, task)
		changed = true
	}

	if !changed {
		return nil
	}
	sortTasks(project.Tasks)
	return s.saveLocked()
}

func (s *Store) TaskWithProject(taskID string) (*Task, *Project, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	task, ok := s.tasks[strings.TrimSpace(taskID)]
	if !ok {
		return nil, nil, false
	}
	project, ok := s.projects[task.ProjectID]
	if !ok {
		return nil, nil, false
	}
	taskCopy := *task
	return &taskCopy, cloneProject(project), true
}

func (s *Store) ListNotifications(userID string) []*Notification {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows := make([]*Notification, 0, len(s.notifications))
	for _, note := range s.notifications {
		if userID != "" && note.UserID != userID {
			continue
		}
		copyNote := *note
		rows = append(rows, &copyNote)
	}
	return rows
}

func (s *Store) ListLedger() []LedgerEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entries := make([]LedgerEntry, len(s.ledger))
	copy(entries, s.ledger)
	return entries
}

func (s *Store) ListPublicLedger() []LedgerEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()

	projectIDs := map[string]bool{}
	taskProjectIDs := map[string]string{}
	for _, project := range s.projects {
		projectIDs[project.ID] = true
		for _, task := range project.Tasks {
			taskProjectIDs[task.ID] = project.ID
		}
	}

	entries := make([]LedgerEntry, 0, len(s.ledger))
	for _, entry := range s.ledger {
		projectID, taskID := publicLedgerScope(entry, projectIDs, taskProjectIDs)
		publicEntry := entry
		publicEntry.FromAccount = publicLedgerAccount(entry.FromAccount, projectID, taskID)
		publicEntry.ToAccount = publicLedgerAccount(entry.ToAccount, projectID, taskID)
		publicEntry.Reference = publicLedgerReference(projectID, taskID, entry.Sequence)
		entries = append(entries, publicEntry)
	}
	return entries
}

func (s *Store) ListLedgerForUser(userID string) []LedgerEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()

	projectIDs := map[string]bool{}
	taskIDs := map[string]bool{}
	for _, project := range s.projects {
		if project.ClientUserID != userID {
			continue
		}
		projectIDs[project.ID] = true
		for _, task := range project.Tasks {
			taskIDs[task.ID] = true
		}
	}

	entries := make([]LedgerEntry, 0, len(s.ledger))
	for _, entry := range s.ledger {
		if ledgerEntryMatches(entry, projectIDs, taskIDs) {
			entries = append(entries, entry)
		}
	}
	return entries
}

func (s *Store) Marketplace() MarketplaceResponse {
	s.mu.RLock()
	defer s.mu.RUnlock()

	response := MarketplaceResponse{
		Stats: MarketplaceStats{
			TokenSymbol:      s.cfg.TokenSymbol,
			LedgerEntryCount: len(s.ledger),
			ProjectCount:     len(s.projects),
			UpdatedAt:        marketplaceLatestLedgerTime(s.ledger),
		},
		Projects:     []*MarketplaceProject{},
		Contributors: []*MarketplaceContributor{},
		Agents:       []*MarketplaceAgent{},
	}

	for _, project := range s.projects {
		row := &MarketplaceProject{
			ID:                project.ID,
			Title:             project.Title,
			Brief:             project.Brief,
			SiteType:          project.SiteType,
			PackageTier:       project.PackageTier,
			Timeline:          project.Timeline,
			Status:            project.Status,
			ClientDisplayName: marketplaceClientDisplayName(project),
			BountyRepoName:    project.BountyRepoName,
			RepoProvider:      project.RepoProvider,
			RepoURL:           marketplacePublicRepoURL(project.RepoURL),
			BudgetCents:       project.BudgetCents,
			WorkPoolCents:     project.WorkPoolCents,
			Tags:              marketplaceProjectTags(project),
			CreatedAt:         project.CreatedAt,
		}
		for _, task := range project.Tasks {
			row.TaskCount++
			switch task.Status {
			case TaskAccepted:
				row.AcceptedTaskCount++
			default:
				row.OpenTaskCount++
			}
		}
		response.Stats.OpenTaskCount += row.OpenTaskCount
		response.Stats.AcceptedTaskCount += row.AcceptedTaskCount
		response.Stats.TotalBudgetCents += project.BudgetCents
		response.Stats.WorkPoolCents += project.WorkPoolCents
		if response.Stats.UpdatedAt == nil || project.CreatedAt.After(*response.Stats.UpdatedAt) {
			updatedAt := project.CreatedAt
			response.Stats.UpdatedAt = &updatedAt
		}
		response.Projects = append(response.Projects, row)
	}

	sort.Slice(response.Projects, func(i, j int) bool {
		return response.Projects[i].CreatedAt.After(response.Projects[j].CreatedAt)
	})

	contributors := map[string]*MarketplaceContributor{}
	agents := map[string]*MarketplaceAgent{}
	for _, task := range s.tasks {
		if task.SuggestedAgentType != "" {
			agent := agents[task.SuggestedAgentType]
			if agent == nil {
				agent = &MarketplaceAgent{
					Type:       task.SuggestedAgentType,
					Title:      marketplaceTitle(task.SuggestedAgentType),
					WorkerKind: task.RequiredWorkerKind,
				}
				agents[task.SuggestedAgentType] = agent
			}
			agent.TaskCount++
			if task.Status != TaskAccepted {
				agent.OpenTaskCount++
				agent.BudgetCents += task.RewardCents
			}
		}

		if task.Status != TaskAccepted || strings.TrimSpace(task.WorkerID) == "" {
			continue
		}
		key := task.WorkerID
		if task.AgentType != "" {
			key += ":" + task.AgentType
		}
		contributor := contributors[key]
		if contributor == nil {
			contributor = &MarketplaceContributor{
				WorkerID:  task.WorkerID,
				Name:      marketplaceWorkerName(task.WorkerID, task.AgentType),
				Kind:      task.WorkerKind,
				AgentType: task.AgentType,
			}
			contributors[key] = contributor
		}
		contributor.TaskCount++
		contributor.EarnedCents += task.RewardCents
		if task.AcceptedAt != nil && task.AcceptedAt.After(contributor.LastPaidAt) {
			contributor.LastPaidAt = *task.AcceptedAt
		}
	}

	for _, contributor := range contributors {
		response.Contributors = append(response.Contributors, contributor)
	}
	sort.Slice(response.Contributors, func(i, j int) bool {
		if response.Contributors[i].EarnedCents == response.Contributors[j].EarnedCents {
			return response.Contributors[i].LastPaidAt.After(response.Contributors[j].LastPaidAt)
		}
		return response.Contributors[i].EarnedCents > response.Contributors[j].EarnedCents
	})

	for _, agent := range agents {
		response.Agents = append(response.Agents, agent)
	}
	sort.Slice(response.Agents, func(i, j int) bool {
		if response.Agents[i].OpenTaskCount == response.Agents[j].OpenTaskCount {
			return response.Agents[i].BudgetCents > response.Agents[j].BudgetCents
		}
		return response.Agents[i].OpenTaskCount > response.Agents[j].OpenTaskCount
	})

	return response
}

func (s *Store) ListUsers() []AdminUser {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows := make([]AdminUser, 0, len(s.users))
	for _, user := range s.users {
		rows = append(rows, s.adminUserRowLocked(user))
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Role != rows[j].Role {
			return rows[i].Role == RoleAdmin
		}
		return rows[i].CreatedAt.After(rows[j].CreatedAt)
	})
	return rows
}

func (s *Store) UpdateUser(userID string, req AdminUpdateUserRequest) (AdminUser, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return AdminUser{}, errors.New("user id is required")
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return AdminUser{}, errors.New("name is required")
	}
	email, err := normalizeEmail(req.Email)
	if err != nil {
		return AdminUser{}, err
	}

	var passwordSalt string
	var passwordHash string
	if strings.TrimSpace(req.Password) != "" {
		passwordSalt, passwordHash, err = hashPassword(req.Password)
		if err != nil {
			return AdminUser{}, err
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	user, ok := s.users[userID]
	if !ok {
		return AdminUser{}, errors.New("user not found")
	}
	for _, other := range s.users {
		if other.ID != userID && strings.EqualFold(other.Email, email) {
			return AdminUser{}, errors.New("email is already registered")
		}
	}

	role := normalizeRole(user.Role)
	if strings.TrimSpace(string(req.Role)) != "" {
		role = normalizeRole(req.Role)
	}
	if normalizeRole(user.Role) == RoleAdmin && role != RoleAdmin && !s.hasOtherAdminLocked(userID) {
		return AdminUser{}, errors.New("at least one admin user is required")
	}

	user.Name = name
	user.CompanyName = strings.TrimSpace(req.CompanyName)
	user.Email = email
	user.Role = role
	if passwordHash != "" {
		user.PasswordSalt = passwordSalt
		user.PasswordHash = passwordHash
	}
	row := s.adminUserRowLocked(user)
	if err := s.saveLocked(); err != nil {
		return AdminUser{}, err
	}
	return row, nil
}

func (s *Store) adminUserRowLocked(user *User) AdminUser {
	row := AdminUser{PublicUser: publicUser(user)}
	for _, project := range s.projects {
		if project.ClientUserID != user.ID {
			continue
		}
		row.ProjectCount++
		row.TotalBudgetCents += project.BudgetCents
		if row.LastProjectAt == nil || project.CreatedAt.After(*row.LastProjectAt) {
			createdAt := project.CreatedAt
			row.LastProjectAt = &createdAt
		}
	}
	return row
}

func (s *Store) hasOtherAdminLocked(userID string) bool {
	for id, user := range s.users {
		if id != userID && normalizeRole(user.Role) == RoleAdmin {
			return true
		}
	}
	return false
}

func (s *Store) AdminSummary() AdminSummary {
	s.mu.RLock()
	defer s.mu.RUnlock()

	summary := AdminSummary{
		TokenSymbol:       s.cfg.TokenSymbol,
		PaymentMode:       paymentMode(s.cfg),
		RepoProvider:      repoProvider(s.cfg),
		PayPalReady:       s.cfg.PayPalReady(),
		CryptoReady:       s.cfg.CryptoReady(),
		GitHubReady:       s.cfg.GitHubReady(),
		SMTPReady:         s.cfg.SMTPReady(),
		DevPaymentEnabled: s.cfg.DevPaymentEnabled,
		BountyRoot:        s.cfg.BountyRoot,
		UploadRoot:        s.cfg.UploadRoot,
		SSLReviews:        s.sslReviewRowsLocked(),
		ProjectCount:      len(s.projects),
		WalletCount:       len(s.wallets),
		NotificationCount: len(s.notifications),
		AttachmentCount:   len(s.attachments),
	}
	for _, user := range s.users {
		summary.UserCount++
		if normalizeRole(user.Role) == RoleAdmin {
			summary.AdminCount++
		} else {
			summary.ClientCount++
		}
	}
	for _, project := range s.projects {
		summary.TotalBudgetCents += project.BudgetCents
		summary.WorkPoolCents += project.WorkPoolCents
		summary.PlatformFeeCents += project.FeeCents
	}
	for _, task := range s.tasks {
		if task.Status == TaskAccepted {
			summary.AcceptedTaskCount++
			summary.PaidTaskCents += task.RewardCents
			continue
		}
		summary.OpenTaskCount++
	}
	return summary
}

func (s *Store) CanAccessTask(userID string, role UserRole, taskID string) bool {
	if normalizeRole(role) == RoleAdmin {
		return true
	}
	s.mu.RLock()
	defer s.mu.RUnlock()

	task, ok := s.tasks[taskID]
	if !ok {
		return false
	}
	project, ok := s.projects[task.ProjectID]
	return ok && project.ClientUserID == userID
}

func (s *Store) AcceptTask(taskID string, req AcceptTaskRequest) (*Task, error) {
	return s.AcceptTaskWithReview(taskID, req, 0, "")
}

func (s *Store) AcceptTaskWithReview(taskID string, req AcceptTaskRequest, rewardCents int64, bountyType string) (*Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	task, ok := s.tasks[taskID]
	if !ok {
		return nil, errors.New("task not found")
	}
	if req.WorkerKind != WorkerHuman && req.WorkerKind != WorkerAgent && req.WorkerKind != WorkerHybrid {
		return nil, errors.New("worker kind must be human, agent, or hybrid")
	}
	if strings.TrimSpace(req.WorkerID) == "" {
		return nil, errors.New("worker id is required")
	}
	if task.RequiredWorkerKind != req.WorkerKind {
		return nil, fmt.Errorf("task requires %s work", task.RequiredWorkerKind)
	}
	if req.WorkerKind != WorkerHuman && strings.TrimSpace(req.AgentType) == "" {
		return nil, errors.New("agent type is required for agent or hybrid work")
	}
	if req.WorkerKind == WorkerHuman && strings.TrimSpace(req.AgentType) != "" {
		return nil, errors.New("agent type must be empty for human work")
	}

	workerID := normalizeWorkerID(req.WorkerID)
	payoutCents := task.RewardCents
	if rewardCents > 0 {
		payoutCents = rewardCents
		task.RewardCents = rewardCents
	}
	task.BountyType = strings.TrimSpace(bountyType)
	now := time.Now().UTC()
	entry := s.addLedger("task_payment", taskReserveAccount(), s.payoutAccountForWorkerLocked(workerID), payoutCents, "task:"+task.ID)
	task.Status = TaskAccepted
	task.WorkerKind = req.WorkerKind
	task.WorkerID = workerID
	task.AgentType = strings.TrimSpace(req.AgentType)
	task.ProofHash = entry.EntryHash
	task.AcceptedAt = &now

	if project, ok := s.projects[task.ProjectID]; ok {
		for index, projectTask := range project.Tasks {
			if projectTask.ID == task.ID {
				taskCopy := *task
				project.Tasks[index] = &taskCopy
				break
			}
		}
		subject := "MergeOS task paid: " + task.Title
		body := fmt.Sprintf("Task #%d was accepted and paid %s %s to %s. Proof hash: %s", task.IssueNumber, formatTokenAmount(payoutCents), normalizedTokenSymbol(s.cfg.TokenSymbol), task.WorkerID, task.ProofHash)
		status := s.emailer.Send(project.ClientEmail, subject, body)
		s.addNotificationLocked(project.ClientUserID, project.ID, "email", subject, body, status)
	}
	if err := s.saveLocked(); err != nil {
		return nil, err
	}

	copyTask := *task
	return &copyTask, nil
}

func (s *Store) TaskPayoutAccount(taskID string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	reference := "task:" + strings.TrimSpace(taskID)
	for index := len(s.ledger) - 1; index >= 0; index-- {
		entry := s.ledger[index]
		if entry.Type == "task_payment" && entry.Reference == reference {
			return entry.ToAccount, true
		}
	}
	task, ok := s.tasks[taskID]
	if !ok || strings.TrimSpace(task.WorkerID) == "" {
		return "", false
	}
	return s.payoutAccountForWorkerLocked(task.WorkerID), true
}

func (s *Store) userByEmailLocked(email string) *User {
	for _, user := range s.users {
		if user.Email == email {
			return user
		}
	}
	return nil
}

func (s *Store) addNotificationLocked(userID, projectID, channel, subject, body, status string) {
	note := &Notification{
		ID:        s.newID("ntf"),
		UserID:    userID,
		ProjectID: projectID,
		Channel:   channel,
		Subject:   subject,
		Body:      body,
		Status:    status,
		CreatedAt: time.Now().UTC(),
	}
	s.notifications[note.ID] = note
}

func (s *Store) newID(prefix string) string {
	id := fmt.Sprintf("%s_%04d", prefix, s.nextID)
	s.nextID++
	return id
}

func (s *Store) splitProjectTasks(project *Project) []*Task {
	tokenSymbol := normalizedTokenSymbol(s.cfg.TokenSymbol)
	type spec struct {
		title      string
		acceptance string
		weight     int64
		kind       WorkerKind
		agent      string
	}
	specs := []spec{
		{"Client discovery and conversion map", "Business goals, audience, sitemap, section inventory and copy outline are approved by the client.", 10, WorkerHuman, ""},
		{"Brand system and responsive page kit", "Colors, type scale, spacing, forms, cards, headers and mobile states are ready for the site build.", 18, WorkerHybrid, "design-agent"},
		{"Elementor-style page builder canvas", "Landing page blocks, drag-ready sections, inspector controls and preview surface run in the customer portal.", 24, WorkerAgent, "frontend-agent"},
		{"Checkout, token and proof ledger", fmt.Sprintf("PayPal/crypto verification, %s mint, reserves, fees and proof ledger are testable through API.", tokenSymbol), 22, WorkerAgent, "go-ledger-agent"},
		{"QA, accessibility and customer preview", "The delivery includes responsive QA, a11y pass, empty/error states and customer preview notes.", 14, WorkerHuman, ""},
		{"Deployment pipeline and private repo handoff", "Child repo has README, issues, environment guidance, smoke check and deploy handoff notes.", 12, WorkerHybrid, "devops-agent"},
	}

	tasks := make([]*Task, 0, len(specs))
	allocated := int64(0)
	for i, item := range specs {
		reward := project.WorkPoolCents * item.weight / 100
		if i == len(specs)-1 {
			reward = project.WorkPoolCents - allocated
		}
		allocated += reward
		task := &Task{
			ID:                 s.newID("tsk"),
			ProjectID:          project.ID,
			IssueNumber:        i + 1,
			Title:              item.title,
			Acceptance:         item.acceptance,
			RewardCents:        reward,
			RequiredWorkerKind: item.kind,
			SuggestedAgentType: item.agent,
			Status:             TaskOpen,
			IssueState:         "open",
			CreatedAt:          time.Now().UTC(),
		}
		tasks = append(tasks, task)
	}
	return tasks
}

func (s *Store) tasksFromImportedIssues(project *Project, issues []*ImportedRepoIssue) []*Task {
	tasks := make([]*Task, 0, len(issues))
	totalWeight := int64(0)
	for _, issue := range issues {
		totalWeight += issueRewardWeight(issue)
	}
	if totalWeight <= 0 {
		totalWeight = int64(len(issues))
	}

	allocated := int64(0)
	for index, issue := range issues {
		weight := issueRewardWeight(issue)
		reward := project.WorkPoolCents * weight / totalWeight
		if index == len(issues)-1 {
			reward = project.WorkPoolCents - allocated
		}
		allocated += reward
		task := &Task{
			ID:                 s.newID("tsk"),
			ProjectID:          project.ID,
			IssueNumber:        issue.Number,
			Title:              fmt.Sprintf("Fix #%d: %s", issue.Number, strings.TrimSpace(issue.Title)),
			Acceptance:         importedIssueAcceptance(issue),
			RewardCents:        reward,
			RequiredWorkerKind: issue.RequiredWorkerKind,
			SuggestedAgentType: strings.TrimSpace(issue.SuggestedAgentType),
			Status:             TaskOpen,
			IssueURL:           strings.TrimSpace(issue.URL),
			IssueState:         normalizeIssueState(issue.State),
			CreatedAt:          time.Now().UTC(),
		}
		tasks = append(tasks, task)
	}
	return tasks
}

func issueRewardWeight(issue *ImportedRepoIssue) int64 {
	if issue == nil {
		return 1
	}
	if issue.EstimatedCents > 0 {
		return issue.EstimatedCents
	}
	if issue.Score > 0 {
		return int64(issue.Score)
	}
	return 1
}

func importedIssueReward(issue *ImportedRepoIssue) int64 {
	if issue != nil && issue.EstimatedCents > 0 {
		return issue.EstimatedCents
	}
	return 100
}

func normalizeIssueState(value string) string {
	state := strings.ToLower(strings.TrimSpace(value))
	if state == "closed" || state == "close" {
		return "closed"
	}
	return "open"
}

func sortTasks(tasks []*Task) {
	sort.SliceStable(tasks, func(i, j int) bool {
		left, right := tasks[i], tasks[j]
		if left == nil {
			return false
		}
		if right == nil {
			return true
		}
		if left.ProjectID != right.ProjectID {
			return left.ProjectID < right.ProjectID
		}
		if left.IssueNumber != right.IssueNumber {
			return left.IssueNumber < right.IssueNumber
		}
		if !left.CreatedAt.Equal(right.CreatedAt) {
			return left.CreatedAt.Before(right.CreatedAt)
		}
		return left.ID < right.ID
	})
}

func (s *Store) syncProjectTaskSnapshotLocked(project *Project, task *Task) {
	if project == nil || task == nil {
		return
	}
	taskCopy := *task
	for index, projectTask := range project.Tasks {
		if projectTask != nil && projectTask.ID == task.ID {
			project.Tasks[index] = &taskCopy
			return
		}
	}
	project.Tasks = append(project.Tasks, &taskCopy)
}

func importedIssueAcceptance(issue *ImportedRepoIssue) string {
	if issue == nil {
		return "Resolve the imported GitHub issue and provide verification notes."
	}
	parts := []string{
		fmt.Sprintf("Resolve GitHub issue #%d and include verification notes.", issue.Number),
	}
	if strings.TrimSpace(issue.URL) != "" {
		parts = append(parts, "Source issue: "+strings.TrimSpace(issue.URL)+".")
	}
	if issue.Complexity != "" {
		parts = append(parts, "Complexity: "+issue.Complexity+".")
	}
	if len(issue.Reasons) > 0 {
		parts = append(parts, "Scoring signals: "+strings.Join(issue.Reasons, ", ")+".")
	}
	parts = append(parts, "Acceptance requires passing checks, a clear fix summary, and evidence that the original issue can be closed.")
	return strings.Join(parts, " ")
}

func (s *Store) addLedger(entryType, from, to string, amountCents int64, reference string) LedgerEntry {
	previous := strings.Repeat("0", 64)
	if len(s.ledger) > 0 {
		previous = s.ledger[len(s.ledger)-1].EntryHash
	}
	entry := LedgerEntry{
		Sequence:     len(s.ledger) + 1,
		Type:         entryType,
		FromAccount:  from,
		ToAccount:    to,
		AmountCents:  amountCents,
		Reference:    reference,
		PreviousHash: previous,
		CreatedAt:    time.Now().UTC(),
	}
	entry.EntryHash = ledgerEntryHash(entry)
	s.ledger = append(s.ledger, entry)
	return entry
}

func normalizeLedgerAccounts(entries []LedgerEntry) ([]LedgerEntry, bool) {
	normalized := make([]LedgerEntry, len(entries))
	changed := false
	for index, entry := range entries {
		if account, ok := normalizeLedgerAccount(entry.FromAccount); ok {
			entry.FromAccount = account
			changed = true
		}
		if account, ok := normalizeLedgerAccount(entry.ToAccount); ok {
			entry.ToAccount = account
			changed = true
		}
		normalized[index] = entry
	}
	if !changed {
		return normalized, false
	}

	previous := strings.Repeat("0", 64)
	for index := range normalized {
		normalized[index].PreviousHash = previous
		normalized[index].EntryHash = ledgerEntryHash(normalized[index])
		previous = normalized[index].EntryHash
	}
	return normalized, true
}

func normalizeLedgerAccount(account string) (string, bool) {
	trimmed := strings.TrimSpace(account)
	lower := strings.ToLower(trimmed)
	if strings.HasPrefix(lower, "wallet:") {
		normalized := walletAccount(trimmed)
		if !validWalletAddress(normalized) || normalized == trimmed {
			return "", false
		}
		return normalized, true
	}
	if strings.HasPrefix(lower, "reserve:task:") {
		return taskReserveAccount(), true
	}
	return "", false
}

func ledgerEntryHash(entry LedgerEntry) string {
	payload := fmt.Sprintf("%d|%s|%s|%s|%d|%s|%s|%s", entry.Sequence, entry.Type, entry.FromAccount, entry.ToAccount, entry.AmountCents, entry.Reference, entry.PreviousHash, entry.CreatedAt.Format(time.RFC3339Nano))
	sum := sha256.Sum256([]byte(payload))
	return hex.EncodeToString(sum[:])
}

func taskReserveAccount() string {
	return "reserve:task"
}

func (s *Store) load() error {
	if s.storage != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		state, found, err := s.storage.Load(ctx)
		if err != nil {
			return err
		}
		if found {
			if s.applyState(state) {
				return s.saveLocked()
			}
			return nil
		}
		legacy, legacyFound, err := loadJSONState(s.cfg.StatePath)
		if err != nil {
			return fmt.Errorf("load legacy state for postgres import: %w", err)
		}
		if legacyFound {
			s.applyState(legacy)
			return s.saveLocked()
		}
		return nil
	}
	state, found, err := loadJSONState(s.cfg.StatePath)
	if err != nil {
		return err
	}
	if !found {
		return nil
	}
	if s.applyState(state) {
		return s.saveLocked()
	}
	return nil
}

func loadJSONState(path string) (persistedState, bool, error) {
	if strings.TrimSpace(path) == "" {
		return persistedState{}, false, nil
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return persistedState{}, false, nil
	}
	if err != nil {
		return persistedState{}, false, err
	}
	var state persistedState
	if err := json.Unmarshal(data, &state); err != nil {
		return persistedState{}, false, err
	}
	return state, true, nil
}

func (s *Store) applyState(state persistedState) bool {
	migrated := false
	if state.NextID > 0 {
		s.nextID = state.NextID
	}
	s.adminSettings = defaultAdminSettings(s.cfg)
	if state.AdminSettings != nil {
		s.adminSettings = *state.AdminSettings
		s.adminSettings.LLMProvider = normalizedLLMProviderOrDefault(s.adminSettings.LLMProvider)
		if strings.TrimSpace(s.adminSettings.LLMModel) == "" && strings.TrimSpace(s.adminSettings.GeminiReviewModel) != "" {
			s.adminSettings.LLMModel = s.adminSettings.GeminiReviewModel
		}
		s.adminSettings.LLMModel = normalizedLLMModelOrDefault(s.adminSettings.LLMProvider, s.adminSettings.LLMModel)
		if s.adminSettings.LLMProvider == "gemini" {
			s.adminSettings.GeminiReviewModel = s.adminSettings.LLMModel
		} else {
			s.adminSettings.GeminiReviewModel = normalizedGeminiReviewModelOrDefault(s.adminSettings.GeminiReviewModel)
		}
		if s.adminSettings.UpdatedAt.IsZero() {
			s.adminSettings.UpdatedAt = time.Now().UTC()
		}
	}
	s.ledger, migrated = normalizeLedgerAccounts(state.Ledger)
	s.projects = map[string]*Project{}
	s.tasks = map[string]*Task{}
	s.users = map[string]*User{}
	s.wallets = map[string]*Wallet{}
	s.sessions = map[string]*Session{}
	s.notifications = map[string]*Notification{}
	s.attachments = map[string]*Attachment{}
	s.sslReviews = map[string]*SSLReviewStatus{}
	s.geminiAPIKeys = map[string]*GeminiAPIKey{}
	s.geminiWebhookLogs = map[string]*GeminiWebhookLog{}
	for _, project := range state.Projects {
		if project == nil || project.ID == "" {
			continue
		}
		for _, task := range project.Tasks {
			if task == nil {
				continue
			}
			workerID := normalizeWorkerID(task.WorkerID)
			if workerID != task.WorkerID {
				task.WorkerID = workerID
				migrated = true
			}
		}
		s.projects[project.ID] = project
	}
	for _, task := range state.Tasks {
		if task == nil || task.ID == "" {
			continue
		}
		workerID := normalizeWorkerID(task.WorkerID)
		if workerID != task.WorkerID {
			taskCopy := *task
			taskCopy.WorkerID = workerID
			task = &taskCopy
			migrated = true
		}
		s.tasks[task.ID] = task
	}
	for _, user := range state.Users {
		if user == nil || user.ID == "" {
			continue
		}
		user.WalletAddress = normalizeWalletAddress(user.WalletAddress)
		user.GitHubUsername = normalizeGitHubUsername(user.GitHubUsername)
		s.users[user.ID] = user
	}
	for _, wallet := range state.Wallets {
		if wallet == nil {
			continue
		}
		wallet.Address = normalizeWalletAddress(wallet.Address)
		wallet.GitHubUsername = normalizeGitHubUsername(wallet.GitHubUsername)
		if !validWalletAddress(wallet.Address) {
			continue
		}
		s.wallets[wallet.Address] = wallet
	}
	now := time.Now().UTC()
	for _, session := range state.Sessions {
		if session == nil || session.Token == "" {
			continue
		}
		if now.Before(session.ExpiresAt) {
			s.sessions[session.Token] = session
		}
	}
	for _, notification := range state.Notifications {
		if notification == nil || notification.ID == "" {
			continue
		}
		s.notifications[notification.ID] = notification
	}
	for _, attachment := range state.Attachments {
		if attachment == nil || attachment.ID == "" {
			continue
		}
		if attachment.URL == "" {
			attachment.URL = "/api/uploads/" + attachment.ID + "/download"
		}
		s.attachments[attachment.ID] = attachment
	}
	for _, review := range state.SSLReviews {
		if review == nil || review.Domain == "" {
			continue
		}
		review.Domain = cleanDomain(review.Domain)
		s.sslReviews[review.Domain] = cloneSSLReview(review)
	}
	for _, key := range state.GeminiAPIKeys {
		if key == nil || strings.TrimSpace(key.KeyValue) == "" {
			continue
		}
		keyCopy := *key
		if keyCopy.ID == "" {
			keyCopy.ID = geminiAPIKeyID(keyCopy.KeyValue)
		}
		keyCopy.Provider = normalizedLLMProviderOrDefault(keyCopy.Provider)
		keyCopy.Model = normalizedLLMModelOrDefault(keyCopy.Provider, keyCopy.Model)
		if keyCopy.KeyHint == "" {
			keyCopy.KeyHint = geminiAPIKeyHint(keyCopy.KeyValue)
		}
		if keyCopy.Status == "" {
			keyCopy.Status = GeminiAPIKeyStatusActive
		}
		s.geminiAPIKeys[keyCopy.ID] = &keyCopy
	}
	for _, log := range state.GeminiWebhookLogs {
		if log == nil || log.ID == "" {
			continue
		}
		logCopy := *log
		logCopy.Labels = append([]string(nil), log.Labels...)
		s.geminiWebhookLogs[logCopy.ID] = &logCopy
	}
	s.trimGeminiWebhookLogsLocked()
	return migrated
}

func (s *Store) saveLocked() error {
	state := s.snapshotLocked()
	if s.storage != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return s.storage.Save(ctx, state)
	}
	return saveJSONState(s.cfg.StatePath, state)
}

func (s *Store) snapshotLocked() persistedState {
	state := persistedState{
		NextID:            s.nextID,
		Projects:          make([]*Project, 0, len(s.projects)),
		Tasks:             make([]*Task, 0, len(s.tasks)),
		Users:             make([]*User, 0, len(s.users)),
		Wallets:           make([]*Wallet, 0, len(s.wallets)),
		Sessions:          make([]*Session, 0, len(s.sessions)),
		Notifications:     make([]*Notification, 0, len(s.notifications)),
		Attachments:       make([]*Attachment, 0, len(s.attachments)),
		SSLReviews:        make([]*SSLReviewStatus, 0, len(s.sslReviews)),
		GeminiAPIKeys:     make([]*GeminiAPIKey, 0, len(s.geminiAPIKeys)),
		GeminiWebhookLogs: make([]*GeminiWebhookLog, 0, len(s.geminiWebhookLogs)),
		AdminSettings:     cloneAdminSettings(s.adminSettings),
		Ledger:            s.ledger,
	}
	for _, project := range s.projects {
		state.Projects = append(state.Projects, cloneProject(project))
	}
	for _, task := range s.tasks {
		taskCopy := *task
		state.Tasks = append(state.Tasks, &taskCopy)
	}
	for _, user := range s.users {
		userCopy := *user
		state.Users = append(state.Users, &userCopy)
	}
	for _, wallet := range s.wallets {
		walletCopy := *wallet
		state.Wallets = append(state.Wallets, &walletCopy)
	}
	for token, session := range s.sessions {
		sessionCopy := *session
		sessionCopy.Token = token
		state.Sessions = append(state.Sessions, &sessionCopy)
	}
	for _, notification := range s.notifications {
		noteCopy := *notification
		state.Notifications = append(state.Notifications, &noteCopy)
	}
	for _, attachment := range s.attachments {
		attachmentCopy := *attachment
		state.Attachments = append(state.Attachments, &attachmentCopy)
	}
	for _, review := range s.sslReviewRowsLocked() {
		state.SSLReviews = append(state.SSLReviews, review)
	}
	for _, key := range s.geminiAPIKeys {
		keyCopy := *key
		state.GeminiAPIKeys = append(state.GeminiAPIKeys, &keyCopy)
	}
	sort.Slice(state.GeminiAPIKeys, func(i, j int) bool {
		return state.GeminiAPIKeys[i].ID < state.GeminiAPIKeys[j].ID
	})
	for _, log := range s.geminiWebhookLogs {
		logCopy := *log
		logCopy.Labels = append([]string(nil), log.Labels...)
		state.GeminiWebhookLogs = append(state.GeminiWebhookLogs, &logCopy)
	}
	sort.Slice(state.GeminiWebhookLogs, func(i, j int) bool {
		return state.GeminiWebhookLogs[i].ReceivedAt.Before(state.GeminiWebhookLogs[j].ReceivedAt)
	})
	return state
}

func saveJSONState(path string, state persistedState) error {
	if strings.TrimSpace(path) == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func defaultAdminSettings(cfg Config) AdminSettings {
	model := normalizedGeminiReviewModelOrDefault(cfg.GeminiReviewModel)
	return AdminSettings{
		LLMProvider:       defaultLLMProvider,
		LLMModel:          model,
		GeminiReviewModel: model,
		UpdatedAt:         time.Now().UTC(),
	}
}

func adminSettingsResponse(settings AdminSettings) AdminSettingsResponse {
	provider := normalizedLLMProviderOrDefault(settings.LLMProvider)
	model := normalizedLLMModelOrDefault(provider, settings.LLMModel)
	return AdminSettingsResponse{
		LLMProvider:              provider,
		LLMModel:                 model,
		LLMProviderOptions:       llmProviderOptions(),
		GeminiReviewModel:        normalizedGeminiReviewModelOrDefault(settings.GeminiReviewModel),
		GeminiReviewModelOptions: append([]string(nil), llmModelsForProvider("gemini")...),
		UpdatedAt:                settings.UpdatedAt,
	}
}

func cloneAdminSettings(settings AdminSettings) *AdminSettings {
	copy := settings
	copy.LLMProvider = normalizedLLMProviderOrDefault(copy.LLMProvider)
	copy.LLMModel = normalizedLLMModelOrDefault(copy.LLMProvider, copy.LLMModel)
	copy.GeminiReviewModel = normalizedGeminiReviewModelOrDefault(copy.GeminiReviewModel)
	if copy.GeminiReviewModel == defaultGeminiReviewModel && copy.LLMProvider == "gemini" {
		copy.GeminiReviewModel = copy.LLMModel
	}
	if copy.UpdatedAt.IsZero() {
		copy.UpdatedAt = time.Now().UTC()
	}
	return &copy
}

func normalizeGeminiReviewModel(value string) (string, error) {
	return normalizeLLMModel("gemini", value)
}

func normalizeLLMProvider(value string) (string, error) {
	provider := strings.ToLower(strings.TrimSpace(value))
	if provider == "" {
		return "", errors.New("LLM provider is required")
	}
	for _, option := range llmProviderDefinitions {
		if provider == option.ID {
			return provider, nil
		}
	}
	return "", errors.New("unsupported LLM provider")
}

func normalizedLLMProviderOrDefault(value string) string {
	provider, err := normalizeLLMProvider(value)
	if err == nil {
		return provider
	}
	return defaultLLMProvider
}

func normalizeLLMModel(provider, value string) (string, error) {
	provider = normalizedLLMProviderOrDefault(provider)
	model := strings.Trim(strings.TrimSpace(value), "/")
	if provider == "gemini" {
		model = strings.TrimPrefix(model, "models/")
	}
	model = strings.TrimSpace(model)
	if model == "" {
		return "", errors.New("LLM model is required")
	}
	for _, allowed := range llmModelsForProvider(provider) {
		if model == allowed {
			return model, nil
		}
	}
	if !validLLMModelName(model) {
		return "", errors.New("LLM model contains unsupported characters")
	}
	return model, nil
}

func normalizedLLMModelOrDefault(provider, value string) string {
	model, err := normalizeLLMModel(provider, value)
	if err == nil {
		return model
	}
	models := llmModelsForProvider(provider)
	if len(models) > 0 {
		return models[0]
	}
	return defaultGeminiReviewModel
}

func normalizedGeminiReviewModelOrDefault(value string) string {
	model, err := normalizeGeminiReviewModel(value)
	if err == nil {
		return model
	}
	return defaultGeminiReviewModel
}

func validGeminiReviewModelName(value string) bool {
	return validLLMModelName(value)
}

func validLLMModelName(value string) bool {
	if len(value) < 3 || len(value) > 96 {
		return false
	}
	for _, char := range value {
		if char >= 'a' && char <= 'z' {
			continue
		}
		if char >= 'A' && char <= 'Z' {
			continue
		}
		if char >= '0' && char <= '9' {
			continue
		}
		switch char {
		case '.', '_', '-', '/', ':':
			continue
		default:
			return false
		}
	}
	return true
}

func llmModelsForProvider(provider string) []string {
	provider = normalizedLLMProviderOrDefault(provider)
	for _, option := range llmProviderDefinitions {
		if option.ID == provider {
			return append([]string(nil), option.Models...)
		}
	}
	return []string{defaultGeminiReviewModel}
}

func llmProviderOptions() []LLMProviderOption {
	options := make([]LLMProviderOption, 0, len(llmProviderDefinitions))
	for _, option := range llmProviderDefinitions {
		options = append(options, LLMProviderOption{
			ID:     option.ID,
			Label:  option.Label,
			Models: append([]string(nil), option.Models...),
		})
	}
	return options
}

func slug(value string) string {
	clean := strings.ToLower(strings.TrimSpace(value))
	clean = strings.ReplaceAll(clean, " ", "-")
	clean = slugClean.ReplaceAllString(clean, "-")
	clean = strings.Trim(clean, "-")
	if clean == "" {
		return "client"
	}
	if len(clean) > 72 {
		clean = strings.Trim(clean[:72], "-")
	}
	return clean
}

func marketplaceLatestLedgerTime(entries []LedgerEntry) *time.Time {
	if len(entries) == 0 {
		return nil
	}
	latest := entries[0].CreatedAt
	for _, entry := range entries[1:] {
		if entry.CreatedAt.After(latest) {
			latest = entry.CreatedAt
		}
	}
	return &latest
}

func marketplaceClientDisplayName(project *Project) string {
	for _, value := range []string{project.CompanyName, project.ClientName} {
		if display := strings.TrimSpace(value); display != "" {
			return display
		}
	}
	return "MergeOS client"
}

func marketplacePublicRepoURL(value string) string {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(value, "https://") || strings.HasPrefix(value, "http://") {
		return value
	}
	return ""
}

func marketplaceProjectTags(project *Project) []string {
	seen := map[string]bool{}
	tags := []string{}
	add := func(value string) {
		value = strings.TrimSpace(value)
		if value == "" {
			return
		}
		key := strings.ToLower(value)
		if seen[key] {
			return
		}
		seen[key] = true
		tags = append(tags, value)
	}

	add(project.SiteType)
	add(project.PackageTier)
	add(project.RepoProvider)
	for _, task := range project.Tasks {
		add(string(task.RequiredWorkerKind))
		add(marketplaceTitle(task.SuggestedAgentType))
	}
	if len(tags) > 6 {
		return tags[:6]
	}
	return tags
}

func marketplaceWorkerName(workerID, agentType string) string {
	if strings.TrimSpace(agentType) != "" {
		return marketplaceTitle(agentType)
	}
	parts := strings.FieldsFunc(workerID, func(r rune) bool {
		return r == ':' || r == '/' || r == '\\' || r == '@'
	})
	if len(parts) > 0 {
		return marketplaceTitle(parts[len(parts)-1])
	}
	return marketplaceTitle(workerID)
}

func marketplaceTitle(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	words := strings.FieldsFunc(value, func(r rune) bool {
		return r == '-' || r == '_' || r == '.' || r == ':'
	})
	for i, word := range words {
		if word == "" {
			continue
		}
		lower := strings.ToLower(word)
		switch lower {
		case "ai", "qa", "ui", "ux", "api", "go":
			words[i] = strings.ToUpper(lower)
		case "devops":
			words[i] = "DevOps"
		default:
			words[i] = strings.ToUpper(lower[:1]) + lower[1:]
		}
	}
	return strings.Join(words, " ")
}

func cloneProject(project *Project) *Project {
	copyProject := *project
	copyProject.Tasks = make([]*Task, 0, len(project.Tasks))
	for _, task := range project.Tasks {
		taskCopy := *task
		copyProject.Tasks = append(copyProject.Tasks, &taskCopy)
	}
	copyProject.Attachments = make([]*Attachment, 0, len(project.Attachments))
	for _, attachment := range project.Attachments {
		copyProject.Attachments = append(copyProject.Attachments, cloneAttachment(attachment))
	}
	return &copyProject
}

func ledgerEntryMatches(entry LedgerEntry, projectIDs, taskIDs map[string]bool) bool {
	haystack := strings.Join([]string{entry.FromAccount, entry.ToAccount, entry.Reference}, "|")
	for projectID := range projectIDs {
		if strings.Contains(haystack, projectID) {
			return true
		}
	}
	for taskID := range taskIDs {
		if strings.Contains(haystack, taskID) {
			return true
		}
	}
	return false
}

func publicLedgerScope(entry LedgerEntry, projectIDs map[string]bool, taskProjectIDs map[string]string) (string, string) {
	haystack := strings.Join([]string{entry.FromAccount, entry.ToAccount, entry.Reference}, "|")
	for projectID := range projectIDs {
		if strings.Contains(haystack, projectID) {
			return projectID, ""
		}
	}
	for taskID, projectID := range taskProjectIDs {
		if strings.Contains(haystack, taskID) {
			return projectID, taskID
		}
	}
	return "", ""
}

func publicLedgerAccount(account, projectID, taskID string) string {
	account = strings.TrimSpace(account)
	if account == "" {
		return ""
	}
	switch {
	case validWalletAddress(account):
		return walletAccount(account)
	case strings.HasPrefix(account, "payment:"):
		return account
	case strings.HasPrefix(account, "issuer:"):
		return "issuer:mergeos"
	case strings.HasPrefix(account, "treasury:"):
		return "treasury:mergeos"
	case strings.HasPrefix(account, "wallet:"):
		return walletAccount(account)
	case strings.HasPrefix(account, "worker:github:"):
		return githubWorkerAccount(strings.TrimPrefix(account, "worker:"))
	case strings.HasPrefix(account, "github:"):
		return githubWorkerAccount(account)
	case strings.HasPrefix(account, "worker:"):
		return "worker:contributor"
	case account == taskReserveAccount():
		return taskReserveAccount()
	case strings.Contains(account, "reserve:task:"):
		return "reserve:task"
	case strings.Contains(account, "reserve:project:"):
		if projectID != "" {
			return "reserve:project:" + projectID
		}
		return "reserve:project"
	case projectID != "":
		return "project:" + projectID
	default:
		return "ledger:public"
	}
}

func publicLedgerReference(projectID, taskID string, sequence int) string {
	if projectID == "" {
		return fmt.Sprintf("ledger:%d", sequence)
	}
	if taskID != "" {
		return fmt.Sprintf("project:%s;task:%s", projectID, taskID)
	}
	return "project:" + projectID
}

func (s *Store) IsPaymentReferenceUsed(reference string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	ref := strings.TrimSpace(strings.ToLower(reference))
	if ref == "" {
		return false
	}
	for _, project := range s.projects {
		if strings.ToLower(project.PaymentReference) == ref {
			return true
		}
	}
	return false
}
