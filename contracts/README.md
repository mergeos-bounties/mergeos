# MergeOS Contracts

This package contains the first MergeOS contract sources for the MRG token economy.

## Contracts

- `MergeOSToken.sol`: minimal ERC20-compatible MRG token with owner-controlled minters.
- `MergeOSTreasury.sol`: treasury vault for operator-approved MRG releases and manual owner sweeps.
- `MergeOSEscrow.sol`: project escrow ledger for deposits, platform fee routing, referenced task reserves, worker payouts, and refunds.
- `MergeOSPayouts.sol`: payout approval ledger that executes approved references once through the treasury.

## Security Invariants

- No contract uses `tx.origin`, `selfdestruct`, `delegatecall`, or `callcode`.
- Treasury and escrow releases require owner or trusted operator authorization.
- Payout approvals require owner or trusted operator authorization and can only execute from the `Approved` state.
- Escrow token transfers use explicit ERC20 return-value checks.
- Escrow payout and refund paths use a local `nonReentrant` guard.
- Escrow task reserve, payout, and refund events require non-zero references so off-chain PR/task evidence can be reconciled with on-chain activity.
- Project funding splits platform fee from the worker pool before task reserves are created.

## Test

```powershell
npm test
```

The current test suite is static and dependency-free so it can run without a Solidity toolchain. Add Foundry or Hardhat tests before production deployment.
