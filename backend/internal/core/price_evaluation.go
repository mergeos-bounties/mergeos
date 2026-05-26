package core

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"
	"time"
)

type LLMProvider string
const (
	LLMOpenAI  LLMProvider = "openai"
	LLMAnthropic LLMProvider = "anthropic"
)

type priceFactor struct {
	Name        string
	AmountCents int64
	Reason      string
}

// aiPriceAnalysis calls an LLM to analyze project details and produce a structured price estimate.
func (s *Server) aiPriceAnalysis(req ProjectPriceEvaluationRequest) (*ProjectPriceEvaluationResponse, error) {
	apiKey := s.cfg.LLMApiKey
	model := s.cfg.LLMModel
	provider := LLMProvider(s.cfg.LLMProvider)

	if apiKey == "" || model == "" {
		return nil, errors.New("LLM not configured: set LLM_API_KEY, LLM_MODEL, LLM_PROVIDER")
	}

	prompt := buildLLMPrompt(req)

	var llmResp *LLMResponse
	var err error

	switch provider {
	case LLMOpenAI:
		llmResp, err = callOpenAI(apiKey, model, prompt)
	case LLMAnthropic:
		llmResp, err = callAnthropic(apiKey, model, prompt)
	default:
		return nil, fmt.Errorf("unsupported LLM provider: %s", provider)
	}

	if err != nil {
		return nil, fmt.Errorf("LLM call failed: %w", err)
	}

	suggested := maxInt64(llmResp.SuggestedPriceCents, 10000)
	low := maxInt64(roundToNearestCents(int64(float64(suggested)*0.85), 5000), 5000)
	high := maxInt64(roundToNearestCents(int64(float64(suggested)*1.2), 5000), suggested+5000)

	if req.ReferenceBudgetCents > 0 {
		weighted := int64(math.Round(float64(suggested)*0.7 + float64(req.ReferenceBudgetCents)*0.3))
		weighted = maxInt64(weighted, 10000)
		suggested = weighted
	}

	return &ProjectPriceEvaluationResponse{
		SuggestedPriceCents: suggested,
		SuggestedRange:      PriceRange{LowCents: low, HighCents: high},
		Confidence:          llmResp.Confidence,
		Breakdown: []PriceBreakdownItem{
			{Category: "AI analysis", AmountCents: suggested, Reason: llmResp.Reasoning},
		},
		Assumptions: llmResp.Assumptions,
		Risks:       llmResp.Risks,
		Editable:    true,
	}, nil
}

func callOpenAI(apiKey, model, prompt string) (*LLMResponse, error) {
	body := map[string]interface{}{
		"model": model,
		"messages": []map[string]string{
			{"role": "system", "content": "You are a JSON-only pricing expert. Respond only with valid JSON."},
			{"role": "user", "content": prompt},
		},
		"temperature": 0.3,
		"response_format": map[string]string{"type": "json_object"},
	}
	payload, _ := json.Marshal(body)

	req, _ := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewReader(payload))
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("OpenAI API error %d: %s", resp.StatusCode, string(respBody))
	}

	var openAIResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(respBody, &openAIResp); err != nil {
		return nil, err
	}
	if len(openAIResp.Choices) == 0 {
		return nil, errors.New("no response from OpenAI")
	}

	var result LLMResponse
	if err := json.Unmarshal([]byte(openAIResp.Choices[0].Message.Content), &result); err != nil {
		return nil, fmt.Errorf("parse OpenAI response: %w", err)
	}
	return &result, nil
}

func callAnthropic(apiKey, model, prompt string) (*LLMResponse, error) {
	body := map[string]interface{}{
		"model":      model,
		"max_tokens": 1024,
		"messages":   []map[string]string{{"role": "user", "content": prompt}},
	}
	payload, _ := json.Marshal(body)

	req, _ := http.NewRequest("POST", "https://api.anthropic.com/v1/messages", bytes.NewReader(payload))
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("Anthropic API error %d: %s", resp.StatusCode, string(respBody))
	}

	var anthropicResp struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(respBody, &anthropicResp); err != nil {
		return nil, err
	}
	if len(anthropicResp.Content) == 0 {
		return nil, errors.New("no response from Anthropic")
	}

	var result LLMResponse
	if err := json.Unmarshal([]byte(anthropicResp.Content[0].Text), &result); err != nil {
		return nil, fmt.Errorf("parse Anthropic response: %w", err)
	}
	return &result, nil
}

func buildLLMPrompt(req ProjectPriceEvaluationRequest) string {
	return fmt.Sprintf("You are a software project pricing expert. Analyze the following project and return a JSON estimate.\n\nProject: %s\nDescription: %s\nType: %s\nTech Stack: %s\nTimeline: %s\nComplexity: %s\nDeliverables: %s\nConstraints: %s\n\nReturn JSON with fields:\n- suggested_price_cents (int, minimum 10000)\n- low_cents (int, 80-90%%%% of suggested)\n- high_cents (int, 110-130%%%% of suggested)\n- confidence (string: "low", "medium", "high")\n- reasoning (string, brief explanation)\n- risks (array of strings)\n- assumptions (array of strings)",
		req.Title, req.Description, req.ProjectType, req.TechStack,
		req.Timeline, req.Complexity, strings.Join(req.Deliverables, ", "), req.Constraints)
}

func maxInt64(a, b int64) int64 {
	if a > b { return a }
	return b
}