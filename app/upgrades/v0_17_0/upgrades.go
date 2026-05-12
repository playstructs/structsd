package v0_17_0

import (
	"context"

	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	"structs/app/upgrades"
)

// CreateUpgradeHandler returns the v0.17.0 upgrade handler. The upgrade is
// binary-only: there are no state migrations, no store-key changes, and no
// param-table changes. All behavior changes are in the ante chain and the
// MsgStructActivate handler, and take effect the moment validators run the
// new binary.
//
// We still register a handler so the on-chain `software-upgrade` governance
// proposal can coordinate the binary swap deterministically across validators.
// The handler is a thin wrapper around RunMigrations to satisfy
// upgrade-keeper's invariants.
func CreateUpgradeHandler(
	mm *module.Manager,
	configurator module.Configurator,
	_ *upgrades.Keepers,
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
