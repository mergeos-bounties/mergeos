# Contributing to MergeOS

Thanks for helping improve MergeOS. This project accepts regular contributions and bounty work. Bounty PRs must follow the extra claim and evidence rules in [BOUNTY-POLICY.md](BOUNTY-POLICY.md).

## Ways to Contribute

- Report a reproducible bug.
- Request a product improvement.
- Claim an approved bounty and submit a PR.
- Improve tests, documentation, security, or developer experience.
- Review open PRs with clear, respectful technical feedback.

## Before You Start Bounty Work

1. Star this repository.
2. Claim the bounty in the linked bounty issue or in [Claim Token issue #1](https://github.com/mergeos-bounties/mergeos/issues/1).
3. Wait for maintainer confirmation when the issue says approval is required.
4. Keep the implementation scoped to the accepted claim.
5. Open a PR that links the exact claim comment or bounty issue.

Unclaimed bounty work is not guaranteed to be reviewed for reward.

## Development Setup

Use Docker Compose for the full local stack:

```powershell
docker compose up --build
```

Useful local URLs:

- Frontend: `http://127.0.0.1:5173`
- Admin: `http://127.0.0.1:5174`
- Scan explorer: `http://127.0.0.1:5175`
- API health: `http://127.0.0.1:8080/api/health`

Manual service commands are documented in [README.md](README.md).

## Test Commands

Run the relevant checks before opening or updating a PR:

```powershell
cd backend
go test ./...
go vet ./...
go build -o dist/mergeos.exe ./cmd/mergeos
```

```powershell
cd frontend
npm ci
npm test --if-present
npm run build
```

```powershell
cd admin
npm ci
npm test --if-present
npm run build
```

```powershell
cd scan
npm ci
npm test --if-present
npm run build
```

PR checks also run secret scanning, Go vulnerability scanning, npm audit, tests, and builds.

## Pull Request Requirements

Every PR should include:

- A clear description of the change.
- Linked issue, claim comment, or bounty context.
- Before and after evidence for UI or behavior changes.
- Test commands and results.
- Notes for migrations, environment variables, deployment risk, or security impact.
- No secrets, private keys, tokens, customer data, or sensitive production details.

## Coding Guidelines

- Keep changes focused and easy to review.
- Prefer the existing project patterns over new abstractions.
- Keep backend business logic testable.
- Keep frontend behavior accessible, responsive, and consistent with the existing UI.
- Add or update tests when the change affects behavior.
- Avoid unrelated formatting, generated artifacts, or dependency churn.

## Review and Merge

Maintainers may ask for evidence, tests, scope reduction, conflict resolution, or security fixes before merge. Passing CI is required unless a maintainer explicitly accepts a documented gap.

Bounty payout eligibility is defined in [BOUNTY-POLICY.md](BOUNTY-POLICY.md).
