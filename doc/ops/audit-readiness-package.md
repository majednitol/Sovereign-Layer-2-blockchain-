# Audit Readiness Package — Sovereign L1

This document provides a comprehensive technical overview of the Sovereign L1 blockchain system to guide external security auditors.

---

## 1. System Overview & Architecture

Sovereign L1 is a hybrid Cosmos SDK / EVM Layer-1 blockchain built with standard Go-based modules and CosmWasm smart contracts.

```
       ┌─────────────────────────────────────────────────────────┐
       │                      EVM Client /                       │
       │                   MetaMask / viem                       │
       └────────────────────────────┬────────────────────────────┘
                                    │ (JSON-RPC)
                                    ▼
       ┌─────────────────────────────────────────────────────────┐
       │                   Cosmos SDK Node                       │
       │  ┌───────────────────────┐   ┌───────────────────────┐  │
       │  │  CosmWasm VM          │   │  EVM Module (x/vm)    │  │
       │  │  - constitution.wasm  │   │                       │  │
       │  │  - governance.wasm    │   │  Precompiles:         │  │
       │  │  - treasury.wasm      │◄──┼─ - x/oracle (0x0801)  │  │
       │  │  - reserve_fund.wasm  │   │  - x/milestone (0x0802)  │
       │  └───────────────────────┘   └───────────────────────┘  │
       │  ┌───────────────────────────────────────────────────┐  │
       │  │  Custom Cosmos SDK Modules                        │  │
       │  │  - x/validator (fixed 30 slots)                   │  │
       │  │  - x/certification (liveness/degraded state)       │  │
       │  │  - x/oracle (median absolute deviation)           │  │
       │  │  - x/milestone (automatic state machine)          │  │
       │  │  - x/settlement (Ed25519 witness verification)    │  │
       │  │  - x/bridge (supply caps, relayer management)     │  │
       │  │  - x/governance-ext (CosmWasm validation)        │  │
       │  └───────────────────────────────────────────────────┘  │
       └────────────────────────────▲────────────────────────────┘
                                    │
                                    │ (NATS JetStream Events)
                                    ▼
       ┌─────────────────────────────────────────────────────────┐
       │                  Off-Chain CQRS Backend                 │
       │  - Ingests blocks/transactions (advisory lock)           │
       │  - Denormalizes data into TimescaleDB                   │
       │  - Serves read APIs via gRPC / gRPC-Web                  │
       └─────────────────────────────────────────────────────────┘
```

Detailed architectural designs are documented in `doc/adr/`:
- [ADR-001: Validator Cardinality](file:///Users/majedurrahman/Sovereign/doc/adr/adr-001-validator-cardinality.md)
- [ADR-002: Certification Liveness](file:///Users/majedurrahman/Sovereign/doc/adr/adr-002-certification-liveness.md)
- [ADR-003: Oracle Commit-Reveal](file:///Users/majedurrahman/Sovereign/doc/adr/adr-003-oracle-commit-reveal.md)
- [ADR-004: Bridge Security Model](file:///Users/majedurrahman/Sovereign/doc/adr/adr-004-bridge-security-model.md)
- [ADR-005: CQRS/NATS Topology](file:///Users/majedurrahman/Sovereign/doc/adr/adr-005-cqrs-nats-topology.md)
- [ADR-006: CosmWasm Governance](file:///Users/majedurrahman/Sovereign/doc/adr/adr-006-cosmwasm-governance.md)
- [ADR-007: Operational Security](file:///Users/majedurrahman/Sovereign/doc/adr/adr-007-operational-security.md)

---

## 2. Compilation, Build, and Testing

All modules can be built and tested locally.

### 2.1 Rust CosmWasm Contracts
```bash
# Compilation
cd contracts
cargo build --target wasm32-unknown-unknown --release

# Run unit and integration tests
cargo test --workspace
```

### 2.2 Solidity Bridge Contracts
```bash
cd bridge
# Install dependencies if needed via forge install
forge build
forge test -vvv --fuzz-runs 50000
```

### 2.3 Go Cosmos SDK Chain
```bash
cd chain
go build ./...
go test ./... -v
```

### 2.4 End-to-End Tests
Ensure Docker is running before executing the e2e test suite:
```bash
cd e2e
go test ./... -v
```

---

## 3. Remediation Log: Internal Security Auditing (Phase A)

We performed an extensive pre-audit hardening pass (Phase A) resolving critical and high-severity issues. The remediation details are summarized below:

### A1: Governance Authorization Bypass (CRITICAL)
- **Problem**: Governance `SubmitProposal` had no sender authentication and immediately executed arbitrary messages, allowing anyone to withdraw funds from the Treasury or rewrite the Constitution.
- **Fix**: Implemented a complete proposal lifecycle: proposal submission, vote approval, and final execution. Enforced that only authorized proposers/voters can participate, and added replay checks.

### A2: Fake Reentrancy Guards in Treasury & Reserve Fund (CRITICAL)
- **Problem**: Reentrancy locks were toggled `true` and then `false` within the same execution transaction before CosmWasm actually dispatched the submessages, rendering the guard useless.
- **Fix**: Wrapped the outgoing transfer as a submessage with a success reply callback (`SubMsg::reply_on_success`). The lock remains active during execution and is only cleared in the `reply()` entry point on success.

### A3: Mock Go Tests Bypassing Rust Logic (HIGH)
- **Problem**: E2E tests for Phase 3 used Go mocks rather than executing the actual compiled WASM files, masking critical integration issues.
- **Fix**: Rewrote the tests to deploy and interact with the real compiled `.wasm` files, ensuring actual state-transition paths are checked.

### A4: Genesis WASM Bytecode Drift (HIGH)
- **Problem**: Embedded WASM bytecodes in `genesis.json` did not match the compiled binaries in `artifacts/`.
- **Fix**: Rebuilt all binaries, generated correct genesis parameters, and added a verification flag (`--verify`) to ensure bytecode matches checksum registries.

### A5: Missing schema.rs Binaries (MEDIUM)
- **Problem**: Drifting JSON schema definitions could occur since no generation binaries existed.
- **Fix**: Added `src/bin/schema.rs` targets across all 4 contracts.

### A6: Stale Multisig Policy and Placeholders (MEDIUM)
- **Problem**: Conflicting multisig documentation and placeholder public keys.
- **Fix**: Updated operational security guidelines to mandate a 5-of-7 cold multisig with clear instructions for offline key tracking.

### A7: Governance Bootstrap Window (LOW)
- **Problem**: Post-instantiation setter permitted a race window before governance address was configured.
- **Fix**: Passed `governance_address` directly into the instantiation message, making it immutable from inception.
