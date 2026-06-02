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
	for _, entry := range ledger {
		if entry.Type == "task_reserve" && entry.ToAccount != taskReserveAccount() {
			t.Fatalf("task reserve account = %q, want %q", entry.ToAccount, taskReserveAccount())
		}
		if strings.Contains(entry.FromAccount, "reserve:task:") || strings.Contains(entry.ToAccount, "reserve:task:") {
			t.Fatalf("ledger entry exposed task reserve id: %#v", entry)
		}
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

func TestAdminSettingsPersistGeminiReviewModel(t *testing.T) {
	tempDir := t.TempDir()
	cfg := Config{
		TokenSymbol:       defaultTokenSymbol,
		StatePath:         filepath.Join(tempDir, "state.json"),
		PlatformFeeBps:    1000,
		DevPaymentEnabled: true,
		DevPaymentCode:    defaultDevPaymentCode,
		GeminiReviewModel: "gemini-2.5-pro",
		GitHubOwner:       defaultGitHubOwner,
		BountyRoot:        filepath.Join(tempDir, "bounties"),
		SMTPFrom:          "noreply@mergeos.local",
	}
	payments := NewPaymentManager(cfg)
	store, err := NewStore(cfg, payments, NewRepoFactory(cfg), NewEmailSender(cfg))
	if err != nil {
		t.Fatal(err)
	}
	if got := store.AdminSettings().GeminiReviewModel; got != "gemini-2.5-pro" {
		t.Fatalf("initial model = %q", got)
	}

	updated, err := store.UpdateAdminSettings(UpdateAdminSettingsRequest{GeminiReviewModel: "models/gemini-2.5-flash-lite"})
	if err != nil {
		t.Fatal(err)
	}
	if updated.GeminiReviewModel != "gemini-2.5-flash-lite" || store.GeminiReviewModel() != "gemini-2.5-flash-lite" {
		t.Fatalf("updated model not applied: %#v", updated)
	}
	if _, err := store.UpdateAdminSettings(UpdateAdminSettingsRequest{GeminiReviewModel: "bad model name"}); err == nil {
		t.Fatal("invalid model name accepted")
	}

	reloaded, err := NewStore(cfg, payments, NewRepoFactory(cfg), NewEmailSender(cfg))
	if err != nil {
		t.Fatal(err)
	}
	if got := reloaded.AdminSettings().GeminiReviewModel; got != "gemini-2.5-flash-lite" {
		t.Fatalf("reloaded model = %q", got)
	}
}

func TestAdminSettingsPersistLLMProviderModel(t *testing.T) {
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

	updated, err := store.UpdateAdminSettings(UpdateAdminSettingsRequest{
		LLMProvider: "openai",
		LLMModel:    "gpt-4o-mini",
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.LLMProvider != "openai" || updated.LLMModel != "gpt-4o-mini" {
		t.Fatalf("updated LLM settings not applied: %#v", updated)
	}
	if len(updated.LLMProviderOptions) == 0 {
		t.Fatal("missing LLM provider options")
	}
	provider, model := store.LLMReviewProviderModel()
	if provider != "openai" || model != "gpt-4o-mini" {
		t.Fatalf("store provider/model = %q/%q", provider, model)
	}

	key, err := store.AddGeminiAPIKey("sk-test-openai-token", "openai", "gpt-4o-mini")
	if err != nil {
		t.Fatal(err)
	}
	if key.Provider != "openai" || key.Model != "gpt-4o-mini" {
		t.Fatalf("key provider/model = %#v", key)
	}
	if !store.HasRunnableGeminiAPIKey() {
		t.Fatal("selected OpenAI token should be runnable")
	}

	reloaded, err := NewStore(cfg, payments, NewRepoFactory(cfg), NewEmailSender(cfg))
	if err != nil {
		t.Fatal(err)
	}
	reloadedSettings := reloaded.AdminSettings()
	if reloadedSettings.LLMProvider != "openai" || reloadedSettings.LLMModel != "gpt-4o-mini" {
		t.Fatalf("reloaded LLM settings = %#v", reloadedSettings)
	}
	reloadedKeys := reloaded.ListGeminiAPIKeyStats()
	if len(reloadedKeys) != 1 || reloadedKeys[0].Provider != "openai" || reloadedKeys[0].Model != "gpt-4o-mini" {
		t.Fatalf("reloaded LLM keys = %#v", reloadedKeys)
	}
}

func TestGitHubAuthLinksMRGWalletAndRoutesPayouts(t *testing.T) {
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

	wallet, err := store.CreateGuestWallet(CreateWalletRequest{})
	if err != nil {
		t.Fatal(err)
	}
	auth, err := store.AuthenticateGitHub(GitHubAuthProfile{
		ID:       "12345",
		Username: "Octo-Builder",
		Name:     "Octo Builder",
		Email:    "octo@example.com",
	}, wallet.Address, wallet.RecoveryCode)
	if err != nil {
		t.Fatal(err)
	}
	if auth.User.WalletAddress != wallet.Address {
		t.Fatalf("wallet address = %q, want %q", auth.User.WalletAddress, wallet.Address)
	}
	if auth.User.GitHubUsername != "octo-builder" {
		t.Fatalf("github username = %q", auth.User.GitHubUsername)
	}

	project, err := store.CreateProject(context.Background(), auth.User.ID, CreateProjectRequest{
		Title:            "GitHub reward route",
		ClientName:       "Octo Builder",
		ClientEmail:      "octo@example.com",
		Brief:            "Create a payable task for a linked GitHub wallet.",
		BudgetCents:      120000,
		PaymentMethod:    PaymentPayPal,
		PaymentReference: defaultDevPaymentCode,
	})
	if err != nil {
		t.Fatal(err)
	}
	accepted, err := store.AcceptTask(project.Tasks[0].ID, AcceptTaskRequest{
		WorkerKind: WorkerHuman,
		WorkerID:   "github:octo-builder",
	})
	if err != nil {
		t.Fatal(err)
	}
	if accepted.ProofHash == "" {
		t.Fatal("accepted task missing proof hash")
	}

	ledger := store.ListLedger()
	payout := ledger[len(ledger)-1]
	expectedAccount := walletAccount(wallet.Address)
	if payout.ToAccount != expectedAccount {
		t.Fatalf("payout account = %q, want %q", payout.ToAccount, expectedAccount)
	}
	if strings.HasPrefix(payout.ToAccount, "wallet:") {
		t.Fatalf("payout account kept legacy wallet prefix: %q", payout.ToAccount)
	}
	if payout.FromAccount != taskReserveAccount() {
		t.Fatalf("payout reserve account = %q, want %q", payout.FromAccount, taskReserveAccount())
	}
	summary, ok := store.WalletSummary(wallet.Address)
	if !ok {
		t.Fatal("wallet summary not found")
	}
	if summary.BalanceCents != project.Tasks[0].RewardCents || summary.GitHubUsername != "octo-builder" {
		t.Fatalf("wallet summary = %#v", summary)
	}

	publicLedger := store.ListPublicLedger()
	publicBody, err := json.Marshal(publicLedger)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(publicBody), wallet.Address) {
		t.Fatalf("public ledger did not expose wallet address: %s", publicBody)
	}
	if strings.Contains(string(publicBody), "github:octo-builder") {
		t.Fatalf("public ledger should expose wallet instead of github alias for linked wallets: %s", publicBody)
	}
	if strings.Contains(string(publicBody), "wallet:") {
		t.Fatalf("public ledger should expose raw wallet addresses: %s", publicBody)
	}
}

func TestLegacyWalletAccountPrefixMigratesToRawAddress(t *testing.T) {
	store := &Store{cfg: Config{GeminiReviewModel: defaultGeminiReviewModel}}
	wallet := &Wallet{
		Address:        "0x1234567890abcdef1234567890abcdef12345678",
		GitHubUsername: "octo-builder",
		CreatedAt:      time.Now().UTC(),
	}
	state := persistedState{
		Wallets: []*Wallet{wallet},
		Tasks: []*Task{
			{
				ID:         "tsk_0001",
				ProjectID:  "prj_0001",
				WorkerID:   legacyWalletAccount(wallet.Address),
				CreatedAt:  time.Now().UTC(),
				AcceptedAt: nil,
			},
		},
		Ledger: []LedgerEntry{
			{
				Sequence:    1,
				Type:        "task_payment",
				FromAccount: "reserve:task:tsk_0001",
				ToAccount:   legacyWalletAccount(wallet.Address),
				AmountCents: 10000,
				Reference:   "task:tsk_0001",
				CreatedAt:   time.Now().UTC(),
			},
		},
	}
	state.Ledger[0].PreviousHash = strings.Repeat("0", 64)
	state.Ledger[0].EntryHash = ledgerEntryHash(state.Ledger[0])

	if !store.applyState(state) {
		t.Fatal("legacy wallet account prefix did not report migration")
	}
	if got := store.ledger[0].ToAccount; got != wallet.Address {
		t.Fatalf("ledger account = %q, want %q", got, wallet.Address)
	}
	if got := store.ledger[0].FromAccount; got != taskReserveAccount() {
		t.Fatalf("reserve account = %q, want %q", got, taskReserveAccount())
	}
	if got := store.tasks["tsk_0001"].WorkerID; got != wallet.Address {
		t.Fatalf("task worker id = %q, want %q", got, wallet.Address)
	}
	summary, ok := store.WalletSummary(wallet.Address)
	if !ok {
		t.Fatal("wallet summary not found")
	}
	if summary.BalanceCents != 10000 || summary.Account != wallet.Address {
		t.Fatalf("wallet summary = %#v", summary)
	}
	publicLedger := store.ListPublicLedger()
	if publicLedger[0].ToAccount != wallet.Address {
		t.Fatalf("public account = %q, want %q", publicLedger[0].ToAccount, wallet.Address)
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

func TestSyncProjectImportedIssuesAddsMissingAndTracksState(t *testing.T) {
	store := &Store{
		cfg:      Config{StatePath: filepath.Join(t.TempDir(), "state.json")},
		nextID:   2,
		projects: map[string]*Project{},
		tasks:    map[string]*Task{},
	}
	project := &Project{ID: "prj_0001", Title: "Repo issues", Tasks: []*Task{}}
	existing := &Task{
		ID:          "tsk_0001",
		ProjectID:   project.ID,
		IssueNumber: 1,
		Title:       "Fix #1",
		Status:      TaskAccepted,
		IssueState:  "open",
		IssueURL:    "https://github.com/mergeos-bounties/mergeos/issues/1",
		CreatedAt:   time.Now().UTC(),
	}
	project.Tasks = append(project.Tasks, existing)
	store.projects[project.ID] = project
	store.tasks[existing.ID] = existing

	err := store.SyncProjectImportedIssues(project.ID, []*ImportedRepoIssue{
		{
			Number:             1,
			Title:              "Already imported",
			State:              "closed",
			URL:                existing.IssueURL,
			EstimatedCents:     100,
			RequiredWorkerKind: WorkerHuman,
		},
		{
			Number:             7,
			Title:              "New issue from GitHub",
			State:              "open",
			URL:                "https://github.com/mergeos-bounties/mergeos/issues/7",
			EstimatedCents:     100,
			RequiredWorkerKind: WorkerAgent,
			SuggestedAgentType: "backend-agent",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	tasks := store.ListTasks("")
	if len(tasks) != 2 {
		t.Fatalf("tasks = %d, want 2", len(tasks))
	}
	if tasks[0].IssueNumber != 1 || tasks[0].IssueState != "closed" || tasks[0].Status != TaskAccepted {
		t.Fatalf("existing issue not updated safely: %#v", tasks[0])
	}
	if tasks[1].IssueNumber != 7 || tasks[1].IssueState != "open" || tasks[1].Status != TaskOpen {
		t.Fatalf("missing issue not added: %#v", tasks[1])
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
	if len(payload.Bounties) == 0 {
		t.Fatalf("marketplace missing open bounty rows: %#v", payload)
	}
	for _, bounty := range payload.Bounties {
		for _, task := range project.Tasks {
			if strings.Contains(bounty.ID, task.ID) || strings.Contains(bounty.IssueURL, task.ID) {
				t.Fatalf("bounty leaked task id: %#v", bounty)
			}
		}
		if bounty.IssueURL != "" && !strings.HasPrefix(bounty.IssueURL, "http") {
			t.Fatalf("bounty issue URL is not public: %#v", bounty)
		}
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
	foundGitHubWorker := false
	for _, entry := range payload {
		if strings.Contains(entry.FromAccount, "client:") || strings.Contains(entry.ToAccount, "client:") {
			t.Fatalf("public ledger leaked client account: %#v", entry)
		}
		if strings.Contains(entry.Reference, project.ID) {
			foundProjectReference = true
		}
		if entry.ToAccount == "github:private-worker" {
			foundGitHubWorker = true
		}
	}
	if !foundProjectReference {
		t.Fatalf("public ledger did not preserve project reference: %#v", payload)
	}
	if !foundGitHubWorker {
		t.Fatalf("public ledger did not expose github worker account: %#v", payload)
	}
}

func TestPublicLedgerUsesPullReferenceForAdminAcceptedTask(t *testing.T) {
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
		Name:        "PR Ledger Client",
		CompanyName: "PR Ledger Co",
		Email:       "pr-ledger@example.com",
		Password:    "password123",
	})
	if err != nil {
		t.Fatal(err)
	}
	project, err := store.CreateProject(context.Background(), auth.User.ID, CreateProjectRequest{
		Title:            "PR ledger proof",
		ClientName:       "PR Ledger Client",
		ClientEmail:      "pr-ledger@example.com",
		Brief:            "Create a task payout whose public reference points at the merged PR.",
		BudgetCents:      120000,
		PaymentMethod:    PaymentPayPal,
		PaymentReference: defaultDevPaymentCode,
	})
	if err != nil {
		t.Fatal(err)
	}
	req, err := acceptRequestForPullAuthor(project.Tasks[0], "pr-author")
	if err != nil {
		t.Fatal(err)
	}
	pullReference := buildPullLedgerReference(project.Tasks[0].ID, "https://github.com/mergeos-bounties/mergeos/pull/120", "Fix PR payout reference")
	if _, err := store.AcceptTaskWithReviewReference(project.Tasks[0].ID, req, 50, "future-medium", pullReference); err != nil {
		t.Fatal(err)
	}
	account, ok := store.TaskPayoutAccount(project.Tasks[0].ID)
	if !ok || account != "github:pr-author" {
		t.Fatalf("task payout account = %q, %v", account, ok)
	}

	found := false
	for _, entry := range store.ListPublicLedger() {
		if entry.Type != "task_payment" {
			continue
		}
		found = true
		if entry.Reference != "pr:https://github.com/mergeos-bounties/mergeos/pull/120;title:Fix PR payout reference" {
			t.Fatalf("public task payout reference = %q", entry.Reference)
		}
		if strings.Contains(entry.Reference, project.ID) || strings.Contains(entry.Reference, project.Tasks[0].ID) {
			t.Fatalf("public task payout reference still exposes project/task id: %s", entry.Reference)
		}
	}
	if !found {
		t.Fatal("public ledger did not include task payout")
	}
}

func TestPublicLiveFeedRouteReturnsSanitizedTimeline(t *testing.T) {
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
		Name:        "Feed Client",
		CompanyName: "Feed Co",
		Email:       "feed@example.com",
		Password:    "password123",
	})
	if err != nil {
		t.Fatal(err)
	}
	project, err := store.CreateProject(context.Background(), auth.User.ID, CreateProjectRequest{
		Title:            "Live feed proof",
		ClientName:       "Private Feed Client",
		CompanyName:      "Feed Co",
		ClientEmail:      "feed@example.com",
		Phone:            "+1 555 0177",
		Brief:            "Create public live feed data without leaking private customer data.",
		BudgetCents:      180000,
		PaymentMethod:    PaymentPayPal,
		PaymentReference: defaultDevPaymentCode,
	})
	if err != nil {
		t.Fatal(err)
	}
	req, err := acceptRequestForPullAuthor(project.Tasks[0], "feed-author")
	if err != nil {
		t.Fatal(err)
	}
	pullReference := buildPullLedgerReference(project.Tasks[0].ID, "https://github.com/mergeos-bounties/mergeos/pull/151", "Live feed proof")
	if _, err := store.AcceptTaskWithReviewReference(project.Tasks[0].ID, req, 5000, "future-medium", pullReference); err != nil {
		t.Fatal(err)
	}
	if err := store.AddGeminiWebhookLog(GeminiWebhookLog{
		EventName:  "pull_request",
		Action:     "opened",
		Repository: "mergeos-bounties/mergeos",
		PullNumber: 151,
		Sender:     "ai-reviewer",
		Status:     "processed",
		StatusCode: http.StatusOK,
		CommentURL: "https://github.com/mergeos-bounties/mergeos/pull/151#issuecomment-1",
		ReceivedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatal(err)
	}

	server := NewServer(cfg, store, payments)
	reqHTTP := httptest.NewRequest(http.MethodGet, "/api/public/live-feed?limit=50", nil)
	resp := httptest.NewRecorder()
	server.Routes().ServeHTTP(resp, reqHTTP)
	if resp.Code != http.StatusOK {
		t.Fatalf("live feed status = %d, body = %s", resp.Code, resp.Body.String())
	}

	body := resp.Body.String()
	privateValues := []string{
		"feed@example.com",
		"+1 555 0177",
		auth.User.ID,
		tempDir,
		defaultDevPaymentCode,
		project.Tasks[0].ID,
	}
	for _, value := range privateValues {
		if strings.Contains(body, value) {
			t.Fatalf("public live feed leaked private value %q: %s", value, body)
		}
	}

	var payload PublicLiveFeedResponse
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Stats.ProjectCount != 1 || payload.Stats.AIActionCount != 1 || payload.Stats.LedgerEntryCount == 0 {
		t.Fatalf("unexpected live feed stats: %#v", payload.Stats)
	}
	seen := map[string]bool{}
	for _, item := range payload.Items {
		seen[item.Type] = true
		if item.Type == "ledger_task_payment" {
			if item.Reference != "pr:https://github.com/mergeos-bounties/mergeos/pull/151;title:Live feed proof" {
				t.Fatalf("task payout feed reference = %q", item.Reference)
			}
			if item.URL != "https://github.com/mergeos-bounties/mergeos/pull/151" {
				t.Fatalf("task payout feed url = %q", item.URL)
			}
		}
	}
	for _, required := range []string{"project_funded", "task_accepted", "ledger_task_payment", "ai_review"} {
		if !seen[required] {
			t.Fatalf("live feed missing %s item: %#v", required, payload.Items)
		}
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

func TestAdminCanCreateManualLedgerCredit(t *testing.T) {
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
		ScanDomain:        "scan.mergeos.shop",
	}
	payments := NewPaymentManager(cfg)
	store, err := NewStore(cfg, payments, NewRepoFactory(cfg), NewEmailSender(cfg))
	if err != nil {
		t.Fatal(err)
	}
	adminAuth, err := store.Register(RegisterRequest{
		Name:     "Admin User",
		Email:    "credit-admin@example.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatal(err)
	}
	server := NewServer(cfg, store, payments)
	body := strings.NewReader(`{"worker_id":"eliasx45","reward_mrg":50,"bounty_type":"future-medium","pr_url":"https://github.com/mergeos-bounties/mergeos/pull/120","pr_title":"Public timeline correction"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/admin/ledger/credits", body)
	req.Header.Set("Authorization", "Bearer "+adminAuth.Token)
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	server.Routes().ServeHTTP(resp, req)
	if resp.Code != http.StatusCreated {
		t.Fatalf("manual credit status = %d, body = %s", resp.Code, resp.Body.String())
	}
	var payload AdminManualCreditResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	if payload.WorkerID != "github:eliasx45" || payload.RewardMRG != 50 || payload.LedgerEntry.Type != "manual_credit" {
		t.Fatalf("manual credit response = %#v", payload)
	}
	if payload.LedgerEntry.ToAccount != "github:eliasx45" {
		t.Fatalf("manual credit account = %q", payload.LedgerEntry.ToAccount)
	}
	if payload.LedgerEntry.Reference != "pr:https://github.com/mergeos-bounties/mergeos/pull/120;title:Public timeline correction" {
		t.Fatalf("manual credit reference = %q", payload.LedgerEntry.Reference)
	}
	if !strings.Contains(payload.CreditURL, "/address/github:eliasx45") {
		t.Fatalf("manual credit URL = %q", payload.CreditURL)
	}
	foundPublicReference := false
	for _, entry := range store.ListPublicLedger() {
		if entry.Type == "manual_credit" && entry.Reference == payload.LedgerEntry.Reference {
			foundPublicReference = true
			break
		}
	}
	if !foundPublicReference {
		t.Fatalf("manual credit missing from public ledger: %#v", store.ListPublicLedger())
	}
}

func TestAdminTasksRouteIncludesAcceptedTasksForAudit(t *testing.T) {
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
		Email:    "review-admin@example.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatal(err)
	}
	project, err := store.CreateProject(context.Background(), adminAuth.User.ID, CreateProjectRequest{
		Title:            "Review queue",
		ClientName:       "Admin User",
		ClientEmail:      "review-admin@example.com",
		Brief:            "Create tasks and keep paid work visible in the admin audit board.",
		BudgetCents:      120000,
		PaymentMethod:    PaymentPayPal,
		PaymentReference: defaultDevPaymentCode,
	})
	if err != nil {
		t.Fatal(err)
	}
	req, err := acceptRequestForPullAuthor(project.Tasks[0], "reviewer")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.AcceptTask(project.Tasks[0].ID, req); err != nil {
		t.Fatal(err)
	}

	server := NewServer(cfg, store, payments)
	adminReq := httptest.NewRequest(http.MethodGet, "/api/admin/tasks", nil)
	adminReq.Header.Set("Authorization", "Bearer "+adminAuth.Token)
	adminResp := httptest.NewRecorder()
	server.Routes().ServeHTTP(adminResp, adminReq)
	if adminResp.Code != http.StatusOK {
		t.Fatalf("admin tasks status = %d, body = %s", adminResp.Code, adminResp.Body.String())
	}
	var payload []Task
	if err := json.Unmarshal(adminResp.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	foundAccepted := false
	for _, task := range payload {
		if task.ID == project.Tasks[0].ID && task.Status == TaskAccepted {
			foundAccepted = true
			break
		}
	}
	if !foundAccepted {
		t.Fatalf("accepted task missing from admin task audit board: %#v", payload)
	}
	if len(payload) != len(project.Tasks) {
		t.Fatalf("admin task count = %d, want %d", len(payload), len(project.Tasks))
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
