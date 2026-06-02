import { describe, it } from "node:test";
import assert from "node:assert/strict";
import { readFileSync, readdirSync } from "node:fs";
import { join } from "node:path";
import { fileURLToPath } from "node:url";

const root = fileURLToPath(new URL("..", import.meta.url));
const src = join(root, "src");
const sources = Object.fromEntries(
  readdirSync(src)
    .filter((file) => file.endsWith(".sol"))
    .map((file) => [file, readFileSync(join(src, file), "utf8")]),
);

describe("contract package", () => {
  it("contains the required MergeOS contract sources", () => {
    assert.deepEqual(Object.keys(sources).sort(), [
      "MergeOSEscrow.sol",
      "MergeOSPayouts.sol",
      "MergeOSToken.sol",
      "MergeOSTreasury.sol",
    ]);
  });

  it("uses Solidity 0.8.24 and avoids banned low-level primitives", () => {
    const banned = [/\btx\.origin\b/, /\bselfdestruct\b/, /\bdelegatecall\b/, /\bcallcode\b/];

    for (const [file, source] of Object.entries(sources)) {
      assert.match(source, /pragma solidity \^0\.8\.24;/, `${file} should pin compiler family`);
      for (const pattern of banned) {
        assert.doesNotMatch(source, pattern, `${file} must not include ${pattern}`);
      }
    }
  });
});

describe("MergeOSToken", () => {
  const source = sources["MergeOSToken.sol"];

  it("exposes ERC20-compatible surface and controlled minting", () => {
    assert.match(source, /contract MergeOSToken/);
    assert.match(source, /event Transfer\(address indexed from, address indexed to, uint256 amount\)/);
    assert.match(source, /event Approval\(address indexed owner, address indexed spender, uint256 amount\)/);
    assert.match(source, /function mint\(address to, uint256 amount\) external onlyMinter/);
    assert.match(source, /function setMinter\(address account, bool enabled\) external onlyOwner/);
    assert.match(source, /function burn\(uint256 amount\) external/);
  });
});

describe("MergeOSTreasury", () => {
  const source = sources["MergeOSTreasury.sol"];

  it("requires operators for payout release", () => {
    assert.match(source, /contract MergeOSTreasury/);
    assert.match(source, /event TreasuryRelease\(address indexed recipient, uint256 amount, bytes32 indexed reference\)/);
    assert.match(source, /function release\(address recipient, uint256 amount, bytes32 reference\) external onlyOperator/);
    assert.match(source, /function setOperator\(address account, bool enabled\) external onlyOwner/);
    assert.match(source, /function _safeTransfer\(address recipient, uint256 amount\) private/);
  });
});

describe("MergeOSPayouts", () => {
  const source = sources["MergeOSPayouts.sol"];

  it("records approved payouts and executes each reference through treasury", () => {
    assert.match(source, /contract MergeOSPayouts/);
    assert.match(source, /interface IMergeOSPayoutTreasury/);
    assert.match(source, /enum PayoutStatus/);
    assert.match(source, /struct Payout/);
    assert.match(source, /event PayoutApproved\(bytes32 indexed payoutId, address indexed recipient, uint256 amount, bytes32 indexed reference\)/);
    assert.match(source, /event PayoutExecuted\(bytes32 indexed payoutId, address indexed recipient, uint256 amount, bytes32 indexed reference\)/);
    assert.match(source, /mapping\(bytes32 => bool\) public reservedReferences/);
    assert.match(source, /function approvePayout\(/);
    assert.match(source, /function executePayout\(bytes32 payoutId\) external onlyOperator/);
    assert.match(source, /treasury\.release\(payout\.recipient, payout\.amount, payout\.reference\)/);
    assert.match(source, /function cancelPayout\(bytes32 payoutId\) external onlyOperator/);
  });

  it("prevents duplicate or unapproved payout execution", () => {
    assert.match(source, /if \(payouts\[payoutId\]\.status != PayoutStatus\.None\) revert PayoutExists\(\)/);
    assert.match(source, /if \(reservedReferences\[reference\]\) revert ReferenceExists\(\)/);
    assert.match(source, /reservedReferences\[reference\] = true/);
    assert.match(source, /if \(payout\.status != PayoutStatus\.Approved\) revert PayoutNotApproved\(\)/);
    assert.match(source, /payout\.status = PayoutStatus\.Executed/);
    assert.match(source, /payout\.status = PayoutStatus\.Cancelled/);
    assert.match(source, /reservedReferences\[payout\.reference\] = false/);
  });
});

describe("MergeOSEscrow", () => {
  const source = sources["MergeOSEscrow.sol"];

  it("tracks project funding, task reserves, release, and refund flows", () => {
    assert.match(source, /contract MergeOSEscrow/);
    assert.match(source, /enum ProjectStatus/);
    assert.match(source, /struct ProjectEscrow/);
    assert.match(source, /struct TaskReserve[\s\S]*bytes32 reserveReference/);
    assert.match(source, /function fundProject\(/);
    assert.match(source, /_safeTransferFrom\(client, address\(this\), amount\)/);
    assert.match(source, /function reserveTask\(/);
    assert.match(source, /function releaseTask\(bytes32 taskId, bytes32 reference\) external onlyOperator nonReentrant/);
    assert.match(source, /function refundTask\(bytes32 taskId, address recipient, bytes32 reference\) external onlyOperator nonReentrant/);
    assert.match(source, /function refundProjectRemainder\(/);
  });

  it("guards external value movement", () => {
    assert.match(source, /modifier nonReentrant\(\)/);
    assert.match(source, /function _safeTransfer\(address recipient, uint256 amount\) private/);
    assert.match(source, /function _safeTransferFrom\(address from, address recipient, uint256 amount\) private/);
    assert.match(source, /event ProjectFunded\(/);
    assert.match(source, /event TaskReserved\([\s\S]*bytes32 reference[\s\S]*\)/);
    assert.match(source, /event TaskPaid\([\s\S]*bytes32 reference[\s\S]*\)/);
    assert.match(source, /event TaskRefunded\([\s\S]*bytes32 reference[\s\S]*\)/);
    assert.match(source, /if \(reference == bytes32\(0\)\) revert InvalidReference\(\)/);
    assert.match(source, /event ProjectRefunded\(/);
  });
});
