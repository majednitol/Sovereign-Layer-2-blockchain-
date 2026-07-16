# Token Economics (Tokenomics) — Sovereign L1

This document outlines the allocation, supply details, and tokenomics model for the Sovereign L1 Blockchain.

---

## 1. Supply Constants

| Parameter | Value | Description |
|:---|:---|:---|
| **Total Supply ($S$)** | `1,000,000,000 CSOV` | Fixed maximum supply of the native token. |
| **Cosmos Base Denomination** | `ucsov` (CSOV) | 6 decimals, used for native staking and governance. |
| **EVM Denomination** | `aesov` (ESOV) | 18 decimals, gas token for the EVM execution layer. |
| **Bridge Wrapped Denomination** | `uwsov` (WSOV) | 6 decimals, Cosmos-wrapped representation of BSC locked tokens. |
| **Inflation Rate** | `0%` | Hard-capped fixed supply; zero block inflation. |

---

## 2. Supply Allocation & Vesting

```
      ┌──────────────────────────────────────────────────────────┐
      │                   CSOV Total Genesis Supply              │
      │                        (1,000,000,000)                   │
      └───────┬──────────────────────┬────────────────────┬──────┘
              │                      │                    │
              ▼                      ▼                    ▼
      ┌──────────────┐       ┌──────────────┐      ┌──────────────┐
      │  Cosmos L1   │       │ BSC LockBox  │      │ Validator    │
      │  Allocation  │       │ Escrow (BSC) │      │ Stake Pool   │
      │   (S - C)    │       │     (C)      │      │  (Genesis)   │
      └──────────────┘       └──────────────┘      └──────────────┘
```

The total genesis allocation is divided as follows:

| Category | Percent | CSOV Amount | Vesting Schedule |
|:---|:---|:---|:---|
| **Treasury Contract** | `🔑 OWNER ACTION` | `🔑 OWNER ACTION` | Governance-controlled milestone disburse. |
| **Reserve Fund Contract** | `🔑 OWNER ACTION` | `🔑 OWNER ACTION` | Milestone achievement gated disburse. |
| **Validator Genesis Stake** | `🔑 OWNER ACTION` | `🔑 OWNER ACTION` | Equalized slots staking keys. |
| **Team & Advisors** | `🔑 OWNER ACTION` | `🔑 OWNER ACTION` | 12-month cliff, 36-month linear vesting. |
| **Community & Ecosystem** | `🔑 OWNER ACTION` | `🔑 OWNER ACTION` | Delegated to community DAO pool. |
| **Public Sale / Airdrop** | `🔑 OWNER ACTION` | `🔑 OWNER ACTION` | Unlocked at launch / staggered release. |
| **Bridge LockBox Escrow (BSC)** | `🔑 OWNER ACTION` | `🔑 OWNER ACTION` | Escrowed in Solidity contract on BSC. |

---

## 3. ESOV/CSOV Peg Mechanism

`🔑 OWNER ACTION — requires project owner definition`

Choose one of the following models to define the relationship between `aesov` (ESOV) and `ucsov` (CSOV):
- **Model A (Dual-Precision Peg)**: ESOV is pegged 1:1 to CSOV (1 CSOV = 1,000,000 ucsov = 1,000,000,000,000,000,000 aesov). The EVM and Cosmos layers use the same underlying token pool, linked via the `x/erc20` keeper.
- **Model B (Independent Valuation)**: ESOV and CSOV are independent assets. A dynamic oracle feed or exchange rate determines their conversion ratio.
