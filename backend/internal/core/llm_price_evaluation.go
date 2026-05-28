package core

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"
)

// LLMPriceEvaluationRequest maps project fields for LLM analysis.
type LLMPriceEvaluationRequest struct {
	Title           string   `json:"title"`
	Description     string   `json:"description"`
	Requirements    []string `json:"requirements"`
	Deliverables    []string `json:"deliverables"`
	Timeline        string   `json:"timeline"`
	TechStack       string   `json:"tech_stack"`
	Complexity      string   `json:"complexity"`
	Constraints     string   `json:"constraints"`
	ReferenceBudget int64    `json:"reference_budget"` // USD
}

// LLMPriceEvaluationResponse is the structured result from LLM analysis.
type LLMPriceEvaluationResponse struct {
	SuggestedLow    int64            `json:"suggested_low"`
	SuggestedHigh   int64            `json:"suggested_high"`
	ConfidenceLevel float64          `json:"confidence_level"`
	TaskBreakdown   map[string]int64 `json:"task_breakdown"`
	Assumptions     []string         `json:"assumptions"`
	Risks           []string         `json:"risks"`
	Rationale       string           `json:"rationale"`
	Editable        bool             `json:"editable"`
}

const llmPriceEvalMaxTokens = 1600

func buildLLMPriceEvaluationPrompt(req LLMPriceEvaluationRequest) string {
	var b strings.Builder
	b.WriteString(`You are an expert software project estimator at MergeOS. Analyze the project details below and provide a structured price evaluation.

Respond ONLY with a JSON object (no markdown fences, no extra text) using this exact schema:
{
  "suggested_low": number,
  "suggested_high": number,
  "confidence_level": number,
  "task_breakdown": { "category_name": amount_in_usd },
  "assumptions": ["string", ...],
  "risks": ["string", ...],
  "rationale": "string"
}

Rules:
- suggested_low and suggested_high are in USD (whole dollars, no cents).
- suggested_low must be <= suggested_high.
- confidence_level is 0.0 to 1.0 (low detail = lower confidence).
- task_breakdown: 3-6 categories covering the scope, each in USD summing roughly to the midpoint.
- assumptions: 2-4 items based on project details provided.
- risks: 2-3 items highlighting real risk factors.
- rationale: 2-3 sentences explaining the estimate.
- Consider tech stack complexity, number of deliverables, timeline pressure, and stated constraints.
`)
	appendEvalField(&b, "Title", req.Title)
	appendEvalField(&b, "Description", req.Description)
	if len(req.Requirements) > 0 {
		appendEvalField(&b, "Requirements", strings.Join(req.Requirements, "\n- "))
	}
	if len(req.Deliverables) > 0 {
		appendEvalField(&b, "Deliverables", strings.Join(req.Deliverables, "\n- "))
	}
	appendEvalField(&b, "Timeline", req.Timeline)
	appendEvalField(&b, "Tech Stack", req.TechStack)
	appendEvalField(&b, "Complexity", req.Complexity)
	appendEvalField(&b, "Constraints", req.Constraints)
	if req.ReferenceBudget > 0 {
		appendEvalField(&b, "Reference Budget (USD)", fmt.Sprintf("%d", req.ReferenceBudget))
	}
	b.WriteString("\nReturn ONLY the JSON object. No markdown, no explanation.\n")
	return b.String()
}

func appendEvalField(b *strings.Builder, name, value string) {
	value = strings.TrimSpace(value)
	if value == "" {
		return
	}
	b.WriteString("\n## ")
	b.WriteString(name)
	b.WriteString("\n")
	b.WriteString(value)
	b.WriteString("\n")
}

// EvaluateProjectLLM performs the LLM evaluation and returns a structured response.
// Falls back to a rule-based estimate if the LLM is unavailable.
func (s *Server) EvaluateProjectLLM(ctx context.Context, req LLMPriceEvaluationRequest) (*LLMPriceEvaluationResponse, error) {
	if s.geminiReviewer == nil || !s.geminiReviewer.Ready() {
		return fallbackPriceEvaluation(req), nil
	}
	prompt := buildLLMPriceEvaluationPrompt(req)
	raw, _, _, err := s.geminiReviewer.generate(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("LLM evaluation failed: %w", err)
	}
	resp, err := parseLLMPriceEvaluation(raw)
	if err != nil {
		return fallbackPriceEvaluation(req), nil
	}
	resp.Editable = true
	return resp, nil
}

// parseLLMPriceEvaluation parses the LLM JSON response into a structured result.
func parseLLMPriceEvaluation(raw string) (*LLMPriceEvaluationResponse, error) {
	raw = stripMarkdownFence(raw)
	var resp LLMPriceEvaluationResponse
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		return nil, fmt.Errorf("failed to parse LLM response: %w", err)
	}
	// Validate and clamp
	if resp.SuggestedLow <= 0 {
		resp.SuggestedLow = 200
	}
	if resp.SuggestedHigh <= 0 || resp.SuggestedHigh < resp.SuggestedLow {
		resp.SuggestedHigh = resp.SuggestedLow * 2
	}
	if resp.ConfidenceLevel <= 0 || resp.ConfidenceLevel > 1 {
		resp.ConfidenceLevel = 0.7
	}
	if len(resp.TaskBreakdown) == 0 {
		mid := (resp.SuggestedLow + resp.SuggestedHigh) / 2
		resp.TaskBreakdown = map[string]int64{
			"Core Development":     int64(math.Round(float64(mid) * 0.50 / 50) * 50),
			"Integration & QA":     int64(math.Round(float64(mid) * 0.30 / 50) * 50),
			"Project Management":   int64(math.Round(float64(mid) * 0.20 / 50) * 50),
		}
	}
	if len(resp.Assumptions) == 0 {
		resp.Assumptions = []string{"Standard development lifecycle with testing and deployment."}
	}
	if len(resp.Risks) == 0 {
		resp.Risks = []string{"Scope changes during development may affect pricing."}
	}
	if resp.Rationale == "" {
		resp.Rationale = "Estimate based on project scope, deliverables, and tech stack."
	}
	return &resp, nil
}

func stripMarkdownFence(text string) string {
	text = strings.TrimSpace(text)
	// Remove ```json ... ``` or ``` ... ``` fences
	if strings.HasPrefix(text, "```") {
		lines := strings.SplitN(text, "\n", 2)
		if len(lines) > 1 {
			rest := lines[1]
			if idx := strings.LastIndex(rest, "```"); idx >= 0 {
				rest = rest[:idx]
			}
			text = strings.TrimSpace(rest)
		}
	}
	return text
}

// fallbackPriceEvaluation produces a rule-based estimate when the LLM is unavailable.
func fallbackPriceEvaluation(req LLMPriceEvaluationRequest) *LLMPriceEvaluationResponse {
	base := 1000.0

	// Tech stack adjustments
	tech := strings.ToLower(req.TechStack)
	if strings.Contains(tech, "react") || strings.Contains(tech, "vue") || strings.Contains(tech, "next") || strings.Contains(tech, "angular") {
		base += 300
	}
	if strings.Contains(tech, "go") || strings.Contains(tech, "rust") || strings.Contains(tech, "fastapi") || strings.Contains(tech, "python") {
		base += 400
	}
	if strings.Contains(tech, "ai") || strings.Contains(tech, "llm") || strings.Contains(tech, "machine learning") || strings.Contains(tech, "ml") {
		base += 800
	}
	if strings.Contains(tech, "kubernetes") || strings.Contains(tech, "docker") || strings.Contains(tech, "devops") || strings.Contains(tech, "aws") || strings.Contains(tech, "gcp") {
		base += 500
	}
	if strings.Contains(tech, "solidity") || strings.Contains(tech, "web3") || strings.Contains(tech, "ethereum") || strings.Contains(tech, "solana") {
		base += 700
	}

	// Deliverables add cost
	deliverableCount := len(req.Deliverables)
	base += float64(deliverableCount * 200)

	// Requirement detail
	reqCount := len(req.Requirements)
	if reqCount > 3 {
		base += float64(reqCount) * 100
	}

	// Complexity multiplier
	complexity := strings.ToLower(req.Complexity)
	switch complexity {
	case "high", "advanced", "complex", "very high", "critical":
		base *= 1.6
	case "medium", "moderate", "intermediate":
		base *= 1.2
	case "low", "simple", "easy":
		base *= 0.8
	}

	// Timeline pressure
	timeline := strings.ToLower(req.Timeline)
	if strings.Contains(timeline, "urgent") || strings.Contains(timeline, "asap") || strings.Contains(timeline, "yesterday") {
		base *= 1.3
	}

	// Constraints add overhead
	if req.Constraints != "" {
		base += 300
	}

	// Blend with reference budget if provided
	if req.ReferenceBudget > 0 {
		base = base*0.7 + float64(req.ReferenceBudget)*0.3
	}

	if base < 150 {
		base = 150
	}

	low := int64(math.Round(base*0.85/50) * 50)
	high := int64(math.Round(base*1.25/50) * 50)
	mid := int64(math.Round(base / 50) * 50)

	breakdown := map[string]int64{
		"Core Features & Logic": int64(math.Round(float64(mid)*0.50/50) * 50),
		"Frontend Integration":  int64(math.Round(float64(mid)*0.25/50) * 50),
		"Testing & CI/CD":       int64(math.Round(float64(mid)*0.15/50) * 50),
		"Project Management":    int64(math.Round(float64(mid)*0.10/50) * 50),
	}

	assumptions := []string{
		"Estimate assumes well-defined interfaces and clean design documents.",
		"Development follows standard lifecycle with code review and automated testing.",
	}
	if deliverableCount > 0 {
		assumptions = append(assumptions, fmt.Sprintf("All %d deliverables are independently testable.", deliverableCount))
	}

	risks := []string{
		"Scope creep from ambiguous or changing deliverables.",
	}
	if strings.Contains(tech, "ai") || strings.Contains(tech, "llm") || strings.Contains(tech, "ml") {
		risks = append(risks, "AI model non-determinism and API latency/rate limits.")
	}
	if strings.Contains(timeline, "urgent") || strings.Contains(timeline, "asap") {
		risks = append(risks, "Urgent timelines may require parallel workstreams and higher coordination overhead.")
	}

	rationale := fmt.Sprintf(
		"Tech stack (%s) and %d deliverables drive the estimate. Complexity %q with %s timeline.",
		req.TechStack, deliverableCount, req.Complexity, req.Timeline,
	)

	return &LLMPriceEvaluationResponse{
		SuggestedLow:    low,
		SuggestedHigh:   high,
		ConfidenceLevel: 0.75,
		TaskBreakdown:   breakdown,
		Assumptions:     assumptions,
		Risks:           risks,
		Rationale:       rationale,
		Editable:        true,
	}
}
