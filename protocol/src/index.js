import { readFileSync } from 'node:fs';

const schemaFiles = {
  'mergeos.admin-ops.v1': '../schemas/admin-ops.v1.schema.json',
  'mergeos.agent.v1': '../schemas/agent.v1.schema.json',
  'mergeos.customer-dashboard.v1': '../schemas/customer-dashboard.v1.schema.json',
  'mergeos.task.v1': '../schemas/task.v1.schema.json',
  'mergeos.workflow.v1': '../schemas/workflow.v1.schema.json',
  'mergeos.event.v1': '../schemas/event.v1.schema.json',
  'mergeos.ledger.v1': '../schemas/ledger.v1.schema.json',
  'mergeos.scan.v1': '../schemas/scan.v1.schema.json',
  'mergeos.worker-dashboard.v1': '../schemas/worker-dashboard.v1.schema.json',
};

export const protocolSchemas = Object.freeze(
  Object.fromEntries(
    Object.entries(schemaFiles).map(([version, path]) => [
      version,
      Object.freeze(JSON.parse(readFileSync(new URL(path, import.meta.url), 'utf8'))),
    ]),
  ),
);

export function schemaForProtocol(value = {}) {
  const version = typeof value === 'string' ? value : value.protocol_version;
  return protocolSchemas[version] || null;
}

export function validateProtocolDocument(document) {
  const schema = schemaForProtocol(document);
  if (!schema) {
    return {
      valid: false,
      errors: [{ path: 'protocol_version', message: 'unsupported protocol_version' }],
    };
  }

  const errors = [];
  validateValue(document, schema, '', errors);
  if (document && document.protocol_version === 'mergeos.workflow.v1') {
    validateWorkflowEdges(document, errors);
  }
  return { valid: errors.length === 0, errors };
}

export function assertProtocolDocument(document) {
  const result = validateProtocolDocument(document);
  if (!result.valid) {
    const message = result.errors.map((error) => `${error.path || '$'}: ${error.message}`).join('; ');
    throw new Error(`Invalid MergeOS protocol document: ${message}`);
  }
  return document;
}

function validateValue(value, schema, path, errors) {
  if (schema.const !== undefined && value !== schema.const) {
    errors.push({ path, message: `must equal ${JSON.stringify(schema.const)}` });
    return;
  }

  if (schema.enum && !schema.enum.includes(value)) {
    errors.push({ path, message: `must be one of ${schema.enum.join(', ')}` });
    return;
  }

  if (schema.type && !matchesType(value, schema.type)) {
    errors.push({ path, message: `must be ${schema.type}` });
    return;
  }

  if (schema.type === 'object') {
    validateObject(value, schema, path, errors);
    return;
  }

  if (schema.type === 'array') {
    validateArray(value, schema, path, errors);
    return;
  }

  if (schema.type === 'string') {
    validateString(value, schema, path, errors);
    return;
  }

  if (schema.type === 'number' || schema.type === 'integer') {
    validateNumber(value, schema, path, errors);
  }
}

function validateObject(value, schema, path, errors) {
  if (!value || typeof value !== 'object' || Array.isArray(value)) return;

  for (const field of schema.required || []) {
    if (value[field] === undefined) {
      errors.push({ path: joinPath(path, field), message: 'is required' });
    }
  }

  const properties = schema.properties || {};
  if (schema.additionalProperties === false) {
    for (const field of Object.keys(value)) {
      if (!properties[field]) {
        errors.push({ path: joinPath(path, field), message: 'is not allowed' });
      }
    }
  }

  for (const [field, fieldSchema] of Object.entries(properties)) {
    if (value[field] !== undefined) {
      validateValue(value[field], fieldSchema, joinPath(path, field), errors);
    }
  }
}

function validateArray(value, schema, path, errors) {
  if (!Array.isArray(value)) return;
  if (schema.minItems !== undefined && value.length < schema.minItems) {
    errors.push({ path, message: `must contain at least ${schema.minItems} item(s)` });
  }
  if (!schema.items) return;
  value.forEach((item, index) => validateValue(item, schema.items, `${path}[${index}]`, errors));
}

function validateString(value, schema, path, errors) {
  if (typeof value !== 'string') return;
  if (schema.minLength !== undefined && value.length < schema.minLength) {
    errors.push({ path, message: `must be at least ${schema.minLength} characters` });
  }
  if (schema.maxLength !== undefined && value.length > schema.maxLength) {
    errors.push({ path, message: `must be at most ${schema.maxLength} characters` });
  }
  if (schema.format === 'date-time' && Number.isNaN(Date.parse(value))) {
    errors.push({ path, message: 'must be an ISO date-time string' });
  }
}

function validateNumber(value, schema, path, errors) {
  if (typeof value !== 'number' || Number.isNaN(value)) return;
  if (schema.type === 'integer' && !Number.isInteger(value)) {
    errors.push({ path, message: 'must be an integer' });
  }
  if (schema.minimum !== undefined && value < schema.minimum) {
    errors.push({ path, message: `must be greater than or equal to ${schema.minimum}` });
  }
  if (schema.maximum !== undefined && value > schema.maximum) {
    errors.push({ path, message: `must be less than or equal to ${schema.maximum}` });
  }
}

function validateWorkflowEdges(document, errors) {
  const nodeIDs = new Set((document.nodes || []).map((node) => node.id));
  for (const [index, edge] of (document.edges || []).entries()) {
    if (!nodeIDs.has(edge.from)) {
      errors.push({ path: `edges[${index}].from`, message: 'must reference an existing node id' });
    }
    if (!nodeIDs.has(edge.to)) {
      errors.push({ path: `edges[${index}].to`, message: 'must reference an existing node id' });
    }
  }
}

function matchesType(value, type) {
  switch (type) {
    case 'object':
      return value !== null && typeof value === 'object' && !Array.isArray(value);
    case 'array':
      return Array.isArray(value);
    case 'string':
      return typeof value === 'string';
    case 'number':
      return typeof value === 'number' && !Number.isNaN(value);
    case 'integer':
      return Number.isInteger(value);
    case 'boolean':
      return typeof value === 'boolean';
    default:
      return true;
  }
}

function joinPath(parent, field) {
  return parent ? `${parent}.${field}` : field;
}
