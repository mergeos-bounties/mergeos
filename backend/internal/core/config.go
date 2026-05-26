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

	CryptoRPCURL           string
	CryptoReceiver         string
	CryptoAsset            string
	CryptoTokenContract    string
	CryptoTokenDecimals    int
	CryptoWeiPerUSDCent    string
	CryptoMinConfirmations int64

	GitHubToken     string
	GitHubOwner     string
	GitHubOwnerType string

	GitHubAppID             string
	GitHubOAuthClientID     string
	GitHubOAuthClientSecret string

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

	LLMApiKey    string
	LLMModel     string
	LLMProvider  string
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
	if env == "production" { payPalDefaultEnv = "live" }
	githubOAuthClientID := firstEnv("GITHUB_APP_CLIENT_ID","GITHUB_OAUTH_CLIENT_ID","GITHUB_CLIENT_ID","MERGEOS_GITHUB_APP_CLIENT_ID","MERGEOS_GITHUB_OAUTH_CLIENT_ID")
	githubOAuthClientSecret := firstEnv("GITHUB_APP_CLIENT_SECRET","GITHUB_OAUTH_CLIENT_SECRET","GITHUB_CLIENT_SECRET","MERGEOS_GITHUB_APP_CLIENT_SECRET","MERGEOS_GITHUB_OAUTH_CLIENT_SECRET")
	googleClientID := firstEnv("GOOGLE_CLIENT_ID","MERGEOS_GOOGLE_CLIENT_ID")
	googleClientSecret := firstEnv("GOOGLE_CLIENT_SECRET","MERGEOS_GOOGLE_CLIENT_SECRET")

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
		PayPalEnvironment:        strings.ToLower(getenv("PAYPAL_ENV", payPalDefaultEnv)),
		PayPalClientID:           os.Getenv("PAYPAL_CLIENT_ID"),
		PayPalClientSecret:       os.Getenv("PAYPAL_CLIENT_SECRET"),
		CryptoRPCURL:             os.Getenv("CRYPTO_RPC_URL"),
		CryptoReceiver:           strings.ToLower(os.Getenv("CRYPTO_RECEIVER")),
		CryptoAsset:              strings.ToLower(getenv("CRYPTO_ASSET", "native")),
		CryptoTokenContract:      strings.ToLower(os.Getenv("CRYPTO_TOKEN_CONTRACT")),
		CryptoTokenDecimals:      int(getenvInt64("CRYPTO_TOKEN_DECIMALS", 6)),
		CryptoWeiPerUSDCent:      os.Getenv("CRYPTO_WEI_PER_USD_CENT"),
		CryptoMinConfirmations:   getenvInt64("CRYPTO_MIN_CONFIRMATIONS", 1),
		GitHubToken:              os.Getenv("GITHUB_TOKEN"),
		GitHubOwner:              getenv("GITHUB_OWNER", defaultGitHubOwner),
		GitHubOwnerType:          strings.ToLower(getenv("GITHUB_OWNER_TYPE", "org")),
		GitHubAppID:              firstEnv("GITHUB_APP_ID","MERGEOS_GITHUB_APP_ID"),
		GitHubOAuthClientID:      githubOAuthClientID,
		GitHubOAuthClientSecret:  githubOAuthClientSecret,
		BountyRoot:               bountyRoot,
		UploadRoot:               uploadRoot,
		SMTPHost:                 os.Getenv("SMTP_HOST"),
		SMTPPort:                 getenv("SMTP_PORT", "587"),
		SMTPUsername:             os.Getenv("SMTP_USERNAME"),
		SMTPPassword:             os.Getenv("SMTP_PASSWORD"),
		SMTPFrom:                 getenv("SMTP_FROM", "noreply@mergeos.local"),
		GoogleClientID:           googleClientID,
		GoogleClientSecret:       googleClientSecret,
		GitHubClientID:           githubOAuthClientID,
		GitHubClientSecret:       githubOAuthClientSecret,
		LLMApiKey:                os.Getenv("LLM_API_KEY"),
		LLMModel:                 getenv("LLM_MODEL", "gpt-4o-mini"),
		LLMProvider:              getenv("LLM_PROVIDER", "openai"),
	}
}
