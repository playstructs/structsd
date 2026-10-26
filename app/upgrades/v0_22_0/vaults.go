package v0_22_0

import (
	"context"

	"structs/app/upgrades"
)

// MigrateProvisionVaults provisions every existing player and guild in globally
// sorted owner order. Ordinary liquid balances move in a separate phase;
// staking and rewards remain at the original delegator addresses.
func MigrateProvisionVaults(ctx context.Context, keepers *upgrades.Keepers) error {
	k := keepers.StructsKeeper
	ids := make([]string, 0)
	for _, player := range k.GetAllPlayer(ctx) {
		ids = append(ids, player.Id)
	}
	for _, guild := range k.GetAllGuild(ctx) {
		ids = append(ids, guild.Id)
	}
	_, err := k.NewCurrentContext(ctx).EnsureVaults(ids...)
	return err
}
