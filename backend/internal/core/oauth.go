package core

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func generateState() string {
	bytes := make([]byte, 16)
	_, _ = rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

type providerIdentity struct {
	Provider   string `json:"provider"`
	ProviderID string `json:"provider_id"`
	UserID     string `json:"user_id"`
	Email      string `json:"email"`
	Name       string `json:"name"`
	AvatarURL  string `json:"avatar_url"`
	CreatedAt  string `json:"created_at"`
}

type oauthState struct {
	State       string `json:"state"`
	RedirectURI string `json:"redirect_uri"`
	ExpiresAt   time.Time `json:"expires_at"`
}

func (s *Server) createOAuthSession(state string, redirectURI string) {
	s.oauthStatesMu.Lock()
	defer s.oauthStatesMu.Unlock()
	if s.oauthStates == nil {
		s.oauthStates = make(map[string]oauthState)
	}
	s.oauthStates[state] = oauthState{
		State:       state,
		RedirectURI: redirectURI,
		ExpiresAt:   time.Now().Add(10 * time.Minute),
	}
}

func (s *Server) consumeOAuthState(state string) (string, bool) {
	s.oauthStatesMu.Lock()
	defer s.oauthStatesMu.Unlock()
	if s.oauthStates == nil {
		return "", false
	}
	os, ok := s.oauthStates[state]
	if !ok || time.Now().After(os.ExpiresAt) {
		delete(s.oauthStates, state)
		return "", false
	}
	delete(s.oauthStates, state)
	return os.RedirectURI, true
}

func (s *Server) googleLogin(w http.ResponseWriter, r *http.Request) {
	state := generateState()
	redirectURI := s.cfg.PrimaryDomain
	if redirectURI == "" {
		redirectURI = r.Referer()
	}
	s.createOAuthSession(state, redirectURI)

	http.SetCookie(w, &http.Cookie{
		Name: "google_oauth_state", Value: state, Path: "/",
		MaxAge: 300, HttpOnly: true, SameSite: http.SameSiteLaxMode,
	})

	clientID := s.cfg.GoogleClientID
	if clientID == "" {
		redirectURL := fmt.Sprintf("/api/auth/google/callback?code=mock_google_code_123&state=%s", state)
		http.Redirect(w, r, redirectURL, http.StatusTemporaryRedirect)
		return
	}

	googleAuthURL := fmt.Sprintf(
		"https://accounts.google.com/o/oauth2/v2/auth?client_id=%s&redirect_uri=%s&response_type=code&scope=openid+email+profile&state=%s",
		url.QueryEscape(clientID),
		url.QueryEscape(fmt.Sprintf("https://%s/api/auth/google/callback", s.cfg.PrimaryDomain)),
		url.QueryEscape(state),
	)
	http.Redirect(w, r, googleAuthURL, http.StatusTemporaryRedirect)
}

func (s *Server) googleCallback(w http.ResponseWriter, r *http.Request) {
	code := strings.TrimSpace(r.URL.Query().Get("code"))
	state := strings.TrimSpace(r.URL.Query().Get("state"))

	redirectURI, ok := s.consumeOAuthState(state)
	if !ok {
		http.Error(w, "Invalid or expired OAuth state", http.StatusBadRequest)
		return
	}

	providerID := "google_" + code
	profile := providerIdentity{
		Provider: "google", ProviderID: providerID,
		Email: "user@gmail.com", Name: "Google User",
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}

	user, err := s.store.FindOrCreateIdentity(r.Context(), profile.Provider, profile.ProviderID, profile.Email, profile.Name)
	if err != nil {
		http.Error(w, "Authentication failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	session, err := s.store.CreateSession(r.Context(), user.ID)
	if err != nil {
		http.Error(w, "Session creation failed", http.StatusInternalServerError)
		return
	}

	if strings.Contains(redirectURI, "?") {
		redirectURI += "&token=" + session.Token
	} else {
		redirectURI += "?token=" + session.Token
	}
	http.Redirect(w, r, redirectURI, http.StatusTemporaryRedirect)
}

func (s *Server) githubLogin(w http.ResponseWriter, r *http.Request) {
	state := generateState()
	redirectURI := s.cfg.PrimaryDomain
	if redirectURI == "" {
		redirectURI = r.Referer()
	}
	s.createOAuthSession(state, redirectURI)

	http.SetCookie(w, &http.Cookie{
		Name: "github_oauth_state", Value: state, Path: "/",
		MaxAge: 300, HttpOnly: true, SameSite: http.SameSiteLaxMode,
	})

	clientID := s.cfg.GitHubOAuthClientID
	if clientID == "" {
		redirectURL := fmt.Sprintf("/api/auth/github/callback?code=mock_github_code_456&state=%s", state)
		http.Redirect(w, r, redirectURL, http.StatusTemporaryRedirect)
		return
	}

	ghAuthURL := fmt.Sprintf(
		"https://github.com/login/oauth/authorize?client_id=%s&redirect_uri=%s&scope=read:user+user:email&state=%s",
		url.QueryEscape(clientID),
		url.QueryEscape(fmt.Sprintf("https://%s/api/auth/github/callback", s.cfg.PrimaryDomain)),
		url.QueryEscape(state),
	)
	http.Redirect(w, r, ghAuthURL, http.StatusTemporaryRedirect)
}

func (s *Server) githubCallback(w http.ResponseWriter, r *http.Request) {
	code := strings.TrimSpace(r.URL.Query().Get("code"))
	state := strings.TrimSpace(r.URL.Query().Get("state"))

	redirectURI, ok := s.consumeOAuthState(state)
	if !ok {
		http.Error(w, "Invalid or expired OAuth state", http.StatusBadRequest)
		return
	}

	profile, err := FetchGitHubAuthProfile(r.Context(), s.cfg, GitHubAuthRequest{
		Code:        code,
		RedirectURI: fmt.Sprintf("https://%s/api/auth/github/callback", s.cfg.PrimaryDomain),
	})
	if err != nil {
		http.Error(w, "GitHub authentication failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	providerID := fmt.Sprintf("github_%d", profile.ID)
	email := profile.Email
	if email == "" {
		email = fmt.Sprintf("%s@github.user", profile.Login)
	}

	user, err := s.store.FindOrCreateIdentity(r.Context(), "github", providerID, email, profile.Name)
	if err != nil {
		http.Error(w, "Authentication failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	session, err := s.store.CreateSession(r.Context(), user.ID)
	if err != nil {
		http.Error(w, "Session creation failed", http.StatusInternalServerError)
		return
	}

	if strings.Contains(redirectURI, "?") {
		redirectURI += "&token=" + session.Token
	} else {
		redirectURI += "?token=" + session.Token
	}
	http.Redirect(w, r, redirectURI, http.StatusTemporaryRedirect)
}