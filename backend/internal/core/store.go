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
	"strconv"
	"strings"
	"sync"
	"time"
)

var slugClean = regexp.MustCompile(`[^a-z0-9-]+`)
var estimatedEffortPattern = regexp.MustCompile(`(?i)estimated effort:\s*([0-9]+(?:\.[0-9]+)?)\s*hours?`)

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

	nextID              int
	projects            map[string]*Project
	tasks               map[string]*Task
	users               map[string]*User
	wallets             map[string]*Wallet
	sessions            map[string]*Session
	notifications       map[string]*Notification
	attachments         map[string]*Attachment
	sslReviews          map[string]*SSLReviewStatus
	geminiAPIKeys       map[string]*GeminiAPIKey
	geminiWebhookLogs   map[string]*GeminiWebhookLog
	testSettingsConfig  TestSettingsConfig
	testSettingsEntries map[string]*TestSettingsEntry
	adminSettings       AdminSettings
	paymentOrders       map[string]*PaymentOrderIntent
	agentLeases         map[string]*AgentLeaseResponse
	ledger              []LedgerEntry
}

type persistedState struct {
	NextID              int                   `json:"next_id"`
	Projects            []*Project            `json:"projects"`
	Tasks               []*Task               `json:"tasks"`
	Users               []*User               `json:"users"`
	Wallets             []*Wallet             `json:"wallets"`
	Sessions            []*Session            `json:"sessions"`
	Notifications       []*Notification       `json:"notifications"`
	Attachments         []*Attachment         `json:"attachments"`
	SSLReviews          []*SSLReviewStatus    `json:"ssl_reviews"`
	GeminiAPIKeys       []*GeminiAPIKey       `json:"gemini_api_keys"`
	GeminiWebhookLogs   []*GeminiWebhookLog   `json:"gemini_webhook_logs"`
	AdminSettings       *AdminSettings        `json:"admin_settings,omitempty"`
	TestSettingsConfig  *TestSettingsConfig   `json:"test_settings_config,omitempty"`
	TestSettingsEntries []*TestSettingsEntry  `json:"test_settings_entries,omitempty"`
	PaymentOrders       []*PaymentOrderIntent `json:"payment_orders,omitempty"`
	Ledger              []LedgerEntry         `json:"ledger"`
}

type statePersistence interface {
	Load(ctx context.Context) (persistedState, bool, error)
	Save(ctx context.Context, state persistedState) error
	Close() error
}

func NewStore(cfg Config, payments *PaymentManager, repos RepoFactory, emailer *EmailSender) (*Store, error) {
	store := &Store{
		cfg:                 cfg,
		payments:            payments,
		repos:               repos,
		emailer:             emailer,
		nextID:              1,
		projects:            map[string]*Project{},
		tasks:               map[string]*Task{},
		users:               map[string]*User{},
		wallets:             map[string]*Wallet{},
		sessions:            map[string]*Session{},
		notifications:       map[string]*Notification{},
		attachments:         map[string]*Attachment{},
		sslReviews:          map[string]*SSLReviewStatus{},
		geminiAPIKeys:       map[string]*GeminiAPIKey{},
		geminiWebhookLogs:   map[string]*GeminiWebhookLog{},
		testSettingsConfig:  TestSettingsConfig{},
		testSettingsEntries: map[string]*TestSettingsEntry{},
		adminSettings:       defaultAdminSettings(cfg),
		paymentOrders:       map[string]*PaymentOrderIntent{},
		agentLeases:         map[string]*AgentLeaseResponse{},
		ledger:              []LedgerEntry{},
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

func (s *Store) RequestPasswordReset(req PasswordResetRequest) (PasswordResetResponse, error) {
	email, err := normalizeEmail(req.Email)
	if err != nil {
		return PasswordResetResponse{}, err
	}

	response := PasswordResetResponse{
		Status:  "ok",
		Message: "If that email is registered, reset instructions have been sent.",
	}
	var recipient string
	var name string
	s.mu.Lock()
	user := s.userByEmailLocked(email)
	if user != nil {
		recipient = user.Email
		name = strings.TrimSpace(user.Name)
		if name == "" {
			name = user.Email
		}
		s.addNotificationLocked(
			user.ID,
			"",
			"email",
			"Password reset requested",
			"Someone requested password reset instructions for your MergeOS account. If this was not you, keep your current password and contact an admin.",
			"logged:password-reset-requested",
		)
		if err := s.saveLocked(); err != nil {
			s.mu.Unlock()
			return PasswordResetResponse{}, err
		}
	}
	s.mu.Unlock()

	if recipient != "" {
		body := strings.Join([]string{
			fmt.Sprintf("Hi %s,", name),
			"",
			"A password reset was requested for your MergeOS account.",
			"Reply to the site admin or use an admin-managed password update to complete the reset.",
			"",
			"If you did not request this, no action is required.",
		}, "\n")
		s.emailer.Send(recipient, "MergeOS password reset requested", body)
	}
	return response, nil
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
	req.PaymentMethod = normalizeFundingPaymentMethod(req.PaymentMethod)
	if req.BudgetCents < 10000 {
		return nil, errors.New("funding payment must be at least 100 USD")
	}
	if req.PaymentMethod != PaymentPayPal && req.PaymentMethod != PaymentCrypto && req.PaymentMethod != PaymentUSDT && req.PaymentMethod != PaymentStripe {
		return nil, errors.New("payment method must be paypal, crypto, solana spl, or stripe")
	}
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
	if req.PaymentMethod == PaymentPayPal {
		if err := s.ValidatePendingPayPalOrderIntent(userID, req.PaymentReference, PaymentOrderFlowProjectFunding, "", "", req.BudgetCents); err != nil {
			return nil, err
		}
	}

	verification, err := s.payments.Verify(ctx, req)
	if err != nil {
		return nil, err
	}
	return s.createFundedProject(ctx, userID, req, sourceRepoURL, importedIssues, verification)
}

// CreateAdminFundedProject creates a funded project for maintainers without a live payment capture.
// Used to seed sister products (e.g. NeraJob) onto the marketplace like Gomi.
func (s *Store) CreateAdminFundedProject(ctx context.Context, userID string, req CreateProjectRequest) (*Project, error) {
	if strings.TrimSpace(userID) == "" {
		return nil, errors.New("login is required")
	}
	if strings.TrimSpace(req.Title) == "" {
		return nil, errors.New("title is required")
	}
	req.PaymentMethod = normalizeFundingPaymentMethod(req.PaymentMethod)
	if req.PaymentMethod == "" {
		req.PaymentMethod = PaymentPayPal
	}
	if req.BudgetCents < 10000 {
		return nil, errors.New("funding payment must be at least 100 USD")
	}
	if req.PaymentMethod != PaymentPayPal && req.PaymentMethod != PaymentCrypto && req.PaymentMethod != PaymentUSDT && req.PaymentMethod != PaymentStripe {
		return nil, errors.New("payment method must be paypal, crypto, solana spl, or stripe")
	}
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
	reference := strings.TrimSpace(req.PaymentReference)
	if reference == "" {
		reference = "ADMIN-PAYPAL-SEED"
	}
	verification := PaymentVerification{
		Provider:  "admin-paypal",
		Reference: reference,
	}
	return s.createFundedProject(ctx, userID, req, sourceRepoURL, importedIssues, verification)
}

func (s *Store) createFundedProject(ctx context.Context, userID string, req CreateProjectRequest, sourceRepoURL string, importedIssues []*ImportedRepoIssue, verification PaymentVerification) (*Project, error) {
	tokenSymbol := normalizedTokenSymbol(s.cfg.TokenSymbol)

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
	clientEmail, err := normalizeEmail(clientEmail)
	if err != nil {
		return nil, err
	}

	projectID := s.newID("prj")
	fee := req.BudgetCents * s.cfg.PlatformFeeBps / 10000
	workPool := req.BudgetCents - fee
	now := time.Now().UTC()
	allowAgents := createProjectAllowsAgents(req)
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
		// Default: local workspace. Overridden below when a public source repo is funded.
		RepoVisibility: "local-bounty-workspace",
		AllowAgents:    &allowAgents,
		BudgetCents:    req.BudgetCents,
		FeeCents:       fee,
		WorkPoolCents:  workPool,
		Status:         ProjectFunded,
		CreatedAt:      now,
	}
	if verification.Provider == "paypal" {
		if err := s.attachPayPalOrderIntentLocked(user.ID, verification.Reference, PaymentOrderFlowProjectFunding, project.ID, "", req.BudgetCents); err != nil {
			return nil, err
		}
	}
	if sourceRepoURL != "" && !strings.Contains(project.Brief, sourceRepoURL) {
		project.Brief = "Source repository: " + sourceRepoURL + "\n\n" + project.Brief
	}
	// Bind to the public source product repo when funding imported issues.
	// Never create a private mergeos-prj_* child repository.
	if sourceRepoURL != "" {
		if fullName, htmlURL := parseGitHubRepoURL(sourceRepoURL); fullName != "" {
			project.BountyRepoName = fullName
			project.RepoURL = htmlURL
			project.RepoProvider = "github"
			project.RepoVisibility = "source-repository"
		}
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
		project.Tasks = s.tasksFromImportedIssuesWithPolicy(project, importedIssues, allowAgents)
	} else {
		project.Tasks = s.splitProjectTasksWithPolicy(project, allowAgents)
	}

	repo, err := s.repos.CreateProjectRepo(ctx, project, project.Tasks)
	if err != nil {
		return nil, err
	}
	project.BountyRepoName = repo.Name
	project.RepoProvider = repo.Provider
	project.RepoURL = repo.URL
	project.RepoLocalPath = repo.LocalPath
	if project.RepoVisibility == "" || project.RepoVisibility == "private-child-bounty-repo" {
		if repo.Provider == "github" {
			project.RepoVisibility = "source-repository"
		} else {
			project.RepoVisibility = "local-bounty-workspace"
		}
	}
	for _, task := range project.Tasks {
		if issue, ok := repo.Issues[task.ID]; ok {
			if task.IssueNumber == 0 {
				task.IssueNumber = issue.Number
			}
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
	repoLabel := project.BountyRepoName
	if repoLabel == "" {
		repoLabel = project.RepoURL
	}
	body := fmt.Sprintf("Hi %s,\n\nYour project %q is funded against %s and split into %d payable tasks.\n\nBudget: %s %s\nWork pool: %s %s\nAttachments: %d\n\nWe will notify you as tasks are accepted.", project.ClientName, project.Title, repoLabel, len(project.Tasks), formatTokenAmount(project.BudgetCents), tokenSymbol, formatTokenAmount(project.WorkPoolCents), tokenSymbol, len(project.Attachments))
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

func (s *Store) ProjectSnapshot(projectID string) (*Project, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	project, ok := s.projects[strings.TrimSpace(projectID)]
	if !ok {
		return nil, false
	}
	return cloneProject(project), true
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
	_, err := s.SyncProjectImportedIssuesReport(projectID, "", issues)
	return err
}

type repositorySuggestedTaskFundingQuote struct {
	ProjectID       string
	ProjectTitle    string
	SuggestedTaskID string
	TaskTitle       string
	RewardCents     int64
	BudgetCents     int64
	Suggestion      RepositorySuggestedTask
}

func (s *Store) RepositorySuggestedTaskFundingQuote(projectID, suggestedTaskID string, rewardCents, budgetCents int64) (repositorySuggestedTaskFundingQuote, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.repositorySuggestedTaskFundingQuoteLocked(projectID, suggestedTaskID, rewardCents, budgetCents)
}

func (s *Store) FundRepositorySuggestedTask(ctx context.Context, userID, projectID, suggestedTaskID string, req FundRepositorySuggestedTaskRequest) (FundRepositorySuggestedTaskResponse, error) {
	if strings.TrimSpace(req.SuggestedTaskID) != "" && strings.TrimSpace(req.SuggestedTaskID) != strings.TrimSpace(suggestedTaskID) {
		return FundRepositorySuggestedTaskResponse{}, errors.New("suggested task id does not match route")
	}
	quote, err := s.RepositorySuggestedTaskFundingQuote(projectID, suggestedTaskID, req.RewardCents, req.BudgetCents)
	if err != nil {
		return FundRepositorySuggestedTaskResponse{}, err
	}
	paymentMethod := normalizeFundingPaymentMethod(req.PaymentMethod)
	if !supportedFundingPaymentMethod(paymentMethod) {
		return FundRepositorySuggestedTaskResponse{}, errors.New("payment method must be paypal, crypto, solana spl, or stripe")
	}
	if paymentMethod == PaymentPayPal {
		if err := s.ValidatePendingPayPalOrderIntent(userID, req.PaymentReference, PaymentOrderFlowRepositoryTaskFunding, quote.ProjectID, quote.SuggestedTaskID, quote.BudgetCents); err != nil {
			return FundRepositorySuggestedTaskResponse{}, err
		}
	}
	verification, err := s.payments.Verify(ctx, CreateProjectRequest{
		BudgetCents:      quote.BudgetCents,
		PaymentMethod:    paymentMethod,
		PaymentReference: req.PaymentReference,
	})
	if err != nil {
		return FundRepositorySuggestedTaskResponse{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	project, ok := s.projects[strings.TrimSpace(projectID)]
	if !ok {
		return FundRepositorySuggestedTaskResponse{}, errors.New("project not found")
	}
	quote, err = s.repositorySuggestedTaskFundingQuoteLocked(projectID, suggestedTaskID, req.RewardCents, req.BudgetCents)
	if err != nil {
		return FundRepositorySuggestedTaskResponse{}, err
	}
	now := time.Now().UTC()
	task := &Task{
		ID:                 s.newID("tsk"),
		ProjectID:          project.ID,
		IssueNumber:        nextProjectIssueNumber(project),
		Title:              quote.Suggestion.Title,
		Acceptance:         repositorySuggestedTaskAcceptance(quote.Suggestion),
		RewardCents:        quote.RewardCents,
		RequiredWorkerKind: quote.Suggestion.WorkerKind,
		SuggestedAgentType: strings.TrimSpace(quote.Suggestion.SuggestedAgentType),
		BountyType:         repositoryScanSuggestionBountyType,
		Status:             TaskOpen,
		IssueState:         "open",
		CreatedAt:          now,
	}
	if !projectAllowsAgents(project) {
		routeTaskToHuman(task)
	}
	if issue, issueErr := s.repos.CreateProjectTask(ctx, project, task); issueErr == nil && issue != nil {
		if issue.Number > 0 {
			task.IssueNumber = issue.Number
		}
		task.IssueURL = strings.TrimSpace(issue.URL)
	}

	fee := quote.BudgetCents * s.cfg.PlatformFeeBps / 10000
	workPool := quote.BudgetCents - fee
	if workPool < quote.RewardCents {
		workPool = quote.RewardCents
		fee = quote.BudgetCents - workPool
	}
	if fee < 0 {
		fee = 0
	}
	project.BudgetCents += quote.BudgetCents
	project.FeeCents += fee
	project.WorkPoolCents += workPool
	project.Status = ProjectFunded
	project.PaymentStatus = "verified"
	if verification.Provider == "paypal" {
		if err := s.attachPayPalOrderIntentLocked(userID, verification.Reference, PaymentOrderFlowRepositoryTaskFunding, project.ID, quote.SuggestedTaskID, quote.BudgetCents); err != nil {
			return FundRepositorySuggestedTaskResponse{}, err
		}
	}

	s.tasks[task.ID] = task
	s.syncProjectTaskSnapshotLocked(project, task)
	sortTasks(project.Tasks)

	clientProjectAccount := "client:" + project.ClientUserID + ":project:" + project.ID
	ledgerReference := repositorySuggestedTaskLedgerReference(project, task, quote.Suggestion)
	s.addLedger("payment_verified", "payment:"+verification.Provider, clientProjectAccount, quote.BudgetCents, verification.Reference)
	s.addLedger("token_mint", "issuer:mergeos", clientProjectAccount, quote.BudgetCents, "mint:"+project.ID+";task:"+task.ID)
	if fee > 0 {
		s.addLedger("platform_fee", "client:"+project.ID, "treasury:mergeos", fee, "fee:"+project.ID+";task:"+task.ID)
	}
	s.addLedger("project_reserve", "client:"+project.ID, "reserve:project:"+project.ID, workPool, "task:"+task.ID+";suggestion:"+quote.SuggestedTaskID)
	taskReserve := s.addLedger("task_reserve", "reserve:project:"+project.ID, taskReserveAccount(), task.RewardCents, ledgerReference)
	s.addNotificationLocked(project.ClientUserID, project.ID, "repo_task_funded", "Repository task funded", fmt.Sprintf("%s is now funded with %s %s reserved for delivery.", task.Title, formatTokenAmount(task.RewardCents), normalizedTokenSymbol(s.cfg.TokenSymbol)), "logged:repo-task-funded")
	if err := s.saveLocked(); err != nil {
		return FundRepositorySuggestedTaskResponse{}, err
	}
	taskCopy := *task
	taskCopy.IssueURL = marketplacePublicRepoURL(taskCopy.IssueURL)
	workPacket := repositorySuggestedTaskWorkPacket(project, task, quote.Suggestion)
	return FundRepositorySuggestedTaskResponse{
		ProtocolVersion:     "mergeos.repo-task-funding.v1",
		Kind:                "repo_task_funding",
		ProjectID:           project.ID,
		SuggestedTaskID:     quote.SuggestedTaskID,
		Task:                &taskCopy,
		LedgerEntries:       []LedgerEntry{taskReserve},
		FundingReference:    taskReserve.Reference,
		EvidenceChecklist:   append([]string(nil), quote.Suggestion.EvidenceRequired...),
		TaskProtocolURL:     workPacket.ContextURLs["task_protocol"],
		WorkflowProtocolURL: workPacket.ContextURLs["workflow_protocol"],
		ScanProtocolURL:     workPacket.ContextURLs["repository_scan"],
		WorkPacket:          workPacket,
	}, nil
}

func (s *Store) repositorySuggestedTaskFundingQuoteLocked(projectID, suggestedTaskID string, rewardCents, budgetCents int64) (repositorySuggestedTaskFundingQuote, error) {
	project, ok := s.projects[strings.TrimSpace(projectID)]
	if !ok {
		return repositorySuggestedTaskFundingQuote{}, errors.New("project not found")
	}
	scan := s.projectRepositoryScanLocked(project)
	suggestion, ok := findRepositorySuggestedTask(scan.SuggestedTasks, suggestedTaskID)
	if !ok {
		return repositorySuggestedTaskFundingQuote{}, errors.New("suggested task not found")
	}
	if !suggestion.ReadyForBounty || suggestion.FundingPacket.CanFund == false || suggestion.FundingPacket.Status == "already_funded" {
		return repositorySuggestedTaskFundingQuote{}, errors.New("suggested task is already funded")
	}
	recommendedReward := suggestion.FundingPacket.RecommendedRewardCents
	if recommendedReward <= 0 {
		recommendedReward = suggestion.EstimatedRewardCents
	}
	if rewardCents <= 0 {
		rewardCents = recommendedReward
	}
	if rewardCents < recommendedReward {
		return repositorySuggestedTaskFundingQuote{}, fmt.Errorf("reward must be at least %d cents", recommendedReward)
	}
	if budgetCents <= 0 {
		budgetCents = repositoryFundingCentsForReward(rewardCents, s.cfg.PlatformFeeBps)
	}
	if budgetCents < 10000 {
		return repositorySuggestedTaskFundingQuote{}, errors.New("escrow funding must be at least 100 USD")
	}
	if budgetCents < rewardCents {
		return repositorySuggestedTaskFundingQuote{}, errors.New("escrow funding must cover the task reward")
	}
	return repositorySuggestedTaskFundingQuote{
		ProjectID:       project.ID,
		ProjectTitle:    publicLiveFeedProjectTitle(project),
		SuggestedTaskID: suggestion.ID,
		TaskTitle:       suggestion.Title,
		RewardCents:     rewardCents,
		BudgetCents:     budgetCents,
		Suggestion:      suggestion,
	}, nil
}

func findRepositorySuggestedTask(tasks []RepositorySuggestedTask, suggestedTaskID string) (RepositorySuggestedTask, bool) {
	suggestedTaskID = strings.TrimSpace(suggestedTaskID)
	for _, task := range tasks {
		if task.ID == suggestedTaskID || task.SourceFindingID == suggestedTaskID {
			return task, true
		}
	}
	return RepositorySuggestedTask{}, false
}

func normalizeFundingPaymentMethod(method PaymentMethod) PaymentMethod {
	switch strings.ToLower(strings.TrimSpace(string(method))) {
	case "card", "credit_card", "debit_card", "credit / debit card":
		return PaymentStripe
	case string(PaymentPayPal):
		return PaymentPayPal
	case string(PaymentCrypto):
		return PaymentCrypto
	case string(PaymentUSDT), "solana", "solana_spl", "usdc":
		return PaymentUSDT
	case string(PaymentStripe):
		return PaymentStripe
	default:
		return method
	}
}

func supportedFundingPaymentMethod(method PaymentMethod) bool {
	return method == PaymentPayPal || method == PaymentCrypto || method == PaymentUSDT || method == PaymentStripe
}

func repositorySuggestedTaskAcceptance(suggestion RepositorySuggestedTask) string {
	lines := []string{
		"Repository scan suggested task.",
		"Source finding: " + suggestion.SourceFindingID,
		"Signal: " + suggestion.Signal,
	}
	if strings.TrimSpace(suggestion.Path) != "" {
		lines = append(lines, "Path: "+strings.TrimSpace(suggestion.Path))
	}
	if len(suggestion.AcceptanceCriteria) > 0 {
		lines = append(lines, "", "Acceptance criteria:")
		for _, item := range suggestion.AcceptanceCriteria {
			if item = strings.TrimSpace(item); item != "" {
				lines = append(lines, "- "+item)
			}
		}
	}
	if len(suggestion.EvidenceRequired) > 0 {
		lines = append(lines, "", "Evidence required:")
		for _, item := range suggestion.EvidenceRequired {
			if item = strings.TrimSpace(item); item != "" {
				lines = append(lines, "- "+item)
			}
		}
	}
	return strings.Join(lines, "\n")
}

func repositorySuggestedTaskWorkPacket(project *Project, task *Task, suggestion RepositorySuggestedTask) AgentWorkPacket {
	if project == nil || task == nil {
		return AgentWorkPacket{
			ContextURLs:     map[string]string{},
			Runbook:         []AgentRunbookStep{},
			ActionPayloads:  []AgentActionPayload{},
			DelegationChain: []string{ceoAgentType, designReviewAgentType},
		}
	}
	bountyID := marketplaceBountyID(task.ProjectID, task.IssueNumber)
	claimEndpoint := "/api/tasks/" + bountyID + "/claim"
	submitEndpoint := "/api/tasks/" + bountyID + "/submit"
	runEndpoint := "/api/projects/" + task.ProjectID + "/agent-runs"
	actionEndpoint := "/api/projects/" + task.ProjectID + "/agent-actions"
	taskProtocolURL := "/api/public/protocol/tasks?task_id=" + bountyID
	agentType := strings.TrimSpace(task.SuggestedAgentType)
	if agentType == "" && task.RequiredWorkerKind == WorkerAgent {
		agentType = "general-ai-agent"
	}
	contextURLs := map[string]string{
		"task_protocol":     taskProtocolURL,
		"agent_queue":       agentQueueEndpoint,
		"workflow_protocol": "/api/public/projects/" + task.ProjectID + "/workflow",
		"workflow_pulse":    "/api/public/projects/" + task.ProjectID + "/ai-workflow",
		"repository_scan":   "/api/public/projects/" + task.ProjectID + "/repo-scan",
		"pr_monitor":        "/api/public/projects/" + task.ProjectID + "/pull-requests",
		"ceo_agent":         "/api/public/protocol/agents",
		"design_review":     agentQueueEndpoint + "#design-review-agent",
	}
	if issueURL := marketplacePublicRepoURL(task.IssueURL); issueURL != "" {
		contextURLs["issue"] = issueURL
	}
	if repoURL := marketplacePublicRepoURL(projectSourceRepoURL(project)); repoURL != "" {
		contextURLs["repository"] = repoURL
	}
	return AgentWorkPacket{
		ClaimEndpoint:       claimEndpoint,
		RunEndpoint:         runEndpoint,
		ActionEndpoint:      actionEndpoint,
		SubmitEndpoint:      submitEndpoint,
		LeasePacket:         agentLeasePacket(bountyID, agentType),
		SupervisorAgentType: ceoAgentType,
		SubagentType:        agentType,
		DesignReviewAgent:   designReviewAgentType,
		DelegationChain:     agentDelegationChain(agentType),
		ContextURLs:         contextURLs,
		Runbook: []AgentRunbookStep{
			{Step: 1, Action: "fetch_scan", Label: "Fetch repository scan protocol", Method: "GET", Endpoint: contextURLs["repository_scan"]},
			{Step: 2, Action: "fetch_task", Label: "Fetch funded task protocol", Method: "GET", Endpoint: taskProtocolURL},
			{Step: 3, Action: "claim_task", Label: "Claim the funded bounty lane", Method: "POST", Endpoint: claimEndpoint},
			{Step: 4, Action: "create_agent_run", Label: "Create branch, PR plan, action payload, and output contracts", Method: "POST", Endpoint: runEndpoint},
			{Step: 5, Action: "run_agent_checks", Label: "Run scan, review, or test evidence actions", Method: "POST", Endpoint: actionEndpoint},
			{Step: 6, Action: "submit_review", Label: "Submit pull request and proof evidence", Method: "POST", Endpoint: submitEndpoint},
		},
		RunPayloads: []AgentActionPayload{
			repositorySuggestedTaskRunPayload("scan", "Create repository scan run plan", runEndpoint, task, suggestion, contextURLs),
			repositorySuggestedTaskRunPayload("review", "Create review run plan", runEndpoint, task, suggestion, contextURLs),
			repositorySuggestedTaskRunPayload("test", "Create test run plan", runEndpoint, task, suggestion, contextURLs),
		},
		ActionPayloads: []AgentActionPayload{
			repositorySuggestedTaskActionPayload("scan", "Run repository scan check", actionEndpoint, task, suggestion, contextURLs),
			repositorySuggestedTaskActionPayload("review", "Review acceptance criteria", actionEndpoint, task, suggestion, contextURLs),
			repositorySuggestedTaskActionPayload("test", "Attach test evidence", actionEndpoint, task, suggestion, contextURLs),
		},
		OutputContracts: []AgentOutputContract{
			{
				Action:            "create_agent_run",
				ArtifactKind:      "agent_run",
				OutputEndpoint:    runEndpoint,
				OutputProtocol:    "mergeos.agent-run.v1",
				OutputProtocolURL: "/protocol/agent-run.v1.schema.json",
				PublicURL:         taskProtocolURL,
			},
			agentQueueOutputContract("scan", task.ProjectID, actionEndpoint, contextURLs),
			agentQueueOutputContract("review", task.ProjectID, actionEndpoint, contextURLs),
			agentQueueOutputContract("test", task.ProjectID, actionEndpoint, contextURLs),
			{
				Action:            "submit",
				ArtifactKind:      "task_submission",
				OutputEndpoint:    submitEndpoint,
				OutputProtocol:    "mergeos.task-submission.v1",
				OutputProtocolURL: "/protocol/task-submission.v1.schema.json",
				PublicURL:         taskProtocolURL,
			},
		},
	}
}

func repositorySuggestedTaskRunPayload(action, label, endpoint string, task *Task, suggestion RepositorySuggestedTask, contextURLs map[string]string) AgentActionPayload {
	bountyID := marketplaceBountyID(task.ProjectID, task.IssueNumber)
	body := map[string]any{
		"action":            action,
		"claim_id":          bountyID,
		"bounty_id":         bountyID,
		"agent_type":        protocolText(task.SuggestedAgentType, 120, "repo-scan-agent"),
		"base_branch":       "main",
		"objective":         task.Title,
		"source_finding_id": suggestion.SourceFindingID,
		"signal":            suggestion.Signal,
		"context_urls":      agentRunContextURLList(contextURLs),
	}
	if strings.TrimSpace(suggestion.Path) != "" {
		body["path"] = strings.TrimSpace(suggestion.Path)
	}
	return AgentActionPayload{
		Action:   action,
		Label:    label,
		Method:   "POST",
		Endpoint: endpoint,
		Body:     body,
	}
}

func repositorySuggestedTaskActionPayload(action, label, endpoint string, task *Task, suggestion RepositorySuggestedTask, contextURLs map[string]string) AgentActionPayload {
	bountyID := marketplaceBountyID(task.ProjectID, task.IssueNumber)
	body := map[string]any{
		"action":            action,
		"status":            "queued",
		"project_id":        task.ProjectID,
		"claim_id":          bountyID,
		"bounty_id":         bountyID,
		"agent_type":        protocolText(task.SuggestedAgentType, 120, "repo-scan-agent"),
		"delegated_by":      ceoAgentType,
		"design_agent":      designReviewAgentType,
		"source_finding_id": suggestion.SourceFindingID,
		"signal":            suggestion.Signal,
		"evidence_required": append([]string(nil), suggestion.EvidenceRequired...),
		"context_urls":      contextURLs,
	}
	if strings.TrimSpace(suggestion.Path) != "" {
		body["path"] = strings.TrimSpace(suggestion.Path)
	}
	return AgentActionPayload{
		Action:   action,
		Label:    label,
		Method:   "POST",
		Endpoint: endpoint,
		Body:     body,
	}
}

func nextProjectIssueNumber(project *Project) int {
	next := 1
	if project == nil {
		return next
	}
	for _, task := range project.Tasks {
		if task != nil && task.IssueNumber >= next {
			next = task.IssueNumber + 1
		}
	}
	return next
}

func repositorySuggestedTaskLedgerReference(project *Project, task *Task, suggestion RepositorySuggestedTask) string {
	reference := marketplacePublicRepoURL(task.IssueURL)
	if reference == "" && project != nil && task.IssueNumber > 0 {
		if repoURL := marketplacePublicRepoURL(projectSourceRepoURL(project)); repoURL != "" {
			reference = fmt.Sprintf("%s/issues/%d", strings.TrimRight(repoURL, "/"), task.IssueNumber)
		}
	}
	if reference == "" {
		reference = "repo-scan:" + suggestion.SourceFindingID
	}
	return ensureTaskLedgerReference(task.ID, reference+";suggestion:"+suggestion.ID+";signal:"+suggestion.Signal)
}

func (s *Store) SyncProjectImportedIssuesReport(projectID, sourceRepoURL string, issues []*ImportedRepoIssue) (ProjectIssueSyncResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	project, ok := s.projects[strings.TrimSpace(projectID)]
	if !ok {
		return ProjectIssueSyncResponse{}, errors.New("project not found")
	}
	now := time.Now().UTC()
	report := ProjectIssueSyncResponse{
		ProtocolVersion: "mergeos.repo-sync.v1",
		Kind:            "repo_sync",
		ProjectID:       project.ID,
		ProjectTitle:    project.Title,
		SourceRepoURL:   strings.TrimSpace(sourceRepoURL),
		IssueMappings:   []ProjectIssueSyncMapping{},
		SyncedAt:        now,
	}

	type syncMappingCandidate struct {
		issue      *ImportedRepoIssue
		task       *Task
		state      string
		syncStatus string
	}
	candidates := []syncMappingCandidate{}
	existing := map[int]*Task{}
	for _, task := range s.tasks {
		if task.ProjectID == project.ID && task.IssueNumber > 0 {
			existing[task.IssueNumber] = task
		}
	}

	changed := false
	for _, issue := range issues {
		if issue == nil || issue.Number <= 0 {
			continue
		}
		report.ImportedIssueCount++
		state := normalizeIssueState(issue.State)
		if state == "closed" {
			report.ClosedIssueCount++
		} else {
			report.OpenIssueCount++
		}
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
				report.UpdatedTaskCount++
			}
			syncStatus := "unchanged"
			if taskChanged {
				syncStatus = "updated"
			}
			candidates = append(candidates, syncMappingCandidate{issue: issue, task: task, state: state, syncStatus: syncStatus})
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
		if !projectAllowsAgents(project) {
			routeTaskToHuman(task)
		}
		s.tasks[task.ID] = task
		existing[issue.Number] = task
		s.syncProjectTaskSnapshotLocked(project, task)
		changed = true
		report.AddedTaskCount++
		candidates = append(candidates, syncMappingCandidate{issue: issue, task: task, state: state, syncStatus: "added"})
	}

	agentDepth := projectRoutingAgentDepthLocked(s.tasks)
	contributor := projectRoutingTopContributorLocked(s.tasks)
	for _, candidate := range candidates {
		report.IssueMappings = append(report.IssueMappings, projectIssueSyncMapping(candidate.issue, candidate.task, candidate.state, candidate.syncStatus, agentDepth, contributor))
	}
	sort.Slice(report.IssueMappings, func(i, j int) bool {
		if report.IssueMappings[i].IssueNumber == report.IssueMappings[j].IssueNumber {
			return report.IssueMappings[i].TaskID < report.IssueMappings[j].TaskID
		}
		return report.IssueMappings[i].IssueNumber < report.IssueMappings[j].IssueNumber
	})
	report.PlanningPacket = repoSyncPlanningPacket(project.ID, report.SourceRepoURL, report.IssueMappings)

	if !changed {
		return report, nil
	}
	sortTasks(project.Tasks)
	return report, s.saveLocked()
}

func projectIssueSyncMapping(issue *ImportedRepoIssue, task *Task, issueState, syncStatus string, agentDepth map[string]int, contributor *ProjectRoutingContributor) ProjectIssueSyncMapping {
	if task == nil {
		return ProjectIssueSyncMapping{}
	}
	claimID := marketplaceBountyID(task.ProjectID, task.IssueNumber)
	issueTitle := task.Title
	issueURL := task.IssueURL
	if issue != nil {
		if title := strings.TrimSpace(issue.Title); title != "" {
			issueTitle = title
		}
		if url := strings.TrimSpace(issue.URL); url != "" {
			issueURL = url
		}
	}
	if issueState == "" {
		issueState = normalizeIssueState(task.IssueState)
	}
	ready, blockedBy := projectRoutingReadiness(task)
	route := projectRoutingRoute(task, ready, blockedBy, agentDepth, contributor)
	return ProjectIssueSyncMapping{
		IssueNumber:        task.IssueNumber,
		IssueTitle:         protocolText(issueTitle, 500, task.Title),
		IssueState:         issueState,
		IssueURL:           marketplacePublicRepoURL(issueURL),
		SyncStatus:         syncStatus,
		TaskID:             task.ID,
		TaskTitle:          task.Title,
		TaskStatus:         task.Status,
		ClaimID:            claimID,
		ClaimEndpoint:      "/api/tasks/" + claimID + "/claim",
		TaskProtocolURL:    "/api/public/protocol/tasks?task_id=" + claimID,
		ActionEndpoint:     "/api/projects/" + task.ProjectID + "/agent-actions",
		RewardCents:        task.RewardCents,
		RewardMRG:          float64(task.RewardCents) / 100,
		EstimatedHours:     marketplaceEstimatedHours(task),
		RequiredWorkerKind: task.RequiredWorkerKind,
		SuggestedAgentType: strings.TrimSpace(task.SuggestedAgentType),
		Routing:            route,
	}
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

func (s *Store) MarkNotificationRead(userID, notificationID string) *Notification {
	s.mu.Lock()
	defer s.mu.Unlock()

	note, ok := s.notifications[notificationID]
	if !ok {
		return nil
	}
	if note.UserID != userID {
		return nil
	}
	now := time.Now().UTC()
	note.ReadAt = &now
	return note
}

func (s *Store) MarkAllNotificationsRead(userID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, note := range s.notifications {
		if note.UserID == userID && note.ReadAt == nil {
			now := time.Now().UTC()
			note.ReadAt = &now
		}
	}
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
		publicEntry.Reference = publicLedgerReference(projectID, taskID, entry.Sequence, entry.Reference)
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
		ProtocolVersion: "mergeos.marketplace.v1",
		Kind:            "marketplace",
		Stats: MarketplaceStats{
			TokenSymbol:      s.cfg.TokenSymbol,
			LedgerEntryCount: len(s.ledger),
			ProjectCount:     len(s.projects),
			UpdatedAt:        marketplaceLatestLedgerTime(s.ledger),
		},
		Projects:     []*MarketplaceProject{},
		Bounties:     []*MarketplaceBounty{},
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
			if taskIsReleased(task) {
				row.AcceptedTaskCount++
			} else if taskIsOpenForClaim(task) {
				row.OpenTaskCount++
				response.Bounties = append(response.Bounties, marketplaceBountyRow(project, task))
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
	sort.Slice(response.Bounties, func(i, j int) bool {
		if response.Bounties[i].CreatedAt.Equal(response.Bounties[j].CreatedAt) {
			return response.Bounties[i].RewardCents > response.Bounties[j].RewardCents
		}
		return response.Bounties[i].CreatedAt.After(response.Bounties[j].CreatedAt)
	})

	contributors := map[string]*MarketplaceContributor{}
	agents := map[string]*MarketplaceAgent{}
	for _, task := range s.tasks {
		if task.SuggestedAgentType != "" {
			agent := agents[task.SuggestedAgentType]
			if agent == nil {
				agent = &MarketplaceAgent{
					Type:               task.SuggestedAgentType,
					Title:              marketplaceTitle(task.SuggestedAgentType),
					WorkerKind:         task.RequiredWorkerKind,
					Role:               "subagent",
					ParentAgentType:    ceoAgentType,
					DelegationEndpoint: agentQueueEndpoint,
				}
				agents[task.SuggestedAgentType] = agent
			}
			agent.TaskCount++
			if taskIsOpenForClaim(task) {
				agent.OpenTaskCount++
				agent.BudgetCents += task.RewardCents
			}
		}

		if !taskIsReleased(task) || strings.TrimSpace(task.WorkerID) == "" {
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
	ensureAgentHierarchy(agents)

	for _, contributor := range contributors {
		hasGitHub, hasWallet, duplicateIdentityCount := s.workerIdentitySignalsForWorkerIDLocked(contributor.WorkerID)
		audit := workerReputationAudit(WorkerReputationAudit{
			WorkerID:               contributor.WorkerID,
			Name:                   contributor.Name,
			Kind:                   contributor.Kind,
			AgentType:              contributor.AgentType,
			CompletedTaskCount:     contributor.TaskCount,
			RewardCents:            contributor.EarnedCents,
			RewardRowCount:         contributor.TaskCount,
			HasGitHub:              hasGitHub,
			HasWallet:              hasWallet,
			DuplicateIdentityCount: duplicateIdentityCount,
			LastPaidAt:             nonZeroTimePointer(contributor.LastPaidAt),
		})
		contributor.ReputationScore = audit.Score
		contributor.ReputationLevel = audit.Level
		contributor.RiskLevel = audit.RiskLevel
		contributor.LedgerProofURL = "/api/public/ledger/proof"
		contributor.Flags = audit.Flags
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

type coreMarketplaceAgentSpec struct {
	Type  string
	Title string
}

var coreMarketplaceAgentSpecs = []coreMarketplaceAgentSpec{
	{Type: designReviewAgentType, Title: "Design Review Agent"},
	{Type: "coding-agent", Title: "Coding Agent"},
	{Type: "qa-agent", Title: "QA Agent"},
	{Type: "review-agent", Title: "Review Agent"},
	{Type: "deployment-agent", Title: "Deployment Agent"},
	{Type: "repo-scan-agent", Title: "Repo Scan Agent"},
	{Type: "security-review-agent", Title: "Security Review Agent"},
}

func ensureAgentHierarchy(agents map[string]*MarketplaceAgent) {
	ensureCoreMarketplaceAgents(agents)

	subagents := []string{}
	totalTaskCount := 0
	totalOpenTaskCount := 0
	totalBudgetCents := int64(0)
	for agentType, agent := range agents {
		if agent == nil || agentType == ceoAgentType {
			continue
		}
		agent.Role = protocolText(agent.Role, 80, "subagent")
		agent.ParentAgentType = protocolText(agent.ParentAgentType, 120, ceoAgentType)
		agent.DelegationEndpoint = protocolText(agent.DelegationEndpoint, 240, agentQueueEndpoint)
		if len(agent.Focus) == 0 {
			agent.Focus = defaultAgentFocus(agent.Type)
		}
		agent.SupportedActions = publicAgentSupportedActions(agent)
		subagents = append(subagents, agent.Type)
		totalTaskCount += agent.TaskCount
		totalOpenTaskCount += agent.OpenTaskCount
		totalBudgetCents += agent.BudgetCents
	}

	agents[ceoAgentType] = &MarketplaceAgent{
		Type:               ceoAgentType,
		Title:              "CEO Strategy Agent",
		WorkerKind:         WorkerAgent,
		Role:               "ceo_planner",
		SubagentTypes:      stableStrings(subagents),
		DelegationEndpoint: agentQueueEndpoint,
		Focus: []string{
			"idea_generation",
			"task_decomposition",
			"subagent_delegation",
			"quality_gate",
		},
		SupportedActions: []string{"generate", "review", "scan"},
		TaskCount:        totalTaskCount,
		OpenTaskCount:    totalOpenTaskCount,
		BudgetCents:      totalBudgetCents,
	}
}

func ensureCoreMarketplaceAgents(agents map[string]*MarketplaceAgent) {
	for _, spec := range coreMarketplaceAgentSpecs {
		agent := agents[spec.Type]
		if agent == nil {
			agent = &MarketplaceAgent{
				Type: spec.Type,
			}
			agents[spec.Type] = agent
		}
		if strings.TrimSpace(agent.Title) == "" {
			agent.Title = spec.Title
		}
		if agent.WorkerKind == "" {
			agent.WorkerKind = WorkerAgent
		}
		if strings.TrimSpace(agent.Role) == "" {
			agent.Role = "subagent"
		}
		if strings.TrimSpace(agent.ParentAgentType) == "" {
			agent.ParentAgentType = ceoAgentType
		}
		if strings.TrimSpace(agent.DelegationEndpoint) == "" {
			agent.DelegationEndpoint = agentQueueEndpoint
		}
		if len(agent.Focus) == 0 {
			agent.Focus = defaultAgentFocus(agent.Type)
		}
		agent.SupportedActions = publicAgentSupportedActions(agent)
	}
}

func defaultAgentFocus(agentType string) []string {
	normalized := strings.ToLower(strings.TrimSpace(agentType))
	if normalized == designReviewAgentType || containsAny(normalized, []string{"design", "ui", "ux"}) {
		return []string{"ux_review", "responsive_design", "visual_quality"}
	}
	if containsAny(normalized, []string{"frontend", "code", "coding", "build"}) {
		return []string{"implementation", "component_quality", "handoff_evidence"}
	}
	if containsAny(normalized, []string{"qa", "test"}) {
		return []string{"test_plan", "smoke_testing", "regression_evidence"}
	}
	if containsAny(normalized, []string{"security", "dependency"}) {
		return []string{"repository_scan", "risk_detection", "security_review"}
	}
	if containsAny(normalized, []string{"review", "audit"}) {
		return []string{"code_review", "pr_review", "evidence_review"}
	}
	if containsAny(normalized, []string{"deploy", "release", "devops"}) {
		return []string{"release_gate", "deployment_health", "rollback_readiness"}
	}
	if containsAny(normalized, []string{"scan"}) {
		return []string{"repository_scan", "risk_detection", "security_review"}
	}
	return []string{"task_execution", "evidence_reporting"}
}

func stringSliceContains(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}

func (s *Store) WorkerDashboard(userID string) WorkerDashboardResponse {
	s.mu.RLock()
	defer s.mu.RUnlock()

	user := s.users[strings.TrimSpace(userID)]
	if user == nil {
		return WorkerDashboardResponse{}
	}

	workerIDs, rewardAccounts := workerIdentitySets(user)
	response := WorkerDashboardResponse{
		ProtocolVersion: "mergeos.worker-dashboard.v1",
		Kind:            "worker_dashboard",
		Profile: WorkerProfile{
			UserID:          user.ID,
			Name:            user.Name,
			Email:           user.Email,
			WalletAddress:   normalizeWalletAddress(user.WalletAddress),
			GitHubUsername:  normalizeGitHubUsername(user.GitHubUsername),
			GitHubAvatarURL: user.GitHubAvatarURL,
		},
		ClaimedTasks:       []WorkerClaimedTask{},
		Rewards:            []WorkerRewardEntry{},
		Reputation:         []WorkerReputation{},
		Proposals:          []WorkerProposal{},
		SubmittedProposals: []WorkerSubmittedProposal{},
		IdentityStatus:     workerIdentityHints(user),
	}

	for _, task := range s.tasks {
		if !taskHasWorker(task) || !workerIDs[workerIdentityKey(task.WorkerID)] {
			continue
		}
		project := s.projects[task.ProjectID]
		response.ClaimedTasks = append(response.ClaimedTasks, workerClaimedTaskRow(project, task))
		response.Stats.ClaimedTaskCount++
		if taskIsReleased(task) {
			response.Stats.RewardCents += task.RewardCents
		}
		if taskIsReleased(task) && task.AcceptedAt != nil && (response.Stats.LastPaidAt == nil || task.AcceptedAt.After(*response.Stats.LastPaidAt)) {
			lastPaidAt := *task.AcceptedAt
			response.Stats.LastPaidAt = &lastPaidAt
		}
	}
	sort.Slice(response.ClaimedTasks, func(i, j int) bool {
		left := response.ClaimedTasks[i].AcceptedAt
		right := response.ClaimedTasks[j].AcceptedAt
		if left == nil || right == nil {
			return response.ClaimedTasks[i].IssueNumber > response.ClaimedTasks[j].IssueNumber
		}
		return left.After(*right)
	})

	for _, entry := range s.ledger {
		if entry.Type != "task_payment" && entry.Type != "manual_credit" {
			continue
		}
		if !rewardAccounts[rewardAccountKey(entry.ToAccount)] {
			continue
		}
		response.Rewards = append(response.Rewards, WorkerRewardEntry{
			Sequence:       entry.Sequence,
			Type:           entry.Type,
			AmountCents:    entry.AmountCents,
			Reference:      publicWorkerRewardReference(entry.Reference),
			EntryHash:      entry.EntryHash,
			LedgerProofURL: "/api/public/ledger/proof",
			CreatedAt:      entry.CreatedAt,
		})
	}
	sort.Slice(response.Rewards, func(i, j int) bool {
		return response.Rewards[i].CreatedAt.After(response.Rewards[j].CreatedAt)
	})

	response.Proposals = workerProposalRows(s.projects, s.tasks, user)
	response.Stats.OpenProposalCount = len(response.Proposals)
	response.SubmittedProposals = s.workerSubmittedProposalsLocked(user.ID)
	response.Stats.SubmittedProposalCount = len(response.SubmittedProposals)
	response.ReputationAudit = workerReputationAudit(WorkerReputationAudit{
		WorkerID:               workerDashboardID(response.Profile),
		Name:                   response.Profile.Name,
		Kind:                   WorkerHuman,
		CompletedTaskCount:     response.Stats.ClaimedTaskCount,
		RewardCents:            response.Stats.RewardCents,
		RewardRowCount:         len(response.Rewards),
		HasGitHub:              response.Profile.GitHubUsername != "",
		HasWallet:              response.Profile.WalletAddress != "",
		DuplicateIdentityCount: s.duplicateIdentityCountLocked(user),
		LastPaidAt:             response.Stats.LastPaidAt,
	})
	response.Stats.ReputationScore = response.ReputationAudit.Score
	response.Stats.RiskLevel = response.ReputationAudit.RiskLevel
	response.Reputation = workerReputationRows(response)
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
	if audit := s.workerReputationAuditForUserLocked(user); strings.TrimSpace(audit.WorkerID) != "" {
		row.WorkerAudit = &audit
	}
	return row
}

func (s *Store) AdminReputation() AdminReputationResponse {
	s.mu.RLock()
	defer s.mu.RUnlock()

	audits := map[string]WorkerReputationAudit{}
	for _, task := range s.tasks {
		if !taskIsReleased(task) || strings.TrimSpace(task.WorkerID) == "" {
			continue
		}
		key := workerReputationKey(task.WorkerID, task.AgentType)
		audit := audits[key]
		if strings.TrimSpace(audit.WorkerID) == "" {
			audit.WorkerID = normalizeWorkerID(task.WorkerID)
			audit.Name = marketplaceWorkerName(task.WorkerID, task.AgentType)
			audit.Kind = task.WorkerKind
			audit.AgentType = strings.TrimSpace(task.AgentType)
			audit.HasGitHub, audit.HasWallet, audit.DuplicateIdentityCount = s.workerIdentitySignalsForWorkerIDLocked(task.WorkerID)
		}
		audit.CompletedTaskCount++
		audit.RewardCents += task.RewardCents
		audit.RewardRowCount++
		if task.AcceptedAt != nil && (audit.LastPaidAt == nil || task.AcceptedAt.After(*audit.LastPaidAt)) {
			lastPaidAt := *task.AcceptedAt
			audit.LastPaidAt = &lastPaidAt
		}
		audits[key] = audit
	}

	for _, user := range s.users {
		if user == nil {
			continue
		}
		audit := s.workerReputationAuditForUserLocked(user)
		if strings.TrimSpace(audit.WorkerID) == "" {
			continue
		}
		key := workerReputationKey(audit.WorkerID, audit.AgentType)
		existing := audits[key]
		if existing.CompletedTaskCount > audit.CompletedTaskCount {
			existing.HasGitHub = existing.HasGitHub || audit.HasGitHub
			existing.HasWallet = existing.HasWallet || audit.HasWallet
			existing.DuplicateIdentityCount = max(existing.DuplicateIdentityCount, audit.DuplicateIdentityCount)
			existing = workerReputationAudit(existing)
			audits[key] = existing
			continue
		}
		audits[key] = audit
	}

	response := AdminReputationResponse{Workers: []WorkerReputationAudit{}}
	for _, audit := range audits {
		audit = workerReputationAudit(audit)
		response.Workers = append(response.Workers, audit)
		response.Stats.WorkerCount++
		response.Stats.CompletedTaskCount += audit.CompletedTaskCount
		switch audit.RiskLevel {
		case "high":
			response.Stats.HighRiskCount++
		case "medium":
			response.Stats.MediumRiskCount++
		default:
			response.Stats.LowRiskCount++
		}
		switch audit.Level {
		case "elite", "trusted":
			response.Stats.TrustedCount++
		case "new":
			response.Stats.NewWorkerCount++
		}
	}
	sort.Slice(response.Workers, func(i, j int) bool {
		left, right := response.Workers[i], response.Workers[j]
		if workerRiskRank(left.RiskLevel) != workerRiskRank(right.RiskLevel) {
			return workerRiskRank(left.RiskLevel) > workerRiskRank(right.RiskLevel)
		}
		if left.Score == right.Score {
			return left.RewardCents > right.RewardCents
		}
		return left.Score > right.Score
	})
	return response
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
		if taskIsReleased(task) {
			summary.AcceptedTaskCount++
			summary.PaidTaskCents += task.RewardCents
			continue
		}
		if taskIsOpenForClaim(task) {
			summary.OpenTaskCount++
		}
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

func (s *Store) ResolveTaskClaimID(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", errors.New("task id is required")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.resolveTaskClaimIDLocked(value)
}

func (s *Store) SelfAcceptTaskRequest(userID, taskID string) (AcceptTaskRequest, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	user := s.users[strings.TrimSpace(userID)]
	if user == nil {
		return AcceptTaskRequest{}, errors.New("login is required")
	}
	task, ok := s.tasks[strings.TrimSpace(taskID)]
	if !ok {
		return AcceptTaskRequest{}, errors.New("task not found")
	}
	if task.Status != TaskOpen {
		return AcceptTaskRequest{}, errors.New("task is already claimed")
	}

	workerID := ""
	if github := normalizeGitHubUsername(user.GitHubUsername); github != "" {
		workerID = githubWorkerAccount(github)
	} else if wallet := normalizeWalletAddress(user.WalletAddress); validWalletAddress(wallet) {
		workerID = walletAccount(wallet)
	}
	if workerID == "" {
		return AcceptTaskRequest{}, errors.New("GitHub or wallet identity is required to claim tasks")
	}

	req := AcceptTaskRequest{
		WorkerKind: task.RequiredWorkerKind,
		WorkerID:   workerID,
	}
	if req.WorkerKind != WorkerHuman {
		req.AgentType = strings.TrimSpace(task.SuggestedAgentType)
		if req.AgentType == "" {
			req.AgentType = "worker-dashboard"
		}
	}
	return req, nil
}

func (s *Store) ClaimTask(taskID string, req AcceptTaskRequest) (*Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	task, err := s.claimTaskLocked(taskID, req)
	if err != nil {
		return nil, err
	}
	if err := s.saveLocked(); err != nil {
		return nil, err
	}

	copyTask := *task
	return &copyTask, nil
}

func (s *Store) AcceptTask(taskID string, req AcceptTaskRequest) (*Task, LedgerEntry, error) {
	return s.AcceptTaskWithReview(taskID, req, 0, "")
}

func (s *Store) AcceptTaskWithReview(taskID string, req AcceptTaskRequest, rewardCents int64, bountyType string) (*Task, LedgerEntry, error) {
	return s.AcceptTaskWithReviewReference(taskID, req, rewardCents, bountyType, "")
}

func (s *Store) AcceptTaskWithReviewReference(taskID string, req AcceptTaskRequest, rewardCents int64, bountyType, reference string) (*Task, LedgerEntry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	task, entry, err := s.acceptTaskWithReviewReferenceLocked(taskID, req, rewardCents, bountyType, reference)
	if err != nil {
		return nil, LedgerEntry{}, err
	}
	if err := s.saveLocked(); err != nil {
		return nil, LedgerEntry{}, err
	}

	copyTask := *task
	return &copyTask, entry, nil
}

func (s *Store) claimTaskLocked(taskID string, req AcceptTaskRequest) (*Task, error) {
	task, ok := s.tasks[taskID]
	if !ok {
		return nil, errors.New("task not found")
	}
	if task.Status != TaskOpen {
		return nil, errors.New("task is already claimed")
	}
	if err := validateTaskWorkerRequest(task, req); err != nil {
		return nil, err
	}

	workerID := normalizeWorkerID(req.WorkerID)
	now := time.Now().UTC()
	task.Status = TaskClaimed
	task.WorkerKind = req.WorkerKind
	task.WorkerID = workerID
	task.AgentType = strings.TrimSpace(req.AgentType)
	task.AcceptedAt = &now

	if project, ok := s.projects[task.ProjectID]; ok {
		updateProjectTaskLocked(project, task)
		subject := "MergeOS task claimed: " + task.Title
		body := fmt.Sprintf("Task #%d was claimed by %s. Payout remains in escrow until review evidence is accepted.", task.IssueNumber, task.WorkerID)
		status := s.emailer.Send(project.ClientEmail, subject, body)
		s.addNotificationLocked(project.ClientUserID, project.ID, "task", subject, body, status)
	}
	return task, nil
}

func (s *Store) acceptTaskWithReviewReferenceLocked(taskID string, req AcceptTaskRequest, rewardCents int64, bountyType, reference string) (*Task, LedgerEntry, error) {
	task, ok := s.tasks[taskID]
	if !ok {
		return nil, LedgerEntry{}, errors.New("task not found")
	}
	if task.Status == TaskAccepted {
		return nil, LedgerEntry{}, errors.New("task is already accepted")
	}
	if err := validateTaskWorkerRequest(task, req); err != nil {
		return nil, LedgerEntry{}, err
	}

	workerID := normalizeWorkerID(req.WorkerID)
	if strings.TrimSpace(task.WorkerID) != "" && workerIdentityKey(task.WorkerID) != workerIdentityKey(workerID) {
		return nil, LedgerEntry{}, errors.New("release worker must match the claimed task")
	}
	payoutCents := task.RewardCents
	if rewardCents > 0 {
		payoutCents = rewardCents
		task.RewardCents = rewardCents
	}
	task.BountyType = strings.TrimSpace(bountyType)
	now := time.Now().UTC()
	entry := s.addLedger("task_payment", taskReserveAccount(), s.payoutAccountForWorkerLocked(workerID), payoutCents, ensureTaskLedgerReference(task.ID, reference))
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
	return task, entry, nil
}

func validateTaskWorkerRequest(task *Task, req AcceptTaskRequest) error {
	if task == nil {
		return errors.New("task not found")
	}
	if req.WorkerKind != WorkerHuman && req.WorkerKind != WorkerAgent && req.WorkerKind != WorkerHybrid {
		return errors.New("worker kind must be human, agent, or hybrid")
	}
	if strings.TrimSpace(req.WorkerID) == "" {
		return errors.New("worker id is required")
	}
	if task.RequiredWorkerKind != req.WorkerKind {
		return fmt.Errorf("task requires %s work", task.RequiredWorkerKind)
	}
	if req.WorkerKind != WorkerHuman && strings.TrimSpace(req.AgentType) == "" {
		return errors.New("agent type is required for agent or hybrid work")
	}
	if req.WorkerKind == WorkerHuman && strings.TrimSpace(req.AgentType) != "" {
		return errors.New("agent type must be empty for human work")
	}
	return nil
}

func (s *Store) TaskPayoutAccount(taskID string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	reference := "task:" + strings.TrimSpace(taskID)
	for index := len(s.ledger) - 1; index >= 0; index-- {
		entry := s.ledger[index]
		if entry.Type == "task_payment" && (entry.Reference == reference || ledgerReferenceTaskID(entry.Reference) == strings.TrimSpace(taskID)) {
			return entry.ToAccount, true
		}
	}
	task, ok := s.tasks[taskID]
	if !ok || strings.TrimSpace(task.WorkerID) == "" {
		return "", false
	}
	return s.payoutAccountForWorkerLocked(task.WorkerID), true
}

func (s *Store) AddManualCredit(workerID string, amountCents int64, reference string) (LedgerEntry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	workerID = normalizeWorkerID(workerID)
	if strings.TrimSpace(workerID) == "" {
		return LedgerEntry{}, errors.New("worker id is required")
	}
	if amountCents <= 0 {
		return LedgerEntry{}, errors.New("amount must be greater than zero")
	}
	reference = strings.TrimSpace(reference)
	if reference == "" {
		return LedgerEntry{}, errors.New("reference is required")
	}
	entry := s.addLedger("manual_credit", taskReserveAccount(), s.payoutAccountForWorkerLocked(workerID), amountCents, reference)
	if err := s.saveLocked(); err != nil {
		return LedgerEntry{}, err
	}
	return entry, nil
}

func (s *Store) userByEmailLocked(email string) *User {
	for _, user := range s.users {
		if user.Email == email {
			return user
		}
	}
	return nil
}

func (s *Store) addNotificationLocked(userID, projectID, channel, subject, body, status string) *Notification {
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
	return note
}

func (s *Store) newID(prefix string) string {
	id := fmt.Sprintf("%s_%04d", prefix, s.nextID)
	s.nextID++
	return id
}

func createProjectAllowsAgents(req CreateProjectRequest) bool {
	if req.AllowAgents == nil {
		return true
	}
	return *req.AllowAgents
}

func projectAllowsAgents(project *Project) bool {
	if project == nil || project.AllowAgents == nil {
		return true
	}
	return *project.AllowAgents
}

func (s *Store) splitProjectTasks(project *Project) []*Task {
	return s.splitProjectTasksWithPolicy(project, true)
}

func (s *Store) splitProjectTasksWithPolicy(project *Project, allowAgents bool) []*Task {
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
		if !allowAgents {
			routeTaskToHuman(task)
		}
		tasks = append(tasks, task)
	}
	return tasks
}

func (s *Store) tasksFromImportedIssues(project *Project, issues []*ImportedRepoIssue) []*Task {
	return s.tasksFromImportedIssuesWithPolicy(project, issues, true)
}

func (s *Store) tasksFromImportedIssuesWithPolicy(project *Project, issues []*ImportedRepoIssue, allowAgents bool) []*Task {
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
		if !allowAgents {
			routeTaskToHuman(task)
		}
		tasks = append(tasks, task)
	}
	return tasks
}

func routeTaskToHuman(task *Task) {
	if task == nil {
		return
	}
	task.RequiredWorkerKind = WorkerHuman
	task.SuggestedAgentType = ""
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
	if issue.EstimatedHours > 0 {
		parts = append(parts, "Estimated effort: "+formatEstimatedHours(issue.EstimatedHours)+".")
	}
	if len(issue.Reasons) > 0 {
		parts = append(parts, "Scoring signals: "+strings.Join(issue.Reasons, ", ")+".")
	}
	parts = append(parts, "Acceptance requires passing checks, a clear fix summary, and evidence that the original issue can be closed.")
	return strings.Join(parts, " ")
}

func formatEstimatedHours(hours float64) string {
	if hours <= 0 {
		return "0 hours"
	}
	rounded := roundHalfHour(hours)
	if rounded == float64(int64(rounded)) {
		return fmt.Sprintf("%d hours", int64(rounded))
	}
	return fmt.Sprintf("%.1f hours", rounded)
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

func normalizeLedgerAccounts(entries []LedgerEntry, walletMigration map[string]string) ([]LedgerEntry, bool) {
	normalized := make([]LedgerEntry, len(entries))
	changed := false
	for index, entry := range entries {
		if account, ok := normalizeLedgerAccount(entry.FromAccount, walletMigration); ok {
			entry.FromAccount = account
			changed = true
		}
		if account, ok := normalizeLedgerAccount(entry.ToAccount, walletMigration); ok {
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

func normalizeLedgerAccount(account string, walletMigration map[string]string) (string, bool) {
	trimmed := strings.TrimSpace(account)
	lower := strings.ToLower(trimmed)
	if address, ok := migratedWalletAccount(trimmed, walletMigration); ok {
		return walletAccount(address), true
	}
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

func walletMigrationKey(value string) string {
	value = strings.TrimSpace(value)
	if validLegacyEVMWalletAddress(value) {
		return strings.ToLower(value)
	}
	return value
}

func buildWalletMigrationMap(state persistedState) map[string]string {
	migration := map[string]string{}
	add := func(legacyAddress, solanaAddress string) {
		legacyAddress = normalizeLegacyWalletAddress(legacyAddress)
		if !validLegacyWalletAddress(legacyAddress) {
			return
		}
		solanaAddress = normalizeWalletAddress(solanaAddress)
		if !validWalletAddress(solanaAddress) {
			solanaAddress = solanaWalletFromLegacy(legacyAddress)
		}
		if !validWalletAddress(solanaAddress) {
			return
		}
		migration[walletMigrationKey(legacyAddress)] = solanaAddress
		migration[walletMigrationKey(legacyWalletAccount(legacyAddress))] = solanaAddress
	}
	for _, wallet := range state.Wallets {
		if wallet == nil {
			continue
		}
		address := normalizeWalletAddress(wallet.Address)
		if validLegacyWalletAddress(address) {
			add(address, "")
		}
		if legacyAddress := normalizeLegacyWalletAddress(wallet.LegacyAddress); legacyAddress != "" {
			add(legacyAddress, address)
		}
	}
	for _, user := range state.Users {
		if user != nil {
			add(user.WalletAddress, "")
		}
	}
	for _, task := range state.Tasks {
		if task != nil {
			add(task.WorkerID, "")
		}
	}
	for _, project := range state.Projects {
		if project == nil {
			continue
		}
		for _, task := range project.Tasks {
			if task != nil {
				add(task.WorkerID, "")
			}
		}
	}
	for _, entry := range state.Ledger {
		add(entry.FromAccount, "")
		add(entry.ToAccount, "")
	}
	return migration
}

func migratedWalletAccount(value string, walletMigration map[string]string) (string, bool) {
	if len(walletMigration) == 0 {
		return "", false
	}
	trimmed := strings.TrimSpace(value)
	lookupValues := []string{trimmed}
	if strings.HasPrefix(strings.ToLower(trimmed), "wallet:") {
		lookupValues = append(lookupValues, trimAddressPrefix(trimmed, "wallet:"))
	}
	for _, candidate := range lookupValues {
		legacyAddress := normalizeLegacyWalletAddress(candidate)
		if !validLegacyWalletAddress(legacyAddress) {
			continue
		}
		if solanaAddress, ok := walletMigration[walletMigrationKey(legacyAddress)]; ok && validWalletAddress(solanaAddress) {
			return solanaAddress, true
		}
	}
	return "", false
}

func normalizeWorkerIDWithWalletMigration(value string, walletMigration map[string]string) string {
	if address, ok := migratedWalletAccount(value, walletMigration); ok {
		return walletAccount(address)
	}
	return normalizeWorkerID(value)
}

func normalizeUserWalletWithMigration(value string, walletMigration map[string]string) (string, bool) {
	if address, ok := migratedWalletAccount(value, walletMigration); ok {
		return address, true
	}
	normalized := normalizeWalletAddress(value)
	if validWalletAddress(normalized) {
		return normalized, normalized != value
	}
	return "", strings.TrimSpace(value) != ""
}

func normalizePersistedWalletWithMigration(wallet *Wallet, walletMigration map[string]string) bool {
	if wallet == nil {
		return false
	}
	changed := false
	previousAddress := wallet.Address
	previousChain := wallet.Chain
	previousLegacyAddress := wallet.LegacyAddress
	if address, ok := migratedWalletAccount(wallet.Address, walletMigration); ok {
		if wallet.LegacyAddress == "" {
			wallet.LegacyAddress = normalizeLegacyWalletAddress(wallet.Address)
		}
		wallet.Address = address
	} else {
		wallet.Address = normalizeWalletAddress(wallet.Address)
	}
	wallet.Chain = walletChainSolana
	wallet.LegacyAddress = normalizeLegacyWalletAddress(wallet.LegacyAddress)
	wallet.GitHubUsername = normalizeGitHubUsername(wallet.GitHubUsername)
	if previousAddress != wallet.Address || previousChain != wallet.Chain || previousLegacyAddress != wallet.LegacyAddress {
		changed = true
	}
	return changed
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
	walletMigration := buildWalletMigrationMap(state)
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
	var ledgerMigrated bool
	s.ledger, ledgerMigrated = normalizeLedgerAccounts(state.Ledger, walletMigration)
	migrated = migrated || ledgerMigrated
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
	s.paymentOrders = map[string]*PaymentOrderIntent{}
	for _, project := range state.Projects {
		if project == nil || project.ID == "" {
			continue
		}
		if project.AllowAgents == nil {
			allowAgents := true
			project.AllowAgents = &allowAgents
			migrated = true
		}
		for _, task := range project.Tasks {
			if task == nil {
				continue
			}
			workerID := normalizeWorkerIDWithWalletMigration(task.WorkerID, walletMigration)
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
		workerID := normalizeWorkerIDWithWalletMigration(task.WorkerID, walletMigration)
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
		normalizedWallet, walletMigrated := normalizeUserWalletWithMigration(user.WalletAddress, walletMigration)
		if user.WalletAddress != normalizedWallet || walletMigrated {
			migrated = true
		}
		user.WalletAddress = normalizedWallet
		user.GitHubUsername = normalizeGitHubUsername(user.GitHubUsername)
		s.users[user.ID] = user
	}
	for _, wallet := range state.Wallets {
		if wallet == nil {
			continue
		}
		if normalizePersistedWalletWithMigration(wallet, walletMigration) {
			migrated = true
		}
		if !validWalletAddress(wallet.Address) {
			continue
		}
		if existing := s.wallets[wallet.Address]; existing != nil {
			if existing.LegacyAddress == "" && wallet.LegacyAddress != "" {
				existing.LegacyAddress = wallet.LegacyAddress
				migrated = true
			}
			if existing.Chain == "" {
				existing.Chain = walletChainSolana
				migrated = true
			}
			continue
		}
		s.wallets[wallet.Address] = wallet
	}
	for _, user := range s.users {
		if user == nil {
			continue
		}
		address := normalizeWalletAddress(user.WalletAddress)
		if !validWalletAddress(address) {
			continue
		}
		if _, exists := s.wallets[address]; exists {
			continue
		}
		s.wallets[address] = &Wallet{
			Address:     address,
			Chain:       walletChainSolana,
			OwnerUserID: user.ID,
			CreatedAt:   user.CreatedAt,
		}
		migrated = true
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
	for _, intent := range state.PaymentOrders {
		if intent == nil || strings.TrimSpace(intent.OrderID) == "" {
			continue
		}
		intentCopy := *intent
		intentCopy.OrderID = strings.TrimSpace(intentCopy.OrderID)
		intentCopy.Provider = strings.ToLower(strings.TrimSpace(intentCopy.Provider))
		if intentCopy.Provider == "" {
			intentCopy.Provider = "paypal"
			migrated = true
		}
		flow, err := validatePaymentOrderFlow(intentCopy.Flow)
		if err != nil {
			continue
		}
		if intentCopy.Flow != flow {
			intentCopy.Flow = flow
			migrated = true
		}
		if strings.TrimSpace(intentCopy.Currency) == "" {
			intentCopy.Currency = "USD"
			migrated = true
		}
		intentCopy.Status = normalizePaymentOrderStatus(intentCopy.Status)
		if intentCopy.CreatedAt.IsZero() {
			intentCopy.CreatedAt = time.Now().UTC()
			migrated = true
		}
		if intentCopy.UpdatedAt.IsZero() {
			intentCopy.UpdatedAt = intentCopy.CreatedAt
			migrated = true
		}
		intentCopy.CapturedAt = cloneTimePtr(intent.CapturedAt)
		s.paymentOrders[intentCopy.OrderID] = &intentCopy
	}

	// Test settings
	s.testSettingsConfig = TestSettingsConfig{}
	if state.TestSettingsConfig != nil {
		s.testSettingsConfig = *state.TestSettingsConfig
	}
	s.testSettingsEntries = map[string]*TestSettingsEntry{}
	for _, entry := range state.TestSettingsEntries {
		if entry == nil || entry.ID == "" {
			continue
		}
		s.testSettingsEntries[entry.ID] = entry
	}

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
		NextID:              s.nextID,
		Projects:            make([]*Project, 0, len(s.projects)),
		Tasks:               make([]*Task, 0, len(s.tasks)),
		Users:               make([]*User, 0, len(s.users)),
		Wallets:             make([]*Wallet, 0, len(s.wallets)),
		Sessions:            make([]*Session, 0, len(s.sessions)),
		Notifications:       make([]*Notification, 0, len(s.notifications)),
		Attachments:         make([]*Attachment, 0, len(s.attachments)),
		SSLReviews:          make([]*SSLReviewStatus, 0, len(s.sslReviews)),
		GeminiAPIKeys:       make([]*GeminiAPIKey, 0, len(s.geminiAPIKeys)),
		GeminiWebhookLogs:   make([]*GeminiWebhookLog, 0, len(s.geminiWebhookLogs)),
		AdminSettings:       cloneAdminSettings(s.adminSettings),
		TestSettingsConfig:  &s.testSettingsConfig,
		TestSettingsEntries: make([]*TestSettingsEntry, 0, len(s.testSettingsEntries)),
		PaymentOrders:       make([]*PaymentOrderIntent, 0, len(s.paymentOrders)),
		Ledger:              s.ledger,
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
	for _, entry := range s.testSettingsEntries {
		entryCopy := *entry
		state.TestSettingsEntries = append(state.TestSettingsEntries, &entryCopy)
	}
	for _, intent := range s.paymentOrders {
		state.PaymentOrders = append(state.PaymentOrders, clonePaymentOrderIntent(intent))
	}
	sort.Slice(state.PaymentOrders, func(i, j int) bool {
		return state.PaymentOrders[i].CreatedAt.Before(state.PaymentOrders[j].CreatedAt)
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

func marketplaceBountyRow(project *Project, task *Task) *MarketplaceBounty {
	claimID := marketplaceBountyID(project.ID, task.IssueNumber)
	row := &MarketplaceBounty{
		ID:                 claimID,
		ClaimID:            claimID,
		ProjectID:          project.ID,
		ProjectTitle:       marketplaceProjectTitle(project),
		IssueNumber:        task.IssueNumber,
		Title:              task.Title,
		Acceptance:         compactText(task.Acceptance),
		RewardCents:        task.RewardCents,
		EstimatedHours:     marketplaceEstimatedHours(task),
		RequiredWorkerKind: task.RequiredWorkerKind,
		SuggestedAgentType: task.SuggestedAgentType,
		BountyType:         task.BountyType,
		EvidenceRequired:   publicTaskEvidenceRequiredForTask(task),
		SourceRepository:   marketplacePublicRepoURL(projectSourceRepoURL(project)),
		IssueURL:           marketplacePublicRepoURL(task.IssueURL),
		ClaimEndpoint:      "/api/tasks/" + claimID + "/claim",
		CreatedAt:          task.CreatedAt,
	}
	if packet := proposalPacketForTask(project, task); packet != nil {
		row.ProposalEndpoint = packet.ProposalEndpoint
		row.ProposalPacket = packet
	}
	return row
}

func marketplaceEstimatedHours(task *Task) float64 {
	if task == nil {
		return 0
	}
	if match := estimatedEffortPattern.FindStringSubmatch(task.Acceptance); len(match) == 2 {
		if value, err := strconv.ParseFloat(match[1], 64); err == nil && value > 0 {
			return roundHalfHour(value)
		}
	}
	return estimatedHoursFromReward(task.RewardCents)
}

func estimatedHoursFromReward(rewardCents int64) float64 {
	if rewardCents <= 0 {
		return 0
	}
	hours := float64(rewardCents) / 10000
	if hours < 1 {
		hours = 1
	}
	if hours > 80 {
		hours = 80
	}
	return roundHalfHour(hours)
}

func marketplaceBountyID(projectID string, issueNumber int) string {
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		projectID = "project"
	}
	if issueNumber > 0 {
		return fmt.Sprintf("%s:%d", projectID, issueNumber)
	}
	return projectID + ":bounty"
}

func marketplaceProjectTitle(project *Project) string {
	if project == nil {
		return "MergeOS project"
	}
	if title := strings.TrimSpace(project.Title); title != "" {
		return title
	}
	return "MergeOS project"
}

func workerIdentitySets(user *User) (map[string]bool, map[string]bool) {
	workerIDs := map[string]bool{}
	rewardAccounts := map[string]bool{}
	addWorker := func(value string) {
		value = workerIdentityKey(value)
		if value != "" {
			workerIDs[value] = true
		}
	}
	addReward := func(value string) {
		value = rewardAccountKey(value)
		if value != "" {
			rewardAccounts[value] = true
		}
	}

	if wallet := normalizeWalletAddress(user.WalletAddress); wallet != "" {
		addWorker(wallet)
		addWorker(walletAccount(wallet))
		addReward(walletAccount(wallet))
	}
	if github := normalizeGitHubUsername(user.GitHubUsername); github != "" {
		addWorker("github:" + github)
		addReward(githubWorkerAccount(github))
	}
	addWorker(user.ID)
	return workerIDs, rewardAccounts
}

func workerIdentityKey(value string) string {
	value = strings.TrimSpace(normalizeWorkerID(value))
	if value == "" {
		return ""
	}
	if validWalletAddress(value) {
		return walletAccount(value)
	}
	if strings.HasPrefix(strings.ToLower(value), "wallet:") {
		address := normalizeWalletAddress(value)
		if validWalletAddress(address) {
			return walletAccount(address)
		}
	}
	return strings.ToLower(value)
}

func rewardAccountKey(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if validWalletAddress(value) {
		return walletAccount(value)
	}
	if strings.HasPrefix(strings.ToLower(value), "wallet:") {
		address := normalizeWalletAddress(value)
		if validWalletAddress(address) {
			return walletAccount(address)
		}
	}
	return strings.ToLower(value)
}

func workerIdentityHints(user *User) []WorkerIdentityHint {
	wallet := normalizeWalletAddress(user.WalletAddress)
	github := normalizeGitHubUsername(user.GitHubUsername)
	githubValue := ""
	if github != "" {
		githubValue = githubWorkerAccount(github)
	}
	return []WorkerIdentityHint{
		{Label: "Solana MRG wallet", Value: wallet, Ready: wallet != ""},
		{Label: "GitHub", Value: githubValue, Ready: github != ""},
	}
}

func workerClaimedTaskRow(project *Project, task *Task) WorkerClaimedTask {
	status := string(task.Status)
	if status == "" || status == string(TaskOpen) {
		status = "claimed"
	}
	row := WorkerClaimedTask{
		ID:                marketplaceBountyID(task.ProjectID, task.IssueNumber),
		ProjectID:         task.ProjectID,
		ProjectTitle:      marketplaceProjectTitle(project),
		IssueNumber:       task.IssueNumber,
		Title:             task.Title,
		Acceptance:        compactText(task.Acceptance),
		RewardCents:       task.RewardCents,
		WorkerKind:        task.WorkerKind,
		AgentType:         task.AgentType,
		Status:            status,
		ProofHash:         task.ProofHash,
		IssueURL:          marketplacePublicRepoURL(task.IssueURL),
		PullRequestURL:    task.PullRequestURL,
		ReviewEvidenceURL: task.ReviewEvidenceURL,
		ReviewNotes:       task.ReviewNotes,
		AcceptedAt:        task.AcceptedAt,
		SubmittedAt:       task.SubmittedAt,
	}
	if taskIsReleased(task) {
		row.LedgerProofURL = "/api/public/ledger/proof"
	}
	return row
}

func taskHasWorker(task *Task) bool {
	return task != nil && task.Status != TaskOpen && strings.TrimSpace(task.WorkerID) != ""
}

func taskIsOpenForClaim(task *Task) bool {
	return task != nil && task.Status == TaskOpen
}

func taskIsReleased(task *Task) bool {
	return task != nil && task.Status == TaskAccepted
}

func publicWorkerRewardReference(reference string) string {
	if pullReference := publicPullLedgerReference(reference); pullReference != "" {
		return pullReference
	}
	taskID := ledgerReferenceTaskID(reference)
	if taskID != "" {
		return "task"
	}
	return sanitizeLedgerReferenceValue(reference)
}

func workerProposalRows(projects map[string]*Project, tasks map[string]*Task, user *User) []WorkerProposal {
	rows := []WorkerProposal{}
	for _, task := range tasks {
		if !taskIsOpenForClaim(task) {
			continue
		}
		project := projects[task.ProjectID]
		claimID := marketplaceBountyID(task.ProjectID, task.IssueNumber)
		packet := proposalPacketForTask(project, task)
		row := WorkerProposal{
			ID:                 claimID,
			ClaimID:            claimID,
			ProjectID:          task.ProjectID,
			ProjectTitle:       marketplaceProjectTitle(project),
			IssueNumber:        task.IssueNumber,
			Title:              task.Title,
			Acceptance:         compactText(task.Acceptance),
			RewardCents:        task.RewardCents,
			EstimatedHours:     marketplaceEstimatedHours(task),
			RequiredWorkerKind: task.RequiredWorkerKind,
			SuggestedAgentType: task.SuggestedAgentType,
			MatchScore:         workerProposalMatchScore(task, user),
			MatchReasons:       workerProposalMatchReasons(task, user),
			EvidenceRequired:   publicTaskEvidenceRequiredForTask(task),
			IssueURL:           marketplacePublicRepoURL(task.IssueURL),
			CreatedAt:          task.CreatedAt,
		}
		if packet != nil {
			row.ProposalEndpoint = packet.ProposalEndpoint
			row.ClaimPacket = packet
		}
		rows = append(rows, row)
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].MatchScore == rows[j].MatchScore {
			if rows[i].RewardCents == rows[j].RewardCents {
				return rows[i].CreatedAt.After(rows[j].CreatedAt)
			}
			return rows[i].RewardCents > rows[j].RewardCents
		}
		return rows[i].MatchScore > rows[j].MatchScore
	})
	if len(rows) > 8 {
		return rows[:8]
	}
	return rows
}

func proposalPacketForTask(project *Project, task *Task) *ProposalPacket {
	if task == nil || task.RequiredWorkerKind == WorkerAgent {
		return nil
	}
	claimID := marketplaceBountyID(task.ProjectID, task.IssueNumber)
	estimatedHours := marketplaceEstimatedHours(task)
	bidCents := task.RewardCents
	if bidCents <= 0 {
		bidCents = 5000
	}
	if estimatedHours <= 0 {
		estimatedHours = estimatedHoursFromReward(bidCents)
	}
	packet := &ProposalPacket{
		CanClaim:         taskIsOpenForClaim(task),
		Status:           "ready",
		ProposalEndpoint: "/api/proposals",
		ContextURLs: map[string]string{
			"task_protocol": "/api/public/protocol/tasks?task_id=" + claimID,
			"marketplace":   "/api/public/marketplace",
		},
		Runbook: []AgentRunbookStep{
			{Step: 1, Action: "read_task", Label: "Read public task protocol and acceptance criteria", Method: "GET", Endpoint: "/api/public/protocol/tasks?task_id=" + claimID},
			{Step: 2, Action: "prepare_proposal", Label: "Attach bid, availability, evidence plan, and worker identity", Method: "POST", Endpoint: "/api/proposals"},
			{Step: 3, Action: "wait_customer_review", Label: "Customer or admin reviews the proposal before claim", Method: "GET", Endpoint: "/api/workers/me"},
		},
		Payload: CreateProposalRequest{
			TaskID:         claimID,
			CoverLetter:    proposalPacketCoverLetter(project, task),
			BidCents:       bidCents,
			EstimatedHours: estimatedHours,
			Availability:   "Available after customer approval",
		},
		EvidenceChecklist: publicTaskEvidenceRequiredForTask(task),
		OutputContracts:   proposalPacketOutputContracts(claimID),
		Warnings: []string{
			"Login and link a GitHub or Solana wallet identity before submitting.",
			"Do not include private customer data in proposal proof.",
		},
	}
	if !packet.CanClaim {
		packet.Status = "unavailable"
	}
	return packet
}

func proposalPacketOutputContracts(claimID string) []AgentOutputContract {
	taskProtocolURL := "/api/public/protocol/tasks?task_id=" + claimID
	return []AgentOutputContract{
		{Action: "submit_proposal", ArtifactKind: "worker_proposal", OutputEndpoint: "/api/proposals", OutputProtocol: "mergeos.proposal.v1", OutputProtocolURL: "/protocol/proposal.v1.schema.json", PublicURL: "/api/public/live-feed"},
		{Action: "notify_customer", ArtifactKind: "proposal_notification", OutputEndpoint: "/api/proposals", OutputProtocol: "mergeos.event.v1", OutputProtocolURL: "/protocol/event.v1.schema.json", PublicURL: "/api/public/live-feed"},
		{Action: "read_task", ArtifactKind: "task_protocol", OutputEndpoint: taskProtocolURL, OutputProtocol: "mergeos.task.v1", OutputProtocolURL: "/protocol/task.v1.schema.json", PublicURL: taskProtocolURL},
	}
}

func proposalPacketCoverLetter(project *Project, task *Task) string {
	title := "this bounty"
	projectTitle := "this MergeOS project"
	acceptance := "the published acceptance criteria"
	if task != nil {
		if value := strings.TrimSpace(task.Title); value != "" {
			title = value
		}
		if value := strings.TrimSpace(task.Acceptance); value != "" {
			acceptance = compactText(value)
		}
	}
	if project != nil {
		projectTitle = marketplaceProjectTitle(project)
	}
	return proposalText(fmt.Sprintf("I can deliver %s for %s. Scope: %s I will attach PR evidence, tests, and release notes through MergeOS.", title, projectTitle, acceptance), 2000)
}

func workerProposalMatchScore(task *Task, user *User) int {
	score := 54
	if normalizeGitHubUsername(user.GitHubUsername) != "" {
		score += 18
	}
	if normalizeWalletAddress(user.WalletAddress) != "" {
		score += 12
	}
	if task.RequiredWorkerKind == WorkerHuman {
		score += 8
	}
	if strings.TrimSpace(task.SuggestedAgentType) != "" {
		score += 6
	}
	if score > 98 {
		return 98
	}
	return score
}

func workerProposalMatchReasons(task *Task, user *User) []string {
	reasons := []string{"open bounty"}
	if normalizeGitHubUsername(user.GitHubUsername) != "" {
		reasons = append(reasons, "github identity linked")
	}
	if normalizeWalletAddress(user.WalletAddress) != "" {
		reasons = append(reasons, "wallet ready")
	}

	switch task.RequiredWorkerKind {
	case WorkerHuman:
		reasons = append(reasons, "human contributor lane")
	case WorkerAgent:
		reasons = append(reasons, workerProposalAgentReason("agent lane", task.SuggestedAgentType))
	case WorkerHybrid:
		reasons = append(reasons, workerProposalAgentReason("hybrid lane", task.SuggestedAgentType))
	}
	if marketplaceEstimatedHours(task) > 0 {
		reasons = append(reasons, "effort estimated")
	}
	return cleanStrings(reasons)
}

func workerProposalAgentReason(prefix, agentType string) string {
	agentType = strings.TrimSpace(agentType)
	if agentType == "" {
		return prefix
	}
	return prefix + ": " + agentType
}

func workerReputationScore(claimedTasks int, rewardCents int64, rewardRows int, hasGitHub, hasWallet bool) int {
	score := 45
	if hasGitHub {
		score += 15
	}
	if hasWallet {
		score += 15
	}
	score += claimedTasks * 8
	score += rewardRows * 2
	score += int(rewardCents / 50000)
	if score > 100 {
		return 100
	}
	return score
}

func workerReputationAudit(audit WorkerReputationAudit) WorkerReputationAudit {
	audit.WorkerID = normalizeWorkerID(audit.WorkerID)
	if strings.TrimSpace(audit.Name) == "" {
		audit.Name = marketplaceWorkerName(audit.WorkerID, audit.AgentType)
	}
	audit.Score = workerReputationScore(audit.CompletedTaskCount, audit.RewardCents, audit.RewardRowCount, audit.HasGitHub, audit.HasWallet)
	if audit.DuplicateIdentityCount > 0 {
		audit.Score -= 25
	}
	if audit.CompletedTaskCount == 0 {
		audit.Score -= 5
	}
	if audit.CompletedTaskCount > 0 && audit.RewardRowCount == 0 {
		audit.Score -= 10
	}
	if audit.Score < 0 {
		audit.Score = 0
	}
	if audit.Score > 100 {
		audit.Score = 100
	}
	audit.Level = workerReputationLevel(audit.Score)
	audit.RiskLevel = workerRiskLevel(audit)
	audit.Flags = workerReputationFlags(audit)
	return audit
}

func workerReputationLevel(score int) string {
	switch {
	case score >= 85:
		return "elite"
	case score >= 70:
		return "trusted"
	case score >= 55:
		return "building"
	default:
		return "new"
	}
}

func workerRiskLevel(audit WorkerReputationAudit) string {
	if audit.DuplicateIdentityCount > 0 || audit.Score < 45 || (audit.CompletedTaskCount > 0 && audit.RewardRowCount == 0) {
		return "high"
	}
	if audit.Score < 70 || !audit.HasGitHub || !audit.HasWallet {
		return "medium"
	}
	return "low"
}

func workerRiskRank(level string) int {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "high":
		return 3
	case "medium":
		return 2
	case "low":
		return 1
	default:
		return 0
	}
}

func workerReputationFlags(audit WorkerReputationAudit) []string {
	flags := []string{}
	if !audit.HasGitHub {
		flags = append(flags, "missing_github_identity")
	}
	if !audit.HasWallet {
		flags = append(flags, "missing_wallet_identity")
	}
	if audit.DuplicateIdentityCount > 0 {
		flags = append(flags, "duplicate_identity")
	}
	if audit.CompletedTaskCount == 0 {
		flags = append(flags, "no_completed_tasks")
	}
	if audit.CompletedTaskCount > 0 && audit.RewardRowCount == 0 {
		flags = append(flags, "missing_reward_ledger")
	}
	return flags
}

func workerDashboardID(profile WorkerProfile) string {
	if profile.GitHubUsername != "" {
		return githubWorkerAccount(profile.GitHubUsername)
	}
	if profile.WalletAddress != "" {
		return walletAccount(profile.WalletAddress)
	}
	return profile.UserID
}

func workerReputationKey(workerID, agentType string) string {
	workerID = workerIdentityKey(workerID)
	agentType = strings.ToLower(strings.TrimSpace(agentType))
	return workerID + "|" + agentType
}

func workerIDHasGitHub(workerID string) bool {
	workerID = workerIdentityKey(workerID)
	return strings.HasPrefix(workerID, "github:") && normalizeGitHubUsername(workerID) != ""
}

func workerIDHasWallet(workerID string) bool {
	workerID = strings.TrimSpace(normalizeWorkerID(workerID))
	return validWalletAddress(workerID)
}

func nonZeroTimePointer(value time.Time) *time.Time {
	if value.IsZero() {
		return nil
	}
	copyValue := value
	return &copyValue
}

func (s *Store) workerReputationAuditForUserLocked(user *User) WorkerReputationAudit {
	workerID := ""
	wallet := normalizeWalletAddress(user.WalletAddress)
	github := normalizeGitHubUsername(user.GitHubUsername)
	if github != "" {
		workerID = githubWorkerAccount(github)
	} else if wallet != "" {
		workerID = walletAccount(wallet)
	}
	if workerID == "" {
		return WorkerReputationAudit{}
	}

	workerIDs, rewardAccounts := workerIdentitySets(user)
	audit := WorkerReputationAudit{
		WorkerID:               workerID,
		Name:                   user.Name,
		Kind:                   WorkerHuman,
		HasGitHub:              github != "",
		HasWallet:              wallet != "",
		DuplicateIdentityCount: s.duplicateIdentityCountLocked(user),
	}
	for _, task := range s.tasks {
		if !taskIsReleased(task) || !workerIDs[workerIdentityKey(task.WorkerID)] {
			continue
		}
		audit.CompletedTaskCount++
		audit.RewardCents += task.RewardCents
		if task.AcceptedAt != nil && (audit.LastPaidAt == nil || task.AcceptedAt.After(*audit.LastPaidAt)) {
			lastPaidAt := *task.AcceptedAt
			audit.LastPaidAt = &lastPaidAt
		}
	}
	for _, entry := range s.ledger {
		if entry.Type != "task_payment" && entry.Type != "manual_credit" {
			continue
		}
		if rewardAccounts[rewardAccountKey(entry.ToAccount)] {
			audit.RewardRowCount++
		}
	}
	return workerReputationAudit(audit)
}

func (s *Store) workerIdentitySignalsForWorkerIDLocked(workerID string) (bool, bool, int) {
	hasGitHub := workerIDHasGitHub(workerID)
	hasWallet := workerIDHasWallet(workerID)
	duplicateIdentityCount := 0
	if hasGitHub {
		username := normalizeGitHubUsername(workerID)
		if user := s.userByGitHubLocked("", username); user != nil {
			hasWallet = hasWallet || normalizeWalletAddress(user.WalletAddress) != ""
			duplicateIdentityCount = s.duplicateIdentityCountLocked(user)
		} else if wallet := s.walletByGitHubLocked(username); wallet != nil {
			hasWallet = hasWallet || normalizeWalletAddress(wallet.Address) != ""
		}
	}
	return hasGitHub, hasWallet, duplicateIdentityCount
}

func (s *Store) duplicateIdentityCountLocked(user *User) int {
	count := 0
	wallet := normalizeWalletAddress(user.WalletAddress)
	github := normalizeGitHubUsername(user.GitHubUsername)
	for id, other := range s.users {
		if other == nil || id == user.ID {
			continue
		}
		if wallet != "" && normalizeWalletAddress(other.WalletAddress) == wallet {
			count++
		}
		if github != "" && normalizeGitHubUsername(other.GitHubUsername) == github {
			count++
		}
	}
	return count
}

func workerReputationRows(response WorkerDashboardResponse) []WorkerReputation {
	identity := "Incomplete"
	if response.Profile.GitHubUsername != "" && response.Profile.WalletAddress != "" {
		identity = "Verified"
	}
	riskTone := "green"
	switch response.ReputationAudit.RiskLevel {
	case "high":
		riskTone = "red"
	case "medium":
		riskTone = "amber"
	}
	lastPaid := "No payouts yet"
	if response.Stats.LastPaidAt != nil {
		lastPaid = response.Stats.LastPaidAt.Format("2006-01-02")
	}
	return []WorkerReputation{
		{Label: "Identity", Value: identity, Tone: "green"},
		{Label: "Level", Value: marketplaceTitle(response.ReputationAudit.Level), Tone: "blue"},
		{Label: "Risk", Value: marketplaceTitle(response.ReputationAudit.RiskLevel), Tone: riskTone},
		{Label: "Completed tasks", Value: strconv.Itoa(response.Stats.ClaimedTaskCount), Tone: "blue"},
		{Label: "Rewards", Value: formatTokenAmount(response.Stats.RewardCents), Tone: "green"},
		{Label: "Last payout", Value: lastPaid, Tone: "amber"},
	}
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
	for projectID := range projectIDs {
		if ledgerEntryReferencesID(entry, projectID) {
			return true
		}
	}
	for taskID := range taskIDs {
		if ledgerEntryReferencesID(entry, taskID) {
			return true
		}
	}
	return false
}

func publicLedgerScope(entry LedgerEntry, projectIDs map[string]bool, taskProjectIDs map[string]string) (string, string) {
	for projectID := range projectIDs {
		if ledgerEntryReferencesID(entry, projectID) {
			return projectID, ""
		}
	}
	for taskID, projectID := range taskProjectIDs {
		if ledgerEntryReferencesID(entry, taskID) {
			return projectID, taskID
		}
	}
	return "", ""
}

func ledgerEntryReferencesID(entry LedgerEntry, id string) bool {
	return ledgerValueReferencesID(entry.FromAccount, id) ||
		ledgerValueReferencesID(entry.ToAccount, id) ||
		ledgerValueReferencesID(entry.Reference, id)
}

func ledgerValueReferencesID(value, id string) bool {
	value = strings.TrimSpace(value)
	id = strings.TrimSpace(id)
	if value == "" || id == "" {
		return false
	}
	if value == id {
		return true
	}
	for _, fieldValue := range splitLedgerReference(value) {
		if strings.TrimSpace(fieldValue) == id {
			return true
		}
	}
	for _, token := range strings.FieldsFunc(value, ledgerReferenceTokenSeparator) {
		if token == id {
			return true
		}
	}
	return false
}

func ledgerReferenceTokenSeparator(r rune) bool {
	switch r {
	case ':', ';', '|', '/', '?', '&', '=', '#', ' ', '\t', '\r', '\n':
		return true
	default:
		return false
	}
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
	case strings.HasPrefix(account, "airdrop:"):
		return "airdrop:pool"
	case strings.HasPrefix(account, "presale:"):
		return "presale:reserve"
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

func publicLedgerReference(projectID, taskID string, sequence int, reference string) string {
	if pullReference := publicPullLedgerReference(reference); pullReference != "" {
		return pullReference
	}
	if walletMigrationReference := publicWalletMigrationLedgerReference(reference); walletMigrationReference != "" {
		return walletMigrationReference
	}
	if projectID == "" {
		return fmt.Sprintf("ledger:%d", sequence)
	}
	if taskID != "" {
		return fmt.Sprintf("project:%s;task:%s", projectID, taskID)
	}
	return "project:" + projectID
}

func publicWalletMigrationLedgerReference(reference string) string {
	fields := splitLedgerReference(reference)
	migrationID := sanitizeLedgerReferenceValue(fields["wallet_migration"])
	legacyChain := sanitizeLedgerReferenceValue(fields["legacy_chain"])
	legacyHash := sanitizeLedgerReferenceValue(fields["legacy_hash"])
	target := sanitizeLedgerReferenceValue(fields["target"])
	instruction := sanitizeLedgerReferenceValue(fields["instruction"])
	if migrationID == "" || legacyHash == "" || target == "" {
		return ""
	}
	parts := []string{
		"wallet_migration:" + migrationID,
		"legacy_hash:" + legacyHash,
		"target:" + target,
	}
	if legacyChain != "" {
		parts = append(parts, "legacy_chain:"+legacyChain)
	}
	if instruction != "" {
		parts = append(parts, "instruction:"+instruction)
	}
	return strings.Join(parts, ";")
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
