import { describe, it } from "node:test";
import assert from "node:assert/strict";
import { existsSync, readFileSync, readdirSync } from "node:fs";
import { join } from "node:path";
import { fileURLToPath } from "node:url";

const root = fileURLToPath(new URL("..", import.meta.url));
const programPath = join(root, "solana", "programs", "mergeos-mrg", "src", "lib.rs");
const programSource = readFileSync(programPath, "utf8");
const readmeSource = readFileSync(join(root, "README.md"), "utf8");
const solanaReadmeSource = readFileSync(join(root, "solana", "README.md"), "utf8");
const idl = JSON.parse(readFileSync(join(root, "solana", "idl", "mergeos_mrg.json"), "utf8"));
const publicIDL = JSON.parse(readFileSync(join(root, "..", "frontend", "public", "contracts", "solana", "mergeos_mrg.v1.idl.json"), "utf8"));
const backendConfigSource = readFileSync(join(root, "..", "backend", "internal", "core", "config.go"), "utf8");

describe("contract package", () => {
  it("uses the active Solana MRG Anchor workspace instead of Solidity sources", () => {
    assert.equal(existsSync(join(root, "solana", "Anchor.toml")), true);
    assert.equal(existsSync(join(root, "solana", "Cargo.toml")), true);
    const srcFiles = existsSync(join(root, "src")) ? readdirSync(join(root, "src")) : [];
    assert.deepEqual(srcFiles.filter((file) => file.endsWith(".sol")), []);
    assert.match(programSource, /use anchor_lang::prelude::\*/);
    assert.match(programSource, /use anchor_spl::token::\{/);
    assert.match(programSource, /#\[program\]\s*pub mod mergeos_mrg/);
    assert.doesNotMatch(programSource, /So11111111111111111111111111111111111111112/);
    assert.match(programSource, /declare_id!\("4gUBWum3fGKfm7BeGXryzXjPDBDLfhVJRcjN5MPnfDNW"\)/);
    assert.equal(idl.address, "4gUBWum3fGKfm7BeGXryzXjPDBDLfhVJRcjN5MPnfDNW");
    assert.deepEqual(publicIDL, idl);
  });

  it("defines the MRG mint, escrow, payout, and migration instructions", () => {
    for (const instruction of [
      "initialize_treasury",
      "mint_verified_mrg",
      "open_escrow",
      "release_payout",
      "register_legacy_wallet",
    ]) {
      assert.match(programSource, new RegExp(`pub fn ${instruction}\\(`));
    }
    assert.deepEqual(idl.instructions.map((item) => item.name), [
      "initializeTreasury",
      "mintVerifiedMrg",
      "openEscrow",
      "releasePayout",
      "registerLegacyWallet",
    ]);
    assert.match(programSource, /pub enum LegacyChain\s*\{[\s\S]*?Trc20,[\s\S]*?Evm,/);
    assert.match(programSource, /pub struct WalletMigration/);
    assert.match(programSource, /b"wallet-migration"[\s\S]*legacy_chain\.seed\(\)\.as_bytes\(\)[\s\S]*legacy_address_hash\.as_ref\(\)/);
  });

  it("keeps ledger reconciliation anchors on every public money event", () => {
    for (const eventName of [
      "MRGMinted",
      "EscrowOpened",
      "PayoutReleased",
      "LegacyWalletRegistered",
    ]) {
      assert.match(programSource, new RegExp(`pub struct ${eventName}`));
    }
    const referenceEvents = [
      "MRGMinted",
      "EscrowOpened",
      "PayoutReleased",
    ];
    for (const eventName of referenceEvents) {
      const eventBlock = programSource.match(new RegExp(`pub struct ${eventName} \\{([\\s\\S]*?)\\}`))?.[1] || "";
      assert.match(eventBlock, /ledger_reference: \[u8; 32\]/, `${eventName} should carry a ledger reference`);
    }
  });

  it("documents the deployment env var that backend actually reads", () => {
    assert.match(backendConfigSource, /os\.Getenv\("MRG_SOLANA_PROGRAM_ID"\)/);
    assert.match(readmeSource, /MRG_SOLANA_PROGRAM_ID/);
    assert.match(solanaReadmeSource, /MRG_SOLANA_PROGRAM_ID/);
    assert.doesNotMatch(solanaReadmeSource, /\bSOLANA_PROGRAM_ID\b/);
  });

  it("publishes token account security invariants in the public IDL", () => {
    assert.ok(idl.metadata.security_invariants.length >= 6);
    for (const invariant of [
      "mint_verified_mrg requires receiver_token_account.mint == treasury_config.token_mint",
      "open_escrow requires escrow_token_account.mint == token_mint and escrow_token_account.owner == escrow_vault PDA",
      "release_payout requires worker_token_account.mint == treasury_config.token_mint",
    ]) {
      assert.ok(idl.metadata.security_invariants.includes(invariant), `missing invariant: ${invariant}`);
    }
  });

  it("documents SDK and protocol helpers for off-chain reference derivation", () => {
    assert.match(readmeSource, /contractReferenceFromLedger/);
    assert.match(readmeSource, /legacyWalletAddressHash/);
    assert.match(readmeSource, /ledger_reference: \[u8; 32\]/);
    assert.match(readmeSource, /legacy_address_hash: \[u8; 32\]/);
  });

  it("avoids EVM and unsafe Solana primitives", () => {
    for (const banned of [
      /\bpragma solidity\b/,
      /\bcontract\s+\w+/,
      /\btx\.origin\b/,
      /\bdelegatecall\b/,
      /\bselfdestruct\b/,
      /\binvoke_unchecked\b/,
    ]) {
      assert.doesNotMatch(programSource, banned);
    }
    assert.match(programSource, /token::transfer/);
    assert.match(programSource, /token::mint_to/);
    assert.match(programSource, /TokenMintMismatch/);
    assert.match(programSource, /TokenAuthorityMismatch/);
  });
});
