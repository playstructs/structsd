package keeper

import (
	"structs/x/structs/types"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type AgreementCache struct {
	AgreementId string

	CC  *CurrentContext

	Ready bool

    Changed bool
    Deleted bool

    AgreementLoaded  bool
	Agreement        types.Agreement

	// TearingDown marks that settlement of this agreement has begun in this
	// operation. Agreement teardown destroys the allocation and allocation
	// teardown settles the agreement, so the two call each other. Because
	// cc.agreements hands both legs the same cache, the second leg would
	// otherwise pay out and decrement provider load all over again.
	TearingDown bool

	PreviousEndBlock uint64
	EndBlockChanged bool

	DurationRemaining       uint64
	DurationRemainingLoaded bool

	DurationPast       uint64
	DurationPastLoaded bool

	Duration       uint64
	DurationLoaded bool

	CurrentBlock       uint64
	CurrentBlockLoaded bool

}


func (cache *AgreementCache) Commit() {
    if cache.Changed {

    	cache.CC.k.logger.Info("Updating Agreement From Cache", "agreementId", cache.AgreementId)

    	if cache.Deleted {
    	    cache.CC.k.RemoveAgreementProviderIndex(cache.CC.ctx, cache.GetProviderId(), cache.GetAgreementId())
    	    cache.CC.k.RemoveAgreementExpirationIndex(cache.CC.ctx, cache.GetEndBlock(), cache.GetAgreementId())
    	    if cache.EndBlockChanged && cache.PreviousEndBlock > 0 {
                cache.CC.k.RemoveAgreementExpirationIndex(cache.CC.ctx, cache.PreviousEndBlock, cache.GetAgreementId())
            }
    	    cache.CC.k.ClearAgreement(cache.CC.ctx, cache.AgreementId)
    	} else {
    		cache.CC.k.SetAgreement(cache.CC.ctx, cache.Agreement)
    		if (cache.EndBlockChanged) {
    		    if cache.PreviousEndBlock > 0 {
                    cache.CC.k.RemoveAgreementExpirationIndex(cache.CC.ctx, cache.PreviousEndBlock, cache.GetAgreementId())
                }
                cache.CC.k.SetAgreementExpirationIndex(cache.CC.ctx, cache.GetEndBlock(), cache.GetAgreementId())
    		}
    	}

        cache.Changed = false

    }

}

func (cache *AgreementCache) IsChanged() bool {
	return cache.Changed
}

func (cache *AgreementCache) ID() string {
	return cache.AgreementId
}



/* Separate Loading functions for each of the underlying containers */

// Load the Agreement record
func (cache *AgreementCache) LoadAgreement() bool {
	cache.Agreement, cache.AgreementLoaded = cache.CC.k.GetAgreement(cache.CC.ctx, cache.AgreementId)

	return cache.AgreementLoaded
}

func (cache *AgreementCache) LoadCurrentBlock() bool {
	uctx := sdk.UnwrapSDKContext(cache.CC.ctx)
	cache.CurrentBlock = uint64(uctx.BlockHeight())
	cache.CurrentBlockLoaded = true
	return cache.CurrentBlockLoaded
}

// LoadDurationRemaining measures the unearned part of the agreement, which is
// what the consumer's remaining collateral is priced from.
//
// It never measures from before the start block. No path opens an agreement in
// the future today — AgreementOpen starts service in the opening block and the
// capacity changes re-base to the current one — so the clamp is a guard rather
// than a description of how agreements begin. It earns its place by bounding the
// damage if that ever changes: collateral is only ever collected for
// EndBlock - StartBlock, so pricing a settlement from an earlier height would
// refund blocks nobody deposited, out of a pool shared with other agreements.
// With LoadDurationPast it holds past + remaining == duration at every height,
// which is the identity the collateral accounting rests on.
func (cache *AgreementCache) LoadDurationRemaining() bool {
	from := cache.GetCurrentBlock()
	if from < cache.GetStartBlock() {
		from = cache.GetStartBlock()
	}

	if cache.GetEndBlock() >= from {
		cache.DurationRemaining = cache.GetEndBlock() - from
	} else {
		cache.DurationRemaining = 0
	}
	cache.DurationRemainingLoaded = true
	return cache.DurationRemainingLoaded
}

// LoadDurationPast measures the served part of the agreement, which the provider
// cancellation penalty is priced from. It stops at the end block: an agreement
// settled after it ended has served its full duration and no more.
//
// Together with LoadDurationRemaining this keeps past + remaining == duration at
// every height, which is the identity the collateral accounting rests on.
func (cache *AgreementCache) LoadDurationPast() bool {
	to := cache.GetCurrentBlock()
	if to > cache.GetEndBlock() {
		to = cache.GetEndBlock()
	}

	if to >= cache.GetStartBlock() {
		cache.DurationPast = to - cache.GetStartBlock()
	} else {
		cache.DurationPast = 0
	}
	cache.DurationPastLoaded = true
	return cache.DurationPastLoaded
}

func (cache *AgreementCache) LoadDuration() bool {
	if cache.GetEndBlock() >= cache.GetStartBlock() {
		cache.Duration = cache.GetEndBlock() - cache.GetStartBlock()
	} else {
		cache.Duration = 0
	}
	cache.DurationLoaded = true
	return cache.DurationLoaded
}

// Update Permission
func (cache *AgreementCache) CanUpdate(activePlayer *PlayerCache) error {
	return cache.CC.PermissionCheck(cache, activePlayer, types.PermUpdate)
}

func (cache *AgreementCache) CanAllocateAsSourceBy(activePlayer *PlayerCache) error {
    return types.NewAllocationError(cache.ID(), "unacceptable_source")
}


/* Getters
 * These will always perform a Load first on the appropriate data if it hasn't occurred yet.
 */

func (cache *AgreementCache) GetAgreement() types.Agreement {
	if !cache.AgreementLoaded {
		cache.LoadAgreement()
	}
	return cache.Agreement
}
func (cache *AgreementCache) GetAgreementId() string { return cache.AgreementId }

func (cache *AgreementCache) GetCurrentBlock() uint64 {
	if !cache.CurrentBlockLoaded {
		cache.LoadCurrentBlock()
	}
	return cache.CurrentBlock
}

// Get the Owner data
func (cache *AgreementCache) GetOwnerId() string {
	if !cache.AgreementLoaded {
		cache.LoadAgreement()
	}
	return cache.Agreement.Owner
}
func (cache *AgreementCache) GetOwner() *PlayerCache {
    return cache.CC.GetPlayer(cache.GetOwnerId())
}

// Get the Provider data
func (cache *AgreementCache) GetProviderId() string {
	if !cache.AgreementLoaded {
		cache.LoadAgreement()
	}
	return cache.Agreement.ProviderId
}
func (cache *AgreementCache) GetProvider() *ProviderCache {
	return cache.CC.GetProvider(cache.GetProviderId())
}

func (cache *AgreementCache) GetAllocationId() string {
	if !cache.AgreementLoaded {
		cache.LoadAgreement()
	}
	return cache.Agreement.AllocationId
}

func (cache *AgreementCache) GetAllocation() (*AllocationCache, bool) {
	return cache.CC.GetAllocation(cache.GetAllocationId())
}

func (cache *AgreementCache) GetCapacity() uint64 {
	if !cache.AgreementLoaded {
		cache.LoadAgreement()
	}
	return cache.Agreement.Capacity
}
func (cache *AgreementCache) GetCapacityInt() math.Int {
	return math.NewIntFromUint64(cache.GetCapacity())
}
func (cache *AgreementCache) GetCapacityDec() math.LegacyDec {
	return math.LegacyNewDecFromInt(cache.GetCapacityInt())
}

func (cache *AgreementCache) GetStartBlock() uint64 {
	if !cache.AgreementLoaded {
		cache.LoadAgreement()
	}
	return cache.Agreement.StartBlock
}
func (cache *AgreementCache) GetEndBlock() uint64 {
	if !cache.AgreementLoaded {
		cache.LoadAgreement()
	}
	return cache.Agreement.EndBlock
}

func (cache *AgreementCache) GetCreator() string {
	if !cache.AgreementLoaded {
		cache.LoadAgreement()
	}
	return cache.Agreement.Creator
}

func (cache *AgreementCache) GetDuration() uint64 {
	if !cache.DurationLoaded {
		cache.LoadDuration()
	}
	return cache.Duration
}
func (cache *AgreementCache) GetDurationInt() math.Int {
	return math.NewIntFromUint64(cache.GetDuration())
}

func (cache *AgreementCache) GetDurationPast() uint64 {
	if !cache.DurationPastLoaded {
		cache.LoadDurationPast()
	}
	return cache.DurationPast
}
func (cache *AgreementCache) GetDurationPastInt() math.Int {
	return math.NewIntFromUint64(cache.GetDurationPast())
}
func (cache *AgreementCache) GetDurationPastDec() math.LegacyDec {
	return math.LegacyNewDecFromInt(cache.GetDurationPastInt())
}

func (cache *AgreementCache) GetDurationRemaining() uint64 {
	if !cache.DurationRemainingLoaded {
		cache.LoadDurationRemaining()
	}
	return cache.DurationRemaining
}
func (cache *AgreementCache) GetDurationRemainingInt() math.Int {
	return math.NewIntFromUint64(cache.GetDurationRemaining())
}
func (cache *AgreementCache) GetDurationRemainingDec() math.LegacyDec {
	return math.LegacyNewDecFromInt(cache.GetDurationRemainingInt())
}

func (cache *AgreementCache) GetOriginalCollateral() math.Int {
	return cache.GetDurationInt().Mul(cache.GetProvider().GetRate().Amount).Mul(cache.GetCapacityInt())
}

func (cache *AgreementCache) GetRemainingCollateral() math.Int {
	return cache.GetDurationRemainingInt().Mul(cache.GetProvider().GetRate().Amount).Mul(cache.GetCapacityInt())
}
func (cache *AgreementCache) GetRemainingCollateralDec() math.LegacyDec {
	return math.LegacyNewDecFromInt(cache.GetRemainingCollateral())
}

/* Committing Setters */

// consumerAccount resolves where this agreement's consumer is paid. The
// PlayerCache accessor drops the bech32 error and hands back an empty address,
// which SendCoins would reject with the error every payout used to discard, so
// the collateral would vanish with the agreement. Resolve it explicitly instead.
func (cache *AgreementCache) consumerAccount() (sdk.AccAddress, error) {
	address := cache.GetOwner().GetPrimaryAddress()

	account, err := sdk.AccAddressFromBech32(address)
	if err != nil {
		return nil, types.NewAgreementSettlementError(cache.GetAgreementId(), "invalid_payout_address").
			WithProvider(cache.GetProviderId()).
			WithPlayer(cache.GetOwnerId()).
			WithAddress(address)
	}

	return account, nil
}

// payConsumer moves collateral the consumer is owed out of the provider's pool.
// This is the consumer's own money and has first claim on the pool, so it is
// paid in full or not at all.
func (cache *AgreementCache) payConsumer(amount math.Int) error {
	if !amount.IsPositive() {
		return nil
	}

	account, err := cache.consumerAccount()
	if err != nil {
		return err
	}

	coins := sdk.NewCoins(sdk.NewCoin(cache.GetProvider().GetRate().Denom, amount))

	if err := cache.CC.k.bankKeeper.SendCoins(cache.CC.ctx, cache.GetProvider().GetCollateralPoolLocation(), account, coins); err != nil {
		return types.NewAgreementSettlementError(cache.GetAgreementId(), "payout_failed").
			WithProvider(cache.GetProviderId()).
			WithPlayer(cache.GetOwnerId()).
			WithAddress(account.String()).
			WithAmount(coins.String())
	}

	return nil
}

// providerCancellationPenaltyAmount is what the provider forfeits for the time
// already served, based on the penalty rate they published.
func (cache *AgreementCache) providerCancellationPenaltyAmount() math.Int {
	rate := math.LegacyNewDecFromInt(cache.GetProvider().GetRate().Amount)
	return cache.GetDurationPastDec().Mul(rate).Mul(cache.GetCapacityDec()).Mul(cache.GetProvider().GetProviderCancellationPenalty()).TruncateInt()
}

// PayoutVoidedProviderCancellationPenalty releases the cancellation penalty to
// the provider, for the cases where they did not cancel and so keep it. This is
// provider revenue and is subordinate to consumer collateral, so it is clamped
// to what the pool can spare.
func (cache *AgreementCache) PayoutVoidedProviderCancellationPenalty() {
	provider := cache.GetProvider()
	provider.SweepRevenue(provider.GetEarningsPoolLocation(), cache.providerCancellationPenaltyAmount(), cache.GetAgreementId())
}

func (cache *AgreementCache) PayoutProviderCancellationPenalty() error {
	return cache.payConsumer(cache.providerCancellationPenaltyAmount())
}

func (cache *AgreementCache) PayoutConsumerCancellationPenaltyAndReturnCollateral() error {
	penalty := cache.GetRemainingCollateralDec().Mul(cache.GetProvider().GetConsumerCancellationPenalty()).TruncateInt()

	// The consumer gets what is left of their collateral first; the penalty they
	// forfeit is provider revenue and only settles once they are whole.
	if err := cache.payConsumer(cache.GetRemainingCollateral().Sub(penalty)); err != nil {
		return err
	}

	provider := cache.GetProvider()
	provider.SweepRevenue(provider.GetEarningsPoolLocation(), penalty, cache.GetAgreementId())

	return nil
}

func (cache *AgreementCache) ReturnRemainingCollateral() error {
	return cache.payConsumer(cache.GetRemainingCollateral())
}

// beginTeardown claims the single settlement this agreement is entitled to in
// this operation. A false return means settlement is already underway and the
// caller is the reciprocal leg of it, so it must not pay out again.
func (cache *AgreementCache) beginTeardown() bool {
	if cache.TearingDown {
		return false
	}
	cache.TearingDown = true
	return true
}

// IsTearingDown reports whether settlement of this agreement has already begun.
func (cache *AgreementCache) IsTearingDown() bool {
	return cache.TearingDown
}

// destroyAllocation tears down the allocation backing this agreement. The
// allocation's own Destroy would normally settle its agreement, but the
// TearingDown flag set by beginTeardown stops it re-entering here.
//
// An allocation that is already gone is nothing to tear down, and both ways of
// discovering that have to agree. GetAllocation answers from cc.allocations
// without re-reading the store, so an allocation removed earlier in the
// operation still reports found; Destroy's own re-read is what notices, and it
// reports it as unknown_allocation. Letting that error out would abort Expire
// before the load decrement and the removal, leaving the provider billing this
// agreement's capacity against the shared collateral pool for good — an expiry
// runs in a block hook and gets exactly one attempt, and the absence it tripped
// over is permanent, so there is nothing a later block could do about it.
func (cache *AgreementCache) destroyAllocation() error {
	allocation, found := cache.GetAllocation()
	if !found {
		return nil
	}
	if !allocation.LoadAllocation() {
		return nil
	}
	return allocation.Destroy()
}

// removeAgreement drops the agreement and its indexes from the store.
//
// The removal is immediate because the rest of the operation must not see the
// agreement any more: ProviderCache.Delete walks the provider index, and
// AllocationCache.Destroy looks the agreement up out of the store.
//
// Deleted is set but Changed deliberately is not. RemoveAgreement already emits
// the EventDelete, and EventDelete is a public API, so forcing Changed would
// make Commit clear the agreement a second time and emit a duplicate. Leaving
// Changed alone means Commit is a no-op in the normal case, and in the case that
// motivates the flag — something mutated the agreement earlier in the operation,
// which sets Changed itself — it takes the Deleted branch instead of writing the
// agreement back, so a settled agreement can never be resurrected.
func (cache *AgreementCache) removeAgreement() {
	cache.CC.k.RemoveAgreementExpirationIndex(cache.CC.ctx, cache.GetEndBlock(), cache.GetAgreementId())
	cache.CC.k.RemoveAgreement(cache.CC.ctx, cache.GetAgreement())

	cache.Deleted = true

	cache.CC.ClearPermissionsForObject(cache.ID())
}

// checkpointProvider sweeps the revenue the provider has earned up to now. It
// must run while this agreement still counts toward the provider's load and
// before any load change, because Checkpoint bills the current load across the
// whole span since the last checkpoint. Every teardown path calls it, rather
// than trusting each caller to: the allocation-driven paths (grid brownout,
// struct destruction, allocation delete) have no natural place to do so.
func (cache *AgreementCache) checkpointProvider() error {
	return cache.GetProvider().Checkpoint()
}

// assertConsumerPayable resolves the consumer's payout destination before
// settlement touches anything.
//
// Teardown is not transactional. Half of these paths run in block hooks that
// cannot abort, so an error raised partway through leaves the bank transfers and
// load changes already made behind with no retry. Everything that can fail on
// data alone is therefore checked up front, which leaves only a genuinely short
// collateral pool able to fail late — the case the solvency invariant and the
// shortfall event exist to surface.
func (cache *AgreementCache) assertConsumerPayable() error {
	_, err := cache.consumerAccount()
	return err
}

func (cache *AgreementCache) PrematureCloseByProvider() error {
	if !cache.beginTeardown() {
		return nil
	}

	if err := cache.assertConsumerPayable(); err != nil {
		return err
	}

	// Destroy the Allocation. Nothing about allocation teardown touches provider
	// load or the bank, so doing it before the settlement keeps every fallible
	// step ahead of every mutation.
	if err := cache.destroyAllocation(); err != nil {
		return err
	}

	if err := cache.checkpointProvider(); err != nil {
		return err
	}

	// Payout Cancellation Penalty
	if err := cache.PayoutProviderCancellationPenalty(); err != nil {
		return err
	}
	if err := cache.ReturnRemainingCollateral(); err != nil {
		return err
	}

	// Decrease the Load on the Provider
	cache.GetProvider().AgreementLoadDecrease(cache.GetCapacity())

	// Destroy the Agreement
	cache.removeAgreement()

	return nil
}

func (cache *AgreementCache) PrematureCloseByConsumer() error {
	if !cache.beginTeardown() {
		return nil
	}

	if err := cache.assertConsumerPayable(); err != nil {
		return err
	}

	// Destroy the Allocation before settling, so a failure cannot leave the bank
	// transfers half done. See PrematureCloseByProvider.
	if err := cache.destroyAllocation(); err != nil {
		return err
	}

	if err := cache.checkpointProvider(); err != nil {
		return err
	}

	if err := cache.PayoutConsumerCancellationPenaltyAndReturnCollateral(); err != nil {
		return err
	}

	// Decrease the Load on the Provider
	cache.GetProvider().AgreementLoadDecrease(cache.GetCapacity())

	// Destroy the Agreement
	cache.removeAgreement()

	return nil

}

// PrematureCloseByAllocation settles an agreement whose allocation has already
// been torn down. It must not destroy the allocation itself: AllocationCache.Destroy
// is the only caller.
//
// This is the teardown that runs in block hooks — grid brownout, struct
// destruction, batched allocation teardown — so it is the one that most needs its
// fallible work done before it mutates anything.
func (cache *AgreementCache) PrematureCloseByAllocation() error {
	if !cache.beginTeardown() {
		return nil
	}

	if err := cache.assertConsumerPayable(); err != nil {
		return err
	}

	if err := cache.checkpointProvider(); err != nil {
		return err
	}

	if err := cache.PayoutProviderCancellationPenalty(); err != nil {
		return err
	}
	if err := cache.ReturnRemainingCollateral(); err != nil {
		return err
	}

	// Decrease the Load on the Provider
	cache.GetProvider().AgreementLoadDecrease(cache.GetCapacity())

	// Destroy the Agreement
	cache.removeAgreement()

	return nil

}

// Expire settles an agreement that ran its full term. The consumer is owed
// nothing: they received the service they paid for, and the cancellation penalty
// is the provider's because they did not cancel.
//
// This runs in the EndBlocker, where a caller can only log and move on, so the
// allocation is destroyed first: it is the one step that can fail, and doing it
// before the settlement means a failure cannot leave revenue swept and load
// released against an agreement that is still in the store.
func (cache *AgreementCache) Expire() error {
	if !cache.beginTeardown() {
		return nil
	}

	// Destroy the Allocation
	if err := cache.destroyAllocation(); err != nil {
		return err
	}

	// This guard is inert and has to stay that way: Checkpoint cannot fail, and
	// the two statements below are the ones an expiry gets a single chance to
	// perform. Returning here would not defer the teardown, it would cancel it,
	// leaving the agreement holding capacity in the provider's load that every
	// later checkpoint bills against other consumers' escrow. If Checkpoint ever
	// becomes fallible, this call site has to stop propagating rather than start.
	if err := cache.checkpointProvider(); err != nil {
		return err
	}

	cache.PayoutVoidedProviderCancellationPenalty()

	// Decrease the Load on the Provider
	cache.GetProvider().AgreementLoadDecrease(cache.GetCapacity())

	// Destroy the Agreement
	cache.removeAgreement()

	return nil

}

/* Setters - SET DOES NOT COMMIT()
 */

func (cache *AgreementCache) ResetStartBlock() {
	uctx := sdk.UnwrapSDKContext(cache.CC.ctx)
	startBlock := uint64(uctx.BlockHeight())
	cache.SetStartBlock(startBlock)
}

func (cache *AgreementCache) SetStartBlock(startBlock uint64) {
	if !cache.AgreementLoaded {
		cache.LoadAgreement()
	}
	cache.Agreement.StartBlock = startBlock

	cache.DurationLoaded = false
	cache.DurationPastLoaded = false
	cache.DurationRemainingLoaded = false

	cache.Changed = true
}

func (cache *AgreementCache) SetEndBlock(endBlock uint64) {
	if !cache.AgreementLoaded {
		cache.LoadAgreement()
	}

	cache.PreviousEndBlock = cache.Agreement.EndBlock
	cache.Agreement.EndBlock = endBlock
    cache.EndBlockChanged = true

	cache.DurationLoaded = false
	cache.DurationPastLoaded = false
	cache.DurationRemainingLoaded = false

	cache.Changed = true
}

// rescaledDuration re-prices the unearned span for a new capacity: the
// collateral left over buys proportionally less time at a higher capacity and
// more at a lower one.
//
// The multiply happens in math.Int because remaining * capacity overflows uint64
// well inside the range a provider may publish — SetDurationRange puts no
// ceiling on durationMaximum.
func (cache *AgreementCache) rescaledDuration(newCapacity uint64) (uint64, error) {
	scaled := math.NewIntFromUint64(cache.GetDurationRemaining()).
		Mul(cache.GetCapacityInt()).
		Quo(math.NewIntFromUint64(newCapacity))

	if !scaled.IsUint64() {
		return 0, types.NewParameterValidationError("capacity", newCapacity, "duration_overflow")
	}

	return scaled.Uint64(), nil
}

func (cache *AgreementCache) CapacityIncrease(amount uint64) error {
	if amount == 0 {
		return types.NewParameterValidationError("capacity", amount, "no_change")
	}

	if cache.GetProvider().GetSubstation().GetAvailableCapacity() < amount {
		return types.NewParameterValidationError("capacity", amount, "exceeds_available").WithSubstation(cache.GetProvider().GetSubstationId()).WithRange(0, cache.GetProvider().GetSubstation().GetAvailableCapacity())
	}

	newCapacity := cache.GetCapacity() + amount
	if newCapacity < cache.GetCapacity() {
		return types.NewParameterValidationError("capacity", amount, "above_maximum").WithRange(cache.GetProvider().GetCapacityMinimum(), cache.GetProvider().GetCapacityMaximum())
	}

	// A capacity change must land inside the terms the provider published, the
	// same range that gated the agreement being opened at all.
	if err := cache.GetProvider().AgreementCapacityVerify(newCapacity); err != nil {
		return err
	}

	newDuration, err := cache.rescaledDuration(newCapacity)
	if err != nil {
		return err
	}
	if err := cache.GetProvider().AgreementDurationVerify(newDuration); err != nil {
		return err
	}

	// Everything that can fail is behind us. The penalty has to be priced before
	// the mutations below, because SetStartBlock resets the elapsed span it is
	// measured over and the capacity write changes the rate it is charged at.
	cache.PayoutVoidedProviderCancellationPenalty()

	cache.SetStartBlock(cache.GetCurrentBlock())
	cache.SetEndBlock(cache.GetStartBlock() + newDuration)

	// Provider Load Increase
	cache.GetProvider().AgreementLoadIncrease(amount)

	cache.Agreement.Capacity = newCapacity

	// Increase the Allocation
	allocation, allocationFound := cache.GetAllocation()
	if allocationFound {
		// TODO error handling
		allocation.SetPower(cache.GetCapacity())
	}

	cache.Changed = true

	return nil
}

func (cache *AgreementCache) CapacityDecrease(amount uint64) error {
	if amount == 0 {
		return types.NewParameterValidationError("capacity", amount, "no_change")
	}

	if cache.GetCapacity() < amount {
		return types.NewParameterValidationError("capacity", amount, "below_minimum").WithRange(0, cache.GetCapacity())
	}

	newCapacity := cache.GetCapacity() - amount
    if newCapacity == 0 {
		return types.NewParameterValidationError("capacity", amount, "below_minimum").WithRange(0, cache.GetCapacity())
    }

	// A capacity change must land inside the terms the provider published, the
	// same range that gated the agreement being opened at all. Without this a
	// decrease toward capacity 1 stretches the remaining span by the old
	// capacity, far past the advertised duration maximum.
	if err := cache.GetProvider().AgreementCapacityVerify(newCapacity); err != nil {
		return err
	}

	newDuration, err := cache.rescaledDuration(newCapacity)
	if err != nil {
		return err
	}
	if err := cache.GetProvider().AgreementDurationVerify(newDuration); err != nil {
		return err
	}

	// Everything that can fail is behind us. The penalty has to be priced before
	// the mutations below, because SetStartBlock resets the elapsed span it is
	// measured over and the capacity write changes the rate it is charged at.
	cache.PayoutVoidedProviderCancellationPenalty()

	cache.SetStartBlock(cache.GetCurrentBlock())
	cache.SetEndBlock(cache.GetStartBlock() + newDuration)

	// Provider Load Decrease
	cache.GetProvider().AgreementLoadDecrease(amount)

	cache.Agreement.Capacity = newCapacity

	// Decrease the Allocation
	allocation, allocationFound := cache.GetAllocation()
	if allocationFound {
        allocation.SetPower(cache.GetCapacity())
	}

	cache.Changed = true

	return nil
}

func (cache *AgreementCache) DurationIncrease(amount uint64) error {

	newDuration := (cache.GetEndBlock() - cache.GetStartBlock()) + amount
	verifyError := cache.GetProvider().AgreementDurationVerify(newDuration)
	if verifyError != nil {
		return verifyError
	}

	cache.SetEndBlock(cache.GetEndBlock() + amount)
	cache.Changed = true

	return nil
}
