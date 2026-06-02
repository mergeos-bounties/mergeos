"use strict";

const test = require("node:test");
const assert = require("node:assert/strict");
const { mergeSettings, parseArgList, providerPreset, splitShellLike } = require("../src/settings");

test("provider presets include codex and claude commands", () => {
  assert.equal(providerPreset("codex").command, "codex");
  assert.deepEqual(providerPreset("claude").args, ["-p", "{{prompt}}"]);
});

test("mergeSettings normalizes base url and provider", () => {
  const settings = mergeSettings({
    mergeos: { baseUrl: "http://localhost:8080/" },
    ai: { provider: "unknown" }
  });

  assert.equal(settings.mergeos.baseUrl, "http://localhost:8080");
  assert.equal(settings.ai.provider, "custom");
});

test("parseArgList accepts JSON arrays", () => {
  assert.deepEqual(parseArgList('["exec","{{prompt}}"]'), ["exec", "{{prompt}}"]);
});

test("splitShellLike preserves quoted values", () => {
  assert.deepEqual(splitShellLike('run --input "{{promptFile}}" --flag'), ["run", "--input", "{{promptFile}}", "--flag"]);
});
