import { createHash } from 'node:crypto';
import { readFileSync } from 'node:fs';

const schemaFiles = {
  'mergeos.agent-action.v1': '../schemas/agent-action.v1.schema.json',
  'mergeos.agent-queue.v1': '../schemas/agent-queue.v1.schema.json',
  'mergeos.agent-runbook.v1': '../schemas/agent-runbook.v1.schema.json',
  'mergeos.admin-ops.v1': '../schemas/admin-ops.v1.schema.json',
  'mergeos.agent.v1': '../schemas/agent.v1.schema.json',
  'mergeos.ai-workflow.v1': '../schemas/ai-workflow.v1.schema.json',
  'mergeos.contributor.v1': '../schemas/contributor.v1.schema.json',
  'mergeos.customer-dashboard.v1': '../schemas/customer-dashboard.v1.schema.json',
  'mergeos.deployment.v1': '../schemas/deployment.v1.schema.json',
  'mergeos.dispute.v1': '../schemas/dispute.v1.schema.json',
  'mergeos.escrow.v1': '../schemas/escrow.v1.schema.json',
  'mergeos.estimate.v1': '../schemas/estimate.v1.schema.json',
  'mergeos.event.v1': '../schemas/event.v1.schema.json',
  'mergeos.ledger-proof.v1': '../schemas/ledger-proof.v1.schema.json',
  'mergeos.ledger.v1': '../schemas/ledger.v1.schema.json',
  'mergeos.live-feed.v1': '../schemas/live-feed.v1.schema.json',
  'mergeos.marketplace.v1': '../schemas/marketplace.v1.schema.json',
  'mergeos.payout-release.v1': '../schemas/payout-release.v1.schema.json',
  'mergeos.payouts.v1': '../schemas/payouts.v1.schema.json',
  'mergeos.pr-monitor.v1': '../schemas/pr-monitor.v1.schema.json',
  'mergeos.proposal.v1': '../schemas/proposal.v1.schema.json',
  'mergeos.release-artifact.v1': '../schemas/release-artifact.v1.schema.json',
  'mergeos.repo-import.v1': '../schemas/repo-import.v1.schema.json',
  'mergeos.repo-sync.v1': '../schemas/repo-sync.v1.schema.json',
  'mergeos.routing.v1': '../schemas/routing.v1.schema.json',
  'mergeos.scan.v1': '../schemas/scan.v1.schema.json',
  'mergeos.task-claim.v1': '../schemas/task-claim.v1.schema.json',
  'mergeos.task-review.v1': '../schemas/task-review.v1.schema.json',
  'mergeos.task-submission.v1': '../schemas/task-submission.v1.schema.json',
  'mergeos.task.v1': '../schemas/task.v1.schema.json',
  'mergeos.token-economy.v1': '../schemas/token-economy.v1.schema.json',
  'mergeos.wallet-migration.v1': '../schemas/wallet-migration.v1.schema.json',
  'mergeos.worker-dashboard.v1': '../schemas/worker-dashboard.v1.schema.json',
  'mergeos.workflow.v1': '../schemas/workflow.v1.schema.json',
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
  validateValue(document, schema, '', errors, schema);
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

export function contractReferenceFromLedger(entry, options = {}) {
  return formatReferenceHex(contractReferenceHex(entry), options);
}

export function contractReferenceBytes(entry) {
  return hexToBytes(contractReferenceHex(entry));
}

export function legacyWalletAddressHash(chain, address, options = {}) {
	const normalizedChain = normalizeLegacyChain(chain);
	const normalizedAddress = normalizeLegacyWalletAddress(address).toLowerCase();
	if (!normalizedAddress) {
		throw new Error('legacy wallet address is required');
	}
	return formatReferenceHex(sha256Hex(`mergeos:legacy-wallet:v1:${normalizedChain}:${normalizedAddress}`), options);
}

export function walletMigrationPDASeedMetadata(chain, address) {
  const legacyAddressHash = legacyWalletAddressHash(chain, address);
  return {
    pda_seeds: ['wallet-migration', normalizeLegacyChain(chain), 'legacy_address_hash_bytes'],
    pda_seed_formats: ['utf8', 'utf8', 'bytes32:hex_decode(contract.args.legacy_address_hash)'],
    legacy_address_hash: legacyAddressHash,
    legacy_address_hash_bytes: hexToBytes(legacyAddressHash),
  };
}

export function normalizeLegacyChain(value = '') {
	const normalized = String(value || '').trim().toLowerCase();
	if (normalized === 'trc20' || normalized === 'tron') return 'trc20';
	if (normalized === 'evm' || normalized === 'ethereum') return 'evm';
	throw new Error('legacy chain must be trc20 or evm');
}

export function normalizeLegacyWalletAddress(value = '') {
  let normalized = String(value || '').trim();
  for (const prefix of ['wallet:', 'tron:', 'trc20:', 'eip155:']) {
    if (normalized.toLowerCase().startsWith(prefix)) {
      normalized = normalized.slice(prefix.length).trim();
    }
  }
  if (/^0x[0-9a-f]{40}$/i.test(normalized)) return normalized.toLowerCase();
  return normalized;
}

function validateValue(value, schema, path, errors, rootSchema = schema) {
  if (schema.$ref) {
    const resolved = resolveSchemaRef(schema.$ref, rootSchema);
    if (!resolved) {
      errors.push({ path, message: `unsupported schema reference ${schema.$ref}` });
      return;
    }
    validateValue(value, resolved, path, errors, rootSchema);
    return;
  }

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
    validateObject(value, schema, path, errors, rootSchema);
    return;
  }

  if (schema.type === 'array') {
    validateArray(value, schema, path, errors, rootSchema);
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

function contractReferenceHex(entry) {
  if (entry === null || entry === undefined) {
    throw new Error('ledger entry is required');
  }
  if (typeof entry === 'string') {
    return contractReferenceFromValue(entry, 'value');
  }
  if (typeof entry !== 'object' || Array.isArray(entry)) {
    return sha256Hex(`mergeos:contract-reference:v1:value:${String(entry)}`);
  }

  for (const field of ['entry_hash', 'public_hash', 'hash']) {
    const value = String(entry[field] || '').trim();
    if (value) {
      return contractReferenceFromValue(value, field);
    }
  }
  if (String(entry.reference || '').trim()) {
    return sha256Hex(`mergeos:contract-reference:v1:reference:${String(entry.reference).trim()}`);
  }
  return sha256Hex(`mergeos:contract-reference:v1:ledger:${stableStringify(entry)}`);
}

function contractReferenceFromValue(value, field) {
  const normalized = String(value || '').trim();
  if (!normalized) {
    throw new Error('ledger reference value is required');
  }
  const hex = normalized.replace(/^0x/i, '');
  if (/^[0-9a-f]{64}$/i.test(hex)) {
    return hex.toLowerCase();
  }
  return sha256Hex(`mergeos:contract-reference:v1:${field}:${normalized}`);
}

function formatReferenceHex(hex, options = {}) {
  const format = String(options.format || 'hex').trim().toLowerCase();
  if (format === 'bytes' || format === 'array') return hexToBytes(hex);
  if (format === 'prefixed-hex' || format === '0x') return `0x${hex}`;
  return hex;
}

function hexToBytes(hex) {
  const normalized = String(hex || '').replace(/^0x/i, '').toLowerCase();
  if (!/^[0-9a-f]{64}$/.test(normalized)) {
    throw new Error('contract reference must be a 32-byte hex string');
  }
  return Array.from({ length: 32 }, (_, index) => Number.parseInt(normalized.slice(index * 2, index * 2 + 2), 16));
}

function sha256Hex(value) {
  return createHash('sha256').update(String(value)).digest('hex');
}

function stableStringify(value) {
  if (value === null || typeof value !== 'object') return JSON.stringify(value);
  if (Array.isArray(value)) return `[${value.map((item) => stableStringify(item)).join(',')}]`;
  return `{${Object.keys(value).sort().map((key) => `${JSON.stringify(key)}:${stableStringify(value[key])}`).join(',')}}`;
}

function resolveSchemaRef(ref, rootSchema) {
  if (!ref.startsWith('#/')) return null;
  return ref
    .slice(2)
    .split('/')
    .map((part) => part.replace(/~1/g, '/').replace(/~0/g, '~'))
    .reduce((current, part) => (current && typeof current === 'object' ? current[part] : undefined), rootSchema) || null;
}

function validateObject(value, schema, path, errors, rootSchema) {
  if (!value || typeof value !== 'object' || Array.isArray(value)) return;

  const fields = Object.keys(value);
  if (schema.minProperties !== undefined && fields.length < schema.minProperties) {
    errors.push({ path, message: `must contain at least ${schema.minProperties} propert${schema.minProperties === 1 ? 'y' : 'ies'}` });
  }
  if (schema.maxProperties !== undefined && fields.length > schema.maxProperties) {
    errors.push({ path, message: `must contain at most ${schema.maxProperties} propert${schema.maxProperties === 1 ? 'y' : 'ies'}` });
  }

  if (schema.propertyNames) {
    for (const field of fields) {
      validateValue(field, schema.propertyNames, joinPath(path, field), errors, rootSchema);
    }
  }

  for (const field of schema.required || []) {
    if (value[field] === undefined) {
      errors.push({ path: joinPath(path, field), message: 'is required' });
    }
  }

  const properties = schema.properties || {};
  if (schema.additionalProperties === false) {
    for (const field of fields) {
      if (!properties[field]) {
        errors.push({ path: joinPath(path, field), message: 'is not allowed' });
      }
    }
  } else if (schema.additionalProperties && typeof schema.additionalProperties === 'object') {
    for (const field of fields) {
      if (!properties[field]) {
        validateValue(value[field], schema.additionalProperties, joinPath(path, field), errors, rootSchema);
      }
    }
  }

  for (const [field, fieldSchema] of Object.entries(properties)) {
    if (value[field] !== undefined) {
      validateValue(value[field], fieldSchema, joinPath(path, field), errors, rootSchema);
    }
  }
}

function validateArray(value, schema, path, errors, rootSchema) {
  if (!Array.isArray(value)) return;
  if (schema.minItems !== undefined && value.length < schema.minItems) {
    errors.push({ path, message: `must contain at least ${schema.minItems} item(s)` });
  }
  if (schema.maxItems !== undefined && value.length > schema.maxItems) {
    errors.push({ path, message: `must contain at most ${schema.maxItems} item(s)` });
  }
  if (!schema.items) return;
  value.forEach((item, index) => validateValue(item, schema.items, `${path}[${index}]`, errors, rootSchema));
}

function validateString(value, schema, path, errors) {
  if (typeof value !== 'string') return;
  if (schema.minLength !== undefined && value.length < schema.minLength) {
    errors.push({ path, message: `must be at least ${schema.minLength} characters` });
  }
  if (schema.maxLength !== undefined && value.length > schema.maxLength) {
    errors.push({ path, message: `must be at most ${schema.maxLength} characters` });
  }
  if (schema.pattern !== undefined && !(new RegExp(schema.pattern).test(value))) {
    errors.push({ path, message: `must match pattern ${schema.pattern}` });
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
  const nodeIDs = new Set();
  for (const node of document.nodes || []) {
    if (node.id) nodeIDs.add(node.id);
    if (node.task_id) nodeIDs.add(node.task_id);
  }
  for (const [index, edge] of (document.edges || []).entries()) {
    if (!nodeIDs.has(edge.from)) {
      errors.push({ path: `edges[${index}].from`, message: 'must reference an existing node id' });
    }
    if (!nodeIDs.has(edge.to)) {
      errors.push({ path: `edges[${index}].to`, message: 'must reference an existing node id' });
    }
  }
  for (const [nodeIndex, node] of (document.nodes || []).entries()) {
    for (const [dependencyIndex, dependency] of (node.dependencies || []).entries()) {
      if (!nodeIDs.has(dependency)) {
        errors.push({ path: `nodes[${nodeIndex}].dependencies[${dependencyIndex}]`, message: 'must reference an existing node id' });
      }
    }
  }
  for (const [actionIndex, action] of (document.next_actions || []).entries()) {
    if (action.target_node_id && !nodeIDs.has(action.target_node_id)) {
      errors.push({ path: `next_actions[${actionIndex}].target_node_id`, message: 'must reference an existing node id' });
    }
    if (action.task_id && !nodeIDs.has(action.task_id)) {
      errors.push({ path: `next_actions[${actionIndex}].task_id`, message: 'must reference an existing node id' });
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
