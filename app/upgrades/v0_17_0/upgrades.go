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
// this upgrade also carries a one-time state recovery for stuck "defusing"
// reactor infusions left behind by the missing AfterUnbondingComplete hook
// in Cosmos SDK v0.53. See docs/incident-2026-05-defusing.md for the full
// write-up. The recovery does two things at upgrade height:
//
//  1. Walks every reactor-type Infusion and calls
//     ReconcileInfusionForDelegation against the live Cosmos staking state.
//     This clears any stale Defusing > 0 left over from UBDs that already
//     matured silently in x/staking's EndBlocker. Empty infusions get
//     enqueued for destruction by the helper and reaped by the next
//     EndBlocker.
//
//  2. Bootstraps the new InfusionMaturitySweepQueue from the still-live
//     UBDs of every defusing infusion so the EndBlocker sweep keeps
//     reconciling them as they mature post-upgrade.
//
// The recovery is idempotent: a double-run reconciles the same set of
// infusions to the same state and writes the same queue rows.
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

		return newVM, nil
	}
}

// MigrateDefusingInfusions performs the two-step recovery documented above.
// Exported so the migration can be exercised by upgrade-handler tests
// independently of module.Manager wiring.
// Failures on any individual infusion are logged but do not abort the upgrade
// (the EndBlocker sweep gives us a continuous self-heal mechanism going
// forward, so an upgrade-time partial recovery still leaves the chain in a
// good state).
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

func NewUpgrade() upgrades.Upgrade {
	return upgrades.Upgrade{
		UpgradeName:          UpgradeName,
		CreateUpgradeHandler: CreateUpgradeHandler,
		StoreUpgrades:        storetypes.StoreUpgrades{},
	}
}
