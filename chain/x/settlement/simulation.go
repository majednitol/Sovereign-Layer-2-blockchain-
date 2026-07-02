package settlement

import (
	"crypto/ed25519"
	"math/rand"
	"sync"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/baseapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
)

var (
	simPubKey  ed25519.PublicKey
	simPrivKey ed25519.PrivateKey
	simKeyOnce sync.Once
)

func getSimKeys() (ed25519.PublicKey, ed25519.PrivateKey) {
	simKeyOnce.Do(func() {
		pub, priv, err := ed25519.GenerateKey(nil)
		if err != nil {
			panic(err)
		}
		simPubKey = pub
		simPrivKey = priv
	})
	return simPubKey, simPrivKey
}

// SimulateMsgSettlement simulates settlement requests.
func SimulateMsgSettlement(k Keeper) simtypes.Operation {
	return func(
		r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context,
		accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		if len(accs) == 0 {
			return simtypes.NoOpMsg(ModuleName, "MsgSettlement", "no accounts available"), nil, nil
		}
		simAccount := accs[r.Intn(len(accs))]
		submitter := sdk.AccAddress(simAccount.Address)

		// Get pre-cached keypair for speed in simulation tests
		pubKey, privKey := getSimKeys()

		witnessID := "witness_1"
		k.SetWitnessPubKey(ctx, witnessID, pubKey)

		payloadHash := []byte("mock_payload_hash_value___________")
		domainSeparator := ComputeDomainSeparator(chainID, payloadHash)
		signature := ed25519.Sign(privKey, domainSeparator)

		msg := MsgSettlement{
			Submitter:    submitter.String(),
			WitnessID:    witnessID,
			Timestamp:    ctx.BlockTime().Unix(),
			PayloadHash:  payloadHash,
			Signature:    signature,
			TransferAmt:  sdk.NewCoins(sdk.NewCoin("usov", math.NewInt(5000))),
			TransferDest: submitter.String(),
		}

		// Run execution check
		err := k.ProcessSettlement(ctx, msg)
		if err != nil {
			return simtypes.NoOpMsg(ModuleName, "MsgSettlement", err.Error()), nil, nil
		}

		return simtypes.NewOperationMsg(&msg, true, ""), nil, nil
	}
}

// SimulateMsgInvalidWitnessSettlement simulates a settlement request with a bad signature.
func SimulateMsgInvalidWitnessSettlement(k Keeper) simtypes.Operation {
	return func(
		r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context,
		accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		if len(accs) == 0 {
			return simtypes.NoOpMsg(ModuleName, "MsgSettlement", "no accounts available"), nil, nil
		}
		simAccount := accs[r.Intn(len(accs))]
		submitter := sdk.AccAddress(simAccount.Address)

		pubKey, _ := getSimKeys()
		witnessID := "witness_1"
		k.SetWitnessPubKey(ctx, witnessID, pubKey)

		payloadHash := []byte("mock_payload_hash_value___________")
		// Incorrect signature
		signature := []byte("invalid_signature_mock_bytes_here")

		msg := MsgSettlement{
			Submitter:    submitter.String(),
			WitnessID:    witnessID,
			Timestamp:    ctx.BlockTime().Unix(),
			PayloadHash:  payloadHash,
			Signature:    signature,
			TransferAmt:  sdk.NewCoins(sdk.NewCoin("usov", math.NewInt(5000))),
			TransferDest: submitter.String(),
		}

		err := k.ProcessSettlement(ctx, msg)
		if err == nil {
			return simtypes.NewOperationMsg(&msg, false, "expected failure for invalid signature"), nil, nil
		}

		return simtypes.NewOperationMsg(&msg, true, "invalid witness settlement simulation passed"), nil, nil
	}
}

// SimulateMsgExpiredTimestampSettlement simulates a settlement request outside of timestamp tolerance window.
func SimulateMsgExpiredTimestampSettlement(k Keeper) simtypes.Operation {
	return func(
		r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context,
		accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		if len(accs) == 0 {
			return simtypes.NoOpMsg(ModuleName, "MsgSettlement", "no accounts available"), nil, nil
		}
		simAccount := accs[r.Intn(len(accs))]
		submitter := sdk.AccAddress(simAccount.Address)

		pubKey, privKey := getSimKeys()
		witnessID := "witness_1"
		k.SetWitnessPubKey(ctx, witnessID, pubKey)

		payloadHash := []byte("mock_payload_hash_value___________")
		domainSeparator := ComputeDomainSeparator(chainID, payloadHash)
		signature := ed25519.Sign(privKey, domainSeparator)

		// Set timestamp out of bounds (+100s when tolerance is 30s)
		params := k.GetParams(ctx)
		expiredTimestamp := ctx.BlockTime().Unix() + params.TimestampToleranceSeconds + 100

		msg := MsgSettlement{
			Submitter:    submitter.String(),
			WitnessID:    witnessID,
			Timestamp:    expiredTimestamp,
			PayloadHash:  payloadHash,
			Signature:    signature,
			TransferAmt:  sdk.NewCoins(sdk.NewCoin("usov", math.NewInt(5000))),
			TransferDest: submitter.String(),
		}

		err := k.ProcessSettlement(ctx, msg)
		if err == nil {
			return simtypes.NewOperationMsg(&msg, false, "expected failure for expired timestamp"), nil, nil
		}

		return simtypes.NewOperationMsg(&msg, true, "expired timestamp settlement simulation passed"), nil, nil
	}
}

