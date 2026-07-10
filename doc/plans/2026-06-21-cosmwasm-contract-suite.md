# Phase 3 CosmWasm Contract Suite Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Implement all missing and partially implemented tasks of Phase 3 (CosmWasm Contract Suite), including Rust integration tests, genesis bytecode injection, on-chain Go E2E verification, JSON schema generation, and cold multi-sig key set definition.

**Architecture:** We will update Cargo configurations to enable contract library usage, write a comprehensive multi-test integration suite in Rust, configure `scripts/generate_genesis.go` to compile and inject bytecode/state into genesis, write a Go integration test against the real Wasm app setup, and document the cold multi-sig set.

**Tech Stack:** Rust (CosmWasm, cw-multi-test, cosmwasm-schema), Go (Cosmos SDK v0.50+, x/wasm, CometBFT).

---

## Proposed Changes

### Task 1: Crate Dependency and Feature Setup
**Files:**
- Modify: [contracts/governance/Cargo.toml](file:///Users/majedurrahman/Sovereign/contracts/governance/Cargo.toml)
- Modify: [contracts/treasury/Cargo.toml](file:///Users/majedurrahman/Sovereign/contracts/treasury/Cargo.toml)
- Modify: [contracts/reserve-fund/Cargo.toml](file:///Users/majedurrahman/Sovereign/contracts/reserve-fund/Cargo.toml)

**Step 1: Update Cargo.toml dependencies**
Add `library` features and dev-dependencies so the `governance` crate can import the others in its integration test suite.

For `contracts/governance/Cargo.toml`:
```toml
[dev-dependencies]
cw-multi-test = "0.19.0"
treasury = { path = "../treasury" }
reserve-fund = { path = "../reserve-fund" }
```

For `contracts/treasury/Cargo.toml` and `contracts/reserve-fund/Cargo.toml`, ensure the following block is present:
```toml
[features]
library = []
```

Also, update `contracts/treasury/src/contract.rs` and `contracts/reserve-fund/src/contract.rs` to conditionalize `entry_point`:
```rust
#[cfg(not(feature = "library"))]
use cosmwasm_std::entry_point;
```
And replace raw `#[entry_point]` with `#[cfg_attr(not(feature = "library"), entry_point)]` on `instantiate`, `execute`, and `query` functions.

**Step 2: Verify compilation passes**
Run: `cargo check --workspace`
Expected: Compile succeeds with zero errors.

---

### Task 2: Rust Multi-Contract Integration Test Suite (`cw-multi-test`)
**Files:**
- Create: [contracts/governance/tests/integration_tests.rs](file:///Users/majedurrahman/Sovereign/contracts/governance/tests/integration_tests.rs)

**Step 1: Write integration tests covering all contract interactions**
Create a comprehensive test suite using `cw-multi-test` that deploys all 4 contracts and validates:
- Instantiation and governance address setting.
- Governance proposal checking (rejects "VIOLATION" rules, accepts compliant rules).
- Treasury withdrawals restricted to governance, emergency pause blocking withdrawals, and cold multi-sig pause.
- Reserve fund disbursement check using a custom query handler mock for `SovereignQuery::Milestone`, reentrancy guards, and minimum balance circuit-breaker.
- Governance replacement procedure (pausing, rotating pointers, unpausing).

```rust
use cosmwasm_std::{
    to_json_binary, Addr, BankMsg, Binary, Coin, Empty, StdError, Uint128,
};
use cw_multi_test::{App, AppBuilder, Contract, ContractWrapper, Executor};
use reserve_fund::msg::{SovereignQuery, MilestoneResponse};

fn constitution_contract() -> Box<dyn Contract<Empty>> {
    let contract = ContractWrapper::new(
        constitution::contract::execute,
        constitution::contract::instantiate,
        constitution::contract::query,
    );
    Box::new(contract)
}

fn treasury_contract() -> Box<dyn Contract<Empty>> {
    let contract = ContractWrapper::new(
        treasury::contract::execute,
        treasury::contract::instantiate,
        treasury::contract::query,
    );
    Box::new(contract)
}

fn reserve_contract() -> Box<dyn Contract<SovereignQuery>> {
    let contract = ContractWrapper::new(
        reserve_fund::contract::execute,
        reserve_fund::contract::instantiate,
        reserve_fund::contract::query,
    );
    Box::new(contract)
}

fn governance_contract() -> Box<dyn Contract<Empty>> {
    let contract = ContractWrapper::new(
        governance::contract::execute,
        governance::contract::instantiate,
        governance::contract::query,
    );
    Box::new(contract)
}

#[test]
fn test_full_cosmwasm_suite() {
    let mut app = AppBuilder::new_custom()
        .with_custom_handler(|_api, _storage, _querier, query: SovereignQuery| -> Result<Binary, StdError> {
            match query {
                SovereignQuery::Milestone { id } => {
                    let achieved = id == "achieved_milestone";
                    Ok(to_json_binary(&MilestoneResponse { is_achieved: achieved })?)
                }
            }
        })
        .build(|router, api, storage| {
            router
                .bank
                .init_balance(
                    storage,
                    &Addr::unchecked("treasury_addr"),
                    vec![Coin::new(1_000_000, "ucsov")],
                )
                .unwrap();
            router
                .bank
                .init_balance(
                    storage,
                    &Addr::unchecked("reserve_addr"),
                    vec![Coin::new(500_000, "ucsov")],
                )
                .unwrap();
        });

    // Deploy contracts and perform complete suite checks...
    // (Verification of pauses, minimum thresholds, compliance violations and pointers rotation)
}
```

**Step 2: Run tests to verify they pass**
Run: `cargo test --workspace`
Expected: PASS

---

### Task 3: JSON Schema Generation setup
**Files:**
- Modify: [contracts/constitution/Cargo.toml](file:///Users/majedurrahman/Sovereign/contracts/constitution/Cargo.toml)
- Modify: [contracts/treasury/Cargo.toml](file:///Users/majedurrahman/Sovereign/contracts/treasury/Cargo.toml)
- Modify: [contracts/reserve-fund/Cargo.toml](file:///Users/majedurrahman/Sovereign/contracts/reserve-fund/Cargo.toml)
- Modify: [contracts/governance/Cargo.toml](file:///Users/majedurrahman/Sovereign/contracts/governance/Cargo.toml)
- Create: [contracts/constitution/src/bin/schema.rs](file:///Users/majedurrahman/Sovereign/contracts/constitution/src/bin/schema.rs)
- Create: [contracts/treasury/src/bin/schema.rs](file:///Users/majedurrahman/Sovereign/contracts/treasury/src/bin/schema.rs)
- Create: [contracts/reserve-fund/src/bin/schema.rs](file:///Users/majedurrahman/Sovereign/contracts/reserve-fund/src/bin/schema.rs)
- Create: [contracts/governance/src/bin/schema.rs](file:///Users/majedurrahman/Sovereign/contracts/governance/src/bin/schema.rs)

**Step 1: Add cosmwasm-schema dependency**
Add `cosmwasm-schema = "1.5.0"` to the Cargo.toml of each contract.

**Step 2: Write binary schema setup**
Create `src/bin/schema.rs` in each contract to export message schemas:
```rust
use cosmwasm_schema::write_api;
use [crate_name]::msg::{ExecuteMsg, InstantiateMsg, QueryMsg};

fn main() {
    write_api! {
        instantiate: InstantiateMsg,
        execute: ExecuteMsg,
        query: QueryMsg,
    }
}
```

**Step 3: Generate schemas**
Run: `cd contracts && cargo run --bin schema`
Expected: Schemas exported successfully into `schema/` directory for each contract.

---

### Task 4: Genesis bytecode injection
**Files:**
- Modify: [scripts/generate_genesis.go](file:///Users/majedurrahman/Sovereign/scripts/generate_genesis.go)

**Step 1: Implement compilation and wasm JSON wiring**
Update `generate_genesis.go` to:
- Automatically compile all contracts using `cargo build --target wasm32-unknown-unknown --release` if missing.
- Read compiled WASM byte arrays.
- Populate `wasm.codes` with Code 1, 2, 3, 4.
- Populate `wasm.contracts` with pre-defined module address instances (`ConstitutionContractAddr`, `TreasuryContractAddr`, `ReserveFundContractAddr`, `GovernanceContractAddr`) and pre-populate their `contract_state` config values (serialized as raw JSON encoded in base64).
- Populate sequence items for next code/contract ID generation.

**Step 2: Run script to generate new genesis.json**
Run: `go run scripts/generate_genesis.go`
Expected: Invariant verification passes, genesis.json is written with 4 contracts.

---

### Task 5: On-chain Go E2E Integration tests
**Files:**
- Create: [e2e/phase_3_integration_test.go](file:///Users/majedurrahman/Sovereign/e2e/phase_3_integration_test.go)

**Step 1: Write E2E test executing live compiled contract code**
Write a Go integration test using the `chain/app` framework. Initialize the full `App` with the populated genesis, check module balances, verify the constitution contract enforces rules, withdraw from treasury, query and trigger disbursements on reserve fund, and perform the governance replacement.

**Step 2: Run verification**
Run: `go test -v ./e2e/phase_3_integration_test.go`
Expected: PASS

---

### Task 6: Publish Cold Multi-Sig Key Holder Set
**Files:**
- Modify: [doc/adr/adr-007-operational-security.md](file:///Users/majedurrahman/Sovereign/doc/adr/adr-007-operational-security.md)

**Step 1: Define key holders**
Define a 3-of-5 cold multi-sig key holder configuration with 5 public keys and their disaster recovery/emergency pause operational duties.

---

## Verification Plan

### Automated Tests
- `cargo test --workspace` to run the Rust suite.
- `go run scripts/generate_genesis.go` to verify genesis invariants.
- `go test -v ./e2e/...` to execute Go module & phase integration tests.

### Manual Verification
- Compile contracts manually: `cd contracts && cargo build --target wasm32-unknown-unknown --release` and check the target binaries size.
