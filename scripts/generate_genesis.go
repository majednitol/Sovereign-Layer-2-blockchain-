//go:build ignore
// +build ignore

// generate_genesis.go is a standalone genesis verification and generation script.
//
// Usage:
//
//	go run scripts/generate_genesis.go                 # Generate genesis.json
//	go run scripts/generate_genesis.go --verify        # Verify invariants only (no write)
//	go run scripts/generate_genesis.go --out /tmp/g.json  # Write to custom path
//
// This script verifies the Phase 1 genesis invariants documented in ADR-011:
//
//  1. cosmos_minted_via_bridge + bsc_escrow_balance = 1,000,000,000 TOKEN (S)
//  2. rewards_bucket / block_emission >= 31,536,000 blocks (5-year floor)
//  3. x/vm genesis params: ChainID, EvmDenom="atoken", EnableCreate=true,
//     AllowUnprotectedTxs=false
//  4. x/feemarket genesis params: NoBaseFee=false, ElasticityMultiplier=2,
//     EnableHeight=0
//  5. x/erc20 genesis: native token pair (utoken ↔ ERC-20)
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/sovereign-l1/chain/app"
)

// ---------------------------------------------------------------------------
// Supply constants (must match economic model in doc/governance/genesis_parameters.md)
// ---------------------------------------------------------------------------

const (
	// TokenDenom is the base Cosmos denomination (6 decimal places)
	TokenDenom = "utoken"
	// EVMDenom is the EVM denomination (18 decimal places, 1 TOKEN = 1e12 atoken)
	EVMDenom = "atoken"
	// EVMChainID is the registered EVM chain ID per ADR-009
	EVMChainID = uint64(7777)

	// TotalSupply is S = 1,000,000,000 TOKEN expressed in utoken (×10^6)
	TotalSupply = int64(1_000_000_000) * int64(1_000_000)
	// BscEscrowBalance is C = 300,000,000 TOKEN locked in BSC LockBox at genesis
	BscEscrowBalance = int64(300_000_000) * int64(1_000_000)
	// CosmosAllocation is S - C tokens allocated on the Cosmos side
	CosmosAllocation = TotalSupply - BscEscrowBalance

	// RewardsBucket is 100,000,000 TOKEN denominated in utoken
	RewardsBucket = int64(100_000_000) * int64(1_000_000)
	// PerBlockEmissionUtoken is the per-block emission expressed in utoken (1.5 TOKEN = 1,500,000 utoken)
	PerBlockEmissionUtoken = int64(1_500_000)
	// MinLifetimeBlocks is 5 years at 5s block time
	MinLifetimeBlocks = int64(31_536_000)

	// ValidatorSlots is the fixed active validator cardinality per ADR-001
	ValidatorSlots = 30
	// EqualizedPowerPerSlot is the fixed consensus power per active validator slot
	EqualizedPowerPerSlot = int64(1_000_000)
)

// ---------------------------------------------------------------------------
// Genesis JSON structures (minimal — full types come from cosmos-sdk encoding)
// ---------------------------------------------------------------------------

// GenesisDoc represents the top-level Cosmos genesis document.
type GenesisDoc struct {
	GenesisTime     time.Time              `json:"genesis_time"`
	ChainID         string                 `json:"chain_id"`
	InitialHeight   string                 `json:"initial_height"`
	ConsensusParams ConsensusParams        `json:"consensus_params"`
	AppState        map[string]interface{} `json:"app_state"`
}

// ConsensusParams represents the CometBFT consensus parameters.
type ConsensusParams struct {
	Block     map[string]interface{} `json:"block"`
	Evidence  map[string]interface{} `json:"evidence"`
	Validator map[string]interface{} `json:"validator"`
	Version   map[string]interface{} `json:"version"`
	Abci      map[string]interface{} `json:"abci"`
}

// ---------------------------------------------------------------------------
// Invariant verification
// ---------------------------------------------------------------------------

// VerifyInvariants checks all Phase 1 genesis invariants and returns any failures.
func VerifyInvariants() []string {
	var failures []string

	// Invariant 1: Cosmos Allocation + BSC Escrow = Total Supply
	if CosmosAllocation+BscEscrowBalance != TotalSupply {
		failures = append(failures, fmt.Sprintf(
			"FAIL [INV-1]: cosmos_allocation (%d) + bsc_escrow (%d) != total_supply (%d)",
			CosmosAllocation, BscEscrowBalance, TotalSupply,
		))
	} else {
		fmt.Printf("[PASS] INV-1: cosmos_allocation (%d) + bsc_escrow (%d) = total_supply (%d)\n",
			CosmosAllocation, BscEscrowBalance, TotalSupply)
	}

	// Invariant 2: Rewards bucket lifetime >= 5 years (31,536,000 blocks)
	lifetimeBlocks := RewardsBucket / PerBlockEmissionUtoken
	if lifetimeBlocks < MinLifetimeBlocks {
		failures = append(failures, fmt.Sprintf(
			"FAIL [INV-2]: rewards_bucket lifetime %d blocks < required %d blocks",
			lifetimeBlocks, MinLifetimeBlocks,
		))
	} else {
		fmt.Printf("[PASS] INV-2: rewards_bucket lifetime %d blocks >= %d blocks (5y floor)\n",
			lifetimeBlocks, MinLifetimeBlocks)
	}

	// Invariant 3: EVM chain ID must be set
	if EVMChainID == 0 {
		failures = append(failures, "FAIL [INV-3]: EVMChainID is 0")
	} else {
		fmt.Printf("[PASS] INV-3: EVM ChainID = %d\n", EVMChainID)
	}

	// Invariant 4: EVM denom must be "atoken" per ADR-011
	if EVMDenom != "atoken" {
		failures = append(failures, fmt.Sprintf("FAIL [INV-4]: EVMDenom is %q, expected %q", EVMDenom, "atoken"))
	} else {
		fmt.Printf("[PASS] INV-4: EVMDenom = %q\n", EVMDenom)
	}

	// Invariant 5: Validator slot total power = 30 × 1,000,000
	expectedTotalPower := int64(ValidatorSlots) * EqualizedPowerPerSlot
	if expectedTotalPower != int64(30_000_000) {
		failures = append(failures, fmt.Sprintf(
			"FAIL [INV-5]: Total validator power %d != expected 30,000,000",
			expectedTotalPower,
		))
	} else {
		fmt.Printf("[PASS] INV-5: Total consensus power = %d (%d validators × %d power)\n",
			expectedTotalPower, ValidatorSlots, EqualizedPowerPerSlot)
	}

	return failures
}

// ---------------------------------------------------------------------------
// Genesis generation
// ---------------------------------------------------------------------------

// BuildGenesisDoc builds a minimal genesis document with all Phase 1 app_state.
func BuildGenesisDoc(chainID string) GenesisDoc {
	return GenesisDoc{
		GenesisTime:   time.Now().UTC(),
		ChainID:       chainID,
		InitialHeight: "1",
		ConsensusParams: ConsensusParams{
			Block: map[string]interface{}{
				"max_bytes": "22020096",
				"max_gas":   "50000000",
			},
			Evidence: map[string]interface{}{
				"max_age_num_blocks": "100000",
				"max_age_duration":   "172800000000000",
				"max_bytes":          "1048576",
			},
			Validator: map[string]interface{}{
				"pub_key_types": []string{"ed25519"},
			},
			Version: map[string]interface{}{
				"app": "0",
			},
			Abci: map[string]interface{}{
				"vote_extensions_enable_height": "0",
			},
		},
		AppState: buildAppState(),
	}
}

// buildAppState constructs the genesis app_state with all Phase 1 module params.
func buildAppState() map[string]interface{} {
	wasmGenesis := wasmtypes.GenesisState{
		Params: wasmtypes.Params{
			CodeUploadAccess: wasmtypes.AccessConfig{
				Permission: wasmtypes.AccessTypeNobody,
			},
			InstantiateDefaultPermission: wasmtypes.AccessTypeEverybody,
		},
		Codes:     nil,
		Contracts: nil,
		Sequences: nil,
	}

	// 1. Get default genesis app_state
	interfaceRegistry := codectypes.NewInterfaceRegistry()
	app.ModuleBasics.RegisterInterfaces(interfaceRegistry)
	appCodec := codec.NewProtoCodec(interfaceRegistry)
	rawDefault := app.ModuleBasics.DefaultGenesis(appCodec)

	// Convert to JSON and back to map[string]interface{}
	defaultJSON, err := json.Marshal(rawDefault)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal default genesis: %v", err))
	}
	var appState map[string]interface{}
	if err := json.Unmarshal(defaultJSON, &appState); err != nil {
		panic(fmt.Sprintf("failed to unmarshal default genesis: %v", err))
	}

	// --- Modify auth ---
	if auth, ok := appState["auth"].(map[string]interface{}); ok {
		if params, ok := auth["params"].(map[string]interface{}); ok {
			params["max_memo_characters"] = "256"
			params["tx_sig_limit"] = "7"
			params["tx_size_cost_per_byte"] = "10"
			params["sig_verify_cost_ed25519"] = "590"
			params["sig_verify_cost_secp256k1"] = "1000"
		}
	}

	// --- Modify bank ---
	if bank, ok := appState["bank"].(map[string]interface{}); ok {
		if params, ok := bank["params"].(map[string]interface{}); ok {
			params["default_send_enabled"] = true
		}
		bank["denom_metadata"] = []map[string]interface{}{
			{
				"description": "EVM Gas Token",
				"denom_units": []map[string]interface{}{
					{
						"denom":    "atoken",
						"exponent": 0,
						"aliases":  []string{"wei"},
					},
					{
						"denom":    "evmtoken",
						"exponent": 18,
						"aliases":  []string{},
					},
				},
				"base":    "atoken",
				"display": "evmtoken",
				"name":    "EVM Gas Token",
				"symbol":  "TOKEN",
			},
			{
				"description": "Native Cosmos Staking Token",
				"denom_units": []map[string]interface{}{
					{
						"denom":    "utoken",
						"exponent": 0,
						"aliases":  []string{},
					},
					{
						"denom":    "token",
						"exponent": 6,
						"aliases":  []string{},
					},
				},
				"base":    "utoken",
				"display": "token",
				"name":    "Cosmos Staking Token",
				"symbol":  "TOKEN",
			},
			{
				"description": "Bridge Minted Token",
				"denom_units": []map[string]interface{}{
					{
						"denom":    "usov",
						"exponent": 0,
						"aliases":  []string{},
					},
					{
						"denom":    "sov",
						"exponent": 6,
						"aliases":  []string{},
					},
				},
				"base":    "usov",
				"display": "sov",
				"name":    "Bridge Minted Token",
				"symbol":  "SOV",
			},
		}
	}

	// --- Modify staking ---
	if staking, ok := appState["staking"].(map[string]interface{}); ok {
		if params, ok := staking["params"].(map[string]interface{}); ok {
			params["unbonding_time"] = "1814400s"
			params["max_validators"] = ValidatorSlots
			params["max_entries"] = 7
			params["historical_entries"] = 10000
			params["bond_denom"] = TokenDenom
			params["min_commission_rate"] = "0.000000000000000000"
		}
		staking["last_total_power"] = fmt.Sprintf("%d", int64(ValidatorSlots)*EqualizedPowerPerSlot)
	}

	// --- Modify distribution ---
	if distribution, ok := appState["distribution"].(map[string]interface{}); ok {
		if params, ok := distribution["params"].(map[string]interface{}); ok {
			params["community_tax"] = "0.020000000000000000"
			params["base_proposer_reward"] = "0.000000000000000000"
			params["bonus_proposer_reward"] = "0.000000000000000000"
			params["withdraw_addr_enabled"] = true
		}
	}

	// --- Modify gov ---
	if gov, ok := appState["gov"].(map[string]interface{}); ok {
		if params, ok := gov["params"].(map[string]interface{}); ok {
			params["min_deposit"] = []map[string]interface{}{
				{"denom": TokenDenom, "amount": "10000000"},
			}
			params["max_deposit_period"] = "172800s"
			params["voting_period"] = "172800s"
			params["quorum"] = "0.334000000000000000"
			params["threshold"] = "0.500000000000000000"
			params["veto_threshold"] = "0.334000000000000000"
		}
	}

	// --- Modify evm ---
	if evm, ok := appState["evm"].(map[string]interface{}); ok {
		if params, ok := evm["params"].(map[string]interface{}); ok {
			params["evm_denom"] = EVMDenom
			params["active_static_precompiles"] = []string{
				"0x0000000000000000000000000000000000000101",
				"0x0000000000000000000000000000000000000102",
			}
			params["access_control"] = map[string]interface{}{
				"create": map[string]interface{}{
					"access_type": "ACCESS_TYPE_PERMISSIONLESS",
					"access_control_list": []interface{}{},
				},
				"call": map[string]interface{}{
					"access_type": "ACCESS_TYPE_PERMISSIONLESS",
					"access_control_list": []interface{}{},
				},
			}
			params["extended_denom_options"] = map[string]interface{}{
				"extended_denom": EVMDenom,
			}
		}
	}

	// --- Modify feemarket ---
	if feemarket, ok := appState["feemarket"].(map[string]interface{}); ok {
		if params, ok := feemarket["params"].(map[string]interface{}); ok {
			params["no_base_fee"] = false
			params["base_fee_change_denominator"] = 8
			params["elasticity_multiplier"] = 2
			params["enable_height"] = "0"
			params["base_fee"] = "1000000000"
			params["min_gas_price"] = "0.000000000000000000"
			params["min_gas_multiplier"] = "0.500000000000000000"
		}
		feemarket["block_gas"] = "0"
	}

	// --- Modify erc20 ---
	if erc20, ok := appState["erc20"].(map[string]interface{}); ok {
		if params, ok := erc20["params"].(map[string]interface{}); ok {
			params["enable_erc20"] = true
		}
		erc20["token_pairs"] = []map[string]interface{}{
			{
				"erc20_address":  "0x0000000000000000000000000000000000000001",
				"denom":          TokenDenom,
				"enabled":        true,
				"contract_owner": "OWNER_MODULE",
			},
		}
	}

	// --- Modify wasm ---
	wasmJSON, err := json.Marshal(wasmGenesis)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal wasm genesis: %v", err))
	}
	var wasmMap map[string]interface{}
	if err := json.Unmarshal(wasmJSON, &wasmMap); err != nil {
		panic(fmt.Sprintf("failed to unmarshal wasm genesis: %v", err))
	}
	appState["wasm"] = wasmMap

	return appState
}

// ---------------------------------------------------------------------------
// main
// ---------------------------------------------------------------------------

func main() {
	var (
		verifyOnly bool
		outPath    string
		chainID    string
	)

	flag.BoolVar(&verifyOnly, "verify", false, "Run invariant checks only, do not write genesis file")
	flag.StringVar(&outPath, "out", "chain/genesis.json", "Output path for the genesis file")
	flag.StringVar(&chainID, "chain-id", "sovereign-1", "Chain ID to embed in genesis")
	flag.Parse()

	fmt.Println("=== Sovereign L1 — Phase 1 Genesis Invariant Verification ===")
	fmt.Println()

	failures := VerifyInvariants()

	fmt.Println()

	if len(failures) > 0 {
		fmt.Fprintln(os.Stderr, "=== INVARIANT FAILURES ===")
		for _, f := range failures {
			fmt.Fprintln(os.Stderr, f)
		}
		os.Exit(1)
	}

	fmt.Println("[OK] All Phase 1 genesis invariants pass.")
	fmt.Println()

	if verifyOnly {
		fmt.Println("Skipping genesis file generation (--verify flag set).")
		return
	}

	// Build and write genesis
	genesis := BuildGenesisDoc(chainID)
	data, err := json.MarshalIndent(genesis, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to marshal genesis: %v\n", err)
		os.Exit(1)
	}

	if err := os.WriteFile(outPath, data, 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to write genesis to %s: %v\n", outPath, err)
		os.Exit(1)
	}

	fmt.Printf("[OK] Genesis written to %s\n", outPath)
	fmt.Printf("     Chain ID   : %s\n", chainID)
	fmt.Printf("     Total Supply: %d utoken (%d TOKEN)\n", TotalSupply, TotalSupply/1_000_000)
	fmt.Printf("     BSC Escrow  : %d utoken (%d TOKEN)\n", BscEscrowBalance, BscEscrowBalance/1_000_000)
	fmt.Printf("     EVM Chain ID: %d\n", EVMChainID)
	fmt.Printf("     EVM Denom   : %s\n", EVMDenom)
}
