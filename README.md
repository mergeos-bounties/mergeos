# MergeOS

[![Build and deploy](https://github.com/mergeos-bounties/mergeos/actions/workflows/deploy.yml/badge.svg?branch=master)](https://github.com/mergeos-bounties/mergeos/actions/workflows/deploy.yml?query=branch%3Amaster)
[![GitHub stars](https://img.shields.io/github/stars/mergeos-bounties/mergeos?style=flat&label=stars)](https://github.com/mergeos-bounties/mergeos/stargazers)

MergeOS is an AI-assisted software maintenance and bounty operating system. A customer funds a project, receives internal project tokens, and MergeOS turns the work into claimable tasks that can be completed by human contributors, AI agents, or hybrid teams.

This repository is the current MergeOS MVP: Go backend, Vue SSR frontend, project funding flow, bounty workspace generation, GitHub issue import, evidence attachments, notifications, admin review, and proof ledger.

`scan/` is the public MergeOS Scan explorer for `scan.mergeos.shop`. It reads the public ledger API and presents MRG token mints, escrow movements, task reserves, payouts, addresses, transaction hashes, and hash-chain proof in a BscScan-style interface.

`sdk/` is the lightweight JavaScript MergeOS SDK for public marketplace/ledger feeds, authenticated task and workflow APIs, admin ops queue calls, and WebSocket event stream helpers.

`contracts/` contains the MRG token, treasury, and escrow contract sources used to model funded projects, platform fees, task reserves, worker payouts, and refunds.

`protocol/` defines the open MergeOS task, workflow graph, and realtime event schemas used by external agents and integrations.

## Production

- App: [https://mergeos.shop](https://mergeos.shop)
- Admin: [https://uta.mergeos.shop](https://uta.mergeos.shop)
- Scan explorer: [https://scan.mergeos.shop](https://scan.mergeos.shop)

## Repository Documents

- [README-INDEX.md](README-INDEX.md): docs map and bounty tracking index.
- [BOUNTY-POLICY.md](BOUNTY-POLICY.md): bounty policy and reward rules.
- [CONTRIBUTING.md](CONTRIBUTING.md): contributor setup, PR rules, tests, and bounty workflow.
- [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md): community behavior expectations and enforcement.
- [SECURITY.md](SECURITY.md): private vulnerability reporting policy.
- [SUPPORT.md](SUPPORT.md): support channels for bugs, features, bounties, and security.
- [LICENSE](LICENSE): MIT license.
- [Claim Token issue #1](https://github.com/mergeos-bounties/mergeos/issues/1): public bounty claim intake.

## New Workflow

1. The customer registers or logs in.
2. The customer creates a new project or imports an existing repository.
3. The customer funds escrow through PayPal, crypto, or the local development verifier.
4. MergeOS records the funded project, mints internal `MRG` token credit, and creates a bounty workspace.
5. The system splits the project into tasks with reward pools, worker type, agent suggestion, and acceptance criteria.
6. Contributors claim available work and open PRs with implementation evidence.
7. Maintainers review the PR, verify the work, merge valid changes, and release the token reward.

## Product Scope

MergeOS currently supports:

- Customer auth with email/password bearer sessions.
- GitHub App login that creates or links a MergeOS account to an MRG wallet.
- Guest MRG wallet creation with BSC-style `0x...` addresses from MergeOS Scan.
- Two environment modes: `local` and `production`.
- Project creation with budget, payment method, attachments, and escrow status.
- Local payment verification through `LOCAL-PAID`.
- PayPal Orders v2 adapter.
- EVM native or ERC-20 receipt verification.
- GitHub open issue import with heuristic complexity, estimated MRG reward, and estimated hours.
- GitHub reward aliases. If a worker has not linked a wallet yet, payouts can still target `github:username`; once linked, payouts route to the user's `0x...` wallet address.
- Local git bounty workspaces or GitHub private bounty repos when `GITHUB_TOKEN` is configured.
- Static repository scan for dependency manifests, technical-debt markers, and secret-hygiene findings.
- Project task dependency graph generation for workflow routing, effort metadata, dependency edges, and release readiness.
- Project escrow summaries for reserve, release, remaining-balance, and overdrawn payout review.
- Project-level live PR monitoring for customer dashboards, including readiness signals reused from admin review.
- Project-level AI agent action logging for review, test, generate, deploy, and scan workflow events.
- Task reward allocation, worker kind, suggested agent type, and acceptance criteria.
- Worker reputation and risk audit signals for marketplace/admin payout review.
- Proof ledger entries with hash chaining.
- Public ledger hash-chain verification API for Scan and external auditors.
- SMTP notifications when configured, persisted notification records when SMTP is not configured.
- Admin APIs for users, projects, tasks, attachments, notifications, ledger, and SSL review.
- JavaScript SDK helpers for public feeds, project deployment workflow, AI workflow, task acceptance, worker dashboards, admin ops queue, and WebSocket events.
- WebSocket realtime stream with connection readiness and public live-feed snapshot events.
- Public live feed stats for funded projects, open tasks, active contributors, active agent lanes, AI actions, and verified escrow.
- MRG contract sources for token issuance, treasury release, project escrow, task reserve, payout, and refund flows.
- Open protocol schemas for task specs, AI agent lanes, workflow graphs, and realtime events.

Roadmap items include deeper AI codebase scanning, automated PR verification, fraud detection, and automatic real payout execution.

## Stack

- Backend: Go `net/http`
- Storage: PostgreSQL when `DATABASE_URL` is set, with legacy JSON state fallback for local development
- Frontend: Vue 3 + Vite SSR
- Admin: Vue 3 + Vite SSR admin console
- Scan: Vue 3 + Vite static explorer served from `scan.mergeos.shop`
- Token symbol: `MRG` by default through `TOKEN_SYMBOL`
- Bounty repos: local git under `BOUNTY_ROOT`, or GitHub private repos with `GITHUB_TOKEN`
- Payments: local verifier, PayPal, EVM native/ERC-20 verifier

## Local Testing

Use Docker Compose for local testing. It starts PostgreSQL, the Go API, the Vue SSR frontend, the admin console, and the Scan explorer with the same wiring used by deployment.

Prerequisites:

- Docker Desktop or Docker Engine with the Compose plugin.
- Local ports `5432`, `8080`, `5173`, `5174`, and `5175` available.

Start everything:

```powershell
docker compose up --build
```

Open:

- Frontend: `http://127.0.0.1:5173`
- Admin: `http://127.0.0.1:5174`
- Scan explorer: `http://127.0.0.1:5175`
- API health: `http://127.0.0.1:8080/api/health`
- PostgreSQL: `127.0.0.1:5432`, database `mergeos_local`, user `mergeos`, password `mergeos`

Local test credentials:

- Admin email: `admin@gmail.com`
- Admin password: `Admin123`
- Local payment reference: `LOCAL-PAID`

Useful commands:

```powershell
# Stop containers but keep local Postgres and uploaded/bounty data volumes.
docker compose down

# Reset all local Docker data and start from an empty PostgreSQL database.
docker compose down -v
docker compose up --build

# Rebuild one service after changing its source.
docker compose up --build backend
docker compose up --build frontend
docker compose up --build admin
docker compose up --build scan
```

If a host port is already busy, override only the published host port and keep the container port unchanged:

```powershell
$env:MERGEOS_BACKEND_PORT='18080'
$env:MERGEOS_FRONTEND_PORT='15173'
$env:MERGEOS_ADMIN_PORT='15174'
$env:MERGEOS_SCAN_PORT='15175'
$env:MERGEOS_POSTGRES_PORT='15432'
docker compose up --build
```

Compose storage:

- `postgres-data`: PostgreSQL data.
- `backend-data`: uploaded files, generated bounty repos, and optional legacy JSON import/export path.

The backend runs in `MERGEOS_ENV=local`, sets `DATABASE_URL=postgres://mergeos:mergeos@postgres:5432/mergeos_local?sslmode=disable`, disables SSL review calls for local tests, and runs the embedded PostgreSQL migrations automatically on startup.

GitHub App user authorization for local testing is optional. To enable "Continue with GitHub" and wallet linking, create a GitHub App, enable user authorization, and set these before starting Compose:

```powershell
$env:MERGEOS_GITHUB_APP_ID='your-app-id'
$env:MERGEOS_GITHUB_APP_CLIENT_ID='your-github-app-client-id'
$env:MERGEOS_GITHUB_APP_CLIENT_SECRET='your-github-app-client-secret'
docker compose up --build
```

For local callbacks, add these authorization callback URLs to the GitHub App as needed:

- `http://127.0.0.1:5173/`
- `http://127.0.0.1:5175/`

Google login is also optional. To enable "Continue with Google", create a Google OAuth client and set these before starting Compose:

```powershell
$env:MERGEOS_GOOGLE_CLIENT_ID='your-google-client-id'
$env:MERGEOS_GOOGLE_CLIENT_SECRET='your-google-client-secret'
docker compose up --build
```

For local Google callbacks, add `http://127.0.0.1:5173/api/auth/google/callback`.

## Manual Service Development

Manual runs are optional and only useful when debugging one service outside Docker. For normal local testing, use Docker Compose above so PostgreSQL, ports, API proxying, and migrations stay consistent.

Run the backend first:

```powershell
cd backend
Copy-Item .env.local.example .env.local
go run ./cmd/mergeos
```

Then run the service you are changing:

```powershell
cd frontend
Copy-Item .env.local.example .env.local
npm install
npm run local
```

Admin:

```powershell
cd admin
Copy-Item .env.local.example .env.local
npm install
npm run local
```

Scan:

```powershell
cd scan
Copy-Item .env.local.example .env.local
npm install
npm run dev
```

## Production

Build the SSR frontend:

```powershell
cd frontend
Copy-Item .env.production.example .env.production
npm install
npm run build:production
```

Build the admin frontend:

```powershell
cd admin
Copy-Item .env.production.example .env.production
npm install
npm run build
```

Build the scan explorer:

```powershell
cd scan
Copy-Item .env.production.example .env.production
npm install
npm run build
```

Start the backend:

```powershell
cd backend
Copy-Item .env.production.example .env.production
$env:MERGEOS_ENV='production'
go run ./cmd/mergeos
```

Start the SSR frontend:

```powershell
cd frontend
npm run production
```

Before real deployment, set production values in `backend/.env.production`: `ADMIN_PASSWORD`, PayPal credentials, crypto verifier settings, GitHub repo settings, SMTP settings, receiver addresses, and SSL review domains. The GitHub deploy workflow builds `scan/`, serves it statically from nginx, proxies `/api/` to the MergeOS backend, and requests certificates for `mergeos.shop`, `uta.mergeos.shop`, and `scan.mergeos.shop`.

## Environment Reference

Backend examples:

- `backend/.env.local.example`
- `backend/.env.production.example`
- `backend/.env.example`

Frontend examples:

- `frontend/.env.local.example`
- `frontend/.env.production.example`

Admin examples:

- `admin/.env.local.example`
- `admin/.env.production.example`

Scan examples:

- `scan/.env.local.example`
- `scan/.env.production.example`

Important backend variables:

- `MERGEOS_ENV`: `local` or `production`
- `DATABASE_URL`: PostgreSQL connection string
- `MERGEOS_STATE_PATH`: legacy JSON state path or import source
- `TOKEN_SYMBOL`: token label shown by the app, default `MRG`
- `PRIMARY_DOMAIN`, `ADMIN_DOMAIN`, `SCAN_DOMAIN`: production hostnames, defaulting to `mergeos.shop`, `uta.mergeos.shop`, and `scan.mergeos.shop`
- `PLATFORM_FEE_BPS`: platform fee basis points
- `DEV_PAYMENT_ENABLED` and `DEV_PAYMENT_CODE`: local verifier
- `PAYPAL_ENV`, `PAYPAL_CLIENT_ID`, `PAYPAL_CLIENT_SECRET`: PayPal Orders v2
- `CRYPTO_RPC_URL`, `CRYPTO_RECEIVER`, `CRYPTO_ASSET`, `CRYPTO_TOKEN_CONTRACT`: crypto verifier
- `GITHUB_TOKEN`, `GITHUB_OWNER`, `GITHUB_OWNER_TYPE`: backend runtime values for GitHub bounty repo creation and admin PR merge actions
- `MERGEOS_GITHUB_TOKEN`: Docker Compose and GitHub Actions secret name that maps into backend `GITHUB_TOKEN`; use a personal access token with repo write access, not the automatic GitHub Actions token
- `GEMINI_API_KEYS`: comma-separated Gemini API key pool used to seed the initial LLM key table; admins can add Gemini, OpenAI, Anthropic, Groq, OpenRouter, DeepSeek, and Mistral tokens later in the admin UI, while request counts and key status are tracked in the database
- `GEMINI_REVIEW_MODEL`: legacy Gemini reviewer model default. Admin settings can now select the active LLM provider and model at runtime.
- `GEMINI_REVIEW_WEBHOOK_SECRET`: GitHub webhook secret used to verify `X-Hub-Signature-256`
- `GEMINI_REVIEW_MAX_PATCH_BYTES`: max patch context sent to Gemini, default `70000`
- `GITHUB_APP_ID`, `GITHUB_APP_CLIENT_ID`, `GITHUB_APP_CLIENT_SECRET`: backend runtime values for GitHub App user authorization, login, and MRG wallet linking
- `MERGEOS_GITHUB_APP_ID`, `MERGEOS_GITHUB_APP_CLIENT_ID`, `MERGEOS_GITHUB_APP_CLIENT_SECRET`: Docker Compose and GitHub Actions secret names that map into the backend runtime values
- `GITHUB_OAUTH_CLIENT_ID`, `GITHUB_OAUTH_CLIENT_SECRET`: legacy backend aliases still accepted for older OAuth configuration
- `GOOGLE_CLIENT_ID`, `GOOGLE_CLIENT_SECRET`: backend runtime values for Google login
- `MERGEOS_GOOGLE_CLIENT_ID`, `MERGEOS_GOOGLE_CLIENT_SECRET`: Docker Compose and GitHub Actions secret names that map into Google login runtime values
- `BOUNTY_ROOT`: local child bounty repo root
- `UPLOAD_ROOT`: attachment storage root
- `SMTP_HOST`, `SMTP_PORT`, `SMTP_USERNAME`, `SMTP_PASSWORD`, `SMTP_FROM`: email notifications

## API Surface

Public:

- `GET /api/health`
- `GET /api/config`
- `GET /api/public/ledger`
- `GET /api/public/ledger/verify`
- `GET /api/public/marketplace`
- `GET /api/public/live-feed`
- `GET /api/public/protocol`
- `GET /api/public/protocol/tasks`
- `GET /api/public/protocol/agents`
- `GET /api/public/protocol/ledger`
- `GET /api/public/protocol/events`
- `WS /api/ws`
- `POST /api/public/repo/issues`
- `POST /api/integrations/github/pr-review` GitHub webhook receiver for automated LLM PR review. Configure GitHub Webhooks with Payload URL `https://uta.mergeos.shop/api/integrations/github/pr-review`, Content type `application/json`, the same secret as `GEMINI_REVIEW_WEBHOOK_SECRET`, and events `Pull requests` plus `Issue comments`.

Auth:

- `POST /api/auth/register`
- `POST /api/auth/login`
- `POST /api/auth/github`
- `GET /api/auth/me`
- `POST /api/auth/logout`

Wallet:

- `POST /api/wallets`
- `GET /api/wallets/{address}`
- `POST /api/wallets/link`

Customer:

- `POST /api/payments/paypal/orders`
- `POST /api/uploads`
- `GET /api/uploads/{id}/download`
- `POST /api/projects`
- `GET /api/projects`
- `GET /api/projects/{id}/escrow`
- `GET /api/projects/{id}/dashboard`
- `GET /api/projects/{id}/pull-requests`
- `GET /api/projects/{id}/deployment`
- `GET /api/projects/{id}/ai-workflow`
- `GET /api/projects/{id}/task-graph`
- `GET /api/projects/{id}/protocol/workflow`
- `GET /api/projects/{id}/repo-scan`
- `GET /api/projects/{id}/protocol/scan`
- `POST /api/projects/{id}/repo-sync`
- `POST /api/projects/{id}/agent-actions`
- `GET /api/tasks`
- `POST /api/tasks/{id}/accept`
- `GET /api/workers/me`
- `POST /api/disputes`
- `GET /api/notifications`
- `GET /api/ledger`

Admin:

- `GET /api/admin/summary`
- `GET /api/admin/users`
- `GET /api/admin/projects`
- `GET /api/admin/tasks`
- `GET /api/admin/notifications`
- `GET /api/admin/attachments`
- `GET /api/admin/ledger`
- `GET /api/admin/ops-queue`
- `GET /api/admin/reputation`
- `POST /api/admin/ledger/credits`
- `GET /api/admin/ssl`
- `POST /api/admin/ssl/review`
- `GET /api/admin/gemini/keys`
- `POST /api/admin/gemini/keys`
- `PATCH /api/admin/gemini/keys/{id}`
- `POST /api/admin/gemini/keys/{id}/test`
- `GET /api/admin/gemini/webhooks`
