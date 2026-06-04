import { describe, it } from "node:test";
import assert from "node:assert/strict";
import { existsSync, readFileSync, readdirSync } from "node:fs";
import { join } from "node:path";
import { fileURLToPath } from "node:url";

const root = fileURLToPath(new URL("..", import.meta.url));
const programPath = join(root, "programs", "mergeos", "src", "lib.rs");
const programSource = readFileSync(programPath, "utf8");

describe("contract package", () => {
  it("uses a Solana Anchor workspace instead of Solidity sources", () => {
    assert.equal(existsSync(join(root, "Anchor.toml")), true);
    assert.equal(existsSync(join(root, "Cargo.toml")), true);
    const srcFiles = existsSync(join(root, "src")) ? readdirSync(join(root, "src")) : [];
    assert.deepEqual(srcFiles.filter((file) => file.endsWith(".sol")), []);
    assert.match(programSource, /use anchor_lang::prelude::\*/);
    assert.match(programSource, /use anchor_spl::token::\{/);
    assert.match(programSource, /#\[program\]\s*pub mod mergeos/);
  });

  it("defines the MRG mint, escrow, payout, and migration instructions", () => {
    for (const instruction of [
      "register_legacy_wallet",
      "mint_mrg",
      "burn_mrg",
      "fund_project",
      "reserve_task",
      "release_task",
      "refund_task",
      "close_project",
    ]) {
      assert.match(programSource, new RegExp(`pub fn ${instruction}\\(`));
    }
    assert.match(programSource, /pub enum LegacyChain\s*\{[\s\S]*?Trc20,[\s\S]*?Evm,/);
    assert.match(programSource, /pub struct WalletMigration/);
  });

  it("keeps ledger reconciliation anchors on every public money event", () => {
    for (const eventName of [
      "LegacyWalletMigrated",
      "MrgMinted",
      "MrgBurned",
      "ProjectFunded",
      "TaskReserved",
      "TaskPaid",
      "TaskRefunded",
      "ProjectClosed",
    ]) {
      assert.match(programSource, new RegExp(`pub struct ${eventName}`));
    }
    const referenceEvents = [
      "MrgMinted",
      "MrgBurned",
      "ProjectFunded",
      "TaskReserved",
      "TaskPaid",
      "TaskRefunded",
      "ProjectClosed",
    ];
    for (const eventName of referenceEvents) {
      const eventBlock = programSource.match(new RegExp(`pub struct ${eventName} \\{([\\s\\S]*?)\\}`))?.[1] || "";
      assert.match(eventBlock, /reference: \[u8; 32\]/, `${eventName} should carry a ledger reference`);
    }
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
    assert.match(programSource, /token::transfer_checked/);
    assert.match(programSource, /token::mint_to/);
    assert.match(programSource, /token::burn/);
  });
});
