import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs/promises';
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
  assert.match(html, /id="ssr-proof"/);
  assert.match(html, />\/admin</);
  assert.doesNotMatch(html, /ssr-outlet/);
  assert.doesNotMatch(html, /<div id="app"><\/div>/);
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
