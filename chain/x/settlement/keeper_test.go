package settlement

import (
	"context"
	"crypto/ed25519"
	"testing"
	"time"

	"cosmossdk.io/math"
	legacytypes "github.com/cosmos/cosmos-sdk/store/v2/types"
	"github.com/cosmos/cosmos-sdk/store/v2/dbadapter"
	dbm "github.com/cosmos/cosmos-db"
	storetypes "github.com/cosmos/cosmos-sdk/store/v2/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type mockMultiStore struct {
	storetypes.MultiStore
	store storetypes.KVStore
}

func (m mockMultiStore) GetKVStore(key storetypes.StoreKey) storetypes.KVStore {
	return m.store
}

func (m mockMultiStore) GetStore(key storetypes.StoreKey) storetypes.Store {
	return m.store
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

type mockBankKeeper struct {
	transfers []struct {
		dest sdk.AccAddress
		amt  sdk.Coins
	}
}

func (m *mockBankKeeper) SendCoins(ctx context.Context, fromAddr sdk.AccAddress, toAddr sdk.AccAddress, amt sdk.Coins) error {
	m.transfers = append(m.transfers, struct {
		dest sdk.AccAddress
		amt  sdk.Coins
	}{toAddr, amt})
	return nil
}

func setupKeeper(t *testing.T, bank BankKeeper) (Keeper, sdk.Context) {
	db := dbm.NewMemDB()
	kvStore := kvStoreV2Wrapper{dbadapter.Store{DB: db}}
	ms := mockMultiStore{store: kvStore}
	ctx := sdk.Context{}.
		WithMultiStore(ms).
		WithGasMeter(legacytypes.NewInfiniteGasMeter()).
		WithBlockTime(time.Unix(1000, 0)).
		WithChainID("sovereign-devnet-1").
		WithEventManager(sdk.NewEventManager())

	storeKey := legacytypes.NewKVStoreKey(StoreKey)
	keeper := NewKeeper(storeKey, nil, bank)
	return keeper, ctx
}

func TestWitnessSettlement(t *testing.T) {
	bank := &mockBankKeeper{}
	keeper, ctx := setupKeeper(t, bank)

	// Generate witness keys
	pubKey, privKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("Failed to generate keys: %v", err)
	}

	witnessID := "witness_abc"
	keeper.SetWitnessPubKey(ctx, witnessID, pubKey)

	payloadHash := []byte("payload_hash_val_abc_123__________")
	domainSeparator := ComputeDomainSeparator(ctx.ChainID(), payloadHash)
	signature := ed25519.Sign(privKey, domainSeparator)

	dest := sdk.AccAddress([]byte("payout_dest_addr____")).String()

	msg := MsgSettlement{
		Submitter:    sdk.AccAddress([]byte("submitter___________")).String(),
		WitnessID:    witnessID,
		Timestamp:    1000, // exact block time
		PayloadHash:  payloadHash,
		Signature:    signature,
		TransferAmt:  sdk.NewCoins(sdk.NewCoin("ucsov", math.NewInt(1000))),
		TransferDest: dest,
	}

	// Case 1: Unregistered witness ID -> fails
	invalidMsg := msg
	invalidMsg.WitnessID = "unregistered_id"
	err = keeper.ProcessSettlement(ctx, invalidMsg)
	if err == nil {
		t.Error("Expected error for unregistered witness ID")
	}

	// Case 2: Timestamp deviation too large (1040 - 1000 = 40s > 30s tolerance) -> fails
	invalidMsg = msg
	invalidMsg.Timestamp = 1040
	err = keeper.ProcessSettlement(ctx, invalidMsg)
	if err == nil {
		t.Error("Expected timestamp deviation error")
	}

	// Case 3: Domain separator chain-ID mismatch (using different chain-ID) -> fails
	wrongSeparator := ComputeDomainSeparator("different-chain-2", payloadHash)
	wrongSignature := ed25519.Sign(privKey, wrongSeparator)
	invalidMsg = msg
	invalidMsg.Signature = wrongSignature
	err = keeper.ProcessSettlement(ctx, invalidMsg)
	if err == nil {
		t.Error("Expected signature verification error due to domain separator mismatch")
	}

	// Case 4: Correct message -> succeeds
	err = keeper.ProcessSettlement(ctx, msg)
	if err != nil {
		t.Errorf("Expected successful settlement verification, got: %v", err)
	}

	// Verify transfer occurred
	if len(bank.transfers) != 1 {
		t.Fatalf("Expected 1 transfer to be triggered, got %d", len(bank.transfers))
	}
	expectedDest := sdk.AccAddress([]byte("payout_dest_addr____"))
	if !bank.transfers[0].dest.Equals(expectedDest) {
		t.Errorf("Expected transfer to %s, got %s", expectedDest, bank.transfers[0].dest)
	}
}
