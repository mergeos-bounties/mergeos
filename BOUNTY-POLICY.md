# MergeOS Bounty Policy

This policy defines how MergeOS bounty work is claimed, reviewed, labeled, and paid in MRG.

## Required Before Work

Contributors must complete these steps before starting bounty work:

1. Star this repository.
2. Claim the bounty in the linked issue or in the [Claim Token issue #1](https://github.com/mergeos-bounties/mergeos/issues/1).
3. Wait for maintainer confirmation when the bounty requires approval.
4. Open a PR that links the exact claim comment or bounty issue.

Claims and PRs from accounts that have not starred this repository are not ready for bounty review.

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

Maintainers use these labels while reviewing bounty PRs. GitHub Copilot code review should flag missing readiness items and suggest the relevant labels in its review summary, but maintainers apply the labels manually:

- `evidence: missing`
- `evidence: provided`
- `star: missing`
- `star: verified`
- `bounty: bug`
- `bounty: feature`

## Payout Rule

MRG token rewards are only eligible after:

1. The contributor has starred the repository.
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
