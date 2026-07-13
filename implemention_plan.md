# Sovereign Layer-1 Blockchain — Implementation Plan
> Cosmos SDK · CometBFT · CosmWasm · BNB Smart Chain Bridge
> Estimated timeline: **6 months (24 weeks)** | Scaffolding → Audited Mainnet
> Uses a parallel multi-team structure (minimum **3 teams, 12–15 engineers total**) — all phases run concurrently where dependencies allow.
> **Team composition must be confirmed in Phase 0 before this timeline is treated as binding.**
> Phase 5 requires its own dedicated sub-team of 5–6 engineers; it cannot share Team C with active E2E test maintenance.

---

## System Architecture

### Service Decomposition — Modular Monolith vs Microservice

```
┌─────────────────────────────────────────────────────────────────────┐
│  MODULAR MONOLITH — Cosmos SDK Chain Binary                         │
│  x/validator · x/certification · x/oracle · x/milestone            │
│  x/settlement · x/governance-ext · x/bridge                        │
│  Shared: ABCI app · IAVL KV store · block processing pipeline       │
│  CQRS: Msg handlers (commands) │ Query handlers (gRPC read-only)    │
└─────────────────────────────────────────────────────────────────────┘
         ↑ CometBFT WebSocket (typed events)
         │
┌────────────────────────────────────────────────────────────────────┐
│  MODULAR MONOLITH — Backend                                         │
│  Module: ingestion  — CometBFT WS → Write DB                       │
│    • singleton StatefulSet; PostgreSQL advisory lock                │
│    • on startup: reconcile from last indexed block before live sub  │
│    • on NATS reconnect: back-fill unpublished events from Write DB  │
│  Module: projection — NATS account:chain (fan-out) → Read DB       │
│    • push subscriber (fan-out); every instance sees every message   │
│    • singleton for aggregate projections (see CQRS boundary notes)  │
│    • also publishes transformed events to NATS account:stream       │
│  Module: api        — gRPC server; reads exclusively from Read DB   │
│    • subscribes to NATS account:stream for server-streaming RPCs    │
│    • grpc-gateway (separate Deployment) handles REST via /api/rest/*│
└────────────────────────────────────────────────────────────────────┘
         │                            │
┌────────────────────┐    ┌───────────────────────────┐
│  Write DB          │    │  Read DB                  │
│  PostgreSQL        │    │  PostgreSQL               │
│  Append-only       │    │  Denormalized projections │
│  event tables      │    │  optimised for queries    │
│                    │    │                           │
│  DB users:         │    │  DB users:                │
│  ingestion_writer  │    │  projection_writer        │
│   (INSERT only     │    │   (INSERT/UPDATE on       │
│    on write schema)│    │    read schema only)      │
│  projection_reader │    │  api_reader               │
│   (SELECT only     │    │   (SELECT only on         │
│    on write schema)│    │    read schema only)      │
└────────────────────┘    └───────────────────────────┘

         NATS JetStream (3-node cluster, R=3, 3 isolated accounts)
         ┌──────────────────────────────────────────────────────┐
         │  account:chain  — ingestion publishes                │
         │                   projection subscribes (fan-out)    │
         │                   projection re-publishes to         │
         │                   account:stream after Read DB write │
         │                                                      │
         │  account:bridge — relayer sig aggregation (isolated) │
         │                   no other service has access        │
         │                                                      │
         │  account:stream — projection publishes               │
         │                   module/api subscribes              │
         │                   (gRPC server-streaming to dApp)    │
         └──────────────────────────────────────────────────────┘
┌─────────────────────────────────────────────────────────────────────┐
│  MICROSERVICE — Relayer (one instance per operator)                  │
│  bsc_watcher · cosmos_watcher · sig_aggregator · submitter          │
│  Horcrux threshold signing; Relayer DB (PostgreSQL, isolated)       │
│  Publishes/subscribes to NATS account:bridge only                   │
└─────────────────────────────────────────────────────────────────────┘
┌─────────────────────────────────────────────────────────────────────┐
│  MICROSERVICE — Oracle Aggregator (one instance per oracle operator) │
│  feed_fetcher · aggregator · commit_submitter · reveal_submitter    │
│  Two-step commit-reveal; HSM PKCS#11 key management                 │
└─────────────────────────────────────────────────────────────────────┘
┌─────────────────────────────────────────────────────────────────────┐
│  Envoy API Gateway                                                   │
│  /api/rest/*    → grpc-gateway (REST, HTTP/1.1) — separate Deployment│
│  /api/grpcweb/* → Backend API gRPC (gRPC-Web transcoding, HTTP/2)   │
│    streaming routes: timeout:0s · idle_timeout:600s                 │
│  /grpc/*        → Chain gRPC (gRPC-Web transcoding, browsers)       │
│  /rpc           → CometBFT JSON-RPC                                 │
│  /ws            → CometBFT WebSocket                                │
│  CORS /api/grpcweb/*: POST+OPTIONS only · Per-identity rate limiting │
│  x-wallet-address gRPC metadata header required for rate limiting   │
│  mTLS on all upstream connections (cert-manager, single CA)         │
│  NOTE: /api/stream WebSocket route REMOVED — all streaming via      │
│  gRPC server-streaming through /api/grpcweb/* only                  │
└─────────────────────────────────────────────────────────────────────┘
         ↑ REST / gRPC-Web server-streaming (auto-reconnect on stream drop)
┌─────────────────────────────────────────────────────────────────────┐
│  Next.js dApp                                                        │
│  CosmJS · Wagmi + RainbowKit                                         │
│  Client SDK: gRPC-Web stubs (auto-generated) + stream auto-reconnect│
└─────────────────────────────────────────────────────────────────────┘
```

---

### CQRS Pattern — Application Points

| Layer | Command Side (Write) | Query Side (Read) |
|---|---|---|
| **Chain modules** | `Msg` handlers mutate IAVL KV state | gRPC `Query` handlers read IAVL KV (read-only) |
| **Off-chain backend** | `ingestion`: CometBFT events → **Write DB** (append-only) | `api`: gRPC/REST from **Read DB** (denormalized projections) |
| **Bridge Relayer** | BSC `Locked` → `MsgBridgeIn`; Cosmos `BridgeOut` → BSC unlock | **Relayer DB** (PostgreSQL, isolated): nonce bitmap state, confirmation tracking, vote records |

**CQRS boundary enforcement — database users and permissions:**

| Module | Database | DB User | Permissions |
|---|---|---|---|
| `module/ingestion` | Write DB | `ingestion_writer` | INSERT on write schema only; no SELECT, no read schema access |
| `module/projection` (replay) | Write DB | `projection_reader` | SELECT on write schema only; no INSERT, no read schema access |
| `module/projection` (write) | Read DB | `projection_writer` | INSERT/UPDATE on read schema only; no write schema access |
| `module/api` | Read DB | `api_reader` | SELECT on read schema only; no INSERT, no write schema access |
| Relayer | Relayer DB | `relayer_rw` | Full access on relayer schema only; no access to Write DB or Read DB |

Enforced at application level (separate config keys per module) and at network level (separate PostgreSQL service DNS names in Kubernetes). No module may be configured with more than its listed permissions.

**Additional CQRS boundary rule — `module/api` chain gRPC prohibition:**
`module/api` **MUST NOT** establish any direct connection to chain gRPC endpoints for serving client queries. All chain-derived data is available via the Read DB projections only. Any direct chain gRPC call from `module/api` is a CQRS read-side boundary violation and must be caught in code review. The only permitted external connections from `module/api` are: Read DB (via `api_reader`) and NATS `account:stream` (for server-streaming subscription). The Envoy route `/grpc/*` → chain gRPC exists for browser/dApp direct chain queries, not for `module/api` internal use.

**Aggregate projections — concurrency safety:**
Aggregate projections (e.g., `bridge_volume_by_period`, `validator_uptime`) are **NOT safe for concurrent fan-out writes** because incremental `total + amount` updates from two simultaneous instances produce double-counting. Aggregate projections are maintained by the `module/projection` pod operating as a **singleton** (StatefulSet with 1 replica + PostgreSQL advisory lock, identical pattern to ingestion). Key-value projections (e.g., `settlement_by_id`, `milestone_status`) are safe for fan-out and may run as multi-replica Deployments.

**NATS event flow (corrected):**
```
ingestion publishes → account:chain
projection consumes account:chain (fan-out) → writes Read DB → publishes account:stream
module/api subscribes account:stream → serves gRPC server-streaming RPCs to dApp
dApp receives events via gRPC server-streaming through Envoy /api/grpcweb/*
```
`module/api` does **not** subscribe to `account:chain`. It only subscribes to `account:stream` for real-time delivery. All data queries go to Read DB only. There is **no** `/api/stream` WebSocket route — the direct NATS→WebSocket bypass was removed to eliminate split API surface and enforce consistent auth/versioning through the gRPC service layer.

**Ingestion crash recovery and NATS outage back-fill:**
- On startup: query Write DB for `MAX(block_height)` → subscribe to CometBFT WebSocket from `MAX(block_height) + 1` to fill any gaps before resuming live subscription
- On NATS reconnection after outage: query Write DB for all events with `nats_published = false` → publish to account:chain in order → mark `nats_published = true`. The Write DB event table includes a `nats_published BOOLEAN DEFAULT false` column; ingestion marks it `true` after successful NATS publish. On reconnect, any rows with `nats_published = false` are back-filled automatically.

---

### NATS JetStream Architecture

3-node JetStream cluster, replication factor R=3. No message acknowledged until written to all 3 nodes.

**Account isolation:**

| Account | Publisher | Subscriber | Credentials |
|---|---|---|---|
| `account:chain` | `module/ingestion` | `module/projection` | Separate NKey per role; stored in Vault |
| `account:bridge` | Relayer instances | Relayer instances | Separate NKey; no other service has these credentials |
| `account:stream` | `module/projection` | `module/api` only (dApp receives via gRPC server-streaming, not direct NATS) | Separate NKey per role |

**Retention policies:**

| Stream | Retention | Rationale |
|---|---|---|
| `chain.blocks` | 30 days, max 10 GB | Re-indexable from CometBFT |
| `chain.events.*` | 90 days, max 50 GB | Projection replay window; Write DB is fallback beyond 90 days |
| `bridge.bsc.*` | 365 days, no size limit | Security-critical; needed for dispute resolution |
| `bridge.sig.*` | 7 days, max 1 GB | Needed only until quorum + submission confirmed |
| `chain.stream.*` | 1 hour, max 500 MB | Real-time delivery only |

**Replay strategy:** events within NATS retention → replay from NATS consumer offset. Events older than retention → replay from Write DB append-only tables (permanent, never deleted). `module/projection` implements both replay paths and selects based on the requested start height vs. stream age.

**Full NATS cluster outage recovery:**
During a complete NATS outage: ingestion continues writing to Write DB with `nats_published = false`. On NATS cluster recovery, ingestion detects reconnection, queries `WHERE nats_published = false ORDER BY block_height`, and publishes in order before resuming live. Projection resumes consuming from its last committed offset; all back-filled events are delivered in order. The Read DB catches up without manual intervention.

---

### Technology Stack

| Technology | Role | Notes |
|---|---|---|
| **NATS JetStream (3-node)** | Event bus; CQRS pipeline; bridge sig coordination | Account isolation per security domain; Write DB back-fill on outage recovery |
| **Envoy Proxy** | API gateway; gRPC-Web transcoding; CORS; rate limiting | Single CA (cert-manager); no Istio (avoids competing CAs) |
| **grpc-gateway** | REST → gRPC translation (HTTP/1.1) | **Separate Kubernetes Deployment** with its own HPA; Envoy proxies `/api/rest/*` to grpc-gateway Service; independent scaling from `module/api` gRPC server |
| **PostgreSQL + TimescaleDB (Write DB)** | Append-only event log; CQRS write side | Two users: `ingestion_writer`, `projection_reader`; WAL archiving + PITR to S3/GCS; hypertable chunked by block_height; automatic chunk compression; continuous aggregates replace aggregate singleton projections. `synchronous_commit = off` (data reconstructable from CometBFT). Decision gate in Phase 5 ADR. |
| **PostgreSQL + TimescaleDB (Read DB)** | Denormalized projections + analytics engine; CQRS read side | Two users: `projection_writer`, `api_reader`; hypertables for `block_stats`, `oracle_submissions`, `validator_signatures`, `bridge_events`; continuous aggregates for TPS, oracle OHLC, validator uptime, bridge volume; `first()`/`last()` hyperfunctions; `percentile_agg` for block time p95/p99. Eliminates aggregate singleton StatefulSet from `module/projection`. Decision gate in Phase 5 ADR. |
| **PostgreSQL (Relayer DB)** | Relayer nonce bitmap, confirmation state | `relayer_rw` user only; isolated from backend |
| **Horcrux** | Validator threshold signing | 2-of-3 across isolated machines; replaces TMKMS entirely |
| **cert-manager** | TLS certificates for all internal and cross-cluster mTLS | Single CA; no Istio — use WireGuard VPN for cross-cluster connectivity |

---

## Environment Configuration Matrix

Every parameter that differs between environments is defined here. This is the single source of truth engineers check before generating a genesis file, deploying a contract, or configuring a service. A value marked **same** is identical across all three environments.

---

### 1. Chain Identifiers

| Parameter | Devnet | Testnet | Mainnet | Why It Differs |
|---|---|---|---|---|
| `chain-id` | `mychain-devnet-1` | `mychain-testnet-1` | `mychain-1` | Ed25519 domain separator in `x/settlement` embeds chain-id; wrong value = mainnet signatures rejected |
| `bech32-prefix` | same | same | same | Must be consistent across all environments |
| `denom` (base) | same | same | same | e.g. `utoken` — set once in ADR |
| `denom` (display) | same | same | same | e.g. `TOKEN` |
| `genesis-time` | `make devnet-up` timestamp | Announced at testnet launch | Announced at mainnet genesis ceremony | Each network has its own genesis block |
| BSC network | BSC Testnet (Chapel) | BSC Testnet (Chapel) | BSC Mainnet | Never use BSC Mainnet for devnet/testnet |
| BSC LockBox address | Devnet deploy | Testnet deploy | Mainnet deploy | Separate contract deployment per network |

---

### 2. Genesis Supply

| Parameter | Devnet | Testnet | Mainnet | Notes |
|---|---|---|---|---|
| Total supply S | `1,000,000,000 utoken` (fake) | `1,000,000,000 utoken` (fake) | Actual S from spec | Devnet/testnet values have no economic meaning |
| BSC circulating C | `100,000,000 utoken` (fake) | `100,000,000 utoken` (fake) | Actual C (live BSC snapshot) | Mainnet C taken from BSC ERC-20 totalSupply at genesis cutoff block |
| Cosmos allocation (S-C) | `900,000,000 utoken` | `900,000,000 utoken` | Actual S-C | Derived; must pass genesis invariant script |
| Bridge escrow | `100,000,000 utoken` | `100,000,000 utoken` | Actual C | Locked in x/bridge module account |
| Rewards bucket balance | Fake value | Fake value | Actual value from spec | Must satisfy 6-month runway invariant |
| Validator set | 1 internal validator | ≥ 5 external validators | Full mainnet validator set | Devnet: single node; testnet/mainnet: independent operators |

---

### 3. Governance Parameters

| Parameter | Devnet | Testnet | Mainnet | Why It Differs |
|---|---|---|---|---|
| `voting_period` | **2 minutes** | **1 hour** | **7 days** | Devnet/testnet: fast iteration; mainnet: full deliberation window |
| `expedited_voting_period` | **1 minute** | **30 minutes** | **24 hours** | Same reasoning |
| `min_deposit` | `1 utoken` | `100 utoken` | Confirm in ADR (suggested 1,000 tokens) | Devnet: no friction; testnet: light friction; mainnet: spam prevention |
| `max_deposit_period` | **5 minutes** | **1 hour** | **14 days** | Mirrors voting_period scale |
| `quorum` | `1%` | `10%` | `33.4%` | Low quorum on devnet/testnet avoids needing many signers |
| `threshold` | `50%` | `50%` | `50%` | Same across all environments |
| `veto_threshold` | `33.4%` | `33.4%` | `33.4%` | Same across all environments |
| CosmWasm gas limit (initial) | `500,000` | `500,000` | `500,000` | Same; governed on-chain post-launch |

> **Critical rule:** never copy governance parameters from devnet or testnet genesis into mainnet genesis. Always use the mainnet column values. This is the most common genesis misconfiguration.

---

### 4. Bridge Parameters

| Parameter | Devnet | Testnet | Mainnet | Why It Differs |
|---|---|---|---|---|
| Standard confirmation depth N | **2 blocks** | **7 blocks** | **15 blocks** | Devnet: speed; testnet: realistic; mainnet: full BSC reorg safety |
| Large transfer confirmation depth | **5 blocks** | **20 blocks** | **50 blocks** | Same reasoning |
| Large transfer threshold | `1,000 utoken` | `10,000 utoken` | Confirm in ADR | Mainnet value based on economic model |
| Bridge rate limit (per block) | No limit | Light limit | Full limit from ADR | Devnet: unrestricted for testing; mainnet: economic safety |
| Relayer quorum threshold T | `1-of-1` | `2-of-3` | Full quorum from ADR | Devnet: single relayer for simplicity |
| Circuit-breaker EOA address | Dev key (hot) | Testnet key (hot) | Mainnet key (cold, HSM) | Mainnet EOA must be a real cold key; never reuse devnet key |
| Gnosis Safe address | N/A (devnet) | Testnet multisig | Mainnet Gnosis Safe | Mainnet Safe requires real hardware key holders |
| BSC block time assumption | 3 s | 3 s | 3 s | Same (BSC block time is fixed) |

---

### 5. Oracle Parameters

| Parameter | Devnet | Testnet | Mainnet | Why It Differs |
|---|---|---|---|---|
| Commit window (blocks) | **5 blocks** | **10 blocks** | **20 blocks** | Devnet: fast rounds; mainnet: time for all operators globally |
| Reveal window (blocks) | **5 blocks** | **10 blocks** | **20 blocks** | Same reasoning |
| Oracle staleness window (blocks) | **20 blocks** | **50 blocks** | **100 blocks** | Mainnet: tolerates temporary operator downtime without stalling milestones |
| Min operator commits per round | `1` | `2` | Confirm in ADR (suggested: majority of set) | Devnet: single oracle operator for testing |
| Oracle operator set | 1 internal | 3 testnet operators | Full set from spec | |

---

### 6. Slashing & Liveness Parameters

| Parameter | Devnet | Testnet | Mainnet | Why It Differs |
|---|---|---|---|---|
| `signed_blocks_window` | **100 blocks** | **1,000 blocks** | **10,000 blocks** | Devnet: fast feedback; mainnet: tolerates brief outage without slash |
| `min_signed_per_window` | `50%` | `50%` | `50%` | Same |
| `slash_fraction_downtime` | `0.01%` | `0.01%` | `0.01%` | Same |
| `slash_fraction_double_sign` | `5%` | `5%` | `5%` | Same |
| `unbonding_time` | **1 minute** | **1 hour** | **21 days** | Devnet: restart validators quickly; mainnet: full security window |
| `evidence_max_age_num_blocks` | `> signed_blocks_window` | `> signed_blocks_window` | `> signed_blocks_window` | Must always exceed signed_blocks_window or evidence expires before slash |
| `x/certification` degraded mode K | `3 consecutive` | `5 consecutive` | `10 consecutive` | Mainnet: more tolerance before entering degraded mode |
| `x/certification` attestation window | **50 blocks** | **200 blocks** | **500 blocks** | Larger window on mainnet for statistical stability |

---

### 7. x/milestone Parameters

| Parameter | Devnet | Testnet | Mainnet | Notes |
|---|---|---|---|---|
| Max active milestones | `50` | `200` | `500` | Mainnet: full load; EndBlocker benchmarked at 500 |
| Milestone deadline tolerance | Short (fast testing) | Medium | Per spec | Exact values from client spec |

---

### 8. Infrastructure

| Parameter | Devnet | Testnet | Mainnet | Notes |
|---|---|---|---|---|
| Chain nodes | 1 (docker-compose) | ≥ 3 full nodes + sentries + 5 validators | ≥ 3 full nodes + sentries + full validator set | |
| Horcrux | No (single node, file-based key) | Yes (2-of-3, testnet signers) | Yes (2-of-3, production HSM) | **Never use file-based key on testnet/mainnet** |
| NATS nodes | 3-node (docker-compose) | 3-node (k8s, single region) | 3-node (k8s, multi-region) | |
| PostgreSQL replication | None (single instance) | Async standby | **Synchronous standby** (`synchronous_commit = on`) | Mainnet: zero data loss requirement |
| PostgreSQL PITR / WAL archiving | Off | Off | **On** (S3/GCS, cross-region) | Enable from mainnet day 0 |
| Pre-upgrade DB snapshot | Not needed | Recommended | **Mandatory** | Taken automatically before every mainnet upgrade |
| Kubernetes | docker-compose | Single-region k8s | **Multi-region k8s** (≥ 2 regions) | |
| Envoy replicas | 1 | 2 | 2 + HPA | |
| Monitoring / alerting | Logs only | Prometheus + Grafana | Prometheus + Grafana + **PagerDuty/OpsGenie** | Production alerting must be live before mainnet genesis |
| CometBFT indexer | `kv` | `kv` | `kv` (full nodes); `null` (validators) | Validators skip indexer for performance |
| Backup drills | Not required | Not required | **Quarterly / semi-annual / annual** | See Backup & DR section |

---

### 9. Envoy / API Gateway

| Parameter | Devnet | Testnet | Mainnet | Notes |
|---|---|---|---|---|
| CORS allowed origin | `*` (wildcard, local) | Testnet dApp domain | Mainnet dApp domain (strict allowlist) | **Never use wildcard on mainnet** |
| TLS | Self-signed (cert-manager local CA) | Let's Encrypt / cert-manager | cert-manager (production CA) | |
| Rate limiting | Off | Light (testing) | Full per-identity limits | |
| mTLS (Envoy → upstreams) | Off | On (testnet certs) | **On** (production certs, cross-cluster WireGuard) | |

---

### 10. Wallet & Client Configuration

| Parameter | Devnet | Testnet | Mainnet | Notes |
|---|---|---|---|---|
| Keplr chain config | Internal only | Published (testnet) | Published (mainnet, chain registry PR) | Do not publish devnet config externally |
| Chain registry entry | No | Testnet entry | **Mainnet entry** (cosmos/chain-registry PR merged) | |
| Faucet | Internal dev faucet | **Public faucet** (via Envoy `/faucet`) | None (tokens have real value) | |
| dApp URL | `localhost` | `testnet.app.example.com` | `app.example.com` | Separate domains per environment |
| Explorer | Internal Ping.pub | Public testnet explorer | Public mainnet explorer | |

---

### 11. Environment Checklist — Before Each Genesis

Run this checklist before generating and distributing any genesis file:

**Devnet:**
- [ ] `chain-id` ends in `-devnet-N`
- [ ] BSC network = Chapel testnet
- [ ] Governance voting_period = 2 minutes
- [ ] Bridge confirmations N = 2
- [ ] No real funds, no real keys

**Testnet:**
- [ ] `chain-id` ends in `-testnet-N`
- [ ] BSC network = Chapel testnet
- [ ] Governance voting_period = 1 hour
- [ ] Bridge confirmations N = 7 / N = 20
- [ ] External validators onboarded
- [ ] Public faucet live

**Mainnet:**
- [ ] `chain-id` = `mychain-1` (final, from ADR)
- [ ] BSC network = BSC Mainnet
- [ ] Governance parameters = mainnet column values (NOT copied from testnet)
- [ ] Bridge confirmations N = 15 / N = 50
- [ ] Genesis supply invariants all pass (run `scripts/generate_genesis.go`)
- [ ] Three independent manual verifications of supply math
- [ ] PITR / WAL archiving active before chain start
- [ ] Monitoring alerts live and tested before chain start
- [ ] Horcrux ceremony complete for all validators
- [ ] Audit: zero unresolved critical / high findings
- [ ] Final audit report published

---

## Implementation Plan — Phase by Phase

---

### Phase 0 — Project Setup & Governance (Week 1)

**Objective:** Establish the engineering foundation before any code is written.

#### 0.1 Legal & Access
- Execute NDA and receive full technical specification
- Confirm: fixed supply S, validator cardinality, partition scheme, economic constants, milestone parameters
- Confirm: token name, denom, bech32 prefix, chain-id, genesis block time
- Confirm: existing BSC ERC-20 contract address and circulating supply **C**
  - Genesis math: `Cosmos_allocation = S - C`, `bridge_escrow = C`, total = S from block 0
- **Confirm team size and composition:** minimum 3 teams + 1 dedicated backend sub-team (12–15 engineers total). If below minimum, timeline must be revised before Phase 1 begins.

#### 0.2 Repository & Toolchain
```
/chain      — Cosmos SDK chain (Go)
/contracts  — CosmWasm suite (Rust)
/bridge     — BSC Solidity contracts (Foundry)
/relayer    — Go multi-sig relayer
/oracle     — Go oracle aggregator (commit-reveal)
/backend    — Off-chain modular monolith
              module/ingestion  (singleton, advisory lock, NATS back-fill)
              module/projection (singleton for aggregates, fan-out for KV)
              module/api        (gRPC server; grpc-gateway is a separate Deployment)
/proto      — Protobuf (single source of truth)
/evm        — EVM layer: custom Solidity contracts deployed on the sovereign chain (post-launch), precompile stubs for x/oracle and x/milestone access from Solidity (Phase 2.9)
/explorer   — Ping.pub (Cosmos chain only); Celatone (CosmWasm contracts — Phase 5.11); Blockscout (EVM — Phase 2.9)
/frontend   — Next.js dApp (with stream auto-reconnect SDK; /dashboard analytics route — Phase 7.3; wagmi/viem for EVM-side interactions)
/infra      — k8s manifests, Terraform, Envoy config (with CORS),
              Horcrux config, NATS cluster, cert-manager, WireGuard VPN
/nats       — NATS cluster config, account credentials, stream definitions
/scripts    — Genesis generation, upgrade handlers, simulation operations
/e2e        — Cross-component E2E test suite
/db         — PostgreSQL migrations: write schema, read schema, relayer schema
              Backup/PITR config, table partitioning definitions
```

- Docker Compose devnet: NATS 3-node cluster, Envoy (with CORS), Write DB + Read DB + Relayer DB PostgreSQL, chain node
- CI: lint → `buf lint` → `buf breaking` → test → build → simapp-simulation (`--NumBlocks=500 --BlockSize=200 --Seed=<random per run>`)
  - Simulation seed randomized per CI run; specific seeds logged for reproducibility
  - `buf breaking` baseline seeded automatically on first backend proto addition
- goreleaser: pinned Go toolchain, deterministic ldflags, CGO disabled
- Secret management: HashiCorp Vault

#### 0.3 Architecture Decision Records (ADRs)

All ADRs written and client-approved before Phase 1 begins. Includes all items from prior versions plus:

- **Validator set:** exact cardinality, partition scheme, non-stake-weighted voting power formula (exact mathematical rule, not "spec-defined"), impact on `x/distribution`, `x/gov`, `x/slashing`, and IBC `HistoricalInfo`
- **`x/certification` liveness:** attestation-distribution bound formula, rolling window size, degraded mode activation (chain-state-driven, not per-validator local counter — see Phase 2.2), bootstrapping behavior (partial denominator for blocks 1 through window_size)
- **`x/certification` vote extension policy:** empty vs. absent vs. malformed behavior; slashing for systematic withholding
- **Oracle commit-reveal:** two-message flow, commit window, reveal window, slash on commit-without-reveal; round behavior during BSC outage (operators skip round; on-chain behavior for rounds below minimum commit count)
- **Oracle staleness state machine:** deadline clock paused during `stale-blocked` (staleness duration does not count against milestone deadline); recovery triggers immediate re-evaluation
- **Bridge threat model:** written as a Phase 0 ADR deliverable, not deferred to Phase 4. Covers nonce model, confirmation depth, quorum threshold, circuit breaker, relayer collusion, replay attacks.
- **Bridge nonce generation:** LockBox generates `nonce = keccak256(msg.sender || block.number || amount || block.timestamp)` — hash-based, **collision-resistant**. User does not supply nonce. Note: `block.timestamp` on BSC is miner-influenceable within ±15 seconds; predictability is not a security property here — only uniqueness is, and the bitmap registry enforces that. Auditors should be briefed on this distinction to avoid false findings.
- **Bridge finality depth:** tiered — N=15 standard, N=50 large transfers (both governance parameters); rationale for BSC reorg risk
- **Bridge nonce model:** bitmap registry, out-of-order confirmation; max in-flight nonces per user (governance parameter)
- **Bridge emergency pause:** circuit-breaker EOA (pause-only, no fund access); Gnosis Safe (pause + unpause); circuit-breaker key compromise runbook (see Phase 8.4)
- **Relayer failure modes:** deterministic promotion ladder for submitter failover (not "any other instance"); quorum timeout; stuck event alert
- **Validator rewards bucket:** total budget, per-block emission, chain lifetime in blocks (explicit value from spec), 6-month alarm threshold
- **Fee market:** `skip-mev/x/feemarket` EIP-1559 dynamic base fees
- **CosmWasm gas limit:** initial 500,000 gas; governance parameter with bounds [100,000, 2,000,000]; bounded enforcement in `MsgUpdateParams` handler; gas limit proposals bypass Constitution check; `MsgMigrateContracts` proposals bypass Constitution check (prevents deadlock)
- **CosmWasm emergency override:** cold multi-sig (5-of-7 hardware keys); key holder set: [define in ADR — e.g., 2 founding team, 2 independent security council, 2 lead validators, 1 legal trustee]; key holders geographically distributed; annual rotation review; `EmergencyPause` blocks `ExecuteMsg` only, never `QueryMsg` — Constitution queries remain available during pause
- **CosmWasm cold multi-sig key holder rotation:** updatable only via a time-locked governance proposal (7-day delay, bypasses Constitution check); holder set defined in ADR and published publicly
- **`x/params` migration:** per-module `MsgUpdateParams` for all custom modules from day one
- **Witness signature scheme:** Ed25519; domain separator = `sha256(chain_id || module_path || message_type || payload_hash)`; timestamp tolerance ±30s of block time
- **`x/authz` blocked message types:** `MsgBridgeIn`, `MsgBridgeOut`, `MsgSubmitOracleCommit`, `MsgRevealOracleReport`, `MsgSettlement`, and `/ethermint.evm.v1.MsgEthereumTx` are registered in the `x/authz` blocked message type list; these six messages cannot be granted to third parties via authz; relayer impersonation via authz grant is prevented at the protocol level
- **NATS account topology:** NKey credentials per account per role; stored in Vault; no cross-account access; credentials rotated annually
- **Kubernetes topology:** chain cluster (full nodes, sentries, validators, Horcrux) and backend cluster (NATS, Envoy, PostgreSQL ×4 — Write DB, Read DB, Relayer DB, Blockscout DB — backend modules) in separate clusters; cross-cluster connectivity via WireGuard VPN + cert-manager mTLS (no Istio)
- **PostgreSQL multi-region replication:** Write DB and Read DB use PostgreSQL streaming replication: primary in Region A, synchronous standby in Region B (replication factor = 2, `synchronous_commit = on`). Failover via Patroni/Stolon. Relayer DB: same pattern. Spec out managed service alternative (Amazon RDS Multi-AZ or equivalent) as fallback.
- **PostgreSQL backup and lifecycle:** daily base backups + continuous WAL archiving to S3/GCS (PITR to any point); Write DB partitioned by block_height (monthly partitions); retention: hot partitions 6 months online, archived to cold storage beyond 6 months; restore drill quarterly
- **TimescaleDB database extension (decision gate before Phase 5.2):** decision to adopt or reject TimescaleDB for Write DB and Read DB must be recorded as a signed ADR before Phase 5.2 implementation begins. Evaluation criteria: (1) OSS licence acceptable (Apache 2.0 for core features); (2) managed service confirmed (Timescale Cloud or self-hosted Helm chart); (3) chunk compression, continuous aggregates, `first()`/`last()` and `percentile_agg()` hyperfunctions verified on devnet; (4) advisory lock, PITR, and `golang-migrate` compatibility verified. **If adopted:** Write DB and Read DB use `timescale/timescaledb:latest-pg16`; module/projection writes to hypertables (`block_stats`, `oracle_submissions`, `validator_signatures`, `bridge_events`) instead of maintaining Go aggregates; aggregate singleton StatefulSet eliminated. **If rejected:** plain PostgreSQL retained for all DBs; module/projection singleton StatefulSet maintained; manual cold-storage archive pipeline required for Write DB. The fallback (rejection) path is documented in Phase 5.2 and 5.10.
- **Envoy CORS policy:** `Access-Control-Allow-Origin: <dApp domain>` on all `/api/*` routes; strict origin allowlist (not wildcard); `Access-Control-Allow-Methods` and `Access-Control-Allow-Headers` specified per route
- **Key rotation procedures:** oracle operators, witnesses, relayers (see Phase 8.4)
- **gRPC server-streaming client reconnection:** dApp SDK implements exponential backoff auto-reconnect (initial 1s, max 30s, jitter) for all server-streaming connections; Envoy restart drops streams; SDK must handle gracefully
- **Dependency version pinning (required before Phase 1):** exact versions of `cosmos-sdk`, `ibc-go`, `cometbft`, `wasmd`/`wasmvm`, and `ibc-go` must be pinned in `go.mod` and recorded in this ADR before Phase 1 begins. Version selection must account for: ABCI++ / vote extension availability (requires CometBFT ≥ 0.38 / SDK ≥ 0.50), `MsgEditValidator` consensus key rotation (SDK ≥ 0.50), gRPC error code compatibility. Example baseline: `cosmos-sdk v0.50.x`, `ibc-go v8.x`, `cometbft v0.38.x`, `wasmvm v2.x`. **Do not start Phase 1 without version confirmation.**
- **IBC and supply invariant:** the `cosmos_minted_via_bridge + bsc_circulating = S` invariant is defined over total Cosmos-side token supply (all chains), not just the local chain balance. Tokens transferred out via IBC remain in the `x/bank` total supply and count as "Cosmos-side". The invariant is correctly interpreted as `bsc_circulating + total_cosmos_chain_supply_excluding_bridge_escrow = S`. IBC-out tokens do not create a new category — they remain within total supply. Document this explicitly so auditors do not flag IBC transfers as invariant violations.
- **API pagination strategy:** all list RPCs (`QueryBridgeActivity`, `QuerySettlements`, etc.) must use cursor-based (keyset) pagination. Offset pagination is prohibited for event-stream data because inserts between pages cause duplicates or gaps. Cursor encodes `(block_height, event_index)` as a base64 opaque value; clients never parse it. Proto contract: `PageRequest { bytes cursor = 1; uint32 limit = 2; }` / `PageResponse { bytes next_cursor = 1; bool has_more = 2; }`.
- **grpc-gateway deployment:** grpc-gateway is deployed as a **separate Kubernetes Deployment** (not a sidecar in the same pod as the gRPC server). This enables independent HPA scaling and prevents REST traffic spikes from causing OOM on the gRPC server pod. Envoy upstream cluster `backend_grpc_gateway` points to the grpc-gateway Service; `backend_grpc` points to the module/api gRPC Service.
- **`x-wallet-address` gRPC metadata header:** all authenticated dApp requests must include `x-wallet-address: <bech32_address>` as a gRPC metadata header. Envoy reads this header for per-wallet rate limiting (descriptor `wallet_address`). The client SDK sets this header on every call after wallet connection. This is the rate-limiting mechanism only — it is not an authentication proof. Document explicitly: lying about the address only rate-limits the liar's own quota.
- **Streaming slow consumer policy:** `module/api` maintains an in-process event channel per active server-streaming client. Channel buffer: 64 events. If a client's channel is full (slow consumer), the stream is terminated with gRPC status `ResourceExhausted`. The client SDK's reconnect loop handles this transparently. Buffer size and eviction policy are governance-accessible configuration values (not hardcoded). Document this as expected API behavior in the SDK README.
- **Advisory lock acquisition timeout:** ingestion instances that fail to acquire the PostgreSQL advisory lock within 10 seconds must exit with a non-zero exit code (triggering k8s crash backoff, not an infinite spin). Add Prometheus alert: "advisory lock not held by any ingestion instance for > 30 seconds."
- **EVM integration approach:** Ethermint (`github.com/evmos/ethermint`, latest stable release pinned in `go.mod`). Selected over `cosmos/evm` (not production-ready at time of writing) and Berachain Polaris (insufficient community tooling). Ethermint is battle-tested on Evmos mainnet since 2021, has the broadest auditor familiarity, and produces a standard ETH JSON-RPC endpoint that Blockscout, MetaMask, ethers.js, and wagmi all consume without modification.
- **EVM chain ID:** a unique integer registered on [chainlist.org](https://chainlist.org) before mainnet genesis. Set at genesis in `x/evm` params — **never changeable after genesis without a hard fork**. Confirm value in ADR before Phase 1. This is a separate identifier from the Cosmos `chain-id` string.
- **Fee market consolidation:** `skip-mev/x/feemarket` (planned in Phase 1.1) is **replaced** by Ethermint's `x/feemarket` module (`github.com/evmos/ethermint/x/feemarket`). Ethermint's `x/feemarket` implements EIP-1559 for both Cosmos txs and EVM txs through a unified fee model. Running both fee market modules simultaneously creates ante handler conflicts and duplicate base fee calculations. Use Ethermint's version only. Initial genesis params: `min_gas_price`, `base_fee`, `elasticity_multiplier = 2`, `enable_height = 0` (active from block 1).
- **EVM denomination and decimal precision:** native bank denom is `utoken` (6 decimals — standard Cosmos). For EVM compatibility, configure `x/evm` `BaseDenom = "atoken"` (atto-token, 18 decimals — matches Ethereum convention). `sdk.DefaultPowerReduction = 10^6` remains unchanged (staking operates in `utoken`). `x/erc20` handles conversion between `utoken` and the EVM `atoken` representation. MetaMask must be configured with 18 decimal places and `atoken` as the currency unit. Document this conversion explicitly for users: "1 TOKEN = 10^18 atoken displayed in MetaMask". This ADR decision is **immutable after genesis**.
- **Ante handler ordering — EVM and CosmWasm coexistence:** `app.go` must use Ethermint's `evmante.NewAnteHandler` which routes `MsgEthereumTx` through the EVM ante decorator chain and all other messages through the standard Cosmos ante chain. The CosmWasm message types (`MsgInstantiateContract`, `MsgExecuteContract`, `MsgStoreCode`) must NOT pass through EVM decorators. Ethermint's handler detects tx type automatically. The critical risk is incorrect module initialization order in `SetOrderBeginBlockers` / `SetOrderEndBlockers`: `x/feemarket` must initialize before `x/evm`; `x/evm` must initialize before `x/erc20`; `wasmd` has no ordering constraint with EVM modules but must be present and correctly initialized. **The CosmWasm + EVM coexistence integration test (Phase 2.9) is mandatory before testnet launch.**
- **Dual address format:** the same secp256k1 private key generates both a bech32 address (`cosmos1...` format, using chain's bech32 prefix) for Cosmos transactions and a hex address (`0x...`) for EVM transactions. These address the same account. Key derivation path: `m/44'/60'/0'/0/0` (Ethereum BIP-44 path) — this is required so MetaMask can derive the correct address. Keplr and Leap support both derivation paths (60' for EVM chains, 118' for Cosmos chains); configure chain registry entry to specify path 60'.
- **`x/authz` blocked list — EVM extension:** add `/ethermint.evm.v1.MsgEthereumTx` to the `x/authz` blocked message type list alongside the existing blocked messages. An authz grant for `MsgEthereumTx` would allow a third party to submit arbitrary EVM transactions on behalf of an account — a critical security hole for smart contract interactions.
- **Precompile policy:** no custom precompiles at mainnet launch. Standard Ethermint precompiles only (`0x0000000000000000000000000000000000000001` through `0x0000000000000000000000000000000000000009` — standard Ethereum precompiles). Post-launch enhancement (governance-approved upgrade): `x/oracle` precompile at `0x0000000000000000000000000000000000000801` exposing current oracle price to Solidity; `x/milestone` precompile at `0x0000000000000000000000000000000000000802` exposing milestone status. Precompile stubs added to `/evm` at Phase 2.9 with a `// TODO: post-launch` marker; not compiled into the mainnet binary until after audit.
- **Blockscout deployment:** self-hosted Blockscout (`blockscout/blockscout:latest`) reads the Ethermint JSON-RPC endpoint (`/evm-rpc`) and WebSocket (`/evm-ws`). Requires its own PostgreSQL instance (**Blockscout DB** — a fourth database, isolated, no sharing with Write DB / Read DB / Relayer DB). Deploy alongside Ping.pub and Celatone. Expose at `/blockscout` via Envoy. Blockscout covers EVM txs, Solidity contracts, ERC-20 transfers, and EVM account state. It does NOT cover Cosmos-native txs (those remain in Ping.pub).

**Deliverable:** All ADRs written, client-approved, signed. Team confirmed. Dependency versions pinned in `go.mod`. Timeline adjusted if below minimum headcount.

---

### Phase 1 — Chain Scaffold & Genesis Configuration (Weeks 1–4)

**Objective:** Single-node devnet with correct fixed genesis supply and CosmWasm runtime.

#### 1.1 Chain Application Scaffold
- Scaffold from `simapp` fork or `ignite chain scaffold`
- Configure: chain-id, bech32 prefix, denom, minimum gas prices
- Wire standard SDK modules: `x/bank`, `x/auth`, `x/staking`, `x/slashing`, `x/distribution`, `x/gov`, `x/upgrade`, `x/feegrant`, `x/authz`
  - **`x/authz` blocked message types:** register `MsgBridgeIn`, `MsgBridgeOut`, `MsgSubmitOracleCommit`, `MsgRevealOracleReport`, `MsgSettlement`, and `/ethermint.evm.v1.MsgEthereumTx` in the blocked message type list during app wiring. These six messages bypass `x/authz` grant checks — they can only be called by the direct signer.
  - **`x/params` not used for custom modules.** All custom modules use per-module `MsgUpdateParams`. `x/params` retained only for standard SDK modules that require it; flagged for migration in the first chain upgrade.
- Wire `ibc-go` baseline; IBC channel creation and packet relay tested on devnet
- Wire **Ethermint `x/feemarket`** EIP-1559 fee market (replaces `skip-mev/x/feemarket` — see Phase 0 ADR on fee market consolidation); initial genesis params: `min_gas_price`, `base_fee`, `elasticity_multiplier = 2`, `enable_height = 0` (active from block 1). **Do not add `skip-mev/x/feemarket` to `go.mod` or `app.go` — Ethermint's version covers both Cosmos and EVM txs through a unified fee model.**
- Wire **Ethermint EVM modules** in `app.go` (required for Phase 2.9):
  - `x/feemarket` — EIP-1559 fee market (before `x/evm` in init order)
  - `x/evm` — EVM execution engine; params: `EVMDenom = "atoken"`, EVM chain ID (from ADR), `EnableCreate = true`, `EnableCall = true`, `AllowUnprotectedTxs = false` (EIP-155 enforced)
  - `x/erc20` — bi-directional native token ↔ ERC-20 wrapping; enables EVM contracts to hold native chain tokens
  - Module init order: `x/feemarket` → `x/evm` → `x/erc20` (enforced in `SetOrderInitGenesis`, `SetOrderBeginBlockers`, `SetOrderEndBlockers`)
- **`x/authz` blocked list — add EVM message type:** add `/ethermint.evm.v1.MsgEthereumTx` alongside the existing blocked types. Without this, a malicious authz grant allows arbitrary EVM contract calls on behalf of an account.
- Wire `x/upgrade` with v1.0.0 no-op handler scaffold (documented no-op; must exist or chain halts at upgrade height)
- **CometBFT indexer configuration:** set `indexer = "kv"` in `config.toml` on all full nodes and ingestion-facing nodes. The ingestion module uses two data paths: (a) live events via CometBFT WebSocket (works with any indexer setting), (b) startup reconciliation via `BlockResults` RPC to fill crash gaps (`BlockResults` reads from the block store — also works regardless of indexer). However, `indexer = "null"` disables `TxSearch` and `BlockSearch` which may be needed for explorer queries and debugging. Set `indexer = "kv"` on full nodes; validators may use `indexer = "null"` for performance since they do not serve public queries.

#### 1.2 `x/staking` Compatibility Scope (blocking prerequisite for Phase 2.1)

`x/validator` overrides staking's validator power assignment. All four downstream modules adjusted:

- **`x/distribution`:** compatibility shim maps slot-based powers to the format `AllocateTokens` expects; or full override if spec requires equal per-validator rewards (per ADR decision)
- **`x/gov`:** governance voting power model chosen in ADR (stake-weighted or slot-weighted); invariant test verifies consistency
- **`x/slashing`:** `ValidatorSigningInfo` created on slot fill, tombstoned on slot ejection; missed-block counter behavior verified throughout lifecycle
- **IBC light clients:** `HistoricalInfo` correctly populated for every block; IBC channel creation and packet relay verified on devnet post-integration

#### 1.3 Genesis Supply Configuration
- Fixed total supply S minted at genesis; no `x/mint`, no inflation
- **Supply math:** `Cosmos_genesis_allocation = S - C`; `bridge_escrow = C`; total = S
- Deterministic genesis generation script: `scripts/generate_genesis.go`
- **Invariant tests in genesis script (must all pass before distributing genesis.json):**
  - `sum(all_genesis_allocations) = S - C`
  - `bridge_escrow = C`
  - `cosmos_minted + bsc_circulating = S`
  - `bucket_balance / per_block_emission ≥ chain_lifetime_blocks` (chain_lifetime from ADR)
  - `bucket_balance / per_block_emission ≥ 6_month_alarm_threshold` on an ongoing basis

**Governance genesis parameters (initial values — all adjustable via governance):**

| Parameter | Initial Value | Rationale |
|---|---|---|
| `voting_period` | 7 days | Long enough for global validator participation; short enough for emergency response |
| `expedited_voting_period` | 24 hours | Available for critical security fixes via `x/gov` expedited proposals |
| `min_deposit` | Confirm in ADR (suggested: 1,000 tokens) | High enough to prevent proposal spam; low enough to be accessible |
| `quorum` | 33.4% | Standard Cosmos SDK minimum; must be met or proposal fails |
| `threshold` | 50% | Simple majority of YES votes (excluding ABSTAIN) |
| `veto_threshold` | 33.4% | NoWithVeto at this fraction burns deposit and rejects proposal |
| `max_deposit_period` | 14 days | Window for deposit accumulation; proposal rejected if not met |
| `unbonding_time` | 21 days | Standard; longer than slashing evidence window (14 days default) |
| `max_validators` | Per ADR cardinality | Exact number from spec; not the Cosmos SDK default of 100 |
| `signed_blocks_window` | 10,000 blocks | Slashing lookback window; adjust based on expected block time |
| `min_signed_per_window` | 50% | Liveness slash threshold; missed > 50% of window → slash |
| `slash_fraction_downtime` | 0.01% | Standard; tombstone not applied for liveness (only for double-sign) |
| `slash_fraction_double_sign` | 5% | Standard; tombstoned immediately on double-sign |
| `evidence_max_age_num_blocks` | Must be > `signed_blocks_window` | Prevents evidence expiry before slashing fires |

All values must be reviewed and confirmed in the Phase 0 ADR against chain economic model. Changes after mainnet require governance votes.

#### 1.4 CosmWasm Runtime
- Integrate `wasmd`; governance-only upload policy
- Pre-compute and reserve module accounts for all four genesis contracts
- On-chain devnet test (not just `cw-multi-test`): deploy canary contract; verify `x/bank` module-account permissions are enforced on real chain
- **CosmWasm + EVM ante handler coexistence (critical — verify in Phase 2.9):** `wasmd` and `x/evm` both register message handlers and interact with `x/bank`. The ante handler chain must route `MsgEthereumTx` through Ethermint's EVM decorators and `MsgExecuteContract` / `MsgInstantiateContract` through the standard Cosmos decorators — **never the reverse**. Ethermint's `evmante.NewAnteHandler` performs this routing automatically via tx type detection. Do NOT add a separate ante handler for CosmWasm or implement custom routing logic. Add `TestCosmWasmEVMCoexistence` integration test in Phase 2.9: submit a CosmWasm `MsgExecuteContract` and a `MsgEthereumTx` in the same block; verify both execute correctly with no interference, no incorrect gas accounting, and no state corruption.

#### 1.5 ABCI++ Hooks Scaffold
- Stub `PrepareProposal` / `ProcessProposal` / `ExtendVote` / `VerifyVoteExtension`
- Bootstrapping note documented in stub: for blocks 1–window_size, use `actual_block_count` as denominator

#### 1.6 Devnet
- `docker-compose`: chain node + Write DB + Read DB + Relayer DB + NATS 3-node cluster + Envoy (with CORS) + WireGuard between backend and chain namespaces
- `make devnet-up` starts clean from scratch
- Verify: gRPC endpoints, REST via grpc-gateway (separate Deployment), gRPC-Web via Envoy, IBC, CosmWasm
- **Database images (if TimescaleDB adopted in Phase 5 ADR):** Write DB and Read DB must use `timescale/timescaledb:latest-pg16` instead of `postgres:16`. Relayer DB stays on `postgres:16`. Update `docker-compose.yml` before Phase 5.2 begins. If TimescaleDB is rejected, keep `postgres:16` for all three.
- **Celatone (if Phase 5.11 adopted):** add `alleslabs/celatone-frontend:latest` to `docker-compose.yml` pointing at devnet CometBFT RPC and Cosmos REST; expose at port 3001; verify contract inspection works against devnet contracts deployed in Phase 3.
- **Ethermint EVM services (Phase 2.9 prerequisite — add before Phase 2.9 begins):**
  - Enable `app.toml` JSON-RPC server: `[json-rpc] enable = true`, `address = "0.0.0.0:8545"`, `ws-address = "0.0.0.0:8546"`, `api = ["eth","net","web3","txpool","debug"]`
  - Envoy routes: `/evm-rpc` → port 8545 (HTTP JSON-RPC); `/evm-ws` → port 8546 (WebSocket JSON-RPC)
  - Verify: `cast block-number --rpc-url http://localhost:80/evm-rpc` returns current block height
- **Blockscout (EVM explorer — add in Phase 2.9):** add `blockscout/blockscout:latest` to `docker-compose.yml`; point at `ETHEREUM_JSONRPC_HTTP_URL = http://chain:8545`; requires its own PostgreSQL (**Blockscout DB**, port 5435 in devnet, isolated from Write/Read/Relayer DBs); expose at port 4000; accessible at `/blockscout` via Envoy.

**Deliverable:** `make devnet-up` → live chain with correct supply, CosmWasm, EVM (JSON-RPC on port 8545, WebSocket on 8546), two-DB CQRS backend, NATS cluster, CORS-enabled Envoy. Blockscout and Celatone accessible locally.

---

### Phase 2 — Custom Cosmos SDK Modules (Weeks 3–12)

**Objective:** All seven custom modules plus E2E test suite; each module registers `WeightedOperations` for simulation.

---

#### 2.1 `x/validator` — Fixed Cardinality, Non-Stake-Weighted Voting (Weeks 3–6)

**Implementation:**
- Wrap `x/staking`; override `EndBlocker` for fixed cardinality slot-based management
- Non-stake-weighted voting power per ADR formula (must be a concrete mathematical rule, not a reference to the spec)
- Validator admission/ejection per partition scheme
- `x/slashing` sync: create `ValidatorSigningInfo` on slot fill, tombstone on ejection
- `x/distribution`, `x/gov`, IBC compatibility per Phase 1.2 decisions

**Simulation `WeightedOperations`:**
- `SimMsgFillValidatorSlot` (weight: 20)
- `SimMsgEjectValidator` (weight: 10)
- `SimGovProposalUpdatePartitionScheme` (weight: 5) — uses the SimGov wrapper (see note below):
  > Governance-gated operations in simulation must be wrapped in a `SimulateGovernanceProposal` helper that: (1) creates the proposal, (2) advances block time past the voting period, (3) votes with quorum, (4) executes the proposal. All governance-gated WeightedOperations across all modules use this wrapper. Without it, governance-gated messages fail immediately and test nothing.

**Tests:** unit (slot allocation, power distribution, admission/ejection, partition boundaries), simulation (50,000 blocks), integration (`x/slashing` sync, `x/distribution` correctness, `x/gov` power model, IBC `HistoricalInfo` format)

---

#### 2.2 `x/certification` — Statistical Finality Attestation (Weeks 5–8)

**Liveness-safe design — critical:**

`x/certification` degraded mode MUST be chain-state-driven, not per-validator local in `ProcessProposal`. Implementation:

- **`EndBlocker`:** after each block commits, store in module KV: `consecutive_rejection_count`. If the proposer's `ProcessProposal` rejected the last block (detectable from BeginBlocker via a flag set by `ProcessProposal`), increment the counter. If counter ≥ K (governance parameter), write `degraded_mode = true` to KV and emit a typed event. This transition is deterministic: all nodes read the same committed state.
- **`ProcessProposal`:** checks `degraded_mode` from the previous block's committed state (not from a local counter). When `degraded_mode = true`, uses the relaxed threshold. All validators see the same committed state → deterministic degraded mode entry.
- **Exit degraded mode:** governance proposal `UpdateCertificationParams` can reset `consecutive_rejection_count` and set `degraded_mode = false` once attestation coverage recovers.

**Other implementation details:**
- `ExtendVote`: validator attaches attestation payload
- `VerifyVoteExtension`: reject malformed; empty and absent count as non-attestation
- **Bootstrapping:** blocks 1 through window_size use `actual_block_count` as denominator (partial window)
- Slashing for systematic withholding: M consecutive missed extensions → slashed by `x/slashing`
- Prometheus metrics: attestation coverage, bound violations, degraded mode active, rejection count

**Simulation `WeightedOperations`:**
- `SimDropValidatorAttestation` (weight: 15)
- `SimRestoreValidatorAttestation` (weight: 15)
- `SimGovProposalUpdateCertificationParams` (weight: 3, uses SimGov wrapper)

**Tests:** unit (bound computation, bootstrapping, degraded mode chain-state transitions, determinism across all nodes), integration (dropout → degraded mode fires → chain does not halt → recovery → exit degraded mode), fuzz (random attestation subsets, window sizes)

---

#### 2.3 `x/oracle` — External Data Feeds with Commit-Reveal (Weeks 5–8)

**Commit-reveal implementation:**
- `MsgCommitOracleHash`: operator submits `sha256(operator_address || feed_id || round_id || value || nonce)`. Mempool shows only hash, not value. Stored in KV.
- `MsgRevealOracleReport`: after commit window closes, operator submits plaintext value + nonce. Chain verifies hash. Late reveals (past reveal window) rejected.
- Slash condition: committed but no reveal within reveal window

**Oracle round boundary under BSC outage (from ADR):**
- If BSC is unavailable between commit and reveal, oracle operators skip the round (submit no commit). This is the safe choice: no commit → no slash.
- If an operator commits during a BSC outage and reveals a stale value: the hash matches (not slashable at protocol level) but the stale value is subject to the on-chain outlier rejection in the aggregation step. If most operators skip the round, the round has fewer than minimum required commits (governance parameter: `min_operator_commits_per_round`); the round is declared `insufficient` and no aggregated value is published. `x/milestone` treats an `insufficient` round identically to a stale round.

**Oracle staleness state machine:**
- `fresh`: aggregated value within N blocks (staleness window)
- `stale`: no fresh aggregation within N blocks; milestone evaluations using this feed are suspended
- **Deadline clock behavior (from ADR):** during `stale-blocked`, the milestone deadline clock is paused. Staleness duration does not count against the milestone deadline. This prevents oracle staleness from being weaponized to expire milestones. When oracle recovers, the deadline clock resumes and the milestone re-evaluates immediately.
- Recovery: fresh aggregation → `x/milestone` immediately re-evaluates all `stale-blocked` milestones using that feed

**Simulation `WeightedOperations`:**
- `SimMsgCommitOracleHash` (weight: 20)
- `SimMsgRevealOracleReport` (weight: 20)
- `SimDropOracleReveal` (weight: 5, slash condition)
- `SimOracleRoundInsufficient` (weight: 3, skipped round)

**Tests:** unit (commit-reveal, hash mismatch, late reveal, staleness, deadline pause/resume, insufficient round handling), integration (full commit-reveal cycle → aggregation → milestone re-evaluation)

---

#### 2.4 `x/milestone` — Vesting & State Transition Gating (Weeks 7–10)

**State machine:**
- `pending`: conditions not yet met; oracle feeds fresh
- `stale-blocked`: one or more feeds stale; deadline clock paused; evaluation suspended
- `achieved`: conditions met; token unlock triggered; terminal
- `expired`: deadline passed (clock not paused) without achieving; terminal; no unlock

**Deadline clock behavior:**
- Clock runs during `pending`, paused during `stale-blocked`
- When feed recovers: `stale-blocked` → `pending` (clock resumes); immediate re-evaluation in the same block
- If conditions are already met when feed recovers: `stale-blocked` → `achieved` directly in the same block (no round-trip through `pending`)

**EndBlocker performance budget:**
- Maximum 500 active milestones (governance parameter); excess queued
- Milestones indexed by oracle feed dependency; stale feeds skipped in O(1)
- Benchmarked: must complete in < 50ms for 500 milestones; failing benchmark blocks merge

**Simulation `WeightedOperations`:**
- `SimMsgCreateMilestone` (weight: 10)
- `SimMsgAchieveMilestone` (weight: 15)
- `SimMilestoneExpiry` (weight: 5)
- `SimMilestoneStaleRecovery` (weight: 8)

**Tests:** unit (all state transitions including stale-blocked→achieved direct path, deadline pause/resume, EndBlocker benchmark), integration (oracle stale → deadline paused → oracle fresh → immediate achievement → vesting release)

---

#### 2.5 `x/settlement` — Institutional-Witness Settlement (Weeks 9–11)

**Implementation:**
```protobuf
message WitnessPayload {
  string witness_id   = 1;
  int64  timestamp    = 2;  // validated: must be within ±30s of block time
  bytes  payload_hash = 3;  // sha256 of settlement data
  bytes  signature    = 4;  // Ed25519 over domain_separator || payload_hash
  // domain_separator = sha256(chain_id || "x/settlement" || "WitnessPayload" || payload_hash)
  // chain_id binding: testnet signature rejected on mainnet even with identical payload
}
```
- Ed25519 signature scheme; domain separator includes chain_id
- Timestamp tolerance: ±30s of block time (governance parameter)
- Witness registry: governance-managed Ed25519 public keys
- Handler: verify domain-separated signature → timestamp validation → transfer → event

**Simulation `WeightedOperations`:**
- `SimMsgSettlement` (weight: 20, mock witness key with correct domain separator)
- `SimMsgInvalidWitnessSettlement` (weight: 5, wrong chain_id → verify rejection)
- `SimMsgExpiredTimestampSettlement` (weight: 3, timestamp outside tolerance → rejection)

**Tests:** unit (Ed25519 verify, domain separator binding, timestamp tolerance, revocation), integration (registry update → settlement → event)

---

#### 2.6 `x/governance-ext` — Extended Governance (Weeks 10–12)

**Implementation:**
- Custom proposal types: `UpdateValidatorSlot`, `UpdateMilestone`, `UpdateOracleOperator`, `UpdateWitnessRegistry`, `UpdateBridgeRelayerSet`
- **`MsgMigrateContracts`** proposal type: bypasses Constitution check (prevents deadlock when Governance contract is being replaced); has a mandatory 7-day time-lock delay before execution; callable only via on-chain governance vote
- **Gas limit governance parameter** for Constitution check: bounds [100,000 — 2,000,000 gas]; `MsgUpdateParams` handler enforces bounds; gas limit update proposals bypass Constitution check
- **Gas limit update proposals and `MsgMigrateContracts` proposals:** these two proposal types explicitly skip the `WasmKeeper.Execute()` Constitution check in the handler. All other proposal types call the Constitution check.
- Failure behavior: if Constitution contract call fails (any reason), entire proposal execution reverts atomically
- All proposal lifecycle events emitted as typed events

**Simulation `WeightedOperations`:**
- `SimGovProposalCustom` (weight: 10, random custom proposal type, uses SimGov wrapper)

**Tests:** unit (gas limit bounds enforcement, Constitution call revert, bypass for gas limit + MsgMigrateContracts proposals, deadlock scenario), integration (full proposal → vote → quorum → execution → state change)

---

#### 2.7 Oracle Aggregator Microservice (`/oracle`) (Weeks 6–9)

**Commit-reveal client:**
- `commit_submitter`: at start of each oracle round, submits `MsgCommitOracleHash`
- `reveal_submitter`: after commit window, submits `MsgRevealOracleReport`; tracks pending commits; if BSC is unavailable at commit time, skips the round (no commit submitted); alerts if a commit was made but reveal deadline is approaching with gRPC unreachable
- **Feed redundancy (mandatory):** each oracle operator configures ≥ 2 independent price source URLs per feed (e.g., primary + fallback from a different data provider). The `feed_fetcher` component: (1) queries all configured sources in parallel, (2) takes the median of received values, (3) if fewer than 2 sources respond, marks the feed as locally-unavailable and skips the round. Single-source oracle operators are prohibited by operator onboarding checklist — if all operators use the same provider, that provider's outage causes an `insufficient` round and stalls milestones.
- HSM key management: PKCS#11 abstraction (`go-crypto11`); HSM product specified in ADR
- Configuration: HSM slot, chain gRPC endpoint (via Envoy), feed URLs (primary + fallback per feed) via Vault, commit/reveal windows must match on-chain params

**Tests:** unit (commit-reveal two-step, BSC outage → skip round logic, reveal timeout detection, PKCS#11 mock), integration (mock feeds + devnet → full pipeline)

---

#### 2.8 Cross-Component E2E Test Suite (`/e2e`) (Weeks 8–12, maintained throughout)

**Primary E2E scenario (14 steps):**
1. BSC user calls `LockBox.lock()` (nonce generated by LockBox via keccak256)
2. BSC watcher waits N confirmations; publishes to NATS `account:bridge`
3. Relayer instances sign; `sig_aggregator` collects quorum via deterministic promotion ladder
4. Designated submitter sends `MsgBridgeIn` to Cosmos chain
5. Chain verifies supply cap (atomic check+mint), mints tokens, emits event
6. `module/ingestion` writes event to Write DB; marks `nats_published = false`; publishes to `account:chain`; marks `nats_published = true`
7. `module/projection` consumes from `account:chain`; writes Read DB projection; publishes to `account:stream`
8. `module/api` subscribes `account:stream`; serves `QueryBridgeActivity` from Read DB
9. Oracle operators submit `MsgCommitOracleHash` (commit phase)
10. Oracle operators submit `MsgRevealOracleReport` (reveal phase); aggregation fires
11. `x/milestone` re-evaluates; oracle value crosses threshold; deadline clock running; milestone → `achieved`
12. Vesting release fires via `x/bank`; bank balance verified
13. Settlement submitted with valid Ed25519 witness payload (correct chain_id domain separator)
14. Governance proposal (`UpdateOracleOperator`, gas limit update); Constitution check; execution

**Failure/chaos E2E scenarios:**
- NATS full outage during bridge event → ingestion writes Write DB with `nats_published=false` → NATS recovers → back-fill publishes in order → projection catches up → Read DB consistent
- Relayer designated submitter goes offline after quorum → promotion ladder activates next relayer → single submitter sends `MsgBridgeIn` → no duplicate submissions
- Circuit-breaker EOA pause: `pause()` called → bridge halts < 60s → Gnosis Safe `unpause()` → bridge resumes
- Oracle staleness: stop oracle → milestone enters `stale-blocked` → deadline clock paused → oracle resumes → clock resumes → milestone achieves
- Ingestion crash: kill pod → k8s restarts → advisory lock acquired → startup reconciliation fills gap → Write DB has no holes
- `x/authz` grant attempt for `MsgBridgeIn`: verify rejection at protocol level

**Deliverable:** All scenarios pass on devnet before testnet launch.

---

#### 2.9 EVM Integration Layer — Ethermint (Weeks 10–14)

> **Parallel workstream:** Phase 2.9 runs concurrently with Phase 4 (Bridge, Weeks 10–15) and the tail of Phase 3 (CosmWasm, Weeks 4–10). These are independent workstreams: a separate EVM-focused sub-team owns Phase 2.9 while the bridge team owns Phase 4. Both sub-teams share the same `app.go` — coordinate daily stand-ups and use feature branches with short-lived merges to avoid wiring conflicts. Phase 2.9 must be complete and accepted before Phase 6 (testnet launch, Week 13).

**Objective:** Wire Ethermint's `x/evm`, `x/feemarket`, and `x/erc20` into `app.go`; prove CosmWasm and EVM can coexist in the same binary without ante handler conflicts; stand up Blockscout; verify MetaMask connects and a Solidity contract deploys.

**Why this is a dedicated phase:** Slide 6 of the Engagement Review explicitly calls this out as "trial-and-error" and the single biggest technical risk of the engagement. The ante handler ordering, module initialization sequence, and CosmWasm coexistence are the areas most likely to produce subtle, hard-to-diagnose state corruption. Isolating EVM integration into a spike with mandatory acceptance criteria before proceeding to testnet is the correct risk-mitigation strategy.

---

##### Step 1 — `go.mod` and module wiring

```bash
# Pin Ethermint version in go.mod (use latest stable tag)
go get github.com/evmos/ethermint@<pinned-version>
go get github.com/evmos/ethermint/x/evm@<pinned-version>
go get github.com/evmos/ethermint/x/feemarket@<pinned-version>
go get github.com/evmos/ethermint/x/erc20@<pinned-version>
# Record exact version in Phase 0 ADR before committing
```

Add to `app.go` `NewApp()` — module registration:

```go
import (
    "github.com/evmos/ethermint/x/evm"
    evmkeeper "github.com/evmos/ethermint/x/evm/keeper"
    evmtypes "github.com/evmos/ethermint/x/evm/types"
    "github.com/evmos/ethermint/x/feemarket"
    feemarketkeeper "github.com/evmos/ethermint/x/feemarket/keeper"
    feemarkettypes "github.com/evmos/ethermint/x/feemarket/types"
    "github.com/evmos/ethermint/x/erc20"
    erc20keeper "github.com/evmos/ethermint/x/erc20/keeper"
    erc20types "github.com/evmos/ethermint/x/erc20/types"
    evmante "github.com/evmos/ethermint/app/ante"
)

// Keepers — in order
app.FeeMarketKeeper = feemarketkeeper.NewKeeper(appCodec, authtypes.NewModuleAddress(govtypes.ModuleName), keys[feemarkettypes.StoreKey], tkeys[feemarkettypes.TransientKey], app.GetSubspace(feemarkettypes.ModuleName))
app.EvmKeeper = *evmkeeper.NewKeeper(appCodec, keys[evmtypes.StoreKey], tkeys[evmtypes.TransientKey], authtypes.NewModuleAddress(govtypes.ModuleName), app.AccountKeeper, app.BankKeeper, app.StakingKeeper, app.FeeMarketKeeper, tracer)
app.Erc20Keeper = erc20keeper.NewKeeper(keys[erc20types.StoreKey], appCodec, authtypes.NewModuleAddress(govtypes.ModuleName), app.AccountKeeper, app.BankKeeper, app.EvmKeeper, app.StakingKeeper)
```

##### Step 2 — Module initialization order (critical)

```go
// SetOrderInitGenesis — x/feemarket MUST precede x/evm; x/evm MUST precede x/erc20
// wasmd (CosmWasm) has no ordering constraint with EVM modules
app.SetOrderInitGenesis(
    // ... existing modules ...
    feemarkettypes.ModuleName,   // before x/evm
    evmtypes.ModuleName,         // after x/feemarket
    erc20types.ModuleName,       // after x/evm
    // wasmd.ModuleName can be anywhere after x/bank
)

// SetOrderBeginBlockers — same constraint
app.SetOrderBeginBlockers(
    // existing ...
    feemarkettypes.ModuleName,
    evmtypes.ModuleName,
    erc20types.ModuleName,
)

// SetOrderEndBlockers — same constraint
app.SetOrderEndBlockers(
    // existing ...
    feemarkettypes.ModuleName,
    evmtypes.ModuleName,
    erc20types.ModuleName,
)
```

##### Step 3 — Ante handler chain

```go
// Replace the existing ante handler setup with Ethermint's unified handler.
// This handles BOTH EVM txs (MsgEthereumTx) and Cosmos txs (everything else)
// through a single entry point. Detection is automatic via tx type.
options := evmante.HandlerOptions{
    AccountKeeper:          app.AccountKeeper,
    BankKeeper:             app.BankKeeper,
    IBCKeeper:              app.IBCKeeper,
    FeeMarketKeeper:        app.FeeMarketKeeper,
    EvmKeeper:              &app.EvmKeeper,
    FeegrantKeeper:         app.FeeGrantKeeper,
    SignModeHandler:        app.TxConfig().SignModeHandler(),
    MaxTxGasWanted:         maxGasWanted,
    ExtensionOptionChecker: evmtypes.HasDynamicFeeExtensionOption,
    TxFeeChecker:           evmante.NewDynamicFeeChecker(app.EvmKeeper),
}
anteHandler, err := evmante.NewAnteHandler(options)
if err != nil {
    panic(fmt.Errorf("failed to create ante handler: %w", err))
}
app.SetAnteHandler(anteHandler)

// ⚠ CRITICAL: Do NOT add a separate CosmWasm-specific ante handler.
// evmante.NewAnteHandler routes MsgEthereumTx → EVM decorator chain
// and all other msgs → standard Cosmos decorator chain automatically.
// Adding a second handler creates a split ante chain and breaks gas accounting.
```

##### Step 4 — Genesis parameters

```go
// x/feemarket genesis (app.toml initial params)
feemarkettypes.DefaultGenesisState() // override as needed:
// NoBaseFee: false (EIP-1559 active from block 1)
// BaseFee: sdk.NewInt(1000000000) // 1 Gwei in atoken
// ElasticityMultiplier: 2
// EnableHeight: 0

// x/evm genesis
evmtypes.DefaultGenesisState() // override:
// ChainConfig.ChainID: big.NewInt(<EVM-chain-id-from-ADR>)
// Params.EvmDenom: "atoken"
// Params.EnableCreate: true
// Params.EnableCall: true
// Params.AllowUnprotectedTxs: false  // EIP-155 replay protection enforced
// Params.ExtraEIPs: []               // no extra EIPs at launch
```

##### Step 5 — JSON-RPC server (`app.toml`)

```toml
[json-rpc]
enable            = true
address           = "0.0.0.0:8545"
ws-address        = "0.0.0.0:8546"
api               = ["eth", "net", "web3", "txpool", "debug"]
gas-cap           = 25000000
evm-timeout       = "5s"
logs-cap          = 10000
block-range-cap   = 10000
http-timeout      = "30s"
http-idle-timeout = "120s"
allow-unprotected-txs = false
max-open-connections  = 0          # unlimited; Envoy handles rate limiting
```

Expose via Envoy (add to Phase 5.6):
```yaml
- prefix: /evm-rpc
  cluster: chain_evm_rpc      # HTTP upstream; port 8545
  timeout: 30s
- prefix: /evm-ws
  cluster: chain_evm_ws       # WebSocket upstream; port 8546
  upgrade: websocket
```

##### Step 6 — CosmWasm + EVM coexistence test (the mandatory spike)

This is the single most important test of Phase 2.9. Run on devnet:

```go
// TestCosmWasmEVMCoexistence in /e2e/evm_cosmwasm_test.go
func TestCosmWasmEVMCoexistence(t *testing.T) {
    // Setup: devnet with both x/evm and wasmd active

    // Test 1: CosmWasm ExecuteContract in same block as MsgEthereumTx
    // Submit both in the same block; verify:
    // - Both txs included (not dropped or rejected)
    // - CosmWasm state change applied correctly
    // - EVM state change applied correctly
    // - Gas charged independently (EVM gas ≠ Cosmos gas consumed for CosmWasm tx)
    // - No StateDB interference between EVM and CosmWasm state stores

    // Test 2: x/bank interaction from both runtimes
    // CosmWasm contract sends BankMsg to address A
    // EVM tx sends value to same address A in same block
    // Verify: final balance = sum of both transfers (no double-spend, no lost funds)

    // Test 3: Native token in EVM (via x/erc20)
    // Register native utoken → ERC-20 via x/erc20 governance proposal
    // Send 100 utoken to an EVM address via Cosmos tx
    // Verify: EVM address can spend via ERC-20 transfer
    // Verify: CosmWasm contract can query the x/bank balance unchanged

    // Test 4: ABCI++ with EVM txs
    // PrepareProposal must correctly handle MsgEthereumTx alongside oracle commits
    // Verify oracle ordering (oracle commits first) doesn't drop EVM txs
}
```

**Pass criteria (all must pass before testnet launch):**
- [ ] Both CosmWasm and EVM txs in same block, no state corruption
- [ ] x/bank balances correct after dual-runtime interactions
- [ ] No ante handler panics or gas accounting errors in logs
- [ ] EVM tx gas reported correctly (in `atoken`); Cosmos tx gas reported correctly (in `utoken`)
- [ ] `TestCosmWasmEVMCoexistence` CI green on 5 consecutive devnet runs

##### Step 7 — Blockscout deployment

```yaml
# docker-compose.yml addition
blockscout-db:
  image: postgres:16
  environment:
    POSTGRES_USER: blockscout
    POSTGRES_PASSWORD: blockscout
    POSTGRES_DB: blockscout
  ports:
    - "5435:5432"   # separate port — NOT shared with Write/Read/Relayer DBs

blockscout:
  image: blockscout/blockscout:latest
  depends_on: [blockscout-db, chain]
  environment:
    ETHEREUM_JSONRPC_VARIANT:   "geth"
    ETHEREUM_JSONRPC_HTTP_URL:  "http://chain:8545"
    ETHEREUM_JSONRPC_WS_URL:    "ws://chain:8546"
    DATABASE_URL:               "postgresql://blockscout:blockscout@blockscout-db:5432/blockscout"
    COIN:                       "TOKEN"          # display name
    CHAIN_ID:                   "<EVM-chain-id>" # from ADR
    NETWORK:                    "MyChain"
    SUBNETWORK:                 "Devnet"
    SECRET_KEY_BASE:            "<random-64-char-secret>"
    PORT:                       "4000"
  ports:
    - "4000:4000"
```

Verify Blockscout on devnet:
```bash
# Deploy a test Solidity contract via Foundry
forge create --rpc-url http://localhost:80/evm-rpc \
  --private-key <devnet-test-key> \
  src/Counter.sol:Counter

# Verify Blockscout indexed the deployment
curl http://localhost:4000/api/v2/transactions/<tx-hash>
# Expect: contract creation tx with decoded ABI

# Verify MetaMask can connect
# Network settings: RPC URL = https://<devnet-domain>/evm-rpc, Chain ID = <from ADR>
# Verify: balance shows correctly; test transfer succeeds
```

##### Step 8 — `x/erc20` native token registration (devnet only)

```bash
# Register the native token as an ERC-20 on the EVM side
# This allows EVM contracts and MetaMask to see the native token as an ERC-20
chaind tx erc20 register-coin utoken \
  --from <chain-team-key> \
  --chain-id mychain-1

# Verify: ERC-20 contract address assigned to native token
chaind query erc20 token-pairs

# In MetaMask: import the ERC-20 contract address
# Verify: MetaMask shows TOKEN balance matching Cosmos-side utoken balance
```

##### Step 9 — Simulation `WeightedOperations`

```go
// x/evm simulation operations
evmsimulation.WeightedOperations(
    simulation.NewWeightedOperation(20, evmsimulation.SimMsgEthSimpleTransfer(ak, bk, ek)),
    simulation.NewWeightedOperation(15, evmsimulation.SimMsgEthContractCreate(ak, bk, ek)),
    simulation.NewWeightedOperation(10, evmsimulation.SimMsgEthContractCall(ak, bk, ek)),
)
// These run alongside existing custom module operations
// Verify simulation doesn't panic on EVM + CosmWasm + bridge txs in same block
```

##### Step 10 — `x/authz` EVM block verification

```go
// Integration test: verify MsgEthereumTx cannot be granted via x/authz
func TestAuthzEVMBlock(t *testing.T) {
    // Attempt to grant MsgEthereumTx via x/authz
    // Expect: rejected with "message type not allowed via authz grant"
    // This confirms Phase 0 ADR authz block is enforced at runtime
}
```

**Deliverable checklist — Phase 2.9:**
- [ ] `github.com/evmos/ethermint` pinned at exact version in `go.mod` and Phase 0 ADR
- [ ] `x/feemarket`, `x/evm`, `x/erc20` wired in `app.go`; module init order correct
- [ ] Ethermint `x/feemarket` in use; `skip-mev/x/feemarket` absent from `go.mod` and `app.go`
- [ ] Ante handler: `evmante.NewAnteHandler` in use; no separate CosmWasm ante handler
- [ ] JSON-RPC server active on port 8545; WebSocket on 8546; reachable via Envoy `/evm-rpc` and `/evm-ws`
- [ ] `cast block-number --rpc-url .../evm-rpc` returns correct block height
- [ ] `TestCosmWasmEVMCoexistence` passes (5 consecutive green runs on devnet)
- [ ] Blockscout deployed; indexes devnet EVM txs; Solidity contract creation visible and decoded
- [ ] MetaMask connects to devnet EVM via `/evm-rpc`; balance displays correctly
- [ ] Native token registered as ERC-20 via `x/erc20`; MetaMask shows TOKEN balance
- [ ] `TestAuthzEVMBlock` passes: `MsgEthereumTx` authz grant rejected at protocol level
- [ ] EVM simulation `WeightedOperations` running in `--NumBlocks=5000` without panic
- [ ] Devnet docker-compose updated: 5 databases (Write DB, Read DB, Relayer DB, Blockscout DB, + TimescaleDB if adopted)

---

### Phase 3 — CosmWasm Contract Suite (Weeks 4–10)

**Toolchain:** `cosmwasm-std`, `cw-storage-plus`, `cw-multi-test` (unit logic) + mandatory on-chain devnet integration tests for all fund-movement paths (`cw-multi-test` does not enforce real `x/bank` module-account permissions).

**Cross-runtime boundary (architectural constraint — read before implementing):** CosmWasm contracts in this phase **cannot call the EVM runtime** and EVM contracts **cannot call CosmWasm contracts** at mainnet launch. There is no IBC-style cross-runtime message passing. The two runtimes are isolated: they share `x/bank` balances (via `x/erc20` conversion for EVM) and can both read chain state, but a Solidity contract cannot invoke a CosmWasm `ExecuteMsg` and a CosmWasm contract cannot call a Solidity function. Post-launch, the `x/oracle` and `x/milestone` precompiles (governance-gated) will be the only sanctioned channel by which Solidity can read data from the Cosmos side — and that is read-only only. Do not design CosmWasm contract logic that assumes cross-runtime call capability.

#### 3.1 Constitution Contract
- Stores constitutional rules; amendment via supermajority + delay
- Read-only `QueryMsg` interface consumed by Governance contract
- **`EmergencyPause` blocks `ExecuteMsg` only, never `QueryMsg`.** Constitution queries remain available during pause so governance continues to function. Implementation: `execute()` entry point checks `is_paused` flag and returns error if true; `query()` entry point has no such check.
- Gas budget: benchmarked to complete within the 500,000 gas initial governance parameter

#### 3.2 Treasury Contract
- Manages treasury module account; threshold multi-sig disbursements
- **Cold multi-sig emergency override:** 5-of-7 hardware keys; key holder set defined in ADR (2 founding team members, 2 independent security council members, 2 lead validators, 1 legal trustee — or equivalent, must be defined before mainnet); geographically distributed; `EmergencyPause{}` is pause-only (no fund access, no unpause)
- Governance-controlled unpause only

#### 3.3 Reserve Fund Contract
- Disbursements gated on `x/milestone` state; circuit-breaker at minimum balance
- Same cold multi-sig emergency override as Treasury

#### 3.4 Governance Contract
- Only contract permitted to call `ExecuteMsg` on Constitution, Treasury, Reserve Fund
- Enforces Constitution compliance (queries Constitution via `QueryMsg`; queries succeed even if Constitution is paused)
- On-chain audit log
- **Governance contract replacement procedure:**
  1. Cold multi-sig pauses Treasury and Reserve Fund (prevents fund movement during replacement)
  2. Governance submits `MsgMigrateContracts` proposal (bypasses Constitution check, has 7-day time-lock)
  3. On execution: new Governance contract instantiated; old Governance contract's cross-contract authority updated to new address; cold multi-sig unpauses Treasury and Reserve Fund
  4. This procedure tested on devnet before mainnet

#### 3.5 Genesis Wiring & Authority
- All four contracts instantiated in `app_state.wasm` at genesis
- Authority relationships set at instantiation; cold multi-sig address hardcoded and published publicly
- **Cold multi-sig address rotation:** updatable via `MsgMigrateContracts` governance proposal (7-day time-lock, bypasses Constitution check); rotation tested on devnet
- Fund migration safety: `ExecuteMsg::MigrateBalance{new_address}` callable by Governance contract (normal upgrade) or cold multi-sig (emergency); tested on devnet

**Testing:**
- `cw-multi-test`: logic, Constitution check, cross-contract authority, rejection, EmergencyPause (ExecuteMsg blocked, QueryMsg passes)
- **On-chain devnet integration tests (mandatory for all fund-movement paths):** real `x/bank` module-account permissions; emergency override; fund migration; Constitution `QueryMsg` during pause; Governance contract replacement procedure

**JSON Schema upload for Celatone (mandatory per Phase 5.11):**
Celatone displays decoded ExecuteMsg and QueryMsg only if a JSON Schema is uploaded alongside the contract code. Without it, messages appear as raw JSON — identical to Ping.pub. Add the following step to every contract code upload:

```bash
# After uploading contract WASM binary:
chaind tx wasm store <contract>.wasm \
  --from <chain-team-key> --chain-id <chain-id> --gas auto

# Upload JSON Schema alongside the code (Celatone reads this by code ID)
# Generate schema from contract:
cd contracts/<contract-name>
cargo schema          # outputs schema/*.json

# Upload to Celatone schema registry (self-hosted instance):
curl -X POST https://<celatone-host>/api/schema/upload \
  -H "Content-Type: application/json" \
  -d @schema/execute_msg.json

# Verify: open Celatone → search code ID → ExecuteMsg shows decoded fields
```

Add this step to the deployment runbook for all four CosmWasm contracts: Constitution, Treasury, Reserve Fund, Governance. Schemas must be uploaded before Phase 9 audit kickoff — auditors use Celatone for contract state inspection.

**Deliverable:** All four contracts (Constitution, Treasury, Reserve Fund, Governance) passing both `cw-multi-test` and on-chain tests; cold multi-sig holders defined; genesis file ready; JSON Schema uploaded for all four contracts and verified decoded in Celatone on devnet.

---

### Phase 4 — BNB Smart Chain Bridge (Weeks 10–15)

**Bridge threat model written as Phase 0 ADR deliverable — not deferred to Phase 4.** Implementation proceeds only after threat model is approved.

#### 4.1 Supply Cap Model
- Cosmos genesis: `S - C` tokens; bridge escrow = `C`
- `x/bridge` invariant: `cosmos_minted_via_bridge + bsc_circulating = S` at all times. **IBC clarification:** tokens transferred out via IBC remain within Cosmos-side `x/bank` total supply and do not break this invariant. The invariant is measured over total `x/bank` supply minus bridge escrow, not just local chain balance. IBC-out tokens are in escrow on the local chain (locked by IBC transfer module) and still count as part of `cosmos_minted_via_bridge`. See Phase 0 ADR for full definition.
- Atomic check+mint in single ABCI handler (single-threaded; safe by construction; documented for auditors)

#### 4.2 BSC Smart Contracts (Foundry)

**LockBox Contract:**
- `lock(amount, cosmosRecipient)` — nonce generated by contract: `nonce = keccak256(msg.sender || block.number || amount || block.timestamp)`. User does NOT supply nonce. **Collision-resistant** (not "unpredictable" — `block.timestamp` is miner-influenceable ±15s on BSC; predictability is not a security property here, only uniqueness is, enforced by the bitmap registry).
- Emits `Locked(user, amount, cosmosRecipient, nonce)`
- Releases only on quorum proof from relayer committee
- **Bitmap nonce registry:** 256-bit nonces tracked in a bitmap; out-of-order confirmation supported; max in-flight nonces per user (governance parameter); nonce expiry for abandoned transactions
- **Tiered confirmation depth:** N=15 standard; N=50 for transfers above value threshold (both governance parameters in `x/bridge`)
- **Fast circuit breaker:** `circuitBreakerAddress` can call `pause()` instantly; pause-only (no fund access, no unpause); Gnosis Safe can pause AND unpause; circuit-breaker EOA rotatable by Gnosis Safe
- Rate limit: max unlock per block (governance parameter)

**Tests (Foundry):**
- Unit: lock, unlock, keccak256 nonce generation, bitmap nonce, tiered confirmation, circuit-breaker pause, Gnosis Safe pause/unpause, rate limit
- Fuzz: `forge test --fuzz-runs 50000` on lock/unlock/nonce
- Invariant: `totalLocked == totalReleasedToUsers + totalPendingUnlock`
- Circuit-breaker drill: < 30s from key access to confirmed pause on testnet

#### 4.3 Cosmos Bridge Module — `x/bridge`
- `MsgBridgeIn`: quorum check → supply cap check+mint (atomic)
- `MsgBridgeOut`: burn → event → BSC release
- Bitmap nonce registry (matches BSC side); out-of-order supported
- Relayer set registry (governance-managed)
- Governance parameters: finality depths N (standard, large), large-transfer threshold, quorum T, rate limit, circuit-breaker address

**Simulation WeightedOperations:**
- `SimMsgBridgeIn` (weight: 15), `SimMsgBridgeOut` (weight: 15), `SimMsgBridgeInCapBreach` (weight: 3, verify rejection)

#### 4.4 Go Relayer Engine (`/relayer`)

- **`bsc_watcher`:** tiered confirmation (N=15 or N=50 per on-chain threshold); publishes to NATS `account:bridge` subject `bridge.bsc.locked.{nonce}`
- **`cosmos_watcher`:** gRPC streaming; publishes to `bridge.cosmos.burnout.{nonce}`
- **`sig_aggregator`:** each relayer independently signs; publishes to `bridge.sig.{nonce}`; deduplicates (same relayer cannot count twice)
  - **Quorum timeout:** T seconds → re-publish with incremented retry counter; max retries (governance parameter); after max → `stuck` event → alert
  - **Deterministic promotion ladder for submitter failover (per-event, within a signing round):**
    - Relayers sorted deterministically by operator address
    - Lowest-index relayer = designated submitter for this event; waits T/2 seconds
    - If `MsgBridgeIn` not seen on-chain after T/2 seconds: second-lowest-index promotes; waits T/4 seconds
    - If still not seen: third-lowest promotes; and so on
    - Only ONE relayer attempts submission at any time — no simultaneous promotions, no duplicate submissions, no wasted gas
    - **Scope:** this ladder operates at the per-event (per-nonce) level within a single signing round. It is distinct from the governance-managed Primary/Secondary/Candidate tier system in the Bridge Relayer Onboarding Guide, which operates at the long-term operator tier level and tracks miss counts over a 10-minute window. These are two complementary mechanisms: the ladder handles per-event submitter fallback; the governance tier system handles long-term operator demotion. Both must be implemented.
- **`submitter`:** gas estimation, sequence management, exponential backoff retry
- **Relayer DB:** bitmap nonce state, confirmation tracking, vote records, stuck event log (isolated PostgreSQL, no sharing with backend)
- NATS offset recovery: resume from offset; fallback to BSC block scan from Relayer DB checkpoint if offset aged out (365-day retention)

#### 4.5 Bridge Security
- Threat model: complete and approved in Phase 0 (not Phase 4)
- Separate audit scope: BSC contracts, `x/bridge`, relayer engine
- Horcrux threshold signing for relayer keys; no hot keys
- Rate limit active from day 1 of mainnet bridge activation
- Emergency pause drilled on testnet: < 60 seconds from alert to confirmed pause
- Circuit-breaker EOA key compromise runbook (see Phase 8.4)

**Deliverable:** Bridge on testnet with bitmap nonces, tiered confirmations, fast circuit-breaker, deterministic promotion ladder, supply invariant verified under load.

---

### Phase 5 — Off-chain Backend (Weeks 12–18, dedicated sub-team of 5–6 engineers)

**Note: Phase 5 extends to Week 18 (not Week 15 as previously stated) to be realistic for the scope. The dedicated backend sub-team works in parallel with Phases 4, 6, and 7 but is not shared with E2E maintenance.**

**Objective:** Full two-DB CQRS pipeline, correct NATS event flow, Envoy with CORS and separate gRPC-Web/REST clusters, grpc-gateway as separate Deployment, 3-node NATS, Ping.pub explorer. All streaming via gRPC server-streaming only — no WebSocket bypass route.

#### 5.1 Protobuf Service Definitions (`/proto`)
- `backend/v1/query.proto`, `backend/v1/stream.proto`, `relayer/v1/relayer.proto`
- `buf generate`: Go stubs, gRPC-Web stubs, REST/OpenAPI, TypeScript types for dApp SDK
- CI: `buf lint`, `buf breaking` (baseline seeded on first addition)

#### 5.2 Write DB Schema and Backup Setup
- Migrations in `/db/write_schema/` via `golang-migrate`
- Event tables: `(block_height BIGINT, event_index INT, event_type TEXT, payload JSONB, nats_published BOOLEAN DEFAULT false, PRIMARY KEY (block_height, event_index))`
- **Table partitioning:** range-partitioned by `block_height` (monthly partitions); automated partition creation script run monthly
- **WAL archiving:** configured from day one; archives to S3/GCS bucket; enables PITR to any second
- **Daily base backup:** scheduled via pg_basebackup or managed service snapshot; 30-day retention
- **PostgreSQL streaming replication:** primary (Region A) + synchronous standby (Region B); `synchronous_commit = off` (see note below); Patroni for automatic failover; Relayer DB uses `synchronous_commit = on` (nonce state is not reconstructable)

**`synchronous_commit = off` on Write DB (safe here, not safe on Relayer DB):**
Write DB is a pure append-only projection of CometBFT block data. Every event in the Write DB is reconstructable at any time by replaying `BlockResults` RPC from block 0. Losing the last few milliseconds of buffered writes on a crash costs at most one block of re-indexing at next startup — not data loss. Setting `synchronous_commit = off` eliminates the synchronous replication round-trip penalty on every INSERT and gives **5–10× write throughput improvement** with zero durability risk for this specific workload.
- Relayer DB: keep `synchronous_commit = on` — nonce bitmap and confirmation state are NOT reconstructable from chain data alone; loss is a security event.
- Read DB: `synchronous_commit = off` acceptable — projections are rebuilt from Write DB on full replay.

**TimescaleDB extension for Write DB:**
TimescaleDB is a drop-in PostgreSQL extension (same SQL, same drivers, same advisory locks, same PITR, same Patroni failover). It adds two capabilities that directly benefit the Write DB at zero increase in operational complexity:

| Capability | Benefit for this system |
|---|---|
| **Automatic chunk compression** | 90–95% storage reduction on cold partitions (blocks older than N days). Replaces the manual cold-storage archive process — TimescaleDB compresses old chunks in-place; no separate pipeline needed. |
| **Continuous aggregates** | Pre-computed, incrementally refreshed materialized views. `bridge_volume_by_period`, `validator_uptime`, and `oracle_participation_rate` become continuous aggregates on the Write DB rather than manually maintained singleton projections in `module/projection`. Removes the aggregate singleton StatefulSet requirement for these three projections. |

Installation is a single extension:
```sql
CREATE EXTENSION IF NOT EXISTS timescaledb;

-- Convert event table to hypertable (replaces manual range partitioning)
SELECT create_hypertable('events', 'block_height',
  chunk_time_interval => 432000,    -- ~30 days at 6s block time = 432,000 blocks
  if_not_exists => TRUE
);

-- Enable compression on chunks older than 60 days
ALTER TABLE events SET (
  timescaledb.compress,
  timescaledb.compress_orderby = 'block_height ASC, event_index ASC',
  timescaledb.compress_segmentby = 'event_type'
);
SELECT add_compression_policy('events', compress_after => 432000 * 2);  -- compress after ~60 days

-- Continuous aggregate: bridge volume by day (replaces module/projection singleton)
CREATE MATERIALIZED VIEW bridge_volume_daily
WITH (timescaledb.continuous) AS
  SELECT time_bucket(432000, block_height) AS period,
         SUM((payload->>'amount')::NUMERIC)  AS volume,
         COUNT(*)                             AS tx_count
  FROM events
  WHERE event_type = 'bridge_locked'
  GROUP BY period
WITH NO DATA;

SELECT add_continuous_aggregate_policy('bridge_volume_daily',
  start_offset => 432000 * 3,
  end_offset   => 432000,
  schedule_interval => INTERVAL '1 hour'
);
```

**Decision gate (Phase 5 ADR):** confirm TimescaleDB before Phase 5.2 implementation begins. Evaluation criteria:
- [ ] TimescaleDB OSS licence acceptable (Apache 2.0 for core features used here)
- [ ] Managed service availability confirmed (Timescale Cloud, or self-hosted on k8s via Helm chart)
- [ ] Chunk compression tested on devnet: verify advisory lock still works, PITR still works, `golang-migrate` migrations still apply
- [ ] If TimescaleDB is rejected: keep manual range partitioning + cold-storage archive pipeline as originally specified

**If TimescaleDB is adopted, remove from `module/projection`:**
- `bridge_volume_by_period` aggregate — replaced by continuous aggregate
- `validator_uptime` aggregate — replaced by continuous aggregate
- `oracle_participation_rate` aggregate — replaced by continuous aggregate
- The aggregate singleton StatefulSet is no longer needed for these three; `module/projection` simplifies to fan-out KV projections only (multi-replica Deployment, no advisory lock)

#### 5.3 CQRS Write Side — `module/ingestion`
- **Singleton StatefulSet:** 1 replica; PostgreSQL advisory lock on startup
- **Startup reconciliation:** query `SELECT MAX(block_height) FROM events` → subscribe CometBFT WebSocket from that height + 1 → replay missed blocks → resume live subscription
- **NATS back-fill on reconnect:** query `WHERE nats_published = false ORDER BY block_height` → publish in order → mark `nats_published = true`; runs automatically on every NATS reconnection
- CometBFT WebSocket → parse typed events from all 7 custom modules → Write DB (`INSERT ... ON CONFLICT (block_height, event_index) DO NOTHING`) → publish to `account:chain` → mark `nats_published = true`

#### 5.4 CQRS Read Side — `module/projection`

**Two deployment modes — conditional on TimescaleDB ADR decision:**

**If TimescaleDB adopted (Phase 5.10):**
- **Single fan-out Deployment (multi-replica, no StatefulSet, no advisory lock)** — all projections are safe for concurrent writes:
  - KV projections: `settlement_by_id`, `milestone_status`, `bridge_pending_by_nonce` (`INSERT ... ON CONFLICT DO UPDATE`)
  - **TimescaleDB hypertable writes (new — required for Phase 5.10 analytics):**
    - `block_stats` — written on every block event: `block_height`, `block_time_ms`, `tx_count`, `avg_fee_uatom`
    - `oracle_submissions` — written on every `x/oracle` reveal event: `block_height`, `asset_id`, `price`, `validator`
    - `validator_signatures` — written on every block: one row per validator per block, `signed = true/false`
    - `bridge_events` — written on every `x/bridge` lock/release event: `block_height`, `event_index`, `direction`, `asset`, `amount`
  - Aggregate projections (`bridge_volume_by_period`, `validator_uptime`, `oracle_participation_rate`) are **not written by module/projection** — they are maintained as TimescaleDB continuous aggregates in Phase 5.10. Do not implement these as Go code.

**If TimescaleDB rejected:**
- **Singleton StatefulSet** (for aggregate projections): `bridge_volume_by_period`, `validator_uptime`, `oracle_participation_rate`; uses advisory lock; same crash-recovery pattern as ingestion
- **Deployment (multi-replica, fan-out)** (for KV projections): `settlement_by_id`, `milestone_status`, `bridge_pending_by_nonce`; no advisory lock needed

**NATS consumer type:** push subscriber (fan-out) on `account:chain` — every instance sees every message.
**Replay paths:**
- NATS offset within retention (90 days): replay from NATS consumer offset on startup
- Beyond retention: replay from Write DB via `projection_reader` user: `SELECT * FROM events WHERE block_height > last_committed ORDER BY block_height`

**After writing Read DB:** publish transformed/enriched events to `account:stream` for real-time delivery to `module/api`.

#### 5.5 gRPC API — `module/api`
- Pure gRPC server; reads from Read DB via `api_reader` user (SELECT only; no Write DB connection)
- **grpc-gateway — separate Deployment:** grpc-gateway runs as its own Kubernetes Deployment (not a sidecar in the `module/api` pod). This provides independent HPA scaling and CPU/memory isolation from the gRPC server. `protoc-gen-grpc-gateway` annotations on all proto RPCs generate REST handlers. Envoy proxies `/api/rest/*` to the grpc-gateway Service. Envoy proxies `/api/grpcweb/*` to the module/api gRPC Service. These are two separate upstream clusters.
- **CI enforcement:** `buf lint` must fail on any RPC in `backend/v1/query.proto` that lacks a `google.api.http` annotation — prevents REST endpoints disappearing silently when devs forget annotations.
- **All list RPCs use cursor-based pagination** (see Phase 0 ADR pagination strategy). No offset pagination anywhere.
- **Server-streaming RPCs:** subscribes to NATS `account:stream`; pushes events to clients via gRPC server-streaming. In-process fan-out with per-client channel buffer of 64 events. Slow consumer (full buffer) → stream terminated with `ResourceExhausted`; client SDK reconnects transparently.
- `module/api` has NO connection to Write DB, NO subscription to `account:chain`, and **NO direct connection to chain gRPC** (CQRS boundary rule — see System Architecture section).

**Analytics RPCs — backend implementation required in Phase 5.5 (not only in Phase 7.3 frontend):**
The following 6 RPCs must be defined in `backend/v1/query.proto` and implemented in `module/api` during Phase 5. They serve data from the TimescaleDB continuous aggregates (Phase 5.10). The Phase 7.3 analytics dashboard frontend consumes them, but the backend must exist first. Add these to the proto file and run `pnpm --filter @workspace/api-spec run codegen` before Phase 7.3 begins:

```protobuf
// All serve data from TimescaleDB continuous aggregates on Read DB
rpc GetTps             (GetTpsRequest)             returns (GetTpsResponse);            // → tps_1h
rpc GetBlockStats      (GetBlockStatsRequest)       returns (GetBlockStatsResponse);     // → block_time_1h
rpc GetBridgeVolume    (GetBridgeVolumeRequest)     returns (GetBridgeVolumeResponse);   // → bridge_volume_1h
rpc GetOraclePrice     (GetOraclePriceRequest)      returns (GetOraclePriceResponse);    // → oracle_price_1h
rpc GetValidatorUptime (GetValidatorUptimeRequest)  returns (GetValidatorUptimeResponse);// → validator_uptime_1d
rpc StreamChainStats   (StreamChainStatsRequest)    returns (stream ChainStatsEvent);    // live TPS via NATS account:stream
```

All RPCs must have `google.api.http` annotations for `buf lint` CI to pass. `StreamChainStats` is a server-streaming RPC — apply `timeout: 0s` / `idle_timeout: 600s` Envoy config (same as other streaming routes). These RPCs are conditional on TimescaleDB adoption; if TimescaleDB is rejected, define them as returning data from manually maintained aggregate tables.

#### 5.6 Envoy API Gateway
```yaml
routes:
  - prefix: /api/rest/
    cluster: backend_grpc_gateway     # HTTP/1.1; grpc-gateway Deployment (separate from module/api)

  - prefix: /api/grpcweb/
    cluster: backend_grpc             # HTTP/2; gRPC-Web transcoding filter applied
    # Streaming routes MUST have these timeouts or Envoy default (60s idle) kills long-lived streams:
    timeout: 0s                       # no global timeout for streaming routes
    idle_timeout: 600s                # drop truly idle streams after 10 min
    route:
      max_stream_duration:
        grpc_timeout_header_max: 0s   # respect client-specified deadlines; 0 = no cap

  # NOTE: /api/stream WebSocket route has been REMOVED.
  # Rationale: direct NATS→WebSocket bypass exposed internal NATS message format to browsers,
  # bypassed service-layer auth/versioning, and created a split API contract.
  # All real-time streaming is served via gRPC server-streaming through /api/grpcweb/ above.

  # EVM JSON-RPC routes (Ethermint — added in Phase 2.9)
  - prefix: /evm-rpc
    cluster: chain_evm_rpc     # HTTP/1.1 upstream to Ethermint JSON-RPC (port 8545)
    timeout: 30s               # JSON-RPC calls are synchronous; 30s covers debug_ calls
    idle_timeout: 0s

  - prefix: /evm-ws
    cluster: chain_evm_ws      # WebSocket upstream to Ethermint WS-RPC (port 8546)
    upgrade: websocket
    timeout: 0s                # WebSocket connections are long-lived
    idle_timeout: 600s         # Drop truly idle WS connections after 10 min
    # Used by: MetaMask "watch block" subscription, wagmi/viem subscriptions, Blockscout indexer
    # CORS: EVM routes need same CORS headers as /api/* — add to Envoy CORS policy

  - prefix: /grpc/
    cluster: chain_grpc               # HTTP/2; gRPC-Web transcoding filter applied
  - prefix: /rpc
    cluster: chain_rpc                # CometBFT JSON-RPC
  - prefix: /ws
    cluster: chain_ws                 # CometBFT WebSocket

cors_policy:
  # /api/rest/* — tooling, explorers, third-party integrations (GET + POST acceptable)
  rest_routes:
    allow_origin_string_match:
      - exact: "https://app.example.com"
      - exact: "https://staging.example.com"
    allow_methods: GET, POST, OPTIONS
    allow_headers: Content-Type, Authorization
    expose_headers: Content-Type
    max_age: 86400

  # /api/grpcweb/* — dApp primary interface (gRPC-Web uses POST exclusively)
  grpcweb_routes:
    allow_origin_string_match:
      - exact: "https://app.example.com"
      - exact: "https://staging.example.com"
    allow_methods: POST, OPTIONS      # GET is NOT valid for gRPC-Web; explicitly excluded
    allow_headers: Content-Type, X-Grpc-Web, X-User-Agent, X-Wallet-Address
    expose_headers: Grpc-Status, Grpc-Message, Grpc-Status-Details-Bin, Grpc-Encoding
    # Grpc-Encoding must be exposed or compressed gRPC-Web responses fail in browsers
    max_age: 86400

rate_limits:
  - descriptor: relayer_mtls_cert     # relayer gRPC submissions; 600 req/min per cert
  - descriptor: wallet_address        # extracted from x-wallet-address gRPC metadata header
                                      # set by client SDK on every call after wallet connection
                                      # 30 req/min per address on write endpoints
                                      # NOTE: this is rate-limiting only, not authentication proof
  - descriptor: ip_address            # read endpoints; 200 req/min per IP; /rpc: 100/min
```

**Rate limiting:** per-identity (relayer mTLS certificate for bridge submissions; wallet address for user writes via `x-wallet-address` header; IP for reads). Flat IP rate limiting on write endpoints explicitly NOT used — bridge relayer throughput must not be blocked. The `x-wallet-address` header must be documented in the SDK as a required metadata field after wallet connection; if absent, requests fall back to IP-based limiting.

**grpc-gateway upstream cluster:** `backend_grpc_gateway` points to the grpc-gateway Kubernetes Service (separate Deployment with its own HPA). `backend_grpc` points to the module/api gRPC Kubernetes Service. These are separate Services with separate cluster entries in Envoy — do not combine them.

**Authentication policy (intentional — document for auditors):**
No HTTP authentication middleware is configured at the Envoy gateway. This is a deliberate design decision, not an oversight:
- All state-mutating operations (`MsgBridgeIn`, `MsgBridgeOut`, oracle commits/reveals, governance votes) require a valid Ed25519 on-chain signature on the transaction itself. The signature is verified by the chain's ante handler — the API gateway never sees private keys and cannot substitute for this check.
- All read endpoints serve public blockchain data that is identical to what any full node RPC exposes. There is no private user data behind the read API.
- The `x-wallet-address` header is used for **rate limiting only** — it is not an authentication proof. A caller who lies about their address harms only their own rate-limit quota.
- This model is consistent with how Cosmos Hub, Osmosis, dYdX v4, and Celestia operate their API gateways.
- **If a future endpoint requires pre-authentication** (e.g., a private notification service or off-chain user preferences): add a JWT validation filter scoped to that specific route only. Do not retrofit gateway-wide authentication — it would break the relayer mTLS flow and add latency to every public read.

#### 5.7 Client SDK — gRPC-Web + Auto-Reconnect
- TypeScript SDK; types from `buf generate`
- gRPC-Web stubs call Envoy `/api/grpcweb/*` — **all queries and streaming use this single path**; no separate WebSocket path
- **`x-wallet-address` metadata header:** set on every call after wallet connection:
  ```typescript
  function buildMetadata(walletAddress?: string): grpc.Metadata {
    const metadata = new grpc.Metadata();
    if (walletAddress) {
      metadata.set('x-wallet-address', walletAddress);  // required for per-wallet rate limiting
    }
    return metadata;
  }
  ```
- **Server-streaming auto-reconnect:** all stream subscriptions wrapped in reconnect loop. `ResourceExhausted` status (slow consumer eviction) is treated as a reconnectable error:
  ```typescript
  async function subscribeWithReconnect(subscribe, onEvent, signal, walletAddress?) {
    let delay = 1000;
    while (!signal.aborted) {
      try {
        await subscribe(buildMetadata(walletAddress), onEvent);
        delay = 1000;
      } catch (err) {
        // ResourceExhausted = slow consumer evicted by server; reconnect normally
        await sleep(delay + jitter());
        delay = Math.min(delay * 2, 30000);
      }
    }
  }
  ```
  Envoy rolling restarts drop streams; slow consumer evictions return `ResourceExhausted`; SDK reconnects transparently in both cases. Users never see a stale bridge tracker.

#### 5.8 NATS 3-Node Cluster (Production)
- `nats-0`, `nats-1`, `nats-2` as StatefulSet with anti-affinity across availability zones
- JetStream R=3: every message replicated to all 3 nodes before ACK
- All stream retention policies from ADR applied at cluster boot
- Account credentials (`account:chain`, `account:bridge`, `account:stream`) stored in Vault; rotated annually
- **Chaos test before testnet graduation:** (a) kill nats-0 → verify continuity; (b) kill all 3 simultaneously → bring back → verify no message loss and consumer offsets correct; (c) simulate ingestion NATS reconnect → verify back-fill from `nats_published=false` rows → Read DB consistent

#### 5.9 Ping.pub Explorer
- Self-hosted; Cosmos chain only (no BSC archive node required)
- **Honest scope:** Ping.pub covers standard Cosmos SDK data only — blocks, transactions, validator set, staking, governance proposals, IBC channels. It does **not** decode custom module messages; `x/settlement`, `x/bridge`, `x/oracle`, `x/milestone`, `x/certification` transactions appear as raw JSON blobs. This is acceptable — Ping.pub serves node operators and validators who need infrastructure visibility, not end users who need custom module UIs.
- **Not used for:** analytics charts, bridge status, oracle price history, settlement lifecycle, or any TimescaleDB aggregate data — those are served by Phase 5.11 (Celatone) and Phase 7.3 (custom analytics dashboard).
- BSC bridge activity is **not** surfaced in Ping.pub; it is surfaced via the CQRS API in Phase 7.3.
- **Blockscout — EVM block explorer (Phase 2.9):** since this chain includes an Ethermint EVM execution layer (confirmed in Phase 0 ADR), Blockscout is the standard EVM explorer for the EVM side. Blockscout reads the Ethermint JSON-RPC endpoint at `/evm-rpc`; it covers EVM transactions, Solidity contract deployments and verification, ERC-20 token transfers, EVM account balances, and internal call traces. It does NOT cover Cosmos-native txs, CosmWasm contract state, or custom module state — those remain in Ping.pub (Cosmos) and Celatone (CosmWasm) respectively. Blockscout requires its own PostgreSQL instance (**Blockscout DB** — a fourth isolated database; no sharing with Write DB, Read DB, or Relayer DB). Expose at `/blockscout` via Envoy. Self-hosted; no SaaS dependency.

#### 5.10 Read DB Analytics Layer (TimescaleDB)

**Motivation:** The Read DB stores denormalized projections queried by `module/api`. Without TimescaleDB, all time-series aggregation (TPS charts, oracle OHLC, validator uptime trends, bridge volume over time, block statistics) must be computed in Go inside `module/projection` and stored as flat pre-aggregated rows. This creates an entire category of application-managed aggregate state: singleton StatefulSets, advisory locks, crash-recovery paths, and Go math that the database should own.

Applying TimescaleDB to the Read DB eliminates this category entirely. The Read DB becomes the analytics engine. `module/projection` shrinks to fan-out key-value projections only.

**Apply TimescaleDB to Read DB — migration 001:**
```sql
CREATE EXTENSION IF NOT EXISTS timescaledb;
```
All time-series tables below are created as hypertables. All existing key-value projection tables (`settlement_by_id`, `milestone_status`, `bridge_pending_by_nonce`) remain as plain PostgreSQL tables — TimescaleDB is additive and does not affect them.

---

**Time-series source tables (hypertables on Read DB):**

```sql
-- Block-level statistics written by module/projection on every block
CREATE TABLE block_stats (
  block_height   BIGINT    NOT NULL,
  block_time_ms  INT       NOT NULL,
  tx_count       INT       NOT NULL,
  avg_fee_uatom  NUMERIC   NOT NULL,
  PRIMARY KEY (block_height)
);
SELECT create_hypertable('block_stats', 'block_height', chunk_time_interval => 432000);

-- Oracle price submissions written by module/projection on every oracle event
CREATE TABLE oracle_submissions (
  block_height  BIGINT   NOT NULL,
  asset_id      TEXT     NOT NULL,
  price         NUMERIC  NOT NULL,
  validator     TEXT     NOT NULL,
  PRIMARY KEY (block_height, asset_id, validator)
);
SELECT create_hypertable('oracle_submissions', 'block_height', chunk_time_interval => 432000);

-- Validator signature records written by module/projection on every block
CREATE TABLE validator_signatures (
  block_height       BIGINT   NOT NULL,
  validator_address  TEXT     NOT NULL,
  signed             BOOLEAN  NOT NULL,
  PRIMARY KEY (block_height, validator_address)
);
SELECT create_hypertable('validator_signatures', 'block_height', chunk_time_interval => 432000);

-- Bridge events written by module/projection on every bridge event
CREATE TABLE bridge_events (
  block_height  BIGINT   NOT NULL,
  event_index   INT      NOT NULL,
  direction     TEXT     NOT NULL,   -- 'lock' | 'release'
  asset         TEXT     NOT NULL,
  amount        NUMERIC  NOT NULL,
  PRIMARY KEY (block_height, event_index)
);
SELECT create_hypertable('bridge_events', 'block_height', chunk_time_interval => 432000);
```

---

**Continuous aggregates (auto-maintained by TimescaleDB — no Go code):**

```sql
-- TPS: average and peak transactions per second per hour
CREATE MATERIALIZED VIEW tps_1h
WITH (timescaledb.continuous) AS
  SELECT time_bucket(600, block_height)        AS period,
         COUNT(*)::FLOAT / 600.0 / 6.0         AS tps_avg,
         MAX(tx_count)::FLOAT / 6.0            AS tps_peak,
         SUM(tx_count)                         AS total_txs
  FROM block_stats
  GROUP BY period
WITH NO DATA;
SELECT add_continuous_aggregate_policy('tps_1h',
  start_offset => 1800, end_offset => 600, schedule_interval => INTERVAL '10 minutes');

-- Block time statistics with percentile sketch (p50/p95/p99 queryable at runtime)
CREATE MATERIALIZED VIEW block_time_1h
WITH (timescaledb.continuous) AS
  SELECT time_bucket(600, block_height)        AS period,
         AVG(block_time_ms)                    AS avg_ms,
         MAX(block_time_ms)                    AS max_ms,
         percentile_agg(block_time_ms)         AS pctls
  FROM block_stats
  GROUP BY period
WITH NO DATA;
SELECT add_continuous_aggregate_policy('block_time_1h',
  start_offset => 1800, end_offset => 600, schedule_interval => INTERVAL '10 minutes');

-- Oracle OHLC candles per asset per hour (uses TimescaleDB first()/last() hyperfunctions)
CREATE MATERIALIZED VIEW oracle_price_1h
WITH (timescaledb.continuous) AS
  SELECT time_bucket(600, block_height)        AS period,
         asset_id,
         first(price, block_height)            AS open,
         max(price)                            AS high,
         min(price)                            AS low,
         last(price, block_height)             AS close,
         count(*)                              AS submission_count
  FROM oracle_submissions
  GROUP BY period, asset_id
WITH NO DATA;
SELECT add_continuous_aggregate_policy('oracle_price_1h',
  start_offset => 1800, end_offset => 600, schedule_interval => INTERVAL '10 minutes');

-- Validator uptime percentage per day
CREATE MATERIALIZED VIEW validator_uptime_1d
WITH (timescaledb.continuous) AS
  SELECT time_bucket(14400, block_height)      AS period,
         validator_address,
         AVG(signed::INT)::FLOAT * 100         AS uptime_pct,
         COUNT(*)                              AS blocks_in_window
  FROM validator_signatures
  GROUP BY period, validator_address
WITH NO DATA;
SELECT add_continuous_aggregate_policy('validator_uptime_1d',
  start_offset => 43200, end_offset => 14400, schedule_interval => INTERVAL '1 hour');

-- Bridge volume per direction per asset per hour
CREATE MATERIALIZED VIEW bridge_volume_1h
WITH (timescaledb.continuous) AS
  SELECT time_bucket(600, block_height)        AS period,
         direction,
         asset,
         SUM(amount)                           AS volume,
         COUNT(*)                              AS tx_count
  FROM bridge_events
  GROUP BY period, direction, asset
WITH NO DATA;
SELECT add_continuous_aggregate_policy('bridge_volume_1h',
  start_offset => 1800, end_offset => 600, schedule_interval => INTERVAL '10 minutes');
```

---

**Example API queries served directly from continuous aggregates:**

```sql
-- Explorer chart: TPS last 24h (hourly buckets)
SELECT period, tps_avg, tps_peak, total_txs
FROM tps_1h
WHERE period > $last_block - 14400
ORDER BY period ASC;

-- Explorer chart: block time p95 last 7 days
SELECT period, avg_ms,
       approx_percentile(0.95, pctls) AS p95_ms,
       approx_percentile(0.99, pctls) AS p99_ms
FROM block_time_1h
WHERE period > $last_block - (14400 * 7)
ORDER BY period ASC;

-- Oracle history: BTC/USD OHLC candles last 48h
SELECT period, open, high, low, close, submission_count
FROM oracle_price_1h
WHERE asset_id = 'BTC-USD'
  AND period > $last_block - 28800
ORDER BY period ASC;

-- Validator detail page: uptime trend last 30 days
SELECT period, uptime_pct, blocks_in_window
FROM validator_uptime_1d
WHERE validator_address = $addr
  AND period > $last_block - (14400 * 30)
ORDER BY period ASC;

-- Bridge volume: daily roll-up from hourly aggregate (no extra view needed)
SELECT time_bucket(14400, period) AS day,
       SUM(volume)                AS daily_volume,
       SUM(tx_count)              AS daily_txs
FROM bridge_volume_1h
WHERE direction = 'lock'
GROUP BY day
ORDER BY day DESC
LIMIT 30;

-- Dashboard: single query, all pre-computed
SELECT
  (SELECT SUM(total_txs)  FROM tps_1h         WHERE period > $now - 14400) AS txs_24h,
  (SELECT AVG(tps_avg)    FROM tps_1h         WHERE period > $now - 14400) AS avg_tps_24h,
  (SELECT SUM(volume)     FROM bridge_volume_1h WHERE direction='lock'
                               AND period > $now - 14400)                  AS bridge_lock_24h,
  (SELECT AVG(uptime_pct) FROM validator_uptime_1d
                               WHERE period = $current_day_bucket)         AS avg_uptime_today;
```

All queries above are sub-millisecond regardless of chain age — continuous aggregates are pre-computed; the API scans only the aggregate rows, never the raw event tables.

---

**What is removed from `module/projection` when this is adopted:**

| Removed from Go code | Replaced by |
|---|---|
| `bridge_volume_by_period` aggregate writer | `bridge_volume_1h` continuous aggregate |
| `validator_uptime` aggregate writer | `validator_uptime_1d` continuous aggregate |
| `oracle_participation_rate` aggregate writer | `oracle_price_1h` continuous aggregate |
| TPS calculation logic | `tps_1h` continuous aggregate |
| Block time p95 calculation | `block_time_1h` + `approx_percentile()` |
| Oracle OHLC calculation | `first()` / `last()` hyperfunctions in `oracle_price_1h` |
| **Aggregate singleton StatefulSet** | **Eliminated entirely** |
| **Advisory lock for aggregates** | **Eliminated entirely** |

`module/projection` becomes a single multi-replica fan-out **Deployment** (no StatefulSet, no advisory lock) handling only:
- `settlement_by_id`
- `milestone_status`
- `bridge_pending_by_nonce`

---

**docker-compose update for Read DB:**
```yaml
read-db:
  image: timescale/timescaledb:latest-pg16   # upgrade from postgres:16
  environment:
    POSTGRES_DB: read_db
    POSTGRES_USER: postgres
    POSTGRES_PASSWORD: ${READ_DB_PASSWORD}
  volumes:
    - read_db_data:/var/lib/postgresql/data
```

**Decision gate (Phase 5 ADR — same gate as Write DB):**
- [ ] TimescaleDB OSS licence acceptable for Read DB use case
- [ ] `first()` / `last()` hyperfunctions confirmed available in chosen managed service tier
- [ ] `percentile_agg` / `approx_percentile` confirmed available (require TimescaleDB Toolkit — included in timescale/timescaledb image, verify on managed service)
- [ ] Continuous aggregate refresh lag acceptable: default 10-minute policy means dashboard data is at most 10 minutes behind chain tip; if real-time is required, reduce `schedule_interval` to `INTERVAL '1 minute'` with `end_offset => 0`
- [ ] If TimescaleDB rejected for Read DB: revert to manually maintained aggregate tables in `module/projection` singleton StatefulSet as originally specified

#### 5.11 Celatone — CosmWasm Explorer

**Why:** Ping.pub (Phase 5.9) shows raw JSON for any CosmWasm contract interaction. Celatone is an open-source Cosmos explorer built specifically for CosmWasm chains (used by Osmosis, Neutron, Terra, Injective). It makes every contract in the system fully inspectable without any new backend — it reads the existing Cosmos REST and CometBFT RPC endpoints already running.

**What it covers that Ping.pub cannot:**

| Capability | Detail |
|---|---|
| Contract code uploads | Code ID, checksum, uploader, block height |
| Contract instantiations | Contract address, init message decoded, admin key |
| Contract executions | ExecuteMsg decoded against stored schema; response events |
| Contract queries | Interactive query execution from the explorer UI |
| Contract state | Raw KV state browser for any contract |
| Migration history | Code ID changes, migration messages |
| CW-20 token balances | Token holders, transfers, total supply |
| Contract admin | Current admin, admin transfer history |

**Contracts this exposes in your system:**
- Bridge contract — lock/release execution history, current pending queue
- Oracle contract — price submission executions, aggregation rounds
- Settlement contract — settlement lifecycle executions, state transitions
- Milestone contract — milestone update executions
- Certification contract — certification grant/revoke executions

**Deployment:**
```yaml
# Self-hosted — reads existing chain endpoints, no new backend
celatone:
  image: alleslabs/celatone-frontend:latest
  environment:
    NEXT_PUBLIC_CHAIN_ID:    "${CHAIN_ID}"
    NEXT_PUBLIC_RPC:         "https://${REPLIT_DEV_DOMAIN}/rpc"
    NEXT_PUBLIC_REST:        "https://${REPLIT_DEV_DOMAIN}/api/rest"
    NEXT_PUBLIC_GRAPHQL_URL: ""     # leave empty — uses REST only
  ports:
    - "3001:3000"
```
No indexer process. No separate database. No additional backend. Celatone is a Next.js frontend that queries chain endpoints directly.

**Routing:** expose at `/celatone` via Envoy; add path rule to `artifact.toml`.

**Audience:** dApp developers integrating with contracts, auditors inspecting contract state during Phase 9 audit, governance participants verifying contract upgrades.

**Limitations:**
- Does not show custom module (`x/settlement`, `x/bridge` etc.) state — only CosmWasm contract state
- Does not show analytics/charts — that is Phase 7.3
- Requires contract schemas (JSON Schema) uploaded alongside code for decoded message display; add `schema upload` step to contract deployment runbook in Phase 3

**Deliverable checklist:**
- [ ] Celatone deployed and accessible at `/celatone` on devnet
- [ ] All 5 CosmWasm contracts (bridge, oracle, settlement, milestone, certification) visible with decoded ExecuteMsg / QueryMsg
- [ ] JSON Schema uploaded for each contract alongside code upload (enables decoded message display)
- [ ] Contract state browser verified against known contract state on devnet
- [ ] Celatone added to auditor documentation package (Phase 8.3) as contract inspection tool

**Deliverable:** Full two-DB CQRS pipeline with correct NATS flow (ingestion→account:chain→projection→Read DB+account:stream→api→gRPC server-streaming→clients). CORS-enabled Envoy with separate gRPC-Web and REST upstreams. `/api/stream` WebSocket route absent — all streaming via `/api/grpcweb/*` only. grpc-gateway running as separate Deployment with independent HPA. Cursor-based pagination on all list RPCs. `x-wallet-address` header in SDK. Slow consumer eviction implemented and tested. Streaming auto-reconnect in SDK handles both Envoy restarts and `ResourceExhausted` evictions. Startup reconciliation and NATS back-fill tested.

**Phase 5 deliverable checklist additions:**
- [ ] TimescaleDB ADR decision confirmed before 5.2 implementation begins (covers both Write DB and Read DB; adopt or reject with documented rationale)
- [ ] If TimescaleDB adopted on Write DB: hypertable created, compression policy active, continuous aggregates for `bridge_volume_daily`, `validator_uptime`, `oracle_participation_rate` verified on devnet
- [ ] If TimescaleDB adopted on Read DB (Phase 5.10): hypertables for `block_stats`, `oracle_submissions`, `validator_signatures`, `bridge_events` created and verified
- [ ] If TimescaleDB adopted on Read DB: continuous aggregates `tps_1h`, `block_time_1h`, `oracle_price_1h`, `validator_uptime_1d`, `bridge_volume_1h` created with refresh policies; `first()`/`last()` and `approx_percentile()` hyperfunctions tested on devnet
- [ ] If TimescaleDB adopted on Read DB: TimescaleDB Toolkit availability confirmed on chosen managed service (required for `percentile_agg`)
- [ ] If TimescaleDB adopted: aggregate singleton StatefulSet removed from `module/projection`; `module/projection` is a fan-out Deployment only (no advisory lock, no StatefulSet)
- [ ] If TimescaleDB adopted: all dashboard/chart/analytics API endpoints verified returning data from continuous aggregates; response time < 10ms on devnet
- [ ] `synchronous_commit = off` set on Write DB and Read DB; `synchronous_commit = on` confirmed on Relayer DB
- [ ] `/api/stream` WebSocket route absent from Envoy config and k8s manifests
- [ ] All streaming via gRPC server-streaming through `/api/grpcweb/*` — verified from browser
- [ ] grpc-gateway deployed as separate Deployment (not sidecar); has its own HPA
- [ ] Cursor-based pagination implemented on all list RPCs; no offset pagination
- [ ] `x-wallet-address` gRPC metadata header documented in SDK README and set on all calls
- [ ] Slow consumer drop policy (channel buffer 64; `ResourceExhausted` on full buffer) implemented and integration-tested
- [ ] Envoy streaming timeouts (`timeout: 0s`, `idle_timeout: 600s`) active and verified
- [ ] `buf lint` CI check enforces `google.api.http` annotation on all query RPCs
- [ ] PgBouncer configured for `module/api` → Read DB connection pool (not deferred to optimization phase)
- [ ] **EVM routes (Phase 5.6 + Phase 2.9):** Envoy `/evm-rpc` route active and returning correct JSON-RPC responses; Envoy `/evm-ws` route active and accepting WebSocket connections; both routes have same CORS headers as `/api/*`
- [ ] **Blockscout (Phase 5.9 + Phase 2.9):** Blockscout container running and accessible at `/blockscout`; Blockscout DB (4th isolated PostgreSQL) running; Blockscout indexes all devnet EVM txs within 1 block; Solidity contract deployments visible and ABI-decoded in Blockscout

---

### Phase 6 — Devnet → Testnet (Weeks 13–19)

**Objective:** Multi-validator public testnet stable for 4 weeks before audit. Code freeze at Week 17 (2 weeks before stability window begins).

#### 6.1 Three-Ring Node Topology
```
[Public / dApp] → [Full Nodes / RPC Fleet] → [Sentry Nodes] → [Validator Nodes]
                                                                       ↓
                                                             [Horcrux Cluster]
```
- Chain cluster and backend cluster separate (WireGuard VPN cross-cluster)
- Network policies: no backend service has network access to validator nodes

#### 6.2 Horcrux Threshold Signing
- 2-of-3 threshold; private key split across 3 isolated signer machines
- Horcrux only — **no TMKMS** (they are alternative approaches; running both creates signing conflicts)
- Key generation ceremony: documented, auditable; inside HSM
- Double-sign protection verified under signer node failure and network partition

#### 6.3 Envoy Gateway Deployment
- 2 replicas, HPA, in backend cluster
- Explicit upstream clusters for each route (gRPC-Web backend, grpc-gateway separate Deployment, chain gRPC, CometBFT RPC, CometBFT WebSocket)
- **EVM routes active (Phase 5.6):** `/evm-rpc` → Ethermint JSON-RPC (port 8545, HTTP); `/evm-ws` → Ethermint WS-RPC (port 8546, WebSocket upgrade). Both routes verified on testnet before Phase 6.4 load test. CORS headers confirmed on EVM routes (same policy as `/api/*`).
- cert-manager manages all mTLS certificates (single CA, no Istio)
- CORS policy active and tested from browser
- Per-identity rate limiting verified: relayer bridge submissions not blocked

#### 6.4 Testnet Launch
- ≥ 5 independent external validator operators
- Genesis ceremony: gentxs → `genesis.json` → simultaneous start
- Load test: block production + bridge + oracle commit-reveal + governance concurrently; **also include EVM tx load: concurrent `MsgEthereumTx` (ETH transfers + Solidity contract calls) alongside Cosmos txs. Verify EVM txs don't starve Cosmos txs from block space and vice versa; measure per-runtime gas utilisation at peak load.**
- **Software upgrade drill (mandatory, before mainnet):** while testnet is running (Weeks 15–17), perform at least one full end-to-end upgrade drill: submit governance upgrade proposal → voting period → chain halts at upgrade height → validators swap binary → chain resumes at correct height. This is the only way to verify the `x/upgrade` handler, the binary swap procedure, and the validator runbook work correctly. A mainnet upgrade that has never been drilled on testnet is a chain-halt risk.
- **Code freeze at Week 17:** no binary changes after Week 17 except critical security fixes (which trigger planned testnet upgrade)

#### 6.5 Chaos Test Suite

**Chain chaos:** kill validator nodes; Horcrux signer failure; double-sign protection verified

**NATS chaos:**
- Kill nats-0 → nats-1/nats-2 continue; no message loss
- Kill all 3 simultaneously → bring back → consumer offsets correct; back-fill from Write DB works
- NATS outage during active bridge event → ingestion writes `nats_published=false` → NATS recovers → back-fill → Read DB consistent; no manual intervention

**Bridge chaos:**
- Kill designated submitter after quorum → promotion ladder activates → single next-in-line submitter sends `MsgBridgeIn` → no duplicate submissions, no wasted gas
- Circuit-breaker pause: < 60s from alert to confirmed pause; Gnosis Safe unpause
- Supply cap breach attempt: verify rejection

**Backend chaos:**
- Kill ingestion → restart → advisory lock → startup reconciliation fills gap → no Write DB holes
- Kill projection aggregate → restart → advisory lock → NATS replay → Read DB aggregates consistent
- Kill projection KV replica → fan-out restart → idempotent updates → no inconsistency
- gRPC server-streaming Envoy restart → SDK auto-reconnect → client resumes without user intervention

**EVM chaos:**
- Kill Ethermint JSON-RPC server (disable `[json-rpc] enable = true` in `app.toml`, restart node) → `/evm-rpc` returns 503 → re-enable → MetaMask reconnects automatically within 30s → no EVM state loss
- EVM tx flood: submit 1,000 `MsgEthereumTx` rapidly from a test account → verify Cosmos txs (oracle commits, bridge submissions) are not evicted from the mempool; verify `MaxTxGasWanted` cap is enforced per EVM tx; verify block production continues without halt
- Blockscout DB crash (kill blockscout-db container) → Blockscout goes offline → Blockscout DB restarted → Blockscout re-indexes from the last indexed block → no EVM tx history lost (source of truth is the chain, not Blockscout)
- EVM ante handler with malformed tx: submit a `MsgEthereumTx` with incorrect EVM chain ID (replay from another chain) → verify rejected with `ErrInvalidChainID`; EIP-155 enforcement confirmed
- CosmWasm + EVM in same block under load: 50 `MsgEthereumTx` + 50 `MsgExecuteContract` in same block → verify both sets execute; verify StateDB and CosmWasm KV store are consistent post-block

#### 6.6 Monitoring Stack
- Prometheus: chain metrics, attestation coverage, oracle staleness, bridge queue, milestone status, NATS cluster health, `nats_published=false` row count (alert if > 0 for > 1 min), rewards bucket runway, node hardware
- **EVM-specific Prometheus metrics (exposed by Ethermint):**
  - `ethermint_evm_tx_count` — EVM txs per block (track load)
  - `ethermint_evm_block_gas_used` — gas used by EVM per block; alert if consistently > 80% of `MaxTxGasWanted × txs_per_block`
  - `ethermint_jsonrpc_requests_total` — JSON-RPC request volume; alert on sudden spike (scraping bot) or sudden drop (JSON-RPC server unhealthy)
  - `ethermint_jsonrpc_errors_total` — JSON-RPC error rate; alert if error rate > 5% sustained for 2 min (excluding expected method-not-found errors)
  - `blockscout_indexing_lag_blocks` — exported from Blockscout health endpoint; alert if Blockscout is > 10 blocks behind chain tip (indicates Blockscout DB or JSON-RPC connectivity issue)
- Grafana dashboards: chain health, bridge, oracle, milestones, NATS, Read DB vs Write DB lag; **add EVM dashboard: EVM tx count, EVM block gas usage, JSON-RPC request rate, Blockscout indexing lag**
- Alerting: chain halt, double-sign risk, bridge stuck event, oracle staleness breach, NATS node failure, rewards bucket < 6-month runway, ingestion crash (advisory lock not held), `nats_published` backlog growing; **add: Ethermint JSON-RPC error rate > 5%; Blockscout indexing lag > 10 blocks; EVM block gas consistently > 80% cap (indicates need to tune `MaxTxGasWanted`)**

**Deliverable:** Stable testnet from Weeks 15–19 (4 weeks); code frozen Week 17; all chaos tests passed (including EVM chaos suite); zero chain halts after code freeze. Blockscout accessible on testnet; MetaMask connects and EVM txs are confirmed end-to-end. EVM monitoring dashboards live and alerting verified.

---

### Phase 7 — Wallet Configuration, Frontend & Client SDK (Weeks 15–19)

#### 7.1 Wallet Configuration
- **Keplr / Leap — Cosmos chain config:** `chainId`, `chainName`, `rpc`, `rest`, `bip44.coinType = 60` (Ethereum BIP-44 path, required for dual-address compatibility), `bech32Config`, `currencies`, `feeCurrencies` (using `utoken`, `gasPriceStep`), `stakeCurrency`. Submit PR to [cosmos/chain-registry](https://github.com/cosmos/chain-registry) before testnet launch.
- **MetaMask — Sovereign chain EVM config** (the chain's own EVM, NOT the BSC config):
  ```json
  {
    "chainId": "0x<EVM-chain-id-hex>",
    "chainName": "MyChain",
    "nativeCurrency": { "name": "TOKEN", "symbol": "TOKEN", "decimals": 18 },
    "rpcUrls": ["https://<domain>/evm-rpc"],
    "blockExplorerUrls": ["https://<domain>/blockscout"]
  }
  ```
  `decimals: 18` matches `x/evm` `BaseDenom = "atoken"`. Users see TOKEN balances correctly. Provide a one-click "Add to MetaMask" button in the dApp using `wallet_addEthereumChain`.
- **MetaMask — BSC config** (existing BSC bridge config — unchanged): standard BSC mainnet / testnet RPC. Separate from the sovereign chain EVM config above.
- **WalletConnect:** register chain metadata; test with WalletConnect v2 on testnet
- **Wallet setup guide:** document both the Keplr (Cosmos side) and MetaMask (EVM side) configurations explicitly. Users need to add **two** network configs — one for each runtime. This is a user-facing communication task as much as a technical one; draft the guide before testnet launch so validators can test it.

#### 7.2 dApp — Next.js
- **Cosmos-side interactions:** CosmJS via Envoy `/grpc/*` (unchanged)
- **All custom module data:** gRPC-Web via Envoy `/api/grpcweb/*` (unchanged)
- **Real-time streams:** gRPC server-streaming via `/api/grpcweb/*` with SDK auto-reconnect (no `/api/stream` WebSocket — removed; all real-time data through gRPC server-streaming only)
- **EVM-side interactions — wagmi / viem:**
  - Add `wagmi` v2 and `viem` v2 to the Next.js dApp for EVM wallet connection and contract interaction
  - Configure wagmi `chain` object using the EVM chain ID from Phase 0 ADR and the `/evm-rpc` and `/evm-ws` endpoints
  - Connect to MetaMask (and other EVM wallets) via wagmi's `useAccount`, `useConnect`, `useDisconnect`
  - **EVM pages in the dApp:**
    - EVM account page: show ETH balance (in `atoken` / TOKEN), EVM address (hex), and link to Blockscout for full tx history
    - ERC-20 token page: show native token ERC-20 balance (registered via `x/erc20`); allow ERC-20 → native bank conversion
    - EVM contract interaction page (post-launch, once Solidity contracts are deployed): use `useContractRead` / `useContractWrite` wagmi hooks with ABI from `/evm` directory
  - **CosmJS + wagmi coexistence:** the dApp supports both wallet types simultaneously. A user with Keplr (Cosmos) and MetaMask (EVM) can interact with both runtimes from the same page. Do NOT force users to choose — display both address formats in the wallet connection widget.
  - Use wagmi's `useWatchBlockNumber` (WebSocket via `/evm-ws`) for live EVM block number display
- **Bridge page:** displays tiered confirmation status (standard vs. large transfer); live countdown
- **Governance page:** gas limit param display so users understand Constitution check cost

**Deliverable:** dApp on testnet with both Cosmos (CosmJS + gRPC-Web) and EVM (wagmi/viem) interactions verified; MetaMask connects to `/evm-rpc`; streaming auto-reconnect tested; EVM account page shows balance.

#### 7.3 Custom Analytics Dashboard

**Why:** Ping.pub covers infrastructure. Celatone covers contracts. Neither covers analytics, custom module UIs, or the TimescaleDB aggregate data built in Phase 5.10. This dashboard is the only frontend that surfaces TPS charts, oracle price history, bridge volume, validator uptime trends, settlement lifecycle, and milestone/certification status — all data that already exists in the CQRS API.

**This is not a new backend.** Every data point below is already served by `module/api` (Phase 5.5). The dashboard is a Next.js page added to the existing dApp — a new `/dashboard` route — consuming the gRPC-Web and REST API already built.

**Pages and data sources:**

```
/dashboard
├── Chain Overview
│     ├── Live TPS           → GetTps RPC          → tps_1h continuous aggregate
│     ├── TPS chart (24h)    → StreamChainStats RPC → tps_1h time series
│     ├── Block time p95     → GetBlockStats RPC    → block_time_1h + approx_percentile
│     ├── Active validators  → GetValidatorSet RPC  → standard Cosmos state
│     └── Total txs (24h)    → GetTps RPC           → tps_1h.total_txs
│
├── Bridge
│     ├── Volume chart       → GetBridgeVolume RPC  → bridge_volume_1h
│     ├── Pending queue      → GetBridgePending RPC → bridge_pending_by_nonce projection
│     ├── Per-tx status      → GetBridgeTx RPC      → settlement_by_id + BSC confirmation
│     └── Lock / Release     → direction split       → bridge_volume_1h.direction
│
├── Oracle
│     ├── Price OHLC chart   → GetOraclePrice RPC   → oracle_price_1h (first/last)
│     ├── Asset selector     → asset_id param        → per-asset filtering
│     └── Participation rate → GetOracleStats RPC   → oracle_price_1h.submission_count
│
├── Validators
│     ├── Uptime table       → GetValidators RPC    → validator_uptime_1d current period
│     ├── Uptime trend chart → GetValidatorUptime   → validator_uptime_1d time series
│     └── Missed blocks      → GetValidatorMissed   → validator_signatures hypertable
│
├── Settlement
│     ├── Pending list       → ListSettlements RPC  → settlement_by_id WHERE status=pending
│     ├── Settlement detail  → GetSettlement RPC    → settlement_by_id projection
│     └── Lifecycle display  → status field         → pending → confirmed → finalized
│
└── Milestones & Certifications
      ├── Milestone timeline → ListMilestones RPC   → milestone_status projection
      └── Certification list → ListCertifications   → certification projection
```

**Real-time updates:** all pages with live data use gRPC server-streaming via `StreamChainStats`, `StreamBridgeEvents`, `StreamOraclePrice` RPCs — same streaming infrastructure built in Phase 5.5. No polling. SDK auto-reconnect handles Envoy restarts transparently.

**Implementation notes:**
- Route: `/dashboard/*` in the existing Next.js dApp (`artifacts/web`)
- Data: import generated gRPC-Web hooks from `@workspace/api-spec` codegen output (already generated in Phase 5.1)
- Charts: Recharts or Victory — lightweight, no canvas dependency, SSR-safe
- No new gRPC RPCs needed if Phase 5.5 already defines `GetBridgeVolume`, `GetOraclePrice`, `GetValidatorUptime`, `StreamChainStats` — verify against `backend/v1/query.proto` and add missing RPCs before Phase 7.3 begins

**New RPCs to add to `backend/v1/query.proto` if not already present:**
```protobuf
// Analytics RPCs — all served from TimescaleDB continuous aggregates
rpc GetTps             (GetTpsRequest)              returns (GetTpsResponse);
rpc GetBlockStats      (GetBlockStatsRequest)        returns (GetBlockStatsResponse);
rpc GetBridgeVolume    (GetBridgeVolumeRequest)      returns (GetBridgeVolumeResponse);
rpc GetOraclePrice     (GetOraclePriceRequest)       returns (GetOraclePriceResponse);
rpc GetValidatorUptime (GetValidatorUptimeRequest)   returns (GetValidatorUptimeResponse);
rpc StreamChainStats   (StreamChainStatsRequest)     returns (stream ChainStatsEvent);
```
Run `pnpm --filter @workspace/api-spec run codegen` after adding; generated hooks available immediately in the dApp.

**Deliverable checklist:**
- [ ] `/dashboard` route accessible in dApp on testnet
- [ ] TPS chart rendering live data from `tps_1h` continuous aggregate
- [ ] Oracle OHLC chart rendering per-asset price history from `oracle_price_1h`
- [ ] Bridge volume chart rendering from `bridge_volume_1h`; pending queue live-updating via streaming
- [ ] Validator uptime trend chart verified against `validator_uptime_1d`
- [ ] Settlement lifecycle display: pending → confirmed → finalized transitions visible in real time
- [ ] All chart data confirmed sourced from TimescaleDB continuous aggregates (not raw table scans)
- [ ] Dashboard loads in < 2s cold on testnet; chart queries < 10ms at the DB layer

**Deliverable:** dApp on testnet, all routing verified, streaming auto-reconnect tested. Custom analytics dashboard at `/dashboard` rendering live chain, bridge, oracle, validator, settlement, and milestone data from CQRS API and TimescaleDB continuous aggregates.

---

### Phase 8 — Reproducible Builds & Pre-Audit Hardening (Weeks 18–21)

#### 8.1 Reproducible Builds
- goreleaser: pinned Go toolchain, deterministic ldflags, CGO disabled
- Docker: multi-stage build, pinned base image digest
- `make verify-build`: two independent machines produce identical SHA256
- cosign signing; SLSA provenance published

#### 8.2 Code Hardening
- `golangci-lint`: zero warnings; `govulncheck`: zero vulnerabilities; `staticcheck`: zero issues
- `go test -race ./...`: zero data races
- Cosmos SDK standard invariant suite + custom: supply cap, validator cardinality, oracle staleness state machine, nonce bitmap consistency, rewards bucket balance, `x/certification` window consistency, `nats_published` consistency (no event written to Write DB without eventual NATS publish)
- Rust: `cargo clippy -- -D warnings` zero warnings; `cargo audit` zero vulnerabilities
- **Simulation coverage:** confirm all seven custom modules' `WeightedOperations` are registered and executed during `--NumBlocks=5000 --BlockSize=50 --Seed=<random>`; governance-gated operations use `SimGovProposalMsg` wrapper

#### 8.3 Documentation for Auditors
- Architecture: two-DB CQRS pipeline with NATS flow diagram (ingestion→account:chain→projection→Read DB+account:stream→api), Envoy routing (separate gRPC-Web, REST, EVM JSON-RPC, and EVM WebSocket clusters), Kubernetes topology (two clusters, WireGuard VPN, no Istio), database user permission matrix (four databases: Write DB, Read DB, Relayer DB, Blockscout DB)
- State machines: `x/certification` (degraded mode as chain-state, not per-validator local), `x/milestone` (deadline clock pause/resume), `x/oracle` (commit-reveal, insufficient round), `x/bridge` (bitmap nonce, tiered confirmation, promotion ladder)
- `x/staking` compatibility analysis: `x/distribution`, `x/gov`, `x/slashing`, IBC `HistoricalInfo`
- CosmWasm authority graph: cold multi-sig key holder identities (published publicly), `EmergencyPause` scope (ExecuteMsg only), Governance contract replacement procedure, fund migration procedure
- Bridge threat model (from Phase 0 ADR)
- NATS account isolation: NKey credentials, subject namespaces, back-fill mechanism, retention policies
- Genesis supply math: S-C derivation, invariant verification in genesis script
- Horcrux setup, key ceremony, key rotation procedures
- `x/authz` blocked message types list with rationale: `MsgBridgeIn`, `MsgBridgeOut`, `MsgSubmitOracleCommit`, `MsgRevealOracleReport`, `MsgSettlement`, `/ethermint.evm.v1.MsgEthereumTx`
- PostgreSQL multi-region replication topology; backup/PITR setup; partition scheme; **TimescaleDB extension configuration** (hypertable chunk interval, compression policy, continuous aggregate refresh policies, `first()`/`last()` and `percentile_agg()` hyperfunction availability — auditors should understand these are DB-maintained, not application-maintained aggregates)
- **Celatone** listed as the primary tool for CosmWasm contract state inspection during audit. Auditors should use Celatone (accessible at `/celatone`) to: inspect contract code uploads and instantiation history, query decoded ExecuteMsg/QueryMsg against stored JSON Schema, browse raw contract KV state, verify migration history. JSON Schema upload procedure (Phase 3) ensures decoded message display works for all four CosmWasm contracts (Constitution, Treasury, Reserve Fund, Governance).
- **EVM layer documentation (Scope E — see Phase 9.2):**
  - Ethermint version pinned in `go.mod`; exact commit/tag recorded in ADR
  - Module initialization order diagram: `x/feemarket` → `x/evm` → `x/erc20` → `wasmd` (wasmd position shown)
  - Ante handler routing diagram: `MsgEthereumTx` path (EVM decorator chain) vs. all other messages (Cosmos decorator chain); explain that routing is by message type, not by tx type hint
  - EVM denomination document: `BaseDenom = "atoken"` (18 decimals); `utoken` (6 decimals) for Cosmos side; `x/erc20` conversion mechanism; MetaMask display values
  - EVM chain ID: value, registration proof (chainlist.org), immutability rationale
  - `x/erc20` token pair registrations: list all registered pairs at genesis; conversion math
  - JSON-RPC endpoint security: which `api` namespaces are exposed (`eth`, `net`, `web3`, `txpool`, `debug`); `debug_` exposure is intentional for devnet only; confirm `debug_` is disabled in mainnet `app.toml`
  - Precompile policy: none at mainnet; stubs in `/evm` marked `// post-launch`; confirm no precompile registered in the mainnet binary
  - CosmWasm + EVM coexistence test results: `TestCosmWasmEVMCoexistence` test report; five consecutive green runs on testnet
  - **Blockscout** listed as the EVM chain inspection tool for auditors. Auditors should use Blockscout (at `/blockscout`) to: verify EVM tx inclusion and gas consumption, inspect Solidity contract deployments, query ERC-20 transfer events, verify EVM account balances. Blockscout is a read-only view of the Ethermint JSON-RPC — it does not affect chain state.

#### 8.4 Key Rotation and Emergency Runbooks

**Oracle operator key rotation:**
1. Submit `UpdateOracleOperator` governance proposal with new Ed25519 public key
2. Both old and new keys valid during 7-day voting period
3. After passage: old key removed from HSM; new key sole valid key; slashing suspended for rotating operator during voting period

**Witness key rotation:**
1. Submit `UpdateWitnessRegistry` governance proposal with new Ed25519 public key
2. In-flight settlements signed with old key: 24-hour grace window after passage (governance parameter)
3. After grace window: old key revoked

**Relayer key rotation:**
1. Submit `UpdateBridgeRelayerSet` governance proposal (add new key, remove old key)
2. Both keys may count toward quorum during voting period
3. After passage: old key removed

**Circuit-breaker EOA key compromise runbook:**
1. On-call engineer detects compromise (unauthorized pause or key theft)
2. Immediately contact all Gnosis Safe signers (defined contact list in runbook, with phone numbers and backup contacts)
3. Gnosis Safe transaction: call `setCircuitBreaker(newEOAAddress)` to rotate the pause-only key
4. Target time to key rotation: < 4 hours from detection (requires Gnosis Safe quorum; contact list kept current and drilled quarterly)
5. During rotation window: bridge is vulnerable to repeated pauses by attacker (no funds at risk; availability only). If unacceptable, Gnosis Safe can pause bridge permanently until rotation is complete.
6. Post-rotation: incident report; review whether attacker paused and if any funds were affected

**Cold multi-sig key holder procedures:**
- Holder set publicly declared in ADR and on-chain (query from `x/governance-ext`)
- Annual rotation drill: verify all 7 keys are accessible; replace any lost/compromised key via `MsgMigrateContracts` governance proposal (7-day time-lock)
- If a holder's key is compromised: emergency rotation via the remaining holders (5-of-6 remaining can initiate a `MsgMigrateContracts` proposal to update the holder set)

**CosmWasm governance contract replacement (in case of critical bug):**
1. Cold multi-sig pauses Treasury and Reserve Fund (`EmergencyPause{}` → blocks `ExecuteMsg`, not `QueryMsg`)
2. Submit `MsgMigrateContracts` governance proposal (bypasses Constitution check; 7-day time-lock)
3. On execution: new Governance contract instantiated; cross-contract authority on all three contracts updated to new Governance address; cold multi-sig unpauses Treasury and Reserve Fund

**PostgreSQL backup restore drill (quarterly):**
1. Take latest backup + WAL archive
2. Restore to isolated environment
3. Verify all three databases (Write DB, Read DB, Relayer DB) restore successfully
4. Run CQRS pipeline against restored Write DB; verify Read DB catches up correctly
5. Document restore time (target: < 4 hours for full restore)

#### 8.5 Internal Penetration Test

**Pass criteria:** all scenarios tested; zero critical or high findings open before external audit. Severity: Critical/High/Medium/Low. Report signed by lead security engineer.

**Scenarios:**
- Bridge: replay (old nonce via bitmap), supply cap bypass, relayer collusion, circuit-breaker denial
- Promotion ladder: two relayers simultaneously receive quorum → verify only one submits
- `x/certification`: degraded mode manipulation (attempt to trigger from ProcessProposal locally → verify chain-state check prevents it), bootstrapping window manipulation
- `x/settlement`: wrong chain_id in domain separator → rejection; timestamp outside tolerance → rejection; expired witness key reuse
- `x/oracle`: commit-without-reveal slash; round with insufficient commits; stale-value reveal (hash matches but economically stale — verify outlier rejection if value is statistical outlier)
- `x/milestone`: staleness weaponization → deadline paused → verify milestone cannot be forced to expire by oracle staleness
- `x/authz`: grant attempt for `MsgBridgeIn` → verify protocol-level rejection; grant attempt for `MsgEthereumTx` → verify protocol-level rejection (EVM authz block from Phase 0 ADR)
- CosmWasm: unauthorised caller; gas exhaustion (verify full revert); cold multi-sig `EmergencyPause` → verify `QueryMsg` still works; Constitution paused → governance proposals still execute (query succeeds)
- NATS: cross-account publishing attempt → verify account isolation rejects it
- CQRS: dual-write attempt (kill advisory lock externally → second instance starts → verify lock prevents write)
- Envoy CORS: cross-origin request from unauthorized domain → verify rejection; authorized domain → verify success
- **EVM attack scenarios:**
  - EVM replay attack: submit `MsgEthereumTx` signed for a different EVM chain ID → verify rejected with `ErrInvalidChainID` (EIP-155 enforcement); submit same EVM tx hash twice → verify nonce rejection (replay protection)
  - EVM gas manipulation: submit `MsgEthereumTx` with `gasPrice < baseFee` → verify rejected by EVM ante handler (EIP-1559 min fee enforcement); submit with `gasLimit > MaxTxGasWanted` → verify rejected
  - EVM block space starvation: flood block with 500 `MsgEthereumTx` from the same account → verify Cosmos txs (oracle commits, bridge) still included in the same block (EVM and Cosmos txs share block gas limit but Cosmos mempool has independent priority)
  - EVM → CosmWasm cross-runtime exploit attempt: deploy a Solidity contract that calls a `0x08xx` address (precompile stub range) → verify no precompile registered (returns nothing / reverts) because precompiles are post-launch only
  - EVM `EnableCreate` governance change attempt: attempt to toggle `EnableCreate` via governance to disable contract deployment → verify parameter change takes effect at next block; re-enable and verify contracts can deploy again (governance path tested, not locked)
  - EVM denomination: attempt to send `MsgEthereumTx` with value denominated incorrectly (e.g. wrong decimal precision) → verify ante handler rejects or truncates correctly; MetaMask-signed tx with correct 18-decimal value → verify correct `x/bank` balance change

---

### Phase 9 — External Security Audit (Weeks 21–24, pre-engaged from Week 14)

> **Timeline risk notice:** The 24-week plan (Phases 0–10) assumes zero critical audit findings and no scope changes. Cosmos chain audits at this complexity typically produce critical/high findings that require 4–8 weeks of remediation plus re-review. If critical findings emerge in Week 23, mainnet launch (Phase 10, Week 25) must be delayed. **Get explicit client sign-off on this assumption in Phase 0 before the timeline is treated as binding.** A Phase 9-B remediation buffer of 4 weeks (Weeks 25–28) should be planned as a contingency; Phase 10 shifts to Week 29 in that scenario.

#### 9.1 Auditor Selection
- **Auditor 1:** Cosmos SDK / Go chain specialist (Informal Systems, Zellic, Oak Security) — **covers Scopes A + B** (Go chain + CosmWasm)
- **Auditor 2:** Solidity / EVM specialist (Trail of Bits, Halborn, Zellic, Spearbit) — **covers Scope C + Scope E** (BSC bridge + EVM integration). Auditor 2 must have prior Ethermint or EVM-on-Cosmos experience (Evmos, Sei, Injective audits are qualifying examples); a firm with only pure EVM (Ethereum L1/L2) experience is insufficient — the CosmWasm coexistence and ante handler ordering are Cosmos-specific concerns. If no single firm covers both C and E, split: engage a second Solidity firm for Scope C and assign Scope E to Auditor 1 as additional scope (both are Go-level concerns at the integration layer).
- **Auditor 3 (or internal red team):** Infrastructure / distributed systems specialist — **covers Scope D** (NATS, CQRS, PostgreSQL, TimescaleDB). If no external firm with this expertise is available in the timeframe, Scope D is red-teamed internally in Phase 8.5 with documented findings and is explicitly called out as not covered by external audit in the published report.
- **Ethermint background package for all auditors:** provide Evmos mainnet audit reports (from Trail of Bits and Informal Systems; publicly available) as context. These cover the Ethermint EVM engine itself. Auditors reviewing Scope E are NOT expected to re-audit the engine — only the integration surface specific to this chain.
- Pre-engage by Week 14; testnet access provided immediately; 7 weeks of async context before kickoff

#### 9.2 Audit Scope
- **Scope A — Go chain:** all custom modules, `x/staking` compatibility shim, `x/distribution` override, genesis config, ABCI++ hooks, degraded mode chain-state design, `x/authz` blocked message types
- **Scope B — CosmWasm:** all four contracts, authority graph, gas limits, `EmergencyPause` scope (ExecuteMsg only), cold multi-sig holder set, Governance replacement procedure, fund migration, cold multi-sig key holder identities
- **Scope C — Bridge:** BSC LockBox (keccak256 nonce generation, bitmap, tiered confirmation, circuit-breaker), `x/bridge`, relayer (deterministic promotion ladder, NATS coordination, Relayer DB)
- **Scope D — Infrastructure:** NATS account isolation, retention policies, `nats_published` back-fill, ingestion singleton (advisory lock, startup reconciliation), projection aggregate singleton (or TimescaleDB continuous aggregates if adopted), CQRS DB user permission matrix, **TimescaleDB hypertable correctness** (chunk compression does not corrupt data; continuous aggregate refresh produces correct values under concurrent hypertable writes; `first()`/`last()` hyperfunction results match raw data; compression policy does not affect PITR or advisory lock behaviour)
- **Scope E — EVM (Ethermint):** `x/evm` module wiring and genesis params (EVM chain ID correctness, `AllowUnprotectedTxs = false`, EVM denom configuration); `x/feemarket` EIP-1559 fee accounting for EVM txs (base fee calculation, gas refunds, priority fee handling); `x/erc20` token pair registration and bi-directional conversion logic (Cosmos → EVM and EVM → Cosmos); ante handler routing correctness (EVM txs must not pass through Cosmos ante decorators and vice versa); `MsgEthereumTx` authz block enforcement; StateDB isolation (EVM state changes must not corrupt `x/bank` or CosmWasm contract storage and vice versa); JSON-RPC endpoint security (`debug_` namespace disabled in mainnet `app.toml`); EVM denomination immutability (confirm no governance path to change `BaseDenom` post-genesis). **Note: Ethermint's core EVM execution engine is not re-audited — Scope E covers the integration surface only (wiring, params, ante handler, denomination mapping, StateDB isolation). The Evmos mainnet audit reports for Ethermint itself should be provided to auditors as background context.**
- **Out of scope:** standard upstream SDK modules unless directly referenced by custom logic; Ethermint core execution engine (covered by Evmos audit history)

#### 9.3 Audit Process
- Week 21: kickoff, codebase walkthrough, documentation review
- Weeks 21–22: active audit execution (two or three firms simultaneously)
- Week 23: preliminary findings; remediation sprint begins immediately
- Weeks 23–24: remediation of all critical and high; re-review
- Week 24 end: final audit report

#### 9.4 Mainnet Gate Criteria
- Zero unresolved critical findings
- Zero unresolved high findings
- All medium findings resolved or formally accepted with written rationale
- Final report published before mainnet genesis

---

### Phase 10 — Mainnet Launch (Week 25)

**Objective:** Audited mainnet live, bridge active, monitoring operational.

#### 10.1 Genesis Preparation
- Finalise parameters: supply S, Cosmos allocation S-C, bridge escrow C, validator slots, contract addresses, module params
- Each validator independently runs genesis script invariant checks before signing
- Genesis file: minimum 1-week review window; three independent manual verifications of supply math
- Validator key generation via Horcrux ceremony; `gentx` submission
- **EVM genesis parameter verification (mandatory — chain team + at least 2 independent verifiers):**
  - `x/evm` `params.evm_denom = "atoken"` — confirm matches ADR; immutable post-genesis
  - `x/evm` `params.chain_config.chain_id` = EVM chain ID (as decimal) — confirm matches chainlist.org registration; immutable post-genesis
  - `x/evm` `params.allow_unprotected_txs = false` — EIP-155 enforced from block 1
  - `x/evm` `params.enable_create = true`, `params.enable_call = true` — EVM contract deployment open from block 1
  - `x/feemarket` `params.no_base_fee = false` — EIP-1559 active from block 1
  - `x/feemarket` `params.base_fee` > 0 — non-zero base fee prevents spam from block 1
  - No custom precompiles registered in genesis — confirm `/evm` directory stubs not compiled in
  - `x/erc20` initial token pairs — confirm only approved pairs listed; no unexpected ERC-20 registrations
  - Run: `chaind validate-genesis --home ~/.chain-mainnet` passes with EVM modules active

#### 10.2 Staged Rollout
- **Day 0:** Chain starts; ≥ 2/3 validators producing blocks
- **Days 1–3:** Internal validation (supply, contracts, oracle commit-reveal on mainnet, NATS back-fill verified, monitoring alerts tested)
- **Day 1 (EVM verification — parallel with Cosmos validation):**
  - Verify Ethermint JSON-RPC returns correct chain block number: `cast block-number --rpc-url https://<domain>/evm-rpc`
  - Submit internal test EVM transfer; confirm in Blockscout within 1 block
  - Verify Blockscout indexing lag ≤ 2 blocks; confirm monitoring alert firing correctly
  - Confirm `debug_` namespace absent from JSON-RPC: `curl -X POST .../evm-rpc -d '{"method":"debug_traceBlock",...}'` returns method-not-found
  - Verify MetaMask "Add to MetaMask" button in dApp correctly adds the mainnet EVM chain config
- **Day 3:** Validator delegations and governance open; **EVM JSON-RPC endpoint made public** (`/evm-rpc` and `/evm-ws` Envoy routes open to public traffic; Blockscout at `/blockscout` publicly accessible)
- **Week 2:** Bridge activation (circuit-breaker EOA live; small test transfer verified before general availability; rate limit active from block 1)
- **Week 3:** Full public access; **MetaMask integration guide published**; dApp EVM account and ERC-20 pages live; wagmi/viem endpoints verified against mainnet

#### 10.3 Production Infrastructure
- Multi-region k8s (≥ 2 regions): chain cluster and backend cluster both multi-region
- **PostgreSQL instances (four, each isolated):**
  - Write DB: primary + synchronous standby in separate regions; Patroni failover; PITR active
  - Read DB: primary + async standby; PITR active; TimescaleDB extension if adopted
  - Relayer DB: primary + async standby; `synchronous_commit = on` enforced
  - **Blockscout DB:** primary + async standby; PITR active. Blockscout DB failure is non-critical (chain continues, Blockscout goes offline); restore SLA target: < 2 hours. Blockscout re-indexes from last known block on restart — no manual data repair needed as long as the chain's JSON-RPC history is complete.
- Monitoring stack live and alert-tested before chain start; **EVM Grafana dashboard live and all five EVM alerts verified firing correctly in staging before mainnet**
- On-call rotation with escalation paths; **include Blockscout DB and EVM JSON-RPC in on-call runbook** (Blockscout restart procedure, JSON-RPC server recovery)

#### 10.4 Post-Launch Operations
- 30 days heightened monitoring; daily bridge volume review; weekly rewards bucket runway check
- Bug bounty (Immunefi); **explicitly include EVM contract interactions and EVM ante handler bypass in scope**
- Monthly security review
- Upgrade procedure: testnet run → governance proposal (2-week notice minimum) → mainnet upgrade
- Every upgrade: named `x/upgrade` handler (no-op if no migrations; must exist); **EVM module migrations: if upgrading Ethermint version, verify `x/evm`, `x/feemarket`, `x/erc20` migration handlers are registered alongside custom module handlers — missing EVM migration handler causes chain halt at upgrade height**
- Quarterly: Horcrux key accessibility drill; circuit-breaker EOA contact list drill; **EVM block gas utilisation review** (check if `MaxTxGasWanted` needs governance adjustment based on 3-month traffic data)
- Annual: key rotation review (oracle operators, witnesses, relayers); cold multi-sig holder audit; PostgreSQL backup restore drill; **Blockscout DB restore drill** (verify Blockscout can re-index from JSON-RPC after restore)
- **Post-launch EVM roadmap (governance-gated, not in scope for mainnet launch):**
  - `x/oracle` precompile at `0x0000000000000000000000000000000000000801` — allows Solidity contracts to read oracle price; requires governance proposal + audit of precompile code before activation
  - `x/milestone` precompile at `0x0000000000000000000000000000000000000802` — allows Solidity to query milestone status
  - Additional `x/erc20` token pair registrations — each new ERC-20 ↔ native token pair requires governance proposal
  - Solidity contracts in `/evm` directory (post-audit) — deployed via governance or team key depending on contract type

---

## Validator Onboarding Guide

This guide is the document you hand to every external validator operator. It covers hardware, software, network topology, Horcrux setup, security requirements, and the exact steps to join each network. Validators sign off the checklist at the end before being admitted to the active set.

---

### Node Types and Roles

Every validator operator runs four distinct machine types. They must never be combined on the same machine.

```
Internet
   ↓
[Full Node / RPC Node]      — public-facing; serves API queries; no validator key
   ↓ (private network)
[Sentry Node × 2]           — DDoS shield; peers only with validator node
   ↓ (private network only)
[Validator Node]            — produces and votes on blocks; connects to Horcrux only
   ↓ (isolated signing network)
[Horcrux Signer × 3]        — holds 1 key shard each; never reachable from internet
```

The validator node must have **zero direct internet exposure**. All inbound P2P traffic arrives via sentry nodes. All signing requests go to Horcrux signers over a dedicated private network.

---

### Hardware Requirements

#### Full Node / RPC Node
| Resource | Minimum | Recommended |
|---|---|---|
| CPU | 4 cores | 8 cores (x86_64) |
| RAM | 16 GB | 32 GB |
| Storage | 500 GB NVMe SSD | 2 TB NVMe SSD (archive: no pruning) |
| Network | 100 Mbps | 1 Gbps |
| OS | Ubuntu 22.04 LTS | Ubuntu 22.04 LTS |

**Full Node / RPC Node — EVM JSON-RPC configuration (mandatory for public RPC nodes):**

Full nodes that serve public traffic (dApp users, MetaMask, Blockscout) must enable the Ethermint JSON-RPC server in `app.toml`:
```toml
[json-rpc]
enable            = true
address           = "0.0.0.0:8545"
ws-address        = "0.0.0.0:8546"
api               = ["eth", "net", "web3", "txpool"]   # no "debug" on public nodes
gas-cap           = 25000000
evm-timeout       = "5s"
logs-cap          = 10000
block-range-cap   = 10000
```
Validator nodes and sentry nodes do **not** expose the EVM JSON-RPC endpoint — this is a full node / RPC fleet concern only. The JSON-RPC server runs on port 8545 and is exposed via Envoy at `/evm-rpc`. WebSocket runs on 8546 and is exposed at `/evm-ws`.

Storage note: EVM state (Ethermint `x/evm` StateDB) adds additional storage compared to a pure Cosmos chain. Archive nodes (no pruning) should budget an additional 300–500 GB for EVM state history at launch, scaling with EVM adoption. Non-archive nodes can prune EVM state along with Cosmos state using standard `pruning` settings in `app.toml`.

#### Sentry Node (× 2 minimum)
| Resource | Minimum | Recommended |
|---|---|---|
| CPU | 4 cores | 8 cores |
| RAM | 8 GB | 16 GB |
| Storage | 200 GB NVMe SSD | 500 GB NVMe SSD |
| Network | 1 Gbps | 1 Gbps (dedicated) |
| OS | Ubuntu 22.04 LTS | Ubuntu 22.04 LTS |

Sentry nodes should be in different availability zones or datacenters from each other.

#### Validator Node
| Resource | Minimum | Recommended |
|---|---|---|
| CPU | 4 cores | 8 cores |
| RAM | 16 GB | 32 GB |
| Storage | 200 GB NVMe SSD | 500 GB NVMe SSD |
| Network | 100 Mbps private | 1 Gbps private (no public internet) |
| OS | Ubuntu 22.04 LTS | Ubuntu 22.04 LTS |

#### Horcrux Signer Node (× 3, one shard each)
| Resource | Minimum | Recommended |
|---|---|---|
| CPU | 2 cores | 4 cores |
| RAM | 4 GB | 8 GB |
| Storage | 50 GB SSD | 100 GB SSD |
| Network | Private only — never internet-facing | Dedicated VLAN between signers and validator |
| OS | Ubuntu 22.04 LTS | Ubuntu 22.04 LTS |
| Location | Physically separate datacenter from other signers | Separate city / availability zone |

The three Horcrux signer machines must be in three physically separate locations. Loss of any one signer does not halt signing and does not expose the key.

---

### Software Requirements

Install on all node machines before beginning:

```bash
# Go (pin to version specified in chain repo go.mod)
wget https://go.dev/dl/go1.x.y.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.x.y.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin

# Chain binary (build from source for reproducibility)
git clone https://github.com/<org>/<chain-repo>
cd <chain-repo>
git checkout v<version>
make build
# Verify: sha256sum build/chaind must match published SHA256 in release notes

# Horcrux (on signer machines only)
wget https://github.com/strangelove-ventures/horcrux/releases/download/v<version>/horcrux_linux-amd64
chmod +x horcrux_linux-amd64
sudo mv horcrux_linux-amd64 /usr/local/bin/horcrux
```

---

### Joining Devnet (Internal Team Only)

Devnet is single-validator and internal. External validators do not join devnet.

```bash
# Initialize node
chaind init <moniker> --chain-id mychain-devnet-1

# Copy devnet genesis (distributed internally via internal repo)
cp devnet/genesis.json ~/.chain/config/genesis.json

# Configure persistent peers (internal devnet node)
# In config.toml: persistent_peers = "<devnet-node-id>@<internal-ip>:26656"

# Start node
chaind start
```

---

### Joining Testnet

#### Step 1 — Apply to Become a Testnet Validator

Submit the validator application form (provided by chain team):
- Operator name and organisation
- Expected node locations (datacenters / cloud regions)
- Experience running Cosmos validators (list of existing chains)
- Confirmation of hardware availability
- Public signing key (generated in Step 2 — **do not submit before generating**)
- Contact information: email + Telegram/Discord handle

Chain team reviews applications and confirms accepted validators via email.

#### Step 2 — Generate Validator Keys

```bash
# Initialize node with testnet chain-id
chaind init <moniker> --chain-id mychain-testnet-1 --home ~/.chain-testnet

# Generate consensus key (Ed25519)
# WARNING: do NOT use the key from this output for mainnet
# For testnet a file-based key is acceptable
# For mainnet you MUST use Horcrux (see Horcrux setup below)
chaind tendermint show-validator --home ~/.chain-testnet
# Output: {"@type":"/cosmos.crypto.ed25519.PubKey","key":"<base64>"}
# This is your validator public key — include in gentx
```

#### Step 3 — Download Testnet Genesis and Sync

```bash
# Download genesis (distributed by chain team via announcement channel)
curl -o ~/.chain-testnet/config/genesis.json \
  https://raw.githubusercontent.com/<org>/<chain-repo>/main/networks/testnet/genesis.json

# Verify genesis hash (chain team publishes expected hash)
sha256sum ~/.chain-testnet/config/genesis.json
# Must match: <published_hash>

# Configure peers (seed nodes provided by chain team)
# In config.toml:
#   seeds = "<seed-node-id>@<seed-ip>:26656"
#   persistent_peers = "<sentry-1-id>@<sentry-1-ip>:26656,<sentry-2-id>@..."

# Sync node (state sync from snapshot is fastest)
# In config.toml [statesync]:
#   enable = true
#   rpc_servers = "<rpc-1>:26657,<rpc-2>:26657"
#   trust_height = <recent_height>      # published by chain team
#   trust_hash = "<block_hash>"         # published by chain team

chaind start --home ~/.chain-testnet
# Wait for sync to complete (check: chaind status | jq .SyncInfo.catching_up)
```

#### Step 4 — Create Validator Transaction

```bash
# Fund your validator address first (use testnet faucet)
curl -X POST https://testnet.app.example.com/faucet \
  -d '{"address":"<your-cosmos-address>"}'

# Submit create-validator transaction
chaind tx staking create-validator \
  --amount 1000000utoken \
  --pubkey '{"@type":"/cosmos.crypto.ed25519.PubKey","key":"<base64>"}' \
  --moniker "<your-moniker>" \
  --commission-rate "0.05" \
  --commission-max-rate "0.20" \
  --commission-max-change-rate "0.01" \
  --min-self-delegation "1" \
  --from <your-key-name> \
  --chain-id mychain-testnet-1 \
  --gas auto --gas-adjustment 1.4 \
  --fees 5000utoken \
  --home ~/.chain-testnet

# Verify validator is in active set
chaind query staking validators --home ~/.chain-testnet | grep <your-moniker>
```

#### Step 5 — Configure Sentry Architecture

```toml
# validator/config/config.toml
[p2p]
pex = false                                    # do NOT advertise to public
persistent_peers = "<sentry-1-id>@<sentry-1-private-ip>:26656,<sentry-2-id>@<sentry-2-private-ip>:26656"
private_peer_ids = ""
addr_book_strict = false

# sentry/config/config.toml
[p2p]
pex = true
persistent_peers = "<validator-id>@<validator-private-ip>:26656"
private_peer_ids = "<validator-node-id>"       # never gossip validator address to peers
unconditional_peer_ids = "<validator-node-id>"
```

#### Step 6 — Notify Chain Team

Post in the validator coordination channel:
- Validator address
- Node ID of your sentry nodes (so chain team can add to seed list if needed)
- Monitoring dashboard URL (if public)
- On-call contact for upgrade coordination

---

### Mainnet Genesis Ceremony

Mainnet requires a more rigorous process than testnet. Every step below is mandatory.

#### Step 1 — Confirm Participation (4 weeks before genesis)

Reply to chain team invitation with:
- Legal entity name and jurisdiction
- Hardware locations confirmed (at least sentry + validator + 3 Horcrux signers)
- Confirmation that Horcrux will be used (file-based key is NOT accepted on mainnet)
- On-call contact for genesis day (must be reachable within 15 minutes)
- Signed validator agreement (provided by chain team)

#### Step 2 — Horcrux Key Ceremony (3 weeks before genesis)

**This is the most critical step. Do not skip or rush it.**

```bash
# On a dedicated offline machine (air-gapped for key generation)
# All three signer operators should be present or connected via verified video call

# Step 2a: Generate threshold key shares
horcrux create-ecies-shards \
  --shards 3 \
  --threshold 2 \
  --output ./key-shards/

# This produces:
#   shard_1_of_3.json  → transferred to Signer Machine 1
#   shard_2_of_3.json  → transferred to Signer Machine 2
#   shard_3_of_3.json  → transferred to Signer Machine 3
# and:
#   validator_public_key.json → your consensus public key for gentx

# Step 2b: Transfer shards to signer machines
# Use encrypted transfer (GPG or age encryption) — never plain text over network
# Verify shard received correctly on each signer machine before proceeding

# Step 2c: Encrypt and back up each shard to air-gapped USB
# On each signer machine:
age -p shard_X_of_3.json > shard_X_of_3.json.age
# Store encrypted USB in physically separate secure location from signer machine
# Verify decryption works before storing: age -d shard_X.json.age

# Step 2d: Configure Horcrux on each signer machine
# signer_1/config.yaml:
cosigner:
  threshold: 2
  shares: 3
  shardID: 1          # unique per signer (1, 2, or 3)
  keyFile: /etc/horcrux/shard_1_of_3.json
  p2pListenAddress: "tcp://0.0.0.0:2222"
  nodes:
    - address: "tcp://<validator-private-ip>:1234"
      publicKey: <validator-node-public-key>
  cosigners:
    - shardID: 2
      address: "tcp://<signer-2-private-ip>:2222"
    - shardID: 3
      address: "tcp://<signer-3-private-ip>:2222"

# Step 2e: Configure validator node to use Horcrux (not file-based key)
# validator/config/config.toml:
[priv_validator]
  key_file = ""                         # empty — no file key
  state_file = ""                       # empty — Horcrux manages state
  laddr = "tcp://0.0.0.0:1234"          # Horcrux connects here

# Step 2f: Start Horcrux on all three signer machines
horcrux start --config /etc/horcrux/config.yaml

# Step 2g: Test signing (without submitting to chain)
horcrux sign-test --config /etc/horcrux/config.yaml
# Must show: "2-of-3 threshold signing successful"
```

Document the key ceremony: date, participants, shard locations, USB encryption password location (hardware password manager, not written down).

#### Step 3 — Submit gentx (2 weeks before genesis)

```bash
# Initialize node with mainnet chain-id
chaind init <moniker> --chain-id mychain-1 --home ~/.chain-mainnet

# Copy pre-genesis file distributed by chain team
cp mainnet/pre-genesis.json ~/.chain-mainnet/config/genesis.json

# Generate gentx using Horcrux public key (from Step 2a validator_public_key.json)
chaind gentx <key-name> 10000000utoken \
  --pubkey '{"@type":"/cosmos.crypto.ed25519.PubKey","key":"<horcrux-pubkey-base64>"}' \
  --moniker "<your-moniker>" \
  --commission-rate "0.05" \
  --commission-max-rate "0.20" \
  --commission-max-change-rate "0.01" \
  --min-self-delegation "1" \
  --chain-id mychain-1 \
  --home ~/.chain-mainnet

# Submit gentx file to chain team via PR to genesis repository
# Deadline: 10 days before genesis
cp ~/.chain-mainnet/config/gentx/gentx-<hash>.json \
   <genesis-repo>/networks/mainnet/gentxs/<your-moniker>.json
```

#### Step 4 — Verify Final Genesis File (1 week before genesis)

Chain team collects all gentxs, produces final `genesis.json`, and distributes it.

```bash
# Download final genesis
curl -o ~/.chain-mainnet/config/genesis.json \
  https://raw.githubusercontent.com/<org>/<chain-repo>/main/networks/mainnet/genesis.json

# Verify genesis hash (chain team publishes expected hash)
sha256sum ~/.chain-mainnet/config/genesis.json
# Must match: <published_hash>

# Run mandatory genesis invariant verifications:
chaind validate-genesis --home ~/.chain-mainnet

# Manual checks (every validator must run these independently):
# 1. Sum of all genesis balances = S - C
# 2. bridge_escrow module account = C
# 3. Your validator entry is present with correct pubkey
# 4. commission_rate, commission_max_rate match what you submitted
# 5. voting_period = 7 days (NOT devnet/testnet values)
# 6. bridge confirmation depths = 15 / 50

# Report any discrepancy to chain team BEFORE genesis day
```

#### Step 5 — Genesis Day Coordination

```
T - 2h   All validators confirm readiness in coordination channel
T - 1h   Start chain binary (the node will automatically wait until genesis time is reached before producing blocks — no special flag required; CometBFT holds at genesis until the genesis.initial_height block time passes)
T - 0    Genesis time reached; all validators start simultaneously
          chaind start --home ~/.chain-mainnet
T + 5m   First block produced (expect ≥ 2/3 validators online)
T + 10m  Confirm in coordination channel: block production confirmed, no errors
T + 30m  Chain team announces genesis successful; monitoring active
```

If your node is not producing blocks at T+10m:
1. Check logs: `journalctl -u chaind -f`
2. Check Horcrux logs on all 3 signer machines
3. Check sentry peer connections: `chaind status | jq .NodeInfo.Peers`
4. Post in coordination channel with error — do not silently fail

---

### Security Requirements

These are not suggestions. Validators who do not meet these requirements will not be admitted to the mainnet active set.

| Requirement | Standard | Verification |
|---|---|---|
| Horcrux threshold signing | 2-of-3 mandatory on mainnet | Chain team verifies by querying consensus key — must match Horcrux pubkey |
| Sentry nodes | ≥ 2 sentries, validator not directly peered with public network | Network policy verified during testnet |
| Firewall | Validator node: only sentry IPs allowed on port 26656; Horcrux port only from signer IPs | Self-reported + spot check |
| SSH access | Key-based only; password auth disabled; root login disabled; SSH port non-standard | Self-reported |
| Horcrux signer isolation | No other services on signer machines; inbound only from validator node IP | Self-reported |
| Key shard backup | Encrypted USB; physically separate from signer machine | Self-reported; demonstrated in key ceremony |
| Monitoring | Node must have uptime monitoring with < 5 min alert latency | Dashboard URL provided to chain team |
| On-call SLA | Validator operator reachable within 15 minutes during business hours; 30 minutes off-hours | Provided in onboarding form |
| Upgrade SLA | Binary staged within 12 hours of upgrade announcement; validator online at upgrade height | Enforced — repeated failures result in governance removal proposal |

---

### Monitoring Setup for Validators

Minimum monitoring every validator must run independently:

```yaml
# Prometheus scrape targets (add to prometheus.yml)
- job_name: 'cosmos-validator'
  static_configs:
    - targets: ['localhost:26660']   # CometBFT metrics port

# Critical alerts to configure:
alerts:
  - name: ValidatorMissedBlocks
    condition: cometbft_consensus_validator_missed_blocks > 50
    severity: P1
    message: "Validator missing blocks — check signing and Horcrux"

  - name: HorcruxSignerOffline
    condition: horcrux_signer_connected < 2
    severity: P1
    message: "Horcrux signer machine offline — 2-of-3 threshold at risk"

  - name: NodeNotSynced
    condition: cometbft_consensus_latest_block_time < now() - 30s
    severity: P1
    message: "Node not syncing — check peer connections"

  - name: PeerCountLow
    condition: cometbft_p2p_peers < 3
    severity: P2
    message: "Low peer count — check sentry node connectivity"

  - name: DiskUsageHigh
    condition: disk_used_percent > 80
    severity: P2
    message: "Disk usage high — prune or expand storage"
```

Share your monitoring dashboard URL with the chain team. This is used during upgrades to verify your node is healthy.

---

### Ongoing Validator Responsibilities

| Responsibility | Frequency | Details |
|---|---|---|
| Binary upgrades | Per upgrade (announced ≥ 2 weeks ahead) | Stage binary before upgrade height; be online at upgrade time |
| Governance participation | Per proposal | Vote on all proposals; non-participation tracked |
| Security patches | As announced | Critical patches applied within 24 hours of announcement |
| Horcrux shard accessibility drill | Annual | Verify each shard can be restored from USB backup |
| Monitoring alert review | Monthly | Confirm all alerts firing correctly |
| Hardware refresh | As needed | Notify chain team 2 weeks before any hardware migration |
| Key rotation | Annual (or on security event) | Follow key rotation runbook in Phase 8.4 |
| On-call contact update | On change | Update contact in validator coordination channel immediately |

---

### Validator Admission Checklist

This checklist is signed off by the validator operator and reviewed by the chain team before the validator is admitted to the active set on testnet or mainnet.

**Identity and Agreement**
- [ ] Validator agreement signed and submitted
- [ ] Legal entity name, jurisdiction, and contact confirmed
- [ ] On-call contact provided (15-minute SLA acknowledged)

**Hardware**
- [ ] Full node / RPC node provisioned to spec
- [ ] ≥ 2 sentry nodes provisioned in separate availability zones
- [ ] Validator node provisioned with no public internet exposure
- [ ] 3 Horcrux signer machines provisioned in 3 physically separate locations

**Key Management**
- [ ] Horcrux key ceremony completed (all 3 shard operators present)
- [ ] Horcrux 2-of-3 threshold signing tested successfully
- [ ] Each key shard backed up to encrypted air-gapped USB
- [ ] USB decryption verified before storing
- [ ] USB stored in physically separate location from signer machine
- [ ] Key ceremony documented (date, participants, shard locations)
- [ ] **No file-based consensus key present on validator node** (mainnet only)

**Network Architecture**
- [ ] Validator node not directly peered with public network
- [ ] Sentry nodes configured with `private_peer_ids = <validator-node-id>`
- [ ] Validator node peers only with sentry node IPs
- [ ] Horcrux signers reachable only from validator node IP on signing port
- [ ] Firewall rules verified and documented

**Node Setup**
- [ ] Genesis file downloaded and SHA256 verified against published hash
- [ ] Genesis invariant checks passed (`chaind validate-genesis`)
- [ ] Manual supply math verified independently
- [ ] Node fully synced (catching_up = false)
- [ ] gentx submitted before deadline (mainnet only)

**Security**
- [ ] SSH: key-based only; password auth disabled; root login disabled
- [ ] All machines patched (OS updates applied)
- [ ] No unnecessary services running on validator or signer machines
- [ ] Monitoring live with < 5 min alert latency

**Operational Readiness**
- [ ] Monitoring dashboard URL shared with chain team
- [ ] Upgrade SLA acknowledged (binary staged within 12 hours of announcement)
- [ ] Coordination channel joined (Discord / Telegram)
- [ ] On-call escalation path documented
- [ ] Governance participation committed

**Signature**

> I confirm that all items above are completed and accurate. I understand that failure to maintain these standards — including missed upgrade deadlines or extended downtime — may result in a governance proposal to remove this validator from the active set.
>
> Operator: _________________________ Date: _____________

---

## Oracle Operator Onboarding Guide

This guide is handed to every operator admitted to the permissioned oracle set. It covers the oracle's role in the chain, HSM setup, feed source configuration with primary and fallback, commit-reveal client setup, monitoring, and the admission checklist operators sign before going live.

---

### Oracle's Role in the Chain

The oracle system provides off-chain price data to on-chain modules (`x/oracle`, `x/settlement`, `x/bridge`) using a commit-reveal scheme that prevents front-running.

```
[External Price Sources]
   ↓  (HTTPS/WebSocket)
[Oracle Aggregator — your off-chain binary]
   ↓  MsgCommitOracleHash (block N)
[x/oracle module — chain]
   ↓  reveal window (block N+1 to N+2)
[Oracle Aggregator — your off-chain binary]
   ↓  MsgRevealOracleReport (block N+1)
[x/oracle module — aggregates all reveals → on-chain median price]
   ↓
[x/settlement, x/bridge, x/certification — consume price]
```

Every admitted oracle operator runs this commit-reveal cycle every block. Missing more than the configured `MissThreshold` in a sliding window triggers an automatic removal from the permissioned set via `x/oracle` slashing logic.

The oracle set is governed by chain governance. Adding or removing an operator requires a governance proposal that passes with a standard quorum.

---

### Who Can Apply

Oracle operators must be able to demonstrate:
- Experience operating production services with ≥ 99.9% uptime SLA
- Access to at least two independent price feed sources (different providers)
- Hardware Security Module (HSM) availability for signing oracle transactions
- On-call coverage (15-minute response during business hours; 30 minutes off-hours)

---

### Architecture: What You Run

```
[Feed Source 1 — Primary]    [Feed Source 2 — Fallback]
           ↓                          ↓
    [Oracle Aggregator binary]  ←  failover logic
           ↓
    [HSM — signs MsgCommitOracleHash / MsgRevealOracleReport]
           ↓
    [Chain gRPC endpoint]
           ↓
    [x/oracle module]
```

Minimum two feed sources are required. The aggregator selects the primary source; if it fails health checks (no response within 3 seconds or price deviation > 5% from last known value), it falls over to the secondary automatically. Oracle transactions are signed exclusively through the HSM — the private key never touches the aggregator process directly.

---

### Machine Requirements

#### Oracle Aggregator Machine

| Resource | Minimum | Recommended |
|---|---|---|
| CPU | 2 cores | 4 cores |
| RAM | 4 GB | 8 GB |
| Storage | 50 GB SSD | 100 GB SSD |
| Network | 100 Mbps | 1 Gbps |
| OS | Ubuntu 22.04 LTS | Ubuntu 22.04 LTS |

#### HSM

| Option | Accepted | Notes |
|---|---|---|
| YubiHSM 2 | Yes (testnet + mainnet) | USB-attached to aggregator machine; inexpensive; sufficient for oracle tx volume |
| AWS CloudHSM | Yes | Higher cost; suitable if already in AWS |
| Azure Dedicated HSM | Yes | Thales Luna based |
| SoftHSM 2 | **Testnet only** | Software HSM; acceptable for testnet; NOT accepted for mainnet |
| File-based private key | **Not accepted** | Never accepted on mainnet |

The private key generated in the HSM must never be exportable in plaintext. YubiHSM 2 enforces this by hardware policy. Cloud HSMs enforce it by IAM policy — document your IAM policy and submit it with your application.

---

### HSM Setup (YubiHSM 2 — Recommended Path)

```bash
# Step 1: Install YubiHSM SDK
wget https://developers.yubico.com/YubiHSM2/Releases/yubihsm2-sdk-<version>-ubuntu2204-amd64.tar.gz
tar -xzf yubihsm2-sdk-*.tar.gz
sudo dpkg -i yubihsm2-sdk/*.deb

# Step 2: Connect YubiHSM 2 via USB and start the connector daemon
sudo systemctl enable yubihsm-connector
sudo systemctl start yubihsm-connector
# Verify: curl http://localhost:12345/connector/status
# Should return: status=OK

# Step 3: Create a dedicated application key for oracle signing
# Use the default admin credentials (change immediately after)
yubihsm-shell
  > connect
  > session open 1 password     # admin session
  > put authkey 0 1 "oracle-operator" all sign-ecdsa:sign-eddsa exportable-under-wrap
  > generate asymmetric-key 0 1 "oracle-signing-key" all sign-ecdsa secp256k1
  # Note the key ID returned — you need this for aggregator config
  > session close 1
  > quit

# Step 4: Change admin credentials immediately
yubihsm-shell
  > connect
  > session open 1 password
  > put authkey 0 1 "admin" all all <new-strong-password>
  > delete object 1 authentication-key  # delete default key
  # WARNING: store new admin password in hardware password manager — loss = key unrecoverable

# Step 5: Extract the public key (safe — public key only)
yubihsm-shell
  > connect
  > session open 2 oracle-operator <operator-password>
  > get pubkey 1 <key-id>
  # Output: compressed secp256k1 pubkey (33 bytes hex)
  # This is your oracle operator address — derive Cosmos address from it
  > session close 2

# Step 6: Derive your Cosmos oracle operator address
# Use chain binary keytool to import the public key and show the bech32 address
chaind keys add oracle-hsm --pubkey <pubkey-hex> --keyring-backend memory
chaind keys show oracle-hsm --address
# Share this address with chain team for governance proposal
```

**SoftHSM 2 (testnet only):**

```bash
sudo apt-get install softhsm2
sudo softhsm2-util --init-token --slot 0 --label "oracle-testnet" --pin <pin> --so-pin <so-pin>
# Use pkcs11-tool to generate secp256k1 key inside SoftHSM
pkcs11-tool --module /usr/lib/softhsm/libsofthsm2.so \
  --login --pin <pin> \
  --keypairgen --key-type EC:secp256k1 \
  --label oracle-signing-key --id 01
```

---

### Feed Source Configuration

The oracle aggregator config file (`/etc/oracle/config.yaml`) controls feed sources, failover logic, and chain connection.

```yaml
# /etc/oracle/config.yaml

# Chain connection
chain:
  grpc_endpoint: "https://grpc.testnet.example.com:9090"   # use your own full node for mainnet
  chain_id: "mychain-1"
  gas_adjustment: 1.4
  gas_prices: "0.025utoken"
  broadcast_mode: "sync"

# HSM signing (YubiHSM 2)
signer:
  type: yubihsm
  connector_url: "http://localhost:12345"
  auth_key_id: 1
  auth_password_env: "YUBIHSM_AUTH_PASSWORD"   # set via environment secret; never hardcoded
  signing_key_id: <key-id-from-setup>           # numeric key ID from HSM setup Step 3

# For SoftHSM 2 (testnet only):
# signer:
#   type: pkcs11
#   module: "/usr/lib/softhsm/libsofthsm2.so"
#   pin_env: "SOFTHSM_PIN"
#   key_label: "oracle-signing-key"

# Price feed sources — minimum 2 required
feeds:
  - id: primary
    provider: binance                    # or coinbase, kraken, coingecko, etc.
    base_url: "https://api.binance.com"
    symbols:
      BNB_USDT: "/api/v3/ticker/price?symbol=BNBUSDT"
      ETH_USDT: "/api/v3/ticker/price?symbol=ETHUSDT"
    timeout_ms: 3000
    health_check_interval_s: 5

  - id: fallback
    provider: coinbase
    base_url: "https://api.coinbase.com"
    symbols:
      BNB_USDT: "/v2/prices/BNB-USDT/spot"
      ETH_USDT: "/v2/prices/ETH-USDT/spot"
    timeout_ms: 3000
    health_check_interval_s: 5

# Failover policy
failover:
  primary: primary
  fallback: fallback
  switch_on_timeout: true
  switch_on_deviation_pct: 5.0         # switch if price deviates > 5% between sources
  recovery_checks_required: 3          # primary must pass 3 consecutive health checks before switching back

# Commit-reveal parameters (must match x/oracle module params)
oracle:
  commit_offset_blocks: 0             # commit in the same block
  reveal_offset_blocks: 1             # reveal 1 block after commit
  salt_length_bytes: 32               # random salt generated per commit
  # Commitment formula (must match x/oracle module MsgCommitOracleHash):
  #   sha256(operator_address || feed_id || round_id || price_string || salt)
  # The module includes operator_address, feed_id, and round_id in the pre-image to prevent
  # cross-operator and cross-round replay of commitment hashes.
  # The simplified notation SHA256(price||salt) in inline comments elsewhere is shorthand only —
  # the on-chain verifier uses the full 5-field pre-image above.

# Monitoring
metrics:
  prometheus_port: 9200
  expose_feed_latency: true
  expose_price_deviation: true
```

---

### Commit-Reveal Client: How It Works

The aggregator runs two goroutines per block:

**Goroutine 1 — Committer (fires at block N)**
```
1. Query current price from active feed source (primary or fallback)
2. Generate 32-byte cryptographically random salt
3. Compute commitment: sha256(operator_address || feed_id || round_id || price_string || salt)
   — matches the full pre-image required by MsgCommitOracleHash on-chain verification
4. Sign MsgCommitOracleHash{commitment, denom_list} via HSM
5. Broadcast transaction
6. Store (block_height, price, salt) in local SQLite for reveal
```

**Goroutine 2 — Revealer (fires at block N+1)**
```
1. Look up (price, salt) stored at block N from local SQLite (the block where the commit was broadcast)
2. Sign MsgRevealOracleReport{price, salt, denom_list} via HSM
3. Broadcast transaction
4. Delete local record (no longer needed)
```

The salt is the protection against front-running: without the salt, nobody can reverse the SHA256 commitment to learn the price before reveal. The salt is stored only in local SQLite — if the aggregator crashes between commit and reveal, the pending reveal is lost and counts as a miss. This is why crash recovery must complete within one block time (≈ 6 seconds).

---

### Crash Recovery and High Availability

A single aggregator machine creates a single point of failure. For mainnet, run a warm standby:

```
[Active Aggregator]  ←→  [Standby Aggregator]
       ↓                         ↓
     [HSM]                  [Same HSM via USB passthrough or network HSM]
       ↓                         ↓
         [Shared SQLite replica (litestream to S3)]
```

Failover approach:
1. Active aggregator writes commit/reveal state to SQLite, streamed to S3 via **litestream** in real time (sub-second replication)
2. Standby monitors active via health endpoint (`/healthz`)
3. If active does not respond within 2 consecutive block times (≈ 12 seconds), standby promotes itself:
   - Downloads latest SQLite snapshot from S3
   - Resumes commit-reveal cycle
4. Both machines must share the same HSM key. Options:
   - Network HSM (AWS CloudHSM / Azure Dedicated HSM) — both machines connect over private network
   - YubiHSM 2 with USB passthrough — only works if both machines are in same rack; not ideal for geographic separation
   - Preferred for mainnet: Network HSM in two-machine active/standby setup

The two aggregator machines must never both try to commit simultaneously — that produces conflicting commits in the same block and counts as a miss. The health-check window (2 block times) ensures only one machine is active at any given time.

---

### Joining the Oracle Set

#### Step 1 — Apply

Submit the oracle operator application (provided by chain team):
- Organisation name and jurisdiction
- Hardware and HSM model planned
- Feed sources planned (primary + fallback providers)
- Uptime SLA you can commit to
- On-call contact (15-minute response SLA)
- Oracle operator Cosmos address (derived from HSM public key)
- Testnet participation history (required for mainnet application)

#### Step 2 — Testnet Trial Period

All oracle operator candidates must run on testnet for a minimum of **4 weeks** before mainnet admission. During the trial period, chain team monitors:
- Miss rate (must be < 1% over 4 weeks)
- Price deviation from other oracle operators (outliers flagged)
- Response time to simulated feed source failures
- Feed source failover behaviour during testnet chaos drills

Performance data from testnet is submitted with the mainnet application.

#### Step 3 — Governance Proposal (Mainnet)

Adding an oracle operator to the permissioned set requires a governance proposal:

```bash
# Chain team submits on behalf of accepted operator:
chaind tx gov submit-proposal add-oracle-operator \
  --operator-address <cosmos-address-from-hsm> \
  --moniker "<operator-name>" \
  --feed-sources "binance,coinbase" \
  --deposit 10000000utoken \
  --title "Add Oracle Operator: <operator-name>" \
  --description "Testnet trial: 4 weeks, miss rate <0.3%, feed failover verified" \
  --from <chain-team-key> \
  --chain-id mychain-1
```

Voting period: 7 days. Quorum: 33.4%. If the proposal passes, the operator address is added to the `x/oracle` permissioned set in state, and the operator can begin submitting commits immediately.

#### Step 4 — Go-Live

```bash
# Fund the oracle operator address (gas for commit/reveal transactions)
# Budget: each commit + reveal pair costs ~2 transactions per block
# At 6s block time: ~14,400 transaction pairs per day
# Gas cost per pair: ~5,000 gas × 2 = 10,000 gas × 0.025utoken/gas = 250utoken/block
# Daily budget: ~3,600,000 utoken (3.6 token/day) — fund address accordingly

# Start oracle aggregator
sudo systemctl enable oracle-aggregator
sudo systemctl start oracle-aggregator

# Verify: check first commit appears on-chain
chaind query oracle pending-commits --oracle-address <your-address> --chain-id mychain-1

# Verify: check first reveal appears on-chain (next block)
chaind query oracle reveal-history --oracle-address <your-address> --limit 5 --chain-id mychain-1

# Notify chain team: oracle is live
```

---

### Security Requirements

| Requirement | Standard | Verification |
|---|---|---|
| HSM signing | YubiHSM 2 or cloud HSM mandatory on mainnet | Chain team verifies signing key is HSM-bound via key attestation certificate |
| Private key exportability | Must be non-exportable; enforced by HSM policy | Submit HSM key attestation certificate with application |
| Feed source diversity | ≥ 2 independent providers (different companies) | Self-reported; validated during testnet trial |
| Failover tested | Must demonstrate automated failover in testnet | Chain team observes during testnet chaos drill |
| Aggregator machine access | Key-based SSH only; password auth disabled | Self-reported |
| Auth password for HSM | Stored in hardware password manager; not hardcoded | Confirmed in checklist |
| Oracle address balance | Must maintain ≥ 30-day gas reserve at current usage | Monitoring alert required |
| No shared oracle key | Oracle signing key must be unique to oracle role; not shared with validator or other services | Self-reported |

---

### Monitoring

```yaml
# Add to prometheus.yml
- job_name: 'oracle-aggregator'
  static_configs:
    - targets: ['localhost:9200']

# Critical alerts for oracle operators:

- name: OracleMissedCommit
  condition: oracle_missed_commits_per_hour > 2
  severity: P1
  message: "Oracle missing commits — check aggregator and HSM connectivity"

- name: OracleFeedSourceFailed
  condition: oracle_active_feed_source == "fallback" for > 5m
  severity: P1
  message: "Oracle running on fallback feed — primary feed source down"

- name: OracleFeedSourcesBothDown
  condition: oracle_healthy_feed_sources == 0
  severity: P0    # page immediately — will start missing commits
  message: "All oracle feed sources down — price submission halted"

- name: OracleHSMUnreachable
  condition: oracle_hsm_connected == 0
  severity: P0
  message: "HSM unreachable — oracle cannot sign transactions"

- name: OracleAddressBalanceLow
  condition: oracle_address_balance_utoken < 10000000   # < 10 token
  severity: P2
  message: "Oracle operator address balance low — top up within 48 hours"

- name: OraclePriceDeviationHigh
  condition: oracle_price_deviation_from_median_pct > 3
  severity: P2
  message: "Oracle price deviating from median — check feed source data quality"
```

Share the monitoring dashboard URL with chain team. It is reviewed monthly and used as evidence of operational quality for annual re-admission.

---

### Annual Re-Admission

Oracle operator status in the permissioned set is reviewed annually by governance. The chain team publishes a report 30 days before the renewal window showing:
- Miss rate over the year (threshold: < 1% to remain without review; > 2% triggers removal proposal)
- Feed source failover events and recovery times
- Security incidents or suspected key compromise events
- Governance participation rate (oracle operators are expected to vote)

Operators below threshold are automatically renewed. Operators above threshold must present a remediation plan to governance before renewal.

---

### Oracle Operator Admission Checklist

This checklist is signed by the oracle operator and reviewed by the chain team before the governance proposal is submitted.

**Identity and Agreement**
- [ ] Oracle operator agreement signed and submitted
- [ ] Legal entity name, jurisdiction, and contact confirmed
- [ ] On-call contact provided (15-minute response SLA acknowledged)
- [ ] Annual re-admission process acknowledged

**Hardware and HSM**
- [ ] Aggregator machine provisioned to spec
- [ ] HSM model confirmed and procured (YubiHSM 2 or cloud HSM)
- [ ] HSM setup completed; oracle signing key generated inside HSM
- [ ] Signing key confirmed non-exportable (attestation certificate available)
- [ ] HSM admin credentials stored in hardware password manager
- [ ] Default HSM admin credentials changed
- [ ] **File-based private key NOT present on aggregator machine** (mainnet)

**Feed Sources**
- [ ] ≥ 2 independent feed sources configured (different providers)
- [ ] Primary feed source configured and health checks passing
- [ ] Fallback feed source configured and health checks passing
- [ ] Automated failover tested (primary killed; fallback activated within 3 health-check cycles)
- [ ] Failover recovery tested (primary restored; traffic returns after 3 consecutive passing checks)
- [ ] Price deviation between sources validated (< 0.5% delta under normal conditions)

**Commit-Reveal Client**
- [ ] Oracle aggregator binary built from verified source (SHA256 matches release)
- [ ] Config file reviewed and committed to internal ops repo
- [ ] Auth password provided via environment secret (not hardcoded in config)
- [ ] Commit-reveal cycle tested end-to-end on testnet (commits and reveals visible on-chain)
- [ ] Local SQLite commit state store confirmed working
- [ ] Crash recovery tested: process killed mid-cycle; verify miss count and recovery

**High Availability (Mainnet)**
- [ ] Warm standby aggregator machine provisioned
- [ ] litestream SQLite replication to S3 (or equivalent) configured and tested
- [ ] Standby promotion tested: active killed; standby takes over within 2 block times
- [ ] Both machines confirmed using same HSM key (network HSM or equivalent)
- [ ] Double-commit guard confirmed: two machines cannot both commit simultaneously

**Testnet Trial (Required for Mainnet)**
- [ ] ≥ 4 weeks of testnet operation completed
- [ ] Miss rate over trial period: ___% (must be < 1%)
- [ ] Feed failover drill completed and observed by chain team
- [ ] Testnet performance report submitted with application

**Monitoring**
- [ ] Prometheus metrics endpoint live (`/metrics` on port 9200)
- [ ] All 6 critical alerts configured with correct thresholds
- [ ] P0 alerts paging on-call within 5 minutes
- [ ] Monitoring dashboard URL shared with chain team
- [ ] Oracle address balance alert verified firing at correct threshold

**Operational Readiness**
- [ ] Oracle operator address funded (≥ 30-day gas reserve)
- [ ] Upgrade SLA acknowledged (oracle binary updated within 12 hours of announcement)
- [ ] Validator coordination channel joined
- [ ] Governance participation committed

**Signature**

> I confirm that all items above are completed and accurate. I understand that a miss rate above 2% over any rolling 30-day period, failure to maintain HSM-only signing, or failure to maintain ≥ 2 independent feed sources may result in a governance proposal to remove this operator from the permissioned oracle set.
>
> Operator: _________________________ Date: _____________

---

## Bridge Relayer Onboarding Guide

This guide is handed to every operator joining the BNB Smart Chain ↔ chain bridge relayer set. It covers the relayer's role, EOA key management, nonce tracking, the promotion ladder from candidate to primary, monitoring, and the admission checklist operators sign before going live.

---

### The Relayer's Role in the Bridge

The bridge moves assets between BNB Smart Chain (BSC) and your Cosmos chain in both directions. Relayers are the off-chain actors that watch events on one side and submit the corresponding transaction on the other side.

```
BNB Smart Chain                         Your Chain
──────────────                          ──────────
User calls LockBox.lock(amount)
   ↓
LockBox emits TokensLocked event
   ↓
[Relayer watches BSC via RPC]
   ↓
Relayer calls MsgBridgeIn on chain
   ↓
                                        x/bridge mints tokens to user

─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─ ─

Your Chain                              BNB Smart Chain
──────────                              ──────────
User calls MsgBridgeOut on chain
   ↓
x/bridge burns tokens, emits event
   ↓
[Relayer watches chain via gRPC]
   ↓
Relayer calls LockBox.release(amount)
   ↓
                                        User receives BEP-20 tokens
```

Relayers do not hold user funds. They hold only enough gas to submit transactions on both sides. The LockBox contract and `x/bridge` module hold the actual locked assets.

Every submitted bridge transaction is identified by a **keccak256 nonce** generated by the LockBox contract. This nonce is stored on-chain in `x/bridge` state. If a relayer submits a duplicate nonce it is rejected — meaning multiple relayers can watch the same events and race to submit without any double-processing risk.

---

### Relayer Roles and the Promotion Ladder

The relayer set has three tiers. Operators enter as candidates and are promoted through the ladder by governance.

```
Tier 1 — Primary Relayer (1 active at a time)
   Responsibility: submit all bridge transactions
   Failover: if Primary misses ≥ 3 events in a 10-minute window,
             Secondary is automatically promoted by x/bridge module

Tier 2 — Secondary Relayer (1–2 operators)
   Responsibility: watch the same events as Primary; submit if Primary misses
   Also submits during planned Primary downtime (upgrade windows)

Tier 3 — Candidate Relayer (any number)
   Responsibility: watch events; submit ONLY if both Primary and Secondary miss
   Not eligible to earn relayer rewards (governance-set parameter)
   Minimum 2-week observation period before promotion to Secondary
```

Promotion from Candidate → Secondary → Primary requires a governance proposal at each step. Chain team publishes miss-rate data used in proposals.

---

### What You Run

```
[BSC RPC node — primary]    [BSC RPC node — fallback]
           ↓                          ↓
[Your Chain gRPC node — primary]    [Your Chain gRPC node — fallback]
                 ↓
        [Relayer binary]
           ↓        ↓
    [EOA hot wallet]   [Nonce tracker DB — SQLite]
           ↓
  [LockBox contract on BSC]
  [x/bridge module on chain]
```

The relayer binary watches both chains simultaneously, maintains its own nonce tracker, and submits transactions using the EOA hot wallet. The EOA signs BSC transactions (Ethereum-style); the operator key signs chain transactions (Cosmos-style).

---

### Machine Requirements

| Resource | Minimum | Recommended |
|---|---|---|
| CPU | 2 cores | 4 cores |
| RAM | 4 GB | 8 GB |
| Storage | 100 GB SSD | 200 GB SSD |
| Network | 100 Mbps | 1 Gbps |
| OS | Ubuntu 22.04 LTS | Ubuntu 22.04 LTS |

The relayer machine must have reliable, low-latency connectivity to both BSC RPC and chain gRPC endpoints. Latency above 500 ms to either endpoint increases the risk of being beaten by another relayer on submission, which counts as a miss for the slower relayer.

---

### EOA Key Management

The relayer uses two keys:

| Key | Chain | Purpose | Storage |
|---|---|---|---|
| EOA hot wallet | BSC (secp256k1 Ethereum) | Sign `LockBox.release()` BSC transactions | Encrypted keystore file |
| Operator key | Your chain (secp256k1 Cosmos) | Sign `MsgBridgeIn` / `MsgBridgeOut` chain transactions | Encrypted keystore file |

**Mainnet key requirements:**
- Both keys stored in encrypted keystore files (not plaintext private keys)
- Keystore password provided via environment secret — never hardcoded in config
- Keystore files backed up to encrypted offline storage (separate from machine)
- EOA address funded with BNB for gas (minimum 30-day reserve)
- Operator address funded with chain token for gas (minimum 30-day reserve)

**For mainnet Primary relayer only:** chain team strongly recommends using AWS KMS or HashiCorp Vault Transit for the EOA key, so the key is never in memory on the relayer machine. Integrate via `go-ethereum` KMS signer. Secondary and Candidate relayers may use encrypted keystore files.

**Generating keys:**

```bash
# BSC EOA key (Ethereum-style)
# Use go-ethereum clef or cast (Foundry) for keystore generation
cast wallet new --keystore-dir ./keystore
# Note: keystore file is created at ./keystore/<address>.json
# Record the address — fund it with BNB before going live

# Chain operator key (Cosmos-style)
chaind keys add relayer-operator \
  --keyring-backend file \
  --keyring-dir ./keyring
chaind keys show relayer-operator --address --keyring-backend file --keyring-dir ./keyring
# Record the address — fund it before going live

# Derive your relayer operator Cosmos address and submit to chain team for governance proposal
```

---

### Nonce Tracking

The relayer maintains a local SQLite database tracking which LockBox nonces have been seen, submitted, and confirmed. This prevents duplicate submissions when the relayer restarts mid-inflight.

```sql
-- Relayer nonce tracker schema (created automatically by relayer binary on first start)
CREATE TABLE bridge_events (
  nonce          TEXT PRIMARY KEY,   -- keccak256 nonce from LockBox event
  direction      TEXT NOT NULL,      -- 'bsc_to_chain' or 'chain_to_bsc'
  amount         TEXT NOT NULL,      -- token amount (string to avoid float precision)
  recipient      TEXT NOT NULL,      -- destination address
  bsc_tx_hash    TEXT,               -- BSC tx where event was emitted (bsc_to_chain)
  chain_tx_hash  TEXT,               -- chain tx where event was emitted (chain_to_bsc)
  status         TEXT NOT NULL,      -- 'seen' | 'submitted' | 'confirmed' | 'failed'
  submit_tx_hash TEXT,               -- relayer's submission tx hash
  seen_at        INTEGER NOT NULL,   -- Unix timestamp
  submitted_at   INTEGER,
  confirmed_at   INTEGER
);
```

On startup the relayer replays any events with status `'seen'` or `'submitted'` that have not been confirmed. This ensures a crash between seeing an event and confirming the submission does not cause a miss.

---

### Relayer Config File

```yaml
# /etc/relayer/config.yaml

# BSC connection
bsc:
  rpc_primary: "https://bsc-dataseed1.binance.org"
  rpc_fallback: "https://bsc-dataseed2.binance.org"
  lockbox_address: "0x<LockBox-contract-address>"
  confirmation_depth: 15              # wait 15 BSC blocks before treating event as final
  poll_interval_ms: 3000              # poll BSC for new events every 3 seconds
  rpc_timeout_ms: 5000

# Chain connection
chain:
  grpc_primary: "https://grpc.mainnet.example.com:9090"
  grpc_fallback: "https://grpc2.mainnet.example.com:9090"
  chain_id: "mychain-1"
  gas_adjustment: 1.4
  gas_prices: "0.025utoken"
  confirmation_depth: 1               # 1 chain block is final (CometBFT BFT finality)

# EOA key (BSC)
bsc_signer:
  type: keystore                       # or: kms (AWS KMS — recommended for Primary)
  keystore_path: "/etc/relayer/keystore/<address>.json"
  password_env: "BSC_KEYSTORE_PASSWORD"

# Operator key (chain)
chain_signer:
  type: keyring_file
  keyring_dir: "/etc/relayer/keyring"
  key_name: "relayer-operator"
  password_env: "CHAIN_KEYRING_PASSWORD"

# Nonce tracker
db:
  path: "/var/lib/relayer/nonces.db"
  backup_s3_bucket: "my-relayer-backup"
  backup_interval_minutes: 5           # litestream-style continuous backup

# Monitoring
metrics:
  prometheus_port: 9300

# Relayer tier (controls submission behaviour)
tier: candidate                        # candidate | secondary | primary
```

---

### BSC RPC Node Requirements

The relayer's BSC RPC endpoint must be **reliable and low-latency**. Public BSC RPC nodes (binance.org endpoints) are acceptable for testnet. For mainnet:

| Option | Recommendation |
|---|---|
| Run your own BSC full node | **Best** — no rate limits, no third-party dependency |
| QuickNode / Alchemy BSC | Acceptable — paid plan with dedicated endpoint |
| Public BSC RPC | **Not accepted for Primary** — rate-limited, no SLA |

Primary relayer must use a dedicated BSC RPC endpoint (owned or paid). Secondary and Candidate may use a paid shared endpoint with fallback.

If the BSC RPC endpoint is unreachable, the relayer enters a **BSC outage hold**: it stops submitting BSC→chain bridge-ins but continues processing chain→BSC bridge-outs using cached event data for up to 10 minutes. After 10 minutes with no BSC RPC contact, the relayer pauses all operations and pages the operator.

---

### Joining the Relayer Set

#### Step 1 — Apply

Submit the relayer operator application (provided by chain team):
- Organisation name and jurisdiction
- BSC RPC endpoint plan (own node / paid provider)
- EOA address (for BSC gas top-up tracking)
- Chain operator address (for governance proposal)
- On-call contact (15-minute response SLA)
- Testnet participation intention

#### Step 2 — Testnet Observation Period (≥ 2 weeks)

All relayer candidates must run on testnet for at least 2 weeks before governance consideration. Chain team monitors:
- Event pickup latency (time from BSC event to chain submission)
- Miss rate (events not submitted within 3 BSC confirmation blocks)
- Recovery behaviour after simulated RPC outage
- Nonce tracker crash-recovery test (process killed mid-inflight)

#### Step 3 — Governance Proposal

```bash
# Chain team submits on behalf of accepted candidate:
chaind tx gov submit-proposal add-relayer \
  --relayer-address <chain-operator-address> \
  --eoa-address <bsc-eoa-address> \
  --tier candidate \
  --moniker "<operator-name>" \
  --deposit 10000000utoken \
  --title "Add Relayer Candidate: <operator-name>" \
  --description "Testnet observation: 2 weeks, miss rate <0.5%, crash recovery verified" \
  --from <chain-team-key> \
  --chain-id mychain-1
```

Voting period: 7 days. Quorum: 33.4%. On passage, the operator address is registered in `x/bridge` state as a Candidate relayer.

#### Step 4 — Promotion to Secondary

After operating as Candidate for ≥ 4 weeks with a miss rate < 0.5%, the operator (or chain team) may submit a promotion proposal:

```bash
chaind tx gov submit-proposal promote-relayer \
  --relayer-address <chain-operator-address> \
  --new-tier secondary \
  --deposit 10000000utoken \
  --title "Promote Relayer to Secondary: <operator-name>" \
  --from <chain-team-key> \
  --chain-id mychain-1
```

#### Step 5 — Promotion to Primary

Primary promotion requires ≥ 8 weeks as Secondary, miss rate < 0.2%, and demonstrated BSC outage recovery. Primary promotion is a higher bar because only one Primary exists — the chain team will solicit community feedback before submitting the proposal.

#### Step 6 — Go-Live (After Governance Passage)

```bash
# Fund EOA address with BNB (budget: ~0.1 BNB/day at current gas prices; fund 30 days minimum)
# Fund chain operator address (budget: ~2 token/day; fund 30 days minimum)

# Start relayer
sudo systemctl enable relayer
sudo systemctl start relayer

# Verify: check first BSC→chain submission
chaind query bridge recent-inbounds --limit 5 --chain-id mychain-1
# Your relay-operator address should appear in submitted_by field

# Verify: check metrics endpoint
curl http://localhost:9300/metrics | grep relayer_events_processed
```

---

### Automatic Failover Behaviour

The `x/bridge` module tracks per-relayer miss counts in a sliding 10-minute window. When the Primary misses ≥ 3 events within the window:

```
x/bridge module emits RelayerDemoted event
Primary → demoted to Secondary tier (immediately, no governance)
Secondary → promoted to Primary tier (immediately, no governance)
Candidate → promoted to Secondary (if one Secondary slot is vacant)
```

This promotion is deterministic based on the ordered relayer list in `x/bridge` state (ordered by governance admission date, oldest first). No human intervention is required. The demoted Primary operator is notified via the chain event and is expected to fix their issue and request re-promotion via governance.

A demoted Primary that fixes their issue within 24 hours and submits evidence (logs, root cause) to chain team may be fast-tracked through governance re-promotion on a 48-hour emergency vote timeline.

---

### Security Requirements

| Requirement | Standard | Verification |
|---|---|---|
| EOA key storage | Encrypted keystore (testnet/secondary); AWS KMS recommended for Primary | Confirmed in checklist |
| Keystore password | Environment secret; never hardcoded in config file | Verified during testnet trial |
| BSC RPC endpoint | Dedicated paid or self-hosted for Primary; public fallback not accepted for Primary | Stated in application |
| EOA address funding | ≥ 30-day BNB gas reserve maintained | Monitored via balance alert |
| Operator address funding | ≥ 30-day chain token gas reserve maintained | Monitored via balance alert |
| Nonce tracker backup | Continuous backup to S3 (or equivalent); recovery tested | Crash-recovery test during testnet |
| Machine access | Key-based SSH only; password auth disabled | Self-reported |
| No key sharing | EOA and operator key used only for relayer role | Self-reported |

---

### Monitoring

```yaml
# Add to prometheus.yml
- job_name: 'relayer'
  static_configs:
    - targets: ['localhost:9300']

# Critical alerts:

- name: RelayerMissedEvent
  condition: relayer_missed_events_10m > 2
  severity: P1
  message: "Relayer missed ≥ 3 events — check BSC RPC and chain gRPC connectivity"

- name: RelayerBSCRPCDown
  condition: relayer_bsc_rpc_healthy == 0
  severity: P0
  message: "BSC RPC unreachable — bridge-in relay halted"

- name: RelayerChainGRPCDown
  condition: relayer_chain_grpc_healthy == 0
  severity: P0
  message: "Chain gRPC unreachable — all relay halted"

- name: RelayerEOABalanceLow
  condition: relayer_eoa_bnb_balance_wei < 50000000000000000  # < 0.05 BNB
  severity: P1
  message: "Relayer EOA BNB balance critical — top up immediately"

- name: RelayerOperatorBalanceLow
  condition: relayer_operator_token_balance_utoken < 5000000  # < 5 token
  severity: P1
  message: "Relayer chain operator balance low — top up within 24 hours"

- name: RelayerInfightNonceStuck
  condition: relayer_inflight_nonce_age_seconds > 120
  severity: P1
  message: "Inflight nonce unconfirmed for > 2 minutes — possible BSC/chain RPC issue"

- name: RelayerDemoted
  condition: relayer_current_tier_changed == 1
  severity: P0
  message: "Relayer tier changed — automatic failover triggered or manual promotion"
```

---

### Gas Budget Planning

Relayers spend gas on both chains. Budget conservatively:

**BSC side (`LockBox.release()`):**
- Gas per call: ~80,000 gas
- BSC gas price: ~3 Gwei (varies)
- Cost per call: 80,000 × 3 Gwei = 0.00024 BNB
- At 100 bridge-outs/day: ~0.024 BNB/day
- Fund 30-day reserve: ~0.72 BNB minimum (round up to 1 BNB for headroom)

**Chain side (`MsgBridgeIn`):**
- Gas per tx: ~150,000 gas
- Gas price: 0.025 utoken/gas
- Cost per tx: 3,750 utoken
- At 100 bridge-ins/day: 375,000 utoken/day
- Fund 30-day reserve: ~11.25 token minimum (round up to 15 token)

Adjust budgets based on actual bridge volume. The monitoring alerts above fire before either balance is critical.

---

### Bridge Relayer Admission Checklist

This checklist is signed by the relayer operator and reviewed by the chain team before the governance proposal is submitted.

**Identity and Agreement**
- [ ] Relayer operator agreement signed and submitted
- [ ] Legal entity name, jurisdiction, and contact confirmed
- [ ] On-call contact provided (15-minute SLA acknowledged)
- [ ] Promotion ladder process acknowledged (Candidate → Secondary → Primary)

**Key Management**
- [ ] EOA key generated; address recorded and shared with chain team
- [ ] Chain operator key generated; address recorded and shared with chain team
- [ ] Keystore files encrypted with strong password
- [ ] Keystore passwords stored in hardware password manager (not written down, not in config)
- [ ] Keystore files backed up to encrypted offline storage
- [ ] **No plaintext private key files on relayer machine**

**Connectivity**
- [ ] BSC RPC primary endpoint configured (dedicated paid or self-hosted for Primary)
- [ ] BSC RPC fallback endpoint configured (different provider from primary)
- [ ] Chain gRPC primary endpoint configured
- [ ] Chain gRPC fallback endpoint configured
- [ ] Connectivity to both chains verified (latency < 200 ms to each)

**Nonce Tracker**
- [ ] SQLite nonce tracker initialised and tested
- [ ] Continuous backup to S3 (or equivalent) configured
- [ ] Crash-recovery test completed: process killed mid-inflight; verified recovery on restart
- [ ] No duplicate submission observed during crash-recovery test

**Testnet Trial**
- [ ] ≥ 2 weeks of testnet operation completed (Candidate minimum)
- [ ] Miss rate over trial: ___% (must be < 0.5% for Candidate; < 0.2% for Primary)
- [ ] BSC RPC outage simulation test completed and observed by chain team
- [ ] Testnet performance report submitted with application

**Gas Reserves**
- [ ] EOA funded with ≥ 30-day BNB reserve
- [ ] Chain operator address funded with ≥ 30-day token reserve
- [ ] Gas budget documented and shared with chain team

**Monitoring**
- [ ] Prometheus metrics endpoint live (port 9300)
- [ ] All 7 monitoring alerts configured with correct thresholds
- [ ] P0 alerts paging on-call within 5 minutes
- [ ] Monitoring dashboard URL shared with chain team

**Operational Readiness**
- [ ] Systemd service configured for automatic restart on crash
- [ ] Upgrade SLA acknowledged (relayer binary updated within 12 hours of announcement)
- [ ] Validator / oracle coordination channel joined
- [ ] Governance participation committed
- [ ] Automatic failover behaviour understood (demotion on ≥ 3 misses in 10 minutes)

**Signature**

> I confirm that all items above are completed and accurate. I understand that a miss rate above 0.5% over any rolling 7-day period, a balance dropping to zero causing a halt, or repeated failure to respond to upgrade announcements may result in automatic tier demotion or a governance proposal to remove this operator from the relayer set.
>
> Operator: _________________________ Date: _____________

---

## Governance Playbook

This playbook covers every proposal type the chain supports. For each type it provides: what the proposal does, who typically submits it, the minimum deposit, the full CLI command template, lifecycle from draft to execution, and common mistakes. Read the governance lifecycle section first — it applies to all proposal types.

---

### Governance Lifecycle (All Proposal Types)

```
1. DRAFT
   Author writes proposal text and consults community
   (Forum post or Discord discussion — 3–5 days for non-urgent proposals)
   ↓
2. SUBMIT + DEPOSIT
   Author submits proposal on-chain with initial deposit
   If deposit < MinDeposit, proposal enters deposit period (max 14 days)
   Other accounts can add deposit; if MinDeposit not reached → proposal rejected, deposit burned
   ↓
3. VOTING PERIOD (7 days on mainnet)
   Validators and delegators vote: Yes / No / Abstain / NoWithVeto
   Quorum: 33.4% of staked supply must participate
   ↓
4. TALLY
   Pass:        Yes > 50% of non-Abstain votes AND quorum reached AND NoWithVeto < 33.4%
   Reject:      Yes ≤ 50% of non-Abstain votes
   Veto:        NoWithVeto ≥ 33.4% → deposit burned (not returned)
   No quorum:   quorum not reached → proposal fails, deposit returned
   ↓
5. EXECUTION (passed proposals only)
   Some types execute automatically at tally (parameter changes)
   Some types execute via x/upgrade at a future block height (software upgrades)
   Some types require chain team to execute manually after passage (contract migration, oracle/relayer set changes)
```

**Deposit amounts (mainnet):**

| Proposal type | Min deposit |
|---|---|
| Text / signalling | 100 token |
| Parameter change | 500 token |
| Software upgrade | 1,000 token |
| Contract migration | 500 token |
| Add oracle operator | 200 token |
| Remove oracle operator | 200 token |
| Add / promote relayer | 200 token |
| Emergency pause | 1,000 token (waived for cold multi-sig in an emergency) |

Deposit is returned to all depositors if the proposal passes or is rejected without veto. Deposit is burned if the proposal is vetoed.

---

### Proposal Type 1 — Text / Signalling

**What it does:** Records a community decision on-chain without automatically executing any state change. Used for: ratifying a policy, approving a roadmap item, formally recording a decision that will be implemented separately.

**Who submits:** Chain team or any community member with sufficient deposit.

```bash
chaind tx gov submit-proposal \
  --type text \
  --title "Ratify Bridge Fee Schedule v1" \
  --description "This proposal ratifies the bridge fee schedule as published at <forum-link>. Fees will be set via a subsequent parameter-change proposal once this passes." \
  --deposit 100000000utoken \
  --from <submitter-key> \
  --chain-id mychain-1 \
  --gas auto --gas-adjustment 1.4 \
  --fees 5000utoken

# Vote
chaind tx gov vote <proposal-id> yes \
  --from <validator-key> \
  --chain-id mychain-1

# Check tally in real time
chaind query gov tally <proposal-id> --chain-id mychain-1
```

**Common mistakes:**
- Submitting a text proposal for something that requires a parameter-change proposal (text proposals do not change state — a separate proposal is always needed after)
- Writing vague description text that the community cannot evaluate — include a specific forum link and a clear statement of what will happen after passage

---

### Proposal Type 2 — Parameter Change

**What it does:** Updates one or more module parameters in state immediately on passage. No code change, no upgrade required. Used for adjusting: governance voting period, quorum, oracle miss threshold, bridge confirmation depths, slashing fractions, etc.

**Who submits:** Chain team (most parameter changes affect security — do not let unknown parties change slashing or oracle parameters without high scrutiny).

**Important:** Parameter changes execute at the end of the voting period block, not at a future height. There is no rollback. Test on testnet first.

```bash
# Example: change oracle MissThreshold from 5% to 3%
chaind tx gov submit-proposal param-change \
  --title "Tighten Oracle Miss Threshold to 3%" \
  --description "Reduces MissThreshold in x/oracle from 5% to 3% to improve price feed reliability. Tested on testnet for 7 days with no unintended removals." \
  --param-changes '[
    {
      "subspace": "oracle",
      "key": "MissThreshold",
      "value": "\"0.03\""
    }
  ]' \
  --deposit 500000000utoken \
  --from <chain-team-key> \
  --chain-id mychain-1 \
  --gas auto --gas-adjustment 1.4 \
  --fees 5000utoken
```

**Multi-parameter change (single proposal):**
```bash
--param-changes '[
  {"subspace": "oracle",  "key": "MissThreshold",        "value": "\"0.03\""},
  {"subspace": "oracle",  "key": "VoteWindow",            "value": "\"100\""},
  {"subspace": "bridge",  "key": "BscConfirmationDepth",  "value": "\"20\""}
]'
```

**Subspace names for all custom modules:**

| Module | Subspace |
|---|---|
| x/oracle | `oracle` |
| x/bridge | `bridge` |
| x/certification | `certification` |
| x/settlement | `settlement` |
| x/validator | `validator` |
| x/milestone | `milestone` |
| x/staking | `staking` |
| x/slashing | `slashing` |
| x/gov | `gov` |

**Common mistakes:**
- Using wrong JSON quoting inside the `value` field (strings need inner escaped quotes: `"\"0.03\""`)
- Changing two parameters in two separate proposals when they should be atomic (combine into one proposal)
- Not testing on testnet: parameter changes to slashing or oracle can immediately remove validators or oracle operators

---

### Proposal Type 3 — Software Upgrade

**What it does:** Schedules a chain halt at a future block height and specifies the new binary name. The chain resumes only when validators upgrade. Full procedure is in the Network Upgrade Procedure section — this entry covers only the governance submission.

**Who submits:** Chain team only (requires coordinated validator preparation).

```bash
# Calculate upgrade height (see Network Upgrade Procedure for formula)
# Target: weekday, 09:00–15:00 UTC, ≥ 14 days from proposal submission

chaind tx gov submit-proposal software-upgrade "<upgrade-name>" \
  --title "Software Upgrade: <upgrade-name>" \
  --description "Upgrades chain to v<version>. Release notes: <url>. Binary SHA256: <sha256>. cosign verify: <command>. Validators must stage the new binary before block <height>." \
  --upgrade-height <height> \
  --upgrade-info '{"binaries":{"linux/amd64":"<download-url>?checksum=sha256:<sha256>"}}' \
  --deposit 1000000000utoken \
  --from <chain-team-key> \
  --chain-id mychain-1 \
  --gas auto --gas-adjustment 1.4 \
  --fees 5000utoken
```

**Checklist before submitting upgrade proposal:**
- [ ] New binary built and published with SHA256 and cosign signature
- [ ] Upgrade handler registered in `app.go` for the exact upgrade name in the proposal
- [ ] Devnet upgrade completed successfully
- [ ] Testnet upgrade completed successfully
- [ ] Upgrade height confirmed to be a weekday, 09:00–15:00 UTC
- [ ] Upgrade height at least 14 days from proposal submission
- [ ] Validator coordination channel notified before proposal submission

**Common mistakes:**
- Upgrade name in the proposal does not exactly match the handler name registered in `app.go` — chain halts permanently
- Upgrade height too soon: validators cannot stage binary in time
- Forgetting to post binary SHA256 in description: validators cannot verify the binary

---

### Proposal Type 4 — CosmWasm Contract Migration

**What it does:** Upgrades one of the four CosmWasm contracts (Constitution, Treasury, Reserve Fund, Governance) to a new code ID. This is a two-step process: first upload the new WASM binary (gets a code ID), then submit the migration proposal.

**Who submits:** Chain team (contract migrations bypass the Constitution check via `MsgMigrateContracts` — this is an intentional architectural decision documented in Phase 3).

```bash
# Step 1: Upload new WASM binary
chaind tx wasm store <contract>.wasm \
  --from <chain-team-key> \
  --chain-id mychain-1 \
  --gas auto --gas-adjustment 1.4 \
  --fees 5000utoken
# Note the code ID from tx result: query tx <hash> | jq '.logs[0].events[] | select(.type=="store_code")'

# Step 2: Submit migration proposal
chaind tx gov submit-proposal migrate-contract \
  --title "Migrate Constitution Contract to v<version>" \
  --description "Migrates Constitution contract from code ID <old> to code ID <new>. Changes: <description>. Audited by: <auditor>. Report: <url>." \
  --contract <contract-address> \
  --code-id <new-code-id> \
  --migrate-msg '{}' \
  --deposit 500000000utoken \
  --from <chain-team-key> \
  --chain-id mychain-1 \
  --gas auto --gas-adjustment 1.4 \
  --fees 5000utoken
```

**Migration message (`--migrate-msg`):** Pass `{}` for a no-op migration. If the new contract version requires migration state, pass the JSON the contract's `migrate` entrypoint expects.

**Finding contract addresses:**
```bash
chaind query wasm list-contracts-by-code <code-id> --chain-id mychain-1
# Or query by label:
chaind query wasm list-contract-by-creator <deployer-address> --chain-id mychain-1
```

**Common mistakes:**
- Uploading WASM binary without verifying it was compiled reproducibly (SHA256 of WASM must match CI build output)
- Migrating the wrong contract address (double-check with `chaind query wasm contract <address>`)
- Passing a non-empty `--migrate-msg` when the contract's `migrate` entrypoint expects `{}` — the migration tx reverts

---

### Proposal Type 5 — Add Oracle Operator

**What it does:** Adds an address to the `x/oracle` permissioned oracle set, allowing it to submit `MsgCommitOracleHash` and `MsgRevealOracleReport` transactions.

**Who submits:** Chain team, after reviewing the oracle operator's testnet performance and admission checklist.

```bash
chaind tx gov submit-proposal \
  --type text \
  --title "Add Oracle Operator: <operator-name>" \
  --description "Adds <operator-name> (<cosmos1...address>) to the permissioned oracle set. Testnet trial: 4 weeks, miss rate <0.3%. Feed sources: binance (primary), coinbase (fallback). HSM: YubiHSM 2 (attestation on file). Admission checklist signed." \
  --deposit 200000000utoken \
  --from <chain-team-key> \
  --chain-id mychain-1

# After proposal passes — chain team executes x/oracle AddOperator tx:
chaind tx oracle add-operator <cosmos1...address> \
  --moniker "<operator-name>" \
  --from <chain-team-key> \
  --chain-id mychain-1
```

Note: x/oracle operator management is executed via a privileged `MsgAddOracleOperator` that is gated on the governance module account. The governance module executes it automatically on proposal passage if the proposal is submitted as a `MsgExecLegacyContent` or a custom governance proposal type. Implement whichever pattern matches your `x/oracle` module's `MsgServer`.

**Common mistakes:**
- Submitting the proposal before the testnet trial is complete (minimum 4 weeks)
- Not including the testnet miss rate in the description — validators need this to make an informed vote

---

### Proposal Type 6 — Remove Oracle Operator

**What it does:** Removes an address from the `x/oracle` permissioned set. The operator immediately stops being able to submit price feeds after execution.

**Who submits:** Chain team (after sustained miss rate breach or security event) or any community member with evidence.

```bash
chaind tx gov submit-proposal \
  --type text \
  --title "Remove Oracle Operator: <operator-name>" \
  --description "Proposes removal of <operator-name> (<cosmos1...address>) from the permissioned oracle set. Reason: 30-day miss rate of 4.2% (threshold: 2%). Evidence: <link-to-miss-rate-report>. Operator was notified on <date> and has not responded." \
  --deposit 200000000utoken \
  --from <submitter-key> \
  --chain-id mychain-1

# After passage — chain team executes x/oracle RemoveOperator tx
chaind tx oracle remove-operator <cosmos1...address> \
  --from <chain-team-key> \
  --chain-id mychain-1
```

**Common mistakes:**
- Removing an operator without notifying them first and giving them time to respond — this should be a last resort after communication failure
- Not providing concrete miss-rate evidence in the description — proposal will likely be vetoed without it

---

### Proposal Type 7 — Add / Promote Bridge Relayer

**What it does:** Adds a new Candidate relayer to the `x/bridge` relayer set, or promotes an existing relayer from Candidate → Secondary → Primary.

**Who submits:** Chain team, after reviewing testnet performance and admission checklist.

```bash
# Add new Candidate
chaind tx gov submit-proposal \
  --type text \
  --title "Add Relayer Candidate: <operator-name>" \
  --description "Adds <operator-name> (chain: cosmos1..., EOA: 0x...) as Candidate bridge relayer. Testnet observation: 2 weeks, miss rate <0.4%. BSC RPC: dedicated QuickNode endpoint. Crash recovery test: passed. Admission checklist signed." \
  --deposit 200000000utoken \
  --from <chain-team-key> \
  --chain-id mychain-1

# After passage:
chaind tx bridge add-relayer <cosmos1...address> \
  --eoa-address 0x... \
  --tier candidate \
  --from <chain-team-key> \
  --chain-id mychain-1

# ─────────────────────────────────────

# Promote Candidate → Secondary
chaind tx gov submit-proposal \
  --type text \
  --title "Promote Relayer to Secondary: <operator-name>" \
  --description "Promotes <operator-name> from Candidate to Secondary tier. Candidate period: 6 weeks. Miss rate: 0.18% (threshold: 0.5%). BSC outage test: passed. Proposal by: chain team." \
  --deposit 200000000utoken \
  --from <chain-team-key> \
  --chain-id mychain-1

# After passage:
chaind tx bridge promote-relayer <cosmos1...address> --new-tier secondary \
  --from <chain-team-key> --chain-id mychain-1

# ─────────────────────────────────────

# Promote Secondary → Primary
chaind tx gov submit-proposal \
  --type text \
  --title "Promote Relayer to Primary: <operator-name>" \
  --description "Promotes <operator-name> from Secondary to Primary tier. Secondary period: 10 weeks. Miss rate: 0.07% (threshold: 0.2%). BSC outage recovery demonstrated twice. Community consultation: <forum-link>." \
  --deposit 200000000utoken \
  --from <chain-team-key> \
  --chain-id mychain-1
```

**Common mistakes:**
- Skipping the minimum observation periods (2 weeks Candidate, 4 weeks Secondary before Primary consideration)
- Promoting to Primary without community forum discussion — Primary has the most operational impact; validators expect to be consulted

---

### Proposal Type 8 — Remove Bridge Relayer

**What it does:** Removes a relayer from the `x/bridge` set entirely. Distinct from automatic demotion (which just changes tier). Full removal is a governance action.

```bash
chaind tx gov submit-proposal \
  --type text \
  --title "Remove Bridge Relayer: <operator-name>" \
  --description "Proposes full removal of <operator-name> from the bridge relayer set. Reason: operator has been unreachable for 14 days following automatic demotion from Primary on <date>. BSC EOA address 0x... has been inactive. Evidence: <link>." \
  --deposit 200000000utoken \
  --from <submitter-key> \
  --chain-id mychain-1

# After passage:
chaind tx bridge remove-relayer <cosmos1...address> \
  --from <chain-team-key> \
  --chain-id mychain-1
```

---

### Proposal Type 9 — Emergency Pause

**What it does:** Activates `EmergencyPause` on the bridge or specific CosmWasm contracts, blocking `ExecuteMsg` (write operations) while allowing `QueryMsg` (reads) to continue. Does not halt the chain. Used when an exploit or critical vulnerability is detected.

**Who submits:** Cold multi-sig (5-of-7) in a genuine emergency. The cold multi-sig can bypass the standard deposit requirement and has a shortened voting period (24 hours on mainnet — matching the `expedited_voting_period` governance parameter) by configuration.

**This is a time-critical action. The steps below are written for speed.**

```bash
# Step 1: Multi-sig prepares and signs the transaction (all signers in parallel)
# Signer A:
chaind tx gov submit-proposal emergency-pause \
  --scope bridge \
  --title "EMERGENCY: Pause Bridge — Active Exploit Suspected" \
  --description "Pausing bridge ExecuteMsg due to anomalous outflow detected at <timestamp>. LockBox address: 0x.... Block height detected: <height>. Team investigating. This pause blocks MsgBridgeIn and MsgBridgeOut until resolved." \
  --deposit 1000000000utoken \
  --generate-only \
  --from <multisig-address> \
  --chain-id mychain-1 \
  > emergency_pause_unsigned.json

# Each signer signs:
chaind tx sign emergency_pause_unsigned.json \
  --multisig <multisig-address> \
  --from <signer-key> \
  --chain-id mychain-1 \
  > sig_<signer>.json

# Step 2: Combine signatures (chain team coordinator)
chaind tx multisign emergency_pause_unsigned.json \
  <multisig-key-name> sig_A.json sig_B.json sig_C.json \
  --chain-id mychain-1 \
  > emergency_pause_signed.json

# Step 3: Broadcast
chaind tx broadcast emergency_pause_signed.json \
  --chain-id mychain-1

# Step 4: Immediately post in validator coordination channel:
# "EMERGENCY PAUSE PROPOSAL SUBMITTED — proposal ID <N> — 24h voting period —
#  Bridge ExecuteMsg will halt on passage. Vote YES to protect users.
#  Evidence: <link>"
```

**Scope options for emergency pause:**
- `bridge` — pauses `x/bridge` module and LockBox contract ExecuteMsg
- `contracts` — pauses all four CosmWasm contracts (ExecuteMsg only)
- `all` — pauses both (use only if scope of exploit is unclear)

**Lifting the pause (after exploit is patched):**
```bash
chaind tx gov submit-proposal emergency-unpause \
  --scope bridge \
  --title "Lift Emergency Pause: Bridge Resumed" \
  --description "Bridge exploit patched in v<version> deployed at upgrade height <height>. Post-mortem: <link>. Audit review completed. Safe to resume." \
  --deposit 1000000000utoken \
  --from <chain-team-key> \
  --chain-id mychain-1
```

**Common mistakes:**
- Pausing `all` when only the bridge is affected — CosmWasm contract downtime affects all dApp users unnecessarily
- Forgetting to post in the validator coordination channel immediately — validators need to know to vote quickly on a 24-hour window
- Lifting the pause before the patch is deployed and verified on testnet

---

### Proposal Type 10 — Validator Removal

**What it does:** Submits a governance proposal to tombstone or force-unbond a validator that has been persistently offline, double-signing, or behaving maliciously. Automatic slashing handles most cases — governance removal is for persistent inactivity or governance-level misconduct.

```bash
chaind tx gov submit-proposal \
  --type text \
  --title "Remove Validator: <moniker>" \
  --description "Proposes removal of validator <moniker> (operator address: cosmosvaloper1...). Reason: 21 consecutive days of >80% missed blocks without communication. Automatic slashing has applied. Delegation is at risk. Evidence: <link-to-validator-performance-data>. Team attempted contact on <dates>." \
  --deposit 500000000utoken \
  --from <submitter-key> \
  --chain-id mychain-1
```

Note: Cosmos SDK does not have a native `MsgForceUnbondValidator` in base governance. If your chain requires on-chain forced removal (not just social consensus), implement a custom `MsgSlashValidator` or `MsgJailValidator` in `x/validator` and gate it on the governance module account.

---

### Writing Good Proposal Descriptions

Every proposal description must answer these five questions, in order:

1. **What** — what will change on-chain if this passes?
2. **Why** — what problem does it solve or what opportunity does it capture?
3. **Evidence** — testnet results, audit reports, miss-rate data, or forum discussion link
4. **Risk** — what is the worst-case outcome if this goes wrong?
5. **Rollback** — can this be undone? If yes, how? If no, say so explicitly.

A description that answers all five takes under 3 minutes for a validator to evaluate. A description that does not answer them will receive Abstain votes at best, NoWithVeto at worst.

**Character limit:** Cosmos SDK does not enforce a hard limit, but keep descriptions under 4,000 characters. Longer descriptions are better hosted at a forum/IPFS link referenced in the description.

---

### Governance Calendar

On mainnet, governance proposals should not be submitted in a way that causes the voting period to overlap with:
- A planned software upgrade (validators are distracted)
- Major public holidays (reduced validator participation)
- A known oracle operator failover event

Chain team maintains a governance calendar in the validator coordination channel. Check it before submitting any proposal.

---

### Governance Monitoring Alerts

```yaml
- name: GovernanceProposalSubmitted
  condition: cosmos_governance_proposal_status{status="voting_period"} > 0
  severity: P2    # informational — notify all validators to review and vote
  message: "New governance proposal in voting period — review and vote before deadline"

- name: GovernanceQuorumAtRisk
  condition: cosmos_governance_proposal_tally_participation_pct < 25
  severity: P2
  message: "Governance proposal quorum at risk — <33.4% participation; promote voting"

- name: GovernanceEmergencyProposal
  condition: cosmos_governance_proposal_voting_period_hours < 26
  severity: P1    # emergency voting window — page all validators
  message: "Emergency governance proposal with <26h voting window — vote immediately"

- name: GovernanceProposalPassed
  condition: cosmos_governance_proposal_status{status="passed"} > 0
  severity: P2
  message: "Governance proposal passed — verify execution completed correctly"
```

---

### Quick Reference: Proposal Type Cheat Sheet

| Action | Proposal Type | Auto-executes? | Typical submitter |
|---|---|---|---|
| Record community decision | Text | No | Anyone |
| Change module parameter | Param change | Yes (on passage) | Chain team |
| Schedule chain upgrade | Software upgrade | At target height | Chain team |
| Upgrade CosmWasm contract | Contract migration | Yes (on passage) | Chain team |
| Add oracle operator | Text + manual exec | No (manual) | Chain team |
| Remove oracle operator | Text + manual exec | No (manual) | Chain team |
| Add relayer (any tier) | Text + manual exec | No (manual) | Chain team |
| Promote relayer tier | Text + manual exec | No (manual) | Chain team |
| Remove relayer | Text + manual exec | No (manual) | Chain team |
| Pause bridge / contracts | Emergency pause | Yes (on passage) | Cold multi-sig |
| Lift pause | Emergency unpause | Yes (on passage) | Chain team |
| Remove validator | Text (social) | No | Anyone with evidence |

---

## Incident Response Playbook

This playbook defines how the team responds to the five most likely production incidents. Each runbook has a severity classification, detection signal, immediate containment steps, resolution steps, escalation path, and post-mortem requirements. Read the severity definitions first — they set response time expectations.

---

### Severity Definitions

| Severity | Name | Response SLA | Who responds | Examples |
|---|---|---|---|---|
| SEV-1 | Critical | Page immediately; acknowledge within 5 min; all-hands | On-call engineer + incident commander + chain team lead | Chain halt, bridge exploit, key compromise |
| SEV-2 | High | Page on-call; acknowledge within 15 min | On-call engineer + one chain team member | Oracle price manipulation, validator double-sign, BSC RPC outage |
| SEV-3 | Medium | Notify on-call; acknowledge within 1 hour | On-call engineer | Single validator offline, single oracle miss spike, relayer demotion |
| SEV-4 | Low | Create ticket; address next business day | On-call engineer (async) | Monitoring alert with no user impact, disk usage warning |

**Incident Commander role:** For SEV-1 incidents, one person is designated Incident Commander. Their only job is coordination — they are not debugging. They run the coordination channel, call the all-hands, and own the post-mortem. The on-call engineer does the technical work and reports to the IC.

---

### Runbook 1 — Chain Halt

**Severity:** SEV-1

**What it is:** The chain stops producing blocks. No new transactions are processed. All user operations are blocked.

**Detection signals:**
- Monitoring alert: `cometbft_consensus_latest_block_time < now() - 30s`
- Block explorer shows no new blocks
- gRPC queries time out or return stale height

**Step 1 — Confirm the halt (T+0 to T+5 min)**
```bash
# Check from multiple nodes; if all agree, the chain is halted
chaind status --node tcp://grpc1.mainnet.example.com:26657 | jq '.SyncInfo.latest_block_height'
chaind status --node tcp://grpc2.mainnet.example.com:26657 | jq '.SyncInfo.latest_block_height'

# Check voting power online
chaind query staking validators --status BOND_STATUS_BONDED --chain-id mychain-1 \
  | jq '[.validators[].tokens | tonumber] | add'
# If < 67% of total bonded tokens are online → not enough VP to produce blocks
```

**Step 2 — Identify cause (T+5 to T+15 min)**
```bash
# On each validator node:
journalctl -u chaind --since "10 minutes ago" | grep -E "PANIC|ERROR|consensus"

# Common causes and their log signatures:
# 1. Not enough validators online → "Waiting for more peers..."
# 2. Consensus deadlock (byzantine) → "Duplicate vote" or repeated propose timeouts
# 3. App hash mismatch → "wrong Block.Header.AppHash"
# 4. OOM kill → "Out of memory: Kill process"
# 5. Disk full → "no space left on device"
```

**Step 3 — Containment by cause**

*Cause: < 67% voting power online (validators offline)*
```bash
# Page all validators via coordination channel immediately:
# "CHAIN HALT — SEV-1 — not enough validators online.
#  All validators: start your nodes NOW. Check status and report in."

# Monitor VP coming back online:
watch -n 5 'chaind query staking validators --status BOND_STATUS_BONDED --chain-id mychain-1 \
  | jq "[.validators[].tokens | tonumber] | add"'
# Chain auto-resumes when > 67% VP is online — no manual restart needed
```

*Cause: App hash mismatch (state divergence)*
```bash
# Identify which validator(s) have diverged:
# Ask each validator to run:
chaind query block <last-good-height> --chain-id mychain-1 | jq '.block.header.app_hash'
# The diverged validator will show a different app_hash

# Diverged validator must:
# 1. Stop their node
# 2. Restore state from last known-good snapshot (see Backup DR runbooks)
# 3. Restart and re-sync from that height
# Chain resumes when diverged VP rejoins with correct state
```

*Cause: OOM or disk full*
```bash
# Immediate: free resources
# Disk full:
du -sh ~/.chain/data/    # identify largest dirs
# Prune snapshots or expand volume, then restart:
sudo systemctl restart chaind

# OOM: increase memory or reduce cache size in config.toml
# [mempool] size = 5000 → 1000
# [statesync] snapshot-keep-recent = 10 → 2
sudo systemctl restart chaind
```

**Step 4 — All-clear (when blocks resume)**
```bash
# Verify block production has resumed on multiple RPC endpoints
chaind status | jq '.SyncInfo.latest_block_height'
# Monitor for 10 consecutive blocks before declaring resolved
```

**Step 5 — Post-mortem (within 48 hours)**
- Duration of halt (start block height → resume block height)
- Root cause
- How it was detected and how long detection took
- Timeline of actions taken
- What was done to prevent recurrence
- User impact (transactions lost; refunds if applicable)

**Escalation:** If chain does not resume within 60 minutes, escalate to chain team lead and all validators — a coordinated restart with a checkpoint may be required.

---

### Runbook 2 — Bridge Exploit / Anomalous Outflow

**Severity:** SEV-1

**What it is:** Tokens are being drained from the bridge in excess of normal activity — either via a contract exploit on the LockBox side or a bug in `x/bridge` minting logic.

**Detection signals:**
- Monitoring alert: `bridge_total_outflow_usd_1h` exceeds threshold (set to 3× 7-day average)
- LockBox contract balance drops faster than expected
- Unusual spike in `MsgBridgeIn` or `MsgBridgeOut` transaction count

**Step 1 — Confirm and scope (T+0 to T+5 min)**
```bash
# Check LockBox balance on BSC (use BSC RPC)
cast call <LockBox-address> "totalLocked()" --rpc-url <bsc-rpc>

# Check x/bridge module account balance on chain
chaind query bank balances <bridge-module-address> --chain-id mychain-1

# Check recent bridge transactions for anomalies
chaind query txs --events "message.action='/mychain.bridge.v1.MsgBridgeIn'" \
  --page 1 --limit 50 --chain-id mychain-1 \
  | jq '.txs[] | {height: .height, amount: .tx.body.messages[0].amount}'

# Assess: is the outflow from one address? Many addresses? One token type?
```

**Step 2 — Emergency pause (T+5 to T+10 min)**

Do not wait for full analysis. If outflow is confirmed anomalous, pause first.

```bash
# Cold multi-sig emergency pause — follow governance playbook Type 9 exactly
# Scope: bridge (not 'all' unless contracts are also affected)

# While multi-sig is assembling signatures, also call LockBox pause directly:
# LockBox has an owner-callable pause function for speed (faster than governance)
cast send <LockBox-address> "pause()" \
  --private-key <LockBox-owner-key> \
  --rpc-url <bsc-rpc>
# This halts release() and lock() on BSC side immediately

# Post in coordination channel:
# "BRIDGE EXPLOIT SUSPECTED — SEV-1 — Bridge pausing NOW.
#  Do not attempt bridge transactions. Investigating."
```

**Step 3 — Freeze attacker funds (T+10 to T+30 min)**
```bash
# Identify attacker address(es) from anomalous tx data
# Check if attacker has already bridged back to BSC
# If funds still on chain: submit MsgFreezeBridgeAccount if implemented in x/bridge
# If funds on BSC: contact BSC validators / centralized exchange compliance teams
#   with tx hashes and attacker EOA address for voluntary freeze

# Document all attacker addresses and amounts for post-mortem and potential recovery
```

**Step 4 — Root cause analysis (T+30 min to T+4 hours)**
```bash
# Replay the exploit transaction in a local fork
# For BSC side:
anvil --fork-url <bsc-rpc> --fork-block-number <block-before-exploit>
# Replay attacker tx and trace
cast run <exploit-tx-hash> --rpc-url http://localhost:8545 --debug

# For chain side:
# Check MsgBridgeIn handler — was nonce validation bypassed?
# Check mint logic — was amount tampered?
chaind query tx <exploit-chain-tx-hash> --chain-id mychain-1 | jq '.raw_log'
```

**Step 5 — Patch and lift pause (hours to days)**
- Fix identified in code
- Reproduce fix on devnet
- Audit fix (emergency expedited review — contact auditor directly, not via standard process)
- Deploy fix via software upgrade (coordinated with validators)
- Lift emergency pause via governance after upgrade confirmed

**Step 6 — Post-mortem (within 72 hours)**
- Total funds at risk and total recovered
- Exploit vector (which function, which invariant failed)
- Timeline from first anomalous tx to pause
- Root cause in code
- Changes to code, monitoring thresholds, and response process

**Escalation:** Immediately on SEV-1 confirmation — chain team lead, legal counsel (if user funds are at risk), security auditor.

---

### Runbook 3 — Oracle Price Manipulation

**Severity:** SEV-2

**What it is:** One or more oracle operators are submitting prices significantly outside the honest median, potentially manipulating `x/settlement` prices or `x/bridge` valuation.

**Detection signals:**
- Monitoring alert: `oracle_price_deviation_from_median_pct > 5` for any single operator
- Anomalous settlement transactions at unexpected prices
- `x/oracle` emits `PriceManipulationSuspected` event (if implemented)

**Step 1 — Identify the outlier operator (T+0 to T+10 min)**
```bash
# Query all oracle reveals for the last N blocks
chaind query oracle aggregate-prevotes --chain-id mychain-1
chaind query oracle aggregate-votes --chain-id mychain-1

# Compare each operator's submitted price against the on-chain median
# Operators deviating > 5% from median are suspect

# Check if the outlier's feed source has failed (could be accidental, not malicious)
# Contact the operator directly via coordination channel before assuming malice
```

**Step 2 — Assess impact**
```bash
# Was the manipulated price used in any settlement?
chaind query txs --events "message.action='/mychain.settlement.v1.MsgSettle'" \
  --page 1 --limit 20 --chain-id mychain-1

# Was the manipulated price used in any bridge valuation?
chaind query txs --events "message.action='/mychain.bridge.v1.MsgBridgeIn'" \
  --page 1 --limit 20 --chain-id mychain-1

# Quantify: how many transactions used the bad price? What is the value difference?
```

**Step 3 — Containment**

*If accidental (feed source failure):*
- Operator fixes their feed source
- Miss threshold removes them automatically if they keep submitting bad prices
- No further action needed; monitor for 30 minutes

*If sustained / suspected malicious:*
```bash
# Submit emergency governance proposal to remove oracle operator
# Use governance playbook Type 6 — Remove Oracle Operator
# Set voting period to 24 hours (emergency) if the operator has significant price influence

# Meanwhile: x/oracle median is resilient to one outlier if ≥ 3 operators remain honest
# Verify the median is stable with the suspect operator excluded from your mental model
```

**Step 4 — Post-mortem (within 48 hours)**
- Which operator submitted outlier prices
- Duration and magnitude of deviation
- Whether the deviation was used in any settled transactions
- User impact and remediation (if settlement at wrong price: governance may vote to compensate)
- Whether the operator was malicious or had a feed source failure

---

### Runbook 4 — Validator Double-Sign

**Severity:** SEV-2

**What it is:** A validator node has signed two different blocks at the same height and round — typically caused by running two validator nodes simultaneously (e.g., botched migration from one machine to another, Horcrux misconfiguration, or old node left running).

**Detection signals:**
- CometBFT emits `DuplicateVote` evidence on-chain
- `x/slashing` automatically tombstones the validator (5% slash + permanent jailing)
- Monitoring alert: validator missed from active set

**Step 1 — Immediate action by the affected validator (T+0)**

If you are the validator who double-signed:
```bash
# STOP BOTH NODES IMMEDIATELY
# Running both while investigating makes it worse
sudo systemctl stop chaind    # on both machines

# Do NOT restart either node until root cause is confirmed
# A tombstoned validator CANNOT rejoin the active set — tombstone is permanent
```

**Step 2 — Confirm tombstone status**
```bash
chaind query slashing signing-info <cosmosvalcons-address> --chain-id mychain-1 \
  | jq '{tombstoned: .tombstoned, jailed: .jailed, missed_blocks: .missed_blocks_counter}'
# tombstoned: true = permanent removal from active set; no recovery possible
```

**Step 3 — Recover delegator funds**
```bash
# Tombstoned validator must unbond — delegators are at risk of continued lockup
# The validator should send a message to delegators to redelegate immediately

# Validator can undelegate their self-delegation:
chaind tx staking unbond <cosmosvaloper-address> <amount>utoken \
  --from <operator-key> \
  --chain-id mychain-1

# Delegators redelegate to healthy validators:
chaind tx staking redelegate \
  <src-cosmosvaloper-address> <dst-cosmosvaloper-address> <amount>utoken \
  --from <delegator-key> --chain-id mychain-1
```

**Step 4 — Root cause analysis**

Common causes of double-sign:
- Old node left running during machine migration — **always stop the old node and verify it is down before starting the new one**
- Horcrux misconfiguration with two independent signers using file-based fallback
- Cloud instance "resumed" from snapshot while original was still running
- Kubernetes pod rescheduled without draining the old pod first

**Step 5 — Create new validator (if operator wishes to continue)**

After tombstone, the operator must create a brand-new validator with a new consensus key. All previous delegation is lost. This requires:
- New Horcrux key ceremony (the old key is compromised by the double-sign event)
- New `create-validator` transaction
- Community outreach to rebuild delegations
- Governance proposal if the operator held a system role (oracle, relayer)

**Step 6 — Post-mortem (within 48 hours)**
- How the double-sign occurred
- How it was detected
- Whether Horcrux prevented it or caused it (Horcrux should have a single-node-locking mechanism)
- Changes to deployment process to prevent recurrence

**Chain team action:** If the double-signing validator was a critical operator (oracle, relayer), immediately invoke Runbook 3 or bridge failover as appropriate.

---

### Runbook 5 — Key Compromise

**Severity:** SEV-1

**What it is:** A private key used in the system has been — or is suspected to have been — exposed to an unauthorized party. Applies to: validator consensus key, oracle HSM key, relayer EOA, LockBox owner key, cold multi-sig shard.

**Detection signals:**
- Unauthorized transaction from a known key
- HSM admin reports suspicious access
- Key found in logs, Git history, or public repository
- Machine hosting a key is confirmed compromised

**Step 1 — Contain immediately (T+0 to T+5 min)**

```bash
# 1. Isolate the compromised machine from the network immediately
#    (firewall rule or cloud security group update — do this FIRST)

# 2. Identify the key type and scope:
#    - Validator consensus key → attacker can double-sign → Runbook 4 applies
#    - Oracle HSM key → attacker can submit fake prices → Runbook 3 applies
#    - Relayer EOA → attacker can submit fraudulent bridge releases → Runbook 2 applies
#    - LockBox owner key → attacker can call pause/unpause → bridge risk
#    - Cold multi-sig shard → attacker has 1 of 5 shards; alert all other shard holders

# 3. For any key that signs financial transactions:
#    → Pause the affected system NOW (bridge, oracle set) via emergency governance
#    → Do not wait for root cause analysis
```

**Step 2 — Rotate the key**

*Validator consensus key:*
```bash
# Stop validator node immediately
sudo systemctl stop chaind

# New Horcrux key ceremony on new hardware (never reuse compromised hardware)
# After new key is ready, submit MsgEditValidator with new consensus pubkey
chaind tx staking edit-validator \
  --new-moniker "<moniker>" \
  --from <operator-key> \
  --chain-id mychain-1
# Note: consensus key rotation requires x/staking to support it (Cosmos SDK 0.50+)
# If not supported: create new validator with new key (old validator tombstones itself via jailing)
```

*Oracle HSM key:*
```bash
# Revoke compromised key in HSM immediately
yubihsm-shell
  > connect
  > session open 1 admin <password>
  > delete object <compromised-key-id> asymmetric-key
  > session close 1

# Generate new key on new or wiped HSM hardware
# Submit governance proposal to update oracle operator address to new key
# (Type 6 remove, then Type 5 add — or Type 2 parameter change if module supports direct rotation)
```

*Relayer EOA:*
```bash
# Transfer all remaining BNB from compromised EOA to a new safe address immediately
cast send <new-safe-address> --value <balance> \
  --private-key <compromised-key> --rpc-url <bsc-rpc>
# (Do this before the attacker drains it)

# Submit governance proposal to update relayer EOA address
# New EOA must be funded before going live
```

*Cold multi-sig shard:*
```bash
# Alert all other shard holders immediately
# 1 compromised shard of 7 does not give signing power (threshold is 5-of-7)
# Submit governance proposal to re-key the multi-sig:
#   - New key ceremony with all 5 (or replacement) shard holders
#   - Update multi-sig address in all contracts and module parameters
```

**Step 3 — Preserve forensic evidence (before wiping machine)**
```bash
# Do NOT wipe the compromised machine until forensics are complete
# Create a disk image for analysis
sudo dd if=/dev/sda of=/tmp/compromised-disk.img bs=4M
# Transfer image to a secure forensics system

# Collect:
# - Auth logs: /var/log/auth.log
# - Shell history: ~/.bash_history
# - Last logins: last -a
# - Open connections at time of incident: ss -tupn (if machine is still running)
# - Any suspicious processes: ps aux
```

**Step 4 — Assess attacker actions**
```bash
# For validator key: check for double-sign evidence
chaind query slashing signing-info <cosmosvalcons-address> --chain-id mychain-1

# For oracle key: check for price manipulation
chaind query oracle reveal-history --oracle-address <oracle-address> --limit 100

# For relayer EOA: check LockBox release() calls from EOA
cast logs --address <LockBox-address> \
  --from-block <compromise-block> --to-block latest \
  --event "Release(address,uint256)" --rpc-url <bsc-rpc>

# Quantify: what did the attacker do? What is the financial impact?
```

**Step 5 — Post-mortem (within 72 hours — security incidents get 72h not 48h)**
- How the key was exposed (logs in Git, SSH brute force, insider, phishing, supply chain)
- Timeline from exposure to detection
- What actions the attacker took with the key
- Financial impact and recovery
- All keys and systems that touched the compromised machine (assume all are compromised)
- Changes to key management, access controls, and monitoring

**Escalation:** Immediately on SEV-1 confirmation — chain team lead, legal counsel, security auditor, and if user funds are at risk, consider a public disclosure timeline (72-hour responsible disclosure is standard).

---

### Post-Mortem Template

Every SEV-1 and SEV-2 incident requires a written post-mortem published to the community within the stated deadline. Use this template:

```markdown
# Post-Mortem: <Incident Title>
**Date:** <YYYY-MM-DD>
**Severity:** SEV-1 / SEV-2
**Duration:** <start time> → <resolution time> (<total duration>)
**Status:** Resolved

## Summary
One paragraph. What happened, what was the impact, how was it resolved.

## Timeline (UTC)
| Time | Event |
|---|---|
| HH:MM | First alert / detection |
| HH:MM | Incident declared |
| HH:MM | Containment action taken |
| HH:MM | Root cause identified |
| HH:MM | Fix deployed |
| HH:MM | All-clear declared |

## Root Cause
Technical explanation of the root cause. Specific — not "human error" but exactly what decision or code path led to the incident.

## Impact
- Users affected: <N users / all users>
- Funds at risk: <amount or "none">
- Funds lost: <amount or "none">
- Transactions affected: <list or "none">
- Chain downtime: <duration or "none">

## What Went Well
- <detection was fast>
- <emergency pause worked correctly>
- <team responded within SLA>

## What Went Poorly
- <alert threshold was too high>
- <runbook step was unclear>
- <escalation path was not followed>

## Action Items
| Item | Owner | Due date |
|---|---|---|
| Reduce alert threshold for X | <name> | YYYY-MM-DD |
| Add monitoring for Y | <name> | YYYY-MM-DD |
| Update runbook step Z | <name> | YYYY-MM-DD |

## Recurrence Prevention
What specific, verifiable changes will prevent this exact incident from happening again?
```

---

### Incident Response Contacts

Maintain this list in a secure, offline-accessible location (not only in a system that may itself be affected by the incident):

| Role | Primary contact | Backup contact | Escalation SLA |
|---|---|---|---|
| On-call engineer | Rotating schedule | Previous on-call | Acknowledge within 5 min (SEV-1) |
| Incident Commander | Chain team lead | Senior engineer | Available within 15 min |
| Security auditor | <auditor contact> | <auditor backup> | Emergency response within 4 hours |
| Legal counsel | <legal contact> | — | Available within 2 hours |
| BSC compliance contact | <binance contact> | — | Asset freeze requests |
| Validator coordination | Discord / Telegram channel | Email list | Broadcast immediately |

---

### Incident Response Monitoring Dashboard

Maintain a dedicated incident dashboard (separate from normal monitoring) that shows at a glance:

```
Chain status:          ● LIVE / ● HALTED
Bridge status:         ● LIVE / ● PAUSED / ● EXPLOIT SUSPECTED
Oracle set status:     ● ALL HEALTHY / ● N OPERATORS DEGRADED
Relayer status:        ● PRIMARY HEALTHY / ● FAILOVER ACTIVE
Active incidents:      <count>
Last incident:         <date> — <title>
Open post-mortem items: <count>
```

This dashboard is the first thing the Incident Commander opens. It must be accessible without any authentication that could itself be affected by an incident (e.g., host it on a separate domain with no dependency on the chain's own infrastructure).

---

## Performance Optimization Guide

Every layer of the system has a performance ceiling. This guide works through each layer from the chain consensus engine outward to the API and client. Apply these in order — fixing the chain first makes all downstream improvements meaningful; fixing the API before fixing the chain wastes effort.

---

### Layer 1 — CometBFT Consensus Engine

CometBFT block time is the fundamental clock of the entire system. Everything downstream is bounded by it.

#### Target: 3-second block time (down from default 5-second)

```toml
# config.toml on all validator and full nodes
[consensus]
timeout_propose          = "1500ms"   # default 3000ms — how long to wait for proposer
timeout_propose_delta    = "300ms"    # default 500ms
timeout_prevote          = "500ms"    # default 1000ms
timeout_prevote_delta    = "300ms"    # default 500ms
timeout_precommit        = "500ms"    # default 1000ms
timeout_precommit_delta  = "300ms"    # default 500ms
timeout_commit           = "800ms"    # default 1000ms — time after block before next round

# Result: ~3.6s block time under normal validator latency
# For ≥ 6 geographically distributed validators, 3s is achievable
# Do NOT go below timeout_commit = 500ms — validators need time to persist state
```

#### Increase transaction throughput per block

```toml
# config.toml
[mempool]
size              = 10000    # default 5000 — max pending txs in mempool
max_txs_bytes     = 104857600  # default 20MB → 100MB — total mempool size
cache_size        = 20000    # default 10000 — tx dedup cache

# In app.toml
[api]
max-recv-msg-size = 104857600  # 100MB — match mempool size

# In genesis.json — set per-block gas limit high enough
# consensus_params.block.max_bytes = 4194304   # 4MB (default 1MB)
# consensus_params.block.max_gas   = 40000000  # 40M gas (default -1 = unlimited; set a real limit)
```

#### Enable ABCI++ PrepareProposal for block optimization

```go
// In app.go — implement PrepareProposal to order and filter transactions
func (app *App) PrepareProposal(ctx sdk.Context, req *abci.RequestPrepareProposal) (*abci.ResponsePrepareProposal, error) {
    // 1. Pull oracle commit txs to front of block (time-sensitive)
    // 2. Pull oracle reveal txs second
    // 3. Fill remainder with normal txs up to MaxBytes
    // 4. Drop txs that exceed gas limit rather than including partial blocks
    //    (partial blocks waste block space and slow tally)
    txs := app.reorderForOracle(req.Txs, req.MaxTxBytes)
    return &abci.ResponsePrepareProposal{Txs: txs}, nil
}
```

This ensures oracle commits and reveals always land in their intended blocks, preventing miss penalties that come from mempool ordering accidents under high load.

#### Mempool: switch to priority mempool

```go
// In app.go NewApp():
app.SetMempool(mempool.NewPriorityMempool(
    mempool.DefaultPriorityNonceMempoolConfig(),
))
// Transactions with higher fees float to the top
// Oracle and relayer txs should be submitted with higher fee to guarantee ordering
```

---

### Layer 2 — IAVL Store and State Performance

The IAVL Merkle tree is the biggest CPU and I/O consumer in the Cosmos SDK. Tuning it correctly is the highest-leverage chain-level optimization.

#### Switch from IAVL to IAVL v2 (Cosmos SDK 0.50+)

IAVL v2 rewrites the tree storage to use a flat key-value layout instead of a tree-of-nodes layout. Benchmark results from real chains show 3–5× reduction in state I/O for read-heavy workloads.

```toml
# app.toml
[store]
app-db-backend = "pebbledb"   # PebbleDB outperforms GoLevelDB significantly for IAVL v2
                               # Benchmark: 2–3× higher write throughput, lower read latency
```

```go
// go.mod — use cosmossdk.io/store v2 when available in your SDK version
// In NewApp, use store.NewCommitMultiStore with StoreV2Options
```

#### Aggressive pruning — do not store unnecessary historical state

```toml
# app.toml
[pruning]
pruning          = "custom"
pruning-keep-recent = "100"     # keep last 100 blocks in state store
pruning-interval    = "10"      # prune every 10 blocks

# Archive nodes (full history for indexing): pruning = "nothing"
# Validator nodes: pruning = "custom" with above settings
# Sentry/RPC nodes serving queries: pruning = "custom" keep-recent = "1000"
```

Pruning reduces the IAVL tree size, which directly reduces Merkle proof computation time per block.

#### PebbleDB tuning for high write throughput

```toml
# app.toml — PebbleDB-specific options (Cosmos SDK 0.50+)
[store.pebbledb]
max-open-files      = 4096
l0-compaction-threshold = 8     # default 4 — delay compaction longer for burst writes
l0-stop-writes-threshold = 24   # default 12
lru-cache-size-bytes = 4294967296   # 4 GB LRU cache (use 25% of validator RAM)
bytes-per-sync       = 4194304      # 4 MB write batch size
```

---

### Layer 3 — NATS JetStream Throughput

NATS JetStream is the message bus between the ingestion service and the projection service. Under high chain activity (many transactions per block), the ingestion service can produce messages faster than projection consumes them.

#### Maximize publish throughput from ingestion

```go
// In ingestion service — publish in async batches, not one-by-one
// Subject names must match the stream definitions in the retention table.
// Ingestion publishes to the chain account (account:chain); subjects are e.g. "chain.events.block".
// The three NATS accounts (account:chain, account:bridge, account:stream) are isolated
// JetStream credential namespaces, not literal subject strings.
js, _ := nc.JetStream()

// BAD: synchronous publish per tx — 6000 tx/block × 1ms RTT = 6 seconds per block
for _, tx := range block.Txs {
    js.Publish("chain.events.block", marshalTx(tx))  // chain account, subject "chain.events.block"
}

// GOOD: async publish with callback — all messages in flight simultaneously
type pubResult struct { pa *nats.PubAck; err error }
results := make([]chan pubResult, len(block.Txs))
for i, tx := range block.Txs {
    ch := make(chan pubResult, 1)
    results[i] = ch
    go func(msg []byte, c chan pubResult) {
        pa, err := js.PublishAsync(subject, msg)
        c <- pubResult{pa, err}
    }(marshalTx(tx), ch)
}
// Wait for all acks
for _, ch := range results {
    r := <-ch
    if r.err != nil { /* handle */ }
}
```

#### NATS server tuning for high throughput

```conf
# /etc/nats/nats.conf
max_payload:         8388608    # 8 MB max message (default 1 MB)
write_deadline:      "5s"
max_connections:     10000
max_pending_size:    134217728  # 128 MB per-connection pending buffer

jetstream {
  store_dir:     "/var/lib/nats/jetstream"
  max_mem_store: 4294967296     # 4 GB in-memory stream buffer
  max_file_store: 137438953472  # 128 GB on-disk stream storage
}
```

#### Stream configuration for minimum latency

```go
// When creating the JetStream stream — tune for throughput
// NOTE: In this architecture there are three separate JetStream streams, one per NATS account
// (account:chain, account:bridge, account:stream). Subject names use dots as JetStream separators
// and must match the retention table (chain.events.*, bridge.bsc.*, bridge.sig.*, chain.stream.*).
// The example below shows the "chain" account stream configuration (account:chain):
js.AddStream(&nats.StreamConfig{
    Name:        "CHAIN",                      // stream name in the chain account
    Subjects:    []string{"chain.events.>"},   // matches chain.events.block, chain.events.tx, etc.
    Storage:     nats.FileStorage,
    Replicas:    3,               // mandatory R=3 for durability — matches 3-node JetStream cluster requirement
    MaxAge:      24 * time.Hour,
    Compression: nats.S2Compression,  // S2 (Snappy-compatible) — fast compression, good ratio
    // Critical for throughput:
    NoAck:       false,           // keep acks — needed for at-least-once guarantee
    Discard:     nats.DiscardOld, // drop old messages under backpressure (not new ones)
    MaxMsgs:     10_000_000,
    MaxBytes:    int64(32 * 1024 * 1024 * 1024), // 32 GB
})
```

#### Consumer: pull consumer with batch fetch

> **Architecture note — push vs. pull:** The architecture section describes projection as a "push subscriber (fan-out)" which is the conceptual delivery model: every projection replica receives every message from the `account:chain` NATS account. In practice, **pull consumers with batch fetch are the implementation pattern** that achieves this fan-out efficiently in JetStream. Each replica holds its own durable pull consumer; JetStream delivers a copy of each message to every consumer independently, preserving fan-out semantics while allowing the batch-fetch throughput optimization below.

```go
// In projection service — pull in batches, not one-by-one
// Subject "chain.events.block" is in account:chain; each projection replica
// has its own durable consumer (e.g., "projection-consumer-A", "projection-consumer-B").
sub, _ := js.PullSubscribe("chain.events.block", "projection-consumer")

for {
    // Fetch up to 500 messages per batch — dramatically reduces RTT overhead
    msgs, err := sub.Fetch(500, nats.MaxWait(2*time.Second))
    if err != nil && err != nats.ErrTimeout { log.Error(err); continue }

    batch := make([]*ProcessedMsg, 0, len(msgs))
    for _, msg := range msgs {
        batch = append(batch, process(msg))
        msg.Ack()
    }
    if len(batch) > 0 {
        bulkInsert(batch) // single PostgreSQL COPY for entire batch
    }
}
```

---

### Layer 4 — PostgreSQL Write DB (Ingestion)

The Write DB receives every transaction the chain produces. At 1000 tx/block × 10 blocks/minute = 10,000 inserts/minute under light load. At peak (block gas limit saturated), this can reach 500,000 inserts/minute.

#### Use COPY instead of INSERT for bulk ingestion

```go
// BAD: individual INSERT — 100,000 round trips
for _, tx := range txs {
    db.Exec("INSERT INTO transactions VALUES ($1,$2,...)", tx.Hash, tx.Height, ...)
}

// GOOD: PostgreSQL COPY — single network round trip for 100,000 rows
conn, _ := pgxpool.Acquire(ctx)
defer conn.Release()

_, err := conn.Conn().CopyFrom(
    ctx,
    pgx.Identifier{"transactions"},
    []string{"hash", "height", "sender", "type", "amount", "created_at"},
    pgx.CopyFromSlice(len(txs), func(i int) ([]any, error) {
        return []any{
            txs[i].Hash, txs[i].Height, txs[i].Sender,
            txs[i].Type, txs[i].Amount, txs[i].CreatedAt,
        }, nil
    }),
)
// COPY is 10–50× faster than batched INSERT for large row counts
```

#### Partition tables by block range, not date

```sql
-- Partitioning by month is too coarse — queries for "last 1000 blocks" span partition boundaries
-- Partition by block height range for chain data

CREATE TABLE transactions (
    height      BIGINT NOT NULL,
    hash        TEXT NOT NULL,
    sender      TEXT NOT NULL,
    type        TEXT NOT NULL,
    amount      NUMERIC NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL
) PARTITION BY RANGE (height);

-- Create partitions in advance — 1M blocks per partition
CREATE TABLE transactions_0_1m    PARTITION OF transactions FOR VALUES FROM (0)          TO (1000000);
CREATE TABLE transactions_1m_2m   PARTITION OF transactions FOR VALUES FROM (1000000)    TO (2000000);
-- Automate partition creation: create the next partition when current reaches 80% capacity

-- Index per partition (automatically inherited):
-- PostgreSQL only scans the relevant partition for height-range queries → massive speedup
CREATE INDEX ON transactions (height, sender);
CREATE INDEX ON transactions (sender, created_at DESC);
```

#### Write DB connection pool and PostgreSQL settings

```sql
-- postgresql.conf tuning for write-heavy workload:
shared_buffers         = 8GB       -- 25% of RAM (32GB machine)
work_mem               = 64MB      -- per-sort operation; high for bulk loads
maintenance_work_mem   = 2GB       -- for VACUUM, CREATE INDEX
wal_buffers            = 64MB      -- default 4MB; increase for high write throughput
checkpoint_completion_target = 0.9 -- spread checkpoint I/O over 90% of checkpoint interval
checkpoint_timeout     = 15min     -- default 5min; longer = fewer checkpoints = less I/O spike
max_wal_size           = 4GB       -- allow larger WAL before checkpoint forced
random_page_cost       = 1.1       -- SSD: reduce from default 4.0; enables index-only scans
effective_cache_size   = 24GB      -- tell planner how much OS cache is available
synchronous_commit     = on        -- KEEP ON: data safety; do not turn off for ingestion speed
                                   -- Instead: use COPY batching to reduce commit frequency
```

```go
// pgxpool config for ingestion_writer
pool, _ := pgxpool.NewWithConfig(ctx, &pgxpool.Config{
    ConnConfig:            connCfg,
    MaxConns:              20,           // ingestion is CPU-bound not connection-bound
    MinConns:              5,
    MaxConnLifetime:       30 * time.Minute,
    MaxConnIdleTime:       5 * time.Minute,
    HealthCheckPeriod:     1 * time.Minute,
})
```

#### Indexes: create only what queries actually use

```sql
-- Every unused index adds write overhead (updated on every INSERT/COPY)
-- Run this monthly to find indexes never used:
SELECT schemaname, tablename, indexname, idx_scan
FROM pg_stat_user_indexes
WHERE idx_scan = 0
ORDER BY schemaname, tablename;

-- Drop unused indexes immediately — each one slows every INSERT

-- Essential indexes only:
CREATE INDEX CONCURRENTLY idx_tx_sender_height ON transactions (sender, height DESC);
CREATE INDEX CONCURRENTLY idx_tx_type_height   ON transactions (type, height DESC);
-- Use CONCURRENTLY: does not lock table during creation
```

---

### Layer 5 — PostgreSQL Read DB (Projection / Query)

The Read DB serves all user-facing API queries. It must be fast under concurrent read load.

#### Connection pooling with PgBouncer in transaction mode

```ini
# /etc/pgbouncer/pgbouncer.ini
[databases]
readdb = host=127.0.0.1 port=5432 dbname=readdb

[pgbouncer]
pool_mode          = transaction    # release connection after each transaction
max_client_conn    = 10000          # frontend connections (from API servers)
default_pool_size  = 50             # backend connections to PostgreSQL per database
min_pool_size      = 10
reserve_pool_size  = 10
reserve_pool_timeout = 3
server_idle_timeout = 600
```

Transaction-mode pooling means 10,000 concurrent API clients can share 50 actual PostgreSQL connections. Without PgBouncer, each API client holds a PostgreSQL connection open for its lifetime — PostgreSQL degrades sharply above ~500 connections.

#### Materialized views for expensive aggregates

```sql
-- BAD: compute account balance on every API request by summing all transactions
SELECT SUM(amount) FROM transactions WHERE sender = $1 AND type = 'credit';
-- This scans potentially millions of rows per request

-- GOOD: maintain a materialized summary refreshed by the projection service
CREATE MATERIALIZED VIEW account_balances AS
SELECT
    address,
    SUM(CASE WHEN type = 'credit' THEN amount ELSE 0 END) -
    SUM(CASE WHEN type = 'debit'  THEN amount ELSE 0 END) AS balance,
    COUNT(*) AS tx_count,
    MAX(height) AS last_activity_height
FROM transactions
GROUP BY address;

CREATE UNIQUE INDEX ON account_balances (address);

-- Projection service refreshes after each batch insert (not CONCURRENTLY for small tables;
-- CONCURRENTLY for large tables to avoid locking):
REFRESH MATERIALIZED VIEW CONCURRENTLY account_balances;
```

#### Partial indexes for hot query patterns

```sql
-- If 90% of queries are for the last 30 days, index only recent data:
CREATE INDEX idx_tx_recent_sender ON transactions (sender, created_at DESC)
WHERE created_at > NOW() - INTERVAL '30 days';
-- This index is tiny (recent rows only) but covers the vast majority of queries

-- For bridge-specific queries:
CREATE INDEX idx_bridge_pending ON bridge_events (created_at DESC)
WHERE status = 'pending';
-- Tiny index (only pending rows) — most rows are 'confirmed' and excluded
```

#### Read replica routing in the API layer

```go
// Route read queries to replica; write queries to primary
type DBPool struct {
    primary  *pgxpool.Pool  // projection_reader user — SELECT only
    // For API: always use primary (Read DB has no writes from API)
}

// In API handlers — use prepared statements for hot endpoints
type Queries struct {
    getAccountBalance *pgxpool.Pool
    getRecentTxs      *pgxpool.Pool
}

// Prepare once at startup, execute many times
// pgx supports prepared statements natively — avoids re-parsing on every request
```

---

### Layer 6 — API Server (Express + gRPC)

> **Implementation note:** The backend API server is a **Go gRPC service** (module/api), not a Node.js/Express server. The TypeScript/Node.js code samples in this section illustrate the *concepts* of HTTP/2, response compression, Redis caching, and SSE backpressure — these patterns apply equally to Go implementations using `net/http`, `grpc-gateway`, and `go-redis`. Map each sample to its Go equivalent when implementing. The grpc-gateway Deployment (separate from `module/api`) handles HTTP/1.1↔gRPC transcoding; Envoy handles TLS termination. HTTP/2 is already in use at the transport layer automatically for gRPC-Web routes.

#### HTTP/2 for all external traffic

```typescript
// CONCEPTUAL ILLUSTRATION (TypeScript) — implement the equivalent pattern in Go/Envoy
// api-server/src/index.ts — enable HTTP/2 via spdy or native node http2
import http2 from 'http2';
import fs from 'fs';

const server = http2.createSecureServer({
    key:  fs.readFileSync('/etc/ssl/private/server.key'),
    cert: fs.readFileSync('/etc/ssl/certs/server.crt'),
    allowHTTP1: true,  // fallback for clients that don't support HTTP/2
});
// HTTP/2 multiplexes multiple requests over one TCP connection
// For a wallet client making 10 concurrent API calls: 1 connection vs 10 connections
```

#### Response compression

```typescript
import compression from 'compression';

app.use(compression({
    level: 6,           // zlib level 6: good ratio, fast enough for API responses
    threshold: 1024,    // only compress responses > 1KB
    filter: (req, res) => {
        // Never compress SSE streams — breaks chunked transfer
        if (req.path.includes('/stream')) return false;
        return compression.filter(req, res);
    },
}));
```

#### Response caching for stable data

```typescript
import { createClient } from 'redis';
const redis = createClient({ url: process.env.REDIS_URL });

// Cache immutable data: historical blocks, confirmed transactions
async function getCachedBlock(height: number) {
    const key = `block:${height}`;
    const cached = await redis.get(key);
    if (cached) return JSON.parse(cached);

    const block = await db.query('SELECT * FROM blocks WHERE height = $1', [height]);
    // Historical blocks never change — cache indefinitely
    await redis.set(key, JSON.stringify(block), { EX: 86400 * 30 }); // 30 days
    return block;
}

// Cache mutable data with short TTL: account balances, validator set
async function getCachedBalance(address: string) {
    const key = `balance:${address}`;
    const cached = await redis.get(key);
    if (cached) return JSON.parse(cached);

    const balance = await db.query(
        'SELECT balance FROM account_balances WHERE address = $1', [address]
    );
    await redis.set(key, JSON.stringify(balance), { EX: 6 }); // 6s = ~1 block
    return balance;
}
```

#### Connection keep-alive and request batching

```typescript
// Keep TCP connections alive — eliminates handshake overhead for repeat clients
app.use((req, res, next) => {
    res.setHeader('Connection', 'keep-alive');
    res.setHeader('Keep-Alive', 'timeout=30');
    next();
});

// Batch endpoint — clients can send multiple queries in one HTTP request
app.post('/batch', async (req, res) => {
    const requests: BatchRequest[] = req.body.requests;
    // Execute all requests in parallel
    const results = await Promise.all(requests.map(r => handleRequest(r)));
    res.json({ results });
});
// A wallet loading its dashboard makes 1 request instead of 8
```

#### Server-Sent Events — NOT USED IN THIS SYSTEM

> **⚠ Architecture note:** The `/api/stream` WebSocket and SSE routes were **removed** (see Phase 5.6 Envoy config). All real-time streaming is served exclusively via **gRPC server-streaming through `/api/grpcweb/`** (see Phase 5.5). The SSE pattern below is retained for reference only — showing the backpressure and heartbeat technique — but **do not implement an SSE endpoint in this system**. Any SSE `res.setHeader('Content-Type', 'text/event-stream')` route would contradict the architecture decision to eliminate direct NATS→browser bypass routes.

**Reference only — gRPC server-streaming equivalent handles these concerns:**
- Heartbeat / keep-alive: gRPC ping frames on the HTTP/2 connection; Envoy `idle_timeout: 600s`
- Backpressure: in-process per-client channel buffer (64 events); full buffer → `ResourceExhausted` → SDK reconnect
- Slow consumer eviction: `ResourceExhausted` gRPC status code; client SDK exponential backoff reconnect

```typescript
// ⛔ DO NOT USE — SSE reference pattern only
// This pattern is shown for the heartbeat/backpressure technique.
// In this system, use gRPC server-streaming (see Phase 5.5 StreamChainStats / StreamAccountEvents).
app.get('/stream/account/:address', (req, res) => {
    res.setHeader('Content-Type', 'text/event-stream');
    res.setHeader('Cache-Control', 'no-cache');
    res.setHeader('X-Accel-Buffering', 'no'); // disable Nginx buffering for SSE

    // Heartbeat every 15s — prevents proxy timeout disconnection
    const heartbeat = setInterval(() => res.write(':heartbeat\n\n'), 15_000);

    // Backpressure: if client is slow, drain the buffer before sending more
    const sub = nats.subscribe(`account.${req.params.address}`);
    (async () => {
        for await (const msg of sub) {
            if (res.writableLength > 65536) {
                // Buffer > 64KB — client is too slow; skip this message
                continue;
            }
            res.write(`data: ${msg.data}\n\n`);
        }
    })();

    req.on('close', () => {
        clearInterval(heartbeat);
        sub.unsubscribe();
    });
});
```

---

### Layer 7 — CosmWasm Contract Optimization

CosmWasm contracts run inside the chain's WASM runtime. Every `ExecuteMsg` and `QueryMsg` consumes gas and adds to block processing time.

#### Minimize storage reads — batch state access

```rust
// BAD: read each field separately — N storage reads
let owner = OWNER.load(deps.storage)?;
let paused = PAUSED.load(deps.storage)?;
let fee = FEE.load(deps.storage)?;

// GOOD: store related fields in one item — 1 storage read
#[cw_serde]
pub struct Config {
    pub owner: Addr,
    pub paused: bool,
    pub fee: Uint128,
}
const CONFIG: Item<Config> = Item::new("config");
let cfg = CONFIG.load(deps.storage)?;  // single read
```

#### Use raw queries for hot read paths

```rust
// GOOD: raw storage read avoids deserialization for simple checks
// 3–5× faster than Item::load for existence checks
let key = OWNER.key();
let exists = deps.storage.get(key.as_slice()).is_some();

// For QueryMsg handlers that are called frequently:
// Return only the fields the caller needs — not the entire Config struct
#[cw_serde]
pub enum QueryMsg {
    IsPaused {},           // returns bool only
    GetFee {},             // returns Uint128 only
    GetConfig {},          // returns full Config (used rarely)
}
```

#### Reserve gas budget at entry point

```rust
pub fn execute(
    deps: DepsMut,
    env: Env,
    info: MessageInfo,
    msg: ExecuteMsg,
) -> Result<Response, ContractError> {
    // NOTE: cosmwasm-std does NOT expose a gas_remaining() function on Env.
    // CosmWasm's gas metering is automatic: the VM charges gas per WASM instruction
    // and halts with an OutOfGas error if the limit is exceeded before return.
    // Do NOT write env.gas_remaining() — it will fail to compile.
    //
    // To guard against out-of-gas mid-execution, ensure the caller sets a
    // sufficient gas limit (enforce minimum via msg validation or ante handler),
    // and keep each ExecuteMsg handler bounded in its WASM instruction count.
    // You can simulate worst-case gas cost with: chaind tx simulate ...
    //
    // route to handler
    match msg {
        ExecuteMsg::SomeAction { .. } => execute_some_action(deps, env, info),
    }
}
```

#### Compile with optimization

```bash
# ALWAYS use the CosmWasm Rust optimizer — never deploy debug builds
docker run --rm -v "$(pwd)":/code \
  --mount type=volume,source="$(basename "$(pwd)")_cache",target=/code/target \
  --mount type=volume,source=registry_cache,target=/usr/local/cargo/registry \
  cosmwasm/optimizer:0.16.0

# Output: artifacts/<contract>.wasm
# Optimizer applies: wasm-opt -O3, strip debug symbols, dead code elimination
# Size reduction: typically 40–60% vs unoptimized build
# Gas reduction: directly proportional to WASM instruction count — smaller = cheaper

# Verify optimization:
wasm-objdump -h artifacts/<contract>.wasm | grep -E "Code|Data"
# Code section < 200KB is excellent; > 500KB needs investigation
```

---

### Layer 8 — Bridge Relayer Latency

The bridge relayer's speed determines how quickly users receive their tokens after initiating a bridge transaction. Minimize every delay.

#### Watch events via WebSocket, not polling

```go
// BAD: poll BSC every 3 seconds — adds up to 3 seconds of latency per event
ticker := time.NewTicker(3 * time.Second)
for range ticker.C {
    logs, _ := ethClient.FilterLogs(ctx, query)
    for _, log := range logs { process(log) }
}

// GOOD: subscribe to events via WebSocket — near-zero latency
wsClient, _ := ethclient.Dial("wss://bsc-ws-node.example.com")
sub, _ := wsClient.SubscribeFilterLogs(ctx, query, logsCh)

go func() {
    for {
        select {
        case err := <-sub.Err():
            // Reconnect on error
            wsClient, sub = reconnect()
        case log := <-logsCh:
            go process(log)  // process in goroutine; don't block subscription
        }
    }
}()
// WebSocket events arrive within milliseconds of BSC block production
```

#### Skip unnecessary confirmation waits for small amounts

```go
// For large bridge amounts: wait full 15 BSC confirmations (safety)
// For small amounts (< $100 equivalent): submit after 3 confirmations
// This is a governance parameter: bridge.FastTrackThresholdUSD

func confirmationDepth(amount *big.Int, priceUSD float64) uint64 {
    valueUSD := new(big.Float).Mul(
        new(big.Float).SetInt(amount),
        big.NewFloat(priceUSD),
    )
    if valueUSD.Cmp(big.NewFloat(fastTrackThreshold)) < 0 {
        return 3   // fast track for small amounts
    }
    return 15      // full confirmation for large amounts
}
```

#### Transaction submission: use eth_sendRawTransaction with pre-signed tx

```go
// Pre-sign BSC release transactions using HSM as soon as confirmation depth reached
// Submit immediately — do not wait for chain gRPC round trip first

// Pipeline:
// 1. Confirmation depth reached
// 2. Sign release tx via HSM (async — does not block event processing)
// 3. Submit signed tx to BSC (fire and forget)
// 4. Submit MsgBridgeConfirm to chain (record that release was sent)
// 5. Monitor BSC tx for confirmation
// Total latency target: < 500ms from confirmation depth reached to BSC tx submitted
```

---

### Layer 9 — Oracle Feed Latency

Oracle price latency affects `x/settlement` accuracy. Lower latency = prices closer to real-time.

#### WebSocket price feeds instead of REST polling

```go
// BAD: REST poll every 3 seconds — price is always 0–3 seconds stale
ticker := time.NewTicker(3 * time.Second)
for range ticker.C {
    price, _ = fetchBinanceREST()
}

// GOOD: WebSocket stream — price updates within milliseconds of market move
conn, _, _ := websocket.DefaultDialer.Dial(
    "wss://stream.binance.com:9443/ws/bnbusdt@ticker", nil,
)
for {
    _, msg, err := conn.ReadMessage()
    if err != nil { conn = reconnect(); continue }
    price = parsePrice(msg)
    lastUpdated = time.Now()
}
```

#### Commit on first available price per block, not at block end

```go
// Watch for new block events and commit immediately when new block starts
// This gives maximum time for the commit tx to propagate before block closes

blockCh := subscribeNewBlocks(chainClient)
for block := range blockCh {
    price := latestPrice.Load()   // atomic read of WebSocket-updated price
    salt  := generateSalt()
    commitment := sha256.Sum256(append([]byte(price), salt...))

    // Submit commit tx immediately — do not buffer
    go submitCommit(block.Height, commitment)
    storePendingReveal(block.Height, price, salt)
}
```

---

### Layer 10 — Network and Infrastructure

#### Validator node: tune kernel networking

```bash
# /etc/sysctl.d/99-chain.conf — apply on all validator and full nodes
# TCP buffer sizes
net.core.rmem_max           = 134217728    # 128MB receive buffer
net.core.wmem_max           = 134217728    # 128MB send buffer
net.ipv4.tcp_rmem           = 4096 87380 134217728
net.ipv4.tcp_wmem           = 4096 65536 134217728

# Connection handling
net.core.somaxconn          = 65535        # max listen backlog
net.core.netdev_max_backlog = 65535
net.ipv4.tcp_max_syn_backlog = 65535

# TIME_WAIT handling — important for high-connection-rate API servers
net.ipv4.tcp_tw_reuse       = 1
net.ipv4.tcp_fin_timeout    = 15

# Apply immediately (and at boot via sysctl.d):
sysctl --system
```

#### Envoy: connection pooling and circuit breaking

```yaml
# envoy.yaml — upstream cluster config for API server
clusters:
  - name: api_server
    type: STRICT_DNS
    connect_timeout: 0.25s
    lb_policy: LEAST_REQUEST        # route to least-loaded instance
    circuit_breakers:
      thresholds:
        - priority: DEFAULT
          max_connections:       1000
          max_pending_requests:  1000
          max_requests:          5000
          max_retries:           10
    upstream_connection_options:
      tcp_keepalive:
        keepalive_time: 300
    http2_protocol_options: {}      # enable HTTP/2 upstream
    load_assignment:
      cluster_name: api_server
      endpoints:
        - lb_endpoints:
            - endpoint:
                address:
                  socket_address: { address: api-server, port_value: 3000 }
```

#### Separate network interfaces for validator P2P and signing

```bash
# On validator node: use separate NICs for different traffic types
# eth0 — sentry P2P traffic (CometBFT port 26656)
# eth1 — Horcrux signing traffic (port 1234)
# eth2 — monitoring and management (Prometheus port 26660)

# This prevents signing traffic from competing with P2P traffic under high load
# High P2P traffic during catch-up sync can delay signing → missed blocks
# Separation eliminates this interference

# In config.toml: bind priv_validator_laddr to eth1 IP specifically
[priv_validator]
laddr = "tcp://<eth1-ip>:1234"
```

---

### Layer 11 — Profiling and Benchmarking

No optimization is worth making without measuring. These are the tools and targets.

#### Chain-level profiling

```bash
# Enable pprof on all nodes (non-public endpoint — firewall to internal only)
# In config.toml:
[rpc]
pprof_laddr = "localhost:6060"

# CPU profile during high load:
go tool pprof http://localhost:6060/debug/pprof/profile?seconds=30
# Look for: IAVL tree operations, ante handler, custom module EndBlock hooks

# Memory profile:
go tool pprof http://localhost:6060/debug/pprof/heap
# Look for: unbounded caches, large byte slice allocations in message handlers

# Goroutine leak check:
curl http://localhost:6060/debug/pprof/goroutine?debug=2 | grep -A5 "goroutine "
# Goroutine count should be stable under sustained load; growing count = leak
```

#### PostgreSQL query performance

```sql
-- Enable pg_stat_statements to track every query
CREATE EXTENSION IF NOT EXISTS pg_stat_statements;

-- Find the slowest queries (run weekly):
SELECT
    query,
    calls,
    mean_exec_time,
    total_exec_time,
    rows / calls AS avg_rows
FROM pg_stat_statements
ORDER BY mean_exec_time DESC
LIMIT 20;

-- Find queries doing sequential scans on large tables:
SELECT relname, seq_scan, seq_tup_read, idx_scan
FROM pg_stat_user_tables
WHERE seq_scan > 0
ORDER BY seq_tup_read DESC;
-- Any table with high seq_tup_read needs a targeted index

-- Reset stats after adding indexes to measure improvement:
SELECT pg_stat_reset();
SELECT pg_stat_statements_reset();
```

#### API load testing

```bash
# Use k6 for load testing the API server
# Target: p99 latency < 200ms at 1000 concurrent users

# k6 script: scripts/load-test.js
import http from 'k6/http';
import { check, sleep } from 'k6';

export let options = {
    stages: [
        { duration: '2m', target: 100  },   // ramp up
        { duration: '5m', target: 1000 },   // sustained load
        { duration: '2m', target: 0    },   // ramp down
    ],
    thresholds: {
        http_req_duration: ['p(99)<200'],    // 99th percentile < 200ms
        http_req_failed:   ['rate<0.01'],    // < 1% error rate
    },
};

export default function() {
    const r = http.get(`${__ENV.API_URL}/api/v1/accounts/${randomAddress()}/balance`);
    check(r, { 'status 200': (r) => r.status === 200 });
    sleep(0.1);
}

# Run:
k6 run --env API_URL=https://testnet.example.com scripts/load-test.js
```

#### Performance targets (mainnet)

| Metric | Target | Measured at |
|---|---|---|
| Block time | ≤ 3.5 seconds | 7+ validators, 3 continents |
| Transactions per second | ≥ 500 TPS | At 40M gas/block, 80K gas avg tx |
| Oracle commit-to-reveal latency | ≤ 1 block | WebSocket feed, HSM signing |
| Bridge BSC→chain latency | ≤ 100 seconds | 15 BSC confirmations × 3s + relay |
| Bridge chain→BSC latency | ≤ 30 seconds | 1 chain confirmation + relay |
| API p50 response time | ≤ 20ms | Cached endpoints |
| API p99 response time | ≤ 200ms | All endpoints, 1000 concurrent users |
| NATS ingestion lag | ≤ 1 block behind chain | Projection service batch consumer |
| PostgreSQL Read DB p99 query | ≤ 10ms | With materialized views + indexes |

---

### Performance Optimization Priority Order

Apply in this order. Each layer unblocks the next.

```
1. CometBFT timeouts + PebbleDB           → reduces block time (everything depends on this)
2. IAVL pruning + mempool priority         → increases TPS ceiling
3. NATS async publish + batch consumer     → eliminates ingestion lag
4. PostgreSQL COPY + partitioning          → handles sustained write volume
5. Materialized views + partial indexes    → makes Read DB queries fast
6. PgBouncer transaction pooling           → allows 10,000 concurrent API clients
7. Redis caching in API layer              → eliminates repeat DB queries
8. HTTP/2 + SSE backpressure               → reduces client connection overhead
9. CosmWasm optimizer + batch state reads  → reduces per-tx gas cost
10. WebSocket price feeds + oracle pipelining → minimizes oracle latency
11. Bridge WebSocket events + pre-signing  → minimizes bridge latency
12. Kernel network tuning + Envoy config   → removes infrastructure bottlenecks
```

Do not skip ahead. Optimizing the API before fixing the chain is like widening a highway on-ramp when the highway itself is a single lane.

---

## Network Upgrade Procedure

Every software upgrade follows this process on all three networks. The steps are identical in structure — only the timelines and coordination requirements differ.

---

### Upgrade Architecture — How `x/upgrade` Works

```
Governance proposal submitted (MsgSoftwareUpgrade)
         ↓
Voting period passes with quorum + threshold
         ↓
Proposal executes → upgrade plan written to x/upgrade KV store
  (plan contains: name, height, info URL)
         ↓
Chain runs normally until upgrade_height - 1
         ↓
At upgrade_height: chain binary calls upgrade handler by name
  • If handler exists → executes migrations → chain continues
  • If handler missing → chain panics and halts permanently
         ↓
Validators swap binary before upgrade_height
Chain resumes at upgrade_height with new binary
```

**Non-negotiable rule:** every upgrade proposal must name a handler that exists in the new binary. If there are no state migrations, the handler is a documented no-op — but it must be present. A chain that halts waiting for a handler that does not exist requires a coordinated emergency recovery across all validators.

---

### Upgrade Handler Template

Every upgrade in `/scripts/upgrades/<version>/handler.go`:

```go
// v1.1.0 upgrade handler
// State migrations: none (no-op handler — binary swap only)
// Rationale: this upgrade only updates the relayer binary and bridge parameters;
//            no on-chain state schema changes required.
func CreateUpgradeHandler(
    mm *module.Manager,
    configurator module.Configurator,
) upgradetypes.UpgradeHandler {
    return func(ctx sdk.Context, plan upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
        logger := ctx.Logger().With("upgrade", plan.Name)
        logger.Info("running upgrade handler", "name", plan.Name, "height", ctx.BlockHeight())

        // Run any module migrations here
        // Example: return mm.RunMigrations(ctx, configurator, vm)

        // No-op: return existing version map unchanged
        return vm, nil
    }
}
```

Registered in `app.go`:
```go
app.UpgradeKeeper.SetUpgradeHandler("v1.1.0", upgrades.CreateUpgradeHandler(app.ModuleManager, app.Configurator()))
```

**File naming convention:** `/scripts/upgrades/v<major>.<minor>.<patch>/handler.go`
Handler name in proposal must exactly match the string registered in `SetUpgradeHandler`.

---

### Devnet Upgrade Procedure

**Purpose:** validate that the upgrade handler compiles, runs without panic, and produces correct state.
**Timeline:** no governance required — restart at any time.

```
Step 1  Write upgrade handler in /scripts/upgrades/<version>/handler.go
Step 2  Register handler in app.go SetUpgradeHandler
Step 3  Build new binary: make build
Step 4  Submit upgrade proposal via CLI (voting_period = 2 minutes on devnet):
          chaind tx gov submit-proposal software-upgrade v1.x.x \
            --upgrade-height <current_height + 20> \
            --title "v1.x.x upgrade" \
            --deposit 1utoken
Step 5  Vote YES immediately (single validator, quorum = 1%):
          chaind tx gov vote <proposal-id> yes
Step 6  Wait for upgrade_height (takes ~40s at devnet block time)
Step 7  Swap binary: kill process → replace binary → restart node
Step 8  Verify: chain resumes at correct height; no panic; handler log visible
Step 9  Verify: any state migrations ran correctly (query affected state)
Step 10 If panic: stop node; fix handler; rebuild; repeat from Step 3
```

**Pass criteria:** chain resumes without panic; upgrade handler log emitted; state queries return expected values.

---

### Testnet Upgrade Procedure

**Purpose:** full dress rehearsal of the mainnet upgrade procedure with real external validators.
**Timeline:** 48-hour notice minimum to external validators.

```
Step 1  Upgrade branch merged and binary built (reproducible build verified)
Step 2  Publish release notes: binary hash, upgrade height formula, migration notes
        → post in validator Discord / Telegram ≥ 48 hours before proposal
Step 3  Deploy new binary to internal full nodes first; verify it syncs correctly
Step 4  Submit governance proposal (voting_period = 1 hour on testnet):
          upgrade-height = current_height + (blocks_per_hour × 24)
          This gives validators 24 hours from proposal pass to binary swap
Step 5  External validators vote; quorum = 10%; threshold = 50%
Step 6  Proposal passes → upgrade plan written to chain
Step 7  Validators prepare: download new binary; verify SHA256 against release notes
Step 8  At upgrade_height − 50 blocks: validators confirm binary is staged
Step 9  Chain halts at upgrade_height
Step 10 All validators swap binary and restart within 10-minute window
Step 11 Chain resumes when ≥ 2/3 validators are on new binary
Step 12 Verify: block production resumes; upgrade handler log visible on all nodes;
               state migrations correct; all backend services reconnect
Step 13 Monitor for 30 minutes post-upgrade: no unexpected panics or state errors
```

**Pass criteria:** chain resumes < 10 minutes after halt; all validators upgraded; no state corruption.
**Testnet upgrade is mandatory before any mainnet upgrade — no exceptions.**

---

### Mainnet Upgrade Procedure

**Purpose:** production software upgrade with zero tolerance for extended chain halt.
**Timeline:** minimum **2 weeks** public notice before upgrade height.

#### Pre-Upgrade Preparation (Weeks Before)

```
Week -3  Finalize upgrade branch; all tests passing; reproducible build verified
          on two independent machines (identical SHA256)
Week -2  MANDATORY: run complete upgrade on testnet (see Testnet procedure above)
          Testnet upgrade must PASS before mainnet proposal is submitted
Week -2  Publish upgrade announcement:
          • new binary SHA256 (multiple formats: sha256, sha512, cosign signature)
          • upgrade height (calculated from current block production rate)
          • upgrade time (estimated, based on block height)
          • migration notes (what state changes, what parameters change)
          • rollback procedure (in case of failure)
          • validator action required (yes/no; what they need to do)
Week -2  Submit governance proposal:
          upgrade-height chosen such that estimated upgrade time is:
            • weekday (not Friday — avoid weekend recovery scenarios)
            • business hours UTC (09:00–15:00 UTC)
            • ≥ 14 days from proposal submission
Week -1  Reminder announcement to validators (7 days out)
Day -1   Final reminder (24 hours out); validators confirm binary staged
```

#### Upgrade Day — Step by Step

```
T - 2h   All internal team on-call; monitoring dashboards open
T - 1h   Alert: upgrade in 1 hour — validators confirm readiness in coordination channel
T - 30m  Take pre-upgrade snapshot of all PostgreSQL databases (Write DB, Read DB, Relayer DB)
          → verify snapshot completes successfully before proceeding
T - 10m  Alert: 10 minutes to upgrade height; all validators confirm binary staged
T - 0    Chain halts at upgrade_height (block production stops)
          → this is expected; it is not a chain failure
T + 0m   All validators swap binary and restart simultaneously
T + 5m   Chain resumes when ≥ 2/3 validators are on new binary
          → target: < 5 minutes of halt
T + 10m  Verify:
          • Block production resumed (check explorer + monitoring)
          • Upgrade handler log emitted on all internal nodes
          • All state migrations ran correctly (query affected KV paths)
          • module/ingestion reconnected; Write DB receiving new blocks
          • module/projection consuming new blocks; Read DB up to date
          • NATS publishing resumed; nats_published=false backlog = 0
          • Bridge relayers reconnected and processing events
          • Oracle aggregator reconnected and submitting rounds
T + 30m  All-clear: post in validator channel; announce to community
T + 24h  Post-upgrade monitoring complete; heightened alert window ends
```

**Halt duration target: < 10 minutes.** If halt exceeds 20 minutes without chain resuming, begin rollback assessment.

#### Upgrade Height Calculation

```go
// How to calculate upgrade height
currentHeight := getLatestBlockHeight()         // query from chain
targetUpgradeTime := time.Now().Add(14 * 24 * time.Hour)  // 14 days from now
                   // adjust to nearest weekday 12:00 UTC
avgBlockTime := 3.5 * time.Second               // measure from last 10,000 blocks (use actual chain metric, not this constant)
blocksUntilUpgrade := targetUpgradeTime.Sub(time.Now()) / avgBlockTime
upgradeHeight := currentHeight + blocksUntilUpgrade

// Add 5% buffer for block time variance
upgradeHeight = upgradeHeight * 1.05
```

Re-verify estimated time 24 hours before upgrade (block production rate may drift). If estimated time is off by > 30 minutes, submit a revised proposal or announce the adjusted estimate.

---

### Rollback Procedure (Upgrade Failed)

**Trigger:** chain halts at upgrade_height and does NOT resume within 20 minutes.

Two failure modes require different responses:

#### Mode A — Handler Panic (chain halted, no state written)

The upgrade handler panicked before completing. State is unchanged (CometBFT ABCI contract: if `BeginBlock` returns error, block is not committed).

```
Step 1  Confirm: chain is at upgrade_height and not producing blocks
Step 2  Diagnose: read panic log on internal nodes to identify the handler error
Step 3  Decision: can fix be compiled and deployed within 2 hours?
        YES → patch handler; rebuild; distribute new binary to all validators
              announce delay in coordination channel
              validators restart with patched binary
              chain resumes
        NO  → proceed to Mode A rollback
Step 4  Rollback (if fix is not feasible quickly):
        a. All validators revert to previous binary
        b. Submit emergency governance proposal to cancel the upgrade plan:
             chaind tx gov submit-proposal cancel-software-upgrade
        c. Chain resumes with old binary at upgrade_height + 1
        d. Plan next upgrade attempt with fixed handler
```

#### Mode B — State Corruption (chain resumed but state is wrong)

The upgrade handler ran but produced incorrect state. Chain is producing blocks but data is inconsistent.

```
Step 1  Halt chain: coordinate emergency stop via validator coordination channel
        (all validators stop binary simultaneously)
Step 2  Restore all PostgreSQL databases from pre-upgrade snapshot (taken at T-30m)
Step 3  Restore chain data directory from pre-upgrade archive node snapshot
Step 4  All validators revert to previous binary
Step 5  Restart chain at pre-upgrade height with old binary
Step 6  Verify: chain resumes at correct pre-upgrade height; state correct
Step 7  Announce to community: upgrade rolled back; reason; next steps
Step 8  Post-mortem within 48 hours; fix handler; re-run full testnet upgrade
```

**Target rollback RTO: < 4 hours** (limited by PostgreSQL restore time from snapshot).

---

### Upgrade Communication Templates

#### Announcement (2 weeks before)

```
[UPGRADE NOTICE] v1.x.x — <Chain Name> Mainnet Upgrade

Upgrade height: <height>
Estimated time: <date> <time> UTC (±30 minutes depending on block time)
Voting period ends: <date>

What is changing:
• <brief description of changes>
• State migrations: <yes/no — describe if yes>
• Parameter changes: <list if any>

Validator action required:
• Download new binary before upgrade height
• Binary: <download URL>
• SHA256: <hash>
• cosign verify: <command>

No action required from token holders or dApp users.

Full release notes: <URL>
Rollback procedure: <URL>
```

#### All-Clear (after successful upgrade)

```
[UPGRADE COMPLETE] v1.x.x — <Chain Name> Mainnet Upgrade

Chain resumed at height: <height>
Halt duration: <N> minutes
All validators upgraded: ✅
State migrations: ✅
Bridge and oracle operational: ✅

Thank you to all validators for coordinating the upgrade.
```

---

### Upgrade Checklist

**Before submitting proposal:**
- [ ] Upgrade handler written and registered in `app.go`
- [ ] Handler name in proposal exactly matches `SetUpgradeHandler` string
- [ ] Reproducible build verified on two independent machines (identical SHA256)
- [ ] cosign signature published
- [ ] All tests passing on upgrade branch
- [ ] **Testnet upgrade completed successfully** (mandatory)
- [ ] Pre-upgrade snapshot procedure tested on testnet
- [ ] Rollback procedure tested on testnet
- [ ] Upgrade height is a weekday, 09:00–15:00 UTC, ≥ 14 days from proposal
- [ ] Announcement published in all validator channels

**On upgrade day:**
- [ ] Pre-upgrade PostgreSQL snapshot taken and verified (T-30m)
- [ ] All validators confirmed binary staged (T-10m)
- [ ] Internal team on-call and monitoring dashboards open
- [ ] Rollback decision tree agreed upon (at what point do we roll back?)

**After upgrade:**
- [ ] Block production resumed < 10 minutes
- [ ] Upgrade handler log visible
- [ ] State migrations verified
- [ ] ingestion/projection/relayer/oracle all reconnected
- [ ] All-clear announced to community
- [ ] Post-upgrade monitoring for 24 hours

---

## System Backup & Disaster Recovery

### Data Inventory, RPO & RTO Targets

| System | What Is Stored | RPO | RTO | Priority |
|---|---|---|---|---|
| Write DB (PostgreSQL) | Append-only event log — permanent CQRS source of truth | 0 s (synchronous standby) | < 4 h | **P0** |
| Cosmos chain state | IAVL KV store, block store, app state | 0 blocks (replicated across validators) | < 2 h (state sync) | **P0** |
| Horcrux key shards | Ed25519 validator private key material (split 3-of-3) | N/A (static; never rotated except ceremony) | < 24 h | **P0** |
| HashiCorp Vault | NKeys, PKCS#11 credentials, DB passwords, TLS certs | 0 (Vault Raft HA cluster) | < 1 h | **P0** |
| Relayer DB (PostgreSQL) | Nonce bitmap, confirmation state, vote records | 0 s (synchronous standby) | < 2 h | **P1** |
| NATS JetStream cluster | Event streams (bridge events 365-day retention) | 0 (R=3 cluster, sync writes) | < 1 h | **P1** |
| Read DB (PostgreSQL) | Denormalized projections (fully rebuildable from Write DB) | N/A (rebuildable) | < 6 h (rebuild) | **P1** |
| Vault configuration | Policies, auth mounts, secret engines | Vault Raft snapshot | < 1 h | **P1** |
| k8s cluster state | Deployments, ConfigMaps, PVCs, Secrets, RBAC | Velero hourly + etcd backup | < 30 m | **P2** |
| Chain archive node | Full block history from genesis (no pruning) | Nightly snapshot | < 24 h (full sync) | **P2** |
| NATS stream config | Stream definitions, account NKeys | git + Vault | < 30 m | **P2** |
| BSC LockBox state | On BSC network — not operator-owned | N/A (BSC is source of truth) | N/A | — |
| CosmWasm contract state | Part of Cosmos chain IAVL state | Same as chain state | Same as chain state | **P0** |

---

### PostgreSQL Backup Strategy (Write DB, Read DB, Relayer DB)

#### Continuous WAL Archiving (primary mechanism)

```
PostgreSQL primary
  → WAL segment completed (every 16 MB or max_wal_size / wal_keep_size)
  → archive_command copies to S3/GCS immediately
  → S3/GCS: server-side AES-256 encryption
  → S3 Cross-Region Replication (CRR) / GCS Object Replication → Region B bucket
  → Barman or pgBackRest manages archive and catalog
```

- WAL archive retention: **90 days** (PITR window)
- Full base backup via `pg_basebackup` or pgBackRest: **nightly at 02:00 UTC**
- No recovery gap: between any nightly base backup and any WAL segment = complete history of every transaction

#### Backup Retention Schedule

| Backup Type | Retention | Storage Tier |
|---|---|---|
| Daily base backup | 30 days | S3 Standard / GCS Standard |
| Weekly snapshot | 90 days | S3 Standard-IA / GCS Nearline |
| Monthly snapshot | 2 years | S3 Glacier / GCS Coldline |
| WAL archive segments | 90 days | S3 Standard / GCS Standard |
| Pre-upgrade snapshot | Forever | S3 Glacier / GCS Archive |

All backups encrypted with AES-256 (S3 SSE-S3 or customer-managed KMS key). Backup encryption key stored in Vault (separate from database credentials).

#### Write DB — Special Considerations

The Write DB is the **permanent source of truth** for the entire CQRS pipeline. Two independent recovery paths exist:

1. **Fast path (< 4 h):** restore from S3/GCS backup + WAL replay to latest point → restart ingestion → startup reconciliation fills any remaining gap via `BlockResults` RPC from archive node
2. **Full re-index path (hours to days):** start fresh Write DB → run ingestion from block 0 against archive node → entire event log rebuilt from chain. Used only if backup is corrupted AND archive node is available. Time scales linearly with chain age.

**Pre-upgrade snapshot:** before every chain binary upgrade, take a manual base backup of all three databases. Stored permanently. Allows point-in-time rollback to pre-upgrade state if the upgrade causes data corruption.

#### Read DB — Recovery Is Rebuild

Read DB contains only denormalized projections — no unique data. On complete loss:
1. Start fresh Read DB from migration baseline
2. Restart `module/projection` with `last_committed_height = 0`
3. Projection replays all Write DB events via `projection_reader` user
4. `module/api` returns partial/degraded data during catch-up (document in API as expected during DR window)
5. Catch-up rate depends on projection throughput and Write DB size

Read DB PVC is still backed up by Velero (hourly) as a fast-restart option to avoid full rebuild after minor failures.

#### PostgreSQL Streaming Replication Topology

```
Region A (primary)                    Region B (synchronous standby)
┌─────────────────┐                   ┌──────────────────────────────┐
│ Write DB primary│ ─── WAL stream ──▶│ Write DB standby             │
│                 │  synchronous_commit│ (hot standby, read-only)     │
│ Patroni leader  │       = on         │ Patroni replica              │
└─────────────────┘                   └──────────────────────────────┘
         │                                         │
         └──── both archive WAL ──────────────────▶ S3/GCS (cross-region)
```

`synchronous_commit = on` guarantees zero data loss on primary failure. Patroni promotes standby in < 30 s automatically. DNS-based k8s Service updates connection string transparently.

---

### Cosmos Chain State Backup

#### State Sync Snapshots

- All full nodes serve state sync snapshots: interval every **1,000 blocks**
- Snapshot format: IAVL tree export + block metadata
- New validator or full node syncs from snapshot in minutes (not hours from genesis)
- Snapshot chunks cached in memory and served via P2P; no separate storage backend required

#### Archive Node

- **Minimum one dedicated archive node** (pruning = `nothing`) running at all times
- Archive node serves: `BlockResults`, `TxSearch`, `ABCIQuery` for all historical heights
- Ingestion startup reconciliation uses archive node exclusively for gap-filling
- **Archive node data backup:** nightly compressed snapshot of `$CHAIN_HOME/data/` to S3/GCS
  - Compression: `zstd` (fast, good ratio for IAVL data)
  - Retention: 90 days rolling
  - Cross-region replication active
- Archive node restore: download snapshot → decompress → start full node (no state sync needed — full history already present)

#### Genesis File and Upgrade Handlers

Both are **version-controlled in git** and recoverable at any time:
- `genesis.json`: stored in `/chain/config/genesis.json`; tagged at every testnet and mainnet launch
- Upgrade handlers: `/scripts/upgrades/<version>/handler.go`; present for every past upgrade

#### Validator Key Material (Horcrux Shards — Most Critical)

```
Shard 1 → Signer machine 1  +  Air-gapped USB (encrypted, AES-256)
Shard 2 → Signer machine 2  +  Air-gapped USB (encrypted, AES-256)
Shard 3 → Signer machine 3  +  Air-gapped USB (encrypted, AES-256)
```

- USB backups stored in physically separate locations (not the same datacenter as signer machines)
- USB encryption password stored in hardware password manager (air-gapped, not in Vault or cloud)
- **Never store Horcrux shards in S3, GCS, or any cloud storage** — air-gapped only
- Shard backup verified: annual drill — restore each shard from USB onto a test machine; verify 2-of-3 signing produces valid block signatures; verify no double-sign event during test

---

### NATS JetStream Backup

#### Built-in Redundancy (Primary Protection)

NATS 3-node R=3 cluster: every message written to all 3 nodes before ACK. Single-node or two-node failures are self-healing — no backup restore required.

#### Stream Configuration Backup

Stream definitions are declarative configuration stored in git (`/nats/streams/*.json`). Restore procedure on full cluster loss:
1. Provision 3 new NATS nodes from k8s StatefulSet manifest
2. Apply stream configuration: `nats stream add --config /nats/streams/*.json`
3. Apply account NKey credentials from Vault
4. Ingestion reconnects → back-fills `nats_published=false` rows from Write DB in order
5. All consumers reconnect → resume from last committed offset (stored in durable consumer state on NATS; if lost, rebuild from Write DB)

#### Bridge Stream — Special Retention

`bridge.bsc.*` stream: 365-day retention, no size limit. This is the most critical NATS stream. If all 3 NATS nodes are simultaneously and catastrophically destroyed (entire cluster hardware loss):
- Re-populate bridge events from Write DB: `SELECT * FROM events WHERE event_type LIKE 'bridge.%' ORDER BY block_height`
- One-time migration script: `/scripts/nats-repopulate-bridge.go`
- Tested in devnet before mainnet launch

---

### HashiCorp Vault Backup

#### Vault HA Architecture

```
Vault node 1 (active)  ─┐
Vault node 2 (standby) ─┼── Integrated Raft storage (R=3)
Vault node 3 (standby) ─┘
         │
         └── Automated Raft snapshot → S3/GCS (encrypted, hourly)
```

- Vault Raft snapshots: **hourly** via Vault snapshot agent; 30-day retention in S3/GCS
- Snapshot encryption: Vault encrypts snapshot contents before writing to S3/GCS
- Unseal keys: Shamir secret sharing **5-of-7**; same key holders as cold multi-sig (2 founding team, 2 security council, 2 lead validators, 1 legal trustee); stored air-gapped (not in cloud)
- Root token: generated once at Vault init; stored in air-gapped hardware security device; used only for break-glass emergencies

#### Vault Restore Procedure

1. Provision 3 new Vault nodes
2. `vault operator raft snapshot restore <snapshot_file>` on leader
3. Gather 5 unseal key holders; unseal all 3 nodes
4. Verify all secret paths accessible: NKeys, DB credentials, PKCS#11 slots, TLS certificates
5. Restart all services that depend on Vault agent injection
6. RTO target: **< 1 hour** (gated on assembling 5 unseal holders — contact list drilled quarterly)

---

### Kubernetes Cluster Backup

#### etcd Backup (Control Plane State)

- etcd snapshot: every **6 hours** via `etcdctl snapshot save`
- Stored in S3/GCS; 7-day retention; cross-region replication
- Restore: `etcdctl snapshot restore` → restart control plane components
- Tested semi-annually

#### Velero Application Backup

- Velero installed in both chain and backend clusters
- Schedule: **hourly** (namespace-scoped); 7-day retention
- Backs up: Deployments, StatefulSets, ConfigMaps, Secrets, PVCs (via volume snapshots), RBAC
- **Important:** PostgreSQL PVCs are backed up at the PostgreSQL WAL level (more granular). Velero PVC snapshots are a supplementary fast-restart mechanism, not the primary recovery path.
- Cert-manager: TLS certificates are recreated automatically by cert-manager on cluster restore; no separate backup needed

#### Infrastructure as Code

All k8s manifests, Terraform configs, Helm values, and Envoy configs are in git. A complete cluster can be reproduced from scratch using:
```bash
terraform apply     # provision cloud resources
helm install        # deploy all services
kubectl apply -f /infra/k8s/
```
This is tested as part of the devnet `make devnet-up` procedure.

---

### Disaster Recovery Runbooks

#### DR Scenario 1 — Write DB Primary Failure (Patroni Auto-Failover)

**Trigger:** primary PostgreSQL pod crashes or becomes unreachable  
**Expected behavior:** fully automatic, no human action required for initial recovery

1. Patroni detects primary unavailable (health check timeout: 30 s)
2. Patroni elects Region B standby as new primary (< 30 s total)
3. k8s Service DNS updates to point to new primary
4. `module/ingestion` connection pool reconnects to new primary automatically
5. Advisory lock re-acquired by ingestion (previous session ended → lock released automatically)
6. **Data loss:** zero (`synchronous_commit = on`)
7. **Manual action required:** provision new standby in Region A; join to Patroni cluster; verify replication lag returns to 0

**Alert:** `pg_patroni_last_leader_change` metric fires PagerDuty if leader changes

---

#### DR Scenario 2 — Write DB Catastrophic Loss (Both Primary and Standby)

**Trigger:** entire Region A + Region B PostgreSQL destroyed  
**RTO target: < 4 hours**

```
Step 1  Provision new PostgreSQL instance (Region A)                         ~10 min
Step 2  Download latest base backup from S3/GCS                              ~20 min
Step 3  Apply WAL archives to latest available point (pgBackRest restore)    ~30 min
Step 4  Start PostgreSQL; verify data integrity                              ~10 min
Step 5  Update k8s Secret `write-db-primary-url` to new instance            ~5 min
Step 6  Restart module/ingestion (startup reconciliation fills gap)          ~10 min
Step 7  Verify Write DB MAX(block_height) matches chain height - N           ~10 min
Step 8  Provision Region B standby; configure streaming replication          ~30 min
Total                                                                        ~2 h
```

Remaining gap (backup time → recovery time) is filled by ingestion startup reconciliation from archive node. No permanent data loss.

---

#### DR Scenario 3 — Full NATS Cluster Loss

**Trigger:** all 3 NATS nodes destroyed simultaneously  
**RTO target: < 1 hour**

```
Step 1  Deploy 3 new NATS StatefulSet pods from k8s manifest                ~5 min
Step 2  Apply stream configuration from git                                  ~2 min
Step 3  Restore account NKey credentials from Vault                          ~5 min
Step 4  Ingestion detects NATS reconnection; queries nats_published=false   ~1 min
Step 5  Back-fill from Write DB in block_height order; marks published       ~varies
Step 6  Projection consumers reconnect; resume from Write DB offset          ~5 min
Step 7  Verify Read DB is consistent (compare with Write DB event count)     ~10 min
Total                                                                        ~30 min + back-fill time
```

Bridge event stream: if 365-day bridge data is lost, run `/scripts/nats-repopulate-bridge.go` to reload from Write DB. This is a one-time operation and is non-blocking for chain operation.

---

#### DR Scenario 4 — Vault Complete Loss

**Trigger:** all 3 Vault nodes destroyed  
**RTO target: < 1 hour** (gated on assembling 5 unseal holders)

```
Step 1  Provision 3 new Vault nodes from k8s manifest                       ~10 min
Step 2  Restore latest Vault Raft snapshot from S3/GCS                      ~5 min
Step 3  Contact unseal key holders (5 of 7 required); coordinate online     ~30 min
Step 4  Unseal all 3 Vault nodes (5-of-7 Shamir key shares entered)        ~10 min
Step 5  Verify all secret paths (NKeys, DB creds, PKCS#11, TLS)            ~10 min
Step 6  Restart Vault-agent-injected pods (all services)                     ~5 min
Total                                                                        ~1 h
```

**Quarterly drill:** assemble 5 key holders online (test the coordination procedure), perform unseal on a test Vault instance restored from latest snapshot, verify all credentials are readable. Log drill results.

---

#### DR Scenario 5 — Read DB Complete Loss

**Trigger:** Read DB destroyed  
**RTO target: < 6 hours** (degraded service during rebuild, not outage)

```
Step 1  Start fresh Read DB from migration baseline (empty schema)           ~5 min
Step 2  Restart module/projection with last_committed_height = 0            ~2 min
Step 3  Projection replays all Write DB events in order                     ~hours (scales with chain age)
Step 4  API returns partial data during catch-up (degraded mode, not down)  ongoing
Step 5  Monitor projection lag metric until lag = 0                         ~varies
Step 6  Verify API query results match expected values                      ~30 min
```

**Optimization:** if Velero PVC snapshot is recent (< 1 hour old), restore from Velero snapshot instead of full rebuild. Projection resumes from the snapshot's committed offset rather than from 0.

---

#### DR Scenario 6 — Horcrux Signer Machine Loss (1 of 3)

**Trigger:** one signer machine destroyed  
**RTO target: < 24 hours** (chain continues signing with 2 remaining machines — no halt)

```
Step 1  Verify chain continues (2-of-3 threshold still met; no halt)        immediate
Step 2  Alert: signer machine offline                                        ~1 min (auto)
Step 3  Provision replacement signer machine in isolated network ring        ~2 h
Step 4  Retrieve encrypted key shard from air-gapped USB (physical access)  ~2 h
Step 5  Restore shard: decrypt with hardware password manager               ~30 min
Step 6  Configure Horcrux on replacement machine; connect to cluster        ~1 h
Step 7  Verify 3-of-3 signing resumes; confirm no double-sign event         ~30 min
Total                                                                        ~6 h
```

---

#### DR Scenario 7 — Full Region A Datacenter Loss

**Trigger:** entire Region A unavailable (cloud zone/region outage)  
**RTO target: < 2 hours** (chain never halts if validators are distributed)

| Component | Region A | Region B | Action |
|---|---|---|---|
| Chain validators | Subset of validators | Remaining validators | Chain continues if ≥ 2/3 of voting power in Region B |
| PostgreSQL Write DB | Primary (down) | Standby (Patroni auto-promotes) | Auto-failover < 30 s |
| PostgreSQL Read DB | Primary (down) | Standby (Patroni auto-promotes) | Auto-failover < 30 s |
| NATS | Nodes (down) | Nodes | If NATS nodes are spread across regions: R=3 cluster continues with remaining nodes |
| Backend cluster | Down | k8s workloads reschedule to Region B | Pod rescheduling < 5 min (if multi-region cluster) |
| Vault | Node(s) (down) | Node(s) | Raft cluster continues with remaining nodes |
| Ingestion | Down | Rescheduled to Region B | Reconnects to Write DB (now in Region B); startup reconciliation |

**Critical prerequisite:** validators must be distributed across ≥ 2 independent regions before mainnet. A single-region validator set means chain halts on regional outage regardless of backend resilience.

---

#### DR Scenario 8 — Chain Binary Upgrade Gone Wrong

**Trigger:** chain halts or produces corrupted state after an upgrade  
**RTO target: < 4 hours**

```
Step 1  Identify failure: upgrade handler panic, state corruption, or halt   ~15 min
Step 2  Halt all validators (coordinate emergency stop)                      ~30 min
Step 3  Restore Write DB, Read DB, Relayer DB from pre-upgrade snapshot      ~30 min
Step 4  Restore chain data directory from pre-upgrade archive node snapshot  ~30 min
Step 5  Roll back chain binary to previous version on all validators         ~30 min
Step 6  Restart chain at pre-upgrade height with old binary                  ~30 min
Step 7  Verify chain resumes; post-mortem on failed upgrade handler          ~30 min
Total                                                                        ~3.5 h
```

**Prevention:** mandatory upgrade drill on testnet (Phase 6.4) before any mainnet upgrade. Pre-upgrade snapshots taken automatically as part of upgrade runbook.

---

### Backup Testing and Drill Schedule

| Drill | Frequency | Responsible | Pass Criteria |
|---|---|---|---|
| PostgreSQL Write DB restore | Quarterly | Infra lead | Restore completes in < 4 h; ingestion reconciliation fills gap; all genesis invariants pass |
| PostgreSQL Read DB rebuild | Quarterly | Infra lead | Fresh Read DB catches up from Write DB; API returns correct data after catch-up |
| Patroni auto-failover | Quarterly | Infra lead | Failover < 30 s; zero data loss; ingestion reconnects automatically |
| NATS full cluster loss + recovery | Monthly (in chaos suite) | Infra lead | Cluster recovers < 1 h; back-fill from Write DB completes; no Read DB gaps |
| Vault restore + unseal | Semi-annual | Security lead | Snapshot restores; 5-of-7 holders assembled in drill; all secrets readable in < 1 h |
| Horcrux shard restore | Annual | Validator team | Shard from USB restores to test machine; 2-of-3 signing produces valid signatures; no double-sign |
| Full Region A loss simulation | Annual | All on-call | Full system operational in Region B < 2 h; chain never halted; RTO logged |
| k8s etcd restore | Semi-annual | Infra lead | Cluster state restored; all workloads resume correctly |
| Archive node restore | Semi-annual | Infra lead | Snapshot downloads and decompresses; archive node serves BlockResults for all heights |
| Chain binary upgrade drill | Per upgrade (on testnet) | Chain team | Upgrade on testnet completes without halt; upgrade handler executes correctly |
| Pre-upgrade snapshot verification | Per upgrade | Infra lead | Snapshot taken; rollback procedure tested on testnet before mainnet upgrade |

**All drills must be logged:** date, participants, RTO achieved, pass/fail, gaps identified, remediation plan. Drill logs reviewed in monthly security review.

---

### Backup Infrastructure Costs and Tooling

| Component | Tool | Notes |
|---|---|---|
| PostgreSQL WAL archiving | pgBackRest or Barman | pgBackRest recommended: parallel WAL archiving, compression, cloud-native S3/GCS support |
| PostgreSQL base backups | pgBackRest `backup --type=full` | Nightly; encrypted; cross-region replicated |
| Chain archive node backup | Custom script: `tar + zstd` of `$CHAIN_HOME/data/` | Nightly; uploaded to S3/GCS |
| NATS cluster backup | Stream config in git; NKeys in Vault; bridge repopulate script | No special tooling required |
| Vault backup | Vault snapshot agent (built-in) | Hourly Raft snapshots to S3/GCS |
| k8s backup | Velero + Restic (for PVC volumes) | Hourly; 7-day retention |
| etcd backup | `etcdctl snapshot save` via CronJob | Every 6 hours; S3/GCS |
| Horcrux shard backup | Manual; encrypted USB + hardware password manager | Air-gapped only; annually verified |
| Monitoring | Prometheus alerts for: backup job failures, WAL archiving lag > 5 min, last successful backup > 25 h, Vault unseal count change, NATS node count < 3 | All backup failures are P1 alerts |

---

### Backup Monitoring Alerts

| Alert | Condition | Severity | Action |
|---|---|---|---|
| WAL archiving stalled | Last archived WAL segment > 5 min ago | P1 | Investigate archive_command; verify S3/GCS connectivity |
| Base backup overdue | Last successful base backup > 25 hours ago | P1 | Run manual base backup immediately |
| Vault snapshot overdue | Last Vault snapshot > 2 hours ago | P1 | Verify Vault snapshot agent; run manual snapshot |
| Velero backup failed | Last Velero backup failed or missing | P2 | Investigate Velero logs; run manual backup |
| Patroni leader change | Unexpected leader change event | P1 | Verify new primary healthy; provision standby replacement |
| Horcrux signer offline | Signer machine not responding | P1 | Provision replacement; restore shard from USB |
| NATS node count < 3 | One or more NATS nodes down | P1 | Provision replacement NATS node; verify R=3 |
| Replica lag > 60 s | PostgreSQL standby replication lag | P1 | Investigate network between regions; check disk I/O |
| S3/GCS bucket inaccessible | Backup writes failing | P0 | Verify IAM credentials; verify bucket policy; failover to Region B bucket |

---

### Summary: What You Can Lose and How Long Recovery Takes

| Failure | Data Loss | Service Impact | Recovery |
|---|---|---|---|
| Write DB primary failure | Zero (sync standby) | None (auto-failover < 30 s) | Automatic |
| Write DB total loss | Near-zero (continuous WAL archiving — only in-flight WAL segments at moment of failure); any remaining gap filled by ingestion startup reconciliation from archive node | Full CQRS pipeline down during restore | < 4 h |
| Read DB total loss | Zero (rebuilt from Write DB) | API degraded (partial data) during rebuild | < 6 h |
| NATS full cluster loss | Zero (back-filled from Write DB) | Projection paused during recovery | < 1 h |
| Chain archive node loss | None (chain continues) | Ingestion gap-filling unavailable | < 24 h (full sync) |
| Vault total loss | Zero (Raft snapshot, hourly) | All services down during restore | < 1 h |
| Horcrux signer loss (1 of 3) | None (chain continues signing) | None | < 24 h |
| Region A total loss | Zero (sync replication) | Brief degradation; auto-failover | < 2 h |
| Chain upgrade failure | Zero (pre-upgrade snapshot) | Chain halted during rollback | < 4 h |

---

## Summary Timeline

```
Month 1  (Weeks 1-4):
  Week 1     - Phase 0: NDA, team sizing (12-15 eng confirmed), ADRs including bridge threat
                        model and nonce generation spec, repos, CI (randomized sim seed),
                        toolchain
  Weeks 1-4  - Phase 1: scaffold, x/staking compat scope, genesis supply math (S-C),
                         CosmWasm, devnet (3 PG DBs + NATS 3-node + CORS Envoy)
  Weeks 3-4  - Phase 2 start: x/validator (Team A, compat scope resolved first)
  Week 4     - Phase 3 start: Constitution + Treasury (Team B)

Month 2  (Weeks 5-8):
  Weeks 5-8  - Phase 2: x/certification (chain-state degraded mode) + x/oracle
                         (commit-reveal + BSC outage policy) (Team A)
  Weeks 5-8  - Phase 3: Reserve Fund + Governance contracts (Team B)
  Weeks 6-9  - Phase 2.7: Oracle Aggregator Microservice (Team C)
  Weeks 7-8  - Phase 2: x/milestone (deadline clock pause, stale-blocked edge case) (Team A)
  Weeks 8-12 - Phase 2.8: E2E test suite (Team C, maintained throughout)

Month 3  (Weeks 9-12):
  Weeks 9-10 - Phase 2: x/milestone completes (direct stale-blocked→achieved path)
  Weeks 9-10 - Phase 3 complete: all 4 contracts + on-chain devnet tests (ExecuteMsg/QueryMsg pause)
  Weeks 9-11 - Phase 2: x/settlement (Ed25519, chain-id domain, timestamp tolerance)
  Weeks 10-12 - Phase 2 complete: x/governance-ext (gas limit bounds, MsgMigrateContracts bypass,
                                   SimGov wrapper for gated operations)
  Weeks 10-15 - Phase 4: LockBox (keccak256 nonce) + x/bridge + relayer (promotion ladder)
  Week 12    - Phase 5 start: dedicated sub-team (5-6 eng): proto → Write DB schema + backup
                               → ingestion (reconciliation + back-fill) → projection (two modes)
                               → api (account:stream sub) → Envoy (CORS, separate clusters)
                               → SDK (auto-reconnect)
  Week 14    - Pre-engage 3 auditors; testnet access provided

Month 4  (Weeks 13-17):
  Weeks 13-15 - Phase 4 complete: bridge testnet, supply invariant under load
  Weeks 13-18 - Phase 5: two-DB CQRS pipeline (5-6 eng sub-team, 6 weeks total)
  Weeks 13-18 - Phase 6 start: 3-ring topology (Horcrux only), 2-cluster k8s (WireGuard, no Istio)
  Weeks 15-18 - Phase 7: wallet configs, dApp (streaming auto-reconnect), SDK
  Week 17    - CODE FREEZE for chain binary
                NATS chaos tests run (including full cluster outage + back-fill verification)
                E2E suite: all primary and secondary scenarios green

Month 5  (Weeks 18-21):
  Weeks 18-19 - Phase 5 complete: full pipeline verified, CORS tested, streaming reconnect tested
  Weeks 15-19 - Phase 6 complete: stable testnet (Weeks 15-19, 4+ weeks); zero halts after
                                   code freeze (Week 17)
  Weeks 18-19 - Phase 7 complete: dApp live on testnet, wallet configs published
  Weeks 18-21 - Phase 8: hardening (frozen binary), key rotation + emergency runbooks,
                          circuit-breaker EOA runbook, cold multi-sig holder declaration,
                          PostgreSQL backup drill, internal pen test (with pass criteria),
                          auditor documentation
  Week 21    - Phase 9 start: auditor kickoff (7 weeks testnet context)

Month 6  (Weeks 22-25):
  Weeks 21-22 - Active audit (two or three firms)
  Week 23    - Findings + remediation sprint
  Weeks 23-24 - Remediation + re-review
  Week 24 end - Final audit report
  Week 25    - Phase 10: mainnet genesis ceremony + staged launch
```

---

## Risk Register

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Bridge exploit post-launch | Medium | Critical | Two audits; circuit-breaker EOA < 60s; rate limits; bug bounty; quarterly drill |
| Supply cap breach | Low | Critical | Atomic check+mint; supply math (S-C) in genesis script; bridge audit scope |
| `x/staking` fork breaking `x/distribution`, `x/gov`, IBC | High | Critical | Full compat scope Phase 1.2; all four modules audited and tested on devnet |
| `x/certification` chain halt under dropout | Medium | Critical | Chain-state degraded mode (deterministic, not per-validator); liveness alerts |
| NATS SPOF | High | Critical | 3-node R=3 cluster; Write DB back-fill on reconnect; chaos-tested |
| Oracle front-running on vesting | Medium | High | Commit-reveal; value never in mempool |
| Oracle staleness forcing milestone expiry | Medium | High | Deadline clock paused during `stale-blocked`; staleness cannot force expiry |
| Witness signature replay (cross chain-id) | Medium | High | Ed25519 domain separator includes chain_id; unit-tested |
| CosmWasm governance deadlock | Low | Critical | `MsgMigrateContracts` bypasses Constitution check; cold multi-sig pauses funds during replacement |
| Constitution pause blocks all governance | Low | High | `EmergencyPause` blocks ExecuteMsg only; QueryMsg unaffected; tested in devnet |
| `cw-multi-test` passes but on-chain fails | High | High | Mandatory on-chain devnet tests for all fund-movement paths |
| Bitmap nonce: user-supplied nonce collision | High (if user-supplied) | High | LockBox generates nonce via keccak256; user does not supply nonce |
| Relayer duplicate submission on failover | Medium | High | Deterministic promotion ladder (only one relayer promotes at a time) |
| Relayer quorum stuck | Medium | High | Quorum timeout; promotion ladder; stuck event alert; runbook |
| `x/authz` relayer impersonation | Medium | High | MsgBridgeIn/Out/Oracle blocked in x/authz blocked list from day one |
| Circuit-breaker EOA key compromise | Low | Medium | Runbook; < 4h rotation target; quarterly contact list drill; Gnosis Safe can permanent-pause during rotation |
| Cold multi-sig key compromise | Low | Critical | 5-of-7 threshold; geographic distribution; annual rotation drill; remaining holders initiate emergency rotation |
| NATS back-fill gap (events never published) | Medium | High | `nats_published` column; back-fill on reconnect; monitoring alert if backlog > 1 min |
| Write DB crash leaves ingestion gap | Medium | High | Startup reconciliation from MAX(block_height); PITR backup |
| Aggregate projection double-count | High | High | Aggregate projections run as singleton with advisory lock |
| Concurrent projection race on aggregates | High | High | Singleton StatefulSet for aggregate projections; fan-out Deployment for KV projections only |
| gRPC streaming drop on Envoy restart | High | Medium | SDK auto-reconnect (exponential backoff, max 30s); tested in chaos suite |
| CORS blocking browser API calls | High | High | CORS configured in Envoy from devnet; tested on all /api/* routes |
| PostgreSQL multi-region failure | Medium | Critical | Synchronous standby; Patroni failover; PITR; quarterly restore drill |
| Write DB grows unbounded | High | Medium | **If TimescaleDB adopted:** automatic chunk compression (7-day chunks compress after 7 days; typically 10–20× ratio); no manual partition management needed. **If TimescaleDB rejected:** monthly range partitioning by block_height; WAL archiving; cold storage archive |
| Phase 5 scope underestimated | High | High | Dedicated sub-team (5-6 eng, 6 weeks); separate from E2E maintenance team |
| Team size insufficient | High | Critical | Confirmed in Phase 0.1; 12-15 eng minimum; timeline revised if below |
| Audit Scope D not covered by specialist | Medium | High | Third audit firm or internal red team with documented findings |
| `x/params` deprecation | High | High | Per-module `MsgUpdateParams` from day one; no custom module uses `x/params` |
| Genesis supply error | Low | Critical | Invariant tests in genesis script; three independent verifications |
| Auditor backlog delay | Medium | Medium | Pre-engage by Week 14; three firms in parallel |
| Code freeze vs hardening conflict | Medium | High | Code freeze Week 17; hardening on frozen binary; binary changes = planned testnet upgrade |
| Ante handler conflict (EVM + CosmWasm) | High | Critical | Use `evmante.NewAnteHandler` only; no custom CosmWasm ante handler; `TestCosmWasmEVMCoexistence` passes 5× before testnet; Phase 2.9 is a dedicated spike |
| EVM StateDB corrupting CosmWasm storage | Medium | Critical | `x/evm` StateDB and `wasmd` use separate KV store keys; `TestCosmWasmEVMCoexistence` Test 2 verifies x/bank balances are correct after dual-runtime writes in same block |
| EVM chain ID collision (not registered) | High | High | Register on chainlist.org before genesis; confirm in Phase 0 ADR; Chain ID is immutable post-genesis |
| EVM denomination misconfiguration (MetaMask shows wrong amounts) | High | High | ADR specifies `BaseDenom = "atoken"`, 18 decimals; MetaMask config in wallet setup guide; `x/erc20` conversion tested on devnet; immutable post-genesis |
| `debug_` JSON-RPC namespace exposed in mainnet | Medium | High | Disable in mainnet `app.toml` (`api = ["eth","net","web3","txpool"]`, no `debug`); verify in pre-mainnet config audit; Scope E item |
| Ethermint dependency brings in incompatible Go module | High | Medium | Pin exact version in Phase 0 ADR before Phase 1; run `go mod tidy` immediately; resolve conflicts before any other Phase 1 work; no `replace` directives for Ethermint sub-modules |

---

## Definition of Done

- [ ] Team: 12–15 engineers confirmed; timeline adjusted if below minimum
- [ ] All Phase 0 ADRs signed off: includes bridge threat model, keccak256 nonce spec, `x/authz` blocked list, cold multi-sig key holder identities, CORS policy, streaming reconnect spec, PostgreSQL replication + backup spec, circuit-breaker EOA runbook
- [ ] `x/staking` compat scope (Phase 1.2): `x/distribution`, `x/gov`, `x/slashing`, IBC `HistoricalInfo` — all verified on devnet
- [ ] `x/authz` blocked message types: `MsgBridgeIn`, `MsgBridgeOut`, `MsgSubmitOracleCommit`, `MsgRevealOracleReport`, `MsgSettlement`, `/ethermint.evm.v1.MsgEthereumTx` — all six blocked; all six tested (`TestAuthzEVMBlock` covers the EVM entry)
- [ ] All seven custom Go modules: unit, integration, simulation with `WeightedOperations` (governance-gated ops use `SimGovProposalMsg` wrapper); `--NumBlocks=5000 --Seed=<random>` passes
- [ ] Oracle commit-reveal: value never visible in mempool (only hash); BSC outage → skip round (no slash); insufficient round → `x/milestone` treats as stale
- [ ] `x/certification` degraded mode: chain-state driven (not per-validator local); determinism verified across 3+ validator nodes in integration test; chain does not halt under dropout
- [ ] `x/milestone` deadline clock: paused during `stale-blocked`; resumes on feed recovery; direct `stale-blocked→achieved` path tested; staleness cannot force expiry
- [ ] Gas limit governance parameter: bounds [100,000–2,000,000] enforced; `MsgMigrateContracts` and gas limit proposals bypass Constitution check
- [ ] CosmWasm `EmergencyPause`: blocks `ExecuteMsg` only; `QueryMsg` succeeds during pause; tested on devnet
- [ ] CosmWasm cold multi-sig: 5-of-7 key holders declared publicly; Governance replacement procedure tested on devnet
- [ ] LockBox nonce: keccak256-generated by contract; user does not supply nonce
- [ ] Bridge relayer promotion ladder: only one relayer submits at a time; no duplicate `MsgBridgeIn` under failover scenario
- [ ] NATS back-fill: `nats_published` column; back-fill on reconnect; full cluster outage → back-fill → Read DB consistent (tested in chaos suite)
- [ ] Ingestion startup reconciliation: gap from crash filled on restart; no holes in Write DB
- [ ] Aggregate projections singleton: advisory lock; no double-counting under concurrent writes
- [ ] Write DB users: `ingestion_writer` (INSERT only), `projection_reader` (SELECT only on write schema) — both configured and enforced
- [ ] `module/api`: reads Read DB only; subscribes `account:stream` only; no Write DB connection, no `account:chain` subscription
- [ ] Envoy: CORS on all `/api/*` routes (tested from browser); separate upstream clusters for gRPC-Web and REST; per-identity rate limiting (relayer cert, wallet address, IP)
- [ ] gRPC streaming SDK auto-reconnect: tested under Envoy rolling restart; users see no interruption
- [ ] PostgreSQL: synchronous standby in second region; Patroni failover; WAL archiving + PITR active; monthly range partitioning on Write DB; quarterly restore drill passed
- [ ] Circuit-breaker EOA key compromise runbook: documented; contact list drilled quarterly; < 4h rotation target
- [ ] NATS chaos: full 3-node outage → back-fill → consistent (tested)
- [ ] Phase 5 complete by Week 18 (6 weeks, dedicated sub-team)
- [ ] Code freeze Week 17; zero chain halts after code freeze (Weeks 17–19)
- [ ] Internal pen test: zero critical/high open; all scenarios tested (original 12 scenarios + 6 EVM attack scenarios added in Phase 8.5 = 18 named scenarios; total pen test coverage confirmed by lead security engineer); report signed off
- [ ] Reproducible builds: identical SHA256 on two machines; SLSA provenance published
- [ ] Audit Scope D: covered by specialist firm or internal red team with documented results
- [ ] External audit: zero unresolved critical/high; report published before mainnet genesis
- [ ] Genesis: each validator independently verifies supply invariants; ceremony documented
- [ ] Emergency runbooks written and drilled: bridge pause < 60s, NATS recovery, Horcrux signer failure, CosmWasm governance replacement, PostgreSQL restore
- [ ] **TimescaleDB (if adopted):** TimescaleDB ADR signed before Phase 5.2; devnet images use `timescale/timescaledb:latest-pg16`; all 4 hypertables created and accepting writes from `module/projection`; all 5 continuous aggregates (`tps_1h`, `block_time_1h`, `oracle_price_1h`, `validator_uptime_1d`, `bridge_volume_1h`) refreshing on schedule; compression policy active on Write DB; `synchronous_commit = off` confirmed on Write DB primary only; dashboard queries verified returning correct values against testnet data; TimescaleDB PITR drill passed (restore does not corrupt compressed chunks)
- [ ] **Celatone (if adopted):** self-hosted Celatone instance reachable at `/celatone` on testnet and mainnet; JSON Schema uploaded for all five CosmWasm contracts and verified to show decoded ExecuteMsg/QueryMsg fields; contract migration history visible for all Phase 3 deployments; referenced explicitly in auditor documentation and Phase 9 kickoff materials
- [ ] **Analytics dashboard (Phase 7.3):** all 6 analytics RPCs (`GetTps`, `GetBlockStats`, `GetBridgeVolume`, `GetOraclePrice`, `GetValidatorUptime`, `StreamChainStats`) implemented in `module/api` and returning testnet data; `/dashboard` route accessible in Next.js dApp; all 6 dashboard pages render data without errors; `StreamChainStats` auto-reconnect tested under Envoy rolling restart
- [ ] **EVM — Phase 2.9 gate (all must pass before testnet launch):**
  - [ ] `github.com/evmos/ethermint` pinned at exact version in `go.mod` and Phase 0 ADR
  - [ ] `x/feemarket`, `x/evm`, `x/erc20` wired in `app.go`; `skip-mev/x/feemarket` absent from `go.mod`
  - [ ] Module init order enforced: `x/feemarket` → `x/evm` → `x/erc20` in `SetOrderInitGenesis`, `SetOrderBeginBlockers`, `SetOrderEndBlockers`
  - [ ] `evmante.NewAnteHandler` in use as the sole ante handler; no separate CosmWasm ante handler
  - [ ] `TestCosmWasmEVMCoexistence` CI green on 5 consecutive devnet runs
  - [ ] `TestAuthzEVMBlock` passes: `/ethermint.evm.v1.MsgEthereumTx` authz grant rejected at protocol level
  - [ ] `cast block-number --rpc-url .../evm-rpc` returns correct block height on devnet
  - [ ] MetaMask connects to devnet via `/evm-rpc`; account balance shows correctly with 18 decimals
  - [ ] Blockscout indexes devnet EVM txs; Solidity contract deployment visible and ABI-decoded
  - [ ] Native token registered as ERC-20 via `x/erc20`; MetaMask balance matches Cosmos-side `utoken` balance (with correct decimal conversion)
  - [ ] EVM simulation `WeightedOperations` completes `--NumBlocks=5000` without panic
  - [ ] `debug_` namespace confirmed absent from mainnet `app.toml` (only `eth`, `net`, `web3`, `txpool`)
  - [ ] Scope E added to audit engagement letter; Evmos/Ethermint audit reports provided to auditors as background
- [ ] **EVM — Wallet and dApp:**
  - [ ] Keplr/Leap chain config uses `coinType = 60` (Ethereum BIP-44); verified MetaMask derives same hex address from same private key
  - [ ] MetaMask "Add to MetaMask" one-click button in dApp; uses `wallet_addEthereumChain` with correct chain ID and `/evm-rpc` URL
  - [ ] wagmi v2 + viem v2 integrated in Next.js dApp; EVM account page shows TOKEN balance; EVM transactions submit via MetaMask
  - [ ] Blockscout accessible at `/blockscout` on testnet; EVM txs submitted from dApp visible in Blockscout within 1 block
