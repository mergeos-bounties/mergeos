package core

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
)

// AIEvaluationProvider defines which LLM provider to use.
type AIEvaluationProvider string

const (
	AIProviderGemini AIEvaluationProvider = "gemini"
	AIProviderOpenAI AIEvaluationProvider = "openai"
)

// AIEvaluationService handles AI-powered project price evaluation
// using the existing Gemini infrastructure with optional OpenAI fallback.
type AIEvaluationService struct {
	cfg   Config
	store *Store
}

func NewAIEvaluationService(cfg Config, store *Store) *AIEvaluationService {
	return &AIEvaluationService{cfg: cfg, store: store}
}

// aiPricePrompt builds a structured prompt for the LLM.
func aiPricePrompt(req EvaluateProjectRequest) string {
	var b strings.Builder
	b.WriteString(`You are a software project pricing expert. Analyze the project and return ONLY valid JSON matching this schema:
{
  "suggested_low": 5000,
  "suggested_high": 12000,
  "confidence_level": 0.85,
  "task_breakdown": {
    "Core Features": 8000,
    "Frontend": 4000,
    "Testing": 2000
  },
  "assumptions": ["List of assumptions"],
  "risks": ["List of risks"],
  "rationale": "Brief explanation"
}

Project details:
`)
	if req.Description != "" {
		fmt.Fprintf(&b, "Description: %s\n", req.Description)
	}
	if req.TechStack != "" {
		fmt.Fprintf(&b, "Tech Stack: %s\n", req.TechStack)
	}
	if len(req.Deliverables) > 0 {
		fmt.Fprintf(&b, "Deliverables: %s\n", strings.Join(req.Deliverables, ", "))
	}
	if req.Timeline != "" {
		fmt.Fprintf(&b, "Timeline: %s\n", req.Timeline)
	}
	if req.Complexity != "" {
		fmt.Fprintf(&b, "Complexity: %s\n", req.Complexity)
	}
	if req.Constraints != "" {
		fmt.Fprintf(&b, "Constraints: %s\n", req.Constraints)
	}
	if len(req.Requirements) > 0 {
		fmt.Fprintf(&b, "Requirements: %s\n", strings.Join(req.Requirements, "; "))
	}
	if req.ReferenceBudget > 0 {
		fmt.Fprintf(&b, "Reference Budget (USD): %d\n", req.ReferenceBudget)
	}
	b.WriteString("\nReturn ONLY the JSON object, no markdown, no code fences.")
	return b.String()
}

// EvaluateWithGemini calls the Gemini API using the existing key pool.
func (s *AIEvaluationService) EvaluateWithGemini(ctx context.Context, req EvaluateProjectRequest) (*EvaluateProjectResponse, error) {
	if s.store == nil || !s.store.HasRunnableGeminiAPIKey() {
		return nil, errors.New("no Gemini API keys available")
	}

	// Use the existing GeminiReviewService pattern
	reviewer := NewGeminiReviewService(s.cfg, s.store)
	responseMimeType := "application/json"

	model := strings.Trim(strings.TrimSpace(s.cfg.GeminiReviewModel), "/")
	if model == "" {
		model = "gemini-2.5-flash"
	}
	model = strings.TrimPrefix(model, "models/")

	prompt := aiPricePrompt(req)
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
			"temperature":       0.2,
			"responseMimeType":  responseMimeType,
			"maxOutputTokens":   1024,
		},
	}

	candidates := s.store.GeminiAPIKeyCandidates()
	var lastErr error
	for _, candidate := range candidates {
		_ = s.store.MarkGeminiAPIKeyAttempt(candidate.ID)
		text, err := reviewer.generateWithKey(ctx, candidate.KeyValue, payloadJSON(payload))
		if err == nil {
			_ = s.store.MarkGeminiAPIKeySuccess(candidate.ID, 200)
			return parseAIEvaluationResponse(text)
		}
		lastErr = err
		if isGeminiQuotaError(err) {
			_ = s.store.MarkGeminiAPIKeyQuotaLimited(candidate.ID, geminiErrorStatusCode(err), err.Error())
			continue
		}
		if isGeminiKeySpecificError(err) {
			_ = s.store.MarkGeminiAPIKeyError(candidate.ID, geminiErrorStatusCode(err), err.Error())
			continue
		}
		_ = s.store.MarkGeminiAPIKeyError(candidate.ID, geminiErrorStatusCode(err), err.Error())
		return nil, fmt.Errorf("AI evaluation failed: %w", err)
	}
	if lastErr == nil {
		lastErr = errors.New("AI evaluation: no Gemini keys available")
	}
	return nil, lastErr
}

// EvaluateWithRuleBased uses the existing heuristic engine.
func (s *AIEvaluationService) EvaluateWithRuleBased(req EvaluateProjectRequest) *EvaluateProjectResponse {
	return evaluateProjectRuleBased(req)
}

// evaluateProjectRuleBased is the existing rule-based evaluation logic.
func evaluateProjectRuleBased(req EvaluateProjectRequest) *EvaluateProjectResponse {
	var basePrice int64 = 1000
	tech := strings.ToLower(req.TechStack)
	if strings.Contains(tech, "react") || strings.Contains(tech, "vue") || strings.Contains(tech, "next") {
		basePrice += 300
	}
	if strings.Contains(tech, "go") || strings.Contains(tech, "rust") || strings.Contains(tech, "fastapi") {
		basePrice += 400
	}
	if strings.Contains(tech, "ai") || strings.Contains(tech, "llm") || strings.Contains(tech, "machine learning") {
		basePrice += 800
	}
	if strings.Contains(tech, "kubernetes") || strings.Contains(tech, "docker") || strings.Contains(tech, "devops") {
		basePrice += 500
	}

	basePrice += int64(len(req.Deliverables) * 150)
	basePrice += int64(len(req.Requirements) * 100)

	complexity := strings.ToLower(req.Complexity)
	if complexity == "high" {
		basePrice = int64(float64(basePrice) * 1.6)
	} else if complexity == "low" {
		basePrice = int64(float64(basePrice) * 0.8)
	}

	if req.ReferenceBudget > 0 {
		basePrice = (basePrice + req.ReferenceBudget) / 2
	}

	if basePrice < 150 {
		basePrice = 150
	}

	low := int64(float64(basePrice) * 0.85)
	high := int64(float64(basePrice) * 1.25)
	low = (low / 50) * 50
	high = (high / 50) * 50

	breakdown := map[string]int64{
		"Core Features & Logic": int64(float64(basePrice) * 0.50),
		"Frontend Integration":  int64(float64(basePrice) * 0.25),
		"Testing & CI/CD":       int64(float64(basePrice) * 0.15),
		"Project Management":    int64(float64(basePrice) * 0.10),
	}

	assumptions := []string{
		"The project has well-defined interfaces and clean design docs.",
		"Development will be conducted in a sandbox or staging environment.",
	}
	if len(req.Deliverables) > 0 {
		assumptions = append(assumptions, fmt.Sprintf("All %d listed deliverables are independent and testable.", len(req.Deliverables)))
	}
	if strings.Contains(tech, "go") {
		assumptions = append(assumptions, "The project relies on native Go modules and clean standard library conventions.")
	}

	risks := []string{"Scope creep due to changing or ambiguous deliverables."}
	if strings.Contains(tech, "ai") || strings.Contains(tech, "llm") {
		risks = append(risks, "AI model non-determinism and API latency/rate limits.")
	}
	if strings.Contains(tech, "kubernetes") || strings.Contains(tech, "docker") {
		risks = append(risks, "Configuration drifts and target environment deployment discrepancies.")
	}

	return &EvaluateProjectResponse{
		SuggestedLow:    low,
		SuggestedHigh:   high,
		ConfidenceLevel: 0.90,
		TaskBreakdown:   breakdown,
		Assumptions:     assumptions,
		Risks:           risks,
		Rationale:       fmt.Sprintf("Based on the tech stack (%s), the estimated effort is %s complexity.", req.TechStack, req.Complexity),
	}
}

// Evaluate performs AI evaluation with Gemini, falling back to rule-based.
func (s *AIEvaluationService) Evaluate(ctx context.Context, req EvaluateProjectRequest) (*EvaluateProjectResponse, string, error) {
	if s.store != nil && s.store.HasRunnableGeminiAPIKey() {
		resp, err := s.EvaluateWithGemini(ctx, req)
		if err == nil {
			return resp, "gemini", nil
		}
		// Fall through to rule-based
	}
	return s.EvaluateWithRuleBased(req), "rule-based", nil
}

// parseAIEvaluationResponse parses the LLM text response into EvaluateProjectResponse.
func parseAIEvaluationResponse(text string) (*EvaluateProjectResponse, error) {
	text = strings.TrimSpace(text)
	// Strip markdown code fences
	if strings.HasPrefix(text, "```") {
		lines := strings.Split(text, "\n")
		var clean []string
		for _, line := range lines {
			if strings.HasPrefix(line, "```") {
				continue
			}
			clean = append(clean, line)
		}
		text = strings.Join(clean, "\n")
		text = strings.TrimSpace(text)
	}

	var resp EvaluateProjectResponse
	if err := json.Unmarshal([]byte(text), &resp); err != nil {
		return nil, fmt.Errorf("AI evaluation parse error: %w", err)
	}

	// Validate and fixup
	if resp.SuggestedLow <= 0 {
		resp.SuggestedLow = 500
	}
	if resp.SuggestedHigh <= resp.SuggestedLow {
		resp.SuggestedHigh = int64(math.Round(float64(resp.SuggestedLow) * 1.5))
	}
	if resp.ConfidenceLevel <= 0 {
		resp.ConfidenceLevel = 0.7
	}
	if len(resp.TaskBreakdown) == 0 {
		resp.TaskBreakdown = map[string]int64{"Development": resp.SuggestedHigh}
	}
	if len(resp.Assumptions) == 0 {
		resp.Assumptions = []string{"AI evaluation provided the estimate based on project details."}
	}
	if len(resp.Risks) == 0 {
		resp.Risks = []string{"Scope changes after review may affect final price."}
	}
	if resp.Rationale == "" {
		resp.Rationale = "AI-generated estimate based on project scope and complexity."
	}

	return &resp, nil
}

// payloadJSON serializes an arbitrary map as JSON text.
func payloadJSON(v any) string {
	data, err := json.Marshal(v)
	if err != nil {
		return "{}"
	}
	return string(data)
}
