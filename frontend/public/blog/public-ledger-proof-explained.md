# Public ledger proof explained

MergeOS publishes a **public ledger** so funding, reserves, and payouts are inspectable without exposing private customer secrets.

## What you can verify

- Project funding and token mint rows  
- Escrow / reserve movements  
- Task payments and manual credits to `github:username` or wallets  
- Hash chain integrity via verification APIs  

## Scan explorer

[scan.mergeos.shop](https://scan.mergeos.shop) presents ledger rows in a BscScan-style UI: addresses, sequences, and transaction-like references.

GitHub worker aliases such as `github:alice` and `worker:github:alice` are normalized so contributor payouts aggregate on one public identity where possible.

## For auditors and customers

1. Open the public ledger or Scan address page  
2. Confirm payout reference includes the merged PR URL when applicable  
3. Verify hash chaining with the public verify endpoint  

Transparency does not replace private security review—but it makes *economic* outcomes hard to rewrite silently.

Related: [Escrow-backed software delivery](/blog/escrow-backed-software-delivery).
