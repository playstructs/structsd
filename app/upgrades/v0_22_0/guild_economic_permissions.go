package v0_22_0

import (
	"context"
	"strings"

	"structs/app/upgrades"
	"structs/x/structs/types"
)

// MigrateGuildEconomicPermissions prevents previously inert guild bits from
// becoming authority over new treasury and energy operations. Owners retain
// their live ownership authority and may explicitly delegate after upgrading.
func MigrateGuildEconomicPermissions(ctx context.Context, keepers *upgrades.Keepers) error {
	k := keepers.StructsKeeper
	newCapabilities := types.PermAssetsAll | types.PermSourceAllocation | types.PermAllocationConnection
	for _, row := range k.GetAllPermissionExport(ctx) {
		object, _, ok := strings.Cut(row.PermissionId, "@")
		if !ok || !strings.HasPrefix(object, "0-") {
			continue
		}
		if _, found := k.GetGuild(ctx, object); !found {
			continue
		}
		if types.Permission(row.Value)&newCapabilities != 0 {
			cc := k.NewCurrentContext(ctx)
			cc.PermissionRemove([]byte(row.PermissionId), newCapabilities)
			cc.CommitAll()
		}
	}
	for _, row := range k.GetAllGuildRankPermissionExport(ctx) {
		if !strings.HasPrefix(row.ObjectId, "0-") || types.Permission(row.Permissions)&newCapabilities == 0 {
			continue
		}
		if _, found := k.GetGuild(ctx, row.ObjectId); !found {
			continue
		}
		cc := k.NewCurrentContext(ctx)
		cc.RemovePermissionsGuildRank(cc.GetGuild(row.ObjectId), cc.GetGuild(row.GuildId), types.Permission(row.Permissions)&newCapabilities)
		cc.CommitAll()
	}
	return nil
}
