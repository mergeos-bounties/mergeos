package core

import (
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
	"CRYPTO_TOKEN_CONTRACT",
	"CRYPTO_TOKEN_DECIMALS",
	"CRYPTO_WEI_PER_USD_CENT",
	"CRYPTO_MIN_CONFIRMATIONS",
	"GITHUB_TOKEN",
	"GITHUB_OWNER",
	"GITHUB_OWNER_TYPE",
	"GITHUB_OAUTH_CLIENT_ID",
	"GITHUB_OAUTH_CLIENT_SECRET",
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
