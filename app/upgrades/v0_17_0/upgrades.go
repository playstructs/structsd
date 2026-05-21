package v0_17_0

import (
	"context"

	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	"structs/app/upgrades"
	structstypes "structs/x/structs/types"
)

// CreateUpgradeHandler returns the v0.17.0 upgrade handler.
//
// Beyond the binary-only ante / handler hardening described in constants.go,
// this upgrade carries two one-time state recoveries at upgrade height:
//
//  1. Stuck "defusing" reactor infusions left behind by the missing
//     AfterUnbondingComplete hook in Cosmos SDK v0.53. See
//     docs/incident-2026-05-defusing.md for the full write-up. The recovery
//     walks every reactor-type Infusion, calls ReconcileInfusionForDelegation
//     against the live Cosmos staking state to clear stale Defusing > 0,
//     and bootstraps the new InfusionMaturitySweepQueue from any still-live
//     UBDs so the EndBlocker keeps reconciling them post-upgrade.
//
//  2. Orphan grid-attribute rows whose key does not match the canonical
//     "<gridAttributeType>-<objectType>-<objectId>" format. See
//     docs/incident-2026-05-grid-orphan.md. The pre-daac34c
//     AutoResizeAllocation flow could write to "2-" when an automated
//     allocation had no destination set yet; one such row exists on
//     structstestnet-111 with value 840000. The keeper-level guard added in
//     this release prevents new rows from being written; this migration
//     cleans up the historical leak.
//
// Both recoveries are idempotent: a double-run reconciles the same set of
// infusions to the same state, writes the same queue rows, and finds no
// remaining malformed grid keys to prune.
func CreateUpgradeHandler(
	mm *module.Manager,
	configurator module.Configurator,
	keepers *upgrades.Keepers,
) upgradetypes.UpgradeHandler {
	return func(ctx context.Context, _ upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
		newVM, err := mm.RunMigrations(ctx, configurator, fromVM)
		if err != nil {
			return newVM, err
		}

		if err := MigrateDefusingInfusions(ctx, keepers); err != nil {
			return newVM, err
		}

		if err := MigrateMalformedGridAttributes(ctx, keepers); err != nil {
			return newVM, err
		}

		return newVM, nil
	}
}

// MigrateDefusingInfusions performs the two-step recovery documented above.
// Exported so the migration can be exercised by upgrade-handler tests
// independently of module.Manager wiring.
//
// Failures on any individual infusion are logged but do not abort the upgrade
// (the EndBlocker sweep gives us a continuous self-heal mechanism going
// forward, so an upgrade-time partial recovery still leaves the chain in a
// good state).
//
// Idempotency: this function is safe to invoke more than once on the same
// state. ReconcileInfusionForDelegation is a pure projection of the live
// Cosmos staking state onto the infusion record (it recomputes Fuel and
// Defusing from scratch rather than incrementing), and
// EnqueueInfusionMaturitySweep writes to a (CompletionTime, infusionKey) map
// key so re-enqueueing the same UBD entry overwrites with the same value.
// This matters in two scenarios operators commonly hit:
//
//   - A validator restarts from a snapshot taken just before the upgrade
//     height and replays the upgrade block.
//   - An operator runs the migration manually on a recovered snapshot before
//     joining the live network (e.g. as part of incident triage).
//
// In both cases re-running the migration produces the same Defusing values
// and the same maturity-queue contents byte-for-byte.
func MigrateDefusingInfusions(ctx context.Context, keepers *upgrades.Keepers) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	logger := sdkCtx.Logger().With("upgrade", UpgradeName, "phase", "migrateDefusingInfusions")

	allInfusions := keepers.StructsKeeper.GetAllInfusion(ctx)

	var (
		scanned       int
		reconciled    int
		queueRowsAdded int
	)

	for _, infusion := range allInfusions {
		if infusion.DestinationType != structstypes.ObjectType_reactor {
			continue
		}
		if infusion.Defusing == 0 {
			continue
		}
		scanned++

		reactor, found := keepers.StructsKeeper.GetReactor(ctx, infusion.DestinationId)
		if !found {
			logger.Warn("reactor missing for defusing infusion; skipping", "infusionDestination", infusion.DestinationId, "address", infusion.Address)
			continue
		}

		playerAddress, err := sdk.AccAddressFromBech32(infusion.Address)
		if err != nil {
			logger.Warn("invalid delegator address on infusion", "address", infusion.Address, "error", err)
			continue
		}
		validatorAddress, err := sdk.ValAddressFromBech32(reactor.Validator)
		if err != nil {
			logger.Warn("invalid validator address on reactor", "reactor", reactor.Id, "validator", reactor.Validator, "error", err)
			continue
		}

		// Bootstrap the maturity queue from any UBD entries still on file
		// before we reconcile, so the EndBlocker sweep continues handling
		// them post-upgrade. We go through the structs keeper's staking
		// adapter rather than keepers.StakingKeeper directly so that the
		// migration is testable with the same mock staking keeper used by
		// the rest of the structs unit tests.
		ubd, err := keepers.StructsKeeper.StakingKeeper().GetUnbondingDelegation(ctx, playerAddress, validatorAddress)
		if err == nil {
			infusionId := reactor.Id + "-" + playerAddress.String()
			outstanding := math.ZeroInt()
			for _, entry := range ubd.Entries {
				keepers.StructsKeeper.EnqueueInfusionMaturitySweep(ctx, entry.CompletionTime, infusionId)
				outstanding = outstanding.Add(entry.Balance)
				queueRowsAdded++
			}
			logger.Info("bootstrapped maturity queue rows for defusing infusion", "infusionId", infusionId, "entries", len(ubd.Entries), "outstandingBalance", outstanding.String())
		}

		// Now reconcile. If all UBDs already matured, this clears Defusing
		// to zero and (if Power is also zero) enqueues the infusion for
		// destruction in the next EndBlocker.
		keepers.StructsKeeper.ReconcileInfusionForDelegation(ctx, playerAddress, validatorAddress)
		reconciled++
	}

	logger.Info("v0.17.0 defusing-infusion recovery complete", "infusionsScanned", scanned, "infusionsReconciled", reconciled, "maturityQueueRowsAdded", queueRowsAdded)
	return nil
}

// MigrateMalformedGridAttributes deletes any grid-attribute rows whose key
// does not match the canonical "<int>-<int>-<int>" format. See
// docs/incident-2026-05-grid-orphan.md for the originating incident
// (testnet had a single "2-" row at 840000 from a pre-daac34c
// AutoResizeAllocation write with an empty destination).
//
// Idempotent: a re-run finds no malformed rows and returns nil.
//
// Failures cannot occur today (the underlying keeper helper does not return
// an error) but the signature mirrors MigrateDefusingInfusions so we can
// tighten this later without churning the upgrade-handler wiring.
func MigrateMalformedGridAttributes(ctx context.Context, keepers *upgrades.Keepers) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	logger := sdkCtx.Logger().With("upgrade", UpgradeName, "phase", "migrateMalformedGridAttributes")

	pruned := keepers.StructsKeeper.PruneMalformedGridAttributes(ctx)
	if len(pruned) == 0 {
		logger.Info("no malformed grid-attribute rows found")
		return nil
	}

	for _, key := range pruned {
		logger.Info("pruned malformed grid-attribute row", "gridAttributeId", key)
	}
	logger.Info("v0.17.0 malformed-grid-attribute recovery complete", "rowsPruned", len(pruned))
	return nil
}

func NewUpgrade() upgrades.Upgrade {
	return upgrades.Upgrade{
		UpgradeName:          UpgradeName,
		CreateUpgradeHandler: CreateUpgradeHandler,
		StoreUpgrades:        storetypes.StoreUpgrades{},
	}
}
