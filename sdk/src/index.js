import { createHash } from 'node:crypto';

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
  taskSubmitted: 'task.submitted',
  taskPaid: 'task.paid',
  prOpened: 'pr.opened',
  prReviewed: 'pr.reviewed',
  deploymentUpdated: 'deployment.updated',
  repoIssuesSynced: 'repo.issues.synced',
  proposalSubmitted: 'proposal.submitted',
  proposalAccepted: 'proposal.accepted',
  proposalDeclined: 'proposal.declined',
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

  runtimeConfig() {
    return this.request('/api/config', { auth: false });
  }

  publicLedger() {
    return this.request('/api/public/ledger', { auth: false });
  }

  publicLedgerVerification() {
    return this.request('/api/public/ledger/verify', { auth: false });
  }

  publicLedgerProof() {
    return this.request('/api/public/ledger/proof', { auth: false });
  }

  publicLedgerEvents(options = {}) {
    const limit = Number(options.limit) > 0 ? `?limit=${encodeURIComponent(Number(options.limit))}` : '';
    return this.request(`/api/public/ledger/events${limit}`, { auth: false });
  }

  publicTokenEconomy() {
    return this.request('/api/public/token-economy', { auth: false });
  }

  publicLiveFeed(options = {}) {
    const limit = Number(options.limit) > 0 ? `?limit=${encodeURIComponent(Number(options.limit))}` : '';
    return this.request(`/api/public/live-feed${limit}`, { auth: false });
  }

  publicProtocolManifest() {
    return this.request('/api/public/protocol', { auth: false });
  }

  publicProtocolTasks(options = {}) {
    const query = queryString({
      limit: Number(options.limit) > 0 ? Number(options.limit) : '',
      task_id: options.task_id || options.taskID || '',
    });
    return this.request(`/api/public/protocol/tasks${query}`, { auth: false });
  }

  publicProtocolAgentQueue(options = {}) {
    const query = queryString({
      limit: Number(options.limit) > 0 ? Number(options.limit) : '',
    });
    return this.request(`/api/public/protocol/agent-queue${query}`, { auth: false });
  }

  publicAgentRunbook(id = 'mergeide-agent.v1') {
    const runbookID = encodeURIComponent(String(id || 'mergeide-agent.v1'));
    return this.request(`/protocol/runbooks/${runbookID}.json`, { auth: false });
  }

  publicProtocolAgents(options = {}) {
    const limit = Number(options.limit) > 0 ? `?limit=${encodeURIComponent(Number(options.limit))}` : '';
    return this.request(`/api/public/protocol/agents${limit}`, { auth: false });
  }

  publicProtocolContributors(options = {}) {
    const limit = Number(options.limit) > 0 ? `?limit=${encodeURIComponent(Number(options.limit))}` : '';
    return this.request(`/api/public/protocol/contributors${limit}`, { auth: false });
  }

  publicProtocolLedger() {
    return this.request('/api/public/protocol/ledger', { auth: false });
  }

  publicMergeIDEWindowsRelease() {
    return this.request('/downloads/mergeide-windows-latest.json', { auth: false });
  }

  publicProtocolEvents(options = {}) {
    const limit = Number(options.limit) > 0 ? `?limit=${encodeURIComponent(Number(options.limit))}` : '';
    return this.request(`/api/public/protocol/events${limit}`, { auth: false });
  }

  publicProjectDeployment(projectID) {
    return this.request(`/api/public/projects/${encodeURIComponent(projectID)}/deployment`, { auth: false });
  }

  publicProjectAIWorkflow(projectID) {
    return this.request(`/api/public/projects/${encodeURIComponent(projectID)}/ai-workflow`, { auth: false });
  }

  publicProjectWorkflow(projectID) {
    return this.request(`/api/public/projects/${encodeURIComponent(projectID)}/workflow`, { auth: false });
  }

  publicProjectRepositoryScan(projectID) {
    return this.request(`/api/public/projects/${encodeURIComponent(projectID)}/repo-scan`, { auth: false });
  }

  publicProjectPullRequests(projectID) {
    return this.request(`/api/public/projects/${encodeURIComponent(projectID)}/pull-requests`, { auth: false });
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

  createWalletMigration(payload) {
    return this.request('/api/wallets/migrations', { method: 'POST', body: payload });
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

  projectPayouts(projectID) {
    return this.request(`/api/projects/${encodeURIComponent(projectID)}/payouts`);
  }

  projectAutoRelease(projectID, payload = {}) {
    return this.request(`/api/projects/${encodeURIComponent(projectID)}/auto-release`, { method: 'POST', body: payload });
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

  createProjectAgentAction(projectID, payload = {}) {
    return this.request(`/api/projects/${encodeURIComponent(projectID)}/agent-actions`, {
      method: 'POST',
      body: agentActionPayload(payload.action || 'review', payload),
    });
  }

  recordAgentReview(projectID, payload = {}) {
    return this.createProjectAgentAction(projectID, agentActionPayload('review', payload));
  }

  recordAgentTest(projectID, payload = {}) {
    return this.createProjectAgentAction(projectID, agentActionPayload('test', payload));
  }

  recordAgentGeneration(projectID, payload = {}) {
    return this.createProjectAgentAction(projectID, agentActionPayload('generate', payload));
  }

  recordDeployment(projectID, payload = {}) {
    return this.createProjectAgentAction(projectID, agentActionPayload('deploy', payload));
  }

  recordAgentScan(projectID, payload = {}) {
    return this.createProjectAgentAction(projectID, agentActionPayload('scan', payload));
  }

  projectTaskGraph(projectID) {
    return this.request(`/api/projects/${encodeURIComponent(projectID)}/task-graph`);
  }

  projectRouting(projectID) {
    return this.request(`/api/projects/${encodeURIComponent(projectID)}/routing`);
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

  claimTask(taskID, payload) {
    return this.request(`/api/tasks/${encodeURIComponent(taskID)}/claim`, { method: 'POST', body: payload });
  }

  submitTask(taskID, payload) {
    return this.request(`/api/tasks/${encodeURIComponent(taskID)}/submit`, { method: 'POST', body: payload });
  }

  workerDashboard() {
    return this.request('/api/workers/me');
  }

  createProposal(payload) {
    return this.request('/api/proposals', { method: 'POST', body: payload });
  }

  decideProposal(proposalID, payload) {
    return this.request(`/api/proposals/${encodeURIComponent(proposalID)}/decision`, { method: 'POST', body: payload });
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

export function contractReferenceFromLedger(entry, options = {}) {
  return formatContractReference(contractReferenceHex(entry), options);
}

export function contractReferenceBytes(entry) {
  return hexToBytes(contractReferenceHex(entry));
}

export function legacyWalletAddressHash(chain, address, options = {}) {
	const normalizedChain = normalizeLegacyChain(chain);
	const normalizedAddress = normalizeLegacyWalletAddress(address).toLowerCase();
	if (!normalizedAddress) {
		throw new Error('legacy wallet address is required');
	}
	return formatContractReference(sha256Hex(`mergeos:legacy-wallet:v1:${normalizedChain}:${normalizedAddress}`), options);
}

export function walletMigrationPDASeedMetadata(chain, address) {
  const legacyAddressHash = legacyWalletAddressHash(chain, address);
  return {
    pda_seeds: ['wallet-migration', normalizeLegacyChain(chain), 'legacy_address_hash_bytes'],
    pda_seed_formats: ['utf8', 'utf8', 'bytes32:hex_decode(contract.args.legacy_address_hash)'],
    legacy_address_hash: legacyAddressHash,
    legacy_address_hash_bytes: hexToBytes(legacyAddressHash),
  };
}

export function normalizeLegacyChain(value = '') {
	const normalized = String(value || '').trim().toLowerCase();
	if (normalized === 'trc20' || normalized === 'tron') return 'trc20';
	if (normalized === 'evm' || normalized === 'ethereum') return 'evm';
	throw new Error('legacy chain must be trc20 or evm');
}

export function normalizeLegacyWalletAddress(value = '') {
  let normalized = String(value || '').trim();
  for (const prefix of ['wallet:', 'tron:', 'trc20:', 'eip155:']) {
    if (normalized.toLowerCase().startsWith(prefix)) {
      normalized = normalized.slice(prefix.length).trim();
    }
  }
  if (/^0x[0-9a-f]{40}$/i.test(normalized)) return normalized.toLowerCase();
  return normalized;
}

export function agentActionEventType(action = '') {
  const normalized = String(action || '').trim().toLowerCase();
  return agentActionEventTypes[normalized] || 'agent.action';
}

function contractReferenceHex(entry) {
  if (entry === null || entry === undefined) {
    throw new Error('ledger entry is required');
  }
  if (typeof entry === 'string') {
    return contractReferenceFromValue(entry, 'value');
  }
  if (typeof entry !== 'object' || Array.isArray(entry)) {
    return sha256Hex(`mergeos:contract-reference:v1:value:${String(entry)}`);
  }

  for (const field of ['entry_hash', 'public_hash', 'hash']) {
    const value = String(entry[field] || '').trim();
    if (value) {
      return contractReferenceFromValue(value, field);
    }
  }
  if (String(entry.reference || '').trim()) {
    return sha256Hex(`mergeos:contract-reference:v1:reference:${String(entry.reference).trim()}`);
  }
  return sha256Hex(`mergeos:contract-reference:v1:ledger:${stableStringify(entry)}`);
}

function contractReferenceFromValue(value, field) {
  const normalized = String(value || '').trim();
  if (!normalized) {
    throw new Error('ledger reference value is required');
  }
  const hex = normalized.replace(/^0x/i, '');
  if (/^[0-9a-f]{64}$/i.test(hex)) {
    return hex.toLowerCase();
  }
  return sha256Hex(`mergeos:contract-reference:v1:${field}:${normalized}`);
}

function formatContractReference(hex, options = {}) {
  const format = String(options.format || 'hex').trim().toLowerCase();
  if (format === 'bytes' || format === 'array') return hexToBytes(hex);
  if (format === 'prefixed-hex' || format === '0x') return `0x${hex}`;
  return hex;
}

function hexToBytes(hex) {
  const normalized = String(hex || '').replace(/^0x/i, '').toLowerCase();
  if (!/^[0-9a-f]{64}$/.test(normalized)) {
    throw new Error('contract reference must be a 32-byte hex string');
  }
  return Array.from({ length: 32 }, (_, index) => Number.parseInt(normalized.slice(index * 2, index * 2 + 2), 16));
}

function sha256Hex(value) {
  return createHash('sha256').update(String(value)).digest('hex');
}

function stableStringify(value) {
  if (value === null || typeof value !== 'object') return JSON.stringify(value);
  if (Array.isArray(value)) return `[${value.map((item) => stableStringify(item)).join(',')}]`;
  return `{${Object.keys(value).sort().map((key) => `${JSON.stringify(key)}:${stableStringify(value[key])}`).join(',')}}`;
}

export function deploymentAgentActionPayload(payload = {}) {
  return agentActionPayload('deploy', payload);
}

export function agentActionPayload(action, payload = {}) {
  const normalizedAction = normalizeAgentAction(action);
  const referenceURL = payload.reference_url || payload.referenceURL || payload.deployment_url || payload.deploymentURL || payload.url || '';
  const durationMillis = payload.duration_millis ?? payload.durationMillis;
  const pullNumber = payload.pull_number ?? payload.pullNumber;
  const contextURLs = payload.context_urls || payload.contextURLs || payload.contextUrls || [];
  const evidence = payload.evidence || [];
  const runbook = payload.runbook || [];
  const checks = payload.checks || [];
  const body = {
    action: normalizedAction,
    agent_type: payload.agent_type || payload.agentType || defaultAgentTypeForAction(normalizedAction),
    status: payload.status || 'processed',
    reference_url: referenceURL,
    duration_millis: Number(durationMillis) > 0 ? Number(durationMillis) : 0,
    pull_number: Number(pullNumber) > 0 ? Number(pullNumber) : 0,
    labels: Array.isArray(payload.labels) ? payload.labels : [],
    context_urls: Array.isArray(contextURLs) ? contextURLs : [],
    evidence: Array.isArray(evidence) ? evidence : [],
    runbook: Array.isArray(runbook) ? runbook : [],
    checks: Array.isArray(checks) ? checks : [],
  };
  const claimID = payload.claim_id || payload.claimId || '';
  const bountyID = payload.bounty_id || payload.bountyId || '';
  if (claimID) body.claim_id = claimID;
  if (bountyID) body.bounty_id = bountyID;
  return body;
}

export function normalizeAgentAction(action = '') {
  const normalized = String(action || '').trim().toLowerCase();
  if (normalized === 'gen') return 'generate';
  if (['review', 'test', 'generate', 'deploy', 'scan'].includes(normalized)) return normalized;
  return 'review';
}

function defaultAgentTypeForAction(action = '') {
  return {
    review: 'review-agent',
    test: 'qa-agent',
    generate: 'coding-agent',
    deploy: 'deployment-agent',
    scan: 'scan-agent',
  }[action] || 'ai-agent';
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
    task_submitted: workflowEventTypes.taskSubmitted,
    task_accepted: workflowEventTypes.taskClaimed,
    proposal_submitted: workflowEventTypes.proposalSubmitted,
    proposal_accepted: workflowEventTypes.proposalAccepted,
    proposal_declined: workflowEventTypes.proposalDeclined,
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

export function protocolEventsFromMessage(message = {}) {
  const event = protocolEventFromMessage(message);
  if (event) return [event];
  const events = message && typeof message === 'object' ? message.events?.events : null;
  if (!Array.isArray(events)) return [];
  return events.filter((item) => item && typeof item === 'object');
}

export function protocolTypeFromMessage(message = {}) {
  const event = protocolEventFromMessage(message);
  if (event && typeof event.type === 'string' && event.type.trim()) {
    return event.type.trim();
  }
  if (message && typeof message.protocol_type === 'string' && message.protocol_type.trim()) {
    return message.protocol_type.trim();
  }
  const messageType = String(message?.type || '').trim();
  if (
    !messageType ||
    messageType === 'connection_ready' ||
    messageType === 'realtime_ready' ||
    messageType === 'live_feed_snapshot' ||
    messageType === 'realtime_snapshot' ||
    messageType === 'realtime_heartbeat' ||
    messageType === 'admin_ops_updated'
  ) {
    return '';
  }
  return liveFeedTypeToProtocolEventType(message?.type, message?.action);
}

export function protocolEventGroup(type = '') {
  const normalized = String(type || '').trim().toLowerCase();
  if (normalized.startsWith('agent.')) return 'agent';
  if (normalized.startsWith('pr.')) return 'pull_request';
  if (normalized.startsWith('task.')) return 'task';
  if (normalized.startsWith('proposal.')) return 'proposal';
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

function queryString(params = {}) {
  const search = new URLSearchParams();
  for (const [key, value] of Object.entries(params)) {
    if (value !== undefined && value !== null && String(value).trim() !== '') {
      search.set(key, String(value));
    }
  }
  const value = search.toString();
  return value ? `?${value}` : '';
}

function passwordPayload(passwordOrPayload) {
  if (passwordOrPayload && typeof passwordOrPayload === 'object') {
    return passwordOrPayload;
  }
  return { password: String(passwordOrPayload || '') };
}
