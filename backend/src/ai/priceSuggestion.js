const { Configuration, OpenAIApi } = require('openai');

const configuration = new Configuration({
  apiKey: process.env.OPENAI_API_KEY,
});
const openai = new OpenAIApi(configuration);

/**
 * Analyze project details and suggest a price.
 * @param {Object} projectInput - Project information from the user.
 * @param {string} projectInput.description - Project description.
 * @param {string} projectInput.requirements - Detailed requirements.
 * @param {string} projectInput.deliverables - Expected deliverables.
 * @param {string} projectInput.timeline - Expected timeline.
 * @param {string} projectInput.techStack - Technologies to be used.
 * @param {string} projectInput.complexity - Complexity level (low/medium/high).
 * @param {string} projectInput.constraints - Any constraints.
 * @param {number} [projectInput.referenceBudget] - Optional reference budget.
 * @returns {Promise<Object>} Structured price suggestion.
 */
async function suggestPrice(projectInput) {
  const prompt = buildPrompt(projectInput);

  const completion = await openai.createChatCompletion({
    model: 'gpt-4',
    messages: [
      {
        role: 'system',
        content: 'You are an AI assistant that evaluates software project details and suggests a reasonable price. Respond with a JSON object only, no extra text.',
      },
      { role: 'user', content: prompt },
    ],
    temperature: 0.3,
    max_tokens: 1000,
  });

  const responseText = completion.data.choices[0].message.content.trim();
  return parseResponse(responseText);
}

function buildPrompt(input) {
  return `
Evaluate the following software project and suggest a price or price range.

Project Description:
${input.description}

Requirements:
${input.requirements}

Deliverables:
${input.deliverables}

Timeline:
${input.timeline}

Tech Stack:
${input.techStack}

Complexity:
${input.complexity}

Constraints:
${input.constraints}

${input.referenceBudget ? `Reference Budget: $${input.referenceBudget}` : ''}

Return a JSON object with the following structure:
{
  "suggestedPrice": number,
  "suggestedPriceRange": { "min": number, "max": number },
  "confidenceLevel": "low" | "medium" | "high",
  "breakdown": [
    { "category": string, "amount": number, "description": string }
  ],
  "assumptions": [string],
  "risks": [string]
}
`;
}

function parseResponse(text) {
  try {
    const parsed = JSON.parse(text);
    // Validate required fields
    if (
      typeof parsed.suggestedPrice !== 'number' ||
      !parsed.suggestedPriceRange ||
      typeof parsed.suggestedPriceRange.min !== 'number' ||
      typeof parsed.suggestedPriceRange.max !== 'number' ||
      !['low', 'medium', 'high'].includes(parsed.confidenceLevel) ||
      !Array.isArray(parsed.breakdown) ||
      !Array.isArray(parsed.assumptions) ||
      !Array.isArray(parsed.risks)
    ) {
      throw new Error('Invalid response structure');
    }
    return parsed;
  } catch (error) {
    throw new Error(`Failed to parse AI response: ${error.message}`);
  }
}

module.exports = { suggestPrice };
