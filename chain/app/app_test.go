package app

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// TestLivenessBootstrapping asserts the rolling window denominator behaves correctly.
func TestLivenessBootstrapping(t *testing.T) {
	windowSize := int64(10000)

	// Case 1: Height H <= 1 (Edge Case)
	ratioH1 := GetLivenessSigningRatio(0, 1, windowSize)
	if ratioH1 != 1.0 {
		t.Errorf("Expected ratio at block height 1 to be 1.0, got %f", ratioH1)
	}

	// Case 2: Bootstrapping Period (Height H = 100 < windowSize)
	// Denominator should be 100. If 90 blocks are signed, ratio = 90/100 = 0.90
	ratioH100 := GetLivenessSigningRatio(90, 100, windowSize)
	expectedH100 := 0.90
	if ratioH100 != expectedH100 {
		t.Errorf("Expected ratio at block height 100 to be %f, got %f", expectedH100, ratioH100)
	}

	// Case 3: Beyond Bootstrapping Period (Height H = 12000 > windowSize)
	// Denominator should be windowSize (10000). If 9000 blocks are signed, ratio = 9000/10000 = 0.90
	ratioH12000 := GetLivenessSigningRatio(9000, 12000, windowSize)
	expectedH12000 := 0.90
	if ratioH12000 != expectedH12000 {
		t.Errorf("Expected ratio at block height 12000 to be %f, got %f", expectedH12000, ratioH12000)
	}
}

// mockStakingKeeper implements the StakingKeeperStub interface for testing.
type mockStakingKeeper struct {
	powers map[string]int64
}

func (m mockStakingKeeper) GetLastValidatorPower(ctx sdk.Context, valAddr sdk.ValAddress) int64 {
	return m.powers[valAddr.String()]
}

func (m mockStakingKeeper) GetLastTotalPower(ctx sdk.Context) math.Int {
	var total int64
	for _, p := range m.powers {
		total += p
	}
	return math.NewInt(total)
}

// TestStakingCompatibilityKeeper verifies slot-based equalized voting power mappings.
func TestStakingCompatibilityKeeper(t *testing.T) {
	valAddr1 := sdk.ValAddress([]byte("val1________________"))
	valAddr2 := sdk.ValAddress([]byte("val2________________"))

	mockKeeper := mockStakingKeeper{
		powers: map[string]int64{
			valAddr1.String(): 50,
			valAddr2.String(): 0,
		},
	}
	keeper := StakingCompatibilityKeeper{
		stakingKeeper: mockKeeper,
		MaxValidators: 30,
	}

	ctx := sdk.Context{}

	// GetEqualizedValidatorPower
	p1 := keeper.GetEqualizedValidatorPower(ctx, valAddr1)
	if p1 != 1000000 {
		t.Errorf("Expected power for val1 to be 1000000, got %d", p1)
	}

	p2 := keeper.GetEqualizedValidatorPower(ctx, valAddr2)
	if p2 != 0 {
		t.Errorf("Expected power for val2 to be 0, got %d", p2)
	}

	// GetEqualizedTotalPower
	totalPower := keeper.GetEqualizedTotalPower(ctx)
	expectedTotal := math.NewInt(30000000)
	if !totalPower.Equal(expectedTotal) {
		t.Errorf("Expected total power to be %s, got %s", expectedTotal, totalPower)
	}
}

// TestWasmModuleAccounts asserts deterministic genesis CosmWasm contract accounts exist and are unique.
func TestWasmModuleAccounts(t *testing.T) {
	// Assert addresses are not empty
	if len(ConstitutionContractAddr) == 0 {
		t.Error("ConstitutionContractAddr should not be empty")
	}
	if len(TreasuryContractAddr) == 0 {
		t.Error("TreasuryContractAddr should not be empty")
	}
	if len(ReserveFundContractAddr) == 0 {
		t.Error("ReserveFundContractAddr should not be empty")
	}
	if len(GovernanceContractAddr) == 0 {
		t.Error("GovernanceContractAddr should not be empty")
	}

	// Assert uniqueness of all pre-computed addresses
	addrs := map[string]bool{
		ConstitutionContractAddr.String(): true,
		TreasuryContractAddr.String():     true,
		ReserveFundContractAddr.String():  true,
		GovernanceContractAddr.String():   true,
	}
	if len(addrs) != 4 {
		t.Errorf("Expected 4 unique addresses, got %d", len(addrs))
	}
}

// TestNewApp verifies that the application constructor initializes components without panics.
func TestNewApp(t *testing.T) {
	app := NewApp(nil, nil, nil, false, nil)
	if app == nil {
		t.Fatal("Expected NewApp to return a non-nil App instance")
	}

	if app.Name() != Name {
		t.Errorf("Expected App Name to be %s, got %s", Name, app.Name())
	}

	if app.AppCodec() == nil {
		t.Error("Expected AppCodec to be initialized")
	}
}

