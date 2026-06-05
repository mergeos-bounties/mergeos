package core

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type Server struct {
	cfg            Config
	store          *Store
	payments       *PaymentManager
	geminiReviewer *GeminiReviewService
	eventHub       *eventHub
}

func NewServer(cfg Config, store *Store, payments *PaymentManager) *Server {
	return &Server{cfg: cfg, store: store, payments: payments, geminiReviewer: NewGeminiReviewService(cfg, store), eventHub: newEventHub()}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", s.health)
	mux.HandleFunc("GET /api/config", s.config)
	mux.HandleFunc("GET /api/public/marketplace", s.marketplace)
	mux.HandleFunc("GET /api/public/ledger", s.publicLedger)
	mux.HandleFunc("GET /api/public/ledger/verify", s.publicLedgerVerify)
	mux.HandleFunc("GET /api/public/live-feed", s.publicLiveFeed)
	mux.HandleFunc("GET /api/public/protocol", s.publicProtocolManifest)
	mux.HandleFunc("GET /api/public/protocol/ledger", s.publicProtocolLedger)
	mux.HandleFunc("GET /api/public/protocol/tasks", s.publicProtocolTasks)
	mux.HandleFunc("GET /api/public/protocol/agents", s.publicProtocolAgents)
	mux.HandleFunc("GET /api/public/protocol/events", s.publicProtocolEvents)
	mux.HandleFunc("GET /api/public/projects/{id}/deployment", s.publicProjectDeployment)
	mux.HandleFunc("POST /api/public/repo/issues", s.importRepoIssues)
	mux.HandleFunc("POST /api/integrations/github/pr-review", s.geminiReviewWebhook)
	mux.HandleFunc("POST /api/payments/crypto/webhook", s.cryptoWebhook)
	mux.HandleFunc("POST /api/auth/register", s.register)
	mux.HandleFunc("POST /api/auth/login", s.login)
	mux.HandleFunc("POST /api/auth/password-reset", s.requestPasswordReset)
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
	mux.HandleFunc("POST /api/wallets/migrations", s.createWalletMigration)
	mux.HandleFunc("POST /api/payments/paypal/orders", s.createPayPalOrder)
	mux.HandleFunc("POST /api/payments/paypal/webhook", s.handlePayPalWebhook)
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
	mux.HandleFunc("GET /api/admin/ops-queue", s.adminOpsQueue)
	mux.HandleFunc("GET /api/admin/reputation", s.adminReputation)
	mux.HandleFunc("POST /api/admin/ledger/credits", s.createAdminLedgerCredit)
	mux.HandleFunc("GET /api/admin/settings", s.adminSettings)
	mux.HandleFunc("PATCH /api/admin/settings", s.updateAdminSettings)
	mux.HandleFunc("GET /api/admin/ssl", s.adminSSLReviews)
	mux.HandleFunc("POST /api/admin/ssl/review", s.reviewAdminSSL)
	mux.HandleFunc("GET /api/admin/gemini/keys", s.adminGeminiKeys)
	mux.HandleFunc("POST /api/admin/gemini/keys", s.addAdminGeminiKey)
	mux.HandleFunc("PATCH /api/admin/gemini/keys/{id}", s.updateAdminGeminiKey)
	mux.HandleFunc("POST /api/admin/gemini/keys/{id}/test", s.testAdminGeminiKey)
	mux.HandleFunc("GET /api/admin/gemini/webhooks", s.adminGeminiWebhookLogs)
	mux.HandleFunc("GET /api/projects", s.projects)
	mux.HandleFunc("GET /api/projects/{id}/escrow", s.projectEscrow)
	mux.HandleFunc("GET /api/projects/{id}/payouts", s.projectPayouts)
	mux.HandleFunc("GET /api/projects/{id}/dashboard", s.projectDashboard)
	mux.HandleFunc("GET /api/projects/{id}/pull-requests", s.projectPullRequests)
	mux.HandleFunc("GET /api/projects/{id}/deployment", s.projectDeployment)
	mux.HandleFunc("GET /api/projects/{id}/ai-workflow", s.projectAIWorkflow)
	mux.HandleFunc("GET /api/projects/{id}/task-graph", s.projectTaskGraph)
	mux.HandleFunc("GET /api/projects/{id}/protocol/workflow", s.projectWorkflowProtocol)
	mux.HandleFunc("GET /api/projects/{id}/repo-scan", s.projectRepositoryScan)
	mux.HandleFunc("GET /api/projects/{id}/protocol/scan", s.projectRepositoryScanProtocol)
	mux.HandleFunc("POST /api/projects/{id}/repo-sync", s.syncProjectRepoIssues)
	mux.HandleFunc("POST /api/projects/{id}/agent-actions", s.createProjectAgentAction)
	mux.HandleFunc("POST /api/projects", s.createProject)
	mux.HandleFunc("POST /api/projects/evaluate", s.evaluateProject)
	mux.HandleFunc("POST /api/projects/evaluate-price", s.evaluateProjectPrice)
	mux.HandleFunc("POST /api/projects/evaluate-llm", s.evaluateProjectWithLLM)
	mux.HandleFunc("GET /api/tasks", s.tasks)
	mux.HandleFunc("POST /api/tasks/", s.acceptTask)
	mux.HandleFunc("GET /api/workers/me", s.workerDashboard)
	mux.HandleFunc("GET /api/notifications", s.notifications)
	mux.HandleFunc("POST /api/notifications/read", s.markNotificationRead)
	mux.HandleFunc("POST /api/notifications/read-all", s.markAllNotificationsRead)
	mux.HandleFunc("POST /api/disputes", s.createDispute)
	mux.HandleFunc("GET /api/ws", s.wsHandler)
	mux.HandleFunc("GET /api/ledger", s.ledger)

	// Test publish settings - admin endpoints
	mux.HandleFunc("GET /api/admin/test-settings", s.adminGetTestSettings)
	mux.HandleFunc("PATCH /api/admin/test-settings", s.adminUpdateTestSettings)
	mux.HandleFunc("GET /api/admin/test-settings/entries", s.adminListTestEntries)
	mux.HandleFunc("POST /api/admin/test-settings/entries", s.adminAddTestEntry)
	mux.HandleFunc("PATCH /api/admin/test-settings/entries/{id}", s.adminUpdateTestEntry)
	mux.HandleFunc("DELETE /api/admin/test-settings/entries/{id}", s.adminDeleteTestEntry)

	// Test publish settings - public endpoints (password-gated)
	mux.HandleFunc("POST /api/public/test-settings/auth", s.publicTestSettingsAuth)
	mux.HandleFunc("POST /api/public/test-settings/entries/list", s.publicListTestEntries)
	mux.HandleFunc("POST /api/public/test-settings/entries", s.publicAddTestEntry)
	mux.HandleFunc("POST /api/public/test-settings/entries/{id}/reveal", s.publicRevealTestEntry)
	mux.HandleFunc("PATCH /api/public/test-settings/entries/{id}", s.publicUpdateTestEntry)
	mux.HandleFunc("DELETE /api/public/test-settings/entries/{id}", s.publicDeleteTestEntry)
	mux.HandleFunc("GET /api/public/test-settings/status", s.publicTestStatus)
	return withCORS(s.cfg, mux)
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
		StripeReady:       s.cfg.StripeReady(),
		StripePublicKey:   s.cfg.StripePublishableKey,
		PaymentRails:      paymentRails(s.cfg),
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

func (s *Server) publicLedgerVerify(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, s.store.VerifyLedger())
}

func (s *Server) publicLiveFeed(w http.ResponseWriter, r *http.Request) {
	limit := 0
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil {
			limit = parsed
		}
	}
	writeJSON(w, http.StatusOK, s.store.PublicLiveFeed(limit))
}

func (s *Server) publicProtocolManifest(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, ProtocolManifest())
}

func (s *Server) publicProtocolLedger(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, LedgerProtocolResponse{
		ProtocolVersion: "mergeos.ledger.v1",
		Kind:            "ledger",
		TokenSymbol:     normalizedTokenSymbol(s.cfg.TokenSymbol),
		Verification:    s.store.VerifyLedger(),
		Entries:         s.store.ListPublicLedger(),
	})
}

func (s *Server) publicProtocolEvents(w http.ResponseWriter, r *http.Request) {
	limit := 0
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil {
			limit = parsed
		}
	}
	writeJSON(w, http.StatusOK, s.store.PublicEventProtocol(limit))
}

func (s *Server) publicProtocolAgents(w http.ResponseWriter, r *http.Request) {
	limit := 0
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil {
			limit = parsed
		}
	}
	writeJSON(w, http.StatusOK, s.store.PublicAgentProtocol(limit))
}

func (s *Server) publicProtocolTasks(w http.ResponseWriter, r *http.Request) {
	limit := 0
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil {
			limit = parsed
		}
	}
	writeJSON(w, http.StatusOK, s.store.PublicTaskProtocol(limit))
}

func (s *Server) publicProjectDeployment(w http.ResponseWriter, r *http.Request) {
	deployment, err := s.store.PublicProjectDeployment(r.PathValue("id"))
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, deployment)
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

func (s *Server) requestPasswordReset(w http.ResponseWriter, r *http.Request) {
	var req PasswordResetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	response, err := s.store.RequestPasswordReset(req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, response)
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
	user, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	var req CreateWalletRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err != io.EOF {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	wallet, err := s.store.CreateUserWallet(user.ID, req)
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

func (s *Server) createWalletMigration(w http.ResponseWriter, r *http.Request) {
	user, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	var req CreateWalletMigrationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	migration, err := s.store.CreateWalletMigration(user.ID, req, s.cfg)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, migration)
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

func (s *Server) projectDeployment(w http.ResponseWriter, r *http.Request) {
	user, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	projectID := strings.TrimSpace(r.PathValue("id"))
	if !s.store.CanAccessProject(user.ID, user.Role, projectID) {
		writeError(w, http.StatusForbidden, "project access is required")
		return
	}
	deployment, err := s.store.ProjectDeployment(projectID)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, deployment)
}

func (s *Server) projectEscrow(w http.ResponseWriter, r *http.Request) {
	user, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	projectID := strings.TrimSpace(r.PathValue("id"))
	if !s.store.CanAccessProject(user.ID, user.Role, projectID) {
		writeError(w, http.StatusForbidden, "project access is required")
		return
	}
	escrow, err := s.store.ProjectEscrow(projectID)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, escrow)
}

func (s *Server) projectPayouts(w http.ResponseWriter, r *http.Request) {
	user, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	projectID := strings.TrimSpace(r.PathValue("id"))
	if !s.store.CanAccessProject(user.ID, user.Role, projectID) {
		writeError(w, http.StatusForbidden, "project access is required")
		return
	}
	payouts, err := s.store.ProjectPayouts(projectID)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, payouts)
}

func (s *Server) projectDashboard(w http.ResponseWriter, r *http.Request) {
	user, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	projectID := strings.TrimSpace(r.PathValue("id"))
	if !s.store.CanAccessProject(user.ID, user.Role, projectID) {
		writeError(w, http.StatusForbidden, "project access is required")
		return
	}
	dashboard, err := s.store.ProjectDashboard(projectID)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	project, ok := s.store.ProjectSnapshot(projectID)
	if !ok {
		writeError(w, http.StatusNotFound, "project not found")
		return
	}
	client, err := newAdminGitHubClient(s.cfg, false)
	if err != nil {
		dashboard.PullRequestError = sanitizeLedgerReferenceValue(err.Error())
	} else {
		dashboard.PullRequests = projectPullRequestsMonitor(r.Context(), client, project)
		if dashboard.PullRequests.UpdatedAt.After(dashboard.UpdatedAt) {
			dashboard.UpdatedAt = dashboard.PullRequests.UpdatedAt
			dashboard.Project.UpdatedAt = dashboard.PullRequests.UpdatedAt
		}
	}
	writeJSON(w, http.StatusOK, dashboard)
}

func (s *Server) projectPullRequests(w http.ResponseWriter, r *http.Request) {
	user, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	projectID := strings.TrimSpace(r.PathValue("id"))
	if !s.store.CanAccessProject(user.ID, user.Role, projectID) {
		writeError(w, http.StatusForbidden, "project access is required")
		return
	}
	project, ok := s.store.ProjectSnapshot(projectID)
	if !ok {
		writeError(w, http.StatusNotFound, "project not found")
		return
	}
	client, err := newAdminGitHubClient(s.cfg, false)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, projectPullRequestsMonitor(r.Context(), client, project))
}

func (s *Server) projectAIWorkflow(w http.ResponseWriter, r *http.Request) {
	user, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	projectID := strings.TrimSpace(r.PathValue("id"))
	if !s.store.CanAccessProject(user.ID, user.Role, projectID) {
		writeError(w, http.StatusForbidden, "project access is required")
		return
	}
	workflow, err := s.store.ProjectAIWorkflow(projectID)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, workflow)
}

func (s *Server) projectTaskGraph(w http.ResponseWriter, r *http.Request) {
	user, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	projectID := strings.TrimSpace(r.PathValue("id"))
	if !s.store.CanAccessProject(user.ID, user.Role, projectID) {
		writeError(w, http.StatusForbidden, "project access is required")
		return
	}
	graph, err := s.store.ProjectTaskGraph(projectID)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, graph)
}

func (s *Server) projectWorkflowProtocol(w http.ResponseWriter, r *http.Request) {
	user, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	projectID := strings.TrimSpace(r.PathValue("id"))
	if !s.store.CanAccessProject(user.ID, user.Role, projectID) {
		writeError(w, http.StatusForbidden, "project access is required")
		return
	}
	document, err := s.store.ProjectWorkflowProtocol(projectID)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, document)
}

func (s *Server) projectRepositoryScan(w http.ResponseWriter, r *http.Request) {
	user, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	projectID := strings.TrimSpace(r.PathValue("id"))
	if !s.store.CanAccessProject(user.ID, user.Role, projectID) {
		writeError(w, http.StatusForbidden, "project access is required")
		return
	}
	scan, err := s.store.ProjectRepositoryScan(projectID)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, scan)
}

func (s *Server) projectRepositoryScanProtocol(w http.ResponseWriter, r *http.Request) {
	user, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	projectID := strings.TrimSpace(r.PathValue("id"))
	if !s.store.CanAccessProject(user.ID, user.Role, projectID) {
		writeError(w, http.StatusForbidden, "project access is required")
		return
	}
	document, err := s.store.ProjectRepositoryScanProtocol(projectID)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, document)
}

func (s *Server) syncProjectRepoIssues(w http.ResponseWriter, r *http.Request) {
	user, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	projectID := strings.TrimSpace(r.PathValue("id"))
	if !s.store.CanAccessProject(user.ID, user.Role, projectID) {
		writeError(w, http.StatusForbidden, "project access is required")
		return
	}
	project, ok := s.store.ProjectSnapshot(projectID)
	if !ok {
		writeError(w, http.StatusNotFound, "project not found")
		return
	}
	repoURL := projectSourceRepoURL(project)
	if repoURL == "" {
		writeError(w, http.StatusBadRequest, "source repository is not configured for this project")
		return
	}
	imported, err := ImportRepoIssues(r.Context(), s.cfg, ImportRepoIssuesRequest{RepoURL: repoURL})
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	report, err := s.store.SyncProjectImportedIssuesReport(projectID, imported.RepoURL, imported.Issues)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := s.store.RecordRepoIssueSyncEvent(report); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.broadcastLiveFeedEvent("repo_issues_synced")
	writeJSON(w, http.StatusOK, report)
}

func (s *Server) createProjectAgentAction(w http.ResponseWriter, r *http.Request) {
	user, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	projectID := strings.TrimSpace(r.PathValue("id"))
	if !s.store.CanAccessProject(user.ID, user.Role, projectID) {
		writeError(w, http.StatusForbidden, "project access is required")
		return
	}
	var req AgentActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	response, err := s.store.RecordProjectAgentAction(projectID, req)
	if err != nil {
		status := http.StatusBadRequest
		if strings.Contains(err.Error(), "not found") {
			status = http.StatusNotFound
		}
		writeError(w, status, err.Error())
		return
	}
	s.broadcastLiveFeedEvent("agent_action")
	writeJSON(w, http.StatusCreated, response)
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

func (s *Server) createDispute(w http.ResponseWriter, r *http.Request) {
	user, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	var req CreateDisputeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if strings.TrimSpace(req.TaskID) != "" {
		taskID, err := s.store.ResolveTaskClaimID(req.TaskID)
		if err != nil {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		req.TaskID = taskID
	}
	response, err := s.store.CreateDispute(user.ID, user.Role, req)
	if err != nil {
		status := http.StatusBadRequest
		if strings.Contains(err.Error(), "access") {
			status = http.StatusForbidden
		} else if strings.Contains(err.Error(), "not found") {
			status = http.StatusNotFound
		}
		writeError(w, status, err.Error())
		return
	}
	s.broadcastAdminOpsUpdated()
	writeJSON(w, http.StatusCreated, response)
}

func (s *Server) markNotificationRead(w http.ResponseWriter, r *http.Request) {
	user, ok := s.requireUser(w, r)
	if !ok {
		return
	}

	var req struct {
		NotificationID string `json:"notification_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	note := s.store.MarkNotificationRead(user.ID, req.NotificationID)
	if note == nil {
		http.Error(w, "notification not found", http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, note)
}

func (s *Server) markAllNotificationsRead(w http.ResponseWriter, r *http.Request) {
	user, ok := s.requireUser(w, r)
	if !ok {
		return
	}

	s.store.MarkAllNotificationsRead(user.ID)
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) wsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := wsUpgrade(w, r)
	if err != nil {
		return
	}
	s.eventHub.add(conn)
	if err := s.writeWSInitialEvents(conn); err != nil {
		s.eventHub.remove(conn)
		conn.close()
		return
	}
	go conn.readLoop(s.eventHub)
}

func (s *Server) writeWSInitialEvents(conn *wsConn) error {
	for _, event := range s.wsInitialEvents() {
		data, err := json.Marshal(event)
		if err != nil {
			return err
		}
		if err := conn.writeText(data); err != nil {
			return err
		}
	}
	return nil
}

func (s *Server) wsInitialEvents() []map[string]interface{} {
	now := time.Now().UTC()
	return []map[string]interface{}{
		{
			"protocol_version": "mergeos.event.v1",
			"kind":             "connection",
			"type":             "connection_ready",
			"status":           "ok",
			"token_symbol":     normalizedTokenSymbol(s.cfg.TokenSymbol),
			"created_at":       now,
		},
		{
			"protocol_version": "mergeos.event.v1",
			"kind":             "snapshot",
			"type":             "live_feed_snapshot",
			"feed":             s.store.PublicLiveFeed(20),
			"events":           s.store.PublicEventProtocol(20),
			"created_at":       now,
		},
	}
}

func (s *Server) broadcastLiveFeedEvent(eventType string) {
	feed := s.store.PublicLiveFeed(20)
	payload := map[string]interface{}{
		"type":       eventType,
		"feed":       feed,
		"created_at": time.Now().UTC(),
	}
	if event := protocolEventForBroadcast(eventType, feed); event != nil {
		payload["event"] = event
		payload["protocol_type"] = event.Type
	}
	s.eventHub.broadcastAll(payload)
}

func (s *Server) broadcastAdminOpsUpdated() {
	s.eventHub.broadcastAll(map[string]interface{}{
		"protocol_version": "mergeos.event.v1",
		"kind":             "admin_ops_signal",
		"type":             "admin_ops_updated",
		"created_at":       time.Now().UTC(),
	})
}

func protocolEventForBroadcast(eventType string, feed PublicLiveFeedResponse) *EventProtocolDocument {
	for _, item := range feed.Items {
		if !broadcastMatchesFeedType(eventType, item.Type) {
			continue
		}
		event := publicLiveFeedProtocolEvent(item)
		return &event
	}
	return nil
}

func broadcastMatchesFeedType(eventType, feedType string) bool {
	switch strings.TrimSpace(eventType) {
	case "project_created":
		return feedType == "project_funded"
	default:
		return strings.TrimSpace(eventType) == feedType
	}
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

func (s *Server) adminLedger(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	writeJSON(w, http.StatusOK, s.store.ListLedger())
}

func (s *Server) adminOpsQueue(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	writeJSON(w, http.StatusOK, s.store.AdminOpsQueue())
}

func (s *Server) adminReputation(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	writeJSON(w, http.StatusOK, s.store.AdminReputation())
}

func (s *Server) adminSettings(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	writeJSON(w, http.StatusOK, s.store.AdminSettings())
}

func (s *Server) updateAdminSettings(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	var req UpdateAdminSettingsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	settings, err := s.store.UpdateAdminSettings(req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, settings)
}

func (s *Server) adminSSLReviews(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	writeJSON(w, http.StatusOK, s.store.ListSSLReviews())
}

func (s *Server) reviewAdminSSL(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	reviews, err := s.store.ReviewSSLNow(r.Context(), "manual")
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, reviews)
}

func (s *Server) adminGeminiKeys(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	writeJSON(w, http.StatusOK, s.store.ListGeminiAPIKeyStats())
}

func (s *Server) addAdminGeminiKey(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	var req AddGeminiAPIKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	keyValue := strings.TrimSpace(req.KeyValue)
	if keyValue == "" {
		keyValue = strings.TrimSpace(req.APIKey)
	}
	key, err := s.store.AddGeminiAPIKey(keyValue, req.Provider, req.Model)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, key)
}

func (s *Server) updateAdminGeminiKey(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	var req UpdateGeminiAPIKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	key, err := s.store.UpdateGeminiAPIKey(r.PathValue("id"), req.Status, req.ResetCounts)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, key)
}

func (s *Server) testAdminGeminiKey(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	var req TestGeminiAPIKeyRequest
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if strings.TrimSpace(string(body)) != "" {
		if err := json.Unmarshal(body, &req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
	}
	if s.geminiReviewer == nil {
		writeError(w, http.StatusServiceUnavailable, "LLM reviewer is not configured")
		return
	}
	result, err := s.geminiReviewer.TestAPIKey(r.Context(), r.PathValue("id"), req.Provider, req.Model)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) adminGeminiWebhookLogs(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAdmin(w, r); !ok {
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	writeJSON(w, http.StatusOK, s.store.ListGeminiWebhookLogs(limit))
}

func (s *Server) uploadAttachment(w http.ResponseWriter, r *http.Request) {
	user, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	if err := r.ParseMultipartForm(maxUploadBytes * 3); err != nil {
		writeError(w, http.StatusBadRequest, "invalid multipart upload")
		return
	}
	files := r.MultipartForm.File["files"]
	if len(files) == 0 {
		if file, header, err := r.FormFile("file"); err == nil {
			_ = file.Close()
			files = append(files, header)
		}
	}
	if len(files) == 0 {
		writeError(w, http.StatusBadRequest, "at least one file is required")
		return
	}
	attachments := make([]*Attachment, 0, len(files))
	for _, header := range files {
		attachment, err := s.store.SaveAttachment(user.ID, header)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		attachments = append(attachments, attachment)
	}
	writeJSON(w, http.StatusCreated, attachments)
}

func (s *Server) downloadAttachment(w http.ResponseWriter, r *http.Request) {
	user, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	path := strings.TrimPrefix(r.URL.Path, "/api/uploads/")
	id := strings.TrimSuffix(path, "/download")
	if id == "" || id == path {
		writeError(w, http.StatusNotFound, "route not found")
		return
	}
	attachment, ok := s.store.AttachmentForDownload(id)
	if !ok {
		writeError(w, http.StatusNotFound, "attachment not found")
		return
	}
	if normalizeRole(user.Role) != RoleAdmin && attachment.UserID != user.ID {
		writeError(w, http.StatusForbidden, "admin access is required")
		return
	}
	w.Header().Set("Content-Type", attachment.ContentType)
	w.Header().Set("Content-Disposition", "inline; filename=\""+strings.ReplaceAll(attachment.OriginalName, "\"", "")+"\"")
	http.ServeFile(w, r, attachment.StoredPath)
}

func (s *Server) ledger(w http.ResponseWriter, r *http.Request) {
	user, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	if normalizeRole(user.Role) == RoleAdmin {
		writeJSON(w, http.StatusOK, s.store.ListLedger())
		return
	}
	writeJSON(w, http.StatusOK, s.store.ListLedgerForUser(user.ID))
}

func (s *Server) evaluateProjectPrice(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireUser(w, r); !ok {
		return
	}
	var req ProjectPriceEvaluationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	result, err := EvaluateProjectPrice(req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) evaluateProjectWithLLM(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireUser(w, r); !ok {
		return
	}
	var req LLMPriceEvaluationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	result, err := s.EvaluateProjectLLM(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) createProject(w http.ResponseWriter, r *http.Request) {
	user, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	var req CreateProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	project, err := s.store.CreateProject(r.Context(), user.ID, req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	s.broadcastLiveFeedEvent("project_created")
	writeJSON(w, http.StatusCreated, project)
}

func (s *Server) createPayPalOrder(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireUser(w, r); !ok {
		return
	}
	var req CreatePayPalOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	order, err := s.payments.CreatePayPalOrder(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, order)
}

func (s *Server) acceptTask(w http.ResponseWriter, r *http.Request) {
	user, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	path := strings.TrimPrefix(r.URL.Path, "/api/tasks/")
	taskID := strings.TrimSuffix(path, "/accept")
	if taskID == "" || taskID == path {
		writeError(w, http.StatusNotFound, "route not found")
		return
	}
	claimID := taskID
	resolvedTaskID, err := s.store.ResolveTaskClaimID(taskID)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	taskID = resolvedTaskID

	var req AcceptTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err != io.EOF {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if !s.store.CanAccessTask(user.ID, user.Role, taskID) {
		selfReq, err := s.store.SelfAcceptTaskRequest(user.ID, taskID)
		if err != nil {
			writeError(w, http.StatusForbidden, err.Error())
			return
		}
		req = selfReq
	}

	task, err := s.store.AcceptTask(taskID, req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	s.broadcastLiveFeedEvent("task_accepted")
	writeJSON(w, http.StatusOK, taskClaimProtocolDocument(claimID, task))
}

func (s *Server) workerDashboard(w http.ResponseWriter, r *http.Request) {
	user, ok := s.requireUser(w, r)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, s.store.WorkerDashboard(user.ID))
}

func (s *Server) requireUser(w http.ResponseWriter, r *http.Request) (*User, bool) {
	user, ok := s.store.UserByToken(r.Header.Get("Authorization"))
	if !ok {
		writeError(w, http.StatusUnauthorized, "login is required")
		return nil, false
	}
	return user, true
}

func (s *Server) requireAdmin(w http.ResponseWriter, r *http.Request) (*User, bool) {
	user, ok := s.requireUser(w, r)
	if !ok {
		return nil, false
	}
	if normalizeRole(user.Role) != RoleAdmin {
		writeError(w, http.StatusForbidden, "admin access is required")
		return nil, false
	}
	return user, true
}

func withCORS(cfg Config, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if origin := corsAllowedOrigin(cfg, r.Header.Get("Origin")); origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		}
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type,Authorization,X-Hub-Signature-256,X-GitHub-Event,X-GitHub-Delivery,X-MergeOS-Signature,X-MergeOS-Event")
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PATCH,DELETE,OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func corsAllowedOrigin(cfg Config, origin string) string {
	origin = strings.TrimSpace(origin)
	env := normalizeEnvironment(cfg.Environment)
	if origin == "" {
		if env == "production" {
			return ""
		}
		return "*"
	}

	parsed, err := url.Parse(origin)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return ""
	}
	scheme := strings.ToLower(parsed.Scheme)
	host := strings.ToLower(parsed.Hostname())
	if env != "production" && (host == "localhost" || host == "127.0.0.1" || host == "::1") && (scheme == "http" || scheme == "https") {
		return origin
	}
	if scheme != "https" {
		return ""
	}
	for _, domain := range []string{cfg.PrimaryDomain, cfg.AdminDomain, cfg.ScanDomain} {
		if host == cleanDomain(domain) {
			return origin
		}
	}
	return ""
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func paymentMode(cfg Config) string {
	if cfg.PayPalReady() || cfg.CryptoReady() || cfg.StripeReady() {
		return "live-adapters"
	}
	if cfg.DevPaymentEnabled {
		return "local-dev-verifier"
	}
	return "not-configured"
}

func paymentRails(cfg Config) []PaymentRailOption {
	devEnabled := cfg.DevPaymentEnabled
	paypalEnabled := cfg.PayPalReady() || devEnabled
	cryptoEnabled := cfg.CryptoReady() || devEnabled
	usdtReady := cfg.CryptoReady() && cfg.CryptoAsset == "spl" && strings.TrimSpace(cfg.CryptoTokenContract) != ""
	usdtEnabled := usdtReady || devEnabled
	stripeEnabled := cfg.StripeReady() || devEnabled
	return []PaymentRailOption{
		{
			ID:                "paypal",
			Label:             "PayPal",
			Method:            string(PaymentPayPal),
			Caption:           "Sandbox/live checkout",
			Enabled:           paypalEnabled,
			Ready:             cfg.PayPalReady(),
			DisabledReason:    disabledPaymentRailReason(paypalEnabled, "PayPal credentials are not configured."),
			RequiresReference: !devEnabled,
		},
		{
			ID:                "crypto",
			Label:             publicCryptoRailLabel(cfg),
			Method:            string(PaymentCrypto),
			Caption:           "Solana SPL transfer",
			Enabled:           cryptoEnabled,
			Ready:             cfg.CryptoReady(),
			DisabledReason:    disabledPaymentRailReason(cryptoEnabled, "Solana SPL verifier is not configured."),
			RequiresReference: !devEnabled,
			Asset:             strings.ToUpper(strings.TrimSpace(cfg.CryptoAsset)),
			Receiver:          cfg.CryptoReceiver,
			TokenContract:     cfg.CryptoTokenContract,
		},
		{
			ID:                "usdt",
			Label:             "Solana SPL",
			Method:            string(PaymentUSDT),
			Caption:           "Backward-compatible Solana SPL rail",
			Enabled:           usdtEnabled,
			Ready:             usdtReady,
			DisabledReason:    disabledPaymentRailReason(usdtEnabled, "Solana SPL verifier is not configured."),
			RequiresReference: !devEnabled,
			Asset:             "SPL",
			Receiver:          cfg.CryptoReceiver,
			TokenContract:     cfg.CryptoTokenContract,
		},
		{
			ID:                "stripe",
			Label:             "Credit / Debit card",
			Method:            string(PaymentStripe),
			Caption:           "Stripe PaymentIntent",
			Enabled:           stripeEnabled,
			Ready:             cfg.StripeReady(),
			DisabledReason:    disabledPaymentRailReason(stripeEnabled, "Stripe verifier is not configured."),
			RequiresReference: !devEnabled,
			PublicKey:         cfg.StripePublishableKey,
		},
		{
			ID:                "bank",
			Label:             "Bank transfer",
			Method:            "bank",
			Caption:           "Manual treasury rail",
			Enabled:           false,
			Ready:             false,
			DisabledReason:    "Bank transfer requires manual treasury review.",
			RequiresReference: true,
		},
	}
}

func disabledPaymentRailReason(enabled bool, reason string) string {
	if enabled {
		return ""
	}
	return reason
}

func publicCryptoRailLabel(cfg Config) string {
	if cfg.CryptoAsset == "spl" && strings.TrimSpace(cfg.CryptoTokenContract) != "" {
		return "Solana SPL"
	}
	return "Solana"
}

func repoProvider(cfg Config) string {
	if cfg.GitHubReady() {
		return "github-private:" + cfg.GitHubOwner
	}
	return "local-git"
}

func (s *Server) devPaymentCode() string {
	if !s.cfg.DevPaymentEnabled {
		return ""
	}
	return s.cfg.DevPaymentCode
}

func (s *Server) evaluateProject(w http.ResponseWriter, r *http.Request) {
	_, ok := s.requireUser(w, r)
	if !ok {
		return
	}

	var req EvaluateProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	var basePrice int64 = 1000

	tech := strings.ToLower(req.TechStack)
	if strings.Contains(tech, "react") || strings.Contains(tech, "vue") || strings.Contains(tech, "next") {
		basePrice += 300
	}
	if strings.Contains(tech, "go") || strings.Contains(tech, "rust") || strings.Contains(tech, "fastapi") {
		basePrice += 400
	}
	if strings.Contains(tech, "ai") || strings.Contains(tech, "llm") || strings.Contains(tech, "machine learning") {
		basePrice += 800
	}
	if strings.Contains(tech, "kubernetes") || strings.Contains(tech, "docker") || strings.Contains(tech, "devops") {
		basePrice += 500
	}

	basePrice += int64(len(req.Deliverables) * 150)
	basePrice += int64(len(req.Requirements) * 100)

	complexity := strings.ToLower(req.Complexity)
	if complexity == "high" {
		basePrice = int64(float64(basePrice) * 1.6)
	} else if complexity == "low" {
		basePrice = int64(float64(basePrice) * 0.8)
	}

	if req.ReferenceBudget > 0 {
		basePrice = (basePrice + req.ReferenceBudget) / 2
	}

	if basePrice < 150 {
		basePrice = 150
	}

	low := int64(float64(basePrice) * 0.85)
	high := int64(float64(basePrice) * 1.25)

	low = (low / 50) * 50
	high = (high / 50) * 50

	breakdown := map[string]int64{
		"Core Features & Logic": int64(float64(basePrice) * 0.50),
		"Frontend Integration":  int64(float64(basePrice) * 0.25),
		"Testing & CI/CD":       int64(float64(basePrice) * 0.15),
		"Project Management":    int64(float64(basePrice) * 0.10),
	}

	assumptions := []string{
		"The project has well-defined interfaces and clean design docs.",
		"Development will be conducted in a sandbox or staging environment.",
	}
	if len(req.Deliverables) > 0 {
		assumptions = append(assumptions, fmt.Sprintf("All %d listed deliverables are independent and testable.", len(req.Deliverables)))
	}
	if strings.Contains(tech, "go") {
		assumptions = append(assumptions, "The project relies on native Go modules and clean standard library conventions.")
	}

	risks := []string{
		"Scope creep due to changing or ambiguous deliverables.",
	}
	if strings.Contains(tech, "ai") || strings.Contains(tech, "llm") {
		risks = append(risks, "AI model non-determinism and API latency/rate limits.")
	}
	if strings.Contains(tech, "kubernetes") || strings.Contains(tech, "devops") {
		risks = append(risks, "Configuration drifts and target environment deployment discrepancies.")
	}

	rationale := fmt.Sprintf("Based on the tech stack (%s), the estimated effort is %s complexity. The price range represents core development, frontend binding, and automated testing.", req.TechStack, req.Complexity)

	resp := EvaluateProjectResponse{
		SuggestedLow:    low,
		SuggestedHigh:   high,
		ConfidenceLevel: 0.90,
		TaskBreakdown:   breakdown,
		Assumptions:     assumptions,
		Risks:           risks,
		Rationale:       rationale,
	}

	writeJSON(w, http.StatusOK, resp)
}

type CryptoWebhookRequest struct {
	UserID        string   `json:"userId"`
	Title         string   `json:"title"`
	ClientName    string   `json:"clientName"`
	CompanyName   string   `json:"companyName"`
	ClientEmail   string   `json:"clientEmail"`
	Phone         string   `json:"phone"`
	SiteType      string   `json:"siteType"`
	PackageTier   string   `json:"packageTier"`
	Timeline      string   `json:"timeline"`
	Brief         string   `json:"brief"`
	BudgetCents   int64    `json:"budgetCents"`
	AttachmentIDs []string `json:"attachmentIds"`
	SourceRepoURL string   `json:"sourceRepoURL"`
	TxHash        string   `json:"txHash"`
}

func (s *Server) cryptoWebhook(w http.ResponseWriter, r *http.Request) {
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "failed to read body")
		return
	}

	// 1. Signature validation (HMAC-SHA256)
	if s.cfg.CryptoWebhookSecret != "" {
		signatureHex := r.Header.Get("X-MergeOS-Signature")
		if signatureHex == "" {
			writeError(w, http.StatusUnauthorized, "missing signature header")
			return
		}
		expectedMac := hmac.New(sha256.New, []byte(s.cfg.CryptoWebhookSecret))
		expectedMac.Write(bodyBytes)
		expectedSignature := hex.EncodeToString(expectedMac.Sum(nil))
		if !hmac.Equal([]byte(signatureHex), []byte(expectedSignature)) {
			writeError(w, http.StatusUnauthorized, "invalid signature")
			return
		}
	}

	var req CryptoWebhookRequest
	if err := json.Unmarshal(bodyBytes, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON payload")
		return
	}

	// 2. Replay attack protection (unique transaction hash)
	if s.store.IsPaymentReferenceUsed(req.TxHash) {
		writeError(w, http.StatusConflict, "transaction hash has already been used")
		return
	}

	// 3. Assemble and execute Verify and CreateProject
	projectReq := CreateProjectRequest{
		Title:            req.Title,
		ClientName:       req.ClientName,
		CompanyName:      req.CompanyName,
		ClientEmail:      req.ClientEmail,
		Phone:            req.Phone,
		SiteType:         req.SiteType,
		PackageTier:      req.PackageTier,
		Timeline:         req.Timeline,
		Brief:            req.Brief,
		PaymentMethod:    PaymentCrypto,
		PaymentReference: req.TxHash,
		BudgetCents:      req.BudgetCents,
		AttachmentIDs:    req.AttachmentIDs,
		SourceRepoURL:    req.SourceRepoURL,
	}

	project, err := s.store.CreateProject(r.Context(), req.UserID, projectReq)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, project)
}
