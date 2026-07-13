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
  taskChangesRequested: 'task.changes_requested',
  taskAccepted: 'task.accepted',
  taskPaid: 'task.paid',
  payoutReleased: 'payout.released',
  prOpened: 'pr.opened',
  prReviewed: 'pr.reviewed',
  prReadyForRelease: 'pr.ready_for_release',
  deploymentUpdated: 'deployment.updated',
  repoIssuesSynced: 'repo.issues.synced',
  proposalSubmitted: 'proposal.submitted',
  proposalAccepted: 'proposal.accepted',
  proposalDeclined: 'proposal.declined',
  ledgerRecorded: 'ledger.recorded',
  airdropClaimed: 'airdrop.claimed',
  presaleReserved: 'presale.reserved',
  walletMigrated: 'wallet.migrated',
  notificationsUpdated: 'notification.updated',
  paymentVerified: 'payment.verified',
  agentAction: 'agent.action',
  agentLeased: 'agent.leased',
  agentHeartbeat: 'agent.heartbeat',
  agentReleased: 'agent.released',
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
    const limit = publicLimitQuery(options.limit);
    return this.request(`/api/public/ledger/events${limit}`, { auth: false });
  }

  publicTokenEconomy() {
    return this.request('/api/public/token-economy', { auth: false });
  }

  publicAirdropMissions() {
    return this.request('/api/public/airdrop/missions', { auth: false });
  }

  publicLiveFeed(options = {}) {
    return this.request(`/api/public/live-feed${liveFeedQueryString(options)}`, { auth: false });
  }

  publicProtocolManifest() {
    return this.request('/api/public/protocol', { auth: false });
  }

  publicArchitectureManifest() {
    return this.request('/system/mergeos-architecture.v1.json', { auth: false });
  }

  async publicArchitectureDiscovery() {
    return architectureManifestDiscovery(await this.publicArchitectureManifest());
  }

  async publicProtocolDiscovery() {
    return protocolManifestDiscovery(await this.publicProtocolManifest());
  }

  publicProtocolTasks(options = {}) {
    const query = queryString({
      limit: finitePositiveLimit(options.limit),
      task_id: options.task_id || options.taskID || '',
    });
    return this.request(`/api/public/protocol/tasks${query}`, { auth: false });
  }

  publicProtocolAgentQueue(options = {}) {
    const query = queryString({
      limit: finitePositiveLimit(options.limit),
    });
    return this.request(`/api/public/protocol/agent-queue${query}`, { auth: false });
  }

  publicAgentRunbook(id = 'mergeide-agent.v1') {
    const runbookID = encodeURIComponent(String(id || 'mergeide-agent.v1'));
    return this.request(`/protocol/runbooks/${runbookID}.json`, { auth: false });
  }

  publicProtocolAgents(options = {}) {
    const limit = publicLimitQuery(options.limit);
    return this.request(`/api/public/protocol/agents${limit}`, { auth: false });
  }

  publicProtocolContributors(options = {}) {
    const limit = publicLimitQuery(options.limit);
    return this.request(`/api/public/protocol/contributors${limit}`, { auth: false });
  }

  publicProtocolLedger() {
    return this.request('/api/public/protocol/ledger', { auth: false });
  }

  publicMergeIDEWindowsRelease() {
    return this.request('/downloads/mergeide-windows-latest.json', { auth: false });
  }

  publicSolanaMRGContractProofManifest() {
    return this.request('/contracts/solana/mergeos_mrg.proof-manifest.v1.json', { auth: false });
  }

  publicProtocolEvents(options = {}) {
    return this.request(`/api/public/protocol/events${liveFeedQueryString(options)}`, { auth: false });
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

  createCardPaymentIntent(payload) {
    return this.request('/api/payments/card/intents', { method: 'POST', body: payload });
  }

  createAirdropClaim(payload) {
    return this.request('/api/airdrop/claims', { method: 'POST', body: airdropClaimPayload(payload) });
  }

  claimAirdrop(payload) {
    return this.createAirdropClaim(payload);
  }

  createPresaleReservation(payload) {
    return this.request('/api/presale/reservations', { method: 'POST', body: presaleReservationPayload(payload) });
  }

  reservePresale(payload) {
    return this.createPresaleReservation(payload);
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

  projectAutoReleaseFromPRMonitorTask(projectID, task = {}, overrides = {}) {
    return this.projectAutoRelease(projectID, autoReleasePayloadFromPRMonitorTask(task, overrides));
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

  createProjectAgentReviewFromPRMonitorTask(projectID, task = {}, overrides = {}) {
    const packet = task.review_packet && typeof task.review_packet === 'object' ? task.review_packet : {};
    const endpoint = overrides.review_endpoint || overrides.reviewEndpoint || packet.review_endpoint || `/api/projects/${encodeURIComponent(projectID)}/agent-actions`;
    return this.request(endpoint, { method: 'POST', body: agentReviewPayloadFromPRMonitorTask(task, overrides) });
  }

  createDeploymentValidationFromDeployment(projectID, deployment = {}, overrides = {}) {
    const packet = deployment.validation_packet && typeof deployment.validation_packet === 'object' ? deployment.validation_packet : {};
    const endpoint = overrides.validation_endpoint || overrides.validationEndpoint || packet.validation_endpoint || `/api/projects/${encodeURIComponent(projectID)}/agent-actions`;
    return this.request(endpoint, { method: 'POST', body: deploymentValidationPayloadFromDeployment(deployment, overrides) });
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

  executeRoutingPacket(routeOrPacket = {}, overrides = {}) {
    const packet = routingPacketFromRoute(routeOrPacket);
    const method = String(overrides.method || packet.method || 'POST').toUpperCase();
    const endpoint = overrides.endpoint || overrides.routing_endpoint || overrides.routingEndpoint || packet.endpoint || '';
    const options = { method };
    if (method !== 'GET') {
      options.body = routingPacketPayload(routeOrPacket, overrides);
    }
    return this.request(endpoint, options);
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

  createRepositorySuggestedTaskPayPalOrder(projectID, suggestedTaskID, payload = {}) {
    return this.request(
      `/api/projects/${encodeURIComponent(projectID)}/repo-scan/suggested-tasks/${encodeURIComponent(suggestedTaskID)}/paypal-order`,
      { method: 'POST', body: repositorySuggestedTaskPayPalOrderPayload(suggestedTaskID, payload) },
    );
  }

  repositorySuggestedTaskPayPalOrder(projectID, suggestedTaskID, payload = {}) {
    return this.createRepositorySuggestedTaskPayPalOrder(projectID, suggestedTaskID, payload);
  }

  fundRepositorySuggestedTask(projectID, suggestedTaskID, payload = {}) {
    return this.request(
      `/api/projects/${encodeURIComponent(projectID)}/repo-scan/suggested-tasks/${encodeURIComponent(suggestedTaskID)}/fund`,
      { method: 'POST', body: repositorySuggestedTaskFundingPayload(suggestedTaskID, payload) },
    );
  }

  fundSuggestedRepositoryTask(projectID, suggestedTaskID, payload = {}) {
    return this.fundRepositorySuggestedTask(projectID, suggestedTaskID, payload);
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

  claimAgentQueueTask(task = {}, overrides = {}) {
    const endpoint = overrides.claim_endpoint || overrides.claimEndpoint || task.claim_endpoint || task.work_packet?.claim_endpoint || '';
    const taskID = agentQueueTaskClaimID(task, endpoint);
    if (endpoint) {
      return this.request(endpoint, { method: 'POST', body: agentQueueClaimPayload(task, overrides) });
    }
    return this.claimTask(taskID, agentQueueClaimPayload(task, overrides));
  }

  createAgentQueueLease(taskOrPacket = {}, overrides = {}) {
    return this.request(agentLeaseEndpointFromWorkPacket(taskOrPacket, overrides), {
      method: 'POST',
      body: agentLeasePayload(taskOrPacket, overrides),
    });
  }

  heartbeatAgentQueueLease(leaseOrPacket = {}, overrides = {}) {
    return this.createAgentQueueLease(leaseOrPacket, { ...overrides, status: overrides.status || 'heartbeat' });
  }

  createAgentRunFromWorkPacket(workPacket = {}, action = 'generate', overrides = {}) {
    const runOverrides = { ...overrides, action };
    return this.request(agentRunEndpointFromWorkPacket(workPacket, runOverrides), {
      method: 'POST',
      body: agentRunPayloadFromWorkPacket(workPacket, action, overrides),
    });
  }

  submitTask(taskID, payload) {
    return this.request(`/api/tasks/${encodeURIComponent(taskID)}/submit`, { method: 'POST', body: payload });
  }

  requestTaskChanges(taskID, payload) {
    return this.request(`/api/tasks/${encodeURIComponent(taskID)}/request-changes`, { method: 'POST', body: payload });
  }

  workerDashboard() {
    return this.request('/api/workers/me');
  }

  createProposal(payload) {
    return this.request('/api/proposals', { method: 'POST', body: payload });
  }

  createProposalFromBounty(bounty = {}, overrides = {}) {
    const endpoint = overrides.proposal_endpoint
      || overrides.proposalEndpoint
      || bounty.proposal_endpoint
      || bounty.proposal_packet?.proposal_endpoint
      || bounty.claim_packet?.proposal_endpoint
      || '/api/proposals';
    return this.request(endpoint, { method: 'POST', body: proposalPayloadFromBounty(bounty, overrides) });
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

  adminDisputes() {
    return this.request('/api/admin/disputes');
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
    const path = options.path || `/api/ws${liveFeedQueryString(options)}`;
    return new this.WebSocketImpl(this.webSocketURL(path), options.protocols);
  }

  connectDiscoveredEvents(discoveryOrManifest = {}, options = {}) {
    if (!this.WebSocketImpl) {
      throw new Error('WebSocket is not available; pass WebSocketImpl to MergeOSClient');
    }
    const manifest = protocolManifestSource(discoveryOrManifest);
    const realtime = protocolManifestRealtime(manifest);
    const path = options.path || protocolManifestEventStreamPath(manifest, options);
    const protocols = options.protocols || (realtime.protocolVersion ? [realtime.protocolVersion] : undefined);
    return new this.WebSocketImpl(this.webSocketURL(path), protocols);
  }
}

export function createMergeOSClient(options = {}) {
  return new MergeOSClient(options);
}

export function architectureManifestDiscovery(manifest = {}) {
  const repositories = architectureManifestRepositories(manifest);
  const users = architectureManifestUsers(manifest);
  const publicURLs = architectureManifestPublicURLs(manifest);
  const aiWorkflow = architectureManifestAIWorkflow(manifest);
  return {
    manifest,
    protocolVersion: manifest.protocol_version || 'mergeos.architecture.v1',
    product: manifest.product || 'MergeOS',
    positioning: manifest.positioning || '',
    thesis: manifest.thesis || '',
    repositories,
    repositoryByName: Object.fromEntries(repositories.map((repo) => [repo.name, repo])),
    users,
    userByType: Object.fromEntries(users.map((user) => [user.type, user])),
    frontendSystem: manifest.frontend_system && typeof manifest.frontend_system === 'object' ? manifest.frontend_system : {},
    backendSystem: manifest.backend_system && typeof manifest.backend_system === 'object' ? manifest.backend_system : {},
    aiLayer: manifest.ai_layer && typeof manifest.ai_layer === 'object' ? manifest.ai_layer : {},
    aiWorkflow,
    marketplaceSystem: manifest.marketplace_system && typeof manifest.marketplace_system === 'object' ? manifest.marketplace_system : {},
    publicURLs,
  };
}

export function architectureManifestRepositories(manifest = {}) {
  return (Array.isArray(manifest.repository_architecture) ? manifest.repository_architecture : [])
    .filter((repo) => repo && typeof repo === 'object')
    .map((repo) => ({
      name: repo.name || '',
      role: repo.role || '',
      contains: normalizeStringList(repo.contains),
      raw: repo,
    }));
}

export function architectureManifestUsers(manifest = {}) {
  return (Array.isArray(manifest.users) ? manifest.users : [])
    .filter((user) => user && typeof user === 'object')
    .map((user) => ({
      type: user.type || '',
      examples: normalizeStringList(user.examples),
      dashboard: user.dashboard || '',
      surface: user.surface || '',
      raw: user,
    }));
}

export function architectureManifestAIWorkflow(manifest = {}) {
  return normalizeStringList(manifest.ai_layer?.workflow);
}

export function architectureManifestPublicURLs(manifest = {}) {
  const urls = manifest.public_urls;
  return urls && typeof urls === 'object' ? { ...urls } : {};
}

export function architectureManifestPublicURL(manifest = {}, key = '') {
  return architectureManifestPublicURLs(manifest)[key] || '';
}

export function protocolManifestDiscovery(manifest = {}) {
  const documents = protocolManifestDocuments(manifest);
  const endpoints = protocolManifestEndpoints(manifest);
  const contextURLs = protocolManifestContextURLs(manifest);
  const realtime = protocolManifestRealtime(manifest);
  return {
    manifest,
    protocolVersion: manifest.protocol_version || 'mergeos.protocol.manifest.v1',
    status: manifest.status || '',
    generatedAt: manifest.generated_at || '',
    stats: {
      schemaCount: Number(manifest.stats?.schema_count) || documents.length,
      publicEndpointCount: Number(manifest.stats?.public_endpoint_count) || endpoints.length,
      agentContextURLCount: Number(manifest.stats?.agent_context_url_count) || Object.keys(contextURLs).length,
      realtimeStreamCount: Number(manifest.stats?.realtime_stream_count) || (realtime.websocketPath ? 1 : 0),
    },
    documents,
    endpoints,
    contextURLs,
    realtime,
    documentByVersion: Object.fromEntries(documents.map((document) => [document.protocolVersion, document])),
    endpointByID: Object.fromEntries(endpoints.map((endpoint) => [endpoint.id, endpoint])),
  };
}

export function protocolManifestDocuments(manifest = {}) {
  const docs = Array.isArray(manifest.documents) ? manifest.documents : [];
  const schemas = Array.isArray(manifest.schemas) ? manifest.schemas : [];
  const source = docs.length ? docs : schemas;
  return source
    .filter((item) => item && typeof item === 'object')
    .map((item, index) => ({
      protocolVersion: item.protocol_version || item.version || '',
      kind: item.kind || '',
      title: item.title || titleFromProtocolKind(item.kind || item.version || `document_${index + 1}`),
      schemaURL: item.schema_url || '',
      publicEndpoint: item.public_endpoint || '',
      description: item.description || '',
      raw: item,
    }));
}

export function protocolManifestEndpoints(manifest = {}) {
  return (Array.isArray(manifest.endpoints) ? manifest.endpoints : [])
    .filter((item) => item && typeof item === 'object')
    .map((item) => ({
      id: item.id || endpointID(item),
      method: item.method || 'GET',
      path: item.path || '/',
      protocolVersion: item.protocol_version || item.protocol || '',
      protocol: item.protocol || item.protocol_version || '',
      auth: item.auth || '',
      access: item.access || accessFromEndpointAuth(item.auth),
      category: item.category || categoryFromEndpoint(item),
      description: item.description || '',
      raw: item,
    }));
}

export function protocolManifestContextURLs(manifest = {}) {
  const urls = manifest.agent_context?.context_urls;
  return urls && typeof urls === 'object' ? { ...urls } : {};
}

export function protocolManifestRealtime(manifest = {}) {
  const source = protocolManifestSource(manifest);
  const realtime = source.realtime && typeof source.realtime === 'object' ? source.realtime : {};
  return {
    protocolVersion: realtime.protocol_version || 'mergeos.event.v1',
    websocketPath: realtime.websocket_path || '/api/ws',
    readyEvent: realtime.ready_event || 'realtime_ready',
    snapshotEvent: realtime.snapshot_event || 'realtime_snapshot',
    heartbeatEvent: realtime.heartbeat_event || 'realtime_heartbeat',
    topics: Array.isArray(realtime.topics) ? realtime.topics.filter(Boolean) : [],
    raw: realtime,
  };
}

export function protocolManifestEventStreamPath(discoveryOrManifest = {}, options = {}) {
  const realtime = protocolManifestRealtime(discoveryOrManifest);
  return appendQueryString(realtime.websocketPath, liveFeedQueryString(options));
}

export function protocolManifestDocument(manifest = {}, selector = '') {
  const normalized = normalizeSelector(selector);
  return protocolManifestDocuments(manifest).find((document) => (
    normalizeSelector(document.protocolVersion) === normalized
    || normalizeSelector(document.kind) === normalized
    || normalizeSelector(document.title) === normalized
  )) || null;
}

export function protocolManifestEndpoint(manifest = {}, selector = '') {
  const normalized = normalizeSelector(selector);
  return protocolManifestEndpoints(manifest).find((endpoint) => (
    normalizeSelector(endpoint.id) === normalized
    || normalizeSelector(endpoint.protocolVersion) === normalized
    || normalizeSelector(endpoint.protocol) === normalized
    || normalizeSelector(endpoint.path) === normalized
    || normalizeSelector(endpoint.category) === normalized
  )) || null;
}

export function protocolManifestContextURL(manifest = {}, key = '', params = {}) {
  const normalized = normalizeSelector(key);
  const urls = protocolManifestContextURLs(manifest);
  const entry = Object.entries(urls).find(([candidate]) => normalizeSelector(candidate) === normalized);
  if (!entry) return '';
  return fillPathTemplate(entry[1], params);
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

export function airdropClaimPayload(payload = {}) {
  return compactPayload({
    mission_id: firstTokenWorkflowValue(payload, ['mission_id', 'missionID', 'mission']),
    wallet_address: normalizeSolanaWalletAddress(firstTokenWorkflowValue(payload, ['wallet_address', 'walletAddress', 'wallet'])),
    allocation_mrg: tokenWorkflowInteger(firstTokenWorkflowValue(payload, ['allocation_mrg', 'allocationMRG', 'allocation']), 250),
    worker_id: firstTokenWorkflowValue(payload, ['worker_id', 'workerID', 'worker']),
    task_reference: firstTokenWorkflowValue(payload, ['task_reference', 'taskReference', 'task']),
    proof_url: firstTokenWorkflowValue(payload, ['proof_url', 'proofURL', 'proofUrl', 'proof']),
    proof_signals: tokenWorkflowList(firstTokenWorkflowValue(payload, ['proof_signals', 'proofSignals', 'signals'], [])),
    notes: firstTokenWorkflowValue(payload, ['notes', 'note']),
  });
}

export function presaleReservationPayload(payload = {}) {
  return compactPayload({
    tier: firstTokenWorkflowValue(payload, ['tier'], 'builder'),
    wallet_address: normalizeSolanaWalletAddress(firstTokenWorkflowValue(payload, ['wallet_address', 'walletAddress', 'wallet'])),
    reserve_mrg: tokenWorkflowInteger(firstTokenWorkflowValue(payload, ['reserve_mrg', 'reserveMRG', 'reserve', 'amountMRG']), 25000),
    funding_rail: firstTokenWorkflowValue(payload, ['funding_rail', 'fundingRail', 'rail'], 'solana'),
    funding_reference: firstTokenWorkflowValue(payload, ['funding_reference', 'fundingReference', 'reference']),
    notes: firstTokenWorkflowValue(payload, ['notes', 'note']),
  });
}

export function repositorySuggestedTaskFundingPayload(suggestedTaskID, payload = {}) {
  return compactPayload({
    suggested_task_id: firstPayloadValue(payload, ['suggested_task_id', 'suggestedTaskID', 'task_id', 'taskID'], suggestedTaskID),
    reward_cents: integerPayloadValue(firstPayloadValue(payload, ['reward_cents', 'rewardCents', 'reward']), 0),
    budget_cents: integerPayloadValue(firstPayloadValue(payload, ['budget_cents', 'budgetCents', 'budget']), 0),
    payment_method: firstPayloadValue(payload, ['payment_method', 'paymentMethod', 'method'], 'card'),
    payment_reference: firstPayloadValue(payload, ['payment_reference', 'paymentReference', 'reference']),
  });
}

export function repositorySuggestedTaskPayPalOrderPayload(suggestedTaskID, payload = {}) {
  return compactPayload({
    suggested_task_id: firstPayloadValue(payload, ['suggested_task_id', 'suggestedTaskID', 'task_id', 'taskID'], suggestedTaskID),
    reward_cents: integerPayloadValue(firstPayloadValue(payload, ['reward_cents', 'rewardCents', 'reward']), 0),
    budget_cents: integerPayloadValue(firstPayloadValue(payload, ['budget_cents', 'budgetCents', 'budget']), 0),
    return_url: firstPayloadValue(payload, ['return_url', 'returnURL', 'returnUrl']),
    cancel_url: firstPayloadValue(payload, ['cancel_url', 'cancelURL', 'cancelUrl']),
  });
}

export function normalizeSolanaWalletAddress(value = '') {
  return String(value || '').trim();
}

export function isLikelySolanaWallet(value = '') {
  return /^[1-9A-HJ-NP-Za-km-z]{32,44}$/.test(normalizeSolanaWalletAddress(value));
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

export function deploymentValidationPayloadFromDeployment(deployment = {}, overrides = {}) {
  const packet = deployment.validation_packet && typeof deployment.validation_packet === 'object' ? deployment.validation_packet : {};
  const packetPayload = packet.payload && typeof packet.payload === 'object' ? packet.payload : {};
  const targetStage = packet.target_stage && typeof packet.target_stage === 'object' ? packet.target_stage : {};
  const contextURLs = overrides.context_urls || overrides.contextURLs || packetPayload.context_urls || packet.context_urls || [];
  const evidence = overrides.evidence || packetPayload.evidence || ['deployment_handoff', 'release_gate', 'preview_health'];
  const runbook = overrides.runbook || packetPayload.runbook || packet.runbook || [];
  const checks = overrides.checks || packetPayload.checks || packet.stage_checks || [];
  return agentActionPayload('deploy', {
    ...packetPayload,
    ...overrides,
    bounty_id: overrides.bounty_id || overrides.bountyId || packetPayload.bounty_id || targetStage.bounty_id || '',
    agent_type: overrides.agent_type || overrides.agentType || packetPayload.agent_type || 'deployment-agent',
    delegated_by: overrides.delegated_by || overrides.delegatedBy || packetPayload.delegated_by || 'ceo-strategy-agent',
    subagent_type: overrides.subagent_type || overrides.subagentType || packetPayload.subagent_type || 'deployment-agent',
    status: overrides.status || packetPayload.status || packet.status || 'processed',
    reference_url: overrides.reference_url || overrides.referenceURL || packetPayload.reference_url || targetStage.url || '',
    context_urls: contextURLs,
    evidence,
    runbook,
    checks,
    delegation_chain: overrides.delegation_chain || overrides.delegationChain || packetPayload.delegation_chain || ['ceo-strategy-agent', 'deployment-agent'],
  });
}

export function deploymentValidationOutputContracts(deployment = {}, action = '') {
  const packet = deployment.validation_packet && typeof deployment.validation_packet === 'object' ? deployment.validation_packet : {};
  const contracts = Array.isArray(packet.output_contracts) ? packet.output_contracts.filter((contract) => contract && typeof contract === 'object') : [];
  const normalizedAction = String(action || '').trim().toLowerCase();
  if (!normalizedAction) return contracts;
  return contracts.filter((contract) => String(contract.action || '').trim().toLowerCase() === normalizedAction);
}

export function aiWorkflowStages(workflow = {}, status = '') {
  const stages = Array.isArray(workflow.stages) ? workflow.stages : [];
  const normalizedStatus = normalizeSelector(status);
  const rows = stages.filter((stage) => stage && typeof stage === 'object');
  if (!normalizedStatus) return rows;
  return rows.filter((stage) => normalizeSelector(stage.status) === normalizedStatus);
}

export function aiWorkflowStage(workflow = {}, selector = '') {
  const normalized = normalizeSelector(selector || workflow.current_step || workflow.currentStep || '');
  if (!normalized) return aiWorkflowCurrentStage(workflow);
  return aiWorkflowStages(workflow).find((stage) => (
    normalizeSelector(stage.id) === normalized
    || normalizeSelector(stage.title) === normalized
    || normalizeSelector(stage.artifact_kind) === normalized
    || normalizeSelector(stage.output_protocol) === normalized
  )) || null;
}

export function aiWorkflowCurrentStage(workflow = {}) {
  const stages = aiWorkflowStages(workflow);
  const current = normalizeSelector(workflow.current_step || workflow.currentStep || '');
  return stages.find((stage) => normalizeSelector(stage.id) === current)
    || stages.find((stage) => normalizeSelector(stage.status) === 'in-progress')
    || stages.find((stage) => normalizeSelector(stage.status) === 'pending')
    || stages[stages.length - 1]
    || null;
}

export function aiWorkflowStageContextURLs(stageOrWorkflow = {}, selector = '') {
  const stage = selector ? aiWorkflowStage(stageOrWorkflow, selector) : stageOrWorkflow;
  const urls = stage?.context_urls || stage?.contextURLs;
  return urls && typeof urls === 'object' && !Array.isArray(urls) ? { ...urls } : {};
}

export function aiWorkflowStageActionContract(stageOrWorkflow = {}, selector = '') {
  const stage = selector ? aiWorkflowStage(stageOrWorkflow, selector) : stageOrWorkflow;
  if (!stage || typeof stage !== 'object') return null;
  return compactPayload({
    stage_id: stage.id,
    title: stage.title,
    status: stage.status,
    artifact_kind: stage.artifact_kind || stage.artifactKind,
    input_endpoint: stage.input_endpoint || stage.inputEndpoint,
    output_endpoint: stage.output_endpoint || stage.outputEndpoint,
    output_protocol: stage.output_protocol || stage.outputProtocol,
    output_protocol_url: stage.output_protocol_url || stage.outputProtocolURL,
    action_endpoint: stage.action_endpoint || stage.actionEndpoint,
    context_urls: aiWorkflowStageContextURLs(stage),
    checklist: Array.isArray(stage.checklist) ? stage.checklist.filter(Boolean) : [],
    output_ids: Array.isArray(stage.output_ids) ? stage.output_ids.filter(Boolean) : [],
    produced_count: Number(stage.produced_count ?? stage.producedCount) || 0,
  });
}

export function agentProtocolAgents(document = {}, filters = {}) {
  const agents = Array.isArray(document.agents) ? document.agents : Array.isArray(document) ? document : [];
  return agents
    .filter((agent) => agent && typeof agent === 'object')
    .filter((agent) => agentMatchesProtocolFilters(agent, filters))
    .sort(compareAgentProtocolFit);
}

export function agentSupportsAction(agent = {}, action = '') {
  const normalized = normalizeAgentAction(action);
  const actions = normalizeStringList(agent.supported_actions || agent.supportedActions);
  if (!actions.length) return false;
  return actions.map(normalizeAgentAction).includes(normalized);
}

export function agentHasCapability(agent = {}, capability = '') {
  const normalized = normalizeSelector(capability);
  if (!normalized) return true;
  const capabilities = normalizeStringList(agent.capabilities);
  const tags = normalizeStringList(agent.tags);
  return [...capabilities, ...tags].some((value) => normalizeSelector(value) === normalized);
}

export function bestAgentForAction(document = {}, action = '', filters = {}) {
  const agents = agentProtocolAgents(document, { ...filters, action });
  return agents[0] || null;
}

export function agentActionPayload(action, payload = {}) {
  const normalizedAction = normalizeAgentAction(action);
  const referenceURL = payload.reference_url || payload.referenceURL || payload.deployment_url || payload.deploymentURL || payload.url || '';
  const durationMillis = payload.duration_millis ?? payload.durationMillis;
  const pullNumber = payload.pull_number ?? payload.pullNumber;
  const contextURLs = payload.context_urls || payload.contextURLs || payload.contextUrls || [];
  const evidence = payload.evidence || payload.evidence_required || payload.evidenceRequired || [];
  const runbook = payload.runbook || [];
  const checks = payload.checks || [];
  const delegatedBy = payload.delegated_by || payload.delegatedBy || '';
  const designAgent = payload.design_agent || payload.designAgent || '';
  const subagentType = payload.subagent_type || payload.subagentType || '';
  const delegationChain = payload.delegation_chain || payload.delegationChain || [];
  const body = {
    action: normalizedAction,
    agent_type: payload.agent_type || payload.agentType || defaultAgentTypeForAction(normalizedAction),
    status: payload.status || 'processed',
    reference_url: referenceURL,
    duration_millis: nonNegativeInteger(durationMillis),
    pull_number: positiveInteger(pullNumber),
    labels: Array.isArray(payload.labels) ? payload.labels : [],
    context_urls: normalizeContextURLList(contextURLs),
    evidence: normalizeStringList(evidence),
    runbook: normalizeRunbookList(runbook),
    checks: Array.isArray(checks) ? checks : [],
  };
  if (delegatedBy) body.delegated_by = delegatedBy;
  if (designAgent) body.design_agent = designAgent;
  if (subagentType) body.subagent_type = subagentType;
  if (Array.isArray(delegationChain) && delegationChain.length) body.delegation_chain = delegationChain;
  const sourceFindingID = payload.source_finding_id || payload.sourceFindingID || '';
  const signal = payload.signal || '';
  const sourcePath = payload.path || payload.source_path || payload.sourcePath || '';
  if (sourceFindingID) body.source_finding_id = sourceFindingID;
  if (signal) body.signal = signal;
  if (sourcePath) body.path = sourcePath;
  const claimID = payload.claim_id || payload.claimId || '';
  const bountyID = payload.bounty_id || payload.bountyId || '';
  if (claimID) body.claim_id = claimID;
  if (bountyID) body.bounty_id = bountyID;
  return body;
}

export function agentActionPayloadFromWorkPacket(workPacket = {}, action = 'review', overrides = {}) {
  const normalizedAction = normalizeAgentAction(action);
  const actionPayloads = Array.isArray(workPacket.action_payloads) ? workPacket.action_payloads : [];
  const selected = actionPayloads.find((item) => normalizeAgentAction(item?.action || item?.body?.action) === normalizedAction) || actionPayloads[0] || {};
  const body = selected.body && typeof selected.body === 'object' ? selected.body : {};
  const contextURLs = overrides.context_urls || overrides.contextURLs || overrides.contextUrls || body.context_urls || workPacket.context_urls || {};
  return agentActionPayload(body.action || selected.action || normalizedAction, {
    ...body,
    ...overrides,
    context_urls: contextURLs,
    runbook: overrides.runbook || body.runbook || workPacket.runbook || [],
    delegated_by: overrides.delegated_by || overrides.delegatedBy || body.delegated_by || workPacket.supervisor_agent_type,
    design_agent: overrides.design_agent || overrides.designAgent || body.design_agent || workPacket.design_review_agent,
    subagent_type: overrides.subagent_type || overrides.subagentType || body.subagent_type || workPacket.subagent_type,
    delegation_chain: overrides.delegation_chain || overrides.delegationChain || body.delegation_chain || workPacket.delegation_chain || [],
  });
}

export function agentRunEndpointFromWorkPacket(workPacket = {}, overrides = {}) {
  return overrides.run_endpoint
    || overrides.runEndpoint
    || workPacket.run_endpoint
    || workPacket.runEndpoint
    || selectedAgentRunPayload(workPacket, overrides.action || overrides.agentAction || '')?.endpoint
    || '';
}

export function agentRunPayloadFromWorkPacket(workPacket = {}, action = 'generate', overrides = {}) {
  const normalizedAction = normalizeAgentAction(action || overrides.action || overrides.agentAction);
  const selected = selectedAgentRunPayload(workPacket, normalizedAction);
  const body = selected.body && typeof selected.body === 'object' ? selected.body : {};
  const contextURLs = overrides.context_urls || overrides.contextURLs || overrides.contextUrls || body.context_urls || workPacket.context_urls || {};
  return compactPayload({
    ...body,
    ...overrides,
    action: normalizeAgentAction(overrides.action || overrides.agentAction || body.action || selected.action || normalizedAction),
    claim_id: overrides.claim_id || overrides.claimId || body.claim_id || workPacket.claim_id || '',
    bounty_id: overrides.bounty_id || overrides.bountyId || body.bounty_id || body.claim_id || workPacket.bounty_id || workPacket.claim_id || '',
    agent_type: overrides.agent_type || overrides.agentType || body.agent_type || workPacket.subagent_type || '',
    base_branch: overrides.base_branch || overrides.baseBranch || body.base_branch || 'main',
    objective: overrides.objective || body.objective || '',
    context_urls: normalizeContextURLList(contextURLs),
  });
}

function selectedAgentRunPayload(workPacket = {}, action = '') {
  const normalizedAction = normalizeAgentAction(action);
  const runPayloads = Array.isArray(workPacket.run_payloads) ? workPacket.run_payloads : [];
  return runPayloads.find((item) => normalizeAgentAction(item?.action || item?.body?.action) === normalizedAction) || runPayloads[0] || {};
}

export function agentWorkPacketOutputContracts(workPacket = {}, action = '') {
  const contracts = Array.isArray(workPacket.output_contracts) ? workPacket.output_contracts : [];
  const normalizedAction = String(action || '').trim().toLowerCase();
  const rows = contracts.filter((contract) => contract && typeof contract === 'object');
  if (!normalizedAction) return rows;
  return rows.filter((contract) => String(contract.action || '').trim().toLowerCase() === normalizedAction);
}

export function repoPlanningPacket(document = {}) {
  if (!document || typeof document !== 'object') return {};
  const packet = document.planning_packet && typeof document.planning_packet === 'object'
    ? document.planning_packet
    : document;
  return packet && typeof packet === 'object' && !Array.isArray(packet) ? packet : {};
}

export function repoPlanningSteps(document = {}, status = '') {
  const packet = repoPlanningPacket(document);
  const steps = Array.isArray(packet.steps) ? packet.steps.filter((step) => step && typeof step === 'object') : [];
  const normalizedStatus = String(status || '').trim().toLowerCase();
  if (!normalizedStatus) return steps;
  return steps.filter((step) => String(step.status || '').trim().toLowerCase() === normalizedStatus);
}

export function repoPlanningOutputContracts(document = {}, selector = '') {
  const packet = repoPlanningPacket(document);
  const contracts = Array.isArray(packet.output_contracts) ? packet.output_contracts.filter((contract) => contract && typeof contract === 'object') : [];
  const normalizedSelector = String(selector || '').trim().toLowerCase();
  if (!normalizedSelector) return contracts;
  return contracts.filter((contract) => {
    const action = String(contract.action || '').trim().toLowerCase();
    const protocol = String(contract.output_protocol || '').trim().toLowerCase();
    return action === normalizedSelector || protocol === normalizedSelector;
  });
}

export function agentReviewPayloadFromPRMonitorTask(task = {}, overrides = {}) {
  const packet = task.review_packet && typeof task.review_packet === 'object' ? task.review_packet : {};
  const packetPayload = packet.payload && typeof packet.payload === 'object' ? packet.payload : {};
  const pullRequest = packet.pull_request && typeof packet.pull_request === 'object'
    ? packet.pull_request
    : Array.isArray(task.pull_requests)
      ? task.pull_requests[0] || {}
      : {};
  return agentActionPayload('review', {
    ...packetPayload,
    ...overrides,
    claim_id: overrides.claim_id || overrides.claimId || packetPayload.claim_id || task.task_id || '',
    bounty_id: overrides.bounty_id || overrides.bountyId || packetPayload.bounty_id || '',
    agent_type: overrides.agent_type || overrides.agentType || packetPayload.agent_type || 'review-agent',
    delegated_by: overrides.delegated_by || overrides.delegatedBy || packetPayload.delegated_by || 'ceo-strategy-agent',
    subagent_type: overrides.subagent_type || overrides.subagentType || packetPayload.subagent_type || 'review-agent',
    status: overrides.status || packetPayload.status || packet.status || 'processed',
    pull_number: overrides.pull_number ?? overrides.pullNumber ?? packetPayload.pull_number ?? pullRequest.number,
    reference_url: overrides.reference_url || overrides.referenceURL || packetPayload.reference_url || pullRequest.url || pullRequest.html_url || '',
    labels: Array.isArray(overrides.labels) ? overrides.labels : Array.isArray(packetPayload.labels) ? packetPayload.labels : Array.isArray(pullRequest.labels) ? pullRequest.labels : [],
    context_urls: overrides.context_urls || overrides.contextURLs || packetPayload.context_urls || packet.context_urls || [],
    evidence: overrides.evidence || packetPayload.evidence || pullRequest.readiness?.signals || [],
    runbook: overrides.runbook || packetPayload.runbook || packet.runbook || [],
    checks: overrides.checks || packetPayload.checks || [],
    delegation_chain: overrides.delegation_chain || overrides.delegationChain || packetPayload.delegation_chain || ['ceo-strategy-agent', 'review-agent'],
  });
}

export function agentQueueClaimPayload(task = {}, overrides = {}) {
  const workPacket = task.work_packet && typeof task.work_packet === 'object' ? task.work_packet : {};
  const workerKind = overrides.worker_kind || overrides.workerKind || task.worker_kind || task.required_worker_kind || 'agent';
  const agentType = overrides.agent_type || overrides.agentType || task.agent_type || workPacket.subagent_type || '';
  const workerID = overrides.worker_id || overrides.workerID || '';
  const payoutAccount = overrides.payout_account || overrides.payoutAccount || '';
  const body = compactPayload({
    worker_kind: workerKind,
    worker_id: workerID,
    agent_type: agentType,
    payout_account: payoutAccount,
  });
  if (Array.isArray(overrides.labels) && overrides.labels.length) body.labels = overrides.labels;
  return body;
}

export function agentQueueTaskClaimID(task = {}, endpoint = '') {
  const explicit = task.bounty_id || task.claim_id || task.claimID || task.id || '';
  if (explicit) return String(explicit);
  const source = endpoint || task.claim_endpoint || task.work_packet?.claim_endpoint || '';
  const match = String(source).match(/\/api\/tasks\/([^/]+)\/claim(?:\?|$)/);
  return match ? decodeURIComponent(match[1]) : '';
}

export function agentLeasePacketFromWorkPacket(taskOrPacket = {}) {
  if (!taskOrPacket || typeof taskOrPacket !== 'object') return {};
  if (taskOrPacket.lease_packet && typeof taskOrPacket.lease_packet === 'object') return taskOrPacket.lease_packet;
  if (taskOrPacket.work_packet?.lease_packet && typeof taskOrPacket.work_packet.lease_packet === 'object') return taskOrPacket.work_packet.lease_packet;
  return taskOrPacket;
}

export function agentLeaseEndpointFromWorkPacket(taskOrPacket = {}, overrides = {}) {
  const packet = agentLeasePacketFromWorkPacket(taskOrPacket);
  return overrides.lease_endpoint
    || overrides.leaseEndpoint
    || overrides.heartbeat_endpoint
    || overrides.heartbeatEndpoint
    || packet.lease_endpoint
    || packet.heartbeat_endpoint
    || '/api/agent-queue/leases';
}

export function agentLeasePayload(taskOrPacket = {}, overrides = {}) {
  const packet = agentLeasePacketFromWorkPacket(taskOrPacket);
  const packetPayload = packet.payload && typeof packet.payload === 'object' ? packet.payload : {};
  const claimID = overrides.claim_id
    || overrides.claimId
    || packetPayload.claim_id
    || taskOrPacket.claim_id
    || taskOrPacket.claimID
    || taskOrPacket.bounty_id
    || taskOrPacket.id
    || '';
  const bountyID = overrides.bounty_id || overrides.bountyId || packetPayload.bounty_id || claimID;
  return compactPayload({
    lease_id: overrides.lease_id || overrides.leaseId || taskOrPacket.lease_id || taskOrPacket.leaseID || '',
    claim_id: claimID,
    bounty_id: bountyID,
    agent_type: overrides.agent_type || overrides.agentType || packetPayload.agent_type || taskOrPacket.agent_type || taskOrPacket.work_packet?.subagent_type || '',
    status: overrides.status || packetPayload.status || taskOrPacket.status || 'leased',
  });
}

export function routingPacketFromRoute(routeOrPacket = {}) {
  if (!routeOrPacket || typeof routeOrPacket !== 'object') return {};
  if (routeOrPacket.routing_packet && typeof routeOrPacket.routing_packet === 'object') return routeOrPacket.routing_packet;
  if (routeOrPacket.work_packet && typeof routeOrPacket.work_packet === 'object') return routeOrPacket.work_packet;
  if (routeOrPacket.proposal_packet && typeof routeOrPacket.proposal_packet === 'object') return routeOrPacket.proposal_packet;
  if (routeOrPacket.lease_packet && typeof routeOrPacket.lease_packet === 'object') return routeOrPacket.lease_packet;
  return routeOrPacket;
}

export function routingPacketPayload(routeOrPacket = {}, overrides = {}) {
  const packet = routingPacketFromRoute(routeOrPacket);
  const packetPayload = packet.payload && typeof packet.payload === 'object' ? packet.payload : {};
  const action = String(overrides.action || packet.action || routeOrPacket.recommended_next_action || '').trim().toLowerCase();
  if (action === 'route_to_agent' || action === 'route_hybrid_pair' || packet.output_contracts?.some((item) => item?.output_protocol === 'mergeos.agent-lease.v1')) {
    return agentLeasePayload({ ...routeOrPacket, lease_packet: packet }, overrides);
  }
  if (action === 'invite_contributor' || action === 'publish_bounty' || packet.output_contracts?.some((item) => item?.output_protocol === 'mergeos.proposal.v1')) {
    return proposalPayloadFromBounty({ ...routeOrPacket, proposal_packet: packet }, overrides);
  }
  return compactPayload({ ...packetPayload, ...overrides.payload });
}

export function routingPacketOutputContracts(routeOrPacket = {}, action = '') {
  const packet = routingPacketFromRoute(routeOrPacket);
  const contracts = Array.isArray(packet.output_contracts) ? packet.output_contracts : [];
  const normalized = String(action || '').trim().toLowerCase();
  if (!normalized) return contracts;
  return contracts.filter((item) => String(item?.action || '').trim().toLowerCase() === normalized);
}

export function proposalPayloadFromBounty(bounty = {}, overrides = {}) {
  const packet = bounty.proposal_packet && typeof bounty.proposal_packet === 'object'
    ? bounty.proposal_packet
    : bounty.claim_packet && typeof bounty.claim_packet === 'object'
      ? bounty.claim_packet
      : {};
  const packetPayload = packet.payload && typeof packet.payload === 'object' ? packet.payload : {};
  const bidCents = Number(overrides.bid_cents ?? overrides.bidCents ?? packetPayload.bid_cents ?? bounty.reward_cents) || 0;
  const estimatedHours = Number(overrides.estimated_hours ?? overrides.estimatedHours ?? packetPayload.estimated_hours ?? bounty.estimated_hours) || 0;
  return compactPayload({
    task_id: overrides.task_id || overrides.taskID || packetPayload.task_id || bounty.claim_id || bounty.id || '',
    cover_letter: overrides.cover_letter || overrides.coverLetter || packetPayload.cover_letter || '',
    bid_cents: bidCents || undefined,
    estimated_hours: estimatedHours || undefined,
    availability: overrides.availability || packetPayload.availability || 'Available after customer approval',
  });
}

export function proposalPacketOutputContracts(bounty = {}, action = '') {
  const packet = bounty.proposal_packet && typeof bounty.proposal_packet === 'object'
    ? bounty.proposal_packet
    : bounty.claim_packet && typeof bounty.claim_packet === 'object'
      ? bounty.claim_packet
      : {};
  const contracts = Array.isArray(packet.output_contracts) ? packet.output_contracts : [];
  const normalized = String(action || '').trim().toLowerCase();
  if (!normalized) return contracts;
  return contracts.filter((item) => String(item?.action || '').trim().toLowerCase() === normalized);
}

export function adminOpsActionOutputContracts(actionOrItem = {}, action = '') {
  const source = actionOrItem && typeof actionOrItem === 'object' ? actionOrItem : {};
  const actions = Array.isArray(source.actions) ? source.actions : [source];
  const normalized = String(action || (Array.isArray(source.actions) ? '' : source.type || '')).trim().toLowerCase();
  const contracts = actions.flatMap((row) => {
    if (!row || typeof row !== 'object') return [];
    const rows = Array.isArray(row.output_contracts) ? row.output_contracts : [];
    if (!normalized) return rows;
    return rows.filter((contract) => String(contract?.action || '').trim().toLowerCase() === normalized);
  });
  return contracts.filter((contract) => contract && typeof contract === 'object');
}

export function adminOpsQueueOutputContracts(queue = {}, action = '') {
  const contracts = Array.isArray(queue?.output_contracts) ? queue.output_contracts : [];
  const normalized = String(action || '').trim().toLowerCase();
  if (!normalized) return contracts.filter((contract) => contract && typeof contract === 'object');
  return contracts.filter((contract) => String(contract?.action || '').trim().toLowerCase() === normalized);
}

export function autoReleasePayloadFromPRMonitorTask(task = {}, overrides = {}) {
  const packet = task.auto_release_packet && typeof task.auto_release_packet === 'object' ? task.auto_release_packet : {};
  const packetPayload = packet.payload && typeof packet.payload === 'object' ? packet.payload : {};
  const overrideCandidates = Array.isArray(overrides.candidates) ? overrides.candidates : null;
  const packetCandidates = Array.isArray(packetPayload.candidates) ? packetPayload.candidates : [];
  const candidates = overrideCandidates || (packetCandidates.length ? packetCandidates : [autoReleaseCandidateFromPRMonitorTask(task, overrides)]);
  const taskIDs = Array.isArray(overrides.task_ids)
    ? overrides.task_ids
    : Array.isArray(overrides.taskIDs)
      ? overrides.taskIDs
      : Array.isArray(packetPayload.task_ids)
        ? packetPayload.task_ids
        : candidates.map((candidate) => candidate.task_id).filter(Boolean);
  return {
    task_ids: taskIDs,
    policy: overrides.policy || packetPayload.policy || packet.policy || 'mergeos.auto_release.low_risk_pr.v1',
    candidates,
  };
}

export function autoReleaseCandidateFromPRMonitorTask(task = {}, overrides = {}) {
  const pullRequests = Array.isArray(task.pull_requests) ? task.pull_requests : [];
  const selectedPull = overrides.pull_request || overrides.pullRequest || pullRequests.find((pull) => pull?.readiness?.status === 'ready') || pullRequests[0] || {};
  const readiness = selectedPull.readiness && typeof selectedPull.readiness === 'object' ? selectedPull.readiness : {};
  const validationSignals = overrides.validation_signals || overrides.validationSignals || readiness.signals || [];
  const deploymentStatus = overrides.deployment_status || overrides.deploymentStatus || task.deployment_status || (validationSignals.includes('deployment: verified') ? 'validated' : 'not_required');
  return compactPayload({
    task_id: overrides.task_id || overrides.taskID || task.task_id || '',
    worker_kind: overrides.worker_kind || overrides.workerKind || task.worker_kind || 'human',
    worker_id: overrides.worker_id || overrides.workerID || task.worker_id || '',
    agent_type: overrides.agent_type || overrides.agentType || task.agent_type || '',
    reward_cents: Number(overrides.reward_cents ?? overrides.rewardCents ?? task.reward_cents) || 0,
    repository: overrides.repository || task.repository || '',
    pull_request_number: Number(overrides.pull_request_number ?? overrides.pullRequestNumber ?? selectedPull.number) || 0,
    pull_request_url: overrides.pull_request_url || overrides.pullRequestURL || selectedPull.html_url || selectedPull.url || '',
    pull_request_title: overrides.pull_request_title || overrides.pullRequestTitle || selectedPull.title || task.title || '',
    readiness_status: overrides.readiness_status || overrides.readinessStatus || readiness.status || 'needs_review',
    can_merge: Boolean(overrides.can_merge ?? overrides.canMerge ?? readiness.can_merge),
    risk_level: overrides.risk_level || overrides.riskLevel || readiness.risk_level || 'medium',
    deployment_status: deploymentStatus,
    validation_signals: normalizeStringList(validationSignals),
    draft: Boolean(overrides.draft ?? selectedPull.draft),
    can_release: Boolean(overrides.can_release ?? overrides.canRelease ?? task.auto_release_packet?.can_auto_release ?? false),
  });
}

export function autoReleaseProofsFromResponse(response = {}) {
  return Array.isArray(response.release_proofs) ? response.release_proofs.filter((proof) => proof && typeof proof === 'object') : [];
}

export function autoReleaseLedgerProofLinksFromResponse(response = {}) {
  return autoReleaseProofsFromResponse(response)
    .map((proof) => {
      const url = String(proof.ledger_proof_url || '').trim();
      if (!url) return null;
      return {
        kind: 'auto_release',
        task_id: proof.task_id || '',
        claim_id: proof.claim_id || '',
        pull_request_url: proof.pull_request_url || '',
        ledger_reference: proof.ledger_reference || '',
        url,
      };
    })
    .filter(Boolean);
}

export function workerDashboardProofLinks(dashboard = {}, kind = '') {
  const normalized = String(kind || '').trim().toLowerCase();
  const links = [];
  for (const task of Array.isArray(dashboard.claimed_tasks) ? dashboard.claimed_tasks : []) {
    const url = String(task?.ledger_proof_url || '').trim();
    if (!url) continue;
    links.push({
      kind: 'claimed_task',
      task_id: task.id || '',
      project_id: task.project_id || '',
      issue_number: Number(task.issue_number) || 0,
      title: task.title || '',
      url,
    });
  }
  for (const reward of Array.isArray(dashboard.rewards) ? dashboard.rewards : []) {
    const url = String(reward?.ledger_proof_url || '').trim();
    if (!url) continue;
    links.push({
      kind: 'reward',
      sequence: Number(reward.sequence) || 0,
      type: reward.type || '',
      reference: reward.reference || reward.entry_hash || '',
      url,
    });
  }
  if (!normalized) return links;
  return links.filter((link) => link.kind === normalized);
}

function nonNegativeInteger(value) {
  const parsed = Number(value);
  if (!Number.isFinite(parsed) || parsed <= 0) return 0;
  return Math.floor(parsed);
}

function positiveInteger(value) {
  const parsed = Number(value);
  if (!Number.isFinite(parsed) || parsed <= 0) return 0;
  return Math.floor(parsed);
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
  return normalized === 'agent.action'
    || normalized === workflowEventTypes.agentLeased
    || normalized === workflowEventTypes.agentHeartbeat
    || normalized === workflowEventTypes.agentReleased
    || Object.values(agentActionEventTypes).includes(normalized);
}

export function liveFeedTypeToProtocolEventType(type = '', action = '') {
  const normalized = String(type || '').trim().toLowerCase();
  if (normalized === 'ledger_payment_verified') return workflowEventTypes.paymentVerified;
  if (normalized.startsWith('ledger_task_payment')) return workflowEventTypes.payoutReleased;
  if (normalized === 'ledger_airdrop_claim') return workflowEventTypes.airdropClaimed;
  if (normalized === 'ledger_presale_reservation') return workflowEventTypes.presaleReserved;
  if (normalized === 'ledger_wallet_migration') return workflowEventTypes.walletMigrated;
  if (normalized.startsWith('ledger_')) return workflowEventTypes.ledgerRecorded;
  if (normalized === 'notifications_updated') return workflowEventTypes.notificationsUpdated;
  if (normalized === 'agent_action') return agentActionEventType(action);
  if (normalized === 'agent_lease') return agentLeaseEventType(action);
  return {
    project_funded: workflowEventTypes.projectFunded,
    task_opened: workflowEventTypes.taskCreated,
    task_claimed: workflowEventTypes.taskClaimed,
    task_submitted: workflowEventTypes.taskSubmitted,
    task_changes_requested: workflowEventTypes.taskChangesRequested,
    task_accepted: workflowEventTypes.taskAccepted,
    proposal_submitted: workflowEventTypes.proposalSubmitted,
    proposal_accepted: workflowEventTypes.proposalAccepted,
    proposal_declined: workflowEventTypes.proposalDeclined,
    pr_opened: workflowEventTypes.prOpened,
    ai_review: workflowEventTypes.prReviewed,
    pr_ready_for_release: workflowEventTypes.prReadyForRelease,
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
  if (normalized.startsWith('airdrop.') || normalized.startsWith('presale.')) return 'token';
  if (normalized.startsWith('wallet.')) return 'wallet';
  if (normalized.startsWith('notification.')) return 'notification';
  if (normalized.startsWith('payment.')) return 'payment';
  if (normalized.startsWith('payout.')) return 'payout';
  if (normalized.startsWith('ledger.')) return 'ledger';
  return 'unknown';
}

export function agentLeaseEventType(action = '') {
  switch (String(action || '').trim().toLowerCase()) {
    case 'heartbeat':
      return workflowEventTypes.agentHeartbeat;
    case 'released':
      return workflowEventTypes.agentReleased;
    default:
      return workflowEventTypes.agentLeased;
  }
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

function finitePositiveLimit(limit) {
  const value = Number(limit);
  if (!Number.isFinite(value) || value <= 0) return '';
  return Math.floor(value);
}

function publicLimitQuery(limit) {
  const value = finitePositiveLimit(limit);
  return value === '' ? '' : `?limit=${encodeURIComponent(value)}`;
}

function liveFeedQueryString(options = {}) {
  return queryString({
    limit: finitePositiveLimit(options.limit),
    after_id: options.after_id || options.afterID || options.cursor || '',
    since: normalizeSinceQueryValue(options.since),
  });
}

function appendQueryString(path = '/api/ws', query = '') {
  const normalizedPath = String(path || '/api/ws');
  const normalizedQuery = String(query || '');
  if (!normalizedQuery) return normalizedPath;
  const suffix = normalizedQuery.startsWith('?') ? normalizedQuery.slice(1) : normalizedQuery;
  if (!suffix) return normalizedPath;
  return `${normalizedPath}${normalizedPath.includes('?') ? '&' : '?'}${suffix}`;
}

function protocolManifestSource(discoveryOrManifest = {}) {
  if (discoveryOrManifest && typeof discoveryOrManifest === 'object' && discoveryOrManifest.manifest) {
    return discoveryOrManifest.manifest;
  }
  return discoveryOrManifest || {};
}

function normalizeSinceQueryValue(value) {
  if (!value) return '';
  if (value instanceof Date) return value.toISOString();
  return String(value);
}

function passwordPayload(passwordOrPayload) {
  if (passwordOrPayload && typeof passwordOrPayload === 'object') {
    return passwordOrPayload;
  }
  return { password: String(passwordOrPayload || '') };
}

function firstPayloadValue(source = {}, keys = [], fallback = '') {
  for (const key of keys) {
    if (source[key] !== undefined && source[key] !== null && String(source[key]).trim() !== '') {
      return source[key];
    }
  }
  return fallback;
}

function normalizeSelector(value = '') {
  return String(value || '').trim().toLowerCase().replace(/[\s_]+/g, '-');
}

function titleFromProtocolKind(value = '') {
  const raw = String(value || '').replace(/^mergeos\./, '').replace(/\.v\d+$/, '');
  return raw
    .split(/[-_.\s]+/)
    .filter(Boolean)
    .map((part) => `${part.slice(0, 1).toUpperCase()}${part.slice(1)}`)
    .join(' ') || 'Protocol Document';
}

function endpointID(endpoint = {}) {
  return `${String(endpoint.method || 'GET').toLowerCase()}:${String(endpoint.path || '/')
    .replace(/^\/+|\/+$/g, '')
    .replace(/[{}]/g, '')
    .replace(/[/?=&]+/g, '-')}`;
}

function accessFromEndpointAuth(auth = '') {
  const value = String(auth || '').trim();
  if (!value || value === 'none') return 'public';
  if (value === 'project') return 'project';
  return 'authenticated';
}

function categoryFromEndpoint(endpoint = {}) {
  const haystack = `${endpoint.path || ''} ${endpoint.protocol || endpoint.protocol_version || ''} ${endpoint.description || ''}`.toLowerCase();
  if (haystack.includes('agent')) return 'agents';
  if (haystack.includes('marketplace') || haystack.includes('task')) return 'tasks';
  if (haystack.includes('workflow') || haystack.includes('routing')) return 'workflow';
  if (haystack.includes('repo')) return 'repository';
  if (haystack.includes('deployment') || haystack.includes('pull-request')) return 'deployment';
  if (haystack.includes('ledger') || haystack.includes('escrow') || haystack.includes('payout') || haystack.includes('token')) return 'ledger';
  if (haystack.includes('live-feed') || haystack.includes('events') || String(endpoint.method || '').toUpperCase() === 'WS') return 'realtime';
  if (haystack.includes('admin') || haystack.includes('dispute')) return 'admin';
  return 'discovery';
}

function fillPathTemplate(path = '', params = {}) {
  return String(path || '').replace(/\{([^}]+)\}/g, (match, key) => {
    const camelKey = toCamelCase(key);
    const acronymKey = camelKey.replace(/Id$/, 'ID');
    const value = params[key] ?? params[camelKey] ?? params[acronymKey] ?? params[normalizeSelector(key)];
    return value === undefined || value === null || String(value) === '' ? match : encodeURIComponent(String(value));
  });
}

function toCamelCase(value = '') {
  return String(value || '').replace(/[_-]([a-z0-9])/gi, (_, part) => part.toUpperCase());
}

function integerPayloadValue(value, fallback) {
  if (value === undefined || value === null || String(value).trim() === '') {
    return fallback;
  }
  const number = Number(value);
  if (!Number.isFinite(number)) {
    return value;
  }
  return Math.trunc(number);
}

function firstTokenWorkflowValue(source = {}, keys = [], fallback = '') {
  for (const key of keys) {
    if (source[key] !== undefined && source[key] !== null && String(source[key]).trim() !== '') {
      return source[key];
    }
  }
  return fallback;
}

function tokenWorkflowInteger(value, fallback) {
  if (value === undefined || value === null || String(value).trim() === '') {
    return fallback;
  }
  const number = Number(value);
  if (!Number.isFinite(number)) {
    return value;
  }
  return Math.trunc(number);
}

function tokenWorkflowList(value = []) {
  const items = Array.isArray(value) ? value : String(value || '').split(',');
  const normalized = items
    .map((item) => String(item || '').trim())
    .filter(Boolean);
  return normalized.length ? normalized : undefined;
}

function normalizeContextURLList(value = []) {
  const items = Array.isArray(value)
    ? value
    : value && typeof value === 'object'
      ? Object.values(value)
      : String(value || '').split(',');
  return items.map((item) => String(item || '').trim()).filter(Boolean);
}

export function evidenceRequiredFromEvent(source) {
  if (!source || typeof source !== 'object') return [];
  const raw = source.evidence_required || source.evidenceRequired || source.evidence || [];
  const items = Array.isArray(raw) ? raw : String(raw || '').split(',');
  const seen = new Set();
  return items.reduce((acc, item) => {
    const key = normalizeEvidenceKey(item);
    if (key && !seen.has(key)) {
      seen.add(key);
      acc.push(key);
    }
    return acc;
  }, []);
}

export function hasEvidenceRequirement(source, requirement = '') {
  if (!source || typeof source !== 'object') return false;
  const required = evidenceRequiredFromEvent(source);
  const normalized = normalizeEvidenceKey(requirement);
  return required.includes(normalized);
}

function normalizeEvidenceKey(value = '') {
  return String(value || '').trim().toLowerCase().replace(/-/g, '_').replace(/\s+/g, '_') || '';
}

function normalizeStringList(value = []) {
  const items = Array.isArray(value) ? value : String(value || '').split(',');
  return items.map((item) => String(item || '').trim()).filter(Boolean);
}

function normalizeRunbookList(value = []) {
  const items = Array.isArray(value) ? value : String(value || '').split('\n');
  return items.map((item) => {
    if (!item || typeof item !== 'object') return String(item || '').trim();
    const prefix = item.step ? `${item.step}. ` : '';
    const label = item.label || item.action || 'agent step';
    const method = item.method ? `${item.method} ` : '';
    const endpoint = item.endpoint ? ` (${method}${item.endpoint})` : '';
    return `${prefix}${label}${endpoint}`.trim();
  }).filter(Boolean);
}

function compactPayload(payload = {}) {
  const compacted = {};
  for (const [key, value] of Object.entries(payload)) {
    if (value !== undefined && value !== null && String(value).trim() !== '') {
      compacted[key] = value;
    }
  }
  return compacted;
}

function agentMatchesProtocolFilters(agent = {}, filters = {}) {
  const action = filters.action || filters.agentAction || '';
  if (action && !agentSupportsAction(agent, action)) return false;
  const capability = filters.capability || filters.capabilityKey || '';
  if (capability && !agentHasCapability(agent, capability)) return false;
  const status = filters.status || '';
  if (status && normalizeSelector(agent.status) !== normalizeSelector(status)) return false;
  const agentType = filters.agent_type || filters.agentType || filters.type || '';
  if (agentType && normalizeSelector(agent.type) !== normalizeSelector(agentType)) return false;
  const role = filters.role || '';
  if (role && normalizeSelector(agent.role) !== normalizeSelector(role)) return false;
  if (filters.openOnly || filters.hasOpenTasks) {
    const openCount = Number(agent.open_task_count ?? agent.openTaskCount) || 0;
    const openIDs = Array.isArray(agent.open_task_ids) ? agent.open_task_ids : [];
    if (openCount <= 0 && openIDs.length === 0) return false;
  }
  return true;
}

function compareAgentProtocolFit(a = {}, b = {}) {
  const statusScore = (agent) => (normalizeSelector(agent.status) === 'active' ? 2 : 0);
  const openScore = (agent) => {
    const count = Number(agent.open_task_count ?? agent.openTaskCount) || 0;
    const ids = Array.isArray(agent.open_task_ids) ? agent.open_task_ids.length : 0;
    return Math.max(count, ids);
  };
  const budgetScore = (agent) => Number(agent.budget_mrg ?? agent.budgetMRG) || 0;
  return (statusScore(b) - statusScore(a))
    || (openScore(b) - openScore(a))
    || (budgetScore(b) - budgetScore(a))
    || String(a.type || a.title || '').localeCompare(String(b.type || b.title || ''));
}
