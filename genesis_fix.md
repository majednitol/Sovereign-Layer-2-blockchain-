# Sovereign L2 Genesis Fix Plan

**Scope:** 7 confirmed bugs in `scripts/generate_genesis.go` that produce incorrect output in
`chain/genesis.json`, `chain/genesis.dev.json`, and `chain/genesis.prod.json`.

**Strategy:** Fix the generator (`generate_genesis.go`) so that regenerating the three files
produces correct output. After all code edits, re-run the generator and commit the updated
genesis files alongside the script changes.

**Stack versions:** `cosmos-sdk v0.54.3` · `CosmWasm/wasmd v0.70.2` · `cometbft v0.39.3`

---

## Pre-flight: understand the repo layout

```
scripts/generate_genesis.go   ← single source of truth; generates all 3 genesis files
chain/genesis.json            ← Docker default / staging  (env="staging")
chain/genesis.dev.json        ← local devnet              (env="dev")
chain/genesis.prod.json       ← mainnet template          (env="prod")
chain/entrypoint.sh           ← Docker init; funds oracle operator on devnet
scripts/genesis-ceremony.sh   ← pre-launch checks for mainnet
```

All seven fixes below are in `scripts/generate_genesis.go` unless otherwise stated.

---

## FIX 1 — `gov.expedited_min_deposit` uses `"stake"` denom (all three files)

### Why it is wrong
`cosmos-sdk v0.54.3` `DefaultParams()` uses `sdk.DefaultBondDenom` ("stake") for **both**
`min_deposit` and `expedited_min_deposit`. The generator patches `min_deposit` to `ucsov` but
never touches `expedited_min_deposit`, so it stays at SDK default:
```json
{ "amount": "50000000", "denom": "stake" }
```
`"stake"` tokens do not exist on this chain. Expedited proposals can never collect a deposit
and are permanently broken.

### What to change

**File:** `scripts/generate_genesis.go`  
**Location:** inside the `// --- Modify gov ---` block, after line 612 (`params["veto_threshold"]`)

**Before (lines 602–613):**
```go
// --- Modify gov ---
if gov, ok := appState["gov"].(map[string]interface{}); ok {
    if params, ok := gov["params"].(map[string]interface{}); ok {
        params["min_deposit"] = []map[string]interface{}{
            {"denom": TokenDenom, "amount": "10000000"},
        }
        params["max_deposit_period"] = "172800s"
        params["voting_period"]      = "172800s"
        params["quorum"]             = "0.334000000000000000"
        params["threshold"]          = "0.500000000000000000"
        params["veto_threshold"]     = "0.334000000000000000"
    }
}
```

**After:**
```go
// --- Modify gov ---
if gov, ok := appState["gov"].(map[string]interface{}); ok {
    if params, ok := gov["params"].(map[string]interface{}); ok {
        params["min_deposit"] = []map[string]interface{}{
            {"denom": TokenDenom, "amount": "10000000"},
        }
        // FIX 1: expedited_min_deposit must use TokenDenom, not SDK default "stake".
        // SDK v0.54 default is 5× min_deposit with denom="stake" — always override.
        params["expedited_min_deposit"] = []map[string]interface{}{
            {"denom": TokenDenom, "amount": "50000000"},
        }
        params["max_deposit_period"]    = "172800s"
        params["voting_period"]         = "172800s"
        params["expedited_voting_period"] = "86400s"   // 1 day (matches SDK default, explicit)
        params["quorum"]                = "0.334000000000000000"
        params["threshold"]             = "0.500000000000000000"
        params["veto_threshold"]        = "0.334000000000000000"
        params["expedited_threshold"]   = "0.667000000000000000"
    }
}
```

### Verification
After regenerating, check all three files:
```bash
jq '.app_state.gov.params | {min_deposit, expedited_min_deposit}' chain/genesis.json
# Must show: expedited_min_deposit[0].denom = "ucsov"
```

---

## FIX 2 — RewardsBucket (100 M TOKEN) absent from `bank.balances` and `bank.supply`

### Why it is wrong
The generator defines:
```go
RewardsBucket     = int64(100_000_000) * int64(1_000_000)  // 100M TOKEN in ucsov
OperationalFloat  = CosmosAllocation - RewardsBucket - TreasuryAllocation - ReserveFundAllocation
                  = 700M - 100M - 400M - 200M = 0
```
INV-6 checks the arithmetic on **constants** and passes. But the bank population block (lines
708–729) only writes balances for Treasury and Reserve Fund. No account address, no `auth`
entry, and no `bank.balances` entry are ever created for RewardsBucket.

Result: `bank.supply = 600 M TOKEN` when it should be 700 M TOKEN.

### Decision required before coding
**You must choose an address for the RewardsBucket account.** Options:

| Option | Address to use | Notes |
|--------|---------------|-------|
| A (recommended) | New dedicated bech32 address generated from a fresh mnemonic | Cleanest; use for a rewards-distribution multisig or future module account |
| B (temporary) | `cosmos1398hwtqy7s935s26xezpxp6fdf063s93sd9dfh` (cold multisig) | Simple; team controls it; rename to ops multisig later |
| C | A new Cosmos SDK module account (`x/rewards`) | Requires adding a custom module; out of scope for genesis fix alone |

**The plan below uses Option B as a concrete placeholder.** Replace the address with the real
one when decided. The address is stored in a single constant so changing it touches one line.

### What to change

**File:** `scripts/generate_genesis.go`

**Step 2a — Add the constant** (near the other address constants, around line 96):
```go
// RewardsBucketAddress is the account that holds the 100M TOKEN block-reward reserve.
// MUST be a dedicated address controlled by the team; replace with real address before mainnet.
// Currently set to the cold multisig as a temporary placeholder.
const RewardsBucketAddress = "cosmos1398hwtqy7s935s26xezpxp6fdf063s93sd9dfh"
```

**Step 2b — Add to `auth.accounts`** (inside the block that appends contract accounts,
around lines 682–697 — the section that calls `createGenesisAccount` for each contract addr):

After the loop that adds all four contract addresses to `auth["accounts"]`, add:
```go
// FIX 2: RewardsBucket account must exist in auth before bank credits the balance.
accounts = append(accounts, createGenesisAccount(RewardsBucketAddress))
auth["accounts"] = accounts
```

**Step 2c — Add to `bank.balances`** (inside the bank block, after the Reserve Fund entry,
around line 727):

```go
// FIX 2: RewardsBucket — 100M TOKEN block-reward reserve.
// This balance was defined in the constants but never written to bank.
balances = append(balances, map[string]interface{}{
    "address": RewardsBucketAddress,
    "coins": []map[string]interface{}{
        {"denom": TokenDenom, "amount": fmt.Sprintf("%d", RewardsBucket)},
    },
})
```

**Step 2d — Update `bank.supply`**  
Locate where `bank["supply"]` is written (search for `"supply"` inside the bank block).
Change the total to `TotalSupply - BscEscrowBalance` (= CosmosAllocation = 700 M TOKEN):
```go
bank["supply"] = []map[string]interface{}{
    {"denom": TokenDenom, "amount": fmt.Sprintf("%d", CosmosAllocation)},
}
```
If no explicit supply override exists, add this line inside the `if bank, ok := ...` block,
because the default genesis supply will be wrong after balances are added.

**Step 2e — Strengthen INV-6** to actually check the genesis bank state instead of just
constants (optional but recommended). After the existing INV-6 constant check, add a runtime
check when the genesis file is being verified:
```go
// INV-6b: verify bank supply matches CosmosAllocation exactly.
expectedBankSupply := CosmosAllocation
if bankSupply != expectedBankSupply {
    failures = append(failures, fmt.Sprintf(
        "FAIL [INV-6b]: bank.supply (%d) != cosmos_allocation (%d) — %d TOKEN unaccounted",
        bankSupply, expectedBankSupply, (expectedBankSupply-bankSupply)/1_000_000,
    ))
}
```

### Verification
```bash
jq '.app_state.bank.supply' chain/genesis.json
# Must show: { "amount": "700000000000000", "denom": "ucsov" }

jq '[.app_state.bank.balances[] | select(.address == "cosmos1398hwtqy7s935s26xezpxp6fdf063s93sd9dfh")]' chain/genesis.json
# Must show the 100M TOKEN balance

jq '[.app_state.bank.balances[].coins[0].amount | tonumber] | add' chain/genesis.json
# Must equal 700000000000000
```

---

## FIX 3 — `bridge.supply_cap` is 1 M TOKEN instead of 300 M TOKEN (all three files)

### Why it is wrong
Line 775 hard-codes `1000000000000` (= 1,000,000 TOKEN in ucsov).
The BSC LockBox holds `BscEscrowBalance = 300,000,000 TOKEN`. The cap must equal or exceed
the escrow so that all bridged funds can eventually be minted on the Cosmos side.

### What to change

**File:** `scripts/generate_genesis.go`  
**Location:** line 775 inside the `appState["bridge"]` block

**Before:**
```go
appState["bridge"] = map[string]interface{}{
    "params": map[string]interface{}{
        ...
        "supply_cap": 1000000000000,
        ...
    },
    ...
}
```

**After:**
```go
appState["bridge"] = map[string]interface{}{
    "params": map[string]interface{}{
        ...
        // FIX 3: supply_cap must equal BscEscrowBalance so all BSC-escrowed
        // tokens can be bridged over. Previous value (1M TOKEN) was 300× too small.
        "supply_cap": fmt.Sprintf("%d", BscEscrowBalance),
        ...
    },
    ...
}
```

> Note: `supply_cap` in the genesis JSON is a string (proto `string` type in the bridge module's
> params protobuf). Change the value from the integer literal `1000000000000` to
> `fmt.Sprintf("%d", BscEscrowBalance)`. Confirm the bridge module's proto type; if it is
> `int64`, use `BscEscrowBalance` directly.

### Verification
```bash
jq '.app_state.bridge.params.supply_cap' chain/genesis.json
# Must show: "300000000000000"
```

---

## FIX 4 — `genesis.json` ships with invalid `OWNER_ACTION_REQUIRED` bridge addresses

### Why it is wrong
The generator uses `BRIDGE_GNOSIS_SAFE_ADDRESS`, `BRIDGE_CIRCUIT_BREAKER_ADDRESS`, and
`BRIDGE_LOCKBOX_ADDRESS` env vars. For the `prod` env it panics when they are missing. For
non-prod envs the fallback values are:
```go
gnosisSafe    = "cosmos1devgsaddressplaceholderxxxxx"    // invalid bech32 (bad checksum)
circuitBreaker = "cosmos1devcbaddressplaceholderxxxxx"   // invalid bech32 (bad checksum)
lockbox        = "0xdeace2ea00000000000000000000000000000000"
```
The `genesis.json` currently committed to the repo was generated before these fallbacks existed
and still contains `OWNER_ACTION_REQUIRED:...` strings. After FIX 4 the file must be
regenerated. However, the two dev placeholder addresses are still invalid bech32 — the
validation check (`len(addr) < 10`) is too weak. Fix the fallbacks to use real valid addresses.

### What to change

**File:** `scripts/generate_genesis.go`  
**Location:** non-prod fallback block (lines 743–752)

**Before:**
```go
} else {
    if gnosisSafe == "" {
        gnosisSafe = "cosmos1devgsaddressplaceholderxxxxx"
    }
    if lockbox == "" {
        lockbox = "0xdeace2ea00000000000000000000000000000000"
    }
    if circuitBreaker == "" {
        circuitBreaker = "cosmos1devcbaddressplaceholderxxxxx"
    }
}
```

**After:**
```go
} else {
    // FIX 4: Use real valid bech32 addresses for dev/staging so genesis.json
    // can actually start a node. Replace with real testnet addresses before
    // running a shared devnet. lockbox is a BSC contract address (hex).
    if gnosisSafe == "" {
        gnosisSafe = "cosmos1398hwtqy7s935s26xezpxp6fdf063s93sd9dfh"  // cold multisig (dev stand-in)
    }
    if lockbox == "" {
        lockbox = "0xdeace2ea00000000000000000000000000000000"  // dev EOA stand-in
    }
    if circuitBreaker == "" {
        circuitBreaker = "cosmos1398hwtqy7s935s26xezpxp6fdf063s93sd9dfh"  // cold multisig (dev stand-in)
    }
}
```

Also tighten the validation to enforce a full-length bech32 address instead of just `len > 10`:
```go
// Replace the weak length check with a proper bech32 length check
if !strings.HasPrefix(gnosisSafe, "cosmos1") || len(gnosisSafe) < 39 {
    panic(fmt.Sprintf("invalid gnosis_safe_address %q: must be a valid bech32 cosmos1 address", gnosisSafe))
}
if !strings.HasPrefix(circuitBreaker, "cosmos1") || len(circuitBreaker) < 39 {
    panic(fmt.Sprintf("invalid circuit_breaker_address %q: must be a valid bech32 cosmos1 address", circuitBreaker))
}
```

### Also update `scripts/genesis-ceremony.sh`
Extend Check 2 to cover `genesis.json` as well as `genesis.prod.json`:
```bash
# FIX 4: also validate the Docker-default staging genesis
for GENESIS_FILE in "${GENESIS_BASE}" "$(dirname "${GENESIS_BASE}")/genesis.json"; do
    PLACEHOLDER_HITS=$(grep -c "OWNER_ACTION_REQUIRED" "${GENESIS_FILE}" 2>/dev/null || true)
    if [ "${PLACEHOLDER_HITS}" -gt 0 ]; then
        echo "FAIL: ${GENESIS_FILE} still contains ${PLACEHOLDER_HITS} OWNER_ACTION_REQUIRED placeholder(s)"
        exit 1
    fi
done
```

### Verification
After regenerating:
```bash
grep -c "OWNER_ACTION_REQUIRED" chain/genesis.json      # must be 0
grep -c "OWNER_ACTION_REQUIRED" chain/genesis.dev.json  # must be 0
jq '.app_state.bridge.params | {gnosis_safe_address, circuit_breaker_address, lockbox_address}' chain/genesis.json
# All three must be real addresses, not placeholders
```

---

## FIX 5 — Oracle operator in `genesis.dev.json` uses wrong address type and wrong address

### Why it is wrong
```json
"operators": ["cosmosvaloper17955zu57r288jeja39nc67x6qx9rda3xj263wa"]
```
Two bugs simultaneously:
1. `cosmosvaloper1` is the validator-operator bech32 prefix. Oracle operators must be `cosmos1`
   account addresses. The cosmos1 and cosmosvaloper1 prefixes encode **different key types** —
   a `cosmosvaloper1` address cannot sign oracle commit/reveal transactions.
2. The checksum suffix differs (`xh7wyzw` for the funded account vs `xj263wa` in genesis),
   meaning it is a completely different, unfunded account.

The funded mock oracle operator in `chain/entrypoint.sh`:
```
cosmos17955zu57r288jeja39nc67x6qx9rda3xh7wyzw
```

### What to change

**File:** `scripts/generate_genesis.go`  
**Location:** the section that sets `oracle` state for the dev genesis (search for `"operators"` or the `cosmosvaloper` string in the file)

The dev oracle operator constant should be added alongside the other dev-specific addresses:
```go
// DevOracleOperator is the cosmos1 account address funded in entrypoint.sh for oracle duties.
// MUST use cosmos1 prefix (account address), NOT cosmosvaloper1 (validator operator prefix).
const DevOracleOperator = "cosmos17955zu57r288jeja39nc67x6qx9rda3xh7wyzw"
```

Find where the oracle genesis state is written for `env == "dev"` and replace the hard-coded
operator string:

**Before:**
```go
"operators": []string{"cosmosvaloper17955zu57r288jeja39nc67x6qx9rda3xj263wa"},
```

**After:**
```go
// FIX 5: operator must be cosmos1 (account) prefix, not cosmosvaloper1 (validator key).
// Also corrected the address — previous value had a different checksum from the
// funded account in entrypoint.sh.
"operators": []string{DevOracleOperator},
```

### Verification
```bash
jq '.app_state.oracle.operators' chain/genesis.dev.json
# Must show: ["cosmos17955zu57r288jeja39nc67x6qx9rda3xh7wyzw"]

# Confirm it starts with cosmos1 (not cosmosvaloper1)
jq -r '.app_state.oracle.operators[0]' chain/genesis.dev.json | grep -c "^cosmos1[^v]"
# Must return 1
```

---

## FIX 6 — Wasm `instantiate_config = "Everybody"` on governance contracts (all three files)

### Why it is wrong
Both the global wasm param and all four per-code configs are set to `AccessTypeEverybody`:
```go
// Line 308, 319, 330, 341 — one per contract code:
Permission: wasmtypes.AccessTypeEverybody

// Line 479 — global default:
InstantiateDefaultPermission: wasmtypes.AccessTypeEverybody
```
Anyone on the chain can instantiate a new copy of the Constitution, Treasury, Reserve Fund,
or Governance contract. This allows creating unauthorized shadow governance infrastructure.

`wasmd v0.70.2` `DefaultParams()` defaults to `Everybody` — the chain explicitly sets this
and must explicitly restrict it.

### What to change

**File:** `scripts/generate_genesis.go`

**Step 6a — Restrict per-code `instantiate_config` for prod/staging**

The codes block (lines 301–346) sets `Permission: wasmtypes.AccessTypeEverybody` for all
four codes. Governance contracts must only be instantiable by the cold multisig (or not at
all after bootstrap). Change each of the four `InstantiateConfig` blocks:

```go
// FIX 6: Restrict governance contract instantiation to the cold multisig.
// AccessTypeEverybody allows anyone to deploy shadow governance contracts.
// Use AccessTypeAnyOfAddresses so only the authorized deployer can instantiate.
InstantiateConfig: wasmtypes.AccessConfig{
    Permission: wasmtypes.AccessTypeAnyOfAddresses,
    Addresses:  []string{"cosmos1398hwtqy7s935s26xezpxp6fdf063s93sd9dfh"}, // cold multisig
},
```

Apply this to **all four** codes: Constitution (CodeID 1), Treasury (CodeID 2),
Reserve Fund (CodeID 3), and Governance (CodeID 4).

For `genesis.dev.json`, keep `AccessTypeEverybody` for development convenience:
```go
var codeInstantiatePermission wasmtypes.AccessConfig
if env == "dev" {
    // Dev: allow anyone to instantiate for easy testing
    codeInstantiatePermission = wasmtypes.AccessConfig{
        Permission: wasmtypes.AccessTypeEverybody,
    }
} else {
    // Staging / Prod: restrict to cold multisig only
    codeInstantiatePermission = wasmtypes.AccessConfig{
        Permission: wasmtypes.AccessTypeAnyOfAddresses,
        Addresses:  []string{"cosmos1398hwtqy7s935s26xezpxp6fdf063s93sd9dfh"},
    }
}
// Then use codeInstantiatePermission in every CodeInfo.InstantiateConfig
```

**Step 6b — Restrict global `instantiate_default_permission`**

In the `wasmGenesis` struct (around line 477–479):

**Before:**
```go
wasmGenesis := wasmtypes.GenesisState{
    Params: wasmtypes.Params{
        CodeUploadAccess: wasmtypes.AccessConfig{
            Permission: wasmtypes.AccessTypeNobody,
        },
        InstantiateDefaultPermission: wasmtypes.AccessTypeEverybody,  // ← wrong
    },
    ...
}
```

**After:**
```go
// FIX 6b: default instantiate permission for all codes.
// Nobody for prod (only per-code overrides can grant access).
// Everybody for dev (ease of testing).
var defaultInstantiatePerm wasmtypes.AccessType
if env == "dev" {
    defaultInstantiatePerm = wasmtypes.AccessTypeEverybody
} else {
    defaultInstantiatePerm = wasmtypes.AccessTypeNobody
}

wasmGenesis := wasmtypes.GenesisState{
    Params: wasmtypes.Params{
        CodeUploadAccess: wasmtypes.AccessConfig{
            Permission: wasmtypes.AccessTypeNobody,
        },
        InstantiateDefaultPermission: defaultInstantiatePerm,
    },
    ...
}
```

### Verification
```bash
# Prod / staging
jq '.app_state.wasm.params' chain/genesis.json
# instantiate_default_permission must be "Nobody"

jq '[.app_state.wasm.codes[].code_info.instantiate_config]' chain/genesis.json
# All four must show: { "permission": "AnyOfAddresses", "addresses": ["cosmos1398..."] }

# Dev
jq '.app_state.wasm.params.instantiate_default_permission' chain/genesis.dev.json
# "Everybody" (intentional for dev)
```

---

## FIX 7 — CosmWasm code `creator` fields are circular / bootstrap paradox (all three files)

### Why it is wrong
```
Code 1 (Constitution):   creator = cosmos1shqsrlqalvzwearmrjq8yy788qhzagz6jdq79g  ← Constitution's own contract address
Code 2 (Treasury):       creator = cosmos1wteqf5yrveajhx7zg745p8f46he09gxc2q9fn8  ← Governance addr (Code 4, didn't exist yet)
Code 3 (Reserve Fund):   creator = cosmos1wteqf5yrveajhx7zg745p8f46he09gxc2q9fn8  ← same paradox
Code 4 (Governance):     creator = cosmos1shqsrlqalvzwearmrjq8yy788qhzagz6jdq79g  ← Constitution contract address
```
Same problem for contract `ContractInfo.Creator` fields (lines 353, 378, 403, 428).
Contracts cannot be their own creators; addresses that hadn't been deployed yet cannot have
uploaded earlier code. Any audit, indexer, or access-control logic that queries creator
provenance will return misleading results.

### What to change

**File:** `scripts/generate_genesis.go`

Add a constant for the canonical genesis deployer (one line, easy to update):
```go
// GenesisDeployer is the address recorded as the creator of all wasm codes and
// contracts in genesis. This must be a real account address — it represents
// whoever is notionally responsible for the genesis deployment.
const GenesisDeployer = "cosmos1398hwtqy7s935s26xezpxp6fdf063s93sd9dfh" // cold multisig
```

Then replace every `Creator:` field for the four codes (lines ~306, 317, 328, 339)
and the four contracts (lines ~353, 378, 403, 428):

**Before (example — Code 1):**
```go
Creator: "cosmos1shqsrlqalvzwearmrjq8yy788qhzagz6jdq79g",
```

**After (all eight occurrences — four codes + four contracts):**
```go
Creator: GenesisDeployer,
```

### Verification
```bash
jq '[.app_state.wasm.codes[].code_info.creator] | unique' chain/genesis.prod.json
# Must show exactly one address: ["cosmos1398hwtqy7s935s26xezpxp6fdf063s93sd9dfh"]

jq '[.app_state.wasm.contracts[].contract_info.creator] | unique' chain/genesis.prod.json
# Same — one address only
```

---

## Execution order

Apply the fixes in this sequence to avoid merge conflicts:

```
1. Add constants (GenesisDeployer, DevOracleOperator, RewardsBucketAddress)   — top of file
2. FIX 7  — swap Creator fields (uses new GenesisDeployer constant)
3. FIX 6  — swap instantiate permissions (uses env conditional)
4. FIX 5  — swap oracle operator (uses new DevOracleOperator constant)
5. FIX 3  — change supply_cap to BscEscrowBalance
6. FIX 4  — fix bridge address fallbacks and validation
7. FIX 2  — add RewardsBucketAddress to auth + bank + supply
8. FIX 1  — add expedited_min_deposit to gov block
```

---

## Regenerate and commit genesis files

After all edits, regenerate all three genesis files. The generator compiles Rust contracts;
if the Rust toolchain is unavailable in the agent's environment, generate locally and commit
the resulting JSON files.

```bash
# Dev genesis (no env vars needed)
go run scripts/generate_genesis.go --env dev
# → writes chain/genesis.dev.json

# Staging / Docker-default genesis
go run scripts/generate_genesis.go --env staging
# → writes chain/genesis.json

# Prod genesis (requires real bridge addresses)
BRIDGE_GNOSIS_SAFE_ADDRESS="cosmos1<real>"      \
BRIDGE_LOCKBOX_ADDRESS="0x<real>"               \
BRIDGE_CIRCUIT_BREAKER_ADDRESS="cosmos1<real>"  \
go run scripts/generate_genesis.go --env prod
# → writes chain/genesis.prod.json
```

Run the built-in invariant verifier after each generation:
```bash
go run scripts/generate_genesis.go --verify
# All INV-1 through INV-6 must show [PASS]
```

---

## Post-fix checklist

Run these `jq` assertions against all three output files to confirm every fix landed:

```bash
for F in chain/genesis.json chain/genesis.dev.json chain/genesis.prod.json; do
  echo "=== $F ==="

  # FIX 1: expedited_min_deposit denom must be ucsov
  jq -e '.app_state.gov.params.expedited_min_deposit[0].denom == "ucsov"' $F

  # FIX 2: bank supply must be 700M TOKEN (700000000000000 ucsov)
  jq -e '.app_state.bank.supply[0].amount == "700000000000000"' $F

  # FIX 3: bridge supply_cap must be 300M TOKEN (300000000000000 ucsov)
  jq -e '.app_state.bridge.params.supply_cap == "300000000000000"' $F

  # FIX 4: no OWNER_ACTION_REQUIRED strings anywhere
  grep -c "OWNER_ACTION_REQUIRED" $F && echo "FAIL: placeholders remain" || echo "PASS: no placeholders"

  # FIX 7: all wasm code creators are the same single address
  jq -e '[.app_state.wasm.codes[].code_info.creator] | unique | length == 1' $F
done

# FIX 5: dev oracle operator uses cosmos1 (not cosmosvaloper1)
jq -e '.app_state.oracle.operators[0] | startswith("cosmos1") and (startswith("cosmosvaloper") | not)' chain/genesis.dev.json

# FIX 6: prod instantiate_default_permission is Nobody
jq -e '.app_state.wasm.params.instantiate_default_permission == "Nobody"' chain/genesis.json
jq -e '.app_state.wasm.params.instantiate_default_permission == "Nobody"' chain/genesis.prod.json
jq -e '[.app_state.wasm.codes[].code_info.instantiate_config.permission] | all(. == "AnyOfAddresses")' chain/genesis.json
```

---

## Files changed summary

| File | Changes |
|------|---------|
| `scripts/generate_genesis.go` | All 7 fixes (constants, gov block, bank block, bridge block, wasm codes, wasm contracts, oracle operator, wasm params) |
| `chain/genesis.json` | Regenerated — all 7 fixes reflected |
| `chain/genesis.dev.json` | Regenerated — FIX 1, 2, 3, 5, 7; FIX 6 keeps `Everybody` for dev |
| `chain/genesis.prod.json` | Regenerated — all 7 fixes reflected |
| `scripts/genesis-ceremony.sh` | FIX 4 — extend placeholder check to cover `genesis.json` |

---

## Out of scope (known configuration concerns, not bugs)

These items were confirmed to be real but are design/parameter choices rather than broken
logic. Address them as separate decisions:

| # | Issue | Current | Recommended |
|---|-------|---------|-------------|
| C1 | `slashing.signed_blocks_window` | `"100"` blocks (~100 s) | `"10000"` blocks |
| C2 | `staking.min_commission_rate` | `"0"` | `"0.05"` (5%) |
| C3 | `gov.veto_threshold == quorum` | Both `"0.334"` | Raise veto to `"0.500"` |
| C4 | `app.toml halt-height` duplicate key | Two entries | Remove the duplicate |
| C5 | `app.toml enable-indexer = false` | `false` (Docker fixes it) | Set `true` in the config file directly |
| C6 | `genesis.dev.json wasm.code_upload_access` | `"Everybody"` | Intentional for dev; add a comment |
