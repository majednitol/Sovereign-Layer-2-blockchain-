package bridge

import (
	"context"
	"crypto/rand"
	"testing"

	"cosmossdk.io/math"
	legacytypes "github.com/cosmos/cosmos-sdk/store/v2/types"
	"github.com/cosmos/cosmos-sdk/store/v2/dbadapter"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
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
	minted   sdk.Coins
	burned   sdk.Coins
	balances map[string]sdk.Coins
}

func (m *mockBankKeeper) MintCoins(ctx context.Context, moduleName string, amt sdk.Coins) error {
	m.minted = m.minted.Add(amt...)
	m.balances[moduleName] = m.balances[moduleName].Add(amt...)
	return nil
}

func (m *mockBankKeeper) BurnCoins(ctx context.Context, moduleName string, amt sdk.Coins) error {
	m.burned = m.burned.Add(amt...)
	m.balances[moduleName] = m.balances[moduleName].Sub(amt...)
	return nil
}

func (m *mockBankKeeper) SendCoinsFromModuleToAccount(ctx context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error {
	m.balances[senderModule] = m.balances[senderModule].Sub(amt...)
	m.balances[recipientAddr.String()] = m.balances[recipientAddr.String()].Add(amt...)
	return nil
}

func (m *mockBankKeeper) SendCoinsFromAccountToModule(ctx context.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error {
	m.balances[senderAddr.String()] = m.balances[senderAddr.String()].Sub(amt...)
	m.balances[recipientModule] = m.balances[recipientModule].Add(amt...)
	return nil
}

func setupKeeper(t *testing.T) (Keeper, sdk.Context, *mockBankKeeper) {
	db := dbm.NewMemDB()
	kvStore := kvStoreV2Wrapper{dbadapter.Store{DB: db}}
	ms := mockMultiStore{store: kvStore}
	ctx := sdk.Context{}.
		WithMultiStore(ms).
		WithGasMeter(legacytypes.NewInfiniteGasMeter()).
		WithChainID("sovereign-devnet-1").
		WithEventManager(sdk.NewEventManager())

	bank := &mockBankKeeper{
		balances: make(map[string]sdk.Coins),
	}

	storeKey := legacytypes.NewKVStoreKey(StoreKey)
	keeper := NewKeeper(storeKey, nil, bank)
	return keeper, ctx, bank
}

func TestKeeperParamsAndRelayers(t *testing.T) {
	keeper, ctx, _ := setupKeeper(t)

	// Test default params
	params := keeper.GetParams(ctx)
	if params.QuorumThreshold != 3 {
		t.Errorf("Expected default quorum threshold 3, got %d", params.QuorumThreshold)
	}

	// Set custom params
	params.QuorumThreshold = 2
	keeper.SetParams(ctx, params)
	params2 := keeper.GetParams(ctx)
	if params2.QuorumThreshold != 2 {
		t.Errorf("Expected updated quorum threshold 2, got %d", params2.QuorumThreshold)
	}

	// Relayer registration
	rel1 := Relayer{Address: "cosmos1rel1", PubKey: []byte("pubkey1")}
	rel2 := Relayer{Address: "cosmos1rel2", PubKey: []byte("pubkey2")}
	keeper.SetRelayer(ctx, rel1)
	keeper.SetRelayer(ctx, rel2)

	relayers := keeper.GetRelayers(ctx)
	if len(relayers) != 2 {
		t.Errorf("Expected 2 relayers, got %d", len(relayers))
	}
}

func TestNonceBitmapRegistry(t *testing.T) {
	keeper, ctx, _ := setupKeeper(t)

	// Random 256-bit nonces
	nonce1 := make([]byte, 32)
	nonce2 := make([]byte, 32)
	_, _ = rand.Read(nonce1)
	_, _ = rand.Read(nonce2)

	if keeper.IsNonceProcessed(ctx, nonce1) {
		t.Fatal("Nonce 1 should not be processed yet")
	}

	keeper.SetNonceProcessed(ctx, nonce1)
	if !keeper.IsNonceProcessed(ctx, nonce1) {
		t.Fatal("Nonce 1 should be processed")
	}
	if keeper.IsNonceProcessed(ctx, nonce2) {
		t.Fatal("Nonce 2 should not be processed")
	}
}

func TestProcessBridgeInAndOut(t *testing.T) {
	keeper, ctx, bank := setupKeeper(t)

	// Generate relayer keys
	var privs []*secp256k1.PrivKey
	for i := 0; i < 3; i++ {
		priv := secp256k1.GenPrivKey()
		privs = append(privs, priv)
		keeper.SetRelayer(ctx, Relayer{
			Address: sdk.AccAddress(priv.PubKey().Address()).String(),
			PubKey:  priv.PubKey().Bytes(),
		})
	}

	params := keeper.GetParams(ctx)
	params.QuorumThreshold = 2
	params.SupplyCap = 1000000
	keeper.SetParams(ctx, params)

	receiver := sdk.AccAddress([]byte("receiver_addr")).String()
	amount := sdk.NewCoins(sdk.NewCoin("usov", math.NewInt(50000)))
	nonce := []byte("unique_nonce_val_1234567890")

	// Pre-sign the payload hash
	hash := ComputeBridgeMessageHash(receiver, amount, nonce)
	var signatures [][]byte
	for _, priv := range privs {
		sig, err := priv.Sign(hash)
		if err != nil {
			t.Fatalf("Failed to sign: %v", err)
		}
		signatures = append(signatures, sig)
	}

	msg := MsgBridgeIn{
		Submitter:  sdk.AccAddress([]byte("submitter_addr")).String(),
		Receiver:   receiver,
		Amount:     amount,
		Nonce:      nonce,
		Signatures: signatures[:2], // 2 signatures satisfies quorum threshold (2)
	}

	// 1. Success case: BridgeIn
	err := keeper.ProcessBridgeIn(ctx, msg)
	if err != nil {
		t.Fatalf("ProcessBridgeIn failed: %v", err)
	}

	// Check balance
	if bank.balances[receiver].AmountOf("usov").Int64() != 50000 {
		t.Errorf("Expected balance 50000, got %d", bank.balances[receiver].AmountOf("usov").Int64())
	}

	// Check supply cap invariant updated
	if keeper.GetCosmosMinted(ctx) != 50000 {
		t.Errorf("Expected cosmos minted 50000, got %d", keeper.GetCosmosMinted(ctx))
	}

	// 2. Replay case: fails
	err = keeper.ProcessBridgeIn(ctx, msg)
	if err == nil {
		t.Fatal("Expected error on replay of nonce")
	}

	// 3. Quorum deficiency case: fails
	msg2 := msg
	msg2.Nonce = []byte("different_nonce_1")
	msg2.Signatures = msg2.Signatures[:1] // 1 signature < threshold (2)
	err = keeper.ProcessBridgeIn(ctx, msg2)
	if err == nil {
		t.Fatal("Expected signature quorum check to fail")
	}

	// 4. Supply cap breach case: fails
	msg3 := msg
	msg3.Nonce = []byte("different_nonce_2")
	msg3.Amount = sdk.NewCoins(sdk.NewCoin("usov", math.NewInt(960000))) // 50,000 + 960,000 = 1,010,000 > 1,000,000 cap
	// Re-sign for different amount
	hash3 := ComputeBridgeMessageHash(receiver, msg3.Amount, msg3.Nonce)
	var signatures3 [][]byte
	for _, priv := range privs {
		sig, _ := priv.Sign(hash3)
		signatures3 = append(signatures3, sig)
	}
	msg3.Signatures = signatures3[:2]
	err = keeper.ProcessBridgeIn(ctx, msg3)
	if err == nil {
		t.Fatal("Expected supply cap check to fail")
	}

	// 5. Success case: BridgeOut (Withdrawal / Burn)
	msgOut := MsgBridgeOut{
		Sender:       receiver,
		BscRecipient: "0x1111111111111111111111111111111111111111",
		Amount:       sdk.NewCoins(sdk.NewCoin("usov", math.NewInt(20000))),
	}

	err = keeper.ProcessBridgeOut(ctx, msgOut)
	if err != nil {
		t.Fatalf("ProcessBridgeOut failed: %v", err)
	}

	// Balance and supply cap updated
	if bank.balances[receiver].AmountOf("usov").Int64() != 30000 {
		t.Errorf("Expected balance 30000 after burn, got %d", bank.balances[receiver].AmountOf("usov").Int64())
	}
	if keeper.GetCosmosMinted(ctx) != 30000 {
		t.Errorf("Expected cosmos minted 30000 after burn, got %d", keeper.GetCosmosMinted(ctx))
	}
}
