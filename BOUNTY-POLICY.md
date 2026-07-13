# MergeOS Bounty Policy

This policy defines how MergeOS bounty work is claimed, reviewed, labeled, and paid in MRG.

## Required Before Work

Contributors must complete these steps before starting bounty work:

1. **Follow** the org https://github.com/mergeos-bounties
2. **Star** https://github.com/mergeos-bounties/mergeos
3. **Star** https://github.com/mergeos-bounties/mergeos-contracts
4. Claim the bounty in the linked issue or in the [Claim Token issue #1](https://github.com/mergeos-bounties/mergeos/issues/1).
5. Wait for maintainer confirmation when the bounty requires approval.
6. Open a PR that links the exact claim comment or bounty issue.

Claims and PRs from accounts that have not **followed** https://github.com/mergeos-bounties and starred **both** `mergeos` + `mergeos-contracts` are not ready for bounty review (starring only a product repo is not enough).

## Bounty Types

- Bug bounty: fixes broken, incorrect, crashing, insecure, or regressed behavior.
- Feature bounty: adds new product behavior, integrations, workflows, APIs, screens, or meaningful enhancements.

PRs are labeled with one of:

- `bounty: bug`
- `bounty: feature`

## Reward Scale

| Bounty scope | Reward |
| --- | ---: |
| Bug fix or small feature | 25 MRG |
| Medium feature | 50 MRG |
| Large feature | 100 MRG |
| Extra-large feature or system-level work | 200 MRG |

Maintainers decide the final size before payout. A bounty can be moved up or down if the implemented scope is meaningfully different from the accepted scope.

## Review Readiness

Every bounty PR must include:

- Link to the claim comment or bounty issue.
- Bounty type and expected reward size.
- Before/after visual evidence for UI changes.
- Logs, request/response examples, or test output for backend/non-UI changes.
- Test commands and results.
- Notes for migrations, environment variables, risk, or deployment changes.

Maintainers use these labels while reviewing bounty PRs. The `Gemini PR review` webhook sends new and updated PRs to the MergeOS reviewer service. That service checks org follow + core stars (mergeos + mergeos-contracts), evidence, tests, bounty context, and code risk, then comments on the PR with the readiness summary:

- `evidence: missing`
- `evidence: provided`
- `star: missing`
- `star: verified`
- `bounty: bug`
- `bounty: feature`

GitHub Copilot code review can still flag code-quality and readiness issues when quota is available, but bounty readiness must not depend on Copilot being available.

## Payout Rule

MRG token rewards are only eligible after:

1. The contributor follows https://github.com/mergeos-bounties and has starred both `mergeos-bounties/mergeos` and `mergeos-bounties/mergeos-contracts`.
2. The PR links the accepted claim or bounty issue.
3. Evidence is accepted by maintainers.
4. Required tests pass or maintainers accept the stated test gap.
5. The PR is merged.
6. The reward is recorded in the bounty index or payout ledger.

## Public Tracking

Keep [README-INDEX.md](README-INDEX.md) updated when:

- A bounty is opened.
- A contributor claim is accepted.
- A PR is opened for a bounty.
- A PR is merged.
- MRG is released.
