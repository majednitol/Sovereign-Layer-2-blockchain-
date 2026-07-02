package app

import (
	"context"

	storetypes "github.com/cosmos/cosmos-sdk/store/v2/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"
)

const UpgradeNameV1 = "v1.0.0"

// RegisterUpgradeHandlers registers the upgrade handlers for the chain
func (app *App) RegisterUpgradeHandlers() {
	app.UpgradeKeeper.SetUpgradeHandler(
		UpgradeNameV1,
		func(ctx context.Context, plan upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
			sdkCtx := sdk.UnwrapSDKContext(ctx)
			sdkCtx.Logger().Info("Executing Sovereign L1 v1.0.0 Upgrade Handler...")

			// Standard SDK v50 upgraded migrations can be executed here.
			// This represents a no-op scaffold so the chain doesn't crash at the upgrade height boundary.
			return app.mm.RunMigrations(sdkCtx, app.configurator(), fromVM)
		},
	)

	upgradeInfo, err := app.UpgradeKeeper.ReadUpgradeInfoFromDisk()
	if err != nil {
		panic(err)
	}

	if upgradeInfo.Name == UpgradeNameV1 && !app.UpgradeKeeper.IsSkipHeight(upgradeInfo.Height) {
		storeUpgrades := storetypes.StoreUpgrades{
			Added: []string{
				// Register any store names added in this migration (e.g. custom modules)
			},
		}
		// Configure store loader to execute state migrations safely
		app.SetStoreLoader(upgradetypes.UpgradeStoreLoader(upgradeInfo.Height, &storeUpgrades))
	}
}

// configurator stub representation for App compilation compatibility
type configurator struct{}
var _ configurator = configurator{}

func (c configurator) RunMigrations(ctx sdk.Context, fromVM module.VersionMap) (module.VersionMap, error) {
	return fromVM, nil
}

func (app *App) configurator() module.Configurator {
	// Stub return Configurator
	return nil
}
