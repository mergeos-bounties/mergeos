# MergeOS Solana Contracts

This package now contains the MergeOS Solana/Anchor program for the MRG token economy.

## Program

- `programs/mergeos/src/lib.rs`: Anchor program for MRG SPL mint operations, project escrow, task payout/refund, project close proof events, and legacy wallet migration.
- `Anchor.toml`: localnet Anchor workspace configuration.
- `Cargo.toml`: Rust workspace configuration for the Solana program.

## Migration From TRC20/EVM

Legacy TRC20/TRON and EVM wallet identifiers are not used as payout accounts anymore. Backend state migration hashes each legacy wallet into a deterministic Solana wallet address for internal continuity, then the Solana program records the old-chain proof through `register_legacy_wallet`.

The `WalletMigration` account stores:

- `legacy_chain`: `Trc20` or `Evm`.
- `legacy_address_hash`: a 32-byte hash of the old wallet address.
- `solana_wallet`: the new Solana wallet public key.
- `registered_by`: the operator that registered the migration.

## Security Invariants

- No Solidity or EVM primitives remain in this package.
- MRG minting uses SPL Token CPI `mint_to`.
- Token movement uses SPL Token CPI `transfer_checked`.
- Token burning uses SPL Token CPI `burn`.
- Every money-moving event carries a `[u8; 32] reference` for MergeOS ledger reconciliation.
- Escrow and task reserve accounts are PDA-backed records, keyed by project/task proof ids.
- Legacy wallet migration stores hashes, not raw old wallet text.

## Ledger References

References are deterministic 32-byte proof anchors from MergeOS public ledger rows. Backend operators should derive them from ledger `entry_hash`, `public_hash`, or the `contractReferenceFromLedger` helper in `@mergeos/sdk` / `@mergeos/protocol`.

## Test

```powershell
npm test
```

The current suite is static and dependency-free, so it runs without an Anchor toolchain. Run `anchor test` with a Solana local validator before production deployment.
