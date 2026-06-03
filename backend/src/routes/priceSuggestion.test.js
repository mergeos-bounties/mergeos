const request = require('supertest');
const express = require('express');
const priceSuggestionRouter = require('./priceSuggestion');

jest.mock('../ai/priceSuggestion', () => ({
  suggestPrice: jest.fn(),
}));

const { suggestPrice } = require('../ai/priceSuggestion');

const app = express();
app.use(express.json());
app.use('/api/price-suggestion', priceSuggestionRouter);

describe('POST /api/price-suggestion', () => {
  const validBody = {
    description: 'A web app',
    requirements: 'Auth, CRUD',
    deliverables: 'Code, docs',
    timeline: '2 months',
    techStack: 'React, Node',
    complexity: 'low',
    constraints: 'None',
  };

  it('should return 400 if required fields are missing', async () => {
    const res = await request(app).post('/api/price-suggestion').send({});
    expect(res.status).toBe(400);
    expect(res.body).toHaveProperty('error');
  });

  it('should return 200 with suggestion on valid input', async () => {
    const mockSuggestion = {
      suggestedPrice: 20000,
      suggestedPriceRange: { min: 15000, max: 25000 },
      confidenceLevel: 'medium',
      breakdown: [],
      assumptions: ['Standard rates'],
      risks: ['Scope creep'],
    };
    suggestPrice.mockResolvedValue(mockSuggestion);

    const res = await request(app).post('/api/price-suggestion').send(validBody);
    expect(res.status).toBe(200);
    expect(res.body).toEqual(mockSuggestion);
  });

  it('should return 500 if AI service fails', async () => {
    suggestPrice.mockRejectedValue(new Error('AI error'));

    const res = await request(app).post('/api/price-suggestion').send(validBody);
    expect(res.status).toBe(500);
    expect(res.body).toHaveProperty('error');
  });
});
