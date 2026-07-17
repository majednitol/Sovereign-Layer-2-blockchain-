package bridge

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"

	"cosmossdk.io/math"
	storetypes "github.com/cosmos/cosmos-sdk/store/v2/types"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type BankKeeper interface {
	MintCoins(ctx context.Context, moduleName string, amt sdk.Coins) error
	BurnCoins(ctx context.Context, moduleName string, amt sdk.Coins) error
	SendCoinsFromModuleToAccount(ctx context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error
	SendCoinsFromAccountToModule(ctx context.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error
}

type Keeper struct {
	storeKey   storetypes.StoreKey
	cdc        codec.BinaryCodec
	bankKeeper BankKeeper
}

func NewKeeper(storeKey storetypes.StoreKey, cdc codec.BinaryCodec, bankKeeper BankKeeper) Keeper {
	return Keeper{
		storeKey:   storeKey,
		cdc:        cdc,
		bankKeeper: bankKeeper,
	}
}

func (k Keeper) GetParams(ctx sdk.Context) Params {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(ParamsKey)
	if bz == nil {
		// C1 FIX: Return safe zero-value defaults instead of placeholder addresses.
		// These defaults will cause ValidateGenesis to fail if genesis doesn't
		// explicitly set bridge params — which is the desired behavior.
		return Params{
			StandardFinalityDepth:  15,
			LargeFinalityDepth:     50,
			LargeTransferThreshold: 5000000000,
			QuorumThreshold:        3,
			MaxUnlockPerBlock:      100000000000,
			CircuitBreakerAddress:  "", // MUST be set via genesis
			GnosisSafeAddress:      "", // MUST be set via genesis
			SupplyCap:              "0", // MUST be set via genesis — "0" blocks all deposits
			LockBoxAddress:         "", // MUST be set via genesis
		}
	}
	var params Params
	if err := json.Unmarshal(bz, &params); err != nil {
		panic(fmt.Sprintf("failed to unmarshal bridge params from store: %v", err))
	}
	return params
}

func (k Keeper) SetParams(ctx sdk.Context, params Params) {
	store := ctx.KVStore(k.storeKey)
	bz, _ := json.Marshal(params)
	store.Set(ParamsKey, bz)
}

// Relayer management
func (k Keeper) GetRelayers(ctx sdk.Context) []Relayer {
	store := ctx.KVStore(k.storeKey)
	iterator := storetypes.KVStorePrefixIterator(store, RelayerKeyPrefix)
	defer iterator.Close()

	var relayers []Relayer
	for ; iterator.Valid(); iterator.Next() {
		var r Relayer
		_ = json.Unmarshal(iterator.Value(), &r)
		relayers = append(relayers, r)
	}
	return relayers
}

func (k Keeper) SetRelayer(ctx sdk.Context, r Relayer) {
	store := ctx.KVStore(k.storeKey)
	bz, _ := json.Marshal(r)
	store.Set(append(RelayerKeyPrefix, []byte(r.Address)...), bz)
}

func (k Keeper) DeleteRelayer(ctx sdk.Context, address string) {
	store := ctx.KVStore(k.storeKey)
	store.Delete(append(RelayerKeyPrefix, []byte(address)...))
}

func (k Keeper) GetCosmosMinted(ctx sdk.Context) math.Int {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get([]byte("cosmos_minted"))
	if bz == nil {
		return math.ZeroInt()
	}
	var s string
	if err := json.Unmarshal(bz, &s); err != nil {
		panic(fmt.Sprintf("failed to unmarshal cosmos_minted: %v", err))
	}
	val, ok := math.NewIntFromString(s)
	if !ok {
		panic(fmt.Sprintf("failed to parse cosmos_minted: %s", s))
	}
	return val
}

func (k Keeper) SetCosmosMinted(ctx sdk.Context, val math.Int) {
	store := ctx.KVStore(k.storeKey)
	bz, err := json.Marshal(val.String())
	if err != nil {
		panic(err)
	}
	store.Set([]byte("cosmos_minted"), bz)
}

// Nonce bitmap management (arbitrary length nonce out-of-order execution)
func (k Keeper) IsNonceProcessed(ctx sdk.Context, nonce []byte) bool {
	store := ctx.KVStore(k.storeKey)

	var nonceInt big.Int
	nonceInt.SetBytes(nonce)

	var wordIndex big.Int
	wordIndex.Div(&nonceInt, big.NewInt(256))

	var bitIndex big.Int
	bitIndex.Mod(&nonceInt, big.NewInt(256))

	key := append(NonceKeyPrefix, wordIndex.Bytes()...)
	bz := store.Get(key)
	if bz == nil {
		return false
	}

	var word big.Int
	word.SetBytes(bz)

	var bitMask big.Int
	bitMask.Lsh(big.NewInt(1), uint(bitIndex.Uint64()))

	var andResult big.Int
	andResult.And(&word, &bitMask)

	return andResult.Cmp(big.NewInt(0)) != 0
}

func (k Keeper) SetNonceProcessed(ctx sdk.Context, nonce []byte) {
	store := ctx.KVStore(k.storeKey)

	var nonceInt big.Int
	nonceInt.SetBytes(nonce)

	var wordIndex big.Int
	wordIndex.Div(&nonceInt, big.NewInt(256))

	var bitIndex big.Int
	bitIndex.Mod(&nonceInt, big.NewInt(256))

	key := append(NonceKeyPrefix, wordIndex.Bytes()...)
	bz := store.Get(key)

	var word big.Int
	if bz != nil {
		word.SetBytes(bz)
	}

	var bitMask big.Int
	bitMask.Lsh(big.NewInt(1), uint(bitIndex.Uint64()))

	word.Or(&word, &bitMask)
	store.Set(key, word.Bytes())
}

// ProcessBridgeIn handles minting logic after validating signatures and supply caps
func (k Keeper) ProcessBridgeIn(ctx sdk.Context, msg MsgBridgeIn) error {
	params := k.GetParams(ctx)

	// 1. Verify nonce replay protection
	if k.IsNonceProcessed(ctx, msg.Nonce) {
		return fmt.Errorf("nonce has already been processed")
	}

	// 2. Aggregate and check signatures
	hash := ComputeBridgeMessageHash(msg.Receiver, msg.Amount, msg.Nonce)
	relayers := k.GetRelayers(ctx)
	signedMap := make(map[string]bool)

	for _, sig := range msg.Signatures {
		var cleanSig []byte
		if len(sig) == 65 {
			cleanSig = sig[:64] // Strip recovery byte V for CometBFT/Cosmos verification
		} else {
			cleanSig = sig
		}

		for _, relayer := range relayers {
			if signedMap[relayer.Address] {
				continue
			}
			pubKey := secp256k1.PubKey{Key: relayer.PubKey}
			if pubKey.VerifySignature(hash, cleanSig) {
				signedMap[relayer.Address] = true
				break
			}
		}
	}

	if len(signedMap) < int(params.QuorumThreshold) {
		return fmt.Errorf("insufficient unique relayer signatures: got %d, expected %d", len(signedMap), params.QuorumThreshold)
	}

	// 3. Verify supply cap atomic invariant
	sovCoins := msg.Amount.AmountOf("uwsov")
	cosmosMinted := k.GetCosmosMinted(ctx)
	newMinted := cosmosMinted.Add(sovCoins)

	supplyCap, ok := math.NewIntFromString(params.SupplyCap)
	if !ok {
		return fmt.Errorf("invalid supply cap configured in params: %s", params.SupplyCap)
	}

	if newMinted.GT(supplyCap) {
		return fmt.Errorf("bridge deposit of %s exceeds supply cap (%s)", msg.Amount.String(), supplyCap.String())
	}

	// 4. Update state and mint tokens
	k.SetNonceProcessed(ctx, msg.Nonce)
	k.SetCosmosMinted(ctx, newMinted)

	recipientAddr, err := sdk.AccAddressFromBech32(msg.Receiver)
	if err != nil {
		return fmt.Errorf("invalid receiver address: %w", err)
	}

	if k.bankKeeper != nil {
		err = k.bankKeeper.MintCoins(ctx, ModuleName, msg.Amount)
		if err != nil {
			return fmt.Errorf("failed to mint coins: %w", err)
		}
		err = k.bankKeeper.SendCoinsFromModuleToAccount(ctx, ModuleName, recipientAddr, msg.Amount)
		if err != nil {
			return fmt.Errorf("failed to send minted coins: %w", err)
		}
	}

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"MsgBridgeIn",
		sdk.NewAttribute("receiver", msg.Receiver),
		sdk.NewAttribute("amount", msg.Amount.String()),
		sdk.NewAttribute("nonce", fmt.Sprintf("%x", msg.Nonce)),
	))

	return nil
}

// ProcessBridgeOut burns tokens to initiate a transfer out to BSC
func (k Keeper) ProcessBridgeOut(ctx sdk.Context, msg MsgBridgeOut) error {
	senderAddr, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return fmt.Errorf("invalid sender address: %w", err)
	}

	sovCoins := msg.Amount.AmountOf("uwsov")
	cosmosMinted := k.GetCosmosMinted(ctx)
	if cosmosMinted.LT(sovCoins) {
		return fmt.Errorf("insufficient bridge circulation: current %s, burning %s", cosmosMinted.String(), sovCoins.String())
	}

	if k.bankKeeper != nil {
		err = k.bankKeeper.SendCoinsFromAccountToModule(ctx, senderAddr, ModuleName, msg.Amount)
		if err != nil {
			return fmt.Errorf("failed to send coins to bridge module: %w", err)
		}
		err = k.bankKeeper.BurnCoins(ctx, ModuleName, msg.Amount)
		if err != nil {
			return fmt.Errorf("failed to burn coins: %w", err)
		}
	}

	k.SetCosmosMinted(ctx, cosmosMinted.Sub(sovCoins))

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"MsgBridgeOut",
		sdk.NewAttribute("sender", msg.Sender),
		sdk.NewAttribute("bsc_recipient", msg.BscRecipient),
		sdk.NewAttribute("amount", msg.Amount.String()),
	))

	return nil
}

func (k Keeper) RegisterInvariants(ir sdk.InvariantRegistry) {
	ir.RegisterRoute(ModuleName, "supply", k.SupplyInvariant)
	ir.RegisterRoute(ModuleName, "nonce-bitmap", k.NonceBitmapInvariant)
}

func (k Keeper) SupplyInvariant(ctx sdk.Context) (string, bool) {
	cosmosMinted := k.GetCosmosMinted(ctx)
	params := k.GetParams(ctx)
	supplyCap, ok := math.NewIntFromString(params.SupplyCap)
	if !ok {
		return fmt.Sprintf("invalid supply cap configured: %s", params.SupplyCap), true
	}
	if cosmosMinted.GT(supplyCap) {
		return fmt.Sprintf("bridge supply cap breached: minted %s, cap %s", cosmosMinted.String(), supplyCap.String()), true
	}
	return "bridge supply cap invariant holds", false
}

func (k Keeper) NonceBitmapInvariant(ctx sdk.Context) (string, bool) {
	store := ctx.KVStore(k.storeKey)
	iterator := storetypes.KVStorePrefixIterator(store, NonceKeyPrefix)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		valBz := iterator.Value()
		if len(valBz) == 0 {
			return "bridge nonce bitmap invariant breach: empty word found in store", true
		}
		var word big.Int
		word.SetBytes(valBz)
		if word.Cmp(big.NewInt(0)) == 0 {
			return "bridge nonce bitmap invariant breach: zero word found in store", true
		}
	}
	return "bridge nonce bitmap invariant holds", false
}
