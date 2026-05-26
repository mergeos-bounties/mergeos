package core

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCreateProjectCreatesLocalBountyRepoAndPersistsLedger(t *testing.T) {
	tempDir := t.TempDir()
	cfg := Config{
		TokenSymbol:       defaultTokenSymbol,
		StatePath:         filepath.Join(tempDir, "state.json"),
		PlatformFeeBps:    1000,
		DevPaymentEnabled: true,
		DevPaymentCode:    defaultDevPaymentCode,
		GitHubOwner:       defaultGitHubOwner,
		BountyRoot:        filepath.Join(tempDir, "bounties"),
		SMTPFrom:          "noreply@mergeos.local",
	}
	payments := NewPaymentManager(cfg)
	store, err := NewStore(cfg, payments, NewRepoFactory(cfg), NewEmailSender(cfg))
	if err != nil {
		t.Fatal(err)
	}
	auth, err := store.Register(RegisterRequest{
		Name:        "Test Client",
		CompanyName: "Test Co",
		Email:       "client@example.com",
		Password:    "password123",
	})
	if err != nil {
		t.Fatal(err)
	}

	project, err := store.CreateProject(context.Background(), auth.User.ID, CreateProjectRequest{
		Title:            "Agency website build",
		ClientName:       "Test Client",
		ClientEmail:      "client@example.com",
		Brief:            "Build a funded website bounty.",
		BudgetCents:      200000,
		PaymentMethod:    PaymentPayPal,
		PaymentReference: defaultDevPaymentCode,
	})
	if err != nil {
		t.Fatal(err)
	}

	if project.RepoProvider != "local-git" {
		t.Fatalf("repo provider = %q", project.RepoProvider)
	}
	if _, err := os.Stat(filepath.Join(project.RepoLocalPath, ".git")); err != nil {
		t.Fatalf("expected local git repo: %v", err)
	}
	if len(project.Tasks) != 6 {
		t.Fatalf("tasks = %d", len(project.Tasks))
	}
	ledger := store.ListLedger()
	if len(ledger) != 10 {
		t.Fatalf("ledger entries after create = %d", len(ledger))
	}
	expectedPayerAccount := "client:" + auth.User.ID + ":project:" + project.ID
	var mintEntry *LedgerEntry
	for i := range ledger {
		if ledger[i].Type == "token_mint" {
			mintEntry = &ledger[i]
			break
		}
	}
	if mintEntry == nil {
		t.Fatal("missing token_mint ledger entry")
	}
	if mintEntry.ToAccount != expectedPayerAccount || mintEntry.Reference != "mint:"+project.ID {
		t.Fatalf("token mint ledger entry not tied to payer/project: %#v", mintEntry)
	}
	if len(store.ListNotifications(auth.User.ID)) != 2 {
		t.Fatalf("notifications after create = %d", len(store.ListNotifications(auth.User.ID)))
	}

	accepted, err := store.AcceptTask(project.Tasks[0].ID, AcceptTaskRequest{
		WorkerKind: WorkerHuman,
		WorkerID:   "github:reviewer",
	})
	if err != nil {
		t.Fatal(err)
	}
	if accepted.Status != TaskAccepted || accepted.ProofHash == "" {
		t.Fatalf("accepted task missing status/proof: %#v", accepted)
	}

	reloaded, err := NewStore(cfg, payments, NewRepoFactory(cfg), NewEmailSender(cfg))
	if err != nil {
		t.Fatal(err)
	}
	if len(reloaded.ListProjects(auth.User.ID)) != 1 {
		t.Fatalf("reloaded project count = %d", len(reloaded.ListProjects(auth.User.ID)))
	}
	if len(reloaded.ListLedger()) != 11 {
		t.Fatalf("reloaded ledger entries = %d", len(reloaded.ListLedger()))
	}
}

func TestImportedRepoIssuesBecomeFundedTasks(t *testing.T) {
	store := &Store{nextID: 1}
	project := &Project{
		ID:            "prj_0001",
		Title:         "Fix repo issues",
		WorkPoolCents: 90000,
	}
	issues := []*ImportedRepoIssue{
		{
			Number:             3,
			Title:              "AI project evaluation for price suggestion",
			URL:                "https://github.com/mergeos-bounties/mergeos/issues/3",
			Score:              80,
			Complexity:         "high",
			EstimatedCents:     42000,
			RequiredWorkerKind: WorkerAgent,
			SuggestedAgentType: "backend-agent",
			Reasons:            []string{"open GitHub issue", "backend surface"},
		},
		{
			Number:             2,
			Title:              "Implement social login",
			URL:                "https://github.com/mergeos-bounties/mergeos/issues/2",
			Score:              60,
			Complexity:         "medium",
			EstimatedCents:     30000,
			RequiredWorkerKind: WorkerHybrid,
			SuggestedAgentType: "security-review-agent",
			Reasons:            []string{"open GitHub issue", "auth risk"},
		},
		{
			Number:             1,
			Title:              "Claim MRG Tokens for Bug Bounty Reports",
			URL:                "https://github.com/mergeos-bounties/mergeos/issues/1",
			Score:              30,
			Complexity:         "low",
			EstimatedCents:     18000,
			RequiredWorkerKind: WorkerHuman,
			Reasons:            []string{"open GitHub issue"},
		},
	}

	tasks := store.tasksFromImportedIssues(project, issues)
	if len(tasks) != len(issues) {
		t.Fatalf("tasks = %d", len(tasks))
	}
	if tasks[0].IssueNumber != 3 || tasks[0].IssueURL != issues[0].URL || !strings.Contains(tasks[0].Title, "Fix #3") {
		t.Fatalf("first task did not preserve source issue: %#v", tasks[0])
	}
	var total int64
	for _, task := range tasks {
		total += task.RewardCents
		if !strings.Contains(task.Acceptance, "Source issue: https://github.com/mergeos-bounties/mergeos/issues/") {
			t.Fatalf("task acceptance missing source issue: %#v", task)
		}
	}
	if total != project.WorkPoolCents {
		t.Fatalf("task rewards = %d, want %d", total, project.WorkPoolCents)
	}
	if tokenAmountFromCents(100000) != 100000 {
		t.Fatalf("token amount = %d, want 100000", tokenAmountFromCents(100000))
	}
}

func TestPublicMarketplaceRouteReturnsSanitizedLiveData(t *testing.T) {
	tempDir := t.TempDir()
	cfg := Config{
		TokenSymbol:       defaultTokenSymbol,
		StatePath:         filepath.Join(tempDir, "state.json"),
		PlatformFeeBps:    1000,
		DevPaymentEnabled: true,
		DevPaymentCode:    defaultDevPaymentCode,
		GitHubOwner:       defaultGitHubOwner,
		BountyRoot:        filepath.Join(tempDir, "bounties"),
		SMTPFrom:          "noreply@mergeos.local",
	}
	payments := NewPaymentManager(cfg)
	store, err := NewStore(cfg, payments, NewRepoFactory(cfg), NewEmailSender(cfg))
	if err != nil {
		t.Fatal(err)
	}
	auth, err := store.Register(RegisterRequest{
		Name:        "Marketplace Client",
		CompanyName: "Marketplace Co",
		Email:       "client@example.com",
		Password:    "password123",
	})
	if err != nil {
		t.Fatal(err)
	}

	project, err := store.CreateProject(context.Background(), auth.User.ID, CreateProjectRequest{
		Title:            "Customer portal rebuild",
		ClientName:       "Private Client",
		CompanyName:      "Marketplace Co",
		ClientEmail:      "client@example.com",
		Phone:            "+1 555 0101",
		SiteType:         "Web Development",
		PackageTier:      "Launch",
		Brief:            "Rebuild the customer portal with a responsive interface and proof ledger.",
		BudgetCents:      250000,
		PaymentMethod:    PaymentPayPal,
		PaymentReference: defaultDevPaymentCode,
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, task := range project.Tasks {
		if task.RequiredWorkerKind == WorkerHuman {
			if _, err := store.AcceptTask(task.ID, AcceptTaskRequest{
				WorkerKind: WorkerHuman,
				WorkerID:   "github:maya-dev",
			}); err != nil {
				t.Fatal(err)
			}
			break
		}
	}

	server := NewServer(cfg, store, payments)
	req := httptest.NewRequest(http.MethodGet, "/api/public/marketplace", nil)
	resp := httptest.NewRecorder()
	server.Routes().ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("marketplace status = %d, body = %s", resp.Code, resp.Body.String())
	}

	body := resp.Body.String()
	if strings.Contains(body, "client@example.com") || strings.Contains(body, "+1 555 0101") || strings.Contains(body, auth.User.ID) || strings.Contains(body, tempDir) {
		t.Fatalf("public marketplace leaked private customer data: %s", body)
	}

	var payload MarketplaceResponse
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Stats.ProjectCount != 1 || payload.Stats.OpenTaskCount == 0 || payload.Stats.TotalBudgetCents != 250000 {
		t.Fatalf("unexpected stats: %#v", payload.Stats)
	}
	if len(payload.Projects) != 1 {
		t.Fatalf("project count = %d", len(payload.Projects))
	}
	if payload.Projects[0].ClientDisplayName != "Marketplace Co" || len(payload.Projects[0].Tags) == 0 {
		t.Fatalf("project row missing public display data: %#v", payload.Projects[0])
	}
	if len(payload.Contributors) != 1 || payload.Contributors[0].EarnedCents == 0 {
		t.Fatalf("contributors missing real paid task data: %#v", payload.Contributors)
	}
	if len(payload.Agents) == 0 || payload.Agents[0].OpenTaskCount == 0 {
		t.Fatalf("agents missing real task demand: %#v", payload.Agents)
	}
}

func TestPublicLedgerRouteReturnsSanitizedLiveData(t *testing.T) {
	tempDir := t.TempDir()
	cfg := Config{
		TokenSymbol:       defaultTokenSymbol,
		StatePath:         filepath.Join(tempDir, "state.json"),
		PlatformFeeBps:    1000,
		DevPaymentEnabled: true,
		DevPaymentCode:    defaultDevPaymentCode,
		GitHubOwner:       defaultGitHubOwner,
		BountyRoot:        filepath.Join(tempDir, "bounties"),
		SMTPFrom:          "noreply@mergeos.local",
	}
	payments := NewPaymentManager(cfg)
	store, err := NewStore(cfg, payments, NewRepoFactory(cfg), NewEmailSender(cfg))
	if err != nil {
		t.Fatal(err)
	}
	auth, err := store.Register(RegisterRequest{
		Name:        "Ledger Client",
		CompanyName: "Ledger Co",
		Email:       "ledger@example.com",
		Password:    "password123",
	})
	if err != nil {
		t.Fatal(err)
	}
	project, err := store.CreateProject(context.Background(), auth.User.ID, CreateProjectRequest{
		Title:            "Public proof ledger",
		ClientName:       "Private Ledger Client",
		CompanyName:      "Ledger Co",
		ClientEmail:      "ledger@example.com",
		Phone:            "+1 555 0199",
		Brief:            "Create ledger entries that should be public without exposing customer data.",
		BudgetCents:      150000,
		PaymentMethod:    PaymentPayPal,
		PaymentReference: defaultDevPaymentCode,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.AcceptTask(project.Tasks[0].ID, AcceptTaskRequest{
		WorkerKind: WorkerHuman,
		WorkerID:   "github:private-worker",
	}); err != nil {
		t.Fatal(err)
	}

	server := NewServer(cfg, store, payments)
	req := httptest.NewRequest(http.MethodGet, "/api/public/ledger", nil)
	resp := httptest.NewRecorder()
	server.Routes().ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("public ledger status = %d, body = %s", resp.Code, resp.Body.String())
	}

	body := resp.Body.String()
	privateValues := []string{
		"ledger@example.com",
		"+1 555 0199",
		auth.User.ID,
		tempDir,
		"github:private-worker",
		defaultDevPaymentCode,
	}
	for _, value := range privateValues {
		if strings.Contains(body, value) {
			t.Fatalf("public ledger leaked private value %q: %s", value, body)
		}
	}

	var payload []LedgerEntry
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if len(payload) == 0 {
		t.Fatal("public ledger returned no entries")
	}
	foundProjectReference := false
	for _, entry := range payload {
		if strings.Contains(entry.FromAccount, "client:") || strings.Contains(entry.ToAccount, "client:") {
			t.Fatalf("public ledger leaked client account: %#v", entry)
		}
		if strings.Contains(entry.Reference, project.ID) {
			foundProjectReference = true
		}
	}
	if !foundProjectReference {
		t.Fatalf("public ledger did not preserve project reference: %#v", payload)
	}
}

func TestAdminAutoPromoteAndRoutes(t *testing.T) {
	tempDir := t.TempDir()
	cfg := Config{
		TokenSymbol:       defaultTokenSymbol,
		StatePath:         filepath.Join(tempDir, "state.json"),
		PlatformFeeBps:    1000,
		DevPaymentEnabled: true,
		DevPaymentCode:    defaultDevPaymentCode,
		GitHubOwner:       defaultGitHubOwner,
		BountyRoot:        filepath.Join(tempDir, "bounties"),
		SMTPFrom:          "noreply@mergeos.local",
		AdminAutoPromote:  true,
	}
	payments := NewPaymentManager(cfg)
	store, err := NewStore(cfg, payments, NewRepoFactory(cfg), NewEmailSender(cfg))
	if err != nil {
		t.Fatal(err)
	}
	adminAuth, err := store.Register(RegisterRequest{
		Name:     "Admin User",
		Email:    "admin@example.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatal(err)
	}
	if adminAuth.User.Role != RoleAdmin {
		t.Fatalf("first user role = %q", adminAuth.User.Role)
	}
	clientAuth, err := store.Register(RegisterRequest{
		Name:     "Client User",
		Email:    "client-two@example.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatal(err)
	}
	if clientAuth.User.Role != RoleClient {
		t.Fatalf("second user role = %q", clientAuth.User.Role)
	}

	server := NewServer(cfg, store, payments)
	clientReq := httptest.NewRequest(http.MethodGet, "/api/admin/summary", nil)
	clientReq.Header.Set("Authorization", "Bearer "+clientAuth.Token)
	clientResp := httptest.NewRecorder()
	server.Routes().ServeHTTP(clientResp, clientReq)
	if clientResp.Code != http.StatusForbidden {
		t.Fatalf("client admin summary status = %d", clientResp.Code)
	}

	adminReq := httptest.NewRequest(http.MethodGet, "/api/admin/summary", nil)
	adminReq.Header.Set("Authorization", "Bearer "+adminAuth.Token)
	adminResp := httptest.NewRecorder()
	server.Routes().ServeHTTP(adminResp, adminReq)
	if adminResp.Code != http.StatusOK {
		t.Fatalf("admin summary status = %d, body = %s", adminResp.Code, adminResp.Body.String())
	}
}

func TestConfiguredAdminBootstrapCanLogin(t *testing.T) {
	tempDir := t.TempDir()
	cfg := Config{
		TokenSymbol:       defaultTokenSymbol,
		StatePath:         filepath.Join(tempDir, "state.json"),
		PlatformFeeBps:    1000,
		DevPaymentEnabled: true,
		DevPaymentCode:    defaultDevPaymentCode,
		GitHubOwner:       defaultGitHubOwner,
		BountyRoot:        filepath.Join(tempDir, "bounties"),
		SMTPFrom:          "noreply@mergeos.local",
		AdminEmail:        defaultLocalAdminEmail,
		AdminPassword:     defaultLocalAdminPassword,
		AdminName:         "MergeOS Admin",
		AdminCompanyName:  "MergeOS",
	}
	payments := NewPaymentManager(cfg)
	store, err := NewStore(cfg, payments, NewRepoFactory(cfg), NewEmailSender(cfg))
	if err != nil {
		t.Fatal(err)
	}

	auth, err := store.Login(LoginRequest{
		Email:    defaultLocalAdminEmail,
		Password: defaultLocalAdminPassword,
	})
	if err != nil {
		t.Fatal(err)
	}
	if auth.User.Role != RoleAdmin {
		t.Fatalf("configured admin role = %q", auth.User.Role)
	}
}

func TestAdminCanUpdateUserAndPassword(t *testing.T) {
	tempDir := t.TempDir()
	cfg := Config{
		TokenSymbol:       defaultTokenSymbol,
		StatePath:         filepath.Join(tempDir, "state.json"),
		PlatformFeeBps:    1000,
		DevPaymentEnabled: true,
		DevPaymentCode:    defaultDevPaymentCode,
		GitHubOwner:       defaultGitHubOwner,
		BountyRoot:        filepath.Join(tempDir, "bounties"),
		SMTPFrom:          "noreply@mergeos.local",
		AdminEmail:        defaultLocalAdminEmail,
		AdminPassword:     defaultLocalAdminPassword,
		AdminName:         "MergeOS Admin",
		AdminCompanyName:  "MergeOS",
	}
	payments := NewPaymentManager(cfg)
	store, err := NewStore(cfg, payments, NewRepoFactory(cfg), NewEmailSender(cfg))
	if err != nil {
		t.Fatal(err)
	}
	clientAuth, err := store.Register(RegisterRequest{
		Name:        "Client User",
		CompanyName: "Old Co",
		Email:       "client@example.com",
		Password:    "password123",
	})
	if err != nil {
		t.Fatal(err)
	}
	adminAuth, err := store.Login(LoginRequest{Email: defaultLocalAdminEmail, Password: defaultLocalAdminPassword})
	if err != nil {
		t.Fatal(err)
	}

	server := NewServer(cfg, store, payments)
	body := strings.NewReader(`{"name":"Updated Client","company_name":"New Co","email":"updated@example.com","role":"client","password":"newpass123"}`)
	req := httptest.NewRequest(http.MethodPatch, "/api/admin/users/"+clientAuth.User.ID, body)
	req.Header.Set("Authorization", "Bearer "+adminAuth.Token)
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	server.Routes().ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("update user status = %d, body = %s", resp.Code, resp.Body.String())
	}
	var updated AdminUser
	if err := json.NewDecoder(resp.Body).Decode(&updated); err != nil {
		t.Fatal(err)
	}
	if updated.Name != "Updated Client" || updated.Email != "updated@example.com" || updated.CompanyName != "New Co" {
		t.Fatalf("updated user = %#v", updated)
	}
	if _, err := store.Login(LoginRequest{Email: "updated@example.com", Password: "password123"}); err == nil {
		t.Fatal("old password still works")
	}
	if _, err := store.Login(LoginRequest{Email: "updated@example.com", Password: "newpass123"}); err != nil {
		t.Fatalf("new password login failed: %v", err)
	}
}

func TestCannotDemoteOnlyAdmin(t *testing.T) {
	tempDir := t.TempDir()
	cfg := Config{
		TokenSymbol:       defaultTokenSymbol,
		StatePath:         filepath.Join(tempDir, "state.json"),
		PlatformFeeBps:    1000,
		DevPaymentEnabled: true,
		DevPaymentCode:    defaultDevPaymentCode,
		GitHubOwner:       defaultGitHubOwner,
		BountyRoot:        filepath.Join(tempDir, "bounties"),
		SMTPFrom:          "noreply@mergeos.local",
		AdminEmail:        defaultLocalAdminEmail,
		AdminPassword:     defaultLocalAdminPassword,
		AdminName:         "MergeOS Admin",
		AdminCompanyName:  "MergeOS",
	}
	payments := NewPaymentManager(cfg)
	store, err := NewStore(cfg, payments, NewRepoFactory(cfg), NewEmailSender(cfg))
	if err != nil {
		t.Fatal(err)
	}
	adminAuth, err := store.Login(LoginRequest{Email: defaultLocalAdminEmail, Password: defaultLocalAdminPassword})
	if err != nil {
		t.Fatal(err)
	}
	server := NewServer(cfg, store, payments)
	body := strings.NewReader(`{"name":"MergeOS Admin","company_name":"MergeOS","email":"admin@gmail.com","role":"client"}`)
	req := httptest.NewRequest(http.MethodPatch, "/api/admin/users/"+adminAuth.User.ID, body)
	req.Header.Set("Authorization", "Bearer "+adminAuth.Token)
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	server.Routes().ServeHTTP(resp, req)
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("only admin demotion status = %d, body = %s", resp.Code, resp.Body.String())
	}
}

func TestStoreImportsLegacyJSONWhenPostgresStateIsEmpty(t *testing.T) {
	tempDir := t.TempDir()
	legacyPath := filepath.Join(tempDir, "mergeos-state.json")
	legacyState := persistedState{
		NextID: 42,
		Users: []*User{{
			ID:           "usr_0001",
			Name:         "Legacy User",
			Email:        "legacy@example.com",
			Role:         RoleClient,
			PasswordSalt: "salt",
			PasswordHash: "hash",
			CreatedAt:    time.Now().UTC(),
		}},
	}
	if err := saveJSONState(legacyPath, legacyState); err != nil {
		t.Fatal(err)
	}

	storage := &memoryStatePersistence{}
	store := &Store{
		cfg:           Config{StatePath: legacyPath},
		storage:       storage,
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
	if err := store.load(); err != nil {
		t.Fatal(err)
	}
	if store.nextID != 42 {
		t.Fatalf("nextID = %d", store.nextID)
	}
	if len(store.users) != 1 {
		t.Fatalf("users = %d", len(store.users))
	}
	if !storage.saved {
		t.Fatal("legacy state was not saved into configured storage")
	}
	if len(storage.state.Users) != 1 || storage.state.Users[0].Email != "legacy@example.com" {
		t.Fatalf("saved users = %#v", storage.state.Users)
	}
}

func TestPostgresPersistenceRoundTrip(t *testing.T) {
	databaseURL := os.Getenv("MERGEOS_TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("MERGEOS_TEST_DATABASE_URL is not set")
	}
	tempDir := t.TempDir()
	cfg := Config{
		TokenSymbol:       defaultTokenSymbol,
		DatabaseURL:       databaseURL,
		StatePath:         filepath.Join(tempDir, "legacy-state.json"),
		PlatformFeeBps:    1000,
		DevPaymentEnabled: true,
		DevPaymentCode:    defaultDevPaymentCode,
		GitHubOwner:       defaultGitHubOwner,
		BountyRoot:        filepath.Join(tempDir, "bounties"),
		SMTPFrom:          "noreply@mergeos.local",
	}
	storage, err := newPostgresPersistence(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := storage.Save(context.Background(), persistedState{NextID: 1}); err != nil {
		t.Fatal(err)
	}
	if err := storage.Close(); err != nil {
		t.Fatal(err)
	}

	payments := NewPaymentManager(cfg)
	store, err := NewStore(cfg, payments, NewRepoFactory(cfg), NewEmailSender(cfg))
	if err != nil {
		t.Fatal(err)
	}
	auth, err := store.Register(RegisterRequest{
		Name:     "Postgres User",
		Email:    "postgres@example.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}

	reloaded, err := NewStore(cfg, payments, NewRepoFactory(cfg), NewEmailSender(cfg))
	if err != nil {
		t.Fatal(err)
	}
	defer reloaded.Close()
	user, ok := reloaded.UserByToken("Bearer " + auth.Token)
	if !ok {
		t.Fatal("reloaded store did not recognize persisted session")
	}
	if user.Email != "postgres@example.com" {
		t.Fatalf("reloaded email = %q", user.Email)
	}
}

type memoryStatePersistence struct {
	state persistedState
	found bool
	saved bool
}

func (m *memoryStatePersistence) Load(context.Context) (persistedState, bool, error) {
	return m.state, m.found, nil
}

func (m *memoryStatePersistence) Save(_ context.Context, state persistedState) error {
	m.state = state
	m.found = true
	m.saved = true
	return nil
}

func (m *memoryStatePersistence) Close() error {
	return nil
}
