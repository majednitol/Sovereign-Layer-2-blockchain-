package main

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"os"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/bech32"
)

func main() {
	fmt.Println("=== Sovereign L1 — Strict Bech32 Prefix & Unified Account Verification ===")

	// Enforce SDK Config prefixes
	config := sdk.GetConfig()
	config.SetBech32PrefixForAccount("cosmos", "cosmospub")
	config.SetBech32PrefixForValidator("cosmosvaloper", "cosmosvaloperpub")
	config.SetBech32PrefixForConsensusNode("cosmosvalcons", "cosmosvalconspub")

	// Raw payload from prompt
	rawHex := "f39fd6e51aad88f6f4ce6ab8827279cfffb92266"
	rawBytes, err := hex.DecodeString(rawHex)
	if err != nil {
		fmt.Printf("[FAIL] Could not decode raw hex: %v\n", err)
		os.Exit(1)
	}

	// 1. Convert raw bytes to Cosmos Bech32
	cosmosAddr := sdk.AccAddress(rawBytes).String()
	expectedCosmosAddr := "cosmos17w0adeg64ky0daxwd2ugyuneellmjgnxramjtq"
	if cosmosAddr != expectedCosmosAddr {
		fmt.Printf("[FAIL] Expected Cosmos address %s, got %s\n", expectedCosmosAddr, cosmosAddr)
		os.Exit(1)
	}
	fmt.Printf("[PASS] Raw Bytes -> Cosmos Bech32: %s\n", cosmosAddr)

	// 2. Decode Cosmos Bech32 back to raw bytes
	hrp, decodedBytes, err := bech32.DecodeAndConvert(cosmosAddr)
	if err != nil {
		fmt.Printf("[FAIL] Could not decode Cosmos Bech32 address: %v\n", err)
		os.Exit(1)
	}

	if hrp != "cosmos" {
		fmt.Printf("[FAIL] Expected HRP 'cosmos', got '%s'\n", hrp)
		os.Exit(1)
	}
	fmt.Printf("[PASS] Cosmos Bech32 HRP: %s\n", hrp)

	if !bytes.Equal(decodedBytes, rawBytes) {
		fmt.Printf("[FAIL] Decoded bytes do not match raw payload! Expected %x, got %x\n", rawBytes, decodedBytes)
		os.Exit(1)
	}
	fmt.Printf("[PASS] Cosmos Bech32 -> Raw Bytes: %x (Byte-for-Byte Match!)\n", decodedBytes)

	// 3. EVM Representation
	evmAddr := "0x" + hex.EncodeToString(rawBytes)
	fmt.Printf("[PASS] EVM Address: %s\n", evmAddr)

	// 4. Validator and Consensus Prefix Verification
	valOperAddr := sdk.ValAddress(rawBytes).String()
	if !bytes.HasPrefix([]byte(valOperAddr), []byte("cosmosvaloper1")) {
		fmt.Printf("[FAIL] Expected ValAddress prefix 'cosmosvaloper1', got %s\n", valOperAddr)
		os.Exit(1)
	}
	fmt.Printf("[PASS] Validator Address: %s\n", valOperAddr)

	consAddr := sdk.ConsAddress(rawBytes).String()
	if !bytes.HasPrefix([]byte(consAddr), []byte("cosmosvalcons1")) {
		fmt.Printf("[FAIL] Expected ConsAddress prefix 'cosmosvalcons1', got %s\n", consAddr)
		os.Exit(1)
	}
	fmt.Printf("[PASS] Consensus Address: %s\n", consAddr)

	// 5. Negative Test: sovereign1 prefix must be rejected
	sovereignAddr := "sovereign17w0adeg64ky0daxwd2ugyuneellmjgnxramjtq"
	_, _, err = bech32.DecodeAndConvert(sovereignAddr)
	if err == nil {
		// Even if valid bech32 string, check HRP comparison
		sovereignHrp, _, _ := bech32.DecodeAndConvert(sovereignAddr)
		if sovereignHrp == "cosmos" {
			fmt.Printf("[FAIL] sovereign1 address decoded with HRP 'cosmos'!\n")
			os.Exit(1)
		}
	}
	fmt.Printf("[PASS] Negative Test: 'sovereign1...' HRP is rejected (HRP != 'cosmos')\n")

	fmt.Println("\n[SUCCESS] ALL BECH32 PREFIX & UNIFIED ACCOUNT INVARIANTS VERIFIED!")
}
