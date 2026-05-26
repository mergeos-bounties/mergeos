<template>
  <div v-if="!user" class="auth-shell">
    <section class="auth-panel">
      <div class="brand-lockup">
        <div class="brand-mark">
          <PanelsTopLeft :size="25" />
        </div>
        <div>
          <p class="eyebrow">Client portal</p>
          <h1>MergeOS</h1>
        </div>
      </div>

      <div class="auth-copy">
        <h2>Fund a website, get a private bounty repo, track every paid task.</h2>
        <p>Register as a client, verify PayPal or crypto payment, and MergeOS converts the budget into MERGE credits for human and agent delivery.</p>
      </div>

      <div class="segmented">
        <button :class="{ active: authMode === 'register' }" @click="authMode = 'register'">Register</button>
        <button :class="{ active: authMode === 'login' }" @click="authMode = 'login'">Login</button>
      </div>

      <form class="auth-form" @submit.prevent="submitAuth">
        <label v-if="authMode === 'register'">
          Full name
          <input v-model="authForm.name" autocomplete="name" />
        </label>
        <label v-if="authMode === 'register'">
          Company
          <input v-model="authForm.company_name" autocomplete="organization" />
        </label>
        <label>
          Email
          <input v-model="authForm.email" autocomplete="email" type="email" />
        </label>
        <label>
          Password
          <input v-model="authForm.password" autocomplete="current-password" type="password" />
        </label>
        <button class="primary-button" :disabled="authBusy">
          <LogIn :size="17" />
          <span>{{ authBusy ? 'Working...' : authMode === 'register' ? 'Create account' : 'Login' }}</span>
        </button>
        <p v-if="errorMessage" class="error-line">{{ errorMessage }}</p>
      </form>

      <div class="auth-runtime">
        <span>{{ runtimeConfig?.payment_mode || 'loading payment' }}</span>
        <span>{{ runtimeConfig?.repo_provider || 'loading repo' }}</span>
        <span>{{ runtimeConfig?.smtp_ready ? 'smtp ready' : 'email log mode' }}</span>
      </div>
    </section>
  </div>

  <div v-else-if="isAdmin" class="admin-shell">
    <header class="topbar">
      <div class="brand-lockup compact">
        <div class="brand-mark">
          <PanelsTopLeft :size="23" />
        </div>
        <div>
          <p class="eyebrow">Admin console</p>
          <h1>MergeOS</h1>
        </div>
      </div>
      <nav class="top-tabs">
        <button :class="{ active: adminTab === 'overview' }" @click="adminTab = 'overview'">
          <LayoutDashboard :size="16" />
          <span>Overview</span>
        </button>
        <button :class="{ active: adminTab === 'projects' }" @click="adminTab = 'projects'">
          <FolderKanban :size="16" />
          <span>Projects</span>
        </button>
        <button :class="{ active: adminTab === 'users' }" @click="adminTab = 'users'">
          <UserRound :size="16" />
          <span>Users</span>
        </button>
        <button :class="{ active: adminTab === 'ledger' }" @click="adminTab = 'ledger'">
          <WalletCards :size="16" />
          <span>Ledger</span>
        </button>
        <button :class="{ active: adminTab === 'inbox' }" @click="adminTab = 'inbox'">
          <Mail :size="16" />
          <span>Email</span>
        </button>
      </nav>
      <div class="topbar-spacer" />
      <div class="status-pill">
        <ShieldCheck :size="16" />
        <span>{{ adminSummary?.repo_provider || runtimeConfig?.repo_provider || 'admin ready' }}</span>
      </div>
      <button class="icon-button" title="Refresh admin data" @click="refreshAll">
        <RefreshCw :size="18" />
      </button>
      <button class="icon-button" title="Logout" @click="logout">
        <LogOut :size="18" />
      </button>
    </header>

    <aside class="admin-sidebar">
      <div class="panel-heading">
        <ShieldCheck :size="18" />
        <span>Admin</span>
      </div>
      <div class="profile-card">
        <strong>{{ user.name }}</strong>
        <span>{{ user.company_name || 'MergeOS' }}</span>
        <small>{{ user.email }}</small>
      </div>
      <div class="runtime-card">
        <div>
          <span>Payment</span>
          <strong>{{ adminSummary?.payment_mode || runtimeConfig?.payment_mode || 'loading' }}</strong>
        </div>
        <div>
          <span>Repo</span>
          <strong>{{ adminSummary?.repo_provider || runtimeConfig?.repo_provider || 'loading' }}</strong>
        </div>
        <div>
          <span>Email</span>
          <strong>{{ adminSummary?.smtp_ready ? 'smtp' : 'log' }}</strong>
        </div>
        <div>
          <span>Token</span>
          <strong>{{ tokenSymbol }}</strong>
        </div>
      </div>
      <div class="project-list compact-list">
        <div class="panel-heading">
          <FolderKanban :size="18" />
          <span>Projects</span>
        </div>
        <button
          v-for="project in projects"
          :key="project.id"
          :class="['project-row', { selected: adminCurrentProject?.id === project.id }]"
          @click="selectAdminProject(project)"
        >
          <span>
            <strong>{{ project.title }}</strong>
            <small>{{ project.client_email }}</small>
          </span>
          <b>{{ money(project.budget_cents) }}</b>
        </button>
      </div>
    </aside>

    <main class="portal-main admin-main">
      <section class="summary-strip">
        <article>
          <span>Funded</span>
          <strong>{{ money(adminSummary?.total_budget_cents) }}</strong>
        </article>
        <article>
          <span>Work pool</span>
          <strong>{{ money(adminSummary?.work_pool_cents) }}</strong>
        </article>
        <article>
          <span>Open tasks</span>
          <strong>{{ adminSummary?.open_task_count || 0 }}</strong>
        </article>
        <article>
          <span>Clients</span>
          <strong>{{ adminSummary?.client_count || 0 }}</strong>
        </article>
      </section>

      <section v-if="adminTab === 'overview'" class="admin-board">
        <div class="checkout-panel">
          <div class="panel-heading">
            <LayoutDashboard :size="18" />
            <span>Operations</span>
          </div>
          <div class="admin-metric-grid">
            <article>
              <span>Projects</span>
              <strong>{{ adminSummary?.project_count || 0 }}</strong>
            </article>
            <article>
              <span>Paid tasks</span>
              <strong>{{ adminSummary?.accepted_task_count || 0 }}</strong>
            </article>
            <article>
              <span>Fees</span>
              <strong>{{ money(adminSummary?.platform_fee_cents) }}</strong>
            </article>
            <article>
              <span>Paid out</span>
              <strong>{{ money(adminSummary?.paid_task_cents) }}</strong>
            </article>
            <article>
              <span>Users</span>
              <strong>{{ adminSummary?.user_count || 0 }}</strong>
            </article>
            <article>
              <span>Files</span>
              <strong>{{ adminSummary?.attachment_count || 0 }}</strong>
            </article>
          </div>
        </div>

        <div class="checkout-panel ssl-panel">
          <div class="panel-heading action-heading">
            <ShieldCheck :size="18" />
            <span>SSL review</span>
            <button class="panel-action" :disabled="sslReviewBusy" title="Review SSL now" @click="reviewSSL">
              <RefreshCw :size="15" />
              <span>{{ sslReviewBusy ? 'Checking...' : 'Review' }}</span>
            </button>
          </div>
          <div v-if="sslReviews.length" class="ssl-domain-list">
            <article v-for="review in sslReviews" :key="review.domain" class="ssl-domain-row">
              <div class="ssl-domain-head">
                <strong>{{ review.domain }}</strong>
                <span :class="['ssl-state', review.status]">{{ sslStatusLabel(review.status) }}</span>
              </div>
              <div class="ssl-facts">
                <span>Expires</span>
                <strong>{{ sslDaysText(review) }}</strong>
                <span>Issuer</span>
                <strong>{{ review.issuer || 'n/a' }}</strong>
                <span>Valid until</span>
                <strong>{{ formatDate(review.not_after) }}</strong>
                <span>Last check</span>
                <strong>{{ formatDate(review.last_checked_at) }}</strong>
              </div>
              <p v-if="review.error" class="ssl-error">{{ review.error }}</p>
            </article>
          </div>
          <p v-else class="muted-line">No SSL domains configured.</p>
        </div>

        <div class="project-list">
          <div class="panel-heading">
            <CheckCircle2 :size="18" />
            <span>Open task queue</span>
          </div>
          <button
            v-for="task in adminOpenTasks"
            :key="task.id"
            :class="['task-row', { selected: adminSelectedTask?.id === task.id }]"
            @click="selectAdminTask(task)"
          >
            <span :class="['status-dot', task.status]" />
            <span>{{ task.title }}</span>
            <strong>{{ money(task.reward_cents) }}</strong>
          </button>
        </div>
      </section>

      <section v-if="adminTab === 'projects'" class="admin-board">
        <div class="project-list">
          <div class="panel-heading">
            <FolderKanban :size="18" />
            <span>Funded projects</span>
          </div>
          <button
            v-for="project in projects"
            :key="project.id"
            :class="['admin-project-row', { selected: adminCurrentProject?.id === project.id }]"
            @click="selectAdminProject(project)"
          >
            <span>
              <strong>{{ project.title }}</strong>
              <small>{{ project.client_name }} / {{ project.client_email }}</small>
            </span>
            <span>{{ project.payment_provider }}</span>
            <b>{{ money(project.budget_cents) }}</b>
          </button>
        </div>

        <div class="checkout-panel">
          <div class="panel-heading">
            <SplitSquareVertical :size="18" />
            <span>Project detail</span>
          </div>
          <div v-if="adminCurrentProject" class="admin-detail">
            <p class="eyebrow">{{ adminCurrentProject.site_type }} / {{ adminCurrentProject.timeline }}</p>
            <h2>{{ adminCurrentProject.title }}</h2>
            <div class="manifest-grid">
              <span>Client</span>
              <strong>{{ adminCurrentProject.client_name }}</strong>
              <span>Company</span>
              <strong>{{ adminCurrentProject.company_name || 'n/a' }}</strong>
              <span>Budget</span>
              <strong>{{ money(adminCurrentProject.budget_cents) }}</strong>
              <span>Work pool</span>
              <strong>{{ money(adminCurrentProject.work_pool_cents) }}</strong>
              <span>Files</span>
              <strong>{{ attachmentCountForProject(adminCurrentProject.id) }}</strong>
              <span>Created</span>
              <strong>{{ formatDate(adminCurrentProject.created_at) }}</strong>
            </div>
            <a v-if="adminCurrentProject.repo_url" class="approval-link" :href="adminCurrentProject.repo_url" target="_blank" rel="noreferrer">
              <ExternalLink :size="16" />
              <span>Open repo</span>
            </a>
            <div class="task-list">
              <button
                v-for="task in adminProjectTasks"
                :key="task.id"
                :class="['task-row', { selected: adminSelectedTask?.id === task.id }]"
                @click="selectAdminTask(task)"
              >
                <span :class="['status-dot', task.status]" />
                <span>{{ task.title }}</span>
                <strong>{{ money(task.reward_cents) }}</strong>
              </button>
            </div>
          </div>
        </div>
      </section>

      <section v-if="adminTab === 'users'" class="checkout-panel">
        <div class="panel-heading">
          <UserRound :size="18" />
          <span>Users</span>
        </div>
        <div class="admin-table">
          <div class="admin-table-head">
            <span>User</span>
            <span>Role</span>
            <span>Projects</span>
            <span>Funded</span>
            <span>Last login</span>
          </div>
          <div v-for="row in adminUsers" :key="row.id" class="admin-table-row">
            <span>
              <strong>{{ row.name }}</strong>
              <small>{{ row.email }}</small>
            </span>
            <span>{{ row.role }}</span>
            <span>{{ row.project_count }}</span>
            <span>{{ money(row.total_budget_cents) }}</span>
            <span>{{ formatDate(row.last_login_at) }}</span>
          </div>
        </div>
      </section>

      <section v-if="adminTab === 'ledger'" class="checkout-panel">
        <div class="panel-heading">
          <WalletCards :size="18" />
          <span>Ledger</span>
        </div>
        <div class="admin-table ledger-table">
          <div class="admin-table-head">
            <span>#</span>
            <span>Type</span>
            <span>Amount</span>
            <span>Reference</span>
            <span>Hash</span>
          </div>
          <div v-for="entry in adminLedgerRows" :key="entry.sequence" class="admin-table-row">
            <span>{{ entry.sequence }}</span>
            <span>{{ entry.type }}</span>
            <span>{{ money(entry.amount_cents) }}</span>
            <span>{{ entry.reference }}</span>
            <span>{{ shortHash(entry.entry_hash) }}</span>
          </div>
        </div>
      </section>

      <section v-if="adminTab === 'inbox'" class="inbox-grid">
        <div class="email-list">
          <div class="panel-heading">
            <Mail :size="18" />
            <span>Customer emails</span>
          </div>
          <article v-for="note in notifications" :key="note.id" class="email-card">
            <span>{{ note.status }}</span>
            <strong>{{ note.subject }}</strong>
            <p>{{ note.body }}</p>
          </article>
        </div>
      </section>
    </main>

    <aside class="inspector admin-inspector">
      <div class="panel-heading">
        <SplitSquareVertical :size="18" />
        <span>Task control</span>
      </div>
      <div v-if="adminSelectedTask" class="task-inspector">
        <p class="eyebrow">{{ adminSelectedTask.status }} / {{ adminSelectedTask.required_worker_kind }}</p>
        <h3>{{ adminSelectedTask.title }}</h3>
        <p>{{ adminSelectedTask.acceptance }}</p>
        <a v-if="adminSelectedTask.issue_url" :href="adminSelectedTask.issue_url" target="_blank" rel="noreferrer">
          <ExternalLink :size="16" />
          <span>Open issue</span>
        </a>
        <div class="manifest-grid">
          <span>Project</span>
          <strong>{{ projectTitle(adminSelectedTask.project_id) }}</strong>
          <span>Worker</span>
          <strong>{{ adminSelectedTask.worker_id || 'pending' }}</strong>
          <span>Reward</span>
          <strong>{{ money(adminSelectedTask.reward_cents) }} {{ tokenSymbol }}</strong>
          <span>Proof</span>
          <strong>{{ adminSelectedTask.proof_hash ? shortHash(adminSelectedTask.proof_hash) : 'pending' }}</strong>
        </div>
        <label>
          Worker kind
          <select v-model="workerForm.worker_kind">
            <option value="human">Human</option>
            <option value="agent">Agent</option>
            <option value="hybrid">Hybrid</option>
          </select>
        </label>
        <label>
          Worker ID
          <input v-model="workerForm.worker_id" placeholder="github:alice or agent:web-001" />
        </label>
        <label>
          Agent type
          <input v-model="workerForm.agent_type" :disabled="workerForm.worker_kind === 'human'" placeholder="frontend-agent" />
        </label>
        <button class="primary-button" :disabled="adminSelectedTask.status === 'accepted' || accepting" @click="acceptAdminSelectedTask">
          <CheckCircle2 :size="17" />
          <span>{{ adminSelectedTask.status === 'accepted' ? 'Paid' : 'Accept and pay' }}</span>
        </button>
        <p v-if="errorMessage" class="error-line">{{ errorMessage }}</p>
      </div>
    </aside>
  </div>

  <div v-else class="app-shell">
    <header class="topbar">
      <div class="brand-lockup compact">
        <div class="brand-mark">
          <PanelsTopLeft :size="23" />
        </div>
        <div>
          <p class="eyebrow">Private client workspace</p>
          <h1>MergeOS</h1>
        </div>
      </div>
      <nav class="top-tabs">
        <button :class="{ active: portalTab === 'workspace' }" @click="portalTab = 'workspace'">
          <LayoutDashboard :size="16" />
          <span>Workspace</span>
        </button>
        <button :class="{ active: portalTab === 'billing' }" @click="portalTab = 'billing'">
          <WalletCards :size="16" />
          <span>Billing</span>
        </button>
        <button :class="{ active: portalTab === 'inbox' }" @click="portalTab = 'inbox'">
          <Mail :size="16" />
          <span>Email</span>
        </button>
      </nav>
      <div class="topbar-spacer" />
      <div class="status-pill">
        <ShieldCheck :size="16" />
        <span>{{ statusLabel }}</span>
      </div>
      <button class="icon-button" title="Refresh workspace" @click="refreshAll">
        <RefreshCw :size="18" />
      </button>
      <button class="icon-button" title="Logout" @click="logout">
        <LogOut :size="18" />
      </button>
    </header>

    <aside class="customer-panel">
      <div class="panel-heading">
        <UserRound :size="18" />
        <span>Customer</span>
      </div>

      <div class="profile-card">
        <strong>{{ user.name }}</strong>
        <span>{{ user.company_name || 'Independent client' }}</span>
        <small>{{ user.email }}</small>
      </div>

      <div class="runtime-card">
        <div>
          <span>Payment</span>
          <strong>{{ runtimeConfig?.payment_mode || 'loading' }}</strong>
        </div>
        <div>
          <span>Repo</span>
          <strong>{{ runtimeConfig?.repo_provider || 'loading' }}</strong>
        </div>
        <div>
          <span>Email</span>
          <strong>{{ runtimeConfig?.smtp_ready ? 'smtp' : 'log' }}</strong>
        </div>
        <div>
          <span>Token</span>
          <strong>{{ tokenSymbol }}</strong>
        </div>
      </div>

      <label>
        Contact name
        <input v-model="projectForm.client_name" />
      </label>
      <label>
        Contact email
        <input v-model="projectForm.client_email" type="email" />
      </label>
      <label>
        Phone
        <input v-model="projectForm.phone" />
      </label>
      <label>
        Company
        <input v-model="projectForm.company_name" />
      </label>
    </aside>

    <main class="portal-main">
      <section class="summary-strip">
        <article>
          <span>Total funded</span>
          <strong>{{ money(totalBudget) }}</strong>
        </article>
        <article>
          <span>MERGE reserved</span>
          <strong>{{ money(totalPool) }}</strong>
        </article>
        <article>
          <span>Open tasks</span>
          <strong>{{ openTasks.length }}</strong>
        </article>
        <article>
          <span>Paid tasks</span>
          <strong>{{ acceptedTasks.length }}</strong>
        </article>
      </section>

      <section v-if="portalTab === 'workspace'" class="workspace-grid">
        <div class="canvas-column">
          <div class="canvas-toolbar">
            <div>
              <p class="eyebrow">Website order</p>
              <h2>{{ currentProject?.title || 'Create a funded website project' }}</h2>
            </div>
            <div class="toolbar-metrics">
              <span>{{ currentTasks.length }} issues</span>
              <span>{{ currentProject?.repo_provider || runtimeConfig?.repo_provider || 'local-git' }}</span>
            </div>
          </div>

          <section class="builder-canvas">
            <div class="canvas-section hero-section">
              <div class="section-handle">BRIEF</div>
              <div>
                <p class="eyebrow">{{ projectForm.site_type }} / {{ projectForm.timeline }}</p>
                <h3>{{ currentProject?.company_name || projectForm.company_name }} delivery room</h3>
                <p>{{ currentProject?.brief || projectForm.brief }}</p>
              </div>
              <div class="quote-block">
                <span>{{ currentProject?.payment_provider || 'checkout pending' }}</span>
                <strong>{{ money(currentProject?.budget_cents || projectForm.budget_cents) }}</strong>
              </div>
            </div>

            <div v-if="currentProject?.attachments?.length" class="attachment-preview">
              <button
                v-for="attachment in currentProject.attachments"
                :key="attachment.id"
                type="button"
                class="attachment-chip"
                @click="openAttachment(attachment)"
              >
                <FileImage v-if="attachment.is_image" :size="18" />
                <Paperclip v-else :size="18" />
                <span>{{ attachment.original_name }}</span>
              </button>
            </div>

            <div v-if="currentTasks.length" class="canvas-grid">
              <button
                v-for="task in currentTasks"
                :key="task.id"
                :class="['task-tile', { selected: selectedTask?.id === task.id, accepted: task.status === 'accepted' }]"
                @click="selectTask(task)"
              >
                <span class="issue-label">Issue #{{ task.issue_number }}</span>
                <strong>{{ task.title }}</strong>
                <small>{{ task.required_worker_kind }} / {{ money(task.reward_cents) }} {{ tokenSymbol }}</small>
              </button>
            </div>

            <div v-else class="empty-canvas">
              <Database :size="30" />
              <strong>No funded bounty yet</strong>
              <span>{{ runtimeConfig?.dev_payment_enabled ? `Use ${runtimeConfig.dev_payment_code} as the local payment reference.` : 'Configure PayPal or crypto credentials first.' }}</span>
            </div>
          </section>

          <section class="repo-strip">
            <div class="strip-header">
              <GitBranch :size="18" />
              <a v-if="currentProject?.repo_url" :href="currentProject.repo_url" target="_blank" rel="noreferrer">
                {{ currentProject.bounty_repo_name }}
              </a>
              <span v-else>mergeos-bounties repo pending</span>
            </div>
            <div class="task-list">
              <button
                v-for="task in currentTasks"
                :key="task.id"
                :class="['task-row', { selected: selectedTask?.id === task.id }]"
                @click="selectTask(task)"
              >
                <span :class="['status-dot', task.status]" />
                <span>{{ task.title }}</span>
                <strong>{{ money(task.reward_cents) }}</strong>
              </button>
            </div>
          </section>
        </div>

        <aside class="order-panel">
          <div class="panel-heading">
            <FilePlus2 :size="18" />
            <span>New project</span>
          </div>
          <label>
            Project name
            <input v-model="projectForm.title" />
          </label>
          <label>
            Site type
            <select v-model="projectForm.site_type">
              <option>Landing page</option>
              <option>Business website</option>
              <option>SaaS website</option>
              <option>Storefront</option>
              <option>Web app shell</option>
            </select>
          </label>
          <label>
            Package
            <select v-model="projectForm.package_tier">
              <option>Launch</option>
              <option>Growth</option>
              <option>Scale</option>
            </select>
          </label>
          <label>
            Timeline
            <select v-model="projectForm.timeline">
              <option>7 days</option>
              <option>14 days</option>
              <option>30 days</option>
            </select>
          </label>
          <label>
            Budget USD
            <input v-model.number="budgetUsd" min="100" type="number" />
          </label>
          <label>
            Brief
            <textarea v-model="projectForm.brief" rows="5" />
          </label>
          <label class="upload-control">
            Reference files
            <input type="file" multiple @change="uploadProjectFiles" />
            <span class="upload-surface">
              <UploadCloud :size="18" />
              <span>{{ uploadBusy ? 'Uploading...' : 'Add images or files' }}</span>
            </span>
          </label>
          <div v-if="uploadedAttachments.length" class="attachment-list pending">
            <div v-for="attachment in uploadedAttachments" :key="attachment.id" class="attachment-row">
              <FileImage v-if="attachment.is_image" :size="17" />
              <Paperclip v-else :size="17" />
              <span>
                <strong>{{ attachment.original_name }}</strong>
                <small>{{ attachment.content_type }} / {{ fileSize(attachment.size_bytes) }}</small>
              </span>
              <button class="remove-attachment" title="Remove file" @click="removeUploadedAttachment(attachment.id)">
                <X :size="15" />
              </button>
            </div>
          </div>
          <button class="primary-button" :disabled="creating || uploadBusy" @click="createProject">
            <CreditCard :size="17" />
            <span>{{ creating ? 'Funding...' : 'Verify payment and create repo' }}</span>
          </button>
          <p v-if="errorMessage" class="error-line">{{ errorMessage }}</p>
        </aside>
      </section>

      <section v-if="portalTab === 'billing'" class="billing-grid">
        <div class="checkout-panel">
          <div class="panel-heading">
            <WalletCards :size="18" />
            <span>Checkout</span>
          </div>
          <div class="payment-choice">
            <button :class="{ active: projectForm.payment_method === 'paypal' }" @click="projectForm.payment_method = 'paypal'">PayPal</button>
            <button :class="{ active: projectForm.payment_method === 'crypto' }" @click="projectForm.payment_method = 'crypto'">Crypto</button>
          </div>
          <button
            v-if="projectForm.payment_method === 'paypal'"
            class="secondary-button"
            :disabled="!runtimeConfig?.paypal_ready || preparingPayPal"
            @click="preparePayPalOrder"
          >
            <WalletCards :size="17" />
            <span>{{ preparingPayPal ? 'Creating order...' : 'Create PayPal order' }}</span>
          </button>
          <a v-if="paypalOrder?.approval_url" class="approval-link" :href="paypalOrder.approval_url" target="_blank" rel="noreferrer">
            <ExternalLink :size="16" />
            <span>Open PayPal approval</span>
          </a>
          <label>
            Payment reference
            <input v-model="projectForm.payment_reference" :placeholder="paymentReferencePlaceholder" />
          </label>
          <div v-if="projectForm.payment_method === 'crypto'" class="receiver-card">
            <span>Receiver</span>
            <strong>{{ runtimeConfig?.crypto_receiver || 'Configure CRYPTO_RECEIVER in backend env' }}</strong>
          </div>
          <div class="billing-ledger">
            <div v-for="entry in recentLedger" :key="entry.sequence" class="ledger-line">
              <span>#{{ entry.sequence }} {{ entry.type }}</span>
              <strong>{{ money(entry.amount_cents) }}</strong>
            </div>
          </div>
        </div>

        <div class="project-list">
          <div class="panel-heading">
            <FolderKanban :size="18" />
            <span>Funded projects</span>
          </div>
          <button
            v-for="project in projects"
            :key="project.id"
            :class="['project-row', { selected: currentProject?.id === project.id }]"
            @click="selectProject(project)"
          >
            <span>
              <strong>{{ project.title }}</strong>
              <small>{{ project.bounty_repo_name }}</small>
            </span>
            <b>{{ money(project.budget_cents) }}</b>
          </button>
        </div>
      </section>

      <section v-if="portalTab === 'inbox'" class="inbox-grid">
        <div class="email-list">
          <div class="panel-heading">
            <Mail :size="18" />
            <span>Customer emails</span>
          </div>
          <article v-for="note in notifications" :key="note.id" class="email-card">
            <span>{{ note.status }}</span>
            <strong>{{ note.subject }}</strong>
            <p>{{ note.body }}</p>
          </article>
        </div>
      </section>
    </main>

    <aside class="inspector">
      <div class="panel-heading">
        <SplitSquareVertical :size="18" />
        <span>Task inspector</span>
      </div>

      <div class="repo-summary">
        <p class="eyebrow">Child bounty repo</p>
        <h3>{{ currentProject?.bounty_repo_name || 'Not created' }}</h3>
        <p>{{ currentProject?.repo_provider || runtimeConfig?.repo_provider || 'local-git' }}</p>
        <a v-if="currentProject?.repo_url" :href="currentProject.repo_url" target="_blank" rel="noreferrer">
          <ExternalLink :size="16" />
          <span>Open repo</span>
        </a>
        <div v-if="currentProject?.attachments?.length" class="repo-attachments">
          <button
            v-for="attachment in currentProject.attachments"
            :key="attachment.id"
            type="button"
            @click="openAttachment(attachment)"
          >
            <FileImage v-if="attachment.is_image" :size="16" />
            <Paperclip v-else :size="16" />
            <span>{{ attachment.original_name }}</span>
          </button>
        </div>
      </div>

      <div v-if="selectedTask" class="task-inspector">
        <p class="eyebrow">Selected issue</p>
        <h3>{{ selectedTask.title }}</h3>
        <p>{{ selectedTask.acceptance }}</p>
        <a v-if="selectedTask.issue_url" :href="selectedTask.issue_url" target="_blank" rel="noreferrer">
          <ExternalLink :size="16" />
          <span>Open issue</span>
        </a>

        <div class="manifest-grid">
          <span>Required</span>
          <strong>{{ selectedTask.required_worker_kind }}</strong>
          <span>Suggested</span>
          <strong>{{ selectedTask.suggested_agent_type || 'human-review' }}</strong>
          <span>Reward</span>
          <strong>{{ money(selectedTask.reward_cents) }} {{ tokenSymbol }}</strong>
          <span>Proof</span>
          <strong>{{ selectedTask.proof_hash ? shortHash(selectedTask.proof_hash) : 'pending' }}</strong>
        </div>

        <label>
          Worker kind
          <select v-model="workerForm.worker_kind">
            <option value="human">Human</option>
            <option value="agent">Agent</option>
            <option value="hybrid">Hybrid</option>
          </select>
        </label>
        <label>
          Worker ID
          <input v-model="workerForm.worker_id" placeholder="github:alice or agent:web-001" />
        </label>
        <label>
          Agent type
          <input v-model="workerForm.agent_type" :disabled="workerForm.worker_kind === 'human'" placeholder="frontend-agent" />
        </label>
        <button class="primary-button" :disabled="selectedTask.status === 'accepted' || accepting" @click="acceptSelectedTask">
          <CheckCircle2 :size="17" />
          <span>{{ selectedTask.status === 'accepted' ? 'Paid' : 'Accept and pay' }}</span>
        </button>
      </div>
    </aside>
  </div>
</template>

<script setup>
import { computed, onMounted, reactive, ref, watch } from 'vue';
import {
  CheckCircle2,
  CreditCard,
  Database,
  ExternalLink,
  FileImage,
  FilePlus2,
  FolderKanban,
  GitBranch,
  LayoutDashboard,
  LogIn,
  LogOut,
  Mail,
  PanelsTopLeft,
  Paperclip,
  RefreshCw,
  ShieldCheck,
  SplitSquareVertical,
  UploadCloud,
  UserRound,
  WalletCards,
  X,
} from '@lucide/vue/dist/esm/lucide-vue.mjs';

const runtimeConfig = ref(null);
const user = ref(null);
const authMode = ref('register');
const hasWindow = typeof window !== 'undefined';
const token = ref(hasWindow ? localStorage.getItem('mergeos_token') || '' : '');
const authBusy = ref(false);
const projects = ref([]);
const tasks = ref([]);
const ledger = ref([]);
const notifications = ref([]);
const adminSummary = ref(null);
const adminUsers = ref([]);
const attachments = ref([]);
const sslReviews = ref([]);
const selectedProjectId = ref('');
const selectedTask = ref(null);
const selectedAdminProjectId = ref('');
const adminSelectedTask = ref(null);
const portalTab = ref('workspace');
const adminTab = ref('overview');
const creating = ref(false);
const accepting = ref(false);
const preparingPayPal = ref(false);
const uploadBusy = ref(false);
const sslReviewBusy = ref(false);
const errorMessage = ref('');
const paypalOrder = ref(null);
const uploadedAttachments = ref([]);

const authForm = reactive({
  name: 'Thanh Truc Client',
  company_name: 'Thanh Truc Solutions',
  email: 'client@mergeos.local',
  password: 'mergeos123',
});

const projectForm = reactive({
  client_name: 'Thanh Truc Client',
  company_name: 'Thanh Truc Solutions',
  client_email: 'client@mergeos.local',
  phone: '+84 900 000 000',
  title: 'Elementor-style company website',
  site_type: 'Business website',
  package_tier: 'Growth',
  timeline: '14 days',
  brief: 'Build a polished responsive website with services, portfolio, lead form, payment-ready checkout, and customer dashboard preview.',
  budget_cents: 240000,
  payment_method: 'paypal',
  payment_reference: 'LOCAL-PAID',
});

const workerForm = reactive({
  worker_kind: 'agent',
  worker_id: 'agent:mergeos-web-001',
  agent_type: 'frontend-agent',
});

const isAdmin = computed(() => user.value?.role === 'admin');
const currentProject = computed(() => {
  if (selectedProjectId.value) {
    return projects.value.find((project) => project.id === selectedProjectId.value) || projects.value[projects.value.length - 1] || null;
  }
  return projects.value[projects.value.length - 1] || null;
});
const currentTasks = computed(() => {
  if (!currentProject.value) return [];
  return tasks.value.filter((task) => task.project_id === currentProject.value.id);
});
const acceptedTasks = computed(() => tasks.value.filter((task) => task.status === 'accepted'));
const openTasks = computed(() => tasks.value.filter((task) => task.status !== 'accepted'));
const recentLedger = computed(() => ledger.value.slice(-8).reverse());
const adminCurrentProject = computed(() => {
  if (selectedAdminProjectId.value) {
    return projects.value.find((project) => project.id === selectedAdminProjectId.value) || projects.value[0] || null;
  }
  return projects.value[0] || null;
});
const adminProjectTasks = computed(() => {
  if (!adminCurrentProject.value) return [];
  return tasks.value.filter((task) => task.project_id === adminCurrentProject.value.id);
});
const adminOpenTasks = computed(() => tasks.value.filter((task) => task.status !== 'accepted'));
const adminLedgerRows = computed(() => ledger.value.slice().reverse());
const totalBudget = computed(() => projects.value.reduce((sum, project) => sum + project.budget_cents, 0));
const totalPool = computed(() => projects.value.reduce((sum, project) => sum + project.work_pool_cents, 0));
const tokenSymbol = computed(() => runtimeConfig.value?.token_symbol || 'MERGE');
const statusLabel = computed(() => {
  if (currentProject.value) return `${currentProject.value.payment_provider} verified`;
  return runtimeConfig.value?.repo_provider || 'ready';
});
const paymentReferencePlaceholder = computed(() => {
  if (projectForm.payment_method === 'paypal') return 'PayPal order id';
  if (runtimeConfig.value?.crypto_asset === 'erc20') return 'EVM tx hash for stablecoin transfer';
  return 'EVM tx hash';
});
const budgetUsd = computed({
  get: () => Math.round(projectForm.budget_cents / 100),
  set: (value) => {
    projectForm.budget_cents = Math.max(100, Number(value || 100)) * 100;
  },
});

function money(cents) {
  return new Intl.NumberFormat('en-US', {
    style: 'currency',
    currency: 'USD',
    maximumFractionDigits: 0,
  }).format((cents || 0) / 100);
}

function fileSize(bytes) {
  if (!bytes) return '0 B';
  const units = ['B', 'KB', 'MB', 'GB'];
  const power = Math.min(Math.floor(Math.log(bytes) / Math.log(1024)), units.length - 1);
  const value = bytes / (1024 ** power);
  return `${value.toFixed(value >= 10 || power === 0 ? 0 : 1)} ${units[power]}`;
}

function formatDate(value) {
  if (!value) return 'n/a';
  return new Intl.DateTimeFormat('en-US', {
    dateStyle: 'medium',
    timeStyle: 'short',
  }).format(new Date(value));
}

function sslStatusLabel(status) {
  const labels = {
    ok: 'Valid',
    warning: 'Expiring',
    expired: 'Expired',
    error: 'Issue',
    pending: 'Pending',
  };
  return labels[status] || status || 'Pending';
}

function sslDaysText(review) {
  if (!review?.not_after) return 'waiting';
  const days = Number(review.days_remaining || 0);
  if (days < 0) return `${Math.abs(days)} days expired`;
  if (days === 0) return 'expires today';
  return `${days} days left`;
}

function projectTitle(projectId) {
  return projects.value.find((project) => project.id === projectId)?.title || projectId;
}

function attachmentCountForProject(projectId) {
  return attachments.value.filter((attachment) => attachment.project_id === projectId).length;
}

function shortHash(hash) {
  return `${hash.slice(0, 8)}...${hash.slice(-6)}`;
}

async function api(path, options = {}) {
  const isFormData = typeof FormData !== 'undefined' && options.body instanceof FormData;
  const response = await fetch(path, {
    ...options,
    headers: {
      ...(isFormData ? {} : { 'Content-Type': 'application/json' }),
      ...(token.value ? { Authorization: `Bearer ${token.value}` } : {}),
      ...(options.headers || {}),
    },
  });
  const payload = await response.json();
  if (!response.ok) {
    if (response.status === 401) clearSession();
    throw new Error(payload.error || 'Request failed');
  }
  return payload;
}

async function loadConfig() {
  runtimeConfig.value = await api('/api/config');
  if (runtimeConfig.value.dev_payment_enabled && !projectForm.payment_reference) {
    projectForm.payment_reference = runtimeConfig.value.dev_payment_code;
  }
}

async function submitAuth() {
  authBusy.value = true;
  errorMessage.value = '';
  try {
    const path = authMode.value === 'register' ? '/api/auth/register' : '/api/auth/login';
    const body = authMode.value === 'register'
      ? authForm
      : { email: authForm.email, password: authForm.password };
    const auth = await api(path, { method: 'POST', body: JSON.stringify(body) });
    setSession(auth);
    if (!isAdmin.value) syncProjectContact();
    await refreshProtected();
  } catch (error) {
    errorMessage.value = error.message;
  } finally {
    authBusy.value = false;
  }
}

function setSession(auth) {
  token.value = auth.token;
  user.value = auth.user;
  if (hasWindow) localStorage.setItem('mergeos_token', auth.token);
}

function clearSession() {
  token.value = '';
  user.value = null;
  projects.value = [];
  tasks.value = [];
  ledger.value = [];
  notifications.value = [];
  adminSummary.value = null;
  adminUsers.value = [];
  attachments.value = [];
  sslReviews.value = [];
  selectedAdminProjectId.value = '';
  adminSelectedTask.value = null;
  uploadedAttachments.value = [];
  if (hasWindow) localStorage.removeItem('mergeos_token');
}

function syncProjectContact() {
  if (!user.value) return;
  projectForm.client_name = user.value.name;
  projectForm.company_name = user.value.company_name;
  projectForm.client_email = user.value.email;
}

async function restoreSession() {
  if (!token.value) return;
  try {
    user.value = await api('/api/auth/me');
    if (!isAdmin.value) syncProjectContact();
    await refreshProtected();
  } catch {
    clearSession();
  }
}

async function refreshAll() {
  await loadConfig();
  if (user.value) {
    await refreshProtected();
  }
}

async function refreshProtected() {
  if (isAdmin.value) {
    const [summary, userRows, projectRows, taskRows, ledgerRows, noteRows, attachmentRows, sslRows] = await Promise.all([
      api('/api/admin/summary'),
      api('/api/admin/users'),
      api('/api/admin/projects'),
      api('/api/admin/tasks'),
      api('/api/admin/ledger'),
      api('/api/admin/notifications'),
      api('/api/admin/attachments'),
      api('/api/admin/ssl'),
    ]);
    adminSummary.value = summary;
    adminUsers.value = userRows;
    projects.value = projectRows;
    tasks.value = taskRows;
    ledger.value = ledgerRows;
    notifications.value = noteRows.slice().reverse();
    attachments.value = attachmentRows;
    sslReviews.value = sslRows.length ? sslRows : (summary.ssl_reviews || []);
    if (!selectedAdminProjectId.value && projectRows.length) {
      selectedAdminProjectId.value = projectRows[0].id;
    }
    reconcileAdminSelection();
    return;
  }

  const [projectRows, taskRows, ledgerRows, noteRows] = await Promise.all([
    api('/api/projects'),
    api('/api/tasks'),
    api('/api/ledger'),
    api('/api/notifications'),
  ]);
  projects.value = projectRows;
  tasks.value = taskRows;
  ledger.value = ledgerRows;
  notifications.value = noteRows.slice().reverse();
  if (!selectedProjectId.value && projectRows.length) {
    selectedProjectId.value = projectRows[projectRows.length - 1].id;
  }
  reconcileSelection();
}

async function reviewSSL() {
  sslReviewBusy.value = true;
  errorMessage.value = '';
  try {
    const rows = await api('/api/admin/ssl/review', { method: 'POST' });
    sslReviews.value = rows;
    if (adminSummary.value) {
      adminSummary.value = { ...adminSummary.value, ssl_reviews: rows };
    }
  } catch (error) {
    errorMessage.value = error.message;
  } finally {
    sslReviewBusy.value = false;
  }
}

async function preparePayPalOrder() {
  preparingPayPal.value = true;
  errorMessage.value = '';
  try {
    const order = await api('/api/payments/paypal/orders', {
      method: 'POST',
      body: JSON.stringify({
        amount_cents: projectForm.budget_cents,
        description: projectForm.title,
        return_url: hasWindow ? window.location.href : 'http://127.0.0.1:5173',
        cancel_url: hasWindow ? window.location.href : 'http://127.0.0.1:5173',
      }),
    });
    paypalOrder.value = order;
    projectForm.payment_reference = order.order_id;
  } catch (error) {
    errorMessage.value = error.message;
  } finally {
    preparingPayPal.value = false;
  }
}

async function createProject() {
  creating.value = true;
  errorMessage.value = '';
  try {
    const project = await api('/api/projects', {
      method: 'POST',
      body: JSON.stringify({
        ...projectForm,
        attachment_ids: uploadedAttachments.value.map((attachment) => attachment.id),
      }),
    });
    selectedProjectId.value = project.id;
    uploadedAttachments.value = [];
    await refreshProtected();
    portalTab.value = 'workspace';
  } catch (error) {
    errorMessage.value = error.message;
  } finally {
    creating.value = false;
  }
}

async function uploadProjectFiles(event) {
  const input = event.target;
  const files = Array.from(input.files || []);
  if (!files.length) return;

  uploadBusy.value = true;
  errorMessage.value = '';
  try {
    const formData = new FormData();
    files.forEach((file) => formData.append('files', file));
    const attachments = await api('/api/uploads', {
      method: 'POST',
      body: formData,
    });
    uploadedAttachments.value = uploadedAttachments.value.concat(attachments);
  } catch (error) {
    errorMessage.value = error.message;
  } finally {
    uploadBusy.value = false;
    input.value = '';
  }
}

function removeUploadedAttachment(id) {
  uploadedAttachments.value = uploadedAttachments.value.filter((attachment) => attachment.id !== id);
}

async function openAttachment(attachment) {
  errorMessage.value = '';
  try {
    const response = await fetch(attachment.url, {
      headers: {
        ...(token.value ? { Authorization: `Bearer ${token.value}` } : {}),
      },
    });
    if (!response.ok) {
      throw new Error('Could not open file');
    }
    const blob = await response.blob();
    const blobURL = URL.createObjectURL(blob);
    if (hasWindow) {
      const opened = window.open(blobURL, '_blank', 'noopener,noreferrer');
      if (!opened) {
        const link = document.createElement('a');
        link.href = blobURL;
        link.download = attachment.original_name || 'attachment';
        link.click();
      }
      window.setTimeout(() => URL.revokeObjectURL(blobURL), 60000);
    }
  } catch (error) {
    errorMessage.value = error.message;
  }
}

function selectAdminProject(project) {
  selectedAdminProjectId.value = project.id;
  reconcileAdminSelection();
}

function selectAdminTask(task) {
  adminSelectedTask.value = task;
  workerForm.worker_kind = task.required_worker_kind;
  workerForm.worker_id = task.required_worker_kind === 'human' ? 'github:admin-reviewer' : 'agent:mergeos-admin-001';
  workerForm.agent_type = task.required_worker_kind === 'human' ? '' : (task.suggested_agent_type || 'custom-agent');
  errorMessage.value = '';
}

async function acceptAdminSelectedTask() {
  if (!adminSelectedTask.value) return;
  accepting.value = true;
  errorMessage.value = '';
  try {
    await api(`/api/tasks/${adminSelectedTask.value.id}/accept`, {
      method: 'POST',
      body: JSON.stringify(workerForm),
    });
    await refreshProtected();
  } catch (error) {
    errorMessage.value = error.message;
  } finally {
    accepting.value = false;
  }
}

function selectProject(project) {
  selectedProjectId.value = project.id;
  reconcileSelection();
}

function selectTask(task) {
  selectedTask.value = task;
  workerForm.worker_kind = task.required_worker_kind;
  workerForm.worker_id = task.required_worker_kind === 'human' ? 'github:client-reviewer' : 'agent:mergeos-web-001';
  workerForm.agent_type = task.required_worker_kind === 'human' ? '' : (task.suggested_agent_type || 'custom-agent');
  errorMessage.value = '';
}

async function acceptSelectedTask() {
  if (!selectedTask.value) return;
  accepting.value = true;
  errorMessage.value = '';
  try {
    await api(`/api/tasks/${selectedTask.value.id}/accept`, {
      method: 'POST',
      body: JSON.stringify(workerForm),
    });
    await refreshProtected();
  } catch (error) {
    errorMessage.value = error.message;
  } finally {
    accepting.value = false;
  }
}

async function logout() {
  try {
    await api('/api/auth/logout', { method: 'POST', body: JSON.stringify({}) });
  } finally {
    clearSession();
  }
}

function reconcileSelection() {
  if (selectedTask.value) {
    const fresh = currentTasks.value.find((task) => task.id === selectedTask.value.id);
    selectedTask.value = fresh || currentTasks.value[0] || null;
    return;
  }
  if (currentTasks.value.length) {
    selectTask(currentTasks.value[0]);
  }
}

function reconcileAdminSelection() {
  if (adminSelectedTask.value) {
    const fresh = tasks.value.find((task) => task.id === adminSelectedTask.value.id);
    adminSelectedTask.value = fresh || adminProjectTasks.value[0] || adminOpenTasks.value[0] || null;
    return;
  }
  if (adminProjectTasks.value.length) {
    selectAdminTask(adminProjectTasks.value[0]);
    return;
  }
  if (adminOpenTasks.value.length) {
    selectAdminTask(adminOpenTasks.value[0]);
  }
}

watch(currentTasks, reconcileSelection);
watch(adminProjectTasks, reconcileAdminSelection);

onMounted(async () => {
  await loadConfig();
  await restoreSession();
});
</script>
