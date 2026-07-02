# ADR 009: EVM Chain ID Selection & Registration

## Context & Problem Statement
Sovereign L1 runs a standard EVM execution layer side-by-side with Cosmos. The EVM layer requires a unique 64-bit integer identifier (EVM Chain ID) to prevent replay attacks across different networks (EIP-155). This identifier is separate from the alphanumeric Cosmos Chain ID (e.g. `sovereign-testnet-1`).

## Decision & Design

1. **Assigned EVM Chain ID**: `7777`
2. **Immutability**: The EVM Chain ID is set at genesis in the `x/vm` genesis parameters and cannot be altered without a hard fork.
3. **Replay Protection**: EIP-155 replay protection is strictly enforced by setting `AllowUnprotectedTxs = false` in the EVM config. Any transaction submitted without a valid signature matching Chain ID `7777` will be rejected by the ante handler.
4. **Registry Submission**: Before testnet and mainnet launches, the network metadata must be submitted to the [chainlist.org](https://chainlist.org) registry to ensure wallet and explorer compatibility.

## Metadata Reference
- **Chain ID**: `7777`
- **Network Name**: `Sovereign L1`
- **Currency Symbol**: `atoken`
- **Decimals**: `18`
