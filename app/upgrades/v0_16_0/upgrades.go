package v0_16_0

import (
	"context"
	"strings"

	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	"structs/app/upgrades"
	"structs/x/structs/types"
)

const (
	OldPermAll      = types.PermAll &^ types.PermGuildUGCUpdate
	OldPermGuildAll = types.PermGuildAll &^ types.PermGuildUGCUpdate
	UGCBitIndex     = 24
)

func CreateUpgradeHandler(
	mm *module.Manager,
	configurator module.Configurator,
	keepers *upgrades.Keepers,
) upgradetypes.UpgradeHandler {
	return func(ctx context.Context, plan upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
		migrateUGCPermissions(ctx, keepers)
		migrateStructTypes(ctx, keepers)
		return mm.RunMigrations(ctx, configurator, fromVM)
	}
}

func migrateUGCPermissions(ctx context.Context, keepers *upgrades.Keepers) {
	k := keepers.StructsKeeper

	allPerms := k.GetAllPermissionExport(ctx)
	for _, rec := range allPerms {
		val := types.Permission(rec.Value)

		if strings.HasPrefix(rec.PermissionId, "8-") {
			if val&OldPermAll == OldPermAll {
				k.SetPermissionsByBytes(ctx, []byte(rec.PermissionId), val|types.PermGuildUGCUpdate)
			}
			continue
		}

		if strings.HasPrefix(rec.PermissionId, "0-") && strings.Contains(rec.PermissionId, "@") {
			if val&OldPermGuildAll == OldPermGuildAll {
				k.SetPermissionsByBytes(ctx, []byte(rec.PermissionId), val|types.PermGuildUGCUpdate)
			}
		}
	}

	allGuilds := k.GetAllGuild(ctx)
	for _, guild := range allGuilds {
		register := k.ReadGuildRankRegister(ctx, guild.Id, guild.Id)
		register[UGCBitIndex] = 1
		k.WriteGuildRankRegister(ctx, guild.Id, guild.Id, register)
	}
}

func migrateStructTypes(ctx context.Context, keepers *upgrades.Keepers) {
	for _, st := range types.CreateStructTypeGenesis() {
		keepers.StructsKeeper.SetStructType(ctx, st)
	}
}

func NewUpgrade() upgrades.Upgrade {
	return upgrades.Upgrade{
		UpgradeName:          UpgradeName,
		CreateUpgradeHandler: CreateUpgradeHandler,
		StoreUpgrades:        storetypes.StoreUpgrades{},
	}
}
