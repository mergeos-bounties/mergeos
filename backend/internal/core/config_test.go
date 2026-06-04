package core

import (
	"bytes"
	"net/http/httptest"
	"os"
	"path/filepath"
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
	"CRYPTO_RPC_URL",
	"CRYPTO_RECEIVER",
	"CRYPTO_ASSET",
	"CRYPTO_TOKEN_MINT",
	"CRYPTO_TOKEN_CONTRACT",
	"CRYPTO_TOKEN_DECIMALS",
	"CRYPTO_MIN_CONFIRMATIONS",
	"GITHUB_TOKEN",
	"GITHUB_OWNER",
	"GITHUB_OWNER_TYPE",
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

	cfg := LoadConfig()
	if cfg.CryptoAsset != "spl" {
		t.Fatalf("crypto asset = %q", cfg.CryptoAsset)
	}
	if cfg.CryptoReceiver != receiver || cfg.CryptoTokenContract != mint {
		t.Fatalf("crypto receiver/token = %q/%q", cfg.CryptoReceiver, cfg.CryptoTokenContract)
	}
	if !cfg.CryptoReady() {
		t.Fatal("solana spl config should be ready")
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

func TestLocalOAuthRedirectBaseUsesForwardedFrontendHost(t *testing.T) {
	server := NewServer(Config{Environment: "local"}, nil, nil)
	request := httptest.NewRequest("GET", "http://127.0.0.1:18080/api/auth/google/callback", nil)
	request.Header.Set("X-Forwarded-Proto", "http")
	request.Header.Set("X-Forwarded-Host", "127.0.0.1:15173")

	if got, want := server.getFrontRedirectBase(request), "http://127.0.0.1:15173"; got != want {
		t.Fatalf("redirect base = %q, want %q", got, want)
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
