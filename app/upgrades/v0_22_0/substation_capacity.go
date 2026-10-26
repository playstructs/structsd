package v0_22_0

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/app/upgrades"
)

// MigrateSubstationConnectionCapacity repairs shares left stale when allocation
// creation/resizing changed outbound load without refreshing the source station.
// Apply the fixed 10% operating loss and enqueue overloads; normal bounded
// GridCascade, not migration-specific teardown, settles their allocations.
// Every station is the question at upgrade time; no live path performs this scan.
// A fresh CurrentContext per station bounds retained caches and cannot overwrite
// an earlier phase's grid writes with values read before that phase committed.
func MigrateSubstationConnectionCapacity(ctx context.Context, keepers *upgrades.Keepers) error {
	repaired, queued := 0, 0
	for _, row := range keepers.StructsKeeper.GetAllSubstation(ctx) {
		cc := keepers.StructsKeeper.NewCurrentContext(ctx)
		station := cc.GetSubstation(row.Id)
		if station.GetLoad() > station.GetUsableCapacity() {
			if err := keepers.StructsKeeper.AppendGridCascadeQueue(ctx, row.Id); err != nil {
				return err
			}
			queued++
		}
		previous := station.GetConnectionCapacity()
		cc.UpdateSubstationConnectionCapacity(row.Id)
		if previous != station.GetConnectionCapacity() {
			cc.CommitAll()
			repaired++
		}
	}
	sdk.UnwrapSDKContext(ctx).Logger().Info("Repaired substation connection capacity", "upgrade", UpgradeName, "repaired", repaired, "overloaded", queued)
	return nil
}
