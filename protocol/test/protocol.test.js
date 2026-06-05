import assert from 'node:assert/strict';
import test from 'node:test';
import {
  assertProtocolDocument,
  contractReferenceBytes,
  contractReferenceFromLedger,
  legacyWalletAddressHash,
  normalizeLegacyChain,
  normalizeLegacyWalletAddress,
  protocolSchemas,
  schemaForProtocol,
  validateProtocolDocument,
  walletMigrationPDASeedMetadata,
} from '../src/index.js';

test('loads stable task, workflow, ledger, and event schemas', () => {
  assert.deepEqual(Object.keys(protocolSchemas).sort(), [
    'mergeos.admin-ops.v1',
    'mergeos.agent-action.v1',
    'mergeos.agent.v1',
    'mergeos.ai-workflow.v1',
    'mergeos.contributor.v1',
    'mergeos.customer-dashboard.v1',
    'mergeos.deployment.v1',
    'mergeos.dispute.v1',
    'mergeos.escrow.v1',
    'mergeos.estimate.v1',
    'mergeos.event.v1',
    'mergeos.ledger.v1',
    'mergeos.live-feed.v1',
    'mergeos.marketplace.v1',
    'mergeos.payout-release.v1',
    'mergeos.payouts.v1',
    'mergeos.pr-monitor.v1',
    'mergeos.proposal.v1',
    'mergeos.repo-import.v1',
    'mergeos.repo-sync.v1',
    'mergeos.scan.v1',
    'mergeos.task-claim.v1',
    'mergeos.task.v1',
    'mergeos.wallet-migration.v1',
    'mergeos.worker-dashboard.v1',
    'mergeos.workflow.v1',
  ]);
  assert.equal(schemaForProtocol('mergeos.task.v1').title, 'MergeOS Task v1');
});

test('validates marketplace protocol documents', () => {
  const now = '2026-06-05T00:00:00.000Z';
  const marketplace = {
    protocol_version: 'mergeos.marketplace.v1',
    kind: 'marketplace',
    stats: {
      project_count: 1,
      open_task_count: 2,
      accepted_task_count: 1,
      ledger_entry_count: 3,
      total_budget_cents: 250000,
      work_pool_cents: 225000,
      token_symbol: 'MRG',
      updated_at: now,
    },
    projects: [
      {
        id: 'prj_0001',
        title: 'Customer portal rebuild',
        brief: 'Rebuild the customer portal with a responsive interface and proof ledger.',
        status: 'funded',
        client_display_name: 'Marketplace Co',
        bounty_repo_name: 'mergeos-bounties/mergeos',
        repo_provider: 'github',
        repo_url: 'https://github.com/mergeos-bounties/mergeos',
        budget_cents: 250000,
        work_pool_cents: 225000,
        task_count: 3,
        open_task_count: 2,
        accepted_task_count: 1,
        tags: ['github', 'marketplace'],
        created_at: now,
      },
    ],
    bounties: [
      {
        id: 'prj_0001:issue:12',
        claim_id: 'claim_12',
        project_id: 'prj_0001',
        project_title: 'Customer portal rebuild',
        issue_number: 12,
        title: 'Fix checkout UI',
        acceptance: 'Tests pass and deployment preview is linked.',
        reward_cents: 5000,
        estimated_hours: 4,
        required_worker_kind: 'human',
        suggested_agent_type: 'qa-agent',
        bounty_type: 'future-small',
        evidence_required: ['tests', 'deploy_preview'],
        source_repository: 'https://github.com/mergeos-bounties/mergeos',
        issue_url: 'https://github.com/mergeos-bounties/mergeos/issues/12',
        created_at: now,
      },
    ],
    contributors: [
      {
        worker_id: 'github:maya-dev',
        name: 'maya-dev',
        kind: 'human',
        task_count: 1,
        earned_cents: 5000,
        last_paid_at: now,
        reputation_score: 84,
        reputation_level: 'Trusted',
        risk_level: 'low',
      },
    ],
    agents: [
      {
        type: 'qa-agent',
        title: 'QA Agent',
        worker_kind: 'agent',
        task_count: 2,
        open_task_count: 2,
        budget_cents: 10000,
      },
    ],
  };

  assert.equal(validateProtocolDocument(marketplace).valid, true);

  const invalid = validateProtocolDocument({
    ...marketplace,
    kind: 'market',
    stats: { ...marketplace.stats, open_task_count: -1 },
    agents: [{ ...marketplace.agents[0], worker_kind: 'bot' }],
  });
  assert.equal(invalid.valid, false);
  assert(invalid.errors.some((error) => error.path === 'kind'));
  assert(invalid.errors.some((error) => error.path === 'stats.open_task_count'));
  assert(invalid.errors.some((error) => error.path === 'agents[0].worker_kind'));
});

test('validates live feed protocol documents', () => {
  const now = '2026-06-05T00:00:00.000Z';
  const feed = {
    protocol_version: 'mergeos.live-feed.v1',
    kind: 'live_feed',
    stats: {
      project_count: 1,
      open_task_count: 2,
      accepted_task_count: 1,
      active_contributor_count: 3,
      active_agent_count: 2,
      ledger_entry_count: 6,
      ai_action_count: 4,
      total_budget_cents: 250000,
      token_symbol: 'MRG',
      updated_at: now,
    },
    items: [
      {
        id: 'project:prj_0001',
        type: 'project_funded',
        title: 'Project funded',
        body: 'Customer portal opened with escrow-backed delivery.',
        project_id: 'prj_0001',
        project_title: 'Customer portal rebuild',
        actor: 'Marketplace Co',
        amount_cents: 250000,
        reference: 'project:prj_0001',
        url: 'https://github.com/mergeos-bounties/mergeos',
        status: 'funded',
        created_at: now,
      },
      {
        id: 'task-accepted:prj_0001:12',
        type: 'task_accepted',
        title: 'Task #12 accepted',
        body: 'Fix checkout UI - Tests pass and deployment preview is linked.',
        project_id: 'prj_0001',
        project_title: 'Customer portal rebuild',
        actor: 'maya-dev',
        amount_cents: 5000,
        reference: 'https://github.com/mergeos-bounties/mergeos/issues/12',
        evidence_required: ['tests', 'deploy_preview'],
        status: 'accepted',
        created_at: now,
      },
      {
        id: 'ledger:6',
        type: 'ledger_task_payment',
        title: 'Task payout released',
        body: 'Customer portal rebuild recorded Task payout released.',
        project_id: 'prj_0001',
        project_title: 'Customer portal rebuild',
        actor: 'github:maya-dev',
        amount_cents: 5000,
        ledger_sequence: 6,
        entry_hash: 'a'.repeat(64),
        reference: 'pr:https://github.com/mergeos-bounties/mergeos/pull/151;title:Live feed proof',
        url: 'https://github.com/mergeos-bounties/mergeos/pull/151',
        status: 'verified',
        created_at: now,
      },
      {
        id: 'ai:log_0001',
        type: 'agent_action',
        title: 'AI agent tested PR #151',
        body: 'QA Agent ran test for mergeos-bounties/mergeos PR #151.',
        actor: 'QA Agent',
        action: 'test',
        reference: 'mergeos-bounties/mergeos#151',
        url: 'https://github.com/mergeos-bounties/mergeos/pull/151#issuecomment-1',
        status: 'processed',
        created_at: now,
      },
    ],
  };

  assert.equal(validateProtocolDocument(feed).valid, true);

  const invalid = validateProtocolDocument({
    ...feed,
    kind: 'feed',
    stats: { ...feed.stats, active_agent_count: -1 },
    items: [{ ...feed.items[0], type: 'unknown_feed_type', created_at: 'not-a-date' }],
  });
  assert.equal(invalid.valid, false);
  assert(invalid.errors.some((error) => error.path === 'kind'));
  assert(invalid.errors.some((error) => error.path === 'stats.active_agent_count'));
  assert(invalid.errors.some((error) => error.path === 'items[0].type'));
  assert(invalid.errors.some((error) => error.path === 'items[0].created_at'));
});

test('validates repository import protocol documents', () => {
  const now = '2026-06-05T00:00:00.000Z';
  const report = {
    protocol_version: 'mergeos.repo-import.v1',
    kind: 'repo_import',
    owner: 'mergeos-bounties',
    name: 'mergeos',
    repo_url: 'https://github.com/mergeos-bounties/mergeos',
    issue_count: 2,
    total_estimated_cents: 21000,
    total_estimated_hours: 10.5,
    issues: [
      {
        number: 42,
        title: 'Payment checkout crashes after auth token refresh',
        state: 'open',
        url: 'https://github.com/mergeos-bounties/mergeos/issues/42',
        labels: ['bug', 'checkout'],
        comments: 4,
        score: 91,
        complexity: 'high',
        estimated_cents: 15000,
        estimated_hours: 7.5,
        required_worker_kind: 'hybrid',
        suggested_agent_type: 'security-review-agent',
        reasons: ['GitHub issue', 'production risk'],
        created_at: now,
        updated_at: now,
      },
      {
        number: 43,
        title: 'Responsive footer polish',
        state: 'open',
        url: 'https://github.com/mergeos-bounties/mergeos/issues/43',
        labels: ['frontend'],
        comments: 1,
        score: 45,
        complexity: 'medium',
        estimated_cents: 6000,
        estimated_hours: 3,
        required_worker_kind: 'agent',
        suggested_agent_type: 'frontend-agent',
        reasons: ['frontend surface'],
        created_at: now,
        updated_at: now,
      },
    ],
  };

  assert.equal(validateProtocolDocument(report).valid, true);

  const invalid = validateProtocolDocument({
    ...report,
    kind: 'repo_scan',
    issue_count: -1,
    issues: [{ ...report.issues[0], score: 101, required_worker_kind: 'bot', updated_at: 'not-a-date' }],
  });
  assert.equal(invalid.valid, false);
  assert(invalid.errors.some((error) => error.path === 'kind'));
  assert(invalid.errors.some((error) => error.path === 'issue_count'));
  assert(invalid.errors.some((error) => error.path === 'issues[0].score'));
  assert(invalid.errors.some((error) => error.path === 'issues[0].required_worker_kind'));
  assert(invalid.errors.some((error) => error.path === 'issues[0].updated_at'));
});

test('validates repository sync protocol documents', () => {
  const sync = {
    protocol_version: 'mergeos.repo-sync.v1',
    kind: 'repo_sync',
    project_id: 'prj_0001',
    project_title: 'Customer portal rebuild',
    source_repo_url: 'https://github.com/mergeos-bounties/mergeos',
    imported_issue_count: 8,
    added_task_count: 3,
    updated_task_count: 2,
    open_issue_count: 6,
    closed_issue_count: 2,
    synced_at: '2026-06-05T00:00:00.000Z',
  };

  assert.equal(validateProtocolDocument(sync).valid, true);

  const invalid = validateProtocolDocument({
    ...sync,
    kind: 'sync',
    added_task_count: -1,
    synced_at: 'not-a-date',
  });
  assert.equal(invalid.valid, false);
  assert(invalid.errors.some((error) => error.path === 'kind'));
  assert(invalid.errors.some((error) => error.path === 'added_task_count'));
  assert(invalid.errors.some((error) => error.path === 'synced_at'));
});

test('validates dispute protocol documents', () => {
  const now = '2026-06-05T00:00:00.000Z';
  const dispute = {
    protocol_version: 'mergeos.dispute.v1',
    kind: 'dispute',
    dispute_id: 'ntf_0001',
    project_id: 'prj_0001',
    task_id: 'tsk_0001',
    user_id: 'usr_0001',
    severity: 'critical',
    status: 'dispute:critical',
    subject: 'Milestone evidence mismatch',
    body: 'Task #12: The submitted evidence does not match the deployed result.',
    notification: {
      id: 'ntf_0001',
      user_id: 'usr_0001',
      project_id: 'prj_0001',
      channel: 'dispute',
      subject: 'Milestone evidence mismatch',
      body: 'Task #12: The submitted evidence does not match the deployed result.',
      status: 'dispute:critical',
      created_at: now,
    },
    created_at: now,
  };

  assert.equal(validateProtocolDocument(dispute).valid, true);

  const invalid = validateProtocolDocument({
    ...dispute,
    kind: 'moderation',
    severity: 'urgent',
    created_at: 'not-a-date',
    notification: { ...dispute.notification, channel: 'email' },
  });
  assert.equal(invalid.valid, false);
  assert(invalid.errors.some((error) => error.path === 'kind'));
  assert(invalid.errors.some((error) => error.path === 'severity'));
  assert(invalid.errors.some((error) => error.path === 'notification.channel'));
  assert(invalid.errors.some((error) => error.path === 'created_at'));
});

test('validates agent action protocol documents', () => {
  const now = '2026-06-05T00:00:00.000Z';
  const action = {
    protocol_version: 'mergeos.agent-action.v1',
    kind: 'agent_action',
    action_id: 'gwh_0001',
    project_id: 'prj_0001',
    action: 'test',
    agent_type: 'qa-agent',
    status: 'processed',
    repository: 'mergeos-bounties/mergeos',
    pull_number: 777,
    reference_url: 'https://github.com/mergeos-bounties/mergeos/pull/777',
    labels: ['smoke', 'release-gate'],
    duration_millis: 1234,
    received_at: now,
    completed_at: now,
    log: {
      id: 'gwh_0001',
      event_name: 'agent_action',
      action: 'test',
      repository: 'mergeos-bounties/mergeos',
      pull_number: 777,
      sender: 'agent:qa-agent',
      status: 'processed',
      status_code: 200,
      comment_url: 'https://github.com/mergeos-bounties/mergeos/pull/777',
      labels: ['smoke', 'release-gate'],
      duration_millis: 1234,
      received_at: now,
      completed_at: now,
    },
  };

  assert.equal(validateProtocolDocument(action).valid, true);

  const invalid = validateProtocolDocument({
    ...action,
    kind: 'agent',
    action: 'lint',
    status: 'done',
    log: { ...action.log, event_name: 'webhook', status_code: 99 },
  });
  assert.equal(invalid.valid, false);
  assert(invalid.errors.some((error) => error.path === 'kind'));
  assert(invalid.errors.some((error) => error.path === 'action'));
  assert(invalid.errors.some((error) => error.path === 'status'));
  assert(invalid.errors.some((error) => error.path === 'log.event_name'));
  assert(invalid.errors.some((error) => error.path === 'log.status_code'));
});

test('validates escrow protocol documents', () => {
  const now = '2026-06-05T00:00:00.000Z';
  const escrow = {
    protocol_version: 'mergeos.escrow.v1',
    kind: 'escrow',
    project_id: 'prj_0001',
    project_title: 'Customer portal rebuild',
    token_symbol: 'MRG',
    release_status: 'releasing',
    budget_cents: 250000,
    fee_cents: 25000,
    work_pool_cents: 225000,
    project_reserve_cents: 225000,
    task_reserve_cents: 225000,
    task_payment_cents: 5000,
    manual_credit_cents: 0,
    released_cents: 5000,
    remaining_cents: 220000,
    overdrawn_cents: 0,
    unallocated_cents: 0,
    paid_task_count: 1,
    open_task_count: 2,
    updated_at: now,
    tasks: [
      {
        task_id: 'tsk_0001',
        issue_number: 12,
        title: 'Fix checkout UI',
        status: 'accepted',
        release_status: 'released',
        reward_cents: 5000,
        paid_cents: 5000,
        remaining_cents: 0,
        overpaid_cents: 0,
        worker_id: 'github:maya-dev',
        proof_hash: 'proof_123',
        issue_url: 'https://github.com/mergeos-bounties/mergeos/issues/12',
        updated_at: now,
      },
      {
        task_id: 'tsk_0002',
        issue_number: 13,
        title: 'Validate deployment preview',
        status: 'open',
        release_status: 'reserved',
        reward_cents: 10000,
        paid_cents: 0,
        remaining_cents: 10000,
        overpaid_cents: 0,
        updated_at: now,
      },
    ],
  };

  assert.equal(validateProtocolDocument(escrow).valid, true);

  const invalid = validateProtocolDocument({
    ...escrow,
    kind: 'escrow_state',
    release_status: 'waiting',
    tasks: [{ ...escrow.tasks[0], release_status: 'unknown', paid_cents: -1 }],
  });
  assert.equal(invalid.valid, false);
  assert(invalid.errors.some((error) => error.path === 'kind'));
  assert(invalid.errors.some((error) => error.path === 'release_status'));
  assert(invalid.errors.some((error) => error.path === 'tasks[0].release_status'));
  assert(invalid.errors.some((error) => error.path === 'tasks[0].paid_cents'));
});

test('validates payout settlement protocol documents', () => {
  const now = '2026-06-05T00:00:00.000Z';
  const payouts = {
    protocol_version: 'mergeos.payouts.v1',
    kind: 'payouts',
    project_id: 'prj_0001',
    project_title: 'Customer portal rebuild',
    token_symbol: 'MRG',
    release_status: 'releasing',
    work_pool_cents: 225000,
    released_cents: 5000,
    remaining_cents: 220000,
    overdrawn_cents: 0,
    task_count: 2,
    paid_task_count: 1,
    open_task_count: 1,
    release_count: 1,
    updated_at: now,
    payouts: [
      {
        task_id: 'tsk_0001',
        issue_number: 12,
        title: 'Fix checkout UI',
        type: 'task_payment',
        status: 'accepted',
        release_status: 'released',
        worker_id: 'github:maya-dev',
        payout_account: 'So1anaWorkerWallet1111111111111111111111111',
        reward_cents: 5000,
        paid_cents: 5000,
        remaining_cents: 0,
        overpaid_cents: 0,
        ledger_sequence: 7,
        ledger_entry_count: 1,
        entry_hash: 'c'.repeat(64),
        proof_hash: 'c'.repeat(64),
        reference: 'pr:https://github.com/mergeos-bounties/mergeos/pull/190;title:Payout proof',
        url: 'https://github.com/mergeos-bounties/mergeos/pull/190',
        released_at: now,
        updated_at: now,
      },
      {
        task_id: 'tsk_0002',
        issue_number: 13,
        title: 'Validate deployment preview',
        type: 'reserved',
        status: 'open',
        release_status: 'reserved',
        reward_cents: 10000,
        paid_cents: 0,
        remaining_cents: 10000,
        overpaid_cents: 0,
        ledger_entry_count: 0,
        reference: 'https://github.com/mergeos-bounties/mergeos/issues/13',
        updated_at: now,
      },
    ],
  };

  assert.equal(validateProtocolDocument(payouts).valid, true);

  const invalid = validateProtocolDocument({
    ...payouts,
    kind: 'payments',
    release_status: 'waiting',
    payouts: [{ ...payouts.payouts[0], type: 'wire', paid_cents: -1, released_at: 'not-a-date' }],
  });
  assert.equal(invalid.valid, false);
  assert(invalid.errors.some((error) => error.path === 'kind'));
  assert(invalid.errors.some((error) => error.path === 'release_status'));
  assert(invalid.errors.some((error) => error.path === 'payouts[0].type'));
  assert(invalid.errors.some((error) => error.path === 'payouts[0].paid_cents'));
  assert(invalid.errors.some((error) => error.path === 'payouts[0].released_at'));

  const release = {
    protocol_version: 'mergeos.payout-release.v1',
    kind: 'auto_release',
    project_id: 'prj_0001',
    policy: 'mergeos.auto_release.low_risk_pr.v1',
    released_count: 1,
    skipped_count: 0,
    released: [{ protocol_version: 'mergeos.task-claim.v1', kind: 'task_claim', task_id: 'tsk_0001' }],
    skipped: [],
    payouts,
  };
  assert.equal(validateProtocolDocument(release).valid, true);

  const unsupportedRelease = validateProtocolDocument({
    ...release,
    protocol_version: 'mergeos.payout-release.v2',
  });
  assert.equal(unsupportedRelease.valid, false);
  assert(unsupportedRelease.errors.some((error) => error.path === 'protocol_version'));

  const invalidRelease = validateProtocolDocument({
    ...release,
    skipped: [{ task_id: 'tsk_0002', reason: '' }],
  });
  assert.equal(invalidRelease.valid, false);
  assert(invalidRelease.errors.some((error) => error.path === 'skipped[0].reason'));
});

test('validates deployment protocol documents', () => {
  const now = '2026-06-05T00:00:00.000Z';
  const deployment = {
    protocol_version: 'mergeos.deployment.v1',
    kind: 'deployment',
    project_id: 'prj_0001',
    project_title: 'Customer portal rebuild',
    status: 'validating',
    progress: 60,
    updated_at: now,
    stages: [
      {
        id: 'deployment_handoff',
        title: 'Deployment handoff',
        body: 'Deployment pipeline and handoff notes have been accepted.',
        status: 'complete',
        tone: 'green',
        source_task_issue_number: 13,
        reference: 'https://github.com/mergeos-bounties/mergeos/issues/13',
        url: 'https://github.com/mergeos-bounties/mergeos/issues/13',
        updated_at: now,
      },
      {
        id: 'release_gate',
        title: 'Release gate',
        body: '1 of 3 delivery tasks are accepted and paid.',
        status: 'in_progress',
        tone: 'blue',
        reference: 'project:prj_0001',
        updated_at: now,
      },
    ],
    signals: [
      {
        id: 'ledger:7',
        type: 'ledger_task_payment',
        title: 'Task payout released',
        body: 'Customer portal rebuild recorded Task payout released.',
        status: 'verified',
        reference: 'project:prj_0001;issue:13',
        url: 'https://github.com/mergeos-bounties/mergeos/issues/13',
        created_at: now,
      },
      {
        id: 'ai:log_0001',
        type: 'agent_action',
        title: 'AI agent deployed PR #151',
        body: 'Deploy Agent ran deployment handoff for mergeos-bounties/mergeos PR #151.',
        status: 'processed',
        reference: 'mergeos-bounties/mergeos#151',
        created_at: now,
      },
    ],
  };

  assert.equal(validateProtocolDocument(deployment).valid, true);

  const invalid = validateProtocolDocument({
    ...deployment,
    kind: 'deployments',
    progress: 101,
    stages: [{ ...deployment.stages[0], tone: 'success' }],
    signals: [{ ...deployment.signals[0], created_at: 'not-a-date' }],
  });
  assert.equal(invalid.valid, false);
  assert(invalid.errors.some((error) => error.path === 'kind'));
  assert(invalid.errors.some((error) => error.path === 'progress'));
  assert(invalid.errors.some((error) => error.path === 'stages[0].tone'));
  assert(invalid.errors.some((error) => error.path === 'signals[0].created_at'));
});

test('validates AI workflow protocol documents', () => {
  const now = '2026-06-05T00:00:00.000Z';
  const workflow = {
    protocol_version: 'mergeos.ai-workflow.v1',
    kind: 'ai_workflow',
    project_id: 'prj_0001',
    project_title: 'Customer portal rebuild',
    status: 'orchestrating',
    progress: 71,
    current_step: 'pr_review',
    task_count: 3,
    agent_task_count: 1,
    human_task_count: 1,
    hybrid_task_count: 1,
    ai_action_count: 2,
    updated_at: now,
    stages: [
      {
        id: 'repo_import',
        title: 'Repository context',
        body: 'Repository context is attached to the delivery workflow.',
        status: 'complete',
        tone: 'green',
        reference: 'https://github.com/mergeos-bounties/mergeos',
        url: 'https://github.com/mergeos-bounties/mergeos',
        updated_at: now,
      },
      {
        id: 'pr_review',
        title: 'AI review and agent actions',
        body: '1 opened PRs are waiting for AI review or agent execution.',
        status: 'in_progress',
        tone: 'blue',
        reference: 'mergeos-bounties/mergeos',
        updated_at: now,
      },
      {
        id: 'deployment_validation',
        title: 'Deployment validation',
        body: 'Deployment validation is 60% complete.',
        status: 'in_progress',
        tone: 'blue',
        reference: 'project:prj_0001',
        updated_at: now,
      },
    ],
    signals: [
      {
        id: 'ai:log_0001',
        type: 'agent_action',
        title: 'AI agent reviewed PR #151',
        body: 'Review Agent ran review for mergeos-bounties/mergeos PR #151.',
        status: 'processed',
        reference: 'mergeos-bounties/mergeos#151',
        url: 'https://github.com/mergeos-bounties/mergeos/pull/151#issuecomment-1',
        created_at: now,
      },
      {
        id: 'deployment:prj_0001',
        type: 'deployment_validation',
        title: 'Deployment validation',
        body: 'Deployment validation is 60% complete.',
        status: 'validating',
        reference: 'project:prj_0001',
        created_at: now,
      },
    ],
  };

  assert.equal(validateProtocolDocument(workflow).valid, true);

  const invalid = validateProtocolDocument({
    ...workflow,
    kind: 'workflow',
    progress: 101,
    current_step: 'manual_review',
    stages: [{ ...workflow.stages[0], id: 'unknown_stage', status: 'running' }],
    signals: [{ ...workflow.signals[0], created_at: 'not-a-date' }],
  });
  assert.equal(invalid.valid, false);
  assert(invalid.errors.some((error) => error.path === 'kind'));
  assert(invalid.errors.some((error) => error.path === 'progress'));
  assert(invalid.errors.some((error) => error.path === 'current_step'));
  assert(invalid.errors.some((error) => error.path === 'stages[0].id'));
  assert(invalid.errors.some((error) => error.path === 'stages[0].status'));
  assert(invalid.errors.some((error) => error.path === 'signals[0].created_at'));
});

test('validates PR monitor protocol documents', () => {
  const now = '2026-06-05T00:00:00.000Z';
  const monitor = {
    protocol_version: 'mergeos.pr-monitor.v1',
    kind: 'pr_monitor',
    project_id: 'prj_0001',
    project_title: 'Customer portal rebuild',
    updated_at: now,
    stats: {
      task_count: 2,
      linked_task_count: 1,
      pull_request_count: 2,
      open_pull_request_count: 1,
      merged_pull_request_count: 1,
      ready_count: 1,
      needs_review_count: 0,
      blocked_count: 1,
      error_count: 0,
      auto_release_ready_count: 1,
    },
    tasks: [
      {
        task_id: 'tsk_0001',
        issue_number: 12,
        title: 'Fix checkout UI',
        status: 'open',
        reward_cents: 12500,
        worker_kind: 'human',
        worker_id: 'github:maya-dev',
        issue_url: 'https://github.com/mergeos-bounties/mergeos/issues/12',
        repository: 'mergeos-bounties/mergeos',
        monitor_status: 'synced',
        auto_release_packet: {
          status: 'ready',
          can_auto_release: true,
          release_endpoint: '/api/projects/prj_0001/auto-release',
          method: 'POST',
          payload: {
            task_ids: ['tsk_0001'],
            policy: 'mergeos.auto_release.low_risk_pr.v1',
            candidates: [{
              task_id: 'tsk_0001',
              worker_kind: 'human',
              worker_id: 'github:maya-dev',
              reward_cents: 12500,
              repository: 'mergeos-bounties/mergeos',
              pull_request_number: 151,
              pull_request_url: 'https://github.com/mergeos-bounties/mergeos/pull/151',
              pull_request_title: 'Fix checkout UI',
              readiness_status: 'ready',
              can_merge: true,
              risk_level: 'low',
              draft: false,
              can_release: true,
            }],
          },
        },
        updated_at: now,
        pull_requests: [
          {
            number: 151,
            title: 'Fix checkout UI',
            state: 'open',
            html_url: 'https://github.com/mergeos-bounties/mergeos/pull/151',
            merge_url: 'https://api.github.com/repos/mergeos-bounties/mergeos/pulls/151/merge',
            author: 'maya-dev',
            draft: false,
            merged: false,
            mergeable_state: 'clean',
            base_ref: 'master',
            head_ref: 'fix-checkout',
            labels: ['evidence: provided', 'star: verified'],
            readiness: {
              status: 'ready',
              can_merge: true,
              risk_level: 'low',
              signals: ['evidence_ready'],
            },
            created_at: now,
            updated_at: now,
          },
          {
            number: 152,
            title: 'Delete deploy workflow',
            state: 'open',
            html_url: 'https://github.com/mergeos-bounties/mergeos/pull/152',
            author: 'risky-builder',
            draft: false,
            merged: false,
            mergeable_state: 'dirty',
            labels: ['evidence: missing'],
            readiness: {
              status: 'blocked',
              can_merge: false,
              risk_level: 'high',
              blockers: ['workflow file changed'],
              warnings: ['missing evidence'],
            },
            created_at: now,
            updated_at: now,
          },
        ],
      },
      {
        task_id: 'tsk_0002',
        issue_number: 13,
        title: 'Validate deployment preview',
        status: 'open',
        monitor_status: 'unlinked',
        updated_at: now,
        pull_requests: [],
      },
    ],
  };

  assert.equal(validateProtocolDocument(monitor).valid, true);

  const invalid = validateProtocolDocument({
    ...monitor,
    kind: 'pull_requests',
    stats: { ...monitor.stats, blocked_count: -1 },
    tasks: [{ ...monitor.tasks[0], monitor_status: 'waiting', pull_requests: [{ ...monitor.tasks[0].pull_requests[0], readiness: { status: 'unknown', can_merge: 'yes', risk_level: 'critical' } }] }],
  });
  assert.equal(invalid.valid, false);
  assert(invalid.errors.some((error) => error.path === 'kind'));
  assert(invalid.errors.some((error) => error.path === 'stats.blocked_count'));
  assert(invalid.errors.some((error) => error.path === 'tasks[0].monitor_status'));
  assert(invalid.errors.some((error) => error.path === 'tasks[0].pull_requests[0].readiness.status'));
  assert(invalid.errors.some((error) => error.path === 'tasks[0].pull_requests[0].readiness.can_merge'));
  assert(invalid.errors.some((error) => error.path === 'tasks[0].pull_requests[0].readiness.risk_level'));
});

test('validates admin operations protocol documents', () => {
  const now = '2026-06-05T00:00:00.000Z';
  const queue = {
    protocol_version: 'mergeos.admin-ops.v1',
    kind: 'admin_ops',
    stats: {
      total_count: 7,
      dispute_count: 1,
      moderation_count: 2,
      payout_review_count: 2,
      proposal_count: 1,
      fraud_count: 1,
      security_count: 1,
      critical_count: 1,
      updated_at: now,
    },
    items: [
      {
        id: 'dispute:ntf_1',
        type: 'dispute',
        severity: 'critical',
        title: 'Delivery notification needs review',
        body: 'Milestone evidence mismatch.',
        project_id: 'prj_0001',
        project_title: 'Admin ops proof',
        user_id: 'usr_1',
        reference: 'dispute',
        status: 'dispute:critical',
        actions: [{ id: 'refresh-queue', label: 'Refresh Queue', type: 'refresh_admin_ops' }],
        created_at: now,
      },
      {
        id: 'payout:tsk_1',
        type: 'payout_review',
        severity: 'high',
        title: 'Issue #12 needs payout review',
        body: 'Issue closed while task is still open.',
        project_id: 'prj_0001',
        task_id: 'tsk_1',
        issue_number: 12,
        reference: 'issue:12',
        url: 'https://github.com/mergeos-bounties/mergeos/issues/12',
        status: 'needs_payout_review',
        actions: [
          { id: 'review-prs', label: 'Review PRs', type: 'review_task_pulls' },
          { id: 'open-issue', label: 'Open Issue', type: 'open_url', url: 'https://github.com/mergeos-bounties/mergeos/issues/12' },
        ],
        created_at: now,
      },
      {
        id: 'proposal:ntf_2',
        type: 'proposal_review',
        severity: 'medium',
        title: 'Worker proposal submitted',
        body: 'Worker offered delivery terms for issue #18.',
        project_id: 'prj_0001',
        project_title: 'Admin ops proof',
        task_id: 'tsk_18',
        issue_number: 18,
        user_id: 'usr_worker_1',
        reference: 'proposal:submitted;task:bounty-prj_0001-18;worker:github:worker-dev',
        url: 'https://github.com/mergeos-bounties/mergeos/issues/18',
        status: 'submitted',
        actions: [
          { id: 'open-proposal-task', label: 'Open Task', type: 'open_url', url: 'https://github.com/mergeos-bounties/mergeos/issues/18' },
          { id: 'refresh-queue', label: 'Refresh Queue', type: 'refresh_admin_ops' },
        ],
        created_at: now,
      },
      {
        id: 'security:expired.mergeos.local',
        type: 'security_moderation',
        severity: 'critical',
        title: 'SSL certificate needs review',
        body: 'certificate expired',
        reference: 'expired.mergeos.local',
        status: 'expired',
        actions: [{ id: 'run-ssl-review', label: 'Run SSL Review', type: 'run_ssl_review' }],
        created_at: now,
      },
      {
        id: 'fraud-duplicate:pr',
        type: 'fraud_review',
        severity: 'high',
        title: 'Duplicate payout reference',
        body: 'Two payout rows share one pull request.',
        reference: 'pr:https://github.com/mergeos-bounties/mergeos/pull/404',
        status: 'duplicate_payout_reference',
        actions: [{ id: 'open-proof', label: 'Open Proof', type: 'open_url', url: 'https://github.com/mergeos-bounties/mergeos/pull/404' }],
        created_at: now,
      },
    ],
  };

  assert.equal(validateProtocolDocument(queue).valid, true);

  const invalid = validateProtocolDocument({
    ...queue,
    kind: 'admin_queue',
    stats: { ...queue.stats, total_count: -1 },
    items: [{ ...queue.items[0], type: 'unknown_review' }],
  });
  assert.equal(invalid.valid, false);
  assert(invalid.errors.some((error) => error.path === 'kind'));
  assert(invalid.errors.some((error) => error.path === 'stats.total_count'));
  assert(invalid.errors.some((error) => error.path === 'items[0].type'));
});

test('validates worker proposal protocol documents', () => {
  const now = '2026-06-05T00:00:00.000Z';
  const proposal = {
    protocol_version: 'mergeos.proposal.v1',
    kind: 'proposal',
    proposal: {
      id: 'ntf_worker_1',
      project_id: 'prj_0001',
      project_title: 'Proposal routing',
      task_id: 'bounty-prj_0001-18',
      claim_id: 'bounty-prj_0001-18',
      worker_id: 'github:worker-dev',
      issue_number: 18,
      title: 'Add worker proposal flow',
      cover_letter: 'I can deliver the worker proposal workflow with tests and public protocol proof.',
      bid_cents: 12345,
      estimated_hours: 9.5,
      availability: 'Available this week',
      status: 'submitted',
      reference: 'proposal:submitted;task:bounty-prj_0001-18;worker:github:worker-dev',
      created_at: now,
      updated_at: now,
    },
    worker_notification: {
      id: 'ntf_worker_1',
      user_id: '',
      project_id: 'prj_0001',
      channel: 'proposal',
      subject: 'Proposal submitted: Add worker proposal flow',
      body: 'I can deliver the worker proposal workflow with tests and public protocol proof.',
      status: 'proposal:submitted;task:bounty-prj_0001-18;worker:github:worker-dev',
      created_at: now,
    },
    customer_notification: {
      id: 'ntf_customer_1',
      user_id: '',
      project_id: 'prj_0001',
      channel: 'proposal',
      subject: 'Proposal submitted: Add worker proposal flow',
      body: 'github:worker-dev submitted a proposal for Add worker proposal flow.',
      status: 'proposal:submitted;task:bounty-prj_0001-18;worker:github:worker-dev',
      created_at: now,
    },
  };

  assert.equal(validateProtocolDocument(proposal).valid, true);

  const invalid = validateProtocolDocument({
    ...proposal,
    kind: 'proposal_submission',
    proposal: { ...proposal.proposal, bid_cents: 0, status: 'queued' },
    customer_notification: { ...proposal.customer_notification, channel: 'task' },
  });
  assert.equal(invalid.valid, false);
  assert(invalid.errors.some((error) => error.path === 'kind'));
  assert(invalid.errors.some((error) => error.path === 'proposal.bid_cents'));
  assert(invalid.errors.some((error) => error.path === 'proposal.status'));
  assert(invalid.errors.some((error) => error.path === 'customer_notification.channel'));
});

test('validates customer dashboard protocol documents', () => {
  const now = '2026-06-05T00:00:00.000Z';
  const dashboard = {
    protocol_version: 'mergeos.customer-dashboard.v1',
    kind: 'customer_dashboard',
    project: {
      project_id: 'prj_0001',
      title: 'Dashboard aggregate proof',
      brief: 'Customer delivery workspace with escrow, PRs, tasks, and AI workflow.',
      status: 'funded',
      repo_provider: 'github',
      repo_url: 'https://github.com/mergeos-bounties/mergeos',
      bounty_repo_name: 'mergeos-bounties/mergeos',
      budget_cents: 210000,
      fee_cents: 21000,
      work_pool_cents: 189000,
      task_count: 3,
      open_task_count: 2,
      accepted_task_count: 1,
      agent_task_count: 1,
      human_task_count: 1,
      hybrid_task_count: 1,
      created_at: now,
      updated_at: now,
    },
    escrow: {
      project_id: 'prj_0001',
      project_title: 'Dashboard aggregate proof',
      token_symbol: 'MRG',
      release_status: 'funded',
      budget_cents: 210000,
      fee_cents: 21000,
      work_pool_cents: 189000,
      project_reserve_cents: 210000,
      task_reserve_cents: 189000,
      task_payment_cents: 5000,
      manual_credit_cents: 0,
      released_cents: 5000,
      remaining_cents: 184000,
      overdrawn_cents: 0,
      unallocated_cents: 0,
      paid_task_count: 1,
      open_task_count: 2,
      updated_at: now,
      tasks: [
        {
          task_id: 'tsk_0001',
          issue_number: 1,
          title: 'Wire dashboard data',
          status: 'accepted',
          release_status: 'released',
          reward_cents: 5000,
          paid_cents: 5000,
          remaining_cents: 0,
          overpaid_cents: 0,
          worker_id: 'github:worker-dev',
          updated_at: now,
        },
      ],
    },
    payouts: {
      protocol_version: 'mergeos.payouts.v1',
      kind: 'payouts',
      project_id: 'prj_0001',
      project_title: 'Dashboard aggregate proof',
      token_symbol: 'MRG',
      release_status: 'releasing',
      work_pool_cents: 189000,
      released_cents: 5000,
      remaining_cents: 184000,
      overdrawn_cents: 0,
      task_count: 3,
      paid_task_count: 1,
      open_task_count: 2,
      release_count: 1,
      updated_at: now,
      payouts: [
        {
          task_id: 'tsk_0001',
          issue_number: 1,
          title: 'Wire dashboard data',
          type: 'task_payment',
          status: 'accepted',
          release_status: 'released',
          worker_id: 'github:worker-dev',
          payout_account: 'github:worker-dev',
          reward_cents: 5000,
          paid_cents: 5000,
          remaining_cents: 0,
          overpaid_cents: 0,
          ledger_sequence: 7,
          ledger_entry_count: 1,
          entry_hash: 'd'.repeat(64),
          proof_hash: 'd'.repeat(64),
          reference: 'project:prj_0001;task:tsk_0001',
          released_at: now,
          updated_at: now,
        },
      ],
    },
    deployment: {
      project_id: 'prj_0001',
      project_title: 'Dashboard aggregate proof',
      status: 'validating',
      progress: 67,
      updated_at: now,
      stages: [
        { id: 'deployment_handoff', title: 'Deployment handoff', body: 'Preview is linked.', status: 'complete', tone: 'success', updated_at: now },
      ],
      signals: [
        { id: 'deployment:prj_0001', type: 'deployment_validation', title: 'Deployment', body: 'Preview validation is running.', status: 'validating', created_at: now },
      ],
    },
    ai_workflow: {
      project_id: 'prj_0001',
      project_title: 'Dashboard aggregate proof',
      status: 'orchestrating',
      progress: 71,
      current_step: 'pr_review',
      task_count: 3,
      agent_task_count: 1,
      human_task_count: 1,
      hybrid_task_count: 1,
      ai_action_count: 2,
      updated_at: now,
      stages: [{ id: 'repo_import', title: 'Repo import', status: 'complete' }],
      signals: [{ id: 'agent:review', type: 'agent_action', status: 'processed' }],
    },
    task_graph: {
      project_id: 'prj_0001',
      project_title: 'Dashboard aggregate proof',
      status: 'active',
      progress: 60,
      updated_at: now,
      stats: { node_count: 3, edge_count: 2, ready_count: 1, blocked_count: 0, complete_count: 1, open_count: 2 },
      nodes: [{ id: 'n1', task_id: 'tsk_0001', title: 'Wire dashboard data' }],
      edges: [{ id: 'e1', from: 'n1', to: 'n2', relation: 'sequence' }],
    },
    repository_scan: {
      project_id: 'prj_0001',
      project_title: 'Dashboard aggregate proof',
      status: 'ready',
      summary: 'Scanned repository context for issues and dependencies.',
      stats: { file_count: 12, scanned_files: 10, skipped_files: 2, dependency_files: 1, finding_count: 1 },
      languages: [{ language: 'Go', extension: '.go', file_count: 6 }],
      dependencies: [{ path: 'package.json', ecosystem: 'npm', package_count: 4, has_lockfile: true }],
      findings: [{ id: 'finding_1', severity: 'medium', category: 'quality', title: 'Add tests' }],
      updated_at: now,
    },
    pull_requests: {
      project_id: 'prj_0001',
      project_title: 'Dashboard aggregate proof',
      stats: {
        task_count: 3,
        linked_task_count: 1,
        pull_request_count: 1,
        open_pull_request_count: 1,
        merged_pull_request_count: 0,
        ready_count: 1,
        needs_review_count: 0,
        blocked_count: 0,
        error_count: 0,
        auto_release_ready_count: 1,
      },
      tasks: [{ task_id: 'tsk_0001', monitor_status: 'ready' }],
      updated_at: now,
    },
    proposals: [
      {
        id: 'ntf_worker_1',
        project_id: 'prj_0001',
        project_title: 'Dashboard aggregate proof',
        task_id: 'bounty-prj_0001-2',
        claim_id: 'bounty-prj_0001-2',
        worker_id: 'github:worker-dev',
        issue_number: 2,
        title: 'Wire proposal data',
        cover_letter: 'I can wire proposal data into the customer dashboard.',
        bid_cents: 14000,
        estimated_hours: 8,
        availability: 'Available this week',
        status: 'submitted',
        reference: 'proposal:submitted;task:bounty-prj_0001-2;worker:github:worker-dev',
        created_at: now,
        updated_at: now,
      },
    ],
    updated_at: now,
  };

  assert.equal(validateProtocolDocument(dashboard).valid, true);

  const invalid = validateProtocolDocument({
    ...dashboard,
    kind: 'project_dashboard',
    project: { ...dashboard.project, task_count: -1 },
  });
  assert.equal(invalid.valid, false);
  assert(invalid.errors.some((error) => error.path === 'kind'));
  assert(invalid.errors.some((error) => error.path === 'project.task_count'));
});

test('validates worker dashboard protocol documents', () => {
  const dashboard = {
    protocol_version: 'mergeos.worker-dashboard.v1',
    kind: 'worker_dashboard',
    profile: {
      user_id: 'usr_worker_1',
      name: 'Worker Dev',
      email: 'worker@example.com',
      wallet_address: 'So1anaWorkerWallet1111111111111111111111111',
      github_username: 'worker-dev',
      github_avatar_url: 'https://avatars.githubusercontent.com/u/1001',
    },
    stats: {
      claimed_task_count: 1,
      open_proposal_count: 2,
      submitted_proposal_count: 1,
      reward_cents: 5000,
      reputation_score: 88,
      risk_level: 'low',
      last_paid_at: '2026-06-05T00:00:00.000Z',
    },
    claimed_tasks: [
      {
        id: 'tsk_accepted',
        project_id: 'prj_0001',
        project_title: 'Worker dashboard proof',
        issue_number: 12,
        title: 'Ship worker route',
        acceptance: 'Tests pass and evidence is linked.',
        reward_cents: 5000,
        worker_kind: 'human',
        proof_hash: 'abc123',
        issue_url: 'https://github.com/mergeos-bounties/mergeos/issues/12',
        accepted_at: '2026-06-05T00:00:00.000Z',
      },
    ],
    rewards: [
      {
        sequence: 7,
        type: 'task_payment',
        amount_cents: 5000,
        reference: 'project:prj_0001;task:tsk_accepted',
        entry_hash: 'b'.repeat(64),
        created_at: '2026-06-05T00:00:00.000Z',
      },
    ],
    reputation: [
      { label: 'Score', value: '88 / 100', tone: 'success' },
    ],
    reputation_audit: {
      worker_id: 'github:worker-dev',
      name: 'Worker Dev',
      kind: 'human',
      score: 88,
      level: 'Trusted',
      risk_level: 'low',
      completed_task_count: 1,
      reward_cents: 5000,
      reward_row_count: 1,
      has_github: true,
      has_wallet: true,
      duplicate_identity_count: 0,
      flags: [],
      last_paid_at: '2026-06-05T00:00:00.000Z',
    },
    proposals: [
      {
        id: 'proposal_1',
        claim_id: 'claim_public_1',
        project_id: 'prj_0002',
        project_title: 'Proposal routing',
        issue_number: 18,
        title: 'Add worker match',
        acceptance: 'Route based on identity and reputation.',
        reward_cents: 7000,
        estimated_hours: 4,
        required_worker_kind: 'human',
        match_score: 91,
        match_reasons: ['GitHub identity linked', 'Low risk reputation'],
        evidence_required: ['tests'],
        issue_url: 'https://github.com/mergeos-bounties/mergeos/issues/18',
        created_at: '2026-06-05T00:00:00.000Z',
      },
    ],
    submitted_proposals: [
      {
        id: 'ntf_worker_1',
        project_id: 'prj_0002',
        project_title: 'Proposal routing',
        task_id: 'claim_public_1',
        claim_id: 'claim_public_1',
        worker_id: 'github:worker-dev',
        issue_number: 18,
        title: 'Add worker match',
        cover_letter: 'I can deliver this route with tests and public protocol proof.',
        bid_cents: 7000,
        estimated_hours: 4,
        availability: 'Available this week',
        status: 'submitted',
        reference: 'proposal:submitted;task:claim_public_1;worker:github:worker-dev',
        created_at: '2026-06-05T00:00:00.000Z',
        updated_at: '2026-06-05T00:00:00.000Z',
      },
    ],
    identity_status: [
      { label: 'GitHub', value: 'worker-dev', ready: true },
      { label: 'Wallet', value: 'linked', ready: true },
    ],
  };

  assert.equal(validateProtocolDocument(dashboard).valid, true);

  const invalid = validateProtocolDocument({
    ...dashboard,
    kind: 'worker',
    stats: { ...dashboard.stats, risk_level: 'critical' },
  });
  assert.equal(invalid.valid, false);
  assert(invalid.errors.some((error) => error.path === 'kind'));
  assert(invalid.errors.some((error) => error.path === 'stats.risk_level'));
});

test('validates an agent protocol document', () => {
  const result = validateProtocolDocument({
    protocol_version: 'mergeos.agent.v1',
    kind: 'agent',
    id: 'agt_qa_agent',
    type: 'qa-agent',
    title: 'QA Agent',
    worker_kind: 'agent',
    supported_actions: ['review', 'test'],
    capabilities: ['qa_validation', 'evidence_reporting'],
    task_count: 3,
    open_task_count: 2,
    budget_mrg: 150,
    status: 'active',
    open_task_ids: ['prj_0001:issue:12'],
  });

  assert.equal(result.valid, true);
  assert.deepEqual(result.errors, []);

  const invalid = validateProtocolDocument({
    protocol_version: 'mergeos.agent.v1',
    kind: 'agent',
    id: 'x',
    type: '',
    title: 'QA Agent',
    worker_kind: 'bot',
    supported_actions: ['unknown'],
    capabilities: [],
    task_count: -1,
    open_task_count: 0,
    budget_mrg: 0,
    status: 'busy',
  });
  assert.equal(invalid.valid, false);
  assert(invalid.errors.some((error) => error.path === 'worker_kind'));
  assert(invalid.errors.some((error) => error.path === 'supported_actions[0]'));
});

test('validates a contributor protocol document', () => {
  const result = validateProtocolDocument({
    protocol_version: 'mergeos.contributor.v1',
    kind: 'contributor',
    id: 'ctr_github_maya_dev',
    worker_id: 'github:maya-dev',
    display_name: 'Maya Dev',
    worker_kind: 'hybrid',
    agent_type: 'security-review-agent',
    completed_task_count: 4,
    earned_mrg: 325,
    reputation_score: 92,
    reputation_level: 'elite',
    risk_level: 'low',
    last_paid_at: '2026-06-05T00:00:00.000Z',
    matched_task_ids: ['prj_0001:issue:12'],
    capabilities: ['human_agent_collaboration', 'security_review', 'evidence_reporting'],
    flags: ['github_verified'],
    tags: ['contributor', 'hybrid', 'elite'],
  });

  assert.equal(result.valid, true);
  assert.deepEqual(result.errors, []);

  const invalid = validateProtocolDocument({
    protocol_version: 'mergeos.contributor.v1',
    kind: 'worker',
    id: 'x',
    worker_id: '',
    display_name: 'Maya Dev',
    worker_kind: 'bot',
    completed_task_count: -1,
    earned_mrg: -5,
    reputation_score: 101,
    reputation_level: 'elite',
    risk_level: 'critical',
    last_paid_at: 'not-a-date',
    capabilities: [],
  });
  assert.equal(invalid.valid, false);
  assert(invalid.errors.some((error) => error.path === 'kind'));
  assert(invalid.errors.some((error) => error.path === 'worker_kind'));
  assert(invalid.errors.some((error) => error.path === 'risk_level'));
});

test('validates a task protocol document', () => {
  const result = validateProtocolDocument({
    protocol_version: 'mergeos.task.v1',
    kind: 'task',
    id: 'tsk_0001',
    project_id: 'prj_0001',
    title: 'Fix PayPal return capture',
    reward_mrg: 50,
    estimated_hours: 6.5,
    complexity: 'medium',
    risk_level: 'high',
    worker_kind: 'human',
    acceptance_criteria: ['Frontend test passes', 'PayPal evidence attached'],
    evidence_required: ['tests', 'screenshot'],
    dependencies: ['tsk_0000'],
  });

  assert.equal(result.valid, true);
  assert.deepEqual(result.errors, []);
});

test('validates task claim protocol documents', () => {
  const now = '2026-06-05T00:00:00.000Z';
  const claim = {
    protocol_version: 'mergeos.task-claim.v1',
    kind: 'task_claim',
    id: 'tsk_0001',
    claim_id: 'claim_public_1',
    task_id: 'tsk_0001',
    project_id: 'prj_0001',
    issue_number: 12,
    title: 'Fix PayPal return capture',
    status: 'accepted',
    worker_kind: 'human',
    worker_id: 'github:worker-dev',
    reward_cents: 5000,
    proof_hash: 'abc123proof',
    accepted_at: now,
    task: {
      id: 'tsk_0001',
      project_id: 'prj_0001',
      issue_number: 12,
      title: 'Fix PayPal return capture',
      acceptance: 'Frontend test passes',
      reward_cents: 5000,
      required_worker_kind: 'human',
      suggested_agent_type: '',
      status: 'accepted',
      worker_kind: 'human',
      worker_id: 'github:worker-dev',
      proof_hash: 'abc123proof',
      created_at: now,
      accepted_at: now,
    },
  };

  assert.equal(validateProtocolDocument(claim).valid, true);

  const invalid = validateProtocolDocument({
    ...claim,
    kind: 'task',
    status: 'open',
    worker_kind: 'bot',
    reward_cents: -1,
  });
  assert.equal(invalid.valid, false);
  assert(invalid.errors.some((error) => error.path === 'kind'));
  assert(invalid.errors.some((error) => error.path === 'status'));
  assert(invalid.errors.some((error) => error.path === 'worker_kind'));
  assert(invalid.errors.some((error) => error.path === 'reward_cents'));
});

test('validates project estimate protocol documents', () => {
  const estimate = {
    protocol_version: 'mergeos.estimate.v1',
    kind: 'project_estimate',
    suggested_price_cents: 420000,
    suggested_range: { low_cents: 360000, high_cents: 500000 },
    confidence: 'high',
    breakdown: [
      { category: 'Base scope', amount_cents: 180000, reason: 'Core planning, implementation, review, and delivery work.' },
      { category: 'Technical surface', amount_cents: 120000, reason: 'Multiple technologies increase integration and testing effort.' },
    ],
    assumptions: ['Estimate assumes one production-ready implementation pass plus review and QA.'],
    risks: ['Major scope changes after publishing can move the price range.'],
    editable: true,
  };

  assert.equal(validateProtocolDocument(estimate).valid, true);

  const invalid = validateProtocolDocument({
    ...estimate,
    kind: 'estimate',
    confidence: 'certain',
    suggested_price_cents: -1,
    suggested_range: { low_cents: -1, high_cents: 0 },
    breakdown: [],
  });
  assert.equal(invalid.valid, false);
  assert(invalid.errors.some((error) => error.path === 'kind'));
  assert(invalid.errors.some((error) => error.path === 'confidence'));
  assert(invalid.errors.some((error) => error.path === 'suggested_price_cents'));
  assert(invalid.errors.some((error) => error.path === 'suggested_range.low_cents'));
  assert(invalid.errors.some((error) => error.path === 'breakdown'));
});

test('rejects invalid task fields and unknown versions', () => {
  const invalid = validateProtocolDocument({
    protocol_version: 'mergeos.task.v1',
    kind: 'task',
    id: 'x',
    title: 'x',
    reward_mrg: -1,
    worker_kind: 'bot',
    acceptance_criteria: [],
    unexpected: true,
  });

  assert.equal(invalid.valid, false);
  assert(invalid.errors.some((error) => error.path === 'worker_kind'));
  assert(invalid.errors.some((error) => error.path === 'unexpected'));
  assert.equal(validateProtocolDocument({ protocol_version: 'mergeos.future.v9' }).valid, false);
});

test('validates workflow node and edge references', () => {
  const valid = validateProtocolDocument({
    protocol_version: 'mergeos.workflow.v1',
    kind: 'workflow',
    id: 'wf_0001',
    project_id: 'prj_0001',
    status: 'active',
    progress: 25,
    current_step: 'contributor_routing',
    nodes: [
      { id: 'n1', task_id: 'tsk_1', issue_number: 1, title: 'Implement', lane: 'implementation', status: 'open', reward_mrg: 40, estimated_hours: 5, worker_kind: 'human' },
      { id: 'n2', task_id: 'tsk_2', issue_number: 2, title: 'Validate', lane: 'validation', status: 'ready', reward_mrg: 10, estimated_hours: 1.5, worker_kind: 'agent', agent_type: 'qa-agent', dependencies: ['tsk_1'] },
    ],
    edges: [{ from: 'n1', to: 'n2', relation: 'sequence' }],
  });
  assert.equal(valid.valid, true);

  const invalid = validateProtocolDocument({
    protocol_version: 'mergeos.workflow.v1',
    kind: 'workflow',
    id: 'wf_0001',
    project_id: 'prj_0001',
    progress: 101,
    current_step: 'unknown',
    nodes: [{ id: 'n1', task_id: 'tsk_1', title: 'Implement', lane: 'implementation', status: 'open' }],
    edges: [{ from: 'n1', to: 'missing', relation: 'sequence' }],
  });
  assert.equal(invalid.valid, false);
  assert(invalid.errors.some((error) => error.path === 'progress'));
  assert(invalid.errors.some((error) => error.path === 'current_step'));
  assert(invalid.errors.some((error) => error.path === 'edges[0].to'));
});

test('validates event protocol documents and assertion helper', () => {
  const event = {
    protocol_version: 'mergeos.event.v1',
    kind: 'event',
    id: 'evt_0001',
    type: 'task.paid',
    occurred_at: '2026-06-02T00:00:00.000Z',
    actor: 'mergeos-admin',
    task_id: 'tsk_0001',
    amount_mrg: 50,
    payload: { reference: 'pr:https://github.com/mergeos-bounties/mergeos/pull/120' },
  };

  assert.equal(assertProtocolDocument(event), event);
  assert.throws(() => assertProtocolDocument({ ...event, occurred_at: 'not-a-date' }), /date-time/);

  const agentEvent = validateProtocolDocument({
    protocol_version: 'mergeos.event.v1',
    kind: 'event',
    id: 'evt_agent_tested',
    type: 'agent.tested',
    occurred_at: '2026-06-02T00:00:00.000Z',
    actor: 'QA Agent',
    reference: 'mergeos-bounties/mergeos#777',
    payload: { action: 'test' },
  });
  assert.equal(agentEvent.valid, true);
  assert.deepEqual(agentEvent.errors, []);

  const pullOpenedEvent = validateProtocolDocument({
    protocol_version: 'mergeos.event.v1',
    kind: 'event',
    id: 'evt_pr_opened',
    type: 'pr.opened',
    occurred_at: '2026-06-02T00:00:00.000Z',
    actor: 'github:contributor',
    reference: 'mergeos-bounties/mergeos#151',
    payload: { pull_number: 151 },
  });
  assert.equal(pullOpenedEvent.valid, true);
  assert.deepEqual(pullOpenedEvent.errors, []);

  const unknownEvent = validateProtocolDocument({
    ...event,
    id: 'evt_unknown',
    type: 'pr.unknown',
  });
  assert.equal(unknownEvent.valid, false);
  assert(unknownEvent.errors.some((error) => error.path === 'type'));
});

test('validates public ledger protocol documents', () => {
  const ledger = {
    protocol_version: 'mergeos.ledger.v1',
    kind: 'ledger',
    token_symbol: 'MRG',
    verification: {
      valid: true,
      entry_count: 1,
      last_sequence: 1,
      last_hash: 'b'.repeat(64),
      updated_at: '2026-06-03T00:00:00.000Z',
    },
    entries: [
      {
        sequence: 1,
        type: 'task_payment',
        from_account: 'escrow:task-reserve',
        to_account: 'github:contributor',
        amount_cents: 5000,
        reference: 'project:prj_0001;task:tsk_0001;pr:https://github.com/mergeos-bounties/mergeos/pull/120',
        previous_hash: '0'.repeat(64),
        entry_hash: 'b'.repeat(64),
        created_at: '2026-06-03T00:00:00.000Z',
      },
    ],
  };

  assert.equal(validateProtocolDocument(ledger).valid, true);
  const invalid = validateProtocolDocument({
    ...ledger,
    verification: { ...ledger.verification, last_hash: 'short' },
  });
  assert.equal(invalid.valid, false);
  assert(invalid.errors.some((error) => error.path === 'verification.last_hash'));
});

test('validates repository scan protocol documents', () => {
  const result = validateProtocolDocument({
    protocol_version: 'mergeos.scan.v1',
    kind: 'repository_scan',
    id: 'scan_prj_0001_20260603',
    project_id: 'prj_0001',
    project_title: 'Payment Gateway',
    status: 'ready',
    summary: 'Scanned 12 text files across 18 repository files.',
    updated_at: '2026-06-03T00:00:00.000Z',
    stats: {
      file_count: 18,
      scanned_files: 12,
      skipped_files: 6,
      dependency_files: 2,
      finding_count: 2,
    },
    languages: [
      { language: 'JavaScript', extension: '.js', file_count: 7 },
      { language: 'Go', extension: '.go', file_count: 5 },
    ],
    dependencies: [
      { path: 'package.json', ecosystem: 'npm', package_count: 4, has_lockfile: true },
    ],
    findings: [
      {
        id: 'repo-finding-001',
        severity: 'high',
        category: 'security',
        title: 'Dangerous dynamic JavaScript execution',
        body: 'Dynamic code execution was detected.',
        path: 'src/app.js',
        line: 42,
        signal: 'dangerous_js_execution',
      },
      {
        id: 'repo-finding-002',
        severity: 'medium',
        category: 'dependency',
        title: 'Floating npm dependency version',
        path: 'package.json',
        line: 0,
        signal: 'dependency_unpinned',
      },
    ],
  });

  assert.equal(result.valid, true);
  assert.deepEqual(result.errors, []);

  const invalid = validateProtocolDocument({
    protocol_version: 'mergeos.scan.v1',
    kind: 'repository_scan',
    id: 'scan_1',
    project_id: 'prj_1',
    status: 'ready',
    stats: { file_count: 1, scanned_files: 1, finding_count: 1 },
    findings: [{ id: 'finding', severity: 'unknown', category: 'security', title: 'Bad', signal: 'bad' }],
  });
  assert.equal(invalid.valid, false);
  assert(invalid.errors.some((error) => error.path === 'findings[0].severity'));
});

test('validates repository issue sync events', () => {
  const result = validateProtocolDocument({
    protocol_version: 'mergeos.event.v1',
    kind: 'event',
    id: 'evt_repo_sync_1',
    type: 'repo.issues.synced',
    occurred_at: '2026-06-03T00:00:00.000Z',
    actor: 'mergeos-api',
    project_id: 'prj_0001',
    reference: 'https://github.com/mergeos-bounties/mergeos',
    payload: {
      imported_issue_count: 8,
      added_task_count: 2,
      updated_task_count: 1,
    },
  });

  assert.equal(result.valid, true);
  assert.deepEqual(result.errors, []);
});

test('derives deterministic Solana contract references from ledger entries', () => {
  const entryHash = 'A'.repeat(64);
  const ledgerEntry = {
    sequence: 7,
    type: 'task_payment',
    reference: 'pr:https://github.com/mergeos-bounties/mergeos/pull/120',
    entry_hash: entryHash,
  };

  assert.equal(contractReferenceFromLedger(ledgerEntry), entryHash.toLowerCase());
  assert.equal(contractReferenceFromLedger(`0x${entryHash}`), entryHash.toLowerCase());
  assert.deepEqual(contractReferenceBytes(ledgerEntry), Array(32).fill(170));
  assert.equal(contractReferenceFromLedger(ledgerEntry, { format: 'prefixed-hex' }), `0x${entryHash.toLowerCase()}`);

  const referenceHash = contractReferenceFromLedger({ reference: ledgerEntry.reference });
  assert.match(referenceHash, /^[0-9a-f]{64}$/);
  assert.equal(referenceHash, contractReferenceFromLedger({ reference: ledgerEntry.reference }));
  assert.notEqual(referenceHash, entryHash.toLowerCase());
});

test('derives deterministic legacy wallet migration hashes', () => {
  const tronHash = legacyWalletAddressHash('tron', '  TXYZ987654321  ');
  const trc20Hash = legacyWalletAddressHash('trc20', 'txyz987654321');
  const prefixedHash = legacyWalletAddressHash('trc20', 'tron:TXYZ987654321');
  const evmBytes = legacyWalletAddressHash('ethereum', '0xAbC0000000000000000000000000000000000000', { format: 'bytes' });

  assert.equal(normalizeLegacyChain('TRON'), 'trc20');
  assert.equal(normalizeLegacyChain('Ethereum'), 'evm');
  assert.equal(normalizeLegacyWalletAddress('wallet:0xAbC0000000000000000000000000000000000000'), '0xabc0000000000000000000000000000000000000');
  assert.equal(tronHash, trc20Hash);
  assert.equal(tronHash, prefixedHash);
  assert.match(tronHash, /^[0-9a-f]{64}$/);
  assert.equal(evmBytes.length, 32);
  assert.throws(() => normalizeLegacyChain('btc'), /trc20 or evm/);
  assert.throws(() => legacyWalletAddressHash('trc20', ''), /address is required/);

  const pda = walletMigrationPDASeedMetadata('tron', 'tron:TXYZ987654321');
  assert.deepEqual(pda.pda_seeds, ['wallet-migration', 'trc20', 'legacy_address_hash_bytes']);
  assert.deepEqual(pda.pda_seed_formats, ['utf8', 'utf8', 'bytes32:hex_decode(contract.args.legacy_address_hash)']);
  assert.equal(pda.legacy_address_hash, tronHash);
  assert.equal(pda.legacy_address_hash_bytes.length, 32);
});

test('validates wallet migration protocol documents', () => {
  const legacyAddressHash = 'd'.repeat(64);
  const migration = {
    protocol_version: 'mergeos.wallet-migration.v1',
    kind: 'wallet_migration',
    migration_id: 'wmg_dddddddddddddddd',
    status: 'pending_contract_registration',
    legacy_chain: 'trc20',
    legacy_address: 'TXYZ987654321987654321987654321999',
    legacy_address_hash: legacyAddressHash,
    target_chain: 'solana',
    target_address: 'So1anaWorkerWallet1111111111111111111111111',
    target_account: 'So1anaWorkerWallet1111111111111111111111111',
    token_symbol: 'MRG',
    required_proofs: ['legacy_wallet_ownership_signature', 'anchor_register_legacy_wallet_transaction'],
    contract: {
      network: 'devnet',
      program_id: '',
      program_ready: false,
      instruction: 'register_legacy_wallet',
      pda_seeds: ['wallet-migration', 'trc20', 'legacy_address_hash_bytes'],
      pda_seed_formats: ['utf8', 'utf8', 'bytes32:hex_decode(contract.args.legacy_address_hash)'],
      args: {
        legacy_chain: 'trc20',
        legacy_address_hash: legacyAddressHash,
        solana_wallet: 'So1anaWorkerWallet1111111111111111111111111',
      },
    },
    wallet: {
      address: 'So1anaWorkerWallet1111111111111111111111111',
      account: 'So1anaWorkerWallet1111111111111111111111111',
      chain: 'solana',
      owner_linked: true,
      created_at: '2026-06-05T00:00:00.000Z',
    },
    created_at: '2026-06-05T00:00:00.000Z',
  };

  assert.equal(validateProtocolDocument(migration).valid, true);

  const invalid = validateProtocolDocument({
    ...migration,
    target_chain: 'trc20',
    contract: {
      ...migration.contract,
      pda_seed_formats: ['utf8', 'utf8', 'hex-string'],
    },
  });
  assert.equal(invalid.valid, false);
  assert(invalid.errors.some((error) => error.path === 'target_chain'));
  assert(invalid.errors.some((error) => error.path === 'contract.pda_seed_formats[2]'));
});
