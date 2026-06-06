"use strict";

const path = require("node:path");
const { claimTask, findTask, listTasks, login, recordAgentAction, submitTaskEvidence } = require("./api");
const { loadSettings, mergeSettings, parseArgList, readSettingsFile, saveSettings, settingsPath } = require("./settings");
const { prepareTaskArtifacts, runAIForTask } = require("./runner");

async function main(argv) {
  const [command = "help", ...rest] = argv;
  const flags = parseFlags(rest);
  switch (command) {
    case "configure":
      return configure(flags);
    case "login":
      return loginCommand(flags);
    case "tasks":
      return tasksCommand(flags);
    case "prompt":
      return promptCommand(flags);
    case "run":
      return runCommand(flags);
    case "claim":
      return claimCommand(flags);
    case "submit":
      return submitCommand(flags);
    case "next":
      return nextCommand(flags);
    case "help":
    case "--help":
    case "-h":
      return help();
    default:
      throw new Error(`unknown command: ${command}`);
  }
}

async function configure(flags) {
  const filePath = flags.settings || settingsPath();
  const current = await readSettingsFile(filePath);
  const updates = settingsFromFlags(flags);
  const next = mergeSettings(current, updates);
  await saveSettings(next, filePath);
  console.log(`MergeIDE settings saved to ${filePath}`);
}

async function loginCommand(flags) {
  const email = requiredFlag(flags, "email");
  const password = requiredFlag(flags, "password");
  const filePath = flags.settings || settingsPath();
  const current = await readSettingsFile(filePath);
  const settings = mergeSettings(current, settingsFromFlags(flags));
  const auth = await login(settings, email, password);
  const next = mergeSettings(settings, { mergeos: { token: auth.token } });
  await saveSettings(next, filePath);
  console.log(`Logged in as ${auth.user && auth.user.email ? auth.user.email : email}`);
}

async function tasksCommand(flags) {
  const settings = await loadSettings(flags.settings, settingsFromFlags(flags));
  let tasks = await listTasks(settings);
  if (flags.open) {
    tasks = tasks.filter((task) => task.status === "open");
  }
  if (flags.json) {
    console.log(JSON.stringify(tasks, null, 2));
    return;
  }
  printTasks(tasks);
}

async function promptCommand(flags) {
  const taskID = requiredPositional(flags, "task id");
  const settings = await loadSettings(flags.settings, settingsFromFlags(flags));
  const task = await findTask(settings, taskID);
  const artifacts = await prepareTaskArtifacts(settings, task, {
    workspaceRoot: flags.workspace
  });
  console.log(artifacts.promptFile);
}

async function runCommand(flags) {
  const taskID = requiredPositional(flags, "task id");
  const settings = await loadSettings(flags.settings, settingsFromFlags(flags));
  const task = await findTask(settings, taskID);
  const result = await runAIForTask(settings, task, {
    workspaceRoot: flags.workspace
  });
  if (result.code !== 0) {
    throw new Error(`AI CLI exited with code ${result.code}`);
  }
  if (flags.claim || settings.claim.afterRun) {
    const claimed = await claimTask(settings, task, claimOverrides(flags));
    printClaimed(claimed);
  }
  if (shouldSubmitAfterRun(flags)) {
    await submitAndRecord(settings, task, flags);
  }
}

async function claimCommand(flags) {
  const taskID = requiredPositional(flags, "task id");
  const settings = await loadSettings(flags.settings, settingsFromFlags(flags));
  const task = await findTask(settings, taskID);
  const claimed = await claimTask(settings, task, claimOverrides(flags));
  printClaimed(claimed);
}

async function submitCommand(flags) {
  const taskID = requiredPositional(flags, "task id");
  const settings = await loadSettings(flags.settings, settingsFromFlags(flags));
  const task = await findTask(settings, taskID);
  await submitAndRecord(settings, task, flags);
}

async function nextCommand(flags) {
  const settings = await loadSettings(flags.settings, settingsFromFlags(flags));
  const tasks = await listTasks(settings);
  const task = selectNextTask(tasks, flags);
  if (!task) {
    console.log("No open MergeOS task matched the current filters.");
    return;
  }
  console.log(`Selected ${task.id}: ${task.title}`);
  const result = await runAIForTask(settings, task, {
    workspaceRoot: flags.workspace
  });
  if (result.code !== 0) {
    throw new Error(`AI CLI exited with code ${result.code}`);
  }
  if (flags.claim || settings.claim.afterRun) {
    const claimed = await claimTask(settings, task, claimOverrides(flags));
    printClaimed(claimed);
  }
  if (shouldSubmitAfterRun(flags)) {
    await submitAndRecord(settings, task, flags);
  }
}

function parseFlags(args) {
  const flags = { _: [] };
  for (let index = 0; index < args.length; index += 1) {
    const item = args[index];
    if (!item.startsWith("--")) {
      flags._.push(item);
      continue;
    }
    const raw = item.slice(2);
    const [key, inlineValue] = raw.split(/=(.*)/s).filter(Boolean);
    if (inlineValue !== undefined) {
      flags[toCamel(key)] = inlineValue;
      continue;
    }
    const next = args[index + 1];
    if (!next || next.startsWith("--")) {
      flags[toCamel(key)] = true;
      continue;
    }
    flags[toCamel(key)] = next;
    index += 1;
  }
  return flags;
}

function settingsFromFlags(flags) {
  const updates = {};
  if (flags.mergeosUrl) {
    updates.mergeos = { ...(updates.mergeos || {}), baseUrl: flags.mergeosUrl };
  }
  if (flags.token) {
    updates.mergeos = { ...(updates.mergeos || {}), token: flags.token };
  }
  if (flags.provider) {
    updates.ai = { ...(updates.ai || {}), provider: flags.provider };
  }
  if (flags.command) {
    updates.ai = { ...(updates.ai || {}), command: flags.command };
  }
  if (flags.args) {
    updates.ai = { ...(updates.ai || {}), args: parseArgList(flags.args) };
  }
  if (flags.workspace) {
    updates.workspace = { ...(updates.workspace || {}), root: path.resolve(flags.workspace) };
  }
  if (flags.workerId) {
    updates.worker = { ...(updates.worker || {}), id: flags.workerId };
  }
  if (flags.agentType) {
    updates.worker = { ...(updates.worker || {}), agentType: flags.agentType };
  }
  if (flags.autoClaim !== undefined) {
    updates.claim = { afterRun: flags.autoClaim === true || flags.autoClaim === "true" };
  }
  return updates;
}

function claimOverrides(flags) {
  return {
    workerId: flags.workerId,
    workerKind: flags.workerKind,
    agentType: flags.agentType
  };
}

async function submitAndRecord(settings, task, flags) {
  const payload = submissionFromFlags(flags);
  const submitted = await submitTaskEvidence(settings, task, payload);
  console.log(`Submitted ${submitted.claim_id || submitted.id} for review`);
  if (flags.agentAction === false || flags.noAgentAction) {
    return submitted;
  }
  const action = await recordAgentAction(settings, submitted.project_id || task.project_id || task.projectID, agentActionFromSubmission(settings, task, submitted, flags));
  console.log(`Recorded agent ${action.action} evidence ${action.action_id || action.id}`);
  return submitted;
}

function submissionFromFlags(flags) {
  const payload = {
    pull_request_url: flags.pullRequestUrl || flags.prUrl,
    evidence_url: flags.evidenceUrl,
    review_notes: flags.reviewNotes || flags.notes
  };
  if (!payload.pull_request_url && !payload.evidence_url && !payload.review_notes) {
    throw new Error("--pr-url, --pull-request-url, --evidence-url, or --notes is required");
  }
  return payload;
}

function agentActionFromSubmission(settings, task, submitted, flags) {
  const pullRequestURL = submitted.pull_request_url || flags.pullRequestUrl || flags.prUrl || "";
  const evidenceURL = submitted.review_evidence_url || flags.evidenceUrl || "";
  const referenceURL = flags.referenceUrl || pullRequestURL || evidenceURL;
  const notes = submitted.review_notes || flags.reviewNotes || flags.notes || "MergeIDE submitted task evidence for review.";
  return {
    action: flags.action || "generate",
    claim_id: submitted.claim_id || task.claim_id || task.id,
    bounty_id: submitted.claim_id || task.claim_id || task.id,
    agent_type: flags.agentType || settings.worker.agentType || task.suggested_agent_type || "mergeide",
    status: flags.status || "processed",
    reference_url: referenceURL,
    pull_number: flags.pullNumber || pullNumberFromURL(pullRequestURL),
    labels: flags.labels,
    context_urls: defaultContextURLs(submitted),
    evidence: [notes, pullRequestURL, evidenceURL].filter(Boolean),
    runbook: [
      "Claim the funded task before recording evidence.",
      "Submit pull request or external evidence for customer review.",
      "Wait for customer or admin release before payout."
    ],
    delegated_by: flags.delegatedBy,
    design_agent: flags.designAgent,
    subagent_type: flags.subagentType,
    delegation_chain: flags.delegationChain
  };
}

function shouldSubmitAfterRun(flags) {
  return Boolean(flags.submit || flags.prUrl || flags.pullRequestUrl || flags.evidenceUrl || flags.notes || flags.reviewNotes);
}

function defaultContextURLs(submitted) {
  const projectID = submitted.project_id || submitted.projectID || "";
  const claimID = submitted.claim_id || submitted.claimID || "";
  return [
    claimID ? `/api/public/protocol/tasks?task_id=${encodeURIComponent(claimID)}` : "",
    projectID ? `/api/public/projects/${encodeURIComponent(projectID)}/workflow` : "",
    projectID ? `/api/public/projects/${encodeURIComponent(projectID)}/pull-requests` : ""
  ].filter(Boolean);
}

function pullNumberFromURL(value) {
  const match = String(value || "").match(/\/pull\/(\d+)(?:\b|[/?#])/);
  return match ? Number(match[1]) : 0;
}

function printClaimed(claimed) {
  const id = claimed.claim_id || claimed.id;
  const worker = claimed.worker_id ? ` by ${claimed.worker_id}` : "";
  console.log(`Claimed ${id}${worker}; payout is pending review`);
}

function selectNextTask(tasks, flags) {
  const openTasks = tasks.filter((task) => task && task.status === "open");
  const kind = flags.kind;
  const agent = flags.agent;
  return openTasks.find((task) => {
    if (kind && task.required_worker_kind !== kind) {
      return false;
    }
    if (agent && task.suggested_agent_type !== agent) {
      return false;
    }
    return true;
  });
}

function printTasks(tasks) {
  if (!tasks.length) {
    console.log("No MergeOS tasks found.");
    return;
  }
  for (const task of tasks) {
    const reward = (Number(task.reward_cents || 0) / 100).toFixed(2);
    console.log(`${task.id}\t${task.status}\t${task.required_worker_kind}\t${reward} MRG\t${task.title}`);
  }
}

function requiredFlag(flags, key) {
  if (!flags[key]) {
    throw new Error(`--${key.replace(/[A-Z]/g, (char) => `-${char.toLowerCase()}`)} is required`);
  }
  return flags[key];
}

function requiredPositional(flags, label) {
  const value = flags._[0];
  if (!value) {
    throw new Error(`${label} is required`);
  }
  return value;
}

function toCamel(value) {
  return value.replace(/-([a-z])/g, (_, char) => char.toUpperCase());
}

function help() {
  console.log(`MergeIDE

Usage:
  mergeide configure --mergeos-url http://localhost:8080 --provider claude --worker-id github:you
  mergeide login --email admin@gmail.com --password Admin123
  mergeide tasks --open
  mergeide prompt <task-id>
  mergeide run <task-id> [--claim] [--submit --pr-url <url>]
  mergeide claim <task-id>
  mergeide submit <task-id> --pr-url <url> [--evidence-url <url>] [--notes <text>]
  mergeide next [--kind agent] [--claim] [--submit --pr-url <url>]

AI CLI placeholders:
  {{prompt}}     Full task prompt
  {{promptFile}} Prompt markdown path
  {{taskFile}}   Task JSON path
  {{taskId}}     MergeOS task id
`);
}

module.exports = {
  main,
  parseFlags,
  selectNextTask,
  settingsFromFlags
};
