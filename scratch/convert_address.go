package main

import (
	"encoding/hex"
	"fmt"

	"github.com/cosmos/cosmos-sdk/types/bech32"
)

func main() {
	_, bytes, err := bech32.DecodeAndConvert("sovereign1qwxelvm2le0zjkstmus9xmyeprajdpdur8nnl6")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	cosmosAddr, _ := bech32.ConvertAndEncode("cosmos", bytes)
	fmt.Printf("EVM Hex Address: 0x%s\n", hex.EncodeToString(bytes))
	fmt.Printf("Canonical Cosmos Bech32 Address: %s\n", cosmosAddr)
}
