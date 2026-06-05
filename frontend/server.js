import fs from 'node:fs/promises';
import { createReadStream } from 'node:fs';
import http from 'node:http';
import https from 'node:https';
import net from 'node:net';
import path from 'node:path';
import tls from 'node:tls';
import { fileURLToPath, pathToFileURL } from 'node:url';
import { renderSeoHead } from './src/seo.js';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const defaultClientDist = path.resolve(__dirname, 'dist/client');
const defaultServerEntry = path.resolve(__dirname, 'dist/server/entry-server.js');
const mimeTypes = {
  '.css': 'text/css; charset=utf-8',
  '.html': 'text/html; charset=utf-8',
  '.js': 'text/javascript; charset=utf-8',
  '.json': 'application/json; charset=utf-8',
  '.png': 'image/png',
  '.jpg': 'image/jpeg',
  '.jpeg': 'image/jpeg',
  '.svg': 'image/svg+xml',
  '.webp': 'image/webp',
  '.woff': 'font/woff',
  '.woff2': 'font/woff2',
};


export function normalizeMode(value) {
  switch (String(value || '').trim().toLowerCase()) {
    case 'prod':
    case 'production':
      return 'production';
    case 'dev':
    case 'development':
    case 'local':
    case '':
      return 'local';
    default:
      return 'local';
  }
}

export function resolveMode(argv = process.argv, env = process.env) {
  const modeArg = readArgValue(argv, '--mode');
  if (modeArg) return normalizeMode(modeArg);
  if (argv.includes('--prod')) return 'production';
  return normalizeMode(env.MERGEOS_ENV || env.NODE_ENV);
}

export function shouldRunProduction(argv = process.argv, env = process.env, mode = resolveMode(argv, env)) {
  return mode === 'production' || argv.includes('--prod') || env.NODE_ENV === 'production';
}

export function parseEnvText(text) {
  const entries = {};
  for (const rawLine of String(text || '').split('\n')) {
    const line = rawLine.trim();
    if (!line || line.startsWith('#')) continue;
    const separator = line.indexOf('=');
    if (separator <= 0) continue;
    const key = line.slice(0, separator).trim();
    let value = line.slice(separator + 1).trim();
    if (!key) continue;
    value = value.replace(/^['"]|['"]$/g, '');
    entries[key] = value;
  }
  return entries;
}

export async function loadEnvFiles(mode, { cwd = __dirname, env = process.env } = {}) {
  for (const fileName of [`.env.${normalizeMode(mode)}`, '.env']) {
    let data;
    try {
      data = await fs.readFile(path.join(cwd, fileName), 'utf-8');
    } catch {
      continue;
    }
    for (const [key, value] of Object.entries(parseEnvText(data))) {
      if (String(env[key] || '').trim() === '') {
        env[key] = value;
      }
    }
  }
}

export function createRuntimeConfig({ argv = process.argv, env = process.env, cwd = __dirname } = {}) {
  const mode = resolveMode(argv, env);
  const production = shouldRunProduction(argv, env, mode);
  const port = Number(env.FRONTEND_PORT || env.SSR_PORT || (production ? 8081 : 5173));
  return {
    mode,
    production,
    cwd,
    host: env.FRONTEND_HOST || '127.0.0.1',
    port,
    hmrPort: Number(env.VITE_HMR_PORT || port + 10000),
    apiTarget: env.API_TARGET || 'http://localhost:8080',
    clientDist: path.resolve(cwd, env.CLIENT_DIST || 'dist/client'),
    serverEntry: path.resolve(cwd, env.SERVER_ENTRY || 'dist/server/entry-server.js'),
  };
}

export async function createMergeOSServer(config) {
  let vite;
  let productionTemplate;
  let productionRender;

  if (!config.production) {
    const { createServer } = await import('vite');
    vite = await createServer({
      appType: 'custom',
      server: {
        middlewareMode: true,
        hmr: { port: config.hmrPort },
      },
    });
  } else {
    productionTemplate = await fs.readFile(path.join(config.clientDist, 'index.html'), 'utf-8');
    productionRender = (await import(pathToFileURL(config.serverEntry))).render;
  }

  const server = http.createServer(async (req, res) => {
    attachConnectionErrorHandlers(req, res);
    try {
      if (req.url?.startsWith('/api')) {
        proxyApi(req, res, config.apiTarget);
        return;
      }

      if (config.production && await serveStatic(req, res, config.clientDist)) {
        return;
      }

      if (vite) {
        vite.middlewares(req, res, async () => {
          await renderUrl(req, res, { vite, productionTemplate, productionRender, cwd: config.cwd });
        });
        return;
      }

      await renderUrl(req, res, { vite, productionTemplate, productionRender, cwd: config.cwd });
    } catch (error) {
      if (vite) vite.ssrFixStacktrace(error);
      res.statusCode = 500;
      res.setHeader('Content-Type', 'text/plain; charset=utf-8');
      res.end(error.stack || error.message);
    }
  });

  server.on('connection', (socket) => {
    if (!socket.__mergeosErrorHandlerAttached) {
      socket.__mergeosErrorHandlerAttached = true;
      socket.on('error', handleConnectionError);
    }
  });

  server.on('clientError', (error, socket) => {
    if (!isExpectedConnectionError(error)) {
      console.error(error);
    }
    socket.destroy();
  });

  server.on('upgrade', (req, socket, head) => {
    if (!req.url?.startsWith('/api/ws')) {
      socket.destroy();
      return;
    }
    proxyWs(req, socket, head, config.apiTarget);
  });

  return server;
}

function attachConnectionErrorHandlers(req, res) {
  req.on('error', handleConnectionError);
  res.on('error', handleConnectionError);
  if (req.socket && !req.socket.__mergeosErrorHandlerAttached) {
    req.socket.__mergeosErrorHandlerAttached = true;
    req.socket.on('error', handleConnectionError);
  }
}

function handleConnectionError(error) {
  if (isExpectedConnectionError(error)) return;
  console.error(error);
}

function isExpectedConnectionError(error) {
  return ['ECONNRESET', 'EPIPE', 'ERR_STREAM_PREMATURE_CLOSE'].includes(error?.code);
}

export async function startServer({ argv = process.argv, env = process.env, cwd = __dirname } = {}) {
  const mode = resolveMode(argv, env);
  await loadEnvFiles(mode, { cwd, env });
  env.MERGEOS_ENV = mode;
  if (mode === 'production') env.NODE_ENV = 'production';

  const config = createRuntimeConfig({ argv, env, cwd });
  const server = await createMergeOSServer(config);
  server.listen(config.port, config.host, () => {
    console.log(`MergeOS SSR frontend (${config.mode}) listening on http://${config.host}:${config.port}`);
  });
  return { server, config };
}

async function renderUrl(req, res, context) {
  const url = req.url || '/';
  const template = context.vite
    ? await context.vite.transformIndexHtml(url, await fs.readFile(path.resolve(context.cwd, 'index.html'), 'utf-8'))
    : context.productionTemplate;
  const render = context.vite
    ? (await context.vite.ssrLoadModule('/src/entry-server.js')).render
    : context.productionRender;
  const appHtml = await render(url);
  const origin = publicOriginFromRequest(req);
  const html = injectSeoHead(template, url, origin).replace('<!--ssr-outlet-->', appHtml);
  res.statusCode = 200;
  res.setHeader('Content-Type', 'text/html; charset=utf-8');
  res.end(html);
}

function publicOriginFromRequest(req) {
  const host = req.headers['x-forwarded-host'] || req.headers.host || '127.0.0.1';
  const proto = req.headers['x-forwarded-proto'] || (req.socket.encrypted ? 'https' : 'http');
  return `${Array.isArray(proto) ? proto[0] : proto}://${Array.isArray(host) ? host[0] : host}`;
}

function injectSeoHead(template, url, origin) {
  const seoHead = renderSeoHead(url, { origin });
  if (template.includes('<!--seo-head-->')) {
    return template.replace('<!--seo-head-->', seoHead);
  }
  if (/<title\b[^>]*>[\s\S]*?<\/title>/i.test(template)) {
    return template.replace(/<title\b[^>]*>[\s\S]*?<\/title>/i, seoHead);
  }
  return template.replace('</head>', `${seoHead}\n  </head>`);
}

async function serveStatic(req, res, clientDist = defaultClientDist) {
  const pathname = decodeURIComponent(new URL(req.url || '/', 'http://127.0.0.1').pathname);
  if (pathname === '/') return false;

  const requestedPath = path.normalize(path.join(clientDist, pathname));
  if (!requestedPath.startsWith(clientDist)) {
    res.statusCode = 403;
    res.end('Forbidden');
    return true;
  }

  let stat;
  try {
    stat = await fs.stat(requestedPath);
  } catch {
    return false;
  }
  if (!stat.isFile()) return false;

  res.statusCode = 200;
  res.setHeader('Content-Type', mimeTypes[path.extname(requestedPath)] || 'application/octet-stream');
  createReadStream(requestedPath).pipe(res);
  return true;
}

function proxyApi(req, res, apiTarget) {
  const target = new URL(apiTarget);
  const transport = target.protocol === 'https:' ? https : http;
  const forwardedHost = req.headers['x-forwarded-host'] || req.headers.host || target.host;
  const forwardedProto = req.headers['x-forwarded-proto'] || (req.socket.encrypted ? 'https' : 'http');
  const proxyReq = transport.request({
    protocol: target.protocol,
    hostname: target.hostname,
    port: target.port,
    method: req.method,
    path: req.url,
    headers: {
      ...req.headers,
      host: target.host,
      'x-forwarded-host': forwardedHost,
      'x-forwarded-proto': forwardedProto,
    },
  }, (proxyRes) => {
    res.writeHead(proxyRes.statusCode || 500, proxyRes.headers);
    proxyRes.on('error', handleConnectionError);
    proxyRes.pipe(res);
  });

  proxyReq.on('error', (error) => {
    if (res.destroyed || isExpectedConnectionError(error)) return;
    res.statusCode = 502;
    res.setHeader('Content-Type', 'application/json; charset=utf-8');
    res.end(JSON.stringify({ error: `api proxy failed: ${error.message}` }));
  });

  req.on('aborted', () => {
    proxyReq.destroy();
  });
  req.pipe(proxyReq);
}

function proxyWs(req, socket, head, apiTarget) {
  const target = new URL(apiTarget);
  const useTLS = target.protocol === 'https:';
  const targetPort = Number(target.port || (useTLS ? 443 : 80));
  const forwardedHost = req.headers['x-forwarded-host'] || req.headers.host || target.host;
  const forwardedProto = req.headers['x-forwarded-proto'] || (req.socket.encrypted ? 'https' : 'http');
  const proxySocket = useTLS
    ? tls.connect({ host: target.hostname, port: targetPort, servername: target.hostname })
    : net.connect({ host: target.hostname, port: targetPort });
  let targetConnected = false;

  proxySocket.once(useTLS ? 'secureConnect' : 'connect', () => {
    targetConnected = true;
    const headers = {
      ...req.headers,
      host: target.host,
      'x-forwarded-host': forwardedHost,
      'x-forwarded-proto': forwardedProto,
    };
    const lines = [`${req.method || 'GET'} ${req.url || '/api/ws'} HTTP/${req.httpVersion || '1.1'}`];
    for (const [name, value] of Object.entries(headers)) {
      if (Array.isArray(value)) {
        for (const item of value) lines.push(`${name}: ${item}`);
      } else if (value !== undefined) {
        lines.push(`${name}: ${value}`);
      }
    }
    lines.push('', '');
    proxySocket.pipe(socket, { end: false });
    socket.pipe(proxySocket, { end: false });
    proxySocket.write(lines.join('\r\n'));
    if (head?.length) proxySocket.write(head);
    proxySocket.on('end', () => socket.end());
    proxySocket.on('close', () => socket.destroy());
    socket.on('close', () => proxySocket.destroy());
    proxySocket.on('error', () => socket.destroy());
    socket.on('error', () => proxySocket.destroy());
    proxySocket.resume();
    socket.resume();
  });

  proxySocket.on('error', () => {
    if (!targetConnected && !socket.destroyed) {
      socket.write('HTTP/1.1 502 Bad Gateway\r\n\r\n');
    }
    socket.destroy();
  });
}

function readArgValue(argv, name) {
  const inline = argv.find((arg) => arg.startsWith(`${name}=`));
  if (inline) return inline.slice(name.length + 1);
  const index = argv.indexOf(name);
  if (index >= 0) return argv[index + 1] || '';
  return '';
}

if (process.argv[1] && pathToFileURL(process.argv[1]).href === import.meta.url) {
  startServer().catch((error) => {
    console.error(error);
    process.exit(1);
  });
}

export const paths = {
  clientDist: defaultClientDist,
  serverEntry: defaultServerEntry,
};
