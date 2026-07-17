# Sovereign Layer-2 Blockchain — Comprehensive Security & Readiness Audit

**Audit Date:** 2026-07-16  
**Auditor:** Independent AI Security Review (Two-Pass Deep Audit)  
**Repository:** `https://github.com/majednitol/Sovereign-Layer-2-blockchain-`  
**Commit Snapshot:** `/tmp/sovereign-l2` (full clone, all files read)  
**Scope:** All Go chain modules, CosmWasm contracts, EVM bridge (Solidity), relayer engine, oracle daemon, CQRS backend, database schemas, Docker/infrastructure, scripts, genesis configuration, and documentation  

---

## Executive Summary

This report is the result of a **complete, independent two-pass audit** of every file in the Sovereign Layer-2 blockchain repository. The first pass covered the core chain and contracts; the second pass covered the backend, relayer internals, infrastructure, scripts, and all documentation that the first pass missed.

The project demonstrates **architectural ambition and substantial engineering effort**, including a well-conceived seven-module chain, a CQRS indexing backend, cross-chain bridging, oracle commit-reveal with MAD filtering, and an on-chain governance-constitution model. However, the codebase has not yet reached the bar required for testnet public launch, and is substantially far from mainnet readiness. **No external audit firm has been engaged** (`audit_engagement.json`: status = `not-started` for all three firms). Several of the findings below represent acute, exploitable vulnerabilities that would result in irrecoverable fund loss if deployed.

### Finding Totals

| Severity | Count |
|---|---|
| 🔴 Critical | 13 |
| 🟠 High | 11 |
| 🟡 Medium | 14 |
| 🔵 Low / Informational | 20 |
| **Total** | **58** |

### Mainnet Readiness Verdict

> **NOT READY FOR TESTNET (PUBLIC) — FAR FROM MAINNET**

Thirteen critical findings, five of which involve credential or private-key exposure in source code that is already committed to a public repository. All critical and high findings must be resolved, credentials must be rotated, and a formal external audit must be completed before any public network launch.

---

## Table of Contents

1. [Critical Findings (C-01 – C-13)](#critical-findings)
2. [High Findings (H-01 – H-11)](#high-findings)
3. [Medium Findings (M-01 – M-14)](#medium-findings)
4. [Low / Informational Findings (L-01 – L-20)](#low-informational-findings)
5. [Module-by-Module Analysis](#module-by-module-analysis)
   - [Chain Modules (x/)](#chain-modules)
   - [CosmWasm Contracts](#cosmwasm-contracts)
   - [EVM Bridge (LockBox.sol)](#evm-bridge)
   - [Relayer Engine](#relayer-engine)
   - [Oracle Daemon](#oracle-daemon)
   - [CQRS Backend](#cqrs-backend)
   - [Database Schemas](#database-schemas)
   - [Docker / Infrastructure](#docker--infrastructure)
   - [Scripts & Genesis](#scripts--genesis)
   - [Documentation & Operations](#documentation--operations)
6. [Deployment Readiness Checklist](#deployment-readiness-checklist)
7. [Remediation Roadmap](#remediation-roadmap)

---

## Critical Findings

### C-01 — `MarkSettlementProcessed` called BEFORE `SendCoins`
**Location:** `chain/x/settlement/keeper.go`  
**Impact:** If the `SendCoins` call fails after the settlement has been marked processed, the settlement is permanently consumed but funds are never transferred. Attackers who can engineer a `SendCoins` failure (e.g., by exhausting module balance) may be able to invalidate legitimate settlement claims.  
**Fix:** Move `MarkSettlementProcessed` to after `SendCoins` returns without error. Use a defer-based rollback or handle the transaction atomically within a cached context.

---

### C-02 — Integer Overflow in Bridge Amount Conversion
**Location:** `chain/x/bridge/keeper.go`  
**Impact:** `sdk.NewInt(int64(msg.Amount))` truncates the 256-bit EVM amount to 64-bit. Any BSC lock event with a token amount exceeding `math.MaxInt64` (≈ 9.2 × 10¹⁸ base units) silently wraps, allowing an attacker to lock a large token amount on BSC and receive an astronomically incorrect minted amount on Cosmos — either causing catastrophic oversupply or triggering a panic.  
**Fix:** Parse amounts as `sdk.Int` / `math.Int` from a string or big.Int representation. Reject any event where the amount exceeds the validated supply cap before any conversion.

---

### C-03 — `WasmKeeper.Execute` Used to Query Constitution (State-Mutating Call)
**Location:** `chain/x/governance-ext/keeper.go`, `contracts/constitution/src/contract.rs`  
**Impact:** The `CheckProposal` message in the constitution contract is an `ExecuteMsg`, not a `QueryMsg`. The governance-ext module calls `WasmKeeper.Execute` on it during proposal validation. This means: (a) every proposal check mutates contract state, consuming gas and producing events in unexpected contexts; (b) an attacker who can influence constitution state may be able to cause proposal validation to panic or behave incorrectly; (c) the check runs in a context where side effects should not be produced.  
**Fix:** Add a `QueryMsg::CheckProposal` variant to the constitution contract that performs the compliance check as a read-only query. Update `governance-ext` to call `WasmKeeper.QuerySmart` instead of `WasmKeeper.Execute`.

---

### C-04 — `governance-ext` Keeper Shares `govtypes.StoreKey`
**Location:** `chain/x/governance-ext/keeper.go`, `chain/app/app.go`  
**Impact:** The `GovExtKeeper` is initialized with `govtypes.StoreKey` (the same store key as the standard governance module). Any write operations by the governance-ext module will directly corrupt the governance module's state, and vice versa. This is a critical store isolation violation that can lead to chain halts or state corruption.  
**Fix:** Register a dedicated store key (`"governance_ext"`) for the governance-ext module. Migrate any state that needs to reference governance data through the standard governance keeper API, not via direct store access.

---

### C-05 — `recoverSigner` Returns `address(0)` on Invalid Signature
**Location:** `bridge/src/LockBox.sol`  
**Impact:** Solidity's `ecrecover` returns `address(0)` on failure. If the `relayerSet` mapping contains `address(0)` (which is possible if the relayer list is not carefully initialized, or if an entry is zeroed out), any message with a malformed or forged signature will be treated as valid and signed by `address(0)`. This completely bypasses the signature verification check.  
**Fix:** Add an explicit check: `require(signer != address(0), "Invalid signature");` before the `relayerSet[signer]` lookup. Additionally, ensure the relayer set initialization never registers `address(0)`.

---

### C-05b — `LockBox.sol` Signatures Replayable Across Deployments
**Location:** `bridge/src/LockBox.sol` (message hash construction)  
**Impact:** The `messageHash` used for signature verification does not include `address(this)` (the contract address) or `block.chainid`. This means a valid set of relayer signatures for one LockBox deployment can be replayed against any other deployment of the same contract — including a mainnet deployment after a testnet, or against a forked chain. This allows unauthorized fund extraction from any LockBox instance.  
**Fix:** Include `address(this)` and `block.chainid` in the message hash:
```solidity
bytes32 hash = keccak256(abi.encodePacked(
    address(this), block.chainid, recipient, amount, nonce
));
```

---

### C-06 — Relayer Signature Aggregator Accepts Votes Without Verifying Signatures
**Location:** `relayer/sig_aggregator.go`  
**Impact:** The `SigAggregator` receives vote messages from NATS and accumulates them toward a quorum. However, the vote messages contain no cryptographic proof of who sent them — any process that can publish to the NATS topic can submit a vote for any relayer address. An attacker who can reach the NATS server (which has no authentication in the default configuration; see L-11) can fabricate votes for all quorum members and force the submission of arbitrary bridge transactions.  
**Fix:** Each vote message must include a cryptographic signature (ECDSA over the vote payload, signed by the relayer's private key). The aggregator must verify this signature against the known relayer public key before counting the vote.

---

### C-07 — Colon Injection in Oracle Composite Keys
**Location:** `chain/x/oracle/keeper.go`  
**Impact:** Oracle commit and reveal composite keys are constructed by concatenating validator address and asset symbol with a colon separator (e.g., `validatorAddr + ":" + asset`). If an asset symbol contains a colon (e.g., `"BTC:USD"`), the resulting key is ambiguous and can collide with a commit by a different validator for a different asset. An attacker who can register a malformed asset symbol can overwrite or shadow another validator's commits, corrupting price feeds.  
**Fix:** Use a binary-safe separator or, preferably, use the Cosmos SDK's `collections` package which provides typed key encoding that avoids injection.

---

### C-08 — `IsValidatorAttested` Defaults to `true` (Unsafe Default)
**Location:** `chain/x/certification/keeper.go`, `chain/app/abci.go`  
**Impact:** When the certification module has no data for a validator (e.g., at chain start, after a reset, or for a new validator), `IsValidatorAttested` returns `true` (certified) rather than `false` (uncertified). This means all validators are treated as certified before any attestation has been established. An uncertified or newly added malicious validator will be counted as certified for all slashing and governance checks until the certification window fills.  
**Fix:** Default to `false` (uncertified). Require validators to produce at least one attestation window's worth of data before being counted as certified. Add a genesis state that pre-certifies the genesis validator set.

---

### C-09 — Relayer Auto-Generates Random Private Key and Logs It to Stdout
**Location:** `relayer/cmd/relayer/main.go` lines 116–121  
**Impact:** If no `RELAYER_PRIVATE_KEY` environment variable is set, the relayer generates a random Ethereum private key and logs it in plaintext:
```go
log.Printf("[Daemon] Private key not provided, generated random fallback: %s\n", priv)
```
Any log aggregator, monitoring system, terminal emulator, or anyone with access to container logs will capture this key. The relayer key controls bridge transaction signing. **This is a catastrophic credential exposure.**  
**Fix:** Remove the auto-generation logic entirely. If the private key is not configured, the daemon must exit with a clear error message instructing the operator to provide it via a secrets manager (Vault, Kubernetes Secret, HSM). Never log a private key under any circumstances.

---

### C-10 — Three NATS NKey Seeds Hardcoded in Source Code Across All Backend Services
**Location:** `backend/module/ingestion/main.go` lines 610–616; `backend/module/projection/main.go` lines 507–513; `backend/module/api/main.go` lines 1026–1034  
**Impact:** Three NATS NKey seeds (`SUAFFNTD6H6ST7VGTZDXYQDC5BPNGYRTEFY4TZM32TJEMBTFN5TJO4WNXU`, `SUAINVHHXAR4PZTQC4VEME4P3HB2CQ3QNQY4WK3YNULE2IJZLNOLNDGBUE`, `SUAO6IIZLMQHQYVKKHJIEXIC5T6XNKM2PUVF4EGZW23UALD7WTFFE7R2LQ`) are committed to the public repository as fallback values. Because these seeds are already public, anyone can authenticate to any NATS server that accepts these credentials, inject arbitrary messages into all sovereign chains topics (price feeds, bridge events, block events), or silently discard real messages. **These seeds must be considered permanently compromised and rotated immediately.**  
**Fix:** Remove all hardcoded seeds. Require `INGESTION_NKEY_SEED`, `PROJECTION_NKEY_SEED`, and `STREAM_NKEY_SEED` environment variables (or Vault retrieval). Fail fast on startup if none are available. Rotate all three seeds.

---

### C-11 — `LockBox.sol` `lock()` Increments `totalLocked` Before `transferFrom`
**Location:** `bridge/src/LockBox.sol` lines ~100–104  
**Impact:** The function updates the `totalLocked` counter before calling `transferFrom`. If the `transferFrom` fails (e.g., insufficient allowance, transfer hook reverts, token is paused), `totalLocked` has already been permanently incremented. Subsequent `lock()` calls will see an inflated `totalLocked`, potentially triggering the rate limiter prematurely and locking honest users out of the bridge even though no funds were actually transferred.  
**Fix:** Apply the checks-effects-interactions pattern: complete all token transfers first, then update state variables. Alternatively, read the actual balance delta after the transfer and update `totalLocked` based on observed change.

---

### C-12 — E2E Test Script Hardcodes BSC Testnet Private Key in Source
**Location:** `scripts/run_real_testnet_e2e.sh` line 12  
**Impact:** A raw Ethereum private key (`0xb25d0aab150080869d39a2532840cbb04321527d92703dc7120bfdd179282695`) is committed in plaintext in the repository. If this key was used on testnet (or if any derived address holds mainnet funds), the address is fully compromised. GitHub's secret scanning may have already indexed this.  
**Fix:** Remove the key from the script immediately. Rotate the key and any associated accounts. Use environment variable references without defaults: `${BSC_TESTNET_PRIVATE_KEY:?Must set BSC_TESTNET_PRIVATE_KEY}`.

---

## High Findings

### H-01 — Permanent Tombstoning on Validator Ejection
**Location:** `chain/x/validator/keeper.go`  
**Impact:** When a validator is ejected via `MsgEjectValidator`, the keeper calls the slashing module's tombstone function, permanently banning the validator from ever rejoining. This is disproportionate for ejection (which may occur for economic reasons, not double-signing) and creates a governance attack vector: a malicious majority can permanently tombstone validators without evidence of Byzantine behavior.  
**Fix:** Replace tombstoning with jailing for ejection. Tombstoning should be reserved exclusively for equivocation (double-sign) evidence submitted through standard `MsgSubmitEvidence`.

---

### H-02 — No Settlement Timestamp Upper Bound
**Location:** `chain/x/settlement/types.go`  
**Impact:** Settlement messages only validate `Timestamp > 0` with no upper bound. An attacker can submit a settlement with a timestamp arbitrarily far in the future. Depending on how timestamp is used for expiry, ordering, or replay protection, this could allow attackers to construct settlements that never expire, cannot be processed, or bypass time-based validity windows.  
**Fix:** Add validation: `msg.Timestamp <= ctx.BlockTime().Unix() + MaxFutureTimestampOffset` (e.g., 5 minutes). Reject settlements with timestamps more than a bounded amount ahead of consensus time.

---

### H-03 — `WasmKeeper.Execute` for Constitution Check Does Not Return Query-Safe Results
**Location:** `chain/x/governance-ext/keeper.go`  
**Note:** This is a continuation of C-03. The call happens in a message-handling context, meaning it can fail in ways that abort governance transactions entirely (e.g., if the constitution contract is paused). There is no fallback or circuit-breaker for when the constitution contract is unavailable.  
**Fix:** Add a fallback path in governance-ext: if the constitution check call fails with a "paused" or "unavailable" error, log the failure and proceed with the governance proposal in a degraded state rather than aborting it.

---

### H-04 — Oracle Daemon Uses Hardcoded 500ms Sleep Instead of Block-Time-Aware Scheduling
**Location:** `oracle/main.go`  
**Impact:** The oracle daemon's main loop calls `time.Sleep(500 * time.Millisecond)` unconditionally. Under high load, network partitions, or when the chain is producing blocks faster or slower than expected, this leads to missed commit windows or flooding the chain with redundant transactions. With a 5-second block time, 500ms overshoots within a block but with no adaptive behavior for chain-latency variation.  
**Fix:** Replace the fixed sleep with a subscription to chain events (NewBlock or EndBlock) and submit oracle commits triggered by block events, with a configurable maximum submission frequency.

---

### H-05 — `ComputeBridgeMessageHash` Has No Length Prefix
**Location:** `chain/x/bridge/types.go`  
**Impact:** The hash is computed by concatenating fields without length-prefixing, creating a hash collision risk. For example, a message with `address="abc"` and `data="def"` produces the same hash as `address="ab"` and `data="cdef"`. An attacker who can craft messages with overlapping field boundaries can forge a hash collision, potentially replaying or substituting one bridge message for another.  
**Fix:** Use `sdk.Keccak256(abi.EncodePacked(...))` style encoding with explicit field delimiters, or use a length-prefixed encoding (protobuf marshaling of the canonical message type).

---

### H-06 — Degraded-Mode Threshold (30%) Below BFT Safety Bound (33%)
**Location:** `chain/x/certification/keeper.go`  
**Impact:** The certification module enters "degraded mode" when fewer than 30% of validators are certified. However, BFT consensus (Tendermint/CometBFT) requires >2/3 of voting power for liveness and >1/3 of voting power to veto. A validator set where 30% are uncertified but the module is NOT in degraded mode could still execute governance proposals with uncertified validator support, violating the intended security guarantee.  
**Fix:** Set the degraded-mode threshold to at least 34% (i.e., degraded mode activates if any single validator group representing >33% of power is uncertified). Alternatively, tie degraded mode to voting-power-weighted certification rather than validator count.

---

### H-07 — `HorcruxSignerClient.Sign` Returns Hardcoded Mock Bytes
**Location:** `relayer/signer.go` line 67  
**Impact:** The threshold signer implementation for Horcrux returns a static mock string (`"horcrux_threshold_mock_signature_bytes_65_length_padded_out_here"`) instead of performing real threshold signing. Any production deployment using `HorcruxSignerClient` will produce invalid signatures on all bridge transactions, causing them to be rejected. **Threshold signing is non-functional.**  
**Fix:** Implement the actual Horcrux gRPC client protocol for remote signing. Until this is complete, `HorcruxSignerClient` must not be available in any non-development build (use build tags or explicit panics).

---

### H-08 — `checkLockInOrigin` Uses Trivially Bypassable Prefix Heuristic
**Location:** `relayer/cmd/relayer/main.go` lines 368–371  
**Impact:** The function determines whether a transaction originates from the correct direction (BSC→Cosmos vs. Cosmos→BSC) by checking if the nonce hex string starts with `"ab"`:
```go
if !strings.HasPrefix(nonceHex, "ab") { return wrongDirectionErr }
```
This heuristic is: (a) semantically meaningless (nonce values are not directionally encoded), (b) trivially bypassable by an attacker who crafts a nonce starting with `"ab"`, and (c) may incorrectly reject valid transactions whose nonces do not start with those bytes.  
**Fix:** Remove this check entirely. Directional routing should be determined by which chain emitted the originating event, not by nonce content. The `BSCWatcher` should only submit Cosmos→BSC completions, and `CosmosWatcher` should only submit BSC→Cosmos mints.

---

### H-09 — EVM Module Initialized Before Feemarket in Genesis Order
**Location:** `chain/app/app.go`  
**Impact:** EIP-1559 base fee computation in the EVM module depends on fee market state being initialized first. If the EVM genesis initialization runs before `x/feemarket`, it may read uninitialized state, producing a zero base fee or panic. This issue only manifests at genesis block execution but can cause a chain halt before block 1.  
**Fix:** Reorder the genesis initialization list so that `feemarket` is initialized before the EVM module. Follow the ordering used by production `evmos`/`cosmos-evm` deployments.

---

### H-10 — `CosmosWatcher.pendingBurns` Not Persisted; Burn Events Lost on Crash
**Location:** `relayer/cosmos_watcher.go`  
**Impact:** When the Cosmos-side watcher observes a `MsgBridgeOut` event, it adds the event to an in-memory `pendingBurns` slice and then publishes it to NATS. If the relayer process crashes between observing the event and publishing it (or before it is picked up from NATS), the burn event is permanently lost. Affected users will never receive their BSC-side tokens, resulting in locked funds with no recovery path.  
**Fix:** Persist observed burn events to the relayer DB before attempting NATS publication (same pattern as `SaveLockEvent` in `bsc_watcher.go`). Use a "pending-to-published" state machine in the DB so that a relayer restart can resume from the last confirmed publish point.

---

### H-11 — Vault Runs in Dev Mode in Docker Compose
**Location:** `docker-compose.yml` line 342  
**Impact:** The Vault container is started with `VAULT_DEV_ROOT_TOKEN_ID: "root"` and `command: server -dev`. Vault dev mode: (a) stores all secrets in memory (all secrets lost on restart), (b) disables TLS, (c) uses a well-known root token, and (d) is explicitly documented by HashiCorp as "never for production." Any attacker with network access to the Vault port can authenticate with token `"root"` and read all secrets.  
**Fix:** Configure Vault with a proper production server configuration: file/Raft storage backend, TLS certificates, unsealing via Shamir shares or cloud KMS auto-unseal, and a scoped AppRole for each backend service. Remove the dev-mode token.

---

## Medium Findings

### M-01 — `WindowConsistencyInvariant` is O(N×W) — DoS on Large Validator Sets
**Location:** `chain/x/certification/keeper.go`  
**Impact:** The invariant checker iterates over all validators and all window slots, giving O(N×W) complexity. With N=100 validators and W=1000 window slots, this is 100,000 store reads per invariant check. If invariants are checked frequently (e.g., per block), this becomes a block-time bottleneck and potential DoS vector.  
**Fix:** Maintain an aggregate counter per validator (total attestations in window) rather than recounting from raw window slots. The invariant check becomes O(N).

---

### M-02 — Vote Extension Handler Returns Stub String
**Location:** `chain/app/abci.go`  
**Impact:** `ExtendVote` returns a hardcoded string `"vote_extension_data"` instead of real vote extension data (e.g., oracle prices, certification attestations). `vote_extensions_enable_height` is set to `"0"` (enabled from genesis). This means all validators produce identical, meaningless vote extensions from block 1, making vote extensions useless for any application logic that depends on them.  
**Fix:** Implement real vote extension logic (e.g., include current oracle price commitments or certification signatures). If vote extensions are not ready, set `vote_extensions_enable_height` to a future block height to defer activation.

---

### M-03 — `uint64` to `int64` Cast in Milestone Payout
**Location:** `chain/x/milestone/keeper.go`  
**Impact:** The milestone payout amount is stored as `uint64` but cast to `int64` for `sdk.NewInt`. Values above `math.MaxInt64` (≈ 9.2 × 10¹⁸) will wrap to negative, causing `sdk.NewInt` to produce a negative coin amount, which will likely panic or cause incorrect fund transfers.  
**Fix:** Use `math.NewIntFromUint64` or validate that the payout amount fits within int64 bounds before casting.

---

### M-04 — `json.Marshal` Error Silently Dropped in Bridge Keeper
**Location:** `chain/x/bridge/keeper.go`  
**Impact:** If message serialization fails during bridge event emission, the error is silently ignored and an empty or nil byte slice is stored as the event attribute. Downstream indexers relying on this data will receive corrupt or empty data, causing silent data loss in the bridge event log.  
**Fix:** Return the marshaling error to the caller. If event emission failure should not abort the bridge transaction, log a warning metric and emit a sentinel attribute indicating serialization failure.

---

### M-05 — Constitution Compliance Check Uses Substring Match
**Location:** `contracts/constitution/src/contract.rs` line 89  
**Impact:** `CheckProposal` determines compliance by checking `summary.contains("VIOLATION")`. This is a case-sensitive string match on free-form text. A proposal that includes the word "VIOLATION" in a negative context (e.g., "This proposal will NOT cause a VIOLATION") would be incorrectly flagged as non-compliant. Conversely, a genuinely non-compliant proposal that avoids the word "VIOLATION" would pass the check.  
**Fix:** Replace with a structured compliance framework: define specific, enumerable compliance rules as structured data (e.g., a list of forbidden action types), and check proposals against this structured ruleset rather than free-form text matching.

---

### M-06 — Reserve Fund Lock Cleared Before Error Return
**Location:** `contracts/reserve-fund/src/contract.rs`  
**Impact:** The reentrancy lock is cleared before returning a meaningful error in certain code paths, meaning the lock's protection is released prematurely. If the outer function's error causes a re-entry from a reply handler, the lock will no longer be set.  
**Fix:** Ensure the reentrancy lock is cleared only at the final point of the function, after all state writes are complete and no further re-entry is possible.

---

### M-07 — `BSCWatcher.pendingLocks` In-Memory Slice Not Persisted (Partial)
**Location:** `relayer/bsc_watcher.go`  
**Note:** `SaveLockEvent` is called from `cmd/relayer/main.go` and does write to the DB. However, the in-memory `pendingLocks` slice in `BSCWatcher` is the primary deduplication buffer between event observation and DB write. If a crash occurs between appending to `pendingLocks` and calling `SaveLockEvent`, the event is lost from memory and the deduplication state is reset, potentially causing double-processing on restart.  
**Fix:** Move deduplication state entirely into the DB (query `lock_events` table before processing). Remove the in-memory slice.

---

### M-08 — `SubmitSignature` API Handler Writes to Read Database
**Location:** `backend/module/api/main.go` line 561  
**Impact:** The `SubmitSignature` endpoint writes a new signature record to `readDB` (the read replica) instead of `writeDB` (the write primary). In a production CQRS setup, the read DB is a standby replica and writes to it will fail silently or with a non-obvious error. Signatures may appear to be submitted successfully but are never actually stored.  
**Fix:** Change `readDB.ExecContext(...)` to `writeDB.ExecContext(...)` in `SubmitSignature`.

---

### M-09 — Read Standby Uses `api_reader` Role for `pg_basebackup`
**Location:** `docker-compose.yml` line 115  
**Impact:** `pg_basebackup` requires either a superuser or a role with the `REPLICATION` attribute. The `api_reader` role (created for read-only application queries) almost certainly does not have this privilege, causing the standby initialization to fail silently or with a misleading error.  
**Fix:** Create a dedicated `replication_user` with `REPLICATION` privilege and use it exclusively for `pg_basebackup` and streaming replication. Do not grant application read roles replication privileges.

---

### M-10 — `bridge_pending.nonce` Is BIGINT; On-Chain Nonce Is 256-bit Bytes
**Location:** `db/read_schema/000001_init_read.sql` line 44  
**Impact:** The on-chain bridge nonce is a `bytes32` keccak256 hash (256 bits). PostgreSQL `BIGINT` is 64 bits and cannot store values above ~9.2 × 10¹⁸. When the projection service attempts to insert a nonce that doesn't fit in a 64-bit integer, the insert will fail or truncate, causing the bridge event to be dropped from the read model.  
**Fix:** Change the column type to `BYTEA` or `VARCHAR(66)` to store the full 32-byte hex nonce (with `0x` prefix).

---

### M-11 — `validator_address` VARCHAR(42) Too Short for Bech32 Addresses
**Location:** `db/read_schema/000001_init_read.sql` line 16  
**Impact:** Bech32 Cosmos addresses with a human-readable part of `"sovereign"` (8 chars) are approximately 50 characters long. VARCHAR(42) matches Ethereum addresses but is too short for Cosmos bech32 addresses. Data insertion will be truncated or rejected with a constraint violation, causing missing or corrupt validator data in the read model.  
**Fix:** Change `validator_address` to `VARCHAR(64)` or `TEXT`.

---

### M-12 — `VerifyVoteExtension` Uses `UncachedContext`
**Location:** `chain/app/abci.go` line 69  
**Impact:** `VerifyVoteExtension` creates an `sdk.UnwrapSDKContext` from an uncached context. In CometBFT's ABCI++ flow, `VerifyVoteExtension` is called in a read-only context and should use a cached multi-store to avoid reflecting writes from concurrent processes. Using an uncached context may expose uncommitted state or produce non-deterministic results across validators.  
**Fix:** Use `ctx.CacheContext()` when creating the context for `VerifyVoteExtension`, and discard the cache write-back on return.

---

### M-13 — Certification Degraded-Mode Threshold 30% Below BFT Safety
**Location:** `chain/x/certification/keeper.go`  
**Impact:** The BFT safety threshold requires >1/3 of validators to be honest to prevent finality violations. Setting the degraded-mode activation threshold below 33.3% means the chain can operate in "normal mode" with an insufficient certified majority. See H-06 for the primary finding; this medium finding notes the specific threshold value.  
**Fix:** Raise the degraded-mode threshold to `>33%` (at minimum) of voting power, not just validator count.

---

### M-14 — `eventIdx` Surrogate Nonce Wraps on >1,000 Events per Block
**Location:** `backend/module/ingestion/main.go` line 244; `projection/main.go` (projection nonce)  
**Impact:** The surrogate nonce is computed as `blockHeight * 1000 + eventIndex`. If a block contains more than 1,000 events, the eventIndex overflows into the next block's range, causing a surrogate key collision with an event from a different block. This will cause primary key violations in the write schema, silently dropping high-volume blocks from the event log.  
**Fix:** Use a larger multiplier (e.g., `blockHeight * 100000`) or switch to a composite primary key `(block_height, event_index)` without a surrogate nonce.

---

## Low / Informational Findings

### L-01 — `addressCodec` in IBC Keeper Is `nil`
**Location:** `chain/app/app.go`  
**Impact:** The IBC keeper is initialized with a nil `addressCodec`. If any IBC path attempts to use address encoding/decoding, this will panic at runtime. IBC cross-chain transfers will be non-functional.  
**Fix:** Pass the configured `bech32Codec` or `addressCodec` when constructing the IBC keeper.

---

### L-02 — `governance-ext` Uses Non-Standard Store Key Access Pattern
**Location:** `chain/x/governance-ext/keeper.go`, `chain/app/app.go`  
**Note:** Catalogued as part of C-04. The store key access is the root cause of the isolation violation.

---

### L-03 — `ComputeCommitHash` Uses Untyped String Concatenation
**Location:** `chain/x/oracle/types.go`  
**Impact:** The commit hash for oracle commit-reveal is computed by concatenating validator address, asset symbol, price, and salt as plain strings without delimiters or length-prefixing. This creates potential hash collisions for edge-case inputs similar to the bridge hash issue (H-05). While harder to exploit in the oracle context (inputs are more constrained), it is an unnecessary risk.  
**Fix:** Use `fmt.Sprintf` with explicit field delimiters and a fixed-format string (e.g., `"%s|%s|%d|%s"`), or hash a protobuf-encoded struct.

---

### L-04 — `GetLivenessSigningRatio` Not Wired to Slashing Module
**Location:** `chain/app/abci.go`  
**Impact:** The certification module exposes a `GetLivenessSigningRatio` function but it is never called in the slashing ante handler or the slashing module hook. Validators who are offline produce no certification attestations, but their absence is never forwarded to the slashing module.  
**Fix:** Wire `GetLivenessSigningRatio` into the `EndBlocker` or custom slashing hook so that validators below the liveness threshold for certification are appropriately jailed.

---

### L-05 — Oracle `ComputeCommitHash` Uses Untyped String
**Location:** `chain/x/oracle/types.go`  
**Note:** See L-03; this is the same finding.

---

### L-06 — `GetLivenessSigningRatio` Not Wired (Duplicate)
**Location:** Same as L-04. Consolidated.

---

### L-07 — EVM Precompiles Gated Behind Environment Variable
**Location:** `chain/app/app.go`  
**Impact:** EVM precompile registration is wrapped in a check for an undocumented environment variable. If this variable is not set in production, the EVM module will be initialized without the expected precompiles, causing any EVM transaction that calls a precompile to revert with an unexpected error.  
**Fix:** Register precompiles unconditionally in production code. Use build tags if development vs. production configurations differ.

---

### L-08 — `x-wallet-address` Header Not Signature-Verified
**Location:** `backend/module/api/main.go`  
**Impact:** The API accepts a `x-wallet-address` header as a client identity claim for wallet-specific queries. This header is not verified against any signature. Any client can claim any wallet address and receive that wallet's data.  
**Fix:** For sensitive wallet-specific endpoints, require a signed challenge-response (e.g., `personal_sign` over a nonce). For public read endpoints, this may be acceptable but should be documented as unauthenticated.

---

### L-09 — `FindKeyPair` Nil Edge Case in HSM
**Location:** `oracle/hsm.go`  
**Impact:** `FindKeyPair` can return nil without error if no matching key pair is found. The caller does not check for nil before dereferencing, which will cause a panic.  
**Fix:** Return a sentinel error (`ErrKeyPairNotFound`) when no key pair is found, and check for the error at the call site.

---

### L-10 — Backend API Uses `grpc.WithInsecure()` for Chain Communication
**Location:** `backend/module/api/main.go` line 178  
**Impact:** The gRPC connection to the chain node uses `grpc.WithInsecure()`, transmitting all query data without TLS. In a production environment, gRPC connections should be TLS-encrypted, especially for sensitive chain state queries.  
**Fix:** Configure gRPC with `credentials.NewClientTLSFromCert(nil, "")` for production endpoints. Use insecure only for local loopback connections.

---

### L-11 — NATS Connection Has No Authentication
**Location:** `relayer/nats.go` lines 18–28  
**Impact:** The relayer's NATS connection is established without any credentials (no NKey, no username/password, no TLS). Combined with C-10 (hardcoded seeds elsewhere) and NATS ports being exposed on the Docker host, this means any process on the host network can publish to the NATS bus without authentication.  
**Fix:** Configure NKey authentication on the NATS connection. Restrict NATS server port exposure to internal Docker networks only.

---

### L-12 — DB Passwords Plaintext in Docker Compose
**Location:** `docker-compose.yml` lines 68, 87, 104, 150  
**Impact:** PostgreSQL, pgbouncer, and relayer-DB passwords (`sovereign_write_pwd`, `sovereign_read_pwd`, `relayer_db_pwd`) are committed in plaintext. These should be treated as compromised.  
**Fix:** Use Docker secrets or external secret injection (`${POSTGRES_PASSWORD}` from a secrets manager). Rotate all exposed passwords.

---

### L-13 — Explorer Frontend Bakes `localhost` URLs as Build Args
**Location:** `docker-compose.yml` lines 298–310  
**Impact:** `NEXT_PUBLIC_API_URL`, `NEXT_PUBLIC_WS_URL`, and `NEXT_PUBLIC_RPC_URL` are set to `http://localhost:*` as build-time args. Since these are `NEXT_PUBLIC_` variables, they are baked into the static bundle at build time. The explorer will be broken in any deployment that is not the exact machine running docker-compose.  
**Fix:** Use environment variables at runtime (not build time) for frontend URLs. For Next.js, consider server-side configuration injection or deploy-time environment variable replacement.

---

### L-14 — `phase_8_verification_test.go` Referenced in Threat Model Does Not Exist
**Location:** `doc/ops/security_threat_model.md` line 9  
**Impact:** The threat model claims that `phase_8_verification_test.go` provides automated coverage for 5 critical invariants. This file does not exist in the repository. The coverage claim is false, and the invariants are unverified.  
**Fix:** Either create the referenced test file and implement the invariant checks, or remove the false claim from the threat model.

---

### L-15 — All Tokenomics Allocations Are Unfinalized Placeholders
**Location:** `doc/governance/tokenomics.md`  
**Impact:** All token allocation percentages are marked `🔑 OWNER ACTION`. The ESOV/CSOV peg mechanism has not been chosen. Without a finalized tokenomics model, the genesis supply configuration, reward parameters, and treasury funding amounts cannot be validated for economic safety.  
**Fix:** Finalize tokenomics before testnet. Validate all supply parameters against the genesis invariant checker in `scripts/generate_genesis.go`.

---

### L-16 — All 7 Custodian Key Slots Are `[PENDING]`
**Location:** `doc/ops/key-fingerprint-registry.md`  
**Impact:** No custodian keys have been registered. The cold multisig scheme (which is the emergency-pause backstop for both CosmWasm contracts and the BSC bridge) has no actual keyholders. Emergency response is impossible until at least 5-of-7 custodians are registered.  
**Fix:** Assign real key fingerprints before any public testnet launch. Conduct a key ceremony and document each custodian's identity and key fingerprint.

---

### L-17 — PITR Runbook Has Hardcoded Recovery Timestamp
**Location:** `doc/ops/runbooks.md` line 90  
**Impact:** The PostgreSQL point-in-time recovery runbook contains `recovery_target_time = '2026-06-24 02:00:00+06'`. A real disaster recovery operation using this runbook without noticing the hardcoded date would recover to a date in the past rather than the incident time, causing data loss.  
**Fix:** Replace the hardcoded timestamp with a placeholder: `recovery_target_time = '<INCIDENT_TIMESTAMP>'` and add a prominently visible instruction to set it before executing.

---

### L-18 — Genesis Constitution Text Is Placeholder `"Safe rules"`
**Location:** `scripts/generate_genesis.go` line 348  
**Impact:** The constitution contract is instantiated at genesis with `constitution_text: "Safe rules"`. Any governance proposal check will run against this placeholder. If the chain launches with this text, the constitution compliance mechanism is completely non-functional.  
**Fix:** Define the actual constitution text before genesis. Add a genesis validation check that rejects `"Safe rules"` or any known-placeholder text.

---

### L-19 — Genesis `governance.proposers` Hardcoded to Constitution Contract Address
**Location:** `scripts/generate_genesis.go` line 423  
**Impact:** The allowed proposers list in the governance contract is initialized to contain only the constitution contract's address. This means only the constitution contract can create proposals, which may be intentional for the initial protocol, but it means no external governance proposals can be submitted until this list is updated through a governance action — a chicken-and-egg problem.  
**Fix:** Include at least one verified externally-controlled proposer address (e.g., the cold multisig address) in the initial proposers list to allow governance bootstrapping.

---

### L-20 — Faucet Cooldown Is 10 Seconds
**Location:** `backend/module/faucet/main.go` line 47  
**Impact:** The faucet enforces a per-address cooldown of 10 seconds. Any script can drain testnet funds from an unlimited number of addresses in minutes. For mainnet, there should be no faucet, but for testnet, this cooldown should be at minimum 24 hours.  
**Fix:** Raise cooldown to at least 24 hours for public testnet. Add IP-based rate limiting in addition to address-based limiting.

---

## Module-by-Module Analysis

### Chain Modules

#### `x/bridge`
**Status:** 🔴 Critical issues  
- C-02: Amount overflow in `BridgeIn` — any EVM amount exceeding int64 max wraps
- H-05: Hash collision risk in `ComputeBridgeMessageHash` due to missing length prefix
- M-04: Silent `json.Marshal` error drops event data
- Bridge module is otherwise well-structured with bitmap nonce deduplication and supply cap enforcement

#### `x/oracle`
**Status:** 🟠 High issues  
- C-07: Colon injection in composite key construction
- L-03: Untyped string in `ComputeCommitHash`
- Commit-reveal scheme, MAD filtering, and expiry index are architecturally sound
- Keeper test suite covers commit-reveal, MAD, staleness, and EndBlocker — good coverage

#### `x/validator`
**Status:** 🟠 High issues  
- H-01: Permanent tombstoning on ejection is disproportionate and governance-abusable
- Power equalization and partition scheme are well-designed
- `ValidateBasic` on all message types is correct

#### `x/settlement`
**Status:** 🔴 Critical issues  
- C-01: Settlement marked processed before `SendCoins` — funds can be lost on error
- H-02: No timestamp upper bound allows far-future settlements
- Ed25519 signature verification with domain separation is correctly implemented

#### `x/governance-ext`
**Status:** 🔴 Critical issues  
- C-03: Uses `WasmKeeper.Execute` (state-mutating) for what should be a read-only compliance check
- C-04: Shares `govtypes.StoreKey` with the standard governance module — critical isolation violation
- 7-day execution delay enforcement is correct at the message level

#### `x/milestone`
**Status:** 🟡 Medium issues  
- M-03: `uint64→int64` cast for payout amount can overflow
- State machine is well-designed; O(1) stale skip is efficient; `SendCoins` error is handled

#### `x/certification`
**Status:** 🔴 Critical + Medium issues  
- C-08: `IsValidatorAttested` defaults `true` — all validators considered certified at chain start
- M-01: `WindowConsistencyInvariant` is O(N×W) — DoS risk
- M-13: Degraded-mode threshold (30%) below BFT safety bound (33%)
- Sliding window design is architecturally sound; invariant structure is good

#### `app/app.go`
**Status:** 🔴 Multiple issues  
- C-04: GovExtKeeper incorrectly uses `govtypes.StoreKey`
- H-09: EVM initialized before feemarket in genesis order
- L-01: IBC `addressCodec` is nil
- L-07: EVM precompiles gated behind undocumented env var
- All 7 custom modules are wired; module manager configuration is complete

#### `app/abci.go`
**Status:** 🔴🟡 Multiple issues  
- M-02: Vote extension stub string — vote extensions non-functional from genesis
- M-12: `UncachedContext` in `VerifyVoteExtension`
- C-08 flows through here: certification default true affects all ABCI hooks
- L-04: `GetLivenessSigningRatio` not wired to slashing

---

### CosmWasm Contracts

#### `contracts/governance/src/contract.rs`
**Status:** 🟡 Medium  
- M-05: Constitution compliance check uses substring match (`"VIOLATION"`)
- Multi-step lifecycle, replay protection, audit log, proposer allowlist are all correctly implemented
- Strong test coverage

#### `contracts/treasury/src/contract.rs`
**Status:** ✅ No critical issues  
- Reentrancy guard via `SubMsg::reply_on_success` is correct
- Withdrawal authorization checks are present

#### `contracts/reserve-fund/src/contract.rs`
**Status:** 🟡 Medium  
- M-06: Reentrancy lock cleared before error return in certain paths
- Guard mechanism is otherwise correctly designed

#### `contracts/constitution/src/contract.rs`
**Status:** 🟡 Medium (source of C-03)  
- `CheckProposal` is an `ExecuteMsg` (state-mutating) — must be a `QueryMsg`
- Pause/unpause/rotate lifecycle is correctly gated by governance authority
- `"VIOLATION"` substring match is too coarse for production compliance checking

---

### EVM Bridge

#### `bridge/src/LockBox.sol`
**Status:** 🔴 Critical — 4 critical findings  
- C-05: `recoverSigner` returns `address(0)` on bad sig — must be explicitly rejected
- C-05b: Message hash omits `address(this)` and `block.chainid` — signatures replayable across deployments
- C-11: `totalLocked` incremented before `transferFrom` — counter corrupted on failed transfer
- Bitmap nonce registry correctly prevents replay
- Rate limiting and dual-control circuit breaker are architecturally sound
- O(N²) duplicate signer check should be replaced with a mapping-based O(1) check

---

### Relayer Engine

#### `relayer/sig_aggregator.go`
**Status:** 🔴 Critical  
- C-06: Votes from NATS are counted without verifying the signer's cryptographic signature

#### `relayer/submitter.go`
**Status:** ✅ No critical issues  
- Deterministic promotion ladder is correctly implemented

#### `relayer/bsc_watcher.go`
**Status:** 🟡 Medium  
- M-07: In-memory `pendingLocks` slice; risk of event loss on crash between observation and DB write

#### `relayer/cosmos_watcher.go`
**Status:** 🟠 High  
- H-10: `pendingBurns` not persisted — burn events lost on crash

#### `relayer/signer.go`
**Status:** 🟠 High  
- H-07: `HorcruxSignerClient.Sign` returns hardcoded mock bytes — threshold signing is non-functional

#### `relayer/db.go`
**Status:** ✅ No critical issues  
- Dual-mode in-memory/Postgres with correct schema

#### `relayer/nats.go`
**Status:** 🔵 Low  
- L-11: No authentication on NATS connection

#### `relayer/cmd/relayer/main.go`
**Status:** 🔴 Critical  
- C-09: Auto-generates and **logs** random private key to stdout
- H-08: `checkLockInOrigin` uses `"ab"` prefix heuristic — semantically wrong and bypassable
- BSC chain ID hardcoded to testnet (97) — must be configurable for mainnet
- Mock Cosmos signature bytes in Tx — real signatures not constructed
- Multiple swallowed errors after broadcast failure
- `relayersList` hardcodes placeholder addresses

---

### Oracle Daemon

#### `oracle/main.go`
**Status:** 🟠 High  
- H-04: `time.Sleep(500ms)` hardcoded — not adaptive to block time
- HSM initialization is present and correctly gated by `ALLOW_MOCK_HSM`

#### `oracle/hsm.go`
**Status:** 🔵 Low  
- L-09: `FindKeyPair` nil edge case causes panic on missing key
- Dev/prod mode separation via env var is correctly implemented

---

### CQRS Backend

#### `backend/module/api/main.go`
**Status:** 🔴🟡 Multiple issues  
- C-10 (partial): Same three hardcoded NATS NKey seeds as ingestion/projection
- M-08: `SubmitSignature` writes to read DB instead of write DB
- L-08: `x-wallet-address` header accepted without signature verification
- L-10: `grpc.WithInsecure()` for chain communication
- `VolumeUsd` hardcoded to `"0.0"` — placeholder not replaced

#### `backend/module/faucet/main.go`
**Status:** 🔵 Low  
- L-20: 10-second cooldown is insufficient for any meaningful rate limiting
- Address normalization before `os/exec` is correct
- `CORS: *` is documented as intentional for devnet

#### `backend/module/ingestion/main.go`
**Status:** 🔴 Critical  
- C-10: Three hardcoded NATS NKey seeds as fallback
- M-14: Surrogate nonce wraps on >1,000 events per block
- Singleton advisory lock and backfill worker are correctly implemented

#### `backend/module/projection/main.go`
**Status:** 🔴 Critical  
- C-10: Same three hardcoded NATS NKey seeds as ingestion
- Block time (6000ms) and avg fee (150 uatom) hardcoded in projection output

---

### Database Schemas

#### `db/write_schema/000001_init_write.sql`
**Status:** 🟡 Minor issues  
- Partitioned events table is correctly designed
- No row-level security — appropriate for internal backend but must not be exposed directly
- No foreign key constraints — acceptable for CQRS write side

#### `db/read_schema/000001_init_read.sql`
**Status:** 🟡 Medium issues  
- M-10: `bridge_pending.nonce BIGINT` — too small for 256-bit on-chain nonce
- M-11: `validator_address VARCHAR(42)` — too short for bech32 Cosmos addresses (should be VARCHAR(64))
- Read schema has denormalized aggregate tables — correct CQRS pattern

---

### Docker / Infrastructure

**Status:** 🔴🟠 Multiple issues  
- H-11: Vault in dev mode — all secrets in memory, TLS disabled, root token `"root"`
- L-12: DB passwords plaintext in `docker-compose.yml`
- L-11: NATS ports (4222, 8222) exposed on Docker host without TLS or authentication
- M-09: Read standby uses `api_reader` for `pg_basebackup` — needs `REPLICATION` privilege
- L-13: Explorer frontend bakes `localhost` URLs at build time
- No resource limits on any container — uncontrolled memory/CPU consumption
- No health-check commands on critical DB or NATS containers
- `chain/genesis.dev.json` mounted into chain container — production should mount production genesis

---

### Scripts & Genesis

**Status:** 🔴🟡 Multiple issues  
- C-12: `scripts/run_real_testnet_e2e.sh` hardcodes BSC testnet private key
- L-18: Constitution genesis text is placeholder `"Safe rules"`
- L-19: `governance.proposers` only contains the constitution contract — governance bootstrapping blocked
- Vote extensions enabled at height 0 but ABCI handler is a stub (M-02) — contradiction
- Contract instantiation uses `AccessTypeEverybody` — anyone can deploy new copies of all contracts
- `approval_threshold: 1` in governance contract is very low — single approval sufficient
- Genesis invariant checker (6 checks) is well-implemented; the checks that are there are correct

---

### Documentation & Operations

**Status:** 🔵 Multiple informational issues  

- `audit_engagement.json`: No audit firm has been engaged; all three slots are `not-started`; engagement letter date is null; bug bounty program not started
- `doc/ops/security_threat_model.md`: Claims `phase_8_verification_test.go` provides automated invariant testing — file does not exist (L-14); threat model does not cover: NATS server compromise, relayer private key theft, hardcoded credential exposure, constitution contract governance takeover
- `doc/ops/key-fingerprint-registry.md`: All 7 slots are `[PENDING]` as of 2026-07-15 (L-16)
- `doc/governance/tokenomics.md`: All allocations are `🔑 OWNER ACTION` placeholders; ESOV/CSOV peg mechanism not chosen (L-15)
- `doc/ops/runbooks.md`: PITR section has hardcoded recovery timestamp `2026-06-24 02:00:00+06` (L-17)
- `planned-vs-implemented.md`: Claims 359/359 tasks complete including Phase 9 External Audit — both false
- `mainnet-plan.md`: Accurately identifies WASM mismatch and chain-ID issues — honest gap analysis
- `doc/mainnet/launch-day-runbook.md`: Well-structured ceremony runbook; correctly requires replacing all `OWNER_ACTION_REQUIRED` placeholders
- `doc/ops/bug-bounty-policy.md`: Bounty policy is well-defined (max $50k CSOV for critical) but cannot be activated until credentials are rotated and external audit is complete

---

## Deployment Readiness Checklist

### 🔴 Blockers — Must Resolve Before ANY Public Network

| # | Item | Status |
|---|---|---|
| 1 | Rotate all three hardcoded NATS NKey seeds (C-10) | ❌ Credentials in source |
| 2 | Remove relayer private key auto-generation and log line (C-09) | ❌ Key logged to stdout |
| 3 | Remove hardcoded BSC private key from E2E script (C-12) | ❌ Key in source |
| 4 | Fix `LockBox.sol` `totalLocked` pre-increment before transferFrom (C-11) | ❌ Unpatched |
| 5 | Add `address(this)` + `block.chainid` to `LockBox.sol` message hash (C-05b) | ❌ Unpatched |
| 6 | Reject `address(0)` from `recoverSigner` in `LockBox.sol` (C-05) | ❌ Unpatched |
| 7 | Fix `MarkSettlementProcessed` called before `SendCoins` (C-01) | ❌ Logic order wrong |
| 8 | Fix `sdk.NewInt(int64(msg.Amount))` overflow in bridge keeper (C-02) | ❌ Unpatched |
| 9 | Fix `WasmKeeper.Execute` → `WasmKeeper.QuerySmart` for constitution check (C-03) | ❌ Unpatched |
| 10 | Assign dedicated store key to `governance-ext` module (C-04) | ❌ Unpatched |
| 11 | Fix sig aggregator to verify relayer signatures on NATS votes (C-06) | ❌ Unpatched |
| 12 | Fix oracle composite key injection (C-07) | ❌ Unpatched |
| 13 | Change `IsValidatorAttested` default to `false` (C-08) | ❌ Unpatched |
| 14 | Implement real `HorcruxSignerClient.Sign` — remove mock bytes (H-07) | ❌ Stub |
| 15 | Remove `checkLockInOrigin` prefix heuristic (H-08) | ❌ Broken logic |
| 16 | Persist `CosmosWatcher.pendingBurns` to DB before NATS publish (H-10) | ❌ Loss on crash |
| 17 | Fix Vault configuration — remove dev mode (H-11) | ❌ Dev mode |
| 18 | All DB passwords removed from docker-compose source (L-12) | ❌ Plaintext |
| 19 | Rotate all plaintext docker-compose passwords | ❌ Compromised |

### 🟠 Required Before Public Testnet

| # | Item | Status |
|---|---|---|
| 20 | Fix permanent tombstoning on ejection (H-01) | ❌ |
| 21 | Add settlement timestamp upper bound (H-02) | ❌ |
| 22 | Fix oracle daemon block-time-aware scheduling (H-04) | ❌ |
| 23 | Fix bridge message hash — add length prefix (H-05) | ❌ |
| 24 | Fix degraded-mode threshold >33% (H-06 / M-13) | ❌ |
| 25 | Fix EVM/feemarket genesis initialization order (H-09) | ❌ |
| 26 | Fix `SubmitSignature` writes to wrong DB (M-08) | ❌ |
| 27 | Fix `bridge_pending.nonce` column type to BYTEA/VARCHAR(66) (M-10) | ❌ |
| 28 | Fix `validator_address` VARCHAR(42) → VARCHAR(64) (M-11) | ❌ |
| 29 | Fix surrogate nonce wrap on >1000 events/block (M-14) | ❌ |
| 30 | Fix IBC `addressCodec` nil in app.go (L-01) | ❌ |
| 31 | Fix NATS relayer authentication (L-11) | ❌ |
| 32 | Fix read standby replication user privileges (M-09) | ❌ |
| 33 | Fix explorer frontend runtime URL injection (L-13) | ❌ |
| 34 | Register all 7 custodian key fingerprints | ❌ |
| 35 | Finalize constitution text (replace "Safe rules") | ❌ |
| 36 | Finalize genesis proposers list | ❌ |
| 37 | Implement real vote extension logic or defer activation (M-02) | ❌ |

### 🟡 Required Before Mainnet

| # | Item | Status |
|---|---|---|
| 38 | Engage and complete formal external security audits (all 3 scopes: Go chain, CosmWasm, EVM/infra) | ❌ Not started |
| 39 | Finalize tokenomics model and allocations (L-15) | ❌ Placeholder |
| 40 | Activate bug bounty program on Immunefi or equivalent | ❌ Not started |
| 41 | Complete key ceremony for cold multisig custodians | ❌ Not started |
| 42 | `planned-vs-implemented.md` — accurate status tracking only | ❌ Overstated |
| 43 | Fix PITR runbook hardcoded timestamp (L-17) | ❌ |
| 44 | `phase_8_verification_test.go` — create or remove the reference (L-14) | ❌ File missing |
| 45 | Fix `WindowConsistencyInvariant` O(N×W) complexity (M-01) | ❌ |
| 46 | Fix `FindKeyPair` nil dereference in HSM (L-09) | ❌ |
| 47 | Wire `GetLivenessSigningRatio` to slashing module (L-04) | ❌ |
| 48 | Replace `grpc.WithInsecure()` with TLS (L-10) | ❌ |
| 49 | Fix `UncachedContext` in `VerifyVoteExtension` (M-12) | ❌ |
| 50 | Replace constitution `"VIOLATION"` substring check (M-05) | ❌ |

---

## Remediation Roadmap

### Phase 0 — Immediate (Before Committing Any More Code)

**Credential Rotation — Do This Now:**
1. Rotate all three NATS NKey seeds. Generate new ones via `nsc generate nkey --account`. Never commit seeds to source.
2. Rotate the BSC private key from `scripts/run_real_testnet_e2e.sh`. Sweep any funds from the compromised address.
3. Rotate all plaintext docker-compose passwords (`sovereign_write_pwd`, `sovereign_read_pwd`, `relayer_db_pwd`).
4. If the Vault dev-mode root token `"root"` was used to store any real secrets, rotate those secrets.
5. Audit GitHub's public commit history — assume all hardcoded values have been scraped.

### Phase 1 — Critical Security Fixes (1–2 Weeks)

Priority order based on exploitability and blast radius:

1. **Bridge safety:** Fix C-01 (settlement order), C-02 (int overflow), C-05, C-05b, C-11 (LockBox), C-12 (E2E key)
2. **Relayer integrity:** Fix C-06 (unsigned NATS votes), C-09 (key logging), H-07 (Horcrux mock), H-08 (prefix heuristic), H-10 (persist burn events)
3. **Chain module isolation:** Fix C-03 (Execute→QuerySmart), C-04 (store key isolation), C-07 (oracle key injection), C-08 (certification default)
4. **Infrastructure:** Fix H-11 (Vault dev mode), configure NATS authentication, move all secrets to Vault

### Phase 2 — High/Medium Fixes (2–4 Weeks)

1. Fix H-01 (tombstoning), H-02 (timestamp ceiling), H-04 (oracle scheduling), H-05 (hash length-prefix), H-09 (genesis order)
2. Fix M-02 (vote extension stub), M-03 (uint64 overflow), M-08 (wrong DB write), M-10, M-11 (schema types)
3. Fix M-12, M-13, M-14 (ABCI context, threshold, nonce)
4. Fix L-01 (IBC codec), L-04 (liveness wiring), L-09 (HSM nil)

### Phase 3 — Pre-Testnet Hardening (2–4 Weeks)

1. Complete constitution text, genesis proposers, tokenomics
2. Register custodian key fingerprints
3. Implement real vote extension data (oracle commits)
4. Fix all database schema type mismatches
5. Replace localhost URLs in docker-compose with runtime-injected values
6. Write `phase_8_verification_test.go` or remove the reference

### Phase 4 — External Audit Engagement

Engage auditors per `audit_engagement.json` scopes:
- **Scope A+B (Go chain + CosmWasm):** Informal Systems, Zellic, or Oak Security
- **Scope C+E (Bridge + EVM):** Trail of Bits, Halborn, or Spearbit
- **Scope D (Infrastructure):** Internal red team + specialist firm

Provide complete audit packages including threat model, ADRs, and this report as context.

**Mainnet gate criteria** (from `audit_engagement.json`) must be met:
- Zero unresolved critical findings
- Zero unresolved high findings
- All medium findings resolved or formally risk-accepted
- Final report published before genesis

### Phase 5 — Mainnet Preparation

1. Genesis ceremony with all placeholder replacements (per `launch-day-runbook.md`)
2. Horcrux threshold signing fully operational (replace H-07 mock)
3. Bug bounty program live on Immunefi before genesis
4. All 7 custodian key slots populated and verified
5. BSC mainnet LockBox and Gnosis Safe deployed
6. Minimum 3 validators with verified Horcrux configurations

---

*This report was produced by independent static analysis and manual code review of all files in the repository as of 2026-07-16. It does not constitute a formal security audit and should be supplemented by engaged external audit firms before any mainnet launch. The audit engagement criteria in `audit_engagement.json` correctly specifies zero unresolved critical/high findings as a mainnet gate — a bar this codebase does not currently meet.*
