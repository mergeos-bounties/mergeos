package core

import (
	"errors"
	"math"
	"strings"
)

type priceFactor struct {
	Name        string
	AmountCents int64
	Reason      string
}

func EvaluateProjectPrice(req ProjectPriceEvaluationRequest) (*ProjectPriceEvaluationResponse, error) {
	description := compactText(req.Description)
	if description == "" {
		description = compactText(req.Requirements)
	}
	if description == "" {
		return nil, errValidation("project description is required")
	}

	projectType := compactText(req.ProjectType)
	techStack := compactText(req.TechStack)
	complexity := strings.ToLower(compactText(req.Complexity))
	timeline := strings.ToLower(compactText(req.Timeline))
	requirements := compactText(req.Requirements)
	constraints := compactText(req.Constraints)
	deliverables := cleanStrings(req.Deliverables)

	base := int64(180000)
	if projectType != "" {
		switch {
		case strings.Contains(strings.ToLower(projectType), "mobile"):
			base = 260000
		case strings.Contains(strings.ToLower(projectType), "ai"), strings.Contains(strings.ToLower(projectType), "ml"):
			base = 320000
		case strings.Contains(strings.ToLower(projectType), "contract"), strings.Contains(strings.ToLower(projectType), "web3"):
			base = 300000
		case strings.Contains(strings.ToLower(projectType), "web"):
			base = 220000
		}
	}

	factors := []priceFactor{{Name: "Base scope", AmountCents: base, Reason: "Core planning, implementation, review, and delivery work."}}
	if len(deliverables) > 0 {
		amount := int64(len(deliverables)) * 45000
		factors = append(factors, priceFactor{Name: "Deliverables", AmountCents: amount, Reason: "Each named deliverable adds implementation and acceptance work."})
	}
	if techStack != "" {
		stackParts := splitCSVish(techStack)
		amount := int64(len(stackParts)) * 25000
		if amount > 150000 {
			amount = 150000
		}
		factors = append(factors, priceFactor{Name: "Technical surface", AmountCents: amount, Reason: "Multiple technologies increase integration and testing effort."})
	}
	if len(requirements) > 220 {
		factors = append(factors, priceFactor{Name: "Requirement detail", AmountCents: int64(math.Min(180000, float64(len(requirements)/120)*30000)), Reason: "Longer requirement sets usually imply more cases and constraints."})
	}
	if constraints != "" {
		factors = append(factors, priceFactor{Name: "Constraints", AmountCents: 60000, Reason: "Explicit constraints add delivery risk and review overhead."})
	}
	if strings.Contains(timeline, "urgent") || strings.Contains(timeline, "asap") || strings.Contains(timeline, "week") {
		factors = append(factors, priceFactor{Name: "Timeline pressure", AmountCents: 90000, Reason: "Short timelines require more coordination and execution buffer."})
	}
	switch complexity {
	case "high", "advanced", "complex":
		factors = append(factors, priceFactor{Name: "Complexity", AmountCents: 160000, Reason: "High complexity requires deeper design, tests, and risk controls."})
	case "medium", "moderate":
		factors = append(factors, priceFactor{Name: "Complexity", AmountCents: 70000, Reason: "Moderate complexity adds implementation and validation work."})
	}

	total := int64(0)
	breakdown := make([]PriceBreakdownItem, 0, len(factors))
	for _, factor := range factors {
		total += factor.AmountCents
		breakdown = append(breakdown, PriceBreakdownItem{Category: factor.Name, AmountCents: factor.AmountCents, Reason: factor.Reason})
	}

	if req.ReferenceBudgetCents > 0 {
		weighted := int64(math.Round(float64(total)*0.7 + float64(req.ReferenceBudgetCents)*0.3))
		breakdown = append(breakdown, PriceBreakdownItem{Category: "Reference budget calibration", AmountCents: weighted - total, Reason: "Blends the estimate toward the user's reference budget without replacing scope-based pricing."})
		total = weighted
	}

	if total < 10000 {
		total = 10000
	}
	low := roundToNearestCents(int64(math.Round(float64(total)*0.85)), 5000)
	high := roundToNearestCents(int64(math.Round(float64(total)*1.2)), 5000)
	suggested := roundToNearestCents(total, 5000)

	confidence := "medium"
	if len(deliverables) >= 3 && len(requirements) > 120 && techStack != "" {
		confidence = "high"
	} else if len(deliverables) == 0 || len(description) < 80 {
		confidence = "low"
	}

	return &ProjectPriceEvaluationResponse{
		ProtocolVersion:     "mergeos.estimate.v1",
		Kind:                "project_estimate",
		SuggestedPriceCents: suggested,
		SuggestedRange:      PriceRange{LowCents: low, HighCents: high},
		Confidence:          confidence,
		Breakdown:           breakdown,
		Assumptions:         priceAssumptions(projectType, deliverables, techStack),
		Risks:               priceRisks(req, confidence),
		Editable:            true,
	}, nil
}

func errValidation(message string) error { return errors.New(message) }

func compactText(value string) string {
	return strings.Join(strings.Fields(value), " ")
}

func cleanStrings(values []string) []string {
	items := make([]string, 0, len(values))
	for _, value := range values {
		clean := compactText(value)
		if clean != "" {
			items = append(items, clean)
		}
	}
	return items
}

func splitCSVish(value string) []string {
	fields := strings.FieldsFunc(value, func(r rune) bool {
		return r == ',' || r == ';' || r == '/' || r == '|'
	})
	return cleanStrings(fields)
}

func roundToNearestCents(value int64, nearest int64) int64 {
	if nearest <= 0 {
		return value
	}
	return ((value + nearest/2) / nearest) * nearest
}

func priceAssumptions(projectType string, deliverables []string, techStack string) []string {
	assumptions := []string{
		"Estimate assumes one production-ready implementation pass plus review and QA.",
		"Final funding can be edited before the project is published.",
	}
	if projectType != "" {
		assumptions = append(assumptions, "Project type is treated as "+projectType+".")
	}
	if len(deliverables) > 0 {
		assumptions = append(assumptions, "Named deliverables are independently reviewable milestones.")
	}
	if techStack != "" {
		assumptions = append(assumptions, "The listed tech stack is required, not merely preferred.")
	}
	return assumptions
}

func priceRisks(req ProjectPriceEvaluationRequest, confidence string) []string {
	risks := []string{}
	if confidence == "low" {
		risks = append(risks, "Scope is light on detail; add deliverables and constraints before funding.")
	}
	if req.Constraints != "" {
		risks = append(risks, "Hard constraints may change cost if they conflict with the implementation plan.")
	}
	if strings.Contains(strings.ToLower(req.Timeline), "urgent") || strings.Contains(strings.ToLower(req.Timeline), "asap") {
		risks = append(risks, "Urgent timelines may require a higher reward pool to attract qualified contributors.")
	}
	if len(risks) == 0 {
		risks = append(risks, "Major scope changes after publishing can move the price range.")
	}
	return risks
}
