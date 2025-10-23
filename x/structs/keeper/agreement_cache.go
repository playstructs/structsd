package keeper

import (
	"context"

    sdk "github.com/cosmos/cosmos-sdk/types"
    sdkerrors "cosmossdk.io/errors"
	"structs/x/structs/types"
	"cosmossdk.io/math"
)

type AgreementCache struct {
	AgreementId string
	K          *Keeper
	Ctx        context.Context

	AnyChange bool

	Ready bool

	AgreementLoaded  bool
	AgreementChanged bool
	Agreement        types.Agreement

	PreviousEndBlock uint64
	EndBlockChanged bool

	OwnerLoaded bool
	Owner       *PlayerCache

	ProviderLoaded bool
	Provider       *ProviderCache

    DurationRemaining uint64
    DurationRemainingLoaded bool

    DurationPast uint64
    DurationPastLoaded bool

    Duration uint64
    DurationLoaded bool

    CurrentBlock uint64
    CurrentBlockLoaded bool

	// TODO allocationCache


}

// Build this initial Agreement Cache object
func (k *Keeper) GetAgreementCacheFromId(ctx context.Context, agreementId string) AgreementCache {
	return AgreementCache{
		AgreementId: agreementId,
		K:          k,
		Ctx:        ctx,

		AnyChange: false,

		OwnerLoaded: false,

		ProviderLoaded: false,

		AgreementLoaded:  false,
		AgreementChanged: false,

		DurationRemainingLoaded: false,
		DurationPastLoaded: false,
		DurationLoaded: false,

		CurrentBlockLoaded: false,
	}
}

func (cache *AgreementCache) Commit() {
	cache.AnyChange = false

	cache.K.logger.Debug("Updating Agreement From Cache","agreementId",cache.AgreementId)

	if cache.AgreementChanged {
		cache.K.SetAgreement(cache.Ctx, cache.Agreement)
		cache.AgreementChanged = false
	}

	if cache.Owner != nil && cache.GetOwner().IsChanged() {
		cache.GetOwner().Commit()
	}

	if cache.Provider != nil && cache.GetProvider().IsChanged() {
		cache.GetProvider().Commit()
	}

	if cache.EndBlockChanged {
	    if (cache.PreviousEndBlock > 0) {
	        cache.K.RemoveAgreementExpirationIndex(cache.Ctx, cache.PreviousEndBlock, cache.GetAgreementId())
	    }
	    cache.K.SetAgreementExpirationIndex(cache.Ctx, cache.GetEndBlock(), cache.GetAgreementId())
	}

}

func (cache *AgreementCache) IsChanged() bool {
	return cache.AnyChange
}

func (cache *AgreementCache) Changed() {
	cache.AnyChange = true
}

/* Separate Loading functions for each of the underlying containers */

// Load the Agreement record
func (cache *AgreementCache) LoadAgreement() bool {
	agreement, agreementFound := cache.K.GetAgreement(cache.Ctx, cache.AgreementId)

	if agreementFound {
		cache.Agreement = agreement
		cache.AgreementLoaded = true
	}

	return agreementFound
}

// Load the Player data
func (cache *AgreementCache) LoadOwner() bool {
	newOwner, _ := cache.K.GetPlayerCacheFromId(cache.Ctx, cache.GetOwnerId())
	cache.Owner = &newOwner
	cache.OwnerLoaded = true
	return cache.OwnerLoaded
}

func (cache *AgreementCache) LoadCurrentBlock() bool {
    uctx := sdk.UnwrapSDKContext(cache.Ctx)
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

func (cache *AgreementCache) ManualLoadOwner(owner *PlayerCache) {
    cache.Owner = owner
    cache.OwnerLoaded = true
}

// Load the Agreements Provider
func (cache *AgreementCache) LoadProvider() bool {
	newProvider := cache.K.GetProviderCacheFromId(cache.Ctx, cache.GetProviderId())
	cache.Provider = &newProvider
	cache.ProviderLoaded = true
	return cache.ProviderLoaded
}

func (cache *AgreementCache) ManualLoadProvider(provider *ProviderCache) {
    cache.Provider = provider
    cache.ProviderLoaded = true
}



// Update Permission
func (cache *AgreementCache) CanUpdate(activePlayer *PlayerCache) (error) {
    return cache.PermissionCheck(types.PermissionUpdate, activePlayer)
}


func (cache *AgreementCache) PermissionCheck(permission types.Permission, activePlayer *PlayerCache) (error) {
    // Make sure the address calling this has permissions
    if (!cache.K.PermissionHasOneOf(cache.Ctx, GetAddressPermissionIDBytes(activePlayer.GetActiveAddress()), permission)) {
        return sdkerrors.Wrapf(types.ErrPermission, "Calling address (%s) has no (%d) permissions ", activePlayer.GetActiveAddress(), permission)
    }

    if !activePlayer.HasPlayerAccount() {
        return sdkerrors.Wrapf(types.ErrPermission, "Calling address (%s) has no Account", activePlayer.GetActiveAddress())
    } else {
        if (activePlayer.GetPlayerId() != cache.GetOwnerId()) {
            if (!cache.K.PermissionHasOneOf(cache.Ctx, GetObjectPermissionIDBytes(cache.GetAgreementId(), activePlayer.GetPlayerId()), permission)) {
               return sdkerrors.Wrapf(types.ErrPermission, "Calling account (%s) has no (%d) permissions on target agreement (%s)", activePlayer.GetPlayerId(), permission, cache.GetAgreementId())
            }
        }
    }
    return nil
}




/* Getters
 * These will always perform a Load first on the appropriate data if it hasn't occurred yet.
 */


func (cache *AgreementCache) GetAgreement() types.Agreement { if !cache.AgreementLoaded { cache.LoadAgreement() }; return cache.Agreement }
func (cache *AgreementCache) GetAgreementId() string { return cache.AgreementId }

func (cache *AgreementCache) GetCurrentBlock() uint64 { if !cache.CurrentBlockLoaded { cache.LoadCurrentBlock() }; return cache.CurrentBlock }

// Get the Owner data
func (cache *AgreementCache) GetOwnerId() string { if !cache.AgreementLoaded { cache.LoadAgreement() }; return cache.Agreement.Owner }
func (cache *AgreementCache) GetOwner() *PlayerCache { if !cache.OwnerLoaded { cache.LoadOwner() }; return cache.Owner }

// Get the Provider data
func (cache *AgreementCache) GetProviderId() string { if !cache.AgreementLoaded { cache.LoadAgreement() }; return cache.Agreement.ProviderId }
func (cache *AgreementCache) GetProvider() *ProviderCache { if !cache.ProviderLoaded { cache.LoadProvider() }; return cache.Provider }


func (cache *AgreementCache) GetAllocationId() string { if !cache.AgreementLoaded { cache.LoadAgreement() }; return cache.Agreement.AllocationId }
// TODO func GetAllocation()

func (cache *AgreementCache) GetCapacity() uint64 { if !cache.AgreementLoaded { cache.LoadAgreement() }; return cache.Agreement.Capacity }
func (cache *AgreementCache) GetCapacityInt() math.Int { return math.NewIntFromUint64(cache.GetCapacity()) }
func (cache *AgreementCache) GetCapacityDec() math.LegacyDec { return math.LegacyNewDecFromInt(cache.GetCapacityInt()) }

func (cache *AgreementCache) GetStartBlock() uint64 { if !cache.AgreementLoaded { cache.LoadAgreement() }; return cache.Agreement.StartBlock }
func (cache *AgreementCache) GetEndBlock() uint64 { if !cache.AgreementLoaded { cache.LoadAgreement() }; return cache.Agreement.EndBlock }

func (cache *AgreementCache) GetCreator() string { if !cache.AgreementLoaded { cache.LoadAgreement() }; return cache.Agreement.Creator }

func (cache *AgreementCache) GetDuration() uint64 { if !cache.DurationLoaded { cache.LoadDuration() }; return cache.Duration }
func (cache *AgreementCache) GetDurationInt() math.Int { return math.NewIntFromUint64(cache.GetDuration()) }

func (cache *AgreementCache) GetDurationPast() uint64 { if !cache.DurationPastLoaded { cache.LoadDurationPast() }; return cache.DurationPast }
func (cache *AgreementCache) GetDurationPastInt() math.Int { return math.NewIntFromUint64(cache.GetDurationPast()) }
func (cache *AgreementCache) GetDurationPastDec() math.LegacyDec { return math.LegacyNewDecFromInt(cache.GetDurationPastInt()) }

func (cache *AgreementCache) GetDurationRemaining() uint64 { if !cache.DurationRemainingLoaded { cache.LoadDurationRemaining() }; return cache.DurationRemaining }
func (cache *AgreementCache) GetDurationRemainingInt() math.Int {  return math.NewIntFromUint64(cache.GetDurationRemaining()) }
func (cache *AgreementCache) GetDurationRemainingDec() math.LegacyDec {  return math.LegacyNewDecFromInt(cache.GetDurationRemainingInt()) }

func (cache *AgreementCache) GetOriginalCollateral() math.Int {
    return cache.GetDurationInt().Mul(cache.GetProvider().GetRate().Amount).Mul(cache.GetCapacityInt())
}

func (cache *AgreementCache) GetRemainingCollateral() math.Int {
    return cache.GetDurationRemainingInt().Mul(cache.GetProvider().GetRate().Amount).Mul(cache.GetCapacityInt())
}
func (cache *AgreementCache) GetRemainingCollateralDec() math.LegacyDec { return math.LegacyNewDecFromInt(cache.GetRemainingCollateral()) }

/* Committing Setters */
func (cache *AgreementCache) PayoutVoidedProviderCancellationPenalty() {
    rate := math.LegacyNewDecFromInt(cache.GetProvider().GetRate().Amount)
    penalty := cache.GetDurationPastDec().Mul(rate).Mul(cache.GetCapacityDec()).Mul(cache.GetProvider().GetProviderCancellationPenalty()).TruncateInt()
    penaltyCoin := sdk.NewCoins(sdk.NewCoin(cache.GetProvider().GetRate().Denom, penalty))

    cache.K.bankKeeper.SendCoins(cache.Ctx, cache.GetProvider().GetCollateralPoolLocation(), cache.GetProvider().GetEarningsPoolLocation(), penaltyCoin)

}

func (cache *AgreementCache) PayoutProviderCancellationPenalty() {
    rate := math.LegacyNewDecFromInt(cache.GetProvider().GetRate().Amount)
    penalty := cache.GetDurationPastDec().Mul(rate).Mul(cache.GetCapacityDec()).Mul(cache.GetProvider().GetProviderCancellationPenalty()).TruncateInt()
    penaltyCoin := sdk.NewCoins(sdk.NewCoin(cache.GetProvider().GetRate().Denom, penalty))

    cache.K.bankKeeper.SendCoins(cache.Ctx, cache.GetProvider().GetCollateralPoolLocation(), cache.GetOwner().GetPrimaryAccount(), penaltyCoin)

}

func (cache *AgreementCache) PayoutConsumerCancellationPenaltyAndReturnCollateral() {
    penalty := cache.GetRemainingCollateralDec().Mul(cache.GetProvider().GetConsumerCancellationPenalty()).TruncateInt()
    penaltyCoin := sdk.NewCoins(sdk.NewCoin(cache.GetProvider().GetRate().Denom, penalty))

    cache.K.bankKeeper.SendCoins(cache.Ctx, cache.GetProvider().GetCollateralPoolLocation(), cache.GetProvider().GetEarningsPoolLocation(), penaltyCoin)

    remainingCollateral := cache.GetRemainingCollateral().Sub(penalty)
    remainingCollateralCoin := sdk.NewCoins(sdk.NewCoin(cache.GetProvider().GetRate().Denom, remainingCollateral))

    cache.K.bankKeeper.SendCoins(cache.Ctx, cache.GetProvider().GetCollateralPoolLocation(), cache.GetOwner().GetPrimaryAccount(), remainingCollateralCoin)

}

func (cache *AgreementCache) ReturnRemainingCollateral() {
    remainingCollateralCoin := sdk.NewCoins(sdk.NewCoin(cache.GetProvider().GetRate().Denom, cache.GetRemainingCollateral()))

    cache.K.bankKeeper.SendCoins(cache.Ctx, cache.GetProvider().GetCollateralPoolLocation(), cache.GetOwner().GetPrimaryAccount(), remainingCollateralCoin)
}

func (cache *AgreementCache) PrematureCloseByProvider() (error) {
    // Payout Cancellation Penalty
    cache.PayoutProviderCancellationPenalty()
    cache.ReturnRemainingCollateral()

    // Destroy the Allocation
    cache.K.DestroyAllocation(cache.Ctx, cache.GetAllocationId())

    // Decrease the Load on the Provider
    cache.GetProvider().AgreementLoadDecrease(cache.GetCapacity())

    // Destroy the Agreement
    cache.K.RemoveAgreementExpirationIndex(cache.Ctx, cache.GetEndBlock(), cache.GetAgreementId())
    cache.K.RemoveAgreement(cache.Ctx, cache.GetAgreement())

    return nil
}

func (cache *AgreementCache) PrematureCloseByConsumer() (error){

    cache.PayoutConsumerCancellationPenaltyAndReturnCollateral()

    // Destroy the Allocation
    cache.K.DestroyAllocation(cache.Ctx, cache.GetAllocationId())

    // Decrease the Load on the Provider
    cache.GetProvider().AgreementLoadDecrease(cache.GetCapacity())

    // Destroy the Agreement
    cache.K.RemoveAgreementExpirationIndex(cache.Ctx, cache.GetEndBlock(), cache.GetAgreementId())
    cache.K.RemoveAgreement(cache.Ctx, cache.GetAgreement())

    return  nil

}

func (cache *AgreementCache) PrematureCloseByAllocation() (error){
    cache.PayoutProviderCancellationPenalty()
    cache.ReturnRemainingCollateral()

    // Decrease the Load on the Provider
    cache.GetProvider().AgreementLoadDecrease(cache.GetCapacity())
    cache.GetProvider().Commit()

    // Destroy the Agreement
    cache.K.RemoveAgreementExpirationIndex(cache.Ctx, cache.GetEndBlock(), cache.GetAgreementId())
    cache.K.RemoveAgreement(cache.Ctx, cache.GetAgreement())

    return  nil

}


func (cache *AgreementCache) Expire() (error){
    cache.PayoutVoidedProviderCancellationPenalty()

    // Decrease the Load on the Provider
    cache.GetProvider().AgreementLoadDecrease(cache.GetCapacity())

    // Destroy the Allocation
    cache.K.DestroyAllocation(cache.Ctx, cache.GetAllocationId())

    // Destroy the Agreement
    cache.K.RemoveAgreementExpirationIndex(cache.Ctx, cache.GetEndBlock(), cache.GetAgreementId())
    cache.K.RemoveAgreement(cache.Ctx, cache.GetAgreement())

    return  nil

}

/* Setters - SET DOES NOT COMMIT()
 */

func (cache *AgreementCache) ResetStartBlock() {
    uctx := sdk.UnwrapSDKContext(cache.Ctx)
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


    cache.Changed()
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

    cache.Changed()
}

func (cache *AgreementCache) CapacityIncrease(amount uint64) (error){
    if cache.GetProvider().GetSubstation().GetAvailableCapacity() < amount {
        return sdkerrors.Wrapf(types.ErrGridMalfunction, "Substation (%s) cannot afford the increase", cache.GetProvider().GetSubstationId())
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
    allocation, allocationFound := cache.K.GetAllocation(cache.Ctx, cache.GetAllocationId())
    if allocationFound {
        // TODO error handling
        cache.K.SetAllocation(cache.Ctx, allocation, cache.GetCapacity())
    }

    cache.Changed()

    return nil
}


func (cache *AgreementCache) CapacityDecrease(amount uint64) (error){
    cache.PayoutVoidedProviderCancellationPenalty()

    // new duration length
        // remaining duration = end block - current block
        // new duration = (remaining duration * old capacity) / new capacity .Truncate()
    if cache.GetCapacity() < amount {
        return sdkerrors.Wrapf(types.ErrGridMalfunction, "Cannot decrease passed zero")
    }
    newCapacity := cache.GetCapacity() - amount

    newDuration := (cache.GetDurationRemaining() * cache.GetCapacity()) / newCapacity

    cache.SetStartBlock(cache.GetCurrentBlock() )
    cache.SetEndBlock(cache.GetStartBlock() + newDuration)

    // Provider Load Increase
    cache.GetProvider().AgreementLoadDecrease(amount)

    cache.Agreement.Capacity = cache.GetCapacity() - amount

    // Decrease the Allocation
    allocation, allocationFound := cache.K.GetAllocation(cache.Ctx, cache.GetAllocationId())
    if allocationFound {
        // TODO error handling
        cache.K.SetAllocation(cache.Ctx, allocation, cache.GetCapacity())
    }

    cache.Changed()

    return nil
}


func (cache *AgreementCache) DurationIncrease(amount uint64) (error){

    newDuration := (cache.GetEndBlock() - cache.GetStartBlock()) + amount
    verifyError := cache.GetProvider().AgreementVerify(newDuration, cache.GetCapacity())
    if verifyError != nil {
        return verifyError
    }

    cache.SetEndBlock(cache.GetEndBlock() + amount)
    cache.Changed()

    return nil
}
