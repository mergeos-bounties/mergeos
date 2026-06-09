import React, { useState } from 'react';

// AI project price evaluation component
// Issue #3: AI project evaluation for price suggestion

const API_ENDPOINT = '/api/ai/evaluate-project';

function ProjectEvaluator() {
  const [form, setForm] = useState({
    description: '',
    requirements: '',
    deliverables: '',
    timeline: '',
    techStack: '',
    complexity: 'medium',
    constraints: '',
    referenceBudget: '',
  });
  const [result, setResult] = useState(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState(null);
  const [editing, setEditing] = useState(false);
  const [editedPrice, setEditedPrice] = useState('');

  const handleChange = (e) => {
    setForm({ ...form, [e.target.name]: e.target.value });
  };

  const validateForm = () => {
    if (!form.description.trim()) return 'Project description is required';
    if (!form.timeline.trim()) return 'Timeline is required';
    return null;
  };

  const evaluateProject = async () => {
    const validationError = validateForm();
    if (validationError) {
      setError(validationError);
      return;
    }

    setLoading(true);
    setError(null);
    setResult(null);

    try {
      const response = await fetch(API_ENDPOINT, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(form),
      });

      if (!response.ok) {
        const errData = await response.json().catch(() => ({}));
        throw new Error(errData.message || `API error: ${response.status}`);
      }

      const data = await response.json();
      setResult(data);
      setEditedPrice(data.suggestedPrice?.toString() || '');
    } catch (err) {
      setError(err.message || 'Failed to evaluate project. Please try again.');
    } finally {
      setLoading(false);
    }
  };

  const handleAcceptPrice = () => {
    // User accepted the price — trigger callback
    if (onPriceAccepted) {
      onPriceAccepted({
        originalPrice: result.suggestedPrice,
        acceptedPrice: parseFloat(editedPrice),
        breakdown: result.breakdown,
        assumptions: result.assumptions,
      });
    }
    setEditing(false);
  };

  return (
    <div className="project-evaluator">
      <h2>AI Project Price Evaluator</h2>
      <p>Enter your project details below for an AI-powered price suggestion.</p>

      <div className="form-grid">
        <div className="form-group full-width">
          <label>Project Description *</label>
          <textarea
            name="description"
            value={form.description}
            onChange={handleChange}
            placeholder="Describe your project in detail..."
            rows={4}
          />
        </div>

        <div className="form-group">
          <label>Requirements</label>
          <textarea
            name="requirements"
            value={form.requirements}
            onChange={handleChange}
            placeholder="List key requirements..."
            rows={3}
          />
        </div>

        <div className="form-group">
          <label>Deliverables</label>
          <textarea
            name="deliverables"
            value={form.deliverables}
            onChange={handleChange}
            placeholder="What will be delivered?"
            rows={3}
          />
        </div>

        <div className="form-group">
          <label>Timeline *</label>
          <input
            type="text"
            name="timeline"
            value={form.timeline}
            onChange={handleChange}
            placeholder="e.g., 3 months, Q2 2026"
          />
        </div>

        <div className="form-group">
          <label>Tech Stack</label>
          <input
            type="text"
            name="techStack"
            value={form.techStack}
            onChange={handleChange}
            placeholder="e.g., React, Node.js, PostgreSQL"
          />
        </div>

        <div className="form-group">
          <label>Complexity</label>
          <select name="complexity" value={form.complexity} onChange={handleChange}>
            <option value="low">Low — Simple CRUD / basic UI</option>
            <option value="medium">Medium — Multiple integrations</option>
            <option value="high">High — Complex architecture</option>
            <option value="very-high">Very High — Distributed systems / AI</option>
          </select>
        </div>

        <div className="form-group">
          <label>Constraints</label>
          <textarea
            name="constraints"
            value={form.constraints}
            onChange={handleChange}
            placeholder="Budget limits, deadlines, team size..."
            rows={2}
          />
        </div>

        <div className="form-group">
          <label>Reference Budget ($)</label>
          <input
            type="number"
            name="referenceBudget"
            value={form.referenceBudget}
            onChange={handleChange}
            placeholder="Optional: your budget range"
            min="0"
          />
        </div>
      </div>

      <button
        className="evaluate-btn"
        onClick={evaluateProject}
        disabled={loading}
      >
        {loading ? '🔄 Evaluating...' : '🤖 Evaluate Project'}
      </button>

      {/* Error State */}
      {error && (
        <div className="error-card">
          <p>⚠️ {error}</p>
          <button onClick={evaluateProject}>🔄 Retry</button>
        </div>
      )}

      {/* Loading State */}
      {loading && (
        <div className="loading-card">
          <div className="spinner" />
          <p>Analyzing project scope, complexity, and risks...</p>
        </div>
      )}

      {/* Result State */}
      {result && !loading && (
        <div className="result-card">
          <div className="price-section">
            <h3>Suggested Price</h3>
            {editing ? (
              <div className="price-edit">
                <input
                  type="number"
                  value={editedPrice}
                  onChange={(e) => setEditedPrice(e.target.value)}
                />
                <button onClick={handleAcceptPrice}>Confirm</button>
                <button onClick={() => setEditing(false)}>Cancel</button>
              </div>
            ) : (
              <div className="price-display">
                <span className="price-amount">${result.suggestedPrice?.toLocaleString()}</span>
                <span className="confidence">Confidence: {result.confidence}%</span>
                <button onClick={() => setEditing(true)}>✏️ Edit</button>
              </div>
            )}
          </div>

          <div className="breakdown-section">
            <h4>Breakdown by Category</h4>
            <table>
              <thead>
                <tr>
                  <th>Category</th>
                  <th>Estimate</th>
                  <th>% of Total</th>
                </tr>
              </thead>
              <tbody>
                {result.breakdown?.map((item, i) => (
                  <tr key={i}>
                    <td>{item.category}</td>
                    <td>${item.estimate?.toLocaleString()}</td>
                    <td>{item.percentage}%</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>

          <div className="assumptions-section">
            <h4>Assumptions</h4>
            <ul>
              {result.assumptions?.map((a, i) => (
                <li key={i}>{a}</li>
              ))}
            </ul>
          </div>

          <div className="risks-section">
            <h4>Risk Factors</h4>
            <ul>
              {result.risks?.map((r, i) => (
                <li key={i}>
                  <strong>{r.factor}:</strong> {r.impact} (Probability: {r.probability})
                </li>
              ))}
            </ul>
          </div>

          <div className="actions">
            <button className="accept-btn" onClick={handleAcceptPrice}>
              ✅ Accept Price
            </button>
            <button className="reevaluate-btn" onClick={evaluateProject}>
              🔄 Re-evaluate
            </button>
          </div>
        </div>
      )}

      <style>{`
        .project-evaluator { max-width: 800px; margin: 0 auto; padding: 24px; }
        .form-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 16px; }
        .form-group.full-width { grid-column: 1 / -1; }
        .form-group { display: flex; flex-direction: column; }
        .form-group label { font-weight: 600; margin-bottom: 4px; }
        .error-card { background: #fff0f0; border: 1px solid #ffcccc; padding: 16px; border-radius: 8px; margin: 16px 0; }
        .loading-card { text-align: center; padding: 32px; }
        .result-card { background: #f8f9fa; border: 1px solid #dee2e6; border-radius: 8px; padding: 24px; margin: 16px 0; }
        .price-amount { font-size: 2em; font-weight: bold; color: #2d9cdb; }
        .evaluate-btn { background: #2d9cdb; color: white; padding: 12px 24px; border: none; border-radius: 6px; cursor: pointer; font-size: 16px; margin: 16px 0; }
        .evaluate-btn:disabled { opacity: 0.6; }
        table { width: 100%; border-collapse: collapse; }
        th, td { padding: 8px; text-align: left; border-bottom: 1px solid #dee2e6; }
        .accept-btn { background: #27ae60; color: white; padding: 8px 16px; border: none; border-radius: 4px; cursor: pointer; }
        .reevaluate-btn { background: #95a5a6; color: white; padding: 8px 16px; border: none; border-radius: 4px; cursor: pointer; margin-left: 8px; }
      `}</style>
    </div>
  );
}

export default ProjectEvaluator;
