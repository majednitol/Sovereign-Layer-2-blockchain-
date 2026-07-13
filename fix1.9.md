# Fix Plan: Task 1.9 — Fund Genesis Contract Accounts

**File:** `scripts/generate_genesis.go` (only file that needs changes)  
**Test file:** `e2e/phase_1_integration_test.go` (one addition)  
**Reviewer:** Do not delete any existing code — only add new code at the locations specified.

---

## Root Cause

Four CosmWasm governance contracts are deployed in `wasm.contracts` at genesis but their
addresses are **never added** to `auth.accounts` or `bank.balances`.

Genesis module init order in `app.go` is:

```
auth → bank → … → wasm
```

Bank's `InitGenesis` runs **before** wasmd creates contract accounts. If the contract
addresses are not in `auth.accounts` when bank initialises, bank panics with
`"account does not exist"` when it tries to credit balances to those addresses.

---

## Contract Addresses (already hardcoded in the script — do not change them)

| Contract | Address |
|----------|---------|
| Constitution | `cosmos1shqsrlqalvzwearmrjq8yy788qhzagz6jdq79g` |
| Treasury | `cosmos1w8kmv94zcf8yysgw9dp8yzq6ffe2e8m0uj8dm0` |
| Reserve Fund | `cosmos1dag3w9ydhzmwpvd6asrt8elexa8s27ph7895jc` |
| Governance | `cosmos1wteqf5yrveajhx7zg745p8f46he09gxc2q9fn8` |

---

## Change 1 — Add funding constants inside the existing `const` block

**File:** `scripts/generate_genesis.go`  
**Location:** Inside the existing `const ( … )` block, directly **after** the line
`EqualizedPowerPerSlot = int64(1_000_000)` and **before** the closing `)`.

Add:

```go
	// ---------------------------------------------------------------------------
	// Genesis Contract Funding Allocations
	// TreasuryAllocation + ReserveFundAllocation + RewardsBucket must equal CosmosAllocation.
	// CONFIRM values against doc/governance/genesis_parameters.md before mainnet.
	// ---------------------------------------------------------------------------

	// TreasuryAllocation is the ucsov balance credited to the Treasury contract at genesis.
	TreasuryAllocation = int64(400_000_000) * int64(1_000_000) // 400,000,000 TOKEN

	// ReserveFundAllocation is the ucsov balance credited to the Reserve Fund contract at genesis.
	ReserveFundAllocation = int64(200_000_000) * int64(1_000_000) // 200,000,000 TOKEN

	// OperationalFloat is what remains after contracts and rewards are funded.
	// Set to 0 here: full CosmosAllocation is accounted for on-chain.
	// If the ops multisig needs a float, reduce TreasuryAllocation by that amount
	// and add a MultisigAllocation constant — INV-6 will enforce the sum stays valid.
	OperationalFloat = CosmosAllocation - RewardsBucket - TreasuryAllocation - ReserveFundAllocation
	// 700,000,000 − 100,000,000 − 400,000,000 − 200,000,000 = 0 TOKEN
```

> **Note on amounts:** Treasury=400M and ReserveFund=200M fully account for the 600M non-rewards
> Cosmos allocation, leaving `OperationalFloat = 0`. If the team needs a multisig float,
> reduce `TreasuryAllocation` by that amount and add a `MultisigAllocation` constant. INV-6
> (Change 2 below) will catch any arithmetic mistake at genesis generation time.

---

## Change 2 — Add `contractAddrs` var block

**File:** `scripts/generate_genesis.go`  
**Location:** **After** the `const ( … )` block closes and **before** the first function
(`func init()` or `func VerifyInvariants()`).

Add:

```go
// contractAddrs holds the bech32 addresses of the four governance CosmWasm contracts.
// These MUST match the ContractAddress fields in buildAppState()'s wasm genesis section
// and the addresses computed by chain/app/wasm.go's init() via types.NewModuleAddress.
var contractAddrs = struct {
	Constitution string
	Treasury     string
	ReserveFund  string
	Governance   string
}{
	Constitution: "cosmos1shqsrlqalvzwearmrjq8yy788qhzagz6jdq79g",
	Treasury:     "cosmos1w8kmv94zcf8yysgw9dp8yzq6ffe2e8m0uj8dm0",
	ReserveFund:  "cosmos1dag3w9ydhzmwpvd6asrt8elexa8s27ph7895jc",
	Governance:   "cosmos1wteqf5yrveajhx7zg745p8f46he09gxc2q9fn8",
}
```

---

## Change 3 — Add INV-6 inside `VerifyInvariants()`

**File:** `scripts/generate_genesis.go`  
**Location:** Inside `func VerifyInvariants() []string`, **after the INV-5 block** and
**before** `return failures`.

Add:

```go
	// Invariant 6: Contract funding + rewards bucket must equal Cosmos allocation exactly.
	// This prevents tokens being created from thin air or silently lost.
	contractFunding := TreasuryAllocation + ReserveFundAllocation
	coveredByOnChain := contractFunding + RewardsBucket + OperationalFloat
	if coveredByOnChain != CosmosAllocation {
		failures = append(failures, fmt.Sprintf(
			"FAIL [INV-6]: contract_funding (%d) + rewards_bucket (%d) + operational_float (%d) = %d != cosmos_allocation (%d) — tokens created from thin air or lost",
			contractFunding, RewardsBucket, OperationalFloat, coveredByOnChain, CosmosAllocation,
		))
	} else {
		fmt.Printf("[PASS] INV-6: treasury (%d) + reserve_fund (%d) + rewards_bucket (%d) + float (%d) = cosmos_allocation (%d)\n",
			TreasuryAllocation, ReserveFundAllocation, RewardsBucket, OperationalFloat, CosmosAllocation)
	}
```

---

## Change 4 — Inject contract accounts and balances in `buildAppState()`

**File:** `scripts/generate_genesis.go`  
**Location:** Inside `func buildAppState(env string) map[string]interface{}`.

Find this exact line:

```go
	appState["wasm"] = wasmMap
```

**Immediately after that line** (before `// --- Modify bridge ---`), insert:

```go
	// --- Fund governance contract accounts (applies to BOTH dev and prod) ---
	//
	// All four contract addresses must be in auth.accounts before bank's InitGenesis
	// runs, because genesis module order is: auth → bank → … → wasm.
	// Bank will panic if it tries to credit a balance to a non-existent account.
	//
	// Only Treasury and Reserve Fund receive token balances; Constitution and
	// Governance receive auth.accounts entries only (they hold no tokens at genesis).

	allContractAddrs := []string{
		contractAddrs.Constitution,
		contractAddrs.Treasury,
		contractAddrs.ReserveFund,
		contractAddrs.Governance,
	}

	if auth, ok := appState["auth"].(map[string]interface{}); ok {
		var accounts []interface{}
		if accs, ok := auth["accounts"].([]interface{}); ok {
			accounts = accs
		}
		for _, addr := range allContractAddrs {
			accounts = append(accounts, createGenesisAccount(addr))
		}
		auth["accounts"] = accounts
	}

	if bank, ok := appState["bank"].(map[string]interface{}); ok {
		var balances []interface{}
		if bals, ok := bank["balances"].([]interface{}); ok {
			balances = bals
		}

		// Treasury: holds the main protocol reserve in ucsov.
		balances = append(balances, map[string]interface{}{
			"address": contractAddrs.Treasury,
			"coins": []map[string]interface{}{
				{"denom": TokenDenom, "amount": fmt.Sprintf("%d", TreasuryAllocation)},
			},
		})

		// Reserve Fund: holds the safety reserve in ucsov.
		balances = append(balances, map[string]interface{}{
			"address": contractAddrs.ReserveFund,
			"coins": []map[string]interface{}{
				{"denom": TokenDenom, "amount": fmt.Sprintf("%d", ReserveFundAllocation)},
			},
		})

		// Constitution and Governance receive zero tokens — no balances entry needed.

		bank["balances"] = balances
	}
```

---

## Change 5 — Tighten the prod guard to verify contract accounts are present

**File:** `scripts/generate_genesis.go`  
**Location:** Inside the existing `else if env == "prod"` block (the block that currently
panics if dev addresses are found). **After the existing dev-address check loop**, add:

```go
		// Verify all four governance contract accounts ARE present in prod genesis.
		// Missing entries would cause bank InitGenesis to panic on chain start.
		requiredContracts := map[string]bool{
			contractAddrs.Constitution: false,
			contractAddrs.Treasury:     false,
			contractAddrs.ReserveFund:  false,
			contractAddrs.Governance:   false,
		}
		if accs, ok := auth["accounts"].([]interface{}); ok {
			for _, acc := range accs {
				if accMap, ok := acc.(map[string]interface{}); ok {
					addr, _ := accMap["address"].(string)
					if _, wanted := requiredContracts[addr]; wanted {
						requiredContracts[addr] = true
					}
				}
			}
		}
		for addr, found := range requiredContracts {
			if !found {
				panic(fmt.Sprintf("production genesis is missing required governance contract account: %s", addr))
			}
		}
```

---

## Change 6 — Add INV-6 to the E2E supply math test

**File:** `e2e/phase_1_integration_test.go`  
**Location:** Inside `func TestPhase1Integration_SupplyMathInvariants`, **before the
closing `}`** of that function (currently ends around line 379).

Add:

```go
	// Invariant 6: Contract funding must be fully accounted for within Cosmos allocation
	const (
		inv6TreasuryAlloc    = int64(400_000_000) * int64(1_000_000)
		inv6ReserveAlloc     = int64(200_000_000) * int64(1_000_000)
		inv6RewardsBucket    = int64(100_000_000) * int64(1_000_000)
		inv6CosmosAllocation = int64(700_000_000) * int64(1_000_000)
	)
	inv6OnChain := inv6TreasuryAlloc + inv6ReserveAlloc + inv6RewardsBucket
	if inv6OnChain > inv6CosmosAllocation {
		t.Fatalf("FAIL INV-6: on_chain_allocation (%d) > cosmos_allocation (%d) — tokens created from thin air",
			inv6OnChain, inv6CosmosAllocation)
	}
	operationalFloat := inv6CosmosAllocation - inv6OnChain
	t.Logf("[PASS] INV-6: treasury (%d) + reserve_fund (%d) + rewards_bucket (%d) + float (%d) = cosmos_allocation (%d)",
		inv6TreasuryAlloc, inv6ReserveAlloc, inv6RewardsBucket, operationalFloat, inv6CosmosAllocation)
```

---

## Verification commands (run after all changes are applied)

```bash
# 1. Confirm INV-6 passes alongside existing INV-1 through INV-5
go run scripts/generate_genesis.go --verify

# Expected output includes:
# [PASS] INV-6: treasury (...) + reserve_fund (...) + rewards_bucket (...) + float (0) = cosmos_allocation (...)

# 2. Regenerate dev genesis with the funded contracts
go run scripts/generate_genesis.go --env dev

# 3. Confirm Treasury appears in bank.balances
jq '.app_state.bank.balances[] | select(.address == "cosmos1w8kmv94zcf8yysgw9dp8yzq6ffe2e8m0uj8dm0")' \
  chain/genesis.dev.json

# Expected:
# { "address": "cosmos1w8km...", "coins": [{ "denom": "ucsov", "amount": "400000000000000" }] }

# 4. Confirm Reserve Fund appears in bank.balances
jq '.app_state.bank.balances[] | select(.address == "cosmos1dag3w9ydhzmwpvd6asrt8elexa8s27ph7895jc")' \
  chain/genesis.dev.json

# Expected:
# { "address": "cosmos1dag3...", "coins": [{ "denom": "ucsov", "amount": "200000000000000" }] }

# 5. Confirm all 4 contracts are in auth.accounts
jq '[.app_state.auth.accounts[] | .address] | map(select(test("cosmos1shq|cosmos1w8k|cosmos1dag|cosmos1wte")))' \
  chain/genesis.dev.json

# Expected: array of 4 addresses

# 6. Run the E2E supply math tests
go test -v -run TestPhase1Integration_SupplyMathInvariants ./e2e/...
```

---

## Summary table

| # | What | File | Where |
|---|------|------|-------|
| 1 | Add `TreasuryAllocation`, `ReserveFundAllocation`, `OperationalFloat` constants | `scripts/generate_genesis.go` | End of existing `const` block |
| 2 | Add `contractAddrs` var with 4 addresses | `scripts/generate_genesis.go` | After `const` block, before first function |
| 3 | Add INV-6 to `VerifyInvariants()` | `scripts/generate_genesis.go` | Before `return failures` |
| 4 | Inject `auth.accounts` (×4) + `bank.balances` (Treasury, Reserve Fund) in `buildAppState()` | `scripts/generate_genesis.go` | Right after `appState["wasm"] = wasmMap` |
| 5 | Add prod guard verifying 4 contract accounts are present | `scripts/generate_genesis.go` | Inside `else if env == "prod"` block |
| 6 | Add INV-6 assertion to E2E supply math test | `e2e/phase_1_integration_test.go` | Before closing `}` of `TestPhase1Integration_SupplyMathInvariants` |
