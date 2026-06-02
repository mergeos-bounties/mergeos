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

test('creates public live feed requests without auth', async () => {
  const fetchImpl = fakeFetch([{ status: 200, body: { items: [] } }]);
  const client = new MergeOSClient({
    baseURL: 'https://mergeos.shop/',
    token: 'secret-token',
    fetchImpl,
  });

  const payload = await client.publicLiveFeed({ limit: 80 });

  assert.deepEqual(payload, { items: [] });
  assert.equal(fetchImpl.calls[0].url, 'https://mergeos.shop/api/public/live-feed?limit=80');
  assert.equal(fetchImpl.calls[0].options.headers.Authorization, undefined);
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

test('exposes project workflow and admin ops routes', async () => {
  const fetchImpl = fakeFetch([
    { status: 200, body: { status: 'validating' } },
    { status: 200, body: { status: 'orchestrating' } },
    { status: 200, body: { stats: { total_count: 1 }, items: [] } },
  ]);
  const client = new MergeOSClient({ token: 'admin-token', fetchImpl });

  await client.projectDeployment('prj_1');
  await client.projectAIWorkflow('prj_1');
  const ops = await client.adminOpsQueue();

  assert.equal(ops.stats.total_count, 1);
  assert.equal(fetchImpl.calls[0].url, '/api/projects/prj_1/deployment');
  assert.equal(fetchImpl.calls[1].url, '/api/projects/prj_1/ai-workflow');
  assert.equal(fetchImpl.calls[2].url, '/api/admin/ops-queue');
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
