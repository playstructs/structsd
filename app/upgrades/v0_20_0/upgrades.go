package v0_20_0

import (
	"context"

	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	"structs/app/upgrades"
)

// CreateUpgradeHandler returns the v0.20.0 upgrade handler.
//
// v0.20.0 is a binary-only behavior fix (the dead-bomber defender-ordering
// correction described in constants.go) and carries no state migration. The
// handler runs the standard module migrations and nothing else; RunMigrations
// is still invoked for consistency with the SDK module set even though the
// structs module registers no per-version migrations.
func CreateUpgradeHandler(
	mm *module.Manager,
	configurator module.Configurator,
	keepers *upgrades.Keepers,
) upgradetypes.UpgradeHandler {
	return func(ctx context.Context, _ upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
		return mm.RunMigrations(ctx, configurator, fromVM)
	}
}

func NewUpgrade() upgrades.Upgrade {
	return upgrades.Upgrade{
		UpgradeName:          UpgradeName,
		CreateUpgradeHandler: CreateUpgradeHandler,
		StoreUpgrades:        storetypes.StoreUpgrades{},
	}
}
