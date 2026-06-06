import assert from 'node:assert/strict';
import fs from 'node:fs/promises';
import http from 'node:http';
import os from 'node:os';
import path from 'node:path';
import test from 'node:test';
import {
  createAdminServer,
  createRuntimeConfig,
  normalizeMode,
  resolveMode,
  shouldRunProduction,
} from './server.js';

async function listen(server) {
  await new Promise((resolve) => server.listen(0, '127.0.0.1', resolve));
  return server.address().port;
}

async function request(url, options = {}) {
  const response = await fetch(url, options);
  return {
    status: response.status,
    headers: response.headers,
    text: await response.text(),
  };
}

async function withTempAdminDist(t) {
  const cwd = await fs.mkdtemp(path.join(os.tmpdir(), 'mergeos-admin-test-'));
  const clientDist = path.join(cwd, 'dist', 'client');
  const serverDist = path.join(cwd, 'dist', 'server');
  await fs.mkdir(clientDist, { recursive: true });
  await fs.mkdir(serverDist, { recursive: true });
  await fs.writeFile(path.join(clientDist, 'index.html'), '<main id="app"><!--ssr-outlet--></main>');
  await fs.writeFile(path.join(clientDist, 'favicon.svg'), '<svg xmlns="http://www.w3.org/2000/svg"></svg>');
  await fs.writeFile(path.join(clientDist, 'ops.json'), JSON.stringify({ ready: true }));
  await fs.writeFile(
    path.join(serverDist, 'entry-server.js'),
    'export async function render(route) { return `<section data-route="${route}">Admin SSR</section>`; }\n',
  );
  t.after(async () => {
    await fs.rm(cwd, { recursive: true, force: true });
  });
  return { cwd, clientDist, serverEntry: path.join(serverDist, 'entry-server.js') };
}

test('normalizes and resolves admin runtime modes', () => {
  assert.equal(normalizeMode('prod'), 'production');
  assert.equal(normalizeMode('production'), 'production');
  assert.equal(normalizeMode('dev'), 'local');
  assert.equal(normalizeMode(''), 'local');

  assert.equal(resolveMode(['node', 'server.js', '--mode', 'production'], { MERGEOS_ENV: 'local' }), 'production');
  assert.equal(shouldRunProduction(['node', 'server.js', '--prod'], { NODE_ENV: 'development' }), true);
});

test('creates production admin runtime config', async (t) => {
  const { cwd, clientDist, serverEntry } = await withTempAdminDist(t);
  const config = createRuntimeConfig({
    argv: ['node', 'server.js', '--mode', 'production'],
    env: {
      ADMIN_FRONTEND_HOST: '0.0.0.0',
      ADMIN_FRONTEND_PORT: '9090',
      API_TARGET: 'http://127.0.0.1:8080',
      CLIENT_DIST: clientDist,
      SERVER_ENTRY: serverEntry,
    },
    cwd,
  });

  assert.equal(config.mode, 'production');
  assert.equal(config.production, true);
  assert.equal(config.host, '0.0.0.0');
  assert.equal(config.port, 9090);
  assert.equal(config.clientDist, clientDist);
  assert.equal(config.serverEntry, serverEntry);
});

test('production admin server renders SSR and static assets', async (t) => {
  const { cwd, clientDist, serverEntry } = await withTempAdminDist(t);
  const server = await createAdminServer(createRuntimeConfig({
    argv: ['node', 'server.js', '--prod'],
    env: { CLIENT_DIST: clientDist, SERVER_ENTRY: serverEntry },
    cwd,
  }));
  const port = await listen(server);
  t.after(() => server.close());

  const page = await request(`http://127.0.0.1:${port}/moderation`);
  assert.equal(page.status, 200);
  assert.match(page.text, /Admin SSR/);
  assert.match(page.text, /data-route="\/moderation"/);

  const json = await request(`http://127.0.0.1:${port}/ops.json`);
  assert.equal(json.status, 200);
  assert.equal(json.headers.get('content-type'), 'application/json; charset=utf-8');
  assert.deepEqual(JSON.parse(json.text), { ready: true });

});

test('admin API proxy forwards requests to the backend target', async (t) => {
  const apiServer = http.createServer((req, res) => {
    res.setHeader('Content-Type', 'application/json; charset=utf-8');
    res.end(JSON.stringify({
      method: req.method,
      path: req.url,
      host: req.headers.host,
    }));
  });
  const apiPort = await listen(apiServer);
  t.after(() => apiServer.close());

  const { cwd, clientDist, serverEntry } = await withTempAdminDist(t);
  const adminServer = await createAdminServer(createRuntimeConfig({
    argv: ['node', 'server.js', '--prod'],
    env: {
      API_TARGET: `http://127.0.0.1:${apiPort}`,
      CLIENT_DIST: clientDist,
      SERVER_ENTRY: serverEntry,
    },
    cwd,
  }));
  const adminPort = await listen(adminServer);
  t.after(() => adminServer.close());

  const proxied = await request(`http://127.0.0.1:${adminPort}/api/admin/ops-queue?limit=5`, { method: 'POST' });
  assert.equal(proxied.status, 200);
  assert.deepEqual(JSON.parse(proxied.text), {
    method: 'POST',
    path: '/api/admin/ops-queue?limit=5',
    host: `127.0.0.1:${apiPort}`,
  });
});

test('admin app exposes required operations console surfaces', async () => {
  const source = await fs.readFile(new URL('./src/App.vue', import.meta.url), 'utf-8');

  for (const required of [
    'Treasury',
    'Users',
    'Disputes',
    'Payouts',
    'Moderation',
    'ops-queue',
    'ledger',
    'ssl',
    'review',
    'manual-credit-form',
    'handleOpsQueueAction',
  ]) {
    assert.match(source, new RegExp(required, 'i'));
  }
});
