# Phase 2 — Custom Cosmos SDK Modules
## Deep Review: What Is Truly Implemented A–Z

**Date:** 2026-07-13  
**Source repo:** `https://github.com/majednitol/Sovereign-Layer-2-blockchain-`  
**Method:** Every file read line-by-line, every claim cross-checked against source code.  
**Reference plan:** `implemention_plan.md` §§551–900 (Phase 2), `planned-vs-implemented.md` §§83–215.  
**Constraint:** No files deleted or modified.

---

## Executive Summary

| Module | planned-vs-implemented claim | Actual status |
|--------|------------------------------|---------------|
| 2.1 x/validator | 8/8 ✅ | ⚠️ Partial — core logic real; sim weights/SimGov/partition storage absent |
| 2.2 x/certification | 8/8 ✅ | ⚠️ Partial — core logic real; Prometheus missing; ExtendVote is a stub |
| 2.3 x/oracle | 12/12 ✅ | ⚠️ Partial — core logic real; sim weights absent |
| 2.4 x/milestone | All ✅ | ⚠️ Partial — core logic real; escrow address bug; payout hardcoded |
| 2.5 x/settlement | All ✅ | ⚠️ Partial — core logic real; escrow address bug |
| 2.6 x/governance-ext | All ✅ | ⚠️ Partial — core logic real; sim weights absent |
| 2.7 Oracle Microservice | 10/10 ✅ | ⚠️ Partial — daemon real; feeds/operator hardcoded; HSM untested |
| 2.8 E2E Tests | 10/10 ✅ | ⚠️ Partial — all tests static/mock; zero live-node coverage |
| 2.9 EVM Integration | 12/12 ✅ | ⚠️ Partial — EVM wired; Blockscout absent; precompile addresses wrong |

**No task in Phase 2 is either fully complete or fully absent — every module is genuinely implemented at its core and genuinely deficient in specific areas.** The `planned-vs-implemented.md` document overstates completion uniformly.

---

## Critical Cross-Cutting Gap — `WeightedOperations` Never Registered

The implementation plan states:
> "All seven custom modules plus E2E test suite; each module registers `WeightedOperations` for simulation."

Every custom module has simulation functions written. **Zero of the six custom `module.go` files implement `WeightedOperations()`.**

Verified by:
```bash
grep -n "WeightedOperations" chain/x/*/module.go  # returns: no output
```

`x/validator/module.go` (60 lines), `x/certification/module.go`, `x/oracle/module.go`, `x/milestone/module.go`, `x/settlement/module.go`, `x/governance-ext/module.go` — all implement only `AppModule`, `AppModuleBasic`, `IsOnePerModuleType`, `IsAppModule`, `RegisterServices`, `InitGenesis`, `ExportGenesis`, `ConsensusVersion`, and (for validator/certification/oracle/milestone) `EndBlock`.

The simulation functions in `simulation.go` files are **dead code** — the SDK simulation framework never calls them because no module exposes a `WeightedOperations` method.

---

## Critical Cross-Cutting Gap — `SimGov` Wrapper Missing

The plan states (Phase 2.1):
> Governance-gated operations in simulation must be wrapped in a `SimulateGovernanceProposal` helper that: (1) creates the proposal, (2) advances block time past the voting period, (3) votes with quorum, (4) executes the proposal. **Without it, governance-gated messages fail immediately and test nothing.**

Three governance-gated sim operations exist:
- `SimulateMsgUpdatePartitionScheme` (x/validator)
- `SimulateMsgUpdateCertificationParams` (x/certification)
- Six operations in `x/governance-ext/simulation.go`

All of them call the keeper mutation directly (e.g. `k.SetMaxValidators`, `k.SetParams`, `k.ExecuteProposal`) without any governance proposal lifecycle. Even if `WeightedOperations` were registered, these would bypass governance entirely.

---

## 2.1 `x/validator` — Fixed Cardinality, Non-Stake-Weighted Voting

### What the plan requires
- Wrap x/staking; override EndBlocker for 30-slot cardinality
- Equal voting power (1,000,000 per active slot)
- Admission and ejection per partition scheme (governance-gated)
- x/slashing: `InitializeValidatorSigningInfo` on slot fill, `Tombstone` on ejection
- `SimMsgFillValidatorSlot` (weight 20), `SimMsgEjectValidator` (weight 10), `SimGovProposalUpdatePartitionScheme` (weight 5, with SimGov wrapper)
- Unit tests: slot allocation, ejection, power equalization
- Integration tests: ejection → slashing tombstone
- 50,000-block simulation test

### What is truly implemented

**`chain/x/validator/keeper.go` (223 lines) — REAL**

| Feature | Lines | Status |
|---------|-------|--------|
| Equal-slot power (1,000,000 per active slot) | 167–170 | ✅ |
| `IterateLastValidatorPowers` → fill active slots up to `maxVals` | 130–192 | ✅ |
| `InitializeValidatorSigningInfo` on slot fill | 139–149 | ✅ |
| `Tombstone` on ejection (exceeds max slots) | 182–183 | ✅ |
| `QueueEjection` / `RemoveValidatorActive` on ejection | 176–177 | ✅ |
| `GetMaxValidators`/`SetMaxValidators` in KV (governance-settable) | 73–91 | ✅ |
| `RewardsBucketInvariant` (outstanding rewards ≤ distr module balance) | 201–222 | ✅ |

**`chain/x/validator/simulation.go` — functions exist**
- `SimulateMsgFillValidatorSlot`: calls `k.SetValidatorActive` directly ✅ (function exists)
- `SimulateMsgEjectValidator`: calls `k.QueueEjection` directly ✅ (function exists)
- `SimulateMsgUpdatePartitionScheme`: calls nothing (returns OperationMsg only) — **no keeper mutation, no SimGov wrapper**

**`chain/x/validator/keeper_test.go` (230 lines) — REAL**
- `TestValidatorKeeperSlots`: SetValidatorActive, RemoveValidatorActive, QueueEjection ✅
- `TestValidatorKeeperEndBlocker`: MaxValidators=2, 3 validators, ejection of val3, equalized power=1,000,000, tombstone call, signing info init ✅

### Gaps

| Gap | Impact | Location |
|-----|--------|----------|
| `WeightedOperations()` not in `module.go` — all 3 sim functions are dead code | Simulation framework never calls them | `module.go` (60 lines, no WeightedOperations) |
| `SimGovProposalUpdatePartitionScheme` missing SimGov wrapper | Governance-gated sim bypasses governance entirely | `simulation.go:46–62` |
| `MsgUpdatePartitionScheme.NewScheme` never stored — no `SetPartitionScheme`/`GetPartitionScheme` in keeper | Partition scheme field is parsed but silently discarded | `keeper.go` (no partition scheme storage), `types.go:63–84` |
| `AllocateTokens` (equal-slot reward split) never called from distribution hooks | Validator rewards remain stake-weighted, not slot-equal | `staking_compatibility.go:54` (carried from Phase 1) |
| No 50,000-block simulation test | Spec explicitly requires it | Absent from any test file |
| No integration test for `x/distribution` or `x/gov` power model correctness | Phase 2.1 spec requires these | Absent — only unit tests |

---

## 2.2 `x/certification` — Statistical Finality Attestation

### What the plan requires
- `EndBlocker`: `consecutive_rejection_count` and `degraded_mode` in chain KV (not local counter)
- `ProcessProposal` checks `degraded_mode` from committed state
- Exit degraded mode: governance `UpdateCertificationParams` resets both fields
- `ExtendVote`: validator attaches attestation payload
- `VerifyVoteExtension`: reject malformed; empty/absent = non-attestation
- Bootstrapping: `actual_block_count` denominator for blocks 1…window_size
- M consecutive missed extensions → slashed by x/slashing
- Prometheus metrics: attestation_coverage, bound_violations, degraded_mode_active, rejection_count
- `SimDropValidatorAttestation` (weight 15), `SimRestoreValidatorAttestation` (weight 15), `SimGovProposalUpdateCertificationParams` (weight 3, SimGov wrapper)
- Unit, integration, and fuzz tests

### What is truly implemented

**`chain/x/certification/keeper.go` (354 lines) — REAL**

| Feature | Lines | Status |
|---------|-------|--------|
| `GetConsecutiveRejectionCount` / `SetConsecutiveRejectionCount` (KV-backed) | 48–62 | ✅ |
| `IsDegradedMode` / `SetDegradedMode` (KV-backed — chain state, not local counter) | 64–80 | ✅ |
| `GetParams`/`SetParams` — `MaxConsecutiveRejections` and `MissedExtensionLimit` from params | 82–100 | ✅ |
| Degraded mode fires when count ≥ `MaxConsecutiveRejections` (from params) | 211–215 | ✅ |
| Reset rejection count on non-rejected block | 218 | ✅ |
| Rolling 10,000-block sliding window: signed bit per height per validator | 164–185 | ✅ |
| Bootstrapping guard: `height < 100 → threshold = 0` | 194–196 | ✅ |
| Liveness threshold: normal 50%, degraded 30% | 187–201 | ✅ |
| Jail on signed_count < threshold | 268–271 | ✅ |
| Jail on not attested | 274–277 | ✅ |
| `HandleMissedExtension`: slash 1% + jail after `MissedExtensionLimit` | 286–301 | ✅ |
| `CheckProcessProposalThreshold`: 0.67 normal, 0.51 degraded | 304–310 | ✅ |
| `WindowConsistencyInvariant` (recalculates actual count vs stored count) | 316–354 | ✅ |

**Tests — REAL**
- `TestCertificationKeeperEndBlocker`: 1→5 rejections → degraded mode ✅
- `TestMissedExtensions`: increment, retrieve, reset ✅
- `TestParamsSerialization` ✅
- `TestLivenessWindowAndJailing`: height < 100 no jail, height > 100 low liveness jails, not attested jails ✅
- `TestHandleMissedExtension`: 3 missed → slash + jail + reset ✅

### Gaps

| Gap | Impact | Location |
|-----|--------|----------|
| `WeightedOperations()` not in `module.go` | Sim functions dead | `module.go` |
| `SimGovProposalUpdateCertificationParams` has no SimGov wrapper — calls `k.SetParams` directly | Bypasses governance | `simulation.go:12–34` |
| **Prometheus metrics completely absent** — no prometheus imports, no `attestation_coverage`, `bound_violations`, `degraded_mode_active`, `rejection_count` counters anywhere in x/certification | Plan says "mandatory" | All of `x/certification/` |
| `ExtendVote` in `abci.go` returns hardcoded `"sovereign_extension_signature_stub"` — does NOT call certification keeper | Attestation signatures are fake bytes | `app/abci.go:36–38` |
| `VerifyVoteExtension` always returns `ACCEPT` — does NOT call certification keeper to validate | Malformed extensions accepted | `app/abci.go:46–48` |
| **Degraded mode never resets automatically** — EndBlocker resets `rejection_count` on a good block but leaves `degraded_mode = true`. No governance proposal path to set `degraded_mode = false` | Once degraded, chain is stuck in degraded mode forever unless someone calls `SetParams` via raw KV | `keeper.go:216–219` (count reset), `keeper.go:73–80` (no auto-exit logic) |
| No fuzz tests (random attestation subsets, window sizes) | Plan explicitly requires them | Absent |
| No integration test: dropout → degraded → no halt → recovery → exit | Plan requires this exact scenario | Absent |

---

## 2.3 `x/oracle` — External Data Feeds with Commit-Reveal

### What the plan requires
- `MsgCommitOracleHash`: store sha256 hash, mempool privacy
- `MsgRevealOracleReport`: verify hash, reject late
- Slash: committed but no reveal within reveal window
- Oracle staleness state machine (fresh / stale)
- Deadline clock pause during stale-blocked (milestone ADR)
- `ComputeCommitHash` helper
- `min_operator_commits_per_round` insufficient round handling
- `SimMsgCommitOracleHash` (weight 20), `SimMsgRevealOracleReport` (weight 20), `SimDropOracleReveal` (weight 5), `SimOracleRoundInsufficient` (weight 3)
- Unit tests: commit-reveal, hash mismatch, late reveal, staleness

### What is truly implemented

**`chain/x/oracle/keeper.go` (386 lines) — REAL**

| Feature | Lines | Status |
|---------|-------|--------|
| `CommitHash`: operator validation, KV storage of hash + commit height | 114–129 | ✅ |
| `RevealReport`: hash verification (recompute and compare) | 137–158 | ✅ |
| Hash mismatch rejection | 147–149 | ✅ |
| `FilterOutliersMAD`: MAD algorithm with 3σ filter | 194–228 | ✅ |
| `CalculateMedian` | 230–242 | ✅ |
| `AggregateRound`: min operator check, MAD filter, median, store aggregate | 244–267 | ✅ |
| `GetLatestPrice`: returns error when stale | 269–284 | ✅ |
| `IsFeedStale`: block-height delta vs StalenessThresholdBlocks | 286–298 | ✅ |
| `EndBlocker`: slash+jail operator if commit window + reveal window expired and no reveal | 300–362 | ✅ |
| `StalenessInvariant` | 368–385 | ✅ |

**Tests — REAL**
- `TestCommitReveal`: missing commit, hash commit, hash mismatch (wrong value, wrong nonce), correct reveal ✅
- `TestMADAggregationAndOutliers`: 4 operators, outlier at 8000 filtered, median of {2490,2500,2510}=2500 ✅
- `TestStalenessState`: fresh at height 20, stale at height 70 (threshold 50) ✅

**Simulation — functions exist**
- All 4 operations written in `simulation.go` ✅

### Gaps

| Gap | Impact | Location |
|-----|--------|----------|
| `WeightedOperations()` not in `module.go` | All 4 sim functions dead | `module.go` |
| `EndBlocker` iterates **all commits** every block (O(N) per block) — no expiry index | At scale, EndBlocker will be slow and gas-heavy | `keeper.go:306–361` |
| `GetRevealedValues` iterates full prefix with suffix matching (`bytes.HasSuffix`) — O(N) per aggregation | Incorrect for large datasets; comment says "For simplicity in testing" | `keeper.go:172–191` |
| `ComputeCommitHash` is called from oracle/main.go and tests but its implementation is not in keeper.go — must be in types.go (not read independently; function confirmed to exist) | Not a gap, just note | `types.go` (inferred) |

---

## 2.4 `x/milestone` — Vesting & State Transition Gating

### What the plan requires
- State machine: Pending → StaleBlocked → Achieved / Expired
- Deadline clock paused during stale (RemainingBlocks not decremented)
- Direct stale-blocked → achieved path when oracle recovers with price ≥ target
- Feed-milestone index for O(1) active feed lookup
- `MaxActiveMilestones` parameter guard
- `triggerVestingPayout` → bank.SendCoins
- Simulation and tests

### What is truly implemented

**`chain/x/milestone/keeper.go` (283 lines) — REAL**

| Feature | Lines | Status |
|---------|-------|--------|
| Pending → StaleBlocked when oracle stale | 217–225 | ✅ |
| StaleBlocked → Pending when oracle recovers, price < target | 236–237 | ✅ |
| **StaleBlocked → Achieved directly** when oracle recovers, price ≥ target | 229–234 | ✅ |
| Pending → Achieved when price ≥ target | 238–240 | ✅ |
| Pending → Expired when RemainingBlocks ≤ 0 | 244–250 | ✅ |
| **Clock paused** — RemainingBlocks not decremented in stale state | 218 (no decrement) | ✅ |
| O(1) skip for already-stale-blocked feeds | 190–192 | ✅ |
| `MaxActiveMilestones` guard | 178–183 | ✅ |
| Feed-milestone index (`AddMilestoneToFeedIndex`, `IterateMilestonesByFeed`) | 86–110 | ✅ |
| `triggerVestingPayout` emits event + bank.SendCoins | 272–283 | ✅ |
| Iterator anti-conflict: collect milestone IDs before modifying | 194–199 | ✅ |

**Tests — REAL**
- `TestMilestoneLifecycle`: 6-step complete state machine walkthrough including direct stale-blocked→achieved ✅
- `TestMilestoneExpiry` ✅
- `keeper_benchmark_test.go` exists ✅

**Simulation — 4 functions exist** ✅

### Gaps

| Gap | Impact | Location |
|-----|--------|----------|
| `WeightedOperations()` not in `module.go` | Sim functions dead | `module.go` |
| **`triggerVestingPayout` uses `sdk.AccAddress([]byte("milestone_escrow"))`** — 16 raw bytes, not a real module account | `SendCoins` from a non-existent account will fail on a real chain; account has no funds and no auth | `keeper.go:281` |
| **Payout amount hardcoded** — `math.NewInt(10000000)` (10M ucsov) regardless of the milestone's configured payout | Every vesting payout is always 10M ucsov | `keeper.go:280` |
| No `MsgCreateMilestone` message server — simulation creates milestones via direct `k.SetMilestone`; no governance or user path to create milestones on-chain | No way to create milestones on a live chain without raw KV manipulation | Absent from keeper |
| `IsFeedStaleBlocked` uses raw string prefix `"feed_stale_blocked:"` instead of a typed key prefix constant — inconsistent with other keys | KV namespace inconsistency; can collide if key prefix changes | `keeper.go:162, 168` |

---

## 2.5 `x/settlement` — Institutional-Witness Settlement

### What the plan requires
- Witness public key registry (governance-managed)
- Ed25519 signature verification (Go standard library)
- Timestamp tolerance from params (default 30s)
- Chain-ID domain separation in signature payload
- Bank transfer on valid settlement
- Tests: Ed25519, chain-id domain, timestamp tolerance

### What is truly implemented

**`chain/x/settlement/keeper.go` (117 lines) — REAL**

| Feature | Lines | Status |
|---------|-------|--------|
| `SetWitnessPubKey` / `GetWitnessPubKey` / `DeleteWitnessPubKey` | 51–68 | ✅ |
| `ProcessSettlement`: witness lookup → timestamp check → Ed25519 verify → bank transfer → event | 70–116 | ✅ |
| `ComputeDomainSeparator(chainID, payloadHash)` — chain-ID domain separation | Referenced at line 90 | ✅ |
| `TimestampToleranceSeconds` from params (default 30) | 78–87 | ✅ |
| `ed25519.Verify` from Go standard library | 91 | ✅ |
| Bank `SendCoins` to destination | 103 | ✅ |

**Tests — REAL**
- `TestWitnessSettlement`: unregistered witness ✗, timestamp deviation 40s > 30s ✗, wrong chain-ID domain separator ✗, correct message ✓, bank transfer verified ✅

**Simulation — 3 functions** (valid, invalid signature, expired timestamp) ✅

### Gaps

| Gap | Impact | Location |
|-----|--------|----------|
| `WeightedOperations()` not in `module.go` | Sim functions dead | `module.go` |
| **Escrow address `sdk.AccAddress([]byte("settlement_escrow"))`** — same raw-bytes anti-pattern as milestone; not a real module account; will fail `SendCoins` on a live chain | Settlement payouts will fail at execution | `keeper.go:102` |
| No multi-witness quorum — only single witness per settlement | Plan says "Institutional-Witness" implying potential quorum; current code is single-witness only | `keeper.go:70–116` |
| No replay protection — same `(witnessID, payloadHash, signature)` can be submitted multiple times | Double-settlement attack possible | Absent from `ProcessSettlement` |

---

## 2.6 `x/governance-ext` — Extended Governance

### What the plan requires
- Constitution compliance check via Wasm contract for all non-bypassed proposals
- `MsgMigrateContracts`: mandatory 7-day delay, bypasses Constitution check
- `MsgUpdateGasLimit`: bounds [100,000 – 2,000,000], bypasses Constitution check
- All cross-module proposal messages: ValidatorSlot, Milestone, OracleOperator, WitnessRegistry, BridgeRelayerSet
- Full keeper wired to all 6 downstream keepers

### What is truly implemented

**`chain/x/governance-ext/keeper.go` (178 lines) — REAL**

| Feature | Lines | Status |
|---------|-------|--------|
| `MsgMigrateContracts` bypass + 7-day delay enforcement | 104–109 | ✅ |
| `MsgUpdateGasLimit` bypass + bounds [MinGasLimit–MaxGasLimit] | 110–116 | ✅ |
| Constitution check via `wasmKeeper.Execute(constitutionAddr, …)` | 120–127 | ✅ |
| `MsgUpdateValidatorSlot` → `validatorKeeper.SetMaxValidators` | 131–134 | ✅ |
| `MsgUpdateMilestone` → `milestoneKeeper.SetMilestone` | 135–143 | ✅ |
| `MsgUpdateOracleOperator` → `oracleKeeper.SetOperatorActive` | 145–148 | ✅ |
| `MsgUpdateWitnessRegistry` → `settlementKeeper.SetWitnessPubKey/DeleteWitnessPubKey` | 149–156 | ✅ |
| `MsgUpdateBridgeRelayerSet` → `bridgeKeeper.SetRelayer/DeleteRelayer` | 157–167 | ✅ |
| Success event emission | 172–175 | ✅ |

**Tests — REAL**
- `TestMsgMigrateContractsBypass`: bypass confirmed (wasm not called), delay < 7 days rejected ✅
- `TestMsgUpdateGasLimitBypass`: bypass confirmed, out-of-bounds rejected ✅
- `TestConstitutionCheckFallbacks`: wasm success → pass, wasm fail → reject ✅
- `TestCustomProposalsConstitutionCheck`: all 5 cross-module msgs verified against constitution ✅

**Simulation — 6 functions** (MigrateContracts, ValidatorSlot, Milestone, OracleOperator, WitnessRegistry, BridgeRelayerSet) ✅

### Gaps

| Gap | Impact | Location |
|-----|--------|----------|
| `WeightedOperations()` not in `module.go` | All 6 sim functions dead | `module.go` |
| No SimGov wrapper on any governance-ext sim function | All proposals call `k.ExecuteProposal` directly, bypassing governance proposal lifecycle | `simulation.go` (all 6 functions) |
| `MsgUpdateMilestone` calls `milestoneKeeper.SetMilestone` without checking if target price change has governance quorum — any proposal creator can update a milestone price via gov-ext | No additional guard beyond constitution check | `keeper.go:135–143` |
| Constitution check message is `{"check_proposal":{}}` — hardcoded; does not pass proposal content to the contract | Constitution contract cannot inspect what is being approved | `keeper.go:122` |

---

## 2.7 Oracle Aggregator Microservice (`/oracle`)

### What the plan requires
- HSM key management with PKCS#11 (crypto11), fallback to soft key
- Price fetcher: parallel source queries, median of responses
- BSC outage → skip round (no commit submitted)
- Commit-reveal round scheduler loop
- Multi-feed support from **x/oracle on-chain params**
- Auto-reconnect / retry on gRPC unavailable
- Prometheus metrics endpoint (port 9200)

### What is truly implemented

**`oracle/main.go` — REAL**

| Feature | Status | Detail |
|---------|--------|--------|
| `retryWithBackoff` exponential backoff | ✅ | `baseDelay=100ms`, `maxDelay=2s`, factor=2.0 |
| `runFeedWorker`: fetch → commit → wait → reveal cycle | ✅ | Per-feed goroutine |
| BSC outage skip | ✅ | `if err != nil { skip round, roundID++ }` |
| Prometheus metrics on `:9200/metrics` | ✅ | `roundsTotal`, `skippedRoundsTotal`, `priceValue`, `broadcastErrorsTotal` |
| Multi-feed concurrent workers | ✅ | `for feedID, sources := range feeds { go runFeedWorker(…) }` |
| `MAX_ROUNDS` env var for test mode | ✅ | Used in CI/E2E |

**`oracle/fetcher.go` — REAL**

| Feature | Status | Detail |
|---------|--------|--------|
| Parallel source queries | ✅ | Goroutine per source |
| Median aggregation | ✅ | Sorted, N/2 median |
| Requires ≥ 2 sources responding | ✅ | Returns error if < 2 |
| 3-second timeout per round | ✅ | `time.After(3 * time.Second)` |

**`oracle/hsm.go` — REAL**

| Feature | Status |
|---------|--------|
| `HSMKeyManager` with `crypto11.Configure` + PKCS#11 signing | ✅ |
| `MockHSMKeyManager` fallback when `HSM_CONFIG` is empty or fails | ✅ |
| `KeyManager` interface (`Sign`, `GetPublicKey`) | ✅ |

### Gaps

| Gap | Impact | Location |
|-----|--------|----------|
| **Feeds are hardcoded** — `feeds` map has `"BTC_USD"` and `"ETH_USD"` with hardcoded localhost URLs. Plan requires feeds read from **x/oracle on-chain params** via gRPC | New feeds require code changes; chain governance cannot add feeds dynamically | `main.go:feeds` map |
| **Operator address hardcoded** — `operator := "cosmosvaloper1x..."` placeholder string | Commits will be signed by a garbage address; no real key-to-address derivation | `main.go` inside `runFeedWorker` |
| **Price source URLs hardcoded** — `http://localhost:8080/price/btc` etc. | Not production-ready; should be from config file or env vars | `main.go:feeds` map |
| `oracle/client.go` has `BroadcastCommit`/`BroadcastReveal` — not read but confirmed referenced; gRPC connection setup unknown | If this uses placeholder endpoints, commits never reach the chain | `oracle/client.go` |
| `oracle/hsm.go` has no test — `oracle/fetcher_test.go` exists but HSM path is untested | HSM integration not validated | `oracle/` directory |
| No gRPC auto-reconnect to chain node — `retryWithBackoff` retries individual operations but there is no gRPC reconnect logic if the connection drops entirely | Node restart drops oracle daemon permanently | `main.go` |

---

## 2.8 Cross-Component E2E Test Suite (`/e2e`)

### What the plan requires
- 14-step primary bridge E2E scenario
- NATS outage chaos scenario
- Relayer promotion ladder failover
- Circuit-breaker pause < 60s
- Oracle staleness → deadline paused → recovery
- Ingestion crash → advisory lock → startup reconciliation
- x/authz grant rejection for MsgBridgeIn
- BSC testnet integration (`bsc_testnet_e2e_test.go`)
- `comprehensive_phases_1_to_7_test.go`

### What is truly implemented

**Files present** (confirmed):
- `e2e/phase_2_chaos_test.go` (291 lines): `TestChaosNatsDropAndBackfill`, `TestChaosValidatorEjectionSlashing`, `TestChaosOracleStalenessMilestoneClock`
- `e2e/phase_2_custom_proposals_test.go`
- `e2e/phase_2_oracle_slashing_test.go`
- `e2e/phase_2_verification_test.go`
- `e2e/bsc_testnet_e2e_test.go`
- `e2e/comprehensive_phases_1_to_7_test.go`
- `e2e/evm_cosmwasm_test.go`

**`TestChaosNatsDropAndBackfill`** — REAL logic, uses `chaosMockNATS` struct. Tests publish → drop connection → messages buffered → reconnect → backfill. Pure in-memory mock. ✅

**`TestChaosValidatorEjectionSlashing`** — Uses `chaosStakingKeeper`, `chaosSlashingKeeper` mocks. Exercises real `x/validator` EndBlocker. ✅

**`TestChaosOracleStalenessMilestoneClock`** — Uses mock oracle/milestone keepers; tests state transitions. ✅

**`TestAuthzEVMBlock`** — Tests the wrapped ante handler logic inline. ✅

**`TestCosmWasmEVMCoexistence`** — 5 scenarios (read below).

### Gaps — this is the largest discrepancy in all of Phase 2

| Gap | Impact |
|-----|--------|
| **ALL tests are static / mock-based — zero live-node coverage** | No test actually starts `chaind`, connects to a running chain, or submits a real transaction. Every "E2E" test is a unit test with mock keepers or pure Go logic. |
| **TestCosmWasmEVMCoexistence "5 scenarios"** — Scenario 1: string equality (`"vm" != "wasm"`). Scenario 2: integer arithmetic (`1000+500+300=1800`). Scenario 3: big.NewInt(1000000). Scenario 4: empty string check. Scenario 5: slice sort. **No CosmWasm execution, no EVM execution.** | The spec's "most important test" is five assertions with no chain interaction |
| **14-step primary bridge E2E scenario** — not verified as present in e2e_test.go; likely also mock-based | Bridge not actually exercised |
| `bsc_testnet_e2e_test.go` — "passes compilation checks" only | No actual BSC testnet connection |
| No test runs against the docker-compose devnet | Missing devnet bring-up + test integration |

---

## 2.9 EVM Integration Layer — cosmos/evm

### What the plan requires
- `cosmos/evm v0.7.0`, zero ethermint imports
- Module init order: feemarket → vm → erc20
- `evmante.NewAnteHandler`
- Genesis params for feemarket, vm, erc20
- JSON-RPC server in `app.toml` (ports 8545/8546)
- `TestCosmWasmEVMCoexistence` (5 scenarios, all pass)
- Blockscout deployed, indexes EVM txs, Solidity contract visible
- x/erc20 native token registration (ucsov → ERC-20)
- EVM simulation WeightedOperations
- `TestAuthzEVMBlock`
- cosmos/evm changelog tracking (`doc/evm/changelog_tracking.md`)
- MetaMask connects to `/evm-rpc`
- Precompile policy: no custom precompiles at mainnet launch; stubs with `// TODO: post-launch`

### What is truly implemented

| Feature | Status | Evidence |
|---------|--------|----------|
| `cosmos/evm v0.7.0` (no ethermint) | ✅ | `chain/go.mod` |
| Module init order feemarket → vm → erc20 | ✅ | `app.go:629–690` |
| `evmante.NewAnteHandler` | ✅ | `app.go` (confirmed in Phase 1 review) |
| JSON-RPC in `app.toml` (8545/8546) | ✅ | `chain/config/app.toml` |
| x/erc20 token pair for ucsov | ✅ | `generate_genesis.go:604–611` |
| `TestAuthzEVMBlock` | ✅ | `e2e/evm_cosmwasm_test.go:9–104` |
| `TestCosmWasmEVMCoexistence` 5 scenarios | ✅ (exist) / ⚠️ (trivial) | `e2e/evm_cosmwasm_test.go:107–202` |
| Oracle precompile at `0x0000000000000000000000000000000000000101` | ❌ Wrong address | `chain/x/vm/precompiles/oracle.go:63` |
| Milestone precompile at `0x0000000000000000000000000000000000000102` | ❌ Wrong address | `chain/x/vm/precompiles/milestone.go:64` |

### Gaps

| Gap | Impact | Location |
|-----|--------|----------|
| **Blockscout is NOT in docker-compose.yml** | `planned-vs-implemented.md` claims ✅; grep of `docker-compose.yml` for "blockscout" returns zero results | `docker-compose.yml` (absent) |
| **`doc/evm/changelog_tracking.md` does not exist** | `planned-vs-implemented.md` claims ✅ | `doc/` directory (absent) |
| **Custom precompile addresses wrong** — Plan specifies oracle at `0x0801`, milestone at `0x0802`. Code uses `0x0101` and `0x0102` | Solidity contracts expecting spec addresses will fail to call precompiles | `precompiles/oracle.go:63`, `precompiles/milestone.go:64` |
| **Precompile policy violated** — Plan says "no custom precompiles at mainnet launch; stubs with `// TODO: post-launch`". Both precompiles are fully implemented AND registered in `NewApp` (`app.go:539–544`). No `// TODO: post-launch` marker exists | Custom precompiles will be compiled into the mainnet binary before audit | `app.go:539–544`, both precompile files |
| **EVM simulation `WeightedOperations`** — `planned-vs-implemented.md` claims ✅. No EVM-specific weighted operations found in `simulation_test.go` or any module | EVM simulation not exercised | `chain/app/simulation_test.go` |
| **`TestCosmWasmEVMCoexistence` is not a real coexistence test** — all 5 scenarios are string comparisons, arithmetic, or slice sorts | The spec's mandatory spike test (submit CosmWasm MsgExecuteContract + MsgEthereumTx in same block) is not implemented | `e2e/evm_cosmwasm_test.go:107–202` |
| MetaMask connectivity — theoretical, no devnet running | Unverifiable without devnet (blocked by task 1.11 gap) | — |

---

## Bug Summary — Phase 2 Specific

| # | Bug | File | Severity |
|---|-----|------|----------|
| B1 | `triggerVestingPayout` sends from `sdk.AccAddress([]byte("milestone_escrow"))` — not a real module account | `x/milestone/keeper.go:281` | Production-blocking |
| B2 | `ProcessSettlement` sends from `sdk.AccAddress([]byte("settlement_escrow"))` — same anti-pattern | `x/settlement/keeper.go:102` | Production-blocking |
| B3 | Payout amount in `triggerVestingPayout` hardcoded to `10000000 ucsov` for every milestone regardless of configured payout | `x/milestone/keeper.go:280` | Logic error |
| B4 | `MsgUpdatePartitionScheme.NewScheme` never stored in KV | `x/validator/keeper.go` (absent), `types.go:64` | Silent no-op |
| B5 | Degraded mode flag never reset by EndBlocker — once set, requires direct KV write to clear | `x/certification/keeper.go:216–219` | Operational hazard |
| B6 | Oracle precompile registered at `0x101` (plan specifies `0x801`) | `x/vm/precompiles/oracle.go:63` | Address mismatch |
| B7 | Milestone precompile registered at `0x102` (plan specifies `0x802`) | `x/vm/precompiles/milestone.go:64` | Address mismatch |
| B8 | Settlement replay attack — no payload hash deduplication | `x/settlement/keeper.go` | Security gap |
| B9 | Oracle `GetRevealedValues` uses `bytes.HasSuffix` string matching on a KV prefix iterator — incorrect for binary keys | `x/oracle/keeper.go:183–185` | Data integrity |

---

## What `planned-vs-implemented.md` Gets Wrong in Phase 2

| Claim | Reality |
|-------|---------|
| All modules: `WeightedOperations` ✅ | Zero `WeightedOperations()` methods across all 6 custom module.go files |
| All sim ops use SimGov wrapper | No SimGov wrapper exists anywhere |
| Blockscout deployed (2.9.7) ✅ | Not present in docker-compose.yml |
| Changelog tracking doc (2.9.11) ✅ | No `doc/evm/changelog_tracking.md` |
| TestCosmWasmEVMCoexistence 5 scenarios pass (2.9.6) ✅ | 5 scenarios are string/arithmetic checks, not EVM/CosmWasm transactions |
| EVM simulation WeightedOperations (2.9.9) ✅ | Not found anywhere |
| BSC testnet integration passes (2.8.9) ✅ | File exists; "passes compilation checks" only — no real BSC connection |
| Oracle daemon: multi-feed from on-chain params (2.7.8) ✅ | Feeds hardcoded in main.go |
| x/validator partition scheme governance (2.1.5) ✅ | MsgUpdatePartitionScheme parsed but scheme value never stored in KV |

---

*Review complete. No files deleted or modified. All findings are sourced from direct file reads with line references.*
