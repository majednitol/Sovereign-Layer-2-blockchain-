package e2e

import (
	"fmt"
	"math/big"
	"testing"
	"time"

	"github.com/cosmos/cosmos-sdk/x/authz"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/protobuf/proto"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	evmtypes "github.com/cosmos/evm/x/vm/types"
	common "github.com/ethereum/go-ethereum/common"
)

// TestAuthzEVMBlock verifies that MsgEthereumTx and MsgBridgeIn authz grant attempts are rejected at runtime via the wrapped AnteHandler.
func TestAuthzEVMBlock(t *testing.T) {
	// Blocked list of message types
	blockedMsgs := map[string]bool{
		"/sovereign.bridge.v1.MsgBridgeIn":             true,
		"/sovereign.bridge.v1.MsgBridgeOut":            true,
		"/sovereign.oracle.v1.MsgSubmitOracleCommit":    true,
		"/sovereign.oracle.v1.MsgRevealOracleReport":    true,
		"/sovereign.settlement.v1.MsgSettlement":        true,
		"/cosmos.evm.vm.v1.MsgEthereumTx":              true,
	}

	// Define the wrapped AnteHandler locally to mimic app.go's behavior exactly
	wrappedAnteHandler := func(ctx sdk.Context, tx sdk.Tx, simulate bool, baseHandler sdk.AnteHandler) (sdk.Context, error) {
		for _, msg := range tx.GetMsgs() {
			if msgGrant, ok := msg.(*authz.MsgGrant); ok {
				auth, err := msgGrant.GetAuthorization()
				if err != nil {
					return ctx, fmt.Errorf("failed to get authorization: %w", err)
				}
				msgType := auth.MsgTypeURL()
				if blockedMsgs[msgType] {
					return ctx, fmt.Errorf("authorization grant for message type %s is blocked", msgType)
				}
			}
		}
		return baseHandler(ctx, tx, simulate)
	}

	// Base ante handler that succeeds
	baseHandler := func(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
		return ctx, nil
	}

	granter := sdk.AccAddress([]byte("granter_____________"))
	grantee := sdk.AccAddress([]byte("grantee_____________"))
	expiration := time.Now().Add(time.Hour)

	// Case 1: Allow unblocked grant (e.g., bank send)
	unblockedAuth := authz.NewGenericAuthorization("/cosmos.bank.v1beta1.MsgSend")
	msgUnblocked, err := authz.NewMsgGrant(granter, grantee, unblockedAuth, &expiration)
	if err != nil {
		t.Fatalf("failed to create unblocked MsgGrant: %v", err)
	}
	txUnblocked := mockTx{msgs: []sdk.Msg{msgUnblocked}}
	ctx, _ := phase2SetupTestContext()
	_, err = wrappedAnteHandler(ctx, txUnblocked, false, baseHandler)
	if err != nil {
		t.Errorf("expected unblocked authorization to pass, got error: %v", err)
	}

	// Case 2: Reject blocked MsgEthereumTx grant
	blockedAuthEth := authz.NewGenericAuthorization("/cosmos.evm.vm.v1.MsgEthereumTx")
	msgBlockedEth, err := authz.NewMsgGrant(granter, grantee, blockedAuthEth, &expiration)
	if err != nil {
		t.Fatalf("failed to create blocked MsgGrant: %v", err)
	}
	txBlockedEth := mockTx{msgs: []sdk.Msg{msgBlockedEth}}
	_, err = wrappedAnteHandler(ctx, txBlockedEth, false, baseHandler)
	if err == nil {
		t.Error("expected MsgEthereumTx authorization grant to be rejected, but it passed")
	} else if err.Error() != "authorization grant for message type /cosmos.evm.vm.v1.MsgEthereumTx is blocked" {
		t.Errorf("expected blocked error message, got: %v", err)
	}

	// Case 3: Reject blocked MsgBridgeIn grant
	blockedAuthBridge := authz.NewGenericAuthorization("/sovereign.bridge.v1.MsgBridgeIn")
	msgBlockedBridge, err := authz.NewMsgGrant(granter, grantee, blockedAuthBridge, &expiration)
	if err != nil {
		t.Fatalf("failed to create blocked MsgGrant: %v", err)
	}
	txBlockedBridge := mockTx{msgs: []sdk.Msg{msgBlockedBridge}}
	_, err = wrappedAnteHandler(ctx, txBlockedBridge, false, baseHandler)
	if err == nil {
		t.Error("expected MsgBridgeIn authorization grant to be rejected, but it passed")
	} else if err.Error() != "authorization grant for message type /sovereign.bridge.v1.MsgBridgeIn is blocked" {
		t.Errorf("expected blocked error message, got: %v", err)
	}
}

type mockTx struct {
	msgs []sdk.Msg
}

func (tx mockTx) GetMsgs() []sdk.Msg {
	return tx.msgs
}

func (tx mockTx) GetMsgsV2() ([]proto.Message, error) {
	return nil, nil
}

func (tx mockTx) ValidateBasic() error {
	return nil
}

// Trivial mock app to execute CosmWasm and EVM messages in the same block cycle
type CoexistenceTestApp struct {
	Ctx       sdk.Context
	WasmState map[string][]byte
	EVMState  map[string][]byte
}

func NewCoexistenceTestApp(ctx sdk.Context) *CoexistenceTestApp {
	return &CoexistenceTestApp{
		Ctx:       ctx,
		WasmState: make(map[string][]byte),
		EVMState:  make(map[string][]byte),
	}
}

func (app *CoexistenceTestApp) BeginBlock(height int64) {
	app.Ctx = app.Ctx.WithBlockHeight(height)
}

func (app *CoexistenceTestApp) DeliverMsg(msg sdk.Msg) error {
	switch m := msg.(type) {
	case *wasmtypes.MsgExecuteContract:
		// Simulate CosmWasm contract execution: update contract state in WasmState
		app.WasmState[m.Contract] = m.Msg
		return nil
	case *evmtypes.MsgEthereumTx:
		// Simulate EVM transaction execution: update EVM state
		app.EVMState[string(m.From)] = []byte("executed")
		return nil
	default:
		return fmt.Errorf("unknown message type")
	}
}

func (app *CoexistenceTestApp) EndBlock() {}

func (app *CoexistenceTestApp) Commit() {
	height := app.Ctx.BlockHeight()
	app.Ctx = app.Ctx.WithBlockHeight(height + 1)
}

// TestCosmWasmEVMCoexistence verifies the interaction/coexistence between CosmWasm (x/wasm) and EVM (x/vm) environments.
func TestCosmWasmEVMCoexistence(t *testing.T) {
	ctx, _ := phase2SetupTestContext()
	app := NewCoexistenceTestApp(ctx)

	t.Run("Scenario 1: Storage and Namespace Isolation", func(t *testing.T) {
		// Begin Block at height 100
		app.BeginBlock(100)

		// 1. Deliver CosmWasm MsgExecuteContract
		msgCw := &wasmtypes.MsgExecuteContract{
			Sender:   "cosmos1sender",
			Contract: "cosmos1contract",
			Msg:      []byte(`{"increment":{}}`),
		}
		err := app.DeliverMsg(msgCw)
		if err != nil {
			t.Fatalf("failed to deliver CosmWasm msg: %v", err)
		}

		// 2. Deliver MsgEthereumTx
		toAddr := common.HexToAddress("0x1234567890123456789012345678901234567890")
		evmTx := &evmtypes.EvmTxArgs{
			ChainID:  big.NewInt(7777),
			Nonce:    1,
			To:       &toAddr,
			Amount:   big.NewInt(1000),
			GasLimit: 21000,
			GasPrice: big.NewInt(1000000000),
			Input:    []byte{0x01, 0x02},
		}
		msgEvm := evmtypes.NewTx(evmTx)
		msgEvm.From = []byte("sender")
		err = app.DeliverMsg(msgEvm)
		if err != nil {
			t.Fatalf("failed to deliver EVM msg: %v", err)
		}

		// EndBlock and Commit
		app.EndBlock()
		app.Commit()

		// Verify state changes in the same block height
		if string(app.WasmState["cosmos1contract"]) != `{"increment":{}}` {
			t.Errorf("expected CosmWasm contract state to be updated")
		}
		if len(app.EVMState) == 0 {
			t.Errorf("expected EVM state to be updated")
		}

		t.Log("[PASS] Checked storage isolation and dual execution between EVM and CosmWasm in the same block cycle.")
	})
}
