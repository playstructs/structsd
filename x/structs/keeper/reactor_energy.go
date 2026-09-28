package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	"structs/x/structs/types"
)

// reactorEnergyRatio permits output only from a live, bonded, unjailed validator.
func reactorEnergyRatio(validator stakingtypes.Validator, err error) uint64 {
	if err != nil {
		return 0
	}

	if validator.IsJailed() || !validator.IsBonded() {
		return 0
	}

	return types.ReactorFuelToEnergyConversion
}

// reactorSourceEligible checks current validator health without walking its
// infusions or changing stored commission capacity. Queue membership is not a
// health signal: a permissionless restart must not disable a healthy source.
// A real slash gates commission until accounting commits for its latest revision.
func (cc *CurrentContext) reactorSourceEligible(reactorID string) bool {

	revision, err := cc.k.GetReactorCommissionRevision(cc.ctx, reactorID)
	if err != nil || revision.Issued != revision.Certified {
		return false
	}
	reactor := cc.GetReactor(reactorID)
	if reactor.Deleted || reactor.CheckReactor() != nil {
		return false
	}
	address, err := sdk.ValAddressFromBech32(reactor.GetReactor().Validator)
	if err != nil {
		return false
	}
	validator, err := cc.k.stakingKeeper.GetValidator(cc.ctx, address)
	return err == nil && validator.IsBonded() && !validator.IsJailed()
}

/* ReactorGateEnergy is the legacy synchronous gate. Active-economy staking
 * hooks must queue cleanup instead. This helper also supports explicit fixture
 * setup; it zeroes stored ratios without changing fuel or the delegation.
 */
func (k Keeper) ReactorGateEnergy(ctx context.Context, validatorAddress sdk.ValAddress) {
	cc := k.NewCurrentContext(ctx)
	defer cc.CommitAll()

	reactorBytes, reactorBytesFound := k.GetReactorBytesFromValidator(ctx, validatorAddress.Bytes())
	if !reactorBytesFound {
		return
	}
	reactor, _ := k.GetReactorByBytes(ctx, reactorBytes)

	gated := 0
	for _, infusion := range cc.GetAllInfusionByDestination(reactor.Id) {
		if infusion.CheckInfusion() != nil {
			continue
		}

		if infusion.GetInfusion().Ratio == 0 {
			continue
		}

		infusion.SetRatio(0)
		gated++
	}

	if gated > 0 {
		k.logger.Info("Reactor energy gated", "validator", validatorAddress.String(), "reactorId", reactor.Id, "infusions", gated)
	}
}

// QueueReactorRestoreEnergy records a real bonding transition with constant
// work. A new transition must reset the infusion cursor: an intervening gate
// can have zeroed positions that an earlier pass already restored. Cross-reactor
// scheduling is independent, so that reset cannot monopolize the queue.
func (k Keeper) QueueReactorRestoreEnergy(ctx context.Context, validatorAddress sdk.ValAddress) {
	k.queueReactorEnergyReconcile(ctx, validatorAddress)
}

// Both departure and return reset the finite interval: either transition can
// affect rows already visited by an earlier pass. The worker always reads live
// health, so a delayed shutdown job cannot switch off a subsequently bonded row.
func (k Keeper) queueReactorEnergyReconcile(ctx context.Context, validatorAddress sdk.ValAddress) {
	reactorBytes, found := k.GetReactorBytesFromValidator(ctx, validatorAddress.Bytes())
	if !found {
		return
	}
	reactor, found := k.GetReactorByBytes(ctx, reactorBytes)
	if !found {
		return
	}
	k.EnqueueReactorSlashReconcile(ctx, reactor.Id)
}

/* ReactorRestoreEnergy reconciles against live staking state instead of
 * blindly restoring a ratio that could relight stale fuel.
 */
func (k Keeper) ReactorRestoreEnergy(ctx context.Context, validatorAddress sdk.ValAddress) {
	cc := k.NewCurrentContext(ctx)
	defer cc.CommitAll()

	reactorBytes, reactorBytesFound := k.GetReactorBytesFromValidator(ctx, validatorAddress.Bytes())
	if !reactorBytesFound {
		return
	}
	reactor, _ := k.GetReactorByBytes(ctx, reactorBytes)

	for _, infusion := range cc.GetAllInfusionByDestination(reactor.Id) {
		if infusion.CheckInfusion() != nil {
			continue
		}

		playerAddress, err := sdk.AccAddressFromBech32(infusion.GetInfusion().Address)
		if err != nil {
			k.logger.Warn("ReactorRestoreEnergy: invalid delegator address", "reactorId", reactor.Id, "address", infusion.GetInfusion().Address, "error", err)
			continue
		}

		if _, _, err := k.reconcileInfusionForDelegation(ctx, cc, playerAddress, validatorAddress); err != nil {
			k.logger.Error("ReactorRestoreEnergy: reconcile failed", "reactorId", reactor.Id, "address", playerAddress.String(), "error", err)
		}
	}
}
