package v0_23_0

import (
	"context"

	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	"structs/app/upgrades"
)

// CreateUpgradeHandler runs the standard module migrations for v0.23.0.
// Backfill planet sectors without replaying the v0.22.0 economy conversion.
// The substation maximum still defaults to unlimited on a missing grid row.
func CreateUpgradeHandler(
	mm *module.Manager,
	configurator module.Configurator,
	keepers *upgrades.Keepers,
) upgradetypes.UpgradeHandler {
	return func(ctx context.Context, _ upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
		versionMap, err := mm.RunMigrations(ctx, configurator, fromVM)
		if err != nil {
			return nil, err
		}
		keepers.StructsKeeper.BackfillPlanetSectors(ctx)
		return versionMap, nil
	}
}

func NewUpgrade() upgrades.Upgrade {
	return upgrades.Upgrade{
		UpgradeName:          UpgradeName,
		CreateUpgradeHandler: CreateUpgradeHandler,
		StoreUpgrades:        storetypes.StoreUpgrades{},
	}
}
