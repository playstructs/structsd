package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	"structs/x/structs/types"
)

// reactorEnergyRatio fails closed when a validator is missing or jailed while
// leaving the infusion's stake-backed fuel untouched.
func reactorEnergyRatio(validator stakingtypes.Validator, err error) uint64 {
	if err != nil {
		return 0
	}

	if validator.IsJailed() {
		return 0
	}

	return types.ReactorFuelToEnergyConversion
}

/* ReactorGateEnergy zeroes every stored infusion's ratio without changing its
 * fuel, defusing balance, or underlying delegation.
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

		k.reconcileInfusionForDelegation(ctx, cc, playerAddress, validatorAddress)
	}
}
