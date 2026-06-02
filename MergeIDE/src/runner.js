"use strict";

const fs = require("node:fs/promises");
const path = require("node:path");
const { spawn } = require("node:child_process");
const { buildTaskPrompt } = require("./prompt");
const { providerPreset } = require("./settings");

async function prepareTaskArtifacts(settings, task, options = {}) {
  const workspaceRoot = resolveWorkspaceRoot(settings, options);
  const artifactRoot = options.artifactRoot || path.join(workspaceRoot, ".mergeide", "tasks", task.id);
  await fs.mkdir(artifactRoot, { recursive: true });
  const taskFile = path.join(artifactRoot, "task.json");
  const promptFile = path.join(artifactRoot, "prompt.md");
  const prompt = buildTaskPrompt(task, {
    tokenSymbol: options.tokenSymbol,
    agentType: settings.worker && settings.worker.agentType,
    workspaceRoot
  });
  await fs.writeFile(taskFile, `${JSON.stringify(task, null, 2)}\n`, "utf8");
  await fs.writeFile(promptFile, `${prompt}\n`, "utf8");
  return {
    artifactRoot,
    prompt,
    promptFile,
    taskFile,
    workspaceRoot
  };
}

function resolveWorkspaceRoot(settings, options = {}) {
  const root = options.workspaceRoot || settings.workspace && settings.workspace.root || process.cwd();
  return path.resolve(root);
}

function resolveAIInvocation(settings, artifacts, task) {
  const provider = settings.ai && settings.ai.provider || "custom";
  const preset = providerPreset(provider);
  const command = settings.ai && settings.ai.command ? settings.ai.command : preset.command;
  const rawArgs = settings.ai && Array.isArray(settings.ai.args) && settings.ai.args.length
    ? settings.ai.args
    : preset.args;
  if (!command) {
    throw new Error("AI CLI command is empty. Configure mergeide.aiCommand or MERGEIDE_AI_CLI.");
  }
  const replacements = {
    "{{prompt}}": artifacts.prompt,
    "{{promptFile}}": artifacts.promptFile,
    "{{taskFile}}": artifacts.taskFile,
    "{{taskId}}": task.id
  };
  const args = rawArgs.map((arg) => renderTemplate(String(arg), replacements));
  return { command, args };
}

function renderTemplate(input, replacements) {
  let output = input;
  for (const [key, value] of Object.entries(replacements)) {
    output = output.split(key).join(value);
  }
  return output;
}

async function runAIForTask(settings, task, options = {}) {
  const artifacts = await prepareTaskArtifacts(settings, task, options);
  const invocation = resolveAIInvocation(settings, artifacts, task);
  return spawnAI(invocation.command, invocation.args, {
    cwd: artifacts.workspaceRoot,
    env: process.env,
    stdio: "inherit"
  });
}

function spawnAI(command, args, options) {
  return new Promise((resolve, reject) => {
    const child = spawn(command, args, {
      cwd: options.cwd,
      env: options.env,
      stdio: options.stdio || "inherit",
      shell: false
    });
    child.on("error", reject);
    child.on("exit", (code, signal) => {
      resolve({
        code: code === null ? 1 : code,
        signal
      });
    });
  });
}

module.exports = {
  prepareTaskArtifacts,
  renderTemplate,
  resolveAIInvocation,
  resolveWorkspaceRoot,
  runAIForTask,
  spawnAI
};
