const jsonHeaders = { 'Content-Type': 'application/json' };

export const agentActionEventTypes = Object.freeze({
  review: 'agent.reviewed',
  test: 'agent.tested',
  generate: 'agent.generated',
  deploy: 'agent.deployed',
  scan: 'agent.scanned',
});

export const workflowEventTypes = Object.freeze({
  projectFunded: 'project.funded',
  taskCreated: 'task.created',
  taskClaimed: 'task.claimed',
  taskPaid: 'task.paid',
  prOpened: 'pr.opened',
  prReviewed: 'pr.reviewed',
  deploymentUpdated: 'deployment.updated',
  repoIssuesSynced: 'repo.issues.synced',
  ledgerRecorded: 'ledger.recorded',
  agentAction: 'agent.action',
});

export class MergeOSClient {
  constructor(options = {}) {
    this.baseURL = normalizeBaseURL(options.baseURL || '');
    this.token = options.token || '';
    this.fetchImpl = options.fetchImpl || globalThis.fetch;
    this.WebSocketImpl = options.WebSocketImpl || globalThis.WebSocket;
  }

  setToken(token = '') {
    this.token = token;
    return this;
  }

  async request(path, options = {}) {
    if (!this.fetchImpl) {
      throw new Error('fetch is not available; pass fetchImpl to MergeOSClient');
    }

    const method = options.method || 'GET';
    const hasRawBody = Object.hasOwn(options, 'rawBody');
    const headers = {
      ...(hasRawBody ? {} : jsonHeaders),
      ...(this.token && options.auth !== false ? { Authorization: `Bearer ${this.token}` } : {}),
      ...(options.headers || {}),
    };
    const response = await this.fetchImpl(this.url(path), {
      method,
      headers,
      body: hasRawBody ? options.rawBody : options.body === undefined ? undefined : JSON.stringify(options.body),
    });
    const text = await response.text();
    const payload = parseJSON(text);
    if (!response.ok) {
      const error = new Error(payload.error || response.statusText || 'MergeOS request failed');
      error.status = response.status;
      error.payload = payload;
      throw error;
    }
    return payload;
  }

  url(path) {
    const normalizedPath = String(path || '/');
    if (/^https?:\/\//i.test(normalizedPath)) return normalizedPath;
    return `${this.baseURL}${normalizedPath.startsWith('/') ? normalizedPath : `/${normalizedPath}`}`;
  }

  register(payload) {
    return this.request('/api/auth/register', { method: 'POST', body: payload, auth: false });
  }

  login(payload) {
    return this.request('/api/auth/login', { method: 'POST', body: payload, auth: false });
  }

  githubLogin(payload) {
    return this.request('/api/auth/github', { method: 'POST', body: payload, auth: false });
  }

  logout() {
    return this.request('/api/auth/logout', { method: 'POST', body: {} });
  }

  me() {
    return this.request('/api/auth/me');
  }

  publicMarketplace() {
    return this.request('/api/public/marketplace', { auth: false });
  }

  publicLedger() {
    return this.request('/api/public/ledger', { auth: false });
  }

  publicLedgerVerification() {
    return this.request('/api/public/ledger/verify', { auth: false });
  }

  publicLiveFeed(options = {}) {
    const limit = Number(options.limit) > 0 ? `?limit=${encodeURIComponent(Number(options.limit))}` : '';
    return this.request(`/api/public/live-feed${limit}`, { auth: false });
  }

  publicProtocolManifest() {
    return this.request('/api/public/protocol', { auth: false });
  }

  publicProtocolTasks(options = {}) {
    const limit = Number(options.limit) > 0 ? `?limit=${encodeURIComponent(Number(options.limit))}` : '';
    return this.request(`/api/public/protocol/tasks${limit}`, { auth: false });
  }

  publicProtocolAgents(options = {}) {
    const limit = Number(options.limit) > 0 ? `?limit=${encodeURIComponent(Number(options.limit))}` : '';
    return this.request(`/api/public/protocol/agents${limit}`, { auth: false });
  }

  publicProtocolEvents(options = {}) {
    const limit = Number(options.limit) > 0 ? `?limit=${encodeURIComponent(Number(options.limit))}` : '';
    return this.request(`/api/public/protocol/events${limit}`, { auth: false });
  }

  importRepoIssues(payload) {
    return this.request('/api/public/repo/issues', { method: 'POST', body: payload, auth: false });
  }

  publicTestSettingsStatus() {
    return this.request('/api/public/test-settings/status', { auth: false });
  }

  publicTestSettingsAuth(passwordOrPayload) {
    return this.request('/api/public/test-settings/auth', {
      method: 'POST',
      body: passwordPayload(passwordOrPayload),
      auth: false,
    });
  }

  publicTestSettingsEntries(passwordOrPayload) {
    return this.request('/api/public/test-settings/entries/list', {
      method: 'POST',
      body: passwordPayload(passwordOrPayload),
      auth: false,
    });
  }

  publicAddTestSettingsEntry(passwordOrPayload, payload = {}) {
    return this.request('/api/public/test-settings/entries', {
      method: 'POST',
      body: { ...passwordPayload(passwordOrPayload), ...payload },
      auth: false,
    });
  }

  publicUpdateTestSettingsEntry(entryID, passwordOrPayload, payload = {}) {
    return this.request(`/api/public/test-settings/entries/${encodeURIComponent(entryID)}`, {
      method: 'PATCH',
      body: { ...passwordPayload(passwordOrPayload), ...payload },
      auth: false,
    });
  }

  publicDeleteTestSettingsEntry(entryID, passwordOrPayload) {
    return this.request(`/api/public/test-settings/entries/${encodeURIComponent(entryID)}`, {
      method: 'DELETE',
      body: passwordPayload(passwordOrPayload),
      auth: false,
    });
  }

  publicRevealTestSettingsEntry(entryID, passwordOrPayload) {
    return this.request(`/api/public/test-settings/entries/${encodeURIComponent(entryID)}/reveal`, {
      method: 'POST',
      body: passwordPayload(passwordOrPayload),
      auth: false,
    });
  }

  createWallet(payload) {
    return this.request('/api/wallets', { method: 'POST', body: payload });
  }

  wallet(address) {
    return this.request(`/api/wallets/${encodeURIComponent(address)}`);
  }

  linkWallet(payload) {
    return this.request('/api/wallets/link', { method: 'POST', body: payload });
  }

  createPayPalOrder(payload) {
    return this.request('/api/payments/paypal/orders', { method: 'POST', body: payload });
  }

  uploadAttachment(body, options = {}) {
    return this.request('/api/uploads', {
      method: 'POST',
      rawBody: body,
      headers: options.headers || {},
    });
  }

  listProjects() {
    return this.request('/api/projects');
  }

  createProject(payload) {
    return this.request('/api/projects', { method: 'POST', body: payload });
  }

  evaluateProjectPrice(payload) {
    return this.request('/api/projects/evaluate-price', { method: 'POST', body: payload });
  }

  evaluateProjectWithLLM(payload) {
    return this.request('/api/projects/evaluate-llm', { method: 'POST', body: payload });
  }

  projectDeployment(projectID) {
    return this.request(`/api/projects/${encodeURIComponent(projectID)}/deployment`);
  }

  projectEscrow(projectID) {
    return this.request(`/api/projects/${encodeURIComponent(projectID)}/escrow`);
  }

  projectDashboard(projectID) {
    return this.request(`/api/projects/${encodeURIComponent(projectID)}/dashboard`);
  }

  projectPullRequests(projectID) {
    return this.request(`/api/projects/${encodeURIComponent(projectID)}/pull-requests`);
  }

  projectAIWorkflow(projectID) {
    return this.request(`/api/projects/${encodeURIComponent(projectID)}/ai-workflow`);
  }

  createProjectAgentAction(projectID, payload) {
    return this.request(`/api/projects/${encodeURIComponent(projectID)}/agent-actions`, { method: 'POST', body: payload });
  }

  projectTaskGraph(projectID) {
    return this.request(`/api/projects/${encodeURIComponent(projectID)}/task-graph`);
  }

  projectWorkflowProtocol(projectID) {
    return this.request(`/api/projects/${encodeURIComponent(projectID)}/protocol/workflow`);
  }

  projectRepositoryScan(projectID) {
    return this.request(`/api/projects/${encodeURIComponent(projectID)}/repo-scan`);
  }

  projectRepositoryScanProtocol(projectID) {
    return this.request(`/api/projects/${encodeURIComponent(projectID)}/protocol/scan`);
  }

  syncProjectRepoIssues(projectID) {
    return this.request(`/api/projects/${encodeURIComponent(projectID)}/repo-sync`, { method: 'POST' });
  }

  listTasks() {
    return this.request('/api/tasks');
  }

  acceptTask(taskID, payload) {
    return this.request(`/api/tasks/${encodeURIComponent(taskID)}/accept`, { method: 'POST', body: payload });
  }

  workerDashboard() {
    return this.request('/api/workers/me');
  }

  createDispute(payload) {
    return this.request('/api/disputes', { method: 'POST', body: payload });
  }

  ledger() {
    return this.request('/api/ledger');
  }

  adminSummary() {
    return this.request('/api/admin/summary');
  }

  adminOpsQueue() {
    return this.request('/api/admin/ops-queue');
  }

  adminReputation() {
    return this.request('/api/admin/reputation');
  }

  adminUsers() {
    return this.request('/api/admin/users');
  }

  updateAdminUser(userID, payload) {
    return this.request(`/api/admin/users/${encodeURIComponent(userID)}`, { method: 'PATCH', body: payload });
  }

  adminProjects() {
    return this.request('/api/admin/projects');
  }

  adminTasks() {
    return this.request('/api/admin/tasks');
  }

  adminTaskPullRequests(taskID) {
    return this.request(`/api/admin/tasks/${encodeURIComponent(taskID)}/pulls`);
  }

  mergeAdminTaskPullRequest(taskID, pullNumber, payload) {
    return this.request(`/api/admin/tasks/${encodeURIComponent(taskID)}/pulls/${encodeURIComponent(pullNumber)}/merge`, {
      method: 'POST',
      body: payload,
    });
  }

  adminNotifications() {
    return this.request('/api/admin/notifications');
  }

  adminAttachments() {
    return this.request('/api/admin/attachments');
  }

  adminLedger() {
    return this.request('/api/admin/ledger');
  }

  creditMRG(payload) {
    return this.request('/api/admin/ledger/credits', { method: 'POST', body: payload });
  }

  adminSettings() {
    return this.request('/api/admin/settings');
  }

  updateAdminSettings(payload) {
    return this.request('/api/admin/settings', { method: 'PATCH', body: payload });
  }

  adminSSLReviews() {
    return this.request('/api/admin/ssl');
  }

  reviewAdminSSL(payload) {
    return this.request('/api/admin/ssl/review', { method: 'POST', body: payload });
  }

  adminGeminiKeys() {
    return this.request('/api/admin/gemini/keys');
  }

  addAdminGeminiKey(payload) {
    return this.request('/api/admin/gemini/keys', { method: 'POST', body: payload });
  }

  updateAdminGeminiKey(keyID, payload) {
    return this.request(`/api/admin/gemini/keys/${encodeURIComponent(keyID)}`, { method: 'PATCH', body: payload });
  }

  testAdminGeminiKey(keyID, payload = {}) {
    return this.request(`/api/admin/gemini/keys/${encodeURIComponent(keyID)}/test`, { method: 'POST', body: payload });
  }

  adminGeminiWebhooks() {
    return this.request('/api/admin/gemini/webhooks');
  }

  adminTestSettings() {
    return this.request('/api/admin/test-settings');
  }

  updateAdminTestSettings(payload) {
    return this.request('/api/admin/test-settings', { method: 'PATCH', body: payload });
  }

  adminTestSettingsEntries() {
    return this.request('/api/admin/test-settings/entries');
  }

  addAdminTestSettingsEntry(payload) {
    return this.request('/api/admin/test-settings/entries', { method: 'POST', body: payload });
  }

  updateAdminTestSettingsEntry(entryID, payload) {
    return this.request(`/api/admin/test-settings/entries/${encodeURIComponent(entryID)}`, {
      method: 'PATCH',
      body: payload,
    });
  }

  deleteAdminTestSettingsEntry(entryID) {
    return this.request(`/api/admin/test-settings/entries/${encodeURIComponent(entryID)}`, { method: 'DELETE' });
  }

  webSocketURL(path = '/api/ws') {
    const normalizedPath = String(path || '/api/ws');
    if (/^wss?:\/\//i.test(normalizedPath)) return normalizedPath;
    if (!this.baseURL) return normalizedPath;
    const url = new URL(this.url(normalizedPath));
    url.protocol = url.protocol === 'https:' ? 'wss:' : 'ws:';
    return url.toString();
  }

  connectEvents(options = {}) {
    if (!this.WebSocketImpl) {
      throw new Error('WebSocket is not available; pass WebSocketImpl to MergeOSClient');
    }
    return new this.WebSocketImpl(this.webSocketURL(options.path), options.protocols);
  }
}

export function createMergeOSClient(options = {}) {
  return new MergeOSClient(options);
}

export function agentActionEventType(action = '') {
  const normalized = String(action || '').trim().toLowerCase();
  return agentActionEventTypes[normalized] || 'agent.action';
}

export function isAgentActionEventType(type = '') {
  const normalized = String(type || '').trim().toLowerCase();
  return normalized === 'agent.action' || Object.values(agentActionEventTypes).includes(normalized);
}

export function liveFeedTypeToProtocolEventType(type = '', action = '') {
  const normalized = String(type || '').trim().toLowerCase();
  if (normalized.startsWith('ledger_task_payment')) return workflowEventTypes.taskPaid;
  if (normalized.startsWith('ledger_')) return workflowEventTypes.ledgerRecorded;
  if (normalized === 'agent_action') return agentActionEventType(action);
  return {
    project_funded: workflowEventTypes.projectFunded,
    task_opened: workflowEventTypes.taskCreated,
    task_accepted: workflowEventTypes.taskClaimed,
    pr_opened: workflowEventTypes.prOpened,
    ai_review: workflowEventTypes.prReviewed,
    deployment_validation: workflowEventTypes.deploymentUpdated,
    repo_issues_synced: workflowEventTypes.repoIssuesSynced,
  }[normalized] || workflowEventTypes.agentAction;
}

export function protocolEventFromMessage(message = {}) {
  if (!message || typeof message !== 'object' || !message.event || typeof message.event !== 'object') {
    return null;
  }
  return message.event;
}

export function protocolTypeFromMessage(message = {}) {
  const event = protocolEventFromMessage(message);
  if (event && typeof event.type === 'string' && event.type.trim()) {
    return event.type.trim();
  }
  if (message && typeof message.protocol_type === 'string' && message.protocol_type.trim()) {
    return message.protocol_type.trim();
  }
  return liveFeedTypeToProtocolEventType(message?.type, message?.action);
}

export function protocolEventGroup(type = '') {
  const normalized = String(type || '').trim().toLowerCase();
  if (normalized.startsWith('agent.')) return 'agent';
  if (normalized.startsWith('pr.')) return 'pull_request';
  if (normalized.startsWith('task.')) return 'task';
  if (normalized.startsWith('project.')) return 'project';
  if (normalized.startsWith('deployment.')) return 'deployment';
  if (normalized.startsWith('repo.')) return 'repository';
  if (normalized.startsWith('ledger.')) return 'ledger';
  return 'unknown';
}

export function isWorkflowEventType(type = '') {
  const normalized = String(type || '').trim().toLowerCase();
  return Object.values(workflowEventTypes).includes(normalized) || isAgentActionEventType(normalized);
}

function normalizeBaseURL(value) {
  return String(value || '').replace(/\/+$/, '');
}

function parseJSON(text) {
  if (!text) return {};
  try {
    return JSON.parse(text);
  } catch {
    return { raw: text };
  }
}

function passwordPayload(passwordOrPayload) {
  if (passwordOrPayload && typeof passwordOrPayload === 'object') {
    return passwordOrPayload;
  }
  return { password: String(passwordOrPayload || '') };
}
