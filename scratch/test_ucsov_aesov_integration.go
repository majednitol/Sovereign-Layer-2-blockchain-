package main

import (
	"fmt"

	"cosmossdk.io/math"
	dbm "github.com/cosmos/cosmos-db"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	evmaddress "github.com/cosmos/evm/encoding/address"
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/sovereign-l1/chain/app"
)

func main() {
	db := dbm.NewMemDB()
	sovereignApp := app.NewApp(nil, db, nil, true, map[string]interface{}{"home": "/tmp/test-sovereign"})

	ctx := sovereignApp.BaseApp.NewContext(false)

	// Create test address
	codec := evmaddress.NewEvmCodec("cosmos")
	ethHexAddr := "0xB83A53C4cF68D82921ae113640Eea96Df28BAC06"
	addrBytes, err := codec.StringToBytes(ethHexAddr)
	if err != nil {
		panic(err)
	}
	accAddr := sdk.AccAddress(addrBytes)
	ethAddr := ethcommon.BytesToAddress(accAddr.Bytes())

	fmt.Println("==================================================================")
	fmt.Println("       EMPIRICAL INTEGRATION TEST: ucsov vs aesov IN BANK & EVM")
	fmt.Println("==================================================================")

	// ------------------------------------------------------------------
	// STEP 1: INITIAL STATE (Before any minting)
	// ------------------------------------------------------------------
	balUcsovStep1 := sovereignApp.BankKeeper.GetBalance(ctx, accAddr, "ucsov")
	balAesovStep1 := sovereignApp.BankKeeper.GetBalance(ctx, accAddr, "aesov")
	supplyUcsovStep1 := sovereignApp.BankKeeper.GetSupply(ctx, "ucsov")
	supplyAesovStep1 := sovereignApp.BankKeeper.GetSupply(ctx, "aesov")

	fmt.Println("\n[STEP 1] Initial Balances & Supply:")
	fmt.Printf("  - User ucsov balance  : %s\n", balUcsovStep1.String())
	fmt.Printf("  - User aesov balance  : %s\n", balAesovStep1.String())
	fmt.Printf("  - Bank Total ucsov Supply: %s\n", supplyUcsovStep1.String())
	fmt.Printf("  - Bank Total aesov Supply: %s\n", supplyAesovStep1.String())

	// ------------------------------------------------------------------
	// STEP 2: MINT 1,000,000 ucsov (1 CSOV) TO USER ADDRESS VIA x/bank
	// ------------------------------------------------------------------
	mintAmountUtoken := math.NewInt(1_000_000)
	mintCoins := sdk.NewCoins(sdk.NewCoin("ucsov", mintAmountUtoken))

	err = sovereignApp.BankKeeper.MintCoins(ctx, banktypes.ModuleName, mintCoins)
	if err != nil {
		panic(fmt.Sprintf("Failed to mint ucsov: %v", err))
	}
	err = sovereignApp.BankKeeper.SendCoinsFromModuleToAccount(ctx, banktypes.ModuleName, accAddr, mintCoins)
	if err != nil {
		panic(fmt.Sprintf("Failed to send ucsov to account: %v", err))
	}

	balUcsovStep2 := sovereignApp.BankKeeper.GetBalance(ctx, accAddr, "ucsov")
	balAesovStep2 := sovereignApp.BankKeeper.GetBalance(ctx, accAddr, "aesov")
	supplyUcsovStep2 := sovereignApp.BankKeeper.GetSupply(ctx, "ucsov")
	supplyAesovStep2 := sovereignApp.BankKeeper.GetSupply(ctx, "aesov")

	fmt.Println("\n[STEP 2] After Minting 1,000,000 ucsov (1 CSOV) via x/bank:")
	fmt.Printf("  - User ucsov balance  : %s\n", balUcsovStep2.String())
	fmt.Printf("  - User aesov balance  : %s\n", balAesovStep2.String())
	fmt.Printf("  - Bank Total ucsov Supply: %s\n", supplyUcsovStep2.String())
	fmt.Printf("  - Bank Total aesov Supply: %s\n", supplyAesovStep2.String())

	// ------------------------------------------------------------------
	// STEP 3: EVM MODULE PARAMETERS & CONVERTED QUERY
	// ------------------------------------------------------------------
	evmParams := sovereignApp.EVMKeeper.GetParams(ctx)
	fmt.Printf("\n[STEP 3] EVM Module Parameters:\n")
	fmt.Printf("  - EVM Configured evm_denom: %s\n", evmParams.EvmDenom)
	fmt.Printf("  - Eth Address derived from AccAddress bytes: %s\n", ethAddr.Hex())

	// ------------------------------------------------------------------
	// STEP 4: EVM GAS FEE DEDUCTION (1,500 ucsov)
	// ------------------------------------------------------------------
	gasFeeUtoken := math.NewInt(1_500)
	feeCoins := sdk.NewCoins(sdk.NewCoin("ucsov", gasFeeUtoken))

	err = sovereignApp.BankKeeper.SendCoinsFromAccountToModule(ctx, accAddr, authtypes.FeeCollectorName, feeCoins)
	if err != nil {
		panic(fmt.Sprintf("Failed to deduct gas fee: %v", err))
	}

	balUcsovStep4 := sovereignApp.BankKeeper.GetBalance(ctx, accAddr, "ucsov")
	balAesovStep4 := sovereignApp.BankKeeper.GetBalance(ctx, accAddr, "aesov")
	feeCollectorAddr := sovereignApp.AccountKeeper.GetModuleAddress(authtypes.FeeCollectorName)
	balFeeCollector := sovereignApp.BankKeeper.GetBalance(ctx, feeCollectorAddr, "ucsov")
	supplyUcsovStep4 := sovereignApp.BankKeeper.GetSupply(ctx, "ucsov")

	fmt.Println("\n[STEP 4] After Deducting EVM Gas Fee (1,500 ucsov):")
	fmt.Printf("  - User ucsov balance        : %s\n", balUcsovStep4.String())
	fmt.Printf("  - User aesov balance        : %s\n", balAesovStep4.String())
	fmt.Printf("  - FeeCollector ucsov balance: %s\n", balFeeCollector.String())
	fmt.Printf("  - Bank Total ucsov Supply   : %s\n", supplyUcsovStep4.String())

	// ------------------------------------------------------------------
	// STEP 5: TOTAL SUPPLY INVARIANT AUDIT
	// ------------------------------------------------------------------
	fmt.Println("\n[STEP 5] Total Supply Invariant Audit:")
	fmt.Printf("  - Bank ucsov total supply unchanged by fee transfer? %v\n", supplyUcsovStep2.Equal(supplyUcsovStep4))
	fmt.Printf("  - User ucsov + FeeCollector ucsov = Initial Mint (1,000,000)? %v\n", balUcsovStep4.Amount.Add(balFeeCollector.Amount).Equal(mintAmountUtoken))
	fmt.Printf("  - Is aesov supply strictly 0 in x/bank? %v\n", supplyAesovStep1.Amount.IsZero() && supplyAesovStep2.Amount.IsZero())

	fmt.Println("\n==================================================================")
	fmt.Println("                    INTEGRATION TEST COMPLETE                     ")
	fmt.Println("==================================================================")
}
