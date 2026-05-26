import test from 'node:test';
import assert from 'node:assert/strict';
import {
  aggregateAccounts,
  buildExplorerStats,
  filterEntries,
  findExplorerTarget,
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

  assert.equal(findExplorerTarget(entries, accounts, 'bbbbbbbb').kind, 'tx');
  assert.equal(findExplorerTarget(entries, accounts, '#2').kind, 'block');
  assert.equal(findExplorerTarget(entries, accounts, 'project:prj_0001').kind, 'address');
});

test('filters entries by type, account and free text', () => {
  assert.equal(filterEntries(entries, { type: 'token_mint' }).length, 1);
  assert.equal(filterEntries(entries, { account: 'project:prj_0001' }).length, 2);
  assert.equal(filterEntries(entries, { query: 'paypal' }).length, 1);
});

test('builds explorer-level stats from ledger and marketplace payloads', () => {
  const stats = buildExplorerStats(entries, { stats: { project_count: 3 }, projects: [] }, 'MRG');

  assert.equal(stats.totalTransactions, 2);
  assert.equal(stats.mintedTokens, 100000);
  assert.equal(stats.projectCount, 3);
  assert.equal(stats.chainOk, true);
});
