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
  assert.deepEqual(architectureManifest.repository_architecture.map((repo) => repo.name), ['mergeos-app', 'mergeos-contracts', 'mergeos-sdk', 'mergeos-protocol']);
  assert.deepEqual(architectureManifest.users.map((row) => row.type), ['customers', 'contributors', 'ai_agents', 'admins']);
  assert.ok(architectureManifest.frontend_system.stack.includes('Vue 3'));
  assert.ok(architectureManifest.frontend_system.stack.includes('Vite SSR'));
  assert.ok(architectureManifest.frontend_system.public_pages.includes('Marketplace'));
  assert.ok(architectureManifest.frontend_system.authenticated_dashboards.includes('Customer Dashboard'));
  assert.deepEqual(Object.keys(architectureManifest.frontend_system.authenticated_dashboard_urls), ['customer_dashboard', 'worker_dashboard', 'admin_console']);
  assert.equal(architectureManifest.frontend_system.authenticated_dashboard_urls.customer_dashboard.page, '/dashboard');
  assert.equal(architectureManifest.frontend_system.authenticated_dashboard_urls.customer_dashboard.api, '/api/projects/{id}/dashboard');
  assert.equal(architectureManifest.frontend_system.authenticated_dashboard_urls.worker_dashboard.page, '/dashboard?section=worker');
  assert.equal(architectureManifest.frontend_system.authenticated_dashboard_urls.worker_dashboard.api, '/api/workers/me');
  assert.equal(architectureManifest.frontend_system.authenticated_dashboard_urls.admin_console.page, '/dashboard?section=admin');
  assert.equal(architectureManifest.frontend_system.authenticated_dashboard_urls.admin_console.api, '/api/admin/ops-queue');
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
  assert.deepEqual(architectureManifest.ai_layer.workflow, [
    'Import Repository',
    'Issue Scan',
    'Task Generation',
    'Reward Estimation',
    'Contributor Routing',
    'PR Review',
    'Deployment Validation',
  ]);
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
  });
  assert.equal(architectureManifest.public_urls.marketplace_api, '/api/public/marketplace');
  assert.equal(architectureManifest.public_urls.live_feed_api, '/api/public/live-feed');
  assert.equal(architectureManifest.public_urls.agent_queue_api, '/api/public/protocol/agent-queue');
  assert.equal(architectureManifest.public_urls.ledger_api, '/api/public/ledger');
  assert.equal(architectureManifest.public_urls.ledger_events_api, '/api/public/ledger/events');
  assert.equal(architectureManifest.public_urls.ledger_verify_api, '/api/public/ledger/verify');
  assert.equal(architectureManifest.public_urls.ledger_proof_api, '/api/public/ledger/proof');
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

  assert.match(appSource, /const publicProtocolManifestPath = '\/api\/public\/protocol';/);
  assert.equal(paymentOrderSchema.properties.provider.enum.includes('paypal'), true);
  assert.equal(paymentOrderSchema.properties.provider.enum.includes('stripe'), true);
  assert.match(manifestSource, /mergeos\.payment-order\.v1/);
  assert.match(manifestSource, /payment-order\.v1\.schema\.json/);
  assert.match(manifestSource, /\/contracts\/solana\/mergeos_mrg\.proof-manifest\.v1\.json/);
  assert.match(manifestSource, /mergeos\.solana-contract-proof\.v1/);
  assert.match(manifestSource, /\/system\/mergeos-architecture\.v1\.json/);
  assert.match(manifestSource, /mergeos\.architecture\.v1/);
  assert.match(manifestSource, /architecture\.v1\.schema\.json/);
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
});

test('public home keeps a short decision-screen rhythm', async () => {
  const appSource = await fs.readFile(new URL('./src/App.vue', import.meta.url), 'utf-8');
  const cssSource = await fs.readFile(new URL('./src/styles.css', import.meta.url), 'utf-8');

  assert.match(cssSource, /Home quality pass: one short decision screen/);
  assert.match(appSource, /class="public-notification-feed home-feed-preview"/);
  assert.match(appSource, /class="home-mergeide-inline-link"/);
  assert.match(appSource, /class="home-system-summary"/);
  assert.match(appSource, /homeSystemSummaryRows/);
  assert.match(appSource, /class="home-compact-flow"/);
  assert.match(appSource, /homePipelineRows/);
  assert.match(appSource, /class="home-system-explainer"/);
  assert.match(appSource, /localizedHomeWorkflowCards\.slice\(0, 4\)/);
  assert.match(appSource, /homeLiveStats\.slice\(0, 2\)/);
  assert.match(appSource, /MergeOS turns funded software work into verified delivery\./);
  assert.match(appSource, /One command layer for software projects: capture the brief, import repo context, lock escrow, route tasks to builders or AI agents, review PRs, validate deployment, and publish payout proof\./);
  assert.match(appSource, /title: 'Product OS'[\s\S]*Project intake, repo import, AI task graph, escrow, PR monitor, deployment gates, and ledger proof stay in one operating flow\./);
  assert.match(appSource, /title: 'Delivery lanes'[\s\S]*Route funded work to human contributors, AI agents, or hybrid teams with shared scope, acceptance criteria, and payout state\./);
  assert.match(appSource, /title: 'Public proof layer'[\s\S]*Marketplace activity, escrow, token mint, PR evidence, deployment checks, SDK context, and protocol documents are discoverable\./);
  assert.match(appSource, /title: 'Repo OS', body: 'Issues, debt, PRs, deploys'/);
  assert.match(appSource, /title: 'AI routing', body: 'Scan, split, estimate, review'/);
  assert.match(appSource, /title: 'Market \+ escrow', body: 'Funded bounties and payouts'/);
  assert.match(appSource, /title: 'MRG proof', body: 'Solana token and public ledger'/);
  assert.match(appSource, /title: 'Route', body: 'Human, AI, or hybrid tasks'/);
  assert.match(appSource, /title: 'Prove', body: 'PR, deploy, payout ledger'/);
  assert.match(cssSource, /\.public-home-page\s*\{[\s\S]*padding-block: clamp\(14px, 2\.4vw, 30px\) clamp\(20px, 3vw, 38px\) !important;/);
  assert.match(cssSource, /\.public-home-page \.home-container\s*\{[\s\S]*max-width: 1080px !important;/);
  assert.match(cssSource, /\.public-home-hero\s*\{[\s\S]*grid-template-columns: minmax\(0, 1fr\) minmax\(286px, 332px\) !important;/);
  assert.match(cssSource, /\.public-home-copy h1\s*\{[\s\S]*font-size: clamp\(42px, 4\.7vw, 66px\) !important;/);
  assert.match(cssSource, /\.home-command-panel\s*\{[\s\S]*max-width: 332px !important;/);
  assert.match(cssSource, /\.home-mergeide-inline-link,[\s\S]*\.home-proof-stack,[\s\S]*\.home-mergeide-signal,[\s\S]*\.home-public-graph-proof,[\s\S]*\.home-command-panel \.home-pipeline,[\s\S]*\.home-command-panel \.home-feed-preview\s*\{[\s\S]*display: none !important;/);
  assert.match(cssSource, /\.home-command-panel \.public-stat-grid article:nth-child\(n \+ 3\)\s*\{[\s\S]*display: none !important;/);
  assert.match(cssSource, /@media \(max-width: 980px\)[\s\S]*\.home-command-panel\s*\{[\s\S]*display: block !important;[\s\S]*max-width: 560px !important;/);
  assert.match(cssSource, /@media \(max-width: 620px\)[\s\S]*\.public-home-copy h1\s*\{[\s\S]*font-size: clamp\(31px, 10vw, 40px\) !important;/);
  assert.match(cssSource, /\.home-system-explainer\s*\{[\s\S]*grid-template-columns: repeat\(4, minmax\(0, 1fr\)\);/);
  assert.match(cssSource, /\.home-system-summary\s*\{[\s\S]*grid-template-columns: repeat\(4, minmax\(0, 1fr\)\);/);
  assert.match(cssSource, /\.home-compact-flow\s*\{[\s\S]*grid-template-columns: repeat\(3, minmax\(0, 1fr\)\);/);
  assert.match(cssSource, /@media \(max-width: 620px\)[\s\S]*\.home-system-explainer\s*\{[\s\S]*grid-template-columns: minmax\(0, 1fr\);/);
  assert.match(cssSource, /@media \(max-width: 620px\)[\s\S]*\.home-system-summary\s*\{[\s\S]*grid-template-columns: minmax\(0, 1fr\);/);
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
  assert.match(cssSource, /@media \(max-width: 760px\)[\s\S]*\.dashboard-shell \.dashboard-role-map\s*\{[\s\S]*grid-auto-flow: column !important;[\s\S]*grid-auto-columns: minmax\(238px, 72vw\) !important;/);
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

  for (const page of ['airdrop', 'presale', 'whitepaper']) {
    assert.match(appSource, new RegExp(`${page}: '/${page}'`));
    assert.match(seoSource, new RegExp(`${page}: '/${page}'`));
  }
  assert.match(appSource, /v-else-if="publicTokenPage"/);
  assert.match(appSource, /const publicTokenPageDefinitions = \{/);
  assert.match(appSource, /title: 'A task-based airdrop for verified software delivery\.'/);
  assert.match(appSource, /title: 'Reserve MRG through a transparent presale workflow\.'/);
  assert.match(appSource, /title: 'The operating system for AI software delivery\.'/);
  assert.match(appSource, /action: \{ page: 'airdrop' \}/);
  assert.match(appSource, /action: \{ page: 'presale' \}/);
  assert.match(appSource, /action: \{ page: 'whitepaper' \}/);
  assert.match(appSource, /const whitepaperDownloadPath = '\/whitepaper\/mergeos-whitepaper\.md'/);
  assert.match(appSource, /command: 'download-whitepaper'/);
  assert.match(appSource, /function downloadWhitepaper\(\)/);
  assert.match(appSource, /class="token-whitepaper-thesis"/);
  assert.match(appSource, /class="token-whitepaper-brief"/);
  assert.match(appSource, /class="token-whitepaper-section-list"/);
  assert.match(appSource, /const publicWhitepaperThesisRows = computed\(\(\) => \[/);
  assert.match(appSource, /const publicWhitepaperChapterSections = computed\(\(\) => \[/);
  assert.match(appSource, /The paper is structured around executable product proof/);
  assert.match(appSource, /id="token-workflow"/);
  assert.match(appSource, /@submit\.prevent="submitAirdropClaim"/);
  assert.match(appSource, /@submit\.prevent="submitPresaleReservation"/);
  assert.match(appSource, /api\('\/api\/airdrop\/claims'/);
  assert.match(appSource, /api\('\/api\/presale\/reservations'/);
  assert.match(appSource, /function submitAirdropClaim\(\)/);
  assert.match(appSource, /function submitPresaleReservation\(\)/);
  assert.match(appSource, /class="token-workflow-proof-board"/);
  assert.match(appSource, /const tokenWorkflowProofRows = computed\(\(\) => \{/);
  assert.match(appSource, /const targetType = publicPage\.value === 'airdrop' \? 'airdrop_claim' : 'presale_reservation';/);
  assert.match(appSource, /function mapTokenWorkflowProofRow\(entry = \{\}\)/);
  assert.match(appSource, /reference\.match\(isAirdrop \? \/airdrop:\(\[\^;\]\+\)\/ : \/presale:\(\[\^;\]\+\)\//);
  assert.match(appSource, /command: 'airdrop-claim'/);
  assert.match(appSource, /command: 'presale-reserve'/);
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
  assert.match(cssSource, /@media \(max-width: 620px\)[\s\S]*\.token-workflow-proof-list article\s*\{[\s\S]*grid-template-columns: 32px minmax\(0, 1fr\);/);
  assert.match(cssSource, /\.token-proof-result small\s*\{[\s\S]*overflow: visible;[\s\S]*white-space: normal;[\s\S]*overflow-wrap: anywhere;/);
  assert.match(cssSource, /\.token-whitepaper-thesis p\s*\{[\s\S]*-webkit-line-clamp: 2;/);
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
  assert.match(cssSource, /\.dashboard-shell \.dashboard-role-map\s*\{[\s\S]*grid-auto-flow: column !important;[\s\S]*overflow-x: auto !important;/);
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
  assert.match(cssSource, /@media \(max-width: 430px\)[\s\S]*\.dashboard-shell \.dashboard-role-map\s*\{[\s\S]*grid-auto-flow: column !important;/);
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
  assert.match(cssSource, /\.dashboard-shell \.dash-command-copy p\s*\{[\s\S]*-webkit-line-clamp: 2;/);
  assert.match(cssSource, /\.dashboard-shell \.dash-command-metrics,[\s\S]*\.dashboard-shell \.dashboard-role-map\s*\{[\s\S]*mask-image: linear-gradient/);
  assert.match(cssSource, /\.notification-dropdown,[\s\S]*\.dashboard-shell \.notification-dropdown,[\s\S]*\.dashboard-shell \.account-context-menu,/);
  assert.match(cssSource, /\.dashboard-shell :is\(input, select, textarea\)\s*\{[\s\S]*font-size: 16px;/);
  assert.match(cssSource, /\.project-flow-shell \.project-flow-main\s*\{[\s\S]*padding-bottom: calc\(86px \+ env\(safe-area-inset-bottom\)\);/);
  assert.match(cssSource, /\.project-account-menu \.account-context-menu,[\s\S]*\.project-flow-actions \.locale-context-menu\s*\{[\s\S]*top: auto !important;[\s\S]*bottom: calc\(12px \+ env\(safe-area-inset-bottom\)\) !important;/);
  assert.match(cssSource, /\.project-flow-shell \.project-step-actions,[\s\S]*\.project-flow-shell \.funding-actions\s*\{[\s\S]*backdrop-filter: blur\(16px\);/);
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
