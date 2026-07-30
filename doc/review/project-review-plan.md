# Sovereign L1 — Comprehensive Project Review Plan

> **Skills applied:** `@brainstorming` · `@api-design-principles` · `@frontend-design` · `@debugging-strategies`
> **Date:** 2026-07-30
> **Reviewer:** Cline (AI Software Engineer)
> **Repository:** `/Users/majedurrahman/Sovereign`

---

## 1. Executive Summary

This document outlines a structured, whole-project review plan for the **Sovereign L1 Blockchain** — a Cosmos SDK-based Layer-1 with EVM compatibility (via `cosmos/evm`), a BSC bridge, CosmWasm smart contracts, a CQRS off-chain backend, a custom Next.js explorer, and full Kubernetes infrastructure.

The review applies four complementary skill methodologies to systematically evaluate every layer of the stack, from on-chain consensus logic to frontend UX, from API design to infrastructure hardening.

---

## 2. Project Architecture Overview (Discovered)

```
┌─────────────────────────────────────────────────────────────────────────┐
│                        SOVEREIGN L1 BLOCKCHAIN                           │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│  CHAIN (Go / Cosmos SDK v0.50+ / CometBFT / cosmos/evm v0.7.0)        │
│  ├── 7 Custom Modules: x/validator, x/certification, x/oracle,        │
│  │   x/milestone, x/settlement, x/governance-ext, x/bridge             │
│  ├── EVM: x/vm, x/feemarket, x/erc20, x/precisebank                   │
│  ├── CosmWasm (wasmd) with Constitution contract integration          │
│  ├── IBC (ibc-go v11) with transfer + ERC-20 middleware               │
│  ├── Custom precompiles (Oracle, Milestone) — gated behind env var    │
│  └── Staking compatibility layer (equal-slot voting power)            │
│                                                                         │
│  CONTRACTS (Rust / CosmWasm)                                           │
│  ├── Constitution, Governance, Treasury, Reserve-Fund                 │
│  ├── cw-counter, cw20, cw721, cw1155                                  │
│  └── JSON schemas for Celatone integration                             │
│                                                                         │
│  BRIDGE (Solidity / Foundry)                                           │
│  ├── LockBox.sol — nonce bitmap, quorum unlock, circuit breaker       │
│  ├── MockERC20.sol for testing                                         │
│  └── DeployLockBox.s.sol                                               │
│                                                                         │
│  RELAYER (Go)                                                          │
│  ├── BSC watcher (ethclient log filtering)                             │
│  ├── Cosmos watcher (CometBFT BlockResults polling)                   │
│  ├── Sig aggregator (quorum voting via NATS)                           │
│  ├── Submitter (deterministic promotion ladder)                        │
│  ├── Horcrux threshold signing                                         │
│  └ NATS EventBus + RelayerDB (PostgreSQL/in-memory)                   │
│                                                                         │
│  ORACLE (Go)                                                           │
│  ├── Feed fetcher (multi-source, median)                               │
│  ├── Commit-reveal client (gRPC tx broadcast)                          │
│  ├── HSM key management (PKCS#11 / Thales crypto11)                    │
│  └── Prometheus metrics (:9200)                                        │
│                                                                         │
│  BACKEND (Go / CQRS)                                                   │
│  ├── Ingestion (CometBFT RPC → Write DB → NATS, advisory lock)        │
│  ├── Projection (NATS → Read DB hypertables → NATS stream)            │
│  ├── API (gRPC + grpc-gateway REST, Read DB queries, Redis cache)     │
│  └── Faucet (token distribution via Envoy /faucet)                    │
│                                                                         │
│  EXPLORER (Next.js 16 / React 19 / TypeScript)                        │
│  ├── 60+ routes: blocks, txs, validators, bridge, oracle, EVM,       │
│  │   governance, milestones, settlements, IBC, contracts, analytics   │
│  ├── TanStack Query + Recharts + Tailwind + RainbowKit + Wagmi        │
│  └── API client with Zod schema validation                             │
│                                                                         │
│  EXPLORER-API (Go)                                                     │
│  ├── gRPC + REST gateway (5554 lines)                                  │
│  ├── Read DB + Redis cache + NATS streaming                            │
│  ├── Etherscan-compatible API endpoint                                 │
│  └── Contract verification (EVM + CosmWasm)                            │
│                                                                         │
│  EXPLORER-INDEXER (Go)                                                 │
│  └── Indexes chain data into explorer schema on shared Read DB        │
│                                                                         │
│  FRONTEND (Next.js / dApp)                                             │
│  ├── Dashboard with real gRPC analytics data                           │
│  ├── Bridge UI with MetaMask integration                               │
│  ├── Governance page with CosmJS tx signing                            │
│  └── gRPC-Web client with auto-reconnect                               │
│                                                                         │
│  INFRA (K8s / Terraform / Docker)                                      │
│  ├── Multi-region K8s (chain cluster + backend cluster)               │
│  ├── Envoy gateway, cert-manager mTLS, WireGuard VPN                   │
│  ├── NATS 3-node JetStream cluster (R=3)                               │
│  ├── PostgreSQL ×4 (Write, Read, Relayer, Blockscout)                  │
│  ├── TimescaleDB hypertables + continuous aggregates                   │
│  ├── Horcrux 2-of-3 threshold signing                                  │
│  ├── HashiCorp Vault for secrets                                       │
│  ├── Prometheus + Grafana monitoring                                   │
│  └── Helm charts + Terraform IaC                                       │
│                                                                         │
│  E2E TESTS (Go)                                                        │
│  ├── 25+ test files covering phases 0–10                               │
│  ├── Bridge lifecycle, chaos scenarios, EVM coexistence                │
│  └── BSC testnet integration                                           │
│                                                                         │
│  DOCUMENTATION                                                         │
│  ├── 13 ADRs (validator, certification, oracle, bridge, CQRS, etc.)   │
│  ├── Phase audit reports, gap analyses                                 │
│  ├── Runbooks (key rotation, backup drills, incident response)        │
│  └── Mainnet launch runbook + community checklist                      │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## 3. Review Methodology — Skill Application Matrix

### 3.1 Brainstorming Skill → Architecture & Design Review

**Objective:** Validate architectural decisions, identify design gaps, and surface hidden assumptions.

| Review Area | Key Questions | Artifacts to Examine |
|---|---|---|
| Module decomposition | Are the 7 custom modules correctly bounded? Any cross-module coupling violations? | `chain/x/*/keeper.go`, `chain/x/*/types.go` |
| CQRS boundary | Is the write/read split enforced at DB user level? Any direct chain gRPC from `module/api`? | `backend/module/api/main.go`, `db/read_schema/`, `db/write_schema/` |
| EVM coexistence | Does CosmWasm + EVM ante handler routing work correctly? Any state corruption risks? | `chain/app/app.go` (ante handler), `e2e/evm_cosmwasm_test.go` |
| Bridge threat model | Is the nonce model collision-resistant? Is quorum enforced on both sides? | `bridge/src/LockBox.sol`, `chain/x/bridge/keeper.go` |
| NATS topology | Are account isolations correct? Can a relayer read chain events? | `nats/nats.conf`, `backend/module/ingestion/main.go` |
| Genesis invariants | Are supply math invariants actually enforced? | `scripts/generate_genesis.go`, `chain/x/bridge/keeper.go` |
| Precompile policy | Are custom precompiles safely gated? What happens if enabled? | `chain/app/app.go` (ENABLE_CUSTOM_PRECOMPILES), `chain/x/vm/precompiles/` |

### 3.2 API Design Principles Skill → API & Interface Review

**Objective:** Evaluate REST/gRPC API design for consistency, pagination, error handling, versioning, and developer experience.

| Review Area | Key Questions | Artifacts to Examine |
|---|---|---|
| Proto definitions | Are message names consistent? Are field types correct? Is pagination cursor-based? | `proto/`, `chain/api/explorer/v1/` |
| REST gateway | Are REST paths consistent? Is CORS correctly configured? Rate limiting? | `explorer-api/main.go`, `infra/envoy.yaml` |
| gRPC streaming | Are server-streaming RPCs correctly implemented? Slow consumer eviction? | `explorer-api/main.go` (StreamLatestBlocks, StreamConsensusRound) |
| Error codes | Are gRPC status codes used correctly? NotFound vs Internal vs InvalidArgument? | `explorer-api/main.go`, `backend/module/api/main.go` |
| API versioning | Is `/api/rest/v1/` consistently used? Breaking change strategy? | All API entry points |
| Etherscan compat | Is the Etherscan-compatible API correctly structured? | `explorer-api/main.go` (handleEtherscan) |
| Explorer API vs Backend API | Are there overlapping APIs? Which is canonical? | `explorer-api/main.go` vs `backend/module/api/main.go` |

### 3.3 Frontend Design Skill → Explorer & dApp Review

**Objective:** Evaluate UX patterns, component architecture, data fetching, error states, accessibility, and design system consistency.

| Review Area | Key Questions | Artifacts to Examine |
|---|---|---|
| Design system | Is there a consistent component library? Are Card, Badge, DataTable reusable? | `explorer/components/ui/`, `explorer/app/globals.css` |
| Data fetching | Is TanStack Query used correctly? Cache keys? Refetch intervals? Error boundaries? | All `explorer/app/**/page.tsx` files |
| Schema validation | Are Zod schemas matching API responses? Are errors caught? | `explorer/lib/schemas.ts`, `explorer/lib/api-client.ts` |
| Loading states | Are skeleton/spinner states present? Is `isLoading` used consistently? | All page components |
| Error states | What happens when API is down? Are there fallback UIs? | All page components |
| Responsive design | Do pages work on mobile? Are grid breakpoints correct? | All page components, `tailwind.config.js` |
| Wallet integration | Is Keplr + MetaMask coexistence handled? Address conversion? | `explorer/components/MultiWalletButton.tsx`, `explorer/store/wallet.ts` |
| EVM pages | Are EVM explorer pages functional? Blockscout integration? | `explorer/app/evm/` routes |
| Search | Does global search work for blocks, txs, addresses, contracts? | `explorer/components/SearchBar.tsx`, `explorer/app/search/` |
| Accessibility | Are ARIA labels present? Color contrast sufficient? Keyboard navigation? | All components |
| Performance | Are images optimized? Is code splitting used? Bundle size? | `next.config.ts`, `package.json` |

### 3.4 Debugging Strategies Skill → Security & Reliability Review

**Objective:** Systematically hunt for bugs, vulnerabilities, race conditions, and edge cases using adversarial testing reasoning.

| Review Area | Key Questions | Artifacts to Examine |
|---|---|---|
| Bridge replay protection | Can a nonce be processed twice? Cross-chain replay? | `bridge/src/LockBox.sol`, `chain/x/bridge/keeper.go` |
| Supply cap enforcement | Can minting exceed cap? Is the check truly atomic? | `chain/x/bridge/keeper.go` (ProcessBridgeIn) |
| Signature verification | ECDSA malleability? High-S? Zero address recovery? Duplicate sigs? | `bridge/src/LockBox.sol` (recoverSigner), `chain/x/bridge/keeper.go` |
| Oracle commit-reveal | Can a committed hash be revealed with different value? Late reveal? | `chain/x/oracle/keeper.go` |
| Degraded mode determinism | Is degraded mode chain-state-driven? All nodes agree? | `chain/x/certification/keeper.go` |
| Ingestion crash recovery | Does advisory lock prevent split-brain? Gap filling correct? | `backend/module/ingestion/main.go` |
| NATS outage back-fill | Are events delivered in order after recovery? Any data loss? | `backend/module/ingestion/main.go` (runBackfillWorker) |
| Relayer quorum | Can a single relayer double-submit? Promotion ladder deterministic? | `relayer/submitter.go`, `relayer/sig_aggregator.go` |
| EVM precision | Is 6-decimal ↔ 18-decimal conversion lossless? | `chain/x/erc20/`, `chain/x/precisebank/` |
| Genesis config | Are devnet/testnet/mainnet params correctly separated? | `chain/genesis*.json`, `scripts/generate_genesis.go` |
| x/authz blocking | Are all 6 message types actually blocked? Bypass possible? | `chain/app/app.go` (setAnteHandler) |
| Module init order | Is feemarket → vm → erc20 enforced in all hooks? | `chain/app/app.go` (SetOrder*) |

---

## 4. Detailed Review Checklist

### Phase A — Chain Core (Cosmos SDK + EVM + CosmWasm)

#### A.1 App Wiring (`chain/app/app.go`)
- [ ] Verify `cosmos/evm` version pinning matches ADR-008
- [ ] Verify module init order: `precisebank → feemarket → vm → erc20` in all of InitGenesis, BeginBlocker, EndBlocker, PreBlocker
- [ ] Verify ante handler routes `MsgEthereumTx` through EVM decorators and all other msgs through SDK decorators
- [ ] Verify `x/authz` blocked list contains all 6 message types (MsgBridgeIn, MsgBridgeOut, MsgSubmitOracleCommit, MsgRevealOracleReport, MsgSettlement, MsgEthereumTx)
- [ ] Verify custom precompiles are gated behind `ENABLE_CUSTOM_PRECOMPILES` env var
- [ ] Verify `EVMChainID = 7777` matches ADR-009 and chainlist.org registration
- [ ] Verify `sdk.DefaultPowerReduction = 10^6` for staking
- [ ] Verify IBC transfer stack includes ERC-20 middleware
- [ ] Verify CosmWasm `CodeUploadAccess` is set to `Nobody` (governance-only upload)
- [ ] Verify Constitution contract address is wired to `x/governance-ext`
- [ ] Verify EVM mempool configuration doesn't panic in test mode
- [ ] Verify `NoCheckProposalTxVerifier` doesn't skip necessary validation

#### A.2 Custom Modules (`chain/x/`)
- [ ] **x/validator**: Verify slot-based cardinality (30 slots), ejection logic, tombstone on ejection, signing info initialization
- [ ] **x/validator**: Verify non-stake-weighted voting power formula is concrete (not "spec-defined")
- [ ] **x/certification**: Verify degraded mode is chain-state-driven (not per-validator local counter)
- [ ] **x/certification**: Verify bootstrapping window uses `actual_block_count` as denominator
- [ ] **x/certification**: Verify slashing for systematic withholding (M consecutive missed)
- [ ] **x/oracle**: Verify commit-reveal hash computation (`sha256(operator || feed_id || round_id || value || nonce)`)
- [ ] **x/oracle**: Verify late reveal rejection, slash on commit-without-reveal
- [ ] **x/oracle**: Verify staleness state machine (fresh → stale → stale-blocked → recovery)
- [ ] **x/oracle**: Verify insufficient round handling (min_operator_commits_per_round)
- [ ] **x/milestone**: Verify state machine (pending → stale-blocked → achieved/expired)
- [ ] **x/milestone**: Verify deadline clock pause during stale-blocked, resume on recovery
- [ ] **x/milestone**: Verify direct `stale-blocked → achieved` path on feed recovery
- [ ] **x/milestone**: Verify EndBlocker benchmark < 50ms for 500 milestones
- [ ] **x/settlement**: Verify Ed25519 signature with domain separator (`sha256(chain_id || module_path || message_type || payload_hash)`)
- [ ] **x/settlement**: Verify timestamp tolerance ±30s
- [ ] **x/settlement**: Verify witness registry is governance-managed
- [ ] **x/governance-ext**: Verify Constitution check via WasmKeeper.Execute
- [ ] **x/governance-ext**: Verify `MsgMigrateContracts` bypasses Constitution check with 7-day time-lock
- [ ] **x/governance-ext**: Verify gas limit bounds [100k–2M] enforcement
- [ ] **x/bridge**: Verify atomic check+mint (supply cap check before mint)
- [ ] **x/bridge**: Verify nonce bitmap replay protection (out-of-order supported)
- [ ] **x/bridge**: Verify `cosmos_minted` accounting is correct on both BridgeIn and BridgeOut
- [ ] **x/bridge**: Verify supply invariant and nonce bitmap invariant are registered
- [ ] **x/bridge**: Verify signature verification strips V byte correctly for CometBFT
- [ ] **x/bridge**: Verify quorum threshold check counts unique relayers only

#### A.3 Simulation & Testing
- [ ] Verify all 7 custom modules have `WeightedOperations` registered
- [ ] Verify governance-gated operations use `SimulateGovernanceProposal` wrapper
- [ ] Verify `TestAppSimulation` runs 5000 blocks without panic
- [ ] Verify `TestCosmWasmEVMCoexistence` covers all 5 scenarios
- [ ] Verify `go test -race ./...` passes with zero data races

#### A.4 Genesis
- [ ] Verify `scripts/generate_genesis.go` enforces all supply invariants
- [ ] Verify devnet genesis has devnet params (2-min voting, N=2 confirmations)
- [ ] Verify testnet genesis has testnet params (1-hour voting, N=7/N=20)
- [ ] Verify mainnet genesis has mainnet params (7-day voting, N=15/N=50)
- [ ] Verify no devnet/testnet params leaked into mainnet genesis
- [ ] Verify bridge escrow = C (BSC circulating supply)
- [ ] Verify `Cosmos_allocation = S - C`

### Phase B — CosmWasm Contracts (`contracts/`)

- [ ] **Constitution**: Verify `EmergencyPause` blocks `ExecuteMsg` only, not `QueryMsg`
- [ ] **Constitution**: Verify rules storage and retrieval
- [ ] **Governance**: Verify Constitution compliance check call
- [ ] **Governance**: Verify on-chain audit log
- [ ] **Treasury**: Verify threshold multi-sig disbursements
- [ ] **Treasury**: Verify cold multi-sig pause capability
- [ ] **Reserve-Fund**: Verify milestone-gated disbursements
- [ ] **Reserve-Fund**: Verify circuit-breaker functionality
- [ ] Verify all 4 contracts have JSON schemas generated
- [ ] Verify integration tests in `contracts/governance/tests/`
- [ ] Verify `cargo clippy -- -D warnings` passes with zero warnings
- [ ] Verify `cargo audit` passes with zero vulnerabilities
- [ ] Verify all contracts compile to `wasm32-unknown-unknown`

### Phase C — Bridge (Solidity + Go Relayer)

#### C.1 LockBox.sol
- [ ] Verify nonce generation is collision-resistant (`keccak256(msg.sender || nonce || block.number || amount || block.timestamp)`)
- [ ] Verify bitmap nonce registry prevents replay
- [ ] Verify quorum threshold enforcement with unique signature counting
- [ ] Verify ECDSA malleability protection (non-zero recovered address check)
- [ ] Verify rate limiting per block (`maxUnlockPerBlock`)
- [ ] Verify circuit breaker: EOA pause-only, Gnosis Safe unpause
- [ ] Verify `rotateCircuitBreaker` is Gnosis Safe restricted
- [ ] Verify `totalLocked >= totalReleasedToUsers` invariant
- [ ] Verify Foundry fuzz tests run with 50000+ runs
- [ ] Verify Foundry invariant tests pass
- [ ] Verify deploy script `DeployLockBox.s.sol` is correct
- [ ] Check for reentrancy in `lock()` and `unlock()`
- [ ] Check for ERC20 return value handling (some tokens don't return bool)
- [ ] Check for fee-on-transfer token compatibility
- [ ] Check for ERC777 hook reentrancy

#### C.2 Relayer (`relayer/`)
- [ ] Verify BSC watcher implements tiered confirmation depth (N=15 standard, N=50 large)
- [ ] Verify Cosmos watcher polls BlockResults for `MsgBridgeOut` events
- [ ] Verify sig_aggregator deduplicates votes
- [ ] Verify submitter promotion ladder is deterministic
- [ ] Verify relayer DB checkpoint recovery on restart
- [ ] Verify NATS subject naming is consistent (`bridge.bsc.locked.*`, `bridge.sig.vote.*`)
- [ ] Verify Horcrux signer integration
- [ ] Verify Prometheus metrics on :9300
- [ ] Verify private key is not logged or leaked
- [ ] Verify BSC chain ID is queried dynamically (not hardcoded)
- [ ] Check `broadcastCosmosMsgBridgeIn` for proper tx signing (current code has mock signature)
- [ ] Check `broadcastBSCContractUnlock` for hardcoded chain ID 97 (BSC testnet)

### Phase D — Oracle (`oracle/`)

- [ ] Verify feed fetcher queries ≥2 sources in parallel and takes median
- [ ] Verify fail-safe: skips round if <2 sources respond
- [ ] Verify commit-reveal two-step flow (commit hash → wait for block → reveal value)
- [ ] Verify HSM PKCS#11 key management abstraction
- [ ] Verify operator address is derived from HSM (not hardcoded)
- [ ] Verify BSC outage → skip round logic (no commit submitted)
- [ ] Verify retry with exponential backoff on gRPC failures
- [ ] Verify Prometheus metrics on :9200
- [ ] Verify multi-feed concurrent workers
- [ ] Verify active feeds are queried from chain (with config fallback)
- [ ] Verify nonce uses `crypto/rand` (256-bit entropy)
- [ ] Check for goroutine leaks in feed worker loop

### Phase E — Backend CQRS (`backend/`)

#### E.1 Ingestion
- [ ] Verify PostgreSQL advisory lock singleton (ID 41892305)
- [ ] Verify lock acquisition timeout (10s, non-zero exit)
- [ ] Verify startup reconciliation from `MAX(block_height)`
- [ ] Verify NATS reconnect back-fill (`nats_published = false` → publish in order)
- [ ] Verify large-payload ref-pointer (>750KB threshold)
- [ ] Verify Prometheus metric: `advisory_lock_held`
- [ ] Verify event parsing handles begin/tx/end block events
- [ ] Verify only successful transactions (code=0) are ingested
- [ ] Check for HTTP client timeout on CometBFT RPC calls

#### E.2 Projection
- [ ] Verify NATS `account:chain` subscription
- [ ] Verify writes to TimescaleDB hypertables (block_stats, oracle_submissions, validator_signatures, bridge_events)
- [ ] Verify KV projections (settlement_by_id, milestone_status, bridge_pending)
- [ ] Verify publish to NATS `account:stream` after Read DB write
- [ ] Verify aggregate projections are singleton-safe (advisory lock or TimescaleDB continuous aggregates)

#### E.3 API
- [ ] Verify gRPC server starts and registers all services
- [ ] Verify grpc-gateway REST proxy
- [ ] Verify cursor-based pagination on all list RPCs
- [ ] Verify `x-wallet-address` metadata header support
- [ ] Verify slow consumer eviction (channel buffer 64, `ResourceExhausted`)
- [ ] Verify PgBouncer connection pooling
- [ ] Verify Redis caching with appropriate TTLs
- [ ] Verify analytics RPCs query TimescaleDB continuous aggregates
- [ ] Verify `StreamChainStats` uses NATS `account:stream`
- [ ] Check for SQL injection in dynamic LIKE queries (ListTxsByAddress)

#### E.4 Faucet
- [ ] Verify rate limiting per address
- [ ] Verify faucet is disabled on mainnet
- [ ] Verify Envoy route `/faucet` is correctly configured

### Phase F — Explorer Frontend (`explorer/`)

#### F.1 Architecture & Design System
- [ ] Verify component library consistency (Card, Badge, DataTable, Tabs, Sparkline)
- [ ] Verify Tailwind config has consistent color palette and spacing
- [ ] Verify dark theme is default and consistent
- [ ] Verify font loading (JetBrains Mono, Space Grotesk)
- [ ] Check for hardcoded colors that should use Tailwind tokens

#### F.2 Data Layer
- [ ] Verify `apiClient.get()` uses Zod schema validation on all calls
- [ ] Verify TanStack Query cache keys are unique and consistent
- [ ] Verify refetch intervals are appropriate (5s for stats, 10s for charts)
- [ ] Verify error handling in `apiClient` (network errors, 404, 500)
- [ ] Verify loading states are shown consistently
- [ ] Verify empty states are handled (no blocks, no txs)
- [ ] Check for missing error boundaries

#### F.3 Page Coverage (60+ routes)
- [ ] Verify home page: stats summary, TPS chart, recent blocks/txs
- [ ] Verify blocks list + detail pages
- [ ] Verify transactions list + detail pages
- [ ] Verify validators list + detail pages (with slot grid, signing heatmap)
- [ ] Verify bridge pages (deposit, withdraw, history, tx detail)
- [ ] Verify oracle pages (feeds, operators, rounds)
- [ ] Verify governance pages (proposals list, detail, submit)
- [ ] Verify milestone pages
- [ ] Verify settlement pages
- [ ] Verify EVM pages (blocks, txs, contracts, tokens, NFTs, verify)
- [ ] Verify contract pages (CosmWasm: list, detail, execute, query, verified)
- [ ] Verify IBC pages (channels, assets)
- [ ] Verify analytics page (TPS, block time, bridge volume, validator uptime)
- [ ] Verify search page
- [ ] Verify faucet page
- [ ] Verify staking page
- [ ] Verify address detail page (with send/stake tabs)
- [ ] Verify status/network/params pages

#### F.4 Wallet Integration
- [ ] Verify Keplr connection (window.keplr.enable)
- [ ] Verify MetaMask connection (wallet_addEthereumChain)
- [ ] Verify dual wallet coexistence (both active simultaneously)
- [ ] Verify address conversion (bech32 ↔ hex)
- [ ] Verify wallet state management (Zustand store)

#### F.5 Performance & Accessibility
- [ ] Verify `next.config.ts` has appropriate image optimization
- [ ] Check for unnecessary client-side rendering ("use client" overuse)
- [ ] Verify keyboard navigation works on interactive elements
- [ ] Verify color contrast meets WCAG AA
- [ ] Verify ARIA labels on icon-only buttons
- [ ] Check for layout shift (CLS) on data loading

### Phase G — Explorer API (`explorer-api/`)

- [ ] Verify gRPC service registration
- [ ] Verify REST gateway with CORS headers
- [ ] Verify security headers (CSP, X-Content-Type-Options, X-Frame-Options, X-XSS-Protection)
- [ ] Verify IP rate limiting
- [ ] Verify Redis caching with TTL
- [ ] Verify NATS streaming (StreamLatestBlocks, StreamConsensusRound)
- [ ] Verify Etherscan-compatible API
- [ ] Verify EVM contract verification endpoint
- [ ] Verify CosmWasm contract verification endpoint
- [ ] Verify all custom REST handlers (portfolio, faucet, mempool, stats, gas-price, etc.)
- [ ] Verify analytics endpoints (TPS, block-time, validator-uptime, bridge-volume)
- [ ] Check for SQL injection vulnerabilities in dynamic queries
- [ ] Check for proper error responses (not generic 500s)
- [ ] Verify `resolveAddresses` handles all bech32 prefixes (cosmos, sovereign, sov)

### Phase H — Explorer Indexer (`explorer-indexer/`)

- [ ] Verify indexer reads from correct CometBFT endpoint
- [ ] Verify indexer writes to `explorer` schema on shared Read DB
- [ ] Verify indexer handles chain reorgs
- [ ] Verify indexer tracks last indexed height
- [ ] Verify indexer doesn't conflict with backend projection

### Phase I — Frontend dApp (`frontend/`)

- [ ] Verify dashboard uses real gRPC analytics RPCs
- [ ] Verify bridge form submits real MetaMask BSC lock tx
- [ ] Verify governance page submits real CosmJS proposals
- [ ] Verify gRPC-Web client with auto-reconnect
- [ ] Verify `x-wallet-address` header injection
- [ ] Verify supply API routes (`/api/total-supply`, `/api/circulating-supply`)
- [ ] Verify `DataStatusIndicator` tracks bridge live/offline state

### Phase J — Infrastructure (`infra/`)

#### J.1 Kubernetes
- [ ] Verify validator node deployment with correct config
- [ ] Verify sentry node deployment
- [ ] Verify Horcrux signer deployment (2-of-3)
- [ ] Verify backend deployment (ingestion, projection, api)
- [ ] Verify NATS StatefulSet with anti-affinity (3 replicas)
- [ ] Verify database StatefulSets (Write, Read, Relayer, Blockscout)
- [ ] Verify Envoy gateway deployment with HPA
- [ ] Verify grpc-gateway separate deployment with HPA
- [ ] Verify network policies (validator/sentry/DB isolation)
- [ ] Verify cert-manager mTLS configuration
- [ ] Verify multi-region database (synchronous standby)
- [ ] Verify multi-region network (WireGuard VPN)
- [ ] Verify faucet deployment
- [ ] Verify explorer deployment
- [ ] Verify frontend deployment
- [ ] Verify monitoring deployment (Prometheus + Grafana)

#### J.2 Terraform
- [ ] Verify VPC configuration
- [ ] Verify EKS cluster configuration
- [ ] Verify RDS configuration
- [ ] Verify S3 bucket configuration
- [ ] Verify outputs are correct

#### J.3 Helm
- [ ] Verify Chart.yaml is correctly versioned
- [ ] Verify values.yaml has all required parameters
- [ ] Verify templates render correctly

#### J.4 Monitoring
- [ ] Verify Prometheus scrape config targets all services
- [ ] Verify alert rules cover: ChainHalted, OracleStaleness, AdvisoryLockLost, NATSOutage, BridgeStuck
- [ ] Verify Grafana dashboards have all required panels
- [ ] Verify Grafana provisioning (datasources + dashboards)

#### J.5 Docker
- [ ] Verify Dockerfile multi-stage builds
- [ ] Verify base image digests are pinned
- [ ] Verify docker-compose.yml has all services
- [ ] Verify NATS 3-node cluster config
- [ ] Verify Envoy config has CORS, rate limiting, mTLS

### Phase K — E2E Tests (`e2e/`)

- [ ] Verify 14-step primary bridge E2E scenario
- [ ] Verify NATS outage chaos scenario
- [ ] Verify relayer promotion ladder failover
- [ ] Verify circuit-breaker pause < 60s
- [ ] Verify oracle staleness → deadline paused → recovery
- [ ] Verify ingestion crash → advisory lock → reconciliation
- [ ] Verify x/authz grant rejection for MsgBridgeIn
- [ ] Verify BSC testnet integration
- [ ] Verify EVM + CosmWasm coexistence test
- [ ] Verify all phase verification tests (0–10)
- [ ] Check for test mocks that hide production bugs
- [ ] Check for tests that don't actually exercise production code paths

### Phase L — Documentation (`doc/`)

- [ ] Verify all 13 ADRs are complete and client-approved
- [ ] Verify runbooks cover all key rotation scenarios
- [ ] Verify audit readiness package is complete
- [ ] Verify mainnet launch runbook is actionable
- [ ] Verify gap analyses are resolved
- [ ] Verify changelog tracking for cosmos/evm is maintained

---

## 5. Critical Risk Areas (Priority-Ordered)

### 🔴 P0 — Blocking / Must Fix Before Mainnet

| # | Risk | Component | Why It Blocks |
|---|---|---|---|
| 1 | External security audit not started | All | Phase 9 requires zero critical/high findings before mainnet |
| 2 | Relayer `broadcastCosmosMsgBridgeIn` uses mock signature | `relayer/cmd/relayer/main.go:431` | Bridge deposits won't actually work on mainnet |
| 3 | Relayer hardcodes BSC chain ID 97 (testnet) | `relayer/cmd/relayer/main.go:476` | BSC mainnet unlock will fail |
| 4 | External validator onboarding not started | Phase 6.6 | Testnet requires ≥5 external validators |
| 5 | 4-week stability run not started | Phase 6.9 | Required before audit engagement |

### 🟡 P1 — High Risk / Should Fix Before Testnet

| # | Risk | Component | Impact |
|---|---|---|---|
| 6 | SQL injection in `ListTxsByAddress` LIKE query | `explorer-api/main.go:723` | Data breach |
| 7 | CORS wildcard `*` in explorer-api REST | `explorer-api/main.go:170` | Cross-origin attacks on mainnet |
| 8 | `StreamConsensusRound` returns mock data | `explorer-api/main.go:929` | Explorer shows fake consensus data |
| 9 | No error boundaries in explorer frontend | `explorer/app/` | White screen on API failure |
| 10 | `fetchBalances` hardcodes `chain-node:1317` | `explorer-api/main.go:819` | Won't work in production K8s |

### 🟢 P2 — Medium Risk / Fix During Hardening

| # | Risk | Component | Impact |
|---|---|---|---|
| 11 | Explorer has 60+ routes — coverage gaps likely | `explorer/app/` | Broken pages |
| 12 | No accessibility audit performed | `explorer/` | WCAG non-compliance |
| 13 | `planned-vs-implemented.md` claims 359/359 complete | Documentation | False confidence — some items are stubs |
| 14 | Two overlapping API services (explorer-api + backend/api) | Architecture | API surface confusion |
| 15 | TimescaleDB adoption decision gate not formally recorded | Phase 5 ADR | Architecture ambiguity |

---

## 6. Review Execution Order

```
Step 1: Build Verification
  ├── go build ./... (all Go modules)
  ├── forge build (bridge)
  ├── cargo build (contracts)
  ├── pnpm build (explorer + frontend)
  └── docker-compose config validation

Step 2: Test Execution
  ├── go test ./... -race (all Go modules)
  ├── forge test -vvv (bridge)
  ├── cargo test (contracts)
  ├── pnpm test (frontend)
  └── go test ./e2e/... (E2E suite)

Step 3: Static Analysis
  ├── go vet ./...
  ├── golangci-lint run
  ├── govulncheck ./...
  ├── forge coverage
  ├── cargo clippy -- -D warnings
  ├── cargo audit
  ├── eslint (explorer + frontend)
  └── tsc --noEmit (TypeScript type check)

Step 4: Security Review (Debugging Strategies)
  ├── Bridge replay attack analysis
  ├── Supply cap enforcement verification
  ├── Signature verification audit
  ├── Oracle commit-reveal integrity
  ├── x/authz bypass attempt
  └── Genesis parameter validation

Step 5: API Design Review
  ├── Proto definition audit
  ├── REST endpoint consistency
  ├── Pagination strategy verification
  ├── Error code correctness
  └── API versioning compliance

Step 6: Frontend Review
  ├── Design system audit
  ├── Page coverage verification (all 60+ routes)
  ├── Data fetching pattern review
  ├── Error/loading state audit
  ├── Accessibility scan
  └── Performance audit (Lighthouse)

Step 7: Infrastructure Review
  ├── K8s manifest validation
  ├── Terraform plan review
  ├── Network policy verification
  ├── Monitoring coverage check
  └── Docker image security scan

Step 8: Documentation Review
  ├── ADR completeness
  ├── Runbook actionability
  └── Gap analysis resolution

Step 9: Synthesis & Report
  ├── Consolidate all findings
  ├── Rank by severity
  ├── Create remediation roadmap
  └── Present final review document
```

---

## 7. Deliverables

| Deliverable | Format | Description |
|---|---|---|
| Build & Test Report | Markdown | Results of all build and test executions |
| Security Findings | Markdown | Vulnerabilities found, severity, remediation |
| API Design Review | Markdown | API consistency, pagination, error handling assessment |
| Frontend Review | Markdown | UX, design system, accessibility, performance assessment |
| Infrastructure Review | Markdown | K8s, Terraform, monitoring, security assessment |
| Remediation Roadmap | Markdown | Prioritized list of fixes with effort estimates |
| Final Review Document | Markdown | Consolidated executive summary with go/no-go recommendation |

---

## 8. Tools & Commands

```bash
# Go
go build ./...
go test ./... -race -count=1
go vet ./...
golangci-lint run
govulncheck ./...

# Solidity
cd bridge && forge build && forge test -vvv --gas-report
cd bridge && forge coverage
cd bridge && forge test --fuzz-runs 50000

# Rust
cd contracts && cargo build --target wasm32-unknown-unknown
cd contracts && cargo clippy -- -D warnings
cd contracts && cargo audit
cd contracts && cargo test

# TypeScript
cd explorer && pnpm install && pnpm build && pnpm lint
cd frontend && pnpm install && pnpm build && pnpm lint

# Docker
docker-compose config --quiet

# K8s
kubectl apply --dry-run=client -f infra/k8s/

# Terraform
cd infra/terraform && terraform validate
```

---

*This review plan is a living document. Findings will be appended as each phase is executed.*