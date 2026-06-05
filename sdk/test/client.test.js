import assert from 'node:assert/strict';
import test from 'node:test';
import {
  MergeOSClient,
  agentActionPayload,
  agentActionEventType,
  agentActionEventTypes,
  contractReferenceBytes,
  contractReferenceFromLedger,
  createMergeOSClient,
  deploymentAgentActionPayload,
  isAgentActionEventType,
  isWorkflowEventType,
  legacyWalletAddressHash,
  liveFeedTypeToProtocolEventType,
  normalizeAgentAction,
  normalizeLegacyChain,
  protocolEventFromMessage,
  protocolEventsFromMessage,
  protocolEventGroup,
  protocolTypeFromMessage,
  workflowEventTypes,
} from '../src/index.js';

function fakeFetch(responses = []) {
  const calls = [];
  const fetchImpl = async (url, options = {}) => {
    calls.push({ url, options });
    const next = responses.shift() || { status: 200, body: {} };
    return {
      ok: next.status >= 200 && next.status < 300,
      status: next.status,
      statusText: next.statusText || '',
      text: async () => JSON.stringify(next.body),
    };
  };
  fetchImpl.calls = calls;
  return fetchImpl;
}

test('creates public feed and ledger verification requests without auth', async () => {
  const fetchImpl = fakeFetch([
    { status: 200, body: { items: [] } },
    { status: 200, body: { protocol_version: 'mergeos.protocol.manifest.v1' } },
    { status: 200, body: { tasks: [] } },
    { status: 200, body: { agents: [] } },
    { status: 200, body: { protocol_version: 'mergeos.ledger.v1', entries: [] } },
    { status: 200, body: { events: [] } },
    { status: 200, body: { valid: true } },
  ]);
  const client = new MergeOSClient({
    baseURL: 'https://mergeos.shop/',
    token: 'secret-token',
    fetchImpl,
  });

  const payload = await client.publicLiveFeed({ limit: 80 });
  const manifest = await client.publicProtocolManifest();
  const tasks = await client.publicProtocolTasks({ limit: 80 });
  const agents = await client.publicProtocolAgents({ limit: 80 });
  const ledger = await client.publicProtocolLedger();
  const events = await client.publicProtocolEvents({ limit: 80 });
  const verification = await client.publicLedgerVerification();

  assert.deepEqual(payload, { items: [] });
  assert.equal(manifest.protocol_version, 'mergeos.protocol.manifest.v1');
  assert.deepEqual(tasks, { tasks: [] });
  assert.deepEqual(agents, { agents: [] });
  assert.equal(ledger.protocol_version, 'mergeos.ledger.v1');
  assert.deepEqual(events, { events: [] });
  assert.deepEqual(verification, { valid: true });
  assert.equal(fetchImpl.calls[0].url, 'https://mergeos.shop/api/public/live-feed?limit=80');
  assert.equal(fetchImpl.calls[0].options.headers.Authorization, undefined);
  assert.equal(fetchImpl.calls[1].url, 'https://mergeos.shop/api/public/protocol');
  assert.equal(fetchImpl.calls[1].options.headers.Authorization, undefined);
  assert.equal(fetchImpl.calls[2].url, 'https://mergeos.shop/api/public/protocol/tasks?limit=80');
  assert.equal(fetchImpl.calls[2].options.headers.Authorization, undefined);
  assert.equal(fetchImpl.calls[3].url, 'https://mergeos.shop/api/public/protocol/agents?limit=80');
  assert.equal(fetchImpl.calls[3].options.headers.Authorization, undefined);
  assert.equal(fetchImpl.calls[4].url, 'https://mergeos.shop/api/public/protocol/ledger');
  assert.equal(fetchImpl.calls[4].options.headers.Authorization, undefined);
  assert.equal(fetchImpl.calls[5].url, 'https://mergeos.shop/api/public/protocol/events?limit=80');
  assert.equal(fetchImpl.calls[5].options.headers.Authorization, undefined);
  assert.equal(fetchImpl.calls[6].url, 'https://mergeos.shop/api/public/ledger/verify');
  assert.equal(fetchImpl.calls[6].options.headers.Authorization, undefined);
});

test('derives Solana ledger references and legacy wallet hashes for operators', () => {
  const entryHash = 'B'.repeat(64);
  const ledgerEntry = {
    sequence: 11,
    type: 'manual_credit',
    reference: 'pr:https://github.com/mergeos-bounties/mergeos/pull/151',
    entry_hash: entryHash,
  };

  assert.equal(contractReferenceFromLedger(ledgerEntry), entryHash.toLowerCase());
  assert.equal(contractReferenceFromLedger(`0x${entryHash}`), entryHash.toLowerCase());
  assert.deepEqual(contractReferenceBytes(ledgerEntry), Array(32).fill(187));
  assert.equal(contractReferenceFromLedger(ledgerEntry, { format: '0x' }), `0x${entryHash.toLowerCase()}`);

  const fallback = contractReferenceFromLedger({ reference: ledgerEntry.reference });
  assert.match(fallback, /^[0-9a-f]{64}$/);
  assert.equal(fallback, contractReferenceFromLedger({ reference: ledgerEntry.reference }));
  assert.notEqual(fallback, entryHash.toLowerCase());

  assert.equal(normalizeLegacyChain('TRON'), 'trc20');
  assert.equal(normalizeLegacyChain('ethereum'), 'evm');
  assert.equal(legacyWalletAddressHash('tron', 'TXYZ987654321'), legacyWalletAddressHash('trc20', 'txyz987654321'));
  assert.equal(legacyWalletAddressHash('evm', '0xAbC0000000000000000000000000000000000000', { format: 'bytes' }).length, 32);
});

test('maps typed agent action event protocol values', () => {
  assert.equal(agentActionEventTypes.test, 'agent.tested');
  assert.equal(agentActionEventType('review'), 'agent.reviewed');
  assert.equal(agentActionEventType('TEST'), 'agent.tested');
  assert.equal(agentActionEventType('generate'), 'agent.generated');
  assert.equal(agentActionEventType('deploy'), 'agent.deployed');
  assert.equal(agentActionEventType('scan'), 'agent.scanned');
  assert.equal(agentActionEventType('unknown'), 'agent.action');
  assert.equal(isAgentActionEventType('agent.generated'), true);
  assert.equal(isAgentActionEventType('agent.action'), true);
  assert.equal(isAgentActionEventType('task.paid'), false);
});

test('builds deployment agent action payloads for deployment evidence', async () => {
  const payload = deploymentAgentActionPayload({
    url: 'https://vercel.example/deployments/mergeos-preview',
    status: 'processed',
    durationMillis: 42000,
    pullNumber: 120,
    labels: ['preview', 'release-gate'],
  });
  assert.deepEqual(payload, {
    action: 'deploy',
    agent_type: 'deployment-agent',
    status: 'processed',
    reference_url: 'https://vercel.example/deployments/mergeos-preview',
    duration_millis: 42000,
    pull_number: 120,
    labels: ['preview', 'release-gate'],
  });

  const fetchImpl = fakeFetch([
    { status: 201, body: { log: { event_name: 'agent_action', action: 'deploy' } } },
  ]);
  const client = new MergeOSClient({ token: 'agent-token', fetchImpl });
  const response = await client.recordDeployment('prj_1', { deploymentURL: payload.reference_url });

  assert.equal(response.log.action, 'deploy');
  assert.equal(fetchImpl.calls[0].url, '/api/projects/prj_1/agent-actions');
  assert.equal(fetchImpl.calls[0].options.method, 'POST');
  assert.equal(fetchImpl.calls[0].options.headers.Authorization, 'Bearer agent-token');
  assert.equal(fetchImpl.calls[0].options.body, JSON.stringify({
    action: 'deploy',
    agent_type: 'deployment-agent',
    status: 'processed',
    reference_url: 'https://vercel.example/deployments/mergeos-preview',
    duration_millis: 0,
    pull_number: 0,
    labels: [],
  }));
});

test('builds and sends typed AI agent action helpers', async () => {
  assert.equal(normalizeAgentAction('gen'), 'generate');
  assert.equal(normalizeAgentAction('bad'), 'review');
  assert.deepEqual(agentActionPayload('test', {
    referenceURL: 'https://github.com/acme/repo/pull/12',
    status: 'running',
    pullNumber: 12,
    labels: ['smoke'],
  }), {
    action: 'test',
    agent_type: 'qa-agent',
    status: 'running',
    reference_url: 'https://github.com/acme/repo/pull/12',
    duration_millis: 0,
    pull_number: 12,
    labels: ['smoke'],
  });

  const fetchImpl = fakeFetch([
    { status: 201, body: { log: { action: 'review' } } },
    { status: 201, body: { log: { action: 'test' } } },
    { status: 201, body: { log: { action: 'generate' } } },
    { status: 201, body: { log: { action: 'scan' } } },
  ]);
  const client = new MergeOSClient({ token: 'agent-token', fetchImpl });
  await client.recordAgentReview('prj_1', { pullNumber: 10 });
  await client.recordAgentTest('prj_1', { status: 'running' });
  await client.recordAgentGeneration('prj_1', { agentType: 'code-agent' });
  await client.recordAgentScan('prj_1', { url: 'https://scan.example/report' });

  assert.equal(fetchImpl.calls.length, 4);
  assert.equal(fetchImpl.calls[0].options.body, JSON.stringify({
    action: 'review',
    agent_type: 'review-agent',
    status: 'processed',
    reference_url: '',
    duration_millis: 0,
    pull_number: 10,
    labels: [],
  }));
  assert.equal(fetchImpl.calls[1].options.body, JSON.stringify({
    action: 'test',
    agent_type: 'qa-agent',
    status: 'running',
    reference_url: '',
    duration_millis: 0,
    pull_number: 0,
    labels: [],
  }));
  assert.equal(fetchImpl.calls[2].options.body, JSON.stringify({
    action: 'generate',
    agent_type: 'code-agent',
    status: 'processed',
    reference_url: '',
    duration_millis: 0,
    pull_number: 0,
    labels: [],
  }));
  assert.equal(fetchImpl.calls[3].options.body, JSON.stringify({
    action: 'scan',
    agent_type: 'scan-agent',
    status: 'processed',
    reference_url: 'https://scan.example/report',
    duration_millis: 0,
    pull_number: 0,
    labels: [],
  }));
});

test('loads runtime config without auth for payment rail discovery', async () => {
  const fetchImpl = fakeFetch([
    { status: 200, body: { payment_rails: [{ id: 'paypal', enabled: true }, { id: 'stripe', enabled: false }] } },
  ]);
  const client = new MergeOSClient({ token: 'secret-token', fetchImpl });

  const config = await client.runtimeConfig();

  assert.equal(fetchImpl.calls[0].url, '/api/config');
  assert.equal(fetchImpl.calls[0].options.headers.Authorization, undefined);
  assert.equal(config.payment_rails.length, 2);
  assert.equal(config.payment_rails[1].id, 'stripe');
});

test('maps live feed records to workflow event protocol values', () => {
  const protocolEvent = {
    protocol_version: 'mergeos.event.v1',
    kind: 'event',
    type: 'task.paid',
    payload: { ledger_sequence: 7 },
  };

  assert.equal(workflowEventTypes.prOpened, 'pr.opened');
  assert.equal(liveFeedTypeToProtocolEventType('project_funded'), 'project.funded');
  assert.equal(liveFeedTypeToProtocolEventType('task_opened'), 'task.created');
  assert.equal(liveFeedTypeToProtocolEventType('task_accepted'), 'task.claimed');
  assert.equal(liveFeedTypeToProtocolEventType('pr_opened'), 'pr.opened');
  assert.equal(liveFeedTypeToProtocolEventType('ai_review'), 'pr.reviewed');
  assert.equal(liveFeedTypeToProtocolEventType('deployment_validation'), 'deployment.updated');
  assert.equal(liveFeedTypeToProtocolEventType('repo_issues_synced'), 'repo.issues.synced');
  assert.equal(liveFeedTypeToProtocolEventType('ledger_task_payment'), 'task.paid');
  assert.equal(liveFeedTypeToProtocolEventType('ledger_manual_credit'), 'ledger.recorded');
  assert.equal(liveFeedTypeToProtocolEventType('agent_action', 'test'), 'agent.tested');
  assert.equal(liveFeedTypeToProtocolEventType('unknown'), 'agent.action');
  assert.equal(protocolEventFromMessage({ event: protocolEvent }), protocolEvent);
  assert.equal(protocolEventFromMessage({ type: 'ledger_manual_credit' }), null);
  assert.deepEqual(protocolEventsFromMessage({ event: protocolEvent }), [protocolEvent]);
  assert.deepEqual(protocolEventsFromMessage({ events: { events: [protocolEvent, null, 'bad'] } }), [protocolEvent]);
  assert.deepEqual(protocolEventsFromMessage({ type: 'connection_ready' }), []);
  assert.equal(protocolTypeFromMessage({ event: protocolEvent, protocol_type: 'ledger.recorded' }), 'task.paid');
  assert.equal(protocolTypeFromMessage({ protocol_type: 'ledger.recorded', type: 'ledger_task_payment' }), 'ledger.recorded');
  assert.equal(protocolTypeFromMessage({ type: 'ledger_manual_credit' }), 'ledger.recorded');
  assert.equal(protocolTypeFromMessage({ type: 'connection_ready' }), '');
  assert.equal(protocolTypeFromMessage({ type: 'live_feed_snapshot', events: { events: [protocolEvent] } }), '');
  assert.equal(protocolTypeFromMessage({ type: 'admin_ops_updated' }), '');
  assert.equal(protocolEventGroup('pr.opened'), 'pull_request');
  assert.equal(protocolEventGroup('task.paid'), 'task');
  assert.equal(protocolEventGroup('agent.tested'), 'agent');
  assert.equal(protocolEventGroup('repo.issues.synced'), 'repository');
  assert.equal(isWorkflowEventType('deployment.updated'), true);
  assert.equal(isWorkflowEventType('agent.scanned'), true);
  assert.equal(isWorkflowEventType('unknown.event'), false);
});

test('exposes public repo import and password-gated test settings without auth', async () => {
  const fetchImpl = fakeFetch([
    { status: 200, body: { issue_count: 2 } },
    { status: 200, body: { test_mode_enabled: true } },
    { status: 200, body: { authenticated: true } },
    { status: 200, body: [{ id: 'tse_1' }] },
    { status: 201, body: { id: 'tse_2' } },
    { status: 200, body: { id: 'tse_2', status: 'disabled' } },
    { status: 200, body: { ok: true } },
    { status: 200, body: { id: 'tse_2', setting_value: 'secret-value' } },
  ]);
  const client = new MergeOSClient({ token: 'secret-token', fetchImpl });

  await client.importRepoIssues({ repo_url: 'https://github.com/acme/repo' });
  await client.publicTestSettingsStatus();
  await client.publicTestSettingsAuth('pw');
  await client.publicTestSettingsEntries('pw');
  await client.publicAddTestSettingsEntry('pw', {
    integration_type: 'llm',
    setting_key: 'TASK_LLM_TEST_KEY',
    setting_value: 'value',
  });
  await client.publicUpdateTestSettingsEntry('tse_2', 'pw', { status: 'disabled' });
  await client.publicDeleteTestSettingsEntry('tse_2', 'pw');
  const revealed = await client.publicRevealTestSettingsEntry('tse_2', 'pw');

  assert.equal(fetchImpl.calls[0].url, '/api/public/repo/issues');
  assert.equal(fetchImpl.calls[1].url, '/api/public/test-settings/status');
  assert.equal(fetchImpl.calls[2].url, '/api/public/test-settings/auth');
  assert.equal(fetchImpl.calls[3].url, '/api/public/test-settings/entries/list');
  assert.equal(fetchImpl.calls[4].url, '/api/public/test-settings/entries');
  assert.equal(fetchImpl.calls[5].url, '/api/public/test-settings/entries/tse_2');
  assert.equal(fetchImpl.calls[5].options.method, 'PATCH');
  assert.equal(fetchImpl.calls[6].options.method, 'DELETE');
  assert.equal(fetchImpl.calls[6].options.headers.Authorization, undefined);
  assert.equal(fetchImpl.calls[7].url, '/api/public/test-settings/entries/tse_2/reveal');
  assert.equal(fetchImpl.calls[7].options.method, 'POST');
  assert.equal(fetchImpl.calls[7].options.headers.Authorization, undefined);
  assert.equal(revealed.setting_value, 'secret-value');
  assert.equal(fetchImpl.calls[4].options.body, JSON.stringify({
    password: 'pw',
    integration_type: 'llm',
    setting_key: 'TASK_LLM_TEST_KEY',
    setting_value: 'value',
  }));
});

test('sends bearer token and JSON body for task acceptance and disputes', async () => {
  const fetchImpl = fakeFetch([
    { status: 200, body: { id: 'tsk_1', status: 'accepted' } },
    { status: 201, body: { notification: { id: 'ntf_1', status: 'dispute:high' } } },
  ]);
  const client = createMergeOSClient({ baseURL: 'http://127.0.0.1:8080', token: 'abc', fetchImpl });

  const payload = { worker_kind: 'human', worker_id: 'github:worker' };
  const task = await client.acceptTask('tsk_1', payload);
  const disputePayload = { task_id: 'tsk_1', body: 'Evidence needs maintainer review.' };
  const dispute = await client.createDispute(disputePayload);

  assert.equal(task.status, 'accepted');
  assert.equal(dispute.notification.status, 'dispute:high');
  assert.equal(fetchImpl.calls[0].url, 'http://127.0.0.1:8080/api/tasks/tsk_1/accept');
  assert.equal(fetchImpl.calls[0].options.method, 'POST');
  assert.equal(fetchImpl.calls[0].options.headers.Authorization, 'Bearer abc');
  assert.equal(fetchImpl.calls[0].options.body, JSON.stringify(payload));
  assert.equal(fetchImpl.calls[1].url, 'http://127.0.0.1:8080/api/disputes');
  assert.equal(fetchImpl.calls[1].options.method, 'POST');
  assert.equal(fetchImpl.calls[1].options.headers.Authorization, 'Bearer abc');
  assert.equal(fetchImpl.calls[1].options.body, JSON.stringify(disputePayload));
});

test('supports wallet, payment, and raw upload helper routes', async () => {
  const uploadBody = { marker: 'form-data' };
  const fetchImpl = fakeFetch([
    { status: 201, body: { address: 'mrg_1' } },
    { status: 200, body: { address: 'mrg_1' } },
    { status: 200, body: { linked: true } },
    { status: 201, body: { order_id: 'ord_1' } },
    { status: 201, body: { id: 'att_1' } },
  ]);
  const client = createMergeOSClient({ token: 'abc', fetchImpl });

  await client.createWallet({ label: 'primary' });
  await client.wallet('mrg_1');
  await client.linkWallet({ address: 'mrg_1' });
  await client.createPayPalOrder({ project_id: 'prj_1' });
  await client.uploadAttachment(uploadBody, { headers: { 'X-Upload': '1' } });

  assert.equal(fetchImpl.calls[0].url, '/api/wallets');
  assert.equal(fetchImpl.calls[1].url, '/api/wallets/mrg_1');
  assert.equal(fetchImpl.calls[2].url, '/api/wallets/link');
  assert.equal(fetchImpl.calls[3].url, '/api/payments/paypal/orders');
  assert.equal(fetchImpl.calls[4].url, '/api/uploads');
  assert.equal(fetchImpl.calls[4].options.body, uploadBody);
  assert.equal(fetchImpl.calls[4].options.headers['Content-Type'], undefined);
  assert.equal(fetchImpl.calls[4].options.headers['X-Upload'], '1');
});

test('exposes project workflow and admin ops routes', async () => {
  const fetchImpl = fakeFetch([
    { status: 200, body: { protocol_version: 'mergeos.escrow.v1', release_status: 'funded' } },
    { status: 200, body: { project: { project_id: 'prj_1' }, task_graph: { stats: { node_count: 2 } } } },
    { status: 200, body: { protocol_version: 'mergeos.pr-monitor.v1', stats: { pull_request_count: 2 }, tasks: [] } },
    { status: 200, body: { protocol_version: 'mergeos.deployment.v1', status: 'validating' } },
    { status: 200, body: { protocol_version: 'mergeos.ai-workflow.v1', status: 'orchestrating' } },
    { status: 201, body: { log: { event_name: 'agent_action', action: 'test' } } },
    { status: 200, body: { stats: { node_count: 2 }, nodes: [], edges: [] } },
    { status: 200, body: { protocol_version: 'mergeos.workflow.v1', progress: 25, current_step: 'contributor_routing', nodes: [], edges: [] } },
    { status: 200, body: { status: 'ready', stats: { scanned_files: 3 }, findings: [] } },
    { status: 200, body: { protocol_version: 'mergeos.scan.v1', findings: [] } },
    { status: 200, body: { added_task_count: 1, updated_task_count: 2 } },
    { status: 200, body: { stats: { total_count: 1 }, items: [] } },
    { status: 200, body: { stats: { worker_count: 1 }, workers: [] } },
  ]);
  const client = new MergeOSClient({ token: 'admin-token', fetchImpl });

  const escrow = await client.projectEscrow('prj_1');
  const dashboard = await client.projectDashboard('prj_1');
  const pulls = await client.projectPullRequests('prj_1');
  const deployment = await client.projectDeployment('prj_1');
  const aiWorkflow = await client.projectAIWorkflow('prj_1');
  const agentAction = await client.createProjectAgentAction('prj_1', { action: 'test', agent_type: 'qa-agent' });
  const graph = await client.projectTaskGraph('prj_1');
  const workflowProtocol = await client.projectWorkflowProtocol('prj_1');
  const scan = await client.projectRepositoryScan('prj_1');
  const scanProtocol = await client.projectRepositoryScanProtocol('prj_1');
  const sync = await client.syncProjectRepoIssues('prj_1');
  const ops = await client.adminOpsQueue();
  const reputation = await client.adminReputation();

  assert.equal(escrow.protocol_version, 'mergeos.escrow.v1');
  assert.equal(dashboard.project.project_id, 'prj_1');
  assert.equal(pulls.protocol_version, 'mergeos.pr-monitor.v1');
  assert.equal(pulls.stats.pull_request_count, 2);
  assert.equal(deployment.protocol_version, 'mergeos.deployment.v1');
  assert.equal(aiWorkflow.protocol_version, 'mergeos.ai-workflow.v1');
  assert.equal(agentAction.log.action, 'test');
  assert.equal(graph.stats.node_count, 2);
  assert.equal(workflowProtocol.protocol_version, 'mergeos.workflow.v1');
  assert.equal(workflowProtocol.progress, 25);
  assert.equal(workflowProtocol.current_step, 'contributor_routing');
  assert.equal(scan.stats.scanned_files, 3);
  assert.equal(scanProtocol.protocol_version, 'mergeos.scan.v1');
  assert.equal(sync.added_task_count, 1);
  assert.equal(ops.stats.total_count, 1);
  assert.equal(reputation.stats.worker_count, 1);
  assert.equal(fetchImpl.calls[0].url, '/api/projects/prj_1/escrow');
  assert.equal(fetchImpl.calls[1].url, '/api/projects/prj_1/dashboard');
  assert.equal(fetchImpl.calls[2].url, '/api/projects/prj_1/pull-requests');
  assert.equal(fetchImpl.calls[3].url, '/api/projects/prj_1/deployment');
  assert.equal(fetchImpl.calls[4].url, '/api/projects/prj_1/ai-workflow');
  assert.equal(fetchImpl.calls[5].url, '/api/projects/prj_1/agent-actions');
  assert.equal(fetchImpl.calls[5].options.method, 'POST');
  assert.equal(fetchImpl.calls[6].url, '/api/projects/prj_1/task-graph');
  assert.equal(fetchImpl.calls[7].url, '/api/projects/prj_1/protocol/workflow');
  assert.equal(fetchImpl.calls[8].url, '/api/projects/prj_1/repo-scan');
  assert.equal(fetchImpl.calls[9].url, '/api/projects/prj_1/protocol/scan');
  assert.equal(fetchImpl.calls[10].url, '/api/projects/prj_1/repo-sync');
  assert.equal(fetchImpl.calls[10].options.method, 'POST');
  assert.equal(fetchImpl.calls[11].url, '/api/admin/ops-queue');
  assert.equal(fetchImpl.calls[12].url, '/api/admin/reputation');
});

test('exposes admin operations, review, settings, and integration routes', async () => {
  const fetchImpl = fakeFetch([
    { status: 200, body: [{ id: 'usr_1' }] },
    { status: 200, body: { id: 'usr_1', role: 'admin' } },
    { status: 200, body: [{ id: 'prj_1' }] },
    { status: 200, body: [{ id: 'tsk_1' }] },
    { status: 200, body: [{ number: 120 }] },
    { status: 200, body: { pull_request: { number: 120 } } },
    { status: 200, body: [{ id: 'note_1' }] },
    { status: 200, body: [{ id: 'att_1' }] },
    { status: 200, body: { llm_provider: 'gemini' } },
    { status: 200, body: { llm_provider: 'openrouter' } },
    { status: 200, body: [] },
    { status: 200, body: { status: 'ok' } },
    { status: 200, body: [] },
    { status: 201, body: { id: 'key_1' } },
    { status: 200, body: { id: 'key_1', status: 'disabled' } },
    { status: 200, body: { ok: true } },
    { status: 200, body: [] },
    { status: 200, body: { test_mode_enabled: true } },
    { status: 200, body: { test_mode_enabled: false } },
    { status: 200, body: [{ id: 'tse_1' }] },
    { status: 201, body: { id: 'tse_2' } },
    { status: 200, body: { id: 'tse_2', status: 'disabled' } },
    { status: 200, body: { ok: true } },
  ]);
  const client = new MergeOSClient({ token: 'admin-token', fetchImpl });

  await client.adminUsers();
  await client.updateAdminUser('usr_1', { name: 'Admin', email: 'admin@example.com' });
  await client.adminProjects();
  await client.adminTasks();
  await client.adminTaskPullRequests('tsk_1');
  await client.mergeAdminTaskPullRequest('tsk_1', 120, { worker_id: 'github:contributor', reward_mrg: 50, bounty_type: 'future-small' });
  await client.adminNotifications();
  await client.adminAttachments();
  await client.adminSettings();
  await client.updateAdminSettings({ llm_provider: 'openrouter' });
  await client.adminSSLReviews();
  await client.reviewAdminSSL({ domain: 'mergeos.shop' });
  await client.adminGeminiKeys();
  await client.addAdminGeminiKey({ key_value: 'secret' });
  await client.updateAdminGeminiKey('key_1', { status: 'disabled' });
  await client.testAdminGeminiKey('key_1');
  await client.adminGeminiWebhooks();
  await client.adminTestSettings();
  await client.updateAdminTestSettings({ test_mode_enabled: false });
  await client.adminTestSettingsEntries();
  await client.addAdminTestSettingsEntry({ integration_type: 'llm', setting_key: 'TASK_KEY', setting_value: 'value' });
  await client.updateAdminTestSettingsEntry('tse_2', { status: 'disabled' });
  await client.deleteAdminTestSettingsEntry('tse_2');

  assert.equal(fetchImpl.calls[0].url, '/api/admin/users');
  assert.equal(fetchImpl.calls[1].url, '/api/admin/users/usr_1');
  assert.equal(fetchImpl.calls[4].url, '/api/admin/tasks/tsk_1/pulls');
  assert.equal(fetchImpl.calls[5].url, '/api/admin/tasks/tsk_1/pulls/120/merge');
  assert.equal(fetchImpl.calls[5].options.method, 'POST');
  assert.equal(fetchImpl.calls[10].url, '/api/admin/ssl');
  assert.equal(fetchImpl.calls[12].url, '/api/admin/gemini/keys');
  assert.equal(fetchImpl.calls[15].url, '/api/admin/gemini/keys/key_1/test');
  assert.equal(fetchImpl.calls[17].url, '/api/admin/test-settings');
  assert.equal(fetchImpl.calls[18].options.method, 'PATCH');
  assert.equal(fetchImpl.calls[22].url, '/api/admin/test-settings/entries/tse_2');
  assert.equal(fetchImpl.calls[22].options.method, 'DELETE');
});

test('throws response errors with status and payload', async () => {
  const fetchImpl = fakeFetch([{ status: 403, body: { error: 'admin access is required' } }]);
  const client = new MergeOSClient({ fetchImpl });

  await assert.rejects(
    () => client.adminSummary(),
    (error) => {
      assert.equal(error.status, 403);
      assert.deepEqual(error.payload, { error: 'admin access is required' });
      return true;
    },
  );
});

test('builds websocket URLs for event streams', () => {
  const client = new MergeOSClient({ baseURL: 'https://mergeos.shop' });

  assert.equal(client.webSocketURL('/api/ws'), 'wss://mergeos.shop/api/ws');
  assert.equal(client.webSocketURL('ws://localhost:8080/api/ws'), 'ws://localhost:8080/api/ws');
});
