# MergeOS

MergeOS is a private main platform for customer-funded website delivery. A client registers, logs in, submits contact/project details, verifies PayPal or crypto payment, receives internal MERGE credits, and MergeOS creates a private child bounty repo under `mergeos-bounties` with payable human/agent tasks.

## Stack

- Backend: Go `net/http`
- Frontend: Vue 3 + Vite
- Auth: email/password accounts with bearer sessions
- Local state: file-backed JSON at `backend/data/mergeos-state.json`
- Child bounty repos: local git repos by default, GitHub private repos in `mergeos-bounties` when configured
- Payment adapters: PayPal Orders v2, EVM native/ERC20 receipt verifier, local dev verifier
- Email: SMTP when configured, persisted email log otherwise

## Environments

MergeOS has two explicit run modes:

- `local`: loads `.env.local`, enables the local payment verifier by default, and can auto-promote the first user/admin bootstrap.
- `production`: loads `.env.production`, disables dev payment by default, requires explicit admin credentials, and defaults PayPal to `live`.

Real environment variables always win over env file values. Mode-specific files are loaded first, then `.env` only fills missing values.

Example files:

- `backend/.env.local.example`
- `backend/.env.production.example`
- `frontend/.env.local.example`
- `frontend/.env.production.example`

## Run Local

Terminal 1:

```powershell
cd backend
Copy-Item .env.local.example .env.local
go run ./cmd/mergeos
```

Terminal 2:

```powershell
cd frontend
Copy-Item .env.local.example .env.local
npm install
npm run local
```

Open `http://127.0.0.1:5173`.

## Run Production

Build the SSR frontend:

```powershell
cd frontend
Copy-Item .env.production.example .env.production
npm install
npm run build:production
```

Start the backend and SSR frontend with production env files:

```powershell
cd backend
Copy-Item .env.production.example .env.production
$env:MERGEOS_ENV='production'
go run ./cmd/mergeos
```

```powershell
cd frontend
npm run production
```

Before real deployment, set real production values in `backend/.env.production`: `ADMIN_PASSWORD`, PayPal, crypto, GitHub, SMTP, and any receiver addresses.

## Local Flow

1. Register or log in from the MergeOS customer portal.
2. Use `LOCAL-PAID` as the payment reference in local mode.
3. Submit a funded website project.
4. MergeOS writes state, mints MERGE credits, creates a child git repo under `bounties/`, splits tasks, commits the repo, and logs customer emails.
5. Accept a task with a human/agent manifest to create a ledger proof and payment entry.

## Live Adapters

Use `backend/.env.example` as the environment reference.

PayPal:

- `PAYPAL_ENV=sandbox` or `live`
- `PAYPAL_CLIENT_ID`
- `PAYPAL_CLIENT_SECRET`

GitHub child bounty repos:

- `GITHUB_TOKEN`
- `GITHUB_OWNER=mergeos-bounties`
- `GITHUB_OWNER_TYPE=org`

Crypto:

- Native coin: `CRYPTO_ASSET=native`, `CRYPTO_RPC_URL`, `CRYPTO_RECEIVER`, `CRYPTO_WEI_PER_USD_CENT`
- ERC20 stablecoin: `CRYPTO_ASSET=erc20`, `CRYPTO_RPC_URL`, `CRYPTO_RECEIVER`, `CRYPTO_TOKEN_CONTRACT`, `CRYPTO_TOKEN_DECIMALS`

Email:

- `SMTP_HOST`
- `SMTP_PORT`
- `SMTP_USERNAME`
- `SMTP_PASSWORD`
- `SMTP_FROM`

## API

- `POST /api/auth/register`
- `POST /api/auth/login`
- `GET /api/auth/me`
- `POST /api/auth/logout`
- `GET /api/config`
- `POST /api/payments/paypal/orders`
- `POST /api/projects`
- `GET /api/projects`
- `GET /api/tasks`
- `POST /api/tasks/{id}/accept`
- `GET /api/notifications`
- `GET /api/ledger`
- `GET /api/admin/summary`
- `GET /api/admin/users`
- `GET /api/admin/projects`
- `GET /api/admin/tasks`
- `GET /api/admin/notifications`
- `GET /api/admin/attachments`
- `GET /api/admin/ledger`
