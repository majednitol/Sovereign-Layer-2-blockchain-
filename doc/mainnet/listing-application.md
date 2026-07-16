# CoinMarketCap and CoinGecko Listing Application Guide

This document outlines the API endpoints, token details, and supply schedules required to submit listing applications to CoinMarketCap (CMC) and CoinGecko (CG).

---

## 1. Token Asset Specifications

- **Token Name:** Sovereign Token
- **Token Symbol:** WSOV
- **Sovereign L1 Minimal Denom:** `uwsov`
- **Sovereign L1 Decimals:** 6
- **BSC Wrapped Token Address:** *[To be populated by the contract deployer post-migration/deployment]*
- **BSC Wrapped Token Decimals:** 18
- **Block Explorers:**
  - Sovereign L1 Explorer: `https://explorer.sovereign.l1`
  - BSC Scan: `https://bscscan.com`

---

## 2. Dynamic Supply API Endpoints (Required by CMC & CG)

Listing sites require plain-text (JSON or raw text) endpoints that output the current supply metrics dynamically.

### Total Supply Endpoint
- **URL:** `/api/total-supply`
- **Format:** Raw numeric text (e.g. `100000000`)
- **Query Parameters:** `?format=raw` (returns raw string) or `?format=json` (returns JSON payload)

### Circulating Supply Endpoint
- **URL:** `/api/circulating-supply`
- **Format:** Raw numeric text (e.g. `15000000`)
- **Logic:** `Total Supply` minus `Earmarked Treasury Balance` minus `Locked Bridge Pool / Relayer Pool`.

---

## 3. Supply Verification and Auditing

To verify the supply figures:
- Ensure the relayer account address and bridge lockbox balance on Sovereign L1 are excluded from circulating supply.
- Use `chaind query bank balances <address>` to audit on-chain accounts.
