package core

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	defaultDevPaymentCode     = "LOCAL-PAID"
	defaultTokenSymbol        = "MRG"
	defaultGitHubOwner        = "mergeos-bounties"
	defaultPrimaryDomain      = "mergeos.shop"
	defaultAdminDomain        = "uta.mergeos.shop"
	defaultScanDomain         = "scan.mergeos.shop"
	defaultLocalAdminEmail    = "admin@gmail.com"
	defaultLocalAdminPassword = "Admin123"
)

type Config struct {
	Environment              string
	TokenSymbol              string
	StatePath                string
	DatabaseURL              string
	PlatformFeeBps           int64
	DevPaymentEnabled        bool
	DevPaymentCode           string
	AdminEmail               string
	AdminPassword            string
	AdminName                string
	AdminCompanyName         string
	AdminAutoPromote         bool
	PrimaryDomain            string
	AdminDomain              string
	ScanDomain               string
	SSLReviewEnabled         bool
	SSLReviewDomains         []string
	SSLReviewIntervalMinutes int64
	SSLExpiryWarnDays        int64

	PayPalEnvironment  string
	PayPalClientID     string
	PayPalClientSecret string

	StripePublishableKey string
	StripeSecretKey      string
	StripeWebhookSecret  string

	CryptoRPCURL           string
	CryptoReceiver         string
	CryptoAsset            string
	CryptoTokenContract    string
	CryptoTokenDecimals    int
	CryptoMinConfirmations int64
	CryptoWebhookSecret    string
	SolanaProgramID        string

	GitHubToken     string
	GitHubOwner     string
	GitHubOwnerType string

	GeminiAPIKeys             []string
	LLMAPIKeys                []LLMAPIKeyConfig
	GeminiReviewModel         string
	GeminiReviewWebhookSecret string
	GeminiReviewMaxPatchBytes int64

	GitHubAppID             string
	GitHubOAuthClientID     string
	GitHubOAuthClientSecret string
	OAuthMockEnabled        bool

	BountyRoot string
	UploadRoot string

	SMTPHost     string
	SMTPPort     string
	SMTPUsername string
	SMTPPassword string
	SMTPFrom     string

	GoogleClientID     string
	GoogleClientSecret string
	GitHubClientID     string
	GitHubClientSecret string
}

type LLMAPIKeyConfig struct {
	Provider  string
	Model     string
	KeyValues []string
}

func LoadConfig() Config {
	env := normalizeEnvironment(os.Getenv("MERGEOS_ENV"))
	loadEnvironmentFiles(env)

	statePath := getenv("MERGEOS_STATE_PATH", filepath.Join("data", "mergeos-state.json"))
	bountyRoot := getenv("BOUNTY_ROOT", filepath.Join("..", "bounties"))
	uploadRoot := getenv("UPLOAD_ROOT", filepath.Join("data", "uploads"))
	primaryDomain := cleanDomain(getenv("PRIMARY_DOMAIN", defaultPrimaryDomain))
	adminDomain := cleanDomain(getenv("ADMIN_DOMAIN", defaultAdminDomain))
	scanDomain := cleanDomain(getenv("SCAN_DOMAIN", defaultScanDomain))
	devPaymentDefault := env != "production"
	adminAutoPromoteDefault := env != "production"
	adminEmail := os.Getenv("ADMIN_EMAIL")
	adminPassword := os.Getenv("ADMIN_PASSWORD")
	if env != "production" {
		adminEmail = getenv("ADMIN_EMAIL", defaultLocalAdminEmail)
		adminPassword = getenv("ADMIN_PASSWORD", defaultLocalAdminPassword)
	}
	payPalDefaultEnv := "sandbox"
	if env == "production" {
		payPalDefaultEnv = "live"
	}
	githubOAuthClientID := firstEnv(
		"GITHUB_APP_CLIENT_ID",
		"GITHUB_OAUTH_CLIENT_ID",
		"GITHUB_CLIENT_ID",
		"MERGEOS_GITHUB_APP_CLIENT_ID",
		"MERGEOS_GITHUB_OAUTH_CLIENT_ID",
	)
	githubOAuthClientSecret := firstEnv(
		"GITHUB_APP_CLIENT_SECRET",
		"GITHUB_OAUTH_CLIENT_SECRET",
		"GITHUB_CLIENT_SECRET",
		"MERGEOS_GITHUB_APP_CLIENT_SECRET",
		"MERGEOS_GITHUB_OAUTH_CLIENT_SECRET",
	)
	googleClientID := firstEnv("GOOGLE_CLIENT_ID", "MERGEOS_GOOGLE_CLIENT_ID")
	googleClientSecret := firstEnv("GOOGLE_CLIENT_SECRET", "MERGEOS_GOOGLE_CLIENT_SECRET")
	oauthMockDefault := env != "production"
	geminiAPIKeys := splitEnvList(firstEnv(
		"GEMINI_API_KEYS",
		"MERGEOS_GEMINI_API_KEYS",
		"GEMINI_API_KEY",
		"MERGEOS_GEMINI_API_KEY",
	))
	geminiReviewModel := getenv("GEMINI_REVIEW_MODEL", "gemini-2.5-flash")

	return Config{
		Environment:              env,
		TokenSymbol:              getenv("TOKEN_SYMBOL", defaultTokenSymbol),
		StatePath:                statePath,
		DatabaseURL:              os.Getenv("DATABASE_URL"),
		PlatformFeeBps:           getenvInt64("PLATFORM_FEE_BPS", 1000),
		DevPaymentEnabled:        getenvBool("DEV_PAYMENT_ENABLED", devPaymentDefault),
		DevPaymentCode:           getenv("DEV_PAYMENT_CODE", defaultDevPaymentCode),
		AdminEmail:               adminEmail,
		AdminPassword:            adminPassword,
		AdminName:                getenv("ADMIN_NAME", "MergeOS Admin"),
		AdminCompanyName:         getenv("ADMIN_COMPANY_NAME", "MergeOS"),
		AdminAutoPromote:         getenvBool("ADMIN_AUTO_PROMOTE_FIRST_USER", adminAutoPromoteDefault),
		PrimaryDomain:            primaryDomain,
		AdminDomain:              adminDomain,
		ScanDomain:               scanDomain,
		SSLReviewEnabled:         getenvBool("SSL_REVIEW_ENABLED", true),
		SSLReviewDomains:         sslReviewDomains(primaryDomain, adminDomain, scanDomain),
		SSLReviewIntervalMinutes: getenvInt64("SSL_REVIEW_INTERVAL_MINUTES", 360),
		SSLExpiryWarnDays:        getenvInt64("SSL_EXPIRY_WARN_DAYS", 14),

		PayPalEnvironment:  strings.ToLower(getenv("PAYPAL_ENV", payPalDefaultEnv)),
		PayPalClientID:     os.Getenv("PAYPAL_CLIENT_ID"),
		PayPalClientSecret: os.Getenv("PAYPAL_CLIENT_SECRET"),

		StripePublishableKey: os.Getenv("STRIPE_PUBLISHABLE_KEY"),
		StripeSecretKey:      os.Getenv("STRIPE_SECRET_KEY"),
		StripeWebhookSecret:  os.Getenv("STRIPE_WEBHOOK_SECRET"),

		CryptoRPCURL:           os.Getenv("CRYPTO_RPC_URL"),
		CryptoReceiver:         normalizeWalletAddress(os.Getenv("CRYPTO_RECEIVER")),
		CryptoAsset:            strings.ToLower(getenv("CRYPTO_ASSET", "spl")),
		CryptoTokenContract:    normalizeWalletAddress(firstEnv("CRYPTO_TOKEN_MINT", "CRYPTO_TOKEN_CONTRACT")),
		CryptoTokenDecimals:    int(getenvInt64("CRYPTO_TOKEN_DECIMALS", 6)),
		CryptoMinConfirmations: getenvInt64("CRYPTO_MIN_CONFIRMATIONS", 1),
		CryptoWebhookSecret:    os.Getenv("CRYPTO_WEBHOOK_SECRET"),
		SolanaProgramID:        normalizeWalletAddress(os.Getenv("MRG_SOLANA_PROGRAM_ID")),

		GitHubToken:     firstEnv("GITHUB_TOKEN", "MERGEOS_GITHUB_TOKEN"),
		GitHubOwner:     getenv("GITHUB_OWNER", defaultGitHubOwner),
		GitHubOwnerType: strings.ToLower(getenv("GITHUB_OWNER_TYPE", "org")),

		GeminiAPIKeys:             geminiAPIKeys,
		LLMAPIKeys:                llmAPIKeyConfigsFromEnv(geminiAPIKeys, geminiReviewModel),
		GeminiReviewModel:         geminiReviewModel,
		GeminiReviewWebhookSecret: firstEnv("GEMINI_REVIEW_WEBHOOK_SECRET", "MERGEOS_GEMINI_REVIEW_WEBHOOK_SECRET"),
		GeminiReviewMaxPatchBytes: getenvInt64("GEMINI_REVIEW_MAX_PATCH_BYTES", 70000),

		GitHubAppID:             firstEnv("GITHUB_APP_ID", "MERGEOS_GITHUB_APP_ID"),
		GitHubOAuthClientID:     githubOAuthClientID,
		GitHubOAuthClientSecret: githubOAuthClientSecret,
		OAuthMockEnabled:        getenvBool("MERGEOS_OAUTH_MOCK_ENABLED", getenvBool("OAUTH_MOCK_ENABLED", oauthMockDefault)),

		BountyRoot: bountyRoot,
		UploadRoot: uploadRoot,

		SMTPHost:     os.Getenv("SMTP_HOST"),
		SMTPPort:     getenv("SMTP_PORT", "587"),
		SMTPUsername: os.Getenv("SMTP_USERNAME"),
		SMTPPassword: os.Getenv("SMTP_PASSWORD"),
		SMTPFrom:     getenv("SMTP_FROM", "noreply@mergeos.local"),

		GoogleClientID:     googleClientID,
		GoogleClientSecret: googleClientSecret,
		GitHubClientID:     githubOAuthClientID,
		GitHubClientSecret: githubOAuthClientSecret,
	}
}

func llmAPIKeyConfigsFromEnv(geminiAPIKeys []string, geminiReviewModel string) []LLMAPIKeyConfig {
	configs := []LLMAPIKeyConfig{}
	add := func(provider, model string, keyValues []string) {
		if len(keyValues) == 0 {
			return
		}
		configs = append(configs, LLMAPIKeyConfig{
			Provider:  provider,
			Model:     model,
			KeyValues: keyValues,
		})
	}

	add("gemini", geminiReviewModel, geminiAPIKeys)
	for _, provider := range []string{"openai", "anthropic", "groq", "openrouter", "deepseek", "mistral"} {
		upper := strings.ToUpper(provider)
		keys := splitEnvList(firstEnv(
			upper+"_API_KEYS",
			"MERGEOS_"+upper+"_API_KEYS",
			upper+"_API_KEY",
			"MERGEOS_"+upper+"_API_KEY",
		))
		model := firstEnv(
			upper+"_REVIEW_MODEL",
			"MERGEOS_"+upper+"_REVIEW_MODEL",
			upper+"_MODEL",
			"MERGEOS_"+upper+"_MODEL",
		)
		add(provider, model, keys)
	}
	return configs
}

func sslReviewDomains(primaryDomain, adminDomain, scanDomain string) []string {
	raw := strings.TrimSpace(os.Getenv("SSL_REVIEW_DOMAINS"))
	if raw == "" {
		raw = primaryDomain + "," + adminDomain + "," + scanDomain
	}
	seen := map[string]bool{}
	domains := []string{}
	for _, item := range strings.Split(raw, ",") {
		domain := cleanDomain(item)
		if domain == "" || seen[domain] {
			continue
		}
		seen[domain] = true
		domains = append(domains, domain)
	}
	return domains
}

func cleanDomain(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.TrimPrefix(value, "https://")
	value = strings.TrimPrefix(value, "http://")
	value = strings.Trim(value, "/")
	if host, _, ok := strings.Cut(value, ":"); ok {
		value = host
	}
	if host, _, ok := strings.Cut(value, "/"); ok {
		value = host
	}
	return strings.TrimSpace(value)
}

func normalizeEnvironment(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "prod", "production":
		return "production"
	case "dev", "development", "local", "":
		return "local"
	default:
		return "local"
	}
}

func loadEnvironmentFiles(env string) {
	loadDotEnv(".env." + normalizeEnvironment(env))
	loadDotEnv(".env")
}

func (c Config) PayPalReady() bool {
	return c.PayPalClientID != "" && c.PayPalClientSecret != ""
}

func (c Config) CryptoReady() bool {
	if c.CryptoRPCURL == "" || c.CryptoReceiver == "" {
		return false
	}
	if c.CryptoAsset != "" && c.CryptoAsset != "spl" {
		return false
	}
	return validWalletAddress(c.CryptoReceiver) && validWalletAddress(c.CryptoTokenContract)
}

func (c Config) StripeReady() bool {
	return c.StripePublishableKey != "" && c.StripeSecretKey != "" && c.StripeWebhookSecret != ""
}

func (c Config) GitHubReady() bool {
	return c.GitHubToken != "" && c.GitHubOwner != ""
}

func (c Config) GeminiReviewReady() bool {
	return c.GitHubToken != "" && c.GeminiReviewWebhookSecret != ""
}

func (c Config) GitHubOAuthReady() bool {
	return c.GitHubOAuthClientID != "" && c.GitHubOAuthClientSecret != ""
}

func (c Config) GoogleOAuthReady() bool {
	return c.GoogleClientID != "" && c.GoogleClientSecret != ""
}

func (c Config) OAuthMockReady() bool {
	return c.Environment != "production" && c.OAuthMockEnabled
}

func (c Config) SMTPReady() bool {
	return c.SMTPHost != "" && c.SMTPUsername != "" && c.SMTPPassword != "" && c.SMTPFrom != ""
}

func getenv(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func firstEnv(keys ...string) string {
	for _, key := range keys {
		value := strings.TrimSpace(os.Getenv(key))
		if value != "" {
			return value
		}
	}
	return ""
}

func splitEnvList(value string) []string {
	parts := strings.FieldsFunc(value, func(r rune) bool {
		return r == ',' || r == ';' || r == '\n' || r == '\r'
	})
	result := []string{}
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}

func getenvBool(key string, fallback bool) bool {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func loadDotEnv(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		if key == "" || os.Getenv(key) != "" {
			continue
		}
		value = strings.TrimSpace(value)
		value = strings.Trim(value, `"'`)
		_ = os.Setenv(key, value)
	}
}

func getenvInt64(key string, fallback int64) int64 {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return fallback
	}
	return parsed
}
