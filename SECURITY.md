# Security Policy

## Supported Versions

MergeOS is currently an MVP. Security fixes are accepted for the `master` branch.

| Version                | Supported |
| ---------------------- | --------- |
| `master`               | Yes       |
| Older commits or forks | No        |

## Reporting a Vulnerability

Do not open a public issue for a suspected vulnerability.

Use GitHub private vulnerability reporting when available:

https://github.com/mergeos-bounties/mergeos/security/advisories/new

If private reporting is unavailable, contact the repository maintainers privately through GitHub and include only the minimum detail needed to confirm the issue.

## What to Include

- Affected component: backend, frontend, admin, scan, deployment, or dependency.
- Clear reproduction steps.
- Expected impact and affected data or permissions.
- Relevant logs, request examples, or screenshots with secrets redacted.
- Suggested fix, if known.

## Scope

In scope:

- Authentication and session issues.
- Authorization bypasses.
- Secret exposure.
- Unsafe file upload or download behavior.
- Payment, wallet, ledger, bounty payout, or admin workflow integrity issues.
- Stored or reflected cross-site scripting.
- Dependency vulnerabilities with a reachable exploit path.

Out of scope:

- Denial-of-service reports without a practical exploit path.
- Social engineering.
- Spam, phishing, or attacks against third-party services.
- Reports that require access to another user's account or private data without consent.
- Vulnerabilities only present in local development configuration.

## Disclosure

Please give maintainers a reasonable chance to investigate and fix confirmed vulnerabilities before public disclosure.

We aim to acknowledge credible reports within 7 days and provide a status update within 14 days when possible.
