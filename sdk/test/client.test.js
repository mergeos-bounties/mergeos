import assert from 'node:assert/strict';
import test from 'node:test';
import {
  MergeOSClient,
  airdropClaimPayload,
  agentActionPayloadFromWorkPacket,
  agentReviewPayloadFromPRMonitorTask,
  agentActionPayload,
  agentActionEventType,
  agentActionEventTypes,
  agentLeaseEndpointFromWorkPacket,
  agentLeaseEventType,
  agentLeasePayload,
  agentLeasePacketFromWorkPacket,
  agentQueueClaimPayload,
  agentQueueTaskClaimID,
  agentWorkPacketOutputContracts,
  autoReleasePayloadFromPRMonitorTask,
  autoReleaseProofsFromResponse,
  contractReferenceBytes,
  contractReferenceFromLedger,
  createMergeOSClient,
  deploymentAgentActionPayload,
  deploymentValidationPayloadFromDeployment,
  isAgentActionEventType,
  isLikelySolanaWallet,
  isWorkflowEventType,
  legacyWalletAddressHash,
  liveFeedTypeToProtocolEventType,
  normalizeAgentAction,
  normalizeLegacyChain,
  normalizeLegacyWalletAddress,
  normalizeSolanaWalletAddress,
  presaleReservationPayload,
  proposalPayloadFromBounty,
  protocolEventFromMessage,
  protocolEventsFromMessage,
  protocolEventGroup,
  protocolTypeFromMessage,
  repoPlanningOutputContracts,
  repoPlanningPacket,
  repoPlanningSteps,
  repositorySuggestedTaskFundingPayload,
  repositorySuggestedTaskPayPalOrderPayload,
  routingPacketFromRoute,
  routingPacketOutputContracts,
  routingPacketPayload,
  walletMigrationPDASeedMetadata,
  workflowEventTypes,
} from '../src/index.js';

function fakeFetch(responses = []) {
  const calls = [];
  const fetchImpl = async (url, options = {}) => {
    calls.push({ url, options });
    const next = responses.shift() || { status: 200, body: {} };
    return {
      ok: next.status >= 200 && next.status < 300,
      status: next.status,
      statusText: next.statusText || '',
      text: async () => JSON.stringify(next.body),
    };
  };
  fetchImpl.calls = calls;
  return fetchImpl;
}

test('creates public feed and ledger verification requests without auth', async () => {
  const fetchImpl = fakeFetch([
    { status: 200, body: { items: [] } },
    { status: 200, body: { protocol_version: 'mergeos.protocol.manifest.v1' } },
    { status: 200, body: { tasks: [] } },
    { status: 200, body: { tasks: [{ id: 'prj_1:12' }] } },
    { status: 200, body: { protocol_version: 'mergeos.agent-queue.v1', kind: 'agent_queue', tasks: [] } },
    { status: 200, body: { agents: [] } },
    { status: 200, body: { contributors: [] } },
    { status: 200, body: { protocol_version: 'mergeos.ledger.v1', entries: [] } },
    { status: 200, body: { protocol_version: 'mergeos.release-artifact.v1', kind: 'release_artifact' } },
    { status: 200, body: { protocol_version: 'mergeos.ledger-proof.v1', valid: true } },
    { status: 200, body: { protocol_version: 'mergeos.live-feed.v1', items: [] } },
    { status: 200, body: { protocol_version: 'mergeos.token-economy.v1', totals: {} } },
    { status: 200, body: { protocol_version: 'mergeos.airdrop-missions.v1', kind: 'airdrop_missions', missions: [] } },
    { status: 200, body: { events: [] } },
    { status: 200, body: { protocol_version: 'mergeos.deployment.v1', status: 'validating' } },
    { status: 200, body: { protocol_version: 'mergeos.ai-workflow.v1', status: 'orchestrating' } },
    { status: 200, body: { protocol_version: 'mergeos.workflow.v1', progress: 20, current_step: 'contributor_routing', nodes: [], edges: [] } },
    { status: 200, body: { protocol_version: 'mergeos.scan.v1', kind: 'repository_scan', suggested_tasks: [] } },
    { status: 200, body: { protocol_version: 'mergeos.pr-monitor.v1', stats: { pull_request_count: 1 }, tasks: [] } },
    { status: 200, body: { valid: true } },
  ]);
  const client = new MergeOSClient({
    baseURL: 'https://mergeos.shop/',
    token: 'secret-token',
    fetchImpl,
  });

  const payload = await client.publicLiveFeed({ limit: 80, afterID: 'event:latest', since: '2026-06-06T00:00:00Z' });
  const manifest = await client.publicProtocolManifest();
  const tasks = await client.publicProtocolTasks({ limit: 80 });
  const scopedTasks = await client.publicProtocolTasks({ taskID: 'prj_1:12' });
  const queue = await client.publicProtocolAgentQueue({ limit: 80 });
  const agents = await client.publicProtocolAgents({ limit: 80 });
  const contributors = await client.publicProtocolContributors({ limit: 80 });
  const ledger = await client.publicProtocolLedger();
  const mergeIDERelease = await client.publicMergeIDEWindowsRelease();
  const proof = await client.publicLedgerProof();
  const ledgerEvents = await client.publicLedgerEvents({ limit: 20 });
  const economy = await client.publicTokenEconomy();
  const missions = await client.publicAirdropMissions();
  const events = await client.publicProtocolEvents({ limit: 80, cursor: 'event:latest' });
  const deployment = await client.publicProjectDeployment('prj_public');
  const workflow = await client.publicProjectAIWorkflow('prj_public');
  const workflowGraph = await client.publicProjectWorkflow('prj_public');
  const repositoryScan = await client.publicProjectRepositoryScan('prj_public');
  const pulls = await client.publicProjectPullRequests('prj_public');
  const verification = await client.publicLedgerVerification();

  assert.deepEqual(payload, { items: [] });
  assert.equal(manifest.protocol_version, 'mergeos.protocol.manifest.v1');
  assert.deepEqual(tasks, { tasks: [] });
  assert.equal(scopedTasks.tasks[0].id, 'prj_1:12');
  assert.equal(queue.protocol_version, 'mergeos.agent-queue.v1');
  assert.deepEqual(agents, { agents: [] });
  assert.deepEqual(contributors, { contributors: [] });
  assert.equal(ledger.protocol_version, 'mergeos.ledger.v1');
  assert.equal(mergeIDERelease.protocol_version, 'mergeos.release-artifact.v1');
  assert.equal(proof.protocol_version, 'mergeos.ledger-proof.v1');
  assert.equal(ledgerEvents.protocol_version, 'mergeos.live-feed.v1');
  assert.equal(economy.protocol_version, 'mergeos.token-economy.v1');
  assert.equal(missions.protocol_version, 'mergeos.airdrop-missions.v1');
  assert.deepEqual(events, { events: [] });
  assert.equal(deployment.protocol_version, 'mergeos.deployment.v1');
  assert.equal(workflow.protocol_version, 'mergeos.ai-workflow.v1');
  assert.equal(workflowGraph.protocol_version, 'mergeos.workflow.v1');
  assert.equal(repositoryScan.protocol_version, 'mergeos.scan.v1');
  assert.equal(pulls.protocol_version, 'mergeos.pr-monitor.v1');
  assert.deepEqual(verification, { valid: true });
  assert.equal(fetchImpl.calls[0].url, 'https://mergeos.shop/api/public/live-feed?limit=80&after_id=event%3Alatest&since=2026-06-06T00%3A00%3A00Z');
  assert.equal(fetchImpl.calls[0].options.headers.Authorization, undefined);
  assert.equal(fetchImpl.calls[1].url, 'https://mergeos.shop/api/public/protocol');
  assert.equal(fetchImpl.calls[1].options.headers.Authorization, undefined);
  assert.equal(fetchImpl.calls[2].url, 'https://mergeos.shop/api/public/protocol/tasks?limit=80');
  assert.equal(fetchImpl.calls[2].options.headers.Authorization, undefined);
  assert.equal(fetchImpl.calls[3].url, 'https://mergeos.shop/api/public/protocol/tasks?task_id=prj_1%3A12');
  assert.equal(fetchImpl.calls[3].options.headers.Authorization, undefined);
  assert.equal(fetchImpl.calls[4].url, 'https://mergeos.shop/api/public/protocol/agent-queue?limit=80');
  assert.equal(fetchImpl.calls[4].options.headers.Authorization, undefined);
  assert.equal(fetchImpl.calls[5].url, 'https://mergeos.shop/api/public/protocol/agents?limit=80');
  assert.equal(fetchImpl.calls[5].options.headers.Authorization, undefined);
  assert.equal(fetchImpl.calls[6].url, 'https://mergeos.shop/api/public/protocol/contributors?limit=80');
  assert.equal(fetchImpl.calls[6].options.headers.Authorization, undefined);
  assert.equal(fetchImpl.calls[7].url, 'https://mergeos.shop/api/public/protocol/ledger');
  assert.equal(fetchImpl.calls[7].options.headers.Authorization, undefined);
  assert.equal(fetchImpl.calls[8].url, 'https://mergeos.shop/downloads/mergeide-windows-latest.json');
  assert.equal(fetchImpl.calls[8].options.headers.Authorization, undefined);
  assert.equal(fetchImpl.calls[9].url, 'https://mergeos.shop/api/public/ledger/proof');
  assert.equal(fetchImpl.calls[9].options.headers.Authorization, undefined);
  assert.equal(fetchImpl.calls[10].url, 'https://mergeos.shop/api/public/ledger/events?limit=20');
  assert.equal(fetchImpl.calls[10].options.headers.Authorization, undefined);
  assert.equal(fetchImpl.calls[11].url, 'https://mergeos.shop/api/public/token-economy');
  assert.equal(fetchImpl.calls[11].options.headers.Authorization, undefined);
  assert.equal(fetchImpl.calls[12].url, 'https://mergeos.shop/api/public/airdrop/missions');
  assert.equal(fetchImpl.calls[12].options.headers.Authorization, undefined);
  assert.equal(fetchImpl.calls[13].url, 'https://mergeos.shop/api/public/protocol/events?limit=80&after_id=event%3Alatest');
  assert.equal(fetchImpl.calls[13].options.headers.Authorization, undefined);
  assert.equal(fetchImpl.calls[14].url, 'https://mergeos.shop/api/public/projects/prj_public/deployment');
  assert.equal(fetchImpl.calls[14].options.headers.Authorization, undefined);
  assert.equal(fetchImpl.calls[15].url, 'https://mergeos.shop/api/public/projects/prj_public/ai-workflow');
  assert.equal(fetchImpl.calls[15].options.headers.Authorization, undefined);
  assert.equal(fetchImpl.calls[16].url, 'https://mergeos.shop/api/public/projects/prj_public/workflow');
  assert.equal(fetchImpl.calls[16].options.headers.Authorization, undefined);
  assert.equal(fetchImpl.calls[17].url, 'https://mergeos.shop/api/public/projects/prj_public/repo-scan');
  assert.equal(fetchImpl.calls[17].options.headers.Authorization, undefined);
  assert.equal(fetchImpl.calls[18].url, 'https://mergeos.shop/api/public/projects/prj_public/pull-requests');
  assert.equal(fetchImpl.calls[18].options.headers.Authorization, undefined);
  assert.equal(fetchImpl.calls[19].url, 'https://mergeos.shop/api/public/ledger/verify');
  assert.equal(fetchImpl.calls[19].options.headers.Authorization, undefined);
});

test('derives Solana ledger references and legacy wallet hashes for operators', () => {
  const entryHash = 'B'.repeat(64);
  const ledgerEntry = {
    sequence: 11,
    type: 'manual_credit',
    reference: 'pr:https://github.com/mergeos-bounties/mergeos/pull/151',
    entry_hash: entryHash,
  };

  assert.equal(contractReferenceFromLedger(ledgerEntry), entryHash.toLowerCase());
  assert.equal(contractReferenceFromLedger(`0x${entryHash}`), entryHash.toLowerCase());
  assert.deepEqual(contractReferenceBytes(ledgerEntry), Array(32).fill(187));
  assert.equal(contractReferenceFromLedger(ledgerEntry, { format: '0x' }), `0x${entryHash.toLowerCase()}`);

  const fallback = contractReferenceFromLedger({ reference: ledgerEntry.reference });
  assert.match(fallback, /^[0-9a-f]{64}$/);
  assert.equal(fallback, contractReferenceFromLedger({ reference: ledgerEntry.reference }));
  assert.notEqual(fallback, entryHash.toLowerCase());

  assert.equal(normalizeLegacyChain('TRON'), 'trc20');
  assert.equal(normalizeLegacyChain('ethereum'), 'evm');
  assert.equal(legacyWalletAddressHash('tron', 'TXYZ987654321'), legacyWalletAddressHash('trc20', 'txyz987654321'));
  assert.equal(legacyWalletAddressHash('tron', 'tron:TXYZ987654321'), legacyWalletAddressHash('trc20', 'txyz987654321'));
  assert.equal(normalizeLegacyWalletAddress('eip155:0xAbC0000000000000000000000000000000000000'), '0xabc0000000000000000000000000000000000000');
  assert.equal(legacyWalletAddressHash('evm', '0xAbC0000000000000000000000000000000000000', { format: 'bytes' }).length, 32);
  const pda = walletMigrationPDASeedMetadata('tron', 'tron:TXYZ987654321');
  assert.deepEqual(pda.pda_seeds, ['wallet-migration', 'trc20', 'legacy_address_hash_bytes']);
  assert.equal(pda.legacy_address_hash_bytes.length, 32);
});

test('loads public external agent runbook without auth', async () => {
  const fetchImpl = fakeFetch([
    {
      status: 200,
      body: {
        protocol_version: 'mergeos.agent-runbook.v1',
        kind: 'agent_runbook',
        id: 'mergeide-agent.v1',
      },
    },
  ]);
  const client = new MergeOSClient({
    baseURL: 'https://mergeos.shop',
    token: 'secret-token',
    fetchImpl,
  });

  const runbook = await client.publicAgentRunbook();

  assert.equal(runbook.protocol_version, 'mergeos.agent-runbook.v1');
  assert.equal(fetchImpl.calls[0].url, 'https://mergeos.shop/protocol/runbooks/mergeide-agent.v1.json');
  assert.equal(fetchImpl.calls[0].options.headers.Authorization, undefined);
});

test('builds and sends claim-safe agent queue task payloads', async () => {
  const queueTask = {
    id: 'prj_1:12',
    bounty_id: 'prj_1:12',
    worker_kind: 'agent',
    agent_type: 'qa-agent',
    claim_endpoint: '/api/tasks/prj_1:12/claim',
    work_packet: {
      claim_endpoint: '/api/tasks/prj_1:12/claim',
      subagent_type: 'qa-agent',
      lease_packet: {
        lease_endpoint: '/api/agent-queue/leases',
        heartbeat_endpoint: '/api/agent-queue/leases',
        method: 'POST',
        ttl_seconds: 900,
        heartbeat_seconds: 120,
        payload: {
          claim_id: 'prj_1:12',
          bounty_id: 'prj_1:12',
          agent_type: 'qa-agent',
          status: 'leased',
        },
      },
    },
  };
  const payload = agentQueueClaimPayload(queueTask, {
    workerID: 'github:mergeos-qa-agent',
    payoutAccount: 'solana:11111111111111111111111111111111',
  });

  assert.deepEqual(payload, {
    worker_kind: 'agent',
    worker_id: 'github:mergeos-qa-agent',
    agent_type: 'qa-agent',
    payout_account: 'solana:11111111111111111111111111111111',
  });
  assert.equal(agentQueueTaskClaimID(queueTask), 'prj_1:12');
  assert.equal(agentQueueTaskClaimID({}, '/api/tasks/prj_2%3A7/claim'), 'prj_2:7');

  const fetchImpl = fakeFetch([
    { status: 200, body: { protocol_version: 'mergeos.task-claim.v1', kind: 'task_claim', claim_id: 'prj_1:12', status: 'claimed' } },
    { status: 200, body: { protocol_version: 'mergeos.task-claim.v1', kind: 'task_claim', claim_id: 'prj_2:7', status: 'claimed' } },
    { status: 201, body: { protocol_version: 'mergeos.agent-lease.v1', kind: 'agent_lease', lease_id: 'agl_1', claim_id: 'prj_1:12', status: 'leased' } },
    { status: 200, body: { protocol_version: 'mergeos.agent-lease.v1', kind: 'agent_lease', lease_id: 'agl_1', claim_id: 'prj_1:12', status: 'heartbeat' } },
  ]);
  const client = new MergeOSClient({ token: 'agent-token', fetchImpl });
  const claimed = await client.claimAgentQueueTask(queueTask, { workerID: 'github:mergeos-qa-agent' });
  const fallbackClaim = await client.claimAgentQueueTask({
    bounty_id: 'prj_2:7',
    worker_kind: 'hybrid',
    work_packet: { subagent_type: 'review-agent' },
  }, { workerID: 'github:hybrid-reviewer' });
  const leasePayload = agentLeasePayload(queueTask);
  const lease = await client.createAgentQueueLease(queueTask);
  const heartbeat = await client.heartbeatAgentQueueLease(lease, { agentType: 'qa-agent' });

  assert.equal(claimed.kind, 'task_claim');
  assert.equal(fallbackClaim.claim_id, 'prj_2:7');
  assert.equal(lease.kind, 'agent_lease');
  assert.equal(heartbeat.status, 'heartbeat');
  assert.equal(agentLeaseEndpointFromWorkPacket(queueTask), '/api/agent-queue/leases');
  assert.equal(agentLeasePacketFromWorkPacket(queueTask).ttl_seconds, 900);
  assert.deepEqual(leasePayload, {
    claim_id: 'prj_1:12',
    bounty_id: 'prj_1:12',
    agent_type: 'qa-agent',
    status: 'leased',
  });
  assert.equal(fetchImpl.calls[0].url, '/api/tasks/prj_1:12/claim');
  assert.equal(fetchImpl.calls[0].options.method, 'POST');
  assert.equal(fetchImpl.calls[0].options.headers.Authorization, 'Bearer agent-token');
  assert.equal(fetchImpl.calls[0].options.body, JSON.stringify({
    worker_kind: 'agent',
    worker_id: 'github:mergeos-qa-agent',
    agent_type: 'qa-agent',
  }));
  assert.equal(fetchImpl.calls[1].url, '/api/tasks/prj_2%3A7/claim');
  assert.equal(fetchImpl.calls[1].options.body, JSON.stringify({
    worker_kind: 'hybrid',
    worker_id: 'github:hybrid-reviewer',
    agent_type: 'review-agent',
  }));
  assert.equal(fetchImpl.calls[2].url, '/api/agent-queue/leases');
  assert.equal(fetchImpl.calls[2].options.body, JSON.stringify(leasePayload));
  assert.equal(fetchImpl.calls[3].url, '/api/agent-queue/leases');
  assert.equal(fetchImpl.calls[3].options.body, JSON.stringify({
    lease_id: 'agl_1',
    claim_id: 'prj_1:12',
    bounty_id: 'prj_1:12',
    agent_type: 'qa-agent',
    status: 'heartbeat',
  }));
});

test('executes project routing packets for agent leases and contributor proposals', async () => {
  const agentRoute = {
    claim_id: 'prj_1:12',
    recommended_next_action: 'route_to_agent',
    routing_packet: {
      action: 'route_to_agent',
      method: 'POST',
      endpoint: '/api/agent-queue/leases',
      payload: {
        claim_id: 'prj_1:12',
        bounty_id: 'prj_1:12',
        agent_type: 'qa-agent',
        status: 'leased',
      },
      output_contracts: [
        {
          action: 'lease',
          artifact_kind: 'agent_lease',
          output_endpoint: '/api/agent-queue/leases',
          output_protocol: 'mergeos.agent-lease.v1',
          output_protocol_url: '/protocol/agent-lease.v1.schema.json',
        },
      ],
    },
  };
  const proposalRoute = {
    claim_id: 'prj_1:13',
    recommended_next_action: 'invite_contributor',
    reward_cents: 7000,
    routing_packet: {
      action: 'invite_contributor',
      method: 'POST',
      endpoint: '/api/proposals',
      payload: {
        task_id: 'prj_1:13',
        cover_letter: 'I can deliver this route with tests.',
        bid_cents: 7000,
        estimated_hours: 5,
      },
      output_contracts: [
        {
          action: 'propose',
          artifact_kind: 'worker_proposal',
          output_endpoint: '/api/proposals',
          output_protocol: 'mergeos.proposal.v1',
          output_protocol_url: '/protocol/proposal.v1.schema.json',
        },
      ],
    },
  };
  const fetchImpl = fakeFetch([
    { status: 201, body: { protocol_version: 'mergeos.agent-lease.v1', kind: 'agent_lease', claim_id: 'prj_1:12' } },
    { status: 201, body: { protocol_version: 'mergeos.proposal.v1', kind: 'proposal', proposal: { status: 'submitted' } } },
  ]);
  const client = new MergeOSClient({ token: 'routing-token', fetchImpl });

  const agentPacket = routingPacketFromRoute(agentRoute);
  const agentPayload = routingPacketPayload(agentRoute, { status: 'heartbeat' });
  const agentContracts = routingPacketOutputContracts(agentRoute, 'lease');
  const lease = await client.executeRoutingPacket(agentRoute, { status: 'heartbeat' });
  const proposal = await client.executeRoutingPacket(proposalRoute, { availability: 'Available Monday' });

  assert.equal(agentPacket.endpoint, '/api/agent-queue/leases');
  assert.deepEqual(agentPayload, {
    claim_id: 'prj_1:12',
    bounty_id: 'prj_1:12',
    agent_type: 'qa-agent',
    status: 'heartbeat',
  });
  assert.equal(agentContracts[0].output_protocol, 'mergeos.agent-lease.v1');
  assert.equal(lease.kind, 'agent_lease');
  assert.equal(proposal.proposal.status, 'submitted');
  assert.equal(fetchImpl.calls[0].url, '/api/agent-queue/leases');
  assert.equal(fetchImpl.calls[0].options.headers.Authorization, 'Bearer routing-token');
  assert.equal(fetchImpl.calls[0].options.body, JSON.stringify(agentPayload));
  assert.equal(fetchImpl.calls[1].url, '/api/proposals');
  assert.equal(fetchImpl.calls[1].options.body, JSON.stringify({
    task_id: 'prj_1:13',
    cover_letter: 'I can deliver this route with tests.',
    bid_cents: 7000,
    estimated_hours: 5,
    availability: 'Available Monday',
  }));
});

test('maps typed agent action event protocol values', () => {
  assert.equal(agentActionEventTypes.test, 'agent.tested');
  assert.equal(agentActionEventType('review'), 'agent.reviewed');
  assert.equal(agentActionEventType('TEST'), 'agent.tested');
  assert.equal(agentActionEventType('generate'), 'agent.generated');
  assert.equal(agentActionEventType('deploy'), 'agent.deployed');
  assert.equal(agentActionEventType('scan'), 'agent.scanned');
  assert.equal(agentActionEventType('unknown'), 'agent.action');
  assert.equal(agentLeaseEventType('leased'), 'agent.leased');
  assert.equal(agentLeaseEventType('heartbeat'), 'agent.heartbeat');
  assert.equal(agentLeaseEventType('released'), 'agent.released');
  assert.equal(isAgentActionEventType('agent.generated'), true);
  assert.equal(isAgentActionEventType('agent.heartbeat'), true);
  assert.equal(isAgentActionEventType('agent.action'), true);
  assert.equal(isAgentActionEventType('task.paid'), false);
});

test('builds deployment agent action payloads for deployment evidence', async () => {
  const payload = deploymentAgentActionPayload({
    url: 'https://vercel.example/deployments/mergeos-preview',
    status: 'processed',
    durationMillis: 42000,
    pullNumber: 120,
    labels: ['preview', 'release-gate'],
  });
  assert.deepEqual(payload, {
    action: 'deploy',
    agent_type: 'deployment-agent',
    status: 'processed',
    reference_url: 'https://vercel.example/deployments/mergeos-preview',
    duration_millis: 42000,
    pull_number: 120,
    labels: ['preview', 'release-gate'],
    context_urls: [],
    evidence: [],
    runbook: [],
    checks: [],
  });

  const fetchImpl = fakeFetch([
    { status: 201, body: { protocol_version: 'mergeos.agent-action.v1', kind: 'agent_action', log: { event_name: 'agent_action', action: 'deploy' } } },
  ]);
  const client = new MergeOSClient({ token: 'agent-token', fetchImpl });
  const response = await client.recordDeployment('prj_1', { deploymentURL: payload.reference_url });

  assert.equal(response.protocol_version, 'mergeos.agent-action.v1');
  assert.equal(response.kind, 'agent_action');
  assert.equal(response.log.action, 'deploy');
  assert.equal(fetchImpl.calls[0].url, '/api/projects/prj_1/agent-actions');
  assert.equal(fetchImpl.calls[0].options.method, 'POST');
  assert.equal(fetchImpl.calls[0].options.headers.Authorization, 'Bearer agent-token');
  assert.equal(fetchImpl.calls[0].options.body, JSON.stringify({
    action: 'deploy',
    agent_type: 'deployment-agent',
    status: 'processed',
    reference_url: 'https://vercel.example/deployments/mergeos-preview',
    duration_millis: 0,
    pull_number: 0,
    labels: [],
    context_urls: [],
    evidence: [],
    runbook: [],
    checks: [],
  }));
});

test('builds and sends deployment validation payloads from deployment packets', async () => {
  const deployment = {
    protocol_version: 'mergeos.deployment.v1',
    validation_packet: {
      status: 'needs_validation',
      validation_endpoint: '/api/projects/prj_1/agent-actions',
      target_stage: {
        id: 'deployment_handoff',
        issue: 13,
        url: 'https://vercel.example/deployments/mergeos-preview',
      },
      context_urls: {
        deployment: '/api/projects/prj_1/deployment',
        payouts: '/api/projects/prj_1/payouts',
      },
      runbook: [
        { step: 1, action: 'inspect_deployment', endpoint: '/api/projects/prj_1/deployment' },
      ],
      payload: {
        action: 'deploy',
        bounty_id: 'prj_1:13',
        agent_type: 'deployment-agent',
        delegated_by: 'ceo-strategy-agent',
        subagent_type: 'deployment-agent',
        status: 'processed',
        reference_url: 'https://vercel.example/deployments/mergeos-preview',
        context_urls: ['/api/projects/prj_1/deployment'],
        evidence: ['deployment_handoff'],
        checks: [{ name: 'deployment_handoff', status: 'complete', summary: 'Preview linked.' }],
        delegation_chain: ['ceo-strategy-agent', 'deployment-agent'],
      },
    },
  };
  const payload = deploymentValidationPayloadFromDeployment(deployment);
  assert.equal(payload.action, 'deploy');
  assert.equal(payload.bounty_id, 'prj_1:13');
  assert.equal(payload.claim_id, undefined);
  assert.equal(payload.reference_url, 'https://vercel.example/deployments/mergeos-preview');
  assert.deepEqual(payload.context_urls, ['/api/projects/prj_1/deployment']);
  assert.deepEqual(payload.delegation_chain, ['ceo-strategy-agent', 'deployment-agent']);

  const fetchImpl = fakeFetch([
    { status: 201, body: { protocol_version: 'mergeos.agent-action.v1', kind: 'agent_action', log: { event_name: 'agent_action', action: 'deploy' } } },
  ]);
  const client = new MergeOSClient({ token: 'agent-token', fetchImpl });
  const response = await client.createDeploymentValidationFromDeployment('prj_1', deployment);

  assert.equal(response.kind, 'agent_action');
  assert.equal(fetchImpl.calls[0].url, '/api/projects/prj_1/agent-actions');
  assert.equal(fetchImpl.calls[0].options.method, 'POST');
  assert.equal(fetchImpl.calls[0].options.headers.Authorization, 'Bearer agent-token');
  assert.equal(fetchImpl.calls[0].options.body, JSON.stringify(payload));
});

test('builds and sends typed AI agent action helpers', async () => {
  assert.equal(normalizeAgentAction('gen'), 'generate');
  assert.equal(normalizeAgentAction('bad'), 'review');
  assert.deepEqual(agentActionPayload('test', {
    referenceURL: 'https://github.com/acme/repo/pull/12',
    status: 'running',
    pullNumber: 12,
    claimId: 'claim_public_1',
    bountyId: 'prj_1:12',
    labels: ['smoke'],
    contextURLs: ['https://mergeos.shop/api/public/projects/prj_1/workflow'],
    evidence: ['Smoke tests passed'],
    runbook: ['Fetch task packet', 'Run smoke suite'],
    checks: [
      { name: 'Smoke suite', status: 'passed', summary: 'Preview route passed.' },
    ],
    delegatedBy: 'ceo-strategy-agent',
    designAgent: 'design-review-agent',
    subagentType: 'qa-agent',
    delegationChain: ['ceo-strategy-agent', 'design-review-agent', 'qa-agent'],
  }), {
    action: 'test',
    agent_type: 'qa-agent',
    status: 'running',
    reference_url: 'https://github.com/acme/repo/pull/12',
    duration_millis: 0,
    pull_number: 12,
    claim_id: 'claim_public_1',
    bounty_id: 'prj_1:12',
    labels: ['smoke'],
    context_urls: ['https://mergeos.shop/api/public/projects/prj_1/workflow'],
    evidence: ['Smoke tests passed'],
    runbook: ['Fetch task packet', 'Run smoke suite'],
    checks: [
      { name: 'Smoke suite', status: 'passed', summary: 'Preview route passed.' },
    ],
    delegated_by: 'ceo-strategy-agent',
    design_agent: 'design-review-agent',
    subagent_type: 'qa-agent',
    delegation_chain: ['ceo-strategy-agent', 'design-review-agent', 'qa-agent'],
  });

  const fetchImpl = fakeFetch([
    { status: 201, body: { log: { action: 'review' } } },
    { status: 201, body: { log: { action: 'test' } } },
    { status: 201, body: { log: { action: 'generate' } } },
    { status: 201, body: { log: { action: 'scan' } } },
  ]);
  const client = new MergeOSClient({ token: 'agent-token', fetchImpl });
  await client.recordAgentReview('prj_1', {
    pullNumber: 10,
    claimId: 'claim_public_2',
    bountyId: 'prj_1:14',
    delegatedBy: 'ceo-strategy-agent',
    designAgent: 'design-review-agent',
    subagentType: 'review-agent',
    delegationChain: ['ceo-strategy-agent', 'design-review-agent', 'review-agent'],
  });
  await client.recordAgentTest('prj_1', { status: 'running' });
  await client.recordAgentGeneration('prj_1', { agentType: 'code-agent' });
  await client.recordAgentScan('prj_1', { url: 'https://scan.example/report' });

  assert.equal(fetchImpl.calls.length, 4);
  assert.equal(fetchImpl.calls[0].options.body, JSON.stringify({
    action: 'review',
    agent_type: 'review-agent',
    status: 'processed',
    reference_url: '',
    duration_millis: 0,
    pull_number: 10,
    labels: [],
    context_urls: [],
    evidence: [],
    runbook: [],
    checks: [],
    delegated_by: 'ceo-strategy-agent',
    design_agent: 'design-review-agent',
    subagent_type: 'review-agent',
    delegation_chain: ['ceo-strategy-agent', 'design-review-agent', 'review-agent'],
    claim_id: 'claim_public_2',
    bounty_id: 'prj_1:14',
  }));
  assert.equal(fetchImpl.calls[1].options.body, JSON.stringify({
    action: 'test',
    agent_type: 'qa-agent',
    status: 'running',
    reference_url: '',
    duration_millis: 0,
    pull_number: 0,
    labels: [],
    context_urls: [],
    evidence: [],
    runbook: [],
    checks: [],
  }));
  assert.equal(fetchImpl.calls[2].options.body, JSON.stringify({
    action: 'generate',
    agent_type: 'code-agent',
    status: 'processed',
    reference_url: '',
    duration_millis: 0,
    pull_number: 0,
    labels: [],
    context_urls: [],
    evidence: [],
    runbook: [],
    checks: [],
  }));
  assert.equal(fetchImpl.calls[3].options.body, JSON.stringify({
    action: 'scan',
    agent_type: 'scan-agent',
    status: 'processed',
    reference_url: 'https://scan.example/report',
    duration_millis: 0,
    pull_number: 0,
    labels: [],
    context_urls: [],
    evidence: [],
    runbook: [],
    checks: [],
  }));
});

test('normalizes direct agent action claim identifiers', async () => {
  const fetchImpl = fakeFetch([
    { status: 201, body: { protocol_version: 'mergeos.agent-action.v1', kind: 'agent_action', claim_id: 'claim_public_3' } },
  ]);
  const client = new MergeOSClient({ token: 'agent-token', fetchImpl });
  const response = await client.createProjectAgentAction('prj_1', {
    action: 'test',
    agentType: 'qa-agent',
    claimId: 'claim_public_3',
    bountyId: 'prj_1:15',
  });

  assert.equal(response.claim_id, 'claim_public_3');
  assert.equal(fetchImpl.calls[0].options.body, JSON.stringify({
    action: 'test',
    agent_type: 'qa-agent',
    status: 'processed',
    reference_url: '',
    duration_millis: 0,
    pull_number: 0,
    labels: [],
    context_urls: [],
    evidence: [],
    runbook: [],
    checks: [],
    claim_id: 'claim_public_3',
    bounty_id: 'prj_1:15',
  }));
});

test('loads runtime config without auth for payment rail discovery', async () => {
  const fetchImpl = fakeFetch([
    { status: 200, body: { payment_rails: [{ id: 'paypal', enabled: true }, { id: 'stripe', enabled: false }] } },
  ]);
  const client = new MergeOSClient({ token: 'secret-token', fetchImpl });

  const config = await client.runtimeConfig();

  assert.equal(fetchImpl.calls[0].url, '/api/config');
  assert.equal(fetchImpl.calls[0].options.headers.Authorization, undefined);
  assert.equal(config.payment_rails.length, 2);
  assert.equal(config.payment_rails[1].id, 'stripe');
});

test('maps live feed records to workflow event protocol values', () => {
  const protocolEvent = {
    protocol_version: 'mergeos.event.v1',
    kind: 'event',
    type: 'task.paid',
    payload: { ledger_sequence: 7 },
  };

  assert.equal(workflowEventTypes.prOpened, 'pr.opened');
  assert.equal(liveFeedTypeToProtocolEventType('project_funded'), 'project.funded');
  assert.equal(liveFeedTypeToProtocolEventType('task_opened'), 'task.created');
  assert.equal(liveFeedTypeToProtocolEventType('task_claimed'), 'task.claimed');
  assert.equal(liveFeedTypeToProtocolEventType('task_submitted'), 'task.submitted');
  assert.equal(liveFeedTypeToProtocolEventType('task_changes_requested'), 'task.changes_requested');
  assert.equal(liveFeedTypeToProtocolEventType('task_accepted'), 'task.accepted');
  assert.equal(liveFeedTypeToProtocolEventType('pr_opened'), 'pr.opened');
  assert.equal(liveFeedTypeToProtocolEventType('ai_review'), 'pr.reviewed');
  assert.equal(liveFeedTypeToProtocolEventType('deployment_validation'), 'deployment.updated');
  assert.equal(liveFeedTypeToProtocolEventType('repo_issues_synced'), 'repo.issues.synced');
  assert.equal(liveFeedTypeToProtocolEventType('proposal_submitted'), 'proposal.submitted');
  assert.equal(liveFeedTypeToProtocolEventType('proposal_accepted'), 'proposal.accepted');
  assert.equal(liveFeedTypeToProtocolEventType('proposal_declined'), 'proposal.declined');
  assert.equal(liveFeedTypeToProtocolEventType('ledger_task_payment'), 'task.paid');
  assert.equal(liveFeedTypeToProtocolEventType('ledger_airdrop_claim'), 'airdrop.claimed');
  assert.equal(liveFeedTypeToProtocolEventType('ledger_presale_reservation'), 'presale.reserved');
  assert.equal(liveFeedTypeToProtocolEventType('ledger_wallet_migration'), 'wallet.migrated');
  assert.equal(liveFeedTypeToProtocolEventType('ledger_manual_credit'), 'ledger.recorded');
  assert.equal(liveFeedTypeToProtocolEventType('agent_action', 'test'), 'agent.tested');
  assert.equal(liveFeedTypeToProtocolEventType('agent_lease', 'heartbeat'), 'agent.heartbeat');
  assert.equal(liveFeedTypeToProtocolEventType('unknown'), 'agent.action');
  assert.equal(protocolEventFromMessage({ event: protocolEvent }), protocolEvent);
  assert.equal(protocolEventFromMessage({ type: 'ledger_manual_credit' }), null);
  assert.deepEqual(protocolEventsFromMessage({ event: protocolEvent }), [protocolEvent]);
  assert.deepEqual(protocolEventsFromMessage({ events: { events: [protocolEvent, null, 'bad'] } }), [protocolEvent]);
  assert.deepEqual(protocolEventsFromMessage({ type: 'connection_ready' }), []);
  assert.equal(protocolTypeFromMessage({ event: protocolEvent, protocol_type: 'ledger.recorded' }), 'task.paid');
  assert.equal(protocolTypeFromMessage({ protocol_type: 'ledger.recorded', type: 'ledger_task_payment' }), 'ledger.recorded');
  assert.equal(protocolTypeFromMessage({ type: 'ledger_manual_credit' }), 'ledger.recorded');
  assert.equal(protocolTypeFromMessage({ type: 'ledger_wallet_migration' }), 'wallet.migrated');
  assert.equal(protocolTypeFromMessage({ type: 'connection_ready' }), '');
  assert.equal(protocolTypeFromMessage({ type: 'realtime_ready' }), '');
  assert.equal(protocolTypeFromMessage({ type: 'live_feed_snapshot', events: { events: [protocolEvent] } }), '');
  assert.equal(protocolTypeFromMessage({ type: 'realtime_snapshot' }), '');
  assert.equal(protocolTypeFromMessage({ type: 'realtime_heartbeat' }), '');
  assert.equal(protocolTypeFromMessage({ type: 'admin_ops_updated' }), '');
  assert.equal(protocolEventGroup('pr.opened'), 'pull_request');
  assert.equal(protocolEventGroup('task.paid'), 'task');
  assert.equal(protocolEventGroup('proposal.accepted'), 'proposal');
  assert.equal(protocolEventGroup('agent.tested'), 'agent');
  assert.equal(protocolEventGroup('agent.leased'), 'agent');
  assert.equal(protocolEventGroup('repo.issues.synced'), 'repository');
  assert.equal(protocolEventGroup('airdrop.claimed'), 'token');
  assert.equal(protocolEventGroup('presale.reserved'), 'token');
  assert.equal(protocolEventGroup('wallet.migrated'), 'wallet');
  assert.equal(isWorkflowEventType('deployment.updated'), true);
  assert.equal(isWorkflowEventType('airdrop.claimed'), true);
  assert.equal(isWorkflowEventType('presale.reserved'), true);
  assert.equal(isWorkflowEventType('wallet.migrated'), true);
  assert.equal(isWorkflowEventType('agent.scanned'), true);
  assert.equal(isWorkflowEventType('agent.released'), true);
  assert.equal(isWorkflowEventType('unknown.event'), false);
});

test('exposes public repo import and password-gated test settings without auth', async () => {
  const fetchImpl = fakeFetch([
    { status: 200, body: { protocol_version: 'mergeos.repo-import.v1', issue_count: 2 } },
    { status: 200, body: { test_mode_enabled: true } },
    { status: 200, body: { authenticated: true } },
    { status: 200, body: [{ id: 'tse_1' }] },
    { status: 201, body: { id: 'tse_2' } },
    { status: 200, body: { id: 'tse_2', status: 'disabled' } },
    { status: 200, body: { ok: true } },
    { status: 200, body: { id: 'tse_2', setting_value: 'secret-value' } },
  ]);
  const client = new MergeOSClient({ token: 'secret-token', fetchImpl });

  const imported = await client.importRepoIssues({ repo_url: 'https://github.com/acme/repo' });
  await client.publicTestSettingsStatus();
  await client.publicTestSettingsAuth('pw');
  await client.publicTestSettingsEntries('pw');
  await client.publicAddTestSettingsEntry('pw', {
    integration_type: 'llm',
    setting_key: 'TASK_LLM_TEST_KEY',
    setting_value: 'value',
  });
  await client.publicUpdateTestSettingsEntry('tse_2', 'pw', { status: 'disabled' });
  await client.publicDeleteTestSettingsEntry('tse_2', 'pw');
  const revealed = await client.publicRevealTestSettingsEntry('tse_2', 'pw');

  assert.equal(fetchImpl.calls[0].url, '/api/public/repo/issues');
  assert.equal(imported.protocol_version, 'mergeos.repo-import.v1');
  assert.equal(imported.issue_count, 2);
  assert.equal(fetchImpl.calls[1].url, '/api/public/test-settings/status');
  assert.equal(fetchImpl.calls[2].url, '/api/public/test-settings/auth');
  assert.equal(fetchImpl.calls[3].url, '/api/public/test-settings/entries/list');
  assert.equal(fetchImpl.calls[4].url, '/api/public/test-settings/entries');
  assert.equal(fetchImpl.calls[5].url, '/api/public/test-settings/entries/tse_2');
  assert.equal(fetchImpl.calls[5].options.method, 'PATCH');
  assert.equal(fetchImpl.calls[6].options.method, 'DELETE');
  assert.equal(fetchImpl.calls[6].options.headers.Authorization, undefined);
  assert.equal(fetchImpl.calls[7].url, '/api/public/test-settings/entries/tse_2/reveal');
  assert.equal(fetchImpl.calls[7].options.method, 'POST');
  assert.equal(fetchImpl.calls[7].options.headers.Authorization, undefined);
  assert.equal(revealed.setting_value, 'secret-value');
  assert.equal(fetchImpl.calls[4].options.body, JSON.stringify({
    password: 'pw',
    integration_type: 'llm',
    setting_key: 'TASK_LLM_TEST_KEY',
    setting_value: 'value',
  }));
});

test('sends bearer token and JSON body for task acceptance, proposals, and disputes', async () => {
  const fetchImpl = fakeFetch([
    { status: 200, body: { protocol_version: 'mergeos.task-claim.v1', kind: 'task_claim', id: 'tsk_1', task_id: 'tsk_1', status: 'accepted' } },
    { status: 200, body: { protocol_version: 'mergeos.task-claim.v1', kind: 'task_claim', id: 'tsk_2', claim_id: 'prj_1:13', status: 'claimed' } },
    { status: 200, body: { protocol_version: 'mergeos.task-submission.v1', kind: 'task_submission', claim_id: 'prj_1:13', status: 'submitted' } },
    { status: 200, body: { protocol_version: 'mergeos.task-review.v1', kind: 'task_review', claim_id: 'prj_1:13', decision: 'changes_requested', status: 'claimed' } },
    { status: 201, body: { protocol_version: 'mergeos.proposal.v1', kind: 'proposal', proposal: { id: 'ntf_1', status: 'submitted' } } },
    { status: 200, body: { protocol_version: 'mergeos.proposal.v1', kind: 'proposal', proposal: { id: 'ntf_1', status: 'accepted' } } },
    { status: 201, body: { protocol_version: 'mergeos.dispute.v1', kind: 'dispute', notification: { id: 'ntf_1', status: 'dispute:high' } } },
  ]);
  const client = createMergeOSClient({ baseURL: 'http://127.0.0.1:8080', token: 'abc', fetchImpl });

  const payload = { worker_kind: 'human', worker_id: 'github:worker' };
  const task = await client.acceptTask('tsk_1', payload);
  const claimed = await client.claimTask('prj_1:13', payload);
  const reviewPayload = {
    pull_request_url: 'https://github.com/mergeos-bounties/mergeos/pull/13',
    review_notes: 'Acceptance criteria verified.',
  };
  const submitted = await client.submitTask('prj_1:13', reviewPayload);
  const changesPayload = { review_notes: 'Please add browser evidence before release.' };
  const review = await client.requestTaskChanges('prj_1:13', changesPayload);
  const proposalPayload = {
    task_id: 'bounty-prj_1-12',
    cover_letter: 'I can ship this task with tests and review evidence.',
    bid_cents: 12000,
    estimated_hours: 8,
    availability: 'Available this week',
  };
  const proposal = await client.createProposal(proposalPayload);
  const proposalDecision = await client.decideProposal('ntf_1', { decision: 'accepted' });
  const disputePayload = { task_id: 'tsk_1', body: 'Evidence needs maintainer review.' };
  const dispute = await client.createDispute(disputePayload);

  assert.equal(task.protocol_version, 'mergeos.task-claim.v1');
  assert.equal(task.kind, 'task_claim');
  assert.equal(task.status, 'accepted');
  assert.equal(claimed.protocol_version, 'mergeos.task-claim.v1');
  assert.equal(claimed.claim_id, 'prj_1:13');
  assert.equal(claimed.status, 'claimed');
  assert.equal(submitted.protocol_version, 'mergeos.task-submission.v1');
  assert.equal(submitted.status, 'submitted');
  assert.equal(review.protocol_version, 'mergeos.task-review.v1');
  assert.equal(review.decision, 'changes_requested');
  assert.equal(proposal.protocol_version, 'mergeos.proposal.v1');
  assert.equal(proposal.kind, 'proposal');
  assert.equal(proposal.proposal.status, 'submitted');
  assert.equal(proposalDecision.proposal.status, 'accepted');
  assert.equal(dispute.protocol_version, 'mergeos.dispute.v1');
  assert.equal(dispute.kind, 'dispute');
  assert.equal(dispute.notification.status, 'dispute:high');
  assert.equal(fetchImpl.calls[0].url, 'http://127.0.0.1:8080/api/tasks/tsk_1/accept');
  assert.equal(fetchImpl.calls[0].options.method, 'POST');
  assert.equal(fetchImpl.calls[0].options.headers.Authorization, 'Bearer abc');
  assert.equal(fetchImpl.calls[0].options.body, JSON.stringify(payload));
  assert.equal(fetchImpl.calls[1].url, 'http://127.0.0.1:8080/api/tasks/prj_1%3A13/claim');
  assert.equal(fetchImpl.calls[1].options.method, 'POST');
  assert.equal(fetchImpl.calls[1].options.headers.Authorization, 'Bearer abc');
  assert.equal(fetchImpl.calls[1].options.body, JSON.stringify(payload));
  assert.equal(fetchImpl.calls[2].url, 'http://127.0.0.1:8080/api/tasks/prj_1%3A13/submit');
  assert.equal(fetchImpl.calls[2].options.method, 'POST');
  assert.equal(fetchImpl.calls[2].options.headers.Authorization, 'Bearer abc');
  assert.equal(fetchImpl.calls[2].options.body, JSON.stringify(reviewPayload));
  assert.equal(fetchImpl.calls[3].url, 'http://127.0.0.1:8080/api/tasks/prj_1%3A13/request-changes');
  assert.equal(fetchImpl.calls[3].options.method, 'POST');
  assert.equal(fetchImpl.calls[3].options.headers.Authorization, 'Bearer abc');
  assert.equal(fetchImpl.calls[3].options.body, JSON.stringify(changesPayload));
  assert.equal(fetchImpl.calls[4].url, 'http://127.0.0.1:8080/api/proposals');
  assert.equal(fetchImpl.calls[4].options.method, 'POST');
  assert.equal(fetchImpl.calls[4].options.headers.Authorization, 'Bearer abc');
  assert.equal(fetchImpl.calls[4].options.body, JSON.stringify(proposalPayload));
  assert.equal(fetchImpl.calls[5].url, 'http://127.0.0.1:8080/api/proposals/ntf_1/decision');
  assert.equal(fetchImpl.calls[5].options.method, 'POST');
  assert.equal(fetchImpl.calls[5].options.headers.Authorization, 'Bearer abc');
  assert.equal(fetchImpl.calls[5].options.body, JSON.stringify({ decision: 'accepted' }));
  assert.equal(fetchImpl.calls[6].url, 'http://127.0.0.1:8080/api/disputes');
  assert.equal(fetchImpl.calls[6].options.method, 'POST');
  assert.equal(fetchImpl.calls[6].options.headers.Authorization, 'Bearer abc');
  assert.equal(fetchImpl.calls[6].options.body, JSON.stringify(disputePayload));
});

test('creates proposal payloads from public bounty packets', async () => {
  const fetchImpl = fakeFetch([
    { status: 201, body: { protocol_version: 'mergeos.proposal.v1', kind: 'proposal', proposal: { id: 'ntf_packet', status: 'submitted' } } },
  ]);
  const client = createMergeOSClient({ baseURL: 'https://mergeos.shop', token: 'abc', fetchImpl });
  const bounty = {
    id: 'bounty-prj_1-12',
    claim_id: 'claim_12',
    reward_cents: 5000,
    estimated_hours: 4,
    proposal_packet: {
      proposal_endpoint: '/api/proposals',
      payload: {
        task_id: 'claim_12',
        cover_letter: 'I can deliver this bounty with tests.',
        bid_cents: 9000,
        estimated_hours: 6,
        availability: 'Available after customer approval',
      },
    },
  };

  assert.deepEqual(proposalPayloadFromBounty(bounty, { coverLetter: 'Ready this week.' }), {
    task_id: 'claim_12',
    cover_letter: 'Ready this week.',
    bid_cents: 9000,
    estimated_hours: 6,
    availability: 'Available after customer approval',
  });

  const proposal = await client.createProposalFromBounty(bounty, { availability: 'Available Monday' });

  assert.equal(proposal.proposal.status, 'submitted');
  assert.equal(fetchImpl.calls[0].url, 'https://mergeos.shop/api/proposals');
  assert.equal(fetchImpl.calls[0].options.method, 'POST');
  assert.equal(fetchImpl.calls[0].options.headers.Authorization, 'Bearer abc');
  assert.equal(fetchImpl.calls[0].options.body, JSON.stringify({
    task_id: 'claim_12',
    cover_letter: 'I can deliver this bounty with tests.',
    bid_cents: 9000,
    estimated_hours: 6,
    availability: 'Available Monday',
  }));
});

test('supports wallet, payment, and raw upload helper routes', async () => {
  const uploadBody = { marker: 'form-data' };
  const fetchImpl = fakeFetch([
    { status: 201, body: { address: 'mrg_1' } },
    { status: 200, body: { address: 'mrg_1' } },
    { status: 200, body: { linked: true } },
    { status: 201, body: { protocol_version: 'mergeos.wallet-migration.v1', kind: 'wallet_migration' } },
    { status: 201, body: { order_id: 'ord_1', payment_reference: 'ord_1', provider: 'paypal', flow: 'project_funding' } },
    { status: 201, body: { payment_reference: 'pi_1', provider: 'stripe' } },
    { status: 201, body: { id: 'att_1' } },
  ]);
  const client = createMergeOSClient({ token: 'abc', fetchImpl });

  await client.createWallet({ label: 'primary' });
  await client.wallet('mrg_1');
  await client.linkWallet({ address: 'mrg_1' });
  await client.createWalletMigration({ legacy_chain: 'trc20', legacy_address: 'TXYZ987654321987654321987654321999' });
  const paypalOrder = await client.createPayPalOrder({ amount_cents: 120000, description: 'MergeOS PayPal funding', flow: 'project_funding' });
  await client.createCardPaymentIntent({ amount_cents: 120000, description: 'MergeOS card funding' });
  await client.uploadAttachment(uploadBody, { headers: { 'X-Upload': '1' } });

  assert.equal(fetchImpl.calls[0].url, '/api/wallets');
  assert.equal(fetchImpl.calls[1].url, '/api/wallets/mrg_1');
  assert.equal(fetchImpl.calls[2].url, '/api/wallets/link');
  assert.equal(fetchImpl.calls[3].url, '/api/wallets/migrations');
  assert.equal(fetchImpl.calls[3].options.method, 'POST');
  assert.equal(fetchImpl.calls[3].options.headers.Authorization, 'Bearer abc');
  assert.equal(fetchImpl.calls[3].options.body, JSON.stringify({
    legacy_chain: 'trc20',
    legacy_address: 'TXYZ987654321987654321987654321999',
  }));
  assert.equal(fetchImpl.calls[4].url, '/api/payments/paypal/orders');
  assert.equal(fetchImpl.calls[4].options.method, 'POST');
  assert.equal(fetchImpl.calls[4].options.headers.Authorization, 'Bearer abc');
  assert.equal(fetchImpl.calls[4].options.body, JSON.stringify({
    amount_cents: 120000,
    description: 'MergeOS PayPal funding',
    flow: 'project_funding',
  }));
  assert.equal(paypalOrder.payment_reference, 'ord_1');
  assert.equal(fetchImpl.calls[5].url, '/api/payments/card/intents');
  assert.equal(fetchImpl.calls[5].options.method, 'POST');
  assert.equal(fetchImpl.calls[5].options.headers.Authorization, 'Bearer abc');
  assert.equal(fetchImpl.calls[5].options.body, JSON.stringify({ amount_cents: 120000, description: 'MergeOS card funding' }));
  assert.equal(fetchImpl.calls[6].url, '/api/uploads');
  assert.equal(fetchImpl.calls[6].options.body, uploadBody);
  assert.equal(fetchImpl.calls[6].options.headers['Content-Type'], undefined);
  assert.equal(fetchImpl.calls[6].options.headers['X-Upload'], '1');
});

test('supports token workflow helpers for airdrop claims and presale reservations', async () => {
  const wallet = '11111111111111111111111111111111';
  const fetchImpl = fakeFetch([
    {
      status: 201,
      body: {
        protocol_version: 'mergeos.airdrop-claim.v1',
        kind: 'airdrop_claim',
        claim_id: 'airdrop_1',
        ledger_entry: { entry_hash: 'a'.repeat(64) },
      },
    },
    {
      status: 201,
      body: {
        protocol_version: 'mergeos.presale-reservation.v1',
        kind: 'presale_reservation',
        reservation_id: 'presale_1',
        ledger_entry: { entry_hash: 'b'.repeat(64) },
      },
    },
  ]);
  const client = createMergeOSClient({ token: 'abc', fetchImpl });

  assert.equal(normalizeSolanaWalletAddress(` ${wallet} `), wallet);
  assert.equal(isLikelySolanaWallet(wallet), true);
  assert.equal(isLikelySolanaWallet('0xabc'), false);

  const claimPayload = airdropClaimPayload({
    missionID: 'mission_delivery_proof',
    walletAddress: ` ${wallet} `,
    allocationMRG: '350',
    workerID: 'github:builder',
    taskReference: 'prj_public_0001:12',
    proofURL: 'https://github.com/acme/repo/pull/12',
    proofSignals: ['repo_import', 'pull_request'],
    notes: 'Verified task evidence.',
  });
  const reservePayload = presaleReservationPayload({
    walletAddress: wallet,
    reserveMRG: '25000',
    fundingRail: 'usdc',
    fundingReference: 'usdc:tx_123',
    notes: 'Founder tier reservation.',
  });

  const claim = await client.createAirdropClaim(claimPayload);
  const reservation = await client.reservePresale(reservePayload);

  assert.equal(claim.protocol_version, 'mergeos.airdrop-claim.v1');
  assert.equal(reservation.protocol_version, 'mergeos.presale-reservation.v1');
  assert.deepEqual(claimPayload, {
    mission_id: 'mission_delivery_proof',
    wallet_address: wallet,
    allocation_mrg: 350,
    worker_id: 'github:builder',
    task_reference: 'prj_public_0001:12',
    proof_url: 'https://github.com/acme/repo/pull/12',
    proof_signals: ['repo_import', 'pull_request'],
    notes: 'Verified task evidence.',
  });
  assert.deepEqual(reservePayload, {
    tier: 'builder',
    wallet_address: wallet,
    reserve_mrg: 25000,
    funding_rail: 'usdc',
    funding_reference: 'usdc:tx_123',
    notes: 'Founder tier reservation.',
  });
  assert.equal(fetchImpl.calls[0].url, '/api/airdrop/claims');
  assert.equal(fetchImpl.calls[0].options.method, 'POST');
  assert.equal(fetchImpl.calls[0].options.headers.Authorization, 'Bearer abc');
  assert.equal(fetchImpl.calls[0].options.body, JSON.stringify(claimPayload));
  assert.equal(fetchImpl.calls[1].url, '/api/presale/reservations');
  assert.equal(fetchImpl.calls[1].options.method, 'POST');
  assert.equal(fetchImpl.calls[1].options.headers.Authorization, 'Bearer abc');
  assert.equal(fetchImpl.calls[1].options.body, JSON.stringify(reservePayload));
});

test('exposes project estimate protocol route', async () => {
  const fetchImpl = fakeFetch([
    {
      status: 200,
      body: {
        protocol_version: 'mergeos.estimate.v1',
        kind: 'project_estimate',
        suggested_price_cents: 420000,
        suggested_range: { low_cents: 360000, high_cents: 500000 },
        confidence: 'high',
        breakdown: [],
        assumptions: [],
        risks: [],
        editable: true,
      },
    },
  ]);
  const client = new MergeOSClient({ token: 'client-token', fetchImpl });

  const estimate = await client.evaluateProjectPrice({ description: 'Build an AI project workflow.' });

  assert.equal(estimate.protocol_version, 'mergeos.estimate.v1');
  assert.equal(estimate.kind, 'project_estimate');
  assert.equal(fetchImpl.calls[0].url, '/api/projects/evaluate-price');
  assert.equal(fetchImpl.calls[0].options.method, 'POST');
  assert.equal(fetchImpl.calls[0].options.headers.Authorization, 'Bearer client-token');
});

test('exposes project workflow and admin ops routes', async () => {
  const fetchImpl = fakeFetch([
    { status: 200, body: { protocol_version: 'mergeos.escrow.v1', release_status: 'funded' } },
    { status: 200, body: { protocol_version: 'mergeos.payouts.v1', release_status: 'releasing', payouts: [] } },
    { status: 200, body: { protocol_version: 'mergeos.payout-release.v1', kind: 'auto_release', released_count: 1, skipped_count: 0 } },
    { status: 200, body: { project: { project_id: 'prj_1' }, task_graph: { stats: { node_count: 2 } } } },
    { status: 200, body: { protocol_version: 'mergeos.pr-monitor.v1', stats: { pull_request_count: 2 }, tasks: [] } },
    { status: 200, body: { protocol_version: 'mergeos.deployment.v1', status: 'validating' } },
    { status: 200, body: { protocol_version: 'mergeos.ai-workflow.v1', status: 'orchestrating' } },
    { status: 201, body: { protocol_version: 'mergeos.agent-action.v1', kind: 'agent_action', log: { event_name: 'agent_action', action: 'test' } } },
    { status: 200, body: { stats: { node_count: 2 }, nodes: [], edges: [] } },
    { status: 200, body: { protocol_version: 'mergeos.routing.v1', kind: 'project_routing', stats: { ready_count: 1 }, routes: [] } },
    { status: 200, body: { protocol_version: 'mergeos.workflow.v1', progress: 25, current_step: 'contributor_routing', nodes: [], edges: [] } },
    { status: 200, body: { status: 'ready', stats: { scanned_files: 3 }, findings: [] } },
    { status: 200, body: { protocol_version: 'mergeos.scan.v1', findings: [] } },
    {
      status: 200,
      body: {
        protocol_version: 'mergeos.repo-sync.v1',
        added_task_count: 1,
        updated_task_count: 2,
        planning_packet: {
          status: 'ready',
          supervisor_agent_type: 'ceo-agent',
          context_urls: {
            repo_sync: '/api/projects/prj_1/repo-sync',
            routing: '/api/projects/prj_1/routing',
          },
          runbook: [
            { step: 1, action: 'review_generated_tasks', label: 'Review generated tasks', method: 'POST', endpoint: '/api/projects/prj_1/repo-sync' },
          ],
          steps: [
            { id: 'task_generation', title: 'Task generation', status: 'complete', artifact_kind: 'repo_sync', output_endpoint: '/api/projects/prj_1/repo-sync', output_protocol: 'mergeos.repo-sync.v1', output_protocol_url: '/protocol/repo-sync.v1.schema.json' },
            { id: 'contributor_routing', title: 'Contributor routing', status: 'ready', artifact_kind: 'routing_plan', output_endpoint: '/api/projects/prj_1/routing', output_protocol: 'mergeos.routing.v1', output_protocol_url: '/protocol/routing.v1.schema.json' },
          ],
          output_contracts: [
            { action: 'generate_tasks', artifact_kind: 'repo_sync', output_endpoint: '/api/projects/prj_1/repo-sync', output_protocol: 'mergeos.repo-sync.v1', output_protocol_url: '/protocol/repo-sync.v1.schema.json' },
          ],
          summary: {
            issue_count: 1,
            task_count: 1,
            agent_task_count: 1,
            human_task_count: 0,
            hybrid_task_count: 0,
            total_reward_cents: 25000,
            total_estimated_hours: 2.5,
          },
        },
        issue_mappings: [{
          issue_number: 12,
          task_id: 'tsk_12',
          claim_id: 'prj_1:12',
          claim_endpoint: '/api/tasks/prj_1:12/claim',
          task_protocol_url: '/api/public/protocol/tasks?task_id=prj_1:12',
          reward_cents: 25000,
          required_worker_kind: 'agent',
          routing: {
            claim_id: 'prj_1:12',
            protocol_url: '/api/public/protocol/tasks?task_id=prj_1:12',
            recommended_next_action: 'route_to_agent',
            routing_packet: {
              action: 'route_to_agent',
              method: 'POST',
              endpoint: '/api/agent-queue/leases',
              payload: {
                claim_id: 'prj_1:12',
                bounty_id: 'prj_1:12',
                agent_type: 'qa-agent',
                status: 'leased',
              },
              output_contracts: [
                { action: 'lease', output_protocol: 'mergeos.agent-lease.v1' },
              ],
            },
          },
        }],
      },
    },
    { status: 200, body: { stats: { total_count: 1 }, items: [] } },
    { status: 200, body: { stats: { worker_count: 1 }, workers: [] } },
  ]);
  const client = new MergeOSClient({ token: 'admin-token', fetchImpl });

  const escrow = await client.projectEscrow('prj_1');
  const payouts = await client.projectPayouts('prj_1');
  const release = await client.projectAutoRelease('prj_1', {
    task_ids: ['tsk_1'],
    policy: 'mergeos.auto_release.low_risk_pr.v1',
    candidates: [{
      task_id: 'tsk_1',
      worker_kind: 'human',
      worker_id: 'github:builder',
      reward_cents: 5000,
      repository: 'mergeos-bounties/mergeos',
      pull_request_number: 151,
      pull_request_url: 'https://github.com/mergeos-bounties/mergeos/pull/151',
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
  const dashboard = await client.projectDashboard('prj_1');
  const pulls = await client.projectPullRequests('prj_1');
  const deployment = await client.projectDeployment('prj_1');
  const aiWorkflow = await client.projectAIWorkflow('prj_1');
  const agentAction = await client.createProjectAgentAction('prj_1', { action: 'test', agent_type: 'qa-agent' });
  const graph = await client.projectTaskGraph('prj_1');
  const routing = await client.projectRouting('prj_1');
  const workflowProtocol = await client.projectWorkflowProtocol('prj_1');
  const scan = await client.projectRepositoryScan('prj_1');
  const scanProtocol = await client.projectRepositoryScanProtocol('prj_1');
  const sync = await client.syncProjectRepoIssues('prj_1');
  const ops = await client.adminOpsQueue();
  const reputation = await client.adminReputation();

  assert.equal(escrow.protocol_version, 'mergeos.escrow.v1');
  assert.equal(payouts.protocol_version, 'mergeos.payouts.v1');
  assert.equal(release.protocol_version, 'mergeos.payout-release.v1');
  assert.equal(release.released_count, 1);
  assert.equal(dashboard.project.project_id, 'prj_1');
  assert.equal(pulls.protocol_version, 'mergeos.pr-monitor.v1');
  assert.equal(pulls.stats.pull_request_count, 2);
  assert.equal(deployment.protocol_version, 'mergeos.deployment.v1');
  assert.equal(aiWorkflow.protocol_version, 'mergeos.ai-workflow.v1');
  assert.equal(agentAction.protocol_version, 'mergeos.agent-action.v1');
  assert.equal(agentAction.log.action, 'test');
  assert.equal(graph.stats.node_count, 2);
  assert.equal(routing.protocol_version, 'mergeos.routing.v1');
  assert.equal(routing.kind, 'project_routing');
  assert.equal(workflowProtocol.protocol_version, 'mergeos.workflow.v1');
  assert.equal(workflowProtocol.progress, 25);
  assert.equal(workflowProtocol.current_step, 'contributor_routing');
  assert.equal(scan.stats.scanned_files, 3);
  assert.equal(scanProtocol.protocol_version, 'mergeos.scan.v1');
  assert.equal(sync.protocol_version, 'mergeos.repo-sync.v1');
  assert.equal(sync.added_task_count, 1);
  assert.equal(repoPlanningPacket(sync).summary.task_count, 1);
  assert.equal(repoPlanningSteps(sync, 'ready')[0].id, 'contributor_routing');
  assert.equal(repoPlanningOutputContracts(sync, 'mergeos.repo-sync.v1')[0].action, 'generate_tasks');
  assert.equal(sync.issue_mappings[0].claim_id, 'prj_1:12');
  assert.equal(sync.issue_mappings[0].claim_endpoint, '/api/tasks/prj_1:12/claim');
  assert.equal(sync.issue_mappings[0].routing.recommended_next_action, 'route_to_agent');
  assert.equal(sync.issue_mappings[0].routing.routing_packet.endpoint, '/api/agent-queue/leases');
  assert.deepEqual(routingPacketPayload(sync.issue_mappings[0].routing), {
    claim_id: 'prj_1:12',
    bounty_id: 'prj_1:12',
    agent_type: 'qa-agent',
    status: 'leased',
  });
  assert.equal(ops.stats.total_count, 1);
  assert.equal(reputation.stats.worker_count, 1);
  assert.equal(fetchImpl.calls[0].url, '/api/projects/prj_1/escrow');
  assert.equal(fetchImpl.calls[1].url, '/api/projects/prj_1/payouts');
  assert.equal(fetchImpl.calls[2].url, '/api/projects/prj_1/auto-release');
  assert.equal(fetchImpl.calls[2].options.method, 'POST');
  assert.equal(fetchImpl.calls[2].options.headers.Authorization, 'Bearer admin-token');
  assert.equal(fetchImpl.calls[2].options.body, JSON.stringify({
    task_ids: ['tsk_1'],
    policy: 'mergeos.auto_release.low_risk_pr.v1',
    candidates: [{
      task_id: 'tsk_1',
      worker_kind: 'human',
      worker_id: 'github:builder',
      reward_cents: 5000,
      repository: 'mergeos-bounties/mergeos',
      pull_request_number: 151,
      pull_request_url: 'https://github.com/mergeos-bounties/mergeos/pull/151',
      pull_request_title: 'Ship accepted work',
      readiness_status: 'ready',
      can_merge: true,
      risk_level: 'low',
      deployment_status: 'not_required',
      validation_signals: ['evidence: provided', 'star: verified'],
      draft: false,
      can_release: true,
    }],
  }));
  assert.equal(fetchImpl.calls[3].url, '/api/projects/prj_1/dashboard');
  assert.equal(fetchImpl.calls[4].url, '/api/projects/prj_1/pull-requests');
  assert.equal(fetchImpl.calls[5].url, '/api/projects/prj_1/deployment');
  assert.equal(fetchImpl.calls[6].url, '/api/projects/prj_1/ai-workflow');
  assert.equal(fetchImpl.calls[7].url, '/api/projects/prj_1/agent-actions');
  assert.equal(fetchImpl.calls[7].options.method, 'POST');
  assert.equal(fetchImpl.calls[8].url, '/api/projects/prj_1/task-graph');
  assert.equal(fetchImpl.calls[9].url, '/api/projects/prj_1/routing');
  assert.equal(fetchImpl.calls[10].url, '/api/projects/prj_1/protocol/workflow');
  assert.equal(fetchImpl.calls[11].url, '/api/projects/prj_1/repo-scan');
  assert.equal(fetchImpl.calls[12].url, '/api/projects/prj_1/protocol/scan');
  assert.equal(fetchImpl.calls[13].url, '/api/projects/prj_1/repo-sync');
  assert.equal(fetchImpl.calls[13].options.method, 'POST');
  assert.equal(fetchImpl.calls[14].url, '/api/admin/ops-queue');
  assert.equal(fetchImpl.calls[15].url, '/api/admin/reputation');
});

test('builds auto-release payloads from PR monitor task packets', () => {
  const task = {
    task_id: 'tsk_1',
    title: 'Ship accepted work',
    reward_cents: 5000,
    worker_kind: 'agent',
    worker_id: 'github:merge-agent',
    agent_type: 'deployment-agent',
    repository: 'mergeos-bounties/mergeos',
    auto_release_packet: {
      can_auto_release: true,
      policy: 'mergeos.auto_release.low_risk_pr.v1',
      payload: {
        task_ids: ['tsk_1'],
        policy: 'mergeos.auto_release.low_risk_pr.v1',
        candidates: [{
          task_id: 'tsk_1',
          worker_kind: 'agent',
          worker_id: 'github:merge-agent',
          agent_type: 'deployment-agent',
          reward_cents: 5000,
          repository: 'mergeos-bounties/mergeos',
          pull_request_number: 151,
          pull_request_url: 'https://github.com/mergeos-bounties/mergeos/pull/151',
          pull_request_title: 'Ship accepted work',
          readiness_status: 'ready',
          can_merge: true,
          risk_level: 'low',
          deployment_status: 'validated',
          validation_signals: ['evidence: provided', 'deployment: verified'],
          draft: false,
          can_release: true,
        }],
      },
    },
    pull_requests: [{
      number: 151,
      title: 'Ship accepted work',
      html_url: 'https://github.com/mergeos-bounties/mergeos/pull/151',
      draft: false,
      readiness: {
        status: 'ready',
        can_merge: true,
        risk_level: 'low',
        signals: ['evidence: provided', 'deployment: verified'],
      },
    }],
  };

  const fromPacket = autoReleasePayloadFromPRMonitorTask(task);
  assert.deepEqual(fromPacket, task.auto_release_packet.payload);

  const fallback = autoReleasePayloadFromPRMonitorTask({
    ...task,
    auto_release_packet: undefined,
  }, {
    policy: 'mergeos.auto_release.agent_verified_pr.v1',
  });
  assert.deepEqual(fallback.task_ids, ['tsk_1']);
  assert.equal(fallback.policy, 'mergeos.auto_release.agent_verified_pr.v1');
  assert.equal(fallback.candidates[0].task_id, 'tsk_1');
  assert.equal(fallback.candidates[0].worker_kind, 'agent');
  assert.equal(fallback.candidates[0].agent_type, 'deployment-agent');
  assert.equal(fallback.candidates[0].pull_request_number, 151);
  assert.equal(fallback.candidates[0].deployment_status, 'validated');
  assert.deepEqual(fallback.candidates[0].validation_signals, ['evidence: provided', 'deployment: verified']);
  assert.equal(fallback.candidates[0].can_release, false);

  const fetchImpl = fakeFetch([
    {
      status: 200,
      body: {
        protocol_version: 'mergeos.payout-release.v1',
        kind: 'auto_release',
        released_count: 1,
        skipped_count: 0,
        release_proofs: [{
          task_id: 'tsk_1',
          claim_id: 'prj_1:1',
          pull_request_url: 'https://github.com/mergeos-bounties/mergeos/pull/151',
          deployment_status: 'validated',
          ledger_reference: 'task:tsk_1;deployment_validation:validated',
        }],
      },
    },
  ]);
  const client = new MergeOSClient({ token: 'agent-token', fetchImpl });
  return client.projectAutoReleaseFromPRMonitorTask('prj_1', task).then((response) => {
    assert.equal(response.kind, 'auto_release');
    assert.equal(autoReleaseProofsFromResponse(response)[0].deployment_status, 'validated');
    assert.deepEqual(autoReleaseProofsFromResponse({ release_proofs: [null, response.release_proofs[0]] }), [response.release_proofs[0]]);
    assert.deepEqual(autoReleaseProofsFromResponse({}), []);
    assert.equal(fetchImpl.calls[0].url, '/api/projects/prj_1/auto-release');
    assert.equal(fetchImpl.calls[0].options.method, 'POST');
    assert.equal(fetchImpl.calls[0].options.headers.Authorization, 'Bearer agent-token');
    assert.equal(fetchImpl.calls[0].options.body, JSON.stringify(task.auto_release_packet.payload));
  });
});

test('builds and sends review agent payloads from PR monitor review packets', async () => {
  const task = {
    task_id: 'tsk_1',
    review_packet: {
      status: 'blocked',
      review_endpoint: '/api/projects/prj_1/agent-actions',
      pull_request: {
        number: 152,
        url: 'https://github.com/mergeos-bounties/mergeos/pull/152',
        labels: ['evidence: missing'],
      },
      payload: {
        action: 'review',
        claim_id: 'tsk_1',
        bounty_id: 'prj_1:12',
        agent_type: 'review-agent',
        delegated_by: 'ceo-strategy-agent',
        subagent_type: 'review-agent',
        status: 'processed',
        pull_number: 152,
        reference_url: 'https://github.com/mergeos-bounties/mergeos/pull/152',
        labels: ['evidence: missing'],
        context_urls: ['/api/projects/prj_1/pull-requests'],
        evidence: ['evidence: missing'],
        runbook: ['Verify PR links to the funded bounty issue.'],
        checks: [{ name: 'blocker', status: 'blocked', summary: 'workflow file changed' }],
        delegation_chain: ['ceo-strategy-agent', 'review-agent'],
      },
    },
  };
  const payload = agentReviewPayloadFromPRMonitorTask(task);
  assert.equal(payload.action, 'review');
  assert.equal(payload.claim_id, 'tsk_1');
  assert.equal(payload.pull_number, 152);
  assert.equal(payload.reference_url, 'https://github.com/mergeos-bounties/mergeos/pull/152');
  assert.deepEqual(payload.delegation_chain, ['ceo-strategy-agent', 'review-agent']);
  assert.equal(payload.checks[0].status, 'blocked');

  const fetchImpl = fakeFetch([
    { status: 201, body: { protocol_version: 'mergeos.agent-action.v1', kind: 'agent_action', action: 'review', status: 'processed' } },
  ]);
  const client = new MergeOSClient({ token: 'agent-token', fetchImpl });
  const response = await client.createProjectAgentReviewFromPRMonitorTask('prj_1', task);

  assert.equal(response.kind, 'agent_action');
  assert.equal(fetchImpl.calls[0].url, '/api/projects/prj_1/agent-actions');
  assert.equal(fetchImpl.calls[0].options.method, 'POST');
  assert.equal(fetchImpl.calls[0].options.headers.Authorization, 'Bearer agent-token');
  assert.equal(fetchImpl.calls[0].options.body, JSON.stringify(payload));
});

test('funds repository scan suggested tasks and builds agent work packet actions', async () => {
  const workPacket = {
    claim_endpoint: '/api/tasks/prj_1:12/claim',
    action_endpoint: '/api/projects/prj_1/agent-actions',
    submit_endpoint: '/api/tasks/prj_1:12/submit',
    supervisor_agent_type: 'ceo-orchestrator',
    subagent_type: 'security-agent',
    design_review_agent: 'design-review-agent',
    delegation_chain: ['ceo-orchestrator', 'security-agent', 'design-review-agent'],
    context_urls: {
      task_protocol: '/api/public/protocol/tasks?task_id=prj_1:12',
      workflow_protocol: '/api/public/projects/prj_1/workflow',
      repository_scan: '/api/public/projects/prj_1/repo-scan',
    },
    runbook: [
      { step: 1, action: 'fetch_scan', endpoint: '/api/public/projects/prj_1/repo-scan' },
      { step: 2, action: 'claim_task', endpoint: '/api/tasks/prj_1:12/claim' },
    ],
    action_payloads: [{
      action: 'scan',
      label: 'Run repository scan check',
      method: 'POST',
      endpoint: '/api/projects/prj_1/agent-actions',
      body: {
        action: 'scan',
        status: 'queued',
        project_id: 'prj_1',
        claim_id: 'prj_1:12',
        bounty_id: 'prj_1:12',
        agent_type: 'security-agent',
        source_finding_id: 'finding:auth:1',
        signal: 'secret_pattern',
        path: 'backend/internal/core/auth.go',
        context_urls: {
          task_protocol: '/api/public/protocol/tasks?task_id=prj_1:12',
          repository_scan: '/api/public/projects/prj_1/repo-scan',
        },
        evidence_required: ['Attach scan output'],
      },
    }],
    output_contracts: [{
      action: 'scan',
      artifact_kind: 'repository_scan',
      output_endpoint: '/api/projects/prj_1/agent-actions',
      output_protocol: 'mergeos.agent-action.v1',
      output_protocol_url: '/protocol/agent-action.v1.schema.json',
      public_url: '/api/public/projects/prj_1/repo-scan',
    }],
  };
  const fetchImpl = fakeFetch([
    { status: 201, body: { order_id: 'ord_repo_1', payment_reference: 'ord_repo_1', flow: 'repository_task_funding' } },
    {
      status: 201,
      body: {
        protocol_version: 'mergeos.repo-task-funding.v1',
        kind: 'repo_task_funding',
        project_id: 'prj_1',
        suggested_task_id: 'finding:auth:1',
        task_protocol_url: '/api/public/protocol/tasks?task_id=prj_1:12',
        work_packet: workPacket,
      },
    },
  ]);
  const client = new MergeOSClient({ token: 'client-token', fetchImpl });

  const paypalPayload = repositorySuggestedTaskPayPalOrderPayload('finding:auth:1', {
    rewardCents: 25000,
    budgetCents: 30000,
    returnURL: 'https://mergeos.shop/paypal/return',
    cancelUrl: 'https://mergeos.shop/paypal/cancel',
  });
  const fundingPayload = repositorySuggestedTaskFundingPayload('finding:auth:1', {
    rewardCents: 25000,
    budgetCents: 30000,
    paymentMethod: 'card',
    paymentReference: 'LOCAL-PAID',
  });
  const order = await client.createRepositorySuggestedTaskPayPalOrder('prj_1', 'finding:auth:1', paypalPayload);
  const funded = await client.fundRepositorySuggestedTask('prj_1', 'finding:auth:1', fundingPayload);
  const agentPayload = agentActionPayloadFromWorkPacket(funded.work_packet, 'scan', {
    status: 'processed',
    referenceURL: 'https://scan.example/report',
  });

  assert.deepEqual(paypalPayload, {
    suggested_task_id: 'finding:auth:1',
    reward_cents: 25000,
    budget_cents: 30000,
    return_url: 'https://mergeos.shop/paypal/return',
    cancel_url: 'https://mergeos.shop/paypal/cancel',
  });
  assert.deepEqual(fundingPayload, {
    suggested_task_id: 'finding:auth:1',
    reward_cents: 25000,
    budget_cents: 30000,
    payment_method: 'card',
    payment_reference: 'LOCAL-PAID',
  });
  assert.equal(order.flow, 'repository_task_funding');
  assert.equal(funded.protocol_version, 'mergeos.repo-task-funding.v1');
  assert.equal(fetchImpl.calls[0].url, '/api/projects/prj_1/repo-scan/suggested-tasks/finding%3Aauth%3A1/paypal-order');
  assert.equal(fetchImpl.calls[0].options.method, 'POST');
  assert.equal(fetchImpl.calls[0].options.headers.Authorization, 'Bearer client-token');
  assert.equal(fetchImpl.calls[0].options.body, JSON.stringify(paypalPayload));
  assert.equal(fetchImpl.calls[1].url, '/api/projects/prj_1/repo-scan/suggested-tasks/finding%3Aauth%3A1/fund');
  assert.equal(fetchImpl.calls[1].options.method, 'POST');
  assert.equal(fetchImpl.calls[1].options.body, JSON.stringify(fundingPayload));
  assert.deepEqual(agentPayload.context_urls, [
    '/api/public/protocol/tasks?task_id=prj_1:12',
    '/api/public/projects/prj_1/repo-scan',
  ]);
  assert.equal(agentPayload.action, 'scan');
  assert.equal(agentPayload.agent_type, 'security-agent');
  assert.equal(agentPayload.delegated_by, 'ceo-orchestrator');
  assert.equal(agentPayload.design_agent, 'design-review-agent');
  assert.equal(agentPayload.subagent_type, 'security-agent');
  assert.deepEqual(agentPayload.delegation_chain, ['ceo-orchestrator', 'security-agent', 'design-review-agent']);
  assert.equal(agentPayload.source_finding_id, 'finding:auth:1');
  assert.equal(agentPayload.signal, 'secret_pattern');
  assert.equal(agentPayload.path, 'backend/internal/core/auth.go');
  assert.equal(agentPayload.reference_url, 'https://scan.example/report');
  assert.deepEqual(agentPayload.evidence, ['Attach scan output']);
  assert.equal(agentWorkPacketOutputContracts(funded.work_packet, 'scan')[0].public_url, '/api/public/projects/prj_1/repo-scan');
  assert.equal(agentWorkPacketOutputContracts(funded.work_packet).length, 1);
  assert.deepEqual(agentPayload.runbook, [
    '1. fetch_scan (/api/public/projects/prj_1/repo-scan)',
    '2. claim_task (/api/tasks/prj_1:12/claim)',
  ]);
});

test('exposes admin operations, review, settings, and integration routes', async () => {
  const fetchImpl = fakeFetch([
    { status: 200, body: [{ id: 'usr_1' }] },
    { status: 200, body: { id: 'usr_1', role: 'admin' } },
    { status: 200, body: [{ id: 'prj_1' }] },
    { status: 200, body: [{ id: 'tsk_1' }] },
    { status: 200, body: [{ number: 120 }] },
    { status: 200, body: { pull_request: { number: 120 } } },
    { status: 200, body: [{ id: 'note_1' }] },
    { status: 200, body: [{ id: 'att_1' }] },
    { status: 200, body: { llm_provider: 'gemini' } },
    { status: 200, body: { llm_provider: 'openrouter' } },
    { status: 200, body: [] },
    { status: 200, body: { status: 'ok' } },
    { status: 200, body: [] },
    { status: 201, body: { id: 'key_1' } },
    { status: 200, body: { id: 'key_1', status: 'disabled' } },
    { status: 200, body: { ok: true } },
    { status: 200, body: [] },
    { status: 200, body: { test_mode_enabled: true } },
    { status: 200, body: { test_mode_enabled: false } },
    { status: 200, body: [{ id: 'tse_1' }] },
    { status: 201, body: { id: 'tse_2' } },
    { status: 200, body: { id: 'tse_2', status: 'disabled' } },
    { status: 200, body: { ok: true } },
  ]);
  const client = new MergeOSClient({ token: 'admin-token', fetchImpl });

  await client.adminUsers();
  await client.updateAdminUser('usr_1', { name: 'Admin', email: 'admin@example.com' });
  await client.adminProjects();
  await client.adminTasks();
  await client.adminTaskPullRequests('tsk_1');
  await client.mergeAdminTaskPullRequest('tsk_1', 120, { worker_id: 'github:contributor', reward_mrg: 50, bounty_type: 'future-small' });
  await client.adminNotifications();
  await client.adminAttachments();
  await client.adminSettings();
  await client.updateAdminSettings({ llm_provider: 'openrouter' });
  await client.adminSSLReviews();
  await client.reviewAdminSSL({ domain: 'mergeos.shop' });
  await client.adminGeminiKeys();
  await client.addAdminGeminiKey({ key_value: 'secret' });
  await client.updateAdminGeminiKey('key_1', { status: 'disabled' });
  await client.testAdminGeminiKey('key_1');
  await client.adminGeminiWebhooks();
  await client.adminTestSettings();
  await client.updateAdminTestSettings({ test_mode_enabled: false });
  await client.adminTestSettingsEntries();
  await client.addAdminTestSettingsEntry({ integration_type: 'llm', setting_key: 'TASK_KEY', setting_value: 'value' });
  await client.updateAdminTestSettingsEntry('tse_2', { status: 'disabled' });
  await client.deleteAdminTestSettingsEntry('tse_2');

  assert.equal(fetchImpl.calls[0].url, '/api/admin/users');
  assert.equal(fetchImpl.calls[1].url, '/api/admin/users/usr_1');
  assert.equal(fetchImpl.calls[4].url, '/api/admin/tasks/tsk_1/pulls');
  assert.equal(fetchImpl.calls[5].url, '/api/admin/tasks/tsk_1/pulls/120/merge');
  assert.equal(fetchImpl.calls[5].options.method, 'POST');
  assert.equal(fetchImpl.calls[10].url, '/api/admin/ssl');
  assert.equal(fetchImpl.calls[12].url, '/api/admin/gemini/keys');
  assert.equal(fetchImpl.calls[15].url, '/api/admin/gemini/keys/key_1/test');
  assert.equal(fetchImpl.calls[17].url, '/api/admin/test-settings');
  assert.equal(fetchImpl.calls[18].options.method, 'PATCH');
  assert.equal(fetchImpl.calls[22].url, '/api/admin/test-settings/entries/tse_2');
  assert.equal(fetchImpl.calls[22].options.method, 'DELETE');
});

test('throws response errors with status and payload', async () => {
  const fetchImpl = fakeFetch([{ status: 403, body: { error: 'admin access is required' } }]);
  const client = new MergeOSClient({ fetchImpl });

  await assert.rejects(
    () => client.adminSummary(),
    (error) => {
      assert.equal(error.status, 403);
      assert.deepEqual(error.payload, { error: 'admin access is required' });
      return true;
    },
  );
});

test('builds websocket URLs for event streams', () => {
  const sockets = [];
  class FakeWebSocket {
    constructor(url, protocols) {
      this.url = url;
      this.protocols = protocols;
      sockets.push(this);
    }
  }
  const client = new MergeOSClient({ baseURL: 'https://mergeos.shop' });
  const realtimeClient = new MergeOSClient({ baseURL: 'https://mergeos.shop', WebSocketImpl: FakeWebSocket });

  assert.equal(client.webSocketURL('/api/ws'), 'wss://mergeos.shop/api/ws');
  assert.equal(client.webSocketURL('ws://localhost:8080/api/ws'), 'ws://localhost:8080/api/ws');
  realtimeClient.connectEvents({ limit: 12, afterID: 'event:latest', since: new Date('2026-06-06T00:00:00Z'), protocols: ['mergeos.event.v1'] });
  assert.equal(sockets[0].url, 'wss://mergeos.shop/api/ws?limit=12&after_id=event%3Alatest&since=2026-06-06T00%3A00%3A00.000Z');
  assert.deepEqual(sockets[0].protocols, ['mergeos.event.v1']);
});
