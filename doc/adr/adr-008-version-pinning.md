# ADR 008: Dependency Version Pinning & Floor Alignment

## Context & Problem Statement
Sovereign L1 incorporates the `cosmos/evm` module (`github.com/cosmos/evm`) to enable EVM compatibility. `cosmos/evm` is built on top of modern Cosmos SDK APIs (specifically store/v2) and imposes strict dependency requirements. Attempting to run the chain on outdated versions (e.g. Cosmos SDK v0.50.x) creates compilation and runtime errors. Therefore, we must establish a dependency version pinning policy.

## Decision & Design

We define the following version floor constraints in `go.mod`:

| Component | Target Version | Rationale |
|-----------|----------------|-----------|
| **`cosmos-sdk`** | `v0.54.x` | Required for store/v2 and core module registry interfaces. |
| **`cometbft`** | `v0.39.x` | Matches SDK v0.54 transaction validation and block execution interfaces. |
| **`ibc-go`** | `v11.x` | Alignment with SDK v0.54 and wasmd updates. |
| **`wasmvm`** | `v2.x` | Integration with CosmWasm virtual machine for SDK v0.54. |
| **`cosmos/evm`** | `v0.6.x` | Target EVM execution module. Pinned to exact tag to mitigate pre-v1 breaking changes. |

### Configuration Rules
1. **Strict Version Lock**: All dependency entries in `go.mod` must be pinned to exact tags.
2. **Zero `replace` Directives**: Use of `replace` directives in `go.mod` for core modules is prohibited to prevent dependency resolution loops and ensure clean build pipelines.
3. **Pre-v1 Integration Risk Mitigation**: The `cosmos/evm` module is pre-v1. To handle breaking changes:
   - Perform weekly reviews of upstream updates.
   - Any breaking updates must be handled in a dedicated migration sprint before upgrading local locks.

## Alternatives Considered
- **Using Ethermint (`github.com/evmos/ethermint`)**: Rejected because it is no longer actively developed and lacks out-of-the-box integration with newer Cosmos SDK store models.
