import test from 'node:test';
import assert from 'node:assert/strict';
import {
  aggregateAccounts,
  accountRole,
  buildExplorerStats,
  filterEntries,
  findExplorerTarget,
  githubProfileURL,
  githubUsernameFromAccount,
  normalizeLedgerAccount,
  normalizeExplorerPath,
  parseExplorerRoute,
  parseLedgerReference,
  paymentModeLabel,
  tokenAmountFromCents,
  verifyLedgerChain,
} from './explorer.js';

const entries = [
  {
    sequence: 1,
    type: 'payment_verified',
    from_account: 'payment:paypal',
    to_account: 'project:prj_0001',
    amount_cents: 100000,
    reference: 'project:prj_0001',
    previous_hash: '0'.repeat(64),
    entry_hash: 'a'.repeat(64),
    created_at: '2026-05-26T00:00:00Z',
  },
  {
    sequence: 2,
    type: 'token_mint',
    from_account: 'issuer:mergeos',
    to_account: 'project:prj_0001',
    amount_cents: 100000,
    reference: 'mint:prj_0001',
    previous_hash: 'a'.repeat(64),
    entry_hash: 'b'.repeat(64),
    created_at: '2026-05-26T00:01:00Z',
  },
];

test('converts funded cents to MRG token amount', () => {
  assert.equal(tokenAmountFromCents(100000), 100000);
});

test('verifies the public ledger hash chain', () => {
  assert.equal(verifyLedgerChain(entries).ok, true);
  assert.equal(verifyLedgerChain([{ ...entries[1], previous_hash: 'bad' }]).ok, false);
});

test('aggregates account activity for address pages', () => {
  const accounts = aggregateAccounts(entries);
  const project = accounts.find((row) => row.account === 'project:prj_0001');

  assert.equal(project.received_cents, 200000);
  assert.equal(project.tx_count, 2);
});

test('finds transactions, blocks and addresses from one search box', () => {
  const accounts = aggregateAccounts(entries);
  accounts.push({
    account: '0x1234567890abcdef1234567890abcdef12345678',
    tx_count: 1,
  });

  assert.equal(findExplorerTarget(entries, accounts, 'bbbbbbbb').kind, 'tx');
  assert.equal(findExplorerTarget(entries, accounts, '#2').kind, 'block');
  assert.equal(findExplorerTarget(entries, accounts, 'project:prj_0001').kind, 'address');
  assert.equal(findExplorerTarget(entries, accounts, '0x1234567890abcdef1234567890abcdef12345678').kind, 'address');
  assert.equal(findExplorerTarget(entries, accounts, 'wallet:0x1234567890abcdef1234567890abcdef12345678').kind, 'address');
});

test('treats github aliases as short public addresses', () => {
  assert.equal(githubUsernameFromAccount('github:hummusonrails'), 'hummusonrails');
  assert.equal(githubUsernameFromAccount('worker:github:hummusonrails'), 'hummusonrails');
  assert.equal(githubProfileURL('github:hummusonrails'), 'https://github.com/hummusonrails');
  assert.equal(accountRole('github:hummusonrails'), 'GitHub Contributor');
});

test('shows production-friendly payment mode labels', () => {
  assert.equal(paymentModeLabel('local-dev-verifier'), 'MergeOS verifier');
  assert.equal(paymentModeLabel('live-adapters'), 'Live payment adapters');
  assert.equal(paymentModeLabel(''), 'Not configured');
});

test('normalizes legacy wallet account labels to raw addresses', () => {
  assert.equal(normalizeLedgerAccount('wallet:0x1234567890abcdef1234567890abcdef12345678'), '0x1234567890abcdef1234567890abcdef12345678');
  assert.equal(accountRole('0x1234567890abcdef1234567890abcdef12345678'), 'MRG Wallet');
  assert.equal(normalizeLedgerAccount('reserve:task:tsk_0010'), 'reserve:task');
});

test('parses history routes and legacy hash routes', () => {
  assert.deepEqual(parseExplorerRoute('/address/wallet%3A0x123', ''), { name: 'address', value: 'wallet:0x123' });
  assert.deepEqual(parseExplorerRoute('/', '#/tx/bbbbbbbb'), { name: 'tx', value: 'bbbbbbbb' });
  assert.deepEqual(parseExplorerRoute('/block/2', ''), { name: 'block', value: '2' });
  assert.deepEqual(parseExplorerRoute('/unknown', ''), { name: 'home', value: '' });
  assert.equal(normalizeExplorerPath('address/0x123'), '/address/0x123');
});

test('filters entries by type, account and free text', () => {
  assert.equal(filterEntries(entries, { type: 'token_mint' }).length, 1);
  assert.equal(filterEntries(entries, { account: 'project:prj_0001' }).length, 2);
  assert.equal(filterEntries(entries, { query: 'paypal' }).length, 1);
});

test('builds explorer-level stats from ledger and marketplace payloads', () => {
  const stats = buildExplorerStats([
    ...entries,
    {
      sequence: 3,
      type: 'manual_credit',
      from_account: 'reserve:task',
      to_account: 'github:eliasx45',
      amount_cents: 50,
      reference: 'pr:https://github.com/mergeos-bounties/mergeos/pull/120;title:Public timeline correction',
      previous_hash: 'b'.repeat(64),
      entry_hash: 'c'.repeat(64),
      created_at: '2026-05-26T00:02:00Z',
    },
  ], { stats: { project_count: 3 }, projects: [] }, 'MRG');

  assert.equal(stats.totalTransactions, 3);
  assert.equal(stats.mintedTokens, 100000);
  assert.equal(stats.payoutCents, 50);
  assert.equal(stats.projectCount, 3);
  assert.equal(stats.chainOk, true);
});

test('parses pull request ledger references into safe link metadata', () => {
  const reference = parseLedgerReference('pr:https://github.com/mergeos-bounties/mergeos/pull/132;title:fix: auth modal responsive for extra-small screens (<450px) (#13)');

  assert.equal(reference.kind, 'github_pr');
  assert.equal(reference.href, 'https://github.com/mergeos-bounties/mergeos/pull/132');
  assert.equal(reference.title, 'fix: auth modal responsive for extra-small screens (<450px) (#13)');
  assert.equal(reference.shortLabel, 'PR #132');
  assert.equal(reference.provider, 'mergeos-bounties/mergeos');
});
