# Security Threat Model & Auditor Package

This document presents the security threat model, module audits, and internal penetration testing details for the Sovereign L1 Blockchain.

---

## 1. System Invariants & Core Verification Strategy

The security-critical custom modules in Sovereign L1 implement five key invariants to enforce structural and state consistency. Because the standard Cosmos SDK `crisis` module is bypassed, validation is verified via the E2E verification test suite (`phase_8_verification_test.go`) simulating state transitions and executing invariant logic.

| Module | Invariant Name | Mitigation Objective | Breach Condition |
| :--- | :--- | :--- | :--- |
| `x/oracle` | `OracleStaleness` | Protect against stale oracle prices in downstream logic. | Price feed contains an aggregated price older than `StalenessThresholdBlocks`. |
| `x/bridge` | `NonceBitmap` | Prevent double-spend / replay attacks on token bridge. | The nonce bitmap word is empty (all zeros) for a recorded nonce range. |
| `x/validator` | `RewardsBucket` | Prevent reward over-allocations and validator fund leakage. | Outstanding validator rewards exceed the distribution module account's native coin balance. |
| `x/certification` | `WindowConsistency` | Prevent spoofing or manipulation of liveness tracking window. | A validator's signed extension count does not match the sum of bits in the sliding window. |
| `x/bridge` | `SupplyCap` | Prevent mint/bridge-in exploits beyond max allowed supply. | Sum of bridge-minted tokens exceeds the defined global `SupplyCap`. |

---

## 2. Threat Analysis & Penetration Vectors

We evaluate potential threats targeting the Sovereign L1 system components:

### 2.1 Oracle Manipulation & Price Feed Hijack
- **Threat Vector**: Colluding validator/operators report artificial prices to force milestone triggers or alter precompile consumption costs.
- **Mitigation**:
  - Outlier filtering using Median Absolute Deviation (MAD).
  - Staleness invariant checking to disable milestone evaluation on stale feeds.
  - Delay windows separating commit and reveal phases.

### 2.2 Bridge Nonce Replay & double-spending
- **Threat Vector**: Submitting a previously executed `MsgBridgeIn` transaction to mint tokens multiple times.
- **Mitigation**:
  - Nonce bitmap records every processed cross-chain transfer.
  - Invariant checks verify bitmap word integrity.
  - Multi-sig or threshold-signature validation for witness relayers.

### 2.3 Validator Liveness Tracking Mismatch
- **Threat Vector**: Validators manipulating block proposal/extension sign counts to avoid slashing or ejection.
- **Mitigation**:
  - Sliding window bit-vector tracked in `x/certification`.
  - Invariant checks verify that total signed counts are mathematically consistent with the sliding window bit state.

### 2.4 Reward Account Insolvency
- **Threat Vector**: Staking rewards are over-allocated or unauthorized withdrawals occur in validator rewards.
- **Mitigation**:
  - Invariant checks assert `sum(outstanding_rewards) <= balance(distribution_module_account)`.
  - Stubs for `BankKeeper` and `DistrKeeper` dynamically query balance states for verification.

---

## 3. Penetration Test & Audit Verification Plan

Audit verification is automated via the test suite:
1. **Mock State Setup**: Populate KVStore directly with invalid/corrupted records matching the breach condition.
2. **Invariant Run**: Run the keeper's registered invariant function.
3. **Breach Assertions**: Ensure the invariant returns a non-empty descriptive text and `true` (indicating a breach has occurred).
4. **Reproducible Builds**: Validate that the build artifact is deterministic across different machines.
