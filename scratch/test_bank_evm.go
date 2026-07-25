package main

import (
	"fmt"
	"reflect"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/sovereign-l1/chain/app"
)

func main() {
	// 1. Inspect app.App struct fields for BankKeeper and EVMKeeper
	appInstance := app.App{}
	
	tBank := reflect.TypeOf(appInstance.BankKeeper)
	tEVM := reflect.TypeOf(appInstance.EVMKeeper)

	fmt.Println("=== EXACT CODE IMPL AUDIT ===")
	fmt.Printf("BankKeeper Type : %v\n", tBank)
	fmt.Printf("EVMKeeper Type  : %v\n", tEVM)
}
