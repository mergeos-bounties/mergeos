<template>
  <div v-if="projectWizardVisible" class="project-flow-shell">
    <div v-if="toastMessage" class="toast project-flow-toast">
      {{ toastMessage }}
    </div>

    <header class="project-flow-navbar">
      <a class="brand-link" href="/" @click.prevent="closeProjectWizard(); openPublicPage('home')">
        <span class="brand-mark" aria-hidden="true">
          <img src="/favicon.svg" alt="" />
        </span>
        <strong>MergeOS</strong>
      </a>

      <nav class="nav-links project-flow-nav" aria-label="Project setup navigation">
        <a href="/product" @click.prevent="closeProjectWizard(); openPublicPage('product')">
          Product
          <ChevronDown :size="13" />
        </a>
        <a href="/solutions" @click.prevent="closeProjectWizard(); openPublicPage('solutions')">
          Solutions
          <ChevronDown :size="13" />
        </a>
        <a href="/marketplace" @click.prevent="closeProjectWizard(); openPublicPage('marketplace')">Marketplace</a>
        <a href="/how-it-works" @click.prevent="closeProjectWizard(); openPublicPage('how-it-works')">How it works</a>
        <a href="/ledger" @click.prevent="closeProjectWizard(); openPublicPage('ledger')">Ledger Logs</a>
      </nav>

      <div class="nav-actions project-flow-actions">
        <button class="locale-button icon-only" type="button" aria-label="Language settings" @click="showToast('Language settings')">
          <Globe2 :size="17" />
          EN
          <ChevronDown :size="13" />
        </button>
        <template v-if="user">
          <button class="dash-icon-button light" aria-label="Messages" type="button" @click="showToast('Opening messages...')">
            <MessageCircle :size="17" />
          </button>
          <button class="dash-profile slim" type="button" @click="logout">
            <span class="profile-avatar">{{ initialsFor(user.name || user.email) }}</span>
            <span>{{ user.name || user.email || 'Signed-in user' }}</span>
            <ChevronDown :size="14" />
          </button>
        </template>
        <template v-else>
          <button class="secondary-button compact" type="button" @click="openAuthFromProjectWizard('login')">Log in</button>
        </template>
        <button class="primary-button compact" type="button" @click="restartProjectWizard">
          Start a project
          <ArrowRight :size="16" />
        </button>
      </div>
    </header>

    <main class="project-flow-main" :class="`stage-${projectWizardStage}`">
      <aside class="project-flow-sidebar">
        <button class="back-link" type="button" @click="closeProjectWizard">
          <ArrowLeft :size="15" />
          Back to home
        </button>

        <div class="project-flow-title">
          <h1>{{ projectWizardStage === 'success' ? 'Payment complete' : 'Start a project' }}</h1>
          <p>{{ wizardIntroCopy }}</p>
        </div>

        <nav class="project-step-list" aria-label="Project setup steps">
          <button
            v-for="step in projectSetupSteps"
            :key="step.number"
            :class="{ active: projectWizardStage === 'setup' && projectWizardStep === step.number, done: projectWizardStage !== 'setup' || projectWizardStep > step.number }"
            type="button"
            @click="goProjectStep(step.number)"
          >
            <span>
              <CheckCircle2 v-if="projectWizardStage !== 'setup' || projectWizardStep > step.number" :size="16" />
              <template v-else>{{ step.number }}</template>
            </span>
            <strong>{{ step.label }}</strong>
            <small>{{ step.description }}</small>
          </button>
        </nav>

        <article v-if="projectWizardStage === 'setup' && projectWizardStep === 1" class="wizard-help-card">
          <Sparkles :size="17" />
          <strong>Need help?</strong>
          <p>Our AI assistant can help you structure your project.</p>
          <button type="button" @click="showToast('AI assistant is preparing your brief...')">
            Use AI assistant
          </button>
        </article>

        <article v-else-if="projectWizardStage === 'setup' && projectWizardStep === 3" class="wizard-help-card accent">
          <Calculator :size="17" />
          <strong>AI Budget Estimator</strong>
          <p>Let AI analyze your requirements and suggest the right budget range.</p>
          <button type="button" :disabled="priceEvaluationBusy" @click="runProjectPriceEvaluation">
            {{ priceEvaluationBusy ? 'Estimating...' : 'Estimate with AI' }}
          </button>
        </article>

        <article v-else class="wizard-quality-card">
          <Sparkles :size="17" />
          <strong>AI Review</strong>
          <p>{{ projectQualityCopy }}</p>
          <div class="quality-score">
            <span>{{ projectQualityScoreLabel }}</span>
            <small>Quality score</small>
          </div>
        </article>
      </aside>

      <section class="project-flow-board">
        <article v-if="projectWizardStage === 'setup'" class="wizard-card project-step-panel">
          <header class="wizard-card-heading">
            <div>
              <span class="step-kicker">{{ projectWizardStep }}. {{ currentProjectStep.label }}</span>
              <h2>{{ currentProjectStep.title }}</h2>
              <p>{{ currentProjectStep.helper }}</p>
            </div>
            <button v-if="projectWizardStep === 4" class="secondary-button compact ghost" type="button" @click="showToast('Opening preview...')">
              <Eye :size="15" />
              Preview
            </button>
          </header>

          <div v-if="projectWizardStep === 1" class="wizard-form-grid">
            <label class="wizard-field full">
              <span>Project title <b>*</b></span>
              <input v-model.trim="projectSetupForm.title" placeholder="Enter a clear project title" />
            </label>

            <label class="wizard-field full">
              <span>Short description <b>*</b></span>
              <textarea
                v-model.trim="projectSetupForm.shortDescription"
                rows="5"
                maxlength="1000"
                placeholder="Describe your project, what you want to build, and the problem you're solving..."
              />
              <small>{{ projectSetupForm.shortDescription.length }} / 1000</small>
            </label>

            <section class="wizard-section full">
              <div class="wizard-section-title">
                <strong>Project type <b>*</b></strong>
                <small>What kind of work do you need?</small>
              </div>
              <div class="project-type-grid">
                <button
                  v-for="type in projectTypeOptions"
                  :key="type.label"
                  :class="{ selected: projectSetupForm.projectType === type.label }"
                  class="select-tile"
                  type="button"
                  @click="projectSetupForm.projectType = type.label"
                >
                  <component :is="type.icon" :size="18" />
                  <strong>{{ type.label }}</strong>
                  <small>{{ type.caption }}</small>
                  <CheckCircle2 v-if="projectSetupForm.projectType === type.label" class="tile-check" :size="16" />
                </button>
              </div>
            </section>

            <label class="wizard-field full">
              <span>Tech stack <small>(optional)</small></span>
              <input v-model.trim="projectSetupForm.techStack" placeholder="Add technologies or frameworks" />
            </label>

            <section class="wizard-section full attach-repo">
              <div class="attach-repo-head">
                <div>
                  <strong>Attach repository <small>(optional)</small></strong>
                  <p>Load open issues and turn them into scored fix tasks.</p>
                </div>
                <button class="secondary-button compact" :disabled="repoImportBusy" type="button" @click="loadRepoIssues">
                  <RefreshCw :size="15" />
                  {{ repoImportBusy ? 'Loading issues' : 'Load issues' }}
                </button>
              </div>
              <label class="wizard-field full repo-url-field">
                <span>GitHub repository</span>
                <input
                  v-model.trim="projectSetupForm.repoUrl"
                  placeholder="https://github.com/owner/repo"
                  @keyup.enter="loadRepoIssues"
                />
              </label>
              <p v-if="repoImportError" class="modal-error repo-import-error">{{ repoImportError }}</p>
              <div v-if="repoImportedIssues.length" class="repo-issue-panel">
                <div class="repo-issue-summary">
                  <strong>{{ repoImportResult.owner }}/{{ repoImportResult.name }}</strong>
                  <span>{{ repoImportedIssues.length }} issues · {{ formatMRGFromCents(repoImportedEstimateCents) }} scored</span>
                </div>
                <article v-for="issue in repoImportedIssues.slice(0, 4)" :key="issue.number" class="repo-issue-row">
                  <span>#{{ issue.number }}</span>
                  <div>
                    <strong>{{ issue.title }}</strong>
                    <small>Score {{ issue.score }} · {{ issue.complexity }} · {{ formatMRGFromCents(issue.estimated_cents) }}</small>
                  </div>
                </article>
              </div>
            </section>
          </div>

          <div v-else-if="projectWizardStep === 2" class="wizard-form-grid">
            <section class="wizard-section full">
              <div class="wizard-section-title">
                <strong>Project overview</strong>
                <small>Provide more details about your project goals and what you want to achieve.</small>
              </div>
              <div class="rich-editor">
                <div class="editor-toolbar" aria-label="Text formatting tools">
                  <button type="button" aria-label="Bold"><strong>B</strong></button>
                  <button type="button" aria-label="Italic"><em>I</em></button>
                  <button type="button" aria-label="Bulleted list"><ListTodo :size="15" /></button>
                  <button type="button" aria-label="Quote"><Quote :size="15" /></button>
                  <button type="button" aria-label="Link"><Link2 :size="15" /></button>
                </div>
                <textarea
                  v-model.trim="projectSetupForm.overview"
                  rows="7"
                  maxlength="6000"
                  placeholder="Describe in detail what you need built, the goals, key features, and any important context..."
                />
                <small>{{ projectSetupForm.overview.length }} / 6000</small>
              </div>
            </section>

            <section class="wizard-section full">
              <div class="wizard-section-title row">
                <div>
                  <strong>Key deliverables</strong>
                  <small>What are the main things you expect to receive?</small>
                </div>
                <button class="text-action" type="button" @click="addDeliverable">
                  <Plus :size="14" />
                  Add deliverable
                </button>
              </div>
              <div class="deliverable-list">
                <label v-for="(deliverable, index) in projectDeliverables" :key="index" class="deliverable-row">
                  <GripVertical :size="15" />
                  <input v-model.trim="projectDeliverables[index]" :placeholder="projectDeliverablePlaceholders[index] || 'Describe another deliverable'" />
                  <button type="button" :aria-label="`Remove deliverable ${index + 1}`" @click="removeDeliverable(index)">
                    <X :size="14" />
                  </button>
                </label>
              </div>
            </section>

            <label class="wizard-field split">
              <span>Project requirements</span>
              <textarea
                v-model.trim="projectSetupForm.requirements"
                rows="7"
                maxlength="2000"
                placeholder="Add constraints, quality bar, compliance, or integration requirements"
              />
              <small>{{ projectSetupForm.requirements.length }} / 2000</small>
            </label>

            <section class="wizard-section split reference-dropzone">
              <UploadCloud :size="24" />
              <strong>Drag & drop files here</strong>
              <button class="text-action" type="button" @click="showToast('File upload coming soon.')">browse your files</button>
              <small>Supports images, PDFs, docs, links up to 200MB.</small>
            </section>
          </div>

          <div v-else-if="projectWizardStep === 3" class="wizard-form-grid">
            <section class="wizard-section full budget-row">
              <label class="wizard-field compact-field">
                <span>Budget (MRG)</span>
                <div class="currency-input">
                  <span class="currency-chip">{{ tokenSymbol }}</span>
                  <input v-model.number="projectSetupForm.budgetAmount" placeholder="0" type="number" min="10000" step="1000" />
                </div>
              </label>

              <div class="wizard-section grow">
                <div class="wizard-section-title">
                  <strong>Budget type</strong>
                  <small>Choose how you want to set the budget.</small>
                </div>
                <div class="budget-type-grid">
                  <button
                    v-for="budgetType in budgetTypeOptions"
                    :key="budgetType.label"
                    :class="{ selected: projectSetupForm.budgetType === budgetType.label }"
                    class="select-tile horizontal"
                    type="button"
                    @click="projectSetupForm.budgetType = budgetType.label"
                  >
                    <component :is="budgetType.icon" :size="18" />
                    <span>{{ budgetType.label }}</span>
                    <CheckCircle2 v-if="projectSetupForm.budgetType === budgetType.label" class="tile-check" :size="15" />
                  </button>
                </div>
              </div>
            </section>

            <section class="wizard-section full ai-pricing-section">
              <div class="ai-pricing-card">
                <div class="ai-pricing-header">
                  <Sparkles class="sparkle-icon animated-sparkle" :size="18" />
                  <div>
                    <strong>AI Price Suggestion Engine</strong>
                    <small>Let our AI evaluate your scope, tech stack, and deliverables to suggest a fair budget range.</small>
                  </div>
                </div>

                <div class="ai-pricing-inputs">
                  <div class="wizard-field">
                    <span>Project Complexity</span>
                    <div class="complexity-selector">
                      <button
                        v-for="lvl in ['Low', 'Medium', 'High']"
                        :key="lvl"
                        :class="{ selected: projectSetupForm.complexity === lvl }"
                        type="button"
                        class="complexity-btn"
                        @click="projectSetupForm.complexity = lvl"
                      >
                        {{ lvl }}
                      </button>
                    </div>
                  </div>

                  <label class="wizard-field">
                    <span>Project Constraints & Compliance (Optional)</span>
                    <input
                      v-model="projectSetupForm.constraints"
                      type="text"
                      placeholder="Add compliance, delivery, or technical constraints"
                    />
                  </label>
                </div>

                <div class="ai-pricing-action">
                  <button
                    class="primary-button compact ai-evaluate-btn"
                    type="button"
                    :disabled="aiEvaluationLoading"
                    @click="triggerAiEvaluation"
                  >
                    <RefreshCw v-if="aiEvaluationLoading" class="loading-spin" :size="15" />
                    <Sparkles v-else :size="15" />
                    {{ aiEvaluationLoading ? 'Evaluating scope...' : 'Get AI price recommendation' }}
                  </button>
                </div>

                <div v-if="aiEvaluationError" class="ai-evaluation-error">
                  <Bug :size="16" />
                  <span>{{ aiEvaluationError }}</span>
                </div>

                <div v-if="aiEvaluationResult" class="ai-evaluation-results-box">
                  <div class="suggestion-hero">
                    <div class="hero-range">
                      <small>Suggested budget range</small>
                      <strong>{{ formatMRGFromUSD(aiEvaluationResult.suggested_low) }} - {{ formatMRGFromUSD(aiEvaluationResult.suggested_high) }}</strong>
                      <span class="confidence-badge">Confidence: {{ Math.round(aiEvaluationResult.confidence_level * 100) }}%</span>
                    </div>
                    <button
                      class="secondary-button compact apply-suggestion-btn"
                      type="button"
                      @click="applyAiSuggestedPrice"
                    >
                      <CheckCircle2 :size="14" />
                      Use Suggested Budget
                    </button>
                  </div>

                  <div class="results-details-grid">
                    <div class="results-col">
                      <h4>Task Breakdown</h4>
                      <ul class="breakdown-list">
                        <li v-for="(amount, task) in aiEvaluationResult.task_breakdown" :key="task">
                          <span class="task-name">{{ task }}</span>
                          <span class="task-price">{{ formatMRGFromUSD(amount) }}</span>
                        </li>
                      </ul>
                    </div>

                    <div class="results-col">
                      <h4>Rationale</h4>
                      <p class="rationale-text">{{ aiEvaluationResult.rationale }}</p>
                    </div>
                  </div>

                  <div class="results-details-grid extra-meta">
                    <div class="results-col">
                      <h4>Assumptions</h4>
                      <ul class="meta-bullets">
                        <li v-for="(assumption, i) in aiEvaluationResult.assumptions" :key="i">
                          {{ assumption }}
                        </li>
                      </ul>
                    </div>

                    <div class="results-col">
                      <h4>Identified Risks</h4>
                      <ul class="meta-bullets risks">
                        <li v-for="(risk, i) in aiEvaluationResult.risks" :key="i">
                          {{ risk }}
                        </li>
                      </ul>
                    </div>
                  </div>
                </div>
              </div>
            </section>

            <section v-if="priceEvaluation" class="wizard-section full price-estimate-card">
              <div class="wizard-section-title row">
                <div>
                  <strong>Suggested price</strong>
                  <small>{{ priceEvaluation.confidence }} confidence · editable before publishing</small>
                </div>
                <button class="text-action" type="button" @click="applyPriceEvaluation">
                  <CheckCircle2 :size="14" />
                  Use estimate
                </button>
              </div>
              <div class="price-estimate-summary">
                <strong>{{ formatMRGFromCents(priceEvaluation.suggested_price_cents) }}</strong>
                <span>{{ formatMRGFromCents(priceEvaluation.suggested_range.low_cents) }} - {{ formatMRGFromCents(priceEvaluation.suggested_range.high_cents) }}</span>
              </div>
              <div class="price-breakdown-grid">
                <article v-for="item in priceEvaluation.breakdown.slice(0, 4)" :key="item.category">
                  <strong>{{ item.category }}</strong>
                  <span>{{ formatMRGFromCents(item.amount_cents) }}</span>
                  <small>{{ item.reason }}</small>
                </article>
              </div>
              <p v-if="priceEvaluation.risks?.length">{{ priceEvaluation.risks[0] }}</p>
            </section>

            <p v-if="priceEvaluationError" class="project-payment-error full">{{ priceEvaluationError }}</p>

            <section class="wizard-section full timeline-box">
              <div class="wizard-section-title">
                <strong>Timeline</strong>
                <small>When should this project be completed?</small>
              </div>
              <div class="timeline-grid">
                <label class="wizard-field">
                  <span>Start date</span>
                  <input v-model="projectSetupForm.startDate" type="date" />
                </label>
                <label class="wizard-field">
                  <span>Deadline</span>
                  <input v-model="projectSetupForm.deadline" type="date" />
                </label>
                <div class="duration-ring">
                  <strong>{{ projectDurationDays || '--' }}</strong>
                  <small>days</small>
                </div>
              </div>
            </section>

            <section class="wizard-section full">
              <div class="wizard-section-title">
                <strong>Funding & payment</strong>
                <small>All payments are secured by escrow.</small>
              </div>
              <div class="payment-method-grid">
                <button
                  v-for="method in fundingMethodOptions"
                  :key="method.label"
                  :class="{ selected: projectSetupForm.fundingMethod === method.label }"
                  class="select-tile horizontal rich"
                  type="button"
                  @click="projectSetupForm.fundingMethod = method.label"
                >
                  <component :is="method.icon" :size="18" />
                  <span>
                    <strong>{{ method.label }}</strong>
                    <small>{{ method.caption }}</small>
                  </span>
                  <CheckCircle2 v-if="projectSetupForm.fundingMethod === method.label" class="tile-check" :size="15" />
                </button>
              </div>
            </section>

            <section class="wizard-section full settings-grid">
              <label class="wizard-field">
                <span>Project visibility</span>
                <select v-model="projectSetupForm.visibility">
                  <option>Public</option>
                  <option>Private</option>
                  <option>Invite only</option>
                </select>
              </label>
              <label class="wizard-field">
                <span>Allow AI agents</span>
                <select v-model="projectSetupForm.allowAgents">
                  <option :value="true">Yes, allow AI agents to work</option>
                  <option :value="false">No, human talent only</option>
                </select>
              </label>
              <label class="wizard-field">
                <span>Skills required</span>
                <input v-model.trim="projectSetupForm.skills" placeholder="Select skills" />
              </label>
            </section>
          </div>

          <div v-else class="review-grid">
            <section class="review-card wide">
              <button type="button" aria-label="Edit project information" @click="goProjectStep(1)">
                <PenLine :size="15" />
              </button>
              <h3>
                <FileCheck2 :size="17" />
                Project information
              </h3>
              <dl>
                <div>
                  <dt>Title</dt>
                  <dd>{{ projectTitleLabel }}</dd>
                </div>
                <div>
                  <dt>Type</dt>
                  <dd>{{ projectTypeLabel }}</dd>
                </div>
                <div>
                  <dt>Short description</dt>
                  <dd>{{ projectDescriptionLabel }}</dd>
                </div>
              </dl>
            </section>

            <section class="review-card">
              <button type="button" aria-label="Edit scope" @click="goProjectStep(2)">
                <PenLine :size="15" />
              </button>
              <h3>
                <ListTodo :size="17" />
                Scope & requirements
              </h3>
              <ul v-if="visibleDeliverables.length">
                <li v-for="deliverable in visibleDeliverables" :key="deliverable">
                  <CheckCircle2 :size="14" />
                  {{ deliverable }}
                </li>
              </ul>
              <p v-else>{{ projectDeliverablesPlaceholder }}</p>
            </section>

            <section class="review-card">
              <button type="button" aria-label="Edit budget" @click="goProjectStep(3)">
                <PenLine :size="15" />
              </button>
              <h3>
                <CreditCard :size="17" />
                Budget & timeline
              </h3>
              <dl>
                <div>
                  <dt>Budget range</dt>
                  <dd>{{ projectBudgetRangeLabel }}</dd>
                </div>
                <div>
                  <dt>Estimated total</dt>
                  <dd>{{ projectEstimatedTotalLabel }}</dd>
                </div>
                <div>
                  <dt>Timeline</dt>
                  <dd>{{ projectTimelineLabel }}</dd>
                </div>
              </dl>
            </section>

            <section class="review-card escrow-review">
              <h3>
                <ShieldCheck :size="17" />
                Payment protection
              </h3>
              <p>Your project will be protected by MergeOS Escrow. Funds are held securely and released only when milestones are approved.</p>
              <div class="escrow-steps">
                <span><LockKeyhole :size="15" /> Funds secured</span>
                <span><GitPullRequest :size="15" /> Work in progress</span>
                <span><CheckCircle2 :size="15" /> Review & approve</span>
                <span><CircleDollarSign :size="15" /> Release funds</span>
              </div>
            </section>
          </div>

          <footer class="project-step-actions">
            <button class="secondary-button compact" type="button" @click="projectWizardBack">
              <ArrowLeft :size="15" />
              Back
            </button>
            <div>
              <button class="secondary-button compact ghost" type="button" @click="showToast('Draft saved locally.')">Save draft</button>
              <button class="primary-button compact" type="button" @click="nextProjectStep">
                {{ projectWizardStep === 4 ? 'Publish project' : 'Continue' }}
                <SendHorizontal v-if="projectWizardStep === 4" :size="15" />
                <ArrowRight v-else :size="15" />
              </button>
            </div>
          </footer>
        </article>

        <article v-else-if="projectWizardStage === 'funding'" class="wizard-card funding-panel">
          <button class="back-link" type="button" @click="projectWizardBack">
            <ArrowLeft :size="15" />
            Back to project setup
          </button>

          <header class="funding-heading">
            <div>
              <h2>Your project is published!</h2>
              <p>To start receiving proposals, add funds to your Escrow. Funds are secure and only released when work is approved.</p>
            </div>
            <div class="escrow-banner">
              <ShieldCheck :size="19" />
              <span>Your payment is protected by MergeOS Escrow.</span>
            </div>
          </header>

          <section class="wizard-section full">
            <div class="wizard-section-title">
              <strong>1. Choose amount</strong>
              <small>Add funds to your escrow to get tokens and attract top talent.</small>
            </div>
            <div class="funding-amount-grid">
              <button
                v-for="option in fundingAmountOptions"
                :key="option.amount"
                :class="{ selected: projectFundingAmount === option.amount }"
                class="funding-amount-card"
                type="button"
                @click="projectFundingAmount = option.amount"
              >
                <strong>{{ formatMoney(option.amount) }}</strong>
                <small>{{ formatMRG(option.tokens) }}</small>
                <span v-if="option.popular">Popular</span>
              </button>
            </div>
            <label class="wizard-field full">
              <span>Custom amount (USD)</span>
              <input v-model.number="projectFundingAmount" type="number" min="100" step="100" />
            </label>
            <div class="token-receipt">
              <span>You will receive</span>
              <strong>{{ projectTokenAmountLabel }}</strong>
              <small>1 USD = {{ TOKEN_RATE_PER_USD }} {{ tokenSymbol }}</small>
            </div>
          </section>

          <section class="wizard-section full">
            <div class="wizard-section-title">
              <strong>2. Payment method</strong>
              <small>Choose your preferred payment method.</small>
            </div>
            <div class="payment-choice-grid">
              <button
                v-for="method in paymentMethodOptions"
                :key="method.label"
                :class="{ selected: projectPaymentMethod === method.label }"
                class="select-tile horizontal rich"
                type="button"
                @click="projectPaymentMethod = method.label"
              >
                <component :is="method.icon" :size="18" />
                <span>
                  <strong>{{ method.label }}</strong>
                  <small>{{ method.caption }}</small>
                </span>
              </button>
            </div>
            <div class="card-input-grid">
              <label class="wizard-field full">
                <span>Card number</span>
                <input placeholder="1234 1234 1234 1234" />
              </label>
              <label class="wizard-field">
                <span>Expiry date</span>
                <input placeholder="MM / YY" />
              </label>
              <label class="wizard-field">
                <span>CVC</span>
                <input placeholder="CVC" />
              </label>
              <label class="wizard-field">
                <span>Cardholder name</span>
                <input placeholder="Name on card" />
              </label>
            </div>
          </section>

          <footer class="funding-actions">
            <span><Lock :size="14" /> Your payment is secure and encrypted.</span>
            <div>
              <small>Total to pay</small>
              <strong>{{ projectFundingAmountLabel }}</strong>
              <button class="primary-button compact" :disabled="projectPaymentBusy" type="button" @click="completeProjectFunding">
                {{ projectPaymentButtonLabel }}
                <LockKeyhole :size="15" />
              </button>
            </div>
          </footer>
          <p v-if="!user" class="funding-login-note">
            <LockKeyhole :size="14" />
            Log in before payment so MergeOS can record the payment, mint tokens, and attach the ledger entries to your project.
          </p>
          <p v-if="projectPaymentError" class="modal-error funding-error">{{ projectPaymentError }}</p>
        </article>

        <article v-else class="wizard-card payment-success-panel">
          <div class="success-hero">
            <span class="success-check"><CheckCircle2 :size="54" /></span>
            <h2>Payment successful!</h2>
            <p>{{ successProjectTitle }} is now funded and ready to go. You will receive proposals from top talent soon.</p>
          </div>

          <section class="payment-details-box">
            <div class="payment-details-heading">
              <strong>Payment details</strong>
              <span>Paid</span>
            </div>
            <div class="payment-detail-grid">
              <div>
                <small>Amount paid</small>
                <strong>{{ projectFundingAmountLabel }}</strong>
              </div>
              <div>
                <small>Tokens received</small>
                <strong>{{ projectTokenAmountLabel }}</strong>
              </div>
              <div>
                <small>Payment method</small>
                <strong>{{ projectPaymentMethod }}</strong>
              </div>
              <div>
                <small>Date & time</small>
                <strong>{{ formatLedgerDateTime(fundedProject?.created_at).full }}</strong>
              </div>
              <div>
                <small>Ledger ref</small>
                <strong>{{ successPaymentReference || 'recorded' }}</strong>
              </div>
            </div>
            <p>
              <ShieldCheck :size="17" />
              Your payment is protected by MergeOS Escrow. Funds are secure and will only be released when work is approved.
            </p>
          </section>

          <section class="next-steps">
            <h3>What happens next?</h3>
            <div class="next-step-grid">
              <article v-for="item in successNextSteps" :key="item.title">
                <span>{{ item.step }}</span>
                <component :is="item.icon" :size="22" />
                <strong>{{ item.title }}</strong>
                <p>{{ item.body }}</p>
              </article>
            </div>
          </section>

          <section class="tokens-box">
            <span class="token-emblem"><CircleDollarSign :size="26" /></span>
            <div>
              <h3>You've received {{ projectTokenAmountLabel }}</h3>
              <p>Use your tokens to boost your project, feature it in the marketplace, or unlock premium matching.</p>
            </div>
            <button class="secondary-button compact" type="button" @click="closeProjectWizard(); openPublicPage('ledger')">Ledger Logs</button>
            <button class="primary-button compact" type="button" @click="closeProjectWizard">
              View my project
              <ArrowRight :size="15" />
            </button>
          </section>
        </article>
      </section>

      <aside class="project-flow-rail">
        <article v-if="projectWizardStage === 'setup' && projectWizardStep === 1" class="rail-card">
          <h3>How it works</h3>
          <ol class="rail-steps">
            <li v-for="item in howItWorks" :key="item">
              <span>{{ howItWorks.indexOf(item) + 1 }}</span>
              {{ item }}
            </li>
          </ol>
        </article>

        <article v-if="projectWizardStage === 'setup' && projectWizardStep === 2" class="rail-card">
          <h3>Tips for a great scope</h3>
          <ul class="rail-check-list">
            <li v-for="tip in scopeTips" :key="tip">
              <CheckCircle2 :size="14" />
              {{ tip }}
            </li>
          </ul>
        </article>

        <article v-if="projectWizardStage === 'setup' && projectWizardStep === 2" class="rail-card purple">
          <h3>AI can help you</h3>
          <p>Generate a detailed scope and requirements from a simple description.</p>
          <button type="button" @click="showToast('AI generated scope suggestions.')">Generate with AI</button>
        </article>

        <article v-if="projectWizardStage === 'setup' && projectWizardStep === 3" class="rail-card project-summary-mini">
          <button type="button" @click="goProjectStep(1)">Edit</button>
          <h3>Project summary</h3>
          <div class="mini-project">
            <span>{{ projectInitial }}</span>
            <div>
              <strong>{{ projectTitleLabel }}</strong>
              <small>{{ projectTypeLabel }}</small>
            </div>
          </div>
          <dl>
            <div>
              <dt>Budget</dt>
              <dd>{{ projectBudgetSummaryLabel }}</dd>
            </div>
            <div>
              <dt>Timeline</dt>
              <dd>{{ projectTimelineLabel }}</dd>
            </div>
            <div>
              <dt>Payment</dt>
              <dd>Escrow (Secure)</dd>
            </div>
            <div>
              <dt>Visibility</dt>
              <dd>{{ projectSetupForm.visibility }}</dd>
            </div>
          </dl>
        </article>

        <article v-if="projectWizardStage === 'setup' && projectWizardStep === 3" class="rail-card budget-suggestion">
          <h3>AI Budget Suggestion</h3>
          <strong>{{ projectBudgetRangeLabel }}</strong>
          <p>{{ projectBudgetAmount ? 'Based on your current project inputs.' : 'Add a budget to calculate an estimate.' }}</p>
          <div class="budget-sparkline" aria-hidden="true">
            <span v-for="height in sparklineHeights" :key="height" :style="{ height: `${height}%` }" />
          </div>
        </article>

        <article v-if="projectWizardStage === 'setup' && projectWizardStep === 4" class="rail-card project-preview-mini">
          <h3>Project preview</h3>
          <div class="mini-project">
            <span>{{ projectInitial }}</span>
            <div>
              <strong>{{ projectTitleLabel }}</strong>
              <small>{{ projectTypeLabel }}</small>
            </div>
          </div>
          <dl>
            <div>
              <dt>Budget</dt>
              <dd>{{ projectBudgetRangeLabel }}</dd>
            </div>
            <div>
              <dt>Timeline</dt>
              <dd>{{ projectTimelineLabel }}</dd>
            </div>
            <div>
              <dt>Experience</dt>
              <dd>Intermediate - Expert</dd>
            </div>
            <div>
              <dt>Deliverables</dt>
              <dd>{{ projectDeliverableCountLabel }}</dd>
            </div>
          </dl>
        </article>

        <article v-if="projectWizardStage === 'setup' && projectWizardStep === 4" class="rail-card">
          <h3>Cost breakdown</h3>
          <dl>
            <div>
              <dt>Client budget</dt>
              <dd>{{ projectBudgetRangeLabel }}</dd>
            </div>
            <div>
              <dt>Platform fee (8%)</dt>
              <dd>{{ formatMRG(projectPlatformFeeLow) }} - {{ formatMRG(projectPlatformFeeHigh) }}</dd>
            </div>
            <div>
              <dt>Escrow fee (2%)</dt>
              <dd>{{ formatMRG(projectEscrowFeeLow) }} - {{ formatMRG(projectEscrowFeeHigh) }}</dd>
            </div>
            <div class="strong-row">
              <dt>Estimated total</dt>
              <dd>{{ projectEstimatedRangeLabel }}</dd>
            </div>
          </dl>
        </article>

        <article v-if="projectWizardStage === 'funding' || projectWizardStage === 'success'" class="rail-card project-summary-mini">
          <button v-if="projectWizardStage === 'funding'" type="button" @click="goProjectStep(4)">Edit</button>
          <h3>Project summary</h3>
          <div class="mini-project">
            <span>{{ projectInitial }}</span>
            <div>
              <strong>{{ projectTitleLabel }}</strong>
              <small>{{ projectTypeLabel }}</small>
            </div>
          </div>
          <dl>
            <div>
              <dt>Budget</dt>
              <dd>{{ projectBudgetRangeLabel }}</dd>
            </div>
            <div>
              <dt>Timeline</dt>
              <dd>{{ projectTimelineLabel }}</dd>
            </div>
            <div>
              <dt>Experience level</dt>
              <dd>Intermediate - Expert</dd>
            </div>
            <div>
              <dt>Team size</dt>
              <dd>Not specified</dd>
            </div>
          </dl>
        </article>

        <article v-if="projectWizardStage === 'funding'" class="rail-card">
          <h3>Escrow & tokens</h3>
          <dl>
            <div>
              <dt>Amount added</dt>
              <dd>{{ projectFundingAmountLabel }}</dd>
            </div>
            <div>
              <dt>Platform fee (8%)</dt>
              <dd>-{{ formatMoney(projectFundingPlatformFee) }}</dd>
            </div>
            <div>
              <dt>Escrow fee (2%)</dt>
              <dd>-{{ formatMoney(projectFundingEscrowFee) }}</dd>
            </div>
            <div class="strong-row">
              <dt>You will receive</dt>
              <dd>{{ projectTokenAmountLabel }}</dd>
            </div>
          </dl>
        </article>

        <article v-if="projectWizardStage === 'success'" class="rail-card next-action-card">
          <h3>Next steps</h3>
          <button v-for="item in postPaymentActions" :key="item" type="button" @click="showToast(`${item} opened.`)">
            <CheckCircle2 :size="15" />
            {{ item }}
            <ArrowRight :size="14" />
          </button>
        </article>
      </aside>
    </main>

    <footer class="project-flow-footer">
      <div class="footer-progress">
        <span>Step {{ footerStepNumber }} of 4</span>
        <i><b :style="{ width: `${footerProgress}%` }" /></i>
      </div>
      <nav aria-label="Project flow progress">
        <span
          v-for="item in projectFooterSteps"
          :key="item.label"
          :class="{ active: item.active, done: item.done }"
        >
          <CheckCircle2 v-if="item.done" :size="15" />
          <small v-else>{{ item.number }}</small>
          {{ item.label }}
        </span>
      </nav>
      <p>
        <ShieldCheck :size="17" />
        {{ footerProtectionCopy }}
      </p>
    </footer>
  </div>

  <div v-else-if="user && !publicModeVisible" class="dashboard-shell">
    <div v-if="toastMessage" class="toast dashboard-toast">
      {{ toastMessage }}
    </div>

    <aside class="dash-sidebar" aria-label="Customer navigation">
      <button class="dash-brand" type="button" @click="showToast('Opening dashboard home...')">
        <span class="brand-mark" aria-hidden="true">
          <img src="/favicon.svg" alt="" />
        </span>
        <strong>MergeOS</strong>
      </button>

      <nav class="dash-side-nav">
        <section v-for="section in sidebarSections" :key="section.label">
          <p>{{ section.label }}</p>
          <button
            v-for="item in section.items"
            :key="item.label"
            :class="{ active: item.active }"
            type="button"
            @click="handleDashboardNav(item)"
          >
            <component :is="item.icon" :size="16" />
            {{ item.label }}
          </button>
        </section>
      </nav>

      <article class="mrg-card">
        <span class="mrg-medal">
          <Trophy :size="18" />
        </span>
        <strong>Earn MRG</strong>
        <p>Complete tasks and get paid in MRG tokens.</p>
        <button type="button" @click="showToast('Opening MRG rewards...')">
          Learn more
          <ArrowRight :size="14" />
        </button>
      </article>
    </aside>

    <section class="dash-workspace">
      <header class="dash-topbar">
        <label class="dash-search">
          <Search :size="16" />
          <input v-model.trim="dashboardSearch" placeholder="Search your live projects..." />
          <kbd>Ctrl K</kbd>
        </label>

        <nav class="dash-topnav" aria-label="Dashboard sections">
          <button
            v-for="item in topNavItems"
            :key="item.label"
            :class="{ active: item.active }"
            type="button"
            @click="handleDashboardNav(item)"
          >
            {{ item.label }}
          </button>
        </nav>

        <div class="dash-top-actions">
          <button class="dash-icon-button" aria-label="Notifications" type="button" @click="toggleNotificationPanel">
            <Bell :size="17" />
            <span v-if="unreadNotificationCount > 0" class="notif-badge">{{ unreadNotificationCount }}</span>
          </button>

          <!-- Notification Panel Dropdown -->
          <div v-if="notificationPanelOpen" class="notification-panel" @click.stop>
            <div class="notification-panel-header">
              <h3>Notifications</h3>
              <div class="notification-panel-actions">
                <button class="notif-action-btn" type="button" @click="markAllNotificationsRead" v-if="unreadNotificationCount > 0">
                  <CheckCircle2 :size="14" /> Mark all read
                </button>
                <button class="notif-action-btn" type="button" @click="fetchNotifications">
                  <RefreshCw :size="14" /> Refresh
                </button>
              </div>
            </div>
            <div class="notification-panel-body" v-if="!notificationsLoading">
              <div v-if="notifications.length === 0" class="notification-empty">
                <Bell :size="32" />
                <p>No notifications yet</p>
                <small>You'll see updates about your projects here</small>
              </div>
              <ul v-else class="notification-list">
                <li v-for="notif in notifications" :key="notif.id"
                    :class="['notification-item', { unread: !notif.read }]"
                    @click="markNotificationRead(notif)">
                  <div class="notif-icon" :class="`notif-${notif.channel || 'info'}`">
                    <Bell v-if="notif.channel === 'system'" :size="14" />
                    <DollarSign v-else-if="notif.channel === 'payment'" :size="14" />
                    <GitPullRequest v-else-if="notif.channel === 'project'" :size="14" />
                    <MessageCircle v-else :size="14" />
                  </div>
                  <div class="notif-content">
                    <strong>{{ notif.subject || 'Notification' }}</strong>
                    <p>{{ notif.body || notif.message }}</p>
                    <small class="notif-time">{{ formatNotificationTime(notif.created_at) }}</small>
                  </div>
                  <div v-if="!notif.read" class="notif-unread-dot"></div>
                </li>
              </ul>
            </div>
            <div v-else class="notification-panel-body notification-loading">
              <RefreshCw :size="24" class="notif-spinner" />
              <p>Loading notifications...</p>
            </div>
          </div>
          <button class="primary-button compact" type="button" @click="openProjectWizard">
            <Plus :size="16" />
            New Project
          </button>
          <button class="dash-profile" type="button" @click="logout">
            <span class="profile-avatar">{{ initialsFor(user.name || user.email) }}</span>
            <span>
              <strong>{{ user.name || user.email || 'Signed-in user' }}</strong>
              <small>{{ user.wallet_address ? shortWallet(user.wallet_address) : 'Customer' }}</small>
            </span>
            <ChevronDown :size="14" />
          </button>
        </div>
      </header>

      <main class="dash-content">
        <section class="dash-main">
          <div class="dash-breadcrumb">
            <Home :size="14" />
            <span>My Projects</span>
            <ChevronDown :size="13" />
            <strong>{{ dashboardProjectView.title }}</strong>
          </div>

          <section v-if="dashboardError" class="dash-empty-state">
            <strong>Could not load your projects</strong>
            <p>{{ dashboardError }}</p>
            <button class="secondary-button compact" type="button" @click="loadDashboardData">Retry</button>
          </section>

          <section class="dash-project-header">
            <div class="dash-project-title">
              <span class="project-photo">{{ dashboardProjectView.initials }}</span>
              <div>
                <h1>{{ dashboardProjectView.title }}</h1>
                <p>{{ dashboardProjectView.body }}</p>
              </div>
              <span class="live-badge">{{ dashboardProjectView.status }}</span>
            </div>

            <div class="dash-project-actions">
              <button type="button" @click="loadDashboardData">
                <RefreshCw :size="15" />
                Refresh
              </button>
              <button type="button" @click="showToast('Project share link copied.')">
                <Share2 :size="15" />
                Share
              </button>
              <button type="button" aria-label="More project actions" @click="showToast('Opening project actions...')">
                <MoreHorizontal :size="16" />
              </button>
            </div>
          </section>

          <section class="dash-metrics" aria-label="Project summary">
            <article>
              <span>Budget</span>
              <strong>{{ dashboardProjectView.budget }}</strong>
              <small>{{ dashboardProjectView.budgetCaption }}</small>
            </article>
            <article>
              <span>Progress</span>
              <strong>{{ dashboardProjectView.progress }}%</strong>
              <div class="mini-progress"><i :style="{ width: `${dashboardProjectView.progress}%` }" /></div>
            </article>
            <article>
              <span>Tasks</span>
              <strong>{{ dashboardProjectView.taskSummary }}</strong>
            </article>
            <article>
              <span>Repository</span>
              <strong>{{ dashboardProjectView.repo }}</strong>
            </article>
            <article>
              <span>Created</span>
              <strong>{{ dashboardProjectView.created }}</strong>
            </article>
          </section>

          <div class="dash-tabs" role="tablist" aria-label="Project tabs">
            <button
              v-for="tabItem in dashboardTabs"
              :key="tabItem"
              :class="{ active: tabItem === 'Overview' }"
              type="button"
              role="tab"
            >
              {{ tabItem }}
              <span v-if="tabItem === 'Tasks'">{{ dashboardTaskRows.length }}</span>
            </button>
          </div>

          <section class="dash-overview-grid">
            <article class="dash-card progress-overview-card">
              <h2>Progress Overview</h2>
              <div class="progress-card-body">
                <div class="progress-ring large" :style="dashboardRingStyle" :aria-label="`${dashboardProgress} percent completed`">
                  <strong>{{ dashboardProgress }}%</strong>
                  <span>Completed</span>
                </div>
                <div class="progress-legend compact">
                  <span><i class="green-dot" />Completed <b>{{ dashboardAcceptedTasks.length }} tasks</b></span>
                  <span><i class="blue-dot" />Open <b>{{ dashboardOpenTasks.length }} tasks</b></span>
                  <span><i class="orange-dot" />Ledger <b>{{ dashboardProjectLedger.length }} entries</b></span>
                  <span><i class="gray-dot" />Escrow <b>{{ formatMRGFromCents(dashboardLedgerFundingCents) }}</b></span>
                </div>
              </div>
            </article>

            <article class="dash-card budget-card">
              <h2>Budget & Payments</h2>
              <div class="budget-lines">
                <span>Total Budget <strong>{{ formatMRGFromCents(dashboardSelectedProject?.budget_cents) }}</strong></span>
                <span>Work Pool <strong>{{ formatMRGFromCents(dashboardSelectedProject?.work_pool_cents) }}</strong></span>
                <span>Released <strong>{{ formatMRGFromCents(dashboardLedgerPayoutCents || dashboardSpentCents) }}</strong></span>
                <span>Remaining <strong>{{ formatMRGFromCents(dashboardRemainingCents) }}</strong></span>
              </div>
              <button type="button" @click="openPublicPage('ledger')">View Ledger</button>
            </article>

            <article class="dash-card analysis-card">
              <div class="card-title-row">
                <h2>Work Split</h2>
                <span>Live</span>
              </div>
              <p>Task routing is loaded from the funded project plan.</p>
              <div class="risk-grid">
                <span v-for="item in dashboardWorkSplit" :key="item.label" :class="item.className">
                  {{ item.label }}
                  <strong>{{ item.value }}</strong>
                </span>
              </div>
              <button type="button" @click="showToast('Opening task routing...')">View Routing</button>
            </article>
          </section>

          <section class="dash-card live-pr-board">
            <div class="card-title-row">
              <div>
                <h2>Project Tasks</h2>
                <p>Real tasks split from the funded project and loaded from the backend.</p>
              </div>
              <button type="button" @click="loadDashboardData">Refresh Tasks</button>
            </div>

            <div v-if="dashboardTaskRows.length" class="dash-pr-list">
              <article v-for="task in dashboardTaskRows" :key="task.id" class="dash-pr-row">
                <span class="contributor-avatar">{{ task.initials }}</span>
                <div class="dash-pr-main">
                  <strong>#{{ task.issueNumber }} {{ task.title }}</strong>
                  <small>{{ task.acceptance }}</small>
                  <span>{{ task.reference }}</span>
                </div>
                <div class="dash-pr-stat">
                  <strong>{{ task.reward }}</strong>
                  <small>Reward</small>
                </div>
                <div class="dash-pr-stat positive">
                  <strong>{{ task.kind }}</strong>
                  <small>Worker</small>
                </div>
                <div class="dash-pr-stat negative">
                  <strong>{{ task.agent }}</strong>
                  <small>Agent</small>
                </div>
                <b :class="task.statusClass">{{ task.status }}</b>
              </article>
            </div>
            <article v-else class="dash-empty-state compact">
              <strong>{{ dashboardLoading ? 'Loading tasks...' : 'No tasks yet' }}</strong>
              <p>{{ dashboardLoading ? 'Fetching your project task split.' : 'Fund a project to generate real tasks.' }}</p>
            </article>

            <div class="watching-line">
              <ListTodo :size="14" />
              {{ dashboardTaskRows.length }} real tasks loaded
            </div>
          </section>
        </section>

        <aside class="dash-rail">
          <section class="dash-card rail-card wallet-summary-card">
            <div class="card-title-row">
              <h2>MRG Wallet</h2>
              <span class="recording-dot">Live</span>
            </div>
            <div class="wallet-address-box">
              <small>Wallet address</small>
              <strong>{{ user.wallet_address || 'Creating wallet...' }}</strong>
            </div>
            <div class="wallet-link-row">
              <span>{{ user.github_username ? `github:${user.github_username}` : 'GitHub not linked' }}</span>
            </div>
            <div class="wallet-action-row">
              <button class="rail-link-button" :disabled="!user.wallet_address" type="button" @click="openWalletOnScan(user.wallet_address)">
                View on Scan
              </button>
              <button class="rail-link-button" :disabled="authBusy || !githubOAuthReady" type="button" @click="startGitHubLogin">
                Link GitHub
              </button>
            </div>
          </section>

          <section class="dash-card rail-card project-picker-card">
            <div class="card-title-row">
              <h2>My Projects</h2>
              <span>{{ dashboardProjectList.length }}</span>
            </div>
            <div v-if="dashboardProjectList.length" class="dashboard-project-list">
              <button
                v-for="project in dashboardProjectList"
                :key="project.id"
                :class="{ active: project.id === dashboardSelectedProject?.id }"
                type="button"
                @click="selectedDashboardProjectID = project.id"
              >
                <span class="contributor-avatar">{{ initialsFor(project.title || project.company_name || 'MP') }}</span>
                <div>
                  <strong>{{ project.title }}</strong>
                  <small>{{ formatMRGFromCents(project.budget_cents) }} - {{ (project.tasks || []).length }} tasks</small>
                </div>
              </button>
            </div>
            <article v-else class="dash-empty-state compact">
              <strong>{{ dashboardLoading ? 'Loading projects...' : 'No projects found' }}</strong>
              <p>{{ dashboardLoading ? 'Syncing your workspace.' : 'Create and fund a project to populate this list.' }}</p>
            </article>
          </section>

          <section class="dash-card rail-card">
            <div class="card-title-row">
              <h2>Live Activity</h2>
              <span class="recording-dot">Live</span>
            </div>
            <div v-if="dashboardActivityRows.length" class="rail-activity-list">
              <article v-for="activity in dashboardActivityRows" :key="activity.key">
                <span :class="['activity-icon', activity.color]">
                  <component :is="activity.icon" :size="14" />
                </span>
                <div>
                  <strong>{{ activity.title }}</strong>
                  <small>{{ activity.time }}</small>
                </div>
              </article>
            </div>
            <article v-else class="dash-empty-state compact">
              <strong>No ledger activity</strong>
              <p>Project ledger entries will appear after funding.</p>
            </article>
            <button class="rail-link-button" type="button" @click="openPublicPage('ledger')">
              View ledger
            </button>
          </section>

          <section class="dash-card rail-card chat-card">
            <div class="card-title-row">
              <h2>Ledger Snapshot</h2>
              <span class="online-dot">{{ dashboardLedgerRows.length }} rows</span>
            </div>
            <div v-if="dashboardLedgerRows.length" class="chat-list dashboard-ledger-list">
              <article v-for="entry in dashboardLedgerRows" :key="entry.key">
                <span class="contributor-avatar">LG</span>
                <div>
                  <div class="chat-meta">
                    <strong>{{ entry.title }}</strong>
                    <small>{{ entry.value }}</small>
                  </div>
                  <p>{{ entry.ref }}</p>
                </div>
              </article>
            </div>
            <article v-else class="dash-empty-state compact">
              <strong>No ledger rows</strong>
              <p>Funding and payout entries will appear here.</p>
            </article>
          </section>
        </aside>
      </main>
    </section>
  </div>

  <div v-else class="home-shell">
    <div v-if="toastMessage" class="toast">
      {{ toastMessage }}
    </div>

    <header class="home-navbar">
      <div class="home-container nav-inner">
        <a class="brand-link" href="/" @click.prevent="openPublicPage('home')">
          <span class="brand-mark" aria-hidden="true">
            <img src="/favicon.svg" alt="" />
          </span>
          <strong>MergeOS</strong>
        </a>

        <nav class="nav-links" aria-label="Primary">
          <a href="/product" :class="{ 'nav-active': publicPage === 'product' }" @click.prevent="openPublicPage('product')">
            Product
            <ChevronDown :size="13" />
          </a>
          <a href="/solutions" :class="{ 'nav-active': publicPage === 'solutions' }" @click.prevent="openPublicPage('solutions')">
            Solutions
            <ChevronDown :size="13" />
          </a>
          <a href="/marketplace" :class="{ 'nav-active': publicPage === 'marketplace' }" @click.prevent="openPublicPage('marketplace')">Marketplace</a>
          <a href="/how-it-works" :class="{ 'nav-active': publicPage === 'how-it-works' }" @click.prevent="openPublicPage('how-it-works')">How it works</a>
          <a href="/ledger" :class="{ 'nav-active': publicPage === 'ledger' }" @click.prevent="openPublicPage('ledger')">Ledger Logs</a>
        </nav>

        <div class="nav-actions">
          <template v-if="user">
            <button class="secondary-button compact" type="button" @click="openDashboard">Dashboard</button>
            <span class="user-pill">{{ user.name || user.email }}</span>
            <button class="secondary-button compact" type="button" @click="logout">Logout</button>
          </template>
          <template v-else>
            <button class="secondary-button compact" type="button" @click="openAuth('login')">Log in</button>
            <button class="primary-button compact" type="button" @click="openAuth('register')">Sign up</button>
          </template>
        </div>
      </div>
    </header>

    <main v-if="publicPage === 'home'" id="top" class="public-home-page">
      <div class="home-container public-home-layout">
        <section class="public-home-hero" aria-labelledby="home-title">
          <div class="public-home-copy">
            <span class="marketplace-eyebrow">HOME</span>
            <h1 id="home-title">MergeOS</h1>
            <p>Post funded software work, match with talent or AI agents, and verify every payment through real ledger logs.</p>
            <div class="marketplace-actions">
              <button class="primary-button large" type="button" @click="openProjectWizard">
                Start a project
                <ArrowRight :size="16" />
              </button>
              <button class="secondary-button large" type="button" @click="openPublicPage('marketplace')">
                Find talent
                <UsersRound :size="16" />
              </button>
              <button class="secondary-button large" type="button" @click="openPublicPage('ledger')">
                Ledger Logs
                <Link2 :size="16" />
              </button>
            </div>
          </div>

          <aside class="public-home-panel" aria-label="Live platform summary">
            <div class="ledger-card-head">
              <h2>Live platform</h2>
              <span class="ledger-live-dot">Live</span>
            </div>
            <div class="public-stat-grid">
              <article v-for="stat in homeLiveStats" :key="stat.label">
                <strong>{{ stat.value }}</strong>
                <span>{{ stat.label }}</span>
              </article>
            </div>
          </aside>
        </section>

        <section class="public-workflow-grid" aria-label="MergeOS workflows">
          <button v-for="card in homeWorkflowCards" :key="card.title" type="button" @click="handlePublicAction(card.action)">
            <span :class="['public-card-icon', card.tone]">
              <component :is="card.icon" :size="19" />
            </span>
            <strong>{{ card.title }}</strong>
            <p>{{ card.body }}</p>
            <small>
              {{ card.cta }}
              <ArrowRight :size="13" />
            </small>
          </button>
        </section>

        <section class="public-talent-strip" aria-label="Talent matching">
          <div>
            <span class="marketplace-eyebrow">TALENT</span>
            <h2>Find the right builder without logging in first.</h2>
            <p>Browse live funded work, agent queues, and contributor signals before deciding whether to start a project.</p>
          </div>
          <div class="public-talent-list">
            <article v-for="row in homeTalentRows" :key="row.title">
              <span :class="['public-card-icon', row.tone]">
                <component :is="row.icon" :size="18" />
              </span>
              <div>
                <strong>{{ row.title }}</strong>
                <small>{{ row.body }}</small>
              </div>
            </article>
          </div>
        </section>
      </div>
    </main>

    <main v-else-if="publicInfoPage" id="top" class="public-info-page">
      <div class="home-container public-info-layout">
        <section class="public-info-hero">
          <div>
            <span class="marketplace-eyebrow">{{ publicInfoPage.eyebrow }}</span>
            <h1>{{ publicInfoPage.title }}</h1>
            <p>{{ publicInfoPage.body }}</p>
            <div class="marketplace-actions">
              <button
                v-for="action in publicInfoPage.actions"
                :key="action.label"
                :class="[action.primary ? 'primary-button' : 'secondary-button', 'large']"
                type="button"
                @click="handlePublicAction(action)"
              >
                {{ action.label }}
                <component :is="action.icon" :size="16" />
              </button>
            </div>
          </div>
          <aside class="public-info-summary">
            <article v-for="item in publicInfoPage.summary" :key="item.label">
              <span :class="['public-card-icon', item.tone]">
                <component :is="item.icon" :size="18" />
              </span>
              <div>
                <strong>{{ item.label }}</strong>
                <small>{{ item.value }}</small>
              </div>
            </article>
          </aside>
        </section>

        <section class="public-info-grid" aria-label="Page details">
          <article v-for="item in publicInfoPage.features" :key="item.title">
            <span :class="['public-card-icon', item.tone]">
              <component :is="item.icon" :size="18" />
            </span>
            <strong>{{ item.title }}</strong>
            <p>{{ item.body }}</p>
          </article>
        </section>
      </div>
    </main>

    <main v-else-if="publicPage === 'ledger'" id="top" class="ledger-page">
      <div class="home-container ledger-shell">
        <section class="ledger-hero">
          <div class="ledger-hero-copy">
            <span class="marketplace-eyebrow">LEDGER LOGS</span>
            <div class="ledger-title-row">
              <h1>Ledger Logs</h1>
              <span class="ledger-public-badge">
                <Globe2 :size="14" />
                Real data
              </span>
            </div>
            <p>Transparent platform activity from the live ledger. Payments, token mints, reserves, and payouts are loaded from the backend.</p>

            <div class="ledger-trust-row" aria-label="Ledger trust signals">
              <article v-for="item in ledgerTrustItems" :key="item.title">
                <span :class="['ledger-trust-icon', item.tone]">
                  <component :is="item.icon" :size="16" />
                </span>
                <div>
                  <strong>{{ item.title }}</strong>
                  <small>{{ item.body }}</small>
                </div>
              </article>
            </div>
          </div>

          <aside class="ledger-live-card" aria-label="Live MergeOS metrics">
            <div class="ledger-card-head">
              <h2>Live on MergeOS</h2>
              <span class="ledger-live-dot">Live</span>
            </div>
            <div class="ledger-live-grid">
              <article v-for="stat in ledgerLiveStats" :key="stat.label">
                <strong>{{ stat.value }}</strong>
                <span>{{ stat.label }}</span>
              </article>
            </div>
            <button type="button" @click="loadLedgerData">
              <BarChart3 :size="14" />
              Refresh live feed
              <ArrowRight :size="14" />
            </button>
          </aside>
        </section>

        <section class="ledger-content">
          <div class="ledger-main-card">
            <div class="ledger-tabs-row">
              <div class="ledger-tabs" role="tablist" aria-label="Ledger activity">
                <button
                  v-for="tabItem in ledgerTabs"
                  :key="tabItem"
                  :class="{ active: tabItem === 'All Activity' }"
                  type="button"
                  role="tab"
                  @click="showToast(`Filtering ${tabItem}...`)"
                >
                  {{ tabItem }}
                </button>
              </div>

              <div class="ledger-table-actions">
                <button type="button" @click="showToast('Project filter opened.')">
                  All Projects
                  <ChevronDown :size="13" />
                </button>
                <button type="button" @click="showToast('Advanced filters opened.')">
                  <Filter :size="14" />
                  Filters
                </button>
              </div>
            </div>

            <div class="ledger-table-wrap">
              <table class="ledger-table">
                <thead>
                  <tr>
                    <th>Time (UTC)</th>
                    <th>Type</th>
                    <th>Project</th>
                    <th>Amount</th>
                    <th>Status</th>
                    <th>Tx / Ref</th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-if="ledgerLoading">
                    <td class="ledger-state-cell" colspan="6">Loading real ledger entries...</td>
                  </tr>
                  <tr v-else-if="ledgerError">
                    <td class="ledger-state-cell error" colspan="6">{{ ledgerError }}</td>
                  </tr>
                  <tr v-else-if="ledgerEvents.length === 0">
                    <td class="ledger-state-cell" colspan="6">No ledger entries yet. Fund a project to mint tokens and create the first logs.</td>
                  </tr>
                  <template v-else>
                    <tr v-for="event in ledgerEvents" :key="event.key">
                      <td>
                        <strong>{{ event.date }}</strong>
                        <span>{{ event.time }}</span>
                      </td>
                      <td>
                        <span :class="['ledger-event-type', event.tone]">
                          <component :is="event.icon" :size="15" />
                          {{ event.type }}
                        </span>
                      </td>
                      <td>
                        <div class="ledger-project-cell">
                          <span :class="['ledger-project-logo', event.projectTone]">{{ event.projectInitial }}</span>
                          <div>
                            <strong>{{ event.project }}</strong>
                            <span>by {{ event.company }}</span>
                          </div>
                        </div>
                      </td>
                      <td>
                        <strong :class="['ledger-amount', event.amountTone]">{{ event.amount }}</strong>
                        <span v-if="event.secondaryAmount">{{ event.secondaryAmount }}</span>
                      </td>
                      <td>
                        <span class="ledger-status">Verified</span>
                      </td>
                      <td>
                        <button class="ledger-ref-button" type="button" @click="showToast(`Opening ${event.ref}...`)">
                          {{ event.ref }}
                          <Link2 :size="12" />
                        </button>
                      </td>
                    </tr>
                  </template>
                </tbody>
              </table>
            </div>

            <button class="ledger-load-button" type="button" @click="loadLedgerData">
              Refresh ledger
              <ChevronDown :size="13" />
            </button>
          </div>

          <aside class="ledger-rail">
            <section class="ledger-side-card">
              <div class="side-card-head">
                <h2>Trending Projects</h2>
                <button type="button" @click="showToast('Opening trending projects...')">View all <ArrowRight :size="13" /></button>
              </div>
              <div class="ledger-project-list">
                <article v-if="ledgerTrendingProjects.length === 0">
                  <span class="ledger-project-logo green">M</span>
                  <div>
                    <div>
                      <strong>No funded projects yet</strong>
                    </div>
                    <small>Real projects will appear after payment.</small>
                    <p>
                      <b>0 {{ tokenSymbol }} Escrow</b>
                      <span>0 Contributors</span>
                      <span>0 PRs</span>
                    </p>
                  </div>
                </article>
                <article v-for="project in ledgerTrendingProjects" :key="project.title">
                  <span :class="['ledger-project-logo', project.tone]">{{ project.initial }}</span>
                  <div>
                    <div>
                      <strong>{{ project.title }}</strong>
                      <span class="ledger-live-dot compact">Live</span>
                    </div>
                    <small>by {{ project.company }}</small>
                    <p>
                      <b>{{ project.escrow }}</b>
                      <span>{{ project.contributors }} Contributors</span>
                      <span>{{ project.prs }} PRs</span>
                    </p>
                  </div>
                </article>
              </div>
            </section>

            <section class="ledger-side-card ledger-verified-card">
              <h2>
                <ShieldCheck :size="16" />
                Verified by MergeOS
              </h2>
              <ul>
                <li v-for="check in ledgerVerificationChecks" :key="check">
                  <CheckCircle2 :size="13" />
                  {{ check }}
                </li>
              </ul>
              <button type="button" @click="showToast('Opening transparency docs...')">
                Learn more about transparency
                <ArrowRight :size="13" />
              </button>
            </section>

            <section class="ledger-side-card ledger-chain-card">
              <h2>Explore on-chain</h2>
              <p>All transactions are recorded on-chain and verifiable on the blockchain.</p>
              <button type="button" @click="showToast('Opening block explorer...')">
                View on Explorer
                <Link2 :size="13" />
              </button>
              <div class="ledger-chain-row" v-for="chain in ledgerChainRows" :key="chain.label">
                <span>{{ chain.label }}</span>
                <strong>{{ chain.value }}</strong>
              </div>
            </section>
          </aside>
        </section>

        <section class="ledger-footer-stats" aria-label="Ledger totals">
          <article>
              <ShieldCheck :size="18" />
              <div>
                <strong>Built for transparency.</strong>
                <span>Ready for builder verification.</span>
              </div>
            </article>
          <article v-for="stat in ledgerFooterStats" :key="stat.label">
            <strong>{{ stat.value }}</strong>
            <span>{{ stat.label }}</span>
          </article>
        </section>
      </div>
    </main>

    <main v-else-if="publicPage === 'marketplace'" id="top" class="marketplace-page">
      <div class="home-container marketplace-layout">
        <section class="marketplace-main">
          <section class="marketplace-hero" aria-labelledby="marketplace-title">
            <div class="marketplace-copy">
              <span class="marketplace-eyebrow">MARKETPLACE</span>
              <h1 id="marketplace-title">
                Explore funded work and AI tasks <span>from live escrow data</span>
              </h1>
              <p>
                Browse real MergeOS projects, open task pools, contributors, and AI work queues backed by the platform ledger.
              </p>

              <div class="marketplace-actions">
                <button class="primary-button large" type="button" @click="openProjectWizard">
                  Post a Project
                </button>
                <button class="secondary-button large" type="button" @click="openPublicPage('how-it-works')">
                  <span class="play-icon" aria-hidden="true">
                    <ArrowRight :size="14" />
                  </span>
                  How it works
                </button>
              </div>

              <div class="marketplace-trust" aria-label="Marketplace trust signals">
                <article v-for="item in marketplaceTrustItems" :key="item.title">
                  <span :class="['marketplace-trust-icon', item.tone]">
                    <component :is="item.icon" :size="17" />
                  </span>
                  <div>
                    <strong>{{ item.title }}</strong>
                    <small>{{ item.body }}</small>
                  </div>
                </article>
              </div>
            </div>

            <aside class="marketplace-visual" aria-label="Talent and AI matching preview">
              <span class="market-route route-one"></span>
              <span class="market-route route-two"></span>
              <span class="route-node route-node-green">
                <CheckCircle2 :size="17" />
              </span>
              <span class="route-node route-node-purple">
                <UsersRound :size="18" />
              </span>
              <span class="route-node route-node-orange">
                <Code2 :size="18" />
              </span>

              <article class="market-float-card talent-preview-card">
                <div class="talent-card-top">
                  <span class="market-avatar avatar-green">{{ marketplaceHeroProject.clientInitials }}</span>
                  <div>
                    <span class="star-icons">
                      <CheckCircle2 :size="13" fill="currentColor" />
                    </span>
                    <small>{{ marketplaceHeroProject.taskLabel }}</small>
                    <b>{{ marketplaceHeroProject.badge }}</b>
                  </div>
                </div>
                <strong>{{ marketplaceHeroProject.title }}</strong>
                <div class="mini-tags">
                  <span v-for="tag in marketplaceHeroProject.tags.slice(0, 3)" :key="tag">{{ tag }}</span>
                </div>
                <div class="talent-card-bottom">
                  <strong>{{ marketplaceHeroProject.budget }}</strong>
                  <span>{{ marketplaceHeroProject.timeline }}</span>
                </div>
              </article>

              <article class="market-code-card">
                <code>
                  function <span>mergeOS</span>() {<br />
                  &nbsp;&nbsp;return <b>"Ship faster"</b>;<br />
                  }
                </code>
              </article>

              <article class="market-float-card agent-preview-card">
                <div>
                  <span class="agent-icon">
                    <Bot :size="18" />
                  </span>
                  <small>AI Agent</small>
                  <button aria-label="More AI agent actions" type="button" @click="showToast('Opening agent actions...')">
                    <MoreHorizontal :size="17" />
                  </button>
                </div>
                <strong>{{ marketplaceHeroAgent.title }}</strong>
                <p>{{ marketplaceHeroAgent.body }}</p>
              </article>
            </aside>
          </section>

          <section class="marketplace-filter-panel" aria-label="Search and filters">
            <label class="marketplace-search">
              <Search :size="18" />
              <input v-model.trim="marketplaceSearch" placeholder="Search real projects, tasks, or clients..." />
            </label>

            <div class="marketplace-selects">
              <button v-for="filter in marketplaceFilters" :key="filter" type="button" @click="showToast(`Filtering by ${filter}...`)">
                {{ filter }}
                <ChevronDown :size="14" />
              </button>
              <button class="more-filter-button" type="button" @click="showToast('Opening more filters...')">
                <Filter :size="15" />
                More filters
              </button>
            </div>

            <div class="marketplace-categories" role="tablist" aria-label="Marketplace categories">
              <button
                v-for="category in marketplaceCategories"
                :key="category"
                :class="{ active: category === activeMarketplaceCategory }"
                type="button"
                role="tab"
                @click="activeMarketplaceCategory = category"
              >
                {{ category }}
              </button>
            </div>
          </section>

          <section id="marketplace-projects" class="featured-projects-section">
            <div class="section-heading-row">
              <h2>
                <Star :size="17" />
                Live Projects
              </h2>
              <div class="marketplace-data-status">
                <span v-if="marketplaceLoading">Loading live data...</span>
                <template v-else-if="marketplaceError">
                  <span>{{ marketplaceError }}</span>
                  <button type="button" @click="loadMarketplaceData">Retry</button>
                </template>
                <span v-else>{{ marketplaceSummaryLabel }}</span>
              </div>
            </div>

            <div v-if="marketplaceProjectsView.length" class="marketplace-project-grid">
              <article
                v-for="project in marketplaceProjectsView"
                :key="project.id"
                class="marketplace-project-card"
                :style="{ '--project-accent': project.accent, '--project-soft': project.soft }"
              >
                <div class="project-card-top">
                  <span class="project-market-icon">
                    <component :is="project.icon" :size="24" />
                  </span>
                  <span :class="['project-status-badge', project.badgeTone]">{{ project.badge }}</span>
                </div>
                <h3>{{ project.title }}</h3>
                <p>{{ project.body }}</p>
                <div class="project-tag-row">
                  <span v-for="tag in project.tags" :key="tag">{{ tag }}</span>
                  <span v-if="project.extra">+{{ project.extra }}</span>
                </div>
                <div class="project-money-row">
                  <strong>{{ project.budget }}</strong>
                  <span :class="{ urgent: project.urgent }">{{ project.timeline }}</span>
                </div>
                <div class="project-client-row">
                  <span class="market-avatar small" :class="project.avatarTone">{{ project.clientInitials }}</span>
                  <strong>{{ project.client }}</strong>
                  <CheckCircle2 v-if="project.verified" :size="15" />
                  <span class="project-rating">
                    <ListTodo :size="13" />
                    {{ project.taskLabel }}
                  </span>
                </div>
              </article>
            </div>
            <article v-else class="marketplace-empty-state">
              <strong>{{ marketplaceLoading ? 'Loading projects...' : 'No matching live projects' }}</strong>
              <p>{{ marketplaceLoading ? 'Fetching current marketplace data from MergeOS.' : 'Try another search or post a funded project to create the first marketplace listing.' }}</p>
              <button v-if="!marketplaceLoading" class="secondary-button compact" type="button" @click="marketplaceSearch = ''; activeMarketplaceCategory = 'All'">
                Clear filters
              </button>
            </article>

            <button class="view-projects-button" type="button" @click="loadMarketplaceData">
              Refresh live data
              <ArrowRight :size="15" />
            </button>
          </section>

          <section id="marketplace-benefits" class="marketplace-benefit-strip" aria-label="Marketplace benefits">
            <article v-for="benefit in marketplaceBenefits" :key="benefit.title">
              <span>
                <component :is="benefit.icon" :size="23" />
              </span>
              <div>
                <strong>{{ benefit.title }}</strong>
                <p>{{ benefit.body }}</p>
              </div>
            </article>
          </section>
        </section>

        <aside class="marketplace-rail">
          <section class="marketplace-side-card">
            <div class="side-card-head">
              <h2>Top Contributors</h2>
              <button type="button" @click="showToast('Opening contributors...')">View all</button>
            </div>
            <div class="contributor-list">
              <article v-for="contributor in marketplaceContributorsView" :key="contributor.workerId">
                <span>{{ contributor.rank }}</span>
                <span class="market-avatar small" :class="contributor.tone">{{ contributor.initials }}</span>
                <div>
                  <strong>{{ contributor.name }}</strong>
                  <small>{{ contributor.role }}</small>
                  <small>{{ contributor.earned }} earned</small>
                </div>
              </article>
              <article v-if="!marketplaceContributorsView.length" class="marketplace-side-empty">
                <div>
                  <strong>No payouts yet</strong>
                  <small>Accepted task contributors will appear here.</small>
                </div>
              </article>
            </div>
          </section>

          <section class="marketplace-side-card">
            <div class="side-card-head">
              <h2>AI Work Queue</h2>
              <button type="button" @click="loadMarketplaceData">Refresh</button>
            </div>
            <div class="agent-list">
              <article v-for="agent in marketplaceAgentsView" :key="agent.type">
                <span :class="['popular-agent-icon', agent.tone]">
                  <component :is="agent.icon" :size="21" />
                </span>
                <div>
                  <strong>{{ agent.title }}</strong>
                  <small>{{ agent.body }}</small>
                </div>
              </article>
              <article v-if="!marketplaceAgentsView.length" class="marketplace-side-empty">
                <div>
                  <strong>No agent tasks yet</strong>
                  <small>Open AI-scoped tasks will appear here.</small>
                </div>
              </article>
            </div>
          </section>
        </aside>
      </div>
    </main>

    <div v-if="authVisible" class="modal-backdrop" role="presentation" @click.self="closeAuth">
      <section class="auth-modal" role="dialog" aria-modal="true" aria-labelledby="auth-title">
        <button class="auth-close-button" aria-label="Close" type="button" @click="closeAuth">
          <X :size="24" />
        </button>

        <div class="auth-modal-main">
          <div class="auth-form-panel">
            <div class="auth-brand">
              <span class="auth-brand-mark" aria-hidden="true">
                <img src="/favicon.svg" alt="" />
              </span>
              <strong>MergeOS</strong>
            </div>

            <header class="auth-copy">
              <h2 id="auth-title">
                <template v-if="authMode === 'register'">Create your account</template>
                <template v-else>Welcome back <span class="wave-mark">&#128075;</span></template>
              </h2>
              <p>
                {{ authMode === 'register'
                  ? 'Join thousands of builders and ship great software, faster with AI and top talent.'
                  : 'Log in to your MergeOS account to continue building and collaborating.' }}
              </p>
            </header>

            <div class="social-auth-row">
              <button type="button" @click="loginWithSocial('google')">
                <svg class="social-brand-logo google-mark" viewBox="0 0 48 48" aria-hidden="true" focusable="false">
                  <path fill="#EA4335" d="M24 9.5c3.54 0 6.71 1.22 9.21 3.6l6.85-6.85C35.9 2.38 30.47 0 24 0 14.62 0 6.51 5.38 2.56 13.22l7.98 6.19C12.43 13.72 17.74 9.5 24 9.5z" />
                  <path fill="#4285F4" d="M46.98 24.55c0-1.57-.15-3.09-.38-4.55H24v9.02h12.94c-.58 2.96-2.26 5.48-4.78 7.18l7.73 6c4.51-4.18 7.09-10.36 7.09-17.65z" />
                  <path fill="#FBBC05" d="M10.53 28.59A14.5 14.5 0 0 1 9.75 24c0-1.59.28-3.14.78-4.59l-7.98-6.19A23.9 23.9 0 0 0 0 24c0 3.86.92 7.5 2.56 10.78l7.97-6.19z" />
                  <path fill="#34A853" d="M24 48c6.48 0 11.93-2.13 15.89-5.81l-7.73-6c-2.15 1.45-4.92 2.3-8.16 2.3-6.26 0-11.57-4.22-13.47-9.91l-7.98 6.19C6.51 42.62 14.62 48 24 48z" />
                </svg>
                Continue with Google
              </button>
              <button type="button" :disabled="authBusy || !githubOAuthReady" @click="startGitHubLogin">
                <svg class="social-brand-logo github-mark" viewBox="0 0 24 24" aria-hidden="true" focusable="false">
                  <path fill="currentColor" fill-rule="evenodd" clip-rule="evenodd" d="M12 .5C5.65.5.5 5.65.5 12c0 5.08 3.29 9.39 7.86 10.91.58.11.79-.25.79-.56v-2.17c-3.2.7-3.88-1.36-3.88-1.36-.52-1.33-1.28-1.68-1.28-1.68-1.05-.72.08-.7.08-.7 1.16.08 1.77 1.19 1.77 1.19 1.03 1.76 2.7 1.25 3.36.96.1-.75.4-1.25.73-1.54-2.55-.29-5.24-1.28-5.24-5.68 0-1.26.45-2.28 1.19-3.09-.12-.29-.52-1.46.11-3.04 0 0 .97-.31 3.17 1.18a11.1 11.1 0 0 1 5.77 0c2.2-1.49 3.17-1.18 3.17-1.18.63 1.58.23 2.75.11 3.04.74.81 1.19 1.83 1.19 3.09 0 4.41-2.69 5.38-5.25 5.67.41.35.78 1.05.78 2.12v3.19c0 .31.21.67.8.56A11.51 11.51 0 0 0 23.5 12C23.5 5.65 18.35.5 12 .5z" />
                </svg>
                {{ githubOAuthReady ? 'Continue with GitHub' : 'Configure GitHub App' }}
              </button>
            </div>

            <div class="auth-divider">
              <span>or</span>
            </div>

            <form class="auth-form" @submit.prevent="submitAuth">
              <label v-if="authMode === 'register'" class="auth-field">
                <span>Full name</span>
                <div class="input-shell">
                  <User :size="18" />
                  <input v-model="authForm.name" autocomplete="name" placeholder="Enter your full name" />
                </div>
              </label>

              <label class="auth-field">
                <span>Email address</span>
                <div class="input-shell">
                  <Mail :size="18" />
                  <input v-model="authForm.email" autocomplete="email" placeholder="Enter your email address" type="email" />
                </div>
              </label>

              <label class="auth-field">
                <span>Password</span>
                <div class="input-shell">
                  <Lock :size="18" />
                  <input
                    v-model="authForm.password"
                    :autocomplete="authMode === 'register' ? 'new-password' : 'current-password'"
                    :placeholder="authMode === 'register' ? 'Create a password' : 'Enter your password'"
                    :type="showPassword ? 'text' : 'password'"
                  />
                  <button :aria-label="showPassword ? 'Hide password' : 'Show password'" type="button" @click="showPassword = !showPassword">
                    <Eye :size="18" />
                  </button>
                </div>
              </label>

              <label v-if="authMode === 'register'" class="auth-field">
                <span>Confirm password</span>
                <div class="input-shell">
                  <Lock :size="18" />
                  <input
                    v-model="authForm.confirm_password"
                    autocomplete="new-password"
                    placeholder="Confirm your password"
                    :type="showConfirmPassword ? 'text' : 'password'"
                  />
                  <button :aria-label="showConfirmPassword ? 'Hide confirm password' : 'Show confirm password'" type="button" @click="showConfirmPassword = !showConfirmPassword">
                    <Eye :size="18" />
                  </button>
                </div>
              </label>

              <div v-if="authMode === 'register'" class="auth-option-row compact">
                <label class="auth-check">
                  <input v-model="authTermsAccepted" type="checkbox" />
                  <span>I agree to the <button type="button" @click="showToast('Opening terms...')">Terms of Service</button> and <button type="button" @click="showToast('Opening privacy policy...')">Privacy Policy</button></span>
                </label>
              </div>
              <div v-else class="auth-option-row">
                <label class="auth-check">
                  <input v-model="authRememberMe" type="checkbox" />
                  <span>Remember me</span>
                </label>
                <button class="auth-link-button" type="button" @click="showToast('Password reset coming soon...')">Forgot password?</button>
              </div>

              <p v-if="errorMessage" class="modal-error">{{ errorMessage }}</p>

              <button class="auth-submit-button" :disabled="authBusy" type="submit">
                {{ authBusy ? 'Working...' : authMode === 'register' ? 'Create account' : 'Log in' }}
              </button>
            </form>

            <p class="auth-switch-line">
              {{ authMode === 'register' ? 'Already have an account?' : "Don't have an account?" }}
              <button type="button" @click="setAuthMode(authMode === 'register' ? 'login' : 'register')">
                {{ authMode === 'register' ? 'Log in' : 'Sign up' }}
              </button>
            </p>

            <div v-if="authMode === 'login'" class="auth-security-note">
              <span><ShieldCheck :size="18" /></span>
              <p>Protected by escrow. Your payments and data are always secure.</p>
            </div>
          </div>

          <aside class="auth-benefit-panel">
            <h3>{{ authMode === 'register' ? 'Why join MergeOS?' : 'Why builders love MergeOS' }}</h3>

            <div class="auth-benefit-list">
              <article v-for="benefit in authBenefits" :key="benefit.registerTitle" class="auth-benefit">
                <span :class="['auth-benefit-icon', benefit.tone]">
                  <component :is="benefit.icon" :size="28" />
                </span>
                <div>
                  <strong>{{ authMode === 'register' ? benefit.registerTitle : benefit.loginTitle }}</strong>
                  <p>{{ benefit.body }}</p>
                </div>
              </article>
            </div>

            <div v-if="authMode === 'register'" class="auth-orbit-visual" aria-hidden="true">
              <div class="code-card">
                <Code2 :size="18" />
                <span></span>
                <span></span>
                <span></span>
              </div>
              <div class="agent-card">
                <Sparkles :size="20" />
                <strong>AI Agent</strong>
                <CheckCircle2 :size="17" />
              </div>
              <div class="rating-card">
                <span class="mini-avatar">MRG</span>
                <strong>Wallet ready</strong>
                <small>Link after signup</small>
              </div>
              <span class="orbit-avatar left">MRG</span>
              <span class="orbit-avatar right">DAO</span>
              <span class="orbit-check top"><CheckCircle2 :size="18" /></span>
              <span class="orbit-check bottom"><CheckCircle2 :size="18" /></span>
            </div>

            <template v-else>
              <article class="auth-quote-card">
                <ShieldCheck :size="24" />
                <p>Create an account to save projects, link an MRG wallet, and record funding on the live ledger.</p>
                <div>
                  <span class="mini-avatar">MRG</span>
                  <strong>Account data appears after login</strong>
                  <small>Login to view profile details.</small>
                </div>
              </article>

              <div class="auth-trusted">
                <small>Live account areas</small>
                <div>
                  <strong>Projects</strong>
                  <strong>Wallet</strong>
                  <strong>Ledger</strong>
                  <strong>Tasks</strong>
                </div>
              </div>
            </template>
          </aside>
        </div>
      </section>
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted, onUnmounted, reactive, ref } from 'vue';
import {
  ArrowLeft,
  ArrowRight,
  BarChart3,
  Bell,
  Bot,
  Box,
  Bug,
  Calculator,
  CheckCircle2,
  ChevronDown,
  CircleDollarSign,
  Code2,
  Compass,
  CreditCard,
  Eye,
  Filter,
  FileCheck2,
  FolderKanban,
  GitBranch,
  GitPullRequest,
  Globe2,
  GripVertical,
  Home,
  LayoutDashboard,
  Link2,
  ListTodo,
  Lock,
  LockKeyhole,
  Mail,
  MessageCircle,
  MoreHorizontal,
  PenLine,
  Phone,
  Plus,
  Quote,
  RefreshCw,
  Rocket,
  Search,
  SendHorizontal,
  ShieldCheck,
  Share2,
  Sparkles,
  Star,
  Trophy,
  UploadCloud,
  User,
  UsersRound,
  X,
  Zap,
} from '@lucide/vue';

const hasWindow = typeof window !== 'undefined';
const TOKEN_RATE_PER_USD = 100;
const DASHBOARD_REFRESH_MS = 5000;
const publicPagePaths = {
  home: '/',
  product: '/product',
  solutions: '/solutions',
  marketplace: '/marketplace',
  'how-it-works': '/how-it-works',
  ledger: '/ledger',
};
const publicPageNames = new Set(Object.keys(publicPagePaths));
const projectWizardStepPaths = {
  1: '/project/new',
  2: '/project/new/scope',
  3: '/project/new/budget',
  4: '/project/new/review',
};
const projectWizardStagePaths = {
  funding: '/project/new/funding',
  success: '/project/new/success',
};

const props = defineProps({
  initialPath: { type: String, default: '' },
});

function normalizeRoutePath(path = '/') {
  const pathname = String(path || '/').split('?')[0].split('#')[0] || '/';
  const normalized = pathname.replace(/\/+$/, '') || '/';
  return normalized.startsWith('/') ? normalized : `/${normalized}`;
}

function normalizePublicPage(page = 'home') {
  return publicPageNames.has(page) ? page : 'home';
}

function publicPageFromPath(path = '/') {
  const normalizedPath = normalizeRoutePath(path);
  const match = Object.entries(publicPagePaths).find(([, routePath]) => routePath === normalizedPath);
  return match?.[0] || 'home';
}

function publicPathForPage(page = 'home') {
  return publicPagePaths[normalizePublicPage(page)] || '/';
}

function normalizeProjectWizardStep(step = 1) {
  return Math.min(4, Math.max(1, Number(step) || 1));
}

function projectWizardRouteFromPath(path = '/') {
  const normalizedPath = normalizeRoutePath(path);
  const stepMatch = Object.entries(projectWizardStepPaths).find(([, routePath]) => routePath === normalizedPath);
  if (stepMatch) return { stage: 'setup', step: Number(stepMatch[0]) };
  if (normalizedPath === '/project/new/details' || normalizedPath === '/projects/new') return { stage: 'setup', step: 1 };
  if (normalizedPath === projectWizardStagePaths.funding) return { stage: 'funding', step: 4 };
  if (normalizedPath === projectWizardStagePaths.success) return { stage: 'success', step: 4 };
  return null;
}

function projectWizardPathForState(stage = 'setup', step = 1) {
  if (stage === 'funding') return projectWizardStagePaths.funding;
  if (stage === 'success') return projectWizardStagePaths.success;
  return projectWizardStepPaths[normalizeProjectWizardStep(step)] || projectWizardStepPaths[1];
}

function getBrowserStorage() {
  if (!hasWindow || !('localStorage' in window)) {
    return null;
  }
  try {
    return window.localStorage;
  } catch {
    return null;
  }
}

const browserStorage = getBrowserStorage();

function readStoredToken() {
  try {
    return browserStorage?.getItem('mergeos_token') || '';
  } catch {
    return '';
  }
}

function writeStoredToken(value) {
  try {
    browserStorage?.setItem('mergeos_token', value);
  } catch {
    // Storage can be disabled in embedded browsers; auth still works for this session.
  }
}

function removeStoredToken() {
  try {
    browserStorage?.removeItem('mergeos_token');
  } catch {
    // Ignore storage failures so logout never leaves the UI stuck.
  }
}

const token = ref(readStoredToken());
const user = ref(null);
const authVisible = ref(false);
const authMode = ref('login');
const authBusy = ref(false);
const authRememberMe = ref(false);
const authTermsAccepted = ref(true);
const errorMessage = ref('');
const showPassword = ref(false);
const showConfirmPassword = ref(false);
const toastMessage = ref('');
let toastTimer = 0;

const initialRoutePath = props.initialPath || (hasWindow ? window.location.pathname : '/');
const initialProjectWizardRoute = projectWizardRouteFromPath(initialRoutePath);
const initialPublicPage = publicPageFromPath(initialRoutePath);
const publicPage = ref(initialPublicPage);
const publicModeVisible = ref(Boolean(initialProjectWizardRoute) || initialPublicPage !== 'home');

const projectWizardVisible = ref(Boolean(initialProjectWizardRoute));
const projectWizardStage = ref(initialProjectWizardRoute?.stage || 'setup');
const projectWizardStep = ref(initialProjectWizardRoute?.step || 1);
const projectFundingAmount = ref('');
const projectPaymentMethod = ref('Credit / Debit card');
const projectPaymentBusy = ref(false);
const projectPaymentError = ref('');
const pendingProjectPaymentAfterAuth = ref(false);
const authReturnToProjectWizard = ref(false);
const fundedProject = ref(null);
const runtimeConfig = ref(null);
const ledgerRawEntries = ref([]);
const ledgerProjects = ref([]);
const ledgerLoading = ref(false);
const ledgerError = ref('');
const marketplaceData = ref({
  stats: {},
  projects: [],
  contributors: [],
  agents: [],
});
const marketplaceLoading = ref(true);
const marketplaceError = ref('');
const marketplaceSearch = ref('');
const activeMarketplaceCategory = ref('All');
const dashboardProjects = ref([]);
const dashboardTasks = ref([]);
const dashboardLedgerEntries = ref([]);
const dashboardLoading = ref(false);
const dashboardError = ref('');
const dashboardSearch = ref('');
const selectedDashboardProjectID = ref('');
const priceEvaluation = ref(null);
const priceEvaluationBusy = ref(false);
const priceEvaluationError = ref('');
const repoImportBusy = ref(false);
const repoImportError = ref('');
const repoImportResult = ref(null);
let dashboardRefreshTimer = 0;

const projectSetupForm = reactive({
  title: '',
  shortDescription: '',
  projectType: '',
  techStack: '',
  repoUrl: '',
  overview: '',
  requirements: '',
  budgetAmount: '',
  budgetType: 'Fixed price',
  startDate: '',
  deadline: '',
  fundingMethod: 'Escrow',
  visibility: 'Public',
  allowAgents: true,
  skills: '',
  complexity: 'Medium',
  constraints: '',
});

const aiEvaluationResult = ref(null);
const aiEvaluationLoading = ref(false);
const aiEvaluationError = ref('');

const projectDeliverables = ref(['']);

const projectDeliverablePlaceholders = [
  'Describe a key deliverable',
  'Describe the next deliverable',
  'Add integration or workflow deliverables',
  'Add QA, launch, or handoff deliverables',
];

const projectSetupSteps = [
  {
    number: 1,
    label: 'Project details',
    title: 'Let\'s start with the basics',
    description: 'Describe your project',
    helper: 'Tell us about your idea and we will help you build it with the right people and AI.',
  },
  {
    number: 2,
    label: 'Scope & requirements',
    title: 'Define the work to be done',
    description: 'Define what you need',
    helper: 'Write the goals, key deliverables, and constraints that contributors should understand.',
  },
  {
    number: 3,
    label: 'Budget & timeline',
    title: 'Set budget, deadline, and funding method',
    description: 'Set budget and deadline',
    helper: 'Choose how much you want to spend and how payment should be protected.',
  },
  {
    number: 4,
    label: 'Review & publish',
    title: 'Review your project details and publish',
    description: 'Review and post your project',
    helper: 'Confirm everything before publishing your project to top talent.',
  },
];

const projectTypeOptions = [
  { label: 'Web Development', caption: 'Web apps', icon: Globe2 },
  { label: 'Repo Issue Fix', caption: 'Existing repo', icon: Bug },
  { label: 'Mobile Development', caption: 'iOS and Android', icon: Compass },
  { label: 'AI / ML', caption: 'Agents and models', icon: Bot },
  { label: 'Smart Contract', caption: 'Web3', icon: Link2 },
  { label: 'Other', caption: 'Custom work', icon: MoreHorizontal },
];

const budgetTypeOptions = [
  { label: 'Fixed price', icon: CircleDollarSign },
  { label: 'Range', icon: BarChart3 },
  { label: 'Hourly', icon: CreditCard },
];

const fundingMethodOptions = [
  { label: 'Escrow', caption: 'Funds are held securely until work is completed.', icon: ShieldCheck },
  { label: 'Milestone based', caption: 'Pay in stages as milestones are completed.', icon: GitBranch },
  { label: 'Upfront payment', caption: 'Pay the full amount upfront.', icon: CreditCard },
  { label: 'Custom', caption: 'Discuss payment terms with contributors.', icon: MoreHorizontal },
];

const fundingAmountOptions = [
  { amount: 500, tokens: 50000 },
  { amount: 1000, tokens: 100000 },
  { amount: 2000, tokens: 200000, popular: true },
  { amount: 5000, tokens: 500000 },
];

const paymentMethodOptions = [
  { label: 'Credit / Debit card', caption: 'Visa, Mastercard, Amex', icon: CreditCard },
  { label: 'USDC', caption: 'Ethereum, Polygon, Arbitrum', icon: CircleDollarSign },
  { label: 'Bank transfer', caption: 'Worldwide bank transfer', icon: FileCheck2 },
  { label: 'PayPal', caption: 'Fast and secure', icon: CreditCard },
];

const howItWorks = [
  'You post your project and fund escrow.',
  'We match you with top talent or AI agents.',
  'Work happens transparently with updates.',
  'You review, approve, and release payment.',
  'Project delivered with full ownership.',
];

const scopeTips = [
  'Be specific about what you need.',
  'List key features and deliverables.',
  'Add references or examples.',
  'Mention any technical constraints.',
  'Clear scope equals better proposals.',
];

const sparklineHeights = [28, 36, 44, 40, 58, 47, 66, 50, 61, 72, 46, 82];

const successNextSteps = [
  {
    step: 1,
    title: 'We notify top talent',
    body: 'We will match your project with relevant talent.',
    icon: UsersRound,
  },
  {
    step: 2,
    title: 'Receive proposals',
    body: 'Top talent will send you their proposals.',
    icon: FileCheck2,
  },
  {
    step: 3,
    title: 'Review & hire',
    body: 'Review proposals, chat, and hire the best fit.',
    icon: MessageCircle,
  },
  {
    step: 4,
    title: 'Start your project',
    body: 'Work begins and funds are held safely in escrow.',
    icon: Rocket,
  },
];

const postPaymentActions = [
  'Complete your project details',
  'Invite team members',
  'Boost your project',
  'Explore your dashboard',
];

const currentProjectStep = computed(() => projectSetupSteps.find((step) => step.number === projectWizardStep.value) || projectSetupSteps[0]);
const visibleDeliverables = computed(() => {
  const items = projectDeliverables.value.map((item) => item.trim()).filter(Boolean);
  return items;
});
const projectTitleLabel = computed(() => projectSetupForm.title.trim() || 'Untitled project');
const projectTypeLabel = computed(() => projectSetupForm.projectType || 'Select a project type');
const projectDescriptionLabel = computed(() => projectSetupForm.shortDescription.trim() || 'Add a short project description');
const projectDeliverablesPlaceholder = 'No deliverables added yet';
const projectDeliverableCountLabel = computed(() =>
  visibleDeliverables.value.length ? `${visibleDeliverables.value.length} items` : projectDeliverablesPlaceholder,
);
const repoImportedIssues = computed(() => Array.isArray(repoImportResult.value?.issues) ? repoImportResult.value.issues : []);
const repoImportedEstimateCents = computed(() =>
  repoImportedIssues.value.reduce((total, issue) => total + (Number(issue.estimated_cents) || 0), 0),
);
const projectBudgetAmount = computed(() => Math.max(0, Number(projectSetupForm.budgetAmount) || 0));
const projectBudgetLow = computed(() => {
  if (aiEvaluationResult.value && projectBudgetAmount.value === mrgFromUSD(Math.round((aiEvaluationResult.value.suggested_low + aiEvaluationResult.value.suggested_high) / 2))) {
    return mrgFromUSD(aiEvaluationResult.value.suggested_low);
  }
  return Math.round(projectBudgetAmount.value * 0.85);
});
const projectBudgetHigh = computed(() => {
  if (aiEvaluationResult.value && projectBudgetAmount.value === mrgFromUSD(Math.round((aiEvaluationResult.value.suggested_low + aiEvaluationResult.value.suggested_high) / 2))) {
    return mrgFromUSD(aiEvaluationResult.value.suggested_high);
  }
  return Math.round(projectBudgetAmount.value * 1.25);
});
const projectPlatformFeeLow = computed(() => Math.round(projectBudgetLow.value * 0.08));
const projectPlatformFeeHigh = computed(() => Math.round(projectBudgetHigh.value * 0.08));
const projectEscrowFeeLow = computed(() => Math.round(projectBudgetLow.value * 0.02));
const projectEscrowFeeHigh = computed(() => Math.round(projectBudgetHigh.value * 0.02));
const projectEstimatedLow = computed(() => projectBudgetLow.value + projectPlatformFeeLow.value + projectEscrowFeeLow.value);
const projectEstimatedHigh = computed(() => projectBudgetHigh.value + projectPlatformFeeHigh.value + projectEscrowFeeHigh.value);
const projectEstimatedTotal = computed(() => Math.round(projectBudgetAmount.value * 1.1));
const projectFundingPlatformFee = computed(() => Math.round((Number(projectFundingAmount.value) || 0) * 0.08));
const projectFundingEscrowFee = computed(() => Math.round((Number(projectFundingAmount.value) || 0) * 0.02));
const projectTokenAmount = computed(() => Math.round((Number(projectFundingAmount.value) || 0) * TOKEN_RATE_PER_USD));
const projectInitial = computed(() => (projectSetupForm.title.trim().charAt(0) || 'M').toUpperCase());
const projectBudgetRangeLabel = computed(() =>
  projectBudgetAmount.value > 0 ? `${formatMRG(projectBudgetLow.value)} - ${formatMRG(projectBudgetHigh.value)}` : 'Budget not set',
);
const projectBudgetSummaryLabel = computed(() =>
  projectBudgetAmount.value > 0 ? `${formatMRG(projectBudgetAmount.value)} (${projectSetupForm.budgetType})` : 'Budget not set',
);
const projectEstimatedTotalLabel = computed(() => (projectBudgetAmount.value > 0 ? formatMRG(projectEstimatedTotal.value) : 'Not calculated yet'));
const projectEstimatedRangeLabel = computed(() =>
  projectBudgetAmount.value > 0 ? `${formatMRG(projectEstimatedLow.value)} - ${formatMRG(projectEstimatedHigh.value)}` : 'Not calculated yet',
);
const projectFundingAmountLabel = computed(() => (Number(projectFundingAmount.value) > 0 ? `${formatMoney(projectFundingAmount.value)} USD` : 'Choose amount'));
const projectTokenAmountLabel = computed(() => (projectTokenAmount.value > 0 ? `${projectTokenAmount.value} ${tokenSymbol.value}` : 'Choose an amount'));
const projectDurationDays = computed(() => {
  if (!projectSetupForm.startDate || !projectSetupForm.deadline) return 0;
  const start = Date.parse(`${projectSetupForm.startDate}T00:00:00Z`);
  const end = Date.parse(`${projectSetupForm.deadline}T00:00:00Z`);
  if (!Number.isFinite(start) || !Number.isFinite(end) || end < start) return 0;
  return Math.max(1, Math.round((end - start) / 86400000));
});
const projectTimelineLabel = computed(() => {
  const start = formatDateInputLabel(projectSetupForm.startDate);
  const deadline = formatDateInputLabel(projectSetupForm.deadline);
  if (start && deadline) return `${start} - ${deadline}${projectDurationDays.value ? ` (${projectDurationDays.value} days)` : ''}`;
  if (start) return `Starts ${start}`;
  if (deadline) return `Due ${deadline}`;
  return 'Timeline not set';
});
const projectQualityScore = computed(() => {
  const filledSections = [
    projectSetupForm.title.trim(),
    projectSetupForm.shortDescription.trim(),
    projectSetupForm.projectType,
    projectSetupForm.overview.trim(),
    visibleDeliverables.value.length ? 'deliverables' : '',
    projectBudgetAmount.value > 0 ? 'budget' : '',
    projectSetupForm.deadline,
  ].filter(Boolean).length;

  return filledSections ? Math.round((filledSections / 7) * 100) : 0;
});
const projectQualityScoreLabel = computed(() => (projectQualityScore.value ? String(projectQualityScore.value) : '--'));
const projectQualityCopy = computed(() => {
  if (!projectQualityScore.value) return 'Complete the brief to generate a quality check.';
  if (projectQualityScore.value >= 75) return 'Your brief has enough detail for a strong review.';
  return 'Keep adding scope, budget, and timing details to improve the brief.';
});
const wizardIntroCopy = computed(() => {
  if (projectWizardStage.value === 'funding') {
    return 'Add escrow funding so contributors can send stronger proposals.';
  }

  if (projectWizardStage.value === 'success') {
    return 'Your project is funded and ready for matching.';
  }

  return 'Tell us about your project so we can match you with the right talent or AI agents.';
});
const footerStepNumber = computed(() => (projectWizardStage.value === 'setup' ? projectWizardStep.value : 4));
const footerProgress = computed(() => {
  if (projectWizardStage.value === 'success') {
    return 100;
  }

  return Math.min(100, footerStepNumber.value * 25);
});
const footerProtectionCopy = computed(() => {
  if (projectWizardStage.value === 'success') {
    return 'Your project is funded and ready to receive proposals.';
  }

  if (projectWizardStage.value === 'funding') {
    return 'Your payment is protected by escrow.';
  }

  return 'Your project is protected by escrow after funding.';
});
const projectFooterSteps = computed(() =>
  projectSetupSteps.map((step) => ({
    number: step.number,
    label: step.label.split(' ')[0],
    active: projectWizardStage.value === 'setup' && projectWizardStep.value === step.number,
    done: projectWizardStage.value !== 'setup' || projectWizardStep.value > step.number,
  })),
);

const authForm = reactive({
  name: '',
  company_name: '',
  email: '',
  password: '',
  confirm_password: '',
});

const defaultLoginAuth = {
  name: '',
  company_name: '',
  email: '',
  password: '',
  confirm_password: '',
};

const defaultRegisterAuth = {
  name: '',
  company_name: '',
  email: '',
  password: '',
  confirm_password: '',
};

const authBenefits = [
  {
    icon: ShieldCheck,
    tone: 'green',
    registerTitle: 'Secure & Escrow Protected',
    loginTitle: 'Secure Escrow',
    body: 'All payments are protected with escrow until the work is completed.',
  },
  {
    icon: Zap,
    tone: 'purple',
    registerTitle: 'AI-Powered Matching',
    loginTitle: 'AI-Powered Matching',
    body: 'We match you with the best talent or AI agents for your project.',
  },
  {
    icon: UsersRound,
    tone: 'yellow',
    registerTitle: 'Top Global Talent',
    loginTitle: 'Top Global Talent',
    body: 'Access thousands of verified developers and specialists.',
  },
  {
    icon: Rocket,
    tone: 'blue',
    registerTitle: 'Ship Faster',
    loginTitle: 'Ship Faster',
    body: 'Collaborate seamlessly and ship high-quality software.',
  },
];

const ledgerTrustItems = [
  {
    icon: ShieldCheck,
    tone: 'green',
    title: '100% Transparent',
    body: 'On-chain verified',
  },
  {
    icon: Bell,
    tone: 'blue',
    title: 'Real-time Updates',
    body: 'Live activity stream',
  },
  {
    icon: LockKeyhole,
    tone: 'green',
    title: 'Verified by MergeOS',
    body: 'Escrow-protected',
  },
];

const ledgerTabs = ['All Activity', 'Escrow & Payments', 'Tasks & PRs', 'Milestones', 'AI Actions', 'Token Events'];

const ledgerVerificationChecks = [
  'Escrow-protected payments',
  'On-chain transaction verification',
  'Code & delivery verification',
  'Dispute resolution system',
];

const ledgerProjectIndex = computed(() => {
  const index = new Map();
  for (const project of ledgerProjects.value) {
    index.set(project.id, project);
  }
  if (fundedProject.value) {
    index.set(fundedProject.value.id, fundedProject.value);
  }
  return index;
});

const tokenSymbol = computed(() => runtimeConfig.value?.token_symbol || 'MRG');
const githubOAuthReady = computed(() => Boolean(runtimeConfig.value?.github_oauth_ready && runtimeConfig.value?.github_oauth_client_id));
const projectPaymentAmountCents = computed(() => Math.round(Math.max(100, Number(projectFundingAmount.value) || 100) * 100));
const projectPaymentButtonLabel = computed(() => {
  if (projectPaymentBusy.value) {
    return 'Recording payment...';
  }
  return user.value ? 'Add funds & get tokens' : 'Log in to pay';
});
const successProjectTitle = computed(() => fundedProject.value?.title || projectTitleLabel.value);
const successPaymentReference = computed(() => fundedProject.value?.payment_reference || '');

const ledgerEvents = computed(() => ledgerRawEntries.value.slice().reverse().map(mapLedgerEntry));
const ledgerMintedTokenTotal = computed(() =>
  ledgerRawEntries.value
    .filter((entry) => entry.type === 'token_mint')
    .reduce((total, entry) => total + tokenAmountFromCents(entry.amount_cents), 0),
);
const ledgerVerifiedFundingCents = computed(() =>
  ledgerRawEntries.value
    .filter((entry) => entry.type === 'payment_verified')
    .reduce((total, entry) => total + (Number(entry.amount_cents) || 0), 0),
);
const publicVerifiedFundingCents = computed(() => {
  const ledgerFunding = ledgerVerifiedFundingCents.value;
  if (ledgerFunding > 0) return ledgerFunding;
  return Number(marketplaceStats.value.total_budget_cents) || 0;
});
const publicMintedTokenTotal = computed(() => {
  if (ledgerMintedTokenTotal.value > 0) return ledgerMintedTokenTotal.value;
  return tokenAmountFromCents(publicVerifiedFundingCents.value);
});
const ledgerProjectCount = computed(() => {
  const ids = new Set();
  for (const entry of ledgerRawEntries.value) {
    const id = extractProjectID(entry);
    if (id) ids.add(id);
  }
  return ids.size;
});
const publicProjectCount = computed(() =>
  ledgerProjectCount.value
  || Number(marketplaceStats.value.project_count)
  || ledgerProjects.value.length
  || marketplaceData.value.projects.length
  || 0,
);
const ledgerLiveStats = computed(() => [
  { value: String(ledgerRawEntries.value.length), label: 'Ledger entries' },
  { value: formatPublicTokenAmount(publicMintedTokenTotal.value), label: 'Tokens minted' },
  { value: formatLedgerMRGFromCents(publicVerifiedFundingCents.value), label: 'Verified funding' },
  { value: String(ledgerRawEntries.value.filter((entry) => entry.type === 'task_payment').length), label: 'Payments released' },
]);
const ledgerTrendingProjects = computed(() => {
  const grouped = new Map();
  for (const entry of ledgerRawEntries.value) {
    const projectID = extractProjectID(entry);
    if (!projectID) continue;
    const project = ledgerProjectIndex.value.get(projectID);
    const current = grouped.get(projectID) || {
      initial: projectInitialFor(project?.title || projectID),
      tone: projectToneFor(projectID),
      title: project?.title || `Project ${projectID.slice(-6)}`,
      company: project?.company_name || project?.client_name || 'MergeOS client',
      escrowCents: 0,
      contributors: 0,
      prs: 0,
    };
    if (entry.type === 'payment_verified') {
      current.escrowCents += Number(entry.amount_cents) || 0;
    }
    if (entry.type === 'task_reserve') {
      current.contributors += 1;
    }
    if (entry.type === 'task_payment') {
      current.prs += 1;
    }
    grouped.set(projectID, current);
  }
  return Array.from(grouped.values()).slice(0, 4).map((project) => ({
    ...project,
    escrow: `${formatLedgerMRGFromCents(project.escrowCents)} Escrow`,
  }));
});
const ledgerChainRows = computed(() => [
  { label: 'Token', value: tokenSymbol.value },
  { label: 'Payment mode', value: paymentModeLabel(runtimeConfig.value?.payment_mode) },
  { label: 'Repo provider', value: repoProviderLabel(runtimeConfig.value?.repo_provider) },
]);
const ledgerFooterStats = computed(() => [
  { value: formatLedgerMRGFromCents(publicVerifiedFundingCents.value), label: 'Verified funding' },
  { value: String(publicProjectCount.value), label: 'Funded projects' },
  { value: formatPublicTokenAmount(publicMintedTokenTotal.value), label: 'Tokens minted' },
  { value: String(ledgerRawEntries.value.length), label: 'Ledger entries' },
]);

const marketplaceFilters = ['Category', 'Budget', 'Delivery time'];

const marketplaceProjectPalettes = [
  { accent: '#0f9f78', soft: '#e9f8f1', icon: Globe2, badgeTone: 'green', avatarTone: 'avatar-green' },
  { accent: '#2563eb', soft: '#eff6ff', icon: BarChart3, badgeTone: 'purple', avatarTone: 'avatar-blue' },
  { accent: '#d97706', soft: '#fffbeb', icon: Code2, badgeTone: 'yellow', avatarTone: 'avatar-rose' },
  { accent: '#7c3aed', soft: '#f5f3ff', icon: Bot, badgeTone: 'purple', avatarTone: 'avatar-slate' },
];

const marketplaceAvatarTones = ['avatar-green', 'avatar-blue', 'avatar-rose', 'avatar-slate'];

const emptyMarketplaceProject = {
  id: 'empty-marketplace',
  icon: FolderKanban,
  badge: 'NO LIVE DATA',
  badgeTone: 'green',
  title: 'No funded projects yet',
  body: 'Post and fund a project to publish a real marketplace listing.',
  tags: ['Escrow', 'Tasks', 'Ledger'],
  extra: 0,
  budget: '0 MRG',
  timeline: 'Waiting for first project',
  client: 'MergeOS',
  clientInitials: 'M',
  avatarTone: 'avatar-green',
  taskLabel: '0 tasks',
  verified: true,
  accent: '#0f9f78',
  soft: '#e9f8f1',
};

const marketplaceStats = computed(() => marketplaceData.value?.stats || {});
const marketplaceTrustItems = computed(() => [
  {
    icon: ShieldCheck,
    tone: 'green',
    title: formatPublicMRGFromCents(marketplaceStats.value.total_budget_cents),
    body: 'Verified escrow',
  },
  {
    icon: ListTodo,
    tone: 'blue',
    title: `${Number(marketplaceStats.value.open_task_count) || 0} open tasks`,
    body: 'Ready for builders',
  },
  {
    icon: Zap,
    tone: 'yellow',
    title: `${Number(marketplaceStats.value.ledger_entry_count) || 0} ledger entries`,
    body: 'Public proof',
  },
]);

const homeLiveStats = computed(() => [
  { value: String(Number(marketplaceStats.value.project_count) || marketplaceData.value.projects.length || 0), label: 'Funded projects' },
  { value: String(Number(marketplaceStats.value.open_task_count) || 0), label: 'Open tasks' },
  { value: formatPublicMRGFromCents(marketplaceStats.value.total_budget_cents), label: 'Verified escrow' },
  { value: formatPublicTokenAmount(publicMintedTokenTotal.value), label: 'Tokens minted' },
]);

const homeWorkflowCards = [
  {
    title: 'Product',
    body: 'Run project intake, escrow funding, repo handoff, task splitting, and proof ledger from one flow.',
    cta: 'View product',
    icon: Rocket,
    tone: 'green',
    action: { page: 'product' },
  },
  {
    title: 'Solutions',
    body: 'Choose human talent, AI agents, or hybrid delivery for SaaS builds, repo fixes, and marketplace tasks.',
    cta: 'Explore solutions',
    icon: Compass,
    tone: 'blue',
    action: { page: 'solutions' },
  },
  {
    title: 'Marketplace',
    body: 'Browse live funded projects, open tasks, contributor signals, and AI work queues before signing in.',
    cta: 'Find talent',
    icon: UsersRound,
    tone: 'purple',
    action: { page: 'marketplace' },
  },
  {
    title: 'How it works',
    body: 'Post work, fund escrow, mint tokens for the payer, match talent, and release payouts with ledger proof.',
    cta: 'See workflow',
    icon: GitPullRequest,
    tone: 'amber',
    action: { page: 'how-it-works' },
  },
];

const homeTalentRows = [
  { title: 'Human contributors', body: 'Reviewed builders for scoped project work and repo issues.', icon: User, tone: 'green' },
  { title: 'AI agents', body: 'Specialized agents for frontend, ledger, QA, and DevOps tasks.', icon: Bot, tone: 'purple' },
  { title: 'Hybrid delivery', body: 'AI speed with human review, escrow, and acceptance criteria.', icon: ShieldCheck, tone: 'blue' },
];

const publicInfoPages = {
  product: {
    eyebrow: 'PRODUCT',
    title: 'Project delivery with escrow and proof built in',
    body: 'MergeOS turns a project brief or existing repo issue list into funded tasks, verified payments, token mint logs, and contributor-ready work.',
    actions: [
      { label: 'Start a project', primary: true, icon: ArrowRight, command: 'project' },
      { label: 'View ledger', icon: Link2, page: 'ledger' },
    ],
    summary: [
      { label: 'Project wizard', value: 'Details, scope, budget, review, funding', icon: FolderKanban, tone: 'green' },
      { label: 'Repo issue scoring', value: 'Import repo issues and score work items', icon: Bug, tone: 'amber' },
      { label: 'Ledger proof', value: 'Payment verified and token_mint logs', icon: ShieldCheck, tone: 'blue' },
    ],
    features: [
      { title: 'Start from a brief', body: 'Create a full project from details, scope, budget, and timeline screens.', icon: FileCheck2, tone: 'green' },
      { title: 'Start from a repo', body: 'Use an existing repository and load issues for scoring and task planning.', icon: GitBranch, tone: 'blue' },
      { title: 'Fund the right project', body: 'Payment is only allowed after login so every ledger record ties to the payer and project.', icon: LockKeyhole, tone: 'purple' },
    ],
  },
  solutions: {
    eyebrow: 'SOLUTIONS',
    title: 'Match the work to the right delivery model',
    body: 'Use MergeOS for complete builds, issue fixing, agent-assisted implementation, escrow-protected work, and verified payout workflows.',
    actions: [
      { label: 'Find talent', primary: true, icon: UsersRound, page: 'marketplace' },
      { label: 'Start a project', icon: ArrowRight, command: 'project' },
    ],
    summary: [
      { label: 'For founders', value: 'Ship complete products with escrow', icon: Rocket, tone: 'green' },
      { label: 'For repo owners', value: 'Turn issues into scored bounty tasks', icon: Bug, tone: 'amber' },
      { label: 'For teams', value: 'Blend contributors and AI agents', icon: UsersRound, tone: 'blue' },
    ],
    features: [
      { title: 'Complete project delivery', body: 'Post a product request and fund it through escrow-backed workflows.', icon: FolderKanban, tone: 'green' },
      { title: 'Existing repo fixes', body: 'Import a repo, load issues, score priority, and publish fix orders.', icon: GitPullRequest, tone: 'blue' },
      { title: 'AI agent support', body: 'Route focused implementation tasks to specialized agents with human review paths.', icon: Bot, tone: 'purple' },
    ],
  },
  'how-it-works': {
    eyebrow: 'HOW IT WORKS',
    title: 'From brief to funded, verifiable work',
    body: 'The public flow starts without auth. Login is required only when money moves, so payment, token mint, and project records stay correct.',
    actions: [
      { label: 'Post a project', primary: true, icon: ArrowRight, command: 'project' },
      { label: 'Browse marketplace', icon: UsersRound, page: 'marketplace' },
    ],
    summary: [
      { label: '1. Describe', value: 'Project brief or repo issues', icon: FileCheck2, tone: 'green' },
      { label: '2. Fund', value: 'Login, pay, mint payer tokens', icon: LockKeyhole, tone: 'blue' },
      { label: '3. Verify', value: 'Ledger logs and task payouts', icon: ShieldCheck, tone: 'purple' },
    ],
    features: [
      { title: 'No forced auth upfront', body: 'Visitors can view home, marketplace, talent signals, and product pages before login.', icon: Globe2, tone: 'green' },
      { title: 'Auth before payment', body: 'Checkout gates login and attaches payment to the correct user and project.', icon: LockKeyhole, tone: 'blue' },
      { title: 'Real ledger logs', body: 'Ledger Logs shows backend payment_verified and token_mint records from the API.', icon: Link2, tone: 'purple' },
    ],
  },
};

const publicInfoPage = computed(() => publicInfoPages[publicPage.value] || null);

const marketplaceCategories = computed(() => {
  const categories = ['All'];
  const seen = new Set(categories);
  for (const project of marketplaceData.value.projects || []) {
    for (const tag of project.tags || []) {
      const label = toTitleLabel(tag);
      if (!label || seen.has(label)) continue;
      seen.add(label);
      categories.push(label);
    }
  }
  return categories.slice(0, 10);
});
const marketplaceProjectsView = computed(() => {
  const query = marketplaceSearch.value.toLowerCase();
  const active = activeMarketplaceCategory.value;
  return (marketplaceData.value.projects || [])
    .map(mapMarketplaceProject)
    .filter((project) => {
      const matchesCategory = active === 'All' || project.tags.some((tag) => toTitleLabel(tag) === active);
      const matchesSearch = !query || marketplaceSearchHaystack(project).includes(query);
      return matchesCategory && matchesSearch;
    });
});
const marketplaceSummaryLabel = computed(() => {
  const stats = marketplaceStats.value;
  const projects = Number(stats.project_count) || marketplaceData.value.projects?.length || 0;
  const tasks = Number(stats.open_task_count) || 0;
  return `${projects} live projects · ${tasks} open tasks · ${formatPublicMRGFromCents(stats.total_budget_cents)} verified`;
});
const marketplaceHeroProject = computed(() => marketplaceProjectsView.value[0] || emptyMarketplaceProject);
const marketplaceContributorsView = computed(() =>
  (marketplaceData.value.contributors || []).map((contributor, index) => ({
    rank: index + 1,
    workerId: contributor.worker_id || contributor.name || `contributor-${index}`,
    initials: initialsFor(contributor.name || contributor.worker_id || 'FW'),
    name: contributor.name || contributor.worker_id || 'Contributor',
    role: contributor.agent_type ? toTitleLabel(contributor.agent_type) : toTitleLabel(contributor.kind || 'human contributor'),
    earned: formatPublicMRGFromCents(contributor.earned_cents),
    tone: marketplaceAvatarTones[index % marketplaceAvatarTones.length],
  })),
);
const marketplaceAgentsView = computed(() =>
  (marketplaceData.value.agents || []).map((agent, index) => ({
    type: agent.type || `agent-${index}`,
    icon: marketplaceAgentIcon(agent.type),
    title: agent.title || toTitleLabel(agent.type || 'AI Agent'),
    body: `${Number(agent.open_task_count) || 0} open tasks · ${formatPublicMRGFromCents(agent.budget_cents)} pool`,
    tone: ['green', 'blue', 'yellow', 'red'][index % 4],
  })),
);
const marketplaceHeroAgent = computed(() => marketplaceAgentsView.value[0] || {
  type: 'empty-agent',
  icon: Bot,
  title: 'No open agent queue',
  body: 'Funded AI-scoped tasks will appear here.',
  tone: 'green',
});
const dashboardSortedProjects = computed(() =>
  dashboardProjects.value.slice().sort((a, b) => new Date(b.created_at || 0) - new Date(a.created_at || 0)),
);
const dashboardProjectList = computed(() => {
  const query = dashboardSearch.value.toLowerCase();
  if (!query) return dashboardSortedProjects.value;
  return dashboardSortedProjects.value.filter((project) => [
    project.title,
    project.brief,
    project.company_name,
    project.client_name,
    project.bounty_repo_name,
    project.repo_provider,
  ].filter(Boolean).join(' ').toLowerCase().includes(query));
});
const dashboardSelectedProject = computed(() => {
  if (!dashboardSortedProjects.value.length) return null;
  return dashboardSortedProjects.value.find((project) => project.id === selectedDashboardProjectID.value) || dashboardSortedProjects.value[0];
});
const dashboardSelectedTasks = computed(() => {
  const project = dashboardSelectedProject.value;
  if (!project) return [];
  const rows = new Map();
  for (const task of project.tasks || []) {
    rows.set(task.id, task);
  }
  for (const task of dashboardTasks.value) {
    if (task.project_id === project.id) {
      rows.set(task.id, task);
    }
  }
  return Array.from(rows.values()).sort((a, b) => (Number(a.issue_number) || 0) - (Number(b.issue_number) || 0));
});
const dashboardAcceptedTasks = computed(() => dashboardSelectedTasks.value.filter((task) => task.status === 'accepted'));
const dashboardOpenTasks = computed(() => dashboardSelectedTasks.value.filter((task) => task.status !== 'accepted'));
const dashboardProgress = computed(() => {
  const total = dashboardSelectedTasks.value.length;
  if (!total) return 0;
  return Math.round((dashboardAcceptedTasks.value.length / total) * 100);
});
const dashboardSpentCents = computed(() => dashboardAcceptedTasks.value.reduce((total, task) => total + (Number(task.reward_cents) || 0), 0));
const dashboardRemainingCents = computed(() => Math.max(0, (Number(dashboardSelectedProject.value?.work_pool_cents) || 0) - dashboardSpentCents.value));
const dashboardRingStyle = computed(() => ({
  background: `conic-gradient(var(--green) 0 ${dashboardProgress.value}%, #e8eef1 ${dashboardProgress.value}% 100%)`,
}));
const dashboardProjectLedger = computed(() => {
  const project = dashboardSelectedProject.value;
  if (!project) return [];
  return dashboardLedgerEntries.value.filter((entry) => dashboardLedgerEntryMatchesProject(entry, project, dashboardSelectedTasks.value));
});
const dashboardLedgerFundingCents = computed(() =>
  dashboardProjectLedger.value
    .filter((entry) => entry.type === 'payment_verified')
    .reduce((total, entry) => total + (Number(entry.amount_cents) || 0), 0),
);
const dashboardLedgerPayoutCents = computed(() =>
  dashboardProjectLedger.value
    .filter((entry) => entry.type === 'task_payment')
    .reduce((total, entry) => total + (Number(entry.amount_cents) || 0), 0),
);
const dashboardProjectView = computed(() => {
  const project = dashboardSelectedProject.value;
  if (!project) {
    return {
      id: '',
      title: dashboardLoading.value ? 'Loading your projects' : 'No projects yet',
      body: dashboardLoading.value ? 'Fetching funded work from MergeOS.' : 'Start and fund a project to see real tasks, escrow, and ledger activity here.',
      initials: 'MP',
      status: dashboardLoading.value ? 'Syncing' : 'Empty',
      budget: `0 ${tokenSymbol.value}`,
      budgetCaption: 'MRG budget',
      repo: 'No repo yet',
      created: '-',
      taskSummary: '0 / 0',
      progress: 0,
    };
  }
  return {
    id: project.id,
    title: project.title || 'Untitled project',
    body: trimMarketplaceText(project.brief, 'Funded MergeOS project with escrow-backed tasks.'),
    initials: initialsFor(project.title || project.company_name || project.client_name || 'MP'),
    status: toTitleLabel(project.status || 'funded'),
    budget: formatMRGFromCents(project.budget_cents),
    budgetCaption: 'MRG budget',
    repo: shortRepoLabel(project),
    created: formatDashboardDate(project.created_at),
    taskSummary: `${dashboardAcceptedTasks.value.length} / ${dashboardSelectedTasks.value.length}`,
    progress: dashboardProgress.value,
  };
});
const dashboardWorkSplit = computed(() => {
  const tasks = dashboardSelectedTasks.value;
  return [
    { label: 'Human', className: 'critical', value: tasks.filter((task) => task.required_worker_kind === 'human').length },
    { label: 'Hybrid', className: 'high', value: tasks.filter((task) => task.required_worker_kind === 'hybrid').length },
    { label: 'Agent', className: 'medium', value: tasks.filter((task) => task.required_worker_kind === 'agent').length },
  ];
});
const dashboardTaskRows = computed(() => dashboardSelectedTasks.value.map(mapDashboardTask));
const dashboardActivityRows = computed(() =>
  dashboardProjectLedger.value.slice().reverse().slice(0, 6).map(mapDashboardActivity),
);
const dashboardLedgerRows = computed(() =>
  dashboardProjectLedger.value.slice().reverse().slice(0, 5).map((entry) => {
    const meta = ledgerMetaFor(entry.type);
    return {
      key: `${entry.sequence}-${entry.entry_hash || entry.reference}`,
      title: meta.type,
      value: formatMRGFromCents(entry.amount_cents),
      ref: shortLedgerReference(entry.reference || entry.entry_hash || `#${entry.sequence}`),
    };
  }),
);
const dashboardNotificationCount = computed(() => Math.min(9, dashboardActivityRows.value.length));

const marketplaceBenefits = [
  {
    icon: LockKeyhole,
    title: 'Secure Payments',
    body: 'Your payments are protected with escrow until the work is completed.',
  },
  {
    icon: Sparkles,
    title: 'AI Matching',
    body: 'Our AI matches you with the best talent and solutions for your project.',
  },
  {
    icon: Globe2,
    title: 'Global Talent',
    body: 'Access top developers and AI agents from around the world.',
  },
];

const sidebarSections = [
  {
    label: 'Main',
    items: [
      { label: 'Overview', icon: LayoutDashboard, toast: 'Opening overview...' },
      { label: 'My Projects', icon: FolderKanban, active: true, toast: 'Opening projects...' },
      { label: 'Tasks', icon: ListTodo, toast: 'Opening tasks...' },
      { label: 'Repositories', icon: GitBranch, toast: 'Opening repositories...' },
      { label: 'Payments', icon: CreditCard, toast: 'Opening payments...' },
      { label: 'Notifications', icon: Bell, action: () => { toggleNotificationPanel() } },
    ],
  },
  {
    label: 'Discover',
    items: [
      { label: 'Talent Marketplace', icon: UsersRound, page: 'marketplace' },
      { label: 'Bounty Explorer', icon: Compass, toast: 'Opening bounty explorer...' },
      { label: 'AI Agents', icon: Bot, toast: 'Opening AI agents...' },
    ],
  },
  {
    label: 'Tools',
    items: [
      { label: 'Repo Import', icon: UploadCloud, toast: 'Opening repo import...' },
      { label: 'AI Issue Scanner', icon: Search, toast: 'Opening AI issue scanner...' },
      { label: 'Estimate Cost', icon: Calculator, toast: 'Opening cost estimator...' },
    ],
  },
];

const topNavItems = [
  { label: 'Dashboard', active: true, toast: 'Opening dashboard...' },
  { label: 'Projects', toast: 'Opening projects...' },
  { label: 'Marketplace', page: 'marketplace' },
  { label: 'Repos', toast: 'Opening repositories...' },
  { label: 'Payments', toast: 'Opening payments...' },
  { label: 'Analytics', toast: 'Opening analytics...' },
];

const dashboardTabs = ['Overview', 'Tasks', 'Activity', 'Ledger', 'Files', 'Settings'];

function initialsFor(value = '') {
  const parts = value
    .replace(/@.*/, '')
    .split(/[\s._-]+/)
    .filter(Boolean);
  const letters = parts.length > 1
    ? `${parts[0][0]}${parts[1][0]}`
    : (parts[0] || 'MR').slice(0, 2);
  return letters.toUpperCase();
}

function shortWallet(value = '') {
  const address = String(value || '').trim();
  if (address.length <= 14) return address || 'MRG wallet';
  return `${address.slice(0, 6)}...${address.slice(-6)}`;
}

function randomOAuthState() {
  if (hasWindow && window.crypto?.getRandomValues) {
    const bytes = new Uint8Array(16);
    window.crypto.getRandomValues(bytes);
    return Array.from(bytes, (byte) => byte.toString(16).padStart(2, '0')).join('');
  }
  return `${Date.now()}-${Math.random().toString(16).slice(2)}`;
}

function openWalletOnScan(address = '') {
  const wallet = String(address || '').trim();
  if (!wallet || !hasWindow) return;
  window.open(`https://scan.mergeos.shop/address/${encodeURIComponent(wallet)}`, '_blank', 'noopener,noreferrer');
}

async function startGitHubLogin() {
  if (!hasWindow) return;
  errorMessage.value = '';
  const cfg = await loadRuntimeConfig();
  if (!cfg.github_oauth_ready || !cfg.github_oauth_client_id) {
    errorMessage.value = 'GitHub App login is not configured yet.';
    showToast(errorMessage.value);
    return;
  }

  const state = randomOAuthState();
  const redirectURI = `${window.location.origin}${window.location.pathname}`;
  window.sessionStorage.setItem('mergeos_github_oauth_state', state);
  window.sessionStorage.setItem('mergeos_github_oauth_redirect', redirectURI);
  const params = new URLSearchParams({
    client_id: cfg.github_oauth_client_id,
    redirect_uri: redirectURI,
    state,
  });
  window.location.href = `https://github.com/login/oauth/authorize?${params.toString()}`;
}

async function handleGitHubCallback() {
  if (!hasWindow) return false;
  const params = new URLSearchParams(window.location.search);
  const code = params.get('code');
  const state = params.get('state');
  if (!code) return false;

  const expectedState = window.sessionStorage.getItem('mergeos_github_oauth_state') || '';
  const redirectURI = window.sessionStorage.getItem('mergeos_github_oauth_redirect') || `${window.location.origin}${window.location.pathname}`;
  window.sessionStorage.removeItem('mergeos_github_oauth_state');
  window.sessionStorage.removeItem('mergeos_github_oauth_redirect');
  window.history.replaceState({ publicPage: publicPage.value }, '', window.location.pathname || '/');

  if (!expectedState || state !== expectedState) {
    errorMessage.value = 'GitHub sign-in state did not match. Please try again.';
    showToast(errorMessage.value);
    return true;
  }

  authBusy.value = true;
  try {
    const auth = await publicApi('/api/auth/github', {
      method: 'POST',
      body: JSON.stringify({ code, redirect_uri: redirectURI }),
    });
    setSession(auth);
    showToast(auth.user?.wallet_address ? 'GitHub linked to your MRG wallet.' : 'Logged in with GitHub.');
  } catch (error) {
    errorMessage.value = error.message;
    showToast(error.message);
  } finally {
    authBusy.value = false;
  }
  return true;
}

function showToast(message) {
  toastMessage.value = message;
  if (!hasWindow) return;
  if (toastTimer) window.clearTimeout(toastTimer);
  toastTimer = window.setTimeout(() => {
    toastMessage.value = '';
  }, 2200);
}

function scrollToSection(id) {
  if (!hasWindow) return;
  const section = document.getElementById(id);
  if (section) {
    section.scrollIntoView({ behavior: 'smooth' });
  }
}

function loadPublicPageData(page) {
  if (page === 'ledger') {
    void loadLedgerData();
    return;
  }
  if (page === 'marketplace' || page === 'home') {
    void loadMarketplaceData({ silent: true });
    void loadLedgerData({ silent: true });
  }
}

function updatePublicBrowserPath(page, replace = false) {
  if (!hasWindow) return;
  const targetPath = publicPathForPage(page);
  const currentPath = normalizeRoutePath(window.location.pathname);
  if (currentPath === targetPath && !window.location.search && !window.location.hash) {
    return;
  }
  const method = replace ? 'replaceState' : 'pushState';
  window.history[method]({ publicPage: page }, '', targetPath);
}

function updateProjectWizardBrowserPath(replace = false) {
  if (!hasWindow) return;
  const targetPath = projectWizardPathForState(projectWizardStage.value, projectWizardStep.value);
  const currentPath = normalizeRoutePath(window.location.pathname);
  if (currentPath === targetPath && !window.location.search && !window.location.hash) {
    return;
  }
  const method = replace ? 'replaceState' : 'pushState';
  window.history[method](
    { projectWizard: true, stage: projectWizardStage.value, step: projectWizardStep.value },
    '',
    targetPath,
  );
}

function openPublicPage(page, options = {}) {
  publicModeVisible.value = true;
  projectWizardVisible.value = false;
  const nextPage = normalizePublicPage(page);
  publicPage.value = nextPage;
  loadPublicPageData(nextPage);
  updatePublicBrowserPath(nextPage, Boolean(options.replace));
  if (!hasWindow) return;
  if (options.scroll === false) return;
  window.requestAnimationFrame(() => {
    window.scrollTo({ top: 0, behavior: 'smooth' });
  });
}

function syncPublicPageFromBrowserPath() {
  if (!hasWindow) return;
  publicModeVisible.value = true;
  const wizardRoute = projectWizardRouteFromPath(window.location.pathname);
  if (wizardRoute) {
    projectWizardVisible.value = true;
    projectWizardStage.value = wizardRoute.stage;
    projectWizardStep.value = wizardRoute.step;
    return;
  }
  projectWizardVisible.value = false;
  const nextPage = publicPageFromPath(window.location.pathname);
  publicPage.value = nextPage;
  loadPublicPageData(nextPage);
}

function handlePublicAction(action = {}) {
  if (action.command === 'project') {
    openProjectWizard();
    return;
  }
  if (action.page) {
    openPublicPage(action.page);
    return;
  }
  showToast(action.label ? `${action.label} opened.` : 'Opening page...');
}

function openDashboard() {
  publicModeVisible.value = false;
  if (user.value) {
    void loadDashboardData({ silent: true });
  }
  if (!hasWindow) return;
  window.requestAnimationFrame(() => {
    window.scrollTo({ top: 0, behavior: 'smooth' });
  });
}

function handleDashboardNav(item) {
  if (item.page) {
    openPublicPage(item.page);
    return;
  }
  if (item.label === 'Dashboard') {
    openDashboard();
    return;
  }
  showToast(item.toast || `${item.label} opened.`);
}

function openMarketplaceSection(id) {
  publicModeVisible.value = true;
  publicPage.value = 'marketplace';
  updatePublicBrowserPath('marketplace');
  loadPublicPageData('marketplace');
  if (!hasWindow) return;
  window.requestAnimationFrame(() => scrollToSection(id));
}

function scrollProjectFlowTop() {
  if (!hasWindow) return;
  window.scrollTo({ top: 0, behavior: 'smooth' });
}

function openProjectWizard(options = {}) {
  publicModeVisible.value = true;
  projectWizardVisible.value = true;
  projectWizardStage.value = 'setup';
  projectWizardStep.value = 1;
  errorMessage.value = '';
  updateProjectWizardBrowserPath(Boolean(options.replace));
  scrollProjectFlowTop();
}

function restartProjectWizard(options = {}) {
  publicModeVisible.value = true;
  projectWizardVisible.value = true;
  projectWizardStage.value = 'setup';
  projectWizardStep.value = 1;
  updateProjectWizardBrowserPath(Boolean(options.replace));
  scrollProjectFlowTop();
}

function closeProjectWizard(options = {}) {
  projectWizardVisible.value = false;
  projectWizardStage.value = 'setup';
  projectWizardStep.value = 1;
  if (options.updatePath !== false) {
    updatePublicBrowserPath(publicPage.value, Boolean(options.replace));
  }
}

async function loadRepoIssues() {
  const repoURL = projectSetupForm.repoUrl.trim();
  repoImportError.value = '';
  if (!repoURL) {
    repoImportError.value = 'Enter a GitHub repo URL first.';
    return;
  }

  repoImportBusy.value = true;
  try {
    const result = await publicApi('/api/public/repo/issues', {
      method: 'POST',
      body: JSON.stringify({ repo_url: repoURL }),
    });
    repoImportResult.value = result;
    projectSetupForm.repoUrl = result.repo_url || repoURL;
    if (repoImportedIssues.value.length) {
      projectSetupForm.projectType = 'Repo Issue Fix';
      projectSetupForm.title = `Fix all issues in ${result.owner}/${result.name}`;
      projectSetupForm.shortDescription = `Fix ${repoImportedIssues.value.length} open GitHub issues in ${result.owner}/${result.name}.`;
      projectSetupForm.overview = repoImportedIssues.value
        .map((issue) => `#${issue.number} ${issue.title} - score ${issue.score}, ${issue.complexity}`)
        .join('\n');
      projectDeliverables.value = repoImportedIssues.value.map((issue) => `Fix #${issue.number}: ${issue.title}`);
      showToast(`${repoImportedIssues.value.length} issues loaded and scored.`);
    } else {
      showToast('No open issues found.');
    }
  } catch (error) {
    repoImportResult.value = null;
    repoImportError.value = error.message || 'Could not load repo issues.';
    showToast(repoImportError.value);
  } finally {
    repoImportBusy.value = false;
  }
}

function openAuthFromProjectWizard(mode = 'login') {
  closeProjectWizard();
  openAuth(mode);
}

function goProjectStep(stepNumber) {
  projectWizardStage.value = 'setup';
  projectWizardStep.value = normalizeProjectWizardStep(stepNumber);
  updateProjectWizardBrowserPath();
  scrollProjectFlowTop();
}

function nextProjectStep() {
  if (projectWizardStep.value < 4) {
    projectWizardStep.value += 1;
    updateProjectWizardBrowserPath();
    scrollProjectFlowTop();
    return;
  }

  if (projectBudgetAmount.value > 0 && projectBudgetAmount.value < TOKEN_RATE_PER_USD * 100) {
    projectSetupForm.budgetAmount = TOKEN_RATE_PER_USD * 100;
  }
  projectFundingAmount.value = Math.max(100, Math.ceil(projectBudgetAmount.value / TOKEN_RATE_PER_USD) || projectFundingAmount.value);
  projectWizardStage.value = 'funding';
  updateProjectWizardBrowserPath();
  scrollProjectFlowTop();
  showToast('Project published. Add funds to start receiving proposals.');
}

function buildPriceEvaluationPayload() {
  return {
    title: projectSetupForm.title,
    description: [projectSetupForm.shortDescription, projectSetupForm.overview].filter(Boolean).join('\n\n'),
    project_type: projectSetupForm.projectType,
    requirements: projectSetupForm.requirements,
    deliverables: visibleDeliverables.value,
    timeline: projectTimelineLabel.value,
    tech_stack: projectSetupForm.techStack,
    complexity: projectSetupForm.allowAgents ? 'moderate' : 'high',
    constraints: projectSetupForm.skills,
    reference_budget_cents: centsFromMRG(projectSetupForm.budgetAmount),
  };
}

async function runProjectPriceEvaluation() {
  priceEvaluationError.value = '';
  if (!user.value) {
    authReturnToProjectWizard.value = true;
    projectWizardVisible.value = false;
    openAuth('login');
    showToast('Log in to estimate this project.');
    return;
  }
  if (priceEvaluationBusy.value) return;
  priceEvaluationBusy.value = true;
  try {
    priceEvaluation.value = await api('/api/projects/evaluate-price', {
      method: 'POST',
      body: JSON.stringify(buildPriceEvaluationPayload()),
    });
    applyPriceEvaluation();
    showToast('Budget estimate generated.');
  } catch (error) {
    priceEvaluationError.value = error.message;
    showToast(error.message);
  } finally {
    priceEvaluationBusy.value = false;
  }
}

function applyPriceEvaluation() {
  if (!priceEvaluation.value?.suggested_price_cents) return;
  projectSetupForm.budgetAmount = Math.max(TOKEN_RATE_PER_USD * 100, tokenAmountFromCents(priceEvaluation.value.suggested_price_cents));
  projectSetupForm.budgetType = 'Range';
}

function projectWizardBack() {
  if (projectWizardStage.value === 'success') {
    projectWizardStage.value = 'funding';
    updateProjectWizardBrowserPath();
    scrollProjectFlowTop();
    return;
  }

  if (projectWizardStage.value === 'funding') {
    projectWizardStage.value = 'setup';
    projectWizardStep.value = 4;
    updateProjectWizardBrowserPath();
    scrollProjectFlowTop();
    return;
  }

  if (projectWizardStep.value > 1) {
    projectWizardStep.value -= 1;
    updateProjectWizardBrowserPath();
    scrollProjectFlowTop();
    return;
  }

  closeProjectWizard();
}

function requireLoginForProjectPayment() {
  pendingProjectPaymentAfterAuth.value = true;
  authReturnToProjectWizard.value = true;
  projectPaymentError.value = '';
  projectWizardVisible.value = false;
  openAuth('login');
  showToast('Log in to continue payment.');
}

async function completeProjectFunding() {
  projectFundingAmount.value = Math.max(100, Number(projectFundingAmount.value) || 100);
  projectPaymentError.value = '';

  if (!user.value) {
    requireLoginForProjectPayment();
    return;
  }

  if (projectPaymentBusy.value) return;
  projectPaymentBusy.value = true;
  try {
    await loadRuntimeConfig();
    if (!paymentReferenceForProject()) {
      throw new Error('Payment reference is missing. Configure PayPal/Crypto checkout or enable local dev payments.');
    }
    const project = await api('/api/projects', {
      method: 'POST',
      body: JSON.stringify(buildCreateProjectPayload()),
    });
    fundedProject.value = project;
    projectWizardVisible.value = true;
    projectWizardStage.value = 'success';
    projectWizardStep.value = 4;
    updateProjectWizardBrowserPath();
    pendingProjectPaymentAfterAuth.value = false;
    authReturnToProjectWizard.value = false;
    await loadLedgerData({ silent: true });
    await loadMarketplaceData({ silent: true });
    await loadDashboardData({ silent: true, selectProjectID: project.id });
    scrollProjectFlowTop();
    showToast('Payment recorded and tokens minted.');
  } catch (error) {
    if (error.status === 401 || /login is required/i.test(error.message || '')) {
      requireLoginForProjectPayment();
      return;
    }
    projectPaymentError.value = error.message;
    showToast(error.message);
  } finally {
    projectPaymentBusy.value = false;
  }
}

function addDeliverable() {
  projectDeliverables.value.push('');
}

function removeDeliverable(index) {
  if (projectDeliverables.value.length <= 1) {
    projectDeliverables.value = [''];
    return;
  }

  projectDeliverables.value.splice(index, 1);
}

function loginWithSocial(provider) {
  showToast(`Redirecting to ${provider === 'google' ? 'Google' : 'GitHub'}...`);
  window.location.href = `/api/auth/${provider}/login`;
}

async function triggerAiEvaluation() {
  aiEvaluationLoading.value = true;
  aiEvaluationError.value = '';
  aiEvaluationResult.value = null;
  try {
    const payload = {
      description: projectSetupForm.overview || projectSetupForm.shortDescription || '',
      requirements: projectSetupForm.requirements
        ? projectSetupForm.requirements.split('\n').map(r => r.trim()).filter(Boolean)
        : [],
      deliverables: visibleDeliverables.value,
      timeline: projectTimelineLabel.value,
      tech_stack: projectSetupForm.techStack || '',
      complexity: projectSetupForm.complexity || 'Medium',
      constraints: projectSetupForm.constraints || '',
      reference_budget: Math.round(usdFromMRG(projectSetupForm.budgetAmount))
    };

    const response = await api('/api/projects/evaluate', {
      method: 'POST',
      body: JSON.stringify(payload)
    });

    aiEvaluationResult.value = response;
  } catch (err) {
    console.error('AI evaluation failed:', err);
    aiEvaluationError.value = err.message || 'AI evaluation failed. Please try again.';
  } finally {
    aiEvaluationLoading.value = false;
  }
}

function applyAiSuggestedPrice() {
  if (aiEvaluationResult.value) {
    const avg = Math.round((aiEvaluationResult.value.suggested_low + aiEvaluationResult.value.suggested_high) / 2);
    projectSetupForm.budgetAmount = mrgFromUSD(avg);
    showToast(`Applied AI suggested budget: ${formatMRGFromUSD(avg)}`);
  }
}

function formatMoney(value) {
  return new Intl.NumberFormat('en-US', {
    style: 'currency',
    currency: 'USD',
    maximumFractionDigits: 0,
  }).format(Number(value) || 0);
}

function mrgFromUSD(value = 0) {
  return Math.round((Number(value) || 0) * TOKEN_RATE_PER_USD);
}

function usdFromMRG(value = 0) {
  return (Number(value) || 0) / TOKEN_RATE_PER_USD;
}

function centsFromMRG(value = 0) {
  return Math.round(usdFromMRG(value) * 100);
}

function formatMRG(value = 0) {
  return `${formatCompactNumber(value)} ${tokenSymbol.value}`;
}

function formatMRGFromUSD(value = 0) {
  return formatMRG(mrgFromUSD(value));
}

function formatMRGFromCents(cents = 0) {
  return formatMRG(tokenAmountFromCents(cents));
}

function formatLedgerMRGFromCents(cents = 0) {
  return formatMRGFromCents(cents);
}

function formatPublicMRGFromCents(cents = 0) {
  return formatMRGFromCents(cents);
}

function formatPublicTokenAmount(amount = 0) {
  return `${formatCompactNumber(amount)} ${tokenSymbol.value}`;
}

function formatCompactNumber(value = 0) {
  return new Intl.NumberFormat('en-US', {
    maximumFractionDigits: value >= 100 ? 0 : 1,
  }).format(Number(value) || 0);
}

function tokenAmountFromCents(cents = 0) {
  return Math.round(((Number(cents) || 0) / 100) * TOKEN_RATE_PER_USD);
}

function toTitleLabel(value = '') {
  return String(value)
    .trim()
    .split(/[\s._:-]+/)
    .filter(Boolean)
    .map((word) => {
      const lower = word.toLowerCase();
      if (['ai', 'api', 'qa', 'ui', 'ux', 'go'].includes(lower)) return lower.toUpperCase();
      if (lower === 'devops') return 'DevOps';
      return `${lower.charAt(0).toUpperCase()}${lower.slice(1)}`;
    })
    .join(' ');
}

function paymentModeLabel(value = '') {
  const normalized = String(value || '').trim().toLowerCase();
  return {
    'live-adapters': 'Live payment adapters',
    'local-dev-verifier': 'MergeOS verifier',
    'not-configured': 'Not configured',
    '': 'Not loaded',
  }[normalized] || toTitleLabel(value);
}

function repoProviderLabel(value = '') {
  const normalized = String(value || '').trim().toLowerCase();
  if (!normalized) return 'Not loaded';
  if (normalized.startsWith('github-private:')) return 'GitHub private repos';
  if (normalized === 'local-git') return 'MergeOS repositories';
  return toTitleLabel(value);
}

function formatMarketplaceDate(value) {
  const date = value ? new Date(value) : null;
  if (!date || Number.isNaN(date.getTime())) return 'Funded';
  return `Funded ${date.toLocaleDateString('en-US', { month: 'short', day: 'numeric' })}`;
}

function trimMarketplaceText(value = '', fallback = 'Funded MergeOS project with escrow-backed tasks and ledger proof.') {
  const text = String(value || '').replace(/\s+/g, ' ').trim();
  if (!text) return fallback;
  return text.length > 150 ? `${text.slice(0, 147).trim()}...` : text;
}

function marketplaceProjectIcon(project = {}) {
  const text = [
    project.title,
    project.brief,
    project.site_type,
    ...(project.tags || []),
  ].join(' ').toLowerCase();
  if (text.includes('mobile')) return Phone;
  if (text.includes('ai') || text.includes('agent')) return Bot;
  if (text.includes('analytics') || text.includes('dashboard')) return BarChart3;
  if (text.includes('payment') || text.includes('checkout') || text.includes('ledger')) return CreditCard;
  if (text.includes('api') || text.includes('code')) return Code2;
  return Globe2;
}

function marketplaceAgentIcon(type = '') {
  const text = String(type).toLowerCase();
  if (text.includes('design')) return PenLine;
  if (text.includes('ledger') || text.includes('go')) return CreditCard;
  if (text.includes('devops')) return GitBranch;
  if (text.includes('qa') || text.includes('test')) return CheckCircle2;
  if (text.includes('front')) return Code2;
  return Bot;
}

function mapMarketplaceProject(project = {}, index = 0) {
  const palette = marketplaceProjectPalettes[index % marketplaceProjectPalettes.length];
  const rawTags = (project.tags || []).map(toTitleLabel).filter(Boolean);
  const tags = rawTags.length ? rawTags : [toTitleLabel(project.site_type || 'Project')];
  const openTasks = Number(project.open_task_count) || 0;
  const acceptedTasks = Number(project.accepted_task_count) || 0;
  const taskCount = Number(project.task_count) || openTasks + acceptedTasks;
  const client = project.client_display_name || 'MergeOS client';
  const badge = openTasks > 0 ? `${openTasks} OPEN` : (acceptedTasks > 0 ? 'PAID OUT' : toTitleLabel(project.status || 'LIVE'));
  return {
    id: project.id || `project-${index}`,
    icon: marketplaceProjectIcon(project),
    badge,
    badgeTone: openTasks > 0 ? palette.badgeTone : 'green',
    title: project.title || 'Untitled project',
    body: trimMarketplaceText(project.brief),
    tags: tags.slice(0, 3),
    extra: Math.max(0, tags.length - 3),
    budget: formatPublicMRGFromCents(project.budget_cents),
    timeline: project.timeline || formatMarketplaceDate(project.created_at),
    client,
    clientInitials: initialsFor(client),
    avatarTone: palette.avatarTone,
    taskLabel: `${taskCount} tasks`,
    verified: project.status === 'funded',
    urgent: openTasks > 0,
    accent: palette.accent,
    soft: palette.soft,
  };
}

function marketplaceSearchHaystack(project = {}) {
  return [
    project.title,
    project.body,
    project.client,
    project.budget,
    project.timeline,
    ...(project.tags || []),
  ].join(' ').toLowerCase();
}

function formatDashboardDate(value) {
  const date = value ? new Date(value) : null;
  if (!date || Number.isNaN(date.getTime())) return '-';
  return date.toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' });
}

function formatDateInputLabel(value = '') {
  if (!value) return '';
  const date = new Date(`${value}T00:00:00Z`);
  if (Number.isNaN(date.getTime())) return '';
  return date.toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric', timeZone: 'UTC' });
}

function shortRepoLabel(project = {}) {
  const repo = String(project.bounty_repo_name || project.repo_url || project.repo_provider || '').trim();
  if (!repo) return 'No repo yet';
  return repo.replace(/^mergeos-bounties\//, '');
}

function dashboardLedgerEntryMatchesProject(entry = {}, project = {}, tasks = []) {
  const haystack = [
    entry.reference,
    entry.from_account,
    entry.to_account,
  ].filter(Boolean).join('|');
  if (project.id && haystack.includes(project.id)) return true;
  if (project.bounty_repo_name && haystack.includes(project.bounty_repo_name)) return true;
  return tasks.some((task) => task.id && haystack.includes(task.id));
}

function taskIssueReference(task = {}) {
  if (task.issue_url) {
    const parts = String(task.issue_url).split(/[\\/]/).filter(Boolean);
    return parts.slice(-2).join('/');
  }
  return task.suggested_agent_type ? toTitleLabel(task.suggested_agent_type) : toTitleLabel(task.required_worker_kind || 'task');
}

function mapDashboardTask(task = {}) {
  const status = task.status === 'accepted' ? 'Accepted' : 'Open';
  return {
    id: task.id || `${task.project_id}-${task.issue_number}`,
    initials: String(task.issue_number || 'T').padStart(2, '0').slice(-2),
    issueNumber: task.issue_number || '-',
    title: task.title || 'Untitled task',
    acceptance: trimMarketplaceText(task.acceptance, 'Acceptance criteria not provided.'),
    reference: taskIssueReference(task),
    reward: formatMRGFromCents(task.reward_cents),
    kind: toTitleLabel(task.required_worker_kind || task.worker_kind || 'task'),
    agent: task.suggested_agent_type ? toTitleLabel(task.suggested_agent_type) : '-',
    status,
    statusClass: task.status === 'accepted' ? 'accepted' : 'open',
  };
}

function mapDashboardActivity(entry = {}) {
  const meta = ledgerMetaFor(entry.type);
  return {
    key: `${entry.sequence}-${entry.entry_hash || entry.reference}`,
    title: meta.type,
    icon: meta.icon,
    color: meta.tone === 'amber' ? 'yellow' : meta.tone,
    time: formatLedgerDateTime(entry.created_at).full,
  };
}

function formatLedgerDateTime(value) {
  const date = value ? new Date(value) : new Date();
  if (Number.isNaN(date.getTime())) {
    return { date: '-', time: '-', full: '-' };
  }
  return {
    date: date.toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric', timeZone: 'UTC' }),
    time: date.toLocaleTimeString('en-US', { hour: '2-digit', minute: '2-digit', timeZone: 'UTC' }),
    full: `${date.toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric', timeZone: 'UTC' })} ${date.toLocaleTimeString('en-US', { hour: '2-digit', minute: '2-digit', timeZone: 'UTC' })} UTC`,
  };
}

function projectInitialFor(value = '') {
  return (String(value).trim().charAt(0) || 'M').toUpperCase();
}

function projectToneFor(value = '') {
  const tones = ['green', 'blue', 'purple'];
  const total = String(value).split('').reduce((sum, char) => sum + char.charCodeAt(0), 0);
  return tones[total % tones.length];
}

function extractProjectID(entry = {}) {
  const haystack = [
    entry.reference,
    entry.from_account,
    entry.to_account,
  ].filter(Boolean).join(' ');
  return haystack.match(/prj_[a-z0-9]+/i)?.[0] || '';
}

function ledgerMetaFor(type = '') {
  const normalized = String(type);
  if (normalized === 'token_mint') {
    return { type: 'Token Minted', icon: Box, tone: 'green', amountTone: 'positive' };
  }
  if (normalized === 'payment_verified') {
    return { type: 'Payment Verified', icon: ShieldCheck, tone: 'green', amountTone: 'positive' };
  }
  if (normalized === 'platform_fee') {
    return { type: 'Platform Fee', icon: CircleDollarSign, tone: 'amber', amountTone: 'negative' };
  }
  if (normalized === 'project_reserve') {
    return { type: 'Escrow Reserved', icon: LockKeyhole, tone: 'blue', amountTone: 'neutral' };
  }
  if (normalized === 'task_reserve') {
    return { type: 'Task Reserve', icon: FileCheck2, tone: 'blue', amountTone: 'neutral' };
  }
  if (normalized === 'task_payment') {
    return { type: 'Payout Released', icon: CircleDollarSign, tone: 'green', amountTone: 'negative' };
  }
  return { type: normalized.replaceAll('_', ' '), icon: Compass, tone: 'slate', amountTone: 'neutral' };
}

function mapLedgerEntry(entry) {
  const projectID = extractProjectID(entry);
  const project = ledgerProjectIndex.value.get(projectID);
  const meta = ledgerMetaFor(entry.type);
  const when = formatLedgerDateTime(entry.created_at);
  const tokenAmount = tokenAmountFromCents(entry.amount_cents);
  const projectTitle = project?.title || (projectID ? `Project ${projectID.slice(-6)}` : 'MergeOS ledger');
  const company = project?.company_name || project?.client_name || 'MergeOS';
  return {
    key: `${entry.sequence}-${entry.entry_hash || entry.reference}`,
    date: when.date,
    time: when.time,
    type: meta.type,
    icon: meta.icon,
    tone: meta.tone,
    projectInitial: projectInitialFor(projectTitle),
    projectTone: projectToneFor(projectID || projectTitle),
    project: projectTitle,
    company,
    amount: `${formatCompactNumber(Math.abs(tokenAmount))} ${tokenSymbol.value}`,
    secondaryAmount: entry.type === 'payment_verified' ? 'funding verified' : entry.type === 'token_mint' ? 'mint log' : '',
    amountTone: meta.amountTone,
    ref: shortLedgerReference(entry.reference || entry.entry_hash || `#${entry.sequence}`),
  };
}

function shortLedgerReference(value = '') {
  const text = String(value);
  if (text.length <= 18) return text;
  return `${text.slice(0, 8)}...${text.slice(-6)}`;
}

function paymentMethodForProject() {
  return projectPaymentMethod.value === 'USDC' ? 'crypto' : 'paypal';
}

function paymentReferenceForProject() {
  if (runtimeConfig.value?.dev_payment_enabled && runtimeConfig.value?.dev_payment_code) {
    return runtimeConfig.value.dev_payment_code;
  }
  return successPaymentReference.value || '';
}

function buildCreateProjectPayload() {
  const name = user.value?.name || authForm.name || 'MergeOS Client';
  const email = user.value?.email || authForm.email;
  return {
    title: projectSetupForm.title,
    client_name: name,
    company_name: user.value?.company_name || authForm.company_name || 'MergeOS Customer',
    client_email: email,
    site_type: projectSetupForm.projectType,
    package_tier: projectSetupForm.budgetType,
    timeline: projectTimelineLabel.value,
    brief: buildProjectBrief(),
    budget_cents: projectPaymentAmountCents.value,
    payment_method: paymentMethodForProject(),
    payment_reference: paymentReferenceForProject(),
    attachment_ids: [],
    source_repo_url: projectSetupForm.repoUrl || '',
  };
}

function buildProjectBrief() {
  return [
    projectSetupForm.repoUrl && `Source repository: ${projectSetupForm.repoUrl}`,
    projectSetupForm.shortDescription,
    projectSetupForm.overview && `Overview:\n${projectSetupForm.overview}`,
    repoImportedIssues.value.length && `Imported issues:\n${repoImportedIssues.value.map((issue) => `- #${issue.number} ${issue.title} (score ${issue.score}, ${issue.complexity})`).join('\n')}`,
    visibleDeliverables.value.length && `Deliverables:\n${visibleDeliverables.value.map((item) => `- ${item}`).join('\n')}`,
    projectSetupForm.requirements && `Requirements:\n${projectSetupForm.requirements}`,
    projectSetupForm.techStack && `Tech stack: ${projectSetupForm.techStack}`,
    `Visibility: ${projectSetupForm.visibility}`,
    `AI agents: ${projectSetupForm.allowAgents ? 'Allowed' : 'Not allowed'}`,
    projectSetupForm.skills && `Skills: ${projectSetupForm.skills}`,
  ].filter(Boolean).join('\n\n');
}

function resetAuthForm(mode = authMode.value) {
  Object.assign(authForm, mode === 'login' ? defaultLoginAuth : defaultRegisterAuth);
  authTermsAccepted.value = mode === 'register';
  authRememberMe.value = false;
  showPassword.value = false;
  showConfirmPassword.value = false;
}

function setAuthMode(mode) {
  authMode.value = mode;
  resetAuthForm(mode);
  errorMessage.value = '';
}

function createRequestError(response, payload = {}) {
  const error = new Error(payload.error || 'Request failed');
  error.status = response.status;
  return error;
}

async function api(path, options = {}) {
  const response = await fetch(path, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      ...(token.value ? { Authorization: `Bearer ${token.value}` } : {}),
      ...(options.headers || {}),
    },
  });
  const payload = await response.json();
  if (!response.ok) {
    if (response.status === 401 && path !== '/api/auth/login' && path !== '/api/auth/register') {
      clearSession();
    }
    throw createRequestError(response, payload);
  }
  return payload;
}

async function publicApi(path, options = {}) {
  const response = await fetch(path, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      ...(options.headers || {}),
    },
  });
  const text = await response.text();
  let payload = {};
  try {
    payload = text ? JSON.parse(text) : {};
  } catch {
    payload = { error: text || 'Request failed' };
  }
  if (!response.ok) {
    throw createRequestError(response, payload);
  }
  return payload;
}

async function loadMarketplaceData(options = {}) {
  const silent = Boolean(options.silent);
  if (!silent) marketplaceLoading.value = true;
  marketplaceError.value = '';
  try {
    const payload = await publicApi('/api/public/marketplace');
    marketplaceData.value = {
      stats: payload.stats || {},
      projects: Array.isArray(payload.projects) ? payload.projects : [],
      contributors: Array.isArray(payload.contributors) ? payload.contributors : [],
      agents: Array.isArray(payload.agents) ? payload.agents : [],
    };
    if (!marketplaceCategories.value.includes(activeMarketplaceCategory.value)) {
      activeMarketplaceCategory.value = 'All';
    }
  } catch (error) {
    marketplaceError.value = error.message || 'Could not load marketplace data';
  } finally {
    marketplaceLoading.value = false;
  }
}

async function loadRuntimeConfig() {
  if (runtimeConfig.value) {
    return runtimeConfig.value;
  }
  runtimeConfig.value = await api('/api/config');
  return runtimeConfig.value;
}

async function loadLedgerData(options = {}) {
  ledgerError.value = '';
  if (!options.silent) {
    ledgerLoading.value = true;
  }
  try {
    const [entries, marketplace] = await Promise.all([
      publicApi('/api/public/ledger'),
      publicApi('/api/public/marketplace'),
    ]);
    ledgerRawEntries.value = Array.isArray(entries) ? entries : [];
    ledgerProjects.value = Array.isArray(marketplace.projects) ? marketplace.projects : [];
  } catch (error) {
    ledgerError.value = error.message;
  } finally {
    ledgerLoading.value = false;
  }
}

async function loadDashboardData(options = {}) {
  if (!token.value) {
    dashboardProjects.value = [];
    dashboardTasks.value = [];
    dashboardLedgerEntries.value = [];
    selectedDashboardProjectID.value = '';
    return;
  }

  const silent = Boolean(options.silent);
  if (!silent) dashboardLoading.value = true;
  dashboardError.value = '';
  try {
    const [projects, tasks, entries] = await Promise.all([
      api('/api/projects'),
      api('/api/tasks'),
      api('/api/ledger'),
    ]);
    dashboardProjects.value = Array.isArray(projects) ? projects : [];
    dashboardTasks.value = Array.isArray(tasks) ? tasks : [];
    dashboardLedgerEntries.value = Array.isArray(entries) ? entries : [];
    const requestedProjectID = options.selectProjectID || selectedDashboardProjectID.value;
    const selectedExists = dashboardProjects.value.some((project) => project.id === requestedProjectID);
    selectedDashboardProjectID.value = selectedExists
      ? requestedProjectID
      : (dashboardSortedProjects.value[0]?.id || '');
  } catch (error) {
    dashboardError.value = error.message || 'Could not load projects';
  } finally {
    dashboardLoading.value = false;
  }
}

function startDashboardRealtime() {
  if (!hasWindow || dashboardRefreshTimer) return;
  dashboardRefreshTimer = window.setInterval(() => {
    if (!token.value || !user.value) return;
    if (document.visibilityState === 'hidden') return;
    void loadDashboardData({ silent: true });
  }, DASHBOARD_REFRESH_MS);
}

function stopDashboardRealtime() {
  if (!hasWindow || !dashboardRefreshTimer) return;
  window.clearInterval(dashboardRefreshTimer);
  dashboardRefreshTimer = 0;
}

function openAuth(mode = 'login') {
  setAuthMode(mode);
  authVisible.value = true;
}

function closeAuth() {
  if (authBusy.value) return;
  authVisible.value = false;
  errorMessage.value = '';
  if (authReturnToProjectWizard.value) {
    projectWizardVisible.value = true;
    authReturnToProjectWizard.value = false;
    pendingProjectPaymentAfterAuth.value = false;
  }
}

function setSession(auth) {
  token.value = auth.token;
  user.value = auth.user;
  authVisible.value = false;
  errorMessage.value = '';
  writeStoredToken(auth.token);
  if (authReturnToProjectWizard.value) {
    projectWizardVisible.value = true;
    authReturnToProjectWizard.value = false;
  }
  if (pendingProjectPaymentAfterAuth.value) {
    pendingProjectPaymentAfterAuth.value = false;
    void completeProjectFunding();
  }
  if (publicPage.value === 'ledger') {
    void loadLedgerData({ silent: true });
  }
  void loadDashboardData({ silent: true });
  startDashboardRealtime();
}

function clearSession() {
  stopDashboardRealtime();
  token.value = '';
  user.value = null;
  authVisible.value = false;
  ledgerError.value = '';
  dashboardProjects.value = [];
  dashboardTasks.value = [];
  dashboardLedgerEntries.value = [];
  dashboardError.value = '';
  selectedDashboardProjectID.value = '';
  removeStoredToken();
}

async function submitAuth() {
  errorMessage.value = '';
  if (authMode.value === 'register') {
    if (!authTermsAccepted.value) {
      errorMessage.value = 'Please accept the terms before creating an account.';
      return;
    }
    if (authForm.password !== authForm.confirm_password) {
      errorMessage.value = 'Passwords do not match.';
      return;
    }
  }

  authBusy.value = true;
  try {
    const path = authMode.value === 'register' ? '/api/auth/register' : '/api/auth/login';
    const body = authMode.value === 'register'
      ? {
        name: authForm.name,
        company_name: authForm.company_name,
        email: authForm.email,
        password: authForm.password,
      }
      : { email: authForm.email, password: authForm.password };
    const auth = await api(path, { method: 'POST', body: JSON.stringify(body) });
    setSession(auth);
    showToast(authMode.value === 'register' ? 'Account created.' : 'Logged in.');
  } catch (error) {
    errorMessage.value = error.message;
  } finally {
    authBusy.value = false;
  }
}

async function restoreSession() {
  if (!token.value) return;
  try {
    user.value = await api('/api/auth/me');
    await loadDashboardData({ silent: true });
    startDashboardRealtime();
    if (publicPage.value === 'ledger') {
      void loadLedgerData({ silent: true });
    }
  } catch {
    clearSession();
  }
}

async function logout() {
  try {
    await api('/api/auth/logout', { method: 'POST', body: JSON.stringify({}) });
  } finally {
    clearSession();
    showToast('Logged out.');
  }
}

onMounted(async () => {
  if (hasWindow) {
    const params = new URLSearchParams(window.location.search);
    const oauthToken = params.get('token');
    if (oauthToken) {
      token.value = oauthToken;
      writeStoredToken(oauthToken);
      const cleanUrl = window.location.pathname + window.location.hash;
      window.history.replaceState({}, document.title, cleanUrl);
      showToast('Successfully logged in via OAuth!');
    }
  }

  const handledGitHubCallback = await handleGitHubCallback();
  if (hasWindow) {
    window.addEventListener('popstate', syncPublicPageFromBrowserPath);
    if (!handledGitHubCallback) {
      if (projectWizardVisible.value) {
        updateProjectWizardBrowserPath(true);
      } else {
        updatePublicBrowserPath(publicPage.value, true);
      }
    }
  }
  const runtimePromise = loadRuntimeConfig().catch((error) => showToast(error.message));
  await Promise.all([
    runtimePromise,
    restoreSession(),
    loadMarketplaceData({ silent: true }),
    loadLedgerData({ silent: true }),
  ]);
});

onUnmounted(() => {
  if (hasWindow) {
    window.removeEventListener('popstate', syncPublicPageFromBrowserPath);
  }
  stopDashboardRealtime();
});
// --- Notification System ---
const notificationPanelOpen = ref(false);
const notifications = ref([]);
const notificationsLoading = ref(false);
const unreadNotificationCount = computed(() => notifications.value.filter(n => !n.read).length);

function toggleNotificationPanel() {
  notificationPanelOpen.value = !notificationPanelOpen.value;
  if (notificationPanelOpen.value && notifications.value.length === 0) {
    fetchNotifications();
  }
}

async function fetchNotifications() {
  notificationsLoading.value = true;
  try {
    const token = localStorage.getItem('mergeos_token');
    const headers = { 'Content-Type': 'application/json' };
    if (token) headers['Authorization'] = token;

    const resp = await fetch('/api/notifications', { headers });
    if (resp.ok) {
      const data = await resp.json();
      notifications.value = (Array.isArray(data) ? data : (data.notifications || [])).slice(0, 50);
    } else {
      // Use fallback sample data for testing
      notifications.value = [
        { id: 1, subject: 'Welcome to MergeOS', body: 'Your account has been created successfully.', channel: 'system', read: false, created_at: new Date().toISOString() },
        { id: 2, subject: 'Project created', body: 'Your project has been created and is now visible in the marketplace.', channel: 'project', read: false, created_at: new Date(Date.now() - 3600000).toISOString() },
        { id: 3, subject: 'Payment received', body: 'A payment of 500 MRG has been processed for your project.', channel: 'payment', read: true, created_at: new Date(Date.now() - 86400000).toISOString() },
      ];
    }
  } catch (err) {
    console.error('Failed to fetch notifications:', err);
    // Fallback
    notifications.value = [
      { id: 1, subject: 'Welcome to MergeOS', body: 'Your account has been created successfully.', channel: 'system', read: false, created_at: new Date().toISOString() },
    ];
  } finally {
    notificationsLoading.value = false;
  }
}

function markNotificationRead(notif) {
  notif.read = true;
}

async function markAllNotificationsRead() {
  notifications.value.forEach(n => n.read = true);
  // Sync with backend if user is logged in
  try {
    const token = localStorage.getItem('mergeos_token');
    if (token) {
      await fetch('/api/notifications/read-all', {
        method: 'POST',
        headers: { 'Authorization': token, 'Content-Type': 'application/json' }
      });
    }
  } catch (err) {
    console.error('Failed to mark all read:', err);
  }
}

function formatNotificationTime(dateStr) {
  if (!dateStr) return '';
  const date = new Date(dateStr);
  const now = new Date();
  const diffMs = now - date;
  const diffMin = Math.floor(diffMs / 60000);
  if (diffMin < 1) return 'Just now';
  if (diffMin < 60) return `${diffMin}m ago`;
  const diffHr = Math.floor(diffMin / 60);
  if (diffHr < 24) return `${diffHr}h ago`;
  const diffDay = Math.floor(diffHr / 24);
  if (diffDay < 7) return `${diffDay}d ago`;
  return date.toLocaleDateString();
}

// Close notification panel when clicking outside
function handleClickOutside(e) {
  const panel = document.querySelector('.notification-panel');
  const btn = document.querySelector('[aria-label="Notifications"]');
  if (notificationPanelOpen.value && panel && !panel.contains(e.target) && btn && !btn.contains(e.target)) {
    notificationPanelOpen.value = false;
  }
}
if (typeof document !== 'undefined') {
  document.addEventListener('click', handleClickOutside);
}

</script>
