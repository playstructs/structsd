package keeper

import (

	sdk "github.com/cosmos/cosmos-sdk/types"
	"structs/x/structs/types"
	"cosmossdk.io/math"
)

/* Setup Reactor (when a validator is created)
 *
 * Triggered during Staking Hooks:
 *   AfterValidatorCreated
 */
func (k Keeper) ReactorInitialize(ctx sdk.Context, validatorAddress sdk.ValAddress) (reactor types.Reactor) {

	validator, _ := k.stakingKeeper.GetValidator(ctx, validatorAddress)

	/* Does this Reactor exist? */
	reactorBytes, reactorBytesFound := k.GetReactorBytesFromValidator(ctx, validatorAddress.String())
	if reactorBytesFound {
		reactor, _ = k.GetReactorByBytes(ctx, reactorBytes, false)
	} else {
		/* Build the initial Reactor object */
		reactor = types.CreateEmptyReactor()
		reactor.SetValidator(validatorAddress.String())
		k.SetReactorValidatorBytes(ctx, reactor.Id, validatorAddress.String())


		/*
		 * Commit Reactor to the Keeper
		 */
		reactorId := k.AppendReactor(ctx, reactor)
        reactor.SetId(reactorId)

        activated, _ := k.ReactorActivate(ctx, reactor, validator)
        if (activated) {
            reactor.SetActivated(true)
            k.SetReactor(ctx, reactor)
        }



	}
	return reactor
}


/* Change Reactor Allocations for Player Delegations
 *
 * Triggered during Staking Hooks:
 *   AfterDelegationModified
 *
 */
func (k Keeper) ReactorUpdatePlayerAllocation(ctx sdk.Context, playerAddress sdk.AccAddress, validatorAddress sdk.ValAddress) (reactor types.Reactor) {

	/* Does this Reactor exist? */
	reactorBytes, reactorBytesFound := k.GetReactorBytesFromValidator(ctx, validatorAddress.String())
	if !reactorBytesFound {
        return
	}
    reactor, _ = k.GetReactorByBytes(ctx, reactorBytes, false)
	validator, _ := k.stakingKeeper.GetValidator(ctx, validatorAddress)

    delegation, _ := k.stakingKeeper.GetDelegation(ctx, playerAddress, validatorAddress)

    if validator.GetDelegatorShares().IsZero() {
        k.DestroyInfusion(ctx, playerAddress, validatorAddress)

        return
    }

    delegationShare := ((delegation.Shares.Quo(validator.DelegatorShares)).Mul(math.LegacyNewDecFromInt(validator.Tokens))).RoundInt()

    infusion := k.UpsertInfusion()


    k.SetInfusion(ctx, infusion)





    // need to define a guild or reactor parameter for allowed allocation of contributions
    // need to define a guild or reactor parameter for minimum contributions before allocations are allowed


	// Update the connected Substations with the new details
	k.CascadeReactorAllocationFailure(ctx, reactor)


	return reactor
}


/* Update Reactor Details (Primarily In-Game Permissions/Ownership)
 *
 * Triggered during Staking Hooks:
 *   BeforeValidatorModified (Ugh, why isn't this AfterValidatorModified)
 *
 */
func (k Keeper) ReactorUpdateFromValidator(ctx sdk.Context, validatorAddress sdk.ValAddress) (reactor types.Reactor) {

	validator, _ := k.stakingKeeper.GetValidator(ctx, validatorAddress)

	/* Does this Reactor exist? */
	reactorBytes, reactorBytesFound := k.GetReactorBytesFromValidator(ctx, validatorAddress.String())
	if reactorBytesFound {
		reactor, _ = k.GetReactorByBytes(ctx, reactorBytes, false)
	}



	/* Sync Energy Levels
	 *
	 * May as well update power levels while we are here
	 */
	reactor.SetEnergy(validator)

	k.SetReactor(ctx, reactor)

    activated, _ := k.ReactorActivate(ctx, reactor, validator)
    if (activated) {
        reactor.SetActivated(true)
        k.SetReactor(ctx, reactor)
    }


	// Update the connected Substations with the new details
	k.CascadeReactorAllocationFailure(ctx, reactor)

	return reactor
}
