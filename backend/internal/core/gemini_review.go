package core

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"
)

const geminiReviewMarker = "<!-- mergeos-gemini-pr-review -->"
const geminiAPIKeyTestPrompt = "Reply with exactly this short sentence: MergeOS Gemini key test ok."

type GeminiReviewService struct {
	cfg    Config
	store  *Store
	client *http.Client
}

func NewGeminiReviewService(cfg Config, store *Store) *GeminiReviewService {
	return &GeminiReviewService{
		cfg:   cfg,
		store: store,
		client: &http.Client{
			Timeout: 90 * time.Second,
		},
	}
}

type GeminiReviewWebhookRequest struct {
	EventName   string `json:"event_name"`
	Action      string `json:"action"`
	Repository  string `json:"repository"`
	PullNumber  int    `json:"pull_number"`
	DeliveryID  string `json:"delivery_id"`
	Sender      string `json:"sender"`
	PullRequest struct {
		Number  int    `json:"number"`
		Title   string `json:"title"`
		Body    string `json:"body"`
		HTMLURL string `json:"html_url"`
		Author  string `json:"author"`
		BaseRef string `json:"base_ref"`
		HeadRef string `json:"head_ref"`
		HeadSHA string `json:"head_sha"`
		Draft   bool   `json:"draft"`
	} `json:"pull_request"`
}

type GeminiReviewWebhookResponse struct {
	OK               bool     `json:"ok"`
	Repository       string   `json:"repository"`
	PullNumber       int      `json:"pull_number"`
	CommentURL       string   `json:"comment_url,omitempty"`
	Labels           []string `json:"labels,omitempty"`
	EvidenceProvided bool     `json:"evidence_provided"`
	StarVerified     bool     `json:"star_verified"`
	Model            string   `json:"model"`
	KeyID            string   `json:"key_id,omitempty"`
}

type geminiReviewPullRequest struct {
	Number  int    `json:"number"`
	Title   string `json:"title"`
	Body    string `json:"body"`
	HTMLURL string `json:"html_url"`
	Draft   bool   `json:"draft"`
	User    struct {
		Login string `json:"login"`
	} `json:"user"`
	Base struct {
		Ref string `json:"ref"`
		SHA string `json:"sha"`
	} `json:"base"`
	Head struct {
		Ref string `json:"ref"`
		SHA string `json:"sha"`
	} `json:"head"`
}

type geminiReviewFile struct {
	Filename  string `json:"filename"`
	Status    string `json:"status"`
	Additions int    `json:"additions"`
	Deletions int    `json:"deletions"`
	Patch     string `json:"patch"`
}

type geminiReviewComment struct {
	ID      int64  `json:"id"`
	Body    string `json:"body"`
	HTMLURL string `json:"html_url"`
	User    struct {
		Login string `json:"login"`
		Type  string `json:"type"`
	} `json:"user"`
}

type geminiReviewIssue struct {
	Number  int    `json:"number"`
	Title   string `json:"title"`
	Body    string `json:"body"`
	HTMLURL string `json:"html_url"`
	Labels  []struct {
		Name string `json:"name"`
	} `json:"labels"`
}

func (s *Server) geminiReviewWebhook(w http.ResponseWriter, r *http.Request) {
	started := time.Now().UTC()
	eventName := strings.TrimSpace(r.Header.Get("X-GitHub-Event"))
	deliveryID := strings.TrimSpace(r.Header.Get("X-GitHub-Delivery"))
	logWebhook := func(status string, statusCode int, message string, req *GeminiReviewWebhookRequest, result *GeminiReviewWebhookResponse) {
		if s.store == nil {
			return
		}
		completed := time.Now().UTC()
		entry := GeminiWebhookLog{
			DeliveryID:     deliveryID,
			EventName:      eventName,
			Status:         status,
			StatusCode:     statusCode,
			Error:          message,
			DurationMillis: completed.Sub(started).Milliseconds(),
			ReceivedAt:     started,
			CompletedAt:    &completed,
		}
		if req != nil {
			entry.EventName = firstNonEmpty(req.EventName, entry.EventName)
			entry.Action = req.Action
			entry.Repository = req.Repository
			entry.PullNumber = req.PullNumber
			entry.Sender = req.Sender
			entry.DeliveryID = firstNonEmpty(req.DeliveryID, entry.DeliveryID)
		}
		if result != nil {
			entry.Repository = firstNonEmpty(result.Repository, entry.Repository)
			if result.PullNumber > 0 {
				entry.PullNumber = result.PullNumber
			}
			entry.CommentURL = result.CommentURL
			entry.KeyID = result.KeyID
			entry.Labels = append([]string(nil), result.Labels...)
		}
		_ = s.store.AddGeminiWebhookLog(entry)
	}

	if !s.cfg.GeminiReviewReady() || s.geminiReviewer == nil || !s.geminiReviewer.Ready() {
		logWebhook("service_unavailable", http.StatusServiceUnavailable, "Gemini reviewer is not configured", nil, nil)
		writeError(w, http.StatusServiceUnavailable, "Gemini reviewer is not configured")
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		logWebhook("bad_request", http.StatusBadRequest, "could not read request body", nil, nil)
		writeError(w, http.StatusBadRequest, "could not read request body")
		return
	}
	signature := r.Header.Get("X-Hub-Signature-256")
	if strings.TrimSpace(signature) == "" {
		signature = r.Header.Get("X-MergeOS-Signature")
	}
	if !verifyMergeOSSignature(s.cfg.GeminiReviewWebhookSecret, signature, body) {
		logWebhook("unauthorized", http.StatusUnauthorized, "invalid review webhook signature", nil, nil)
		writeError(w, http.StatusUnauthorized, "invalid review webhook signature")
		return
	}
	req, ok, err := geminiReviewRequestFromGitHubWebhook(eventName, body)
	if err != nil {
		logWebhook("bad_request", http.StatusBadRequest, "invalid JSON body", nil, nil)
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if !ok {
		logWebhook("skipped", http.StatusAccepted, "", nil, nil)
		writeJSON(w, http.StatusAccepted, map[string]any{"ok": true, "skipped": true})
		return
	}
	req.DeliveryID = deliveryID
	result, err := s.geminiReviewer.ReviewPullRequest(r.Context(), req)
	if err != nil {
		logWebhook("failed", http.StatusBadGateway, err.Error(), &req, nil)
		writeError(w, http.StatusBadGateway, err.Error())
		return
	}
	logWebhook("processed", http.StatusOK, "", &req, &result)
	writeJSON(w, http.StatusOK, result)
}

func (s *GeminiReviewService) Ready() bool {
	return s.store != nil && s.store.HasRunnableGeminiAPIKey()
}

func geminiReviewRequestFromGitHubWebhook(eventName string, body []byte) (GeminiReviewWebhookRequest, bool, error) {
	eventName = strings.TrimSpace(eventName)
	switch eventName {
	case "pull_request":
		var payload struct {
			Action     string `json:"action"`
			Number     int    `json:"number"`
			Repository struct {
				FullName string `json:"full_name"`
			} `json:"repository"`
			Sender struct {
				Login string `json:"login"`
			} `json:"sender"`
			PullRequest struct {
				Number  int    `json:"number"`
				Title   string `json:"title"`
				Body    string `json:"body"`
				HTMLURL string `json:"html_url"`
				Draft   bool   `json:"draft"`
				User    struct {
					Login string `json:"login"`
				} `json:"user"`
				Base struct {
					Ref string `json:"ref"`
				} `json:"base"`
				Head struct {
					Ref string `json:"ref"`
					SHA string `json:"sha"`
				} `json:"head"`
			} `json:"pull_request"`
		}
		if err := json.Unmarshal(body, &payload); err != nil {
			return GeminiReviewWebhookRequest{}, false, err
		}
		if !supportedGeminiPullRequestAction(payload.Action) {
			return GeminiReviewWebhookRequest{}, false, nil
		}
		req := GeminiReviewWebhookRequest{
			EventName:  eventName,
			Action:     payload.Action,
			Repository: payload.Repository.FullName,
			PullNumber: payload.Number,
			Sender:     payload.Sender.Login,
		}
		req.PullRequest.Number = payload.PullRequest.Number
		req.PullRequest.Title = payload.PullRequest.Title
		req.PullRequest.Body = payload.PullRequest.Body
		req.PullRequest.HTMLURL = payload.PullRequest.HTMLURL
		req.PullRequest.Author = payload.PullRequest.User.Login
		req.PullRequest.BaseRef = payload.PullRequest.Base.Ref
		req.PullRequest.HeadRef = payload.PullRequest.Head.Ref
		req.PullRequest.HeadSHA = payload.PullRequest.Head.SHA
		req.PullRequest.Draft = payload.PullRequest.Draft
		return req, true, nil
	case "issue_comment":
		var payload struct {
			Action     string `json:"action"`
			Repository struct {
				FullName string `json:"full_name"`
			} `json:"repository"`
			Sender struct {
				Login string `json:"login"`
			} `json:"sender"`
			Comment struct {
				Body string `json:"body"`
				User struct {
					Login string `json:"login"`
					Type  string `json:"type"`
				} `json:"user"`
			} `json:"comment"`
			Issue struct {
				Number      int         `json:"number"`
				Title       string      `json:"title"`
				Body        string      `json:"body"`
				HTMLURL     string      `json:"html_url"`
				PullRequest interface{} `json:"pull_request"`
				User        struct {
					Login string `json:"login"`
				} `json:"user"`
			} `json:"issue"`
		}
		if err := json.Unmarshal(body, &payload); err != nil {
			return GeminiReviewWebhookRequest{}, false, err
		}
		if payload.Issue.PullRequest == nil || (payload.Action != "created" && payload.Action != "edited") {
			return GeminiReviewWebhookRequest{}, false, nil
		}
		if strings.Contains(payload.Comment.Body, geminiReviewMarker) {
			return GeminiReviewWebhookRequest{}, false, nil
		}
		req := GeminiReviewWebhookRequest{
			EventName:  eventName,
			Action:     payload.Action,
			Repository: payload.Repository.FullName,
			PullNumber: payload.Issue.Number,
			Sender:     payload.Sender.Login,
		}
		req.PullRequest.Number = payload.Issue.Number
		req.PullRequest.Title = payload.Issue.Title
		req.PullRequest.Body = payload.Issue.Body
		req.PullRequest.HTMLURL = payload.Issue.HTMLURL
		req.PullRequest.Author = payload.Issue.User.Login
		return req, true, nil
	default:
		return GeminiReviewWebhookRequest{}, false, nil
	}
}

func supportedGeminiPullRequestAction(action string) bool {
	switch strings.TrimSpace(action) {
	case "opened", "edited", "reopened", "synchronize", "ready_for_review":
		return true
	default:
		return false
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func verifyMergeOSSignature(secret, signature string, body []byte) bool {
	secret = strings.TrimSpace(secret)
	signature = strings.TrimSpace(signature)
	if secret == "" || !strings.HasPrefix(signature, "sha256=") {
		return false
	}
	expectedMAC := hmac.New(sha256.New, []byte(secret))
	expectedMAC.Write(body)
	expected := "sha256=" + hex.EncodeToString(expectedMAC.Sum(nil))
	return subtle.ConstantTimeCompare([]byte(expected), []byte(signature)) == 1
}

func (s *GeminiReviewService) ReviewPullRequest(ctx context.Context, req GeminiReviewWebhookRequest) (GeminiReviewWebhookResponse, error) {
	target, err := githubIssueTargetFromRepository(req.Repository, req.PullNumber)
	if err != nil {
		return GeminiReviewWebhookResponse{}, err
	}
	if req.PullNumber <= 0 {
		return GeminiReviewWebhookResponse{}, errors.New("pull_number is required")
	}
	gh, err := newAdminGitHubClient(s.cfg, true)
	if err != nil {
		return GeminiReviewWebhookResponse{}, err
	}
	pr, err := gh.reviewPullRequest(ctx, target, req.PullNumber)
	if err != nil {
		return GeminiReviewWebhookResponse{}, err
	}
	files, err := gh.reviewPullRequestFiles(ctx, target, req.PullNumber)
	if err != nil {
		return GeminiReviewWebhookResponse{}, err
	}
	comments, err := gh.reviewIssueComments(ctx, target, req.PullNumber)
	if err != nil {
		return GeminiReviewWebhookResponse{}, err
	}
	linkedIssues, err := gh.linkedReviewIssues(ctx, target, pr)
	if err != nil {
		return GeminiReviewWebhookResponse{}, err
	}
	starVerified, err := gh.reviewAuthorStarred(ctx, target, pr.User.Login)
	if err != nil {
		return GeminiReviewWebhookResponse{}, err
	}
	evidenceProvided := reviewEvidenceProvided(pr, comments)

	prompt := buildGeminiReviewPrompt(pr, files, comments, linkedIssues, starVerified, evidenceProvided, s.cfg.GeminiReviewMaxPatchBytes)
	review, keyID, model, err := s.generate(ctx, prompt)
	if err != nil {
		return GeminiReviewWebhookResponse{}, err
	}
	commentBody := renderGeminiReviewComment(review, starVerified, evidenceProvided)
	commentURL, err := gh.upsertReviewComment(ctx, target, req.PullNumber, geminiReviewMarker, commentBody)
	if err != nil {
		return GeminiReviewWebhookResponse{}, err
	}
	labels, err := gh.applyGeminiReadinessLabels(ctx, target, req.PullNumber, starVerified, evidenceProvided)
	if err != nil {
		return GeminiReviewWebhookResponse{}, err
	}
	return GeminiReviewWebhookResponse{
		OK:               true,
		Repository:       target.fullName(),
		PullNumber:       req.PullNumber,
		CommentURL:       commentURL,
		Labels:           labels,
		EvidenceProvided: evidenceProvided,
		StarVerified:     starVerified,
		Model:            model,
		KeyID:            keyID,
	}, nil
}

func (s *GeminiReviewService) generate(ctx context.Context, prompt string) (string, string, string, error) {
	if s.store == nil {
		return "", "", "", errors.New("LLM key store is required")
	}
	provider, model := s.store.LLMReviewProviderModel()
	candidates := s.store.GeminiAPIKeyCandidatesForProvider(provider)
	if len(candidates) == 0 {
		return "", "", "", fmt.Errorf("no active %s API keys are configured", provider)
	}
	var lastErr error
	for _, candidate := range candidates {
		if err := s.store.MarkGeminiAPIKeyAttempt(candidate.ID); err != nil {
			return "", "", "", err
		}
		text, err := s.generateWithKeyAndProvider(ctx, provider, candidate.KeyValue, model, prompt, 2200)
		if err == nil {
			if markErr := s.store.MarkGeminiAPIKeySuccess(candidate.ID, http.StatusOK); markErr != nil {
				return "", "", "", markErr
			}
			return text, candidate.ID, model, nil
		}
		lastErr = err
		statusCode := geminiErrorStatusCode(err)
		if isGeminiQuotaError(err) {
			_ = s.store.MarkGeminiAPIKeyQuotaLimited(candidate.ID, statusCode, err.Error())
			continue
		}
		if isGeminiKeySpecificError(err) {
			_ = s.store.MarkGeminiAPIKeyError(candidate.ID, statusCode, err.Error())
			continue
		}
		_ = s.store.MarkGeminiAPIKeyError(candidate.ID, statusCode, err.Error())
		return "", "", "", err
	}
	if lastErr == nil {
		lastErr = errors.New("LLM review failed")
	}
	return "", "", "", lastErr
}

func (s *GeminiReviewService) generateWithKey(ctx context.Context, key, prompt string) (string, error) {
	model := strings.Trim(strings.TrimSpace(s.store.GeminiReviewModel()), "/")
	if model == "" {
		model = defaultGeminiReviewModel
	}
	return s.generateWithKeyAndModel(ctx, key, model, prompt, 2200)
}

func (s *GeminiReviewService) generateWithKeyAndProvider(ctx context.Context, provider, key, model, prompt string, maxOutputTokens int) (string, error) {
	provider = normalizedLLMProviderOrDefault(provider)
	model = normalizedLLMModelOrDefault(provider, model)
	switch provider {
	case "gemini":
		return s.generateWithKeyAndModel(ctx, key, model, prompt, maxOutputTokens)
	case "openai":
		return s.generateOpenAICompatible(ctx, "https://api.openai.com/v1/chat/completions", key, model, prompt, maxOutputTokens, nil)
	case "groq":
		return s.generateOpenAICompatible(ctx, "https://api.groq.com/openai/v1/chat/completions", key, model, prompt, maxOutputTokens, nil)
	case "openrouter":
		return s.generateOpenAICompatible(ctx, "https://openrouter.ai/api/v1/chat/completions", key, model, prompt, maxOutputTokens, map[string]string{
			"HTTP-Referer": "https://mergeos.shop",
			"X-Title":      "MergeOS",
		})
	case "deepseek":
		return s.generateOpenAICompatible(ctx, "https://api.deepseek.com/chat/completions", key, model, prompt, maxOutputTokens, nil)
	case "mistral":
		return s.generateOpenAICompatible(ctx, "https://api.mistral.ai/v1/chat/completions", key, model, prompt, maxOutputTokens, nil)
	case "anthropic":
		return s.generateAnthropic(ctx, key, model, prompt, maxOutputTokens)
	default:
		return "", fmt.Errorf("unsupported LLM provider %q", provider)
	}
}

func (s *GeminiReviewService) TestAPIKey(ctx context.Context, keyID, provider, model string) (TestGeminiAPIKeyResponse, error) {
	if s.store == nil {
		return TestGeminiAPIKeyResponse{}, errors.New("LLM key store is required")
	}
	candidate, err := s.store.GeminiAPIKeyCandidateByID(keyID)
	if err != nil {
		return TestGeminiAPIKeyResponse{}, err
	}
	provider = strings.TrimSpace(provider)
	if provider == "" {
		provider = candidate.Provider
	}
	normalizedProvider, err := normalizeLLMProvider(provider)
	if err != nil {
		return TestGeminiAPIKeyResponse{}, err
	}
	if normalizedProvider != candidate.Provider {
		return TestGeminiAPIKeyResponse{}, errors.New("selected provider does not match this API key")
	}
	model = strings.TrimSpace(model)
	if model == "" {
		model = candidate.Model
	}
	normalizedModel, err := normalizeLLMModel(normalizedProvider, model)
	if err != nil {
		return TestGeminiAPIKeyResponse{}, err
	}

	startedAt := time.Now()
	_, err = s.generateWithKeyAndProvider(ctx, normalizedProvider, candidate.KeyValue, normalizedModel, geminiAPIKeyTestPrompt, 48)
	duration := time.Since(startedAt).Milliseconds()
	if err == nil {
		key, recordErr := s.store.RecordGeminiAPIKeyTestResult(candidate.ID, GeminiAPIKeyStatusActive, http.StatusOK, "")
		if recordErr != nil {
			return TestGeminiAPIKeyResponse{}, recordErr
		}
		message := "Key test passed"
		if key.Status == GeminiAPIKeyStatusDisabled {
			message = "Key test passed, but the key is still disabled"
		}
		return TestGeminiAPIKeyResponse{
			OK:             true,
			Provider:       normalizedProvider,
			Model:          normalizedModel,
			Key:            key,
			StatusCode:     http.StatusOK,
			DurationMillis: duration,
			Message:        message,
		}, nil
	}

	statusCode := geminiErrorStatusCode(err)
	status := GeminiAPIKeyStatusError
	if isGeminiQuotaError(err) {
		status = GeminiAPIKeyStatusQuotaLimited
	}
	key, recordErr := s.store.RecordGeminiAPIKeyTestResult(candidate.ID, status, statusCode, err.Error())
	if recordErr != nil {
		return TestGeminiAPIKeyResponse{}, recordErr
	}
	return TestGeminiAPIKeyResponse{
		OK:             false,
		Provider:       normalizedProvider,
		Model:          normalizedModel,
		Key:            key,
		StatusCode:     statusCode,
		DurationMillis: duration,
		Error:          truncateGeminiKeyError(err.Error()),
	}, nil
}

func (s *GeminiReviewService) generateWithKeyAndModel(ctx context.Context, key, model, prompt string, maxOutputTokens int) (string, error) {
	model = strings.Trim(strings.TrimSpace(model), "/")
	if model == "" {
		model = defaultGeminiReviewModel
	}
	model = strings.TrimPrefix(model, "models/")
	if maxOutputTokens <= 0 {
		maxOutputTokens = 2200
	}
	endpoint := "https://generativelanguage.googleapis.com/v1beta/models/" + url.PathEscape(model) + ":generateContent"
	payload := map[string]any{
		"contents": []map[string]any{
			{
				"role": "user",
				"parts": []map[string]string{
					{"text": prompt},
				},
			},
		},
		"generationConfig": map[string]any{
			"temperature":     0.2,
			"maxOutputTokens": maxOutputTokens,
		},
	}
	var body bytes.Buffer
	if err := json.NewEncoder(&body).Encode(payload); err != nil {
		return "", err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, &body)
	if err != nil {
		return "", err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-goog-api-key", strings.TrimSpace(key))
	resp, err := s.client.Do(httpReq)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", geminiAPIError{StatusCode: resp.StatusCode, Body: readBody(resp.Body)}
	}
	var decoded struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return "", err
	}
	for _, candidate := range decoded.Candidates {
		for _, part := range candidate.Content.Parts {
			if text := strings.TrimSpace(part.Text); text != "" {
				return text, nil
			}
		}
	}
	return "", errors.New("Gemini returned an empty review")
}

func (s *GeminiReviewService) generateOpenAICompatible(ctx context.Context, endpoint, key, model, prompt string, maxOutputTokens int, headers map[string]string) (string, error) {
	if maxOutputTokens <= 0 {
		maxOutputTokens = 2200
	}
	payload := map[string]any{
		"model": model,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
	}
	if openAIReasoningChatModel(model) {
		payload["max_completion_tokens"] = maxOutputTokens
	} else {
		payload["temperature"] = 0.2
		payload["max_tokens"] = maxOutputTokens
	}
	var body bytes.Buffer
	if err := json.NewEncoder(&body).Encode(payload); err != nil {
		return "", err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, &body)
	if err != nil {
		return "", err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+strings.TrimSpace(key))
	for name, value := range headers {
		httpReq.Header.Set(name, value)
	}
	resp, err := s.client.Do(httpReq)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", geminiAPIError{StatusCode: resp.StatusCode, Body: readBody(resp.Body)}
	}
	var decoded struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return "", err
	}
	for _, choice := range decoded.Choices {
		if text := strings.TrimSpace(choice.Message.Content); text != "" {
			return text, nil
		}
	}
	return "", errors.New("LLM returned an empty response")
}

func (s *GeminiReviewService) generateAnthropic(ctx context.Context, key, model, prompt string, maxOutputTokens int) (string, error) {
	if maxOutputTokens <= 0 {
		maxOutputTokens = 2200
	}
	payload := map[string]any{
		"model":       model,
		"max_tokens":  maxOutputTokens,
		"temperature": 0.2,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
	}
	var body bytes.Buffer
	if err := json.NewEncoder(&body).Encode(payload); err != nil {
		return "", err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.anthropic.com/v1/messages", &body)
	if err != nil {
		return "", err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", strings.TrimSpace(key))
	httpReq.Header.Set("anthropic-version", "2023-06-01")
	resp, err := s.client.Do(httpReq)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", geminiAPIError{StatusCode: resp.StatusCode, Body: readBody(resp.Body)}
	}
	var decoded struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return "", err
	}
	for _, part := range decoded.Content {
		if text := strings.TrimSpace(part.Text); text != "" {
			return text, nil
		}
	}
	return "", errors.New("Anthropic returned an empty response")
}

func openAIReasoningChatModel(model string) bool {
	model = strings.ToLower(strings.TrimSpace(model))
	if slash := strings.LastIndex(model, "/"); slash >= 0 {
		model = model[slash+1:]
	}
	return strings.HasPrefix(model, "o1") || strings.HasPrefix(model, "o3") || strings.HasPrefix(model, "o4")
}

type geminiAPIError struct {
	StatusCode int
	Body       string
}

func (e geminiAPIError) Error() string {
	body := strings.TrimSpace(e.Body)
	if body == "" {
		return fmt.Sprintf("LLM request failed with status %d", e.StatusCode)
	}
	return fmt.Sprintf("LLM request failed (%d): %s", e.StatusCode, body)
}

func isGeminiQuotaError(err error) bool {
	var apiErr geminiAPIError
	if errors.As(err, &apiErr) {
		if apiErr.StatusCode == http.StatusTooManyRequests {
			return true
		}
		body := strings.ToLower(apiErr.Body)
		return apiErr.StatusCode == http.StatusForbidden && (strings.Contains(body, "quota") || strings.Contains(body, "rate"))
	}
	return false
}

func isGeminiKeySpecificError(err error) bool {
	var apiErr geminiAPIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode == http.StatusUnauthorized || apiErr.StatusCode == http.StatusForbidden || apiErr.StatusCode == http.StatusTooManyRequests
	}
	return false
}

func geminiErrorStatusCode(err error) int {
	var apiErr geminiAPIError
	if errors.As(err, &apiErr) {
		return apiErr.StatusCode
	}
	return 0
}

func githubIssueTargetFromRepository(repository string, pullNumber int) (githubIssueTarget, error) {
	parts := strings.Split(strings.TrimSpace(repository), "/")
	if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
		return githubIssueTarget{}, fmt.Errorf("repository must be owner/repo, got %q", repository)
	}
	return githubIssueTarget{
		Owner:       strings.TrimSpace(parts[0]),
		Repo:        strings.TrimSpace(parts[1]),
		IssueNumber: pullNumber,
	}, nil
}

func reviewEvidenceProvided(pr geminiReviewPullRequest, comments []geminiReviewComment) bool {
	author := strings.TrimSpace(pr.User.Login)
	parts := []string{pr.Body}
	for _, comment := range comments {
		if comment.User.Login == author {
			parts = append(parts, comment.Body)
		}
	}
	text := strings.Join(parts, "\n\n")
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)!\[[^\]]*]\([^)]+\)`),
		regexp.MustCompile(`(?i)github\.com/user-attachments/assets/`),
		regexp.MustCompile(`(?i)\.(png|jpe?g|gif|webp|mp4|mov|webm)(\?|#|\)|\s|$)`),
		regexp.MustCompile(`(?i)\b(screenshot|screen shot|video|gif|recording|loom|imgur|user-attachments)\b`),
		regexp.MustCompile(`(?i)\b(browser check|playwright|responsive qa|viewport)\b`),
		regexp.MustCompile(`(?i)\b(go test\s+\.\/\.\.|npm test|npm run build|npm run build:local|pnpm test|yarn test)\b`),
		regexp.MustCompile(`(?i)\b(tests?|build)\s+(pass(ed)?|ok|succeed(ed)?)\b`),
	}
	for _, pattern := range patterns {
		if pattern.MatchString(text) {
			return true
		}
	}
	return false
}

func buildGeminiReviewPrompt(pr geminiReviewPullRequest, files []geminiReviewFile, comments []geminiReviewComment, linkedIssues []geminiReviewIssue, starVerified, evidenceProvided bool, maxPatchBytes int64) string {
	var builder strings.Builder
	builder.WriteString("You are MergeOS maintainer reviewer for bounty pull requests.\n")
	builder.WriteString("Review code like a senior engineer. Prioritize real bugs, regressions, security risks, broken behavior, missing tests, and unsafe scope changes.\n")
	builder.WriteString("Also enforce bounty readiness rules: repository star, evidence, bounty issue/claim context, test commands, and no unrelated rewrites.\n")
	builder.WriteString("Do not say LGTM or approve if star/evidence/tests/build/scope are missing or risky.\n")
	builder.WriteString("Write a concise GitHub PR comment in English. Start with blocking findings. If no blocking code issue is visible, say that clearly, then list readiness gaps.\n")
	builder.WriteString("Use this exact structure: `Findings`, `Bounty Readiness`, `Tests/Evidence Needed`, `Suggested Labels`.\n\n")
	builder.WriteString("Repository star verified: ")
	builder.WriteString(fmt.Sprintf("%t\n", starVerified))
	builder.WriteString("Evidence detected: ")
	builder.WriteString(fmt.Sprintf("%t\n\n", evidenceProvided))
	builder.WriteString(fmt.Sprintf("PR #%d: %s\nAuthor: %s\nURL: %s\nBase: %s %s\nHead: %s %s\nDraft: %t\n\n", pr.Number, pr.Title, pr.User.Login, pr.HTMLURL, pr.Base.Ref, pr.Base.SHA, pr.Head.Ref, pr.Head.SHA, pr.Draft))
	builder.WriteString("PR body:\n")
	builder.WriteString(truncateText(pr.Body, 5000))
	builder.WriteString("\n\nLinked bounty issues/comments:\n")
	for _, issue := range linkedIssues {
		builder.WriteString(fmt.Sprintf("- #%d %s %s\n", issue.Number, issue.Title, issue.HTMLURL))
		builder.WriteString(truncateText(issue.Body, 1200))
		builder.WriteString("\n")
	}
	if len(linkedIssues) == 0 {
		builder.WriteString("- No linked issue context fetched.\n")
	}
	builder.WriteString("\nRecent PR comments by contributor/maintainers:\n")
	for _, comment := range comments {
		if strings.Contains(comment.Body, geminiReviewMarker) {
			continue
		}
		builder.WriteString(fmt.Sprintf("- @%s: %s\n", comment.User.Login, truncateText(comment.Body, 800)))
	}
	builder.WriteString("\nChanged files and patches:\n")
	remaining := int(maxPatchBytes)
	if remaining <= 0 {
		remaining = 70000
	}
	for _, file := range files {
		header := fmt.Sprintf("\n--- %s (%s, +%d -%d) ---\n", file.Filename, file.Status, file.Additions, file.Deletions)
		if remaining <= len(header) {
			break
		}
		builder.WriteString(header)
		remaining -= len(header)
		patch := file.Patch
		if len(patch) > remaining {
			patch = truncateText(patch, remaining)
		}
		builder.WriteString(patch)
		builder.WriteString("\n")
		remaining -= len(patch)
		if remaining <= 0 {
			break
		}
	}
	return builder.String()
}

func renderGeminiReviewComment(review string, starVerified, evidenceProvided bool) string {
	readiness := []string{}
	if evidenceProvided {
		readiness = append(readiness, "- Evidence signal: `evidence: provided`")
	} else {
		readiness = append(readiness, "- Evidence signal: `evidence: missing`")
	}
	if starVerified {
		readiness = append(readiness, "- Repository star: `star: verified`")
	} else {
		readiness = append(readiness, "- Repository star: `star: missing`")
	}
	return geminiReviewMarker + "\n" + strings.TrimSpace(review) + "\n\n---\nMergeOS automated readiness signals:\n" + strings.Join(readiness, "\n")
}

func truncateText(value string, max int) string {
	value = strings.TrimSpace(value)
	if max <= 0 || len(value) <= max {
		return value
	}
	if max < 40 {
		return value[:max]
	}
	return value[:max-32] + "\n...[truncated]..."
}

func linkedIssueNumbers(text string) []int {
	pattern := regexp.MustCompile(`#([0-9]+)`)
	seen := map[int]bool{}
	numbers := []int{}
	for _, match := range pattern.FindAllStringSubmatch(text, -1) {
		var number int
		if _, err := fmt.Sscanf(match[1], "%d", &number); err == nil && number > 0 && !seen[number] {
			seen[number] = true
			numbers = append(numbers, number)
		}
	}
	sort.Ints(numbers)
	return numbers
}

func (c *adminGitHubClient) reviewPullRequest(ctx context.Context, target githubIssueTarget, number int) (geminiReviewPullRequest, error) {
	endpoint := fmt.Sprintf("https://api.github.com/repos/%s/%s/pulls/%d", url.PathEscape(target.Owner), url.PathEscape(target.Repo), number)
	var pr geminiReviewPullRequest
	if err := c.githubJSON(ctx, http.MethodGet, endpoint, nil, &pr); err != nil {
		return geminiReviewPullRequest{}, err
	}
	return pr, nil
}

func (c *adminGitHubClient) reviewPullRequestFiles(ctx context.Context, target githubIssueTarget, number int) ([]geminiReviewFile, error) {
	endpoint := fmt.Sprintf("https://api.github.com/repos/%s/%s/pulls/%d/files?per_page=100", url.PathEscape(target.Owner), url.PathEscape(target.Repo), number)
	var files []geminiReviewFile
	if err := c.githubJSON(ctx, http.MethodGet, endpoint, nil, &files); err != nil {
		return nil, err
	}
	return files, nil
}

func (c *adminGitHubClient) reviewIssueComments(ctx context.Context, target githubIssueTarget, number int) ([]geminiReviewComment, error) {
	endpoint := fmt.Sprintf("https://api.github.com/repos/%s/%s/issues/%d/comments?per_page=100", url.PathEscape(target.Owner), url.PathEscape(target.Repo), number)
	var comments []geminiReviewComment
	if err := c.githubJSON(ctx, http.MethodGet, endpoint, nil, &comments); err != nil {
		return nil, err
	}
	return comments, nil
}

func (c *adminGitHubClient) reviewIssue(ctx context.Context, target githubIssueTarget, number int) (geminiReviewIssue, error) {
	endpoint := fmt.Sprintf("https://api.github.com/repos/%s/%s/issues/%d", url.PathEscape(target.Owner), url.PathEscape(target.Repo), number)
	var issue geminiReviewIssue
	if err := c.githubJSON(ctx, http.MethodGet, endpoint, nil, &issue); err != nil {
		return geminiReviewIssue{}, err
	}
	return issue, nil
}

func (c *adminGitHubClient) linkedReviewIssues(ctx context.Context, target githubIssueTarget, pr geminiReviewPullRequest) ([]geminiReviewIssue, error) {
	numbers := linkedIssueNumbers(pr.Title + "\n" + pr.Body)
	issues := []geminiReviewIssue{}
	for _, number := range numbers {
		if number == target.IssueNumber {
			continue
		}
		issue, err := c.reviewIssue(ctx, target, number)
		if err != nil {
			continue
		}
		issues = append(issues, issue)
	}
	return issues, nil
}

func (c *adminGitHubClient) reviewAuthorStarred(ctx context.Context, target githubIssueTarget, login string) (bool, error) {
	login = strings.ToLower(strings.TrimSpace(login))
	if login == "" {
		return false, nil
	}
	for page := 1; page <= 50; page++ {
		endpoint := fmt.Sprintf("https://api.github.com/repos/%s/%s/stargazers?per_page=100&page=%d", url.PathEscape(target.Owner), url.PathEscape(target.Repo), page)
		var users []struct {
			Login string `json:"login"`
		}
		if err := c.githubJSON(ctx, http.MethodGet, endpoint, nil, &users); err != nil {
			return false, err
		}
		for _, user := range users {
			if strings.ToLower(user.Login) == login {
				return true, nil
			}
		}
		if len(users) < 100 {
			return false, nil
		}
	}
	return false, nil
}

func (c *adminGitHubClient) upsertReviewComment(ctx context.Context, target githubIssueTarget, number int, marker, body string) (string, error) {
	comments, err := c.reviewIssueComments(ctx, target, number)
	if err != nil {
		return "", err
	}
	for _, comment := range comments {
		if strings.Contains(comment.Body, marker) {
			endpoint := fmt.Sprintf("https://api.github.com/repos/%s/%s/issues/comments/%d", url.PathEscape(target.Owner), url.PathEscape(target.Repo), comment.ID)
			var updated struct {
				HTMLURL string `json:"html_url"`
			}
			if err := c.githubJSON(ctx, http.MethodPatch, endpoint, map[string]string{"body": body}, &updated); err != nil {
				return "", err
			}
			return updated.HTMLURL, nil
		}
	}
	return c.commentPullRequest(ctx, target, number, body)
}

func (c *adminGitHubClient) applyGeminiReadinessLabels(ctx context.Context, target githubIssueTarget, number int, starVerified, evidenceProvided bool) ([]string, error) {
	labels := []string{}
	evidenceLabel := "evidence: missing"
	if evidenceProvided {
		evidenceLabel = "evidence: provided"
	}
	starLabel := "star: missing"
	if starVerified {
		starLabel = "star: verified"
	}
	for _, label := range []string{evidenceLabel, starLabel} {
		if err := c.addIssueLabel(ctx, target, number, label); err != nil {
			return labels, err
		}
		labels = append(labels, label)
	}
	for _, label := range []string{"evidence: missing", "evidence: provided", "star: missing", "star: verified"} {
		if label == evidenceLabel || label == starLabel {
			continue
		}
		if err := c.removeIssueLabel(ctx, target, number, label); err != nil {
			return labels, err
		}
	}
	return labels, nil
}

func (c *adminGitHubClient) addIssueLabel(ctx context.Context, target githubIssueTarget, number int, label string) error {
	endpoint := fmt.Sprintf("https://api.github.com/repos/%s/%s/issues/%d/labels", url.PathEscape(target.Owner), url.PathEscape(target.Repo), number)
	return c.githubJSON(ctx, http.MethodPost, endpoint, map[string][]string{"labels": []string{label}}, nil)
}

func (c *adminGitHubClient) removeIssueLabel(ctx context.Context, target githubIssueTarget, number int, label string) error {
	endpoint := fmt.Sprintf("https://api.github.com/repos/%s/%s/issues/%d/labels/%s", url.PathEscape(target.Owner), url.PathEscape(target.Repo), number, url.PathEscape(label))
	err := c.githubJSON(ctx, http.MethodDelete, endpoint, nil, nil)
	if err == nil {
		return nil
	}
	var apiErr githubAPIError
	if errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusNotFound {
		return nil
	}
	return err
}

// GenerateStructured calls Gemini with a structured payload that includes responseMimeType,
// enabling JSON mode. Returns the response text and key ID. This is used by the AI evaluation service.
func (s *GeminiReviewService) GenerateStructured(ctx context.Context, payload map[string]any) (string, string, error) {
	if s.store == nil {
		return "", "", errors.New("Gemini key store is required")
	}
	candidates := s.store.GeminiAPIKeyCandidates()
	if len(candidates) == 0 {
		return "", "", errors.New("no active Gemini API keys are configured")
	}
	var lastErr error
	for _, candidate := range candidates {
		if err := s.store.MarkGeminiAPIKeyAttempt(candidate.ID); err != nil {
			return "", "", err
		}
		text, err := s.generateWithPayload(ctx, candidate.KeyValue, payload)
		if err == nil {
			if markErr := s.store.MarkGeminiAPIKeySuccess(candidate.ID, http.StatusOK); markErr != nil {
				return "", "", markErr
			}
			return text, candidate.ID, nil
		}
		lastErr = err
		statusCode := geminiErrorStatusCode(err)
		if isGeminiQuotaError(err) {
			_ = s.store.MarkGeminiAPIKeyQuotaLimited(candidate.ID, statusCode, err.Error())
			continue
		}
		if isGeminiKeySpecificError(err) {
			_ = s.store.MarkGeminiAPIKeyError(candidate.ID, statusCode, err.Error())
			continue
		}
		_ = s.store.MarkGeminiAPIKeyError(candidate.ID, statusCode, err.Error())
		return "", "", err
	}
	if lastErr == nil {
		lastErr = errors.New("Gemini structured generation failed")
	}
	return "", "", lastErr
}

// generateWithPayload sends a structured payload to Gemini (supports responseMimeType for JSON).
func (s *GeminiReviewService) generateWithPayload(ctx context.Context, key string, payload map[string]any) (string, error) {
	model := strings.Trim(strings.TrimSpace(s.cfg.GeminiReviewModel), "/")
	if model == "" {
		model = "gemini-2.5-flash"
	}
	model = strings.TrimPrefix(model, "models/")
	endpoint := "https://generativelanguage.googleapis.com/v1beta/models/" + url.PathEscape(model) + ":generateContent"
	
	var body bytes.Buffer
	if err := json.NewEncoder(&body).Encode(payload); err != nil {
		return "", err
	}
	
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, &body)
	if err != nil {
		return "", err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-goog-api-key", strings.TrimSpace(key))
	
	resp, err := s.client.Do(httpReq)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", geminiAPIError{StatusCode: resp.StatusCode, Body: readBody(resp.Body)}
	}
	
	var decoded struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return "", err
	}
	for _, candidate := range decoded.Candidates {
		for _, part := range candidate.Content.Parts {
			if text := strings.TrimSpace(part.Text); text != "" {
				return text, nil
			}
		}
	}
	return "", errors.New("Gemini returned an empty response")
}
