package core

import (
	"bytes"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var configEnvKeys = []string{
	"MERGEOS_ENV",
	"MERGEOS_STATE_PATH",
	"DATABASE_URL",
	"TOKEN_SYMBOL",
	"PLATFORM_FEE_BPS",
	"DEV_PAYMENT_ENABLED",
	"DEV_PAYMENT_CODE",
	"ADMIN_EMAIL",
	"ADMIN_PASSWORD",
	"ADMIN_NAME",
	"ADMIN_COMPANY_NAME",
	"ADMIN_AUTO_PROMOTE_FIRST_USER",
	"PRIMARY_DOMAIN",
	"ADMIN_DOMAIN",
	"SCAN_DOMAIN",
	"SSL_REVIEW_DOMAINS",
	"SSL_REVIEW_ENABLED",
	"SSL_REVIEW_INTERVAL_MINUTES",
	"SSL_EXPIRY_WARN_DAYS",
	"PAYPAL_ENV",
	"PAYPAL_CLIENT_ID",
	"PAYPAL_CLIENT_SECRET",
	"PAYPAL_WEBHOOK_ID",
	"CRYPTO_RPC_URL",
	"CRYPTO_RECEIVER",
	"CRYPTO_ASSET",
	"CRYPTO_TOKEN_MINT",
	"CRYPTO_TOKEN_CONTRACT",
	"CRYPTO_TOKEN_DECIMALS",
	"CRYPTO_MIN_CONFIRMATIONS",
	"CRYPTO_WEBHOOK_SECRET",
	"MRG_SOLANA_PROGRAM_ID",
	"GITHUB_TOKEN",
	"GITHUB_OWNER",
	"GITHUB_OWNER_TYPE",
	"GEMINI_API_KEYS",
	"MERGEOS_GEMINI_API_KEYS",
	"GEMINI_API_KEY",
	"MERGEOS_GEMINI_API_KEY",
	"GEMINI_REVIEW_MODEL",
	"OPENAI_API_KEYS",
	"MERGEOS_OPENAI_API_KEYS",
	"OPENAI_API_KEY",
	"MERGEOS_OPENAI_API_KEY",
	"OPENAI_REVIEW_MODEL",
	"MERGEOS_OPENAI_REVIEW_MODEL",
	"OPENAI_MODEL",
	"MERGEOS_OPENAI_MODEL",
	"ANTHROPIC_API_KEYS",
	"MERGEOS_ANTHROPIC_API_KEYS",
	"ANTHROPIC_API_KEY",
	"MERGEOS_ANTHROPIC_API_KEY",
	"ANTHROPIC_REVIEW_MODEL",
	"MERGEOS_ANTHROPIC_REVIEW_MODEL",
	"ANTHROPIC_MODEL",
	"MERGEOS_ANTHROPIC_MODEL",
	"GROQ_API_KEYS",
	"MERGEOS_GROQ_API_KEYS",
	"GROQ_API_KEY",
	"MERGEOS_GROQ_API_KEY",
	"GROQ_REVIEW_MODEL",
	"MERGEOS_GROQ_REVIEW_MODEL",
	"GROQ_MODEL",
	"MERGEOS_GROQ_MODEL",
	"OPENROUTER_API_KEYS",
	"MERGEOS_OPENROUTER_API_KEYS",
	"OPENROUTER_API_KEY",
	"MERGEOS_OPENROUTER_API_KEY",
	"OPENROUTER_REVIEW_MODEL",
	"MERGEOS_OPENROUTER_REVIEW_MODEL",
	"OPENROUTER_MODEL",
	"MERGEOS_OPENROUTER_MODEL",
	"DEEPSEEK_API_KEYS",
	"MERGEOS_DEEPSEEK_API_KEYS",
	"DEEPSEEK_API_KEY",
	"MERGEOS_DEEPSEEK_API_KEY",
	"DEEPSEEK_REVIEW_MODEL",
	"MERGEOS_DEEPSEEK_REVIEW_MODEL",
	"DEEPSEEK_MODEL",
	"MERGEOS_DEEPSEEK_MODEL",
	"MISTRAL_API_KEYS",
	"MERGEOS_MISTRAL_API_KEYS",
	"MISTRAL_API_KEY",
	"MERGEOS_MISTRAL_API_KEY",
	"MISTRAL_REVIEW_MODEL",
	"MERGEOS_MISTRAL_REVIEW_MODEL",
	"MISTRAL_MODEL",
	"MERGEOS_MISTRAL_MODEL",
	"GITHUB_APP_ID",
	"GITHUB_APP_CLIENT_ID",
	"GITHUB_APP_CLIENT_SECRET",
	"GITHUB_OAUTH_CLIENT_ID",
	"GITHUB_OAUTH_CLIENT_SECRET",
	"GITHUB_CLIENT_ID",
	"GITHUB_CLIENT_SECRET",
	"GOOGLE_CLIENT_ID",
	"GOOGLE_CLIENT_SECRET",
	"MERGEOS_GOOGLE_CLIENT_ID",
	"MERGEOS_GOOGLE_CLIENT_SECRET",
	"OAUTH_MOCK_ENABLED",
	"MERGEOS_OAUTH_MOCK_ENABLED",
	"MERGEOS_GITHUB_APP_ID",
	"MERGEOS_GITHUB_APP_CLIENT_ID",
	"MERGEOS_GITHUB_APP_CLIENT_SECRET",
	"MERGEOS_GITHUB_OAUTH_CLIENT_ID",
	"MERGEOS_GITHUB_OAUTH_CLIENT_SECRET",
	"BOUNTY_ROOT",
	"UPLOAD_ROOT",
	"SMTP_HOST",
	"SMTP_PORT",
	"SMTP_USERNAME",
	"SMTP_PASSWORD",
	"SMTP_FROM",
}

func TestLoadConfigUsesLocalEnvFileBeforeFallback(t *testing.T) {
	withTempConfigDir(t)
	clearConfigEnv(t)

	writeEnvFile(t, ".env.local", "TOKEN_SYMBOL=LOCAL\nDEV_PAYMENT_ENABLED=true\n")
	writeEnvFile(t, ".env", "TOKEN_SYMBOL=BASE\nGITHUB_OWNER=base-owner\n")

	cfg := LoadConfig()
	if cfg.Environment != "local" {
		t.Fatalf("environment = %q", cfg.Environment)
	}
	if cfg.TokenSymbol != "LOCAL" {
		t.Fatalf("token symbol = %q", cfg.TokenSymbol)
	}
	if cfg.GitHubOwner != "base-owner" {
		t.Fatalf("github owner = %q", cfg.GitHubOwner)
	}
	if !cfg.DevPaymentEnabled {
		t.Fatal("local dev payment should be enabled")
	}
}

func TestLoadConfigLocalDefaultsIncludeAdminBootstrap(t *testing.T) {
	withTempConfigDir(t)
	clearConfigEnv(t)

	cfg := LoadConfig()
	if cfg.AdminEmail != defaultLocalAdminEmail {
		t.Fatalf("admin email = %q", cfg.AdminEmail)
	}
	if cfg.AdminPassword != defaultLocalAdminPassword {
		t.Fatalf("admin password = %q", cfg.AdminPassword)
	}
}

func TestLoadConfigDefaultsCryptoToSolanaSPL(t *testing.T) {
	withTempConfigDir(t)
	clearConfigEnv(t)
	receiver := base58Encode(bytes.Repeat([]byte{4}, walletAddressBytes))
	mint := base58Encode(bytes.Repeat([]byte{5}, walletAddressBytes))
	t.Setenv("CRYPTO_RPC_URL", "https://api.mainnet-beta.solana.com")
	t.Setenv("CRYPTO_RECEIVER", receiver)
	t.Setenv("CRYPTO_TOKEN_MINT", mint)
	t.Setenv("MRG_SOLANA_PROGRAM_ID", base58Encode(bytes.Repeat([]byte{6}, walletAddressBytes)))

	cfg := LoadConfig()
	if cfg.CryptoAsset != "spl" {
		t.Fatalf("crypto asset = %q", cfg.CryptoAsset)
	}
	if cfg.CryptoReceiver != receiver || cfg.CryptoTokenContract != mint {
		t.Fatalf("crypto receiver/token = %q/%q", cfg.CryptoReceiver, cfg.CryptoTokenContract)
	}
	if !validWalletAddress(cfg.SolanaProgramID) {
		t.Fatalf("solana program id = %q", cfg.SolanaProgramID)
	}
	if !cfg.CryptoReady() {
		t.Fatal("solana spl config should be ready")
	}
}

func TestLoadConfigDoesNotInventSolanaProgramID(t *testing.T) {
	withTempConfigDir(t)
	clearConfigEnv(t)

	cfg := LoadConfig()

	if cfg.SolanaProgramID != "" {
		t.Fatalf("solana program id default = %q, want empty until deployment", cfg.SolanaProgramID)
	}
}

func TestDeployWorkflowPassesSolanaRuntimeConfig(t *testing.T) {
	workflowPath := filepath.Join("..", "..", "..", ".github", "workflows", "deploy.yml")
	source, err := os.ReadFile(workflowPath)
	if err != nil {
		t.Fatalf("read deploy workflow: %v", err)
	}
	workflow := string(source)
	for _, name := range []string{
		"CRYPTO_RPC_URL",
		"CRYPTO_RECEIVER",
		"CRYPTO_ASSET",
		"CRYPTO_TOKEN_MINT",
		"CRYPTO_TOKEN_CONTRACT",
		"CRYPTO_TOKEN_DECIMALS",
		"CRYPTO_MIN_CONFIRMATIONS",
		"CRYPTO_WEBHOOK_SECRET",
		"MRG_SOLANA_PROGRAM_ID",
	} {
		if !strings.Contains(workflow, name+": ${{ secrets."+name+" }}") {
			t.Fatalf("deploy workflow does not map %s from GitHub secrets", name)
		}
		if !strings.Contains(workflow, "Environment="+name+"=$"+name) {
			t.Fatalf("systemd service does not export %s", name)
		}
	}
	if !strings.Contains(workflow, `MRG_SOLANA_PROGRAM_ID="$MRG_SOLANA_PROGRAM_ID"`) {
		t.Fatal("fallback nohup path does not export MRG_SOLANA_PROGRAM_ID")
	}
}

func TestLoadConfigProductionDefaultsAreStrict(t *testing.T) {
	withTempConfigDir(t)
	clearConfigEnv(t)
	t.Setenv("MERGEOS_ENV", "production")

	cfg := LoadConfig()
	if cfg.Environment != "production" {
		t.Fatalf("environment = %q", cfg.Environment)
	}
	if cfg.DevPaymentEnabled {
		t.Fatal("production dev payment should default to disabled")
	}
	if cfg.AdminAutoPromote {
		t.Fatal("production admin auto promote should default to disabled")
	}
	if cfg.AdminEmail != "" {
		t.Fatalf("production admin email should not default, got %q", cfg.AdminEmail)
	}
	if cfg.AdminPassword != "" {
		t.Fatal("production admin password should not default")
	}
	if cfg.PayPalEnvironment != "live" {
		t.Fatalf("paypal env = %q", cfg.PayPalEnvironment)
	}
	if cfg.ScanDomain != defaultScanDomain {
		t.Fatalf("scan domain = %q", cfg.ScanDomain)
	}
	if len(cfg.SSLReviewDomains) != 3 {
		t.Fatalf("ssl review domains = %#v", cfg.SSLReviewDomains)
	}
}

func TestLoadConfigRealEnvWinsOverEnvFiles(t *testing.T) {
	withTempConfigDir(t)
	clearConfigEnv(t)
	t.Setenv("TOKEN_SYMBOL", "REAL")

	writeEnvFile(t, ".env.local", "TOKEN_SYMBOL=LOCAL\n")

	cfg := LoadConfig()
	if cfg.TokenSymbol != "REAL" {
		t.Fatalf("token symbol = %q", cfg.TokenSymbol)
	}
}

func TestLoadConfigUsesGitHubAppCredentialsForOAuth(t *testing.T) {
	withTempConfigDir(t)
	clearConfigEnv(t)

	t.Setenv("GITHUB_APP_ID", "12345")
	t.Setenv("GITHUB_APP_CLIENT_ID", "app-client")
	t.Setenv("GITHUB_APP_CLIENT_SECRET", "app-secret")
	t.Setenv("GITHUB_OAUTH_CLIENT_ID", "legacy-client")
	t.Setenv("GITHUB_OAUTH_CLIENT_SECRET", "legacy-secret")

	cfg := LoadConfig()
	if cfg.GitHubAppID != "12345" {
		t.Fatalf("github app id = %q", cfg.GitHubAppID)
	}
	if cfg.GitHubOAuthClientID != "app-client" {
		t.Fatalf("github oauth client id = %q", cfg.GitHubOAuthClientID)
	}
	if cfg.GitHubOAuthClientSecret != "app-secret" {
		t.Fatalf("github oauth client secret = %q", cfg.GitHubOAuthClientSecret)
	}
	if cfg.GitHubClientID != cfg.GitHubOAuthClientID || cfg.GitHubClientSecret != cfg.GitHubOAuthClientSecret {
		t.Fatal("legacy github client fields should use the same GitHub App credentials")
	}
}

func TestLoadConfigSeedsLLMProviderAPIKeysFromEnv(t *testing.T) {
	withTempConfigDir(t)
	clearConfigEnv(t)

	t.Setenv("GEMINI_API_KEY", "gemini-env-key")
	t.Setenv("GEMINI_REVIEW_MODEL", "gemini-2.0-flash")
	t.Setenv("OPENAI_API_KEYS", "sk-openai-one,sk-openai-two")
	t.Setenv("OPENAI_REVIEW_MODEL", "gpt-4.1-mini")
	t.Setenv("MERGEOS_ANTHROPIC_API_KEY", "anthropic-env-key")

	cfg := LoadConfig()
	if len(cfg.GeminiAPIKeys) != 1 || cfg.GeminiAPIKeys[0] != "gemini-env-key" {
		t.Fatalf("legacy gemini keys = %#v", cfg.GeminiAPIKeys)
	}
	byProvider := map[string]LLMAPIKeyConfig{}
	for _, item := range cfg.LLMAPIKeys {
		byProvider[item.Provider] = item
	}
	if got := byProvider["gemini"]; got.Model != "gemini-2.0-flash" || len(got.KeyValues) != 1 || got.KeyValues[0] != "gemini-env-key" {
		t.Fatalf("gemini LLM key config = %#v", got)
	}
	if got := byProvider["openai"]; got.Model != "gpt-4.1-mini" || len(got.KeyValues) != 2 {
		t.Fatalf("openai LLM key config = %#v", got)
	}
	if got := byProvider["anthropic"]; got.Model != "" || len(got.KeyValues) != 1 || got.KeyValues[0] != "anthropic-env-key" {
		t.Fatalf("anthropic LLM key config = %#v", got)
	}
}

func TestLoadConfigUsesMergeOSGoogleCredentials(t *testing.T) {
	withTempConfigDir(t)
	clearConfigEnv(t)

	t.Setenv("MERGEOS_GOOGLE_CLIENT_ID", "google-client")
	t.Setenv("MERGEOS_GOOGLE_CLIENT_SECRET", "google-secret")

	cfg := LoadConfig()
	if cfg.GoogleClientID != "google-client" {
		t.Fatalf("google client id = %q", cfg.GoogleClientID)
	}
	if cfg.GoogleClientSecret != "google-secret" {
		t.Fatalf("google client secret = %q", cfg.GoogleClientSecret)
	}
}

func TestOAuthMockReadinessIsNeverEnabledInProduction(t *testing.T) {
	withTempConfigDir(t)
	clearConfigEnv(t)

	t.Setenv("MERGEOS_ENV", "production")
	t.Setenv("MERGEOS_OAUTH_MOCK_ENABLED", "true")

	cfg := LoadConfig()
	if cfg.OAuthMockReady() {
		t.Fatal("production OAuth mock flow must stay disabled even when explicitly requested")
	}
}

func TestLocalOAuthRedirectBaseUsesForwardedFrontendHost(t *testing.T) {
	server := NewServer(Config{Environment: "local"}, nil, nil)
	request := httptest.NewRequest("GET", "http://127.0.0.1:18080/api/auth/google/callback", nil)
	request.Header.Set("X-Forwarded-Proto", "http")
	request.Header.Set("X-Forwarded-Host", "127.0.0.1:15173")

	if got, want := server.getFrontRedirectBase(request), "http://127.0.0.1:15173"; got != want {
		t.Fatalf("redirect base = %q, want %q", got, want)
	}
}

func TestOAuthRedirectBaseOnlyDowngradesExactLoopbackHosts(t *testing.T) {
	server := NewServer(Config{PrimaryDomain: "mergeos.shop"}, nil, nil)

	spoofed := httptest.NewRequest("GET", "https://evil-localhost.example/api/auth/google/callback", nil)
	spoofed.Host = "evil-localhost.example"
	if got, want := server.getFrontRedirectBase(spoofed), "https://mergeos.shop"; got != want {
		t.Fatalf("spoofed redirect base = %q, want %q", got, want)
	}

	loopback := httptest.NewRequest("GET", "http://127.0.0.1:18080/api/auth/google/callback", nil)
	if got, want := server.getFrontRedirectBase(loopback), "http://mergeos.shop"; got != want {
		t.Fatalf("loopback redirect base = %q, want %q", got, want)
	}
}

func withTempConfigDir(t *testing.T) {
	t.Helper()
	previousDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	tempDir := t.TempDir()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(previousDir); err != nil {
			t.Fatalf("restore cwd: %v", err)
		}
	})
}

func clearConfigEnv(t *testing.T) {
	t.Helper()
	for _, key := range configEnvKeys {
		t.Setenv(key, "")
	}
}

func writeEnvFile(t *testing.T, name, contents string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(".", name), []byte(contents), 0644); err != nil {
		t.Fatal(err)
	}
}
