package main

import (
	"bytes"
	"crypto/ed25519"
	"fmt"
	"github.com/cosmos/cosmos-sdk/types"
	cryptoed25519 "github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
)

func main() {
	seed := make([]byte, 32)
	copy(seed, []byte("oracle_operator_seed_for_mock_hsm"))
	reader := bytes.NewReader(seed)
	pub, _, err := ed25519.GenerateKey(reader)
	if err != nil {
		panic(err)
	}

	cosmosPubKey := &cryptoed25519.PubKey{Key: pub}
	addr := types.ValAddress(cosmosPubKey.Address().Bytes())
	accAddr := types.AccAddress(addr.Bytes()).String()
	fmt.Printf("ValAddress: %s\n", addr.String())
	fmt.Printf("AccAddress: %s\n", accAddr)
}
