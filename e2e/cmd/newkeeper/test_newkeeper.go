package main

import (
	"fmt"
	"reflect"
	evmkeeper "github.com/cosmos/evm/x/vm/keeper"
)

func main() {
	fmt.Printf("NewKeeper Type: %s\n", reflect.TypeOf(evmkeeper.NewKeeper))
}
