package keeper

import (
	"context"

	"structs/x/structs/types"

	// Used in Randomness Orb

	"fmt"
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
	}
}

func (cache *ProviderCache) Commit() {
	cache.AnyChange = false

	fmt.Printf("\n Updating Provider From Cache (%s) \n", cache.ProviderId)

	if cache.ProviderChanged {
		cache.K.SetProvider(cache.Ctx, cache.Provider)
		cache.ProviderChanged = false
	}

	if cache.Owner != nil && cache.GetOwner().IsChanged() {
		cache.GetOwner().Commit()
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

// Load the Provider record
func (cache *ProviderCache) LoadProvider() {
	provider, providerFound := cache.K.GetProvider(cache.Ctx, cache.ProviderId)

	if providerFound {
		cache.Provider = provider
		cache.ProviderLoaded = true
	}
}

/* Getters
 * These will always perform a Load first on the appropriate data if it hasn't occurred yet.
 */

// Get the Owner ID data
func (cache *ProviderCache) GetOwnerId() string {
	if !cache.ProviderLoaded {
		cache.LoadProvider()
	}
	return cache.Provider.Owner
}

// Get the Owner data
func (cache *ProviderCache) GetOwner() *PlayerCache {
	if !cache.OwnerLoaded {
		cache.LoadOwner()
	}
	return cache.Owner
}

func (cache *ProviderCache) GetProvider() types.Provider {
	if !cache.ProviderLoaded {
		cache.LoadProvider()
	}
	return cache.Provider
}

func (cache *ProviderCache) GetProviderId() string {
	return cache.ProviderId
}

/* Setters - SET DOES NOT COMMIT()
 * These will always perform a Load first on the appropriate data if it hasn't occurred yet.
 */

func (cache *ProviderCache) SetProvider(provider types.Provider) {
	cache.Provider = provider
	cache.ProviderChanged = true
}
