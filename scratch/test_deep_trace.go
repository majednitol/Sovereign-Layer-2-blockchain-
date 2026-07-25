package main

import (
	"fmt"

	"cosmossdk.io/math"
	dbm "github.com/cosmos/cosmos-db"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	evmaddress "github.com/cosmos/evm/encoding/address"
	erc20types "github.com/cosmos/evm/x/erc20/types"
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/sovereign-l1/chain/app"
)

func main() {
	db := dbm.NewMemDB()
	sovereignApp := app.NewApp(nil, db, nil, true, map[string]interface{}{"home": "/tmp/test-sovereign"})

	// Access BaseApp context or BankKeeper directly
	ctx := sdk.Context{}.WithChainID("sovereign-1")

	// Create test address
	codec := evmaddress.NewEvmCodec("cosmos")
	ethHexAddr := "0xB83A53C4cF68D82921ae113640Eea96Df28BAC06"
	addrBytes, err := codec.StringToBytes(ethHexAddr)
	if err != nil {
		panic(err)
	}
	accAddr := sdk.AccAddress(addrBytes)

	fmt.Println("=== DEEP CODE-LEVEL TRACE ===")
	fmt.Printf("Testing Account Address (Bech32): %s\n", accAddr.String())
	fmt.Printf("Testing Account Address (Hex)   : %s\n", ethHexAddr)

	// 1. Initial State Check in x/bank
	balUcsovInit := sovereignApp.BankKeeper.GetBalance(ctx, accAddr, "ucsov")
	balAesovInit := sovereignApp.BankKeeper.GetBalance(ctx, accAddr, "aesov")
	fmt.Printf("\n[Step 1] Initial x/bank balances:\n")
	fmt.Printf("  - ucsov (Cosmos Base): %s\n", balUcsovInit.String())
	fmt.Printf("  - aesov (EVM Unit)   : %s\n", balAesovInit.String())

	// 2. Fund account with ucsov (100 CSOV = 100,000,000 ucsov) via x/bank MintCoins
	mintCoins := sdk.NewCoins(sdk.NewCoin("ucsov", math.NewInt(100_000_000)))
	err = sovereignApp.BankKeeper.MintCoins(ctx, banktypes.ModuleName, mintCoins)
	if err != nil {
		panic(err)
	}
	err = sovereignApp.BankKeeper.SendCoinsFromModuleToAccount(ctx, banktypes.ModuleName, accAddr, mintCoins)
	if err != nil {
		panic(err)
	}

	balUcsovAfterMint := sovereignApp.BankKeeper.GetBalance(ctx, accAddr, "ucsov")
	balAesovAfterMint := sovereignApp.BankKeeper.GetBalance(ctx, accAddr, "aesov")
	fmt.Printf("\n[Step 2] After minting 100,000,000 ucsov to account via x/bank:\n")
	fmt.Printf("  - ucsov (Cosmos Base): %s\n", balUcsovAfterMint.String())
	fmt.Printf("  - aesov (EVM Unit)   : %s\n", balAesovAfterMint.String())

	// 3. EVM Module Parameters
	params := sovereignApp.EVMKeeper.GetParams(ctx)
	fmt.Printf("\n[Step 3] EVM Module Configured EvmDenom: %s\n", params.EvmDenom)

	ethAddr := ethcommon.BytesToAddress(accAddr.Bytes())
	fmt.Printf("  - ethcommon.BytesToAddress(accAddr.Bytes()): %s\n", ethAddr.Hex())

	// 4. Test Deducting Gas Fee in EVM (Deduct 1,500 ucsov)
	feeAmountUtoken := math.NewInt(1_500)
	feeCoins := sdk.NewCoins(sdk.NewCoin("ucsov", feeAmountUtoken))
	err = sovereignApp.BankKeeper.SendCoinsFromAccountToModule(ctx, accAddr, authtypes.FeeCollectorName, feeCoins)
	if err != nil {
		panic(err)
	}

	balUcsovAfterFee := sovereignApp.BankKeeper.GetBalance(ctx, accAddr, "ucsov")
	feeCollectorAddr := sovereignApp.AccountKeeper.GetModuleAddress(authtypes.FeeCollectorName)
	balFeeCollector := sovereignApp.BankKeeper.GetBalance(ctx, feeCollectorAddr, "ucsov")

	fmt.Printf("\n[Step 4] After Deducting Gas Fee (1,500 ucsov):\n")
	fmt.Printf("  - User ucsov balance        : %s\n", balUcsovAfterFee.String())
	fmt.Printf("  - FeeCollector ucsov balance: %s\n", balFeeCollector.String())

	// 5. Test x/erc20 Escrow ConvertCoin
	erc20ModuleAddr := sovereignApp.AccountKeeper.GetModuleAddress(erc20types.ModuleName)
	balErc20ModuleBefore := sovereignApp.BankKeeper.GetBalance(ctx, erc20ModuleAddr, "ucsov")
	fmt.Printf("\n[Step 5] x/erc20 Escrow Check:\n")
	fmt.Printf("  - x/erc20 Module Account Address: %s\n", erc20ModuleAddr.String())
	fmt.Printf("  - x/erc20 Module ucsov Escrow    : %s\n", balErc20ModuleBefore.String())

	fmt.Println("\n=== TRACE COMPLETE ===")
}
