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
func (k Keeper) ReactorInitialize(ctx sdk.Context, validatorAddress sdk.ValAddress)  {

	/* Does this Reactor exist? */
	var reactor types.Reactor
	reactorBytes, reactorBytesFound := k.GetReactorBytesFromValidator(ctx, validatorAddress.Bytes())
	if reactorBytesFound {
		reactor, _ = k.GetReactorByBytes(ctx, reactorBytes, false)
	} else {
		/* Build the initial Reactor object */
		reactor = types.CreateEmptyReactor()
		reactor.SetValidator(validatorAddress.String())
		reactor.SetRawAddress(validatorAddress.Bytes())

		/*
		 * Commit Reactor to the Keeper
		 */
		reactorId := k.AppendReactor(ctx, reactor)
		reactor.SetId(reactorId)
		k.SetReactorValidatorBytes(ctx, reactor.Id, validatorAddress.Bytes())


        /*
         * Reverse engineer the sdk.ValAddress into a regular sdk.AccAddress
         *
         * This will allow us to create a player account with the correct permissions
         */

        var identity sdk.AccAddress
        identity = validatorAddress.Bytes()
        playerId := k.GetPlayerIdFromAddress(ctx, identity.String())

        var player types.Player
        if (playerId == 0) {
            // No Player Found, Creating..
            player = types.CreateEmptyPlayer()
            player.SetCreator(identity.String())

            playerId = k.AppendPlayer(ctx, player)
            player.SetId(playerId)

            // TODO Add Related Address
        } else {
           player, _ = k.GetPlayer(ctx, playerId)
        }

        // Add the player as a permissioned user of the reactor
        k.ReactorPermissionAdd(ctx, reactor.Id, player.Id, types.ReactorPermissionAll)


        // Build the Primary Substation
        // This will be unpowered at first since there is likely no
        // delegations of fuel to the reactor at this phase.
        substation := types.CreateEmptySubstation()
        substation.SetOwner(playerId)
        substation.SetCreator(player.Creator)

        substationId := k.AppendSubstation(ctx, substation)
        substation.SetId(substationId)

        k.SubstationPermissionAdd(ctx, substation.Id, player.Id, types.SubstationPermissionAll)

        // Wasteful right now that we're writing this a couple times
        // to the keeper, but we'll clean it up later.
        reactor.SetServiceSubstationId(substation.Id)


        reactor.DelegateTaxOnAllocations, _ = math.LegacyNewDecFromStr("0.04")
        k.SetReactor(ctx, reactor)

	}
}


/* Change Reactor Allocations for Player Delegations
 *
 * Triggered during Staking Hooks:
 *   AfterDelegationModified
 *
 */
func (k Keeper) ReactorUpdatePlayerAllocation(ctx sdk.Context, playerAddress sdk.AccAddress, validatorAddress sdk.ValAddress) (reactor types.Reactor) {

	/* Does this Reactor exist? */
	reactorBytes, reactorBytesFound := k.GetReactorBytesFromValidator(ctx, validatorAddress.Bytes())
	if !reactorBytesFound {
        return
	}
    reactor, _ = k.GetReactorByBytes(ctx, reactorBytes, false)
	validator, _ := k.stakingKeeper.GetValidator(ctx, validatorAddress)

    delegation, _ := k.stakingKeeper.GetDelegation(ctx, playerAddress, validatorAddress)


    delegationShare := ((delegation.Shares.Quo(validator.DelegatorShares)).Mul(math.LegacyNewDecFromInt(validator.Tokens))).RoundInt()


    _, sourceAllocation, withDynamicSourceAllocation, playerAllocation, withDynamicPlayerAllocation := k.UpsertInfusion(ctx, types.ObjectType_reactor, reactor.Id, playerAddress.String(), delegationShare.Uint64(), reactor.AutomatedAllocations, reactor.DelegateTaxOnAllocations)

    if (reactor.AutomatedAllocations && withDynamicSourceAllocation) {
        // Connect Allocation to Substation
        sourceAllocation.Connect(reactor.ServiceSubstationId)
        _ = k.SubstationIncrementEnergy(ctx, reactor.ServiceSubstationId, sourceAllocation.Power)
        k.SetAllocation(ctx, sourceAllocation)
    }

    if (reactor.AutomatedAllocations && withDynamicPlayerAllocation) {
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
            playerAllocation.Connect(substation.Id)
            _ = k.SubstationIncrementEnergy(ctx, substation.Id, playerAllocation.Power)
            k.SetAllocation(ctx, playerAllocation)

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

    // Currently no need to run updates after the Validator Description is updated
    // but we may use this in the future

}
