package core

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestOAuthStateCookieUsesLaxAndSecureOnHTTPS(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "https://mergeos.local/api/auth/github/login", nil)
	req.Header.Set("X-Forwarded-Proto", "https")

	cookie := oauthStateCookie("github_oauth_state", "state-123", req)
	if cookie.SameSite != http.SameSiteLaxMode {
		t.Fatalf("SameSite = %v, want %v", cookie.SameSite, http.SameSiteLaxMode)
	}
	if !cookie.Secure {
		t.Fatal("expected secure cookie for https requests")
	}
}

func TestGitHubBrowserLoginSetsHardenedStateCookie(t *testing.T) {
	srv := &Server{cfg: Config{}}
	req := httptest.NewRequest(http.MethodGet, "https://mergeos.local/api/auth/github/login", nil)
	req.Header.Set("X-Forwarded-Proto", "https")
	rr := httptest.NewRecorder()

	srv.githubBrowserLogin(rr, req)

	if rr.Code != http.StatusTemporaryRedirect {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusTemporaryRedirect)
	}
	cookies := rr.Result().Cookies()
	var stateCookie *http.Cookie
	for i := range cookies {
		if cookies[i].Name == "github_oauth_state" {
			stateCookie = cookies[i]
			break
		}
	}
	if stateCookie == nil {
		t.Fatal("missing github_oauth_state cookie")
	}
	if stateCookie.SameSite != http.SameSiteLaxMode {
		t.Fatalf("SameSite = %v, want %v", stateCookie.SameSite, http.SameSiteLaxMode)
	}
	if !stateCookie.Secure {
		t.Fatal("expected secure state cookie when forwarded proto is https")
	}
	if !strings.Contains(rr.Header().Get("Location"), "https://github.com/login/oauth/authorize") {
		t.Fatalf("redirect location = %q", rr.Header().Get("Location"))
	}
}

func TestGoogleLoginSetsHardenedStateCookie(t *testing.T) {
	srv := &Server{cfg: Config{}}
	req := httptest.NewRequest(http.MethodGet, "https://mergeos.local/api/auth/google/login", nil)
	req.Header.Set("X-Forwarded-Proto", "https")
	rr := httptest.NewRecorder()

	srv.googleLogin(rr, req)

	if rr.Code != http.StatusTemporaryRedirect {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusTemporaryRedirect)
	}
	cookies := rr.Result().Cookies()
	var stateCookie *http.Cookie
	for i := range cookies {
		if cookies[i].Name == "google_oauth_state" {
			stateCookie = cookies[i]
			break
		}
	}
	if stateCookie == nil {
		t.Fatal("missing google_oauth_state cookie")
	}
	if stateCookie.SameSite != http.SameSiteLaxMode {
		t.Fatalf("SameSite = %v, want %v", stateCookie.SameSite, http.SameSiteLaxMode)
	}
	if !stateCookie.Secure {
		t.Fatal("expected secure state cookie when forwarded proto is https")
	}
	if !strings.Contains(rr.Header().Get("Location"), "/api/auth/google/callback") {
		t.Fatalf("redirect location = %q", rr.Header().Get("Location"))
	}
}
