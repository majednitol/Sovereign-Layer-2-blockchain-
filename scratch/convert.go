package main

import (
	"fmt"
	"encoding/hex"
	"github.com/cosmos/cosmos-sdk/types/bech32"
)

func main() {
	evmAddrBytes, _ := hex.DecodeString("f39fd6e51aad88f6f4ce6ab8827279cfffb92266")
	bech32Addr, _ := bech32.ConvertAndEncode("cosmos", evmAddrBytes)
	fmt.Printf("EVM address 0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266 => Bech32: %s\n", bech32Addr)
}
