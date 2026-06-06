# MergeOS Solana Contracts

This package is the Solana home for MRG, escrow, treasury, payout, and legacy wallet migration state.

The program is written as an Anchor-compatible scaffold so the web app, protocol schemas, SDK helpers, and external agents agree on the same instruction names, account seeds, and byte formats before the program is deployed.

## Program

- Program: `mergeos_mrg`
- Token symbol: `MRG`
- Target chain: Solana
- Legacy chains: `trc20`, `evm`
- Public IDL: `/contracts/solana/mergeos_mrg.v1.idl.json`

## Instructions

| Instruction | Purpose |
| --- | --- |
| `initialize_treasury` | Stores the MergeOS treasury authority, MRG mint, treasury receiver token account, and token metadata. |
| `mint_verified_mrg` | Mints MRG after MergeOS records a verified funding ledger row. The ledger row hash is stored as a 32-byte `ledger_reference`. |
| `open_escrow` | Locks MRG into a project escrow vault keyed by a 32-byte `project_id`. |
| `release_payout` | Releases MRG from an escrow vault to a worker token account and records a payout receipt keyed by a 32-byte `payout_id`. |
| `register_legacy_wallet` | Links an old TRC20/EVM wallet hash to a Solana wallet for migration proof. |

## PDA Seeds

The wallet migration PDA must match `mergeos.wallet-migration.v1` and the SDK helpers:

```text
["wallet-migration", legacy_chain, legacy_address_hash_bytes]
```

The third seed is **raw 32-byte data** decoded from `contract.args.legacy_address_hash`. Do not pass the 64-character hex string as UTF-8 bytes.

Other PDAs:

```text
["treasury"]
["escrow", project_id_bytes]
["payout", payout_id_bytes]
```

## Token Account Guards

The program constrains every MRG movement to the configured treasury mint:

- `mint_verified_mrg` only mints into a receiver token account whose mint matches `treasury_config.token_mint`.
- `open_escrow` requires the requested mint to match `treasury_config.token_mint`, the funder account to be owned by the funder, and the escrow token account to be owned by the escrow PDA.
- `release_payout` requires both the escrow and worker token accounts to use the configured MRG mint, and requires the escrow token account to remain owned by the escrow PDA.

## Local Workflow

With Anchor installed:

```bash
anchor build
anchor test
```

Without Anchor, CI still verifies the published IDL and the static frontend copy:

```bash
node --test contracts/solana/test/idl.test.js
```

## Deployment Notes

After deployment:

1. Set `CRYPTO_TOKEN_CONTRACT` to the MRG SPL mint.
2. Set `CRYPTO_RECEIVER` to the treasury receiver token account.
3. Set `MRG_SOLANA_PROGRAM_ID` to the deployed `mergeos_mrg` program id.
4. Keep `wallet-migration.v1` responses on `program_ready=false` until the program id is a valid Solana public key.
