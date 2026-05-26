package core

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
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
