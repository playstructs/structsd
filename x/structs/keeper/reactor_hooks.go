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
	cc := k.NewCurrentContext(ctx)
	defer cc.CommitAll()

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
		reactor = k.AppendReactor(ctx, reactor)

		k.SetReactorValidatorBytes(ctx, reactor.Id, validatorAddress.Bytes())

		/*
		 * Convert the sdk.ValAddress into a regular sdk.AccAddress
		 *
		 * This will allow us to create a player account with the correct permissions
		 */

		var identity sdk.AccAddress
		identity = validatorAddress.Bytes()
		player := cc.UpsertPlayer(identity.String())

		// Add the player as a permissioned user of the reactor
		permissionId := GetObjectPermissionIDBytes(reactor.Id, player.Id)
		cc.PermissionAdd(permissionId, types.PermissionAll)

		// TODO apply the energy distribution to the reactor player account
		delegation, err := k.stakingKeeper.GetDelegation(ctx, identity, validatorAddress)
		if err == nil {
			validator, _ := k.stakingKeeper.GetValidator(ctx, validatorAddress)
			delegationShare := ((delegation.Shares.Quo(validator.DelegatorShares)).Mul(math.LegacyNewDecFromInt(validator.Tokens))).RoundInt()

			infusion := cc.GetInfusion(types.ObjectType_reactor, reactor.Id, identity.String())

			infusion.SetRatio(types.ReactorFuelToEnergyConversion)
			infusion.SetFuelAndCommission(delegationShare.Uint64(), reactor.DefaultCommission)
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
	cc := k.NewCurrentContext(ctx)
	defer cc.CommitAll()

	/* Does this Reactor exist? */
	reactorBytes, reactorBytesFound := k.GetReactorBytesFromValidator(ctx, validatorAddress.Bytes())
	if !reactorBytesFound {
		return
	}
	reactor, _ := k.GetReactorByBytes(ctx, reactorBytes)
	validator, _ := k.stakingKeeper.GetValidator(ctx, validatorAddress)

	cc.UpsertPlayer(playerAddress.String())
	infusion := cc.GetInfusion(types.ObjectType_reactor, reactor.Id, playerAddress.String())

	delegation, err := k.stakingKeeper.GetDelegation(ctx, playerAddress, validatorAddress)

	if err == nil {

		delegationShare := ((delegation.Shares.Quo(validator.DelegatorShares)).Mul(math.LegacyNewDecFromInt(validator.Tokens))).RoundInt()

		infusion.SetRatio(types.ReactorFuelToEnergyConversion)
		infusion.SetFuelAndCommission(delegationShare.Uint64(), reactor.DefaultCommission)
	}

	unbondingDelegation, err := k.stakingKeeper.GetUnbondingDelegation(ctx, playerAddress, validatorAddress)
	amount := math.ZeroInt()
	if err == nil {
		for _, entry := range unbondingDelegation.Entries {
			amount = amount.Add(entry.Balance)
		}
	}
	if infusion.GetDefusing() != amount.Uint64() {
		infusion.SetDefusing(amount.Uint64())
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
	cc := k.NewCurrentContext(ctx)
	defer cc.CommitAll()

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

		infusion := cc.GetInfusion(types.ObjectType_reactor, reactor.Id, playerAddress.String())

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

		uctx := sdk.UnwrapSDKContext(ctx)
		_ = uctx.EventManager().EmitTypedEvent(&types.EventAlphaInfuse{&types.EventAlphaInfuseDetail{PrimaryAddress: unbondingDelegation.DelegatorAddress, Amount: amount.Uint64()}})

	}
}

/* Update Reactor Infusions for All Delegations When Validator is Slashed
 *
 * Triggered during Staking Hooks:
 *   BeforeValidatorSlashed
 *
 * This function updates all infusion fuel values for delegators when a validator
 * is slashed. Since slashing reduces the validator's tokens (but not delegation shares),
 * the value of each delegation share decreases proportionally.
 */
func (k Keeper) ReactorUpdateInfusionsFromSlashing(ctx context.Context, validatorAddress sdk.ValAddress, slashFraction math.LegacyDec) {
	cc := k.NewCurrentContext(ctx)
	defer cc.CommitAll()

	/* Does this Reactor exist? */
	reactorBytes, reactorBytesFound := k.GetReactorBytesFromValidator(ctx, validatorAddress.Bytes())
	if !reactorBytesFound {
		return
	}
	reactor, _ := k.GetReactorByBytes(ctx, reactorBytes)

	/* Get the current validator state (before slashing) */
	validator, err := k.stakingKeeper.GetValidator(ctx, validatorAddress)
	if err != nil {
		k.logger.Error("Failed to get validator in ReactorUpdateInfusionsFromSlashing", "validator", validatorAddress.String(), "error", err)
		return
	}

	/* Calculate what the validator's tokens will be after slashing
	 * fraction is the percentage to slash (e.g., 0.05 = 5%)
	 * tokensAfterSlash = tokens * (1 - fraction)
	 */
	tokensAfterSlash := math.LegacyNewDecFromInt(validator.Tokens).Mul(math.LegacyOneDec().Sub(slashFraction))

	/* Get all delegations for this validator */
	delegations, err := k.stakingKeeper.GetValidatorDelegations(ctx, validatorAddress)
	if err != nil {
		k.logger.Error("Failed to get validator delegations in ReactorUpdateInfusionsFromSlashing", "validator", validatorAddress.String(), "error", err)
		return
	}

	/* Iterate through all delegations and update their infusions */
	for _, delegation := range delegations {
		delegatorAddr, err := sdk.AccAddressFromBech32(delegation.DelegatorAddress)
		if err != nil {
			k.logger.Error("Failed to parse delegator address", "address", delegation.DelegatorAddress, "error", err)
			continue
		}

		/* Calculate the new delegation share value after slashing
		 * Formula: (delegation.Shares / validator.DelegatorShares) * tokensAfterSlash
		 * Note: Delegation shares don't change during slashing, only the token value per share decreases
		 */
		delegationShare := ((delegation.Shares.Quo(validator.DelegatorShares)).Mul(tokensAfterSlash)).RoundInt()

		k.UpsertPlayer(ctx, delegatorAddr.String())
		infusion := cc.GetInfusion(types.ObjectType_reactor, reactor.Id, delegatorAddr.String())

		infusion.SetRatio(types.ReactorFuelToEnergyConversion)
		infusion.SetFuelAndCommission(delegationShare.Uint64(), reactor.DefaultCommission)

		/* Also check unbonding delegations (they may also be affected by slashing) */
		unbondingDelegation, err := k.stakingKeeper.GetUnbondingDelegation(ctx, delegatorAddr, validatorAddress)
		amount := math.ZeroInt()
		if err == nil {
			for _, entry := range unbondingDelegation.Entries {
				amount = amount.Add(entry.Balance)
			}
		}
		if infusion.GetDefusing() != amount.Uint64() {
			infusion.SetDefusing(amount.Uint64())
		}
	}
}
