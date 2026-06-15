package core

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

func TestEvaluateProjectPriceReturnsStructuredEditableSuggestion(t *testing.T) {
	result, err := EvaluateProjectPrice(ProjectPriceEvaluationRequest{
		Title:        "AI pricing workflow",
		Description:  "Build an authenticated web app that imports project details and suggests bounty prices.",
		ProjectType:  "AI / ML",
		Requirements: "Use a testable service layer, structured API response, loading and retry states, and manual override before publishing.",
		Deliverables: []string{"API endpoint", "Estimator UI", "Tests", "Documentation"},
		Timeline:     "urgent two week launch",
		TechStack:    "Go, Vue, PostgreSQL",
		Complexity:   "high",
		Constraints:  "No client-side secrets and deterministic fallback when AI providers are unavailable.",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.SuggestedPriceCents <= 0 || result.SuggestedRange.LowCents <= 0 || result.SuggestedRange.HighCents < result.SuggestedRange.LowCents {
		t.Fatalf("invalid price range: %#v", result)
	}
	if result.ProtocolVersion != "mergeos.estimate.v1" || result.Kind != "project_estimate" {
		t.Fatalf("unexpected estimate protocol header: %#v", result)
	}
	if !result.Editable {
		t.Fatal("price suggestion must be editable before publishing")
	}
	if result.Confidence == "low" {
		t.Fatalf("confidence = %q", result.Confidence)
	}
	if len(result.Breakdown) < 4 || len(result.Assumptions) == 0 || len(result.Risks) == 0 {
		t.Fatalf("missing structured details: %#v", result)
	}
}

func TestEvaluateProjectPriceRouteRequiresAuthAndReturnsJSON(t *testing.T) {
	tempDir := t.TempDir()
	cfg := Config{
		TokenSymbol:       defaultTokenSymbol,
		StatePath:         filepath.Join(tempDir, "state.json"),
		PlatformFeeBps:    1000,
		DevPaymentEnabled: true,
		DevPaymentCode:    defaultDevPaymentCode,
		GitHubOwner:       defaultGitHubOwner,
		BountyRoot:        filepath.Join(tempDir, "bounties"),
		SMTPFrom:          "noreply@mergeos.local",
	}
	payments := NewPaymentManager(cfg)
	store, err := NewStore(cfg, payments, NewRepoFactory(cfg), NewEmailSender(cfg))
	if err != nil {
		t.Fatal(err)
	}
	auth, err := store.Register(RegisterRequest{
		Name:        "Pricing Client",
		CompanyName: "Pricing Co",
		Email:       "pricing@example.com",
		Password:    "password123",
	})
	if err != nil {
		t.Fatal(err)
	}

	server := NewServer(cfg, store, payments)
	payload := ProjectPriceEvaluationRequest{
		Description:  "Create a customer dashboard with project imports and estimation controls.",
		ProjectType:  "Web Development",
		Requirements: "Display structured results, retry failures, and allow the user to edit the accepted budget.",
		Deliverables: []string{"Backend API", "Vue UI", "Tests"},
		TechStack:    "Go, Vue",
	}
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/projects/evaluate-price", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+auth.Token)
	resp := httptest.NewRecorder()
	server.Routes().ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", resp.Code, resp.Body.String())
	}
	var result ProjectPriceEvaluationResponse
	if err := json.Unmarshal(resp.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.ProtocolVersion != "mergeos.estimate.v1" || result.Kind != "project_estimate" {
		t.Fatalf("unexpected estimate protocol header: %#v", result)
	}
	if result.SuggestedPriceCents == 0 || len(result.Breakdown) == 0 || !result.Editable {
		t.Fatalf("unexpected response: %#v", result)
	}
}

type mockRoundTripper struct {
	roundTrip func(*http.Request) (*http.Response, error)
}

func (m *mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return m.roundTrip(req)
}

func TestEvaluateProjectPriceRouteWithLLMReady(t *testing.T) {
	tempDir := t.TempDir()
	cfg := Config{
		TokenSymbol:       defaultTokenSymbol,
		StatePath:         filepath.Join(tempDir, "state.json"),
		PlatformFeeBps:    1000,
		DevPaymentEnabled: true,
		DevPaymentCode:    defaultDevPaymentCode,
		GitHubOwner:       defaultGitHubOwner,
		BountyRoot:        filepath.Join(tempDir, "bounties"),
		SMTPFrom:          "noreply@mergeos.local",
	}
	payments := NewPaymentManager(cfg)
	store, err := NewStore(cfg, payments, NewRepoFactory(cfg), NewEmailSender(cfg))
	if err != nil {
		t.Fatal(err)
	}
	auth, err := store.Register(RegisterRequest{
		Name:        "Pricing Client",
		CompanyName: "Pricing Co",
		Email:       "pricing@example.com",
		Password:    "password123",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Add an API key so that geminiReviewer is Ready()
	key, err := store.AddGeminiAPIKey("test-gemini-api-key-value-12345")
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.RecordGeminiAPIKeyTestResult(key.ID, GeminiAPIKeyStatusActive, http.StatusOK, "")
	if err != nil {
		t.Fatal(err)
	}

	server := NewServer(cfg, store, payments)

	innerJSON := `{
		"suggested_low": 1500,
		"suggested_high": 3000,
		"confidence_level": 0.9,
		"task_breakdown": {
			"Backend API": 1000,
			"Frontend UI": 1000,
			"Testing": 500
		},
		"assumptions": ["Mocked assumption 1", "Mocked assumption 2"],
		"risks": ["Mocked risk 1"],
		"rationale": "Mocked rationale"
	}`

	type geminiPart struct {
		Text string `json:"text"`
	}
	type geminiContent struct {
		Parts []geminiPart `json:"parts"`
	}
	type geminiCandidate struct {
		Content geminiContent `json:"content"`
	}
	type geminiResponse struct {
		Candidates []geminiCandidate `json:"candidates"`
	}

	geminiResp := geminiResponse{
		Candidates: []geminiCandidate{
			{
				Content: geminiContent{
					Parts: []geminiPart{
						{Text: innerJSON},
					},
				},
			},
		},
	}

	mockResponseBytes, err := json.Marshal(geminiResp)
	if err != nil {
		t.Fatal(err)
	}

	mockRT := &mockRoundTripper{
		roundTrip: func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
				Body:       io.NopCloser(bytes.NewReader(mockResponseBytes)),
			}, nil
		},
	}
	server.geminiReviewer.client = &http.Client{Transport: mockRT}

	payload := ProjectPriceEvaluationRequest{
		Title:        "Mocked Project",
		Description:  "Create a customer dashboard with project imports and estimation controls.",
		ProjectType:  "Web Development",
		Requirements: "Display structured results, retry failures, and allow the user to edit the accepted budget.",
		Deliverables: []string{"Backend API", "Vue UI", "Tests"},
		TechStack:    "Go, Vue",
		Complexity:   "medium",
	}
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/projects/evaluate-price", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+auth.Token)
	resp := httptest.NewRecorder()
	server.Routes().ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", resp.Code, resp.Body.String())
	}

	var result ProjectPriceEvaluationResponse
	if err := json.Unmarshal(resp.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}

	if result.ProtocolVersion != "mergeos.estimate.v1" || result.Kind != "project_estimate" {
		t.Fatalf("unexpected estimate protocol header: %#v", result)
	}
	if result.SuggestedPriceCents != 225000 { // (1500+3000)/2 = 2250 -> 225000 cents
		t.Fatalf("unexpected suggested price: %d", result.SuggestedPriceCents)
	}
	if result.SuggestedRange.LowCents != 150000 || result.SuggestedRange.HighCents != 300000 {
		t.Fatalf("unexpected suggested range: %#v", result.SuggestedRange)
	}
	if result.Confidence != "high" {
		t.Fatalf("expected high confidence, got %q", result.Confidence)
	}
	if len(result.Breakdown) != 3 {
		t.Fatalf("expected 3 breakdown items, got %d", len(result.Breakdown))
	}
	if len(result.Assumptions) != 2 || result.Assumptions[0] != "Mocked assumption 1" {
		t.Fatalf("unexpected assumptions: %#v", result.Assumptions)
	}
	if len(result.Risks) != 1 || result.Risks[0] != "Mocked risk 1" {
		t.Fatalf("unexpected risks: %#v", result.Risks)
	}
}
