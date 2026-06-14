package core

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
)

// Helper to generate random state
func generateState() string {
	bytes := make([]byte, 16)
	_, _ = rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// Redirects to provider
func (s *Server) googleLogin(w http.ResponseWriter, r *http.Request) {
	state := generateState()
	// Set HttpOnly state cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "google_oauth_state",
		Value:    state,
		Path:     "/",
		MaxAge:   300, // 5 mins
		HttpOnly: true,
	})

	clientID := s.cfg.GoogleClientID
	if clientID == "" {
		if s.cfg.OAuthMockReady() {
			redirectURL := fmt.Sprintf("/api/auth/google/callback?code=mock_google_code_123&state=%s", state)
			http.Redirect(w, r, redirectURL, http.StatusTemporaryRedirect)
			return
		}
		writeError(w, http.StatusServiceUnavailable, "Google OAuth is not configured")
		return
	}

	redirectURI := s.getFrontRedirectBase(r) + "/api/auth/google/callback"
	authURL := fmt.Sprintf("https://accounts.google.com/o/oauth2/v2/auth?client_id=%s&redirect_uri=%s&response_type=code&scope=openid%%20profile%%20email&state=%s",
		url.QueryEscape(clientID),
		url.QueryEscape(redirectURI),
		url.QueryEscape(state),
	)
	http.Redirect(w, r, authURL, http.StatusTemporaryRedirect)
}

func (s *Server) googleCallback(w http.ResponseWriter, r *http.Request) {
	// Validate state
	stateCookie, err := r.Cookie("google_oauth_state")
	if err != nil || stateCookie.Value != r.URL.Query().Get("state") {
		writeError(w, http.StatusForbidden, "CSRF validation failed: state mismatch")
		return
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		writeError(w, http.StatusBadRequest, "OAuth code missing")
		return
	}

	var email, name string

	if code == "mock_google_code_123" {
		if !s.cfg.OAuthMockReady() {
			writeError(w, http.StatusBadRequest, "Mock Google OAuth is disabled")
			return
		}
		email = "mock.google.user@gmail.com"
		name = "Mock Google User"
	} else {
		if !s.cfg.GoogleOAuthReady() {
			writeError(w, http.StatusServiceUnavailable, "Google OAuth is not configured")
			return
		}
		// Real Token Exchange
		redirectURI := s.getFrontRedirectBase(r) + "/api/auth/google/callback"
		tokenURL := "https://oauth2.googleapis.com/token"

		data := url.Values{}
		data.Set("code", code)
		data.Set("client_id", s.cfg.GoogleClientID)
		data.Set("client_secret", s.cfg.GoogleClientSecret)
		data.Set("redirect_uri", redirectURI)
		data.Set("grant_type", "authorization_code")

		resp, err := http.PostForm(tokenURL, data)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to connect to Google token endpoint")
			return
		}
		defer resp.Body.Close()

		var tokenResp struct {
			AccessToken string `json:"access_token"`
			Error       string `json:"error"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to parse Google token response")
			return
		}

		if tokenResp.AccessToken == "" {
			writeError(w, http.StatusBadRequest, "Google token exchange failed: "+tokenResp.Error)
			return
		}

		// Fetch User Info
		req, _ := http.NewRequest("GET", "https://www.googleapis.com/oauth2/v3/userinfo", nil)
		req.Header.Set("Authorization", "Bearer "+tokenResp.AccessToken)
		userInfoResp, err := http.DefaultClient.Do(req)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to fetch Google user info")
			return
		}
		defer userInfoResp.Body.Close()

		var userInfo struct {
			Email string `json:"email"`
			Name  string `json:"name"`
		}
		if err := json.NewDecoder(userInfoResp.Body).Decode(&userInfo); err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to parse Google user info")
			return
		}

		email = userInfo.Email
		name = userInfo.Name
	}

	if email == "" {
		writeError(w, http.StatusBadRequest, "Could not retrieve email from Google")
		return
	}

	// Login/Register
	auth, err := s.store.LoginOrRegisterOAuth(email, name, "Google")
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Redirect back to frontend with session token
	frontendRedirectURL := fmt.Sprintf("%s/?token=%s", s.getFrontRedirectBase(r), auth.Token)
	http.Redirect(w, r, frontendRedirectURL, http.StatusTemporaryRedirect)
}

func (s *Server) githubBrowserLogin(w http.ResponseWriter, r *http.Request) {
	state := generateState()
	http.SetCookie(w, &http.Cookie{
		Name:     "github_oauth_state",
		Value:    state,
		Path:     "/",
		MaxAge:   300,
		HttpOnly: true,
	})

	clientID := s.cfg.GitHubOAuthClientID
	if clientID == "" {
		if s.cfg.OAuthMockReady() {
			redirectURL := fmt.Sprintf("/api/auth/github/callback?code=mock_github_code_123&state=%s", state)
			http.Redirect(w, r, redirectURL, http.StatusTemporaryRedirect)
			return
		}
		writeError(w, http.StatusServiceUnavailable, "GitHub OAuth is not configured")
		return
	}

	redirectURI := s.getFrontRedirectBase(r) + "/api/auth/github/callback"
	authURL := fmt.Sprintf("https://github.com/login/oauth/authorize?client_id=%s&redirect_uri=%s&state=%s",
		url.QueryEscape(clientID),
		url.QueryEscape(redirectURI),
		url.QueryEscape(state),
	)
	http.Redirect(w, r, authURL, http.StatusTemporaryRedirect)
}

func (s *Server) githubCallback(w http.ResponseWriter, r *http.Request) {
	stateCookie, err := r.Cookie("github_oauth_state")
	if err != nil || stateCookie.Value != r.URL.Query().Get("state") {
		writeError(w, http.StatusForbidden, "CSRF validation failed: state mismatch")
		return
	}

	code := r.URL.Query().Get("code")
	if code == "" {
		writeError(w, http.StatusBadRequest, "OAuth code missing")
		return
	}

	var email, name string

	if code == "mock_github_code_123" {
		if !s.cfg.OAuthMockReady() {
			writeError(w, http.StatusBadRequest, "Mock GitHub OAuth is disabled")
			return
		}
		email = "mock.github.user@gmail.com"
		name = "Mock GitHub User"
	} else {
		if !s.cfg.GitHubOAuthReady() {
			writeError(w, http.StatusServiceUnavailable, "GitHub OAuth is not configured")
			return
		}
		// Real Token Exchange
		redirectURI := s.getFrontRedirectBase(r) + "/api/auth/github/callback"
		tokenURL := "https://github.com/login/oauth/access_token"

		data := url.Values{}
		data.Set("code", code)
		data.Set("client_id", s.cfg.GitHubOAuthClientID)
		data.Set("client_secret", s.cfg.GitHubOAuthClientSecret)
		data.Set("redirect_uri", redirectURI)

		req, _ := http.NewRequest("POST", tokenURL, strings.NewReader(data.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Accept", "application/json")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to connect to GitHub token endpoint")
			return
		}
		defer resp.Body.Close()

		var tokenResp struct {
			AccessToken string `json:"access_token"`
			Error       string `json:"error"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to parse GitHub token response")
			return
		}

		if tokenResp.AccessToken == "" {
			writeError(w, http.StatusBadRequest, "GitHub token exchange failed: "+tokenResp.Error)
			return
		}

		// Fetch User Info
		reqUser, _ := http.NewRequest("GET", "https://api.github.com/user", nil)
		reqUser.Header.Set("Authorization", "Bearer "+tokenResp.AccessToken)
		reqUser.Header.Set("User-Agent", "mergeos-api")

		userResp, err := http.DefaultClient.Do(reqUser)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to fetch GitHub user profile")
			return
		}
		defer userResp.Body.Close()

		var userInfo struct {
			Login string `json:"login"`
			Name  string `json:"name"`
			Email string `json:"email"`
		}
		if err := json.NewDecoder(userResp.Body).Decode(&userInfo); err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to parse GitHub user profile")
			return
		}

		name = userInfo.Name
		if name == "" {
			name = userInfo.Login
		}
		email = userInfo.Email

		// If email is empty/private, fetch all user emails from GitHub
		if email == "" {
			reqEmails, _ := http.NewRequest("GET", "https://api.github.com/user/emails", nil)
			reqEmails.Header.Set("Authorization", "Bearer "+tokenResp.AccessToken)
			reqEmails.Header.Set("User-Agent", "mergeos-api")

			emailsResp, err := http.DefaultClient.Do(reqEmails)
			if err == nil {
				defer emailsResp.Body.Close()
				var emails []struct {
					Email   string `json:"email"`
					Primary bool   `json:"primary"`
				}
				if json.NewDecoder(emailsResp.Body).Decode(&emails) == nil {
					for _, em := range emails {
						if em.Primary {
							email = em.Email
							break
						}
					}
				}
			}
		}
	}

	if email == "" {
		writeError(w, http.StatusBadRequest, "Could not retrieve email from GitHub")
		return
	}

	// Login/Register
	auth, err := s.store.LoginOrRegisterOAuth(email, name, "GitHub")
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Redirect back to frontend with session token
	frontendRedirectURL := fmt.Sprintf("%s/?token=%s", s.getFrontRedirectBase(r), auth.Token)
	http.Redirect(w, r, frontendRedirectURL, http.StatusTemporaryRedirect)
}

func (s *Server) getFrontRedirectBase(r *http.Request) string {
	if s.cfg.Environment == "local" {
		if forwardedHost := firstForwardedHeader(r.Header.Get("X-Forwarded-Host")); forwardedHost != "" {
			scheme := firstForwardedHeader(r.Header.Get("X-Forwarded-Proto"))
			if scheme != "https" {
				scheme = "http"
			}
			return scheme + "://" + forwardedHost
		}
		return "http://127.0.0.1:5173"
	}
	scheme := "https"
	if isLoopbackRedirectHost(r.Host) {
		scheme = "http"
	}
	return scheme + "://" + s.cfg.PrimaryDomain
}

func isLoopbackRedirectHost(host string) bool {
	host = strings.TrimSpace(host)
	if host == "" {
		return false
	}
	if parsedHost, _, err := net.SplitHostPort(host); err == nil {
		host = parsedHost
	}
	host = strings.Trim(host, "[]")
	return host == "localhost" || host == "127.0.0.1" || host == "::1"
}

func firstForwardedHeader(value string) string {
	head, _, _ := strings.Cut(value, ",")
	return strings.TrimSpace(head)
}
