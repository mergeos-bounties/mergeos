# MergeOS Contracts

This package contains the first MergeOS contract sources for the MRG token economy.

## Contracts

- `MergeOSToken.sol`: minimal ERC20-compatible MRG token with owner-controlled minters.
- `MergeOSTreasury.sol`: treasury vault for operator-approved MRG releases and manual owner sweeps.
- `MergeOSEscrow.sol`: project escrow ledger for deposits, platform fee routing, task reserves, worker payouts, and refunds.

## Security Invariants

- No contract uses `tx.origin`, `selfdestruct`, `delegatecall`, or `callcode`.
- Treasury and escrow releases require owner or trusted operator authorization.
- Escrow token transfers use explicit ERC20 return-value checks.
- Escrow payout and refund paths use a local `nonReentrant` guard.
- Project funding splits platform fee from the worker pool before task reserves are created.

## Test

```powershell
npm test
```

The current test suite is static and dependency-free so it can run without a Solidity toolchain. Add Foundry or Hardhat tests before production deployment.
