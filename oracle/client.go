package main

import (
	"context"
	"fmt"
	"sync"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdkclient "github.com/cosmos/cosmos-sdk/client/grpc/cmtservice"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	cryptosecp256k1 "github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	signing "github.com/cosmos/cosmos-sdk/types/tx/signing"
	gogoproto "github.com/cosmos/gogoproto/proto"
	oracle "github.com/sovereign-l1/chain/x/oracle"
	"google.golang.org/grpc"
)

type ChainClient struct {
	endpoint   string
	keyManager KeyManager
	chainID    string

	mu       sync.Mutex
	sequence uint64
	accNum   uint64
	seqInit  bool
}

func NewChainClient(endpoint string, km KeyManager, chainID string) *ChainClient {
	return &ChainClient{
		endpoint:   endpoint,
		keyManager: km,
		chainID:    chainID,
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

func (c *ChainClient) GetLatestBlockHeight(ctx context.Context) (int64, error) {
	conn, err := grpc.DialContext(ctx, c.endpoint, grpc.WithInsecure())
	if err != nil {
		return 0, fmt.Errorf("failed to dial gRPC for block height query: %w", err)
	}
	defer conn.Close()

	client := sdkclient.NewServiceClient(conn)
	resp, err := client.GetLatestBlock(ctx, &sdkclient.GetLatestBlockRequest{})
	if err != nil {
		return 0, fmt.Errorf("failed to query latest block: %w", err)
	}
	if resp.Block == nil {
		return 0, fmt.Errorf("invalid latest block response structure")
	}
	return resp.Block.Header.Height, nil
}

// initSequence fetches the current account number and sequence from the chain.
func (c *ChainClient) initSequence(ctx context.Context, address string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.seqInit {
		return nil
	}

	conn, err := grpc.DialContext(ctx, c.endpoint, grpc.WithInsecure())
	if err != nil {
		return fmt.Errorf("failed to dial gRPC for account query: %w", err)
	}
	defer conn.Close()

	authClient := authtypes.NewQueryClient(conn)
	resp, err := authClient.Account(ctx, &authtypes.QueryAccountRequest{Address: address})
	if err != nil {
		return fmt.Errorf("failed to query account %s: %w", address, err)
	}

	var baseAccount authtypes.BaseAccount
	if err := baseAccount.Unmarshal(resp.Account.Value); err != nil {
		return fmt.Errorf("failed to unmarshal account: %w", err)
	}

	c.accNum = baseAccount.AccountNumber
	c.sequence = baseAccount.Sequence
	c.seqInit = true
	fmt.Printf("[L1 Client] Initialized account: number=%d, sequence=%d\n", c.accNum, c.sequence)
	return nil
}

// getAndIncrementSequence returns the current sequence and increments it atomically.
func (c *ChainClient) getAndIncrementSequence() uint64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	seq := c.sequence
	c.sequence++
	return seq
}

func (c *ChainClient) sendTx(ctx context.Context, msg sdk.Msg) error {
	c.mu.Lock()
	if !c.seqInit {
		c.mu.Unlock()
		pubKeyBytes := c.keyManager.GetPublicKey()
		cosmosPubKey := &cryptosecp256k1.PubKey{Key: pubKeyBytes}
		accAddr := sdk.AccAddress(cosmosPubKey.Address().Bytes()).String()
		if err := c.initSequence(ctx, accAddr); err != nil {
			return fmt.Errorf("failed to auto-initialize sequence for %s: %w", accAddr, err)
		}
	} else {
		c.mu.Unlock()
	}

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

	interfaceRegistry := codectypes.NewInterfaceRegistry()
	cryptocodec.RegisterInterfaces(interfaceRegistry)
	interfaceRegistry.RegisterImplementations((*sdk.Msg)(nil), &oracle.MsgCommitOracleHash{}, &oracle.MsgRevealOracleReport{})

	protoCodec := codec.NewProtoCodec(interfaceRegistry)
	bodyBytes, err := protoCodec.Marshal(txBody)
	if err != nil {
		return fmt.Errorf("failed to marshal tx body: %w", err)
	}

	// Build proper AuthInfo with signer info and fee
	pubKeyBytes := c.keyManager.GetPublicKey()
	cosmosPubKey := &cryptosecp256k1.PubKey{Key: pubKeyBytes}
	anyPubKey, err := codectypes.NewAnyWithValue(cosmosPubKey)
	if err != nil {
		return fmt.Errorf("failed to pack public key: %w", err)
	}

	seq := c.getAndIncrementSequence()

	authInfo := &txtypes.AuthInfo{
		SignerInfos: []*txtypes.SignerInfo{
			{
				PublicKey: anyPubKey,
				ModeInfo: &txtypes.ModeInfo{
					Sum: &txtypes.ModeInfo_Single_{
						Single: &txtypes.ModeInfo_Single{
							Mode: signing.SignMode_SIGN_MODE_DIRECT,
						},
					},
				},
				Sequence: seq,
			},
		},
		Fee: &txtypes.Fee{
			Amount:   sdk.NewCoins(sdk.NewInt64Coin("aesov", 1000000000000000)),
			GasLimit: 200000,
		},
	}

	authInfoBytes, err := protoCodec.Marshal(authInfo)
	if err != nil {
		return fmt.Errorf("failed to marshal auth info: %w", err)
	}

	// Construct the correct SignDoc per Cosmos SDK spec
	signDoc := &txtypes.SignDoc{
		BodyBytes:     bodyBytes,
		AuthInfoBytes: authInfoBytes,
		ChainId:       c.chainID,
		AccountNumber: c.accNum,
	}
	signDocBytes, err := protoCodec.Marshal(signDoc)
	if err != nil {
		return fmt.Errorf("failed to marshal sign doc: %w", err)
	}

	sig, err := c.keyManager.Sign(signDocBytes)
	if err != nil {
		return fmt.Errorf("failed to sign tx: %w", err)
	}

	tx := &txtypes.Tx{
		Body:       txBody,
		AuthInfo:   authInfo,
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

	// C1 FIX: Return gRPC errors instead of swallowing them
	resp, err := txClient.BroadcastTx(ctx, req)
	if err != nil {
		return fmt.Errorf("gRPC broadcast failed: %w", err)
	}

	if resp.TxResponse != nil && resp.TxResponse.Code != 0 {
		// Sequence mismatch — reset and retry on next call
		if resp.TxResponse.Code == 32 { // sdkerrors.ErrWrongSequence
			c.mu.Lock()
			c.seqInit = false
			c.mu.Unlock()
		}
		return fmt.Errorf("tx failed with code %d: %s", resp.TxResponse.Code, resp.TxResponse.RawLog)
	}

	if resp.TxResponse != nil {
		fmt.Printf("[L1 Client] Tx broadcasted successfully: %s (seq=%d)\n", resp.TxResponse.TxHash, seq)
	}
	return nil
}

// GetOperatorAddress dynamically derives the validator operator address from the HSM key.
func (c *ChainClient) GetOperatorAddress() (string, error) {
	pubKeyBytes := c.keyManager.GetPublicKey()
	if len(pubKeyBytes) != 33 {
		return "", fmt.Errorf("invalid public key length: got %d, expected 33", len(pubKeyBytes))
	}
	cosmosPubKey := &cryptosecp256k1.PubKey{Key: pubKeyBytes}
	addr := sdk.ValAddress(cosmosPubKey.Address().Bytes())
	return addr.String(), nil
}

// GetActiveFeeds queries the chain for active feeds registered in x/milestone store.
func (c *ChainClient) GetActiveFeeds(ctx context.Context) ([]string, error) {
	conn, err := grpc.DialContext(ctx, c.endpoint, grpc.WithInsecure())
	if err != nil {
		return nil, fmt.Errorf("failed to dial gRPC: %w", err)
	}
	defer conn.Close()

	client := sdkclient.NewServiceClient(conn)
	req := &sdkclient.ABCIQueryRequest{
		Path: "/store/milestone/subspace",
		Data: []byte{0x03}, // ActiveFeedsKeyPrefix
	}

	resp, err := client.ABCIQuery(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("ABCI query failed: %w", err)
	}

	if resp.Code != 0 {
		return nil, fmt.Errorf("ABCI query returned error code %d: %s", resp.Code, resp.Log)
	}

	var pairs KVPairs
	if err := gogoproto.Unmarshal(resp.Value, &pairs); err != nil {
		return nil, fmt.Errorf("failed to unmarshal KVPairs: %w", err)
	}

	var feeds []string
	for _, pair := range pairs.Pairs {
		// Key format is prefix (0x03) + feedID
		if len(pair.Key) > 1 && pair.Key[0] == 0x03 {
			feedID := string(pair.Key[1:])
			feeds = append(feeds, feedID)
		}
	}

	return feeds, nil
}

type KVPair struct {
	Key   []byte `protobuf:"bytes,1,opt,name=key,proto3" json:"key,omitempty"`
	Value []byte `protobuf:"bytes,2,opt,name=value,proto3" json:"value,omitempty"`
}

func (m *KVPair) Reset()         { *m = KVPair{} }
func (m *KVPair) String() string { return gogoproto.CompactTextString(m) }
func (*KVPair) ProtoMessage()    {}

type KVPairs struct {
	Pairs []KVPair `protobuf:"bytes,1,rep,name=pairs,proto3" json:"pairs"`
}

func (m *KVPairs) Reset()         { *m = KVPairs{} }
func (m *KVPairs) String() string { return gogoproto.CompactTextString(m) }
func (*KVPairs) ProtoMessage()    {}
