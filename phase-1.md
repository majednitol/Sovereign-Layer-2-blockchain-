# Sovereign L1 Blockchain — Client Deliverable

> **Scope:** Phase 1 (Chain Scaffold & Genesis Configuration) and the complete Faucet (backend service + explorer UI).
> All other phases, modules, and implementation details are withheld and remain proprietary.
>
> **Chain:** Sovereign L1 · Cosmos chain ID `sovereign-testnet-1` · EVM chain ID `7777`
> **Token denominations:** `ucsov` (6 dec, staking/gov) · `aesov` (18 dec, EVM gas) · `uwsov` (6 dec, bridge-minted)
> **Source verified against:** `chain/app/app.go`, `chain/app/staking_compatibility.go`, `chain/app/upgrades.go`, `chain/app/wasm.go`, `chain/config/app.toml`, `scripts/generate_genesis.go`, `backend/module/faucet/main.go`, `backend/module/faucet/main_test.go`, `explorer/app/faucet/page.tsx`

---

## Table of Contents

1. [Phase 1 — Chain Scaffold & Genesis Configuration](#phase-1--chain-scaffold--genesis-configuration)
   - [1.1 Dependency & Version Pinning](#11-dependency--version-pinning)
   - [1.2 NewApp() — Global Power Reduction Override](#12-newapp--global-power-reduction-override)
   - [1.3 Interface Registry & Custom Signers](#13-interface-registry--custom-signers)
   - [1.4 KV Store Keys & Object Store Keys](#14-kv-store-keys--object-store-keys)
   - [1.5 EVM Stack Keeper Wiring (feemarket → vm → erc20)](#15-evm-stack-keeper-wiring-feemarket--vm--erc20)
   - [1.6 EVM Custom Precompile Registration](#16-evm-custom-precompile-registration)
   - [1.7 IBC Keeper Wiring & erc20 IBC Middleware](#17-ibc-keeper-wiring--erc20-ibc-middleware)
   - [1.8 CosmWasm Keeper & Reserved Contract Addresses](#18-cosmwasm-keeper--reserved-contract-addresses)
   - [1.9 Custom Module Keepers](#19-custom-module-keepers)
   - [1.10 governance-ext Keeper & Constitution Wiring](#110-governance-ext-keeper--constitution-wiring)
   - [1.11 Module Registration & Block Hook Ordering](#111-module-registration--block-hook-ordering)
   - [1.12 Ante Handler — cosmos/evm Unified Handler](#112-ante-handler--cosmosevm-unified-handler)
   - [1.13 authz Blocked Message Types](#113-authz-blocked-message-types)
   - [1.14 EVM Mempool Configuration](#114-evm-mempool-configuration)
   - [1.15 Staking Compatibility Layer](#115-staking-compatibility-layer)
   - [1.16 EndBlocker — Equalized Validator Updates](#116-endblocker--equalized-validator-updates)
   - [1.17 Upgrade Handler (v1.0.0)](#117-upgrade-handler-v100)
   - [1.18 app.toml — JSON-RPC Configuration](#118-apptoml--json-rpc-configuration)
   - [1.19 Genesis Script & Supply Invariants](#119-genesis-script--supply-invariants)
   - [1.20 Genesis Parameters — Full Module Configuration](#120-genesis-parameters--full-module-configuration)
   - [Phase 1 Task Checklist](#phase-1-task-checklist)
2. [Faucet — Backend Service](#faucet--backend-service)
   - [Architecture Overview](#architecture-overview)
   - [Configuration & Startup](#configuration--startup)
   - [HTTP Handler — Request Flow](#http-handler--request-flow)
   - [Address Normalization](#address-normalization)
   - [Per-Address Cooldown](#per-address-cooldown)
   - [Serialized Tx Dispatch & Retry Logic](#serialized-tx-dispatch--retry-logic)
   - [Sequence Number Resolution](#sequence-number-resolution)
   - [CORS Policy](#cors-policy)
   - [Known API Gap — Response Fields](#known-api-gap--response-fields)
   - [Test Coverage](#test-coverage)
3. [Faucet — Explorer UI](#faucet--explorer-ui)
   - [Component State Variables](#component-state-variables)
   - [Wallet Auto-Fill](#wallet-auto-fill)
   - [Client-Side Cooldown Countdown](#client-side-cooldown-countdown)
   - [Request Flow (UI Side)](#request-flow-ui-side)
   - [State Rendering](#state-rendering)
   - [Faucet Info Sidebar](#faucet-info-sidebar)

---

## Phase 1 — Chain Scaffold & Genesis Configuration

Phase 1 establishes the entire chain foundation: pinning all external dependencies, wiring 3 EVM modules + IBC + CosmWasm inside `app.go`, installing the unified ante handler, setting up the equalized staking compatibility layer, and generating the genesis file with verified supply invariants.

---

### 1.1 Dependency & Version Pinning

All module versions are pinned in `chain/go.mod` and aligned across the 7-module Go workspace (`go.work`).

| Dependency | Pinned Version | Notes |
|---|---|---|
| `github.com/cosmos/evm` | `v0.7.0` | EVM runtime (`x/vm`), fee market, erc20 |
| `github.com/cosmos/cosmos-sdk` | `v0.54.3` | Cosmos SDK base |
| `github.com/cometbft/cometbft` | `v0.39.3` | Consensus engine |
| `github.com/cosmos/ibc-go/v11` | `v11.0.0` | Upgraded from v8 |
| `github.com/CosmWasm/wasmd` | `v0.70.2` | CosmWasm runtime |
| `github.com/ethereum/go-ethereum` | `v1.17.0` | EVM primitives |

`skip-mev/feemarket` was fully removed from `go.mod`. The project uses `cosmos/evm/x/feemarket` exclusively (ADR-010).

---

### 1.2 NewApp() — Global Power Reduction Override

The very **first line** of `NewApp()` before any keeper construction overrides the Cosmos SDK's global power reduction constant:

```go
func NewApp(...) *App {
    sdk.DefaultPowerReduction = math.NewInt(1_000_000)
    ...
}
```

**Why this matters:** Cosmos SDK uses `DefaultPowerReduction` to convert token amounts to consensus voting power (e.g. `power = tokens / DefaultPowerReduction`). Setting it to `1,000,000` means each validator active slot is treated as exactly 1 unit of consensus power when the equalized power value is also `1,000,000`. This ensures the equalized power model is coherent across all SDK modules (distribution, slashing, IBC historical info) without requiring changes to each one.

---

### 1.3 Interface Registry & Custom Signers

Two EVM message types require custom signer-extraction functions because their signers are derived differently from the standard Cosmos signing flow. Both are registered at app construction:

```go
signingOptions := signing.Options{
    AddressCodec:          evmaddress.NewEvmCodec(sdk.Bech32PrefixAccAddr),
    ValidatorAddressCodec: evmaddress.NewEvmCodec(sdk.Bech32PrefixValAddr),
    CustomGetSigners: map[protoreflect.FullName]signing.GetSignersFunc{
        evmtypes.MsgEthereumTxCustomGetSigner.MsgType:     evmtypes.MsgEthereumTxCustomGetSigner.Fn,
        erc20types.MsgConvertERC20CustomGetSigner.MsgType: erc20types.MsgConvertERC20CustomGetSigner.Fn,
    },
}
```

The `evmaddress.NewEvmCodec` ensures bech32 addresses are decoded/encoded using the EVM-compatible codec for both regular and validator addresses.

---

### 1.4 KV Store Keys & Object Store Keys

Two distinct store key sets are allocated:

**KV Store Keys** (persistent, committed to state tree):

```go
keys := storetypes.NewKVStoreKeys(
    // Standard SDK
    authtypes.StoreKey, banktypes.StoreKey, stakingtypes.StoreKey,
    distrtypes.StoreKey, slashingtypes.StoreKey, govtypes.StoreKey,
    upgradetypes.StoreKey, feegrant.StoreKey, authzkeeper.StoreKey,
    consensusparamtypes.StoreKey,
    // IBC
    ibcexported.StoreKey, ibctransfertypes.StoreKey,
    // Cosmos EVM
    evmtypes.StoreKey, feemarkettypes.StoreKey, erc20types.StoreKey,
    // CosmWasm
    wasm.StoreKey,
    // Custom Modules
    validator.StoreKey, certification.StoreKey, oracle.StoreKey,
    milestone.StoreKey, settlement.StoreKey, bridge.StoreKey,
)
```

**Object Store Keys** (transient per-block data, reset on every `Commit`):

```go
oKeys := storetypes.NewObjectStoreKeys(banktypes.ObjectStoreKey, evmtypes.ObjectKey)
```

`evmtypes.ObjectKey` is used by the EVM keeper for per-block bloom filter accumulation and gas tracking. Both store sets are mounted at the end of `NewApp()`:

```go
app.MountKVStores(keys)
app.MountObjectStores(oKeys)
```

---

### 1.5 EVM Stack Keeper Wiring (feemarket → vm → erc20)

Three EVM keepers are initialized in strict dependency order. **feemarket must exist before vm; erc20 requires vm but its pointer is passed to vm at construction so it must be declared first.**

#### FeeMarketKeeper (EIP-1559)

```go
app.FeeMarketKeeper = feemarketkeeper.NewKeeper(
    appCodec,
    authtypes.NewModuleAddress(govtypes.ModuleName), // authority
    keys[feemarkettypes.StoreKey],
)
```

#### EVMKeeper (x/vm)

```go
app.EVMKeeper = evmkeeper.NewKeeper(
    appCodec,
    keys[evmtypes.StoreKey],
    oKeys[evmtypes.ObjectKey],          // object store for per-block EVM state
    nil,                                // nonTransientKeys (not needed without transient stores)
    authtypes.NewModuleAddress(govtypes.ModuleName),
    app.AccountKeeper,
    app.BankKeeper,
    app.StakingKeeper,
    app.FeeMarketKeeper,
    app.ConsensusParamsKeeper,
    &app.Erc20Keeper,                   // pointer — Erc20Keeper wired before this call
    EVMChainID,                         // 7777
    "",                                 // tracer (empty = no tracing)
)
```

> **Note:** `&app.Erc20Keeper` is passed as a pointer before `app.Erc20Keeper` is fully initialized. The Erc20Keeper is initialized immediately after, and the pointer ensures EVMKeeper has the final reference.

#### Erc20Keeper (x/erc20)

```go
app.Erc20Keeper = erc20keeper.NewKeeper(
    keys[erc20types.StoreKey],
    appCodec,
    authtypes.NewModuleAddress(govtypes.ModuleName),
    app.AccountKeeper,
    app.BankKeeper,
    app.EVMKeeper,
    app.StakingKeeper,
    app.TransferKeeper,
)
```

---

### 1.6 EVM Custom Precompile Registration

After both `EVMKeeper` and the custom module keepers are initialized, two static precompiles are registered at fixed addresses:

```go
app.EVMKeeper.RegisterStaticPrecompile(
    precompiles.OraclePrecompileAddress,
    precompiles.NewOraclePrecompile(app.OracleKeeper),
)
app.EVMKeeper.RegisterStaticPrecompile(
    precompiles.MilestonePrecompileAddress,
    precompiles.NewMilestonePrecompile(app.MilestoneKeeper),
)
```

| Address | Module | Exposed Methods |
|---|---|---|
| `0x0000000000000000000000000000000000000101` | `x/oracle` | `getLatestPrice(feedID)`, `isFeedStale(feedID)` |
| `0x0000000000000000000000000000000000000102` | `x/milestone` | `getMilestone(id)` |

These are registered as `active_static_precompiles` in genesis (see §1.20).

---

### 1.7 IBC Keeper Wiring & erc20 IBC Middleware

Three IBC keepers are initialized: `IBCKeeper`, `TransferKeeper`, and the IBC Tendermint light client module.

```go
app.IBCKeeper = ibckeeper.NewKeeper(...)

app.TransferKeeper = transferkeeper.NewKeeper(
    appCodec,
    nil,   // addressCodec — set nil for scaffold; will wire EvmCodec in production
    runtime.NewKVStoreService(keys[ibctransfertypes.StoreKey]),
    app.IBCKeeper.ChannelKeeper,
    app.MsgServiceRouter(),
    app.AccountKeeper,
    app.BankKeeper,
    authAddr,
)
```

> **⚠️ Known scaffold gap:** `TransferKeeper` is initialized with `nil` for `addressCodec`. This must be replaced with `evmaddress.NewEvmCodec(...)` before production to ensure EVM-format addresses can receive IBC transfers.

**erc20 IBC Middleware** — the transfer IBC module stack is wrapped so that incoming IBC tokens can be auto-converted to their ERC-20 representation:

```go
var transferStack porttypes.IBCModule
transferStack = transfer.NewIBCModule(app.TransferKeeper)
transferStack = erc20.NewIBCMiddleware(app.Erc20Keeper, transferStack) // ← wraps transfer

ibcRouter := porttypes.NewRouter()
ibcRouter.AddRoute(ibctransfertypes.ModuleName, transferStack)
app.IBCKeeper.SetRouter(ibcRouter)
```

**Tendermint light client module** is separately routed so foreign chains' IBC clients can verify Sovereign L1 headers:

```go
clientKeeper := app.IBCKeeper.ClientKeeper
storeProvider := app.IBCKeeper.ClientKeeper.GetStoreProvider()
tmLightClientModule := ibctm.NewLightClientModule(appCodec, storeProvider)
clientKeeper.AddRoute(ibctm.ModuleName, &tmLightClientModule)
```

---

### 1.8 CosmWasm Keeper & Reserved Contract Addresses

**Reserved contract addresses** are defined in `chain/app/wasm.go` using deterministic module account derivation. All four are computed in an `init()` function so they are available before `NewApp()` runs:

```go
// chain/app/wasm.go
var (
    ConstitutionContractAddr sdk.AccAddress
    TreasuryContractAddr     sdk.AccAddress
    ReserveFundContractAddr  sdk.AccAddress
    GovernanceContractAddr   sdk.AccAddress
)

func init() {
    ConstitutionContractAddr = types.NewModuleAddress("wasm.constitution")
    TreasuryContractAddr     = types.NewModuleAddress("wasm.treasury")
    ReserveFundContractAddr  = types.NewModuleAddress("wasm.reserve")
    GovernanceContractAddr   = types.NewModuleAddress("wasm.governance")
}
```

**WasmKeeper** is initialized with `nobody` upload access (governance-only WASM uploads):

```go
homePath := cast.ToString(appOpts["home"])
wasmDir := filepath.Join(homePath, "wasm")
nodeConfig, err := wasm.ReadNodeConfig(MapAppOptions(appOpts))

app.WasmKeeper = wasmkeeper.NewKeeper(
    appCodec,
    runtime.NewKVStoreService(keys[wasm.StoreKey]),
    app.AccountKeeper,
    app.BankKeeper,
    app.StakingKeeper,
    distrkeeper.NewQuerier(app.DistrKeeper),
    app.IBCKeeper.ChannelKeeper,
    app.IBCKeeper.ChannelKeeper,
    app.IBCKeeper.ChannelKeeperV2,
    app.TransferKeeper,
    app.MsgServiceRouter(),
    app.GRPCQueryRouter(),
    wasmDir,
    nodeConfig,
    wasmtypes.VMConfig{},
    wasmkeeper.BuiltInCapabilities(),
    authtypes.NewModuleAddress(govtypes.ModuleName).String(),
    GetWasmOpts(nil)...,
)
```

The `CodeUploadAccess` parameter is set to `Nobody` in genesis — contracts may only be stored via governance proposals.

---

### 1.9 Custom Module Keepers

All seven custom module keepers are initialized after the CosmWasm keeper (so `WasmKeeper` is available when `GovExtKeeper` is wired):

```go
app.ValidatorKeeper     = validator.NewKeeper(keys[validator.StoreKey], appCodec,
                              app.StakingKeeper, app.SlashingKeeper, app.BankKeeper,
                              app.DistrKeeper, 30)           // 30 = MaxValidatorSlots

app.OracleKeeper        = oracle.NewKeeper(keys[oracle.StoreKey], appCodec,
                              app.StakingKeeper, app.SlashingKeeper)

app.MilestoneKeeper     = milestone.NewKeeper(keys[milestone.StoreKey], appCodec,
                              app.OracleKeeper, app.BankKeeper)

app.SettlementKeeper    = settlement.NewKeeper(keys[settlement.StoreKey], appCodec,
                              app.BankKeeper)

app.BridgeKeeper        = bridge.NewKeeper(keys[bridge.StoreKey], appCodec,
                              app.BankKeeper)

app.CertificationKeeper = certification.NewKeeper(keys[certification.StoreKey], appCodec,
                              app.StakingKeeper, app.SlashingKeeper)
```

---

### 1.10 governance-ext Keeper & Constitution Wiring

`GovExtKeeper` is initialized last because it depends on all other custom keepers. It receives a **`DefaultPermissionKeeper` wrapper** (not the raw `WasmKeeper`) and the `ConstitutionContractAddr` so every governance proposal is gated by the Constitution CosmWasm contract:

```go
app.GovExtKeeper = gext.NewKeeper(
    keys[govtypes.StoreKey],
    appCodec,
    wasmkeeper.NewDefaultPermissionKeeper(app.WasmKeeper), // ← permission-gated wrapper
    ConstitutionContractAddr,   // constitution queried before every proposal
    app.ValidatorKeeper,
    app.MilestoneKeeper,
    app.OracleKeeper,
    app.SettlementKeeper,
    app.BridgeKeeper,
)
```

---

### 1.11 Module Registration & Block Hook Ordering

All modules (standard + IBC + EVM + CosmWasm + 7 custom) are registered with the module manager. The block hook ordering enforces EVM-specific constraints:

**PreBlockers** (`app.mm.SetOrderPreBlockers`):
```
upgrade → vm → auth → bank → distribution → staking → slashing → gov
→ feegrant → authz → consensus → ibc → ibctm → transfer
→ feemarket → erc20 → wasm → validator → certification → oracle
→ milestone → settlement → govext → bridge
```

**BeginBlockers** (`app.mm.SetOrderBeginBlockers`):
```
feemarket → vm → erc20           ← EVM first (EIP-1559 base fee update before block execution)
→ ibc → transfer → distribution → slashing → staking → auth → bank
→ gov → upgrade → wasm → feegrant → authz → consensus → ibctm → genutil
→ validator → certification → oracle → milestone → settlement → govext → bridge
```

**EndBlockers** (`app.mm.SetOrderEndBlockers`):
```
vm → erc20 → feemarket           ← feemarket LAST (reads full block gas used)
→ gov → staking → ibc → transfer → distribution → slashing → auth → bank
→ upgrade → wasm → feegrant → authz → consensus → ibctm → genutil
→ validator → certification → oracle → milestone → settlement → govext → bridge
```

**InitGenesis** (`app.mm.SetOrderInitGenesis`):
```
auth → bank → distribution → staking → slashing → gov → consensus → ibc
→ vm → feemarket → erc20         ← InitGenesis order: vm first, then feemarket, then erc20
→ transfer → genutil → upgrade → wasm → feegrant → authz → ibctm
→ validator → certification → oracle → milestone → settlement → govext → bridge
```

> **Important distinction:** Keeper *initialization order* is `feemarket → vm → erc20` (feemarket must exist before vm is constructed). Genesis *initialization order* is `vm → feemarket → erc20` (EVM state must be seeded before feemarket reads its params, erc20 last). Both are correct; they serve different purposes.

After module registration, the `Configurator` is properly initialized:

```go
app.Configurator = module.NewConfigurator(
    app.appCodec,
    app.MsgServiceRouter(),
    app.GRPCQueryRouter(),
)
app.mm.RegisterServices(app.Configurator)
```

---

### 1.12 Ante Handler — cosmos/evm Unified Handler

The standard Cosmos SDK ante handler is replaced with the `cosmos/evm` unified ante handler. This single handler correctly routes both Cosmos SDK transactions and raw EVM transactions (`MsgEthereumTx`) through the appropriate validation pipelines.

`setAnteHandler()` is called after module registration but before `LoadLatestVersion()`:

```go
func (app *App) setAnteHandler() {
    options := evmante.HandlerOptions{
        Cdc:                    app.appCodec,
        AccountKeeper:          app.AccountKeeper,
        BankKeeper:             app.BankKeeper,
        ExtensionOptionChecker: antetypes.HasDynamicFeeExtensionOption,
        EvmKeeper:              app.EVMKeeper,
        FeegrantKeeper:         app.FeeGrantKeeper,
        IBCKeeper:              app.IBCKeeper,
        FeeMarketKeeper:        app.FeeMarketKeeper,
        MaxTxGasWanted:         0,              // 0 = no per-tx gas cap via code; configured in app.toml
        DynamicFeeChecker:      true,           // enables EIP-1559 dynamic fee checking
        SigGasConsumer:         evmante.SigVerificationGasConsumer,
        SignModeHandler:        app.txConfig.SignModeHandler(),
        PendingTxListener:      func(ethcommon.Hash) {},
    }
    if err := options.Validate(); err != nil {
        panic(err)    // misconfigured options cause a startup panic
    }

    baseHandler := evmante.NewAnteHandler(options)
    // ... authz blocked message wrapper applied on top (see §1.13)
    app.SetAnteHandler(wrappedAnteHandler)
}
```

---

### 1.13 authz Blocked Message Types

Six message types are blocked from being delegated via `x/authz`. The block list is defined as a `map[string]bool` **inside `setAnteHandler()`** — there is no separate function for this. The ante handler wrapper checks every `MsgGrant` before passing to the base handler:

```go
blockedMsgs := map[string]bool{
    "/sovereign.bridge.v1.MsgBridgeIn":           true,
    "/sovereign.bridge.v1.MsgBridgeOut":          true,
    "/sovereign.oracle.v1.MsgSubmitOracleCommit": true,
    "/sovereign.oracle.v1.MsgRevealOracleReport": true,
    "/sovereign.settlement.v1.MsgSettlement":     true,
    "/cosmos.evm.vm.v1.MsgEthereumTx":            true,
}

wrappedAnteHandler := func(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
    for _, msg := range tx.GetMsgs() {
        if msgGrant, ok := msg.(*authz.MsgGrant); ok {
            auth, err := msgGrant.GetAuthorization()
            if err != nil {
                return ctx, fmt.Errorf("failed to get authorization: %w", err)
            }
            if blockedMsgs[auth.MsgTypeURL()] {
                return ctx, fmt.Errorf(
                    "authorization grant for message type %s is blocked",
                    auth.MsgTypeURL(),
                )
            }
        }
    }
    return baseHandler(ctx, tx, simulate)
}
```

| Blocked Message Type | Reason |
|---|---|
| `/sovereign.bridge.v1.MsgBridgeIn` | Bridge mint — must originate from relayer quorum, not delegated |
| `/sovereign.bridge.v1.MsgBridgeOut` | Bridge burn — must be direct user action |
| `/sovereign.oracle.v1.MsgSubmitOracleCommit` | Oracle commit — must come from registered operator |
| `/sovereign.oracle.v1.MsgRevealOracleReport` | Oracle reveal — must come from same operator |
| `/sovereign.settlement.v1.MsgSettlement` | Settlement — requires Ed25519 witness signature |
| `/cosmos.evm.vm.v1.MsgEthereumTx` | Raw EVM tx — cannot be authz-delegated by design |

---

### 1.14 EVM Mempool Configuration

After `setAnteHandler()` and before `LoadLatestVersion()`, the EVM-priority nonce mempool is configured:

```go
app.configureEVMMempool(MapAppOptions(appOpts), sdkLogger)

if loadLatest {
    if err := app.LoadLatestVersion(); err != nil {
        panic(err)
    }
}
```

`configureEVMMempool()` wires CometBFT handlers (`ReapTxs`, `InsertTx`, `CheckTx`, `PrepareProposal`) required for block production. Without this, CometBFT cannot produce blocks because `ReapTxs` has no handler. It is wrapped in a `recover()` to gracefully skip configuration in test environments where stores are not yet mounted.

---

### 1.15 Staking Compatibility Layer

`chain/app/staking_compatibility.go` implements the `StakingCompatibilityKeeper` shim that translates the equalized-slot voting model (ADR-001) into the interfaces expected by `x/distribution`, `x/gov`, and IBC.

#### GetEqualizedValidatorPower

```go
const EqualizedPowerPerSlot = 1_000_000

func (k StakingCompatibilityKeeper) GetEqualizedValidatorPower(
    ctx sdk.Context, valAddr sdk.ValAddress,
) int64 {
    rawPower := k.stakingKeeper.GetLastValidatorPower(ctx, valAddr)
    if rawPower > 0 {
        return 1_000_000 // any active slot gets exactly 1,000,000
    }
    return 0
}

func (k StakingCompatibilityKeeper) GetEqualizedTotalPower(_ sdk.Context) math.Int {
    // Total = MaxValidators (30) × 1,000,000
    return math.NewInt(int64(k.MaxValidators) * 1_000_000)
}
```

#### AllocateTokens — Equal-Slot Reward Split with Rounding Remainder

```go
func (k StakingCompatibilityKeeper) AllocateTokens(
    ctx sdk.Context,
    totalRewards sdk.DecCoins,
    activeValidators []stakingtypes.ValidatorI,
) {
    if len(activeValidators) == 0 {
        return
    }
    slotCount := int64(len(activeValidators))
    perValidatorRewards := totalRewards.QuoDec(math.LegacyNewDec(slotCount))

    for i, val := range activeValidators {
        rewardSlice := perValidatorRewards
        // Give the rounding remainder to the first validator to ensure
        // sum(allocated) == totalRewards exactly (no dust lost)
        if i == 0 {
            allocated := perValidatorRewards.MulDec(math.LegacyNewDec(slotCount))
            remainder := totalRewards.Sub(allocated)
            rewardSlice = perValidatorRewards.Add(remainder...)
        }
        if k.distrKeeper != nil {
            k.distrKeeper.AllocateTokensToValidator(ctx, val, rewardSlice)
        }
    }
}
```

#### OverrideHistoricalInfo — IBC Light Client Compatibility

IBC requires `HistoricalInfo` records (stored each block) to reflect validator powers. This override rewrites them with the equalized token amounts:

```go
func (app *App) OverrideHistoricalInfo(ctx sdk.Context) {
    height := ctx.BlockHeight()
    hi, err := app.StakingKeeper.GetHistoricalInfo(ctx, height)
    if err != nil {
        return
    }
    powerReduction := app.StakingKeeper.PowerReduction(ctx)  // = 1,000,000 (set in §1.2)
    for i := range hi.Valset {
        if hi.Valset[i].IsBonded() {
            // Tokens = powerReduction × equalized_power = 1,000,000 × 1,000,000
            hi.Valset[i].Tokens = powerReduction.Mul(math.NewInt(1_000_000))
        }
    }
    _ = app.StakingKeeper.SetHistoricalInfo(ctx, height, &hi)
}
```

---

### 1.16 EndBlocker — Equalized Validator Updates

The `EndBlocker` calls two equalization functions after the standard module end-block:

```go
func (app *App) EndBlocker(ctx sdk.Context) (sdk.EndBlock, error) {
    res, err := app.mm.EndBlock(ctx)
    if err != nil {
        return res, err
    }
    // 1. Rewrite HistoricalInfo for IBC light client compatibility
    app.OverrideHistoricalInfo(ctx)
    // 2. Override all non-zero ValidatorUpdate powers sent to CometBFT
    res.ValidatorUpdates = app.GetEqualizedValidatorUpdates(ctx, res.ValidatorUpdates)
    return res, nil
}
```

`GetEqualizedValidatorUpdates` overrides the actual consensus updates returned to CometBFT. Without this, CometBFT would use staking-weighted powers instead of the equalized 1,000,000:

```go
func (app *App) GetEqualizedValidatorUpdates(
    ctx sdk.Context,
    updates []abci.ValidatorUpdate,
) []abci.ValidatorUpdate {
    equalizedUpdates := make([]abci.ValidatorUpdate, len(updates))
    for i, update := range updates {
        equalizedUpdates[i] = update
        if update.Power > 0 {
            equalizedUpdates[i].Power = 1_000_000
        }
    }
    return equalizedUpdates
}
```

---

### 1.17 Upgrade Handler (v1.0.0)

`chain/app/upgrades.go` registers the `v1.0.0` upgrade handler scaffold:

```go
const UpgradeNameV1 = "v1.0.0"

func (app *App) RegisterUpgradeHandlers() {
    app.UpgradeKeeper.SetUpgradeHandler(
        UpgradeNameV1,
        func(ctx context.Context, plan upgradetypes.Plan, fromVM module.VersionMap,
        ) (module.VersionMap, error) {
            sdkCtx := sdk.UnwrapSDKContext(ctx)
            sdkCtx.Logger().Info("Executing Sovereign L1 v1.0.0 Upgrade Handler...")
            return app.mm.RunMigrations(sdkCtx, app.configurator(), fromVM)
        },
    )
    // Store loader for safe KV store additions at upgrade height
    upgradeInfo, _ := app.UpgradeKeeper.ReadUpgradeInfoFromDisk()
    if upgradeInfo.Name == UpgradeNameV1 && !app.UpgradeKeeper.IsSkipHeight(upgradeInfo.Height) {
        app.SetStoreLoader(upgradetypes.UpgradeStoreLoader(upgradeInfo.Height,
            &storetypes.StoreUpgrades{Added: []string{}}))
    }
}
```

Since all custom modules are present in genesis from day 1, no store additions are needed for the v1.0.0 upgrade — the scaffold is correct for a genesis-only chain launch.

---

### 1.18 app.toml — JSON-RPC Configuration

`chain/config/app.toml` — full JSON-RPC section as implemented:

```toml
[json-rpc]
enable = true
address = "0.0.0.0:8545"        # Standard Ethereum JSON-RPC port
ws-address = "0.0.0.0:8546"     # Standard Ethereum WebSocket port

# All namespaces required for MetaMask, Blockscout, and wagmi interoperability
api = "eth,net,web3,txpool,debug,personal"

# Cap on gas usable in eth_call / estimateGas (0 = infinite)
gas-cap = 25000000

# Global timeout for eth_call
evm-timeout = "5s"

# Global tx fee cap for send-transaction variants (in ether, 0 = no cap)
txfee-cap = 1.0

# Maximum number of open filters
filter-cap = 200
```

---

### 1.19 Genesis Script & Supply Invariants

`scripts/generate_genesis.go` is a **standalone script** (not a normal package import):

```go
//go:build ignore
// +build ignore
//
// Usage:
//   go run scripts/generate_genesis.go            # generate genesis.json
//   go run scripts/generate_genesis.go --verify   # invariant check only (no write)
//   go run scripts/generate_genesis.go --out /tmp/g.json
```

#### Supply Constants

```go
const (
    TokenDenom = "ucsov"
    EVMDenom   = "aesov"
    EVMChainID = uint64(7777)

    TotalSupply          = int64(1_000_000_000) * int64(1_000_000) // 1 billion CSOV in ucsov
    BscEscrowBalance     = int64(300_000_000)   * int64(1_000_000) // 300M CSOV locked in LockBox
    CosmosAllocation     = TotalSupply - BscEscrowBalance          // 700M CSOV on Cosmos side

    RewardsBucket        = int64(100_000_000) * int64(1_000_000)   // 100M CSOV emission pool
    PerBlockEmissionUtoken = int64(1_500_000)                       // 1.5 CSOV per block
    MinLifetimeBlocks    = int64(31_536_000)                        // 5-year floor at 5s/block

    ValidatorSlots       = 30
    EqualizedPowerPerSlot = int64(1_000_000)
)
```

#### Five Genesis Invariants

The script checks all five before writing `genesis.json`. Any failure exits non-zero and blocks chain launch:

| # | Invariant | Check |
|---|---|---|
| INV-1 | `CosmosAllocation + BscEscrowBalance == TotalSupply` | 700M + 300M = 1B ✅ |
| INV-2 | `RewardsBucket / PerBlockEmission >= MinLifetimeBlocks` | 100M/1.5 ≈ 66.7M blocks > 31.5M ✅ |
| INV-3 | `EVMChainID != 0` | 7777 ✅ |
| INV-4 | `EVMDenom == "aesov"` | matches ADR-011 ✅ |
| INV-5 | `ValidatorSlots × EqualizedPowerPerSlot == 30,000,000` | 30 × 1,000,000 ✅ |

#### CosmWasm Contract Compilation

`buildAppState()` calls `compileContracts()` at runtime to compile and embed the four genesis CosmWasm contracts. This requires `cargo` and `wasm-opt` to be present:

```go
func compileContracts() error {
    cmd := exec.Command("cargo", "build",
        "--target", "wasm32-unknown-unknown", "--release", "--lib")
    cmd.Dir = "contracts"
    cmd.Env = append(os.Environ(), "RUSTFLAGS=-C target-feature=-bulk-memory")
    // ...
    for _, contract := range []string{
        "constitution.wasm", "treasury.wasm", "reserve_fund.wasm", "governance.wasm",
    } {
        exec.Command("wasm-opt", "--llvm-memory-copy-fill-lowering", path, "-o", path).Run()
    }
}
```

#### Dev vs Prod Environments

The script accepts `--env dev` or `--env prod`:

- **dev**: Injects two genesis holder accounts with large balances of all three denominations (for testnet bootstrapping)
- **prod**: Panics if those dev accounts are found — prevents accidental dev-funding in mainnet genesis

---

### 1.20 Genesis Parameters — Full Module Configuration

All module parameters written by `buildAppState()`:

#### auth
```
max_memo_characters = 256
tx_sig_limit = 7
tx_size_cost_per_byte = 10
sig_verify_cost_ed25519 = 590
sig_verify_cost_secp256k1 = 1000
```

#### bank — Denom Metadata (all 3 tokens)
| Token | Base | Display | Exponent |
|---|---|---|---|
| ESOV (EVM gas) | `aesov` | `esov` | 18 |
| CSOV (staking) | `ucsov` | `csov` | 6 |
| WSOV (bridge) | `uwsov` | `wsov` | 6 |

#### staking
```
bond_denom = "ucsov"
max_validators = 30
unbonding_time = 1814400s   (21 days)
max_entries = 7
historical_entries = 10000
last_total_power = 30,000,000   (30 slots × 1,000,000)
```

#### governance
```
min_deposit = 10,000,000 ucsov  (10 CSOV)
max_deposit_period = 172800s    (48 hours)
voting_period = 172800s         (48 hours)
quorum = 0.334
threshold = 0.500
veto_threshold = 0.334
```

#### distribution
```
community_tax = 0.020
base_proposer_reward = 0.000    (disabled — equal slot split)
bonus_proposer_reward = 0.000   (disabled — equal slot split)
withdraw_addr_enabled = true
```

#### x/vm (EVM)
```
evm_denom = "aesov"
active_static_precompiles = ["0x...0101", "0x...0102"]
access_control.create = ACCESS_TYPE_PERMISSIONLESS
access_control.call = ACCESS_TYPE_PERMISSIONLESS
```

#### x/feemarket (EIP-1559)
```
no_base_fee = false
base_fee_change_denominator = 8
elasticity_multiplier = 2
enable_height = 0
base_fee = "1000000000"          (1 Gwei)
min_gas_price = "0.025"
min_gas_multiplier = "0.500"
```

#### x/erc20
```
enable_erc20 = true
token_pairs = [{ denom: "ucsov", erc20_address: "0x...0001", owner: MODULE }]
```

#### x/wasm
```
code_upload_access = Nobody     (governance-only uploads)
instantiate_default_permission = Everybody
```
Four genesis contracts pre-loaded: `constitution` (code 1), `treasury` (code 2), `reserve_fund` (code 3), `governance` (code 4). Sequences start at 5.

#### x/bridge
```
standard_finality_depth = 15
large_finality_depth = 50
large_transfer_threshold = 5,000,000,000 uwsov   (5,000 WSOV)
quorum_threshold = 3
max_unlock_per_block = 100,000,000,000 uwsov      (100,000 WSOV)
supply_cap = 1,000,000,000,000 uwsov              (1,000,000 WSOV)
```

---

### Phase 1 Task Checklist

| # | Task | Status | Source File |
|---|---|---|---|
| 1.1 | Pin `github.com/cosmos/evm@v0.7.0` in `go.mod` | ✅ | `chain/go.mod` |
| 1.2 | Remove `skip-mev/feemarket` from `go.mod` | ✅ | `chain/go.mod` |
| 1.3 | Upgrade to `ibc-go v11` (from v8) | ✅ | `chain/go.mod` |
| 1.4 | Wire `x/feemarket` keeper in `app.go` | ✅ | `app.go:453` |
| 1.5 | Wire `x/vm` keeper in `app.go` | ✅ | `app.go:461` |
| 1.6 | Wire `x/erc20` keeper in `app.go` | ✅ | `app.go:475` |
| 1.7 | Module init order: `feemarket → vm → erc20` (keeper); `vm → feemarket → erc20` (InitGenesis) | ✅ | `app.go:639–709` |
| 1.8 | Wire IBC modules (`IBCKeeper`, `TransferKeeper`, erc20 IBC middleware) | ✅ | `app.go:432–500` |
| 1.9 | Replace ante handler with `cosmos/evm/ante.NewAnteHandler` | ✅ | `app.go:766` |
| 1.10 | `MsgEthereumTx` authz block registration (inline string map in `setAnteHandler`) | ✅ | `app.go:793–799` |
| 1.11 | `x/vm` genesis params: EVM denom `aesov`, precompile addresses, permissionless access | ✅ | `generate_genesis.go` |
| 1.12 | `x/feemarket` genesis params: base fee 1 Gwei, elasticity 2, no-base-fee false | ✅ | `generate_genesis.go` |
| 1.13 | `x/erc20` genesis: native token pair (`ucsov ↔ ERC-20`) | ✅ | `generate_genesis.go` |
| 1.14 | `app.toml` JSON-RPC: port 8545/8546, 6 namespaces, gas-cap, txfee-cap, filter-cap | ✅ | `chain/config/app.toml` |
| 1.15 | `StakingCompatibilityKeeper.GetEqualizedValidatorPower` (1,000,000 per active slot) | ✅ | `staking_compatibility.go` |
| 1.16 | `StakingCompatibilityKeeper.AllocateTokens` (equal-slot split, first-validator remainder) | ✅ | `staking_compatibility.go` |
| 1.17 | IBC `HistoricalInfo` override via `OverrideHistoricalInfo` in `EndBlocker` | ✅ | `staking_compatibility.go`, `app.go:926` |
| 1.18 | `GetEqualizedValidatorUpdates` in `EndBlocker` (overrides consensus power to CometBFT) | ✅ | `staking_compatibility.go`, `app.go:927` |
| 1.19 | Upgrade handler v1.0.0 scaffold (`RegisterUpgradeHandlers`) — **⚠️ `configurator()` returns nil bug** | ✅ scaffold / ⚠️ bug | `upgrades.go` |
| 1.20 | CosmWasm wired with `Nobody` upload policy | ✅ | `app.go:508`, `generate_genesis.go` |
| 1.21 | `ConstitutionContractAddr` and 3 other contract addresses derived in `wasm.go` init() | ✅ | `wasm.go` |
| 1.22 | `GovExtKeeper` wired with `DefaultPermissionKeeper` wrapper + `ConstitutionContractAddr` | ✅ | `app.go:551` |
| 1.23 | `x/authz` blocked message types (6 types as string map in `setAnteHandler`) | ✅ | `app.go:793` |
| 1.24 | `scripts/generate_genesis.go` — 5-invariant genesis verification (build-ignore tag) | ✅ | `scripts/generate_genesis.go` |
| 1.25 | Genesis supply: 1B CSOV total, 700M Cosmos + 300M BSC escrow, all 5 invariants pass | ✅ | `generate_genesis.go` |
| 1.26 | `sdk.DefaultPowerReduction = math.NewInt(1_000_000)` set at top of `NewApp()` | ✅ | `app.go:258` |
| 1.27 | Object store keys (`oKeys`) for EVM bloom filter / gas tracking | ✅ | `app.go:315` |
| 1.28 | EVM mempool configured (`configureEVMMempool`) before `LoadLatestVersion()` | ✅ | `app.go:755` |

**All 28 Phase 1 implementation points verified against source.**

---

## Faucet — Backend Service

**Source:** `backend/module/faucet/main.go` (329 lines)
**Endpoint:** `POST /faucet`
**Default listen:** `:8000`

### Architecture Overview

```
Client (Browser / curl)
        │
        │  POST /faucet  { "address": "cosmos1..." }
        ▼
┌─────────────────────────────────────────────────────────┐
│                  Faucet HTTP Server                      │
│                                                         │
│  1.  CORS preflight check                               │
│  2.  Method check (POST only)                           │
│  3.  JSON decode                                        │
│  4.  normalizeAddress() — hex / bech32 / raw-hex        │
│  5.  Per-address cooldown gate (in-memory map)          │
│  6.  txMu.Lock() ← serializes ALL sends                 │
│  7.  getAccountSequence() from chain node               │
│  8.  exec "chaind tx bank send" with retry loop (×3)    │
│  9.  Parse JSON txhash + code from output               │
│  10. Record cooldown timestamp                          │
│  11. Return { success: true, tx_hash: "..." }           │
└─────────────────────────────────────────────────────────┘
        │ exec
        ▼
  chaind tx bank send <key> <address> <amount>ucsov
    --node <CometBFT RPC>
    --broadcast-mode sync
    --gas auto  --gas-adjustment 1.5
    --gas-prices 1000000000aesov
    --sequence <queried_sequence>
```

---

### Configuration & Startup

All configuration is read from **CLI flags with environment variable fallbacks**:

| Flag | Env Var | Default | Description |
|---|---|---|---|
| `--node` | `NODE_URL` | `http://localhost:26657` | CometBFT RPC endpoint |
| `--key` | `FAUCET_KEY` | `faucet` | Keyring key name |
| `--denom` | `DENOM` | `ucsov` | Base token denomination |
| `--amount` | `FAUCET_AMOUNT` | `10000000` | Amount to send (in base units = 10 CSOV) |
| `--listen` | _(none)_ | `:8000` | HTTP listen address |
| `--chain-id` | `CHAIN_ID` | `sovereign-1` | Chain ID for signing |
| `CHAIN_HOME` | `CHAIN_HOME` | `/root/.sovereign` | Chain home dir (keyring location) |

```go
func main() {
    flag.StringVar(&nodeURL, "node", os.Getenv("NODE_URL"), "CometBFT RPC endpoint")
    flag.StringVar(&keyName, "key", os.Getenv("FAUCET_KEY"), "Name of key in keyring")
    flag.StringVar(&denom, "denom", os.Getenv("DENOM"), "Denomination of token")
    flag.StringVar(&faucetAmount, "amount", os.Getenv("FAUCET_AMOUNT"), "Amount to transfer")
    flag.StringVar(&listenAddr, "listen", ":8000", "Listen address")
    flag.StringVar(&chainID, "chain-id", os.Getenv("CHAIN_ID"), "Chain ID")
    flag.Parse()
    // defaults applied if env vars empty ...
    http.HandleFunc("/faucet", handleFaucet)
    log.Printf("Faucet daemon listening on %s", listenAddr)
    http.ListenAndServe(listenAddr, nil)
}
```

---

### HTTP Handler — Request Flow

`handleFaucet` is the sole HTTP handler. Complete decision tree:

```
OPTIONS                 → 200 OK + CORS headers (preflight)
!POST                   → 405  { success: false, error: "Only POST allowed" }
bad JSON                → 400  { success: false, error: "Invalid JSON" }
bad address             → 400  { success: false, error: "Invalid address: ..." }
cooldown active         → 429  { success: false, error: "Please wait N seconds..." }
exec / tx error (fatal) → 500  { success: false, error: "<raw_log or exec error>" }
tx code != 0            → 500  { success: false, error: "<raw_log>" }
success                 → 200  { success: true, tx_hash: "<64-char hex>" }
```

Response struct:

```go
type FaucetResponse struct {
    TxHash  string `json:"tx_hash,omitempty"`
    Success bool   `json:"success"`
    Error   string `json:"error,omitempty"`
}
```

---

### Address Normalization

`normalizeAddress` accepts three input formats and always returns a `cosmos1…` bech32 address:

```go
func normalizeAddress(input string) (string, error) {
    input = strings.TrimSpace(input)
    if input == "" {
        return "", fmt.Errorf("empty address")
    }

    // 1. EVM hex address (0x + 20 bytes)
    if strings.HasPrefix(input, "0x") {
        hexStr := strings.TrimPrefix(input, "0x")
        bytes, err := hex.DecodeString(hexStr)
        if err != nil || len(bytes) != 20 {
            return "", fmt.Errorf("invalid hex address: %v", err)
        }
        return sdk.AccAddress(bytes).String(), nil
    }

    // 2. Bech32 — accepts prefix: cosmos | sov | sovereign
    hrp, bytes, err := bech32.DecodeAndConvert(input)
    if err != nil {
        // 3. Raw hex without 0x prefix (20 bytes)
        bytes, errHex := hex.DecodeString(input)
        if errHex == nil && len(bytes) == 20 {
            return sdk.AccAddress(bytes).String(), nil
        }
        return "", fmt.Errorf("invalid address format: not a valid Bech32 or hex address")
    }
    if hrp != "cosmos" && hrp != "sov" && hrp != "sovereign" {
        return "", fmt.Errorf("unsupported address prefix: %s", hrp)
    }
    return sdk.AccAddress(bytes).String(), nil
}
```

| Input | Accepted |
|---|---|
| `0xe1f1a509...` (EVM hex, 20 bytes) | ✅ |
| `cosmos1abc...` | ✅ |
| `sovereign1abc...` | ✅ |
| `sov1abc...` | ✅ |
| Raw hex without `0x` (20 bytes) | ✅ |
| `eth1abc...` (unsupported prefix) | ❌ 400 |
| `0x1234` (too short hex) | ❌ 400 |
| Empty string | ❌ 400 |
| Garbage string | ❌ 400 |

---

### Per-Address Cooldown

An in-memory map tracks the last successful send time per normalized address. The cooldown check happens **before** acquiring `txMu` so the global send mutex is never held during a rejection:

```go
var (
    addressCooldowns   = make(map[string]time.Time)
    addressCooldownsMu sync.Mutex
    cooldownDuration   = 10 * time.Second   // devnet value; production: 24 hours
)

// In handleFaucet:
addressCooldownsMu.Lock()
if lastSend, ok := addressCooldowns[address]; ok {
    if time.Since(lastSend) < cooldownDuration {
        remaining := cooldownDuration - time.Since(lastSend)
        addressCooldownsMu.Unlock()
        w.WriteHeader(http.StatusTooManyRequests)
        json.NewEncoder(w).Encode(FaucetResponse{
            Success: false,
            Error:   fmt.Sprintf("Please wait %d seconds before requesting again",
                         int(remaining.Seconds())+1),
        })
        return
    }
}
addressCooldownsMu.Unlock()
```

After a successful send:
```go
addressCooldownsMu.Lock()
addressCooldowns[address] = time.Now()
addressCooldownsMu.Unlock()
```

> **Note:** The cooldown map is in-memory only and resets on daemon restart.

---

### Serialized Tx Dispatch & Retry Logic

A global `sync.Mutex` serializes all faucet transactions to prevent concurrent sequence number collisions:

```go
var txMu sync.Mutex

// In handleFaucet (after cooldown check):
txMu.Lock()
defer txMu.Unlock()
```

**Command arguments:**

```go
cmdArgs := []string{
    "tx", "bank", "send",
    keyName, address, faucetAmount + denom,
    "--node",            nodeURL,
    "--keyring-backend", "test",
    "--chain-id",        chainID,
    "--home",            chainHome,
    "--yes",
    "--broadcast-mode",  "sync",
    "--gas",             "auto",
    "--gas-adjustment",  "1.5",
    "--gas-prices",      "1000000000aesov",   // 1 Gwei (runtime value)
    "--output",          "json",
    "--sequence",        strconv.FormatUint(seq, 10),
}
```

**Retry loop** (up to 3 retries, 2-second wait between attempts):

```go
maxRetries := 3
for attempt := 0; attempt <= maxRetries; attempt++ {
    if attempt > 0 {
        time.Sleep(2 * time.Second)
        newSeq, err := getAccountSequence(chainHome)
        if err == nil {
            // update --sequence in cmdArgs
        }
    }
    cmd := exec.Command("chaind", cmdArgs...)
    output, err = cmd.CombinedOutput()
    // Retry only on recoverable errors:
    if strings.Contains(string(output), "tx already seen") ||
       strings.Contains(string(output), "account sequence mismatch") {
        continue
    }
    break
}
```

After the retry loop, the tx output JSON is parsed for `txhash` and `code`. If `code != 0`, a 500 is returned with `raw_log` as the error.

---

### Sequence Number Resolution

Before every send, the faucet queries the current sequence number from the chain to avoid stale-nonce collisions across restarts:

```go
func getAccountSequence(chainHome string) (uint64, error) {
    cmd := exec.Command("chaind", "query", "auth", "account", keyName,
        "--node",   nodeURL,
        "--home",   chainHome,
        "--output", "json",
    )
    out, _ := cmd.CombinedOutput()

    // Handles both Cosmos SDK v0.47+ nested and flat response shapes
    var result struct {
        Account  struct{ Sequence string `json:"sequence"` } `json:"account"`
        Sequence string `json:"sequence"`
    }
    json.Unmarshal(out, &result)

    seqStr := result.Account.Sequence
    if seqStr == "" {
        seqStr = result.Sequence
    }
    return strconv.ParseUint(seqStr, 10, 64)
}
```

If the query fails, the send proceeds without `--sequence` and the node assigns the next available nonce.

---

### CORS Policy

Permissive CORS headers are set for every request (devnet configuration):

```go
w.Header().Set("Access-Control-Allow-Origin",  "*")
w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
w.Header().Set("Content-Type", "application/json")
```

`OPTIONS` preflight returns `200 OK` immediately with an empty body.

---

### Known API Gap — Response Fields

The `FaucetResponse` struct has only three fields:

```go
type FaucetResponse struct {
    TxHash  string `json:"tx_hash,omitempty"`
    Success bool   `json:"success"`
    Error   string `json:"error,omitempty"`
}
```

The Explorer UI reads two additional fields — `data.message` and `data.cooldownSeconds` — that **do not exist** in this struct. The UI gracefully falls back (`data.message || "Tokens successfully requested!"` and `data.cooldownSeconds || 86400`), but these fields should be added to `FaucetResponse` for a complete implementation.

---

### Test Coverage

**Source:** `backend/module/faucet/main_test.go` (249 lines)

All tests use `httptest.NewRecorder` and call `handleFaucet` directly without a real chain node.

**7 test functions:**

| Function | What it tests |
|---|---|
| `TestHandleFaucetCORSPreflight` | OPTIONS → 200, `Access-Control-Allow-Origin: *`, `Allow-Methods: POST, OPTIONS` |
| `TestHandleFaucetRejectsGET` | GET → 405, `{ success: false, error: "Only POST allowed" }` |
| `TestHandleFaucetRejectsInvalidJSON` | Malformed body → 400, `{ success: false, error: "Invalid JSON" }` |
| `TestHandleFaucetRejectsEmptyAddress` | `{"address":""}` → 400, `success: false` |
| `TestHandleFaucetRejectsGarbageAddress` | `{"address":"not-a-real-address-xyz"}` → 400, `success: false` |
| `TestNormalizeAddress` | 8 sub-cases via `t.Run` |
| `TestCommandArgsContainBroadcastMode` | 4 assertions on expected CLI args (see below) |

**`TestNormalizeAddress` sub-cases:**

| Sub-case | Input | Expected |
|---|---|---|
| empty string | `""` | error |
| garbage input | `"xyz123"` | error |
| too short hex | `"0x1234"` | error |
| invalid hex chars | `"0xGGGG..."` | error |
| valid 20-byte hex | `"0x000...0001"` | no error |
| valid cosmos bech32 | `"cosmos1qypqx..."` | no error |
| valid sovereign bech32 | `"sovereign1qwx..."` | no error |
| unsupported prefix | `"eth1qypqx..."` | error |

**`TestCommandArgsContainBroadcastMode`** — single function, 4 inline assertions:

1. No deprecated `-b` shorthand in args
2. `--broadcast-mode sync` present
3. `--gas-prices 0aesov` present *(see note below)*
4. `--home <CHAIN_HOME>` present

> **⚠️ Gas price discrepancy:** The test's hardcoded `expectedArgs` uses `"0aesov"` for `--gas-prices`, but the actual `cmdArgs` constructed in `handleFaucet` at runtime uses `"1000000000aesov"` (1 Gwei). The test is asserting against a locally-built expected slice, not against the live command — so it does not catch this inconsistency. Production deployments should reconcile this: either `0aesov` (relies on node-side min gas price enforcement) or `1000000000aesov` (explicit 1 Gwei fee floor).

Run tests:
```bash
go test -v ./backend/module/faucet/...
```

---

## Faucet — Explorer UI

**Source:** `explorer/app/faucet/page.tsx`
**Route:** `/faucet`
**Framework:** Next.js App Router — `"use client"` directive

---

### Component State Variables

```tsx
const {
    walletType,       // e.g. "keplr" | "metamask" | null
    connected,        // boolean
    address: walletAddress,
    connectWallet,
    disconnectWallet,
} = useWalletStore();

const [targetAddress,    setTargetAddress]    = useState("");
const [loading,          setLoading]          = useState(false);
const [successMsg,       setSuccessMsg]        = useState("");
const [errorMsg,         setErrorMsg]          = useState("");
const [txHash,           setTxHash]            = useState("");
const [cooldownRemaining, setCooldownRemaining] = useState<number>(0);

const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8082";
```

`formatSuccessMessage` is a module-level helper that truncates any 64-character hex string in a message to `first10...last10` to prevent layout overflow:

```tsx
const formatSuccessMessage = (msg: string) => {
    if (!msg) return "";
    const hexRegex = /\b([a-fA-F0-9]{10})[a-fA-F0-9]{44}([a-fA-F0-9]{10})\b/;
    return msg.replace(hexRegex, "$1...$2");
};
```

---

### Wallet Auto-Fill

When the connected wallet changes, the address input is automatically populated:

```tsx
useEffect(() => {
    if (connected && walletAddress) {
        setTargetAddress(walletAddress);
    }
}, [connected, walletAddress]);
```

The wallet connect/disconnect button is rendered inline above the address field, driven by the `connected` boolean and `walletType` from the store.

---

### Client-Side Cooldown Countdown

The UI tracks cooldown state in `localStorage` keyed by address, so the countdown survives page refreshes without querying the backend:

```tsx
useEffect(() => {
    if (!targetAddress) return;

    const checkCooldown = () => {
        const claimTime = localStorage.getItem(`faucet_next_claim_${targetAddress}`);
        if (claimTime) {
            const remaining = Math.ceil((Number(claimTime) - Date.now()) / 1000);
            if (remaining > 0) {
                setCooldownRemaining(remaining);
            } else {
                setCooldownRemaining(0);
                localStorage.removeItem(`faucet_next_claim_${targetAddress}`);
            }
        } else {
            setCooldownRemaining(0);
        }
    };

    checkCooldown();
    const interval = setInterval(checkCooldown, 1000); // tick every second
    return () => clearInterval(interval);
}, [targetAddress]);
```

Countdown formatted as `HH:MM:SS`:

```tsx
const formatCooldown = (sec: number): string => {
    const h = Math.floor(sec / 3600);
    const m = Math.floor((sec % 3600) / 60);
    const s = sec % 60;
    return [h, m, s].map(v => v.toString().padStart(2, "0")).join(":");
};
```

When `cooldownRemaining > 0` the submit button is replaced by the countdown and the form is disabled.

---

### Request Flow (UI Side)

Full endpoint called: `POST ${NEXT_PUBLIC_API_URL}/api/rest/v1/explorer/faucet`

```tsx
const handleRequestTokens = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!targetAddress.trim()) {
        setErrorMsg("Please enter a valid address.");
        return;
    }
    setLoading(true);
    setErrorMsg(""); setSuccessMsg(""); setTxHash("");

    try {
        const resp = await fetch(`${API_BASE}/api/rest/v1/explorer/faucet`, {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({ address: targetAddress.trim() }),
        });

        let data: any = {};
        const contentType = resp.headers.get("content-type");
        if (contentType?.includes("application/json")) {
            data = await resp.json();
        } else {
            data = { error: await resp.text() };
        }

        if (resp.ok && data.success) {
            setSuccessMsg(data.message || "Tokens successfully requested!");
            const cooldown = Number(data.cooldownSeconds || 86400);
            localStorage.setItem(
                `faucet_next_claim_${targetAddress.trim()}`,
                String(Date.now() + cooldown * 1000)
            );
            setCooldownRemaining(cooldown);
            if (data.tx_hash) setTxHash(data.tx_hash);

        } else if (resp.status === 429) {
            const cooldown = Number(data.cooldownSeconds || 86400);
            localStorage.setItem(
                `faucet_next_claim_${targetAddress.trim()}`,
                String(Date.now() + cooldown * 1000)
            );
            setCooldownRemaining(cooldown);
            setErrorMsg(data.error || "Rate limit: 1 request per 24 hours.");

        } else {
            setErrorMsg(data.error || "An unexpected error occurred.");
        }
    } catch (err: any) {
        setErrorMsg("Network error. Please try again.");
    } finally {
        setLoading(false);
    }
};
```

> `data.message` and `data.cooldownSeconds` are read from the response but are not present in the current backend `FaucetResponse` struct (see §[Known API Gap](#known-api-gap--response-fields)). The fallbacks keep the UI functional.

---

### State Rendering

**Submit button states:**

```tsx
{loading ? (
    <><Loader2 className="h-4 w-4 animate-spin" /><span>Requesting...</span></>
) : cooldownRemaining > 0 ? (
    <span>Next request in: {formatCooldown(cooldownRemaining)}</span>
) : (
    <span>Request Tokens</span>
)}
```

**Success panel** (shown when `successMsg` is set):

```tsx
<div className="p-4 bg-green-950/20 border border-green-900 rounded-xl
                flex items-start space-x-2 text-green-400">
    <CheckCircle className="h-5 w-5 mt-0.5 flex-shrink-0" />
    <div className="text-sm">
        <span className="font-bold block">Success</span>
        <span className="text-xs">{formatSuccessMessage(successMsg)}</span>
    </div>
    {txHash && (
        <div className="mt-2 flex items-center gap-1 text-xs">
            <code className="font-mono">{txHash.slice(0,10)}...{txHash.slice(-10)}</code>
            <Link href={`/txs/${txHash}`} className="flex items-center gap-1">
                View on Explorer <ArrowRight className="h-3.5 w-3.5" />
            </Link>
        </div>
    )}
</div>
```

**Error panel** (shown when `errorMsg` is set):

```tsx
<div className="p-4 bg-red-950/20 border border-red-900 rounded-xl
                flex items-start space-x-2 text-red-400">
    <AlertCircle className="h-5 w-5 mt-0.5 flex-shrink-0" />
    <div className="text-sm">
        <span className="font-bold block">Error</span>
        <span className="text-xs leading-normal">{errorMsg}</span>
    </div>
</div>
```

---

### Faucet Info Sidebar

Static right-column information panel:

```tsx
<div className="bg-gray-950 border border-gray-900 rounded-2xl p-6 shadow-xl
                space-y-4 text-sm text-gray-400">
    <h3 className="text-base font-bold text-white">Faucet Info</h3>
    <div className="space-y-3">
        <div className="pb-3 border-b border-gray-900">
            <span className="block text-xs uppercase font-bold text-gray-500">
                Distribution Amount
            </span>
            <span className="text-white font-medium">10,000,000 ucsov (10 CSOV)</span>
        </div>
        <div className="pb-3 border-b border-gray-900">
            <span className="block text-xs uppercase font-bold text-gray-500">Rate Limit</span>
            <span className="text-white font-medium">
                1 request per address / IP every 24 hours
            </span>
        </div>
        <div>
            <span className="block text-xs uppercase font-bold text-gray-500">
                Supported Formats
            </span>
            <ul className="list-disc pl-4 space-y-1 mt-1 text-xs">
                <li>Cosmos Addresses (<code className="font-mono">cosmos1...</code>)</li>
            </ul>
        </div>
    </div>
    <div className="pt-4 border-t border-gray-900 text-xs text-gray-500 leading-normal">
        This faucet is strictly for development and testing purposes on the Sovereign Devnet.
        The tokens distributed here have no real monetary value.
    </div>
</div>
```

| Field | Value |
|---|---|
| Distribution Amount | 10,000,000 ucsov (10 CSOV) per request |
| Rate Limit | 1 request per address per 24 hours |
| Supported Formats | Cosmos bech32 (`cosmos1...`) |
| Disclaimer | Devnet only — no monetary value |

---

*Document prepared for client distribution. All other phases, modules, and implementation details are withheld and remain proprietary.*
