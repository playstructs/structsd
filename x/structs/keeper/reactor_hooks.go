package keeper

import (

	sdk "github.com/cosmos/cosmos-sdk/types"

	"context"

	"structs/x/structs/types"
	"cosmossdk.io/math"

	"fmt"

)


/* Setup Reactor (when a validator is created)
 *
 * Triggered during Staking Hooks:
 *   AfterValidatorCreated
 */
func (k Keeper) ReactorInitialize(ctx context.Context, validatorAddress sdk.ValAddress)  {

    /* Does this Reactor exist? */
    var reactor types.Reactor
    reactorBytes, reactorBytesFound := k.GetReactorBytesFromValidator(ctx, validatorAddress.Bytes())
    if reactorBytesFound {
        reactor, _ = k.GetReactorByBytes(ctx, reactorBytes, false)
    } else {
        /* Build the initial Reactor object */
        reactor = types.CreateEmptyReactor()
        reactor.Validator = validatorAddress.String()
        reactor.RawAddress = validatorAddress.Bytes()

        /*
         * Commit Reactor to the Keeper
         */
        reactor.DefaultCommission, _ = math.LegacyNewDecFromStr("0.04")
        reactor := k.AppendReactor(ctx, reactor)

        k.SetReactorValidatorBytes(ctx, reactor.Id, validatorAddress.Bytes())


        /*
         * Convert the sdk.ValAddress into a regular sdk.AccAddress
         *
         * This will allow us to create a player account with the correct permissions
         */

        var identity sdk.AccAddress
        identity = validatorAddress.Bytes()
        player := k.UpsertPlayer(ctx, identity.String(), false)

        // Add the player as a permissioned user of the reactor
        permissionId := GetObjectPermissionIDBytes(reactor.Id, player.Id)
        k.PermissionAdd(ctx, permissionId, types.PermissionAll)

        // TODO apply the energy distribution to the reactor player account
        delegation, err := k.stakingKeeper.GetDelegation(ctx, identity, validatorAddress)
        if (err == nil) {
            validator, _ := k.stakingKeeper.GetValidator(ctx, validatorAddress)
            delegationShare := ((delegation.Shares.Quo(validator.DelegatorShares)).Mul(math.LegacyNewDecFromInt(validator.Tokens))).RoundInt()

            k.UpsertInfusion(ctx, types.ObjectType_reactor, reactor.Id, identity.String(), player, delegationShare.Uint64(), reactor.DefaultCommission)
        }

    }

}


/* Change Reactor Allocations for Player Delegations
 *
 * Triggered during Staking Hooks:
 *   AfterDelegationModified
 *
 */
func (k Keeper) ReactorUpdatePlayerAllocation(ctx context.Context, playerAddress sdk.AccAddress, validatorAddress sdk.ValAddress) {

	/* Does this Reactor exist? */
	reactorBytes, reactorBytesFound := k.GetReactorBytesFromValidator(ctx, validatorAddress.Bytes())
	if !reactorBytesFound {
        return
	}
    reactor, _ := k.GetReactorByBytes(ctx, reactorBytes, false)
	validator, _ := k.stakingKeeper.GetValidator(ctx, validatorAddress)


    delegation, err := k.stakingKeeper.GetDelegation(ctx, playerAddress, validatorAddress)


    if (err == nil) {

        delegationShare := ((delegation.Shares.Quo(validator.DelegatorShares)).Mul(math.LegacyNewDecFromInt(validator.Tokens))).RoundInt()
        player := k.UpsertPlayer(ctx, playerAddress.String(), true)

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
        k.UpsertInfusion(ctx, types.ObjectType_reactor, reactor.Id, playerAddress.String(), player, delegationShare.Uint64(), reactor.DefaultCommission)

    }
}


/* Update Reactor Details (Primarily In-Game Permissions/Ownership)
 *
 * Triggered during Staking Hooks:
 *   BeforeValidatorModified (Ugh, why isn't this AfterValidatorModified)
 *
 */
func (k Keeper) ReactorUpdateFromValidator(ctx context.Context, validatorAddress sdk.ValAddress)  {

    // Currently no need to run updates after the Validator Description is updated
    // but we may use this in the future

}


func (k Keeper) ReactorRemoveInfusion(ctx context.Context, unbondingId uint64) {
    fmt.Printf("New Unbonding Request %d \n", unbondingId)
    unbondingDelegation, err := k.stakingKeeper.GetUnbondingDelegationByUnbondingID(ctx, unbondingId)

    fmt.Printf("Delegator Address: %s \n", unbondingDelegation.DelegatorAddress)
    fmt.Printf("Validator Address: %s \n", unbondingDelegation.ValidatorAddress)

    if (err == nil) {
        var playerAddress sdk.AccAddress
        playerAddress, _ = sdk.AccAddressFromBech32(unbondingDelegation.DelegatorAddress)
        var validatorAddress sdk.ValAddress
        validatorAddress, _ = sdk.ValAddressFromBech32(unbondingDelegation.ValidatorAddress)

        /* Does this Reactor exist? It really should... */
        reactorBytes, _ := k.GetReactorBytesFromValidator(ctx, validatorAddress.Bytes())
        reactor, _ := k.GetReactorByBytes(ctx, reactorBytes, false)

        unbondingInfusion, _ := k.GetInfusion(ctx, reactor.Id, playerAddress.String())

        amount := math.ZeroInt()
        for _, entry := range unbondingDelegation.Entries {
            amount = amount.Add(entry.InitialBalance)
        }

        unbondingInfusion.Defusing = amount
        k.SetInfusion(ctx, unbondingInfusion)
    }
}
