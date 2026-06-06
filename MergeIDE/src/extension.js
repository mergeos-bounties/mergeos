"use strict";

const path = require("node:path");
const { listTasks } = require("./api");
const { loadSettings, mergeSettings, settingsPath } = require("./settings");

let vscode;

function activate(context) {
  vscode = require("vscode");
  context.subscriptions.push(
    vscode.commands.registerCommand("mergeide.showTasks", () => showTasks(context)),
    vscode.commands.registerCommand("mergeide.runTask", () => runTask(context)),
    vscode.commands.registerCommand("mergeide.claimTask", () => claimTask(context)),
    vscode.commands.registerCommand("mergeide.submitTask", () => submitTask(context)),
    vscode.commands.registerCommand("mergeide.configureAiCli", () => configureAiCli())
  );
}

function deactivate() {}

async function showTasks(context) {
  const settings = await extensionSettings();
  const tasks = await listTasks(settings);
  const picked = await pickTask(tasks);
  if (!picked) {
    return;
  }
  const action = await vscode.window.showQuickPick(["Run with AI CLI", "Claim task", "Submit evidence", "Open prompt only"], {
    placeHolder: picked.label
  });
  if (action === "Run with AI CLI") {
    openTerminal(context, ["run", picked.task.id]);
  } else if (action === "Claim task") {
    openTerminal(context, ["claim", picked.task.id]);
  } else if (action === "Submit evidence") {
    await submitTaskCommand(context, picked.task.id);
  } else if (action === "Open prompt only") {
    openTerminal(context, ["prompt", picked.task.id]);
  }
}

async function runTask(context) {
  const taskID = await vscode.window.showInputBox({ prompt: "MergeOS task id" });
  if (taskID) {
    openTerminal(context, ["run", taskID]);
  }
}

async function claimTaskCommand(context) {
  const taskID = await vscode.window.showInputBox({ prompt: "MergeOS task id to claim" });
  if (taskID) {
    openTerminal(context, ["claim", taskID]);
  }
}

async function claimTask(context) {
  return claimTaskCommand(context);
}

async function submitTaskCommand(context, providedTaskID = "") {
  const taskID = providedTaskID || await vscode.window.showInputBox({ prompt: "MergeOS task id to submit" });
  if (!taskID) {
    return;
  }
  const prURL = await vscode.window.showInputBox({ prompt: "GitHub pull request URL" });
  const evidenceURL = await vscode.window.showInputBox({ prompt: "Optional evidence URL", value: "" });
  const notes = await vscode.window.showInputBox({ prompt: "Review notes", value: "MergeIDE submitted implementation evidence for review." });
  const args = ["submit", taskID];
  if (prURL) args.push("--pr-url", prURL);
  if (evidenceURL) args.push("--evidence-url", evidenceURL);
  if (notes) args.push("--notes", notes);
  openTerminal(context, args);
}

async function submitTask(context) {
  return submitTaskCommand(context);
}

async function configureAiCli() {
  const uri = vscode.Uri.file(settingsPath());
  await vscode.window.showTextDocument(uri, { preview: false });
}

async function pickTask(tasks) {
  const items = tasks.map((task) => ({
    label: `${task.id} ${task.title}`,
    description: `${task.status} · ${task.required_worker_kind} · ${formatMRG(task.reward_cents)} MRG`,
    detail: task.acceptance,
    task
  }));
  return vscode.window.showQuickPick(items, {
    matchOnDescription: true,
    matchOnDetail: true,
    placeHolder: "Select a MergeOS task"
  });
}

function openTerminal(context, args) {
  const terminal = vscode.window.createTerminal({ name: "MergeIDE" });
  const binPath = path.join(context.extensionPath, "bin", "mergeide.js");
  const command = ["node", quote(binPath), ...args.map(quote)].join(" ");
  terminal.show();
  terminal.sendText(command);
}

async function extensionSettings() {
  const config = vscode.workspace.getConfiguration("mergeide");
  const workspaceFolder = vscode.workspace.workspaceFolders && vscode.workspace.workspaceFolders[0];
  const configuredWorkspaceRoot = configuredValue(config, "workspaceRoot");
  const workspaceRoot = configuredWorkspaceRoot || (workspaceFolder ? workspaceFolder.uri.fsPath : "");
  const vscodeSettings = {
    mergeos: {
      baseUrl: configuredValue(config, "mergeosUrl"),
      token: configuredValue(config, "token")
    },
    ai: {
      provider: configuredValue(config, "aiProvider"),
      command: configuredValue(config, "aiCommand"),
      args: configuredValue(config, "aiArgs")
    },
    workspace: {
      root: workspaceRoot
    },
    worker: {
      id: configuredValue(config, "workerId"),
      agentType: configuredValue(config, "agentType")
    },
    claim: {
      afterRun: configuredValue(config, "autoClaimAfterRun")
    }
  };
  return mergeSettings(await loadSettings(), vscodeSettings);
}

function configuredValue(config, key) {
  const inspection = config.inspect(key);
  if (!inspection) {
    return undefined;
  }
  if (inspection.workspaceFolderValue !== undefined) {
    return inspection.workspaceFolderValue;
  }
  if (inspection.workspaceValue !== undefined) {
    return inspection.workspaceValue;
  }
  if (inspection.globalValue !== undefined) {
    return inspection.globalValue;
  }
  return undefined;
}

function formatMRG(cents) {
  return (Number(cents || 0) / 100).toFixed(2);
}

function quote(value) {
  const text = String(value);
  if (!/[\s"]/g.test(text)) {
    return text;
  }
  return `"${text.replace(/"/g, '\\"')}"`;
}

module.exports = {
  activate,
  deactivate
};
