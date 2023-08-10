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


    delegationShare := ((delegation.Shares.Quo(validator.DelegatorShares)).Mul(math.LegacyNewDecFromInt(validator.Tokens))).RoundInt()


    _, allocation, withDynamicAllocation := k.UpsertInfusion(ctx, types.ObjectType_reactor, reactor.Id, playerAddress.String(), delegationShare.Uint64(), reactor.AutomatedAllocations, reactor.DelegateTaxOnAllocations)


    if (reactor.AutomatedAllocations && withDynamicAllocation) {
        playerId := k.GetPlayerIdFromAddress(ctx, playerAddress.String())

        var player types.Player
        if (playerId == 0) {
            // No Player Found, Creating..
            player = types.CreateEmptyPlayer()
            player.SetCreator(playerAddress.String())
            playerId = k.AppendPlayer(ctx, player)
            player.SetId(playerId)
        } else {
            player, _ = k.GetPlayer(ctx, playerId)
        }

        // Now let's get the player some power
        if (player.SubstationId == 0) {
            var substation types.Substation
            substation = types.CreateEmptySubstation()
            substationId := k.GetNextSubstationId(ctx)
            substation.SetId(substationId)
            substation.SetOwner(playerId)
            substation.SetCreator(player.Creator)
            substation.SetPlayerConnectionAllocation(types.InitialReactorOwnerEnergy)
            k.SetSubstation(ctx, substation)

            k.SubstationPermissionAdd(ctx, substation.Id, playerId, types.SubstationPermissionAll)

            // Connect Allocation to Substation
            allocation.Connect(substation.Id)
            _ = k.SubstationIncrementEnergy(ctx, substation.Id, allocation.Power)

            // Connect Player to Substation
            k.SubstationIncrementConnectedPlayerLoad(ctx, substation.Id, 1)
            player.SetSubstation(substation.Id)
            k.SetPlayer(ctx, player)
        }
    }

    // need to define a guild or reactor parameter for allowed allocation of contributions
    // need to define a guild or reactor parameter for minimum contributions before allocations are allowed




	// Update the connected Substations with the new details
	//k.CascadeReactorAllocationFailure(ctx, reactor)


	return reactor
}


/* Update Reactor Details (Primarily In-Game Permissions/Ownership)
 *
 * Triggered during Staking Hooks:
 *   BeforeValidatorModified (Ugh, why isn't this AfterValidatorModified)
 *
 */
func (k Keeper) ReactorUpdateFromValidator(ctx sdk.Context, validatorAddress sdk.ValAddress)  {

	validator, _ := k.stakingKeeper.GetValidator(ctx, validatorAddress)

	/* Does this Reactor exist? */
	reactorBytes, reactorBytesFound := k.GetReactorBytesFromValidator(ctx, validatorAddress.String())
	if reactorBytesFound {
		reactor, _ := k.GetReactorByBytes(ctx, reactorBytes, false)

        activated, _ := k.ReactorActivate(ctx, reactor, validator)
        if (activated) {
            reactor.SetActivated(true)
            k.SetReactor(ctx, reactor)
        }

	}

}
