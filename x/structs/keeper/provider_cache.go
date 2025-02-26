package keeper

import (
	"context"

	"structs/x/structs/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "cosmossdk.io/errors"

	// Used in Randomness Orb

	"fmt"
	"cosmossdk.io/math"
)

type ProviderCache struct {
	ProviderId string
	K          *Keeper
	Ctx        context.Context

	AnyChange bool

	Ready bool

	ProviderLoaded  bool
	ProviderChanged bool
	Provider        types.Provider

	OwnerLoaded bool
	Owner       *PlayerCache

	CheckpointBlockAttributeId   string
	CheckpointBlock              uint64
	CheckpointBlockLoaded        bool
	CheckpointBlockChanged       bool

	AgreementLoadAttributeId    string
	AgreementLoad               uint64
	AgreementLoadLoaded         bool
	AgreementLoadChanged        bool

	// AgreementCache[]

}

// Build this initial Provider Cache object
func (k *Keeper) GetProviderCacheFromId(ctx context.Context, providerId string) ProviderCache {
	return ProviderCache{
		ProviderId: providerId,
		K:          k,
		Ctx:        ctx,

		AnyChange: false,

		OwnerLoaded: false,

		ProviderLoaded:  false,
		ProviderChanged: false,

		CheckpointBlockAttributeId: GetGridAttributeIDByObjectId(types.GridAttributeType_checkpointBlock, providerId),

		AgreementLoadAttributeId: GetGridAttributeIDByObjectId(types.GridAttributeType_load, providerId),
	}
}

func (cache *ProviderCache) Commit() {
	cache.AnyChange = false

	fmt.Printf("\n Updating Provider From Cache (%s) \n", cache.ProviderId)

	if cache.ProviderChanged {
		cache.K.SetProvider(cache.Ctx, cache.Provider)
		cache.ProviderChanged = false
	}

	// TODO Add substation

	if cache.Owner != nil && cache.GetOwner().IsChanged() {
		cache.GetOwner().Commit()
	}

    if (cache.CheckpointBlockChanged) {
        cache.K.SetGridAttribute(cache.Ctx, cache.CheckpointBlockAttributeId, cache.CheckpointBlock)
        cache.CheckpointBlockChanged = false
    }

    if (cache.AgreementLoadChanged) {
        cache.K.SetGridAttribute(cache.Ctx, cache.AgreementLoadAttributeId, cache.AgreementLoad)
        cache.AgreementLoadChanged = false
    }

}

func (cache *ProviderCache) IsChanged() bool {
	return cache.AnyChange
}

func (cache *ProviderCache) Changed() {
	cache.AnyChange = true
}

/* Separate Loading functions for each of the underlying containers */

// Load the Player data
func (cache *ProviderCache) LoadOwner() bool {
	newOwner, _ := cache.K.GetPlayerCacheFromId(cache.Ctx, cache.GetOwnerId())
	cache.Owner = &newOwner
	cache.OwnerLoaded = true
	return cache.OwnerLoaded
}

func (cache *ProviderCache) ManualLoadOwner(owner *PlayerCache) {
    cache.Owner = owner
    cache.OwnerLoaded = true
}

// Load the Provider record
func (cache *ProviderCache) LoadProvider() {
	provider, providerFound := cache.K.GetProvider(cache.Ctx, cache.ProviderId)

	if providerFound {
		cache.Provider = provider
		cache.ProviderLoaded = true
	}
}



func (cache *ProviderCache) LoadCheckpointBlock() {
    cache.CheckpointBlock = cache.K.GetGridAttribute(cache.Ctx, cache.CheckpointBlockAttributeId)
    cache.CheckpointBlockLoaded = true
}

func (cache *ProviderCache) LoadAgreementLoad() {
    cache.AgreementLoad = cache.K.GetGridAttribute(cache.Ctx, cache.AgreementLoadAttributeId)
    cache.AgreementLoadLoaded = true
}



/* Getters
 * These will always perform a Load first on the appropriate data if it hasn't occurred yet.
 */


func (cache *ProviderCache) GetProvider() types.Provider { if !cache.ProviderLoaded { cache.LoadProvider() }; return cache.Provider }
func (cache *ProviderCache) GetProviderId() string { return cache.ProviderId }


// Get the Owner data
func (cache *ProviderCache) GetOwnerId() string { if !cache.ProviderLoaded { cache.LoadProvider() }; return cache.Provider.Owner }
func (cache *ProviderCache) GetOwner() *PlayerCache { if !cache.OwnerLoaded { cache.LoadOwner() }; return cache.Owner }

func (cache *ProviderCache) GetSubstationId() string { if !cache.ProviderLoaded { cache.LoadProvider() }; return cache.Provider.SubstationId }
// TODO func (cache *ProviderCache) GetSubstation() *SubstationCache {}

func (cache *ProviderCache) GetRate() sdk.Coin { if !cache.ProviderLoaded { cache.LoadProvider() }; return cache.Provider.Rate }

func (cache *ProviderCache) GetAccessPolicy() types.ProviderAccessPolicy { if !cache.ProviderLoaded { cache.LoadProvider() }; return cache.Provider.AccessPolicy }

func (cache *ProviderCache) GetCapacityMinimum() uint64 { if !cache.ProviderLoaded { cache.LoadProvider() }; return cache.Provider.CapacityMinimum }
func (cache *ProviderCache) GetCapacityMaximum() uint64 { if !cache.ProviderLoaded { cache.LoadProvider() }; return cache.Provider.CapacityMaximum }
func (cache *ProviderCache) GetDurationMinimum() uint64 { if !cache.ProviderLoaded { cache.LoadProvider() }; return cache.Provider.DurationMinimum }
func (cache *ProviderCache) GetDurationMaximum() uint64 { if !cache.ProviderLoaded { cache.LoadProvider() }; return cache.Provider.DurationMaximum }

func (cache *ProviderCache) GetProviderCancellationPenalty() math.LegacyDec { if !cache.ProviderLoaded { cache.LoadProvider() }; return cache.Provider.ProviderCancellationPenalty }
func (cache *ProviderCache) GetConsumerCancellationPenalty() math.LegacyDec { if !cache.ProviderLoaded { cache.LoadProvider() }; return cache.Provider.ConsumerCancellationPenalty }

func (cache *ProviderCache) GetCreator() string { if !cache.ProviderLoaded { cache.LoadProvider() }; return cache.Provider.Creator }

func (cache *ProviderCache) GetAgreementLoad() uint64 { if !cache.AgreementLoadLoaded { cache.LoadAgreementLoad() }; return cache.AgreementLoad }
func (cache *ProviderCache) GetCheckpointBlock() uint64 { if !cache.CheckpointBlockLoaded { cache.LoadCheckpointBlock() }; return cache.CheckpointBlock }

func (cache *ProviderCache) GetCollateralPoolLocation() string { return types.ProviderCollateralPool + cache.GetProviderId() }
func (cache *ProviderCache) GetEarningsPoolLocation() string { return types.ProviderEarningsPool + cache.GetProviderId() }

func (cache *ProviderCache) AgreementVerify(capacity uint64, duration uint64) (error) {
    // min < capacity < max
    if cache.GetCapacityMinimum() > capacity {
        return sdkerrors.Wrapf(types.ErrInvalidParameters, "Capacity (%d) cannot be lower than Minimum Capacity (%d)", capacity, cache.GetCapacityMinimum())
    }
    if capacity > cache.GetCapacityMaximum() {
        return sdkerrors.Wrapf(types.ErrInvalidParameters, "Capacity (%d) cannot be greater than Maximum Capacity (%d)", capacity, cache.GetCapacityMaximum())
    }

    // min < duration < max
    if cache.GetDurationMinimum() > duration {
        return sdkerrors.Wrapf(types.ErrInvalidParameters, "Duration (%d) cannot be lower than Minimum Duration (%d)", duration, cache.GetDurationMinimum())
    }
    if duration > cache.GetDurationMaximum() {
        return sdkerrors.Wrapf(types.ErrInvalidParameters, "Duration (%d) cannot be greater than Maximum Duration (%d)", duration, cache.GetDurationMaximum())
    }

    // Can the Substation support the added capacity
    substation := cache.K.GetSubstationCacheFromId(cache.Ctx, cache.GetSubstationId())
    if capacity > substation.GetAvailableCapacity(){
        return sdkerrors.Wrapf(types.ErrInvalidParameters, "Desired Capacity (%d) is beyond what the Substation (%s) can support (%d) for this Provider (%s)", capacity, substation.GetSubstationId(), substation.GetAvailableCapacity(), cache.GetProviderId())
    }

    return nil

}


/* Permissions */


// Update Permission
func (cache *ProviderCache) CanUpdate(activePlayer *PlayerCache) (error) {
    return cache.PermissionCheck(types.PermissionUpdate, activePlayer)
}

// Assets Permission
func (cache *ProviderCache) CanWithdrawBalance(activePlayer *PlayerCache) (error) {
    return cache.PermissionCheck(types.PermissionAssets, activePlayer)
}


func (cache *ProviderCache) PermissionCheck(permission types.Permission, activePlayer *PlayerCache) (error) {
    // Make sure the address calling this has permissions
    if (!cache.K.PermissionHasOneOf(cache.Ctx, GetAddressPermissionIDBytes(activePlayer.GetActiveAddress()), permission)) {
        return sdkerrors.Wrapf(types.ErrPermission, "Calling address (%s) has no (%d) permissions ", activePlayer.GetActiveAddress(), permission)
    }

    if !activePlayer.HasPlayerAccount() {
        return sdkerrors.Wrapf(types.ErrPermission, "Calling address (%s) has no Account", activePlayer.GetActiveAddress())
    } else {
        if (activePlayer.GetPlayerId() != cache.GetOwnerId()) {
            if (!cache.K.PermissionHasOneOf(cache.Ctx, GetObjectPermissionIDBytes(cache.GetProviderId(), activePlayer.GetPlayerId()), permission)) {
               return sdkerrors.Wrapf(types.ErrPermission, "Calling account (%s) has no (%d) permissions on target substation (%s)", activePlayer.GetPlayerId(), permission, cache.GetProviderId())
            }
        }
    }
    return nil
}

func (cache *ProviderCache) CanOpenAgreement(activePlayer *PlayerCache) (error) {

    if cache.GetAccessPolicy() == types.ProviderAccessPolicy_openMarket {
        if !activePlayer.HasPlayerAccount() {
            return sdkerrors.Wrapf(types.ErrPermission, "Calling address (%s) has no Account", activePlayer.GetActiveAddress())
        }

    } else if cache.GetAccessPolicy() == types.ProviderAccessPolicy_guildMarket {
        if !cache.K.ProviderGuildAccessAllowed(cache.Ctx, cache.GetProviderId(), activePlayer.GetGuildId()) {
            return sdkerrors.Wrapf(types.ErrPermission, "Calling account (%s) is not a member of an approved guild (%s)", activePlayer.GetPlayerId(), activePlayer.GetGuildId())
        }

    } else if cache.GetAccessPolicy() == types.ProviderAccessPolicy_closedMarket {
        return sdkerrors.Wrapf(types.ErrPermission, "Provider (%s) is not accepting new Agreements", cache.GetProviderId())

    } else {
            return sdkerrors.Wrapf(types.ErrPermission, "We're not really sure why it's not allowed, but it isn't. Pls tell an adult")
    }

    return nil
}

/* Committing Setters */

func (cache *ProviderCache) WithdrawBalanceAndCommit(destinationAddress string) (error) {

    destinationAcc, errParam := sdk.AccAddressFromBech32(destinationAddress)
    if errParam != nil {
        return errParam
    }

    // First handle the balances available via checkpoint
    uctx := sdk.UnwrapSDKContext(cache.Ctx)
    currentBlock := uint64(uctx.BlockHeight())
    blockDifference := currentBlock - cache.GetCheckpointBlock()

    blocks := math.LegacyNewDecFromInt(math.NewIntFromUint64(blockDifference))
    rate := math.LegacyNewDecFromInt(cache.GetRate().Amount)
    load := math.LegacyNewDecFromInt(math.NewIntFromUint64(cache.GetAgreementLoad()))

    prePenaltyDeductionAmount := blocks.Mul(rate).Mul(load)
    penaltyDeductionAmount := prePenaltyDeductionAmount.Mul(cache.GetProviderCancellationPenalty())

    finalWithdrawBalance := prePenaltyDeductionAmount.Sub(penaltyDeductionAmount).TruncateInt()

    withdrawAmountCoin := sdk.NewCoins(sdk.NewCoin(cache.GetRate().Denom, finalWithdrawBalance))

    errSend := cache.K.bankKeeper.SendCoinsFromModuleToAccount(cache.Ctx, cache.GetCollateralPoolLocation(), destinationAcc, withdrawAmountCoin)
    if errSend != nil {
        return errSend
    }

    cache.SetCheckpointBlock(currentBlock)

    // Now handle the value available in the Earnings pool
    // Get Balance
    earningsBalances := cache.K.bankKeeper.SpendableCoins(cache.Ctx, cache.K.accountKeeper.GetModuleAddress(cache.GetEarningsPoolLocation()))
    // Transfer
    errSend = cache.K.bankKeeper.SendCoinsFromModuleToAccount(cache.Ctx, cache.GetEarningsPoolLocation(), destinationAcc, earningsBalances)
    if errSend != nil {
        return errSend
    }

    cache.Commit()

    return nil
}



func (cache *ProviderCache) GrantGuildsAndCommit(guildIdSet []string) (error) {
    for _, guildId := range guildIdSet {
        _, found := cache.K.GetGuild(cache.Ctx, guildId)
        if !found {
            return sdkerrors.Wrapf(types.ErrObjectNotFound, "Guild ID (%s) not found ", guildId)
        }
        cache.K.ProviderGrantGuild(cache.Ctx, cache.GetProviderId(), guildId)
    }
    return nil
}

func (cache *ProviderCache) RevokeGuildsAndCommit(guildIdSet []string) (error) {
    for _, guildId := range guildIdSet {
        cache.K.ProviderRevokeGuild(cache.Ctx, cache.GetProviderId(), guildId)
    }
    return nil
}

/* Setters - SET DOES NOT COMMIT()
 */


func (cache *ProviderCache) ResetCheckpointBlock() {
    uctx := sdk.UnwrapSDKContext(cache.Ctx)
    cache.CheckpointBlock = uint64(uctx.BlockHeight())
    cache.CheckpointBlockLoaded = true
    cache.CheckpointBlockChanged = true
    cache.Changed()
}

func (cache *ProviderCache) SetCheckpointBlock(block uint64) {
    cache.CheckpointBlock = block
    cache.CheckpointBlockLoaded = true
    cache.CheckpointBlockChanged = true
    cache.Changed()
}

func (cache *ProviderCache) AgreementLoadIncrease(amount uint64) {
    cache.AgreementLoad = cache.GetAgreementLoad() + amount
    cache.AgreementLoadChanged = true
}

func (cache *ProviderCache) AgreementLoadDecrease(amount uint64) {
    if amount > cache.GetAgreementLoad() {
        cache.AgreementLoad = 0
    } else {
        cache.AgreementLoad = cache.GetAgreementLoad() - amount
    }
    cache.AgreementLoadChanged = true
}

func (cache *ProviderCache) SetAccessPolicy(accessPolicy types.ProviderAccessPolicy) {
    cache.Provider.SetAccessPolicy(accessPolicy)
    cache.Changed()
}

func (cache *ProviderCache) SetCapacityMaximum(maximum uint64) (error){
    paramError := cache.Provider.SetCapacityMaximum(maximum)
    if paramError != nil {
        cache.Changed()
    }
    return paramError
}

func (cache *ProviderCache) SetCapacityMinimum(minimum uint64) (error){
    paramError := cache.Provider.SetCapacityMinimum(minimum)
    if paramError != nil {
        cache.Changed()
    }
    return paramError
}

func (cache *ProviderCache) SetDurationMaximum(maximum uint64) (error){
    paramError := cache.Provider.SetDurationMaximum(maximum)
    if paramError != nil {
        cache.Changed()
    }
    return paramError
}

func (cache *ProviderCache) SetDurationMinimum(minimum uint64) (error){
    paramError := cache.Provider.SetDurationMinimum(minimum)
    if paramError != nil {
        cache.Changed()
    }
    return paramError
}


func (cache *ProviderCache) CheckPoint() (error) {

    // First handle the balances available via checkpoint
    uctx := sdk.UnwrapSDKContext(cache.Ctx)
    currentBlock := uint64(uctx.BlockHeight())
    blockDifference := currentBlock - cache.GetCheckpointBlock()

    blocks := math.LegacyNewDecFromInt(math.NewIntFromUint64(blockDifference))
    rate := math.LegacyNewDecFromInt(cache.GetRate().Amount)
    load := math.LegacyNewDecFromInt(math.NewIntFromUint64(cache.GetAgreementLoad()))

    prePenaltyDeductionAmount := blocks.Mul(rate).Mul(load)
    penaltyDeductionAmount := prePenaltyDeductionAmount.Mul(cache.GetProviderCancellationPenalty())

    finalWithdrawBalance := prePenaltyDeductionAmount.Sub(penaltyDeductionAmount).TruncateInt()

    withdrawAmountCoin := sdk.NewCoins(sdk.NewCoin(cache.GetRate().Denom, finalWithdrawBalance))

    errSend := cache.K.bankKeeper.SendCoinsFromModuleToModule(cache.Ctx, cache.GetCollateralPoolLocation(), cache.GetEarningsPoolLocation(), withdrawAmountCoin)
    if errSend != nil {
        return errSend
    }

    cache.SetCheckpointBlock(currentBlock)

    return nil
}
