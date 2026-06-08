import test from 'node:test';
import assert from 'node:assert/strict';
import fs from 'node:fs/promises';
import http from 'node:http';
import net from 'node:net';
import os from 'node:os';
import path from 'node:path';
import {
  createMergeOSServer,
  createRuntimeConfig,
  loadEnvFiles,
  normalizeMode,
  parseEnvText,
  resolveMode,
  shouldRunProduction,
} from './server.js';

function compareSemver(left = '', right = '') {
  const leftParts = String(left).split('.').map((part) => Number.parseInt(part, 10) || 0);
  const rightParts = String(right).split('.').map((part) => Number.parseInt(part, 10) || 0);
  const length = Math.max(leftParts.length, rightParts.length);
  for (let index = 0; index < length; index += 1) {
    if ((leftParts[index] || 0) > (rightParts[index] || 0)) return 1;
    if ((leftParts[index] || 0) < (rightParts[index] || 0)) return -1;
  }
  return 0;
}

test('normalizes run modes', () => {
  assert.equal(normalizeMode('prod'), 'production');
  assert.equal(normalizeMode('production'), 'production');
  assert.equal(normalizeMode('dev'), 'local');
  assert.equal(normalizeMode('local'), 'local');
  assert.equal(normalizeMode(''), 'local');
});

test('resolves mode from CLI before environment', () => {
  const argv = ['node', 'server.js', '--mode', 'production'];
  const env = { MERGEOS_ENV: 'local' };
  assert.equal(resolveMode(argv, env), 'production');
  assert.equal(shouldRunProduction(argv, env), true);
});

test('parses env file lines', () => {
  assert.deepEqual(parseEnvText(`
    # comment
    API_TARGET="http://127.0.0.1:8080"
    FRONTEND_PORT=5173
  `), {
    API_TARGET: 'http://127.0.0.1:8080',
    FRONTEND_PORT: '5173',
  });
});

test('loads mode env before fallback without overriding real env', async () => {
  const cwd = await fs.mkdtemp(path.join(os.tmpdir(), 'mergeos-frontend-env-'));
  await fs.writeFile(path.join(cwd, '.env.local'), 'FRONTEND_PORT=5173\nAPI_TARGET=http://local-api\n');
  await fs.writeFile(path.join(cwd, '.env'), 'FRONTEND_PORT=9999\nSSR_PORT=6000\n');
  const env = { FRONTEND_PORT: '7000' };

  await loadEnvFiles('local', { cwd, env });

  assert.equal(env.FRONTEND_PORT, '7000');
  assert.equal(env.API_TARGET, 'http://local-api');
  assert.equal(env.SSR_PORT, '6000');
});

test('public system vision preserves the product thesis', async () => {
  const appSource = await fs.readFile(new URL('./src/App.vue', import.meta.url), 'utf-8');
  const seoSource = await fs.readFile(new URL('./src/seo.js', import.meta.url), 'utf-8');
  const whitepaperSource = await fs.readFile(new URL('./public/whitepaper/mergeos-whitepaper.md', import.meta.url), 'utf-8');
  const architectureManifest = JSON.parse(await fs.readFile(new URL('./public/system/mergeos-architecture.v1.json', import.meta.url), 'utf-8'));

  assert.match(appSource, /Product vision[\s\S]*A workflow layer combining GitHub, Stripe, Linear, Upwork, Vercel, and AI agents\./);
  assert.match(appSource, /Core system[\s\S]*GitHub, Stripe, Linear, Upwork, Vercel, and AI agents in one delivery workflow/);
  assert.match(appSource, /Architecture JSON[\s\S]*\/system\/mergeos-architecture\.v1\.json/);
  assert.match(seoSource, /sameAs: \[absoluteUrl\('\/system\/mergeos-architecture\.v1\.json'/);
  assert.match(seoSource, /encodingFormat: 'application\/json'/);
  assert.match(appSource, /Delivery workflow[\s\S]*Repository import, AI scan, task graph, reward estimate, contributor routing, PR review, deployment validation, payout release\./);
  assert.match(appSource, /Repository context, task scope, escrow, PR evidence, deployment gates, and payout release should live in the same operating graph\./);
  assert.match(appSource, /The core loop is import, scan, estimate, fund, route, review, validate, release, and prove\./);
  assert.match(appSource, /Auto-release only opens after PR readiness and required deployment validation\./);
  assert.match(appSource, /MergeOS connects repositories, issues, technical debt, AI agents, contributors, escrow, PR review, deployment validation, MRG token accounting, and public ledger proof in one realtime workflow\./);
  assert.match(whitepaperSource, /MergeOS is not a traditional freelancer marketplace/);
  assert.match(whitepaperSource, /a standalone IDE, or a token-only project/);
  assert.match(whitepaperSource, /GitHub-style repository context, Stripe-style payment verification, Linear-style task operations, Upwork-style contributor markets, Vercel-style deployment proof, and AI agent execution/);
  assert.match(whitepaperSource, /coordination layer for human contributors, AI coding agents, maintainers, customers, reviewers, and treasury operators/);
  assert.match(whitepaperSource, /shared source of truth for software delivery/);
  assert.equal(architectureManifest.protocol_version, 'mergeos.architecture.v1');
  assert.equal(architectureManifest.positioning, 'AI software delivery operating system');
  assert.deepEqual(Object.keys(architectureManifest.system_inputs), ['repositories', 'issues', 'technical_debt', 'bug_fixes', 'pull_requests', 'deployments']);
  assert.equal(architectureManifest.system_inputs.repositories.api, '/api/repos/import');
  assert.equal(architectureManifest.system_inputs.issues.output_protocol, 'mergeos.repo-sync.v1');
  assert.equal(architectureManifest.system_inputs.technical_debt.output_protocol, 'mergeos.scan.v1');
  assert.equal(architectureManifest.system_inputs.bug_fixes.api, '/api/tasks/{id}/submit');
  assert.equal(architectureManifest.system_inputs.pull_requests.api, '/api/public/projects/{id}/pull-requests');
  assert.equal(architectureManifest.system_inputs.deployments.output_protocol, 'mergeos.deployment.v1');
  assert.deepEqual(Object.keys(architectureManifest.product_vision.product_composition), ['github', 'stripe', 'linear', 'upwork', 'vercel', 'ai_agents']);
  assert.equal(architectureManifest.product_vision.product_composition.github.primary_route, '/repo-import');
  assert.equal(architectureManifest.product_vision.product_composition.stripe.primary_route, '/api/payments/order-intents');
  assert.equal(architectureManifest.product_vision.product_composition.linear.primary_route, '/api/projects/{id}/task-graph');
  assert.equal(architectureManifest.product_vision.product_composition.upwork.primary_route, '/marketplace');
  assert.equal(architectureManifest.product_vision.product_composition.vercel.primary_route, '/api/public/protocol/deployment');
  assert.equal(architectureManifest.product_vision.product_composition.ai_agents.primary_route, '/agents');
  assert.deepEqual(Object.keys(architectureManifest.product_vision.core_value_workflows), [
    'import_repository',
    'ai_issue_scan',
    'automatic_task_split',
    'create_bounty',
    'lock_escrow',
    'watch_live_prs',
    'track_deployments',
    'auto_release_payment',
  ]);
  assert.equal(architectureManifest.product_vision.core_value_workflows.create_bounty.output_protocol, 'mergeos.repo-task-funding.v1');
  assert.equal(architectureManifest.product_vision.core_value_workflows.auto_release_payment.output_protocol, 'mergeos.payout-release.v1');
  assert.deepEqual(Object.keys(architectureManifest.product_vision.workflow_routes), architectureManifest.product_vision.core_loop);
  assert.deepEqual(architectureManifest.product_vision.workflow_routes, {
    import_repository: { page: '/project/new', api: '/api/repos/import', proof_surface: '/live-feed' },
    scan_issues_with_ai: { page: '/repo-import', api: '/api/public/projects/{id}/repo-scan', proof_surface: '/live-feed' },
    generate_task_graph: { page: '/dashboard', api: '/api/projects/{id}/task-graph', proof_surface: '/marketplace#marketplace-bounties' },
    estimate_rewards: { page: '/project/new/budget', api: '/api/projects/{id}/estimate', proof_surface: '/ledger' },
    fund_escrow: { page: '/project/new/review', api: '/api/projects/{id}/escrow', proof_surface: '/contracts' },
    route_contributors_or_agents: { page: '/marketplace', api: '/api/public/protocol/routing', proof_surface: '/marketplace' },
    review_pull_requests: { page: '/dashboard', api: '/api/public/projects/{id}/pull-requests', proof_surface: '/ledger' },
    validate_deployment: { page: '/live-feed', api: '/api/public/protocol/deployment', proof_surface: '/ledger' },
    release_payment: { page: '/dashboard?section=admin', api: '/api/public/protocol/payout-settlement', proof_surface: '/ledger' },
    publish_ledger_proof: { page: '/ledger', api: '/api/public/ledger', proof_surface: '/ledger' },
  });
  assert.deepEqual(architectureManifest.repository_architecture.map((repo) => repo.name), ['mergeos-app', 'mergeos-contracts', 'mergeos-sdk', 'mergeos-protocol']);
  assert.deepEqual(architectureManifest.repository_architecture.map((repo) => repo.artifact_urls), [
    { primary: '/', protocol: '/system/mergeos-architecture.v1.json', public_reference: '/protocol' },
    { primary: '/contracts', protocol: '/contracts/solana/mergeos_mrg.v1.idl.json', public_reference: '/contracts/solana/mergeos_mrg.proof-manifest.v1.json' },
    { primary: '/sdk', protocol: '/protocol', public_reference: '/api/public/protocol/manifest' },
    { primary: '/protocol', protocol: '/protocol/architecture.v1.schema.json', public_reference: '/system/mergeos-architecture.v1.json' },
  ]);
  assert.deepEqual(architectureManifest.users.map((row) => row.type), ['customers', 'contributors', 'ai_agents', 'admins']);
  assert.deepEqual(architectureManifest.users.map((row) => row.role_routes), [
    { page: '/dashboard', api: '/api/projects/{id}/dashboard', capabilities: ['project overview', 'live PRs', 'escrow', 'payments', 'tasks', 'AI logs'] },
    { page: '/dashboard?section=worker', api: '/api/workers/me', capabilities: ['claimed tasks', 'rewards', 'reputation', 'proposals'] },
    { page: '/agents', api: '/api/public/agents/queue', capabilities: ['scan repositories', 'generate task packets', 'review PRs', 'test builds', 'validate deployments'] },
    { page: '/dashboard?section=admin', api: '/api/admin/ops-queue', capabilities: ['treasury', 'users', 'disputes', 'payouts', 'moderation'] },
  ]);
  assert.ok(architectureManifest.frontend_system.stack.includes('Vue 3'));
  assert.ok(architectureManifest.frontend_system.stack.includes('Vite SSR'));
  assert.ok(architectureManifest.frontend_system.public_pages.includes('Marketplace'));
  assert.deepEqual(Object.keys(architectureManifest.frontend_system.public_page_routes), [
    'homepage',
    'marketplace',
    'live_feed',
    'ledger_logs',
    'protocol',
    'mergeide',
    'airdrop',
    'presale',
    'whitepaper',
  ]);
  assert.equal(architectureManifest.frontend_system.public_page_routes.homepage.api, '/system/mergeos-architecture.v1.json');
  assert.equal(architectureManifest.frontend_system.public_page_routes.marketplace.api, '/api/public/marketplace');
  assert.equal(architectureManifest.frontend_system.public_page_routes.live_feed.api, '/api/public/live-feed');
  assert.equal(architectureManifest.frontend_system.public_page_routes.ledger_logs.proof_surface, '/api/public/ledger/proof');
  assert.equal(architectureManifest.frontend_system.public_page_routes.protocol.api, '/api/public/protocol/manifest');
  assert.equal(architectureManifest.frontend_system.public_page_routes.mergeide.api, '/downloads/mergeide/latest.json');
  assert.equal(architectureManifest.frontend_system.public_page_routes.airdrop.api, '/api/public/airdrop/missions');
  assert.equal(architectureManifest.frontend_system.public_page_routes.presale.api, '/api/presale/reservations');
  assert.equal(architectureManifest.frontend_system.public_page_routes.whitepaper.api, '/whitepaper/mergeos-whitepaper.md');
  assert.ok(architectureManifest.frontend_system.authenticated_dashboards.includes('Customer Dashboard'));
  assert.deepEqual(Object.keys(architectureManifest.frontend_system.authenticated_dashboard_urls), ['customer_dashboard', 'worker_dashboard', 'admin_console']);
  assert.equal(architectureManifest.frontend_system.authenticated_dashboard_urls.customer_dashboard.page, '/dashboard');
  assert.equal(architectureManifest.frontend_system.authenticated_dashboard_urls.customer_dashboard.api, '/api/projects/{id}/dashboard');
  assert.deepEqual(architectureManifest.frontend_system.authenticated_dashboard_urls.customer_dashboard.capabilities, ['project overview', 'live PRs', 'escrow', 'payments', 'tasks', 'AI logs']);
  assert.equal(architectureManifest.frontend_system.authenticated_dashboard_urls.worker_dashboard.page, '/dashboard?section=worker');
  assert.equal(architectureManifest.frontend_system.authenticated_dashboard_urls.worker_dashboard.api, '/api/workers/me');
  assert.deepEqual(architectureManifest.frontend_system.authenticated_dashboard_urls.worker_dashboard.capabilities, ['claimed tasks', 'rewards', 'reputation', 'proposals']);
  assert.equal(architectureManifest.frontend_system.authenticated_dashboard_urls.admin_console.page, '/dashboard?section=admin');
  assert.equal(architectureManifest.frontend_system.authenticated_dashboard_urls.admin_console.api, '/api/admin/ops-queue');
  assert.deepEqual(architectureManifest.frontend_system.authenticated_dashboard_urls.admin_console.capabilities, ['treasury', 'users', 'disputes', 'payouts', 'moderation']);
  assert.equal(architectureManifest.dashboard_system.role, 'Authenticated command surfaces for customers, workers, and admins after login.');
  assert.deepEqual(Object.keys(architectureManifest.dashboard_system.surfaces), ['customer_dashboard', 'worker_dashboard', 'admin_console']);
  assert.deepEqual(architectureManifest.dashboard_system.surfaces.customer_dashboard.modules, ['project overview', 'live PRs', 'escrow', 'payments', 'tasks', 'AI logs']);
  assert.ok(architectureManifest.dashboard_system.surfaces.customer_dashboard.realtime_events.includes('ai_review'));
  assert.deepEqual(architectureManifest.dashboard_system.surfaces.worker_dashboard.modules, ['claimed tasks', 'rewards', 'reputation', 'proposals']);
  assert.ok(architectureManifest.dashboard_system.surfaces.worker_dashboard.realtime_events.includes('ledger_task_payment'));
  assert.deepEqual(architectureManifest.dashboard_system.surfaces.admin_console.modules, ['treasury', 'users', 'disputes', 'payouts', 'moderation']);
  assert.ok(architectureManifest.dashboard_system.surfaces.admin_console.realtime_events.includes('payout_ready'));
  assert.equal(architectureManifest.dashboard_system.surfaces.admin_console.primary_api, '/api/admin/ops-queue');
  assert.ok(architectureManifest.backend_system.proposed_stack.includes('Go'));
  assert.ok(architectureManifest.backend_system.proposed_stack.includes('Rust-compatible service boundary'));
  assert.ok(architectureManifest.backend_system.proposed_stack.includes('PostgreSQL'));
  assert.ok(architectureManifest.backend_system.proposed_stack.includes('Redis'));
  assert.ok(architectureManifest.backend_system.proposed_stack.includes('GitHub API'));
  assert.ok(architectureManifest.backend_system.proposed_stack.includes('OpenAI API'));
  assert.ok(architectureManifest.backend_system.proposed_stack.includes('WebSocket gateway'));
  assert.deepEqual(architectureManifest.backend_system.runtime_routes, {
    authentication: '/api/auth/session',
    repository_import: '/api/repos/import',
    ai_orchestration: '/api/projects/{id}/ai-workflow',
    task_engine: '/api/projects/{id}/task-graph',
    payment_verification: '/api/payments/order-intents',
    escrow_coordination: '/api/projects/{id}/escrow',
    live_notifications: '/api/ws',
    ledger_system: '/api/public/ledger',
  });
  assert.deepEqual(Object.keys(architectureManifest.backend_system.service_boundaries), [
    'authentication',
    'repositories',
    'ai_orchestration',
    'task_engine',
    'payment_verification',
    'escrow_coordination',
    'live_notifications',
    'ledger_system',
  ]);
  assert.deepEqual(architectureManifest.backend_system.service_boundaries.repositories.output_protocols, ['mergeos.repo-import.v1', 'mergeos.scan.v1', 'mergeos.repo-sync.v1']);
  assert.deepEqual(architectureManifest.backend_system.service_boundaries.task_engine.output_protocols, ['mergeos.workflow.v1', 'mergeos.task.v1', 'mergeos.task-claim.v1', 'mergeos.proposal.v1']);
  assert.deepEqual(architectureManifest.backend_system.service_boundaries.escrow_coordination.output_protocols, ['mergeos.escrow.v1', 'mergeos.payouts.v1', 'mergeos.payout-release.v1']);
  assert.deepEqual(architectureManifest.backend_system.service_boundaries.ledger_system.output_protocols, ['mergeos.ledger.v1', 'mergeos.ledger-proof.v1', 'mergeos.token-economy.v1']);
  assert.ok(architectureManifest.ai_layer.capabilities.includes('estimate complexity'));
  assert.ok(architectureManifest.ai_layer.capabilities.includes('estimate time'));
  assert.ok(architectureManifest.ai_layer.capabilities.includes('estimate budget'));
  assert.ok(architectureManifest.ai_layer.capabilities.includes('assign worker type'));
  assert.deepEqual(architectureManifest.ai_layer.workflow, [
    'Import Repository',
    'Issue Scan',
    'Task Generation',
    'Reward Estimation',
    'Contributor Routing',
    'PR Review',
    'Deployment Validation',
  ]);
  assert.deepEqual(Object.keys(architectureManifest.ai_layer.agent_roles), [
    'coding_agent',
    'review_agent',
    'testing_agent',
    'deployment_agent',
  ]);
  assert.deepEqual(architectureManifest.ai_layer.agent_roles.coding_agent.primary_actions, [
    'implement scoped task',
    'generate patch',
    'attach repository context',
  ]);
  assert.equal(architectureManifest.ai_layer.agent_roles.review_agent.output_protocol, 'mergeos.pr-monitor.v1');
  assert.equal(architectureManifest.ai_layer.agent_roles.testing_agent.output_protocol, 'mergeos.ai-workflow.v1');
  assert.equal(architectureManifest.ai_layer.agent_roles.deployment_agent.output_protocol, 'mergeos.deployment.v1');
  assert.deepEqual(Object.keys(architectureManifest.ai_layer.ai_agent_capabilities), ['review', 'test', 'generate']);
  assert.equal(architectureManifest.ai_layer.ai_agent_capabilities.review.agent_role, 'review_agent');
  assert.equal(architectureManifest.ai_layer.ai_agent_capabilities.test.agent_role, 'testing_agent');
  assert.equal(architectureManifest.ai_layer.ai_agent_capabilities.generate.agent_role, 'coding_agent');
  assert.deepEqual(architectureManifest.ai_layer.action_routes, {
    scan_repository: {
      api: '/api/public/projects/{id}/repo-scan',
      output_protocol: 'mergeos.scan.v1',
      proof_surface: '/live-feed',
    },
    analyze_issues: {
      api: '/api/projects/{id}/ai-workflow',
      output_protocol: 'mergeos.ai-workflow.v1',
      proof_surface: '/dashboard',
    },
    generate_tasks: {
      api: '/api/projects/{id}/task-graph',
      output_protocol: 'mergeos.workflow.v1',
      proof_surface: '/marketplace#marketplace-bounties',
    },
    route_workers: {
      api: '/api/public/protocol/routing',
      output_protocol: 'mergeos.routing.v1',
      proof_surface: '/marketplace',
    },
    review_prs: {
      api: '/api/public/projects/{id}/pull-requests',
      output_protocol: 'mergeos.pr-monitor.v1',
      proof_surface: '/ledger',
    },
    test_builds: {
      api: '/api/public/projects/{id}/ai-workflow',
      output_protocol: 'mergeos.ai-workflow.v1',
      proof_surface: '/live-feed',
    },
    validate_deployments: {
      api: '/api/public/protocol/deployment',
      output_protocol: 'mergeos.deployment.v1',
      proof_surface: '/ledger',
    },
  });
  assert.ok(architectureManifest.marketplace_system.features.includes('Live Projects'));
  assert.ok(architectureManifest.marketplace_system.features.includes('Public Bounties'));
  assert.ok(architectureManifest.marketplace_system.features.includes('AI Agents'));
  assert.deepEqual(architectureManifest.marketplace_system.feature_routes, {
    live_projects: {
      page: '/marketplace#marketplace-projects',
      api: '/api/public/marketplace',
      event_type: 'project_funded',
    },
    public_bounties: {
      page: '/marketplace#marketplace-bounties',
      api: '/api/public/marketplace',
      event_type: 'task_opened',
    },
    contributors: {
      page: '/marketplace#marketplace-contributors',
      api: '/api/public/marketplace',
      event_type: 'proposal_submitted',
    },
    ai_agents: {
      page: '/marketplace#marketplace-agents',
      api: '/api/public/protocol/agent-queue',
      event_type: 'agent_queue',
    },
  });
  assert.deepEqual(architectureManifest.marketplace_system.action_routes, {
    publish_bounty: {
      command: 'Publish funded bounty from repository scope',
      api: '/api/projects/{id}/repo-scan/suggested-tasks/{taskID}/fund',
      output_protocol: 'mergeos.repo-task-funding.v1',
      proof_surface: '/marketplace#marketplace-bounties',
    },
    submit_proposal: {
      command: 'Submit contributor bid and availability packet',
      api: '/api/proposals',
      output_protocol: 'mergeos.proposal.v1',
      proof_surface: '/dashboard?section=worker',
    },
    claim_task: {
      command: 'Claim public bounty work',
      api: '/api/tasks/{id}/claim',
      output_protocol: 'mergeos.task-claim.v1',
      proof_surface: '/live-feed',
    },
    lease_agent_work: {
      command: 'Lease AI agent queue work before execution',
      api: '/api/agent-queue/leases',
      output_protocol: 'mergeos.agent-lease.v1',
      proof_surface: '/agents',
    },
    submit_evidence: {
      command: 'Submit PR, test, deployment, or agent evidence',
      api: '/api/tasks/{id}/submit',
      output_protocol: 'mergeos.task-submission.v1',
      proof_surface: '/ledger',
    },
  });
  assert.equal(architectureManifest.escrow_payment_system.role, 'Funding, escrow reserve, bounty funding, auto-release, and public payout proof lifecycle.');
  assert.deepEqual(architectureManifest.escrow_payment_system.payment_methods, ['card', 'PayPal', 'crypto', 'MRG token reserve']);
  assert.deepEqual(Object.keys(architectureManifest.escrow_payment_system.escrow_lifecycle), [
    'verify_funding',
    'lock_project_escrow',
    'reserve_task_bounty',
    'auto_release_payment',
    'publish_payout_proof',
  ]);
  assert.equal(architectureManifest.escrow_payment_system.escrow_lifecycle.verify_funding.output_protocol, 'mergeos.payment-order.v1');
  assert.equal(architectureManifest.escrow_payment_system.escrow_lifecycle.lock_project_escrow.output_protocol, 'mergeos.escrow.v1');
  assert.equal(architectureManifest.escrow_payment_system.escrow_lifecycle.auto_release_payment.api, '/api/projects/{id}/auto-release');
  assert.equal(architectureManifest.escrow_payment_system.escrow_lifecycle.publish_payout_proof.output_protocol, 'mergeos.ledger-proof.v1');
  assert.deepEqual(architectureManifest.live_feed_system.event_routes, {
    live_prs: {
      page: '/live-feed',
      api: '/api/public/live-feed',
      event_type: 'task_submitted',
      ledger_tab: 'Tasks & PRs',
    },
    deployments: {
      page: '/live-feed',
      api: '/api/public/live-feed',
      event_type: 'deployment_status',
      ledger_tab: 'Milestones',
    },
    contributors: {
      page: '/live-feed',
      api: '/api/public/live-feed',
      event_type: 'task_claimed',
      ledger_tab: 'Tasks & PRs',
    },
    ai_actions: {
      page: '/live-feed',
      api: '/api/public/live-feed',
      event_type: 'ai_review',
      ledger_tab: 'AI Actions',
    },
    payout_events: {
      page: '/live-feed',
      api: '/api/public/live-feed',
      event_type: 'ledger_task_payment',
      ledger_tab: 'Escrow & Payments',
    },
  });
  assert.deepEqual(architectureManifest.ledger_system.proof_routes, {
    payouts: {
      page: '/ledger',
      api: '/api/public/ledger/events',
      event_type: 'ledger_task_payment',
      proof_api: '/api/public/ledger/proof',
    },
    escrow_events: {
      page: '/ledger',
      api: '/api/public/ledger/events',
      event_type: 'payment_verified',
      proof_api: '/api/public/ledger/proof',
    },
    pr_events: {
      page: '/ledger',
      api: '/api/public/ledger/events',
      event_type: 'task_submitted',
      proof_api: '/api/public/ledger/proof',
    },
    ai_actions: {
      page: '/ledger',
      api: '/api/public/ledger/events',
      event_type: 'ai_review',
      proof_api: '/api/public/ledger/proof',
    },
    releases: {
      page: '/ledger',
      api: '/api/public/ledger/events',
      event_type: 'payout_released',
      proof_api: '/api/public/ledger/proof',
    },
  });
  assert.equal(architectureManifest.public_urls.marketplace_api, '/api/public/marketplace');
  assert.equal(architectureManifest.public_urls.live_feed_api, '/api/public/live-feed');
  assert.equal(architectureManifest.public_urls.agent_queue_api, '/api/public/protocol/agent-queue');
  assert.equal(architectureManifest.public_urls.ledger_api, '/api/public/ledger');
  assert.equal(architectureManifest.public_urls.ledger_events_api, '/api/public/ledger/events');
  assert.equal(architectureManifest.public_urls.ledger_verify_api, '/api/public/ledger/verify');
  assert.equal(architectureManifest.public_urls.ledger_proof_api, '/api/public/ledger/proof');
  assert.deepEqual(architectureManifest.token_economy_system.protocol_routes, {
    public_supply: {
      page: '/ledger',
      api: '/api/public/token-economy',
      output_protocol: 'mergeos.token-economy.v1',
      proof_surface: '/ledger',
    },
    airdrop_claims: {
      page: '/airdrop',
      api: '/api/airdrop/claims',
      output_protocol: 'mergeos.airdrop-claim.v1',
      proof_surface: '/api/public/ledger/proof',
    },
    presale_reservations: {
      page: '/presale',
      api: '/api/presale/reservations',
      output_protocol: 'mergeos.presale-reservation.v1',
      proof_surface: '/api/public/ledger/proof',
    },
    wallet_migration: {
      page: '/contracts',
      api: '/api/wallet/migration',
      output_protocol: 'mergeos.wallet-migration.v1',
      proof_surface: '/contracts/solana/mergeos_mrg.proof-manifest.v1.json',
    },
    payout_settlement: {
      page: '/dashboard',
      api: '/api/projects/{id}/payouts',
      output_protocol: 'mergeos.payouts.v1',
      proof_surface: '/ledger',
    },
    solana_contract: {
      page: '/contracts',
      api: '/contracts/solana/mergeos_mrg.v1.idl.json',
      output_protocol: 'mergeos.solana-mrg.v1',
      proof_surface: '/contracts/solana/mergeos_mrg.proof-manifest.v1.json',
    },
  });
  assert.equal(architectureManifest.public_urls.token_economy_api, '/api/public/token-economy');
  assert.equal(architectureManifest.public_urls.airdrop_missions_api, '/api/public/airdrop/missions');
  assert.equal(architectureManifest.public_urls.airdrop_claims_api, '/api/airdrop/claims');
  assert.equal(architectureManifest.public_urls.presale_reservations_api, '/api/presale/reservations');
  assert.equal(architectureManifest.public_urls.project_escrow_api, '/api/projects/{id}/escrow');
  assert.equal(architectureManifest.public_urls.project_payouts_api, '/api/projects/{id}/payouts');
  assert.equal(architectureManifest.public_urls.solana_mrg_idl, '/contracts/solana/mergeos_mrg.v1.idl.json');
  assert.equal(architectureManifest.public_urls.solana_mrg_proof_manifest, '/contracts/solana/mergeos_mrg.proof-manifest.v1.json');
  assert.equal(architectureManifest.public_urls.architecture_manifest, '/system/mergeos-architecture.v1.json');
});

test('public protocol schemas mirror the protocol package schemas', async () => {
  const sourceDir = new URL('../protocol/schemas/', import.meta.url);
  const publicDir = new URL('./public/protocol/', import.meta.url);
  const sourceFiles = (await fs.readdir(sourceDir)).filter((file) => file.endsWith('.schema.json')).sort();
  const publicFiles = (await fs.readdir(publicDir)).filter((file) => file.endsWith('.schema.json')).sort();

  assert.deepEqual(publicFiles, sourceFiles);
  for (const file of sourceFiles) {
    const sourceSchema = JSON.parse(await fs.readFile(new URL(file, sourceDir), 'utf-8'));
    const publicSchema = JSON.parse(await fs.readFile(new URL(file, publicDir), 'utf-8'));
    assert.deepEqual(publicSchema, sourceSchema);
  }
});

test('public protocol links match backend routes', async () => {
  const appSource = await fs.readFile(new URL('./src/App.vue', import.meta.url), 'utf-8');
  const manifestSource = await fs.readFile(new URL('../backend/internal/core/protocol_manifest.go', import.meta.url), 'utf-8');
  const paymentOrderSchema = JSON.parse(await fs.readFile(new URL('./public/protocol/payment-order.v1.schema.json', import.meta.url), 'utf-8'));
  const tokenLaunchBriefSchema = JSON.parse(await fs.readFile(new URL('./public/protocol/token-launch-brief.v1.schema.json', import.meta.url), 'utf-8'));
  const tokenLaunchBriefsSchema = JSON.parse(await fs.readFile(new URL('./public/protocol/token-launch-briefs.v1.schema.json', import.meta.url), 'utf-8'));
  const tokenLaunchCandidatesSchema = JSON.parse(await fs.readFile(new URL('./public/protocol/token-launch-candidates.v1.schema.json', import.meta.url), 'utf-8'));

  assert.match(appSource, /const publicProtocolManifestPath = '\/api\/public\/protocol';/);
  assert.equal(paymentOrderSchema.properties.provider.enum.includes('paypal'), true);
  assert.equal(paymentOrderSchema.properties.provider.enum.includes('stripe'), true);
  assert.equal(tokenLaunchBriefSchema.required.includes('ceo_memo'), true);
  assert.equal(tokenLaunchBriefSchema.required.includes('repository_url'), true);
  assert.equal(tokenLaunchBriefSchema.properties.ceo_memo.required.includes('gates'), true);
  assert.equal(tokenLaunchBriefSchema.properties.ceo_memo.properties.gates.items.properties.status.enum.includes('ready_for_review'), true);
  assert.equal(tokenLaunchBriefSchema.properties.ceo_memo.properties.gates.items.properties.status.enum.includes('needs_evidence'), true);
  assert.equal(tokenLaunchBriefsSchema.properties.protocol_version.const, 'mergeos.token-launch-briefs.v1');
  assert.equal(tokenLaunchBriefsSchema.required.includes('stats'), true);
  assert.equal(tokenLaunchBriefsSchema.required.includes('briefs'), true);
  assert.equal(tokenLaunchBriefsSchema.properties.briefs.items.required.includes('research_source'), true);
  assert.equal(Boolean(tokenLaunchBriefsSchema.properties.briefs.items.properties.project_summary), true);
  assert.equal(Boolean(tokenLaunchBriefsSchema.properties.briefs.items.properties.allocation_policy), true);
  assert.equal(Boolean(tokenLaunchBriefsSchema.properties.briefs.items.properties.proof_policy), true);
  assert.equal(Boolean(tokenLaunchBriefsSchema.properties.briefs.items.properties.wallet_policy), true);
  assert.equal(Boolean(tokenLaunchBriefsSchema.properties.briefs.items.properties.risk_notes), true);
  assert.equal(tokenLaunchCandidatesSchema.properties.protocol_version.const, 'mergeos.token-launch-candidates.v1');
  assert.equal(tokenLaunchCandidatesSchema.required.includes('candidates'), true);
  assert.equal(tokenLaunchCandidatesSchema.properties.stats.required.includes('ready_count'), true);
  assert.equal(tokenLaunchCandidatesSchema.properties.stats.required.includes('review_count'), true);
  assert.equal(tokenLaunchCandidatesSchema.properties.stats.required.includes('hold_count'), true);
  assert.equal(tokenLaunchCandidatesSchema.properties.candidates.items.required.includes('proof_policy'), true);
  assert.equal(tokenLaunchCandidatesSchema.properties.candidates.items.required.includes('next_action'), true);
  assert.equal(tokenLaunchCandidatesSchema.properties.candidates.items.required.includes('readiness_gates'), true);
  assert.equal(tokenLaunchCandidatesSchema.properties.candidates.items.required.includes('research_score'), true);
  assert.equal(tokenLaunchCandidatesSchema.properties.candidates.items.required.includes('decision_options'), true);
  assert.equal(tokenLaunchCandidatesSchema.properties.candidates.items.properties.decision_options.items.properties.key.enum.includes('needs_evidence'), true);
  assert.equal(tokenLaunchCandidatesSchema.properties.candidates.items.properties.readiness_gates.items.properties.state.enum.includes('ready'), true);
  assert.match(manifestSource, /mergeos\.payment-order\.v1/);
  assert.match(manifestSource, /payment-order\.v1\.schema\.json/);
  assert.match(manifestSource, /\/contracts\/solana\/mergeos_mrg\.proof-manifest\.v1\.json/);
  assert.match(manifestSource, /mergeos\.solana-contract-proof\.v1/);
  assert.match(manifestSource, /\/api\/public\/token\/launch-briefs/);
  assert.match(manifestSource, /mergeos\.token-launch-briefs\.v1/);
  assert.match(manifestSource, /token-launch-briefs\.v1\.schema\.json/);
  assert.match(manifestSource, /\/api\/public\/token\/launch-candidates/);
  assert.match(manifestSource, /mergeos\.token-launch-candidates\.v1/);
  assert.match(manifestSource, /token-launch-candidates\.v1\.schema\.json/);
  assert.match(manifestSource, /\/system\/mergeos-architecture\.v1\.json/);
  assert.match(manifestSource, /mergeos\.architecture\.v1/);
  assert.match(manifestSource, /architecture\.v1\.schema\.json/);
  assert.match(manifestSource, /CEO decision gates/);
  assert.match(manifestSource, /returns a CEO memo, launch gates, ledger receipt/);
  assert.match(manifestSource, /"architecture_manifest":\s+"\/system\/mergeos-architecture\.v1\.json"/);
  assert.match(appSource, /function publicTaskProtocolPath\(taskID = ''\)/);
  assert.match(appSource, /return id \? `\/api\/public\/protocol\/tasks\?task_id=\$\{encodeURIComponent\(id\)\}` : '\/api\/public\/protocol\/tasks';/);
  assert.match(appSource, /function publicProjectWorkflowPath\(projectID = ''\)/);
  assert.match(appSource, /return id \? `\/api\/public\/projects\/\$\{encodeURIComponent\(id\)\}\/workflow` : '';/);
  assert.match(appSource, /const protocolDocumentRowsRaw = computed\(\(\) => Array\.isArray\(protocolManifestView\.value\.documents\) \? protocolManifestView\.value\.documents : \[\]\);/);
  assert.match(appSource, /const stats = protocolManifestView\.value\.stats \|\| \{\};/);
  assert.match(appSource, /protocolManifestView\.value\.agent_context\?\.context_urls/);
  assert.match(appSource, /protocolManifestView\.value\.realtime\?\.websocket_path/);
  assert.match(manifestSource, /GeneratedAt:\s+time\.Now\(\)\.UTC\(\)/);
  assert.match(manifestSource, /Documents: protocolManifestDocuments\(schemas, endpoints\)/);
  assert.match(manifestSource, /ContextURLs: contextURLs/);
  assert.match(manifestSource, /WebSocketPath:\s+"\/api\/ws"/);
  assert.doesNotMatch(appSource, /\/api\/public\/protocol\/index/);
  assert.doesNotMatch(appSource, /\/api\/public\/bounties\/[^`'"]*\/protocol/);
  assert.doesNotMatch(appSource, /\/api\/public\/projects\/[^`'"]*\/workflow\/protocol/);
});

test('public protocol page exposes repository architecture artifacts', async () => {
  const appSource = await fs.readFile(new URL('./src/App.vue', import.meta.url), 'utf-8');
  const whitepaperSource = await fs.readFile(new URL('./public/whitepaper/mergeos-whitepaper.md', import.meta.url), 'utf-8');

  assert.match(appSource, /const protocolArtifactBaseRows = \[/);
  assert.match(appSource, /key: 'app'[\s\S]*name: 'mergeos-app'[\s\S]*artifacts: \['Frontend \+ SSR', 'Dashboards', 'Realtime feeds'\]/);
  assert.match(appSource, /key: 'contracts'[\s\S]*name: 'mergeos-contracts'[\s\S]*artifacts: \['MRG token', 'Escrow', 'Payout roots'\]/);
  assert.match(appSource, /key: 'sdk'[\s\S]*name: 'mergeos-sdk'[\s\S]*artifacts: \['JS client', 'Task APIs', 'WebSocket helpers'\]/);
  assert.match(appSource, /key: 'protocol'[\s\S]*name: 'mergeos-protocol'[\s\S]*artifacts: \['Schemas', 'Endpoint matrix', 'Agent runbook'\]/);
  assert.match(appSource, /key: 'architecture'[\s\S]*name: 'mergeos-architecture\.v1'[\s\S]*role: 'Machine-readable product architecture'/);
  assert.match(appSource, /href: '\/system\/mergeos-architecture\.v1\.json'/);
  assert.match(appSource, /contextPaths: \['\/system\/mergeos-architecture\.v1\.json', '\/protocol\/architecture\.v1\.schema\.json', '\/system', publicProtocolManifestPath\]/);
  assert.match(appSource, /artifacts: \['System vision', 'Repository map', 'AI workflow'\]/);
  assert.match(appSource, /Repository architecture[\s\S]*mergeos-app, mergeos-contracts, mergeos-sdk, and future mergeos-protocol/);
  assert.match(appSource, /Future protocol layer[\s\S]*decentralized execution, external AI agents, public integrations, task manifests, and open work standards/);
  assert.match(whitepaperSource, /The main application repository contains the frontend, backend, dashboards, SSR public pages, authentication, repository import, task engine, AI orchestration, payment verification, escrow coordination, realtime WebSocket feeds, public ledger pages, protocol discovery, and admin operations\./);
  assert.match(whitepaperSource, /The contracts repository contains the Solana\/Anchor path for MRG token utility\./);
  assert.match(whitepaperSource, /The SDK gives external clients and agents a small JavaScript interface for MergeOS APIs\./);
  assert.match(whitepaperSource, /The protocol layer defines public document shapes for tasks, claims, reviews, agents, agent queues, runbooks/);
});

test('MergeIDE release manifest points to pinned GitHub release assets', async () => {
  const manifestURL = new URL('./public/downloads/mergeide-windows-latest.json', import.meta.url);
  const manifest = JSON.parse(await fs.readFile(manifestURL, 'utf-8'));
  const repo = 'mergeos-bounties/mergeos';
  const tag = 'mergeide-windows-latest';
  const exe = 'MergeIDE-Windows-x64.exe';

  assert.equal(manifest.protocol_version, 'mergeos.release-artifact.v1');
  assert.equal(manifest.product, 'MergeIDE');
  assert.equal(manifest.release_tag, tag);
  assert.equal(manifest.file_name, exe);
  assert.equal(manifest.provenance.source_repository, repo);
  assert.equal(manifest.provenance.workflow_file, '.github/workflows/mergeide-windows-exe.yml');
  assert.equal(manifest.download_url, `https://github.com/${repo}/releases/download/${tag}/${exe}`);
  assert.equal(manifest.checksum_url, `https://github.com/${repo}/releases/download/${tag}/${exe}.sha256`);
  assert.equal(manifest.build_metadata_url, `https://github.com/${repo}/releases/download/${tag}/MergeIDE-Windows-x64.build.json`);
  assert.equal(manifest.release_url, `https://github.com/${repo}/releases/tag/${tag}`);
  assert.equal(manifest.provenance.workflow_url, `https://github.com/${repo}/actions/workflows/mergeide-windows-exe.yml`);
  assert.ok(manifest.links.some((link) => link.label === 'Windows exe' && link.url === manifest.download_url));
  assert.ok(manifest.links.some((link) => link.label === 'Release workflow' && link.url === manifest.provenance.workflow_url));
});

test('MergeIDE public page exposes the Windows exe download contract', async () => {
  const appSource = await fs.readFile(new URL('./src/App.vue', import.meta.url), 'utf-8');

  assert.match(appSource, /const mergeIdeDownloadFileName = 'MergeIDE-Windows-x64\.exe';/);
  assert.match(
    appSource,
    /const mergeIdeDownloadPath = `\$\{mergeIdeRepositoryPath\}\/releases\/download\/\$\{mergeIdeReleaseTag\}\/\$\{mergeIdeDownloadFileName\}`;/,
  );
  assert.match(appSource, /const mergeIdeReleasePath = `\$\{mergeIdeRepositoryPath\}\/releases\/tag\/\$\{mergeIdeReleaseTag\}`;/);
  assert.match(appSource, /const mergeIdeManifestPath = '\/downloads\/mergeide-windows-latest\.json';/);

  const downloadButtonBindings = appSource.match(
    /class="primary-button large mergeide-download-button"[\s\S]{0,260}:href="mergeIdeDownloadPath"[\s\S]{0,160}:download="mergeIdeDownloadFileName"/g,
  ) || [];
  assert.ok(downloadButtonBindings.length >= 1);
  assert.match(appSource, /class="home-mergeide-inline-link"[\s\S]{0,180}:href="mergeIdeDownloadPath"[\s\S]{0,120}:download="mergeIdeDownloadFileName"/);
  assert.ok(appSource.includes("['Pinned release',"));
  assert.ok(appSource.includes("['Release manifest',"));
});

test('public agent runbook and SDK document PR monitor auto-release plus proposal packets', async () => {
  const runbook = JSON.parse(await fs.readFile(new URL('./public/protocol/runbooks/mergeide-agent.v1.json', import.meta.url), 'utf-8'));
  const sdkReadme = await fs.readFile(new URL('../sdk/README.md', import.meta.url), 'utf-8');

  assert.ok(runbook.supported_agent_types.includes('deployment-agent'));
  assert.ok(runbook.supported_agent_types.includes('repo-scan-agent'));
  assert.ok(runbook.supported_agent_types.includes('security-review-agent'));
  assert.ok(runbook.context_urls.some((row) => row.protocol === 'mergeos.pr-monitor.v1' && row.auth === 'project'));
  assert.ok(runbook.claim_flow.some((step) => step.endpoint === '/api/projects/{id}/auto-release' && step.method === 'POST'));
  assert.ok(runbook.evidence_contract.optional.includes('PR monitor auto_release_packet payload'));
  assert.match(sdkReadme, /Agent Queue Claim/);
  assert.match(sdkReadme, /agentQueueClaimPayload/);
  assert.match(sdkReadme, /agentWorkPacketOutputContracts/);
  assert.match(sdkReadme, /repoPlanningSteps/);
  assert.match(sdkReadme, /repoPlanningOutputContracts/);
  assert.match(sdkReadme, /claimAgentQueueTask\(task, overrides\)/);
  assert.match(sdkReadme, /autoReleasePayloadFromPRMonitorTask/);
  assert.match(sdkReadme, /autoReleaseProofsFromResponse/);
  assert.match(sdkReadme, /autoReleaseLedgerProofLinksFromResponse/);
  assert.match(sdkReadme, /projectAutoReleaseFromPRMonitorTask\(projectID, task\)/);
  assert.match(sdkReadme, /agentReviewPayloadFromPRMonitorTask/);
  assert.match(sdkReadme, /createProjectAgentReviewFromPRMonitorTask\(projectID, task\)/);
  assert.match(sdkReadme, /deploymentValidationPayloadFromDeployment/);
  assert.match(sdkReadme, /createDeploymentValidationFromDeployment\(projectID, deployment/);
  assert.match(sdkReadme, /Marketplace Proposal Packet/);
  assert.match(sdkReadme, /proposalPayloadFromBounty/);
  assert.match(sdkReadme, /proposalPacketOutputContracts/);
  assert.match(sdkReadme, /adminOpsActionOutputContracts/);
  assert.match(sdkReadme, /adminOpsQueueOutputContracts/);
  assert.match(sdkReadme, /workerDashboardProofLinks/);
  assert.match(sdkReadme, /adminDisputes\(\)/);
  assert.match(sdkReadme, /createProposalFromBounty\(bounty, overrides\)/);
});

test('worker dashboard renders ledger proof links for accepted work and rewards', async () => {
  const appSource = await fs.readFile(new URL('./src/App.vue', import.meta.url), 'utf-8');
  const schema = JSON.parse(await fs.readFile(new URL('./public/protocol/worker-dashboard.v1.schema.json', import.meta.url), 'utf-8'));

  assert.match(appSource, /ledgerProofURL: task\.ledger_proof_url \|\| ''/);
  assert.match(appSource, /ledgerProofURL: entry\.ledger_proof_url \|\| ''/);
  assert.match(appSource, /workerDashboard\.value\.rewards/);
  assert.match(appSource, /workerDashboard\.value\.reputation/);
  assert.match(appSource, /workerDashboard\.value\.proposals/);
  assert.match(appSource, /workerDashboard\.value\.submitted_proposals/);
  assert.match(appSource, /v-if="task\.ledgerProofURL"[\s\S]{0,120}Proof/);
  assert.match(appSource, /v-if="reward\.ledgerProofURL"[\s\S]{0,120}Proof/);
  assert.match(appSource, /reputation_audit: payload\.reputation_audit && typeof payload\.reputation_audit === 'object' \? payload\.reputation_audit : \{\}/);
  assert.ok(schema.properties.claimed_tasks.items.properties.ledger_proof_url);
  assert.ok(schema.properties.rewards.items.properties.ledger_proof_url);
});

test('admin dashboard consumes admin ops queue action contract', async () => {
  const appSource = await fs.readFile(new URL('./src/App.vue', import.meta.url), 'utf-8');
  const cssSource = await fs.readFile(new URL('./src/styles.css', import.meta.url), 'utf-8');
  const adminOpsSchema = JSON.parse(await fs.readFile(new URL('./public/protocol/admin-ops.v1.schema.json', import.meta.url), 'utf-8'));

  const actionSchema = adminOpsSchema.properties.items.items.properties.actions.items.properties;
  assert.equal(adminOpsSchema.properties.output_contracts.items.$ref, '#/$defs/outputContract');
  assert.ok(adminOpsSchema.properties.stats.properties.high_count);
  assert.ok(adminOpsSchema.properties.stats.properties.blocked_payout_cents);
  assert.equal(actionSchema.output_contracts.items.$ref, '#/$defs/outputContract');
  assert.ok(adminOpsSchema.$defs.outputContract.required.includes('output_protocol_url'));
  assert.match(appSource, /queueActions: adminOpsQueueActions\(item\)/);
  assert.match(appSource, /api\('\/api\/admin\/disputes'\)/);
  assert.match(appSource, /api\('\/api\/admin\/summary'\)/);
  assert.match(appSource, /api\('\/api\/admin\/ops-queue'\)/);
  assert.match(appSource, /api\('\/api\/admin\/users'\)/);
  assert.match(appSource, /api\('\/api\/admin\/reputation'\)/);
  assert.match(appSource, /api\('\/api\/admin\/ledger'\)/);
  assert.match(appSource, /api\('\/api\/admin\/tasks'\)/);
  assert.match(appSource, /class="admin-triage-strip"/);
  assert.match(appSource, /const adminTriageRows = computed\(\(\) => \{/);
  assert.match(appSource, /function applyAdminTriageFilter\(item = \{\}\)/);
  assert.match(appSource, /function adminOpsQueueActions\(item = \{\}\)/);
  assert.match(appSource, /const adminOpsContractRows = computed\(\(\) =>/);
  assert.match(appSource, /adminConsole\.value\.ops\?\.output_contracts/);
  assert.match(appSource, /class="admin-ops-contract-strip"/);
  assert.match(appSource, /outputContracts: Array\.isArray\(action\.output_contracts\)/);
  assert.match(appSource, /v-for="action in item\.queueActions"/);
  assert.match(appSource, /@click="handleAdminOpsQueueAction\(item, action\)"/);
  assert.match(appSource, /case 'review_task_pulls':/);
  assert.match(appSource, /case 'run_ssl_review':/);
  assert.match(appSource, /method: action\.method \|\| 'POST'/);
  assert.match(appSource, /api\(action\.endpoint \|\| '\/api\/admin\/ssl\/review', options\)/);
  assert.match(appSource, /User Governance/);
  assert.match(appSource, /adminUserFilterRows/);
  assert.match(appSource, /function updateAdminUserRole\(row = \{\}, role = ''\)/);
  assert.match(appSource, /api\(`\/api\/admin\/users\/\$\{encodeURIComponent\(row\.id\)\}`/);
  assert.match(appSource, /mergeos\.admin-user-governance\.v1/);
  assert.match(cssSource, /\.admin-user-control-strip\s*\{[\s\S]*grid-template-columns: repeat\(4, minmax\(0, 1fr\)\);/);
  assert.match(cssSource, /\.admin-user-actions\s*\{[\s\S]*grid-template-columns: repeat\(3, minmax\(0, 1fr\)\);/);
  assert.match(cssSource, /\.dashboard-shell \.admin-user-actions,[\s\S]*\.dashboard-shell \.payment-history-actions\s*\{[\s\S]*grid-template-columns: minmax\(0, 1fr\) !important;/);
});

test('live feed agent packets expose action handoff links', async () => {
  const appSource = await fs.readFile(new URL('./src/App.vue', import.meta.url), 'utf-8');

  assert.match(appSource, /class="live-feed-agent-packet-actions"/);
  assert.match(appSource, /@click="openLiveFeedAgentQueue\(item\)"/);
  assert.match(appSource, /@click="openLiveFeedMergeIDE\(item\)"/);
  assert.match(appSource, /function openLiveFeedAgentQueue\(item = \{\}\)/);
  assert.match(appSource, /function openLiveFeedMergeIDE\(item = \{\}\)/);
  assert.match(appSource, /id="marketplace-agent-packets"/);
  assert.match(appSource, /bountyID: liveFeedAgentBountyID\(item, contextUrls\)/);
});

test('live feed page exposes realtime operating lanes', async () => {
  const appSource = await fs.readFile(new URL('./src/App.vue', import.meta.url), 'utf-8');
  const cssSource = await fs.readFile(new URL('./src/styles.css', import.meta.url), 'utf-8');

  assert.match(appSource, /class="live-feed-operating-strip"/);
  assert.match(appSource, /const liveFeedOperatingRows = computed/);
  assert.match(appSource, /class="live-feed-replay-actions"/);
  assert.match(appSource, /const liveFeedCursorReplayURL = computed/);
  assert.match(appSource, /after_id=\$\{encodeURIComponent\(latest\.id\)\}/);
  assert.match(appSource, /const liveFeedSinceReplayURL = computed/);
  assert.match(appSource, /copyLiveFeedReplayURL\('cursor'\)/);
  assert.match(appSource, /label: 'Live PRs'/);
  assert.match(appSource, /label: 'Deployments'/);
  assert.match(appSource, /label: 'Active contributors'/);
  assert.match(appSource, /label: 'AI actions'/);
  assert.match(appSource, /if \(payload\.type === 'contributor_activity'\)/);
  assert.match(appSource, /if \(payload\.type === 'ai_review'\)/);
  assert.match(appSource, /if \(realtimeDeploymentEventTypes\.has\(payload\.type\)\)/);
  assert.match(appSource, /const realtimeDeploymentEventTypes = new Set\(\['deployment_status'\]\)/);
  assert.match(appSource, /deploymentRealtimeLiveFeedItem\(payload\)/);
  assert.match(appSource, /handleWSAgentAction\(payload\)/);
  assert.match(cssSource, /\.live-feed-operating-strip\s*\{[\s\S]*grid-template-columns: repeat\(4, minmax\(0, 1fr\)\);/);
  assert.match(cssSource, /\.live-feed-replay-actions\s*\{[\s\S]*grid-template-columns: repeat\(2, minmax\(0, 1fr\)\);/);
  assert.match(cssSource, /@media \(max-width: 760px\)[\s\S]*\.live-feed-operating-strip\s*\{[\s\S]*grid-template-columns: 1fr;/);
});

test('live feed exposes the full AI workflow trace', async () => {
  const appSource = await fs.readFile(new URL('./src/App.vue', import.meta.url), 'utf-8');

  assert.match(appSource, /const liveFeedWorkflowTraceDefinitions = \[/);
  assert.match(appSource, /phase: 'Import Repository'[\s\S]*phase: 'Issue Scan'[\s\S]*phase: 'Task Generation'[\s\S]*phase: 'Reward Estimation'[\s\S]*phase: 'Contributor Routing'[\s\S]*phase: 'PR Review'[\s\S]*phase: 'Deployment Validation'/);
  assert.match(appSource, /rawTypes: \['project_funded', 'repo_scan'\]/);
  assert.match(appSource, /body: 'Repository scan rows expose bugs, technical debt, dependencies, and risk signals before tasking\.'/);
  assert.match(appSource, /schema: 'task_opened \| agent_action'/);
  assert.match(appSource, /schema: 'task_claimed \| agent_\*'/);
  assert.match(appSource, /schema: 'task_submitted \| task_changes_requested \| task_accepted \| ai_review'/);
  assert.match(appSource, /schema: 'deployment_validation \| deployment_status'/);
  assert.match(appSource, /phase: 'Payout \/ Ledger Release'/);
});

test('public menus and signed-in mobile layout keep reachable compact surfaces', async () => {
  const cssSource = await fs.readFile(new URL('./src/styles.css', import.meta.url), 'utf-8');

  assert.match(cssSource, /Public menu \+ signed-in mobile stabilization owner/);
  assert.match(cssSource, /\.nav-menu\.open::after\s*\{[\s\S]*height: 132px;/);
  assert.match(cssSource, /\.nav-context-menu\s*\{[\s\S]*z-index: 430;[\s\S]*overflow: hidden;/);
  assert.match(cssSource, /\.product-menu\s*\{[\s\S]*width: min\(1180px, calc\(100vw - 32px\)\);/);
  assert.match(cssSource, /\.hero-section\s*\{[\s\S]*min-height: calc\(100dvh - 82px\);/);
  assert.match(cssSource, /\.product-console\s*\{[\s\S]*max-width: 610px;/);
  assert.match(cssSource, /@media \(max-width: 980px\)[\s\S]*\.nav-context-menu::before\s*\{[\s\S]*pointer-events: none;/);
  assert.match(cssSource, /\.dashboard-shell \.dash-mobile-nav\s*\{[\s\S]*grid-template-columns: repeat\(4, minmax\(0, 1fr\)\);/);
  assert.match(cssSource, /\.notification-dropdown,[\s\S]*\.dashboard-shell \.notification-dropdown,[\s\S]*\.dashboard-shell \.account-context-menu,[\s\S]*\.dashboard-shell \.dashboard-project-actions-panel\s*\{[\s\S]*position: fixed !important;/);
  assert.match(cssSource, /max-width: calc\(100vw - \(var\(--dash-mobile-gutter, 14px\) \* 2\)\) !important;/);
  assert.match(cssSource, /@media \(max-width: 620px\)[\s\S]*\.home-navbar \.nav-inner\s*\{[\s\S]*grid-template-columns: minmax\(0, 1fr\) auto auto !important;/);
  assert.match(cssSource, /@media \(max-width: 620px\)[\s\S]*\.home-navbar \.public-nav-actions\s*\{[\s\S]*grid-column: 2 !important;[\s\S]*grid-row: 1 !important;/);
  assert.match(cssSource, /@media \(max-width: 620px\)[\s\S]*\.home-navbar \.hamburger-button\s*\{[\s\S]*grid-column: 3 !important;[\s\S]*grid-row: 1 !important;/);
  assert.match(cssSource, /@media \(max-width: 620px\)[\s\S]*\.home-navbar \.public-nav-actions \.locale-button\s*\{[\s\S]*font-size: 11px !important;/);
  assert.match(cssSource, /Public mobile nav fit: language, account, and menu must never clip on token pages/);
  assert.match(cssSource, /\/\* Public mobile nav fit:[\s\S]*\.home-navbar \.nav-inner\s*\{[\s\S]*display: grid !important;[\s\S]*grid-template-columns: minmax\(0, 1fr\) auto auto !important;/);
  assert.match(cssSource, /\/\* Public mobile nav fit:[\s\S]*\.home-navbar \.public-nav-actions\s*\{[\s\S]*grid-template-columns: 38px 38px !important;[\s\S]*margin-right: 42px !important;/);
  assert.match(cssSource, /\/\* Public mobile nav fit:[\s\S]*\.home-navbar \.account-icon-button > svg \+ svg,[\s\S]*\.home-navbar \.account-icon-button > \.profile-avatar \+ svg\s*\{[\s\S]*display: none !important;/);
});

test('public home keeps a short decision-screen rhythm', async () => {
  const appSource = await fs.readFile(new URL('./src/App.vue', import.meta.url), 'utf-8');
  const cssSource = await fs.readFile(new URL('./src/styles.css', import.meta.url), 'utf-8');

  assert.match(cssSource, /Home executive pass: a short premium decision screen/);
  assert.match(appSource, /class="public-notification-feed home-feed-preview"/);
  assert.match(appSource, /class="home-mergeide-inline-link"/);
  assert.match(appSource, /class="home-value-strip"/);
  assert.match(appSource, /class="home-definition-strip"/);
  assert.match(appSource, /const homeDefinitionRows = computed/);
  assert.match(appSource, /class="home-explain-strip"/);
  assert.match(appSource, /v-for="row in homeOperatingRows"/);
  assert.match(appSource, /class="home-ceo-token-desk"/);
  assert.match(appSource, /tokenDeskTitle: 'Research airdrop and presale candidates before opening MRG\.'/);
  assert.match(appSource, /const homeTokenSignalRows = computed/);
  assert.match(appSource, /homeSystemSummaryRows/);
  assert.match(appSource, /homeOperatingRows/);
  assert.match(appSource, /homePipelineRows/);
  assert.doesNotMatch(appSource, /class="home-operating-note"/);
  assert.doesNotMatch(appSource, /class="home-system-summary"/);
  assert.doesNotMatch(appSource, /class="home-compact-flow"/);
  assert.doesNotMatch(appSource, /class="home-system-explainer"/);
  assert.doesNotMatch(appSource, /localizedHomeWorkflowCards\.slice\(0, 4\)/);
  assert.match(appSource, /homeLiveStats\.slice\(0, 2\)/);
  assert.match(appSource, /Fund software work, route tasks, prove delivery\./);
  assert.match(appSource, /MergeOS turns a brief or repo into funded work: CEO agents scope it, builders and AI agents execute it, escrow and Solana MRG track money, and every PR, deploy, payout, and receipt lands on a public proof ledger\./);
  assert.match(appSource, /definitionRows: \[[\s\S]*title: 'Brief to scope', body: 'Product brief, repo, files, issues, budget, deadline, and acceptance criteria become one funded work packet\.'/);
  assert.match(appSource, /title: 'CEO routing', body: 'CEO agents split scope into builder, AI agent, QA, and DevOps tasks with owners and status\.'/);
  assert.match(appSource, /title: 'Escrow \+ MRG', body: 'Escrow funding, reserve state, and Solana MRG accounting follow each task before payout\.'/);
  assert.match(appSource, /title: 'Proof ledger', body: 'PRs, deploys, approvals, payouts, receipts, and contract references stay public and traceable\.'/);
  assert.match(appSource, /title: 'Đầu vào', body: 'Brief, repo, issue, file, budget và deadline\.'/);
  assert.match(appSource, /MergeOS biến brief hoặc repo thành funded tasks có CEO-agent planning, builder\/AI routing, escrow, PR\/deploy checks, Solana MRG accounting và public ledger proof\./);
  assert.match(appSource, /title: 'Product OS'[\s\S]*Project intake, repo import, AI task graph, escrow, PR monitor, deployment gates, and ledger proof stay in one operating flow\./);
  assert.match(appSource, /title: 'Delivery lanes'[\s\S]*Route funded work to human contributors, AI agents, or hybrid teams with shared scope, acceptance criteria, and payout state\./);
  assert.match(appSource, /title: 'Public proof layer'[\s\S]*Marketplace activity, escrow, token mint, PR evidence, deployment checks, SDK context, and protocol documents are discoverable\./);
  assert.match(appSource, /title: 'Repo OS', body: 'Issues, debt, PRs, deploys'/);
  assert.match(appSource, /title: 'AI routing', body: 'Scan, split, estimate, review'/);
  assert.match(appSource, /title: 'Market \+ escrow', body: 'Funded bounties and payouts'/);
  assert.match(appSource, /title: 'MRG proof', body: 'Solana token and public ledger'/);
  assert.match(appSource, /operatingRows: \[[\s\S]*title: '1\. Import context', body: 'Brief, repo, issues, files, budget, and acceptance criteria enter one workspace\.'/);
  assert.match(appSource, /title: '2\. CEO routes work', body: 'The CEO agent turns scope into funded tasks for builders, agents, QA, and DevOps\.'/);
  assert.match(appSource, /title: '3\. Prove delivery', body: 'PR, deploy, acceptance, Solana MRG, and payout ledger evidence stay traceable\.'/);
  assert.match(appSource, /title: '2\. CEO route việc', body: 'CEO agent biến scope thành funded task cho builder, agent, QA và DevOps\.'/);
  assert.match(appSource, /title: '3\. Prove delivery', body: 'PR, deploy, acceptance, Solana MRG và payout ledger luôn truy vết được\.'/);
  assert.match(appSource, /title: 'Route', body: 'Human, AI, or hybrid tasks'/);
  assert.match(appSource, /title: 'Prove', body: 'PR, deploy, payout ledger'/);
  assert.match(cssSource, /\.public-home-page\s*\{[\s\S]*padding-block: clamp\(8px, 1\.4vw, 18px\) clamp\(14px, 2vw, 24px\) !important;/);
  assert.match(cssSource, /Homepage product polish: wider desktop rhythm, shorter proof rail/);
  assert.match(cssSource, /\.public-home-page \.home-container\s*\{[\s\S]*max-width: 1120px !important;/);
  assert.match(cssSource, /\.public-home-hero\s*\{[\s\S]*grid-template-columns: minmax\(0, 1fr\) minmax\(320px, 380px\) !important;/);
  assert.match(cssSource, /\.public-home-copy h1\s*\{[\s\S]*font-size: clamp\(42px, 4\.4vw, 68px\) !important;/);
  assert.match(cssSource, /\.home-definition-strip\s*\{[\s\S]*grid-template-columns: repeat\(4, minmax\(0, 1fr\)\) !important;/);
  assert.match(cssSource, /\.home-definition-strip small\s*\{[\s\S]*display: block !important;/);
  assert.match(cssSource, /Home mobile proof strip: keep the full product meaning without turning the first screen into a long stack/);
  assert.match(cssSource, /@media \(max-width: 620px\)[\s\S]*\.home-definition-strip\s*\{[\s\S]*grid-template-columns: repeat\(2, minmax\(0, 1fr\)\) !important;/);
  assert.match(cssSource, /@media \(max-width: 620px\)[\s\S]*\.home-definition-strip article\s*\{[\s\S]*min-height: 104px !important;/);
  assert.match(cssSource, /@media \(max-width: 620px\)[\s\S]*\.home-definition-strip small\s*\{[\s\S]*-webkit-line-clamp: 3 !important;/);
  assert.match(cssSource, /\.home-command-panel\s*\{[\s\S]*max-width: 380px !important;/);
  assert.match(cssSource, /\.home-explain-strip\s*\{[\s\S]*grid-template-columns: repeat\(3, minmax\(0, 1fr\)\) !important;/);
  assert.match(cssSource, /@media \(max-width: 980px\)[\s\S]*\.home-explain-strip\s*\{[\s\S]*grid-template-columns: 1fr !important;/);
  assert.match(cssSource, /\.home-system-summary,[\s\S]*\.home-compact-flow,[\s\S]*\.home-system-explainer,[\s\S]*\.home-command-panel \.home-feed-preview,[\s\S]*\.home-command-panel \.home-pipeline\s*\{[\s\S]*display: none !important;/);
  assert.match(cssSource, /\.home-command-panel \.public-stat-grid article:nth-child\(n \+ 3\)\s*\{[\s\S]*display: none !important;/);
  assert.match(cssSource, /@media \(max-width: 980px\)[\s\S]*\.home-command-panel\s*\{[\s\S]*max-width: 100% !important;/);
  assert.match(cssSource, /@media \(max-width: 620px\)[\s\S]*\.public-home-copy h1\s*\{[\s\S]*font-size: clamp\(26px, 7\.4vw, 31px\) !important;/);
  assert.match(cssSource, /Home description pass: enough product meaning, still a short first screen/);
  assert.match(cssSource, /\.home-explain-strip article\s*\{[\s\S]*min-height: 62px !important;/);
  assert.match(cssSource, /\.home-explain-strip span\s*\{[\s\S]*-webkit-line-clamp: 2 !important;/);
  assert.match(cssSource, /Home compact product story pass: enough meaning without a tall manual page/);
  assert.match(cssSource, /\/\* Home compact product story pass:[\s\S]*\.public-home-page\s*\{[\s\S]*min-height: auto !important;[\s\S]*padding-block: clamp\(10px, 1\.8vw, 24px\) clamp\(18px, 2\.6vw, 32px\) !important;/);
  assert.match(cssSource, /\/\* Home compact product story pass:[\s\S]*\.public-home-page \.home-container\s*\{[\s\S]*max-width: 1040px !important;/);
  assert.match(cssSource, /\/\* Home compact product story pass:[\s\S]*\.home-command-panel\s*\{[\s\S]*align-self: start !important;/);
  assert.match(cssSource, /Home CEO front door pass: shorter, cleaner, more like a product cockpit than a long brochure/);
  assert.match(cssSource, /\/\* Home CEO front door pass:[\s\S]*\.public-home-page\s*\{[\s\S]*padding-block: clamp\(20px, 3vw, 38px\) clamp\(22px, 3vw, 40px\) !important;/);
  assert.match(cssSource, /\/\* Home CEO front door pass:[\s\S]*\.public-home-copy h1\s*\{[\s\S]*font-size: clamp\(42px, 5vw, 66px\) !important;/);
  assert.match(cssSource, /\/\* Home CEO front door pass:[\s\S]*\.home-command-panel \.home-feed-preview\s*\{[\s\S]*display: none !important;/);
  assert.match(cssSource, /@media \(max-width: 980px\)[\s\S]*\.home-command-panel\s*\{[\s\S]*display: none !important;/);
  assert.match(cssSource, /Home complete description final pass: concise first screen, complete product meaning/);
  assert.match(cssSource, /\/\* Home complete description final pass:[\s\S]*\.public-home-page \.home-container\s*\{[\s\S]*max-width: 1080px !important;/);
  assert.match(cssSource, /\/\* Home complete description final pass:[\s\S]*\.public-home-copy p\s*\{[\s\S]*max-width: 62ch !important;[\s\S]*line-height: 1\.48 !important;/);
  assert.match(cssSource, /\/\* Home complete description final pass:[\s\S]*\.home-definition-strip article\s*\{[\s\S]*min-height: 96px !important;/);
  assert.match(cssSource, /\/\* Home complete description final pass:[\s\S]*@media \(max-width: 620px\)[\s\S]*\.home-definition-strip small\s*\{[\s\S]*-webkit-line-clamp: 2 !important;/);
  assert.match(cssSource, /Home full-description restore: let the proof cards say enough without becoming a tall brochure/);
  assert.match(cssSource, /\/\* Home full-description restore:[\s\S]*\.home-definition-strip small\s*\{[\s\S]*-webkit-line-clamp: unset !important;/);
  assert.match(cssSource, /\/\* Home full-description restore:[\s\S]*@media \(max-width: 620px\)[\s\S]*\.home-definition-strip\s*\{[\s\S]*grid-template-columns: minmax\(0, 1fr\) !important;/);
  assert.match(cssSource, /\/\* Home full-description restore:[\s\S]*@media \(max-width: 620px\)[\s\S]*\.home-explain-strip\s*\{[\s\S]*display: none !important;/);
  assert.match(cssSource, /Home CEO token desk final pass: one short premium decision screen, not a long brochure/);
  assert.match(cssSource, /\/\* Home CEO token desk final pass:[\s\S]*\.public-home-page\s*\{[\s\S]*min-height: calc\(100dvh - 74px\) !important;/);
  assert.match(cssSource, /\.home-ceo-token-desk\s*\{[\s\S]*grid-template-columns: minmax\(0, 1fr\) auto;/);
  assert.match(cssSource, /\.home-ceo-token-desk > div\s*\{[\s\S]*grid-template-columns: repeat\(3, minmax\(74px, 1fr\)\);/);
  assert.match(cssSource, /@media \(max-width: 620px\)[\s\S]*\.home-ceo-token-desk > div\s*\{[\s\S]*grid-template-columns: repeat\(3, minmax\(0, 1fr\)\) !important;/);
  assert.match(cssSource, /Home studio pass: concise, premium first screen with enough product proof/);
  assert.match(cssSource, /\/\* Home studio pass:[\s\S]*\.public-home-page\s*\{[\s\S]*padding-block: clamp\(22px, 3vw, 44px\) clamp\(18px, 2\.4vw, 30px\) !important;/);
  assert.match(cssSource, /\/\* Home studio pass:[\s\S]*\.public-home-copy h1\s*\{[\s\S]*font-size: clamp\(44px, 5\.1vw, 70px\) !important;/);
  assert.match(cssSource, /\/\* Home studio pass:[\s\S]*\.home-command-panel \.public-stat-grid\s*\{[\s\S]*grid-template-columns: minmax\(0, 1fr\) !important;/);
  assert.match(cssSource, /\/\* Home studio pass:[\s\S]*@media \(max-width: 620px\)[\s\S]*\.public-home-copy > p\s*\{[\s\S]*display: block !important;/);
  assert.match(cssSource, /Home CEO-grade cut: one tight product decision screen, no tall side console/);
  assert.match(cssSource, /\/\* Home CEO-grade cut:[\s\S]*\.public-home-page\s*\{[\s\S]*min-height: calc\(100dvh - 74px\) !important;[\s\S]*align-items: center !important;/);
  assert.match(cssSource, /\/\* Home CEO-grade cut:[\s\S]*\.public-home-page \.home-container\s*\{[\s\S]*max-width: 980px !important;/);
  assert.match(cssSource, /\/\* Home CEO-grade cut:[\s\S]*\.public-home-copy\s*\{[\s\S]*width: min\(100%, 760px\) !important;[\s\S]*margin-inline: auto !important;/);
  assert.match(cssSource, /Home narrative restore: enough product context without returning to a long landing page/);
  assert.match(cssSource, /\.home-value-strip\s*\{[\s\S]*grid-template-columns: repeat\(3, minmax\(0, 1fr\)\) !important;/);
  assert.match(cssSource, /@media \(max-width: 760px\)[\s\S]*\.home-value-strip\s*\{[\s\S]*grid-template-columns: minmax\(0, 1fr\) !important;/);
  assert.match(cssSource, /Home phone viewport lock: complete copy must fit inside the visible phone width/);
  assert.match(cssSource, /@media \(max-width: 620px\)[\s\S]*\.public-home-page \.home-container,[\s\S]*\.public-home-copy,[\s\S]*\.home-value-strip,[\s\S]*\.home-definition-strip,[\s\S]*\.home-ceo-token-desk/);
  assert.match(cssSource, /@media \(max-width: 620px\)[\s\S]*width: min\(366px, calc\(100vw - 24px\)\) !important;/);
  assert.match(cssSource, /\/\* Home CEO-grade cut:[\s\S]*\.home-explain-strip,[\s\S]*\.home-command-panel\s*\{[\s\S]*display: none !important;/);
  assert.match(cssSource, /\/\* Home CEO-grade cut:[\s\S]*@media \(max-width: 760px\)[\s\S]*\.public-home-page \.home-container\s*\{[\s\S]*width: calc\(100vw - 20px\) !important;[\s\S]*max-width: calc\(100vw - 20px\) !important;/);
  assert.match(cssSource, /\/\* Home CEO-grade cut:[\s\S]*@media \(max-width: 520px\)[\s\S]*\.home-definition-strip\s*\{[\s\S]*grid-template-columns: minmax\(0, 1fr\) !important;/);
  assert.match(cssSource, /\/\* Home CEO-grade cut:[\s\S]*@media \(max-width: 520px\)[\s\S]*\.home-definition-strip article\s*\{[\s\S]*min-height: auto !important;[\s\S]*overflow: visible !important;/);
  assert.match(cssSource, /\/\* Home CEO-grade cut:[\s\S]*@media \(max-width: 520px\)[\s\S]*\.home-definition-strip small\s*\{[\s\S]*-webkit-line-clamp: unset !important;[\s\S]*overflow: visible !important;/);
  assert.match(cssSource, /\/\* Home CEO-grade cut:[\s\S]*@media \(max-width: 520px\)[\s\S]*\.public-home-copy \.marketplace-actions\s*\{[\s\S]*grid-template-columns: minmax\(0, 1fr\) !important;/);
});

test('frontend system exposes required public pages and dashboard roles', async () => {
  const appSource = await fs.readFile(new URL('./src/App.vue', import.meta.url), 'utf-8');
  const packageSource = JSON.parse(await fs.readFile(new URL('./package.json', import.meta.url), 'utf-8'));
  const mainSource = await fs.readFile(new URL('./src/main.js', import.meta.url), 'utf-8');
  const clientSource = await fs.readFile(new URL('./src/entry-client.js', import.meta.url), 'utf-8');

  assert.ok(packageSource.dependencies.vue);
  assert.ok(packageSource.dependencies['@vue/server-renderer']);
  assert.ok(packageSource.devDependencies.vite);
  assert.match(packageSource.scripts['build:production'], /vite build --mode production --outDir dist\/client --ssrManifest && vite build --mode production --ssr src\/entry-server\.js/);
  assert.match(mainSource, /createSSRApp/);
  assert.match(clientSource, /hasSSRMarkup \? createHydratedApp\(initialPath\) : createClientApp/);
  assert.match(appSource, /Vue 3, Vite SSR, Tailwind-style design tokens, WebSocket updates, and realtime event hydration/);
  assert.match(appSource, /new WebSocket\(wsURL\(\)\)/);
  for (const [page, pathValue] of [
    ['home', '/'],
    ['marketplace', '/marketplace'],
    ['live', '/live-feed'],
    ['ledger', '/ledger'],
    ['customers', '/customers'],
    ['contributors', '/contributors'],
    ['agents', '/agents'],
    ['admins', '/admins'],
  ]) {
    assert.match(appSource, new RegExp(`${page}: '${pathValue.replace('/', '\\/')}'`));
  }
  assert.match(appSource, /marketplace: \['\/work-marketplace'[\s\S]*'\/live-projects'[\s\S]*'\/public-bounties'[\s\S]*'\/ai-agent-marketplace'\]/);
  assert.match(appSource, /live: \['\/live'[\s\S]*'\/live-prs'[\s\S]*'\/deployment-feed'[\s\S]*'\/ai-action-feed'/);
  assert.match(appSource, /ledger: \['\/ledger-logs'[\s\S]*'\/escrow-events'[\s\S]*'\/payout-logs'[\s\S]*'\/ai-action-logs'[\s\S]*'\/pr-proof-logs'\]/);
  assert.match(appSource, /const dashboardRoleCoverageRows = computed\(\(\) => \{/);
  assert.match(appSource, /label: 'Worker Dashboard'/);
  assert.match(appSource, /label: 'Admin Console'/);
  assert.match(appSource, /key: 'customer'[\s\S]*title: 'Project owner cockpit'/);
  assert.match(appSource, /key: 'worker'[\s\S]*title: 'Contributor workbench'/);
  assert.match(appSource, /key: 'ai-agents'[\s\S]*title: 'Agent execution layer'/);
  assert.match(appSource, /key: 'admin'[\s\S]*title: 'Treasury and trust ops'/);
  assert.match(appSource, /label: 'Project overview'[\s\S]*label: 'Live PRs'[\s\S]*label: 'Escrow'[\s\S]*label: 'Payments'[\s\S]*label: 'Tasks'[\s\S]*label: 'AI logs'/);
  assert.match(appSource, /label: 'Claimed tasks'[\s\S]*label: 'Rewards'[\s\S]*label: 'Reputation'[\s\S]*label: 'Proposals'/);
  assert.match(appSource, /label: 'Coding agents'[\s\S]*label: 'Review agents'[\s\S]*label: 'Testing agents'[\s\S]*label: 'Deployment agents'/);
  assert.match(appSource, /label: 'Treasury'[\s\S]*label: 'Users'[\s\S]*label: 'Disputes'[\s\S]*label: 'Payouts'[\s\S]*label: 'Moderation'/);
  assert.match(appSource, /Founders, startups, SaaS teams, and repo owners/);
  assert.match(appSource, /Frontend, backend, design, QA, DevOps, and security contributors/);
  assert.match(appSource, /Coding, review, testing, deployment, and security agents/);
  assert.match(appSource, /treasury operators, dispute handlers, moderators, and payout managers/);
});

test('customer dashboard exposes compact operating lanes after login', async () => {
  const appSource = await fs.readFile(new URL('./src/App.vue', import.meta.url), 'utf-8');
  const cssSource = await fs.readFile(new URL('./src/styles.css', import.meta.url), 'utf-8');

  assert.match(appSource, /class="customer-dashboard-operating-strip"/);
  assert.match(appSource, /const customerDashboardOperatingRows = computed/);
  assert.match(appSource, /label: 'Project overview'/);
  assert.match(appSource, /label: 'Live PRs'/);
  assert.match(appSource, /label: 'Escrow'/);
  assert.match(appSource, /label: 'Payments'/);
  assert.match(appSource, /label: 'Tasks'/);
  assert.match(appSource, /label: 'AI logs'/);
  assert.match(appSource, /api\('\/api\/customers\/me'\)/);
  assert.match(appSource, /api\(`\/api\/projects\/\$\{encodeURIComponent\(targetProjectID\)\}\/escrow`\)/);
  assert.match(appSource, /api\(`\/api\/projects\/\$\{encodeURIComponent\(targetProjectID\)\}\/ai-workflow`\)/);
  assert.match(appSource, /api\(`\/api\/projects\/\$\{encodeURIComponent\(targetProjectID\)\}\/dashboard`\)/);
  assert.match(appSource, /payload\.workflow_pulse \|\| dashboardWorkflowPulseFromProjectDashboard\(payload, targetProjectID\)/);
  assert.match(appSource, /escrow: payload\.escrow \|\| null/);
  assert.match(appSource, /pull_requests: payload\.pull_requests \|\| null/);
  assert.match(appSource, /loadDashboardProjectDashboardData\(selectedDashboardProjectID\.value, \{ silent: true \}\)/);
  assert.match(appSource, /function handleCustomerDashboardOperatingLane/);
  assert.match(appSource, /class="dashboard-role-proof"/);
  assert.match(appSource, /nextStep: 'Review delivery'/);
  assert.match(appSource, /proof: 'AI evidence'/);
  assert.match(cssSource, /\.customer-dashboard-operating-strip\s*\{[\s\S]*grid-template-columns: repeat\(6, minmax\(0, 1fr\)\);/);
  assert.match(cssSource, /@media \(max-width: 760px\)[\s\S]*\.dashboard-shell \.customer-dashboard-operating-strip\s*\{[\s\S]*grid-template-columns: repeat\(2, minmax\(0, 1fr\)\) !important;/);
  assert.match(cssSource, /Signed-in home quality pass: compact command center/);
  assert.match(cssSource, /\.dashboard-shell \.dash-command-strip\s*\{[\s\S]*width: min\(100% - 36px, 1120px\);[\s\S]*grid-template-columns: minmax\(0, 0\.95fr\) minmax\(360px, 1fr\) !important;/);
  assert.match(cssSource, /\.dashboard-shell \.dash-command-copy h1\s*\{[\s\S]*font-size: clamp\(34px, 3\.4vw, 48px\) !important;/);
  assert.match(cssSource, /\.dashboard-shell \.dashboard-role-map\s*\{[\s\S]*grid-template-columns: repeat\(4, minmax\(0, 1fr\)\) !important;/);
  assert.match(cssSource, /\.dashboard-shell \.dashboard-role-proof\s*\{[\s\S]*grid-template-columns: repeat\(2, minmax\(0, 1fr\)\) !important;/);
  assert.match(cssSource, /\.dashboard-shell \.dashboard-role-stats,[\s\S]*\.dashboard-shell \.dashboard-role-lanes\s*\{[\s\S]*display: none !important;/);
  assert.match(cssSource, /Signed-in mobile readability owner/);
  assert.match(cssSource, /@media \(max-width: 760px\)[\s\S]*\.dashboard-shell \.dashboard-role-map\s*\{[\s\S]*grid-auto-flow: row !important;[\s\S]*grid-template-columns: minmax\(0, 1fr\) !important;[\s\S]*mask-image: none !important;/);
  assert.doesNotMatch(cssSource, /\.dashboard-shell \.dashboard-role-map article:nth-child\(n \+ 3\)\s*\{[\s\S]*display: none !important;/);
});

test('customer PR monitor consumes backend pull-request task groups', async () => {
  const appSource = await fs.readFile(new URL('./src/App.vue', import.meta.url), 'utf-8');

  assert.match(appSource, /api\(`\/api\/projects\/\$\{encodeURIComponent\(targetProjectID\)\}\/pull-requests`\)/);
  assert.doesNotMatch(appSource, /api\(`\/api\/projects\/\$\{encodeURIComponent\(targetProjectID\)\}\/pulls`\)/);
  assert.match(appSource, /const taskRows = Array\.isArray\(payload\.tasks\) \? payload\.tasks : \[\];/);
  assert.match(appSource, /taskRows\.flatMap\(\(task\) =>/);
  assert.match(appSource, /const pullRows = Array\.isArray\(task\.pull_requests\) \? task\.pull_requests : \[\];/);
  assert.match(appSource, /task_id: pull\.task_id \|\| task\.task_id \|\| task\.id \|\| ''/);
  assert.match(appSource, /dashboardPullRequests\.value = normalizeDashboardPullRequestsPayload\(\{/);
});

test('ledger logs exposes compact proof timeline coverage', async () => {
  const appSource = await fs.readFile(new URL('./src/App.vue', import.meta.url), 'utf-8');
  const cssSource = await fs.readFile(new URL('./src/styles.css', import.meta.url), 'utf-8');

  assert.match(appSource, /class="ledger-proof-timeline"/);
  assert.match(appSource, /class="ledger-verify-card"/);
  assert.match(appSource, /<h1>Ledger Logs<\/h1>/);
  assert.match(appSource, /Transparent platform activity from the live ledger\. Payments, token mints, PR handoffs, deployment gates, AI actions, and payouts are loaded from the backend\./);
  assert.match(appSource, /Public ledger'[\s\S]*Payouts, escrow events, AI actions, releases, deployment checks, and proof logs for trust/);
  assert.match(appSource, /Ledger events'[\s\S]*Fetch sanitized escrow, payout, token, PR, deployment, and release proof rows/);
  assert.match(appSource, /Latest escrow, PR, AI, and release evidence/);
  assert.match(appSource, /const ledgerVerification = ref\(null\);/);
  assert.match(appSource, /const ledgerVerificationSummary = computed/);
  assert.match(appSource, /publicApi\('\/api\/public\/ledger\/verify'\)/);
  assert.match(appSource, /copyLedgerVerifyPacket/);
  assert.match(appSource, /const ledgerProofTimelineRows = computed/);
  assert.match(appSource, /ledgerProofLanes\.value/);
  assert.match(appSource, /key: 'escrow-proof'[\s\S]*title: 'Escrow funding'[\s\S]*Payment verification, project reserve, treasury movement, and escrow lock records/);
  assert.match(appSource, /key: 'pr-proof'[\s\S]*title: 'PR handoff'[\s\S]*Submitted reviews, accepted PRs, task claims, and repository workflow events/);
  assert.match(appSource, /key: 'ai-proof'[\s\S]*title: 'AI audit'[\s\S]*AI review webhooks and agent action packets tied to routed software work/);
  assert.match(appSource, /key: 'release-proof'[\s\S]*title: 'Release proof'[\s\S]*Payout releases, auto-release policy evidence, task payments, and manual credits/);
  assert.match(appSource, /ledgerTabs = \['All Activity', 'Escrow & Payments', 'Tasks & PRs', 'Milestones', 'AI Actions', 'Token Events'\]/);
  assert.match(appSource, /if \(normalized === 'ai_review'\)/);
  assert.match(appSource, /if \(normalized === 'agent_action'\)/);
  assert.match(appSource, /if \(normalized === 'ledger_task_payment'\) return 'release proof'/);
  assert.match(appSource, /mapLedgerTransparencyEvent\(latest\)/);
  assert.match(appSource, /applyLedgerProofLane\(row\.lane\)/);
  assert.match(cssSource, /\.ledger-proof-timeline-list\s*\{[\s\S]*grid-template-columns: repeat\(4, minmax\(0, 1fr\)\);/);
  assert.match(cssSource, /\.ledger-verify-grid\s*\{[\s\S]*grid-template-columns: repeat\(3, minmax\(0, 1fr\)\);/);
  assert.match(cssSource, /@media \(max-width: 560px\)[\s\S]*\.ledger-proof-timeline-list\s*\{[\s\S]*grid-template-columns: minmax\(0, 1fr\);/);
});

test('agent work packets expose authenticated lease action', async () => {
  const appSource = await fs.readFile(new URL('./src/App.vue', import.meta.url), 'utf-8');
  const cssSource = await fs.readFile(new URL('./src/styles.css', import.meta.url), 'utf-8');

  assert.match(appSource, /const leasingAgentTaskID = ref\(''\);/);
  assert.match(appSource, /const agentLeaseResponses = reactive\(\{\}\);/);
  assert.match(appSource, /@click="leaseAgentWorkPacket\(packet\)"/);
  assert.match(appSource, /@click="leaseAgentWorkPacket\(task\)"/);
  assert.match(appSource, /Lease packet/);
  assert.match(appSource, /async function leaseAgentWorkPacket\(task = \{\}\)/);
  assert.match(appSource, /api\('\/api\/agent-queue\/leases', \{/);
  assert.match(appSource, /status: existingLease\.lease_id \? 'heartbeat' : 'leased'/);
  assert.match(appSource, /agentLeaseResponses\[claimID\] = lease;/);
  assert.match(cssSource, /\.marketplace-agent-lease-row\s*\{[\s\S]*grid-template-columns: minmax\(128px, auto\) minmax\(0, 1fr\);/);
});

test('repo import exposes publish path to bounties, agents, and live proof', async () => {
  const appSource = await fs.readFile(new URL('./src/App.vue', import.meta.url), 'utf-8');

  assert.match(appSource, /class="repo-import-publish-plan"/);
  assert.match(appSource, /const repoImportPublishPlanRows = computed\(\(\) => \{/);
  assert.match(appSource, /const repoImportPublishPlanSummary = computed\(\(\) => \{/);
  assert.match(appSource, /@click="openImportedRepoPublishPreview\('bounties'\)"/);
  assert.match(appSource, /@click="openImportedRepoPublishPreview\('agents'\)"/);
  assert.match(appSource, /@click="openImportedRepoLiveProof"/);
  assert.match(appSource, /function openImportedRepoPublishPreview\(target = 'bounties'\)/);
  assert.match(appSource, /projectSetupForm\.allowAgents = true;/);
  assert.match(appSource, /openDashboardSection\(isAgentPreview \? 'agents' : 'bounties'\)/);
  assert.match(appSource, /openMarketplaceSection\(isAgentPreview \? 'marketplace-agent-packets' : 'marketplace-bounties'\)/);
  assert.match(appSource, /activeLiveFeedType\.value = 'Repository Scan';/);
});

test('repo scan suggested tasks expose routing packets for funded work', async () => {
  const appSource = await fs.readFile(new URL('./src/App.vue', import.meta.url), 'utf-8');
  const scanSchema = JSON.parse(await fs.readFile(new URL('./public/protocol/scan.v1.schema.json', import.meta.url), 'utf-8'));
  const suggestedTask = scanSchema.properties.suggested_tasks.items;

  assert.ok(suggestedTask.required.includes('routing_packet'));
  assert.equal(suggestedTask.properties.routing_packet.$ref, '#/$defs/routingPacket');
  assert.deepEqual(scanSchema.$defs.routingPacket.required, ['action', 'method', 'endpoint', 'context_urls', 'runbook']);
  assert.match(appSource, /const routingPacket = row\.routing_packet && typeof row\.routing_packet === 'object' \? row\.routing_packet : \{\};/);
  assert.match(appSource, /routingAction: routingActionLabel\(routingPacket\.action \|\| ''\)/);
  assert.match(appSource, /routingContract: Array\.isArray\(routingPacket\.output_contracts\)/);
});

test('marketplace proposal packets expose output contracts for contributors', async () => {
  const appSource = await fs.readFile(new URL('./src/App.vue', import.meta.url), 'utf-8');
  const marketplaceSchema = JSON.parse(await fs.readFile(new URL('./public/protocol/marketplace.v1.schema.json', import.meta.url), 'utf-8'));
  const proposalPacket = marketplaceSchema.properties.bounties.items.properties.proposal_packet;

  assert.ok(marketplaceSchema.properties.bounties.items.required.includes('claim_endpoint'));
  assert.equal(marketplaceSchema.properties.bounties.items.properties.claim_endpoint.maxLength, 240);
  assert.match(appSource, /claimEndpoint: bounty\.claim_endpoint \|\| `\/api\/tasks\/\$\{claimID\}\/claim`/);
  assert.equal(proposalPacket.properties.output_contracts.items.$ref, '#/$defs/outputContract');
  assert.ok(marketplaceSchema.$defs.outputContract.required.includes('output_protocol_url'));
  assert.match(appSource, /function workerProposalContractRows\(contracts = \[\]\)/);
  assert.match(appSource, /contractRows: workerProposalContractRows\(packet\.output_contracts\)/);
  assert.match(appSource, /proposal\.evidenceRows\.length \|\| proposal\.payloadRows\.length \|\| proposal\.contractRows\.length/);
});

test('marketplace page exposes all operating lanes at a glance', async () => {
  const appSource = await fs.readFile(new URL('./src/App.vue', import.meta.url), 'utf-8');
  const cssSource = await fs.readFile(new URL('./src/styles.css', import.meta.url), 'utf-8');

  assert.match(appSource, /class="marketplace-os-strip"/);
  assert.match(appSource, /const marketplaceOperatingRows = computed/);
  assert.match(appSource, /Marketplace is the realtime work economy for MergeOS: live projects, public bounties, contributors, and AI work queues backed by the platform ledger\./);
  assert.match(appSource, /Marketplace la realtime work economy cua MergeOS: live projects, public bounties, contributors va AI work queues duoc bao chung boi platform ledger\./);
  assert.match(appSource, /description: 'List live projects, public bounties, contributors, and AI agents\.'/);
  assert.match(appSource, /label: 'Live projects'/);
  assert.match(appSource, /label: 'Public bounties'/);
  assert.match(appSource, /label: 'Contributors'/);
  assert.match(appSource, /label: 'AI agents'/);
  assert.match(appSource, /caption: `\$\{packetCount\} executable work packets`/);
  assert.match(appSource, /openMarketplaceSection\('marketplace-agent/);
  assert.match(appSource, /if \(payload\.type === 'agent_queue'\)/);
  assert.match(appSource, /if \(payload\.type === 'agent_presence'\)/);
  assert.match(appSource, /if \(payload\.type === 'agent_claim'\)/);
  assert.match(appSource, /if \(payload\.type === 'agent_submit'\)/);
  assert.match(appSource, /if \(payload\.type === 'agent_release'\)/);
  assert.match(appSource, /hydrateAgentQueueData\(queue\)/);
  assert.match(cssSource, /\.marketplace-os-strip\s*\{[\s\S]*grid-template-columns: repeat\(4, minmax\(0, 1fr\)\);/);
  assert.match(cssSource, /@media \(max-width: 760px\)[\s\S]*\.marketplace-os-strip\s*\{[\s\S]*grid-template-columns: minmax\(0, 1fr\);/);
});

test('marketplace contributors expose routeable delivery disciplines', async () => {
  const appSource = await fs.readFile(new URL('./src/App.vue', import.meta.url), 'utf-8');
  const cssSource = await fs.readFile(new URL('./src/styles.css', import.meta.url), 'utf-8');
  const marketplaceSchema = JSON.parse(await fs.readFile(new URL('./public/protocol/marketplace.v1.schema.json', import.meta.url), 'utf-8'));
  const contributorProperties = marketplaceSchema.properties.contributors.items.properties;

  assert.deepEqual(contributorProperties.disciplines.items.enum, ['frontend', 'backend', 'design', 'qa', 'devops', 'security']);
  assert.equal(contributorProperties.primary_discipline.type, 'string');
  assert.match(appSource, /function marketplaceContributorDisciplineLabels\(contributor = \{\}\)/);
  assert.match(appSource, /contributor\.primary_discipline/);
  assert.match(appSource, /Array\.isArray\(contributor\.disciplines\)/);
  assert.match(appSource, /disciplineLabel: disciplineLabels\.join\(' \/ '\)/);
  assert.match(appSource, /class="marketplace-contributor-disciplines"/);
  assert.match(appSource, /Frontend, backend, design, QA, DevOps, and security contributors/);
  assert.match(cssSource, /\.marketplace-contributor-disciplines\s*\{[\s\S]*text-overflow: ellipsis;[\s\S]*white-space: nowrap;/);
});

test('marketplace AI agent matrix covers all AI agent lanes', async () => {
  const appSource = await fs.readFile(new URL('./src/App.vue', import.meta.url), 'utf-8');
  const cssSource = await fs.readFile(new URL('./src/styles.css', import.meta.url), 'utf-8');

  assert.match(appSource, /class="marketplace-agent-matrix"/);
  assert.match(appSource, /const marketplaceAgentCapabilityMatrix = computed/);
  assert.match(appSource, /const marketplaceAgentCapabilityDefinitions = \[[\s\S]*key: 'generate'[\s\S]*key: 'code'[\s\S]*key: 'review'[\s\S]*key: 'test'[\s\S]*key: 'secure'[\s\S]*key: 'deploy'/);
  assert.match(appSource, /key: 'generate'[\s\S]*title: 'Generate task graph'[\s\S]*output: 'Task packets, rewards, lanes'/);
  assert.match(appSource, /key: 'code'[\s\S]*title: 'Code implementation'[\s\S]*evidence: 'PR URL and commit refs'/);
  assert.match(appSource, /key: 'review'[\s\S]*title: 'Review pull requests'[\s\S]*evidence: 'Review webhook record'/);
  assert.match(appSource, /key: 'test'[\s\S]*title: 'Test and QA'[\s\S]*evidence: 'Test log and screenshot'/);
  assert.match(appSource, /key: 'secure'[\s\S]*title: 'Security validation'[\s\S]*evidence: 'Audit note and findings'/);
  assert.match(appSource, /key: 'deploy'[\s\S]*title: 'Deployment gate'[\s\S]*evidence: 'Deployment proof row'/);
  assert.match(appSource, /Review, test, generate, code, secure, and deploy with proof/);
  assert.match(appSource, /AI agents can review pull requests, test builds, generate task graphs, code scoped fixes, validate security, and gate deployments with proof\./);
  assert.match(cssSource, /\.marketplace-agent-matrix-grid\s*\{[\s\S]*grid-template-columns: repeat\(2, minmax\(0, 1fr\)\);/);
});

test('auto-release exposes payout output contracts in schema and dashboard', async () => {
  const appSource = await fs.readFile(new URL('./src/App.vue', import.meta.url), 'utf-8');
  const releaseSchema = JSON.parse(await fs.readFile(new URL('./public/protocol/payout-release.v1.schema.json', import.meta.url), 'utf-8'));

  assert.ok(releaseSchema.required.includes('output_contracts'));
  assert.equal(releaseSchema.properties.output_contracts.items.$ref, '#/$defs/outputContract');
  assert.ok(releaseSchema.$defs.outputContract.required.includes('output_protocol_url'));
  assert.ok(releaseSchema.properties.release_proofs.items.required.includes('ledger_proof_url'));
  assert.equal(releaseSchema.properties.release_proofs.items.properties.ledger_proof_url.maxLength, 512);
  assert.match(appSource, /contractRows: autoReleaseContractRows\(packet\.output_contracts \|\| \[\]\)/);
  assert.match(appSource, /function autoReleaseContractRows\(contracts = \[\]\)/);
  assert.match(appSource, /v-if="dashboardAutoReleaseControl\.contractRows\.length"/);
});

test('public mega menu keeps a large hover bridge to its fixed panel', async () => {
  const cssSource = await fs.readFile(new URL('./src/styles.css', import.meta.url), 'utf-8');

  assert.match(cssSource, /\.nav-menu\.open::after\s*\{[\s\S]*left: -72px;[\s\S]*right: -72px;[\s\S]*height: 108px;/);
  assert.match(cssSource, /\.nav-context-menu::before\s*\{[\s\S]*left: -36px;[\s\S]*right: -36px;[\s\S]*top: -108px;[\s\S]*height: 108px;/);
  assert.match(cssSource, /@media \(max-width: 700px\)[\s\S]*\.nav-context-menu\s*\{[\s\S]*overflow-y: auto;[\s\S]*overscroll-behavior: contain;/);
  assert.match(cssSource, /\.public-nav-actions \.locale-context-menu,[\s\S]*\.project-flow-actions \.locale-context-menu\s*\{[\s\S]*position: fixed;[\s\S]*max-height: min\(72dvh, 420px\);[\s\S]*overflow-y: auto;/);
});

test('public agents page exposes CEO orchestrator and subagent delegation model', async () => {
  const appSource = await fs.readFile(new URL('./src/App.vue', import.meta.url), 'utf-8');
  const seoSource = await fs.readFile(new URL('./src/seo.js', import.meta.url), 'utf-8');

  assert.match(appSource, /class="public-agent-chief-node"/);
  assert.match(appSource, /const publicAgentChiefNode = computed\(\(\) => \{/);
  assert.match(appSource, /const publicAgentSubagentRows = computed\(\(\) => \{/);
  assert.match(appSource, /CEO ORCHESTRATOR/);
  assert.match(appSource, /Start CEO brief/);
  assert.match(appSource, /Design subagent/);
  assert.match(appSource, /Coding subagent/);
  assert.match(appSource, /Security subagent/);
  assert.match(seoSource, /CEO AI orchestrator/);
  assert.match(seoSource, /Subagent delegation/);
  assert.match(seoSource, /CEO orchestrator/);
});

test('public backend page exposes the proposed runtime stack', async () => {
  const appSource = await fs.readFile(new URL('./src/App.vue', import.meta.url), 'utf-8');
  const cssSource = await fs.readFile(new URL('./src/styles.css', import.meta.url), 'utf-8');

  assert.match(appSource, /class="public-backend-stack-strip"/);
  assert.match(appSource, /const publicBackendRuntimeStackRows = computed/);
  assert.match(appSource, /const publicBackendSurfaceRows = computed\(\(\) => \{/);
  assert.match(appSource, /key: 'auth'[\s\S]*key: 'repo'[\s\S]*key: 'ai'[\s\S]*key: 'tasks'[\s\S]*key: 'payments'[\s\S]*key: 'realtime'[\s\S]*key: 'ledger'[\s\S]*key: 'protocol'/);
  assert.match(appSource, /Auth, repositories, AI orchestration, escrow, realtime, and ledger APIs in one backend loop/);
  assert.match(appSource, /authentication, repository imports, AI scans, task generation, payment verification, escrow reserves, live notifications, public protocol documents, and ledger proof/);
  assert.match(appSource, /Authentication and sessions[\s\S]*OAuth, wallet, role, password reset, and dashboard access/);
  assert.match(appSource, /Repository import[\s\S]*GitHub issues, source context, dependencies, and technical debt markers/);
  assert.match(appSource, /AI orchestration[\s\S]*Issue scans, task graph generation, reward estimates, routing, and agent packets/);
  assert.match(appSource, /Task engine[\s\S]*Create, claim, submit, review, accept, dispute, auto-release, and payout commands/);
  assert.match(appSource, /Payment and escrow[\s\S]*Card, PayPal, crypto, project reserve, task reserve, platform fee, and payout release/);
  assert.match(appSource, /Realtime gateway[\s\S]*WebSocket snapshots stream marketplace, project, PR, deployment, agent, payout, and admin state changes/);
  assert.match(appSource, /Ledger proof[\s\S]*Sanitized payment, token mint, escrow, PR, deployment, release, and contract references/);
  assert.match(appSource, /Authenticate actor[\s\S]*Import repository[\s\S]*Run AI analysis[\s\S]*Generate task graph[\s\S]*Verify funding[\s\S]*Route work live[\s\S]*Review and release[\s\S]*Publish proof/);
  for (const label of ['Go / Rust', 'PostgreSQL', 'Redis', 'GitHub API', 'OpenAI API', 'WebSocket gateway']) {
    assert.match(appSource, new RegExp(`label: '${label.replace('/', '\\/')}'`));
  }
  assert.match(cssSource, /\.public-backend-stack-strip\s*\{[\s\S]*grid-template-columns: repeat\(6, minmax\(0, 1fr\)\);/);
  assert.match(cssSource, /@media \(max-width: 980px\)[\s\S]*\.public-backend-stack-strip\s*\{[\s\S]*grid-template-columns: repeat\(3, minmax\(0, 1fr\)\);/);
  assert.match(cssSource, /@media \(max-width: 520px\)[\s\S]*\.public-backend-stack-strip\s*\{[\s\S]*grid-template-columns: repeat\(2, minmax\(0, 1fr\)\);/);
});

test('public agents page exposes AI layer capability checklist', async () => {
  const appSource = await fs.readFile(new URL('./src/App.vue', import.meta.url), 'utf-8');
  const cssSource = await fs.readFile(new URL('./src/styles.css', import.meta.url), 'utf-8');

  assert.match(appSource, /class="public-agent-capability-strip"/);
  assert.match(appSource, /class="public-agent-action-contracts"/);
  assert.match(appSource, /const publicAgentCapabilityRows = computed/);
  assert.match(appSource, /const publicAgentActionContractRows = computed/);
  assert.match(appSource, /label: 'Scan repository'/);
  assert.match(appSource, /Detect bugs, technical debt, and dependencies/);
  assert.match(appSource, /label: 'Analyze issues'/);
  assert.match(appSource, /Estimate complexity, time, and budget/);
  assert.match(appSource, /label: 'Generate tasks'/);
  assert.match(appSource, /Create task graph and assign worker type/);
  assert.match(appSource, /label: 'Review PRs'/);
  assert.match(appSource, /Code review, security review, and deployment validation/);
  assert.match(appSource, /AI scans source repositories, imported issues, dependencies, technical debt markers, secrets risk, and bug candidates before task creation\./);
  assert.match(appSource, /Models estimate complexity, time, budget, security exposure, test gaps, and deployment constraints for each issue\./);
  assert.match(appSource, /The task engine converts analysis into scoped work packets with acceptance criteria, evidence requirements, dependencies, and suggested lane\./);
  assert.match(appSource, /Create scoped task packets, reward estimates, worker kind, suggested agent type, and dependency order\./);
  assert.match(appSource, /Review agents inspect patches for correctness, regressions, acceptance criteria coverage, risk notes, and release readiness\./);
  assert.match(appSource, /action: 'scan'[\s\S]*label: 'Scan agent'[\s\S]*proof: '\/api\/public\/projects\/\{id\}\/repo-scan'/);
  assert.match(appSource, /action: 'generate'[\s\S]*label: 'Coding agent'[\s\S]*outputProtocol: 'mergeos\.agent-action\.v1'/);
  assert.match(appSource, /action: 'review'[\s\S]*label: 'Review agent'[\s\S]*proof: '\/api\/public\/projects\/\{id\}\/pull-requests'/);
  assert.match(appSource, /action: 'test'[\s\S]*label: 'QA agent'[\s\S]*proof: '\/api\/public\/projects\/\{id\}\/ai-workflow'/);
  assert.match(appSource, /action: 'deploy'[\s\S]*label: 'Deploy agent'[\s\S]*outputProtocol: 'mergeos\.deployment\.v1'/);
  assert.match(appSource, /copyPublicAgentActionContract/);
  assert.match(appSource, /Review, QA, security, DevOps, customer approval, and payout release can all be tracked from SDK consumers\./);
  assert.match(cssSource, /\.public-agent-capability-strip\s*\{[\s\S]*grid-template-columns: repeat\(4, minmax\(0, 1fr\)\);/);
  assert.match(cssSource, /\.public-agent-action-contracts\s*\{[\s\S]*grid-template-columns: repeat\(5, minmax\(0, 1fr\)\);/);
  assert.match(cssSource, /@media \(max-width: 980px\)[\s\S]*\.public-agent-capability-strip\s*\{[\s\S]*grid-template-columns: repeat\(2, minmax\(0, 1fr\)\);/);
  assert.match(cssSource, /@media \(max-width: 980px\)[\s\S]*\.public-agent-action-contracts\s*\{[\s\S]*grid-template-columns: repeat\(2, minmax\(0, 1fr\)\);/);
  assert.match(cssSource, /@media \(max-width: 520px\)[\s\S]*\.public-agent-capability-strip\s*\{[\s\S]*grid-template-columns: 1fr;/);
  assert.match(cssSource, /@media \(max-width: 520px\)[\s\S]*\.public-agent-action-contracts\s*\{[\s\S]*grid-template-columns: 1fr;/);
});

test('public token pages expose airdrop, presale, and whitepaper routes', async () => {
  const appSource = await fs.readFile(new URL('./src/App.vue', import.meta.url), 'utf-8');
  const cssSource = await fs.readFile(new URL('./src/styles.css', import.meta.url), 'utf-8');
  const seoSource = await fs.readFile(new URL('./src/seo.js', import.meta.url), 'utf-8');
  const whitepaperSource = await fs.readFile(new URL('./public/whitepaper/mergeos-whitepaper.md', import.meta.url), 'utf-8');
  const tokenLaunchCandidatesSchema = JSON.parse(await fs.readFile(new URL('./public/protocol/token-launch-candidates.v1.schema.json', import.meta.url), 'utf-8'));

  for (const page of ['airdrop', 'presale', 'whitepaper']) {
    assert.match(appSource, new RegExp(`${page}: '/${page}'`));
    assert.match(seoSource, new RegExp(`${page}: '/${page}'`));
  }
  assert.match(appSource, /v-else-if="publicTokenPage"/);
  assert.match(appSource, /const publicTokenPageDefinitions = \{/);
  assert.match(appSource, /title: 'A task-based airdrop for verified software delivery\.'/);
  assert.match(appSource, /title: 'Reserve MRG through a transparent presale workflow\.'/);
  assert.match(appSource, /title: 'The operating system for AI software delivery\.'/);
  assert.match(appSource, /aria-label="Refresh token proof data"/);
  assert.match(appSource, /action: \{ page: 'airdrop' \}/);
  assert.match(appSource, /action: \{ page: 'presale' \}/);
  assert.match(appSource, /action: \{ page: 'whitepaper' \}/);
  assert.match(appSource, /const whitepaperDownloadPath = '\/whitepaper\/mergeos-whitepaper\.md'/);
  assert.match(appSource, /command: 'download-whitepaper'/);
  assert.match(appSource, /function downloadWhitepaper\(\)/);
  assert.match(appSource, /class="token-whitepaper-thesis"/);
  assert.match(appSource, /class="token-whitepaper-brief"/);
  assert.match(appSource, /class="token-whitepaper-section-list"/);
  assert.match(appSource, /v-if="publicPage !== 'whitepaper'" class="token-content-grid"/);
  assert.match(appSource, /class="token-hero-ceo-strip"/);
  assert.match(appSource, /CEO launch research summary/);
  assert.match(appSource, /v-for="row in tokenCeoQueueStatRows"/);
  assert.ok(appSource.indexOf('class="token-hero-ceo-strip"') < appSource.indexOf('class="token-ceo-research-panel"'));
  assert.match(appSource, /class="wizard-field token-compact-half"/);
  assert.match(appSource, /const publicWhitepaperThesisRows = computed\(\(\) => \[/);
  assert.match(appSource, /const publicWhitepaperChapterSections = computed\(\(\) => \[/);
  assert.match(appSource, /The paper is structured around executable product proof/);
  assert.match(appSource, /id="token-workflow"/);
  assert.match(appSource, /class="token-ceo-research-panel"/);
  assert.match(appSource, /CEO LAUNCH DECISION/);
  assert.match(appSource, /v-if="action\.command === 'token-launch-brief'"/);
  assert.match(appSource, /href="#token-ceo-brief"/);
  assert.match(appSource, /@click="handlePublicAction\(action\)"/);
  assert.match(appSource, /label: 'Send CEO brief', primary: true, icon: ArrowRight, command: 'token-launch-brief'/);
  assert.match(appSource, /label: 'Send CEO brief', primary: true, icon: UserCheck, command: 'token-launch-brief'/);
  assert.match(appSource, /Queue API/);
  assert.match(appSource, /class="token-ceo-decision-strip"/);
  assert.match(appSource, /class="token-ceo-queue-stats"/);
  assert.match(appSource, /class="token-ceo-live-queue"/);
  assert.match(appSource, /class="token-ceo-live-context" role="group"/);
  assert.match(appSource, /function tokenCeoLiveContextRows\(brief = \{\}, launchType = 'airdrop'\)/);
  assert.match(appSource, /brief\.project_summary \|\| gate/);
  assert.match(appSource, /class="token-ceo-live-empty"/);
  assert.match(appSource, /class="token-ceo-candidate-lane"/);
  assert.match(appSource, /class="token-ceo-candidate-empty"/);
  assert.match(appSource, /class="token-ceo-empty-gates"/);
  assert.match(appSource, /const tokenCeoCandidateEmptyCopy = computed/);
  assert.match(appSource, /const tokenLaunchCandidatesLoading = ref\(false\)/);
  assert.match(appSource, /const tokenLaunchCandidatesError = ref\(''\)/);
  assert.match(appSource, /Candidate API unavailable/);
  assert.match(appSource, /Open CEO brief/);
  assert.match(appSource, /Review data/);
  assert.match(appSource, /checks: \['Mission demand', 'Anti-bot gate', 'Ledger proof'\]/);
  assert.match(appSource, /checks: \['Utility score', 'Funding proof', 'Wallet gate'\]/);
  assert.match(appSource, /class="token-ceo-candidate-policy"/);
  assert.match(appSource, /class="token-ceo-candidate-readiness"/);
  assert.match(appSource, /token-ceo-candidate-verdict/);
  assert.match(appSource, /function tokenLaunchCandidateVerdict/);
  assert.match(appSource, /Hold \$\{launchLabel\}/);
  assert.match(appSource, /Review \$\{launchLabel\}/);
  assert.match(appSource, /Ready \$\{launchLabel\}/);
  assert.match(appSource, /row\.decisionPreview\.nextAction/);
  assert.match(appSource, /nextAction: String\(nextAction \|\| ''\)\.trim\(\)/);
  assert.match(appSource, /candidate\.next_action/);
  assert.match(appSource, /candidate\.readiness_gates/);
  assert.match(appSource, /class="token-ceo-project-queue"/);
  assert.match(appSource, /class="token-ceo-source-packet"/);
  assert.match(appSource, /class="token-ceo-signal-chips"/);
  assert.match(appSource, /class="token-ceo-brief-gates"/);
  assert.match(appSource, /tokenCeoLaunchBriefCopy\.quickGates/);
  assert.match(appSource, /class="token-ceo-launch-context"/);
  assert.match(appSource, /CEO launch brief context/);
  assert.match(appSource, /tokenCeoLaunchBriefCopy\.launchTypeLabel/);
  assert.match(appSource, /tokenCeoLaunchBriefCopy\.ledgerMemo/);
  assert.match(appSource, /class="token-ceo-decision-context"/);
  assert.match(appSource, /id="token-ceo-brief" class="token-ceo-brief-card"/);
  assert.match(appSource, /@submit\.prevent="submitTokenLaunchBrief"/);
  assert.match(appSource, /class="token-ceo-memo-result"/);
  assert.match(appSource, /class="token-ceo-memo-summary"/);
  assert.match(appSource, /const tokenLaunchBriefMemoSummaryRows = computed/);
  assert.match(appSource, /Ready gates/);
  assert.match(appSource, /Review gates/);
  assert.match(appSource, /Hold gates/);
  assert.match(appSource, /tokenLaunchBriefResult\.ceo_memo\.decision_label/);
  assert.match(appSource, /tokenLaunchBriefResult\.ceo_memo\.gates\?\.length/);
  assert.match(appSource, /class="token-ceo-memo-source"/);
  assert.match(appSource, /tokenLaunchBriefResult\.repository_url/);
  assert.match(appSource, /Research source/);
  assert.match(appSource, /Ledger board/);
  assert.match(appSource, /tokenLaunchBriefResult[\s\S]{0,900}openPublicPage\('ledger'\)/);
  assert.match(appSource, /action\.command === 'token-ceo-brief'/);
  assert.match(appSource, /action\.command === 'token-launch-brief'/);
  assert.match(appSource, /if \(action\.command === 'token-launch-brief'\) \{[\s\S]*prefillTokenLaunchBrief\(\);[\s\S]*nextTick\(\)\.then\(\(\) => scrollTokenLaunchBriefCardIntoView\(\)\);[\s\S]*window\.setTimeout\(scrollTokenLaunchBriefCardIntoView, 220\);[\s\S]*return;/);
  assert.ok(appSource.indexOf('class="token-ceo-research-panel"') < appSource.indexOf('class="token-content-grid"'));
  assert.ok(appSource.indexOf('class="token-ceo-decision-strip"') < appSource.indexOf('class="token-ceo-brief-card"'));
  assert.ok(appSource.indexOf('class="token-ceo-candidate-lane"') < appSource.indexOf('class="token-ceo-live-queue"'));
  assert.ok(appSource.indexOf('class="token-ceo-project-queue"') < appSource.indexOf('class="token-ceo-brief-card"'));
  assert.ok(appSource.indexOf('class="token-ceo-research-grid"') < appSource.indexOf('class="token-ceo-brief-card"'));
  assert.match(appSource, /CEO airdrop readiness review\./);
  assert.match(appSource, /CEO presale readiness review\./);
  assert.match(appSource, /const tokenCeoQueueURL = computed/);
  assert.match(appSource, /\/api\/public\/token\/launch-briefs\?launch_type=\$\{publicPage\.value === 'presale' \? 'presale' : 'airdrop'\}/);
  assert.match(appSource, /const tokenCeoCandidatesURL = computed/);
  assert.match(appSource, /function tokenLaunchCandidateAPIPath\(launchType = ''\)/);
  assert.match(appSource, /\/api\/public\/token\/launch-candidates\?launch_type=\$\{type === 'presale' \? 'presale' : 'airdrop'\}/);
  assert.match(appSource, /async function loadTokenLaunchCandidates\(launchType = ''\)/);
  assert.match(appSource, /tokenLaunchCandidatesLoading\.value = true/);
  assert.match(appSource, /tokenLaunchCandidatesError\.value = error\?\.message \|\| 'CEO candidate queue is temporarily unavailable\.'/);
  assert.match(appSource, /tokenLaunchCandidatesLoading\.value = false/);
  assert.match(appSource, /if \(page === 'airdrop' \|\| page === 'presale'\) void loadTokenLaunchCandidates\(page\)/);
  assert.match(appSource, /const candidatePath = publicPage\.value === 'airdrop' \|\| publicPage\.value === 'presale'/);
  assert.match(appSource, /const tokenCeoDecisionRows = computed/);
  assert.match(appSource, /const tokenCeoQueueStatRows = computed/);
  assert.match(appSource, /Ready to open/);
  assert.match(appSource, /candidateStats\.ready_count/);
  assert.match(appSource, /const tokenCeoLiveQueueRows = computed/);
  assert.match(appSource, /const tokenCeoLiveEmptyCopy = computed/);
  assert.match(appSource, /const tokenCeoCandidateRows = computed/);
  assert.match(appSource, /const tokenLaunchCandidatesData = ref/);
  assert.match(appSource, /publicApi\(tokenLaunchCandidateAPIPath\(launchType\)\)/);
  assert.match(appSource, /publicApi\(candidatePath\)/);
  assert.match(appSource, /tokenLaunchCandidatesData\.value\?\.candidates/);
  assert.match(appSource, /candidate\.research_source/);
  assert.match(appSource, /candidate\.proof_policy/);
  assert.match(appSource, /class="token-ceo-candidate-context" role="group"/);
  assert.match(appSource, /function tokenLaunchCandidateContextRows\(candidate = \{\}, readinessRows = \[\], launchType = 'airdrop'\)/);
  assert.match(appSource, /CEO brief/);
  assert.match(appSource, /Policy gates/);
  assert.match(appSource, /Proof gates/);
  assert.match(appSource, /row\.proofSignalRows/);
  assert.match(appSource, /function tokenLaunchCandidateScore/);
  assert.match(appSource, /function tokenLaunchCandidateDecisionRows/);
  assert.match(appSource, /function tokenLaunchCandidateDecisionRowsFromAPI\(rows = \[\], launchType = 'airdrop', score = 0\)/);
  assert.match(appSource, /const fallbackRows = tokenLaunchCandidateDecisionRows\(launchType, score\)/);
  assert.match(appSource, /contradictsLaunch/);
  assert.match(appSource, /proofPolicy: contradictsLaunch \? fallback\.proofPolicy/);
  assert.match(appSource, /label: fallback\.label \|\| row\.label/);
  assert.match(appSource, /function tokenLaunchCandidateDecisionPreview\(rows = \[\], nextAction = ''\)/);
  assert.match(appSource, /function tokenLaunchCandidateReadinessRows/);
  assert.match(appSource, /function tokenLaunchCandidateReadinessRowsFromAPI\(rows = \[\], fallback = \[\]\)/);
  assert.match(appSource, /tokenLaunchCandidateReadinessRowsFromAPI\(candidate\.readiness_gates, fallbackReadinessRows\)/);
  assert.match(appSource, /label: 'Demand', value: `\$\{openCount\} open \/ \$\{acceptedCount\} accepted`/);
  assert.match(appSource, /label: 'Reserve', value: `\$\{formatCompactNumber\(pool\)\} MRG pool`/);
  assert.match(appSource, /const readinessRows = tokenLaunchCandidateReadinessRows/);
  assert.match(appSource, /readinessRows,/);
  assert.match(appSource, /Number\(candidate\.research_score\) \|\| tokenLaunchCandidateScore/);
  assert.match(appSource, /tokenLaunchCandidateDecisionRowsFromAPI\(candidate\.decision_options, launchType, score\)/);
  assert.match(appSource, /decisionPreview: tokenLaunchCandidateDecisionPreview\(decisionRows, candidate\.next_action\)/);
  assert.match(appSource, /function applyTokenLaunchCandidateDecision\(candidate = \{\}, decision = \{\}\)/);
  assert.match(appSource, /row\.scoreLabel/);
  assert.match(appSource, /scoreLabel: `\$\{score\}% fit`/);
  assert.match(appSource, /class="token-ceo-candidate-decisions"/);
  assert.match(appSource, /class="token-ceo-candidate-signals"/);
  assert.match(appSource, /proofSignalRows = signals\.slice\(0, 3\)\.map/);
  assert.match(appSource, /proofSignalExtra: Math\.max\(0, signals\.length - proofSignalRows\.length\)/);
  assert.match(appSource, /\['ai', 'api', 'ceo', 'dao', 'idl', 'mrg', 'pr', 'qa', 'sdk', 'ui', 'url', 'ux', 'go'\]\.includes\(lower\)/);
  assert.match(appSource, /Open presale/);
  assert.match(appSource, /Open missions/);
  assert.match(appSource, /Need proof/);
  assert.match(appSource, /Hold launch/);
  assert.match(appSource, /CEO \$\{launchLabel\} decision: request more evidence before opening/);
  assert.match(appSource, /class="token-ceo-candidate-actions"/);
  assert.match(appSource, /marketplaceData\.value\.projects[\s\S]*marketplaceData\.value\.bounties/);
  assert.match(appSource, /Use for CEO brief/);
  assert.match(appSource, /function prefillTokenLaunchBriefFromCandidate\(candidate = \{\}\)/);
  assert.match(appSource, /function scrollTokenLaunchBriefCardIntoView\(\)/);
  assert.match(appSource, /document\.getElementById\('token-ceo-brief'\) \|\| document\.querySelector\('\.token-ceo-brief-card'\)/);
  assert.match(appSource, /const navHeight = Math\.round\(document\.querySelector\('\.home-navbar'\)\?\.getBoundingClientRect\(\)\.height \|\| 64\)/);
  assert.match(appSource, /window\.scrollTo\(\{ top, behavior \}\)/);
  assert.match(appSource, /if \(behavior === 'auto'\) window\.scrollTo\(0, top\)/);
  assert.match(appSource, /window\.setTimeout\(run, 60\)/);
  assert.match(appSource, /window\.setTimeout\(run, 140\)/);
  assert.match(appSource, /window\.setTimeout\(\(\) => run\('auto'\), 260\)/);
  assert.match(appSource, /window\.setTimeout\(\(\) => run\('auto'\), 360\)/);
  assert.match(appSource, /window\.setTimeout\(\(\) => run\('auto'\), 760\)/);
  assert.match(appSource, /window\.setTimeout\(\(\) => run\('auto'\), 1600\)/);
  assert.match(appSource, /window\.setTimeout\(\(\) => run\('auto'\), 2600\)/);
  assert.match(appSource, /window\.setTimeout\(\(\) => run\('auto'\), 4200\)/);
  assert.match(appSource, /CEO candidate loaded\./);
  assert.match(appSource, /tokenLaunchBriefDecisionContext\.label = label/);
  assert.match(appSource, /tokenLaunchBriefDecisionContext\.candidate = candidate\.title/);
  assert.match(appSource, /\.filter\(\(brief\) => brief\.launch_type === currentType\)[\s\S]*\.slice\(0, 3\)/);
  assert.match(appSource, /Source[\s\S]*<Link2 :size="10"/);
  assert.match(appSource, /No airdrop memo recorded yet\./);
  assert.match(appSource, /No presale memo recorded yet\./);
  assert.match(appSource, /const tokenCeoProjectResearchRows = computed/);
  assert.match(appSource, /const tokenCeoSourcePacketRows = computed/);
  assert.match(appSource, /const tokenCeoResearchSignalRows = computed/);
  assert.match(appSource, /const tokenCeoLaunchBriefCopy = computed/);
  assert.match(appSource, /const tokenLaunchBriefProofCount = computed/);
  assert.match(appSource, /entry\?\.type !== 'token_launch_brief'/);
  assert.match(appSource, /return String\(entry\.reference \|\| ''\)\.includes\(`type:\$\{targetLaunchType\}`\);/);
  assert.match(appSource, /\$\{ceoMemos\} CEO memos/);
  assert.match(appSource, /\$\{ceoMemos\} CEO memos, \$\{proofRows\} public proof rows available/);
  assert.match(appSource, /\$\{ceoMemos\} CEO memos, \$\{proofRows\} ledger rows, Solana proof manifest linked/);
  assert.match(appSource, /Send a CEO airdrop brief\./);
  assert.match(appSource, /Send a CEO presale brief\./);
  assert.match(appSource, /launchTypeLabel: 'Earned airdrop'/);
  assert.match(appSource, /ledgerMemo: 'type:airdrop'/);
  assert.match(appSource, /launchTypeLabel: 'MRG presale'/);
  assert.match(appSource, /ledgerMemo: 'type:presale'/);
  assert.match(appSource, /Fill CEO template/);
  assert.match(appSource, /quickGates: \['Source', 'Wallet', 'Proof', 'Risk'\]/);
  assert.match(appSource, /quickGates: \['Utility', 'Wallet', 'Funding', 'Risk'\]/);
  assert.match(appSource, /Wallet policy <b>\*<\/b>/);
  assert.match(appSource, /CEO risk notes <b>\*<\/b>/);
  assert.match(appSource, /Research URL <b>\*<\/b>/);
  assert.match(appSource, /class="wizard-field token-ceo-research-url-field"/);
  assert.match(appSource, /urlPlaceholder: 'https:\/\/github\.com\/org\/repo or public proof page'/);
  assert.match(appSource, /urlPlaceholder: 'https:\/\/project\.site\/whitepaper or Solana contract proof'/);
  assert.match(appSource, /Use a repo, task board, docs, website, or public proof URL for CEO research\./);
  assert.match(appSource, /Research URL is required for CEO launch research\./);
  assert.match(appSource, /Research URL must start with http:\/\/ or https:\/\/\./);
  assert.match(appSource, /tokenLaunchBriefFieldError\('wallet_policy'\)/);
  assert.match(appSource, /tokenLaunchBriefFieldError\('risk_notes'\)/);
  assert.match(appSource, /walletPlaceholder: 'Require Solana wallet uniqueness, duplicate review, and anti-bot checks\.'/);
  assert.match(appSource, /riskPlaceholder: 'Flag reserve caps, reversal risk, contract mismatch, and compliance language\.'/);
  assert.match(appSource, /function resetTokenLaunchBriefForm\(\)/);
  assert.match(appSource, /function prefillTokenLaunchBrief\(\)/);
  assert.match(appSource, /resetTokenLaunchBriefForm\(\);/);
  assert.match(appSource, /CEO research template added\./);
  assert.match(appSource, /MRG presale readiness review/);
  assert.match(appSource, /Earned MRG airdrop mission review/);
  assert.match(appSource, /openProjectWizard\(\{ intent: 'token-launch' \}\)/);
  assert.match(appSource, /nextIntent === 'token-launch'/);
  assert.match(appSource, /const tokenLaunchBriefMode = ref\(''\)/);
  assert.match(appSource, /const tokenLaunchBriefForm = reactive/);
  assert.match(appSource, /const tokenLaunchBriefDecisionContext = reactive/);
  assert.match(appSource, /const tokenLaunchBriefValidationMap = computed/);
  assert.match(appSource, /errors\.wallet_policy = 'Wallet policy must explain wallet ownership or uniqueness checks\.'/);
  assert.match(appSource, /errors\.risk_notes = 'CEO risk notes must explain the launch risk review\.'/);
  assert.match(appSource, /api\('\/api\/token\/launch-briefs'/);
  assert.match(appSource, /function submitTokenLaunchBrief\(\)/);
  assert.match(appSource, /ceo_memo/);
  assert.match(appSource, /class="wizard-token-brief-card"/);
  assert.match(appSource, /const tokenLaunchWizardBriefCopy = computed/);
  assert.match(appSource, /const tokenLaunchWizardChecklist = computed/);
  assert.match(appSource, /CEO presale research mode/);
  assert.match(appSource, /CEO airdrop research mode/);
  assert.match(appSource, /tokenLaunchBriefMode\.value = isPresale \? 'presale' : 'airdrop'/);
  assert.match(appSource, /CEO research brief for earned MRG airdrop/);
  assert.match(appSource, /CEO research brief for MRG presale window/);
  assert.match(appSource, /Mission demand/);
  assert.match(appSource, /Reserve receipt/);
  assert.match(appSource, /Project seeking a task-based MRG airdrop/);
  assert.match(appSource, /Project seeking an MRG presale window/);
  assert.match(appSource, /Research URL, bounty demand, proof depth, wallet risk/);
  assert.match(appSource, /Whitepaper, utility, reserve cap, funding rail, contract proof/);
  assert.match(appSource, /Research candidates/);
  assert.match(appSource, /API \+ ledger/);
  assert.match(appSource, /mission_demand/);
  assert.match(appSource, /utility_readiness/);
  assert.match(appSource, /research_signals: tokenCeoResearchSignalRows\.value/);
  assert.match(appSource, /tokenLaunchBriefResult\.research_signals\?\.length/);
  assert.match(appSource, /Project brief intake/);
  assert.match(appSource, /Project utility intake/);
  assert.match(appSource, /Proof risk review/);
  assert.match(appSource, /Open-window memo/);
  assert.match(appSource, /Mission-market fit/);
  assert.match(appSource, /Utility and allocation readiness/);
  assert.match(appSource, /Proof and anti-bot gate/);
  assert.match(appSource, /Contract and ledger proof/);
  assert.match(appSource, /@submit\.prevent="submitAirdropClaim"/);
  assert.match(appSource, /@submit\.prevent="submitPresaleReservation"/);
  assert.match(appSource, /api\('\/api\/airdrop\/claims'/);
  assert.match(appSource, /api\('\/api\/presale\/reservations'/);
  assert.match(appSource, /function submitAirdropClaim\(\)/);
  assert.match(appSource, /function submitPresaleReservation\(\)/);
  assert.match(appSource, /class="token-workflow-proof-board"/);
  assert.match(appSource, /class="token-ceo-memo-lane"/);
  assert.match(appSource, /class="token-workflow-proof-empty-steps"/);
  assert.match(appSource, /class="token-workflow-proof-empty-actions"/);
  assert.match(appSource, /@click="openTokenLaunchBriefFromProofBoard"/);
  assert.match(appSource, /function openTokenLaunchBriefFromProofBoard\(\)/);
  assert.match(appSource, /document\.querySelector\('\.token-ceo-brief-card'\)/);
  assert.match(appSource, /const tokenWorkflowProofEmptySteps = computed/);
  assert.match(appSource, /CEO launch proof appears before claims/);
  assert.match(appSource, /CEO launch proof appears before reserves/);
  assert.match(appSource, /Send a CEO airdrop brief first\. MergeOS records the memo, gate summary, and ledger proof before earned missions open\./);
  assert.match(appSource, /Send a CEO presale brief first\. MergeOS records utility, wallet, funding, risk, and contract gates before reserve receipts open\./);
  assert.match(appSource, /memo hash before claim review/);
  assert.match(appSource, /memo hash before reservations/);
  assert.match(appSource, /const tokenWorkflowProofRows = computed\(\(\) => \{/);
  assert.match(appSource, /const tokenWorkflowCeoMemoRows = computed\(\(\) => \{/);
  assert.match(appSource, /const tokenLaunchBriefsData = ref/);
  assert.match(appSource, /publicApi\('\/api\/public\/token\/launch-briefs'\)/);
  assert.match(appSource, /function mapTokenLaunchBriefQueueRow\(brief = \{\}\)/);
  assert.match(appSource, /const targetType = publicPage\.value === 'airdrop' \? 'airdrop_claim' : 'presale_reservation';/);
  assert.match(appSource, /const targetLaunchType = publicPage\.value === 'presale' \? 'presale' : 'airdrop';/);
  assert.match(appSource, /entry\?\.type !== 'token_launch_brief'/);
  assert.match(appSource, /reference\.includes\(`type:\$\{targetLaunchType\}`\)/);
  assert.match(appSource, /function mapTokenWorkflowProofRow\(entry = \{\}\)/);
  assert.match(appSource, /const isLaunchBrief = entry\.type === 'token_launch_brief';/);
  assert.match(appSource, /let idPattern = \/presale:\(\[\^;\]\+\)\//);
  assert.ok(tokenLaunchCandidatesSchema.properties.candidates.items.required.includes('next_action'));
  assert.equal(tokenLaunchCandidatesSchema.properties.candidates.items.properties.next_action.maxLength, 260);
  assert.match(appSource, /if \(isLaunchBrief\) idPattern = \/launch_brief:\(\[\^;\]\+\)\//);
  assert.match(appSource, /if \(isAirdrop\) idPattern = \/airdrop:\(\[\^;\]\+\)\//);
  assert.match(appSource, /const gateSummaryMatch = reference\.match\(\/gate_summary:\(\[\^;\]\+\)\/\);/);
  assert.match(appSource, /const sourceMatch = reference\.match\(\/source:\(\[\^;\]\+\)\/\) \|\| reference\.match\(\/repo:\(\[\^;\]\+\)\/\);/);
  assert.match(appSource, /const gateSummary = gateSummaryMatch\?\.\[1\]/);
  assert.match(appSource, /\$\{toTitleLabel\(decisionMatch\?\.\[1\] \|\| 'pending open decision'\)\} \/ \$\{gateSummary\}\$\{sourceMatch\?\.\[1\] \? ' \/ source linked' : ''\}/);
  assert.match(appSource, /amount: isLaunchBrief \? 'CEO memo' : formatLedgerMRGFromCents\(entry\.amount_cents\)/);
  assert.match(appSource, /command: 'token-launch-brief'/);
  assert.match(appSource, /action\.command === 'airdrop-claim' \|\| action\.command === 'presale-reserve'/);
  assert.match(appSource, /function refreshTokenPageData\(\)/);
  assert.match(appSource, /async function copyWhitepaperOutline\(\)/);
  assert.match(seoSource, /MergeOS Airdrop \| Task-based MRG rewards with public proof/);
  assert.match(seoSource, /MergeOS Presale \| MRG reserve workflow, Solana token path, and ledger receipts/);
  assert.match(seoSource, /MergeOS Whitepaper \| AI software delivery OS architecture and MRG economy/);
  assert.match(seoSource, /sameAs: \[absoluteUrl\('\/whitepaper\/mergeos-whitepaper\.md'/);
  assert.match(whitepaperSource, /# MergeOS Whitepaper/);
  assert.match(whitepaperSource, /## 4\. Repository Architecture/);
  assert.match(whitepaperSource, /## 7\. AI Layer/);
  assert.match(whitepaperSource, /## 10\. MRG Economy/);
  assert.match(whitepaperSource, /## 12\. Protocol and SDK/);
  assert.match(whitepaperSource, /## 13\. Airdrop Missions/);
  assert.match(whitepaperSource, /## 14\. Presale Workflow/);
  assert.match(whitepaperSource, /## 15\. Security, Privacy, and Compliance/);
  assert.match(whitepaperSource, /## 16\. Roadmap/);
  assert.match(cssSource, /\.token-workflow-proof-board\s*\{[\s\S]*background: rgba\(255, 255, 255, 0\.86\);/);
  assert.match(cssSource, /\.token-ceo-memo-lane\s*\{[\s\S]*background: linear-gradient/);
  assert.match(cssSource, /\.token-ceo-memo-lane article\s*\{[\s\S]*grid-template-columns: 30px minmax\(0, 1fr\) auto;/);
  assert.match(cssSource, /\.token-workflow-proof-empty-steps\s*\{[\s\S]*grid-template-columns: repeat\(3, minmax\(0, 1fr\)\);/);
  assert.match(cssSource, /\.token-workflow-proof-empty-actions\s*\{[\s\S]*display: flex;/);
  assert.match(cssSource, /\.token-workflow-proof-empty-actions button:first-child\s*\{[\s\S]*background: var\(--green\);/);
  assert.match(cssSource, /\.token-workflow-proof-empty-steps\s*\{[\s\S]*grid-template-columns: 1fr;[\s\S]*gap: 6px;/);
  assert.match(cssSource, /\.token-workflow-proof-empty-actions\s*\{[\s\S]*display: grid;[\s\S]*grid-template-columns: 1fr;/);
  assert.match(cssSource, /CEO token launch polish: show gates before intake, keep the review surface short/);
  assert.match(cssSource, /\.token-ceo-research-panel\s*\{[\s\S]*border: 1px solid rgba\(79, 70, 229, 0\.15\);/);
  assert.match(cssSource, /\.token-ceo-head-actions\s*\{[\s\S]*justify-items: end;/);
  assert.match(cssSource, /\.token-ceo-decision-strip\s*\{[\s\S]*grid-template-columns: repeat\(3, minmax\(0, 1fr\)\);/);
  assert.match(cssSource, /\.token-ceo-queue-stats\s*\{[\s\S]*grid-template-columns: repeat\(3, minmax\(0, 1fr\)\);/);
  assert.match(cssSource, /\.token-ceo-live-queue\s*\{[\s\S]*grid-template-columns: repeat\(3, minmax\(0, 1fr\)\);/);
  assert.match(cssSource, /\.token-ceo-live-context\s*\{[\s\S]*grid-template-columns: repeat\(3, minmax\(0, 1fr\)\);/);
  assert.match(cssSource, /\.token-ceo-live-actions\s*\{[\s\S]*display: flex;/);
  assert.match(cssSource, /\.token-ceo-live-empty\s*\{[\s\S]*grid-template-columns: 30px minmax\(0, 1fr\) auto;/);
  assert.match(cssSource, /\.token-ceo-candidate-lane\s*\{[\s\S]*grid-template-columns: repeat\(3, minmax\(0, 1fr\)\);/);
  assert.match(cssSource, /\.token-ceo-candidate-empty\s*\{[\s\S]*grid-template-columns: 34px minmax\(0, 1fr\) auto;/);
  assert.match(cssSource, /\.token-ceo-candidate-empty-actions button\s*\{[\s\S]*background: #0f9f78;/);
  assert.match(cssSource, /\.token-ceo-candidate-actions\s*\{[\s\S]*grid-column: 2;/);
  assert.match(cssSource, /\.token-ceo-candidate-actions a,[\s\S]*\.token-ceo-candidate-actions button\s*\{[\s\S]*text-decoration: none;/);
  assert.match(cssSource, /\.token-ceo-candidate-lane small b\s*\{[\s\S]*border-radius: 999px;/);
  assert.match(cssSource, /\.token-ceo-candidate-policy\s*\{[\s\S]*background: rgba\(240, 253, 250, 0\.72\);/);
  assert.match(cssSource, /\.token-ceo-candidate-policy em\s*\{[\s\S]*font-style: normal;[\s\S]*font-weight: 900;/);
  assert.match(cssSource, /\.token-ceo-candidate-decisions\s*\{[\s\S]*grid-template-columns: repeat\(3, minmax\(0, 1fr\)\);/);
  assert.match(cssSource, /\.token-ceo-candidate-decisions button\.approve\s*\{[\s\S]*background: #ecfdf5;/);
  assert.match(cssSource, /\.token-ceo-project-queue\s*\{[\s\S]*grid-template-columns: repeat\(3, minmax\(0, 1fr\)\);/);
  assert.match(cssSource, /\.token-ceo-source-packet\s*\{[\s\S]*grid-template-columns: repeat\(3, minmax\(0, 1fr\)\);/);
  assert.match(cssSource, /\.token-ceo-signal-chips\s*\{[\s\S]*flex-wrap: wrap;/);
  assert.match(cssSource, /\.token-ceo-brief-gates\s*\{[\s\S]*grid-template-columns: repeat\(4, minmax\(0, max-content\)\);/);
  assert.match(cssSource, /\.token-ceo-decision-context\s*\{[\s\S]*grid-template-columns: auto minmax\(0, 1fr\) auto;/);
  assert.match(cssSource, /\.token-ceo-brief-card\s*\{[\s\S]*grid-template-columns: minmax\(0, 1fr\);/);
  assert.match(cssSource, /\.token-ceo-brief-card\s*\{[\s\S]*scroll-margin-top: 86px;/);
  assert.match(cssSource, /\.token-ceo-launch-context\s*\{[\s\S]*grid-template-columns: repeat\(2, minmax\(0, max-content\)\);/);
  assert.match(cssSource, /\.token-ceo-launch-context span\s*\{[\s\S]*border-radius: 999px;/);
  assert.match(cssSource, /\.token-ceo-launch-context b\s*\{[\s\S]*text-transform: uppercase;/);
  assert.match(cssSource, /\.token-hero-ceo-strip\s*\{[\s\S]*grid-template-columns: repeat\(3, minmax\(0, 1fr\)\);/);
  assert.match(cssSource, /\.token-hero-ceo-strip article\s*\{[\s\S]*border-radius: 8px;/);
  assert.match(cssSource, /@media \(max-width: 620px\)[\s\S]*\.token-hero-ceo-strip\s*\{[\s\S]*grid-template-columns: repeat\(3, minmax\(0, 1fr\)\) !important;/);
  assert.match(cssSource, /\.token-ceo-brief-form\s*\{[\s\S]*grid-template-columns: repeat\(2, minmax\(0, 1fr\)\);/);
  assert.match(cssSource, /\.token-ceo-brief-actions\s*\{[\s\S]*grid-template-columns: minmax\(0, 1fr\) auto auto;/);
  assert.match(cssSource, /\.token-ceo-research-grid\s*\{[\s\S]*grid-template-columns: repeat\(3, minmax\(0, 1fr\)\);/);
  assert.match(cssSource, /\.token-ceo-memo-result\s*\{[\s\S]*grid-column: 1 \/ -1;/);
  assert.match(cssSource, /\.token-ceo-memo-summary\s*\{[\s\S]*grid-template-columns: repeat\(3, minmax\(0, 1fr\)\);/);
  assert.match(cssSource, /\.token-ceo-memo-summary span\.hold\s*\{[\s\S]*background: rgba\(254, 242, 242, 0\.78\);/);
  assert.match(cssSource, /\.token-ceo-memo-source\s*\{[\s\S]*display: inline-flex;/);
  assert.match(cssSource, /\.token-proof-result-actions a,[\s\S]*\.token-proof-result-actions button\s*\{[\s\S]*display: inline-flex;/);
  assert.match(cssSource, /\.token-ceo-memo-gates\s*\{[\s\S]*grid-template-columns: repeat\(4, minmax\(0, 1fr\)\);/);
  assert.match(cssSource, /\.token-ceo-project-queue article\s*\{[\s\S]*min-height: 104px !important;/);
  assert.match(cssSource, /\.token-ceo-source-packet article\s*\{[\s\S]*grid-template-columns: 30px minmax\(0, 1fr\);/);
  assert.match(cssSource, /\.token-ceo-research-grid article\s*\{[\s\S]*min-height: 118px !important;/);
  assert.match(cssSource, /\.wizard-token-brief-card\s*\{[\s\S]*grid-template-columns: 34px minmax\(0, 1fr\);/);
  assert.match(cssSource, /\.wizard-token-brief-card li\s*\{[\s\S]*grid-template-columns: 58px minmax\(0, 1fr\);/);
  assert.match(cssSource, /@media \(max-width: 620px\)[\s\S]*\.token-workflow-proof-list article\s*\{[\s\S]*grid-template-columns: 32px minmax\(0, 1fr\);/);
  assert.match(cssSource, /@media \(max-width: 620px\)[\s\S]*\.token-ceo-memo-lane article,[\s\S]*\.token-workflow-proof-list article\s*\{[\s\S]*grid-template-columns: 32px minmax\(0, 1fr\);/);
  assert.match(cssSource, /@media \(max-width: 620px\)[\s\S]*\.token-status-panel \.ledger-card-head\s*\{[\s\S]*grid-template-columns: minmax\(0, 1fr\) auto;/);
  assert.match(cssSource, /Token mobile refresh affordance: icon-only, stable, and not a clipped text pill/);
  assert.match(cssSource, /@media \(max-width: 620px\)[\s\S]*\.token-status-panel \.ledger-card-head button\s*\{[\s\S]*width: 32px !important;[\s\S]*font-size: 0 !important;/);
  assert.match(cssSource, /@media \(max-width: 620px\)[\s\S]*\.token-metric-grid\s*\{[\s\S]*grid-template-columns: repeat\(2, minmax\(0, 1fr\)\);/);
  assert.match(cssSource, /@media \(max-width: 620px\)[\s\S]*\.token-status-panel > p\s*\{[\s\S]*display: none;/);
  assert.match(cssSource, /@media \(max-width: 620px\)[\s\S]*\.token-ceo-research-grid,[\s\S]*\.token-ceo-decision-strip,[\s\S]*\.token-ceo-project-queue,[\s\S]*\.token-whitepaper-thesis/);
  assert.match(cssSource, /@media \(max-width: 620px\)[\s\S]*\.token-ceo-brief-card\s*\{[\s\S]*grid-template-columns: 1fr;/);
  assert.match(cssSource, /@media \(max-width: 620px\)[\s\S]*\.token-ceo-brief-form\s*\{[\s\S]*grid-template-columns: 1fr;/);
  assert.match(cssSource, /Token mobile rhythm: keep airdrop, presale, and CEO research pages decisive instead of long/);
  assert.match(cssSource, /@media \(max-width: 620px\)[\s\S]*\.token-hero-copy h1\s*\{[\s\S]*font-size: clamp\(28px, 8\.4vw, 34px\) !important;/);
  assert.match(cssSource, /@media \(max-width: 620px\)[\s\S]*\.token-metric-grid article:nth-child\(n \+ 3\)\s*\{[\s\S]*display: none !important;/);
  assert.match(cssSource, /@media \(max-width: 620px\)[\s\S]*\.token-ceo-brief-form\s*\{[\s\S]*grid-template-columns: repeat\(2, minmax\(0, 1fr\)\) !important;/);
  assert.match(cssSource, /@media \(max-width: 620px\)[\s\S]*\.token-ceo-memo-gates\s*\{[\s\S]*grid-template-columns: repeat\(2, minmax\(0, 1fr\)\) !important;/);
  assert.match(cssSource, /@media \(max-width: 620px\)[\s\S]*\.token-ceo-memo-source\s*\{[\s\S]*width: 100% !important;/);
  assert.match(cssSource, /@media \(max-width: 620px\)[\s\S]*\.token-ceo-live-queue article:nth-child\(n \+ 3\),[\s\S]*display: none !important;/);
  assert.match(cssSource, /@media \(max-width: 620px\)[\s\S]*\.token-ceo-live-context span:nth-child\(n \+ 3\)\s*\{[\s\S]*display: none !important;/);
  assert.match(cssSource, /@media \(max-width: 620px\)[\s\S]*\.token-ceo-live-empty\s*\{[\s\S]*grid-template-columns: minmax\(0, 1fr\) !important;/);
  assert.match(cssSource, /@media \(max-width: 620px\)[\s\S]*\.token-ceo-candidate-lane article:nth-child\(n \+ 3\),[\s\S]*display: none !important;/);
  assert.match(cssSource, /@media \(max-width: 620px\)[\s\S]*\.token-ceo-project-queue article:nth-child\(n \+ 3\),[\s\S]*\.token-ceo-research-grid article:nth-child\(n \+ 3\)\s*\{[\s\S]*display: none !important;/);
  assert.match(cssSource, /@media \(max-width: 620px\)[\s\S]*\.token-ceo-source-packet,[\s\S]*\.token-ceo-research-grid\s*\{[\s\S]*grid-template-columns: repeat\(2, minmax\(0, 1fr\)\) !important;/);
  assert.match(cssSource, /@media \(max-width: 620px\)[\s\S]*\.token-ceo-research-grid small\s*\{[\s\S]*display: none !important;/);
  assert.match(cssSource, /Token CEO mobile cockpit: lead with candidate decision and brief/);
  assert.match(cssSource, /@media \(max-width: 620px\)[\s\S]*\.token-ceo-head-actions\s*\{[\s\S]*display: none !important;/);
  assert.match(cssSource, /@media \(max-width: 620px\)[\s\S]*\.token-ceo-queue-stats\s*\{[\s\S]*display: none !important;/);
  assert.match(cssSource, /@media \(max-width: 620px\)[\s\S]*\.token-ceo-project-queue,[\s\S]*\.token-ceo-source-packet,[\s\S]*\.token-ceo-research-grid,[\s\S]*\.token-ceo-live-queue,[\s\S]*\.token-ceo-live-empty\s*\{[\s\S]*display: none !important;/);
  assert.match(cssSource, /@media \(max-width: 620px\)[\s\S]*\.token-ceo-candidate-lane em\s*\{[\s\S]*white-space: nowrap !important;[\s\S]*text-overflow: ellipsis !important;/);
  assert.match(cssSource, /@media \(max-width: 620px\)[\s\S]*\.token-ceo-candidate-policy\s*\{[\s\S]*background: rgba\(255, 255, 255, 0\.82\) !important;/);
  assert.match(cssSource, /@media \(max-width: 620px\)[\s\S]*\.token-ceo-brief-form\s*\{[\s\S]*grid-template-columns: minmax\(0, 1fr\) !important;/);
  assert.match(cssSource, /@media \(max-width: 620px\)[\s\S]*\.token-ceo-signal-chips\s*\{[\s\S]*display: none !important;/);
  assert.match(cssSource, /Token workflow mobile forms: compact enough to complete/);
  assert.match(cssSource, /@media \(max-width: 620px\)[\s\S]*\.token-form-grid\s*\{[\s\S]*grid-template-columns: minmax\(0, 1fr\) !important;/);
  assert.match(cssSource, /@media \(max-width: 620px\)[\s\S]*\.token-form-grid input,[\s\S]*\.token-form-grid textarea,[\s\S]*\.token-form-grid select\s*\{[\s\S]*min-height: 40px !important;/);
  assert.match(cssSource, /@media \(max-width: 620px\)[\s\S]*\.token-ceo-brief-form \.wizard-field\.full textarea\s*\{[\s\S]*min-height: 116px !important;/);
  assert.match(cssSource, /Token CEO brief compact form pass: keep mobile decision flow above the fold/);
  assert.match(cssSource, /\/\* Token CEO brief compact form pass:[\s\S]*\.token-ceo-brief-form\s*\{[\s\S]*grid-template-columns: repeat\(2, minmax\(0, 1fr\)\) !important;/);
  assert.match(cssSource, /\/\* Token CEO brief compact form pass:[\s\S]*\.token-ceo-brief-form \.wizard-field small\s*\{[\s\S]*display: none !important;/);
  assert.match(cssSource, /\/\* Token CEO brief compact form pass:[\s\S]*\.token-ceo-brief-form \.token-ceo-research-url-field\s*\{[\s\S]*grid-column: 1 \/ -1 !important;/);
  assert.match(cssSource, /\/\* Token CEO brief compact form pass:[\s\S]*\.token-ceo-brief-form \.token-ceo-research-url-field small\s*\{[\s\S]*display: -webkit-box !important;[\s\S]*-webkit-line-clamp: 1 !important;/);
  assert.match(cssSource, /\/\* Token CEO brief compact form pass:[\s\S]*\.token-ceo-brief-form \.wizard-field\.full textarea\s*\{[\s\S]*min-height: 108px !important;/);
  assert.match(cssSource, /Token CEO candidate decision polish: turn raw research signals into scan-friendly evidence chips/);
  assert.match(cssSource, /\.token-ceo-candidate-context\s*\{/);
  assert.match(cssSource, /\.token-ceo-candidate-context small\s*\{/);
  assert.match(cssSource, /\.token-ceo-candidate-context span:nth-child\(n \+ 3\)\s*\{[\s\S]*display: none !important;/);
  assert.match(cssSource, /\.token-ceo-candidate-context small\s*\{[\s\S]*font-size: 9px !important;/);
  assert.match(cssSource, /\.token-ceo-candidate-signals\s*\{[\s\S]*display: flex;[\s\S]*flex-wrap: wrap;/);
  assert.match(cssSource, /\.token-ceo-candidate-verdict\s*\{[\s\S]*grid-template-columns: auto minmax\(0, 1fr\);/);
  assert.match(cssSource, /\.token-ceo-candidate-verdict\.ready\s*\{[\s\S]*background: rgba\(236, 253, 245, 0\.82\);/);
  assert.match(cssSource, /\.token-ceo-candidate-verdict\.review\s*\{[\s\S]*background: rgba\(255, 251, 235, 0\.78\);/);
  assert.match(cssSource, /\.token-ceo-candidate-verdict\.hold\s*\{[\s\S]*background: rgba\(254, 242, 242, 0\.76\);/);
  assert.match(cssSource, /\.token-ceo-candidate-readiness\s*\{[\s\S]*grid-template-columns: repeat\(3, minmax\(0, 1fr\)\);/);
  assert.match(cssSource, /\.token-ceo-candidate-readiness span\.ready\s*\{[\s\S]*background: rgba\(236, 253, 245, 0\.82\);/);
  assert.match(cssSource, /\.token-ceo-candidate-readiness span\.hold\s*\{[\s\S]*background: rgba\(254, 242, 242, 0\.76\);/);
  assert.match(cssSource, /@media \(max-width: 620px\)[\s\S]*\.token-ceo-candidate-signals\s*\{[\s\S]*grid-template-columns: repeat\(2, minmax\(0, 1fr\)\) !important;/);
  assert.match(cssSource, /@media \(max-width: 620px\)[\s\S]*\.token-ceo-candidate-readiness\s*\{[\s\S]*grid-template-columns: repeat\(3, minmax\(0, 1fr\)\) !important;/);
  assert.match(cssSource, /Token CEO mobile action hierarchy: make candidate review feel like one clear CEO decision/);
  assert.match(cssSource, /@media \(max-width: 620px\)[\s\S]*\.token-ceo-candidate-actions\s*\{[\s\S]*grid-template-columns: minmax\(0, 0\.78fr\) minmax\(0, 1\.22fr\) !important;/);
  assert.match(cssSource, /@media \(max-width: 620px\)[\s\S]*\.token-ceo-candidate-actions button\s*\{[\s\S]*background: #0f9f78 !important;[\s\S]*color: #ffffff !important;/);
  assert.match(cssSource, /@media \(max-width: 620px\)[\s\S]*\.token-ceo-candidate-decisions button\s*\{[\s\S]*border-radius: 8px !important;/);
  assert.match(cssSource, /Token CEO empty candidate polish: make the fallback feel like a decision gate, not an API state/);
  assert.match(cssSource, /\.token-ceo-empty-gates\s*\{[\s\S]*display: flex !important;[\s\S]*flex-wrap: wrap !important;/);
  assert.match(cssSource, /@media \(max-width: 620px\)[\s\S]*\.token-ceo-empty-gates\s*\{[\s\S]*grid-template-columns: repeat\(3, minmax\(0, 1fr\)\) !important;/);
  assert.match(cssSource, /Token CEO brief intake polish: show the required decision gates before the fields/);
  assert.match(cssSource, /@media \(max-width: 620px\)[\s\S]*\.token-ceo-brief-gates\s*\{[\s\S]*grid-template-columns: repeat\(4, minmax\(0, 1fr\)\) !important;/);
  assert.match(cssSource, /Token CEO mobile first decision: put candidate actions inside the first viewport/);
  assert.match(cssSource, /\/\* Token CEO mobile first decision:[\s\S]*\.token-hero\s*\{[\s\S]*gap: 4px !important;/);
  assert.match(cssSource, /\/\* Token CEO mobile first decision:[\s\S]*\.token-ceo-decision-strip\s*\{[\s\S]*display: none !important;/);
  assert.match(cssSource, /Token CEO mobile launch desk pass: make airdrop\/presale feel like a short CEO decision/);
  assert.match(cssSource, /\/\* Token CEO mobile launch desk pass:[\s\S]*\.token-page-airdrop \.token-hero,[\s\S]*\.token-page-presale \.token-hero\s*\{[\s\S]*display: block !important;[\s\S]*padding: 10px !important;/);
  assert.match(cssSource, /\/\* Token CEO mobile launch desk pass:[\s\S]*\.token-page-airdrop \.token-status-panel,[\s\S]*\.token-page-presale \.token-status-panel\s*\{[\s\S]*display: none !important;/);
  assert.match(cssSource, /\/\* Token CEO mobile launch desk pass:[\s\S]*\.token-page-airdrop \.token-hero-copy p,[\s\S]*\.token-page-presale \.token-hero-copy p\s*\{[\s\S]*-webkit-line-clamp: 2 !important;/);
  assert.match(cssSource, /\/\* Token CEO mobile launch desk pass:[\s\S]*\.token-page-airdrop \.token-ceo-launch-context,[\s\S]*\.token-page-presale \.token-ceo-launch-context,/);
  assert.match(cssSource, /Token mobile viewport lock: CEO launch pages must never crop copy or actions/);
  assert.match(cssSource, /\/\* Token mobile viewport lock:[\s\S]*\.token-page-airdrop,[\s\S]*\.token-page-presale\s*\{[\s\S]*max-width: 100vw !important;[\s\S]*overflow-x: hidden !important;/);
  assert.match(cssSource, /\/\* Token mobile viewport lock:[\s\S]*\.token-page-airdrop \.home-container,[\s\S]*\.token-page-presale \.home-container\s*\{[\s\S]*width: calc\(100vw - 24px\) !important;[\s\S]*max-width: calc\(100vw - 24px\) !important;/);
  assert.match(cssSource, /\/\* Token mobile viewport lock:[\s\S]*\.token-page-airdrop \.token-hero-copy h1,[\s\S]*\.token-page-presale \.token-hero-copy h1\s*\{[\s\S]*font-size: clamp\(25px, 7\.6vw, 30px\) !important;[\s\S]*overflow-wrap: anywhere !important;/);
  assert.match(cssSource, /\/\* Token mobile viewport lock:[\s\S]*\.token-page-airdrop \.token-hero-copy p,[\s\S]*\.token-page-presale \.token-hero-copy p,[\s\S]*\.token-page-airdrop \.token-ceo-candidate-empty p,[\s\S]*-webkit-line-clamp: unset !important;/);
  assert.match(cssSource, /\/\* Token mobile viewport lock:[\s\S]*\.token-page-airdrop \.token-ceo-research-head,[\s\S]*\.token-page-presale \.token-ceo-research-head\s*\{[\s\S]*grid-template-columns: 34px minmax\(0, 1fr\) !important;/);
  assert.match(cssSource, /\/\* Token mobile viewport lock:[\s\S]*\.token-page-airdrop \.token-ceo-candidate-empty,[\s\S]*\.token-page-presale \.token-ceo-candidate-empty\s*\{[\s\S]*grid-template-columns: 28px minmax\(0, 1fr\) !important;/);
  assert.match(cssSource, /\/\* Token mobile viewport lock:[\s\S]*\.token-page-airdrop \.token-ceo-candidate-empty-actions,[\s\S]*\.token-page-presale \.token-ceo-candidate-empty-actions\s*\{[\s\S]*grid-column: 1 \/ -1 !important;[\s\S]*grid-template-columns: minmax\(0, 1fr\) minmax\(0, 1fr\) !important;/);
  assert.match(cssSource, /@media \(max-width: 430px\)[\s\S]*\.token-page-airdrop \.home-container,[\s\S]*\.token-page-presale \.home-container\s*\{[\s\S]*width: min\(360px, calc\(100vw - 24px\)\) !important;/);
  assert.match(cssSource, /@media \(max-width: 430px\)[\s\S]*\.token-page-airdrop \.token-hero-copy p,[\s\S]*\.token-page-presale \.token-hero-copy p,[\s\S]*max-width: 330px !important;/);
  assert.match(cssSource, /Token universal phone lock: whitepaper, presale, and airdrop share one safe viewport/);
  assert.match(cssSource, /\/\* Token universal phone lock:[\s\S]*@media \(max-width: 620px\)[\s\S]*\.token-page \.home-container,[\s\S]*\.token-page \.token-shell\s*\{[\s\S]*width: min\(360px, calc\(100vw - 24px\)\) !important;/);
  assert.match(cssSource, /\/\* Token universal phone lock:[\s\S]*@media \(max-width: 620px\)[\s\S]*\.token-page \.ledger-title-row\s*\{[\s\S]*grid-template-columns: minmax\(0, 1fr\) !important;/);
  assert.match(cssSource, /\/\* Token universal phone lock:[\s\S]*@media \(max-width: 620px\)[\s\S]*\.token-page \.token-hero-copy h1,[\s\S]*\.token-page \.token-whitepaper-copy h2\s*\{[\s\S]*font-size: clamp\(24px, 7\.4vw, 30px\) !important;/);
  assert.match(cssSource, /\/\* Token universal phone lock:[\s\S]*@media \(max-width: 620px\)[\s\S]*\.token-page-whitepaper \.token-status-panel \.ledger-card-head button\s*\{[\s\S]*display: none !important;/);
  assert.match(cssSource, /\/\* Token universal phone lock:[\s\S]*\.token-page \.home-container,[\s\S]*\.token-page \.token-shell\s*\{[\s\S]*width: min\(360px, calc\(100vw - 24px\)\) !important;/);
  assert.match(cssSource, /\/\* Token universal phone lock:[\s\S]*\.token-page \.token-whitepaper-reader,[\s\S]*\.token-page \.token-whitepaper-copy,[\s\S]*\.token-page \.token-whitepaper-brief/);
  assert.match(cssSource, /\/\* Token universal phone lock:[\s\S]*\.token-page \.token-whitepaper-copy h2,[\s\S]*\.token-page \.token-whitepaper-copy p,[\s\S]*max-width: 330px !important;/);
  assert.match(cssSource, /\/\* Token universal phone lock:[\s\S]*\.token-page \.marketplace-actions,[\s\S]*\.token-page \.token-whitepaper-actions\s*\{[\s\S]*grid-template-columns: minmax\(0, 1fr\) !important;/);
  assert.match(cssSource, /Token desktop CEO runway: get from launch thesis to CEO decision faster/);
  assert.match(cssSource, /\/\* Token desktop CEO runway:[\s\S]*@media \(min-width: 981px\)[\s\S]*\.token-page \.token-shell\s*\{[\s\S]*gap: 18px !important;/);
  assert.match(cssSource, /\/\* Token desktop CEO runway:[\s\S]*\.token-page \.token-hero\s*\{[\s\S]*padding-block: clamp\(34px, 4\.2vw, 58px\) clamp\(18px, 2vw, 28px\) !important;/);
  assert.match(cssSource, /\/\* Token desktop CEO runway:[\s\S]*\.token-page \.token-hero-copy h1\s*\{[\s\S]*max-width: 820px !important;[\s\S]*font-size: clamp\(38px, 3\.35vw, 50px\) !important;/);
  assert.match(cssSource, /\/\* Token desktop CEO runway:[\s\S]*\.token-page \.token-ceo-research-panel\s*\{[\s\S]*padding: 18px 20px !important;/);
  assert.match(cssSource, /\.token-proof-result small\s*\{[\s\S]*overflow: visible;[\s\S]*white-space: normal;[\s\S]*overflow-wrap: anywhere;/);
  assert.match(cssSource, /\.token-whitepaper-thesis p\s*\{[\s\S]*-webkit-line-clamp: 2;/);
  assert.match(cssSource, /Whitepaper mobile skim: keep the route decisive before the full reader begins/);
  assert.match(cssSource, /@media \(max-width: 620px\)[\s\S]*\.token-whitepaper-thesis article:nth-child\(n \+ 3\),[\s\S]*\.token-whitepaper-index article:nth-child\(n \+ 3\)\s*\{[\s\S]*display: none !important;/);
  assert.match(cssSource, /Token workflow mobile skim: show the decision path, then get users to the form faster/);
  assert.match(cssSource, /@media \(max-width: 620px\)[\s\S]*\.token-timeline-list li:nth-child\(n \+ 4\)\s*\{[\s\S]*display: none !important;/);
  assert.match(cssSource, /@media \(max-width: 620px\)[\s\S]*\.token-timeline-list li\s*\{[\s\S]*grid-template-columns: 28px minmax\(0, 1fr\) !important;/);
  assert.match(cssSource, /Token workflow mobile field pairs: keep optional metadata compact while preserving full-width proof fields/);
  assert.match(cssSource, /@media \(max-width: 620px\)[\s\S]*\.token-form-grid\s*\{[\s\S]*grid-template-columns: repeat\(2, minmax\(0, 1fr\)\) !important;/);
  assert.match(cssSource, /@media \(max-width: 620px\)[\s\S]*\.token-form-grid \.token-compact-half\s*\{[\s\S]*grid-column: span 1 !important;/);
});

test('contracts page exposes Solana proof manifest alongside the public IDL', async () => {
  const appSource = await fs.readFile(new URL('./src/App.vue', import.meta.url), 'utf-8');
  const proofManifest = JSON.parse(await fs.readFile(new URL('./public/contracts/solana/mergeos_mrg.proof-manifest.v1.json', import.meta.url), 'utf-8'));

  assert.equal(proofManifest.protocol_version, 'mergeos.solana-contract-proof.v1');
  assert.equal(proofManifest.program, 'mergeos_mrg');
  assert.ok(proofManifest.instruction_map.some((row) => row.instruction === 'openEscrow' && row.ledger_types.includes('task_reserve')));
  assert.match(appSource, /const solanaMRGProofManifestPath = '\/contracts\/solana\/mergeos_mrg\.proof-manifest\.v1\.json';/);
  assert.match(appSource, /openExternalURL\(solanaMRGProofManifestPath\)/);
  assert.match(appSource, /manifestAction: 'Open proof manifest'/);
  assert.match(appSource, /key: 'proof-manifest'/);
  assert.match(appSource, /mergeos\.solana-contract-proof\.v1/);
});

test('signed-in mobile dashboard keeps nav, actions, and popovers phone-safe', async () => {
  const cssSource = await fs.readFile(new URL('./src/styles.css', import.meta.url), 'utf-8');
  const appSource = await fs.readFile(new URL('./src/App.vue', import.meta.url), 'utf-8');

  assert.match(cssSource, /Signed-in mobile system/);
  assert.match(appSource, /class="dash-mobile-nav"/);
  assert.match(appSource, /dashboardMobilePrimaryNav/);
  assert.match(appSource, /toggleDashboardMobileSearch/);
  assert.match(cssSource, /\.dashboard-shell \.dash-mobile-nav\s*\{[\s\S]*grid-template-columns: repeat\(5, minmax\(0, 1fr\)\);/);
  assert.match(cssSource, /\.dashboard-shell \.dash-side-nav\s*\{[\s\S]*display: none;/);
  assert.match(cssSource, /\.dashboard-shell \.dash-top-actions\s*\{[\s\S]*grid-template-columns: 44px 44px minmax\(0, 1fr\) 44px;/);
  assert.match(cssSource, /\.notification-dropdown\s*\{[\s\S]*bottom: calc\(12px \+ var\(--dashboard-mobile-bottom-inset, 0px\) \+ env\(safe-area-inset-bottom\)\) !important;/);
  assert.match(cssSource, /\.account-context-menu,[\s\S]*\.dashboard-account-menu \.account-context-menu\s*\{[\s\S]*bottom: calc\(12px \+ env\(safe-area-inset-bottom\)\);/);
  assert.match(cssSource, /Signed-in mobile readability owner/);
  assert.match(cssSource, /\.dashboard-shell \.dashboard-role-map\s*\{[\s\S]*grid-auto-flow: row !important;[\s\S]*overflow-x: visible !important;[\s\S]*mask-image: none !important;/);
  assert.match(cssSource, /\.dashboard-shell \.dashboard-role-proof\s*\{[\s\S]*grid-template-columns: repeat\(2, minmax\(0, 1fr\)\) !important;/);
  assert.match(cssSource, /Signed-in mobile overflow guard/);
  assert.match(cssSource, /\.dashboard-shell \.admin-console-grid,[\s\S]*\.dashboard-shell \.payment-summary-grid,[\s\S]*grid-template-columns: minmax\(0, 1fr\);/);
  assert.match(cssSource, /\.dashboard-shell \.admin-ops-row,[\s\S]*\.dashboard-shell \.payment-history-row,[\s\S]*grid-template-columns: minmax\(0, 1fr\);/);
  assert.match(cssSource, /\.dashboard-shell \.notification-center-list article,[\s\S]*\.dashboard-shell \.dash-pr-row,[\s\S]*\.dashboard-shell \.auto-release-runbook li,[\s\S]*\.dashboard-shell \.agent-runbook-strip li\s*\{[\s\S]*grid-template-columns: minmax\(0, 1fr\);/);
  assert.match(cssSource, /\.dashboard-shell \.repo-finding-list article,[\s\S]*\.dashboard-shell \.repo-task-graph-list article,[\s\S]*grid-template-columns: minmax\(0, 1fr\);/);
  assert.match(cssSource, /\.dashboard-shell \.admin-ops-row-actions,[\s\S]*\.dashboard-shell \.worker-proposal-actions,[\s\S]*grid-template-columns: minmax\(0, 1fr\);/);
  assert.match(cssSource, /\.dashboard-shell \.agent-task-actions,[\s\S]*\.dashboard-shell \.repo-import-publish-actions,[\s\S]*grid-template-columns: minmax\(0, 1fr\);/);
  assert.match(cssSource, /\.dashboard-shell \.dashboard-tool-form button,[\s\S]*\.dashboard-shell \.payment-history-side button,[\s\S]*\.dashboard-shell \.dash-pr-actions a,[\s\S]*\.dashboard-shell \.repo-import-publish-actions button\s*\{[\s\S]*min-height: 42px;/);
  assert.match(cssSource, /\.dashboard-shell \.dash-topnav\s*\{[\s\S]*display: none;/);
  assert.match(cssSource, /\.dashboard-shell \.dashboard-project-actions-panel\s*\{[\s\S]*position: fixed;[\s\S]*bottom: calc\(12px \+ env\(safe-area-inset-bottom\)\);/);
  assert.match(cssSource, /\.notification-dropdown\s*\{[\s\S]*left: clamp\(12px, 4vw, 18px\) !important;[\s\S]*right: clamp\(12px, 4vw, 18px\) !important;/);
  assert.match(cssSource, /\.account-menu\.open \.account-context-menu,[\s\S]*opacity: 1 !important;[\s\S]*visibility: visible !important;/);
  assert.match(cssSource, /@media \(max-width: 430px\)[\s\S]*\.dashboard-shell \.dashboard-role-map\s*\{[\s\S]*grid-auto-flow: row !important;/);
  assert.match(cssSource, /\.mobile-nav-panel\s*\{[\s\S]*height: 100dvh;[\s\S]*max-height: 100dvh;/);
  assert.match(cssSource, /\.auth-modal\s*\{[\s\S]*max-height: calc\(100dvh - 64px\);/);
  assert.match(cssSource, /Signed-in mobile content guard/);
  assert.match(cssSource, /\.dashboard-shell \.auto-release-payload-strip span,[\s\S]*\.dashboard-shell \.worker-claim-context a,[\s\S]*\.dashboard-shell \.dashboard-reference-list a,[\s\S]*overflow-wrap: anywhere;/);
  assert.match(cssSource, /\.dashboard-shell \.worker-claim-packet,[\s\S]*\.dashboard-shell \.worker-claim-warnings\s*\{[\s\S]*grid-template-columns: minmax\(0, 1fr\);/);
  assert.match(cssSource, /\.dashboard-shell \.ledger-table thead\s*\{[\s\S]*display: none;/);
  assert.match(cssSource, /\.dashboard-shell \.ledger-table tr\s*\{[\s\S]*border: 1px solid #e5edf0;[\s\S]*border-radius: 10px;/);
  assert.match(cssSource, /\.dashboard-shell \.ledger-table td:nth-child\(5\)::before\s*\{[\s\S]*content: "Reference";/);
  assert.match(cssSource, /Signed-in mobile polish pass/);
  assert.match(cssSource, /body:has\(\.dashboard-shell\)[\s\S]*overflow-x: clip;/);
  assert.match(cssSource, /\.notification-dropdown\s*\{[\s\S]*bottom: calc\(12px \+ var\(--dashboard-mobile-bottom-inset, 0px\) \+ env\(safe-area-inset-bottom\)\) !important;/);
  assert.match(cssSource, /\.notification-dropdown-item\s*\{[\s\S]*grid-template-columns: 10px minmax\(0, 1fr\);/);
  assert.match(cssSource, /Signed-in mobile repair pass/);
  assert.match(cssSource, /\.dashboard-shell \.dash-search\.mobile-open\s*\{[\s\S]*display: grid;/);
  assert.match(cssSource, /\.dashboard-shell \.dash-content\s*\{[\s\S]*padding: 0 0 calc\(22px \+ env\(safe-area-inset-bottom\)\);/);
  assert.match(cssSource, /\.dashboard-shell \.dash-project-title\s*\{[\s\S]*grid-template-columns: 44px minmax\(0, 1fr\);/);
  assert.match(cssSource, /\.dashboard-shell \.auto-release-command-head,[\s\S]*\.dashboard-shell \.payment-history-meta\s*\{[\s\S]*grid-template-columns: minmax\(0, 1fr\);/);
  assert.match(cssSource, /@media \(max-width: 430px\)[\s\S]*\.dashboard-shell \.dash-top-actions\s*\{[\s\S]*grid-template-columns: 42px 42px minmax\(0, 1fr\) 42px;/);
  assert.match(cssSource, /Signed-in mobile hardening pass/);
  assert.match(cssSource, /\.dashboard-shell,[\s\S]*\.dashboard-shell \*\s*\{[\s\S]*box-sizing: border-box;/);
  assert.match(cssSource, /\.dashboard-shell \.dash-mobile-menu\s*\{[\s\S]*position: fixed;[\s\S]*max-height: min\(58dvh, 420px\);/);
  assert.match(cssSource, /\.dashboard-shell \.notification-dropdown,[\s\S]*\.dashboard-account-menu \.account-context-menu\s*\{[\s\S]*max-width: calc\(100vw - 24px\) !important;/);
  assert.match(cssSource, /\.dashboard-shell \.dashboard-tool-form,[\s\S]*\.dashboard-shell \.worker-claim-warnings\s*\{[\s\S]*grid-template-columns: minmax\(0, 1fr\);/);
  assert.match(cssSource, /@media \(max-width: 430px\)[\s\S]*\.dashboard-shell \.dash-mobile-nav button\s*\{[\s\S]*min-height: 44px;/);
  assert.match(cssSource, /Signed-in mobile login-safe pass/);
  assert.match(cssSource, /\.dashboard-shell \.dash-top-actions\s*\{[\s\S]*grid-template-columns: 44px 44px minmax\(0, 1fr\) 44px;/);
  assert.match(cssSource, /\.dashboard-shell \.dash-topnav\s*\{[\s\S]*display: none !important;/);
  assert.match(cssSource, /\.dashboard-shell \.dash-mobile-menu\s*\{[\s\S]*max-width: calc\(100vw - \(var\(--dash-mobile-gutter, 14px\) \* 2\)\);[\s\S]*overflow-y: auto;[\s\S]*-webkit-overflow-scrolling: touch;/);
  assert.match(cssSource, /@media \(max-width: 430px\)[\s\S]*\.dashboard-shell \.dash-top-actions\s*\{[\s\S]*grid-template-columns: 42px 42px minmax\(0, 1fr\) 42px !important;/);
  assert.match(cssSource, /\.dashboard-shell \.notification-dropdown,[\s\S]*\.dashboard-shell \.dashboard-project-actions-panel\s*\{[\s\S]*left: clamp\(12px, 4vw, 18px\) !important;[\s\S]*right: clamp\(12px, 4vw, 18px\) !important;[\s\S]*max-width: calc\(100vw - \(var\(--dash-mobile-gutter, 14px\) \* 2\)\) !important;[\s\S]*overscroll-behavior: contain;/);
  assert.match(cssSource, /\.dashboard-shell \.dashboard-project-actions-panel\s*\{[\s\S]*width: auto !important;/);
  assert.match(cssSource, /@media \(max-width: 430px\)[\s\S]*\.project-step-actions > div\s*\{[\s\S]*grid-template-columns: minmax\(0, 1fr\);/);
  assert.match(cssSource, /\.project-preview-dialog\s*\{[\s\S]*max-height: calc\(100dvh - 20px\);[\s\S]*overscroll-behavior: contain;/);
  assert.match(cssSource, /\.project-step-list strong\s*\{[\s\S]*font-size: 10\.5px;[\s\S]*-webkit-line-clamp: 2;/);
  assert.match(cssSource, /Project wizard mobile final owner/);
  assert.match(cssSource, /body:has\(\.project-flow-shell\)[\s\S]*overflow-x: clip;/);
  assert.match(cssSource, /\.project-flow-main\s*\{[\s\S]*grid-template-columns: minmax\(0, 1fr\) !important;/);
  assert.match(cssSource, /\.project-step-actions,[\s\S]*\.funding-actions\s*\{[\s\S]*max-width: calc\(100vw - \(var\(--project-mobile-gutter\) \* 2\)\);/);
  assert.match(cssSource, /\.project-account-menu \.account-context-menu,[\s\S]*\.project-flow-actions \.locale-context-menu\s*\{[\s\S]*position: fixed;[\s\S]*top: calc\(62px \+ env\(safe-area-inset-top\)\) !important;[\s\S]*max-height: min\(62dvh, 420px\);/);
  assert.match(cssSource, /\.project-preview-dialog\s*\{[\s\S]*max-width: calc\(100vw - \(var\(--project-mobile-gutter\) \* 2\)\);[\s\S]*-webkit-overflow-scrolling: touch;/);
  assert.match(cssSource, /Signed-in mobile final QA owner/);
  assert.match(cssSource, /\.dashboard-shell\s*\{[\s\S]*--dash-touch-target: 44px;/);
  assert.match(cssSource, /\.dashboard-shell \.dash-mobile-nav\s*\{[\s\S]*grid-template-columns: repeat\(5, minmax\(0, 1fr\)\) !important;/);
  assert.match(cssSource, /\.dashboard-shell \.dash-top-actions\s*\{[\s\S]*grid-template-columns: var\(--dash-touch-target\) var\(--dash-touch-target\) minmax\(88px, 1fr\) var\(--dash-touch-target\) !important;/);
  assert.match(cssSource, /\.dashboard-shell \.dash-command-copy p\s*\{[\s\S]*-webkit-line-clamp: unset !important;/);
  assert.match(cssSource, /\.dashboard-shell \.dashboard-role-map\s*\{[\s\S]*mask-image: none !important;/);
  assert.match(cssSource, /Signed-in mobile executive pass: dashboard starts as an app, not a long role manual/);
  assert.match(cssSource, /\/\* Signed-in mobile executive pass:[\s\S]*\.dashboard-shell \.dash-command-copy p\s*\{[\s\S]*-webkit-line-clamp: 2 !important;/);
  assert.match(cssSource, /\/\* Signed-in mobile executive pass:[\s\S]*\.dashboard-shell \.dash-command-metrics article:nth-child\(n \+ 3\)\s*\{[\s\S]*display: none !important;/);
  assert.match(cssSource, /\/\* Signed-in mobile executive pass:[\s\S]*\.dashboard-shell \.dashboard-role-map\s*\{[\s\S]*grid-template-columns: repeat\(4, minmax\(0, 1fr\)\) !important;/);
  assert.match(cssSource, /\/\* Signed-in mobile executive pass:[\s\S]*\.dashboard-shell \.dashboard-role-map article.active\s*\{[\s\S]*grid-column: 1 \/ -1 !important;/);
  assert.match(cssSource, /\/\* Signed-in mobile executive pass:[\s\S]*\.dashboard-shell \.dashboard-role-map article.active \.dashboard-role-proof\s*\{[\s\S]*grid-template-columns: repeat\(2, minmax\(0, 1fr\)\) !important;/);
  assert.match(cssSource, /\.notification-dropdown,[\s\S]*\.dashboard-shell \.notification-dropdown,[\s\S]*\.dashboard-shell \.account-context-menu,/);
  assert.match(cssSource, /\.dashboard-shell :is\(input, select, textarea\)\s*\{[\s\S]*font-size: 16px;/);
  assert.match(cssSource, /\.project-flow-shell \.project-flow-main\s*\{[\s\S]*padding-bottom: calc\(86px \+ env\(safe-area-inset-bottom\)\);/);
  assert.match(cssSource, /\.project-account-menu \.account-context-menu,[\s\S]*\.project-flow-actions \.locale-context-menu\s*\{[\s\S]*top: auto !important;[\s\S]*bottom: calc\(12px \+ env\(safe-area-inset-bottom\)\) !important;/);
  assert.match(cssSource, /\.project-flow-shell \.project-step-actions,[\s\S]*\.project-flow-shell \.funding-actions\s*\{[\s\S]*backdrop-filter: blur\(16px\);/);
  assert.match(cssSource, /Project wizard mobile action bar polish/);
  assert.match(cssSource, /\.project-flow-shell\s*\{[\s\S]*--project-mobile-action-height: 64px;/);
  assert.match(cssSource, /\.project-flow-shell \.project-step-actions\s*\{[\s\S]*grid-template-columns: 42px minmax\(0, 1fr\) !important;[\s\S]*min-height: var\(--project-mobile-action-height\) !important;/);
  assert.match(cssSource, /\.project-flow-shell \.project-step-actions > \.secondary-button\s*\{[\s\S]*font-size: 0 !important;/);
  assert.match(cssSource, /\.project-flow-shell \.project-step-actions > div\s*\{[\s\S]*grid-template-columns: minmax\(0, 0\.92fr\) minmax\(0, 1\.08fr\) !important;/);
  assert.match(cssSource, /@media \(max-width: 380px\)[\s\S]*\.dashboard-shell \.dash-mobile-nav button span\s*\{[\s\S]*display: none;/);
  assert.match(cssSource, /@media \(max-width: 380px\)[\s\S]*\.dashboard-shell \.dash-top-actions \.primary-button\.compact span\s*\{[\s\S]*display: none;/);
  assert.match(cssSource, /@media \(max-width: 380px\)[\s\S]*\.notification-dropdown-actions\s*\{[\s\S]*grid-template-columns: minmax\(0, 1fr\) !important;/);
  assert.match(cssSource, /Signed-in mobile continuity owner/);
  assert.match(cssSource, /\.dashboard-shell \.dash-sidebar\s*\{[\s\S]*position: sticky;[\s\S]*z-index: 720;/);
  assert.match(cssSource, /\.dashboard-shell \.dash-topbar\s*\{[\s\S]*position: sticky;[\s\S]*backdrop-filter: blur\(14px\);/);
  assert.match(cssSource, /\.dashboard-shell \.dash-search\.mobile-open\s*\{[\s\S]*display: grid !important;/);
  assert.match(cssSource, /\.dashboard-shell :is\(\.wallet-summary-card, \.project-picker-card, \.customer-protocol-card, \.workflow-pulse-card, \.ai-workflow-card, \.notification-center-card\)\s*\{[\s\S]*overflow: hidden;/);
  assert.match(cssSource, /\.dashboard-shell \.wallet-address-box strong,[\s\S]*\.dashboard-shell \.repo-import-publish-steps :is\(strong, small, b\)\s*\{[\s\S]*overflow-wrap: anywhere;/);
  assert.match(cssSource, /\.dashboard-shell \.dashboard-project-list button,[\s\S]*\.dashboard-shell \.repo-import-publish-steps li\s*\{[\s\S]*grid-template-columns: minmax\(0, 1fr\) !important;/);
  assert.match(cssSource, /@media \(max-width: 430px\)[\s\S]*\.dashboard-shell \.dash-top-actions\s*\{[\s\S]*grid-template-columns: 42px 42px minmax\(58px, 1fr\) 42px !important;/);
  assert.match(cssSource, /Signed-in mobile unified owner/);
  assert.match(appSource, /class="dash-mobile-menu-backdrop"/);
  assert.match(cssSource, /\.dashboard-shell\s*\{[\s\S]*--dash-mobile-header-height: 92px;[\s\S]*--project-action-bar-height: 86px;/);
  assert.match(cssSource, /\.dashboard-shell \.dash-topbar\s*\{[\s\S]*position: static !important;[\s\S]*backdrop-filter: none !important;/);
  assert.match(cssSource, /\.dashboard-shell \.dash-mobile-menu-backdrop\s*\{[\s\S]*position: fixed;[\s\S]*backdrop-filter: blur\(2px\);/);
  assert.match(cssSource, /\.dashboard-shell \.dash-mobile-menu\s*\{[\s\S]*top: calc\(var\(--dash-mobile-header-height\) \+ 8px \+ env\(safe-area-inset-top\)\) !important;[\s\S]*box-shadow: 0 22px 56px rgba\(15, 23, 42, 0\.18\);/);
  assert.match(cssSource, /\.notification-dropdown,[\s\S]*\.dashboard-shell \.notification-dropdown\s*\{[\s\S]*z-index: var\(--dash-mobile-sheet-z, 980\) !important;/);
  assert.match(cssSource, /\.notification-dropdown\s*\{[\s\S]*left: var\(--dash-mobile-gutter, clamp\(12px, 4vw, 18px\)\) !important;[\s\S]*width: auto !important;/);
  assert.match(cssSource, /@media \(max-width: 430px\)[\s\S]*\.notification-dropdown-actions\s*\{[\s\S]*grid-template-columns: minmax\(0, 1fr\) !important;/);
  assert.match(cssSource, /\.dashboard-shell \.admin-dispute-item-actions,[\s\S]*\.dashboard-shell \.admin-ops-row-actions\s*\{[\s\S]*display: flex !important;[\s\S]*flex-wrap: wrap;/);
  assert.match(cssSource, /\.project-flow-main\s*\{[\s\S]*padding-bottom: calc\(var\(--project-action-bar-height\) \+ 34px \+ env\(safe-area-inset-bottom\)\) !important;/);
  assert.match(cssSource, /\.project-step-list\s*\{[\s\S]*grid-template-columns: repeat\(4, minmax\(118px, 1fr\)\);[\s\S]*overflow-x: auto !important;/);
  assert.match(cssSource, /Project wizard mobile final compact pass: make the logged-in creation flow feel like an app, not stacked desktop panels/);
  assert.match(cssSource, /\/\* Project wizard mobile final compact pass:[\s\S]*\.project-flow-shell \.project-flow-title\s*\{[\s\S]*display: none !important;/);
  assert.match(cssSource, /\/\* Project wizard mobile final compact pass:[\s\S]*\.project-flow-shell \.project-step-list\s*\{[\s\S]*grid-auto-flow: column !important;[\s\S]*overflow-x: auto !important;/);
  assert.match(cssSource, /\/\* Project wizard mobile final compact pass:[\s\S]*\.project-flow-shell \.wizard-validation-banner\s*\{[\s\S]*max-height: 132px !important;/);
  assert.match(cssSource, /\/\* Project wizard mobile final compact pass:[\s\S]*\.project-flow-shell \.quality-check-list\s*\{[\s\S]*display: none !important;/);
  assert.match(cssSource, /\/\* Project wizard mobile final compact pass:[\s\S]*\.project-flow-shell \.project-step-actions\s*\{[\s\S]*min-height: var\(--project-mobile-action-height\) !important;/);
  assert.match(cssSource, /@media \(max-width: 430px\)[\s\S]*\.dashboard-shell \.admin-dispute-lane,[\s\S]*\.dashboard-shell \.admin-ops-row\s*\{[\s\S]*grid-template-columns: minmax\(0, 1fr\);/);
  assert.match(appSource, /dashboardNotificationMenuPlacement\.value = 'mobile-sheet';/);
  assert.match(appSource, /window\.visualViewport\?\.addEventListener\('resize', updateDashboardNotificationMenuPosition\);/);
  assert.match(appSource, /<span>New Project<\/span>/);
  assert.match(appSource, /payload\.type === 'notifications_updated'/);
  assert.match(appSource, /function handleWSNotificationsUpdated\(payload = \{\}\)/);
  assert.match(appSource, /loadDashboardNotifications\(\{ silent: true \}\)/);
  assert.match(appSource, /payload\.type === 'admin_ops_updated'/);
  assert.match(appSource, /function handleWSAdminOpsUpdated\(payload = \{\}\)/);
  assert.match(appSource, /loadAdminConsoleData\(\{ silent: true \}\)/);
  assert.match(appSource, /async function loadDashboardNotifications\(options = \{\}\)/);
  assert.match(cssSource, /Signed-in mobile visual QA sweep/);
  assert.match(cssSource, /\.dashboard-shell \.dash-sidebar\s*\{[\s\S]*position: sticky !important;[\s\S]*z-index: 980;/);
  assert.match(cssSource, /\.dashboard-shell \.dash-command-metrics\s*\{[\s\S]*grid-auto-flow: column;[\s\S]*overflow-x: auto;/);
  assert.match(cssSource, /\.dashboard-shell \.worker-dashboard-grid,[\s\S]*\.dashboard-shell \.payment-summary-grid\s*\{[\s\S]*grid-template-columns: minmax\(0, 1fr\) !important;/);
  assert.match(cssSource, /\.dashboard-shell \.admin-user-list article,[\s\S]*\.dashboard-shell \.admin-reputation-list article,[\s\S]*grid-template-columns: minmax\(0, 1fr\) !important;/);
  assert.match(cssSource, /\.dashboard-shell \.admin-user-control-strip\s*\{[\s\S]*grid-auto-flow: column;[\s\S]*grid-auto-columns: minmax\(132px, 42vw\);[\s\S]*overflow-x: auto;[\s\S]*scroll-snap-type: x proximity;/);
  assert.match(cssSource, /\.dashboard-shell \.admin-user-control-strip button\s*\{[\s\S]*scroll-snap-align: start;[\s\S]*min-height: 48px;/);
  assert.match(cssSource, /\.dashboard-shell \.worker-proposal-actions,[\s\S]*\.dashboard-shell \.payment-history-actions\s*\{[\s\S]*grid-template-columns: minmax\(0, 1fr\) !important;/);
  assert.match(cssSource, /\.project-step-actions,[\s\S]*\.funding-actions\s*\{[\s\S]*position: sticky;[\s\S]*bottom: 0;/);
  assert.match(cssSource, /Signed-in mobile customer workflow owner/);
  assert.match(cssSource, /\.dashboard-shell :is\([\s\S]*\.escrow-stat-grid,[\s\S]*\.routing-stat-grid,[\s\S]*\.repo-intel-stats,[\s\S]*\.deployment-signal-strip,[\s\S]*\.ai-log-stat-grid,[\s\S]*\)\s*\{[\s\S]*grid-template-columns: repeat\(2, minmax\(0, 1fr\)\) !important;/);
  assert.match(cssSource, /\.dashboard-shell :is\([\s\S]*\.escrow-task-list article,[\s\S]*\.routing-lane-strip article,[\s\S]*\.routing-route-list article,[\s\S]*\.routing-proposal-list article,[\s\S]*\.deployment-status-row,[\s\S]*\.deployment-stage-list article,[\s\S]*\.ai-log-list article,[\s\S]*\)\s*\{[\s\S]*grid-template-columns: minmax\(0, 1fr\) !important;/);
  assert.match(cssSource, /\.dashboard-shell :is\([\s\S]*\.delivery-snapshot-actions,[\s\S]*\.escrow-control-actions,[\s\S]*\.routing-board-actions,[\s\S]*\.repo-intel-actions,[\s\S]*\.ai-log-actions,[\s\S]*\.pr-monitor-heading-actions,[\s\S]*\.routing-proposal-buttons[\s\S]*\)\s*\{[\s\S]*grid-template-columns: repeat\(2, minmax\(0, 1fr\)\) !important;/);
  assert.match(cssSource, /@media \(max-width: 430px\)[\s\S]*\.dashboard-shell :is\([\s\S]*\.escrow-stat-grid,[\s\S]*\.routing-stat-grid,[\s\S]*\.repo-intel-stats,[\s\S]*\.deployment-signal-strip,[\s\S]*\.routing-proposal-buttons[\s\S]*\)\s*\{[\s\S]*grid-template-columns: minmax\(0, 1fr\) !important;/);
  assert.match(cssSource, /Signed-in post-login mobile owner/);
  assert.match(cssSource, /\.dashboard-shell \.dash-sidebar\s*\{[\s\S]*grid-template-columns: 44px minmax\(0, 1fr\);[\s\S]*min-height: var\(--dash-mobile-header-height\) !important;/);
  assert.match(cssSource, /\.dashboard-shell \.dash-brand strong,[\s\S]*\.dashboard-shell \.mrg-card,[\s\S]*\.dashboard-shell \.dash-side-nav\s*\{[\s\S]*display: none !important;/);
  assert.match(cssSource, /\.notification-dropdown,[\s\S]*\.dashboard-shell \.notification-dropdown,[\s\S]*\.dashboard-shell \.account-context-menu,[\s\S]*\.dashboard-account-menu \.account-context-menu\s*\{[\s\S]*bottom: calc\(12px \+ env\(safe-area-inset-bottom\)\) !important;[\s\S]*max-height: min\(60dvh, 460px\) !important;/);
  assert.match(cssSource, /\.dashboard-tool-form label\.invalid,[\s\S]*\.dashboard-shell :is\(input, select, textarea\):invalid\s*\{[\s\S]*scroll-margin-top: calc\(var\(--dash-mobile-header-height\) \+ 16px\);/);
  assert.match(cssSource, /Signed-in mobile cockpit pass: get users into the actual workspace faster/);
  assert.match(cssSource, /\/\* Signed-in mobile cockpit pass:[\s\S]*\.dashboard-shell \.dash-command-strip\s*\{[\s\S]*grid-template-columns: minmax\(0, 1fr\) minmax\(92px, 0\.36fr\) !important;/);
  assert.match(cssSource, /\/\* Signed-in mobile cockpit pass:[\s\S]*\.dashboard-shell \.dash-command-metrics article:nth-child\(n \+ 3\)\s*\{[\s\S]*display: none !important;/);
  assert.match(cssSource, /\/\* Signed-in mobile cockpit pass:[\s\S]*\.dashboard-shell \.dashboard-role-map\s*\{[\s\S]*grid-template-columns: repeat\(4, minmax\(0, 1fr\)\) !important;/);
  assert.match(cssSource, /\/\* Signed-in mobile cockpit pass:[\s\S]*\.dashboard-shell \.dashboard-role-map article:not\(\.active\) \.dashboard-role-proof,/);
  assert.match(cssSource, /\/\* Signed-in mobile cockpit pass:[\s\S]*\.dashboard-shell \.dashboard-role-map article.active\s*\{[\s\S]*min-height: 112px !important;/);
  assert.match(cssSource, /\/\* Signed-in mobile cockpit pass:[\s\S]*\.dashboard-shell \.dashboard-role-map article.active \.dashboard-role-stats,[\s\S]*\.dashboard-shell \.dashboard-role-map article.active \.dashboard-role-lanes\s*\{[\s\S]*display: none !important;/);
  assert.match(cssSource, /Signed-in mobile role switcher pass: keep the role map as navigation, not a second hero/);
  assert.match(cssSource, /\/\* Signed-in mobile role switcher pass:[\s\S]*\.dashboard-shell \.dashboard-role-map article.active\s*\{[\s\S]*grid-template-columns: minmax\(0, 1fr\) auto !important;[\s\S]*min-height: 78px !important;/);
  assert.match(cssSource, /\/\* Signed-in mobile role switcher pass:[\s\S]*\.dashboard-shell \.dashboard-role-map article.active p,[\s\S]*\.dashboard-shell \.dashboard-role-map article.active \.dashboard-role-proof,/);
  assert.match(cssSource, /\/\* Signed-in mobile role switcher pass:[\s\S]*\.dashboard-shell \.dashboard-role-map article.active \.dashboard-role-primary\s*\{[\s\S]*min-height: 34px !important;/);
  assert.match(cssSource, /@media \(max-width: 430px\)[\s\S]*\.dashboard-shell \.dash-command-metrics\s*\{[\s\S]*grid-auto-flow: row !important;[\s\S]*mask-image: none !important;/);
});

test('AI workflow dashboard exposes stage checklists from the protocol contract', async () => {
  const appSource = await fs.readFile(new URL('./src/App.vue', import.meta.url), 'utf-8');
  const cssSource = await fs.readFile(new URL('./src/styles.css', import.meta.url), 'utf-8');
  const aiWorkflowSchema = JSON.parse(await fs.readFile(new URL('./public/protocol/ai-workflow.v1.schema.json', import.meta.url), 'utf-8'));
  const workflowSchema = JSON.parse(await fs.readFile(new URL('./public/protocol/workflow.v1.schema.json', import.meta.url), 'utf-8'));

  const aiStageSchema = aiWorkflowSchema.properties.stages.items;
  const workflowStageSchema = workflowSchema.properties.stages.items;

  assert.ok(aiStageSchema.required.includes('checklist'));
  assert.ok(aiStageSchema.required.includes('actor_lane'));
  assert.deepEqual(aiStageSchema.properties.actor_lane.enum, ['system', 'ai', 'human', 'hybrid', 'deployment_agent']);
  assert.equal(aiStageSchema.properties.checklist.minItems, 1);
  assert.equal(workflowStageSchema.properties.checklist.maxItems, 8);
  assert.match(appSource, /<em v-if="stage\.checklistLabel">\{\{ stage\.checklistLabel \}\}<\/em>/);
  assert.match(appSource, /<em v-if="stage\.actorLabel">\{\{ stage\.actorLabel \}\}<\/em>/);
  assert.match(appSource, /const checklist = Array\.isArray\(stage\.checklist\) \? stage\.checklist\.filter\(Boolean\) : \[\];/);
  assert.match(appSource, /checklistLabel: checklist\.length \? `Checks: \$\{checklist\.slice\(0, 2\)/);
  assert.match(appSource, /actorLabel: stage\.actor_lane \? `Lane: \$\{toTitleLabel\(stage\.actor_lane\)\}` : ''/);
  assert.match(cssSource, /\.ai-workflow-list em\s*\{[\s\S]*-webkit-line-clamp: 2;/);
});

test('frontend Vite stays above the Windows launch-editor advisory floor', async () => {
  const packageSource = JSON.parse(await fs.readFile(new URL('./package.json', import.meta.url), 'utf-8'));
  const lockSource = JSON.parse(await fs.readFile(new URL('./package-lock.json', import.meta.url), 'utf-8'));
  const requestedVite = packageSource.devDependencies?.vite || '';
  const lockedVite = lockSource.packages?.['node_modules/vite']?.version || '';

  assert.equal(requestedVite, '8.0.16');
  assert.equal(lockedVite, '8.0.16');
  assert.equal(compareSemver(lockedVite, '5.4.9') >= 0, true);
});

test('creates runtime config for production defaults', () => {
  const env = {
    NODE_ENV: 'production',
    API_TARGET: 'https://api.example.com',
  };
  const config = createRuntimeConfig({ argv: ['node', 'server.js'], env, cwd: process.cwd() });

  assert.equal(config.mode, 'production');
  assert.equal(config.production, true);
  assert.equal(config.port, 8081);
  assert.equal(config.apiTarget, 'https://api.example.com');
});

test('production server injects SSR HTML into the app shell', async (t) => {
  const cwd = await fs.mkdtemp(path.join(os.tmpdir(), 'mergeos-frontend-ssr-'));
  const clientDist = path.join(cwd, 'client');
  const serverDir = path.join(cwd, 'server');
  const serverEntry = path.join(serverDir, 'entry-server.mjs');
  await fs.mkdir(clientDist, { recursive: true });
  await fs.mkdir(serverDir, { recursive: true });
  await fs.writeFile(
    path.join(clientDist, 'index.html'),
    '<!doctype html><html><body><div id="app"><!--ssr-outlet--></div></body></html>',
  );
  await fs.writeFile(
    serverEntry,
    "export async function render(url) { return `<main id=\"ssr-proof\">${url}</main>`; }\n",
  );

  const server = await createMergeOSServer({
    mode: 'production',
    production: true,
    cwd,
    host: '127.0.0.1',
    port: 0,
    hmrPort: 0,
    apiTarget: 'http://127.0.0.1:65535',
    clientDist,
    serverEntry,
  });
  t.after(() => server.close());
  await new Promise((resolve) => server.listen(0, '127.0.0.1', resolve));

  const address = server.address();
  const response = await fetch(`http://127.0.0.1:${address.port}/admin`);
  const html = await response.text();

  assert.equal(response.status, 200);
  assert.match(response.headers.get('cache-control') || '', /no-store/);
  assert.match(html, /id="ssr-proof"/);
  assert.match(html, />\/admin</);
  assert.doesNotMatch(html, /ssr-outlet/);
  assert.doesNotMatch(html, /<div id="app"><\/div>/);
});

test('production server serves protocol schema assets as JSON', async (t) => {
  const cwd = await fs.mkdtemp(path.join(os.tmpdir(), 'mergeos-frontend-schema-'));
  const clientDist = path.join(cwd, 'client');
  const protocolDir = path.join(clientDist, 'protocol');
  const serverDir = path.join(cwd, 'server');
  const serverEntry = path.join(serverDir, 'entry-server.mjs');
  await fs.mkdir(protocolDir, { recursive: true });
  await fs.mkdir(serverDir, { recursive: true });
  await fs.writeFile(
    path.join(clientDist, 'index.html'),
    '<!doctype html><html><body><div id="app"><!--ssr-outlet--></div></body></html>',
  );
  await fs.writeFile(
    path.join(protocolDir, 'ledger.v1.schema.json'),
    JSON.stringify({ title: 'MergeOS Ledger v1' }),
  );
  await fs.writeFile(serverEntry, "export async function render() { return '<main></main>'; }\n");

  const server = await createMergeOSServer({
    mode: 'production',
    production: true,
    cwd,
    host: '127.0.0.1',
    port: 0,
    hmrPort: 0,
    apiTarget: 'http://127.0.0.1:65535',
    clientDist,
    serverEntry,
  });
  t.after(() => server.close());
  await new Promise((resolve) => server.listen(0, '127.0.0.1', resolve));

  const address = server.address();
  const response = await fetch(`http://127.0.0.1:${address.port}/protocol/ledger.v1.schema.json`);
  const payload = await response.json();

  assert.equal(response.status, 200);
  assert.match(response.headers.get('content-type') || '', /application\/json/);
  assert.equal(payload.title, 'MergeOS Ledger v1');
});

test('production server serves MergeIDE download manifest and preview kit', async (t) => {
  const cwd = await fs.mkdtemp(path.join(os.tmpdir(), 'mergeos-frontend-downloads-'));
  const clientDist = path.join(cwd, 'client');
  const downloadsDir = path.join(clientDist, 'downloads');
  const runbooksDir = path.join(clientDist, 'protocol', 'runbooks');
  const serverDir = path.join(cwd, 'server');
  const serverEntry = path.join(serverDir, 'entry-server.mjs');
  await fs.mkdir(downloadsDir, { recursive: true });
  await fs.mkdir(runbooksDir, { recursive: true });
  await fs.mkdir(serverDir, { recursive: true });
  await fs.writeFile(
    path.join(clientDist, 'index.html'),
    '<!doctype html><html><body><div id="app"><!--ssr-outlet--></div></body></html>',
  );
  await fs.writeFile(
    path.join(downloadsDir, 'mergeide-windows-latest.json'),
    JSON.stringify({ protocol_version: 'mergeos.release-artifact.v1', kind: 'release_artifact' }),
  );
  await fs.writeFile(
    path.join(runbooksDir, 'mergeide-agent.v1.json'),
    JSON.stringify({ protocol_version: 'mergeos.agent-runbook.v1', kind: 'agent_runbook' }),
  );
  await fs.writeFile(path.join(downloadsDir, 'mergeide-preview-kit.md'), '# MergeIDE Preview Kit\n');
  await fs.writeFile(serverEntry, "export async function render() { return '<main></main>'; }\n");

  const server = await createMergeOSServer({
    mode: 'production',
    production: true,
    cwd,
    host: '127.0.0.1',
    port: 0,
    hmrPort: 0,
    apiTarget: 'http://127.0.0.1:65535',
    clientDist,
    serverEntry,
  });
  t.after(() => server.close());
  await new Promise((resolve) => server.listen(0, '127.0.0.1', resolve));

  const address = server.address();
  const manifestResponse = await fetch(`http://127.0.0.1:${address.port}/downloads/mergeide-windows-latest.json`);
  const manifest = await manifestResponse.json();
  const kitResponse = await fetch(`http://127.0.0.1:${address.port}/downloads/mergeide-preview-kit.md`);
  const kit = await kitResponse.text();
  const runbookResponse = await fetch(`http://127.0.0.1:${address.port}/protocol/runbooks/mergeide-agent.v1.json`);
  const runbook = await runbookResponse.json();

  assert.equal(manifestResponse.status, 200);
  assert.match(manifestResponse.headers.get('content-type') || '', /application\/json/);
  assert.equal(manifest.protocol_version, 'mergeos.release-artifact.v1');
  assert.equal(kitResponse.status, 200);
  assert.match(kitResponse.headers.get('content-type') || '', /text\/markdown/);
  assert.match(kit, /MergeIDE Preview Kit/);
  assert.equal(runbookResponse.status, 200);
  assert.match(runbookResponse.headers.get('content-type') || '', /application\/json/);
  assert.equal(runbook.protocol_version, 'mergeos.agent-runbook.v1');
});

test('production server marks hashed assets as immutable', async (t) => {
  const cwd = await fs.mkdtemp(path.join(os.tmpdir(), 'mergeos-frontend-cache-'));
  const clientDist = path.join(cwd, 'client');
  const assetsDir = path.join(clientDist, 'assets');
  const serverDir = path.join(cwd, 'server');
  const serverEntry = path.join(serverDir, 'entry-server.mjs');
  await fs.mkdir(assetsDir, { recursive: true });
  await fs.mkdir(serverDir, { recursive: true });
  await fs.writeFile(path.join(clientDist, 'index.html'), '<!doctype html><html><body><div id="app"><!--ssr-outlet--></div></body></html>');
  await fs.writeFile(path.join(assetsDir, 'index-test.js'), 'window.__mergeos_asset = true;\n');
  await fs.writeFile(serverEntry, "export async function render() { return '<main></main>'; }\n");

  const server = await createMergeOSServer({
    mode: 'production',
    production: true,
    cwd,
    host: '127.0.0.1',
    port: 0,
    hmrPort: 0,
    apiTarget: 'http://127.0.0.1:65535',
    clientDist,
    serverEntry,
  });
  t.after(() => server.close());
  await new Promise((resolve) => server.listen(0, '127.0.0.1', resolve));

  const address = server.address();
  const response = await fetch(`http://127.0.0.1:${address.port}/assets/index-test.js`);
  const script = await response.text();

  assert.equal(response.status, 200);
  assert.match(response.headers.get('cache-control') || '', /immutable/);
  assert.match(script, /__mergeos_asset/);
});

test('production server does not fall back missing assets to app HTML', async (t) => {
  const cwd = await fs.mkdtemp(path.join(os.tmpdir(), 'mergeos-frontend-missing-asset-'));
  const clientDist = path.join(cwd, 'client');
  const serverDir = path.join(cwd, 'server');
  const serverEntry = path.join(serverDir, 'entry-server.mjs');
  await fs.mkdir(clientDist, { recursive: true });
  await fs.mkdir(serverDir, { recursive: true });
  await fs.writeFile(path.join(clientDist, 'index.html'), '<!doctype html><html><body><div id="app"><!--ssr-outlet--></div></body></html>');
  await fs.writeFile(serverEntry, "export async function render() { return '<main></main>'; }\n");

  const server = await createMergeOSServer({
    mode: 'production',
    production: true,
    cwd,
    host: '127.0.0.1',
    port: 0,
    hmrPort: 0,
    apiTarget: 'http://127.0.0.1:65535',
    clientDist,
    serverEntry,
  });
  t.after(() => server.close());
  await new Promise((resolve) => server.listen(0, '127.0.0.1', resolve));

  const address = server.address();
  const response = await fetch(`http://127.0.0.1:${address.port}/assets/missing.css`);
  const body = await response.text();

  assert.equal(response.status, 404);
  assert.equal(response.headers.get('content-type'), 'text/plain; charset=utf-8');
  assert.doesNotMatch(body, /<!doctype html>/i);
});

test('API proxy forwards the public frontend host for auth redirects', async (t) => {
  const api = http.createServer((req, res) => {
    res.setHeader('Content-Type', 'application/json');
    res.end(JSON.stringify({
      host: req.headers.host,
      forwardedHost: req.headers['x-forwarded-host'],
      forwardedProto: req.headers['x-forwarded-proto'],
    }));
  });
  t.after(() => api.close());
  await new Promise((resolve) => api.listen(0, '127.0.0.1', resolve));
  const apiAddress = api.address();

  const cwd = await fs.mkdtemp(path.join(os.tmpdir(), 'mergeos-frontend-proxy-'));
  const clientDist = path.join(cwd, 'client');
  const serverDir = path.join(cwd, 'server');
  const serverEntry = path.join(serverDir, 'entry-server.mjs');
  await fs.mkdir(clientDist, { recursive: true });
  await fs.mkdir(serverDir, { recursive: true });
  await fs.writeFile(path.join(clientDist, 'index.html'), '<!doctype html><html><body><div id="app"><!--ssr-outlet--></div></body></html>');
  await fs.writeFile(serverEntry, "export async function render() { return '<main></main>'; }\n");

  const server = await createMergeOSServer({
    mode: 'production',
    production: true,
    cwd,
    host: '127.0.0.1',
    port: 0,
    hmrPort: 0,
    apiTarget: `http://127.0.0.1:${apiAddress.port}`,
    clientDist,
    serverEntry,
  });
  t.after(() => server.close());
  await new Promise((resolve) => server.listen(0, '127.0.0.1', resolve));
  const frontendAddress = server.address();

  const response = await fetch(`http://127.0.0.1:${frontendAddress.port}/api/header-check`);
  const headers = await response.json();

  assert.equal(headers.host, `127.0.0.1:${apiAddress.port}`);
  assert.equal(headers.forwardedHost, `127.0.0.1:${frontendAddress.port}`);
  assert.equal(headers.forwardedProto, 'http');
});

test('WebSocket proxy upgrades /api/ws to the API target', async (t) => {
  let upgradeHeaders;
  const api = http.createServer();
  api.on('upgrade', (req, socket) => {
    upgradeHeaders = req.headers;
    socket.write([
      'HTTP/1.1 101 Switching Protocols',
      'Upgrade: websocket',
      'Connection: Upgrade',
      'Sec-WebSocket-Accept: test',
      '',
      '',
    ].join('\r\n'));
    socket.end();
  });
  t.after(() => api.close());
  await new Promise((resolve) => api.listen(0, '127.0.0.1', resolve));
  const apiAddress = api.address();

  const cwd = await fs.mkdtemp(path.join(os.tmpdir(), 'mergeos-frontend-ws-proxy-'));
  const clientDist = path.join(cwd, 'client');
  const serverDir = path.join(cwd, 'server');
  const serverEntry = path.join(serverDir, 'entry-server.mjs');
  await fs.mkdir(clientDist, { recursive: true });
  await fs.mkdir(serverDir, { recursive: true });
  await fs.writeFile(path.join(clientDist, 'index.html'), '<!doctype html><html><body><div id="app"><!--ssr-outlet--></div></body></html>');
  await fs.writeFile(serverEntry, "export async function render() { return '<main></main>'; }\n");

  const server = await createMergeOSServer({
    mode: 'production',
    production: true,
    cwd,
    host: '127.0.0.1',
    port: 0,
    hmrPort: 0,
    apiTarget: `http://127.0.0.1:${apiAddress.port}`,
    clientDist,
    serverEntry,
  });
  t.after(() => server.close());
  await new Promise((resolve) => server.listen(0, '127.0.0.1', resolve));
  const frontendAddress = server.address();

  const response = await new Promise((resolve, reject) => {
    const client = net.createConnection(frontendAddress.port, '127.0.0.1');
    let data = '';
    client.on('connect', () => {
      client.write([
        'GET /api/ws HTTP/1.1',
        `Host: 127.0.0.1:${frontendAddress.port}`,
        'Connection: Upgrade',
        'Upgrade: websocket',
        'Sec-WebSocket-Key: test',
        'Sec-WebSocket-Version: 13',
        '',
        '',
      ].join('\r\n'));
    });
    client.on('data', (chunk) => {
      data += chunk.toString('utf-8');
      if (data.includes('\r\n\r\n')) {
        client.destroy();
        resolve(data);
      }
    });
    client.on('error', reject);
    client.setTimeout(2500, () => reject(new Error('websocket proxy timed out')));
  });

  assert.match(response, /101 Switching Protocols/);
  assert.equal(upgradeHeaders.host, `127.0.0.1:${apiAddress.port}`);
  assert.equal(upgradeHeaders['x-forwarded-host'], `127.0.0.1:${frontendAddress.port}`);
  assert.equal(upgradeHeaders['x-forwarded-proto'], 'http');
});

test('shared Vue entry leaves browser mounting to the client hydration entry', async () => {
  const main = await fs.readFile(new URL('./src/main.js', import.meta.url), 'utf-8');
  const client = await fs.readFile(new URL('./src/entry-client.js', import.meta.url), 'utf-8');

  assert.match(main, /createSSRApp/);
  assert.doesNotMatch(main, /\.mount\(/);
  assert.doesNotMatch(main, /typeof document|createClientApp/);
  assert.match(client, /firstElementChild/);
  assert.match(client, /const initialPath = window\.location\.pathname/);
  assert.match(client, /createHydratedApp\(initialPath\) : createClientApp\(App, \{ initialPath \}\)/);
  assert.match(client, /app\.mount\('#app'\)/);
});
