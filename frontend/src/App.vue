<template>
  <div v-if="projectWizardVisible" class="project-flow-shell">
    <div v-if="toastMessage" class="toast project-flow-toast" role="status" aria-live="polite">
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
        <a href="/live" @click.prevent="closeProjectWizard(); openPublicPage('live')">Live Feed</a>
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
          <button type="button" @click="openScopeAssistant">
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
              <input v-model.trim="projectSetupForm.title" :placeholder="projectSetupForm.projectType === 'Bug Fix' ? 'Briefly describe the bug you need fixed' : 'Enter a clear project title'" />
            </label>

            <label class="wizard-field full">
              <span>Short description <b>*</b></span>
              <textarea
                v-model.trim="projectSetupForm.shortDescription"
                rows="5"
                maxlength="1000"
                :placeholder="projectSetupForm.projectType === 'Bug Fix' ? 'Describe the bug, expected behavior, and how to reproduce it' : 'Describe your project, what you want to build, and the problem you are solving...'"
              />
              <small>{{ projectSetupForm.shortDescription.length }} / 1000</small>
            </label>

            <section class="wizard-section full">
              <div class="wizard-section-title">
                <strong>Project type <b>*</b></strong>
                <small>What kind of work do you need?</small>
              </div>
              <div class="project-type-grid project-type-split">
                <button
                  v-for="type in projectTypeOptions"
                  :key="type.label"
                  :class="{ selected: projectSetupForm.projectType === type.label }"
                  class="select-tile select-tile-big"
                  type="button"
                  @click="projectSetupForm.projectType = type.label"
                >
                  <component :is="type.icon" :size="24" />
                  <strong>{{ type.label }}</strong>
                  <small>{{ type.caption }}</small>
                  <CheckCircle2 v-if="projectSetupForm.projectType === type.label" class="tile-check" :size="20" />
                </button>
              </div>
            </section>

            <label class="wizard-field full">
              <span>Tech stack <small>(optional)</small></span>
              <input v-model.trim="projectSetupForm.techStack"
                :placeholder="projectSetupForm.projectType === 'Bug Fix' ? 'Technologies used in the existing project' : 'Add technologies or frameworks'" />
            </label>

            <section v-if="projectSetupForm.projectType === 'Bug Fix'" class="wizard-section full attach-repo">
              <div class="attach-repo-head">
                <div>
                  <strong>Attach repository <b>*</b></strong>
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
                  ref="repoImportInput"
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

            <section v-if="projectSetupForm.projectType === 'New Project'" class="wizard-section full">
              <div class="new-project-hint">
                <p>Describe your idea below — we will help you scope it, price it, and find the right contributor.</p>
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

            <section
              class="wizard-section split reference-dropzone"
              @dragover.prevent
              @drop.prevent="uploadProjectAttachments"
            >
              <UploadCloud :size="24" />
              <strong>Drag & drop files here</strong>
              <button class="text-action" :disabled="attachmentUploadBusy" type="button" @click="openAttachmentPicker">
                {{ attachmentUploadBusy ? 'Uploading...' : 'browse your files' }}
              </button>
              <input
                ref="attachmentInput"
                class="hidden-file-input"
                multiple
                type="file"
                @change="uploadProjectAttachments"
              />
              <small>Supports images, PDFs, docs, links up to 200MB.</small>
              <p v-if="attachmentUploadError" class="attachment-error">{{ attachmentUploadError }}</p>
              <div v-if="projectAttachments.length" class="attachment-list" aria-label="Project attachments">
                <article v-for="file in projectAttachments" :key="file.id" class="attachment-row">
                  <FileText :size="15" />
                  <div>
                    <strong>{{ file.original_name || 'Attachment' }}</strong>
                    <small>{{ formatFileSize(file.size_bytes) }} · {{ file.content_type || 'file' }}</small>
                  </div>
                  <button type="button" :aria-label="`Remove ${file.original_name || 'attachment'}`" @click="removeProjectAttachment(file.id)">
                    <X :size="14" />
                  </button>
                </article>
              </div>
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

            <section v-if="projectAttachments.length" class="review-card">
              <button type="button" aria-label="Edit attachments" @click="goProjectStep(2)">
                <PenLine :size="15" />
              </button>
              <h3>
                <FileText :size="17" />
                Reference files
              </h3>
              <ul>
                <li v-for="file in projectAttachments" :key="file.id">
                  <CheckCircle2 :size="14" />
                  {{ file.original_name || 'Attachment' }}
                </li>
              </ul>
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
              <button class="secondary-button compact ghost" type="button" @click="saveProjectDraft">Save draft</button>
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
            <button class="primary-button compact" type="button" @click="openFundedProjectDashboard('Overview')">
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
          <button type="button" @click="generateScopeSuggestions">Generate with AI</button>
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
          <button v-for="item in postPaymentActions" :key="item.label" type="button" @click="handlePostPaymentAction(item)">
            <CheckCircle2 :size="15" />
            {{ item.label }}
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
    <div v-if="toastMessage" class="toast dashboard-toast" role="status" aria-live="polite">
      {{ toastMessage }}
    </div>

    <aside class="dash-sidebar" aria-label="Customer navigation">
      <button class="dash-brand" type="button" @click="openDashboardSection('projects')">
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
            :class="{ active: isDashboardNavActive(item) }"
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
        <button type="button" @click="openDashboardSection('worker')">
          Learn more
          <ArrowRight :size="14" />
        </button>
      </article>
    </aside>

    <section class="dash-workspace">
      <header class="dash-topbar">
        <label class="dash-search">
          <Search :size="16" />
          <input v-model.trim="dashboardSearch" :placeholder="dashboardSearchPlaceholder" />
          <kbd>Ctrl K</kbd>
        </label>

        <nav class="dash-topnav" aria-label="Dashboard sections">
          <button
            v-for="item in topNavItems"
            :key="item.label"
            :class="{ active: isDashboardNavActive(item) }"
            type="button"
            @click="handleDashboardNav(item)"
          >
            {{ item.label }}
          </button>
        </nav>

        <div class="dash-top-actions">
          <button class="dash-icon-button" aria-label="Notifications" type="button" @click="openDashboardSection('notifications')">
            <Bell :size="18" />
            <span v-if="dashboardNotificationCount > 0" class="notification-badge">{{ dashboardNotificationCount }}</span>
          </button>
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

      <section class="dash-command-strip" aria-label="Dashboard command summary">
        <div class="dash-command-copy">
          <span class="marketplace-eyebrow">{{ dashboardSectionEyebrow }}</span>
          <h1>{{ dashboardCommandTitle }}</h1>
          <p>{{ dashboardCommandBody }}</p>
        </div>
        <div class="dash-command-metrics">
          <article v-for="metric in dashboardCommandStats" :key="metric.label">
            <span :class="['public-card-icon', metric.tone]">
              <component :is="metric.icon" :size="18" />
            </span>
            <div>
              <strong>{{ metric.value }}</strong>
              <small>{{ metric.label }}</small>
            </div>
          </article>
        </div>
      </section>

      <main class="dash-content">
        <section class="dash-main">
          <template v-if="dashboardSection === 'admin'">
            <div class="dash-breadcrumb">
              <Home :size="14" />
              <span>Admin</span>
              <ChevronDown :size="13" />
              <strong>Treasury & Ops</strong>
            </div>

            <section v-if="adminConsoleError" class="dash-empty-state">
              <strong>Could not load admin console</strong>
              <p>{{ adminConsoleError }}</p>
              <button class="secondary-button compact" type="button" @click="loadAdminConsoleData">Retry</button>
            </section>

            <template v-else>
              <section class="dash-project-header admin-console-header">
                <div class="dash-project-title">
                  <span class="project-photo payment-photo">
                    <ShieldCheck :size="18" />
                  </span>
                  <div>
                    <h1>Admin Console</h1>
                    <p>Treasury, users, payout review, disputes, moderation, and reputation signals.</p>
                  </div>
                  <span class="live-badge">{{ adminSummaryView.status }}</span>
                </div>

                <div class="dash-project-actions">
                  <button type="button" @click="loadAdminConsoleData">
                    <RefreshCw :size="15" />
                    Refresh
                  </button>
                  <button type="button" @click="openPublicPage('ledger')">
                    <Link2 :size="15" />
                    Public ledger
                  </button>
                </div>
              </section>

              <section class="dash-overview-grid admin-summary-grid">
                <article class="dash-card payment-summary-card">
                  <span>Total Budget</span>
                  <strong>{{ adminSummaryView.totalBudget }}</strong>
                  <small>{{ adminSummaryView.projects }} projects / {{ adminSummaryView.openTasks }} open tasks</small>
                </article>
                <article class="dash-card payment-summary-card">
                  <span>Work Pool</span>
                  <strong>{{ adminSummaryView.workPool }}</strong>
                  <small>{{ adminSummaryView.acceptedTasks }} accepted tasks</small>
                </article>
                <article class="dash-card payment-summary-card">
                  <span>Paid Tasks</span>
                  <strong>{{ adminSummaryView.paidTasks }}</strong>
                  <small>{{ adminSummaryView.platformFee }} platform fees</small>
                </article>
                <article class="dash-card payment-summary-card">
                  <span>Users</span>
                  <strong>{{ adminSummaryView.users }}</strong>
                  <small>{{ adminSummaryView.admins }} admins / {{ adminSummaryView.paymentMode }}</small>
                </article>
              </section>

              <section class="admin-console-grid">
                <article class="dash-card admin-treasury-card">
                  <div class="card-title-row">
                    <div>
                      <h2>Treasury Readiness</h2>
                      <p>Payment and integration status from backend runtime config.</p>
                    </div>
                    <span>{{ adminSummaryView.repoProvider }}</span>
                  </div>
                  <div class="admin-readiness-list">
                    <span :class="{ ready: adminSummaryView.payPalReady }">PayPal <strong>{{ adminSummaryView.payPalReady ? 'Ready' : 'Missing' }}</strong></span>
                    <span :class="{ ready: adminSummaryView.cryptoReady }">Crypto <strong>{{ adminSummaryView.cryptoReady ? 'Ready' : 'Missing' }}</strong></span>
                    <span :class="{ ready: adminSummaryView.githubReady }">GitHub <strong>{{ adminSummaryView.githubReady ? 'Ready' : 'Missing' }}</strong></span>
                    <span :class="{ ready: adminSummaryView.smtpReady }">SMTP <strong>{{ adminSummaryView.smtpReady ? 'Ready' : 'Missing' }}</strong></span>
                  </div>
                </article>

                <article class="dash-card admin-ssl-card">
                  <div class="card-title-row">
                    <div>
                      <h2>SSL & Deployment Security</h2>
                      <p>Certificate checks for public, admin, and scan domains before deploy handoff.</p>
                    </div>
                    <span>{{ adminSSLStats.label }}</span>
                  </div>
                  <div class="admin-ssl-actions">
                    <span>{{ adminSSLStats.ready }} ready / {{ adminSSLStats.total }} domains</span>
                    <button type="button" :disabled="adminSSLReviewBusy" @click="runAdminSSLReview">
                      {{ adminSSLReviewBusy ? 'Reviewing...' : 'Run Review' }}
                    </button>
                  </div>
                  <p v-if="adminSSLReviewError" class="deployment-error">{{ adminSSLReviewError }}</p>
                  <div v-if="adminSSLRows.length" class="admin-ssl-list">
                    <article v-for="row in adminSSLRows" :key="row.id">
                      <span :class="['admin-ops-icon', row.tone]">
                        <Lock :size="15" />
                      </span>
                      <div>
                        <strong>{{ row.domain }}:{{ row.port }}</strong>
                        <small>{{ row.issuer }} / expires {{ row.expiry }} / {{ row.daysRemaining }}</small>
                        <b>{{ row.dnsSummary }}</b>
                        <small v-if="row.error" class="admin-pr-blockers">{{ row.error }}</small>
                      </div>
                      <div class="admin-ssl-side">
                        <em :class="row.tone">{{ row.status }}</em>
                        <small>{{ row.checkedBy }}</small>
                      </div>
                    </article>
                  </div>
                  <article v-else class="dash-empty-state compact">
                    <strong>{{ adminConsoleLoading ? 'Loading SSL reviews...' : 'No SSL domains configured' }}</strong>
                    <p>{{ adminConsoleLoading ? 'Fetching certificate review state.' : 'Configured SSL review domains will appear here.' }}</p>
                  </article>
                </article>

                <article class="dash-card admin-llm-card">
                  <div class="card-title-row">
                    <div>
                      <h2>AI Review Providers</h2>
                      <p>Runtime provider, API key pool, and webhook health for automated PR review.</p>
                    </div>
                    <span>{{ adminLLMStats.label }}</span>
                  </div>

                  <form class="admin-llm-form" @submit.prevent="submitAdminLLMSettings">
                    <label>
                      <span>Provider</span>
                      <select v-model="adminLLMForm.provider" @change="handleAdminLLMProviderChange">
                        <option v-for="option in adminLLMProviderOptions" :key="option.id" :value="option.id">{{ option.label }}</option>
                      </select>
                    </label>
                    <label>
                      <span>Model</span>
                      <select v-model="adminLLMForm.model">
                        <option v-for="model in adminLLMModelOptions" :key="model" :value="model">{{ model }}</option>
                      </select>
                    </label>
                    <button type="submit" :disabled="adminLLMBusy">Save Runtime</button>
                  </form>

                  <form class="admin-llm-key-form" @submit.prevent="submitAdminLLMKey">
                    <label>
                      <span>New API key</span>
                      <input v-model.trim="adminLLMForm.apiKey" type="password" autocomplete="new-password" placeholder="Provider API key" />
                    </label>
                    <button type="submit" :disabled="adminLLMBusy || !adminLLMKeyReady">Add Key</button>
                  </form>
                  <p v-if="adminLLMError" class="deployment-error">{{ adminLLMError }}</p>

                  <div v-if="adminLLMKeyRows.length" class="admin-llm-key-list">
                    <article v-for="key in adminLLMKeyRows" :key="key.id">
                      <span :class="['admin-ops-icon', key.tone]">
                        <Bot :size="15" />
                      </span>
                      <div>
                        <strong>{{ key.provider }} / {{ key.model }}</strong>
                        <small>{{ key.keyHint }} / {{ key.requestCount }} requests / {{ key.successCount }} ok / {{ key.quotaErrorCount }} quota</small>
                        <b>{{ key.lastStatusCode ? `HTTP ${key.lastStatusCode}` : 'No status code' }} / {{ key.lastUsed }}</b>
                        <small v-if="key.lastError" class="admin-pr-blockers">{{ key.lastError }}</small>
                      </div>
                      <div class="admin-llm-side">
                        <em :class="key.tone">{{ key.status }}</em>
                        <button type="button" :disabled="adminLLMKeyBusyID === key.testBusyID" @click="testAdminLLMKey(key)">
                          {{ adminLLMKeyBusyID === key.testBusyID ? 'Testing...' : 'Test' }}
                        </button>
                        <button
                          type="button"
                          :disabled="adminLLMKeyBusyID === key.resetBusyID"
                          @click="updateAdminLLMKey(key, '', true)"
                        >
                          Reset
                        </button>
                        <button
                          type="button"
                          :disabled="adminLLMKeyBusyID === key.toggleBusyID"
                          @click="updateAdminLLMKey(key, key.toggleStatus)"
                        >
                          {{ key.toggleLabel }}
                        </button>
                      </div>
                    </article>
                  </div>
                  <article v-else class="dash-empty-state compact">
                    <strong>{{ adminConsoleLoading ? 'Loading AI keys...' : 'No AI review keys' }}</strong>
                    <p>{{ adminConsoleLoading ? 'Fetching provider key pool.' : 'Add a provider key to enable automated PR review and LLM cost evaluation.' }}</p>
                  </article>

                  <div v-if="adminLLMWebhookRows.length" class="admin-llm-webhook-list">
                    <article v-for="log in adminLLMWebhookRows" :key="log.id">
                      <span :class="['admin-ops-icon', log.tone]">
                        <GitPullRequest :size="15" />
                      </span>
                      <div>
                        <strong>{{ log.title }}</strong>
                        <small>{{ log.body }} / {{ log.when }}</small>
                        <b>{{ log.repository }} / {{ log.duration }}</b>
                        <small v-if="log.error" class="admin-pr-blockers">{{ log.error }}</small>
                      </div>
                      <div class="admin-llm-side">
                        <em :class="log.tone">{{ log.status }}</em>
                        <button v-if="log.commentURL" type="button" @click="openExternalURL(log.commentURL)">Comment</button>
                      </div>
                    </article>
                  </div>
                </article>

                <article class="dash-card admin-credit-card">
                  <div class="card-title-row">
                    <div>
                      <h2>Manual MRG Credit</h2>
                      <p>Create payout credit with PR URL or admin reference.</p>
                    </div>
                    <span>{{ adminCreditForm.rewardMRG }} MRG</span>
                  </div>
                  <form class="admin-credit-form" @submit.prevent="submitAdminManualCredit">
                    <label>
                      <span>Worker ID</span>
                      <input v-model.trim="adminCreditForm.workerID" placeholder="github:contributor or wallet" />
                    </label>
                    <div class="admin-credit-row">
                      <label>
                        <span>Reward MRG</span>
                        <input v-model.number="adminCreditForm.rewardMRG" min="1" type="number" />
                      </label>
                      <label>
                        <span>Bounty type</span>
                        <select v-model="adminCreditForm.bountyType">
                          <option value="future-small">Future small</option>
                          <option value="future-medium">Future medium</option>
                          <option value="bug-large">Bug large</option>
                          <option value="major-feature">Major feature</option>
                        </select>
                      </label>
                    </div>
                    <label>
                      <span>PR URL</span>
                      <input v-model.trim="adminCreditForm.prURL" placeholder="https://github.com/owner/repo/pull/123" />
                    </label>
                    <label>
                      <span>PR title</span>
                      <input v-model.trim="adminCreditForm.prTitle" placeholder="Pull request title" />
                    </label>
                    <div class="admin-credit-row">
                      <label>
                        <span>Task ID</span>
                        <input v-model.trim="adminCreditForm.taskID" placeholder="tsk_0001" />
                      </label>
                      <label>
                        <span>Reference</span>
                        <input v-model.trim="adminCreditForm.reference" placeholder="manual admin reference" />
                      </label>
                    </div>
                    <p v-if="adminCreditError" class="deployment-error">{{ adminCreditError }}</p>
                    <div v-if="adminCreditResultView" class="admin-credit-result">
                      <strong>{{ adminCreditResultView.reward }} credited</strong>
                      <small>{{ adminCreditResultView.workerID }} / {{ adminCreditResultView.reference }}</small>
                      <button v-if="adminCreditResultView.creditURL" type="button" @click="openExternalURL(adminCreditResultView.creditURL)">Open Scan</button>
                    </div>
                    <button type="submit" :disabled="adminCreditBusy || !adminCreditReady">
                      {{ adminCreditBusy ? 'Crediting...' : 'Credit Worker' }}
                    </button>
                  </form>
                </article>

                <article class="dash-card admin-test-settings-card">
                  <div class="card-title-row">
                    <div>
                      <h2>Test Publish Settings</h2>
                      <p>Enable password-gated public test keys for LLM, PayPal, and USDT task work.</p>
                    </div>
                    <span>{{ adminTestSettingsStatus }}</span>
                  </div>
                  <form class="admin-test-settings-form" @submit.prevent="submitAdminTestSettings">
                    <label class="admin-toggle-row">
                      <input v-model="adminTestSettings.test_mode_enabled" type="checkbox" />
                      <span>
                        Public test mode
                        <strong>{{ adminTestSettings.test_mode_enabled ? 'Enabled' : 'Disabled' }}</strong>
                      </span>
                    </label>
                    <label>
                      <span>Shared password</span>
                      <input v-model="adminTestSettingsPassword" type="password" autocomplete="new-password" placeholder="Set or rotate public password" />
                    </label>
                    <p v-if="adminTestSettingsError" class="deployment-error">{{ adminTestSettingsError }}</p>
                    <div class="admin-test-settings-actions">
                      <button type="submit" :disabled="adminTestSettingsBusy">
                        {{ adminTestSettingsBusy ? 'Saving...' : 'Save Settings' }}
                      </button>
                      <button type="button" @click="openPublicPage('test-settings')">Open Public Page</button>
                    </div>
                  </form>
                  <div v-if="adminTestSettingsRows.length" class="admin-test-key-list">
                    <article v-for="entry in adminTestSettingsRows" :key="entry.id">
                      <span class="admin-ops-icon green">
                        <LockKeyhole :size="15" />
                      </span>
                      <div>
                        <strong>{{ entry.displayName }}</strong>
                        <small>{{ entry.integrationType }} / {{ entry.settingKey }} / {{ entry.valueHint }}</small>
                        <b v-if="entry.mapKeys.length">{{ entry.mapKeys.join(', ') }}</b>
                      </div>
                      <em>{{ entry.status }}</em>
                    </article>
                  </div>
                  <article v-else class="dash-empty-state compact">
                    <strong>{{ adminConsoleLoading ? 'Loading test keys...' : 'No published test keys' }}</strong>
                    <p>{{ adminConsoleLoading ? 'Fetching test publish settings.' : 'Use the public test page to add database-backed keys after password access is enabled.' }}</p>
                  </article>
                </article>

                <article class="dash-card admin-pr-review-card">
                  <div class="card-title-row">
                    <div>
                      <h2>PR Review & Merge</h2>
                      <p>Check evidence, repository star, conflict, scope, and security blockers before payout.</p>
                    </div>
                    <span>{{ adminTaskReviewRows.length }} tasks</span>
                  </div>

                  <form class="admin-merge-form" @submit.prevent>
                    <label>
                      <span>Reward MRG</span>
                      <input v-model.number="adminMergeForm.rewardMRG" min="1" type="number" />
                    </label>
                    <label>
                      <span>Bounty type</span>
                      <select v-model="adminMergeForm.bountyType">
                        <option value="future-small">Future small</option>
                        <option value="future-medium">Future medium</option>
                        <option value="bug-large">Bug large</option>
                        <option value="major-feature">Major feature</option>
                      </select>
                    </label>
                  </form>

                  <p v-if="adminTaskPullsError" class="deployment-error">{{ adminTaskPullsError }}</p>
                  <p v-if="adminMergeError" class="deployment-error">{{ adminMergeError }}</p>
                  <div v-if="adminMergeResultView" class="admin-credit-result admin-merge-result">
                    <strong>{{ adminMergeResultView.reward }} credited</strong>
                    <small>{{ adminMergeResultView.workerID }} / {{ adminMergeResultView.title }} / {{ adminMergeResultView.bountyType }}</small>
                    <div class="admin-merge-result-actions">
                      <button v-if="adminMergeResultView.creditURL" type="button" @click="openExternalURL(adminMergeResultView.creditURL)">Open Scan</button>
                      <button v-if="adminMergeResultView.commentURL" type="button" @click="openExternalURL(adminMergeResultView.commentURL)">Open Comment</button>
                    </div>
                    <small v-if="adminMergeResultView.commentError">Comment failed: {{ adminMergeResultView.commentError }}</small>
                  </div>

                  <div class="admin-pr-review-layout">
                    <div v-if="adminTaskReviewRows.length" class="admin-pr-task-list">
                      <article v-for="row in adminTaskReviewRows" :key="row.id">
                        <div>
                          <strong>#{{ row.issueNumber || '-' }} {{ row.title }}</strong>
                          <small>{{ row.status }} / {{ row.workerKind }} / {{ row.reward }} / {{ row.bountyType }}</small>
                          <b>{{ row.projectID }} / {{ row.id }}</b>
                        </div>
                        <div class="admin-pr-task-actions">
                          <button type="button" :disabled="adminTaskPullsLoadingID === row.id" @click="loadAdminTaskPulls(row.id)">
                            {{ adminTaskPullsLoadingID === row.id ? 'Loading...' : 'Load PRs' }}
                          </button>
                          <button v-if="row.issueURL" type="button" @click="openExternalURL(row.issueURL)">Issue</button>
                        </div>
                      </article>
                    </div>
                    <article v-else class="dash-empty-state compact">
                      <strong>{{ adminConsoleLoading ? 'Loading tasks...' : 'No GitHub-linked tasks' }}</strong>
                      <p>{{ adminConsoleLoading ? 'Fetching admin task rows.' : 'Tasks with GitHub issue links will appear here.' }}</p>
                    </article>

                    <div v-if="adminLoadedPullGroups.length" class="admin-pr-pull-list">
                      <section v-for="group in adminLoadedPullGroups" :key="group.taskID" class="admin-pr-group">
                        <header>
                          <strong>{{ group.title }}</strong>
                          <button v-if="group.issueURL" type="button" @click="openExternalURL(group.issueURL)">Issue #{{ group.issueNumber }}</button>
                        </header>
                        <article v-for="pull in group.pullRequests" :key="pull.key">
                          <span :class="['admin-ops-icon', pull.tone]">
                            <GitPullRequest :size="15" />
                          </span>
                          <div>
                            <strong>#{{ pull.number }} {{ pull.title }}</strong>
                            <small>@{{ pull.author }} / {{ pull.state }} / {{ pull.riskLevel }} risk / {{ pull.fileCount }} files</small>
                            <div class="admin-pr-signal-row">
                              <span :class="{ ready: pull.evidenceReady }">Evidence</span>
                              <span :class="{ ready: pull.starReady }">Star</span>
                              <span :class="pull.tone">{{ pull.status }}</span>
                            </div>
                            <b v-if="pull.labels.length">{{ pull.labels.join(', ') }}</b>
                            <small v-if="pull.blockers.length" class="admin-pr-blockers">Blockers: {{ pull.blockers.slice(0, 2).join('; ') }}</small>
                            <small v-else-if="pull.warnings.length" class="admin-pr-warnings">Warnings: {{ pull.warnings.slice(0, 2).join('; ') }}</small>
                            <small v-else-if="pull.signals.length">Signals: {{ pull.signals.join(', ') }}</small>
                          </div>
                          <div class="admin-pr-side">
                            <em :class="pull.tone">{{ pull.canMerge ? 'Ready' : 'Blocked' }}</em>
                            <button v-if="pull.htmlURL" type="button" @click="openExternalURL(pull.htmlURL)">Open</button>
                            <button
                              type="button"
                              :disabled="!pull.canMerge || !adminMergeReady || adminMergeBusyID === pull.key"
                              @click="mergeAdminTaskPull(group.taskID, pull)"
                            >
                              {{ adminMergeBusyID === pull.key ? 'Merging...' : pull.merged ? 'Credit' : 'Merge' }}
                            </button>
                          </div>
                        </article>
                      </section>
                    </div>
                    <article v-else class="dash-empty-state compact">
                      <strong>No loaded PRs</strong>
                      <p>Load a task to review linked pull requests and readiness evidence.</p>
                    </article>
                  </div>
                </article>

                <article class="dash-card admin-ops-card">
                  <div class="card-title-row">
                    <div>
                      <h2>Ops Queue</h2>
                      <p>Disputes, payout review, moderation, fraud, and security signals.</p>
                    </div>
                    <span>{{ formatCompactNumber(adminOpsQueue.stats?.total_count) }} rows</span>
                  </div>
                  <div class="admin-ops-stats">
                    <span v-for="stat in adminOpsStats" :key="stat.label" :class="stat.tone">
                      {{ stat.label }}
                      <strong>{{ stat.value }}</strong>
                    </span>
                  </div>
                  <div v-if="adminOpsRows.length" class="admin-ops-list">
                    <article v-for="item in adminOpsRows" :key="item.id">
                      <span :class="['admin-ops-icon', item.tone]">
                        <ShieldCheck :size="15" />
                      </span>
                      <div>
                        <strong>{{ item.title }}</strong>
                        <small>{{ item.body }}</small>
                        <b>{{ item.type }} / {{ item.reference || item.project || item.status }}</b>
                      </div>
                      <div class="admin-ops-side">
                        <em :class="item.tone">{{ item.severity }}</em>
                        <button v-if="item.url" type="button" @click="openExternalURL(item.url)">Open</button>
                      </div>
                    </article>
                  </div>
                  <article v-else class="dash-empty-state compact">
                    <strong>{{ adminConsoleLoading ? 'Loading ops queue...' : 'No ops items' }}</strong>
                    <p>{{ adminConsoleLoading ? 'Fetching admin queue.' : 'No dispute, payout, moderation, fraud, or security items require review.' }}</p>
                  </article>
                </article>

                <article class="dash-card admin-users-card">
                  <div class="card-title-row">
                    <div>
                      <h2>Users</h2>
                      <p>Admin users, clients, wallets, GitHub identities, project spend, and worker audit hints.</p>
                    </div>
                    <span>{{ adminUserRows.length }} shown</span>
                  </div>
                  <div v-if="adminUserRows.length" class="admin-users-list">
                    <article v-for="row in adminUserRows" :key="row.id">
                      <span :class="['contributor-avatar', row.tone]">{{ initialsFor(row.name) }}</span>
                      <div>
                        <strong>{{ row.name }}</strong>
                        <small>{{ row.email }} / {{ row.role }} / {{ row.company }}</small>
                        <b>{{ row.github }} / {{ row.wallet }} / {{ row.projects }} projects / {{ row.budget }}</b>
                      </div>
                      <div class="admin-user-side">
                        <em :class="row.tone">{{ row.risk }}</em>
                        <button type="button" :disabled="!row.workerID" @click="prefillAdminCreditFromUser(row)">Credit</button>
                      </div>
                    </article>
                  </div>
                  <article v-else class="dash-empty-state compact">
                    <strong>{{ adminConsoleLoading ? 'Loading users...' : 'No users' }}</strong>
                    <p>{{ adminConsoleLoading ? 'Fetching admin user rows.' : 'Registered users will appear here.' }}</p>
                  </article>
                </article>

                <article class="dash-card admin-reputation-card">
                  <div class="card-title-row">
                    <div>
                      <h2>Worker Reputation</h2>
                      <p>{{ formatCompactNumber(adminReputationStats.worker_count) }} workers / {{ formatCompactNumber(adminReputationStats.completed_task_count) }} completed tasks</p>
                    </div>
                    <span>{{ formatCompactNumber(adminReputationStats.high_risk_count) }} high risk</span>
                  </div>
                  <div v-if="adminReputationRows.length" class="admin-reputation-list">
                    <article v-for="worker in adminReputationRows" :key="worker.id">
                      <span :class="['contributor-avatar', worker.tone]">{{ initialsFor(worker.name) }}</span>
                      <div>
                        <strong>{{ worker.name }}</strong>
                        <small>{{ worker.level }} / {{ worker.completed }} tasks / {{ worker.rewards }}</small>
                        <b v-if="worker.flags.length">{{ worker.flags.slice(0, 2).join(', ') }}</b>
                      </div>
                      <em :class="worker.tone">{{ worker.score }}</em>
                    </article>
                  </div>
                  <article v-else class="dash-empty-state compact">
                    <strong>{{ adminConsoleLoading ? 'Loading reputation...' : 'No worker reputation rows' }}</strong>
                    <p>{{ adminConsoleLoading ? 'Fetching worker audit signals.' : 'Paid workers will appear after accepted tasks and ledger payouts.' }}</p>
                  </article>
                </article>
              </section>
            </template>
          </template>

          <template v-else-if="dashboardSection === 'payments'">
            <div class="dash-breadcrumb">
              <Home :size="14" />
              <span>Payments</span>
              <ChevronDown :size="13" />
              <strong>{{ dashboardPaymentView.title }}</strong>
            </div>

            <section v-if="dashboardError" class="dash-empty-state">
              <strong>Could not load your payment history</strong>
              <p>{{ dashboardError }}</p>
              <button class="secondary-button compact" type="button" @click="loadDashboardData">Retry</button>
            </section>

            <template v-else>
              <section class="dash-project-header payment-history-header">
                <div class="dash-project-title">
                  <span class="project-photo payment-photo">
                    <CreditCard :size="18" />
                  </span>
                  <div>
                    <h1>{{ dashboardPaymentView.title }}</h1>
                    <p>{{ dashboardPaymentView.body }}</p>
                  </div>
                  <span class="live-badge">{{ dashboardPaymentView.status }}</span>
                </div>

                <div class="dash-project-actions">
                  <button type="button" @click="loadDashboardData">
                    <RefreshCw :size="15" />
                    Refresh
                  </button>
                  <button type="button" @click="openPublicPage('ledger')">
                    <Link2 :size="15" />
                    View Ledger
                  </button>
                  <button type="button" aria-label="Copy payment history state" @click="showToast('Payment history is live and synced from your ledger.')">
                    <MoreHorizontal :size="16" />
                  </button>
                </div>
              </section>

              <section class="dash-overview-grid payment-summary-grid">
                <article v-for="item in dashboardPaymentSummary" :key="item.label" class="dash-card payment-summary-card">
                  <span>{{ item.label }}</span>
                  <strong>{{ item.value }}</strong>
                  <small>{{ item.caption }}</small>
                </article>
              </section>

              <section class="dash-card payment-history-card">
                <div class="card-title-row">
                  <div>
                    <h2>Ledger-backed payment activity</h2>
                    <p>Funding, escrow, fees, mint logs, and payouts for the selected project are grouped in one place.</p>
                  </div>
                  <span>{{ dashboardPaymentRows.length }} rows</span>
                </div>

                <div v-if="dashboardPaymentRows.length" class="payment-history-list">
                  <article v-for="row in dashboardPaymentRows" :key="row.key" class="payment-history-row">
                    <div class="payment-history-main">
                      <span :class="['ledger-event-type', row.tone]">{{ row.type }}</span>
                      <strong>{{ row.title }}</strong>
                      <p>{{ row.body }}</p>
                    </div>

                    <div class="payment-history-meta">
                      <span>
                        <small>Method</small>
                        <strong>{{ row.method }}</strong>
                      </span>
                      <span>
                        <small>Status</small>
                        <strong>{{ row.status }}</strong>
                      </span>
                      <span>
                        <small>Counterparty</small>
                        <strong>{{ row.counterparty }}</strong>
                      </span>
                    </div>

                    <div class="payment-history-side">
                      <strong :class="['payment-history-amount', row.amountClass]">{{ row.amount }}</strong>
                      <small>{{ row.when }}</small>
                      <button type="button" @click="showToast(`Reference: ${row.rawReference}`)">
                        {{ row.reference }}
                      </button>
                    </div>
                  </article>
                </div>
                <article v-else class="dash-empty-state compact">
                  <strong>{{ dashboardLoading ? 'Loading payment history...' : 'No payment history yet' }}</strong>
                  <p>{{ dashboardLoading ? 'Fetching real funding and payout logs.' : 'Fund a project or release a task payout to create the first payment row.' }}</p>
                </article>

                <div class="watching-line">
                  <CreditCard :size="14" />
                  {{ dashboardPaymentRows.length }} payment events loaded
                </div>
              </section>
            </template>
          </template>

          <template v-else-if="dashboardSection === 'worker'">
            <div class="dash-breadcrumb">
              <Home :size="14" />
              <span>Worker</span>
              <ChevronDown :size="13" />
              <strong>{{ workerDashboardView.title }}</strong>
            </div>

            <section v-if="workerDashboardError" class="dash-empty-state">
              <strong>Could not load worker dashboard</strong>
              <p>{{ workerDashboardError }}</p>
              <button class="secondary-button compact" type="button" @click="loadWorkerDashboardData">Retry</button>
            </section>

            <template v-else>
              <section class="dash-project-header worker-dashboard-header">
                <div class="dash-project-title">
                  <span class="project-photo">{{ workerDashboardView.initials }}</span>
                  <div>
                    <h1>{{ workerDashboardView.title }}</h1>
                    <p>{{ workerDashboardView.body }}</p>
                  </div>
                  <span class="live-badge">{{ workerDashboardView.status }}</span>
                </div>

                <div class="dash-project-actions">
                  <button type="button" @click="loadWorkerDashboardData">
                    <RefreshCw :size="15" />
                    Refresh
                  </button>
                  <button type="button" @click="openPublicPage('marketplace')">
                    <UsersRound :size="15" />
                    Marketplace
                  </button>
                </div>
              </section>

              <section class="dash-metrics" aria-label="Worker summary">
                <article v-for="metric in workerDashboardMetrics" :key="metric.label">
                  <span>{{ metric.label }}</span>
                  <strong>{{ metric.value }}</strong>
                  <small>{{ metric.caption }}</small>
                </article>
              </section>

              <section class="worker-dashboard-grid">
                <article class="dash-card worker-reputation-card">
                  <div class="card-title-row">
                    <h2>Reputation</h2>
                    <span>{{ workerReputationScore }} / 100</span>
                  </div>
                  <div class="worker-score-ring" :style="workerScoreRingStyle">
                    <strong>{{ workerReputationScore }}</strong>
                    <span>Score</span>
                  </div>
                  <div class="risk-grid">
                    <span v-for="item in workerReputationRows" :key="item.label" :class="item.tone">
                      {{ item.label }}
                      <strong>{{ item.value }}</strong>
                    </span>
                  </div>
                </article>

                <article class="dash-card worker-identity-card">
                  <div class="card-title-row">
                    <h2>Identity</h2>
                    <span>{{ workerIdentityReadyCount }} ready</span>
                  </div>
                  <div class="worker-identity-list">
                    <article v-for="item in workerIdentityRows" :key="item.label">
                      <span :class="['notification-dot', item.ready ? 'green' : 'amber']" />
                      <div>
                        <strong>{{ item.label }}</strong>
                        <small>{{ item.value || 'Not linked' }}</small>
                      </div>
                      <b>{{ item.ready ? 'Ready' : 'Needed' }}</b>
                    </article>
                  </div>
                  <button class="rail-link-button" :disabled="authBusy || !githubOAuthReady" type="button" @click="startGitHubLogin">
                    Link GitHub
                  </button>
                </article>
              </section>

              <section class="dash-card live-pr-monitor-card">
                <div class="card-title-row">
                  <div>
                    <h2>Live PR Monitor</h2>
                    <p>{{ dashboardPullRequestSummary.body }}</p>
                  </div>
                  <button type="button" :disabled="dashboardPullRequestsLoading || !dashboardSelectedProject" @click="loadDashboardPullRequestsData(dashboardSelectedProject?.id)">
                    <RefreshCw :size="14" />
                  </button>
                </div>
                <div class="live-pr-metrics" aria-label="Pull request monitor summary">
                  <article>
                    <span>Open</span>
                    <strong>{{ dashboardPullRequestSummary.open }}</strong>
                  </article>
                  <article>
                    <span>Ready</span>
                    <strong>{{ dashboardPullRequestSummary.ready }}</strong>
                  </article>
                  <article>
                    <span>Blocked</span>
                    <strong>{{ dashboardPullRequestSummary.blocked }}</strong>
                  </article>
                  <article>
                    <span>Status</span>
                    <strong>{{ dashboardPullRequestSummary.status }}</strong>
                  </article>
                </div>
                <div v-if="dashboardPullRequestsError" class="deployment-error">{{ dashboardPullRequestsError }}</div>
                <div v-else-if="dashboardPullRequestRows.length" class="live-pr-monitor-list">
                  <article v-for="pull in dashboardPullRequestRows" :key="pull.id">
                    <span :class="['metric-icon', pull.tone]">
                      <GitPullRequest :size="16" />
                    </span>
                    <div>
                      <strong>#{{ pull.number }} {{ pull.title }}</strong>
                      <small>{{ pull.task }} / {{ pull.author }} / {{ pull.risk }}</small>
                      <b>{{ pull.updatedAt }}</b>
                    </div>
                    <div class="live-pr-monitor-side">
                      <span :class="['ledger-event-type', pull.tone]">{{ pull.status }}</span>
                      <button v-if="pull.url" type="button" @click="openExternalURL(pull.url)">Open</button>
                    </div>
                  </article>
                </div>
                <article v-else class="dash-empty-state compact">
                  <strong>{{ dashboardPullRequestsLoading ? 'Loading pull requests...' : 'No linked PRs yet' }}</strong>
                  <p>{{ dashboardPullRequestsLoading ? 'Syncing GitHub linked PRs.' : 'Contributor PRs linked to project issues will appear here.' }}</p>
                </article>
              </section>

              <section class="dash-card live-pr-board">
                <div class="card-title-row">
                  <div>
                    <h2>Claimed Tasks</h2>
                    <p>Accepted work matched to your wallet or GitHub identity.</p>
                  </div>
                  <button type="button" @click="loadWorkerDashboardData">Refresh</button>
                </div>
                <div v-if="workerClaimedTaskRows.length" class="dash-pr-list">
                  <article v-for="task in workerClaimedTaskRows" :key="task.id" class="dash-pr-row">
                    <span class="contributor-avatar">{{ task.initials }}</span>
                    <div class="dash-pr-main">
                      <strong>#{{ task.issueNumber }} {{ task.title }}</strong>
                      <small>{{ task.acceptance }}</small>
                      <span>{{ task.project }}</span>
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
                      <strong>{{ task.when }}</strong>
                      <small>Accepted</small>
                    </div>
                    <b class="accepted">Paid</b>
                  </article>
                </div>
                <article v-else class="dash-empty-state compact">
                  <strong>{{ workerDashboardLoading ? 'Loading claimed tasks...' : 'No claimed tasks yet' }}</strong>
                  <p>{{ workerDashboardLoading ? 'Matching tasks to your GitHub and wallet identities.' : 'Accepted tasks paid to your identity will appear here.' }}</p>
                </article>
              </section>

              <section class="worker-dashboard-grid">
                <article class="dash-card">
                  <div class="card-title-row">
                    <h2>Rewards</h2>
                    <span>{{ workerRewardRows.length }} rows</span>
                  </div>
                  <div v-if="workerRewardRows.length" class="worker-reward-list">
                    <article v-for="reward in workerRewardRows" :key="reward.key">
                      <div>
                        <strong>{{ reward.amount }}</strong>
                        <small>{{ reward.type }} · {{ reward.when }}</small>
                      </div>
                      <span>{{ reward.ref }}</span>
                    </article>
                  </div>
                  <article v-else class="dash-empty-state compact">
                    <strong>No reward entries</strong>
                    <p>MRG task payouts and manual credits will be listed here.</p>
                  </article>
                </article>

                <article class="dash-card">
                  <div class="card-title-row">
                    <h2>Proposal Opportunities</h2>
                    <span>{{ workerProposalRows.length }}</span>
                  </div>
                  <div v-if="workerProposalRows.length" class="worker-proposal-list">
                    <article v-for="proposal in workerProposalRows" :key="proposal.id">
                      <div>
                        <strong>{{ proposal.title }}</strong>
                        <small>{{ proposal.project }} · {{ proposal.lane }}</small>
                      </div>
                      <span>{{ proposal.reward }}</span>
                      <b>{{ proposal.matchScore }}%</b>
                      <div class="worker-proposal-actions">
                        <button v-if="proposal.url" type="button" @click="openExternalURL(proposal.url)">Issue</button>
                        <button type="button" :disabled="!proposal.claimCommand" @click="copyClaimCommand(proposal.claimCommand)">Copy Claim</button>
                      </div>
                    </article>
                  </div>
                  <article v-else class="dash-empty-state compact">
                    <strong>No open proposal matches</strong>
                    <p>Open marketplace bounties will appear here when available.</p>
                  </article>
                </article>
              </section>
            </template>
          </template>

          <template v-else>
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

            <template v-else>
              <section ref="dashboardProjectHeader" class="dash-project-header" tabindex="-1">
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
                  <button type="button" @click="copyDashboardProjectLink">
                    <Share2 :size="15" />
                    Share
                  </button>
                  <button type="button" aria-label="More project actions" @click="openDashboardProjectTab('Settings')">
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
                  :class="{ active: tabItem === activeDashboardTab }"
                  type="button"
                  role="tab"
                  :aria-selected="tabItem === activeDashboardTab"
                  @click="openDashboardProjectTab(tabItem)"
                >
                  {{ tabItem }}
                  <span v-if="tabItem === 'Tasks'">{{ dashboardTaskRows.length }}</span>
                </button>
              </div>

              <section ref="dashboardOverviewPanel" class="dash-overview-grid" tabindex="-1">
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
                  <div class="card-title-row">
                    <div>
                      <h2>Budget & Payments</h2>
                      <p>{{ dashboardEscrowView.paidTasks }} paid / {{ dashboardEscrowView.openTasks }} open tasks</p>
                    </div>
                    <span>{{ dashboardEscrowView.status }}</span>
                  </div>
                  <div class="budget-lines">
                    <span>Total Budget <strong>{{ dashboardEscrowView.budget }}</strong></span>
                    <span>Work Pool <strong>{{ dashboardEscrowView.workPool }}</strong></span>
                    <span>Escrow Reserve <strong>{{ dashboardEscrowView.reserve }}</strong></span>
                    <span>Task Reserve <strong>{{ dashboardEscrowView.taskReserve }}</strong></span>
                    <span>Released <strong>{{ dashboardEscrowView.released }}</strong></span>
                    <span>Remaining <strong>{{ dashboardEscrowView.remaining }}</strong></span>
                  </div>
                  <p v-if="dashboardEscrowError" class="budget-warning red">{{ dashboardEscrowError }}</p>
                  <p v-else-if="dashboardEscrowView.hasOverdrawn" class="budget-warning red">Overdrawn by {{ dashboardEscrowView.overdrawn }}</p>
                  <p v-else-if="dashboardEscrowView.hasUnallocated" class="budget-warning">Unallocated pool: {{ dashboardEscrowView.unallocated }}</p>
                  <button type="button" @click="openDashboardSection('payments')">Open Payments</button>
                </article>

                <article class="dash-card analysis-card">
                  <div class="card-title-row">
                    <div>
                      <h2>Work Split</h2>
                      <p>{{ dashboardTaskGraphView.body }}</p>
                    </div>
                    <span>{{ dashboardTaskGraphView.status }}</span>
                  </div>
                  <div class="task-graph-progress">
                    <span><i :style="{ width: `${dashboardTaskGraphView.progress}%` }" /></span>
                    <strong>{{ dashboardTaskGraphView.progress }}%</strong>
                  </div>
                  <div class="risk-grid">
                    <span v-for="item in dashboardWorkSplit" :key="item.label" :class="item.className">
                      {{ item.label }}
                      <strong>{{ item.value }}</strong>
                    </span>
                  </div>
                  <div class="task-graph-metrics">
                    <span>Ready <strong>{{ dashboardTaskGraphView.ready }}</strong></span>
                    <span>Blocked <strong>{{ dashboardTaskGraphView.blocked }}</strong></span>
                    <span>Edges <strong>{{ dashboardTaskGraphView.edges }}</strong></span>
                  </div>
                  <div v-if="dashboardTaskGraphError" class="deployment-error">{{ dashboardTaskGraphError }}</div>
                  <div v-else-if="dashboardTaskGraphRows.length" class="task-graph-list">
                    <article v-for="node in dashboardTaskGraphRows" :key="node.id">
                      <span :class="['task-graph-icon', node.tone]">
                        <ListTodo :size="14" />
                      </span>
                      <div>
                        <strong>#{{ node.issueNumber }} {{ node.title }}</strong>
                        <small>{{ node.lane }} / {{ node.reward }} / {{ node.blockedBy }}</small>
                      </div>
                      <em :class="node.tone">{{ node.status }}</em>
                    </article>
                  </div>
                  <article v-else class="dash-empty-state compact">
                    <strong>{{ dashboardTaskGraphLoading ? 'Loading task graph...' : 'No graph nodes' }}</strong>
                    <p>{{ dashboardTaskGraphLoading ? 'Resolving dependency edges.' : 'Fund a project to generate graph nodes.' }}</p>
                  </article>
                  <button type="button" :disabled="dashboardTaskGraphLoading || !dashboardSelectedProject" @click="loadDashboardTaskGraphData(dashboardSelectedProject?.id)">Refresh Graph</button>
                </article>

                <article ref="dashboardRepositoryScanCard" class="dash-card repository-scan-card" tabindex="-1">
                  <div class="card-title-row">
                    <div>
                      <h2>Repository Scan</h2>
                      <p>{{ dashboardRepositoryScanView.body }}</p>
                    </div>
                    <button type="button" aria-label="Refresh repository scan" :disabled="dashboardRepositoryScanLoading || !dashboardSelectedProject" @click="loadDashboardRepositoryScanData(dashboardSelectedProject?.id)">
                      <RefreshCw :size="14" />
                    </button>
                  </div>
                  <div class="repository-scan-metrics" aria-label="Repository scan summary">
                    <article>
                      <span>Files</span>
                      <strong>{{ dashboardRepositoryScanView.scanned }}</strong>
                    </article>
                    <article>
                      <span>Findings</span>
                      <strong>{{ dashboardRepositoryScanView.findings }}</strong>
                    </article>
                    <article>
                      <span>Dependencies</span>
                      <strong>{{ dashboardRepositoryScanView.dependencies }}</strong>
                    </article>
                    <article>
                      <span>Status</span>
                      <strong>{{ dashboardRepositoryScanView.status }}</strong>
                    </article>
                  </div>
                  <div v-if="dashboardRepositoryScanError" class="deployment-error">{{ dashboardRepositoryScanError }}</div>
                  <div v-else-if="dashboardRepositoryScanFindings.length" class="repository-finding-list">
                    <article v-for="finding in dashboardRepositoryScanFindings" :key="finding.id">
                      <span :class="['repository-finding-icon', finding.tone]">
                        <Bug :size="15" />
                      </span>
                      <div>
                        <strong>{{ finding.title }}</strong>
                        <small>{{ finding.body }}</small>
                        <b>{{ finding.pathLine }}</b>
                      </div>
                      <em :class="finding.tone">{{ finding.severity }}</em>
                    </article>
                  </div>
                  <article v-else class="dash-empty-state compact">
                    <strong>{{ dashboardRepositoryScanLoading ? 'Scanning repository...' : 'No repository findings' }}</strong>
                    <p>{{ dashboardRepositoryScanLoading ? 'Loading code, dependency, and debt signals.' : `Last scan: ${dashboardRepositoryScanView.updatedAt}` }}</p>
                  </article>
                </article>

                <article class="dash-card deployment-card">
                  <div class="card-title-row">
                    <div>
                      <h2>Deployment Validation</h2>
                      <p>{{ dashboardDeploymentView.body }}</p>
                    </div>
                    <span>{{ dashboardDeploymentView.status }}</span>
                  </div>
                  <div class="deployment-status-row">
                    <span class="deployment-icon"><Rocket :size="18" /></span>
                    <div>
                      <strong>{{ dashboardDeploymentView.progress }}%</strong>
                      <small>{{ dashboardDeploymentView.updatedAt }}</small>
                    </div>
                    <button type="button" aria-label="Refresh deployment validation" :disabled="dashboardDeploymentLoading || !dashboardSelectedProject" @click="loadDashboardDeploymentData(dashboardSelectedProject?.id)">
                      <RefreshCw :size="14" />
                    </button>
                  </div>
                  <div v-if="dashboardDeploymentError" class="deployment-error">{{ dashboardDeploymentError }}</div>
                  <div v-else-if="dashboardDeploymentStages.length" class="deployment-stage-list">
                    <article v-for="stage in dashboardDeploymentStages" :key="stage.id">
                      <span :class="['notification-dot', stage.tone]" />
                      <div>
                        <strong>{{ stage.title }}</strong>
                        <small>{{ stage.body }}</small>
                        <b v-if="stage.reference">{{ stage.reference }}</b>
                      </div>
                      <em>{{ stage.status }}</em>
                    </article>
                  </div>
                  <article v-else class="dash-empty-state compact">
                    <strong>{{ dashboardDeploymentLoading ? 'Loading deployment...' : 'No deployment signal' }}</strong>
                    <p>{{ dashboardDeploymentLoading ? 'Syncing deployment stages.' : 'Select a funded project to load deployment validation.' }}</p>
                  </article>
                  <div v-if="dashboardDeploymentSignals.length" class="deployment-signal-strip">
                    <span v-for="signal in dashboardDeploymentSignals" :key="signal.id">
                      {{ signal.title }}
                    </span>
                  </div>
                </article>
              </section>

              <section ref="dashboardTasksPanel" class="dash-card live-pr-board" tabindex="-1">
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
            </template>
          </template>
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
                @click="selectDashboardProject(project.id)"
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

          <section ref="dashboardActivityPanel" class="dash-card rail-card" tabindex="-1">
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

          <section class="dash-card rail-card ai-workflow-card">
            <div class="card-title-row">
              <h2>AI Orchestration</h2>
              <span>{{ dashboardAIWorkflowView.status }}</span>
            </div>
            <div class="ai-workflow-summary">
              <span><Bot :size="16" /></span>
              <div>
                <strong>{{ dashboardAIWorkflowView.progress }}%</strong>
                <small>{{ dashboardAIWorkflowView.body }}</small>
              </div>
            </div>
            <div v-if="dashboardAIWorkflowError" class="deployment-error">{{ dashboardAIWorkflowError }}</div>
            <div v-else-if="dashboardAIWorkflowStages.length" class="ai-workflow-list">
              <article v-for="stage in dashboardAIWorkflowStages" :key="stage.id">
                <span :class="['notification-dot', stage.tone]" />
                <div>
                  <strong>{{ stage.title }}</strong>
                  <small>{{ stage.body }}</small>
                </div>
                <b>{{ stage.status }}</b>
              </article>
            </div>
            <article v-else class="dash-empty-state compact">
              <strong>{{ dashboardAIWorkflowLoading ? 'Loading workflow...' : 'No AI workflow yet' }}</strong>
              <p>{{ dashboardAIWorkflowLoading ? 'Syncing orchestration stages.' : 'Select a project to inspect AI routing.' }}</p>
            </article>
            <button class="rail-link-button" type="button" :disabled="dashboardAIWorkflowLoading || !dashboardSelectedProject" @click="loadDashboardAIWorkflowData(dashboardSelectedProject?.id)">
              Refresh workflow
            </button>
          </section>

          <section ref="dashboardNotificationCenter" class="dash-card rail-card notification-center-card" tabindex="-1">
            <div class="card-title-row">
              <h2>Notifications</h2>
              <span>{{ dashboardNotificationCount }} new</span>
            </div>
            <div v-if="dashboardNotificationRows.length" class="notification-center-list">
              <article
                v-for="note in dashboardNotificationRows"
                :key="note.id"
                :class="{ 'is-unread': note.isUnread }"
                role="button"
                tabindex="0"
                @click="handleNotificationClick(note)"
                @keyup.enter="handleNotificationClick(note)"
              >
                <span :class="['notification-dot', note.tone, { 'unread-pulse': note.isUnread }]" />
                <div>
                  <strong>{{ note.subject }}</strong>
                  <p>{{ note.body }}</p>
                  <small>{{ note.meta }}</small>
                </div>
                <span v-if="note.isUnread" class="notification-new-badge" aria-label="Unread">New</span>
              </article>
            </div>
            <article v-else class="dash-empty-state compact">
              <strong>{{ dashboardNotificationsLoading ? 'Loading notifications...' : 'No notifications yet' }}</strong>
              <p>{{ dashboardNotificationsLoading ? 'Fetching delivery records.' : dashboardNotificationsError || 'Project updates and delivery notices will appear here.' }}</p>
            </article>
            <div class="notification-actions-row">
              <button class="rail-link-button" type="button" @click="loadDashboardNotifications">
                Refresh
              </button>
              <button v-if="dashboardNotificationCount > 0" class="rail-link-button" type="button" @click="markAllNotificationsRead">
                Mark all read
              </button>
            </div>
          </section>

          <section ref="dashboardLedgerPanel" class="dash-card rail-card chat-card" tabindex="-1">
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
    <div v-if="toastMessage" class="toast" role="status" aria-live="polite">
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
          <a href="/live" :class="{ 'nav-active': publicPage === 'live' }" @click.prevent="openPublicPage('live')">Live Feed</a>
          <a href="/how-it-works" :class="{ 'nav-active': publicPage === 'how-it-works' }" @click.prevent="openPublicPage('how-it-works')">How it works</a>
          <a href="/ledger" :class="{ 'nav-active': publicPage === 'ledger' }" @click.prevent="openPublicPage('ledger')">Ledger Logs</a>
          <a href="/test-settings" :class="{ 'nav-active': publicPage === 'test-settings' }" @click.prevent="openPublicPage('test-settings')">Test Keys</a>
        </nav>

        <button
          class="hamburger-button"
          type="button"
          :aria-label="mobileMenuOpen ? 'Close navigation' : 'Open navigation'"
          :aria-expanded="mobileMenuOpen"
          aria-controls="mobile-nav-panel"
          @click="mobileMenuOpen = !mobileMenuOpen"
        >
          <Menu v-if="!mobileMenuOpen" :size="22" />
          <X v-else :size="22" />
        </button>

        <div v-if="mobileMenuOpen" class="mobile-nav-overlay" aria-hidden="true" @click="mobileMenuOpen = false"></div>
        <nav v-if="mobileMenuOpen" id="mobile-nav-panel" class="mobile-nav-panel" aria-label="Mobile navigation">
          <a href="/product" :class="{ 'nav-active': publicPage === 'product' }" @click.prevent="mobileMenuOpen = false; openPublicPage('product')">Product <ChevronDown :size="13" /></a>
          <a href="/solutions" :class="{ 'nav-active': publicPage === 'solutions' }" @click.prevent="mobileMenuOpen = false; openPublicPage('solutions')">Solutions <ChevronDown :size="13" /></a>
          <a href="/marketplace" :class="{ 'nav-active': publicPage === 'marketplace' }" @click.prevent="mobileMenuOpen = false; openPublicPage('marketplace')">Marketplace</a>
          <a href="/live" :class="{ 'nav-active': publicPage === 'live' }" @click.prevent="mobileMenuOpen = false; openPublicPage('live')">Live Feed</a>
          <a href="/how-it-works" :class="{ 'nav-active': publicPage === 'how-it-works' }" @click.prevent="mobileMenuOpen = false; openPublicPage('how-it-works')">How it works</a>
          <a href="/ledger" :class="{ 'nav-active': publicPage === 'ledger' }" @click.prevent="mobileMenuOpen = false; openPublicPage('ledger')">Ledger Logs</a>
          <a href="/test-settings" :class="{ 'nav-active': publicPage === 'test-settings' }" @click.prevent="mobileMenuOpen = false; openPublicPage('test-settings')">Test Keys</a>
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
            <span class="marketplace-eyebrow">MERGEOS DELIVERY OS</span>
            <h1 id="home-title">MergeOS turns funded software work into verified delivery.</h1>
            <p>Post a brief, fund escrow, route tasks to builders or agents, and prove every payout through the live ledger.</p>
            <div class="marketplace-actions">
              <button class="primary-button large" type="button" @click="openProjectWizard">
                Start a project
                <ArrowRight :size="16" />
              </button>
              <button class="secondary-button large" type="button" @click="openPublicPage('marketplace')">
                View live work
                <UsersRound :size="16" />
              </button>
              <button class="secondary-button large" type="button" @click="openPublicPage('ledger')">
                Open proof ledger
                <Link2 :size="16" />
              </button>
            </div>

            <div class="home-proof-stack" aria-label="Delivery guarantees">
              <article>
                <ShieldCheck :size="17" />
                <span>Escrow first</span>
              </article>
              <article>
                <GitPullRequest :size="17" />
                <span>Repo-aware tasks</span>
              </article>
              <article>
                <BarChart3 :size="17" />
                <span>Ledger proof</span>
              </article>
            </div>
          </div>

          <aside class="public-home-panel home-command-panel" aria-label="Live platform summary">
            <div class="home-command-head">
              <span class="home-command-mark" aria-hidden="true">
                <img src="/favicon.svg" alt="" />
              </span>
              <div>
                <span>Live command center</span>
                <strong>Marketplace, tasks, escrow, ledger</strong>
              </div>
              <span class="ledger-live-dot">Live</span>
            </div>

            <div class="public-stat-grid">
              <article v-for="stat in homeLiveStats" :key="stat.label">
                <strong>{{ stat.value }}</strong>
                <span>{{ stat.label }}</span>
              </article>
            </div>

            <div class="home-pipeline" aria-label="Project pipeline">
              <article>
                <span><FileCheck2 :size="15" /></span>
                <strong>Brief</strong>
                <small>Scope and acceptance criteria</small>
              </article>
              <article>
                <span><CreditCard :size="15" /></span>
                <strong>Fund</strong>
                <small>Escrow and token mint</small>
              </article>
              <article>
                <span><CheckCircle2 :size="15" /></span>
                <strong>Verify</strong>
                <small>PR review and payout log</small>
              </article>
            </div>

            <div class="public-notification-feed" aria-live="polite">
              <div class="public-notification-head">
                <span>
                  <Bell :size="15" />
                </span>
                <strong>Recent updates</strong>
                <small>{{ publicNotificationRows.length }}</small>
              </div>
              <article v-for="note in publicNotificationRows.slice(0, 3)" :key="note.id">
                <i :class="['notification-dot', note.tone]" />
                <div>
                  <strong>{{ note.subject }}</strong>
                  <p>{{ note.body }}</p>
                  <small>{{ note.meta }}</small>
                </div>
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
            <span class="marketplace-eyebrow">DELIVERY LANES</span>
            <h2>Choose human builders, agents, or a hybrid lane from the same funded workflow.</h2>
            <p>Browse live work and contributor signals before login. Sign in only when a project, wallet, or payment needs to be attached.</p>
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

    <main v-else-if="publicPage === 'test-settings'" id="top" class="test-settings-page">
      <div class="home-container test-settings-shell">
        <section class="test-settings-hero">
          <div>
            <span class="marketplace-eyebrow">TEST PUBLISH SETTINGS</span>
            <div class="ledger-title-row">
              <h1>Public Test Keys</h1>
              <span class="ledger-public-badge">
                <LockKeyhole :size="14" />
                {{ publicTestSettingsModeLabel }}
              </span>
            </div>
            <p>Password-gated test keys for temporary LLM, PayPal, and USDT task integrations.</p>
          </div>
          <button class="secondary-button compact" type="button" @click="loadPublicTestSettingsStatus">
            <RefreshCw :size="14" />
            Refresh status
          </button>
        </section>

        <section class="test-settings-grid">
          <article class="test-settings-card">
            <div class="card-title-row">
              <div>
                <h2>Access</h2>
                <p>{{ publicTestSettingsStatus.test_mode_enabled ? 'Enter the shared test password.' : 'Test mode is currently disabled.' }}</p>
              </div>
              <span>{{ publicTestSettingsAuthenticated ? 'Unlocked' : 'Locked' }}</span>
            </div>
            <form class="test-settings-form" @submit.prevent="unlockPublicTestSettings">
              <label>
                <span>Password</span>
                <input v-model="publicTestSettingsPassword" type="password" autocomplete="current-password" placeholder="Shared test password" />
              </label>
              <button class="primary-button compact" :disabled="publicTestSettingsBusy || !publicTestSettingsStatus.test_mode_enabled" type="submit">
                {{ publicTestSettingsBusy ? 'Checking...' : 'Unlock' }}
              </button>
            </form>
            <p v-if="publicTestSettingsError" class="modal-error">{{ publicTestSettingsError }}</p>
          </article>

          <article class="test-settings-card">
            <div class="card-title-row">
              <div>
                <h2>Add Test Key</h2>
                <p>Saved to database and rejected if the key collides with ENV names.</p>
              </div>
              <span>{{ publicTestSettingsRows.length }} keys</span>
            </div>
            <form class="test-settings-form" @submit.prevent="addPublicTestSettingsEntry">
              <div class="test-settings-type-row" role="radiogroup" aria-label="Integration type">
                <button
                  v-for="option in publicTestSettingsIntegrationOptions"
                  :key="option.value"
                  :class="{ active: publicTestSettingsForm.integrationType === option.value }"
                  type="button"
                  @click="publicTestSettingsForm.integrationType = option.value"
                >
                  {{ option.label }}
                </button>
              </div>
              <label>
                <span>Display name</span>
                <input v-model.trim="publicTestSettingsForm.displayName" placeholder="Task LLM key" />
              </label>
              <label>
                <span>Setting key</span>
                <input v-model.trim="publicTestSettingsForm.settingKey" placeholder="TASK_LLM_TEST_KEY" />
              </label>
              <label>
                <span>Primary value</span>
                <input v-model="publicTestSettingsForm.settingValue" placeholder="Paste test value" />
              </label>
              <div class="test-settings-kv-list">
                <div class="test-settings-kv-head">
                  <span>Key value map</span>
                  <button type="button" @click="addPublicTestSettingsKVRow">
                    <Plus :size="13" />
                    Add
                  </button>
                </div>
                <div v-for="(row, index) in publicTestSettingsKeyValueRows" :key="index" class="test-settings-kv-row">
                  <input v-model.trim="row.key" placeholder="name" />
                  <input v-model="row.value" placeholder="value" />
                  <button type="button" :aria-label="`Remove key ${index + 1}`" @click="removePublicTestSettingsKVRow(index)">
                    <X :size="13" />
                  </button>
                </div>
              </div>
              <button class="primary-button compact" :disabled="publicTestSettingsBusy || !publicTestSettingsAuthenticated" type="submit">
                {{ publicTestSettingsBusy ? 'Saving...' : 'Save Test Key' }}
              </button>
            </form>
          </article>
        </section>

        <section class="test-settings-card test-settings-list-card">
          <div class="card-title-row">
            <div>
              <h2>Published Test Keys</h2>
              <p>{{ publicTestSettingsAuthenticated ? 'Values are masked after save.' : 'Unlock to load database entries.' }}</p>
            </div>
            <button class="secondary-button compact" :disabled="publicTestSettingsBusy || !publicTestSettingsAuthenticated" type="button" @click="loadPublicTestSettingsEntries">
              <RefreshCw :size="14" />
              Refresh
            </button>
          </div>
          <div v-if="publicTestSettingsLoading" class="live-feed-state">Loading test keys...</div>
          <div v-else-if="publicTestSettingsRows.length" class="test-settings-entry-list">
            <article v-for="entry in publicTestSettingsRows" :key="entry.id">
              <span class="test-settings-entry-icon">
                <LockKeyhole :size="16" />
              </span>
              <div>
                <strong>{{ entry.displayName }}</strong>
                <small>{{ entry.integrationType }} / {{ entry.settingKey }} / {{ entry.valueHint }}</small>
                <span v-if="entry.mapKeys.length">{{ entry.mapKeys.join(', ') }}</span>
              </div>
              <div class="test-settings-entry-side">
                <b>{{ entry.status }}</b>
                <small>{{ entry.updatedAt }}</small>
                <button type="button" @click="deletePublicTestSettingsEntry(entry.id)">Delete</button>
              </div>
            </article>
          </div>
          <article v-else class="marketplace-empty-state compact">
            <strong>{{ publicTestSettingsAuthenticated ? 'No test keys yet' : 'Locked' }}</strong>
            <p>{{ publicTestSettingsAuthenticated ? 'Add the first test integration key above.' : 'Enter the public test password to view entries.' }}</p>
          </article>
        </section>
      </div>
    </main>

    <main v-else-if="publicPage === 'live'" id="top" class="ledger-page live-feed-page">
      <div class="home-container ledger-shell live-feed-shell">
        <section class="ledger-hero">
          <div class="ledger-hero-copy">
            <span class="marketplace-eyebrow">LIVE FEED</span>
            <div class="ledger-title-row">
              <h1>Live Feed</h1>
              <span class="ledger-public-badge">
                <Zap :size="14" />
                Realtime
              </span>
            </div>
            <p>Public MergeOS activity from funded projects, open bounty tasks, PR payouts, AI review webhooks, and ledger-backed payment events.</p>

            <div class="ledger-trust-row" aria-label="Live feed trust signals">
              <article>
                <span class="ledger-trust-icon green">
                  <GitPullRequest :size="16" />
                </span>
                <div>
                  <strong>PR and task events</strong>
                  <small>Accepted work and open tasks</small>
                </div>
              </article>
              <article>
                <span class="ledger-trust-icon blue">
                  <Bot :size="16" />
                </span>
                <div>
                  <strong>AI actions</strong>
                  <small>Review webhook status</small>
                </div>
              </article>
              <article>
                <span class="ledger-trust-icon green">
                  <ShieldCheck :size="16" />
                </span>
                <div>
                  <strong>Ledger proof</strong>
                  <small>Sanitized public references</small>
                </div>
              </article>
            </div>
          </div>

          <aside class="ledger-live-card" aria-label="Live feed metrics">
            <div class="ledger-card-head">
              <h2>Realtime Snapshot</h2>
              <span class="ledger-live-dot">Live</span>
            </div>
            <div class="ledger-live-grid">
              <article v-for="stat in liveFeedStats" :key="stat.label">
                <strong>{{ stat.value }}</strong>
                <span>{{ stat.label }}</span>
              </article>
            </div>
            <button type="button" @click="loadLiveFeedData">
              <RefreshCw :size="14" />
              Refresh live feed
              <ArrowRight :size="14" />
            </button>
          </aside>
        </section>

        <section class="live-feed-content">
          <div class="ledger-main-card live-feed-main-card">
            <div class="live-feed-toolbar">
              <div>
                <span class="marketplace-eyebrow">PUBLIC TIMELINE</span>
                <h2>Delivery activity</h2>
              </div>
              <div class="ledger-table-actions">
                <button type="button" @click="loadLiveFeedData">
                  <RefreshCw :size="14" />
                  Refresh
                </button>
                <button type="button" @click="openPublicPage('ledger')">
                  Ledger Logs
                  <ArrowRight :size="13" />
                </button>
              </div>
            </div>

            <div v-if="liveFeedLoading" class="live-feed-state">Loading public activity...</div>
            <div v-else-if="liveFeedError" class="live-feed-state error">{{ liveFeedError }}</div>
            <div v-else-if="!filteredLiveFeedItems.length" class="live-feed-state">{{ liveFeedEmptyStateCopy }}</div>
            <div v-else class="live-feed-list">
              <article v-for="item in filteredLiveFeedItems" :key="item.id" class="live-feed-row">
                <span :class="['ledger-event-type', item.tone]">
                  <component :is="item.icon" :size="15" />
                  {{ item.typeLabel }}
                </span>
                <div class="live-feed-row-copy">
                  <div>
                    <strong>{{ item.title }}</strong>
                    <span>{{ item.status }}</span>
                  </div>
                  <p>{{ item.body }}</p>
                  <small>{{ item.project }} · {{ item.actor }} · {{ item.meta }}</small>
                </div>
                <div class="live-feed-row-meta">
                  <strong v-if="item.amount" :class="['ledger-amount', item.amountTone]">{{ item.amount }}</strong>
                  <span v-else>{{ item.date }}</span>
                  <button v-if="item.url" class="ledger-ref-button" type="button" @click="openExternalURL(item.url)">
                    {{ item.reference || 'Open' }}
                    <Link2 :size="12" />
                  </button>
                  <span v-else-if="item.reference">{{ item.reference }}</span>
                </div>
              </article>
            </div>
          </div>

          <aside class="ledger-rail">
            <section class="ledger-side-card">
              <div class="side-card-head">
                <h2>Activity Types</h2>
                <button type="button" @click="loadLiveFeedData">Refresh</button>
              </div>
              <div class="live-feed-type-list">
                <button
                  v-for="row in liveFeedActivityTypes"
                  :key="row.label"
                  :class="{ active: row.label === activeLiveFeedType }"
                  type="button"
                  :aria-pressed="row.label === activeLiveFeedType"
                  @click="activeLiveFeedType = row.label"
                >
                  <span :class="['notification-dot', row.tone]" />
                  <strong>{{ row.label }}</strong>
                  <small>{{ row.count }}</small>
                </button>
                <article v-if="!liveFeedItemsView.length">
                  <span class="notification-dot blue" />
                  <strong>Waiting for events</strong>
                  <small>0</small>
                </article>
              </div>
            </section>

            <section class="ledger-side-card">
              <h2>Latest Signal</h2>
              <div v-if="liveFeedLatestProject" class="live-feed-latest">
                <span :class="['ledger-project-logo', liveFeedLatestProject.tone]">{{ projectInitialFor(liveFeedLatestProject.project) }}</span>
                <div>
                  <strong>{{ liveFeedLatestProject.title }}</strong>
                  <p>{{ liveFeedLatestProject.body }}</p>
                  <small>{{ liveFeedLatestProject.meta }}</small>
                </div>
              </div>
              <p v-else class="live-feed-empty-copy">No current signal.</p>
            </section>
          </aside>
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
                  :class="{ active: tabItem === activeLedgerTab }"
                  type="button"
                  role="tab"
                  :aria-selected="tabItem === activeLedgerTab"
                  @click="activeLedgerTab = tabItem"
                >
                  {{ tabItem }}
                </button>
              </div>

              <div class="ledger-table-actions">
                <label class="ledger-project-filter">
                  <span>Project</span>
                  <select v-model="activeLedgerProjectFilter" aria-label="Filter ledger by project">
                    <option v-for="project in ledgerProjectFilterOptions" :key="project" :value="project">{{ project }}</option>
                  </select>
                  <ChevronDown :size="13" />
                </label>
                <button type="button" :disabled="!ledgerFiltersActive" @click="resetLedgerFilters">
                  <Filter :size="14" />
                  Reset
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
                  <tr v-else-if="filteredLedgerEvents.length === 0">
                    <td class="ledger-state-cell" colspan="6">{{ ledgerEmptyStateCopy }}</td>
                  </tr>
                  <template v-else>
                    <tr v-for="event in filteredLedgerEvents" :key="event.key">
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
                <button type="button" @click="openMarketplaceSection('marketplace-projects')">View all <ArrowRight :size="13" /></button>
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
              <button type="button" @click="openPublicPage('how-it-works')">
                Learn more about transparency
                <ArrowRight :size="13" />
              </button>
            </section>

            <section class="ledger-side-card ledger-chain-card">
              <h2>Explore on-chain</h2>
              <p>All transactions are recorded on-chain and verifiable on the blockchain.</p>
              <button type="button" @click="openExternalURL(scanBaseURL())">
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
                  <button aria-label="More AI agent actions" type="button" @click="openMarketplaceSection('marketplace-agents')">
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
              <button
                v-for="filter in marketplaceFilters"
                :key="filter"
                :class="{ active: filter === activeMarketplaceFilter }"
                type="button"
                :aria-pressed="filter === activeMarketplaceFilter"
                @click="activeMarketplaceFilter = filter"
              >
                {{ filter }}
                <ChevronDown :size="14" />
              </button>
              <button class="more-filter-button" type="button" @click="resetMarketplaceFilters">
                <Filter :size="15" />
                Reset filters
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
              <button v-if="!marketplaceLoading" class="secondary-button compact" type="button" @click="resetMarketplaceFilters">
                Clear filters
              </button>
            </article>

            <button class="view-projects-button" type="button" @click="loadMarketplaceData">
              Refresh live data
              <ArrowRight :size="15" />
            </button>
          </section>

          <section id="marketplace-bounties" class="marketplace-bounty-section" aria-label="Open bounties">
            <div class="section-heading-row">
              <h2>
                <ListTodo :size="17" />
                Open Bounties
              </h2>
              <div class="marketplace-data-status">
                <span>{{ marketplaceBountiesView.length }} live bounty tasks</span>
              </div>
            </div>

            <div v-if="marketplaceBountiesView.length" class="marketplace-bounty-list">
              <article v-for="bounty in marketplaceBountiesView" :key="bounty.id">
                <span :class="['marketplace-bounty-icon', bounty.tone]">
                  <component :is="bounty.icon" :size="18" />
                </span>
                <div>
                  <div class="marketplace-bounty-title">
                    <strong>{{ bounty.title }}</strong>
                    <span>{{ bounty.issue }}</span>
                  </div>
                  <p>{{ bounty.acceptance }}</p>
                  <small>{{ bounty.project }} · {{ bounty.lane }}</small>
                </div>
                <div class="marketplace-bounty-meta">
                  <strong>{{ bounty.reward }}</strong>
                  <div class="marketplace-bounty-actions">
                    <button v-if="bounty.url" type="button" @click="openExternalURL(bounty.url)">
                      Issue
                      <Link2 :size="12" />
                    </button>
                    <button type="button" :disabled="!bounty.claimCommand" @click="copyClaimCommand(bounty.claimCommand)">
                      Copy Claim
                    </button>
                  </div>
                </div>
              </article>
            </div>
            <article v-else class="marketplace-empty-state compact">
              <strong>{{ marketplaceLoading ? 'Loading bounties...' : 'No open bounties' }}</strong>
              <p>{{ marketplaceLoading ? 'Fetching task-level marketplace rows.' : 'Funded tasks that are still open will appear here.' }}</p>
            </article>
          </section>

          <section id="marketplace-agents" class="marketplace-agent-section" aria-label="AI agent operations">
            <div class="section-heading-row">
              <h2>
                <Bot :size="17" />
                AI Agent Operations
              </h2>
              <div class="marketplace-data-status">
                <span>{{ marketplaceAgentQueueRows.length }} active agent lanes</span>
              </div>
            </div>

            <div v-if="marketplaceAgentQueueRows.length" class="marketplace-agent-grid">
              <article v-for="agent in marketplaceAgentQueueRows" :key="agent.type">
                <div class="marketplace-agent-head">
                  <span :class="['popular-agent-icon', agent.tone]">
                    <component :is="agent.icon" :size="21" />
                  </span>
                  <div>
                    <strong>{{ agent.title }}</strong>
                    <small>{{ agent.workerKind }} / {{ agent.status }}</small>
                  </div>
                </div>
                <p>{{ agent.body }}</p>
                <div class="marketplace-agent-stats">
                  <span>
                    <strong>{{ agent.openTasks }}</strong>
                    <small>Open</small>
                  </span>
                  <span>
                    <strong>{{ agent.totalTasks }}</strong>
                    <small>Total</small>
                  </span>
                  <span>
                    <strong>{{ agent.budget }}</strong>
                    <small>Pool</small>
                  </span>
                </div>
                <div class="marketplace-agent-capabilities">
                  <span v-for="capability in agent.capabilities" :key="capability">{{ capability }}</span>
                </div>
                <div class="marketplace-agent-task-list">
                  <small v-for="task in agent.nextTasks" :key="task.id">{{ task.issue }} / {{ task.title }} / {{ task.reward }}</small>
                  <small v-if="!agent.nextTasks.length">No open task attached to this lane yet.</small>
                </div>
              </article>
            </div>
            <article v-else class="marketplace-empty-state compact">
              <strong>{{ marketplaceLoading ? 'Loading agent lanes...' : 'No AI agent lanes yet' }}</strong>
              <p>{{ marketplaceLoading ? 'Fetching agent work queue from live marketplace data.' : 'AI-scoped tasks will appear here after repository scan and task generation.' }}</p>
            </article>
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

        <aside id="marketplace-contributors" class="marketplace-rail">
          <section class="marketplace-side-card">
            <div class="side-card-head">
              <h2>Top Contributors</h2>
              <button type="button" @click="openMarketplaceSection('marketplace-contributors')">View all</button>
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
      <section ref="authDialog" class="auth-modal" role="dialog" aria-modal="true" aria-labelledby="auth-title" tabindex="-1" @keydown.esc="closeAuth">
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
import { computed, nextTick, onMounted, onUnmounted, reactive, ref, watch } from 'vue';
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
  FileText,
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
  Menu,
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
  live: '/live',
  'how-it-works': '/how-it-works',
  ledger: '/ledger',
  'test-settings': '/test-settings',
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
const projectDraftStorageKey = 'mergeos_project_setup_draft';

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
const authDialog = ref(null);
const authMode = ref('login');
const authBusy = ref(false);
const authRememberMe = ref(false);
const authTermsAccepted = ref(true);
const errorMessage = ref('');
const mobileMenuOpen = ref(false);
const showPassword = ref(false);
const showConfirmPassword = ref(false);
const toastMessage = ref('');
const publicNotifications = ref([]);
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
const liveFeedData = ref({
  stats: {},
  items: [],
});
const liveFeedLoading = ref(true);
const liveFeedError = ref('');
const activeLiveFeedType = ref('All Activity');
const marketplaceData = ref({
  stats: {},
  projects: [],
  bounties: [],
  contributors: [],
  agents: [],
});
const marketplaceLoading = ref(true);
const marketplaceError = ref('');
const marketplaceSearch = ref('');
const activeMarketplaceCategory = ref('All');
const activeMarketplaceFilter = ref('Category');
const activeLedgerTab = ref('All Activity');
const activeLedgerProjectFilter = ref('All Projects');
const publicTestSettingsStatus = ref({ test_mode_enabled: false });
const publicTestSettingsEntries = ref([]);
const publicTestSettingsPassword = ref('');
const publicTestSettingsAuthenticated = ref(false);
const publicTestSettingsLoading = ref(false);
const publicTestSettingsBusy = ref(false);
const publicTestSettingsError = ref('');
const publicTestSettingsForm = reactive({
  integrationType: 'llm',
  displayName: '',
  settingKey: '',
  settingValue: '',
});
const publicTestSettingsKeyValueRows = ref([{ key: '', value: '' }]);
const dashboardProjects = ref([]);
const dashboardTasks = ref([]);
const dashboardLedgerEntries = ref([]);
const dashboardEscrow = ref(null);
const dashboardEscrowLoading = ref(false);
const dashboardEscrowError = ref('');
const dashboardDeployment = ref(null);
const dashboardDeploymentLoading = ref(false);
const dashboardDeploymentError = ref('');
const dashboardAIWorkflow = ref(null);
const dashboardAIWorkflowLoading = ref(false);
const dashboardAIWorkflowError = ref('');
const dashboardTaskGraph = ref(null);
const dashboardTaskGraphLoading = ref(false);
const dashboardTaskGraphError = ref('');
const dashboardPullRequests = ref(null);
const dashboardPullRequestsLoading = ref(false);
const dashboardPullRequestsError = ref('');
const dashboardRepositoryScan = ref(null);
const dashboardRepositoryScanLoading = ref(false);
const dashboardRepositoryScanError = ref('');
const dashboardNotifications = ref([]);
const dashboardNotificationsLoading = ref(false);
const dashboardNotificationsError = ref('');
const adminSummary = ref(null);
const adminOpsQueue = ref({ stats: {}, items: [] });
const adminReputation = ref({ stats: {}, workers: [] });
const adminUsers = ref([]);
const adminTasks = ref([]);
const adminTaskPulls = ref({});
const adminTaskPullsLoadingID = ref('');
const adminTaskPullsError = ref('');
const adminMergeBusyID = ref('');
const adminMergeError = ref('');
const adminMergeResult = ref(null);
const adminSSLReviews = ref([]);
const adminSSLReviewBusy = ref(false);
const adminSSLReviewError = ref('');
const adminSettings = ref({ llm_provider_options: [] });
const adminLLMKeys = ref([]);
const adminLLMWebhooks = ref([]);
const adminLLMBusy = ref(false);
const adminLLMError = ref('');
const adminLLMKeyBusyID = ref('');
const adminLLMForm = reactive({
  provider: 'gemini',
  model: 'gemini-2.5-flash',
  apiKey: '',
});
const adminTestSettings = ref({ test_mode_enabled: false, updated_at: '' });
const adminTestSettingsEntries = ref([]);
const adminTestSettingsPassword = ref('');
const adminTestSettingsBusy = ref(false);
const adminTestSettingsError = ref('');
const adminConsoleLoading = ref(false);
const adminConsoleError = ref('');
const adminCreditBusy = ref(false);
const adminCreditError = ref('');
const adminCreditResult = ref(null);
const adminCreditForm = reactive({
  workerID: '',
  rewardMRG: 50,
  bountyType: 'future-small',
  taskID: '',
  prURL: '',
  prTitle: '',
  reference: '',
});
const adminMergeForm = reactive({
  rewardMRG: 50,
  bountyType: 'future-small',
});
const workerDashboard = ref({
  profile: {},
  stats: {},
  claimed_tasks: [],
  rewards: [],
  reputation: [],
  proposals: [],
  identity_status: [],
});
const workerDashboardLoading = ref(false);
const workerDashboardError = ref('');
const dashboardLoading = ref(false);
const dashboardError = ref('');
const dashboardSearch = ref('');
const dashboardSection = ref('projects');
const activeDashboardTab = ref('Overview');
const selectedDashboardProjectID = ref('');
const dashboardProjectHeader = ref(null);
const dashboardOverviewPanel = ref(null);
const dashboardTasksPanel = ref(null);
const dashboardRepositoryScanCard = ref(null);
const dashboardActivityPanel = ref(null);
const dashboardLedgerPanel = ref(null);
const dashboardNotificationCenter = ref(null);
const repoImportInput = ref(null);
const attachmentInput = ref(null);
const priceEvaluation = ref(null);
const priceEvaluationBusy = ref(false);
const priceEvaluationError = ref('');
const repoImportBusy = ref(false);
const repoImportError = ref('');
const repoImportResult = ref(null);
const projectAttachments = ref([]);
const attachmentUploadBusy = ref(false);
const attachmentUploadError = ref('');
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
  { label: 'New Project', caption: 'Brand-new project or idea from scratch', icon: Sparkles },
  { label: 'Bug Fix', caption: 'Fix an issue in an existing repository', icon: Bug },
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
  { label: 'Complete your project details', action: 'dashboard', tab: 'Overview' },
  { label: 'Invite team members', action: 'invite', section: 'marketplace-contributors' },
  { label: 'Boost your project', action: 'marketplace', section: 'marketplace-projects' },
  { label: 'Explore your dashboard', action: 'dashboard', tab: 'Tasks' },
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

watch(authVisible, async (visible) => {
  if (!visible) return;
  await nextTick();
  authDialog.value?.focus();
});

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
const ledgerTabTypes = {
  'Escrow & Payments': new Set(['payment_verified', 'platform_fee', 'project_reserve', 'task_reserve', 'task_payment']),
  'Tasks & PRs': new Set(['task_reserve', 'task_payment']),
  Milestones: new Set(['project_reserve', 'task_reserve']),
  'Token Events': new Set(['token_mint']),
};

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
const ledgerTabFilteredEvents = computed(() => {
  const activeTab = activeLedgerTab.value;
  if (activeTab === 'All Activity') return ledgerEvents.value;
  if (activeTab === 'AI Actions') {
    return ledgerEvents.value.filter((event) => event.rawType.includes('ai'));
  }
  const allowedTypes = ledgerTabTypes[activeTab];
  if (!allowedTypes) return ledgerEvents.value;
  return ledgerEvents.value.filter((event) => allowedTypes.has(event.rawType));
});
const ledgerProjectFilterOptions = computed(() => {
  const projects = new Set(['All Projects']);
  for (const event of ledgerEvents.value) {
    if (event.project) projects.add(event.project);
  }
  return Array.from(projects);
});
const filteredLedgerEvents = computed(() => {
  if (activeLedgerProjectFilter.value === 'All Projects') return ledgerTabFilteredEvents.value;
  return ledgerTabFilteredEvents.value.filter((event) => event.project === activeLedgerProjectFilter.value);
});
const ledgerFiltersActive = computed(() =>
  activeLedgerTab.value !== 'All Activity' || activeLedgerProjectFilter.value !== 'All Projects',
);
const ledgerEmptyStateCopy = computed(() =>
  activeLedgerProjectFilter.value !== 'All Projects'
    ? `No ${activeLedgerTab.value.toLowerCase()} entries for ${activeLedgerProjectFilter.value}.`
    : activeLedgerTab.value === 'All Activity'
    ? 'No ledger entries yet. Fund a project to mint tokens and create the first logs.'
    : `No ${activeLedgerTab.value.toLowerCase()} entries yet.`,
);
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

const liveFeedStats = computed(() => {
  const stats = liveFeedData.value?.stats || {};
  return [
    { value: String(Number(stats.project_count) || publicProjectCount.value), label: 'Projects' },
    { value: String(Number(stats.open_task_count) || 0), label: 'Open tasks' },
    { value: String(Number(stats.ai_action_count) || 0), label: 'AI actions' },
    { value: formatPublicMRGFromCents(stats.total_budget_cents || publicVerifiedFundingCents.value), label: 'Escrow' },
  ];
});
const liveFeedItemsView = computed(() =>
  (liveFeedData.value.items || []).map(mapPublicLiveFeedItem),
);
const liveFeedActivityTypes = computed(() => {
  const counts = new Map();
  for (const item of liveFeedItemsView.value) {
    counts.set(item.typeLabel, (counts.get(item.typeLabel) || 0) + 1);
  }
  const rows = Array.from(counts.entries()).slice(0, 6).map(([label, count], index) => ({
    label,
    count,
    tone: ['green', 'blue', 'purple', 'amber', 'slate', 'green'][index % 6],
  }));
  return [
    { label: 'All Activity', count: liveFeedItemsView.value.length, tone: 'blue' },
    ...rows,
  ];
});
const filteredLiveFeedItems = computed(() => {
  if (activeLiveFeedType.value === 'All Activity') return liveFeedItemsView.value;
  return liveFeedItemsView.value.filter((item) => item.typeLabel === activeLiveFeedType.value);
});
const liveFeedEmptyStateCopy = computed(() =>
  activeLiveFeedType.value === 'All Activity'
    ? 'No live activity yet. Fund a project to create the first public events.'
    : `No ${activeLiveFeedType.value.toLowerCase()} events yet.`,
);
const liveFeedLatestProject = computed(() =>
  filteredLiveFeedItems.value.find((item) => item.rawType === 'project_funded') || filteredLiveFeedItems.value[0] || null,
);

const marketplaceFilters = ['Category', 'Budget', 'Delivery time'];

const publicTestSettingsIntegrationOptions = [
  { label: 'LLM', value: 'llm' },
  { label: 'PayPal', value: 'paypal' },
  { label: 'USDT', value: 'usdt' },
];

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
const publicNotificationRows = computed(() => {
  const actionRows = publicNotifications.value.map((note) => ({
    id: note.id,
    subject: note.subject,
    body: note.body,
    meta: note.meta,
    tone: note.tone,
    createdAt: note.createdAt,
  }));
  const liveRows = liveFeedItemsView.value.slice(0, 4).map((item) => ({
    id: `live-${item.id}`,
    subject: item.title,
    body: item.body,
    meta: item.meta,
    tone: item.tone === 'green' || item.tone === 'blue' ? item.tone : 'blue',
    createdAt: item.createdAt,
  }));
  const ledgerRows = ledgerEvents.value.slice(0, 4).map((event) => ({
    id: `ledger-${event.key}`,
    subject: event.type,
    body: `${event.project} recorded ${event.amount}.`,
    meta: event.time,
    tone: event.tone === 'green' || event.tone === 'blue' ? event.tone : 'blue',
    createdAt: event.createdAt,
  }));
  const rows = [...actionRows, ...liveRows, ...ledgerRows].sort((a, b) => new Date(b.createdAt || 0) - new Date(a.createdAt || 0));
  if (rows.length) return rows;
  return [{
    id: 'empty-public-notification',
    subject: marketplaceLoading.value ? 'Loading platform updates' : 'No live updates yet',
    body: marketplaceLoading.value ? 'Fetching the latest ledger and marketplace status.' : 'Funding, marketplace, and ledger activity will appear here.',
    meta: marketplaceLoading.value ? 'Syncing' : 'Waiting for activity',
    tone: 'blue',
    createdAt: new Date(0).toISOString(),
  }];
});

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
const publicTestSettingsModeLabel = computed(() =>
  publicTestSettingsStatus.value?.test_mode_enabled ? 'Test mode active' : 'Test mode disabled',
);
const publicTestSettingsRows = computed(() =>
  (publicTestSettingsEntries.value || []).map((entry) => {
    const mapKeys = Object.keys(entry.key_value_map || {});
    const when = formatLedgerDateTime(entry.updated_at);
    return {
      id: entry.id,
      integrationType: toTitleLabel(entry.integration_type || 'test'),
      displayName: entry.display_name || entry.setting_key || 'Test key',
      settingKey: entry.setting_key || '-',
      valueHint: entry.setting_value_hint || '****',
      mapKeys,
      status: toTitleLabel(entry.status || 'active'),
      updatedAt: when.full,
    };
  }),
);

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
  const projects = (marketplaceData.value.projects || [])
    .map(mapMarketplaceProject)
    .filter((project) => {
      const matchesCategory = active === 'All' || project.tags.some((tag) => toTitleLabel(tag) === active);
      const matchesSearch = !query || marketplaceSearchHaystack(project).includes(query);
      return matchesCategory && matchesSearch;
    });
  return sortMarketplaceRows(projects, activeMarketplaceFilter.value);
});
const marketplaceBountiesView = computed(() =>
  (marketplaceData.value.bounties || [])
    .map(mapMarketplaceBounty)
    .filter((bounty) => !marketplaceSearch.value || marketplaceBountyHaystack(bounty).includes(marketplaceSearch.value.toLowerCase()))
    .sort((a, b) => sortMarketplaceBounties(a, b, activeMarketplaceFilter.value))
    .slice(0, 8),
);
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
const marketplaceAgentQueueRows = computed(() =>
  (marketplaceData.value.agents || [])
    .filter((agent) => Number(agent.task_count) > 0 || Number(agent.open_task_count) > 0)
    .map(mapMarketplaceAgentQueue),
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
const dashboardSearchPlaceholder = computed(() =>
  dashboardSection.value === 'admin'
    ? 'Search admin queues, users, treasury, or payouts...'
    : dashboardSection.value === 'payments'
    ? 'Search payments, refs, methods, or statuses...'
    : dashboardSection.value === 'worker'
      ? 'Search claimed tasks, rewards, or proposal matches...'
      : 'Search your live projects...',
);
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
const dashboardRemainingCents = computed(() => Math.max(0, Number(dashboardEscrow.value?.remaining_cents ?? ((Number(dashboardSelectedProject.value?.work_pool_cents) || 0) - dashboardSpentCents.value)) || 0));
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
const dashboardEscrowView = computed(() => {
  const project = dashboardSelectedProject.value;
  const escrow = dashboardEscrow.value;
  const workPool = Number(escrow?.work_pool_cents ?? project?.work_pool_cents) || 0;
  const reserve = Number(escrow?.project_reserve_cents ?? project?.work_pool_cents) || 0;
  const released = Number(escrow?.released_cents ?? (dashboardLedgerPayoutCents.value || dashboardSpentCents.value)) || 0;
  const remaining = Math.max(0, Number(escrow?.remaining_cents ?? (workPool - released)) || 0);
  const overdrawn = Math.max(0, Number(escrow?.overdrawn_cents) || 0);
  const unallocated = Math.max(0, Number(escrow?.unallocated_cents) || 0);
  return {
    status: toTitleLabel(escrow?.release_status || (dashboardEscrowLoading.value ? 'syncing' : project ? 'funded' : 'empty')),
    budget: formatMRGFromCents(escrow?.budget_cents ?? project?.budget_cents),
    fee: formatMRGFromCents(escrow?.fee_cents ?? project?.fee_cents),
    workPool: formatMRGFromCents(workPool),
    reserve: formatMRGFromCents(reserve),
    taskReserve: formatMRGFromCents(escrow?.task_reserve_cents ?? workPool),
    released: formatMRGFromCents(released),
    remaining: formatMRGFromCents(remaining),
    overdrawn: formatMRGFromCents(overdrawn),
    unallocated: formatMRGFromCents(unallocated),
    paidTasks: Number(escrow?.paid_task_count) || dashboardAcceptedTasks.value.length,
    openTasks: Number(escrow?.open_task_count) || dashboardOpenTasks.value.length,
    hasOverdrawn: overdrawn > 0,
    hasUnallocated: unallocated > 0,
    updatedAt: escrow?.updated_at ? formatLedgerDateTime(escrow.updated_at).full : '-',
  };
});
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
const dashboardTaskGraphStats = computed(() => dashboardTaskGraph.value?.stats || {});
const dashboardTaskGraphView = computed(() => {
  if (!dashboardSelectedProject.value) {
    return {
      status: dashboardTaskGraphLoading.value ? 'Syncing' : 'Empty',
      body: 'Task graph appears after a project is selected.',
      progress: 0,
      ready: '0',
      blocked: '0',
      edges: '0',
    };
  }
  if (!dashboardTaskGraph.value) {
    return {
      status: dashboardTaskGraphLoading.value ? 'Syncing' : 'Waiting',
      body: dashboardTaskGraphLoading.value ? 'Loading task dependencies.' : 'No task graph payload loaded.',
      progress: 0,
      ready: '0',
      blocked: '0',
      edges: '0',
    };
  }
  const stats = dashboardTaskGraphStats.value;
  const ready = Number(stats.ready_count) || 0;
  const blocked = Number(stats.blocked_count) || 0;
  const edges = Number(stats.edge_count) || 0;
  const progress = Math.max(0, Math.min(100, Number(dashboardTaskGraph.value.progress) || 0));
  return {
    status: toTitleLabel(dashboardTaskGraph.value.status || 'planning'),
    body: `${formatCompactNumber(ready)} ready / ${formatCompactNumber(blocked)} blocked / ${formatCompactNumber(edges)} dependencies`,
    progress,
    ready: formatCompactNumber(ready),
    blocked: formatCompactNumber(blocked),
    edges: formatCompactNumber(edges),
  };
});
const dashboardTaskGraphRows = computed(() =>
  (dashboardTaskGraph.value?.nodes || [])
    .slice()
    .sort((a, b) => taskGraphNodeSortWeight(b) - taskGraphNodeSortWeight(a))
    .slice(0, 4)
    .map(mapDashboardTaskGraphNode),
);
const dashboardDeploymentView = computed(() => {
  const deployment = dashboardDeployment.value;
  if (!dashboardSelectedProject.value) {
    return {
      status: dashboardDeploymentLoading.value ? 'Syncing' : 'Empty',
      progress: 0,
      body: 'Deployment gates appear after a project is selected.',
      updatedAt: '-',
    };
  }
  if (!deployment) {
    return {
      status: dashboardDeploymentLoading.value ? 'Syncing' : 'Waiting',
      progress: 0,
      body: dashboardDeploymentLoading.value ? 'Fetching deployment validation gates.' : 'No deployment validation payload loaded.',
      updatedAt: '-',
    };
  }
  const when = formatLedgerDateTime(deployment.updated_at);
  return {
    status: toTitleLabel(deployment.status || 'queued'),
    progress: Math.max(0, Math.min(100, Number(deployment.progress) || 0)),
    body: `${deployment.project_title || dashboardSelectedProject.value.title || 'Project'} release gate from backend validation.`,
    updatedAt: when.full,
  };
});
const dashboardDeploymentStages = computed(() =>
  (dashboardDeployment.value?.stages || []).map(mapDashboardDeploymentStage),
);
const dashboardDeploymentSignals = computed(() =>
  (dashboardDeployment.value?.signals || []).slice(0, 3).map(mapDashboardDeploymentSignal),
);
const dashboardAIWorkflowView = computed(() => {
  const workflow = dashboardAIWorkflow.value;
  if (!dashboardSelectedProject.value) {
    return {
      status: dashboardAIWorkflowLoading.value ? 'Syncing' : 'Empty',
      progress: 0,
      body: 'AI workflow appears after a project is selected.',
    };
  }
  if (!workflow) {
    return {
      status: dashboardAIWorkflowLoading.value ? 'Syncing' : 'Waiting',
      progress: 0,
      body: dashboardAIWorkflowLoading.value ? 'Fetching orchestration stages.' : 'No orchestration payload loaded.',
    };
  }
  return {
    status: toTitleLabel(workflow.status || 'queued'),
    progress: Math.max(0, Math.min(100, Number(workflow.progress) || 0)),
    body: `${Number(workflow.ai_action_count) || 0} AI actions - ${Number(workflow.agent_task_count) || 0} agent tasks`,
  };
});
const dashboardAIWorkflowStages = computed(() =>
  (dashboardAIWorkflow.value?.stages || []).map(mapDashboardAIWorkflowStage),
);
const dashboardPullRequestStats = computed(() => dashboardPullRequests.value?.stats || {});
const dashboardPullRequestSummary = computed(() => {
  if (!dashboardSelectedProject.value) {
    return {
      status: dashboardPullRequestsLoading.value ? 'Syncing' : 'Empty',
      body: 'PR monitor appears after a project is selected.',
      open: '0',
      ready: '0',
      blocked: '0',
    };
  }
  if (!dashboardPullRequests.value) {
    return {
      status: dashboardPullRequestsLoading.value ? 'Syncing' : 'Waiting',
      body: dashboardPullRequestsLoading.value ? 'Fetching linked pull requests.' : 'No pull request monitor payload loaded.',
      open: '0',
      ready: '0',
      blocked: '0',
    };
  }
  const stats = dashboardPullRequestStats.value;
  return {
    status: `${formatCompactNumber(stats.pull_request_count)} PRs`,
    body: `${formatCompactNumber(stats.linked_task_count)} linked tasks / ${formatCompactNumber(stats.error_count)} sync errors`,
    open: formatCompactNumber(stats.open_pull_request_count),
    ready: formatCompactNumber(stats.ready_count),
    blocked: formatCompactNumber(stats.blocked_count),
  };
});
const dashboardPullRequestRows = computed(() => {
  const rows = [];
  for (const task of dashboardPullRequests.value?.tasks || []) {
    for (const pull of task.pull_requests || []) {
      rows.push(mapDashboardPullRequest(task, pull));
    }
  }
  return rows.sort((a, b) => new Date(b.updatedAtRaw || 0) - new Date(a.updatedAtRaw || 0)).slice(0, 5);
});
const dashboardRepositoryScanStats = computed(() => dashboardRepositoryScan.value?.stats || {});
const dashboardRepositoryScanView = computed(() => {
  if (!dashboardSelectedProject.value) {
    return {
      status: dashboardRepositoryScanLoading.value ? 'Scanning' : 'Empty',
      body: 'Repository scan appears after a project is selected.',
      scanned: '0',
      findings: '0',
      dependencies: '0',
      updatedAt: '-',
    };
  }
  if (!dashboardRepositoryScan.value) {
    return {
      status: dashboardRepositoryScanLoading.value ? 'Scanning' : 'Waiting',
      body: dashboardRepositoryScanLoading.value ? 'Scanning repository signals.' : 'No repository scan payload loaded.',
      scanned: '0',
      findings: '0',
      dependencies: '0',
      updatedAt: '-',
    };
  }
  const scan = dashboardRepositoryScan.value;
  const stats = dashboardRepositoryScanStats.value;
  const dependencyCount = Number(stats.dependency_files) || (Array.isArray(scan.dependencies) ? scan.dependencies.length : 0);
  const findingCount = Number(stats.finding_count) || (Array.isArray(scan.findings) ? scan.findings.length : 0);
  const scannedFiles = Number(stats.scanned_files) || 0;
  const when = formatLedgerDateTime(scan.updated_at);
  return {
    status: toTitleLabel(scan.status || 'scanned'),
    body: trimMarketplaceText(scan.summary, `${formatCompactNumber(scannedFiles)} files scanned / ${formatCompactNumber(findingCount)} findings`),
    scanned: formatCompactNumber(scannedFiles),
    findings: formatCompactNumber(findingCount),
    dependencies: formatCompactNumber(dependencyCount),
    updatedAt: when.full,
  };
});
const dashboardRepositoryScanFindings = computed(() =>
  (dashboardRepositoryScan.value?.findings || [])
    .slice()
    .sort((a, b) => repositoryFindingSeverityRank(b.severity) - repositoryFindingSeverityRank(a.severity))
    .slice(0, 4)
    .map(mapDashboardRepositoryFinding),
);
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
const dashboardNotificationRows = computed(() =>
  dashboardNotifications.value
    .slice()
    .sort((a, b) => new Date(b.created_at || 0) - new Date(a.created_at || 0))
    .slice(0, 8)
    .map(mapDashboardNotification),
);
const dashboardNotificationCount = computed(() => dashboardNotifications.value.filter((n) => !n.read_at).length);
const dashboardPaymentRows = computed(() => {
  const project = dashboardSelectedProject.value;
  if (!project) return [];
  const query = dashboardSearch.value.trim().toLowerCase();
  const rows = dashboardProjectLedger.value
    .filter((entry) => financialLedgerTypes.has(entry.type))
    .slice()
    .reverse()
    .map((entry) => mapDashboardPaymentRow(entry, project, dashboardSelectedTasks.value));
  if (!query) return rows;
  return rows.filter((row) => [
    row.type,
    row.title,
    row.body,
    row.method,
    row.status,
    row.counterparty,
    row.reference,
    row.rawReference,
  ].filter(Boolean).join(' ').toLowerCase().includes(query));
});
const dashboardPaymentView = computed(() => {
  const project = dashboardSelectedProject.value;
  if (!project) {
    return {
      title: dashboardLoading.value ? 'Loading payment history' : 'Payment history',
      body: dashboardLoading.value
        ? 'Fetching funding, escrow, and payout rows from the authenticated ledger.'
        : 'Select a funded project to inspect its payment flow after login.',
      status: dashboardLoading.value ? 'Syncing' : 'Empty',
    };
  }
  return {
    title: project.title || 'Payment history',
    body: `Track verified funding, escrow holds, token mints, platform fees, and payouts for ${project.title || 'this project'}.`,
    status: dashboardPaymentRows.value.length ? 'Live' : 'Waiting',
  };
});
const dashboardPaymentSummary = computed(() => {
  const project = dashboardSelectedProject.value;
  const projectMethod = project?.payment_method ? toTitleLabel(project.payment_method) : 'Not set';
  const projectProvider = paymentProviderLabel(project?.payment_provider || '');
  const projectStatus = project?.payment_status ? toTitleLabel(project.payment_status) : 'No status';
  const latestPayment = dashboardPaymentRows.value[0];
  return [
    {
      label: 'Verified Funding',
      value: formatMRGFromCents(dashboardLedgerFundingCents.value),
      caption: dashboardPaymentRows.value.filter((row) => row.type === 'Payment Verified').length ? 'Escrow-ready funding logs' : 'No verified funding yet',
    },
    {
      label: 'Released Payouts',
      value: dashboardEscrowView.value.released,
      caption: dashboardPaymentRows.value.filter((row) => row.type === 'Payout Released').length ? `${dashboardEscrowView.value.remaining} still reserved` : 'No payouts released yet',
    },
    {
      label: 'Method & Provider',
      value: `${projectMethod} / ${projectProvider}`,
      caption: projectStatus,
    },
    {
      label: 'Latest Activity',
      value: latestPayment ? latestPayment.amount : `0 ${tokenSymbol.value}`,
      caption: latestPayment ? latestPayment.when : 'No financial events yet',
    },
  ];
});
const workerDashboardProfile = computed(() => workerDashboard.value.profile || {});
const workerDashboardStats = computed(() => workerDashboard.value.stats || {});
const workerReputationScore = computed(() => Math.max(0, Math.min(100, Number(workerDashboardStats.value.reputation_score) || 0)));
const workerDashboardView = computed(() => {
  const profile = workerDashboardProfile.value;
  const display = profile.github_username ? `github:${profile.github_username}` : (profile.name || profile.email || 'Worker dashboard');
  return {
    title: display,
    body: 'Track claimed tasks, rewards, reputation, and proposal-ready bounty matches from the worker side of MergeOS.',
    initials: initialsFor(profile.github_username || profile.name || profile.email || 'WK'),
    status: workerDashboardLoading.value ? 'Syncing' : 'Worker',
  };
});
const workerDashboardMetrics = computed(() => [
  {
    label: 'Claimed tasks',
    value: String(Number(workerDashboardStats.value.claimed_task_count) || 0),
    caption: 'Accepted and paid work',
  },
  {
    label: 'Rewards',
    value: formatMRGFromCents(workerDashboardStats.value.reward_cents),
    caption: 'MRG earned',
  },
  {
    label: 'Reputation',
    value: `${workerReputationScore.value}`,
    caption: 'Identity and payout score',
  },
  {
    label: 'Proposals',
    value: String(Number(workerDashboardStats.value.open_proposal_count) || 0),
    caption: 'Open bounty matches',
  },
]);
const workerClaimedTaskRows = computed(() =>
  (workerDashboard.value.claimed_tasks || []).map(mapWorkerClaimedTask),
);
const workerRewardRows = computed(() =>
  (workerDashboard.value.rewards || []).map(mapWorkerReward),
);
const workerReputationRows = computed(() =>
  (workerDashboard.value.reputation || []).map((row) => ({
    label: row.label || 'Signal',
    value: row.value || '-',
    tone: row.tone || 'medium',
  })),
);
const workerProposalRows = computed(() =>
  (workerDashboard.value.proposals || []).map(mapWorkerProposal),
);
const workerIdentityRows = computed(() => workerDashboard.value.identity_status || []);
const workerIdentityReadyCount = computed(() => workerIdentityRows.value.filter((row) => row.ready).length);
const workerScoreRingStyle = computed(() => ({
  background: `conic-gradient(var(--green) 0 ${workerReputationScore.value}%, #e8eef1 ${workerReputationScore.value}% 100%)`,
}));
const isAdminUser = computed(() => user.value?.role === 'admin');
const adminSummaryView = computed(() => {
  const summary = adminSummary.value || {};
  return {
    status: adminConsoleLoading.value ? 'Syncing' : isAdminUser.value ? 'Admin' : 'Restricted',
    totalBudget: formatMRGFromCents(summary.total_budget_cents),
    workPool: formatMRGFromCents(summary.work_pool_cents),
    platformFee: formatMRGFromCents(summary.platform_fee_cents),
    paidTasks: formatMRGFromCents(summary.paid_task_cents),
    users: formatCompactNumber(summary.user_count),
    admins: formatCompactNumber(summary.admin_count),
    projects: formatCompactNumber(summary.project_count),
    openTasks: formatCompactNumber(summary.open_task_count),
    acceptedTasks: formatCompactNumber(summary.accepted_task_count),
    paymentMode: toTitleLabel(summary.payment_mode || 'payment'),
    repoProvider: toTitleLabel(summary.repo_provider || 'repo'),
    payPalReady: Boolean(summary.paypal_ready),
    cryptoReady: Boolean(summary.crypto_ready),
    githubReady: Boolean(summary.github_ready),
    smtpReady: Boolean(summary.smtp_ready),
  };
});
const adminOpsStats = computed(() => {
  const stats = adminOpsQueue.value?.stats || {};
  return [
    { label: 'Disputes', value: formatCompactNumber(stats.dispute_count), tone: 'amber' },
    { label: 'Payout Reviews', value: formatCompactNumber(stats.payout_review_count), tone: 'blue' },
    { label: 'Moderation', value: formatCompactNumber(stats.moderation_count), tone: 'purple' },
    { label: 'Fraud', value: formatCompactNumber(stats.fraud_count), tone: 'red' },
    { label: 'Security', value: formatCompactNumber(stats.security_count), tone: 'green' },
    { label: 'Critical', value: formatCompactNumber(stats.critical_count), tone: 'red' },
  ];
});
const adminOpsRows = computed(() =>
  (adminOpsQueue.value?.items || []).slice(0, 8).map(mapAdminOpsItem),
);
const adminReputationRows = computed(() =>
  (adminReputation.value?.workers || []).slice(0, 6).map(mapAdminReputationWorker),
);
const adminReputationStats = computed(() => adminReputation.value?.stats || {});
const adminUserRows = computed(() =>
  (adminUsers.value || []).slice(0, 8).map(mapAdminUserRow),
);
const adminSSLRows = computed(() =>
  (adminSSLReviews.value || []).map(mapAdminSSLReview),
);
const adminSSLStats = computed(() => {
  const rows = adminSSLRows.value;
  return {
    total: rows.length,
    ready: rows.filter((row) => row.tone === 'green').length,
    attention: rows.filter((row) => row.tone !== 'green').length,
    label: rows.length ? `${rows.filter((row) => row.tone !== 'green').length} need review` : 'No domains',
  };
});
const adminLLMProviderOptions = computed(() => {
  const options = adminSettings.value?.llm_provider_options;
  if (Array.isArray(options) && options.length) return options;
  return [{ id: 'gemini', label: 'Gemini', models: ['gemini-2.5-flash'] }];
});
const adminLLMSelectedProvider = computed(() =>
  adminLLMProviderOptions.value.find((option) => option.id === adminLLMForm.provider) || adminLLMProviderOptions.value[0],
);
const adminLLMModelOptions = computed(() => {
  const models = adminLLMSelectedProvider.value?.models;
  return Array.isArray(models) && models.length ? models : [adminLLMForm.model].filter(Boolean);
});
const adminLLMKeyRows = computed(() =>
  (adminLLMKeys.value || []).map(mapAdminLLMKey),
);
const adminLLMWebhookRows = computed(() =>
  (adminLLMWebhooks.value || []).slice(0, 5).map(mapAdminLLMWebhook),
);
const adminLLMStats = computed(() => {
  const rows = adminLLMKeyRows.value;
  return {
    total: rows.length,
    active: rows.filter((row) => row.statusValue === 'active').length,
    errors: rows.filter((row) => row.tone === 'red').length,
    label: `${rows.filter((row) => row.statusValue === 'active').length}/${rows.length} active`,
  };
});
const adminLLMKeyReady = computed(() =>
  adminLLMForm.provider && adminLLMForm.model && adminLLMForm.apiKey.trim().length >= 8,
);
const adminTaskReviewRows = computed(() =>
  (adminTasks.value || [])
    .filter((task) => task?.issue_url || task?.issue_number)
    .sort((a, b) => {
      const aOpen = String(a?.status || '').toLowerCase() === 'open' ? 0 : 1;
      const bOpen = String(b?.status || '').toLowerCase() === 'open' ? 0 : 1;
      return aOpen - bOpen || Number(b?.issue_number || 0) - Number(a?.issue_number || 0);
    })
    .slice(0, 8)
    .map(mapAdminTaskReviewRow),
);
const adminLoadedPullGroups = computed(() =>
  Object.values(adminTaskPulls.value || {})
    .filter((group) => group?.task_id)
    .map(mapAdminTaskPullGroup)
    .filter((group) => group.pullRequests.length),
);
const adminMergeReady = computed(() =>
  Number(adminMergeForm.rewardMRG) > 0 && Boolean(adminMergeForm.bountyType),
);
const adminMergeResultView = computed(() => {
  const result = adminMergeResult.value;
  if (!result) return null;
  return {
    title: result.pull_request?.title || `PR #${result.pull_request?.number || ''}`,
    workerID: result.worker_id || '',
    reward: `${formatCompactNumber(result.reward_mrg)} ${tokenSymbol.value}`,
    bountyType: toTitleLabel(result.bounty_type || ''),
    creditURL: result.credit_url || '',
    commentURL: result.comment_url || '',
    commentError: result.comment_error || '',
  };
});
const adminTestSettingsRows = computed(() =>
  (adminTestSettingsEntries.value || []).slice(0, 5).map(mapAdminTestSettingsEntry),
);
const adminTestSettingsStatus = computed(() =>
  adminTestSettings.value?.test_mode_enabled ? 'Enabled' : 'Disabled',
);
const adminCreditReady = computed(() =>
  Boolean(adminCreditForm.workerID.trim())
  && Number(adminCreditForm.rewardMRG) > 0
  && (Boolean(adminCreditForm.prURL.trim()) || Boolean(adminCreditForm.reference.trim())),
);
const adminCreditResultView = computed(() => {
  const result = adminCreditResult.value;
  if (!result) return null;
  return {
    workerID: result.worker_id || adminCreditForm.workerID,
    reward: `${formatCompactNumber(result.reward_mrg)} ${tokenSymbol.value}`,
    bountyType: toTitleLabel(result.bounty_type || adminCreditForm.bountyType),
    reference: result.ledger_entry?.reference || '',
    creditURL: result.credit_url || '',
  };
});
const dashboardSectionEyebrow = computed(() => {
  if (dashboardSection.value === 'admin') return 'ADMIN OPS';
  if (dashboardSection.value === 'payments') return 'PAYMENTS';
  if (dashboardSection.value === 'worker') return 'WORKER OPS';
  return 'PROJECT OPS';
});
const dashboardCommandTitle = computed(() =>
  dashboardSection.value === 'admin'
    ? 'Admin treasury and moderation console'
    : dashboardSection.value === 'payments'
    ? 'Payments, escrow, and payout proof'
    : dashboardSection.value === 'worker'
      ? 'Worker rewards and proposal console'
      : 'Project delivery command center',
);
const dashboardCommandBody = computed(() => {
  if (dashboardSection.value === 'admin') {
    return 'Review treasury totals, payout risks, disputes, moderation, security, and worker reputation from the admin APIs.';
  }
  if (dashboardSection.value === 'payments') {
    return 'Track funding, escrow holds, token minting, fees, and task payouts for the selected project.';
  }
  if (dashboardSection.value === 'worker') {
    return 'See paid tasks, MRG rewards, identity readiness, reputation signals, and open bounty matches.';
  }
  return 'Watch scope, tasks, budget, repo context, notifications, and ledger activity from one workspace.';
});
const dashboardCommandStats = computed(() => {
  if (dashboardSection.value === 'admin') {
    return [
      { label: 'Treasury budget', value: adminSummaryView.value.totalBudget, icon: CircleDollarSign, tone: 'green' },
      { label: 'Ops queue', value: formatCompactNumber(adminOpsQueue.value?.stats?.total_count), icon: ShieldCheck, tone: 'blue' },
      { label: 'LLM keys', value: formatCompactNumber(adminLLMStats.value.active), icon: Bot, tone: 'purple' },
      { label: 'SSL review', value: formatCompactNumber(adminSSLStats.value.attention), icon: Lock, tone: 'amber' },
    ];
  }
  if (dashboardSection.value === 'worker') {
    return [
      { label: 'Claimed', value: String(Number(workerDashboardStats.value.claimed_task_count) || 0), icon: GitPullRequest, tone: 'green' },
      { label: 'Rewards', value: formatMRGFromCents(workerDashboardStats.value.reward_cents), icon: CircleDollarSign, tone: 'blue' },
      { label: 'Reputation', value: `${workerReputationScore.value}`, icon: Trophy, tone: 'purple' },
      { label: 'Proposals', value: String(Number(workerDashboardStats.value.open_proposal_count) || 0), icon: Compass, tone: 'amber' },
    ];
  }
  return [
    {
      label: 'Active projects',
      value: String(dashboardProjectList.value.length),
      icon: FolderKanban,
      tone: 'green',
    },
    {
      label: 'Open tasks',
      value: String(dashboardOpenTasks.value.length),
      icon: ListTodo,
      tone: 'blue',
    },
    {
      label: 'Verified funding',
      value: formatMRGFromCents(dashboardLedgerFundingCents.value),
      icon: ShieldCheck,
      tone: 'purple',
    },
    {
      label: 'Unread notices',
      value: String(dashboardNotificationCount.value),
      icon: Bell,
      tone: 'amber',
    },
  ];
});

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

const sidebarSections = computed(() => {
  const mainItems = [
    { label: 'Overview', icon: LayoutDashboard, section: 'projects' },
    { label: 'My Projects', icon: FolderKanban, section: 'projects' },
    { label: 'Tasks', icon: ListTodo, section: 'projects', toast: 'Opening tasks...' },
    { label: 'Worker Dashboard', icon: User, section: 'worker' },
    { label: 'Repositories', icon: GitBranch, action: 'issue-scanner' },
    { label: 'Payments', icon: CreditCard, section: 'payments' },
    { label: 'Notifications', icon: Bell, section: 'notifications' },
  ];
  if (isAdminUser.value) {
    mainItems.splice(1, 0, { label: 'Admin Console', icon: ShieldCheck, section: 'admin' });
  }
  return [
    { label: 'Main', items: mainItems },
    {
      label: 'Discover',
      items: [
        { label: 'Talent Marketplace', icon: UsersRound, page: 'marketplace' },
        { label: 'Bounty Explorer', icon: Compass, marketplaceSection: 'marketplace-bounties' },
        { label: 'AI Agents', icon: Bot, marketplaceSection: 'marketplace-agents' },
      ],
    },
    {
      label: 'Tools',
      items: [
        { label: 'Repo Import', icon: UploadCloud, action: 'repo-import' },
        { label: 'AI Issue Scanner', icon: Search, action: 'issue-scanner' },
        { label: 'Estimate Cost', icon: Calculator, action: 'cost-estimator' },
      ],
    },
  ];
});

const topNavItems = computed(() => {
  const items = [
    { label: 'Dashboard', section: 'projects' },
    { label: 'Projects', section: 'projects' },
    { label: 'Worker', section: 'worker' },
    { label: 'Marketplace', page: 'marketplace' },
    { label: 'Repos', action: 'issue-scanner' },
    { label: 'Payments', section: 'payments' },
    { label: 'Analytics', action: 'project-analytics' },
  ];
  if (isAdminUser.value) {
    items.splice(2, 0, { label: 'Admin', section: 'admin' });
  }
  return items;
});

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

function openExternalURL(url = '') {
  const target = String(url || '').trim();
  if (!target || !hasWindow) return;
  if (!/^https?:\/\//i.test(target)) return;
  window.open(target, '_blank', 'noopener,noreferrer');
}

function scanBaseURL() {
  const domain = String(runtimeConfig.value?.scan_domain || 'scan.mergeos.shop')
    .trim()
    .replace(/^https?:\/\//i, '')
    .replace(/\/+$/, '');
  return `https://${domain || 'scan.mergeos.shop'}`;
}

function openWalletOnScan(address = '') {
  const wallet = String(address || '').trim();
  if (!wallet || !hasWindow) return;
  openExternalURL(`${scanBaseURL()}/address/${encodeURIComponent(wallet)}`);
}

async function copyDashboardProjectLink() {
  const project = dashboardSelectedProject.value;
  const projectID = String(project?.id || selectedDashboardProjectID.value || '').trim();
  const projectKey = String(project?.title || projectID).trim();
  const sharePath = projectKey ? `/ledger?project=${encodeURIComponent(projectKey)}` : '/ledger';
  const shareURL = hasWindow ? `${window.location.origin}${sharePath}` : sharePath;
  if (hasWindow && navigator.clipboard?.writeText) {
    try {
      await navigator.clipboard.writeText(shareURL);
      showToast('Project ledger link copied.');
      return;
    } catch {
      // Fall through to a visible fallback.
    }
  }
  showToast(`Project ledger link: ${shareURL}`);
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
  pushPublicNotification(message);
  if (!hasWindow) return;
  if (toastTimer) window.clearTimeout(toastTimer);
  toastTimer = window.setTimeout(() => {
    toastMessage.value = '';
  }, 2200);
}

async function copyClaimCommand(command = '') {
  const value = String(command || '').trim();
  if (!value) return;
  if (hasWindow && navigator.clipboard?.writeText) {
    try {
      await navigator.clipboard.writeText(value);
      showToast(`Claim command copied: ${value}`);
      return;
    } catch {
      // Fall through to visible fallback below.
    }
  }
  showToast(`Claim command: ${value}`);
}

function resetLedgerFilters() {
  activeLedgerTab.value = 'All Activity';
  activeLedgerProjectFilter.value = 'All Projects';
}

function applyLedgerProjectQueryFilter() {
  if (!hasWindow) return;
  const requestedProject = new URLSearchParams(window.location.search).get('project');
  const normalized = String(requestedProject || '').trim().toLowerCase();
  if (!normalized) return;
  const match = ledgerEvents.value.find((event) =>
    event.project.toLowerCase() === normalized || event.projectID.toLowerCase() === normalized,
  );
  if (match) {
    activeLedgerProjectFilter.value = match.project;
  }
}

function resetMarketplaceFilters() {
  marketplaceSearch.value = '';
  activeMarketplaceCategory.value = 'All';
  activeMarketplaceFilter.value = 'Category';
}

function pushPublicNotification(message) {
  if (!message || (user.value && !publicModeVisible.value && !projectWizardVisible.value)) return;
  const createdAt = new Date().toISOString();
  const body = projectWizardVisible.value ? 'Project setup status changed.' : 'Public session status changed.';
  publicNotifications.value = [
    {
      id: `public-${Date.now()}`,
      subject: String(message),
      body,
      meta: formatLedgerDateTime(createdAt).full,
      tone: projectWizardVisible.value ? 'green' : 'blue',
      createdAt,
    },
    ...publicNotifications.value,
  ].slice(0, 6);
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
  if (page === 'test-settings') {
    void loadPublicTestSettingsStatus();
    if (publicTestSettingsAuthenticated.value) {
      void loadPublicTestSettingsEntries();
    }
    return;
  }
  if (page === 'live') {
    void loadLiveFeedData();
    void loadMarketplaceData({ silent: true });
    return;
  }
  if (page === 'marketplace' || page === 'home') {
    void loadMarketplaceData({ silent: true });
    void loadLedgerData({ silent: true });
    void loadLiveFeedData({ silent: true });
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
  dashboardSection.value = 'projects';
  if (user.value) {
    void loadDashboardData({ silent: true });
    void loadWorkerDashboardData({ silent: true });
    if (isAdminUser.value) {
      void loadAdminConsoleData({ silent: true });
    }
  }
  if (!hasWindow) return;
  window.requestAnimationFrame(() => {
    window.scrollTo({ top: 0, behavior: 'smooth' });
  });
}

async function selectDashboardProject(projectID) {
  if (!projectID) return;
  selectedDashboardProjectID.value = projectID;
  activeDashboardTab.value = 'Overview';
  void loadDashboardEscrowData(projectID, { silent: true });
  void loadDashboardDeploymentData(projectID, { silent: true });
  void loadDashboardAIWorkflowData(projectID, { silent: true });
  void loadDashboardTaskGraphData(projectID, { silent: true });
  void loadDashboardPullRequestsData(projectID, { silent: true });
  void loadDashboardRepositoryScanData(projectID, { silent: true });
  await nextTick();
  if (!hasWindow) return;
  dashboardProjectHeader.value?.scrollIntoView({ behavior: 'smooth', block: 'start' });
  dashboardProjectHeader.value?.focus({ preventScroll: true });
}

function focusDashboardPanel(panelRef, block = 'start') {
  void nextTick(() => {
    if (!hasWindow) return;
    window.requestAnimationFrame(() => {
      panelRef.value?.scrollIntoView({ behavior: 'smooth', block });
      panelRef.value?.focus?.({ preventScroll: true });
    });
  });
}

function openDashboardProjectTab(tab = 'Overview') {
  const nextTab = dashboardTabs.includes(tab) ? tab : 'Overview';
  activeDashboardTab.value = nextTab;
  if (nextTab === 'Overview') {
    focusDashboardPanel(dashboardOverviewPanel);
    return;
  }
  if (nextTab === 'Tasks') {
    focusDashboardPanel(dashboardTasksPanel);
    return;
  }
  if (nextTab === 'Activity') {
    focusDashboardPanel(dashboardActivityPanel);
    return;
  }
  if (nextTab === 'Ledger') {
    focusDashboardPanel(dashboardLedgerPanel);
    return;
  }
  if (nextTab === 'Files') {
    focusDashboardPanel(dashboardRepositoryScanCard, 'center');
    return;
  }
  focusDashboardPanel(dashboardProjectHeader);
}

async function openFundedProjectDashboard(tab = 'Overview') {
  const projectID = String(fundedProject.value?.id || selectedDashboardProjectID.value || '').trim();
  projectWizardVisible.value = false;
  publicModeVisible.value = false;
  dashboardSection.value = 'projects';
  if (projectID) {
    selectedDashboardProjectID.value = projectID;
  }
  if (user.value) {
    await loadDashboardData({ silent: true, selectProjectID: projectID });
  }
  openDashboardProjectTab(tab);
  showToast('Project dashboard opened.');
}

function fundedProjectShareURL(path = '/marketplace') {
  const project = fundedProject.value || dashboardSelectedProject.value || {};
  const projectKey = String(project.id || project.title || projectTitleLabel.value || '').trim();
  const query = projectKey ? `?project=${encodeURIComponent(projectKey)}` : '';
  return hasWindow ? `${window.location.origin}${path}${query}` : `${path}${query}`;
}

async function copyFundedProjectInviteLink() {
  const shareURL = fundedProjectShareURL('/marketplace');
  if (hasWindow && navigator.clipboard?.writeText) {
    try {
      await navigator.clipboard.writeText(shareURL);
      showToast('Project invite link copied.');
      return;
    } catch {
      // Use visible fallback below.
    }
  }
  showToast(`Project invite link: ${shareURL}`);
}

async function handlePostPaymentAction(item = {}) {
  if (item.action === 'dashboard') {
    await openFundedProjectDashboard(item.tab || 'Overview');
    return;
  }
  if (item.action === 'invite') {
    await copyFundedProjectInviteLink();
    openMarketplaceSection(item.section || 'marketplace-contributors');
    return;
  }
  if (item.action === 'marketplace') {
    activeMarketplaceCategory.value = 'All';
    marketplaceSearch.value = fundedProject.value?.title || projectTitleLabel.value;
    openMarketplaceSection(item.section || 'marketplace-projects');
    showToast('Marketplace opened for project boosting.');
    return;
  }
  showToast(item.label ? `${item.label} opened.` : 'Opening next step.');
}

function focusRepoImportField() {
  void nextTick(() => {
    if (!hasWindow) return;
    window.requestAnimationFrame(() => {
      repoImportInput.value?.scrollIntoView({ behavior: 'smooth', block: 'center' });
      repoImportInput.value?.focus({ preventScroll: true });
    });
  });
}

function focusDashboardRepositoryScan() {
  void nextTick(() => {
    if (!hasWindow) return;
    window.requestAnimationFrame(() => {
      dashboardRepositoryScanCard.value?.scrollIntoView({ behavior: 'smooth', block: 'center' });
      dashboardRepositoryScanCard.value?.focus({ preventScroll: true });
    });
  });
}

function openRepoImportTool() {
  openProjectWizard();
  projectSetupForm.projectType = 'Bug Fix';
  repoImportError.value = '';
  focusRepoImportField();
  showToast('Repo import opened. Paste a GitHub repository URL.');
}

async function openIssueScannerTool() {
  openDashboardSection('projects');
  if (!dashboardProjects.value.length && token.value) {
    await loadDashboardData({ silent: true });
  }

  const targetProjectID = selectedDashboardProjectID.value || dashboardSelectedProject.value?.id || dashboardSortedProjects.value[0]?.id || '';
  if (!targetProjectID) {
    showToast('Create or import a project before running repository scan.');
    return;
  }

  selectedDashboardProjectID.value = targetProjectID;
  void loadDashboardRepositoryScanData(targetProjectID);
  focusDashboardRepositoryScan();
  showToast('Repository scan opened.');
}

function openCostEstimatorTool() {
  publicModeVisible.value = true;
  projectWizardVisible.value = true;
  projectWizardStage.value = 'setup';
  projectWizardStep.value = 3;
  errorMessage.value = '';
  updateProjectWizardBrowserPath();
  scrollProjectFlowTop();
  showToast('Cost estimator opened. Set budget, deadline, and funding method.');
}

function handleDashboardNav(item) {
  if (item.page) {
    openPublicPage(item.page);
    return;
  }
  if (item.marketplaceSection) {
    openMarketplaceSection(item.marketplaceSection);
    return;
  }
  if (item.section) {
    openDashboardSection(item.section);
    return;
  }
  if (item.action === 'repo-import') {
    openRepoImportTool();
    return;
  }
  if (item.action === 'issue-scanner') {
    void openIssueScannerTool();
    return;
  }
  if (item.action === 'cost-estimator') {
    openCostEstimatorTool();
    return;
  }
  if (item.action === 'project-analytics') {
    openDashboardSection('projects');
    openDashboardProjectTab('Overview');
    return;
  }
  showToast(item.toast || `${item.label} opened.`);
}

function openDashboardSection(section) {
  publicModeVisible.value = false;
  if (section === 'payments' || section === 'projects' || section === 'worker' || section === 'admin') {
    if (section === 'admin' && !isAdminUser.value) {
      showToast('Admin role is required.');
      return;
    }
    dashboardSection.value = section;
    if (section === 'worker') {
      void loadWorkerDashboardData({ silent: true });
    }
    if (section === 'admin') {
      void loadAdminConsoleData({ silent: true });
    }
    if (!hasWindow) return;
    window.requestAnimationFrame(() => {
      window.scrollTo({ top: 0, behavior: 'smooth' });
    });
    return;
  }
  dashboardSection.value = 'projects';
  if (section === 'notifications') {
    void loadDashboardNotifications();
    if (!hasWindow) return;
    window.requestAnimationFrame(() => {
      dashboardNotificationCenter.value?.scrollIntoView({ behavior: 'smooth', block: 'start' });
      dashboardNotificationCenter.value?.focus({ preventScroll: true });
    });
  }
}

function isDashboardNavActive(item = {}) {
  if (item.section === 'admin') return dashboardSection.value === 'admin';
  if (item.section === 'payments') return dashboardSection.value === 'payments';
  if (item.section === 'projects') return dashboardSection.value === 'projects';
  if (item.section === 'worker') return dashboardSection.value === 'worker';
  return false;
}

function openMarketplaceSection(id) {
  publicModeVisible.value = true;
  projectWizardVisible.value = false;
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

function projectSetupIsEmpty() {
  return [
    projectSetupForm.title,
    projectSetupForm.shortDescription,
    projectSetupForm.projectType,
    projectSetupForm.techStack,
    projectSetupForm.repoUrl,
    projectSetupForm.overview,
    projectSetupForm.requirements,
    projectSetupForm.budgetAmount,
  ].every((value) => !String(value || '').trim())
    && projectDeliverables.value.every((item) => !String(item || '').trim())
    && projectAttachments.value.length === 0;
}

function projectDraftPayload() {
  return {
    saved_at: new Date().toISOString(),
    form: { ...projectSetupForm },
    deliverables: projectDeliverables.value.slice(),
    attachments: projectAttachments.value.slice(),
    repo_import_result: repoImportResult.value,
    funding_amount: projectFundingAmount.value,
    payment_method: projectPaymentMethod.value,
  };
}

function applyProjectDraft(draft = {}) {
  const form = draft.form || {};
  for (const key of Object.keys(projectSetupForm)) {
    if (Object.prototype.hasOwnProperty.call(form, key)) {
      projectSetupForm[key] = form[key];
    }
  }
  const deliverables = Array.isArray(draft.deliverables) ? draft.deliverables : [];
  projectDeliverables.value = deliverables.length ? deliverables : [''];
  projectAttachments.value = Array.isArray(draft.attachments) ? draft.attachments.filter((file) => file?.id) : [];
  repoImportResult.value = draft.repo_import_result || null;
  projectFundingAmount.value = draft.funding_amount || projectFundingAmount.value;
  projectPaymentMethod.value = draft.payment_method || projectPaymentMethod.value;
}

function restoreProjectDraftIfEmpty() {
  if (!browserStorage || !projectSetupIsEmpty()) return;
  try {
    const draft = JSON.parse(browserStorage.getItem(projectDraftStorageKey) || 'null');
    if (draft?.form) {
      applyProjectDraft(draft);
      showToast('Project draft restored.');
    }
  } catch {
    browserStorage.removeItem(projectDraftStorageKey);
  }
}

function saveProjectDraft() {
  try {
    browserStorage?.setItem(projectDraftStorageKey, JSON.stringify(projectDraftPayload()));
    showToast('Project draft saved locally.');
  } catch {
    showToast('Could not save project draft locally.');
  }
}

function generateScopeSuggestions() {
  const importedIssues = repoImportedIssues.value.slice(0, 5);
  const projectName = projectSetupForm.title.trim() || 'this project';
  const baseGoal = projectSetupForm.shortDescription.trim()
    || projectSetupForm.overview.trim()
    || `Deliver ${projectName} with verified scope, tests, and handoff.`;

  if (!projectSetupForm.projectType && (projectSetupForm.repoUrl || importedIssues.length)) {
    projectSetupForm.projectType = 'Bug Fix';
  }
  if (!projectSetupForm.overview.trim()) {
    projectSetupForm.overview = importedIssues.length
      ? importedIssues.map((issue) => `#${issue.number} ${issue.title} - ${issue.complexity || 'review'} priority`).join('\n')
      : baseGoal;
  }
  if (!projectSetupForm.requirements.trim()) {
    projectSetupForm.requirements = [
      'Confirm acceptance criteria before implementation.',
      'Keep changes scoped to the requested repository or workflow.',
      'Add tests, screenshots, or ledger evidence for the changed path.',
      'Document any deployment, payment, or security assumptions.',
    ].join('\n');
  }
  const generatedDeliverables = importedIssues.length
    ? importedIssues.map((issue) => `Fix #${issue.number}: ${issue.title}`)
    : [
      `Define implementation scope for ${projectName}`,
      'Ship the core workflow with responsive UI and backend integration',
      'Verify behavior with tests or runtime evidence',
      'Provide handoff notes and public proof where applicable',
    ];
  if (!visibleDeliverables.value.length) {
    projectDeliverables.value = generatedDeliverables;
  }
  showToast('Scope suggestions generated.');
}

function openScopeAssistant() {
  if (projectWizardStep.value < 2) {
    goProjectStep(2);
  }
  generateScopeSuggestions();
}

function openProjectWizard(options = {}) {
  publicModeVisible.value = true;
  projectWizardVisible.value = true;
  projectWizardStage.value = 'setup';
  projectWizardStep.value = 1;
  errorMessage.value = '';
  restoreProjectDraftIfEmpty();
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
      projectSetupForm.projectType = 'Bug Fix';
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

function openAttachmentPicker() {
  attachmentInput.value?.click();
}

function projectAttachmentFilesFromEvent(event) {
  const source = event?.dataTransfer?.files || event?.target?.files;
  return source ? Array.from(source).filter((file) => file?.name) : [];
}

function resetAttachmentInput(event) {
  const target = event?.target;
  if (target && 'value' in target) {
    target.value = '';
  }
}

function normalizeAttachmentResponse(payload) {
  if (Array.isArray(payload)) return payload;
  if (Array.isArray(payload?.attachments)) return payload.attachments;
  return payload?.id ? [payload] : [];
}

function requireLoginForProjectAttachmentUpload() {
  authReturnToProjectWizard.value = true;
  projectWizardVisible.value = false;
  openAuth('login');
  showToast('Log in to upload attachments.');
}

async function uploadProjectAttachments(event) {
  const files = projectAttachmentFilesFromEvent(event);
  resetAttachmentInput(event);
  if (!files.length || attachmentUploadBusy.value) return;

  attachmentUploadError.value = '';
  if (!user.value || !token.value) {
    requireLoginForProjectAttachmentUpload();
    return;
  }

  const form = new FormData();
  files.forEach((file) => form.append('files', file));
  attachmentUploadBusy.value = true;
  try {
    const response = await fetch('/api/uploads', {
      method: 'POST',
      headers: {
        Authorization: `Bearer ${token.value}`,
      },
      body: form,
    });
    const payload = await response.json().catch(() => ({}));
    if (!response.ok) {
      if (response.status === 401) {
        clearSession();
        requireLoginForProjectAttachmentUpload();
        return;
      }
      throw createRequestError(response, payload && !Array.isArray(payload) ? payload : {});
    }
    const uploaded = normalizeAttachmentResponse(payload).filter((file) => file?.id);
    if (!uploaded.length) {
      throw new Error('Upload finished but no attachment IDs were returned.');
    }
    const existingIDs = new Set(projectAttachments.value.map((file) => file.id));
    projectAttachments.value = [
      ...projectAttachments.value,
      ...uploaded.filter((file) => !existingIDs.has(file.id)),
    ];
    showToast(`${uploaded.length} file${uploaded.length === 1 ? '' : 's'} attached.`);
  } catch (error) {
    attachmentUploadError.value = error.message || 'Could not upload attachments.';
    showToast(attachmentUploadError.value);
  } finally {
    attachmentUploadBusy.value = false;
  }
}

function removeProjectAttachment(attachmentID) {
  projectAttachments.value = projectAttachments.value.filter((file) => file.id !== attachmentID);
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
    projectAttachments.value = [];
    browserStorage?.removeItem(projectDraftStorageKey);
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
    
    // Try LLM-powered evaluation first, with fallback to rule-based
    try {
      const response = await api('/api/projects/evaluate-llm', {
        method: 'POST',
        body: JSON.stringify(payload)
      });
      aiEvaluationResult.value = response;
    } catch (_llmErr) {
      // LLM endpoint unavailable — use rule-based evaluation
      const response = await api('/api/projects/evaluate', {
        method: 'POST',
        body: JSON.stringify(payload)
      });
      aiEvaluationResult.value = response;
    }
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

function formatFileSize(bytes = 0) {
  const size = Math.max(0, Number(bytes) || 0);
  if (size < 1024) return `${size} B`;
  if (size < 1024 * 1024) return `${(size / 1024).toFixed(size < 1024 * 10 ? 1 : 0)} KB`;
  return `${(size / (1024 * 1024)).toFixed(size < 1024 * 1024 * 10 ? 1 : 0)} MB`;
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

function marketplaceAgentCapabilities(type = '') {
  const text = String(type || '').toLowerCase();
  if (text.includes('design')) {
    return ['UI review', 'Brand kit', 'Responsive QA'];
  }
  if (text.includes('ledger') || text.includes('go') || text.includes('payment')) {
    return ['Payment checks', 'Ledger proof', 'Replay review'];
  }
  if (text.includes('devops') || text.includes('deploy')) {
    return ['Deploy verify', 'SSL review', 'Smoke tests'];
  }
  if (text.includes('qa') || text.includes('test')) {
    return ['Regression tests', 'Evidence review', 'A11y pass'];
  }
  if (text.includes('front')) {
    return ['UI generation', 'Component tests', 'PR review'];
  }
  return ['Repo scan', 'Task generation', 'PR review'];
}

function mapMarketplaceAgentQueue(agent = {}, index = 0) {
  const type = agent.type || `agent-${index}`;
  const openTasks = Number(agent.open_task_count) || 0;
  const totalTasks = Number(agent.task_count) || openTasks;
  const relatedTasks = (marketplaceData.value.bounties || [])
    .filter((bounty) => String(bounty.suggested_agent_type || '').toLowerCase() === String(type).toLowerCase())
    .slice(0, 3)
    .map((bounty, taskIndex) => mapMarketplaceBounty(bounty, taskIndex));
  return {
    type,
    icon: marketplaceAgentIcon(type),
    title: agent.title || toTitleLabel(type || 'AI Agent'),
    workerKind: toTitleLabel(agent.worker_kind || 'agent'),
    status: openTasks > 0 ? 'Accepting work' : 'Standing by',
    body: `${toTitleLabel(type)} can review, test, generate, and validate scoped work from funded repository tasks.`,
    openTasks: formatCompactNumber(openTasks),
    totalTasks: formatCompactNumber(totalTasks),
    budget: formatPublicMRGFromCents(agent.budget_cents),
    capabilities: marketplaceAgentCapabilities(type),
    nextTasks: relatedTasks,
    tone: ['green', 'blue', 'yellow', 'red'][index % 4],
  };
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
    openTasks,
    budgetCents: Number(project.budget_cents) || 0,
    createdTime: Date.parse(project.created_at || '') || 0,
    verified: project.status === 'funded',
    urgent: openTasks > 0,
    accent: palette.accent,
    soft: palette.soft,
  };
}

function mapMarketplaceBounty(bounty = {}, index = 0) {
  const workerKind = bounty.required_worker_kind || 'human';
  const agentType = bounty.suggested_agent_type || '';
  const issueNumber = Number(bounty.issue_number) || 0;
  const claimCommand = issueNumber > 0 ? `/attempt #${issueNumber}` : '';
  return {
    id: bounty.id || `${bounty.project_id || 'project'}:${issueNumber || index}`,
    icon: agentType ? marketplaceAgentIcon(agentType) : (workerKind === 'human' ? User : Bot),
    title: bounty.title || 'Open bounty',
    acceptance: trimMarketplaceText(bounty.acceptance, 'Acceptance criteria will appear after task generation.'),
    project: bounty.project_title || 'MergeOS project',
    reward: formatPublicMRGFromCents(bounty.reward_cents),
    rewardCents: Number(bounty.reward_cents) || 0,
    lane: agentType ? toTitleLabel(agentType) : toTitleLabel(workerKind),
    issue: issueNumber > 0 ? `#${issueNumber}` : 'Task',
    issueNumber,
    claimCommand,
    url: bounty.issue_url || '',
    tone: ['green', 'blue', 'purple', 'amber'][index % 4],
  };
}

function sortMarketplaceRows(rows = [], filter = 'Category') {
  const sorted = rows.slice();
  if (filter === 'Budget') {
    return sorted.sort((a, b) => b.budgetCents - a.budgetCents || b.openTasks - a.openTasks || b.createdTime - a.createdTime);
  }
  if (filter === 'Delivery time') {
    return sorted.sort((a, b) => b.openTasks - a.openTasks || b.createdTime - a.createdTime || b.budgetCents - a.budgetCents);
  }
  return sorted;
}

function sortMarketplaceBounties(a = {}, b = {}, filter = 'Category') {
  if (filter === 'Budget') {
    return b.rewardCents - a.rewardCents || b.issueNumber - a.issueNumber;
  }
  if (filter === 'Delivery time') {
    return b.issueNumber - a.issueNumber || b.rewardCents - a.rewardCents;
  }
  return 0;
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

function marketplaceBountyHaystack(bounty = {}) {
  return [
    bounty.title,
    bounty.acceptance,
    bounty.project,
    bounty.reward,
    bounty.lane,
    bounty.issue,
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

const financialLedgerTypes = new Set([
  'payment_verified',
  'platform_fee',
  'project_reserve',
  'task_reserve',
  'task_payment',
  'token_mint',
]);

function taskForLedgerEntry(entry = {}, tasks = []) {
  const haystack = [
    entry.reference,
    entry.from_account,
    entry.to_account,
  ].filter(Boolean).join('|');
  return tasks.find((task) => task.id && haystack.includes(task.id)) || null;
}

function paymentProviderLabel(value = '') {
  const normalized = String(value || '').trim().replace(/^payment:/, '');
  if (!normalized) return 'MergeOS';
  if (normalized === 'local-dev-verifier') return 'Local Verifier';
  return toTitleLabel(normalized.replaceAll('-', ' '));
}

function ledgerAccountLabel(value = '') {
  const text = String(value || '').trim();
  if (!text) return 'MergeOS';
  if (text.startsWith('payment:')) return `${paymentProviderLabel(text)} gateway`;
  if (text.startsWith('wallet:')) return shortWallet(text.slice('wallet:'.length));
  if (text.startsWith('github:')) return text;
  if (text.startsWith('task:')) return shortLedgerReference(text.slice('task:'.length));
  if (text.startsWith('project:')) return shortLedgerReference(text.slice('project:'.length));
  return shortLedgerReference(text);
}

function paymentStatusForEntry(entry = {}, project = {}) {
  if (entry.type === 'payment_verified') return project.payment_status ? toTitleLabel(project.payment_status) : 'Verified';
  if (entry.type === 'task_payment') return 'Released';
  if (entry.type === 'token_mint') return 'Minted';
  if (entry.type === 'platform_fee') return 'Charged';
  return 'Held';
}

function paymentMethodForEntry(entry = {}, project = {}, task = null) {
  if (entry.type === 'payment_verified') {
    return `${toTitleLabel(project.payment_method || 'payment')} / ${paymentProviderLabel(project.payment_provider || entry.from_account)}`;
  }
  if (entry.type === 'task_payment') {
    return task ? `Task payout / #${task.issue_number || '-'}` : 'Task payout / wallet';
  }
  if (entry.type === 'token_mint') return `${tokenSymbol.value} mint / ledger`;
  if (entry.type === 'platform_fee') return `Platform fee / ${paymentProviderLabel(project.payment_provider || entry.from_account)}`;
  if (entry.type === 'project_reserve') return 'Escrow reserve / project';
  return task ? `Task reserve / #${task.issue_number || '-'}` : 'Task reserve / delivery';
}

function paymentBodyForEntry(entry = {}, project = {}, task = null) {
  if (entry.type === 'payment_verified') {
    return `${project.title || 'This project'} was funded and verified for escrow-backed delivery.`;
  }
  if (entry.type === 'task_payment') {
    return task
      ? `Released for #${task.issue_number || '-'} ${task.title || 'project task'}.`
      : `Released to ${ledgerAccountLabel(entry.to_account)} from the task reserve.`;
  }
  if (entry.type === 'token_mint') return `${tokenSymbol.value} mint recorded for the payer after funding verification.`;
  if (entry.type === 'platform_fee') return 'Platform fee logged as part of the verified payment flow.';
  if (entry.type === 'project_reserve') return 'Escrow reserve created to protect the full project budget.';
  return task
    ? `Budget reserved for #${task.issue_number || '-'} ${task.title || 'project task'}.`
    : 'Budget reserved for task delivery.';
}

function mapDashboardPaymentRow(entry = {}, project = {}, tasks = []) {
  const meta = ledgerMetaFor(entry.type);
  const when = formatLedgerDateTime(entry.created_at);
  const task = taskForLedgerEntry(entry, tasks);
  const amount = formatMRGFromCents(entry.amount_cents);
  return {
    key: `${entry.sequence}-${entry.entry_hash || entry.reference}`,
    type: meta.type,
    tone: meta.tone,
    title: project.title || 'MergeOS payment history',
    body: paymentBodyForEntry(entry, project, task),
    method: paymentMethodForEntry(entry, project, task),
    status: paymentStatusForEntry(entry, project),
    counterparty: entry.type === 'task_payment' ? ledgerAccountLabel(entry.to_account) : paymentProviderLabel(project.payment_provider || entry.from_account),
    amount: meta.amountTone === 'negative' ? `- ${amount}` : meta.amountTone === 'positive' ? `+ ${amount}` : amount,
    amountClass: meta.amountTone,
    when: when.full,
    reference: shortLedgerReference(entry.reference || entry.entry_hash || `#${entry.sequence}`),
    rawReference: entry.reference || entry.entry_hash || `#${entry.sequence}`,
  };
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

function taskGraphNodeSortWeight(node = {}) {
  if (node.status === 'accepted') return 1;
  if (node.ready) return 3;
  if (Array.isArray(node.blocked_by) && node.blocked_by.length) return 4;
  return 2;
}

function taskGraphLaneTone(lane = '') {
  const normalized = String(lane || '').toLowerCase();
  if (normalized === 'deployment') return 'blue';
  if (normalized === 'validation') return 'green';
  if (normalized === 'design') return 'purple';
  if (normalized === 'backend') return 'amber';
  if (normalized === 'agent') return 'blue';
  return 'slate';
}

function mapDashboardTaskGraphNode(node = {}) {
  const blockedCount = Array.isArray(node.blocked_by) ? node.blocked_by.length : 0;
  const status = node.status === 'accepted'
    ? 'Complete'
    : node.ready
      ? 'Ready'
      : blockedCount
        ? 'Blocked'
        : toTitleLabel(node.status || 'open');
  return {
    id: node.id || node.task_id || node.title || 'task-node',
    issueNumber: node.issue_number || '-',
    title: node.title || 'Untitled task node',
    lane: toTitleLabel(node.lane || 'implementation'),
    status,
    reward: formatMRGFromCents(node.reward_cents),
    blockedBy: blockedCount ? `${blockedCount} blockers` : 'No blockers',
    tone: status === 'Blocked' ? 'red' : status === 'Ready' ? 'green' : status === 'Complete' ? 'blue' : taskGraphLaneTone(node.lane),
  };
}

function mapDashboardDeploymentStage(stage = {}) {
  const reference = stage.reference || (stage.source_task_issue_number ? `issue:${stage.source_task_issue_number}` : '');
  return {
    id: stage.id || stage.title || reference,
    title: stage.title || 'Deployment stage',
    body: trimMarketplaceText(stage.body, 'Deployment validation stage.'),
    status: toTitleLabel(stage.status || 'pending'),
    tone: stage.tone || (stage.status === 'complete' ? 'green' : stage.status === 'in_progress' ? 'blue' : 'amber'),
    reference: shortLedgerReference(reference),
  };
}

function mapDashboardDeploymentSignal(signal = {}) {
  return {
    id: signal.id || `${signal.type || 'signal'}-${signal.created_at || signal.reference || signal.title}`,
    title: signal.title || toTitleLabel(signal.type || 'Signal'),
    reference: shortLedgerReference(signal.reference || ''),
    status: toTitleLabel(signal.status || 'live'),
  };
}

function mapDashboardAIWorkflowStage(stage = {}) {
  return {
    id: stage.id || stage.title || stage.reference,
    title: stage.title || 'AI workflow stage',
    body: trimMarketplaceText(stage.body, 'AI orchestration stage.'),
    status: toTitleLabel(stage.status || 'pending'),
    tone: stage.tone || (stage.status === 'complete' ? 'green' : stage.status === 'in_progress' ? 'blue' : 'amber'),
  };
}

function mapDashboardPullRequest(task = {}, pull = {}) {
  const readiness = pull.readiness || {};
  const status = readiness.status || (pull.merged ? 'merged' : pull.state || 'open');
  const when = formatLedgerDateTime(pull.updated_at || pull.created_at);
  return {
    id: `${task.task_id || task.issue_number}-${pull.number}`,
    number: pull.number || '-',
    title: pull.title || 'Untitled pull request',
    task: task.issue_number ? `Issue #${task.issue_number}` : (task.title || 'Task'),
    author: pull.author ? `@${pull.author}` : 'Unknown author',
    url: pull.html_url || '',
    state: pull.merged ? 'Merged' : toTitleLabel(pull.state || 'open'),
    status: toTitleLabel(status),
    tone: dashboardPullReadinessTone(readiness, pull),
    risk: readiness.risk_level ? `${toTitleLabel(readiness.risk_level)} risk` : 'No risk score',
    updatedAt: when.full,
    updatedAtRaw: pull.updated_at || pull.created_at || '',
  };
}

function dashboardPullReadinessTone(readiness = {}, pull = {}) {
  if (readiness.status === 'blocked' || readiness.risk_level === 'high') return 'red';
  if (readiness.status === 'needs_review' || readiness.risk_level === 'medium') return 'amber';
  if (readiness.status === 'ready' || pull.merged) return 'green';
  return 'blue';
}

function repositoryFindingSeverityRank(severity = '') {
  const normalized = String(severity || '').toLowerCase();
  if (normalized === 'critical') return 5;
  if (normalized === 'high') return 4;
  if (normalized === 'medium') return 3;
  if (normalized === 'low') return 2;
  return 1;
}

function repositoryFindingTone(severity = '') {
  const normalized = String(severity || '').toLowerCase();
  if (normalized === 'critical' || normalized === 'high') return 'red';
  if (normalized === 'medium') return 'amber';
  if (normalized === 'low') return 'blue';
  return 'green';
}

function mapDashboardRepositoryFinding(finding = {}) {
  const path = finding.path || 'repository';
  const line = Number(finding.line) || 0;
  return {
    id: finding.id || `${path}-${line}-${finding.title || finding.category || finding.severity || 'finding'}`,
    title: finding.title || toTitleLabel(finding.category || 'Repository signal'),
    body: trimMarketplaceText(finding.body || finding.signal, 'Repository scan finding.'),
    severity: toTitleLabel(finding.severity || 'info'),
    category: toTitleLabel(finding.category || 'code'),
    pathLine: line > 0 ? `${path}:${line}` : path,
    tone: repositoryFindingTone(finding.severity),
  };
}

function mapWorkerClaimedTask(task = {}) {
  const when = formatLedgerDateTime(task.accepted_at);
  return {
    id: task.id || `${task.project_id}-${task.issue_number}`,
    initials: String(task.issue_number || 'W').padStart(2, '0').slice(-2),
    issueNumber: task.issue_number || '-',
    title: task.title || 'Accepted task',
    acceptance: trimMarketplaceText(task.acceptance, 'Acceptance criteria not provided.'),
    project: task.project_title || 'MergeOS project',
    reward: formatMRGFromCents(task.reward_cents),
    kind: task.agent_type ? toTitleLabel(task.agent_type) : toTitleLabel(task.worker_kind || 'worker'),
    when: when.date === '-' ? '-' : when.date,
  };
}

function mapWorkerReward(entry = {}) {
  const when = formatLedgerDateTime(entry.created_at);
  return {
    key: `${entry.sequence}-${entry.entry_hash || entry.reference}`,
    type: entry.type === 'manual_credit' ? 'Manual credit' : 'Task payout',
    amount: formatMRGFromCents(entry.amount_cents),
    ref: shortLedgerReference(entry.reference || entry.entry_hash || `#${entry.sequence}`),
    when: when.full,
  };
}

function mapWorkerProposal(proposal = {}) {
  const agentType = proposal.suggested_agent_type || '';
  const issueNumber = Number(proposal.issue_number) || 0;
  return {
    id: proposal.id || `${proposal.project_id}-${proposal.issue_number}`,
    title: proposal.title || 'Open bounty',
    project: proposal.project_title || 'MergeOS project',
    lane: agentType ? toTitleLabel(agentType) : toTitleLabel(proposal.required_worker_kind || 'worker'),
    reward: formatMRGFromCents(proposal.reward_cents),
    matchScore: Number(proposal.match_score) || 0,
    issue: issueNumber > 0 ? `#${issueNumber}` : 'Task',
    issueNumber,
    claimCommand: issueNumber > 0 ? `/attempt #${issueNumber}` : '',
    url: proposal.issue_url || '',
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

function mapDashboardNotification(note = {}) {
  const when = formatLedgerDateTime(note.created_at);
  return {
    id: note.id || `${note.subject}-${note.created_at}`,
    subject: note.subject || 'Notification',
    body: trimMarketplaceText(note.body, 'MergeOS status update.'),
    meta: `${toTitleLabel(note.channel || 'app')} • ${toTitleLabel(note.status || 'logged')} • ${when.full}`,
    tone: note.status === 'failed' ? 'red' : note.project_id ? 'green' : 'blue',
    isUnread: !note.read_at,
    rawId: note.id,
  };
}

function adminOpsTone(value = '') {
  const normalized = String(value || '').toLowerCase();
  if (normalized === 'critical' || normalized === 'high' || normalized.includes('fraud')) return 'red';
  if (normalized === 'medium' || normalized.includes('dispute')) return 'amber';
  if (normalized.includes('security') || normalized === 'low') return 'green';
  return 'blue';
}

function mapAdminOpsItem(item = {}) {
  const when = formatLedgerDateTime(item.created_at);
  const tone = adminOpsTone(item.severity || item.type);
  return {
    id: item.id || `${item.type || 'ops'}-${item.created_at || item.reference || item.title}`,
    type: toTitleLabel(item.type || 'ops'),
    severity: toTitleLabel(item.severity || 'review'),
    title: item.title || 'Admin review item',
    body: trimMarketplaceText(item.body, 'Admin operations item requires review.'),
    project: item.project_title || (item.project_id ? `Project ${String(item.project_id).slice(-6)}` : ''),
    reference: item.reference || (item.issue_number ? `Issue #${item.issue_number}` : ''),
    status: toTitleLabel(item.status || 'open'),
    url: item.url || '',
    tone,
    when: when.full,
  };
}

function mapAdminReputationWorker(worker = {}) {
  const name = worker.name || worker.worker_id || 'Worker';
  return {
    id: worker.worker_id || name,
    name,
    level: worker.level || worker.risk_level || 'Worker',
    risk: toTitleLabel(worker.risk_level || 'unknown'),
    score: Number(worker.score) || 0,
    completed: formatCompactNumber(worker.completed_task_count),
    rewards: formatMRGFromCents(worker.reward_cents),
    flags: Array.isArray(worker.flags) ? worker.flags : [],
    tone: adminOpsTone(worker.risk_level),
  };
}

function mapAdminUserRow(row = {}) {
  const workerID = row.github_username
    ? `github:${row.github_username}`
    : (row.wallet_address || row.id || '');
  const lastProject = row.last_project_at ? formatLedgerDateTime(row.last_project_at).date : '-';
  return {
    id: row.id || row.email || row.name,
    name: row.name || row.email || 'User',
    email: row.email || '',
    role: toTitleLabel(row.role || 'client'),
    company: row.company_name || 'MergeOS',
    wallet: row.wallet_address ? shortWallet(row.wallet_address) : 'No wallet',
    github: row.github_username ? `@${row.github_username}` : 'No GitHub',
    workerID,
    projects: formatCompactNumber(row.project_count),
    budget: formatMRGFromCents(row.total_budget_cents),
    lastProject,
    risk: row.worker_audit?.risk_level ? toTitleLabel(row.worker_audit.risk_level) : 'No audit',
    tone: row.role === 'admin' ? 'blue' : adminOpsTone(row.worker_audit?.risk_level || 'low'),
  };
}

function adminSSLTone(status = '') {
  const normalized = String(status || '').toLowerCase();
  if (normalized === 'expired' || normalized === 'error') return 'red';
  if (normalized === 'warning' || normalized === 'pending') return 'amber';
  return 'green';
}

function mapAdminSSLReview(review = {}) {
  const lastChecked = review.last_checked_at ? formatLedgerDateTime(review.last_checked_at).full : 'Not checked';
  const nextCheck = review.next_check_at ? formatLedgerDateTime(review.next_check_at).full : 'Not scheduled';
  const notAfter = review.not_after ? formatLedgerDateTime(review.not_after).date : '-';
  const dnsNames = Array.isArray(review.dns_names) ? review.dns_names : [];
  const status = String(review.status || 'pending').toLowerCase();
  const days = Number(review.days_remaining) || 0;
  return {
    id: `${review.domain || 'domain'}:${review.port || '443'}`,
    domain: review.domain || 'Unknown domain',
    port: review.port || '443',
    status: toTitleLabel(status),
    issuer: review.issuer || 'Unknown issuer',
    subject: review.subject || review.domain || 'Unknown subject',
    serialNumber: review.serial_number || '',
    dnsNames,
    dnsSummary: dnsNames.length ? dnsNames.slice(0, 3).join(', ') : 'No SANs recorded',
    expiry: notAfter,
    daysRemaining: status === 'pending' ? 'Pending' : `${days} days`,
    lastChecked,
    nextCheck,
    checkedBy: review.checked_by || 'system',
    error: review.error || '',
    tone: adminSSLTone(status),
  };
}

function adminLLMTone(status = '') {
  const normalized = String(status || '').toLowerCase();
  if (normalized === 'error') return 'red';
  if (normalized === 'quota_limited' || normalized === 'disabled') return 'amber';
  return 'green';
}

function mapAdminLLMKey(key = {}) {
  const lastUsed = key.last_used_at ? formatLedgerDateTime(key.last_used_at).full : 'Never used';
  const statusValue = key.status || 'active';
  const toggleStatus = statusValue === 'disabled' ? 'active' : 'disabled';
  const id = key.id || key.key_hint || key.provider || 'llm-key';
  return {
    id,
    provider: toTitleLabel(key.provider || 'gemini'),
    providerValue: key.provider || 'gemini',
    model: key.model || '-',
    keyHint: key.key_hint || 'hidden key',
    status: toTitleLabel(statusValue),
    statusValue,
    requestCount: formatCompactNumber(key.request_count),
    successCount: formatCompactNumber(key.success_count),
    quotaErrorCount: formatCompactNumber(key.quota_error_count),
    lastStatusCode: key.last_status_code || 0,
    lastError: key.last_error || '',
    lastUsed,
    testBusyID: `${id}:test`,
    resetBusyID: `${id}:reset`,
    toggleBusyID: `${id}:${toggleStatus}`,
    toggleStatus,
    toggleLabel: toggleStatus === 'active' ? 'Enable' : 'Disable',
    tone: adminLLMTone(key.status),
  };
}

function mapAdminLLMWebhook(log = {}) {
  const when = formatLedgerDateTime(log.received_at);
  const status = String(log.status || 'logged').toLowerCase();
  return {
    id: log.id || `${log.delivery_id || 'delivery'}-${log.received_at || log.pull_number || ''}`,
    title: log.pull_number ? `PR #${log.pull_number}` : (log.repository || 'AI review webhook'),
    body: `${log.event_name || 'event'} / ${log.action || 'received'} / ${log.sender || 'GitHub'}`,
    repository: log.repository || '-',
    status: toTitleLabel(status),
    statusValue: status,
    statusCode: log.status_code || 0,
    duration: `${formatCompactNumber(log.duration_millis)} ms`,
    labels: Array.isArray(log.labels) ? log.labels : [],
    error: log.error || '',
    commentURL: log.comment_url || '',
    when: when.full,
    tone: status === 'failed' || status === 'error' ? 'red' : status === 'skipped' ? 'amber' : 'green',
  };
}

function mapAdminTestSettingsEntry(entry = {}) {
  const mapKeys = Object.keys(entry.key_value_map || {});
  const when = formatLedgerDateTime(entry.updated_at);
  return {
    id: entry.id || `${entry.integration_type}-${entry.setting_key}`,
    integrationType: toTitleLabel(entry.integration_type || 'test'),
    displayName: entry.display_name || entry.setting_key || 'Test key',
    settingKey: entry.setting_key || '-',
    valueHint: entry.setting_value_hint || '****',
    mapKeys,
    status: toTitleLabel(entry.status || 'active'),
    updatedAt: when.full,
  };
}

function adminTaskRewardMRG(task = {}) {
  const reward = tokenAmountFromCents(task.reward_cents);
  return reward > 0 ? reward : 50;
}

function mapAdminTaskReviewRow(task = {}) {
  const title = task.title || `Task ${task.id || ''}`;
  return {
    id: task.id || `${task.project_id || 'task'}-${task.issue_number || title}`,
    issueNumber: Number(task.issue_number) || 0,
    title,
    projectID: task.project_id || '',
    issueURL: task.issue_url || '',
    status: toTitleLabel(task.status || 'open'),
    workerKind: toTitleLabel(task.required_worker_kind || task.worker_kind || 'worker'),
    bountyType: toTitleLabel(task.bounty_type || 'future-small'),
    reward: formatMRGFromCents(task.reward_cents),
    rewardMRG: adminTaskRewardMRG(task),
    acceptedWorker: task.worker_id || '',
  };
}

function adminPullTone(readiness = {}) {
  const risk = String(readiness.risk_level || readiness.status || '').toLowerCase();
  if (risk.includes('high') || risk.includes('blocked')) return 'red';
  if (risk.includes('medium') || risk.includes('review')) return 'amber';
  return 'green';
}

function mapAdminTaskPullGroup(group = {}) {
  const issueNumber = Number(group.issue_number) || 0;
  return {
    taskID: group.task_id || '',
    issueNumber,
    title: group.repository || `Issue #${issueNumber || '-'}`,
    issueURL: group.issue_url || '',
    repository: group.repository || '',
    pullRequests: (group.pull_requests || []).map((pull) => mapAdminTaskPullRow(group, pull)),
  };
}

function mapAdminTaskPullRow(group = {}, pull = {}) {
  const readiness = pull.readiness || {};
  const blockers = Array.isArray(readiness.blockers) ? readiness.blockers : [];
  const warnings = Array.isArray(readiness.warnings) ? readiness.warnings : [];
  const signals = Array.isArray(readiness.signals) ? readiness.signals : [];
  const labels = Array.isArray(pull.labels) ? pull.labels : [];
  const files = Array.isArray(pull.changed_files) ? pull.changed_files : [];
  const updated = formatLedgerDateTime(pull.updated_at || pull.created_at);
  const tone = adminPullTone(readiness);
  return {
    key: `${group.task_id || 'task'}:${pull.number || pull.html_url || pull.title}`,
    taskID: group.task_id || '',
    number: Number(pull.number) || 0,
    title: pull.title || `Pull request #${pull.number || ''}`,
    author: pull.author || 'unknown',
    state: toTitleLabel(pull.merged ? 'merged' : pull.state || 'open'),
    htmlURL: pull.html_url || '',
    mergeURL: pull.merge_url || '',
    draft: Boolean(pull.draft),
    merged: Boolean(pull.merged),
    baseRef: pull.base_ref || '-',
    headRef: pull.head_ref || '-',
    labels: labels.slice(0, 5),
    fileCount: files.length,
    status: toTitleLabel(readiness.status || 'review'),
    canMerge: Boolean(readiness.can_merge),
    riskLevel: toTitleLabel(readiness.risk_level || 'low'),
    blockers,
    warnings,
    signals,
    evidenceReady: signals.includes('evidence: provided'),
    starReady: signals.includes('star: verified'),
    tone,
    updatedAt: updated.full,
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

function liveFeedMetaFor(type = '') {
  const normalized = String(type);
  if (normalized.startsWith('ledger_')) {
    return ledgerMetaFor(normalized.replace(/^ledger_/, ''));
  }
  if (normalized === 'project_funded') {
    return { type: 'Project Funded', icon: FolderKanban, tone: 'green', amountTone: 'positive' };
  }
  if (normalized === 'task_opened') {
    return { type: 'Task Opened', icon: ListTodo, tone: 'blue', amountTone: 'neutral' };
  }
  if (normalized === 'task_accepted') {
    return { type: 'PR Accepted', icon: GitPullRequest, tone: 'green', amountTone: 'positive' };
  }
  if (normalized === 'deployment_validation') {
    return { type: 'Deployment Validation', icon: Rocket, tone: 'blue', amountTone: 'muted' };
  }
  if (normalized === 'ai_review') {
    return { type: 'AI Review', icon: Bot, tone: 'purple', amountTone: 'muted' };
  }
  return { type: toTitleLabel(normalized || 'Activity'), icon: Compass, tone: 'slate', amountTone: 'neutral' };
}

function mapPublicLiveFeedItem(item = {}) {
  const meta = liveFeedMetaFor(item.type);
  const when = formatLedgerDateTime(item.created_at);
  const amountCents = Number(item.amount_cents) || 0;
  const projectTitle = item.project_title || 'MergeOS';
  const reference = item.reference || '';
  return {
    id: item.id || `${item.type || 'activity'}-${item.created_at || reference || item.title || 'row'}`,
    rawType: item.type || '',
    typeLabel: meta.type,
    icon: meta.icon,
    tone: meta.tone,
    amountTone: meta.amountTone,
    title: item.title || meta.type,
    body: trimMarketplaceText(item.body, 'MergeOS live activity recorded.'),
    project: projectTitle,
    actor: item.actor || 'MergeOS',
    amount: amountCents ? formatPublicMRGFromCents(amountCents) : '',
    reference: shortLedgerReference(reference),
    rawReference: reference,
    url: item.url || '',
    status: toTitleLabel(item.status || 'live'),
    date: when.date,
    time: when.time,
    meta: `${when.full} • ${toTitleLabel(item.status || 'live')}`,
    createdAt: item.created_at,
  };
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
    createdAt: entry.created_at,
    rawType: String(entry.type || ''),
    projectID,
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
    attachment_ids: projectAttachments.value.map((file) => file.id).filter(Boolean),
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
    projectAttachments.value.length && `Reference files:\n${projectAttachments.value.map((file) => `- ${file.original_name || file.id}`).join('\n')}`,
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

function addPublicTestSettingsKVRow() {
  publicTestSettingsKeyValueRows.value.push({ key: '', value: '' });
}

function removePublicTestSettingsKVRow(index) {
  if (publicTestSettingsKeyValueRows.value.length <= 1) {
    publicTestSettingsKeyValueRows.value = [{ key: '', value: '' }];
    return;
  }
  publicTestSettingsKeyValueRows.value.splice(index, 1);
}

function resetPublicTestSettingsForm() {
  publicTestSettingsForm.displayName = '';
  publicTestSettingsForm.settingKey = '';
  publicTestSettingsForm.settingValue = '';
  publicTestSettingsKeyValueRows.value = [{ key: '', value: '' }];
}

function publicTestSettingsKeyValueMap() {
  return publicTestSettingsKeyValueRows.value.reduce((acc, row) => {
    const key = String(row.key || '').trim();
    if (!key) return acc;
    acc[key] = String(row.value || '');
    return acc;
  }, {});
}

function publicTestSettingsPasswordPayload(extra = {}) {
  return {
    password: publicTestSettingsPassword.value,
    ...extra,
  };
}

async function loadPublicTestSettingsStatus(options = {}) {
  const silent = Boolean(options.silent);
  if (!silent) publicTestSettingsLoading.value = true;
  publicTestSettingsError.value = '';
  try {
    const payload = await publicApi('/api/public/test-settings/status');
    publicTestSettingsStatus.value = {
      test_mode_enabled: Boolean(payload.test_mode_enabled),
    };
    if (!publicTestSettingsStatus.value.test_mode_enabled) {
      publicTestSettingsAuthenticated.value = false;
      publicTestSettingsEntries.value = [];
    }
  } catch (error) {
    publicTestSettingsError.value = error.message || 'Could not load test settings status';
  } finally {
    if (!silent) publicTestSettingsLoading.value = false;
  }
}

async function unlockPublicTestSettings() {
  publicTestSettingsError.value = '';
  if (!publicTestSettingsPassword.value.trim()) {
    publicTestSettingsError.value = 'Password is required.';
    return;
  }
  publicTestSettingsBusy.value = true;
  try {
    await publicApi('/api/public/test-settings/auth', {
      method: 'POST',
      body: JSON.stringify(publicTestSettingsPasswordPayload()),
    });
    publicTestSettingsAuthenticated.value = true;
    await loadPublicTestSettingsEntries({ silent: true });
    showToast('Test keys unlocked.');
  } catch (error) {
    publicTestSettingsAuthenticated.value = false;
    publicTestSettingsError.value = error.message || 'Could not unlock test keys';
  } finally {
    publicTestSettingsBusy.value = false;
  }
}

async function loadPublicTestSettingsEntries(options = {}) {
  if (!publicTestSettingsPassword.value.trim()) {
    publicTestSettingsError.value = 'Password is required.';
    return;
  }
  const silent = Boolean(options.silent);
  if (!silent) publicTestSettingsLoading.value = true;
  publicTestSettingsError.value = '';
  try {
    const rows = await publicApi('/api/public/test-settings/entries/list', {
      method: 'POST',
      body: JSON.stringify(publicTestSettingsPasswordPayload()),
    });
    publicTestSettingsEntries.value = Array.isArray(rows) ? rows : [];
  } catch (error) {
    publicTestSettingsError.value = error.message || 'Could not load test keys';
  } finally {
    if (!silent) publicTestSettingsLoading.value = false;
  }
}

async function addPublicTestSettingsEntry() {
  publicTestSettingsError.value = '';
  if (!publicTestSettingsAuthenticated.value) {
    publicTestSettingsError.value = 'Unlock with the shared password first.';
    return;
  }
  if (!publicTestSettingsForm.settingKey.trim() || !publicTestSettingsForm.settingValue.trim()) {
    publicTestSettingsError.value = 'Setting key and primary value are required.';
    return;
  }
  publicTestSettingsBusy.value = true;
  try {
    const entry = await publicApi('/api/public/test-settings/entries', {
      method: 'POST',
      body: JSON.stringify(publicTestSettingsPasswordPayload({
        integration_type: publicTestSettingsForm.integrationType,
        display_name: publicTestSettingsForm.displayName || publicTestSettingsForm.settingKey,
        setting_key: publicTestSettingsForm.settingKey,
        setting_value: publicTestSettingsForm.settingValue,
        key_value_map: publicTestSettingsKeyValueMap(),
      })),
    });
    publicTestSettingsEntries.value = [entry, ...publicTestSettingsEntries.value];
    resetPublicTestSettingsForm();
    showToast('Test key saved.');
  } catch (error) {
    publicTestSettingsError.value = error.message || 'Could not save test key';
  } finally {
    publicTestSettingsBusy.value = false;
  }
}

async function deletePublicTestSettingsEntry(entryID) {
  if (!entryID || !publicTestSettingsAuthenticated.value) return;
  publicTestSettingsBusy.value = true;
  publicTestSettingsError.value = '';
  try {
    await publicApi(`/api/public/test-settings/entries/${encodeURIComponent(entryID)}`, {
      method: 'DELETE',
      body: JSON.stringify(publicTestSettingsPasswordPayload()),
    });
    publicTestSettingsEntries.value = publicTestSettingsEntries.value.filter((entry) => entry.id !== entryID);
    showToast('Test key deleted.');
  } catch (error) {
    publicTestSettingsError.value = error.message || 'Could not delete test key';
  } finally {
    publicTestSettingsBusy.value = false;
  }
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
      bounties: Array.isArray(payload.bounties) ? payload.bounties : [],
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

async function loadLiveFeedData(options = {}) {
  const silent = Boolean(options.silent);
  if (!silent) liveFeedLoading.value = true;
  liveFeedError.value = '';
  try {
    const payload = await publicApi('/api/public/live-feed?limit=80');
    liveFeedData.value = {
      stats: payload.stats || {},
      items: Array.isArray(payload.items) ? payload.items : [],
    };
    if (!liveFeedActivityTypes.value.some((row) => row.label === activeLiveFeedType.value)) {
      activeLiveFeedType.value = 'All Activity';
    }
  } catch (error) {
    liveFeedError.value = error.message || 'Could not load live feed';
  } finally {
    liveFeedLoading.value = false;
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
    applyLedgerProjectQueryFilter();
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
    dashboardEscrow.value = null;
    dashboardEscrowLoading.value = false;
    dashboardEscrowError.value = '';
    dashboardDeployment.value = null;
    dashboardDeploymentLoading.value = false;
    dashboardDeploymentError.value = '';
    dashboardAIWorkflow.value = null;
    dashboardAIWorkflowLoading.value = false;
    dashboardAIWorkflowError.value = '';
    dashboardTaskGraph.value = null;
    dashboardTaskGraphLoading.value = false;
    dashboardTaskGraphError.value = '';
    dashboardPullRequests.value = null;
    dashboardPullRequestsLoading.value = false;
    dashboardPullRequestsError.value = '';
    dashboardRepositoryScan.value = null;
    dashboardRepositoryScanLoading.value = false;
    dashboardRepositoryScanError.value = '';
    dashboardSection.value = 'projects';
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
    if (selectedDashboardProjectID.value) {
      const detailLoads = [
        loadDashboardEscrowData(selectedDashboardProjectID.value, { silent: true }),
        loadDashboardDeploymentData(selectedDashboardProjectID.value, { silent: true }),
        loadDashboardAIWorkflowData(selectedDashboardProjectID.value, { silent: true }),
      ];
      if (!options.skipTaskGraph) {
        detailLoads.push(loadDashboardTaskGraphData(selectedDashboardProjectID.value, { silent: true }));
      }
      if (!options.skipPullRequests) {
        detailLoads.push(loadDashboardPullRequestsData(selectedDashboardProjectID.value, { silent: true }));
      }
      if (!options.skipRepositoryScan) {
        detailLoads.push(loadDashboardRepositoryScanData(selectedDashboardProjectID.value, { silent: true }));
      }
      await Promise.all(detailLoads);
    } else {
      dashboardEscrow.value = null;
      dashboardEscrowError.value = '';
      dashboardDeployment.value = null;
      dashboardDeploymentError.value = '';
      dashboardAIWorkflow.value = null;
      dashboardAIWorkflowError.value = '';
      dashboardTaskGraph.value = null;
      dashboardTaskGraphError.value = '';
      dashboardPullRequests.value = null;
      dashboardPullRequestsError.value = '';
      dashboardRepositoryScan.value = null;
      dashboardRepositoryScanError.value = '';
    }
  } catch (error) {
    dashboardError.value = error.message || 'Could not load projects';
  } finally {
    dashboardLoading.value = false;
  }
}

async function loadDashboardEscrowData(projectID, options = {}) {
  const targetProjectID = String(projectID || '').trim();
  if (!token.value || !targetProjectID) {
    dashboardEscrow.value = null;
    dashboardEscrowError.value = '';
    return;
  }

  const silent = Boolean(options.silent);
  if (!silent) dashboardEscrowLoading.value = true;
  dashboardEscrowError.value = '';
  try {
    const payload = await api(`/api/projects/${encodeURIComponent(targetProjectID)}/escrow`);
    dashboardEscrow.value = {
      project_id: payload.project_id || targetProjectID,
      project_title: payload.project_title || '',
      token_symbol: payload.token_symbol || tokenSymbol.value,
      release_status: payload.release_status || 'funded',
      budget_cents: Number(payload.budget_cents) || 0,
      fee_cents: Number(payload.fee_cents) || 0,
      work_pool_cents: Number(payload.work_pool_cents) || 0,
      project_reserve_cents: Number(payload.project_reserve_cents) || 0,
      task_reserve_cents: Number(payload.task_reserve_cents) || 0,
      task_payment_cents: Number(payload.task_payment_cents) || 0,
      manual_credit_cents: Number(payload.manual_credit_cents) || 0,
      released_cents: Number(payload.released_cents) || 0,
      remaining_cents: Number(payload.remaining_cents) || 0,
      overdrawn_cents: Number(payload.overdrawn_cents) || 0,
      unallocated_cents: Number(payload.unallocated_cents) || 0,
      paid_task_count: Number(payload.paid_task_count) || 0,
      open_task_count: Number(payload.open_task_count) || 0,
      updated_at: payload.updated_at,
      tasks: Array.isArray(payload.tasks) ? payload.tasks : [],
    };
  } catch (error) {
    if (targetProjectID === selectedDashboardProjectID.value) {
      dashboardEscrow.value = null;
      dashboardEscrowError.value = error.message || 'Could not load escrow summary';
    }
  } finally {
    dashboardEscrowLoading.value = false;
  }
}

async function loadDashboardDeploymentData(projectID, options = {}) {
  const targetProjectID = String(projectID || '').trim();
  if (!token.value || !targetProjectID) {
    dashboardDeployment.value = null;
    dashboardDeploymentError.value = '';
    return;
  }

  const silent = Boolean(options.silent);
  if (!silent) dashboardDeploymentLoading.value = true;
  dashboardDeploymentError.value = '';
  try {
    const payload = await api(`/api/projects/${encodeURIComponent(targetProjectID)}/deployment`);
    dashboardDeployment.value = {
      project_id: payload.project_id || targetProjectID,
      project_title: payload.project_title || '',
      status: payload.status || 'queued',
      progress: Number(payload.progress) || 0,
      updated_at: payload.updated_at,
      stages: Array.isArray(payload.stages) ? payload.stages : [],
      signals: Array.isArray(payload.signals) ? payload.signals : [],
    };
  } catch (error) {
    if (targetProjectID === selectedDashboardProjectID.value) {
      dashboardDeployment.value = null;
      dashboardDeploymentError.value = error.message || 'Could not load deployment validation';
    }
  } finally {
    dashboardDeploymentLoading.value = false;
  }
}

async function loadDashboardAIWorkflowData(projectID, options = {}) {
  const targetProjectID = String(projectID || '').trim();
  if (!token.value || !targetProjectID) {
    dashboardAIWorkflow.value = null;
    dashboardAIWorkflowError.value = '';
    return;
  }

  const silent = Boolean(options.silent);
  if (!silent) dashboardAIWorkflowLoading.value = true;
  dashboardAIWorkflowError.value = '';
  try {
    const payload = await api(`/api/projects/${encodeURIComponent(targetProjectID)}/ai-workflow`);
    dashboardAIWorkflow.value = {
      project_id: payload.project_id || targetProjectID,
      project_title: payload.project_title || '',
      status: payload.status || 'queued',
      progress: Number(payload.progress) || 0,
      task_count: Number(payload.task_count) || 0,
      agent_task_count: Number(payload.agent_task_count) || 0,
      human_task_count: Number(payload.human_task_count) || 0,
      hybrid_task_count: Number(payload.hybrid_task_count) || 0,
      ai_action_count: Number(payload.ai_action_count) || 0,
      updated_at: payload.updated_at,
      stages: Array.isArray(payload.stages) ? payload.stages : [],
      signals: Array.isArray(payload.signals) ? payload.signals : [],
    };
  } catch (error) {
    if (targetProjectID === selectedDashboardProjectID.value) {
      dashboardAIWorkflow.value = null;
      dashboardAIWorkflowError.value = error.message || 'Could not load AI workflow';
    }
  } finally {
    dashboardAIWorkflowLoading.value = false;
  }
}

async function loadDashboardTaskGraphData(projectID, options = {}) {
  const targetProjectID = String(projectID || '').trim();
  if (!token.value || !targetProjectID) {
    dashboardTaskGraph.value = null;
    dashboardTaskGraphError.value = '';
    return;
  }

  const silent = Boolean(options.silent);
  if (!silent) dashboardTaskGraphLoading.value = true;
  dashboardTaskGraphError.value = '';
  try {
    const payload = await api(`/api/projects/${encodeURIComponent(targetProjectID)}/task-graph`);
    dashboardTaskGraph.value = {
      project_id: payload.project_id || targetProjectID,
      project_title: payload.project_title || '',
      status: payload.status || 'planning',
      progress: Number(payload.progress) || 0,
      stats: payload.stats || {},
      nodes: Array.isArray(payload.nodes) ? payload.nodes : [],
      edges: Array.isArray(payload.edges) ? payload.edges : [],
      updated_at: payload.updated_at,
    };
  } catch (error) {
    if (targetProjectID === selectedDashboardProjectID.value) {
      dashboardTaskGraph.value = null;
      dashboardTaskGraphError.value = error.message || 'Could not load task graph';
    }
  } finally {
    dashboardTaskGraphLoading.value = false;
  }
}

async function loadDashboardRepositoryScanData(projectID, options = {}) {
  const targetProjectID = String(projectID || '').trim();
  if (!token.value || !targetProjectID) {
    dashboardRepositoryScan.value = null;
    dashboardRepositoryScanError.value = '';
    return;
  }

  const silent = Boolean(options.silent);
  if (!silent) dashboardRepositoryScanLoading.value = true;
  dashboardRepositoryScanError.value = '';
  try {
    const payload = await api(`/api/projects/${encodeURIComponent(targetProjectID)}/repo-scan`);
    dashboardRepositoryScan.value = {
      project_id: payload.project_id || targetProjectID,
      project_title: payload.project_title || '',
      status: payload.status || 'scanned',
      summary: payload.summary || '',
      stats: payload.stats || {},
      languages: Array.isArray(payload.languages) ? payload.languages : [],
      dependencies: Array.isArray(payload.dependencies) ? payload.dependencies : [],
      findings: Array.isArray(payload.findings) ? payload.findings : [],
      updated_at: payload.updated_at,
    };
  } catch (error) {
    if (targetProjectID === selectedDashboardProjectID.value) {
      dashboardRepositoryScan.value = null;
      dashboardRepositoryScanError.value = error.message || 'Could not load repository scan';
    }
  } finally {
    dashboardRepositoryScanLoading.value = false;
  }
}

async function loadDashboardPullRequestsData(projectID, options = {}) {
  const targetProjectID = String(projectID || '').trim();
  if (!token.value || !targetProjectID) {
    dashboardPullRequests.value = null;
    dashboardPullRequestsError.value = '';
    return;
  }

  const silent = Boolean(options.silent);
  if (!silent) dashboardPullRequestsLoading.value = true;
  dashboardPullRequestsError.value = '';
  try {
    const payload = await api(`/api/projects/${encodeURIComponent(targetProjectID)}/pull-requests`);
    dashboardPullRequests.value = {
      project_id: payload.project_id || targetProjectID,
      project_title: payload.project_title || '',
      stats: payload.stats || {},
      tasks: Array.isArray(payload.tasks) ? payload.tasks : [],
      updated_at: payload.updated_at,
    };
  } catch (error) {
    if (targetProjectID === selectedDashboardProjectID.value) {
      dashboardPullRequests.value = null;
      dashboardPullRequestsError.value = error.message || 'Could not load pull request monitor';
    }
  } finally {
    dashboardPullRequestsLoading.value = false;
  }
}

async function loadWorkerDashboardData(options = {}) {
  if (!token.value) {
    workerDashboard.value = {
      profile: {},
      stats: {},
      claimed_tasks: [],
      rewards: [],
      reputation: [],
      proposals: [],
      identity_status: [],
    };
    workerDashboardError.value = '';
    return;
  }

  const silent = Boolean(options.silent);
  if (!silent) workerDashboardLoading.value = true;
  workerDashboardError.value = '';
  try {
    const payload = await api('/api/workers/me');
    workerDashboard.value = {
      profile: payload.profile || {},
      stats: payload.stats || {},
      claimed_tasks: Array.isArray(payload.claimed_tasks) ? payload.claimed_tasks : [],
      rewards: Array.isArray(payload.rewards) ? payload.rewards : [],
      reputation: Array.isArray(payload.reputation) ? payload.reputation : [],
      proposals: Array.isArray(payload.proposals) ? payload.proposals : [],
      identity_status: Array.isArray(payload.identity_status) ? payload.identity_status : [],
    };
  } catch (error) {
    workerDashboardError.value = error.message || 'Could not load worker dashboard';
  } finally {
    workerDashboardLoading.value = false;
  }
}

async function loadAdminConsoleData(options = {}) {
  if (!token.value || !isAdminUser.value) {
    adminSummary.value = null;
    adminOpsQueue.value = { stats: {}, items: [] };
    adminReputation.value = { stats: {}, workers: [] };
    adminUsers.value = [];
    adminTasks.value = [];
    adminTaskPulls.value = {};
    adminTaskPullsLoadingID.value = '';
    adminTaskPullsError.value = '';
    adminMergeError.value = '';
    adminMergeResult.value = null;
    adminSSLReviews.value = [];
    adminSSLReviewBusy.value = false;
    adminSSLReviewError.value = '';
    adminSettings.value = { llm_provider_options: [] };
    adminLLMKeys.value = [];
    adminLLMWebhooks.value = [];
    adminLLMBusy.value = false;
    adminLLMError.value = '';
    adminLLMKeyBusyID.value = '';
    adminLLMForm.provider = 'gemini';
    adminLLMForm.model = 'gemini-2.5-flash';
    adminLLMForm.apiKey = '';
    adminTestSettings.value = { test_mode_enabled: false, updated_at: '' };
    adminTestSettingsEntries.value = [];
    adminConsoleError.value = '';
    return;
  }

  const silent = Boolean(options.silent);
  const shouldLoadTasks = !silent || !adminTasks.value.length || Boolean(options.refreshTasks);
  if (!silent) adminConsoleLoading.value = true;
  adminConsoleError.value = '';
  try {
    const [summary, opsQueue, reputation, users, tasks, sslReviews, settings, llmKeys, llmWebhooks, testSettings, testEntries] = await Promise.all([
      api('/api/admin/summary'),
      api('/api/admin/ops-queue'),
      api('/api/admin/reputation'),
      api('/api/admin/users'),
      shouldLoadTasks ? api('/api/admin/tasks') : Promise.resolve(adminTasks.value),
      api('/api/admin/ssl'),
      api('/api/admin/settings'),
      api('/api/admin/gemini/keys'),
      api('/api/admin/gemini/webhooks?limit=5'),
      api('/api/admin/test-settings'),
      api('/api/admin/test-settings/entries'),
    ]);
    adminSummary.value = summary || {};
    adminOpsQueue.value = {
      stats: opsQueue?.stats || {},
      items: Array.isArray(opsQueue?.items) ? opsQueue.items : [],
    };
    adminReputation.value = {
      stats: reputation?.stats || {},
      workers: Array.isArray(reputation?.workers) ? reputation.workers : [],
    };
    adminUsers.value = Array.isArray(users) ? users : [];
    adminTasks.value = Array.isArray(tasks) ? tasks : [];
    adminSSLReviews.value = Array.isArray(sslReviews) ? sslReviews : [];
    adminSettings.value = settings || { llm_provider_options: [] };
    hydrateAdminLLMForm(adminSettings.value);
    adminLLMKeys.value = Array.isArray(llmKeys) ? llmKeys : [];
    adminLLMWebhooks.value = Array.isArray(llmWebhooks) ? llmWebhooks : [];
    adminTestSettings.value = {
      test_mode_enabled: Boolean(testSettings?.test_mode_enabled),
      updated_at: testSettings?.updated_at || '',
    };
    adminTestSettingsEntries.value = Array.isArray(testEntries) ? testEntries : [];
  } catch (error) {
    adminConsoleError.value = error.message || 'Could not load admin console';
  } finally {
    adminConsoleLoading.value = false;
  }
}

async function submitAdminTestSettings() {
  adminTestSettingsBusy.value = true;
  adminTestSettingsError.value = '';
  try {
    const payload = {
      test_mode_enabled: Boolean(adminTestSettings.value.test_mode_enabled),
      test_password: adminTestSettingsPassword.value || undefined,
    };
    const updated = await api('/api/admin/test-settings', {
      method: 'PATCH',
      body: JSON.stringify(payload),
    });
    adminTestSettings.value = {
      test_mode_enabled: Boolean(updated?.test_mode_enabled),
      updated_at: updated?.updated_at || '',
    };
    adminTestSettingsPassword.value = '';
    await loadPublicTestSettingsStatus({ silent: true });
    showToast(`Test settings ${adminTestSettings.value.test_mode_enabled ? 'enabled' : 'disabled'}.`);
  } catch (error) {
    adminTestSettingsError.value = error.message || 'Could not update test settings';
  } finally {
    adminTestSettingsBusy.value = false;
  }
}

async function runAdminSSLReview() {
  adminSSLReviewBusy.value = true;
  adminSSLReviewError.value = '';
  try {
    const rows = await api('/api/admin/ssl/review', { method: 'POST' });
    adminSSLReviews.value = Array.isArray(rows) ? rows : [];
    await loadAdminConsoleData({ silent: true });
    showToast('SSL review completed.');
  } catch (error) {
    adminSSLReviewError.value = error.message || 'Could not run SSL review';
  } finally {
    adminSSLReviewBusy.value = false;
  }
}

function hydrateAdminLLMForm(settings = {}) {
  const provider = settings.llm_provider || adminLLMForm.provider || 'gemini';
  const options = Array.isArray(settings.llm_provider_options) && settings.llm_provider_options.length
    ? settings.llm_provider_options
    : adminLLMProviderOptions.value;
  const selected = options.find((option) => option.id === provider) || options[0];
  adminLLMForm.provider = selected?.id || provider;
  const models = Array.isArray(selected?.models) ? selected.models : [];
  const model = settings.llm_model || adminLLMForm.model || models[0] || '';
  adminLLMForm.model = models.includes(model) ? model : (models[0] || model);
}

function handleAdminLLMProviderChange() {
  const selected = adminLLMProviderOptions.value.find((option) => option.id === adminLLMForm.provider);
  const models = Array.isArray(selected?.models) ? selected.models : [];
  adminLLMForm.model = models[0] || '';
}

async function submitAdminLLMSettings() {
  adminLLMBusy.value = true;
  adminLLMError.value = '';
  try {
    const settings = await api('/api/admin/settings', {
      method: 'PATCH',
      body: JSON.stringify({
        llm_provider: adminLLMForm.provider,
        llm_model: adminLLMForm.model,
      }),
    });
    adminSettings.value = settings || adminSettings.value;
    hydrateAdminLLMForm(adminSettings.value);
    showToast(`AI review provider set to ${toTitleLabel(adminLLMForm.provider)}.`);
  } catch (error) {
    adminLLMError.value = error.message || 'Could not update AI review provider';
  } finally {
    adminLLMBusy.value = false;
  }
}

async function submitAdminLLMKey() {
  adminLLMBusy.value = true;
  adminLLMError.value = '';
  try {
    await api('/api/admin/gemini/keys', {
      method: 'POST',
      body: JSON.stringify({
        api_key: adminLLMForm.apiKey,
        provider: adminLLMForm.provider,
        model: adminLLMForm.model,
      }),
    });
    adminLLMForm.apiKey = '';
    await loadAdminConsoleData({ silent: true });
    showToast('AI review API key added.');
  } catch (error) {
    adminLLMError.value = error.message || 'Could not add AI review API key';
  } finally {
    adminLLMBusy.value = false;
  }
}

async function updateAdminLLMKey(row = {}, status = '', resetCounts = false) {
  if (!row.id) return;
  const action = resetCounts ? 'reset' : status || 'update';
  adminLLMKeyBusyID.value = `${row.id}:${action}`;
  adminLLMError.value = '';
  try {
    await api(`/api/admin/gemini/keys/${encodeURIComponent(row.id)}`, {
      method: 'PATCH',
      body: JSON.stringify({
        status,
        reset_counts: Boolean(resetCounts),
      }),
    });
    await loadAdminConsoleData({ silent: true });
    showToast(resetCounts ? 'AI key counters reset.' : `AI key ${toTitleLabel(status)}.`);
  } catch (error) {
    adminLLMError.value = error.message || 'Could not update AI review API key';
  } finally {
    adminLLMKeyBusyID.value = '';
  }
}

async function testAdminLLMKey(row = {}) {
  if (!row.id) return;
  adminLLMKeyBusyID.value = `${row.id}:test`;
  adminLLMError.value = '';
  try {
    const result = await api(`/api/admin/gemini/keys/${encodeURIComponent(row.id)}/test`, {
      method: 'POST',
      body: JSON.stringify({
        provider: row.providerValue,
        model: row.model,
      }),
    });
    await loadAdminConsoleData({ silent: true });
    showToast(result.ok ? 'AI key test passed.' : (result.error || 'AI key test failed.'));
  } catch (error) {
    adminLLMError.value = error.message || 'Could not test AI review API key';
  } finally {
    adminLLMKeyBusyID.value = '';
  }
}

async function loadAdminTaskPulls(taskID, options = {}) {
  const id = String(taskID || '').trim();
  if (!id) return;
  const silent = Boolean(options.silent);
  adminTaskPullsError.value = '';
  if (!silent) adminTaskPullsLoadingID.value = id;
  try {
    const payload = await api(`/api/admin/tasks/${encodeURIComponent(id)}/pulls`);
    adminTaskPulls.value = {
      ...adminTaskPulls.value,
      [id]: {
        task_id: payload?.task_id || id,
        issue_number: payload?.issue_number || 0,
        issue_url: payload?.issue_url || '',
        repository: payload?.repository || '',
        pull_requests: Array.isArray(payload?.pull_requests) ? payload.pull_requests : [],
      },
    };
    const taskRow = adminTaskReviewRows.value.find((row) => row.id === id);
    if (taskRow?.rewardMRG && Number(adminMergeForm.rewardMRG) <= 0) {
      adminMergeForm.rewardMRG = taskRow.rewardMRG;
    }
  } catch (error) {
    adminTaskPullsError.value = error.message || 'Could not load linked pull requests';
  } finally {
    if (adminTaskPullsLoadingID.value === id) {
      adminTaskPullsLoadingID.value = '';
    }
  }
}

async function mergeAdminTaskPull(taskID, pull = {}) {
  const id = String(taskID || '').trim();
  const number = Number(pull.number) || 0;
  if (!id || number <= 0) return;
  adminMergeError.value = '';
  adminMergeResult.value = null;
  if (!adminMergeReady.value) {
    adminMergeError.value = 'Reward MRG and bounty type are required before merging.';
    return;
  }
  const key = `${id}:${number}`;
  adminMergeBusyID.value = key;
  try {
    const result = await api(`/api/admin/tasks/${encodeURIComponent(id)}/pulls/${number}/merge`, {
      method: 'POST',
      body: JSON.stringify({
        reward_mrg: Number(adminMergeForm.rewardMRG) || 0,
        bounty_type: adminMergeForm.bountyType,
      }),
    });
    adminMergeResult.value = result;
    showToast(`Merged PR #${number} and credited ${formatCompactNumber(result.reward_mrg)} ${tokenSymbol.value}.`);
    await loadAdminTaskPulls(id, { silent: true });
    await loadAdminConsoleData({ silent: true, refreshTasks: true });
  } catch (error) {
    adminMergeError.value = error.message || 'Could not merge pull request';
  } finally {
    if (adminMergeBusyID.value === key) {
      adminMergeBusyID.value = '';
    }
  }
}

function prefillAdminCreditFromUser(row = {}) {
  if (!row.workerID) return;
  adminCreditForm.workerID = row.workerID;
  if (!adminCreditForm.reference && !adminCreditForm.prURL) {
    adminCreditForm.reference = `admin-credit:${row.id}`;
  }
}

async function submitAdminManualCredit() {
  adminCreditError.value = '';
  adminCreditResult.value = null;
  if (!adminCreditReady.value) {
    adminCreditError.value = 'Worker ID, reward, and PR URL or reference are required.';
    return;
  }
  adminCreditBusy.value = true;
  try {
    const payload = {
      worker_id: adminCreditForm.workerID,
      reward_mrg: Number(adminCreditForm.rewardMRG) || 0,
      bounty_type: adminCreditForm.bountyType,
      task_id: adminCreditForm.taskID || undefined,
      pr_url: adminCreditForm.prURL || undefined,
      pr_title: adminCreditForm.prTitle || undefined,
      reference: adminCreditForm.reference || undefined,
    };
    const result = await api('/api/admin/ledger/credits', {
      method: 'POST',
      body: JSON.stringify(payload),
    });
    adminCreditResult.value = result;
    adminCreditForm.reference = '';
    adminCreditForm.prURL = '';
    adminCreditForm.prTitle = '';
    showToast(`Credited ${formatCompactNumber(result.reward_mrg)} ${tokenSymbol.value}.`);
    await loadAdminConsoleData({ silent: true });
  } catch (error) {
    adminCreditError.value = error.message || 'Could not create manual credit';
  } finally {
    adminCreditBusy.value = false;
  }
}

async function loadDashboardNotifications() {
  if (!token.value) {
    dashboardNotifications.value = [];
    dashboardNotificationsError.value = '';
    return;
  }
  dashboardNotificationsLoading.value = true;
  dashboardNotificationsError.value = '';
  try {
    const rows = await api('/api/notifications');
    dashboardNotifications.value = Array.isArray(rows) ? rows : [];
  } catch (error) {
    dashboardNotificationsError.value = error.message || 'Could not load notifications';
  } finally {
    dashboardNotificationsLoading.value = false;
  }
}

async function markAllNotificationsRead() {
  if (!token.value) return;
  try {
    await api('/api/notifications/read-all', { method: 'POST' });
    await loadDashboardNotifications();
  } catch (error) {
    showToast(error.message || 'Could not mark notifications as read.');
  }
}

async function markNotificationAsRead(notificationId) {
  if (!token.value || !notificationId) return;
  try {
    await api('/api/notifications/read', {
      method: 'POST',
      body: JSON.stringify({ notification_id: notificationId }),
    });
    const idx = dashboardNotifications.value.findIndex((n) => n.id === notificationId);
    if (idx !== -1) {
      dashboardNotifications.value[idx] = {
        ...dashboardNotifications.value[idx],
        read_at: new Date().toISOString(),
      };
    }
  } catch {
    // silently ignore — notification will refresh on next poll
  }
}

function handleNotificationClick(note) {
  if (note.isUnread) {
    markNotificationAsRead(note.id);
  }
}

function startDashboardRealtime() {
  if (!hasWindow || dashboardRefreshTimer) return;
  dashboardRefreshTimer = window.setInterval(() => {
    if (!token.value || !user.value) return;
    if (document.visibilityState === 'hidden') return;
    void loadDashboardData({ silent: true, skipPullRequests: true, skipRepositoryScan: true, skipTaskGraph: true });
    if (dashboardSection.value === 'worker') {
      void loadWorkerDashboardData({ silent: true });
    }
    if (dashboardSection.value === 'admin' && isAdminUser.value) {
      void loadAdminConsoleData({ silent: true });
    }
    void loadDashboardNotifications();
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
  if (publicPage.value === 'live') {
    void loadLiveFeedData({ silent: true });
  }
  void loadDashboardData({ silent: true });
  void loadWorkerDashboardData({ silent: true });
  if (isAdminUser.value) {
    void loadAdminConsoleData({ silent: true });
  }
  void loadDashboardNotifications();
  startDashboardRealtime();
}

let _ws = null;
let _wsReconnectTimer = 0;
let _wsReconnectDelay = 1000;
const _wsSeenProjectIDs = new Set();

function wsURL() {
  if (!hasWindow) return '';
  const proto = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
  return `${proto}//${window.location.host}/api/ws`;
}

function connectWebSocket() {
  if (!hasWindow || _ws) return;
  try {
    _ws = new WebSocket(wsURL());
  } catch {
    _ws = null;
    scheduleWSReconnect();
    return;
  }
  _ws.onopen = () => {
    _wsReconnectDelay = 1000;
  };
  _ws.onmessage = (event) => {
    let payload;
    try {
      payload = JSON.parse(event.data);
    } catch {
      return;
    }
    handleWSEvent(payload);
  };
  _ws.onerror = () => {};
  _ws.onclose = () => {
    _ws = null;
    scheduleWSReconnect();
  };
}

function disconnectWebSocket() {
  if (_wsReconnectTimer) {
    window.clearTimeout(_wsReconnectTimer);
    _wsReconnectTimer = 0;
  }
  _wsReconnectDelay = 1000;
  _wsSeenProjectIDs.clear();
  if (!_ws) return;
  _ws.onclose = null;
  _ws.onerror = null;
  _ws.onmessage = null;
  _ws.onopen = null;
  try {
    _ws.close();
  } catch {
    // ignore close errors
  }
  _ws = null;
}

function scheduleWSReconnect() {
  if (!hasWindow || _wsReconnectTimer) return;
  _wsReconnectTimer = window.setTimeout(() => {
    _wsReconnectTimer = 0;
    _wsReconnectDelay = Math.min(_wsReconnectDelay * 2, 30000);
    connectWebSocket();
  }, _wsReconnectDelay);
}

function handleWSEvent(payload = {}) {
  if (!payload || payload.type !== 'project_created') return;
  const project = payload.project;
  if (!project || !project.id) return;

  if (_wsSeenProjectIDs.has(project.id)) return;
  _wsSeenProjectIDs.add(project.id);

  if (user.value) {
    const isAdmin = user.value.role === 'admin';
    const isOwner = user.value.id === project.client_user_id;
    if (isAdmin || isOwner) {
      const exists = dashboardProjects.value.some((p) => p.id === project.id);
      if (!exists) {
        dashboardProjects.value = [project, ...dashboardProjects.value];
        if (!selectedDashboardProjectID.value) {
          selectedDashboardProjectID.value = project.id;
        }
      }
    }
  }

  if (marketplaceData.value && Array.isArray(marketplaceData.value.projects)) {
    const exists = marketplaceData.value.projects.some((p) => p.id === project.id);
    if (!exists) {
      marketplaceData.value = {
        ...marketplaceData.value,
        projects: [project, ...marketplaceData.value.projects],
        stats: {
          ...marketplaceData.value.stats,
          project_count: (Number(marketplaceData.value.stats?.project_count) || marketplaceData.value.projects.length) + 1,
          total_budget_cents: (Number(marketplaceData.value.stats?.total_budget_cents) || 0) + (Number(project.budget_cents) || 0),
        },
      };
    }
  }
  void loadMarketplaceData({ silent: true });
  void loadLiveFeedData({ silent: true });
}

function clearSession() {
  disconnectWebSocket();
  stopDashboardRealtime();
  token.value = null;
  user.value = null;
  authVisible.value = false;
  ledgerError.value = '';
  errorMessage.value = '';
  dashboardProjects.value = [];
  dashboardTasks.value = [];
  dashboardLedgerEntries.value = [];
  dashboardEscrow.value = null;
  dashboardEscrowLoading.value = false;
  dashboardEscrowError.value = '';
  dashboardDeployment.value = null;
  dashboardDeploymentLoading.value = false;
  dashboardDeploymentError.value = '';
  dashboardAIWorkflow.value = null;
  dashboardAIWorkflowLoading.value = false;
  dashboardAIWorkflowError.value = '';
  dashboardTaskGraph.value = null;
  dashboardTaskGraphLoading.value = false;
  dashboardTaskGraphError.value = '';
  dashboardPullRequests.value = null;
  dashboardPullRequestsLoading.value = false;
  dashboardPullRequestsError.value = '';
  dashboardRepositoryScan.value = null;
  dashboardRepositoryScanLoading.value = false;
  dashboardRepositoryScanError.value = '';
  dashboardNotifications.value = [];
  dashboardNotificationsError.value = '';
  workerDashboard.value = {
    profile: {},
    stats: {},
    claimed_tasks: [],
    rewards: [],
    reputation: [],
    proposals: [],
    identity_status: [],
  };
  workerDashboardError.value = '';
  adminSummary.value = null;
  adminOpsQueue.value = { stats: {}, items: [] };
  adminReputation.value = { stats: {}, workers: [] };
  adminUsers.value = [];
  adminTasks.value = [];
  adminTaskPulls.value = {};
  adminTaskPullsLoadingID.value = '';
  adminTaskPullsError.value = '';
  adminMergeBusyID.value = '';
  adminMergeError.value = '';
  adminMergeResult.value = null;
  adminSSLReviews.value = [];
  adminSSLReviewBusy.value = false;
  adminSSLReviewError.value = '';
  adminSettings.value = { llm_provider_options: [] };
  adminLLMKeys.value = [];
  adminLLMWebhooks.value = [];
  adminLLMBusy.value = false;
  adminLLMError.value = '';
  adminLLMKeyBusyID.value = '';
  adminLLMForm.provider = 'gemini';
  adminLLMForm.model = 'gemini-2.5-flash';
  adminLLMForm.apiKey = '';
  adminTestSettings.value = { test_mode_enabled: false, updated_at: '' };
  adminTestSettingsEntries.value = [];
  adminTestSettingsPassword.value = '';
  adminTestSettingsBusy.value = false;
  adminTestSettingsError.value = '';
  adminConsoleLoading.value = false;
  adminConsoleError.value = '';
  adminCreditError.value = '';
  adminCreditResult.value = null;
  dashboardError.value = '';
  dashboardSection.value = 'projects';
  selectedDashboardProjectID.value = '';
  pendingProjectPaymentAfterAuth.value = false;
  authReturnToProjectWizard.value = false;
  removeStoredToken();

  if (!publicModeVisible.value || projectWizardVisible.value) {
    projectWizardVisible.value = false;
    openPublicPage('home');
  }
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
    await loadWorkerDashboardData({ silent: true });
    if (isAdminUser.value) {
      await loadAdminConsoleData({ silent: true });
    }
    await loadDashboardNotifications();
    startDashboardRealtime();
    if (publicPage.value === 'ledger') {
      void loadLedgerData({ silent: true });
    }
  } catch {
    clearSession();
  }
}

async function logout() {
  const req = api('/api/auth/logout', { method: 'POST', body: JSON.stringify({}) }).catch((err) => {
    console.warn('Backend logout failed gracefully', err);
  });

  publicModeVisible.value = true;
  clearSession();
  publicPage.value = 'home';
  updatePublicBrowserPath('home', true);
  showToast('Logged out.');

  await req;
}

onMounted(async () => {
  connectWebSocket();
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
    loadLiveFeedData({ silent: true }),
    publicPage.value === 'test-settings' ? loadPublicTestSettingsStatus({ silent: true }) : Promise.resolve(),
  ]);
});

onUnmounted(() => {
  disconnectWebSocket();
  if (hasWindow) {
    window.removeEventListener('popstate', syncPublicPageFromBrowserPath);
  }
  stopDashboardRealtime();
});
</script>
