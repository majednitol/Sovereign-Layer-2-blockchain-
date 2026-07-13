# Fix Plan: nil Configurator Panic at Upgrade Height

**File:** `chain/app/upgrades.go` (only file that needs changes)  
**Severity:** Production-blocking — chain halts with a panic at the v1.0.0 upgrade height.  
**Constraint:** Do not delete or modify any other file.

---

## Exact Bug

`upgrades.go` defines a private method `configurator()` on `*App` that unconditionally
returns `nil`:

```go
// upgrades.go:52-55  ← BUG
func (app *App) configurator() module.Configurator {
    return nil
}
```

The upgrade handler then calls:

```go
// upgrades.go:24  ← caller
return app.mm.RunMigrations(sdkCtx, app.configurator(), fromVM)
```

`RunMigrations` receives `nil` for its `module.Configurator` argument and panics
immediately when it tries to call methods on it.

---

## Why the Fix is One Line

`app.go` already has everything needed:

| What | Location in app.go |
|------|-------------------|
| `Configurator module.Configurator` field on `App` struct | line 239–240 |
| `app.Configurator = module.NewConfigurator(appCodec, MsgServiceRouter(), GRPCQueryRouter())` | line 730 |
| `app.mm.RegisterServices(app.Configurator)` — confirms it is valid | line 731 |
| `app.RegisterUpgradeHandlers()` called **after** line 730 | line 736 |

The struct field `app.Configurator` is already populated before `RegisterUpgradeHandlers()`
runs. The private method `configurator()` is never needed — it only hides the field with a
nil-returning shadow.

---

## Change 1 — Replace the nil-returning call with the struct field

**Location:** `upgrades.go` line 24, inside the upgrade handler closure.

Find:

```go
			return app.mm.RunMigrations(sdkCtx, app.configurator(), fromVM)
```

Replace with:

```go
			return app.mm.RunMigrations(sdkCtx, app.Configurator, fromVM)
```

---

## Change 2 — Delete the nil-returning method and its dead-code stub type

**Location:** `upgrades.go` lines 44–55.

Find and **delete** the entire block:

```go
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
```

Replace with nothing (remove entirely).

> **Why delete instead of leaving it:** The `type configurator struct{}` is a local type
> that shadows the `module.Configurator` interface name in the file scope. Leaving it
> alongside the fix is confusing and the `var _ configurator = configurator{}` compile
> check is meaningless since the local type satisfies itself trivially. The method
> `(app *App) configurator()` would become an unreachable dead method. Remove all of it.

---

## Change 3 — Remove the now-unused imports (if any become unused after Change 2)

After deleting the stub block, check whether `sdk "github.com/cosmos/cosmos-sdk/types"`
is still used in the file. It is used on line 19 (`sdk.UnwrapSDKContext`) so it stays.
The `module` import is used on line 18 and line 24, so it stays. No import changes needed.

---

## Final state of `upgrades.go` after all changes

The file should look exactly like this:

```go
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

			// Standard SDK v50 module migrations. app.Configurator is initialised
			// in NewApp (app.go) before RegisterUpgradeHandlers is called.
			return app.mm.RunMigrations(sdkCtx, app.Configurator, fromVM)
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
```

---

## Verification commands (run after applying the changes)

```bash
# 1. Confirm the file compiles cleanly
cd chain && go build ./app/...

# 2. Confirm no reference to the deleted method remains
grep -n "configurator()" chain/app/upgrades.go
# Expected: no output

# 3. Confirm the struct field is used correctly
grep -n "app\.Configurator" chain/app/upgrades.go
# Expected: one line — the RunMigrations call

# 4. Confirm app.Configurator is still initialised in NewApp
grep -n "app\.Configurator" chain/app/app.go
# Expected: two lines — NewConfigurator assignment (line ~730) and RegisterServices call (line ~731)

# 5. Run existing app tests
cd chain && go test ./app/... -v -count=1
```

---

## Summary

| # | Action | Location |
|---|--------|----------|
| 1 | Change `app.configurator()` → `app.Configurator` | `upgrades.go` line 24 |
| 2 | Delete the entire stub block (type + var + two methods, lines 44–55) | `upgrades.go` lines 44–55 |
| 3 | Verify imports (no change needed) | `upgrades.go` import block |
