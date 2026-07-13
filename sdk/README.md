# MergeOS SDK

Small JavaScript client for MergeOS task, workflow, ledger, and event APIs.

## Install locally

```powershell
cd sdk
npm test
```

## Usage

```js
import {
  adminOpsActionOutputContracts,
  adminOpsQueueOutputContracts,
  airdropClaimPayload,
  agentActionPayloadFromWorkPacket,
  agentActionPayload,
  agentActionEventType,
  agentWorkPacketOutputContracts,
  aiWorkflowCurrentStage,
  aiWorkflowStageActionContract,
  aiWorkflowStageContextURLs,
  contractReferenceFromLedger,
  createMergeOSClient,
  deploymentAgentActionPayload,
  deploymentValidationPayloadFromDeployment,
  isLikelySolanaWallet,
  legacyWalletAddressHash,
  presaleReservationPayload,
  proposalPacketOutputContracts,
  repositorySuggestedTaskFundingPayload,
  repositorySuggestedTaskPayPalOrderPayload,
  protocolEventFromMessage,
  protocolEventsFromMessage,
  protocolEventGroup,
  protocolTypeFromMessage,
  repoPlanningOutputContracts,
  repoPlanningSteps,
  routingPacketOutputContracts,
  routingPacketPayload,
  walletMigrationPDASeedMetadata,
  workerDashboardProofLinks,
} from '@mergeos/sdk';

const mergeos = createMergeOSClient({
  baseURL: 'https://mergeos.shop',
  token: process.env.MERGEOS_TOKEN,
});

const projects = await mergeos.listProjects();
const estimate = await mergeos.evaluateProjectPrice({ description: 'Build an AI delivery workflow.' });
console.log(estimate.protocol_version, estimate.suggested_price_cents);
const escrow = await mergeos.projectEscrow(projects[0].id);
const payouts = await mergeos.projectPayouts(projects[0].id);
console.log(payouts.protocol_version, payouts.release_status);
const pulls = await mergeos.projectPullRequests(projects[0].id);
console.log(pulls.protocol_version, pulls.stats.ready_count);
const autoReleasePacket = pulls.tasks.find((row) => row.auto_release_packet)?.auto_release_packet;
if (autoReleasePacket) {
  // Deployment-sensitive tasks only become auto-release candidates after preview/deployment validation proof.
  const release = await mergeos.projectAutoRelease(projects[0].id, autoReleasePacket.payload);
  console.log(release.kind, release.released_count);
}
const deployment = await mergeos.projectDeployment(projects[0].id);
console.log(deployment.protocol_version, deployment.progress);
if (deployment.validation_packet) {
  const deploymentValidation = await mergeos.createDeploymentValidationFromDeployment(projects[0].id, deployment);
  console.log(deploymentValidation.protocol_version, deploymentValidation.kind);
}
const workflow = await mergeos.projectAIWorkflow(projects[0].id);
console.log(workflow.protocol_version, workflow.current_step);
console.log(workflow.stages[0]?.artifact_kind, workflow.stages[0]?.output_protocol_url);
const currentStage = aiWorkflowCurrentStage(workflow);
const currentContract = aiWorkflowStageActionContract(currentStage);
const currentContextURLs = aiWorkflowStageContextURLs(currentStage);
console.log(currentContract.output_protocol, currentContract.action_endpoint, currentStage.checklist);
console.log(currentContextURLs.deployment_evidence || currentContextURLs.workflow);
const agentAction = await mergeos.createProjectAgentAction(projects[0].id, {
  action: 'test',
  agent_type: 'qa-agent',
});
console.log(agentAction.protocol_version, agentAction.action);
const eventType = agentActionEventType(agentAction.log.action); // "agent.tested"
await mergeos.recordAgentReview(projects[0].id, { pullNumber: 120 });
await mergeos.recordAgentTest(projects[0].id, { status: 'running' });
await mergeos.recordAgentGeneration(projects[0].id, { agentType: 'coding-agent' });
await mergeos.recordAgentScan(projects[0].id, { url: 'https://scan.example/report' });
await mergeos.recordDeployment(projects[0].id, {
  deploymentURL: 'https://vercel.example/deployments/mergeos-preview',
  status: 'processed',
});
const deployPayload = deploymentAgentActionPayload({ url: 'https://vercel.example/deployments/mergeos-preview' });
const validationPayload = deploymentValidationPayloadFromDeployment(deployment);
const testPayload = agentActionPayload('test', { status: 'processed' });
const graph = await mergeos.projectTaskGraph(projects[0].id);
const routing = await mergeos.projectRouting(projects[0].id);
console.log(routing.protocol_version, routing.stats.ready_count);
const nextRoute = routing.routes?.find((route) => route.ready);
if (nextRoute) {
  const contracts = routingPacketOutputContracts(nextRoute);
  const agentActionContracts = contracts.filter((contract) => contract.output_protocol === 'mergeos.agent-action.v1');
  const packetPayload = routingPacketPayload(nextRoute);
  console.log(nextRoute.claim_id, nextRoute.routing_packet.endpoint, agentActionContracts.map((contract) => contract.action));
  await mergeos.executeRoutingPacket(nextRoute, packetPayload);
}
const workflowProtocol = await mergeos.projectWorkflowProtocol(projects[0].id);
console.log(workflowProtocol.current_step, workflowProtocol.progress);
const scan = await mergeos.projectRepositoryScan(projects[0].id);
const scanProtocol = await mergeos.projectRepositoryScanProtocol(projects[0].id);
const syncReport = await mergeos.syncProjectRepoIssues(projects[0].id);
console.log(syncReport.protocol_version, syncReport.added_task_count, syncReport.issue_mappings[0]?.claim_endpoint);
const readyPlanningSteps = repoPlanningSteps(syncReport, 'ready');
const syncContracts = repoPlanningOutputContracts(syncReport, 'mergeos.repo-sync.v1');
console.log(readyPlanningSteps[0]?.id, syncContracts[0]?.output_protocol_url);
const ops = await mergeos.adminOpsQueue();
const opsActionContracts = adminOpsActionOutputContracts(ops.items[0] || {}, 'refresh_admin_ops');
console.log(ops.protocol_version, ops.stats.total_count, opsActionContracts[0]?.output_protocol);
const suggestedTask = scan.suggested_tasks?.find((task) => task.funding_packet?.can_fund);
if (suggestedTask) {
  const fundingPayload = repositorySuggestedTaskFundingPayload(suggestedTask.id, {
    rewardCents: suggestedTask.funding_packet.recommended_reward_cents,
    budgetCents: suggestedTask.funding_packet.recommended_funding_cents,
    paymentMethod: 'card',
    paymentReference: process.env.MERGEOS_PAYMENT_REFERENCE,
  });
  const fundedTask = await mergeos.fundRepositorySuggestedTask(projects[0].id, suggestedTask.id, fundingPayload);
  const outputContracts = agentWorkPacketOutputContracts(fundedTask.work_packet, 'scan');
  const scanAction = agentActionPayloadFromWorkPacket(fundedTask.work_packet, 'scan', {
    status: 'processed',
    referenceURL: 'https://scan.example/report',
  });
  await mergeos.createProjectAgentAction(projects[0].id, scanAction);
}
const solanaReference = contractReferenceFromLedger({ entry_hash: 'a'.repeat(64) }, { format: 'bytes' });
const legacyHash = legacyWalletAddressHash('trc20', 'TXYZ987654321', { format: 'bytes' });
const pda = walletMigrationPDASeedMetadata('trc20', 'TXYZ987654321');
const migration = await mergeos.createWalletMigration({
  legacy_chain: 'trc20',
  legacy_address: 'TXYZ987654321',
});
console.log(migration.protocol_version, migration.contract.instruction);

const wallet = '11111111111111111111111111111111';
if (isLikelySolanaWallet(wallet)) {
  const claim = await mergeos.claimAirdrop(airdropClaimPayload({
    missionID: 'mission_delivery_proof',
    walletAddress: wallet,
    taskReference: 'prj_public_0001:12',
    proofURL: 'https://github.com/acme/repo/pull/12',
  }));
  console.log(claim.protocol_version, claim.ledger_entry.entry_hash);

  const reservation = await mergeos.reservePresale(presaleReservationPayload({
    walletAddress: wallet,
    reserveMRG: 25000,
    fundingRail: 'usdc',
    fundingReference: 'usdc:tx_123',
  }));
  console.log(reservation.protocol_version, reservation.ledger_entry.entry_hash);
}
```

## Evidence Requirement Helpers

```js
import { evidenceRequiredFromEvent, hasEvidenceRequirement } from '@mergeos/sdk';

const event = {
  evidence_required: ['security-review', 'test_output', 'security-review', ''],
  type: 'task.submitted',
};
const requirements = evidenceRequiredFromEvent(event);
// ['security_review', 'test_output']

if (hasEvidenceRequirement(event, 'security-review')) {
  console.log('Security review evidence is required for this task.');
}

// Works with live-feed items, arrays, and comma-separated strings too
const feedItem = { evidenceRequired: 'scan_output,deployment_url' };
console.log(evidenceRequiredFromEvent(feedItem)); // ['scan_output', 'deployment_url']
```

Keys are normalized: `security-review` → `security_review`, blanks and duplicates are ignored.

## Public APIs

```js
await mergeos.publicMarketplace();
await mergeos.runtimeConfig(); // includes payment_rails discovery
await mergeos.publicLedger();
await mergeos.publicLedgerVerification();
await mergeos.publicLedgerProof();
await mergeos.publicLedgerEvents({ limit: 40 });
await mergeos.publicTokenEconomy();
await mergeos.publicLiveFeed({ limit: 80 });
await mergeos.publicLiveFeed({ limit: 80, afterID: 'event:latest', since: '2026-06-06T00:00:00Z' });
await mergeos.publicProtocolManifest();
await mergeos.publicArchitectureManifest();
const architecture = await mergeos.publicArchitectureDiscovery();
await mergeos.publicProtocolTasks({ limit: 80 });
await mergeos.publicProtocolTasks({ taskID: 'prj_public_0001:12' });
await mergeos.publicProtocolAgentQueue({ limit: 80 });
await mergeos.publicAgentRunbook();
await mergeos.publicProtocolAgents({ limit: 80 });
await mergeos.publicProtocolContributors({ limit: 80 });
await mergeos.publicProtocolLedger();
await mergeos.publicMergeIDEWindowsRelease();
await mergeos.publicProtocolEvents({ limit: 80, cursor: 'event:latest' });
await mergeos.publicProjectDeployment('prj_public_0001');
await mergeos.publicProjectAIWorkflow('prj_public_0001');
await mergeos.publicProjectWorkflow('prj_public_0001');
await mergeos.publicProjectRepositoryScan('prj_public_0001');
await mergeos.publicProjectPullRequests('prj_public_0001');
const repoImport = await mergeos.importRepoIssues({ repo_url: 'https://github.com/acme/repo' });
console.log(repoImport.protocol_version, repoImport.issue_count);
console.log(architecture.repositoryByName['mergeos-app'].contains, architecture.aiWorkflow);
await mergeos.publicTestSettingsStatus();
await mergeos.publicTestSettingsAuth('shared-password');
await mergeos.publicTestSettingsEntries('shared-password');
await mergeos.publicRevealTestSettingsEntry('tse_0001', 'shared-password');
```

## Protocol Discovery Helpers

```js
import {
  protocolManifestContextURL,
  protocolManifestDocument,
  protocolManifestEndpoint,
  protocolManifestEventStreamPath,
  protocolManifestRealtime,
} from '@mergeos/sdk';

const discovery = await mergeos.publicProtocolDiscovery();

console.log(discovery.stats.schemaCount, discovery.stats.publicEndpointCount);
console.log(discovery.realtime.websocketPath, discovery.realtime.topics);
console.log(discovery.documentByVersion['mergeos.agent-queue.v1'].publicEndpoint);

const workflowURL = protocolManifestContextURL(discovery.manifest, 'project_workflow', {
  projectID: 'prj_public_0001',
});
const taskURL = protocolManifestContextURL(discovery.manifest, 'task_protocol', {
  bounty_id: 'prj_public_0001:12',
});
const workflowEndpoint = protocolManifestEndpoint(discovery.manifest, 'mergeos.workflow.v1');
const agentQueueDoc = protocolManifestDocument(discovery.manifest, 'agent_queue');
const realtime = protocolManifestRealtime(discovery.manifest);
const streamPath = protocolManifestEventStreamPath(discovery, {
  limit: 40,
  afterID: 'event:latest',
});

console.log(workflowURL, taskURL, workflowEndpoint.path, agentQueueDoc.schemaURL, realtime.readyEvent, streamPath);
```

`publicProtocolDiscovery()` normalizes the enriched `/api/public/protocol` manifest into documents, endpoints, realtime stream metadata, and agent context URLs. External agents, IDE extensions, and integration clients should start here before fetching task manifests, workflow graphs, repository scans, deployment status, or ledger proof.

## Task and workflow APIs

```js
const paypalOrder = await mergeos.createPayPalOrder({
  amount_cents: 120000,
  description: 'MergeOS project funding',
  flow: 'project_funding',
  return_url: 'https://mergeos.shop/paypal/return',
  cancel_url: 'https://mergeos.shop/paypal/cancel',
});
console.log(paypalOrder.order_id, paypalOrder.payment_reference, paypalOrder.approval_url);

await mergeos.createProject(projectPayload);
await mergeos.projectEscrow('prj_0001');
await mergeos.projectPayouts('prj_0001');
await mergeos.projectAutoRelease('prj_0001', {
  task_ids: ['tsk_0001'],
  candidates: [{
    task_id: 'tsk_0001',
    worker_kind: 'human',
    worker_id: 'github:contributor',
    reward_cents: 12000,
    repository: 'acme/repo',
    pull_request_number: 120,
    pull_request_url: 'https://github.com/acme/repo/pull/120',
    pull_request_title: 'Ship accepted work',
    readiness_status: 'ready',
    can_merge: true,
    risk_level: 'low',
    deployment_status: 'not_required',
    validation_signals: ['evidence: provided', 'star: verified'],
    draft: false,
    can_release: true,
  }],
});
await mergeos.projectDashboard('prj_0001');
await mergeos.projectPullRequests('prj_0001');
await mergeos.projectDeployment('prj_0001');
await mergeos.projectAIWorkflow('prj_0001');
await mergeos.createProjectAgentAction('prj_0001', {
  action: 'review',
  agent_type: 'review-agent',
  claimId: 'prj_0001:12',
  bountyId: 'prj_0001:12',
  pull_number: 120,
  reference_url: 'https://github.com/acme/repo/pull/120',
});
await mergeos.recordAgentReview('prj_0001', { pullNumber: 120 });
await mergeos.recordAgentTest('prj_0001', { status: 'processed', labels: ['smoke'] });
await mergeos.recordAgentGeneration('prj_0001', { agentType: 'coding-agent' });
await mergeos.recordAgentScan('prj_0001', { url: 'https://scan.example/report' });
await mergeos.recordDeployment('prj_0001', {
  url: 'https://vercel.example/deployments/mergeos-preview',
  status: 'processed',
  labels: ['preview', 'release-gate'],
});
await mergeos.projectTaskGraph('prj_0001');
await mergeos.projectRouting('prj_0001');
await mergeos.projectWorkflowProtocol('prj_0001');
await mergeos.projectRepositoryScan('prj_0001');
await mergeos.projectRepositoryScanProtocol('prj_0001');
const repoSync = await mergeos.syncProjectRepoIssues('prj_0001');
console.log(repoSync.issue_mappings[0]?.claim_id, repoSync.issue_mappings[0]?.routing?.recommended_next_action);
const suggestedFundingPayload = repositorySuggestedTaskFundingPayload('finding_auth_001', {
  rewardCents: 25000,
  budgetCents: 30000,
  paymentMethod: 'card',
  paymentReference: process.env.MERGEOS_PAYMENT_REFERENCE,
});
const repoTaskOrderPayload = repositorySuggestedTaskPayPalOrderPayload('finding_auth_001', {
  rewardCents: 25000,
  budgetCents: 30000,
  returnURL: 'https://mergeos.shop/paypal/return',
  cancelURL: 'https://mergeos.shop/paypal/cancel',
});
await mergeos.createRepositorySuggestedTaskPayPalOrder('prj_0001', 'finding_auth_001', repoTaskOrderPayload);
const fundedRepoTask = await mergeos.fundRepositorySuggestedTask('prj_0001', 'finding_auth_001', suggestedFundingPayload);
console.log(fundedRepoTask.protocol_version, fundedRepoTask.task_protocol_url, fundedRepoTask.work_packet.claim_endpoint);
const repoTaskAgentAction = agentActionPayloadFromWorkPacket(fundedRepoTask.work_packet, 'scan', {
  status: 'processed',
  referenceURL: 'https://scan.example/report',
});
await mergeos.createProjectAgentAction('prj_0001', repoTaskAgentAction);
await mergeos.listTasks();
const agentClaim = await mergeos.claimTask('prj_0001:12', {
  worker_kind: 'agent',
  worker_id: 'agent:qa-agent',
  agent_type: 'qa-agent',
});
console.log(agentClaim.protocol_version, agentClaim.claim_id);
const submission = await mergeos.submitTask(agentClaim.claim_id, {
  pull_request_url: 'https://github.com/acme/repo/pull/120',
  evidence_url: 'https://vercel.example/deployments/mergeos-preview',
  review_notes: 'Acceptance criteria verified with tests and preview evidence.',
});
console.log(submission.protocol_version, submission.status);
const claim = await mergeos.acceptTask('tsk_0001', {
  worker_kind: 'human',
  worker_id: 'github:contributor',
});
console.log(claim.protocol_version, claim.status);
const proposal = await mergeos.createProposal({
  task_id: 'bounty-prj_0001-12',
  cover_letter: 'I can ship this bounty with tests, PR evidence, and release notes.',
  bid_cents: 12000,
  estimated_hours: 8,
  availability: 'Available this week',
});
console.log(proposal.protocol_version, proposal.proposal.status);
const proposalDecision = await mergeos.decideProposal(proposal.proposal.id, {
  decision: 'accepted',
});
console.log(proposalDecision.protocol_version, proposalDecision.proposal.status);

const marketplace = await mergeos.publicMarketplace();
const bounty = marketplace.bounties.find((row) => row.proposal_packet?.can_claim);
if (bounty) {
  const packetProposal = await mergeos.createProposalFromBounty(bounty, {
    coverLetter: 'I can deliver this public packet with PR evidence and tests.',
    availability: 'Available this week',
  });
  console.log(packetProposal.protocol_version, packetProposal.proposal.status);
}

const dispute = await mergeos.createDispute({
  task_id: 'tsk_0001',
  body: 'Evidence needs maintainer review.',
  severity: 'high',
});
console.log(dispute.protocol_version, dispute.severity);
const worker = await mergeos.workerDashboard();
console.log(workerDashboardProofLinks(worker).map((row) => row.url));
```

## Token Workflow APIs

```js
const claimPayload = airdropClaimPayload({
  missionID: 'mission_delivery_proof',
  walletAddress: '11111111111111111111111111111111',
  allocationMRG: 250,
  workerID: 'github:builder',
  taskReference: 'prj_public_0001:12',
  proofURL: 'https://github.com/acme/repo/pull/12',
  notes: 'Accepted task evidence.',
});
const claim = await mergeos.claimAirdrop(claimPayload);
console.log(claim.kind, claim.claim_id, claim.ledger_entry.entry_hash);

const reservationPayload = presaleReservationPayload({
  tier: 'builder',
  walletAddress: '11111111111111111111111111111111',
  reserveMRG: 25000,
  fundingRail: 'usdc',
  fundingReference: 'usdc:tx_123',
});
const reservation = await mergeos.reservePresale(reservationPayload);
console.log(reservation.kind, reservation.reservation_id, reservation.ledger_proof_url);
```

Both methods require an authenticated user token. They return `mergeos.airdrop-claim.v1` or `mergeos.presale-reservation.v1` documents with a ledger row and public proof URL.

## External Agent Runbook

```js
const runbook = await mergeos.publicAgentRunbook();
console.log(runbook.protocol_version, runbook.supervisor_agent_type, runbook.workflow.length);
```

The default runbook is `/protocol/runbooks/mergeide-agent.v1.json`. It gives MergeIDE, Codex-style coding agents, review agents, QA agents, deployment agents, security agents, and design review subagents a shared public order of operations before claiming funded work.

## Agent Capability Routing

```js
import { agentProtocolAgents, bestAgentForAction } from '@mergeos/sdk';

const agentProtocol = await mergeos.publicProtocolAgents({ limit: 80 });
const deploymentAgent = bestAgentForAction(agentProtocol, 'deploy', {
  capability: 'deployment_validation',
  openOnly: true,
});
const generationAgents = agentProtocolAgents(agentProtocol, { action: 'generate' });
console.log(deploymentAgent?.type, generationAgents.map((agent) => agent.type));
```

`agentProtocolAgents()` filters and ranks public `mergeos.agent.v1` documents by supported action, capability, status, role, type, or open task availability. `bestAgentForAction()` returns the highest-fit active agent for routing review, test, generate, deploy, or scan work.

## Agent Queue Claim

```js
import { agentLeasePayload, agentQueueClaimPayload } from '@mergeos/sdk';

const queue = await mergeos.publicProtocolAgentQueue({ limit: 20 });
const task = queue.tasks.find((row) => row.readiness === 'agent_ready');

if (task) {
  const leasePayload = agentLeasePayload(task);
  const lease = await mergeos.createAgentQueueLease(task, leasePayload);
  await mergeos.heartbeatAgentQueueLease(lease);

  const payload = agentQueueClaimPayload(task, {
    workerID: 'github:mergeos-qa-agent',
    payoutAccount: 'solana:11111111111111111111111111111111',
  });
  const claim = await mergeos.claimAgentQueueTask(task, payload);
  console.log(lease.kind, claim.kind, claim.claim_id, claim.status);
}
```

`createAgentQueueLease(task, overrides)` and `heartbeatAgentQueueLease(lease, overrides)` use the work packet `lease_packet` and return `mergeos.agent-lease.v1`. `claimAgentQueueTask(task, overrides)` prefers the queue row `claim_endpoint`, falls back to the bounty id, and preserves `worker_kind` plus `agent_type` from the public work packet. Use the lease and claim steps before `agentActionPayloadFromWorkPacket()` so the agent records evidence only after it owns the task.

## Marketplace Proposal Packet

```js
import { proposalPacketOutputContracts, proposalPayloadFromBounty } from '@mergeos/sdk';

const marketplace = await mergeos.publicMarketplace();
const bounty = marketplace.bounties.find((row) => row.proposal_packet?.can_claim);

if (bounty) {
  const payload = proposalPayloadFromBounty(bounty, {
    coverLetter: 'I can ship this bounty with tests, PR evidence, and release notes.',
  });
  const contracts = proposalPacketOutputContracts(bounty);
  const proposal = await mergeos.createProposalFromBounty(bounty, payload);
  console.log(proposal.kind, proposal.proposal.status, contracts.map((row) => row.output_protocol));
}
```

Public bounty rows expose `proposal_endpoint`, `proposal_packet.payload`, and `proposal_packet.output_contracts` so contributors, CLIs, and agents can submit proposals without reverse engineering dashboard forms. `proposalPacketOutputContracts(bounty, action?)` returns the proposal, notification, and task protocol artifacts the submit flow will create. `createProposalFromBounty(bounty, overrides)` prefers the packet endpoint, then falls back to `/api/proposals`.

## Agent Work Packet Output Contracts

Agent queue and repository task funding packets expose `work_packet.output_contracts`. Each row tells an external agent which action it is recording, the required output endpoint, the protocol document it produces, and the public URL where evidence should appear. Use `agentWorkPacketOutputContracts(workPacket, action)` to filter contracts for a specific action such as `scan`, `review`, `test`, or `deploy`.

## PR Monitor Auto-Release

```js
import { agentReviewPayloadFromPRMonitorTask, autoReleaseLedgerProofLinksFromResponse, autoReleasePayloadFromPRMonitorTask, autoReleaseProofsFromResponse } from '@mergeos/sdk';

const monitor = await mergeos.projectPullRequests('prj_0001');
const reviewTask = monitor.tasks.find((row) => row.review_packet);
if (reviewTask) {
  const reviewPayload = agentReviewPayloadFromPRMonitorTask(reviewTask);
  const review = await mergeos.createProjectAgentReviewFromPRMonitorTask('prj_0001', reviewTask, reviewPayload);
  console.log(review.kind, review.action, review.status);
}

const task = monitor.tasks.find((row) => row.auto_release_packet?.can_auto_release);

if (task) {
  const payload = autoReleasePayloadFromPRMonitorTask(task);
  const release = await mergeos.projectAutoRelease('prj_0001', payload);
  const proofs = autoReleaseProofsFromResponse(release);
  const proofLinks = autoReleaseLedgerProofLinksFromResponse(release);
  console.log(release.kind, release.released_count, release.payouts.kind, proofs[0]?.ledger_reference, proofLinks[0]?.url);
}
```

`projectAutoReleaseFromPRMonitorTask(projectID, task)` performs the same payload build and POST in one call. The helper prefers the backend-provided `auto_release_packet.payload`, then falls back to a candidate derived from PR readiness, deployment validation signals, worker identity, and reward metadata. Use `autoReleaseProofsFromResponse(response)` after release to read the public PR, deployment validation, policy, and ledger reference proof trail. Use `autoReleaseLedgerProofLinksFromResponse(response)` when an integration only needs the public ledger proof URLs tied to released tasks.

## Deployment Validation Packet

Authenticated deployment documents can include `validation_packet`, a CEO-orchestrated handoff for deployment agents. Use `deploymentValidationPayloadFromDeployment(deployment, overrides)` to build the deploy action payload, or `createDeploymentValidationFromDeployment(projectID, deployment, overrides)` to post it to the packet endpoint in one call. `deploymentValidationOutputContracts(deployment, action?)` returns the deployment evidence and ledger proof contracts the agent is expected to refresh. Public deployment documents omit the packet but still expose `ledger_proof_url`.

`createProjectAgentReviewFromPRMonitorTask(projectID, task)` uses the authenticated `review_packet` from PR monitor rows to record review-agent evidence against `/api/projects/{id}/agent-actions`. The packet includes PR number, review checks, context URLs, runbook steps, and CEO-to-review-agent delegation metadata.

## Admin APIs

```js
await mergeos.adminSummary();
await mergeos.adminUsers();
await mergeos.adminProjects();
await mergeos.adminTasks();
await mergeos.adminTaskPullRequests('tsk_0001');
await mergeos.mergeAdminTaskPullRequest('tsk_0001', 120, {
  worker_id: 'github:contributor',
  reward_mrg: 50,
  bounty_type: 'future-small',
});
await mergeos.adminOpsQueue();
await mergeos.adminDisputes();
await mergeos.adminReputation();
await mergeos.creditMRG({
  worker_id: 'github:contributor',
  reward_mrg: 50,
  bounty_type: 'future-medium',
  pr_url: 'https://github.com/mergeos-bounties/mergeos/pull/120',
});
await mergeos.adminSettings();
await mergeos.adminSSLReviews();
await mergeos.adminGeminiKeys();
await mergeos.adminTestSettings();
await mergeos.updateAdminTestSettings({
  test_mode_enabled: true,
  test_password: 'shared-password',
});
await mergeos.adminTestSettingsEntries();
```

Admin ops queue responses and actions expose `output_contracts` so treasury operators, moderation tooling, and admin subagents can see which protocol artifact each action refreshes or opens. Use `adminOpsQueueOutputContracts(queue, action?)` for the queue-level admin/ledger proof contracts, and `adminOpsActionOutputContracts(actionOrItem, action?)` to inspect a single action or every action attached to a queue item.

```js
const queue = await mergeos.adminOpsQueue();
const queueContracts = adminOpsQueueOutputContracts(queue, 'prove_ledger');
const item = queue.items.find((row) => row.actions?.length);
const contracts = adminOpsActionOutputContracts(item, 'run_ssl_review');
console.log(queueContracts.concat(contracts).map((row) => row.output_protocol));
```

## Event API

```js
const socket = mergeos.connectEvents({ limit: 40, afterID: 'event:latest' });
socket.onmessage = (event) => {
  const message = JSON.parse(event.data);
  const protocolEvent = protocolEventFromMessage(message);
  const protocolEvents = protocolEventsFromMessage(message);
  const protocolType = protocolTypeFromMessage(message);
  if (protocolType) {
    console.log(protocolEventGroup(protocolType), protocolType, protocolEvent || message);
  }
  for (const item of protocolEvents) {
    console.log('protocol event', item.type, item);
  }
};
```

Agents and IDE extensions can also open the stream from discovery metadata so the WebSocket path and protocol version follow `/api/public/protocol`.

```js
const discovery = await mergeos.publicProtocolDiscovery();
const socket = mergeos.connectDiscoveredEvents(discovery, {
  limit: 40,
  cursor: 'event:latest',
});
```

The stream sends `connection_ready` and `live_feed_snapshot` events immediately after connect, then broadcasts live project, PR, payout, token workflow, wallet migration, notification refresh, and ledger events. Pass `afterID`/`cursor` or `since` to `publicLiveFeed`, `publicProtocolEvents`, or `connectEvents` to replay only events newer than the last seen cursor. SDK helpers map live feed records such as `pr_opened`, `agent_action`, `ledger_task_payment`, `ledger_airdrop_claim`, `ledger_presale_reservation`, `ledger_wallet_migration`, `notifications_updated`, and `ledger_manual_credit` to stable protocol events such as `pr.opened`, `agent.tested`, `task.paid`, `airdrop.claimed`, `presale.reserved`, `wallet.migrated`, `notification.updated`, and `ledger.recorded`.

`notifications_updated` is a public-safe signal only. It tells authenticated clients to refresh `/api/notifications` with their own token and does not contain private notification rows, email addresses, or message bodies.

## Solana Contract Helpers

```js
const reference = contractReferenceFromLedger(ledgerEntry, { format: 'bytes' });
const legacyAddressHash = legacyWalletAddressHash('evm', '0xabc...', { format: 'bytes' });
const proofManifest = await mergeos.publicSolanaMRGContractProofManifest();
```

Use these helpers when sending `reference: [u8; 32]` or `legacy_address_hash: [u8; 32]` to the MergeOS Anchor program. They prefer ledger `entry_hash` / `public_hash` and only hash a sanitized reference when no ledger hash exists.

`createWalletMigration()` returns a `mergeos.wallet-migration.v1` document with the Solana target wallet, legacy address hash, PDA seed labels/formats, and `register_legacy_wallet` args for the Anchor program. Decode `contract.args.legacy_address_hash` into 32 raw bytes for the PDA hash seed; do not pass the 64-character hex string as a literal seed.

The public IDL is available at `/contracts/solana/mergeos_mrg.v1.idl.json`. It includes `initializeTreasury`, `mintVerifiedMrg`, `openEscrow`, `releasePayout`, and `registerLegacyWallet`. Map protocol `legacy_chain` strings to the Anchor enum as `trc20 -> LegacyChain::Trc20` and `evm -> LegacyChain::Evm`.

The public proof manifest at `/contracts/solana/mergeos_mrg.proof-manifest.v1.json` maps ledger types such as `token_mint`, `task_reserve`, `task_payment`, and `wallet_migration` to Anchor instruction names, PDA seed formats, SDK helper calls, and public audit sources.

## MergeIDE Release Artifact

```js
const release = await mergeos.publicMergeIDEWindowsRelease();
console.log(release.protocol_version, release.download_url, release.provenance.release_tag);
```

The helper reads `/downloads/mergeide-windows-latest.json`, a `mergeos.release-artifact.v1` document that exposes the Windows exe, GitHub Release page, workflow provenance, digest source URL, and preview-kit fallback for external agents.
