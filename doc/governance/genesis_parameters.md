# Genesis Parameters & Governance Constants

This document outlines the formal technical and governance constants established for the Sovereign Layer-1 Blockchain during Phase 0.

## 1. Supply & Economic Parameters

| Parameter | Value | Description |
|-----------|-------|-------------|
| **Total Supply ($S$)** | `1,000,000,000 CSOV` | Fixed maximum token supply at genesis. |
| **Cosmos Allocation** | `S - C` | Tokens minted directly on the Sovereign L1 Cosmos side. |
| **Bridge Escrow ($C$)** | Dynamic | Circulating supply of the ERC-20 token on BNB Smart Chain (BSC) locked in the bridge lockbox. |
| **Base Denomination** | `ucsov` (CSOV) | Native Cosmos SDK staking/fee token (6 decimals). |
| **EVM Denomination** | `aesov` (ESOV) | Gas token for the EVM execution layer (18 decimals). |
| **Bridge Denomination** | `uwsov` (WSOV) | Representation of BSC-escrowed SOV tokens on Cosmos (6 decimals). |
| **Inflation Rate** | `0%` | Hard-capped fixed supply; zero block inflation. |

---

## 2. Validator Set & Staking Constants

| Parameter | Value | Description |
|-----------|-------|-------------|
| **Active Set Size ($M$)** | `30` | Fixed slot-based active validator cardinality. |
| **Power Equalization** | `1,000,000` | Fixed voting power assigned to every active validator regardless of stake. |
| **Unbonding Period** | `1814400s` (21 days) | Duration required to unbond delegated stake. |
| **Signing Window** | `10000` blocks | Rolling block window for liveness tracking. |
| **Min Signed Threshold** | `50%` | Minimum blocks that must be signed within the window to avoid ejection. |

---

## 3. Address Prefix Settings (Bech32)

| Type | Prefix | Example |
|------|--------|---------|
| **Account address** | `cosmos` | `cosmos1qy...` |
| **Account public key** | `cosmospub` | `cosmospub1qy...` |
| **Validator operator** | `cosmosvaloper` | `cosmosvaloper1qy...` |
| **Validator consensus** | `cosmosvalcons` | `cosmosvalcons1qy...` |

---

## 4. Chain Identifiers

- **Cosmos Chain ID**: `sovereign-testnet-1` (for testnet), `sovereign-1` (for mainnet)
- **EVM Chain ID**: `9001` (registered on chainlist.org)
- **Address Derivation Path**: `m/44'/60'/0'/0/0` (Ethereum BIP-44 path for dual-address compatibility)
