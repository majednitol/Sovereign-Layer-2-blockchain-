//go:build ignore
// +build ignore

package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/sovereign-l1/chain/app"
)

func main() {
	interfaceRegistry := codectypes.NewInterfaceRegistry()
	app.ModuleBasics.RegisterInterfaces(interfaceRegistry)
	appCodec := codec.NewProtoCodec(interfaceRegistry)

	raw := app.ModuleBasics.DefaultGenesis(appCodec)
	data, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println(string(data))
}
