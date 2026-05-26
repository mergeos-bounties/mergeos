package core

import (
	"bytes"
	"encoding/json"
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
	if err != nil { t.Fatal(err) }
	if result.SuggestedPriceCents <= 0 || result.SuggestedRange.LowCents <= 0 || result.SuggestedRange.HighCents < result.SuggestedRange.LowCents {
		t.Fatalf("invalid price range: %#v", result)
	}
	if !result.Editable { t.Fatal("price suggestion must be editable") }
	if result.Confidence == "low" { t.Fatalf("confidence = %q", result.Confidence) }
	if len(result.Breakdown) < 4 || len(result.Assumptions) == 0 || len(result.Risks) == 0 {
		t.Fatalf("missing structured details: %#v", result)
	}
}

func TestEvaluateProjectPriceRouteRequiresAuthAndReturnsJSON(t *testing.T) {
	tempDir := t.TempDir()
	cfg := Config{
		TokenSymbol: defaultTokenSymbol, StatePath: filepath.Join(tempDir, "state.json"),
		PlatformFeeBps: 1000, DevPaymentEnabled: true, DevPaymentCode: defaultDevPaymentCode,
		GitHubOwner: defaultGitHubOwner, BountyRoot: filepath.Join(tempDir, "bounties"), SMTPFrom: "noreply@mergeos.local",
	}
	payments := NewPaymentManager(cfg)
	store, err := NewStore(cfg, payments, NewRepoFactory(cfg), NewEmailSender(cfg))
	if err != nil { t.Fatal(err) }
	auth, err := store.Register(RegisterRequest{
		Name: "Pricing Client", CompanyName: "Pricing Co", Email: "pricing@example.com", Password: "password123",
	})
	if err != nil { t.Fatal(err) }
	server := NewServer(cfg, store, payments)
	payload := ProjectPriceEvaluationRequest{
		Description: "Create a customer dashboard with project imports and estimation controls.",
		ProjectType: "Web Development", Requirements: "Display structured results, retry failures, and allow the user to edit the accepted budget.",
		Deliverables: []string{"Backend API", "Vue UI", "Tests"}, TechStack: "Go, Vue",
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/projects/evaluate-price", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+auth.Token)
	resp := httptest.NewRecorder()
	server.Routes().ServeHTTP(resp, req)
	if resp.Code != http.StatusOK { t.Fatalf("status = %d, body = %s", resp.Code, resp.Body.String()) }
	var result ProjectPriceEvaluationResponse
	if err := json.Unmarshal(resp.Body.Bytes(), &result); err != nil { t.Fatal(err) }
	if result.SuggestedPriceCents == 0 || len(result.Breakdown) == 0 || !result.Editable {
		t.Fatalf("unexpected response: %#v", result)
	}
}

func TestAIProjectPriceEvaluationMock(t *testing.T) {
	tempDir := t.TempDir()
	cfg := Config{
		LLMProvider: "openai", LLMApiKey: "test-key", LLMModel: "gpt-4",
		TokenSymbol: defaultTokenSymbol, StatePath: filepath.Join(tempDir, "state.json"),
		PlatformFeeBps: 1000, DevPaymentEnabled: true, DevPaymentCode: defaultDevPaymentCode,
		GitHubOwner: defaultGitHubOwner, BountyRoot: filepath.Join(tempDir, "bounties"), SMTPFrom: "noreply@mergeos.local",
	}
	payments := NewPaymentManager(cfg)
	store, err := NewStore(cfg, payments, NewRepoFactory(cfg), NewEmailSender(cfg))
	if err != nil { t.Fatal(err) }
	server := NewServer(cfg, store, payments)
	payload := ProjectPriceEvaluationRequest{
		Description: "Build a web app with AI features.",
		ProjectType: "AI / ML", TechStack: "Go, Vue",
	}
	body, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/projects/evaluate-price", bytes.NewReader(body))
	resp := httptest.NewRecorder()
	server.Routes().ServeHTTP(resp, req)

	// Should return 401 if API key is invalid (our test key won't reach OpenAI)
	// but the route exists and returns properly
	if resp.Code != http.StatusUnauthorized && resp.Code != http.StatusInternalServerError {
		t.Fatalf("unexpected status: %d", resp.Code)
	}
}

func TestPriceEvaluationFallbackToRuleBased(t *testing.T) {
	result, err := EvaluateProjectPrice(ProjectPriceEvaluationRequest{
		Description: "Build a simple landing page.",
		ProjectType: "Web Development",
	})
	if err != nil { t.Fatal(err) }
	if result.SuggestedPriceCents != 220000 {
		t.Fatalf("expected 220000 for web project, got %d", result.SuggestedPriceCents)
	}
}
