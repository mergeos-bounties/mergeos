# MergeOS Solana Contracts

This package now contains the MergeOS Solana/Anchor program for the MRG token economy.

## Program

- `solana/programs/mergeos-mrg/src/lib.rs`: active Anchor program for MRG SPL mint operations, project escrow, payout receipts, treasury config, and legacy wallet migration.
- `solana/idl/mergeos_mrg.json`: source IDL mirrored to the frontend public contract artifact.
- `solana/mergeos_mrg.proof-manifest.v1.json`: source proof manifest mirrored to the frontend public contract artifact.
- `solana/Anchor.toml`: active localnet Anchor workspace configuration.
- `solana/Cargo.toml`: Rust workspace configuration for the active Solana program.

The checked-in localnet program id is `4gUBWum3fGKfm7BeGXryzXjPDBDLfhVJRcjN5MPnfDNW`. Production must deploy the program with its own keypair and set `MRG_SOLANA_PROGRAM_ID`; the backend leaves `program_ready` false when that value is missing.

## Migration From TRC20/EVM

Legacy TRC20/TRON and EVM wallet identifiers are not used as payout accounts anymore. Backend state migration links the old address to a Solana MRG wallet, then the Solana program records the old-chain proof through `register_legacy_wallet`.

The `WalletMigration` account stores:

- `legacy_chain`: `Trc20` or `Evm`.
- `legacy_address_hash`: a 32-byte hash of the old wallet address.
- `solana_wallet`: the new Solana wallet public key.
- `owner`: the MergeOS user/operator that registered the migration.

## Security Invariants

- No Solidity or EVM primitives remain in this package.
- MRG minting uses SPL Token CPI `mint_to`.
- Token movement uses SPL Token CPI `transfer`.
- Every money-moving event carries a `[u8; 32] ledger_reference` for MergeOS ledger reconciliation.
- Treasury, escrow vault, payout receipt, and wallet migration accounts are PDA-backed records keyed by public proof ids.
- MRG token accounts are constrained to the configured treasury mint and expected signer/PDA authorities.
- Legacy wallet migration stores hashes, not raw old wallet text.

## Ledger References

References are deterministic 32-byte proof anchors from MergeOS public ledger rows. Backend operators should derive them from ledger `entry_hash`, `public_hash`, or the `contractReferenceFromLedger` helper in `@mergeos/sdk` / `@mergeos/protocol`.

```js
import { contractReferenceFromLedger, legacyWalletAddressHash } from '@mergeos/sdk';

const reference = contractReferenceFromLedger(publicLedgerEntry, { format: 'bytes' });
const legacyAddressHash = legacyWalletAddressHash('trc20', legacyAddress, { format: 'bytes' });
```

Use the resulting arrays directly for Anchor instruction args such as `ledger_reference: [u8; 32]` and `legacy_address_hash: [u8; 32]`.

For `register_legacy_wallet`, derive the PDA with seeds `[b"wallet-migration", legacy_chain.seed(), legacy_address_hash.as_ref()]`. The hash seed is raw 32-byte data, not the 64-character hex string.

## Test

```powershell
npm test
```

The current suite is static and dependency-free, so it runs without an Anchor toolchain. Run `anchor test` with a Solana local validator before production deployment.
