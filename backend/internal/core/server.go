package core

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
)

type Server struct {
	cfg            Config
	store          *Store
	payments       *PaymentManager
	geminiReviewer *GeminiReviewService
	paypalBaseURL  string // test override for PayPal API base URL
}

func NewServer(cfg Config, store *Store, payments *PaymentManager) *Server {
	return &Server{cfg: cfg, store: store, payments: payments, geminiReviewer: NewGeminiReviewService(cfg, store)}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", s.health)
	mux.HandleFunc("GET /api/config", s.config)
	mux.HandleFunc("GET /api/public/marketplace", s.marketplace)
	mux.HandleFunc("GET /api/public/ledger", s.publicLedger)
	mux.HandleFunc("POST /api/public/repo/issues", s.importRepoIssues)
	mux.HandleFunc("POST /api/integrations/github/pr-review", s.geminiReviewWebhook)
	mux.HandleFunc("POST /api/auth/register", s.register)
	mux.HandleFunc("POST /api/auth/login", s.login)
	mux.HandleFunc("POST /api/auth/github", s.githubLogin)
	mux.HandleFunc("GET /api/auth/me", s.me)
	mux.HandleFunc("POST /api/auth/logout", s.logout)
	mux.HandleFunc("GET /api/auth/google/login", s.googleLogin)
	mux.HandleFunc("GET /api/auth/google/callback", s.googleCallback)
	mux.HandleFunc("GET /api/auth/github/login", s.githubBrowserLogin)
	mux.HandleFunc("GET /api/auth/github/callback", s.githubCallback)
	mux.HandleFunc("POST /api/wallets", s.createWallet)
	mux.HandleFunc("GET /api/wallets/{address}", s.wallet)
	mux.HandleFunc("POST /api/wallets/link", s.linkWallet)
	mux.HandleFunc("POST /api/payments/paypal/orders", s.createPayPalOrder)
	mux.HandleFunc("POST /api/uploads", s.uploadAttachment)
	mux.HandleFunc("GET /api/uploads/", s.downloadAttachment)
	mux.HandleFunc("GET /api/admin/summary", s.adminSummary)
	mux.HandleFunc("GET /api/admin/users", s.adminUsers)
	mux.HandleFunc("PATCH /api/admin/users/{id}", s.updateAdminUser)
	mux.HandleFunc("GET /api/admin/projects", s.adminProjects)
	mux.HandleFunc("GET /api/admin/tasks", s.adminTasks)
	mux.HandleFunc("GET /api/admin/tasks/{id}/pulls", s.adminTaskPullRequests)
	mux.HandleFunc("POST /api/admin/tasks/{id}/pulls/{number}/merge", s.mergeAdminTaskPullRequest)
	mux.HandleFunc("GET /api/admin/notifications", s.adminNotifications)
	mux.HandleFunc("GET /api/admin/attachments", s.adminAttachments)
	mux.HandleFunc("GET /api/admin/ledger", s.adminLedger)
	mux.HandleFunc("GET /api/admin/ssl", s.adminSSLReviews)
	mux.HandleFunc("POST /api/admin/ssl/review", s.reviewAdminSSL)
	mux.HandleFunc("GET /api/admin/gemini/keys", s.adminGeminiKeys)
	mux.HandleFunc("POST /api/admin/gemini/keys", s.addAdminGeminiKey)
	mux.HandleFunc("PATCH /api/admin/gemini/keys/{id}", s.updateAdminGeminiKey)
	mux.HandleFunc("GET /api/admin/gemini/webhooks", s.adminGeminiWebhookLogs)
	mux.HandleFunc("GET /api/projects", s.projects)
	mux.HandleFunc("POST /api/projects", s.createProject)
	mux.HandleFunc("POST /api/projects/evaluate", s.evaluateProject)
	mux.HandleFunc("POST /api/projects/evaluate-price", s.evaluateProjectPrice)
	mux.HandleFunc("GET /api/tasks", s.tasks)
	mux.HandleFunc("POST /api/tasks/", s.acceptTask)
	mux.HandleFunc("GET /api/notifications", s.notifications)
	mux.HandleFunc("GET /api/ledger", s.ledger)
	return withCORS(mux)
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, StatusResponse{
		Service:      "MergeOS API",
		Version:      "0.3.0",
		Environment:  s.cfg.Environment,
		TokenSymbol:  s.cfg.TokenSymbol,
		PaymentMode:  paymentMode(s.cfg),
		RepoProvider: repoProvider(s.cfg),
	})
}

func (s *Server) config(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, RuntimeConfigResponse{
		Environment:       s.cfg.Environment,
		TokenSymbol:       s.cfg.TokenSymbol,
		PaymentMode:       paymentMode(s.cfg),
		RepoProvider:      repoProvider(s.cfg),
		GitHubOAuthReady:  s.cfg.GitHubOAuthReady(),
		GitHubOAuthClient: s.cfg.GitHubOAuthClientID,
		PayPalReady:       s.cfg.PayPalReady(),
		CryptoReady:       s.cfg.CryptoReady(),
		GitHubReady:       s.cfg.GitHubReady(),
		SMTPReady:         s.cfg.SMTPReady(),
		DevPaymentEnabled: s.cfg.DevPaymentEnabled,
		DevPaymentCode:    s.devPaymentCode(),
		CryptoReceiver:    s.cfg.CryptoReceiver,
		CryptoAsset:       s.cfg.CryptoAsset,
		CryptoToken:       s.cfg.CryptoTokenContract,
		BountyRoot:        s.cfg.BountyRoot,
		UploadRoot:        s.cfg.UploadRoot,
		AdminBootstrap:    s.cfg.AdminAutoPromote || strings.TrimSpace(s.cfg.AdminEmail) != "",
		PrimaryDomain:     s.cfg.PrimaryDomain,
		AdminDomain:       s.cfg.AdminDomain,
		ScanDomain:        s.cfg.ScanDomain,
		SSLReviewDomains:  s.cfg.SSLReviewDomains,
	})
}

func (s *Server) importRepoIssues(w http.ResponseWriter, r *http.Request) {
	var req ImportRepoIssuesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	result, err := ImportRepoIssues(r.Context(), s.cfg, req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) marketplace(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.store.Marketplace())
}

func (s *Server) publicLedger(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.store.ListPublicLedger())
}

func (s *Server) register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	auth, err := s.store.Register(req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, auth)
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	auth, err := s.store.Login(req)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, auth)
}

func (s *Server) githubLogin(w http.ResponseWriter, r *http.Request) {
	if !s.cfg.GitHubOAuthReady() {
		writeError(w, http.StatusBadRequest, "github app login is not configured")
		return
	}
	var req GitHubAuthRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	profile, err := FetchGitHubAuthProfile(r.Context(), s.cfg, req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	auth, err := s.store.AuthenticateGitHub(profile, req.WalletAddress, req.RecoveryCode)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, auth)
}

func (s *Server) me(w http.ResponseWriter, r *http.Request) {
	user, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, publicUser(user))
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	s.store.Logout(r.Header.Get("Authorization"))
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) createWallet(w http.ResponseWriter, r *http.Request) {
	var req CreateWalletRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err != io.EOF {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	wallet, err := s.store.CreateGuestWallet(req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, wallet)
}

func (s *Server) wallet(w http.ResponseWriter, r *http.Request) {
	wallet, ok := s.store.WalletSummary(r.PathValue("address"))
	if !ok {
		writeError(w, http.StatusNotFound, "wallet not found")
		return
	}
	writeJSON(w, http.StatusOK, wallet)
}

func (s *Server) linkWallet(w http.ResponseWriter, r *http.Request) {
	user, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	var req LinkWalletRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	updated, err := s.store.LinkWalletToUser(user.ID, req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (s *Server) projects(w http.ResponseWriter, r *http.Request) {
	user, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	userID := user.ID
	if normalizeRole(user.Role) == RoleAdmin {
		userID = ""
	}
	writeJSON(w, http.StatusOK, s.store.ListProjects(userID))
}

func (s *Server) tasks(w http.ResponseWriter, r *http.Request) {
	user, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	userID := user.ID
	if normalizeRole(user.Role) == RoleAdmin {
		userID = ""
	}
	writeJSON(w, http.StatusOK, s.store.ListTasks(userID))
}

func (s *Server) notifications(w http.ResponseWriter, r *http.Request) {
	user, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	userID := user.ID
	if normalizeRole(user.Role) == RoleAdmin {
		userID = ""
	}
	writeJSON(w, http.StatusOK, s.store.ListNotifications(userID))
}

func (s *Server) adminSummary(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	writeJSON(w, http.StatusOK, s.store.AdminSummary())
}

func (s *Server) adminUsers(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	writeJSON(w, http.StatusOK, s.store.ListUsers())
}

func (s *Server) updateAdminUser(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	var req AdminUpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	user, err := s.store.UpdateUser(r.PathValue("id"), req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, user)
}

func (s *Server) adminProjects(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	writeJSON(w, http.StatusOK, s.store.ListProjects(""))
}

func (s *Server) adminTasks(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	s.syncAdminProjectIssues(r.Context())
	writeJSON(w, http.StatusOK, s.store.ListTasks(""))
}

func (s *Server) adminNotifications(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	writeJSON(w, http.StatusOK, s.store.ListNotifications(""))
}

func (s *Server) adminAttachments(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	writeJSON(w, http.StatusOK, s.store.ListAttachments(""))
}

func (s *Server) adminLedger(w http.ResponseWriter, r *http.Request) PayPalWebhookID    string

	CryptoRPCURL           string
	CryptoReceiver         string
	CryptoAsset            string
	CryptoTokenContract    string
	CryptoTokenDecimals    int
	CryptoWeiPerUSDCent    string
	CryptoMinConfirmations int64
	CryptoWebhookSecret    string

	GitHubToken     string
	GitHubOwner     string
	GitHubOwnerType string

	GeminiAPIKeys             []string
	GeminiReviewModel         string
	GeminiReviewWebhookSecret string
	GeminiReviewMaxPatchBytes int64

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
		PayPalWebhookID:    os.Getenv("PAYPAL_WEBHOOK_ID"),

		CryptoRPCURL:           os.Getenv("CRYPTO_RPC_URL"),
		CryptoReceiver:         strings.ToLower(os.Getenv("CRYPTO_RECEIVER")),
		CryptoAsset:            strings.ToLower(getenv("CRYPTO_ASSET", "native")),
		CryptoTokenContract:    strings.ToLower(os.Getenv("CRYPTO_TOKEN_CONTRACT")),
		CryptoTokenDecimals:    int(getenvInt64("CRYPTO_TOKEN_DECIMALS", 6)),
		CryptoWeiPerUSDCent:    os.Getenv("CRYPTO_WEI_PER_USD_CENT"),
		CryptoMinConfirmations: getenvInt64("CRYPTO_MIN_CONFIRMATIONS", 1),
		CryptoWebhookSecret:    os.Getenv("CRYPTO_WEBHOOK_SECRET"),

		GitHubToken:     firstEnv("GITHUB_TOKEN", "MERGEOS_GITHUB_TOKEN"),
		GitHubOwner:     getenv("GITHUB_OWNER", defaultGitHubOwner),
		GitHubOwnerType: strings.ToLower(getenv("GITHUB_OWNER_TYPE", "org")),

		GeminiAPIKeys: splitEnvList(firstEnv(
			"GEMINI_API_KEYS",
			"MERGEOS_GEMINI_API_KEYS",
			"GEMINI_API_KEY",
			"MERGEOS_GEMINI_API_KEY",
		)),
		GeminiReviewModel:         getenv("GEMINI_REVIEW_MODEL", "gemini-2.5-flash"),
		GeminiReviewWebhookSecret: firstEnv("GEMINI_REVIEW_WEBHOOK_SECRET", "MERGEOS_GEMINI_REVIEW_WEBHOOK_SECRET"),
		GeminiReviewMaxPatchBytes: getenvInt64("GEMINI_REVIEW_MAX_PATCH_BYTES", 70000),

		GitHubAppID:             firstEnv("GITHUB_APP_ID", "MERGEOS_GITHUB_APP_ID"),
		GitHubOAuthClientID:     githubOAuthClientID,
		GitHubOAuthClientSecret: githubOAuthClientSecret,

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

func (c Config) PayPalWebhookReady() bool {
	return c.PayPalReady() && c.PayPalWebhookID != ""
}

func (c Config) CryptoReady() bool {
	if c.CryptoRPCURL == "" || c.CryptoReceiver == "" {
		return false
	}
	if c.CryptoAsset == "erc20" {
		return c.CryptoTokenContract != ""
	}
	return c.CryptoWeiPerUSDCent != ""
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
