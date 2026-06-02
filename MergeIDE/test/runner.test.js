"use strict";

const test = require("node:test");
const assert = require("node:assert/strict");
const { renderTemplate, resolveAIInvocation } = require("../src/runner");
const { mergeSettings } = require("../src/settings");

test("renderTemplate replaces all placeholders", () => {
  const result = renderTemplate("{{taskId}} {{promptFile}} {{taskFile}}", {
    "{{taskId}}": "tsk_1",
    "{{promptFile}}": "prompt.md",
    "{{taskFile}}": "task.json"
  });

  assert.equal(result, "tsk_1 prompt.md task.json");
});

test("resolveAIInvocation uses configured custom command and args", () => {
  const settings = mergeSettings({
    ai: {
      provider: "custom",
      command: "my-ai",
      args: ["run", "--task", "{{taskFile}}", "{{taskId}}"]
    }
  });
  const invocation = resolveAIInvocation(settings, {
    prompt: "hello",
    promptFile: "prompt.md",
    taskFile: "task.json"
  }, { id: "tsk_99" });

  assert.equal(invocation.command, "my-ai");
  assert.deepEqual(invocation.args, ["run", "--task", "task.json", "tsk_99"]);
});
