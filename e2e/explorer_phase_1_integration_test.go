package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
	"time"
)

func TestExplorerPhase1Integration(t *testing.T) {
	// Setup URLs from environment or defaults
	faucetURL := os.Getenv("TEST_FAUCET_URL")
	if faucetURL == "" {
		faucetURL = "http://127.0.0.1:8000"
	}
	explorerURL := os.Getenv("TEST_EXPLORER_API_URL")
	if explorerURL == "" {
		explorerURL = "http://127.0.0.1:8082"
	}

	// Ping faucet (a GET to /faucet returns 405 Method Not Allowed, but proves it's reachable)
	fresp, err := http.Get(faucetURL + "/faucet")
	if err != nil {
		t.Skipf("Skipping integration test: faucet service not reachable at %s: %v", faucetURL, err)
	}
	fresp.Body.Close()

	// Ping explorer api
	eresp, err := http.Get(explorerURL + "/api/rest/v1/explorer/blocks")
	if err != nil {
		t.Skipf("Skipping integration test: explorer API service not reachable at %s: %v", explorerURL, err)
	}
	eresp.Body.Close()

	t.Logf("Running Explorer Phase 1 Integration Test using Faucet: %s, Explorer API: %s", faucetURL, explorerURL)

	// Step 1: Send a faucet request to trigger a real transaction
	address := "sov1wskntnrxxnq9x2f95wuyf0z9fezr3azw46qht0" // test address
	reqBody, _ := json.Marshal(map[string]string{"address": address})
	postResp, err := http.Post(faucetURL+"/faucet", "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		t.Fatalf("Failed to request tokens from faucet: %v", err)
	}
	defer postResp.Body.Close()

	if postResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(postResp.Body)
		t.Fatalf("Faucet request failed with status %d: %s", postResp.StatusCode, string(body))
	}

	var faucetResult map[string]interface{}
	if err := json.NewDecoder(postResp.Body).Decode(&faucetResult); err != nil {
		t.Fatalf("Failed to decode faucet response: %v", err)
	}

	// Verify that faucet returns a non-empty tx_hash
	txHashRaw, ok := faucetResult["tx_hash"]
	if !ok || txHashRaw == "" {
		t.Logf("Faucet result: %v", faucetResult)
		txHashRaw, ok = faucetResult["hash"]
		if !ok || txHashRaw == "" {
			t.Fatalf("Faucet response did not include a valid transaction hash")
		}
	}
	txHash := fmt.Sprintf("%v", txHashRaw)
	t.Logf("Faucet transaction successfully sent. Tx Hash: %s", txHash)

	// Step 2: Poll explorer API blocks and txs endpoints until the tx is indexed
	t.Log("Waiting for the transaction to be indexed by explorer-indexer...")
	var indexedTx map[string]interface{}
	found := false

	// Retry up to 10 times with 3 seconds sleep (30s max)
	for i := 0; i < 10; i++ {
		time.Sleep(3 * time.Second)

		txResp, err := http.Get(fmt.Sprintf("%s/api/rest/v1/explorer/txs/%s", explorerURL, txHash))
		if err != nil {
			t.Logf("Attempt %d: failed to fetch tx details: %v", i+1, err)
			continue
		}
		defer txResp.Body.Close()

		if txResp.StatusCode == http.StatusOK {
			if err := json.NewDecoder(txResp.Body).Decode(&indexedTx); err == nil {
				found = true
				break
			}
		}
	}

	if !found {
		t.Fatalf("Transaction %s was not indexed by explorer-indexer within 30 seconds", txHash)
	}

	// Step 3: Validate the indexed transaction details
	t.Log("Validating indexed transaction fields...")
	
	// Status should be 0 (success)
	statusVal, ok := indexedTx["status"]
	if !ok {
		t.Error("Transaction detail response missing 'status' field")
	} else if fmt.Sprintf("%v", statusVal) != "0" {
		t.Errorf("Expected transaction status to be 0 (success), got %v", statusVal)
	}

	// Gas used should be greater than zero
	_, ok = indexedTx["gasUsed"]
	if !ok {
		t.Error("Transaction detail response missing 'gasUsed' field")
	}

	// Message type should be Cosmos bank send
	msgTypesVal, ok := indexedTx["msgTypes"]
	if !ok {
		t.Error("Transaction detail response missing 'msgTypes' field")
	} else {
		msgTypes, isSlice := msgTypesVal.([]interface{})
		if !isSlice || len(msgTypes) == 0 {
			t.Errorf("Expected non-empty msgTypes list, got: %v", msgTypesVal)
		} else {
			expectedMsgType := "/cosmos.bank.v1beta1.MsgSend"
			matched := false
			for _, m := range msgTypes {
				if fmt.Sprintf("%v", m) == expectedMsgType {
					matched = true
				}
			}
			if !matched {
				t.Errorf("Expected msgTypes to contain %s, got: %v", expectedMsgType, msgTypes)
			}
		}
	}

	// Step 4: Verify blocks endpoint
	t.Log("Validating blocks API list and pagination...")
	blocksResp, err := http.Get(explorerURL + "/api/rest/v1/explorer/blocks?limit=5")
	if err != nil {
		t.Fatalf("Failed to fetch blocks list: %v", err)
	}
	defer blocksResp.Body.Close()

	if blocksResp.StatusCode != http.StatusOK {
		t.Fatalf("Blocks API returned non-OK status: %d", blocksResp.StatusCode)
	}

	var blockList map[string]interface{}
	if err := json.NewDecoder(blocksResp.Body).Decode(&blockList); err != nil {
		t.Fatalf("Failed to decode block list response: %v", err)
	}

	blocksVal, ok := blockList["blocks"]
	if !ok {
		t.Fatalf("Blocks list response missing 'blocks' key")
	}
	blocksSlice, ok := blocksVal.([]interface{})
	if !ok || len(blocksSlice) == 0 {
		t.Fatalf("Expected at least one block to be indexed, got empty list")
	}

	t.Logf("Blocks list returned %d blocks. First block height: %v", len(blocksSlice), blocksSlice[0].(map[string]interface{})["height"])

	t.Log("[PASS] All Phase 1 Explorer integration checks completed successfully!")
}
