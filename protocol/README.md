# MergeOS Protocol

Open protocol definitions for MergeOS tasks, workflow graphs, and realtime events.

The live app exposes protocol discovery at `GET /api/public/protocol`.

## Documents

- `mergeos.task.v1`: a claimable bounty task with reward, worker lane, dependencies, and evidence requirements.
- `mergeos.workflow.v1`: a project workflow graph with nodes and dependency edges.
- `mergeos.event.v1`: a realtime ledger/workflow event emitted by apps, agents, or integrations.
- `mergeos.scan.v1`: a repository scan document with dependency manifests, language counts, and security/debt findings.

Event types include project funding, task creation/claim/payment, PR review, repository issue sync, deployment updates, ledger records, and agent actions.

## Usage

```js
import { protocolSchemas, validateProtocolDocument } from '@mergeos/protocol';

const result = validateProtocolDocument({
  protocol_version: 'mergeos.task.v1',
  kind: 'task',
  id: 'tsk_0001',
  title: 'Fix payment return flow',
  reward_mrg: 50,
  worker_kind: 'human',
  acceptance_criteria: ['Tests pass', 'Evidence attached'],
});

if (!result.valid) {
  console.error(result.errors);
}
```

The validator is intentionally dependency-free. It covers the fields MergeOS agents need before submitting work, without requiring a full JSON Schema engine.
