package e2e

import (
	"bytes"
	"context"
	"testing"

	"cosmossdk.io/math"
	storetypes "github.com/cosmos/cosmos-sdk/store/v2/types"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdkcrypto "github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	"github.com/sovereign-l1/chain/x/oracle"
)

// Mock Staking Keeper for Oracle Slashing Test
type mockOracleStakingKeeper struct {
	validators map[string]stakingtypes.Validator
}

func (m mockOracleStakingKeeper) GetValidator(ctx context.Context, valAddr sdk.ValAddress) (stakingtypes.Validator, error) {
	val, ok := m.validators[valAddr.String()]
	if !ok {
		return stakingtypes.Validator{}, stakingtypes.ErrNoValidatorFound
	}
	return val, nil
}

// Mock Slashing Keeper for Oracle Slashing Test
type mockOracleSlashingKeeper struct {
	slashed []struct {
		consAddr sdk.ConsAddress
		fraction math.LegacyDec
		power    int64
	}
	jailed []sdk.ConsAddress
}

func (m *mockOracleSlashingKeeper) Slash(ctx context.Context, consAddr sdk.ConsAddress, fraction math.LegacyDec, power, distributionHeight int64) error {
	m.slashed = append(m.slashed, struct {
		consAddr sdk.ConsAddress
		fraction math.LegacyDec
		power    int64
	}{consAddr, fraction, power})
	return nil
}

func (m *mockOracleSlashingKeeper) Jail(ctx context.Context, consAddr sdk.ConsAddress) error {
	m.jailed = append(m.jailed, consAddr)
	return nil
}

func TestPhase2OracleSlashingAndJailing(t *testing.T) {
	ctx, _ := phase2SetupTestContext()
	storeKey := storetypes.NewKVStoreKey(oracle.StoreKey)

	// Generate a valid valoper address
	valAddr := sdk.ValAddress(sdk.AccAddress([]byte("validator_addr_12345")))
	operator := valAddr.String()

	pk := sdkcrypto.GenPrivKey().PubKey()
	anyPk, err := codectypes.NewAnyWithValue(pk)
	if err != nil {
		t.Fatalf("Failed to create Any: %v", err)
	}

	validator := stakingtypes.Validator{
		OperatorAddress: operator,
		ConsensusPubkey: anyPk,
		Tokens:          math.NewInt(100000000), // 100 Power
	}

	stakingKeeper := mockOracleStakingKeeper{
		validators: map[string]stakingtypes.Validator{
			operator: validator,
		},
	}
	slashingKeeper := &mockOracleSlashingKeeper{}

	k := oracle.NewKeeper(storeKey, nil, stakingKeeper, slashingKeeper)
	k.SetParams(ctx, oracle.Params{
		CommitWindow:             10,
		RevealWindow:             10,
		MinOperatorCommits:       1,
		StalenessThresholdBlocks: 100,
	})

	feedID := "BTC_USD"
	roundID := uint64(1)
	value := uint64(50000)
	nonce := "secret"

	// 1. Commit hash at height 10
	hash := oracle.ComputeCommitHash(operator, feedID, roundID, value, nonce)
	err = k.CommitHash(ctx, operator, feedID, roundID, hash)
	if err != nil {
		t.Fatalf("Failed to commit hash: %v", err)
	}

	// 2. Call EndBlocker within window -> no slashing
	ctx = ctx.WithBlockHeight(25) // commit height 10 + commit window 10 + reveal window 10 = 30 max reveal height. height 25 is within window.
	k.EndBlocker(ctx)
	if len(slashingKeeper.slashed) != 0 || len(slashingKeeper.jailed) != 0 {
		t.Error("Expected no slashing/jailing within reveal window")
	}

	// 3. Advance block height past the window and call EndBlocker -> triggers slashing & jailing
	ctx = ctx.WithBlockHeight(31) // block 31 is after the reveal window (30)
	k.EndBlocker(ctx)

	if len(slashingKeeper.slashed) != 1 {
		t.Fatalf("Expected 1 slashing call, got %d", len(slashingKeeper.slashed))
	}
	if len(slashingKeeper.jailed) != 1 {
		t.Fatalf("Expected 1 jailing call, got %d", len(slashingKeeper.jailed))
	}

	consAddr, err := validator.GetConsAddr()
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(slashingKeeper.slashed[0].consAddr, consAddr) {
		t.Errorf("Expected slashed validator to be %s, got %s", consAddr, slashingKeeper.slashed[0].consAddr)
	}
	expectedFraction := math.LegacyNewDecWithPrec(1, 2) // 1%
	if !slashingKeeper.slashed[0].fraction.Equal(expectedFraction) {
		t.Errorf("Expected slash fraction to be %s, got %s", expectedFraction, slashingKeeper.slashed[0].fraction)
	}
	if !bytes.Equal(slashingKeeper.jailed[0], consAddr) {
		t.Errorf("Expected jailed validator to be %s, got %s", consAddr, slashingKeeper.jailed[0])
	}
}
