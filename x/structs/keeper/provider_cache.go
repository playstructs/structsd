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


// CheckProvider reports whether a provider actually exists at this cache's id.
//
// The other caches spell this the same way. It exists because the context getter
// that produced the cache reads nothing: see CurrentContext.GetExistingProvider.
func (cache *ProviderCache) CheckProvider() error {
	if !cache.ProviderLoaded {
		if !cache.LoadProvider() {
			return types.NewObjectNotFoundError("provider", cache.ProviderId)
		}
	}
	return nil
}


/* Getters
 * These will always perform a Load first on the appropriate data if it hasn't occurred yet.
 */


func (cache *ProviderCache) GetProvider() types.Provider { if !cache.ProviderLoaded { cache.LoadProvider() }; return cache.Provider }
func (cache *ProviderCache) GetProviderId() string { return cache.ProviderId }

func (cache *ProviderCache) GetOwner() *PlayerCache {
    return cache.CC.GetPlayer(cache.GetOwnerId())
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

// GetProviderCollateralPoolLocation is where consumers' agreement collateral is
// held. It is keyed only by provider, so every agreement of that provider shares
// the one account.
func GetProviderCollateralPoolLocation(providerId string) sdk.AccAddress { return authtypes.NewModuleAddress(types.ProviderCollateralPool + providerId) }

// GetProviderEarningsPoolLocation is where a provider's earned revenue lands
// once swept out of the collateral pool.
func GetProviderEarningsPoolLocation(providerId string) sdk.AccAddress { return authtypes.NewModuleAddress(types.ProviderEarningsPool + providerId) }

func (cache *ProviderCache) GetCollateralPoolLocation() sdk.AccAddress { return GetProviderCollateralPoolLocation(cache.GetProviderId()) }
func (cache *ProviderCache) GetEarningsPoolLocation() sdk.AccAddress { return GetProviderEarningsPoolLocation(cache.GetProviderId()) }

// SweepRevenue moves provider revenue out of the collateral pool, clamped to
// what the pool actually holds.
//
// Consumer collateral has first claim on the shared pool; provider revenue is
// subordinate. A shortfall is reported rather than returned because block-hook
// settlement must continue.
func (cache *ProviderCache) SweepRevenue(destination sdk.AccAddress, amount math.Int, agreementId string) {
    if !amount.IsPositive() {
        return
    }

    denom := cache.GetRate().Denom
    available := cache.CC.k.bankKeeper.SpendableCoin(cache.CC.ctx, cache.GetCollateralPoolLocation(), denom).Amount

    paid := amount
    if available.LT(amount) {
        paid = available
    }

    if paid.IsPositive() {
        errSend := cache.CC.k.bankKeeper.SendCoins(cache.CC.ctx, cache.GetCollateralPoolLocation(), destination, sdk.NewCoins(sdk.NewCoin(denom, paid)))
        if errSend != nil {
            cache.CC.k.logger.Error("Provider revenue sweep failed",
                "providerId", cache.GetProviderId(),
                "agreementId", agreementId,
                "amount", paid.String(),
                "denom", denom,
                "error", errSend,
            )
            paid = math.ZeroInt()
        }
    }

    shortfall := amount.Sub(paid)
    if shortfall.IsZero() {
        return
    }

    cache.CC.k.logger.Error("Provider collateral pool could not cover provider revenue",
        "providerId", cache.GetProviderId(),
        "agreementId", agreementId,
        "requested", amount.String(),
        "paid", paid.String(),
        "shortfall", shortfall.String(),
        "denom", denom,
    )

    ctxSDK := sdk.UnwrapSDKContext(cache.CC.ctx)
    _ = ctxSDK.EventManager().EmitTypedEvent(&types.EventProviderRevenueShortfall{
        &types.EventProviderRevenueShortfallDetail{
            ProviderId:  cache.GetProviderId(),
            AgreementId: agreementId,
            Denom:       denom,
            Requested:   amount.String(),
            Paid:        paid.String(),
            Shortfall:   shortfall.String(),
        },
    })
}

func (cache *ProviderCache) AgreementVerify(capacity uint64, duration uint64) (error) {
    if err := cache.AgreementCapacityVerify(capacity); err != nil {
        return err
    }

    if err := cache.AgreementDurationVerify(duration); err != nil {
        return err
    }

    // Can the Substation support the added capacity
    substation := cache.CC.GetSubstation(cache.GetSubstationId())
    if capacity > substation.GetAvailableCapacity(){
        return types.NewParameterValidationError("capacity", capacity, "exceeds_available").WithSubstation(substation.GetSubstationId()).WithRange(0, substation.GetAvailableCapacity())
    }

    return nil

}

// AgreementCapacityVerify holds the published capacity range. Opening an
// agreement and later changing its capacity share this so a modification cannot
// land outside the terms the provider advertised.
func (cache *ProviderCache) AgreementCapacityVerify(capacity uint64) error {
    if cache.GetCapacityMinimum() > capacity {
        return types.NewParameterValidationError("capacity", capacity, "below_minimum").WithRange(cache.GetCapacityMinimum(), cache.GetCapacityMaximum())
    }
    if capacity > cache.GetCapacityMaximum() {
        return types.NewParameterValidationError("capacity", capacity, "above_maximum").WithRange(cache.GetCapacityMinimum(), cache.GetCapacityMaximum())
    }
    return nil
}

func (cache *ProviderCache) AgreementDurationVerify(duration uint64) error {
    if cache.GetDurationMinimum() > duration {
        return types.NewParameterValidationError("duration", duration, "below_minimum").WithRange(cache.GetDurationMinimum(), cache.GetDurationMaximum())
    }
    if duration > cache.GetDurationMaximum() {
        return types.NewParameterValidationError("duration", duration, "above_maximum").WithRange(cache.GetDurationMinimum(), cache.GetDurationMaximum())
    }
    return nil
}


/* Permissions */

// Delete Permission
func (cache *ProviderCache) CanBeDeletedBy(activePlayer *PlayerCache) (error) {
    return cache.CC.PermissionCheck(cache, activePlayer,types.PermDelete)
}

// Update Permission
func (cache *ProviderCache) CanBeUpdatedBy(activePlayer *PlayerCache) (error) {
    return cache.CC.PermissionCheck(cache, activePlayer,types.PermUpdate)
}

// Assets Permission
func (cache *ProviderCache) CanWithdrawBalanceBy(activePlayer *PlayerCache) (error) {
    return cache.CC.PermissionCheck(cache, activePlayer, types.PermProviderWithdraw)
}


func (cache *ProviderCache) CanOpenAgreement(activePlayer *PlayerCache) (error) {

    if cache.GetAccessPolicy() == types.ProviderAccessPolicy_openMarket {
        if !activePlayer.HasPlayerAccount() {
            return types.NewPlayerRequiredError(cache.CC.SignerAddress(), "agreement_open")
        }
    } else if cache.GetAccessPolicy() == types.ProviderAccessPolicy_guildMarket {
        if err := cache.CC.PermissionCheck(cache, activePlayer, types.PermProviderOpen); err != nil {
            return err
        }

    } else if cache.GetAccessPolicy() == types.ProviderAccessPolicy_closedMarket {
        return types.NewProviderAccessError(cache.GetProviderId(), "closed_market").WithPlayer(activePlayer.GetPlayerId())

    } else {
        return types.NewProviderAccessError(cache.GetProviderId(), "unknown").WithPlayer(activePlayer.GetPlayerId())
    }

    // The access policy decides who may contract with this provider. It does not
    // authorize moving the player's money, and PermProviderOpen is an access grant
    // rather than a spend one: AgreementOpen debits the primary address for the
    // collateral, so the signing key needs the same bit PlayerSend requires.
    //
    // Checked against the player rather than the provider, which is what keeps
    // open-market open. The owner shortcut in PermissionCheck satisfies the object
    // layer, leaving this a check of the signing key's own bits.
    return activePlayer.CanTransferTokensBy(activePlayer)
}

/* Committing Setters */

func (cache *ProviderCache) WithdrawBalanceAndCommit(destinationAddress string) (error) {

    // A withdrawal names where the money goes, so the destination is
    // transaction-chosen and gets the same treatment as PlayerSend's recipient.
    destinationAcc, errParam := cache.CC.k.resolveExternalRecipient(destinationAddress)
    if errParam != nil {
        return errParam
    }

    // Sweep everything earned up to now out of the collateral pool and into the
    // earnings pool. Going through Checkpoint keeps the accrual maths and the
    // clamp against consumer collateral in one place.
    if errCheckpoint := cache.Checkpoint(); errCheckpoint != nil {
        return errCheckpoint
    }

    // Now handle the value available in the Earnings pool
    // Get Balance
    earningsBalances := cache.CC.k.bankKeeper.SpendableCoins(cache.CC.ctx, cache.GetEarningsPoolLocation())
    // Transfer
    errSend := cache.CC.k.bankKeeper.SendCoins(cache.CC.ctx, cache.GetEarningsPoolLocation(), destinationAcc, earningsBalances)
    if errSend != nil {
        return errSend
    }

    cache.Commit()

    return nil
}




func (cache *ProviderCache) Delete() (error) {

    // Get List of Agreements
    // Each close checkpoints this provider before touching its own load, so the
    // list is settled one agreement at a time rather than in bulk.
    agreements := cache.CC.k.GetAllAgreementIdByProviderIndex(cache.CC.ctx, cache.GetProviderId())
    for _, agreementId := range agreements {
        agreement := cache.CC.GetAgreement(agreementId)
        if err := agreement.PrematureCloseByProvider(); err != nil {
            return err
        }
    }

    if err := cache.drainPoolsToOwner(); err != nil {
        return err
    }
    cache.CC.k.RemoveProviderPoolAddresses(cache.CC.ctx, cache.GetProviderId())

    cache.CC.ClearGridAttribute(cache.CheckpointBlockAttributeId)
    cache.CC.ClearGridAttribute(cache.AgreementLoadAttributeId)

    cache.CC.ClearPermissionsForObject(cache.ID())

    cache.Deleted = true
    cache.Changed = true
    return nil
}

// drainPoolsToOwner empties the provider's collateral and earnings pools into
// the owner's primary address.
//
// It runs only from Delete, after the loop above has settled every agreement, so
// every consumer has already been made whole and what is left belongs to the
// provider. It has to run there and nowhere else: both pool addresses are
// derived from the provider id, and once the provider record is gone
// WithdrawBalanceAndCommit can no longer load it to reach them, so anything left
// behind is unreachable for good. There is normally something left — every
// Checkpoint and every penalty payout truncates, and the remainder stays in the
// collateral pool.
//
// The destination is resolved before any coins move, because a drain that failed
// half way through would leave the rest stranded with no second attempt. It is
// only demanded when there is something to send, so that an owner record that
// cannot be paid does not make an already empty provider undeletable.
func (cache *ProviderCache) drainPoolsToOwner() error {
    collateralPool := cache.GetCollateralPoolLocation()
    earningsPool := cache.GetEarningsPoolLocation()

    collateral := cache.CC.k.bankKeeper.SpendableCoins(cache.CC.ctx, collateralPool)
    earnings := cache.CC.k.bankKeeper.SpendableCoins(cache.CC.ctx, earningsPool)

    if collateral.IsZero() && earnings.IsZero() {
        return nil
    }

    destination, errParam := sdk.AccAddressFromBech32(cache.GetOwner().GetPrimaryAddress())
    if errParam != nil {
        return errParam
    }

    // Collateral first, so that whatever it still holds leaves with the earnings.
    if !collateral.IsZero() {
        if errSend := cache.CC.k.bankKeeper.SendCoins(cache.CC.ctx, collateralPool, earningsPool, collateral); errSend != nil {
            return errSend
        }
    }

    return cache.CC.k.bankKeeper.SendCoins(cache.CC.ctx, earningsPool, destination, collateral.Add(earnings...))
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

func (cache *ProviderCache) SetAccessPolicy(accessPolicy types.ProviderAccessPolicy) error {
    err := cache.Provider.SetAccessPolicy(accessPolicy)
    if err != nil {
        return err
    }
    cache.Changed = true
    return nil
}

func (cache *ProviderCache) SetCapacityMaximum(maximum uint64) (error){
    paramError := cache.Provider.SetCapacityMaximum(maximum)
    if paramError == nil {
        cache.Changed = true
    }
    return paramError
}

func (cache *ProviderCache) SetCapacityMinimum(minimum uint64) (error){
    paramError := cache.Provider.SetCapacityMinimum(minimum)
    if paramError == nil {
        cache.Changed = true
    }
    return paramError
}

func (cache *ProviderCache) SetDurationMaximum(maximum uint64) (error){
    paramError := cache.Provider.SetDurationMaximum(maximum)
    if paramError == nil {
        cache.Changed = true
    }
    return paramError
}

func (cache *ProviderCache) SetDurationMinimum(minimum uint64) (error){
    paramError := cache.Provider.SetDurationMinimum(minimum)
    if paramError == nil {
        cache.Changed = true
    }
    return paramError
}


// Checkpoint sweeps the revenue the provider has earned since the last
// checkpoint into their earnings pool. It bills the provider's *current*
// agreement load across the whole span since that checkpoint, so it must run
// before any load change or the new load is billed over the old span. Every
// agreement teardown path checkpoints for exactly that reason, and so do
// AgreementOpen and the two capacity handlers.
//
// It cannot currently fail: SweepRevenue clamps and reports shortfalls, then
// the checkpoint advances. The error return is retained for caller stability.
func (cache *ProviderCache) Checkpoint() (error) {

    // First handle the balances available via checkpoint
    uctx := sdk.UnwrapSDKContext(cache.CC.ctx)
    currentBlock := uint64(uctx.BlockHeight())

    checkpointBlock := cache.GetCheckpointBlock()
    if checkpointBlock >= currentBlock {
        return nil
    }
    blockDifference := currentBlock - checkpointBlock

    blocks := math.LegacyNewDecFromInt(math.NewIntFromUint64(blockDifference))
    rate := math.LegacyNewDecFromInt(cache.GetRate().Amount)
    load := math.LegacyNewDecFromInt(math.NewIntFromUint64(cache.GetAgreementLoad()))

    prePenaltyDeductionAmount := blocks.Mul(rate).Mul(load)
    penaltyDeductionAmount := prePenaltyDeductionAmount.Mul(cache.GetProviderCancellationPenalty())

    checkpointBalance := prePenaltyDeductionAmount.Sub(penaltyDeductionAmount).TruncateInt()

    cache.SweepRevenue(cache.GetEarningsPoolLocation(), checkpointBalance, "")

    cache.SetCheckpointBlock(currentBlock)

    return nil
}


func (cache *ProviderCache) CanAllocateAsSourceBy(activePlayer *PlayerCache) error {
    return types.NewAllocationError(cache.ID(), "unacceptable_source")
}