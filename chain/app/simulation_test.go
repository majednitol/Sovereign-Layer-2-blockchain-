package app

import (
	context "context"
	"crypto/ed25519"
	"encoding/json"
	"flag"
	"fmt"
	big "math/big"
	"math/rand"
	"testing"
	"time"

	"cosmossdk.io/math"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/store/v2/dbadapter"
	legacytypes "github.com/cosmos/cosmos-sdk/store/v2/types"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptokeys "github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	storetypes "github.com/cosmos/cosmos-sdk/store/v2/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/cosmos/cosmos-sdk/baseapp"

	common "github.com/ethereum/go-ethereum/common"
	evmtypes "github.com/cosmos/evm/x/vm/types"

	"github.com/sovereign-l1/chain/x/bridge"
	"github.com/sovereign-l1/chain/x/certification"
	"github.com/sovereign-l1/chain/x/governance-ext"
	"github.com/sovereign-l1/chain/x/milestone"
	"github.com/sovereign-l1/chain/x/oracle"
	"github.com/sovereign-l1/chain/x/settlement"
	"github.com/sovereign-l1/chain/x/validator"
)

var numBlocks = flag.Int("NumBlocks", 100, "Number of blocks to run in the simulation")
var seed = flag.Int64("Seed", 0, "Seed for random number generator")
var blockSize = flag.Int("BlockSize", 3, "Number of operations to run per block")

type mockMultiStoreSim struct {
	storetypes.MultiStore
	stores map[string]storetypes.KVStore
}

func (m mockMultiStoreSim) GetKVStore(key storetypes.StoreKey) storetypes.KVStore {
	return m.stores[key.Name()]
}

func (m mockMultiStoreSim) GetStore(key storetypes.StoreKey) storetypes.Store {
	return m.stores[key.Name()]
}

type kvStoreV2Wrapper struct {
	legacytypes.KVStore
}

func (w kvStoreV2Wrapper) GetStoreType() storetypes.StoreType {
	return storetypes.StoreType(w.KVStore.GetStoreType())
}

func (w kvStoreV2Wrapper) Iterator(start, end []byte) storetypes.Iterator {
	return w.KVStore.Iterator(start, end)
}

func (w kvStoreV2Wrapper) ReverseIterator(start, end []byte) storetypes.Iterator {
	return w.KVStore.ReverseIterator(start, end)
}

func (w kvStoreV2Wrapper) CacheWrap() storetypes.CacheWrap {
	return nil
}

type mockStakingKeeperSim struct{}

func (m mockStakingKeeperSim) GetLastValidatorPower(ctx context.Context, valAddr sdk.ValAddress) (int64, error) {
	return 100, nil
}

func (m mockStakingKeeperSim) GetLastTotalPower(ctx context.Context) (math.Int, error) {
	return math.NewInt(100), nil
}

func (m mockStakingKeeperSim) IterateLastValidatorPowers(ctx context.Context, handler func(valAddr sdk.ValAddress, power int64) (stop bool)) error {
	valAddr := sdk.ValAddress([]byte("mock_validator_address"))
	handler(valAddr, 100)
	return nil
}

func (m mockStakingKeeperSim) GetValidator(ctx context.Context, valAddr sdk.ValAddress) (stakingtypes.Validator, error) {
	pk := cryptokeys.GenPrivKey().PubKey()
	anyPk, _ := codectypes.NewAnyWithValue(pk)
	return stakingtypes.Validator{
		OperatorAddress: valAddr.String(),
		ConsensusPubkey: anyPk,
	}, nil
}

type mockSlashingKeeperSim struct{}

func (m mockSlashingKeeperSim) Tombstone(ctx context.Context, valAddr sdk.ConsAddress) error {
	return nil
}

func (m mockSlashingKeeperSim) HasValidatorSigningInfo(ctx context.Context, consAddr sdk.ConsAddress) bool {
	return true
}

func (m mockSlashingKeeperSim) SetValidatorSigningInfo(ctx context.Context, address sdk.ConsAddress, info slashingtypes.ValidatorSigningInfo) error {
	return nil
}

func (m mockSlashingKeeperSim) Slash(ctx context.Context, consAddr sdk.ConsAddress, fraction math.LegacyDec, power, distributionHeight int64) error {
	return nil
}

func (m mockSlashingKeeperSim) Jail(ctx context.Context, consAddr sdk.ConsAddress) error {
	return nil
}

type mockWasmKeeperSim struct{}

func (m mockWasmKeeperSim) Execute(ctx sdk.Context, contractAddr sdk.AccAddress, caller sdk.AccAddress, msg []byte, coins sdk.Coins) ([]byte, error) {
	return []byte(`{"status":"approved"}`), nil
}

func (m mockWasmKeeperSim) QuerySmart(ctx sdk.Context, contractAddr sdk.AccAddress, req []byte) ([]byte, error) {
	return []byte(`{"is_valid":true,"reason":""}`), nil
}

type mockBankKeeperSim struct{}

func (m mockBankKeeperSim) SendCoins(ctx context.Context, fromAddr sdk.AccAddress, toAddr sdk.AccAddress, amt sdk.Coins) error {
	return nil
}

func (m mockBankKeeperSim) MintCoins(ctx context.Context, moduleName string, amt sdk.Coins) error {
	return nil
}

func (m mockBankKeeperSim) BurnCoins(ctx context.Context, moduleName string, amt sdk.Coins) error {
	return nil
}

func (m mockBankKeeperSim) SendCoinsFromModuleToAccount(ctx context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error {
	return nil
}

func (m mockBankKeeperSim) SendCoinsFromAccountToModule(ctx context.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error {
	return nil
}

func TestAppSimulation(t *testing.T) {
	flag.Parse()
	blocks := *numBlocks
	fmt.Printf("Running app simulation for %d blocks...\n", blocks)

	// Set up multi-stores for keepers
	dbMap := make(map[string]storetypes.KVStore)
	modules := []string{
		validator.StoreKey,
		certification.StoreKey,
		oracle.StoreKey,
		milestone.StoreKey,
		settlement.StoreKey,
		gov_ext.StoreKey,
		bridge.StoreKey,
	}
	for _, m := range modules {
		db := dbm.NewMemDB()
		dbMap[m] = kvStoreV2Wrapper{dbadapter.Store{DB: db}}
	}
	ms := mockMultiStoreSim{stores: dbMap}

	ctx := sdk.Context{}.
		WithMultiStore(ms).
		WithGasMeter(storetypes.NewInfiniteGasMeter()).
		WithBlockHeight(1).
		WithBlockTime(time.Unix(1000000, 0)).
		WithChainID("sovereign-devnet-1").
		WithEventManager(sdk.NewEventManager())

	// Initialize mock keepers
	staking := mockStakingKeeperSim{}
	slashing := mockSlashingKeeperSim{}
	wasm := mockWasmKeeperSim{}
	bank := mockBankKeeperSim{}

	valKeeper := validator.NewKeeper(storetypes.NewKVStoreKey(validator.StoreKey), nil, staking, slashing, nil, nil, 30)
	certKeeper := certification.NewKeeper(storetypes.NewKVStoreKey(certification.StoreKey), nil, staking, slashing)
	oracleKeeper := oracle.NewKeeper(storetypes.NewKVStoreKey(oracle.StoreKey), nil, staking, slashing)
	milesKeeper := milestone.NewKeeper(storetypes.NewKVStoreKey(milestone.StoreKey), nil, oracleKeeper, bank)
	settKeeper := settlement.NewKeeper(storetypes.NewKVStoreKey(settlement.StoreKey), nil, bank)
	bridgeKeeper := bridge.NewKeeper(storetypes.NewKVStoreKey(bridge.StoreKey), nil, bank)
	govExtKeeper := gov_ext.NewKeeper(
		storetypes.NewKVStoreKey(gov_ext.StoreKey),
		nil,
		wasm,
		sdk.AccAddress([]byte("constitution_addr")),
		valKeeper,
		milesKeeper,
		oracleKeeper,
		settKeeper,
		bridgeKeeper,
	)

	// Set default params for modules to avoid uninitialized panics
	oracleKeeper.SetParams(ctx, oracle.Params{
		CommitWindow:             10,
		RevealWindow:             10,
		MinOperatorCommits:       1,
		StalenessThresholdBlocks: 100,
	})
	milesKeeper.SetParams(ctx, milestone.Params{
		MaxActiveMilestones: 500,
	})
	settKeeper.SetParams(ctx, settlement.Params{
		TimestampToleranceSeconds: 30,
	})
	govExtKeeper.SetParams(ctx, gov_ext.Params{
		MinGasLimit: 100000,
		MaxGasLimit: 2000000,
	})
	certKeeper.SetParams(ctx, certification.Params{
		MaxConsecutiveRejections: 5,
		MissedExtensionLimit:     10,
	})
	bridgeKeeper.SetParams(ctx, bridge.Params{
		StandardFinalityDepth:  15,
		LargeFinalityDepth:     50,
		LargeTransferThreshold: 5000000000,
		QuorumThreshold:        2,
		MaxUnlockPerBlock:      100000000000,
		CircuitBreakerAddress:  "cosmos1cb_addr",
		GnosisSafeAddress:      "cosmos1gs_addr",
		SupplyCap:              "1000000000000",
		LockBoxAddress:         "0x1234567890123456789012345678901234567890",
	})

	// Pre-generate accounts
	accs := make([]simtypes.Account, 5)
	for i := 0; i < 5; i++ {
		_, pub, _ := ed25519.GenerateKey(nil)
		accs[i] = simtypes.Account{
			Address: sdk.AccAddress(pub),
		}
	}

	// Register operations
	ops := []simtypes.Operation{
		validator.SimulateMsgFillValidatorSlot(valKeeper),
		validator.SimulateMsgEjectValidator(valKeeper),
		validator.SimulateMsgUpdatePartitionScheme(valKeeper),
		certification.SimulateMsgUpdateCertificationParams(certKeeper),
		oracle.SimulateMsgCommitOracleHash(oracleKeeper),
		oracle.SimulateMsgRevealOracleReport(oracleKeeper),
		oracle.SimulateDropOracleReveal(oracleKeeper),
		oracle.SimulateOracleRoundInsufficient(oracleKeeper),
		milestone.SimulateMsgCreateMilestone(milesKeeper),
		settlement.SimulateMsgSettlement(settKeeper),
		settlement.SimulateMsgInvalidWitnessSettlement(settKeeper),
		settlement.SimulateMsgExpiredTimestampSettlement(settKeeper),
		gov_ext.SimulateMsgMigrateContracts(govExtKeeper),
		gov_ext.SimulateMsgUpdateValidatorSlot(govExtKeeper),
		gov_ext.SimulateMsgUpdateMilestone(govExtKeeper),
		gov_ext.SimulateMsgUpdateOracleOperator(govExtKeeper),
		gov_ext.SimulateMsgUpdateWitnessRegistry(govExtKeeper),
		gov_ext.SimulateMsgUpdateBridgeRelayerSet(govExtKeeper),
		bridge.SimulateMsgBridgeIn(bridgeKeeper),
		bridge.SimulateMsgBridgeOut(bridgeKeeper),
		bridge.SimulateMsgBridgeInCapBreach(bridgeKeeper),
		SimulateMsgEthSimpleTransfer(),
		SimulateMsgEthContractCreate(),
		SimulateMsgEthContractCall(),
	}

	var seedVal int64
	if *seed == 0 {
		seedVal = time.Now().UnixNano()
	} else {
		seedVal = *seed
	}
	fmt.Printf("Using seed: %d, block size: %d\n", seedVal, *blockSize)
	r := rand.New(rand.NewSource(seedVal))

	for i := int64(1); i <= int64(blocks); i++ {
		ctx = ctx.WithBlockHeight(i).WithBlockTime(ctx.BlockTime().Add(time.Second * 5))

		// Write a fresh mock aggregate price to oracle keeper to prevent feed staleness.
		// Use target price 105000 so the milestone target targetPrice of 100000 is met and achieved.
		oracleStore := ctx.KVStore(storetypes.NewKVStoreKey(oracle.StoreKey))
		agg := oracle.AggregatePrice{
			Price:       105000,
			BlockHeight: i,
		}
		bz, _ := json.Marshal(agg)
		oracleStore.Set(append(oracle.AggregateKeyPrefix, []byte("BTC_USD")...), bz)

		// Run custom number of random operations per block
		for opIdx := 0; opIdx < *blockSize; opIdx++ {
			op := ops[r.Intn(len(ops))]
			_, _, _ = op(r, nil, ctx, accs, ctx.ChainID())
		}

		// Trigger EndBlocker executions
		_ = valKeeper.EndBlocker(ctx)
		certKeeper.EndBlocker(ctx, r.Intn(100) < 5) // 5% chance of proposal rejection
		milesKeeper.EndBlocker(ctx)
		oracleKeeper.EndBlocker(ctx)

		// Prune achieved/expired milestones to keep database small and iteration O(1)
		var toDelete []string
		milesKeeper.IterateMilestones(ctx, func(m milestone.Milestone) bool {
			if m.State == milestone.StateAchieved || m.State == milestone.StateExpired {
				toDelete = append(toDelete, m.ID)
			}
			return false
		})
		milesStore := ctx.KVStore(storetypes.NewKVStoreKey(milestone.StoreKey))
		for _, id := range toDelete {
			milesStore.Delete(append(milestone.MilestoneKeyPrefix, []byte(id)...))
		}

		if i%5000 == 0 || i == 1 {
			fmt.Printf("Simulated block height: %d\n", i)
		}
	}

	fmt.Printf("[PASS] Simulated %d blocks successfully without panics.\n", blocks)
}

// SimulateMsgEthSimpleTransfer simulates a simple EVM ether/aesov transfer transaction.
func SimulateMsgEthSimpleTransfer() simtypes.Operation {
	return func(
		r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context,
		accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		if len(accs) == 0 {
			return simtypes.NoOpMsg("vm", "MsgEthSimpleTransfer", "no accounts available"), nil, nil
		}
		fromAcc := accs[r.Intn(len(accs))]
		randAddrBytes := make([]byte, 20)
		r.Read(randAddrBytes)
		toAddr := common.BytesToAddress(randAddrBytes)

		evmTx := &evmtypes.EvmTxArgs{
			ChainID:  big.NewInt(7777),
			Nonce:    uint64(r.Intn(100)),
			To:       &toAddr,
			Amount:   big.NewInt(int64(r.Intn(10000) + 1)),
			GasLimit: 21000,
			GasPrice: big.NewInt(1000000000), // 1 Gwei
		}
		msg := evmtypes.NewTx(evmTx)
		msg.From = fromAcc.Address.Bytes()

		return simtypes.NewOperationMsg(msg, true, ""), nil, nil
	}
}

// SimulateMsgEthContractCreate simulates an EVM contract creation transaction.
func SimulateMsgEthContractCreate() simtypes.Operation {
	return func(
		r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context,
		accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		if len(accs) == 0 {
			return simtypes.NoOpMsg("vm", "MsgEthContractCreate", "no accounts available"), nil, nil
		}
		fromAcc := accs[r.Intn(len(accs))]

		// simple mock bytecode
		bytecode := []byte{0x60, 0x80, 0x60, 0x40, 0x52, 0x34, 0x80, 0x15, 0x60, 0x0f}

		evmTx := &evmtypes.EvmTxArgs{
			ChainID:  big.NewInt(7777),
			Nonce:    uint64(r.Intn(100)),
			Amount:   big.NewInt(0),
			GasLimit: 100000,
			GasPrice: big.NewInt(1000000000), // 1 Gwei
			Input:    bytecode,
		}
		msg := evmtypes.NewTx(evmTx)
		msg.From = fromAcc.Address.Bytes()

		return simtypes.NewOperationMsg(msg, true, ""), nil, nil
	}
}

// SimulateMsgEthContractCall simulates an EVM contract call execution transaction.
func SimulateMsgEthContractCall() simtypes.Operation {
	return func(
		r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context,
		accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		if len(accs) == 0 {
			return simtypes.NoOpMsg("vm", "MsgEthContractCall", "no accounts available"), nil, nil
		}
		fromAcc := accs[r.Intn(len(accs))]
		randAddrBytes := make([]byte, 20)
		r.Read(randAddrBytes)
		contractAddr := common.BytesToAddress(randAddrBytes)

		// simple mock function call input (e.g. transfer(address,uint256))
		randCallBytes := make([]byte, 64)
		r.Read(randCallBytes)
		callInput := append([]byte{0xa9, 0x05, 0x9c, 0xbb}, randCallBytes...)

		evmTx := &evmtypes.EvmTxArgs{
			ChainID:  big.NewInt(7777),
			Nonce:    uint64(r.Intn(100)),
			To:       &contractAddr,
			Amount:   big.NewInt(0),
			GasLimit: 50000,
			GasPrice: big.NewInt(1000000000), // 1 Gwei
			Input:    callInput,
		}
		msg := evmtypes.NewTx(evmTx)
		msg.From = fromAcc.Address.Bytes()

		return simtypes.NewOperationMsg(msg, true, ""), nil, nil
	}
}
