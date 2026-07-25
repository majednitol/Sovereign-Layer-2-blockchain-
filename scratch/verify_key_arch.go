package main

import (
	"fmt"

	evmaddress "github.com/cosmos/evm/encoding/address"
	evmcrypto "github.com/cosmos/evm/crypto/ethsecp256k1"
	ethcommon "github.com/ethereum/go-ethereum/common"
)

func main() {
	privKey, err := evmcrypto.GenerateKey()
	if err != nil {
		panic(err)
	}

	pubKey := privKey.PubKey()
	cosmosRawAddr := pubKey.Address()

	// Convert 20-byte raw address to Bech32 ("cosmos1...") using evmaddress.NewEvmCodec("cosmos")
	codec := evmaddress.NewEvmCodec("cosmos")
	bech32Str, err := codec.BytesToString(cosmosRawAddr.Bytes())
	if err != nil {
		panic(err)
	}

	ethAddr := ethcommon.BytesToAddress(cosmosRawAddr.Bytes())

	fmt.Println("=== KEY ARCHITECTURE VERIFICATION ===")
	fmt.Printf("Key Algorithm           : %s\n", pubKey.Type())
	fmt.Printf("Raw 20-byte Address Hex : 0x%x\n", cosmosRawAddr.Bytes())
	fmt.Printf("EVM Address (0x...)     : %s\n", ethAddr.Hex())
	fmt.Printf("Cosmos Address (cosmos1): %s\n", bech32Str)
	fmt.Printf("Raw 20 Bytes Match?     : %v\n", ethcommon.BytesToAddress(cosmosRawAddr.Bytes()) == ethAddr)
}
