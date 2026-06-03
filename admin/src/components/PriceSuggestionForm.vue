<template>
  <div class="price-suggestion-form">
    <h2>AI Project Price Suggestion</h2>
    <form @submit.prevent="submitEvaluation">
      <div class="form-group">
        <label for="description">Project Description *</label>
        <textarea id="description" v-model="form.description" required></textarea>
      </div>
      <div class="form-group">
        <label for="requirements">Requirements *</label>
        <textarea id="requirements" v-model="form.requirements" required></textarea>
      </div>
      <div class="form-group">
        <label for="deliverables">Deliverables *</label>
        <textarea id="deliverables" v-model="form.deliverables" required></textarea>
      </div>
      <div class="form-group">
        <label for="timeline">Timeline *</label>
        <input id="timeline" v-model="form.timeline" required />
      </div>
      <div class="form-group">
        <label for="techStack">Tech Stack *</label>
        <input id="techStack" v-model="form.techStack" required />
      </div>
      <div class="form-group">
        <label for="complexity">Complexity *</label>
        <select id="complexity" v-model="form.complexity" required>
          <option value="low">Low</option>
          <option value="medium">Medium</option>
          <option value="high">High</option>
        </select>
      </div>
      <div class="form-group">
        <label for="constraints">Constraints *</label>
        <textarea id="constraints" v-model="form.constraints" required></textarea>
      </div>
      <div class="form-group">
        <label for="referenceBudget">Reference Budget (optional)</label>
        <input id="referenceBudget" v-model.number="form.referenceBudget" type="number" min="0" />
      </div>
      <button type="submit" :disabled="loading">
        {{ loading ? 'Evaluating...' : 'Evaluate Price' }}
      </button>
    </form>

    <div v-if="error" class="error">{{ error }}</div>

    <div v-if="result" class="result">
      <h3>Suggested Price: ${{ result.suggestedPrice }}</h3>
      <p v-if="result.suggestedPriceRange">
        Range: ${{ result.suggestedPriceRange.min }} – ${{ result.suggestedPriceRange.max }}
      </p>
      <p>Confidence: {{ result.confidenceLevel }}</p>

      <div v-if="result.breakdown && result.breakdown.length">
        <h4>Breakdown</h4>
        <ul>
          <li v-for="item in result.breakdown" :key="item.category">
            <strong>{{ item.category }}</strong>: ${{ item.amount }} – {{ item.description }}
          </li>
        </ul>
      </div>

      <div v-if="result.assumptions && result.assumptions.length">
        <h4>Assumptions</h4>
        <ul>
          <li v-for="(assumption, index) in result.assumptions" :key="index">{{ assumption }}</li>
        </ul>
      </div>

      <div v-if="result.risks && result.risks.length">
        <h4>Risks</h4>
        <ul>
          <li v-for="(risk, index) in result.risks" :key="index">{{ risk }}</li>
        </ul>
      </div>

      <div class="actions">
        <label for="finalPrice">Final Price (edit if needed):</label>
        <input id="finalPrice" v-model.number="finalPrice" type="number" min="0" />
        <button @click="acceptPrice">Accept & Save</button>
      </div>
    </div>
  </div>
</template>

<script>
export default {
  name: 'PriceSuggestionForm',
  data() {
    return {
      form: {
        description: '',
        requirements: '',
        deliverables: '',
        timeline: '',
        techStack: '',
        complexity: 'medium',
        constraints: '',
        referenceBudget: null,
      },
      loading: false,
      error: null,
      result: null,
      finalPrice: null,
    };
  },
  methods: {
    async submitEvaluation() {
      this.loading = true;
      this.error = null;
      this.result = null;

      try {
        const response = await fetch('/api/price-suggestion', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(this.form),
        });

        if (!response.ok) {
          const errData = await response.json();
          throw new Error(errData.error || 'Failed to get suggestion');
        }

        const data = await response.json();
        this.result = data;
        this.finalPrice = data.suggestedPrice;
      } catch (err) {
        this.error = err.message;
      } finally {
        this.loading = false;
      }
    },
    acceptPrice() {
      // Emit event with the final price for parent to handle
      this.$emit('price-accepted', this.finalPrice);
    },
  },
};
</script>

<style scoped>
.price-suggestion-form {
  max-width: 600px;
  margin: 0 auto;
  padding: 20px;
}
.form-group {
  margin-bottom: 15px;
}
label {
  display: block;
  margin-bottom: 5px;
  font-weight: bold;
}
textarea, input, select {
  width: 100%;
  padding: 8px;
  box-sizing: border-box;
}
button {
  padding: 10px 20px;
  background-color: #007bff;
  color: white;
  border: none;
  cursor: pointer;
}
button:disabled {
  background-color: #cccccc;
}
.error {
  color: red;
  margin-top: 10px;
}
.result {
  margin-top: 20px;
  border-top: 1px solid #ccc;
  padding-top: 20px;
}
.actions {
  margin-top: 20px;
}
</style>
