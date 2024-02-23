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
         * Convert the sdk.ValAddress into a regular sdk.AccAddress
         *
         * This will allow us to create a player account with the correct permissions
         */

        var identity sdk.AccAddress
        identity = validatorAddress.Bytes()
        player := k.UpsertPlayer(ctx, identity.String())

        // Add the player as a permissioned user of the reactor
        k.ReactorPermissionAdd(ctx, reactor.Id, player.Id, types.ReactorPermissionAll)


        // Build the Primary Substation
        // This will be unpowered at first since there is likely no
        // delegations of fuel to the reactor at this phase.
        substation := k.AppendSubstation(ctx, types.InitialReactorOwnerEnergy, player)

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
func (k Keeper) ReactorUpdatePlayerAllocation(ctx sdk.Context, playerAddress sdk.AccAddress, validatorAddress sdk.ValAddress) {

	/* Does this Reactor exist? */
	reactorBytes, reactorBytesFound := k.GetReactorBytesFromValidator(ctx, validatorAddress.Bytes())
	if !reactorBytesFound {
        return
	}
    reactor, _ := k.GetReactorByBytes(ctx, reactorBytes, false)
	validator, _ := k.stakingKeeper.GetValidator(ctx, validatorAddress)


    delegation, delegationFound := k.stakingKeeper.GetDelegation(ctx, playerAddress, validatorAddress)


    if (delegationFound) {

        delegationShare := ((delegation.Shares.Quo(validator.DelegatorShares)).Mul(math.LegacyNewDecFromInt(validator.Tokens))).RoundInt()

        player := k.UpsertPlayer(ctx, playerAddress.String())


        /*
         * Returns if needed (
               infusion types.Infusion,
               newInfusionFuel uint64,
               oldInfusionFuel uint64,
               newInfusionPower uint64,
               oldInfusionPower uint64,
               newCommissionPower uint64,
               oldCommissionPower uint64,
               newPlayerPower uint64,
               oldPlayerPower uint64,
               err error
           )
        */
        k.UpsertInfusion(ctx, types.ObjectType_reactor, reactor.Id, player, delegationShare.Uint64(), reactor.DefaultCommission)

    }
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


func (k Keeper) ReactorRemoveInfusion(ctx sdk.Context, unbondingId uint64) {

    unbondingDelegation , unbondingDelegationFound := k.stakingKeeper.GetUnbondingDelegationByUnbondingID(ctx, unbondingId)
    if (unbondingDelegationFound) {

        var playerAddress sdk.AccAddress
        playerAddress, _ = sdk.AccAddressFromBech32(unbondingDelegation.DelegatorAddress)
        var validatorAddress sdk.ValAddress
        validatorAddress, _ = sdk.ValAddressFromBech32(unbondingDelegation.ValidatorAddress)

        /* Does this Reactor exist? It really should... */
        reactorBytes, _ := k.GetReactorBytesFromValidator(ctx, validatorAddress.Bytes())
        reactor, _ := k.GetReactorByBytes(ctx, reactorBytes, false)


        if (unbondingDelegationFound) {
            unbondingInfusion, _ := k.GetInfusion(ctx, types.ObjectType_reactor, reactor.Id, playerAddress.String())
            k.DestroyInfusion(ctx, unbondingInfusion)
        }

        k.CascadeReactorAllocationFailure(ctx, reactor)

    }

}
