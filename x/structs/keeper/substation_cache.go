package keeper

import (
	"context"

	//sdk "github.com/cosmos/cosmos-sdk/types"
    sdkerrors "cosmossdk.io/errors"
	"structs/x/structs/types"

    "fmt"

)


type SubstationCache struct {
    SubstationId string
    K *Keeper
    Ctx context.Context

    Ready bool

    AnyChange bool

    SubstationLoaded  bool
    SubstationChanged bool
    Substation  types.Substation

    OwnerLoaded bool
    OwnerChanged bool
    Owner *PlayerCache

    GridLoaded bool
    GridChanged bool
    Grid *GridCache

}

// Build this initial Substation Cache object
// This does no validation on the provided substationId
func (k *Keeper) GetSubstationCacheFromId(ctx context.Context, substationId string) (SubstationCache) {
    return SubstationCache{
        SubstationId: substationId,
        K: k,
        Ctx: ctx,

        AnyChange: false,

    }
}

// Build this initial Substation Cache object
// This does no validation on the provided substationId
func (k *Keeper) InitiateSubstation(ctx context.Context, creatorAddress string, owner *PlayerCache, allocation types.Allocation) (SubstationCache, error) {

    // Append Substation
    owner.LoadPlayer()
    substation, _, _ := k.AppendSubstation(ctx, allocation, owner.Player)

    // Start to put the pieces together
    substationCache := SubstationCache{
                  SubstationId: substation.Id,
                  K: k,
                  Ctx: ctx,

                  AnyChange: true,

                  Substation: substation,
                  SubstationChanged: false,
                  SubstationLoaded: true,

                  Owner: owner,
                  OwnerLoaded: true,
    }

    return substationCache, nil
}


func (cache *SubstationCache) Commit() () {
    cache.AnyChange = false

    fmt.Printf("\n Updating Substation From Cache (%s) \n", cache.SubstationId)

    if (cache.SubstationChanged) {
        cache.K.SetSubstation(cache.Ctx, cache.Substation)
        cache.SubstationChanged = false
    }

    if (cache.Owner != nil && cache.GetOwner().IsChanged()) {
        cache.GetOwner().Commit()
    }

}

func (cache *SubstationCache) IsChanged() bool {
    return cache.AnyChange
}

func (cache *SubstationCache) Changed() {
    cache.AnyChange = true
}

/* Separate Loading functions for each of the underlying containers */

// Load the core Substation data
func (cache *SubstationCache) LoadSubstation() (bool) {
    cache.Substation, cache.SubstationLoaded = cache.K.GetSubstation(cache.Ctx, cache.SubstationId)
    return cache.SubstationLoaded
}

// Load the Player data
func (cache *SubstationCache) LoadOwner() (bool) {
    newOwner, _ := cache.K.GetPlayerCacheFromId(cache.Ctx, cache.GetOwnerId())
    cache.Owner = &newOwner
    cache.OwnerLoaded = true
    return cache.OwnerLoaded
}

// Load the Grid cache object
func (cache *SubstationCache) LoadGrid() (bool) {
    newGrid := cache.K.GetGridCacheFromId(cache.Ctx, cache.GetSubstationId())
    cache.Grid = &newGrid
    cache.GridLoaded = true
    return cache.GridLoaded
}


// Set the Owner data manually
// Useful for loading multiple defenders
func (cache *SubstationCache) ManualLoadOwner(owner *PlayerCache) {
    cache.Owner = owner
    cache.OwnerLoaded = true
}


/* Getters
 * These will always perform a Load first on the appropriate data if it hasn't occurred yet.
 */

func (cache *SubstationCache) GetSubstation()   (types.Substation)  { if (!cache.SubstationLoaded) { cache.LoadSubstation() }; return cache.Substation }
func (cache *SubstationCache) GetSubstationId() (string)            { return cache.SubstationId }

func (cache *SubstationCache) GetOwner()        (*PlayerCache)      { if (!cache.OwnerLoaded) { cache.LoadOwner() }; return cache.Owner }
func (cache *SubstationCache) GetOwnerId()      (string)            { if (!cache.SubstationLoaded) { cache.LoadSubstation() }; return cache.Substation.Owner }

func (cache *SubstationCache) GetGrid()         (*GridCache)        { if (!cache.GridLoaded) { cache.LoadGrid() }; return cache.Grid }

func (cache *SubstationCache) GetAvailableCapacity() (uint64)       { return cache.GetGrid().GetCapacity() - cache.GetGrid().GetLoad() }

/* Setters - SET DOES NOT COMMIT()
 * These will always perform a Load first on the appropriate data if it hasn't occurred yet.
 */

// Set the Owner Id data
func (cache *SubstationCache) SetOwnerId(owner string) {
    if (!cache.SubstationLoaded) { cache.LoadSubstation() }

    cache.Substation.Owner = owner
    cache.SubstationChanged = true
    cache.Changed()

    // Player object might be stale now
    cache.OwnerLoaded = false
}



// Delete Permission
func (cache *SubstationCache) CanBeDeleteDBy(activePlayer *PlayerCache) (error) {
    return cache.PermissionCheck(types.PermissionDelete, activePlayer);
}

// Association Permission
func (cache *SubstationCache) CanManagePlayerConnections(activePlayer *PlayerCache) (error) {
    return cache.PermissionCheck(types.PermissionAssociations, activePlayer)
}

// Grid Permission
func (cache *SubstationCache) CanManageAllocationConnections(activePlayer *PlayerCache) (error) {
    return cache.PermissionCheck(types.PermissionGrid, activePlayer)
}

// Asset Permission
func (cache *SubstationCache) CanCreateAllocations(activePlayer *PlayerCache) (error) {
    return cache.PermissionCheck(types.PermissionAssets, activePlayer)
}

func (cache *SubstationCache) PermissionCheck(permission types.Permission, activePlayer *PlayerCache) (err error) {
    // Make sure the address calling this has Play permissions
    if (!cache.K.PermissionHasOneOf(cache.Ctx, GetAddressPermissionIDBytes(activePlayer.GetActiveAddress()), permission)) {
        err = sdkerrors.Wrapf(types.ErrPermission, "Calling address (%s) has no (%d) permissions ", activePlayer.GetActiveAddress(), permission)

    }

    if !activePlayer.HasPlayerAccount() {
        err = sdkerrors.Wrapf(types.ErrPermission, "Calling address (%s) has no Account", activePlayer.GetActiveAddress())
    } else {
        if (err != nil) {
            if (activePlayer.GetPlayerId() != cache.GetOwnerId()) {
                if (!cache.K.PermissionHasOneOf(cache.Ctx, GetObjectPermissionIDBytes(cache.GetSubstationId(), activePlayer.GetPlayerId()), permission)) {
                   err = sdkerrors.Wrapf(types.ErrPermission, "Calling account (%s) has no (%d) permissions on target substation (%s)", activePlayer.GetPlayerId(), permission, cache.GetSubstationId())
                }
            }
        }
    }
    return
}


