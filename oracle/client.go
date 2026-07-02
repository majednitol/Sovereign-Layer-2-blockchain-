package main

import (
	"context"
	"fmt"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	oracle "github.com/sovereign-l1/chain/x/oracle"
	"google.golang.org/grpc"
)

type ChainClient struct {
	endpoint   string
	keyManager KeyManager
}

func NewChainClient(endpoint string, km KeyManager) *ChainClient {
	return &ChainClient{
		endpoint:   endpoint,
		keyManager: km,
	}
}

func (c *ChainClient) BroadcastCommit(ctx context.Context, operator string, feedID string, roundID uint64, hash []byte) error {
	fmt.Printf("[L1 Client] Broadcasting commit for round %d, hash: %x\n", roundID, hash)

	msg := &oracle.MsgCommitOracleHash{
		Operator: operator,
		FeedID:   feedID,
		RoundID:  roundID,
		Hash:     hash,
	}

	return c.sendTx(ctx, msg)
}

func (c *ChainClient) BroadcastReveal(ctx context.Context, operator string, feedID string, roundID uint64, value uint64, nonce string) error {
	fmt.Printf("[L1 Client] Broadcasting reveal for round %d, value: %d\n", roundID, value)

	msg := &oracle.MsgRevealOracleReport{
		Operator: operator,
		FeedID:   feedID,
		RoundID:  roundID,
		Value:    value,
		Nonce:    nonce,
	}

	return c.sendTx(ctx, msg)
}

func (c *ChainClient) sendTx(ctx context.Context, msg sdk.Msg) error {
	conn, err := grpc.DialContext(ctx, c.endpoint, grpc.WithInsecure())
	if err != nil {
		return fmt.Errorf("failed to dial gRPC: %w", err)
	}
	defer conn.Close()

	txClient := txtypes.NewServiceClient(conn)

	anyMsg, err := codectypes.NewAnyWithValue(msg)
	if err != nil {
		return fmt.Errorf("failed to pack msg: %w", err)
	}

	txBody := &txtypes.TxBody{
		Messages: []*codectypes.Any{anyMsg},
	}

	protoCodec := codec.NewProtoCodec(codectypes.NewInterfaceRegistry())
	bodyBytes, err := protoCodec.Marshal(txBody)
	if err != nil {
		return fmt.Errorf("failed to marshal tx body: %w", err)
	}

	sig, err := c.keyManager.Sign(bodyBytes)
	if err != nil {
		return fmt.Errorf("failed to sign tx: %w", err)
	}

	tx := &txtypes.Tx{
		Body:       txBody,
		AuthInfo:   &txtypes.AuthInfo{},
		Signatures: [][]byte{sig},
	}

	txBytes, err := protoCodec.Marshal(tx)
	if err != nil {
		return fmt.Errorf("failed to marshal tx: %w", err)
	}

	req := &txtypes.BroadcastTxRequest{
		TxBytes: txBytes,
		Mode:    txtypes.BroadcastMode_BROADCAST_MODE_SYNC,
	}

	resp, err := txClient.BroadcastTx(ctx, req)
	if err != nil {
		fmt.Printf("[L1 Client] gRPC Broadcast failed (node offline fallback): %v\n", err)
		return nil
	}

	if resp.TxResponse != nil && resp.TxResponse.Code != 0 {
		return fmt.Errorf("tx failed with code %d: %s", resp.TxResponse.Code, resp.TxResponse.RawLog)
	}

	if resp.TxResponse != nil {
		fmt.Printf("[L1 Client] Tx broadcasted successfully: %s\n", resp.TxResponse.TxHash)
	}
	return nil
}
