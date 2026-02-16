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

	// TODO allocationCache

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

func (cache *AgreementCache) LoadDurationRemaining() bool {
	cache.DurationRemaining = cache.GetEndBlock() - cache.GetCurrentBlock()
	cache.DurationRemainingLoaded = true
	return cache.DurationRemainingLoaded
}

func (cache *AgreementCache) LoadDurationPast() bool {
	cache.DurationPast = cache.GetCurrentBlock() - cache.GetStartBlock()
	cache.DurationPastLoaded = true
	return cache.DurationPastLoaded
}

func (cache *AgreementCache) LoadDuration() bool {
	cache.Duration = cache.GetEndBlock() - cache.GetStartBlock()
	cache.DurationLoaded = true
	return cache.DurationLoaded
}

// Update Permission
func (cache *AgreementCache) CanUpdate(activePlayer *PlayerCache) error {
	return cache.PermissionCheck(types.PermissionUpdate, activePlayer)
}

func (cache *AgreementCache) PermissionCheck(permission types.Permission, activePlayer *PlayerCache) error {
	// Make sure the address calling this has permissions
	if !cache.CC.PermissionHasOneOf(GetAddressPermissionIDBytes(activePlayer.GetActiveAddress()), permission) {
		return types.NewPermissionError("address", activePlayer.GetActiveAddress(), "", "", uint64(permission), "agreement_action")
	}

	if !activePlayer.HasPlayerAccount() {
		return types.NewPlayerRequiredError(activePlayer.GetActiveAddress(), "agreement_action")
	} else {
		if activePlayer.GetPlayerId() != cache.GetOwnerId() {
			if !cache.CC.PermissionHasOneOf(GetObjectPermissionIDBytes(cache.GetAgreementId(), activePlayer.GetPlayerId()), permission) {
				return types.NewPermissionError("player", activePlayer.GetPlayerId(), "agreement", cache.GetAgreementId(), uint64(permission), "agreement_action")
			}
		}
	}
	return nil
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
    owner, _ := cache.CC.GetPlayer(cache.GetOwnerId())
    return owner
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

// TODO func GetAllocation()
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
func (cache *AgreementCache) PayoutVoidedProviderCancellationPenalty() {
	rate := math.LegacyNewDecFromInt(cache.GetProvider().GetRate().Amount)
	penalty := cache.GetDurationPastDec().Mul(rate).Mul(cache.GetCapacityDec()).Mul(cache.GetProvider().GetProviderCancellationPenalty()).TruncateInt()
	penaltyCoin := sdk.NewCoins(sdk.NewCoin(cache.GetProvider().GetRate().Denom, penalty))

	cache.CC.k.bankKeeper.SendCoins(cache.CC.ctx, cache.GetProvider().GetCollateralPoolLocation(), cache.GetProvider().GetEarningsPoolLocation(), penaltyCoin)

}

func (cache *AgreementCache) PayoutProviderCancellationPenalty() {
	rate := math.LegacyNewDecFromInt(cache.GetProvider().GetRate().Amount)
	penalty := cache.GetDurationPastDec().Mul(rate).Mul(cache.GetCapacityDec()).Mul(cache.GetProvider().GetProviderCancellationPenalty()).TruncateInt()
	penaltyCoin := sdk.NewCoins(sdk.NewCoin(cache.GetProvider().GetRate().Denom, penalty))

	cache.CC.k.bankKeeper.SendCoins(cache.CC.ctx, cache.GetProvider().GetCollateralPoolLocation(), cache.GetOwner().GetPrimaryAccount(), penaltyCoin)

}

func (cache *AgreementCache) PayoutConsumerCancellationPenaltyAndReturnCollateral() {
	penalty := cache.GetRemainingCollateralDec().Mul(cache.GetProvider().GetConsumerCancellationPenalty()).TruncateInt()
	penaltyCoin := sdk.NewCoins(sdk.NewCoin(cache.GetProvider().GetRate().Denom, penalty))

	cache.CC.k.bankKeeper.SendCoins(cache.CC.ctx, cache.GetProvider().GetCollateralPoolLocation(), cache.GetProvider().GetEarningsPoolLocation(), penaltyCoin)

	remainingCollateral := cache.GetRemainingCollateral().Sub(penalty)
	remainingCollateralCoin := sdk.NewCoins(sdk.NewCoin(cache.GetProvider().GetRate().Denom, remainingCollateral))

	cache.CC.k.bankKeeper.SendCoins(cache.CC.ctx, cache.GetProvider().GetCollateralPoolLocation(), cache.GetOwner().GetPrimaryAccount(), remainingCollateralCoin)

}

func (cache *AgreementCache) ReturnRemainingCollateral() {
	remainingCollateralCoin := sdk.NewCoins(sdk.NewCoin(cache.GetProvider().GetRate().Denom, cache.GetRemainingCollateral()))

	cache.CC.k.bankKeeper.SendCoins(cache.CC.ctx, cache.GetProvider().GetCollateralPoolLocation(), cache.GetOwner().GetPrimaryAccount(), remainingCollateralCoin)
}

func (cache *AgreementCache) PrematureCloseByProvider() error {
	// Payout Cancellation Penalty
	cache.PayoutProviderCancellationPenalty()
	cache.ReturnRemainingCollateral()

	// Destroy the Allocation
	allocation, found := cache.GetAllocation()
    if found {
        allocation.Destroy()
    }

	// Decrease the Load on the Provider
	cache.GetProvider().AgreementLoadDecrease(cache.GetCapacity())

	// Destroy the Agreement
	cache.CC.k.RemoveAgreementExpirationIndex(cache.CC.ctx, cache.GetEndBlock(), cache.GetAgreementId())
	cache.CC.k.RemoveAgreement(cache.CC.ctx, cache.GetAgreement())

	return nil
}

func (cache *AgreementCache) PrematureCloseByConsumer() error {

	cache.PayoutConsumerCancellationPenaltyAndReturnCollateral()

	// Destroy the Allocation
	allocation, found := cache.GetAllocation()
    if found {
        allocation.Destroy()
    }

	// Decrease the Load on the Provider
	cache.GetProvider().AgreementLoadDecrease(cache.GetCapacity())

	// Destroy the Agreement
	cache.CC.k.RemoveAgreementExpirationIndex(cache.CC.ctx, cache.GetEndBlock(), cache.GetAgreementId())
	cache.CC.k.RemoveAgreement(cache.CC.ctx, cache.GetAgreement())

	return nil

}

func (cache *AgreementCache) PrematureCloseByAllocation() error {
	cache.PayoutProviderCancellationPenalty()
	cache.ReturnRemainingCollateral()

	// Decrease the Load on the Provider
	cache.GetProvider().AgreementLoadDecrease(cache.GetCapacity())
	cache.GetProvider().Commit()

	// Destroy the Agreement
	cache.CC.k.RemoveAgreementExpirationIndex(cache.CC.ctx, cache.GetEndBlock(), cache.GetAgreementId())
	cache.CC.k.RemoveAgreement(cache.CC.ctx, cache.GetAgreement())

	return nil

}

func (cache *AgreementCache) Expire() error {
	cache.PayoutVoidedProviderCancellationPenalty()

	// Decrease the Load on the Provider
	cache.GetProvider().AgreementLoadDecrease(cache.GetCapacity())

	// Destroy the Allocation
	allocation, found := cache.GetAllocation()
    if found {
        allocation.Destroy()
    }

	// Destroy the Agreement
	cache.CC.k.RemoveAgreementExpirationIndex(cache.CC.ctx, cache.GetEndBlock(), cache.GetAgreementId())
	cache.CC.k.RemoveAgreement(cache.CC.ctx, cache.GetAgreement())

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

func (cache *AgreementCache) CapacityIncrease(amount uint64) error {
	if cache.GetProvider().GetSubstation().GetAvailableCapacity() < amount {
		return types.NewParameterValidationError("capacity", amount, "exceeds_available").WithSubstation(cache.GetProvider().GetSubstationId()).WithRange(0, cache.GetProvider().GetSubstation().GetAvailableCapacity())
	}

	cache.PayoutVoidedProviderCancellationPenalty()

	// new duration length
	// remaining duration = end block - current block
	// new duration = (remaining duration * old capacity) / new capacity .Truncate()
	newCapacity := cache.GetCapacity() + amount
	newDuration := (cache.GetDurationRemaining() * cache.GetCapacity()) / newCapacity

	cache.SetStartBlock(cache.GetCurrentBlock())
	cache.SetEndBlock(cache.GetStartBlock() + newDuration)

	// Provider Load Increase
	cache.GetProvider().AgreementLoadIncrease(amount)

	cache.Agreement.Capacity = cache.GetCapacity() + amount

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
	cache.PayoutVoidedProviderCancellationPenalty()

	// new duration length
	// remaining duration = end block - current block
	// new duration = (remaining duration * old capacity) / new capacity .Truncate()
	if cache.GetCapacity() < amount {
		return types.NewParameterValidationError("capacity", amount, "below_minimum").WithRange(0, cache.GetCapacity())
	}

	newCapacity := cache.GetCapacity() - amount
    if newCapacity == 0 {
		return types.NewParameterValidationError("capacity", amount, "below_minimum").WithRange(0, cache.GetCapacity())
    }

	newDuration := (cache.GetDurationRemaining() * cache.GetCapacity()) / newCapacity

	cache.SetStartBlock(cache.GetCurrentBlock())
	cache.SetEndBlock(cache.GetStartBlock() + newDuration)

	// Provider Load Increase
	cache.GetProvider().AgreementLoadDecrease(amount)

	cache.Agreement.Capacity = cache.GetCapacity() - amount

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
	verifyError := cache.GetProvider().AgreementVerify(cache.GetCapacity(), newDuration)
	if verifyError != nil {
		return verifyError
	}

	cache.SetEndBlock(cache.GetEndBlock() + amount)
	cache.Changed = true

	return nil
}
