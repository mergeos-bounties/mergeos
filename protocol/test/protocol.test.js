import assert from 'node:assert/strict';
import test from 'node:test';
import { assertProtocolDocument, protocolSchemas, schemaForProtocol, validateProtocolDocument } from '../src/index.js';

test('loads stable task, workflow, and event schemas', () => {
  assert.deepEqual(Object.keys(protocolSchemas).sort(), [
    'mergeos.event.v1',
    'mergeos.task.v1',
    'mergeos.workflow.v1',
  ]);
  assert.equal(schemaForProtocol('mergeos.task.v1').title, 'MergeOS Task v1');
});

test('validates a task protocol document', () => {
  const result = validateProtocolDocument({
    protocol_version: 'mergeos.task.v1',
    kind: 'task',
    id: 'tsk_0001',
    project_id: 'prj_0001',
    title: 'Fix PayPal return capture',
    reward_mrg: 50,
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
      { id: 'n1', task_id: 'tsk_1', title: 'Implement', lane: 'implementation', status: 'open', reward_mrg: 40 },
      { id: 'n2', task_id: 'tsk_2', title: 'Validate', lane: 'validation', status: 'ready', reward_mrg: 10 },
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
});
