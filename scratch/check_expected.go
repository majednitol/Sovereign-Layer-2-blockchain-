package main

import (
	"fmt"
	"github.com/cosmos/cosmos-sdk/types/bech32"
)

func main() {
	// Let's decode the original address
	_, data, err := bech32.DecodeAndConvert("cosmos1n4agesyhv32aw03zu3xlsemvc3dvq4d635c5mq")
	if err != nil {
		fmt.Printf("Decode error: %v\n", err)
		return
	}

	fmt.Printf("Decoded data bytes (hex): %x\n", data)

	// In Cosmos SDK, standard address length is 20 bytes.
	// But EVM addresses converted to Bech32 are sometimes encoded differently,
	// or the Bech32 format on the chain is Bech32m instead of Bech32!
	// Let's check both Bech32 and Bech32m!
	// Let's print out what Address would have the checksum "uccuw0" for prefix "cosmos"
	// and what Address would have the checksum "p96qj5" for prefix "sovereign"
	
	// Let's try to convert and encode with different prefixes and print their checksums
	for _, prefix := range []string{"cosmos", "sovereign"} {
		addr, _ := bech32.ConvertAndEncode(prefix, data)
		fmt.Printf("Standard Bech32 (%s): %s\n", prefix, addr)
	}
}
