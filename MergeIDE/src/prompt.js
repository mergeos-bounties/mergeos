"use strict";

function buildTaskPrompt(task, options = {}) {
  const tokenSymbol = options.tokenSymbol || "MRG";
  const taskID = value(task.id);
  const title = value(task.title, "Untitled task");
  const acceptance = value(task.acceptance, "No acceptance criteria supplied.");
  const issueURL = value(task.issue_url || task.issueURL, "");
  const reward = Number(task.reward_cents || 0) / 100;
  const workerKind = value(task.required_worker_kind, "agent");
  const suggestedAgent = value(task.suggested_agent_type, options.agentType || "mergeide");
  const workspaceRoot = value(options.workspaceRoot, "the current repository");

  return [
    "# MergeIDE Task",
    "",
    "You are running inside MergeIDE, a Visual Studio Code style workspace connected to MergeOS.",
    `Workspace: ${workspaceRoot}`,
    "",
    "Complete the MergeOS task below, verify your work, and create one git commit that can be submitted for review.",
    "",
    "## Task",
    `- ID: ${taskID}`,
    `- Title: ${title}`,
    `- Reward: ${reward.toFixed(2)} ${tokenSymbol}`,
    `- Required worker kind: ${workerKind}`,
    `- Suggested agent: ${suggestedAgent}`,
    issueURL ? `- Issue URL: ${issueURL}` : "- Issue URL: not provided",
    "",
    "## Acceptance",
    acceptance,
    "",
    "## Required Workflow",
    "1. Inspect the repository before editing.",
    "2. Keep changes scoped to this task.",
    "3. Run the most relevant tests or verification commands available in the repo.",
    `4. Commit the finished work with a message that starts with \"MergeIDE ${taskID}:\".`,
    "5. Submit pull request or evidence URL back to MergeOS for customer/admin review.",
    "6. Do not release payout yourself; payout is released only after review.",
    "",
    "Use mergeide claim to reserve the worker lane, then mergeide submit --pr-url <url> to attach review evidence."
  ].join("\n");
}

function value(input, fallback = "") {
  const text = input === undefined || input === null ? "" : String(input).trim();
  return text || fallback;
}

module.exports = {
  buildTaskPrompt
};
