# ADR 010: Fee Market Consolidation

## Context & Problem Statement
Sovereign L1 originally considered integrating `skip-mev/x/feemarket` to handle dynamic base fees. However, introducing multiple fee market systems (one for Cosmos SDK transactions and another for EVM transactions) creates complexity, inconsistent fee calculations, and potential MEV exploits. We need a unified fee model.

## Decision & Design

1. **Elimination of `skip-mev`**: We reject the inclusion of `skip-mev/x/feemarket` in `go.mod`.
2. **Unified Fee Market Module**: We exclusively use the native `x/feemarket` module bundled within the `cosmos/evm` repository (`github.com/cosmos/evm/x/feemarket`).
3. **Cosmos & EVM Coexistence**: The native `cosmos/evm/x/feemarket` module handles fee calculations for both standard Cosmos SDK messages and EVM transactions through a unified EIP-1559 implementation.
4. **Genesis Parameters**:
   - `NoBaseFee = false` (enables base fee calculation from block 1)
   - `ElasticityMultiplier = 2`
   - `EnableHeight = 0` (immediately active)
5. **Ante Handler Integration**: The ante handler checks the gas price against the computed dynamic base fee for all transaction types, rejecting any transaction that underpays the base fee.
