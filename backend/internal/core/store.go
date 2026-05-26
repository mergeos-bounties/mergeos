package core

import (
	"context"
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

type Store struct {
	mu       sync.RWMutex
	cfg      Config
	payments *PaymentManager
	repos    RepoFactory
	emailer  *EmailSender
	storage  statePersistence

	nextID        int
	projects      map[string]*Project
	tasks         map[string]*Task
	users         map[string]*User
	sessions      map[string]*Session
	notifications map[string]*Notification
	attachments   map[string]*Attachment
	sslReviews    map[string]*SSLReviewStatus
	ledger        []LedgerEntry
}

type persistedState struct {
	NextID        int                `json:"next_id"`
	Projects      []*Project         `json:"projects"`
	Tasks         []*Task            `json:"tasks"`
	Users         []*User            `json:"users"`
	Sessions      []*Session         `json:"sessions"`
	Notifications []*Notification    `json:"notifications"`
	Attachments   []*Attachment      `json:"attachments"`
	SSLReviews    []*SSLReviewStatus `json:"ssl_reviews"`
	Ledger        []LedgerEntry      `json:"ledger"`
}

type statePersistence interface {
	Load(ctx context.Context) (persistedState, bool, error)
	Save(ctx context.Context, state persistedState) error
	Close() error
}

func NewStore(cfg Config, payments *PaymentManager, repos RepoFactory, emailer *EmailSender) (*Store, error) {
	store := &Store{
		cfg:           cfg,
		payments:      payments,
		repos:         repos,
		emailer:       emailer,
		nextID:        1,
		projects:      map[string]*Project{},
		tasks:         map[string]*Task{},
		users:         map[string]*User{},
		sessions:      map[string]*Session{},
		notifications: map[string]*Notification{},
		attachments:   map[string]*Attachment{},
		sslReviews:    map[string]*SSLReviewStatus{},
		ledger:        []LedgerEntry{},
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
	return store, nil
}

func (s *Store) Close() error {
	if s.storage == nil {
		return nil
	}
	return s.storage.Close()
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
		return nil, errors.New("budget must be at least 100 USD")
	}
	if req.PaymentMethod != PaymentPayPal && req.PaymentMethod != PaymentCrypto {
		return nil, errors.New("payment method must be paypal or crypto")
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
	project.Tasks = s.splitProjectTasks(project)

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
			task.IssueURL = issue.URL
		}
	}

	s.projects[projectID] = project
	s.addLedger("payment_verified", "payment:"+verification.Provider, "client:"+projectID, req.BudgetCents, verification.Reference)
	s.addLedger("token_mint", "issuer:mergeos", "client:"+projectID, req.BudgetCents, "mint:"+projectID)
	s.addLedger("platform_fee", "client:"+projectID, "treasury:mergeos", fee, "fee:"+projectID)
	s.addLedger("project_reserve", "client:"+projectID, "reserve:project:"+projectID, workPool, "repo:"+project.BountyRepoName)

	for _, task := range project.Tasks {
		s.tasks[task.ID] = task
		reference := fmt.Sprintf("%s/issues/%d", project.BountyRepoName, task.IssueNumber)
		if task.IssueURL != "" {
			reference = task.IssueURL
		}
		s.addLedger("task_reserve", "reserve:project:"+projectID, "reserve:task:"+task.ID, task.RewardCents, reference)
	}
	subject := "MergeOS project funded: " + project.Title
	body := fmt.Sprintf("Hi %s,\n\nYour project %q is funded. MergeOS created bounty repo %s and split it into %d payable tasks.\n\nBudget: %s USD\nWork pool: %s MERGE\nAttachments: %d\n\nWe will notify you as tasks are accepted.", project.ClientName, project.Title, project.BountyRepoName, len(project.Tasks), centsToPayPalValue(project.BudgetCents), centsToPayPalValue(project.WorkPoolCents), len(project.Attachments))
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
	return tasks
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

func (s *Store) ListUsers() []AdminUser {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows := make([]AdminUser, 0, len(s.users))
	byID := map[string]int{}
	for _, user := range s.users {
		row := AdminUser{PublicUser: publicUser(user)}
		byID[user.ID] = len(rows)
		rows = append(rows, row)
	}
	for _, project := range s.projects {
		index, ok := byID[project.ClientUserID]
		if !ok {
			continue
		}
		rows[index].ProjectCount++
		rows[index].TotalBudgetCents += project.BudgetCents
		if rows[index].LastProjectAt == nil || project.CreatedAt.After(*rows[index].LastProjectAt) {
			createdAt := project.CreatedAt
			rows[index].LastProjectAt = &createdAt
		}
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Role != rows[j].Role {
			return rows[i].Role == RoleAdmin
		}
		return rows[i].CreatedAt.After(rows[j].CreatedAt)
	})
	return rows
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
	s.mu.Lock()
	defer s.mu.Unlock()

	task, ok := s.tasks[taskID]
	if !ok {
		return nil, errors.New("task not found")
	}
	if task.Status == TaskAccepted {
		return nil, errors.New("task already accepted")
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

	now := time.Now().UTC()
	entry := s.addLedger("task_payment", "reserve:task:"+task.ID, "worker:"+strings.TrimSpace(req.WorkerID), task.RewardCents, "task:"+task.ID)
	task.Status = TaskAccepted
	task.WorkerKind = req.WorkerKind
	task.WorkerID = strings.TrimSpace(req.WorkerID)
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
		body := fmt.Sprintf("Task #%d was accepted and paid to %s. Proof hash: %s", task.IssueNumber, task.WorkerID, task.ProofHash)
		status := s.emailer.Send(project.ClientEmail, subject, body)
		s.addNotificationLocked(project.ClientUserID, project.ID, "email", subject, body, status)
	}
	if err := s.saveLocked(); err != nil {
		return nil, err
	}

	copyTask := *task
	return &copyTask, nil
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
		{"Checkout, token and proof ledger", "PayPal/crypto verification, MERGE mint, reserves, fees and proof ledger are testable through API.", 22, WorkerAgent, "go-ledger-agent"},
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
			CreatedAt:          time.Now().UTC(),
		}
		tasks = append(tasks, task)
	}
	return tasks
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
	payload := fmt.Sprintf("%d|%s|%s|%s|%d|%s|%s|%s", entry.Sequence, entry.Type, entry.FromAccount, entry.ToAccount, entry.AmountCents, entry.Reference, entry.PreviousHash, entry.CreatedAt.Format(time.RFC3339Nano))
	sum := sha256.Sum256([]byte(payload))
	entry.EntryHash = hex.EncodeToString(sum[:])
	s.ledger = append(s.ledger, entry)
	return entry
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
			s.applyState(state)
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
	s.applyState(state)
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

func (s *Store) applyState(state persistedState) {
	if state.NextID > 0 {
		s.nextID = state.NextID
	}
	s.ledger = state.Ledger
	s.projects = map[string]*Project{}
	s.tasks = map[string]*Task{}
	s.users = map[string]*User{}
	s.sessions = map[string]*Session{}
	s.notifications = map[string]*Notification{}
	s.attachments = map[string]*Attachment{}
	s.sslReviews = map[string]*SSLReviewStatus{}
	for _, project := range state.Projects {
		if project == nil || project.ID == "" {
			continue
		}
		s.projects[project.ID] = project
	}
	for _, task := range state.Tasks {
		if task == nil || task.ID == "" {
			continue
		}
		s.tasks[task.ID] = task
	}
	for _, user := range state.Users {
		if user == nil || user.ID == "" {
			continue
		}
		s.users[user.ID] = user
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
		NextID:        s.nextID,
		Projects:      make([]*Project, 0, len(s.projects)),
		Tasks:         make([]*Task, 0, len(s.tasks)),
		Users:         make([]*User, 0, len(s.users)),
		Sessions:      make([]*Session, 0, len(s.sessions)),
		Notifications: make([]*Notification, 0, len(s.notifications)),
		Attachments:   make([]*Attachment, 0, len(s.attachments)),
		SSLReviews:    make([]*SSLReviewStatus, 0, len(s.sslReviews)),
		Ledger:        s.ledger,
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
