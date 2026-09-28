package keeper

import (
	"context"
	stdmath "math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	vestingexported "github.com/cosmos/cosmos-sdk/x/auth/vesting/exported"
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

	// Rotation or registration keeps the source registered and accessible. Preserve its
	// native operator stake and vesting-account stake without moving their
	// obligations; remain strict for other delegation blockers.
	DelegationTransferStrictRetainSelf
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

	_, vestingSource := k.accountKeeper.GetAccount(ctx, from).(vestingexported.VestingAccount)
	movable, blocked, err := k.partitionTransferableDelegations(ctx, from, to, delegations, policy, vestingSource)
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
		// A reconcile that fails here has already lost the source credit, so
		// it must fail the transaction rather than leave the destination
		// uncredited.
		if err := k.reconcileExistingInfusionForDelegation(ctx, cc, from, validatorAddress); err != nil {
			return err
		}
		if _, _, err := k.reconcileInfusionForDelegation(ctx, cc, toAcc, validatorAddress); err != nil {
			return err
		}
	}

	if policy == DelegationTransferStrictRetainSelf {
		// Registration staged the new address owner in this same context.
		// Reconcile only after all transfers/hook commits, so retained stake
		// credits that owner without a stale outer grid cache overwriting hooks.
		for _, delegation := range delegations {
			validatorAddress, _ := sdk.ValAddressFromBech32(delegation.ValidatorAddress)
			if vestingSource || from.Equals(sdk.AccAddress(validatorAddress)) {
				if _, _, err := k.reconcileInfusionForDelegation(ctx, cc, from, validatorAddress); err != nil {
					return err
				}
			}
		}
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

/* partitionTransferableDelegations preflights every source delegation. This
 * transaction helper does not migrate queue slices or operation-ID indices,
 * so it refuses in-flight unbondings and redelegations. A coordinated upgrade
 * can use the SDK's queue/index setters, but must move all related records and
 * preserve slashing and maturity semantics; those are outside this helper.
 */
func (k Keeper) partitionTransferableDelegations(
	ctx context.Context,
	from sdk.AccAddress,
	to string,
	delegations []stakingtypes.Delegation,
	policy DelegationTransferPolicy,
	vestingSource bool,
) (movable []delegationMovement, blocked []sdk.ValAddress, err error) {
	for _, delegation := range delegations {
		validatorAddress, validatorAddressErr := sdk.ValAddressFromBech32(delegation.ValidatorAddress)
		if validatorAddressErr != nil {
			return nil, nil, types.NewAddressValidationError(delegation.ValidatorAddress, "invalid_format")
		}

		// SetDelegation moves no vesting counters. Keep the native account and
		// all its stake together, regardless of which validator holds it. A
		// revoke must still clear its capacity without being held up by locks.
		if vestingSource {
			if policy == DelegationTransferStrictRetainSelf {
				continue
			}
			if policy != DelegationTransferDisown {
				return nil, nil, types.NewDelegationTransferError(from.String(), to, "vesting_account").WithValidator(validatorAddress.String())
			}
			blocked = append(blocked, validatorAddress)
			continue
		}

		if policy == DelegationTransferStrictRetainSelf && from.Equals(sdk.AccAddress(validatorAddress)) {
			continue
		}

		reason, checkErr := k.delegationTransferBlocker(ctx, from, validatorAddress)
		if checkErr != nil {
			return nil, nil, checkErr
		}

		if reason == "" {
			movable = append(movable, delegationMovement{delegation: delegation, validator: validatorAddress})
			continue
		}

		if policy != DelegationTransferDisown {
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
	// Rekeying self-stake would bypass native minimum-self-delegation and
	// make native unjailing unable to find it. Revocation may disown it, but
	// never relocate it through the assembled transfer.
	if from.Equals(sdk.AccAddress(validatorAddress)) {
		return "validator_self_delegation", nil
	}

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
