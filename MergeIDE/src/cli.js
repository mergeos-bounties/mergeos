"use strict";

const path = require("node:path");
const { claimTask, findTask, listTasks, login } = require("./api");
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
    console.log(`Claimed ${claimed.id} with proof ${claimed.proof_hash || "n/a"}`);
  }
}

async function claimCommand(flags) {
  const taskID = requiredPositional(flags, "task id");
  const settings = await loadSettings(flags.settings, settingsFromFlags(flags));
  const task = await findTask(settings, taskID);
  const claimed = await claimTask(settings, task, claimOverrides(flags));
  console.log(`Claimed ${claimed.id} with proof ${claimed.proof_hash || "n/a"}`);
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
    console.log(`Claimed ${claimed.id} with proof ${claimed.proof_hash || "n/a"}`);
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
  mergeide run <task-id> [--claim]
  mergeide claim <task-id>
  mergeide next [--kind agent] [--claim]

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
