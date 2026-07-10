package main

import (
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/bech32"
)

type FaucetRequest struct {
	Address string `json:"address"`
}

type FaucetResponse struct {
	TxHash  string `json:"tx_hash,omitempty"`
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

var (
	nodeURL      string
	keyName      string
	denom        string
	faucetAmount string
	listenAddr   string
	chainID      string

	// txMu serializes all faucet transactions to prevent concurrent
	// sequence number collisions ("tx already seen" mempool errors).
	txMu sync.Mutex

	// addressCooldowns tracks the last faucet send time per address
	// to enforce a minimum interval between requests.
	addressCooldowns   = make(map[string]time.Time)
	addressCooldownsMu sync.Mutex
	cooldownDuration   = 10 * time.Second
)

func main() {
	flag.StringVar(&nodeURL, "node", os.Getenv("NODE_URL"), "CometBFT RPC endpoint")
	flag.StringVar(&keyName, "key", os.Getenv("FAUCET_KEY"), "Name of key in keyring")
	flag.StringVar(&denom, "denom", os.Getenv("DENOM"), "Denomination of token")
	flag.StringVar(&faucetAmount, "amount", os.Getenv("FAUCET_AMOUNT"), "Amount to transfer (e.g. 10000000)")
	flag.StringVar(&listenAddr, "listen", ":8000", "Listen address")
	flag.StringVar(&chainID, "chain-id", os.Getenv("CHAIN_ID"), "Chain ID")
	flag.Parse()

	if nodeURL == "" {
		nodeURL = "http://localhost:26657"
	}
	if keyName == "" {
		keyName = "faucet"
	}
	if denom == "" {
		denom = "ucsov"
	}
	if faucetAmount == "" {
		faucetAmount = "10000000"
	}
	if chainID == "" {
		chainID = "sovereign-1"
	}

	http.HandleFunc("/faucet", handleFaucet)
	log.Printf("Faucet daemon listening on %s", listenAddr)
	if err := http.ListenAndServe(listenAddr, nil); err != nil {
		log.Fatalf("Faucet daemon server error: %v", err)
	}
}

func handleFaucet(w http.ResponseWriter, r *http.Request) {
	// CORS headers — allow browser requests from any origin (devnet only)
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Content-Type", "application/json")

	// Handle CORS preflight
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(FaucetResponse{Success: false, Error: "Only POST allowed"})
		return
	}

	var req FaucetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(FaucetResponse{Success: false, Error: "Invalid JSON"})
		return
	}

	address, err := normalizeAddress(req.Address)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(FaucetResponse{Success: false, Error: fmt.Sprintf("Invalid address: %v", err)})
		return
	}

	// Per-address cooldown to prevent rapid-fire requests
	addressCooldownsMu.Lock()
	if lastSend, ok := addressCooldowns[address]; ok {
		if time.Since(lastSend) < cooldownDuration {
			remaining := cooldownDuration - time.Since(lastSend)
			addressCooldownsMu.Unlock()
			w.WriteHeader(http.StatusTooManyRequests)
			json.NewEncoder(w).Encode(FaucetResponse{
				Success: false,
				Error:   fmt.Sprintf("Please wait %d seconds before requesting again", int(remaining.Seconds())+1),
			})
			return
		}
	}
	addressCooldownsMu.Unlock()

	amountStr := faucetAmount + denom
	log.Printf("Executing faucet transfer of %s to %s", amountStr, address)

	// Resolve chain home directory for keyring access
	chainHome := os.Getenv("CHAIN_HOME")
	if chainHome == "" {
		chainHome = "/root/.sovereign"
	}

	// Serialize all faucet transactions to prevent concurrent sequence
	// number collisions that cause "tx already seen" mempool errors.
	txMu.Lock()
	defer txMu.Unlock()

	// Query the faucet account's current sequence number from the node
	// to avoid stale sequence reuse.
	seq, seqErr := getAccountSequence(chainHome)

	// Build the send command with explicit sequence if available.
	cmdArgs := []string{
		"tx", "bank", "send",
		keyName, address, amountStr,
		"--node", nodeURL,
		"--keyring-backend", "test",
		"--chain-id", chainID,
		"--home", chainHome,
		"--yes",
		"--broadcast-mode", "sync",
		"--gas", "auto",
		"--gas-adjustment", "1.5",
		"--gas-prices", "1000000000aesov",
		"--output", "json",
	}
	if seqErr == nil {
		cmdArgs = append(cmdArgs, "--sequence", strconv.FormatUint(seq, 10))
	}

	var output []byte
	var txResult struct {
		TxHash string `json:"txhash"`
		Code   uint32 `json:"code"`
		RawLog string `json:"raw_log"`
	}

	// Retry loop: if we hit "tx already seen" or sequence mismatch,
	// wait briefly for the previous tx to commit and retry with the
	// updated sequence.
	maxRetries := 3
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			log.Printf("Faucet tx retry attempt %d/%d for %s", attempt, maxRetries, address)
			time.Sleep(2 * time.Second)
			// Re-query sequence for the retry
			newSeq, err := getAccountSequence(chainHome)
			if err == nil {
				// Update --sequence in cmdArgs
				updated := false
				for i, arg := range cmdArgs {
					if arg == "--sequence" && i+1 < len(cmdArgs) {
						cmdArgs[i+1] = strconv.FormatUint(newSeq, 10)
						updated = true
						break
					}
				}
				if !updated {
					cmdArgs = append(cmdArgs, "--sequence", strconv.FormatUint(newSeq, 10))
				}
			}
		}

		cmd := exec.Command("chaind", cmdArgs...)
		output, err = cmd.CombinedOutput()
		if err != nil {
			outStr := strings.TrimSpace(string(output))
			// Retry on "tx already seen" or "account sequence mismatch"
			if attempt < maxRetries && (strings.Contains(outStr, "tx already seen") || strings.Contains(outStr, "account sequence mismatch")) {
				log.Printf("Retryable tx error for %s: %s", address, outStr)
				continue
			}
			log.Printf("chaind send command failed: %v, output: %s", err, outStr)
			w.WriteHeader(http.StatusInternalServerError)
			errMsg := fmt.Sprintf("Tx broadcast failed: %v", err)
			if len(output) > 0 {
				errMsg = fmt.Sprintf("Tx broadcast failed: %s", outStr)
			}
			json.NewEncoder(w).Encode(FaucetResponse{Success: false, Error: errMsg})
			return
		}
		// Command succeeded, break out of retry loop
		break
	}

	// Parse JSON output from Cosmos SDK tx command to extract transaction hash
	jsonStr := string(output)
	if idx := strings.Index(jsonStr, "{"); idx != -1 {
		jsonStr = jsonStr[idx:]
	}

	if err := json.Unmarshal([]byte(jsonStr), &txResult); err != nil {
		log.Printf("Failed to unmarshal command output: %s", string(output))
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(FaucetResponse{Success: true, TxHash: "unknown_broadcast_success"})
		return
	}

	if txResult.Code != 0 {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(FaucetResponse{Success: false, Error: txResult.RawLog})
		return
	}

	// Record successful send time for cooldown
	addressCooldownsMu.Lock()
	addressCooldowns[address] = time.Now()
	addressCooldownsMu.Unlock()

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(FaucetResponse{Success: true, TxHash: txResult.TxHash})
}

// getAccountSequence queries the faucet account's current sequence number
// from the chain node so we can pass --sequence explicitly and avoid
// stale nonce collisions.
func getAccountSequence(chainHome string) (uint64, error) {
	cmd := exec.Command("chaind", "query", "auth", "account", keyName,
		"--node", nodeURL,
		"--home", chainHome,
		"--output", "json",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return 0, fmt.Errorf("failed to query account: %v", err)
	}

	// Extract sequence from JSON output
	jsonStr := string(out)
	if idx := strings.Index(jsonStr, "{"); idx != -1 {
		jsonStr = jsonStr[idx:]
	}

	var result struct {
		Account struct {
			Sequence string `json:"sequence"`
		} `json:"account"`
		// Also try flat structure
		Sequence string `json:"sequence"`
	}
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return 0, fmt.Errorf("failed to parse account info: %v", err)
	}

	seqStr := result.Account.Sequence
	if seqStr == "" {
		seqStr = result.Sequence
	}
	if seqStr == "" {
		return 0, fmt.Errorf("sequence not found in account response")
	}

	seq, err := strconv.ParseUint(seqStr, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid sequence value %q: %v", seqStr, err)
	}
	return seq, nil
}

func normalizeAddress(input string) (string, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return "", fmt.Errorf("empty address")
	}

	// 1. Check if it's a hex address (0x...)
	if strings.HasPrefix(input, "0x") {
		hexStr := strings.TrimPrefix(input, "0x")
		bytes, err := hex.DecodeString(hexStr)
		if err != nil || len(bytes) != 20 {
			return "", fmt.Errorf("invalid hex address: %v", err)
		}
		return sdk.AccAddress(bytes).String(), nil
	}

	// 2. Try parsing as Bech32 (cosmos or sov)
	hrp, bytes, err := bech32.DecodeAndConvert(input)
	if err != nil {
		// Maybe it's raw hex without 0x prefix
		bytes, errHex := hex.DecodeString(input)
		if errHex == nil && len(bytes) == 20 {
			return sdk.AccAddress(bytes).String(), nil
		}
		return "", fmt.Errorf("invalid address format: not a valid Bech32 or hex address")
	}

	if hrp != "cosmos" && hrp != "sov" && hrp != "sovereign" {
		return "", fmt.Errorf("unsupported address prefix: %s", hrp)
	}

	return sdk.AccAddress(bytes).String(), nil
}
