import fs from 'node:fs/promises';
import { createReadStream } from 'node:fs';
import http from 'node:http';
import https from 'node:https';
import path from 'node:path';
import { fileURLToPath, pathToFileURL } from 'node:url';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
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

function readArgValue(argv, name) {
  const inline = argv.find((arg) => arg.startsWith(`${name}=`));
  if (inline) return inline.slice(name.length + 1);
  const index = argv.indexOf(name);
  if (index >= 0) return argv[index + 1] || '';
  return '';
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

export function createRuntimeConfig({ argv = process.argv, env = process.env, cwd = __dirname } = {}) {
  const mode = resolveMode(argv, env);
  const production = shouldRunProduction(argv, env, mode);
  const port = Number(env.ADMIN_FRONTEND_PORT || env.ADMIN_PORT || (production ? 8082 : 5174));
  return {
    mode,
    production,
    cwd,
    host: env.ADMIN_FRONTEND_HOST || env.ADMIN_HOST || '127.0.0.1',
    port,
    hmrPort: Number(env.VITE_HMR_PORT || port + 10000),
    apiTarget: env.API_TARGET || 'http://127.0.0.1:8080',
    clientDist: path.resolve(cwd, env.CLIENT_DIST || 'dist/client'),
    serverEntry: path.resolve(cwd, env.SERVER_ENTRY || 'dist/server/entry-server.js'),
  };
}

export async function createAdminServer(config) {
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

  return http.createServer(async (req, res) => {
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
}

async function renderUrl(req, res, context) {
  const route = req.url || '/';
  const template = context.vite
    ? await context.vite.transformIndexHtml(route, await fs.readFile(path.resolve(context.cwd, 'index.html'), 'utf-8'))
    : context.productionTemplate;
  const render = context.vite
    ? (await context.vite.ssrLoadModule('/src/entry-server.js')).render
    : context.productionRender;
  const html = template.replace('<!--ssr-outlet-->', await render(route));
  res.statusCode = 200;
  res.setHeader('Content-Type', 'text/html; charset=utf-8');
  res.end(html);
}

async function serveStatic(req, res, clientDist) {
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
  const proxyReq = transport.request({
    protocol: target.protocol,
    hostname: target.hostname,
    port: target.port,
    method: req.method,
    path: req.url,
    headers: {
      ...req.headers,
      host: target.host,
    },
  }, (proxyRes) => {
    res.writeHead(proxyRes.statusCode || 500, proxyRes.headers);
    proxyRes.pipe(res);
  });

  proxyReq.on('error', (error) => {
    res.statusCode = 502;
    res.setHeader('Content-Type', 'application/json; charset=utf-8');
    res.end(JSON.stringify({ error: `api proxy failed: ${error.message}` }));
  });

  req.pipe(proxyReq);
}

export async function startServer({ argv = process.argv, env = process.env, cwd = __dirname } = {}) {
  const config = createRuntimeConfig({ argv, env, cwd });
  const server = await createAdminServer(config);
  server.listen(config.port, config.host, () => {
    console.log(`MergeOS SSR admin (${config.mode}) listening on http://${config.host}:${config.port}`);
  });
  return { server, config };
}

if (process.argv[1] && pathToFileURL(process.argv[1]).href === import.meta.url) {
  startServer().catch((error) => {
    console.error(error);
    process.exit(1);
  });
}
