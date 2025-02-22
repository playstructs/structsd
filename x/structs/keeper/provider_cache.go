package keeper

import (
	"context"

	"structs/x/structs/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

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

func (cache *ProviderCache) GetSubstationID() string { if !cache.ProviderLoaded { cache.LoadProvider() }; return cache.Provider.SubstationId }
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

/* Setters - SET DOES NOT COMMIT()
 */

func (cache *ProviderCache) ResetCheckpointBlock() {
    uctx := sdk.UnwrapSDKContext(cache.Ctx)
    cache.CheckpointBlock = uint64(uctx.BlockHeight())
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
