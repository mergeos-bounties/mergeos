package core

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
)

type Server struct {
	cfg           Config
	store         *Store
	payments      *PaymentManager
	oauthStates   map[string]oauthState
	oauthStatesMu sync.Mutex
}

func NewServer(cfg Config, store *Store, payments *PaymentManager) *Server {
	return &Server{cfg: cfg, store: store, payments: payments, oauthStates: make(map[string]oauthState)}
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
	mux.HandleFunc("GET /api/auth/google/login", s.googleLogin)
	mux.HandleFunc("GET /api/auth/google/callback", s.googleCallback)
	mux.HandleFunc("POST /api/auth/github", s.githubAuthLogin)
	mux.HandleFunc("GET /api/auth/github/login", s.githubLogin)
	mux.HandleFunc("GET /api/auth/github/callback", s.githubCallback)
	mux.HandleFunc("GET /api/auth/me", s.authRequired(s.me))
	mux.HandleFunc("POST /api/auth/logout", s.authRequired(s.logout))
	mux.HandleFunc("POST /api/projects/evaluate-price", s.estimate)
	mux.HandleFunc("POST /api/projects", s.authRequired(s.createProject))
	mux.HandleFunc("GET /api/projects", s.authRequired(s.listMyProjects))
	mux.HandleFunc("GET /api/tasks", s.authRequired(s.listMyTasks))
	mux.HandleFunc("GET /api/ledger", s.authRequired(s.myLedger))
	return mux
}