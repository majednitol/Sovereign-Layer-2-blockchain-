# Phase 1 — AI Agent Instructions
## Code Update / Removal Guide for Phase 1: Chain Scaffold & Genesis Configuration

**Repository branch:** `phase-1`  
**Purpose:** This file tells you exactly what to fix, what to leave alone, and what to remove so that Phase 1 is fully complete and clean.

---

## How to Read This File

| Section | What to do |
|---------|-----------|
| **KEEP AS-IS** | Code is correct. Do not touch. |
| **FIX** | Code is incomplete or wrong. Apply the exact change described. |
| **REMOVE** | Code or file should not exist in Phase 1 scope. Delete it. |

---

## KEEP AS-IS — All 20 Fully Implemented Phase 1 Tasks

The following 20 tasks are correctly implemented. Do not modify them.

| Task | What Is Implemented | Key Files |
|------|-------------------|-----------|
| 1.1 | `cosmos/evm v0.7.0` pinned in go.mod | `chain/go.mod` |
| 1.2 | `skip-mev/feemarket` removed from go.mod | `chain/go.mod` |
| 1.3 | `ibc-go/v11` in go.mod | `chain/go.mod` |
| 1.4 | `x/feemarket` keeper wired in `app.go` | `chain/app/app.go` |
| 1.5 | `x/vm` keeper wired in `app.go` | `chain/app/app.go` |
| 1.6 | `x/erc20` keeper wired in `app.go` | `chain/app/app.go` |
| 1.7 | Module order: feemarket→vm→erc20 in BeginBlocker/EndBlocker/InitGenesis | `chain/app/app.go` |
| 1.9 | Ante handler replaced with `evmante.NewAnteHandler` + authz wrapper | `chain/app/app.go` (`setAnteHandler`) |
| 1.10 | `/cosmos.evm.vm.v1.MsgEthereumTx` in authz blocked map | `chain/app/app.go` (`setAnteHandler`) |
| 1.11 | x/vm genesis params: ChainID=7777, EvmDenom="atoken", EnableCreate, AllowUnprotectedTxs=false | `scripts/generate_genesis.go`, `chain/config/app.toml` |
| 1.12 | x/feemarket genesis params: NoBaseFee=false, ElasticityMultiplier=2, EnableHeight=0 | `scripts/generate_genesis.go` |
| 1.13 | x/erc20 genesis: native token pair utoken ↔ 0x…0001 | `scripts/generate_genesis.go` |
| 1.14 | app.toml [json-rpc]: port 8545/8546, namespaces, gas-cap | `chain/config/app.toml` |
| 1.15 | `GetEqualizedValidatorPower` returns 1,000,000 for active validators | `chain/app/staking_compatibility.go` |
| 1.16 | `AllocateTokens` equal-slot reward split | `chain/app/staking_compatibility.go` |
| 1.17 | `OverrideHistoricalInfo` + `GetEqualizedValidatorUpdates` | `chain/app/staking_compatibility.go` |
| 1.18 | Upgrade handler `v1.0.0` scaffold (see FIX below for one bug) | `chain/app/upgrades.go` |
| 1.21 | All 5 native authz blocked types (bridge/oracle/settlement) | `chain/app/app.go` (`setAnteHandler`) |
| 1.22 | `scripts/generate_genesis.go` invariant verification script | `scripts/generate_genesis.go` |
| 1.23 | Genesis supply invariants: S−C + bridge escrow = 1B TOKEN | `scripts/generate_genesis.go`, `e2e/phase_1_verification_test.go` |

---

## FIX — 3 Incomplete Items

---

### FIX 1 — Task 1.8: Wire `ibcFeeKeeper` (ICS-29 fee middleware) in `app.go`

**Problem:** `ibcFeeKeeper` is missing entirely. The planned task requires wiring `ibcKeeper` + `ibcTransferKeeper` + `ibcFeeKeeper`. The first two exist; the third does not.

**Severity:** Low (does not break devnet; blocks ICS-29 packet fee incentivization)

**What to add in `chain/app/app.go`:**

1. Add import:
```go
ibcfeekeeper "github.com/cosmos/ibc-go/v11/modules/apps/fee/keeper"
ibcfeetypes "github.com/cosmos/ibc-go/v11/modules/apps/fee/types"
```

2. Add store key in `NewKVStoreKeys(...)` call:
```go
ibcfeetypes.StoreKey,
```

3. Add keeper field to the `App` struct:
```go
IBCFeeKeeper ibcfeekeeper.Keeper
```

4. Initialize the keeper after `IBCKeeper` is initialized:
```go
app.IBCFeeKeeper = ibcfeekeeper.NewKeeper(
    appCodec,
    runtime.NewKVStoreService(keys[ibcfeetypes.StoreKey]),
    app.IBCKeeper.ChannelKeeper,
    app.IBCKeeper.ChannelKeeper,
    app.IBCKeeper.PortKeeper,
    app.AccountKeeper,
    app.BankKeeper,
)
```

5. Register the module in `module.NewManager(...)`:
```go
ibcfee.NewAppModule(app.IBCFeeKeeper),
```

6. Add `ibcfeetypes.ModuleName` to `SetOrderBeginBlockers`, `SetOrderEndBlockers`, and `SetOrderInitGenesis` lists (place alongside other IBC modules).

7. Update the IBC router to wrap the transfer stack with the fee middleware:
```go
// Replace the existing plain transfer stack with a fee-wrapped one:
transferIBCModule := ibctransfer.NewIBCModule(app.TransferKeeper)
feeTransferStack := ibcfee.NewIBCMiddleware(transferIBCModule, app.IBCFeeKeeper)
ibcRouter.AddRoute(ibctransfertypes.ModuleName, feeTransferStack)
```

---

### FIX 2 — Task 1.19: Regenerate `chain/genesis.json` with `CodeUploadAccess = Nobody`

**Problem:** `scripts/generate_genesis.go` correctly sets `CodeUploadAccess = AccessTypeNobody` but the committed `chain/genesis.json` still has `"permission": "Everybody"`. The file is out of sync with the script.

**Severity:** Low (wrong value is live in committed genesis; must fix before any testnet launch)

**What to do:**

Run the genesis generation script to overwrite `chain/genesis.json`:

```bash
go run scripts/generate_genesis.go --out chain/genesis.json --chain-id sovereign-1
```

After running, verify that `chain/genesis.json` contains:
```json
"code_upload_access": {
  "permission": "Nobody"
}
```

Do **not** edit `chain/genesis.json` by hand. Always regenerate via the script.

---

### FIX 3 — Task 1.18: Fix `configurator()` returning `nil` in `chain/app/upgrades.go`

**Problem:** `configurator()` returns `nil`. If the `v1.0.0` upgrade height is ever reached on-chain, `mm.RunMigrations(ctx, nil, fromVM)` will panic.

**Severity:** Medium (runtime panic on upgrade execution)

**Current broken code in `chain/app/upgrades.go`:**
```go
func configurator() module.Configurator {
    return nil  // TODO: implement
}
```

**Fix — replace with a real configurator stored on the App struct:**

In `chain/app/app.go`, add a field and store the configurator after `ModuleManager` is set:
```go
// Add field to App struct:
configurator module.Configurator

// After mm is assigned, register services:
app.configurator = module.NewConfigurator(app.appCodec, app.MsgServiceRouter(), app.GRPCQueryRouter())
app.ModuleManager.RegisterServices(app.configurator)
```

In `chain/app/upgrades.go`, replace `configurator()` with a reference to the field:
```go
// Remove the standalone configurator() function entirely.
// In RegisterUpgradeHandlers, change the handler to:
app.UpgradeKeeper.SetUpgradeHandler(UpgradeNameV1, func(ctx context.Context, plan upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
    sdkCtx := sdk.UnwrapSDKContext(ctx)
    return app.ModuleManager.RunMigrations(sdkCtx, app.configurator, fromVM)
})
```

> Note: If `RegisterUpgradeHandlers` is called before `ModuleManager.RegisterServices`, you may need to reorder initialization in `NewApp`. `RegisterServices` must be called before `RegisterUpgradeHandlers`.

---

## NOT REQUIRED IN PHASE 1 — Do Not Implement

The following items appear in the codebase as `.gitkeep` placeholders. **Do not add any code to these directories** — they belong to Phase 2 and later.

| Directory | Phase | Reason Not Phase 1 |
|-----------|-------|--------------------|
| `chain/x/certification/` | Phase 2 | x/certification keeper not planned until Phase 2 |
| `chain/x/oracle/` | Phase 2 | Oracle commit-reveal keeper is Phase 2 |
| `chain/x/bridge/` | Phase 2 | Bridge Cosmos module is Phase 2 |
| `chain/x/milestone/` | Phase 2 | Milestone keeper is Phase 2 |
| `chain/x/settlement/` | Phase 2 | Settlement keeper is Phase 2 |
| `chain/x/validator/` | Phase 2 | Validator registry is Phase 2 |
| `chain/x/governance-ext/` | Phase 2 | Extended governance keeper is Phase 2 |
| `chain/x/vm/precompiles/` | Phase 3 | EVM precompiles are Phase 3 |
| `relayer/` | Phase 2 | Go relayer daemon is Phase 2 |
| `oracle/` | Phase 2 | Oracle aggregator is Phase 2 |
| `backend/` | Phase 3 | Backend API is Phase 3 |
| `explorer-indexer/` | Phase 3 | Explorer indexer is Phase 3 |
| `frontend/` | Phase 4 | Frontend dApp is Phase 4 |
| `explorer/` | Phase 4 | Explorer UI is Phase 4 |

---

## KNOWN DOCUMENTATION MISALIGNMENTS — Fix Docs, Not Code

These are documentation errors. The code is correct; the documents need updating.

### Doc Fix 1 — EVM Chain ID

**Problem:** `doc/adr/adr-009-evm-chain-id.md` and `doc/governance/genesis_parameters.md` reference EVM Chain ID `9001`. The code uses `7777`.

**File to fix:** `doc/adr/adr-009-evm-chain-id.md` and `doc/governance/genesis_parameters.md`

**Change:** Replace all references to `9001` with `7777`. Update any chainlist.org registration notes to reflect the actual registered ID.

---

### Doc Fix 2 — Bech32 Prefix

**Problem:** `doc/governance/genesis_parameters.md` states the bech32 prefix is `sov` / `sovpub` / `sovvaloper` / `sovvalcons`. However, `chain/cmd/chaind/main.go` calls `cfg.SetBech32PrefixForAccount(sdk.Bech32PrefixAccAddr, ...)` which uses SDK default values — no literal `"sov"` is assigned anywhere in the code.

**Options (pick one):**

**Option A — Fix the code** (recommended):  
In `chain/cmd/chaind/main.go` inside `setupSDKConfig()`, add explicit assignments before the `cfg.Set*` calls:
```go
sdk.Bech32PrefixAccAddr    = "sov"
sdk.Bech32PrefixAccPub     = "sovpub"
sdk.Bech32PrefixValAddr    = "sovvaloper"
sdk.Bech32PrefixValPub     = "sovvaloperpub"
sdk.Bech32PrefixConsAddr   = "sovvalcons"
sdk.Bech32PrefixConsPub    = "sovvalconspub"
```

**Option B — Fix the documentation:**  
Update `doc/governance/genesis_parameters.md` to state the prefix is `cosmos` (SDK default) until explicitly set.

---

## Task 1.20 — WasmKeeper / governance-ext Wiring (Deferred — No Action Required in Phase 1)

`ConstitutionContractAddr` is defined in `chain/app/wasm.go`. Wiring it into `GovKeeper` requires the `x/governance-ext` keeper to exist first. Since `chain/x/governance-ext/` is a `.gitkeep` stub, this wiring is **correctly deferred to Phase 2**. No action needed for Phase 1.

---

## Summary of Agent Actions

| Priority | Action | File(s) |
|----------|--------|---------|
| 🔴 Medium | Fix `configurator()` nil — prevents upgrade panic | `chain/app/app.go`, `chain/app/upgrades.go` |
| 🟡 Low | Wire `ibcFeeKeeper` (ICS-29) in app.go | `chain/app/app.go` |
| 🟡 Low | Regenerate `chain/genesis.json` via script (CodeUploadAccess=Nobody) | `scripts/generate_genesis.go` → `chain/genesis.json` |
| 📝 Doc | Correct EVM Chain ID from 9001 → 7777 in ADR-009 and genesis_parameters.md | `doc/adr/adr-009-evm-chain-id.md`, `doc/governance/genesis_parameters.md` |
| 📝 Doc | Add explicit Bech32 `sov` prefix assignment in `setupSDKConfig()` **or** correct the doc | `chain/cmd/chaind/main.go` **or** `doc/governance/genesis_parameters.md` |
| ✅ None | All other 20 Phase 1 tasks — fully implemented, leave unchanged | — |
