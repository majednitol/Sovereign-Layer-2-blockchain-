<p align="center">
  <img src="https://img.shields.io/badge/Cosmos_SDK-v0.54-blue?style=for-the-badge&logo=cosmos" alt="Cosmos SDK" />
  <img src="https://img.shields.io/badge/EVM-Cosmos_EVM-blueviolet?style=for-the-badge&logo=ethereum" alt="Cosmos EVM" />
  <img src="https://img.shields.io/badge/CosmWasm-v0.70-orange?style=for-the-badge" alt="CosmWasm" />
  <img src="https://img.shields.io/badge/IBC-v11-green?style=for-the-badge" alt="IBC" />
  <img src="https://img.shields.io/badge/Go-1.25-00ADD8?style=for-the-badge&logo=go" alt="Go" />
  <img src="https://img.shields.io/badge/Rust-stable-brown?style=for-the-badge&logo=rust" alt="Rust" />
  <img src="https://img.shields.io/badge/License-Proprietary-red?style=for-the-badge" alt="License" />
</p>

<h1 align="center">🏛️ Sovereign L1 Blockchain</h1>

<p align="center">
  <strong>A production-grade Layer 1 blockchain built on Cosmos SDK with full EVM compatibility, CosmWasm smart contracts, IBC interoperability, and a custom cross-chain bridge to BSC.</strong>
</p>

<p align="center">
  <a href="#-architecture">Architecture</a> •
  <a href="#-features">Features</a> •
  <a href="#-tokenomics">Tokenomics</a> •
  <a href="#-quick-start">Quick Start</a> •
  <a href="#-project-structure">Project Structure</a> •
  <a href="#-custom-modules">Custom Modules</a> •
  <a href="#-smart-contracts">Smart Contracts</a> •
  <a href="#-infrastructure">Infrastructure</a> •
  <a href="#-testing">Testing</a> •
  <a href="#-deployment">Deployment</a>
</p>

---

##  Overview

**Sovereign** is a next-generation Layer 1 blockchain that combines the best of the Cosmos ecosystem with full Ethereum Virtual Machine (EVM) compatibility. It is designed for institutional-grade decentralized governance, cross-chain asset transfers, and programmable smart contracts — all while maintaining the security and performance guarantees expected of a production mainnet.

### Key Highlights

| Feature | Detail |
|---|---|
| **Consensus** | CometBFT (Tendermint) with instant finality |
| **Smart Contracts** | Dual-VM: CosmWasm (Rust/WASM) + EVM (Solidity) |
| **Interoperability** | IBC v11 for Cosmos chains + custom BSC bridge |
| **Token Standard** | Native `CSOV` (6 decimal Cosmos / 18 decimal EVM) |
| **Chain ID** | `sovereign-1` (Cosmos) / `7777` (EVM) |
| **Wallet Support** | Keplr, MetaMask, Cosmostation, OKX Wallet |
| **Validator Model** | Equalized power (1:1 per slot, 30 max validators) |
| **Governance** | On-chain + CosmWasm governance contracts |

---

##  Architecture

```
                         ┌─────────────────────────────────────────────┐
                         │             User-Facing Layer               │
                         │                                             │
                         │  ┌─────────┐  ┌──────────┐  ┌───────────┐  │
                         │  │Frontend │  │ Explorer │  │  Wallets  │  │
                         │  │ :3000   │  │  :3001   │  │ Keplr/MM  │  │
                         │  └────┬────┘  └────┬─────┘  └─────┬─────┘  │
                         └───────┼────────────┼──────────────┼────────┘
                                 │            │              │
                         ┌───────▼────────────▼──────────────▼────────┐
                         │             API Gateway (Envoy)            │
                         │                  :8080                     │
                         └───────┬────────────┬──────────────┬────────┘
                                 │            │              │
              ┌──────────────────┼────────────┼──────────────┼─────────────────┐
              │                  │     Off-Chain Services    │                 │
              │  ┌───────────────▼──┐  ┌──────▼──────┐  ┌───▼──────────────┐  │
              │  │   Backend API   │  │Explorer API │  │   Faucet Svc    │  │
              │  │   :8081/9090    │  │  :8082      │  │   :8000         │  │
              │  └───────┬─────────┘  └──────┬──────┘  └───┬──────────────┘  │
              │          │                   │             │                  │
              │  ┌───────▼───────────────────▼─────────────▼──────────────┐  │
              │  │              NATS JetStream 3-Node Cluster             │  │
              │  │           (Event Sourcing / CQRS Message Bus)          │  │
              │  │                        :4222                           │  │
              │  └──────────────────────────┬─────────────────────────────┘  │
              │                             │                                │
              │  ┌──────────┐  ┌────────────▼──┐  ┌──────────────────────┐  │
              │  │ DB-Write │  │  DB-Read +    │  │    DB-Relayer        │  │
              │  │ (Timescale) │  Standby +   │  │   (PostgreSQL)       │  │
              │  │  :5433   │  │  PgBouncer   │  │    :5435             │  │
              │  └──────────┘  └──────────────┘  └──────────────────────┘  │
              └────────────────────────────────────────────────────────────┘
                                 │
              ┌──────────────────▼─────────────────────────────────────────┐
              │                    Chain Node (chaind)                      │
              │                                                            │
              │   ┌──────────┐  ┌─────────┐  ┌──────────┐  ┌──────────┐  │
              │   │CometBFT  │  │Cosmos   │  │   EVM    │  │ CosmWasm │  │
              │   │RPC :26657│  │gRPC:9090│  │JSON-RPC  │  │ Runtime  │  │
              │   │          │  │REST:1317│  │  :8545   │  │          │  │
              │   └──────────┘  └─────────┘  └──────────┘  └──────────┘  │
              │                                                            │
              │   Custom Modules: validator | certification | oracle |     │
              │   milestone | settlement | bridge | governance-ext |       │
              │   erc20 | feemarket | vm                                   │
              └────────────────────────────────────────────────────────────┘
                                 │
              ┌──────────────────▼─────────────────────────────────────────┐
              │              Cross-Chain Infrastructure                     │
              │                                                            │
              │  ┌────────────┐  ┌─────────────┐  ┌────────────────────┐  │
              │  │  IBC       │  │ BSC Bridge  │  │  Oracle Service   │  │
              │  │  Relayer   │  │ (LockBox)   │  │  (Price Feeds)    │  │
              │  └────────────┘  └─────────────┘  └────────────────────┘  │
              └────────────────────────────────────────────────────────────┘
```

### Architecture Decisions

All major design decisions are documented as Architecture Decision Records (ADRs) in [`doc/adr/`](doc/adr/):

| ADR | Title |
|-----|-------|
| ADR-001 | Validator Cardinality (Equalized Power Model) |
| ADR-002 | Certification Liveness |
| ADR-003 | Oracle Commit-Reveal Scheme |
| ADR-004 | Bridge Security Model |
| ADR-005 | CQRS + NATS Topology |
| ADR-006 | CosmWasm Governance Contracts |
| ADR-007 | Operational Security |
| ADR-008 | Version Pinning |
| ADR-009 | EVM Chain ID Registration (7777) |
| ADR-010 | Fee Market Consolidation |
| ADR-011 | EVM Denomination |
| ADR-012 | Pagination Strategy |
| ADR-013 | gRPC Gateway Deployment |

---

## ✨ Features

### Dual Virtual Machine

- **CosmWasm**: Write smart contracts in Rust, compiled to WASM. Includes governance, treasury, reserve fund, constitution, CW20, CW721, and CW1155 contracts.
- **EVM**: Full Ethereum compatibility via `cosmos/evm`. Deploy Solidity contracts (ERC-20, ERC-721, ERC-1155, ERC-4626). EIP-1559 fee market with dynamic base fees.

### Cross-Chain Bridge (BSC ↔ Sovereign)

- **LockBox Contract**: Solidity-based escrow on BSC with Gnosis Safe multisig governance.
- **Relayer Service**: Go-based watcher + signature aggregator + submitter pipeline.
- **Horcrux Signing**: Threshold signing for bridge transactions (production mode).
- **Circuit Breaker**: Automated pause mechanism for anomaly detection.

### On-Chain Governance

- **Standard SDK Gov**: Proposals, deposits, voting (Yes/No/NoWithVeto/Abstain).
- **Expedited Proposals**: Fast-track governance for urgent changes.
- **CosmWasm Governance Contract**: Extended governance logic in WASM.
- **Constitution Contract**: On-chain constitutional amendments.
- **Governance Extensions Module** (`x/governance-ext`): Custom proposal types.

### Oracle System

- **Commit-Reveal Scheme**: MEV-resistant price feed submission.
- **HSM Support**: Hardware Security Module integration for production oracle operators.
- **Multi-Source Aggregation**: Fetches from configurable price data sources.
- **Slashing**: Operators are slashed for downtime or price manipulation.

### Validator System

- **Equalized Power Model**: Each validator slot receives equal voting power (1,000,000 per slot).
- **30 Validator Slots**: Fixed maximum cardinality.
- **Certification Module**: Validators must be certified before joining the active set.
- **Milestone Tracking**: On-chain milestone verification for project progress.

### Custom Block Explorer

- **Full-Featured Explorer**: 45+ pages covering blocks, transactions, accounts, validators, governance, staking, contracts, tokens, NFTs, bridge, oracle, EVM, IBC, analytics, and more.
- **Real-Time Indexer**: Go-based indexer that subscribes to chain events via NATS and indexes into PostgreSQL.
- **REST + gRPC API**: Dual-protocol explorer API with Redis caching and PgBouncer connection pooling.
- **Multi-Wallet Connect**: Keplr, MetaMask, Cosmostation, OKX Wallet integration.

### Wallet Support

| Wallet | Type | Features |
|--------|------|----------|
| **Keplr** | Cosmos | Native chain suggestion, `eth_secp256k1` signing |
| **MetaMask** | EVM | `wallet_addEthereumChain` auto-config, live account sync |
| **Cosmostation** | Cosmos | `cos_addChain` integration, coinType 60 derivation |
| **OKX Wallet** | Multi-Chain | Cosmos + EVM dual support |

---

## 💰 Tokenomics

### Token: **CSOV**

| Parameter | Value |
|-----------|-------|
| **Total Supply** | 1,000,000,000 CSOV |
| **Base Denomination** | `ucsov` (1 CSOV = 1,000,000 ucsov) |
| **EVM Decimals** | 18 (auto-converted via PreciseBank) |
| **Per-Block Emission** | 1.5 CSOV |
| **Minimum Lifetime** | 5 years (31,536,000 blocks at 5s) |

### Supply Allocation

```
┌──────────────────────────────────────────────────┐
│               Total Supply: 1B CSOV              │
├──────────────────┬───────────────────────────────┤
│ Treasury         │  400,000,000 (40%)            │
│ BSC Escrow       │  300,000,000 (30%)            │
│ Reserve Fund     │  200,000,000 (20%)            │
│ Rewards Bucket   │  100,000,000 (10%)            │
└──────────────────┴───────────────────────────────┘
```

---

##  Quick Start

### Prerequisites

| Tool | Version | Purpose |
|------|---------|---------|
| **Go** | ≥ 1.25 | Chain, backend, oracle, relayer |
| **Rust** | stable | CosmWasm smart contracts |
| **Node.js** | ≥ 20 | Frontend & explorer |
| **pnpm** | ≥ 9 | JavaScript package management |
| **Docker** | latest | Container orchestration |
| **Docker Compose** | v2+ | Multi-service local dev |
| **Buf** | latest | Protobuf code generation |
| **wasm-opt** | latest | WASM optimization (for contracts) |

### Option 1: Docker Compose (Recommended)

Spin up the entire stack with a single command:

```bash
# Clone the repository
git clone https://github.com/sovereign-l1/sovereign.git
cd sovereign

# Copy environment config
cp .env.example .env.dev

# Build and start all services
docker compose up --build -d
```

This starts **15 services**:

| Service | Port | Description |
|---------|------|-------------|
| `chain-node` | 26657, 9090, 1317, 8545, 8546 | Blockchain node (CometBFT + gRPC + REST + EVM) |
| `frontend` | 3000 | User-facing DApp |
| `explorer-frontend` | 3001 | Block explorer |
| `explorer-api` | 8082 | Explorer REST/gRPC API |
| `explorer-indexer` | — | Real-time chain indexer |
| `backend-api` | 8081, 9091 | Off-chain backend service |
| `faucet-service` | 8000 | Devnet token faucet |
| `envoy-gateway` | 8080 | API gateway |
| `nats-0/1/2` | 4222, 8222 | NATS JetStream cluster |
| `db-write` | 5433 | Write database (TimescaleDB) |
| `db-read` | 5434 | Read database + standby replica |
| `db-relayer` | 5435 | Relayer database |
| `pgbouncer-read` | 6432 | Connection pooler |
| `explorer-redis` | 6379 | Explorer cache |
| `vault` | 8200 | HashiCorp Vault (secrets) |

### Option 2: Local Binary Build

```bash
# Build all Go binaries
make build

# Binaries output to ./bin/
ls bin/
# chaind  oracle  ingestion  projection  api  faucet

# Build CosmWasm contracts
make build-cw-assets

# Contract WASMs output to ./artifacts/
ls artifacts/
# constitution.wasm  cw_counter.wasm  cw20_token.wasm
# cw721_nft.wasm  cw1155_multi.wasm  governance.wasm
# reserve_fund.wasm  treasury.wasm  checksums.txt
```

### Option 3: Frontend Development

```bash
# Install JavaScript dependencies
pnpm install

# Start the DApp frontend (dev mode)
pnpm --filter frontend dev       # → http://localhost:3000

# Start the explorer frontend (dev mode)
pnpm --filter explorer dev       # → http://localhost:3001
```

---

##  Project Structure

```
Sovereign/
├── chain/                    #  Blockchain node (Cosmos SDK + EVM + CosmWasm)
│   ├── app/                  #    Application wiring (app.go)
│   ├── cmd/chaind/           #    CLI entrypoint
│   ├── config/               #    Node configuration
│   ├── x/                    #    Custom modules (see below)
│   ├── wasm/                 #    WASM integration
│   ├── genesis.dev.json      #    Development genesis
│   ├── genesis.prod.json     #    Production genesis
│   └── Dockerfile            #    Chain node container
│
├── backend/                  # ⚙️ Off-chain backend (CQRS pattern)
│   └── module/
│       ├── api/              #    REST/gRPC API server
│       ├── faucet/           #    Token faucet service
│       ├── ingestion/        #    Event ingestion (NATS consumer → DB-Write)
│       └── projection/       #    Read model projection (DB-Write → DB-Read)
│
├── oracle/                   #  Oracle price feed service
│   ├── main.go               #    Oracle daemon entrypoint
│   ├── fetcher.go            #    Multi-source price fetcher
│   ├── client.go             #    Chain submission client
│   └── hsm.go                #    Hardware Security Module support
│
├── relayer/                  #  Cross-chain bridge relayer
│   ├── bsc_watcher.go        #    BSC chain event watcher
│   ├── cosmos_watcher.go     #    Sovereign chain event watcher
│   ├── sig_aggregator.go     #    Multi-sig aggregation
│   ├── signer_horcrux.go     #    Threshold signing (production)
│   ├── submitter.go          #    Transaction submitter
│   └── db.go                 #    Relayer state database
│
├── bridge/                   #  BSC bridge contracts (Foundry)
│   ├── src/LockBox.sol       #    LockBox escrow contract
│   ├── test/                 #    Foundry tests
│   └── script/               #    Deployment scripts
│
├── contracts/                #  CosmWasm smart contracts (Rust)
│   ├── constitution/         #    On-chain constitution
│   ├── governance/           #    Extended governance logic
│   ├── treasury/             #    Treasury management
│   ├── reserve-fund/         #    Reserve fund management
│   ├── cw20/                 #    Fungible token standard
│   ├── cw721/                #    NFT standard
│   ├── cw1155/               #    Multi-token standard
│   └── cw-counter/           #    Example counter contract
│
├── evm/                      #  EVM smart contracts (Solidity)
│   └── src/
│       ├── IMilestone.sol     #    Milestone precompile interface
│       ├── IOracle.sol        #    Oracle precompile interface
│       ├── TestERC20.sol      #    ERC-20 test token
│       ├── TestERC721.sol     #    ERC-721 test NFT
│       ├── TestERC1155.sol    #    ERC-1155 multi-token
│       └── TestERC4626.sol    #    ERC-4626 tokenized vault
│
├── explorer/                 #  Block explorer frontend (Next.js)
│   ├── app/                  #    45+ page routes
│   ├── components/           #    Reusable UI components
│   ├── store/                #    Zustand state management
│   └── lib/                  #    Utility libraries
│
├── explorer-api/             #  Explorer REST/gRPC API server (Go)
│   ├── main.go               #    API server with 100+ endpoints
│   └── openapi.json          #    OpenAPI specification
│
├── explorer-indexer/         #  Real-time chain indexer (Go)
│   └── main.go               #    Event indexer (NATS + RPC)
│
├── frontend/                 #  User DApp frontend (Next.js)
│   ├── app/                  #    Dashboard, governance, faucet
│   ├── components/           #    WalletConnect, UI components
│   └── config/wallets.json   #    Multi-wallet configuration
│
├── proto/                    #  Protocol Buffers definitions
│   ├── backend/              #    Backend service protos
│   ├── explorer/             #    Explorer service protos
│   └── relayer/              #    Relayer service protos
│
├── db/                       #  Database schemas
│   ├── write_schema/         #    Write DB (TimescaleDB) migrations
│   ├── read_schema/          #    Read DB projection schema
│   └── relayer_schema/       #    Relayer state schema
│
├── infra/                    #  Infrastructure
│   ├── terraform/            #    Cloud infrastructure as code
│   ├── helm/                 #    Kubernetes Helm charts
│   ├── k8s/                  #    Raw Kubernetes manifests
│   ├── monitoring/           #    Prometheus + Grafana dashboards
│   ├── vault/                #    HashiCorp Vault configuration
│   ├── horcrux/              #    Threshold signing setup
│   ├── nats/                 #    NATS cluster configuration
│   ├── wireguard/            #    VPN mesh networking
│   └── envoy.yaml            #    API gateway configuration
│
├── e2e/                      #  End-to-end tests (34 test files)
│   ├── comprehensive_phases_1_to_7_test.go
│   ├── bsc_testnet_e2e_test.go
│   ├── cosmwasm_counter_test.go
│   ├── explorer_*_test.go    #    Explorer integration tests
│   └── phase_*_test.go       #    Phased verification tests
│
├── scripts/                  #  Operational scripts
│   ├── generate_genesis.go   #    Genesis file generator
│   ├── genesis-ceremony.sh   #    Multi-validator genesis ceremony
│   ├── vault-init.sh         #    Vault initialization
│   └── deploy-testnet.sh     #    Testnet deployment
│
├── doc/                      #  Documentation
│   ├── adr/                  #    Architecture Decision Records
│   ├── governance/           #    Governance documentation
│   ├── mainnet/              #    Mainnet launch checklist
│   ├── ops/                  #    Operations runbooks
│   └── testnet/              #    Testnet documentation
│
├── docker-compose.yml        #  Full-stack local orchestration
├── Makefile                  # ⚡ Build, test, lint, verify commands
├── go.work                   #  Go workspace (7 modules)
├── pnpm-workspace.yaml       #  JS workspace (frontend + explorer)
├── .goreleaser.yaml          #  Release automation
└── .github/workflows/ci.yml  #  CI/CD pipeline
```

---

##  Custom Modules

The chain implements 10 custom Cosmos SDK modules under `chain/x/`:

### `x/validator` — Equalized Validator Set

Manages the validator cardinality model where each validator slot is assigned equal voting power, preventing stake centralization.

### `x/certification` — Validator Certification

Requires validators to pass a certification process before joining the active set. Ensures operational competence and security compliance.

### `x/oracle` — Price Oracle

Commit-reveal price feed system resistant to MEV frontrunning. Operators submit hashed commitments, then reveal prices. Aggregation produces consensus prices used across the chain.

### `x/milestone` — Project Milestones

On-chain milestone tracking and verification. Includes an EVM precompile (`IMilestone.sol`) for cross-VM access.

### `x/settlement` — Settlement Layer

Handles settlement finality for cross-chain transactions and bridge operations.

### `x/bridge` — BSC Cross-Chain Bridge

Native module for the BSC ↔ Sovereign bridge. Manages mint/burn operations, escrow accounting, and bridge state transitions. Has `Minter` and `Burner` permissions.

### `x/governance-ext` — Extended Governance

Custom proposal types and governance extensions beyond the standard Cosmos SDK `x/gov` module.

### `x/erc20` — ERC-20 Bridge (Cosmos EVM)

Automatic conversion between native Cosmos coins and ERC-20 tokens on the EVM side.

### `x/feemarket` — EIP-1559 Fee Market (Cosmos EVM)

Dynamic base fee adjustment following the EIP-1559 model. Configurable elasticity multiplier and base fee change denominator.

### `x/vm` — EVM Runtime (Cosmos EVM)

Full EVM execution environment with custom precompiles for accessing native chain modules from Solidity contracts.

---

## 📜 Smart Contracts

### CosmWasm Contracts (Rust → WASM)

| Contract | Description |
|----------|-------------|
| `constitution` | On-chain constitutional document with amendment voting |
| `governance` | Extended governance logic beyond SDK defaults |
| `treasury` | Treasury fund management with spending proposals |
| `reserve-fund` | Reserve fund with controlled release mechanisms |
| `cw20` | Fungible token standard (CW-20) |
| `cw721` | Non-fungible token standard (CW-721) |
| `cw1155` | Multi-token standard (CW-1155) |
| `cw-counter` | Example/test counter contract |

**Build contracts:**
```bash
make build-cw-assets
```

### EVM Contracts (Solidity)

| Contract | Description |
|----------|-------------|
| `TestERC20.sol` | ERC-20 fungible token |
| `TestERC721.sol` | ERC-721 NFT |
| `TestERC1155.sol` | ERC-1155 multi-token |
| `TestERC4626.sol` | ERC-4626 tokenized vault |
| `IMilestone.sol` | Precompile interface for milestone queries |
| `IOracle.sol` | Precompile interface for oracle price queries |

### Bridge Contracts (Foundry)

| Contract | Description |
|----------|-------------|
| `LockBox.sol` | BSC-side escrow with Gnosis Safe multisig control |
| `MockERC20.sol` | Mock token for testing |

---

## 🗄️ Data Architecture (CQRS)

Sovereign uses a **Command Query Responsibility Segregation (CQRS)** pattern powered by **NATS JetStream**:

```
Chain Events → Ingestion Service → DB-Write (TimescaleDB)
                                        │
                                   NATS JetStream
                                        │
                               Projection Service → DB-Read (TimescaleDB)
                                                         │
                                                    PgBouncer → API / Explorer
```

| Database | Role | Engine | Port |
|----------|------|--------|------|
| `db-write` | Command store (event source of truth) | TimescaleDB (PG 16) | 5433 |
| `db-read` | Query-optimized read model | TimescaleDB (PG 16) | 5434 |
| `db-read-standby` | Hot standby replica | TimescaleDB (PG 16) | — |
| `db-relayer` | Bridge relayer state | PostgreSQL 16 | 5435 |
| `explorer-redis` | Explorer query cache | Redis 7 | 6379 |

---

##  Infrastructure

### Kubernetes (Production)

- **Helm Charts**: `infra/helm/sovereign-chain/` — Full Helm chart for chain node deployment.
- **K8s Manifests**: `infra/k8s/` — Raw Kubernetes resources.
- **Terraform**: `infra/terraform/` — Cloud infrastructure provisioning (AWS/GCP).

### Security

- **HashiCorp Vault**: Secret management for validator keys, oracle credentials, and bridge signing keys.
- **Horcrux**: Threshold signing (Shamir's Secret Sharing) for validator and bridge keys — no single point of compromise.
- **WireGuard**: Encrypted mesh VPN between validator nodes.
- **NATS NKEY Authentication**: Public-key based auth for all NATS connections (no passwords).
- **Cosign**: Binary attestation and signing for release artifacts.
- **SLSA Provenance**: Supply chain security for all build artifacts.

### Monitoring

- **Prometheus**: Metrics collection from all services (`infra/monitoring/prometheus.yml`).
- **Grafana Dashboards**: Pre-configured dashboards (`infra/monitoring/dashboards/`).
- **Alert Rules**: Automated alerting for chain health, validator liveness, and bridge anomalies (`infra/monitoring/alerts.rules.yml`).

### API Gateway

- **Envoy Proxy**: Routes traffic to backend services with load balancing, rate limiting, and TLS termination (`infra/envoy.yaml`).

---

##  Testing

### Test Suite

```bash
# Run all Go tests (chain, oracle, relayer, backend, e2e)
make test

# Run specific test suites
go test -v ./chain/...
go test -v ./oracle/...
go test -v ./relayer/...
go test -v ./backend/...
go test -v ./e2e/...

# Lint all code (Go + Rust + TypeScript)
make lint

# TypeScript type checking
pnpm --filter explorer exec tsc --noEmit

# CosmWasm contract tests
cd contracts && cargo test --workspace
```

### E2E Test Coverage

The `e2e/` directory contains **34 comprehensive test files** covering:

- **Phase 0-10**: Progressive verification of all chain features
- **BSC Bridge**: Cross-chain integration tests with BSC testnet
- **CosmWasm**: Smart contract deployment and execution
- **Explorer**: API endpoint and data integrity validation
- **Infrastructure Gaps**: Coverage for edge cases and failure modes
- **Bech32 Verification**: Address encoding/decoding validation
- **Real Testnet**: Integration tests against live testnet

### Reproducible Builds

```bash
# Verify binary reproducibility (SHA-256 comparison)
make verify-build

# Output: [SUCCESS] Pinned build flags produce 100% identical binary signatures locally.
```

### Load Testing

```bash
# Explorer API load test (k6)
k6 run explorer-api/k6_load_test.js
```

### SimApp Simulation

```bash
# Randomized simulation (500 blocks, 200 txs per block)
go test -v ./chain/app -run TestAppSimulation -NumBlocks=500 -BlockSize=200 -Seed=$RANDOM
```

---

##  Deployment

### CI/CD Pipeline

The GitHub Actions CI pipeline (`.github/workflows/ci.yml`) runs:

1. **Lint**: golangci-lint, cargo clippy, tsc, ESLint
2. **Buf Lint**: Protobuf schema validation
3. **Buf Breaking**: Backward compatibility check against `master`
4. **Test**: Full Go test suite
5. **Build**: All binaries + explorer frontend
6. **SimApp**: Randomized simulation with logged seed
7. **Helm Lint**: Chart validation
8. **Terraform Validate**: Infrastructure config check
9. **Docker Build**: Container image simulation

### Release Process

Releases are automated via **GoReleaser** (`.goreleaser.yaml`):

```bash
# Create a release (handled by CI on tag push)
goreleaser release

# Produces:
# - chaind (linux/darwin, amd64/arm64)
# - oracle (linux/darwin, amd64/arm64)
# - backend-api, backend-ingestion, backend-projection
# - SHA-256 checksums
# - Cosign signatures (keyless)
# - SLSA provenance attestations
```

### Genesis Ceremony

For multi-validator network launch:

```bash
# Generate genesis file from environment variables
go run scripts/generate_genesis.go

# Run the full genesis ceremony (multi-validator)
./scripts/genesis-ceremony.sh

# Verify LP lock
./scripts/verify-lp-lock.sh
```

### Testnet Deployment

```bash
# Deploy to testnet
./scripts/deploy-testnet.sh

# Run testnet launch ceremony
./scripts/testnet_launch_ceremony.sh
```

---

## 🔧 Configuration

### Environment Variables

Copy `.env.example` to `.env.dev` or `.env.prod` and configure:

| Variable | Default | Description |
|----------|---------|-------------|
| `CHAIN_ID` | `sovereign-1` | Cosmos chain identifier |
| `EVM_CHAIN_ID` | `7777` | EVM chain identifier |
| `TOKEN_DENOM` | `ucsov` | Native token denomination |
| `TOTAL_SUPPLY` | `1000000000000000` | Total supply in base units |
| `VALIDATOR_SLOTS` | `30` | Maximum validator count |
| `GOV_VOTING_PERIOD` | `172800s` | Governance voting period |
| `STAKING_UNBONDING_TIME` | `1814400s` | Unbonding period (21 days) |
| `FEEMARKET_BASE_FEE` | `1000000000` | Initial EIP-1559 base fee |

See [`.env.example`](.env.example) for the complete list with detailed comments.

### Wallet Configuration

Wallet connection parameters are defined in [`frontend/config/wallets.json`](frontend/config/wallets.json):

- **Keplr**: Chain suggestion with `eth_secp256k1` key type, coinType 60
- **MetaMask**: `wallet_addEthereumChain` with chain ID `0x1E61` (7777)
- **Cosmostation**: `cos_addChain` with Sovereign chain parameters
- **WalletConnect**: Namespace config for both `eip155` and `cosmos` protocols

---

##  API Endpoints

### Chain Node

| Protocol | Port | URL | Description |
|----------|------|-----|-------------|
| CometBFT RPC | 26657 | `http://localhost:26657` | Tendermint RPC (blocks, txs, consensus) |
| Cosmos gRPC | 9090 | `localhost:9090` | Protobuf-based chain queries |
| Cosmos REST | 1317 | `http://localhost:1317` | LCD (Light Client Daemon) REST API |
| EVM JSON-RPC | 8545 | `http://localhost:8545` | Ethereum-compatible RPC |
| EVM WebSocket | 8546 | `ws://localhost:8546` | Ethereum event subscriptions |

### Application Services

| Service | Port | URL | Description |
|---------|------|-----|-------------|
| Frontend DApp | 3000 | `http://localhost:3000` | User-facing application |
| Block Explorer | 3001 | `http://localhost:3001` | Chain explorer |
| Explorer API | 8082 | `http://localhost:8082` | Explorer REST API |
| Backend API | 8081 | `http://localhost:8081` | Off-chain backend REST |
| Backend gRPC | 9091 | `localhost:9091` | Off-chain backend gRPC |
| Faucet | 8000 | `http://localhost:8000` | Devnet token faucet |
| Envoy Gateway | 8080 | `http://localhost:8080` | API gateway |
| Vault | 8200 | `http://localhost:8200` | Secret management UI |
| NATS Monitor | 8222 | `http://localhost:8222` | NATS JetStream dashboard |

---

## 🏗️ Build Commands

```bash
# Full project build (all Go binaries)
make build

# Build CosmWasm WASM artifacts
make build-cw-assets

# Generate protobuf code
make generate

# Run all tests
make test

# Lint all code (Go + Rust)
make lint

# Verify reproducible builds
make verify-build

# Clean build artifacts
make clean
```

---

##  Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Follow the existing code patterns and ADR conventions
4. Write comprehensive tests (unit + e2e)
5. Ensure `make lint` and `make test` pass
6. Submit a Pull Request

### Code Standards

- **Go**: Follow `golangci-lint` rules (`.golangci.yml`)
- **Rust**: Pass `cargo clippy --workspace -- -D warnings`
- **TypeScript**: ESLint + strict TypeScript (`tsconfig.json`)
- **Protobuf**: Pass `buf lint` and `buf breaking` checks

---

##  License

This project is proprietary software. All rights reserved.

---

<p align="center">
  <strong>Built with ❤️ on Cosmos SDK • Powered by CometBFT Consensus</strong>
</p>
