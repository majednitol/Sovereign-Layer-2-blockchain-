package e2e

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
)

// LogEntry represents an Ethereum transaction log entry returned by cast receipt
type LogEntry struct {
	Address string   `json:"address"`
	Topics  []string `json:"topics"`
	Data    string   `json:"data"`
}

// TxReceipt represents a transaction receipt returned by cast receipt --json
type TxReceipt struct {
	Status          string     `json:"status"`
	Logs            []LogEntry `json:"logs"`
	TransactionHash string     `json:"transactionHash"`
}

func runCommand(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("failed to run %s %v: %v\nstderr: %s", name, args, err, stderr.String())
	}
	return strings.TrimSpace(stdout.String()), nil
}

func TestBSCTestnetE2ELockBox(t *testing.T) {
	// 1. Check if required environment variables are set
	privKey := os.Getenv("BSC_TESTNET_PRIVATE_KEY")
	lockBoxAddr := os.Getenv("BSC_LOCKBOX_ADDRESS")
	erc20Addr := os.Getenv("BSC_ERC20_ADDRESS")
	rpcURL := os.Getenv("BSC_TESTNET_RPC_URL")

	if rpcURL == "" {
		rpcURL = "https://bsc-testnet-rpc.publicnode.com"
	}

	if privKey == "" || lockBoxAddr == "" || erc20Addr == "" {
		t.Skip("Skipping live BSC testnet E2E test. Set BSC_TESTNET_PRIVATE_KEY, BSC_LOCKBOX_ADDRESS, and BSC_ERC20_ADDRESS to run.")
	}

	// Resolve wallet address from private key
	walletAddr, err := runCommand("cast", "wallet", "address", "--private-key", privKey)
	if err != nil {
		t.Fatalf("Failed to resolve wallet address: %v", err)
	}
	t.Logf("Running testnet verification with wallet: %s", walletAddr)

	// Step 1: Check initial ERC20 balance
	balanceStr, err := runCommand("cast", "call", erc20Addr, "balanceOf(address)(uint256)", walletAddr, "--rpc-url", rpcURL)
	if err != nil {
		t.Fatalf("Failed to query token balance: %v", err)
	}
	t.Logf("Initial ERC20 balance: %s", balanceStr)

	// Step 2: Approve LockBox contract to spend tokens
	approveAmt := "1000000000000000000000000" // 1,000,000 tokens (18 decimals)
	t.Log("Sending ERC20 approve transaction...")
	approveTx, err := runCommand("cast", "send", erc20Addr, "approve(address,uint256)(bool)", lockBoxAddr, approveAmt, "--private-key", privKey, "--rpc-url", rpcURL, "--legacy")
	if err != nil {
		t.Fatalf("Failed to approve tokens: %v", err)
	}
	t.Logf("Approve TX successful. Receipt summary: %s", strings.Split(approveTx, "\n")[0])

	// Step 3: Lock 1 ERC20 token in LockBox
	lockAmt := "1000000000000000000" // 1 token (18 decimals)
	recipient := "cosmos1recipientaddress"
	t.Log("Sending LockBox lock transaction...")
	lockTxOutput, err := runCommand("cast", "send", lockBoxAddr, "lock(uint256,string)", lockAmt, recipient, "--private-key", privKey, "--rpc-url", rpcURL, "--legacy")
	if err != nil {
		t.Fatalf("Failed to lock tokens: %v", err)
	}

	// Extract transaction hash from output
	txHash := ""
	for _, line := range strings.Split(lockTxOutput, "\n") {
		if strings.HasPrefix(line, "transactionHash") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				txHash = parts[1]
			}
		}
	}
	if txHash == "" {
		t.Fatalf("Could not extract transaction hash from lock output:\n%s", lockTxOutput)
	}
	t.Logf("Lock transaction broadcast successfully: %s", txHash)

	// Step 4: Retrieve transaction receipt and parse nonce from logs
	receiptJSON, err := runCommand("cast", "receipt", txHash, "--json", "--rpc-url", rpcURL)
	if err != nil {
		t.Fatalf("Failed to retrieve transaction receipt: %v", err)
	}

	var receipt TxReceipt
	if err := json.Unmarshal([]byte(receiptJSON), &receipt); err != nil {
		t.Fatalf("Failed to parse receipt JSON: %v", err)
	}

	// Find the Locked event log
	// Event signature hash: keccak256("Locked(address,uint256,string,uint256)")
	lockedEventTopic := "0xc3a90879daa4563778b9d284a6e6548021dbe1516dfb66d972958cf1c08a2cc1"
	var lockedLog *LogEntry
	for _, log := range receipt.Logs {
		if len(log.Topics) > 0 && strings.ToLower(log.Topics[0]) == lockedEventTopic {
			lockedLog = &log
			break
		}
	}
	if lockedLog == nil {
		t.Fatal("Locked event log not found in receipt")
	}

	// Locked event unindexed fields in data: amount (uint256), offset (uint256), nonce (uint256), length (uint256), cosmosRecipient (string)
	// Nonce is stored at offset 64 bytes (third 32-byte word in data)
	dataHex := strings.TrimPrefix(lockedLog.Data, "0x")
	if len(dataHex) < 192 {
		t.Fatalf("Unexpected event log data length: %d", len(dataHex))
	}
	nonceHex := "0x" + dataHex[128:192]
	t.Logf("Extracted lock nonce: %s", nonceHex)

	// Step 5: Compute inner and outer messages, then sign with relayer key
	// ABI packed encoding of (walletAddr, lockAmt, nonceHex)
	// walletAddr = 20 bytes, lockAmt = 32 bytes, nonceHex = 32 bytes
	walletAddrClean := strings.TrimPrefix(walletAddr, "0x")
	lockAmtHex := fmt.Sprintf("%064x", 1000000000000000000)
	nonceHexClean := strings.TrimPrefix(nonceHex, "0x")

	packedPayload := walletAddrClean + lockAmtHex + nonceHexClean
	innerHash, err := runCommand("cast", "keccak", "0x"+packedPayload)
	if err != nil {
		t.Fatalf("Failed to compute keccak of payload: %v", err)
	}

	// Sign message hash using cast wallet sign
	sig, err := runCommand("cast", "wallet", "sign", innerHash, "--private-key", privKey)
	if err != nil {
		t.Fatalf("Failed to sign message hash: %v", err)
	}
	t.Logf("Generated signature: %s", sig)

	// Step 6: Verify signature recovery on-chain
	// Build prefixed Ethereum signed message hash: keccak256("\x19Ethereum Signed Message:\n32" + innerHash)
	prefixHex := hex.EncodeToString([]byte("\x19Ethereum Signed Message:\n32"))
	innerHashClean := strings.TrimPrefix(innerHash, "0x")
	prefixedPayload := prefixHex + innerHashClean
	prefixedHash, err := runCommand("cast", "keccak", "0x"+prefixedPayload)
	if err != nil {
		t.Fatalf("Failed to compute prefixed keccak: %v", err)
	}

	recovered, err := runCommand("cast", "call", lockBoxAddr, "recoverSigner(bytes32,bytes)(address)", prefixedHash, sig, "--rpc-url", rpcURL)
	if err != nil {
		t.Fatalf("recoverSigner call failed: %v", err)
	}
	if strings.ToLower(recovered) != strings.ToLower(walletAddr) {
		t.Fatalf("Recovered signer mismatch: got %s, expected %s", recovered, walletAddr)
	}
	t.Log("On-chain signature recovery successfully verified.")

	// Step 7: Unlock tokens on BSC Testnet
	signaturesArg := fmt.Sprintf("[%s]", sig)
	t.Log("Sending LockBox unlock transaction...")
	unlockTx, err := runCommand("cast", "send", lockBoxAddr, "unlock(address,uint256,uint256,bytes[])", walletAddr, lockAmt, nonceHex, signaturesArg, "--private-key", privKey, "--rpc-url", rpcURL, "--legacy")
	if err != nil {
		t.Fatalf("Failed to unlock tokens: %v", err)
	}
	t.Logf("Unlock transaction successful: %s", strings.Split(unlockTx, "\n")[0])

	// Step 8: Assert balance returned to normal
	finalBalanceStr, err := runCommand("cast", "call", erc20Addr, "balanceOf(address)(uint256)", walletAddr, "--rpc-url", rpcURL)
	if err != nil {
		t.Fatalf("Failed to query final balance: %v", err)
	}
	t.Logf("Final ERC20 balance: %s", finalBalanceStr)
	if finalBalanceStr != balanceStr {
		t.Errorf("Expected final balance to return to initial balance %s, got %s", balanceStr, finalBalanceStr)
	}

	// Step 9: Verify replay protection
	t.Log("Verifying replay protection (attempting second unlock)...")
	_, err = runCommand("cast", "send", lockBoxAddr, "unlock(address,uint256,uint256,bytes[])", walletAddr, lockAmt, nonceHex, signaturesArg, "--private-key", privKey, "--rpc-url", rpcURL, "--legacy")
	if err == nil {
		t.Error("Replay transaction expected to revert, but succeeded")
	} else {
		t.Log("Replay transaction successfully rejected (reverted).")
	}

	// Step 10: Verify pause and unpause
	t.Log("Testing circuit breaker pause...")
	_, err = runCommand("cast", "send", lockBoxAddr, "pause()", "--private-key", privKey, "--rpc-url", rpcURL, "--legacy")
	if err != nil {
		t.Fatalf("Failed to pause bridge: %v", err)
	}

	isPaused, err := runCommand("cast", "call", lockBoxAddr, "paused()(bool)", "--rpc-url", rpcURL)
	if err != nil || isPaused != "true" {
		t.Fatalf("Bridge failed to transition to paused state: %v, paused=%s", err, isPaused)
	}

	t.Log("Verifying lock reverts when paused...")
	_, err = runCommand("cast", "send", lockBoxAddr, "lock(uint256,string)", lockAmt, recipient, "--private-key", privKey, "--rpc-url", rpcURL, "--legacy")
	if err == nil {
		t.Error("Lock operation expected to revert when paused, but succeeded")
	} else {
		t.Log("Lock operation successfully reverted while paused.")
	}

	t.Log("Testing unpause...")
	_, err = runCommand("cast", "send", lockBoxAddr, "unpause()", "--private-key", privKey, "--rpc-url", rpcURL, "--legacy")
	if err != nil {
		t.Fatalf("Failed to unpause bridge: %v", err)
	}

	isPaused, err = runCommand("cast", "call", lockBoxAddr, "paused()(bool)", "--rpc-url", rpcURL)
	if err != nil || isPaused != "false" {
		t.Fatalf("Bridge failed to transition to unpaused state: %v, paused=%s", err, isPaused)
	}

	t.Log("E2E Testnet verification completed successfully!")
}
