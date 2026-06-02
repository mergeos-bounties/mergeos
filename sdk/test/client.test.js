import assert from 'node:assert/strict';
import test from 'node:test';
import { MergeOSClient, createMergeOSClient } from '../src/index.js';

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
    { status: 200, body: { valid: true } },
  ]);
  const client = new MergeOSClient({
    baseURL: 'https://mergeos.shop/',
    token: 'secret-token',
    fetchImpl,
  });

  const payload = await client.publicLiveFeed({ limit: 80 });
  const verification = await client.publicLedgerVerification();

  assert.deepEqual(payload, { items: [] });
  assert.deepEqual(verification, { valid: true });
  assert.equal(fetchImpl.calls[0].url, 'https://mergeos.shop/api/public/live-feed?limit=80');
  assert.equal(fetchImpl.calls[0].options.headers.Authorization, undefined);
  assert.equal(fetchImpl.calls[1].url, 'https://mergeos.shop/api/public/ledger/verify');
  assert.equal(fetchImpl.calls[1].options.headers.Authorization, undefined);
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

  assert.equal(fetchImpl.calls[0].url, '/api/public/repo/issues');
  assert.equal(fetchImpl.calls[1].url, '/api/public/test-settings/status');
  assert.equal(fetchImpl.calls[2].url, '/api/public/test-settings/auth');
  assert.equal(fetchImpl.calls[3].url, '/api/public/test-settings/entries/list');
  assert.equal(fetchImpl.calls[4].url, '/api/public/test-settings/entries');
  assert.equal(fetchImpl.calls[5].url, '/api/public/test-settings/entries/tse_2');
  assert.equal(fetchImpl.calls[5].options.method, 'PATCH');
  assert.equal(fetchImpl.calls[6].options.method, 'DELETE');
  assert.equal(fetchImpl.calls[6].options.headers.Authorization, undefined);
  assert.equal(fetchImpl.calls[4].options.body, JSON.stringify({
    password: 'pw',
    integration_type: 'llm',
    setting_key: 'TASK_LLM_TEST_KEY',
    setting_value: 'value',
  }));
});

test('sends bearer token and JSON body for task acceptance', async () => {
  const fetchImpl = fakeFetch([{ status: 200, body: { id: 'tsk_1', status: 'accepted' } }]);
  const client = createMergeOSClient({ baseURL: 'http://127.0.0.1:8080', token: 'abc', fetchImpl });

  const payload = { worker_kind: 'human', worker_id: 'github:worker' };
  const task = await client.acceptTask('tsk_1', payload);

  assert.equal(task.status, 'accepted');
  assert.equal(fetchImpl.calls[0].url, 'http://127.0.0.1:8080/api/tasks/tsk_1/accept');
  assert.equal(fetchImpl.calls[0].options.method, 'POST');
  assert.equal(fetchImpl.calls[0].options.headers.Authorization, 'Bearer abc');
  assert.equal(fetchImpl.calls[0].options.body, JSON.stringify(payload));
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
    { status: 200, body: { release_status: 'funded' } },
    { status: 200, body: { stats: { pull_request_count: 2 }, tasks: [] } },
    { status: 200, body: { status: 'validating' } },
    { status: 200, body: { status: 'orchestrating' } },
    { status: 200, body: { stats: { node_count: 2 }, nodes: [], edges: [] } },
    { status: 200, body: { status: 'ready', stats: { scanned_files: 3 }, findings: [] } },
    { status: 200, body: { stats: { total_count: 1 }, items: [] } },
    { status: 200, body: { stats: { worker_count: 1 }, workers: [] } },
  ]);
  const client = new MergeOSClient({ token: 'admin-token', fetchImpl });

  await client.projectEscrow('prj_1');
  const pulls = await client.projectPullRequests('prj_1');
  await client.projectDeployment('prj_1');
  await client.projectAIWorkflow('prj_1');
  const graph = await client.projectTaskGraph('prj_1');
  const scan = await client.projectRepositoryScan('prj_1');
  const ops = await client.adminOpsQueue();
  const reputation = await client.adminReputation();

  assert.equal(pulls.stats.pull_request_count, 2);
  assert.equal(graph.stats.node_count, 2);
  assert.equal(scan.stats.scanned_files, 3);
  assert.equal(ops.stats.total_count, 1);
  assert.equal(reputation.stats.worker_count, 1);
  assert.equal(fetchImpl.calls[0].url, '/api/projects/prj_1/escrow');
  assert.equal(fetchImpl.calls[1].url, '/api/projects/prj_1/pull-requests');
  assert.equal(fetchImpl.calls[2].url, '/api/projects/prj_1/deployment');
  assert.equal(fetchImpl.calls[3].url, '/api/projects/prj_1/ai-workflow');
  assert.equal(fetchImpl.calls[4].url, '/api/projects/prj_1/task-graph');
  assert.equal(fetchImpl.calls[5].url, '/api/projects/prj_1/repo-scan');
  assert.equal(fetchImpl.calls[6].url, '/api/admin/ops-queue');
  assert.equal(fetchImpl.calls[7].url, '/api/admin/reputation');
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
