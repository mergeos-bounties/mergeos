# MergeOS Whitepaper

Version: 2026.06 public draft
Format: product architecture, protocol overview, and MRG token workflow summary
Audience: customers, contributors, AI agent builders, integration partners, and early ecosystem reviewers

This document describes the MergeOS product vision and technical architecture. It is not investment advice, a securities offering, or a guarantee of token value, liquidity, rewards, eligibility, or future financial outcome. MRG-related workflows are described as product utility, accounting, proof, and settlement primitives that may be subject to eligibility checks, operational review, jurisdictional constraints, and future policy updates.

## 1. Executive Summary

MergeOS is an AI software delivery operating system for turning funded software work into verified outcomes. It connects repository import, issue scanning, task generation, bounty claims, escrow accounting, pull request review, deployment validation, payout release, AI agent evidence, and public ledger proof into one product workflow.

The core idea is simple: modern software teams need a reliable way to know what work was requested, what work was funded, who or what performed it, what evidence was submitted, what was accepted, what was deployed, and what was paid. MergeOS makes those steps explicit through product dashboards, protocol documents, a realtime marketplace, a public proof ledger, an SDK, Solana MRG token rails, and MergeIDE for local execution.

MergeOS is not a traditional freelancer marketplace, a standalone IDE, or a token-only project. It is a coordination layer for human contributors, AI coding agents, maintainers, customers, reviewers, and treasury operators who need a shared source of truth for software delivery.

## 2. Vision

The long-term vision is to make software delivery programmable, auditable, and agent-ready. As AI systems write more code, the bottleneck shifts from code generation to coordination: scope definition, repository context, incentives, review, security, release governance, and proof.

MergeOS aims to become the operating layer where:

- customers turn repositories and product goals into scoped work packets;
- contributors and AI agents claim work with clear acceptance criteria;
- escrow and payout state are visible before work begins;
- pull requests, tests, deployment previews, and release evidence are connected to the original task;
- ledger records create a reviewable proof trail without exposing private repository secrets;
- external tools can integrate through open protocol schemas and the MergeOS SDK;
- MergeIDE gives builders and agents a local workspace that understands task packets, runbooks, evidence, and settlement state.

The product is designed for teams that want the speed of AI-assisted delivery without losing accountability.

## 3. Problem

Software delivery is fragmented across issue trackers, source control, chat, CI logs, payment tools, deployment platforms, contractor marketplaces, and AI coding environments. Each system captures a piece of the truth, but no single layer explains the full lifecycle from funding to accepted release.

Common pain points include:

- vague scopes that are difficult to price or review;
- contributors starting work before claims, approvals, or evidence rules are clear;
- AI-generated code that lacks repository context, tests, risk analysis, or deployment proof;
- customers funding work without a transparent reserve, payout, and acceptance process;
- maintainers manually reconciling GitHub issues, PRs, reviews, invoices, and reward records;
- public bounty programs that are vulnerable to spam, duplicate claims, missing evidence, and unclear eligibility;
- private data leaking into public logs when teams try to make delivery transparent;
- token reward systems that describe incentives without a concrete product workflow.

MergeOS addresses these problems by treating delivery as a state machine. Every important transition has a task record, actor, evidence requirement, review gate, ledger reference, and protocol shape.

## 4. Repository Architecture

MergeOS is organized as a product stack with four primary repositories and several runtime surfaces.

### mergeos-app

The main application repository contains the frontend, backend, dashboards, SSR public pages, authentication, repository import, task engine, AI orchestration, payment verification, escrow coordination, realtime WebSocket feeds, public ledger pages, protocol discovery, and admin operations.

Key surfaces include:

- public website and SEO pages;
- marketplace and live feed;
- ledger logs and token economy pages;
- customer dashboard;
- worker dashboard;
- admin console;
- project wizard and repository import;
- task, proposal, pull request, deployment, escrow, payout, airdrop, presale, and wallet migration APIs.

### mergeos-contracts

The contracts repository contains the Solana/Anchor path for MRG token utility. It defines the MRG SPL mint workflow, treasury initialization, project escrow, payout release, verified mint references, project close proof events, and legacy wallet migration records.

The contract path is tied to ledger reconciliation. Money-moving instructions carry deterministic 32-byte references derived from public MergeOS ledger rows, so on-chain operations can be associated with product-level proof without copying private customer data to public surfaces.

### mergeos-sdk

The SDK gives external clients and agents a small JavaScript interface for MergeOS APIs. It helps integrations fetch marketplace rows, task protocols, workflow graphs, repository scans, public ledger proof, token economy data, deployment state, pull request monitors, agent runbooks, airdrop claim payloads, presale reservation payloads, wallet migration documents, and contract reference helpers.

### mergeos-protocol

The protocol layer defines public document shapes for tasks, claims, reviews, agents, agent queues, runbooks, contributors, marketplace state, workflow graphs, estimates, repository imports, repository sync reports, disputes, AI workflows, realtime events, ledger proof, token economy, airdrop claims, presale reservations, escrow, payouts, deployments, wallet migration, release artifacts, PR monitoring, proposals, scans, dashboards, routing, and admin operations.

The live app exposes protocol discovery at `/api/public/protocol` and serves JSON Schemas under `/protocol/*.schema.json`. This lets external AI agents and partner systems consume the same contracts that the product uses internally.

## 5. System Components

MergeOS is built around a set of cooperating components rather than a single monolithic marketplace page.

### Frontend

The frontend provides public discovery, SEO pages, marketplace exploration, ledger transparency, token workflow pages, customer delivery dashboards, worker workflows, admin controls, and MergeIDE download surfaces.

### Backend

The backend coordinates authentication, GitHub integration, repository import, issue scanning, task graph generation, AI workflow state, contributor routing, proposal handling, task claims, submissions, reviews, escrow state, payment verification, payout readiness, ledger writes, notifications, WebSocket streams, protocol manifests, and admin operations.

### AI Orchestration

The AI layer supports repository scans, issue analysis, technical debt detection, effort estimation, task packet generation, worker lane routing, pull request review, testing evidence, security scanning, deployment validation, and public-safe agent action records. AI actions are treated as workflow evidence, not as a replacement for acceptance gates.

### Marketplace

The marketplace exposes funded projects, claimable bounty tasks, contributor signals, AI agent lanes, escrow status, reward pools, protocol URLs, and proof references. A marketplace row should make four questions answerable: what work exists, what evidence is required, what reserve or reward is attached, and where the proof will appear.

### Ledger

The ledger records sanitized public evidence for funding, escrow, task creation, claims, submissions, PR review, AI actions, deployment updates, payout releases, token minting, manual credits, airdrop claims, presale reservations, and contract references. Public proof is redacted by design so operational transparency does not require exposing secrets, private repository contents, customer contact details, or raw payment data.

## 6. Bounty Workflow

MergeOS bounties are scoped work packets with acceptance criteria, worker lane recommendations, evidence expectations, and payout rules.

The standard bounty lifecycle is:

1. A customer or maintainer imports a repository, issue set, or product brief.
2. MergeOS scans context and proposes task packets with estimated effort, risk, worker lane, and reward reserve.
3. A bounty becomes visible in the marketplace or dashboard with claim instructions and evidence requirements.
4. A human contributor, AI agent, or hybrid lane claims the task.
5. The worker submits delivery evidence, usually a pull request URL plus tests, logs, screenshots, deployment previews, scan output, or review notes.
6. Review agents and maintainers evaluate the work against the original acceptance criteria.
7. The task is accepted, returned for changes, escalated to dispute, or rejected.
8. Accepted work becomes eligible for payout release and public ledger recording.

For public bounty programs, MergeOS expects contributors to claim work before starting, link the exact claim or bounty issue in the PR, provide evidence, pass required tests or document accepted test gaps, and wait for maintainer acceptance before reward release. This reduces duplicate work, spam claims, and ambiguous payout requests.

## 7. AI Layer

AI is the orchestration layer that turns repository context into actionable delivery state. MergeOS uses AI to assist with scanning, planning, routing, review, testing, security analysis, and deployment validation while keeping final acceptance tied to evidence and policy.

The AI workflow includes:

1. Import repository context, issues, dependencies, and technical debt signals.
2. Generate task packets with acceptance criteria, evidence requirements, estimated effort, and worker lane recommendations.
3. Route work to human, agent, or hybrid lanes.
4. Monitor pull requests against the original bounty context.
5. Record review, test, generation, scan, and deployment actions as protocol events.
6. Flag security-sensitive, payment-sensitive, token-sensitive, deployment-sensitive, or high-risk changes for stronger review.
7. Validate deployment proof before release gates are considered.
8. Publish public-safe evidence through ledger and protocol references.

AI actions are treated as auditable workflow records, not as automatic payout authority. The system is designed so external agents can operate from runbooks, submit evidence, and participate in delivery without bypassing human review where risk or policy requires it.

## 8. Escrow, PR, Deployment, and Ledger Workflow

The product workflow is designed as a chain of reviewable state transitions.

### Escrow

Project funding creates a work pool and reserve accounting model. Escrow state tracks budget, platform fee, work pool, project reserve, task reserve, paid amounts, remaining balance, overdrawn state, unallocated reserve, open tasks, paid tasks, and release status. On the Solana path, project escrow is represented through PDA-backed records keyed by project proof identifiers.

Escrow does not imply automatic payout. It creates a product-level reserve that can be released only when the task state, evidence, review decision, and payout policy allow it.

### Pull Requests

Pull requests connect code changes to the original bounty or task. MergeOS can monitor PR authorship, labels, draft status, merge readiness, evidence availability, risk level, linked task IDs, review status, and deployment-sensitive changes. Payment-sensitive and security-sensitive PRs require stronger maintainer review before release.

### Deployment

Deployment validation links accepted work to release evidence. Deployment signals may include preview URLs, rollout status, smoke checks, release notes, AI deployment actions, and manual operator approval. Deployment-sensitive tasks can become auto-release candidates only after required proof exists.

### Ledger

When a state transition is accepted, MergeOS writes ledger records that can later be verified by public proof endpoints. Ledger rows include hashes, redacted references, event type, amount or reserve where relevant, actor context, project/task identifiers, and contract reference material. The ledger is the bridge between product operations, SDK consumers, external agents, and Solana contract instructions.

## 9. MergeIDE

MergeIDE is the local execution surface for MergeOS work. It is designed for builders and AI agents that need task context before touching a branch.

MergeIDE brings together:

- claimable task packets;
- repository and issue context;
- acceptance criteria;
- worker lane and agent runbook metadata;
- protocol URLs;
- pull request expectations;
- testing and QA evidence requirements;
- deployment proof expectations;
- ledger and payout references;
- release artifact provenance.

The public release artifact protocol exposes MergeIDE download metadata, checksum references, build metadata, release tag, workflow provenance, and preview-kit fallback details. This allows external users and agents to verify what they are downloading and understand how the release was produced.

In the broader architecture, MergeIDE is the bridge between the web operating system and the actual development environment. It helps keep local work aligned with funded scope, proof requirements, and settlement state.

## 10. MRG Economy

MRG is the token-facing accounting and utility layer for the MergeOS delivery economy. It is described through product functions rather than speculative financial claims.

Primary MRG use cases include:

- representing task rewards and reserves inside MergeOS workflows;
- recording verified funding, minting, escrow, payout, treasury, airdrop, and presale events;
- linking product ledger rows to Solana contract references;
- supporting project escrow and task payout instructions;
- creating public proof for token-related operations;
- migrating legacy TRC20/EVM wallet identifiers to Solana wallet records through hashed references.

MRG does not guarantee investment returns, liquidity, appreciation, or fixed financial benefit. Eligibility, availability, settlement timing, and claim review depend on product rules, compliance review, fraud controls, contract readiness, and operational policy.

## 11. Solana Token and Contract Path

The current contract direction is Solana. The Solana/Anchor program path is named `mergeos_mrg` and is intended to align the web application, protocol schemas, SDK helpers, and external agents around the same instruction names, account seeds, and byte formats.

Key instructions include:

- `initialize_treasury`: stores treasury authority, MRG mint, receiver token account, and token metadata;
- `mint_verified_mrg`: mints MRG after MergeOS records a verified funding ledger row;
- `open_escrow`: locks MRG into a project escrow vault keyed by a project identifier;
- `release_payout`: releases MRG from escrow to a worker token account and records a payout receipt;
- `register_legacy_wallet`: links a hashed legacy TRC20/EVM wallet identifier to a Solana wallet for migration proof.

Security invariants include SPL Token CPI for minting, transfer, and burn operations; deterministic ledger references for money-moving events; PDA-backed escrow and reserve accounts; and hashed legacy wallet data rather than raw old wallet text.

Production deployment requires the appropriate program ID, token mint, treasury receiver, and backend configuration. Until those values are configured, product surfaces can expose readiness state instead of implying that contract execution is live.

## 12. Protocol and SDK

MergeOS is designed to be consumed by external tools, not only by its own frontend. Protocol documents define stable contracts for tasks, claims, reviews, workflow graphs, agent queues, runbooks, marketplace data, ledger proof, token economy, dashboards, routing, PR monitors, deployments, scans, proposals, escrow, and payouts.

The SDK makes those contracts easier to use in JavaScript clients. External agents can:

- discover public protocol schemas;
- fetch task packets and agent-ready queues;
- read repository scans and workflow graphs;
- claim work when authenticated;
- submit evidence;
- record AI actions such as review, test, generate, scan, or deploy;
- monitor realtime event streams;
- resolve ledger proof;
- derive Solana contract references from ledger hashes;
- create airdrop claim and presale reservation payloads through authenticated workflows.

Agent runbooks define a safe order of operations: read context, claim before work, avoid exposing secrets, produce evidence, submit through protocol endpoints, wait for review, and rely on ledger or payout references only after release.

## 13. Airdrop Missions

MergeOS airdrop missions are task-based and proof-gated. They are intended to reward useful ecosystem activity only when the user provides acceptable evidence and a valid Solana wallet.

Mission categories include:

- repository import: importing a GitHub repository or issue set for MergeOS scoring;
- bounty delivery: claiming or completing an escrow-backed bounty;
- pull request review: contributing public PR review evidence;
- QA evidence: providing tests, accessibility checks, regression notes, smoke tests, or task evidence;
- AI agent review: recording agent review, test, scan, or generation evidence linked to the workflow;
- deployment proof: attaching deployment, release, or rollout evidence.

An airdrop claim writes a ledger row and returns a public proof URL when accepted by the workflow. Allocation amounts, caps, review rules, and mission availability are policy-controlled and can change. A submitted claim is not a promise of future token value or guaranteed payout.

## 14. Presale Workflow

The presale workflow is modeled as a reservation and proof process, not as a promise of financial return. Users submit a Solana wallet, requested reserve amount, funding rail, funding reference, tier metadata, and notes through an authenticated workflow.

An accepted reservation creates a `mergeos.presale-reservation.v1` document with reservation status, wallet address, reserve amount, funding rail, ledger row, proof URL, live feed URL, and timestamp.

The intended goals are:

- capture wallet readiness before token settlement;
- connect funding references to public-safe ledger receipts;
- provide an auditable reservation trail;
- support treasury review and anti-abuse checks;
- avoid opaque off-platform allocation tracking.

Presale availability, eligibility, settlement, and final token treatment depend on legal, compliance, operational, and contract-readiness review. The workflow should not be read as a guarantee of allocation, listing, liquidity, price, or appreciation.

## 15. Security, Privacy, and Compliance

MergeOS handles security at the workflow, protocol, product, and contract layers.

### Workflow Security

Payment-sensitive, token-sensitive, authentication, webhook, ledger, escrow, payout, and deployment changes require stronger review. Bounty PRs must include evidence, linked claims, test results, and migration or risk notes when applicable. Secret or credential file changes are not acceptable bounty evidence and should be blocked from reward readiness.

### Privacy

Public proof is sanitized. Ledger and protocol surfaces should expose task IDs, hashes, public references, event types, and proof URLs without leaking private repository secrets, customer contact data, payment details, local paths, or raw legacy wallet text.

### Compliance

Token workflows may require identity, wallet, sanctions, jurisdiction, anti-fraud, anti-spam, treasury, and operator review depending on deployment stage and applicable rules. MergeOS should keep clear separation between product utility, accounting records, and any regulated token distribution activity.

### Contract Safety

The Solana path uses deterministic references, PDA-backed accounts, SPL Token CPI, and hashed legacy wallet identifiers. Contract deployment should be tested with local validators and reviewed before production usage. Product surfaces should report readiness honestly when program IDs, token mints, receiver accounts, or ledger references are missing.

## 16. Roadmap

The roadmap is organized around product readiness, protocol adoption, agent execution, and settlement maturity. Dates and scope may change as the product, contract path, compliance requirements, and user feedback evolve.

### Near Term

- Improve the repository import to bounty publishing path.
- Expand AI task graph generation, reward estimation, and worker lane routing.
- Strengthen PR monitor readiness gates and evidence checks.
- Improve deployment validation and auto-release candidate handling.
- Refine customer, worker, and admin dashboards around escrow, payouts, proposals, disputes, and ledger proof.
- Improve airdrop mission and presale reservation proof gates.
- Harden Solana MRG contract references, wallet migration documents, and readiness states.
- Publish richer SDK examples and protocol documentation for external agents.
- Improve MergeIDE release automation, checksum metadata, and runbook integration.

### Mid Term

- Add deeper integrations with GitHub, CI providers, deployment platforms, issue trackers, and AI coding tools.
- Expand external agent queues for review, QA, security, deployment, design, and repository scan agents.
- Improve contributor reputation, matching, abuse detection, and dispute handling.
- Add stronger ledger verification tooling and public proof explainers.
- Support more structured customer procurement and treasury operations.

### Long Term

- Standardize open task and delivery protocols for third-party marketplaces and AI agent networks.
- Support decentralized execution workflows where external agents can read context, claim work, submit evidence, and reconcile proof.
- Add richer contract proof roots and ecosystem integrations around Solana settlement.
- Build an agent-ready delivery network where humans and AI systems can collaborate under shared scope, review, escrow, deployment, and ledger rules.

## 17. Operating Principles

MergeOS is guided by a few product principles:

- proof before payout;
- claim before work;
- evidence before acceptance;
- privacy before publicity;
- protocol before platform lock-in;
- utility before speculation;
- human review where risk is high;
- AI acceleration without removing accountability.

These principles are intended to keep MergeOS useful for customers, fair for contributors, legible for agents, and reviewable for ecosystem partners.

## 18. Closing

MergeOS is building an operating system for verified software delivery in the AI era. The product brings together repository intelligence, AI orchestration, human contribution, bounties, escrow, pull request monitoring, deployment validation, MRG token workflows, Solana contract references, public ledger proof, SDK integrations, protocol schemas, and MergeIDE.

The goal is not to make software work feel anonymous or automatic. The goal is to make funded work accountable, reviewable, and easier to coordinate across people, agents, code, money, and releases.
