# MergeOS

MergeOS is an AI-assisted software maintenance and bounty operating system. A customer funds a project, receives internal project tokens, and MergeOS turns the work into claimable tasks that can be completed by human contributors, AI agents, or hybrid teams.

This repository is the current MergeOS MVP: Go backend, Vue SSR frontend, project funding flow, bounty workspace generation, GitHub issue import, evidence attachments, notifications, admin review, and proof ledger.

`scan/` is the public MergeOS Scan explorer for `scan.mergeos.shop`. It reads the public ledger API and presents MRG token mints, escrow movements, task reserves, payouts, addresses, transaction hashes, and hash-chain proof in a BscScan-style interface.

## README Index

- [README-INDEX.md](README-INDEX.md): docs map and current bounty intake index.
- [BOUNTY-POLICY.md](BOUNTY-POLICY.md): bounty rules, reward scale, evidence requirements, star requirement, and payout policy.
- [Claim Token issue #1](https://github.com/mergeos-bounties/mergeos/issues/1): public intake queue for new bug bounty claims.

## New Workflow

1. The customer registers or logs in.
2. The customer creates a new project or imports an existing repository.
3. The customer funds escrow through PayPal, crypto, or the local development verifier.
4. MergeOS records the funded project, mints internal `MRG` token credit, and creates a bounty workspace.
5. The system splits the project into tasks with reward pools, worker type, agent suggestion, and acceptance criteria.
6. Contributors who want token rewards must star this repository, then comment in the Claim Token issue before starting work.
7. The contributor opens a PR after the fix, including screenshots/log evidence and a clear description.
8. Maintainers review the PR, verify the evidence, merge valid work, and release the token reward.

## Claim Token Bounty Program

All new bounty bugs must start from this issue:

[Claim MRG Tokens for Bug Bounty Reports - Comment New Bugs Here Before Opening a PR](https://github.com/mergeos-bounties/mergeos/issues/1)

Do not open a separate issue for a new bounty bug unless a maintainer asks for it. Comment in the Claim Token issue first so the project has one clear intake queue for token claims.

### Claim Rules

- One bug equals one claim comment.
- Contributors must star this repository before claiming or starting bounty work.
- A claim must include impact, steps to reproduce, expected result, actual result, and evidence.
- A maintainer must confirm the claim before the contributor starts bounty work.
- Duplicates, vague reports, missing reproduction steps, or reports without evidence are not eligible yet.
- A PR must link back to the exact claim comment.
- Token payout is only eligible after the PR is reviewed, evidence is accepted, tests pass, and the PR is merged.
- Do not paste secrets, private keys, customer data, or exploitable production details into public comments. Describe the impact and ask a maintainer to move sensitive validation private.

### Bounty Reward Scale

| Bounty scope | Reward |
| --- | ---: |
| Bug fix or small feature | 25 MRG |
| Medium feature | 50 MRG |
| Large feature | 100 MRG |
| Extra-large feature or system-level work | 200 MRG |

See [BOUNTY-POLICY.md](BOUNTY-POLICY.md) for full bounty rules and [README-INDEX.md](README-INDEX.md) for the current bounty intake list.

### Claim Comment Template

```markdown
### Bug title
Short name for the bug.

### Impact
Who is affected, where it happens, and severity.

### Steps to reproduce
1. ...
2. ...
3. ...

### Expected result
What should happen.

### Actual result
What happens now.

### Evidence
Attach screenshot, GIF, video, log, request/response, or failing test output.

### Proposed fix
Optional fix direction.

### Claim info
GitHub handle:
Wallet/token receiver:
```

## Pull Request Requirements

Every bounty PR must include:

- Link to the Claim Token issue comment.
- Bounty type: bug bounty or feature bounty.
- Bounty size: small, medium, large, or extra-large.
- The PR author must have starred this repository.
- Summary of the bug and the fix.
- Evidence before and after the fix. Screenshots or GIFs are preferred for UI bugs. Logs, request/response examples, or test output are acceptable for backend bugs.
- Test commands that were run and their result.
- Any risk, migration, environment variable, or deployment note.

PRs from authors who have not starred the repository, or PRs without evidence, are not ready for bounty review.

## Product Scope

MergeOS currently supports:

- Customer auth with email/password bearer sessions.
- Two environment modes: `local` and `production`.
- Project creation with budget, payment method, attachments, and escrow status.
- Local payment verification through `LOCAL-PAID`.
- PayPal Orders v2 adapter.
- EVM native or ERC-20 receipt verification.
- GitHub open issue import with heuristic scoring.
- Local git bounty workspaces or GitHub private bounty repos when `GITHUB_TOKEN` is configured.
- Task reward allocation, worker kind, suggested agent type, and acceptance criteria.
- Proof ledger entries with hash chaining.
- SMTP notifications when configured, persisted notification records when SMTP is not configured.
- Admin APIs for users, projects, tasks, attachments, notifications, ledger, and SSL review.

Roadmap items include full AI codebase scanning, task dependency DAGs, automated PR verification, reputation scoring, fraud detection, and automatic real payout execution.

## Stack

- Backend: Go `net/http`
- Storage: PostgreSQL when `DATABASE_URL` is set, with legacy JSON state fallback for local development
- Frontend: Vue 3 + Vite SSR
- Admin: Vue 3 + Vite static admin console
- Scan: Vue 3 + Vite static explorer served from `scan.mergeos.shop`
- Token symbol: `MRG` by default through `TOKEN_SYMBOL`
- Bounty repos: local git under `BOUNTY_ROOT`, or GitHub private repos with `GITHUB_TOKEN`
- Payments: local verifier, PayPal, EVM native/ERC-20 verifier

## Run Local

Backend:

```powershell
cd backend
Copy-Item .env.local.example .env.local
go run ./cmd/mergeos
```

Frontend:

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

Open `http://127.0.0.1:5173`.
Open admin at `http://127.0.0.1:5174`.
Open scan at `http://127.0.0.1:5175`.

Local payment reference:

```text
LOCAL-PAID
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
- `GITHUB_TOKEN`, `GITHUB_OWNER`, `GITHUB_OWNER_TYPE`: GitHub bounty repo creation
- `BOUNTY_ROOT`: local child bounty repo root
- `UPLOAD_ROOT`: attachment storage root
- `SMTP_HOST`, `SMTP_PORT`, `SMTP_USERNAME`, `SMTP_PASSWORD`, `SMTP_FROM`: email notifications

## API Surface

Public:

- `GET /api/health`
- `GET /api/config`
- `GET /api/public/ledger`
- `GET /api/public/marketplace`
- `POST /api/public/repo/issues`

Auth:

- `POST /api/auth/register`
- `POST /api/auth/login`
- `GET /api/auth/me`
- `POST /api/auth/logout`

Customer:

- `POST /api/payments/paypal/orders`
- `POST /api/uploads`
- `GET /api/uploads/{id}/download`
- `POST /api/projects`
- `GET /api/projects`
- `GET /api/tasks`
- `POST /api/tasks/{id}/accept`
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
- `GET /api/admin/ssl`
- `POST /api/admin/ssl/review`

## Maintainer Bounty Checklist

For every token claim:

1. Confirm the bug is reproducible and not a duplicate.
2. Assign priority, bounty amount, and claimant in the Claim Token issue.
3. Require the PR to link the exact claim comment.
4. Verify screenshots, GIFs, logs, or test output.
5. Run or review the relevant tests.
6. Merge only after the fix and evidence match the accepted claim.
7. Record token release in the ledger or payout process.
