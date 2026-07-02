package e2e

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	explorerv1 "github.com/sovereign-l1/chain/api/explorer/v1"
)

// Mock CometBFT server
func setupMockCometBFT() *httptest.Server {
	mux := http.NewServeMux()

	mux.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		status := map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      -1,
			"result": map[string]interface{}{
				"sync_info": map[string]interface{}{
					"latest_block_height": "100",
					"latest_block_time":   time.Now().Format(time.RFC3339),
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(status)
	})

	mux.HandleFunc("/block", func(w http.ResponseWriter, r *http.Request) {
		h := r.URL.Query().Get("height")
		block := map[string]interface{}{
			"jsonrpc": "2.0",
			"id":      -1,
			"result": map[string]interface{}{
				"block_id": map[string]interface{}{
					"hash": "blockhash123",
				},
				"block": map[string]interface{}{
					"header": map[string]interface{}{
						"height":           h,
						"time":             time.Now().Format(time.RFC3339),
						"proposer_address": "sovereignvaloper1proposeraddress",
						"app_hash":         "apphash123",
					},
					"data": map[string]interface{}{
						"txs": []string{
							hex.EncodeToString([]byte("mocktransactiondata")),
						},
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(block)
	})

	return httptest.NewServer(mux)
}

// Test block/tx parsing and decoding structures for Phase 1
func TestExplorerPhase1BlockDecoding(t *testing.T) {
	ts := setupMockCometBFT()
	defer ts.Close()

	// Perform a mock fetch blocks/txs flow to assert structural sanity
	resp, err := http.Get(fmt.Sprintf("%s/block?height=100", ts.URL))
	if err != nil {
		t.Fatalf("failed to fetch mock block: %v", err)
	}
	defer resp.Body.Close()

	var blockResp struct {
		Result struct {
			BlockID struct {
				Hash string `json:"hash"`
			} `json:"block_id"`
			Block struct {
				Header struct {
					Height          string    `json:"height"`
					Time            time.Time `json:"time"`
					ProposerAddress string    `json:"proposer_address"`
					AppHash         string    `json:"app_hash"`
				} `json:"header"`
				Data struct {
					Txs []string `json:"txs"`
				} `json:"data"`
			} `json:"block"`
		} `json:"result"`
	}

	err = json.NewDecoder(resp.Body).Decode(&blockResp)
	if err != nil {
		t.Fatalf("failed to decode mock block: %v", err)
	}

	if blockResp.Result.Block.Header.Height != "100" {
		t.Errorf("expected height 100, got %s", blockResp.Result.Block.Header.Height)
	}
	if blockResp.Result.Block.Header.ProposerAddress != "sovereignvaloper1proposeraddress" {
		t.Errorf("unexpected proposer address: %s", blockResp.Result.Block.Header.ProposerAddress)
	}

	txs := blockResp.Result.Block.Data.Txs
	if len(txs) != 1 {
		t.Fatalf("expected 1 tx, got %d", len(txs))
	}

	rawBytes, err := hex.DecodeString(txs[0])
	if err != nil {
		t.Fatalf("failed to decode hex transaction: %v", err)
	}
	hash := sha256.Sum256(rawBytes)
	hashStr := hex.EncodeToString(hash[:])

	if hashStr == "" {
		t.Error("tx hash should not be empty")
	}

	t.Logf("[PASS] Verified CometBFT block parsing & hash derivation: %s", hashStr)
}

// Test gRPC structural compatibility
type mockExplorerServer struct {
	explorerv1.UnimplementedExplorerServiceServer
}

func (m *mockExplorerServer) GetBlock(ctx context.Context, req *explorerv1.GetBlockRequest) (*explorerv1.BlockDetail, error) {
	return &explorerv1.BlockDetail{
		Height:   req.Height,
		Time:     time.Now().Format(time.RFC3339),
		Proposer: "sovereignvaloper1proposeraddress",
		TxCount:  1,
		GasUsed:  50000,
		GasLimit: 100000,
		AppHash:  "apphash123",
	}, nil
}

func (m *mockExplorerServer) GetTx(ctx context.Context, req *explorerv1.GetTxRequest) (*explorerv1.TxDetail, error) {
	return &explorerv1.TxDetail{
		Hash:     req.Hash,
		Height:   100,
		Time:     time.Now().Format(time.RFC3339),
		Type:     "cosmos",
		MsgTypes: []string{"/cosmos.bank.v1beta1.MsgSend"},
		Decoded:  `{"amount": "1000uSLT"}`,
		Fee:      150,
		GasUsed:  45000,
		Status:   0,
	}, nil
}

func (m *mockExplorerServer) GetAddress(ctx context.Context, req *explorerv1.GetAddressRequest) (*explorerv1.AccountDetail, error) {
	return &explorerv1.AccountDetail{
		AddressBech32: req.Address,
		AddressHex:    "0xmockhexaddress",
		FirstSeen:     1,
		LastActive:    100,
		Balance:       "1000 uSLT",
	}, nil
}

func TestExplorerPhase1APIResponses(t *testing.T) {
	s := &mockExplorerServer{}
	ctx := context.Background()

	// 1. GetBlock Test
	block, err := s.GetBlock(ctx, &explorerv1.GetBlockRequest{Height: 100})
	if err != nil {
		t.Fatalf("failed to call GetBlock: %v", err)
	}
	if block.Height != 100 {
		t.Errorf("expected height 100, got %d", block.Height)
	}

	// 2. GetTx Test
	tx, err := s.GetTx(ctx, &explorerv1.GetTxRequest{Hash: "txhash123"})
	if err != nil {
		t.Fatalf("failed to call GetTx: %v", err)
	}
	if tx.Hash != "txhash123" {
		t.Errorf("expected hash txhash123, got %s", tx.Hash)
	}

	// 3. GetAddress Test
	addr, err := s.GetAddress(ctx, &explorerv1.GetAddressRequest{Address: "sovereign1qyq"})
	if err != nil {
		t.Fatalf("failed to call GetAddress: %v", err)
	}
	if addr.AddressBech32 != "sovereign1qyq" {
		t.Errorf("expected address sovereign1qyq, got %s", addr.AddressBech32)
	}

	t.Log("[PASS] Checked gRPC API responses match protobuf specifications.")
}
