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
  agentActionPayload,
  agentActionEventType,
  contractReferenceFromLedger,
  createMergeOSClient,
  deploymentAgentActionPayload,
  legacyWalletAddressHash,
  protocolEventFromMessage,
  protocolEventsFromMessage,
  protocolEventGroup,
  protocolTypeFromMessage,
  walletMigrationPDASeedMetadata,
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
  const release = await mergeos.projectAutoRelease(projects[0].id, autoReleasePacket.payload);
  console.log(release.kind, release.released_count);
}
const deployment = await mergeos.projectDeployment(projects[0].id);
console.log(deployment.protocol_version, deployment.progress);
const workflow = await mergeos.projectAIWorkflow(projects[0].id);
console.log(workflow.protocol_version, workflow.current_step);
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
const testPayload = agentActionPayload('test', { status: 'processed' });
const graph = await mergeos.projectTaskGraph(projects[0].id);
const routing = await mergeos.projectRouting(projects[0].id);
console.log(routing.protocol_version, routing.stats.ready_count);
const workflowProtocol = await mergeos.projectWorkflowProtocol(projects[0].id);
console.log(workflowProtocol.current_step, workflowProtocol.progress);
const scan = await mergeos.projectRepositoryScan(projects[0].id);
const scanProtocol = await mergeos.projectRepositoryScanProtocol(projects[0].id);
const syncReport = await mergeos.syncProjectRepoIssues(projects[0].id);
console.log(syncReport.protocol_version, syncReport.added_task_count);
const solanaReference = contractReferenceFromLedger({ entry_hash: 'a'.repeat(64) }, { format: 'bytes' });
const legacyHash = legacyWalletAddressHash('trc20', 'TXYZ987654321', { format: 'bytes' });
const pda = walletMigrationPDASeedMetadata('trc20', 'TXYZ987654321');
const migration = await mergeos.createWalletMigration({
  legacy_chain: 'trc20',
  legacy_address: 'TXYZ987654321',
});
console.log(migration.protocol_version, migration.contract.instruction);
```

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
await mergeos.publicProtocolManifest();
await mergeos.publicProtocolTasks({ limit: 80 });
await mergeos.publicProtocolTasks({ taskID: 'prj_public_0001:12' });
await mergeos.publicProtocolAgentQueue({ limit: 80 });
await mergeos.publicProtocolAgents({ limit: 80 });
await mergeos.publicProtocolContributors({ limit: 80 });
await mergeos.publicProtocolLedger();
await mergeos.publicMergeIDEWindowsRelease();
await mergeos.publicProtocolEvents({ limit: 80 });
await mergeos.publicProjectDeployment('prj_public_0001');
await mergeos.publicProjectAIWorkflow('prj_public_0001');
await mergeos.publicProjectWorkflow('prj_public_0001');
await mergeos.publicProjectPullRequests('prj_public_0001');
const repoImport = await mergeos.importRepoIssues({ repo_url: 'https://github.com/acme/repo' });
console.log(repoImport.protocol_version, repoImport.issue_count);
await mergeos.publicTestSettingsStatus();
await mergeos.publicTestSettingsAuth('shared-password');
await mergeos.publicTestSettingsEntries('shared-password');
await mergeos.publicRevealTestSettingsEntry('tse_0001', 'shared-password');
```

## Task and workflow APIs

```js
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
await mergeos.syncProjectRepoIssues('prj_0001');
await mergeos.listTasks();
const agentClaim = await mergeos.claimTask('prj_0001:12', {
  worker_kind: 'agent',
  worker_id: 'agent:qa-agent',
  agent_type: 'qa-agent',
});
console.log(agentClaim.protocol_version, agentClaim.claim_id);
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
const dispute = await mergeos.createDispute({
  task_id: 'tsk_0001',
  body: 'Evidence needs maintainer review.',
  severity: 'high',
});
console.log(dispute.protocol_version, dispute.severity);
await mergeos.workerDashboard();
```

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

## Event API

```js
const socket = mergeos.connectEvents();
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

The stream sends `connection_ready` and `live_feed_snapshot` events immediately after connect, then broadcasts live project, PR, payout, and ledger events. SDK helpers map live feed records such as `pr_opened`, `agent_action`, `ledger_task_payment`, and `ledger_manual_credit` to stable protocol events such as `pr.opened`, `agent.tested`, `task.paid`, and `ledger.recorded`.

## Solana Contract Helpers

```js
const reference = contractReferenceFromLedger(ledgerEntry, { format: 'bytes' });
const legacyAddressHash = legacyWalletAddressHash('evm', '0xabc...', { format: 'bytes' });
```

Use these helpers when sending `reference: [u8; 32]` or `legacy_address_hash: [u8; 32]` to the MergeOS Anchor program. They prefer ledger `entry_hash` / `public_hash` and only hash a sanitized reference when no ledger hash exists.

`createWalletMigration()` returns a `mergeos.wallet-migration.v1` document with the Solana target wallet, legacy address hash, PDA seed labels/formats, and `register_legacy_wallet` args for the Anchor program. Decode `contract.args.legacy_address_hash` into 32 raw bytes for the PDA hash seed; do not pass the 64-character hex string as a literal seed.

The public IDL is available at `/contracts/solana/mergeos_mrg.v1.idl.json`. It includes `initializeTreasury`, `mintVerifiedMrg`, `openEscrow`, `releasePayout`, and `registerLegacyWallet`. Map protocol `legacy_chain` strings to the Anchor enum as `trc20 -> LegacyChain::Trc20` and `evm -> LegacyChain::Evm`.

## MergeIDE Release Artifact

```js
const release = await mergeos.publicMergeIDEWindowsRelease();
console.log(release.protocol_version, release.download_url, release.provenance.release_tag);
```

The helper reads `/downloads/mergeide-windows-latest.json`, a `mergeos.release-artifact.v1` document that exposes the Windows exe, GitHub Release page, workflow provenance, digest source URL, and preview-kit fallback for external agents.
