package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	"context"

	"structs/x/structs/types"

	"cosmossdk.io/math"
)

/* delegationShareValue converts delegation shares into their current token
 * value using the validator's own token pool.
 */
func delegationShareValue(shares math.LegacyDec, validator stakingtypes.Validator) math.Int {
	if validator.Tokens.IsNil() {
		return math.ZeroInt()
	}

	return delegationShareValueAgainst(shares, validator.DelegatorShares, math.LegacyNewDecFromInt(validator.Tokens))
}

/* delegationShareValueAgainst converts delegation shares into their token value
 * against an explicit token pool. The slashing path needs this because it
 * projects the post-slash pool rather than reading the stored one.
 *
 * Returns zero rather than dividing when the validator holds no shares, both
 * because LegacyDec.Quo panics on a zero divisor and because a share-less
 * validator backs no value. The nil checks cover the zero-value Validator that
 * a failed GetValidator leaves behind.
 */
func delegationShareValueAgainst(shares math.LegacyDec, delegatorShares math.LegacyDec, tokens math.LegacyDec) math.Int {
	if shares.IsNil() || delegatorShares.IsNil() || delegatorShares.IsZero() || tokens.IsNil() {
		return math.ZeroInt()
	}

	return shares.Quo(delegatorShares).Mul(tokens).RoundInt()
}

/* commissionMatches reports whether an infusion already carries the commission
 * it is about to be written with, so a reconcile can skip the write.
 *
 * A nil Dec on either side counts as a mismatch rather than a panic: LegacyDec
 * wraps a big.Int pointer, Equal dereferences it, and a record predating the
 * field carries a nil one. Treating that as a mismatch also repairs it.
 */
func commissionMatches(infusion *InfusionCache, commission math.LegacyDec) bool {
	stored := infusion.GetInfusion().Commission

	return !stored.IsNil() && !commission.IsNil() && stored.Equal(commission)
}

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

		/*
		 * Convert the sdk.ValAddress into a regular sdk.AccAddress
		 *
		 * This will allow us to create a player account with the correct permissions
		 */

		var identity sdk.AccAddress
		identity = validatorAddress.Bytes()
		player := cc.UpsertPlayer(identity.String())

		// Add the player as a permissioned user of the reactor
		permissionId := GetObjectPermissionIDBytes(reactor.Id, player.GetPlayerId())
		cc.PermissionAdd(permissionId, types.PermReactorAll)

		// apply the energy distribution to the reactor player account
		delegation, err := k.stakingKeeper.GetDelegation(ctx, identity, validatorAddress)
		if err == nil {
			validator, validatorErr := k.stakingKeeper.GetValidator(ctx, validatorAddress)
			delegationShare := delegationShareValue(delegation.Shares, validator)

			infusion := cc.UpsertInfusion(types.ObjectType_reactor, reactor.Id, identity.String(), player.GetPlayerId())

			infusion.SetRatio(reactorEnergyRatio(validator, validatorErr))
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

	k.reconcileInfusionForDelegation(ctx, cc, playerAddress, validatorAddress)
}

/* ReactorInfusionDelegationRemoved zeroes an infusion's fuel when the Cosmos
 * delegation behind it is removed outright.
 *
 * Triggered during Staking Hooks:
 *   BeforeDelegationRemoved
 *
 * This hook fires before staking deletes the row, so reconciliation would read
 * stale shares and restore the old fuel. Defusing remains until the unbonding
 * balance matures.
 */
func (k Keeper) ReactorInfusionDelegationRemoved(ctx context.Context, playerAddress sdk.AccAddress, validatorAddress sdk.ValAddress) {
	cc := k.NewCurrentContext(ctx)
	defer cc.CommitAll()

	reactorBytes, reactorBytesFound := k.GetReactorBytesFromValidator(ctx, validatorAddress.Bytes())
	if !reactorBytesFound {
		return
	}
	reactor, _ := k.GetReactorByBytes(ctx, reactorBytes)

	// Load, never create. A delegation that never had an infusion has nothing
	// to clear, and upserting one here would leave an empty record behind.
	infusion := cc.GetInfusion(reactor.Id, playerAddress.String())
	if infusion.CheckInfusion() != nil {
		return
	}

	// Fuel first: once it is zero the power distribution is zero whatever the
	// ratio, so the ratio alignment below writes no further grid deltas.
	if infusion.GetFuel() != 0 {
		infusion.SetFuel(0)
	}

	validator, validatorErr := k.stakingKeeper.GetValidator(ctx, validatorAddress)
	ratio := reactorEnergyRatio(validator, validatorErr)
	if infusion.GetInfusion().Ratio != ratio {
		infusion.SetRatio(ratio)
	}
}

// ReconcileInfusionForDelegation refreshes one infusion from live staking state.
// It is a no-op when the validator's reactor is not initialized.
func (k Keeper) ReconcileInfusionForDelegation(ctx context.Context, playerAddress sdk.AccAddress, validatorAddress sdk.ValAddress) {
	cc := k.NewCurrentContext(ctx)
	defer cc.CommitAll()

	k.reconcileInfusionForDelegation(ctx, cc, playerAddress, validatorAddress)
}

func (k Keeper) reconcileInfusionForDelegation(ctx context.Context, cc *CurrentContext, playerAddress sdk.AccAddress, validatorAddress sdk.ValAddress) {
	reactorBytes, reactorBytesFound := k.GetReactorBytesFromValidator(ctx, validatorAddress.Bytes())
	if !reactorBytesFound {
		return
	}
	reactor, reactorFound := k.GetReactorByBytes(ctx, reactorBytes)
	if !reactorFound {
		return
	}
	validator, validatorErr := k.stakingKeeper.GetValidator(ctx, validatorAddress)
	ratio := reactorEnergyRatio(validator, validatorErr)

	player := cc.UpsertPlayer(playerAddress.String())
	infusion := cc.UpsertInfusion(types.ObjectType_reactor, reactor.Id, playerAddress.String(), player.GetPlayerId())
	k.reconcileInfusionState(ctx, infusion, playerAddress, validatorAddress, validator, ratio, reactor.DefaultCommission)
}

// reconcileExistingInfusionForDelegation is the removal-side form used after a
// delegation move. A source that never had an infusion has nothing to create.
func (k Keeper) reconcileExistingInfusionForDelegation(ctx context.Context, cc *CurrentContext, playerAddress sdk.AccAddress, validatorAddress sdk.ValAddress) {
	reactorBytes, reactorBytesFound := k.GetReactorBytesFromValidator(ctx, validatorAddress.Bytes())
	if !reactorBytesFound {
		return
	}
	reactor, reactorFound := k.GetReactorByBytes(ctx, reactorBytes)
	if !reactorFound {
		return
	}

	infusion := cc.GetInfusion(reactor.Id, playerAddress.String())
	if infusion.CheckInfusion() != nil {
		return
	}

	validator, validatorErr := k.stakingKeeper.GetValidator(ctx, validatorAddress)
	ratio := reactorEnergyRatio(validator, validatorErr)
	k.reconcileInfusionState(ctx, infusion, playerAddress, validatorAddress, validator, ratio, reactor.DefaultCommission)
}

func (k Keeper) reconcileInfusionState(
	ctx context.Context,
	infusion *InfusionCache,
	playerAddress sdk.AccAddress,
	validatorAddress sdk.ValAddress,
	validator stakingtypes.Validator,
	ratio uint64,
	commission math.LegacyDec,
) {
	delegation, err := k.stakingKeeper.GetDelegation(ctx, playerAddress, validatorAddress)

	// Each write is guarded on the value actually changing. SetInfusion emits an
	// EventInfusion on every write, and this function is run across every row on
	// chain by the v0.21.0 reconcile migration, so an unguarded write would emit
	// an event per healthy infusion.
	if err == nil {
		delegationShare := delegationShareValue(delegation.Shares, validator)

		if infusion.GetInfusion().Ratio != ratio {
			infusion.SetRatio(ratio)
		}
		if infusion.GetFuel() != delegationShare.Uint64() || !commissionMatches(infusion, commission) {
			infusion.SetFuelAndCommission(delegationShare.Uint64(), commission)
		}
	} else if infusion.GetFuel() != 0 {
		// No active delegation but stale fuel remains (e.g. recovery sweep
		// after a full undelegate followed by a missed AfterDelegationModified
		// path, or a full redelegation predating the BeforeDelegationRemoved
		// hook). Clear it so the destruction queue can reclaim the record once
		// Defusing also drops to zero.
		if infusion.GetInfusion().Ratio != ratio {
			infusion.SetRatio(ratio)
		}
		infusion.SetFuel(0)
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
	k.logger.Info("Unbonding Request", "unbondingId", unbondingId)

	if err != nil {
		// AfterUnbondingInitiated also fires for redelegations and validator
		// unbondings, neither of which resolve to an UBD. Those cases are not
		// relevant to the structs reactor flow; log at debug and exit.
		k.logger.Debug("AfterUnbondingInitiated: no UBD for id (likely a redelegation or validator unbonding)", "unbondingId", unbondingId)
		return
	}

	k.logger.Info("Delegation Found", "unbondingId", unbondingId, "delegator", unbondingDelegation.DelegatorAddress, "validator", unbondingDelegation.ValidatorAddress)

	var playerAddress sdk.AccAddress
	playerAddress, _ = sdk.AccAddressFromBech32(unbondingDelegation.DelegatorAddress)
	var validatorAddress sdk.ValAddress
	validatorAddress, _ = sdk.ValAddressFromBech32(unbondingDelegation.ValidatorAddress)

	/* Does this Reactor exist? It really should... */
	reactorBytes, reactorBytesFound := k.GetReactorBytesFromValidator(ctx, validatorAddress.Bytes())
	if !reactorBytesFound {
		k.logger.Warn("ReactorInfusionUnbonding: no reactor for validator", "validator", validatorAddress.String(), "delegator", playerAddress.String())
		return
	}
	reactor, _ := k.GetReactorByBytes(ctx, reactorBytes)

	// Hoisted above SetRatio so the ratio reflects validator health rather than
	// assuming a healthy reactor; the fuel calculation below reuses it.
	validator, validatorErr := k.stakingKeeper.GetValidator(ctx, validatorAddress)

	player := cc.UpsertPlayer(playerAddress.String())
	infusion := cc.UpsertInfusion(types.ObjectType_reactor, reactor.Id, playerAddress.String(), player.GetPlayerId())

	infusion.SetRatio(reactorEnergyRatio(validator, validatorErr))
	infusion.SetCommission(reactor.DefaultCommission)

	amount := math.ZeroInt()
	for _, entry := range unbondingDelegation.Entries {
		amount = amount.Add(entry.Balance) // should this be entry.InitialBalance?
	}
	infusion.SetDefusing(amount.Uint64())

	// Cosmos SDK v0.53 does not expose an unbonding-completion hook, so we
	// pre-record each UBD entry's CompletionTime and let the structs
	// EndBlocker reconcile the corresponding infusion once that block time
	// passes. Idempotent on (CompletionTime, infusionId).
	infusionId := reactor.Id + "-" + playerAddress.String()
	for _, entry := range unbondingDelegation.Entries {
		k.EnqueueInfusionMaturitySweep(ctx, entry.CompletionTime, infusionId)
	}

	delegation, err := k.stakingKeeper.GetDelegation(ctx, playerAddress, validatorAddress)
	if err == nil {
		infusion.SetFuel(delegationShareValue(delegation.Shares, validator).Uint64())
	} else {
		infusion.SetFuel(uint64(0))
	}

	uctx := sdk.UnwrapSDKContext(ctx)
	_ = uctx.EventManager().EmitTypedEvent(&types.EventAlphaInfuse{&types.EventAlphaInfuseDetail{PrimaryAddress: unbondingDelegation.DelegatorAddress, Amount: amount.Uint64()}})
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
	validator, validatorErr := k.stakingKeeper.GetValidator(ctx, validatorAddress)
	if validatorErr != nil {
		k.logger.Error("Failed to get validator in ReactorUpdateInfusionsFromSlashing", "validator", validatorAddress.String(), "error", validatorErr)
		return
	}

	/* BeforeValidatorSlashed fires before x/slashing calls Jail, so the ratio
	 * derived here is legitimately the pre-jail one and will be non-zero even
	 * for an infraction that is about to jail. Do not "correct" this: the jail
	 * is caught later in the same block by AfterValidatorBeginUnbonding during
	 * staking's EndBlock.
	 *
	 * This hook is also not jail coverage in its own right. Staking skips it
	 * entirely when the slash burns nothing (see x/staking/keeper/slash.go,
	 * the tokensToBurn.IsZero early return), so a downtime jail under a zero
	 * slash fraction never reaches this code at all.
	 */
	ratio := reactorEnergyRatio(validator, validatorErr)

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
		delegationShare := delegationShareValueAgainst(delegation.Shares, validator.DelegatorShares, tokensAfterSlash)

		player := cc.UpsertPlayer(delegatorAddr.String())
		infusion := cc.UpsertInfusion(types.ObjectType_reactor, reactor.Id, delegatorAddr.String(), player.GetPlayerId())

		infusion.SetRatio(ratio)
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
