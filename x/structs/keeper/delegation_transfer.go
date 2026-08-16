package keeper

import (
	"context"
	stdmath "math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	"structs/x/structs/types"
)

// DelegationTransferPolicy decides how an address move handles a delegation
// that cannot be transferred safely.
type DelegationTransferPolicy int

const (
	// DelegationTransferStrict fails the whole message.
	DelegationTransferStrict DelegationTransferPolicy = iota

	// DelegationTransferDisown leaves blocked stake in place but removes the
	// capacity credited to an address the player no longer owns.
	DelegationTransferDisown
)

/* MoveDelegationsToAddress assembles the delegation transfer Cosmos does not
 * provide. It merges shares and rebuilds distribution state and both infusions.
 * Reconciliation happens after the transfer loop so hook-owned nested commits
 * cannot be overwritten by stale values in the caller's CurrentContext.
 */
func (k Keeper) MoveDelegationsToAddress(ctx context.Context, cc *CurrentContext, from sdk.AccAddress, to string, policy DelegationTransferPolicy) error {
	toAcc, toErr := sdk.AccAddressFromBech32(to)
	if toErr != nil {
		return types.NewAddressValidationError(to, "invalid_format")
	}

	if from.Equals(toAcc) {
		return nil
	}

	delegations, err := k.stakingKeeper.GetDelegatorDelegations(ctx, from, stdmath.MaxUint16)
	if err != nil {
		return err
	}

	// GetDelegatorDelegations truncates at maxRetrieve; a partial move must fail.
	if len(delegations) >= stdmath.MaxUint16 {
		return types.NewDelegationTransferError(from.String(), to, "too_many_delegations").WithCount(len(delegations))
	}

	movable, blocked, err := k.partitionTransferableDelegations(ctx, from, to, delegations, policy)
	if err != nil {
		return err
	}

	touched := make([]sdk.ValAddress, 0, len(movable))
	for _, movement := range movable {
		if err := k.transferDelegation(ctx, movement, from, toAcc); err != nil {
			return err
		}
		touched = append(touched, movement.validator)
	}

	for _, validatorAddress := range touched {
		k.reconcileInfusionForDelegation(ctx, cc, from, validatorAddress)
		k.reconcileInfusionForDelegation(ctx, cc, toAcc, validatorAddress)
	}

	for _, validatorAddress := range blocked {
		k.disownInfusion(ctx, cc, from, validatorAddress)
	}

	return nil
}

type delegationMovement struct {
	delegation stakingtypes.Delegation
	validator  sdk.ValAddress
}

/* partitionTransferableDelegations preflights every source delegation. SDK
 * queue rows and indices cannot be rekeyed, so in-flight unbonding delegations
 * and redelegations cannot follow the delegation safely.
 */
func (k Keeper) partitionTransferableDelegations(
	ctx context.Context,
	from sdk.AccAddress,
	to string,
	delegations []stakingtypes.Delegation,
	policy DelegationTransferPolicy,
) (movable []delegationMovement, blocked []sdk.ValAddress, err error) {
	for _, delegation := range delegations {
		validatorAddress, validatorAddressErr := sdk.ValAddressFromBech32(delegation.ValidatorAddress)
		if validatorAddressErr != nil {
			return nil, nil, types.NewAddressValidationError(delegation.ValidatorAddress, "invalid_format")
		}

		reason, checkErr := k.delegationTransferBlocker(ctx, from, validatorAddress)
		if checkErr != nil {
			return nil, nil, checkErr
		}

		if reason == "" {
			movable = append(movable, delegationMovement{delegation: delegation, validator: validatorAddress})
			continue
		}

		if policy == DelegationTransferStrict {
			return nil, nil, types.NewDelegationTransferError(from.String(), to, reason).WithValidator(validatorAddress.String())
		}

		k.logger.Info("Delegation left behind by an address move",
			"from", from.String(), "to", to, "validator", validatorAddress.String(), "reason", reason)
		blocked = append(blocked, validatorAddress)
	}

	return movable, blocked, nil
}

// delegationTransferBlocker returns why a delegation cannot be transferred.
func (k Keeper) delegationTransferBlocker(ctx context.Context, from sdk.AccAddress, validatorAddress sdk.ValAddress) (string, error) {
	receiving, receivingErr := k.stakingKeeper.HasReceivingRedelegation(ctx, from, validatorAddress)
	if receivingErr != nil {
		return "", receivingErr
	}
	if receiving {
		return "redelegation_in_flight", nil
	}

	if _, unbondingErr := k.stakingKeeper.GetUnbondingDelegation(ctx, from, validatorAddress); unbondingErr == nil {
		return "defusing_in_flight", nil
	}

	// A delegation without starting info cannot be settled before its shares move.
	hasStartingInfo, startingInfoErr := k.distributionKeeper.HasDelegatorStartingInfo(ctx, validatorAddress, from)
	if startingInfoErr != nil {
		return "", startingInfoErr
	}
	if !hasStartingInfo {
		return "missing_distribution_state", nil
	}

	return "", nil
}

/* transferDelegation settles rewards at the old share counts, merges the
 * destination shares, then initializes distribution at the new count.
 */
func (k Keeper) transferDelegation(ctx context.Context, movement delegationMovement, from sdk.AccAddress, toAcc sdk.AccAddress) error {
	validatorAddress := movement.validator

	if err := k.distributionHooks.BeforeDelegationSharesModified(ctx, from, validatorAddress); err != nil {
		return err
	}

	shares := movement.delegation.Shares
	destination, destinationErr := k.stakingKeeper.GetDelegation(ctx, toAcc, validatorAddress)
	if destinationErr == nil {
		if err := k.distributionHooks.BeforeDelegationSharesModified(ctx, toAcc, validatorAddress); err != nil {
			return err
		}
		shares = shares.Add(destination.Shares)
	} else {
		// AfterDelegationModified initializes against the period incremented here.
		if err := k.distributionHooks.BeforeDelegationCreated(ctx, toAcc, validatorAddress); err != nil {
			return err
		}
	}

	if err := k.stakingKeeper.RemoveDelegation(ctx, movement.delegation); err != nil {
		return err
	}

	merged := movement.delegation
	merged.DelegatorAddress = toAcc.String()
	merged.Shares = shares
	if err := k.stakingKeeper.SetDelegation(ctx, merged); err != nil {
		return err
	}

	return k.distributionHooks.AfterDelegationModified(ctx, toAcc, validatorAddress)
}

// disownInfusion removes capacity backed by stake left at a revoked address;
// the Cosmos delegation itself remains untouched.
func (k Keeper) disownInfusion(ctx context.Context, cc *CurrentContext, from sdk.AccAddress, validatorAddress sdk.ValAddress) {
	reactorBytes, reactorBytesFound := k.GetReactorBytesFromValidator(ctx, validatorAddress.Bytes())
	if !reactorBytesFound {
		return
	}
	reactor, _ := k.GetReactorByBytes(ctx, reactorBytes)

	infusion := cc.GetInfusion(reactor.Id, from.String())
	if infusion.CheckInfusion() != nil {
		return
	}

	infusion.Destroy()
}
