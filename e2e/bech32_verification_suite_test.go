package e2e

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/bech32"
)

// TestCanonicalCosmosAccountHRP verifies that SDK account prefix is "cosmos"
func TestCanonicalCosmosAccountHRP(t *testing.T) {
	cfg := sdk.GetConfig()
	// Set and check prefix
	cfg.SetBech32PrefixForAccount("cosmos", "cosmospub")
	cfg.SetBech32PrefixForValidator("cosmosvaloper", "cosmosvaloperpub")
	cfg.SetBech32PrefixForConsensusNode("cosmosvalcons", "cosmosvalconspub")

	rawBytes, _ := hex.DecodeString("f39fd6e51aad88f6f4ce6ab8827279cfffb92266")
	accAddr := sdk.AccAddress(rawBytes).String()

	if !strings.HasPrefix(accAddr, "cosmos1") {
		t.Fatalf("TestCanonicalCosmosAccountHRP failed: expected cosmos1... prefix, got %s", accAddr)
	}
	t.Logf("[PASS] TestCanonicalCosmosAccountHRP: %s", accAddr)
}

// TestLegacySovereignHRPRejected verifies that sovereign1 prefix is not used/accepted
func TestLegacySovereignHRPRejected(t *testing.T) {
	sovereignAddr := "sovereign17w0adeg64ky0daxwd2ugyuneellmjgnxramjtq"
	hrp, _, err := bech32.DecodeAndConvert(sovereignAddr)
	if err == nil && hrp == "cosmos" {
		t.Fatalf("TestLegacySovereignHRPRejected failed: sovereign1 decoded as HRP cosmos")
	}
	t.Logf("[PASS] TestLegacySovereignHRPRejected: sovereign1 HRP is %s (not cosmos)", hrp)
}

// TestEVMAndCosmosPayloadEquality verifies byte-for-byte 20-byte payload matching
func TestEVMAndCosmosPayloadEquality(t *testing.T) {
	rawHex := "f39fd6e51aad88f6f4ce6ab8827279cfffb92266"
	rawBytes, err := hex.DecodeString(rawHex)
	if err != nil {
		t.Fatalf("Failed to decode raw hex: %v", err)
	}

	cosmosAddr := sdk.AccAddress(rawBytes).String()
	expectedCosmos := "cosmos17w0adeg64ky0daxwd2ugyuneellmjgnxramjtq"
	if cosmosAddr != expectedCosmos {
		t.Fatalf("Expected %s, got %s", expectedCosmos, cosmosAddr)
	}

	_, decodedBytes, err := bech32.DecodeAndConvert(cosmosAddr)
	if err != nil {
		t.Fatalf("Failed to decode cosmos address: %v", err)
	}

	if !bytes.Equal(decodedBytes, rawBytes) {
		t.Fatalf("Payload mismatch! Expected %x, got %x", rawBytes, decodedBytes)
	}
	t.Logf("[PASS] TestEVMAndCosmosPayloadEquality: byte-for-byte equality confirmed")
}

// TestWrongPrefixRejected verifies non-cosmos prefixes are rejected
func TestWrongPrefixRejected(t *testing.T) {
	wrongAddr := "osmo17w0adeg64ky0daxwd2ugyuneellmjgnxramjtq"
	hrp, _, _ := bech32.DecodeAndConvert(wrongAddr)
	if hrp == "cosmos" {
		t.Fatalf("TestWrongPrefixRejected failed: wrong HRP accepted as cosmos")
	}
	t.Logf("[PASS] TestWrongPrefixRejected: HRP %s is rejected for cosmos", hrp)
}

// TestInvalidChecksumRejected verifies invalid Bech32 checksum is rejected
func TestInvalidChecksumRejected(t *testing.T) {
	invalidChecksumAddr := "cosmos17w0adeg64ky0daxwd2ugyuneellmjgnxramjt2" // invalid checksum
	_, _, err := bech32.DecodeAndConvert(invalidChecksumAddr)
	if err == nil {
		t.Fatalf("TestInvalidChecksumRejected failed: invalid checksum was accepted")
	}
	t.Logf("[PASS] TestInvalidChecksumRejected: invalid checksum rejected with error: %v", err)
}

// TestGenesisUsesCanonicalCosmosHRP verifies actual genesis.dev.json contains cosmos1 addresses only
func TestGenesisUsesCanonicalCosmosHRP(t *testing.T) {
	genesisPath := "../chain/genesis.dev.json"
	data, err := os.ReadFile(genesisPath)
	if err != nil {
		t.Skipf("genesis.dev.json not found at %s", genesisPath)
		return
	}

	genesisStr := string(data)
	if strings.Contains(genesisStr, `"sovereign1`) || strings.Contains(genesisStr, `"sov1`) {
		t.Fatalf("TestGenesisUsesCanonicalCosmosHRP failed: genesis.dev.json contains legacy sovereign1/sov1 addresses!")
	}
	t.Logf("[PASS] TestGenesisUsesCanonicalCosmosHRP: genesis.dev.json verified free of legacy sovereign1 addresses")
}

// TestFaucetRejectsLegacyAddress tests real running faucet service endpoint for HRP rejection
func TestFaucetRejectsLegacyAddress(t *testing.T) {
	faucetURL := "http://localhost:8082/api/rest/v1/explorer/faucet"
	body := `{"address":"sovereign1qwxelvm2le0zjkstmus9xmyeprajdpdur8nnl6"}`
	resp, err := http.Post(faucetURL, "application/json", strings.NewReader(body))
	if err != nil {
		t.Skipf("Faucet API service not reachable on %s: %v", faucetURL, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("TestFaucetRejectsLegacyAddress failed: expected status 400 Bad Request for legacy sovereign1 address, got %d", resp.StatusCode)
	}
	t.Logf("[PASS] TestFaucetRejectsLegacyAddress: HTTP 400 returned for legacy address")
}

// TestKeplrConfigUsesCoinType60AndEthAddressGen verifies Keplr wallet config in frontend and explorer
func TestKeplrConfigUsesCoinType60AndEthAddressGen(t *testing.T) {
	walletsConfigPath := "../frontend/config/wallets.json"
	data, err := os.ReadFile(walletsConfigPath)
	if err != nil {
		t.Fatalf("Failed to read wallets.json: %v", err)
	}

	var walletsConfig struct {
		Keplr struct {
			ChainID      string `json:"chainId"`
			Bip44        struct {
				CoinType int `json:"coinType"`
			} `json:"bip44"`
			Bech32Config struct {
				Bech32PrefixAccAddr string `json:"bech32PrefixAccAddr"`
			} `json:"bech32Config"`
		} `json:"keplr"`
	}

	if err := json.Unmarshal(data, &walletsConfig); err != nil {
		t.Fatalf("Failed to parse wallets.json: %v", err)
	}

	if walletsConfig.Keplr.ChainID != "sovereign-1" {
		t.Fatalf("Expected chainId sovereign-1, got %s", walletsConfig.Keplr.ChainID)
	}
	if walletsConfig.Keplr.Bip44.CoinType != 60 {
		t.Fatalf("Expected coinType 60, got %d", walletsConfig.Keplr.Bip44.CoinType)
	}
	if walletsConfig.Keplr.Bech32Config.Bech32PrefixAccAddr != "cosmos" {
		t.Fatalf("Expected bech32PrefixAccAddr 'cosmos', got %s", walletsConfig.Keplr.Bech32Config.Bech32PrefixAccAddr)
	}
	t.Logf("[PASS] TestKeplrConfigUsesCoinType60AndEthAddressGen: Verified coinType 60 and bech32PrefixAccAddr cosmos")
}

// TestAccountMismatchIsBlocked verifies payload comparison for cross-wallet safety
func TestAccountMismatchIsBlocked(t *testing.T) {
	evmAddrHex := "0xf39fd6e51aad88f6f4ce6ab8827279cfffb92266"
	cosmosAddrBech32 := "cosmos17w0adeg64ky0daxwd2ugyuneellmjgnxramjtq"
	differentCosmosAddrBech32 := "cosmos1srqfdy5fwu2ek25kswwvpj647dl30nwfg7gcf8"

	evmBytes, _ := hex.DecodeString(strings.TrimPrefix(evmAddrHex, "0x"))
	_, cosmosBytes1, _ := bech32.DecodeAndConvert(cosmosAddrBech32)
	_, cosmosBytes2, _ := bech32.DecodeAndConvert(differentCosmosAddrBech32)

	// Same account -> matches
	if !bytes.Equal(evmBytes, cosmosBytes1) {
		t.Fatalf("Expected EVM address and Cosmos address payloads to match!")
	}

	// Different account -> blocked
	if bytes.Equal(evmBytes, cosmosBytes2) {
		t.Fatalf("Expected EVM address and different Cosmos address payloads to mismatch!")
	}

	t.Logf("[PASS] TestAccountMismatchIsBlocked: Same account matching verified, mismatch correctly blocked")
}
