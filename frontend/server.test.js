import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs/promises';
import http from 'node:http';
import net from 'node:net';
import os from 'node:os';
import path from 'node:path';
import {
  createMergeOSServer,
  createRuntimeConfig,
  loadEnvFiles,
  normalizeMode,
  parseEnvText,
  resolveMode,
  shouldRunProduction,
} from './server.js';

test('normalizes run modes', () => {
  assert.equal(normalizeMode('prod'), 'production');
  assert.equal(normalizeMode('production'), 'production');
  assert.equal(normalizeMode('dev'), 'local');
  assert.equal(normalizeMode('local'), 'local');
  assert.equal(normalizeMode(''), 'local');
});

test('resolves mode from CLI before environment', () => {
  const argv = ['node', 'server.js', '--mode', 'production'];
  const env = { MERGEOS_ENV: 'local' };
  assert.equal(resolveMode(argv, env), 'production');
  assert.equal(shouldRunProduction(argv, env), true);
});

test('parses env file lines', () => {
  assert.deepEqual(parseEnvText(`
    # comment
    API_TARGET="http://127.0.0.1:8080"
    FRONTEND_PORT=5173
  `), {
    API_TARGET: 'http://127.0.0.1:8080',
    FRONTEND_PORT: '5173',
  });
});

test('loads mode env before fallback without overriding real env', async () => {
  const cwd = await fs.mkdtemp(path.join(os.tmpdir(), 'mergeos-frontend-env-'));
  await fs.writeFile(path.join(cwd, '.env.local'), 'FRONTEND_PORT=5173\nAPI_TARGET=http://local-api\n');
  await fs.writeFile(path.join(cwd, '.env'), 'FRONTEND_PORT=9999\nSSR_PORT=6000\n');
  const env = { FRONTEND_PORT: '7000' };

  await loadEnvFiles('local', { cwd, env });

  assert.equal(env.FRONTEND_PORT, '7000');
  assert.equal(env.API_TARGET, 'http://local-api');
  assert.equal(env.SSR_PORT, '6000');
});

test('public protocol schemas mirror the protocol package schemas', async () => {
  const sourceDir = new URL('../protocol/schemas/', import.meta.url);
  const publicDir = new URL('./public/protocol/', import.meta.url);
  const sourceFiles = (await fs.readdir(sourceDir)).filter((file) => file.endsWith('.schema.json')).sort();
  const publicFiles = (await fs.readdir(publicDir)).filter((file) => file.endsWith('.schema.json')).sort();

  assert.deepEqual(publicFiles, sourceFiles);
  for (const file of sourceFiles) {
    const sourceSchema = JSON.parse(await fs.readFile(new URL(file, sourceDir), 'utf-8'));
    const publicSchema = JSON.parse(await fs.readFile(new URL(file, publicDir), 'utf-8'));
    assert.deepEqual(publicSchema, sourceSchema);
  }
});

test('creates runtime config for production defaults', () => {
  const env = {
    NODE_ENV: 'production',
    API_TARGET: 'https://api.example.com',
  };
  const config = createRuntimeConfig({ argv: ['node', 'server.js'], env, cwd: process.cwd() });

  assert.equal(config.mode, 'production');
  assert.equal(config.production, true);
  assert.equal(config.port, 8081);
  assert.equal(config.apiTarget, 'https://api.example.com');
});

test('production server injects SSR HTML into the app shell', async (t) => {
  const cwd = await fs.mkdtemp(path.join(os.tmpdir(), 'mergeos-frontend-ssr-'));
  const clientDist = path.join(cwd, 'client');
  const serverDir = path.join(cwd, 'server');
  const serverEntry = path.join(serverDir, 'entry-server.mjs');
  await fs.mkdir(clientDist, { recursive: true });
  await fs.mkdir(serverDir, { recursive: true });
  await fs.writeFile(
    path.join(clientDist, 'index.html'),
    '<!doctype html><html><body><div id="app"><!--ssr-outlet--></div></body></html>',
  );
  await fs.writeFile(
    serverEntry,
    "export async function render(url) { return `<main id=\"ssr-proof\">${url}</main>`; }\n",
  );

  const server = await createMergeOSServer({
    mode: 'production',
    production: true,
    cwd,
    host: '127.0.0.1',
    port: 0,
    hmrPort: 0,
    apiTarget: 'http://127.0.0.1:65535',
    clientDist,
    serverEntry,
  });
  t.after(() => server.close());
  await new Promise((resolve) => server.listen(0, '127.0.0.1', resolve));

  const address = server.address();
  const response = await fetch(`http://127.0.0.1:${address.port}/admin`);
  const html = await response.text();

  assert.equal(response.status, 200);
  assert.match(response.headers.get('cache-control') || '', /no-store/);
  assert.match(html, /id="ssr-proof"/);
  assert.match(html, />\/admin</);
  assert.doesNotMatch(html, /ssr-outlet/);
  assert.doesNotMatch(html, /<div id="app"><\/div>/);
});

test('production server serves protocol schema assets as JSON', async (t) => {
  const cwd = await fs.mkdtemp(path.join(os.tmpdir(), 'mergeos-frontend-schema-'));
  const clientDist = path.join(cwd, 'client');
  const protocolDir = path.join(clientDist, 'protocol');
  const serverDir = path.join(cwd, 'server');
  const serverEntry = path.join(serverDir, 'entry-server.mjs');
  await fs.mkdir(protocolDir, { recursive: true });
  await fs.mkdir(serverDir, { recursive: true });
  await fs.writeFile(
    path.join(clientDist, 'index.html'),
    '<!doctype html><html><body><div id="app"><!--ssr-outlet--></div></body></html>',
  );
  await fs.writeFile(
    path.join(protocolDir, 'ledger.v1.schema.json'),
    JSON.stringify({ title: 'MergeOS Ledger v1' }),
  );
  await fs.writeFile(serverEntry, "export async function render() { return '<main></main>'; }\n");

  const server = await createMergeOSServer({
    mode: 'production',
    production: true,
    cwd,
    host: '127.0.0.1',
    port: 0,
    hmrPort: 0,
    apiTarget: 'http://127.0.0.1:65535',
    clientDist,
    serverEntry,
  });
  t.after(() => server.close());
  await new Promise((resolve) => server.listen(0, '127.0.0.1', resolve));

  const address = server.address();
  const response = await fetch(`http://127.0.0.1:${address.port}/protocol/ledger.v1.schema.json`);
  const payload = await response.json();

  assert.equal(response.status, 200);
  assert.match(response.headers.get('content-type') || '', /application\/json/);
  assert.equal(payload.title, 'MergeOS Ledger v1');
});

test('production server serves MergeIDE download manifest and preview kit', async (t) => {
  const cwd = await fs.mkdtemp(path.join(os.tmpdir(), 'mergeos-frontend-downloads-'));
  const clientDist = path.join(cwd, 'client');
  const downloadsDir = path.join(clientDist, 'downloads');
  const runbooksDir = path.join(clientDist, 'protocol', 'runbooks');
  const serverDir = path.join(cwd, 'server');
  const serverEntry = path.join(serverDir, 'entry-server.mjs');
  await fs.mkdir(downloadsDir, { recursive: true });
  await fs.mkdir(runbooksDir, { recursive: true });
  await fs.mkdir(serverDir, { recursive: true });
  await fs.writeFile(
    path.join(clientDist, 'index.html'),
    '<!doctype html><html><body><div id="app"><!--ssr-outlet--></div></body></html>',
  );
  await fs.writeFile(
    path.join(downloadsDir, 'mergeide-windows-latest.json'),
    JSON.stringify({ protocol_version: 'mergeos.release-artifact.v1', kind: 'release_artifact' }),
  );
  await fs.writeFile(
    path.join(runbooksDir, 'mergeide-agent.v1.json'),
    JSON.stringify({ protocol_version: 'mergeos.agent-runbook.v1', kind: 'agent_runbook' }),
  );
  await fs.writeFile(path.join(downloadsDir, 'mergeide-preview-kit.md'), '# MergeIDE Preview Kit\n');
  await fs.writeFile(serverEntry, "export async function render() { return '<main></main>'; }\n");

  const server = await createMergeOSServer({
    mode: 'production',
    production: true,
    cwd,
    host: '127.0.0.1',
    port: 0,
    hmrPort: 0,
    apiTarget: 'http://127.0.0.1:65535',
    clientDist,
    serverEntry,
  });
  t.after(() => server.close());
  await new Promise((resolve) => server.listen(0, '127.0.0.1', resolve));

  const address = server.address();
  const manifestResponse = await fetch(`http://127.0.0.1:${address.port}/downloads/mergeide-windows-latest.json`);
  const manifest = await manifestResponse.json();
  const kitResponse = await fetch(`http://127.0.0.1:${address.port}/downloads/mergeide-preview-kit.md`);
  const kit = await kitResponse.text();
  const runbookResponse = await fetch(`http://127.0.0.1:${address.port}/protocol/runbooks/mergeide-agent.v1.json`);
  const runbook = await runbookResponse.json();

  assert.equal(manifestResponse.status, 200);
  assert.match(manifestResponse.headers.get('content-type') || '', /application\/json/);
  assert.equal(manifest.protocol_version, 'mergeos.release-artifact.v1');
  assert.equal(kitResponse.status, 200);
  assert.match(kitResponse.headers.get('content-type') || '', /text\/markdown/);
  assert.match(kit, /MergeIDE Preview Kit/);
  assert.equal(runbookResponse.status, 200);
  assert.match(runbookResponse.headers.get('content-type') || '', /application\/json/);
  assert.equal(runbook.protocol_version, 'mergeos.agent-runbook.v1');
});

test('production server marks hashed assets as immutable', async (t) => {
  const cwd = await fs.mkdtemp(path.join(os.tmpdir(), 'mergeos-frontend-cache-'));
  const clientDist = path.join(cwd, 'client');
  const assetsDir = path.join(clientDist, 'assets');
  const serverDir = path.join(cwd, 'server');
  const serverEntry = path.join(serverDir, 'entry-server.mjs');
  await fs.mkdir(assetsDir, { recursive: true });
  await fs.mkdir(serverDir, { recursive: true });
  await fs.writeFile(path.join(clientDist, 'index.html'), '<!doctype html><html><body><div id="app"><!--ssr-outlet--></div></body></html>');
  await fs.writeFile(path.join(assetsDir, 'index-test.js'), 'window.__mergeos_asset = true;\n');
  await fs.writeFile(serverEntry, "export async function render() { return '<main></main>'; }\n");

  const server = await createMergeOSServer({
    mode: 'production',
    production: true,
    cwd,
    host: '127.0.0.1',
    port: 0,
    hmrPort: 0,
    apiTarget: 'http://127.0.0.1:65535',
    clientDist,
    serverEntry,
  });
  t.after(() => server.close());
  await new Promise((resolve) => server.listen(0, '127.0.0.1', resolve));

  const address = server.address();
  const response = await fetch(`http://127.0.0.1:${address.port}/assets/index-test.js`);
  const script = await response.text();

  assert.equal(response.status, 200);
  assert.match(response.headers.get('cache-control') || '', /immutable/);
  assert.match(script, /__mergeos_asset/);
});

test('API proxy forwards the public frontend host for auth redirects', async (t) => {
  const api = http.createServer((req, res) => {
    res.setHeader('Content-Type', 'application/json');
    res.end(JSON.stringify({
      host: req.headers.host,
      forwardedHost: req.headers['x-forwarded-host'],
      forwardedProto: req.headers['x-forwarded-proto'],
    }));
  });
  t.after(() => api.close());
  await new Promise((resolve) => api.listen(0, '127.0.0.1', resolve));
  const apiAddress = api.address();

  const cwd = await fs.mkdtemp(path.join(os.tmpdir(), 'mergeos-frontend-proxy-'));
  const clientDist = path.join(cwd, 'client');
  const serverDir = path.join(cwd, 'server');
  const serverEntry = path.join(serverDir, 'entry-server.mjs');
  await fs.mkdir(clientDist, { recursive: true });
  await fs.mkdir(serverDir, { recursive: true });
  await fs.writeFile(path.join(clientDist, 'index.html'), '<!doctype html><html><body><div id="app"><!--ssr-outlet--></div></body></html>');
  await fs.writeFile(serverEntry, "export async function render() { return '<main></main>'; }\n");

  const server = await createMergeOSServer({
    mode: 'production',
    production: true,
    cwd,
    host: '127.0.0.1',
    port: 0,
    hmrPort: 0,
    apiTarget: `http://127.0.0.1:${apiAddress.port}`,
    clientDist,
    serverEntry,
  });
  t.after(() => server.close());
  await new Promise((resolve) => server.listen(0, '127.0.0.1', resolve));
  const frontendAddress = server.address();

  const response = await fetch(`http://127.0.0.1:${frontendAddress.port}/api/header-check`);
  const headers = await response.json();

  assert.equal(headers.host, `127.0.0.1:${apiAddress.port}`);
  assert.equal(headers.forwardedHost, `127.0.0.1:${frontendAddress.port}`);
  assert.equal(headers.forwardedProto, 'http');
});

test('WebSocket proxy upgrades /api/ws to the API target', async (t) => {
  let upgradeHeaders;
  const api = http.createServer();
  api.on('upgrade', (req, socket) => {
    upgradeHeaders = req.headers;
    socket.write([
      'HTTP/1.1 101 Switching Protocols',
      'Upgrade: websocket',
      'Connection: Upgrade',
      'Sec-WebSocket-Accept: test',
      '',
      '',
    ].join('\r\n'));
    socket.end();
  });
  t.after(() => api.close());
  await new Promise((resolve) => api.listen(0, '127.0.0.1', resolve));
  const apiAddress = api.address();

  const cwd = await fs.mkdtemp(path.join(os.tmpdir(), 'mergeos-frontend-ws-proxy-'));
  const clientDist = path.join(cwd, 'client');
  const serverDir = path.join(cwd, 'server');
  const serverEntry = path.join(serverDir, 'entry-server.mjs');
  await fs.mkdir(clientDist, { recursive: true });
  await fs.mkdir(serverDir, { recursive: true });
  await fs.writeFile(path.join(clientDist, 'index.html'), '<!doctype html><html><body><div id="app"><!--ssr-outlet--></div></body></html>');
  await fs.writeFile(serverEntry, "export async function render() { return '<main></main>'; }\n");

  const server = await createMergeOSServer({
    mode: 'production',
    production: true,
    cwd,
    host: '127.0.0.1',
    port: 0,
    hmrPort: 0,
    apiTarget: `http://127.0.0.1:${apiAddress.port}`,
    clientDist,
    serverEntry,
  });
  t.after(() => server.close());
  await new Promise((resolve) => server.listen(0, '127.0.0.1', resolve));
  const frontendAddress = server.address();

  const response = await new Promise((resolve, reject) => {
    const client = net.createConnection(frontendAddress.port, '127.0.0.1');
    let data = '';
    client.on('connect', () => {
      client.write([
        'GET /api/ws HTTP/1.1',
        `Host: 127.0.0.1:${frontendAddress.port}`,
        'Connection: Upgrade',
        'Upgrade: websocket',
        'Sec-WebSocket-Key: test',
        'Sec-WebSocket-Version: 13',
        '',
        '',
      ].join('\r\n'));
    });
    client.on('data', (chunk) => {
      data += chunk.toString('utf-8');
      if (data.includes('\r\n\r\n')) {
        client.destroy();
        resolve(data);
      }
    });
    client.on('error', reject);
    client.setTimeout(2500, () => reject(new Error('websocket proxy timed out')));
  });

  assert.match(response, /101 Switching Protocols/);
  assert.equal(upgradeHeaders.host, `127.0.0.1:${apiAddress.port}`);
  assert.equal(upgradeHeaders['x-forwarded-host'], `127.0.0.1:${frontendAddress.port}`);
  assert.equal(upgradeHeaders['x-forwarded-proto'], 'http');
});

test('shared Vue entry leaves browser mounting to the client hydration entry', async () => {
  const main = await fs.readFile(new URL('./src/main.js', import.meta.url), 'utf-8');
  const client = await fs.readFile(new URL('./src/entry-client.js', import.meta.url), 'utf-8');

  assert.match(main, /createSSRApp/);
  assert.doesNotMatch(main, /\.mount\(/);
  assert.doesNotMatch(main, /typeof document|createClientApp/);
  assert.match(client, /firstElementChild/);
  assert.match(client, /const initialPath = window\.location\.pathname/);
  assert.match(client, /createHydratedApp\(initialPath\) : createClientApp\(App, \{ initialPath \}\)/);
  assert.match(client, /app\.mount\('#app'\)/);
});
