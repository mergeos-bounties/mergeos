package core

import (
	"encoding/json"
	"net/http"
	"strings"
)

type Server struct {
	cfg      Config
	store    *Store
	payments *PaymentManager
}

func NewServer(cfg Config, store *Store, payments *PaymentManager) *Server {
	return &Server{cfg: cfg, store: store, payments: payments}
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", s.health)
	mux.HandleFunc("GET /api/config", s.config)
	mux.HandleFunc("GET /api/public/marketplace", s.marketplace)
	mux.HandleFunc("GET /api/public/ledger", s.publicLedger)
	mux.HandleFunc("POST /api/public/repo/issues", s.importRepoIssues)
	mux.HandleFunc("POST /api/auth/register", s.register)
	mux.HandleFunc("POST /api/auth/login", s.login)
	mux.HandleFunc("GET /api/auth/me", s.me)
	mux.HandleFunc("POST /api/auth/logout", s.logout)
	mux.HandleFunc("GET /api/auth/google/login", s.googleLogin)
	mux.HandleFunc("GET /api/auth/google/callback", s.googleCallback)
	mux.HandleFunc("GET /api/auth/github/login", s.githubLogin)
	mux.HandleFunc("GET /api/auth/github/callback", s.githubCallback)
	mux.HandleFunc("POST /api/payments/paypal/orders", s.createPayPalOrder)
	mux.HandleFunc("POST /api/uploads", s.uploadAttachment)
	mux.HandleFunc("GET /api/uploads/", s.downloadAttachment)
	mux.HandleFunc("GET /api/admin/summary", s.adminSummary)
	mux.HandleFunc("GET /api/admin/users", s.adminUsers)
	mux.HandleFunc("GET /api/admin/projects", s.adminProjects)
	mux.HandleFunc("GET /api/admin/tasks", s.adminTasks)
	mux.HandleFunc("GET /api/admin/notifications", s.adminNotifications)
	mux.HandleFunc("GET /api/admin/attachments", s.adminAttachments)
	mux.HandleFunc("GET /api/admin/ledger", s.adminLedger)
	mux.HandleFunc("GET /api/admin/ssl", s.adminSSLReviews)
	mux.HandleFunc("POST /api/admin/ssl/review", s.reviewAdminSSL)
	mux.HandleFunc("GET /api/projects", s.projects)
	mux.HandleFunc("POST /api/projects", s.createProject)
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
	if !s.store.CanAccessTask(user.ID, user.Role, taskID) {
		writeError(w, http.StatusForbidden, "admin access is required")
		return
	}

	var req AcceptTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	task, err := s.store.AcceptTask(taskID, req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, task)
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

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type,Authorization")
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
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
	if cfg.PayPalReady() || cfg.CryptoReady() {
		return "live-adapters"
	}
	if cfg.DevPaymentEnabled {
		return "local-dev-verifier"
	}
	return "not-configured"
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
