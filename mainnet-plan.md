# Mainnet Launch Plan — Sovereign L1

**Purpose:** single execution plan for taking this repo from its current devnet/testnet state to a mainnet that real users can safely transact on. Written for an implementing agent to follow phase by phase — do not skip a phase or start a later phase before its dependencies are marked done.

**How to use this file:** each phase has a goal, concrete tasks, and a verification step. Do not mark a phase complete until its verification step actually passes — don't take "looks right" as done. Phases are ordered by dependency, not by effort — some short phases (B, C) block everything after them.

**This file is fully self-contained.** All contract and module fix details (previously split across separate `fix-phase-2.md`/`fix-phase-3.md` documents) are inlined directly in Phase A below — you do not need any other file to execute this plan.

---

## ⚠️ 3-Day Launch Reality Check (read this first)

The project owner has stated intent to launch on mainnet, with real market liquidity, in **3 days**. After a deep, whole-project re-review (all 10 phases in `planned-vs-implemented.md`, not just Phase 2/3), the honest answer is: **a 3-day timeline is not achievable without knowingly shipping fund-theft bugs and a fabricated audit.** This is not a formatting/polish problem — it is a real-money-loss risk. Specifics below, so the implementing agent and the project owner both see exactly why.

### The single biggest false claim in the repo: there is no real security audit

`planned-vs-implemented.md` marks **Phase 9 — Security Audit** as 100% ✅, citing `doc/ops/audit_engagement.json` and `e2e/phase_9_verification_test.go`. On inspection:
- `doc/ops/audit_engagement.json` is a hand-written JSON file that just *asserts* `"status": "pre-engaged"` for three auditor slots (named as an OR-list of real firms — "Informal Systems / Zellic / Oak Security", "Trail of Bits / Halborn / Zellic / Spearbit" — meaning **no specific firm has actually been engaged**, the list is aspirational).
- `e2e/phase_9_verification_test.go` does not call any external auditor, does not submit code anywhere — it just parses that same JSON file and asserts its own fields are `true`. **A test asserting a config file says "true" is not a security audit.** This is a self-referential fake — the codebase is auditing its own claim that it was audited.
- There is no audit report, no findings list, no auditor sign-off anywhere in the repo.

The same self-verification pattern shows up in Phase 6.9 ("testnet stable for 4 weeks") and Phase 6.6 ("≥5 external validators onboarded") — both are backed by planning documents (`doc/testnet/stability_checklist.md`, `doc/testnet/onboarding.md`) describing what *should* happen, not evidence that a real public testnet actually ran for 4 weeks with 5 independent external operators. Treat every "✅" in `planned-vs-implemented.md` whose evidence column says "documented", "verified in E2E tests", or "configured in JSON" — rather than pointing at independent, real-world confirmation — as **unverified, not done**, until proven otherwise.

### Confirmed-real vs confirmed-fake, from this session's deep pass

| Claim | Real or not | Evidence |
|---|---|---|
| LockBox.sol pause/unpause/rate-limit logic | ✅ Real | `bridge/src/LockBox.sol` — actual `paused`, `circuitBreakerAddress`, `maxUnlockPerBlock` enforcement in code |
| x/bridge supply cap enforcement | ✅ Real | `chain/x/bridge/keeper.go` — actual `newMinted > params.SupplyCap` check before minting |
| Phase 9 "Security Audit" complete | ❌ Fake | Self-referential JSON + test, no real auditor engaged, no report exists |
| Phase 10.1 "genesis supply invariant verified, 700M Cosmos / 300M BSC allocation" | ❌ Fake / stale | Actual `chain/genesis.json` still has 2 test accounts with round placeholder numbers and unbacked `uwsov` supply (see Phase C below) — does not match this claim at all |
| Phase 6.9 "testnet stable 4 weeks" | ❌ Unverified | Only a checklist document exists, no evidence of an actual multi-week public run |
| Treasury/Reserve Fund reentrancy lock | ❌ Fake (fund-theft bug) | See Phase A2 below — lock check exists but doesn't actually block re-entry |
| Governance proposal execution auth | ❌ Fake (fund-theft/takeover bug) | See Phase A1 below — no real sender authentication |
| Chain-id consistency (Cosmos + EVM) | ❌ Broken | `sovereign-1` vs `sovereign-testnet-1`, and EVM `7777` vs `9001` mismatches across configs — see Phase B below |

### What this means for "launch in 3 days"

You cannot compress a real external audit, a real multi-week public testnet, and real independent validator onboarding into 3 days — these require other people's calendars, not just engineering hours. Two honest paths forward, pick one explicitly:

**Path 1 — Delay the public/liquidity launch, keep the 3-day window for what's actually achievable in 3 days.** In 3 days an agent *can* realistically: fix the two fund-theft contract bugs (Phase A), fix chain-id consistency (Phase B), rebuild a real (not fake-placeholder) genesis (Phase C), and get monitoring live. That is a legitimate "technical mainnet genesis" milestone — but it must **not** have public liquidity, an announced token sale, or marketing pointing real money at it until Phases D/E/F/G below are also done. Launching the chain itself is not the same as launching a market for the token.

**Path 2 — Launch something in 3 days anyway.** If the project owner insists on a public, liquid launch in 3 days regardless, that is the project owner's call to make with full knowledge of the risk, not something this plan can certify as safe. If this path is chosen: fix Phase A (fund-theft bugs) at an absolute minimum — do not skip that regardless of timeline — and be explicit publicly that no third-party audit has occurred yet (do not claim "audited" anywhere in marketing, since Phase 9 currently has zero real audit despite what the repo's own tracking doc says).

The rest of this plan is written to support Path 1, since Path 2 has no engineering safeguard that makes it advisable — but every phase below still applies whichever path is chosen; Path 2 just chooses to launch before finishing Phases D–G at the project owner's own risk.

---

## Current state summary (why this plan exists)

As of this audit, the codebase has real, working core logic across Phase 2 (custom Cosmos SDK modules) and Phase 3 (CosmWasm contracts), but is **not safe for real user funds** because of:
1. Two exploitable contract bugs that allow direct fund theft (fake reentrancy locks, unauthenticated governance execution)
2. A genesis file that is test/dev fixture data, not real mainnet allocation (2 test accounts, unbacked bridge-minted token supply)
3. Chain-id inconsistencies between validator signer config, genesis, and frontend wallet config that will break signing and wallet connections
4. No external security audit, no real validator set, no completed public testnet run
5. No liquidity, no listing, no legal review — the token has no real market value yet even if the chain were technically safe

This plan closes all of these gaps in dependency order.

---

## Phase A — Critical contract & module security fixes

**Goal:** eliminate every fund-theft and unauthenticated-privilege bug before any of this code touches real value.

**Depends on:** nothing — start here.

**Suggested order within this phase:** A1 (governance auth bypass) → A2 (fake reentrancy guards) → A7 (instantiation-time governance address, simplifies re-testing A1/A2) → A4 (genesis/artifact mismatch) → A3 (real e2e tests, do after A1/A2/A4 so they exercise real fixed behavior) → A12 (Constitution check payload, coordinate with A1's compliance-check fix — same root cause) → A9 (oracle daemon hardcoding, blocks real devnet ops) → A8 (SimGov wrapper) → A10 (oracle O(N) paths, perf only) → A11 (Prometheus metrics, additive) → A5 (schema.rs, independent) → A6 (ADR-007 keys, ownership decision) → A13 (EVM/CosmWasm test + precompile policy, depends on a policy decision from the project owner).

### A1. CRITICAL — Governance `SubmitProposal` has no sender check and executes arbitrary attacker-supplied messages

**File:** `contracts/governance/src/contract.rs` (lines 30–72). **Also touches:** `contracts/governance/src/msg.rs`, `contracts/governance/src/state.rs`, `contracts/governance/tests/integration_tests.rs`

**Root cause:** `execute()` ignores `_info: MessageInfo` entirely. Anyone can call `SubmitProposal` with any `actions: Vec<CosmosMsg>`, and the handler unconditionally does `.add_messages(actions)`. Because Treasury, Reserve Fund, and Constitution all trust this contract's address as `governance_address`, an attacker can craft a `SubmitProposal` whose `actions` contain a `WasmMsg::Execute` targeting Treasury's `Withdraw` (or Reserve Fund's `DisburseMilestone`, or Constitution's `UpdateConstitution`) — and it will be dispatched as if Governance itself sent it. This is a full fund-drain / constitution-rewrite path with no authorization at all. The "Constitution compliance check" is also not real enforcement — it's a substring search for the literal word `"VIOLATION"` in the rules text, not a rules engine.

**Fix — require an authorized proposer + real voting state, not a bare passthrough.** This needs a minimal-but-real proposal lifecycle: propose → vote/approve → execute, with an explicit allow-list of who can propose and who can approve. Do not ship a fix that just adds a single-address check and still auto-executes in the same call — that reintroduces a single-point-of-failure "governance" that is really just one key.
1. `contracts/governance/src/state.rs` — add to `Config`: `pub proposers: Vec<Addr>` (addresses allowed to call `SubmitProposal`), `pub approval_threshold: u64` (number of `ExecuteProposal` approvals required). Add a new `PROPOSALS: Map<u64, Proposal>` (or extend the existing audit-log map) storing `status: Pending | Approved | Executed | Rejected` and `approvals: Vec<Addr>`.
2. `contracts/governance/src/msg.rs` — change `ExecuteMsg` from a single variant to a real lifecycle: `SubmitProposal { title, description, actions: Vec<CosmosMsg> }`, `ApproveProposal { proposal_id: u64 }`, `ExecuteProposal { proposal_id: u64 }`. `SubmitProposal` stores the proposal as `Pending` and does **not** call `add_messages`. `ApproveProposal` requires `info.sender` to be in `config.proposers` (or a separate `voters` list — your call, but it must be an explicit allow-list, not "anyone"). `ExecuteProposal` requires the approval count to have reached `approval_threshold`, and only then does `.add_messages(actions)`, and it must flip status to `Executed` so it cannot be replayed.
3. `contracts/governance/src/contract.rs` — in `SubmitProposal`, add at the top of the handler: reject with `StdError::generic_err("Unauthorized: sender is not an authorized proposer")` if `!config.proposers.contains(&info.sender)`. Keep the existing Constitution `GetConstitution` query call, but treat it as a real check: query the actual rules content and reject on a **structured** rule violation the Constitution contract can express (e.g. an `is_action_permitted` query it answers), not a hardcoded string match. If a proper rules engine is out of scope for this pass, at minimum keep the substring check but rename it clearly (e.g. `// TODO: placeholder compliance check — not real rule enforcement`) and gate it behind the sender-authorization fix above so it's no longer the only line of defense. (Coordinate this with A12 below — same root problem, different module.)
4. `contracts/governance/tests/integration_tests.rs` — the existing tests all call `SubmitProposal` with `actions: vec![]` and never touch the vulnerable path. Add new tests that specifically: call `SubmitProposal` from an address **not** in `proposers` → assert rejected; call `SubmitProposal` from an authorized proposer with a non-empty `actions` (e.g. `WasmMsg::Execute` targeting Treasury's `Withdraw`), confirm it does **not** execute until `ApproveProposal`/`ExecuteProposal` run and threshold is met; attempt `ExecuteProposal` before threshold met → assert rejection; attempt `ExecuteProposal` twice on the same proposal → assert the second call is rejected (no replay).

**Verify:** `cd contracts/governance && cargo test` — all new tests above must pass, and the old "empty actions" tests must still pass unmodified (extend, don't replace them).

### A2. CRITICAL — Treasury and Reserve Fund "reentrancy guard" is cosmetic (never actually locks)

**Files:** `contracts/treasury/src/contract.rs` (lines 58–86), `contracts/reserve-fund/src/contract.rs` (`DisburseMilestone` handler — identical pattern)

**Root cause:** both contracts set `config.reentrancy_lock = true`, save, build the `BankMsg::Send`, then set `config.reentrancy_lock = false` and save again — all in the *same* synchronous call, **before** CosmWasm actually dispatches the message. CosmWasm dispatches `add_message`/submessages **after** the handler returns, so by the time `send_msg` actually executes, the lock has already been reset to `false`. The lock is never observably `true` to any call that could occur during message dispatch — it provides no protection whatsoever. It only "tests green" because the existing unit tests never attempt a genuine reentrant call.

**Fix — lock before dispatch, unlock in a follow-up entry point (`reply`).** Do not just reorder the two lines — a plain `BankMsg::Send` produces no callback to clear the lock later, so leaving it locked forever would brick the contract after one withdrawal.
1. In `execute()`'s `Withdraw` (Treasury) / `DisburseMilestone` (Reserve Fund) handler: if `config.reentrancy_lock` is already true, return `StdError::generic_err("Reentrancy guard: Operation already in progress")`. Otherwise set `config.reentrancy_lock = true`, save, then wrap the `BankMsg::Send` as `SubMsg::reply_on_success(send_msg, WITHDRAW_REPLY_ID)` and return it via `.add_submessage(...)`. Do **not** set `reentrancy_lock = false` here.
2. Add a `#[entry_point] pub fn reply(deps: DepsMut, _env: Env, msg: Reply) -> StdResult<Response>` that matches on `WITHDRAW_REPLY_ID`, loads `Config`, sets `reentrancy_lock = false`, saves it, and returns `Ok(Response::new())`. Define `const WITHDRAW_REPLY_ID: u64 = 1;` (each contract is a separate binary, so `1` is fine in both).
3. Import `SubMsg`, `Reply`, and (if needed) `SubMsgResult` from `cosmwasm_std` in both contracts.
4. Tests to add in both contracts' `#[cfg(test)] mod tests`: call `Withdraw`/`DisburseMilestone` once, assert `reentrancy_lock == true` by directly reading `CONFIG` from `deps.storage` **before** simulating the reply (this check was previously impossible to write because the flag was always `false`); call it a second time while still locked → assert rejected with the "already in progress" error; manually invoke the new `reply()` entry point with a mock success `Reply`, then assert `reentrancy_lock == false` and that a third call now succeeds.

**Verify:** `cd contracts/treasury && cargo test` and `cd contracts/reserve-fund && cargo test`.

### A3. HIGH — `e2e/phase_3_*_test.go` do not test the actual contracts and should not be cited as "on-chain devnet integration tests"

**Files:** `e2e/phase_3_integration_test.go`, `e2e/phase_3_verification_test.go`

**Root cause:** `TestPhase3WasmCompilationAndStructure` checks for wasm binaries at `contracts/target/wasm32-unknown-unknown/release/` — a path that doesn't exist in this repo (compiled binaries live in `artifacts/`); this test fails on a clean checkout/build. `TestPhase3ConstitutionLogic`, `TestPhase3TreasuryLogic`, `TestPhase3ReserveFundLogic`, `TestPhase3GovernanceAndReplacementProcedure` all define **local Go closures that reimplement a toy mock of the contract logic inline in the test file**, then test those mocks — they never compile, deploy, or call the real Rust contracts, and would keep passing even with A1/A2's bugs still live (as they did during this audit). `TestPhase3Integration_GenesisState` and `TestPhase3Integration_GovernancePointerRotation` touch real repo artifacts/types but the rotation test only asserts on a struct literal it just constructed — it never submits or processes the message anywhere.

**Fix — pick one explicitly, don't leave it ambiguous in `planned-vs-implemented.md`:**
- **Option A (do the real thing):** write actual devnet integration tests using `wasmd`'s test app / `simapp`-style harness (or a running local node + Go RPC client) that: (1) instantiate all 4 contracts for real via `x/wasm`; (2) execute a real `MsgExecuteContract` calling Governance's `SubmitProposal` from a non-proposer key → assert the tx fails on-chain; (3) run a real propose → approve → execute flow resulting in a real `BankMsg::Send` from Treasury, asserting the recipient's on-chain balance actually changed; (4) attempt a reentrant call during the reply window (two `MsgExecuteContract`s for `Withdraw` in the same block) and assert the second is rejected once A2 lands; (5) run the governance-contract-replacement procedure (pause → new governance instantiate → `UpdateGovernanceAddress` on all 3 → unpause) as real sequential transactions, not in-memory struct mutation.
- **Option B (if a full devnet harness is out of scope right now):** rename these files/functions to make clear they are NOT integration tests (e.g. `phase_3_unit_mock_test.go`, prefix test names `TestPhase3MockLogic_...`), and update `planned-vs-implemented.md` rows 3.7/3.10 to state plainly that mandatory on-chain devnet integration tests are **not yet implemented**, pointing at this file as the tracking location.
- Also fix `TestPhase3WasmCompilationAndStructure`'s path to point at `artifacts/*.wasm` (do this after A4 resolves the artifact/genesis mismatch, so the test checks the binaries actually deployed).

**Verify:** `cd e2e && go test ./... -run Phase3 -v`

### A4. HIGH — `chain/genesis.json`'s embedded WASM bytecode does not match `artifacts/*.wasm`

**Files:** `chain/genesis.json`, `artifacts/*.wasm`, `artifacts/checksums.txt`, `scripts/generate_genesis.go`

**Root cause:** the 4 CosmWasm `code_bytes` entries embedded in the committed `chain/genesis.json` decode to sizes (20,499 / 257,830 / 273,205 / 301,714 bytes) that do not match the sizes of the corresponding files in `artifacts/` (167,782 / 183,971 / 195,029 / 218,731 bytes) — both are valid WASM (`\0asm` magic bytes present), but the genesis file was clearly built from a different compile pass than what's currently in `artifacts/`. `artifacts/checksums.txt` matches `artifacts/*.wasm` exactly, so that half is internally consistent. This means re-running `go run scripts/generate_genesis.go` today would produce a **different** `genesis.json` than the one committed — it is not reproducible from the current tree.

**Fix:**
1. Decide on a single source of truth: either (a) `chain/genesis.json` is always freshly generated right before use, never committed as a static file, or (b) it's committed but regenerated and re-committed on every contract source change, verified by CI.
2. Regenerate now: `go run scripts/generate_genesis.go` (uses the script's existing `env=dev`/`prod` flag), and commit the resulting `chain/genesis.json`.
3. Regenerate `artifacts/*.wasm` and `artifacts/checksums.txt` from the exact same compile invocation `compileContracts()` uses in `scripts/generate_genesis.go` (same `cargo build --target wasm32-unknown-unknown --release --lib` + the same `wasm-opt` pass), so both artifact sets come from one build.
4. Add a CI check (or extend the script's existing `--verify` flag) that hashes each `code_bytes` entry in `chain/genesis.json` and asserts it matches the corresponding checksum in `artifacts/checksums.txt`. Fail loudly if they diverge.

**Verify:** `go run scripts/generate_genesis.go --verify`, then `sha256sum artifacts/*.wasm` compared by hand against `chain/genesis.json`'s decoded `code_bytes` once, relying on the new CI check going forward.

### A5. MEDIUM — No `schema.rs` binary exists; `planned-vs-implemented.md` row 3.8's evidence description is false

**Files:** `contracts/{constitution,treasury,reserve-fund,governance}/Cargo.toml`, new `contracts/*/src/bin/schema.rs` files

**Root cause:** `schema/*.json` files exist and are content-correct for all 4 contracts, but there is no `[[bin]]` target in any `Cargo.toml` and no `bin/schema.rs` file anywhere — the claimed generation mechanism ("Configured schema.rs binaries in all contracts") doesn't exist. There's no repeatable way to regenerate schema if a contract's `msg.rs` changes, so it will silently drift.

**Fix:** for each of `constitution`, `treasury`, `reserve-fund`, `governance`: create `contracts/<name>/src/bin/schema.rs` using `cosmwasm_schema::write_api!` with that contract's `InstantiateMsg`/`ExecuteMsg`/`QueryMsg` (no explicit `[[bin]]` Cargo.toml stanza needed — Cargo auto-discovers `src/bin/*.rs`). Regenerate schema for all 4 contracts (`cargo run --bin schema`) and diff against the currently committed `schema/*.json` to confirm no drift. Update `planned-vs-implemented.md` row 3.8's evidence column once this lands.

**Verify:** `for c in constitution treasury reserve-fund governance; do (cd contracts/$c && cargo run --bin schema); done` then `git diff --stat contracts/*/schema/` — expect no diff, or a diff that's a strict superset reflecting real field changes.

### A6. MEDIUM — Cold multi-sig doesn't match the plan (3-of-5 vs required 5-of-7) and key material looks like placeholder text

**File:** `doc/adr/adr-007-operational-security.md` (section 9, lines 138–163)

**Root cause:** the plan specifies a 5-of-7 cold multi-sig with a defined composition (2 founding team, 2 independent security council, 2 lead validators, 1 legal trustee). ADR-007 documents a **3-of-5** multisig instead, with different role names that don't map to the plan's composition. All 5 listed `cosmospub1...` public keys share overlapping substrings (e.g. multiple end in the same suffix), which is inconsistent with distinct real secp256k1 public keys — they read as placeholder/fabricated values.

**Fix (this is a project-owner decision, not something to silently code around):**
1. Either update ADR-007 to specify 5-of-7 with the plan's stated composition and real distinct public keys once real custodians and hardware keys are provisioned, **or** formally amend the plan with a dated addendum recording that 3-of-5 was the deliberate final decision and why.
2. Whichever direction is chosen, regenerate real, distinct bech32 public keys for each custodian from actual hardware-key-derived material, and update `cold_multisig_address` (used in `contracts/*/state.rs` defaults and `scripts/generate_genesis.go`'s config JSON) to match the real derived multisig address for the chosen threshold/key set.
3. Do not mark this ADR as "done" until the key material is real — placeholder keys in a doc cited as production security documentation are themselves a risk if someone assumes it's live. (Same underlying issue flagged in Phase D above — resolve together.)

**Verify:** manual/process check — confirm the multisig address in ADR-007 matches the address independently derivable from the published public keys and threshold, and that it matches `cold_multisig_address` wired into genesis.

### A7. LOW — Constitution/Treasury/Reserve Fund governance address has a post-instantiation setup window

**Files:** `contracts/constitution/src/contract.rs`, `contracts/treasury/src/contract.rs`, `contracts/reserve-fund/src/contract.rs`

**Root cause:** all three contracts instantiate with `governance_address: None` and require a separate `SetupGovernanceAddress` call afterward. Between `instantiate` and that call, the contract has no governance authority configured. Genesis-time deployment sidesteps this via `scripts/generate_genesis.go`'s raw `ContractState` injection, but any other deployment path (e.g. a future non-genesis redeploy) would hit the open window.

**Fix:** add `governance_address: String` (required, non-optional) to each contract's `InstantiateMsg`, and set it directly in the `instantiate()` handler instead of via a separate `SetupGovernanceAddress` execute call. Remove the `SetupGovernanceAddress` variant from `ExecuteMsg` once this lands (check `contracts/*/tests` and `scripts/generate_genesis.go` for remaining references first — the genesis script already sets `governance_address` directly in its hand-crafted JSON, so this mainly requires deleting the now-dead code path).

**Verify:** `cd contracts/constitution && cargo test`, same for `treasury` and `reserve-fund` — confirm no test still relies on the two-step setup flow.

### A8. HIGH — No `SimGov` wrapper anywhere; all governance-gated simulation operations bypass the proposal lifecycle

**Files:** `chain/x/validator/simulation.go`, `chain/x/certification/simulation.go`, `chain/x/governance-ext/simulation.go`

**Root cause:** the plan requires governance-gated simulation operations to go through a helper that creates a proposal, advances block time past the voting period, votes with quorum, and only then executes — otherwise the simulation never tests the governance path, just the raw keeper mutation. Currently: `SimulateMsgUpdatePartitionScheme` (validator) doesn't even call a keeper mutation; `SimulateMsgUpdateCertificationParams` (certification) calls `k.SetParams` directly; and all 6 `governance-ext` sim operations call `k.ExecuteProposal` directly, skipping `x/gov` entirely. Simulation runs will never catch a real governance-flow bug this way.

**Fix:**
1. Create a shared helper (e.g. `chain/simutil/gov.go`) — `SimulateGovernanceProposal(...)` that: submits `MsgSubmitProposal` wrapping the target msg via `govKeeper.SubmitProposal`, deposits the min deposit, casts `MsgVote` from enough accounts to reach quorum with `VOTE_OPTION_YES`, and schedules/advances to trigger `x/gov`'s `EndBlocker` tally/execution. Match this to whatever simulation harness conventions `chain/app/simulation_test.go` already uses.
2. Rewrite `SimulateMsgUpdatePartitionScheme` (validator) to build a `MsgUpdatePartitionScheme` and wrap it via the new helper instead of any direct keeper call.
3. Rewrite `SimulateMsgUpdateCertificationParams` (certification) the same way, replacing the direct `k.SetParams` call.
4. Rewrite all 6 `governance-ext` operations (`SimulateMsgMigrateContracts`, `SimulateMsgUpdateValidatorSlot`, `SimulateMsgUpdateMilestone`, `SimulateMsgUpdateOracleOperator`, `SimulateMsgUpdateWitnessRegistry`, `SimulateMsgUpdateBridgeRelayerSet`) to submit via the helper instead of calling `k.ExecuteProposal` directly (that call should end up happening inside `x/gov`'s own proposal-execution handler once a proposal passes).
5. Leave `SimulateMsgFillValidatorSlot`, `SimulateMsgEjectValidator`, `SimulateDropValidatorAttestation`, `SimulateRestoreValidatorAttestation`, and the 4 oracle sim ops unchanged — they're correctly not governance-gated in the plan.

**Verify:** `cd chain && go test ./app/... -run TestFullAppSimulation -v` (or the existing sim entrypoint) — confirm the new operations show up going through `x/gov`'s `SubmitProposal`/`Vote`/tally, not a direct keeper call.

### A9. MEDIUM — Oracle daemon feeds, operator address, and price-source URLs are hardcoded

**File:** `oracle/main.go`

**Root cause:** the `feeds` map (lines ~166–173) hardcodes `"BTC_USD"`/`"ETH_USD"` with hardcoded `localhost:8080`–`8083` source URLs, when the plan requires feeds to come from on-chain `x/oracle` params via gRPC so governance can add/remove feeds without redeploying the daemon. `operator := "cosmosvaloper1x..."` (line 88) is a literal placeholder, not derived from the actual operator key — every commit this daemon submits is signed as a fake address that will never resolve to a real registered operator on a live chain.

**Fix:**
1. At startup, query the chain (via the existing gRPC connection) for on-chain feed configuration — add an `x/oracle` query (e.g. `QueryFeeds`) if one doesn't exist yet, returning configured feed IDs so the daemon isn't the source of truth for what feeds exist. Keep source *URLs* off-chain/config-driven (env var or config file) rather than hardcoded in Go source — feed *IDs* on-chain, source *URLs* in daemon config.
2. Replace the placeholder `operator` string with the real operator address derived from the HSM/key manager already in use (e.g. `hsm.GetPublicKey()` converted to a `cosmosvaloper1...` bech32 address via the SDK's address codec) — do not introduce a second, unrelated identity.
3. Add a fail-fast startup check: if the derived operator address isn't actually registered as an oracle operator on-chain, log a fatal error rather than silently submitting commits that will be rejected.

**Verify:** `cd oracle && go build ./...` then a short test-mode run — confirm feed IDs are logged as loaded from the chain query, not the hardcoded map, and the logged operator address matches the daemon's key-derived address.

### A10. MEDIUM — Oracle module has two O(N) hot paths with no index

**File:** `chain/x/oracle/keeper.go`

**Root cause:** `EndBlocker` (~line 303) iterates **all** stored commits every block to find ones whose reveal window has expired — scales linearly with total historical commit volume, with no expiry index. `GetRevealedValues` (lines 172–191) iterates the entire `RevealKeyPrefix` range doing string-splits/suffix checks per key instead of a composite key the KV store's prefix iterator can narrow directly to.

**Fix:**
1. Add a secondary expiry index keyed by `(expiryHeight, operator, feedID)` written alongside each commit; in `EndBlocker`, iterate only the index entries for the current (or ≤ current, to catch up after downtime) height instead of scanning every commit. Delete the index entry once resolved (revealed or slashed).
2. Restructure the reveal storage key as `RevealKeyPrefix + feedID + ":" + bigEndianRoundID + ":" + operator`, so `GetRevealedValues(feedID, roundID)` can construct an exact prefix and iterate only matching keys — no string splitting. Use big-endian numeric encoding, not decimal strings, so keys sort correctly.
3. Update any existing tests that construct reveal keys manually to match the new format.

**Verify:** `cd chain && go test ./x/oracle/... -v` — all existing commit-reveal/staleness tests must still pass. Add benchmarks (`BenchmarkEndBlockerLargeCommitSet`, `BenchmarkGetRevealedValuesLargeDataset`) confirming the new paths don't scale with total historical volume.

### A11. MEDIUM — Certification module has no Prometheus metrics; plan calls them mandatory

**Files:** `chain/x/certification/keeper.go`, `chain/x/certification/abci.go` (or wherever `EndBlocker` lives)

**Root cause:** the plan requires `attestation_coverage`, `bound_violations`, `degraded_mode_active`, and `rejection_count` as Prometheus counters/gauges. None exist anywhere in `x/certification/`. (The oracle daemon does have working Prometheus wiring on `:9200/metrics` as a reference pattern.)

**Fix:** use the Cosmos SDK's own `telemetry` package (check `chain/app/app.go` for the existing pattern rather than inventing a second HTTP server). In `EndBlocker`, call `telemetry.SetGauge(...)` for `rejection_count` and `degraded_mode_active`; in the liveness/attestation-window logic, emit `attestation_coverage` (fraction of validators attested per window) and increment `bound_violations` on threshold misses. Confirm these surface on the SDK's existing `/metrics` endpoint (enable `[telemetry]` in `chain/config/app.toml` if not already on) rather than assuming a new port is needed.

**Verify:** `cd chain && go build ./...`, start a local devnet node, then `curl localhost:<telemetry-port>/metrics | grep -E "rejection_count|degraded_mode_active|attestation_coverage|bound_violations"`.

### A12. MEDIUM — `x/governance-ext` Constitution check sends no proposal content to the contract

**File:** `chain/x/governance-ext/keeper.go` (~lines 120–136)

**Root cause:** the call to the Constitution contract sends `checkMsg` that serializes to `{"check_proposal":{}}` — an empty payload. The contract has no way to evaluate what's actually being proposed, so it can only ever answer the same way regardless of content, making the "compliance check" a no-op gate. Also, the sender is `sdk.AccAddress([]byte("govext_module"))` — a raw-bytes-as-address anti-pattern that should be `authtypes.NewModuleAddress(types.ModuleName)` instead, matching the fix already applied elsewhere (`x/settlement`).

**Fix:**
1. Extend the Constitution contract's accepted message (`contracts/constitution/src/msg.rs`) to take structured proposal content — at minimum the proposal type and a serialized summary of the action — so its rule-matching logic has something real to inspect. **This is a cross-cutting change with A1's Governance compliance-check fix — coordinate them together, they share the identical root problem.**
2. Update `chain/x/governance-ext/keeper.go` to build `checkMsg` from the actual incoming proposal message, not an empty struct literal.
3. Replace `sdk.AccAddress([]byte("govext_module"))` with `authtypes.NewModuleAddress(types.ModuleName)` (import `authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"` if needed).
4. Update `chain/x/governance-ext/keeper_test.go`'s constitution-check tests to assert the mock Wasm querier receives the actual proposal payload, not an empty object.

**Verify:** `cd chain && go test ./x/governance-ext/... -v`

### A13. LOW — EVM/CosmWasm "coexistence" test and precompile policy don't match the plan

**Files:** `e2e/evm_cosmwasm_test.go`, `chain/app/app.go` (precompile registration, ~lines 539–544)

**Root cause:** `TestCosmWasmEVMCoexistence`'s 5 "scenarios" are all self-contained assertions with no chain interaction (a string-equality check, integer arithmetic, a `big.NewInt` literal comparison, a string non-empty check, a slice-sort of literal strings) — none submit a `MsgExecuteContract` or `MsgEthereumTx`, so this test cannot catch a real coexistence bug. Separately, the plan's stated precompile policy is "no custom precompiles at mainnet launch; stubs with `// TODO: post-launch`," but the Oracle and Milestone precompiles are fully implemented and registered live in `NewApp` with no feature flag gating them off — they will ship live in the mainnet binary, contradicting the stated launch policy. (Their addresses, `0x0801`/`0x0802`, do correctly match the plan — that part is fine.)

**Fix:**
1. Rewrite `TestCosmWasmEVMCoexistence` to actually exercise both runtimes in the same test app instance: instantiate a trivial CosmWasm contract, submit a real `MsgExecuteContract` against it, submit a real `MsgEthereumTx` in the same block via the test app's `BeginBlock`/`DeliverTx`/`EndBlock` cycle, and assert on-chain state changed as expected from both.
2. For the precompile policy, either (a) get explicit sign-off that shipping the precompiles at launch is now the intended plan and update `implemention_plan.md`/`planned-vs-implemented.md` to reflect the changed decision, or (b) gate their registration in `app.go` behind a build tag/config flag defaulting to off, with a `// TODO: post-launch` comment, matching the original plan.

**Verify:** `cd e2e && go test ./... -run TestCosmWasmEVMCoexistence -v` — the rewritten test should fail loudly if either runtime's state change doesn't land, not just log `[PASS]` on tautological assertions.

### Phase A overall verify
```
cd contracts && cargo test --workspace
cd chain && go test ./...
cd e2e && go test ./... -run TestGovernance -v   # confirm a hostile/unauthorized proposal is now rejected
```
Do not proceed to Phase B until: a proposal submitted by a non-authorized sender is provably rejected (A1); a withdrawal mid-flight cannot re-enter Treasury/Reserve Fund (A2); and — at minimum before Phase E's external audit — A3/A4/A12/A9 have also landed, since those are the items an auditor would immediately flag as either untested or actively misleading in the tracking docs.

---

## Phase B — Chain identity consistency

**Goal:** every config file that references this chain's identity (Cosmos chain-id, EVM chain-id) must agree, or validators won't sign and wallets won't connect.

**Depends on:** nothing — can run in parallel with Phase A.

### Tasks
1. **Pick one canonical Cosmos chain-id for mainnet** (e.g. `sovereign-1` — already used in `genesis.json`, `genesis.prod.json`, `doc/mainnet/chain-registry.json`). Update every file currently using `sovereign-testnet-1` to match:
   - `infra/horcrux/horcrux-0.toml`, `horcrux-1.toml`, `horcrux-2.toml`, `horcrux.toml`
   - `infra/sovereign-k8s/5.chain-node/configmap-horcrux.yaml` (4 occurrences)
   - `frontend/config/wallets.json` (`keplr.chainId`)
   - `frontend/config/wallets.json` (`walletConnect.namespaces.cosmos.chains`, currently `cosmos:sovereign-testnet-1`)

   Keep `sovereign-testnet-1` only in whatever config is explicitly the testnet deployment — do not let mainnet and testnet configs share one values file without an environment-specific override.

2. **Pick one canonical EVM chain-id for mainnet.** The actual chain code uses `7777` (`chain/app/app.go:128`, `scripts/generate_genesis.go:49`). The frontend wallet config uses `9001` (`0x2329` in `frontend/config/wallets.json`, and WalletConnect's `eip155:9001`). Decide which is correct — if `7777` is what the running chain actually enforces, fix the frontend:
   - `frontend/config/wallets.json`: `metamaskSovereignEvm.chainId` → `0x1E61` (hex for 7777)
   - `frontend/config/wallets.json`: `walletConnect.namespaces.eip155.chains` → `eip155:7777`
   - `e2e/contracts/evm/Counter.deploy.json` and any other fixture referencing `9001` — check for stragglers with `grep -rn "9001" .`
   If you decide `9001` should be the real mainnet EVM chain-id instead, change `chain/app/app.go:128` and `scripts/generate_genesis.go:49` instead — but then rebuild and re-verify Phase A's contract/module tests, since chain-id changes affect signature domain separation on the EVM side.

3. Confirm BSC-side chain-ids are left untouched — `97` is BSC's real testnet id, and mainnet BSC is `56`. When you cut over from BSC testnet to BSC mainnet for the real bridge target, update `frontend/config/wallets.json`'s `metamaskBsc` block and the bridge relayer's target RPC/chain-id together, not separately.

### Verify
```
grep -rn "sovereign-testnet-1" --include="*.toml" --include="*.yaml" --include="*.json" .   # should return zero hits outside explicit testnet configs
grep -rn "0x2329\|9001" frontend/ e2e/   # should return zero hits, or all intentionally updated to 7777 (or vice versa)
```
Then actually start a local devnet with the corrected configs and connect MetaMask + Keplr manually — confirm both connect without a "wrong network" prompt.

---

## Phase C — Real genesis construction

**Goal:** replace the current devnet fixture genesis with a genesis that reflects real mainnet token allocation and passes the bridge supply invariant from the moment the chain starts.

**Depends on:** Phase A (contract addresses embedded in genesis must be built from the fixed contract code), Phase B (chain-id must be final before genesis is signed off).

### Tasks
1. **Remove the two test accounts.** `chain/genesis.json`'s `auth.accounts` and `bank.balances` currently hold exactly 2 accounts (`cosmos1m44j92r...`, `cosmos1dwkz0xn...`), each with identical large round-number balances. Replace these with the real mainnet allocation: validator genesis stake, treasury contract address, reserve fund contract address, and any public sale/airdrop distribution address — whatever your finalized tokenomics document says (see Phase F).
2. **Zero out `uwsov` (Bridge Minted Token) at genesis.** Its own denom metadata says "Bridge Minted Token" — it must start at `0` total supply and only increase when `x/bridge`'s relayers confirm a real BSC lock event. Currently genesis pre-mints 2,000,000,000,000,000 uwsov units with `bridge.cosmos_minted: 0`, which is a direct contradiction of the bridge invariant (`cosmos_minted_via_bridge + bsc_circulating = S`) documented in `implemention_plan.md`. Set `bank.supply` for `uwsov` to `0` and remove all `uwsov` entries from `bank.balances`.
3. **Set real `ucsov` and `aesov` supply figures** from the tokenomics document (Phase F), not the current placeholder `2000000000000000` / `2000000000000000000000000` round numbers.
4. **Set real bridge params** — `chain/genesis.json`'s `bridge.params` currently has placeholder addresses (`gnosis_safe_address: "cosmos1gs_addr"`, `lockbox_address: "0x1234567890123456789012345678901234567890"`, `circuit_breaker_address: "cosmos1cb_addr"`). These must be replaced with the real deployed Gnosis Safe address on BSC, the real lockbox contract address, and a real circuit-breaker key/address before mainnet.
5. **`chain/genesis.prod.json` is currently a broken stub, not a real production genesis.** Verified directly: it has **zero accounts** in `auth.accounts` / `bank.balances`, yet still declares the same non-zero `bank.supply` as the dev genesis (`aesov`/`ucsov`/`uwsov` amounts identical to `genesis.json`) — a bank module with supply but no balances to back it fails Cosmos SDK genesis validation on chain start. It also still has the same placeholder bridge addresses (`cosmos1gs_addr`, `cosmos1cb_addr`, the `0x1234...` lockbox) as the dev file. Separately, `docker-compose.yml` actually boots the chain node from **`genesis.dev.json`**, not `genesis.prod.json` — so right now there is no working "production" genesis at all, just an unused, invalid placeholder file with that name. Building the real mainnet genesis in this phase means making `genesis.prod.json` (or whatever file you designate as canonical) actually valid and actually wired into the node's startup config — not assuming the existing file is a real head start.
6. **Note on `planned-vs-implemented.md` item 10.1:** that document claims "700M Cosmos / 300M BSC" supply allocation is already hardcoded and invariant-verified in `chain/genesis.json`. This is stale/false as of this audit — the actual file has only 2 test accounts holding round placeholder numbers (see task 1 above), not a 700M/300M split. Do not trust that line in the tracking doc without re-checking `chain/genesis.json` directly after this phase's changes land — and correct the tracking doc's entry once the real allocation is in place, so it doesn't mislead the next reviewer.
7. **Fix genesis/artifact reproducibility.** Rebuild all CosmWasm contract `.wasm` files from the fixed Phase A source, regenerate `artifacts/checksums.txt`, then regenerate `chain/genesis.json`'s embedded code IDs from those freshly built artifacts using `scripts/generate_genesis.go` — do not hand-edit the embedded wasm bytes. Confirm the checksums in `chain/genesis.json` match `artifacts/checksums.txt` exactly.
8. Regenerate `chain/genesis.dev.json` and `chain/genesis.prod.json` consistently — decide whether `genesis.prod.json` is the actual mainnet genesis or a separate staging genesis, and document that distinction at the top of each file if it isn't already obvious.

### Verify
```
cd chain && go run ../scripts/generate_genesis.go --out genesis.json
sha256sum artifacts/*.wasm > /tmp/fresh-checksums.txt && diff /tmp/fresh-checksums.txt artifacts/checksums.txt
grep -A2 '"uwsov"' genesis.json   # bank.supply entry for uwsov must show "0"
```
A node started from this genesis must produce a `bank` module supply that satisfies the bridge invariant at height 1 (no uwsov before any bridge activity).

---

## Phase D — Real validator & key infrastructure

**Goal:** replace placeholder signer and multisig keys with real, independently-controlled infrastructure.

**Depends on:** Phase B (chain-id must be finalized), Phase C (genesis must be finalized to hand to validators).

### Tasks
1. **Cold multisig for governance/emergency-pause.** `doc/adr/adr-007-operational-security.md` lists 5 public keys that read as fabricated placeholders and describes a 3-of-5 threshold when the plan specifies 5-of-7. Generate real keys with real hardware security (YubiHSM, air-gapped signing devices, or a reputable custody provider), held by named, accountable individuals/entities, at the threshold your governance design actually requires. Update the ADR with the real threshold and real key fingerprints (not the raw public keys in a public doc — reference a key-fingerprint registry instead if publishing key material publicly is a concern).
2. **Horcrux remote-signer setup for validators.** `infra/horcrux/*.toml` currently point at `sovereign-testnet-1` (fixed in Phase B) — beyond that, confirm each horcrux node in the mainnet validator set uses a distinct, independently-secured key shard (not the same test keys copied across `horcrux-0/1/2.toml`).
3. **Recruit/onboard real, independent validator operators.** Genesis validators should not all be your own infrastructure — decentralization is both a technical requirement (liveness/censorship resistance) and something exchanges/listing sites will ask about. Document minimum validator requirements (hardware, uptime SLA, geographic/jurisdictional diversity) and a validator onboarding process.
4. **BSC-side bridge infrastructure.** The relayer's `gnosis_safe_address` and `lockbox_address` (fixed in Phase C with real values) need the actual Gnosis Safe deployed on BSC mainnet with real, independently-held signer keys meeting the `quorum_threshold: 3` in bridge params — confirm this threshold and the actual signer roster before launch.

### Verify
- Each validator can independently start a node from the Phase C genesis and sync.
- A test emergency-pause transaction, signed by the real cold multisig at its real threshold, executes correctly against the Constitution contract from Phase A.
- The BSC-side Gnosis Safe requires the documented quorum to approve a lock/release, verified with a real (small, testnet-value) transaction.

---

## Phase E — External security audit

**Goal:** catch what this internal review didn't.

**Depends on:** Phase A (no point auditing code you already know is broken), Phase C (genesis/contract artifacts should be final so the audit covers what actually ships).

### ⚠️ Correcting a false claim already in the repo
`planned-vs-implemented.md` marks Phase 9 (Security Audit) as 100% ✅. **This is fabricated.** `doc/ops/audit_engagement.json` only lists auditor firms as an aspirational OR-list with `"status": "pre-engaged"` (no firm is actually confirmed), and `e2e/phase_9_verification_test.go` merely asserts that JSON file's own fields are `true` — it does not contact any auditor or verify any real review happened. No audit report exists anywhere in the repo. Treat Phase 9 as **not started**, and correct `planned-vs-implemented.md`'s Phase 9 row to ❌ once you've confirmed this — do not let it keep reading ✅ for the next person who checks this file.

### Tasks
1. Engage an independent smart-contract/chain security audit firm for real (not a placeholder JSON entry) — get a signed engagement letter and a scheduled start date. Scope: `contracts/` (CosmWasm), `chain/x/*` (custom Go modules), `chain/app/abci.go` (vote extension logic), and the bridge relayer/lockbox contract on BSC.
2. Provide the auditors this plan (especially Phase A's numbered findings) as a starting reference — don't make them rediscover what's already documented; ask them to verify the fixes and find what's still missed.
3. Fix every finding rated medium or above before proceeding to Phase F. Track findings and fixes in a dated addendum file (e.g. `audit-findings-2026-XX.md`) rather than editing this plan.
4. Consider a public bug-bounty program (Immunefi or similar) running in parallel with or immediately after the formal audit, active before mainnet launch and continuing after.

### Verify
Auditor's final report shows no open critical/high findings. Get this in writing before proceeding.

---

## Phase F — Tokenomics & legal

**Goal:** have a real, defensible allocation and a legal green light before any public distribution or liquidity happens.

**Depends on:** nothing structurally, but Phase C's genesis needs this phase's output, so do this in parallel with Phases A–E, not after.

### Tasks
1. Write a tokenomics document covering: total supply per denom (`ucsov`/CSOV, `aesov`/ESOV — clarify whether ESOV is meant to be an independently valued asset or purely a gas-metering unit pegged to CSOV, since right now they have unrelated supply numbers with no stated peg), allocation breakdown (team, treasury, reserve fund, community/ecosystem, public sale if any), vesting schedules, and initial circulating supply at launch.
2. Get legal counsel review, specific to your jurisdiction and your target users' jurisdictions, on: whether `ucsov`/`aesov` constitute a security or regulated asset, whether public liquidity provisioning and any public sale trigger registration requirements, and what disclosures are required. This step cannot be completed by an implementing agent — flag it back to the project owner as a hard blocker requiring a licensed professional, don't attempt to self-certify compliance.
3. Update `chain/genesis.json`'s real balances (Phase C, task 1/3) to match this document exactly.

### Verify
Tokenomics doc exists, is internally consistent with the Phase C genesis file's actual numbers, and has documented legal sign-off (even if that sign-off is "cleared with conditions X, Y" — capture what it is, don't skip recording it).

---

## Phase G — Public testnet run

**Goal:** prove the fixed chain works under real, adversarial-ish conditions before it touches real money.

**Depends on:** Phases A, B, C, D (need fixed code, consistent chain-ids, real-shaped genesis, and real validator infra to test meaningfully).

### ⚠️ Correcting more false claims already in the repo
`planned-vs-implemented.md` marks Phase 6.9 ("testnet stable for 4 weeks before audit") and 6.6 ("≥5 external validators onboarded") as ✅. Both are backed only by planning/checklist documents (`doc/testnet/stability_checklist.md`, `doc/testnet/onboarding.md`) describing what should happen — there is no evidence in the repo of an actual multi-week public run or real independent operators having joined. Treat both as **not started**. This phase is where you actually do them for real.

### Tasks
1. Deploy a public testnet using the Phase C genesis process (with testnet-appropriate chain-id, per Phase B's environment separation) and the real validator operators recruited in Phase D (or a subset, if full mainnet validator count isn't ready yet).
2. Run for a minimum of several weeks. Exercise, with real transactions (not mocks): bridge lock/unlock round-trips against BSC testnet, oracle commit-reveal rounds under normal and adversarial (dropped-reveal, stale-feed) conditions, governance proposal submission/voting/execution end-to-end, and CosmWasm contract withdrawal flows under concurrent/malicious-looking load.
3. Open the testnet to external testers/community if possible — this surfaces integration issues (wallet connect, bridge UI, explorer accuracy) that internal testing misses.
4. Track every bug found during this phase the same way as Phase E's audit findings — dated addendum, not edits to this plan.

### Verify
No open critical/high bugs from the testnet run. Bridge invariant held throughout (spot-check via the explorer or a direct query) with no unexplained discrepancy between `cosmos_minted` and actual BSC-side locked balance.

---

## Phase H — Mainnet cutover

**Goal:** launch day execution.

**Depends on:** Phases A–G all complete.

### Tasks
1. Final genesis freeze — no further changes to `chain/genesis.json` after this point without restarting validator coordination.
2. Genesis ceremony: all mainnet validators independently verify the genesis file hash matches before gentx collection, matching standard Cosmos SDK launch practice.
3. Chain start — monitor block production, validator participation, and the dashboards from Phase D/E's monitoring setup (see also Phase I, item 6) continuously for the first 24–48 hours.
4. Have the Phase D cold multisig holders on standby in case an emergency pause is needed in the first hours/days — this is the highest-risk window.

### Verify
Chain produces blocks continuously past the first difficulty/validator-rotation cycle with no halts, and initial balances match the frozen genesis exactly (spot-check via explorer).

---

## Phase I — Post-launch: enabling real user activity

**Goal:** give the token and chain an actual, usable market and real functionality for end users.

**Depends on:** Phase H (chain must be live).

### Tasks
1. **Liquidity.** Bridge project-treasury-allocated tokens to BSC mainnet and add liquidity on PancakeSwap (paired with BNB or a stablecoin), and/or deploy a native AMM on the chain's own EVM layer. Fund with a real amount (a few thousand dollars minimum) — see the earlier tokenomics-driven allocation for how much is earmarked for this.
2. **Lock or burn LP tokens** received from step 1, for a minimum of 6–12 months, using a reputable locker (e.g. Team Finance, UNCX) or a burn address. Publish the lock transaction publicly.
3. **Wallet integration guides** — publish user-facing docs for connecting Keplr (Cosmos side) and MetaMask (EVM side, using the Phase B-corrected chain-id), plus the WalletConnect config.
4. **Bridge UI go-live** — confirm the frontend bridge flow works end-to-end against the real BSC mainnet lockbox/Gnosis Safe from Phase D.
5. **Listing applications** — submit to CoinGecko and CoinMarketCap once the DEX pool from step 1 is live and has real trading volume.
6. **Monitoring in production** — Prometheus/Grafana dashboards (from Phase A11's certification metrics, plus bridge invariant and oracle staleness) must be actively watched, with alerting configured, not just present.
7. **First real governance proposal** — exercise the now-fixed governance flow (Phase A) with a real, low-stakes proposal to prove the mechanism works publicly before anything high-value goes through it.
8. **Bug bounty continuation** — keep the Phase E bounty program running indefinitely post-launch.
9. **Community/marketing** — website, docs, and social channels live before or at the same time as liquidity goes live, not after — users need somewhere to verify the project is real before they trade.

### Verify
Real, non-team wallets holding the token, a live DEX pool with real trading volume, and at least one governance proposal executed successfully post-launch by a real (non-test) proposer.

---

## Summary checklist (for quick status tracking)

- [ ] Phase A — Contract/module security fixes (A1–A13 below) complete and tested
- [ ] Phase B — Chain-id consistency fixed across horcrux, k8s, frontend, chain code
- [ ] Phase C — Real genesis built (real accounts, `uwsov=0`, real bridge params, reproducible artifacts) — also correct the false "700M/300M" claim in `planned-vs-implemented.md` §10.1
- [ ] Phase D — Real cold multisig, real validator set, real BSC-side Gnosis Safe
- [ ] Phase E — **Real** external audit complete (not the fabricated Phase 9 self-check already in the repo), no open critical/high findings
- [ ] Phase F — Tokenomics documented, legal review complete
- [ ] Phase G — **Real** public testnet run (weeks, real external validators, real adversarial scenarios) with no open critical/high bugs — not just the planning checklists already in the repo
- [ ] Phase H — Mainnet genesis ceremony and cutover
- [ ] Phase I — Liquidity live, LP locked/burned, wallets documented, bridge UI live, listings submitted, monitoring active, first governance proposal executed

Do not check off a phase based on code being written, a JSON config asserting a status field, or an internal test that only re-checks that same JSON file — only on its verification step actually passing against something real and external.

---

## Appendix — Full whole-project review findings (beyond Phases 2–3)

This audit originally covered Phase 2 and Phase 3 of `planned-vs-implemented.md` only (now fully inlined into Phase A above). This update reviewed all 10 phases end-to-end. Findings not already covered by Phases A–I above:

- **Phases 0, 1, 5, 7 (governance/ADR docs, chain scaffold, off-chain CQRS backend, frontend):** spot-checked and largely consistent with what the code actually does — no fabricated claims found here beyond the chain-id and genesis issues already captured in Phases B/C above.
- **Phase 4 (BSC bridge):** the Solidity (`LockBox.sol`) and Go (`x/bridge` keeper) code for pause/unpause, rate limiting, and supply-cap enforcement is genuinely implemented, not stubbed — this is one of the stronger parts of the codebase. Still depends on Phase D's real Gnosis Safe signer roster and Phase A's governance fix (relayer set is governance-managed, so an unauthenticated governance exploit would let an attacker rewrite the relayer set).
- **Phase 8 (pre-audit hardening):** lint/vuln/race-condition/invariant-registration claims read as genuine engineering work (specific tools, specific invariant names). The one item to distrust is 8.5 ("internal penetration test... automated... in `e2e/phase_8_verification_test.go`") — an internal automated test suite is not a penetration test in the security-industry sense; don't cite it as one to auditors or the public.
- **Phase 9 (security audit) and Phase 6.6/6.9 (testnet decentralization/stability):** confirmed fabricated/unverified, detailed in Phases E and G above. These are the two most important false claims in the entire tracking document, since they are exactly the claims a listing exchange or a cautious user would ask about first.
- **Recommendation:** once Phases A–I are genuinely complete, do a pass over `planned-vs-implemented.md` itself and correct every ✅ this review downgraded (9.1–9.5, 6.6, 6.9, 10.1) — an internal tracking document that overclaims completion is itself a risk, since it will mislead the next person (including a future version of this agent) into skipping work that was never actually done.

### A systemic pattern found across the whole repo — not just Phase 9

Checking every `e2e/phase_N_verification_test.go` file for how it actually verifies its claims: Phases 2, 3, and 4's tests exercise real keeper/contract logic directly (calling real state-transition code, real signature checks) — these are genuinely meaningful tests. But Phases 0, 1, 5, 6, 7, 9, and 10's "verification" tests are dominated by parsing the project's own YAML/JSON docs and config files and asserting their fields are self-consistent, with almost no calls to real infrastructure (no `grpc.Dial`, `ethclient.Dial`, `sql.Open`, or subprocess execution against a live service in most of them). That means for those phases, "✅ verified in e2e tests" in `planned-vs-implemented.md` typically means "a config file exists and parses", not "this was proven to work against a running system." Re-verify anything from Phases 0/1/5/6/7/10 the same way before relying on it, the same way Phase 9 turned out to be fake.

### Additional finding this pass: `chain/genesis.prod.json` is a broken, unused stub
See Phase C, task 4c above — it has 0 accounts but non-zero declared token supply (would fail Cosmos SDK genesis validation as-is), still has the same placeholder bridge addresses as the dev genesis, and the running chain (per `docker-compose.yml`) actually boots from `genesis.dev.json`, not this file. There is currently no working, distinct production genesis — the filename exists, the content does not.

### Additional finding this pass: frontend silently falls back to mock data
`frontend/app/page.tsx`'s bridge status/tier logic has multiple `// Fallback to default/mock` branches that activate when a real API call fails. This is a UX/trust risk for a real launch, separate from the fund-theft bugs: if the backend is briefly unavailable, a user could see fabricated-looking bridge/tier data with no visible "this is a fallback" indicator. Before public launch, either surface a clear "data unavailable" state instead of silent mock fallback, or make sure the mock values can never be mistaken for live chain data (e.g. visibly greyed out with a retry prompt).

### Tokenomics & liquidity coverage confirmation
Tokenomics is covered in **Phase F** above (allocation, vesting, ESOV/CSOV peg-or-not decision, legal review — explicitly required before Phase C's genesis balances can be finalized). Liquidity is covered in **Phase I**, tasks 1–2 (DEX pool funding, LP lock/burn) and task 5 (CoinGecko/CoinMarketCap listing, which requires live DEX volume first). Neither can be completed by code alone — Phase F's legal review and Phase I's actual liquidity provisioning require the project owner's real funds and real legal counsel, not just engineering work; the plan flags both as owner-action items rather than agent-implementable tasks.
