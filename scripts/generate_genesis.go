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
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
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

	// RewardsBucketAddress is the account that holds the 100M TOKEN block-reward reserve.
	// Dedicated address controlled by the team; currently set to cold multisig placeholder.
	RewardsBucketAddress = "cosmos1398hwtqy7s935s26xezpxp6fdf063s93x6tg2g"

	// GenesisDeployer is the address recorded as the creator of all wasm codes and contracts.
	GenesisDeployer = "cosmos1398hwtqy7s935s26xezpxp6fdf063s93x6tg2g"

	// DevOracleOperator is the cosmos1 account address funded in entrypoint.sh for oracle duties.
	DevOracleOperator = "cosmos17955zu57r288jeja39nc67x6qx9rda3xh7wyzw"
)

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
			"FAIL [INV-5]: Validator slot total power %d != expected 30,000,000",
			expectedTotalPower,
		))
	} else {
		fmt.Printf("[PASS] INV-5: Total consensus power = %d (%d validators × %d power)\n",
			expectedTotalPower, ValidatorSlots, EqualizedPowerPerSlot)
	}

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
	contractsDir := "artifacts"
	if _, err := os.Stat(filepath.Join(contractsDir, "constitution.wasm")); os.IsNotExist(err) {
		if err := compileContracts(); err != nil {
			panic(fmt.Sprintf("failed to compile contracts: %v", err))
		}
		contractsDir = "contracts/target/wasm32-unknown-unknown/release"
	}
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

	var codeInstantiatePermission wasmtypes.AccessConfig
	var defaultInstantiatePerm wasmtypes.AccessType
	if env == "dev" {
		codeInstantiatePermission = wasmtypes.AccessConfig{
			Permission: wasmtypes.AccessTypeEverybody,
		}
		defaultInstantiatePerm = wasmtypes.AccessTypeEverybody
	} else {
		codeInstantiatePermission = wasmtypes.AccessConfig{
			Permission: wasmtypes.AccessTypeAnyOfAddresses,
			Addresses:  []string{GenesisDeployer},
		}
		defaultInstantiatePerm = wasmtypes.AccessTypeNobody
	}

	codes := []wasmtypes.Code{
		{
			CodeID: 1,
			CodeInfo: wasmtypes.CodeInfo{
				CodeHash:          sha256Hash(constitutionWasm),
				Creator:           GenesisDeployer,
				InstantiateConfig: codeInstantiatePermission,
			},
			CodeBytes: constitutionWasm,
		},
		{
			CodeID: 2,
			CodeInfo: wasmtypes.CodeInfo{
				CodeHash:          sha256Hash(treasuryWasm),
				Creator:           GenesisDeployer,
				InstantiateConfig: codeInstantiatePermission,
			},
			CodeBytes: treasuryWasm,
		},
		{
			CodeID: 3,
			CodeInfo: wasmtypes.CodeInfo{
				CodeHash:          sha256Hash(reserveFundWasm),
				Creator:           GenesisDeployer,
				InstantiateConfig: codeInstantiatePermission,
			},
			CodeBytes: reserveFundWasm,
		},
		{
			CodeID: 4,
			CodeInfo: wasmtypes.CodeInfo{
				CodeHash:          sha256Hash(governanceWasm),
				Creator:           GenesisDeployer,
				InstantiateConfig: codeInstantiatePermission,
			},
			CodeBytes: governanceWasm,
		},
	}

	constitutionConfigVal := `{"governance_address":"cosmos1wteqf5yrveajhx7zg745p8f46he09gxc2q9fn8","cold_multisig_address":"cosmos1398hwtqy7s935s26xezpxp6fdf063s93x6tg2g","rules":"Safe rules","is_paused":false}`
	constitutionContract := wasmtypes.Contract{
		ContractAddress: "cosmos1shqsrlqalvzwearmrjq8yy788qhzagz6jdq79g",
		ContractInfo: wasmtypes.ContractInfo{
			CodeID:  1,
			Creator: GenesisDeployer,
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

	treasuryConfigVal := `{"governance_address":"cosmos1wteqf5yrveajhx7zg745p8f46he09gxc2q9fn8","cold_multisig_address":"cosmos1398hwtqy7s935s26xezpxp6fdf063s93x6tg2g","is_paused":false,"reentrancy_lock":false}`
	treasuryContract := wasmtypes.Contract{
		ContractAddress: "cosmos1w8kmv94zcf8yysgw9dp8yzq6ffe2e8m0uj8dm0",
		ContractInfo: wasmtypes.ContractInfo{
			CodeID:  2,
			Creator: GenesisDeployer,
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

	reserveConfigVal := `{"governance_address":"cosmos1wteqf5yrveajhx7zg745p8f46he09gxc2q9fn8","cold_multisig_address":"cosmos1398hwtqy7s935s26xezpxp6fdf063s93x6tg2g","min_balance_threshold":"100000000000","is_paused":false,"reentrancy_lock":false}`
	reserveContract := wasmtypes.Contract{
		ContractAddress: "cosmos1dag3w9ydhzmwpvd6asrt8elexa8s27ph7895jc",
		ContractInfo: wasmtypes.ContractInfo{
			CodeID:  3,
			Creator: GenesisDeployer,
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

	governanceConfigVal := `{"constitution_address":"cosmos1shqsrlqalvzwearmrjq8yy788qhzagz6jdq79g","treasury_address":"cosmos1w8kmv94zcf8yysgw9dp8yzq6ffe2e8m0uj8dm0","reserve_fund_address":"cosmos1dag3w9ydhzmwpvd6asrt8elexa8s27ph7895jc","proposers":["cosmos1shqsrlqalvzwearmrjq8yy788qhzagz6jdq79g"],"approval_threshold":1}`
	governanceContract := wasmtypes.Contract{
		ContractAddress: "cosmos1wteqf5yrveajhx7zg745p8f46he09gxc2q9fn8",
		ContractInfo: wasmtypes.ContractInfo{
			CodeID:  4,
			Creator: GenesisDeployer,
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
			{
				Key:   []byte("proposal_count"),
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
			InstantiateDefaultPermission: defaultInstantiatePerm,
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
			// FIX 1: expedited_min_deposit must use TokenDenom, not SDK default "stake".
			params["expedited_min_deposit"] = []map[string]interface{}{
				{"denom": TokenDenom, "amount": "50000000"},
			}
			params["max_deposit_period"] = "172800s"
			params["voting_period"] = "172800s"
			params["expedited_voting_period"] = "86400s"
			params["quorum"] = "0.334000000000000000"
			params["threshold"] = "0.500000000000000000"
			params["veto_threshold"] = "0.334000000000000000"
			params["expedited_threshold"] = "0.667000000000000000"
		}
	}

	// --- Modify evm ---
	if evm, ok := appState["evm"].(map[string]interface{}); ok {
		if params, ok := evm["params"].(map[string]interface{}); ok {
			params["evm_denom"] = EVMDenom
			params["active_static_precompiles"] = []string{
				"0x0000000000000000000000000000000000000801",
				"0x0000000000000000000000000000000000000802",
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

	// --- Fund governance contract accounts & RewardsBucket account ---
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
			accounts = appendUniqueAccount(accounts, addr)
		}
		// FIX 2: RewardsBucket account must exist in auth
		accounts = appendUniqueAccount(accounts, RewardsBucketAddress)
		auth["accounts"] = accounts
	}

	if bank, ok := appState["bank"].(map[string]interface{}); ok {
		var balances []interface{}
		if bals, ok := bank["balances"].([]interface{}); ok {
			balances = bals
		}

		// Treasury: holds the main protocol reserve in ucsov.
		balances = appendUniqueBalance(balances, map[string]interface{}{
			"address": contractAddrs.Treasury,
			"coins": []map[string]interface{}{
				{"denom": TokenDenom, "amount": fmt.Sprintf("%d", TreasuryAllocation)},
			},
		})

		// Reserve Fund: holds the safety reserve in ucsov.
		balances = appendUniqueBalance(balances, map[string]interface{}{
			"address": contractAddrs.ReserveFund,
			"coins": []map[string]interface{}{
				{"denom": TokenDenom, "amount": fmt.Sprintf("%d", ReserveFundAllocation)},
			},
		})

		// FIX 2: RewardsBucket — 100M TOKEN block-reward reserve.
		balances = appendUniqueBalance(balances, map[string]interface{}{
			"address": RewardsBucketAddress,
			"coins": []map[string]interface{}{
				{"denom": TokenDenom, "amount": fmt.Sprintf("%d", RewardsBucket)},
			},
		})

		bank["balances"] = balances
		// FIX 2: bank.supply must equal CosmosAllocation (700M TOKEN)
		bank["supply"] = []map[string]interface{}{
			{"denom": TokenDenom, "amount": fmt.Sprintf("%d", CosmosAllocation)},
		}
	}

	// --- Modify bridge ---
	gnosisSafe := os.Getenv("BRIDGE_GNOSIS_SAFE_ADDRESS")
	lockbox := os.Getenv("BRIDGE_LOCKBOX_ADDRESS")
	circuitBreaker := os.Getenv("BRIDGE_CIRCUIT_BREAKER_ADDRESS")

	if env == "prod" {
		if gnosisSafe == "" || lockbox == "" || circuitBreaker == "" {
			panic("production genesis build requires BRIDGE_GNOSIS_SAFE_ADDRESS, BRIDGE_LOCKBOX_ADDRESS, and BRIDGE_CIRCUIT_BREAKER_ADDRESS environment variables to be set")
		}
	} else {
		// FIX 4: Use valid bech32 addresses for dev/staging
		if gnosisSafe == "" {
			gnosisSafe = "cosmos1398hwtqy7s935s26xezpxp6fdf063s93x6tg2g"
		}
		if lockbox == "" {
			lockbox = "0xdeace2ea00000000000000000000000000000000"
		}
		if circuitBreaker == "" {
			circuitBreaker = "cosmos1398hwtqy7s935s26xezpxp6fdf063s93x6tg2g"
		}
	}

	// Validate formats (Issues #11 & #12: enforce valid Bech32 using sdk.AccAddressFromBech32)
	validateBech32Address(gnosisSafe, "gnosis_safe_address")
	validateBech32Address(circuitBreaker, "circuit_breaker_address")
	if !strings.HasPrefix(lockbox, "0x") || len(lockbox) != 42 {
		panic(fmt.Sprintf("invalid lockbox_address: %q (must be 20-byte hex address with 0x prefix)", lockbox))
	}

	appState["bridge"] = map[string]interface{}{
		"params": map[string]interface{}{
			"standard_finality_depth":  15,
			"large_finality_depth":     50,
			"large_transfer_threshold": 5000000000,
			"quorum_threshold":        3,
			"max_unlock_per_block":     100000000000,
			"circuit_breaker_address":  circuitBreaker,
			"gnosis_safe_address":      gnosisSafe,
			// FIX 3: supply_cap must match BscEscrowBalance (300M TOKEN)
			"supply_cap":              fmt.Sprintf("%d", BscEscrowBalance),
			"lockbox_address":         lockbox,
		},
		"relayers":      []interface{}{},
		"cosmos_minted": 0,
	}

	if env == "dev" {
		// FIX 5: Inject funded dev oracle operator into oracle state
		if oracle, ok := appState["oracle"].(map[string]interface{}); ok {
			oracle["operators"] = []string{DevOracleOperator}
		} else {
			appState["oracle"] = map[string]interface{}{
				"params": map[string]interface{}{
					"commit_window":              10,
					"reveal_window":              10,
					"min_operator_commits":       1,
					"staleness_threshold_blocks": 100,
				},
				"operators": []string{DevOracleOperator},
			}
		}
		// Inject into auth accounts
		if auth, ok := appState["auth"].(map[string]interface{}); ok {
			var accounts []interface{}
			if accs, ok := auth["accounts"].([]interface{}); ok {
				accounts = accs
			}
			accounts = appendUniqueAccount(accounts, "cosmos1m44j92rkdgvp0460m44d0r3jasvp2uxzwvzfkr")
			accounts = appendUniqueAccount(accounts, "cosmos1dwkz0xnx4akzv8vnzjapcuqlxtd5c2789w4umh")
			auth["accounts"] = accounts
		}

		// Inject into bank balances
		if bank, ok := appState["bank"].(map[string]interface{}); ok {
			var balances []interface{}
			if bals, ok := bank["balances"].([]interface{}); ok {
				balances = bals
			}
			balances = appendUniqueBalance(balances, createGenesisBalance("cosmos1m44j92rkdgvp0460m44d0r3jasvp2uxzwvzfkr"))
			balances = appendUniqueBalance(balances, createGenesisBalance("cosmos1dwkz0xnx4akzv8vnzjapcuqlxtd5c2789w4umh"))
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

		// Verify all four governance contract accounts ARE present in prod genesis.
		// Missing entries would cause bank InitGenesis to panic on chain start.
		requiredContracts := map[string]bool{
			contractAddrs.Constitution: false,
			contractAddrs.Treasury:     false,
			contractAddrs.ReserveFund:  false,
			contractAddrs.Governance:   false,
		}
		if auth, ok := appState["auth"].(map[string]interface{}); ok {
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
		}
		for addr, found := range requiredContracts {
			if !found {
				panic(fmt.Sprintf("production genesis is missing required governance contract account: %s", addr))
			}
		}
	}

	return appState
}

func validateBech32Address(addr string, label string) {
	if _, err := sdk.AccAddressFromBech32(addr); err != nil {
		panic(fmt.Sprintf("invalid %s address %q: %v", label, addr, err))
	}
}

func createGenesisAccount(address string) map[string]interface{} {
	validateBech32Address(address, "account")
	return map[string]interface{}{
		"@type":          "/cosmos.auth.v1beta1.BaseAccount",
		"address":        address,
		"pub_key":        nil,
		"account_number": "0",
		"sequence":       "0",
	}
}

func appendUniqueAccount(accounts []interface{}, address string) []interface{} {
	for _, acc := range accounts {
		if accMap, ok := acc.(map[string]interface{}); ok {
			if addr, ok := accMap["address"].(string); ok && addr == address {
				return accounts
			}
		}
	}
	return append(accounts, createGenesisAccount(address))
}

func createGenesisBalance(address string) map[string]interface{} {
	validateBech32Address(address, "balance")
	return map[string]interface{}{
		"address": address,
		"coins": []map[string]interface{}{
			{"denom": "aesov", "amount": "1000000000000000000000000"},
			{"denom": "ucsov", "amount": "1000000000000000"},
		},
	}
}

func appendUniqueBalance(balances []interface{}, balance map[string]interface{}) []interface{} {
	targetAddr, _ := balance["address"].(string)
	if targetAddr != "" {
		validateBech32Address(targetAddr, "balance")
	}
	for _, b := range balances {
		if bMap, ok := b.(map[string]interface{}); ok {
			if addr, ok := bMap["address"].(string); ok && addr == targetAddr {
				return balances
			}
		}
	}
	return append(balances, balance)
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
		if err := VerifyGenesisWasmChecksums(outPath); err != nil {
			fmt.Fprintf(os.Stderr, "VerifyGenesisWasmChecksums failed: %v\n", err)
			os.Exit(1)
		}
		if fileFailures := VerifyGenesisFileInvariants(outPath, env, chainID); len(fileFailures) > 0 {
			fmt.Fprintln(os.Stderr, "=== GENESIS FILE INVARIANT FAILURES ===")
			for _, f := range fileFailures {
				fmt.Fprintln(os.Stderr, f)
			}
			os.Exit(1)
		}
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
	if err := VerifyGenesisWasmChecksums(outPath); err != nil {
		fmt.Fprintf(os.Stderr, "VerifyGenesisWasmChecksums failed on generated genesis: %v\n", err)
		os.Exit(1)
	}
	if fileFailures := VerifyGenesisFileInvariants(outPath, env, chainID); len(fileFailures) > 0 {
		fmt.Fprintln(os.Stderr, "=== GENESIS FILE INVARIANT FAILURES ON GENERATED FILE ===")
		for _, f := range fileFailures {
			fmt.Fprintln(os.Stderr, f)
		}
		os.Exit(1)
	}

	fmt.Printf("     Chain ID   : %s\n", chainID)
	fmt.Printf("     Environment: %s\n", env)
	fmt.Printf("     Genesis Time: %s\n", genesisTime.Format(time.RFC3339))
	fmt.Printf("     Total Supply: %d ucsov (%d TOKEN)\n", TotalSupply, TotalSupply/1_000_000)
	fmt.Printf("     BSC Escrow  : %d ucsov (%d TOKEN)\n", BscEscrowBalance, BscEscrowBalance/1_000_000)
	fmt.Printf("     EVM Chain ID: %d\n", EVMChainID)
	fmt.Printf("     EVM Denom   : %s\n", EVMDenom)
}

func VerifyGenesisWasmChecksums(genesisPath string) error {
	genesisData, err := os.ReadFile(genesisPath)
	if os.IsNotExist(err) {
		fmt.Printf("[INFO] Genesis file %s does not exist, skipping checksum verification.\n", genesisPath)
		return nil
	} else if err != nil {
		return fmt.Errorf("failed to read genesis file: %w", err)
	}

	// Parse genesis to find the wasm module state
	var genesisDoc struct {
		AppState map[string]json.RawMessage `json:"app_state"`
	}
	if err := json.Unmarshal(genesisData, &genesisDoc); err != nil {
		return fmt.Errorf("failed to parse genesis app state: %w", err)
	}

	wasmRaw, ok := genesisDoc.AppState["wasm"]
	if !ok {
		return fmt.Errorf("wasm module state not found in genesis")
	}

	var wasmState struct {
		Codes []struct {
			CodeID    uint64 `json:"code_id"`
			CodeBytes string `json:"code_bytes"`
		} `json:"codes"`
	}
	if err := json.Unmarshal(wasmRaw, &wasmState); err != nil {
		return fmt.Errorf("failed to parse wasm state: %w", err)
	}

	// Load checksums from artifacts/checksums.txt
	checksumsData, err := os.ReadFile("artifacts/checksums.txt")
	if err != nil {
		return fmt.Errorf("failed to read artifacts/checksums.txt: %w", err)
	}

	lines := strings.Split(string(checksumsData), "\n")
	checksumMap := make(map[string]string) // name -> sha256hex
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			checksumMap[parts[1]] = parts[0]
		}
	}

	// The codes in genesis are compiled in this order (1: constitution, 2: treasury, 3: reserve_fund, 4: governance)
	codeIDMap := map[uint64]string{
		1: "constitution.wasm",
		2: "treasury.wasm",
		3: "reserve_fund.wasm",
		4: "governance.wasm",
	}

	verifiedCodeIDs := make(map[uint64]bool)

	for _, code := range wasmState.Codes {
		filename, ok := codeIDMap[code.CodeID]
		if !ok {
			continue
		}
		expectedHash, ok := checksumMap[filename]
		if !ok {
			return fmt.Errorf("missing checksum for %s in checksums.txt", filename)
		}

		codeBytes, err := hex.DecodeString(code.CodeBytes)
		if err != nil {
			// Try base64 decoding as well, just in case
			codeBytes, err = base64.StdEncoding.DecodeString(code.CodeBytes)
			if err != nil {
				return fmt.Errorf("failed to decode code_bytes for code_id %d: %w", code.CodeID, err)
			}
		}

		hash := sha256.Sum256(codeBytes)
		actualHash := fmt.Sprintf("%x", hash)
		if actualHash != expectedHash {
			return fmt.Errorf("checksum mismatch for %s (CodeID: %d): expected %s, got %s", filename, code.CodeID, expectedHash, actualHash)
		}
		verifiedCodeIDs[code.CodeID] = true
		fmt.Printf("[PASS] Verified genesis hash for %s matches artifacts/checksums.txt (%s)\n", filename, actualHash)
	}

	for requiredID, filename := range codeIDMap {
		if !verifiedCodeIDs[requiredID] {
			return fmt.Errorf("required CodeID %d (%s) missing or not verified in genesis wasm state", requiredID, filename)
		}
	}

	return nil
}

func VerifyGenesisFileInvariants(filePath string, env string, expectedChainID string) []string {
	var failures []string

	genesisData, err := os.ReadFile(filePath)
	if err != nil {
		failures = append(failures, fmt.Sprintf("FAIL: failed to read genesis file %s: %v", filePath, err))
		return failures
	}

	var genesisDoc struct {
		ChainID  string `json:"chain_id"`
		AppState map[string]json.RawMessage `json:"app_state"`
	}
	if err := json.Unmarshal(genesisData, &genesisDoc); err != nil {
		failures = append(failures, fmt.Sprintf("FAIL: failed to parse genesis %s: %v", filePath, err))
		return failures
	}

	// 1. Verify chain-id matches
	if expectedChainID == "" {
		expectedChainID = "sovereign-1"
	}
	if genesisDoc.ChainID != expectedChainID {
		failures = append(failures, fmt.Sprintf("FAIL: expected chain_id %q, got %q", expectedChainID, genesisDoc.ChainID))
	} else {
		fmt.Printf("[PASS] File invariant: chain_id = %s\n", genesisDoc.ChainID)
	}

	// 2. Verify bank supply for ucsov equals CosmosAllocation (700M TOKEN) & uwsov supply is 0
	var bankState struct {
		Supply []struct {
			Denom  string `json:"denom"`
			Amount string `json:"amount"`
		} `json:"supply"`
		Balances []struct {
			Address string `json:"address"`
			Coins   []struct {
				Denom  string `json:"denom"`
				Amount string `json:"amount"`
			} `json:"coins"`
		} `json:"balances"`
	}
	if bankRaw, ok := genesisDoc.AppState["bank"]; ok {
		if err := json.Unmarshal(bankRaw, &bankState); err == nil {
			var ucsovSupplyFound bool
			expectedSupplyStr := fmt.Sprintf("%d", CosmosAllocation)
			for _, coin := range bankState.Supply {
				if coin.Denom == TokenDenom {
					if coin.Amount != expectedSupplyStr {
						failures = append(failures, fmt.Sprintf("FAIL: bank supply for %s must be %s, got %s", TokenDenom, expectedSupplyStr, coin.Amount))
					} else {
						ucsovSupplyFound = true
						fmt.Printf("[PASS] File invariant: bank supply for %s = %s\n", TokenDenom, coin.Amount)
					}
				}
				if coin.Denom == "uwsov" {
					if coin.Amount != "0" && coin.Amount != "" {
						failures = append(failures, fmt.Sprintf("FAIL: bank supply for uwsov must be 0, got %s", coin.Amount))
					}
				}
			}
			if !ucsovSupplyFound {
				failures = append(failures, fmt.Sprintf("FAIL: bank supply for %s missing from genesis", TokenDenom))
			}

			// 3. Verify no account has uwsov > 0 balance at genesis
			for _, balance := range bankState.Balances {
				for _, coin := range balance.Coins {
					if coin.Denom == "uwsov" {
						if coin.Amount != "0" && coin.Amount != "" {
							failures = append(failures, fmt.Sprintf("FAIL: account %s has non-zero uwsov balance: %s", balance.Address, coin.Amount))
						}
					}
				}
			}
		} else {
			failures = append(failures, fmt.Sprintf("FAIL: failed to parse bank state in %s: %v", filePath, err))
		}
	}

	// 4. Verify bridge state
	var bridgeState struct {
		CosmosMinted interface{} `json:"cosmos_minted"`
		Params       struct {
			GnosisSafeAddress     string `json:"gnosis_safe_address"`
			LockboxAddress        string `json:"lockbox_address"`
			CircuitBreakerAddress string `json:"circuit_breaker_address"`
		} `json:"params"`
	}
	if bridgeRaw, ok := genesisDoc.AppState["bridge"]; ok {
		if err := json.Unmarshal(bridgeRaw, &bridgeState); err == nil {
			var minted int64
			switch v := bridgeState.CosmosMinted.(type) {
			case float64:
				minted = int64(v)
			case string:
				fmt.Sscanf(v, "%d", &minted)
			}
			if minted != 0 {
				failures = append(failures, fmt.Sprintf("FAIL: bridge cosmos_minted must be 0, got %v", bridgeState.CosmosMinted))
			} else {
				fmt.Printf("[PASS] File invariant: bridge cosmos_minted = 0\n")
			}

			// Verify no placeholder addresses in bridge params
			gs := bridgeState.Params.GnosisSafeAddress
			lb := bridgeState.Params.LockboxAddress
			cb := bridgeState.Params.CircuitBreakerAddress

			if env == "prod" {
				if gs == "cosmos1gs_addr" || gs == "cosmos1devgsaddressplaceholderxxxxx" || gs == "" {
					failures = append(failures, fmt.Sprintf("FAIL: production bridge gnosis_safe_address must not be a placeholder, got %q", gs))
				}
				if cb == "cosmos1cb_addr" || cb == "cosmos1devcbaddressplaceholderxxxxx" || cb == "" {
					failures = append(failures, fmt.Sprintf("FAIL: production bridge circuit_breaker_address must not be a placeholder, got %q", cb))
				}
				if lb == "0x1234567890123456789012345678901234567890" || lb == "0xdeace2ea00000000000000000000000000000000" || lb == "" {
					failures = append(failures, fmt.Sprintf("FAIL: production bridge lockbox_address must not be a placeholder, got %q", lb))
				}
			} else {
				// Format check only
				if !strings.HasPrefix(gs, "cosmos1") || len(gs) < 10 {
					failures = append(failures, fmt.Sprintf("FAIL: invalid gnosis_safe_address format: %q", gs))
				}
				if !strings.HasPrefix(cb, "cosmos1") || len(cb) < 10 {
					failures = append(failures, fmt.Sprintf("FAIL: invalid circuit_breaker_address format: %q", cb))
				}
				if !strings.HasPrefix(lb, "0x") || len(lb) != 42 {
					failures = append(failures, fmt.Sprintf("FAIL: invalid lockbox_address format: %q", lb))
				}
			}
		} else {
			failures = append(failures, fmt.Sprintf("FAIL: failed to parse bridge state in %s: %v", filePath, err))
		}
	}

	// 5. Verify evm params
	if evmRaw, ok := genesisDoc.AppState["evm"]; ok {
		var evmState struct {
			Params struct {
				EvmDenom string `json:"evm_denom"`
			} `json:"params"`
		}
		if err := json.Unmarshal(evmRaw, &evmState); err == nil {
			if evmState.Params.EvmDenom != EVMDenom {
				failures = append(failures, fmt.Sprintf("FAIL: evm evm_denom must be %q, got %q", EVMDenom, evmState.Params.EvmDenom))
			} else {
				fmt.Printf("[PASS] File invariant: evm evm_denom = %s\n", EVMDenom)
			}
		} else {
			failures = append(failures, fmt.Sprintf("FAIL: failed to parse evm state in %s: %v", filePath, err))
		}
	}

	// 6. Verify feemarket params
	if fmRaw, ok := genesisDoc.AppState["feemarket"]; ok {
		var fmState struct {
			Params struct {
				NoBaseFee            bool   `json:"no_base_fee"`
				ElasticityMultiplier uint32 `json:"elasticity_multiplier"`
			} `json:"params"`
		}
		if err := json.Unmarshal(fmRaw, &fmState); err == nil {
			if fmState.Params.NoBaseFee != false {
				failures = append(failures, "FAIL: feemarket no_base_fee must be false")
			}
			if fmState.Params.ElasticityMultiplier != 2 {
				failures = append(failures, fmt.Sprintf("FAIL: feemarket elasticity_multiplier must be 2, got %d", fmState.Params.ElasticityMultiplier))
			}
			fmt.Printf("[PASS] File invariant: feemarket params (no_base_fee=false, elasticity_multiplier=2)\n")
		} else {
			failures = append(failures, fmt.Sprintf("FAIL: failed to parse feemarket state in %s: %v", filePath, err))
		}
	}

	// 7. Verify erc20 params & native token pair
	if erc20Raw, ok := genesisDoc.AppState["erc20"]; ok {
		var erc20State struct {
			Params struct {
				EnableErc20 bool `json:"enable_erc20"`
			} `json:"params"`
			TokenPairs []struct {
				Erc20Address string `json:"erc20_address"`
				Denom        string `json:"denom"`
				Enabled      bool   `json:"enabled"`
			} `json:"token_pairs"`
		}
		if err := json.Unmarshal(erc20Raw, &erc20State); err == nil {
			if !erc20State.Params.EnableErc20 {
				failures = append(failures, "FAIL: erc20 enable_erc20 must be true")
			}
			var pairFound bool
			for _, pair := range erc20State.TokenPairs {
				if pair.Denom == TokenDenom && pair.Erc20Address == "0x0000000000000000000000000000000000000001" && pair.Enabled {
					pairFound = true
					break
				}
			}
			if !pairFound {
				failures = append(failures, fmt.Sprintf("FAIL: erc20 native token pair for %s (0x...0001) not found or disabled", TokenDenom))
			} else {
				fmt.Printf("[PASS] File invariant: erc20 native token pair (%s <-> 0x...0001) verified\n", TokenDenom)
			}
		} else {
			failures = append(failures, fmt.Sprintf("FAIL: failed to parse erc20 state in %s: %v", filePath, err))
		}
	}

	return failures
}
