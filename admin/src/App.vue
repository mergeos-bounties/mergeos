<template>
  <main v-if="!isAuthenticated" class="login-screen">
    <section class="login-panel" aria-labelledby="admin-login-title">
      <div class="login-brand">
        <span class="brand-mark"><Boxes :size="24" /></span>
        <div>
          <strong>MergeOS Admin</strong>
          <small>Elementor workspace</small>
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
        <span class="brand-mark"><Boxes :size="22" /></span>
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

      <section class="widget-palette" aria-labelledby="widget-palette-title">
        <div class="sidebar-section-title">
          <span id="widget-palette-title">Elementor widgets</span>
          <Plus :size="15" />
        </div>
        <button
          v-for="widget in builderWidgets"
          :key="widget.id"
          :class="{ active: selectedWidget === widget.id }"
          type="button"
          @click="selectWidget(widget.id)"
        >
          <component :is="widget.icon" :size="16" />
          <span>{{ widget.label }}</span>
        </button>
      </section>

      <button class="sidebar-refresh" :disabled="loading" type="button" @click="loadAdminData">
        <RefreshCw :size="16" />
        Refresh data
      </button>
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

      <p v-if="errorMessage" class="workspace-error">{{ errorMessage }}</p>

      <section v-if="activeView === 'builder'" class="builder-workspace">
        <div class="canvas-column">
          <div class="canvas-toolbar">
            <div>
              <span>Elementor canvas</span>
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
                      <small>{{ money(project.budget_cents) }} escrow</small>
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
                      <small>{{ task.status }} · {{ money(task.reward_cents) }}</small>
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
                    <small>{{ money(entry.amount_cents) }} · {{ shortRef(entry.reference) }}</small>
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
            <td>{{ money(project.budget_cents) }}</td>
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
            <p>Tasks stay visible for audit; merged PRs are hidden after credit is recorded.</p>
          </div>
          <div class="task-review-stats" aria-label="Task queue summary">
            <article>
              <strong>{{ number(filteredTasks.length) }}</strong>
              <small>Tasks visible</small>
            </article>
            <article>
              <strong>{{ number(summary.open_task_count) }}</strong>
              <small>Open tasks</small>
            </article>
            <article>
              <strong>{{ number(hiddenMergedPullCount) }}</strong>
              <small>Merged PRs hidden</small>
            </article>
          </div>
        </header>

        <div v-if="!filteredTasks.length" class="task-empty-state">
          <span class="metric-icon green"><CheckCircle2 :size="19" /></span>
          <strong>No tasks found</strong>
          <small>Clear the search to see the full task audit board.</small>
        </div>

        <div v-else class="task-review-list">
          <article v-for="task in filteredTasks" :key="task.id" class="task-review-item">
            <div class="task-review-main">
              <div class="task-review-title">
                <span class="task-issue-mark">{{ task.issue_number || 'T' }}</span>
                <div>
                  <strong>{{ task.title }}</strong>
                  <small>{{ taskIssueLabel(task) }} / {{ taskProjectTitle(task) }}</small>
                </div>
              </div>
              <div class="task-meta-row">
                <span>{{ task.required_worker_kind }}</span>
                <span>{{ task.suggested_agent_type || 'manual review' }}</span>
                <span>{{ task.status }}</span>
              </div>
            </div>

            <aside class="task-review-side">
              <span>Reward</span>
              <strong>{{ mrg(task.reward_cents) }}</strong>
              <small>{{ task.worker_id || 'Unassigned' }}</small>
            </aside>

            <section class="task-pr-section" aria-label="Linked pull requests">
              <div class="task-pr-toolbar">
                <button class="compact-action" :disabled="taskPullsLoading[task.id]" type="button" @click="loadTaskPulls(task, true)">
                  <GitPullRequest :size="14" />
                  {{ taskPullsLoading[task.id] ? 'Checking...' : 'Refresh PRs' }}
                </button>
                <small>{{ taskPullSummary(task) }}</small>
              </div>
              <p v-if="taskPullsError[task.id]" class="inline-error">{{ taskPullsError[task.id] }}</p>
              <p v-else-if="taskPullsLoaded[task.id] && !actionablePullsForTask(task).length" class="muted-inline">{{ emptyPullMessage(task) }}</p>
              <div v-else class="task-pr-list">
                <article v-for="pull in actionablePullsForTask(task)" :key="pull.number" class="task-pr-row">
                  <div class="task-pr-main">
                    <span :class="['metric-icon', pull.merged ? 'green' : pull.draft ? 'amber' : 'blue']">
                      <GitPullRequest :size="16" />
                    </span>
                    <div>
                      <strong>#{{ pull.number }} {{ pull.title }}</strong>
                      <small>@{{ pull.author }} / {{ pullStatus(pull) }} / {{ pull.head_ref || 'head' }} -> {{ pull.base_ref || 'base' }}</small>
                      <em>Credit: github:{{ pull.author }} / {{ mrg(mergeSelection(task, pull).reward_mrg) }}</em>
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
              <p v-if="mergeMessages[task.id]" class="inline-success">{{ mergeMessages[task.id] }}</p>
            </section>
          </article>
        </div>
      </section>

      <section v-else-if="activeView === 'ledger'" class="table-panel">
        <TableHeader title="Ledger" :count="ledgerEntries.length" />
        <DataTable :columns="['Seq', 'Type', 'From', 'To', 'Amount', 'Reference']">
          <tr v-for="entry in ledgerEntries.slice().reverse()" :key="entry.sequence">
            <td>{{ entry.sequence }}</td>
            <td><strong>{{ titleize(entry.type) }}</strong></td>
            <td>{{ entry.from_account || '-' }}</td>
            <td>{{ entry.to_account || '-' }}</td>
            <td>{{ money(entry.amount_cents) }}</td>
            <td>{{ showLedgerHashes ? shortRef(entry.entry_hash) : shortRef(entry.reference) }}</td>
          </tr>
        </DataTable>
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
              <td>{{ money(row.total_budget_cents) }}</td>
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
    </section>
  </div>
</template>

<script setup>
import { computed, defineComponent, h, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue';
import {
  Activity,
  AlertTriangle,
  BarChart3,
  Boxes,
  CheckCircle2,
  CircleDollarSign,
  Columns3,
  Eye,
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
  Plus,
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
const density = ref(2);
const showLedgerHashes = ref(false);
const compactRows = ref(true);
const selectedUserId = ref('');

const summary = ref({});
const users = ref([]);
const projects = ref([]);
const tasks = ref([]);
const notifications = ref([]);
const ledgerEntries = ref([]);
const sslRows = ref([]);
const taskPulls = ref({});
const taskPullsLoaded = ref({});
const taskPullsLoading = ref({});
const taskPullsError = ref({});
const mergeBusy = ref({});
const mergeMessages = ref({});
const mergeSelections = ref({});

const loginForm = reactive({
  email: 'admin@gmail.com',
  password: 'Admin123',
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

const navItems = [
  { id: 'builder', label: 'Elementor', title: 'Elementor admin canvas', kicker: 'BUILDER', icon: PanelLeft },
  { id: 'overview', label: 'Overview', title: 'Platform overview', kicker: 'DASHBOARD', icon: LayoutDashboard },
  { id: 'projects', label: 'Projects', title: 'Funded projects', kicker: 'PROJECTS', icon: FolderKanban },
  { id: 'tasks', label: 'Tasks', title: 'Task operations', kicker: 'TASKS', icon: ListChecks },
  { id: 'ledger', label: 'Ledger', title: 'Proof ledger', kicker: 'LEDGER', icon: Activity },
  { id: 'users', label: 'Users', title: 'User management', kicker: 'USERS', icon: UsersRound },
  { id: 'ssl', label: 'SSL', title: 'SSL monitoring', kicker: 'SECURITY', icon: ShieldCheck },
];

const routeByView = {
  builder: '/',
  overview: '/overview',
  projects: '/projects',
  tasks: '/tasks',
  ledger: '/ledger',
  users: '/users',
  ssl: '/ssl',
};
const viewByRoute = Object.entries(routeByView).reduce((routes, [view, route]) => {
  routes[route] = view;
  return routes;
}, {});

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

const summaryMetrics = computed(() => [
  { label: 'Users', value: number(summary.value.user_count), icon: UsersRound, tone: 'blue' },
  { label: 'Funded projects', value: number(summary.value.project_count), icon: FolderKanban, tone: 'green' },
  { label: 'Open tasks', value: number(summary.value.open_task_count), icon: ListChecks, tone: 'amber' },
  { label: 'Work pool', value: money(summary.value.work_pool_cents), icon: CircleDollarSign, tone: 'purple' },
  { label: 'Paid tasks', value: money(summary.value.paid_task_cents), icon: CheckCircle2, tone: 'green' },
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

const filteredTasks = computed(() => {
  if (!query.value) return tasks.value;
  return tasks.value.filter((task) => haystack(task).includes(query.value));
});

const filteredUsers = computed(() => {
  if (!query.value) return users.value;
  return users.value.filter((row) => haystack(row).includes(query.value));
});

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
    const [summaryData, userData, projectData, taskData, notificationData, ledgerData, sslData] = await Promise.all([
      api('/api/admin/summary'),
      api('/api/admin/users'),
      api('/api/admin/projects'),
      api('/api/admin/tasks'),
      api('/api/admin/notifications'),
      api('/api/admin/ledger'),
      api('/api/admin/ssl'),
    ]);
    summary.value = summaryData || {};
    users.value = Array.isArray(userData) ? userData : [];
    projects.value = Array.isArray(projectData) ? projectData : [];
    tasks.value = Array.isArray(taskData) ? taskData : [];
    notifications.value = Array.isArray(notificationData) ? notificationData : [];
    ledgerEntries.value = Array.isArray(ledgerData) ? ledgerData : [];
    sslRows.value = Array.isArray(sslData) ? sslData : [];
    ensureSelectedUser();
    if (activeView.value === 'tasks') void loadPullsForVisibleTasks();
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

function pullsForTask(task) {
  return taskPulls.value[task.id] || [];
}

function actionablePullsForTask(task) {
  return pullsForTask(task).filter((pull) => !pull.merged);
}

function hiddenMergedPullsForTask(task) {
  return pullsForTask(task).filter((pull) => pull.merged);
}

const hiddenMergedPullCount = computed(() =>
  Object.values(taskPulls.value).reduce((count, pulls) => count + (Array.isArray(pulls) ? pulls.filter((pull) => pull.merged).length : 0), 0),
);

function taskPullSummary(task) {
  const actionable = actionablePullsForTask(task).length;
  const hidden = hiddenMergedPullsForTask(task).length;
  if (!actionable && !hidden) return taskPullsLoaded.value[task.id] ? 'No linked PRs yet' : 'No PR loaded';
  if (actionable && hidden) return `${actionable} active / ${hidden} merged hidden`;
  if (hidden) return `${hidden} merged PR${hidden === 1 ? '' : 's'} hidden`;
  return `${actionable} active PR${actionable === 1 ? '' : 's'}`;
}

function emptyPullMessage(task) {
  return hiddenMergedPullsForTask(task).length ? 'All linked PRs are already merged and hidden.' : 'No linked PRs yet.';
}

function taskProjectTitle(task = {}) {
  return projectLookup.value[task.project_id]?.title || task.project_id || 'Project';
}

function taskIssueLabel(task = {}) {
  if (task.issue_url) return `Issue #${task.issue_number}`;
  return task.id || 'Task';
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

function pullStatus(pull) {
  if (pull.merged) return 'merged';
  if (pull.draft) return 'draft';
  return [pull.state || 'open', pull.mergeable_state].filter(Boolean).join(' / ');
}

function canMergeTaskPull(task, pull) {
  const selection = mergeSelection(task, pull);
  if (!pull?.author || task.status === 'accepted') return false;
  if (mergeBusy.value[mergeKey(task, pull)] || pull.draft) return false;
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

let visiblePullsLoading = false;

async function loadPullsForVisibleTasks() {
  if (visiblePullsLoading || activeView.value !== 'tasks' || !isAuthenticated.value) return;
  visiblePullsLoading = true;
  try {
    for (const task of filteredTasks.value) {
      if (activeView.value !== 'tasks') break;
      await loadTaskPulls(task);
    }
  } finally {
    visiblePullsLoading = false;
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

function selectWidget(id) {
  selectedWidget.value = id;
  setActiveView('builder');
}

function money(cents = 0) {
  return new Intl.NumberFormat('en-US', {
    style: 'currency',
    currency: 'USD',
    maximumFractionDigits: 0,
  }).format((Number(cents) || 0) / 100);
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
  if (view === 'tasks') void loadPullsForVisibleTasks();
});

watch(filteredTasks, () => {
  if (activeView.value === 'tasks') void loadPullsForVisibleTasks();
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
</script>
