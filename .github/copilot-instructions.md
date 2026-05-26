When reviewing pull requests in this repository, treat bounty readiness as part of the code review.

Do not approve or say LGTM when the PR has build failures, missing evidence, missing bounty context, broad unrelated rewrites, deleted tests, security-sensitive changes, or unclear migration/deployment impact.

For every bounty PR, check the PR title, description, linked issues, comments, changed files, and test output for:

- A link to the accepted claim comment or bounty issue.
- A declared bounty type: `bug bounty` or `feature bounty`.
- A declared bounty amount or bounty size.
- Evidence that matches the work:
  - UI changes need before/after screenshots, GIFs, videos, or uploaded image/video links.
  - Backend, API, workflow, infrastructure, and non-UI changes need logs, request/response examples, command output, or test output.
- Test commands and results, or a clear explanation for why tests were not run.
- Notes for migrations, new environment variables, deployment changes, data shape changes, permissions, secrets, or payment/auth/security behavior.
- A checked or explicit statement that the contributor starred this repository before claiming or starting bounty work.

GitHub Copilot code review cannot reliably verify the stargazers list or apply PR labels by itself. If the author only says they starred the repository, treat that as contributor attestation, not verified proof. If the star statement is missing or ambiguous, ask the author or maintainer to verify it before bounty review.

If evidence is missing, leave a direct review comment asking for the exact evidence needed. If the PR appears to be a backend or non-UI change, accept logs or test output as evidence rather than requiring screenshots.

When bounty type is detectable, mention the suggested label in the review summary:

- `bounty: bug` for bug fixes, regressions, crashes, broken behavior, security fixes, or failing tests.
- `bounty: feature` for new product behavior, integrations, APIs, screens, workflows, or enhancements.

When readiness is incomplete, mention the relevant suggested labels for maintainers to apply manually:

- `evidence: missing`
- `evidence: provided`
- `star: missing`
- `star: verified`

Keep review comments concise, specific, and grounded in files, lines, failing commands, or missing PR sections. If the change is unusually broad for the stated bounty, ask whether the unrelated removals or rewrites are intentional.
