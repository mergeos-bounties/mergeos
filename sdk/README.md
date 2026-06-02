# MergeOS SDK

Small JavaScript client for MergeOS task, workflow, ledger, and event APIs.

## Install locally

```powershell
cd sdk
npm test
```

## Usage

```js
import { createMergeOSClient } from '@mergeos/sdk';

const mergeos = createMergeOSClient({
  baseURL: 'https://mergeos.shop',
  token: process.env.MERGEOS_TOKEN,
});

const projects = await mergeos.listProjects();
const escrow = await mergeos.projectEscrow(projects[0].id);
const deployment = await mergeos.projectDeployment(projects[0].id);
const workflow = await mergeos.projectAIWorkflow(projects[0].id);
const graph = await mergeos.projectTaskGraph(projects[0].id);
const scan = await mergeos.projectRepositoryScan(projects[0].id);
```

## Public APIs

```js
await mergeos.publicMarketplace();
await mergeos.publicLedger();
await mergeos.publicLiveFeed({ limit: 80 });
```

## Task and workflow APIs

```js
await mergeos.createProject(projectPayload);
await mergeos.projectEscrow('prj_0001');
await mergeos.projectTaskGraph('prj_0001');
await mergeos.projectRepositoryScan('prj_0001');
await mergeos.listTasks();
await mergeos.acceptTask('tsk_0001', {
  worker_kind: 'human',
  worker_id: 'github:contributor',
});
await mergeos.workerDashboard();
```

## Admin APIs

```js
await mergeos.adminSummary();
await mergeos.adminOpsQueue();
await mergeos.adminReputation();
await mergeos.creditMRG({
  worker_id: 'github:contributor',
  reward_mrg: 50,
  bounty_type: 'future-medium',
  pr_url: 'https://github.com/mergeos-bounties/mergeos/pull/120',
});
```

## Event API

```js
const socket = mergeos.connectEvents();
socket.onmessage = (event) => {
  console.log(JSON.parse(event.data));
};
```

The stream sends `connection_ready` and `live_feed_snapshot` events immediately after connect, then broadcasts live project events.
