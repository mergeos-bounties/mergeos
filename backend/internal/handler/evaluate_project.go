// backend/internal/handler/evaluate_project.go
// AI Project Price Evaluator - Issue #3
//
// This handler accepts project details and uses an AI service
// to suggest a reasonable price for the project.

package handler

import (
	"encoding/json"
	"net/http"
	"os"
)

type EvaluateRequest struct {
	Description     string `json:"description"`
	Requirements    string `json:"requirements"`
	Deliverables    string `json:"deliverables"`
	Timeline        string `json:"timeline"`
	TechStack       string `json:"techStack"`
	Complexity      string `json:"complexity"`
	Constraints     string `json:"constraints"`
	ReferenceBudget string `json:"referenceBudget"`
}

type BreakdownItem struct {
	Category   string  `json:"category"`
	Estimate   float64 `json:"estimate"`
	Percentage int     `json:"percentage"`
}

type RiskFactor struct {
	Factor      string `json:"factor"`
	Impact      string `json:"impact"`
	Probability string `json:"probability"`
}

type EvaluateResponse struct {
	SuggestedPrice float64         `json:"suggestedPrice"`
	Confidence     int             `json:"confidence"`
	Breakdown      []BreakdownItem `json:"breakdown"`
	Assumptions    []string        `json:"assumptions"`
	Risks          []RiskFactor    `json:"risks"`
	Explanation    string          `json:"explanation"`
}

// Validate checks required fields
func (r *EvaluateRequest) Validate() string {
	if r.Description == "" {
		return "Project description is required"
	}
	if r.Timeline == "" {
		return "Timeline is required"
	}
	return ""
}

// EvaluateProjectHandler handles POST /api/ai/evaluate-project
// API key is read from AI_API_KEY env var, never hardcoded
func EvaluateProjectHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	var req EvaluateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"Invalid request body"}`, http.StatusBadRequest)
		return
	}

	if msg := req.Validate(); msg != "" {
		http.Error(w, `{"error":"`+msg+`"}`, http.StatusBadRequest)
		return
	}

	// Read API key from environment - never hardcoded
	apiKey := os.Getenv("AI_API_KEY")
	if apiKey == "" {
		http.Error(w, `{"error":"AI service not configured"}`, http.StatusServiceUnavailable)
		return
	}

	// Call AI service (this would be an external API call)
	result, err := callAIService(apiKey, &req)
	if err != nil {
		http.Error(w, `{"error":"AI evaluation failed: `+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// callAIService sends the project details to the AI service and parses the response
// This logic is kept separate from the UI layer for testability
func callAIService(apiKey string, req *EvaluateRequest) (*EvaluateResponse, error) {
	// Build prompt for AI
	prompt := buildEvaluationPrompt(req)

	// TODO: Replace with actual AI API call (OpenAI, Claude, etc.)
	// For now, return a structured mock response
	// In production, this would call an AI API with the prompt
	_ = apiKey
	_ = prompt

	return mockEvaluate(req), nil
}

// buildEvaluationPrompt creates a structured prompt for the AI
func buildEvaluationPrompt(req *EvaluateRequest) string {
	return `You are a project pricing expert. Given the following project details, suggest a reasonable price.

Project Description: ` + req.Description + `
Requirements: ` + req.Requirements + `
Deliverables: ` + req.Deliverables + `
Timeline: ` + req.Timeline + `
Tech Stack: ` + req.TechStack + `
Complexity: ` + req.Complexity + `
Constraints: ` + req.Constraints + `
Reference Budget: ` + req.ReferenceBudget + `

Return a JSON object with:
- suggestedPrice (number): The suggested price in USD
- confidence (0-100): How confident you are in this estimate
- breakdown (array): {category, estimate, percentage}
- assumptions (array of strings): Key assumptions made
- risks (array): {factor, impact, probability}
- explanation (string): Brief explanation of the pricing logic`
}

// mockEvaluate returns a structured mock response for testing
func mockEvaluate(req *EvaluateRequest) *EvaluateResponse {
	basePrice := 5000.0

	switch req.Complexity {
	case "low":
		basePrice = 3000
	case "medium":
		basePrice = 8000
	case "high":
		basePrice = 15000
	case "very-high":
		basePrice = 30000
	}

	if req.TechStack != "" {
		basePrice *= 1.2
	}

	return &EvaluateResponse{
		SuggestedPrice: basePrice,
		Confidence:     85,
		Breakdown: []BreakdownItem{
			{Category: "Planning & Design", Estimate: basePrice * 0.2, Percentage: 20},
			{Category: "Development", Estimate: basePrice * 0.5, Percentage: 50},
			{Category: "Testing & QA", Estimate: basePrice * 0.15, Percentage: 15},
			{Category: "Deployment & Support", Estimate: basePrice * 0.15, Percentage: 15},
		},
		Assumptions: []string{
			"Project requirements are well-defined and stable",
			"Client provides timely feedback and approvals",
			"Standard tech stack with no unusual constraints",
			"Team has relevant experience with the chosen technology",
		},
		Risks: []RiskFactor{
			{Factor: "Scope Creep", Impact: "Could increase cost by 20-40%", Probability: "Medium"},
			{Factor: "Technology Risk", Impact: "Unfamiliar tech may slow development", Probability: "Low"},
			{Factor: "Timeline Pressure", Impact: "Tight deadlines may require overtime", Probability: "Medium"},
		},
		Explanation: "Price based on complexity, timeline, and industry benchmarks.",
	}
}
