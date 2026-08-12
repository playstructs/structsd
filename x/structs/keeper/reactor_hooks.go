package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	"context"
	stdmath "math"

	"structs/x/structs/types"

	"cosmossdk.io/math"
)

/* reactorEnergyRatio derives a reactor's fuel-to-energy conversion from the
 * health of its validator. Energy models staking rewards, and a jailed
 * validator is out of the active set earning nothing, so it produces nothing.
 *
 * Fails closed. A lookup error, a missing validator, or a jailed one all yield
 * a zero ratio, which zeroes the energy an infusion contributes while leaving
 * its Fuel (and therefore the delegator's stake) untouched.
 *
 * Takes the error from the GetValidator call alongside the validator so that a
 * missing validator can never be mistaken for a healthy zero-value one.
 */
func reactorEnergyRatio(validator stakingtypes.Validator, err error) uint64 {
	if err != nil {
		return 0
	}

	if validator.IsJailed() {
		return 0
	}

	return types.ReactorFuelToEnergyConversion
}

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
 * Staking's Unbond routes a delegation whose shares reach zero through
 * RemoveDelegation, which fires this hook and deliberately skips
 * AfterDelegationModified. A full redelegation therefore used to leave the
 * source infusion's Fuel, Power and grid contributions installed while Delegate
 * on the destination granted capacity for the very same stake, and the other
 * compensating path is no help: AfterUnbondingInitiated fires with a
 * redelegation id, which does not resolve to an unbonding delegation.
 *
 * The fuel is zeroed explicitly rather than by reconciling, because
 * RemoveDelegation calls us before it deletes the row. GetDelegation still
 * succeeds here, and still reports the pre-decrement shares, so a reconcile
 * would rewrite the exact stale value it was meant to clear.
 *
 * Defusing is left alone. It tracks unbonding balances, which outlive the
 * delegation record, and IsEmpty only releases the row to the destruction queue
 * once that has reached zero as well.
 *
 * Idempotent, and a silent no-op when the reactor or the infusion is absent.
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

/* ReconcileInfusionForDelegation refreshes a single (delegator, validator)
 * infusion's Fuel and Defusing fields from the current Cosmos staking state.
 *
 * Used by:
 *   - AfterDelegationModified (via ReactorUpdatePlayerInfusion)
 *   - The EndBlocker maturity sweep (post-CompleteUnbonding reconciliation)
 *   - The v0.17.0 upgrade handler (one-time recovery of stuck defusing state)
 *
 * No-op (silently returns) if the validator's reactor is not yet initialized.
 */
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
	reactor, _ := k.GetReactorByBytes(ctx, reactorBytes)
	validator, validatorErr := k.stakingKeeper.GetValidator(ctx, validatorAddress)
	ratio := reactorEnergyRatio(validator, validatorErr)

	player := cc.UpsertPlayer(playerAddress.String())
	infusion := cc.UpsertInfusion(types.ObjectType_reactor, reactor.Id, playerAddress.String(), player.GetPlayerId())
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
		if infusion.GetFuel() != delegationShare.Uint64() || !commissionMatches(infusion, reactor.DefaultCommission) {
			infusion.SetFuelAndCommission(delegationShare.Uint64(), reactor.DefaultCommission)
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

/* DelegationTransferPolicy decides what a handler does about a delegation that
 * cannot be moved safely. See MoveDelegationsToAddress.
 */
type DelegationTransferPolicy int

const (
	// DelegationTransferStrict fails the whole message. Used where both
	// addresses still belong to the player afterwards, so nothing is lost by
	// making them wait, and where nobody but the player can create the
	// blocking condition in the first place.
	DelegationTransferStrict DelegationTransferPolicy = iota

	// DelegationTransferDisown leaves the delegation where it is and destroys
	// the infusion representing it. Used by AddressRevoke, which is the
	// response to a compromised key: a refusal there is a state the holder of
	// that key could sustain indefinitely with rolling redelegations, while
	// gaining the player nothing, since that same key could always have
	// undelegated the stake directly.
	DelegationTransferDisown
)

/* MoveDelegationsToAddress hands every delegation held by one address to
 * another and rebuilds the reactor infusions on both sides.
 *
 * The three address-move handlers (AddressRevoke, AddressRegister,
 * PlayerUpdatePrimaryAddress) all sweep a player's stake onto their primary
 * address, and Cosmos has no operation for that. There is no "transfer
 * delegation": the move has to be assembled out of primitives, and each of the
 * three records keyed by delegator address has to be carried across by hand.
 *
 * Staking gives no help keeping structs in step. The two store calls are
 * asymmetric: RemoveDelegation fires BeforeDelegationRemoved, which zeroes the
 * source infusion, while SetDelegation is a bare write that fires nothing at
 * all. Left to the hooks the source loses its capacity and the destination is
 * credited by nobody, so the move destroys the player's energy rather than
 * relocating it. Both sides are therefore reconciled explicitly, which also
 * makes the repair independent of whether the hook ran and is what lets the
 * mock staking keeper — which fires no hooks — exercise it.
 *
 * Distribution gets the same treatment for the same reason. SetDelegation
 * firing nothing means the destination pair never gets the DelegatorStartingInfo
 * that prices its rewards, and without it withdrawing, undelegating and
 * redelegating all fail on ErrEmptyDelegationDistInfo forever after. The three
 * distribution hooks below are the only public route to that lifecycle, and the
 * order is fixed by initializeDelegation: it reads the delegation out of
 * staking, so AfterDelegationModified has to come after the write, and it
 * prices from Period-1, so something has to have incremented the period first.
 *
 * Shares merge rather than overwrite. SetDelegation is keyed by
 * (delegator, validator), so writing the source's record at the destination
 * would silently replace a delegation the destination already had, while
 * Validator.DelegatorShares went on counting both — orphaned tokens and a
 * skewed redemption ratio. DelegatorShares is deliberately left alone: the sum
 * is conserved, and LegacyDec addition is exact.
 *
 * Everything that can refuse runs before anything mutates. Error propagation
 * would be enough for atomicity on its own, since BaseApp discards the message
 * cache, but partitioning first is what lets the disown policy tell "cannot
 * move this one" apart from "this whole operation failed".
 *
 * The reconciles run after the loop rather than inside it. The hook commits a
 * CurrentContext of its own, so a reconcile interleaved with it would leave cc
 * holding a grid capacity read from before the hook's write and clobber that
 * write at CommitAll.
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

	// GetDelegatorDelegations truncates at maxRetrieve rather than reporting
	// that it did, and a silently partial move is the one outcome worse than
	// no move at all.
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

// delegationMovement is one source delegation that passed pre-flight.
type delegationMovement struct {
	delegation stakingtypes.Delegation
	validator  sdk.ValAddress
}

/* partitionTransferableDelegations splits the source's delegations into those
 * that can be handed over and those that cannot, refusing outright under the
 * strict policy.
 *
 * The two in-flight cases are not conservatism, they are the limit of what is
 * reachable. An unbonding delegation and a redelegation each carry queue rows
 * (UBDQueue DVPairs, RedelegationQueue DVVTriplets) and unbonding-id indices
 * that no public keeper API can rewrite, so their delegator address cannot
 * follow the delegation.
 *
 * Leaving a redelegation behind is the dangerous one, and it is worth being
 * precise about why. SlashRedelegation resolves the delegation it slashes
 * through the redelegation record's own delegator address, and on a miss it
 * continues rather than failing — so moving the delegation out from under an
 * in-flight redelegation makes that stake unslashable for the source
 * validator's infraction. The same record is what HasReceivingRedelegation
 * consults to stop redelegation hopping, which the move would likewise escape.
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

// delegationTransferBlocker returns the reason a (from, validator) delegation
// cannot be handed to another address, or "" when it can be.
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

	// Absent starting info means BeforeDelegationSharesModified below would
	// fail on ErrEmptyDelegationDistInfo. Only rows written by the pre-v0.21.0
	// rekey are in that state, and MigrateDelegationDistributionState repairs
	// them, but a delegation that cannot be priced must not be moved.
	hasStartingInfo, startingInfoErr := k.distributionKeeper.HasDelegatorStartingInfo(ctx, validatorAddress, from)
	if startingInfoErr != nil {
		return "", startingInfoErr
	}
	if !hasStartingInfo {
		return "missing_distribution_state", nil
	}

	return "", nil
}

/* transferDelegation moves one delegation, settling both sides' rewards first
 * and merging into whatever the destination already held.
 *
 * This is the sequence the SDK itself runs around a share change, applied to
 * two delegators at once: settle what each is owed at its current share count,
 * change the shares, then re-open the reward period against the new count.
 * Withdrawing first is what stops the destination being retroactively paid for
 * periods in which it did not hold the incoming shares.
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
		// Increments the validator period, which initializeDelegation reads
		// back as Period-1 once AfterDelegationModified runs below.
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

/* disownInfusion drops the game's representation of stake that stayed behind on
 * an address the player no longer owns.
 *
 * The delegation itself is untouched and still bonded, so nothing is destroyed
 * at the SDK layer and the key holder can still undelegate it directly. What
 * goes is the capacity the player was being credited for stake they no longer
 * control.
 *
 * The record left behind resolves itself: the next time staking touches that
 * delegation, reconcileInfusionForDelegation runs UpsertPlayer against an
 * address that no longer indexes to anyone, and the address becomes its own
 * player holding its own infusion. A pending maturity-sweep row does the same.
 */
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

/* ReactorGateEnergy zeroes the energy ratio on every infusion in a reactor.
 *
 * Triggered during Staking Hooks:
 *   AfterValidatorBeginUnbonding (when the validator is jailed)
 *
 * Also reachable through MsgReactorRestart and the v0.21.0 migration.
 *
 * Only the ratio changes. Fuel, Defusing, and the underlying Cosmos delegation
 * are all left alone, so no stake moves and the records survive the EndBlocker
 * destruction sweep (see InfusionCache.IsEmpty). What drops is capacity, both
 * the reactor's commission share and every delegator's share, and the grid
 * cascade that follows runs later in the same block.
 *
 * Iterates stored infusions rather than live delegations so that a row whose
 * delegation has already vanished, but which still carries power, is gated too.
 *
 * Idempotent, and a silent no-op when the reactor does not exist.
 */
func (k Keeper) ReactorGateEnergy(ctx context.Context, validatorAddress sdk.ValAddress) {
	cc := k.NewCurrentContext(ctx)
	defer cc.CommitAll()

	reactorBytes, reactorBytesFound := k.GetReactorBytesFromValidator(ctx, validatorAddress.Bytes())
	if !reactorBytesFound {
		return
	}
	reactor, _ := k.GetReactorByBytes(ctx, reactorBytes)

	gated := 0
	for _, infusion := range cc.GetAllInfusionByDestination(reactor.Id) {
		if infusion.CheckInfusion() != nil {
			continue
		}

		// Skip rows already at a zero ratio so a repeat call writes nothing and
		// emits no redundant indexer events.
		if infusion.GetInfusion().Ratio == 0 {
			continue
		}

		infusion.SetRatio(0)
		gated++
	}

	if gated > 0 {
		k.logger.Info("Reactor energy gated", "validator", validatorAddress.String(), "reactorId", reactor.Id, "infusions", gated)
	}
}

/* ReactorRestoreEnergy brings a reactor's infusions back in line with live
 * Cosmos staking state.
 *
 * Triggered during Staking Hooks:
 *   AfterValidatorBonded
 *
 * Also reachable through MsgReactorRestart and the v0.21.0 migration.
 *
 * Deliberately reconciles rather than simply writing a non-zero ratio back. A
 * blind restore would relight rows whose delegation has since disappeared but
 * whose fuel is stale, handing out capacity for stake that no longer exists.
 * Routing each row through reconcileInfusionForDelegation instead reads the
 * live delegation, zeroes fuel where there is none, and derives the ratio from
 * validator health, so this heals stale rows and re-gates a still-jailed
 * validator instead of trusting the caller.
 *
 * Idempotent, and a silent no-op when the reactor does not exist.
 */
func (k Keeper) ReactorRestoreEnergy(ctx context.Context, validatorAddress sdk.ValAddress) {
	cc := k.NewCurrentContext(ctx)
	defer cc.CommitAll()

	reactorBytes, reactorBytesFound := k.GetReactorBytesFromValidator(ctx, validatorAddress.Bytes())
	if !reactorBytesFound {
		return
	}
	reactor, _ := k.GetReactorByBytes(ctx, reactorBytes)

	for _, infusion := range cc.GetAllInfusionByDestination(reactor.Id) {
		if infusion.CheckInfusion() != nil {
			continue
		}

		playerAddress, err := sdk.AccAddressFromBech32(infusion.GetInfusion().Address)
		if err != nil {
			k.logger.Warn("ReactorRestoreEnergy: invalid delegator address", "reactorId", reactor.Id, "address", infusion.GetInfusion().Address, "error", err)
			continue
		}

		k.reconcileInfusionForDelegation(ctx, cc, playerAddress, validatorAddress)
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
