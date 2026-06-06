import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import test from 'node:test';

const sourceIDL = JSON.parse(readFileSync(new URL('../idl/mergeos_mrg.json', import.meta.url), 'utf8'));
const publicIDL = JSON.parse(readFileSync(new URL('../../../frontend/public/contracts/solana/mergeos_mrg.v1.idl.json', import.meta.url), 'utf8'));
const programSource = readFileSync(new URL('../programs/mergeos-mrg/src/lib.rs', import.meta.url), 'utf8');

function instruction(name) {
  const item = sourceIDL.instructions.find((row) => row.name === name);
  assert.ok(item, `missing instruction ${name}`);
  return item;
}

function argNames(item) {
  return item.args.map((arg) => arg.name);
}

function accountNames(item) {
  return item.accounts.map((account) => account.name);
}

test('publishes the same Solana MRG IDL to frontend static assets', () => {
  assert.deepEqual(publicIDL, sourceIDL);
  assert.equal(sourceIDL.metadata.public_idl_url, 'https://mergeos.shop/contracts/solana/mergeos_mrg.v1.idl.json');
});

test('declares MRG treasury, escrow, payout, and wallet migration instructions', () => {
  assert.deepEqual(
    sourceIDL.instructions.map((item) => item.name),
    ['initializeTreasury', 'mintVerifiedMrg', 'openEscrow', 'releasePayout', 'registerLegacyWallet'],
  );

  assert.deepEqual(argNames(instruction('mintVerifiedMrg')), ['ledgerReference', 'amount']);
  assert.deepEqual(argNames(instruction('openEscrow')), ['projectId', 'ledgerReference', 'amount']);
  assert.deepEqual(accountNames(instruction('openEscrow')), [
    'funder',
    'treasuryConfig',
    'tokenMint',
    'funderTokenAccount',
    'escrowVault',
    'escrowTokenAccount',
    'tokenProgram',
    'systemProgram',
  ]);
  assert.deepEqual(argNames(instruction('releasePayout')), ['payoutId', 'ledgerReference', 'amount']);
});

test('guards MRG mint, escrow PDA authority, and payout token accounts', () => {
  assert.deepEqual(sourceIDL.metadata.security_invariants, [
    'mint_verified_mrg requires receiver_token_account.mint == treasury_config.token_mint',
    'open_escrow requires token_mint == treasury_config.token_mint',
    'open_escrow requires funder_token_account.mint == token_mint and funder_token_account.owner == funder',
    'open_escrow requires escrow_token_account.mint == token_mint and escrow_token_account.owner == escrow_vault PDA',
    'release_payout requires escrow_token_account.mint == treasury_config.token_mint and escrow_token_account.owner == escrow_vault PDA',
    'release_payout requires worker_token_account.mint == treasury_config.token_mint',
  ]);

  for (const snippet of [
    'constraint = receiver_token_account.mint == token_mint.key() @ MergeOSError::TokenMintMismatch',
    '#[account(address = treasury_config.token_mint)]',
    'constraint = funder_token_account.mint == token_mint.key() @ MergeOSError::TokenMintMismatch',
    'constraint = funder_token_account.owner == funder.key() @ MergeOSError::TokenAuthorityMismatch',
    'constraint = escrow_token_account.mint == token_mint.key() @ MergeOSError::TokenMintMismatch',
    'constraint = escrow_token_account.owner == escrow_vault.key() @ MergeOSError::TokenAuthorityMismatch',
    'constraint = escrow_token_account.mint == treasury_config.token_mint @ MergeOSError::TokenMintMismatch',
    'constraint = worker_token_account.mint == treasury_config.token_mint @ MergeOSError::TokenMintMismatch',
    'Token account mint does not match the MRG mint',
    'Token account authority does not match the expected PDA or signer',
  ]) {
    assert.ok(programSource.includes(snippet), `program source missing token guard: ${snippet}`);
  }
});

test('keeps wallet migration PDA metadata aligned with protocol helpers', () => {
  assert.deepEqual(sourceIDL.metadata.wallet_migration_pda_seeds, [
    'wallet-migration',
    'legacy_chain',
    'legacy_address_hash_bytes',
  ]);
  assert.deepEqual(sourceIDL.metadata.wallet_migration_pda_seed_formats, [
    'utf8',
    'utf8',
    'bytes32:hex_decode(contract.args.legacy_address_hash)',
  ]);

  const register = instruction('registerLegacyWallet');
  assert.deepEqual(argNames(register), ['legacyChain', 'legacyAddressHash', 'solanaWallet']);
  assert.deepEqual(accountNames(register), ['owner', 'solanaWallet', 'walletMigration', 'systemProgram']);

  const walletMigration = register.accounts.find((account) => account.name === 'walletMigration');
  assert.deepEqual(walletMigration.pda.seeds, [
    { kind: 'const', value: 'wallet-migration' },
    { kind: 'arg', path: 'legacyChain', format: 'utf8:seed()' },
    { kind: 'arg', path: 'legacyAddressHash', format: 'bytes32' },
  ]);
});

test('documents chain enum mapping and bytes32 ledger references', () => {
  assert.deepEqual(sourceIDL.metadata.legacy_chain_wire, {
    trc20: 'LegacyChain::Trc20',
    evm: 'LegacyChain::Evm',
  });
  assert.equal(sourceIDL.metadata.ledger_reference_format, 'bytes32:hex_decode(entry_hash|public_hash|contract_reference)');

  for (const item of ['mintVerifiedMrg', 'openEscrow', 'releasePayout']) {
    const hasBytes32LedgerReference = instruction(item).args.some(
      (arg) => arg.name === 'ledgerReference'
        && Array.isArray(arg.type?.array)
        && arg.type.array[0] === 'u8'
        && arg.type.array[1] === 32,
    );
    assert.equal(hasBytes32LedgerReference, true, `${item} must anchor a bytes32 ledger reference`);
  }
});
