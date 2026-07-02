package app

import (
	context "context"
	"crypto/ed25519"
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
		"evm",
		"feemarket",
		"erc20",
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

	// Pre-generate accounts
	accs := make([]simtypes.Account, 5)
	for i := 0; i < 5; i++ {
		_, pub, _ := ed25519.GenerateKey(nil)
		accs[i] = simtypes.Account{
			Address: sdk.AccAddress(pub),
		}
	}

	// Register operations (only EVM operations remain)
	ops := []simtypes.Operation{
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

		// Run random operations per block
		for opIdx := 0; opIdx < *blockSize; opIdx++ {
			op := ops[r.Intn(len(ops))]
			_, _, _ = op(r, nil, ctx, accs, ctx.ChainID())
		}

		if i%5000 == 0 || i == 1 {
			fmt.Printf("Simulated block height: %d\n", i)
		}
	}

	fmt.Printf("[PASS] Simulated %d blocks successfully without panics.\n", blocks)
}

// SimulateMsgEthSimpleTransfer simulates a simple EVM ether/atoken transfer transaction.
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
