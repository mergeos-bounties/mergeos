import assert from 'node:assert/strict';
import test from 'node:test';
import { assertProtocolDocument, protocolSchemas, schemaForProtocol, validateProtocolDocument } from '../src/index.js';

test('loads stable task, workflow, and event schemas', () => {
  assert.deepEqual(Object.keys(protocolSchemas).sort(), [
    'mergeos.agent.v1',
    'mergeos.event.v1',
    'mergeos.scan.v1',
    'mergeos.task.v1',
    'mergeos.workflow.v1',
  ]);
  assert.equal(schemaForProtocol('mergeos.task.v1').title, 'MergeOS Task v1');
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
    nodes: [{ id: 'n1', task_id: 'tsk_1', title: 'Implement', lane: 'implementation', status: 'open' }],
    edges: [{ from: 'n1', to: 'missing', relation: 'sequence' }],
  });
  assert.equal(invalid.valid, false);
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
