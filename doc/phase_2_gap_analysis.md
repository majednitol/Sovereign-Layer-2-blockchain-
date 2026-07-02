# Phase 2 Implementation Gap Analysis

This document evaluates the implementation of **Phase 2 (Custom Cosmos SDK Modules)** against the specifications in [implementation_plan.md](file:///Users/majedurrahman/Sovereign/implementation_plan.md).

---

## 1. Directory Layout & Folder Renaming Gaps

### A. Off-Chain Monolith Subfolders (Plan vs Codebase)
* **Gap Identified**: The user requested folder naming updates. The master plan lists the backend components under the `/backend` directory as:
  - `module/ingestion`
  - `module/projection`
  - `module/api`
* **Current Codebase Implementation**: The directories in the codebase are named:
  - `backend/cmd/ingestion`
  - `backend/cmd/projection`
  - `backend/cmd/api`
* **Status**: **UNRESOLVED FOLDER NAME CHANGE**. To align the codebase with the updated plan, the `cmd` folder inside the `backend/` sub-project must be renamed to `module`, and references in `backend/Dockerfile` and the root `Makefile` must be updated.

---

## 2. Identified Module Gaps

### A. `x/validator` (Section 2.1)
1. **Slashing Sync (Tombstone)**: The custom keeper declares a `SlashingKeeper` interface containing `Tombstone(ctx, ConsAddress) error`, but this is **never invoked** when a validator is ejected inside `EndBlocker`.
2. **ValidatorSigningInfo Creation**: There is no call to initialize `ValidatorSigningInfo` when a validator slot is filled.
3. **Simulation Operations**: `SimGovProposalUpdatePartitionScheme` (weight: 5) is completely missing from `simulation.go`.

### B. `x/certification` (Section 2.2)
1. **Hardcoded Consecutive Rejections**: The limit of consecutive block proposal rejections that triggers degraded mode is hardcoded to `5` in `EndBlocker` instead of retrieving it dynamically from `params.MaxConsecutiveRejections`.
2. **Missed Extensions Slashing**: Slashing for systematic withholding (M consecutive missed extensions) does not integrate with `x/slashing` and is currently a stub.
3. **Attestation Bootstrapping Window**: Rolling window tracking and using `actual_block_count` as a denominator for blocks $1 \le H \le window\_size$ is missing.
4. **Simulation Operations**: `SimDropValidatorAttestation` and `SimRestoreValidatorAttestation` operations are missing.

### C. `x/milestone` (Section 2.4)
1. **O(1) Milestone Indexing**: Milestones are currently stored sequentially and iterated using a standard prefix iterator in `EndBlocker`. The plan's requirement to index milestones by oracle feed dependency to skip stale feeds in $O(1)$ is not implemented.
2. **Simulation Operations**: `SimMsgAchieveMilestone`, `SimMilestoneExpiry`, and `SimMilestoneStaleRecovery` are missing.

### D. `x/governance-ext` (Section 2.6)
1. **Custom Proposal Message Types**: Custom proposals (`UpdateValidatorSlot`, `UpdateMilestone`, `UpdateOracleOperator`, `UpdateWitnessRegistry`, `UpdateBridgeRelayerSet`) are not implemented as concrete message structures or handled.
2. **Simulation Operations**: `SimGovProposalCustom` is missing.

---

## 3. Gap Remediation Plan

To completely resolve the identified Phase 2 gaps:
1. **Rename Off-Chain Subfolders**: Rename `backend/cmd` to `backend/module`, and update path references in `Makefile` and `backend/Dockerfile`.
2. **Implement Slashing Integration**:
   - Resolve validator consensus addresses during ejection to trigger `slashingKeeper.Tombstone`.
   - Wire `slashingKeeper` to hook into slot allocations and instantiate signing info.
3. **Dynamic Rejection Limit**: Update `x/certification` `EndBlocker` to retrieve consecutive rejection limits from parameter state.
4. **Implement Attestation Rolling Window**: Implement rolling window tracking for attestation finality in `x/certification`.
5. **Optimize Milestone Processing**: Index active milestones by feed dependency in KV-store to support $O(1)$ skips in `x/milestone`.
