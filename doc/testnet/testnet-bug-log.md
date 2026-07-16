# Public Testnet Bug Log — Sovereign L1

This document tracks all bugs found during the public testnet stability window (Phase G), along with periodic bridge invariant spot checks.

---

## 1. Bug Tracking Log

| Bug ID | Severity | Date Found | Component | Description | Fix Status | Fix Commit |
|:---|:---|:---|:---|:---|:---|:---|
| BUG-001 | High | 2026-XX-XX | Relayer | Example bug details. | Resolved | `abc123d` |

---

## 2. Periodic Bridge Invariant Checks

The following query template can be used to compare `cosmos_minted` with the actual BSC locked balance:
```sql
-- Query on Read DB to get total minted on Cosmos side
SELECT total_minted FROM bridge_volume_1h ORDER BY time DESC LIMIT 1;
```

Check Log:

| Date | Height | Cosmos Minted (WSOV) | BSC Locked (WSOV) | Match? | Sign-off / Operator |
|:---|:---|:---|:---|:---|:---|
| | | | | | |
