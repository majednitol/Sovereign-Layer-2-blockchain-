package e2e

import (
	"context"
	"encoding/json"
	"math/big"
	"os"
	"strings"
	"testing"

	"cosmossdk.io/math"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/store/v2/dbadapter"
	storetypes "github.com/cosmos/cosmos-sdk/store/v2/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	"github.com/sovereign-l1/chain/x/bridge"
	"github.com/sovereign-l1/chain/x/certification"
	"github.com/sovereign-l1/chain/x/oracle"
	"github.com/sovereign-l1/chain/x/validator"
)

// mockStakingKeeperPhase8 implements validator.StakingKeeper and certification.StakingKeeper
type mockStakingKeeperPhase8 struct {
	validators []sdk.ValAddress
}

func (m mockStakingKeeperPhase8) GetLastValidatorPower(ctx context.Context, valAddr sdk.ValAddress) (int64, error) {
	for _, v := range m.validators {
		if v.Equals(valAddr) {
			return 100, nil
		}
	}
	return 0, nil
}

func (m mockStakingKeeperPhase8) GetLastTotalPower(ctx context.Context) (math.Int, error) {
	return math.NewInt(int64(len(m.validators) * 100)), nil
}

func (m mockStakingKeeperPhase8) IterateLastValidatorPowers(ctx context.Context, handler func(valAddr sdk.ValAddress, power int64) (stop bool)) error {
	for _, v := range m.validators {
		if handler(v, 100) {
			break
		}
	}
	return nil
}

func (m mockStakingKeeperPhase8) GetValidator(ctx context.Context, valAddr sdk.ValAddress) (stakingtypes.Validator, error) {
	return stakingtypes.Validator{
		OperatorAddress: valAddr.String(),
	}, nil
}

// mockSlashingKeeperPhase8 implements validator.SlashingKeeper and certification.SlashingKeeper
type mockSlashingKeeperPhase8 struct{}

func (m mockSlashingKeeperPhase8) Tombstone(ctx context.Context, valAddr sdk.ConsAddress) error {
	return nil
}
func (m mockSlashingKeeperPhase8) HasValidatorSigningInfo(ctx context.Context, consAddr sdk.ConsAddress) bool {
	return true
}
func (m mockSlashingKeeperPhase8) SetValidatorSigningInfo(ctx context.Context, address sdk.ConsAddress, info slashingtypes.ValidatorSigningInfo) error {
	return nil
}
func (m mockSlashingKeeperPhase8) Slash(ctx context.Context, consAddr sdk.ConsAddress, fraction math.LegacyDec, power, distributionHeight int64) error {
	return nil
}
func (m mockSlashingKeeperPhase8) Jail(ctx context.Context, consAddr sdk.ConsAddress) error {
	return nil
}

// mockBankKeeperPhase8 implements validator.BankKeeper
type mockBankKeeperPhase8 struct {
	balance int64
}

func (m mockBankKeeperPhase8) GetBalance(ctx context.Context, addr sdk.AccAddress, denom string) sdk.Coin {
	return sdk.NewCoin(denom, math.NewInt(m.balance))
}

// mockBridgeBankKeeper implements bridge.BankKeeper
type mockBridgeBankKeeper struct{}

func (m mockBridgeBankKeeper) MintCoins(ctx context.Context, moduleName string, amt sdk.Coins) error {
	return nil
}
func (m mockBridgeBankKeeper) BurnCoins(ctx context.Context, moduleName string, amt sdk.Coins) error {
	return nil
}
func (m mockBridgeBankKeeper) SendCoinsFromModuleToAccount(ctx context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error {
	return nil
}
func (m mockBridgeBankKeeper) SendCoinsFromAccountToModule(ctx context.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error {
	return nil
}

// mockDistrKeeperPhase8 implements validator.DistrKeeper
type mockDistrKeeperPhase8 struct {
	rewards map[string]int64
}

func (m mockDistrKeeperPhase8) GetValidatorOutstandingRewards(ctx context.Context, valAddr sdk.ValAddress) (distrtypes.ValidatorOutstandingRewards, error) {
	amt := m.rewards[valAddr.String()]
	decCoin := sdk.NewDecCoin("usov", math.NewInt(amt))
	return distrtypes.ValidatorOutstandingRewards{
		Rewards: sdk.DecCoins{decCoin},
	}, nil
}

func TestPhase8OracleStalenessInvariant(t *testing.T) {
	db := dbm.NewMemDB()
	storeKey := storetypes.NewKVStoreKey(oracle.StoreKey)
	dbMap := map[string]storetypes.KVStore{
		oracle.StoreKey: kvStoreV2Wrapper{dbadapter.Store{DB: db}},
	}
	ms := mockMultiStore{stores: dbMap}
	ctx := sdk.Context{}.WithMultiStore(ms).WithGasMeter(storetypes.NewInfiniteGasMeter()).WithBlockHeight(100)

	staking := &mockStakingKeeper{}
	slashing := &mockSlashingKeeper{}
	k := oracle.NewKeeper(storeKey, nil, staking, slashing)
	k.SetParams(ctx, oracle.Params{
		StalenessThresholdBlocks: 10,
	})

	// 1. Initial State -> holds
	msg, breached := k.StalenessInvariant(ctx)
	if breached {
		t.Fatalf("Expected no breach initially, got: %s", msg)
	}

	// 2. Add valid price -> holds
	feedID := "BTC_USD"
	agg := oracle.AggregatePrice{
		Price:       60000,
		BlockHeight: 95,
	}
	bz, _ := json.Marshal(agg)
	ctx.KVStore(storeKey).Set(append(oracle.AggregateKeyPrefix, []byte(feedID)...), bz)

	msg, breached = k.StalenessInvariant(ctx)
	if breached {
		t.Fatalf("Expected no breach for fresh price, got: %s", msg)
	}

	// 3. Add corrupt JSON -> breaches
	ctx.KVStore(storeKey).Set(append(oracle.AggregateKeyPrefix, []byte(feedID)...), []byte("invalid-json"))
	msg, breached = k.StalenessInvariant(ctx)
	if !breached {
		t.Fatal("Expected breach for corrupt JSON in store")
	}
	if !strings.Contains(msg, "failed to unmarshal") {
		t.Fatalf("Expected unmarshal breach message, got: %s", msg)
	}
}

func TestPhase8NonceBitmapInvariant(t *testing.T) {
	db := dbm.NewMemDB()
	storeKey := storetypes.NewKVStoreKey(bridge.StoreKey)
	dbMap := map[string]storetypes.KVStore{
		bridge.StoreKey: kvStoreV2Wrapper{dbadapter.Store{DB: db}},
	}
	ms := mockMultiStore{stores: dbMap}
	ctx := sdk.Context{}.WithMultiStore(ms).WithGasMeter(storetypes.NewInfiniteGasMeter())

	bank := mockBridgeBankKeeper{}
	k := bridge.NewKeeper(storeKey, nil, bank)

	// 1. Initial state (no nonces in store) -> holds
	msg, breached := k.NonceBitmapInvariant(ctx)
	if breached {
		t.Fatalf("Expected no breach initially, got: %s", msg)
	}

	// 2. Add valid nonce word (non-zero big.Int) -> holds
	wordIdx := big.NewInt(0)
	wordVal := big.NewInt(7) // binary 111 (nonces 0, 1, 2 processed)
	key := append(bridge.NonceKeyPrefix, wordIdx.Bytes()...)
	ctx.KVStore(storeKey).Set(key, wordVal.Bytes())

	msg, breached = k.NonceBitmapInvariant(ctx)
	if breached {
		t.Fatalf("Expected no breach for valid word, got: %s", msg)
	}

	// 3. Add zero word -> breaches
	ctx.KVStore(storeKey).Set(key, []byte{0x00})
	msg, breached = k.NonceBitmapInvariant(ctx)
	if !breached {
		t.Fatal("Expected breach for zero word in nonce bitmap")
	}
	if !strings.Contains(msg, "zero word found") {
		t.Fatalf("Expected zero word breach message, got: %s", msg)
	}

	// 4. Add empty bytes -> breaches
	ctx.KVStore(storeKey).Set(key, []byte{})
	msg, breached = k.NonceBitmapInvariant(ctx)
	if !breached {
		t.Fatal("Expected breach for empty word in nonce bitmap")
	}
	if !strings.Contains(msg, "empty word found") {
		t.Fatalf("Expected empty word breach message, got: %s", msg)
	}
}

func TestPhase8SupplyInvariant(t *testing.T) {
	db := dbm.NewMemDB()
	storeKey := storetypes.NewKVStoreKey(bridge.StoreKey)
	dbMap := map[string]storetypes.KVStore{
		bridge.StoreKey: kvStoreV2Wrapper{dbadapter.Store{DB: db}},
	}
	ms := mockMultiStore{stores: dbMap}
	ctx := sdk.Context{}.WithMultiStore(ms).WithGasMeter(storetypes.NewInfiniteGasMeter())

	bank := mockBridgeBankKeeper{}
	k := bridge.NewKeeper(storeKey, nil, bank)
	k.SetParams(ctx, bridge.Params{
		SupplyCap: 1000,
	})

	// 1. Initially supply is 0 -> holds
	msg, breached := k.SupplyInvariant(ctx)
	if breached {
		t.Fatalf("Expected no breach initially, got: %s", msg)
	}

	// 2. Set minted supply within cap -> holds
	k.SetCosmosMinted(ctx, 999)
	msg, breached = k.SupplyInvariant(ctx)
	if breached {
		t.Fatalf("Expected no breach within cap, got: %s", msg)
	}

	// 3. Set minted supply above cap -> breaches
	k.SetCosmosMinted(ctx, 1001)
	msg, breached = k.SupplyInvariant(ctx)
	if !breached {
		t.Fatal("Expected breach when supply exceeds cap")
	}
	if !strings.Contains(msg, "supply cap breached") {
		t.Fatalf("Expected supply cap breach message, got: %s", msg)
	}
}

func TestPhase8RewardsBucketInvariant(t *testing.T) {
	db := dbm.NewMemDB()
	storeKey := storetypes.NewKVStoreKey(validator.StoreKey)
	dbMap := map[string]storetypes.KVStore{
		validator.StoreKey: kvStoreV2Wrapper{dbadapter.Store{DB: db}},
	}
	ms := mockMultiStore{stores: dbMap}
	ctx := sdk.Context{}.WithMultiStore(ms).WithGasMeter(storetypes.NewInfiniteGasMeter())

	val1 := sdk.ValAddress([]byte("validator1__________"))
	staking := &mockStakingKeeperPhase8{
		validators: []sdk.ValAddress{val1},
	}
	slashing := &mockSlashingKeeperPhase8{}
	bank := &mockBankKeeperPhase8{
		balance: 1000,
	}
	distr := &mockDistrKeeperPhase8{
		rewards: map[string]int64{
			val1.String(): 900,
		},
	}

	k := validator.NewKeeper(storeKey, nil, staking, slashing, bank, distr, 30)

	// 1. Outstanding rewards (900) <= distribution balance (1000) -> holds
	msg, breached := k.RewardsBucketInvariant(ctx)
	if breached {
		t.Fatalf("Expected no breach when balance covers rewards, got: %s", msg)
	}

	// 2. Outstanding rewards (1100) > distribution balance (1000) -> breaches
	distr.rewards[val1.String()] = 1100
	msg, breached = k.RewardsBucketInvariant(ctx)
	if !breached {
		t.Fatal("Expected breach when rewards exceed balance")
	}
	if !strings.Contains(msg, "rewards bucket invariant breach") {
		t.Fatalf("Expected rewards bucket breach message, got: %s", msg)
	}
}

func TestPhase8WindowConsistencyInvariant(t *testing.T) {
	db := dbm.NewMemDB()
	storeKey := storetypes.NewKVStoreKey(certification.StoreKey)
	dbMap := map[string]storetypes.KVStore{
		certification.StoreKey: kvStoreV2Wrapper{dbadapter.Store{DB: db}},
	}
	ms := mockMultiStore{stores: dbMap}
	ctx := sdk.Context{}.WithMultiStore(ms).WithGasMeter(storetypes.NewInfiniteGasMeter()).WithBlockHeight(10)

	val1 := sdk.ValAddress([]byte("validator1__________"))
	staking := &mockStakingKeeperPhase8{
		validators: []sdk.ValAddress{val1},
	}
	slashing := &mockSlashingKeeperPhase8{}
	k := certification.NewKeeper(storeKey, nil, staking, slashing)

	// 1. Window signed bits sum is 0, stored signed count is 0 -> holds
	msg, breached := k.WindowConsistencyInvariant(ctx)
	if breached {
		t.Fatalf("Expected no breach initially, got: %s", msg)
	}

	// 2. Window has 3 signed blocks, stored signed count is 3 -> holds
	// We simulate setting signed bits at heights 2, 4, 6
	k.SetValidatorSignedBit(ctx, val1, 2, true)
	k.SetValidatorSignedBit(ctx, val1, 4, true)
	k.SetValidatorSignedBit(ctx, val1, 6, true)
	k.SetValidatorSignedCount(ctx, val1, 3)

	msg, breached = k.WindowConsistencyInvariant(ctx)
	if breached {
		t.Fatalf("Expected no breach when stored matches calculated count, got: %s", msg)
	}

	// 3. Stored signed count is 4, but window signed bits sum is 3 -> breaches
	k.SetValidatorSignedCount(ctx, val1, 4)
	msg, breached = k.WindowConsistencyInvariant(ctx)
	if !breached {
		t.Fatal("Expected breach when stored count mismatches calculated bits")
	}
	if !strings.Contains(msg, "signed count mismatch") {
		t.Fatalf("Expected mismatch breach message, got: %s", msg)
	}
}

func TestPhase8DockerfileVerification(t *testing.T) {
	// Read Dockerfile and assert pinned digests exist
	data, err := os.ReadFile("../chain/Dockerfile")
	if err != nil {
		t.Fatalf("Failed to read Dockerfile: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "golang:1.25.9@sha256:d8c558f623b3f2df9547d25e01eb4a86b5155f9a7adfb2cd33a2072f9dcfbe2a") {
		t.Error("Dockerfile golang base image digest is not pinned correctly")
	}
	if !strings.Contains(content, "debian:trixie-slim@sha256:1f7a6b8c946bc77b9492167d7162a8cb9815ea7e31b67272828b610c14b9a9f2") {
		t.Error("Dockerfile debian base image digest is not pinned correctly")
	}
}

func TestPhase8RunbooksVerification(t *testing.T) {
	// Assert runbook and threat model documentation exist
	if _, err := os.Stat("../doc/ops/runbooks.md"); os.IsNotExist(err) {
		t.Error("Operations runbook doc/ops/runbooks.md does not exist")
	}
	if _, err := os.Stat("../doc/ops/security_threat_model.md"); os.IsNotExist(err) {
		t.Error("Security threat model doc/ops/security_threat_model.md does not exist")
	}
}
