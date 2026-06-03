const express = require('express');
const router = express.Router();
const { suggestPrice } = require('../ai/priceSuggestion');

/**
 * POST /api/price-suggestion
 * Body: { description, requirements, deliverables, timeline, techStack, complexity, constraints, referenceBudget? }
 */
router.post('/', async (req, res) => {
  try {
    const {
      description,
      requirements,
      deliverables,
      timeline,
      techStack,
      complexity,
      constraints,
      referenceBudget,
    } = req.body;

    // Validate required fields
    if (!description || !requirements || !deliverables || !timeline || !techStack || !complexity || !constraints) {
      return res.status(400).json({ error: 'Missing required fields' });
    }

    const projectInput = {
      description,
      requirements,
      deliverables,
      timeline,
      techStack,
      complexity,
      constraints,
      referenceBudget: referenceBudget ? Number(referenceBudget) : undefined,
    };

    const suggestion = await suggestPrice(projectInput);
    res.json(suggestion);
  } catch (error) {
    console.error('Price suggestion error:', error);
    res.status(500).json({ error: 'Failed to generate price suggestion' });
  }
});

module.exports = router;
