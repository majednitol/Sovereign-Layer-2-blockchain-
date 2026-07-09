package main

import (
	"fmt"
	"encoding/hex"
	evmaddress "github.com/cosmos/evm/encoding/address"
)

func main() {
	codec := evmaddress.NewEvmCodec("cosmos")
	addrBytes, _ := hex.DecodeString("07882ae1ecb7429a84f1d53048d35c4bb2056877")
	str, _ := codec.BytesToString(addrBytes)
	fmt.Printf("NewEvmCodec('cosmos') for 0x07882Ae1ecB7429a84f1D53048d35c4bB2056877 => %s\n", str)
}
