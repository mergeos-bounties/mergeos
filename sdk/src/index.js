const jsonHeaders = { 'Content-Type': 'application/json' };

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
    const headers = {
      ...jsonHeaders,
      ...(this.token && options.auth !== false ? { Authorization: `Bearer ${this.token}` } : {}),
      ...(options.headers || {}),
    };
    const response = await this.fetchImpl(this.url(path), {
      method,
      headers,
      body: options.body === undefined ? undefined : JSON.stringify(options.body),
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

  projectAIWorkflow(projectID) {
    return this.request(`/api/projects/${encodeURIComponent(projectID)}/ai-workflow`);
  }

  projectTaskGraph(projectID) {
    return this.request(`/api/projects/${encodeURIComponent(projectID)}/task-graph`);
  }

  projectRepositoryScan(projectID) {
    return this.request(`/api/projects/${encodeURIComponent(projectID)}/repo-scan`);
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

  adminLedger() {
    return this.request('/api/admin/ledger');
  }

  creditMRG(payload) {
    return this.request('/api/admin/ledger/credits', { method: 'POST', body: payload });
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
