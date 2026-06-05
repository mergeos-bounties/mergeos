package core

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
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

func TestRuntimeConfigReturnsPaymentRails(t *testing.T) {
	tempDir := t.TempDir()
	cfg := Config{
		TokenSymbol:          defaultTokenSymbol,
		StatePath:            filepath.Join(tempDir, "state.json"),
		PlatformFeeBps:       1000,
		DevPaymentEnabled:    true,
		DevPaymentCode:       defaultDevPaymentCode,
		PayPalClientID:       "paypal-client",
		PayPalClientSecret:   "paypal-secret",
		StripePublishableKey: "pk_test_mergeos",
		StripeSecretKey:      "sk_test_secret",
		StripeWebhookSecret:  "whsec_secret",
		CryptoRPCURL:         "https://rpc.example",
		CryptoReceiver:       "So11111111111111111111111111111111111111112",
		CryptoAsset:          "spl",
		CryptoTokenContract:  "TokenkegQfeZyiNwAJbNbGKPFXCWuBvf9Ss623VQ5DA",
		CryptoTokenDecimals:  6,
		GitHubOwner:          defaultGitHubOwner,
		BountyRoot:           filepath.Join(tempDir, "bounties"),
		SMTPFrom:             "noreply@mergeos.local",
	}
	payments := NewPaymentManager(cfg)
	store, err := NewStore(cfg, payments, NewRepoFactory(cfg), NewEmailSender(cfg))
	if err != nil {
		t.Fatal(err)
	}
	server := NewServer(cfg, store, payments)
	req := httptest.NewRequest(http.MethodGet, "/api/config", nil)
	resp := httptest.NewRecorder()
	server.Routes().ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("config status = %d, body = %s", resp.Code, resp.Body.String())
	}
	body := resp.Body.String()
	for _, secret := range []string{"paypal-secret", "sk_test_secret", "whsec_secret"} {
		if strings.Contains(body, secret) {
			t.Fatalf("config leaked secret %q: %s", secret, body)
		}
	}

	var payload RuntimeConfigResponse
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if !payload.PayPalReady || !payload.CryptoReady || !payload.StripeReady || payload.StripePublicKey != "pk_test_mergeos" {
		t.Fatalf("unexpected payment readiness: %#v", payload)
	}
	rails := map[string]PaymentRailOption{}
	for _, rail := range payload.PaymentRails {
		rails[rail.ID] = rail
	}
	for _, required := range []string{"paypal", "crypto", "usdt", "stripe", "bank"} {
		if rails[required].ID == "" {
			t.Fatalf("missing payment rail %s: %#v", required, payload.PaymentRails)
		}
	}
	if !rails["paypal"].Enabled || rails["paypal"].Method != string(PaymentPayPal) {
		t.Fatalf("paypal rail not enabled: %#v", rails["paypal"])
	}
	if !rails["crypto"].Enabled || rails["crypto"].Label != "Solana SPL" || rails["crypto"].TokenContract == "" {
		t.Fatalf("crypto rail missing metadata: %#v", rails["crypto"])
	}
	if !rails["usdt"].Enabled || rails["usdt"].Label != "Solana SPL" || rails["usdt"].Method != string(PaymentUSDT) || rails["usdt"].TokenContract == "" {
		t.Fatalf("solana alias rail missing metadata: %#v", rails["usdt"])
	}
	if !rails["stripe"].Enabled || !rails["stripe"].Ready || rails["stripe"].Method != string(PaymentStripe) || rails["stripe"].PublicKey != "pk_test_mergeos" || rails["stripe"].DisabledReason != "" {
		t.Fatalf("stripe rail should be enabled when verifier is configured: %#v", rails["stripe"])
	}
	if rails["bank"].Enabled || rails["bank"].DisabledReason == "" {
		t.Fatalf("bank rail should be disabled with reason: %#v", rails["bank"])
	}
}

func TestCreateProjectAcceptsSolanaAliasPaymentMethod(t *testing.T) {
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
		Name:        "Solana Client",
		CompanyName: "Solana Co",
		Email:       "solana-client@example.com",
		Password:    "password123",
	})
	if err != nil {
		t.Fatal(err)
	}
	project, err := store.CreateProject(context.Background(), auth.User.ID, CreateProjectRequest{
		Title:            "Solana funded project",
		ClientName:       "Solana Client",
		CompanyName:      "Solana Co",
		ClientEmail:      "solana-client@example.com",
		Brief:            "Fund a project through the Solana SPL payment rail alias.",
		BudgetCents:      120000,
		PaymentMethod:    PaymentUSDT,
		PaymentReference: defaultDevPaymentCode,
	})
	if err != nil {
		t.Fatal(err)
	}
	if project.PaymentMethod != PaymentUSDT || project.PaymentProvider != "dev-solana-spl" || project.PaymentStatus != "verified" {
		t.Fatalf("unexpected solana alias project payment fields: %#v", project)
	}
}

func TestCreateProjectAcceptsStripePaymentMethod(t *testing.T) {
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
		Name:        "Stripe Client",
		CompanyName: "Stripe Co",
		Email:       "stripe-client@example.com",
		Password:    "password123",
	})
	if err != nil {
		t.Fatal(err)
	}
	project, err := store.CreateProject(context.Background(), auth.User.ID, CreateProjectRequest{
		Title:            "Stripe funded project",
		ClientName:       "Stripe Client",
		CompanyName:      "Stripe Co",
		ClientEmail:      "stripe-client@example.com",
		Brief:            "Fund a project through the Stripe PaymentIntent rail.",
		BudgetCents:      120000,
		PaymentMethod:    PaymentStripe,
		PaymentReference: defaultDevPaymentCode,
	})
	if err != nil {
		t.Fatal(err)
	}
	if project.PaymentMethod != PaymentStripe || project.PaymentProvider != "dev-stripe" || project.PaymentStatus != "verified" {
		t.Fatalf("unexpected stripe project payment fields: %#v", project)
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

func TestPasswordResetRequestIsGenericAndNotifiesExistingUser(t *testing.T) {
	tempDir := t.TempDir()
	cfg := Config{
		TokenSymbol: defaultTokenSymbol,
		StatePath:   filepath.Join(tempDir, "state.json"),
		SMTPFrom:    "noreply@mergeos.local",
	}
	payments := NewPaymentManager(cfg)
	store, err := NewStore(cfg, payments, NewRepoFactory(cfg), NewEmailSender(cfg))
	if err != nil {
		t.Fatal(err)
	}
	auth, err := store.Register(RegisterRequest{
		Name:     "Reset Client",
		Email:    "reset@example.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatal(err)
	}
	before := len(store.ListNotifications(auth.User.ID))

	existing, err := store.RequestPasswordReset(PasswordResetRequest{Email: "reset@example.com"})
	if err != nil {
		t.Fatal(err)
	}
	if existing.Status != "ok" || existing.Message == "" {
		t.Fatalf("unexpected reset response: %#v", existing)
	}
	afterExisting := store.ListNotifications(auth.User.ID)
	if len(afterExisting) != before+1 {
		t.Fatalf("notifications after existing reset = %d, want %d", len(afterExisting), before+1)
	}

	unknown, err := store.RequestPasswordReset(PasswordResetRequest{Email: "missing@example.com"})
	if err != nil {
		t.Fatal(err)
	}
	if unknown.Status != existing.Status || unknown.Message != existing.Message {
		t.Fatalf("reset response enumerates account existence: existing=%#v unknown=%#v", existing, unknown)
	}
	if len(store.ListNotifications(auth.User.ID)) != len(afterExisting) {
		t.Fatal("unknown reset request changed existing user notifications")
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

func TestLegacyWalletAccountPrefixMigratesToSolanaAddress(t *testing.T) {
	store := &Store{cfg: Config{GeminiReviewModel: defaultGeminiReviewModel}}
	legacyAddress := "0x1234567890abcdef1234567890abcdef12345678"
	expectedAddress := solanaWalletFromLegacy(legacyAddress)
	if !validWalletAddress(expectedAddress) {
		t.Fatalf("migration produced invalid Solana wallet %q", expectedAddress)
	}
	wallet := &Wallet{
		Address:        legacyAddress,
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
	if got := store.ledger[0].ToAccount; got != expectedAddress {
		t.Fatalf("ledger account = %q, want %q", got, expectedAddress)
	}
	if got := store.ledger[0].FromAccount; got != taskReserveAccount() {
		t.Fatalf("reserve account = %q, want %q", got, taskReserveAccount())
	}
	if got := store.tasks["tsk_0001"].WorkerID; got != expectedAddress {
		t.Fatalf("task worker id = %q, want %q", got, expectedAddress)
	}
	summary, ok := store.WalletSummary(expectedAddress)
	if !ok {
		t.Fatal("wallet summary not found")
	}
	if summary.BalanceCents != 10000 || summary.Account != expectedAddress || summary.Chain != walletChainSolana || summary.LegacyAddress != legacyAddress {
		t.Fatalf("wallet summary = %#v", summary)
	}
	publicLedger := store.ListPublicLedger()
	if publicLedger[0].ToAccount != expectedAddress {
		t.Fatalf("public account = %q, want %q", publicLedger[0].ToAccount, expectedAddress)
	}
}

func TestCreateWalletMigrationLinksLegacyTRC20ToSolanaMetadata(t *testing.T) {
	tempDir := t.TempDir()
	cfg := Config{
		TokenSymbol:         defaultTokenSymbol,
		StatePath:           filepath.Join(tempDir, "state.json"),
		CryptoRPCURL:        "https://api.devnet.solana.com",
		CryptoReceiver:      base58Encode(bytes.Repeat([]byte{4}, walletAddressBytes)),
		CryptoTokenContract: base58Encode(bytes.Repeat([]byte{5}, walletAddressBytes)),
		CryptoTokenDecimals: 6,
		GeminiReviewModel:   defaultGeminiReviewModel,
		AdminAutoPromote:    false,
		DevPaymentEnabled:   true,
		DevPaymentCode:      defaultDevPaymentCode,
		PlatformFeeBps:      1000,
	}
	store, err := NewStore(cfg, NewPaymentManager(cfg), NewRepoFactory(cfg), NewEmailSender(cfg))
	if err != nil {
		t.Fatal(err)
	}

	auth, err := store.Register(RegisterRequest{
		Name:     "Legacy Tron User",
		Email:    "legacy-tron@example.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatal(err)
	}
	legacyAddress := base58Encode(append([]byte{0x41}, bytes.Repeat([]byte{9}, 24)...))

	migration, err := store.CreateWalletMigration(auth.User.ID, CreateWalletMigrationRequest{
		LegacyChain:   "tron",
		LegacyAddress: "tron:" + legacyAddress,
	}, cfg)
	if err != nil {
		t.Fatal(err)
	}

	if migration.ProtocolVersion != "mergeos.wallet-migration.v1" || migration.Kind != "wallet_migration" {
		t.Fatalf("migration protocol = %#v", migration)
	}
	if migration.Status != "pending_contract_registration" || migration.Contract.ProgramReady {
		t.Fatalf("migration contract readiness = %q/%v", migration.Status, migration.Contract.ProgramReady)
	}
	if migration.LegacyChain != "trc20" || migration.LegacyAddress != legacyAddress {
		t.Fatalf("legacy fields = %q/%q", migration.LegacyChain, migration.LegacyAddress)
	}
	if want := legacyWalletAddressHashHex("trc20", legacyAddress); migration.LegacyAddressHash != want {
		t.Fatalf("legacy hash = %q, want %q", migration.LegacyAddressHash, want)
	}
	if migration.TargetChain != walletChainSolana || !validWalletAddress(migration.TargetAddress) {
		t.Fatalf("target wallet = %q/%q", migration.TargetChain, migration.TargetAddress)
	}
	user, ok := store.UserByToken(auth.Token)
	if !ok || user.WalletAddress != migration.TargetAddress {
		t.Fatalf("user wallet address = %#v, want %q", user, migration.TargetAddress)
	}
	if migration.TargetAddress == solanaWalletFromLegacy(legacyAddress) {
		t.Fatalf("migration API used deterministic legacy-derived address %q instead of a user Solana wallet", migration.TargetAddress)
	}
	if migration.Contract.ProgramID != "" {
		t.Fatalf("program id = %q, want empty until deployment env is configured", migration.Contract.ProgramID)
	}
	if got := migration.Contract.PDASeeds; len(got) != 3 || got[2] != "legacy_address_hash_bytes" {
		t.Fatalf("pda seeds = %#v", got)
	}
	if got := migration.Contract.PDASeedFormats; len(got) != 3 || got[2] != "bytes32:hex_decode(contract.args.legacy_address_hash)" {
		t.Fatalf("pda seed formats = %#v", got)
	}
	summary, ok := store.WalletSummary(migration.TargetAddress)
	if !ok {
		t.Fatal("migration wallet summary not found")
	}
	if summary.Chain != walletChainSolana || summary.LegacyAddress != legacyAddress || !summary.OwnerLinked {
		t.Fatalf("wallet summary = %#v", summary)
	}
}

func TestNewWalletAddressUsesSolanaBase58(t *testing.T) {
	address, err := newWalletAddress()
	if err != nil {
		t.Fatal(err)
	}
	if !validWalletAddress(address) {
		t.Fatalf("new wallet address is invalid: %q", address)
	}
	if strings.HasPrefix(address, "0x") {
		t.Fatalf("new wallet address still uses EVM form: %q", address)
	}
}

func TestLedgerEntryMatchingUsesExactIDBoundaries(t *testing.T) {
	entry := LedgerEntry{
		Type:        "task_payment",
		FromAccount: "reserve:project:prj_0010",
		ToAccount:   "worker:github:builder",
		Reference:   "task:tsk_0010;pr:https://github.com/mergeos-bounties/mergeos/pull/10",
	}

	if ledgerEntryMatches(entry, map[string]bool{"prj_001": true}, map[string]bool{"tsk_001": true}) {
		t.Fatalf("ledger matching accepted prefix IDs: %#v", entry)
	}
	projectID, taskID := publicLedgerScope(entry, map[string]bool{"prj_001": true}, map[string]string{"tsk_001": "prj_001"})
	if projectID != "" || taskID != "" {
		t.Fatalf("public ledger scope accepted prefix IDs: project=%q task=%q", projectID, taskID)
	}

	if !ledgerEntryMatches(entry, map[string]bool{"prj_0010": true}, map[string]bool{}) {
		t.Fatalf("ledger matching missed exact project ID: %#v", entry)
	}
	projectID, taskID = publicLedgerScope(entry, map[string]bool{}, map[string]string{"tsk_0010": "prj_0010"})
	if projectID != "prj_0010" || taskID != "tsk_0010" {
		t.Fatalf("public ledger scope missed exact task ID: project=%q task=%q", projectID, taskID)
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

func TestCreateProjectCanDisableAgentRouting(t *testing.T) {
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
		Name:     "Human Only Client",
		Email:    "human-only@example.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatal(err)
	}
	allowAgents := false
	project, err := store.CreateProject(context.Background(), auth.User.ID, CreateProjectRequest{
		Title:            "Human only routing",
		ClientName:       "Human Only Client",
		ClientEmail:      "human-only@example.com",
		Brief:            "Route this funded project only to human contributors.",
		BudgetCents:      120000,
		PaymentMethod:    PaymentPayPal,
		PaymentReference: defaultDevPaymentCode,
		AllowAgents:      &allowAgents,
	})
	if err != nil {
		t.Fatal(err)
	}
	if project.AllowAgents == nil || *project.AllowAgents {
		t.Fatalf("allow_agents was not persisted as false: %#v", project.AllowAgents)
	}
	if len(project.Tasks) == 0 {
		t.Fatal("expected funded tasks")
	}
	for _, task := range project.Tasks {
		if task.RequiredWorkerKind != WorkerHuman || strings.TrimSpace(task.SuggestedAgentType) != "" {
			t.Fatalf("task was not routed human-only: %#v", task)
		}
	}
	marketplace := store.Marketplace()
	if len(marketplace.Agents) != 0 {
		t.Fatalf("human-only project exposed agent lanes: %#v", marketplace.Agents)
	}
}

func TestSyncProjectImportedIssuesAddsMissingAndTracksState(t *testing.T) {
	store := &Store{
		cfg:      Config{StatePath: filepath.Join(t.TempDir(), "state.json")},
		nextID:   2,
		projects: map[string]*Project{},
		tasks:    map[string]*Task{},
	}
	project := &Project{
		ID:             "prj_0001",
		Title:          "Repo issues",
		ClientEmail:    "private-repo-sync@example.com",
		Phone:          "+1 555 0199",
		BountyRepoName: "mergeos-bounties/mergeos",
		RepoURL:        "https://github.com/mergeos-bounties/mergeos",
		Tasks:          []*Task{},
	}
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

	report, err := store.SyncProjectImportedIssuesReport(project.ID, "https://github.com/mergeos-bounties/mergeos", []*ImportedRepoIssue{
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
	if report.ProtocolVersion != "mergeos.repo-sync.v1" || report.Kind != "repo_sync" {
		t.Fatalf("unexpected repo sync protocol header: %#v", report)
	}
	if report.ProjectID != project.ID || report.SourceRepoURL == "" || report.ImportedIssueCount != 2 || report.AddedTaskCount != 1 || report.UpdatedTaskCount != 1 || report.OpenIssueCount != 1 || report.ClosedIssueCount != 1 {
		t.Fatalf("sync report = %#v", report)
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
	if err := store.RecordRepoIssueSyncEvent(report); err != nil {
		t.Fatal(err)
	}
	feed := store.PublicLiveFeed(20)
	if feed.Stats.AIActionCount != 1 {
		t.Fatalf("repo sync event count = %d", feed.Stats.AIActionCount)
	}
	seenRepoSync := false
	for _, item := range feed.Items {
		if item.Type == "repo_issues_synced" {
			seenRepoSync = item.Actor == "mergeos-repo-sync" && strings.Contains(item.Body, "2 issues")
		}
	}
	if !seenRepoSync {
		t.Fatalf("public live feed missing repo sync event: %#v", feed.Items)
	}
	feedBody, err := json.Marshal(feed)
	if err != nil {
		t.Fatal(err)
	}
	for _, value := range []string{"private-repo-sync@example.com", "+1 555 0199", existing.ID, tasks[1].ID} {
		if strings.Contains(string(feedBody), value) {
			t.Fatalf("repo sync live feed leaked private value %q: %s", value, string(feedBody))
		}
	}
	events := store.PublicEventProtocol(20)
	seenRepoSyncEvent := false
	for _, event := range events.Events {
		if event.Type == "repo.issues.synced" {
			seenRepoSyncEvent = event.Actor == "mergeos-repo-sync"
		}
	}
	if !seenRepoSyncEvent {
		t.Fatalf("public event protocol missing repo sync event: %#v", events.Events)
	}
	deployment, err := store.ProjectDeployment(project.ID)
	if err != nil {
		t.Fatal(err)
	}
	seenDeploymentSignal := false
	for _, signal := range deployment.Signals {
		if signal.Type == "repo_issues_synced" {
			seenDeploymentSignal = true
		}
	}
	if !seenDeploymentSignal {
		t.Fatalf("deployment signals missing repo sync event: %#v", deployment.Signals)
	}
}

func TestSyncProjectImportedIssuesHonorsHumanOnlyPolicy(t *testing.T) {
	allowAgents := false
	store := &Store{
		cfg:      Config{StatePath: filepath.Join(t.TempDir(), "state.json")},
		nextID:   1,
		projects: map[string]*Project{},
		tasks:    map[string]*Task{},
	}
	project := &Project{
		ID:          "prj_0001",
		Title:       "Human only sync",
		AllowAgents: &allowAgents,
		Tasks:       []*Task{},
	}
	store.projects[project.ID] = project

	if _, err := store.SyncProjectImportedIssuesReport(project.ID, "https://github.com/mergeos-bounties/mergeos", []*ImportedRepoIssue{{
		Number:             8,
		Title:              "Agent-looking issue",
		State:              "open",
		URL:                "https://github.com/mergeos-bounties/mergeos/issues/8",
		EstimatedCents:     100,
		RequiredWorkerKind: WorkerAgent,
		SuggestedAgentType: "backend-agent",
	}}); err != nil {
		t.Fatal(err)
	}

	tasks := store.ListTasks("")
	if len(tasks) != 1 {
		t.Fatalf("tasks = %d, want 1", len(tasks))
	}
	if tasks[0].RequiredWorkerKind != WorkerHuman || strings.TrimSpace(tasks[0].SuggestedAgentType) != "" {
		t.Fatalf("synced task did not honor human-only policy: %#v", tasks[0])
	}
}

func TestPublicProtocolManifestRouteReturnsDiscoveryMetadata(t *testing.T) {
	tempDir := t.TempDir()
	cfg := Config{
		TokenSymbol:       defaultTokenSymbol,
		StatePath:         filepath.Join(tempDir, "state.json"),
		PlatformFeeBps:    1000,
		DevPaymentEnabled: true,
		DevPaymentCode:    defaultDevPaymentCode,
		BountyRoot:        filepath.Join(tempDir, "bounties"),
	}
	payments := NewPaymentManager(cfg)
	store, err := NewStore(cfg, payments, NewRepoFactory(cfg), NewEmailSender(cfg))
	if err != nil {
		t.Fatal(err)
	}
	server := NewServer(cfg, store, payments)
	req := httptest.NewRequest(http.MethodGet, "/api/public/protocol", nil)
	resp := httptest.NewRecorder()
	server.Routes().ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("protocol manifest status = %d, body = %s", resp.Code, resp.Body.String())
	}

	var payload ProtocolManifestResponse
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.ProtocolVersion != "mergeos.protocol.manifest.v1" || payload.Kind != "protocol_manifest" {
		t.Fatalf("unexpected manifest header: %#v", payload)
	}
	if len(payload.Schemas) != 30 {
		t.Fatalf("manifest schemas = %d: %#v", len(payload.Schemas), payload.Schemas)
	}
	schemas := map[string]bool{}
	descriptions := map[string]string{}
	for _, schema := range payload.Schemas {
		schemas[schema.Version] = true
		descriptions[schema.Version] = schema.Description
	}
	for _, required := range []string{"mergeos.task.v1", "mergeos.task-claim.v1", "mergeos.agent.v1", "mergeos.contributor.v1", "mergeos.agent-action.v1", "mergeos.agent-queue.v1", "mergeos.marketplace.v1", "mergeos.live-feed.v1", "mergeos.workflow.v1", "mergeos.estimate.v1", "mergeos.wallet-migration.v1", "mergeos.repo-import.v1", "mergeos.repo-sync.v1", "mergeos.dispute.v1", "mergeos.proposal.v1", "mergeos.ai-workflow.v1", "mergeos.event.v1", "mergeos.ledger.v1", "mergeos.ledger-proof.v1", "mergeos.token-economy.v1", "mergeos.escrow.v1", "mergeos.payouts.v1", "mergeos.payout-release.v1", "mergeos.deployment.v1", "mergeos.pr-monitor.v1", "mergeos.scan.v1", "mergeos.customer-dashboard.v1", "mergeos.worker-dashboard.v1", "mergeos.routing.v1", "mergeos.admin-ops.v1"} {
		if !schemas[required] {
			t.Fatalf("manifest missing schema %s: %#v", required, payload.Schemas)
		}
	}
	if !strings.Contains(descriptions["mergeos.workflow.v1"], "current AI workflow step") {
		t.Fatalf("workflow schema description missing current step contract: %#v", descriptions["mergeos.workflow.v1"])
	}
	endpoints := map[string]bool{}
	for _, endpoint := range payload.Endpoints {
		endpoints[endpoint.Method+" "+endpoint.Path] = true
	}
	for _, required := range []string{
		"GET /api/public/marketplace",
		"GET /api/public/live-feed",
		"GET /api/public/protocol/tasks",
		"GET /api/public/protocol/agents",
		"GET /api/public/protocol/agent-queue",
		"GET /api/public/protocol/contributors",
		"GET /api/public/protocol/ledger",
		"GET /api/public/ledger/proof",
		"GET /api/public/ledger/events",
		"GET /api/public/token-economy",
		"GET /api/public/protocol/events",
		"GET /api/public/projects/{id}/deployment",
		"GET /api/public/projects/{id}/ai-workflow",
		"GET /api/public/projects/{id}/workflow",
		"GET /api/public/projects/{id}/pull-requests",
		"POST /api/public/repo/issues",
		"WS /api/ws",
		"GET /api/projects/{id}/protocol/workflow",
		"GET /api/projects/{id}/protocol/scan",
		"GET /api/projects/{id}/routing",
		"POST /api/projects/{id}/repo-sync",
		"POST /api/disputes",
		"POST /api/proposals",
		"POST /api/proposals/{id}/decision",
		"GET /api/projects/{id}/escrow",
		"GET /api/projects/{id}/payouts",
		"POST /api/projects/{id}/auto-release",
		"GET /api/projects/{id}/deployment",
		"GET /api/projects/{id}/ai-workflow",
		"POST /api/projects/{id}/agent-actions",
		"GET /api/projects/{id}/pull-requests",
		"GET /api/projects/{id}/dashboard",
		"GET /api/workers/me",
		"POST /api/projects/evaluate-price",
		"POST /api/tasks/{id}/accept",
		"POST /api/tasks/{id}/claim",
		"POST /api/wallets/migrations",
		"GET /api/admin/ops-queue",
	} {
		if !endpoints[required] {
			t.Fatalf("manifest missing endpoint %s: %#v", required, payload.Endpoints)
		}
	}
}

func TestPublicLedgerEconomyProofAndEventsRoutesReturnLiveProof(t *testing.T) {
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
		Name:     "Ledger Client",
		Email:    "ledger-client@example.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatal(err)
	}
	project, err := store.CreateProject(context.Background(), auth.User.ID, CreateProjectRequest{
		Title:            "Public economy proof",
		ClientName:       "Ledger Client",
		ClientEmail:      "ledger-client@example.com",
		Brief:            "Create a funded project so the public ledger economy has mint, reserve, treasury, and payout rows.",
		BudgetCents:      200000,
		PaymentMethod:    PaymentPayPal,
		PaymentReference: defaultDevPaymentCode,
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, task := range project.Tasks {
		if task.RequiredWorkerKind != WorkerHuman {
			continue
		}
		if _, err := store.AcceptTask(task.ID, AcceptTaskRequest{
			WorkerKind: WorkerHuman,
			WorkerID:   "github:ledger-builder",
		}); err != nil {
			t.Fatal(err)
		}
		break
	}

	server := NewServer(cfg, store, payments)

	economyResp := httptest.NewRecorder()
	server.Routes().ServeHTTP(economyResp, httptest.NewRequest(http.MethodGet, "/api/public/token-economy", nil))
	if economyResp.Code != http.StatusOK {
		t.Fatalf("token economy status = %d, body = %s", economyResp.Code, economyResp.Body.String())
	}
	var economy PublicTokenEconomyResponse
	if err := json.Unmarshal(economyResp.Body.Bytes(), &economy); err != nil {
		t.Fatal(err)
	}
	if economy.ProtocolVersion != "mergeos.token-economy.v1" || economy.Kind != "token_economy" {
		t.Fatalf("unexpected economy header: %#v", economy)
	}
	if economy.Totals.VerifiedFundingCents != project.BudgetCents || economy.Totals.MintedCents != project.BudgetCents {
		t.Fatalf("economy funding/mint totals = %#v, want %d", economy.Totals, project.BudgetCents)
	}
	if economy.Totals.PlatformFeeCents <= 0 || economy.Totals.ReleasedCents <= 0 || economy.Totals.RemainingReserveCents <= 0 {
		t.Fatalf("economy missing fee/release/reserve totals: %#v", economy.Totals)
	}
	if len(economy.Balances) < 5 || len(economy.Flows) == 0 || len(economy.RecentEntries) == 0 {
		t.Fatalf("economy rows incomplete: balances=%d flows=%d recent=%d", len(economy.Balances), len(economy.Flows), len(economy.RecentEntries))
	}

	proofResp := httptest.NewRecorder()
	server.Routes().ServeHTTP(proofResp, httptest.NewRequest(http.MethodGet, "/api/public/ledger/proof", nil))
	if proofResp.Code != http.StatusOK {
		t.Fatalf("ledger proof status = %d, body = %s", proofResp.Code, proofResp.Body.String())
	}
	var proof PublicLedgerProofResponse
	if err := json.Unmarshal(proofResp.Body.Bytes(), &proof); err != nil {
		t.Fatal(err)
	}
	if !proof.Valid || proof.EntryCount == 0 || proof.VerifiedCount != proof.EntryCount || proof.BrokenCount != 0 {
		t.Fatalf("ledger proof invalid: %#v", proof)
	}
	if len(proof.RootHash) != 64 || len(proof.PublicRootHash) != 64 || proof.ContractReference != proof.PublicRootHash {
		t.Fatalf("ledger proof hashes invalid: root=%q public=%q contract=%q", proof.RootHash, proof.PublicRootHash, proof.ContractReference)
	}
	if len(proof.Entries) != proof.EntryCount || len(proof.Entries[0].PublicHash) != 64 {
		t.Fatalf("ledger proof rows invalid: %#v", proof.Entries)
	}

	eventsResp := httptest.NewRecorder()
	server.Routes().ServeHTTP(eventsResp, httptest.NewRequest(http.MethodGet, "/api/public/ledger/events?limit=2", nil))
	if eventsResp.Code != http.StatusOK {
		t.Fatalf("ledger events status = %d, body = %s", eventsResp.Code, eventsResp.Body.String())
	}
	var events PublicLiveFeedResponse
	if err := json.Unmarshal(eventsResp.Body.Bytes(), &events); err != nil {
		t.Fatal(err)
	}
	if len(events.Items) != 2 {
		t.Fatalf("ledger events = %d, want limit 2: %#v", len(events.Items), events.Items)
	}
	for _, item := range events.Items {
		if !strings.HasPrefix(item.Type, "ledger_") || item.EntryHash == "" {
			t.Fatalf("ledger event item missing ledger proof fields: %#v", item)
		}
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
	for _, task := range project.Tasks {
		if strings.Contains(body, task.ID) {
			t.Fatalf("public marketplace leaked internal task id %q: %s", task.ID, body)
		}
	}

	var payload MarketplaceResponse
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.ProtocolVersion != "mergeos.marketplace.v1" || payload.Kind != "marketplace" {
		t.Fatalf("unexpected marketplace protocol header: %#v", payload)
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
		if strings.TrimSpace(bounty.ClaimID) == "" || bounty.ClaimID == bounty.ID && !strings.Contains(bounty.ClaimID, ":") {
			t.Fatalf("bounty missing public claim id: %#v", bounty)
		}
		if bounty.IssueURL != "" && !strings.HasPrefix(bounty.IssueURL, "http") {
			t.Fatalf("bounty issue URL is not public: %#v", bounty)
		}
		if bounty.SourceRepository != "" && !strings.HasPrefix(bounty.SourceRepository, "http") {
			t.Fatalf("bounty source repository URL is not public: %#v", bounty)
		}
		if bounty.EstimatedHours <= 0 {
			t.Fatalf("bounty missing estimated hours: %#v", bounty)
		}
		if len(bounty.EvidenceRequired) == 0 || !containsString(bounty.EvidenceRequired, "tests") {
			t.Fatalf("bounty missing evidence requirements: %#v", bounty)
		}
	}
	if len(payload.Contributors) != 1 || payload.Contributors[0].EarnedCents == 0 {
		t.Fatalf("contributors missing real paid task data: %#v", payload.Contributors)
	}
	if len(payload.Agents) == 0 || payload.Agents[0].OpenTaskCount == 0 {
		t.Fatalf("agents missing real task demand: %#v", payload.Agents)
	}

	protocolReq := httptest.NewRequest(http.MethodGet, "/api/public/protocol/tasks?limit=20", nil)
	protocolResp := httptest.NewRecorder()
	server.Routes().ServeHTTP(protocolResp, protocolReq)
	if protocolResp.Code != http.StatusOK {
		t.Fatalf("task protocol status = %d, body = %s", protocolResp.Code, protocolResp.Body.String())
	}
	protocolBody := protocolResp.Body.String()
	for _, value := range []string{"client@example.com", "+1 555 0101", auth.User.ID, tempDir, defaultDevPaymentCode} {
		if strings.Contains(protocolBody, value) {
			t.Fatalf("task protocol leaked private value %q: %s", value, protocolBody)
		}
	}
	for _, task := range project.Tasks {
		if strings.Contains(protocolBody, task.ID) {
			t.Fatalf("task protocol leaked internal task id %q: %s", task.ID, protocolBody)
		}
	}
	var taskProtocol PublicTaskProtocolResponse
	if err := json.Unmarshal(protocolResp.Body.Bytes(), &taskProtocol); err != nil {
		t.Fatal(err)
	}
	if taskProtocol.Stats.OpenTaskCount != payload.Stats.OpenTaskCount || len(taskProtocol.Tasks) != len(payload.Bounties) {
		t.Fatalf("unexpected task protocol feed: %#v", taskProtocol)
	}
	for _, document := range taskProtocol.Tasks {
		if document.ProtocolVersion != "mergeos.task.v1" || document.Kind != "task" || document.ID == "" {
			t.Fatalf("invalid task protocol header: %#v", document)
		}
		if document.RewardMRG <= 0 || len(document.AcceptanceCriteria) == 0 || len(document.EvidenceRequired) == 0 {
			t.Fatalf("task protocol missing bounty requirements: %#v", document)
		}
		if document.EstimatedHours <= 0 {
			t.Fatalf("task protocol missing estimated hours: %#v", document)
		}
		if document.Complexity == "" || document.RiskLevel == "" {
			t.Fatalf("task protocol missing AI analysis fields: %#v", document)
		}
		if document.SourceRepository == "" || !strings.HasPrefix(document.SourceRepository, "http") {
			t.Fatalf("task protocol missing public source repository: %#v", document)
		}
	}
	filteredTaskReq := httptest.NewRequest(http.MethodGet, "/api/public/protocol/tasks?task_id="+url.QueryEscape(taskProtocol.Tasks[0].ID), nil)
	filteredTaskResp := httptest.NewRecorder()
	server.Routes().ServeHTTP(filteredTaskResp, filteredTaskReq)
	if filteredTaskResp.Code != http.StatusOK {
		t.Fatalf("filtered task protocol status = %d, body = %s", filteredTaskResp.Code, filteredTaskResp.Body.String())
	}
	var filteredTaskProtocol PublicTaskProtocolResponse
	if err := json.Unmarshal(filteredTaskResp.Body.Bytes(), &filteredTaskProtocol); err != nil {
		t.Fatal(err)
	}
	if len(filteredTaskProtocol.Tasks) != 1 || filteredTaskProtocol.Tasks[0].ID != taskProtocol.Tasks[0].ID {
		t.Fatalf("filtered task protocol returned wrong task: %#v", filteredTaskProtocol)
	}

	agentReq := httptest.NewRequest(http.MethodGet, "/api/public/protocol/agents?limit=20", nil)
	agentResp := httptest.NewRecorder()
	server.Routes().ServeHTTP(agentResp, agentReq)
	if agentResp.Code != http.StatusOK {
		t.Fatalf("agent protocol status = %d, body = %s", agentResp.Code, agentResp.Body.String())
	}
	agentBody := agentResp.Body.String()
	for _, value := range []string{"client@example.com", "+1 555 0101", auth.User.ID, tempDir, defaultDevPaymentCode} {
		if strings.Contains(agentBody, value) {
			t.Fatalf("agent protocol leaked private value %q: %s", value, agentBody)
		}
	}
	for _, task := range project.Tasks {
		if strings.Contains(agentBody, task.ID) {
			t.Fatalf("agent protocol leaked internal task id %q: %s", task.ID, agentBody)
		}
	}
	var agentProtocol PublicAgentProtocolResponse
	if err := json.Unmarshal(agentResp.Body.Bytes(), &agentProtocol); err != nil {
		t.Fatal(err)
	}
	if agentProtocol.Stats.OpenTaskCount != payload.Stats.OpenTaskCount || len(agentProtocol.Agents) != len(payload.Agents) {
		t.Fatalf("unexpected agent protocol feed: %#v", agentProtocol)
	}
	foundCEOAgent := false
	foundDesignSubagent := false
	for _, document := range agentProtocol.Agents {
		if document.ProtocolVersion != "mergeos.agent.v1" || document.Kind != "agent" || document.ID == "" {
			t.Fatalf("invalid agent protocol header: %#v", document)
		}
		if len(document.SupportedActions) == 0 || len(document.Capabilities) == 0 {
			t.Fatalf("agent protocol missing routing metadata: %#v", document)
		}
		if document.OpenTaskCount > 0 && len(document.OpenTaskIDs) == 0 && document.Type != ceoAgentType {
			t.Fatalf("task-bearing agent protocol missing open task ids: %#v", document)
		}
		if document.Metadata["event_protocol"] != "mergeos.event.v1" || document.Metadata["event_stream_endpoint"] != "WS /api/ws" || int(document.Metadata["queue_depth"].(float64)) != len(document.OpenTaskIDs) {
			t.Fatalf("agent protocol missing event routing metadata: %#v", document.Metadata)
		}
		if document.Type == ceoAgentType {
			depth, _ := document.Metadata["queue_depth"].(float64)
			foundCEOAgent = document.Role == "ceo_planner" && containsString(document.SubagentTypes, designReviewAgentType) && len(document.OpenTaskIDs) > 0 && int(depth) > 0
		}
		if document.Type == designReviewAgentType {
			depth, _ := document.Metadata["queue_depth"].(float64)
			foundDesignSubagent = document.Role == "subagent" && document.ParentAgentType == ceoAgentType && containsString(document.Focus, "visual_quality") && len(document.OpenTaskIDs) > 0 && int(depth) > 0
		}
	}
	if !foundCEOAgent || !foundDesignSubagent {
		t.Fatalf("agent protocol missing CEO planner or design-review subagent: %#v", agentProtocol.Agents)
	}
	queueReq := httptest.NewRequest(http.MethodGet, "/api/public/protocol/agent-queue?limit=20", nil)
	queueResp := httptest.NewRecorder()
	server.Routes().ServeHTTP(queueResp, queueReq)
	if queueResp.Code != http.StatusOK {
		t.Fatalf("agent queue status = %d, body = %s", queueResp.Code, queueResp.Body.String())
	}
	queueBody := queueResp.Body.String()
	for _, value := range []string{"client@example.com", "+1 555 0101", auth.User.ID, tempDir, defaultDevPaymentCode} {
		if strings.Contains(queueBody, value) {
			t.Fatalf("agent queue leaked private value %q: %s", value, queueBody)
		}
	}
	for _, task := range project.Tasks {
		if strings.Contains(queueBody, task.ID) {
			t.Fatalf("agent queue leaked internal task id %q: %s", task.ID, queueBody)
		}
	}
	var queueProtocol AgentQueueResponse
	if err := json.Unmarshal(queueResp.Body.Bytes(), &queueProtocol); err != nil {
		t.Fatal(err)
	}
	if queueProtocol.ProtocolVersion != "mergeos.agent-queue.v1" || queueProtocol.Kind != "agent_queue" {
		t.Fatalf("unexpected agent queue header: %#v", queueProtocol)
	}
	if queueProtocol.Stats.ReadyCount == 0 || len(queueProtocol.Tasks) == 0 || len(queueProtocol.Agents) == 0 {
		t.Fatalf("agent queue missing ready work: %#v", queueProtocol)
	}
	firstPacket := queueProtocol.Tasks[0].WorkPacket
	for _, required := range []string{"task_protocol", "agent_queue", "workflow_protocol", "workflow_pulse", "pr_monitor", "ceo_agent", "design_review"} {
		if strings.TrimSpace(firstPacket.ContextURLs[required]) == "" {
			t.Fatalf("agent work packet missing context URL %s: %#v", required, firstPacket)
		}
	}
	if firstPacket.SupervisorAgentType != ceoAgentType || firstPacket.DesignReviewAgent != designReviewAgentType || len(firstPacket.DelegationChain) < 2 || firstPacket.DelegationChain[0] != ceoAgentType || firstPacket.DelegationChain[1] != designReviewAgentType {
		t.Fatalf("agent work packet missing CEO/design delegation chain: %#v", firstPacket)
	}
	if !strings.HasPrefix(firstPacket.ContextURLs["task_protocol"], "/api/public/protocol/tasks?task_id=") {
		t.Fatalf("agent work packet task protocol is not task scoped: %#v", firstPacket.ContextURLs)
	}
	if !strings.HasSuffix(firstPacket.ClaimEndpoint, "/claim") {
		t.Fatalf("agent work packet claim endpoint should use claim alias: %#v", firstPacket)
	}
	if firstPacket.ClaimEndpoint == "" || firstPacket.ActionEndpoint == "" || len(firstPacket.Runbook) < 4 || len(firstPacket.ActionPayloads) == 0 {
		t.Fatalf("agent work packet missing executable details: %#v", firstPacket)
	}

	contributorReq := httptest.NewRequest(http.MethodGet, "/api/public/protocol/contributors?limit=20", nil)
	contributorResp := httptest.NewRecorder()
	server.Routes().ServeHTTP(contributorResp, contributorReq)
	if contributorResp.Code != http.StatusOK {
		t.Fatalf("contributor protocol status = %d, body = %s", contributorResp.Code, contributorResp.Body.String())
	}
	contributorBody := contributorResp.Body.String()
	for _, value := range []string{"client@example.com", "+1 555 0101", auth.User.ID, tempDir, defaultDevPaymentCode} {
		if strings.Contains(contributorBody, value) {
			t.Fatalf("contributor protocol leaked private value %q: %s", value, contributorBody)
		}
	}
	for _, task := range project.Tasks {
		if strings.Contains(contributorBody, task.ID) {
			t.Fatalf("contributor protocol leaked internal task id %q: %s", task.ID, contributorBody)
		}
	}
	var contributorProtocol PublicContributorProtocolResponse
	if err := json.Unmarshal(contributorResp.Body.Bytes(), &contributorProtocol); err != nil {
		t.Fatal(err)
	}
	if contributorProtocol.Stats.OpenTaskCount != payload.Stats.OpenTaskCount || len(contributorProtocol.Contributors) != len(payload.Contributors) {
		t.Fatalf("unexpected contributor protocol feed: %#v", contributorProtocol)
	}
	for _, document := range contributorProtocol.Contributors {
		if document.ProtocolVersion != "mergeos.contributor.v1" || document.Kind != "contributor" || document.ID == "" {
			t.Fatalf("invalid contributor protocol header: %#v", document)
		}
		if document.WorkerID == "" || document.DisplayName == "" || document.CompletedTaskCount == 0 || document.EarnedMRG <= 0 || len(document.Capabilities) == 0 {
			t.Fatalf("contributor protocol missing reputation data: %#v", document)
		}
		if document.Metadata["task_protocol_endpoint"] != "GET /api/public/protocol/tasks" || document.Metadata["event_stream_endpoint"] != "WS /api/ws" {
			t.Fatalf("contributor protocol missing routing metadata: %#v", document.Metadata)
		}
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

	protocolReq := httptest.NewRequest(http.MethodGet, "/api/public/protocol/ledger", nil)
	protocolResp := httptest.NewRecorder()
	server.Routes().ServeHTTP(protocolResp, protocolReq)
	if protocolResp.Code != http.StatusOK {
		t.Fatalf("public protocol ledger status = %d, body = %s", protocolResp.Code, protocolResp.Body.String())
	}
	protocolBody := protocolResp.Body.String()
	for _, value := range privateValues {
		if strings.Contains(protocolBody, value) {
			t.Fatalf("public protocol ledger leaked private value %q: %s", value, protocolBody)
		}
	}
	var protocolPayload LedgerProtocolResponse
	if err := json.Unmarshal(protocolResp.Body.Bytes(), &protocolPayload); err != nil {
		t.Fatal(err)
	}
	if protocolPayload.ProtocolVersion != "mergeos.ledger.v1" || protocolPayload.Kind != "ledger" || protocolPayload.TokenSymbol != defaultTokenSymbol {
		t.Fatalf("unexpected public ledger protocol header: %#v", protocolPayload)
	}
	if !protocolPayload.Verification.Valid || protocolPayload.Verification.EntryCount != len(store.ListLedger()) {
		t.Fatalf("unexpected public ledger protocol verification: %#v", protocolPayload.Verification)
	}
	if len(protocolPayload.Entries) != len(payload) {
		t.Fatalf("public ledger protocol entries = %d, want %d", len(protocolPayload.Entries), len(payload))
	}
}

func TestPublicLedgerVerifyRouteDetectsTampering(t *testing.T) {
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
		Name:     "Verify Client",
		Email:    "verify-client@example.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateProject(context.Background(), auth.User.ID, CreateProjectRequest{
		Title:            "Ledger verify proof",
		ClientName:       "Verify Client",
		ClientEmail:      "verify-client@example.com",
		Brief:            "Create ledger entries for hash-chain verification.",
		BudgetCents:      120000,
		PaymentMethod:    PaymentPayPal,
		PaymentReference: defaultDevPaymentCode,
	}); err != nil {
		t.Fatal(err)
	}

	server := NewServer(cfg, store, payments)
	req := httptest.NewRequest(http.MethodGet, "/api/public/ledger/verify", nil)
	resp := httptest.NewRecorder()
	server.Routes().ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("ledger verify status = %d, body = %s", resp.Code, resp.Body.String())
	}
	var validPayload LedgerVerificationResponse
	if err := json.Unmarshal(resp.Body.Bytes(), &validPayload); err != nil {
		t.Fatal(err)
	}
	if !validPayload.Valid || validPayload.EntryCount == 0 || validPayload.LastHash == "" {
		t.Fatalf("expected valid ledger verification: %#v", validPayload)
	}

	store.mu.Lock()
	store.ledger[0].AmountCents++
	store.mu.Unlock()

	tamperedReq := httptest.NewRequest(http.MethodGet, "/api/public/ledger/verify", nil)
	tamperedResp := httptest.NewRecorder()
	server.Routes().ServeHTTP(tamperedResp, tamperedReq)
	if tamperedResp.Code != http.StatusOK {
		t.Fatalf("tampered ledger verify status = %d, body = %s", tamperedResp.Code, tamperedResp.Body.String())
	}
	var tamperedPayload LedgerVerificationResponse
	if err := json.Unmarshal(tamperedResp.Body.Bytes(), &tamperedPayload); err != nil {
		t.Fatal(err)
	}
	if tamperedPayload.Valid || tamperedPayload.BrokenSequence != 1 || !strings.Contains(tamperedPayload.Error, "hash") {
		t.Fatalf("expected tampered ledger verification failure: %#v", tamperedPayload)
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
	if payload.ProtocolVersion != "mergeos.live-feed.v1" || payload.Kind != "live_feed" {
		t.Fatalf("unexpected live feed protocol header: %#v", payload)
	}
	if payload.Stats.ProjectCount != 1 || payload.Stats.AIActionCount != 1 || payload.Stats.LedgerEntryCount == 0 {
		t.Fatalf("unexpected live feed stats: %#v", payload.Stats)
	}
	if payload.Stats.ActiveContributorCount == 0 || payload.Stats.ActiveAgentCount == 0 {
		t.Fatalf("live feed missing active actor stats: %#v", payload.Stats)
	}
	seen := map[string]bool{}
	seenEvidence := map[string]bool{}
	for _, item := range payload.Items {
		seen[item.Type] = true
		if (item.Type == "task_opened" || item.Type == "task_accepted") && containsString(item.EvidenceRequired, "tests") {
			seenEvidence[item.Type] = true
		}
		if item.Type == "ledger_task_payment" {
			if item.Reference != "pr:https://github.com/mergeos-bounties/mergeos/pull/151;title:Live feed proof" {
				t.Fatalf("task payout feed reference = %q", item.Reference)
			}
			if item.URL != "https://github.com/mergeos-bounties/mergeos/pull/151" {
				t.Fatalf("task payout feed url = %q", item.URL)
			}
			if item.LedgerSequence <= 0 || len(item.EntryHash) != 64 {
				t.Fatalf("task payout feed missing ledger proof fields: %#v", item)
			}
		}
	}
	for _, required := range []string{"project_funded", "deployment_validation", "task_accepted", "ledger_task_payment", "pr_opened"} {
		if !seen[required] {
			t.Fatalf("live feed missing %s item: %#v", required, payload.Items)
		}
	}
	for _, required := range []string{"task_opened", "task_accepted"} {
		if !seenEvidence[required] {
			t.Fatalf("live feed %s missing evidence requirements: %#v", required, payload.Items)
		}
	}

	protocolReq := httptest.NewRequest(http.MethodGet, "/api/public/protocol/events?limit=50", nil)
	protocolResp := httptest.NewRecorder()
	server.Routes().ServeHTTP(protocolResp, protocolReq)
	if protocolResp.Code != http.StatusOK {
		t.Fatalf("public protocol events status = %d, body = %s", protocolResp.Code, protocolResp.Body.String())
	}
	protocolBody := protocolResp.Body.String()
	for _, value := range privateValues {
		if strings.Contains(protocolBody, value) {
			t.Fatalf("public protocol events leaked private value %q: %s", value, protocolBody)
		}
	}
	var eventFeed PublicEventProtocolResponse
	if err := json.Unmarshal(protocolResp.Body.Bytes(), &eventFeed); err != nil {
		t.Fatal(err)
	}
	if eventFeed.Stats.ProjectCount != payload.Stats.ProjectCount || len(eventFeed.Events) == 0 {
		t.Fatalf("unexpected protocol event feed: %#v", eventFeed)
	}
	eventTypes := map[string]bool{}
	for _, event := range eventFeed.Events {
		if event.ProtocolVersion != "mergeos.event.v1" || event.Kind != "event" || event.Actor == "" || event.OccurredAt.IsZero() {
			t.Fatalf("invalid protocol event header: %#v", event)
		}
		eventTypes[event.Type] = true
		if event.Type == "task.paid" && (event.AmountMRG == nil || *event.AmountMRG <= 0) {
			t.Fatalf("task paid event missing amount: %#v", event)
		}
		if event.Type == "task.paid" {
			if event.Payload["ledger_sequence"] == nil || len(fmt.Sprint(event.Payload["entry_hash"])) != 64 {
				t.Fatalf("task paid event missing ledger proof payload: %#v", event)
			}
		}
		if (event.Type == "task.created" || event.Type == "task.claimed") && !protocolPayloadStringSliceContains(event.Payload["evidence_required"], "tests") {
			t.Fatalf("task event missing evidence requirements: %#v", event)
		}
	}
	for _, required := range []string{"project.funded", "deployment.updated", "task.claimed", "task.paid", "pr.opened"} {
		if !eventTypes[required] {
			t.Fatalf("protocol events missing %s item: %#v", required, eventFeed.Events)
		}
	}
}

func TestProjectDeploymentRouteReturnsDerivedStatusAndSanitizesData(t *testing.T) {
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
		Name:        "Deploy Client",
		CompanyName: "Deploy Co",
		Email:       "deploy@example.com",
		Password:    "password123",
	})
	if err != nil {
		t.Fatal(err)
	}
	project, err := store.CreateProject(context.Background(), auth.User.ID, CreateProjectRequest{
		Title:            "Deployment proof",
		ClientName:       "Private Deploy Client",
		CompanyName:      "Deploy Co",
		ClientEmail:      "deploy@example.com",
		Phone:            "+1 555 0199",
		Brief:            "Create deployment validation data without leaking private customer data.",
		BudgetCents:      210000,
		PaymentMethod:    PaymentPayPal,
		PaymentReference: defaultDevPaymentCode,
	})
	if err != nil {
		t.Fatal(err)
	}

	var deployTask *Task
	for _, task := range project.Tasks {
		if strings.Contains(strings.ToLower(task.Title+" "+task.Acceptance+" "+task.SuggestedAgentType), "deploy") {
			deployTask = task
			break
		}
	}
	if deployTask == nil {
		t.Fatal("project did not create a deployment task")
	}
	req, err := acceptRequestForPullAuthor(deployTask, "deploy-author")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.AcceptTask(deployTask.ID, req); err != nil {
		t.Fatal(err)
	}
	if err := store.AddGeminiWebhookLog(GeminiWebhookLog{
		EventName:  "pull_request",
		Action:     "synchronize",
		Repository: project.BountyRepoName,
		PullNumber: 211,
		Sender:     "deploy-author",
		Status:     "processed",
		StatusCode: http.StatusOK,
		CommentURL: "https://github.com/mergeos-bounties/mergeos/pull/211#issuecomment-2",
		ReceivedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatal(err)
	}

	server := NewServer(cfg, store, payments)
	reqHTTP := httptest.NewRequest(http.MethodGet, "/api/projects/"+project.ID+"/deployment", nil)
	reqHTTP.Header.Set("Authorization", "Bearer "+auth.Token)
	resp := httptest.NewRecorder()
	server.Routes().ServeHTTP(resp, reqHTTP)
	if resp.Code != http.StatusOK {
		t.Fatalf("deployment status = %d, body = %s", resp.Code, resp.Body.String())
	}

	body := resp.Body.String()
	for _, value := range []string{
		"deploy@example.com",
		"+1 555 0199",
		auth.User.ID,
		defaultDevPaymentCode,
		tempDir,
		deployTask.ID,
	} {
		if strings.Contains(body, value) {
			t.Fatalf("deployment response leaked private value %q: %s", value, body)
		}
	}

	var payload ProjectDeploymentResponse
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.ProtocolVersion != "mergeos.deployment.v1" || payload.Kind != "deployment" {
		t.Fatalf("unexpected deployment protocol header: %#v", payload)
	}
	if payload.ProjectID != project.ID || payload.Status != "validating" || payload.Progress == 0 {
		t.Fatalf("unexpected deployment summary: %#v", payload)
	}
	seenStages := map[string]bool{}
	for _, stage := range payload.Stages {
		seenStages[stage.ID] = true
		if stage.ID == "deployment_handoff" && stage.Status != deploymentStageComplete {
			t.Fatalf("deployment handoff stage was not complete: %#v", stage)
		}
	}
	for _, required := range []string{"repo_handoff", "task_routing", "qa_validation", "deployment_handoff", "release_gate"} {
		if !seenStages[required] {
			t.Fatalf("deployment response missing stage %s: %#v", required, payload.Stages)
		}
	}
	if len(payload.Signals) == 0 {
		t.Fatalf("deployment response missing ledger/AI signals: %#v", payload.Signals)
	}

	otherAuth, err := store.Register(RegisterRequest{
		Name:     "Other Client",
		Email:    "other-client@example.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatal(err)
	}
	forbiddenReq := httptest.NewRequest(http.MethodGet, "/api/projects/"+project.ID+"/deployment", nil)
	forbiddenReq.Header.Set("Authorization", "Bearer "+otherAuth.Token)
	forbiddenResp := httptest.NewRecorder()
	server.Routes().ServeHTTP(forbiddenResp, forbiddenReq)
	if forbiddenResp.Code != http.StatusForbidden {
		t.Fatalf("other client deployment status = %d", forbiddenResp.Code)
	}
}

func TestProjectDeploymentUsesDeploymentAgentAction(t *testing.T) {
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
		Name:        "Deploy Agent Client",
		CompanyName: "Deploy Agent Co",
		Email:       "deploy-agent@example.com",
		Password:    "password123",
	})
	if err != nil {
		t.Fatal(err)
	}
	project, err := store.CreateProject(context.Background(), auth.User.ID, CreateProjectRequest{
		Title:            "Deployment agent proof",
		ClientName:       "Private Deploy Agent Client",
		CompanyName:      "Deploy Agent Co",
		ClientEmail:      "deploy-agent@example.com",
		Phone:            "+1 555 0188",
		Brief:            "Create deployment agent handoff data without leaking private customer data.",
		BudgetCents:      210000,
		PaymentMethod:    PaymentPayPal,
		PaymentReference: defaultDevPaymentCode,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.RecordProjectAgentAction(project.ID, AgentActionRequest{
		Action:         "deploy",
		AgentType:      "deployment-agent",
		Status:         "processed",
		ReferenceURL:   "https://vercel.example/deployments/mergeos-preview",
		DurationMillis: 42000,
	}); err != nil {
		t.Fatal(err)
	}

	server := NewServer(cfg, store, payments)
	reqHTTP := httptest.NewRequest(http.MethodGet, "/api/projects/"+project.ID+"/deployment", nil)
	reqHTTP.Header.Set("Authorization", "Bearer "+auth.Token)
	resp := httptest.NewRecorder()
	server.Routes().ServeHTTP(resp, reqHTTP)
	if resp.Code != http.StatusOK {
		t.Fatalf("deployment status = %d, body = %s", resp.Code, resp.Body.String())
	}
	body := resp.Body.String()
	for _, value := range []string{
		"deploy-agent@example.com",
		"+1 555 0188",
		auth.User.ID,
		defaultDevPaymentCode,
		tempDir,
	} {
		if strings.Contains(body, value) {
			t.Fatalf("deployment response leaked private value %q: %s", value, body)
		}
	}

	var payload ProjectDeploymentResponse
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	foundDeployStage := false
	foundDeploySignal := false
	for _, stage := range payload.Stages {
		if stage.ID == "deployment_handoff" {
			foundDeployStage = true
			if stage.Status != deploymentStageComplete || stage.URL != "https://vercel.example/deployments/mergeos-preview" {
				t.Fatalf("deployment agent did not complete handoff stage: %#v", stage)
			}
		}
	}
	for _, signal := range payload.Signals {
		if signal.Type == "agent_action" && signal.Status == "processed" && signal.URL == "https://vercel.example/deployments/mergeos-preview" {
			foundDeploySignal = true
			break
		}
	}
	if !foundDeployStage || !foundDeploySignal {
		t.Fatalf("deployment response missing deploy agent evidence: stage=%t signal=%t payload=%#v", foundDeployStage, foundDeploySignal, payload)
	}
}

func TestPublicProjectDeploymentRouteReturnsSanitizedReadiness(t *testing.T) {
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
		Name:        "Public Deploy Client",
		CompanyName: "Public Deploy Co",
		Email:       "public-deploy@example.com",
		Password:    "password123",
	})
	if err != nil {
		t.Fatal(err)
	}
	project, err := store.CreateProject(context.Background(), auth.User.ID, CreateProjectRequest{
		Title:            "Public deployment readiness",
		ClientName:       "Private Public Deploy Client",
		CompanyName:      "Public Deploy Co",
		ClientEmail:      "public-deploy@example.com",
		Phone:            "+1 555 0166",
		Brief:            "Expose public release readiness without leaking private customer data.",
		BudgetCents:      210000,
		PaymentMethod:    PaymentPayPal,
		PaymentReference: defaultDevPaymentCode,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.RecordProjectAgentAction(project.ID, AgentActionRequest{
		Action:         "deploy",
		AgentType:      "deployment-agent",
		Status:         "processed",
		ReferenceURL:   "https://vercel.example/deployments/public-readiness",
		DurationMillis: 24000,
	}); err != nil {
		t.Fatal(err)
	}

	server := NewServer(cfg, store, payments)
	reqHTTP := httptest.NewRequest(http.MethodGet, "/api/public/projects/"+project.ID+"/deployment", nil)
	resp := httptest.NewRecorder()
	server.Routes().ServeHTTP(resp, reqHTTP)
	if resp.Code != http.StatusOK {
		t.Fatalf("public deployment status = %d, body = %s", resp.Code, resp.Body.String())
	}
	body := resp.Body.String()
	for _, value := range []string{
		"public-deploy@example.com",
		"+1 555 0166",
		auth.User.ID,
		defaultDevPaymentCode,
		tempDir,
	} {
		if strings.Contains(body, value) {
			t.Fatalf("public deployment response leaked private value %q: %s", value, body)
		}
	}

	var payload ProjectDeploymentResponse
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.ProtocolVersion != "mergeos.deployment.v1" || payload.Kind != "deployment" {
		t.Fatalf("unexpected public deployment protocol header: %#v", payload)
	}
	if payload.ProjectID != project.ID || payload.Progress == 0 {
		t.Fatalf("unexpected public deployment summary: %#v", payload)
	}
	foundDeploySignal := false
	for _, signal := range payload.Signals {
		if signal.Type == "agent_action" && signal.URL == "https://vercel.example/deployments/public-readiness" {
			foundDeploySignal = true
			break
		}
	}
	if !foundDeploySignal {
		t.Fatalf("public deployment response missing deploy agent signal: %#v", payload.Signals)
	}
}

func TestProjectEscrowRouteReturnsReserveReleaseSummary(t *testing.T) {
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
		Name:        "Escrow Client",
		CompanyName: "Escrow Co",
		Email:       "escrow-client@example.com",
		Password:    "password123",
	})
	if err != nil {
		t.Fatal(err)
	}
	project, err := store.CreateProject(context.Background(), auth.User.ID, CreateProjectRequest{
		Title:            "Escrow proof",
		ClientName:       "Private Escrow Client",
		CompanyName:      "Escrow Co",
		ClientEmail:      "escrow-client@example.com",
		Phone:            "+1 555 0144",
		Brief:            "Create escrow release data without leaking payment references.",
		BudgetCents:      180000,
		PaymentMethod:    PaymentPayPal,
		PaymentReference: defaultDevPaymentCode,
	})
	if err != nil {
		t.Fatal(err)
	}
	task := project.Tasks[0]
	acceptedReward := task.RewardCents
	req, err := acceptRequestForPullAuthor(task, "escrow-author")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.AcceptTask(task.ID, req); err != nil {
		t.Fatal(err)
	}

	server := NewServer(cfg, store, payments)
	reqHTTP := httptest.NewRequest(http.MethodGet, "/api/projects/"+project.ID+"/escrow", nil)
	reqHTTP.Header.Set("Authorization", "Bearer "+auth.Token)
	resp := httptest.NewRecorder()
	server.Routes().ServeHTTP(resp, reqHTTP)
	if resp.Code != http.StatusOK {
		t.Fatalf("escrow status = %d, body = %s", resp.Code, resp.Body.String())
	}

	body := resp.Body.String()
	for _, value := range []string{
		"escrow-client@example.com",
		"+1 555 0144",
		defaultDevPaymentCode,
		tempDir,
	} {
		if strings.Contains(body, value) {
			t.Fatalf("escrow response leaked private value %q: %s", value, body)
		}
	}

	var payload ProjectEscrowResponse
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.ProtocolVersion != "mergeos.escrow.v1" || payload.Kind != "escrow" {
		t.Fatalf("unexpected escrow protocol header: %#v", payload)
	}
	if payload.ProjectID != project.ID || payload.ReleaseStatus != "releasing" || payload.WorkPoolCents != project.WorkPoolCents {
		t.Fatalf("unexpected escrow summary: %#v", payload)
	}
	if payload.ProjectReserveCents != project.WorkPoolCents || payload.TaskReserveCents != project.WorkPoolCents {
		t.Fatalf("unexpected escrow reserves: %#v", payload)
	}
	if payload.TaskPaymentCents != acceptedReward || payload.ReleasedCents != acceptedReward || payload.RemainingCents != project.WorkPoolCents-acceptedReward {
		t.Fatalf("unexpected escrow release totals: %#v", payload)
	}
	if payload.PaidTaskCount != 1 || payload.OpenTaskCount != len(project.Tasks)-1 || len(payload.Tasks) != len(project.Tasks) {
		t.Fatalf("unexpected escrow task counts: %#v", payload)
	}
	foundReleasedTask := false
	for _, row := range payload.Tasks {
		if row.TaskID == task.ID {
			foundReleasedTask = row.ReleaseStatus == "released" && row.PaidCents == acceptedReward && row.WorkerID == "github:escrow-author"
			break
		}
	}
	if !foundReleasedTask {
		t.Fatalf("escrow response missing released task row: %#v", payload.Tasks)
	}

	otherAuth, err := store.Register(RegisterRequest{
		Name:     "Other Escrow Client",
		Email:    "other-escrow-client@example.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatal(err)
	}
	forbiddenReq := httptest.NewRequest(http.MethodGet, "/api/projects/"+project.ID+"/escrow", nil)
	forbiddenReq.Header.Set("Authorization", "Bearer "+otherAuth.Token)
	forbiddenResp := httptest.NewRecorder()
	server.Routes().ServeHTTP(forbiddenResp, forbiddenReq)
	if forbiddenResp.Code != http.StatusForbidden {
		t.Fatalf("other client escrow status = %d", forbiddenResp.Code)
	}
}

func TestProjectPayoutsRouteReturnsSettlementContractAndSanitizesData(t *testing.T) {
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
		Name:        "Payout Client",
		CompanyName: "Payout Co",
		Email:       "payout-client@example.com",
		Password:    "password123",
	})
	if err != nil {
		t.Fatal(err)
	}
	project, err := store.CreateProject(context.Background(), auth.User.ID, CreateProjectRequest{
		Title:            "Payout proof",
		ClientName:       "Private Payout Client",
		CompanyName:      "Payout Co",
		ClientEmail:      "payout-client@example.com",
		Phone:            "+1 555 0190",
		Brief:            "Create payout settlement data without leaking payment references.",
		BudgetCents:      190000,
		PaymentMethod:    PaymentPayPal,
		PaymentReference: defaultDevPaymentCode,
	})
	if err != nil {
		t.Fatal(err)
	}
	task := project.Tasks[0]
	acceptedReward := task.RewardCents
	req, err := acceptRequestForPullAuthor(task, "payout-author")
	if err != nil {
		t.Fatal(err)
	}
	pullReference := buildPullLedgerReference(task.ID, "https://github.com/mergeos-bounties/mergeos/pull/190", "Payout proof")
	if _, err := store.AcceptTaskWithReviewReference(task.ID, req, 0, "", pullReference); err != nil {
		t.Fatal(err)
	}

	server := NewServer(cfg, store, payments)
	reqHTTP := httptest.NewRequest(http.MethodGet, "/api/projects/"+project.ID+"/payouts", nil)
	reqHTTP.Header.Set("Authorization", "Bearer "+auth.Token)
	resp := httptest.NewRecorder()
	server.Routes().ServeHTTP(resp, reqHTTP)
	if resp.Code != http.StatusOK {
		t.Fatalf("payouts status = %d, body = %s", resp.Code, resp.Body.String())
	}

	body := resp.Body.String()
	for _, value := range []string{
		"payout-client@example.com",
		"+1 555 0190",
		auth.User.ID,
		defaultDevPaymentCode,
		tempDir,
	} {
		if strings.Contains(body, value) {
			t.Fatalf("payouts response leaked private value %q: %s", value, body)
		}
	}

	var payload ProjectPayoutsResponse
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.ProtocolVersion != "mergeos.payouts.v1" || payload.Kind != "payouts" {
		t.Fatalf("unexpected payouts protocol header: %#v", payload)
	}
	if payload.ProjectID != project.ID || payload.ReleaseStatus != "releasing" || payload.ReleasedCents != acceptedReward || payload.ReleaseCount != 1 {
		t.Fatalf("unexpected payouts summary: %#v", payload)
	}
	if payload.PaidTaskCount != 1 || payload.OpenTaskCount != len(project.Tasks)-1 || len(payload.Payouts) != len(project.Tasks) {
		t.Fatalf("unexpected payout counts: %#v", payload)
	}
	var paidRow *ProjectPayoutRow
	for index := range payload.Payouts {
		if payload.Payouts[index].TaskID == task.ID {
			paidRow = &payload.Payouts[index]
			break
		}
	}
	if paidRow == nil {
		t.Fatalf("payouts response missing paid task row: %#v", payload.Payouts)
	}
	if paidRow.Type != "task_payment" || paidRow.ReleaseStatus != "released" || paidRow.PaidCents != acceptedReward || paidRow.RemainingCents != 0 {
		t.Fatalf("unexpected paid payout row: %#v", paidRow)
	}
	if paidRow.WorkerID != "github:payout-author" || paidRow.PayoutAccount != "github:payout-author" {
		t.Fatalf("unexpected payout worker/account: %#v", paidRow)
	}
	if paidRow.LedgerSequence <= 0 || paidRow.LedgerEntryCount != 1 || len(paidRow.EntryHash) != 64 || paidRow.ProofHash != paidRow.EntryHash || paidRow.ReleasedAt == nil {
		t.Fatalf("payout row missing ledger proof: %#v", paidRow)
	}
	if paidRow.Reference != "pr:https://github.com/mergeos-bounties/mergeos/pull/190;title:Payout proof" || paidRow.URL != "https://github.com/mergeos-bounties/mergeos/pull/190" {
		t.Fatalf("unexpected payout proof reference: %#v", paidRow)
	}

	otherAuth, err := store.Register(RegisterRequest{
		Name:     "Other Payout Client",
		Email:    "other-payout-client@example.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatal(err)
	}
	forbiddenReq := httptest.NewRequest(http.MethodGet, "/api/projects/"+project.ID+"/payouts", nil)
	forbiddenReq.Header.Set("Authorization", "Bearer "+otherAuth.Token)
	forbiddenResp := httptest.NewRecorder()
	server.Routes().ServeHTTP(forbiddenResp, forbiddenReq)
	if forbiddenResp.Code != http.StatusForbidden {
		t.Fatalf("other client payouts status = %d", forbiddenResp.Code)
	}
}

func TestProjectAutoReleaseRouteReleasesReadyCandidateAndRecordsPolicy(t *testing.T) {
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
		Name:        "Auto Release Client",
		CompanyName: "Auto Co",
		Email:       "auto-release-client@example.com",
		Password:    "password123",
	})
	if err != nil {
		t.Fatal(err)
	}
	project, err := store.CreateProject(context.Background(), auth.User.ID, CreateProjectRequest{
		Title:            "Auto-release proof",
		ClientName:       "Private Auto Client",
		CompanyName:      "Auto Co",
		ClientEmail:      "auto-release-client@example.com",
		Phone:            "+1 555 0188",
		Brief:            "Release low-risk PR payouts automatically without leaking payment references.",
		BudgetCents:      190000,
		PaymentMethod:    PaymentPayPal,
		PaymentReference: defaultDevPaymentCode,
	})
	if err != nil {
		t.Fatal(err)
	}
	task := project.Tasks[0]
	publicTaskID := marketplaceBountyID(project.ID, task.IssueNumber)
	agentType := ""
	if task.RequiredWorkerKind != WorkerHuman {
		agentType = strings.TrimSpace(task.SuggestedAgentType)
		if agentType == "" {
			agentType = "github-pr"
		}
	}
	request := ProjectAutoReleaseRequest{
		TaskIDs: []string{publicTaskID},
		Policy:  defaultAutoReleasePolicy,
		Candidates: []ProjectAutoReleaseCandidate{
			{
				TaskID:            publicTaskID,
				WorkerKind:        task.RequiredWorkerKind,
				WorkerID:          "github:auto-builder",
				AgentType:         agentType,
				RewardCents:       task.RewardCents,
				Repository:        "mergeos-bounties/mergeos",
				PullRequestNumber: 222,
				PullRequestURL:    "https://github.com/mergeos-bounties/mergeos/pull/222",
				PullRequestTitle:  "Auto release proof",
				ReadinessStatus:   "ready",
				CanMerge:          true,
				RiskLevel:         "low",
				CanRelease:        true,
			},
		},
	}

	server := NewServer(cfg, store, payments)
	unsafeRequest := request
	unsafeRequest.Candidates = []ProjectAutoReleaseCandidate{
		{
			TaskID:           publicTaskID,
			WorkerKind:       task.RequiredWorkerKind,
			WorkerID:         "github:auto-builder",
			AgentType:        agentType,
			PullRequestURL:   "https://github.com/mergeos-bounties/mergeos/pull/222",
			PullRequestTitle: "Auto release proof",
		},
	}
	unsafeBody, err := json.Marshal(unsafeRequest)
	if err != nil {
		t.Fatal(err)
	}
	unsafeReq := httptest.NewRequest(http.MethodPost, "/api/projects/"+project.ID+"/auto-release", bytes.NewReader(unsafeBody))
	unsafeReq.Header.Set("Authorization", "Bearer "+auth.Token)
	unsafeResp := httptest.NewRecorder()
	server.Routes().ServeHTTP(unsafeResp, unsafeReq)
	if unsafeResp.Code != http.StatusOK {
		t.Fatalf("unsafe auto-release status = %d, body = %s", unsafeResp.Code, unsafeResp.Body.String())
	}
	var unsafePayload ProjectAutoReleaseResponse
	if err := json.Unmarshal(unsafeResp.Body.Bytes(), &unsafePayload); err != nil {
		t.Fatalf("decode unsafe auto-release: %v", err)
	}
	if unsafePayload.ReleasedCount != 0 || unsafePayload.SkippedCount != 1 || !strings.Contains(unsafePayload.Skipped[0].Reason, "release-ready") {
		t.Fatalf("unsafe auto-release should be skipped by release gate: %#v", unsafePayload)
	}

	bodyBytes, err := json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	reqHTTP := httptest.NewRequest(http.MethodPost, "/api/projects/"+project.ID+"/auto-release", bytes.NewReader(bodyBytes))
	reqHTTP.Header.Set("Authorization", "Bearer "+auth.Token)
	resp := httptest.NewRecorder()
	server.Routes().ServeHTTP(resp, reqHTTP)
	if resp.Code != http.StatusOK {
		t.Fatalf("auto-release status = %d, body = %s", resp.Code, resp.Body.String())
	}

	body := resp.Body.String()
	for _, value := range []string{
		"auto-release-client@example.com",
		"+1 555 0188",
		auth.User.ID,
		defaultDevPaymentCode,
		tempDir,
	} {
		if strings.Contains(body, value) {
			t.Fatalf("auto-release response leaked private value %q: %s", value, body)
		}
	}

	var payload ProjectAutoReleaseResponse
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.ProtocolVersion != "mergeos.payout-release.v1" || payload.Kind != "auto_release" || payload.ProjectID != project.ID {
		t.Fatalf("unexpected auto-release protocol header: %#v", payload)
	}
	if payload.ReleasedCount != 1 || payload.SkippedCount != 0 || len(payload.Released) != 1 {
		t.Fatalf("unexpected auto-release counts: %#v", payload)
	}
	if payload.Payouts.ReleaseCount != 1 || payload.Payouts.ReleasedCents != task.RewardCents {
		t.Fatalf("auto-release did not update payout settlement: %#v", payload.Payouts)
	}
	var paidRow *ProjectPayoutRow
	for index := range payload.Payouts.Payouts {
		if payload.Payouts.Payouts[index].TaskID == task.ID {
			paidRow = &payload.Payouts.Payouts[index]
			break
		}
	}
	if paidRow == nil {
		t.Fatalf("auto-release response missing paid task row: %#v", payload.Payouts.Payouts)
	}
	if paidRow.WorkerID != "github:auto-builder" || paidRow.PayoutAccount != "github:auto-builder" || paidRow.PaidCents != task.RewardCents {
		t.Fatalf("unexpected auto-release payout row: %#v", paidRow)
	}
	if !strings.Contains(paidRow.Reference, "pr:https://github.com/mergeos-bounties/mergeos/pull/222") || !strings.Contains(paidRow.Reference, "auto_release:"+defaultAutoReleasePolicy) {
		t.Fatalf("payout row missing auto-release proof reference: %#v", paidRow)
	}

	secondReq := httptest.NewRequest(http.MethodPost, "/api/projects/"+project.ID+"/auto-release", bytes.NewReader(bodyBytes))
	secondReq.Header.Set("Authorization", "Bearer "+auth.Token)
	secondResp := httptest.NewRecorder()
	server.Routes().ServeHTTP(secondResp, secondReq)
	if secondResp.Code != http.StatusOK {
		t.Fatalf("second auto-release status = %d, body = %s", secondResp.Code, secondResp.Body.String())
	}
	var secondPayload ProjectAutoReleaseResponse
	if err := json.Unmarshal(secondResp.Body.Bytes(), &secondPayload); err != nil {
		t.Fatal(err)
	}
	if secondPayload.ReleasedCount != 0 || secondPayload.SkippedCount != 1 {
		t.Fatalf("expected accepted task to be skipped on second run: %#v", secondPayload)
	}

	otherAuth, err := store.Register(RegisterRequest{
		Name:     "Other Auto Client",
		Email:    "other-auto-client@example.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatal(err)
	}
	forbiddenReq := httptest.NewRequest(http.MethodPost, "/api/projects/"+project.ID+"/auto-release", bytes.NewReader(bodyBytes))
	forbiddenReq.Header.Set("Authorization", "Bearer "+otherAuth.Token)
	forbiddenResp := httptest.NewRecorder()
	server.Routes().ServeHTTP(forbiddenResp, forbiddenReq)
	if forbiddenResp.Code != http.StatusForbidden {
		t.Fatalf("other client auto-release status = %d", forbiddenResp.Code)
	}
}

func TestProjectDashboardRouteAggregatesCustomerWorkflowAndSanitizesData(t *testing.T) {
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
		Name:        "Dashboard Client",
		CompanyName: "Dashboard Co",
		Email:       "dashboard-client@example.com",
		Password:    "password123",
	})
	if err != nil {
		t.Fatal(err)
	}
	project, err := store.CreateProject(context.Background(), auth.User.ID, CreateProjectRequest{
		Title:            "Dashboard aggregate proof",
		ClientName:       "Private Dashboard Client",
		CompanyName:      "Dashboard Co",
		ClientEmail:      "dashboard-client@example.com",
		Phone:            "+1 555 0166",
		Brief:            "Create customer dashboard aggregate data without leaking private payment data.",
		BudgetCents:      210000,
		PaymentMethod:    PaymentPayPal,
		PaymentReference: defaultDevPaymentCode,
	})
	if err != nil {
		t.Fatal(err)
	}

	server := NewServer(cfg, store, payments)
	reqHTTP := httptest.NewRequest(http.MethodGet, "/api/projects/"+project.ID+"/dashboard", nil)
	reqHTTP.Header.Set("Authorization", "Bearer "+auth.Token)
	resp := httptest.NewRecorder()
	server.Routes().ServeHTTP(resp, reqHTTP)
	if resp.Code != http.StatusOK {
		t.Fatalf("dashboard status = %d, body = %s", resp.Code, resp.Body.String())
	}

	body := resp.Body.String()
	for _, value := range []string{
		"dashboard-client@example.com",
		"+1 555 0166",
		auth.User.ID,
		defaultDevPaymentCode,
		tempDir,
	} {
		if strings.Contains(body, value) {
			t.Fatalf("dashboard response leaked private value %q: %s", value, body)
		}
	}

	var payload ProjectDashboardResponse
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.ProtocolVersion != "mergeos.customer-dashboard.v1" || payload.Kind != "customer_dashboard" {
		t.Fatalf("unexpected dashboard protocol header: %#v", payload)
	}
	if payload.Project.ProjectID != project.ID || payload.Project.Title != "Dashboard aggregate proof" {
		t.Fatalf("unexpected dashboard project overview: %#v", payload.Project)
	}
	if payload.Project.TaskCount != len(project.Tasks) || payload.Escrow.ProjectID != project.ID || payload.TaskGraph.Stats.NodeCount != len(project.Tasks) {
		t.Fatalf("dashboard missing task or escrow aggregates: %#v", payload)
	}
	if payload.Payouts.ProtocolVersion != "mergeos.payouts.v1" || payload.Payouts.Kind != "payouts" || payload.Payouts.ProjectID != project.ID {
		t.Fatalf("unexpected dashboard payouts protocol header: %#v", payload.Payouts)
	}
	if payload.Deployment.ProjectID != project.ID || payload.AIWorkflow.ProjectID != project.ID || payload.RepositoryScan.ProjectID != project.ID {
		t.Fatalf("dashboard missing workflow modules: %#v", payload)
	}
	if payload.Deployment.ProtocolVersion != "mergeos.deployment.v1" || payload.Deployment.Kind != "deployment" {
		t.Fatalf("unexpected dashboard deployment protocol header: %#v", payload.Deployment)
	}
	if payload.AIWorkflow.ProtocolVersion != "mergeos.ai-workflow.v1" || payload.AIWorkflow.Kind != "ai_workflow" {
		t.Fatalf("unexpected dashboard AI workflow protocol header: %#v", payload.AIWorkflow)
	}
	if payload.PullRequests.ProjectID != project.ID || payload.UpdatedAt.IsZero() {
		t.Fatalf("dashboard missing pull request monitor shell or timestamp: %#v", payload)
	}
	if payload.PullRequests.ProtocolVersion != "mergeos.pr-monitor.v1" || payload.PullRequests.Kind != "pr_monitor" {
		t.Fatalf("unexpected pull request protocol header: %#v", payload.PullRequests)
	}

	otherAuth, err := store.Register(RegisterRequest{
		Name:     "Other Dashboard Client",
		Email:    "other-dashboard-client@example.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatal(err)
	}
	forbiddenReq := httptest.NewRequest(http.MethodGet, "/api/projects/"+project.ID+"/dashboard", nil)
	forbiddenReq.Header.Set("Authorization", "Bearer "+otherAuth.Token)
	forbiddenResp := httptest.NewRecorder()
	server.Routes().ServeHTTP(forbiddenResp, forbiddenReq)
	if forbiddenResp.Code != http.StatusForbidden {
		t.Fatalf("other client dashboard status = %d", forbiddenResp.Code)
	}
}

func TestProjectAIWorkflowRouteReturnsWorkflowAndSanitizesData(t *testing.T) {
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
		Name:        "AI Client",
		CompanyName: "AI Co",
		Email:       "ai-client@example.com",
		Password:    "password123",
	})
	if err != nil {
		t.Fatal(err)
	}
	project, err := store.CreateProject(context.Background(), auth.User.ID, CreateProjectRequest{
		Title:            "AI workflow proof",
		ClientName:       "Private AI Client",
		CompanyName:      "AI Co",
		ClientEmail:      "ai-client@example.com",
		Phone:            "+1 555 0133",
		Brief:            "Source repository: https://github.com/mergeos-bounties/source-demo\n\nCreate AI workflow data without leaking private customer data.",
		BudgetCents:      230000,
		PaymentMethod:    PaymentPayPal,
		PaymentReference: defaultDevPaymentCode,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.AddGeminiWebhookLog(GeminiWebhookLog{
		EventName:  "pull_request",
		Action:     "opened",
		Repository: project.BountyRepoName,
		PullNumber: 333,
		Sender:     "ai-author",
		Status:     "processed",
		StatusCode: http.StatusOK,
		CommentURL: "https://github.com/mergeos-bounties/mergeos/pull/333#issuecomment-3",
		ReceivedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatal(err)
	}

	server := NewServer(cfg, store, payments)
	reqHTTP := httptest.NewRequest(http.MethodGet, "/api/projects/"+project.ID+"/ai-workflow", nil)
	reqHTTP.Header.Set("Authorization", "Bearer "+auth.Token)
	resp := httptest.NewRecorder()
	server.Routes().ServeHTTP(resp, reqHTTP)
	if resp.Code != http.StatusOK {
		t.Fatalf("ai workflow status = %d, body = %s", resp.Code, resp.Body.String())
	}

	body := resp.Body.String()
	for _, value := range []string{
		"ai-client@example.com",
		"+1 555 0133",
		auth.User.ID,
		defaultDevPaymentCode,
		tempDir,
		project.Tasks[0].ID,
	} {
		if strings.Contains(body, value) {
			t.Fatalf("ai workflow leaked private value %q: %s", value, body)
		}
	}

	var payload ProjectAIWorkflowResponse
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.ProtocolVersion != "mergeos.ai-workflow.v1" || payload.Kind != "ai_workflow" {
		t.Fatalf("unexpected ai workflow protocol header: %#v", payload)
	}
	if payload.ProjectID != project.ID || payload.Status != "orchestrating" || payload.Progress == 0 || payload.AIActionCount != 1 {
		t.Fatalf("unexpected ai workflow summary: %#v", payload)
	}
	if payload.CurrentStep != "pr_review" {
		t.Fatalf("expected current AI workflow step pr_review, got %q", payload.CurrentStep)
	}
	if payload.TaskCount != len(project.Tasks) || payload.AgentTaskCount == 0 || payload.HybridTaskCount == 0 || payload.HumanTaskCount == 0 {
		t.Fatalf("unexpected ai workflow task mix: %#v", payload)
	}
	seenStages := map[string]bool{}
	prReviewStatus := ""
	for _, stage := range payload.Stages {
		seenStages[stage.ID] = true
		if stage.ID == "pr_review" {
			prReviewStatus = stage.Status
		}
	}
	for _, required := range []string{"repo_import", "issue_scan", "task_generation", "reward_estimation", "contributor_routing", "pr_review", "deployment_validation"} {
		if !seenStages[required] {
			t.Fatalf("ai workflow missing stage %s: %#v", required, payload.Stages)
		}
	}
	if prReviewStatus != deploymentStageInProgress {
		t.Fatalf("PR opened should leave review stage in progress, got %q", prReviewStatus)
	}
	if len(payload.Signals) == 0 {
		t.Fatalf("ai workflow missing signals: %#v", payload.Signals)
	}

	otherAuth, err := store.Register(RegisterRequest{
		Name:     "Other AI Client",
		Email:    "other-ai-client@example.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatal(err)
	}
	forbiddenReq := httptest.NewRequest(http.MethodGet, "/api/projects/"+project.ID+"/ai-workflow", nil)
	forbiddenReq.Header.Set("Authorization", "Bearer "+otherAuth.Token)
	forbiddenResp := httptest.NewRecorder()
	server.Routes().ServeHTTP(forbiddenResp, forbiddenReq)
	if forbiddenResp.Code != http.StatusForbidden {
		t.Fatalf("other client ai workflow status = %d", forbiddenResp.Code)
	}
}

func TestPublicProjectAIWorkflowRouteReturnsSanitizedWorkflow(t *testing.T) {
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
		Name:        "Public AI Client",
		CompanyName: "Public AI Co",
		Email:       "public-ai-client@example.com",
		Password:    "password123",
	})
	if err != nil {
		t.Fatal(err)
	}
	project, err := store.CreateProject(context.Background(), auth.User.ID, CreateProjectRequest{
		Title:            "Public AI workflow",
		ClientName:       "Private Public AI Client",
		CompanyName:      "Public AI Co",
		ClientEmail:      "public-ai-client@example.com",
		Phone:            "+1 555 0122",
		Brief:            "Source repository: https://github.com/mergeos-bounties/public-ai-demo\n\nExpose public AI workflow without leaking private customer data.",
		BudgetCents:      230000,
		PaymentMethod:    PaymentPayPal,
		PaymentReference: defaultDevPaymentCode,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.AddGeminiWebhookLog(GeminiWebhookLog{
		EventName:  "agent_action",
		Action:     "review",
		Repository: project.BountyRepoName,
		PullNumber: 444,
		Sender:     "review-agent",
		Status:     "processed",
		StatusCode: http.StatusOK,
		CommentURL: "https://github.com/mergeos-bounties/mergeos/pull/444#issuecomment-4",
		ReceivedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatal(err)
	}

	server := NewServer(cfg, store, payments)
	reqHTTP := httptest.NewRequest(http.MethodGet, "/api/public/projects/"+project.ID+"/ai-workflow", nil)
	resp := httptest.NewRecorder()
	server.Routes().ServeHTTP(resp, reqHTTP)
	if resp.Code != http.StatusOK {
		t.Fatalf("public ai workflow status = %d, body = %s", resp.Code, resp.Body.String())
	}

	body := resp.Body.String()
	for _, value := range []string{
		"public-ai-client@example.com",
		"+1 555 0122",
		auth.User.ID,
		defaultDevPaymentCode,
		tempDir,
		project.Tasks[0].ID,
	} {
		if strings.Contains(body, value) {
			t.Fatalf("public ai workflow leaked private value %q: %s", value, body)
		}
	}

	var payload ProjectAIWorkflowResponse
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.ProtocolVersion != "mergeos.ai-workflow.v1" || payload.Kind != "ai_workflow" {
		t.Fatalf("unexpected public ai workflow protocol header: %#v", payload)
	}
	if payload.ProjectID != project.ID || payload.Status != "orchestrating" || payload.Progress == 0 || payload.AIActionCount != 1 {
		t.Fatalf("unexpected public ai workflow summary: %#v", payload)
	}
	if payload.CurrentStep != "deployment_validation" {
		t.Fatalf("expected public current step deployment_validation, got %q", payload.CurrentStep)
	}
	seenStages := map[string]bool{}
	for _, stage := range payload.Stages {
		seenStages[stage.ID] = true
	}
	for _, required := range []string{"repo_import", "issue_scan", "task_generation", "reward_estimation", "contributor_routing", "pr_review", "deployment_validation"} {
		if !seenStages[required] {
			t.Fatalf("public ai workflow missing stage %s: %#v", required, payload.Stages)
		}
	}
	foundReviewSignal := false
	for _, signal := range payload.Signals {
		if signal.Type == "agent_action" && signal.Status == "processed" {
			foundReviewSignal = true
			break
		}
	}
	if !foundReviewSignal {
		t.Fatalf("public ai workflow missing agent action signal: %#v", payload.Signals)
	}
}

func TestPublicProjectWorkflowRouteReturnsSanitizedGraph(t *testing.T) {
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
		Name:        "Public Graph Client",
		CompanyName: "Public Graph Co",
		Email:       "public-graph-client@example.com",
		Password:    "password123",
	})
	if err != nil {
		t.Fatal(err)
	}
	project, err := store.CreateProject(context.Background(), auth.User.ID, CreateProjectRequest{
		Title:            "Public workflow graph",
		ClientName:       "Private Graph Buyer",
		CompanyName:      "Public Graph Co",
		ClientEmail:      "public-graph-client@example.com",
		Phone:            "+1 555 0166",
		Brief:            "Source repository: https://github.com/mergeos-bounties/public-workflow-demo\n\nExpose a public graph for external agents without leaking private task identifiers.",
		BudgetCents:      260000,
		PaymentMethod:    PaymentPayPal,
		PaymentReference: defaultDevPaymentCode,
	})
	if err != nil {
		t.Fatal(err)
	}

	server := NewServer(cfg, store, payments)
	reqHTTP := httptest.NewRequest(http.MethodGet, "/api/public/projects/"+project.ID+"/workflow", nil)
	resp := httptest.NewRecorder()
	server.Routes().ServeHTTP(resp, reqHTTP)
	if resp.Code != http.StatusOK {
		t.Fatalf("public workflow status = %d, body = %s", resp.Code, resp.Body.String())
	}

	body := resp.Body.String()
	for _, value := range []string{
		"public-graph-client@example.com",
		"+1 555 0166",
		auth.User.ID,
		defaultDevPaymentCode,
		tempDir,
	} {
		if strings.Contains(body, value) {
			t.Fatalf("public workflow leaked private value %q: %s", value, body)
		}
	}
	for _, task := range project.Tasks {
		if strings.Contains(body, task.ID) {
			t.Fatalf("public workflow leaked internal task id %q: %s", task.ID, body)
		}
	}

	var document WorkflowProtocolDocument
	if err := json.Unmarshal(resp.Body.Bytes(), &document); err != nil {
		t.Fatal(err)
	}
	if document.ProtocolVersion != "mergeos.workflow.v1" || document.Kind != "workflow" || document.ProjectID != project.ID {
		t.Fatalf("unexpected public workflow header: %#v", document)
	}
	if document.ID != project.ID+":public-workflow" {
		t.Fatalf("unexpected public workflow id: %#v", document.ID)
	}
	if len(document.Nodes) != len(project.Tasks) || len(document.Edges) == 0 {
		t.Fatalf("public workflow graph mismatch: %#v", document)
	}
	if len(document.Stages) != 7 || len(document.Checks) != 7 || len(document.Evidence) == 0 {
		t.Fatalf("public workflow missing orchestration stages, checks, or evidence: %#v", document)
	}

	publicIDs := map[string]bool{}
	for _, node := range document.Nodes {
		if strings.TrimSpace(node.ID) == "" || node.ID != node.TaskID {
			t.Fatalf("public workflow node did not use claim-safe task id: %#v", node)
		}
		if !strings.HasPrefix(node.TaskID, project.ID+":") {
			t.Fatalf("public workflow node is not keyed by public claim id: %#v", node)
		}
		publicIDs[node.TaskID] = true
	}
	for _, node := range document.Nodes {
		for _, dependency := range node.Dependencies {
			if !publicIDs[dependency] {
				t.Fatalf("public workflow dependency does not reference a public node id: node=%#v dependency=%q", node, dependency)
			}
		}
	}
	for _, action := range document.NextActions {
		if action.TaskID != "" && !publicIDs[action.TaskID] {
			t.Fatalf("public workflow action does not reference a public task id: %#v", action)
		}
		if action.TargetNodeID != "" && !publicIDs[action.TargetNodeID] {
			t.Fatalf("public workflow action does not reference a public node id: %#v", action)
		}
	}
	for _, edge := range document.Edges {
		if !publicIDs[edge.From] || !publicIDs[edge.To] {
			t.Fatalf("public workflow edge does not reference public node ids: %#v", edge)
		}
	}
	if document.Metadata["public"] != true ||
		document.Metadata["workflow_endpoint"] != "/api/public/projects/{id}/workflow" ||
		document.Metadata["task_protocol_endpoint"] != "/api/public/protocol/tasks" {
		t.Fatalf("public workflow missing agent context metadata: %#v", document.Metadata)
	}

	missingReq := httptest.NewRequest(http.MethodGet, "/api/public/projects/missing/workflow", nil)
	missingResp := httptest.NewRecorder()
	server.Routes().ServeHTTP(missingResp, missingReq)
	if missingResp.Code != http.StatusNotFound {
		t.Fatalf("missing public workflow status = %d", missingResp.Code)
	}
}

func TestProjectAgentActionRouteRecordsWorkflowEventAndSanitizesData(t *testing.T) {
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
		Name:        "Agent Client",
		CompanyName: "Agent Co",
		Email:       "agent-client@example.com",
		Password:    "password123",
	})
	if err != nil {
		t.Fatal(err)
	}
	project, err := store.CreateProject(context.Background(), auth.User.ID, CreateProjectRequest{
		Title:            "Agent action proof",
		ClientName:       "Private Agent Client",
		CompanyName:      "Agent Co",
		ClientEmail:      "agent-client@example.com",
		Phone:            "+1 555 0190",
		Brief:            "Create AI agent action evidence without leaking private customer data.",
		BudgetCents:      210000,
		PaymentMethod:    PaymentPayPal,
		PaymentReference: defaultDevPaymentCode,
	})
	if err != nil {
		t.Fatal(err)
	}

	server := NewServer(cfg, store, payments)
	createReq := httptest.NewRequest(http.MethodPost, "/api/projects/"+project.ID+"/agent-actions", strings.NewReader(`{
		"action":"test",
		"agent_type":"qa-agent",
		"status":"processed",
		"pull_number":777,
		"reference_url":"https://github.com/mergeos-bounties/mergeos/pull/777",
		"labels":["evidence: star"],
		"context_urls":[
			"https://mergeos.shop/api/public/projects/prj_0001/workflow",
			"file:///D:/agent/private-plan"
		],
		"evidence":["Smoke tests passed","Preview deployment reachable"],
		"runbook":["Fetch task packet","Run smoke suite","Attach deployment evidence"],
		"checks":[
			{"name":"Smoke suite","status":"passed","summary":"Preview route passed.","reference_url":"https://github.com/mergeos-bounties/mergeos/actions/runs/777"},
			{"name":"Risk review","status":"needs_review","summary":"Manual acceptance note pending.","reference_url":"file:///D:/agent/internal"}
		],
		"duration_millis":1234
	}`))
	createReq.Header.Set("Authorization", "Bearer "+auth.Token)
	createResp := httptest.NewRecorder()
	server.Routes().ServeHTTP(createResp, createReq)
	if createResp.Code != http.StatusCreated {
		t.Fatalf("agent action status = %d, body = %s", createResp.Code, createResp.Body.String())
	}

	privateValues := []string{
		"agent-client@example.com",
		"+1 555 0190",
		auth.User.ID,
		defaultDevPaymentCode,
		tempDir,
		project.Tasks[0].ID,
		"file:///D:/agent/private-plan",
		"file:///D:/agent/internal",
	}
	body := createResp.Body.String()
	for _, value := range privateValues {
		if strings.Contains(body, value) {
			t.Fatalf("agent action response leaked private value %q: %s", value, body)
		}
	}

	var created AgentActionResponse
	if err := json.Unmarshal(createResp.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if created.ProtocolVersion != "mergeos.agent-action.v1" || created.Kind != "agent_action" || created.ActionID == "" {
		t.Fatalf("unexpected agent action protocol header: %#v", created)
	}
	if created.ProjectID != project.ID || created.Action != "test" || created.AgentType != "qa-agent" || created.Status != "processed" || created.ReceivedAt.IsZero() {
		t.Fatalf("unexpected agent action protocol fields: %#v", created)
	}
	if created.Repository != project.BountyRepoName || created.PullNumber != 777 || created.ReferenceURL != "https://github.com/mergeos-bounties/mergeos/pull/777" || created.DurationMillis != 1234 {
		t.Fatalf("unexpected agent action protocol evidence: %#v", created)
	}
	if len(created.ContextURLs) != 1 || created.ContextURLs[0] != "https://mergeos.shop/api/public/projects/prj_0001/workflow" {
		t.Fatalf("agent action context URLs were not sanitized: %#v", created.ContextURLs)
	}
	if len(created.Evidence) != 2 || created.Evidence[0] != "Smoke tests passed" || len(created.Runbook) != 3 {
		t.Fatalf("unexpected agent action packet lists: evidence=%#v runbook=%#v", created.Evidence, created.Runbook)
	}
	if len(created.Checks) != 2 || created.Checks[0].Status != "passed" || created.Checks[0].ReferenceURL == "" || created.Checks[1].Status != "warning" || created.Checks[1].ReferenceURL != "" {
		t.Fatalf("agent action checks were not normalized: %#v", created.Checks)
	}
	if created.Log.EventName != "agent_action" || created.Log.Action != "test" || created.Log.Repository != project.BountyRepoName || created.Log.PullNumber != 777 {
		t.Fatalf("unexpected agent action log: %#v", created.Log)
	}
	if created.Log.Status != "processed" || created.Log.CommentURL != "https://github.com/mergeos-bounties/mergeos/pull/777" || created.Log.DurationMillis != 1234 {
		t.Fatalf("unexpected agent action status fields: %#v", created.Log)
	}

	workflowReq := httptest.NewRequest(http.MethodGet, "/api/projects/"+project.ID+"/ai-workflow", nil)
	workflowReq.Header.Set("Authorization", "Bearer "+auth.Token)
	workflowResp := httptest.NewRecorder()
	server.Routes().ServeHTTP(workflowResp, workflowReq)
	if workflowResp.Code != http.StatusOK {
		t.Fatalf("ai workflow after agent action status = %d, body = %s", workflowResp.Code, workflowResp.Body.String())
	}
	for _, value := range privateValues {
		if strings.Contains(workflowResp.Body.String(), value) {
			t.Fatalf("ai workflow leaked private value %q: %s", value, workflowResp.Body.String())
		}
	}
	var workflow ProjectAIWorkflowResponse
	if err := json.Unmarshal(workflowResp.Body.Bytes(), &workflow); err != nil {
		t.Fatal(err)
	}
	if workflow.AIActionCount != 1 {
		t.Fatalf("ai workflow action count = %d", workflow.AIActionCount)
	}
	seenAgentSignal := false
	for _, signal := range workflow.Signals {
		if signal.Type == "agent_action" {
			seenAgentSignal = true
		}
	}
	if !seenAgentSignal {
		t.Fatalf("ai workflow missing agent action signal: %#v", workflow.Signals)
	}

	feedReq := httptest.NewRequest(http.MethodGet, "/api/public/live-feed?limit=20", nil)
	feedResp := httptest.NewRecorder()
	server.Routes().ServeHTTP(feedResp, feedReq)
	if feedResp.Code != http.StatusOK {
		t.Fatalf("public live feed status = %d, body = %s", feedResp.Code, feedResp.Body.String())
	}
	for _, value := range privateValues {
		if strings.Contains(feedResp.Body.String(), value) {
			t.Fatalf("public live feed leaked private value %q: %s", value, feedResp.Body.String())
		}
	}
	var feed PublicLiveFeedResponse
	if err := json.Unmarshal(feedResp.Body.Bytes(), &feed); err != nil {
		t.Fatal(err)
	}
	if feed.ProtocolVersion != "mergeos.live-feed.v1" || feed.Kind != "live_feed" {
		t.Fatalf("unexpected live feed protocol header: %#v", feed)
	}
	seenAgentItem := false
	for _, item := range feed.Items {
		if item.Type == "agent_action" && item.Actor == "QA Agent" && item.Action == "test" {
			seenAgentItem = true
			if len(item.ContextURLs) != 2 || item.ContextURLs[0] != "https://mergeos.shop/api/public/projects/prj_0001/workflow" || item.ContextURLs[1] != "https://github.com/mergeos-bounties/mergeos/pull/777" {
				t.Fatalf("live feed agent action missing public context URLs: %#v", item.ContextURLs)
			}
			if len(item.Evidence) != 2 || len(item.Runbook) != 3 || len(item.Checks) != 2 {
				t.Fatalf("live feed agent action missing packet fields: %#v", item)
			}
		}
	}
	if !seenAgentItem {
		t.Fatalf("public live feed missing agent action item: %#v", feed.Items)
	}

	protocolReq := httptest.NewRequest(http.MethodGet, "/api/public/protocol/events?limit=20", nil)
	protocolResp := httptest.NewRecorder()
	server.Routes().ServeHTTP(protocolResp, protocolReq)
	if protocolResp.Code != http.StatusOK {
		t.Fatalf("public protocol events status = %d, body = %s", protocolResp.Code, protocolResp.Body.String())
	}
	for _, value := range privateValues {
		if strings.Contains(protocolResp.Body.String(), value) {
			t.Fatalf("public protocol events leaked private value %q: %s", value, protocolResp.Body.String())
		}
	}
	var events PublicEventProtocolResponse
	if err := json.Unmarshal(protocolResp.Body.Bytes(), &events); err != nil {
		t.Fatal(err)
	}
	seenAgentEvent := false
	for _, event := range events.Events {
		if event.Type == "agent.tested" && event.Actor == "QA Agent" && event.Payload["action"] == "test" {
			seenAgentEvent = true
			payloadBytes, err := json.Marshal(event.Payload)
			if err != nil {
				t.Fatal(err)
			}
			payloadText := string(payloadBytes)
			for _, required := range []string{"context_urls", "evidence", "runbook", "checks", "Smoke tests passed", "https://mergeos.shop/api/public/projects/prj_0001/workflow"} {
				if !strings.Contains(payloadText, required) {
					t.Fatalf("public protocol event missing %q in payload: %s", required, payloadText)
				}
			}
			if strings.Contains(payloadText, "file:///") {
				t.Fatalf("public protocol event leaked private URL: %s", payloadText)
			}
		}
	}
	if !seenAgentEvent {
		t.Fatalf("public protocol events missing agent action: %#v", events.Events)
	}

	workerAuth, err := store.AuthenticateGitHub(GitHubAuthProfile{
		ID:       "agent-action-worker-1",
		Username: "evidence-agent",
		Name:     "Evidence Agent",
		Email:    "evidence-agent@example.com",
	}, "", "")
	if err != nil {
		t.Fatal(err)
	}
	var agentTask *Task
	for _, task := range project.Tasks {
		if task.RequiredWorkerKind != WorkerHuman {
			agentTask = task
			break
		}
	}
	if agentTask == nil {
		t.Fatalf("project did not create an agent or hybrid task: %#v", project.Tasks)
	}
	claimID := marketplaceBountyID(project.ID, agentTask.IssueNumber)
	claimReq := httptest.NewRequest(http.MethodPost, "/api/tasks/"+claimID+"/claim", strings.NewReader(`{"worker_kind":"human","worker_id":"github:spoofed","agent_type":"wrong-agent"}`))
	claimReq.Header.Set("Authorization", "Bearer "+workerAuth.Token)
	claimResp := httptest.NewRecorder()
	server.Routes().ServeHTTP(claimResp, claimReq)
	if claimResp.Code != http.StatusOK {
		t.Fatalf("worker claim status = %d, body = %s", claimResp.Code, claimResp.Body.String())
	}
	var accepted TaskClaimResponse
	if err := json.Unmarshal(claimResp.Body.Bytes(), &accepted); err != nil {
		t.Fatal(err)
	}
	if accepted.ClaimID != claimID || accepted.WorkerID != "github:evidence-agent" || accepted.WorkerKind != agentTask.RequiredWorkerKind {
		t.Fatalf("worker claim did not bind public claim id and identity: %#v", accepted)
	}
	if accepted.WorkerKind != WorkerHuman && accepted.AgentType != agentTask.SuggestedAgentType {
		t.Fatalf("worker claim did not use task agent type: %#v", accepted)
	}

	workerBody := fmt.Sprintf(`{
		"action":"review",
		"claim_id":%q,
		"status":"processed",
		"evidence":["Claimed lane review completed"]
	}`, claimID)
	workerReq := httptest.NewRequest(http.MethodPost, "/api/projects/"+project.ID+"/agent-actions", strings.NewReader(workerBody))
	workerReq.Header.Set("Authorization", "Bearer "+workerAuth.Token)
	workerResp := httptest.NewRecorder()
	server.Routes().ServeHTTP(workerResp, workerReq)
	if workerResp.Code != http.StatusCreated {
		t.Fatalf("claimed worker agent action status = %d, body = %s", workerResp.Code, workerResp.Body.String())
	}
	if strings.Contains(workerResp.Body.String(), agentTask.ID) {
		t.Fatalf("claimed worker agent action leaked internal task id %q: %s", agentTask.ID, workerResp.Body.String())
	}
	var workerAction AgentActionResponse
	if err := json.Unmarshal(workerResp.Body.Bytes(), &workerAction); err != nil {
		t.Fatal(err)
	}
	if workerAction.ClaimID != claimID || workerAction.BountyID != claimID || workerAction.ProjectID != project.ID {
		t.Fatalf("claimed worker action missing public claim fields: %#v", workerAction)
	}
	if accepted.WorkerKind != WorkerHuman && workerAction.AgentType != accepted.AgentType {
		t.Fatalf("claimed worker action did not inherit accepted agent type: %#v", workerAction)
	}

	missingClaimReq := httptest.NewRequest(http.MethodPost, "/api/projects/"+project.ID+"/agent-actions", strings.NewReader(`{"action":"test"}`))
	missingClaimReq.Header.Set("Authorization", "Bearer "+workerAuth.Token)
	missingClaimResp := httptest.NewRecorder()
	server.Routes().ServeHTTP(missingClaimResp, missingClaimReq)
	if missingClaimResp.Code != http.StatusForbidden {
		t.Fatalf("worker action without claim status = %d, body = %s", missingClaimResp.Code, missingClaimResp.Body.String())
	}

	wrongAgentReq := httptest.NewRequest(http.MethodPost, "/api/projects/"+project.ID+"/agent-actions", strings.NewReader(fmt.Sprintf(`{"action":"test","bounty_id":%q,"agent_type":"wrong-agent"}`, claimID)))
	wrongAgentReq.Header.Set("Authorization", "Bearer "+workerAuth.Token)
	wrongAgentResp := httptest.NewRecorder()
	server.Routes().ServeHTTP(wrongAgentResp, wrongAgentReq)
	if wrongAgentResp.Code != http.StatusForbidden {
		t.Fatalf("worker action with wrong agent type status = %d, body = %s", wrongAgentResp.Code, wrongAgentResp.Body.String())
	}

	otherWorkerAuth, err := store.AuthenticateGitHub(GitHubAuthProfile{
		ID:       "agent-action-worker-2",
		Username: "other-evidence-agent",
		Name:     "Other Evidence Agent",
		Email:    "other-evidence-agent@example.com",
	}, "", "")
	if err != nil {
		t.Fatal(err)
	}
	otherWorkerReq := httptest.NewRequest(http.MethodPost, "/api/projects/"+project.ID+"/agent-actions", strings.NewReader(fmt.Sprintf(`{"action":"test","bounty_id":%q}`, claimID)))
	otherWorkerReq.Header.Set("Authorization", "Bearer "+otherWorkerAuth.Token)
	otherWorkerResp := httptest.NewRecorder()
	server.Routes().ServeHTTP(otherWorkerResp, otherWorkerReq)
	if otherWorkerResp.Code != http.StatusForbidden {
		t.Fatalf("other worker action with claimed bounty status = %d, body = %s", otherWorkerResp.Code, otherWorkerResp.Body.String())
	}

	otherAuth, err := store.Register(RegisterRequest{
		Name:     "Other Agent Client",
		Email:    "other-agent-client@example.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatal(err)
	}
	forbiddenReq := httptest.NewRequest(http.MethodPost, "/api/projects/"+project.ID+"/agent-actions", strings.NewReader(`{"action":"test"}`))
	forbiddenReq.Header.Set("Authorization", "Bearer "+otherAuth.Token)
	forbiddenResp := httptest.NewRecorder()
	server.Routes().ServeHTTP(forbiddenResp, forbiddenReq)
	if forbiddenResp.Code != http.StatusForbidden {
		t.Fatalf("other client agent action status = %d", forbiddenResp.Code)
	}
}

func TestProjectTaskGraphRouteReturnsAcyclicDependencyGraph(t *testing.T) {
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
		Name:        "Graph Client",
		CompanyName: "Graph Co",
		Email:       "graph-client@example.com",
		Password:    "password123",
	})
	if err != nil {
		t.Fatal(err)
	}
	project, err := store.CreateProject(context.Background(), auth.User.ID, CreateProjectRequest{
		Title:            "Task graph proof",
		ClientName:       "Private Graph Client",
		CompanyName:      "Graph Co",
		ClientEmail:      "graph-client@example.com",
		Phone:            "+1 555 0144",
		Brief:            "Create a task dependency graph for AI routing.",
		BudgetCents:      210000,
		PaymentMethod:    PaymentPayPal,
		PaymentReference: defaultDevPaymentCode,
	})
	if err != nil {
		t.Fatal(err)
	}

	server := NewServer(cfg, store, payments)
	reqHTTP := httptest.NewRequest(http.MethodGet, "/api/projects/"+project.ID+"/task-graph", nil)
	reqHTTP.Header.Set("Authorization", "Bearer "+auth.Token)
	resp := httptest.NewRecorder()
	server.Routes().ServeHTTP(resp, reqHTTP)
	if resp.Code != http.StatusOK {
		t.Fatalf("task graph status = %d, body = %s", resp.Code, resp.Body.String())
	}

	body := resp.Body.String()
	for _, value := range []string{
		"graph-client@example.com",
		"+1 555 0144",
		auth.User.ID,
		defaultDevPaymentCode,
		tempDir,
	} {
		if strings.Contains(body, value) {
			t.Fatalf("task graph leaked private value %q: %s", value, body)
		}
	}

	var payload ProjectTaskGraphResponse
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.ProjectID != project.ID || payload.Stats.NodeCount != len(project.Tasks) || len(payload.Nodes) != len(project.Tasks) {
		t.Fatalf("unexpected task graph nodes: %#v", payload)
	}
	if payload.Stats.EdgeCount == 0 || len(payload.Edges) == 0 {
		t.Fatalf("task graph missing edges: %#v", payload)
	}
	if payload.Stats.ReadyCount != 1 || payload.Stats.BlockedCount == 0 {
		t.Fatalf("unexpected task graph readiness: %#v", payload.Stats)
	}
	issueByTaskID := map[string]int{}
	for _, node := range payload.Nodes {
		issueByTaskID[node.TaskID] = node.IssueNumber
	}
	for _, edge := range payload.Edges {
		if issueByTaskID[edge.From] >= issueByTaskID[edge.To] {
			t.Fatalf("task graph edge is not acyclic by issue order: %#v", edge)
		}
	}
	if !payload.Nodes[0].Ready || len(payload.Nodes[0].BlockedBy) != 0 {
		t.Fatalf("first task should be ready: %#v", payload.Nodes[0])
	}
	if payload.Nodes[0].EstimatedHours <= 0 || payload.Nodes[0].RequiredWorkerKind == "" {
		t.Fatalf("task graph node missing routing estimates: %#v", payload.Nodes[0])
	}

	protocolReq := httptest.NewRequest(http.MethodGet, "/api/projects/"+project.ID+"/protocol/workflow", nil)
	protocolReq.Header.Set("Authorization", "Bearer "+auth.Token)
	protocolResp := httptest.NewRecorder()
	server.Routes().ServeHTTP(protocolResp, protocolReq)
	if protocolResp.Code != http.StatusOK {
		t.Fatalf("workflow protocol status = %d, body = %s", protocolResp.Code, protocolResp.Body.String())
	}
	protocolBody := protocolResp.Body.String()
	for _, value := range []string{
		"graph-client@example.com",
		"+1 555 0144",
		auth.User.ID,
		defaultDevPaymentCode,
		tempDir,
	} {
		if strings.Contains(protocolBody, value) {
			t.Fatalf("workflow protocol leaked private value %q: %s", value, protocolBody)
		}
	}
	var document WorkflowProtocolDocument
	if err := json.Unmarshal(protocolResp.Body.Bytes(), &document); err != nil {
		t.Fatal(err)
	}
	if document.ProtocolVersion != "mergeos.workflow.v1" || document.Kind != "workflow" || document.ProjectID != project.ID {
		t.Fatalf("unexpected workflow protocol header: %#v", document)
	}
	if len(document.Nodes) != len(payload.Nodes) || len(document.Edges) != len(payload.Edges) {
		t.Fatalf("workflow protocol graph mismatch: %#v", document)
	}
	if document.Nodes[0].Status != "ready" || document.Nodes[0].RewardMRG <= 0 {
		t.Fatalf("unexpected workflow protocol first node: %#v", document.Nodes[0])
	}
	if document.Nodes[0].EstimatedHours <= 0 || document.Nodes[0].RequiredWorkerKind == "" {
		t.Fatalf("workflow protocol node missing routing estimates: %#v", document.Nodes[0])
	}
	if len(document.Nodes) > 1 && len(document.Nodes[1].Dependencies) == 0 {
		t.Fatalf("workflow protocol node missing dependencies: %#v", document.Nodes[1])
	}
	if document.Progress != payload.Progress || document.CurrentStep != "contributor_routing" {
		t.Fatalf("workflow protocol missing top-level workflow progress: %#v", document)
	}
	workflowSteps, ok := document.Metadata["workflow_steps"].([]interface{})
	if !ok || len(workflowSteps) != 7 || document.Metadata["current_step"] != "contributor_routing" {
		t.Fatalf("workflow protocol missing AI workflow stage metadata: %#v", document.Metadata)
	}
	if len(document.Stages) != 7 || len(document.Checks) != 7 || len(document.Evidence) == 0 {
		t.Fatalf("workflow protocol missing execution stages, checks, or evidence: %#v", document)
	}
	if len(document.NextActions) == 0 {
		t.Fatalf("workflow protocol missing executable next actions: %#v", document)
	}
	nodeIDs := map[string]bool{}
	for _, node := range document.Nodes {
		nodeIDs[node.ID] = true
		nodeIDs[node.TaskID] = true
	}
	for _, action := range document.NextActions {
		if action.TaskID != "" && !nodeIDs[action.TaskID] {
			t.Fatalf("workflow action references unknown task: %#v", action)
		}
		if action.TargetNodeID != "" && !nodeIDs[action.TargetNodeID] {
			t.Fatalf("workflow action references unknown node: %#v", action)
		}
	}
	routingReq := httptest.NewRequest(http.MethodGet, "/api/projects/"+project.ID+"/routing", nil)
	routingReq.Header.Set("Authorization", "Bearer "+auth.Token)
	routingResp := httptest.NewRecorder()
	server.Routes().ServeHTTP(routingResp, routingReq)
	if routingResp.Code != http.StatusOK {
		t.Fatalf("project routing status = %d, body = %s", routingResp.Code, routingResp.Body.String())
	}
	routingBody := routingResp.Body.String()
	for _, value := range []string{
		"graph-client@example.com",
		"+1 555 0144",
		defaultDevPaymentCode,
		tempDir,
	} {
		if strings.Contains(routingBody, value) {
			t.Fatalf("project routing leaked private value %q: %s", value, routingBody)
		}
	}
	var routing ProjectRoutingResponse
	if err := json.Unmarshal(routingResp.Body.Bytes(), &routing); err != nil {
		t.Fatal(err)
	}
	if routing.ProtocolVersion != "mergeos.routing.v1" || routing.Kind != "project_routing" || routing.ProjectID != project.ID {
		t.Fatalf("unexpected project routing header: %#v", routing)
	}
	if routing.Stats.TaskCount != len(project.Tasks) || routing.Stats.ReadyCount == 0 || len(routing.Routes) != len(project.Tasks) || len(routing.Lanes) == 0 {
		t.Fatalf("project routing missing lanes or ready routes: %#v", routing)
	}
	for _, route := range routing.Routes {
		if route.TaskID == "" || route.RewardCents <= 0 || route.RequiredWorkerKind == "" || route.MatchScore <= 0 || route.RecommendedNextAction == "" {
			t.Fatalf("project routing route missing decision fields: %#v", route)
		}
		if route.RequiredWorkerKind == WorkerAgent || route.RequiredWorkerKind == WorkerHybrid {
			if route.RecommendedAgent == nil || route.RecommendedAgent.Type == "" {
				t.Fatalf("project routing did not attach agent recommendation: %#v", route)
			}
		}
	}
	if err := store.AddGeminiWebhookLog(GeminiWebhookLog{
		EventName:  "pull_request",
		Action:     "opened",
		Repository: project.BountyRepoName,
		PullNumber: 444,
		Sender:     "graph-author",
		Status:     "processed",
		StatusCode: http.StatusOK,
		CommentURL: "https://github.com/mergeos-bounties/mergeos/pull/444",
		ReceivedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatal(err)
	}
	activeProtocolReq := httptest.NewRequest(http.MethodGet, "/api/projects/"+project.ID+"/protocol/workflow", nil)
	activeProtocolReq.Header.Set("Authorization", "Bearer "+auth.Token)
	activeProtocolResp := httptest.NewRecorder()
	server.Routes().ServeHTTP(activeProtocolResp, activeProtocolReq)
	if activeProtocolResp.Code != http.StatusOK {
		t.Fatalf("active workflow protocol status = %d, body = %s", activeProtocolResp.Code, activeProtocolResp.Body.String())
	}
	var activeDocument WorkflowProtocolDocument
	if err := json.Unmarshal(activeProtocolResp.Body.Bytes(), &activeDocument); err != nil {
		t.Fatal(err)
	}
	if activeDocument.CurrentStep != "pr_review" || activeDocument.Metadata["current_step"] != "pr_review" {
		t.Fatalf("workflow protocol did not use active AI workflow step: %#v", activeDocument)
	}

	otherAuth, err := store.Register(RegisterRequest{
		Name:     "Other Graph Client",
		Email:    "other-graph-client@example.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatal(err)
	}
	forbiddenReq := httptest.NewRequest(http.MethodGet, "/api/projects/"+project.ID+"/task-graph", nil)
	forbiddenReq.Header.Set("Authorization", "Bearer "+otherAuth.Token)
	forbiddenResp := httptest.NewRecorder()
	server.Routes().ServeHTTP(forbiddenResp, forbiddenReq)
	if forbiddenResp.Code != http.StatusForbidden {
		t.Fatalf("other client task graph status = %d", forbiddenResp.Code)
	}

	forbiddenRoutingReq := httptest.NewRequest(http.MethodGet, "/api/projects/"+project.ID+"/routing", nil)
	forbiddenRoutingReq.Header.Set("Authorization", "Bearer "+otherAuth.Token)
	forbiddenRoutingResp := httptest.NewRecorder()
	server.Routes().ServeHTTP(forbiddenRoutingResp, forbiddenRoutingReq)
	if forbiddenRoutingResp.Code != http.StatusForbidden {
		t.Fatalf("other client routing status = %d", forbiddenRoutingResp.Code)
	}

	forbiddenProtocolReq := httptest.NewRequest(http.MethodGet, "/api/projects/"+project.ID+"/protocol/workflow", nil)
	forbiddenProtocolReq.Header.Set("Authorization", "Bearer "+otherAuth.Token)
	forbiddenProtocolResp := httptest.NewRecorder()
	server.Routes().ServeHTTP(forbiddenProtocolResp, forbiddenProtocolReq)
	if forbiddenProtocolResp.Code != http.StatusForbidden {
		t.Fatalf("other client workflow protocol status = %d", forbiddenProtocolResp.Code)
	}
}

func TestProjectRepositoryScanRouteReturnsStaticFindings(t *testing.T) {
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
		Name:        "Scan Client",
		CompanyName: "Scan Co",
		Email:       "scan-client@example.com",
		Password:    "password123",
	})
	if err != nil {
		t.Fatal(err)
	}
	project, err := store.CreateProject(context.Background(), auth.User.ID, CreateProjectRequest{
		Title:            "Repository scan proof",
		ClientName:       "Private Scan Client",
		CompanyName:      "Scan Co",
		ClientEmail:      "scan-client@example.com",
		Phone:            "+1 555 0155",
		Brief:            "Create a repository scan proof.",
		BudgetCents:      210000,
		PaymentMethod:    PaymentPayPal,
		PaymentReference: defaultDevPaymentCode,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project.RepoLocalPath, "package.json"), []byte(`{"dependencies":{"vue":"latest"},"devDependencies":{"vite":"^5.0.0"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project.RepoLocalPath, "pyproject.toml"), []byte("[project]\ndependencies = [\"requests>=2\", \"fastapi\"]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project.RepoLocalPath, "Cargo.toml"), []byte("[dependencies]\nserde = \"1\"\ntokio = { version = \"1\" }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project.RepoLocalPath, "composer.json"), []byte(`{"require":{"php":">=8.2","guzzlehttp/guzzle":"^7.0"},"require-dev":{"phpunit/phpunit":"^10.0"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	srcDir := filepath.Join(project.RepoLocalPath, "src")
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "config.js"), []byte("const API_SECRET = 'super-secret-token';\n// TODO tighten this test hook\nwindow.eval(userInput);\ndocument.body.innerHTML = html;\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "server.go"), []byte("package main\n\nfunc crash() {\n\tpanic(\"unexpected\")\n}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	server := NewServer(cfg, store, payments)
	reqHTTP := httptest.NewRequest(http.MethodGet, "/api/projects/"+project.ID+"/repo-scan", nil)
	reqHTTP.Header.Set("Authorization", "Bearer "+auth.Token)
	resp := httptest.NewRecorder()
	server.Routes().ServeHTTP(resp, reqHTTP)
	if resp.Code != http.StatusOK {
		t.Fatalf("repo scan status = %d, body = %s", resp.Code, resp.Body.String())
	}

	body := resp.Body.String()
	for _, value := range []string{
		"scan-client@example.com",
		"+1 555 0155",
		auth.User.ID,
		defaultDevPaymentCode,
		tempDir,
		"super-secret-token",
	} {
		if strings.Contains(body, value) {
			t.Fatalf("repo scan leaked private value %q: %s", value, body)
		}
	}

	var payload ProjectRepositoryScanResponse
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Status != "ready" || payload.Stats.ScannedFiles == 0 || payload.Stats.DependencyFiles != 4 {
		t.Fatalf("unexpected repo scan summary: %#v", payload)
	}
	dependenciesByPath := map[string]RepositoryDependencyFile{}
	for _, dependency := range payload.Dependencies {
		dependenciesByPath[dependency.Path] = dependency
	}
	if dependenciesByPath["package.json"].PackageCount != 2 || dependenciesByPath["pyproject.toml"].PackageCount != 2 || dependenciesByPath["Cargo.toml"].PackageCount != 2 || dependenciesByPath["composer.json"].PackageCount != 3 {
		t.Fatalf("unexpected dependency scan: %#v", payload.Dependencies)
	}
	seenSignals := map[string]bool{}
	for _, finding := range payload.Findings {
		seenSignals[finding.Signal] = true
		if strings.Contains(finding.Body, "super-secret-token") {
			t.Fatalf("finding leaked raw secret: %#v", finding)
		}
	}
	for _, signal := range []string{"lockfile_missing", "dependency_unpinned", "secret_pattern", "todo_fixme", "dangerous_js_execution", "direct_inner_html", "production_panic"} {
		if !seenSignals[signal] {
			t.Fatalf("repo scan missing signal %s: %#v", signal, payload.Findings)
		}
	}
	if payload.Stats.SuggestedTaskCount == 0 || len(payload.SuggestedTasks) == 0 {
		t.Fatalf("repo scan missing suggested tasks: %#v", payload.Stats)
	}
	var taskToFund RepositorySuggestedTask
	for _, task := range payload.SuggestedTasks {
		if task.Signal == "secret_pattern" || task.Signal == "dangerous_js_execution" {
			taskToFund = task
			break
		}
	}
	if taskToFund.ID == "" {
		t.Fatalf("repo scan missing security suggested task: %#v", payload.SuggestedTasks)
	}
	if !taskToFund.ReadyForBounty || !taskToFund.FundingPacket.CanFund || taskToFund.FundingPacket.RecommendedFundingCents < taskToFund.FundingPacket.RecommendedRewardCents || len(taskToFund.FundingPacket.EvidenceChecklist) == 0 {
		t.Fatalf("unexpected funding packet: %#v", taskToFund)
	}
	fundReq := httptest.NewRequest(http.MethodPost, "/api/projects/"+project.ID+"/repo-scan/suggested-tasks/"+taskToFund.ID+"/fund", strings.NewReader(fmt.Sprintf(`{"reward_cents":%d,"budget_cents":%d,"payment_method":"card","payment_reference":%q}`, taskToFund.FundingPacket.RecommendedRewardCents, taskToFund.FundingPacket.RecommendedFundingCents, defaultDevPaymentCode)))
	fundReq.Header.Set("Authorization", "Bearer "+auth.Token)
	fundReq.Header.Set("Content-Type", "application/json")
	fundResp := httptest.NewRecorder()
	server.Routes().ServeHTTP(fundResp, fundReq)
	if fundResp.Code != http.StatusCreated {
		t.Fatalf("repo suggested task fund status = %d, body = %s", fundResp.Code, fundResp.Body.String())
	}
	var fundedPayload FundRepositorySuggestedTaskResponse
	if err := json.Unmarshal(fundResp.Body.Bytes(), &fundedPayload); err != nil {
		t.Fatal(err)
	}
	if fundedPayload.Task == nil || fundedPayload.Task.BountyType != repositoryScanSuggestionBountyType || fundedPayload.Task.RewardCents != taskToFund.FundingPacket.RecommendedRewardCents || fundedPayload.Task.RequiredWorkerKind == "" {
		t.Fatalf("unexpected funded suggested task: %#v", fundedPayload)
	}
	rescanReq := httptest.NewRequest(http.MethodGet, "/api/projects/"+project.ID+"/repo-scan", nil)
	rescanReq.Header.Set("Authorization", "Bearer "+auth.Token)
	rescanResp := httptest.NewRecorder()
	server.Routes().ServeHTTP(rescanResp, rescanReq)
	if rescanResp.Code != http.StatusOK {
		t.Fatalf("repo rescan status = %d, body = %s", rescanResp.Code, rescanResp.Body.String())
	}
	var rescanPayload ProjectRepositoryScanResponse
	if err := json.Unmarshal(rescanResp.Body.Bytes(), &rescanPayload); err != nil {
		t.Fatal(err)
	}
	var seenAlreadyFunded bool
	for _, task := range rescanPayload.SuggestedTasks {
		if task.ID == taskToFund.ID {
			seenAlreadyFunded = task.FundingPacket.Status == "already_funded" && !task.FundingPacket.CanFund
			break
		}
	}
	if !seenAlreadyFunded {
		t.Fatalf("repo rescan did not mark funded suggestion: %#v", rescanPayload.SuggestedTasks)
	}

	protocolReq := httptest.NewRequest(http.MethodGet, "/api/projects/"+project.ID+"/protocol/scan", nil)
	protocolReq.Header.Set("Authorization", "Bearer "+auth.Token)
	protocolResp := httptest.NewRecorder()
	server.Routes().ServeHTTP(protocolResp, protocolReq)
	if protocolResp.Code != http.StatusOK {
		t.Fatalf("repo scan protocol status = %d, body = %s", protocolResp.Code, protocolResp.Body.String())
	}
	protocolBody := protocolResp.Body.String()
	for _, value := range []string{"scan-client@example.com", "+1 555 0155", auth.User.ID, defaultDevPaymentCode, tempDir, "super-secret-token"} {
		if strings.Contains(protocolBody, value) {
			t.Fatalf("repo scan protocol leaked private value %q: %s", value, protocolBody)
		}
	}
	var protocolPayload RepositoryScanProtocolDocument
	if err := json.Unmarshal(protocolResp.Body.Bytes(), &protocolPayload); err != nil {
		t.Fatal(err)
	}
	if protocolPayload.ProtocolVersion != "mergeos.scan.v1" || protocolPayload.Kind != "repository_scan" || protocolPayload.ProjectID != project.ID || protocolPayload.Stats.FindingCount != payload.Stats.FindingCount || protocolPayload.Stats.SuggestedTaskCount != rescanPayload.Stats.SuggestedTaskCount {
		t.Fatalf("unexpected repo scan protocol payload: %#v", protocolPayload)
	}
	if len(protocolPayload.Findings) != len(payload.Findings) {
		t.Fatalf("repo scan protocol findings = %d, want %d", len(protocolPayload.Findings), len(payload.Findings))
	}
	if len(protocolPayload.SuggestedTasks) != len(rescanPayload.SuggestedTasks) {
		t.Fatalf("repo scan protocol suggested tasks = %d, want %d", len(protocolPayload.SuggestedTasks), len(rescanPayload.SuggestedTasks))
	}

	otherAuth, err := store.Register(RegisterRequest{
		Name:     "Other Scan Client",
		Email:    "other-scan-client@example.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatal(err)
	}
	forbiddenReq := httptest.NewRequest(http.MethodGet, "/api/projects/"+project.ID+"/repo-scan", nil)
	forbiddenReq.Header.Set("Authorization", "Bearer "+otherAuth.Token)
	forbiddenResp := httptest.NewRecorder()
	server.Routes().ServeHTTP(forbiddenResp, forbiddenReq)
	if forbiddenResp.Code != http.StatusForbidden {
		t.Fatalf("other client repo scan status = %d", forbiddenResp.Code)
	}
	forbiddenProtocolReq := httptest.NewRequest(http.MethodGet, "/api/projects/"+project.ID+"/protocol/scan", nil)
	forbiddenProtocolReq.Header.Set("Authorization", "Bearer "+otherAuth.Token)
	forbiddenProtocolResp := httptest.NewRecorder()
	server.Routes().ServeHTTP(forbiddenProtocolResp, forbiddenProtocolReq)
	if forbiddenProtocolResp.Code != http.StatusForbidden {
		t.Fatalf("other client repo scan protocol status = %d", forbiddenProtocolResp.Code)
	}
}

func TestWorkerDashboardRouteMatchesGitHubWorkerAndSanitizesData(t *testing.T) {
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
	workerAuth, err := store.AuthenticateGitHub(GitHubAuthProfile{
		ID:        "1001",
		Username:  "worker-dev",
		Name:      "Worker Dev",
		Email:     "worker@example.com",
		AvatarURL: "https://avatars.githubusercontent.com/u/1001",
	}, "", "")
	if err != nil {
		t.Fatal(err)
	}
	clientAuth, err := store.Register(RegisterRequest{
		Name:        "Worker Client",
		CompanyName: "Worker Client Co",
		Email:       "worker-client@example.com",
		Password:    "password123",
	})
	if err != nil {
		t.Fatal(err)
	}
	project, err := store.CreateProject(context.Background(), clientAuth.User.ID, CreateProjectRequest{
		Title:            "Worker dashboard proof",
		ClientName:       "Private Worker Client",
		CompanyName:      "Worker Client Co",
		ClientEmail:      "worker-client@example.com",
		Phone:            "+1 555 0188",
		Brief:            "Create worker dashboard records without exposing private customer data.",
		BudgetCents:      200000,
		PaymentMethod:    PaymentPayPal,
		PaymentReference: defaultDevPaymentCode,
	})
	if err != nil {
		t.Fatal(err)
	}
	var humanTask *Task
	for _, task := range project.Tasks {
		if task.RequiredWorkerKind == WorkerHuman {
			humanTask = task
			break
		}
	}
	if humanTask == nil {
		t.Fatal("project did not create a human task")
	}
	if _, err := store.AcceptTask(humanTask.ID, AcceptTaskRequest{
		WorkerKind: WorkerHuman,
		WorkerID:   "github:worker-dev",
	}); err != nil {
		t.Fatal(err)
	}

	server := NewServer(cfg, store, payments)
	reqHTTP := httptest.NewRequest(http.MethodGet, "/api/workers/me", nil)
	reqHTTP.Header.Set("Authorization", "Bearer "+workerAuth.Token)
	resp := httptest.NewRecorder()
	server.Routes().ServeHTTP(resp, reqHTTP)
	if resp.Code != http.StatusOK {
		t.Fatalf("worker dashboard status = %d, body = %s", resp.Code, resp.Body.String())
	}

	body := resp.Body.String()
	for _, value := range []string{
		"worker-client@example.com",
		"+1 555 0188",
		clientAuth.User.ID,
		defaultDevPaymentCode,
		tempDir,
		humanTask.ID,
	} {
		if strings.Contains(body, value) {
			t.Fatalf("worker dashboard leaked private value %q: %s", value, body)
		}
	}

	var payload WorkerDashboardResponse
	if err := json.Unmarshal(resp.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.ProtocolVersion != "mergeos.worker-dashboard.v1" || payload.Kind != "worker_dashboard" {
		t.Fatalf("unexpected worker dashboard protocol header: %#v", payload)
	}
	if payload.Profile.GitHubUsername != "worker-dev" || payload.Profile.WalletAddress == "" {
		t.Fatalf("worker profile missing linked identity: %#v", payload.Profile)
	}
	if payload.Stats.ClaimedTaskCount != 1 || payload.Stats.RewardCents == 0 || payload.Stats.ReputationScore <= 0 {
		t.Fatalf("unexpected worker stats: %#v", payload.Stats)
	}
	if len(payload.ClaimedTasks) != 1 || payload.ClaimedTasks[0].ProjectTitle != "Worker dashboard proof" {
		t.Fatalf("claimed tasks missing accepted task: %#v", payload.ClaimedTasks)
	}
	if len(payload.Rewards) == 0 {
		t.Fatalf("worker rewards missing payout ledger row: %#v", payload.Rewards)
	}
	if len(payload.Proposals) == 0 {
		t.Fatalf("worker dashboard missing proposal opportunities: %#v", payload.Proposals)
	}
	if payload.Proposals[0].EstimatedHours <= 0 {
		t.Fatalf("worker proposal missing estimated hours: %#v", payload.Proposals[0])
	}
	if len(payload.Proposals[0].MatchReasons) == 0 {
		t.Fatalf("worker proposal missing match reasons: %#v", payload.Proposals[0])
	}
	if len(payload.Proposals[0].EvidenceRequired) == 0 || !containsString(payload.Proposals[0].EvidenceRequired, "tests") {
		t.Fatalf("worker proposal missing evidence requirements: %#v", payload.Proposals[0])
	}
}

func TestWorkerProposalSubmissionRoutesToCustomerDashboardAndAdminOps(t *testing.T) {
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
	workerAuth, err := store.AuthenticateGitHub(GitHubAuthProfile{
		ID:       "proposal-worker-1",
		Username: "proposal-dev",
		Name:     "Proposal Dev",
		Email:    "proposal-dev@example.com",
	}, "", "")
	if err != nil {
		t.Fatal(err)
	}
	clientAuth, err := store.Register(RegisterRequest{
		Name:        "Proposal Client",
		CompanyName: "Proposal Client Co",
		Email:       "proposal-client@example.com",
		Password:    "password123",
	})
	if err != nil {
		t.Fatal(err)
	}
	project, err := store.CreateProject(context.Background(), clientAuth.User.ID, CreateProjectRequest{
		Title:            "Proposal routing proof",
		ClientName:       "Private Proposal Client",
		CompanyName:      "Proposal Client Co",
		ClientEmail:      "proposal-client@example.com",
		Phone:            "+1 555 0199",
		Brief:            "Create proposal routing records without exposing private project metadata.",
		BudgetCents:      220000,
		PaymentMethod:    PaymentPayPal,
		PaymentReference: defaultDevPaymentCode,
	})
	if err != nil {
		t.Fatal(err)
	}
	var humanTask *Task
	for _, task := range project.Tasks {
		if task.RequiredWorkerKind == WorkerHuman {
			humanTask = task
			break
		}
	}
	if humanTask == nil {
		t.Fatal("project did not create a human task")
	}

	publicTaskID := marketplaceBountyID(project.ID, humanTask.IssueNumber)
	body := fmt.Sprintf(`{"task_id":%q,"cover_letter":"I can deliver the acceptance criteria with tests and a deployment note.","bid_cents":12345,"estimated_hours":9.5,"availability":"This week"}`, publicTaskID)
	server := NewServer(cfg, store, payments)
	reqHTTP := httptest.NewRequest(http.MethodPost, "/api/proposals", strings.NewReader(body))
	reqHTTP.Header.Set("Authorization", "Bearer "+workerAuth.Token)
	resp := httptest.NewRecorder()
	server.Routes().ServeHTTP(resp, reqHTTP)
	if resp.Code != http.StatusCreated {
		t.Fatalf("proposal status = %d, body = %s", resp.Code, resp.Body.String())
	}

	responseBody := resp.Body.String()
	for _, value := range []string{
		"proposal-client@example.com",
		"+1 555 0199",
		clientAuth.User.ID,
		defaultDevPaymentCode,
		tempDir,
		humanTask.ID,
	} {
		if strings.Contains(responseBody, value) {
			t.Fatalf("proposal response leaked private value %q: %s", value, responseBody)
		}
	}

	var proposal CreateProposalResponse
	if err := json.Unmarshal(resp.Body.Bytes(), &proposal); err != nil {
		t.Fatal(err)
	}
	if proposal.ProtocolVersion != "mergeos.proposal.v1" || proposal.Kind != "proposal" {
		t.Fatalf("unexpected proposal protocol header: %#v", proposal)
	}
	if proposal.Proposal.TaskID != publicTaskID || proposal.Proposal.ClaimID != publicTaskID || proposal.Proposal.WorkerID != "github:proposal-dev" {
		t.Fatalf("proposal did not expose public task and worker references: %#v", proposal.Proposal)
	}
	if proposal.Proposal.BidCents != 12345 || proposal.Proposal.EstimatedHours != 9.5 || proposal.Proposal.Status != "submitted" {
		t.Fatalf("proposal did not preserve bid and status: %#v", proposal.Proposal)
	}
	if proposal.CustomerNotification.UserID != "" || proposal.WorkerNotification.UserID != "" || strings.Contains(proposal.CustomerNotification.Status, humanTask.ID) {
		t.Fatalf("proposal notifications were not sanitized: %#v %#v", proposal.WorkerNotification, proposal.CustomerNotification)
	}
	proposalFeed := store.PublicLiveFeed(20)
	if proposalFeed.Stats.ProposalCount != 1 {
		t.Fatalf("public feed proposal count = %d, feed = %#v", proposalFeed.Stats.ProposalCount, proposalFeed)
	}
	submittedEventFound := false
	for _, event := range store.PublicEventProtocol(20).Events {
		if event.Type == "proposal.submitted" && event.TaskID == publicTaskID && event.ProjectID == project.ID {
			submittedEventFound = true
			if event.Payload["worker_id"] != nil {
				t.Fatalf("proposal event leaked raw worker payload: %#v", event)
			}
		}
	}
	if !submittedEventFound {
		t.Fatalf("public protocol events missing proposal submitted event: %#v", store.PublicEventProtocol(20).Events)
	}

	workerDashboard := store.WorkerDashboard(workerAuth.User.ID)
	if workerDashboard.Stats.SubmittedProposalCount != 1 || len(workerDashboard.SubmittedProposals) != 1 {
		t.Fatalf("worker dashboard missing submitted proposal: %#v", workerDashboard)
	}
	if workerDashboard.SubmittedProposals[0].TaskID != publicTaskID {
		t.Fatalf("worker dashboard leaked internal task reference: %#v", workerDashboard.SubmittedProposals[0])
	}

	customerDashboard, err := store.ProjectDashboard(project.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(customerDashboard.Proposals) != 1 || customerDashboard.Proposals[0].WorkerID != "github:proposal-dev" {
		t.Fatalf("customer dashboard missing proposal: %#v", customerDashboard.Proposals)
	}

	ops := store.AdminOpsQueue()
	if ops.Stats.ProposalCount != 1 {
		t.Fatalf("admin ops missing proposal review count: %#v", ops.Stats)
	}
	foundProposalOps := false
	for _, item := range ops.Items {
		if item.Type == "proposal_review" && item.ProjectID == project.ID && item.TaskID == humanTask.ID {
			foundProposalOps = true
			break
		}
	}
	if !foundProposalOps {
		t.Fatalf("admin ops missing proposal review item: %#v", ops.Items)
	}

	duplicateReq := httptest.NewRequest(http.MethodPost, "/api/proposals", strings.NewReader(body))
	duplicateReq.Header.Set("Authorization", "Bearer "+workerAuth.Token)
	duplicateResp := httptest.NewRecorder()
	server.Routes().ServeHTTP(duplicateResp, duplicateReq)
	if duplicateResp.Code != http.StatusBadRequest {
		t.Fatalf("duplicate proposal status = %d, body = %s", duplicateResp.Code, duplicateResp.Body.String())
	}
	if store.WorkerDashboard(workerAuth.User.ID).Stats.SubmittedProposalCount != 1 {
		t.Fatal("duplicate proposal created another submitted proposal")
	}

	secondWorkerAuth, err := store.AuthenticateGitHub(GitHubAuthProfile{
		ID:       "proposal-worker-2",
		Username: "backup-dev",
		Name:     "Backup Dev",
		Email:    "backup-dev@example.com",
	}, "", "")
	if err != nil {
		t.Fatal(err)
	}
	secondBody := fmt.Sprintf(`{"task_id":%q,"cover_letter":"I can also ship this with review notes.","bid_cents":15000,"estimated_hours":11,"availability":"Next week"}`, publicTaskID)
	secondReq := httptest.NewRequest(http.MethodPost, "/api/proposals", strings.NewReader(secondBody))
	secondReq.Header.Set("Authorization", "Bearer "+secondWorkerAuth.Token)
	secondResp := httptest.NewRecorder()
	server.Routes().ServeHTTP(secondResp, secondReq)
	if secondResp.Code != http.StatusCreated {
		t.Fatalf("second proposal status = %d, body = %s", secondResp.Code, secondResp.Body.String())
	}

	decisionReq := httptest.NewRequest(http.MethodPost, "/api/proposals/"+proposal.Proposal.ID+"/decision", strings.NewReader(`{"decision":"accepted"}`))
	decisionReq.Header.Set("Authorization", "Bearer "+clientAuth.Token)
	decisionResp := httptest.NewRecorder()
	server.Routes().ServeHTTP(decisionResp, decisionReq)
	if decisionResp.Code != http.StatusOK {
		t.Fatalf("proposal decision status = %d, body = %s", decisionResp.Code, decisionResp.Body.String())
	}
	var decision CreateProposalResponse
	if err := json.Unmarshal(decisionResp.Body.Bytes(), &decision); err != nil {
		t.Fatal(err)
	}
	if decision.Proposal.Status != "accepted" || decision.Proposal.BidCents != 12345 {
		t.Fatalf("proposal decision did not return accepted bid: %#v", decision.Proposal)
	}
	acceptedTask := store.tasks[humanTask.ID]
	if acceptedTask.Status != TaskAccepted || acceptedTask.WorkerID != "github:proposal-dev" || acceptedTask.RewardCents != 12345 {
		t.Fatalf("proposal decision did not accept task with proposal worker and bid: %#v", acceptedTask)
	}
	acceptedProposalEventFound := false
	for _, event := range store.PublicEventProtocol(20).Events {
		if event.Type == "proposal.accepted" && event.TaskID == publicTaskID && event.ProjectID == project.ID {
			acceptedProposalEventFound = true
		}
	}
	if !acceptedProposalEventFound {
		t.Fatalf("public protocol events missing proposal accepted event: %#v", store.PublicEventProtocol(20).Events)
	}

	acceptedWorkerDashboard := store.WorkerDashboard(workerAuth.User.ID)
	if acceptedWorkerDashboard.Stats.ClaimedTaskCount != 1 || acceptedWorkerDashboard.SubmittedProposals[0].Status != "accepted" {
		t.Fatalf("accepted worker dashboard missing accepted proposal and claim: %#v", acceptedWorkerDashboard)
	}
	declinedWorkerDashboard := store.WorkerDashboard(secondWorkerAuth.User.ID)
	if declinedWorkerDashboard.SubmittedProposals[0].Status != "declined" {
		t.Fatalf("unselected worker proposal was not declined: %#v", declinedWorkerDashboard.SubmittedProposals)
	}
	customerDashboard, err = store.ProjectDashboard(project.ID)
	if err != nil {
		t.Fatal(err)
	}
	statusByWorker := map[string]string{}
	for _, row := range customerDashboard.Proposals {
		statusByWorker[row.WorkerID] = row.Status
	}
	if statusByWorker["github:proposal-dev"] != "accepted" || statusByWorker["github:backup-dev"] != "declined" {
		t.Fatalf("customer dashboard proposal statuses not updated: %#v", customerDashboard.Proposals)
	}
	if store.AdminOpsQueue().Stats.ProposalCount != 0 {
		t.Fatalf("accepted proposal left stale admin review item: %#v", store.AdminOpsQueue())
	}
}

func TestWorkerCanSelfClaimProposalRoute(t *testing.T) {
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
	workerAuth, err := store.AuthenticateGitHub(GitHubAuthProfile{
		ID:        "2001",
		Username:  "self-claimer",
		Name:      "Self Claimer",
		Email:     "claimer@example.com",
		AvatarURL: "https://avatars.githubusercontent.com/u/2001",
	}, "", "")
	if err != nil {
		t.Fatal(err)
	}
	clientAuth, err := store.Register(RegisterRequest{
		Name:        "Self Claim Client",
		CompanyName: "Self Claim Co",
		Email:       "self-claim-client@example.com",
		Password:    "password123",
	})
	if err != nil {
		t.Fatal(err)
	}
	project, err := store.CreateProject(context.Background(), clientAuth.User.ID, CreateProjectRequest{
		Title:            "Self claim route",
		ClientName:       "Self Claim Client",
		CompanyName:      "Self Claim Co",
		ClientEmail:      "self-claim-client@example.com",
		Brief:            "Create a bounty that a linked GitHub worker can claim from the dashboard.",
		BudgetCents:      180000,
		PaymentMethod:    PaymentPayPal,
		PaymentReference: defaultDevPaymentCode,
	})
	if err != nil {
		t.Fatal(err)
	}
	humanTasks := []*Task{}
	for _, task := range project.Tasks {
		if task.RequiredWorkerKind == WorkerHuman {
			humanTasks = append(humanTasks, task)
		}
	}
	if len(humanTasks) < 2 {
		t.Fatalf("project did not create enough human tasks: %#v", project.Tasks)
	}
	humanTask := humanTasks[0]
	acceptRouteTask := humanTasks[1]

	dashboard := store.WorkerDashboard(workerAuth.User.ID)
	claimID := ""
	acceptRouteClaimID := ""
	for _, proposal := range dashboard.Proposals {
		if proposal.ProjectID == project.ID && proposal.IssueNumber == humanTask.IssueNumber {
			claimID = proposal.ClaimID
		}
		if proposal.ProjectID == project.ID && proposal.IssueNumber == acceptRouteTask.IssueNumber {
			acceptRouteClaimID = proposal.ClaimID
		}
	}
	if claimID == "" || claimID == humanTask.ID {
		t.Fatalf("worker dashboard proposal missing public claim id for task %q: %#v", humanTask.ID, dashboard.Proposals)
	}
	if acceptRouteClaimID == "" || acceptRouteClaimID == acceptRouteTask.ID {
		t.Fatalf("worker dashboard proposal missing accept-route claim id for task %q: %#v", acceptRouteTask.ID, dashboard.Proposals)
	}

	server := NewServer(cfg, store, payments)
	acceptReq := httptest.NewRequest(http.MethodPost, "/api/tasks/"+acceptRouteClaimID+"/accept", strings.NewReader(`{"worker_kind":"agent","worker_id":"github:spoofed","agent_type":"bad"}`))
	acceptReq.Header.Set("Authorization", "Bearer "+workerAuth.Token)
	acceptResp := httptest.NewRecorder()
	server.Routes().ServeHTTP(acceptResp, acceptReq)
	if acceptResp.Code != http.StatusOK {
		t.Fatalf("accept route self claim status = %d, body = %s", acceptResp.Code, acceptResp.Body.String())
	}
	var acceptedViaAccept TaskClaimResponse
	if err := json.Unmarshal(acceptResp.Body.Bytes(), &acceptedViaAccept); err != nil {
		t.Fatal(err)
	}
	if acceptedViaAccept.ClaimID != acceptRouteClaimID || acceptedViaAccept.TaskID != acceptRouteTask.ID || acceptedViaAccept.WorkerID != "github:self-claimer" {
		t.Fatalf("accept route did not self claim with public claim id: %#v", acceptedViaAccept)
	}

	reqHTTP := httptest.NewRequest(http.MethodPost, "/api/tasks/"+claimID+"/claim", strings.NewReader(`{"worker_kind":"agent","worker_id":"github:spoofed","agent_type":"bad"}`))
	reqHTTP.Header.Set("Authorization", "Bearer "+workerAuth.Token)
	resp := httptest.NewRecorder()
	server.Routes().ServeHTTP(resp, reqHTTP)
	if resp.Code != http.StatusOK {
		t.Fatalf("self claim status = %d, body = %s", resp.Code, resp.Body.String())
	}

	var accepted TaskClaimResponse
	if err := json.Unmarshal(resp.Body.Bytes(), &accepted); err != nil {
		t.Fatal(err)
	}
	if accepted.ProtocolVersion != "mergeos.task-claim.v1" || accepted.Kind != "task_claim" || accepted.ClaimID != claimID {
		t.Fatalf("unexpected self claim protocol header: %#v", accepted)
	}
	if accepted.Status != TaskAccepted || accepted.WorkerKind != WorkerHuman || accepted.WorkerID != "github:self-claimer" || accepted.Task.Status != TaskAccepted {
		t.Fatalf("self claim used wrong worker identity: %#v", accepted)
	}
	if accepted.ProofHash == "" || accepted.AcceptedAt == nil || accepted.TaskID != humanTask.ID || accepted.ProjectID != project.ID {
		t.Fatalf("self claim missing proof fields: %#v", accepted)
	}

	ledgerCount := len(store.ListLedger())
	repeatReq := httptest.NewRequest(http.MethodPost, "/api/tasks/"+humanTask.ID+"/accept", strings.NewReader(`{}`))
	repeatReq.Header.Set("Authorization", "Bearer "+workerAuth.Token)
	repeatResp := httptest.NewRecorder()
	server.Routes().ServeHTTP(repeatResp, repeatReq)
	if repeatResp.Code != http.StatusForbidden && repeatResp.Code != http.StatusBadRequest {
		t.Fatalf("repeat claim status = %d, body = %s", repeatResp.Code, repeatResp.Body.String())
	}
	if len(store.ListLedger()) != ledgerCount {
		t.Fatalf("repeat claim created ledger entries: before=%d after=%d", ledgerCount, len(store.ListLedger()))
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

func TestAdminOpsQueueReturnsDisputeModerationAndPayoutItems(t *testing.T) {
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
		Name:     "Ops Admin",
		Email:    "ops-admin@example.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatal(err)
	}
	clientAuth, err := store.Register(RegisterRequest{
		Name:     "Ops Client",
		Email:    "ops-client@example.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatal(err)
	}
	project, err := store.CreateProject(context.Background(), clientAuth.User.ID, CreateProjectRequest{
		Title:            "Ops queue proof",
		ClientName:       "Ops Client",
		ClientEmail:      "ops-client@example.com",
		Brief:            "Create admin ops queue evidence.",
		BudgetCents:      160000,
		PaymentMethod:    PaymentPayPal,
		PaymentReference: defaultDevPaymentCode,
	})
	if err != nil {
		t.Fatal(err)
	}

	store.mu.Lock()
	closedTask := store.tasks[project.Tasks[0].ID]
	closedTask.IssueState = "closed"
	store.syncProjectTaskSnapshotLocked(store.projects[project.ID], closedTask)
	store.users[adminAuth.User.ID].GitHubUsername = "ops-shared"
	store.users[clientAuth.User.ID].GitHubUsername = "ops-shared"
	store.addNotificationLocked(clientAuth.User.ID, project.ID, "email", "Delivery notice failed", "Customer update could not be sent.", "error:smtp refused")
	store.sslReviews["expired.mergeos.local"] = &SSLReviewStatus{
		Domain:        "expired.mergeos.local",
		Status:        "expired",
		DaysRemaining: -1,
		LastCheckedAt: &closedTask.CreatedAt,
		Error:         "certificate expired",
	}
	store.mu.Unlock()

	if err := store.AddGeminiWebhookLog(GeminiWebhookLog{
		EventName:  "pull_request",
		Action:     "opened",
		Repository: project.BountyRepoName,
		PullNumber: 404,
		Sender:     "ops-reviewer",
		Status:     "unauthorized",
		StatusCode: http.StatusUnauthorized,
		Error:      "bad signature",
		ReceivedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.AddManualCredit("github:ops-reviewer", 5000, "pr:https://github.com/mergeos-bounties/mergeos/pull/404;title:Ops queue proof"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.AddManualCredit("github:ops-reviewer", 5000, "pr:https://github.com/mergeos-bounties/mergeos/pull/404;title:Ops queue proof duplicate"); err != nil {
		t.Fatal(err)
	}

	server := NewServer(cfg, store, payments)
	clientReq := httptest.NewRequest(http.MethodGet, "/api/admin/ops-queue", nil)
	clientReq.Header.Set("Authorization", "Bearer "+clientAuth.Token)
	clientResp := httptest.NewRecorder()
	server.Routes().ServeHTTP(clientResp, clientReq)
	if clientResp.Code != http.StatusForbidden {
		t.Fatalf("client ops queue status = %d", clientResp.Code)
	}

	adminReq := httptest.NewRequest(http.MethodGet, "/api/admin/ops-queue", nil)
	adminReq.Header.Set("Authorization", "Bearer "+adminAuth.Token)
	adminResp := httptest.NewRecorder()
	server.Routes().ServeHTTP(adminResp, adminReq)
	if adminResp.Code != http.StatusOK {
		t.Fatalf("admin ops queue status = %d, body = %s", adminResp.Code, adminResp.Body.String())
	}

	body := adminResp.Body.String()
	if strings.Contains(body, defaultDevPaymentCode) || strings.Contains(body, tempDir) {
		t.Fatalf("admin ops queue leaked hidden implementation value: %s", body)
	}

	var payload AdminOpsQueueResponse
	if err := json.Unmarshal(adminResp.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.ProtocolVersion != "mergeos.admin-ops.v1" || payload.Kind != "admin_ops" {
		t.Fatalf("unexpected admin ops protocol header: %#v", payload)
	}
	if payload.Stats.TotalCount < 7 || payload.Stats.DisputeCount < 1 || payload.Stats.ModerationCount < 2 || payload.Stats.PayoutReviewCount < 3 || payload.Stats.FraudCount < 2 || payload.Stats.SecurityCount < 1 || payload.Stats.CriticalCount < 1 {
		t.Fatalf("unexpected ops queue stats: %#v", payload.Stats)
	}
	seen := map[string]bool{}
	actionSeen := map[string]bool{}
	for _, item := range payload.Items {
		seen[item.Type] = true
		for _, action := range item.Actions {
			actionSeen[item.Type+":"+action.Type] = true
		}
	}
	for _, required := range []string{"payout_review", "payout_audit", "dispute", "moderation", "security_moderation", "fraud_review"} {
		if !seen[required] {
			t.Fatalf("ops queue missing %s item: %#v", required, payload.Items)
		}
	}
	for _, required := range []string{
		"payout_review:review_task_pulls",
		"payout_audit:open_url",
		"security_moderation:run_ssl_review",
		"dispute:refresh_admin_ops",
		"fraud_review:open_url",
		"fraud_review:refresh_admin_ops",
	} {
		if !actionSeen[required] {
			t.Fatalf("ops queue missing action %s: %#v", required, payload.Items)
		}
	}
}

func TestCreateDisputeRouteAddsAdminOpsQueueItem(t *testing.T) {
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
	adminAuth, err := store.Register(RegisterRequest{Name: "Ops Admin", Email: "ops-admin-dispute@example.com", Password: "password123"})
	if err != nil {
		t.Fatal(err)
	}
	clientAuth, err := store.Register(RegisterRequest{Name: "Dispute Client", CompanyName: "Dispute Co", Email: "dispute-client@example.com", Password: "password123"})
	if err != nil {
		t.Fatal(err)
	}
	workerAuth, err := store.Register(RegisterRequest{Name: "Dispute Worker", Email: "dispute-worker@example.com", Password: "password123"})
	if err != nil {
		t.Fatal(err)
	}
	otherAuth, err := store.Register(RegisterRequest{Name: "Other User", Email: "other-dispute@example.com", Password: "password123"})
	if err != nil {
		t.Fatal(err)
	}
	store.mu.Lock()
	store.users[workerAuth.User.ID].GitHubUsername = "worker-dispute"
	store.mu.Unlock()

	project, err := store.CreateProject(context.Background(), clientAuth.User.ID, CreateProjectRequest{
		Title:            "Dispute workflow proof",
		ClientName:       "Private Dispute Client",
		CompanyName:      "Dispute Co",
		ClientEmail:      "dispute-client@example.com",
		Phone:            "+1 555 0188",
		Brief:            "Create dispute queue coverage without leaking private project data.",
		BudgetCents:      180000,
		PaymentMethod:    PaymentPayPal,
		PaymentReference: defaultDevPaymentCode,
	})
	if err != nil {
		t.Fatal(err)
	}
	var humanTask *Task
	for _, task := range project.Tasks {
		if task.RequiredWorkerKind == WorkerHuman {
			humanTask = task
			break
		}
	}
	if humanTask == nil {
		t.Fatal("project did not create a human task")
	}
	if _, err := store.AcceptTask(humanTask.ID, AcceptTaskRequest{WorkerKind: WorkerHuman, WorkerID: "github:worker-dispute"}); err != nil {
		t.Fatal(err)
	}

	server := NewServer(cfg, store, payments)
	clientBody := strings.NewReader(`{"project_id":"` + project.ID + `","subject":"Milestone evidence mismatch","body":"The submitted evidence does not match the deployed result.","severity":"critical"}`)
	clientReq := httptest.NewRequest(http.MethodPost, "/api/disputes", clientBody)
	clientReq.Header.Set("Authorization", "Bearer "+clientAuth.Token)
	clientResp := httptest.NewRecorder()
	server.Routes().ServeHTTP(clientResp, clientReq)
	if clientResp.Code != http.StatusCreated {
		t.Fatalf("client dispute status = %d, body = %s", clientResp.Code, clientResp.Body.String())
	}
	var created CreateDisputeResponse
	if err := json.Unmarshal(clientResp.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if created.ProtocolVersion != "mergeos.dispute.v1" || created.Kind != "dispute" || created.DisputeID == "" || created.Severity != "critical" {
		t.Fatalf("unexpected dispute protocol header: %#v", created)
	}
	if created.ProjectID != project.ID || created.UserID != clientAuth.User.ID || created.Status != "dispute:critical" || created.CreatedAt.IsZero() {
		t.Fatalf("unexpected dispute protocol summary: %#v", created)
	}
	if created.Notification.ProjectID != project.ID || created.Notification.Channel != "dispute" || created.Notification.Status != "dispute:critical" {
		t.Fatalf("unexpected dispute notification: %#v", created.Notification)
	}

	workerBody := strings.NewReader(`{"task_id":"` + humanTask.ID + `","body":"Payment proof needs maintainer review."}`)
	workerReq := httptest.NewRequest(http.MethodPost, "/api/disputes", workerBody)
	workerReq.Header.Set("Authorization", "Bearer "+workerAuth.Token)
	workerResp := httptest.NewRecorder()
	server.Routes().ServeHTTP(workerResp, workerReq)
	if workerResp.Code != http.StatusCreated {
		t.Fatalf("worker dispute status = %d, body = %s", workerResp.Code, workerResp.Body.String())
	}

	otherReq := httptest.NewRequest(http.MethodPost, "/api/disputes", strings.NewReader(`{"project_id":"`+project.ID+`","body":"Unauthorized dispute."}`))
	otherReq.Header.Set("Authorization", "Bearer "+otherAuth.Token)
	otherResp := httptest.NewRecorder()
	server.Routes().ServeHTTP(otherResp, otherReq)
	if otherResp.Code != http.StatusForbidden {
		t.Fatalf("other dispute status = %d, body = %s", otherResp.Code, otherResp.Body.String())
	}

	adminReq := httptest.NewRequest(http.MethodGet, "/api/admin/ops-queue", nil)
	adminReq.Header.Set("Authorization", "Bearer "+adminAuth.Token)
	adminResp := httptest.NewRecorder()
	server.Routes().ServeHTTP(adminResp, adminReq)
	if adminResp.Code != http.StatusOK {
		t.Fatalf("admin ops queue status = %d, body = %s", adminResp.Code, adminResp.Body.String())
	}
	body := adminResp.Body.String()
	for _, value := range []string{"dispute-client@example.com", "+1 555 0188", defaultDevPaymentCode, tempDir} {
		if strings.Contains(body, value) {
			t.Fatalf("admin ops dispute queue leaked private value %q: %s", value, body)
		}
	}
	var queue AdminOpsQueueResponse
	if err := json.Unmarshal(adminResp.Body.Bytes(), &queue); err != nil {
		t.Fatal(err)
	}
	if queue.Stats.DisputeCount < 2 || queue.Stats.CriticalCount < 1 {
		t.Fatalf("ops queue missing dispute stats: %#v", queue.Stats)
	}
	foundCritical := false
	for _, item := range queue.Items {
		if item.Type == "dispute" && item.ProjectID == project.ID && item.Severity == "critical" {
			foundCritical = true
		}
	}
	if !foundCritical {
		t.Fatalf("ops queue missing critical dispute item: %#v", queue.Items)
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

func TestWorkerReputationAuditSurfacesLinkedWalletRisk(t *testing.T) {
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
	client, err := store.Register(RegisterRequest{
		Name:        "Risk Client",
		CompanyName: "Risk Co",
		Email:       "risk-client@example.com",
		Password:    "password123",
	})
	if err != nil {
		t.Fatal(err)
	}
	project, err := store.CreateProject(context.Background(), client.User.ID, CreateProjectRequest{
		Title:            "Reputation audit project",
		ClientName:       "Risk Client",
		ClientEmail:      "risk-client@example.com",
		Brief:            "Create one payable task for reputation scoring.",
		BudgetCents:      200000,
		PaymentMethod:    PaymentPayPal,
		PaymentReference: defaultDevPaymentCode,
	})
	if err != nil {
		t.Fatal(err)
	}
	wallet, err := store.CreateGuestWallet(CreateWalletRequest{})
	if err != nil {
		t.Fatal(err)
	}
	worker, err := store.AuthenticateGitHub(GitHubAuthProfile{
		ID:       "9876",
		Username: "Builder",
		Name:     "Builder",
		Email:    "builder@example.com",
	}, wallet.Address, wallet.RecoveryCode)
	if err != nil {
		t.Fatal(err)
	}

	var task *Task
	for _, candidate := range project.Tasks {
		if candidate.RequiredWorkerKind == WorkerHuman {
			task = candidate
			break
		}
	}
	if task == nil {
		t.Fatal("expected at least one human task")
	}
	if _, err := store.AcceptTask(task.ID, AcceptTaskRequest{
		WorkerKind: WorkerHuman,
		WorkerID:   "github:builder",
	}); err != nil {
		t.Fatal(err)
	}

	dashboard := store.WorkerDashboard(worker.User.ID)
	if dashboard.ReputationAudit.RiskLevel != "low" || !dashboard.ReputationAudit.HasGitHub || !dashboard.ReputationAudit.HasWallet {
		t.Fatalf("worker reputation audit = %#v", dashboard.ReputationAudit)
	}
	if dashboard.Stats.ReputationScore != dashboard.ReputationAudit.Score || dashboard.Stats.RiskLevel != "low" {
		t.Fatalf("worker stats did not mirror audit: %#v", dashboard.Stats)
	}

	marketplace := store.Marketplace()
	var contributor *MarketplaceContributor
	for _, candidate := range marketplace.Contributors {
		if candidate.WorkerID == "github:builder" {
			contributor = candidate
			break
		}
	}
	if contributor == nil {
		t.Fatal("missing marketplace contributor")
	}
	if contributor.RiskLevel != "low" || contributor.ReputationScore == 0 {
		t.Fatalf("marketplace contributor reputation = %#v", contributor)
	}

	adminReputation := store.AdminReputation()
	if adminReputation.Stats.WorkerCount == 0 || adminReputation.Stats.LowRiskCount == 0 {
		t.Fatalf("admin reputation stats = %#v", adminReputation.Stats)
	}
	found := false
	for _, audit := range adminReputation.Workers {
		if audit.WorkerID == "github:builder" {
			found = true
			if audit.RiskLevel != "low" || audit.CompletedTaskCount != 1 {
				t.Fatalf("admin worker audit = %#v", audit)
			}
		}
	}
	if !found {
		t.Fatal("missing admin worker audit for github:builder")
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

func protocolPayloadStringSliceContains(value any, expected string) bool {
	switch typed := value.(type) {
	case []string:
		return containsString(typed, expected)
	case []any:
		for _, item := range typed {
			if fmt.Sprint(item) == expected {
				return true
			}
		}
	}
	return false
}
