export const TOKEN_RATE_PER_USD = 100;

export function normalizeEntry(entry = {}) {
  return {
    sequence: Number(entry.sequence) || 0,
    type: String(entry.type || 'unknown'),
    from_account: normalizeLedgerAccount(entry.from_account),
    to_account: normalizeLedgerAccount(entry.to_account),
    amount_cents: Number(entry.amount_cents) || 0,
    reference: String(entry.reference || ''),
    previous_hash: String(entry.previous_hash || ''),
    entry_hash: String(entry.entry_hash || ''),
    created_at: String(entry.created_at || ''),
  };
}

export function normalizeLedgerAccount(account = '') {
  const value = String(account || '').trim();
  const normalized = value.toLowerCase();
  if (/^wallet:0x[0-9a-f]{40}$/.test(normalized)) return normalized.slice('wallet:'.length);
  if (/^0x[0-9a-f]{40}$/.test(normalized)) return normalized;
  if (normalized.startsWith('reserve:task:')) return 'reserve:task';
  return value;
}

export function sortLedgerEntries(entries = []) {
  return entries.map(normalizeEntry).sort((a, b) => a.sequence - b.sequence);
}

export function tokenAmountFromCents(cents = 0) {
  return Math.round(((Number(cents) || 0) / 100) * TOKEN_RATE_PER_USD);
}

export function formatMoneyFromCents(cents = 0) {
  return new Intl.NumberFormat('en-US', {
    style: 'currency',
    currency: 'USD',
    maximumFractionDigits: 0,
  }).format((Number(cents) || 0) / 100);
}

export function formatCompactNumber(value = 0) {
  return new Intl.NumberFormat('en-US', {
    maximumFractionDigits: Math.abs(Number(value) || 0) >= 100 ? 0 : 1,
  }).format(Number(value) || 0);
}

export function shortHash(value = '', head = 10, tail = 8) {
  const text = String(value || '');
  if (text.length <= head + tail + 3) return text || '-';
  return `${text.slice(0, head)}...${text.slice(-tail)}`;
}

export function toTitleLabel(value = '') {
  return String(value || '')
    .trim()
    .split(/[\s._:-]+/)
    .filter(Boolean)
    .map((word) => {
      const lower = word.toLowerCase();
      if (['ai', 'api', 'qa', 'ui', 'ux', 'go', 'mrg'].includes(lower)) return lower.toUpperCase();
      if (lower === 'devops') return 'DevOps';
      return `${lower.charAt(0).toUpperCase()}${lower.slice(1)}`;
    })
    .join(' ');
}

export function paymentModeLabel(value = '') {
  const normalized = String(value || '').trim().toLowerCase();
  return {
    'live-adapters': 'Live payment adapters',
    'local-dev-verifier': 'MergeOS verifier',
    'not-configured': 'Not configured',
    '': 'Not configured',
  }[normalized] || toTitleLabel(value);
}

export function parseLedgerReference(reference = '') {
  const raw = String(reference || '').trim();
  const fallback = {
    raw,
    kind: raw ? 'text' : 'empty',
    href: '',
    title: '',
    label: raw,
    shortLabel: raw,
    provider: '',
    number: '',
  };
  if (!raw) return fallback;

  const fields = raw.split(';').reduce((acc, part) => {
    const index = part.indexOf(':');
    if (index <= 0) return acc;
    const key = part.slice(0, index).trim().toLowerCase();
    const value = part.slice(index + 1).trim();
    if (key && value) acc[key] = value;
    return acc;
  }, {});

  const url = safeReferenceURL(fields.pr || fields.url || fields.link);
  if (!url) return fallback;

  const pull = githubPullRequestInfo(url);
  const title = fields.title || '';
  const provider = pull ? `${pull.owner}/${pull.repo}` : url.hostname.replace(/^www\./, '');
  const shortLabel = pull ? `PR #${pull.number}` : provider;

  return {
    raw,
    kind: pull ? 'github_pr' : 'link',
    href: url.href,
    title,
    label: title || shortLabel,
    shortLabel,
    provider,
    number: pull?.number || '',
  };
}

function safeReferenceURL(value = '') {
  try {
    const url = new URL(String(value || '').trim());
    return ['http:', 'https:'].includes(url.protocol) ? url : null;
  } catch {
    return null;
  }
}

function githubPullRequestInfo(url) {
  const host = url.hostname.toLowerCase().replace(/^www\./, '');
  const parts = url.pathname.split('/').filter(Boolean);
  if (host !== 'github.com' || parts.length < 4 || parts[2] !== 'pull' || !/^\d+$/.test(parts[3])) return null;
  return {
    owner: parts[0],
    repo: parts[1],
    number: parts[3],
  };
}

export function ledgerTypeMeta(type = '') {
  const normalized = String(type || '').toLowerCase();
  const fallback = { label: toTitleLabel(normalized || 'Ledger Entry'), tone: 'neutral', direction: 'neutral' };
  return {
    token_mint: { label: 'Token Mint', tone: 'mint', direction: 'in' },
    payment_verified: { label: 'Payment Verified', tone: 'success', direction: 'in' },
    platform_fee: { label: 'Platform Fee', tone: 'fee', direction: 'out' },
    project_reserve: { label: 'Project Reserve', tone: 'reserve', direction: 'neutral' },
    task_reserve: { label: 'Task Reserve', tone: 'task', direction: 'neutral' },
    task_payment: { label: 'Task Payout', tone: 'payout', direction: 'out' },
    manual_credit: { label: 'Manual Credit', tone: 'payout', direction: 'out' },
  }[normalized] || fallback;
}

export function formatLedgerDate(value = '') {
  const date = value ? new Date(value) : null;
  if (!date || Number.isNaN(date.getTime())) {
    return { date: '-', time: '-', full: '-' };
  }
  return {
    date: date.toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric', timeZone: 'UTC' }),
    time: date.toLocaleTimeString('en-US', { hour: '2-digit', minute: '2-digit', second: '2-digit', timeZone: 'UTC' }),
    full: `${date.toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric', timeZone: 'UTC' })} ${date.toLocaleTimeString('en-US', { hour: '2-digit', minute: '2-digit', second: '2-digit', timeZone: 'UTC' })} UTC`,
  };
}

export function entrySearchText(entry = {}) {
  return [
    entry.sequence,
    entry.type,
    entry.from_account,
    entry.to_account,
    entry.amount_cents,
    entry.reference,
    entry.previous_hash,
    entry.entry_hash,
    entry.created_at,
  ].join(' ').toLowerCase();
}

export function filterEntries(entries = [], { query = '', type = 'all', account = '' } = {}) {
  const normalizedQuery = String(query || '').trim().toLowerCase();
  const normalizedType = String(type || 'all').toLowerCase();
  const normalizedAccount = normalizeLedgerAccount(account).toLowerCase();
  return entries.filter((entry) => {
    if (normalizedType !== 'all' && entry.type.toLowerCase() !== normalizedType) return false;
    if (normalizedAccount) {
      const from = normalizeLedgerAccount(entry.from_account).toLowerCase();
      const to = normalizeLedgerAccount(entry.to_account).toLowerCase();
      if (from !== normalizedAccount && to !== normalizedAccount) return false;
    }
    if (!normalizedQuery) return true;
    return entrySearchText(entry).includes(normalizedQuery);
  });
}

export function verifyLedgerChain(entries = []) {
  const sorted = sortLedgerEntries(entries);
  const issues = [];
  for (let index = 0; index < sorted.length; index += 1) {
    const entry = sorted[index];
    const expectedSequence = index + 1;
    if (entry.sequence !== expectedSequence) {
      issues.push({ sequence: entry.sequence, issue: `expected sequence ${expectedSequence}` });
    }
    if (index === 0) {
      const validGenesis = !entry.previous_hash || /^0{64}$/.test(entry.previous_hash);
      if (!validGenesis) {
        issues.push({ sequence: entry.sequence, issue: 'invalid genesis previous hash' });
      }
      continue;
    }
    const previous = sorted[index - 1];
    if (entry.previous_hash !== previous.entry_hash) {
      issues.push({ sequence: entry.sequence, issue: 'previous hash mismatch' });
    }
  }
  return {
    ok: issues.length === 0,
    issues,
    height: sorted.length ? sorted[sorted.length - 1].sequence : 0,
    latestHash: sorted.length ? sorted[sorted.length - 1].entry_hash : '',
  };
}

export function aggregateAccounts(entries = []) {
  const accounts = new Map();
  const touch = (name) => {
    const account = String(name || '').trim();
    if (!account) return null;
    if (!accounts.has(account)) {
      accounts.set(account, {
        account,
        sent_cents: 0,
        received_cents: 0,
        sent_count: 0,
        received_count: 0,
        first_seen_at: '',
        last_seen_at: '',
      });
    }
    return accounts.get(account);
  };

  for (const rawEntry of entries) {
    const entry = normalizeEntry(rawEntry);
    const from = touch(entry.from_account);
    const to = touch(entry.to_account);
    if (from) {
      from.sent_cents += entry.amount_cents;
      from.sent_count += 1;
      updateAccountWindow(from, entry.created_at);
    }
    if (to) {
      to.received_cents += entry.amount_cents;
      to.received_count += 1;
      updateAccountWindow(to, entry.created_at);
    }
  }

  return Array.from(accounts.values())
    .map((account) => ({
      ...account,
      tx_count: account.sent_count + account.received_count,
      net_cents: account.received_cents - account.sent_cents,
    }))
    .sort((a, b) => b.tx_count - a.tx_count || Math.abs(b.net_cents) - Math.abs(a.net_cents));
}

function updateAccountWindow(account, timestamp) {
  if (!timestamp) return;
  if (!account.first_seen_at || timestamp < account.first_seen_at) account.first_seen_at = timestamp;
  if (!account.last_seen_at || timestamp > account.last_seen_at) account.last_seen_at = timestamp;
}

export function findExplorerTarget(entries = [], accounts = [], rawQuery = '') {
  const query = String(rawQuery || '').trim();
  const normalized = query.toLowerCase();
  if (!normalized) return null;
  const accountQuery = normalizeLedgerAccount(normalized).toLowerCase();

  const exactHash = entries.find((entry) => entry.entry_hash.toLowerCase() === normalized);
  if (exactHash) return { kind: 'tx', value: exactHash.entry_hash, entry: exactHash };

  const hashPrefix = entries.find((entry) => entry.entry_hash.toLowerCase().startsWith(normalized));
  if (hashPrefix && normalized.length >= 8) return { kind: 'tx', value: hashPrefix.entry_hash, entry: hashPrefix };

  if (/^#?\d+$/.test(normalized)) {
    const sequence = Number(normalized.replace('#', ''));
    const blockEntry = entries.find((entry) => entry.sequence === sequence);
    if (blockEntry) return { kind: 'block', value: String(sequence), entry: blockEntry };
  }

  const exactAccount = accounts.find((row) => normalizeLedgerAccount(row.account).toLowerCase() === accountQuery);
  if (exactAccount) return { kind: 'address', value: exactAccount.account, account: exactAccount };

  const accountPrefix = accounts.find((row) => normalizeLedgerAccount(row.account).toLowerCase().startsWith(accountQuery));
  if (accountPrefix && normalized.length >= 4) return { kind: 'address', value: accountPrefix.account, account: accountPrefix };

  const referenceMatch = entries.find((entry) => entry.reference.toLowerCase() === normalized);
  if (referenceMatch) return { kind: 'tx', value: referenceMatch.entry_hash, entry: referenceMatch };

  return null;
}

export function githubUsernameFromAccount(account = '') {
  const value = String(account || '').trim();
  const normalized = value.toLowerCase();
  if (normalized.startsWith('github:')) return value.slice('github:'.length).trim().replace(/^\/+|\/+$/g, '');
  if (normalized.startsWith('worker:github:')) return value.slice('worker:github:'.length).trim().replace(/^\/+|\/+$/g, '');
  return '';
}

export function githubProfileURL(account = '') {
  const username = githubUsernameFromAccount(account);
  return username ? `https://github.com/${encodeURIComponent(username)}` : '';
}

export function parseExplorerRoute(pathname = '/', hash = '') {
  const legacyHashPath = String(hash || '').replace(/^#/, '');
  const routePath = legacyHashPath.startsWith('/') ? legacyHashPath : String(pathname || '/');
  const parts = routePath.split('?')[0].split('/').filter(Boolean);
  if (parts[0] === 'tx' && parts[1]) return { name: 'tx', value: decodeURIComponent(parts[1]) };
  if (parts[0] === 'address' && parts[1]) return { name: 'address', value: decodeURIComponent(parts.slice(1).join('/')) };
  if (parts[0] === 'block' && parts[1]) return { name: 'block', value: decodeURIComponent(parts[1]) };
  return { name: 'home', value: '' };
}

export function normalizeExplorerPath(path = '/') {
  const value = String(path || '/').trim();
  if (!value || value === '/') return '/';
  return value.startsWith('/') ? value : `/${value}`;
}

export function accountRole(account = '') {
  const value = String(account || '').toLowerCase();
  if (/^(wallet:)?0x[0-9a-f]{40}$/.test(value)) return 'MRG Wallet';
  if (value.startsWith('issuer:')) return 'Issuer';
  if (value.startsWith('treasury:')) return 'Treasury';
  if (value.startsWith('payment:')) return 'Payment Adapter';
  if (value.startsWith('reserve:task')) return 'Task Reserve';
  if (value.startsWith('reserve:project')) return 'Project Reserve';
  if (value.startsWith('project:')) return 'Project Account';
  if (githubUsernameFromAccount(value)) return 'GitHub Contributor';
  if (value.startsWith('worker:')) return 'Contributor';
  if (value.startsWith('client:')) return 'Client';
  return 'Ledger Account';
}

export function buildExplorerStats(entries = [], marketplace = {}, tokenSymbol = 'MRG') {
  const normalizedEntries = entries.map(normalizeEntry);
  const mintedCents = normalizedEntries
    .filter((entry) => entry.type === 'token_mint')
    .reduce((total, entry) => total + entry.amount_cents, 0);
  const fundingCents = normalizedEntries
    .filter((entry) => entry.type === 'payment_verified')
    .reduce((total, entry) => total + entry.amount_cents, 0);
  const payoutCents = normalizedEntries
    .filter((entry) => entry.type === 'task_payment' || entry.type === 'manual_credit')
    .reduce((total, entry) => total + entry.amount_cents, 0);
  const accounts = aggregateAccounts(normalizedEntries);
  const chain = verifyLedgerChain(normalizedEntries);
  const projects = Array.isArray(marketplace.projects) ? marketplace.projects : [];
  const openTasks = projects.reduce((total, project) => total + (Number(project.open_task_count) || 0), 0);

  return {
    totalTransactions: normalizedEntries.length,
    chainHeight: chain.height,
    chainOk: chain.ok,
    latestHash: chain.latestHash,
    uniqueAccounts: accounts.length,
    mintedTokens: tokenAmountFromCents(mintedCents),
    fundingCents,
    payoutCents,
    projectCount: Number(marketplace.stats?.project_count) || projects.length,
    openTasks,
    tokenSymbol,
  };
}
