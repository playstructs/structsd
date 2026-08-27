package v0_22_0

import (
	"context"

	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	"structs/app/upgrades"
	structstypes "structs/x/structs/types"
)

// CreateUpgradeHandler returns the v0.22.0 upgrade handler.
//
// Most of v0.22.0 is binary-only (described in constants.go). The one piece of
// stored state it has to repair is reactor infusion fuel, which was computed
// with a per-delegation rounding that let a sharded stake outlive a slash.
func CreateUpgradeHandler(
	mm *module.Manager,
	configurator module.Configurator,
	keepers *upgrades.Keepers,
) upgradetypes.UpgradeHandler {
	return func(ctx context.Context, _ upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
		// Before anything that can enqueue: the re-key reads every row under
		// the queue prefix as a legacy object-id key, which stops being true the
		// moment a new-shape row exists.
		if err := MigrateGridCascadeQueue(ctx, keepers); err != nil {
			return nil, err
		}

		if err := MigrateInfusionFuelRounding(ctx, keepers); err != nil {
			return nil, err
		}

		if err := MigrateSelfDefenseRegistrations(ctx, keepers); err != nil {
			return nil, err
		}

		return mm.RunMigrations(ctx, configurator, fromVM)
	}
}

/* MigrateInfusionFuelRounding recomputes every reactor infusion's fuel against
 * the truncating conversion.
 *
 * Fuel was rounded per delegation, so a shard worth exactly x.5 rounded up, and
 * a stake split finely enough kept its whole pre-slash fuel - and the grid
 * capacity standing on it - after being slashed. Fixing the arithmetic only
 * corrects an infusion the next time staking touches that delegation, and a
 * delegation nobody moves is never touched, so the inflated capacity would sit
 * there indefinitely. This walks them once.
 *
 * ReconcileInfusionForDelegation reads the live delegation and writes what it
 * finds, so it needs no notion of what the old number was; an infusion whose
 * delegation is gone is zeroed by the same path. Capacity moves down where it
 * moves at all, and the grid cascade that follows is the point rather than a
 * side effect - it sheds the allocations that were only ever powered by
 * rounding.
 */
func MigrateInfusionFuelRounding(ctx context.Context, keepers *upgrades.Keepers) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	logger := sdkCtx.Logger().With("upgrade", UpgradeName, "phase", "migrateInfusionFuelRounding")

	k := keepers.StructsKeeper

	var visited, changed int
	var fuelReleased uint64

	for _, reactor := range k.GetAllReactor(ctx) {
		validatorAddress, err := sdk.ValAddressFromBech32(reactor.Validator)
		if err != nil {
			logger.Error("skipping reactor with unparsable validator address",
				"reactorId", reactor.Id, "validator", reactor.Validator, "error", err)
			continue
		}

		/* Streamed rather than collected. The reconcile below writes, so the
		 * addresses are read out first and the iterator closed before any of
		 * them are touched - but only the addresses are held, not the decoded
		 * infusions, so what this reactor costs is its delegator count in
		 * strings rather than in records. An upgrade runs against live
		 * cardinality with an infinite gas meter; the work is unavoidable, but
		 * holding all of it at once is not, and exhausting memory during an
		 * upgrade block is a worse failure than a slow one.
		 */
		type pendingInfusion struct {
			Address string
			Fuel    uint64
		}
		var infusions []pendingInfusion
		k.IterateInfusionsByDestination(ctx, reactor.Id, func(infusion structstypes.Infusion) {
			infusions = append(infusions, pendingInfusion{Address: infusion.Address, Fuel: infusion.Fuel})
		})

		for _, before := range infusions {
			delegatorAddress, addrErr := sdk.AccAddressFromBech32(before.Address)
			if addrErr != nil {
				logger.Error("skipping infusion with unparsable delegator address",
					"reactorId", reactor.Id, "address", before.Address, "error", addrErr)
				continue
			}

			visited++

			// Load-never-create: a delegator address that is no longer
			// registered must not have a player invented for it here.
			k.ReconcileExistingInfusionForDelegation(ctx, delegatorAddress, validatorAddress)

			after, found := k.GetInfusion(ctx, reactor.Id, before.Address)
			if !found || after.Fuel == before.Fuel {
				continue
			}

			changed++
			if before.Fuel > after.Fuel {
				fuelReleased += before.Fuel - after.Fuel
			}
		}
	}

	logger.Info("reactor infusion fuel recomputed against the truncating conversion",
		"infusionsVisited", visited, "infusionsChanged", changed, "fuelReleased", fuelReleased)

	return nil
}

func NewUpgrade() upgrades.Upgrade {
	return upgrades.Upgrade{
		UpgradeName:          UpgradeName,
		CreateUpgradeHandler: CreateUpgradeHandler,
		StoreUpgrades:        storetypes.StoreUpgrades{},
	}
}

/* MigrateGridCascadeQueue re-keys the pending cascade queue by sequence.
 *
 * GridCascade no longer drains the queue to exhaustion, so what it does not
 * reach has to wait - and the order it waits in is now a safety property rather
 * than an implementation detail. Ordering by object id would let an attacker
 * hold their own over-subscribed substation at the back of the line forever by
 * keeping cheaper-sorting entries in front of it.
 *
 * The old rows carry no ordering of their own, so they are re-appended sorted,
 * which is exactly the order the old cascade would have processed them in.
 */
func MigrateGridCascadeQueue(ctx context.Context, keepers *upgrades.Keepers) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	logger := sdkCtx.Logger().With("upgrade", UpgradeName, "phase", "migrateGridCascadeQueue")

	migrated, err := keepers.StructsKeeper.MigrateGridCascadeQueueToSequence(ctx)
	if err != nil {
		return err
	}

	logger.Info("grid cascade queue re-keyed", "entries", migrated)
	return nil
}

/* MigrateSelfDefenseRegistrations clears any struct registered as its own
 * defender.
 *
 * StructDefenseSet never compared the two ids, and IsProtecting compares
 * locations - a struct is trivially co-located with itself - so the pairing was
 * accepted. The consequence is in the resolution order: StructAttack runs
 * defender counters, then the volley, then the target's own counter, and a
 * self-registered target is picked up in the first pass. Its counter therefore
 * landed before the volley that provoked it, and a counter that destroyed the
 * attacker voided the whole volley, so the target took nothing. The documented
 * rule is that a target counters only after surviving the shots.
 *
 * ResolveDefenders now skips either combatant, so an unmigrated row is already
 * inert. This removes it anyway: the row is also a live registration slot -
 * SetStructDefender lets a struct protect only one target at a time - so leaving
 * it in place would keep that struct's real defensive assignment blocked.
 */
func MigrateSelfDefenseRegistrations(ctx context.Context, keepers *upgrades.Keepers) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	logger := sdkCtx.Logger().With("upgrade", UpgradeName, "phase", "migrateSelfDefenseRegistrations")

	k := keepers.StructsKeeper

	var cleared int
	for _, defender := range k.GetAllStructDefenderExport(ctx) {
		if defender == nil || defender.DefendingStructId != defender.ProtectedStructId {
			continue
		}

		k.ClearStructDefender(ctx, defender.ProtectedStructId, defender.DefendingStructId)
		cleared++

		logger.Info("cleared self-defense registration", "structId", defender.DefendingStructId)
	}

	logger.Info("self-defense registrations cleared", "count", cleared)
	return nil
}
