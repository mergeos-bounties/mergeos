package core

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

func FetchGitHubAuthProfile(ctx context.Context, cfg Config, req GitHubAuthRequest) (GitHubAuthProfile, error) {
	code := strings.TrimSpace(req.Code)
	if code == "" {
		return GitHubAuthProfile{}, errors.New("github code is required")
	}
	if strings.TrimSpace(req.RedirectURI) == "" {
		return GitHubAuthProfile{}, errors.New("github redirect_uri is required")
	}

	client := &http.Client{Timeout: 20 * time.Second}
	token, err := exchangeGitHubCode(ctx, client, cfg, code, strings.TrimSpace(req.RedirectURI))
	if err != nil {
		return GitHubAuthProfile{}, err
	}
	return fetchGitHubProfile(ctx, client, token)
}

func exchangeGitHubCode(ctx context.Context, client *http.Client, cfg Config, code, redirectURI string) (string, error) {
	body := map[string]string{
		"client_id":     cfg.GitHubOAuthClientID,
		"client_secret": cfg.GitHubOAuthClientSecret,
		"code":          code,
		"redirect_uri":  redirectURI,
	}
	var payload bytes.Buffer
	if err := json.NewEncoder(&payload).Encode(body); err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://github.com/login/oauth/access_token", &payload)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("github token exchange failed: %s", readBody(resp.Body))
	}

	var decoded struct {
		AccessToken      string `json:"access_token"`
		Error            string `json:"error"`
		ErrorDescription string `json:"error_description"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return "", err
	}
	if decoded.Error != "" {
		message := decoded.ErrorDescription
		if message == "" {
			message = decoded.Error
		}
		return "", errors.New(message)
	}
	if decoded.AccessToken == "" {
		return "", errors.New("github returned an empty access token")
	}
	return decoded.AccessToken, nil
}

func fetchGitHubProfile(ctx context.Context, client *http.Client, token string) (GitHubAuthProfile, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/user", nil)
	if err != nil {
		return GitHubAuthProfile{}, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := client.Do(req)
	if err != nil {
		return GitHubAuthProfile{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return GitHubAuthProfile{}, fmt.Errorf("github profile request failed: %s", readBody(resp.Body))
	}

	var profile struct {
		ID        int64  `json:"id"`
		Login     string `json:"login"`
		Name      string `json:"name"`
		Email     string `json:"email"`
		AvatarURL string `json:"avatar_url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&profile); err != nil {
		return GitHubAuthProfile{}, err
	}
	result := GitHubAuthProfile{
		ID:        fmt.Sprintf("%d", profile.ID),
		Username:  profile.Login,
		Name:      profile.Name,
		Email:     profile.Email,
		AvatarURL: profile.AvatarURL,
	}
	if result.Email == "" {
		result.Email = fetchPrimaryGitHubEmail(ctx, client, token)
	}
	return result, nil
}

func fetchPrimaryGitHubEmail(ctx context.Context, client *http.Client, token string) string {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/user/emails", nil)
	if err != nil {
		return ""
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return ""
	}

	var rows []struct {
		Email    string `json:"email"`
		Primary  bool   `json:"primary"`
		Verified bool   `json:"verified"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&rows); err != nil {
		return ""
	}
	for _, row := range rows {
		if row.Primary && row.Verified {
			return row.Email
		}
	}
	for _, row := range rows {
		if row.Verified {
			return row.Email
		}
	}
	return ""
}
