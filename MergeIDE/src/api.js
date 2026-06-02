"use strict";

const { normalizeBaseUrl } = require("./settings");

async function apiRequest(settings, method, route, body) {
  const baseUrl = normalizeBaseUrl(settings.mergeos && settings.mergeos.baseUrl);
  const headers = {
    Accept: "application/json"
  };
  if (body !== undefined) {
    headers["Content-Type"] = "application/json";
  }
  const token = settings.mergeos && settings.mergeos.token ? String(settings.mergeos.token).trim() : "";
  if (token) {
    headers.Authorization = token.startsWith("Bearer ") ? token : `Bearer ${token}`;
  }
  const response = await fetch(`${baseUrl}${route}`, {
    method,
    headers,
    body: body === undefined ? undefined : JSON.stringify(body)
  });
  const text = await response.text();
  const payload = text ? parseJson(text) : null;
  if (!response.ok) {
    const message = payload && payload.error ? payload.error : `${response.status} ${response.statusText}`;
    throw new Error(`MergeOS ${method} ${route} failed: ${message}`);
  }
  return payload;
}

function parseJson(text) {
  try {
    return JSON.parse(text);
  } catch (error) {
    throw new Error(`MergeOS returned invalid JSON: ${error.message}`);
  }
}

async function login(settings, email, password) {
  return apiRequest(settings, "POST", "/api/auth/login", { email, password });
}

async function listTasks(settings) {
  const tasks = await apiRequest(settings, "GET", "/api/tasks");
  return Array.isArray(tasks) ? tasks : [];
}

async function findTask(settings, taskID) {
  const tasks = await listTasks(settings);
  const task = tasks.find((row) => row && row.id === taskID);
  if (!task) {
    throw new Error(`task ${taskID} was not found in /api/tasks`);
  }
  return task;
}

async function claimTask(settings, task, overrides = {}) {
  const workerKind = overrides.workerKind || task.required_worker_kind || "agent";
  const request = {
    worker_kind: workerKind,
    worker_id: overrides.workerId || settings.worker.id
  };
  const agentType = overrides.agentType || settings.worker.agentType || "mergeide";
  if (workerKind !== "human") {
    request.agent_type = agentType;
  }
  return apiRequest(settings, "POST", `/api/tasks/${encodeURIComponent(task.id)}/accept`, request);
}

module.exports = {
  apiRequest,
  claimTask,
  findTask,
  listTasks,
  login
};
