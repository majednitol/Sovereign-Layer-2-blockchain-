# Phase 1 Implementation Gap Analysis

This document evaluates the implementation of **Phase 1 (Chain Scaffold & Genesis Configuration)** against the requirements laid out in the master [implementation_plan.md](file:///Users/majedurrahman/Sovereign/implementation_plan.md).

---

## 1. Resolved Items

### A. Extended Governance Directory Name
* **Status**: **RESOLVED**
* **Details**: The directory was named `chain/x/governance-ext` to match the plan, but all Go source files were importing `github.com/sovereign-l1/chain/x/gov_ext`, preventing compilation. This import mismatch has been corrected across `app.go`, `simulation_test.go`, and the `e2e` test files.

### B. EVM Authz Blocklist Registration
* **Status**: **RESOLVED**
* **Details**: The EVM transaction type `/cosmos.evm.vm.v1.MsgEthereumTx` has been added to `setupAuthzBlockedMessages` in `chain/app/app.go` alongside `MsgBridgeIn`, `MsgBridgeOut`, `MsgSubmitOracleCommit`, `MsgRevealOracleReport`, and `MsgSettlement`.

---

## 2. Identified Gaps (Remaining for future phases)

### A. EVM Module Wiring & Imports (Sections 1.1, 1.4, 1.6)
* **Gap 1: Missing `cosmos/evm` Modules**: The `app.go` does not import or wire the three required EVM modules:
  - `x/feemarket` (`github.com/cosmos/evm/x/feemarket`)
  - `x/vm` (`github.com/cosmos/evm/x/vm` - EVM engine)
  - `x/erc20` (`github.com/cosmos/evm/x/erc20` - Token wrapping)
* **Gap 2: Wrong Fee Market Dependency**: The application imports and wires `github.com/skip-mev/feemarket` instead of the bundled `github.com/cosmos/evm/x/feemarket` module.

### B. IBC Protocol Integration (Section 1.1)
* **Gap 3: Missing IBC Wiring**: No `ibc-go` modules are wired or initialized in [app.go](file:///Users/majedurrahman/Sovereign/chain/app/app.go). 
* **Gap 4: Pinned IBC Version Mismatch**: The workspace pins `github.com/cosmos/ibc-go/v8` in [go.mod](file:///Users/majedurrahman/Sovereign/chain/go.mod), whereas the plan specifies `ibc-go v11`.

### C. Staking Compatibility (Section 1.2)
* **Gap 5: Staking & Distribution Hooks Stubbed**: In [staking_compatibility.go](file:///Users/majedurrahman/Sovereign/chain/app/staking_compatibility.go), the slot-based rewards distribution hook and consensus slashing integration are stubs. Rewards are not integrated to override the standard `x/distribution` reward pools, and missed-block jailing is not hooked to slot ejections.
* **Gap 6: Historical Info Override**: IBC light-client `HistoricalInfo` overrides are omitted (dependent on the missing IBC modules).

### D. EVM Coexistence Ante Handler (Section 1.4)
* **Gap 7: Missing Ante Handler Routing**: The ante handler does not route `MsgEthereumTx` through EVM decorators and CosmWasm messages through Cosmos decorators.

---

## 3. Gap Remediation Plan

To address these gaps in the upcoming implementation phases:
1. **Remove skip-mev/feemarket**: Deprecate skip-mev and import `github.com/cosmos/evm`.
2. **Wire EVM in app.go**: Register `vm`, `feemarket`, and `erc20` in the basic manager and keepers. Set the init/begin/end block ordering.
3. **Implement ante.go**: Wire `cosmos/evm/ante.NewAnteHandler` as the unified ante handler.
4. **Wire ibc-go**: Import and register the IBC module set.
5. **Integrate rewards split**: Update `staking_compatibility.go` to intercept `AllocateTokens` and distribute block rewards equally.
