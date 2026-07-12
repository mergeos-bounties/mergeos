# NeraJob in the MergeOS ecosystem

**NeraJob** is a Python toolkit for:

1. Scanning public job boards (pluggable scrapers)  
2. Building CVs from a local profile  
3. Preparing apply packages (cover notes + checklists)  

Repository: [mergeos-bounties/NeraJob](https://github.com/mergeos-bounties/NeraJob)

## Why it sits next to MergeOS

MergeOS funds and proves software delivery. NeraJob is a product surface in the same organization: contributors can claim **scraper and tooling bounties**, merge PRs, and receive **MRG** through the MergeOS ledger—the same economic rails as mergeos core work.

## Current capabilities (v0.1)

- CLI: `nerajob profile`, `scan`, `cv`, `apply`, `jobs`  
- Sample offline feed + RemoteOK adapter  
- Local JSON storage under `data/`  
- Extensible `BaseScraper` registry  

## Open bounty themes

- Remotive, Arbeitnow, Adzuna, USAJOBS, Greenhouse, Lever, and more  
- Rate-limited shared HTTP client  
- Match scoring and PDF CV export  
- Multi-source pack with offline CI mocks  

## Get started

```bash
git clone https://github.com/mergeos-bounties/NeraJob.git
cd NeraJob
python -m venv .venv
# activate venv, then:
pip install -e ".[dev]"
nerajob profile init
nerajob scan -q "python" -n 20
```

Claim workflow: [How to claim MRG bounties](/blog/how-to-claim-mrg-bounties) · [NeraJob BOUNTY.md](https://github.com/mergeos-bounties/NeraJob/blob/master/docs/BOUNTY.md)
