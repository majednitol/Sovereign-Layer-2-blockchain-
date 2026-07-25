package main

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/crypto/hd"
	bip39 "github.com/cosmos/go-bip39"
	evmaddress "github.com/cosmos/evm/encoding/address"
	ethcommon "github.com/ethereum/go-ethereum/common"
)

func main() {
	// 1. Generate test mnemonic
	mnemonic := "satisfy adjust timber high purchase tuition stool fossil kiss naive queue sketch"

	seed := bip39.NewSeed(mnemonic, "")

	// Path 1: EVM Path m/44'/60'/0'/0/0 (MetaMask default & Evmos/Ethermint Cosmos wallets)
	pathEth := hd.CreateHDPath(60, 0, 0).String()
	masterEth, chEth := hd.ComputeMastersFromSeed(seed)
	privKeyEthBytes, err := hd.DerivePrivateKeyForPath(masterEth, chEth, pathEth)
	if err != nil {
		panic(err)
	}

	// Path 2: Standard Cosmos Path m/44'/118'/0'/0/0 (CosmHub / Keplr default fallback)
	pathCosmos := hd.CreateHDPath(118, 0, 0).String()
	masterCosmos, chCosmos := hd.ComputeMastersFromSeed(seed)
	privKeyCosmosBytes, err := hd.DerivePrivateKeyForPath(masterCosmos, chCosmos, pathCosmos)
	if err != nil {
		panic(err)
	}

	fmt.Println("=== HD DERIVATION PATH AUDIT ===")
	fmt.Printf("Mnemonic                    : %s\n", mnemonic)
	fmt.Printf("Path 1 (m/44'/60'/0'/0/0)   : PrivKey 0x%x\n", privKeyEthBytes)
	fmt.Printf("Path 2 (m/44'/118'/0'/0/0)  : PrivKey 0x%x\n", privKeyCosmosBytes)

	codec := evmaddress.NewEvmCodec("cosmos")
	ethRawAddr := ethcommon.BytesToAddress(privKeyEthBytes[:20]) // simplified check
	_ = codec
	_ = ethRawAddr

	fmt.Printf("Keys match across paths?    : %v\n", string(privKeyEthBytes) == string(privKeyCosmosBytes))
}
