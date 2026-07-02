package app

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/CosmWasm/wasmd/x/wasm"
)

// Pre-computed Module Account Addresses for Genesis Smart Contracts
// These reserved accounts allow early bootstrapping of balances and permissions before contracts are deployed.
var (
	// ConstitutionContractAddr is the reserved address of the Sovereign Constitution Contract
	ConstitutionContractAddr sdk.AccAddress

	// TreasuryContractAddr is the reserved address of the Sovereign Treasury Contract
	TreasuryContractAddr     sdk.AccAddress

	// ReserveFundContractAddr is the reserved address of the Sovereign Reserve Fund Contract
	ReserveFundContractAddr  sdk.AccAddress

	// GovernanceContractAddr is the reserved address of the Sovereign Governance Contract
	GovernanceContractAddr   sdk.AccAddress
)

func init() {
	// Generate deterministic addresses matching module account schemes
	ConstitutionContractAddr = types.NewModuleAddress("wasm.constitution")
	TreasuryContractAddr     = types.NewModuleAddress("wasm.treasury")
	ReserveFundContractAddr  = types.NewModuleAddress("wasm.reserve")
	GovernanceContractAddr   = types.NewModuleAddress("wasm.governance")
}

// GetWasmOpts returns standard configuration options for wasmd runtime
func GetWasmOpts(appOpts map[string]interface{}) []wasm.Option {
	var wasmOpts []wasm.Option

	// Governance-only contract uploads are enforced at genesis via CodeUploadAccess parameters.
	// No custom runtime keeper options are needed here.
	return wasmOpts
}
