"use strict";

const test = require("node:test");
const assert = require("node:assert/strict");
const {
  agentActionPayload,
  claimTask,
  findTask,
  recordAgentAction,
  submissionPayload,
  submitTaskEvidence
} = require("../src/api");
const { mergeSettings } = require("../src/settings");

function fakeFetch(responses = []) {
  const calls = [];
  const fetchImpl = async (url, options = {}) => {
    calls.push({ url, options });
    const next = responses.shift() || { status: 200, body: {} };
    return {
      ok: next.status >= 200 && next.status < 300,
      status: next.status,
      statusText: next.statusText || "",
      text: async () => JSON.stringify(next.body)
    };
  };
  fetchImpl.calls = calls;
  return fetchImpl;
}

test("claimTask reserves the worker lane without calling payout release", async () => {
  const originalFetch = global.fetch;
  const fetchImpl = fakeFetch([
    {
      status: 200,
      body: {
        protocol_version: "mergeos.task-claim.v1",
        kind: "task_claim",
        claim_id: "prj_public_0001:12",
        status: "claimed"
      }
    }
  ]);
  global.fetch = fetchImpl;
  try {
    const settings = mergeSettings({
      mergeos: { baseUrl: "https://mergeos.shop", token: "mergeide-token" },
      worker: { id: "github:mergeide-agent", agentType: "mergeide-coding-agent" }
    });
    const task = {
      id: "prj_public_0001:12",
      required_worker_kind: "agent"
    };

    const claimed = await claimTask(settings, task);

    assert.equal(claimed.protocol_version, "mergeos.task-claim.v1");
    assert.equal(fetchImpl.calls[0].url, "https://mergeos.shop/api/tasks/prj_public_0001%3A12/claim");
    assert.equal(fetchImpl.calls[0].options.method, "POST");
    assert.equal(fetchImpl.calls[0].options.headers.Authorization, "Bearer mergeide-token");
    assert.equal(fetchImpl.calls[0].options.body, JSON.stringify({
      worker_kind: "agent",
      worker_id: "github:mergeide-agent",
      agent_type: "mergeide-coding-agent"
    }));
    assert.doesNotMatch(fetchImpl.calls[0].url, /\/accept$/);
  } finally {
    global.fetch = originalFetch;
  }
});

test("findTask resolves public claim identifiers from the task list", async () => {
  const originalFetch = global.fetch;
  const fetchImpl = fakeFetch([
    {
      status: 200,
      body: [{
        id: "tsk_internal_12",
        task_id: "tsk_internal_12",
        claim_id: "prj_public_0001:12",
        bounty_id: "prj_public_0001:12",
        title: "Public bounty"
      }]
    }
  ]);
  global.fetch = fetchImpl;
  try {
    const settings = mergeSettings({
      mergeos: { baseUrl: "https://mergeos.shop", token: "mergeide-token" }
    });

    const task = await findTask(settings, "prj_public_0001:12");

    assert.equal(task.id, "tsk_internal_12");
    assert.equal(fetchImpl.calls[0].url, "https://mergeos.shop/api/tasks");
  } finally {
    global.fetch = originalFetch;
  }
});

test("submitTaskEvidence and recordAgentAction send review evidence contracts", async () => {
  const originalFetch = global.fetch;
  const fetchImpl = fakeFetch([
    {
      status: 200,
      body: {
        protocol_version: "mergeos.task-submission.v1",
        kind: "task_submission",
        claim_id: "prj_public_0001:12",
        project_id: "prj_public_0001",
        pull_request_url: "https://github.com/acme/repo/pull/12",
        review_evidence_url: "https://vercel.example/deployments/mergeos",
        review_notes: "Implementation and tests are ready for review."
      }
    },
    {
      status: 201,
      body: {
        protocol_version: "mergeos.agent-action.v1",
        kind: "agent_action",
        action_id: "act_1",
        action: "generate"
      }
    }
  ]);
  global.fetch = fetchImpl;
  try {
    const settings = mergeSettings({
      mergeos: { baseUrl: "https://mergeos.shop", token: "mergeide-token" }
    });
    const task = { id: "prj_public_0001:12", project_id: "prj_public_0001" };
    const submission = submissionPayload({
      prUrl: "https://github.com/acme/repo/pull/12",
      evidenceUrl: "https://vercel.example/deployments/mergeos",
      notes: "Implementation and tests are ready for review."
    });

    const submitted = await submitTaskEvidence(settings, task, submission);
    const action = await recordAgentAction(settings, submitted.project_id, agentActionPayload({
      action: "generate",
      claimID: submitted.claim_id,
      agentType: "mergeide-coding-agent",
      referenceURL: submitted.pull_request_url,
      pullNumber: 12,
      evidence: [submitted.review_notes, submitted.review_evidence_url],
      runbook: ["Claim task", "Submit evidence", "Wait for review"]
    }));

    assert.equal(action.protocol_version, "mergeos.agent-action.v1");
    assert.equal(fetchImpl.calls[0].url, "https://mergeos.shop/api/tasks/prj_public_0001%3A12/submit");
    assert.equal(fetchImpl.calls[0].options.method, "POST");
    assert.equal(fetchImpl.calls[0].options.body, JSON.stringify({
      pull_request_url: "https://github.com/acme/repo/pull/12",
      evidence_url: "https://vercel.example/deployments/mergeos",
      review_notes: "Implementation and tests are ready for review."
    }));
    assert.equal(fetchImpl.calls[1].url, "https://mergeos.shop/api/projects/prj_public_0001/agent-actions");
    assert.equal(fetchImpl.calls[1].options.method, "POST");
    assert.equal(fetchImpl.calls[1].options.body, JSON.stringify({
      action: "generate",
      claim_id: "prj_public_0001:12",
      bounty_id: "prj_public_0001:12",
      agent_type: "mergeide-coding-agent",
      status: "processed",
      reference_url: "https://github.com/acme/repo/pull/12",
      pull_number: 12,
      evidence: [
        "Implementation and tests are ready for review.",
        "https://vercel.example/deployments/mergeos"
      ],
      runbook: ["Claim task", "Submit evidence", "Wait for review"],
      duration_millis: 0
    }));
  } finally {
    global.fetch = originalFetch;
  }
});
