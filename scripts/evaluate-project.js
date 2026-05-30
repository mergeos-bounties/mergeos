#!/usr/bin/env node
// AI Project Price Evaluation for MergeOS
// Estimates fair MRG token pricing based on complexity

const COMPLEXITY_WEIGHTS = {
  frontend: 0.3,
  backend: 0.4,
  fullstack: 0.6,
  smart_contract: 0.8,
  security: 1.0,
  documentation: 0.2,
};

const URGENCY_MULTIPLIER = { low: 1, medium: 1.5, high: 2.5 };

function evaluateProject(project) {
  const { type, lines, urgency, testCoverage } = project;
  const baseScore = (lines || 100) * (COMPLEXITY_WEIGHTS[type] || 0.3);
  const urgencyFactor = URGENCY_MULTIPLIER[urgency] || 1;
  const coverageBonus = (testCoverage || 0) * 0.5;
  const mrgEstimate = Math.round((baseScore * urgencyFactor + coverageBonus) * 10) / 10;

  return {
    estimatedMRG: mrgEstimate,
    confidence: type === "security" ? "high" : testCoverage > 0.5 ? "medium" : "low",
    breakdown: { baseScore, urgencyFactor, coverageBonus },
  };
}

module.exports = { evaluateProject, COMPLEXITY_WEIGHTS, URGENCY_MULTIPLIER };
