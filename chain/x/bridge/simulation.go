package bridge

import (
	"crypto/sha256"
	"fmt"
	"math/rand"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
)

// SimulateMsgBridgeIn simulates a valid bridge-in message.
func SimulateMsgBridgeIn(k Keeper) simtypes.Operation {
	return func(
		r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context,
		accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		if len(accs) == 0 {
			return simtypes.NoOpMsg(ModuleName, "MsgBridgeIn", "no accounts available"), nil, nil
		}
		simAccount := accs[r.Intn(len(accs))]
		receiver := sdk.AccAddress(simAccount.Address)

		// Setup 3 relayers if none are set
		relayers := k.GetRelayers(ctx)
		var privs []*secp256k1.PrivKey
		if len(relayers) < 3 {
			for i := 0; i < 3; i++ {
				priv := secp256k1.GenPrivKey()
				privs = append(privs, priv)
				addr := sdk.AccAddress(priv.PubKey().Address()).String()
				rVal := Relayer{
					Address: addr,
					PubKey:  priv.PubKey().Bytes(),
				}
				k.SetRelayer(ctx, rVal)
				relayers = append(relayers, rVal)
			}
		} else {
			// In case they exist, we can't easily sign unless we have their private keys.
			// Let's overwrite them with known keys for the simulation.
			relayers = nil
			for i := 0; i < 3; i++ {
				priv := secp256k1.GenPrivKey()
				privs = append(privs, priv)
				addr := sdk.AccAddress(priv.PubKey().Address()).String()
				rVal := Relayer{
					Address: addr,
					PubKey:  priv.PubKey().Bytes(),
				}
				k.SetRelayer(ctx, rVal)
				relayers = append(relayers, rVal)
			}
		}

		params := k.GetParams(ctx)
		amount := sdk.NewCoins(sdk.NewCoin("uwsov", math.NewInt(int64(100+r.Intn(1000)))))
		
		// Generate standard unique nonce
		h := sha256.New()
		h.Write([]byte(receiver.String()))
		h.Write([]byte(amount.String()))
		h.Write([]byte(fmt.Sprintf("%d", r.Int63())))
		nonce := h.Sum(nil)

		hash := ComputeBridgeMessageHash(receiver.String(), amount, nonce)
		var sigs [][]byte
		for i := 0; i < int(params.QuorumThreshold); i++ {
			sig, err := privs[i].Sign(hash)
			if err != nil {
				return simtypes.NoOpMsg(ModuleName, "MsgBridgeIn", err.Error()), nil, nil
			}
			sigs = append(sigs, sig)
		}

		msg := MsgBridgeIn{
			Submitter:  relayers[0].Address,
			Receiver:   receiver.String(),
			Amount:     amount,
			Nonce:      nonce,
			Signatures: sigs,
		}

		err := k.ProcessBridgeIn(ctx, msg)
		if err != nil {
			return simtypes.NoOpMsg(ModuleName, "MsgBridgeIn", err.Error()), nil, nil
		}

		return simtypes.NewOperationMsg(&msg, true, ""), nil, nil
	}
}

// SimulateMsgBridgeOut simulates a valid bridge-out (withdrawal / burn) message.
func SimulateMsgBridgeOut(k Keeper) simtypes.Operation {
	return func(
		r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context,
		accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		if len(accs) == 0 {
			return simtypes.NoOpMsg(ModuleName, "MsgBridgeOut", "no accounts available"), nil, nil
		}
		simAccount := accs[r.Intn(len(accs))]
		sender := sdk.AccAddress(simAccount.Address)

		amount := sdk.NewCoins(sdk.NewCoin("uwsov", math.NewInt(int64(100+r.Intn(1000)))))

		// Fund the sender so burn succeeds
		if k.bankKeeper != nil {
			_ = k.bankKeeper.MintCoins(ctx, ModuleName, amount)
			_ = k.bankKeeper.SendCoinsFromModuleToAccount(ctx, ModuleName, sender, amount)
			
			// Update the cosmos_minted state to reflect the minted amount
			cosmosMinted := k.GetCosmosMinted(ctx)
			k.SetCosmosMinted(ctx, cosmosMinted.Add(amount.AmountOf("uwsov")))
		}

		msg := MsgBridgeOut{
			Sender:       sender.String(),
			BscRecipient: "0xabcdef1234567890abcdef1234567890abcdef12",
			Amount:       amount,
		}

		err := k.ProcessBridgeOut(ctx, msg)
		if err != nil {
			return simtypes.NoOpMsg(ModuleName, "MsgBridgeOut", err.Error()), nil, nil
		}

		return simtypes.NewOperationMsg(&msg, true, ""), nil, nil
	}
}

// SimulateMsgBridgeInCapBreach simulates a bridge-in that exceeds the supply cap and fails.
func SimulateMsgBridgeInCapBreach(k Keeper) simtypes.Operation {
	return func(
		r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context,
		accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		if len(accs) == 0 {
			return simtypes.NoOpMsg(ModuleName, "MsgBridgeIn", "no accounts available"), nil, nil
		}
		simAccount := accs[r.Intn(len(accs))]
		receiver := sdk.AccAddress(simAccount.Address)

		// Setup 3 relayers
		relayers := k.GetRelayers(ctx)
		var privs []*secp256k1.PrivKey
		relayers = nil
		for i := 0; i < 3; i++ {
			priv := secp256k1.GenPrivKey()
			privs = append(privs, priv)
			addr := sdk.AccAddress(priv.PubKey().Address()).String()
			rVal := Relayer{
				Address: addr,
				PubKey:  priv.PubKey().Bytes(),
			}
			k.SetRelayer(ctx, rVal)
			relayers = append(relayers, rVal)
		}

		params := k.GetParams(ctx)
		// Set amount larger than supply cap
		supplyCap, ok := math.NewIntFromString(params.SupplyCap)
		if !ok {
			supplyCap = math.NewInt(1000000)
		}
		capBreachingAmt := supplyCap.Add(math.NewInt(1000))
		amount := sdk.NewCoins(sdk.NewCoin("uwsov", capBreachingAmt))

		h := sha256.New()
		h.Write([]byte(receiver.String()))
		h.Write([]byte(amount.String()))
		h.Write([]byte("cap_breach"))
		nonce := h.Sum(nil)

		hash := ComputeBridgeMessageHash(receiver.String(), amount, nonce)
		var sigs [][]byte
		for i := 0; i < int(params.QuorumThreshold); i++ {
			sig, err := privs[i].Sign(hash)
			if err != nil {
				return simtypes.NoOpMsg(ModuleName, "MsgBridgeIn", err.Error()), nil, nil
			}
			sigs = append(sigs, sig)
		}

		msg := MsgBridgeIn{
			Submitter:  relayers[0].Address,
			Receiver:   receiver.String(),
			Amount:     amount,
			Nonce:      nonce,
			Signatures: sigs,
		}

		err := k.ProcessBridgeIn(ctx, msg)
		if err == nil {
			return simtypes.NewOperationMsg(&msg, false, "expected failure for supply cap breach"), nil, nil
		}

		return simtypes.NewOperationMsg(&msg, true, "supply cap breach correctly rejected"), nil, nil
	}
}
