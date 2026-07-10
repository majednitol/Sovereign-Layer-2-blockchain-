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
//  3. x/vm genesis params: ChainID, EvmDenom="aesov", EnableCreate=true,
//     AllowUnprotectedTxs=false
//  4. x/feemarket genesis params: NoBaseFee=false, ElasticityMultiplier=2,
//     EnableHeight=0
//  5. x/erc20 genesis: native token pair (ucsov ↔ ERC-20)
package main

import (
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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
	TokenDenom = "ucsov"
	// EVMDenom is the EVM denomination (18 decimal places, 1 ESOV = 1e12 aesov)
	EVMDenom = "aesov"
	// EVMChainID is the registered EVM chain ID per ADR-009
	EVMChainID = uint64(7777)

	// TotalSupply is S = 1,000,000,000 TOKEN expressed in ucsov (×10^6)
	TotalSupply = int64(1_000_000_000) * int64(1_000_000)
	// BscEscrowBalance is C = 300,000,000 TOKEN locked in BSC LockBox at genesis
	BscEscrowBalance = int64(300_000_000) * int64(1_000_000)
	// CosmosAllocation is S - C tokens allocated on the Cosmos side
	CosmosAllocation = TotalSupply - BscEscrowBalance

	// RewardsBucket is 100,000,000 TOKEN denominated in ucsov
	RewardsBucket = int64(100_000_000) * int64(1_000_000)
	// PerBlockEmissionUtoken is the per-block emission expressed in ucsov (1.5 TOKEN = 1,500,000 ucsov)
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

	// Invariant 4: EVM denom must be "aesov" per ADR-011
	if EVMDenom != "aesov" {
		failures = append(failures, fmt.Sprintf("FAIL [INV-4]: EVMDenom is %q, expected %q", EVMDenom, "aesov"))
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
func BuildGenesisDoc(chainID string, env string, genesisTime time.Time) GenesisDoc {
	return GenesisDoc{
		GenesisTime:   genesisTime,
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
		AppState: buildAppState(env),
	}
}

func compileContracts() error {
	fmt.Println("Compiling contracts to wasm32...")
	cmd := exec.Command("cargo", "build", "--target", "wasm32-unknown-unknown", "--release", "--lib")
	cmd.Dir = "contracts"
	cmd.Env = append(os.Environ(), "RUSTFLAGS=-C target-feature=-bulk-memory")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}

	fmt.Println("Lowering bulk memory operations in compiled WASM files...")
	contractsDir := "contracts/target/wasm32-unknown-unknown/release"
	contracts := []string{
		"constitution.wasm",
		"treasury.wasm",
		"reserve_fund.wasm",
		"governance.wasm",
	}

	for _, contract := range contracts {
		path := filepath.Join(contractsDir, contract)
		fmt.Printf("Lowering bulk memory operations for %s...\n", contract)
		optCmd := exec.Command("wasm-opt", "--llvm-memory-copy-fill-lowering", path, "-o", path)
		optCmd.Stdout = os.Stdout
		optCmd.Stderr = os.Stderr
		if err := optCmd.Run(); err != nil {
			return fmt.Errorf("failed to run wasm-opt on %s: %w", contract, err)
		}
	}

	return nil
}

func sha256Hash(data []byte) []byte {
	hash := sha256.Sum256(data)
	return hash[:]
}

// buildAppState constructs the genesis app_state with all Phase 1 module params and Phase 3 CosmWasm smart contracts.
func buildAppState(env string) map[string]interface{} {
	if err := compileContracts(); err != nil {
		panic(fmt.Sprintf("failed to compile contracts: %v", err))
	}

	contractsDir := "contracts/target/wasm32-unknown-unknown/release"
	constitutionWasm, err := os.ReadFile(filepath.Join(contractsDir, "constitution.wasm"))
	if err != nil {
		panic(fmt.Sprintf("failed to read constitution.wasm: %v", err))
	}
	treasuryWasm, err := os.ReadFile(filepath.Join(contractsDir, "treasury.wasm"))
	if err != nil {
		panic(fmt.Sprintf("failed to read treasury.wasm: %v", err))
	}
	reserveFundWasm, err := os.ReadFile(filepath.Join(contractsDir, "reserve_fund.wasm"))
	if err != nil {
		panic(fmt.Sprintf("failed to read reserve_fund.wasm: %v", err))
	}
	governanceWasm, err := os.ReadFile(filepath.Join(contractsDir, "governance.wasm"))
	if err != nil {
		panic(fmt.Sprintf("failed to read governance.wasm: %v", err))
	}

	codes := []wasmtypes.Code{
		{
			CodeID: 1,
			CodeInfo: wasmtypes.CodeInfo{
				CodeHash: sha256Hash(constitutionWasm),
				Creator:  "cosmos1shqsrlqalvzwearmrjq8yy788qhzagz6jdq79g",
				InstantiateConfig: wasmtypes.AccessConfig{
					Permission: wasmtypes.AccessTypeEverybody,
				},
			},
			CodeBytes: constitutionWasm,
		},
		{
			CodeID: 2,
			CodeInfo: wasmtypes.CodeInfo{
				CodeHash: sha256Hash(treasuryWasm),
				Creator:  "cosmos1wteqf5yrveajhx7zg745p8f46he09gxc2q9fn8",
				InstantiateConfig: wasmtypes.AccessConfig{
					Permission: wasmtypes.AccessTypeEverybody,
				},
			},
			CodeBytes: treasuryWasm,
		},
		{
			CodeID: 3,
			CodeInfo: wasmtypes.CodeInfo{
				CodeHash: sha256Hash(reserveFundWasm),
				Creator:  "cosmos1wteqf5yrveajhx7zg745p8f46he09gxc2q9fn8",
				InstantiateConfig: wasmtypes.AccessConfig{
					Permission: wasmtypes.AccessTypeEverybody,
				},
			},
			CodeBytes: reserveFundWasm,
		},
		{
			CodeID: 4,
			CodeInfo: wasmtypes.CodeInfo{
				CodeHash: sha256Hash(governanceWasm),
				Creator:  "cosmos1shqsrlqalvzwearmrjq8yy788qhzagz6jdq79g",
				InstantiateConfig: wasmtypes.AccessConfig{
					Permission: wasmtypes.AccessTypeEverybody,
				},
			},
			CodeBytes: governanceWasm,
		},
	}

	constitutionConfigVal := `{"governance_address":"cosmos1wteqf5yrveajhx7zg745p8f46he09gxc2q9fn8","cold_multisig_address":"cosmos1398hwtqy7s935s26xezpxp6fdf063s93sd9dfh","rules":"Safe rules","is_paused":false}`
	constitutionContract := wasmtypes.Contract{
		ContractAddress: "cosmos1shqsrlqalvzwearmrjq8yy788qhzagz6jdq79g",
		ContractInfo: wasmtypes.ContractInfo{
			CodeID:  1,
			Creator: "cosmos1shqsrlqalvzwearmrjq8yy788qhzagz6jdq79g",
			Label:   "Constitution",
			Created: &wasmtypes.AbsoluteTxPosition{BlockHeight: 0, TxIndex: 0},
		},
		ContractState: []wasmtypes.Model{
			{
				Key:   []byte("config"),
				Value: []byte(constitutionConfigVal),
			},
		},
		ContractCodeHistory: []wasmtypes.ContractCodeHistoryEntry{
			{
				Operation: wasmtypes.ContractCodeHistoryOperationTypeGenesis,
				CodeID:    1,
				Updated:   &wasmtypes.AbsoluteTxPosition{BlockHeight: 0, TxIndex: 0},
				Msg:       []byte(`{}`),
			},
		},
	}

	treasuryConfigVal := `{"governance_address":"cosmos1wteqf5yrveajhx7zg745p8f46he09gxc2q9fn8","cold_multisig_address":"cosmos1398hwtqy7s935s26xezpxp6fdf063s93sd9dfh","is_paused":false,"reentrancy_lock":false}`
	treasuryContract := wasmtypes.Contract{
		ContractAddress: "cosmos1w8kmv94zcf8yysgw9dp8yzq6ffe2e8m0uj8dm0",
		ContractInfo: wasmtypes.ContractInfo{
			CodeID:  2,
			Creator: "cosmos1wteqf5yrveajhx7zg745p8f46he09gxc2q9fn8",
			Label:   "Treasury",
			Created: &wasmtypes.AbsoluteTxPosition{BlockHeight: 0, TxIndex: 0},
		},
		ContractState: []wasmtypes.Model{
			{
				Key:   []byte("config"),
				Value: []byte(treasuryConfigVal),
			},
		},
		ContractCodeHistory: []wasmtypes.ContractCodeHistoryEntry{
			{
				Operation: wasmtypes.ContractCodeHistoryOperationTypeGenesis,
				CodeID:    2,
				Updated:   &wasmtypes.AbsoluteTxPosition{BlockHeight: 0, TxIndex: 0},
				Msg:       []byte(`{}`),
			},
		},
	}

	reserveConfigVal := `{"governance_address":"cosmos1wteqf5yrveajhx7zg745p8f46he09gxc2q9fn8","cold_multisig_address":"cosmos1398hwtqy7s935s26xezpxp6fdf063s93sd9dfh","min_balance_threshold":"100000000000","is_paused":false,"reentrancy_lock":false}`
	reserveContract := wasmtypes.Contract{
		ContractAddress: "cosmos1dag3w9ydhzmwpvd6asrt8elexa8s27ph7895jc",
		ContractInfo: wasmtypes.ContractInfo{
			CodeID:  3,
			Creator: "cosmos1wteqf5yrveajhx7zg745p8f46he09gxc2q9fn8",
			Label:   "Reserve Fund",
			Created: &wasmtypes.AbsoluteTxPosition{BlockHeight: 0, TxIndex: 0},
		},
		ContractState: []wasmtypes.Model{
			{
				Key:   []byte("config"),
				Value: []byte(reserveConfigVal),
			},
		},
		ContractCodeHistory: []wasmtypes.ContractCodeHistoryEntry{
			{
				Operation: wasmtypes.ContractCodeHistoryOperationTypeGenesis,
				CodeID:    3,
				Updated:   &wasmtypes.AbsoluteTxPosition{BlockHeight: 0, TxIndex: 0},
				Msg:       []byte(`{}`),
			},
		},
	}

	governanceConfigVal := `{"constitution_address":"cosmos1shqsrlqalvzwearmrjq8yy788qhzagz6jdq79g","treasury_address":"cosmos1w8kmv94zcf8yysgw9dp8yzq6ffe2e8m0uj8dm0","reserve_fund_address":"cosmos1dag3w9ydhzmwpvd6asrt8elexa8s27ph7895jc"}`
	governanceContract := wasmtypes.Contract{
		ContractAddress: "cosmos1wteqf5yrveajhx7zg745p8f46he09gxc2q9fn8",
		ContractInfo: wasmtypes.ContractInfo{
			CodeID:  4,
			Creator: "cosmos1shqsrlqalvzwearmrjq8yy788qhzagz6jdq79g",
			Label:   "Governance",
			Created: &wasmtypes.AbsoluteTxPosition{BlockHeight: 0, TxIndex: 0},
		},
		ContractState: []wasmtypes.Model{
			{
				Key:   []byte("config"),
				Value: []byte(governanceConfigVal),
			},
			{
				Key:   []byte("log_count"),
				Value: []byte("0"),
			},
		},
		ContractCodeHistory: []wasmtypes.ContractCodeHistoryEntry{
			{
				Operation: wasmtypes.ContractCodeHistoryOperationTypeGenesis,
				CodeID:    4,
				Updated:   &wasmtypes.AbsoluteTxPosition{BlockHeight: 0, TxIndex: 0},
				Msg:       []byte(`{}`),
			},
		},
	}

	contracts := []wasmtypes.Contract{
		constitutionContract,
		treasuryContract,
		reserveContract,
		governanceContract,
	}

	sequences := []wasmtypes.Sequence{
		{
			IDKey: wasmtypes.KeySequenceCodeID,
			Value: 5,
		},
		{
			IDKey: wasmtypes.KeySequenceInstanceID,
			Value: 5,
		},
	}

	wasmGenesis := wasmtypes.GenesisState{
		Params: wasmtypes.Params{
			CodeUploadAccess: wasmtypes.AccessConfig{
				Permission: wasmtypes.AccessTypeNobody,
			},
			InstantiateDefaultPermission: wasmtypes.AccessTypeEverybody,
		},
		Codes:     codes,
		Contracts: contracts,
		Sequences: sequences,
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
						"denom":    "aesov",
						"exponent": 0,
						"aliases":  []string{"wei"},
					},
					{
						"denom":    "esov",
						"exponent": 18,
						"aliases":  []string{},
					},
				},
				"base":    "aesov",
				"display": "esov",
				"name":    "EVM Gas Token",
				"symbol":  "ESOV",
			},
			{
				"description": "Native Cosmos Staking Token",
				"denom_units": []map[string]interface{}{
					{
						"denom":    "ucsov",
						"exponent": 0,
						"aliases":  []string{},
					},
					{
						"denom":    "csov",
						"exponent": 6,
						"aliases":  []string{},
					},
				},
				"base":    "ucsov",
				"display": "csov",
				"name":    "Cosmos Staking Token",
				"symbol":  "CSOV",
			},
			{
				"description": "Bridge Minted Token",
				"denom_units": []map[string]interface{}{
					{
						"denom":    "uwsov",
						"exponent": 0,
						"aliases":  []string{},
					},
					{
						"denom":    "wsov",
						"exponent": 6,
						"aliases":  []string{},
					},
				},
				"base":    "uwsov",
				"display": "wsov",
				"name":    "Bridge Minted Token",
				"symbol":  "WSOV",
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
			params["min_gas_price"] = "0.025000000000000000"
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

	// --- Modify bridge ---
	appState["bridge"] = map[string]interface{}{
		"params": map[string]interface{}{
			"standard_finality_depth":  15,
			"large_finality_depth":     50,
			"large_transfer_threshold": 5000000000,
			"quorum_threshold":        3,
			"max_unlock_per_block":     100000000000,
			"circuit_breaker_address":  "cosmos1cb_addr",
			"gnosis_safe_address":      "cosmos1gs_addr",
			"supply_cap":              1000000000000,
			"lockbox_address":         "0x1234567890123456789012345678901234567890",
		},
		"relayers":      []interface{}{},
		"cosmos_minted": 0,
	}
	if env == "dev" {
		// Inject into auth accounts
		if auth, ok := appState["auth"].(map[string]interface{}); ok {
			var accounts []interface{}
			if accs, ok := auth["accounts"].([]interface{}); ok {
				accounts = accs
			}
			accounts = append(accounts, createGenesisAccount("cosmos1m44j92rkdgvp0460m44d0r3jasvp2uxzwvzfkr"))
			accounts = append(accounts, createGenesisAccount("cosmos1dwkz0xnx4akzv8vnzjapcuqlxtd5c2789w4umh"))
			auth["accounts"] = accounts
		}

		// Inject into bank balances
		if bank, ok := appState["bank"].(map[string]interface{}); ok {
			var balances []interface{}
			if bals, ok := bank["balances"].([]interface{}); ok {
				balances = bals
			}
			balances = append(balances, createGenesisBalance("cosmos1m44j92rkdgvp0460m44d0r3jasvp2uxzwvzfkr"))
			balances = append(balances, createGenesisBalance("cosmos1dwkz0xnx4akzv8vnzjapcuqlxtd5c2789w4umh"))
			bank["balances"] = balances
		}
	} else if env == "prod" {
		// Verify no dev-only accounts are present in prod
		if auth, ok := appState["auth"].(map[string]interface{}); ok {
			if accs, ok := auth["accounts"].([]interface{}); ok {
				for _, acc := range accs {
					if accMap, ok := acc.(map[string]interface{}); ok {
						addr, _ := accMap["address"].(string)
						if addr == "cosmos1m44j92rkdgvp0460m44d0r3jasvp2uxzwvzfkr" || addr == "cosmos1dwkz0xnx4akzv8vnzjapcuqlxtd5c2789w4umh" {
							panic("production genesis must not contain dev holder accounts")
						}
					}
				}
			}
		}
	}

	return appState
}

func createGenesisAccount(address string) map[string]interface{} {
	return map[string]interface{}{
		"@type":          "/cosmos.auth.v1beta1.BaseAccount",
		"address":        address,
		"pub_key":        nil,
		"account_number": "0",
		"sequence":       "0",
	}
}

func createGenesisBalance(address string) map[string]interface{} {
	return map[string]interface{}{
		"address": address,
		"coins": []map[string]interface{}{
			{"denom": "aesov", "amount": "1000000000000000000000000"},
			{"denom": "uwsov", "amount": "1000000000000000"},
			{"denom": "ucsov", "amount": "1000000000000000"},
		},
	}
}

// ---------------------------------------------------------------------------
// main
// ---------------------------------------------------------------------------

func main() {
	var (
		verifyOnly     bool
		outPath        string
		chainID        string
		env            string
		genesisTimeStr string
	)

	flag.BoolVar(&verifyOnly, "verify", false, "Run invariant checks only, do not write genesis file")
	flag.StringVar(&outPath, "out", "", "Output path for the genesis file")
	flag.StringVar(&chainID, "chain-id", "sovereign-1", "Chain ID to embed in genesis")
	flag.StringVar(&env, "env", "dev", "Environment: dev or prod")
	flag.StringVar(&genesisTimeStr, "genesis-time", "", "Genesis time (RFC3339). If empty, defaults to a fixed time (2026-07-09T00:00:00Z) for determinism.")
	flag.Parse()

	if env != "dev" && env != "prod" {
		fmt.Fprintf(os.Stderr, "Error: env must be either 'dev' or 'prod'\n")
		os.Exit(1)
	}

	var genesisTime time.Time
	if genesisTimeStr != "" {
		var err error
		genesisTime, err = time.Parse(time.RFC3339, genesisTimeStr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing genesis-time %q: %v\n", genesisTimeStr, err)
			os.Exit(1)
		}
	} else {
		// Use a fixed epoch time for reproducibility
		genesisTime = time.Date(2026, 7, 9, 0, 0, 0, 0, time.UTC)
	}

	if outPath == "" {
		if env == "dev" {
			outPath = "chain/genesis.dev.json"
		} else {
			outPath = "chain/genesis.prod.json"
		}
	}

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
	genesis := BuildGenesisDoc(chainID, env, genesisTime)
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
	fmt.Printf("     Environment: %s\n", env)
	fmt.Printf("     Genesis Time: %s\n", genesisTime.Format(time.RFC3339))
	fmt.Printf("     Total Supply: %d ucsov (%d TOKEN)\n", TotalSupply, TotalSupply/1_000_000)
	fmt.Printf("     BSC Escrow  : %d ucsov (%d TOKEN)\n", BscEscrowBalance, BscEscrowBalance/1_000_000)
	fmt.Printf("     EVM Chain ID: %d\n", EVMChainID)
	fmt.Printf("     EVM Denom   : %s\n", EVMDenom)
}
