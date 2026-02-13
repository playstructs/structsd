package keeper

import (


	"structs/x/structs/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"cosmossdk.io/math"
    authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
)

type ProviderCache struct {
	ProviderId string
	CC         *CurrentContext

	Changed bool
	Deleted bool
	Ready bool

	ProviderLoaded  bool
	Provider        types.Provider

	CheckpointBlockAttributeId  string
	AgreementLoadAttributeId    string
}

func (cache *ProviderCache) Commit() {
	if cache.Changed {
        cache.CC.k.logger.Info("Updating Provider From Cache", "providerId", cache.ProviderId)
        if cache.Deleted {
            cache.CC.k.RemoveProvider(cache.CC.ctx, cache.ProviderId)
        } else {
            cache.CC.k.SetProvider(cache.CC.ctx, cache.Provider)
        }
	}
    cache.Changed = false
}

func (cache *ProviderCache) IsChanged() bool {
	return cache.Changed
}

func (cache *ProviderCache) ID() string {
	return cache.ProviderId
}



/* Separate Loading functions for each of the underlying containers */


// Load the Provider record
func (cache *ProviderCache) LoadProvider() (bool) {
	cache.Provider, cache.ProviderLoaded = cache.CC.k.GetProvider(cache.CC.ctx, cache.ProviderId)
    return cache.ProviderLoaded
}


/* Getters
 * These will always perform a Load first on the appropriate data if it hasn't occurred yet.
 */


func (cache *ProviderCache) GetProvider() types.Provider { if !cache.ProviderLoaded { cache.LoadProvider() }; return cache.Provider }
func (cache *ProviderCache) GetProviderId() string { return cache.ProviderId }

func (cache *ProviderCache) GetOwner() *PlayerCache {
    player, _ := cache.CC.GetPlayer(cache.GetOwnerId())
    return player
}

func (cache *ProviderCache) GetSubstation() *SubstationCache {
    return cache.CC.GetSubstation(cache.GetSubstationId())
}

func (cache *ProviderCache) GetOwnerId() string { if !cache.ProviderLoaded { cache.LoadProvider() }; return cache.Provider.Owner }
func (cache *ProviderCache) GetSubstationId() string { if !cache.ProviderLoaded { cache.LoadProvider() }; return cache.Provider.SubstationId }

func (cache *ProviderCache) GetRate() sdk.Coin { if !cache.ProviderLoaded { cache.LoadProvider() }; return cache.Provider.Rate }

func (cache *ProviderCache) GetAccessPolicy() types.ProviderAccessPolicy { if !cache.ProviderLoaded { cache.LoadProvider() }; return cache.Provider.AccessPolicy }

func (cache *ProviderCache) GetCapacityMinimum() uint64 { if !cache.ProviderLoaded { cache.LoadProvider() }; return cache.Provider.CapacityMinimum }
func (cache *ProviderCache) GetCapacityMaximum() uint64 { if !cache.ProviderLoaded { cache.LoadProvider() }; return cache.Provider.CapacityMaximum }
func (cache *ProviderCache) GetDurationMinimum() uint64 { if !cache.ProviderLoaded { cache.LoadProvider() }; return cache.Provider.DurationMinimum }
func (cache *ProviderCache) GetDurationMaximum() uint64 { if !cache.ProviderLoaded { cache.LoadProvider() }; return cache.Provider.DurationMaximum }

func (cache *ProviderCache) GetProviderCancellationPenalty() math.LegacyDec { if !cache.ProviderLoaded { cache.LoadProvider() }; return cache.Provider.ProviderCancellationPenalty }
func (cache *ProviderCache) GetConsumerCancellationPenalty() math.LegacyDec { if !cache.ProviderLoaded { cache.LoadProvider() }; return cache.Provider.ConsumerCancellationPenalty }

func (cache *ProviderCache) GetCreator() string { if !cache.ProviderLoaded { cache.LoadProvider() }; return cache.Provider.Creator }

func (cache *ProviderCache) GetAgreementLoad() uint64 {
    return cache.CC.GetGridAttribute(cache.AgreementLoadAttributeId)
}

func (cache *ProviderCache) GetCheckpointBlock() uint64 {
    return cache.CC.GetGridAttribute(cache.CheckpointBlockAttributeId)
}

func (cache *ProviderCache) GetCollateralPoolLocation() sdk.AccAddress { return authtypes.NewModuleAddress(types.ProviderCollateralPool + cache.GetProviderId()) }
func (cache *ProviderCache) GetEarningsPoolLocation() sdk.AccAddress { return authtypes.NewModuleAddress(types.ProviderEarningsPool + cache.GetProviderId()) }

func (cache *ProviderCache) AgreementVerify(capacity uint64, duration uint64) (error) {
    // min < capacity < max
    if cache.GetCapacityMinimum() > capacity {
        return types.NewParameterValidationError("capacity", capacity, "below_minimum").WithRange(cache.GetCapacityMinimum(), cache.GetCapacityMaximum())
    }
    if capacity > cache.GetCapacityMaximum() {
        return types.NewParameterValidationError("capacity", capacity, "above_maximum").WithRange(cache.GetCapacityMinimum(), cache.GetCapacityMaximum())
    }

    // min < duration < max
    if cache.GetDurationMinimum() > duration {
        return types.NewParameterValidationError("duration", duration, "below_minimum").WithRange(cache.GetDurationMinimum(), cache.GetDurationMaximum())
    }
    if duration > cache.GetDurationMaximum() {
        return types.NewParameterValidationError("duration", duration, "above_maximum").WithRange(cache.GetDurationMinimum(), cache.GetDurationMaximum())
    }

    // Can the Substation support the added capacity
    substation := cache.CC.GetSubstation(cache.GetSubstationId())
    if capacity > substation.GetAvailableCapacity(){
        return types.NewParameterValidationError("capacity", capacity, "exceeds_available").WithSubstation(substation.GetSubstationId()).WithRange(0, substation.GetAvailableCapacity())
    }

    return nil

}


/* Permissions */

// Delete Permission
func (cache *ProviderCache) CanDelete(activePlayer *PlayerCache) (error) {
    return cache.PermissionCheck(types.PermissionDelete, activePlayer)
}

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
    if (!cache.CC.PermissionHasOneOf(GetAddressPermissionIDBytes(activePlayer.GetActiveAddress()), permission)) {
        return types.NewPermissionError("address", activePlayer.GetActiveAddress(), "", "", uint64(permission), "provider_action")
    }

    if !activePlayer.HasPlayerAccount() {
        return types.NewPlayerRequiredError(activePlayer.GetActiveAddress(), "provider_action")
    } else {
        if (activePlayer.GetPlayerId() != cache.GetOwnerId()) {
            if (!cache.CC.PermissionHasOneOf(GetObjectPermissionIDBytes(cache.GetProviderId(), activePlayer.GetPlayerId()), permission)) {
               return types.NewPermissionError("player", activePlayer.GetPlayerId(), "provider", cache.GetProviderId(), uint64(permission), "provider_action")
            }
        }
    }
    return nil
}

func (cache *ProviderCache) CanOpenAgreement(activePlayer *PlayerCache) (error) {

    if cache.GetAccessPolicy() == types.ProviderAccessPolicy_openMarket {
        if !activePlayer.HasPlayerAccount() {
            return types.NewPlayerRequiredError(activePlayer.GetActiveAddress(), "agreement_open")
        }

    } else if cache.GetAccessPolicy() == types.ProviderAccessPolicy_guildMarket {
        if !cache.CC.k.ProviderGuildAccessAllowed(cache.CC.ctx, cache.GetProviderId(), activePlayer.GetGuildId()) {
            return types.NewProviderAccessError(cache.GetProviderId(), "guild_not_allowed").WithGuild(activePlayer.GetGuildId()).WithPlayer(activePlayer.GetPlayerId())
        }

    } else if cache.GetAccessPolicy() == types.ProviderAccessPolicy_closedMarket {
        return types.NewProviderAccessError(cache.GetProviderId(), "closed_market").WithPlayer(activePlayer.GetPlayerId())

    } else {
        return types.NewProviderAccessError(cache.GetProviderId(), "unknown").WithPlayer(activePlayer.GetPlayerId())
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
    uctx := sdk.UnwrapSDKContext(cache.CC.ctx)
    currentBlock := uint64(uctx.BlockHeight())
    blockDifference := currentBlock - cache.GetCheckpointBlock()

    blocks := math.LegacyNewDecFromInt(math.NewIntFromUint64(blockDifference))
    rate := math.LegacyNewDecFromInt(cache.GetRate().Amount)
    load := math.LegacyNewDecFromInt(math.NewIntFromUint64(cache.GetAgreementLoad()))

    prePenaltyDeductionAmount := blocks.Mul(rate).Mul(load)
    penaltyDeductionAmount := prePenaltyDeductionAmount.Mul(cache.GetProviderCancellationPenalty())

    finalWithdrawBalance := prePenaltyDeductionAmount.Sub(penaltyDeductionAmount).TruncateInt()

    withdrawAmountCoin := sdk.NewCoins(sdk.NewCoin(cache.GetRate().Denom, finalWithdrawBalance))

    errSend := cache.CC.k.bankKeeper.SendCoins(cache.CC.ctx, cache.GetCollateralPoolLocation(), destinationAcc, withdrawAmountCoin)
    if errSend != nil {
        return errSend
    }

    cache.SetCheckpointBlock(currentBlock)

    // Now handle the value available in the Earnings pool
    // Get Balance
    earningsBalances := cache.CC.k.bankKeeper.SpendableCoins(cache.CC.ctx, cache.GetEarningsPoolLocation())
    // Transfer
    errSend = cache.CC.k.bankKeeper.SendCoins(cache.CC.ctx, cache.GetEarningsPoolLocation(), destinationAcc, earningsBalances)
    if errSend != nil {
        return errSend
    }

    cache.Commit()

    return nil
}



func (cache *ProviderCache) GrantGuildsAndCommit(guildIdSet []string) (error) {
    for _, guildId := range guildIdSet {
        guild := cache.CC.GetGuild(guildId)
        if !guild.LoadGuild() {
            return types.NewObjectNotFoundError("guild", guildId)
        }
        cache.CC.k.ProviderGrantGuild(cache.CC.ctx, cache.GetProviderId(), guildId)
    }
    return nil
}

func (cache *ProviderCache) RevokeGuildsAndCommit(guildIdSet []string) (error) {
    for _, guildId := range guildIdSet {
        cache.CC.k.ProviderRevokeGuild(cache.CC.ctx, cache.GetProviderId(), guildId)
    }
    return nil
}

func (cache *ProviderCache) Delete() (error) {

    // Get List of Agreements
    agreements := cache.CC.k.GetAllAgreementIdByProviderIndex(cache.CC.ctx, cache.GetProviderId())
    for _, agreementId := range agreements {
        agreement := cache.CC.GetAgreement(agreementId)
        agreement.PrematureCloseByProvider()
    }

    cache.CC.ClearGridAttribute(cache.CheckpointBlockAttributeId)
    cache.CC.ClearGridAttribute(cache.AgreementLoadAttributeId)

    cache.Deleted = true
    return nil
}


func (cache *ProviderCache) ResetCheckpointBlock() {
    uctx := sdk.UnwrapSDKContext(cache.CC.ctx)
    cache.CC.SetGridAttribute(cache.CheckpointBlockAttributeId, uint64(uctx.BlockHeight()))
}

func (cache *ProviderCache) SetCheckpointBlock(block uint64) {
    cache.CC.SetGridAttribute(cache.CheckpointBlockAttributeId, block)
}

func (cache *ProviderCache) AgreementLoadIncrease(amount uint64) {
    cache.CC.SetGridAttributeIncrement(cache.AgreementLoadAttributeId, amount)
}

func (cache *ProviderCache) AgreementLoadDecrease(amount uint64) {
    cache.CC.SetGridAttributeDecrement(cache.AgreementLoadAttributeId, amount)
}

func (cache *ProviderCache) SetAccessPolicy(accessPolicy types.ProviderAccessPolicy) {
    cache.Provider.SetAccessPolicy(accessPolicy)
    cache.Changed = true
}

func (cache *ProviderCache) SetCapacityMaximum(maximum uint64) (error){
    paramError := cache.Provider.SetCapacityMaximum(maximum)
    if paramError != nil {
        cache.Changed = true
    }
    return paramError
}

func (cache *ProviderCache) SetCapacityMinimum(minimum uint64) (error){
    paramError := cache.Provider.SetCapacityMinimum(minimum)
    if paramError != nil {
        cache.Changed = true
    }
    return paramError
}

func (cache *ProviderCache) SetDurationMaximum(maximum uint64) (error){
    paramError := cache.Provider.SetDurationMaximum(maximum)
    if paramError != nil {
        cache.Changed = true
    }
    return paramError
}

func (cache *ProviderCache) SetDurationMinimum(minimum uint64) (error){
    paramError := cache.Provider.SetDurationMinimum(minimum)
    if paramError != nil {
        cache.Changed = true
    }
    return paramError
}


func (cache *ProviderCache) Checkpoint() (error) {

    // First handle the balances available via checkpoint
    uctx := sdk.UnwrapSDKContext(cache.CC.ctx)
    currentBlock := uint64(uctx.BlockHeight())
    blockDifference := currentBlock - cache.GetCheckpointBlock()

    blocks := math.LegacyNewDecFromInt(math.NewIntFromUint64(blockDifference))
    rate := math.LegacyNewDecFromInt(cache.GetRate().Amount)
    load := math.LegacyNewDecFromInt(math.NewIntFromUint64(cache.GetAgreementLoad()))

    prePenaltyDeductionAmount := blocks.Mul(rate).Mul(load)
    penaltyDeductionAmount := prePenaltyDeductionAmount.Mul(cache.GetProviderCancellationPenalty())

    checkpointBalance := prePenaltyDeductionAmount.Sub(penaltyDeductionAmount).TruncateInt()

    checkpointBalanceCoin := sdk.NewCoins(sdk.NewCoin(cache.GetRate().Denom, checkpointBalance))

    errSend := cache.CC.k.bankKeeper.SendCoins(cache.CC.ctx, cache.GetCollateralPoolLocation(), cache.GetEarningsPoolLocation(), checkpointBalanceCoin)
    if errSend != nil {
        return errSend
    }

    cache.SetCheckpointBlock(currentBlock)

    return nil
}
