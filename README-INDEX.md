# MergeOS README Index

Use this index as the fast public map for MergeOS bounty docs and current bounty intake.

## Core Docs

| Document | Purpose |
| --- | --- |
| [README.md](README.md) | Main project overview, local setup, APIs, and maintainer checklist. |
| [BOUNTY-POLICY.md](BOUNTY-POLICY.md) | Bounty rules, reward scale, evidence requirements, star requirement, and payout policy. |
| [CONTRIBUTING.md](CONTRIBUTING.md) | Contributor setup, PR requirements, test commands, and bounty workflow. |
| [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md) | Community behavior expectations and enforcement policy. |
| [SECURITY.md](SECURITY.md) | Private vulnerability reporting and supported security scope. |
| [SUPPORT.md](SUPPORT.md) | Support channels for bugs, feature requests, bounties, and security. |
| [protocol/README.md](protocol/README.md) | Open task, workflow graph, and event schemas for agents and integrations. |
| [LICENSE](LICENSE) | MIT license for this repository. |
| [Claim Token issue #1](https://github.com/mergeos-bounties/mergeos/issues/1) | Intake queue for new bug bounty claims. |

## Bounty Reward Scale

| Bounty scope | Reward |
| --- | ---: |
| Bug fix or small feature | 25 MRG |
| Medium feature | 50 MRG |
| Large feature | 100 MRG |
| Extra-large feature or system-level work | 200 MRG |

Admin ledger payouts use the same scale (`future-small` 25, `future-medium` 50, `bug-large` 100, `major-feature` 200). Issue titles that advertise higher marketing amounts are capped by this policy at merge time.

## Active Bounty Intake

| Bounty | Type | Reward | Claimed or submitted by | Current status |
| --- | --- | ---: | --- | --- |
| [#1 Claim token intake](https://github.com/mergeos-bounties/mergeos/issues/1) | Claim intake | — | Open | Comment new claims here before opening a PR |
| [#3 AI project evaluation](https://github.com/mergeos-bounties/mergeos/issues/3) | Feature bounty | 50 MRG | Prior claims; task already paid | Legacy open issue — prefer focused [#235](https://github.com/mergeos-bounties/mergeos/issues/235) |
| [#7 PayPal sandbox payment flow](https://github.com/mergeos-bounties/mergeos/issues/7) | Feature bounty | 100 MRG | Multiple closed PRs | Open — non-duplicative sandbox proof vs current master |
| [#8 USDT crypto payment gateway](https://github.com/mergeos-bounties/mergeos/issues/8) | Feature bounty | 100 MRG | Prior accepted payout | Open — full ledger-wired webhook only |
| [#64 QA verification of submitted PRs](https://github.com/mergeos-bounties/mergeos/issues/64) | QA bounty | 300 MRG per PR | Ongoing | Verify open/merged bounty PRs with evidence |
| [#231 Payment rails empty-state](https://github.com/mergeos-bounties/mergeos/issues/231) | Feature bounty | 100 MRG | Unclaimed | Open |
| [#232 Stripe PaymentIntent funding rail](https://github.com/mergeos-bounties/mergeos/issues/232) | Feature bounty | 100 MRG | Unclaimed | Open |
| [#233 SMTP notifications + offline fallback](https://github.com/mergeos-bounties/mergeos/issues/233) | Feature bounty | 50 MRG | Unclaimed | Open |
| [#234 OAuth state cookie hardening](https://github.com/mergeos-bounties/mergeos/issues/234) | Bug bounty | 50 MRG | Unclaimed | Open — no mock regressions |
| [#235 Gemini evaluate-price with fallback](https://github.com/mergeos-bounties/mergeos/issues/235) | Feature bounty | 100 MRG | Unclaimed | Open — backend-only preferred |
| [#236 Funding wizard blocked-state](https://github.com/mergeos-bounties/mergeos/issues/236) | Feature bounty | 50 MRG | Unclaimed | Open |
| [#237 Scan github worker alias aggregate](https://github.com/mergeos-bounties/mergeos/issues/237) | Feature bounty | 50 MRG | Unclaimed | Open / good first issue |
| [#238 SDK evidence_required helpers](https://github.com/mergeos-bounties/mergeos/issues/238) | Feature bounty | 25 MRG | Unclaimed | Open / good first issue |
| [#239 Password reset UX when SMTP offline](https://github.com/mergeos-bounties/mergeos/issues/239) | Bug bounty | 50 MRG | Unclaimed | Open |
| [#240 Admin credit+comment template UX](https://github.com/mergeos-bounties/mergeos/issues/240) | Feature bounty | 100 MRG | Unclaimed | Open |
| [#241 Bank transfer manual verify path](https://github.com/mergeos-bounties/mergeos/issues/241) | Feature bounty | 50 MRG | Unclaimed | Open |
| [#242 Public config LOCAL-PAID leak regression](https://github.com/mergeos-bounties/mergeos/issues/242) | Bug bounty | 25 MRG | Unclaimed | Open / good first issue |

## Awarded Bounties

| PR | Bounty / scope | Contributor | Reward | MRG credit URL | Status |
| --- | --- | --- | ---: | --- | --- |
| [#227](https://github.com/mergeos-bounties/mergeos/pull/227) | SDK public limit sanitization (rebase of #212) | [@yyswhsccc](https://github.com/yyswhsccc) | 25 MRG | [github:yyswhsccc](https://scan.mergeos.shop/address/github:yyswhsccc) | Merged + credited (ledger seq 55) |
| [#225](https://github.com/mergeos-bounties/mergeos/pull/225) | [#17 Project view after login](https://github.com/mergeos-bounties/mergeos/issues/17) | [@sureshchouksey8](https://github.com/sureshchouksey8) | 50 MRG | [github:sureshchouksey8](https://scan.mergeos.shop/address/github:sureshchouksey8) | Merged + credited (ledger seq 46) |
| [#218](https://github.com/mergeos-bounties/mergeos/pull/218) | Backend Go patch version bump | [@yyswhsccc](https://github.com/yyswhsccc) | 25 MRG | [github:yyswhsccc](https://scan.mergeos.shop/address/github:yyswhsccc) | Merged + credited (ledger seq 47) |
| [#217](https://github.com/mergeos-bounties/mergeos/pull/217) | SDK agent action finite numbers | [@yyswhsccc](https://github.com/yyswhsccc) | 25 MRG | [github:yyswhsccc](https://scan.mergeos.shop/address/github:yyswhsccc) | Merged + credited (ledger seq 54) |
| [#216](https://github.com/mergeos-bounties/mergeos/pull/216) | Admin sensitive path case-fold | [@yyswhsccc](https://github.com/yyswhsccc) | 25 MRG | [github:yyswhsccc](https://scan.mergeos.shop/address/github:yyswhsccc) | Merged + credited (ledger seq 53) |
| [#215](https://github.com/mergeos-bounties/mergeos/pull/215) | Manual credit GitHub worker aliases | [@yyswhsccc](https://github.com/yyswhsccc) | 25 MRG | [github:yyswhsccc](https://scan.mergeos.shop/address/github:yyswhsccc) | Merged + credited (ledger seq 52) |
| [#214](https://github.com/mergeos-bounties/mergeos/pull/214) | Attachment download filename sanitize | [@yyswhsccc](https://github.com/yyswhsccc) | 25 MRG | [github:yyswhsccc](https://scan.mergeos.shop/address/github:yyswhsccc) | Merged + credited (ledger seq 51) |
| [#213](https://github.com/mergeos-bounties/mergeos/pull/213) | Scan GitHub worker account normalize | [@yyswhsccc](https://github.com/yyswhsccc) | 25 MRG | [github:yyswhsccc](https://scan.mergeos.shop/address/github:yyswhsccc) | Merged + credited (ledger seq 50) |
| [#211](https://github.com/mergeos-bounties/mergeos/pull/211) | OAuth loopback redirect scheme | [@yyswhsccc](https://github.com/yyswhsccc) | 25 MRG | [github:yyswhsccc](https://scan.mergeos.shop/address/github:yyswhsccc) | Merged + credited (ledger seq 49) |
| [#210](https://github.com/mergeos-bounties/mergeos/pull/210) | Project escrow exact ledger scoping | [@yyswhsccc](https://github.com/yyswhsccc) | 25 MRG | [github:yyswhsccc](https://scan.mergeos.shop/address/github:yyswhsccc) | Merged + credited (ledger seq 48) |
| [#220](https://github.com/mergeos-bounties/mergeos/pull/220) | [#13 Auth modal responsive](https://github.com/mergeos-bounties/mergeos/issues/13) | [@sureshchouksey8](https://github.com/sureshchouksey8) | — | — | Merged via admin earlier |
| [#150](https://github.com/mergeos-bounties/mergeos/pull/150) | Public test-mode publish settings | [@lb1192176991-lab](https://github.com/lb1192176991-lab) | 50 MRG | [ledger seq 40](https://scan.mergeos.shop/) | Merged + credited |
| [#153](https://github.com/mergeos-bounties/mergeos/pull/153) | Notification click-to-read badges | [@zeroknowledge0x](https://github.com/zeroknowledge0x) | 50 MRG | [github:zeroknowledge0x](https://scan.mergeos.shop/address/github:zeroknowledge0x) | Merged + credited |
| [#152](https://github.com/mergeos-bounties/mergeos/pull/152) | Logout state reset order | [@davidsineri](https://github.com/davidsineri) | 50 MRG | [github:davidsineri](https://scan.mergeos.shop/address/github:davidsineri) | Merged + credited |
| [#86](https://github.com/mergeos-bounties/mergeos/pull/86) | [#18 Dashboard payment history](https://github.com/mergeos-bounties/mergeos/issues/18) | [@ryzhkevichpavel-del](https://github.com/ryzhkevichpavel-del) | 25 MRG | [github:ryzhkevichpavel-del](https://scan.mergeos.shop/address/github:ryzhkevichpavel-del) | Approved for merge |
| [#37](https://github.com/mergeos-bounties/mergeos/pull/37) | [#16 Dashboard layout after login](https://github.com/mergeos-bounties/mergeos/issues/16) | [@lb1192176991-lab](https://github.com/lb1192176991-lab) | 25 MRG | [0x1a2c281a70f475c747944b43d21923c3167bf7e1](https://scan.mergeos.shop/address/0x1a2c281a70f475c747944b43d21923c3167bf7e1) | Approved for merge |
| [#81](https://github.com/mergeos-bounties/mergeos/pull/81) | [#17 Project view after login from dashboard](https://github.com/mergeos-bounties/mergeos/issues/17) | [@lb1192176991-lab](https://github.com/lb1192176991-lab) | 25 MRG | [0x1a2c281a70f475c747944b43d21923c3167bf7e1](https://scan.mergeos.shop/address/0x1a2c281a70f475c747944b43d21923c3167bf7e1) | Approved for merge |
