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
      <button class="sidebar-brand" type="button" @click="activeView = 'builder'">
        <span class="brand-mark"><Boxes :size="22" /></span>
        <span>
          <strong>MergeOS</strong>
          <small>Admin Builder</small>
        </span>
      </button>

      <nav class="sidebar-nav" aria-label="Admin navigation">
        <button
          v-for="item in navItems"
          :key="item.id"
          :class="{ active: activeView === item.id }"
          type="button"
          @click="activeView = item.id"
        >
          <component :is="item.icon" :size="17" />
          <span>{{ item.label }}</span>
        </button>
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

      <section v-else-if="activeView === 'tasks'" class="table-panel">
        <TableHeader title="Tasks" :count="filteredTasks.length" />
        <DataTable :columns="['Task', 'Kind', 'Reward', 'Worker', 'Status']">
          <tr v-for="task in filteredTasks" :key="task.id">
            <td><strong>{{ task.title }}</strong><small>{{ task.project_id }}</small></td>
            <td>{{ task.required_worker_kind }}</td>
            <td>{{ money(task.reward_cents) }}</td>
            <td>{{ task.worker_id || task.suggested_agent_type || 'Unassigned' }}</td>
            <td><span :class="['status-pill', task.status === 'accepted' ? 'blue' : 'amber']">{{ task.status }}</span></td>
          </tr>
        </DataTable>
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
import { computed, defineComponent, h, onMounted, reactive, ref } from 'vue';
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
  ShieldCheck,
  SlidersHorizontal,
  Smartphone,
  Tablet,
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
const density = ref(2);
const showLedgerHashes = ref(false);
const compactRows = ref(true);

const summary = ref({});
const users = ref([]);
const projects = ref([]);
const tasks = ref([]);
const notifications = ref([]);
const ledgerEntries = ref([]);
const sslRows = ref([]);

const loginForm = reactive({
  email: 'admin@gmail.com',
  password: 'Admin123',
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
  } catch (error) {
    errorMessage.value = error.message;
  } finally {
    loading.value = false;
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
  activeView.value = 'builder';
}

function money(cents = 0) {
  return new Intl.NumberFormat('en-US', {
    style: 'currency',
    currency: 'USD',
    maximumFractionDigits: 0,
  }).format((Number(cents) || 0) / 100);
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

onMounted(() => {
  void restoreSession();
});
</script>
