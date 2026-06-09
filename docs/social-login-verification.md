# MergeOS Social Login Verification

Scope:
- Bounty #2: Social login
- PR #222: Harden OAuth state cookies for MergeOS social login

What changed:
- Added a shared OAuth state cookie helper.
- Set SameSite=Lax on GitHub and Google OAuth state cookies.
- Set Secure automatically for HTTPS and forwarded-HTTPS requests.
- Added regression tests for both providers.

Why this is relevant to the bounty:
- The social login flow is still the active public bounty lane.
- The change reduces CSRF risk in the login flow without altering the user-facing auth contract.

Verification performed locally:
- Reviewed the OAuth login and callback handlers for GitHub and Google.
- Added unit tests covering:
  - GitHub state cookie hardening
  - Google state cookie hardening
  - secure-cookie behavior behind HTTPS / X-Forwarded-Proto

Notes:
- `go test` could not be executed in this shell because `go` is not installed here.
- The PR is already published and linked to the bounty issue.
