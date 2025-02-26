package keeper

import (
	"context"

    sdk "github.com/cosmos/cosmos-sdk/types"
    sdkerrors "cosmossdk.io/errors"
	"structs/x/structs/types"

	"fmt"
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

	OwnerLoaded bool
	Owner       *PlayerCache

	ProviderLoaded bool
	Provider       *ProviderCache

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
	}
}

func (cache *AgreementCache) Commit() {
	cache.AnyChange = false

	fmt.Printf("\n Updating Agreement From Cache (%s) \n", cache.AgreementId)

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

}

func (cache *AgreementCache) IsChanged() bool {
	return cache.AnyChange
}

func (cache *AgreementCache) Changed() {
	cache.AnyChange = true
}

/* Separate Loading functions for each of the underlying containers */

// Load the Agreement record
func (cache *AgreementCache) LoadAgreement() {
	agreement, agreementFound := cache.K.GetAgreement(cache.Ctx, cache.AgreementId)

	if agreementFound {
		cache.Agreement = agreement
		cache.AgreementLoaded = true
	}
}

// Load the Player data
func (cache *AgreementCache) LoadOwner() bool {
	newOwner, _ := cache.K.GetPlayerCacheFromId(cache.Ctx, cache.GetOwnerId())
	cache.Owner = &newOwner
	cache.OwnerLoaded = true
	return cache.OwnerLoaded
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


// Get the Owner data
func (cache *AgreementCache) GetOwnerId() string { if !cache.AgreementLoaded { cache.LoadAgreement() }; return cache.Agreement.Owner }
func (cache *AgreementCache) GetOwner() *PlayerCache { if !cache.OwnerLoaded { cache.LoadOwner() }; return cache.Owner }

// Get the Provider data
func (cache *AgreementCache) GetProviderId() string { if !cache.AgreementLoaded { cache.LoadAgreement() }; return cache.Agreement.ProviderId }
func (cache *AgreementCache) GetProvider() *ProviderCache { if !cache.ProviderLoaded { cache.LoadProvider() }; return cache.Provider }


func (cache *AgreementCache) GetAllocationId() string { if !cache.AgreementLoaded { cache.LoadAgreement() }; return cache.Agreement.AllocationId }
// TODO func GetAllocation()

func (cache *AgreementCache) GetCapacity() uint64 { if !cache.AgreementLoaded { cache.LoadAgreement() }; return cache.Agreement.Capacity }

func (cache *AgreementCache) GetStartBlock() uint64 { if !cache.AgreementLoaded { cache.LoadAgreement() }; return cache.Agreement.StartBlock }
func (cache *AgreementCache) GetEndBlock() uint64 { if !cache.AgreementLoaded { cache.LoadAgreement() }; return cache.Agreement.EndBlock }

func (cache *AgreementCache) GetCreator() string { if !cache.AgreementLoaded { cache.LoadAgreement() }; return cache.Agreement.Creator }



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
    cache.Changed()
}

func (cache *AgreementCache) SetEndBlock(endBlock uint64) {
    if !cache.AgreementLoaded {
        cache.LoadAgreement()
    }
    cache.Agreement.EndBlock = endBlock
    cache.Changed()
}

func (cache *AgreementCache) CapacityIncrease(amount uint64) (error){
    if cache.GetProvider().GetSubstation().GetAvailableCapacity() < (cache.GetCapacity() + amount){
        return sdkerrors.Wrapf(types.ErrGridMalfunction, "Substation (%s) cannot afford the increase", cache.GetProvider().GetSubstationId())
    }

    // original duration length
    // original capacity
    // new duration length

    // start, end = original duration

    // Provider Payout Consumer Cancellation Penalty
        // start, current block
        // (current - start) * rate * capacity * Penalty

    // Provider Load Increase



    cache.Agreement.Capacity = cache.GetCapacity() + amount
    cache.Changed()

    return nil
}