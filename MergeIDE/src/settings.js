"use strict";

const fs = require("node:fs/promises");
const os = require("node:os");
const path = require("node:path");

const DEFAULT_SETTINGS = Object.freeze({
  brand: {
    name: "MergeIDE",
    upstream: "Code OSS / Visual Studio Code workflow"
  },
  mergeos: {
    baseUrl: "http://localhost:8080",
    token: ""
  },
  ai: {
    provider: "codex",
    command: "",
    args: []
  },
  workspace: {
    root: ""
  },
  worker: {
    id: "mergeide:local",
    agentType: "mergeide"
  },
  claim: {
    afterRun: false
  }
});

const PROVIDER_PRESETS = Object.freeze({
  codex: {
    command: "codex",
    args: ["exec", "{{prompt}}"]
  },
  claude: {
    command: "claude",
    args: ["-p", "{{prompt}}"]
  },
  custom: {
    command: "",
    args: ["{{prompt}}"]
  }
});

function settingsPath() {
  if (process.env.MERGEIDE_SETTINGS) {
    return path.resolve(process.env.MERGEIDE_SETTINGS);
  }
  return path.join(os.homedir(), ".mergeide", "settings.json");
}

function clone(value) {
  return JSON.parse(JSON.stringify(value));
}

function mergeSettings(...sources) {
  const output = clone(DEFAULT_SETTINGS);
  for (const source of sources) {
    if (!source || typeof source !== "object") {
      continue;
    }
    mergeObject(output, source);
  }
  output.mergeos.baseUrl = normalizeBaseUrl(output.mergeos.baseUrl);
  output.ai.provider = normalizeProvider(output.ai.provider);
  return output;
}

function mergeObject(target, source) {
  for (const [key, value] of Object.entries(source)) {
    if (value === undefined) {
      continue;
    }
    if (value && typeof value === "object" && !Array.isArray(value)) {
      if (!target[key] || typeof target[key] !== "object" || Array.isArray(target[key])) {
        target[key] = {};
      }
      mergeObject(target[key], value);
      continue;
    }
    target[key] = value;
  }
}

async function loadSettings(filePath = settingsPath(), overrides = {}) {
  const fileSettings = await readSettingsFile(filePath);
  return mergeSettings(fileSettings, envSettings(), overrides);
}

async function readSettingsFile(filePath = settingsPath()) {
  try {
    const raw = await fs.readFile(filePath, "utf8");
    return JSON.parse(raw);
  } catch (error) {
    if (error && error.code === "ENOENT") {
      return {};
    }
    throw error;
  }
}

async function saveSettings(nextSettings, filePath = settingsPath()) {
  const settings = mergeSettings(nextSettings);
  await fs.mkdir(path.dirname(filePath), { recursive: true });
  await fs.writeFile(filePath, `${JSON.stringify(settings, null, 2)}\n`, "utf8");
  return settings;
}

function envSettings(env = process.env) {
  const settings = {};
  if (env.MERGEOS_URL) {
    settings.mergeos = { ...(settings.mergeos || {}), baseUrl: env.MERGEOS_URL };
  }
  if (env.MERGEOS_TOKEN) {
    settings.mergeos = { ...(settings.mergeos || {}), token: env.MERGEOS_TOKEN };
  }
  if (env.MERGEIDE_AI_PROVIDER) {
    settings.ai = { ...(settings.ai || {}), provider: env.MERGEIDE_AI_PROVIDER };
  }
  if (env.MERGEIDE_AI_CLI) {
    settings.ai = { ...(settings.ai || {}), command: env.MERGEIDE_AI_CLI };
  }
  if (env.MERGEIDE_AI_ARGS) {
    settings.ai = { ...(settings.ai || {}), args: parseArgList(env.MERGEIDE_AI_ARGS) };
  }
  if (env.MERGEIDE_WORKSPACE) {
    settings.workspace = { ...(settings.workspace || {}), root: env.MERGEIDE_WORKSPACE };
  }
  if (env.MERGEIDE_WORKER_ID) {
    settings.worker = { ...(settings.worker || {}), id: env.MERGEIDE_WORKER_ID };
  }
  if (env.MERGEIDE_AGENT_TYPE) {
    settings.worker = { ...(settings.worker || {}), agentType: env.MERGEIDE_AGENT_TYPE };
  }
  return settings;
}

function parseArgList(value) {
  if (Array.isArray(value)) {
    return value.map(String);
  }
  const text = String(value || "").trim();
  if (!text) {
    return [];
  }
  if (text.startsWith("[")) {
    const parsed = JSON.parse(text);
    if (!Array.isArray(parsed)) {
      throw new Error("MERGEIDE_AI_ARGS JSON must be an array");
    }
    return parsed.map(String);
  }
  return splitShellLike(text);
}

function splitShellLike(text) {
  const args = [];
  let current = "";
  let quote = "";
  let escaping = false;

  for (const char of text) {
    if (escaping) {
      current += char;
      escaping = false;
      continue;
    }
    if (char === "\\") {
      escaping = true;
      continue;
    }
    if (quote) {
      if (char === quote) {
        quote = "";
      } else {
        current += char;
      }
      continue;
    }
    if (char === "'" || char === "\"") {
      quote = char;
      continue;
    }
    if (/\s/.test(char)) {
      if (current) {
        args.push(current);
        current = "";
      }
      continue;
    }
    current += char;
  }
  if (escaping) {
    current += "\\";
  }
  if (quote) {
    throw new Error("unterminated quote in argument list");
  }
  if (current) {
    args.push(current);
  }
  return args;
}

function normalizeBaseUrl(value) {
  const baseUrl = String(value || DEFAULT_SETTINGS.mergeos.baseUrl).trim();
  return baseUrl.replace(/\/+$/, "");
}

function normalizeProvider(value) {
  const provider = String(value || DEFAULT_SETTINGS.ai.provider).trim().toLowerCase();
  return PROVIDER_PRESETS[provider] ? provider : "custom";
}

function providerPreset(provider) {
  return PROVIDER_PRESETS[normalizeProvider(provider)];
}

module.exports = {
  DEFAULT_SETTINGS,
  PROVIDER_PRESETS,
  envSettings,
  loadSettings,
  mergeSettings,
  normalizeBaseUrl,
  normalizeProvider,
  parseArgList,
  providerPreset,
  readSettingsFile,
  saveSettings,
  settingsPath,
  splitShellLike
};
