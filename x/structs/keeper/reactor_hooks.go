package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"context"

	"structs/x/structs/types"

	"cosmossdk.io/math"

)

/* Setup Reactor (when a validator is created)
 *
 * Triggered during Staking Hooks:
 *   AfterValidatorCreated
 */
func (k Keeper) ReactorInitialize(ctx context.Context, validatorAddress sdk.ValAddress) {

	/* Does this Reactor exist? */
	var reactor types.Reactor
	reactorBytes, reactorBytesFound := k.GetReactorBytesFromValidator(ctx, validatorAddress.Bytes())
	if reactorBytesFound {
		reactor, _ = k.GetReactorByBytes(ctx, reactorBytes)
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
		player := k.UpsertPlayer(ctx, identity.String())

		// Add the player as a permissioned user of the reactor
		permissionId := GetObjectPermissionIDBytes(reactor.Id, player.Id)
		k.PermissionAdd(ctx, permissionId, types.PermissionAll)

		// TODO apply the energy distribution to the reactor player account
		delegation, err := k.stakingKeeper.GetDelegation(ctx, identity, validatorAddress)
		if err == nil {
			validator, _ := k.stakingKeeper.GetValidator(ctx, validatorAddress)
			delegationShare := ((delegation.Shares.Quo(validator.DelegatorShares)).Mul(math.LegacyNewDecFromInt(validator.Tokens))).RoundInt()

            infusion := k.GetInfusionCache(ctx, types.ObjectType_reactor, reactor.Id, identity.String())

            infusion.SetRatio(types.ReactorFuelToEnergyConversion)
            infusion.SetFuelAndCommission(delegationShare.Uint64(), reactor.DefaultCommission)
            infusion.Commit()

		}
	}

}

/* Change Reactor Allocations for Player Delegations
 *
 * Triggered during Staking Hooks:
 *   AfterDelegationModified
 *
 */
func (k Keeper) ReactorUpdatePlayerInfusion(ctx context.Context, playerAddress sdk.AccAddress, validatorAddress sdk.ValAddress) {

	/* Does this Reactor exist? */
	reactorBytes, reactorBytesFound := k.GetReactorBytesFromValidator(ctx, validatorAddress.Bytes())
	if !reactorBytesFound {
		return
	}
	reactor, _ := k.GetReactorByBytes(ctx, reactorBytes)
	validator, _ := k.stakingKeeper.GetValidator(ctx, validatorAddress)

	delegation, err := k.stakingKeeper.GetDelegation(ctx, playerAddress, validatorAddress)

	if err == nil {

		delegationShare := ((delegation.Shares.Quo(validator.DelegatorShares)).Mul(math.LegacyNewDecFromInt(validator.Tokens))).RoundInt()
		k.UpsertPlayer(ctx, playerAddress.String())

        infusion := k.GetInfusionCache(ctx, types.ObjectType_reactor, reactor.Id, playerAddress.String())

        infusion.SetRatio(types.ReactorFuelToEnergyConversion)
        infusion.SetFuelAndCommission(delegationShare.Uint64(), reactor.DefaultCommission)
        infusion.Commit()
	}

}


/* Update Reactor Details (Primarily In-Game Permissions/Ownership)
 *
 * Triggered during Staking Hooks:
 *   BeforeValidatorModified (Ugh, why isn't this AfterValidatorModified)
 *
 */
func (k Keeper) ReactorUpdateFromValidator(ctx context.Context, validatorAddress sdk.ValAddress) {

	// Currently no need to run updates after the Validator Description is updated
	// but we may use this in the future

}

func (k Keeper) ReactorInfusionUnbonding(ctx context.Context, unbondingId uint64) {

	unbondingDelegation, err := k.stakingKeeper.GetUnbondingDelegationByUnbondingID(ctx, unbondingId)

    k.logger.Info("Unbonding Request", "unbondingId", unbondingId, "delegator", unbondingDelegation.DelegatorAddress, "validator", unbondingDelegation.ValidatorAddress)

	if err == nil {
		var playerAddress sdk.AccAddress
		playerAddress, _ = sdk.AccAddressFromBech32(unbondingDelegation.DelegatorAddress)
		var validatorAddress sdk.ValAddress
		validatorAddress, _ = sdk.ValAddressFromBech32(unbondingDelegation.ValidatorAddress)

		/* Does this Reactor exist? It really should... */
		reactorBytes, _ := k.GetReactorBytesFromValidator(ctx, validatorAddress.Bytes())
		reactor, _ := k.GetReactorByBytes(ctx, reactorBytes)

        infusion := k.GetInfusionCache(ctx, types.ObjectType_reactor, reactor.Id, playerAddress.String())

        infusion.SetRatio(types.ReactorFuelToEnergyConversion)
        infusion.SetCommission(reactor.DefaultCommission)

		amount := math.ZeroInt()
		for _, entry := range unbondingDelegation.Entries {
			amount = amount.Add(entry.Balance) // should this be entry.InitialBalance?
		}
		infusion.SetDefusing(amount.Uint64())

        delegation, err := k.stakingKeeper.GetDelegation(ctx, playerAddress, validatorAddress)
        if err == nil {
            validator, _ := k.stakingKeeper.GetValidator(ctx, validatorAddress)
            delegationShare := ((delegation.Shares.Quo(validator.DelegatorShares)).Mul(math.LegacyNewDecFromInt(validator.Tokens))).RoundInt()
            infusion.SetFuel(delegationShare.Uint64())
        } else {
            infusion.SetFuel(uint64(0))
        }

        infusion.Commit()

		uctx := sdk.UnwrapSDKContext(ctx)
		_ = uctx.EventManager().EmitTypedEvent(&types.EventAlphaInfuse{&types.EventAlphaInfuseDetail{PrimaryAddress: unbondingDelegation.DelegatorAddress, Amount: amount.Uint64()}})

	}
}
