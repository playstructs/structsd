package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	"context"
	"errors"
	"fmt"
	"time"

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
 * The arithmetic is Validator.TokensFromShares spelled out: multiply the shares
 * by the pool first, divide by the share total last, then truncate. Both halves
 * of that matter and neither is style.
 *
 * Truncate, never round. Fuel is written per delegation, and rounding each one
 * independently lets a sum exceed the stake behind it: after a 1% slash a
 * 50-share delegation is worth exactly 49.5, and LegacyDec.RoundInt is banker's
 * rounding, so every such shard rounds back to 50. Split a million across twenty
 * thousand addresses and the slash is undone in full - a million Fuel, and
 * therefore a million of grid capacity, standing on 990,000 tokens. Nothing sums
 * Fuel against validator.Tokens to notice. Truncation cannot do that: each shard
 * is at most its true value, so the total is at most the stake, and the residue
 * is dust nobody is credited for rather than capacity nobody staked for.
 *
 * Multiply first for the same reason in miniature. Dividing first rounds the
 * ratio to LegacyDec's 18 places before it ever meets the pool, so the error
 * scales with the pool; the SDK orders it the other way and this has to agree
 * with the SDK, because that is what the stake is actually worth.
 * Use its truncated division variant too: Quo rounds at 18 decimal places
 * and can turn a value just below one into one before TruncateInt runs.
 * Validator tokens are integral, so the multiplication is exact.
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

	value := shares.Mul(tokens).QuoTruncate(delegatorShares).TruncateInt()

	// Every caller writes this into a uint64 Fuel field, and math.Int.Uint64
	// panics rather than wrapping. A pool that large cannot exist behind a
	// uint64-denominated supply, but these run inside staking hooks that have no
	// way to refuse, so answer with the value that mints nothing.
	if value.IsNegative() || !value.IsUint64() {
		return math.ZeroInt()
	}

	return value
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
func (k Keeper) ReactorUpdatePlayerInfusion(ctx context.Context, playerAddress sdk.AccAddress, validatorAddress sdk.ValAddress) error {
	cc := k.NewCurrentContext(ctx)
	infusion, reduced, err := k.reconcileInfusionForDelegation(ctx, cc, playerAddress, validatorAddress)
	if err != nil { return err }
	if reduced || infusion != nil && infusion.GetDefusing() != 0 {
		// Unbond updates shares before its UBD entry. A same-block merge then
		// skips AfterUnbondingInitiated, so re-read only this pair after staking
		// finishes. Queue keys deduplicate repeated reductions in one block.
		k.EnqueueInfusionMaturitySweep(ctx, sdk.UnwrapSDKContext(ctx).HeaderInfo().Time, infusion.ID())
	}
	cc.CommitAll()
	return nil
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
	// Full unbonding skips AfterDelegationModified. Schedule the same final
	// read without reconciling here, while the removed delegation still exists.
	k.EnqueueInfusionMaturitySweep(ctx, sdk.UnwrapSDKContext(ctx).HeaderInfo().Time, infusion.ID())

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
// It commits only a successful reconciliation and is a no-op when the
// validator's reactor is not initialized.
func (k Keeper) ReconcileInfusionForDelegation(ctx context.Context, playerAddress sdk.AccAddress, validatorAddress sdk.ValAddress) error {
	cc := k.NewCurrentContext(ctx)
	if _, _, err := k.reconcileInfusionForDelegation(ctx, cc, playerAddress, validatorAddress); err != nil {
		k.logger.Error("ReconcileInfusionForDelegation: reconcile failed", "delegator", playerAddress.String(), "validator", validatorAddress.String(), "error", err)
		return err
	}
	cc.CommitAll()
	return nil
}

// ReconcileExistingInfusionForDelegation refreshes one infusion from live
// staking state without creating a player, so it is safe for an unregistered
// delegator address whose infusion still carries phantom fuel (e.g. a pre-fix
// AddressRevoke sweep). A source that never had an infusion is a no-op.
func (k Keeper) ReconcileExistingInfusionForDelegation(ctx context.Context, playerAddress sdk.AccAddress, validatorAddress sdk.ValAddress) error {
	cc := k.NewCurrentContext(ctx)
	if err := k.reconcileExistingInfusionForDelegation(ctx, cc, playerAddress, validatorAddress); err != nil { return err }
	cc.CommitAll()
	return nil
}

func (k Keeper) reconcileInfusionForDelegation(ctx context.Context, cc *CurrentContext, playerAddress sdk.AccAddress, validatorAddress sdk.ValAddress) (*InfusionCache, bool, error) {
	reactorBytes, reactorBytesFound := k.GetReactorBytesFromValidator(ctx, validatorAddress.Bytes())
	if !reactorBytesFound {
		return nil, false, nil
	}
	reactor, reactorFound := k.GetReactorByBytes(ctx, reactorBytes)
	if !reactorFound {
		return nil, false, nil
	}
	validator, validatorErr := k.stakingKeeper.GetValidator(ctx, validatorAddress)
	ratio := reactorEnergyRatio(validator, validatorErr)

	owner, err := cc.reactorInfusionOwner(playerAddress)
	if err != nil {
		k.logger.Error("cannot resolve reactor infusion player", "address", playerAddress.String(), "error", err)
		return nil, false, err
	}
	infusion, err := cc.UpsertOwnedInfusion(types.ObjectType_reactor, reactor.Id, playerAddress.String(), owner.ID())
	if err != nil { k.logger.Error("cannot load owned reactor infusion", "error", err); return nil, false, err }
	oldFuel := infusion.GetFuel()
	if err := k.reconcileInfusionState(ctx, infusion, playerAddress, validatorAddress, validator, ratio, reactor.DefaultCommission); err != nil {
        return nil, false, err
    }
	return infusion, infusion.GetFuel() < oldFuel, nil
}

// reconcileExistingInfusionForDelegation is the removal-side form used after a
// delegation move. A source that never had an infusion has nothing to create.
func (k Keeper) reconcileExistingInfusionForDelegation(ctx context.Context, cc *CurrentContext, playerAddress sdk.AccAddress, validatorAddress sdk.ValAddress) error {
	reactorBytes, reactorBytesFound := k.GetReactorBytesFromValidator(ctx, validatorAddress.Bytes())
	if !reactorBytesFound {
		return nil
	}
	reactor, reactorFound := k.GetReactorByBytes(ctx, reactorBytes)
	if !reactorFound {
		return nil
	}

	infusion := cc.GetInfusion(reactor.Id, playerAddress.String())
	if infusion.CheckInfusion() != nil {
		return nil
	}

	validator, validatorErr := k.stakingKeeper.GetValidator(ctx, validatorAddress)
	ratio := reactorEnergyRatio(validator, validatorErr)
	return k.reconcileInfusionState(ctx, infusion, playerAddress, validatorAddress, validator, ratio, reactor.DefaultCommission)
}

func (k Keeper) reconcileInfusionState(
	ctx context.Context,
	infusion *InfusionCache,
	playerAddress sdk.AccAddress,
	validatorAddress sdk.ValAddress,
	validator stakingtypes.Validator,
	ratio uint64,
	commission math.LegacyDec,
) error {
	row := infusion.GetInfusion()
	if row.DestinationType != types.ObjectType_reactor || row.DestinationId != infusion.DestinationId || row.Address != playerAddress.String() {
		return fmt.Errorf("invalid reactor infusion %s", infusion.ID())
	}
	delegation, err := k.stakingKeeper.GetDelegation(ctx, playerAddress, validatorAddress)
	if err != nil && !errors.Is(err, stakingtypes.ErrNoDelegation) {
		return err
	}
	unbondingDelegation, unbondingErr := k.stakingKeeper.GetUnbondingDelegation(ctx, playerAddress, validatorAddress)
	if unbondingErr != nil && !errors.Is(unbondingErr, stakingtypes.ErrNoUnbondingDelegation) {
		return unbondingErr
	}
	amount := math.ZeroInt()
	retryHeld := false
	// Store exactly the current flat backing. Ineligible reactors retain fuel
	// and unbonding balances, but contribute no power.
	fuel := uint64(0)
	if err == nil { fuel = delegationShareValue(delegation.Shares, validator).Uint64() }
	if infusion.GetInfusion().Ratio != ratio { infusion.SetRatio(ratio) }
	if infusion.GetFuel() != fuel || !commissionMatches(infusion, commission) {
		infusion.SetFuelAndCommission(fuel, commission)
	}

	blockTime := sdk.UnwrapSDKContext(ctx).HeaderInfo().Time
	if unbondingErr == nil {
		for _, entry := range unbondingDelegation.Entries {
			amount = amount.Add(entry.Balance)
			if entry.OnHold() && entry.IsMature(blockTime) {
				retryHeld = true
			}
		}
	}
	if retryHeld {
		// SDK UnbondingCanComplete can pay a matured held entry after our
		// scheduled sweep without firing a staking hook. Keep one future retry
		// per infusion, within the bounded maturity queue, until release is seen.
		// Moving the retry after this block's time lets older work run first.
		k.EnqueueInfusionMaturitySweep(ctx, blockTime.Add(time.Nanosecond), infusion.ID())
	}
	if infusion.GetDefusing() != amount.Uint64() {
		infusion.SetDefusing(amount.Uint64())
	}
	return nil
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

	owner, err := cc.reactorInfusionOwner(playerAddress)
	if err != nil {
		k.logger.Error("cannot resolve reactor infusion player", "address", playerAddress.String(), "error", err)
		return
	}
	infusion, err := cc.UpsertOwnedInfusion(types.ObjectType_reactor, reactor.Id, playerAddress.String(), owner.ID())
	if err != nil { k.logger.Error("cannot load owned reactor infusion", "error", err); return }

	if err := k.reconcileInfusionState(ctx, infusion, playerAddress, validatorAddress, validator, reactorEnergyRatio(validator, validatorErr), reactor.DefaultCommission); err != nil {
		k.logger.Error("cannot reconcile unbonding reactor infusion", "infusionId", infusion.ID(), "error", err)
		return
	}

	amount := math.ZeroInt()
	for _, entry := range unbondingDelegation.Entries {
		amount = amount.Add(entry.Balance) // should this be entry.InitialBalance?
	}

	// Cosmos SDK v0.53 does not expose an unbonding-completion hook, so we
	// pre-record each UBD entry's CompletionTime and let the structs
	// EndBlocker reconcile the corresponding infusion once that block time
	// passes. Idempotent on (CompletionTime, infusionId).
	infusionId := reactor.Id + "-" + playerAddress.String()
	for _, entry := range unbondingDelegation.Entries {
		k.EnqueueInfusionMaturitySweep(ctx, entry.CompletionTime, infusionId)
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
/* ReactorUpdateInfusionsFromSlashing queues a reactor for reconciliation.
 *
 * It used to reconcile every active delegation to the validator right here.
 * BeforeValidatorSlashed runs in BeginBlock against an infinite gas meter, and
 * the delegator set is unbounded and chosen by the delegators, so the cost of
 * one slash block was whatever they had built up - each entry costing a
 * delegation read, an unbonding-delegation read and a grid rewrite. The SDK's
 * own unbounded work during Slash covers unbonding delegations and
 * redelegations, a window-limited set; this walked the full active one.
 *
 * A slash leaves delegation shares untouched, so AfterDelegationModified never
 * fires for it and this hook is the module's only signal that one happened.
 * Reconciliation therefore cannot be dropped, only deferred: the queue carries
 * it into the EndBlocker, a bounded number of infusions at a time.
 *
 * The revision and insertion cutoff are captured, not a projected token balance.
 * The hook fires before staking
 * applies it - which is why the old code derived the post-slash token figure by
 * hand - and by the time the queue runs, live staking state is already the
 * answer.
 *
 * The window this opens is real and worth naming: an infusion not yet reached
 * still carries its pre-slash fuel and the grid capacity standing on it. It is
 * drained at ReactorSlashReconcileBudget per block. In the active economy,
 * commission admission stays gated until the latest slash revision is repaired;
 * player readiness separately checks live delegation backing.
 */
func (k Keeper) ReactorUpdateInfusionsFromSlashing(ctx context.Context, validatorAddress sdk.ValAddress, slashFraction math.LegacyDec) error {
	reactorBytes, reactorBytesFound := k.GetReactorBytesFromValidator(ctx, validatorAddress.Bytes())
	if !reactorBytesFound {
		return nil
	}

	reactor, reactorFound := k.GetReactorByBytes(ctx, reactorBytes)
	if !reactorFound {
		return nil
	}

	if !slashFraction.IsNil() && slashFraction.IsPositive() {
		if err := k.advanceReactorCommissionRevision(ctx, reactor.Id); err != nil {
			return err
		}
	}
	k.EnqueueReactorSlashReconcile(ctx, reactor.Id)
	return nil
}

// A slash pass promises to repair a specific reactor's indexed positions. Unlike
// removal-side reconciliation, missing or mis-keyed state cannot count as success.
func (k Keeper) reconcileSlashedReactorPosition(ctx context.Context, cc *CurrentContext, reactor types.Reactor, address sdk.AccAddress, validatorAddress sdk.ValAddress) error {
    infusion := cc.GetInfusion(reactor.Id, address.String())
    if err := infusion.CheckInfusion(); err != nil { return err }
    row := infusion.GetInfusion()
    if infusion.Deleted || row.DestinationId != reactor.Id || row.Address != address.String() || row.DestinationType != types.ObjectType_reactor {
        return fmt.Errorf("invalid indexed reactor infusion %s", infusion.ID())
    }
    validator, err := k.stakingKeeper.GetValidator(ctx, validatorAddress)
    if err != nil { return err }
    return k.reconcileInfusionState(ctx, infusion, address, validatorAddress, validator, reactorEnergyRatio(validator, nil), reactor.DefaultCommission)
}
