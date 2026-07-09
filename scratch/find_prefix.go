package main

import (
	"fmt"
	"github.com/cosmos/cosmos-sdk/types/bech32"
)

func main() {
	// Our data part decoded from cosmos1n4agesyhv32aw03zu3xlsemvc3dvq4d635c5mq
	_, data, err := bech32.DecodeAndConvert("cosmos1n4agesyhv32aw03zu3xlsemvc3dvq4d635c5mq")
	if err != nil {
		fmt.Printf("Decode error: %v\n", err)
		return
	}

	// Try common prefixes
	prefixes := []string{"cosmos", "sovereign", "chain", "node", "addr", "account", "val"}
	for _, p := range prefixes {
		addr, _ := bech32.ConvertAndEncode(p, data)
		fmt.Printf("Prefix: %s => Address: %s\n", p, addr)
	}
}
