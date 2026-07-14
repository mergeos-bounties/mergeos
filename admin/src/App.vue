<template>
  <main v-if="!isAuthenticated" class="login-screen">
    <section class="login-panel" aria-labelledby="admin-login-title">
      <div class="login-brand">
        <span class="brand-mark" aria-hidden="true"><img src="/favicon.svg" alt="" /></span>
        <div>
          <strong>MergeOS Admin</strong>
          <small>Dashboard</small>
        </div>
      </div>

      <div class="login-copy">
        <span class="eyebrow">ADMIN</span>
        <h1 id="admin-login-title">Control center</h1>
      </div>

      <form class="login-form" @submit.prevent="login">
        <label>
          <span>Email</span>
          <input v-model.trim="loginForm.email" autocomplete="email" type="email" />
        </label>
        <label>
          <span>Password</span>
          <input v-model="loginForm.password" autocomplete="current-password" type="password" />
        </label>
        <p v-if="authError" class="form-error">{{ authError }}</p>
        <button class="primary-action" :disabled="authBusy" type="submit">
          <LogIn :size="16" />
          {{ authBusy ? 'Signing in...' : 'Open admin' }}
        </button>
      </form>
    </section>

    <aside class="login-preview" aria-label="Admin workspace preview">
      <div class="preview-sidebar">
        <span v-for="item in navItems.slice(0, 6)" :key="item.id"></span>
      </div>
      <div class="preview-canvas">
        <div class="preview-toolbar"></div>
        <div class="preview-grid">
          <span></span>
          <span></span>
          <span></span>
          <span></span>
        </div>
      </div>
    </aside>
  </main>

  <div v-else class="admin-shell">
    <aside class="admin-sidebar">
      <a class="sidebar-brand" :href="routeForView('builder')" @click="navigateToView('builder', $event)">
        <span class="brand-mark" aria-hidden="true"><img src="/favicon.svg" alt="" /></span>
        <span>
          <strong>MergeOS</strong>
          <small>Admin Builder</small>
        </span>
      </a>

      <nav class="sidebar-nav" aria-label="Admin navigation">
        <a
          v-for="item in navItems"
          :key="item.id"
          :class="{ active: activeView === item.id }"
          :href="routeForView(item.id)"
          :aria-current="activeView === item.id ? 'page' : undefined"
          @click="navigateToView(item.id, $event)"
        >
          <component :is="item.icon" :size="17" />
          <span>{{ item.label }}</span>
        </a>
      </nav>

    </aside>

    <section class="admin-main">
      <header class="admin-topbar">
        <div>
          <span class="eyebrow">{{ activeNav?.kicker || 'WORKSPACE' }}</span>
          <h1>{{ activeNav?.title || 'Admin workspace' }}</h1>
        </div>
        <div class="topbar-actions">
          <label class="search-box">
            <Search :size="16" />
            <input v-model.trim="search" placeholder="Search projects, tasks, users" />
          </label>
          <div class="device-switch" role="group" aria-label="Canvas preview">
            <button
              v-for="device in devices"
              :key="device.id"
              :class="{ active: activeDevice === device.id }"
              type="button"
              @click="activeDevice = device.id"
            >
              <component :is="device.icon" :size="15" />
            </button>
          </div>
          <button class="icon-button" :disabled="loading" type="button" @click="loadAdminData" aria-label="Refresh">
            <RefreshCw :size="17" />
          </button>
          <button class="icon-button" type="button" @click="logout" aria-label="Log out">
            <LogOut :size="17" />
          </button>
        </div>
      </header>

      <section class="admin-command-strip" aria-label="Admin command summary">
        <div class="admin-command-copy">
          <span class="eyebrow">CONTROL ROOM</span>
          <h2>{{ adminCommandTitle }}</h2>
          <p>{{ adminCommandBody }}</p>
        </div>
        <div class="admin-command-metrics">
          <article v-for="metric in adminCommandMetrics" :key="metric.label">
            <span :class="['metric-icon', metric.tone]">
              <component :is="metric.icon" :size="18" />
            </span>
            <div>
              <strong>{{ metric.value }}</strong>
              <small>{{ metric.label }}</small>
            </div>
          </article>
        </div>
      </section>

      <p v-if="errorMessage" class="workspace-error">{{ errorMessage }}</p>

      <section v-if="activeView === 'builder'" class="builder-workspace">
        <div class="canvas-column">
          <div class="canvas-toolbar">
            <div>
              <span>Dashboard</span>
              <strong>{{ selectedWidgetLabel }}</strong>
            </div>
            <div class="canvas-tools">
              <button type="button"><MousePointer2 :size="15" /> Select</button>
              <button type="button"><Settings2 :size="15" /> Style</button>
              <button type="button"><Eye :size="15" /> Preview</button>
            </div>
          </div>

          <div :class="['elementor-canvas', activeDevice]">
            <section class="canvas-band metrics-band">
              <header>
                <span>Overview</span>
                <strong>Platform command center</strong>
              </header>
              <div class="metric-grid">
                <article v-for="metric in summaryMetrics" :key="metric.label">
                  <span :class="['metric-icon', metric.tone]">
                    <component :is="metric.icon" :size="18" />
                  </span>
                  <div>
                    <strong>{{ metric.value }}</strong>
                    <small>{{ metric.label }}</small>
                  </div>
                </article>
              </div>
            </section>

            <section class="canvas-band split-band">
              <article>
                <header>
                  <span>Projects</span>
                  <strong>Funded queue</strong>
                </header>
                <div class="stack-list">
                  <div v-for="project in filteredProjects.slice(0, 4)" :key="project.id">
                    <span>{{ initials(project.title) }}</span>
                    <div>
                      <strong>{{ project.title }}</strong>
                      <small>{{ mrgFromCents(project.budget_cents) }} escrow</small>
                    </div>
                  </div>
                </div>
              </article>

              <article>
                <header>
                  <span>Tasks</span>
                  <strong>Open work</strong>
                </header>
                <div class="stack-list">
                  <div v-for="task in filteredTasks.slice(0, 4)" :key="task.id">
                    <span>{{ task.issue_number || 'T' }}</span>
                    <div>
                      <strong>{{ task.title }}</strong>
                      <small>{{ task.status }} · {{ mrgFromCents(task.reward_cents) }}</small>
                    </div>
                  </div>
                </div>
              </article>
            </section>

            <section class="canvas-band ledger-band">
              <header>
                <span>Proof ledger</span>
                <strong>Latest verified records</strong>
              </header>
              <div class="ledger-stream">
                <article v-for="entry in ledgerEntries.slice().reverse().slice(0, 5)" :key="entry.sequence">
                  <span>{{ entry.sequence }}</span>
                  <div>
                    <strong>{{ titleize(entry.type) }}</strong>
                    <small>{{ mrgFromCents(entry.amount_cents) }} · {{ shortRef(entry.reference) }}</small>
                  </div>
                </article>
              </div>
            </section>
          </div>
        </div>

        <aside class="inspector-panel">
          <div class="inspector-head">
            <SlidersHorizontal :size="18" />
            <strong>Inspector</strong>
          </div>
          <label>
            <span>Widget</span>
            <select v-model="selectedWidget">
              <option v-for="widget in builderWidgets" :key="widget.id" :value="widget.id">{{ widget.label }}</option>
            </select>
          </label>
          <label>
            <span>Density</span>
            <input v-model="density" type="range" min="1" max="3" />
          </label>
          <div class="inspector-checks">
            <label><input v-model="showLedgerHashes" type="checkbox" /> Hash references</label>
            <label><input v-model="compactRows" type="checkbox" /> Compact rows</label>
          </div>
        </aside>
      </section>

      <section v-else-if="activeView === 'overview'" class="data-grid">
        <article v-for="metric in summaryMetrics" :key="metric.label" class="metric-tile">
          <span :class="['metric-icon', metric.tone]">
            <component :is="metric.icon" :size="19" />
          </span>
          <strong>{{ metric.value }}</strong>
          <small>{{ metric.label }}</small>
        </article>
      </section>

      <section v-else-if="activeView === 'projects'" class="table-panel">
        <TableHeader title="Projects" :count="filteredProjects.length" />
        <DataTable :columns="['Project', 'Client', 'Budget', 'Tasks', 'Status']">
          <tr v-for="project in filteredProjects" :key="project.id">
            <td><strong>{{ project.title }}</strong><small>{{ project.id }}</small></td>
            <td>{{ project.client_name || project.company_name || 'Client' }}</td>
            <td>{{ mrgFromCents(project.budget_cents) }}</td>
            <td>{{ project.tasks?.length || 0 }}</td>
            <td><span class="status-pill green">{{ project.status }}</span></td>
          </tr>
        </DataTable>
      </section>

      <section v-else-if="activeView === 'tasks'" class="task-review-panel">
        <header class="task-review-header">
          <div>
            <span class="eyebrow">REVIEW QUEUE</span>
            <h2>Task review board</h2>
            <p>Review GitHub issues by state and expand linked PRs only when needed.</p>
          </div>
          <div class="task-review-stats" aria-label="Task queue summary">
            <article>
              <strong>{{ number(filteredTasks.length) }}</strong>
              <small>Issues visible</small>
            </article>
            <article>
              <strong>{{ number(issueTabCounts.open) }}</strong>
              <small>Open issues</small>
            </article>
            <article>
              <strong>{{ number(issueTabCounts.closed) }}</strong>
              <small>Closed issues</small>
            </article>
          </div>
        </header>

        <div class="issue-state-tabs" role="tablist" aria-label="Issue state filter">
          <button
            v-for="tab in issueStateTabs"
            :key="tab.id"
            :aria-selected="taskIssueTab === tab.id"
            :class="{ active: taskIssueTab === tab.id }"
            role="tab"
            type="button"
            @click="taskIssueTab = tab.id"
          >
            {{ tab.label }}
            <span>{{ number(tab.count) }}</span>
          </button>
        </div>

        <div v-if="!filteredTasks.length" class="task-empty-state">
          <span class="metric-icon green"><CheckCircle2 :size="19" /></span>
          <strong>No {{ taskIssueTab }} issues found</strong>
          <small>Clear the search or switch tabs to see more GitHub issues.</small>
        </div>

        <div v-else class="task-review-list">
          <article v-for="task in filteredTasks" :key="task.id" class="task-review-item">
            <div class="task-review-main">
              <div class="task-review-title">
                <span class="task-issue-mark">{{ task.issue_number || 'T' }}</span>
                <div>
                  <div class="task-title-line">
                    <strong>{{ task.title }}</strong>
                    <span :class="['status-pill', issueStateForTask(task) === 'closed' ? 'amber' : 'green']">{{ issueStateForTask(task) }}</span>
                  </div>
                  <small>{{ taskIssueLabel(task) }} / {{ taskProjectTitle(task) }}</small>
                </div>
              </div>
              <div class="task-meta-row">
                <span>{{ task.required_worker_kind }}</span>
                <span>{{ task.suggested_agent_type || 'manual review' }}</span>
                <span>{{ task.status }}</span>
              </div>
            </div>

            <section class="task-pr-section" aria-label="Linked pull requests">
              <div class="task-pr-toolbar">
                <button
                  class="compact-action pr-toggle-action"
                  :aria-expanded="isTaskPullsExpanded(task)"
                  :disabled="taskPullsLoading[task.id]"
                  type="button"
                  @click="toggleTaskPulls(task)"
                >
                  <GitPullRequest :size="14" />
                  {{ taskPullsLoading[task.id] ? 'Checking...' : isTaskPullsExpanded(task) ? 'Hide PRs' : 'Show PRs' }}
                  <ChevronDown :class="['pr-chevron', { open: isTaskPullsExpanded(task) }]" :size="15" />
                </button>
                <button v-if="isTaskPullsExpanded(task)" class="compact-action" :disabled="taskPullsLoading[task.id]" type="button" @click="loadTaskPulls(task, true)">
                  <RefreshCw :size="14" />
                  Refresh
                </button>
                <small>{{ taskPullSummary(task) }}</small>
              </div>
              <div v-if="isTaskPullsExpanded(task)" class="task-pr-collapse">
                <p v-if="taskPullsError[task.id]" class="inline-error">{{ taskPullsError[task.id] }}</p>
                <p v-else-if="taskPullsLoaded[task.id] && !visiblePullsForTask(task).length" class="muted-inline">{{ emptyPullMessage(task) }}</p>
                <div v-else class="task-pr-list">
                  <article v-for="pull in visiblePullsForTask(task)" :key="pull.number" class="task-pr-row">
                    <div class="task-pr-main">
                      <span :class="['metric-icon', pull.merged ? 'green' : pull.draft ? 'amber' : 'blue']">
                        <GitPullRequest :size="16" />
                      </span>
                      <div>
                        <strong>#{{ pull.number }} {{ pull.title }}</strong>
                        <small>@{{ pull.author }} / {{ pullStatus(pull) }} / {{ pull.head_ref || 'head' }} -> {{ pull.base_ref || 'base' }}</small>
                        <em>Credit: github:{{ pull.author }} / {{ mrg(mergeSelection(task, pull).reward_mrg) }}</em>
                        <div class="task-pr-readiness">
                          <span :class="['status-pill', pullReadinessTone(pull)]">{{ pullReadinessLabel(pull) }}</span>
                          <span>{{ pullReadinessRisk(pull) }}</span>
                        </div>
                        <ul v-if="pullReadinessNotes(pull).length" class="task-pr-readiness-notes">
                          <li v-for="note in pullReadinessNotes(pull)" :key="note">{{ note }}</li>
                        </ul>
                        <ul v-if="pullReadinessSignals(pull).length" class="task-pr-readiness-signals">
                          <li v-for="signal in pullReadinessSignals(pull)" :key="signal">
                            <CheckCircle2 :size="12" /> {{ signal }}
                          </li>
                        </ul>
                      </div>
                    </div>
                    <div class="bounty-review-controls">
                      <label>
                        <span>Type</span>
                        <select :value="mergeSelection(task, pull).bounty_type" @change="setMergeBounty(task, pull, $event.target.value)">
                          <option v-for="option in bountyOptions" :key="option.id" :value="option.id">{{ option.label }}</option>
                        </select>
                      </label>
                      <label>
                        <span>MRG</span>
                        <input
                          :value="mergeSelection(task, pull).reward_mrg"
                          min="1"
                          step="1"
                          type="number"
                          @input="setMergeReward(task, pull, $event.target.value)"
                        />
                      </label>
                    </div>
                    <div class="task-pr-actions">
                      <a class="compact-action link-action" :href="pull.html_url" target="_blank" rel="noreferrer">View</a>
                      <button class="compact-action merge-action" :disabled="!canMergeTaskPull(task, pull)" type="button" @click="mergeTaskPull(task, pull)">
                        {{ mergeBusy[mergeKey(task, pull)] ? 'Merging...' : pull.merged ? 'Credit' : 'Merge' }}
                      </button>
                    </div>
                  </article>
                </div>
              </div>
              <p v-if="mergeMessages[task.id]" class="inline-success">{{ mergeMessages[task.id] }}</p>
              <aside v-if="mergeCreditReceipt" class="credit-receipt">
                <header class="credit-receipt-header">
                  <CheckCircle2 :size="16" />
                  <span>Credit receipt</span>
                </header>
                <dl class="credit-receipt-dl">
                  <div v-if="mergeCreditReceipt.sequence"><dt>Ledger seq</dt><dd>{{ mergeCreditReceipt.sequence }}</dd></div>
                  <div v-if="mergeCreditReceipt.entryHash"><dt>Proof hash</dt><dd class="hash-cell">{{ shortRef(mergeCreditReceipt.entryHash) }}</dd></div>
                  <div v-if="mergeCreditReceipt.creditUrl"><dt>Scan</dt><dd><a :href="mergeCreditReceipt.creditUrl" target="_blank" rel="noreferrer">View on scan <ExternalLink :size="12" /></a></dd></div>
                  <div v-if="mergeCreditReceipt.commentUrl"><dt>Comment</dt><dd><a :href="mergeCreditReceipt.commentUrl" target="_blank" rel="noreferrer">View on PR <ExternalLink :size="12" /></a></dd></div>
                </dl>
                <button class="compact-action" type="button" @click="copyText(buildMergeCreditComment(mergeCreditReceipt))">
                  <ClipboardCopy :size="14" />
                  Copy comment
                </button>
              </aside>
            </section>
          </article>
        </div>
      </section>

      <section v-else-if="activeView === 'ledger'" class="table-panel">
        <TableHeader title="Ledger" :count="ledgerEntries.length" />
        <form class="manual-credit-form" @submit.prevent="createManualCredit">
          <label>
            <span>Worker</span>
            <input v-model.trim="manualCreditForm.worker_id" autocomplete="off" placeholder="github:eliasx45" required />
          </label>
          <label>
            <span>Type</span>
            <select v-model.trim="manualCreditForm.bounty_type" autocomplete="off" @change="setManualCreditBounty">
              <option v-for="option in bountyOptions" :key="option.id" :value="option.id">{{ option.label }}</option>
            </select>
          </label>
          <label>
            <span>MRG</span>
            <input v-model.number="manualCreditForm.reward_mrg" min="1" step="1" type="number" required />
          </label>
          <label>
            <span>PR URL</span>
            <input v-model.trim="manualCreditForm.pr_url" autocomplete="off" placeholder="https://github.com/org/repo/pull/120" required />
          </label>
          <label>
            <span>PR title</span>
            <input v-model.trim="manualCreditForm.pr_title" autocomplete="off" placeholder="Timeline correction" />
          </label>
          <button class="primary-action" :disabled="manualCreditBusy" type="submit">
            <CircleDollarSign :size="16" />
            {{ manualCreditBusy ? 'Crediting...' : 'Credit MRG' }}
          </button>
          <p v-if="manualCreditError" class="form-error">{{ manualCreditError }}</p>
          <p v-if="manualCreditMessage" class="form-success">{{ manualCreditMessage }}</p>
          <aside v-if="manualCreditReceipt" class="credit-receipt">
            <header class="credit-receipt-header">
              <CheckCircle2 :size="16" />
              <span>Credit receipt</span>
            </header>
            <dl class="credit-receipt-dl">
              <div v-if="manualCreditReceipt.sequence"><dt>Ledger seq</dt><dd>{{ manualCreditReceipt.sequence }}</dd></div>
              <div v-if="manualCreditReceipt.entryHash"><dt>Proof hash</dt><dd class="hash-cell">{{ shortRef(manualCreditReceipt.entryHash) }}</dd></div>
              <div v-if="manualCreditReceipt.creditUrl"><dt>Scan</dt><dd><a :href="manualCreditReceipt.creditUrl" target="_blank" rel="noreferrer">View on scan <ExternalLink :size="12" /></a></dd></div>
            </dl>
            <button class="compact-action" type="button" @click="copyText(buildManualCreditComment(manualCreditReceipt))">
              <ClipboardCopy :size="14" />
              Copy comment
            </button>
          </aside>
        </form>
        <DataTable :columns="['Seq', 'Type', 'From', 'To', 'Amount', 'Reference']">
          <tr v-for="entry in ledgerEntries.slice().reverse()" :key="entry.sequence">
            <td>{{ entry.sequence }}</td>
            <td><strong>{{ titleize(entry.type) }}</strong></td>
            <td>{{ entry.from_account || '-' }}</td>
            <td>{{ entry.to_account || '-' }}</td>
            <td>{{ mrgFromCents(entry.amount_cents) }}</td>
            <td :title="showLedgerHashes ? entry.entry_hash : entry.reference">{{ showLedgerHashes ? shortRef(entry.entry_hash) : shortRef(entry.reference) }}</td>
          </tr>
        </DataTable>
      </section>

      <section v-else-if="activeView === 'ops'" class="ops-workspace">
        <section class="ops-summary-grid" aria-label="Operations queue summary">
          <article v-for="metric in opsQueueMetrics" :key="metric.label">
            <span :class="['metric-icon', metric.tone]">
              <component :is="metric.icon" :size="18" />
            </span>
            <div>
              <strong>{{ metric.value }}</strong>
              <small>{{ metric.label }}</small>
            </div>
          </article>
        </section>

        <section class="ops-queue-panel">
          <header class="task-review-header">
            <div>
              <span class="eyebrow">OPS QUEUE</span>
              <h2>Disputes, fraud, moderation, and payouts attention</h2>
              <p>Review closed unpaid issues, payout fraud signals, delivery failures, AI webhook failures, SSL security items, and manual credit audit rows.</p>
            </div>
            <button class="compact-action" :disabled="loading" type="button" @click="loadAdminData">
              <RefreshCw :size="14" />
              Refresh
            </button>
          </header>

          <p v-if="opsActionError" class="form-error">{{ opsActionError }}</p>
          <p v-else-if="opsActionMessage" class="form-success">{{ opsActionMessage }}</p>

          <div v-if="!filteredOpsQueueItems.length" class="task-empty-state">
            <span class="metric-icon green"><CheckCircle2 :size="19" /></span>
            <strong>No ops queue items</strong>
            <small>Disputes, fraud signals, moderation alerts, and payout review rows will appear here when they need admin attention.</small>
          </div>

          <div v-else class="ops-queue-list">
            <article v-for="item in filteredOpsQueueItems" :key="item.id" class="ops-queue-item">
              <span :class="['metric-icon', opsSeverityTone(item.severity)]">
                <component :is="opsIcon(item.type)" :size="17" />
              </span>
              <div class="ops-queue-main">
                <div class="ops-title-line">
                  <strong>{{ item.title }}</strong>
                  <span :class="['status-pill', opsSeverityTone(item.severity)]">{{ item.severity }}</span>
                  <span class="status-pill blue">{{ opsTypeLabel(item.type) }}</span>
                </div>
                <p>{{ item.body }}</p>
                <small>
                  {{ item.project_title || item.reference || 'MergeOS ops' }}
                  <template v-if="item.issue_number"> / Issue #{{ item.issue_number }}</template>
                  / {{ formatDate(item.created_at) }}
                </small>
              </div>
              <div class="ops-queue-side">
                <span>{{ titleize(item.status || 'needs_review') }}</span>
                <div v-if="opsActionsForItem(item).length" class="ops-action-list">
                  <button
                    v-for="action in opsActionsForItem(item)"
                    :key="action.key"
                    class="compact-action"
                    :class="{ 'link-action': action.type === 'open_url' }"
                    :disabled="opsActionBusyFor(item, action)"
                    type="button"
                    @click="handleOpsQueueAction(item, action)"
                  >
                    <component :is="opsActionIcon(action)" :size="13" />
                    {{ action.label }}
                  </button>
                </div>
                <small>{{ shortRef(opsReference(item)) }}</small>
              </div>
            </article>
          </div>
        </section>
      </section>

      <section v-else-if="activeView === 'users'" class="users-workspace">
        <section class="table-panel users-table-panel">
          <TableHeader title="Users" :count="filteredUsers.length" />
          <DataTable :columns="['User', 'Role', 'Company', 'Projects', 'Total Budget', 'Last Login', '']">
            <tr
              v-for="row in filteredUsers"
              :key="row.id"
              :class="{ selected: selectedUserId === row.id }"
              @click="openUserEditor(row)"
            >
              <td><strong>{{ row.name || row.email }}</strong><small>{{ row.email }}</small></td>
              <td><span :class="['status-pill', row.role === 'admin' ? 'blue' : 'green']">{{ row.role }}</span></td>
              <td>{{ row.company_name || '-' }}</td>
              <td>{{ row.project_count || 0 }}</td>
              <td>{{ mrgFromCents(row.total_budget_cents) }}</td>
              <td>{{ formatDate(row.last_login_at) }}</td>
              <td class="row-action">
                <button class="compact-action" type="button" @click.stop="openUserEditor(row)">
                  <UserCog :size="15" />
                  Edit
                </button>
              </td>
            </tr>
          </DataTable>
        </section>

        <aside class="user-editor-panel">
          <div class="editor-head">
            <span class="metric-icon blue"><UserCog :size="19" /></span>
            <div>
              <span class="eyebrow">USER</span>
              <h2>{{ selectedUser ? 'Edit account' : 'Select a user' }}</h2>
            </div>
          </div>

          <form v-if="selectedUser" class="editor-form" @submit.prevent="saveSelectedUser">
            <section class="form-section">
              <div class="form-section-head">
                <span>Profile</span>
                <span :class="['status-pill', userForm.role === 'admin' ? 'blue' : 'green']">{{ userForm.role }}</span>
              </div>
              <label>
                <span>Name</span>
                <input v-model.trim="userForm.name" autocomplete="name" />
              </label>
              <label>
                <span>Email</span>
                <input v-model.trim="userForm.email" autocomplete="email" type="email" />
              </label>
              <label>
                <span>Company</span>
                <input v-model.trim="userForm.company_name" autocomplete="organization" />
              </label>
              <label>
                <span>Role</span>
                <select v-model="userForm.role">
                  <option value="client">Client</option>
                  <option value="admin">Admin</option>
                </select>
              </label>
            </section>

            <section class="form-section">
              <div class="form-section-head">
                <span>Password</span>
                <KeyRound :size="16" />
              </div>
              <label>
                <span>New password</span>
                <input v-model="userForm.password" autocomplete="new-password" type="password" />
              </label>
              <label>
                <span>Confirm password</span>
                <input v-model="userForm.password_confirm" autocomplete="new-password" type="password" />
              </label>
            </section>

            <p v-if="userEditorError" class="form-error">{{ userEditorError }}</p>
            <p v-if="userEditorMessage" class="form-success">{{ userEditorMessage }}</p>
            <button class="primary-action" :disabled="userEditorBusy" type="submit">
              <Save :size="16" />
              {{ userEditorBusy ? 'Saving...' : 'Save user' }}
            </button>
          </form>
        </aside>
      </section>

      <section v-else-if="activeView === 'ssl'" class="ssl-workspace">
        <section class="ssl-review-panel">
          <div>
            <span class="eyebrow">SECURITY</span>
            <h2>SSL certificate review</h2>
          </div>
          <div class="ssl-status-grid">
            <article>
              <strong>{{ sslRows.length }}</strong>
              <small>Domains</small>
            </article>
            <article>
              <strong>{{ sslOkCount }}</strong>
              <small>Healthy</small>
            </article>
            <article>
              <strong>{{ sslAttentionCount }}</strong>
              <small>Attention</small>
            </article>
          </div>
          <button class="primary-action" :disabled="sslReviewBusy" type="button" @click="reviewSSLNow">
            <ShieldCheck :size="16" />
            {{ sslReviewBusy ? 'Reviewing...' : 'Review SSL now' }}
          </button>
          <p v-if="sslReviewError" class="form-error">{{ sslReviewError }}</p>
          <p v-if="sslReviewMessage" class="form-success">{{ sslReviewMessage }}</p>
        </section>

        <section class="table-panel">
          <TableHeader title="SSL review" :count="sslRows.length" />
          <DataTable :columns="['Domain', 'Status', 'Issuer', 'Days', 'Checked', 'Next Check']">
            <tr v-for="row in sslRows" :key="row.domain">
              <td><strong>{{ row.domain }}</strong><small>{{ row.port || '443' }}</small></td>
              <td><span :class="['status-pill', row.status === 'ok' ? 'green' : 'amber']">{{ row.status || 'pending' }}</span></td>
              <td>{{ row.issuer || '-' }}</td>
              <td>{{ row.days_remaining }}</td>
              <td>{{ formatDate(row.last_checked_at) }}</td>
              <td>{{ formatDate(row.next_check_at) }}</td>
            </tr>
          </DataTable>
        </section>
      </section>

      <section v-else-if="activeView === 'setting'" class="settings-workspace">
        <section class="settings-panel">
          <div class="settings-panel-head">
            <span class="metric-icon purple"><Settings2 :size="19" /></span>
            <div>
              <span class="eyebrow">AI MODEL</span>
              <h2>Review LLM</h2>
              <p>Used by the automated reviewer for PR and issue automation.</p>
            </div>
          </div>

          <form class="settings-form" @submit.prevent="saveAdminSettings">
            <label>
              <span>Provider</span>
              <select v-model.trim="settingsForm.llm_provider" autocomplete="off" @change="syncSelectedProviderModel">
                <option v-for="provider in llmProviderOptions" :key="provider.id" :value="provider.id">{{ provider.label }}</option>
              </select>
            </label>
            <label>
              <span>Model</span>
              <select
                v-model.trim="settingsForm.llm_model"
                autocomplete="off"
              >
                <option v-for="model in settingsModelOptions" :key="model" :value="model">{{ model }}</option>
              </select>
            </label>
            <button class="primary-action" :disabled="settingsBusy" type="submit">
              <Save :size="16" />
              {{ settingsBusy ? 'Saving...' : 'Save model' }}
            </button>
          </form>

          <p v-if="settingsError" class="form-error">{{ settingsError }}</p>
          <p v-if="settingsMessage" class="form-success">{{ settingsMessage }}</p>
        </section>

        <section class="settings-summary-grid" aria-label="Current model settings">
          <article>
            <span>Provider</span>
            <strong>{{ providerLabel(adminSettings.llm_provider || 'gemini') }}</strong>
          </article>
          <article>
            <span>Current model</span>
            <strong>{{ adminSettings.llm_model || adminSettings.gemini_review_model || 'gemini-2.5-flash' }}</strong>
          </article>
          <article>
            <span>Updated</span>
            <strong>{{ formatDate(adminSettings.updated_at) }}</strong>
          </article>
          <article>
            <span>Available presets</span>
            <strong>{{ number(settingsModelOptions.length) }}</strong>
          </article>
        </section>

        <section class="gemini-control-panel">
          <div class="gemini-panel-head">
            <span class="metric-icon purple"><KeyRound :size="19" /></span>
            <div>
              <span class="eyebrow">LLM</span>
              <h2>API tokens</h2>
            </div>
          </div>

          <form class="gemini-key-form" @submit.prevent="addGeminiKey">
            <label>
              <span>Provider</span>
              <select v-model.trim="geminiKeyForm.provider" autocomplete="off" @change="syncKeyProviderModel">
                <option v-for="provider in llmProviderOptions" :key="provider.id" :value="provider.id">{{ provider.label }}</option>
              </select>
            </label>
            <label>
              <span>Default model</span>
              <select v-model.trim="geminiKeyForm.model" autocomplete="off">
                <option v-for="model in keyModelOptions" :key="model" :value="model">{{ model }}</option>
              </select>
            </label>
            <label>
              <span>Token</span>
              <input v-model.trim="geminiKeyForm.key_value" autocomplete="off" placeholder="Paste provider API token" type="password" />
            </label>
            <button class="primary-action" :disabled="geminiKeyBusy" type="submit">
              <Save :size="16" />
              {{ geminiKeyBusy ? 'Adding...' : 'Add token' }}
            </button>
          </form>

          <p v-if="geminiKeyError" class="form-error">{{ geminiKeyError }}</p>
          <p v-if="geminiKeyMessage" class="form-success">{{ geminiKeyMessage }}</p>

          <div class="gemini-status-grid">
            <article>
              <strong>{{ number(geminiKeys.length) }}</strong>
              <small>Total keys</small>
            </article>
            <article>
              <strong>{{ number(geminiActiveCount) }}</strong>
              <small>Runnable</small>
            </article>
            <article>
              <strong>{{ number(geminiAttentionCount) }}</strong>
              <small>Attention</small>
            </article>
          </div>
        </section>

        <section class="table-panel">
          <TableHeader title="LLM API tokens" :count="geminiKeys.length" />
          <DataTable :columns="['Token', 'Provider', 'Model', 'Status', 'Requests', 'Success', 'Quota', 'Last used', 'Actions']">
            <tr v-for="row in geminiKeys" :key="row.id">
              <td><strong>{{ row.key_hint }}</strong><small>{{ row.id }}</small></td>
              <td><strong>{{ providerLabel(row.provider || 'gemini') }}</strong><small>{{ row.provider || 'gemini' }}</small></td>
              <td><strong>{{ row.model || modelFallbackForProvider(row.provider || 'gemini') }}</strong></td>
              <td><span :class="['status-pill', geminiKeyStatusTone(row.status)]">{{ titleize(row.status || 'active') }}</span></td>
              <td>{{ number(row.request_count) }}</td>
              <td>{{ number(row.success_count) }}</td>
              <td>{{ number(row.quota_error_count) }}</td>
              <td>
                <strong>{{ formatDate(row.last_used_at) }}</strong>
                <small v-if="geminiTestResults[row.id]" :class="['gemini-test-result', geminiTestResults[row.id].ok ? 'ok' : 'bad']">
                  {{ geminiTestResults[row.id].message }}
                </small>
                <small v-else>{{ row.last_error || 'No recent error' }}</small>
              </td>
              <td class="row-action">
                <button class="compact-action" :disabled="geminiActionBusy[row.id] || geminiTestBusy[row.id]" type="button" @click="testGeminiKey(row)">
                  <CheckCircle2 :size="14" />
                  {{ geminiTestBusy[row.id] ? 'Testing...' : 'Test' }}
                </button>
                <button class="compact-action" :disabled="geminiActionBusy[row.id] || geminiTestBusy[row.id]" type="button" @click="setGeminiKeyStatus(row, row.status === 'disabled' ? 'active' : 'disabled')">
                  <Power :size="14" />
                  {{ row.status === 'disabled' ? 'Enable' : 'Disable' }}
                </button>
                <button class="compact-action" :disabled="geminiActionBusy[row.id] || geminiTestBusy[row.id]" type="button" @click="resetGeminiKey(row)">
                  <RefreshCw :size="14" />
                  Reset
                </button>
              </td>
            </tr>
          </DataTable>
        </section>
      </section>

      <section v-else-if="activeView === 'logs'" class="logs-workspace">
        <section class="table-panel">
          <header class="table-header">
            <div>
              <span>Events</span>
              <h2>Log</h2>
            </div>
            <button class="compact-action" :disabled="loading" type="button" @click="loadGeminiAdminData">
              <RefreshCw :size="14" />
              Refresh
            </button>
          </header>
          <DataTable :columns="['Received', 'Event', 'Repository', 'Status', 'Key', 'Result']">
            <tr v-for="row in geminiWebhookLogs" :key="row.id">
              <td><strong>{{ formatDate(row.received_at) }}</strong><small>{{ row.duration_millis || 0 }} ms</small></td>
              <td><strong>{{ row.event_name || '-' }}</strong><small>{{ [row.action, row.delivery_id].filter(Boolean).join(' / ') || '-' }}</small></td>
              <td><strong>{{ row.repository || '-' }}</strong><small>{{ row.pull_number ? `PR #${row.pull_number}` : row.sender || '-' }}</small></td>
              <td><span :class="['status-pill', geminiWebhookStatusTone(row.status)]">{{ titleize(row.status || 'received') }}</span></td>
              <td>{{ shortRef(row.key_id) }}</td>
              <td>
                <a v-if="row.comment_url" class="inline-link" :href="row.comment_url" target="_blank" rel="noreferrer">Comment</a>
                <strong v-else>{{ row.error ? 'Error' : 'No comment' }}</strong>
                <small>{{ row.error || (row.labels || []).join(', ') || `HTTP ${row.status_code || 0}` }}</small>
              </td>
            </tr>
          </DataTable>
        </section>
      </section>
    </section>
  </div>
      <section v-if="activeView === 'test-settings'" class="settings-workspace">
        <section class="settings-panel">
          <div class="settings-panel-head">
            <span class="metric-icon purple"><Settings2 :size="19" /></span>
            <div>
              <span class="eyebrow">TEST MODE</span>
              <h2>Publish Settings</h2>
              <p>Enable public test mode for bounty contributors. Contributors can configure integration keys without editing .env files.</p>
            </div>
          </div>

          <form class="settings-form" @submit.prevent="saveTestSettingsConfig">
            <div class="test-mode-toggle">
              <label class="toggle-label">
                <span>Test mode</span>
                <span :class="['toggle-switch', testSettingsConfig.test_mode_enabled ? 'on' : 'off']" @click="toggleTestMode">
                  <span class="toggle-knob"></span>
                </span>
                <strong>{{ testSettingsConfig.test_mode_enabled ? 'Enabled' : 'Disabled' }}</strong>
              </label>
              <p class="toggle-hint">When enabled, contributors with the password can access the public Publish Settings page.</p>
            </div>

            <label>
              <span>New password</span>
              <input v-model="testNewPassword" autocomplete="off" placeholder="Set a new public test password" type="password" />
            </label>

            <button class="primary-action" :disabled="testSettingsBusy" type="submit">
              <Save :size="16" />
              {{ testSettingsBusy ? 'Saving...' : 'Save settings' }}
            </button>
          </form>

          <p v-if="testSettingsError" class="form-error">{{ testSettingsError }}</p>
          <p v-if="testSettingsMessage" class="form-success">{{ testSettingsMessage }}</p>
        </section>

        <section class="settings-summary-grid" aria-label="Test mode status">
          <article><span>Status</span><strong :style="{color: testSettingsConfig.test_mode_enabled ? '#22c55e' : '#f59e0b'}">{{ testSettingsConfig.test_mode_enabled ? 'ONLINE' : 'OFFLINE' }}</strong></article>
          <article><span>Entries</span><strong>{{ testSettingsEntries.length }}</strong></article>
        </section>

        <section class="gemini-control-panel">
          <div class="gemini-panel-head">
            <span class="metric-icon purple"><KeyRound :size="19" /></span>
            <div>
              <span class="eyebrow">INTEGRATION KEYS</span>
              <h2>Test entries</h2>
              <p>Add LLM test keys, PayPal Sandbox credentials, and USDT test receiver settings.</p>
            </div>
          </div>

          <form class="gemini-key-form" @submit.prevent="addTestEntry">
            <label><span>Type</span>
              <select v-model="testForm.integration_type">
                <option value="llm">LLM</option>
                <option value="paypal">PayPal Sandbox</option>
                <option value="usdt">USDT</option>
              </select>
            </label>
            <label><span>Name</span><input v-model.trim="testForm.display_name" placeholder="e.g. OpenAI Test" /></label>
            <label><span>Key</span><input v-model.trim="testForm.setting_key" placeholder="e.g. llm_test_openai_key" /></label>
            <label><span>Value</span><input v-model="testForm.setting_value" placeholder="Paste key" type="password" /></label>
            <label><span>Metadata JSON</span><input v-model="testForm.key_value_json" placeholder='{"env":"sandbox"}' /></label>
            <button class="primary-action" :disabled="testSettingsBusy" type="submit"><Save :size="16" /> Add entry</button>
          </form>
        </section>

        <section class="table-panel">
          <TableHeader title="Test settings entries" :count="testSettingsEntries.length" />
          <DataTable :columns="['Key', 'Type', 'Name', 'Value (masked)', 'Status', 'Created', 'Actions']">
            <tr v-for="entry in testSettingsEntries" :key="entry.id">
              <td><strong>{{ entry.setting_key }}</strong><small>{{ entry.id }}</small></td>
              <td><span class="status-pill blue">{{ entry.integration_type }}</span></td>
              <td>{{ entry.display_name || '-' }}</td>
              <td><code>{{ entry.setting_value_hint }}</code></td>
              <td><span :class="['status-pill', entry.status === 'active' ? 'green' : 'amber']">{{ entry.status }}</span></td>
              <td>{{ formatDate(entry.created_at) }}</td>
              <td class="row-action">
                <button class="compact-action" type="button" @click="deleteTestEntry(entry.id)"><span style="color:#ef4444;">Delete</span></button>
              </td>
            </tr>
          </DataTable>
        </section>
      </section>

</template>

<script setup>
import { computed, defineComponent, h, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue';
import {
  Activity,
  AlertTriangle,
  BarChart3,
  CheckCircle2,
  ChevronDown,
  CircleDollarSign,
  ClipboardCopy,
  Columns3,
  Eye,
  ExternalLink,
  FolderKanban,
  GitPullRequest,
  KeyRound,
  LayoutDashboard,
  ListChecks,
  LogIn,
  LogOut,
  Monitor,
  MousePointer2,
  PanelLeft,
  Power,
  RefreshCw,
  Search,
  Settings2,
  Save,
  ShieldCheck,
  SlidersHorizontal,
  Smartphone,
  Tablet,
  UserCog,
  UsersRound,
} from '@lucide/vue';

const storageKey = 'mergeos_admin_token';
const hasWindow = typeof window !== 'undefined';

const token = ref(hasWindow ? localStorage.getItem(storageKey) || '' : '');
const adminUser = ref(null);
const activeView = ref('builder');
const selectedWidget = ref('metrics');
const activeDevice = ref('desktop');
const search = ref('');
const loading = ref(false);
const authBusy = ref(false);
const authError = ref('');
const errorMessage = ref('');
const userEditorBusy = ref(false);
const userEditorError = ref('');
const userEditorMessage = ref('');
const sslReviewBusy = ref(false);
const sslReviewError = ref('');
const sslReviewMessage = ref('');
const opsActionBusy = ref({});
const opsActionMessage = ref('');
const opsActionError = ref('');
const density = ref(2);
const showLedgerHashes = ref(false);
const compactRows = ref(true);
const selectedUserId = ref('');
const taskIssueTab = ref('open');

const summary = ref({});
const users = ref([]);
const projects = ref([]);
const tasks = ref([]);
const notifications = ref([]);
const ledgerEntries = ref([]);
const opsQueue = ref({ stats: {}, items: [] });
const sslRows = ref([]);
const taskPulls = ref({});
const taskPullsLoaded = ref({});
const taskPullsLoading = ref({});
const taskPullsError = ref({});
const mergeBusy = ref({});
const mergeMessages = ref({});
const mergeSelections = ref({});
const manualCreditBusy = ref(false);
const manualCreditError = ref('');
const manualCreditMessage = ref('');
const mergeCreditReceipt = ref(null);
const manualCreditReceipt = ref(null);
const taskIssueStates = ref({});
const expandedTaskPulls = ref({});
const geminiKeys = ref([]);
const geminiWebhookLogs = ref([]);
const geminiKeyBusy = ref(false);
const geminiKeyError = ref('');
const geminiKeyMessage = ref('');
const geminiActionBusy = ref({});
const geminiTestBusy = ref({});
const geminiTestResults = ref({});
const adminSettings = ref({});
const settingsBusy = ref(false);
const settingsError = ref('');
const settingsMessage = ref('');

// Test publish settings
const testSettingsConfig = ref({});
const testSettingsEntries = ref([]);
const testSettingsBusy = ref(false);
const testSettingsError = ref('');
const testSettingsMessage = ref('');
const testNewPassword = ref('');
const testForm = reactive({
  integration_type: 'llm',
  display_name: '',
  setting_key: '',
  setting_value: '',
  key_value_json: '{}',
});

const loginForm = reactive({
  email: 'admin@gmail.com',
  password: '[test-admin-pass]',
});

const userForm = reactive({
  id: '',
  name: '',
  company_name: '',
  email: '',
  role: 'client',
  password: '',
  password_confirm: '',
});

const geminiKeyForm = reactive({
  provider: 'gemini',
  model: 'gemini-2.5-flash',
  key_value: '',
});

const settingsForm = reactive({
  llm_provider: 'gemini',
  llm_model: 'gemini-2.5-flash',
  gemini_review_model: '',
});

const manualCreditForm = reactive({
  worker_id: '',
  bounty_type: 'future-medium',
  reward_mrg: 50,
  pr_url: '',
  pr_title: '',
});

const navItems = [
  { id: 'builder', label: 'Dashboard', title: 'Dashboard', kicker: 'DASHBOARD', icon: PanelLeft },
  { id: 'overview', label: 'Overview', title: 'Platform overview', kicker: 'DASHBOARD', icon: LayoutDashboard },
  { id: 'projects', label: 'Projects', title: 'Funded projects', kicker: 'PROJECTS', icon: FolderKanban },
  { id: 'tasks', label: 'Tasks', title: 'Task operations', kicker: 'TASKS', icon: ListChecks },
  { id: 'ledger', label: 'Treasury', title: 'Treasury ledger', kicker: 'TREASURY', icon: Activity },
  { id: 'ops', label: 'Ops Queue', title: 'Operations queue', kicker: 'OPS', icon: AlertTriangle },
  { id: 'users', label: 'Users', title: 'User management', kicker: 'USERS', icon: UsersRound },
  { id: 'ssl', label: 'SSL', title: 'SSL monitoring', kicker: 'SECURITY', icon: ShieldCheck },
  { id: 'setting', label: 'Setting', title: 'Settings', kicker: 'SYSTEM', icon: Settings2 },
  { id: 'logs', label: 'Log', title: 'Log', kicker: 'AUTOMATION', icon: KeyRound },
  { id: 'test-settings', label: 'Test Settings', title: 'Test Publish Settings', kicker: 'TEST MODE', icon: Settings2 },
];

const routeByView = {
  builder: '/',
  overview: '/overview',
  projects: '/projects',
  tasks: '/tasks',
  ledger: '/ledger',
  ops: '/ops',
  users: '/users',
  ssl: '/ssl',
  setting: '/setting',
  logs: '/logs',
  'test-settings': '/test-settings',
};
const viewByRoute = Object.entries(routeByView).reduce((routes, [view, route]) => {
  routes[route] = view;
  return routes;
}, {});
viewByRoute['/gemini'] = 'logs';
viewByRoute['/test-settings'] = 'test-settings';

const bountyOptions = [
  { id: 'future-small', label: 'Future small', reward_mrg: 25 },
  { id: 'future-medium', label: 'Future medium', reward_mrg: 50 },
  { id: 'bug-large', label: 'Bug bounty large', reward_mrg: 100 },
  { id: 'major-feature', label: 'Major feature', reward_mrg: 200 },
];

const builderWidgets = [
  { id: 'metrics', label: 'Metric Counter', icon: BarChart3 },
  { id: 'project-list', label: 'Project Queue', icon: FolderKanban },
  { id: 'task-board', label: 'Task Kanban', icon: Columns3 },
  { id: 'ledger-stream', label: 'Ledger Stream', icon: Activity },
  { id: 'ssl-monitor', label: 'SSL Monitor', icon: ShieldCheck },
];

const devices = [
  { id: 'desktop', icon: Monitor },
  { id: 'tablet', icon: Tablet },
  { id: 'mobile', icon: Smartphone },
];

const activeNav = computed(() => navItems.find((item) => item.id === activeView.value));
const selectedWidgetLabel = computed(() => builderWidgets.find((widget) => widget.id === selectedWidget.value)?.label || 'Widget');
const isAuthenticated = computed(() => Boolean(token.value && adminUser.value));
const query = computed(() => search.value.toLowerCase());
const selectedUser = computed(() => users.value.find((row) => row.id === selectedUserId.value) || null);
const sslOkCount = computed(() => sslRows.value.filter((row) => row.status === 'ok').length);
const sslAttentionCount = computed(() => sslRows.value.length - sslOkCount.value);
const tokenSymbol = computed(() => summary.value.token_symbol || 'MRG');
const geminiActiveCount = computed(() => geminiKeys.value.filter((row) => row.status === 'active').length);
const geminiAttentionCount = computed(() => geminiKeys.value.filter((row) => ['quota_limited', 'error', 'disabled'].includes(row.status)).length);
const llmProviderOptions = computed(() => {
  const options = Array.isArray(adminSettings.value.llm_provider_options)
    ? adminSettings.value.llm_provider_options
    : [];
  return options.length ? options : [{ id: 'gemini', label: 'Google Gemini', models: ['gemini-2.5-flash'] }];
});
const settingsModelOptions = computed(() => {
  const provider = llmProviderOptions.value.find((item) => item.id === settingsForm.llm_provider);
  const options = provider?.models ? [...provider.models] : [];
  const current = settingsForm.llm_model || adminSettings.value.llm_model || adminSettings.value.gemini_review_model;
  if (current && !options.includes(current)) options.unshift(current);
  return options;
});
const keyModelOptions = computed(() => {
  const provider = llmProviderOptions.value.find((item) => item.id === geminiKeyForm.provider);
  const options = provider?.models ? [...provider.models] : [];
  if (geminiKeyForm.model && !options.includes(geminiKeyForm.model)) options.unshift(geminiKeyForm.model);
  return options;
});

const summaryMetrics = computed(() => [
  { label: 'Users', value: number(summary.value.user_count), icon: UsersRound, tone: 'blue' },
  { label: 'Funded projects', value: number(summary.value.project_count), icon: FolderKanban, tone: 'green' },
  { label: 'Open tasks', value: number(summary.value.open_task_count), icon: ListChecks, tone: 'amber' },
  { label: 'Work pool', value: mrgFromCents(summary.value.work_pool_cents), icon: CircleDollarSign, tone: 'purple' },
  { label: 'Paid tasks', value: mrgFromCents(summary.value.paid_task_cents), icon: CheckCircle2, tone: 'green' },
  { label: 'Ledger entries', value: number(ledgerEntries.value.length), icon: Activity, tone: 'blue' },
]);

const filteredProjects = computed(() => {
  if (!query.value) return projects.value;
  return projects.value.filter((project) => haystack(project).includes(query.value));
});

const projectLookup = computed(() => {
  const rows = {};
  for (const project of projects.value) {
    rows[project.id] = project;
  }
  return rows;
});

const searchedTasks = computed(() => {
  if (!query.value) return tasks.value;
  return tasks.value.filter((task) => haystack(task).includes(query.value));
});

const issueTabCounts = computed(() => ({
  open: searchedTasks.value.filter((task) => issueStateForTask(task) === 'open').length,
  closed: searchedTasks.value.filter((task) => issueStateForTask(task) === 'closed').length,
}));

const issueStateTabs = computed(() => [
  { id: 'open', label: 'Open', count: issueTabCounts.value.open },
  { id: 'closed', label: 'Closed', count: issueTabCounts.value.closed },
]);

const filteredTasks = computed(() => searchedTasks.value.filter((task) => issueStateForTask(task) === taskIssueTab.value));

const filteredUsers = computed(() => {
  if (!query.value) return users.value;
  return users.value.filter((row) => haystack(row).includes(query.value));
});
const opsQueueStats = computed(() => opsQueue.value?.stats || {});
const opsQueueItems = computed(() => Array.isArray(opsQueue.value?.items) ? opsQueue.value.items : []);
const filteredOpsQueueItems = computed(() => {
  if (!query.value) return opsQueueItems.value;
  return opsQueueItems.value.filter((item) => haystack(item).includes(query.value));
});
const opsQueueMetrics = computed(() => [
  { label: 'Open items', value: number(opsQueueStats.value.total_count), icon: AlertTriangle, tone: opsQueueStats.value.critical_count ? 'red' : 'amber' },
  { label: 'Disputes', value: number(opsQueueStats.value.dispute_count), icon: AlertTriangle, tone: opsQueueStats.value.dispute_count ? 'amber' : 'green' },
  { label: 'Moderation', value: number(opsQueueStats.value.moderation_count), icon: ShieldCheck, tone: opsQueueStats.value.moderation_count ? 'red' : 'green' },
  { label: 'Fraud review', value: number(opsQueueStats.value.fraud_count), icon: AlertTriangle, tone: opsQueueStats.value.fraud_count ? 'red' : 'green' },
  { label: 'Payout review', value: number(opsQueueStats.value.payout_review_count), icon: CircleDollarSign, tone: opsQueueStats.value.payout_review_count ? 'blue' : 'green' },
]);
const adminCommandTitle = computed(() => activeNav.value?.title || 'Admin workspace');
const adminCommandBody = computed(() => {
  if (activeView.value === 'tasks') return 'Review GitHub issues, inspect PRs, assign bounty credit, and keep payout proof aligned with the ledger.';
  if (activeView.value === 'ledger') return 'Credit MRG, audit verified records, and switch between public references and hash evidence.';
  if (activeView.value === 'ops') return 'Triage disputes, moderation alerts, payout fraud risks, SSL security attention, and manual credit audit rows.';
  if (activeView.value === 'users') return 'Inspect account activity, update roles, and keep admin access controlled from one panel.';
  if (activeView.value === 'setting') return 'Tune review models, provider tokens, quotas, and webhook health for the automation layer.';
  if (activeView.value === 'test-settings') return 'Manage public test-mode publish settings, integration keys, and shared test password for bounty contributors.';
  return 'Monitor funded work, task queues, SSL health, user activity, and payout records from one operations surface.';
});
const adminCommandMetrics = computed(() => [
  {
    label: 'Visible tasks',
    value: number(filteredTasks.value.length),
    icon: ListChecks,
    tone: 'amber',
  },
  {
    label: 'Funded projects',
    value: number(filteredProjects.value.length),
    icon: FolderKanban,
    tone: 'green',
  },
  {
    label: 'Ledger rows',
    value: number(ledgerEntries.value.length),
    icon: Activity,
    tone: 'blue',
  },
  {
    label: 'Ops attention',
    value: number(opsQueueStats.value.total_count),
    icon: AlertTriangle,
    tone: opsQueueStats.value.critical_count ? 'red' : opsQueueStats.value.total_count ? 'amber' : 'green',
  },
]);

const TableHeader = defineComponent({
  props: {
    title: { type: String, required: true },
    count: { type: Number, required: true },
  },
  setup(props) {
    return () => h('header', { class: 'table-header' }, [
      h('div', [h('span', 'Data'), h('h2', props.title)]),
      h('strong', `${props.count} rows`),
    ]);
  },
});

const DataTable = defineComponent({
  props: {
    columns: { type: Array, required: true },
  },
  setup(props, { slots }) {
    return () => h('div', { class: 'table-wrap' }, [
      h('table', { class: ['admin-table', compactRows.value ? 'compact' : ''] }, [
        h('thead', [h('tr', props.columns.map((column) => h('th', column)))]),
        h('tbody', slots.default?.() || []),
      ]),
    ]);
  },
});

async function api(path, options = {}) {
  const response = await fetch(path, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      ...(token.value ? { Authorization: `Bearer ${token.value}` } : {}),
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
    if (response.status === 401) logout(false);
    throw new Error(payload.error || 'Request failed');
  }
  return payload;
}

function routeForView(view) {
  return routeByView[view] || routeByView.builder;
}

function viewFromPath(pathname = '/') {
  const normalized = `/${String(pathname || '/').replace(/^\/+|\/+$/g, '')}`;
  if (normalized === '/') return 'builder';
  return viewByRoute[normalized] || 'builder';
}

function navigateToView(view, event) {
  event?.preventDefault();
  setActiveView(view);
}

function setActiveView(view, options = {}) {
  const route = routeForView(view);
  activeView.value = routeByView[view] ? view : 'builder';
  if (!hasWindow || options.push === false) return;

  const current = window.location.pathname || '/';
  if (current === route) return;
  const method = options.replace ? 'replaceState' : 'pushState';
  window.history[method]({ view: activeView.value }, '', route);
}

function syncViewFromLocation(options = {}) {
  if (!hasWindow) return;
  const view = viewFromPath(window.location.pathname);
  activeView.value = view;
  const canonical = routeForView(view);
  if (options.replace && window.location.pathname !== canonical) {
    window.history.replaceState({ view }, '', canonical);
  }
}

function updateDocumentTitle() {
  if (!hasWindow) return;
  document.title = `${activeNav.value?.title || 'Admin workspace'} | MergeOS Admin`;
}

async function login() {
  authBusy.value = true;
  authError.value = '';
  try {
    const auth = await api('/api/auth/login', {
      method: 'POST',
      body: JSON.stringify(loginForm),
      headers: {},
    });
    if (auth.user?.role !== 'admin') {
      throw new Error('Admin access is required.');
    }
    token.value = auth.token;
    adminUser.value = auth.user;
    if (hasWindow) localStorage.setItem(storageKey, auth.token);
    await loadAdminData();
  } catch (error) {
    authError.value = error.message;
    token.value = '';
    adminUser.value = null;
    if (hasWindow) localStorage.removeItem(storageKey);
  } finally {
    authBusy.value = false;
  }
}

async function restoreSession() {
  if (!token.value) return;
  try {
    const user = await api('/api/auth/me');
    if (user.role !== 'admin') {
      throw new Error('Admin access is required.');
    }
    adminUser.value = user;
    await loadAdminData();
  } catch (error) {
    authError.value = error.message;
    logout(false);
  }
}

async function loadAdminData() {
  if (!token.value) return;
  loading.value = true;
  errorMessage.value = '';
  try {
    const [
      summaryData,
      userData,
      projectData,
      taskData,
      notificationData,
      ledgerData,
      opsQueueData,
      sslData,
      settingsData,
      geminiKeyData,
      geminiLogData,
      testSettingsData,
      testEntryData,
    ] = await Promise.all([
      api('/api/admin/summary'),
      api('/api/admin/users'),
      api('/api/admin/projects'),
      api('/api/admin/tasks'),
      api('/api/admin/notifications'),
      api('/api/admin/ledger'),
      api('/api/admin/ops-queue'),
      api('/api/admin/ssl'),
      api('/api/admin/settings'),
      api('/api/admin/gemini/keys'),
      api('/api/admin/gemini/webhooks?limit=100'),
      api('/api/admin/test-settings'),
      api('/api/admin/test-settings/entries'),
    ]);
    summary.value = summaryData || {};
    users.value = Array.isArray(userData) ? userData : [];
    projects.value = Array.isArray(projectData) ? projectData : [];
    tasks.value = Array.isArray(taskData) ? taskData : [];
    notifications.value = Array.isArray(notificationData) ? notificationData : [];
    ledgerEntries.value = Array.isArray(ledgerData) ? ledgerData : [];
    opsQueue.value = {
      stats: opsQueueData?.stats || {},
      items: Array.isArray(opsQueueData?.items) ? opsQueueData.items : [],
    };
    sslRows.value = Array.isArray(sslData) ? sslData : [];
    adminSettings.value = settingsData || {};
    syncSettingsForm();
    geminiKeys.value = Array.isArray(geminiKeyData) ? geminiKeyData : [];
    geminiWebhookLogs.value = Array.isArray(geminiLogData) ? geminiLogData : [];
    testSettingsConfig.value = testSettingsData || {};
    testSettingsEntries.value = Array.isArray(testEntryData) ? testEntryData : [];
    void syncTaskIssueStates(tasks.value);
    ensureSelectedUser();
  } catch (error) {
    errorMessage.value = error.message;
  } finally {
    loading.value = false;
  }
}

function ensureSelectedUser() {
  if (!users.value.length) {
    hydrateUserForm(null);
    return;
  }
  const current = users.value.find((row) => row.id === selectedUserId.value);
  const fallback = users.value.find((row) => row.id === adminUser.value?.id) || users.value[0];
  openUserEditor(current || fallback, { silent: true });
}

function openUserEditor(row, options = {}) {
  if (!row) return;
  selectedUserId.value = row.id;
  hydrateUserForm(row);
  if (!options.silent) {
    userEditorError.value = '';
    userEditorMessage.value = '';
  }
}

function hydrateUserForm(row) {
  userForm.id = row?.id || '';
  userForm.name = row?.name || '';
  userForm.company_name = row?.company_name || '';
  userForm.email = row?.email || '';
  userForm.role = row?.role || 'client';
  userForm.password = '';
  userForm.password_confirm = '';
}

async function saveSelectedUser() {
  userEditorBusy.value = true;
  userEditorError.value = '';
  userEditorMessage.value = '';
  try {
    if (!userForm.id) {
      throw new Error('Select a user first.');
    }
    if (userForm.password || userForm.password_confirm) {
      if (userForm.password !== userForm.password_confirm) {
        throw new Error('Password confirmation does not match.');
      }
    }

    const payload = {
      name: userForm.name,
      company_name: userForm.company_name,
      email: userForm.email,
      role: userForm.role,
    };
    if (userForm.password) {
      payload.password = userForm.password;
    }

    const updated = await api(`/api/admin/users/${encodeURIComponent(userForm.id)}`, {
      method: 'PATCH',
      body: JSON.stringify(payload),
    });
    users.value = users.value.map((row) => (row.id === updated.id ? updated : row));
    if (adminUser.value?.id === updated.id) {
      adminUser.value = {
        ...adminUser.value,
        name: updated.name,
        company_name: updated.company_name,
        email: updated.email,
        role: updated.role,
      };
    }
    openUserEditor(updated, { silent: true });
    userEditorMessage.value = 'User updated.';
  } catch (error) {
    userEditorError.value = error.message;
  } finally {
    userEditorBusy.value = false;
  }
}

function findOpsTask(item = {}) {
  const taskID = String(item.task_id || '').trim();
  if (taskID) {
    const exact = tasks.value.find((task) => task.id === taskID);
    if (exact) return exact;
  }
  const issueNumber = Number(item.issue_number) || 0;
  if (issueNumber > 0) {
    const projectID = String(item.project_id || '').trim();
    return tasks.value.find((task) => Number(task.issue_number) === issueNumber && (!projectID || task.project_id === projectID))
      || tasks.value.find((task) => Number(task.issue_number) === issueNumber);
  }
  return null;
}

function opsActionsForItem(item = {}) {
  const rows = Array.isArray(item.actions) ? item.actions : [];
  const normalized = rows
    .map((action, index) => ({
      key: `${item.id || 'ops'}:${action.id || action.type || index}`,
      id: action.id || `${action.type || 'action'}-${index}`,
      label: action.label || opsActionLabel(action.type),
      type: action.type || 'open_url',
      url: action.url || '',
    }))
    .filter((action) => action.type || action.url);
  if (!normalized.length && item.url) {
    normalized.push({
      key: `${item.id || 'ops'}:open-url`,
      id: 'open-url',
      label: 'Open',
      type: 'open_url',
      url: item.url,
    });
  }
  return normalized;
}

function opsActionLabel(type = '') {
  switch (type) {
    case 'review_task_pulls':
      return 'Review PRs';
    case 'run_ssl_review':
      return 'Run SSL Review';
    case 'refresh_admin_ops':
      return 'Refresh Queue';
    case 'open_url':
      return 'Open';
    default:
      return titleize(type || 'Action');
  }
}

function opsActionIcon(action = {}) {
  switch (action.type) {
    case 'review_task_pulls':
      return GitPullRequest;
    case 'run_ssl_review':
      return ShieldCheck;
    case 'refresh_admin_ops':
      return RefreshCw;
    case 'open_url':
      return ExternalLink;
    default:
      return Activity;
  }
}

function opsActionBusyKey(item = {}, action = {}) {
  return `${item.id || 'ops'}:${action.id || action.type || 'action'}`;
}

function opsActionBusyFor(item = {}, action = {}) {
  const type = action.type || '';
  if (type === 'run_ssl_review') return sslReviewBusy.value || Boolean(opsActionBusy.value[opsActionBusyKey(item, action)]);
  if (type === 'refresh_admin_ops') return loading.value || Boolean(opsActionBusy.value[opsActionBusyKey(item, action)]);
  return Boolean(opsActionBusy.value[opsActionBusyKey(item, action)]);
}

function openOpsURL(url = '') {
  const target = String(url || '').trim();
  if (!target || !hasWindow) return false;
  if (/^https?:\/\//i.test(target)) {
    window.open(target, '_blank', 'noopener,noreferrer');
    return true;
  }
  if (target.startsWith('/')) {
    window.open(new URL(target, window.location.origin).toString(), '_blank', 'noopener,noreferrer');
    return true;
  }
  return false;
}

async function handleOpsQueueAction(item = {}, action = {}) {
  const key = opsActionBusyKey(item, action);
  if (opsActionBusy.value[key]) return;
  opsActionBusy.value = { ...opsActionBusy.value, [key]: true };
  opsActionError.value = '';
  opsActionMessage.value = '';
  try {
    switch (action.type) {
      case 'open_url': {
        if (!openOpsURL(action.url || item.url)) {
          throw new Error('This ops item has no public URL to open.');
        }
        opsActionMessage.value = `Opened ${action.label || 'ops reference'}.`;
        break;
      }
      case 'refresh_admin_ops': {
        await loadAdminData();
        opsActionMessage.value = 'Operations queue refreshed.';
        break;
      }
      case 'review_task_pulls': {
        setActiveView('tasks');
        const task = findOpsTask(item);
        if (task) {
          taskIssueTab.value = issueStateForTask(task);
          search.value = task.issue_number ? String(task.issue_number) : (task.title || task.id || '');
          expandedTaskPulls.value = { ...expandedTaskPulls.value, [task.id]: true };
          await loadTaskPulls(task, true);
          opsActionMessage.value = `Opened PR review for ${taskIssueLabel(task)}.`;
        } else {
          search.value = item.issue_number ? String(item.issue_number) : (item.task_id || item.reference || '');
          opsActionMessage.value = 'Opened task operations. No exact task row matched this ops item.';
        }
        break;
      }
      case 'run_ssl_review': {
        setActiveView('ssl');
        await reviewSSLNow();
        opsActionMessage.value = sslReviewMessage.value || 'SSL review completed.';
        break;
      }
      default: {
        if (action.url || item.url) {
          openOpsURL(action.url || item.url);
          opsActionMessage.value = `Opened ${action.label || 'ops reference'}.`;
        } else {
          opsActionMessage.value = `${action.label || 'Ops action'} is marked for manual review.`;
        }
      }
    }
  } catch (error) {
    opsActionError.value = error.message || 'Could not run ops action.';
  } finally {
    opsActionBusy.value = { ...opsActionBusy.value, [key]: false };
  }
}

async function reviewSSLNow() {
  sslReviewBusy.value = true;
  sslReviewError.value = '';
  sslReviewMessage.value = '';
  try {
    const rows = await api('/api/admin/ssl/review', {
      method: 'POST',
      body: JSON.stringify({}),
    });
    sslRows.value = Array.isArray(rows) ? rows : [];
    sslReviewMessage.value = `Reviewed ${sslRows.value.length} domains.`;
  } catch (error) {
    sslReviewError.value = error.message;
  } finally {
    sslReviewBusy.value = false;
  }
}

function syncSettingsForm() {
  settingsForm.llm_provider = adminSettings.value.llm_provider || 'gemini';
  settingsForm.llm_model = adminSettings.value.llm_model || adminSettings.value.gemini_review_model || modelFallbackForProvider(settingsForm.llm_provider);
  settingsForm.gemini_review_model = settingsForm.llm_provider === 'gemini' ? settingsForm.llm_model : (adminSettings.value.gemini_review_model || 'gemini-2.5-flash');
  syncSelectedProviderModel();
  if (!geminiKeyForm.key_value) {
    geminiKeyForm.provider = settingsForm.llm_provider;
    geminiKeyForm.model = settingsForm.llm_model;
    syncKeyProviderModel();
  }
}

async function saveAdminSettings() {
  settingsBusy.value = true;
  settingsError.value = '';
  settingsMessage.value = '';
  try {
    const updated = await api('/api/admin/settings', {
      method: 'PATCH',
      body: JSON.stringify({
        llm_provider: settingsForm.llm_provider,
        llm_model: settingsForm.llm_model,
        gemini_review_model: settingsForm.llm_provider === 'gemini' ? settingsForm.llm_model : settingsForm.gemini_review_model,
      }),
    });
    adminSettings.value = updated || {};
    syncSettingsForm();
    settingsMessage.value = `Using ${providerLabel(adminSettings.value.llm_provider)} / ${adminSettings.value.llm_model}.`;
  } catch (error) {
    settingsError.value = error.message;
  } finally {
    settingsBusy.value = false;
  }
}

async function loadGeminiAdminData() {
  if (!token.value) return;
  errorMessage.value = '';
  try {
    const [keyData, logData] = await Promise.all([
      api('/api/admin/gemini/keys'),
      api('/api/admin/gemini/webhooks?limit=100'),
      api('/api/admin/test-settings'),
      api('/api/admin/test-settings/entries'),
    ]);
    geminiKeys.value = Array.isArray(keyData) ? keyData : [];
    geminiWebhookLogs.value = Array.isArray(logData) ? logData : [];
  } catch (error) {
    errorMessage.value = error.message;
  }
}

async function addGeminiKey() {
  geminiKeyBusy.value = true;
  geminiKeyError.value = '';
  geminiKeyMessage.value = '';
  try {
    const row = await api('/api/admin/gemini/keys', {
      method: 'POST',
      body: JSON.stringify({
        provider: geminiKeyForm.provider,
        model: geminiKeyForm.model,
        key_value: geminiKeyForm.key_value,
      }),
    });
    geminiKeys.value = [row, ...geminiKeys.value.filter((item) => item.id !== row.id)];
    geminiKeyForm.key_value = '';
    geminiKeyMessage.value = `Added ${providerLabel(row.provider)} token ${row.key_hint}.`;
  } catch (error) {
    geminiKeyError.value = error.message;
  } finally {
    geminiKeyBusy.value = false;
  }
}

async function setGeminiKeyStatus(row, status) {
  if (!row?.id) return;
  geminiActionBusy.value = { ...geminiActionBusy.value, [row.id]: true };
  geminiKeyError.value = '';
  geminiKeyMessage.value = '';
  try {
    const updated = await api(`/api/admin/gemini/keys/${encodeURIComponent(row.id)}`, {
      method: 'PATCH',
      body: JSON.stringify({ status }),
    });
    geminiKeys.value = geminiKeys.value.map((item) => (item.id === updated.id ? updated : item));
    geminiKeyMessage.value = `${updated.key_hint} is ${titleize(updated.status)}.`;
  } catch (error) {
    geminiKeyError.value = error.message;
  } finally {
    geminiActionBusy.value = { ...geminiActionBusy.value, [row.id]: false };
  }
}

async function resetGeminiKey(row) {
  if (!row?.id) return;
  geminiActionBusy.value = { ...geminiActionBusy.value, [row.id]: true };
  geminiKeyError.value = '';
  geminiKeyMessage.value = '';
  try {
    const updated = await api(`/api/admin/gemini/keys/${encodeURIComponent(row.id)}`, {
      method: 'PATCH',
      body: JSON.stringify({ reset_counts: true }),
    });
    geminiKeys.value = geminiKeys.value.map((item) => (item.id === updated.id ? updated : item));
    geminiKeyMessage.value = `Reset counters for ${updated.key_hint}.`;
  } catch (error) {
    geminiKeyError.value = error.message;
  } finally {
    geminiActionBusy.value = { ...geminiActionBusy.value, [row.id]: false };
  }
}

async function testGeminiKey(row) {
  if (!row?.id) return;
  geminiTestBusy.value = { ...geminiTestBusy.value, [row.id]: true };
  geminiKeyError.value = '';
  geminiKeyMessage.value = '';
  geminiTestResults.value = { ...geminiTestResults.value, [row.id]: null };
  const provider = row.provider || 'gemini';
  const model = settingsForm.llm_provider === provider
    ? settingsForm.llm_model
    : (row.model || modelFallbackForProvider(provider));
  try {
    const result = await api(`/api/admin/gemini/keys/${encodeURIComponent(row.id)}/test`, {
      method: 'POST',
      body: JSON.stringify({ provider, model }),
    });
    if (result?.key?.id) {
      geminiKeys.value = geminiKeys.value.map((item) => (item.id === result.key.id ? result.key : item));
    }
    const status = result?.status_code ? `HTTP ${result.status_code}` : 'No status';
    const testedProvider = providerLabel(result?.provider || provider);
    const message = result?.ok
      ? `Test OK on ${testedProvider} / ${result.model || model} (${result.duration_millis || 0} ms)`
      : `Test failed on ${testedProvider} / ${result?.model || model}: ${result?.error || status}`;
    geminiTestResults.value = {
      ...geminiTestResults.value,
      [row.id]: { ok: Boolean(result?.ok), message },
    };
    if (result?.ok) {
      geminiKeyMessage.value = `${row.key_hint} passed with ${testedProvider} / ${result.model || model}.`;
    } else {
      geminiKeyError.value = `${row.key_hint} failed: ${result?.error || status}.`;
    }
  } catch (error) {
    geminiTestResults.value = {
      ...geminiTestResults.value,
      [row.id]: { ok: false, message: error.message },
    };
    geminiKeyError.value = error.message;
  } finally {
    geminiTestBusy.value = { ...geminiTestBusy.value, [row.id]: false };
  }
}

function providerLabel(providerId = 'gemini') {
  const provider = llmProviderOptions.value.find((item) => item.id === providerId);
  return provider?.label || titleize(providerId || 'gemini');
}

function modelFallbackForProvider(providerId = 'gemini') {
  const provider = llmProviderOptions.value.find((item) => item.id === providerId);
  return provider?.models?.[0] || 'gemini-2.5-flash';
}

function syncSelectedProviderModel() {
  if (!settingsModelOptions.value.includes(settingsForm.llm_model)) {
    settingsForm.llm_model = modelFallbackForProvider(settingsForm.llm_provider);
  }
  if (settingsForm.llm_provider === 'gemini') {
    settingsForm.gemini_review_model = settingsForm.llm_model;
  }
}

function syncKeyProviderModel() {
  if (!keyModelOptions.value.includes(geminiKeyForm.model)) {
    geminiKeyForm.model = modelFallbackForProvider(geminiKeyForm.provider);
  }
}

function pullsForTask(task) {
  return taskPulls.value[task.id] || [];
}

function visiblePullsForTask(task) {
  return pullsForTask(task);
}

function taskPullSummary(task) {
  if (taskPullsLoading.value[task.id]) return 'Checking linked PRs';
  if (!taskPullsLoaded.value[task.id]) return '';
  const pulls = pullsForTask(task);
  if (!pulls.length) return 'No linked PRs yet';
  const open = pulls.filter((pull) => !pull.merged && pull.state === 'open').length;
  const merged = pulls.filter((pull) => pull.merged).length;
  const closed = pulls.filter((pull) => !pull.merged && pull.state === 'closed').length;
  const parts = [
    open ? `${open} open` : '',
    merged ? `${merged} merged` : '',
    closed ? `${closed} closed` : '',
  ].filter(Boolean);
  return parts.length ? parts.join(' / ') : `${pulls.length} linked PR${pulls.length === 1 ? '' : 's'}`;
}

function emptyPullMessage() {
  return 'No linked PRs yet.';
}

function taskProjectTitle(task = {}) {
  return projectLookup.value[task.project_id]?.title || task.project_id || 'Project';
}

function taskIssueLabel(task = {}) {
  if (task.issue_url) return `Issue #${task.issue_number}`;
  return task.id || 'Task';
}

function normalizeIssueState(value = '') {
  const state = String(value || '').trim().toLowerCase();
  return state === 'closed' || state === 'close' ? 'closed' : 'open';
}

function issueStateForTask(task = {}) {
  return normalizeIssueState(taskIssueStates.value[task.id] || task.issue_state || task.github_issue_state || 'open');
}

function githubIssueApiURL(task = {}) {
  const raw = String(task.issue_url || '').trim();
  if (!raw || !hasWindow) return '';
  try {
    const parsed = new URL(raw);
    if (!['github.com', 'www.github.com'].includes(parsed.hostname.toLowerCase())) return '';
    const parts = parsed.pathname.split('/').filter(Boolean);
    if (parts.length < 4 || parts[2] !== 'issues') return '';
    return `https://api.github.com/repos/${encodeURIComponent(parts[0])}/${encodeURIComponent(parts[1])}/issues/${encodeURIComponent(parts[3])}`;
  } catch {
    return '';
  }
}

async function syncTaskIssueStates(rows = []) {
  if (!hasWindow) return;
  const candidates = rows
    .map((task) => ({ id: task.id, url: githubIssueApiURL(task) }))
    .filter((row) => row.id && row.url);
  if (!candidates.length) return;

  const updates = {};
  await Promise.allSettled(candidates.map(async (row) => {
    const response = await fetch(row.url, {
      headers: {
        Accept: 'application/vnd.github+json',
        'X-GitHub-Api-Version': '2022-11-28',
      },
    });
    if (!response.ok) return;
    const payload = await response.json();
    updates[row.id] = normalizeIssueState(payload.state);
  }));

  if (Object.keys(updates).length) {
    taskIssueStates.value = { ...taskIssueStates.value, ...updates };
  }
}

function isTaskPullsExpanded(task = {}) {
  return Boolean(expandedTaskPulls.value[task.id]);
}

async function toggleTaskPulls(task = {}) {
  if (!task?.id) return;
  const nextExpanded = !isTaskPullsExpanded(task);
  expandedTaskPulls.value = { ...expandedTaskPulls.value, [task.id]: nextExpanded };
  if (nextExpanded) {
    await loadTaskPulls(task);
  }
}

function mergeKey(task, pull) {
  return `${task.id}:${pull.number}`;
}

function defaultBountyForTask(task = {}) {
  const reward = Number(task.reward_cents) || 25;
  return bountyOptions.find((option) => option.reward_mrg === reward) || bountyOptions[0];
}

function ensureMergeSelection(task, pull) {
  if (!task?.id || !pull?.number) return;
  const key = mergeKey(task, pull);
  if (mergeSelections.value[key]) return;
  const option = defaultBountyForTask(task);
  mergeSelections.value = {
    ...mergeSelections.value,
    [key]: {
      bounty_type: option.id,
      reward_mrg: option.reward_mrg,
    },
  };
}

function mergeSelection(task, pull) {
  ensureMergeSelection(task, pull);
  return mergeSelections.value[mergeKey(task, pull)] || {
    bounty_type: bountyOptions[0].id,
    reward_mrg: bountyOptions[0].reward_mrg,
  };
}

function setMergeBounty(task, pull, value) {
  const option = bountyOptions.find((row) => row.id === value) || bountyOptions[0];
  const key = mergeKey(task, pull);
  mergeSelections.value = {
    ...mergeSelections.value,
    [key]: {
      bounty_type: option.id,
      reward_mrg: option.reward_mrg,
    },
  };
}

function setMergeReward(task, pull, value) {
  const key = mergeKey(task, pull);
  const current = mergeSelection(task, pull);
  const reward = Math.max(1, Math.round(Number(value) || 0));
  mergeSelections.value = {
    ...mergeSelections.value,
    [key]: {
      ...current,
      reward_mrg: reward,
    },
  };
}

function setManualCreditBounty(event) {
  const value = event?.target?.value || manualCreditForm.bounty_type;
  const option = bountyOptions.find((row) => row.id === value) || bountyOptions[1] || bountyOptions[0];
  manualCreditForm.bounty_type = option.id;
  manualCreditForm.reward_mrg = option.reward_mrg;
}

function pullStatus(pull) {
  if (pull.merged) return 'merged';
  if (pull.draft) return 'draft';
  return [pull.state || 'open', pull.mergeable_state].filter(Boolean).join(' / ');
}

function pullReadinessLabel(pull = {}) {
  const status = pull.readiness?.status || '';
  if (!status) return 'Unscored';
  return titleize(status);
}

function pullReadinessRisk(pull = {}) {
  const risk = pull.readiness?.risk_level || '';
  return risk ? `${titleize(risk)} risk` : 'No risk score';
}

function pullReadinessTone(pull = {}) {
  const status = pull.readiness?.status || '';
  const risk = pull.readiness?.risk_level || '';
  if (status === 'blocked' || risk === 'high') return 'red';
  if (status === 'needs_review' || risk === 'medium') return 'amber';
  if (status === 'ready' || risk === 'low') return 'green';
  return 'blue';
}

function pullReadinessNotes(pull = {}) {
  const readiness = pull.readiness || {};
  return [...(readiness.blockers || []), ...(readiness.warnings || [])].slice(0, 6);
}

function pullReadinessSignals(pull = {}) {
  return (pull.readiness?.signals || []).slice(0, 4);
}

function canMergeTaskPull(task, pull) {
  const selection = mergeSelection(task, pull);
  if (!pull?.author) return false;
  if (mergeBusy.value[mergeKey(task, pull)] || pull.draft) return false;
  if (pull.readiness && pull.readiness.can_merge === false) return false;
  if (!selection.bounty_type || Number(selection.reward_mrg) <= 0) return false;
  return pull.merged || pull.state === 'open';
}

async function loadTaskPulls(task, force = false) {
  if (!task?.id || !token.value) return;
  if (!force && (taskPullsLoaded.value[task.id] || taskPullsLoading.value[task.id])) return;
  taskPullsLoading.value = { ...taskPullsLoading.value, [task.id]: true };
  taskPullsError.value = { ...taskPullsError.value, [task.id]: '' };
  try {
    const payload = await api(`/api/admin/tasks/${encodeURIComponent(task.id)}/pulls`);
    taskPulls.value = {
      ...taskPulls.value,
      [task.id]: Array.isArray(payload.pull_requests) ? payload.pull_requests : [],
    };
    for (const pull of taskPulls.value[task.id]) {
      ensureMergeSelection(task, pull);
    }
    taskPullsLoaded.value = { ...taskPullsLoaded.value, [task.id]: true };
  } catch (error) {
    taskPullsError.value = { ...taskPullsError.value, [task.id]: error.message };
    taskPullsLoaded.value = { ...taskPullsLoaded.value, [task.id]: true };
  } finally {
    taskPullsLoading.value = { ...taskPullsLoading.value, [task.id]: false };
  }
}

async function mergeTaskPull(task, pull) {
  if (!canMergeTaskPull(task, pull)) return;
  const key = mergeKey(task, pull);
  const selection = mergeSelection(task, pull);
  mergeBusy.value = { ...mergeBusy.value, [key]: true };
  mergeMessages.value = { ...mergeMessages.value, [task.id]: '' };
  taskPullsError.value = { ...taskPullsError.value, [task.id]: '' };
  try {
    const result = await api(`/api/admin/tasks/${encodeURIComponent(task.id)}/pulls/${pull.number}/merge`, {
      method: 'POST',
      body: JSON.stringify({
        bounty_type: selection.bounty_type,
        reward_mrg: Number(selection.reward_mrg) || 0,
      }),
    });
    if (result.task) {
      tasks.value = tasks.value.map((row) => (row.id === result.task.id ? result.task : row));
    }
    if (result.pull_request) {
      taskPulls.value = {
        ...taskPulls.value,
        [task.id]: pullsForTask(task).map((row) => (row.number === result.pull_request.number ? result.pull_request : row)),
      };
    }
    const commentStatus = result.comment_error ? ` Comment failed: ${result.comment_error}` : ' Commented on PR.';
    mergeMessages.value = { ...mergeMessages.value, [task.id]: `Paid ${mrg(result.reward_mrg || selection.reward_mrg)} to ${result.worker_id || `github:${pull.author}`}.${commentStatus}` };
    const proofHash = result.entry_hash || (result.task && result.task.proof_hash) || '';
    mergeCreditReceipt.value = {
      sequence: result.ledger_sequence,
      entryHash: proofHash,
      creditUrl: result.credit_url,
      commentUrl: result.comment_url,
      workerId: result.worker_id || `github:${pull.author}`,
      rewardMrg: result.reward_mrg || selection.reward_mrg,
      bountyType: result.bounty_type || selection.bounty_type,
      mergeUrl: result.pull_request && result.pull_request.merge_url,
      commentError: result.comment_error,
    };
    const [summaryData, ledgerData] = await Promise.all([
      api('/api/admin/summary'),
      api('/api/admin/ledger'),
    ]);
    summary.value = summaryData || {};
    ledgerEntries.value = Array.isArray(ledgerData) ? ledgerData : [];
  } catch (error) {
    taskPullsError.value = { ...taskPullsError.value, [task.id]: error.message };
  } finally {
    mergeBusy.value = { ...mergeBusy.value, [key]: false };
  }
}

async function createManualCredit() {
  if (!token.value || manualCreditBusy.value) return;
  manualCreditBusy.value = true;
  manualCreditError.value = '';
  manualCreditMessage.value = '';
  try {
    const result = await api('/api/admin/ledger/credits', {
      method: 'POST',
      body: JSON.stringify({
        worker_id: manualCreditForm.worker_id,
        bounty_type: manualCreditForm.bounty_type,
        reward_mrg: Math.max(1, Math.round(Number(manualCreditForm.reward_mrg) || 0)),
        pr_url: manualCreditForm.pr_url,
        pr_title: manualCreditForm.pr_title,
      }),
    });
    const [summaryData, ledgerData] = await Promise.all([
      api('/api/admin/summary'),
      api('/api/admin/ledger'),
    ]);
    summary.value = summaryData || {};
    ledgerEntries.value = Array.isArray(ledgerData) ? ledgerData : [];
    const paidMrg = result.reward_mrg || manualCreditForm.reward_mrg;
    const paidWorker = result.worker_id || manualCreditForm.worker_id;
    manualCreditMessage.value = `Credited ${mrg(paidMrg)} to ${paidWorker}.`;
    if (result.ledger_entry) {
      manualCreditReceipt.value = {
        sequence: result.ledger_entry.sequence,
        entryHash: result.ledger_entry.entry_hash,
        creditUrl: result.credit_url,
        workerId: paidWorker,
        rewardMrg: paidMrg,
        bountyType: result.bounty_type || manualCreditForm.bounty_type,
        prUrl: manualCreditForm.pr_url,
        prTitle: manualCreditForm.pr_title,
      };
    }
    manualCreditForm.worker_id = '';
    manualCreditForm.pr_url = '';
    manualCreditForm.pr_title = '';
  } catch (error) {
    manualCreditError.value = error.message;
  } finally {
    manualCreditBusy.value = false;
  }
}

function logout(callApi = true) {
  const currentToken = token.value;
  token.value = '';
  adminUser.value = null;
  if (hasWindow) localStorage.removeItem(storageKey);
  if (callApi && currentToken) {
    fetch('/api/auth/logout', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${currentToken}`,
      },
      body: JSON.stringify({}),
    }).catch(() => {});
  }
}

function mrgFromCents(cents = 0) {
  return `${number(tokenAmountFromCents(cents))} ${tokenSymbol.value}`;
}

function tokenAmountFromCents(cents = 0) {
  return Math.max(0, Math.round(Number(cents) || 0));
}

function mrg(value = 0) {
  return `${number(value)} ${tokenSymbol.value}`;
}

function number(value = 0) {
  return new Intl.NumberFormat('en-US').format(Number(value) || 0);
}

function initials(value = '') {
  return (String(value).trim().slice(0, 2) || 'MO').toUpperCase();
}

function titleize(value = '') {
  return String(value).replaceAll('_', ' ').replace(/\b\w/g, (char) => char.toUpperCase());
}

function shortRef(value = '') {
  const text = String(value || '');
  if (text.length <= 18) return text || '-';
  return `${text.slice(0, 8)}...${text.slice(-6)}`;
}

function copyText(text) {
  if (!hasWindow || !text) return;
  navigator.clipboard.writeText(text).catch(() => {});
}

function buildMergeCreditComment(receipt) {
  if (!receipt) return '';
  const lines = [
    'MergeOS approved and merged this PR.',
    '',
  ];
  if (receipt.mergeUrl) lines.push(`- Merge URL: ${receipt.mergeUrl}`);
  if (receipt.creditUrl) lines.push(`- MRG credit URL: ${receipt.creditUrl}`);
  lines.push(`- Credited worker: ${receipt.workerId || '-'}`);
  lines.push(`- Bounty type: ${titleize(receipt.bountyType || '')}`);
  lines.push(`- MRG credited: ${receipt.rewardMrg || 0} MRG`);
  if (receipt.entryHash) lines.push(`- Proof hash: ${receipt.entryHash}`);
  return lines.join('\n');
}

function buildManualCreditComment(receipt) {
  if (!receipt) return '';
  const lines = [
    'MergeOS manual credit',
    '',
  ];
  if (receipt.prUrl) lines.push(`- PR URL: ${receipt.prUrl}`);
  if (receipt.prTitle) lines.push(`- PR title: ${receipt.prTitle}`);
  if (receipt.creditUrl) lines.push(`- MRG credit URL: ${receipt.creditUrl}`);
  lines.push(`- Credited worker: ${receipt.workerId || '-'}`);
  lines.push(`- Bounty type: ${titleize(receipt.bountyType || '')}`);
  lines.push(`- MRG credited: ${receipt.rewardMrg || 0} MRG`);
  if (receipt.entryHash) lines.push(`- Proof hash: ${receipt.entryHash}`);
  return lines.join('\n');
}

function geminiKeyStatusTone(status = '') {
  if (status === 'active') return 'green';
  if (status === 'disabled') return 'blue';
  if (status === 'quota_limited') return 'amber';
  return 'red';
}

function geminiWebhookStatusTone(status = '') {
  if (status === 'processed') return 'green';
  if (status === 'skipped') return 'blue';
  if (status === 'received') return 'blue';
  if (status === 'failed' || status === 'unauthorized' || status === 'bad_request' || status === 'service_unavailable') return 'red';
  return 'amber';
}

function opsSeverityTone(severity = '') {
  if (severity === 'critical' || severity === 'high') return 'red';
  if (severity === 'medium') return 'amber';
  if (severity === 'low') return 'blue';
  return 'green';
}

function opsIcon(type = '') {
  if (type.includes('fraud')) return AlertTriangle;
  if (type.includes('payout')) return CircleDollarSign;
  if (type.includes('security')) return ShieldCheck;
  if (type.includes('moderation')) return AlertTriangle;
  if (type.includes('dispute')) return AlertTriangle;
  return Activity;
}

function opsTypeLabel(type = '') {
  if (type === 'payout_review') return 'Payout';
  if (type === 'payout_audit') return 'Audit';
  if (type === 'security_moderation') return 'Security';
  if (type === 'fraud_review') return 'Fraud';
  return titleize(type || 'ops');
}

function opsReference(item = {}) {
  return item.reference || item.task_id || item.project_id || item.user_id || item.id || '';
}

function formatDate(value) {
  if (!value) return '-';
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return '-';
  return date.toLocaleString('en-US', { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' });
}

function haystack(row = {}) {
  return Object.values(row).join(' ').toLowerCase();
}

function handlePopState() {
  syncViewFromLocation();
}

watch(activeView, (view) => {
  updateDocumentTitle();
  if (view === 'users') ensureSelectedUser();
});

onMounted(() => {
  syncViewFromLocation({ replace: true });
  updateDocumentTitle();
  if (hasWindow) window.addEventListener('popstate', handlePopState);
  void restoreSession();
});

onBeforeUnmount(() => {
  if (hasWindow) window.removeEventListener('popstate', handlePopState);
});

// Test Settings
async function saveTestSettingsConfig() {
  testSettingsBusy.value = true; testSettingsError.value = ''; testSettingsMessage.value = '';
  try {
    const payload = {};
    if (testNewPassword.value) payload.test_password = testNewPassword.value;
    const updated = await api('/api/admin/test-settings', { method: 'PATCH', body: JSON.stringify(payload) });
    testSettingsConfig.value = updated || {};
    testSettingsMessage.value = testNewPassword.value ? 'Settings saved. Password updated.' : 'Settings saved.';
    testNewPassword.value = '';
  } catch (e) { testSettingsError.value = e.message; }
  finally { testSettingsBusy.value = false; }
}

async function toggleTestMode() {
  testSettingsBusy.value = true; testSettingsError.value = '';
  try {
    const updated = await api('/api/admin/test-settings', { method: 'PATCH', body: JSON.stringify({ test_mode_enabled: !testSettingsConfig.value.test_mode_enabled }) });
    testSettingsConfig.value = updated || {};
    testSettingsMessage.value = updated.test_mode_enabled ? 'Test mode enabled.' : 'Test mode disabled.';
  } catch (e) { testSettingsError.value = e.message; }
  finally { testSettingsBusy.value = false; }
}

async function addTestEntry() {
  testSettingsBusy.value = true; testSettingsError.value = ''; testSettingsMessage.value = '';
  try {
    let km = {};
    try { km = JSON.parse(testForm.key_value_json || '{}'); } catch { throw new Error('Invalid JSON metadata.'); }
    const entry = await api('/api/admin/test-settings/entries', { method: 'POST', body: JSON.stringify({
      integration_type: testForm.integration_type, display_name: testForm.display_name,
      setting_key: testForm.setting_key, setting_value: testForm.setting_value, key_value_map: km }) });
    testSettingsEntries.value = [entry, ...testSettingsEntries.value];
    testForm.integration_type = 'llm'; testForm.display_name = ''; testForm.setting_key = '';
    testForm.setting_value = ''; testForm.key_value_json = '{}';
    testSettingsMessage.value = 'Added ' + entry.setting_key;
  } catch (e) { testSettingsError.value = e.message; }
  finally { testSettingsBusy.value = false; }
}

async function deleteTestEntry(id) {
  if (!confirm('Delete this entry?')) return;
  testSettingsBusy.value = true; testSettingsError.value = '';
  try {
    await api('/api/admin/test-settings/entries/' + encodeURIComponent(id), { method: 'DELETE' });
    testSettingsEntries.value = testSettingsEntries.value.filter(e => e.id !== id);
    testSettingsMessage.value = 'Entry deleted.';
  } catch (e) { testSettingsError.value = e.message; }
  finally { testSettingsBusy.value = false; }
}

</script>
