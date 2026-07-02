package main

import (
	"fmt"
	evmtypes "github.com/cosmos/evm/x/vm/types"
)

func main() {
	fmt.Printf("DefaultEVMChainID: %d\n", evmtypes.DefaultEVMChainID)
}
