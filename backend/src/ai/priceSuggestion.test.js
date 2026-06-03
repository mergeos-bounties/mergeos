const { suggestPrice } = require('./priceSuggestion');

jest.mock('openai', () => ({
  Configuration: jest.fn(() => ({})),
  OpenAIApi: jest.fn(() => ({
    createChatCompletion: jest.fn(),
  })),
}));

const { OpenAIApi } = require('openai');

describe('suggestPrice', () => {
  const mockProjectInput = {
    description: 'A web application for task management',
    requirements: 'User authentication, task CRUD, real-time updates',
    deliverables: 'Source code, deployment guide, API documentation',
    timeline: '3 months',
    techStack: 'React, Node.js, PostgreSQL',
    complexity: 'medium',
    constraints: 'Must be GDPR compliant',
    referenceBudget: 50000,
  };

  beforeEach(() => {
    jest.clearAllMocks();
  });

  it('should return a valid price suggestion on successful API call', async () => {
    const mockResponse = {
      data: {
        choices: [
          {
            message: {
              content: JSON.stringify({
                suggestedPrice: 45000,
                suggestedPriceRange: { min: 40000, max: 50000 },
                confidenceLevel: 'high',
                breakdown: [
                  { category: 'Frontend', amount: 15000, description: 'React UI development' },
                  { category: 'Backend', amount: 20000, description: 'Node.js API and database' },
                  { category: 'DevOps', amount: 10000, description: 'Deployment and CI/CD' },
                ],
                assumptions: ['Standard hourly rates apply', 'No major scope changes'],
                risks: ['Potential delays due to third-party integrations'],
              }),
            },
          },
        ],
      },
    };

    OpenAIApi.prototype.createChatCompletion.mockResolvedValue(mockResponse);

    const result = await suggestPrice(mockProjectInput);

    expect(result).toHaveProperty('suggestedPrice');
    expect(result).toHaveProperty('suggestedPriceRange');
    expect(result).toHaveProperty('confidenceLevel');
    expect(result).toHaveProperty('breakdown');
    expect(result).toHaveProperty('assumptions');
    expect(result).toHaveProperty('risks');
    expect(result.suggestedPrice).toBe(45000);
    expect(result.confidenceLevel).toBe('high');
  });

  it('should throw an error when API call fails', async () => {
    OpenAIApi.prototype.createChatCompletion.mockRejectedValue(new Error('API error'));

    await expect(suggestPrice(mockProjectInput)).rejects.toThrow('API error');
  });

  it('should throw an error when response is invalid JSON', async () => {
    const mockResponse = {
      data: {
        choices: [
          {
            message: {
              content: 'Invalid response',
            },
          },
        ],
      },
    };

    OpenAIApi.prototype.createChatCompletion.mockResolvedValue(mockResponse);

    await expect(suggestPrice(mockProjectInput)).rejects.toThrow('Failed to parse AI response');
  });

  it('should throw an error when response structure is incomplete', async () => {
    const mockResponse = {
      data: {
        choices: [
          {
            message: {
              content: JSON.stringify({ suggestedPrice: 100 }),
            },
          },
        ],
      },
    };

    OpenAIApi.prototype.createChatCompletion.mockResolvedValue(mockResponse);

    await expect(suggestPrice(mockProjectInput)).rejects.toThrow('Failed to parse AI response');
  });
});
