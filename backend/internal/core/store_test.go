package core

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
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
	if len(store.ListLedger()) != 10 {
		t.Fatalf("ledger entries after create = %d", len(store.ListLedger()))
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
