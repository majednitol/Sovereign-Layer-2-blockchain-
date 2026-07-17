package settlement

import (
	"context"
	"crypto/ed25519"
	"encoding/json"
	"fmt"

	storetypes "github.com/cosmos/cosmos-sdk/store/v2/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
)

type BankKeeper interface {
	SendCoins(ctx context.Context, fromAddr sdk.AccAddress, toAddr sdk.AccAddress, amt sdk.Coins) error
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
		return Params{
			TimestampToleranceSeconds: 30,
		}
	}
	var params Params
	if err := json.Unmarshal(bz, &params); err != nil {
		panic(fmt.Sprintf("failed to unmarshal settlement params: %v", err))
	}
	return params
}

func (k Keeper) SetParams(ctx sdk.Context, params Params) {
	store := ctx.KVStore(k.storeKey)
	bz, _ := json.Marshal(params)
	store.Set(ParamsKey, bz)
}

func (k Keeper) SetWitnessPubKey(ctx sdk.Context, witnessID string, pubKey []byte) {
	store := ctx.KVStore(k.storeKey)
	store.Set(append(WitnessKeyPrefix, []byte(witnessID)...), pubKey)
}

func (k Keeper) DeleteWitnessPubKey(ctx sdk.Context, witnessID string) {
	store := ctx.KVStore(k.storeKey)
	store.Delete(append(WitnessKeyPrefix, []byte(witnessID)...))
}

func (k Keeper) GetWitnessPubKey(ctx sdk.Context, witnessID string) ([]byte, bool) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(append(WitnessKeyPrefix, []byte(witnessID)...))
	if bz == nil {
		return nil, false
	}
	return bz, true
}

// GetAllWitnesses returns all registered witnesses for genesis export.
func (k Keeper) GetAllWitnesses(ctx sdk.Context) []Witness {
	store := ctx.KVStore(k.storeKey)
	iterator := storetypes.KVStorePrefixIterator(store, WitnessKeyPrefix)
	defer iterator.Close()

	var witnesses []Witness
	for ; iterator.Valid(); iterator.Next() {
		witnessID := string(iterator.Key()[len(WitnessKeyPrefix):])
		witnesses = append(witnesses, Witness{
			ID:     witnessID,
			PubKey: iterator.Value(),
		})
	}
	return witnesses
}

func (k Keeper) HasSettlementBeenProcessed(ctx sdk.Context, payloadHash []byte) bool {
	store := ctx.KVStore(k.storeKey)
	return store.Has(append(SettlementNonceKeyPrefix, payloadHash...))
}

func (k Keeper) MarkSettlementProcessed(ctx sdk.Context, payloadHash []byte) {
	store := ctx.KVStore(k.storeKey)
	store.Set(append(SettlementNonceKeyPrefix, payloadHash...), []byte{0x01})
}

// GetAllProcessedNonces returns all processed nonces for genesis export.
func (k Keeper) GetAllProcessedNonces(ctx sdk.Context) [][]byte {
	store := ctx.KVStore(k.storeKey)
	iterator := storetypes.KVStorePrefixIterator(store, SettlementNonceKeyPrefix)
	defer iterator.Close()

	var nonces [][]byte
	for ; iterator.Valid(); iterator.Next() {
		nonce := iterator.Key()[len(SettlementNonceKeyPrefix):]
		nonceCopy := make([]byte, len(nonce))
		copy(nonceCopy, nonce)
		nonces = append(nonces, nonceCopy)
	}
	return nonces
}

func (k Keeper) ProcessSettlement(ctx sdk.Context, msg MsgSettlement) error {
	// 0. Prevent replay attacks
	if k.HasSettlementBeenProcessed(ctx, msg.PayloadHash) {
		return fmt.Errorf("settlement payload has already been processed")
	}

	// 1. Retrieve witness public key
	pubKey, ok := k.GetWitnessPubKey(ctx, msg.WitnessID)
	if !ok {
		return fmt.Errorf("witness ID %s is not registered", msg.WitnessID)
	}

	// 2. Validate timestamp tolerance
	params := k.GetParams(ctx)
	blockTime := ctx.BlockTime().Unix()

	// H-02: Reject future-dated settlements beyond maximum future offset (5 minutes)
	const MaxFutureTimestampOffset = 300 // 5 minutes
	if msg.Timestamp > blockTime+MaxFutureTimestampOffset {
		return fmt.Errorf("settlement timestamp %d is too far in the future (max offset: %ds)", msg.Timestamp, MaxFutureTimestampOffset)
	}

	diff := msg.Timestamp - blockTime
	if diff < 0 {
		diff = -diff
	}
	if diff > params.TimestampToleranceSeconds {
		return fmt.Errorf("settlement timestamp %d deviates too far from block time %d (tolerance: %ds)",
			msg.Timestamp, blockTime, params.TimestampToleranceSeconds)
	}

	// 3. Verify Ed25519 signature
	domainSeparator := ComputeDomainSeparator(ctx.ChainID(), msg.PayloadHash)
	if !ed25519.Verify(pubKey, domainSeparator, msg.Signature) {
		return fmt.Errorf("invalid witness signature for settlement payload")
	}

	// 4. Perform payout / bank transfer
	destAddr, err := sdk.AccAddressFromBech32(msg.TransferDest)
	if err != nil {
		return fmt.Errorf("invalid transfer destination: %w", err)
	}

	if k.bankKeeper != nil {
		escrowAddr := authtypes.NewModuleAddress(ModuleName)
		err = k.bankKeeper.SendCoins(ctx, escrowAddr, destAddr, msg.TransferAmt)
		if err != nil {
			return fmt.Errorf("failed to transfer settlement payout: %w", err)
		}
	}

	// 4.5. Mark payload as processed
	k.MarkSettlementProcessed(ctx, msg.PayloadHash)

	// 5. Emit event
	ctx.EventManager().EmitEvent(sdk.NewEvent(
		"settlement_executed",
		sdk.NewAttribute("witness_id", msg.WitnessID),
		sdk.NewAttribute("destination", msg.TransferDest),
	))

	return nil
}
