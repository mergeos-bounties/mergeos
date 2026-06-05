package core

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestProductionOAuthDoesNotFallbackToMockLogin(t *testing.T) {
	server := NewServer(Config{Environment: "production", PrimaryDomain: defaultPrimaryDomain}, nil, nil)

	for _, tc := range []struct {
		name    string
		handler http.HandlerFunc
		path    string
		message string
	}{
		{
			name:    "google",
			handler: server.googleLogin,
			path:    "/api/auth/google/login",
			message: "Google OAuth is not configured",
		},
		{
			name:    "github",
			handler: server.githubBrowserLogin,
			path:    "/api/auth/github/login",
			message: "GitHub OAuth is not configured",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			resp := httptest.NewRecorder()
			tc.handler(resp, req)

			if resp.Code != http.StatusServiceUnavailable {
				t.Fatalf("status = %d, body = %s", resp.Code, resp.Body.String())
			}
			if location := resp.Header().Get("Location"); location != "" {
				t.Fatalf("unexpected redirect to %q", location)
			}
			if !strings.Contains(resp.Body.String(), tc.message) {
				t.Fatalf("response did not explain disabled OAuth: %s", resp.Body.String())
			}
		})
	}
}

func TestProductionOAuthRejectsMockCallbackCode(t *testing.T) {
	server := NewServer(Config{Environment: "production", PrimaryDomain: defaultPrimaryDomain}, nil, nil)

	for _, tc := range []struct {
		name       string
		handler    http.HandlerFunc
		path       string
		cookieName string
		message    string
	}{
		{
			name:       "google",
			handler:    server.googleCallback,
			path:       "/api/auth/google/callback?code=mock_google_code_123&state=state-1",
			cookieName: "google_oauth_state",
			message:    "Mock Google OAuth is disabled",
		},
		{
			name:       "github",
			handler:    server.githubCallback,
			path:       "/api/auth/github/callback?code=mock_github_code_123&state=state-1",
			cookieName: "github_oauth_state",
			message:    "Mock GitHub OAuth is disabled",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			req.AddCookie(&http.Cookie{Name: tc.cookieName, Value: "state-1"})
			resp := httptest.NewRecorder()
			tc.handler(resp, req)

			if resp.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, body = %s", resp.Code, resp.Body.String())
			}
			if !strings.Contains(resp.Body.String(), tc.message) {
				t.Fatalf("response did not reject mock code: %s", resp.Body.String())
			}
		})
	}
}

func TestLocalOAuthCanUseMockLoginFallback(t *testing.T) {
	server := NewServer(Config{Environment: "local", OAuthMockEnabled: true}, nil, nil)

	for _, tc := range []struct {
		name    string
		handler http.HandlerFunc
		path    string
		code    string
	}{
		{
			name:    "google",
			handler: server.googleLogin,
			path:    "/api/auth/google/login",
			code:    "mock_google_code_123",
		},
		{
			name:    "github",
			handler: server.githubBrowserLogin,
			path:    "/api/auth/github/login",
			code:    "mock_github_code_123",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			resp := httptest.NewRecorder()
			tc.handler(resp, req)

			if resp.Code != http.StatusTemporaryRedirect {
				t.Fatalf("status = %d, body = %s", resp.Code, resp.Body.String())
			}
			if location := resp.Header().Get("Location"); !strings.Contains(location, tc.code) {
				t.Fatalf("redirect location = %q, want mock code %q", location, tc.code)
			}
		})
	}
}
